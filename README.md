**Helmizer**

- [About](#about)
- [Usage](#usage)
- [Configuration](#configuration)
  - [Installation](#installation)
  - [Putting it in your `$PATH`](#putting-it-in-your-path)
    - [Linux (Simplest Option)](#linux-simplest-option)
  - [virtualenv with pip](#virtualenv-with-pip)
  - [Run](#run)
    - [Local Python](#local-python)
    - [~~Docker~~](#docker)
  - [Examples](#examples)
- [Kustomize Options](#kustomize-options)

---

# About

**TLDR**

Generates a `kustomization.yaml` file, optionally providing the ability to run commands (e.g. `helm template`) on your OS prior to generating a kustomization, and will compose the kustomization fields that deal with file paths (e.g. `resources`) with glob-like features, as well as pass-through all other kustomization configuration properties. No need to explicitly enumerate every file to be 'kustomized' individually.

It takes a config file as input, telling **Helmizer** if you want to run any commands. Then if you give it one or more directories for `resources`, for example, it will recursively lookup all of those files and render them into your kustomization.yaml. Want to skip including one file like `templates/secret.yaml`? Just add the relative path to `helmizer.ignore` to `helmizer.yaml`.

---

[Thou shall not _glob_](https://github.com/kubernetes-sigs/kustomize/issues/3205), said the **kustomize** developers (thus far).

**Helmizer** takes various inputs from a YAML config file (`helmizer.yaml` by default) and constructs a [kustomization file](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/) from those inputs. It can run a sequence of commands before rendering the kustomization file.

Instead of manually entering the paths to [`resources`](https://kubectl.docs.kubernetes.io/references/kustomize/resource/) in a kustomization file, this tool will walk any number of directories containing resources and populate the kustomization with these resources. Or only pull in individual files, it's your choice.

I began transitioning my `helm` charts to local manifests via [`helm template`](https://helm.sh/docs/helm/helm_template/), which were then applied to the cluster separately via [Kustomize](https://kustomize.io/). I didn't enjoy having to manually manage the relative paths to files in the **kustomization**. I wanted a repeatable process to generate **Kubernetes** manifests from a helm chart, _and_ tack on any patches or related resources later with a single command. Thus, **helmizer**. **But [Helm](https://helm.sh/) is in no way required to make this tool useful** - have it walk your raw manifests as well. This is just a wrapper that allows combining steps, and glob-like behavior, to managing `kustomization.yaml` files.

# Usage

```
usage: helmizer [-h] [--debug] [--dry-run] [--skip-commands] [--no-sort-keys] [--quiet] [--version] helmizer_config

Helmizer

optional arguments:
  -h, --help       show this help message and exit

  --debug          enable debug logging (default: False)
  --dry-run        do not write to a file system (default: False)
  --skip-commands  skip executing commandSequence, just generate kustomization file (default: False)
  --no-sort-keys   disables alphabetical sorting of keys in output kustomization file (default: False)
  --quiet, -q      quiet output from subprocesses (default: False)
  --version, -v    show program's version number and exit
  helmizer_config  path to helmizer config file
```

# Configuration

- Example `helmizer.yaml` config file. The `helm` command is invoked before the content for `kustomization.yaml` is generated. Any number of commands can be added here.
```yaml
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
  sort-keys: true
  version: '0.1.0'
  ignore: []
kustomize:
  commonAnnotations: {}
  commonLabels: {}
  configMapGenerator: []
  crds: []
  generatorOptions: {}
  images: []
  namePrefix: []
  namespace: ""
  nameSuffix: []
  openapi: {}
  patchesJson6902: []
  patchesStrategicMerge: []
  replacements: []
  replicas: []
  resources: []
  secretGenerator: []
  vars: []
```

<details>
<summary>Click to expand this bit to see the above, but with explanations for the configuration properties</summary>

```yaml
helmizer:
  commandSequence:  # list of commands/args executed serially. Inherits your $PATH
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
  dry-run: false  # optional - if true, does not write to a filesystem
  kustomization-directory: .  # optional - when referring to files with relative paths in a kustomization, if the kustomization.yaml is going to be in a special place relative to its files, set that path here
  kustomization-file-name: kustomization.yaml  # optional - name of kustomization file to write
  sort-keys: true  # optional - sort keys under crds, resources, patchesStrategicMerge
  version: '0.1.0'  # not yet validated
  ignore: []  # optional - list of files/directories to ignore
kustomize:  # this is essentially an overlay for your eventual kustomization.yaml
  commonAnnotations: {}
  commonLabels: {}
  configMapGenerator: []
  crds: []
  generatorOptions: {}
  images: []
  namePrefix: []
  namespace: ""
  nameSuffix: []
  openapi: {}
  patchesJson6902: []
  patchesStrategicMerge: []
  replacements: []
  replicas: []
  resources: []
  secretGenerator: []
  vars: []
```

</details>

## Installation

For local installation/use of the raw script, I use a local virtual environment to isolate dependencies:

```bash
git clone https://github.com/DaemonDude23/helmizer.git -b v0.12.0
cd helmizer
```

## Putting it in your `$PATH`

### Linux (Simplest Option)

1. Create symlink:
```bash
sudo ln -s /absolute/path/to/src/helmizer.py /usr/local/bin/helmizer
```
2. Install dependencies:
```bash
# latest and greatest dependency versions
pip3 install -U -r ./src/requirements.txt

# more flexible requirements
pip3 install -U -r ./src/requirements-old.txt
```

## virtualenv with pip

1. Update pip:
```bash
python3 -m pip install --upgrade pip
```
2. Install `virtualenv` for your user:
```bash
pip3 install -U virtualenv==20.16.2
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
pip3 install -U -r ./src/requirements.txt
pip3 install -U -r ./src/requirements-newest.txt
```

If you need to reset the virtual environment for whatever reason:
```bash
virtualenv --clear ./venv/
```

## Run

**For greater detail on running from examples (they assumes you've ran [helm template](https://helm.sh/docs/helm/helm_template/), see the [resource example](examples/resources/README.md))**

### Local Python

Input file:
```yaml
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
kustomize:
  namespace: sealed-secrets
  resources:
    - ./sealed-secrets/templates/
```

Helmize-ify it:
```bash
python3 ./src/helmizer.py ./examples/resources/helmizer.yaml
```

Output - enumerating the files within the specified directory:
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

### ~~Docker~~

**You may need a custom docker image depending on if you need certain apps when running commands within helmizer**
**I'm not maintaining the docker image anymore, but you could build it easily from the included Dockerfile**

In this example (*Nix OS), we're redirecting program output to the (e.g. `kustomization.yaml`) to the desired file because of issues with UID/GID on files bind-mounted from Docker. The redirect is not required however, you can correct permissions after the fact with `sudo chown -R username:groupname .`.

```bash
docker run --name helmizer \
  --rm \
  -v "$PWD"/examples:/tmp/helmizer -w /tmp/helmizer \
  docker.pkg.github.com/DaemonDude23/helmizer/helmizer:v0.12.0 /usr/src/app/helmizer.py \
    ./resources/ > ./examples/resources/kustomization.yaml
```

## Examples

_With [vscode](https://code.visualstudio.com/) you can utilize the included [launch.json](.vscode/launch.json) to test these more quickly, or reference for your configuration._
The `sealed-secrets` **Helm** chart is used for examples for its small scope. Here's another.

- [commonAnnotations](examples/commonAnnotations/)
- [commonLabels](examples/commonLabels/)
- [configMapGenerator](examples/configMapGenerator/)
- [crds](examples/crds/)
- [generatorOptions](examples/generatorOptions/)
- [images](examples/images/)
- [namePrefix](examples/namePrefix/)
- [namespace](examples/namespace/)
- [nameSuffix](examples/nameSuffix/)
- [openapi](examples/openapi/)
- [patchesJson6902](examples/patchesJson6902/)
- [patchStrategicMerge](examples/patchesStrategicMerge/)
- [replacements](examples/replacements/)
- [resources](examples/resources/)
- [secretGenerator](examples/secretGenerator/)
- [vars](examples/vars/)

---

Which looks easier to write/maintain through future chart updates for the [Prometheus Operator/kube-prometheus-stack](), this helmizer.yaml?


```yaml
helmizer:
  commandSequence:
  - command: "helm"
    args:
      - "-n"
      - "monitoring"
      - "template"
      - "kube-prometheus-stack"
      - --output-dir
      - '..'
      - --include-crds
      - --skip-tests
      - --validate
      - --version
      - '18.0.8'
      - --values
      - 'values.yaml'
      - prometheus-community/kube-prometheus-stack
  sort-keys: true
  version: '0.1.0'
kustomize:
  namespace: monitoring
  resources:
    - ./charts/
    - ./extra/prometheusRules/
    - ./extra/vpa.yaml
    - ./templates/
```

Or this `kustomization.yaml` which `helmizer` generated?

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: monitoring
resources:
- charts/kube-state-metrics/templates/clusterrolebinding.yaml
- charts/kube-state-metrics/templates/deployment.yaml
- charts/kube-state-metrics/templates/podsecuritypolicy.yaml
- charts/kube-state-metrics/templates/psp-clusterrole.yaml
- charts/kube-state-metrics/templates/psp-clusterrolebinding.yaml
- charts/kube-state-metrics/templates/role.yaml
- charts/kube-state-metrics/templates/service.yaml
- charts/kube-state-metrics/templates/serviceaccount.yaml
- charts/prometheus-node-exporter/templates/daemonset.yaml
- charts/prometheus-node-exporter/templates/psp-clusterrole.yaml
- charts/prometheus-node-exporter/templates/psp-clusterrolebinding.yaml
- charts/prometheus-node-exporter/templates/psp.yaml
- charts/prometheus-node-exporter/templates/service.yaml
- charts/prometheus-node-exporter/templates/serviceaccount.yaml
- templates/exporters/core-dns/service.yaml
- templates/exporters/core-dns/servicemonitor.yaml
- templates/exporters/kube-api-server/servicemonitor.yaml
- templates/exporters/kube-controller-manager/service.yaml
- templates/exporters/kube-controller-manager/servicemonitor.yaml
- templates/exporters/kube-etcd/service.yaml
- templates/exporters/kube-etcd/servicemonitor.yaml
- templates/exporters/kube-proxy/service.yaml
- templates/exporters/kube-proxy/servicemonitor.yaml
- templates/exporters/kube-scheduler/service.yaml
- templates/exporters/kube-scheduler/servicemonitor.yaml
- templates/exporters/kube-state-metrics/serviceMonitor.yaml
- templates/exporters/kubelet/servicemonitor.yaml
- templates/exporters/node-exporter/servicemonitor.yaml
- templates/prometheus-operator/clusterrole.yaml
- templates/prometheus-operator/clusterrolebinding.yaml
- templates/prometheus-operator/deployment.yaml
- templates/prometheus-operator/psp-clusterrole.yaml
- templates/prometheus-operator/psp-clusterrolebinding.yaml
- templates/prometheus-operator/psp.yaml
- templates/prometheus-operator/service.yaml
- templates/prometheus-operator/serviceaccount.yaml
- templates/prometheus-operator/servicemonitor.yaml
- templates/prometheus/additionalScrapeConfigs.yaml
- templates/prometheus/clusterrole.yaml
- templates/prometheus/clusterrolebinding.yaml
- templates/prometheus/ingress.yaml
- templates/prometheus/prometheus.yaml
- templates/prometheus/psp-clusterrole.yaml
- templates/prometheus/psp-clusterrolebinding.yaml
- templates/prometheus/psp.yaml
- templates/prometheus/service.yaml
- templates/prometheus/serviceaccount.yaml
- templates/prometheus/servicemonitor.yaml
```

# Kustomize Options

- [Kustomize Docs](https://kubectl.docs.kubernetes.io/references/kustomize/)

All `kustomize` configuration options are supported. See [here](https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/) for reference.
