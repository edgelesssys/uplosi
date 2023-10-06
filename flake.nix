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
          version = "0.1.0";
          uplosi = pkgs.buildGo121Module
            {
              pname = "uplosi";
              version = version;
              src = ./.;
              # this needs to be updated together with go.mod / go.sum
              vendorHash = "sha256-3WLDmw2rhmjrKJ8QXtARS9p8qFx17iwUnljwoUep2uc=";

              preBuild = ''
                export CGO_ENABLED=0
              '';

              ldflags = [ "-s" "-w" "-buildid=" ];
              flags = [ "-trimpath" ];

              meta = with nixpkgs.lib; {
                description = "Upload OS images to cloud provider";
                homepage = "https://github.com/edgelesssys/uplosi";
                maintainers = with maintainers; [
                  katexochen
                  malt3
                ];
                license = licenses.asl20;
              };
            };
        in
        {
          devShells.default = import ./shell.nix { inherit pkgs; };

          packages.uplosi = uplosi;
          defaultPackage = uplosi;

          formatter = nixpkgs.legacyPackages.${system}.nixpkgs-fmt;
        }
      );
}
