{
  lib,
  buildGoModule,
}:
buildGoModule {
  pname = "hetm";
  version = "0.1.0";

  src = lib.fileset.toSource {
    root = ./.;
    fileset = lib.fileset.unions [./go.mod ./go.sum ./main.go];
  };

  vendorHash = "sha256-uvj6uaZG3cFfDxaEyjeO7u5LXjNjLnJaU3jnLWzegnc=";

  env.CGO_ENABLED = 0;
  ldflags = ["-s" "-w"];

  meta = {
    description = "Mints ephemeral headscale preauth keys for GitHub Actions via OIDC";
    mainProgram = "hetm";
  };
}
