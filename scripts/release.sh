#!/usr/bin/env bash
# Usage:
#   ./release.sh <version>
#   ./release.sh prepare <version>
#   ./release.sh tag <version>
#
# Examples:
#   ./release.sh 0.19.2
#   ./release.sh prepare 0.19.2
#   ./release.sh tag 0.19.2
set -euo pipefail

usage() {
  echo "Usage:"
  echo "  $0 <version>"
  echo "  $0 prepare <version>"
  echo "  $0 tag <version>"
  echo ""
  echo "Examples:"
  echo "  $0 0.19.2"
  echo "  $0 prepare 0.19.2"
  echo "  $0 tag 0.19.2"
}

if [[ $# -eq 1 ]]; then
  MODE="prepare"
  VERSION="$1"
elif [[ $# -eq 2 ]]; then
  MODE="$1"
  VERSION="$2"
else
  usage
  exit 1
fi

case "${MODE}" in
  prepare|tag)
    ;;
  *)
    echo "Error: mode must be 'prepare' or 'tag' (got '${MODE}')"
    usage
    exit 1
    ;;
esac

if ! [[ "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Error: version must be in X.Y.Z format (got '${VERSION}')"
  exit 1
fi

TAG="v${VERSION}"
MAIN_BRANCH="${MAIN_BRANCH:-main}"
CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
NIX_OUT_LINK="./build/nix/helmizer"

ensure_clean() {
  if [[ -n "$(git status --porcelain)" ]]; then
    echo "Error: working tree is dirty. Commit or stash changes first."
    git status --short
    exit 1
  fi
}

verify_changelog() {
  echo "--> Checking CHANGELOG.md"
  if grep -q "## ${TAG}" CHANGELOG.md; then
    echo "    CHANGELOG already has entry for ${TAG}, continuing."
    return
  fi

  echo ""
  echo "!!! CHANGELOG.md has no entry for ${TAG}."
  echo "    Add one before continuing. Here's a template:"
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
}

update_release_files() {
  echo "--> Updating version string in src/utilities.go"
  sed -i "s/var version = \"[0-9]*\.[0-9]*\.[0-9]*\"/var version = \"${VERSION}\"/" src/utilities.go

  echo "--> Updating version in flake.nix"
  sed -i "s/version = \"[0-9]*\.[0-9]*\.[0-9]*\";/version = \"${VERSION}\";/" flake.nix

  echo "--> Updating release examples in docs/dev.md"
  sed -i "s/prepare v[0-9]*\.[0-9]*\.[0-9]*/prepare v${VERSION}/" docs/dev.md || true
  sed -i "s/tag v[0-9]*\.[0-9]*\.[0-9]*/tag v${VERSION}/" docs/dev.md || true
  sed -i "s/prepare [0-9]*\.[0-9]*\.[0-9]*/prepare ${VERSION}/" docs/dev.md
  sed -i "s/tag [0-9]*\.[0-9]*\.[0-9]*/tag ${VERSION}/" docs/dev.md
  sed -i "s/git tag v[0-9]*\.[0-9]*\.[0-9]*/git tag v${VERSION}/" docs/dev.md
  sed -i "s/git push origin refs\\/tags\\/v[0-9]*\.[0-9]*\.[0-9]*/git push origin refs\\/tags\\/v${VERSION}/" docs/dev.md

  echo "--> Updating release references in README.md"
  sed -i "s|releases/download/v[0-9]*\.[0-9]*\.[0-9]*/helmizer_[0-9]*\.[0-9]*\.[0-9]*_linux_amd64.tar.gz|releases/download/${TAG}/helmizer_${VERSION}_linux_amd64.tar.gz|" README.md
  sed -i "s|ghcr.io/daemondude23/helmizer/helmizer:v[0-9]*\.[0-9]*\.[0-9]*|ghcr.io/daemondude23/helmizer/helmizer:${TAG}|g" README.md
  sed -i "s|ghcr.io/daemondude23/helmizer/helmizer-helm:v[0-9]*\.[0-9]*\.[0-9]*|ghcr.io/daemondude23/helmizer/helmizer-helm:${TAG}|g" README.md
  sed -i "s|uses: daemondude23/helmizer@v[0-9]*\.[0-9]*\.[0-9]*|uses: daemondude23/helmizer@${TAG}|g" README.md
}

recompute_vendor_hash() {
  echo "--> Recomputing vendorHash"
  cd src
  go mod tidy
  go mod vendor
  VENDOR_HASH="$(nix hash path vendor/)"
  cd ..
  echo "    vendorHash: $VENDOR_HASH"
  sed -i "s|vendorHash = \"sha256-[^\"]*\";|vendorHash = \"${VENDOR_HASH}\";|" flake.nix
}

verify_builds() {
  echo "--> Verifying Go and Nix builds"
  go test ./src/...
  mkdir -p ./build/nix
  rm -f "${NIX_OUT_LINK}"
  nix build .#default --out-link "${NIX_OUT_LINK}"

  BUILT_VERSION="$("${NIX_OUT_LINK}/bin/helmizer" --version)"
  echo "${BUILT_VERSION}"
  if [[ "${BUILT_VERSION}" != "helmizer ${VERSION}" ]]; then
    echo "Error: built version mismatch (got '${BUILT_VERSION}', expected 'helmizer ${VERSION}')"
    exit 1
  fi
}

prepare_release() {
  echo "==> Preparing ${TAG} on branch '${CURRENT_BRANCH}'"
  ensure_clean
  update_release_files
  recompute_vendor_hash
  verify_builds
  verify_changelog

  echo "--> Committing release prep if needed"
  git add src/utilities.go src/utilities_test.go flake.nix flake.lock src/go.mod src/go.sum CHANGELOG.md docs/dev.md README.md scripts/release.sh
  if git diff --cached --quiet; then
    echo "    No tracked release changes to commit."
  else
    git commit -m "Prepare ${TAG} release"
  fi

  echo "--> Pushing current branch"
  git push origin HEAD

  echo ""
  echo "==> Release prep is done."
  echo "Next:"
  echo "  1. Open/merge the release branch into ${MAIN_BRANCH}"
  echo "  2. Checkout ${MAIN_BRANCH} locally and pull latest"
  echo "  3. Run: $0 tag ${VERSION}"
}

tag_release() {
  echo "==> Tagging ${TAG} from branch '${CURRENT_BRANCH}'"
  ensure_clean

  if [[ "${CURRENT_BRANCH}" != "${MAIN_BRANCH}" ]]; then
    echo "Error: tag mode must run from '${MAIN_BRANCH}' (current branch: '${CURRENT_BRANCH}')"
    echo "If your default branch has a different name, set MAIN_BRANCH=<branch>."
    exit 1
  fi

  verify_changelog
  verify_builds

  echo "--> Fetching latest refs and tags"
  git fetch --prune --prune-tags origin

  if git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null; then
    echo "Error: tag '${TAG}' already exists locally."
    exit 1
  fi
  if git ls-remote --exit-code --tags origin "refs/tags/${TAG}" >/dev/null 2>&1; then
    echo "Error: tag '${TAG}' already exists on origin."
    exit 1
  fi

  echo "--> Pushing ${MAIN_BRANCH}"
  git push origin HEAD

  echo "--> Creating annotated tag"
  git tag -a "${TAG}" -m "${TAG}"
  git push origin "refs/tags/${TAG}"

  echo ""
  echo "==> Tag pushed. GitHub Actions should create/update the draft release."
  echo "Final manual step: open the draft release in GitHub, review it, and publish it."
}

case "${MODE}" in
  prepare)
    prepare_release
    ;;
  tag)
    tag_release
    ;;
esac
