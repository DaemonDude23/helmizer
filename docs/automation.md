# CI, maintenance, and releases

These files configure GitHub-hosted workflows and Renovate only; no runner, GitHub setting, or cloud resource is provisioned by them. Nix flakes only see Git-tracked files, so a new file such as `src/VERSION` must be committed or added with `git add --intent-to-add` before `nix build` can read it.

## Cost and cadence

Standard GitHub-hosted runners are free for public repositories. Private repositories consume the owner's included minutes. See [GitHub Actions billing](https://docs.github.com/en/billing/concepts/product-billing/github-actions). This configuration uses Linux runners, bounded job timeouts, Go caches, and BuildKit reuse of intermediate layers within each release job. No routine artifact uploads are retained.

- CI: one job per PR update and main push. Go changes run formatting, module checks, vet, race tests, cross compilation, and govulncheck. Workflow/script changes run workflow and shell lint. Go module/Nix changes run Nix syntax and vendor hash checks. Dockerfile, `.dockerignore`, and Go module changes build both images for the runner's architecture, so auto-merged base image bumps are tested before merge. Documentation-only PRs keep the required check present and run the small release automation test suite, skipping Go tool setup, compilation, scanners, and Nix installation. Main pushes and manual runs execute the full suite. Superseded runs cancel.
- Security: PR govulncheck is included in the Go checks job; govulncheck, gosec, and Trivy run weekly on Monday at 06:17 UTC or manually. Dependency review runs for public-repository PRs; private repository entitlement must be checked before enabling it there.
- Renovate: routine updates before 06:00 Monday in America/Los_Angeles; at most three concurrent PRs. Vulnerability alerts bypass the schedule. Patch/minor/digest maintenance can auto-merge after required checks. Major, Go module, and Nix input updates require review.
- Release: first day of each month at 10:43 UTC, or manual dispatch. Default patch increment, optional minor/major dispatch. Only source/dependency/image changes since the latest stable tag produce a release. Documentation-only changes do not. Schedules are best-effort and run from the default branch.

For budgeting, start with an illustrative estimate: 40 CI runs × 5 minutes + 4 weekly scan runs × 10 aggregate job minutes + one 30-minute release = 270 minutes/month. These are planning assumptions, not measured runtime; initial cache misses and multi-platform builds can cost more. Check actual usage after the first month. Other repositories share the private-repository allowance. Heavy scans also have explicit job timeouts.

## Required repository settings

Before merging, install/enable Renovate and enable the dependency graph and vulnerability alerts. Protect main and require `Go checks` and `Dependency review` (public repositories). `Go checks` includes PR govulncheck; the standalone weekly govulncheck job is not a required PR check. Enable GitHub auto-merge if using Renovate platform auto-merge. Never enable `ignoreTests`. Review bot access and any branch rules requiring reviews: they can prevent automatic merges. Weekly scanner failures should be triaged before approving maintenance; scheduled jobs are not automatically PR required checks.

Go module PRs remain reviewed because hosted Renovate cannot generally run arbitrary post-upgrade commands. Refresh the Nix hash locally:

```bash
(cd src && go mod tidy && go mod vendor)
nix hash path src/vendor
# Copy the result into flake.nix's vendorHash.
rm -rf src/vendor
nix build .#default --no-link
```

## Release behavior

`src/VERSION` is the single version source consumed by Go embedding and Nix. Automated releases update it to the selected tag version in a small commit on detached HEAD, containing only this version edit atop the selected default-branch commit. The tag carries that commit; `main` is never pushed or modified by the release workflow. Source, Nix, binaries, and images at that tag all report the same version. Ad-hoc builds from `main` continue to report its development baseline. Advance the baseline explicitly when preparing a feature release.

Release planning only accepts canonical stable `vX.Y.Z` tags on default-branch history, or release metadata commits changing only `src/VERSION` atop that history with a version matching the tag. It takes the newest valid stable tag, skips unchanged release inputs, and increments the selected component, respecting the development baseline as a minimum. The release-only version difference is excluded from change detection so it cannot cause empty monthly releases. An explicit development baseline increase still requests a release. With no stable tag it uses the baseline. A tag for the same source commit is retried unless its GitHub release is already published. To retry a partial release, dispatch the workflow using that existing tag as the ref. Do not move tags. Manual tags must already have a matching `src/VERSION`; use the explicit preparation path before creating them.

