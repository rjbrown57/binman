package binman

import (
	"fmt"
	"os"
	"testing"

	"github.com/rjbrown57/binman/pkg/constants"
)

func TestGetOr(t *testing.T) {

	relSlice := []BinmanRelease{
		{
			// A basic gitub release
			Repo: "rjbrown57/binman",
		},
		{
			// A basic nested project
			Repo: "mygroup/mysubgroup/myproject",
		},
	}

	for _, rel := range relSlice {
		rel.getOR()
		testRepo := fmt.Sprintf("%s/%s", rel.org, rel.project)
		if testRepo != rel.Repo {
			t.Fatalf("excpected %s : got %s", testRepo, rel.Repo)
		}
	}

}

func TestSetSource(t *testing.T) {

	var githubDefault = Source{Name: "github.com", URL: constants.DefaultGHBaseURL, Apitype: "github"}
	var gitlabDefault = Source{Name: "gitlab.com", URL: constants.DefaultGLBaseURL, Apitype: "gitlab"}

	sourceMap := make(map[string]*Source)
	sourceMap["github.com"] = &githubDefault
	sourceMap["gitlab.com"] = &gitlabDefault

	var tests = []struct {
		rel              BinmanRelease
		expectedSourceId string
		expectedSource   *Source
		expectedReponame string
	}{
		{
			rel:              BinmanRelease{Repo: "rjbrown57/binman"},
			expectedSourceId: "github.com",
			expectedSource:   sourceMap["github.com"],
			expectedReponame: "rjbrown57/binman",
		},
		{
			rel:              BinmanRelease{Repo: "rjbrown57/binman", SourceIdentifier: "gitlab.com"},
			expectedSourceId: "gitlab.com",
			expectedSource:   sourceMap["gitlab.com"],
			expectedReponame: "rjbrown57/binman",
		},
		{
			rel:              BinmanRelease{Repo: "github.com/rjbrown57/binman"},
			expectedSourceId: "github.com",
			expectedSource:   sourceMap["github.com"],
			expectedReponame: "rjbrown57/binman",
		},
		{
			rel:              BinmanRelease{Repo: "gitlab.com/rjbrown57/binman"},
			expectedSourceId: "gitlab.com",
			expectedSource:   sourceMap["gitlab.com"],
			expectedReponame: "rjbrown57/binman",
		},
	}

	for _, test := range tests {
		test.rel.SetSource(sourceMap)
		if test.expectedReponame != test.rel.Repo {
			t.Fatalf("excpected %s : got %s", test.expectedReponame, test.rel.Repo)
		}
		if test.expectedSource != test.rel.source {
			t.Fatalf("excpected %s : got %s", test.expectedSource, test.rel.source)
		}
		if test.expectedSourceId != test.rel.SourceIdentifier {
			t.Fatalf("excpected %s : got %s", test.expectedSourceId, test.rel.SourceIdentifier)
		}
	}

}

func prepTestDir(path string, executablefilematch string) error {
	err := WriteStringtoFile(fmt.Sprintf("%s/%s", path, "binman.tar.gz"), "test-test-test")
	if err != nil {
		return err
	}

	err = WriteStringtoFile(fmt.Sprintf("%s/%s", path, "test.zip"), "test-test-test")
	if err != nil {
		return err
	}

	err = WriteStringtoFile(executablefilematch, "test-test-test")
	if err != nil {
		return err
	}

	err = os.Chmod(fmt.Sprintf(executablefilematch), 0755)
	if err != nil {
		return err
	}

	return nil
}

