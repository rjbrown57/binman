package binman

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/rjbrown57/binman/pkg/constants"
	binmandb "github.com/rjbrown57/binman/pkg/db"
	"github.com/rjbrown57/binman/pkg/downloader"
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

	dbChan := make(chan binmandb.DbMsg)
	dlChan := make(chan downloader.DlMsg)

	githubSource := Source{Name: "github.com", URL: constants.DefaultGHBaseURL, Apitype: "github", Tokenvar: "none"}
	gitlabSource := Source{Name: "gitlab.com", URL: constants.DefaultGLBaseURL, Apitype: "gitlab", Tokenvar: "none"}

	relWithOutPublish := BinmanRelease{
		Repo:         "rjbrown57/binman",
		QueryType:    "release",
		source:       &githubSource,
		dbChan:       dbChan,
		downloadChan: dlChan,
	}

	relWithPublish := BinmanRelease{
		Repo:         "rjbrown57/binman",
		PublishPath:  "binman",
		QueryType:    "release",
		source:       &githubSource,
		dbChan:       dbChan,
		downloadChan: dlChan,
	}

	relQueryByTag := BinmanRelease{
		Repo:         "rjbrown57/binman",
		PublishPath:  "binman",
		QueryType:    "releasebytag",
		source:       &githubSource,
		dbChan:       dbChan,
		downloadChan: dlChan,
	}

	relPostOnly := BinmanRelease{
		Repo:         "rjbrown57/binman",
		PublishPath:  "binman",
		QueryType:    "release",
		PostOnly:     true,
		source:       &githubSource,
		dbChan:       dbChan,
		downloadChan: dlChan,
	}

	relExternalUrl := BinmanRelease{
		Repo:         "rjbrown57/binman",
		QueryType:    "release",
		PostOnly:     false,
		ExternalUrl:  "https://avaluehere.com",
		source:       &githubSource,
		dbChan:       dbChan,
		downloadChan: dlChan,
	}

	relGLBasic := BinmanRelease{
		Repo:         "rjbrown57/binman",
		QueryType:    "release",
		source:       &gitlabSource,
		dbChan:       dbChan,
		downloadChan: dlChan,
	}

	var tests = []struct {
		name            string
		ReturnedActions []Action
		ExpectedActions []string
	}{
		{
			"relwithoutpublish",
			relWithOutPublish.setPreActions("/tmp/", "/tmp/"),
			[]string{"*binman.GetGHReleaseAction", "*binman.ReleaseStatusAction", "*binman.SetUrlAction", "*binman.SetArtifactPathAction", "*binman.SetPostActions"},
		},
		{
			// this release has a preset publish path this means it's a binman get and we don't need to use releasestatusaction
			"relWithPublish",
			relWithPublish.setPreActions("/tmp/", "/tmp/"),
			[]string{"*binman.GetGHReleaseAction", "*binman.SetUrlAction", "*binman.SetArtifactPathAction", "*binman.SetPostActions"},
		},
		{
			"relQueryByTag",
			relQueryByTag.setPreActions("/tmp/", "/tmp/"),
			[]string{"*binman.GetGHReleaseAction", "*binman.SetUrlAction", "*binman.SetArtifactPathAction", "*binman.SetPostActions"},
		},
		{
			"relPostOnly",
			relPostOnly.setPreActions("/tmp/", "/tmp/"),
			[]string{"*binman.GetGHReleaseAction", "*binman.SetArtifactPathAction", "*binman.SetPostActions"},
		},
		{
			"relExternalUrl",
			relExternalUrl.setPreActions("/tmp/", "/tmp/"),
			[]string{"*binman.GetGHReleaseAction", "*binman.ReleaseStatusAction", "*binman.SetUrlAction", "*binman.SetArtifactPathAction", "*binman.SetPostActions"},
		},
		{
			"relGLBasic",
			relGLBasic.setPreActions("/tmp/", "/tmp"),
			[]string{"*binman.GetGLReleaseAction", "*binman.ReleaseStatusAction", "*binman.SetUrlAction", "*binman.SetArtifactPathAction", "*binman.SetPostActions"},
		},
	}

	for _, test := range tests {
		t.Logf("Case %s", test.name)
		t.Logf("returned actions == %s", test.ReturnedActions)
		for k := range test.ReturnedActions {
			if reflect.TypeOf(test.ReturnedActions[k]).String() != test.ExpectedActions[k] {
				t.Fatalf("test %s - Expected %s, got %s", test.name, reflect.TypeOf(test.ReturnedActions[k]).String(), test.ExpectedActions[k])
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
		PublishPath:  "binman",
		artifactPath: "path",
		UpxConfig:    testUpxConfig,
	}

	relWithUpxandPostCommands := BinmanRelease{
		Repo:         "rjbrown57/binman",
		PublishPath:  "binman",
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
			[]string{"*binman.LinkFileAction", "*binman.UpdateDbAction", "*binman.EndWorkAction"},
		},
	}

	for _, test := range tests {
		fmt.Printf("Testing %s\n", test.name)
		for k := range test.ReturnedActions {
			if reflect.TypeOf(test.ReturnedActions[k]).String() != test.ExpectedActions[k] {
				t.Fatalf("Expected %s, got %s", test.ExpectedActions[k], reflect.TypeOf(test.ReturnedActions[k]).String())
			}
		}
	}
}
