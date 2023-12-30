package binman

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteRelNotesAction(t *testing.T) {

	d, err := os.MkdirTemp(os.TempDir(), "binmwrn")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	// Create a dummy asset to detect in a subdir of the temp
	var actions []Action
	var version string = "v0.0.0"
	var bodyContent string = "test-test-test"

	rel := BinmanRelease{
		Repo:        "rjbrown57/binman",
		PublishPath: d,
		Version:     version,
		relNotes:    bodyContent,
	}

	actions = append(actions, rel.AddWriteRelNotesAction())

	if err = actions[0].execute(); err != nil {
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

	var actions []Action
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
		artifactPath: filePath,
		linkPath:     linkPath,
	}

	WriteStringtoFile(rel.artifactPath, content)

	// Add the link task twice, to confirm link is updated successfully
	actions = append(actions, rel.AddLinkFileAction())
	actions = append(actions, rel.AddLinkFileAction())

	for _, task := range actions {
		if err = task.execute(); err != nil {
			t.Fatal("Unable to create release link")
		}

		if f, err := os.Stat(rel.linkPath); err == nil {
			if f.Name() != linkname {
				t.Fatalf("Expected link name %s got %s", linkname, f.Name())
			}

			// Read the written release notes
			contentBytes, err := os.ReadFile(rel.linkPath)
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

	var actions []Action

	const content string = "stringcontent"
	testMode := os.FileMode(int(0755))

	d, err := os.MkdirTemp(os.TempDir(), "binmwrn")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	filePath := filepath.Join(d, "testfile")

	defer os.RemoveAll(d)

	rel := BinmanRelease{
		Repo:         "rjbrown57/binman",
		artifactPath: filePath,
	}

	WriteStringtoFile(rel.artifactPath, content)

	actions = append(actions, rel.AddMakeExecuteableAction())

	if err = actions[0].execute(); err != nil {
		t.Fatal("Unable to create make file executable")
	}

	if f, err := os.Stat(rel.artifactPath); err == nil {
		if f.Mode().Perm() != testMode.Perm() {
			t.Fatalf("Expected %o got %o", testMode, f.Mode())
		}
	} else {
		t.Fatal("Test file was not properly created")
	}
}

// This test will only function properly on linux
func TestOsCommandAction(t *testing.T) {

	var actions []Action

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

	rel := BinmanRelease{
		Repo:         "rjbrown57/binman",
		PostCommands: coms,
		Version:      version,
	}

	actions = append(actions, rel.AddOsCommandAction(0))
	actions = append(actions, rel.AddOsCommandAction(1))
	for i, task := range actions {
		if err := task.execute(); err != nil {
			t.Fatalf("Unable to run post command %d", i)
		}
	}
}
