package binman

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rjbrown57/binman/pkg/constants"
	db "github.com/rjbrown57/binman/pkg/db"
	"github.com/rjbrown57/binman/pkg/downloader"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/rjbrown57/binman/pkg/templating"
)

var (
	ErrNoVersionsFound = errors.New("No Versions detected for release")
)

type NoUpdateError struct {
	RepoName string
	Version  string
}

func (e *NoUpdateError) Error() string {
	return fmt.Sprintf("%s is already at %s version, nothing to do here", e.RepoName, e.Version)
}

type ExcludeError struct {
	RepoName string
	Criteria string
}

func (e *ExcludeError) Error() string {
	return fmt.Sprintf("%s was excluded from consideration because: %s", e.RepoName, e.Criteria)
}

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
	SourceIdentifier string        `yaml:"source,omitempty"`      // Allow setting of source individually
	PublishPath      string        `yaml:"publishpath,omitempty"` // Path Release will be set up at. Typically only set by set commands or library use.
	ArtifactPath     string        `yaml:"-"`                     // Will be set by BinmanRelease.setPaths. This is the source path for the link aka the executable binary
	ExcludeOs        []string      `yaml:"excludeos,omitempty"`   // Allows excluding certain OS's because we know that we'll never have releases for this OS

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
	linkPath         string // Will be set by BinmanRelease.setPaths
	actions          []Action
	versions         []string // Used during clean operations
	output           *OutputOptions

	watchExposeMetrics bool
	watchSync          bool

	dwg          *sync.WaitGroup       // Wait Group for db operations
	dbChan       chan db.DbMsg         // Channel to send to DB
	downloadChan chan downloader.DlMsg // Channel to request file download
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

// SetSource will set the source for a release, it will also trim the source prefix from repo if used
func (r *BinmanRelease) SetSource(sourceMap map[string]*Source) {

	var sourceId string = "github.com"

	repoSlice := strings.Split(r.Repo, "/")

	// Test if user supplied "sourceIdentifier/project/repo" format
	if source, exists := sourceMap[repoSlice[0]]; exists {

		// assign sourceIdentifer only if type is not binman
		// Since binman source relies on knowing the upstream source
		if source.Apitype != "binman" {
			sourceId = source.Name
		}

		// trimIdentifier from Reponame
		repoName := strings.TrimPrefix(r.Repo, repoSlice[0]+"/")
		r.Repo = repoName
		log.Debugf("source %s detected in repo name. Updating repo name to %s", repoSlice[0], r.Repo)
	}

	switch r.SourceIdentifier {
	// If the SourceIdentifier is set to binman then we need to know the sourceID by repo name since binman is a "downstream" source
	case "binman":
		r.source = sourceMap[r.SourceIdentifier]
		r.SourceIdentifier = sourceId
	case "":
		r.SourceIdentifier = sourceId
		fallthrough
	default:
		r.source = sourceMap[r.SourceIdentifier]
	}

	// If default is set to binman then everything will use binman queries
	if sourceMap["default"].Apitype == "binman" {
		r.source = sourceMap["default"]
	}
}

