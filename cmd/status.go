package cmd

import (
	binman "github.com/rjbrown57/binman/pkg"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/spf13/cobra"
)

// Config sub command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "status operations for binman",
	Long:  `status operations for binman`,
	Run: func(cmd *cobra.Command, args []string) {

		// Set the logging options
		log.ConfigureLog(jsonLog, debug)
		binman.OutputDbStatus()
	},
}
