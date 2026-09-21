{
  description = "skills-reconcile CLI";

  inputs = {
    # 26.05 is the final nixpkgs release that supports the required x86_64-darwin system.
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-26.05-darwin";
  };

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forEachSystem =
        f: nixpkgs.lib.genAttrs systems (system: f system (import nixpkgs { inherit system; }));
    in
    {
      packages = forEachSystem (
        _system: pkgs:
        let
          skillsReconcile = pkgs.buildGoModule {
            pname = "skills-reconcile";
            version = "0.0.0";

            src = pkgs.lib.cleanSourceWith {
              src = ./.;
              filter =
                path: _type:
                let
                  relativePath = pkgs.lib.removePrefix "${toString ./.}/" (toString path);
                in
                relativePath == "go.mod"
                || relativePath == "cmd"
                || pkgs.lib.hasPrefix "cmd/" relativePath
                || relativePath == "internal"
                || pkgs.lib.hasPrefix "internal/" relativePath;
            };
            subPackages = [ "cmd/skills-reconcile" ];

            vendorHash = null;

            checkPhase = ''
              runHook preCheck
              go test ./...
              runHook postCheck
            '';

            doInstallCheck = true;
            installCheckPhase = ''
              runHook preInstallCheck

              $out/bin/skills-reconcile --help >/dev/null

              runHook postInstallCheck
            '';

            ldflags = [
              "-s"
              "-w"
            ];

            meta = {
              description = "List Skills in .agents/skills";
              homepage = "https://github.com/y-writings/skills-reconcile";
              license = pkgs.lib.licenses.mit;
              mainProgram = "skills-reconcile";
            };
          };
        in
        {
          skills-reconcile = skillsReconcile;
          default = skillsReconcile;
        }
      );

      apps = forEachSystem (
        system: _pkgs:
        let
          skillsReconcile = {
            type = "app";
            program = "${self.packages.${system}.skills-reconcile}/bin/skills-reconcile";
          };
        in
        {
          skills-reconcile = skillsReconcile;
          default = skillsReconcile;
        }
      );
    };
}
