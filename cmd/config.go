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
