package binman

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-github/v50/github"
	log "github.com/rjbrown57/binman/pkg/logging"
)

func TestLibrary(t *testing.T) {

	log.ConfigureLog(true, 2)

	d, _ := configTestHelper(t)

	t.Cleanup(func() {
		os.RemoveAll(d)
	})

	// Add a default config
	cf := fmt.Sprintf(d + "/" + ".binMan.yaml")
	if err := WriteStringtoFile(cf, dedupConfig); err != nil {
		t.Fatalf("Unable to write test config")
	}

	c := NewBMConfig(cf)
	c.SetConfig()
	c.CollectData()

	for _, x := range c.Msgs {
		log.Infof("%+v", x)
	}

	data := c.Msgs[0].rel.relData.(*github.RepositoryRelease)

	log.Infof("%s - %s", c.Releases[0].Repo, data.GetTagName())
}
