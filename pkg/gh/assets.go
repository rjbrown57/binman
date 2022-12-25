package gh

import (
	"regexp"
	"strings"

	"github.com/google/go-github/v48/github"
	log "github.com/rjbrown57/binman/pkg/logging"
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
			log.Debugf("Selected asset == %+v\n", *asset.Name)
			return *asset.Name, *asset.BrowserDownloadURL
		}
	}

	return "", ""
}

// FindAsset will Return first asset that matches our OS and Arch regexes and one of our supported filetypes
func FindAsset(relArch string, relOS string, version string, project string, assets []*github.ReleaseAsset) (string, string) {

	var possibleAsset github.ReleaseAsset

	// sometimes amd64 is represented as x86_64, so we substitute a regex here that covers both
	if relArch == "amd64" {
		relArch = x86RegEx
	}

	zipRx := regexp.MustCompile(ZipRegEx)
	tarRx := regexp.MustCompile(TarRegEx)
	exeRx := regexp.MustCompile(ExeRegex)
	osRx := regexp.MustCompile(relOS)
	archRx := regexp.MustCompile(relArch)

	// There are exact match assets and possible match assets
	// any 1 exact match asset will terminate the loop, otherwise we will take the last possible match asset

	for _, asset := range assets {
		an := strings.ToLower(asset.GetName())

		// This asset matches our OS/Arch
		if osRx.MatchString(an) && archRx.MatchString(an) {
			log.Debugf("Evaluating asset %s\n %v\n", an, asset)

			// If asset is an exact match one of our supported styles return name+download url
			// Current styles are tar,zip,exe,linux binary
			switch {
			case exeRx.MatchString(an), !strings.Contains(an, "."), tarRx.MatchString(an), zipRx.MatchString(an):
				log.Debugf("Selected asset %s == %+v\n", an, asset)
				return asset.GetName(), asset.GetBrowserDownloadURL()
			}

			log.Debugf("Evaluating %s contains version %s", an, version)
			if strings.Contains(an, version) || strings.Contains(an, strings.Trim(version, "v")) && strings.Contains(an, project) {
				log.Debugf("Possible match by version %s %s", version, asset.GetName())
				possibleAsset = *asset

			}
		}

	}

	return possibleAsset.GetName(), possibleAsset.GetBrowserDownloadURL()
}
