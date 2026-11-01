package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	binman "github.com/rjbrown57/binman/pkg"
)

func TestToolsRunDefaultsDisabledAndEnableAllowsArguments(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "user-config"))
	binDir := filepath.Join(dir, "bin")
	releases := filepath.Join(dir, "releases")
	target := filepath.Join(releases, "repos", "github.com", "org", "tool", "v1", "tool")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	result := filepath.Join(dir, "result")
	if err := os.WriteFile(target, []byte("#!/bin/sh\nprintf '%s\\n' \"$1\" > \"$2\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(binDir, "alias")); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.yaml")
	configText := "config:\n  releasepath: " + releases + "\n  binpath: " + binDir + "\nreleases:\n  - repo: org/tool\n"
	if err := os.WriteFile(configPath, []byte(configText), 0600); err != nil {
		t.Fatal(err)
	}
	oldConfig, oldRepo := config, toolsRunRepo
	config, toolsRunRepo = configPath, "github.com/org/tool"
	t.Cleanup(func() { config, toolsRunRepo = oldConfig, oldRepo })
	if err := binmanWriteTestCatalog(t, false); err != nil {
		t.Fatal(err)
	}
	err := toolsRunCmd.RunE(toolsRunCmd, []string{"alias", "--", "value", result})
	if err == nil {
		t.Fatal("expected disabled tool error")
	}
	if _, err := os.Stat(result); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("disabled tool ran: %v", err)
	}
	if err := setToolEnabled("alias", "github.com/org/tool", true); err != nil {
		t.Fatal(err)
	}
	if err := toolsRunCmd.RunE(toolsRunCmd, []string{"alias", "--", "value", result}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result)
	if err != nil || string(data) != "value\n" {
		t.Fatalf("arguments not forwarded: %q, %v", data, err)
	}
	if err := setToolEnabled("alias", "github.com/org/tool", false); err != nil {
		t.Fatal(err)
	}
	if err := toolsRunCmd.RunE(toolsRunCmd, []string{"alias", "--", "value", result}); err == nil {
		t.Fatal("expected disabled tool error after disabling")
	}
	var output bytes.Buffer
	oldJSON := jsonLog
	jsonLog = true
	toolsCmd.SetOut(&output)
	t.Cleanup(func() {
		jsonLog = oldJSON
		toolsCmd.SetOut(nil)
	})
	if err := showTools(toolsCmd); err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Commentary string           `json:"commentary"`
		Tools      []map[string]any `json:"tools"`
	}
	if err := json.Unmarshal(output.Bytes(), &catalog); err != nil || catalog.Commentary != "global note" || len(catalog.Tools) != 1 || catalog.Tools[0]["enabled"] != false {
		t.Fatalf("unexpected tools JSON: %s (%v)", output.String(), err)
	}
}

func binmanWriteTestCatalog(t *testing.T, enabled bool) error {
	t.Helper()
	enabledValue := "false"
	if enabled {
		enabledValue = "true"
	}
	path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "binman", "agent-tools.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("commentary: global note\ntools:\n  - source: github.com\n    repo: org/tool\n    name: alias\n    enabled: "+enabledValue+"\n"), 0600)
}

func TestToolsEditOpensCatalogWithEditor(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "user-config"))
	binDir := filepath.Join(dir, "bin")
	releaseDir := filepath.Join(dir, "releases")
	target := filepath.Join(releaseDir, "repos", "github.com", "org", "tool", "v1", "tool")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("#!/bin/sh\nprintf 'Useful tool description\\nUsage: alias [command]\\n'\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(binDir, "alias")); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "binman.yaml")
	configData := "config:\n  releasepath: " + releaseDir + "\n  binpath: " + binDir + "\nreleases:\n  - repo: org/tool\n"
	if err := os.WriteFile(configPath, []byte(configData), 0600); err != nil {
		t.Fatal(err)
	}
	oldConfig := config
	config = configPath
	t.Cleanup(func() { config = oldConfig })
	marker := filepath.Join(dir, "opened")
	editorPath := filepath.Join(dir, "editor")
	if err := os.WriteFile(editorPath, []byte("#!/bin/sh\nprintf '%s' \"$1\" > \""+marker+"\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("EDITOR", "editor")
	if err := toolsEditCmd.RunE(toolsEditCmd, nil); err != nil {
		t.Fatal(err)
	}
	opened, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	catalogPath, err := binman.ToolCatalogPath()
	if err != nil {
		t.Fatal(err)
	}
	if string(opened) != catalogPath {
		t.Fatalf("editor opened %q, expected %q", opened, catalogPath)
	}
	if _, err := os.Stat(catalogPath); err != nil {
		t.Fatalf("catalog was not created: %v", err)
	}
	contents, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(contents, []byte("name: alias")) {
		t.Fatalf("tools edit added an unrequested tool to the catalog: %s", contents)
	}
	if _, err := addToolRepository("github.com/org/tool"); err != nil {
		t.Fatal(err)
	}
	contents, err = os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(contents, []byte("name: alias")) || !bytes.Contains(contents, []byte("enabled: true")) {
		t.Fatalf("tool add did not enable and catalog the requested binary: %s", contents)
	}
}

func TestToolAliasUsesToolsSubcommands(t *testing.T) {
	for _, args := range [][]string{{"tools", "add"}, {"tool", "add"}, {"tools", "run"}, {"tool", "run"}} {
		command, _, err := rootCmd.Find(args)
		if err != nil {
			t.Fatalf("find command %v: %v", args, err)
		}
		if len(args) == 2 && args[1] == "add" && command != toolAddCmd {
			t.Errorf("%v resolved to %s, want tools add", args, command.CommandPath())
		}
		if len(args) == 2 && args[1] == "run" && command != toolsRunCmd {
			t.Errorf("%v resolved to %s, want tools run", args, command.CommandPath())
		}
	}
}
