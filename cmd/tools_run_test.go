package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestToolsRunForwardsArgumentsAndExitCode(t *testing.T) {
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
	result := filepath.Join(dir, "result")
	if err := os.WriteFile(target, []byte("#!/bin/sh\nprintf '%s\\n' \"$1\" \"$2\" > \"$3\"\nexit 17\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(binDir, "alias")); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.yaml")
	configText := "config:\n  releasepath: " + releaseDir + "\n  binpath: " + binDir + "\nreleases:\n  - repo: org/tool\n"
	if err := os.WriteFile(configPath, []byte(configText), 0600); err != nil {
		t.Fatal(err)
	}
	oldConfig, oldRepo := config, toolsRunRepo
	config, toolsRunRepo = configPath, ""
	t.Cleanup(func() { config, toolsRunRepo = oldConfig, oldRepo })
	if _, err := addToolRepository("github.com/org/tool"); err != nil {
		t.Fatal(err)
	}

	err := toolsRunCmd.RunE(toolsRunCmd, []string{"alias", "--", "-x", "hello world", result})
	var exit childExitError
	if !errors.As(err, &exit) || exit.code != 17 {
		t.Fatalf("expected exit 17, got %v", err)
	}
	data, err := os.ReadFile(result)
	if err != nil || string(data) != "-x\nhello world\n" {
		t.Fatalf("unexpected arguments: %q, %v", data, err)
	}

	otherTarget := filepath.Join(releaseDir, "repos", "github.com", "other", "tool", "v1", "tool")
	if err := os.MkdirAll(filepath.Dir(otherTarget), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(otherTarget, []byte("#!/bin/sh\nexit 9\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(otherTarget, filepath.Join(binDir, "other-alias")); err != nil {
		t.Fatal(err)
	}
	configText = "config:\n  releasepath: " + releaseDir + "\n  binpath: " + binDir + "\nreleases:\n  - repo: org/tool\n  - repo: other/tool\n"
	if err := os.WriteFile(configPath, []byte(configText), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := addToolRepository("github.com/other/tool"); err != nil {
		t.Fatal(err)
	}
	toolsRunRepo = "github.com/other/tool"
	if err := toolsRunCmd.RunE(toolsRunCmd, []string{"alias"}); err == nil {
		t.Fatal("expected repository filter to reject tool from another repository")
	}
	err = toolsRunCmd.RunE(toolsRunCmd, []string{"other-alias"})
	if !errors.As(err, &exit) || exit.code != 9 {
		t.Fatalf("expected selected tool exit 9, got %v", err)
	}
}
