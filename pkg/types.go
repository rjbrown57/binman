package binman

import (
	b64 "encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

const TarRegEx = `(\.tar$|\.tar\.gz$|\.tgz$)`
const ZipRegEx = `(\.zip$)`

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

// Type that rolls up the above types into one happy family
type GHBMConfig struct {
	Config   BinmanConfig    `yaml:"config"`
	Defaults BinmanDefaults  `yaml:"defaults"`
	Releases []BinmanRelease `yaml:"releases"`
}

func newGHBMConfig(configPath string) *GHBMConfig {
	config := &GHBMConfig{}
	mustUnmarshalYaml(configPath, config)
	return config
}

// Deduplicate releases
func (config *GHBMConfig) deDuplicate() {

	var deduplicatedReleases []BinmanRelease

	releaseMap := make(map[string]BinmanRelease)

	// Iterate over all releases populating releaseMap.
	// We iterate over the slice in reverse. THis way if a contextual config contains a duplicate the version from the contexual config will be tossed out
	for index := len(config.Releases) - 1; index >= 0; index-- {

		// Convert text representation of all values per release to b64.
		// This will allow multiple versions of one repo with different settings
		relString := fmt.Sprintf("%v", config.Releases[index])
		b64string := b64.StdEncoding.EncodeToString([]byte(relString))

		releaseMap[b64string] = config.Releases[index]
	}

	// Make the final release slice
	for _, rel := range releaseMap {
		deduplicatedReleases = append(deduplicatedReleases, rel)
	}

	config.Releases = deduplicatedReleases
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
	}

	if config.Defaults.Os == "" {
		config.Defaults.Os = runtime.GOOS
	}

	// DeDuplicate before we populate defaults
	config.deDuplicate()

	for k := range config.Releases {

		// set project/org variables
		config.Releases[k].getOR()

		// If the user has not supplied an external url check against our map of known external urls
		if config.Releases[k].ExternalUrl == "" {
			config.Releases[k].knownUrlCheck()
		}

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
