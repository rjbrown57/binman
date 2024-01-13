package cmd

import (
	"errors"
	"os"
	"os/exec"

	binman "github.com/rjbrown57/binman/pkg"
	"github.com/spf13/cobra"

	"github.com/rjbrown57/binman/pkg/constants"
	gh "github.com/rjbrown57/binman/pkg/gh"
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
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		validateRepo(args[0])
		Add(binman.NewBMConfig(config).SetConfig(false), args[0])
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

func Add(c *binman.BMConfig, repo string) {

	// todo fix this hack
	tag, err := gh.CheckRepo(gh.GetGHCLient(constants.DefaultGHBaseURL, c.Config.SourceMap["github.com"].Tokenvar), repo)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// Verify release is not present
	if _, err := c.GetRelease(repo); !errors.Is(err, binman.ErrReleaseNotFound) {
		log.Fatalf("%s is already present in %s", repo, c.ConfigPath)
	}

	c.Releases = append(c.Releases, binman.BinmanRelease{Repo: repo})

	newConfig, err := yaml.Marshal(&c)
	if err != nil {
		log.Fatalf("Unable to marshal new config %s", err)
	}

	log.Infof("Adding %s to %s. Latest version is %s", repo, c.ConfigPath, tag)

	// Write back
	err = binman.WriteStringtoFile(c.ConfigPath, string(newConfig))
	if err != nil {
		log.Fatalf("Unable to update config file %s", err)
	}
}

func Get(c *binman.BMConfig) {
	data, err := os.ReadFile(c.ConfigPath)
	if err != nil {
		log.Fatalf("Unable to read file %s", c.ConfigPath)
	}

	log.Infof("Current config(%s):\n%s", c.ConfigPath, string(data))
}
