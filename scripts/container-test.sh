#!/bin/sh

set -eu

image='skills-reconcile:local'

run_cli() {
  docker run --rm --read-only "$@" \
    --env HOME=/home/skills \
    "$image" list
}

assert_output() (
  assertion_label="$1"
  assertion_expected="$2"
  shift 2

  assertion_actual="$(run_cli "$@")"
  if test "$assertion_actual" != "$assertion_expected"; then
    printf 'unexpected %s output: %s\n' \
      "$assertion_label" "$assertion_actual" >&2
    exit 1
  fi
)

assert_failure() (
  assertion_label="$1"
  assertion_stderr="$2"
  shift 2

  set +e
  assertion_stdout="$(run_cli "$@" 2>"$assertion_stderr")"
  assertion_status=$?
  set -e
  if test "$assertion_status" -ne 1 \
    || test -n "$assertion_stdout" \
    || test ! -s "$assertion_stderr"; then
    printf 'unexpected %s result: status=%s stdout=%s\n' \
      "$assertion_label" "$assertion_status" "$assertion_stdout" >&2
    exit 1
  fi
)

cleanup() {
  chmod u+rwx "$smoke_home/group-agents" 2>/dev/null || true
  chmod -R u+rwx "$smoke_home/group-agents" 2>/dev/null || true
  chmod u+rwx "$smoke_home/unreadable-agents/skills/z-locked" \
    2>/dev/null || true
  chmod u+rwx "$smoke_home/external-tree/restricted" 2>/dev/null || true
  chmod -R u+w "$smoke_home" "$external_tree" 2>/dev/null || true
  rm -rf "$smoke_home" "$external_tree"
}

create_fixtures() {
  mkdir -p \
    "$smoke_home/.agents/skills/alpha" \
    "$smoke_home/.agents/skills/zed" \
    "$smoke_home/group-agents/skills/group-skill" \
    "$smoke_home/ancestor-agents/skills" \
    "$smoke_home/external-tree/restricted/blocked-skill" \
    "$smoke_home/relative-root-agents" \
    "$smoke_home/relative-root-target/root-skill" \
    "$smoke_home/relative-entry-agents/skills" \
    "$smoke_home/relative-entry-target/relative-skill" \
    "$smoke_home/unreadable-agents/skills/a-valid" \
    "$smoke_home/unreadable-agents/skills/z-locked" \
    "$smoke_home/missing-agents" \
    "$external_skill"
  touch \
    "$smoke_home/.agents/skills/alpha/SKILL.md" \
    "$smoke_home/.agents/skills/zed/SKILL.md" \
    "$external_skill/SKILL.md" \
    "$smoke_home/group-agents/skills/group-skill/SKILL.md" \
    "$smoke_home/external-tree/restricted/blocked-skill/SKILL.md" \
    "$smoke_home/relative-root-target/root-skill/SKILL.md" \
    "$smoke_home/relative-entry-target/relative-skill/SKILL.md" \
    "$smoke_home/unreadable-agents/skills/a-valid/SKILL.md" \
    "$smoke_home/unreadable-agents/skills/z-locked/SKILL.md"
  ln -s "$external_skill" "$smoke_home/.agents/skills/linked-skill"
  ln -s \
    "$smoke_home/external-tree/restricted/blocked-skill" \
    "$smoke_home/ancestor-agents/skills/blocked-skill"
  ln -s '../relative-root-target' "$smoke_home/relative-root-agents/skills"
  ln -s '../../relative-entry-target/relative-skill' \
    "$smoke_home/relative-entry-agents/skills/relative-linked-skill"
  chmod 000 "$smoke_home/unreadable-agents/skills/z-locked"
  chmod 000 "$smoke_home/external-tree/restricted"
}

