package binman

import (
	"errors"
)

var (
	ErrReleaseNotFound = errors.New("Requested release not found in config")
)

// BinmanMsg contains return messages for binman's concurrent workers
type BinmanMsg struct {
	Err error
	Rel BinmanRelease
}

type UpxConfig struct {
	Enabled string   `yaml:"enabled,omitempty"` // Using a string here instead of a boolean to deal with an unset boolean defaulting to false
	Args    []string `yaml:"args,omitempty"`
}

// BinmanConfig contains Global Config Options
type BinmanConfig struct {
	CleanupArchive bool      `yaml:"cleanup,omitempty"`      // mark true if archive should be cleaned after extraction
	ReleasePath    string    `yaml:"releasepath,omitempty"`  // path to download/link releases from github
	BinPath        string    `yaml:"binpath,omitempty"`      // path to download/link binaries from github
	TokenVar       string    `yaml:"tokenvar,omitempty"`     // Github Auth Token
	NumWorkers     int       `yaml:"maxdownloads,omitempty"` // maximum number of concurrent downloads the user will allow
	UpxConfig      UpxConfig `yaml:"upx,omitempty"`          // Allow upx to shrink extracted
	Sources        []Source  `yaml:"sources,omitempty"`      // Sources to query. By default gitlab and github
	Watch          Watch     `yaml:"watch,omitempty"`        // Watch config object

	SourceMap map[string]*Source `yaml:"-"` // map of names to struct pointers for sources
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
