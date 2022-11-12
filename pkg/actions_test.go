package binman

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-github/v48/github"
)

func TestWriteRelNotesAction(t *testing.T) {

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

	rel.tasks = append(rel.tasks, rel.AddWriteRelNotesAction())

	if err = rel.tasks[0].execute(); err != nil {
		t.Fatal("Unable to write release notes")
	}

	// Read the written release notes
	notesBytes, err := os.ReadFile(filepath.Join(rel.PublishPath, "releaseNotes.txt"))
	if err != nil {
		t.Fatal("Unable to read written release notes")
	}

	if string(notesBytes) != bodyContent {
		t.Fatalf("Want %s, got %s", bodyContent, notesBytes)
	}

}

func TestLinkFileAction(t *testing.T) {

	const content string = "stringcontent"
	const linkname string = "test"

	d, err := os.MkdirTemp(os.TempDir(), "binmwrn")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	filePath := filepath.Join(d, "testfile")
	linkPath := filepath.Join(d, linkname)

	defer os.RemoveAll(d)

	rel := BinmanRelease{
		Repo:         "rjbrown57/binman",
		ArtifactPath: filePath,
		LinkPath:     linkPath,
	}

	writeStringtoFile(rel.ArtifactPath, content)

	// Add the link task twice, to confirm link is updated successfully
	rel.tasks = append(rel.tasks, rel.AddLinkFileAction())
	rel.tasks = append(rel.tasks, rel.AddLinkFileAction())

	for _, task := range rel.tasks {
		if err = task.execute(); err != nil {
			t.Fatal("Unable to create release link")
		}

		if f, err := os.Stat(rel.LinkPath); err == nil {
			if f.Name() != linkname {
				t.Fatalf("Expected link name %s got %s", linkname, f.Name())
			}

			// Read the written release notes
			contentBytes, err := os.ReadFile(rel.LinkPath)
			if err != nil {
				t.Fatal("Unable to read link contents")
			}

			if string(contentBytes) != content {
				t.Fatalf("Want %s, got %s", content, contentBytes)
			}
		} else {
			t.Fatal("Link was not created properly")
		}
	}

}

func TestMakeExecuteableAction(t *testing.T) {

	const content string = "stringcontent"
	testMode := os.FileMode(int(0750))

	d, err := os.MkdirTemp(os.TempDir(), "binmwrn")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	filePath := filepath.Join(d, "testfile")

	defer os.RemoveAll(d)

	rel := BinmanRelease{
		Repo:         "rjbrown57/binman",
		ArtifactPath: filePath,
	}

	writeStringtoFile(rel.ArtifactPath, content)

	rel.tasks = append(rel.tasks, rel.AddMakeExecuteableAction())

	if err = rel.tasks[0].execute(); err != nil {
		t.Fatal("Unable to create make file executable")
	}

	if f, err := os.Stat(rel.ArtifactPath); err == nil {
		if f.Mode().Perm() != testMode.Perm() {
			t.Fatalf("Expected %s got %s", "0750", f.Mode().String())
		}
	} else {
		t.Fatal("Test file was not properly created")
	}
}

// This test will only function properly on linux
func TestOsCommandAction(t *testing.T) {

	coms := []PostCommand{
		{
			Command: "echo",
			Args:    []string{"hooray!"},
		},
		{
			Command: "echo",
			Args:    []string{"Hooray2"},
		},
	}

	var version string = "v0.0.0"

	// Create a fake release
	ghData := github.RepositoryRelease{
		TagName: &version,
	}

	rel := BinmanRelease{
		Repo:         "rjbrown57/binman",
		PostCommands: coms,
		GithubData:   &ghData,
	}

	rel.tasks = append(rel.tasks, rel.AddOsCommandAction(0))
	rel.tasks = append(rel.tasks, rel.AddOsCommandAction(1))
	for i, task := range rel.tasks {
		if err := task.execute(); err != nil {
			t.Fatalf("Unable to run post command %d", i)
		}
	}
}
