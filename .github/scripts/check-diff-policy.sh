#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'diff-policy: %s\n' "$*" >&2
  exit 1
}

if (( $# != 2 )); then
  fail "usage: $0 BASE HEAD"
fi

base=$1
head=$2

git rev-parse --verify "${base}^{commit}" >/dev/null 2>&1 || fail "invalid base revision"
git rev-parse --verify "${head}^{commit}" >/dev/null 2>&1 || fail "invalid head revision"
merge_base=$(git merge-base "$base" "$head") || fail "base and head have no merge base"

implementation_lines=0
test_lines=0
fixture_lines=0
generated_lines=0
documentation_lines=0

while IFS= read -r -d '' record; do
  added=${record%%$'\t'*}
  remainder=${record#*$'\t'}
  deleted=${remainder%%$'\t'*}
  changed_path=${remainder#*$'\t'}
  if [[ -z "$changed_path" ]]; then
    IFS= read -r -d '' _renamed_from
    IFS= read -r -d '' changed_path
  fi
  if [[ "$added" == "-" || "$deleted" == "-" ]]; then
    fail "binary changes are not allowed"
  fi

  changed_lines=$((added + deleted))
  case "$changed_path" in
    testdata/* | */testdata/*)
      fixture_lines=$((fixture_lines + changed_lines))
      ;;
    *_test.go | *_test.sh | tests/* | */tests/*)
      test_lines=$((test_lines + changed_lines))
      ;;
    flake.lock | go.sum | tools/skills/package-lock.json)
      generated_lines=$((generated_lines + changed_lines))
      implementation_lines=$((implementation_lines + changed_lines))
      ;;
    docs/* | README.md | README.* | AGENTS.md | .agents/skills/*)
      documentation_lines=$((documentation_lines + changed_lines))
      ;;
    *)
      implementation_lines=$((implementation_lines + changed_lines))
      ;;
  esac
done < <(git diff --numstat -z --find-renames=100% "$merge_base" "$head" --)

pure_renames=$(
  git diff --name-status --find-renames=100% "$merge_base" "$head" -- |
    awk '$1 == "R100" { count++ } END { print count + 0 }'
)
handwritten_total=$((implementation_lines + test_lines + fixture_lines))

printf '%s\n' \
  "Handwritten implementation: $implementation_lines" \
  "Tests: $test_lines" \
  "Handwritten fixtures: $fixture_lines" \
  "Generated files counted as implementation: $generated_lines" \
  "Documentation: $documentation_lines" \
  "Pure renames: $pure_renames files" \
  "Handwritten total: $handwritten_total"

if (( implementation_lines > 500 )); then
  fail "handwritten implementation exceeds 500 changed lines"
fi
if (( handwritten_total > 1000 )); then
  printf 'diff-policy: WARNING: handwritten total exceeds 1000 changed lines; review the PR split\n' >&2
fi

while IFS= read -r -d '' changed_path; do
  case "$changed_path" in
    skills | skills/*)
      fail "root skills content is not allowed"
      ;;
    skills-manifest.json)
      fail "a root skills-manifest.json is not allowed"
      ;;
    state | state/*)
      fail "root machine state is not allowed"
      ;;
    node_modules | node_modules/* | */node_modules/*)
      fail "node_modules content is not allowed"
      ;;
    skills-reconcile)
      fail "the built skills-reconcile binary is not allowed"
      ;;
    .skill-lock.json | */.skill-lock.json | workspace-projections.json | */workspace-projections.json)
      case "$changed_path" in
        testdata/* | */testdata/*) ;;
        *) fail "machine state is not allowed outside synthetic testdata" ;;
      esac
      ;;
  esac
done < <(git diff --name-only -z --no-renames --diff-filter=ACMRTUXB "$merge_base" "$head" --)

sensitive_pattern='-----BEGIN ([A-Z0-9]+ )*PRIVATE KEY-----|https?://[^[:space:]/:@]+:[^[:space:]@]+@|/(Users|home)/[A-Za-z0-9._-]+|/root(/|[[:space:]]|$)|[A-Za-z]:\\Users\\[^\\[:space:]]+|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|AKIA[A-Z0-9]{16}|sk-[A-Za-z0-9]{20,}'
if git diff --no-ext-diff --unified=0 "$merge_base" "$head" -- |
  awk '/^\+\+\+ / { next } /^\+/ { sub(/^\+/, ""); print }' |
  LC_ALL=C grep -E -e "$sensitive_pattern" >/dev/null; then
  fail "added content contains a credential or machine-specific path"
fi
