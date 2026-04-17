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
        version = "0.19.2";

        helmizer = pkgs.buildGoModule {
          pname = "helmizer";
          inherit version;
          src = ./.;
          modRoot = "src";
          subPackages = [ "." ];
          vendorHash = "sha256-8s2Yu22vj+zphtWWebBdSGNpPHzT/Qayu6Sje8yIve8=";
          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];

          meta = with pkgs.lib; {
            description = "Generate kustomization.yaml files with optional pre/post command execution";
            homepage = "https://github.com/DaemonDude23/helmizer";
            license = licenses.asl20;
            mainProgram = "helmizer";
          };
        };
      in {
        packages.default = helmizer;

        apps.default = flake-utils.lib.mkApp {
          drv = helmizer;
        };

        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go_1_26 pkgs.gopls pkgs.gotools ];
        };
      });
}
