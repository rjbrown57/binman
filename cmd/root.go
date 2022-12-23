package cmd

import (
	"os"

	binman "github.com/rjbrown57/binman/pkg"
	"github.com/spf13/cobra"
)

var debug bool
var jsonLog bool
var config string
var repo string
var version string
var path string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "binman",
	Short: "GitHub Binary Manager",
	Long:  `Github Binary Manager will grab binaries from github for you!`,
	Run: func(cmd *cobra.Command, args []string) {
		if config == "" && repo == "" {
			err := cmd.Root().Help()
			if err != nil {
				os.Exit(1)
			}
			os.Exit(1)
		}

		m := make(map[string]string)
		m["configFile"] = config
		m["repo"] = repo
		m["version"] = version

		binman.Main(m, debug, jsonLog, "config")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func addSubcommands() {
	// add edit/get to config
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configGetCmd)

	// Setup repo flag and add to root
	configAddCmd.Flags().StringVarP(&repo, "repo", "r", "", "Supply repo to add to config in format org/repo")
	configAddCmd.MarkFlagRequired("repo")
	configCmd.AddCommand(configAddCmd)

	// add config to root
	rootCmd.AddCommand(configCmd)

	// Setup
	wd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}

	getCmd.Flags().StringVarP(&path, "path", "p", wd, "path to download file to")
	getCmd.Flags().StringVarP(&version, "version", "v", "", "Specific version to grab via direct download")

	// add config to root
	rootCmd.AddCommand(getCmd)
}

func init() {

	addSubcommands()

	rootCmd.PersistentFlags().StringVarP(&config, "config", "c", "noConfig", "path to config file. Can be set with ${BINMAN_CONFIG} env var")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug logging")
	rootCmd.PersistentFlags().BoolVarP(&jsonLog, "json", "j", false, "enable json style logging")
}
