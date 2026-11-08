# helmizer

Generates `kustomization.yaml` for your locally-rendered YAML manifests, such as with Helm
templates. It reads a `helmizer.yaml`, optionally runs pre/post commands (for example
`helm template`), walks the rendered files, and writes a complete kustomization, because
kustomize itself has no glob support.

The full usage guide, configuration reference and GitHub Action documentation are in the
[README](https://github.com/DaemonDude23/helmizer#readme).

## Install

```sh
nix run github:DaemonDude23/helmizer -- --help
nix profile install github:DaemonDude23/helmizer
```

Release binaries and container images are attached to each
[release](https://github.com/DaemonDude23/helmizer/releases).
