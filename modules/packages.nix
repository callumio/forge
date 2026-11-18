{lib, ...}: {
  perSystem = {pkgs, ...}: {
    packages = lib.packagesFromDirectoryRecursive {
      inherit (pkgs) callPackage;
      directory = ../pkgs;
    };
  };

  flake.overlays.default = final: _:
    lib.packagesFromDirectoryRecursive {
      inherit (final) callPackage;
      directory = ../pkgs;
    };
}
