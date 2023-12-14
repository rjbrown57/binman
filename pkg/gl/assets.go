package gl

import (
	"path/filepath"
	"strings"

	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/xanzy/go-gitlab"
)

func GLGetReleaseAssets(glClient *gitlab.Client, repo string, tag string) ([]*gitlab.ReleaseLink, int64) {
	rel, _, err := glClient.Releases.GetRelease(repo, tag)

	if err == nil {
		return rel.Assets.Links, rel.CreatedAt.Unix()
	}

	log.Debugf("Error getting release for %s:%s - %v", repo, tag, err)

	return nil, 0
}

func GetAssetbyName(relFileName string, assets []*gitlab.ReleaseLink) (string, string) {
	for _, asset := range assets {
		an := strings.ToLower(filepath.Base(asset.DirectAssetURL))

		if an == relFileName {
			log.Debugf("Selected asset == %+v\n", an)
			return an, asset.DirectAssetURL
		}
	}

	return "", ""
}

// GLGetAssetData will create a map of names + download urls
func GLGetAssetData(assets []*gitlab.ReleaseLink) map[string]string {

	m := make(map[string]string)

	for _, asset := range assets {
		m[strings.ToLower(filepath.Base(asset.DirectAssetURL))] = asset.DirectAssetURL
	}

	return m
}
