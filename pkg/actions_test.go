package binman

import (
	"fmt"
	"reflect"
	"testing"
)

func TestRunActions(t *testing.T) {

	relBase := BinmanRelease{
		Repo: "rjbrown57/binman",
	}

	relBase.actions = []Action{relBase.AddEndWorkAction()}

	if relBase.runActions(); relBase.actions != nil {
		t.Fatalf("Test of action array should == nil")
	}
}

func TestSetPreActions(t *testing.T) {

	s := Source{Name: "github.com", URL: defaultGHBaseURL, Apitype: "github", Tokenvar: "none"}

	relWithOutPublish := BinmanRelease{
		Repo:      "rjbrown57/binman",
		QueryType: "release",
		source:    &s,
	}

	relWithPublish := BinmanRelease{
		Repo:        "rjbrown57/binman",
		publishPath: "binman",
		QueryType:   "release",
		source:      &s,
	}

	relQueryByTag := BinmanRelease{
		Repo:        "rjbrown57/binman",
		publishPath: "binman",
		QueryType:   "releasebytag",
		source:      &s,
	}

	relPostOnly := BinmanRelease{
		Repo:        "rjbrown57/binman",
		publishPath: "binman",
		QueryType:   "release",
		PostOnly:    true,
		source:      &s,
	}

	relExternalUrl := BinmanRelease{
		Repo:        "rjbrown57/binman",
		publishPath: "binman",
		QueryType:   "release",
		PostOnly:    false,
		ExternalUrl: "https://avaluehere.com",
		source:      &s,
	}

	var tests = []struct {
		ReturnedActions []Action
		ExpectedActions []string
	}{
		{
			relWithOutPublish.setPreActions("/tmp/"),
			[]string{"*binman.GetGHReleaseAction", "*binman.ReleaseStatusAction", "*binman.SetArtifactPathAction", "*binman.SetPostActions"},
		},
		{
			relWithPublish.setPreActions("/tmp/"),
			[]string{"*binman.GetGHReleaseAction", "*binman.SetArtifactPathAction", "*binman.SetPostActions"},
		},
		{
			relQueryByTag.setPreActions("/tmp/"),
			[]string{"*binman.GetGHReleaseAction", "*binman.SetArtifactPathAction", "*binman.SetPostActions"},
		},
		{
			relPostOnly.setPreActions("/tmp/"),
			[]string{"*binman.GetGHReleaseAction", "*binman.SetArtifactPathAction", "*binman.SetPostActions"},
		},
		{
			relExternalUrl.setPreActions("/tmp/"),
			[]string{"*binman.GetGHReleaseAction", "*binman.SetUrlAction", "*binman.SetArtifactPathAction", "*binman.SetPostActions"},
		},
	}

	for _, test := range tests {
		t.Logf("returned actions == %s", test.ReturnedActions)
		for k := range test.ReturnedActions {
			if reflect.TypeOf(test.ReturnedActions[k]).String() != test.ExpectedActions[k] {
				t.Fatalf("Expected %s, got %s", reflect.TypeOf(test.ReturnedActions[k]).String(), test.ExpectedActions[k])
			}
		}
	}
}

func TestSetPostActions(t *testing.T) {

	relDlOnly := BinmanRelease{
		Repo:         "rjbrown57/binman",
		DownloadOnly: true,
	}

	relPostOnly := BinmanRelease{
		Repo:     "rjbrown57/binman",
		PostOnly: true,
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

	var tests = []struct {
		name            string
		ReturnedActions []Action
		ExpectedActions []string
	}{
		{
			"downloadOnly",
			relDlOnly.setPostActions(),
			[]string{"*binman.DownloadAction", "*binman.SetOsActions"},
		},
		{
			"postOnly",
			relPostOnly.setPostActions(),
			[]string{"*binman.SetOsActions"},
		},
		{
			"basic",
			relBase.setPostActions(),
			[]string{"*binman.DownloadAction", "*binman.FindTargetAction", "*binman.MakeExecuteableAction", "*binman.WriteRelNotesAction", "*binman.SetOsActions"},
		},
		{
			"tar",
			relWithTar.setPostActions(),
			[]string{"*binman.DownloadAction", "*binman.ExtractAction", "*binman.FindTargetAction", "*binman.MakeExecuteableAction", "*binman.WriteRelNotesAction", "*binman.SetOsActions"},
		},
		{
			"zip",
			relWithZip.setPostActions(),
			[]string{"*binman.DownloadAction", "*binman.ExtractAction", "*binman.FindTargetAction", "*binman.MakeExecuteableAction", "*binman.WriteRelNotesAction", "*binman.SetOsActions"},
		},
	}

	for _, test := range tests {
		fmt.Printf("Testing %s\n, %T", test.name, test.ReturnedActions[0])
		for k := range test.ReturnedActions {
			if reflect.TypeOf(test.ReturnedActions[k]).String() != test.ExpectedActions[k] {
				t.Fatalf("Expected %s, got %s", reflect.TypeOf(test.ReturnedActions[k]).String(), test.ExpectedActions[k])
			}
		}
	}
}

func TestSetOsActions(t *testing.T) {

	relBase := BinmanRelease{
		Repo: "rjbrown57/binman",
	}

	testUpxConfig := UpxConfig{
		Enabled: "true",
		Args:    []string{"-k", "-v"},
	}

	tp := PostCommand{
		Command: "echo",
		Args:    []string{"arg1", "arg2"},
	}

	testPostCommands := []PostCommand{tp, tp}

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
			"none",
			relBase.setOsCommands(),
			[]string{"*binman.SetFinalActions"},
		},
		{
			"basicupx",
			relWithUpx.setOsCommands(),
			[]string{"*binman.OsCommandAction", "*binman.SetFinalActions"},
		},
		{
			"basicmultiplepostcommands",
			relWithUpxandPostCommands.setOsCommands(),
			[]string{"*binman.OsCommandAction", "*binman.OsCommandAction", "*binman.SetFinalActions"},
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

func TestSetFinalActions(t *testing.T) {

	relBase := BinmanRelease{
		Repo: "rjbrown57/binman",
	}

	var tests = []struct {
		name            string
		ReturnedActions []Action
		ExpectedActions []string
	}{
		{
			"basic",
			relBase.setFinalActions(),
			[]string{"*binman.LinkFileAction", "*binman.EndWorkAction"},
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
