package binman

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/rodaine/table"
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

func OutputResults(out map[string][]BinmanMsg, debug bool) {

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	upToDateTable := table.New("Repo", "Version", "State")
	upToDateTable.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for key, msgSlice := range out {
		for _, msg := range msgSlice {
			upToDateTable.AddRow(msg.rel.Repo, msg.rel.Version, key)
		}
	}

	upToDateTable.Print()
}

// Main does basic setup, then calls the appropriate functions for asset resolution
func Main(bm *BMConfig) error {

	log.Debugf("binman sync begin\n")

	log.Debugf("binman config = %+v", bm.Config)

	// Should we collapsed into a OutputOptions struct and added to BMConfig
	go getSpinner(log.IsDebug())

	// This should probably be moved to CollectData
	// This should be done when ouput is refactored
	relLength := len(bm.Releases)
	log.Debugf("Process %v Releases", relLength)
	swg.Add(1)
	spinChan <- fmt.Sprintf("Processing %d repos", relLength)

	// This will populate the bm.Releases array + return the list of msgs
	bm.CollectData()

	// Process results
	// This should be broken into it's own function?
	output := make(map[string][]BinmanMsg)

	for _, msg := range bm.Msgs {

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
			err := os.RemoveAll(msg.rel.PublishPath)
			if err != nil {
				log.Debugf("Unable to clean up %s - %s", msg.rel.PublishPath, err)
			}
			log.Debugf("cleaned %s\n", msg.rel.PublishPath)
			log.Debugf("Final release data  %+v\n", msg.rel)
		}
	}

	// Close any channels used in the BMConfig
	bm.BMClose()

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

	// We should update BM config to contain an optional "OutPutConfig" by default it is unset
	// it will contain the table, log level, jsonLog settings etc.
	// We should then be able to call an output interface to get our expected stuff?
	if true {
		OutputResults(output, log.IsDebug())
	}

	log.Debugf("binman finished!")
	return nil
}
