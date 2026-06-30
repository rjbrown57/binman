package binman

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"sync"
	"time"

	"os"

	"github.com/fatih/color"
	db "github.com/rjbrown57/binman/pkg/db"
	log "github.com/rjbrown57/binman/pkg/logging"

	"github.com/rodaine/table"
	bolt "go.etcd.io/bbolt"
)

func dataToBytes(r map[string]any) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(r)
	if err != nil {
		log.Fatalf("Unable to encode for db ingestion")
	}
	return buf.Bytes()
}

func bytesToData(b []byte) map[string]any {

	dataMap := make(map[string]any)

	decoder := gob.NewDecoder(bytes.NewBuffer(b))
	err := decoder.Decode(&dataMap)
	if err != nil {
		log.Fatalf("Unable to decode data from DB %s", err)
	}
	return dataMap
}

// checkDb lets us know if a new db will be created, if an empty string is sent we use os.ConfigDir
func checkNewDb(path string) bool {

	var err error

	if path == "" {
		path, err = os.UserConfigDir()
		if err != nil {
			log.Fatalf("Unable to find userConfigDir")
		}
	}

	dbPath := fmt.Sprintf("%s/binman/binman.db", path)

	_, err = os.Stat(dbPath)
	if os.IsNotExist(err) {
		return true
	}

	return false
}

func getVersionBuckets(tx *bolt.Tx) ([]*bolt.Bucket, error) {

	var buckets []*bolt.Bucket

	err := tx.ForEach(func(name []byte, b *bolt.Bucket) error {
		log.Debugf("scanning source = %s", name)
		return b.ForEachBucket(func(orgKey []byte) error {
			b2 := b.Bucket(orgKey)
			if b2 == nil {
				return fmt.Errorf("missing org bucket %s/%s", name, orgKey)
			}
			return b2.ForEachBucket(func(projKey []byte) error {
				b3 := b2.Bucket(projKey)
				if b3 == nil {
					return fmt.Errorf("missing project bucket %s/%s/%s", name, orgKey, projKey)
				}
				return b3.ForEachBucket(func(versionKey []byte) error {
					versionBucket := b3.Bucket(versionKey)
					if versionBucket == nil {
						return fmt.Errorf("missing version bucket %s/%s/%s/%s", name, orgKey, projKey, versionKey)
					}
					buckets = append(buckets, versionBucket)
					return nil
				})
			})
		})
	})

	return buckets, err
}

// OutputDbStatus will output all keys and values
func OutputDbStatus() error {

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	upToDateTable := table.New("Repo", "Version", "createdAt")
	upToDateTable.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	if checkNewDb("") {
		log.Fatalf("DB has not been initialized yet. Run `binman` first")
	}

	bdb := db.GetDB("", bolt.Options{Timeout: 1 * time.Second, ReadOnly: true})

	viewErr := bdb.View(func(tx *bolt.Tx) error {
		buckets, err := getVersionBuckets(tx)
		if err != nil {
			return err
		}
		for _, bucket := range buckets {
			if bucket == nil {
				log.Warnf("Skipping nil version bucket")
				continue
			}
			data := bucket.Get([]byte("data"))
			if data == nil {
				log.Warnf("Skipping version bucket with no data")
				continue
			}
			dataMap := bytesToData(data)
			// Releases added via populate will not have createdAt/artifactPath populated
			// This will show us as 1969 unless we handle the like this
			createdAt, ok := dataMap["createdAt"].(int64)
			if !ok || createdAt == 0 {
				upToDateTable.AddRow(dataMap["repo"], dataMap["version"], "-")
				continue
			}
			t := time.Unix(createdAt, 0)
			upToDateTable.AddRow(dataMap["repo"], dataMap["version"], t.Format(time.DateTime))
		}
		return nil
	})

	err := bdb.Close()
	if err != nil {
		log.Fatalf("unable to close db - %s", err)
	}
	if viewErr != nil {
		return viewErr
	}

	upToDateTable.Print()

	return nil
}

// populateDB is used to populate the db with data required for clean up
// TODO update with db version info
func populateDB(dbOptions db.DbConfig, config string) error {

	// Create config object.
	c := NewBMConfig(config).SetConfig(false)

	// Start the DB direct. If we use WithDB we will create recursively loops of populateDB calls
	go db.RunDB(dbOptions)

	log.Debugf("Updating binman db from filesystem")

	for _, rel := range c.Releases {

		// Collect all versions from filesystem
		versions := GetVersionFromPath(fmt.Sprintf("%s/repos/%s/%s", rel.ReleasePath, rel.SourceIdentifier, rel.Repo))

		// for each version get the path that we should add to the DB
		for _, v := range versions {
			rel.Version = v
			rel.setpublishPath(rel.ReleasePath, rel.Version)
			dbOptions.Dwg.Add(1)
			var rwg sync.WaitGroup
			rwg.Add(1)

			msg := db.DbMsg{
				Operation:  "write",
				Key:        fmt.Sprintf("%s/%s/%s/data", rel.SourceIdentifier, rel.Repo, rel.Version),
				ReturnChan: make(chan db.DBResponse, 1),
				ReturnWg:   &rwg,
				Data:       dataToBytes(rel.getDataMap()),
			}

			dbOptions.DbChan <- msg
			rwg.Wait()

			close(msg.ReturnChan)
			m := <-msg.ReturnChan
			if m.Err != nil && !errors.Is(m.Err, db.ErrKeyExists) {
				log.Fatalf("Issue writing data to db %v", m.Err)
			}

		}
	}

	close(dbOptions.DbChan)

	log.Debugf("DB Update complete")
	return nil
}
