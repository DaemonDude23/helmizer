# Changelog

- [Changelog](#changelog)
  - [v0.7.0](#v070)
  - [v0.7.0](#v070-1)
  - [v0.5.2](#v052)
  - [v0.5.1](#v051)
  - [v0.5.0](#v050)
  - [v0.4.1](#v041)
  - [v0.4.0](#v040)

---

## [v0.7.0](https://github.com/chicken231/helmizer/releases/tag/v0.7.0)

- Catch when no `helmizer` config detected, giving a user-friendly message.
- Added `helmizer.ignore` section to helmizer config. Define path(s) to files to not ignore when constructing the final kustomization.

## [v0.7.0](https://github.com/chicken231/helmizer/releases/tag/v0.7.0)

- Reduce arguments to 1 positional argument pointing to helmizer config file.
- Fix issues with arguments referencing paths.
- Fix running subprocess _before_ generating kustomization.

## [v0.5.2](https://github.com/chicken231/helmizer/releases/tag/v0.5.2)

- Fix relative/absolute path issues when executing the script from a directory outside of what contains the `kustomization` or `helmizer.yaml`.

## [v0.5.1](https://github.com/chicken231/helmizer/releases/tag/v0.5.1)

- Fixed README

## [v0.5.0](https://github.com/chicken231/helmizer/releases/tag/v0.5.0)

- **Breaking**: Replaced CLI args for things like `resources` with YAML config file.
- Fixed `patchesStrategicMerge` example.
- Huge refactor. Much wow.

## [v0.4.1](https://github.com/chicken231/helmizer/releases/tag/v0.4.1)

- Fixed default key order when `--sort-keys=false`.

## [v0.4.0](https://github.com/chicken231/helmizer/releases/tag/v0.4.0)

- Added optional sorting of keys in flat lists.
- Added optional overriding of `apiVersion`.
- Additions to pre-commit hooks.
- Changed all optional arguments to a default of `false`.
- Upgraded PyYaml to `5.4.1`.
