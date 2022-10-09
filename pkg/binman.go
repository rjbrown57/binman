package binman

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/google/go-github/v44/github"
	"github.com/rjbrown57/binman/pkg/gh"
	"github.com/sirupsen/logrus"
)

const timeout = 60 * time.Second

var log = logrus.New()

func goSyncRepo(ghClient *github.Client, releasePath string, rel BinmanRelease, c chan<- BinmanMsg, wg *sync.WaitGroup) {
	defer wg.Done()

	var assetName, dlUrl string
	var err error
	ctx := context.Background()

	log.Debugf("release %s = %+v", rel.Repo, rel)

	if rel.Version == "" {
		log.Debugf("Querying github api for latest release of %s", rel.Repo)
		// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
		rel.GithubData, _, err = ghClient.Repositories.GetLatestRelease(ctx, rel.Org, rel.Project)
	} else {
		log.Debugf("Querying github api for %s release of %s", rel.Version, rel.Repo)
		// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
		rel.GithubData, _, err = ghClient.Repositories.GetReleaseByTag(ctx, rel.Org, rel.Project, rel.Version)
	}

	if err != nil {
		log.Warnf("error listing releases %v", err)
		c <- BinmanMsg{rel: rel, err: err}
		return
	}

	// Get Path and Verify it DNE before digging through assets
	// If PublishPath is already set ignore these checks. This means we are doing a direct repo download
	if rel.PublishPath == "" {
		rel.setArtifactPath(releasePath, *rel.GithubData.TagName)
		_, err = os.Stat(rel.PublishPath)
		if err == nil {
			log.Infof("Latest version is %s %s is up to date", *rel.GithubData.TagName, rel.Repo)
			c <- BinmanMsg{rel: rel, err: err}
			return
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
		c <- BinmanMsg{rel: rel, err: nil}
		return
	}

	// Set paths based on asset we selected
	rel.setPublishPaths(releasePath, assetName)

	filePath := fmt.Sprintf("%s/%s", rel.PublishPath, assetName)

	// prepare directory path
	err = os.MkdirAll(rel.PublishPath, 0750)
	if err != nil {
		log.Warnf("Error creating %s - %v", rel.PublishPath, err)
		c <- BinmanMsg{rel: rel, err: err}
		return
	}

	// end pre steps

	// download file
	err = downloadFile(filePath, dlUrl)
	if err != nil {
		log.Warnf("Unable to download file : %v", err)
		c <- BinmanMsg{rel: rel, err: err}
		return
	}

	// If user has requested download only move to next release
	if rel.DownloadOnly {
		c <- BinmanMsg{rel: rel, err: err}
		return
	}

	// untar file
	if isTar(filePath) {
		log.Debug("tar extract start")
		err = handleTar(rel.PublishPath, filePath)
		if err != nil {
			log.Warnf("Failed to extract tar file: %v", err)
			c <- BinmanMsg{rel: rel, err: err}
			return
		}
	} else if isZip(filePath) {
		log.Debug("zip extract start")
		err = handleZip(rel.PublishPath, filePath)
		if err != nil {
			log.Warnf("Failed to extract zip file: %v", err)
			c <- BinmanMsg{rel: rel, err: err}
			return
		}
	}

	// If the file still doesn't exist, attempt to find it in sub-directories
	if _, err := os.Stat(rel.ArtifactPath); errors.Is(err, os.ErrNotExist) {
		log.Debugf("Wasn't able to find the artifact at %s, walking the directory to see if we can find it",
			rel.ArtifactPath)
		targetFileName := filepath.Base(rel.ArtifactPath)
		_ = filepath.Walk(rel.PublishPath, func(path string, info os.FileInfo, err error) error {
			log.Debugf("Checking %s, against %s...", targetFileName, info.Name())
			if err == nil && targetFileName == info.Name() {
				log.Debugf("Found match! Using %s as the new artifact path.", path)
				rel.ArtifactPath = path
				return nil
			}
			return nil
		})
		if _, err := os.Stat(rel.ArtifactPath); errors.Is(err, os.ErrNotExist) {
			err := fmt.Errorf("Unable to find file matching '%s' anywhere in the release archive", targetFileName)
			log.Warnf("%v", err)
			c <- BinmanMsg{rel: rel, err: err}
			return
		}
	}

	// make the file executable
	err = os.Chmod(rel.ArtifactPath, 0750)
	if err != nil {
		log.Warnf("Failed to set permissions on %s", rel.PublishPath)
		c <- BinmanMsg{rel: rel, err: err}
		return
	}

	// Create symlink
	err = createReleaseLink(rel.ArtifactPath, rel.LinkPath)
	if err != nil {
		log.Warnf("Failed to make symlink: %v", err)
		c <- BinmanMsg{rel: rel, err: err}
		return
	}

	// Verify symlink is good
	_, err = os.Stat(rel.LinkPath)
	if err != nil {
		log.Warnf("Issue with created symlink: %v", err)
		c <- BinmanMsg{rel: rel, err: err}
		return
	}
	log.Debugf("Symlink Created!")

	// Write Release
	relNotes := rel.GithubData.GetBody()
	if relNotes != "" {
		notePath := fmt.Sprintf("%s/releaseNotes.txt", rel.PublishPath)
		err := writeStringtoFile(notePath, relNotes)
		if err != nil {
			log.Fatalf("Issue writing release notes: %v", err)
			c <- BinmanMsg{rel: rel, err: err}
			return
		}
		log.Debugf("Notes written to %s", notePath)
	}

	// IF enabled shrink via upx
	if rel.UpxConfig.Enabled == "true" {

		args := []string{rel.ArtifactPath}
		// If user supplied extra args add them
		if len(rel.UpxConfig.Args) != 0 {
			args = append(args, rel.UpxConfig.Args...)
		}

		log.Infof("Start upx on %s\n", rel.ArtifactPath)
		out, err := exec.Command("upx", args...).Output()

		if err != nil {
			c <- BinmanMsg{rel: rel, err: err}
			return
		}

		log.Infof("Upx complete on %s\n", rel.ArtifactPath)
		log.Debugf("Upx output %s\n", out)
	}

	c <- BinmanMsg{rel: rel, err: nil}
	return
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

	c := make(chan BinmanMsg)
	var wg sync.WaitGroup
	var releases []BinmanRelease
	var ghClient *github.Client
	var releasePath string

	// Create default path + config file if necessary
	if work["configFile"] == "default" {
		work["configFile"] = mustEnsureDefaultPaths()
	}

	// This should be refactored to be simplified
	if work["repo"] == "" {
		log.Debug("config sync")
		config := newGHBMConfig(work["configFile"])
		log.Debugf("config = %+v", config)

		releases = config.Releases
		log.Debugf("Process %v Releases", len(releases))
		releasePath = config.Config.ReleasePath

		ghClient = gh.GetGHCLient(config.Config.TokenVar)
	} else {
		var err error
		log.Info("direct repo download")
		ghClient = gh.GetGHCLient("none")

		releasePath, err = os.Getwd()
		if err != nil {
			log.Fatal("Unable to get current working directory")
		}

		rel := BinmanRelease{
			Repo:         work["repo"],
			Os:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			PublishPath:  releasePath,
			DownloadOnly: true,
			Version:      work["version"],
		}

		rel.getOR()

		releases = []BinmanRelease{rel}
	}

	// https://github.com/lotusirous/go-concurrency-patterns/blob/main/2-chan/main.go
	for _, rel := range releases {
		wg.Add(1)
		go goSyncRepo(ghClient, releasePath, rel, c, &wg)
	}

	go func(c chan BinmanMsg, wg *sync.WaitGroup) {
		wg.Wait()
		close(c)
	}(c, &wg)

	for msg := range c {
		if msg.err != nil {
			log.Debugf("Repo %s, Error %q\n", msg.rel.Repo, msg.err)
		}
	}

	log.Info("binman finished!")
}
