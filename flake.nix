{
  description = "Upload OS images to cloud provider";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          uplosi = pkgs.buildGoModule {
            pname = "uplosi";
            version = "0.1.2";
            src = ./.;
            # this needs to be updated together with go.mod / go.sum
            vendorHash = "sha256-RsjUPLe8omoN+XGyNhHDxzNfZR7VVTkh/f/On1oCRqM=";

            CGO_ENABLED = 0;

            ldflags = [ "-s" "-w" "-buildid=" ];

            nativeBuildInputs = [ pkgs.installShellFiles ];

            postInstall = ''
              installShellCompletion --cmd uplosi \
                --bash <($out/bin/uplosi completion bash) \
                --fish <($out/bin/uplosi completion fish) \
                --zsh <($out/bin/uplosi completion zsh)
            '';

            meta = with pkgs.lib; {
              description = "Upload OS images to cloud provider";
              homepage = "https://github.com/edgelesssys/uplosi";
              maintainers = with maintainers; [ katexochen malt3 ];
              license = licenses.asl20;
            };
          };
        in
        {
          devShells.default = import ./shell.nix { inherit pkgs; };

          packages = {
            default = uplosi;
            uplosi = uplosi;
          };

          formatter = nixpkgs.legacyPackages.${system}.nixpkgs-fmt;
        }
      );
}
