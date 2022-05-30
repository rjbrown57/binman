package gh

import (
	"regexp"
	"strings"

	"github.com/google/go-github/v44/github"
)

// I should refactor this a bit to use a regex for Arch to interchange amd64 v x86_64
// rel* vars should come in a interface
func GetAssetbyName(relFileName string, assets []*github.ReleaseAsset) (string, string) {
	for _, asset := range assets {
		if *asset.Name == relFileName {
			return *asset.Name, *asset.BrowserDownloadURL
		}
	}

	return "", ""
}

// Return first asset that matches our OS and Arch regexes
func FindAsset(relArch string, relOS string, assets []*github.ReleaseAsset) (string, string) {
	for _, asset := range assets {
		an := strings.ToLower(*asset.Name)
		testOS, _ := regexp.MatchString(relArch, an)
		testArch, _ := regexp.MatchString(relOS, an)
		if testOS && testArch {
			return *asset.Name, *asset.BrowserDownloadURL
		}
	}

	return "", ""
}
