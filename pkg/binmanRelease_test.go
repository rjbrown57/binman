package binman

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-github/v48/github"
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

func TestFindTarget(t *testing.T) {

	d, err := os.MkdirTemp(os.TempDir(), "binmft")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	// Create test file dir within temp dir
	td := fmt.Sprintf("%s/%s", d, "test")
	os.Mkdir(td, 0644)
	if err != nil {
		t.Fatalf("unable to make temp dir %s", td)
	}

	// Create a dummy asset to detect in a subdir of the temp
	var testFileName string = "binman"
	var version string = "v0.0.0"
	afp := fmt.Sprintf("%s/%s", td, testFileName)
	writeStringtoFile(afp, "test-test-test")
	if err != nil {
		t.Fatalf("unable to write string to file %s", afp)
	}

	// Create a fake release
	ghData := github.RepositoryRelease{
		TagName: &version,
	}

	rel := BinmanRelease{
		Repo:         "rjbrown57/binman",
		ArtifactPath: "binman", // a dummy value, this will be used with filepath.base to return the assetName. Post findTarget should == the full path to target
		PublishPath:  d,
		Os:           "linux",
		Arch:         "amd64",
		GithubData:   &ghData,
	}

	rel.findTarget()

	// We should find our file in the subdir
	if rel.ArtifactPath != testFileName {
		t.Fatalf("Expected %s got %s", rel.ArtifactPath, afp)
	}
}
