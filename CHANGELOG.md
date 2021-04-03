# Changelog

- [Changelog](#changelog)
  - [v0.5.1](#v051)
  - [v0.5.0](#v050)
  - [v0.4.1](#v041)
  - [v0.4.0](#v040)

---

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
