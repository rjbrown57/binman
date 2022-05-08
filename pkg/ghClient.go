package binman

import (
	"context"
	"os"
	"strings"

	"github.com/google/go-github/v44/github"
	"golang.org/x/oauth2"
)

// GetGHClient will get a go-github client with auth for api access
func GetGHCLient(tokenvar string) *github.Client {

	// No auth client if user does not supply envvar
	if tokenvar == "none" {
		log.Debugf("Creating GH client without auth")
		return github.NewClient(nil)
	}

	log.Debugf("Creating GH client with auth from $%s", tokenvar)
	ghtoken := os.Getenv(tokenvar)
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghtoken},
	)

	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}

// Add some for of validation here
func getOR(ghRepo string) (string, string) {
	retVal := strings.Split(ghRepo, "/")
	return retVal[0], retVal[1]
}

// checkRelease is used to verify this is the correct asset from GH
func findRelbyType(assetName string, fileType string, arch string, os string) bool {
	if strings.HasSuffix(assetName, fileType) && strings.Contains(assetName, arch) && strings.Contains(assetName, os) {
		return true
	}
	return false
}

// checkRelease is used to verify this is the correct asset from GH
func findRelbyName(assetName string, fileName string) bool {
	if assetName == fileName {
		return true
	}
	return false
}

// I should refactor this a bit to use a regex for Arch to interchange amd64 v x86_64
func getAsset(rel GHBMRelease, assets []*github.ReleaseAsset) (string, string) {
	for _, asset := range assets {
		assetName := *asset.Name
		log.Debugf("%#v checking for %s - %s", assetName, rel.FileType, rel.Arch)
		if findRelbyName(strings.ToLower(assetName), rel.ReleaseFileName) || findRelbyType(strings.ToLower(assetName), rel.FileType, rel.Arch, rel.Os) {
			return assetName, *asset.BrowserDownloadURL
		}
	}

	return "", ""
}
