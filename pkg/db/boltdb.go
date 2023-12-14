package binmandb

import (
	"fmt"
	"sync"
	"time"

	"os"

	log "github.com/rjbrown57/binman/pkg/logging"
	bolt "go.etcd.io/bbolt"
)

func GetDB(dbPath string, o ...bolt.Options) *bolt.DB {

	boltOptions := bolt.Options{Timeout: 1 * time.Second, ReadOnly: false}

	if len(o) != 0 {
		boltOptions = o[0]
	}

	log.Debugf("Starting db with options %+v", boltOptions)

	configPath, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Unable to find userConfigDir")
	}

	if dbPath == "" {
		dbPath = fmt.Sprintf("%s/binman/binman.db", configPath)
	}

	db, err := bolt.Open(dbPath, 0700, &boltOptions)
	if err != nil {
		log.Fatalf("Unable to open DB - %s", err)
	}

	log.Debugf("Opened db at %s", dbPath)

	return db
}

// DbConfig contains required chan/wg + config options
type DbConfig struct {
	Path      string
	Dwg       *sync.WaitGroup
	DbChan    chan DbMsg
	Overwrite bool
}

type DbMsg struct {
	Key        string // "Keys should be in the format key/subkey/subsubkey"
	Data       []byte
	Operation  string // "read/write/delete"
	ReturnChan chan DBResponse
	ReturnWg   *sync.WaitGroup
}

type DBResponse struct {
	Err  error
	Data []byte
}

// RunDB waits for messages on dbChan and performs the appropriate operations
func RunDB(options DbConfig) {
	db := GetDB(options.Path)

	log.Debugf("DB started and waiting for messages")

	for msg := range options.DbChan {

		log.Debugf("DB Processing %s for %s", msg.Operation, msg.Key)

		r := DBResponse{
			Err:  nil,
			Data: nil,
		}

		switch msg.Operation {
		case "read":
			r.Data, r.Err = GetData(msg.Key, db)
		case "write":
			r.Err = WriteData(options.Overwrite, msg.Key, msg.Data, db)
		case "delete":
			r.Err = DeleteData(msg.Key, db)
		default:
			log.Fatalf("DB recieved unkown operation %s", msg.Operation)
		}

		log.Debugf("Responding with %+v", r)

		msg.ReturnChan <- r
		msg.ReturnWg.Done()
		options.Dwg.Done()
	}

	err := db.Close()
	if err != nil {
		log.Fatalf("unable to close db - %s", err)
	}
}

// read from db
func GetData(key string, db *bolt.DB) ([]byte, error) {

	var err error
	bucketKeys := ParseKey(key)
	keyNum := len(bucketKeys)
	dataKey := bucketKeys[keyNum-1]
	var data []byte

	getBuckets := func(tx *bolt.Tx) error {
		b := parseBuckets(bucketKeys, tx)
		data = b.Get(dataKey)
		if data == nil {
			return ErrNilReadResponse
		}
		return nil
	}

	if db.IsReadOnly() {
		err := db.View(getBuckets)
		return data, err
	}

	err = db.Update(getBuckets)
	return data, err
}

// parseBuckets will return a *bolt.Bucket based on a string "key/mykey/mysubkey". If buckets do not exist they will be created.
func parseBuckets(bucketKeys [][]byte, tx *bolt.Tx) *bolt.Bucket {

	var buckets []*bolt.Bucket
	var err error

	keyNum := len(bucketKeys)

	bucket := tx.Bucket(bucketKeys[0])
	if bucket == nil {
		bucket, err = tx.CreateBucket(bucketKeys[0])
		if err != nil {
			log.Fatalf("Unable to create bucket %s, %s", bucketKeys[0], err)
		}
	}

	buckets = append(buckets, bucket)

	switch keyNum {
	case 0, 1:
		log.Warnf("Invalid bucket key supplied")
		return nil
	case 2:
		break
	default:
		// We want to range over bucketsKeys except the first and the last element
		// Using the previous bucket we get the next.
		//  "mykey/mysubkey/mysubsubkey"
		for i := 1; i < keyNum-1; i++ {
			b := buckets[i-1].Bucket(bucketKeys[i])
			if b == nil {
				b, err = buckets[i-1].CreateBucketIfNotExists(bucketKeys[i])
				if err != nil {
					log.Fatalf("Unable to create bucket %s, %v", bucketKeys[i], err)
				}
			}

			buckets = append(buckets, b)
		}
	}

	// return last found bucket
	return buckets[len(buckets)-1]

}

// write to db
func WriteData(overwrite bool, key string, data []byte, db *bolt.DB) error {

	var err error
	bucketKeys := ParseKey(key)
	keyNum := len(bucketKeys)
	dataKey := bucketKeys[keyNum-1]

	err = db.Update(func(tx *bolt.Tx) error {
		b := parseBuckets(bucketKeys, tx)
		if b.Get(dataKey) == nil || overwrite {
			return b.Put(dataKey, data)
		}
		return ErrKeyExists
	})

	return err
}

// DeleteData will delete either key/values or Buckets depending on what is sent by user
func DeleteData(key string, db *bolt.DB) error {
	var err error
	bucketKeys := ParseKey(key)
	keyNum := len(bucketKeys)
	dataKey := bucketKeys[keyNum-1]

	err = db.Update(func(tx *bolt.Tx) error {
		b := parseBuckets(bucketKeys, tx)
		if b.Bucket(dataKey) == nil {
			return b.Delete(dataKey)
		}
		return b.DeleteBucket(dataKey)
	})

	return err
}
