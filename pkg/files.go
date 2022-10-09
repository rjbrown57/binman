package binman

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Test for filetypes
func findfType(filepath string) string {

	zipRegex := regexp.MustCompile(ZipRegEx)
	tarRegex := regexp.MustCompile(TarRegEx)

	switch {
	case tarRegex.MatchString(filepath):
		return "tar"
	case zipRegex.MatchString(filepath):
		return "zip"
	default:
		return "default"
	}
}

func handleZip(publishDir string, zippath string) error {
	archive, err := zip.OpenReader(zippath)
	if err != nil {
		log.Warnf("Unable to open %s", zippath)
		return err
	}
	defer archive.Close()

	for _, f := range archive.File {
		dstPath := filepath.Join(publishDir, f.Name)

		if !strings.HasPrefix(dstPath, filepath.Clean(publishDir)+string(os.PathSeparator)) {
			log.Warnf("Extracted file would have had an invalid path, cannot continue")
			return fmt.Errorf("Extracted file would have had an invalid path, cannot continue")
		}

		if f.FileInfo().IsDir() {
			log.Debugf("creating directory for %s", dstPath)
			err := os.MkdirAll(dstPath, 0750)
			if err != nil {
				log.Warnf("Error creating %s, %v", dstPath, err)
				return fmt.Errorf("Error creating %s, %v", dstPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
			log.Warnf("Error creating %s, %v", filepath.Dir(dstPath), err)
			return fmt.Errorf("Error creating %s, %v", filepath.Dir(dstPath), err)
		}

		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			log.Warnf("Error creating %s, %v", dstPath, err)
			return fmt.Errorf("Error creating %s, %v", dstPath, err)
		}
		defer dstFile.Close()

		fileInArchive, err := f.Open()
		if err != nil {
			log.Warnf("Could not read file inside zip: %s, %v", f.Name, err)
			return fmt.Errorf("Could not read file inside zip: %s, %v", f.Name, err)
		}
		defer fileInArchive.Close()

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			log.Warnf("Could not copy file inside zip: %s, %v", f.Name, err)
			return fmt.Errorf("Could not copy file inside zip: %s, %v", f.Name, err)
		}
	}

	return nil
}

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
				log.Warnf("Error creating %s,%v", newDir, err)
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

	// if none is set no link is requested
	if target == "none" {
		return nil
	}

	// If target exists, remove it
	if _, err := os.Stat(target); err == nil {
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

func writeStringtoFile(path string, thestring string) error {
	return ioutil.WriteFile(path, []byte(thestring), 0600)
}

func downloadFile(path string, url string) error {

	log.Infof("Downloading %s", url)
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

	_, err = io.Copy(io.MultiWriter(out), resp.Body)
	if err != nil {
		return err
	}

	log.Infof("Download %s complete", url)

	return nil
}

// Format strings for processing. Currently used by releaseFileName and DlUrl
func formatString(templateString string, dataString string) string {
	return fmt.Sprintf(templateString, dataString)
}
