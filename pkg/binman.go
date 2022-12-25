package binman

import (
	"os"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/google/go-github/v48/github"
	"github.com/rjbrown57/binman/pkg/gh"
	log "github.com/rjbrown57/binman/pkg/logging"
)

const timeout = 60 * time.Second

// goSyncRepo calls setTasks to arrange all work, then execute each task sequentially
func goSyncRepo(ghClient *github.Client, releasePath string, rel BinmanRelease, c chan<- BinmanMsg, wg *sync.WaitGroup) {
	defer wg.Done()

	var err error

	log.Debugf("release %s = %+v", rel.Repo, rel)

	actions := rel.setPreActions(ghClient, releasePath)
	log.Debugf("Performing %d pre actions for %s", len(actions), rel.Repo)

	for _, task := range actions {
		log.Debugf("Executing %s for %s", reflect.TypeOf(task), rel.Repo)
		err = task.execute()
		// this is hacky, but catches error
		if err != nil && err.Error() == "Noupdate" {
			return
		} else if err != nil {
			log.Warnf("Unable to complete action %s : %v", reflect.TypeOf(task), err)
			c <- BinmanMsg{rel: rel, err: err}
			return
		}
	}

	actions = rel.setPostActions()
	log.Debugf("Performing %d post actions for %s", len(actions), rel.Repo)
	for _, action := range actions {
		log.Debugf("Running task %s for %s", reflect.TypeOf(action), rel.Repo)
		err = action.execute()
		if err != nil {
			log.Warnf("Unable to complete task %s : %v", reflect.TypeOf(action), err)
			c <- BinmanMsg{rel: rel, err: err}
			return
		}
	}

	c <- BinmanMsg{rel: rel, err: nil}
}

func BinmanGetReleasePrep(work map[string]string) []BinmanRelease {

	log.Infof("direct repo download")

	if f, err := os.Stat(work["path"]); !f.IsDir() || err != nil {
		log.Fatalf("Issue detected with %s", work["path"])
	}

	rel := BinmanRelease{
		Repo:             work["repo"],
		Os:               runtime.GOOS,
		Arch:             runtime.GOARCH,
		publishPath:      work["path"],
		QueryType:        "release",
		DownloadOnly:     true,
		cleanupOnFailure: false,
		Version:          work["version"],
	}

	if rel.Version != "" {
		rel.QueryType = "releasebytag"
	}

	rel.getOR()

	return []BinmanRelease{rel}

}

// Main does basic setup, then calls the appropriate functions for asset resolution
func Main(args map[string]string, debug bool, jsonLog bool, launchCommand string) {

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
	config := SetConfig(SetBaseConfig(args["configFile"]))

	log.Debugf("binman config = %+v", config)

	// get github client
	ghClient = gh.GetGHCLient(config.Config.TokenVar)

	switch launchCommand {
	case "get":
		releases = BinmanGetReleasePrep(args)
	case "config":
		releases = config.Releases
		releasePath = config.Config.ReleasePath
	}

	log.Debugf("Process %v Releases", len(releases))

	// https://github.com/lotusirous/go-concurrency-patterns/blob/main/2-chan/main.go
	for _, rel := range releases {
		wg.Add(1)
		go goSyncRepo(ghClient, releasePath, rel, c, &wg)
	}

	go func(c chan BinmanMsg, wg *sync.WaitGroup) {
		wg.Wait()
		close(c)
	}(c, &wg)

	// Process results
	for msg := range c {
		if msg.err == nil {
			continue
		}

		log.Warnf("Repo %s, Error %q\n", msg.rel.Repo, msg.err)
		if msg.rel.cleanupOnFailure {
			err := os.RemoveAll(msg.rel.publishPath)
			if err != nil {
				log.Warnf("Unable to clean up %s - %s", msg.rel.publishPath, err)
			}
			log.Warnf("cleaned %s\n", msg.rel.publishPath)
			log.Debugf("Final release data  %+v\n", msg.rel)
		}
	}

	log.Infof("binman finished!")
}
