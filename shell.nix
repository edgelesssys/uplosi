{ pkgs, ... }:

pkgs.mkShell {
  nativeBuildInputs = with pkgs; [
    go
    golangci-lint
    goreleaser
  ];
}
