# Developers

- [Developers](#developers)
  - [~~PyInstaller~~](#pyinstaller)
    - [~~Linux~~](#linux)
  - [~~Flatpak~~](#flatpak)
  - [Prep release](#prep-release)
    - [Update version nunmber](#update-version-nunmber)

## ~~PyInstaller~~

- https://pyinstaller.readthedocs.io/en/stable/index.html

### ~~Linux~~

1. Install package via `pip`:
```bash
pip install pyinstaller
```
2. Create package
```bash
pyinstaller --distpath dist/linux/ src/helmizer.py
```
3. `zip` contents:
```bash
zip -9 -T -r ./dist/linux/releases/v0.5.0.zip dist/linux/helmizer
```

## ~~Flatpak~~

Build
```bash
flatpak-builder \
  --force-clean\
  ./build/flatpak \
  org.chicken231.Helmizer.yaml
```

Local installation
```bash
flatpak-builder \
  --force-clean \
  --user \
  --install \
  ./build/flatpak \
  org.chicken231.Helmizer.yaml && \
    flatpak run --user org.chicken231.Helmizer
```

Run
```bash
flatpak run --user org.chicken231.Helmizer
flatpak run org.chicken231.Helmizer
```

## Prep release

### Update version nunmber

```bash
find . -type f -exec sed -i 's!v0.0.0!v0.0.1!g' '{}' \;
```
