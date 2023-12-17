package binman

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rjbrown57/binman/pkg/constants"
	db "github.com/rjbrown57/binman/pkg/db"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/rjbrown57/binman/pkg/templating"
)

// BinmanRelease contains info on specifc releases to hunt for
type BinmanRelease struct {
	Os               string        `yaml:"os,omitempty"`
	Arch             string        `yaml:"arch,omitempty"`
	CheckSum         bool          `yaml:"checkSum,omitempty"`
	CleanupArchive   bool          `yaml:"cleanup,omitempty"`         // mark true if archive should be cleaned after extraction
	DownloadOnly     bool          `yaml:"downloadonly,omitempty"`    // Download but do not extract/find/link
	PostOnly         bool          `yaml:"postonly,omitempty"`        // Gather information from source, but perform no actions save os commands
	UpxConfig        UpxConfig     `yaml:"upx,omitempty"`             // Allow shrinking with Upx
	ExternalUrl      string        `yaml:"url,omitempty"`             // User provided external url to use with versions grabbed from GH. Note you must also set ReleaseFileName
	ExtractFileName  string        `yaml:"extractfilename,omitempty"` // The file within the release you want
	ReleaseFileName  string        `yaml:"releasefilename,omitempty"` // Specifc Release filename to look for. This is useful if a project publishes a binary and not a tarball.
	Repo             string        `yaml:"repo"`                      // The specific repo name in github. e.g achore/syft
	LinkName         string        `yaml:"linkname,omitempty"`        // Set what the final link will be. Defaults to project name.
	Version          string        `yaml:"version,omitempty"`         // Pull a specific version
	PostCommands     []PostCommand `yaml:"postcommands,omitempty"`
	QueryType        string        `yaml:"querytype,omitempty"`
	ReleasePath      string        `yaml:"releasepath,omitempty"`
	BinPath          string        `yaml:"binpath,omitempty"`
	SourceIdentifier string        `yaml:"source,omitempty"` // Allow setting of source individually

	createdAtTime    int64 // Unix time that release was created at
	metric           *prometheus.GaugeVec
	relData          interface{} // Data gathered from source
	relNotes         string
	source           *Source
	assetName        string // the target assetName
	cleanupOnFailure bool   // mark true if we need to clean up on failure
	dlUrl            string // the final donwload url
	filepath         string // the target filepath for download
	org              string // Will be provided by constuctor
	project          string // Will be provided by constuctor
	publishPath      string // Path Release will be set up at
	linkPath         string // Will be set by BinmanRelease.setPaths
	artifactPath     string // Will be set by BinmanRelease.setPaths. This is the source path for the link aka the executable binary
	actions          []Action
	versions         []string // Used during clean operations

	watchExposeMetrics bool
	watchSync          bool

	dwg    *sync.WaitGroup // Wait Group for db operations
	dbChan chan db.DbMsg   // Channel to send to DB
}

type PostCommand struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args,omitempty"`
}

// set project and org vars
func (r *BinmanRelease) getOR() {

	n := strings.Split(r.Repo, "/")
	length := len(n)

	// Concatenate everything but the project
	r.org = strings.Join(n[:length-1], "/")

	// Project is always the last element
	r.project = n[length-1]
}

// setSource will set the source for a release, it will also trim the source prefix from repo if used
func (r *BinmanRelease) setSource(sourceMap map[string]*Source) {

	repoSlice := strings.Split(r.Repo, "/")

	// Test if user supplied "sourceIdentifier/project/repo" format
	if source, exists := sourceMap[repoSlice[0]]; exists {
		// assign sourceIdentifer
		r.SourceIdentifier = source.Name

		// trimIdentifier from Reponame
		repoName := strings.TrimPrefix(r.Repo, repoSlice[0]+"/")
		r.Repo = repoName
		log.Debugf("source %s detected in repo name. Updating repo name to %s", repoSlice[0], r.Repo)

	}

	// github.com is default for an unspecified source
	if r.SourceIdentifier == "" {
		r.SourceIdentifier = "github.com"
	}

	// assign pointer to source info for this release
	r.source = sourceMap[r.SourceIdentifier]
}

