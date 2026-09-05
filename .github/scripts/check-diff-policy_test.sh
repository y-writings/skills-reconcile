#!/usr/bin/env bash
set -euo pipefail

policy_script=$(cd "$(dirname "$0")" && pwd)/check-diff-policy.sh
scratch=$(mktemp -d "${TMPDIR:-/tmp}/skills-reconcile-diff-policy.XXXXXX")
trap 'rm -rf -- "$scratch"' EXIT

fail() {
  printf 'check-diff-policy test: %s\n' "$*" >&2
  exit 1
}

new_repo() {
  repo=$(mktemp -d "$scratch/repo.XXXXXX")
  git -C "$repo" init --quiet
  git -C "$repo" config user.name "Synthetic Test"
  git -C "$repo" config user.email "synthetic@example.invalid"
  printf 'base\n' >"$repo/main.go"
  git -C "$repo" add main.go
  git -C "$repo" commit --quiet -m base
  base=$(git -C "$repo" rev-parse HEAD)
}

commit_changes() {
  git -C "$repo" add --all
  git -C "$repo" commit --quiet -m changes
}

assert_contains() {
  local output=$1
  local expected=$2
  [[ "$output" == *"$expected"* ]] || fail "output does not contain: $expected"
}

expect_failure() {
  local expected=$1
  local output
  if output=$(cd "$repo" && "$policy_script" "$base" HEAD 2>&1); then
    fail "policy unexpectedly accepted: $expected"
  fi
  assert_contains "$output" "$expected"
}

test_line_categories() {
  new_repo
  mkdir -p \
    "$repo/cmd/app" \
    "$repo/internal/example/testdata" \
    "$repo/docs" \
    "$repo/.agents/skills/example/agents" \
    "$repo/.agents/skills/example/evals" \
    "$repo/.agents/skills/example/references"
  local unicode_doc=$'docs/日本語\tline\nbreak.md'
  printf 'change\n' >>"$repo/main.go"
  printf 'package app\n' >"$repo/cmd/app/main_test.go"
  printf '{}\n' >"$repo/internal/example/testdata/fixture.json"
  printf '{}\n' >"$repo/flake.lock"
  printf 'note\n' >"$repo/docs/note.md"
  printf 'unicode note\n' >"$repo/$unicode_doc"
  printf 'skill guide\n' >"$repo/.agents/skills/example/SKILL.md"
  printf 'agent guide\n' >"$repo/.agents/skills/example/agents/guide.md"
  printf 'trigger config\n' >"$repo/.agents/skills/example/agents/openai.yaml"
  printf 'eval\n' >"$repo/.agents/skills/example/evals/evals.json"
  printf 'reference\n' >"$repo/.agents/skills/example/references/note.md"
  printf 'rename\n' >"$repo/old.txt"
  git -C "$repo" add old.txt
  git -C "$repo" commit --quiet -m rename-base
  base=$(git -C "$repo" rev-parse HEAD)
  git -C "$repo" mv old.txt new.txt
  commit_changes

  output=$(cd "$repo" && "$policy_script" "$base" HEAD 2>&1)
  assert_contains "$output" "Handwritten implementation: 3"
  assert_contains "$output" "Tests: 2"
  assert_contains "$output" "Handwritten fixtures: 1"
  assert_contains "$output" "Generated files counted as implementation: 1"
  assert_contains "$output" "Documentation: 5"
  assert_contains "$output" "Pure renames: 1 files"
  assert_contains "$output" "Handwritten total: 6"
}

test_implementation_limit() {
  new_repo
  awk 'BEGIN { for (i = 1; i <= 501; i++) print "line" i }' >"$repo/large.go"
  commit_changes
  expect_failure "handwritten implementation exceeds 500 changed lines"
}

test_handwritten_warning() {
  new_repo
  awk 'BEGIN { for (i = 1; i <= 1001; i++) print "line" i }' >"$repo/large_test.go"
  commit_changes
  output=$(cd "$repo" && "$policy_script" "$base" HEAD 2>&1)
  assert_contains "$output" "WARNING: handwritten total exceeds 1000 changed lines"
}

test_unverified_generated_limit() {
  local generated_path
  for generated_path in flake.lock go.sum tools/skills/package-lock.json; do
    new_repo
    mkdir -p "$repo/$(dirname "$generated_path")"
    awk 'BEGIN { for (i = 1; i <= 501; i++) print "line" i }' >"$repo/$generated_path"
    commit_changes
    expect_failure "handwritten implementation exceeds 500 changed lines"
  done
}

