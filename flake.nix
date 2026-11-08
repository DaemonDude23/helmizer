{
  description = "Generates kustomization.yaml for your locally-rendered YAML manifests, such as with Helm templates";

  # The one toolchain pin. Every tool used by pre-commit, CI, and the Makefile
  # comes from this nixpkgs revision; Renovate's lockFileMaintenance bumps it.
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      inherit (nixpkgs) lib;
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      forAllSystems = f: lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
      version = lib.removeSuffix "\n" (builtins.readFile ./src/VERSION);

      # Formatters behind `nix fmt` and the treefmt pre-commit hook (treefmt.toml).
      formatters =
        pkgs: with pkgs; [
          nixfmt
          shfmt
          opentofu
          hclfmt
          gotools
          (goFor pkgs)
        ];
      treefmt =
        pkgs:
        pkgs.writeShellApplication {
          name = "treefmt";
          runtimeInputs = [ pkgs.treefmt ] ++ formatters pkgs;
          text = ''exec treefmt "$@"'';
        };
      # MkDocs with the Material theme, from nixpkgs only (no pip).
      mkDocsPython =
        pkgs:
        pkgs.python3.withPackages (ps: [
          ps.mkdocs-material
          ps.pymdown-extensions
        ]);

      # Go toolchain for the build and the devShell. go.mod pins only the minor
      # version (go 1.27); bump both together for a new minor.
      goFor = pkgs: pkgs.go_1_27;
      mkPackage =
        pkgs:
        (pkgs.buildGoModule.override { go = goFor pkgs; }) {
          pname = "helmizer";
          inherit version;
          # Go sources only (plus testdata, if any), so docs edits don't rebuild.
          src = lib.fileset.toSource {
            root = ./.;
            # The tests read ../examples and ../.vscode/launch.json; VERSION is embedded.
            fileset = lib.fileset.unions [
              ./src
              ./examples
              ./.vscode/launch.json
            ];
          };
          modRoot = "src";
          subPackages = [ "." ];
          # Refresh with scripts/update-vendor-hash.sh after go.mod/go.sum changes.
          vendorHash = "sha256-Q7kEPw7VwUJ9DgTSfxbN0yz+rEJyRs/i0BUQsvFsg7c=";
          env.CGO_ENABLED = "0";
          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];
          # Unit tests of the whole module (buildGoModule's default only tests
          # subPackages), run by `nix flake check`.
          checkPhase = ''
            runHook preCheck
            go test -count=1 ./...
            runHook postCheck
          '';
          meta = {
            description = "Generates kustomization.yaml for your locally-rendered YAML manifests, such as with Helm templates";
            homepage = "https://github.com/DaemonDude23/helmizer";
            license = lib.licenses.asl20;
            mainProgram = "helmizer";
          };
        };
    in
    {
      packages = forAllSystems (pkgs: {
        # The static documentation site (`mkdocs build --strict`). Only the files the
        # site is built from are in its source, so unrelated edits do not rebuild it.
        docs = pkgs.stdenvNoCC.mkDerivation {
          pname = "helmizer-docs";
          inherit version;
          src = lib.fileset.toSource {
            root = ./.;
            fileset = lib.fileset.unions [
              ./mkdocs.yml
              ./docs
              ./CHANGELOG.md
            ];
          };
          nativeBuildInputs = [ (mkDocsPython pkgs) ];
          dontConfigure = true;
          # Silences Material's MkDocs 2.0 banner.
          env.NO_MKDOCS_2_WARNING = "true";
          buildPhase = ''
            runHook preBuild
            mkdocs build --strict --site-dir "$out"
            runHook postBuild
          '';
          dontInstall = true;
        };

        default = mkPackage pkgs;
        helmizer = mkPackage pkgs;
      });

      apps = forAllSystems (pkgs: {
        # `nix run .#docs` (from the repository root) serves the docs with live reload
        # at http://127.0.0.1:8000. Extra arguments go to `mkdocs serve`.
        docs = {
          type = "app";
          program = lib.getExe (
            pkgs.writeShellApplication {
              name = "helmizer-docs";
              runtimeInputs = [ (mkDocsPython pkgs) ];
              text = ''
                export NO_MKDOCS_2_WARNING=true
                if [ ! -f mkdocs.yml ]; then
                  echo "run this from the repository root (no mkdocs.yml in $PWD)" >&2
                  exit 1
                fi
                exec mkdocs serve "$@"
              '';
            }
          );
          meta.description = "Serve the documentation with live reload";
        };

        default = {
          type = "app";
          program = lib.getExe (mkPackage pkgs);
          meta.description = "Generates kustomization.yaml for your locally-rendered YAML manifests, such as with Helm templates";
        };
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages =
            with pkgs;
            [
              (treefmt pkgs)
              pre-commit
              shellcheck
              gitleaks
              typos
              markdownlint-cli2
              editorconfig-checker
              opentofu
              terragrunt
              tflint
              jq
              gnumake
              git
              actionlint
              zizmor
              gh

              (mkDocsPython pkgs)
              lychee

              (goFor pkgs)
              gopls
              gotools
              delve
              golangci-lint
              govulncheck
              gosec
              goreleaser
              syft
              cosign

              # helmizer: the release tooling tests (scripts/test_*.py) and
              # running the examples by hand.
              python3
              kubernetes-helm
              kustomize
            ]
            ++ formatters pkgs;
          NO_MKDOCS_2_WARNING = "true";
          shellHook = ''
            # Install the git hooks once per clone (cheap when already installed).
            if git rev-parse --git-dir >/dev/null 2>&1 && [ ! -f "$(git rev-parse --git-path hooks)/pre-commit" ]; then
              pre-commit install --install-hooks >/dev/null || echo "pre-commit install failed; run it by hand" >&2
            fi
          '';
        };
      });

      formatter = forAllSystems treefmt;

      checks = forAllSystems (
        pkgs:
        {
          # Fails when any file is not formatted (`nix fmt` fixes it).
          formatting =
            pkgs.runCommand "formatting"
              {
                nativeBuildInputs = [ (treefmt pkgs) ];
              }
              ''
                cp -r --no-preserve=mode ${self} src
                cd src
                treefmt --ci --no-cache --tree-root .
                touch $out
              '';
        }
        // lib.mapAttrs' (
          n: lib.nameValuePair "package-${n}"
        ) self.packages.${pkgs.stdenv.hostPlatform.system}
      );
    };
}
