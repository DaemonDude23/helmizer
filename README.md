# Helmizer

- [Helmizer](#helmizer)
  - [About](#about)
  - [Usage](#usage)
  - [Configuration](#configuration)
    - [Examples](#examples)
    - [Installation](#installation)
    - [Putting it in your `$PATH`](#putting-it-in-your-path)
      - [Linux (Simplest Option)](#linux-simplest-option)
    - [virtualenv with pip](#virtualenv-with-pip)
    - [Run](#run)
      - [Local Python](#local-python)
      - [~~Docker~~](#docker)
  - [Kustomize Options](#kustomize-options)
    - [Supported](#supported)
    - [Unsupported (Currently)](#unsupported-currently)
  - [References](#references)

---

## About

**Helmizer** takes various inputs from a YAML config file (`helmizer.yaml` by default) and constructs a [kustomization file](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/) from those inputs. It can run a sequence of commands before rendering the kustomization file.

For example, instead of manually entering the paths to [`resources`](https://kubectl.docs.kubernetes.io/references/kustomize/resource/) in a kustomization file, this tool will walk any number of directories containing resources and populate the kustomization with these resources. Or only pull in individual files, it's your choice.

I began transitioning my `helm` charts to local templates via [helm template](https://helm.sh/docs/helm/helm_template/), which were then applied to the cluster separately via [Kustomize](https://kustomize.io/). I didn't enjoy having to manually manage the relative paths to files in the **kustomization**. I wanted a repeatable way for me to generate **Kubernetes** manifests from a helm chart, and tack on any patches or adjustments later. Thus, **helmizer**. **But [`Helm`](https://helm.sh/) is in no way required to make this tool useful.**

## Usage

```
usage: helmizer [-h] [--debug] [--dry-run] [--quiet] [--version] helmizer_config

Helmizer

optional arguments:
  -h, --help       show this help message and exit

  --debug          enable debug logging (default: False)
  --dry-run        do not write to a file system (default: False)
  --quiet, -q      quiet output from subprocesses (default: False)
  --version        show program's version number and exit
  helmizer_config  path to helmizer config file
```

## Configuration

- Example `helmizer.yaml` config file. The `helm` command is invoked before the content for `kustomization.yaml` is generated. Any number of commands can be added here.
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
  - command: "pre-commit"
    args:
      - 'run'
      - '-a'
      - '||'
      - 'true'
  dry-run: false
  kustomization-directory: .
  kustomization-file-name: kustomization.yaml
  resource-absolute-paths: []
  sort-keys: true
  version: '0.1.0'
  ignore:
    - sealed-secrets/templates/helmizer.yaml
kustomize:
  commonAnnotations: {}
  commonLabels: {}
  components: []
  crds: []
  namePrefix: []
  nameSuffix: []
  namespace: sealed-secrets
  patchesStrategicMerge:
    - extra/
  resources:
    - sealed-secrets/templates/
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

```bash
git clone https://github.com/chicken231/helmizer.git -b v0.8.0
cd helmizer
```

### Putting it in your `$PATH`

#### Linux (Simplest Option)

1. Create symlink:
```bash
sudo ln -s /absolute/path/to/src/helmizer.py /usr/local/bin/helmizer
```
2. Install dependencies:
```bash
pip3 install -r ./src/requirements.txt
```

### virtualenv with pip

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

### Run

**For greater detail on running from examples (they assumes you've ran [helm template](https://helm.sh/docs/helm/helm_template/), see the [resource example](examples/resources/README.md))**

#### Local Python

```bash
python3 ./src/helmizer.py ./examples/resources/helmizer.yaml
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

#### ~~Docker~~

**You may need a custom docker image depending on if you need certain apps when running commands within helmizer**

In this example (*Nix OS), we're redirecting program output to the (e.g. `kustomization.yaml`) to the desired file because of issues with UID/GID on files bind-mounted from Docker. The redirect is not required however, you can correct permissions after the fact with `sudo chown -R username:groupname .`.

```bash
docker run --name helmizer \
  --rm \
  -v "$PWD"/examples:/tmp/helmizer -w /tmp/helmizer \
  docker.pkg.github.com/chicken231/helmizer/helmizer:v0.8.0 /usr/src/app/helmizer.py \
    ./resources/ > ./examples/resources/kustomization.yaml
```

## Kustomize Options

### Supported

- [commonAnnotations](https://kubectl.docs.kubernetes.io/references/kustomize/commonannotations/)
- [commonLabels](https://kubectl.docs.kubernetes.io/references/kustomize/commonlabels/)
- [components](https://kubectl.docs.kubernetes.io/guides/config_management/components/)
- [crds](https://kubectl.docs.kubernetes.io/references/kustomize/crds/)
- [namePrefix](https://kubectl.docs.kubernetes.io/references/kustomize/nameprefix/)
- [namespace](https://kubectl.docs.kubernetes.io/references/kustomize/namespace/)
- [nameSuffix](https://kubectl.docs.kubernetes.io/references/kustomize/namesuffix/)
- [patchStrategicMerge](https://kubectl.docs.kubernetes.io/references/kustomize/patchesstrategicmerge/)
- [resources](https://kubectl.docs.kubernetes.io/references/kustomize/resource/)

### Unsupported (Currently)

- [~~bases~~](https://kubectl.docs.kubernetes.io/references/kustomize/bases/)
- [configMapGenerator](https://kubectl.docs.kubernetes.io/references/kustomize/configmapgenerator/)
- [generatorOptions](https://kubectl.docs.kubernetes.io/references/kustomize/generatoroptions/)
- [images](https://kubectl.docs.kubernetes.io/references/kustomize/images/)
- [patches](https://kubectl.docs.kubernetes.io/references/kustomize/patches/)
- [patchesJson6902](https://kubectl.docs.kubernetes.io/references/kustomize/patchesjson6902/)
- [replicas](https://kubectl.docs.kubernetes.io/references/kustomize/replicas/)
- [secretGenerator](https://kubectl.docs.kubernetes.io/references/kustomize/secretgenerator/)
- [vars](https://kubectl.docs.kubernetes.io/references/kustomize/vars/)

## References

- [Kustomize Docs](https://kubectl.docs.kubernetes.io/references/kustomize/)
