package cmd

import (
	"github.com/rjbrown57/binman/internal"
	binman "github.com/rjbrown57/binman/pkg"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/spf13/cobra"
)

// watch sub command
var watchCmd = &cobra.Command{
	Aliases: []string{"watch"},
	Use:     "server",
	Short:   "start binman in server mode",
	Long:    `start binman in server mode. Allows syncing and serving releases, exposing metrics`,
	Run: func(cmd *cobra.Command, args []string) {
		// Always use json logging with the watch command
		log.ConfigureLog(true, debug)

		internal.StartWatch(binman.NewBMWatch(config))
	},
}
