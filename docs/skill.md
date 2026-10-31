# Codex skill

The `binman-tools` skill helps Codex invoke locally installed tools that the user explicitly added to Binman's agent catalog. For repository allowlisting, catalog policy, checksums, and tool commands, see [Tool catalog and invocation](tools.md).

## Install

Install or update the skill with mise:

```sh
mise run install-skill
```

By default, this copies the skill to `~/.codex/skills/binman-tools`. Set `CODEX_HOME` to use a different Codex home. Once installed, invoke it in Codex with `$binman-tools`, or ask naturally, for example: “Use the binman tools skill to inspect packages in this image.”

## Use the skill

The skill checks `binman tools --json` for the cataloged tool list, considers each tool's user-provided summary and guidance, and invokes selected tools only through `binman tools run`. If a requested installed tool has not been added to the catalog, the skill tells the user to add its repository with `binman tools add <source/org/repo>`; it does not change the catalog itself. `binman tool` is an alias for `binman tools`.

For example:

> Use `$binman-tools` to inventory packages in this container image.

The skill does not install missing tools, add tools to the catalog, or re-enable a disabled tool on the user's behalf.
