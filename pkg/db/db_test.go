package binmandb

import (
	"bytes"
	"testing"
)

func TestParseKey(t *testing.T) {

	basic := make(map[string][]byte)
	basicString := "mykey"
	basic[basicString] = GetBytes(basicString)

	complex := make(map[string][]byte)
	complexString := "mykey/mysubkey/mysubsubkey"
	complex["mykey"] = GetBytes("mykey")
	complex["mysubkey"] = GetBytes("mysubkey")
	complex["mysubsubkey"] = GetBytes("mysubsubkey")

	var tests = []struct {
		caseName  string
		data      map[string][]byte
		keyString string
	}{
		{caseName: "basic", data: basic, keyString: basicString},
		{caseName: "complex", data: complex, keyString: complexString},
	}

	for _, test := range tests {
		byteSlice := ParseKey(test.keyString)
		for _, b := range byteSlice {
			if !bytes.Equal(test.data[string(b)], b) {
				t.Fatalf("%s failed: got %s expected %s", test.caseName, test.data[string(b)], b)
			}
		}
	}
}
