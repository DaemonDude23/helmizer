**patchesStrategicMerge**

- [Generating the Helm Template](#generating-the-helm-template)
- [Generate Kustomization](#generate-kustomization)
  - [Raw Python](#raw-python)
- [Validate](#validate)

---

[https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/patchesstrategicmerge/](https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/patchesstrategicmerge/)

# Generating the Helm Template

```bash
helm -n cert-manager template \
  cert-manager \
  --output-dir . \
  --include-crds \
  --skip-tests \
  --version 1.14.3 \
  jetstack/cert-manager
```

# Generate Kustomization

_These assumes you're in the root directory of this repository_

## Raw Python

```bash
helmizer ./examples/patchStrategicMerge/helmizer.yaml
```

# Validate

```bash
kubectl kustomize ./examples/patchStrategicMerge/
```
