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
          nodejs = pkgs.nodejs_22;

          skillsCli =
            assert pkgs.lib.assertMsg (pkgs.lib.versionAtLeast nodejs.version "22.20.0")
              "skills 1.5.23 requires Node.js 22.20.0 or newer";
            pkgs.buildNpmPackage {
              pname = "skills-cli";
              version = "1.5.23";

              src = ./tools/skills;
              inherit nodejs;

              npmDeps = pkgs.importNpmLock { npmRoot = ./tools/skills; };
              npmConfigHook = pkgs.importNpmLock.npmConfigHook;
              npmFlags = [ "--ignore-scripts" ];
              dontNpmBuild = true;

              nativeBuildInputs = [ pkgs.makeWrapper ];

              installPhase = ''
                runHook preInstall

                mkdir -p $out/bin $out/lib/skills-cli
                cp -R node_modules $out/lib/skills-cli/
                makeWrapper ${nodejs}/bin/node $out/bin/skills \
                  --add-flags $out/lib/skills-cli/node_modules/skills/bin/cli.mjs

                runHook postInstall
              '';

              meta = {
                description = "Pinned skills CLI used by skills-reconcile";
                homepage = "https://github.com/vercel-labs/skills";
                license = pkgs.lib.licenses.mit;
                mainProgram = "skills";
              };
            };

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
                relativePath == "go.mod" || relativePath == "cmd" || pkgs.lib.hasPrefix "cmd/" relativePath;
            };
            subPackages = [ "cmd/skills-reconcile" ];

            vendorHash = null;

            nativeBuildInputs = [ pkgs.makeWrapper ];

            postFixup = ''
              wrapProgram $out/bin/skills-reconcile \
                --set SKILLS_RECONCILE_EXECUTABLE ${skillsCli}/bin/skills
            '';

            doInstallCheck = true;
            installCheckPhase = ''
              runHook preInstallCheck

              $out/bin/skills-reconcile --help >/dev/null
              skills_version="$(${skillsCli}/bin/skills --version)"
              case "$skills_version" in
                1.5.23|"skills 1.5.23"|v1.5.23) ;;
                *)
                  echo "unexpected skills version: $skills_version" >&2
                  exit 1
                  ;;
              esac

              runHook postInstallCheck
            '';

            ldflags = [
              "-s"
              "-w"
            ];

            meta = {
              description = "Reconcile portable Skill manifests with installed Skills";
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
