# $ nix-shell shell.nix

{ pkgs ? import (fetchTarball "https://github.com/NixOS/nixpkgs/archive/nixpkgs-unstable.tar.gz") {} }:

pkgs.mkShell {
  buildInputs = [
    pkgs.go_1_25  # this was the latest available version at the time of writing
    # for diagrams
    pkgs.graphviz
    pkgs.python313
  ];

  shellHook = ''
    export GOPATH=$(pwd)/.go
    export PATH=$GOPATH/bin:$PATH
  '';
}
