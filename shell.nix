{ pkgs, ... }:
pkgs.mkShell {
  nativeBuildInputs = with pkgs; [
    go_1_21
    golangci-lint
    goreleaser
  ];
}