func TestFindTarget(t *testing.T) {

	d, err := os.MkdirTemp(os.TempDir(), "binmft")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	// Create test file dir within temp dir
	td := fmt.Sprintf("%s/%s", d, "test")
	os.Mkdir(td, 0744)
	if err != nil {
		t.Fatalf("unable to make temp dir %s", td)
	}

	var executablefilematch string = fmt.Sprintf("%s/%s", td, "execmatch")

	// add test files
	err = prepTestDir(td, executablefilematch)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy asset to detect in a subdir of the temp
	var testFileName string = "binman"
	var version string = "v0.0.0"
	afp := fmt.Sprintf("%s/%s", td, testFileName)
	WriteStringtoFile(afp, "test-test-test")
	if err != nil {
		t.Fatalf("unable to write string to file %s", afp)
	}

	rel := BinmanRelease{
		Repo:         "rjbrown57/binman",
		ArtifactPath: "binman",
		PublishPath:  d,
		Os:           "linux",
		Arch:         "amd64",
		Version:      version,
	}

	rel.findTarget()

	// We should find our file in the subdir
	if afp != rel.ArtifactPath {
		t.Fatalf("Expected %s got %s", afp, rel.ArtifactPath)
	}

	err = os.Remove(afp)
	if err != nil {
		t.Fatal(err)
	}

	rel.findTarget()

	// we should detect the executable file
	if executablefilematch != rel.ArtifactPath {
		t.Fatalf("Expected %s got %s", executablefilematch, rel.ArtifactPath)
	}

	// Test we can detect .exe
	wfp := fmt.Sprintf("%s/%s.exe", td, testFileName)
	WriteStringtoFile(wfp, "test-test-test")
	rel.ArtifactPath = "binman"
	rel.Os = "windows"
	rel.findTarget()

	if wfp != rel.ArtifactPath {
		t.Fatalf("Expected %s got %s", wfp, rel.ArtifactPath)
	}
}

func TestSetartifactPath(t *testing.T) {

	var version string = "v0.0.0"

	// A release where we have set a specific target with releasefilename
	relWithRelFilename := BinmanRelease{
		Repo:            "rjbrown57/binman",
		ArtifactPath:    "binman",
		PublishPath:     "/tmp/",
		ReleaseFileName: "binman",
		project:         "binmanz",
		LinkName:        "",
		Os:              "linux",
		Arch:            "amd64",
		linkPath:        "path",
		assetName:       "binman",
		org:             "rjbrown57",
		Version:         version,
	}

	// A release where the asset is a tar/tgz/zip and we have specified a path internally
	relWithExtractFilename := BinmanRelease{
		Repo:            "rjbrown57/binman",
		ArtifactPath:    "binman",
		PublishPath:     "/tmp/",
		ExtractFileName: "extractbinman",
		project:         "binman",
		LinkName:        "",
		Os:              "linux",
		Arch:            "amd64",
		linkPath:        "path",
		assetName:       "binman",
		org:             "rjbrown57",
		Version:         version,
	}

	// A release with an external url that is a binary
	relWithUrlNonTar := BinmanRelease{
		Repo:         "rjbrown57/binman",
		ArtifactPath: "binman",
		PublishPath:  "/tmp/",
		ExternalUrl:  "extractbinman",
		project:      "binman",
		LinkName:     "",
		Os:           "linux",
		Arch:         "amd64",
		linkPath:     "path",
		assetName:    "binman",
		org:          "rjbrown57",
		Version:      version,
	}

	// A release with an external url that is a tar/tgz/zip
	relWithUrlTar := BinmanRelease{
		Repo:         "rjbrown57/binman",
		ArtifactPath: "binman",
		PublishPath:  "/tmp/",
		ExternalUrl:  "extractbinman.tar.gz",
		project:      "binman",
		LinkName:     "",
		Os:           "linux",
		Arch:         "amd64",
		linkPath:     "path",
		assetName:    "binman",
		org:          "rjbrown57",
		Version:      version,
	}

	// A basic release we use multiple times
	relBasic := BinmanRelease{
		Repo:         "rjbrown57/binman",
		ArtifactPath: "binman",
		PublishPath:  "/tmp/",
		LinkName:     "",
		project:      "binman",
		Os:           "linux",
		Arch:         "amd64",
		linkPath:     "path",
		assetName:    "binman",
		org:          "rjbrown57",
		Version:      version,
	}

	// A release with the link name set
	relWithLinkName := BinmanRelease{
		Repo:         "rjbrown57/binman",
		ArtifactPath: "binman",
		PublishPath:  "/tmp/",
		project:      "binman",
		LinkName:     "none",
		Os:           "linux",
		Arch:         "amd64",
		linkPath:     "path",
		assetName:    "binman",
		org:          "rjbrown57",
		Version:      version,
	}

	var tests = []struct {
		rel                 BinmanRelease
		expectedLinkPath    string
		exectedartifactPath string
		assetName           string
		releasePath         string
		binPath             string
	}{
		{relWithRelFilename, "/tmp/binmanz", "/tmp/binman", "binman", "/tmp/", "/tmp/"},
		{relWithUrlNonTar, "/tmp/binman", "/tmp/extractbinman", "testfile", "/tmp/", "/tmp/"},
		{relWithUrlTar, "/tmp/binman", "/tmp/extractbinman.tar.gz", "testfile", "/tmp/", "/tmp/"},
		{relWithExtractFilename, "/tmp/binman", "/tmp/extractbinman", "testfile", "/tmp/", "/tmp/"},
		{relWithLinkName, "/tmp/none", "/tmp/test", "test", "/tmp/", "/tmp/"},
		{relBasic, "/tmp/binman", "/tmp/binman", "myfile.tgz", "/tmp/", "/tmp/"},
		{relBasic, "/tmp/binman", "/tmp/binman", "myfile.zip", "/tmp/", "/tmp/"},
		{relBasic, "/tmp/binman", "/tmp/binman", "myfile.tar.gz", "/tmp/", "/tmp/"},
		{relBasic, "/tmp/binman", "/tmp/testfile", "testfile", "/tmp/", "/tmp/"},
	}

	for _, test := range tests {
		test.rel.setArtifactPath(test.releasePath, test.binPath, test.assetName)
		if test.rel.linkPath != test.expectedLinkPath {
			t.Fatalf("Link Path expected %s, got %s", test.expectedLinkPath, test.rel.linkPath)
		}

		if test.rel.ArtifactPath != test.exectedartifactPath {
			t.Fatalf("Artifact Path expected %s, got %s", test.exectedartifactPath, test.rel.ArtifactPath)
		}
	}
}

