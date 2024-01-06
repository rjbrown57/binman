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
				c <- BinmanMsg{Rel: rel, Err: err}
				return
			default:
				c <- BinmanMsg{Rel: rel, Err: err}
				return
			}
		}
	}

	c <- BinmanMsg{Rel: rel, Err: nil}
}

func OutputResults(out map[string][]BinmanMsg, debug bool) {

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	upToDateTable := table.New("Repo", "Version", "State")
	upToDateTable.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for key, msgSlice := range out {
		for _, msg := range msgSlice {
			upToDateTable.AddRow(msg.Rel.Repo, msg.Rel.Version, key)
		}
	}

	upToDateTable.Print()
}

// Main does basic setup, then calls the appropriate functions for asset resolution
func Main(bm *BMConfig) error {

	log.Debugf("binman sync begin\n")

	log.Debugf("binman config = %+v", bm.Config)

	// This should probably be moved to CollectData
	// This should be done when output is refactored
	relLength := len(bm.Releases)
	log.Debugf("Process %v Releases", relLength)

	bm.OutputOptions.SendSpin(fmt.Sprintf("Processing %d repos", relLength))

	// This will populate the bm.Releases array + return the list of msgs
	bm.CollectData()

	// Process results
	// This should be broken into it's own function?
	output := make(map[string][]BinmanMsg)

	for _, msg := range bm.Msgs {

		if msg.Err == nil {
			output["Synced"] = append(output["Synced"], msg)
			continue
		}

		if msg.Err.Error() == "Noupdate" {
			output["Up to Date"] = append(output["Up to Date"], msg)
			continue
		}

		output["Error"] = append(output["Error"], msg)
		if msg.Rel.cleanupOnFailure {
			err := os.RemoveAll(msg.Rel.PublishPath)
			if err != nil {
				log.Debugf("Unable to clean up %s - %s", msg.Rel.PublishPath, err)
			}
			log.Debugf("cleaned %s\n", msg.Rel.PublishPath)
			log.Debugf("Final release data  %+v\n", msg.Rel)
		}
	}

	// Close any channels used in the BMConfig
	bm.BMClose()

	bm.OutputOptions.SendSpin(fmt.Sprintf("spinstop%s", setStopMessage(output)))
	close(bm.OutputOptions.spinChan)
	bm.OutputOptions.swg.Wait()

	if e := len(output["Error"]); e > 0 {
		fmt.Printf("\nErrors(%d): \n", e)
		for _, msg := range output["Error"] {
			fmt.Printf("%s : error = %v\n", msg.Rel.Repo, msg.Err)
		}
	}

	if bm.OutputOptions.Table {
		OutputResults(output, log.IsDebug())
	}

	log.Debugf("binman finished!")
	return nil
}
