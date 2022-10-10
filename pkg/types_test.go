package binman

import (
	"fmt"
	"testing"
)

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
