# patchesJson6902

- [patchesJson6902](#patchesjson6902)
  - [Generating the Helm Template](#generating-the-helm-template)
  - [Generate Kustomization](#generate-kustomization)
    - [Local Python](#local-python)
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
python3 ./src/helmizer.py ./examples/patchesJson6902/helmizer.yaml
```

## Validate

```bash
kubectl kustomize .
```
