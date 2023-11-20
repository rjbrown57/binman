package cmd

import (
	"fmt"
	"os"

	binman "github.com/rjbrown57/binman/pkg"
	log "github.com/rjbrown57/binman/pkg/logging"

	"github.com/spf13/cobra"
)

var cleanDryRun, scan bool
var threshold int

// Config sub command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "clean old versions previously synced by binman",
	Long:  `clean old versions previously synced by binman`,
	Run: func(cmd *cobra.Command, args []string) {
		if threshold == 0 {
			fmt.Println("Error: Please use a non-zero value for threshold")
			os.Exit(1)
		}
		err := binman.Clean(cleanDryRun, debug, jsonLog, scan, threshold, "", config)
		if err != nil {
			log.Fatalf("Failed to run clean %s", err)
		}
	},
}
