package binman

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	semver "github.com/Masterminds/semver/v3"
	db "github.com/rjbrown57/binman/pkg/db"
	log "github.com/rjbrown57/binman/pkg/logging"
	bolt "go.etcd.io/bbolt"
)

var (
	// ErrVersionsListNotSorted is returned when a list of versions is not sorted
	ErrVersionsListNotSorted = errors.New("Provided list of versions is not sorted")
)

func Clean(dryrun, debug, jsonLog, scan bool, threshold int, dbPath, config string) error {
	log.ConfigureLog(jsonLog, debug)

	log.Infof("Binman Clean started")

	var dwg sync.WaitGroup

	dbOptions := db.DbConfig{
		Path:   dbPath,
		Dwg:    &dwg,
		DbChan: make(chan db.DbMsg),
	}

	switch {
	case checkNewDb(dbPath):
		log.Warnf("setting dry run since db will be populated during this run of `binman clean`")
		dryrun = true
		fallthrough
	case scan:
		populateDB(dbOptions, config)
	}

	bdb := db.GetDB(dbPath, bolt.Options{Timeout: 1 * time.Second, ReadOnly: false})

	c := SetConfig(SetBaseConfig(config), dbOptions.Dwg, dbOptions.DbChan)

	for _, rel := range c.Releases {
		// Collect All Versions
		err := rel.getVersions(bdb)
		if err != nil {
			log.Warnf("Unable to get all versions for %s %s", rel.Repo, err)
			continue
		}

		rel.versions, err = sortSemvers(rel.versions)
		if err != nil {
			log.Warnf("Unable to sort semvers for %s %s", rel.Repo, err)
			continue
		}

		if len(rel.versions) < threshold {
			log.Infof("Skipping %s this repo does not have more %d versions. %s", rel.Repo, threshold, rel.versions)
			continue
		}

		// Execute the Clean
		err = rel.cleanOldReleases(dryrun, threshold, bdb)
		if err != nil {
			log.Warnf("Issue cleaning %s %s", rel.Repo, err)
		}
	}

	err := bdb.Close()
	if err != nil {
		log.Fatalf("Unable to close db %s", err)
	}

	log.Infof("Clean complete")
	return nil
}

func sortSemvers(versions []string) ([]string, error) {
	var sorted []string
	var err error

	vs := make([]*semver.Version, len(versions))

	// Convert to semvers
	for i, r := range versions {
		v, err := semver.NewVersion(r)
		if err != nil {
			return nil, err
		}

		vs[i] = v
	}

	sort.Sort(semver.Collection(vs))

	if !sort.SliceIsSorted(vs, func(i, j int) bool {
		return vs[i].LessThan(vs[j])
	}) {
		return nil, ErrVersionsListNotSorted
	}

	// Get the sorted list as an array of strings
	for _, r := range vs {
		sorted = append(sorted, r.Original())
	}

	return sorted, err
}

// getVersions will collect all versions we currently have stored in the DB
func (r *BinmanRelease) getVersions(bdb *bolt.DB) error {
	return bdb.View(func(tx *bolt.Tx) error {
		sourceBucket := tx.Bucket([]byte(r.SourceIdentifier))
		orgBucket := sourceBucket.Bucket([]byte(r.org))
		projBucket := orgBucket.Bucket([]byte(r.project))
		projBucket.ForEachBucket(func(k []byte) error {
			version := string(k)
			_, err := semver.NewVersion(version)
			if err == nil {
				r.versions = append(r.versions, version)
				return nil
			}
			return err
		})
		return nil
	})
}

func (r *BinmanRelease) cleanOldReleases(dryrun bool, threshold int, bdb *bolt.DB) error {

	numVersions := len(r.versions)

	for _, toDelete := range r.versions[:numVersions-threshold] {

		byteData, err := db.GetData(fmt.Sprintf("%s/%s/%s/data", r.SourceIdentifier, r.Repo, toDelete), bdb)
		if err != nil {
			log.Warnf("Issue getting data for %s/%s", r.Repo, toDelete)
		}

		d := bytesToData(byteData)

		log.Infof("%s(%s): %s will be deleted", d["repo"], d["version"], d["publishPath"])

		if dryrun {
			continue
		}

		// If path does not exist err value is nil
		err = os.RemoveAll(d["publishPath"].(string))
		if err != nil {
			log.Fatalf("Error deleting %s", d["publishPath"])
		}

		// Delete the version bucket
		err = db.DeleteData(fmt.Sprintf("%s/%s/%s", r.SourceIdentifier, r.Repo, toDelete), bdb)
		if err != nil {
			log.Fatalf("Error removing %s-%s from db", r.Repo, toDelete)
		}
		log.Infof("%s(%s): cleaned successfully", d["repo"], d["version"])

	}
	return nil
}
