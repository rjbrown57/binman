package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	binman "github.com/rjbrown57/binman/pkg"
	rodaineTable "github.com/rodaine/table"
	"github.com/spf13/cobra"
)

type childExitError struct{ code int }

func (e childExitError) Error() string { return fmt.Sprintf("tool exited with status %d", e.code) }

var toolsRunRepo string
var toolsEnableRepo, toolsDisableRepo string
var toolsUpdateChecksums bool

var toolsCmd = &cobra.Command{
	Use:     "tools",
	Short:   "List and manage installed tool metadata",
	Args:    cobra.NoArgs,
	Aliases: []string{"tool"},
	RunE:    func(cmd *cobra.Command, args []string) error { return showTools(cmd) },
}

var toolAddCmd = &cobra.Command{
	Use:   "add <source/org/repo>",
	Short: "Add installed binaries from a repository to the agent catalog",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		count, err := addToolRepository(args[0])
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Added and enabled %d tool(s) from %s.\n", count, args[0])
		return nil
	},
}

var toolsRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh the saved catalog of installed tools",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		installed, err := binman.ListInstalledTools(config)
		if err != nil {
			return err
		}
		catalog, err := binman.ReadToolCatalog()
		if err != nil {
			return err
		}
		catalog, err = binman.RefreshToolCatalog(catalog, installed, toolsUpdateChecksums)
		if err != nil {
			return err
		}
		if err := binman.WriteToolCatalog(catalog); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Refreshed checksums for %d cataloged tools.\n", len(catalog.Tools))
		return nil
	},
}

var toolsEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the personal tool catalog in your editor",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogPath, err := binman.ToolCatalogPath()
		if err != nil {
			return err
		}
		if _, err := os.Stat(catalogPath); errors.Is(err, os.ErrNotExist) {
			if err := binman.WriteToolCatalog(binman.ToolCatalog{}); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		editorPath, err := exec.LookPath(os.Getenv("EDITOR"))
		if err != nil {
			return fmt.Errorf("unable to find editor %q: %w", os.Getenv("EDITOR"), err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Opening %s with %s\n", catalogPath, editorPath)
		editor := exec.Command(editorPath, catalogPath)
		editor.Stdin, editor.Stdout, editor.Stderr = os.Stdin, cmd.OutOrStdout(), cmd.ErrOrStderr()
		if err := editor.Run(); err != nil {
			return fmt.Errorf("run editor %s: %w", editorPath, err)
		}
		return nil
	},
}

type toolsCatalogOutput struct {
	Commentary string                 `json:"commentary,omitempty"`
	Tools      []binman.InstalledTool `json:"tools"`
}

var toolsRunCmd = &cobra.Command{
	Use:           "run [--repo source/org/repo] <link-name> -- [args...]",
	Short:         "Run an installed tool subject to the personal tool policy",
	Args:          cobra.MinimumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		installed, err := binman.ListInstalledTools(config)
		if err != nil {
			return err
		}
		liveTools := installed
		catalog, err := binman.ReadToolCatalog()
		if err != nil {
			return err
		}
		installed = binman.MergeToolMetadata(installed, catalog)
		var matches []binman.InstalledTool
		for _, tool := range installed {
			if tool.Name == args[0] && (toolsRunRepo == "" || toolsRunRepo == tool.Source+"/"+tool.Repo) {
				matches = append(matches, tool)
			}
		}
		if len(matches) == 0 {
			return unavailableToolError(args[0], liveTools)
		}
		if len(matches) > 1 {
			return fmt.Errorf("tool %q is ambiguous; select one with --repo", args[0])
		}
		tool := matches[0]
		if !tool.Enabled {
			return fmt.Errorf("tool %s/%s (%s) is disabled; enable it with `binman tools enable --repo %s/%s %s`", tool.Source, tool.Repo, tool.Name, tool.Source, tool.Repo, tool.Name)
		}
		childArgs := args[1:]
		if len(childArgs) > 0 && childArgs[0] == "--" {
			childArgs = childArgs[1:]
		}
		if err := binman.VerifyToolInvocation(tool, childArgs); err != nil {
			return err
		}
		err = binman.RunInstalledTool(tool.Path, childArgs, os.Stdin, cmd.OutOrStdout(), cmd.ErrOrStderr())
		if err != nil {
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				code := exit.ExitCode()
				if code < 0 {
					code = 1
				}
				return childExitError{code}
			}
			return fmt.Errorf("run %s: %w", tool.Path, err)
		}
		return nil
	},
}

