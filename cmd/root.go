package cmd

import (
	"fmt"
	"os"
	"strings"

	binman "github.com/rjbrown57/binman/pkg"
	"github.com/spf13/cobra"
)

var imagePath, baseImage, config, path, repo, targetImageName, version string
var jsonLog, table bool
var debug int

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

		binman.Main(m, table, "config")
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

func validateRepo(repo string) {
	if !strings.Contains(repo, "/") {
		fmt.Printf("Error: %s must be in the format org/repo\n", repo)
		os.Exit(1)
	}
}

func addSubcommands() {
	// add edit/get to config
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configAddCmd)

	// add config to root
	rootCmd.AddCommand(configCmd)

	// add watch to root
	rootCmd.AddCommand(watchCmd)

	// add status to root
	rootCmd.AddCommand(statusCmd)

	// add clean to root
	cleanCmd.Flags().BoolVarP(&cleanDryRun, "dryrun", "r", false, "enable dry run for clean")
	cleanCmd.Flags().IntVarP(&threshold, "threshold", "n", 3, "Non-zero amount of releases to retain")
	cleanCmd.Flags().BoolVarP(&scan, "scan", "s", false, "force update of DB pre clean")
	rootCmd.AddCommand(cleanCmd)

	// add build to root
	buildOciCmd.Flags().StringVar(&baseImage, "base", "alpine:latest", "Base image to append synced binaries to")
	buildOciCmd.Flags().StringVar(&repo, "repo", "", "a specific repo to build OCI image for. E.G rjbrown57/binman:v0.10.1. The version string is optional and if omitted the latest version will be used. Leave empty to build a toolbox image of all synced releases")
	buildOciCmd.Flags().StringVar(&targetImageName, "publishPath", "", "target to publish OCI image to. Should be a valid docker image name. If version is left empty it will be generated")
	buildOciCmd.Flags().StringVar(&imagePath, "imageBinPath", "/usr/local/bin/", "Where binaries should be located within the image")
	buildOciCmd.MarkFlagRequired("publishPath")
	buildCmd.AddCommand(buildOciCmd)

	rootCmd.AddCommand(buildCmd)

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
	rootCmd.PersistentFlags().CountVarP(&debug, "debug", "d", "enable debug logging. Set multiple times to increase log level")
	rootCmd.PersistentFlags().BoolVarP(&jsonLog, "json", "j", false, "enable json style logging")
	rootCmd.PersistentFlags().BoolVarP(&table, "table", "t", false, "Output table after sync completion")
}
