package binman

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/rjbrown57/binman/pkg/logging"
	"gopkg.in/yaml.v2"
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
			return fmt.Errorf("extracted file would have had an invalid path, cannot continue")
		}

		if f.FileInfo().IsDir() {
			log.Debugf("creating directory for %s", dstPath)
			err := os.MkdirAll(dstPath, 0750)
			if err != nil {
				log.Warnf("Error creating %s, %v", dstPath, err)
				return fmt.Errorf("error creating %s, %v", dstPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
			log.Warnf("Error creating %s, %v", filepath.Dir(dstPath), err)
			return fmt.Errorf("error creating %s, %v", filepath.Dir(dstPath), err)
		}

		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			log.Warnf("Error creating %s, %v", dstPath, err)
			return fmt.Errorf("error creating %s, %v", dstPath, err)
		}
		defer dstFile.Close()

		fileInArchive, err := f.Open()
		if err != nil {
			log.Warnf("Could not read file inside zip: %s, %v", f.Name, err)
			return fmt.Errorf("could not read file inside zip: %s, %v", f.Name, err)
		}
		defer fileInArchive.Close()

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			log.Warnf("Could not copy file inside zip: %s, %v", f.Name, err)
			return fmt.Errorf("could not copy file inside zip: %s, %v", f.Name, err)
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
		_, err = io.Copy(wf, tar)
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
		log.Fatalf("ExtractTarGz: NewReader failed")
	}

	return uncompressedStream
}

func WriteStringtoFile(path string, thestring string) error {
	return os.WriteFile(path, []byte(thestring), 0600)
}

// mustUnmarshalYaml will Unmarshall from config to GHBMConfig
func mustUnmarshalYaml(configPath string, v interface{}) {
	yamlFile, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		log.Fatalf("err opening %s   #%v\n", configPath, err)
		os.Exit(1)
	}
	err = yaml.Unmarshal(yamlFile, v)
	if err != nil {
		log.Fatalf("unmarhsal error   #%v\n", err)
		os.Exit(1)
	}
}
