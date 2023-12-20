package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	log "github.com/rjbrown57/binman/pkg/logging"
)

// dlMsg is used to communicate with downloader pool
type DlMsg struct {
	Url         string
	Filepath    string
	Wg          *sync.WaitGroup
	ConfirmChan chan error
}

func GetDownloader(downloadChan chan DlMsg, id int) {
	log.Tracef("Downloader %d started", id)
	for msg := range downloadChan {
		log.Debugf("downloader %d is handling %s\n", id, msg.Url)
		err := DownloadFile(msg.Url, msg.Filepath)
		msg.ConfirmChan <- err
		msg.Wg.Done()
	}
	log.Tracef("Downloader %d finished", id)

}

func DownloadFile(url string, path string) error {
	log.Debugf("Downloading %s", url)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode > 200 {
		return fmt.Errorf("failed to download from %s - %v , %d", url, err, resp.StatusCode)
	}

	defer resp.Body.Close()

	out, err := os.Create(filepath.Clean(path))
	if err != nil {
		return err
	}

	defer out.Close()

	_, err = io.Copy(io.MultiWriter(out), resp.Body)
	if err != nil {
		return err
	}

	log.Debugf("Download %s complete", url)
	return nil
}
