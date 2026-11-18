{
  self,
  config,
  ...
}: {
  flake.modules.nixos.mint = {
    config,
    lib,
    pkgs,
    ...
  }: let
    cfg = config.services.mint;
    inherit (lib) mkOption mkEnableOption mkIf types;
  in {
    options.services.mint = {
      enable = mkEnableOption "headscale CI preauth key minter";
      package = mkOption {
        type = types.package;
        default = self.packages.${pkgs.stdenv.hostPlatform.system}.mint;
      };
      audience = mkOption {type = types.str;};
      userId = mkOption {
        type = types.ints.unsigned;
        default = 0;
        description = "headscale user to create keys under, 0 for tags-only keys (needs a recent headscale)";
      };
      allow = mkOption {
        type = types.listOf types.str;
        description = "OIDC sub patterns, trailing * is a prefix match";
      };
      tags = mkOption {
        type = types.listOf types.str;
        default = ["tag:ci"];
      };
      ttl = mkOption {
        type = types.str;
        default = "5m";
      };
      address = mkOption {
        type = types.str;
        default = "127.0.0.1";
      };
      port = mkOption {
        type = types.port;
        default = 8081;
      };
      socket = mkOption {
        type = types.path;
        default = "/run/headscale/headscale.sock";
      };
    };

    config = mkIf cfg.enable {
      systemd.services.mint = {
        wantedBy = ["multi-user.target"];
        wants = ["network-online.target"];
        after = ["network-online.target" "headscale.service"];
        environment = {
          AUDIENCE = cfg.audience;
          HS_USER_ID = toString cfg.userId;
          ALLOW = lib.concatStringsSep "," cfg.allow;
          TAGS = lib.concatStringsSep "," cfg.tags;
          TTL = cfg.ttl;
          BIND = cfg.address;
          PORT = toString cfg.port;
          HS_SOCK = cfg.socket;
        };
        serviceConfig = {
          ExecStart = lib.getExe cfg.package;
          Restart = "on-failure";
          RestartSec = 5;
          DynamicUser = true;
          SupplementaryGroups = ["headscale"];
          NoNewPrivileges = true;
          ProtectSystem = "strict";
          ProtectHome = true;
          PrivateTmp = true;
          PrivateDevices = true;
          ProtectKernelTunables = true;
          ProtectKernelModules = true;
          ProtectControlGroups = true;
          RestrictAddressFamilies = ["AF_UNIX" "AF_INET" "AF_INET6"];
          RestrictNamespaces = true;
          LockPersonality = true;
          MemoryDenyWriteExecute = true;
          SystemCallArchitectures = "native";
          CapabilityBoundingSet = "";
        };
      };
    };
  };

  flake.nixosModules.mint = config.flake.modules.nixos.mint;
}
