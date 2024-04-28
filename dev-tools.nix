# This will create a shell environment with Go 1.21, Python 3.11, and Graphviz available.
# The  shellHook  is a shell script that is run when the shell is started. In this case, it sets the  GOPATH  environment variable to the current directory and adds the  bin  directory to the  PATH .
# To use this shell environment, save the file as  shell.nix  and run  nix-shell .
# $ nix-shell ./dev-tools.nix

{ pkgs ? import (fetchTarball "https://github.com/NixOS/nixpkgs/archive/nixpkgs-unstable.tar.gz") {} }:

pkgs.mkShell {
  buildInputs = [
    pkgs.go_1_21  # this was the latest available version at the time of writing
    # for diagrams
    pkgs.graphviz
    pkgs.python311
  ];

  shellHook = ''
    export GOPATH=$(pwd)/.go
    export PATH=$GOPATH/bin:$PATH
  '';
}
