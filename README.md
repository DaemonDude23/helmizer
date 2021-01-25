# Helmizer

- [Helmizer](#helmizer)
  - [About](#about)
  - [Usage](#usage)
    - [Examples](#examples)
    - [Installation](#installation)
      - [Build Locally (Optional)](#build-locally-optional)
    - [Run](#run)
      - [Local Python](#local-python)
      - [Docker](#docker)
  - [Kustomize Options](#kustomize-options)
    - [Supported](#supported)
    - [Unsupported (Currently)](#unsupported-currently)
  - [References](#references)
  - [TODO](#todo)

---

## About

Helmizer takes various CLI inputs and constructs a kustomization file from those inputs.

For example, instead of manually entering the paths to `resources` in a kustomization file, this tool will walk any number of directories containing resources and populate the kustomization with these resources. Or only pull in individual files, it's your choice.

I began transitioning my helm charts to local templates via [helm template](https://helm.sh/docs/helm/helm_template/), which were then applied to the cluster separately via [Kustomize](https://kustomize.io/). I didn't enjoy having to manually manage the relative paths to files in the **kustomization**. I wanted a repeatable way for me to generate **Kubernetes** manifests from a helm chart, and tack on any patches or adjustments later. Thus, **helmizer**. **But `Helm` is in no way required to make this tool useful.**

## Usage

```bash
usage: helmizer [-h] [--common-labels [COMMON_LABELS ...]] [--debug] [--dry-run] [--kustomization-file-name KUSTOMIZATION_FILE_NAME] [--namespace NAMESPACE]
                [--patches-strategic-merge-paths [PATCHES_STRATEGIC_MERGE_PATHS ...]] [--resource-paths [RESOURCE_PATHS ...]]
                [--resource-absolute-paths [RESOURCE_ABSOLUTE_PATHS ...]] --kustomization-directory KUSTOMIZATION_DIRECTORY [--version]

Helmizer

optional arguments:
  -h, --help            show this help message and exit

  --common-labels [COMMON_LABELS ...]
                        Common Labels where `=` is the assignment operator e.g key=value (default: None)
  --debug               Enable debug logging (default: False)
  --dry-run             Do not write to a file system. (default: True)
  --kustomization-file-name KUSTOMIZATION_FILE_NAME
                        options: `kustomization.yaml`, kustomization.yml, `Kustomization` (default: kustomization.yaml)
  --namespace NAMESPACE, -n NAMESPACE
                        Specify namespace in kustomization (default: None)
  --patches-strategic-merge-paths [PATCHES_STRATEGIC_MERGE_PATHS ...]
                        Path(s) to patch directories or files patchStrategicMerge (default: None)
  --resource-paths [RESOURCE_PATHS ...]
                        Path(s) to resource directories or files (default: None)
  --resource-absolute-paths [RESOURCE_ABSOLUTE_PATHS ...]
                        TODO (default: None)
  --kustomization-directory KUSTOMIZATION_DIRECTORY
                        Path to directory to contain the kustomization file (default: None)
  --version             show program's version number and exit
```

### Examples

- [commonLabels](examples/commonLabels/)
- [patchStrategicMerge](examples/patchStrategicMerge/)
- [resources](examples/resources/)

_With vscode you can utilize the included [launch.json](.vscode/launch.json) to test these more quickly, or reference for your configuration._
The `sealed-secrets` **Helm** chart is used for examples for it's small scope.

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
```
2. Install `virtualenv` for your user:
```bash
pip3 install virtualenv==20.0.33
```
3. Setup relative virtual environment:
```bash
virtualenv --python=python3.9 ./venv/
```
4. 'Activate' this virtual environment for pip3:
```bash
source ./venv/bin/activate
```
5. Install requirements into virtual environment.
```bash
pip3 install --use-feature=2020-resolver -r ./src/requirements.txt
```

If you need to reset the virtual environment for whatever reason:
```bash
virtualenv --clear ./venv/
```

#### Build Locally (Optional)

```bash
docker build -t helmizer:latest .
```

### Run

**For greater detail on running from examples (they assumes you've ran `helm template`, see the [resource example](examples/resources/README.md))**

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
  -v "$PWD"/examples:/tmp/helmizer \
  -w /tmp/helmizer \
  docker.pkg.github.com/chicken231/helmizer/helmizer:v0.2.0 /usr/src/app/helmizer.py \
    -n sealed-secrets \
    --resource-paths ./resources/sealed-secrets/templates/ \
    --kustomization-directory ./resources/ > ./examples/resources/kustomization.yaml
```

## Kustomize Options

### Supported

- [namespace](https://kubectl.docs.kubernetes.io/references/kustomize/namespace/)
- [resources](https://kubectl.docs.kubernetes.io/references/kustomize/resource/)
- [patchStrategicMerge](https://kubectl.docs.kubernetes.io/references/kustomize/patchesstrategicmerge/)
- [commonLabels](https://kubectl.docs.kubernetes.io/references/kustomize/commonlabels/)

### Unsupported (Currently)

- [bases](https://kubectl.docs.kubernetes.io/references/kustomize/bases/)
- [commonAnnotations](https://kubectl.docs.kubernetes.io/references/kustomize/commonannotations/)
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

## TODO

- Support:
  - Non-filesystem manifests.
  - Add 'quiet' flag to guarantee no extra output.
  - Support additional Kustomizations.
  - Config file as a possible alternative to command line args?
  - Optional sorting of keys.
- Redirect examples to not add an _extra_ newline.
- Snap package?
- Homebrew?
- Link to docs for how to auth to pull from GitHub's container registry.
