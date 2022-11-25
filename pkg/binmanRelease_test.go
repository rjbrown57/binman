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
	testRepo := fmt.Sprintf("%s/%s", rel.org, rel.project)
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
	WriteStringtoFile(afp, "test-test-test")
	if err != nil {
		t.Fatalf("unable to write string to file %s", afp)
	}

	// Create a fake release
	ghData := github.RepositoryRelease{
		TagName: &version,
	}

	rel := BinmanRelease{
		Repo:         "rjbrown57/binman",
		artifactPath: "binman",
		publishPath:  d,
		Os:           "linux",
		Arch:         "amd64",
		githubData:   &ghData,
	}

	rel.findTarget()

	// We should find our file in the subdir
	if rel.artifactPath != testFileName {
		t.Fatalf("Expected %s got %s", rel.artifactPath, afp)
	}
}

func TestSetartifactPath(t *testing.T) {

	// A release where we have set a specific target with releasefilename
	relWithRelFilename := BinmanRelease{
		Repo:            "rjbrown57/binman",
		artifactPath:    "binman",
		publishPath:     "/tmp/",
		ReleaseFileName: "binman",
		project:         "binmanz",
		LinkName:        "",
		Os:              "linux",
		Arch:            "amd64",
	}

	// A release where the asset is a tar/tgz/zip and we have specified a path internally
	relWithExtractFilename := BinmanRelease{
		Repo:            "rjbrown57/binman",
		artifactPath:    "binman",
		publishPath:     "/tmp/",
		ExtractFileName: "extractbinman",
		project:         "binman",
		LinkName:        "",
		Os:              "linux",
		Arch:            "amd64",
	}

	// A release with an external url that is a binary
	relWithUrlNonTar := BinmanRelease{
		Repo:         "rjbrown57/binman",
		artifactPath: "binman",
		publishPath:  "/tmp/",
		ExternalUrl:  "extractbinman",
		project:      "binman",
		LinkName:     "",
		Os:           "linux",
		Arch:         "amd64",
	}

	// A release with an external url that is a tar/tgz/zip
	relWithUrlTar := BinmanRelease{
		Repo:         "rjbrown57/binman",
		artifactPath: "binman",
		publishPath:  "/tmp/",
		ExternalUrl:  "extractbinman.tar.gz",
		project:      "binman",
		LinkName:     "",
		Os:           "linux",
		Arch:         "amd64",
	}

	// A basic release we use multiple times
	relBasic := BinmanRelease{
		Repo:         "rjbrown57/binman",
		artifactPath: "binman",
		publishPath:  "/tmp/",
		LinkName:     "",
		project:      "binman",
		Os:           "linux",
		Arch:         "amd64",
	}

	// A release with the link name set
	relWithLinkName := BinmanRelease{
		Repo:         "rjbrown57/binman",
		artifactPath: "binman",
		publishPath:  "/tmp/",
		project:      "binman",
		LinkName:     "none",
		Os:           "linux",
		Arch:         "amd64",
	}

	var tests = []struct {
		rel                 BinmanRelease
		expectedLinkPath    string
		exectedartifactPath string
		assetName           string
		releasePath         string
	}{
		{relWithRelFilename, "/tmp/binmanz", "/tmp/binman", "binman", "/tmp/"},
		{relWithUrlNonTar, "/tmp/binman", "/tmp/extractbinman", "testfile", "/tmp/"},
		{relWithUrlTar, "/tmp/binman", "/tmp/extractbinman.tar.gz", "testfile", "/tmp/"},
		{relWithExtractFilename, "/tmp/binman", "/tmp/extractbinman", "testfile", "/tmp/"},
		{relWithLinkName, "/tmp/none", "/tmp/test", "test", "/tmp/"},
		{relBasic, "/tmp/binman", "/tmp/binman", "myfile.tgz", "/tmp/"},
		{relBasic, "/tmp/binman", "/tmp/binman", "myfile.zip", "/tmp/"},
		{relBasic, "/tmp/binman", "/tmp/binman", "myfile.tar.gz", "/tmp/"},
		{relBasic, "/tmp/binman", "/tmp/testfile", "testfile", "/tmp/"},
	}

	for _, test := range tests {
		test.rel.setArtifactPath(test.releasePath, test.assetName)
		if test.rel.linkPath != test.expectedLinkPath {
			t.Fatalf("Link Path expected %s, got %s", test.expectedLinkPath, test.rel.linkPath)
		}

		if test.rel.artifactPath != test.exectedartifactPath {
			t.Fatalf("Artifact Path expected %s, got %s", test.exectedartifactPath, test.rel.artifactPath)
		}
	}
}

func TestGetDataMap(t *testing.T) {

	// Create a dummy asset to detect in a subdir of the temp
	var version string = "v0.0.0"
	var os string = "linux"
	var arch string = "amd64"

	// Create a fake release
	ghData := github.RepositoryRelease{
		TagName: &version,
	}

	rel := BinmanRelease{
		Os:         os,
		Arch:       arch,
		githubData: &ghData,
	}

	testdataMap := make(map[string]string)
	testdataMap["version"] = version
	testdataMap["os"] = os
	testdataMap["arch"] = arch

	m := rel.getDataMap()
	for k, v := range m {
		if testdataMap[k] != v {
			t.Fatalf("Expected %s got %s", testdataMap[k], v)
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
		{relKnown, KnownUrlMap[testRepo]},
	}

	for _, test := range tests {
		test.rel.knownUrlCheck()
		if test.rel.ExternalUrl != test.expectedurl {
			t.Fatalf("%s Expected %s got %s", test.rel.Repo, test.expectedurl, test.rel.ExternalUrl)
		}
	}
}
