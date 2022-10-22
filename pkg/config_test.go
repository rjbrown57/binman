package binman

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// configTestHelper will provide required test scenarios
func configTestHelper(t *testing.T) (string, string) {
	d, err := os.MkdirTemp(os.TempDir(), "binmantest")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	cdir, err := os.Getwd()
	if err != nil {
		t.Fatal("unable to get current dir")
	}

	return d, cdir

}

func TestSetupConfigPath(t *testing.T) {
	d, _ := configTestHelper(t)

	defer os.RemoveAll(d)

	targetd := fmt.Sprintf("%s/%s", d, "config")

	setupConfigDir(targetd)

	_, err := os.Stat(targetd)
	if err != nil {
		t.Fatalf("Unable to create %s", targetd)
	}

}

func TestSetBaseConfig(t *testing.T) {

	// Test default path is returned
	testString := setBaseConfig("noConfig")
	expectedString := mustEnsureDefaultPaths()

	if testString != expectedString {
		t.Fatalf("%s was expected result, recieved %s", expectedString, testString)
	}

	// Test value of BINMAN_CONFIG is set

	err := os.Setenv("BINMAN_CONFIG", "testvalue")
	if err != nil {
		t.Fatal("Unable to set BINMAN_CONFIG env var")
	}

	testString = setBaseConfig("noConfig")
	if testString != os.Getenv("BINMAN_CONFIG") {
		t.Fatalf("%s was expected result, recieved %s", "testvalue", testString)
	}

	err = os.Unsetenv("BINMAN_CONFIG")
	if err != nil {
		t.Fatal("Unable to unset BINMAN_CONFIG env var")
	}

	// Test user supplied path is returned
	testString = setBaseConfig("testValue")
	if testString != "testValue" {
		t.Fatalf("%s was expected result, recieved %s", "testValue", testString)
	}

}

func TestSetConfig(t *testing.T) {
	d, cdir := configTestHelper(t)

	defer os.RemoveAll(d)
	defer os.Chdir(cdir)

	os.Chdir(d)

	config := setConfig(setBaseConfig("noConfig"))
	baseLength := len(config.Releases)

	if baseLength != 1 {
		t.Fatalf("base release length should be 1. Is %d", baseLength)
	}

	// Add a default config
	cf := fmt.Sprintf(d + "/" + ".binMan.yaml")
	writeStringtoFile(cf, defaultConfig)

	mergedConfig := setConfig(setBaseConfig("noConfig"))
	mergedLength := len(mergedConfig.Releases)

	if baseLength == mergedLength {
		t.Fatalf("config merge has failed. %d should not == %d", baseLength, mergedLength)
	}

}

func TestDetectRepoConfig(t *testing.T) {
	d, cdir := configTestHelper(t)

	defer os.RemoveAll(d)
	defer os.Chdir(cdir)

	os.Chdir(d)

	// test not finding a $PWD/.binMan.yaml
	_, sb := detectRepoConfig()
	if sb == true {
		t.Fatalf("A .binMan.yaml has been falsely detected in %s", d)
	}

	// Add a default config
	cf := fmt.Sprintf(d + "/" + ".binMan.yaml")
	writeStringtoFile(cf, defaultConfig)

	// We should detect a .binMan.yaml this time
	_, sb = detectRepoConfig()
	if sb == false {
		t.Fatalf("A .binMan.yaml has been falsely ignored in %s", cf)
	}
}

func TestMustEnsureDefaultPaths(t *testing.T) {

	binmanConfigPath, err := os.UserConfigDir()
	if err != nil {
		t.Fatal("unable to detect UserConfigDir")
	}

	binmanConfigPath = binmanConfigPath + "/binman"
	binmanConfigFile := binmanConfigPath + "/config"

	mustEnsureDefaultPaths()

	cf, err := ioutil.ReadFile(filepath.Clean(binmanConfigFile))
	if err != nil {
		t.Fatalf("unable to read file %s", binmanConfigFile)
	}

	cfString := string(cf)

	if cfString != defaultConfig {
		t.Fatalf("Extracted data from %s does not equal default config", binmanConfigFile)
	}
}
