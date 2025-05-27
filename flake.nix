{
  description = "Gomuks development environment & packages";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
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
        pkgs = import nixpkgs {
          inherit system;
          config = {
            permittedInsecurePackages = [ "olm-3.2.16" ];
          };
        };

        outPackages = self.outputs.packages.${system};

        # Extract version from version.go
        versionContent = builtins.readFile ./version/version.go;
        versionMatch = builtins.match ''^.*const StaticVersion = "([0-9\.]+)".*$'' versionContent;
        version = builtins.elemAt versionMatch 0;
      in
      {
        packages = {
          gomuks = pkgs.buildGoModule {
            pname = "gomuks";
            inherit version;

            src = ./.;

            # Go dependency hash (should be updated when dependencies are)
            vendorHash = "sha256-oViJGEdep4vk38EwygkUXy51NRdvndJqlXZOxWYCbv4=";

            buildInputs = with pkgs; [
              outPackages.gomuks-web
              olm
            ];

            preBuild = ''
              cp -r ${outPackages.gomuks-web}/dist web/dist
            '';

            # skip non-existant & broken pytest tests
            pytestCheckPhase = '':'';

            subPackages = [ "cmd/gomuks" ];
          };
          # Package for building web dist
          gomuks-web = pkgs.buildNpmPackage rec {
            pname = "gomuks-web";
            inherit version;

            src = ./web;

            # Same as the Go dependency hash but for NPM packages
            npmDepsHash = "sha256-tPRgzp6c9zXbHQIEPf4gEKkBWWzQAfUCNAijYPdONpM=";

            installPhase = ''
              mkdir -p $out/dist
              cp -r dist/* $out/dist/
            '';
          };
          default = outPackages.gomuks;
        };

        devShells = {
          default = pkgs.mkShell {
            packages = with pkgs; [
              glib-networking
              go
              go-task
              go-tools
              gotools
              gst_all_1.gstreamer
              gst_all_1.gst-plugins-base
              gst_all_1.gst-plugins-good
              gst_all_1.gst-plugins-bad
              gst_all_1.gst-plugins-ugly
              gst_all_1.gst-libav
              gst_all_1.gst-vaapi
              libsoup_2_4
              olm
              pkg-config
              pre-commit
              webkitgtk_4_1
            ];
          };
        };
      }
    );
}
