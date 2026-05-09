{
  description = "Fabulous dev shell for OCP Terraform Provider that does not taint system";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = {
    self,
    nixpkgs,
  } @ inputs: let
    supportedSystems = ["x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin"];
    forEachSupportedSystem = f:
      nixpkgs.lib.genAttrs supportedSystems (system:
        f {
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfreePredicate = pkg:
              builtins.elem (nixpkgs.lib.getName pkg) [
                "terraform"
              ];
          };
        });
  in {
    devShells = forEachSupportedSystem ({pkgs}: {
      default = pkgs.mkShell {
        name = "terraform-provider-ocp-devshell";
        packages = with pkgs; [
          bashInteractive
          pre-commit
          git

          alejandra # TODO: check what is used nowadays
          nixd # TODO: check what is used nowadays

          gcc
          go_1_26
          gopls # Go language server
          golangci-lint # Go linter
          delve # go debug tool

          terraform
        ];
        hardeningDisable = ["fortify"]; # otherwise `dlv dap` debug does not work
        env = {
          GO111MODULE = "on";
          CGO_ENABLED = "1";
        };
        shellHook = ''
          echo "Launching OCP Terraform Provider"

          # .env sourcing if exists
          if [ -f .env ]; then
            set -a
            source .env
            set +a
          fi

          # Go
          export GOPATH="$PWD/.devshell/go"
          export GOBIN="$GOPATH/bin"
          export GOMODCACHE="$GOPATH/pkg/mod"
          export GOCACHE="$PWD/.devshell/gocache"
          mkdir -p "$GOBIN" "$GOMODCACHE" "$GOCACHE"
          export PATH="$PATH:$GOBIN"

          # Terraform
          export TF_CLI_CONFIG_FILE="$PWD/.terraformrc" # terraform config
          export TF_DATA_DIR="$PWD/.devshell/terraform" # directory for Terraform data
          export TF_PLUGIN_CACHE_DIR="$TF_DATA_DIR/plugin-cache" # directory for caching provider plugins
          mkdir -p "$TF_PLUGIN_CACHE_DIR" "$TF_DATA_DIR"

          cat <<'EOF' > "$TF_CLI_CONFIG_FILE"
          provider_installation {
            dev_overrides {
                "hashicorp.com/ocp/ocp" = "$GOBIN"
            }
            direct {}
          }
          EOF
        '';
      };
    });
  };
}
