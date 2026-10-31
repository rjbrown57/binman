# Tool catalog and invocation

Binman can make binaries installed through its normal release configuration available to agents. The agent catalog is an explicit allowlist: a tool is unavailable to the skill and cannot be invoked with `binman tools run` until the user adds it.

## Add installed tools

Add a repository after its binaries have been installed by binman:

```sh
binman tools add github.com/anchore/syft
```

This adds and enables every installed binary linked from that repository and records each binary's SHA-256 checksum. `binman tools` lists the cataloged tools; `binman tools --json` emits the catalog for agents. Tools from other installed repositories remain unavailable until added separately.

## Manage the catalog

The personal catalog is stored at `${XDG_CONFIG_HOME:-$HOME/.config}/binman/agent-tools.yaml`. Open it with `binman tools edit`. To manually remove a repository or binary from the allowlist, remove its entry from this file.

Use these commands to manage catalog entries:

```sh
binman tools enable [--repo source/org/repo] <link-name>
binman tools disable [--repo source/org/repo] <link-name>
binman tools refresh [--update-checksums]
```

`disable` prevents a cataloged tool from running while keeping it in the allowlist. `enable` allows it again. `refresh` records checksums that are missing but preserves existing checksum pins. After verifying an updated binary, use `refresh --update-checksums` to accept its new checksum.

Each catalog entry is keyed by source, repository, and link name. The user may add guidance and restrictions in YAML:

```yaml
commentary: Prefer read-only inspection tools when exploring a repository.
tools:
  - source: github.com
    repo: anchore/syft
    name: syft
    enabled: true
    blocked_subcommands:
      - delete
    summary: Inspect package manifests in container images.
    usage: syft packages <image>
    guidance: Use to inventory packages in container images.
```

`binman tools add` and `binman tools enable` populate a missing `checksum` field. Binman checks that checksum before each invocation. Existing checksums remain pinned until explicitly refreshed with `--update-checksums`.

`blocked_subcommands` contains whitespace-separated argument token sequences. Binman rejects invocations containing a listed sequence. `summary`, `usage`, `commentary`, and `guidance` describe what a tool does and when/how an agent should use it. The top-level `commentary` applies to all tools. These descriptions and restrictions are user-authored.

## Run a tool

Use `binman tools run` to invoke an enabled, cataloged tool:

```sh
binman tools run --repo github.com/anchore/syft syft -- packages .
```

Binman forwards standard input and output and returns the tool's exit status. If link names collide, use `--repo source/org/repo` to select the intended binary.

The allowlist, checksums, and blocked subcommands provide policy and integrity checks. They do not sandbox the process: an invoked tool still runs with the caller's filesystem, credentials, and network access.
