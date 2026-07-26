package binman

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v50/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rjbrown57/binman/pkg/constants"
	db "github.com/rjbrown57/binman/pkg/db"
	"github.com/rjbrown57/binman/pkg/downloader"
	"github.com/rjbrown57/binman/pkg/gh"
	"github.com/rjbrown57/binman/pkg/gl"
	log "github.com/rjbrown57/binman/pkg/logging"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const timeout = 60 * time.Second

// goSyncRepo executes all Actions required by a repo. Action are executed sequentially
// Actions are executed in 4 phases
// Pre -> Post -> Os -> Final
// The last action of each phase sets the actions for the next phase
// The Final actions is to set rel.actions = nil and conclude the loop
func goSyncRepo(rel BinmanRelease, c chan<- BinmanMsg) {

	var err error

	rel.actions = rel.setPreActions(rel.ReleasePath, rel.BinPath)

	log.Debugf("release %s = %+v source = %+v", rel.Repo, rel, rel.source)

	for rel.actions != nil {
		if err = rel.runActions(); err != nil {
			switch err.(type) {
			case *NoUpdateError:
				c <- BinmanMsg{Rel: rel, Err: err}
				return
			case *ExcludeError:
				c <- BinmanMsg{Rel: rel, Err: err}
				return
			default:
				c <- BinmanMsg{Rel: rel, Err: err}
				if rel.cleanupOnFailure {
					err := os.RemoveAll(rel.PublishPath)
					if err != nil {
						log.Debugf("Unable to clean up %s - %s", rel.PublishPath, err)
					}
					log.Debugf("cleaned %s\n", rel.PublishPath)
					log.Debugf("Final release data  %+v\n", rel)
				}
				return
			}
		}
	}

	c <- BinmanMsg{Rel: rel, Err: nil}
}

// Type that rolls up the above types into one happy family
type BMConfig struct {
	Config        BinmanConfig    `yaml:"config"`
	ConfigPath    string          `yaml:",omitempty"`
	Defaults      BinmanDefaults  `yaml:"defaults,omitempty"`
	Releases      []BinmanRelease `yaml:"releases"`
	Msgs          []BinmanMsg     `yaml:",omitempty"` // Output from execution
	OutputOptions *OutputOptions  `yaml:",omitempty"`

	Metrics *prometheus.GaugeVec

	// DB Ops
	dbOptions    db.DbConfig
	msgChan      chan BinmanMsg
	downloadChan chan downloader.DlMsg
	wg           sync.WaitGroup

	// Shared per-source API clients, keyed by source name. Built once so we
	// stop minting a fresh client per repo.
	ghClients map[string]*github.Client
	glClients map[string]*gitlab.Client
}

// For running the default sync
func NewBMSync(configPath string, table bool) *BMConfig {
	return NewBMConfig(configPath).WithDb().WithDownloader().WithOutput(table, true).SetConfig(true)
}

// For running the default sync
func NewBMWatch(configPath string) *BMConfig {
	return NewBMConfig(configPath).WithWatch().WithDb().WithDownloader().WithOutput(false, false).SetConfig(false)
}

// For running the get command
func NewGet(r ...BinmanRelease) *BMConfig {
	c := &BMConfig{}

	for _, rel := range r {
		c.Releases = append(c.Releases, rel)
	}

	c = c.WithDownloader().WithOutput(false, true)
	c.SetDefaults()
	c.populateReleases()
	c.startDownloaders()

	return c
}

// This is for when you want to query a single repo
func NewQuery(r ...BinmanRelease) *BMConfig {
	c := &BMConfig{}
	for _, rel := range r {
		c.Releases = append(c.Releases, rel)
	}
	return c
}

func NewBMConfig(configPath string) *BMConfig {

	config := &BMConfig{}
	config.ConfigPath = SetBaseConfig(configPath)

	return config
}

