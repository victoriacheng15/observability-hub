{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  packages = with pkgs; [
    kube-linter
    trivy
    hclfmt
    action-validator
  ];
}
