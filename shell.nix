{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  packages = with pkgs; [
    kube-linter
    hclfmt
    action-validator
  ];
}
    