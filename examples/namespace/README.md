**namespace**

- [Generating the Helm Template](#generating-the-helm-template)
- [Generate Kustomization](#generate-kustomization)
  - [Local Python](#local-python)
- [Validate](#validate)

---

[https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/namespace/](https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/namespace/)

# Generating the Helm Template

```bash
helm -n sealed-secrets template \
  sealed-secrets \
  --output-dir . \
  --include-crds \
  --skip-tests \
  --version 1.12.2 \
  stable/sealed-secrets
```

# Generate Kustomization

_These assumes you're in the root directory of this repository_

## Local Python

```bash
python3 ./src/helmizer.py ./examples/namespace/helmizer.yaml
```

# Validate

```bash
kubectl kustomize ./examples/namespace/
```
