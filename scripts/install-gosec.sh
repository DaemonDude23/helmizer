#!/usr/bin/env bash
# Install the pinned Linux/amd64 release binary without compiling scanner SDKs.
set -euo pipefail
version="${GOSEC_VERSION:?Set GOSEC_VERSION}"
[[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]
[[ "$(uname -s)/$(uname -m)" = Linux/x86_64 ]]
destination="${1:?Supply installation directory}"
mkdir -p "$destination"
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT
archive="gosec_${version#v}_linux_amd64.tar.gz"
base="https://github.com/securego/gosec/releases/download/$version"
curl --fail --location --retry 3 "$base/$archive" --output "$temporary/$archive"
curl --fail --location --retry 3 "$base/gosec_${version#v}_checksums.txt" --output "$temporary/checksums.txt"
(
  cd "$temporary"
  awk -v file="$archive" '$2 == file { print; found=1 } END { if (!found) exit 1 }' checksums.txt > selected-checksum.txt
  sha256sum --check selected-checksum.txt
  tar -xzf "$archive" gosec
)
install -m 0755 "$temporary/gosec" "$destination/gosec"
