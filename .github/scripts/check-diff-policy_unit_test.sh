#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=check-diff-policy.sh
source "$script_dir/check-diff-policy.sh"

fail_test() {
  printf 'check-diff-policy unit test: %s\n' "$*" >&2
  exit 1
}

assert_equal() {
  local expected=$1
  local actual=$2
  local label=$3
  [[ "$actual" == "$expected" ]] ||
    fail_test "$label: expected <$expected>, got <$actual>"
}

assert_class() {
  local path=$1
  local base_mode=$2
  local head_mode=$3
  local expected=$4
  local actual

  actual=$(classify_change "$path" "$base_mode" "$head_mode")
  assert_equal "$expected" "$actual" "classification for $path"
}

assert_forbidden() {
  local path=$1
  local expected=$2
  local actual

  if ! actual=$(forbidden_path_message "$path"); then
    fail_test "expected forbidden path: $path"
  fi
  assert_equal "$expected" "$actual" "forbidden message for $path"
}

assert_allowed_path() {
  local path=$1
  local actual

  if actual=$(forbidden_path_message "$path"); then
    fail_test "unexpected forbidden path $path: $actual"
  fi
}

assert_sensitive() {
  local line=$1
  is_sensitive_line "$line" || fail_test "expected sensitive line: $line"
}

assert_safe_line() {
  local line=$1
  if is_sensitive_line "$line"; then
    fail_test "unexpected sensitive line: $line"
  fi
}

test_classification_precedence() {
  assert_class cmd/app/main.go 100644 100644 implementation
  assert_class cmd/app/main_test.go 100644 100644 test
  assert_class tests/helper.sh 100644 100644 test
  assert_class .agents/skills/example/evals/evals.json 100644 100644 test
  assert_class .agents/skills/example/evals/run.sh 100644 100644 test

  assert_class internal/example/testdata/fixture.json '' 100644 fixture
  assert_class internal/example/testdata/run.sh '' 100644 implementation
  assert_class internal/example/testdata/tool '' 100755 implementation

  assert_class flake.lock 100644 100644 generated
  assert_class flake.lock 100644 100755 implementation
  assert_class go.sum 100644 100644 generated
  assert_class tools/skills/package-lock.json 100644 100644 generated

  assert_class docs/guide.md 100644 100644 documentation
  assert_class README.md 100644 100644 documentation
  assert_class AGENTS.md 100644 100644 documentation
  assert_class .agents/skills/example/SKILL.md 100644 100644 documentation
  assert_class .agents/skills/example/README.md 100644 100644 documentation
  assert_class .agents/skills/example/references/note.md 100644 100644 documentation
  assert_class .agents/skills/example/agents/guide.md 100644 100644 documentation
  assert_class .agents/skills/example/SKILL.md 100644 100755 implementation
  assert_class README.md 100755 '' implementation

  assert_class docs/install.sh 100644 100644 implementation
  assert_class docs/tool 100644 100755 implementation
  assert_class docs/tool 100755 '' implementation
  assert_class .agents/skills/example/references/install.sh 100644 100644 implementation
  assert_class .agents/skills/example/agents/openai.yaml 100644 100644 implementation
  assert_class unknown.data 100644 100644 implementation
}

test_forbidden_paths() {
  assert_forbidden .worktrees/skills 'source skills worktree is not allowed'
  assert_forbidden .worktrees/skills/SKILL.md 'source skills worktree is not allowed'
  assert_forbidden skills 'root skills content is not allowed'
  assert_forbidden skills/example/SKILL.md 'root skills content is not allowed'
  assert_forbidden skills-manifest.json 'a root skills-manifest.json is not allowed'
  assert_forbidden state/cache.json 'root machine state is not allowed'
  assert_forbidden node_modules 'node_modules content is not allowed'
  assert_forbidden vendor/node_modules/example/index.js 'node_modules content is not allowed'
  assert_forbidden skills-reconcile 'the built skills-reconcile binary is not allowed'
  assert_forbidden .skill-lock.json 'machine state is not allowed outside synthetic testdata'
  assert_forbidden cache/workspace-projections.json 'machine state is not allowed outside synthetic testdata'

  assert_allowed_path internal/example/testdata/.skill-lock.json
  assert_allowed_path internal/example/testdata/workspace-projections.json
  assert_allowed_path .worktrees/other/reference.md
  assert_allowed_path cmd/skills-reconcile/main.go
  assert_allowed_path docs/skills-manifest.json
}

