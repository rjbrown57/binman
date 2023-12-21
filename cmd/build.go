package cmd

import (
	binman "github.com/rjbrown57/binman/pkg"
	log "github.com/rjbrown57/binman/pkg/logging"

	"github.com/spf13/cobra"
)

// Config sub command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build oci image or tarballs images based on synced binman releases",
	Long:  `Build OCI image or tarball. See subcommands for appropriate flags.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var buildOciCmd = &cobra.Command{
	Use:   "oci",
	Short: "build oci images based on synced binman releases",
	Long: `build oci images based on synced binman releases. Supply -r to build an image of a specific repo or
	build a toolbox oci image with every synced binary omit the -r flag`,
	Run: func(cmd *cobra.Command, args []string) {

		var err error
		// Set the logging options
		log.ConfigureLog(jsonLog, debug)

		log.Infof("Building OCI Image and publishing to %s", targetImageName)

		if err = binman.BuildOciImage(config, repo, targetImageName, baseImage, imagePath); err != nil {
			log.Fatalf("Failed to build image %s", err)
		}

		log.Infof("%s built successfully", targetImageName)
	},
}