// BMClose will close all channels in use
func (config *BMConfig) BMClose() {

	if config.downloadChan != nil {
		log.Tracef("Closing download chan")
		close(config.downloadChan)
	}

	if config.dbOptions.DbChan != nil {
		// While this is likely unecessary since CollectData will not conclude
		// until the DB responds to each transaction
		// a wait is added here to ensure all transactions have concluded
		config.dbOptions.Dwg.Wait()
		log.Tracef("Closing db chan")
		close(config.dbOptions.DbChan)
	}

	if config.OutputOptions.SpinChan != nil {
		log.Tracef("Closing spin chan")
		close(config.OutputOptions.SpinChan)
		config.OutputOptions.Swg.Wait()
	}
}

func (config *BMConfig) WithDb(dbConfig ...db.DbConfig) *BMConfig {

	var dwg sync.WaitGroup

	if dbConfig != nil {
		config.dbOptions = dbConfig[0]
	} else {
		config.dbOptions = db.DbConfig{
			Dwg:    &dwg,
			DbChan: make(chan db.DbMsg),
			// if a binman sync attempts to write something to the DB it has synced a new release.
			//So we should always allow it to override any possibly out of date info.
			Overwrite: true,
		}
	}

	// Initialize the DB if required
	if checkNewDb(config.dbOptions.Path) {
		log.Debugf("Initializing DB")
		populateDB(config.dbOptions, config.ConfigPath)
		// populateDB will close the channel when done, so we need to open a new one.
		config.dbOptions.DbChan = make(chan db.DbMsg)
	}

	// Start the DB
	go db.RunDB(config.dbOptions)

	return config
}

func (config *BMConfig) WithDownloader() *BMConfig {
	config.downloadChan = make(chan downloader.DlMsg)
	return config
}

// startDownloaders spawns the download worker pool. This must run after the
// config has been parsed, because WithDownloader is chained before SetConfig
// and NumWorkers only holds the user's maxdownloads once the yaml is read.
// Spawning here instead of in WithDownloader is what lets the pool actually
// honor maxdownloads.
func (config *BMConfig) startDownloaders() {

	// Callers like NewQuery never attach a downloader, so there is no pool to start
	if config.downloadChan == nil {
		return
	}

	workers := config.Config.NumWorkers
	if workers < 1 {
		workers = 1
	}

	log.Debugf("launching %d download workers", workers)

	for worker := 1; worker <= workers; worker++ {
		go downloader.GetDownloader(config.downloadChan, worker)
	}
}

// setWatchConfig sets config/releases for watch subcommand
func (config *BMConfig) WithWatch() *BMConfig {

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

	config.Metrics = promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "binman_release"}, []string{"latest", "source", "repo", "version"})

	return config

}

// WithOutput configures output for an operation
func (config *BMConfig) WithOutput(table, spinner bool) *BMConfig {
	config.OutputOptions = NewOutputOptions(table, spinner)
	return config
}

// setConfig will create the appropriate BMConfig and merge if required
func (config *BMConfig) SetConfig(merge bool) *BMConfig {

	mustUnmarshalYaml(config.ConfigPath, config)

	// If ${repoDir}/.binMan.yaml exists we merge it's releases with our main config
	cfg, cfgBool := detectRepoConfig()
	if cfgBool && merge {
		log.Debugf("Found %s merging with main config", cfg)
		tc := NewBMConfig(cfg)
		// append releases from the contextual config
		config.Releases = append(config.Releases, tc.Releases...)
	}

	config.SetDefaults()
	config.cleanReleases()
	config.populateReleases()

	// SetDefaults has finalized NumWorkers by this point, so the pool is sized correctly
	config.startDownloaders()

	return config
}

