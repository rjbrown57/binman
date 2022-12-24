package cmd

import (
	"fmt"
	"os"
	"strings"

	binman "github.com/rjbrown57/binman/pkg"
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
		m := make(map[string]string)
		m["configFile"] = config

		if !strings.Contains(args[0], "/") {
			fmt.Printf("%s must be in the format org/repo", args[0])
			os.Exit(1)
		}

		m["repo"] = args[0]
		m["version"] = version
		m["path"] = path

		binman.Main(m, debug, jsonLog, "get")
	},
}
