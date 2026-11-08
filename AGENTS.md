# helmizer: agent guide

Generates kustomization.yaml for your locally-rendered YAML manifests, such as with Helm templates

This repository follows the **footprint** standard (blueprint `footprint-public-github-go`, recorded
in `.footprint.json`). Read this file before changing anything.

## Toolchain

- Everything runs from the flake devShell. Enter it with `direnv allow` (`.envrc`) or
  `nix develop`. Tool versions come from `flake.lock` only, so never install tools another
  way and never add version pins for tools to CI files.
- `make help` lists the entry points. `make check` is what CI's validate stage runs
  (`pre-commit run --all-files` + `nix flake check`).
- `nix fmt` formats the whole tree (treefmt.toml).

## Conventions

- Commit messages are short imperative sentences ("Add X", "Fix Y"), not Conventional
  Commits. Commits are GPG-signed; never bypass signing.
- User-visible changes get a line under `## [Unreleased]` in `CHANGELOG.md`
  (Keep a Changelog).
- The version lives in `src/VERSION`. `make release` tags `v$(cat src/VERSION)`, and pushing the
  tag publishes.
- Dependencies are updated by Renovate (`renovate.json`). Do not hand-edit `flake.lock`
  except with `nix flake update`.
- Files listed under `managed` in `.footprint.json` come from the footprint blueprint.
  Change them in footprint and re-render, not here, or `footprint.py drift` will flag them.

## Infrastructure

- `infra/tofu/` holds one Terragrunt unit per provider (see its README). State is local
  and gitignored.
- **Never run `terragrunt apply`/`tofu apply`.** Plans are fine (`make tofu-plan`). The
  owner applies.

## GitHub

- CI is `.github/workflows/ci.yml`. Its `ci-ok` job is the only required status check.
- Renovate runs as the Mend Renovate GitHub app (install it on the repository once).
  Dependabot security updates are turned off in `infra/tofu/github`.
- Actions are pinned to full commit SHAs with a version comment. Keep it that way:
  zizmor and Renovate's `helpers:pinGitHubActionDigests` rely on it.

## Go

- Module root: `src`. Layout: `cmd/helmizer` (main), `internal/` (everything else).
- `nix flake check` builds with `buildGoModule` and runs the unit tests. `make test` adds
  the race detector, and `make fuzz` runs the fuzz targets.
- **After any go.mod/go.sum change, run `make vendor-hash`.** Otherwise the Nix build fails
  (pre-commit checks it). CI does this automatically for Renovate branches.
- go.mod pins the minor only (`go 1.27`). The patch toolchain is nixpkgs' `go_1_27`, and a
  new minor means bumping both together.
- Lint: golangci-lint v2 (`.golangci.yml`). Suppress a reviewed finding inline with
  `//nolint:<linter> // reason`.

## helmizer

### What This Is

A Go CLI tool that generates `kustomization.yaml` files. It reads a `helmizer.yaml` config, optionally runs pre/post commands (e.g. `helm template`), walks directories to discover resource files, and renders a complete kustomization with all file paths resolved. Solves kustomize's lack of glob support.

### Project Structure

```text
src/                    # All Go source code
  main.go               # Entry point - parses args, resolves config paths, runs helmizer per config
  cli.go                # CLIArgs struct, logging setup
  helmizer.go           # Core logic: Config/Kustomization structs, command execution,
                        #   config reconciliation, file walking, kustomization writing
  utilities.go          # Helpers: YAML reading, path construction, glob/doublestar matching,
                        #   config path resolution, and the embedded version fallback
  VERSION               # Single version source: embedded via go:embed and read by flake.nix
  charts.go             # `helmizer charts check` subcommand: reports available chart updates
                        #   for Helmfile releases with a version policy and risk score
  helmfile.go           # Helmfile state loading (`helmfile build`) and release/repo parsing
  chart_repo.go         # Helm HTTP(S) repository index fetching and chart version listing
  semver.go             # SemVer parsing, comparison, and version policy/constraint matching
  utilities_test.go     # Basic tests for version output and config-glob resolution
  go.mod                # Module: github.com/DaemonDude23/helmizer, Go 1.27.3 (first release with the GO-2026-6809 fix; keep `flake.lock` nixpkgs on a rev that ships it)
Dockerfile              # Minimal scratch image with just helmizer
Dockerfile.helm         # Alpine image with helmizer + helm (built from pinned source)
action.yml              # GitHub Action definition (docker-based, uses Dockerfile.helm)
.goreleaser.yaml        # Cross-platform release builds (linux/darwin/windows, amd64/arm64/386)
.github/
  workflows/ci.yml        # footprint CI (pre-commit, flake check, race tests, cross builds) + release tooling tests + image builds; `ci-ok` is required
  workflows/security.yml  # gitleaks, trivy, zizmor, govulncheck/gosec (weekly + PRs); dependency review on PRs
  workflows/codeql.yml    # CodeQL; scorecard.yml OpenSSF Scorecard; pages.yml docs site; vendor-hash.yml Renovate vendorHash fixups
  workflows/release.yaml  # Monthly/dispatch/tag release: plan -> verify -> scan images -> goreleaser -> GHCR
renovate.json             # footprint Renovate policy + the Dockerfile.helm custom manager
examples/               # One subdirectory per kustomize feature, each with helmizer.yaml + expected output
infra/tofu/github/      # The GitHub repository's settings (footprint; plan only, the owner applies)
```

