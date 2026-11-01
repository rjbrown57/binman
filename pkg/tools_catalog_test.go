package binman

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToolCatalogMergeAndRefreshPreserveGuidance(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	toolPath := filepath.Join(root, "tool")
	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\nprintf 'A useful package scanner\\nUsage: scanner [flags]\\n'\n"), 0755); err != nil {
		t.Fatal(err)
	}
	installed := []InstalledTool{{Name: "scanner", Source: "github.com", Repo: "org/scanner", Path: toolPath}}
	catalog := ToolCatalog{Commentary: "Prefer read-only tools.", Tools: []ToolMetadata{{
		Source: "github.com", Repo: "org/scanner", Name: "scanner", Enabled: true,
		Commentary: "For repository inventories.", Guidance: "Run only on local files.",
		Summary: "Inspect package manifests.", Usage: "scanner [flags] <path>",
	}}}
	if err := WriteToolCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	refreshed, err := RefreshToolCatalog(catalog, installed, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(refreshed.Tools) != 1 {
		t.Fatalf("expected one refreshed entry, got %d", len(refreshed.Tools))
	}
	tool := refreshed.Tools[0]
	if !tool.Enabled || tool.Commentary != "For repository inventories." || tool.Guidance != "Run only on local files." {
		t.Fatalf("refresh lost user metadata: %#v", tool)
	}
	if tool.Summary != "Inspect package manifests." || tool.Usage != "scanner [flags] <path>" {
		t.Fatalf("unexpected help metadata: %#v", tool)
	}
	checksum, err := ToolChecksum(toolPath)
	if err != nil || tool.Checksum != checksum {
		t.Fatalf("unexpected checksum: %q, %v", tool.Checksum, err)
	}
	tool.BlockedSubcommands = []string{"image delete"}
	checked := MergeToolMetadata(installed, ToolCatalog{Tools: []ToolMetadata{tool}})[0]
	if err := VerifyToolInvocation(checked, []string{"image", "delete", "prod"}); err == nil {
		t.Fatal("expected blocked subcommand")
	}
	if err := VerifyToolInvocation(checked, []string{"image", "inspect", "prod"}); err != nil {
		t.Fatalf("allowed subcommand rejected: %v", err)
	}
	if err := WriteToolCatalog(refreshed); err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadToolCatalog()
	if err != nil {
		t.Fatal(err)
	}
	merged := MergeToolMetadata(installed, loaded)
	if len(merged) != 1 || !merged[0].Enabled || merged[0].Guidance != "Run only on local files." || merged[0].Summary != "Inspect package manifests." {
		t.Fatalf("unexpected merged tool: %#v", merged)
	}
	originalChecksum := loaded.Tools[0].Checksum
	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\necho replaced\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := VerifyToolInvocation(merged[0], nil); err == nil {
		t.Fatal("expected changed binary checksum to be rejected")
	}
	preserved, err := RefreshToolCatalog(loaded, installed, false)
	if err != nil || preserved.Tools[0].Checksum != originalChecksum {
		t.Fatalf("ordinary refresh changed the pinned checksum: %#v, %v", preserved, err)
	}
	updated, err := RefreshToolCatalog(loaded, installed, true)
	if err != nil || updated.Tools[0].Checksum == originalChecksum {
		t.Fatalf("explicit checksum update did not record new binary: %#v, %v", updated, err)
	}
}

func TestToolCatalogValidation(t *testing.T) {
	err := validateToolCatalog(ToolCatalog{Tools: []ToolMetadata{{Source: "github.com", Repo: "org/repo"}, {Source: "github.com", Repo: "org/repo"}}})
	if err == nil {
		t.Fatal("expected incomplete tool identity validation error")
	}
	err = validateToolCatalog(ToolCatalog{Tools: []ToolMetadata{
		{Source: "github.com", Repo: "org/repo", Name: "tool"},
		{Source: "github.com", Repo: "org/repo", Name: "tool"},
	}})
	if err == nil {
		t.Fatal("expected duplicate metadata validation error")
	}
}
