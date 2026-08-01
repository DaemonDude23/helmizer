---
name: prepare-local-release
description: Prepare helmizer release/version alignment locally without publishing. Use when Codex is asked to bump or align the helmizer version, update local release references, refresh Nix vendorHash, or prepare release files for review. All work must stay local: do not create git commits, tags, pushes, pull requests, or releases.
---

# Prepare Local Release

Use this skill to update release-related files for helmizer without performing any irreversible or remote release actions.

## Guardrails

- Do not run `git commit`, `git push`, `git tag`, `gh release`, or PR creation commands.
- Do not run `scripts/release.sh`; its prepare and tag modes commit and push.
- Check `git status --short` before editing and preserve unrelated changes.
- Require an explicit target version in `X.Y.Z` format before editing release files.

## Local Release Alignment

1. Validate the target version:
   - Accept plain semantic versions like `0.20.0`.
   - Use `vX.Y.Z` only for tag strings and documentation references that include tags.
2. Update version-bearing files:
   - `src/utilities.go`: `var version = "X.Y.Z"`
   - `flake.nix`: `version = "X.Y.Z";`
   - `README.md`: release URLs, image tags, and GitHub Action examples that reference `vX.Y.Z`
   - `docs/dev.md`: release command examples and tag/push examples
   - `CHANGELOG.md`: ensure a `## vX.Y.Z` section exists; add a minimal placeholder only when asked or when the user has not provided release notes.
3. Refresh Go/Nix release metadata when dependencies changed or Nix build requires it:
   - Run `cd src && go mod tidy`.
   - Run `cd src && go mod vendor`.
   - Run `nix hash path src/vendor`.
   - Update `vendorHash` in `flake.nix`.
   - Remove `src/vendor` after hashing unless it was already tracked or the user asked to keep it.
4. Validate:
   - `cd src && go test ./...`
   - `mkdir -p ./build/nix && nix build .#default --out-link ./build/nix/helmizer`
   - `./build/nix/helmizer/bin/helmizer --version` should print `helmizer X.Y.Z`.
5. Leave the result uncommitted and report:
   - Files changed.
   - Target version.
   - Validation results.
   - Any manual release note gaps.
