<!-- markdownlint-disable MD013 -->

# Repository instructions

## Source of truth

- Before migration work, read `docs/plan/README.md` and the plan document relevant to the task.
- Treat `.worktrees/skills` at commit `3c15f60` as a read-only reference. Do not modify or delete it.
- The destination CLI and Go module are `skills-reconcile` and
  `github.com/y-writings/skills-reconcile`. Do not add a `skills-sync` compatibility alias.

## Migration scope

- Migrate only the Skill management tool. Never copy or commit Skill bodies, a real
  `skills-manifest.json`, machine state, credentials, or user-specific paths and configuration.
- The prohibition on committing Skill bodies applies to Skills from `.worktrees/skills` and real
  managed Skills. Repository-local Skills under `.agents/skills/**` may be committed when they only
  provide development or migration guidance for `skills-reconcile` and contain none of that
  prohibited source or user data.
- Move one reviewable behavior and its tests at a time. Unsupported inputs and flags must fail
  explicitly instead of being ignored.
- Do not mix bug fixes, specification changes, general refactoring, or unrelated dependency updates
  into a migration change without explicit direction. When the source behavior is ambiguous or
  appears defective, report the evidence before implementing that behavior.

## Safety boundaries

- Run write-capable integration tests only in the isolated container with synthetic fixtures and
  temporary HOME and XDG directories.
- Never mount or use the host HOME, agent directories, real XDG state, real manifests, installed
  Skills, or the Docker socket for those tests.
- Never run `apply`, `prune`, `adopt`, or another write path against real user or workspace data
  during migration. Preserve the read-before-write order defined by the plan.
- Nix is the installation and packaging path; the container is the side-effect isolation boundary.
  Validate both where the current migration phase makes them available.

## Change and PR discipline

- Start each feature PR from the latest successful `main`. Do not create stacked or otherwise
  dependent feature PRs unless explicitly requested.
- Keep each PR buildable and testable. Include the tests and hand-written fixtures for a behavior in
  the same PR as its implementation.
- Keep hand-written non-test implementation changes at or below 500 added-plus-deleted lines.
  Count workflows, scripts, and executable configuration as implementation. Count tests, fixtures,
  generated files, and documentation separately as specified in the plan.
- If total hand-written changes exceed 1,000 added-plus-deleted lines, stop and review whether the
  change contains more than one behavior; do not split implementation from its tests merely to meet
  a line limit.
- Follow the verification and stop conditions in `docs/plan/safety-and-verification.md` before
  requesting review or merge.
