package gh

import (
	"testing"

	"github.com/google/go-github/v50/github"
)

// createTestData is a helper function to create test cases that should never be returned. It should be passed the "successful" test string
func createTestData(passCase string) []*github.ReleaseAsset {

	var bogusString = "what.ami"
	var wrongOs = "file_other_os.zip"
	var wrongEnding = "file_linux_amd64.wrongending"
	var possibleMatch = "file_v0.0.0_linux_amd64"
	var possibleMatch2 = "file_0.0.0_linux_amd64"

	assets := []*github.ReleaseAsset{
		{Name: &bogusString, BrowserDownloadURL: &bogusString},
		{Name: &wrongOs, BrowserDownloadURL: &wrongOs},
		{Name: &wrongEnding, BrowserDownloadURL: &wrongEnding},
		{Name: &passCase, BrowserDownloadURL: &passCase},
		{Name: &passCase, BrowserDownloadURL: &passCase},
		{Name: &passCase, BrowserDownloadURL: &possibleMatch},
		{Name: &passCase, BrowserDownloadURL: &possibleMatch2},
	}

	return assets
}

// TestFindAsset will test each passing asset type
func TestFindAsset(t *testing.T) {

	var passCases = []string{
		"file_linux_amd64.zip",    // a zip
		"file_linux_amd64.tar",    // tar form 1
		"file_linux_amd64.tar.gz", // tar form 2
		"file_linux_amd64.tgz",    // tar form 3
		"file_linux_amd64.exe",    // an exe
		"file_linux_amd64",        // a binary
		"file_v0.0.0_linux_amd64", // possible match 1
		"file_0.0.0_linux_amd64",  // possible match 2
	}
	for _, testString := range passCases {
		name, _ := FindAsset("linux", "amd64", "v0.0.0", "file", createTestData(testString))
		if name != testString {
			t.Fatalf("FindAsset test failed. %s does not match %s", name, testString)
		}
	}

	// find nothing
	var nilString = "file_linux_amd64.nil"
	name, _ := FindAsset("linux", "amd64", "v0.0.0", "file", createTestData(nilString))

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