func TestGetDataMap(t *testing.T) {

	// Create a dummy asset to detect in a subdir of the temp
	var version string = "v0.0.0"
	var os string = "linux"
	var arch string = "amd64"

	rel := BinmanRelease{
		Repo:          "test/test",
		Os:            os,
		Arch:          arch,
		Version:       version,
		createdAtTime: int64(0),
		ArtifactPath:  "test",
		linkPath:      "test",
		PublishPath:   "test",
		assetName:     "test",
	}

	rel.getOR()

	testdataMap := make(map[string]interface{})
	testdataMap["version"] = version
	testdataMap["os"] = os
	testdataMap["arch"] = arch
	testdataMap["createdAt"] = int64(0)
	testdataMap["org"] = "test"
	testdataMap["repo"] = "test/test"
	testdataMap["project"] = "test"
	testdataMap["artifactPath"] = "test"
	testdataMap["publishPath"] = "test"
	testdataMap["linkPath"] = "test"
	testdataMap["assetName"] = "test"

	m := rel.getDataMap()
	for k, v := range m {
		if testdataMap[k] != v {
			t.Fatalf("Expected %s%s got %s", k, testdataMap[k], v)
		}
	}
}

func TestKnownUrlCheck(t *testing.T) {

	relUnknown := BinmanRelease{
		Repo: "rjbrown57/binman",
	}

	relKnown := BinmanRelease{
		Repo: "kubernetes/kubernetes",
	}

	var testRepo string = "kubernetes/kubernetes"

	var tests = []struct {
		rel         BinmanRelease
		expectedurl string
	}{
		{relUnknown, ""},
		{relKnown, constants.KnownUrlMap[testRepo]},
	}

	for _, test := range tests {
		test.rel.knownUrlCheck()
		if test.rel.ExternalUrl != test.expectedurl {
			t.Fatalf("%s Expected %s got %s", test.rel.Repo, test.expectedurl, test.rel.ExternalUrl)
		}
	}
}
