# Helmizer

## What This Is

A Go CLI tool that generates `kustomization.yaml` files. It reads a `helmizer.yaml` config, optionally runs pre/post commands (e.g. `helm template`), walks directories to discover resource files, and renders a complete kustomization with all file paths resolved. Solves kustomize's lack of glob support.

## Project Structure

```
src/                    # All Go source code
  main.go               # Entry point - parses args, resolves config paths, runs helmizer per config
  cli.go                # CLIArgs struct, logging setup
  helmizer.go           # Core logic: Config/Kustomization structs, command execution,
                        #   config reconciliation, file walking, kustomization writing
  utilities.go          # Helpers: YAML reading, path construction, glob/doublestar matching,
                        #   config path resolution
  go.mod                # Module: daemondude23/helmizer/m, Go 1.25
Dockerfile              # Minimal scratch image with just helmizer
Dockerfile.helm         # Scratch image with helmizer + helm binary (from alpine/helm)
action.yml              # GitHub Action definition (docker-based, uses Dockerfile.helm)
.goreleaser.yaml        # Cross-platform release builds (linux/darwin/windows, amd64/arm64/386)
.github/
  workflows/release.yaml  # CI: test -> goreleaser + docker build on tag push
  renovate.json           # Renovate config for Dockerfile, gomod, helm, kustomize, GH Actions
examples/               # One subdirectory per kustomize feature, each with helmizer.yaml + expected output
```

## Building and Testing

```bash
# Build locally
cd src && go build -o helmizer .

# Run tests
cd src && go test -v ./...

# Docker build (minimal)
docker build -t helmizer .

# Docker build (with helm)
docker build -f Dockerfile.helm -t helmizer-helm .
```

## Key Dependencies

- `github.com/alexflint/go-arg` - CLI argument parsing
- `github.com/sirupsen/logrus` - Structured logging
- `gopkg.in/yaml.v3` - YAML marshal/unmarshal

## How It Works

1. Parse CLI args (`CLIArgs` struct in cli.go)
2. Resolve config file paths - supports positional arg or `--config-glob` with `**` doublestar patterns
3. For each config file:
   a. Read and unmarshal `helmizer.yaml` into `Config` (has `Helmizer` + `Kustomize` sections)
   b. Reconcile CLI args with config (CLI overrides config)
   c. Run pre-commands (e.g. `helm template`) with working dir set to config file's directory
   d. Build `Kustomization` struct - for `resources`/`crds`/`patchesStrategicMerge`, walks directories recursively
   e. Write `kustomization.yaml` only if content changed (MD5 comparison)
   f. Run post-commands

## Config File Format

Two top-level keys in `helmizer.yaml`:
- `helmizer:` - tool settings (commands, paths, flags)
- `kustomize:` - pass-through kustomization fields (namespace, resources, patches, images, etc.)

## GitHub Action

The `action.yml` defines a Docker-based action using `Dockerfile.helm`. Inputs:
- `config`: path to a single helmizer config file (default: `helmizer.yaml`)
- `config_glob`: glob pattern(s) for multiple configs (supports `**` and commas)

## Version

Current version: `0.18.0` (hardcoded in utilities.go:18 `Version()` method). Update there + Dockerfiles + README on release.

## Conventions

- No tests exist yet in the source (test step in CI runs `go test ./...` but no `_test.go` files)
- Version string lives in `src/utilities.go` `Version()` method
- Pre-commit hooks configured via `.pre-commit-config.yaml`
- YAML indentation is normalized to 2 spaces via `FixYAMLIndentation`
- Commands execute with working directory set to the helmizer config file's parent directory
