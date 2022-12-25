package cmd

import (
	binmanconfig "github.com/rjbrown57/binman/pkg/config"
	"github.com/spf13/cobra"
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
		binmanconfig.Edit(config)
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
		binmanconfig.Add(config, args[0])
	},
}

// Config add sub command
var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: "View current config",
	Long:  `View current config`,
	Run: func(cmd *cobra.Command, args []string) {
		binmanconfig.Get(config)
	},
}
