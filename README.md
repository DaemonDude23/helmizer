# Helmizer

- [Helmizer](#helmizer)
  - [About](#about)
    - [Kustomize Options](#kustomize-options)
  - [Usage](#usage)
    - [Examples](#examples)
    - [Install a local virtual environment for dependencies](#install-a-local-virtual-environment-for-dependencies)
      - [Docker](#docker)
        - [Build (Optional)](#build-optional)
        - [Run](#run)
  - [References](#references)
  - [TODO](#todo)

---

## About

Helmizer takes various CLI inputs and constructs a kustomization file from those inputs.

For example, instead of manually entering the paths to `resources` in a kustomization file, this tool will walk any number of directories containing resources and populate the kustomization with these resources.

I began transitioning my helm charts to local templates via [helm template](https://helm.sh/docs/helm/helm_template/), which were then applied to the cluster separately via [Kustomize](https://kustomize.io/).

### Kustomize Options

Supported:

- [namespace](https://kubectl.docs.kubernetes.io/references/kustomize/namespace/)
- [resources](https://kubectl.docs.kubernetes.io/references/kustomize/resource/)
- [patchStrategicMerge](https://kubectl.docs.kubernetes.io/references/kustomize/patchesstrategicmerge/)
- [commonLabels](https://kubectl.docs.kubernetes.io/references/kustomize/commonlabels/)

Currently Unsupported:

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

## Usage

### Examples

- [commonLabels](examples/commonLabels/)
- [patchStrategicMerge](examples/patchStrategicMerge/)
- [resources](examples/resources/)

With vscode you can utilize the included [launch.json](.vscode/launch.json) to test these more quickly, or reference for your configuration.

### Install a local virtual environment for dependencies

**Requires use of `pip3`**

```bash
git clone https://github.com/chicken231/helmizer.git
cd helmizer
```

1. Update pip:
```bash
python3 -m pip install --upgrade pip
```
2. Install virtualenv for your user:
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

If you need to reset the virtual environment:
```bash
virtualenv --clear ./venv/
```

#### Docker

##### Build (Optional)

```bash
docker build -t helmizer:latest .
```

##### Run

```bash
docker run --name helmizer \
  --rm \
  -v "$PWD"/examples:/usr/src/app/examples \
  -w /usr/src/app \
  helmizer:latest ./helmizer.py \
    -n sealed-secrets \
    --resource-paths ./examples/resources/sealed-secrets/templates/ \
    --kustomization-directory ./examples/resources/
```

## References

- [Kustomize Docs](https://kubectl.docs.kubernetes.io/references/kustomize/)

## TODO

- Support non-filesystem manifests.
- Config file as a possible alternative to command line args?
