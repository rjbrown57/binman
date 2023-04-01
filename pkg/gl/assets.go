package gl

import (
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/xanzy/go-gitlab"
)

func GLGetReleaseAssets(glClient *gitlab.Client, repo string, tag string) []*gitlab.ReleaseLink {
	rel, _, err := glClient.Releases.GetRelease(repo, tag)

	if err != nil {
		log.Debugf("Error getting release for %s:%s - %v", repo, tag, err)
	}

	return rel.Assets.Links
}

const TarRegEx = `(\.tar$|\.tar\.gz$|\.tgz$)`
const ZipRegEx = `(\.zip$)`
const ExeRegex = `.*\.exe$`
const x86RegEx = `(amd64|x86_64)`
const macOsRx = `(darwin|macos)`

func GetAssetbyName(relFileName string, assets []*gitlab.ReleaseLink) (string, string) {
	for _, asset := range assets {
		an := strings.ToLower(filepath.Base(asset.DirectAssetURL))

		if an == relFileName {
			log.Debugf("Selected asset == %+v\n", an)
			return an, *&asset.DirectAssetURL
		}
	}

	return "", ""
}

// FindAsset will Return first asset that matches our OS and Arch regexes and one of our supported filetypes
func FindAsset(relArch string, relOS string, version string, project string, assets []*gitlab.ReleaseLink) (string, string) {

	var possibleAsset *gitlab.ReleaseLink

	// sometimes amd64 is represented as x86_64, so we substitute a regex here that covers both
	if relArch == "amd64" {
		relArch = x86RegEx
	}

	// gitlab refers to darwin and "macos"  so we substitute a regex here that covers both
	if relOS == "darwin" {
		relOS = macOsRx
	}

	zipRx := regexp.MustCompile(ZipRegEx)
	tarRx := regexp.MustCompile(TarRegEx)
	exeRx := regexp.MustCompile(ExeRegex)
	osRx := regexp.MustCompile(relOS)
	archRx := regexp.MustCompile(relArch)

	// There are exact match assets and possible match assets
	// any 1 exact match asset will terminate the loop, otherwise we will take the last possible match asset

	for _, asset := range assets {
		an := strings.ToLower(filepath.Base(asset.DirectAssetURL))

		// This asset matches our OS/Arch
		if osRx.MatchString(an) && archRx.MatchString(an) {
			log.Debugf("Evaluating asset %s\n %v\n", an, asset)

			// If asset is an exact match one of our supported styles return name+download url
			// Current styles are tar,zip,exe,linux binary
			switch {
			case exeRx.MatchString(an), !strings.Contains(an, "."), tarRx.MatchString(an), zipRx.MatchString(an):
				log.Debugf("Selected asset %s == %+v\n", an, asset)
				return an, asset.DirectAssetURL
			}

			log.Debugf("Evaluating %s contains version %s", an, version)
			if strings.Contains(an, version) || strings.Contains(an, strings.Trim(version, "v")) && strings.Contains(an, project) {
				log.Debugf("Possible match by version %s %s", version, an)
				possibleAsset = asset

			}
		}

	}

	log.Debugf("returning partial match %s %s", project, version)
	return strings.ToLower(filepath.Base(possibleAsset.DirectAssetURL)), possibleAsset.DirectAssetURL

}
