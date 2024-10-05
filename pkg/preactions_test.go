package binman

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseStatusAction(t *testing.T) {

	d, err := os.MkdirTemp(os.TempDir(), "binmwrn")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	// Create a dummy asset to detect in a subdir of the temp
	var version string = "v0.0.0"

	rNoUpdate := BinmanRelease{
		Repo:        "rjbrown57/binman",
		org:         "rjbrown57",
		project:     "binman",
		PublishPath: d,
		Version:     version,
	}

	rRequiresUpdate := BinmanRelease{
		Repo:        "rjbrown57/noexist",
		org:         "rjbrown57",
		project:     "noexist",
		PublishPath: d,
		Version:     version,
	}

	// For the first test this path must exist
	publishPath := filepath.Join(d, "repos", rNoUpdate.org, rNoUpdate.project, version)
	err = CreateDirectory(publishPath)
	if err != nil {
		t.Fatalf("unable to make temp dir %s", publishPath)
	}

	if err = rRequiresUpdate.AddReleaseStatusAction(d).execute(); err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	noUpErr := &NoUpdateError{}
	if err = rNoUpdate.AddReleaseStatusAction(d).execute(); err != nil && !errors.As(err, &noUpErr) {
		t.Fatalf("Expected no error, got %s", err)
	}

}
