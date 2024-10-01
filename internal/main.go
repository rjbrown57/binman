package internal

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/fatih/color"
	. "github.com/rjbrown57/binman/pkg"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/rodaine/table"
)

// TODO does this all just belong in the cmd pkg?

const timeout = 60 * time.Second

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

// Main does basic setup, then calls the appropriate functions for asset resolution and output
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

		switch msg.Err.(type) {
		case *NoUpdateError:
			output["Up to Date"] = append(output["Up to Date"], msg)
			continue
		case *ExcludeError:
			continue
		default:
			// Todo create an error here and use errors.Is
			output["Error"] = append(output["Error"], msg)
		}
	}

	bm.OutputOptions.SendSpin(fmt.Sprintf("spinstop%s", SetStopMessage(output)))

	// Wait and Close any channels used in the BMConfig
	bm.BMClose()

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

// Start watch command to expose metrics and sync on a schedule
func StartWatch(bm *BMConfig) {

	log.Debugf("watch config = %+v", bm.Config.Watch)

	bm.Config.Watch.LatestVersionMap = PopulateLatestMap(bm)

	// Start webserver
	go WatchServe(bm)

	go func() {
		for {

			bm.CollectData()

			// Process results
			for _, msg := range bm.Msgs {

				if msg.Err == nil {
					log.Infof("%s synced new release %s", msg.Rel.Repo, msg.Rel.Version)
					bm.Config.Watch.LatestVersionMap[msg.Rel.Repo] = msg.Rel
					continue
				}

				if msg.Err.Error() == "Noupdate" {
					log.Infof("%s - %s is up to date", msg.Rel.Repo, msg.Rel.Version)
					continue
				}

				log.Infof("Issue syncing %s - %s", msg.Rel.Repo, msg.Err)
			}

			log.Infof("Binman watch iteration complete")
			time.Sleep(time.Duration(bm.Config.Watch.Frequency) * time.Second)

			// Adding context here, this is reset, because if we initially added v0.0.0 of a repo
			// Then on a next sync we add v0.0.1 we would contain both in our metric output
			// Since the goal is to expose what is the latest of a repo this doesn't make sense
			// There is likely a better way to do this, where we don't reset, just update
			// I could do this by storing a metric per Rel, but then I would have lifespan issues I think
			bm.Metrics.Reset()
		}
	}()

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	// Block until a signal is received.
	sig := <-sigs
	log.Infof("Terimnating binman on %s", sig)
}
