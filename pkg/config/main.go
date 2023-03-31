package config

import (
	"os"
	"os/exec"

	binman "github.com/rjbrown57/binman/pkg"
	gh "github.com/rjbrown57/binman/pkg/gh"
	log "github.com/rjbrown57/binman/pkg/logging"
	"gopkg.in/yaml.v3"
)

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

// See if repo is already in config
func releasesContains(r []binman.BinmanRelease, repo string) bool {
	for _, v := range r {
		if v.Repo == repo {
			return true
		}
	}
	return false
}

func Add(config string, repo string) {
	cPath := binman.SetBaseConfig(config)
	// We use NewGHBMConfig here to avoid grabbing contextual configs
	currentConfig := binman.NewGHBMConfig(cPath)

	// todo fix this hack
	tag, err := gh.CheckRepo(gh.GetGHCLient("https://api.github.com", currentConfig.Config.TokenVar), repo)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// Verify release is not present
	if releasesContains(currentConfig.Releases, repo) {
		log.Fatalf("%s is already present in %s", repo, cPath)
	}

	currentConfig.Releases = append(currentConfig.Releases, binman.BinmanRelease{Repo: repo})

	newConfig, err := yaml.Marshal(&currentConfig)
	if err != nil {
		log.Fatalf("Unable to marshal new config %s", err)
	}

	log.Infof("Adding %s to %s. Latest version is %s", repo, cPath, tag)

	// Write back
	err = binman.WriteStringtoFile(cPath, string(newConfig))
	if err != nil {
		log.Fatalf("Unable to update config file %s", err)
	}
}

func Get(config string) {
	cPath := binman.SetBaseConfig(config)
	// We use NewGHBMConfig here to avoid grabbing contextual configs
	c, err := os.ReadFile(cPath)
	if err != nil {
		log.Fatalf("Unable to read file %s", cPath)
	}

	log.Infof("Current config(%s):\n%s", cPath, string(c))
}
