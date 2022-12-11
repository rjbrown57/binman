package binman

import "github.com/google/go-github/v48/github"

/*
All binman work is done by implementations of the Action interface. Work is ordered depending on user request and then executed sequentially.
The work is divided into several stages
get
  * Collect data to act on. Currently this is only github releases.
pre
  * Preperation and validation there is actually work to do
post
  * steps to perform after an asset has been downloaded
*/

type Action interface {
	execute() error
}

// SetPreActions handles query and asset Selection
func (r *BinmanRelease) setPreActions(ghClient *github.Client, releasePath string) []Action {

	var actions []Action

	// Add query task
	actions = append(actions, r.AddGetGHReleaseAction(ghClient))

	// If publishPath is already set we are doing a direct repo download and don't need to set a release path
	// Direct repo actions should be moved to their own command
	if r.publishPath == "" {
		actions = append(actions, r.AddReleaseStatusAction(releasePath))
	}

	// Add remaining preDownload actions
	actions = append(actions,
		r.AddSetUrlAction(),
		r.AddSetArtifactPathAction(releasePath),
	)

	return actions

}

// Set common PostCommands. Currently this is only upx
func (r *BinmanRelease) setCommonPostCommands() {
	if r.UpxConfig.Enabled == "true" {
		// Merge any user args with upx
		args := []string{r.artifactPath}
		args = append(args, r.UpxConfig.Args...)

		UpxCommand := PostCommand{
			Command: "upx",
			Args:    args,
		}

		r.PostCommands = append([]PostCommand{UpxCommand}, r.PostCommands...)
	}
}

// getPostActions will arrange all final work after we have selected an asset
func (r *BinmanRelease) setPostActions() []Action {

	var actions []Action

	// We will always download
	actions = append(actions, r.AddDownloadAction())

	// If we are set to download only stop all postCommands
	if r.DownloadOnly {
		return actions
	}

	// If we are not set to download only, set the rest of the post processing actions
	switch findfType(r.filepath) {
	case "tar":
		actions = append(actions, r.AddExtractAction())
	case "zip":
		actions = append(actions, r.AddExtractAction())
	case "default":
	}

	actions = append(actions, r.AddFindTargetAction(),
		r.AddMakeExecuteableAction(),
		r.AddLinkFileAction(),
		r.AddWriteRelNotesAction())

	// Common PostCommands user has requested. Currently UPX
	r.setCommonPostCommands()

	// Add post commands defined by user if specified
	for index := range r.PostCommands {
		actions = append(actions, r.AddOsCommandAction(index))
	}

	return actions

}
