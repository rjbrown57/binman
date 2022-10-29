package binman

import (
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
