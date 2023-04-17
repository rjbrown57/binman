package binman

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/rjbrown57/binman/pkg/constants"
)

func TestBinmanGetReleasePrep(t *testing.T) {

	sourceMap := make(map[string]*Source)

	var githubDefault = Source{Name: "github.com", URL: constants.DefaultGHBaseURL, Apitype: "github"}
	var gitlabDefault = Source{Name: "gitlab.com", URL: constants.DefaultGLBaseURL, Apitype: "gitlab"}

	sourceMap[githubDefault.Name] = &githubDefault
	sourceMap[gitlabDefault.Name] = &gitlabDefault

	// QueryType releasebytag
	m := make(map[string]string)
	m["configFile"] = "config"
	m["repo"] = "org/repo"
	m["version"] = "v0.0.0"

	releasePath, err := os.Getwd()
	if err != nil {
		t.Fatal("Unable to get current working directory")
	}

	m["path"] = releasePath

	// a second map that will be QueryType release
	m2 := make(map[string]string)
	m2["configFile"] = "config"
	m2["repo"] = "org/repo"
	m2["path"] = releasePath
	m2["version"] = ""

	var tests = []struct {
		Expected BinmanRelease
		Got      []BinmanRelease
		dataMap  map[string]string
	}{
		{Expected: BinmanRelease{Repo: m["repo"],
			project:          "repo",
			org:              "org",
			SourceIdentifier: "github.com",
			source:           sourceMap["github.com"],
			Os:               runtime.GOOS,
			Arch:             runtime.GOARCH,
			publishPath:      releasePath,
			DownloadOnly:     true,
			Version:          m["version"],
			QueryType:        "releasebytag"},
			dataMap: m},
		{Expected: BinmanRelease{Repo: m["repo"],
			project:          "repo",
			org:              "org",
			SourceIdentifier: "github.com",
			source:           sourceMap["github.com"],
			Os:               runtime.GOOS,
			Arch:             runtime.GOARCH,
			publishPath:      releasePath,
			DownloadOnly:     true,
			QueryType:        "release"},
			dataMap: m2},
	}

	for _, test := range tests {
		test.Got = BinmanGetReleasePrep(sourceMap, test.dataMap)
		if fmt.Sprintf("%v", test.Got[0]) != fmt.Sprintf("%v", test.Expected) {
			t.Fatalf("\n got - %+v\n expected - %+v", test.Got[0], test.Expected)
		}
	}
}
