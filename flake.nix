{
  description = "helmizer - runs commands then generates kustomization.yaml files for Kubernetes";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        helmizer = pkgs.buildGoModule {
          pname = "helmizer";
          version = "0.19.0";
          src = ./src;
          vendorHash = "sha256-8s2Yu22vj+zphtWWebBdSGNpPHzT/Qayu6Sje8yIve8=";
        };
      in {
        packages.default = helmizer;

        apps.default = flake-utils.lib.mkApp {
          drv = helmizer;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [ pkgs.go_1_26 pkgs.gopls pkgs.gotools ];
        };
      });
}
