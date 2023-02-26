package binman

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/fatih/color"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/rodaine/table"
)

const TarRegEx = `(\.tar$|\.tar\.gz$|\.tgz$)`
const ZipRegEx = `(\.zip$)`

// BinmanMsg contains return messages for binman's concurrent workers
type BinmanMsg struct {
	err error
	rel BinmanRelease
}

func OutputResults(out map[string][]BinmanMsg, debug bool) {

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	upToDateTable := table.New("Repo", "Version", "State")
	upToDateTable.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for key, msgSlice := range out {
		for _, msg := range msgSlice {
			upToDateTable.AddRow(msg.rel.Repo, msg.rel.Version, key)
		}
	}

	upToDateTable.Print()
}

type UpxConfig struct {
	Enabled string   `yaml:"enabled,omitempty"` // Using a string here instead of a boolean to deal with an unset boolean defaulting to false
	Args    []string `yaml:"args,omitempty"`
}

// BinmanConfig contains Global Config Options
type BinmanConfig struct {
	CleanupArchive bool      `yaml:"cleanup,omitempty"`     // mark true if archive should be cleaned after extraction
	ReleasePath    string    `yaml:"releasepath,omitempty"` //path to download/link releases from github
	TokenVar       string    `yaml:"tokenvar,omitempty"`    //Github Auth Token
	UpxConfig      UpxConfig `yaml:"upx,omitempty"`         // Allow upx to shrink extracted
}

// BinmanDefaults contains default config options. If a value is unset in releases array these will be used.
// This should just be collapsed into BinmanConfig and this struct should be removed
type BinmanDefaults struct {
	Os   string `yaml:"os,omitempty"`   //OS architechrue to look for
	Arch string `yaml:"arch,omitempty"` //OS architechrue to look for
}

// Type that rolls up the above types into one happy family
type GHBMConfig struct {
	Config   BinmanConfig    `yaml:"config"`
	Defaults BinmanDefaults  `yaml:"defaults,omitempty"`
	Releases []BinmanRelease `yaml:"releases"`
}

func NewGHBMConfig(configPath string) *GHBMConfig {
	config := &GHBMConfig{}
	mustUnmarshalYaml(configPath, config)
	return config
}

// Deduplicate releases
func (config *GHBMConfig) cleanReleases() {

	var deduplicatedReleases []BinmanRelease

	releaseMap := make(map[string]BinmanRelease)

	// Iterate over all releases populating releaseMap.
	// We iterate over the slice in reverse. This way if a contextual config contains a duplicate the version from the contexual config will be tossed out
	for index := len(config.Releases) - 1; index >= 0; index-- {

		// Convert string representation of all values to a string representation of the byte array
		// This will allow multiple versions of one repo with different settings, but overwrite in case of duplicate
		relString := fmt.Sprintf("%x", fmt.Sprintf("%v", config.Releases[index]))
		if config.Releases[index].Repo != "" && strings.Contains(config.Releases[index].Repo, "/") {
			releaseMap[relString] = config.Releases[index]
		} else {
			log.Debugf("release %d is malformed. Skipping for now, - %v", index, config.Releases[index])
		}
	}

	// Make the final release slice
	// Since we reversed the order to deduplicate, now "prepend" to restore the original release order
	for _, rel := range releaseMap {
		deduplicatedReleases = append([]BinmanRelease{rel}, deduplicatedReleases...)
	}

	config.Releases = deduplicatedReleases
}

// populateReleases applies defaults and does prep work on each release in our config
func (config *GHBMConfig) populateReleases() {

	var wg sync.WaitGroup

	for k := range config.Releases {
		wg.Add(1)
		go func(index int) {

			defer wg.Done()

			// set project/org variables
			config.Releases[index].getOR()

			// Configure the query type
			// release is the default, if a version is set releasebytag
			// for repos without releases we could offer getting via tag, but it's proven an ugly process
			// https://github.com/rjbrown57/binman/tree/querybytag
			switch config.Releases[index].QueryType {
			case "release":
				fallthrough
			default:
				config.Releases[index].QueryType = "release"

				if config.Releases[index].Version != "" {
					config.Releases[index].QueryType = "releasebytag"
				}
			}

			// If the user has not supplied an external url check against our map of known external urls
			if config.Releases[index].ExternalUrl == "" {
				config.Releases[index].knownUrlCheck()
			}

			// enable UpxShrink
			if config.Config.UpxConfig.Enabled == "true" {
				if config.Releases[index].UpxConfig.Enabled != "false" {
					config.Releases[index].UpxConfig.Enabled = "true"
				}

				// If release has specifc args do nothing, if not set the defaults from config
				if len(config.Releases[index].UpxConfig.Args) == 0 {
					config.Releases[index].UpxConfig.Args = config.Config.UpxConfig.Args
				}
			}

			if config.Config.CleanupArchive {
				config.Releases[index].CleanupArchive = true
			}

			if config.Releases[index].Os == "" {
				config.Releases[index].Os = config.Defaults.Os
			}

			if config.Releases[index].Arch == "" {
				config.Releases[index].Arch = config.Defaults.Arch
			}

			if config.Releases[index].ReleasePath == "" {
				config.Releases[index].ReleasePath = config.Config.ReleasePath
			}

			p, err := filepath.Abs(config.Config.ReleasePath)
			if err != nil {
				log.Fatalf("Unable to get absolute path of %s", config.Config.ReleasePath)
			}
			config.Releases[index].ReleasePath = p

		}(k)
	}
	// Wait until all defaults have been set
	wg.Wait()
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
		log.Debugf("config.tokenvar is not set. Using anonymous authentication. Please be aware you can quickly be rate limited by github. Instructions here https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token")
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
	}

	if config.Defaults.Os == "" {
		config.Defaults.Os = runtime.GOOS
	}
}
