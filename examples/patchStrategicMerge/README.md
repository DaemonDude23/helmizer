# Simple Resources

## Generating the Helm Template

```bash
helm -n sealed-secrets template \
  sealed-secrets \
  --output-dir . \
  --include-crds \
  --version 1.12.2 \
  stable/sealed-secrets
```

## Generate Kustomization

```bash
python3 ../../helmizer.py \
  -n sealed-secrets \
  --resource-paths \
  sealed-secrets/templates/ \
  --kustomization-directory . \
  --patches-strategic-merge-paths patches/
```

## Validate

```bash
kubectl kustomize .
```