// setupClients builds one shared API client per configured source. Clients are
// safe for concurrent use, so we build them once and share them across all repos
// rather than constructing a new client for every release.
func (config *BMConfig) setupClients() {

	if config.ghClients == nil {
		config.ghClients = make(map[string]*github.Client)
	}
	if config.glClients == nil {
		config.glClients = make(map[string]*gitlab.Client)
	}

	for name, source := range config.Config.SourceMap {
		switch source.Apitype {
		case "github":
			if _, ok := config.ghClients[name]; !ok {
				config.ghClients[name] = gh.GetGHCLient(source.URL, source.Tokenvar)
			}
		case "gitlab":
			if _, ok := config.glClients[name]; !ok {
				config.glClients[name] = gl.GetGLClient(source.URL, source.Tokenvar)
			}
		}
	}
}

// checkQuota does a single, non-fatal pre-flight check comparing the number of
// configured github repos against the token's remaining API quota. We only warn:
// the real release calls surface actual rate-limit errors on their own, and the
// /rate_limit endpoint itself can be flaky, so it must never abort the run.
func (config *BMConfig) checkQuota() {

	ghClient, ok := config.ghClients[config.Config.SourceMap["github.com"].Name]
	if !ok {
		return
	}

	var ghRepos int
	for i := range config.Releases {
		if s := config.Releases[i].source; s != nil && s.Apitype == "github" {
			ghRepos++
		}
	}

	if ghRepos == 0 {
		return
	}

	remaining, limit, err := gh.GetRateLimit(ghClient)
	if err != nil {
		log.Warnf("Unable to check GitHub API rate limits, continuing anyway: %v", err)
		return
	}

	log.Debugf("GitHub API quota: %d/%d remaining", remaining, limit)

	if ghRepos > remaining {
		log.Warnf("%d github repos are configured but only %d GitHub API requests remain (quota resets hourly). Some repos may fail this run.", ghRepos, remaining)
	}
}

// Execute will collect data and execute any requested steps
func (config *BMConfig) CollectData() {

	config.setupClients()

	// Assign the shared per-source client to each release before we fan out.
	for i := range config.Releases {
		if s := config.Releases[i].source; s != nil {
			config.Releases[i].ghClient = config.ghClients[s.Name]
			config.Releases[i].glClient = config.glClients[s.Name]
		}
	}

	config.checkQuota()

	c := make(chan BinmanMsg)

	var wg sync.WaitGroup

	// Bound the get phase to NumAPIWorkers so we don't fire one unbounded API
	// request per repo, which is what trips GitHub's rate and abuse limits.
	// It falls back to NumWorkers so maxapiqueries stays optional, while users
	// with many repos can widen the query fanout without widening downloads.
	workers := config.Config.NumAPIWorkers
	if workers < 1 {
		workers = config.Config.NumWorkers
	}
	if workers < 1 {
		workers = len(config.Releases)
	}
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan BinmanRelease)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rel := range jobs {
				goSyncRepo(rel, c)
			}
		}()
	}

	go func() {
		for _, rel := range config.Releases {
			jobs <- rel
		}
		close(jobs)
	}()

	go func(c chan BinmanMsg, wg *sync.WaitGroup) {
		wg.Wait()
		close(c)
	}(c, &wg)

	for msg := range c {
		config.Msgs = append(config.Msgs, msg)
	}
}