test_added_line_extraction() {
  local patch
  local actual
  local expected

  printf -v patch '%s\n' \
    'diff --git a/file.txt b/file.txt' \
    '--- a/file.txt' \
    '+++ b/file.txt' \
    '@@ -1,2 +1,3 @@' \
    ' context' \
    '-removed' \
    '+plain addition' \
    '+++ content resembling a file header' \
    '@@ -8 +9 @@' \
    '+second addition' \
    '\ No newline at end of file'

  actual=$(printf '%s' "$patch" | extract_added_lines)
  expected=$(printf '%s\n' \
    'plain addition' \
    '++ content resembling a file header' \
    'second addition')
  assert_equal "$expected" "$actual" 'added-line extraction'
}

test_sensitive_lines() {
  local home_dir=home
  local users_dir=Users
  local root_dir=root
  local key_type=PRIVATE
  local github_prefix=ghp_
  local fine_grained_prefix=github_pat_
  local aws_prefix=AKIA
  local openai_prefix=sk-
  local scheme=https
  local file_scheme=file
  local database_scheme=postgresql
  local compound_scheme=git+https

  assert_sensitive "/${home_dir}/demo/private"
  assert_sensitive "HOME=/${home_dir}/demo"
  assert_sensitive "'/${home_dir}/日本語/private'"
  assert_sensitive "(/${home_dir}/demo/private)"
  assert_sensitive "[/${home_dir}/demo/private]"
  assert_sensitive "/${users_dir}/日本語/private"
  assert_sensitive "/${root_dir}/.agents/private"
  assert_sensitive "HOME=/${root_dir}"
  assert_sensitive "C:\\${users_dir}\\demo\\private"
  assert_sensitive "C:\\\\${users_dir}\\\\demo\\\\private"
  assert_sensitive "C:/${users_dir}/demo/private"
  assert_sensitive "\"path\":\"\\/${home_dir}\\/demo\\/private\""
  assert_sensitive "${file_scheme}:///${home_dir}/demo/private"
  assert_sensitive "${file_scheme}:///${users_dir}/demo/private"
  assert_sensitive "${file_scheme}:///${root_dir}/private"
  assert_sensitive "${file_scheme}://localhost/${home_dir}/demo/private"
  assert_sensitive "${file_scheme}:///C:/${users_dir}/demo/private"
  assert_sensitive "\"uri\":\"${file_scheme}:\\/\\/\\/${home_dir}\\/demo\\/private\""
  assert_sensitive "-----BEGIN ${key_type} KEY-----"
  assert_sensitive "${github_prefix}aaaaaaaaaaaaaaaaaaaa"
  assert_sensitive "${fine_grained_prefix}aaaaaaaaaaaaaaaaaaaa"
  assert_sensitive "${aws_prefix}ABCDEFGHIJKLMNOP"
  assert_sensitive "${openai_prefix}aaaaaaaaaaaaaaaaaaaa"
  assert_sensitive "${scheme}://user:pass@example.invalid/repo"
  assert_sensitive "${database_scheme}://demo:synthetic@db.example.invalid/app"
  assert_sensitive "${compound_scheme}://demo:synthetic@example.invalid/repo"
  assert_sensitive "\"url\":\"${database_scheme}:\\/\\/demo:synthetic@db.example.invalid\\/app\""

  assert_safe_line 'https://docs.example.invalid/home/getting-started'
  assert_safe_line 'https://docs.example.invalid/Users/getting-started'
  assert_safe_line 'https://docs.example.invalid/root/guide'
  assert_safe_line 'https://docs.example.invalid/file:///home/getting-started'
  assert_safe_line 'https:\/\/docs.example.invalid\/home\/getting-started'
  assert_safe_line 'profile:///home/demo/private'
  assert_safe_line '/homepage/example'
  assert_safe_line '/homebrew/bin'
  assert_safe_line '/rooted/path'
  assert_safe_line 'docs/home/demo'
  assert_safe_line 'prefix/home/demo'
  assert_safe_line "/${home_dir}"
  assert_safe_line "/${home_dir}/"
  assert_safe_line "/${users_dir}"
  assert_safe_line 'https://user@example.invalid/repo'
  assert_safe_line 'postgresql://demo@db.example.invalid/app'
  assert_safe_line 'https://example.invalid/path:segment@example.invalid'
  assert_safe_line "${github_prefix}short"
  assert_safe_line "x${github_prefix}aaaaaaaaaaaaaaaaaaaa"
  assert_safe_line "${aws_prefix}SHORT"
  assert_safe_line "${openai_prefix}short"
}

test_classification_precedence
test_forbidden_paths
test_added_line_extraction
test_sensitive_lines

printf 'check-diff-policy unit tests passed\n'
