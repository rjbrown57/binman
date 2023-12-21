package binman

import (
	"fmt"
	"os"
	"testing"
)

func TestCreateDirectory(t *testing.T) {
	d := fmt.Sprintf("%s/%s", os.TempDir(), "createdtestdir")
	CreateDirectory(d)
	defer os.RemoveAll(d)
	di, err := os.Stat(d)
	if err != nil {
		t.Fatalf("Issue getting directory info - %s", err)
	}

	if !di.IsDir() {
		t.Fatalf("Expected directory isDir for %s is false", d)
	}

}
func TestFindfType(t *testing.T) {
	var tests = []struct {
		testingString string
		exected       string
	}{
		{"myfile.tar.gz", "tar"},
		{"myfile.tgz", "tar"},
		{"myfile.zip", "zip"},
		{"myfile", "default"},
		{"myfile.ending", "default"},
	}

	for _, test := range tests {

		if retval := findfType(test.testingString); retval != test.exected {
			t.Fatalf("For string %s Excpected %s got %s", test.testingString, test.exected, retval)
		}

	}
}

func TestWriteStringtoFile(t *testing.T) {

	var testString string = "test-test-test"

	d, err := os.MkdirTemp(os.TempDir(), "binm")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	writePath := fmt.Sprintf("%s/testString", d)

	WriteStringtoFile(writePath, testString)
	if err != nil {
		t.Fatalf("failed to write test config to %s", writePath)
	}

	testBytes, err := os.ReadFile(writePath)
	if err != nil {
		t.Fatalf("failed to read test file at  %s", writePath)
	}

	if string(testBytes) != testString {
		t.Fatalf("Expected %s got  %s", string(testBytes), testString)
	}

}

func TestCopyFile(t *testing.T) {

	var testString string = "test-test-test"

	d, err := os.MkdirTemp(os.TempDir(), "binm")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	writePath := fmt.Sprintf("%s/testString", d)

	WriteStringtoFile(writePath, testString)
	if err != nil {
		t.Fatalf("failed to write test config to %s", writePath)
	}

	testBytes, err := os.ReadFile(writePath)
	if err != nil {
		t.Fatalf("failed to read test file at  %s", writePath)
	}

	copyTarget := fmt.Sprintf("%s/copyTarget", d)

	err = CopyFile(writePath, copyTarget)
	if err != nil {
		t.Fatalf("failed to copy %s to %s", writePath, copyTarget)
	}

	copyBytes, err := os.ReadFile(copyTarget)
	if err != nil {
		t.Fatalf("failed to read test file at  %s", copyTarget)
	}

	if string(testBytes) != string(copyBytes) {
		t.Fatalf("Expected %s got  %s", string(testBytes), testString)
	}

}

func TestCreateLink(t *testing.T) {
	var testString string = "test-test-test"

	d, err := os.MkdirTemp(os.TempDir(), "binm")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	writePath := fmt.Sprintf("%s/testString", d)

	WriteStringtoFile(writePath, testString)
	if err != nil {
		t.Fatalf("failed to write test config to %s", writePath)
	}

	linkPath := fmt.Sprintf("%s/linkFile", d)

	createLink(writePath, linkPath)
	s, err := os.Readlink(linkPath)
	if err == nil {
		if s != writePath {
			t.Fatalf("%s != %s", s, writePath)
		}

	} else {
		t.Fatalf("Unable to read link at %s", linkPath)
	}

	// One more time to test updating
	createLink(writePath, linkPath)
	s, err = os.Readlink(linkPath)
	if err == nil {
		if s != writePath {
			t.Fatalf("%s != %s", s, writePath)
		}

	} else {
		t.Fatalf("Unable to read link at %s", linkPath)
	}

}

func TestMakeExecutable(t *testing.T) {
	var testString string = "test-test-test"

	d, err := os.MkdirTemp(os.TempDir(), "binm")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	writePath := fmt.Sprintf("%s/testString", d)

	WriteStringtoFile(writePath, testString)
	if err != nil {
		t.Fatalf("failed to write test config to %s", writePath)
	}

	err = MakeExecuteable(writePath)
	if err != nil {
		t.Fatalf("MakeExecutable failed with %s", err)
	}

	f, err := os.Stat(writePath)
	if err != nil {
		t.Fatalf("Unable to read %s", err)
	}

	if mode := f.Mode(); mode&os.ModePerm == 755 {
		t.Fatalf("Permissions for %s are %o not 0755", writePath, mode)
	}
}

// createTestData is a helper function to create test cases that should never be returned. It should be passed the "successful" test string
func createTestData(passCase string) map[string]string {

	var bogusString = "what.ami"
	var wrongOs = "file_other_os.zip"
	var wrongEnding = "file_linux_amd64.wrongending"

	assets := map[string]string{
		bogusString: bogusString,
		wrongOs:     wrongOs,
		wrongEnding: wrongEnding,
		passCase:    passCase,
	}

	return assets
}

// TestSelectAsset will test each passing asset type
func TestSelectAsset(t *testing.T) {

	var passCases = []string{
		"file_linux_amd64.zip",    // a zip
		"file_linux_amd64.tar",    // tar form 1
		"file_linux_amd64.tar.gz", // tar form 2
		"file_linux_amd64.tgz",    // tar form 3
		"file_linux_amd64.exe",    // an exe
		"file_linux_amd64",        // a binary
		"file_0.0.0_linux_amd64",  // possible match 1
		"file_v0.0.0_linux_amd64", // possible match 1
	}

	for _, testString := range passCases {
		name, _ := selectAsset("linux", "amd64", "v0.0.0", "file", createTestData(testString))
		if name != testString {
			t.Fatalf("SelectAsset test failed. %s does not match %s", name, testString)
		}
	}

	// find nothing
	var nilString = "file_linux_amd64.nil"
	name, _ := selectAsset("linux", "amd64", "v1.0.0", "file", createTestData(nilString))

	if name != "" {
		t.Fatalf("SelectAsset nil test failed. Name = %s and should be empty!", name)
	}
}
