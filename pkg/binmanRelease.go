package binman

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v48/github"
	log "github.com/rjbrown57/binman/pkg/logging"
)

// BinmanRelease contains info on specifc releases to hunt for
type BinmanRelease struct {
	Os              string        `yaml:"os,omitempty"`
	Arch            string        `yaml:"arch,omitempty"`
	CheckSum        bool          `yaml:"checkSum,omitempty"`
	DownloadOnly    bool          `yaml:"downloadonly,omitempty"`
	UpxConfig       UpxConfig     `yaml:"upx,omitempty"`             // Allow shrinking with Upx
	ExternalUrl     string        `yaml:"url,omitempty"`             // User provided external url to use with versions grabbed from GH. Note you must also set ReleaseFileName
	ExtractFileName string        `yaml:"extractfilename,omitempty"` // The file within the release you want
	ReleaseFileName string        `yaml:"releasefilename,omitempty"` // Specifc Release filename to look for. This is useful if a project publishes a binary and not a tarball.
	Repo            string        `yaml:"repo"`                      // The specific repo name in github. e.g achore/syft
	LinkName        string        `yaml:"linkname,omitempty"`        // Set what the final link will be. Defaults to project name.
	Version         string        `yaml:"version,omitempty"`         // Pull a specific version
	PostCommands    []PostCommand `yaml:"postcommands,omitempty"`

	githubData   *github.RepositoryRelease
	assetName    string   // the target assetName
	dlUrl        string   // the final donwload url
	filepath     string   // the target filepath for download
	org          string   // Will be provided by constuctor
	project      string   // Will be provided by constuctor
	publishPath  string   // Path Release will be set up at
	linkPath     string   // Will be set by BinmanRelease.setPaths
	artifactPath string   // Will be set by BinmanRelease.setPaths. This is the source path for the link aka the executable binary
	tasks        []Action // the actions we will perform for this release
}

type PostCommand struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args,omitempty"`
}

// getPostStepTasks will arrange all final work after we have selected an asset
func (r *BinmanRelease) getPostStepTasks() {

	// We will always download
	r.tasks = append(r.tasks, r.AddDownloadAction())

	// If we are not set to download only, set the rest of the post processing actions
	if !r.DownloadOnly {
		switch findfType(r.filepath) {
		case "tar":
			r.tasks = append(r.tasks, r.AddExtractAction())
		case "zip":
			r.tasks = append(r.tasks, r.AddExtractAction())
		case "default":
		}

		r.tasks = append(r.tasks, r.AddFindTargetAction(),
			r.AddMakeExecuteableAction(),
			r.AddLinkFileAction(),
			r.AddWriteRelNotesAction())

		// Upx needs to be prepended to PostCommands if user has requested
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

	// Add post commands defined by user if specified
	for index := range r.PostCommands {
		r.tasks = append(r.tasks, r.AddOsCommandAction(index))
	}

}

// set project and org vars
func (r *BinmanRelease) getOR() {
	n := strings.Split(r.Repo, "/")
	r.org = n[0]
	r.project = n[1]
}

func (r *BinmanRelease) findTarget() {

	targetFileName := formatString(filepath.Base(r.artifactPath), r.getDataMap())

	_ = filepath.Walk(r.publishPath, func(path string, info os.FileInfo, err error) error {
		log.Debugf("Checking %s, against %s...", targetFileName, info.Name())
		if err == nil && targetFileName == info.Name() {
			log.Debugf("Found match! Using %s as the new artifact path.", path)
			r.artifactPath = path
		}
		return nil
	})
}

// knownUrlCheck will see if binman is aware of a common external url for this repo.
func (r *BinmanRelease) knownUrlCheck() {
	if url, ok := KnownUrlMap[r.Repo]; ok {
		log.Debugf("%s is a known repo. Updating download url to %s", r.Repo, url)
		r.ExternalUrl = url
	}
}

// Helper method to set artifactPath for a requested release object
// This will be called early in a main loop iteration so we can check if we already have a release
func (r *BinmanRelease) setPublisPath(ReleasePath string, tag string) {
	// Trim trailing / if user provided
	ReleasePath = strings.TrimSuffix(ReleasePath, "/")
	r.publishPath = filepath.Join(ReleasePath, "repos", r.org, r.project, tag)
}

// getDataMap is a helper function to provide data to be used with templating
func (r *BinmanRelease) getDataMap() map[string]string {
	dataMap := make(map[string]string)
	dataMap["version"] = *r.githubData.TagName
	dataMap["os"] = r.Os
	dataMap["arch"] = r.Arch
	dataMap["org"] = r.org
	dataMap["project"] = r.project
	dataMap["artifactPath"] = r.artifactPath
	dataMap["linkpath"] = r.linkPath
	dataMap["filename"] = r.assetName
	return dataMap
}

// Helper method to set paths for a requested release object
func (r *BinmanRelease) setArtifactPath(ReleasePath string, assetName string) {

	// Allow user to supply the name of the final link
	// This is nice for projects like lazygit which is simply too much to type
	// linkname: lg would have lazygit point at lg :)
	var linkName string
	if r.LinkName == "" {
		linkName = r.project
	} else {
		linkName = r.LinkName
	}

	// If a binary is specified by ReleaseFileName use it for source and project for destination
	// else if it's a tar/zip but we have specified the inside file via ExtractFileName. Use ExtractFileName for source and destination
	// else we want default
	if r.ReleaseFileName != "" {
		r.artifactPath = filepath.Join(r.publishPath, r.ReleaseFileName)
		log.Debugf("ReleaseFileName set %s\n", r.artifactPath)
	} else if r.ExtractFileName != "" {
		r.artifactPath = filepath.Join(r.publishPath, r.ExtractFileName)
		log.Debugf("Archive with Filename set %s\n", r.artifactPath)
	} else if r.ExternalUrl != "" {
		switch findfType(assetName) {
		case "tar", "zip":
			r.artifactPath = filepath.Join(r.publishPath, r.project)
		default:
			r.artifactPath = filepath.Join(r.publishPath, filepath.Base(r.ExternalUrl))
		}
		log.Debugf("Archive with ExternalURL set %s\n", r.artifactPath)
	} else {
		// If we find a tar/zip in the assetName assume the name of the binary within the tar
		// Else our default is a binary
		switch findfType(assetName) {
		case "tar", "zip":
			r.artifactPath = filepath.Join(r.publishPath, r.project)
		default:
			r.artifactPath = filepath.Join(r.publishPath, assetName)
		}
		log.Debugf("Default Extraction %s\n", r.artifactPath)
	}

	r.linkPath = filepath.Join(ReleasePath, linkName)
	log.Debugf("Artifact Path %s Link Path %s\n", r.artifactPath, r.project)

	r.filepath = fmt.Sprintf("%s/%s", r.publishPath, r.assetName)
}
