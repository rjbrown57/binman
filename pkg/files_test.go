package binman

import (
	"fmt"
	"os"
	"testing"
)

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
