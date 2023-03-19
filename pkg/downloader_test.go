package binman

import (
	"fmt"
	"sync"
	"testing"
)

func TestGetDownloader(t *testing.T) {

	var rWg sync.WaitGroup

	badDl := dlMsg{
		url:         "https://github.com/rjbrown57/binman/releases/download/vX.X.X/binman_linux_amd64",
		filepath:    "/tmp/path1",
		confirmChan: make(chan error, 1),
		wg:          &rWg,
	}

	goodDl := dlMsg{
		url:         "https://github.com/rjbrown57/binman/releases/download/v0.6.0/binman_linux_amd64",
		filepath:    "/tmp/path1",
		confirmChan: make(chan error, 1),
		wg:          &rWg,
	}

	var tests = []struct {
		testMsg dlMsg
		err     error
	}{
		{badDl, fmt.Errorf("failed to download from https://github.com/rjbrown57/binman/releases/download/vX.X.X/binman_linux_amd64 - <nil> , 404")},
		{goodDl, nil},
	}

	go getDownloader(1)

	for _, test := range tests {
		rWg.Add(1)
		downloadChan <- test.testMsg
		rWg.Wait()
		close(test.testMsg.confirmChan)

		err := <-test.testMsg.confirmChan
		if err != nil && err.Error() != test.err.Error() {
			t.Fatalf("got %v - expected %v\n", err, test.err)
		}
	}
}
