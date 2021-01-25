# Simple Resources

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
  --patches-strategic-merge-paths ./examples/patchStrategicMerge/patches/ \
  --kustomization-directory ./examples/patchStrategicMerge/
```

### Docker

```bash
docker run --name helmizer \
  --rm \
  -v "$PWD":/tmp/helmizer -w /tmp/helmizer \
  docker.pkg.github.com/chicken231/helmizer/helmizer:v0.2.0 /usr/src/app/helmizer.py \
    -n sealed-secrets \
    --resource-paths ./examples/patchStrategicMerge/sealed-secrets/templates/ \
    --patches-strategic-merge-paths ./examples/patchStrategicMerge/extra/ \
    --kustomization-directory ./examples/patchStrategicMerge/ > ./examples/patchStrategicMerge/kustomization.yaml
```

## Validate

```bash
kubectl kustomize .
```
