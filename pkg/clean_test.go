package binman

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	db "github.com/rjbrown57/binman/pkg/db"
	log "github.com/rjbrown57/binman/pkg/logging"
)

// if user does not provide a -c this will be populated at ~/.config/binman/config
const testCleanConfig = `
config:
  releasepath: {{ .releasePath }}
releases:
  - repo: org1/repo1
`

type CleanTest struct {
	caseName  string
	expected  []string
	toCreate  []string
	deleted   []string
	threshold int
}

func TestClean(t *testing.T) {
	log.ConfigureLog(true, 2)

	testCleans := []CleanTest{
		{
			toCreate:  []string{"github.com/org1/repo1/v0.0.0"},
			expected:  []string{"github.com/org1/repo1/v0.0.0"},
			deleted:   []string{},
			threshold: 1,
			caseName:  "NoAction"},
		{
			toCreate:  []string{"github.com/org1/repo1/v0.0.0", "github.com/org1/repo1/v0.0.1", "github.com/org1/repo1/v0.0.2", "github.com/org1/repo1/v0.0.3", "github.com/org1/repo1/v0.0.4"},
			expected:  []string{"github.com/org1/repo1/v0.0.2", "github.com/org1/repo1/v0.0.3", "github.com/org1/repo1/v0.0.4"},
			deleted:   []string{"github.com/org1/repo1/v0.0.0", "github.com/org1/repo1/v0.0.1"},
			threshold: 3,
			caseName:  "Clean1"},
		{
			toCreate:  []string{"github.com/org1/repo1/0.0.0", "github.com/org1/repo1/0.0.1", "github.com/org1/repo1/0.0.2", "github.com/org1/repo1/0.0.3", "github.com/org1/repo1/0.0.4"},
			expected:  []string{"github.com/org1/repo1/0.0.2", "github.com/org1/repo1/0.0.3", "github.com/org1/repo1/0.0.4"},
			deleted:   []string{"github.com/org1/repo1/0.0.0", "github.com/org1/repo1/0.0.1"},
			threshold: 3,
			caseName:  "Clean2"},
		{
			toCreate:  []string{"github.com/org1/repo1/asdf1", "github.com/org1/repo1/asdfv"},
			expected:  []string{"github.com/org1/repo1/asdf1", "github.com/org1/repo1/asdfv"},
			threshold: 1,
			caseName:  "BadVersions"},
	}

	var dwg sync.WaitGroup

	for _, test := range testCleans {
		log.Debugf("Test Case = %s", test.caseName)
		testDir, testConfig := createTestDir(&testing.T{}, test.toCreate, testCleanConfig, test.caseName)
		dbPath := fmt.Sprintf("%s/binman.db", testDir)

		// we need to prepopulate the DB to avoid the DB being initialized by the clean function, and dry run being enabled
		err := populateDB(db.DbConfig{
			Path:   dbPath,
			Dwg:    &dwg,
			DbChan: make(chan db.DbMsg),
		}, testConfig)
		if err != nil {
			log.Fatalf("Issue populating DB %s", err)
		}

		Clean(false, true, test.threshold, dbPath, testConfig)

		testDb := db.GetDB(dbPath)

		// Check our expected keys are still present
		for _, expected := range test.expected {
			_, err := db.GetData(fmt.Sprintf("%s/%s", expected, "data"), testDb)
			if err != nil {
				t.Fatalf("Expected key does not exist %s", expected)
			}
		}

		// Check the keys we expected to be removed, have in fact been removed
		for _, deleted := range test.deleted {
			_, err := db.GetData(fmt.Sprintf("%s/%s", deleted, "data"), testDb)
			if !errors.Is(err, db.ErrNilReadResponse) {
				t.Fatalf("%s/%s should have been deleted but has responded with data", deleted, "data")
			}
		}

		err = testDb.Close()
		if err != nil {
			t.Fatalf("Unable to close db")
		}

		os.RemoveAll(testDir)
		log.Debugf("%s Passed", test.caseName)
	}
}
