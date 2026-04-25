# Developer Docs

## Local Environment

Use the flake-based dev shell when possible:

```bash
nix develop
```

`nix-shell` still works via `shell.nix`, but `nix develop` is the primary path now that the repo ships a locked flake.

## Go Module and Dependency Updates

```bash
cd ./src/

# See available module updates without fighting the vendor directory
go list -mod=mod -m -u all

# Apply updates when needed
go get -u ./...
go mod tidy
```

## Build and Test

```bash
cd ./src/
go test ./...
go build -ldflags="-X main.version=0.19.2" -o ../build/test/helmizer .
```

Build the flake package exactly the way Nix users will consume it:

```bash
mkdir -p ./build/nix
nix build .#default --out-link ./build/nix/helmizer
./build/nix/helmizer/bin/helmizer --version
```

If built on **NixOS**, you can override the interpreter so it'll work on a conventional Linux Distribution:

```bash
patchelf --set-interpreter /lib64/ld-linux-x86-64.so.2 ../build/test/helmizer
```

## Delete All Kustomization Files

Use this when testing regeneration across the examples:

```bash
find . -type f -name kustomization.yaml -exec rm -f '{}' \;
./build/test/helmizer --config-glob "**/helmizer.yaml"
```

## Release Flow

Preferred:

```bash
git checkout -b release/v0.19.2
./scripts/release.sh prepare 0.19.2
```

After that branch is reviewed and merged to `main`:

```bash
git checkout main
git pull --ff-only
./scripts/release.sh tag 0.19.2
```

Notes:

- `./scripts/release.sh 0.19.2` is still supported and defaults to `prepare`.
- `tag` mode expects to run from `main`. Override with `MAIN_BRANCH=<branch>` if your default branch is named differently.
- The script does not publish the GitHub release for you. It pushes the tag, CI creates or updates the draft release, and you publish it manually in GitHub.

Manual equivalent:

```bash
git checkout -b release/v0.19.2
go test ./src/...
mkdir -p ./build/nix
nix build .#default --out-link ./build/nix/helmizer
./build/nix/helmizer/bin/helmizer --version
git add .
git commit -m "Prepare v0.19.2 release"
git push -u origin HEAD

# open/merge PR

git checkout main
git pull --ff-only
go test ./src/...
nix build .#default --out-link ./build/nix/helmizer
./build/nix/helmizer/bin/helmizer --version
git push origin HEAD
git tag -a v0.19.2 -m "v0.19.2"
git push origin refs/tags/v0.19.2
```
