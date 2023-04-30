package gl

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xanzy/go-gitlab"
)

var e1 = "https://what.ami"
var e2 = "https://file_other_os.zip"
var e3 = "https://file_linux_amd64.wrongending"
var e4 = "https://file_v0.0.0_linux_amd64"
var e5 = "https://ile_0.0.0_linux_amd64"

// createTestData is a helper function to create test cases that should never be returned. It should be passed the "successful" test string
func createTestData(passCase string) []*gitlab.ReleaseLink {

	assets := []*gitlab.ReleaseLink{
		{DirectAssetURL: e1},
		{DirectAssetURL: e2},
		{DirectAssetURL: e3},
		{DirectAssetURL: e4},
		{DirectAssetURL: e5},
		{DirectAssetURL: "https://example.com/" + passCase},
	}

	return assets
}

// TestGHGetAssetData will test mapCreation function
func TestGLGetAssetData(t *testing.T) {

	var assetName = "my_awesome_file"

	var passMap = map[string]string{
		filepath.Base(e1):                 e1,
		filepath.Base(e2):                 e2,
		filepath.Base(e3):                 e3,
		filepath.Base(e4):                 e4,
		filepath.Base(e5):                 e5,
		"https://example.com" + assetName: assetName,
	}

	m := GLGetAssetData(createTestData(assetName))

	if reflect.DeepEqual(m, passMap) {
		t.Fatalf("Returned map != passMap")
	}
}

func TestGetAssetbyName(t *testing.T) {
	var assetName = "my_awesome_file"
	name, _ := GetAssetbyName(assetName, createTestData("https://"+assetName))

	if name != assetName {
		t.Fatalf("%s should = %s", assetName, name)
	}
}
