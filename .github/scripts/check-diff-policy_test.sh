#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
policy_script=$script_dir/check-diff-policy.sh
scratch=$(mktemp -d "${TMPDIR:-/tmp}/skills-reconcile-diff-policy.XXXXXX")
trap 'rm -rf -- "$scratch"' EXIT

fail_test() {
  printf 'check-diff-policy integration test: %s\n' "$*" >&2
  exit 1
}

new_repo() {
  repo=$(mktemp -d "$scratch/repo.XXXXXX")
  git -C "$repo" init --quiet
  git -C "$repo" config user.name 'Synthetic Test'
  git -C "$repo" config user.email 'synthetic@example.invalid'
  printf 'base\n' >"$repo/main.go"
  git -C "$repo" add main.go
  git -C "$repo" commit --quiet -m base
  base=$(git -C "$repo" rev-parse HEAD)
}

commit_changes() {
  git -C "$repo" add --all
  git -C "$repo" commit --quiet -m changes
}

run_policy_range() {
  local base_ref=$1
  local head_ref=$2
  (cd "$repo" && "$policy_script" "$base_ref" "$head_ref" 2>&1)
}

expect_success() {
  local base_ref=${1:-$base}
  local head_ref=${2:-HEAD}
  if ! policy_output=$(run_policy_range "$base_ref" "$head_ref"); then
    fail_test "policy unexpectedly failed: $policy_output"
  fi
}

expect_failure() {
  local expected=$1
  local base_ref=${2:-$base}
  local head_ref=${3:-HEAD}
  if policy_output=$(run_policy_range "$base_ref" "$head_ref"); then
    fail_test "policy unexpectedly accepted: $expected"
  fi
  assert_contains "$policy_output" "$expected"
}

assert_contains() {
  local output=$1
  local expected=$2
  [[ "$output" == *"$expected"* ]] ||
    fail_test "output does not contain <$expected>: $output"
}

assert_not_contains() {
  local output=$1
  local unexpected=$2
  [[ "$output" != *"$unexpected"* ]] ||
    fail_test "output unexpectedly contains <$unexpected>: $output"
}

test_reported_categories() {
  new_repo
  mkdir -p \
    "$repo/cmd/app" \
    "$repo/internal/example/testdata" \
    "$repo/docs" \
    "$repo/.agents/skills/example/agents" \
    "$repo/.agents/skills/example/evals" \
    "$repo/.agents/skills/example/references"
  printf 'change\n' >>"$repo/main.go"
  printf 'package app\n' >"$repo/cmd/app/main_test.go"
  printf '{}\n' >"$repo/internal/example/testdata/fixture.json"
  printf '{}\n' >"$repo/flake.lock"
  printf 'note\n' >"$repo/docs/note.md"
  printf 'skill guide\n' >"$repo/.agents/skills/example/SKILL.md"
  printf 'agent guide\n' >"$repo/.agents/skills/example/agents/guide.md"
  printf 'trigger config\n' >"$repo/.agents/skills/example/agents/openai.yaml"
  printf 'eval\n' >"$repo/.agents/skills/example/evals/evals.json"
  printf 'reference\n' >"$repo/.agents/skills/example/references/note.md"
  commit_changes

  expect_success
  assert_contains "$policy_output" 'Handwritten implementation: 3'
  assert_contains "$policy_output" 'Tests: 2'
  assert_contains "$policy_output" 'Handwritten fixtures: 1'
  assert_contains "$policy_output" 'Generated files counted as implementation: 1'
  assert_contains "$policy_output" 'Documentation: 4'
  assert_contains "$policy_output" 'Pure renames: 0 files'
  assert_contains "$policy_output" 'Handwritten total: 6'
}

test_line_limits() {
  new_repo
  awk 'BEGIN { for (i = 1; i <= 500; i++) print "line" i }' >"$repo/large.go"
  commit_changes
  expect_success
  assert_contains "$policy_output" 'Handwritten implementation: 500'

  new_repo
  awk 'BEGIN { for (i = 1; i <= 501; i++) print "line" i }' >"$repo/large.go"
  commit_changes
  expect_failure 'handwritten implementation exceeds 500 changed lines'

  new_repo
  awk 'BEGIN { for (i = 1; i <= 1000; i++) print "line" i }' >"$repo/large_test.go"
  commit_changes
  expect_success
  assert_not_contains "$policy_output" 'WARNING'

  new_repo
  awk 'BEGIN { for (i = 1; i <= 1001; i++) print "line" i }' >"$repo/large_test.go"
  commit_changes
  expect_success
  assert_contains "$policy_output" 'WARNING: handwritten total exceeds 1000 changed lines'
}

test_unverified_generated_limit() {
  new_repo
  awk 'BEGIN { for (i = 1; i <= 501; i++) print "line" i }' >"$repo/flake.lock"
  commit_changes
  expect_failure 'handwritten implementation exceeds 500 changed lines'
}

test_literal_git_paths() {
  new_repo
  local unusual_doc=$'docs/日本語\tline\nbreak.md'
  mkdir -p "$repo/docs"
  printf 'note\n' >"$repo/$unusual_doc"
  commit_changes
  expect_success
  assert_contains "$policy_output" 'Documentation: 1'

  new_repo
  local prohibited=$'skills/日本語\tline\nbreak/SKILL.md'
  mkdir -p "$(dirname "$repo/$prohibited")"
  printf 'synthetic\n' >"$repo/$prohibited"
  commit_changes
  expect_failure 'root skills content is not allowed'
}

