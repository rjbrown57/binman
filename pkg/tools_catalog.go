package binman

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ToolCatalog stores user preferences and user-authored tool descriptions. Installed
// tools themselves are always discovered from the live binman links.
type ToolCatalog struct {
	Commentary string         `yaml:"commentary,omitempty" json:"commentary,omitempty"`
	Tools      []ToolMetadata `yaml:"tools,omitempty" json:"tools,omitempty"`
}

// ToolMetadata is keyed by source, repo, and link name in the local catalog.
type ToolMetadata struct {
	Source             string   `yaml:"source" json:"source"`
	Repo               string   `yaml:"repo" json:"repo"`
	Name               string   `yaml:"name" json:"name"`
	Enabled            bool     `yaml:"enabled" json:"enabled"`
	Checksum           string   `yaml:"checksum,omitempty" json:"checksum,omitempty"`
	BlockedSubcommands []string `yaml:"blocked_subcommands,omitempty" json:"blocked_subcommands,omitempty"`
	Commentary         string   `yaml:"commentary,omitempty" json:"commentary,omitempty"`
	Guidance           string   `yaml:"guidance,omitempty" json:"guidance,omitempty"`
	Summary            string   `yaml:"summary,omitempty" json:"summary,omitempty"`
	Usage              string   `yaml:"usage,omitempty" json:"usage,omitempty"`
}

// ToolChecksum returns the SHA-256 digest of the binary at path.
func ToolChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyToolInvocation enforces the saved checksum and blocked subcommand policy.
func VerifyToolInvocation(tool InstalledTool, args []string) error {
	if tool.Checksum == "" {
		return fmt.Errorf("tool %s/%s (%s) has no checksum; run `binman tools refresh` to record one", tool.Source, tool.Repo, tool.Name)
	}
	checksum, err := ToolChecksum(tool.Path)
	if err != nil {
		return fmt.Errorf("checksum tool %s: %w", tool.Name, err)
	}
	if checksum != tool.Checksum {
		return fmt.Errorf("checksum mismatch for tool %s/%s (%s); verify the binary, then run `binman tools refresh --update-checksums` to trust the new version", tool.Source, tool.Repo, tool.Name)
	}
	for _, blocked := range tool.BlockedSubcommands {
		prefix := strings.Fields(blocked)
		if len(prefix) == 0 {
			continue
		}
		for i := 0; i+len(prefix) <= len(args); i++ {
			matches := true
			for j, token := range prefix {
				if args[i+j] != token {
					matches = false
					break
				}
			}
			if matches {
				return fmt.Errorf("tool %s/%s (%s) subcommand %q is blocked by your binman tools policy", tool.Source, tool.Repo, tool.Name, blocked)
			}
		}
	}
	return nil
}

// ToolCatalogPath returns the per-user metadata file path.
func ToolCatalogPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		var err error
		dir, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(dir, "binman", "agent-tools.yaml"), nil
}

// ReadToolCatalog reads optional metadata. A missing file means no preferences.
func ReadToolCatalog() (ToolCatalog, error) {
	path, err := ToolCatalogPath()
	if err != nil {
		return ToolCatalog{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ToolCatalog{}, nil
	}
	if err != nil {
		return ToolCatalog{}, err
	}
	var catalog ToolCatalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return ToolCatalog{}, fmt.Errorf("parse tool catalog %s: %w", path, err)
	}
	if err := validateToolCatalog(catalog); err != nil {
		return ToolCatalog{}, fmt.Errorf("validate tool catalog %s: %w", path, err)
	}
	return catalog, nil
}

// WriteToolCatalog atomically saves metadata while retaining all user fields.
func WriteToolCatalog(catalog ToolCatalog) error {
	if err := validateToolCatalog(catalog); err != nil {
		return err
	}
	path, err := ToolCatalogPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(catalog)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".agent-tools-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func validateToolCatalog(c ToolCatalog) error {
	seen := make(map[string]bool)
	for _, tool := range c.Tools {
		if tool.Source == "" || tool.Repo == "" || tool.Name == "" {
			return errors.New("each tool entry requires source, repo, and name")
		}
		key := toolKey(tool.Source, tool.Repo, tool.Name)
		if seen[key] {
			return fmt.Errorf("duplicate tool metadata for %s/%s (%s)", tool.Source, tool.Repo, tool.Name)
		}
		if tool.Checksum != "" {
			decoded, err := hex.DecodeString(tool.Checksum)
			if err != nil || len(decoded) != sha256.Size {
				return fmt.Errorf("invalid SHA-256 checksum for %s/%s (%s)", tool.Source, tool.Repo, tool.Name)
			}
		}
		for _, command := range tool.BlockedSubcommands {
			if strings.TrimSpace(command) == "" {
				return fmt.Errorf("blocked_subcommands for %s/%s (%s) cannot contain empty entries", tool.Source, tool.Repo, tool.Name)
			}
		}
		seen[key] = true
	}
	return nil
}

func toolKey(source, repo, name string) string { return source + "\x00" + repo + "\x00" + name }

// MergeToolMetadata exposes only explicitly cataloged tools from the live inventory.
func MergeToolMetadata(installed []InstalledTool, catalog ToolCatalog) []InstalledTool {
	metadata := make(map[string]ToolMetadata, len(catalog.Tools))
	for _, entry := range catalog.Tools {
		metadata[toolKey(entry.Source, entry.Repo, entry.Name)] = entry
	}
	filtered := make([]InstalledTool, 0, len(metadata))
	for _, tool := range installed {
		entry, ok := metadata[toolKey(tool.Source, tool.Repo, tool.Name)]
		if !ok {
			continue
		}
		tool.Enabled = entry.Enabled
		tool.Checksum = entry.Checksum
		tool.BlockedSubcommands = append([]string(nil), entry.BlockedSubcommands...)
		tool.Commentary = entry.Commentary
		tool.Guidance = entry.Guidance
		tool.Summary = entry.Summary
		tool.Usage = entry.Usage
		filtered = append(filtered, tool)
	}
	return filtered
}

// RefreshToolCatalog updates checksums only for tools already in the explicit catalog.
func RefreshToolCatalog(catalog ToolCatalog, installed []InstalledTool, updateChecksums bool) (ToolCatalog, error) {
	live := make(map[string]InstalledTool, len(installed))
	for _, tool := range installed {
		live[toolKey(tool.Source, tool.Repo, tool.Name)] = tool
	}
	for i := range catalog.Tools {
		item := &catalog.Tools[i]
		tool, ok := live[toolKey(item.Source, item.Repo, item.Name)]
		if !ok {
			continue
		}
		if item.Checksum == "" || updateChecksums {
			checksum, err := ToolChecksum(tool.Path)
			if err != nil {
				return ToolCatalog{}, fmt.Errorf("checksum tool %s: %w", tool.Name, err)
			}
			item.Checksum = checksum
		}
	}
	return catalog, nil
}

// Copy tool output to the command stdout while retaining standard exit behavior.
func RunInstalledTool(path string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.Command(path, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	return cmd.Run()
}
