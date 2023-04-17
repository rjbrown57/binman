package binman

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/rjbrown57/binman/pkg/constants"
)

const mergeConfig = `
config:
  releasepath:  #path to keep fetched releases. $HOME/binMan is the default
  tokenvar: #environment variable that contains github token
  upx: #Compress binaries with upx
    enabled: false
    args: [] # arrary of args for upx
releases:
  - repo: anchore/syft
  - repo: anchore/grype
`

const dedupConfig = `
config:
  releasepath:  #path to keep fetched releases. $HOME/binMan is the default
  tokenvar: #environment variable that contains github token
  upx: #Compress binaries with upx
    enabled: false
    args: [] # arrary of args for upx
releases:
  - repo: rjbrown57/binman
  - repo: rjbrown57/binman
    releasefilename:  binman_darwin_amd64 
  - repo: rjbrown57/binman
`

const testConfig = `
config:
  releasepath: thereleasepath
  tokenvar: thetoken
  upx:
    enabled: true
    args: []
releases:
  - repo: rjbrown57/binman
`

const testConfigEmptyVals = `
config:
  releasepath: 
  tokenvar:
  upx:
    enabled: true
    args: []
releases:
  - repo: rjbrown57/binman
`

func TestCleanReleases(t *testing.T) {

	d, err := os.MkdirTemp(os.TempDir(), "binmantest")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	configPath := fmt.Sprintf("%s/config", d)

	WriteStringtoFile(configPath, dedupConfig)
	if err != nil {
		t.Fatalf("failed to write test config to %s", configPath)
	}

	c := NewGHBMConfig(configPath)
	c.cleanReleases()

	if len(c.Releases) != 2 {
		t.Fatal("failed to dedeuplicate release array")
	}
}

func TestSetDefaults(t *testing.T) {
	d, err := os.MkdirTemp(os.TempDir(), "binmantest")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	configPath := fmt.Sprintf("%s/config", d)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Unable to detected user home directory %s", err)
	}

	defaultSourceMap := make(map[string]*Source)
	defaultSourceMap["github.com"] = &Source{Name: "github.com", URL: constants.DefaultGHBaseURL, Apitype: "github", Tokenvar: "thetoken"}
	defaultSourceMap["gitlab.com"] = &Source{Name: "gitlab.com", URL: constants.DefaultGLBaseURL, Apitype: "gitlab", Tokenvar: "thetoken"}

	var tests = []struct {
		config              string
		expectedOs          string
		expectedArch        string
		expectedReleasePath string
		expectedTokenVar    string
		expectedQueryType   string
		expectedSourceMap   map[string]*Source
	}{
		{testConfig, runtime.GOOS, runtime.GOARCH, "thereleasepath", "thetoken", "release", defaultSourceMap},
		{testConfigEmptyVals, runtime.GOOS, runtime.GOARCH, homeDir + "/" + "binMan", "none", "release", defaultSourceMap},
	}

	for _, test := range tests {
		WriteStringtoFile(configPath, test.config)
		if err != nil {
			t.Fatalf("failed to write test config to %s", configPath)
		}
		c := NewGHBMConfig(configPath)
		c.setDefaults()

		// test the defaults
		if c.Defaults.Arch != test.expectedArch || c.Defaults.Os != test.expectedOs {
			t.Fatalf("Expected %s,%s got %s,%s", c.Defaults.Arch, c.Defaults.Os, test.expectedArch, test.expectedOs)
		}

		if c.Config.TokenVar != test.expectedTokenVar {
			t.Fatalf("Expected %s got %s", test.expectedTokenVar, c.Config.TokenVar)
		}

		if c.Config.ReleasePath != test.expectedReleasePath {
			t.Fatalf("Expected %s got %s", test.expectedReleasePath, c.Config.ReleasePath)
		}

		if reflect.DeepEqual(c.Config.sourceMap, test.expectedSourceMap) {
			t.Fatalf("Expected %+v, got %+v,", c.Config.sourceMap["github.com"], test.expectedSourceMap["github.com"])
		}
	}
}

const testConfigPopulateTest = `
config:
  releasepath: "/tmp/"
  tokenvar:
  upx:
    enabled: true
    args: []
releases:
  - repo: rjbrown57/binman
  - repo: rjbrown57/binextractor 
    upx:
      enabled: true
      args: ["-k", "-v"]
  - repo: rjbrown57/lp
    upx:
      enabled: false
  - repo: hashicorp/vault
`

