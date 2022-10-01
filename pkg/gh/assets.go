package gh

import (
	"regexp"
	"strings"

	"github.com/google/go-github/v44/github"
)

const TarRegEx = `(\.tar$|\.tar\.gz$|\.tgz$)`
const ZipRegEx = `(\.zip$)`

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
		// Currently we handle binaries and tars
		testOS, _ := regexp.MatchString(relArch, an)
		testArch, _ := regexp.MatchString(relOS, an)
		// anything following by a "." and then any three characters
		binCheck, _ := regexp.MatchString(`.*\....$`, an)
		tarCheck, _ := regexp.MatchString(TarRegEx, an)
		zipCheck, _ := regexp.MatchString(ZipRegEx, an)
		exeCheck := strings.HasSuffix(an, ".exe")

		// If the asset matches OS/ARCH and binCheck is false or tarCheck is true or exe check is true
		if testOS && testArch && (!binCheck || tarCheck || zipCheck || exeCheck) {
			return *asset.Name, *asset.BrowserDownloadURL
		}
	}

	return "", ""
}