func setToolEnabled(name, repo string, enabled bool) error {
	installed, err := binman.ListInstalledTools(config)
	if err != nil {
		return err
	}
	var matches []binman.InstalledTool
	for _, tool := range installed {
		if tool.Name == name && (repo == "" || repo == tool.Source+"/"+tool.Repo) {
			matches = append(matches, tool)
		}
	}
	if len(matches) == 0 {
		return fmt.Errorf("tool %q is not installed; run `binman tools` to see available tools", name)
	}
	if len(matches) > 1 {
		return fmt.Errorf("tool %q is ambiguous; select one with --repo", name)
	}
	catalog, err := binman.ReadToolCatalog()
	if err != nil {
		return err
	}
	tool := matches[0]
	found := false
	for i := range catalog.Tools {
		entry := &catalog.Tools[i]
		if entry.Source == tool.Source && entry.Repo == tool.Repo && entry.Name == tool.Name {
			entry.Enabled = enabled
			if enabled && entry.Checksum == "" {
				entry.Checksum, err = binman.ToolChecksum(tool.Path)
				if err != nil {
					return fmt.Errorf("checksum tool %s: %w", tool.Name, err)
				}
			}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("tool %q is not in the agent catalog; add it with `binman tools add %s/%s`", name, tool.Source, tool.Repo)
	}
	return binman.WriteToolCatalog(catalog)
}

func unavailableToolError(name string, installed []binman.InstalledTool) error {
	for _, tool := range installed {
		if tool.Name == name {
			return fmt.Errorf("tool %q is installed but not in the agent catalog; add it with `binman tools add %s/%s`", name, tool.Source, tool.Repo)
		}
	}
	return fmt.Errorf("tool %q is not installed by binman; run `binman tools` to see cataloged tools", name)
}

func addToolRepository(repository string) (int, error) {
	parts := strings.SplitN(repository, "/", 2)
	if len(parts) != 2 || parts[0] == "" || !strings.Contains(parts[1], "/") {
		return 0, fmt.Errorf("repository must be in source/org/repo form, for example github.com/anchore/syft")
	}
	installed, err := binman.ListInstalledTools(config)
	if err != nil {
		return 0, err
	}
	catalog, err := binman.ReadToolCatalog()
	if err != nil {
		return 0, err
	}
	added := 0
	for _, tool := range installed {
		if tool.Source+"/"+tool.Repo != repository {
			continue
		}
		found := false
		for i := range catalog.Tools {
			entry := &catalog.Tools[i]
			if entry.Source != tool.Source || entry.Repo != tool.Repo || entry.Name != tool.Name {
				continue
			}
			entry.Enabled = true
			if entry.Checksum == "" {
				entry.Checksum, err = binman.ToolChecksum(tool.Path)
				if err != nil {
					return added, fmt.Errorf("checksum tool %s: %w", tool.Name, err)
				}
			}
			found = true
			break
		}
		if !found {
			checksum, err := binman.ToolChecksum(tool.Path)
			if err != nil {
				return added, fmt.Errorf("checksum tool %s: %w", tool.Name, err)
			}
			catalog.Tools = append(catalog.Tools, binman.ToolMetadata{
				Source: tool.Source, Repo: tool.Repo, Name: tool.Name, Enabled: true, Checksum: checksum,
			})
		}
		added++
	}
	if added == 0 {
		return 0, fmt.Errorf("no installed binman tools found for %s", repository)
	}
	if err := binman.WriteToolCatalog(catalog); err != nil {
		return added, err
	}
	return added, nil
}

var toolsEnableCmd = &cobra.Command{Use: "enable [--repo source/org/repo] <link-name>", Short: "Enable a cataloged tool for binman tools run", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	if err := setToolEnabled(args[0], toolsEnableRepo, true); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Enabled tool %s.\n", args[0])
	return nil
}}
var toolsDisableCmd = &cobra.Command{Use: "disable [--repo source/org/repo] <link-name>", Short: "Disable a tool for binman tools run", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	if err := setToolEnabled(args[0], toolsDisableRepo, false); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Disabled tool %s.\n", args[0])
	return nil
}}

func showTools(cmd *cobra.Command) error {
	tools, err := binman.ListInstalledTools(config)
	if err != nil {
		return err
	}
	catalog, err := binman.ReadToolCatalog()
	if err != nil {
		return err
	}
	tools = binman.MergeToolMetadata(tools, catalog)
	if jsonLog {
		if tools == nil {
			tools = []binman.InstalledTool{}
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(toolsCatalogOutput{Commentary: catalog.Commentary, Tools: tools})
	}
	if len(tools) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No tools added to the agent catalog. Add one with `binman tools add <source/org/repo>`.")
		return err
	}
	header := color.New(color.FgGreen, color.Underline).SprintfFunc()
	firstColumn := color.New(color.FgYellow).SprintfFunc()
	toolTable := rodaineTable.New("Tool", "Repository", "Policy", "Purpose")
	toolTable.WithHeaderFormatter(header).WithFirstColumnFormatter(firstColumn).WithWriter(cmd.OutOrStdout())
	for _, tool := range tools {
		policy, purpose := "disabled", tool.Summary
		if tool.Enabled {
			policy = "enabled"
		}
		if tool.Guidance != "" {
			purpose = tool.Guidance
		} else if tool.Commentary != "" {
			purpose = tool.Commentary
		}
		toolTable.AddRow(tool.Name, tool.Source+"/"+tool.Repo, policy, purpose)
	}
	toolTable.Print()
	return nil
}

func init() {
	toolsRunCmd.Flags().StringVar(&toolsRunRepo, "repo", "", "select a tool by source and repository")
	toolsRunCmd.Flags().SetInterspersed(false)
	toolsEnableCmd.Flags().StringVar(&toolsEnableRepo, "repo", "", "select a tool by source and repository")
	toolsDisableCmd.Flags().StringVar(&toolsDisableRepo, "repo", "", "select a tool by source and repository")
	toolsRefreshCmd.Flags().BoolVar(&toolsUpdateChecksums, "update-checksums", false, "replace saved checksums with the currently installed tools")
	toolsCmd.AddCommand(toolAddCmd, toolsRefreshCmd, toolsEditCmd, toolsRunCmd, toolsEnableCmd, toolsDisableCmd)
	rootCmd.AddCommand(toolsCmd)
}