test_skill_helper_limit() {
  new_repo
  local helper_path=.agents/skills/example/scripts/install.sh
  mkdir -p "$repo/$(dirname "$helper_path")"
  awk 'BEGIN { for (i = 1; i <= 501; i++) print "line" i }' >"$repo/$helper_path"
  commit_changes
  expect_failure "handwritten implementation exceeds 500 changed lines"
}

test_documentation_implementation_limit() {
  local path
  for path in \
    docs/install.sh \
    docs/tool \
    .agents/skills/example/references/install.sh; do
    new_repo
    mkdir -p "$repo/$(dirname "$path")"
    awk 'BEGIN { for (i = 1; i <= 501; i++) print "line" i }' >"$repo/$path"
    if [[ "$path" == docs/tool ]]; then
      chmod +x "$repo/$path"
    fi
    commit_changes
    expect_failure "handwritten implementation exceeds 500 changed lines"
  done

  new_repo
  path=docs/tool
  mkdir -p "$repo/$(dirname "$path")"
  awk 'BEGIN { for (i = 1; i <= 501; i++) print "line" i }' >"$repo/$path"
  chmod +x "$repo/$path"
  commit_changes
  base=$(git -C "$repo" rev-parse HEAD)
  git -C "$repo" rm --quiet -- "$path"
  commit_changes
  expect_failure "handwritten implementation exceeds 500 changed lines"
}

test_forbidden_paths() {
  local path
  for path in \
    skills/example/SKILL.md \
    skills-manifest.json \
    state/cache.json \
    .skill-lock.json \
    workspace-projections.json \
    vendor/node_modules/example/index.js \
    skills-reconcile; do
    new_repo
    mkdir -p "$(dirname "$repo/$path")"
    printf 'synthetic\n' >"$repo/$path"
    commit_changes
    expect_failure "not allowed"
  done

  new_repo
  path=$'skills/日本語\tline\nbreak/SKILL.md'
  mkdir -p "$(dirname "$repo/$path")"
  printf 'synthetic\n' >"$repo/$path"
  commit_changes
  expect_failure "root skills content is not allowed"
}

test_synthetic_state_fixture() {
  new_repo
  mkdir -p "$repo/internal/example/testdata"
  printf '{}\n' >"$repo/internal/example/testdata/.skill-lock.json"
  printf '{}\n' >"$repo/internal/example/testdata/workspace-projections.json"
  commit_changes
  (cd "$repo" && "$policy_script" "$base" HEAD >/dev/null)
}

test_binary_rejection() {
  new_repo
  printf '\0' >"$repo/artifact.bin"
  commit_changes
  expect_failure "binary changes are not allowed"
}

test_sensitive_content() {
  local kind
  for kind in private-key unix-user-path unicode-unix-user-path unicode-macos-user-path root-home-path quoted-root-home-path unterminated-home-path windows-user-path access-token credential-url diff-header-user-path diff-header-token; do
    new_repo
    case "$kind" in
      private-key) printf '%s%s\n' '-----BEGIN ' 'PRIVATE KEY-----' >"$repo/leak.txt" ;;
      unix-user-path) printf '/%s/%s/private\n' Users demo >"$repo/leak.txt" ;;
      unicode-unix-user-path) printf '/%s/%s/private\n' home 日本語 >"$repo/leak.txt" ;;
      unicode-macos-user-path) printf '/%s/%s/private\n' Users 日本語 >"$repo/leak.txt" ;;
      root-home-path) printf '/%s/.agents/private\n' root >"$repo/leak.txt" ;;
      quoted-root-home-path) printf 'HOME="/%s"\n' root >"$repo/leak.txt" ;;
      unterminated-home-path) printf 'HOME=/%s/%s\n' home demo >"$repo/leak.txt" ;;
      windows-user-path) printf '%s:\\%s\\%s\\private\n' C Users demo >"$repo/leak.txt" ;;
      access-token) printf '%s%s\n' ghp_ aaaaaaaaaaaaaaaaaaaa >"$repo/leak.txt" ;;
      credential-url) printf '%s://%s:%s@%s/repo\n' https user pass example.invalid >"$repo/leak.txt" ;;
      diff-header-user-path) printf '++ /%s/%s/private\n' home demo >"$repo/leak.txt" ;;
      diff-header-token) printf '++ %s%s\n' ghp_ aaaaaaaaaaaaaaaaaaaa >"$repo/leak.txt" ;;
    esac
    commit_changes
    expect_failure "added content contains a credential or machine-specific path"
  done
}

test_line_categories
test_implementation_limit
test_handwritten_warning
test_unverified_generated_limit
test_skill_helper_limit
test_documentation_implementation_limit
test_forbidden_paths
test_synthetic_state_fixture
test_binary_rejection
test_sensitive_content

printf 'check-diff-policy tests passed\n'
