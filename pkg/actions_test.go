package binman

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-github/v48/github"
)

func TestSetPreActions(t *testing.T) {

	relWithOutPublish := BinmanRelease{
		Repo: "rjbrown57/binman",
	}

	relWithPublish := BinmanRelease{
		Repo:        "rjbrown57/binman",
		publishPath: "binman",
	}

	var tests = []struct {
		ReturnedActions []Action
		ExpectedActions []string
	}{
		{
			relWithOutPublish.setPreActions(github.NewClient(nil), "/tmp/"),
			[]string{"*binman.GetGHReleaseAction", "*binman.ReleaseStatusAction", "*binman.SetUrlAction", "*binman.SetArtifactPathAction"},
		},
		{
			relWithPublish.setPreActions(github.NewClient(nil), "/tmp/"),
			[]string{"*binman.GetGHReleaseAction", "*binman.SetUrlAction", "*binman.SetArtifactPathAction"},
		},
	}

	for _, test := range tests {
		for k := range test.ReturnedActions {
			if reflect.TypeOf(test.ReturnedActions[k]).String() != test.ExpectedActions[k] {
				t.Fatalf("Expected %s, got %s", reflect.TypeOf(test.ReturnedActions[k]).String(), test.ExpectedActions[k])
			}
		}
	}
}

func TestSetPostActions(t *testing.T) {

	testUpxConfig := UpxConfig{
		Enabled: "true",
		Args:    []string{"-k", "-v"},
	}

	tp := PostCommand{
		Command: "echo",
		Args:    []string{"arg1", "arg2"},
	}

	testPostCommands := []PostCommand{tp, tp}

	relDlOnly := BinmanRelease{
		Repo:         "rjbrown57/binman",
		DownloadOnly: true,
	}

	relBase := BinmanRelease{
		Repo: "rjbrown57/binman",
	}

	// A release with an external url that is a tar/tgz/zip
	relWithTar := BinmanRelease{
		Repo:     "rjbrown57/binman",
		filepath: "extractbinman.tar.gz",
	}

	relWithZip := BinmanRelease{
		Repo:     "rjbrown57/binman",
		filepath: "extractbinman.zip",
	}

	relWithUpx := BinmanRelease{
		Repo:         "rjbrown57/binman",
		publishPath:  "binman",
		artifactPath: "path",
		UpxConfig:    testUpxConfig,
	}

	relWithUpxandPostCommands := BinmanRelease{
		Repo:         "rjbrown57/binman",
		publishPath:  "binman",
		PostCommands: testPostCommands,
	}

	var tests = []struct {
		name            string
		ReturnedActions []Action
		ExpectedActions []string
	}{
		{
			"downloadOnly",
			relDlOnly.setPostActions(),
			[]string{"*binman.DownloadAction"},
		},
		{
			"basic",
			relBase.setPostActions(),
			[]string{"*binman.DownloadAction", "*binman.FindTargetAction", "*binman.MakeExecuteableAction", "*binman.LinkFileAction", "*binman.WriteRelNotesAction"},
		},
		{
			"tar",
			relWithTar.setPostActions(),
			[]string{"*binman.DownloadAction", "*binman.ExtractAction", "*binman.FindTargetAction", "*binman.MakeExecuteableAction", "*binman.LinkFileAction", "*binman.WriteRelNotesAction"},
		},
		{
			"zip",
			relWithZip.setPostActions(),
			[]string{"*binman.DownloadAction", "*binman.ExtractAction", "*binman.FindTargetAction", "*binman.MakeExecuteableAction", "*binman.LinkFileAction", "*binman.WriteRelNotesAction"},
		},
		{
			"basicupx",
			relWithUpx.setPostActions(),
			[]string{"*binman.DownloadAction", "*binman.FindTargetAction", "*binman.MakeExecuteableAction", "*binman.LinkFileAction", "*binman.WriteRelNotesAction", "*binman.OsCommandAction"},
		},
		{
			"basicmultiplepostcommands",
			relWithUpxandPostCommands.setPostActions(),
			[]string{"*binman.DownloadAction", "*binman.FindTargetAction", "*binman.MakeExecuteableAction", "*binman.LinkFileAction", "*binman.WriteRelNotesAction", "*binman.OsCommandAction", "*binman.OsCommandAction", "*binman.OsCommandAction"},
		},
	}

	for _, test := range tests {
		fmt.Printf("Testing %s\n", test.name)
		for k := range test.ReturnedActions {
			if reflect.TypeOf(test.ReturnedActions[k]).String() != test.ExpectedActions[k] {
				t.Fatalf("Expected %s, got %s", reflect.TypeOf(test.ReturnedActions[k]).String(), test.ExpectedActions[k])
			}
		}
	}
}
