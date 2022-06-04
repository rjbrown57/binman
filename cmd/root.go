/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "binman",
	Short: "GitHub Binary Manager",
	Long:  `Github Binary Manager will grab binaries from github for you!`,
	Run: func(cmd *cobra.Command, args []string) {
		if config == "" && repo == "" {
			cmd.Root().Help()
			os.Exit(1)
		}

		m := make(map[string]string)
		m["configFile"] = config
		m["repo"] = repo
		binman.Main(m, debug, jsonLog)
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

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.rlman.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly
	rootCmd.Flags().StringVarP(&config, "config", "c", "", "path to config file")
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "enable debug logging")
	rootCmd.Flags().BoolVarP(&jsonLog, "json", "j", false, "enable json style logging")
	rootCmd.Flags().StringVarP(&repo, "repo", "r", "", "Github repo in format org/repo")
}
