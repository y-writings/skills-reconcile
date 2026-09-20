<!-- markdownlint-disable MD013 -->

# Repository instructions

## Source of truth

- Before product or implementation work, read `docs/plan/README.md`, `docs/plan/cli-contract.md`, and the plan document relevant to the task.
- Treat only behavior explicitly approved in `docs/plan/**` as product authority. Existing code, tests, previous roadmap IDs, and legacy documents do not define the new product.
- Do not implement user-facing behavior that has not been approved in `docs/plan/cli-contract.md`.
- The CLI and Go module remain `skills-reconcile` and `github.com/y-writings/skills-reconcile` unless a later approved contract changes them.

## Current product direction

- The product discovers Agent Skills in known local directories and will later let the user copy selected Skills into a repository they control.
- Do not classify Skills as self-authored or third-party for management eligibility. A Skill selected by the user is a candidate regardless of author or installation source.
- The user is responsible for licensing, use, modification, and redistribution decisions. Do not invent automated license or ownership gating.
- The first milestone is read-only listing of Skills under `$HOME/.agents/skills`.
- Do not add TUI filtering, additional scan roots, repository layout, copying, manifests, apply/prune behavior, or cross-machine deployment before their contracts are separately approved.
- Do not use the private implementation, lockfiles, or source parser of the `skills` package as a product contract.

## Safety boundaries

- Use only synthetic Skill fixtures under test-owned temporary directories for automated tests.
- Never make automated tests read or write the host HOME, real `.agents`, real agent directories, installed Skills, credentials, or user-specific state.
- Until the copy phase is approved, do not commit real Skill bodies or run write paths against real Skill directories.
- Treat symlinks, unreadable entries, missing roots, and paths outside the scan root as explicit contract cases rather than incidental filesystem behavior.
- Read-only commands must not create, modify, move, or delete files.

## Change and PR discipline

- Implement one approved, reviewable behavior and its tests at a time.
- Start each regular feature PR from the latest successful `main`. Create a stack only when explicitly requested, with each layer based on the preceding layer.
- Keep hand-written non-test implementation changes at or below 500 added-plus-deleted lines. Count workflows, scripts, and executable configuration as implementation.
- Keep implementation and its tests in the same PR. If hand-written implementation, tests, and fixtures exceed 1,000 added-plus-deleted lines, stop and review whether the change contains more than one responsibility.
- Do not preserve obsolete behavior through compatibility aliases or speculative abstractions.
- Follow `docs/plan/safety-and-verification.md` before requesting review or merge.