// Deduplicate releases
func (config *BMConfig) cleanReleases() {

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
func (config *BMConfig) populateReleases() {

	var wg sync.WaitGroup

	// This too complex and needs to be simplified
	for k := range config.Releases {
		wg.Add(1)
		go func(index int) {

			defer wg.Done()

			// Set Db wg/chan
			config.Releases[index].dbChan = config.dbOptions.DbChan
			config.Releases[index].dwg = config.dbOptions.Dwg

			config.Releases[index].downloadChan = config.downloadChan

			// set sources
			config.Releases[index].SetSource(config.Config.SourceMap)

			// set project/org variables
			config.Releases[index].getOR()

			// Set Output
			config.Releases[index].output = config.OutputOptions

			// If we are running in watch mode set metric and options
			if config.Config.Watch.enabled {
				config.Releases[index].metric = config.Metrics
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
			if config.Releases[index].ExternalUrl == "" && config.Releases[index].source.Apitype != "binman" {
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

			if config.Releases[index].BinPath == "" {
				config.Releases[index].BinPath = config.Config.BinPath
			}

			p, err = filepath.Abs(config.Config.BinPath)
			if err != nil {
				log.Fatalf("Unable to get absolute path of %s", config.Config.BinPath)
			}
			config.Releases[index].BinPath = p

		}(k)
	}
	// Wait until all defaults have been set
	wg.Wait()
}

func (config *BMConfig) GetRelease(repo string) (BinmanRelease, error) {
	for _, r := range config.Releases {
		if r.Repo == repo {
			return r, nil
		}
	}
	return BinmanRelease{Repo: repo}, ErrReleaseNotFound
}

// SetDefaults will populate defaults, and required values
func (config *BMConfig) SetDefaults() {

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

	// If user does not supply a BinPath var we will use ReleasePath
	if config.Config.BinPath == "" {
		_, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Unable to detect home directory %v", err)
		}
		config.Config.BinPath = config.Config.ReleasePath
	}

	if config.Config.NumWorkers == 0 {
		config.Config.NumWorkers = len(config.Releases)
	}

	if config.Config.TokenVar == "" && config.Config.SourceMap["github.com"].Tokenvar == "" {
		log.Debugf("config.tokenvar is not set. Using anonymous authentication. Please be aware you can quickly be rate limited by github. Instructions here https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token")
		config.Config.SourceMap["github.com"].Tokenvar = "none"
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

	// If the config does not set a default source, we will set it to github.com
	// If the user does set one, then we mark that as the default
	if config.Defaults.Source == "" {
		config.Defaults.Source = config.Config.SourceMap["github.com"].Name
		config.Config.SourceMap["default"] = config.Config.SourceMap["github.com"]
	} else {
		config.Config.SourceMap["default"] = config.Config.SourceMap[config.Defaults.Source]
	}
}

// setDefaultSources will handle merging defaults and user sources
// If github/gitlab source keys are missing we add a default
func setDefaultSources(config *BMConfig) {

	config.Config.SourceMap = make(map[string]*Source)

	var githubDefault = Source{Name: "github.com", URL: constants.DefaultGHBaseURL, Apitype: "github", Tokenvar: config.Config.TokenVar}
	var gitlabDefault = Source{Name: "gitlab.com", URL: constants.DefaultGLBaseURL, Apitype: "gitlab"}

	for index, source := range config.Config.Sources {

		switch source.Apitype {
		case "gitlab", "github", "binman":
		default:
			log.Fatalf("Source %s apitype %s must equal github/gitlab or binman", source.Name, source.Apitype)
		}

		// assign to sourceMap
		config.Config.SourceMap[source.Name] = &config.Config.Sources[index]

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
	if _, exists := config.Config.SourceMap[githubDefault.Name]; !exists {
		config.Config.Sources = append(config.Config.Sources, githubDefault)
		config.Config.SourceMap[githubDefault.Name] = &config.Config.Sources[len(config.Config.Sources)-1]
	}

	// Add gitlab.com to source array and sourceMap if missing
	if _, exists := config.Config.SourceMap[gitlabDefault.Name]; !exists {
		config.Config.Sources = append(config.Config.Sources, gitlabDefault)
		config.Config.SourceMap[gitlabDefault.Name] = &config.Config.Sources[len(config.Config.Sources)-1]
	}
}

type OutputOptions struct {
	Table   bool
	Spinner bool

	SpinChan chan string
	Swg      sync.WaitGroup
}

// Set table for Table based output after since, if a spinner is needed by the operation set Spinner
func NewOutputOptions(table, spinner bool) *OutputOptions {
	o := OutputOptions{Table: table, Spinner: spinner}
	if o.Spinner {
		o.SpinChan = make(chan string)
		go getSpinner(log.IsDebug(), o.SpinChan, &o.Swg)
	}
	return &o
}

func (o *OutputOptions) SendSpin(s string) {
	if o.Spinner {
		o.Swg.Add(1)
		o.SpinChan <- s
	}
}
