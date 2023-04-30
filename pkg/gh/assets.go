package gh

import (
	"strings"

	"github.com/google/go-github/v50/github"
	log "github.com/rjbrown57/binman/pkg/logging"
)

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

// GHGetAssetData will create a map of names + download urls
func GHGetAssetData(assets []*github.ReleaseAsset) map[string]string {
	m := make(map[string]string)

	// create map of names + download urls
	for _, asset := range assets {
		m[strings.ToLower(asset.GetName())] = asset.GetBrowserDownloadURL()
	}

	return m
}
