{
  description = "Upload OS images to cloud provider";

  inputs = {
    nixpkgs = {
      url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    };
    flake-utils = {
      url = "github:numtide/flake-utils";
    };
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      treefmt-nix,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        uplosi = pkgs.buildGoModule {
          pname = "uplosi";
          version = "devel";

          src =
            let
              inherit (pkgs.lib) fileset path hasSuffix;
              root = ./.;
            in
            fileset.toSource {
              inherit root;
              fileset = fileset.unions [
                (fileset.fileFilter (file: hasSuffix ".go" file.name) root)
                (path.append root "go.mod")
                (path.append root "go.sum")
                (path.append root "config/validation.rego")
                (path.append root "measured-boot/internal/testdata/uki.efi")
              ];
            };

          # this needs to be updated together with go.mod / go.sum
          vendorHash = "sha256-2lJmPNLpI1ksFb0EtcjPjyTy7eX1DKeX0F80k9FtGno=";

          CGO_ENABLED = 0;

          ldflags = [ "-s" ];

          nativeBuildInputs = [ pkgs.installShellFiles ];

          postInstall = ''
            installShellCompletion --cmd uplosi \
              --bash <($out/bin/uplosi completion bash) \
              --fish <($out/bin/uplosi completion fish) \
              --zsh <($out/bin/uplosi completion zsh)
          '';
        };

        treefmtEval = treefmt-nix.lib.evalModule pkgs ./treefmt.nix;
      in
      {
        devShells.default = import ./shell.nix { inherit pkgs; };

        packages = {
          default = uplosi;
          inherit uplosi;
        };

        legacyPackages = {
          nixpkgs = nixpkgs.legacyPackages.${system};
        };

        formatter = treefmtEval.config.build.wrapper;

        checks = {
          formatting = treefmtEval.config.build.check self;
        };
      }
    );
}
