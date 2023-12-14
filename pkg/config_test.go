package binman

import (
	"fmt"
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

// Helper to backup user config, and use default
func prepConfig(t *testing.T) (string, string) {

	binmanConfigPath, err := os.UserConfigDir()
	if err != nil {
		t.Fatal("unable to detect UserConfigDir")
	}

	binmanConfigFile := fmt.Sprintf("%s/%s", binmanConfigPath, "binman/config")
	binmanConfigFileBack := fmt.Sprintf("%s/%s", binmanConfigPath, "binman/config.back")

	if _, err := os.Stat(binmanConfigFile); err == nil {
		if err = CopyFile(binmanConfigFile, binmanConfigFileBack); err != nil {
			t.Fatalf("Error creating backup of %s", binmanConfigFile)
		}
		if err = os.Remove(binmanConfigFile); err != nil {
			t.Fatalf("Failed to remove current file %s for test - %s", binmanConfigFile, err)
		}
	}
	return binmanConfigFile, binmanConfigFileBack
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
	testString := SetBaseConfig("noConfig")
	expectedString := mustEnsureDefaultPaths()

	if testString != expectedString {
		t.Fatalf("%s was expected result, recieved %s", expectedString, testString)
	}

	// Test value of BINMAN_CONFIG is set
	err := os.Setenv("BINMAN_CONFIG", "testvalue")
	if err != nil {
		t.Fatal("Unable to set BINMAN_CONFIG env var")
	}

	testString = SetBaseConfig("noConfig")
	if testString != os.Getenv("BINMAN_CONFIG") {
		t.Fatalf("%s was expected result, recieved %s", "testvalue", testString)
	}

	err = os.Unsetenv("BINMAN_CONFIG")
	if err != nil {
		t.Fatal("Unable to unset BINMAN_CONFIG env var")
	}

	// Test user supplied path is returned
	testString = SetBaseConfig("testValue")
	if testString != "testValue" {
		t.Fatalf("%s was expected result, recieved %s", "testValue", testString)
	}

}

func TestSetConfig(t *testing.T) {
	d, cdir := configTestHelper(t)

	defer os.RemoveAll(d)
	defer os.Chdir(cdir)

	os.Chdir(d)

	binmanConfigFile, binmanConfigFileBack := prepConfig(t)

	config := SetConfig(SetBaseConfig("noConfig"), nil, nil)
	baseLength := len(config.Releases)

	if baseLength != 1 {
		if err := CopyFile(binmanConfigFileBack, binmanConfigFile); err != nil {
			t.Fatalf("Error restoring backup of %s", binmanConfigFile)
		}
		t.Fatalf("base release length should be 1. Is %d", baseLength)
	}

	// Add a default config
	cf := fmt.Sprintf(d + "/" + ".binMan.yaml")
	WriteStringtoFile(cf, mergeConfig)

	mergedConfig := SetConfig(SetBaseConfig("noConfig"), nil, nil)
	mergedLength := len(mergedConfig.Releases)

	if baseLength == mergedLength {
		if err := CopyFile(binmanConfigFileBack, binmanConfigFile); err != nil {
			t.Fatalf("Error restoring backup of %s", binmanConfigFile)
		}
		t.Fatalf("config merge has failed. %d should not == %d", baseLength, mergedLength)
	}

	if err := CopyFile(binmanConfigFileBack, binmanConfigFile); err != nil {
		t.Fatalf("Error restoring backup of %s", binmanConfigFile)
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
	WriteStringtoFile(cf, defaultConfig)

	// We should detect a .binMan.yaml this time
	_, sb = detectRepoConfig()
	if sb == false {
		t.Fatalf("A .binMan.yaml has been falsely ignored in %s", cf)
	}
}

func TestMustEnsureDefaultPaths(t *testing.T) {

	binmanConfigFile, binmanConfigFileBack := prepConfig(t)

	mustEnsureDefaultPaths()

	cf, err := os.ReadFile(filepath.Clean(binmanConfigFile))
	if err != nil {
		t.Fatalf("unable to read file %s", binmanConfigFile)
	}

	cfString := string(cf)

	if cfString != defaultConfig {
		if err = CopyFile(binmanConfigFileBack, binmanConfigFile); err != nil {
			t.Fatalf("Error restoring backup of %s", binmanConfigFile)
		}
		t.Fatalf("Extracted data from %s does not equal default config", binmanConfigFile)
	}

	if err = CopyFile(binmanConfigFileBack, binmanConfigFile); err != nil {
		t.Fatalf("Error restoring backup of %s", binmanConfigFile)
	}
}
