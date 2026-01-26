{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  packages = with pkgs; [
    go
    openbao
  ];

  shellHook = ''
    echo "🚀 go:        $(go version)"
  '';
}