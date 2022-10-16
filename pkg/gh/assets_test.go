package gh

import (
	"testing"

	"github.com/google/go-github/v44/github"
)

// createTestData is a helper function to create test cases that should never be returned. It should be passed the "successful" test string
func createTestData(passCase string) []*github.ReleaseAsset {

	var bogusString = "what.ami"
	var wrongOs = "file_other_os.zip"
	var wrongEnding = "file_linux_amd64.wrongending"

	assets := []*github.ReleaseAsset{
		{Name: &bogusString, BrowserDownloadURL: &bogusString},
		{Name: &wrongOs, BrowserDownloadURL: &wrongOs},
		{Name: &wrongEnding, BrowserDownloadURL: &wrongEnding},
		{Name: &passCase, BrowserDownloadURL: &passCase},
	}

	return assets
}

// TestFindAsset will test each passing asset type
func TestFindAsset(t *testing.T) {

	var passCases = []string{"file_linux_amd64.zip", // a zip
		"file_linux_amd64.tar",    // tar form 1
		"file_linux_amd64.tar.gz", // tar form 2
		"file_linux_amd64.tgz",    // tar form 3
		"file_linux_amd64.exe",    // an exe
		"file_linux_amd64",        // a binary
	}
	for _, testString := range passCases {
		name, _ := FindAsset("linux", "amd64", createTestData(testString))
		if name != testString {
			t.Fatalf("FindAsset test failed. %s does not match %s", name, testString)
		}
	}
}

// TestFindAssetNil should not find a match
func TestFindAssetNil(t *testing.T) {
	var nilString = "file_linux_amd64.nil"
	name, _ := FindAsset("linux", "amd64", createTestData(nilString))

	if name != "" {
		t.Fatalf("FindAsset nil test failed. Name should be empty!")
	}
}

func TestGetAssetbyName(t *testing.T) {
	var assetName = "my_awesome_file"
	name, _ := GetAssetbyName(assetName, createTestData(assetName))

	if name != assetName {
		t.Fatalf("%s should = %s", assetName, name)
	}
}
