{ pkgs ? import (fetchTarball "https://github.com/NixOS/nixpkgs/archive/nixos-unstable.tar.gz") {} }:

pkgs.mkShell {
  packages = with pkgs; [
    go_1_25
    openbao
    kubernetes-helm
    k9s
    kube-linter
    hclfmt
    action-validator
  ];
}
    