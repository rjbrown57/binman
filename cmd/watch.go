package cmd

import (
	binman "github.com/rjbrown57/binman/pkg"
	"github.com/spf13/cobra"
)

// watch sub command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "start binman in prometheus exporter mode",
	Long:  `start binman in prometheus exporter mode to expose metrics of latest releases`,
	Run: func(cmd *cobra.Command, args []string) {
		binman.StartWatch(binman.NewBMWatch(config))
	},
}
