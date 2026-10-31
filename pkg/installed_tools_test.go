package binman

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListInstalledTools(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	root := filepath.Join(dir, "releases")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "repos", "github.com", "org", "project", "v1", "project")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(binDir, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "missing"), filepath.Join(binDir, "broken")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/bin/sh", filepath.Join(binDir, "external")); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.yaml")
	config := "config:\n  releasepath: " + root + "\n  binpath: " + binDir + "\nreleases:\n  - repo: org/project\n    linkname: alias\n"
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	tools, err := ListInstalledTools(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].Name != "alias" || tools[0].Repo != "org/project" || tools[0].Source != "github.com" {
		t.Fatalf("unexpected tools: %#v", tools)
	}
}

func TestListInstalledToolsUsesInstallerSourceAndGlobalPaths(t *testing.T) {
	dir := t.TempDir()
	globalBin := filepath.Join(dir, "global-bin")
	globalReleases := filepath.Join(dir, "global-releases")
	overrideBin := filepath.Join(dir, "override-bin")
	overrideReleases := filepath.Join(dir, "override-releases")
	for _, path := range []string{globalBin, overrideBin} {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
	}
	install := func(root, source, repo, name string) {
		t.Helper()
		target := filepath.Join(root, "repos", source, repo, "v1", name)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(globalBin, name)); err != nil {
			t.Fatal(err)
		}
	}
	install(globalReleases, "github.com", "anchore/syft", "syft")
	install(globalReleases, "gitlab.com", "gitlab-org/cli", "glab")
	configPath := filepath.Join(dir, "config.yaml")
	config := "defaults:\n  source: binmank8s\nconfig:\n  releasepath: " + globalReleases +
		"\n  binpath: " + globalBin + "\n  sources:\n    - name: binmank8s\n      apitype: binman\n      url: http://localhost\nreleases:\n" +
		"  - repo: anchore/syft\n    releasepath: " + overrideReleases + "\n    binpath: " + overrideBin + "\n" +
		"  - repo: gitlab.com/gitlab-org/cli\n    releasepath: " + overrideReleases + "\n    binpath: " + overrideBin + "\n"
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	tools, err := ListInstalledTools(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected tools in global paths, got %#v", tools)
	}
	got := map[string]string{}
	for _, tool := range tools {
		got[tool.Name] = tool.Source + "/" + tool.Repo
	}
	if got["syft"] != "github.com/anchore/syft" || got["glab"] != "gitlab.com/gitlab-org/cli" {
		t.Fatalf("unexpected source identities: %#v", got)
	}
}
