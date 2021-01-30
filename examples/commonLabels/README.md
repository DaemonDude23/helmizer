# commonLabels

- [commonLabels](#commonlabels)
  - [Generating the Helm Template](#generating-the-helm-template)
  - [Generate Kustomization](#generate-kustomization)
    - [Raw Python](#raw-python)
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

### Raw Python

```bash
python3 ./src/helmizer.py \
  -n sealed-secrets \
  --resources ./examples/commonLabels/sealed-secrets/templates/ \
  --commonLabels "app.kubernetes.io/helmizer=true" "app.kubernetes.io/yes=false" \
  --kustomization-directory ./examples/commonLabels/
```

### Docker

```bash
docker run --name helmizer \
  --rm \
  -v "$PWD":/tmp/helmizer -w /tmp/helmizer \
  docker.pkg.github.com/chicken231/helmizer/helmizer:v0.4.0 /usr/src/app/helmizer.py \
    -n sealed-secrets \
    --resources ./examples/commonLabels/sealed-secrets/templates/ \
    --commonLabels "app.kubernetes.io/helmizer=true" "app.kubernetes.io/yes=false" \
    --kustomization-directory ./examples/commonLabels/ > ./examples/commonLabels/kustomization.yaml
```

## Validate

```bash
kubectl kustomize .
```
