**Changelog**

- [2026](#2026)
  - [v0.20.0](#v0200)
  - [v0.19.2](#v0192)
  - [v0.19.1](#v0191)
  - [v0.19.0](#v0190)
  - [v0.18.0](#v0180)
- [2025](#2025)
  - [v0.17.0](#v0170)
  - [v0.16.0](#v0160)
- [2024](#2024)
  - [v0.15.0](#v0150)

---

# 2026

## v0.20.0

August 1 2026

**Features**

- Added the `helmizer charts check` subcommand: reads Helmfile state via `helmfile build`, resolves HTTP(S) chart repositories, and reports available chart updates per release with a version policy (`same-major`, `same-minor`, `all`, or `constraint`), a simple risk score, and `table`/`markdown`/`json`/`yaml` output. `--fail-on-update` exits with status 10 for CI gating.
- The earlier experimental `charts diff` and `charts review` subcommands were dropped before release; Helm and Helmfile already cover value/manifest comparison (`helm show values`, `helmfile diff`), so Helmizer stops at surfacing which updates exist.

**Dependencies**

- Updated Go module dependencies, the Nix flake inputs and `vendorHash`, pre-commit hook revisions, Docker base images (`golang` 1.26.5, `alpine` 3.24.1, `alpine/helm` 4.2.3), and GitHub Actions (`actions/checkout@v7`, `actions/setup-go@v7`).

**Docs/Tests**

- Documented `charts check` in the README, refreshed VSCode launch/tasks entries, and added tests for chart version selection, policy constraints, Helmfile chart resolution, and the `charts check` launch configuration.

**Release Tooling**

- Reworked `scripts/release.sh` into two explicit modes: `prepare <version>` on a release branch and `tag <version>` from `main`.
- Fixed the build verification step, which ran `go test ./src/...` from the repo root and always failed because the Go module lives in `src/` with no root `go.mod`. It now runs `(cd src && go test ./...)`; the same correction was applied to the manual steps in `docs/dev.md`.

## v0.19.2

April 17 2026

**Fixes**

- Made release versioning consistent across `helmizer --version`, GoReleaser archives, Docker images, and the Nix flake by stamping builds with `main.version`.
- Committed `flake.lock`, updated the flake to build from the repo root with `modRoot = "src"`, and ignored the local Nix `result` symlink.

**Dependencies**

- Kept the module `go` directive aligned with the Go toolchain currently available in `nixpkgs` (`1.26.1`) while CI and Docker release builds stay on `1.26.2`.
- Verified the direct Go dependencies are already current for this release.

**Docs/Tests**

- Updated README install snippets, container tags, GitHub Action refs, GitLab CI examples, and developer docs for `v0.19.2`.
- Added tests covering version output and `--config-glob` path resolution behavior.

## v0.19.1

April 9 2026

**Fixes**

- Relaxed go.mod so the Nix Flake would function.

## v0.19.0

April 9 2026

**Dependencies**

- Go 1.26.2, `golang.org/x/sys` v0.43.0, `alpine/helm` 4.1.4.

**Fixes**

- Fixed duplicate `SkipPostCommands` reconcile block, nil pointer dereference in `RenameHelmizerKeys`, wrong error variable in `ReadYamlFile`, and `TextFormatter` not applied at TRACE/DEBUG log levels.

**Docs/Misc**

- Added Nix flake for installing via `nix profile install` or NixOS configuration.
- Fixed `Dockerfile.helm` source image reference in README.

---

## [v0.18.0](https://github.com/DaemonDude23/helmizer/releases/tag/v0.18.0)

January 20 2025

**Enhancements**

- Added `buildMetadata`, `helmCharts`, and `labels` support in generated kustomization output, plus examples for each.
- Added `--config-glob` flag so that you don't need to use other recursive tools like `find`.

**Fixes**

- Fixed `labels` typing and the `kustomizationPath` config key so configs load and render correctly.
- Added CA certificates to scratch images for TLS support.

**Docs**

- Updated README configuration examples, `docker` usage, and example lists (including `patchesStrategicMerge` paths).

**Housekeeping**

- Updated CI tooling (checkout `v6`), Go versions (Docker builder + CI to `1.26.1`, module `go` to `1.25`), and release CI now publishes the `helmizer-helm` image.
- Updated pre-commit `mypy` to `v1.19.1` and `diagrams` to `0.25.1`.
- Added VS Code launch entries for `buildMetadata`, `helmCharts`, and `labels` examples.
- Updated examples with latest `cert-manager` chart version.

# 2025

## [v0.17.0](https://github.com/DaemonDude23/helmizer/releases/tag/v0.17.0)

November 4 2025

Just a maintenance release with various dependency updates. No code changes.

**Housekeeping**

- Updated Go to `1.25.3`.
  - Updated Go dependencies.
- Updated Python from `3.12` to `3.13` version and dependencies (just for the diagrams).
- Updated pre-commit hook versions.

## [v0.16.0](https://github.com/DaemonDude23/helmizer/releases/tag/v0.16.0)

February 10 2025

Just a maintenance release with various dependency updates. No code changes.

**Housekeeping**

- Updated Go to `1.23.4`.
  - Updated Go dependencies.
- Added a Dockerfile, testing with docker, and docs for copying helmizer out of a container.
- Removed old Python changelog.
- Removed `asdf` environment variables from `launch.json`.

# 2024

## [v0.15.0](https://github.com/DaemonDude23/helmizer/releases/tag/v0.15.0)

April 27 2024

_Re-written in Golang!_

**Breaking Changes**

- The syntax of the config file is different; now using `camelCase`.

**Enhancements**

- Added the ability to run arbitrary commands (`postCommands`) _after_ rendering the `kustomization.yaml` file.
- Increased speed.
- Easier to install than with **Python**. A smaller file as well.
- Removed some unneeded configuration keys.

**Housekeeping**

- Changed license to **Apache 2.0** since this is a _complete_ rewrite.
- Moved the Python Changelog to its own file.
- Added the use of `helmfile.yaml` in some examples.
- Added a diagram to show an example flow, at least for how I use **helmizer**.
