{
  description = "Euxenion Go development environment";

  # Pin nixpkgs so nix develop uses a predictable Go toolchain.
  inputs.nixpkgs.url =
    "github:NixOS/nixpkgs/319e4cebef10cbb636c6ea392137b52ad105336a";

  outputs = { nixpkgs, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      devShells = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            packages = [
              pkgs.go_1_27
              pkgs.just
              pkgs.git
            ];
          };
        });
    };
}
