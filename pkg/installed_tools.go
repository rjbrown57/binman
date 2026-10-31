package binman

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// InstalledTool is a runnable link belonging to a configured binman release.
type InstalledTool struct {
	Name               string   `json:"name"`
	Source             string   `json:"source"`
	Repo               string   `json:"repo"`
	Path               string   `json:"path"`
	Enabled            bool     `json:"enabled"`
	Checksum           string   `json:"checksum,omitempty"`
	BlockedSubcommands []string `json:"blocked_subcommands,omitempty"`
	Commentary         string   `json:"commentary,omitempty"`
	Guidance           string   `json:"guidance,omitempty"`
	Summary            string   `json:"summary,omitempty"`
	Usage              string   `json:"usage,omitempty"`
}

// ListInstalledTools reads configuration and installed links without syncing or
// opening the database. Only links into a configured release directory qualify.
func ListInstalledTools(configPath string) ([]InstalledTool, error) {
	if configPath == "" {
		configPath = os.Getenv("BINMAN_CONFIG")
	}
	if configPath == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(dir, "binman", "config")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read binman config: %w", err)
	}
	var cfg BMConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse binman config: %w", err)
	}
	if contextual, ok := detectRepoConfig(); ok {
		data, err := os.ReadFile(contextual)
		if err != nil {
			return nil, err
		}
		var local BMConfig
		if err := yaml.Unmarshal(data, &local); err != nil {
			return nil, fmt.Errorf("parse contextual binman config: %w", err)
		}
		cfg.Releases = append(cfg.Releases, local.Releases...)
	}
	if cfg.Config.ReleasePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		cfg.Config.ReleasePath = filepath.Join(home, "binMan")
	}
	if cfg.Config.BinPath == "" {
		cfg.Config.BinPath = cfg.Config.ReleasePath
	}
	// Resolve sources the same way installation does. In particular, a binman
	// default is the query transport; it does not become the upstream storage ID.
	setDefaultSources(&cfg)
	if cfg.Defaults.Source == "" {
		cfg.Config.SourceMap["default"] = cfg.Config.SourceMap["github.com"]
	} else {
		cfg.Config.SourceMap["default"] = cfg.Config.SourceMap[cfg.Defaults.Source]
	}
	releasePath, err := filepath.Abs(cfg.Config.ReleasePath)
	if err != nil {
		return nil, err
	}
	binPath, err := filepath.Abs(cfg.Config.BinPath)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var tools []InstalledTool
	for _, rel := range cfg.Releases {
		release := rel
		release.SetSource(cfg.Config.SourceMap)
		repo, source := release.Repo, release.SourceIdentifier
		if !strings.Contains(repo, "/") || rel.DownloadOnly || rel.PostOnly {
			continue
		}
		root, err := filepath.Abs(filepath.Join(releasePath, "repos", source, repo))
		if err != nil {
			return nil, err
		}
		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			root = resolved
		}
		entries, err := os.ReadDir(binPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read binpath %s: %w", binPath, err)
		}
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink == 0 {
				continue
			}
			path := filepath.Join(binPath, entry.Name())
			target, err := filepath.EvalSymlinks(path)
			if err != nil { // Broken links are not runnable.
				continue
			}
			relative, err := filepath.Rel(root, target)
			if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
				continue
			}
			info, err := os.Stat(target)
			if err != nil || !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0) {
				continue
			}
			if !seen[path+"\x00"+source+"\x00"+repo] {
				tools = append(tools, InstalledTool{Name: entry.Name(), Source: source, Repo: repo, Path: path})
				seen[path+"\x00"+source+"\x00"+repo] = true
			}
		}
	}
	sort.Slice(tools, func(i, j int) bool {
		if tools[i].Name != tools[j].Name {
			return tools[i].Name < tools[j].Name
		}
		return tools[i].Path < tools[j].Path
	})
	return tools, nil
}
