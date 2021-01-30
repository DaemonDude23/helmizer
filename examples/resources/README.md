# resources

- [resources](#resources)
  - [Generating the Helm Template](#generating-the-helm-template)
  - [Generate Kustomization](#generate-kustomization)
    - [Local Python](#local-python)
    - [Docker](#docker)
  - [Validate](#validate)

---

## Generating the Helm Template

```bash
helm -n sealed-secrets template \
  sealed-secrets \
  --output-dir . \
  --include-crds \
  --skip-tests \
  --version 1.12.2 \
  stable/sealed-secrets
```

```bash
helm -n cert-manager template \
  cert-manager \
  --output-dir . \
  --skip-tests \
  --version 1.1.0 \
  jetstack/cert-manager
```

## Generate Kustomization

_These assumes you're in the root directory of this repository_

### Local Python

```bash
python3 ./src/helmizer.py \
  -n sealed-secrets \
  --resources sealed-secrets/templates/ \
  --kustomization-file-name kustomization.yml \
  --kustomization-directory ./examples/resources/
```

With a URL instead of a path to file:

```bash
python3 ./src/helmizer.py \
  -n cert-manager \
  --resources https://github.com/jetstack/cert-manager/releases/download/v1.1.0/cert-manager.yaml \
  --kustomization-file-name kustomization.yml \
  --kustomization-directory ./examples/resources/
```

### Docker

```bash
docker run --name helmizer \
  --rm \
  -v "$PWD":/tmp/helmizer -w /tmp/helmizer \
  docker.pkg.github.com/chicken231/helmizer/helmizer:v0.4.0 /usr/src/app/helmizer.py \
    -n sealed-secrets \
    --resources ./examples/resources/sealed-secrets/templates/ \
    --kustomization-directory ./examples/resources/ > ./examples/resources/kustomization.yaml
```

With a URL instead of a path to file:

```bash
docker run --name helmizer \
  --rm \
  -v "$PWD":/tmp/helmizer -w /tmp/helmizer \
  docker.pkg.github.com/chicken231/helmizer/helmizer:v0.4.0 /usr/src/app/helmizer.py \
    -n cert-manager \
    --resources \
      ./examples/resources/cert-manager/templates/ \
      https://github.com/jetstack/cert-manager/releases/download/v1.1.0/cert-manager.yaml \
    --kustomization-file-name kustomization.yml \
    --kustomization-directory ./examples/resources/ > ./examples/resources/kustomization.yml
```

## Validate

```bash
kubectl kustomize .
```
