---
name: binman-tools
description: Find and run locally installed binman tools when a user task calls for a specialized CLI.
---

# Binman tools

Use `binman tools --json` to get the explicit agent catalog. Only tools listed there have been added for agent use. Apply the global commentary and each tool's commentary and guidance when choosing a tool. If the requested installed tool is absent from the catalog, tell the user to add its repository with `binman tools add <source/org/repo>`; do not add it yourself.

Run a selected tool only through `binman tools run --repo <source/org/repo> <name> -- <args>`. `binman tools add` explicitly adds and enables a repository's installed binaries. If a cataloged tool is disabled, tell the user and ask them to enable it with `binman tools enable [--repo source/org/repo] <name>`; never enable it on the user's behalf. If the requested tool is missing, explain that instead of installing it or silently choosing a substitute.

The personal metadata file is `${XDG_CONFIG_HOME:-$HOME/.config}/binman/agent-tools.yaml`. It stores tool enablement, descriptions, commentary, and usage guidance. Read [references/policy.md](references/policy.md) when creating or changing the file. `binman tools run` enforces catalog membership, enablement, checksums, and blocked subcommands.
