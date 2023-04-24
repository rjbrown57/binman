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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rjbrown57/binman/pkg/constants"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/rodaine/table"
)

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
	CleanupArchive bool      `yaml:"cleanup,omitempty"`      // mark true if archive should be cleaned after extraction
	ReleasePath    string    `yaml:"releasepath,omitempty"`  // path to download/link releases from github
	TokenVar       string    `yaml:"tokenvar,omitempty"`     // Github Auth Token
	NumWorkers     int       `yaml:"maxdownloads,omitempty"` // maximum number of concurrent downloads the user will allow
	UpxConfig      UpxConfig `yaml:"upx,omitempty"`          // Allow upx to shrink extracted
	Sources        []Source  `yaml:"sources,omitempty"`      // Sources to query. By default gitlab and github
	Watch          Watch     `yaml:"watch,omitempty"`        // Watch config object

	sourceMap map[string]*Source // map of names to struct pointers for sources
}

type Watch struct {
	Sync       bool   `yaml:"sync,omitempty"`       // set to true if you want to also pull down releases
	Frequency  int    `yaml:"frequency,omitempty"`  // how often to query for new releases
	Port       string `yaml:"port,omitempty"`       // port to expose prometheus metrics on
	FileServer bool   `yaml:"fileserver,omitempty"` // Start file server of configured release path, must be used in conjunction with sync
	enabled    bool   // private boolean to enable watch mode when invoked by watch subcommand
}

type Source struct {
	Name     string `yaml:"name"`
	Tokenvar string `yaml:"tokenvar,omitempty"`
	URL      string `yaml:"url"`
	Apitype  string `yaml:"apitype"`
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
	metrics  *prometheus.GaugeVec
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

			// set sources
			config.Releases[index].setSource(config.Config.sourceMap)

			// set project/org variables
			config.Releases[index].getOR()

			// If we are running in watch mode set metric and options
			if config.Config.Watch.enabled {
				config.Releases[index].metric = config.metrics
				config.Releases[index].watchSync = config.Config.Watch.Sync
				config.Releases[index].watchExposeMetrics = true
			}

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

	// Set sources if user has not supplied for github.com/gitlab.com
	setDefaultSources(config)

	log.Debugf("set Sources = %+v", config.Config.Sources)

	// If user does not supply a ReleasePath var we will use HOMEDIR/binMan
	if config.Config.ReleasePath == "" {
		hDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Unable to detect home directory %v", err)
		}
		config.Config.ReleasePath = hDir + "/binMan"
	}

	if config.Config.NumWorkers == 0 {
		config.Config.NumWorkers = len(config.Releases)
	}

	if config.Config.TokenVar == "" && config.Config.sourceMap["github.com"].Tokenvar == "" {
		log.Debugf("config.tokenvar is not set. Using anonymous authentication. Please be aware you can quickly be rate limited by github. Instructions here https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token")
		config.Config.sourceMap["github.com"].Tokenvar = "none"
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

// setWatchConfig sets config/releases for watch subcommand
func (config *GHBMConfig) setWatchConfig() {

	config.cleanReleases()
	config.setDefaults()

	// enable watch mode to populate releases sets correct values
	config.Config.Watch.enabled = true

	// Default port to expose metrics / health is 9091
	if config.Config.Watch.Port == "" {
		config.Config.Watch.Port = "9091"
	}

	// Default watch frequency is 60 seconds
	if config.Config.Watch.Frequency == 0 {
		config.Config.Watch.Frequency = 60
	}

	config.metrics = promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "binman_release"}, []string{"latest", "repo", "version"})

	config.populateReleases()
}

// setDefaultSources will handle merging defaults and user sources
// If github/gitlab source keys are missing we add a default
func setDefaultSources(config *GHBMConfig) {

	config.Config.sourceMap = make(map[string]*Source)

	var githubDefault = Source{Name: "github.com", URL: constants.DefaultGHBaseURL, Apitype: "github", Tokenvar: config.Config.TokenVar}
	var gitlabDefault = Source{Name: "gitlab.com", URL: constants.DefaultGLBaseURL, Apitype: "gitlab"}

	for index, source := range config.Config.Sources {

		switch source.Apitype {
		case "gitlab", "github":
		default:
			log.Fatalf("Source %s apitype %s must equal github or gitlab", source.Name, source.Apitype)
		}

		// assign to sourceMap
		config.Config.sourceMap[source.Name] = &config.Config.Sources[index]

		switch source.Name {
		case "github.com":
			// Assign the default url if it's not set correctly
			if source.URL != constants.DefaultGHBaseURL {
				config.Config.Sources[index].URL = constants.DefaultGHBaseURL
			}

			// Compatability for existing githubtoken setting
			if source.Tokenvar == "" && config.Config.TokenVar != "" {
				config.Config.Sources[index].Tokenvar = config.Config.TokenVar
			}
		case "gitlab.com":
			// Assign the default url if it's unset or incorrect
			if source.URL != constants.DefaultGHBaseURL {
				config.Config.Sources[index].URL = constants.DefaultGLBaseURL
			}
		}

	}

	// Add github.com to source array and sourceMap if missing
	if _, exists := config.Config.sourceMap[githubDefault.Name]; !exists {
		config.Config.Sources = append(config.Config.Sources, githubDefault)
		config.Config.sourceMap[githubDefault.Name] = &config.Config.Sources[len(config.Config.Sources)-1]
	}

	// Add gitlab.com to source array and sourceMap if missing
	if _, exists := config.Config.sourceMap[gitlabDefault.Name]; !exists {
		config.Config.Sources = append(config.Config.Sources, gitlabDefault)
		config.Config.sourceMap[gitlabDefault.Name] = &config.Config.Sources[len(config.Config.Sources)-1]
	}
}
