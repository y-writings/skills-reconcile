#!/usr/bin/env bash

fail() {
  printf 'diff-policy: %s\n' "$*" >&2
  exit 1
}

resolve_merge_base() {
  local base=$1
  local head=$2

  git rev-parse --verify "${base}^{commit}" >/dev/null 2>&1 ||
    fail 'invalid base revision'
  git rev-parse --verify "${head}^{commit}" >/dev/null 2>&1 ||
    fail 'invalid head revision'
  git merge-base "$base" "$head" || fail 'base and head have no merge base'
}

object_mode_at() {
  local revision=$1
  local path=$2

  git ls-tree --format='%(objectmode)' "$revision" -- ":(literal)$path"
}

classify_change() {
  local path=$1
  local base_mode=$2
  local head_mode=$3

  case "$path" in
    *_test.go | *_test.sh | tests/* | */tests/* | .agents/skills/*/evals/*)
      printf 'test\n'
      return
      ;;
  esac

  if [[ "$path" == *.sh || "$base_mode" == 100755 || "$head_mode" == 100755 ]]; then
    printf 'implementation\n'
    return
  fi

  case "$path" in
    testdata/* | */testdata/*)
      printf 'fixture\n'
      ;;
    flake.lock | go.sum | tools/skills/package-lock.json)
      printf 'generated\n'
      ;;
    docs/* | README.md | README.* | AGENTS.md | \
      .agents/skills/*/SKILL.md | .agents/skills/*/README.md | \
      .agents/skills/*/references/* | .agents/skills/*/agents/*.md)
      printf 'documentation\n'
      ;;
    *)
      printf 'implementation\n'
      ;;
  esac
}

forbidden_path_message() {
  local path=$1

  case "$path" in
    .worktrees/skills | .worktrees/skills/*)
      printf 'source skills worktree is not allowed\n'
      ;;
    skills | skills/*)
      printf 'root skills content is not allowed\n'
      ;;
    skills-manifest.json)
      printf 'a root skills-manifest.json is not allowed\n'
      ;;
    state | state/*)
      printf 'root machine state is not allowed\n'
      ;;
    node_modules | node_modules/* | */node_modules/*)
      printf 'node_modules content is not allowed\n'
      ;;
    skills-reconcile)
      printf 'the built skills-reconcile binary is not allowed\n'
      ;;
    .skill-lock.json | */.skill-lock.json | workspace-projections.json | */workspace-projections.json)
      case "$path" in
        testdata/* | */testdata/*)
          return 1
          ;;
        *)
          printf 'machine state is not allowed outside synthetic testdata\n'
          ;;
      esac
      ;;
    *)
      return 1
      ;;
  esac
}

extract_added_lines() {
  awk '
    /^diff --git / { in_hunk = 0; next }
    /^@@ / { in_hunk = 1; next }
    in_hunk && /^\+/ { sub(/^\+/, ""); print }
  '
}

is_sensitive_line() {
  local line=$1
  local LC_ALL=C
  local file_uri_boundary
  local path_boundary
  local private_key_pattern
  local credential_url_pattern
  local unix_home_pattern
  local root_home_pattern
  local windows_home_pattern
  local file_uri_unix_home_pattern
  local file_uri_root_home_pattern
  local file_uri_windows_home_pattern
  local github_token_pattern
  local fine_grained_token_pattern
  local aws_key_pattern
  local openai_key_pattern

  line=${line//\\\//\/}
  file_uri_boundary=$'(^|[[:space:]=,"\'`({<]|\\[)'
  path_boundary=$'(^|[[:space:]=:,"\'`({<]|\\[)'
  private_key_pattern='-----BEGIN ([A-Z0-9]+ )*PRIVATE KEY-----'
  credential_url_pattern='(^|[^[:alnum:]+.-])[A-Za-z][A-Za-z0-9+.-]*://[^[:space:]/:@]+:[^[:space:]@]+@'
  unix_home_pattern="${path_boundary}/(Users|home)/[^/[:space:]]+"
  root_home_pattern="${path_boundary}/root([^[:alnum:]_.-]|$)"
  windows_home_pattern="${path_boundary}[A-Za-z]:\\\\+Users\\\\+[^\\\\[:space:]]+"
  file_uri_unix_home_pattern="${file_uri_boundary}file:(//localhost|//)?/(Users|home)/[^/[:space:]]+"
  file_uri_root_home_pattern="${file_uri_boundary}file:(//localhost|//)?/root([^[:alnum:]_.-]|$)"
  file_uri_windows_home_pattern="${file_uri_boundary}file:(//localhost|//)?/[A-Za-z]:/(Users)/[^/[:space:]]+"
  github_token_pattern='(^|[^[:alnum:]_])gh[pousr]_[A-Za-z0-9]{20,}([^[:alnum:]_]|$)'
  fine_grained_token_pattern='(^|[^[:alnum:]_])github_pat_[A-Za-z0-9_]{20,}([^[:alnum:]_]|$)'
  aws_key_pattern='(^|[^[:alnum:]_])AKIA[A-Z0-9]{16}([^[:alnum:]_]|$)'
  openai_key_pattern='(^|[^[:alnum:]_])sk-[A-Za-z0-9]{20,}([^[:alnum:]_]|$)'

  [[ "$line" =~ $private_key_pattern ]] ||
    [[ "$line" =~ $credential_url_pattern ]] ||
    [[ "$line" =~ $unix_home_pattern ]] ||
    [[ "$line" =~ $root_home_pattern ]] ||
    [[ "$line" =~ $windows_home_pattern ]] ||
    [[ "$line" =~ $file_uri_unix_home_pattern ]] ||
    [[ "$line" =~ $file_uri_root_home_pattern ]] ||
    [[ "$line" =~ $file_uri_windows_home_pattern ]] ||
    [[ "$line" =~ $github_token_pattern ]] ||
    [[ "$line" =~ $fine_grained_token_pattern ]] ||
    [[ "$line" =~ $aws_key_pattern ]] ||
    [[ "$line" =~ $openai_key_pattern ]]
}

added_content_has_sensitive_data() {
  local base=$1
  local head=$2
  local line

  while IFS= read -r line || [[ -n "$line" ]]; do
    if is_sensitive_line "$line"; then
      return 0
    fi
  done < <(git diff --no-ext-diff --unified=0 "$base" "$head" -- | extract_added_lines)
  return 1
}

main() {
  if (( $# != 2 )); then
    fail "usage: $0 BASE HEAD"
  fi

  local base=$1
  local head=$2
  local merge_base
  merge_base=$(resolve_merge_base "$base" "$head")

  local implementation_lines=0
  local test_lines=0
  local fixture_lines=0
  local generated_lines=0
  local documentation_lines=0
  local pure_renames=0
  local record added remainder deleted changed_path renamed_from base_path is_rename
  local changed_lines base_mode head_mode category

  while IFS= read -r -d '' record; do
    added=${record%%$'\t'*}
    remainder=${record#*$'\t'}
    deleted=${remainder%%$'\t'*}
    changed_path=${remainder#*$'\t'}
    base_path=$changed_path
    is_rename=0

    if [[ -z "$changed_path" ]]; then
      IFS= read -r -d '' renamed_from || fail 'invalid rename record'
      IFS= read -r -d '' changed_path || fail 'invalid rename record'
      base_path=$renamed_from
      is_rename=1
    fi

    if [[ "$added" == - || "$deleted" == - ]]; then
      fail 'binary changes are not allowed'
    fi

    changed_lines=$((added + deleted))
    base_mode=$(object_mode_at "$merge_base" "$base_path")
    head_mode=$(object_mode_at "$head" "$changed_path")
    category=$(classify_change "$changed_path" "$base_mode" "$head_mode")

    if (( is_rename )) && [[ "$added" == 0 && "$deleted" == 0 && "$base_mode" == "$head_mode" ]]; then
      pure_renames=$((pure_renames + 1))
    fi

    case "$category" in
      implementation)
        implementation_lines=$((implementation_lines + changed_lines))
        ;;
      test)
        test_lines=$((test_lines + changed_lines))
        ;;
      fixture)
        fixture_lines=$((fixture_lines + changed_lines))
        ;;
      generated)
        generated_lines=$((generated_lines + changed_lines))
        implementation_lines=$((implementation_lines + changed_lines))
        ;;
      documentation)
        documentation_lines=$((documentation_lines + changed_lines))
        ;;
      *)
        fail "unknown line category: $category"
        ;;
    esac
  done < <(git diff --numstat -z --find-renames=100% "$merge_base" "$head" --)

  local handwritten_total=$((implementation_lines + test_lines + fixture_lines))
  printf '%s\n' \
    "Handwritten implementation: $implementation_lines" \
    "Tests: $test_lines" \
    "Handwritten fixtures: $fixture_lines" \
    "Generated files counted as implementation: $generated_lines" \
    "Documentation: $documentation_lines" \
    "Pure renames: $pure_renames files" \
    "Handwritten total: $handwritten_total"

  if (( implementation_lines > 500 )); then
    fail 'handwritten implementation exceeds 500 changed lines'
  fi
  if (( handwritten_total > 1000 )); then
    printf 'diff-policy: WARNING: handwritten total exceeds 1000 changed lines; review the PR split\n' >&2
  fi

  local path message
  while IFS= read -r -d '' path; do
    if message=$(forbidden_path_message "$path"); then
      fail "$message"
    fi
  done < <(git diff --name-only -z --no-renames --diff-filter=ACMRTUXB "$merge_base" "$head" --)

  if added_content_has_sensitive_data "$merge_base" "$head"; then
    fail 'added content contains a credential or machine-specific path'
  fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  set -euo pipefail
  main "$@"
fi
