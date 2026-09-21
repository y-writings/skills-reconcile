# skills-reconcile

`skills-reconcile` lists Agent Skills installed directly under
`$HOME/.agents/skills`.

The current milestone is intentionally read-only. It does not scan other
directories, start a TUI, copy Skills, inspect installation provenance, or use
the network.

## List Skills

Each recognized Skill name is printed on its own line in bytewise ascending
order:

```console
$ skills-reconcile list
alpha
zed
```

A Skill is a directory, or a symlink to a directory, whose name follows the
[documented Skill naming rules](docs/plan/cli-contract.md) and which contains a
regular, non-symlink `SKILL.md` file. Run `skills-reconcile --help` for the
supported command form.

## Run with Go

From the repository root:

```sh
go run ./cmd/skills-reconcile list
```

## Run with Nix

The named and default flake apps both run the same CLI:

```sh
nix run .#skills-reconcile -- list
nix run .#default -- list
```

To build the package without running it:

```sh
nix build .#skills-reconcile
```

## Run in a container

Build the runtime image:

```sh
docker build --target runtime --tag skills-reconcile:local .
```

Mount the Skill directory into a synthetic home. The read-only container
filesystem and bind mount preserve the CLI's no-write boundary:

```sh
skills_dir="$HOME/.agents/skills"
docker run --rm --read-only \
  --env HOME=/home/skills \
  --mount \
    type=bind,src="$skills_dir",dst=/home/skills/.agents/skills,readonly \
  skills-reconcile:local list
```

This minimal mount covers directories and relative symlinks whose targets
resolve inside the mounted Skill directory. Absolute symlinks and symlinks to
directories outside `.agents/skills` require an additional read-only mount at
the path where each target resolves inside the container. For an absolute
symlink, mount the target at the same absolute path:

```sh
skills_dir="$HOME/.agents/skills"
external_skill="/absolute/path/to/external-skill"
docker run --rm --read-only \
  --env HOME=/home/skills \
  --mount \
    type=bind,src="$skills_dir",dst=/home/skills/.agents/skills,readonly \
  --mount \
    type=bind,src="$external_skill",dst="$external_skill",readonly \
  skills-reconcile:local list
```

Repeat the additional mount for every external target. For a relative symlink,
set `dst` to the path where the link resolves from
`/home/skills/.agents/skills`.

The command exits with status `0` on success. Invalid arguments, an invalid
`HOME`, an unavailable scan root, or an unreadable recognized entry produce a
diagnostic on stderr and status `1`; discovery failures do not print partial
results.
