package downloader

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestDownloadFile(t *testing.T) {
	var url string = "https://raw.githubusercontent.com/rjbrown57/binman/main/Readme.md"
	d, err := os.MkdirTemp(os.TempDir(), "binm")
	if err != nil {
		t.Fatalf("unable to make temp dir %s", d)
	}

	defer os.RemoveAll(d)

	writePath := fmt.Sprintf("%s/testString", d)
	dlMsg := DlMsg{Url: url, Filepath: writePath}

	err = dlMsg.DownloadFile()
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

func TestNewDlAuth(t *testing.T) {
	var tests = []struct {
		Expected *DlAuth
		Got      *DlAuth
		Token    string
		Header   string
	}{
		{Expected: nil, Token: "", Header: "Authorization"},
		{Expected: nil, Token: "asdf", Header: "Authorization"},
	}

	for _, test := range tests {
		test.Got = NewDlAuth(test.Token, test.Header)
		if test.Got != nil && test.Expected != nil {
			if reflect.DeepEqual(*test.Got, *test.Expected) {
				t.Fatalf("%s does not = %s", test.Got, test.Expected)
			}
		}

	}
}