prepare_group_fixture() {
  if test -z "$supplementary_gid" || test "$(uname -s)" != 'Linux'; then
    return
  fi

  group_probe_enabled=1
  chgrp -R "$supplementary_gid" "$smoke_home/group-agents"
  chmod 040 "$smoke_home/group-agents/skills/group-skill/SKILL.md"
  chmod 050 "$smoke_home/group-agents/skills/group-skill"
  chmod 050 "$smoke_home/group-agents/skills"
  chmod 050 "$smoke_home/group-agents"
}

test_basic_discovery() (
  assert_output 'container CLI' "$(printf 'alpha\nlinked-skill\nzed\n')" \
    "$@" \
    --mount \
      "type=bind,src=$smoke_home/.agents,dst=/home/skills/.agents,readonly" \
    --mount \
      "type=bind,src=$external_tree,dst=$external_tree,readonly"
)

test_relative_scan_root() (
  assert_output 'relative scan root' 'root-skill' \
    "$@" \
    --mount \
      "type=bind,src=$smoke_home/relative-root-agents,dst=/home/skills/.agents,readonly" \
    --mount \
      "type=bind,src=$smoke_home/relative-root-target,dst=/home/skills/relative-root-target,readonly"
)

test_relative_skill_entry() (
  assert_output 'relative Skill entry' 'relative-linked-skill' \
    "$@" \
    --mount \
      "type=bind,src=$smoke_home/relative-entry-agents,dst=/home/skills/.agents,readonly" \
    --mount \
      "type=bind,src=$smoke_home/relative-entry-target,dst=/home/skills/relative-entry-target,readonly"
)

test_supplementary_group() (
  if test "$group_probe_enabled" -ne 1; then
    return
  fi

  group_probe_uid=65534
  if test "$container_uid" -eq "$group_probe_uid"; then
    group_probe_uid=65533
  fi
  # Replace the leading --user value while preserving --group-add arguments.
  shift 2
  assert_output 'supplementary group' 'group-skill' \
    --user "$group_probe_uid:$container_gid" \
    "$@" \
    --mount \
      "type=bind,src=$smoke_home/group-agents,dst=/home/skills/.agents,readonly"
)

test_external_ancestor_permissions() (
  if test "$container_uid" -eq 0 || test "$(uname -s)" != 'Linux'; then
    return
  fi

  assert_failure 'external ancestor' "$smoke_home/ancestor-stderr" \
    "$@" \
    --mount \
      "type=bind,src=$smoke_home/ancestor-agents,dst=/home/skills/.agents,readonly" \
    --mount \
      "type=bind,src=$smoke_home/external-tree,dst=$smoke_home/external-tree,readonly"
)

test_missing_scan_root() (
  assert_failure 'missing scan root' "$smoke_home/missing-stderr" \
    "$@" \
    --mount \
      "type=bind,src=$smoke_home/missing-agents,dst=/home/skills/.agents,readonly"
)

test_unreadable_skill() (
  if test "$container_uid" -eq 0; then
    return
  fi

  assert_failure 'unreadable Skill' "$smoke_home/unreadable-stderr" \
    "$@" \
    --mount \
      "type=bind,src=$smoke_home/unreadable-agents,dst=/home/skills/.agents,readonly"
)

docker build --target runtime --tag "$image" .

smoke_home="$(mktemp -d)"
external_tree="$(mktemp -d)"
external_skill="$external_tree/external-skill"
container_uid="$(id -u)"
container_gid="$(id -g)"
supplementary_gid=''
group_probe_enabled=0
trap cleanup EXIT

set -- --user "$container_uid:$container_gid"
for group_id in $(id -G); do
  if test "$group_id" -ne "$container_gid"; then
    set -- "$@" --group-add "$group_id"
    if test -z "$supplementary_gid"; then
      supplementary_gid="$group_id"
    fi
  fi
done

create_fixtures
prepare_group_fixture
test_basic_discovery "$@"
test_relative_scan_root "$@"
test_relative_skill_entry "$@"
test_supplementary_group "$@"
test_external_ancestor_permissions "$@"
test_missing_scan_root "$@"
test_unreadable_skill "$@"
