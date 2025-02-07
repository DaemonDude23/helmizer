**Changelog**

- [2025](#2025)
  - [v0.15.0](#v0150)
- [2024](#2024)
  - [v0.15.0](#v0150-1)

---

# 2025

## [v0.15.0](https://github.com/DaemonDude23/helmizer/releases/tag/v0.16.0)

Just a maintenance release with various dependency updates. No code changes.

**Housekeeping**

- Updated Go to 1.23.4.
  - Updated Go dependencies.
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
