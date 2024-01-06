package binman

import (
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/theckman/yacspin"
)

func getSpinner(debug bool, spinChan chan (string), swg *sync.WaitGroup) {

	cfg := yacspin.Config{
		Frequency:       100 * time.Millisecond,
		CharSet:         yacspin.CharSets[52],
		Suffix:          "binman",
		SuffixAutoColon: true,
		Colors:          []string{"fgGreen"},
		StopColors:      []string{"fgGreen"},
	}

	spinner, err := yacspin.New(cfg)
	if err != nil {
		log.Debugf("Unable to get spinner - %s", err)
	}

	if !debug {
		swg.Add(1)
		spinner.Start()
	}

	for msg := range spinChan {

		if !strings.Contains(msg, "spinstop") {
			spinner.Message(msg)
		} else {
			spinner.StopMessage(strings.Trim(msg, "spinstop"))
		}
		swg.Done()
		time.Sleep(time.Millisecond * 500)

	}
	spinner.Suffix("")
	spinner.Stop()
	swg.Done()
}

func repoList(bmsg []BinmanMsg) []string {

	var a []string

	for _, msg := range bmsg {
		a = append(a, msg.Rel.Repo)
	}

	return a
}

// Set the stop message based on work completed
func setStopMessage(out map[string][]BinmanMsg) string {
	var stopMsg string

	// Get lengths
	syncedLength := len(out["Synced"])
	noUpdateLength := len(out["Up to Date"])
	errorLength := len(out["Error"])

	if noUpdateLength > 0 {
		stopMsg = stopMsg + fmt.Sprintf("✓ %d repos are up to date ", noUpdateLength)
	}

	if syncedLength > 0 {
		stopMsg = stopMsg + fmt.Sprintf("Δ %d repos %s pulled new versions ", syncedLength, repoList(out["Synced"]))
	}

	if errorLength > 0 {
		stopMsg = stopMsg + fmt.Sprintf("✕ %d repos errored %s during execution ", errorLength, repoList(out["Error"]))
	}

	return stopMsg
}
