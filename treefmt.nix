{ lib, pkgs, ... }:

{
  projectRootFile = "flake.nix";
  programs = {
    # keep-sorted start block=true
    actionlint.enable = true;
    deadnix.enable = true;
    formatjson5 = {
      enable = true;
      indent = 2;
      oneElementLines = true;
      sortArrays = true;
    };
    keep-sorted.enable = true;
    nixfmt.enable = true;
    shellcheck.enable = true;
    shfmt.enable = true;
    statix.enable = true;
    # keep-sorted end
  };
  settings.formatter = {
    yamlfmt = {
      command = "${lib.getExe pkgs.yamlfmt}";
      options = [ "-formatter=retain_line_breaks_single=true" ];
      includes = [
        "*.yaml"
        "*.yml"
      ];
    };
  };
}