The publisher runs Go tests, vet, govulncheck, and a Nix build; builds and scans every image architecture once and records archive checksums; commits the tested version edit and creates/verifies the tag; stages binaries/checksums and GitHub-generated notes in a draft; loads/pushes the exact saved images and creates both versioned image manifests from their immutable digests; promotes both `latest` tags from those same digests; and publishes the release. The same workflow handles scheduled and tag releases, avoiding the `GITHUB_TOKEN` tag-trigger limitation. Full Action SHAs and Docker base image digests are pinned and Renovate-managed. Gosec uses its checksummed release binary to avoid compiling its large scanner dependency tree each week.

Publishing across GitHub and GHCR is not transactional. A failure can leave a draft, uploaded assets, a versioned image, or one promoted `latest` tag. Rerun the same tag to finish. Published releases are skipped; newer tags cannot be replaced by older releases. Retries replace duplicate draft assets, and divergent tag history stops release planning rather than promoting an older version. Scanner databases can change between reruns. High/critical vulnerabilities with available fixes block release; unfixed CVEs are currently omitted from failure criteria and need separate triage.

The scratch image defaults to UID/GID 65532. Ensure its output directory is writable, or explicitly choose the owning UID when running it. The Helm/GitHub Action image retains its existing user behavior for workspace compatibility.

The legacy `scripts/release.sh` still commits and pushes when explicitly invoked. It is not a local-only preview tool. Its pushed tags now trigger automatic publication after validation; manual draft publication is no longer required.

## Configuration trust

Helmizer configurations can run pre/post commands and Helmfile can execute template helpers. Treat config files as local executable code. Use `--skip-commands` to disable Helmizer pre/post commands when appropriate; this does not sandbox Helmfile template evaluation. Gosec suppressions are limited to those intended subprocess/file-path operations and conventional readable manifest permissions, with reasons inline. Output manifests can contain secret literals; control their directory/file permissions when using that feature.

Trivy exceptions in `.trivyignore.yaml` apply only to the root user required by the GitHub Docker action and a priority-class-only strategic merge patch. They do not suppress dependency CVEs or secret findings.

Local image validation found fixed high-severity advisories in the previous Go 1.26.5 builder, Helm 4.2.3, and Alpine 3.24.1 packages. The pins were updated to Go 1.26.9 (standard library fixes), Helm 4.3.0, and Alpine 3.24.2; the Helm runtime also applies available Alpine package upgrades during its build. Image scan gates operate on dependency inventory and can flag vulnerable libraries even when govulncheck finds no reachable vulnerable symbols in Helmizer.

## Image publishing details

Release image publication uses `scripts/release-images.sh`: all five platform images must pass scanning, and their saved archives must still match recorded checksums, before any image is pushed. Publication does not rebuild images or rely on GitHub cache authentication hidden from shell steps. BuildKit shares native builder layers between platforms and variants inside the single job. Architecture-specific version tags are also retained in GHCR, with the two normal versioned multi-platform tags and `latest` tags as the user-facing interface.

Trivy scans mount the source/image archives read-only, use a dedicated writable temporary database directory, and run as the runner's user. The database is shared across image scans in the release job. Weekly scans download the current database directly; there are no large database cache uploads to GitHub storage. This also keeps scanner-generated files outside the checkout so GoReleaser's clean-tree check can pass.

The monthly default is a patch release; it cannot judge semantic compatibility. Dispatch a minor/major release explicitly for new features or breaking changes before the scheduled patch run. Release tags carry a version-only metadata commit when needed, so tagged source/Nix builds and published binaries/images all report the selected tag exactly.

Release-stage tests invoke a fake Docker CLI and verify scan failures, changed archives, missing archives, and version changes stop publication, as well as checking that publishing loads saved images and never rebuilds them. These tests make no registry writes. Real GitHub API publication and ARM container execution still require a future hosted workflow run.
