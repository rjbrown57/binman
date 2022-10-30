package binman

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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

func TestWriteReleaseNotes(t *testing.T) {

	d, err := os.MkdirTemp(os.TempDir(), "binmwrn")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	// Create a dummy asset to detect in a subdir of the temp
	var version string = "v0.0.0"
	var bodyContent string = "test-test-test"

	// Create a fake release
	ghData := github.RepositoryRelease{
		TagName: &version,
		Body:    &bodyContent,
	}

	rel := BinmanRelease{
		Repo:        "rjbrown57/binman",
		PublishPath: d,
		GithubData:  &ghData,
	}

	if err = rel.writeReleaseNotes(); err != nil {
		t.Fatal("Unable to write release notes")
	}

	// Read the written release notes
	notesBytes, err := ioutil.ReadFile(filepath.Join(rel.PublishPath, "releaseNotes.txt"))
	if err != nil {
		t.Fatal("Unable to read written release notes")
	}

	if string(notesBytes) != bodyContent {
		t.Fatalf("Want %s, got %s", bodyContent, notesBytes)
	}

}
