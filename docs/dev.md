# Developer Docs

## Go init

- Install Go dependencies

```bash
cd ./src/
```

```bash
go mod init daemondude23/helmizer/m
```

```bash
# optionally download deps and update go.sum
cd ./src/
go get -u ./...
go mod tidy
```

## Build Executable

```bash
cd ./src/
go build -o ../build/test/helmizer
```

If built on **NixOS**, you can override the interpreter so it'll work on a conventional Linux Distribution:

```bash
patchelf --set-interpreter /lib64/ld-linux-x86-64.so.2 ../build/test/helmizer
```

## Delete All Kustomization Files

- Use when test if the file is being created

```bash
find . -type f -name kustomization.yaml -exec rm -f '{}' \;
```

```bash
./build/test/helmizer --config-glob "**/helmizer.yaml"
```

## Trigger goreleaser

```bash
git tag v0.18.0
git push origin v0.18.0
```

## Nix

- Installs some dependencies you can use to run **Helmizer**.

```bash
nix-shell
```
