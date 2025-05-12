{
  description = "Gomuks development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    (flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.permittedInsecurePackages = [ "olm-3.2.16" ];
        };
      in {
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
      }));
}
