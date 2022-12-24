package binman

import (
	"fmt"
	"os"
	"runtime"
	"testing"
)

func TestBinmanGetReleasePrep(t *testing.T) {
	m := make(map[string]string)
	m["configFile"] = "config"
	m["repo"] = "org/repo"
	m["version"] = "v0.0.0"
	releasePath, err := os.Getwd()
	m["path"] = releasePath
	if err != nil {
		t.Fatal("Unable to get current working director")
	}

	expectedRel := BinmanRelease{
		Repo:         m["repo"],
		project:      "repo",
		org:          "org",
		Os:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		publishPath:  releasePath,
		QueryType:    "release",
		DownloadOnly: true,
		Version:      m["version"],
	}

	gotRel := BinmanGetReleasePrep(m)

	if fmt.Sprintf("%v", gotRel[0]) != fmt.Sprintf("%v", expectedRel) {
		t.Fatalf("%v != %v", gotRel[0], expectedRel)
	}
}
