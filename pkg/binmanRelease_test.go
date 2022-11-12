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
		ArtifactPath: "binman",
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

func TestSetArtifactPath(t *testing.T) {

	// A release where we have set a specific target with releasefilename
	relWithRelFilename := BinmanRelease{
		Repo:            "rjbrown57/binman",
		ArtifactPath:    "binman",
		PublishPath:     "/tmp/",
		ReleaseFileName: "binman",
		Project:         "binmanz",
		LinkName:        "",
		Os:              "linux",
		Arch:            "amd64",
	}

	// A release where the asset is a tar/tgz/zip and we have specified a path internally
	relWithExtractFilename := BinmanRelease{
		Repo:            "rjbrown57/binman",
		ArtifactPath:    "binman",
		PublishPath:     "/tmp/",
		ExtractFileName: "extractbinman",
		Project:         "binman",
		LinkName:        "",
		Os:              "linux",
		Arch:            "amd64",
	}

	// A release with an external url that is a binary
	relWithUrlNonTar := BinmanRelease{
		Repo:         "rjbrown57/binman",
		ArtifactPath: "binman",
		PublishPath:  "/tmp/",
		ExternalUrl:  "extractbinman",
		Project:      "binman",
		LinkName:     "",
		Os:           "linux",
		Arch:         "amd64",
	}

	// A release with an external url that is a tar/tgz/zip
	relWithUrlTar := BinmanRelease{
		Repo:         "rjbrown57/binman",
		ArtifactPath: "binman",
		PublishPath:  "/tmp/",
		ExternalUrl:  "extractbinman.tar.gz",
		Project:      "binman",
		LinkName:     "",
		Os:           "linux",
		Arch:         "amd64",
	}

	// A basic release we use multiple times
	relBasic := BinmanRelease{
		Repo:         "rjbrown57/binman",
		ArtifactPath: "binman",
		PublishPath:  "/tmp/",
		LinkName:     "",
		Project:      "binman",
		Os:           "linux",
		Arch:         "amd64",
	}

	// A release with the link name set
	relWithLinkName := BinmanRelease{
		Repo:         "rjbrown57/binman",
		ArtifactPath: "binman",
		PublishPath:  "/tmp/",
		Project:      "binman",
		LinkName:     "none",
		Os:           "linux",
		Arch:         "amd64",
	}

	var tests = []struct {
		rel                 BinmanRelease
		expectedLinkPath    string
		exectedArtifactPath string
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
		if test.rel.LinkPath != test.expectedLinkPath {
			t.Fatalf("Link Path expected %s, got %s", test.expectedLinkPath, test.rel.LinkPath)
		}

		if test.rel.ArtifactPath != test.exectedArtifactPath {
			t.Fatalf("Artifact Path expected %s, got %s", test.exectedArtifactPath, test.rel.ArtifactPath)
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
		GithubData: &ghData,
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
