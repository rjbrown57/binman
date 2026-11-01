#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
codex_home=${CODEX_HOME:-"$HOME/.codex"}
destination="$codex_home/skills/binman-tools"
mkdir -p "$destination"
cp -R "$repo_root/skills/binman-tools/." "$destination/"
rm -f "$destination/scripts/binman_tools.py"
printf 'Installed binman-tools skill at %s\n' "$destination"
