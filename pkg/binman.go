package binman

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	db "github.com/rjbrown57/binman/pkg/db"
	log "github.com/rjbrown57/binman/pkg/logging"
)

const timeout = 60 * time.Second

var spinChan = make(chan string)
var swg sync.WaitGroup

// goSyncRepo executes all Actions required by a repo. Action are executed sequentially
// Actions are executed in 4 phases
// Pre -> Post -> Os -> Final
// The last action of each phase sets the actions for the next phase
// The Final actions is to set rel.actions = nil and conclude the loop
func goSyncRepo(rel BinmanRelease, c chan<- BinmanMsg, wg *sync.WaitGroup) {
	defer wg.Done()

	var err error

	rel.actions = rel.setPreActions(rel.ReleasePath, rel.BinPath)

	log.Debugf("release %s = %+v source = %+v", rel.Repo, rel, rel.source)

	for rel.actions != nil {
		if err = rel.runActions(); err != nil {
			switch err.Error() {
			case "Noupdate":
				c <- BinmanMsg{rel: rel, err: err}
				return
			default:
				c <- BinmanMsg{rel: rel, err: err}
				return
			}
		}
	}

	c <- BinmanMsg{rel: rel, err: nil}
}

func BinmanGetReleasePrep(sourceMap map[string]*Source, work map[string]string) []BinmanRelease {

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

	rel.setSource(sourceMap)
	rel.getOR()

	return []BinmanRelease{rel}

}

// Main does basic setup, then calls the appropriate functions for asset resolution
func Main(args map[string]string, table bool, launchCommand string) {

	log.Debugf("binman sync begin\n")

	c := make(chan BinmanMsg)
	output := make(map[string][]BinmanMsg)
	var wg sync.WaitGroup
	var releases []BinmanRelease

	var dwg sync.WaitGroup

	dbOptions := db.DbConfig{
		Dwg:    &dwg,
		DbChan: make(chan db.DbMsg),
	}

	if checkNewDb("") {
		log.Debugf("Initializing DB")
		populateDB(dbOptions, args["configFile"])
	}

	// Create config object.
	// setBaseConfig will return the appropriate base config file.
	// setConfig will check for a contextual config and merge with our base config and return the result
	config := SetConfig(SetBaseConfig(args["configFile"]), &dwg, dbOptions.DbChan)

	log.Debugf("binman config = %+v", config.Config)

	switch launchCommand {
	case "get":
		releases = BinmanGetReleasePrep(config.Config.SourceMap, args)
	case "config":
		releases = config.Releases
	}

	go getSpinner(log.IsDebug())
	go db.RunDB(dbOptions)

	// start download workers
	var numWorkers = config.Config.NumWorkers
	log.Debugf("launching %d download workers", numWorkers)
	for worker := 1; worker <= numWorkers; worker++ {
		go getDownloader(worker)
	}

	relLength := len(releases)
	log.Debugf("Process %v Releases", relLength)
	swg.Add(1)
	spinChan <- fmt.Sprintf("Processing %d repos", relLength)

	for _, rel := range releases {
		wg.Add(1)
		go goSyncRepo(rel, c, &wg)
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

	close(downloadChan)

	dwg.Wait()
	close(dbOptions.DbChan)

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
		OutputResults(output, log.IsDebug())
	}

	log.Debugf("binman finished!")
}
