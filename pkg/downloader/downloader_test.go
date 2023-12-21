package downloader

import (
	"fmt"
	"os"
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
