package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	binman "github.com/rjbrown57/binman/pkg"
	"github.com/spf13/cobra"

	"github.com/rjbrown57/binman/pkg/constants"
	gh "github.com/rjbrown57/binman/pkg/gh"
	"github.com/rjbrown57/binman/pkg/gl"
	log "github.com/rjbrown57/binman/pkg/logging"
	"gopkg.in/yaml.v3"
)

// Config sub command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "config operations for binman",
	Long:  `config operation for binman`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Config edit sub command
var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "open binman config with system editor ",
	Long:  `open binman config with system editor`,
	Run: func(cmd *cobra.Command, args []string) {
		Edit(binman.NewBMConfig(config))
	},
}

// Config add sub command
var configAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a repo to your binman config",
	Long:  `Add a repo to your binman config`,
	Run: func(cmd *cobra.Command, args []string) {
		for _, repo := range args {
			validateRepo(repo)
		}
		if err := Add(binman.NewBMConfig(config).SetConfig(false), args); err != nil {
			log.Fatalf("issue adding %s %s", args, err)
		}
	},
}

// Config add sub command
var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: "View current config",
	Long:  `View current config`,
	Run: func(cmd *cobra.Command, args []string) {
		Get(binman.NewBMConfig(config))
	},
}

func Edit(c *binman.BMConfig) {

	editorPath := getEditor()

	log.Infof("opening %s with %s", c.ConfigPath, editorPath)

	cmd := exec.Command(editorPath, c.ConfigPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Fatalf("Error executing %s %s %s ---", editorPath, c.ConfigPath, err)
	}

}

func getEditor() string {
	e, err := exec.LookPath(os.Getenv("EDITOR"))

	if err != nil {
		log.Fatalf("Unable to find editor %s", err)
	}

	return e
}

func Add(c *binman.BMConfig, repos []string) error {

	for _, repo := range repos {

		r, err := c.GetRelease(repo)
		// A nil error means the repo is already in the config
		if err == nil {
			return errors.New(fmt.Sprintf("%s is already present in config", repo))
		}

		r.SetSource(c.Config.SourceMap)

		switch r.SourceIdentifier {
		case "github.com":
			r.Version, err = gh.CheckRepo(gh.GetGHCLient(constants.DefaultGHBaseURL, c.Config.SourceMap["github.com"].Tokenvar), r.Repo)
			if err != nil {
				return err
			}
		case "gitlab.com":
			r.Version, err = gl.GLGetLatestTag(gl.GetGLClient(constants.DefaultGLBaseURL, c.Config.SourceMap["gitlab.com"].Tokenvar), r.Repo)
			if err != nil {
				return err
			}
		}

		log.Infof("Adding %s to %s. Latest version is %s", repo, c.ConfigPath, r.Version)
		c.Releases = append(c.Releases, binman.BinmanRelease{Repo: repo})
	}

	newConfig, err := yaml.Marshal(&c)
	if err != nil {
		log.Fatalf("Unable to marshal new config %s", err)
	}

	// Write back
	err = binman.WriteStringtoFile(c.ConfigPath, string(newConfig))
	if err != nil {
		return err
	}
	return nil
}

func Get(c *binman.BMConfig) {
	data, err := os.ReadFile(c.ConfigPath)
	if err != nil {
		log.Fatalf("Unable to read file %s", c.ConfigPath)
	}

	log.Infof("Current config(%s):\n%s", c.ConfigPath, string(data))
}
