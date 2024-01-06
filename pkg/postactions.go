package binman

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/rjbrown57/binman/pkg/downloader"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/rjbrown57/binman/pkg/templating"
)

type DownloadAction struct {
	r *BinmanRelease
}

func (r *BinmanRelease) AddDownloadAction() Action {
	return &DownloadAction{
		r,
	}
}

func (action *DownloadAction) execute() error {
	// Created a buffered channel since we will not run a recieving goroutine
	// size will always be 1
	confirmChan := make(chan error, 1)
	var rWg sync.WaitGroup

	rWg.Add(1)
	action.r.downloadChan <- downloader.DlMsg{Url: action.r.dlUrl, Filepath: action.r.filepath, Wg: &rWg, ConfirmChan: confirmChan}
	action.r.output.SendSpin(fmt.Sprintf("Downloading %s(%s)", action.r.Repo, action.r.Version))
	rWg.Wait()
	close(confirmChan)

	err := <-confirmChan

	if err != nil {
		action.r.output.SendSpin(fmt.Sprintf("Error Downloading %s(%s)", action.r.Repo, action.r.Version))
		return err
	}

	action.r.output.SendSpin(fmt.Sprintf("Download of %s(%s) finished", action.r.Repo, action.r.Version))

	return nil
}

// link action

type LinkFileAction struct {
	r *BinmanRelease
}

func (r *BinmanRelease) AddLinkFileAction() Action {
	return &LinkFileAction{
		r,
	}
}

func (action *LinkFileAction) execute() error {
	return createLink(action.r.artifactPath, action.r.linkPath)
}

// Remove downloaded archive after extraction
type CleanArchiveAction struct {
	r *BinmanRelease
}

func (r *BinmanRelease) AddCleanArchive() Action {
	return &CleanArchiveAction{
		r,
	}
}

func (action *CleanArchiveAction) execute() error {
	log.Debugf("cleaning up %s", action.r.filepath)
	return os.Remove(action.r.filepath)
}

type MakeExecuteableAction struct {
	r *BinmanRelease
}

func (r *BinmanRelease) AddMakeExecuteableAction() Action {
	return &MakeExecuteableAction{
		r,
	}
}

func (action *MakeExecuteableAction) execute() error {
	return MakeExecuteable(action.r.artifactPath)
}

// WriteReleaseNotes
type WriteRelNotesAction struct {
	r *BinmanRelease
}

func (r *BinmanRelease) AddWriteRelNotesAction() Action {
	return &WriteRelNotesAction{
		r,
	}
}

func (action *WriteRelNotesAction) execute() error {
	if action.r.relNotes != "" {
		notePath := filepath.Join(action.r.PublishPath, "releaseNotes.txt")
		log.Debugf("Notes written to %s", notePath)
		return WriteStringtoFile(notePath, action.r.relNotes)
	}

	return nil
}

// Extract
type ExtractAction struct {
	r *BinmanRelease
}

func (r *BinmanRelease) AddExtractAction() Action {
	return &ExtractAction{
		r,
	}
}

func (action *ExtractAction) execute() error {
	switch findfType(action.r.filepath) {
	case "tar":
		log.Debugf("tar extract start")
		err := handleTar(action.r.PublishPath, action.r.filepath)
		if err != nil {
			log.Debugf("Failed to extract tar file: %v", err)
			return err
		}
	case "zip":
		log.Debugf("zip extract start")
		err := handleZip(action.r.PublishPath, action.r.filepath)
		if err != nil {
			log.Debugf("Failed to extract zip file: %v", err)
			return err
		}
	}

	return nil

}

// FindTarget
type FindTargetAction struct {
	r *BinmanRelease
}

func (r *BinmanRelease) AddFindTargetAction() Action {
	return &FindTargetAction{
		r,
	}
}

func (action *FindTargetAction) execute() error {
	// If the file still doesn't exist, attempt to find it in sub-directories

	if f, err := os.Stat(action.r.artifactPath); errors.Is(err, os.ErrNotExist) || f.IsDir() {
		log.Debugf("Wasn't able to find the artifact at %s, walking the directory to see if we can find it",
			action.r.artifactPath)

		// Walk the directory looking for the file. If found artifact path will be updated
		action.r.findTarget()

		if _, err := os.Stat(action.r.artifactPath); errors.Is(err, os.ErrNotExist) {
			err := fmt.Errorf("unable to find a matching file for %s anywhere in the release archive", action.r.Repo)
			return err
		}
	}

	return nil

}

type OsCommandAction struct {
	r     *BinmanRelease
	index int
}

func (r *BinmanRelease) AddOsCommandAction(index int) Action {
	return &OsCommandAction{
		r,
		index,
	}
}

func (action *OsCommandAction) execute() error {

	command := action.r.PostCommands[action.index].Command

	dataMap := action.r.getDataMap()

	// Template any args
	for i, arg := range action.r.PostCommands[action.index].Args {
		action.r.PostCommands[action.index].Args[i] = templating.TemplateString(arg, dataMap)
	}

	log.Debugf("Starting OS command %s with args %s for %s ", command, action.r.PostCommands[action.index].Args, action.r.Repo)

	out, err := exec.Command(command, action.r.PostCommands[action.index].Args...).Output()

	if err != nil {
		log.Debugf("error output for %s with args %s is %s", command, action.r.PostCommands[action.index].Args, out)
		return err
	}

	log.Debugf("%s with args %s output: \n %s", command, action.r.PostCommands[action.index].Args, out)

	return nil

}