func (r *BinmanRelease) findTarget() {

	targetFileName := templating.TemplateString((filepath.Base(r.artifactPath)), r.getDataMap())

	if r.Os == "windows" {
		targetFileName = targetFileName + ".exe"
		log.Debugf("Running on %s updating target to %s", r.Os, targetFileName)
	}

	tarRx := regexp.MustCompile(constants.TarRegEx)
	ZipRegEx := regexp.MustCompile(constants.ZipRegEx)

	_ = filepath.WalkDir(r.publishPath, func(path string, d os.DirEntry, err error) error {

		// if it's something we should ignore, we ignore it
		if d.IsDir() || tarRx.MatchString(d.Name()) || ZipRegEx.MatchString(d.Name()) {
			log.Debugf("Ignoring %s", path)
			return nil
		}

		log.Debugf("checking %s against %s for exact match", d.Name(), targetFileName)
		if targetFileName == d.Name() {
			log.Debugf("Found exact match! Using %s as the new artifact path.", path)

			r.artifactPath = path
			return fmt.Errorf("search complete")
		}

		// we short circuit at this point if the user is looking for an exact match only
		if r.ExtractFileName != "" {
			return nil
		}

		f, _ := d.Info()
		log.Debugf("checking %s perms %o are 755", d.Name(), f.Mode())
		if mode := f.Mode(); mode&os.ModePerm == 0755 {
			log.Debugf("Possible match found(executable file)! Setting %s as the new artifact path and continuing search for exact match.", path)
			r.artifactPath = path
		}

		return nil
	})

	// If we have selected a different asset internally than what is specified by r.LinkPath we need to update
	if filepath.Base(r.artifactPath) != r.LinkName && r.LinkName != r.Repo {
		// If the user has not specified a LinkName we should set a default here
		if r.LinkName == "" {
			r.LinkName = filepath.Base(r.artifactPath)
		}
		r.linkPath = fmt.Sprintf("%s/%s", filepath.Dir(r.linkPath), r.LinkName)
	}
}

// knownUrlCheck will see if binman is aware of a common external url for this repo.
func (r *BinmanRelease) knownUrlCheck() {
	if url, ok := constants.KnownUrlMap[r.Repo]; ok {
		log.Debugf("%s is a known repo. Updating download url to %s", r.Repo, url)
		r.ExternalUrl = url
	}
}

// Helper method to set artifactPath for a requested release object
// This will be called early in a main loop iteration so we can check if we already have a release
func (r *BinmanRelease) setPublishPath(ReleasePath string, tag string) {
	// Trim trailing / if user provided
	ReleasePath = strings.TrimSuffix(ReleasePath, "/")
	r.publishPath = filepath.Join(ReleasePath, "repos", r.SourceIdentifier, r.org, r.project, tag)
}

// getDataMap is a helper function to provide data to be used with templating
func (r *BinmanRelease) getDataMap() map[string]interface{} {
	dataMap := make(map[string]interface{})
	dataMap["version"] = r.Version
	dataMap["os"] = r.Os
	dataMap["arch"] = r.Arch
	dataMap["repo"] = r.Repo
	dataMap["org"] = r.org
	dataMap["project"] = r.project
	dataMap["artifactPath"] = r.artifactPath
	dataMap["publishPath"] = r.publishPath
	dataMap["linkPath"] = r.linkPath
	dataMap["assetName"] = r.assetName
	dataMap["createdAt"] = r.createdAtTime
	return dataMap
}

// Helper method to set paths for a requested release object
func (r *BinmanRelease) setArtifactPath(ReleasePath, BinPath string, assetName string) {

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
		r.artifactPath = filepath.Join(r.publishPath, templating.TemplateString(r.ReleaseFileName, r.getDataMap()))
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

	r.linkPath = filepath.Join(BinPath, linkName)
	log.Debugf("Artifact Path %s Link Path %s\n", r.artifactPath, r.project)

	r.filepath = fmt.Sprintf("%s/%s", r.publishPath, r.assetName)
}
