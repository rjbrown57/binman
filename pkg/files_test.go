package binman

import (
	"fmt"
	"os"
	"strings"
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

func TestDownloadFile(t *testing.T) {
	var url string = "https://raw.githubusercontent.com/rjbrown57/binman/main/Readme.md"
	d, err := os.MkdirTemp(os.TempDir(), "binm")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	writePath := fmt.Sprintf("%s/testString", d)
	err = DownloadFile(url, writePath)
	if err != nil {
		t.Fatalf("Issue downloading %s - %s", url, err)
	}

	f, err := os.ReadFile(writePath)
	if err != nil {
		t.Fatalf("Issue Reading %s - %s", writePath, err)
	}

	if !strings.Contains(string(f), "binman") {
		t.Fatalf("Exected string 'binman' not found in %s", writePath)
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

	if mode := f.Mode(); mode&os.ModePerm == 750 {
		t.Fatalf("Permissions for %s are %o not 0750", writePath, mode)
	}
}
