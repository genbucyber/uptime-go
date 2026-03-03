{
  description = "Uptime monitoring service written in Go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05-small";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            air
            cobra-cli
            go
            gopls
            goreleaser
          ];

          shellHooks = ''
            export GIN_MODE=debug
          '';
        };

        packages.default = pkgs.buildGoModule {
          pname = "uptime-go";
          version = "0.4.3";
          src = self;
          vendorHash = "sha256-p11U7GGDRO/FiKIZQ98i7lJ98MJ93Vo84x9+FHfqueA=";

          meta = {
            description = "Uptime monitoring service written in Go";
            homepage = "https://github.com/genbucyber/uptime-go";
            maintainers = with pkgs.lib.maintainers; [ Aspiand ];
            platforms = pkgs.lib.platforms.all;
          };
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/uptime-go";
        };
      }
    );
}
