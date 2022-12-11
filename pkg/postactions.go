package binman

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/rjbrown57/binman/pkg/logging"
)

type DownloadAction struct {
	r *BinmanRelease
}

func (r *BinmanRelease) AddDownloadAction() Action {
	return &DownloadAction{
		r,
	}
}

// TODO move download logic to it's own function and call it here
func (action *DownloadAction) execute() error {

	log.Infof("Downloading %s", action.r.dlUrl)
	resp, err := http.Get(action.r.dlUrl)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	out, err := os.Create(filepath.Clean(action.r.filepath))
	if err != nil {
		return err
	}

	defer out.Close()

	_, err = io.Copy(io.MultiWriter(out), resp.Body)
	if err != nil {
		return err
	}

	log.Infof("Download %s complete", action.r.dlUrl)

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
	relNotes := action.r.githubData.GetBody()
	if relNotes != "" {
		notePath := filepath.Join(action.r.publishPath, "releaseNotes.txt")
		log.Debugf("Notes written to %s", notePath)
		return WriteStringtoFile(notePath, relNotes)
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
		err := handleTar(action.r.publishPath, action.r.filepath)
		if err != nil {
			log.Warnf("Failed to extract tar file: %v", err)
			return err
		}
	case "zip":
		log.Debugf("zip extract start")
		err := handleZip(action.r.publishPath, action.r.filepath)
		if err != nil {
			log.Warnf("Failed to extract zip file: %v", err)
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
	if _, err := os.Stat(action.r.artifactPath); errors.Is(err, os.ErrNotExist) {
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
		action.r.PostCommands[action.index].Args[i] = formatString(arg, dataMap)
	}

	log.Infof("Starting OS command %s with args %s for %s ", command, action.r.PostCommands[action.index].Args, action.r.Repo)

	out, err := exec.Command(command, action.r.PostCommands[action.index].Args...).Output()

	if err != nil {
		log.Warnf("error output for %s with args %s is %s", command, action.r.PostCommands[action.index].Args, out)
		return err
	}

	log.Infof("%s with args %s complete on %s", command, action.r.PostCommands[action.index].Args, action.r.Repo)
	log.Debugf("%s with args %s output: \n %s", command, action.r.PostCommands[action.index].Args, out)

	return nil

}
