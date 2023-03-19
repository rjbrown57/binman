package binman

import (
	"sync"

	log "github.com/rjbrown57/binman/pkg/logging"
)

var downloadChan = make(chan dlMsg)

// dlMsg is used to communicate with downloader pool
type dlMsg struct {
	url         string
	filepath    string
	wg          *sync.WaitGroup
	confirmChan chan error
}

func getDownloader(id int) {
	for msg := range downloadChan {
		log.Debugf("downloader %d is handling %s\n", id, msg.url)
		err := DownloadFile(msg.url, msg.filepath)
		msg.confirmChan <- err
		msg.wg.Done()
	}
}
