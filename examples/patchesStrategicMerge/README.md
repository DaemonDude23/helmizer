# patchesStrategicMerge

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
  --resource-paths ./examples/patchStrategicMerge/sealed-secrets/templates/ \
  --patchesStrategicMerge ./examples/patchStrategicMerge/patches/ \
  --kustomization-directory ./examples/patchStrategicMerge/
```

## Validate

```bash
kubectl kustomize .
```
