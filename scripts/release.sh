#!/usr/bin/env bash
# Usage: ./release.sh <version>
# Example: ./release.sh 0.19.2
set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "Usage: $0 <version>"
  echo "Example: $0 0.19.2"
  exit 1
fi

TAG="v${VERSION}"
NIX_OUT_LINK="./build/nix/helmizer"

# Validate semver-ish
if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Error: version must be in X.Y.Z format (got '$VERSION')"
  exit 1
fi

echo "==> Releasing $TAG"

# 1. Ensure working tree is clean
if [[ -n "$(git status --porcelain)" ]]; then
  echo "Error: working tree is dirty. Commit or stash changes first."
  git status --short
  exit 1
fi

# 2. Bump version string in utilities.go
echo "--> Updating version string in src/utilities.go"
sed -i "s/var version = \"[0-9]*\.[0-9]*\.[0-9]*\"/var version = \"${VERSION}\"/" src/utilities.go

# 3. Bump version in flake.nix
echo "--> Updating version in flake.nix"
sed -i "s/version = \"[0-9]*\.[0-9]*\.[0-9]*\";/version = \"${VERSION}\";/" flake.nix

# 4. Refresh docs/dev release examples
echo "--> Updating release examples in docs/dev.md"
sed -i "s/git tag v[0-9]*\.[0-9]*\.[0-9]*/git tag v${VERSION}/" docs/dev.md
sed -i "s/git push origin refs\\/tags\\/v[0-9]*\.[0-9]*\.[0-9]*/git push origin refs\\/tags\\/v${VERSION}/" docs/dev.md

# 5. Refresh README release references
echo "--> Updating release references in README.md"
sed -i "s|releases/download/v[0-9]*\.[0-9]*\.[0-9]*/helmizer_[0-9]*\.[0-9]*\.[0-9]*_linux_amd64.tar.gz|releases/download/${TAG}/helmizer_${VERSION}_linux_amd64.tar.gz|" README.md
sed -i "s|ghcr.io/daemondude23/helmizer/helmizer:v[0-9]*\.[0-9]*\.[0-9]*|ghcr.io/daemondude23/helmizer/helmizer:${TAG}|g" README.md
sed -i "s|ghcr.io/daemondude23/helmizer/helmizer-helm:v[0-9]*\.[0-9]*\.[0-9]*|ghcr.io/daemondude23/helmizer/helmizer-helm:${TAG}|g" README.md
sed -i "s|uses: daemondude23/helmizer@v[0-9]*\.[0-9]*\.[0-9]*|uses: daemondude23/helmizer@${TAG}|g" README.md

# 6. Recompute vendorHash
echo "--> Recomputing vendorHash"
cd src
go mod tidy
go mod vendor
VENDOR_HASH="$(nix hash path vendor/)"
cd ..
echo "    vendorHash: $VENDOR_HASH"
sed -i "s|vendorHash = \"sha256-[^\"]*\";|vendorHash = \"${VENDOR_HASH}\";|" flake.nix

# 7. Build verification
echo "--> Verifying Go and Nix builds"
go test ./src/...
mkdir -p ./build/nix
rm -f "${NIX_OUT_LINK}"
nix build .#default --out-link "${NIX_OUT_LINK}"
"${NIX_OUT_LINK}/bin/helmizer" --version

# 8. Add CHANGELOG entry
echo "--> Checking CHANGELOG.md"
if grep -q "## ${TAG}" CHANGELOG.md; then
  echo "    CHANGELOG already has entry for ${TAG}, skipping."
else
  echo ""
  echo "!!! CHANGELOG.md has no entry for ${TAG}."
  echo "    Add one before committing. Here's a template:"
  echo ""
  echo "## ${TAG}"
  echo ""
  echo "$(date +'%B %-d %Y')"
  echo ""
  echo "**Changes**"
  echo ""
  echo "- ..."
  echo ""
  read -rp "Press enter once you've added the CHANGELOG entry (or Ctrl-C to abort)..."
fi

# 9. Commit
echo "--> Committing version bump"
git add src/utilities.go src/utilities_test.go flake.nix flake.lock src/go.mod src/go.sum CHANGELOG.md docs/dev.md README.md scripts/release.sh
git commit -m "v${VERSION}"

# 10. Tag and push
echo "--> Tagging and pushing"
git fetch --prune --prune-tags
git tag "$TAG"
git push origin HEAD
git push origin "refs/tags/$TAG"

echo ""
echo "==> Done. Goreleaser CI will now build and publish the release."
