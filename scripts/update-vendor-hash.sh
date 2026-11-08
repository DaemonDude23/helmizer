#!/usr/bin/env bash
# Recompute the Go vendorHash in flake.nix from go.mod/go.sum without a full
# Nix build: buildGoModule's vendor hash is the NAR hash of `go mod vendor`.
#
#   scripts/update-vendor-hash.sh           rewrite flake.nix when stale
#   scripts/update-vendor-hash.sh --check   fail when stale (pre-commit, CI)
#
# Renovate Go updates are fixed up by CI with this script (vendor-hash job).
set -euo pipefail

go_dir=src
check=false
[ "${1:-}" = --check ] && check=true

cd "$(git rev-parse --show-toplevel)"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

(cd "$go_dir" && GOFLAGS=-mod=mod go mod vendor -o "$tmp/vendor")
if [ -d "$tmp/vendor" ] && [ -n "$(ls -A "$tmp/vendor")" ]; then
  want="\"$(nix hash path "$tmp/vendor")\""
else
  # No dependencies: buildGoModule requires null.
  want=null
fi

have=$(sed -n 's/^[[:space:]]*vendorHash = \(.*\);.*/\1/p' flake.nix)
if [ -z "$have" ]; then
  echo "flake.nix has no vendorHash line" >&2
  exit 1
fi
if [ "$have" = "$want" ]; then
  exit 0
fi
if $check; then
  echo "flake.nix vendorHash is stale: have $have, want $want (run scripts/update-vendor-hash.sh)" >&2
  exit 1
fi
sed -i "s|^\([[:space:]]*vendorHash = \).*;|\1$want;|" flake.nix
echo "vendorHash: $have -> $want"
