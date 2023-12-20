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

func dataToBytes(r map[string]interface{}) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(r)
	if err != nil {
		log.Fatalf("Unable to encode for db ingestion")
	}
	return buf.Bytes()
}

func bytesToData(b []byte) map[string]interface{} {

	dataMap := make(map[string]interface{})

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

func getVersionBuckets(tx *bolt.Tx) []*bolt.Bucket {

	var buckets []*bolt.Bucket

	tx.ForEach(func(name []byte, b *bolt.Bucket) error {
		log.Debugf("scanning source = %s", name)
		b.ForEachBucket(func(orgKey []byte) error {
			b2 := b.Bucket(orgKey)
			b2.ForEachBucket(func(projKey []byte) error {
				b3 := b2.Bucket(projKey)
				b3.ForEachBucket(func(versionKey []byte) error {
					buckets = append(buckets, b3.Bucket(versionKey))
					return nil
				})
				return nil
			})
			return nil
		})
		return nil
	})

	return buckets
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

	db := db.GetDB("", bolt.Options{Timeout: 1 * time.Second, ReadOnly: true})

	db.View(func(tx *bolt.Tx) error {
		for _, bucket := range getVersionBuckets(tx) {
			dataMap := bytesToData(bucket.Get([]byte("data")))
			// Releases added via populate will not have createdAt/artifactPath populated
			// This will show us as 1969 unless we handle the like this
			if dataMap["createdAt"].(int64) == 0 {
				upToDateTable.AddRow(dataMap["repo"], dataMap["version"], "-")
				continue
			}
			t := time.Unix(dataMap["createdAt"].(int64), 0)
			upToDateTable.AddRow(dataMap["repo"], dataMap["version"], t.Format(time.DateTime))
		}
		return nil
	})

	err := db.Close()
	if err != nil {
		log.Fatalf("unable to close db - %s", err)
	}

	upToDateTable.Print()

	return nil
}

// populateDB is used to populate the db with data required for clean up
func populateDB(dbOptions db.DbConfig, config string) error {

	go db.RunDB(dbOptions)

	// Create config object.
	// setBaseConfig will return the appropriate base config file.
	// setConfig will check for a contextual config and merge with our base config and return the result
	c := SetConfig(SetBaseConfig(config), nil, nil, nil)

	log.Debugf("Updating binman db from filesystem")

	for _, rel := range c.Releases {
		repoPath := fmt.Sprintf("%s/repos/%s/%s", rel.ReleasePath, rel.SourceIdentifier, rel.Repo)

		// scan for versions in the path
		files, err := os.ReadDir(repoPath)
		if err != nil {
			log.Fatalf("Unable to read dir %s %s", repoPath, err)
		}

		versions := make([]string, 0)

		for _, f := range files {
			if f.IsDir() {
				versions = append(versions, f.Name())
			}
		}

		log.Debugf("Versions %s for %s found", versions, repoPath)

		// for each version get the path that we should add to the DB
		for _, v := range versions {
			rel.Version = v
			rel.setPublishPath(rel.ReleasePath, rel.Version)
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
	dbOptions.Dwg.Wait()
	close(dbOptions.DbChan)
	return nil
}
