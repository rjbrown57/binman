package binman

import (
	"fmt"
	"os"
	"sync"
	"testing"

	db "github.com/rjbrown57/binman/pkg/db"
	log "github.com/rjbrown57/binman/pkg/logging"
)

// if user does not provide a -c this will be populated at ~/.config/binman/config
const testPopulateConfig = `
config:
  releasepath: {{ .releasePath }}
  sources:
   - name: gitlab.com
     #tokenvar: GL_TOKEN # environment variable that contains gitlab token
     apitype: gitlab
   - name: github.com
     #tokenvar: GH_TOKEN # environment variable that contains github token
     apitype: github
releases:
  - repo: org1/repo1
  - repo: org1/repo2
  - repo: gitlab.com/org2/repo1
`

func createTestDir(t *testing.T, testVersions []string, configTemplate string, testCase string) (string, string) {

	var err error

	dM := make(map[string]interface{})

	dM["releasePath"], err = os.MkdirTemp(os.TempDir(), testCase)
	if err != nil {
		t.Fatalf("Unable to create test dir")
	}

	// Prepare test config
	bmConfigPath := fmt.Sprintf("%s/config", dM["releasePath"])
	// This needs to render in the path
	err = WriteStringtoFile(bmConfigPath, formatString(configTemplate, dM))
	if err != nil {
		t.Fatalf("Failed to render test config to %s", dM["releasePath"])
	}

	for _, path := range testVersions {
		err = os.MkdirAll(fmt.Sprintf("%s/repos/%s", dM["releasePath"], path), 0755)
		if err != nil {
			t.Fatalf("Unable to create %s", fmt.Sprintf("%s/repos/%s", dM["releasePath"], path))
		}
	}

	return dM["releasePath"].(string), bmConfigPath
}

func TestPopulateData(t *testing.T) {

	log.ConfigureLog(true, true)

	testVersions := []string{
		"github.com/org1/repo1/v0.0.0",
		"github.com/org1/repo2/v0.0.0",
		"gitlab.com/org2/repo1/v0.0.1",
	}

	// Create test directory and populate with config + example dir
	testDir, testConfig := createTestDir(&testing.T{}, testVersions, testPopulateConfig, "poptest")

	defer os.RemoveAll(testDir)

	var dwg sync.WaitGroup
	dbOptions := db.DbConfig{
		Dwg:       &dwg,
		DbChan:    make(chan db.DbMsg),
		Path:      fmt.Sprintf("%s/binman.db", testDir),
		Overwrite: false,
	}

	err := populateDB(dbOptions, testConfig)
	if err != nil {
		t.Fatalf("Failed to populated DB")
	}

	testDb := db.GetDB(fmt.Sprintf("%s/binman.db", testDir))

	for _, testKey := range testVersions {
		keys := db.ParseKey(testKey)
		data, err := db.GetData(testKey+"/data", testDb)
		if err != nil {
			t.Fatalf("Failed to get %s from db", testKey)
		}
		dm := bytesToData(data)
		got := dm["version"].(string)
		expected := string(keys[len(keys)-1])
		if got != expected {
			t.Fatalf("got %s , expected %s", got, expected)
		}
	}
}
