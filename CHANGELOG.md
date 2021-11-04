# Changelog

- [Changelog](#changelog)
  - [v0.9.0](#v090)
  - [v0.8.0](#v080)
  - [v0.7.0](#v070)
  - [v0.6.0](#v060)
  - [v0.5.2](#v052)
  - [v0.5.1](#v051)
  - [v0.5.0](#v050)
  - [v0.4.1](#v041)
  - [v0.4.0](#v040)

---

## [v0.9.0](https://github.com/DaemonDude23/helmizer/releases/tag/v0.9.0)

TBD

**Enhancements:**

- Added configuration support for **all** keys in a kustomization.
  -
- Added pre-commit python formatter.

## [v0.8.0](https://github.com/DaemonDude23/helmizer/releases/tag/v0.8.0)

June 10 2021

**Enhancements:**

- Added configuration support for:
  - [`components`](https://kubectl.docs.kubernetes.io/guides/config_management/components/)
  - [`crds`](https://kubectl.docs.kubernetes.io/references/kustomize/crds/)
  - [`namePrefix`](https://kubectl.docs.kubernetes.io/references/kustomize/nameprefix/)
  - [`nameSuffix`](https://kubectl.docs.kubernetes.io/references/kustomize/namesuffix/)
- Added argument `--skip-commands` to not run the `commandSequence` and just generate the `kustomization.yaml`.
- Added more debug statements.

**Bugfixes:**

- Made argument `--dry-run` do what it says it'll do.
- Fixed `latest` tag with build script.

## [v0.7.0](https://github.com/DaemonDude23/helmizer/releases/tag/v0.7.0)

- Catch when no `helmizer` config detected, giving a user-friendly message.
- Added `helmizer.ignore` section to helmizer config. Define path(s) to files to not ignore when constructing the final kustomization.

## [v0.6.0](https://github.com/DaemonDude23/helmizer/releases/tag/v0.6.0)

- Reduce arguments to 1 positional argument pointing to helmizer config file.
- Fix issues with arguments referencing paths.
- Fix running subprocess _before_ generating kustomization.

## [v0.5.2](https://github.com/DaemonDude23/helmizer/releases/tag/v0.5.2)

- Fix relative/absolute path issues when executing the script from a directory outside of what contains the `kustomization` or `helmizer.yaml`.

## [v0.5.1](https://github.com/DaemonDude23/helmizer/releases/tag/v0.5.1)

- Fixed README

## [v0.5.0](https://github.com/DaemonDude23/helmizer/releases/tag/v0.5.0)

- **Breaking**: Replaced CLI args for things like `resources` with YAML config file.
- Fixed `patchesStrategicMerge` example.
- Huge refactor. Much wow.

## [v0.4.1](https://github.com/DaemonDude23/helmizer/releases/tag/v0.4.1)

- Fixed default key order when `--sort-keys=false`.

## [v0.4.0](https://github.com/DaemonDude23/helmizer/releases/tag/v0.4.0)

- Added optional sorting of keys in flat lists.
- Added optional overriding of `apiVersion`.
- Additions to pre-commit hooks.
- Changed all optional arguments to a default of `false`.
- Upgraded PyYaml to `5.4.1`.
