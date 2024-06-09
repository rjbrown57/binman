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
	DlAuth      *DlAuth
}

func (d *DlMsg) DownloadFile() error {
	log.Debugf("Downloading %s", d.Url)

	c := http.Client{}

	r, err := http.NewRequest(http.MethodGet, d.Url, nil)
	if err != nil {
		log.Debugf("%s", err)
	}

	if d.DlAuth != nil {
		r.Header.Set(d.DlAuth.Header, fmt.Sprintf("Bearer %s", d.DlAuth.Token))
	}

	resp, err := c.Do(r)
	if err != nil {
		log.Debugf("%+v %v", resp, err)
		return fmt.Errorf("failed to download from %s - %v", d.Url, err)
	}

	if resp.StatusCode > 200 {
		return fmt.Errorf("failed to download from %s , %d", d.Url, resp.StatusCode)
	}

	defer resp.Body.Close()

	out, err := os.Create(filepath.Clean(d.Filepath))
	if err != nil {
		return err
	}

	defer out.Close()

	_, err = io.Copy(io.MultiWriter(out), resp.Body)
	if err != nil {
		return err
	}

	log.Debugf("Download %s complete", d.Url)
	return nil
}

type DlAuth struct {
	Token  string
	Header string
}

// NewDlAauth will return a bearer token auth header if token is not nil
func NewDlAuth(token, header string) *DlAuth {

	if token == "" {
		return nil
	}

	d := DlAuth{
		Token:  token,
		Header: http.CanonicalHeaderKey(header),
	}

	return &d
}

func GetDownloader(downloadChan chan DlMsg, id int) {
	log.Tracef("Downloader %d started", id)
	for msg := range downloadChan {
		log.Debugf("downloader %d is handling %s\n", id, msg.Url)
		err := msg.DownloadFile()
		msg.ConfirmChan <- err
		msg.Wg.Done()
	}
	log.Tracef("Downloader %d finished", id)

}
