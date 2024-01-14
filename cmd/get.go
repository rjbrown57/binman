package cmd

import (
	"github.com/rjbrown57/binman/internal"
	binman "github.com/rjbrown57/binman/pkg"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/spf13/cobra"
)

// get sub command
var getCmd = &cobra.Command{
	Use:     "get",
	Short:   "get a single repo",
	Args:    cobra.ExactArgs(1),
	Example: "binman get rjbrown57/binman",
	Long:    `get a single repo with binman. Useful with CI/docker`,
	Run: func(cmd *cobra.Command, args []string) {
		validateRepo(args[0])
		log.ConfigureLog(jsonLog, debug)

		r := binman.BinmanRelease{
			Repo:         args[0],
			Version:      version,
			PublishPath:  path,
			DownloadOnly: true,
		}

		if err := internal.Main(binman.NewGet(r)); err != nil {
			log.Fatalf("Failed to get %s", r.Repo)
		}
	},
}
