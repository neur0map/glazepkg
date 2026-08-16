{ lib
, buildGoModule
, installShellFiles
, version ? "dev"
}:

buildGoModule {
  pname = "gpk";
  inherit version;

  src = lib.cleanSource ./..;

  vendorHash = "sha256-QFR5A1EvjMQZrNbiHRXo6DjRL5nyCUWRqk2AZcacfxo=";

  subPackages = [ "cmd/gpk" ];

  ldflags = [ "-s" "-w" "-X" "main.version=${version}" ];

  # The store is read-only and Nix owns the file, so `gpk update` must refuse
  # rather than try to replace it.
  tags = [ "noselfupdate" ];

  nativeBuildInputs = [ installShellFiles ];

  postInstall = ''
    installShellCompletion --cmd gpk \
      --bash <($out/bin/gpk completion bash) \
      --zsh <($out/bin/gpk completion zsh) \
      --fish <($out/bin/gpk completion fish)
  '';

  # The suite shells out to real package managers and reads the user's data dir;
  # neither exists in the sandbox.
  doCheck = false;

  meta = {
    description = "One package viewer for every package manager";
    homepage = "https://github.com/neur0map/glazepkg";
    license = lib.licenses.gpl3Only;
    mainProgram = "gpk";
    platforms = lib.platforms.unix ++ lib.platforms.windows;
  };
}
