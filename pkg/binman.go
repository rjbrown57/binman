package binman

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v48/github"
	"github.com/rjbrown57/binman/pkg/gh"
	log "github.com/rjbrown57/binman/pkg/logging"
)

const timeout = 60 * time.Second

func goSyncRepo(ghClient *github.Client, releasePath string, rel BinmanRelease, c chan<- BinmanMsg, wg *sync.WaitGroup) {
	defer wg.Done()

	var err error
	ctx := context.Background()

	log.Debugf("release %s = %+v", rel.Repo, rel)

	if rel.Version == "" {
		log.Debugf("Querying github api for latest release of %s", rel.Repo)
		// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
		rel.githubData, _, err = ghClient.Repositories.GetLatestRelease(ctx, rel.org, rel.project)
	} else {
		log.Debugf("Querying github api for %s release of %s", rel.Version, rel.Repo)
		// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
		rel.githubData, _, err = ghClient.Repositories.GetReleaseByTag(ctx, rel.org, rel.project, rel.Version)
	}

	if err != nil {
		log.Warnf("error listing releases %v", err)
		c <- BinmanMsg{rel: rel, err: err}
		return
	}

	// Get Path and Verify it DNE before digging through assets
	// If publishPath is already set ignore these checks. This means we are doing a direct repo download
	if rel.publishPath == "" {
		rel.setPublisPath(releasePath, *rel.githubData.TagName)
		_, err = os.Stat(rel.publishPath)
		if err == nil {
			log.Infof("Latest version is %s %s is up to date", *rel.githubData.TagName, rel.Repo)
			c <- BinmanMsg{rel: rel, err: err}
			return
		}
	}

	// If user has set an external url use that to grab target
	// Else Try to find the requested asset
	// User can provide an exact asset name via releaseFilename
	// binman will try to find the release via fileType,Arch
	if rel.ExternalUrl != "" {
		rel.dlUrl = formatString(rel.ExternalUrl, rel.getDataMap())
		log.Debugf("User specified url %s", rel.dlUrl)
		rel.assetName = filepath.Base(rel.dlUrl)
	} else {
		if rel.ReleaseFileName != "" {
			rFilename := formatString(rel.ReleaseFileName, rel.getDataMap())
			log.Debugf("Get asset by name %s", rFilename)
			rel.assetName, rel.dlUrl = gh.GetAssetbyName(rFilename, rel.githubData.Assets)
		} else {
			log.Debugf("Attempt to find asset %s", rel.ReleaseFileName)
			rel.assetName, rel.dlUrl = gh.FindAsset(rel.Arch, rel.Os, rel.githubData.Assets)
		}
	}

	if rel.dlUrl == "" {
		log.Warnf("Target release asset not found for %s", rel.Repo)
		c <- BinmanMsg{rel: rel, err: nil}
		return
	}

	// Set paths based on asset we selected
	rel.setArtifactPath(releasePath, rel.assetName)

	// prepare directory path
	err = os.MkdirAll(rel.publishPath, 0750)
	if err != nil {
		log.Warnf("Error creating %s - %v", rel.publishPath, err)
		c <- BinmanMsg{rel: rel, err: err}
		return
	}

	// end pre steps

	// Collect post step tasks
	rel.getPostStepTasks()

	log.Debugf("Performing %d tasks for %s", len(rel.tasks), rel.Repo)

	for _, task := range rel.tasks {
		log.Debugf("Running task %s for %s", reflect.TypeOf(task), rel.Repo)
		err = task.execute()
		if err != nil {
			log.Warnf("Unable to complete task %s : %v", reflect.TypeOf(task), err)
			c <- BinmanMsg{rel: rel, err: err}
			return
		}
	}

	c <- BinmanMsg{rel: rel, err: nil}
}

// Main does basic setup, then calls the appropriate functions for asset resolution
func Main(work map[string]string, debug bool, jsonLog bool) {

	// Set the logging options
	log.ConfigureLog(jsonLog, debug)
	log.Infof("binman sync begin")

	c := make(chan BinmanMsg)
	var wg sync.WaitGroup
	var releases []BinmanRelease
	var ghClient *github.Client
	var releasePath string

	// Create config object.
	// setBaseConfig will return the appropriate base config file.
	// setConfig will check for a contextual config and merge with our base config and return the result
	config := SetConfig(SetBaseConfig(work["configFile"]))

	log.Debugf("binman config = %+v", config)

	// get github client
	ghClient = gh.GetGHCLient(config.Config.TokenVar)

	// This should be refactored to be simplified
	if work["repo"] != "" {
		var err error
		log.Infof("direct repo download")

		if !strings.Contains(work["repo"], "/") {
			log.Fatalf("Provided repo %s must be in the format org/repo", work["repo"])
		}

		releasePath, err = os.Getwd()
		if err != nil {
			log.Fatalf("Unable to get current working directory")
		}

		rel := BinmanRelease{
			Repo:         work["repo"],
			Os:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			publishPath:  releasePath,
			DownloadOnly: true,
			Version:      work["version"],
		}

		rel.getOR()

		releases = []BinmanRelease{rel}
	} else {
		log.Debugf("config file based sync")
		releases = config.Releases
		log.Debugf("Process %v Releases", len(releases))
		releasePath = config.Config.ReleasePath
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

	log.Infof("binman finished!")
}