### Building and Testing

```bash
# Build locally
cd src && go build -o helmizer .

# Run tests
cd src && go test -v ./...

# Build the flake package
mkdir -p ./build/nix && nix build .#default --out-link ./build/nix/helmizer

# Docker build (minimal)
docker build --build-arg VERSION=0.21.0 -t helmizer .

# Docker build (with helm)
docker build --build-arg VERSION=0.21.0 -f Dockerfile.helm -t helmizer-helm .
```

### Key Dependencies

- `github.com/alexflint/go-arg` - CLI argument parsing
- `github.com/sirupsen/logrus` - Structured logging
- `gopkg.in/yaml.v3` - YAML marshal/unmarshal

### How It Works

1. Parse CLI args (`CLIArgs` struct in cli.go)
2. Resolve config file paths - supports positional arg or `--config-glob` with `**` doublestar patterns
3. For each config file:
   a. Read and unmarshal `helmizer.yaml` into `Config` (has `Helmizer` + `Kustomize` sections)
   b. Reconcile CLI args with config (CLI overrides config)
   c. Run pre-commands (e.g. `helm template`) with working dir set to config file's directory
   d. Build `Kustomization` struct - for `resources`/`crds`/`patchesStrategicMerge`, walks directories recursively
   e. Write `kustomization.yaml` only if content changed (byte comparison)
   f. Run post-commands

### Config File Format

Two top-level keys in `helmizer.yaml`:

- `helmizer:` - tool settings (commands, paths, flags)
- `kustomize:` - pass-through kustomization fields (namespace, resources, patches, images, etc.)

### GitHub Action

The `action.yml` defines a Docker-based action using `Dockerfile.helm`. Inputs:

- `config`: path to a single helmizer config file (default: `helmizer.yaml`)
- `config_glob`: glob pattern(s) for multiple configs (supports `**` and commas)

### Version

Current source version: `0.21.0` in `src/VERSION` (the development baseline). Go embeds it as the fallback and `flake.nix` reads it; release builds stamp `main.version` via the flake, Dockerfiles, and GoReleaser. The Release workflow writes the exact tag version into `src/VERSION` in a version-only commit carried by the tag. See `docs/automation.md`.

### Local Claude Skills

- `/update-dependencies` loads `.claude/skills/update-dependencies/SKILL.md` for local dependency and tooling refreshes.
- `/prepare-local-release` loads `.claude/skills/prepare-local-release/SKILL.md` for local version/release-file alignment.
- These skills are local-only workflows. Do not create commits, tags, pushes, pull requests, or releases when using them.

### Conventions

- Version lives only in `src/VERSION`; bump it explicitly to request a minor/major baseline. Don't reintroduce a hardcoded version in Go or Nix
- There are a few basic `_test.go` files now, but test coverage is still intentionally light
- Pre-commit hooks configured via `.pre-commit-config.yaml` (footprint standard; tools from the flake)
- YAML indentation is normalized to 2 spaces via `FixYAMLIndentation`
- Commands execute with working directory set to the helmizer config file's parent directory
- Releases are automated by `.github/workflows/release.yaml` (monthly patch, or manual dispatch with a bump). For a feature/breaking release, raise `src/VERSION` and add a `CHANGELOG.md` entry in a PR before the run. `scripts/release.sh` is legacy and commits/pushes