func TestPopulateReleases(t *testing.T) {

	d, err := os.MkdirTemp(os.TempDir(), "binmantest")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	configPath := fmt.Sprintf("%s/config", d)

	WriteStringtoFile(configPath, testConfigPopulateTest)
	if err != nil {
		t.Fatalf("failed to write test config to %s", configPath)
	}

	testUpxConfigTrue := UpxConfig{
		Enabled: "true",
	}

	testUpxConfigFalse := UpxConfig{
		Enabled: "false",
	}

	// Releases are marshalled in the reverse of the order set in the config. So we reverse the config order here.
	testRelSlice := []BinmanRelease{
		{
			Repo:         "rjbrown57/binman",
			org:          "rjbrown57",
			project:      "binman",
			Os:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			CheckSum:     false,
			DownloadOnly: false,
			UpxConfig:    testUpxConfigTrue,
			ReleasePath:  filepath.Clean("/tmp/"),
		},
		{
			Repo:         "rjbrown57/binextractor",
			org:          "rjbrown57",
			project:      "extractor",
			Os:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			CheckSum:     false,
			DownloadOnly: false,
			UpxConfig: UpxConfig{
				Enabled: "true",
				Args:    []string{"-k", "-v"},
			},
			ReleasePath: filepath.Clean("/tmp/"),
		},
		{
			Repo:         "rjbrown57/lp",
			org:          "rjbrown57",
			project:      "lp",
			Os:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			CheckSum:     false,
			DownloadOnly: false,
			UpxConfig:    testUpxConfigFalse,
			ReleasePath:  filepath.Clean("/tmp/"),
		},
		{
			Repo:         "hashicorp/vault",
			org:          "hashicorp",
			project:      "vault",
			Os:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			CheckSum:     false,
			DownloadOnly: false,
			UpxConfig:    testUpxConfigTrue,
			ExternalUrl:  `https://releases.hashicorp.com/vault/{{ trimPrefix "v" .version }}/vault_{{ trimPrefix "v" .version }}_{{.os}}_{{.arch}}.zip`,
			ReleasePath:  filepath.Clean("/tmp/"),
		},
	}

	got := NewGHBMConfig(configPath)

	expected := &GHBMConfig{
		Config: BinmanConfig{

			ReleasePath: filepath.Clean("/tmp/"),
			TokenVar:    "none",
			UpxConfig:   testUpxConfigTrue,
		},
		Releases: testRelSlice,
		Defaults: BinmanDefaults{
			Os:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
	}

	got.setDefaults()
	got.populateReleases()

	for k := range got.Releases {

		if got.Releases[k].Repo != expected.Releases[k].Repo {
			t.Fatalf("\n Repo: Got %+v != \n Expected %+v", got.Releases[k].Repo, expected.Releases[k].Repo)
		}

		if got.Releases[k].Arch != expected.Releases[k].Arch {
			t.Fatalf("\n Arch: Got %+v != \n Expected %+v", got.Releases[k].Arch, expected.Releases[k].Arch)
		}

		if got.Releases[k].Os != expected.Releases[k].Os {
			t.Fatalf("\n Os: Got %+v != \n Expected %+v", got.Releases[k].Os, expected.Releases[k].Os)
		}

		if got.Releases[k].UpxConfig.Enabled != expected.Releases[k].UpxConfig.Enabled {
			t.Fatalf("\n UpxConfig: Got %+v != \n Expected %+v", got.Releases[k].UpxConfig.Enabled, expected.Releases[k].UpxConfig.Enabled)
		}

		if len(got.Releases[k].UpxConfig.Args) != len(expected.Releases[k].UpxConfig.Args) {
			t.Fatalf("\n UpxConfig Args: Got %+v != \n Expected %+v", got.Releases[k].UpxConfig.Args, expected.Releases[k])
		}

		if got.Releases[k].ExternalUrl != expected.Releases[k].ExternalUrl {
			t.Fatalf("\n ExternalUrl Got %+v != \n Expected %+v", got.Releases[k].ExternalUrl, expected.Releases[k].ExternalUrl)
		}

		if got.Releases[k].ReleasePath != expected.Releases[k].ReleasePath {
			t.Fatalf("\n ReleasePath Got %+v != Expected %+v", got.Releases[k].ReleasePath, expected.Releases[k].ReleasePath)
		}

	}
}
