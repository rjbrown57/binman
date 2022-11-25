package config

import (
	"os"
	"os/exec"

	binman "github.com/rjbrown57/binman/pkg"
	log "github.com/rjbrown57/binman/pkg/logging"
	"gopkg.in/yaml.v3"
)

// https://github.com/kubernetes/kubectl/blob/da50ec2b223f5ec08bc34b700411c70b2bcc87fd/pkg/cmd/util/editor/editor.go

func Edit(config string) {

	editorPath := getEditor()
	cPath := binman.SetBaseConfig(config)

	log.Infof("opening %s", editorPath)

	cmd := exec.Command(editorPath, cPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Fatalf("Error executing %s %s %s ---", editorPath, cPath, err)
	}

}

func getEditor() string {
	e, err := exec.LookPath(os.Getenv("EDITOR"))

	if err != nil {
		log.Fatalf("Unable to find editor %s", err)
	}

	return e
}

// Notes
// verify repo exists ( this function maybe should be added into the binman general package)
// Read config
// Verify repo is not already present
// Add repo
// Write back to file path
// Add a new repo
func Add(config string, repo string) {
	cPath := binman.SetBaseConfig(config)
	// We use NewGHBMConfig here to avoid grabbing contextual configs
	currentConfig := binman.NewGHBMConfig(cPath)

	// Test if repo is in c.Releases

	// Add the repo
	currentConfig.Releases = append(currentConfig.Releases, binman.BinmanRelease{Repo: repo})
	newConfig, err := yaml.Marshal(&currentConfig)
	if err != nil {
		log.Fatalf("Unable to marshal new config %s", err)
	}

	log.Infof("Adding %s to %s", repo, cPath)

	// Write back
	err = binman.WriteStringtoFile(cPath, string(newConfig))
	if err != nil {
		log.Fatalf("Unable to update config file %s", err)
	}

}
