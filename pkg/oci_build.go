package binman

import (
	"fmt"
	"time"

	db "github.com/rjbrown57/binman/pkg/db"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/rjbrown57/binman/pkg/oci"
	bolt "go.etcd.io/bbolt"
)

func getAssets(r []BinmanRelease) ([]string, error) {

	var assets []string

	bdb := db.GetDB("", bolt.Options{Timeout: 1 * time.Second, ReadOnly: false})

	for _, rel := range r {
		// Collect All Versions
		err := rel.getVersions(bdb)
		if err != nil {
			log.Warnf("Unable to get all versions for %s %s", rel.Repo, err)
			continue
		}

		rel.versions, err = sortSemvers(rel.versions)
		if err != nil {
			log.Warnf("Unable to sort semvers for %s %s", rel.Repo, err)
			continue
		}

		byteData, err := db.GetData(fmt.Sprintf("%s/%s/%s/data", rel.SourceIdentifier, rel.Repo, rel.versions[0]), bdb)
		if err != nil {
			log.Warnf("Issue getting data for %s/%s", rel.Repo, rel.versions[0])
		}

		d := bytesToData(byteData)

		log.Debugf("Adding %s to image", d["linkPath"].(string))
		assets = append(assets, d["linkPath"].(string))
	}

	err := bdb.Close()
	if err != nil {
		log.Fatalf("Unable to close db %s", err)
	}

	return assets, nil
}

func BuildOciImage(config, repo, targetImageName, baseImage, imagePath string) error {

	img, err := oci.MakeBinmanImageBuild(targetImageName, imagePath, baseImage)
	if err != nil {
		return err
	}

	c := NewBMConfig(config).SetConfig(false)

	switch repo {
	case "":
		img.Assets, err = getAssets(c.Releases)
	default:
		r, err := c.GetRelease(repo)
		if err != nil {
			return err
		}
		img.Assets, err = getAssets([]BinmanRelease{r})
	}

	if err != nil {
		log.Fatalf("Issue getting populating asset list for image %s", err)
	}

	return oci.BuildOciImage(&img)
}
