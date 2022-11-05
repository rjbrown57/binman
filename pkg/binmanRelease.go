package binman

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v48/github"
)

// BinmanRelease contains info on specifc releases to hunt for
type BinmanRelease struct {
	Os              string    `yaml:"os,omitempty"`
	Arch            string    `yaml:"arch,omitempty"`
	CheckSum        bool      `yaml:"checkSum,omitempty"`
	DownloadOnly    bool      `yaml:"downloadonly,omitempty"`
	UpxConfig       UpxConfig `yaml:"upx,omitempty"`             // Allow shrinking with Upx
	ExternalUrl     string    `yaml:"url,omitempty"`             // User provided external url to use with versions grabbed from GH. Note you must also set ReleaseFileName
	ExtractFileName string    `yaml:"extractfilename,omitempty"` // The file within the release you want
	ReleaseFileName string    `yaml:"releasefilename,omitempty"` // Specifc Release filename to look for. This is useful if a project publishes a binary and not a tarball.
	Repo            string    `yaml:"repo"`                      // The specific repo name in github. e.g achore/syft
	Org             string    // Will be provided by constuctor
	Project         string    // Will be provided by constuctor
	PublishPath     string    // Path Release will be set up at
	ArtifactPath    string    // Will be set by BinmanRelease.setPaths. This is the source path for the link aka the executable binary
	LinkName        string    `yaml:"linkname,omitempty"` // Set what the final link will be. Defaults to project name.
	LinkPath        string    // Will be set by BinmanRelease.setPaths
	Version         string    `yaml:"version,omitempty"` // Pull a specific version
	GithubData      *github.RepositoryRelease
}

// set project and org vars
func (r *BinmanRelease) getOR() {
	n := strings.Split(r.Repo, "/")
	r.Org = n[0]
	r.Project = n[1]
}

func (r *BinmanRelease) findTarget() {

	targetFileName := formatString(filepath.Base(r.ArtifactPath), r.getDataMap())

	_ = filepath.Walk(r.PublishPath, func(path string, info os.FileInfo, err error) error {
		log.Debugf("Checking %s, against %s...", targetFileName, info.Name())
		if err == nil && targetFileName == info.Name() {
			log.Debugf("Found match! Using %s as the new artifact path.", path)
			r.ArtifactPath = path
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

// Helper method to set artifactpath for a requested release object
// This will be called early in a main loop iteration so we can check if we already have a release
func (r *BinmanRelease) setPublisPath(ReleasePath string, tag string) {
	// Trim trailing / if user provided
	ReleasePath = strings.TrimSuffix(ReleasePath, "/")
	r.PublishPath = filepath.Join(ReleasePath, "repos", r.Org, r.Project, tag)
}

// getDataMap is a helper function to provide data to be used with templating
func (r *BinmanRelease) getDataMap() map[string]string {
	dataMap := make(map[string]string)
	dataMap["version"] = *r.GithubData.TagName
	dataMap["os"] = r.Os
	dataMap["arch"] = r.Arch
	return dataMap
}

// Helper method to set paths for a requested release object
func (r *BinmanRelease) setArtifactPath(ReleasePath string, assetName string) {

	// Allow user to supply the name of the final link
	// This is nice for projects like lazygit which is simply too much to type
	// linkname: lg would have lazygit point at lg :)
	var linkName string
	if r.LinkName == "" {
		linkName = r.Project
	} else {
		linkName = r.LinkName
	}

	// If a binary is specified by ReleaseFileName use it for source and project for destination
	// else if it's a tar/zip but we have specified the inside file via ExtractFileName. Use ExtractFileName for source and destination
	// else we want default
	if r.ReleaseFileName != "" {
		r.ArtifactPath = filepath.Join(r.PublishPath, r.ReleaseFileName)
		log.Debugf("ReleaseFileName set %s\n", r.ArtifactPath)
	} else if r.ExtractFileName != "" {
		r.ArtifactPath = filepath.Join(r.PublishPath, r.ExtractFileName)
		log.Debugf("Archive with Filename set %s\n", r.ArtifactPath)
	} else if r.ExternalUrl != "" {
		switch findfType(assetName) {
		case "tar", "zip":
			r.ArtifactPath = filepath.Join(r.PublishPath, r.Project)
		default:
			r.ArtifactPath = filepath.Join(r.PublishPath, filepath.Base(r.ExternalUrl))
		}
		log.Debugf("Archive with ExternalURL set %s\n", r.ArtifactPath)
	} else {
		// If we find a tar/zip in the assetName assume the name of the binary within the tar
		// Else our default is a binary
		switch findfType(assetName) {
		case "tar", "zip":
			r.ArtifactPath = filepath.Join(r.PublishPath, r.Project)
		default:
			r.ArtifactPath = filepath.Join(r.PublishPath, assetName)
		}
		log.Debugf("Default Extraction %s\n", r.ArtifactPath)
	}

	r.LinkPath = filepath.Join(ReleasePath, linkName)
	log.Debugf("Artifact Path %s Link Path %s\n", r.ArtifactPath, r.Project)
}

func (r *BinmanRelease) writeReleaseNotes() error {
	relNotes := r.GithubData.GetBody()
	if relNotes != "" {
		notePath := filepath.Join(r.PublishPath, "releaseNotes.txt")
		log.Debugf("Notes written to %s", notePath)
		return writeStringtoFile(notePath, relNotes)
	}

	return nil
}
