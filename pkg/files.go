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

	"github.com/rjbrown57/binman/pkg/constants"
	log "github.com/rjbrown57/binman/pkg/logging"

	"gopkg.in/yaml.v3"
)

// Select asset takes a list of possible assets and makes a choice
func selectAsset(relArch string, relOS string, version string, project string, assets map[string]string) (string, string) {

	var possibleAsset struct {
		Name string
		Url  string
	}

	// sometimes amd64 is represented as x86_64, so we substitute a regex here that covers both
	if relArch == "amd64" || relArch == "x86_64" {
		relArch = constants.X86RegEx
	}

	// darwin/osx/macos also has alternate names so we substitute regex
	if relOS == "darwin" {
		relOS = constants.MacOsRx
	}

	zipRx := regexp.MustCompile(constants.ZipRegEx)
	tarRx := regexp.MustCompile(constants.TarRegEx)
	exeRx := regexp.MustCompile(constants.ExeRegex)
	osRx := regexp.MustCompile(strings.ToLower(relOS))
	archRx := regexp.MustCompile(strings.ToLower(relArch))

	// There are exact match assets and possible match assets
	// any 1 exact match asset will terminate the loop, otherwise we will take the last possible match asset
	for name, url := range assets {

		log.Debugf("name %s - url %s", name, url)
		// This asset matches our OS/Arch
		if osRx.MatchString(name) && archRx.MatchString(name) {
			log.Debugf("Evaluating asset %s\n", name)

			// If asset is an exact match one of our supported styles return name+download url
			// Current styles are tar,zip,exe,linux binary
			switch {
			case exeRx.MatchString(name), !strings.Contains(name, "."), tarRx.MatchString(name), zipRx.MatchString(name):
				log.Debugf("Selected asset %s == %s\n", name, url)
				return name, url
			}

			log.Debugf("Evaluating %s contains version %s", name, version)
			if strings.Contains(name, version) || strings.Contains(name, strings.Trim(version, "v")) && strings.Contains(name, project) {
				log.Debugf("Possible match by version %s %s", version, name)
				possibleAsset.Name = name
				possibleAsset.Url = url
			}
		}

	}

	return possibleAsset.Name, possibleAsset.Url
}

// Create the link to new release
func createLink(source string, target string) error {

	// If target exists, remove it
	if _, err := os.Readlink(target); err == nil {
		log.Debugf("Updating %s to %s\n", source, target)
		err := os.Remove(target)
		if err != nil {
			log.Debugf("Unable to remove %s,%v", target, err)
		}
	}

	err := os.Symlink(source, target)
	if err != nil {
		log.Debugf("Creating link %s -> %s\n", source, target)
		return err
	}

	return nil
}

// Test for filetypes
func findfType(filepath string) string {

	zipRegex := regexp.MustCompile(constants.ZipRegEx)
	tarRegex := regexp.MustCompile(constants.TarRegEx)

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
		log.Debugf("Unable to open %s", zippath)
		return err
	}
	defer archive.Close()

	for _, f := range archive.File {
		dstPath := filepath.Join(publishDir, f.Name)

		if !strings.HasPrefix(dstPath, filepath.Clean(publishDir)+string(os.PathSeparator)) {
			log.Debugf("Extracted file would have had an invalid path, cannot continue")
			return fmt.Errorf("extracted file would have had an invalid path, cannot continue")
		}

		if f.FileInfo().IsDir() {
			log.Debugf("creating directory for %s", dstPath)
			err := os.MkdirAll(dstPath, 0750)
			if err != nil {
				log.Debugf("Error creating %s, %v", dstPath, err)
				return fmt.Errorf("error creating %s, %v", dstPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
			log.Debugf("Error creating %s, %v", filepath.Dir(dstPath), err)
			return fmt.Errorf("error creating %s, %v", filepath.Dir(dstPath), err)
		}

		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			log.Debugf("Error creating %s, %v", dstPath, err)
			return fmt.Errorf("error creating %s, %v", dstPath, err)
		}
		defer dstFile.Close()

		fileInArchive, err := f.Open()
		if err != nil {
			log.Debugf("Could not read file inside zip: %s, %v", f.Name, err)
			return fmt.Errorf("could not read file inside zip: %s, %v", f.Name, err)
		}
		defer fileInArchive.Close()

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			log.Debugf("Could not copy file inside zip: %s, %v", f.Name, err)
			return fmt.Errorf("could not copy file inside zip: %s, %v", f.Name, err)
		}
	}

	return nil
}

func handleTar(publishDir string, tarpath string) error {
	f, err := os.Open(filepath.Clean(tarpath))
	if err != nil {
		log.Debugf("Unable to open %s", tarpath)
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
			log.Debugf("Error on %s - %v", tarpath, err)

		}

		log.Debugf("%+v", file)

		publishPath := fmt.Sprintf("%s/%s", publishDir, file.Name)

		// if the file.Name has a / it contains a new directory
		if strings.Contains(file.Name, "/") {
			newDir, _ := filepath.Split(publishPath)
			log.Debugf("creating directory for %s", newDir)
			err := os.MkdirAll(newDir, 0750)
			if err != nil {
				log.Debugf("Error creating %s,%v", newDir, err)
			}
		}

		if file.FileInfo().IsDir() {
			continue
		}

		wf, err := os.Create(filepath.Clean(publishPath))
		if err != nil {
			log.Debugf("Unable to write file %s", publishPath)
			return err
		}

		log.Debugf("tar extract file %s", publishPath)
		_, err = io.Copy(wf, tar)
		if err != nil {
			log.Debugf("Unable to write file %s", publishPath)
			return err
		}

		os.Chmod(filepath.Clean(publishPath), file.FileInfo().Mode())
		if err != nil {
			log.Debugf("Unable to set perms on file %s", publishPath)
			return err
		}

	}
}

func CopyFile(source string, target string) error {
	f, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	return WriteStringtoFile(target, string(f))
}

func CreateDirectory(path string) error {
	// prepare directory path
	err := os.MkdirAll(path, 0750)
	if err != nil {
		log.Debugf("Error creating %s - %v", path, err)
		return err
	}
	return nil
}

// unzip gzip file
func GunZipFile(gzipFile io.Reader) *gzip.Reader {
	uncompressedStream, err := gzip.NewReader(gzipFile)
	if err != nil {
		log.Fatalf("ExtractTarGz: NewReader failed - %s", err)
	}

	return uncompressedStream
}

func MakeExecuteable(path string) error {
	// make the file executable
	f, err := os.Stat(path)
	if err != nil {
		log.Debugf("Failed to open %s", path)
		return err
	}

	// Set perms if required
	if mode := f.Mode(); mode&os.ModePerm != 0755 {
		log.Debugf("Settings perms to 755 for %s", path)
		err = os.Chmod(path, 0755)
		if err != nil {
			log.Debugf("Failed to set permissions on %s", path)
			return err
		}
	}

	return nil
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
