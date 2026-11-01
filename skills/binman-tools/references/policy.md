# Personal binman tool catalog

The catalog is an explicit allowlist. Tools are absent until the user adds a repository with `binman tools add <source/org/repo>`. That command adds and enables all installed binaries from the repository. `binman tools refresh` only updates checksums for catalog entries and preserves user-authored fields:

```yaml
commentary: Prefer read-only inspection tools when exploring a repository.
tools:
  - source: github.com
    repo: example/dangerous-tool
    name: dangerous-tool
    enabled: true
    blocked_subcommands:
      - delete
      - repo remove
    commentary: Avoid this tool for production clusters.
    guidance: Use only for disposable test environments.
    summary: Inspect and change disposable test environments.
    usage: dangerous-tool <command> [flags]
```

Each entry is identified by `source`, `repo`, and `name`, which handles link-name collisions. `binman tools add` enables newly added entries; `binman tools disable` can disable one, and `binman tools enable` can re-enable it. Enabling records the current binary's checksum when no pin exists. Before execution, binman verifies the binary matches that SHA-256 checksum. `binman tools refresh` fills missing checksums but preserves existing checksums; after verifying an updated binary, explicitly accept its new checksum with `binman tools refresh --update-checksums`.

`blocked_subcommands` is a list of whitespace-separated argument token sequences. Binman rejects invocations containing any listed sequence, such as `delete` or `repo remove`. `binman tools run` enforces these blocks. `summary`, `usage`, `commentary`, and `guidance` are user-authored. They describe what the tool does and when/how the agent should use it. Top-level `commentary` applies to all tools.

The live inventory is rebuilt from installed binman links every time the catalog is listed or a tool is run, but only cataloged entries are shown to the skill. A missing catalog means no tools are available; malformed YAML is reported as an error. Checksums and blocked subcommands add integrity and policy checks; they do not isolate the tool from the user's files, credentials, or network access.
