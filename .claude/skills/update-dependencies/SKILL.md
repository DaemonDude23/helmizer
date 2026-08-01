---
name: update-dependencies
description: Update this repository's dependencies and tooling to current/latest local versions. Use when asked to refresh Go modules, Nix flake inputs, Docker base images, GitHub Actions, pre-commit hooks, Renovate-managed versions, or other dependency pins in helmizer. All work must stay local: do not create git commits, tags, pushes, pull requests, or releases.
---

# Update Dependencies

Use this skill to update helmizer dependency pins and tooling locally while preserving the user's working tree.

## Guardrails

- Do not run `git commit`, `git push`, `git tag`, release publication commands, or PR creation commands.
- Do not run `scripts/release.sh`; it commits and pushes by design.
- Check `git status --short` before editing and avoid reverting unrelated user changes.
- Prefer existing repo mechanisms: Go modules in `src/`, Nix flake files at the repo root, Renovate config in `.github/renovate.json`, and pre-commit config in `.pre-commit-config.yaml`.
- Treat network-backed update commands as local repo operations, but request approval if the environment requires it.

## Workflow

1. Inspect dependency surfaces:
   - `src/go.mod` and `src/go.sum`
   - `flake.nix` and `flake.lock`
   - `Dockerfile` and `Dockerfile.helm`
   - `.github/workflows/*.yaml`
   - `.github/renovate.json`
   - `.pre-commit-config.yaml`
   - `docs/diagrams/requirements.txt`
2. Update only the requested surfaces. If the request says "everything" or "latest", cover every surface above that is applicable.
3. For Go dependencies:
   - Run from `src/`.
   - Use `go get -u ./...` for broad updates, or targeted `go get module@latest` for scoped updates.
   - Run `go mod tidy`.
4. For Nix:
   - Run `nix flake update` to refresh `flake.lock`.
   - If Go module changes affect vendoring, refresh `vendorHash` in `flake.nix` by running `go mod vendor` in `src/`, `nix hash path src/vendor`, then remove `src/vendor` unless it was already tracked or requested.
5. For pre-commit:
   - Prefer `pre-commit autoupdate`.
   - Preserve local hook args and enabled/disabled hook choices.
6. For Dockerfiles and GitHub Actions:
   - Update pinned versions conservatively.
   - Preserve image purpose: minimal scratch image in `Dockerfile`, Alpine/Helm image in `Dockerfile.helm`.
7. Validate with the narrowest reliable set:
   - `cd src && go test ./...`
   - `nix build .#default` when Nix inputs, Go dependencies, or `flake.nix` changed.
   - `pre-commit run --all-files` when pre-commit hooks or formatting-sensitive files changed.
8. Summarize changed files, important version movements, and any validation that could not run.
