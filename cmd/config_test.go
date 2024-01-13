package cmd

import (
	"errors"
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
	t.Cleanup(func() { os.Remove(d) })

	configPath := fmt.Sprintf("%s/config", d)
	err := binman.WriteStringtoFile(configPath, testConfig)
	if err != nil {
		t.Fatalf("failed to write test config to %s", configPath)
	}

	Add(binman.NewBMConfig(configPath).SetConfig(false), testRepo)

	c := binman.NewBMConfig(configPath).SetConfig(false)
	if _, err := c.GetRelease(testRepo); errors.Is(err, binman.ErrReleaseNotFound) {
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
