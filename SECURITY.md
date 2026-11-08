# Security policy

## Supported versions

Only the latest release receives security fixes.

## Reporting a vulnerability

Report vulnerabilities privately through
[GitHub security advisories](https://github.com/DaemonDude23/helmizer/security/advisories/new). Please don't open a
public issue.

Include the affected version, a description of the impact, and steps to reproduce. Expect
an acknowledgement within a week. Fixes ship in a patch release, and the advisory credits
you unless you ask otherwise.

## How the project stays patched

- Renovate keeps every dependency current, including the whole toolchain through
  `flake.lock`, and opens security updates without the usual release-age delay.
- A weekly scheduled workflow runs gitleaks, trivy and zizmor, plus language scanners
  (govulncheck/gosec for Go, pip-audit for Python), against the unchanged tree.
