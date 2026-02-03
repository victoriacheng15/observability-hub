{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  packages = with pkgs; [
    go
    openbao
    kubernetes-helm
    k9s
  ];
}