test_rename_records() {
  new_repo
  printf 'rename\n' >"$repo/old.txt"
  git -C "$repo" add old.txt
  git -C "$repo" commit --quiet -m rename-base
  base=$(git -C "$repo" rev-parse HEAD)
  local renamed=$'renamed-日本語\tline\nbreak.txt'
  git -C "$repo" mv old.txt "$renamed"
  commit_changes
  expect_success
  assert_contains "$policy_output" 'Pure renames: 1 files'

  new_repo
  printf 'rename\n' >"$repo/old.txt"
  git -C "$repo" add old.txt
  git -C "$repo" commit --quiet -m rename-base
  base=$(git -C "$repo" rev-parse HEAD)
  local prohibited=$'skills/日本語\tline\nbreak.txt'
  mkdir -p "$repo/skills"
  git -C "$repo" mv old.txt "$prohibited"
  commit_changes
  expect_failure 'root skills content is not allowed'

  new_repo
  printf 'rename\n' >"$repo/tool"
  chmod +x "$repo/tool"
  git -C "$repo" add tool
  git -C "$repo" commit --quiet -m rename-base
  base=$(git -C "$repo" rev-parse HEAD)
  mkdir -p "$repo/docs"
  git -C "$repo" mv tool docs/tool
  chmod -x "$repo/docs/tool"
  commit_changes
  expect_success
  assert_contains "$policy_output" 'Pure renames: 0 files'
}

test_binary_rejection() {
  new_repo
  printf '\0' >"$repo/artifact.bin"
  commit_changes
  expect_failure 'binary changes are not allowed'
}

test_executable_documentation() {
  new_repo
  mkdir -p "$repo/docs"
  awk 'BEGIN { for (i = 1; i <= 501; i++) print "line" i }' >"$repo/docs/tool"
  chmod +x "$repo/docs/tool"
  commit_changes
  expect_failure 'handwritten implementation exceeds 500 changed lines'

  new_repo
  mkdir -p "$repo/docs"
  awk 'BEGIN { for (i = 1; i <= 501; i++) print "line" i }' >"$repo/docs/tool"
  chmod +x "$repo/docs/tool"
  commit_changes
  base=$(git -C "$repo" rev-parse HEAD)
  git -C "$repo" rm --quiet docs/tool
  commit_changes
  expect_failure 'handwritten implementation exceeds 500 changed lines'
}

test_forbidden_state_fixture() {
  new_repo
  mkdir -p "$repo/internal/example/testdata"
  printf '{}\n' >"$repo/internal/example/testdata/.skill-lock.json"
  printf '{}\n' >"$repo/internal/example/testdata/workspace-projections.json"
  commit_changes
  expect_success

  new_repo
  mkdir -p "$repo/cache"
  printf '{}\n' >"$repo/cache/.skill-lock.json"
  commit_changes
  expect_failure 'machine state is not allowed outside synthetic testdata'
}

test_added_content_scan() {
  new_repo
  printf 'https://docs.example.invalid/home/getting-started\n' >"$repo/link.md"
  commit_changes
  expect_success

  new_repo
  local home_dir=home
  printf '++ /%s/demo/private\n' "$home_dir" >"$repo/leak.txt"
  commit_changes
  expect_failure 'added content contains a credential or machine-specific path'

  new_repo
  local file_scheme=file
  printf '%s:///%s/demo/private\n' "$file_scheme" "$home_dir" >"$repo/leak.txt"
  commit_changes
  expect_failure 'added content contains a credential or machine-specific path'

  new_repo
  local token_prefix=ghp_
  printf '%s%s\n' "$token_prefix" aaaaaaaaaaaaaaaaaaaa >"$repo/leak.txt"
  git -C "$repo" add leak.txt
  git -C "$repo" commit --quiet -m sensitive-base
  base=$(git -C "$repo" rev-parse HEAD)
  printf 'safe\n' >"$repo/leak.txt"
  commit_changes
  expect_success
}

test_merge_base_range() {
  new_repo
  local common=$base
  git -C "$repo" switch --quiet -c feature
  printf 'feature\n' >"$repo/feature.go"
  commit_changes
  local feature_head
  feature_head=$(git -C "$repo" rev-parse HEAD)

  git -C "$repo" switch --quiet --detach "$common"
  awk 'BEGIN { for (i = 1; i <= 501; i++) print "line" i }' >"$repo/base-only.go"
  commit_changes
  local base_tip
  base_tip=$(git -C "$repo" rev-parse HEAD)

  expect_success "$base_tip" "$feature_head"
  assert_contains "$policy_output" 'Handwritten implementation: 1'
}

test_invalid_revisions() {
  new_repo
  expect_failure 'invalid base revision' missing-revision HEAD
  expect_failure 'invalid head revision' "$base" missing-revision
}

test_reported_categories
test_line_limits
test_unverified_generated_limit
test_literal_git_paths
test_rename_records
test_binary_rejection
test_executable_documentation
test_forbidden_state_fixture
test_added_content_scan
test_merge_base_range
test_invalid_revisions

printf 'check-diff-policy integration tests passed\n'
