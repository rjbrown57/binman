package binman

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rjbrown57/binman/pkg/gh"
	"github.com/sirupsen/logrus"
)

const timeout = 60 * time.Second

var log = logrus.New()

// This function needs to be massively simplified
// TODO Use of rel/release is hard to read. Refactor to make names more unique
func Main(configFile string, debug bool, jsonLog bool) {

	// logging
	if jsonLog {
		log.Formatter = &logrus.JSONFormatter{}
	}

	log.Out = os.Stdout

	if debug {
		log.Level = logrus.DebugLevel
	} else {
		log.Level = logrus.InfoLevel
	}

	log.Info("binman sync begin")

	config := newGHBMConfig(configFile)

	log.Debugf("Process %v Releases", len(config.Releases))

	ghClient := gh.GetGHCLient(config.Config.TokenVar)

	log.Debugf("config = %+v", config)

	//Learn about contexts
	ctx := context.Background()
	//defer cancel()

	for _, rel := range config.Releases {

		log.Infof("Querying github api for latest release of %s", rel.Repo)
		// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
		release, _, err := ghClient.Repositories.GetLatestRelease(ctx, rel.Org, rel.Project)

		if err != nil {
			log.Fatalf("error listing releases %v", err)
			continue
		}

		// Get Path and Verify it DNE before digging through assets
		rel.setArtifactPath(config.Config.ReleasePath, *release.TagName)
		log.Debugf("release = %+v", rel)

		_, err = os.Stat(rel.PublishPath)
		if err == nil {
			log.Warnf("Latest version is %s. %s is up to date", *release.TagName, rel.Repo)
			continue
		}

		var assetName, dlUrl string

		// If user has set an external url use that to grab target
		// Else Try to find the requested asset
		// User can provide an exact asset name via releaseFilename
		// binman will try to find the release via fileType,Arch
		if rel.ExternalUrl != "" {
			dlUrl = fmt.Sprintf(rel.ExternalUrl, *release.TagName)
			assetName = filepath.Base(dlUrl)

		} else {
			if rel.ReleaseFileName != "" {
				assetName, dlUrl = gh.GetAssetbyName(rel.ReleaseFileName, release.Assets)
			} else {
				assetName, dlUrl = gh.FindAsset(rel.Arch, rel.Os, release.Assets)
			}
		}

		if dlUrl == "" {
			log.Warnf("Target release asset not found for %s", rel.Repo)
			continue
		}

		// Set paths based on asset we selected
		rel.setPublishPaths(config.Config.ReleasePath, assetName)

		filePath := fmt.Sprintf("%s/%s", rel.PublishPath, assetName)
		// filePath should be set in the rel.setPublishPaths

		// prepare directory path
		err = os.MkdirAll(rel.PublishPath, 0750)
		if err != nil {
			log.Warnf("Error creating %s - %v", rel.PublishPath, err)
			continue
		}

		// download file
		err = downloadFile(filePath, dlUrl)
		if err != nil {
			log.Warnf("Unable to download file : %v", err)
			continue
		}

		// If user has requested download only move to next release
		if rel.DownloadOnly {
			continue
		}

		// untar file
		if isTar(filePath) {
			log.Debug("extract start")
			err = handleTar(rel.PublishPath, filePath)
			if err != nil {
				log.Warnf("Failed to extract file : %v", err)
				continue
			}
		}

		// make the file executable
		err = os.Chmod(rel.ArtifactPath, 0750)
		if err != nil {
			log.Warnf("Failed to set permissions on %s", rel.PublishPath)
			continue
		}

		// Create symlink
		err = createReleaseLink(rel.ArtifactPath, rel.LinkPath)
		if err != nil {
			log.Warnf("Failed to make symlink: %v", err)
			continue
		}

		// Verify symlink is good
		_, err = os.Stat(rel.LinkPath)
		if err != nil {
			log.Warnf("Issue with created symlink: %v", err)
			continue
		}
		log.Debugf("Symlink Created!")

		// Write Release
		relNotes := release.GetBody()
		if relNotes != "" {
			notePath := fmt.Sprintf("%s/releaseNotes.txt", rel.PublishPath)
			err := writeNotes(notePath, relNotes)
			if err != nil {
				log.Fatalf("Issue writing release notes: %v", err)
				continue
			}
			log.Debugf("Notes written to %s", notePath)
		}
	}
	log.Info("binman finished!")
}
