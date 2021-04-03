# commonLabels

- [commonLabels](#commonlabels)
  - [Generating the Helm Template](#generating-the-helm-template)
  - [Generate Kustomization](#generate-kustomization)
    - [Raw Python](#raw-python)
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

## Validate

```bash
kubectl kustomize .
```
