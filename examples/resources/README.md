# Simple Resources

- [Simple Resources](#simple-resources)
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

## Generate Kustomization

_These assumes you're in the root directory of this repository_

### Local Python

```bash
python3 ./src/helmizer.py \
  -n sealed-secrets \
  --resource-paths sealed-secrets/templates/ \
  --kustomization-directory .
```

### Docker

```bash
docker run --name helmizer \
  --rm \
  -v "$PWD"/examples:/tmp/helmizer \
  -w /tmp/helmizer \
  docker.pkg.github.com/chicken231/helmizer/helmizer:v0.2.0 /usr/src/app/helmizer.py \
    -n sealed-secrets \
    --resource-paths ./examples/resources/sealed-secrets/templates/ \
    --kustomization-directory ./examples/resources/ > ./examples/resources/kustomization.yaml
```

## Validate

```bash
kubectl kustomize .
```
