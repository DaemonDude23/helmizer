#!/usr/bin/env bash
# Print the CHANGELOG.md section of one version (Keep a Changelog format),
# without its heading. Release jobs use it as the release notes.
#
#   scripts/release-notes.sh 1.2.3
set -euo pipefail

version=${1:?usage: $0 VERSION}
cd "$(git rev-parse --show-toplevel)"
notes=$(awk -v v="$version" '
  $0 ~ "^## \\[" v "\\]" { found = 1; next }
  found && /^## \[/ { exit }
  found { print }
' CHANGELOG.md)
if [ -z "${notes//[[:space:]]/}" ]; then
  echo "CHANGELOG.md has no entries for [$version]" >&2
  exit 1
fi
printf '%s\n' "$notes"
