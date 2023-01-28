package config

import (
	"fmt"
	"os"
	"testing"

	binman "github.com/rjbrown57/binman/pkg"
)

func getTestDir(t *testing.T) string {
	d, err := os.MkdirTemp(os.TempDir(), "binmantest")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}
	return d
}

// See if repo is already in config
func TestReleasesContains(t *testing.T) {

	var r = []binman.BinmanRelease{
		{Repo: "rjbrown57/binextractor"},
		{Repo: "rjbrown57/binman"},
	}

	var tests = []struct {
		testRepo string
		expected bool
	}{
		{testRepo: "rjbrown57/binman", expected: true},
		{testRepo: "rjbrown57/notreal", expected: false},
	}

	for _, test := range tests {
		got := releasesContains(r, test.testRepo)
		if test.expected != got {
			t.Fatalf("%s got %v expected %v", test.testRepo, test.expected, got)
		}
	}
}

const testConfig = `
config:
  releasepath: thereleasepath
  tokenvar: none # we set to 'none' here so ci based test will function
  upx:
    enabled: true
    args: []
releases:
  - repo: rjbrown57/binman
`

func TestAdd(t *testing.T) {
	var testRepo = "rjbrown57/lp"

	d := getTestDir(t)
	defer os.Remove(d)

	configPath := fmt.Sprintf("%s/config", d)
	err := binman.WriteStringtoFile(configPath, testConfig)
	if err != nil {
		t.Fatalf("failed to write test config to %s", configPath)
	}

	Add(configPath, testRepo)

	c := binman.NewGHBMConfig(configPath)
	if !releasesContains(c.Releases, testRepo) {
		t.Fatalf("%s not added to config properly", testRepo)
	}
}

func TestGetEditor(t *testing.T) {
	d := getTestDir(t)
	defer os.Remove(d)

	testVal := fmt.Sprintf("%s/%s", d, "testval")
	binman.WriteStringtoFile(testVal, "testvaldata")
	err := os.Chmod(testVal, 0755)
	if err != nil {
		t.Fatalf("Not able to set execute on %s - %s", testVal, err)
	}

	// add our testdir to the path
	path := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", path, d))

	os.Setenv("EDITOR", testVal)
	got := getEditor()

	if got != testVal {
		t.Fatalf("Got %s expected %s", got, testVal)
	}
}
