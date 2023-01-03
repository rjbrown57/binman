package binman

import (
	"fmt"
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

var spinChan = make(chan string)
var swg sync.WaitGroup

// goSyncRepo calls setTasks to arrange all work, then execute each task sequentially
func goSyncRepo(ghClient *github.Client, rel BinmanRelease, c chan<- BinmanMsg, wg *sync.WaitGroup) {
	defer wg.Done()

	var err error

	log.Debugf("release %s = %+v", rel.Repo, rel)

	actions := rel.setPreActions(ghClient, rel.ReleasePath)
	log.Debugf("Performing %d pre actions for %s", len(actions), rel.Repo)

	for _, task := range actions {
		log.Debugf("Executing %s for %s", reflect.TypeOf(task), rel.Repo)
		err = task.execute()
		// this is hacky, but catches error
		if err != nil && err.Error() == "Noupdate" {
			log.Debugf("%s(%s) is up to date", rel.Repo, rel.Version)
			c <- BinmanMsg{rel: rel, err: err}
			return
		} else if err != nil {
			log.Debugf("Unable to complete action %s : %v", reflect.TypeOf(task), err)
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
			log.Debugf("Unable to complete task %s : %v", reflect.TypeOf(action), err)
			c <- BinmanMsg{rel: rel, err: err}
			return
		}
	}

	actions = rel.setOsCommands()
	if len(actions) > 0 {
		log.Debugf("Performing %d OS commands for %s", len(actions), rel.Repo)
		for _, action := range actions {
			err = action.execute()
			if err != nil {
				log.Debugf("Unable to complete task %s : %v", reflect.TypeOf(action), err)
				c <- BinmanMsg{rel: rel, err: err}
				return
			}
		}
	}

	actions = rel.setFinalActions()
	if len(actions) > 0 {
		log.Debugf("Performing %d final actions for %s", len(actions), rel.Repo)
		for _, action := range actions {
			err = action.execute()
			if err != nil {
				log.Debugf("Unable to complete task %s : %v", reflect.TypeOf(action), err)
				c <- BinmanMsg{rel: rel, err: err}
				return
			}
		}
	}

	c <- BinmanMsg{rel: rel, err: nil}
}

func BinmanGetReleasePrep(work map[string]string) []BinmanRelease {

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
func Main(args map[string]string, debug bool, jsonLog bool, table bool, launchCommand string) {

	// Set the logging options
	log.ConfigureLog(jsonLog, debug)
	log.Debugf("binman sync begin\n")

	c := make(chan BinmanMsg)
	output := make(map[string][]BinmanMsg)
	var wg sync.WaitGroup
	var releases []BinmanRelease
	var ghClient *github.Client

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
	}

	go getSpinner(debug)

	relLength := len(releases)
	log.Debugf("Process %v Releases", relLength)
	swg.Add(1)
	spinChan <- fmt.Sprintf("Processing %d repos", relLength)

	// https://github.com/lotusirous/go-concurrency-patterns/blob/main/2-chan/main.go
	for _, rel := range releases {
		wg.Add(1)
		go goSyncRepo(ghClient, rel, c, &wg)
	}

	go func(c chan BinmanMsg, wg *sync.WaitGroup) {
		wg.Wait()
		close(c)
	}(c, &wg)

	// Process results
	for msg := range c {

		if msg.err == nil {
			output["Synced"] = append(output["Synced"], msg)
			continue
		}

		if msg.err.Error() == "Noupdate" {
			output["Up to Date"] = append(output["Up to Date"], msg)
			continue
		}

		output["Error"] = append(output["Error"], msg)
		if msg.rel.cleanupOnFailure {
			err := os.RemoveAll(msg.rel.publishPath)
			if err != nil {
				log.Debugf("Unable to clean up %s - %s", msg.rel.publishPath, err)
			}
			log.Debugf("cleaned %s\n", msg.rel.publishPath)
			log.Debugf("Final release data  %+v\n", msg.rel)
		}
	}

	swg.Add(1)
	spinChan <- fmt.Sprintf("spinstop%s", setStopMessage(output))
	close(spinChan)
	swg.Wait()

	if e := len(output["Error"]); e > 0 {
		fmt.Printf("\nErrors(%d): \n", e)
		for _, msg := range output["Error"] {
			fmt.Printf("%s : error = %v\n", msg.rel.Repo, msg.err)
		}
	}

	if table {
		OutputResults(output, debug)
	}

	log.Debugf("binman finished!")
}
