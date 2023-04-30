package gh

import (
	"reflect"
	"testing"

	"github.com/google/go-github/v50/github"
)

var bogusString = "what.ami"
var wrongOs = "file_other_os.zip"
var wrongEnding = "file_linux_amd64.wrongending"
var possibleMatch = "file_v0.0.0_linux_amd64"
var possibleMatch2 = "file_0.0.0_linux_amd64"

// createTestData is a helper function to create test cases that should never be returned. It should be passed the "successful" test string
func createTestData(passCase string) []*github.ReleaseAsset {

	assets := []*github.ReleaseAsset{
		{Name: &bogusString, BrowserDownloadURL: &bogusString},
		{Name: &wrongOs, BrowserDownloadURL: &wrongOs},
		{Name: &wrongEnding, BrowserDownloadURL: &wrongEnding},
		{Name: &passCase, BrowserDownloadURL: &passCase},
	}

	return assets
}

// TestGHGetAssetData will test mapCreation function
func TestGHGetAssetData(t *testing.T) {

	var assetName = "my_awesome_file"

	var passMap = map[string]string{
		bogusString: bogusString,
		wrongOs:     wrongOs,
		wrongEnding: wrongEnding,
		assetName:   assetName,
		assetName:   possibleMatch,
		assetName:   possibleMatch2,
	}

	m := GHGetAssetData(createTestData(assetName))

	if reflect.DeepEqual(m, passMap) {
		t.Fatalf("Returned map != passMap")
	}
}

func TestGetAssetbyName(t *testing.T) {
	var assetName = "my_awesome_file"
	name, _ := GetAssetbyName(assetName, createTestData(assetName))

	if name != assetName {
		t.Fatalf("%s should = %s", assetName, name)
	}
}
