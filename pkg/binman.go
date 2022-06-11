package binman

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/go-github/v44/github"
	"github.com/rjbrown57/binman/pkg/gh"
	"github.com/sirupsen/logrus"
)

const timeout = 60 * time.Second

var log = logrus.New()

func syncRepo(ghClient *github.Client, releasePath string, rel BinmanRelease) error {

	var assetName, dlUrl string
	var err error
	ctx := context.Background()

	log.Debugf("release = %+v", rel)

	if rel.Version == "" {
		log.Infof("Querying github api for latest release of %s", rel.Repo)
		// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
		rel.GithubData, _, err = ghClient.Repositories.GetLatestRelease(ctx, rel.Org, rel.Project)
	} else {
		log.Infof("Querying github api for %s release of %s", rel.Version, rel.Repo)
		// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
		rel.GithubData, _, err = ghClient.Repositories.GetReleaseByTag(ctx, rel.Org, rel.Project, rel.Version)
	}

	if err != nil {
		log.Warnf("error listing releases %v", err)
		return err
	}

	// Get Path and Verify it DNE before digging through assets
	// If PublishPath is already set ignore these checks. This means we are doing a direct repo download
	if rel.PublishPath == "" {
		rel.setArtifactPath(releasePath, *rel.GithubData.TagName)
		_, err = os.Stat(rel.PublishPath)
		if err == nil {
			log.Warnf("Latest version is %s. %s is up to date", *rel.GithubData.TagName, rel.Repo)
			return err
		}
	}

	// If user has set an external url use that to grab target
	// Else Try to find the requested asset
	// User can provide an exact asset name via releaseFilename
	// binman will try to find the release via fileType,Arch
	if rel.ExternalUrl != "" {
		dlUrl = fmt.Sprintf(rel.ExternalUrl, *rel.GithubData.TagName)
		assetName = filepath.Base(dlUrl)
	} else {
		if rel.ReleaseFileName != "" {
			assetName, dlUrl = gh.GetAssetbyName(rel.ReleaseFileName, rel.GithubData.Assets)
		} else {
			assetName, dlUrl = gh.FindAsset(rel.Arch, rel.Os, rel.GithubData.Assets)
		}
	}

	if dlUrl == "" {
		log.Warnf("Target release asset not found for %s", rel.Repo)
		return nil
	}

	// Set paths based on asset we selected
	rel.setPublishPaths(releasePath, assetName)

	filePath := fmt.Sprintf("%s/%s", rel.PublishPath, assetName)

	// prepare directory path
	err = os.MkdirAll(rel.PublishPath, 0750)
	if err != nil {
		log.Warnf("Error creating %s - %v", rel.PublishPath, err)
		return err
	}

	// end pre steps

	// download file
	err = downloadFile(filePath, dlUrl)
	if err != nil {
		log.Warnf("Unable to download file : %v", err)
		return err
	}

	// If user has requested download only move to next release
	if rel.DownloadOnly {
		return nil
	}

	// untar file
	if isTar(filePath) {
		log.Debug("extract start")
		err = handleTar(rel.PublishPath, filePath)
		if err != nil {
			log.Warnf("Failed to extract file : %v", err)
		}
	}

	// make the file executable
	err = os.Chmod(rel.ArtifactPath, 0750)
	if err != nil {
		log.Warnf("Failed to set permissions on %s", rel.PublishPath)
		return err
	}

	// Create symlink
	err = createReleaseLink(rel.ArtifactPath, rel.LinkPath)
	if err != nil {
		log.Warnf("Failed to make symlink: %v", err)
		return err
	}

	// Verify symlink is good
	_, err = os.Stat(rel.LinkPath)
	if err != nil {
		log.Warnf("Issue with created symlink: %v", err)
		return err
	}
	log.Debugf("Symlink Created!")

	// Write Release
	relNotes := rel.GithubData.GetBody()
	if relNotes != "" {
		notePath := fmt.Sprintf("%s/releaseNotes.txt", rel.PublishPath)
		err := writeNotes(notePath, relNotes)
		if err != nil {
			log.Fatalf("Issue writing release notes: %v", err)
			return err
		}
		log.Debugf("Notes written to %s", notePath)
	}

	return nil
}

// Main does basic setup, then calls the appropriate functions for asset resolution
func Main(work map[string]string, debug bool, jsonLog bool) {

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

	if work["configFile"] != "" {
		log.Info("config sync")
		config := newGHBMConfig(work["configFile"])

		log.Debugf("Process %v Releases", len(config.Releases))

		ghClient := gh.GetGHCLient(config.Config.TokenVar)

		log.Debugf("config = %+v", config)

		for _, rel := range config.Releases {
			err := syncRepo(ghClient, config.Config.ReleasePath, rel)
			if err != nil {
				log.Warnf("Error syncing %s - %v\n", rel.Repo, err)
			}
		}
	} else {
		log.Info("direct repo download")
		ghClient := gh.GetGHCLient("none")

		cdir, err := os.Getwd()
		if err != nil {
			log.Fatal("Unable to get current working directory")
		}

		rel := BinmanRelease{
			Repo:         work["repo"],
			Os:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			PublishPath:  cdir,
			DownloadOnly: true,
			Version:      work["version"],
		}

		rel.getOR()

		err = syncRepo(ghClient, cdir, rel)
		if err != nil {
			log.Warnf("Error syncing %s - %v\n", rel.Repo, err)
		}
	}

	log.Info("binman finished!")
}
