package binmandb

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	log "github.com/rjbrown57/binman/pkg/logging"
)

type dbTest struct {
	caseName  string
	data      []byte
	keyString string
}

func getTests() []dbTest {

	basicString := "bucket1/key1"
	complexString := "bucket1/bucket2/key1"

	var tests = []dbTest{
		{caseName: "basic", data: []byte("basic"), keyString: basicString},
		{caseName: "complex", data: []byte(complexString), keyString: complexString},
	}
	return tests
}

// TestRunDB will start the db routine and test reading/writing messages and deleting buckets
func TestRunDB(t *testing.T) {

	// Set the logging options
	log.ConfigureLog(true, 2)

	d, err := os.MkdirTemp(os.TempDir(), "binmantest")
	if err != nil {
		t.Fatalf("Unable to create test dir")
	}

	var dwg sync.WaitGroup

	defer os.RemoveAll(d)

	dbOptions := DbConfig{
		Path:      d + "/test.db",
		Dwg:       &dwg,
		DbChan:    make(chan DbMsg),
		Overwrite: false,
	}

	go RunDB(dbOptions)

	tests := getTests()

	for _, test := range tests {
		log.Infof("Testing %s", test.caseName)
		// write messages
		var rwg sync.WaitGroup
		rwg.Add(1)

		m := DbMsg{
			Key:        test.keyString,
			Data:       test.data,
			Operation:  "write",
			ReturnChan: make(chan DBResponse, 1), // Created a buffered channel since we will not run a recieving goroutine. Size is always 1
			ReturnWg:   &rwg,
		}

		dwg.Add(1)
		dbOptions.DbChan <- m
		log.Infof("Waiting for write response")
		rwg.Wait()
		close(m.ReturnChan)

		for rm := range m.ReturnChan {
			if rm.Err != nil {
				t.Fatalf("Error writing data to DB %v", err)
			}
		}

		// Now attempt to read back what we stored
		m.Operation = "read"
		m.ReturnChan = make(chan DBResponse, 1)

		dwg.Add(1)
		rwg.Add(1)
		dbOptions.DbChan <- m
		log.Infof("Waiting for read response")
		rwg.Wait()
		close(m.ReturnChan)

		for rm := range m.ReturnChan {
			if rm.Err != nil {
				t.Fatalf("Error reading data to DB %v", err)
			}
			if !bytes.Equal(rm.Data, test.data) {
				t.Fatalf("Expected %s, got %s", rm.Data, test.data)
			}
			log.Infof("%s - %s", rm.Data, test.data)
		}

		// Now delete the buckets we created
		// Verification will occur by re-opening the DB
		m.Operation = "delete"
		m.ReturnChan = make(chan DBResponse, 1)

		dwg.Add(1)
		rwg.Add(1)
		dbOptions.DbChan <- m
		log.Infof("Waiting for delete response")
		rwg.Wait()
		close(m.ReturnChan)

		for rm := range m.ReturnChan {
			if rm.Err != nil {
				t.Fatalf("Error deleting bucket from DB %v", err)
			}
		}

	}

	close(dbOptions.DbChan)

	// We now need to validate the delete succeeded
	db := GetDB(d + "/test.db")

	for _, test := range tests {
		log.Infof("Testing %s removed", test.caseName)
		_, err := GetData(test.keyString, db)
		if !errors.Is(err, ErrNilReadResponse) {
			t.Fatalf("error for expected delete %s", err)
		}
	}
}

// TestGetData will test write/read functions
func TestGetData(t *testing.T) {

	log.ConfigureLog(true, 2)

	d, err := os.MkdirTemp(os.TempDir(), "binmantest")
	if err != nil {
		t.Fatalf("Unable to create test dir")
	}
	defer os.RemoveAll(d)

	testDb := GetDB(fmt.Sprintf("%s/binman.db", d))

	for _, test := range getTests() {

		err := WriteData(false, test.keyString, test.data, testDb)
		if err != nil {
			t.Fatalf("Failed to write data to db %s = %s %v", test.keyString, test.data, err)
		}

		data, err := GetData(test.keyString, testDb)
		if err != nil {
			t.Fatalf("Failed to get %s from db", test.keyString)
		}

		if !bytes.Equal(test.data, data) {
			t.Fatalf("got %s , expected %s", test.data, data)
		}
	}
}
