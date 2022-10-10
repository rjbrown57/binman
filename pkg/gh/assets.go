package gh

import (
	"regexp"
	"strings"

	"github.com/google/go-github/v44/github"
)

const TarRegEx = `(\.tar$|\.tar\.gz$|\.tgz$)`
const ZipRegEx = `(\.zip$)`
const ExeRegex = `.*\.exe$`

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
	for _, asset := range assets {
		an := strings.ToLower(*asset.Name)

		// Config we have an os/arch match
		testOS, _ := regexp.MatchString(relOS, an)
		testArch, _ := regexp.MatchString(relArch, an)

		// This asset matches our OS/Arch
		if testArch && testOS {
			zipRx := regexp.MustCompile(ZipRegEx)
			tarRx := regexp.MustCompile(TarRegEx)
			exeRx := regexp.MustCompile(ExeRegex)

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
