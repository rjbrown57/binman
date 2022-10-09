package binman

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/go-github/v44/github"
	"gopkg.in/yaml.v2"
)

const TarRegEx = `(\.tar$|\.tar\.gz$|\.tgz$)`
const ZipRegEx = `(\.zip$)`
const x86RegEx = `(amd64|x86_64)`

// BinmanMsg contains return messages for binman's concurrent workers
type BinmanMsg struct {
	err error
	rel BinmanRelease
}

type UpxConfig struct {
	Enabled string   `yaml:"enabled"` // Using a string here instead of a boolean to deal with an unset boolean defaulting to false
	Args    []string `yaml:"args,omitempty"`
}

// BinmanConfig contains Global Config Options
type BinmanConfig struct {
	ReleasePath string    `yaml:"releasepath"`        //path to download/link releases from github
	TokenVar    string    `yaml:"tokenvar,omitempty"` //Github Auth Token
	UpxConfig   UpxConfig `yaml:"upx,omitempty"`      // Allow upx to shrink extracted
}

// BinmanDefaults contains default config options. If a value is unset in releases array these will be used.
// This should just be collapsed into BinmanConfig and this struct should be removed
type BinmanDefaults struct {
	Os   string `yaml:"os,omitempty"`   //OS architechrue to look for
	Arch string `yaml:"arch,omitempty"` //OS architechrue to look for
}

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
	ArtifactPath    string    // Will be set by BinmanRelease.setPaths. This is the source path for the link
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

// Helper method to set artifactpath for a requested release object
// This will be called early in a main loop iteration so we can check if we already have a release
func (r *BinmanRelease) setArtifactPath(ReleasePath string, tag string) {
	// Trim trailing / if user provided
	if strings.HasSuffix(ReleasePath, "/") {
		ReleasePath = strings.TrimSuffix(ReleasePath, "/")
	}
	r.PublishPath = filepath.Join(ReleasePath, "repos", r.Org, r.Project, tag)
}

// Helper method to set paths for a requested release object
func (r *BinmanRelease) setPublishPaths(ReleasePath string, assetName string) {

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
	// else if it's a tar/zip but we have specified the inside file via ExtractFileName. Use ExtraceFileName for source and destination
	// else we want default
	if r.ReleaseFileName != "" {
		r.ArtifactPath = filepath.Join(r.PublishPath, r.ReleaseFileName)
		log.Debugf("ReleaseFileName set %s\n", r.ArtifactPath)
	} else if r.ExtractFileName != "" {
		r.ArtifactPath = filepath.Join(r.PublishPath, r.ExtractFileName)
		log.Debugf("Archive with Filename set %s\n", r.ArtifactPath)
	} else if r.ExternalUrl != "" {
		r.ArtifactPath = filepath.Join(r.PublishPath, filepath.Base(r.ExternalUrl))
		log.Debugf("Archive with ExternalURL set %s\n", r.ArtifactPath)
	} else {
		// If we find a tar/zip in the assetName assume the name of the binary within the tar
		// Else our default is a binary
		switch findfType(assetName) {
		case "tar":
			r.ArtifactPath = filepath.Join(r.PublishPath, r.Project)
		case "zip":
			r.ArtifactPath = filepath.Join(r.PublishPath, r.Project)
		default:
			r.ArtifactPath = filepath.Join(r.PublishPath, assetName)
		}
		log.Debugf("Default Extraction %s\n", r.ArtifactPath)
	}

	r.LinkPath = filepath.Join(ReleasePath, linkName)
	log.Debugf("Artifact Path %s Link Path %s\n", r.ArtifactPath, r.Project)

}

// Type that rolls up the above types into one happy family
type GHBMConfig struct {
	Config   BinmanConfig    `yaml:"config"`
	Defaults BinmanDefaults  `yaml:"defaults"`
	Releases []BinmanRelease `yaml:"releases"`
}

func newGHBMConfig(configPath string) *GHBMConfig {
	config := &GHBMConfig{}
	mustUnmarshalYaml(configPath, config)
	config.setDefaults()
	return config
}

// setDefaults will populate defaults, and required values
func (config *GHBMConfig) setDefaults() {

	// If user does not supply a ReleasePath var we will use HOMEDIR/binMan
	if config.Config.ReleasePath == "" {
		hDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Unable to detect home directory %v", err)
		}
		config.Config.ReleasePath = hDir + "/binMan"
	}

	if config.Config.TokenVar == "" {
		log.Warn("config.tokenvar is not set. Using anonymous authentication. Please be aware you can quickly be rate limited by github. Instructions here https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token")
		config.Config.TokenVar = "none"
	}

	// Check for UPX
	upxInPath := true

	_, err := exec.LookPath("upx")
	if err != nil {
		upxInPath = false
	}

	// Check if we have globally enabled UPX
	if config.Config.UpxConfig.Enabled == "true" && !upxInPath {
		log.Fatalf("Upx is enabled but not present in $PATH. Please install upx or disable in binman config\n")
	}

	log.Debugf("OS = %s Arch = %s", runtime.GOOS, runtime.GOARCH)

	if config.Defaults.Arch == "" {
		config.Defaults.Arch = runtime.GOARCH
		if config.Defaults.Arch == "amd64" {
			config.Defaults.Arch = x86RegEx
		}
	}

	if config.Defaults.Os == "" {
		config.Defaults.Os = runtime.GOOS
	}

	for k := range config.Releases {

		// set project/org variables
		config.Releases[k].getOR()

		// enable UpxShrink
		if config.Config.UpxConfig.Enabled == "true" {
			if config.Releases[k].UpxConfig.Enabled != "false" {
				config.Releases[k].UpxConfig.Enabled = "true"
			}

			// If release has specifc args do nothing, if not set the defaults from config
			if len(config.Releases[k].UpxConfig.Args) == 0 {
				config.Releases[k].UpxConfig.Args = config.Config.UpxConfig.Args
			}
		}

		if config.Releases[k].Os == "" {
			config.Releases[k].Os = config.Defaults.Os
		}

		if config.Releases[k].Arch == "" {
			config.Releases[k].Arch = config.Defaults.Arch
		}
	}
}

// Add an in default values for most fields :)
func mustUnmarshalYaml(configPath string, v interface{}) {
	yamlFile, err := ioutil.ReadFile(filepath.Clean(configPath))
	if err != nil {
		log.Fatalf("err opening %s   #%v\n", configPath, err)
		os.Exit(1)
	}
	err = yaml.Unmarshal(yamlFile, v)
	if err != nil {
		log.Fatalf("unmarhsal error   #%v\n", err)
		os.Exit(1)
	}
}