func (r *BinmanRelease) findTarget() {

	targetFileName := templating.TemplateString((filepath.Base(r.ArtifactPath)), r.getDataMap())

	if r.Os == "windows" {
		targetFileName = targetFileName + ".exe"
		log.Debugf("Running on %s updating target to %s", r.Os, targetFileName)
	}

	tarRx := regexp.MustCompile(constants.TarRegEx)
	ZipRegEx := regexp.MustCompile(constants.ZipRegEx)

	_ = filepath.WalkDir(r.PublishPath, func(path string, d os.DirEntry, err error) error {

		// if it's something we should ignore, we ignore it
		if d.IsDir() || tarRx.MatchString(d.Name()) || ZipRegEx.MatchString(d.Name()) {
			log.Debugf("Ignoring %s", path)
			return nil
		}

		log.Debugf("checking %s against %s for exact match", d.Name(), targetFileName)
		if targetFileName == d.Name() {
			log.Debugf("Found exact match! Using %s as the new artifact path.", path)

			r.ArtifactPath = path
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
			r.ArtifactPath = path
		}

		return nil
	})

	// If we have selected a different asset internally than what is specified by r.LinkPath we need to update
	if filepath.Base(r.ArtifactPath) != r.LinkName && r.LinkName != r.Repo {
		// If the user has not specified a LinkName we should set a default here
		if r.LinkName == "" {
			r.LinkName = filepath.Base(r.ArtifactPath)
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
func (r *BinmanRelease) setpublishPath(ReleasePath string, tag string) {
	// Trim trailing / if user provided
	ReleasePath = strings.TrimSuffix(ReleasePath, "/")
	r.PublishPath = filepath.Join(ReleasePath, "repos", r.SourceIdentifier, r.org, r.project, tag)
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
	dataMap["artifactPath"] = r.ArtifactPath
	dataMap["publishPath"] = r.PublishPath
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
		r.ArtifactPath = filepath.Join(r.PublishPath, templating.TemplateString(r.ReleaseFileName, r.getDataMap()))
		log.Debugf("ReleaseFileName set %s\n", r.ArtifactPath)
	} else if r.ExtractFileName != "" {
		r.ArtifactPath = filepath.Join(r.PublishPath, r.ExtractFileName)
		log.Debugf("Archive with Filename set %s\n", r.ArtifactPath)
	} else if r.ExternalUrl != "" {
		switch findfType(assetName) {
		case "tar", "zip":
			r.ArtifactPath = filepath.Join(r.PublishPath, r.project)
		default:
			r.ArtifactPath = filepath.Join(r.PublishPath, filepath.Base(r.ExternalUrl))
		}
		log.Debugf("Archive with ExternalURL set %s\n", r.ArtifactPath)
	} else {
		// If we find a tar/zip in the assetName assume the name of the binary within the tar
		// Else our default is a binary
		switch findfType(assetName) {
		case "tar", "zip":
			r.ArtifactPath = filepath.Join(r.PublishPath, r.project)
		default:
			r.ArtifactPath = filepath.Join(r.PublishPath, assetName)
		}
		log.Debugf("Default Extraction %s\n", r.ArtifactPath)
	}

	r.linkPath = filepath.Join(BinPath, linkName)
	log.Debugf("Artifact Path %s Link Path %s\n", r.ArtifactPath, r.project)

	r.filepath = fmt.Sprintf("%s/%s", r.PublishPath, r.assetName)
}

// FetchReleaseData will query the DB
func (r *BinmanRelease) FetchReleaseData(versions ...string) error {

	var err error = nil
	// If no versions supplied populate from filesystem
	if len(versions) == 0 {
		repoPath := fmt.Sprintf("%s/repos/%s/%s", r.ReleasePath, r.SourceIdentifier, r.Repo)
		log.Debugf("Scanning %s", repoPath)

		versions, err = sortSemvers(GetVersionFromPath(repoPath))
		if err != nil {
			log.Warnf("Error sorting %s %s", r.Repo, err)
			return err
		}

		if len(versions) == 0 {
			return ErrNoVersionsFound
		}
	}

	r.Version = versions[len(versions)-1]

	r.dwg.Add(1)

	var rwg sync.WaitGroup

	dbMsg := db.DbMsg{
		Operation:  "read",
		Key:        fmt.Sprintf("%s/%s/%s/data", r.SourceIdentifier, r.Repo, r.Version),
		ReturnChan: make(chan db.DBResponse, 1),
		ReturnWg:   &rwg,
	}

	d := dbMsg.Send(r.dbChan)

	if d.Err != nil {
		log.Warnf("failed reading %s from db %s %s", r.Repo, r.Version, d.Err)
		return d.Err
	}

	m := bytesToData(d.Data)

	r.ArtifactPath = m["artifactPath"].(string)
	r.Arch = m["arch"].(string)

	return err
}

func (r *BinmanRelease) displayActions(actions *[]Action) []string {

	var currentActions []string

	for _, action := range *actions {
		currentActions = append(currentActions, reflect.TypeOf(action).String())
	}

	return currentActions
}
