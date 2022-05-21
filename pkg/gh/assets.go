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

func GetAssetbyType(relFileType string, relArch string, relOs string, assets []*github.ReleaseAsset) (string, string) {
	for _, asset := range assets {
		an := strings.ToLower(*asset.Name)
		testArch, _ := regexp.MatchString(relArch, an)
		testFileType, _ := regexp.MatchString(relFileType, an)
		if testFileType && testArch && strings.Contains(an, relOs) {
			return *asset.Name, *asset.BrowserDownloadURL
		}
	}

	return "", ""
}
