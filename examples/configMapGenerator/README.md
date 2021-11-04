# resources

- [resources](#resources)
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
python3 ./src/helmizer.py ./examples/resources/helmizer.yaml
```

## Validate

```bash
kubectl kustomize .
```
