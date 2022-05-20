package gh

import (
	"strings"

	"github.com/google/go-github/v44/github"
)

// checkRelease is used to verify this is the correct asset from GH
func FindRelbyType(assetName string, fileType string, arch string, os string) bool {
	if strings.HasSuffix(assetName, fileType) && strings.Contains(assetName, arch) && strings.Contains(assetName, os) {
		return true
	}
	return false
}

// checkRelease is used to verify this is the correct asset from GH
func FindRelbyName(assetName string, fileName string) bool {
	if assetName == fileName {
		return true
	}
	return false
}

// I should refactor this a bit to use a regex for Arch to interchange amd64 v x86_64
// rel* vars should come in a interface
func GetAsset(relFileName string, relFileType string, relArch string, relOs string, assets []*github.ReleaseAsset) (string, string) {
	for _, asset := range assets {
		assetName := *asset.Name
		if FindRelbyName(strings.ToLower(assetName), relFileName) || FindRelbyType(strings.ToLower(assetName), relFileType, relArch, relOs) {
			return assetName, *asset.BrowserDownloadURL
		}
	}

	return "", ""
}
