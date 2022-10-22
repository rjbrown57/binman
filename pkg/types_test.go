package binman

import (
	"fmt"
	"os"
	"testing"
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

func TestGetOr(t *testing.T) {

	rel := BinmanRelease{
		Repo: "rjbrown57/binman",
	}

	rel.getOR()
	testRepo := fmt.Sprintf("%s/%s", rel.Org, rel.Project)
	if testRepo != rel.Repo {
		t.Fatalf("%s != %s ; Should be equal", testRepo, rel.Repo)
	}

}

func TestDeduplicate(t *testing.T) {

	d, err := os.MkdirTemp(os.TempDir(), "binmantest")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	configPath := fmt.Sprintf("%s/config", d)

	writeStringtoFile(configPath, dedupConfig)
	if err != nil {
		t.Fatalf("failed to write test config to %s", configPath)
	}

	c := newGHBMConfig(configPath)
	c.deDuplicate()

	if len(c.Releases) != 2 {
		t.Fatal("failed to dedeuplicate release array")
	}
}
