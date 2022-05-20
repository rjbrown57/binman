package binman

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
)

func handleTar(publishDir string, tarpath string) error {
	f, err := os.Open(filepath.Clean(tarpath))
	if err != nil {
		log.Warnf("Unable to open %s", tarpath)
		return err
	}

	defer f.Close()

	// Unzip and then handle tar
	// need to detect if this is necessary?
	tar := tar.NewReader(GunZipFile(f))

	for {
		file, err := tar.Next()
		switch err {
		case io.EOF:
			return nil
		case nil:
			break
		default:
			log.Warnf("Error on %s - %v", tarpath, err)

		}

		log.Debugf("%+v", file)

		publishPath := fmt.Sprintf("%s/%s", publishDir, file.Name)

		// if the file.Name has a / it contains a new directory
		if strings.Contains(file.Name, "/") {
			newDir, _ := filepath.Split(publishPath)
			log.Debugf("creating directory for %s", newDir)
			err := os.MkdirAll(newDir, 0750)
			if err != nil {
				log.Warnf("Errore creating %s,%v", newDir, err)
			}
		}

		if file.FileInfo().IsDir() {
			continue
		}

		wf, err := os.Create(filepath.Clean(publishPath))
		if err != nil {
			log.Warnf("Unable to write file %s", publishPath)
			return err
		}

		log.Debugf("tar extract file %s", publishPath)
		io.Copy(wf, tar)
		if err != nil {
			log.Warnf("Unable to write file %s", publishPath)
			return err
		}

	}
}

// unzip gzip file
func GunZipFile(gzipFile io.Reader) *gzip.Reader {
	uncompressedStream, err := gzip.NewReader(gzipFile)
	if err != nil {
		log.Fatal("ExtractTarGz: NewReader failed")
	}

	return uncompressedStream
}

// Create the link to new release
func createReleaseLink(source string, target string) error {

	// If target exists, remove it
	if _, err := os.Stat(target); err != os.ErrExist {
		log.Warnf("Updating %s to %s\n", source, target)
		err := os.Remove(target)
		if err != nil {
			log.Warnf("Unable to remove %s,%v", target, err)
		}
	}

	err := os.Symlink(source, target)
	if err != nil {
		log.Infof("Creating link %s -> %s\n", source, target)
		return err
	}

	return nil
}

func writeNotes(path string, notes string) error {
	return ioutil.WriteFile(path, []byte(notes), 0600)
}

func downloadFile(path string, url string) error {

	log.Infof("Getting %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	out, err := os.Create(filepath.Clean(path))
	if err != nil {
		return err
	}

	defer out.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"downloading",
	)

	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	if err != nil {
		return err
	}

	return nil
}
