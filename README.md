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

Mount the `.agents` parent into a synthetic home. If the host parent is absent,
use an empty temporary directory so the CLI can report the missing scan root
instead of Docker rejecting a nonexistent bind source. The read-only container
filesystem and bind mount preserve the CLI's no-write boundary. For a non-root
user on a rootful Docker daemon, pass the invoking user's numeric UID and
primary GID so the CLI does not default to root and bypass DAC checks.
Supplementary groups and user-namespace mappings remain daemon-specific:

```sh
agents_dir="$HOME/.agents"
if ! test -d "$agents_dir"; then
  agents_dir="$(mktemp -d)"
  trap 'rmdir "$agents_dir"' EXIT
fi
docker run --rm --read-only \
  --user "$(id -u):$(id -g)" \
  --env HOME=/home/skills \
  --mount \
    type=bind,src="$agents_dir",dst=/home/skills/.agents,readonly \
  skills-reconcile:local list
```

This minimal mount covers directories and relative symlinks whose targets
resolve inside the mounted `.agents` directory. Absolute symlinks and symlinks
to directories outside `.agents` require an additional read-only mount at the
path where each target resolves inside the container. For an absolute symlink,
mount the target at the same absolute path:

```sh
agents_dir="$HOME/.agents"
if ! test -d "$agents_dir"; then
  agents_dir="$(mktemp -d)"
  trap 'rmdir "$agents_dir"' EXIT
fi
external_skill="/absolute/path/to/external-skill"
docker run --rm --read-only \
  --user "$(id -u):$(id -g)" \
  --env HOME=/home/skills \
  --mount \
    type=bind,src="$agents_dir",dst=/home/skills/.agents,readonly \
  --mount \
    type=bind,src="$external_skill",dst="$external_skill",readonly \
  skills-reconcile:local list
```

Repeat the additional mount for every external target. Set `dst` to the target
path after resolving a relative symlink from its actual parent in the
container. Resolve the scan-root link `.agents/skills` from
`/home/skills/.agents`; resolve a Skill entry inside that root from
`/home/skills/.agents/skills`.

The command exits with status `0` on success. Invalid arguments, an invalid
`HOME`, an unavailable scan root, or an unreadable recognized entry produce a
diagnostic on stderr and status `1`; discovery failures do not print partial
results.
