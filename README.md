# Helmizer

- [Helmizer](#helmizer)
  - [About](#about)
  - [Usage](#usage)
  - [Configuration](#configuration)
    - [Examples](#examples)
    - [Installation](#installation)
    - [Putting it in your `$PATH`](#putting-it-in-your-path)
      - [Linux](#linux)
      - [Build Locally (Optional)](#build-locally-optional)
    - [Run](#run)
      - [Local Python](#local-python)
      - [Docker](#docker)
  - [Kustomize Options](#kustomize-options)
    - [Supported](#supported)
    - [Unsupported (Currently)](#unsupported-currently)
  - [References](#references)

---

## About

**Helmizer** takes various inputs from a YAML config file (`helmizer.yaml` by default) and constructs a [kustomization file](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/) from those inputs.

For example, instead of manually entering the paths to [`resources`](https://kubectl.docs.kubernetes.io/references/kustomize/resource/) in a kustomization file, this tool will walk any number of directories containing resources and populate the kustomization with these resources. Or only pull in individual files, it's your choice.

I began transitioning my `helm` charts to local templates via [helm template](https://helm.sh/docs/helm/helm_template/), which were then applied to the cluster separately via [Kustomize](https://kustomize.io/). I didn't enjoy having to manually manage the relative paths to files in the **kustomization**. I wanted a repeatable way for me to generate **Kubernetes** manifests from a helm chart, and tack on any patches or adjustments later. Thus, **helmizer**. **But [`Helm`](https://helm.sh/) is in no way required to make this tool useful.**

## Usage

The recommended way of using **Helmizer** is via a [YAML config file](./examples/resources/helmizer.yaml). But it can be run entirely from the CLI at parity with config. Don't combine them at the same time, though (e.g. `resources` defined in `helmizer.yaml` and CLI at the same time).

```
usage: helmizer [-h] [--debug] [--dry-run] [--helmizer-config HELMIZER_CONFIG] [--kustomization-directory KUSTOMIZATION_DIR] [--quiet] [--version]

Helmizer

optional arguments:
  -h, --help            show this help message and exit

  --debug               Enable debug logging (default: False)
  --dry-run             Do not write to a file system. (default: False)
  --helmizer-config HELMIZER_CONFIG
                        Override helmizer file path (default: None)
  --kustomization-directory KUSTOMIZATION_DIR
                        Set path containing kustomization (default: None)
  --quiet, -q           Quiet output from subprocesses (default: False)
  --version             show program's version number and exit```
```

## Configuration

Example `helmizer.yaml` config file. The `helm` command is invoked before the content for `kustomization.yaml` is generated. Any number of commands can be added here.

```yml
helmizer:
  commandSequence:
  - command: "helm"
    args:
      - "-n"
      - "sealed-secrets"
      - "template"
      - "sealed-secrets"
      - --output-dir
      - '.'
      - --include-crds
      - --skip-tests
      - --version
      - '1.12.2'
      - stable/sealed-secrets
  dry-run: false
  kustomization-directory: .
  kustomization-file-name: kustomization.yaml
  resource-absolute-paths: []
  sort-keys: true
  version: '0.1.0'
kustomize:
  namespace: sealed-secrets
  resources:
    - sealed-secrets/templates/
  patchesStrategicMerge:
    - extra/
  commonAnnotations: {}
  commonLabels: {}
```

### Examples

- [commonAnnotations](examples/commonAnnotations/)
- [commonLabels](examples/commonLabels/)
- [patchStrategicMerge](examples/patchesStrategicMerge/)
- [resources](examples/resources/)

_With [vscode](https://code.visualstudio.com/) you can utilize the included [launch.json](.vscode/launch.json) to test these more quickly, or reference for your configuration._
The `sealed-secrets` **Helm** chart is used for examples for its small scope.

### Installation

For local installation/use of the raw script, I use a local virtual environment to isolate dependencies:

**Requires use of `pip3`**

```bash
git clone https://github.com/chicken231/helmizer.git
cd helmizer
```

1. Update pip:
```bash
python3 -m pip install --upgrade pip
./venv/bin/python -m pip install --upgrade pip
```
2. Install `virtualenv` for your user:
```bash
./venv/bin/pip3 install virtualenv==20.4.2
```
3. Setup relative virtual environment:
```bash
virtualenv --python=python3.9 ./venv/
```
4. _Activate_ this virtual environment for pip3:
```bash
source ./venv/bin/activate
```
5. Install requirements into virtual environment.
```bash
pip3 install -r ./src/requirements.txt
```

If you need to reset the virtual environment for whatever reason:
```bash
virtualenv --clear ./venv/
```

### Putting it in your `$PATH`

#### Linux

```bash
sudo ln -s /absolute/path/to/src/helmizer.py /usr/local/bin/helmizer
```

```bash
pip3 install -r ./src/requirements.txt
```

#### Build Locally (Optional)

```bash
docker build -t helmizer:v0.5.2 .
```

### Run

**For greater detail on running from examples (they assumes you've ran [helm template](https://helm.sh/docs/helm/helm_template/), see the [resource example](examples/resources/README.md))**

#### Local Python

```bash
python3 ./src/helmizer.py \
  -n sealed-secrets \
  --resource-paths ./examples/resources/sealed-secrets/templates/ \
  --kustomization-directory ./examples/resources/
```

Output:
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: sealed-secrets
resources:
- sealed-secrets/templates/deployment.yaml
- sealed-secrets/templates/role-binding.yaml
- sealed-secrets/templates/service.yaml
- sealed-secrets/templates/cluster-role-binding.yaml
- sealed-secrets/templates/cluster-role.yaml
- sealed-secrets/templates/service-account.yaml
- sealed-secrets/templates/sealedsecret-crd.yaml
- sealed-secrets/templates/role.yaml
```

#### Docker

In this example (*Nix OS), we're redirecting program output to the (e.g. `kustomization.yaml`) to the desired file because of issues with UID/GID on files bind-mounted from Docker. The redirect is not required however, you can correct permissions after the fact with `sudo chown -R username:groupname .`.

```bash
docker run --name helmizer \
  --rm \
  -v "$PWD"/examples:/tmp/helmizer -w /tmp/helmizer \
  docker.pkg.github.com/chicken231/helmizer/helmizer:v0.5.2 /usr/src/app/helmizer.py \
    -n sealed-secrets \
    --resource-paths ./resources/sealed-secrets/templates/ \
    --kustomization-directory ./resources/ > ./examples/resources/kustomization.yaml
```

## Kustomize Options

### Supported

- [commonAnnotations](https://kubectl.docs.kubernetes.io/references/kustomize/commonannotations/)
- [commonLabels](https://kubectl.docs.kubernetes.io/references/kustomize/commonlabels/)
- [namespace](https://kubectl.docs.kubernetes.io/references/kustomize/namespace/)
- [patchStrategicMerge](https://kubectl.docs.kubernetes.io/references/kustomize/patchesstrategicmerge/)
- [resources](https://kubectl.docs.kubernetes.io/references/kustomize/resource/)

### Unsupported (Currently)

- [~~bases~~](https://kubectl.docs.kubernetes.io/references/kustomize/bases/)
- [components](https://kubectl.docs.kubernetes.io/references/kustomize/components/)
- [configMapGenerator](https://kubectl.docs.kubernetes.io/references/kustomize/configmapgenerator/)
- [crds](https://kubectl.docs.kubernetes.io/references/kustomize/crds/)
- [generatorOptions](https://kubectl.docs.kubernetes.io/references/kustomize/generatoroptions/)
- [images](https://kubectl.docs.kubernetes.io/references/kustomize/images/)
- [namePrefix](https://kubectl.docs.kubernetes.io/references/kustomize/nameprefix/)
- [nameSuffix](https://kubectl.docs.kubernetes.io/references/kustomize/namesuffix/)
- [patches](https://kubectl.docs.kubernetes.io/references/kustomize/patches/)
- [patchesJson6902](https://kubectl.docs.kubernetes.io/references/kustomize/patchesjson6902/)
- [replicas](https://kubectl.docs.kubernetes.io/references/kustomize/replicas/)
- [secretGenerator](https://kubectl.docs.kubernetes.io/references/kustomize/secretgenerator/)
- [vars](https://kubectl.docs.kubernetes.io/references/kustomize/vars/)

## References

- [Kustomize Docs](https://kubectl.docs.kubernetes.io/references/kustomize/)
