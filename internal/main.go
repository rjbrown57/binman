package internal

import (
	"fmt"
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

		// Todo create an error here and use errors.Is
		if msg.Err.Error() == "Noupdate" {
			output["Up to Date"] = append(output["Up to Date"], msg)
			continue
		}
		output["Error"] = append(output["Error"], msg)
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
