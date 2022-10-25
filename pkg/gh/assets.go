package gh

import (
	"regexp"
	"strings"

	"github.com/google/go-github/v48/github"
)

const TarRegEx = `(\.tar$|\.tar\.gz$|\.tgz$)`
const ZipRegEx = `(\.zip$)`
const ExeRegex = `.*\.exe$`
const x86RegEx = `(amd64|x86_64)`

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

// FindAsset will Return first asset that matches our OS and Arch regexes and one of our supported filetypes
func FindAsset(relArch string, relOS string, assets []*github.ReleaseAsset) (string, string) {

	// sometimes amd64 is represented as x86_64, so we substitute a regex here that covers both
	if relArch == "amd64" {
		relArch = x86RegEx
	}

	zipRx := regexp.MustCompile(ZipRegEx)
	tarRx := regexp.MustCompile(TarRegEx)
	exeRx := regexp.MustCompile(ExeRegex)
	osRx := regexp.MustCompile(relOS)
	archRx := regexp.MustCompile(relArch)

	for _, asset := range assets {
		an := strings.ToLower(*asset.Name)

		// This asset matches our OS/Arch
		if osRx.MatchString(an) && archRx.MatchString(an) {
			// If asset matches one of our supported styles return name+download url
			// Current styles are tar,zip,exe,linux binary
			switch {
			case exeRx.MatchString(an), !strings.Contains(an, "."), tarRx.MatchString(an), zipRx.MatchString(an):
				return *asset.Name, *asset.BrowserDownloadURL

			}

		}

	}

	return "", ""
}
