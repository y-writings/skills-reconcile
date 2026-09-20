---
name: skills-reconcile-roadmap-pr
description: Advance exactly one currently approved skills-reconcile roadmap item through investigation, implementation, verification, and a pull request. Use only when explicitly invoked in this repository. Re-read the current repository instructions and plan on every invocation; never treat this skill as product authority or reuse obsolete roadmap assumptions.
---

<!-- markdownlint-disable MD013 -->

# Skills Reconcile Roadmap PR

## Role

Use this skill as a workflow for completing one current roadmap item. It controls how to inspect,
implement, verify, and publish a change; it does not define what `skills-reconcile` should do.

The repository's current `AGENTS.md` and `docs/plan/**` are the only product and process authority.
Read them again on every invocation. Do not carry command names, roadmap IDs, domain models, safety
rules, dependencies, or discarded assumptions from an earlier session or from this skill.

If this skill and the current repository documents ever disagree, follow the repository documents and
report the skill drift before continuing.

## Invocation boundary

- Run this workflow only after the user explicitly invokes `$skills-reconcile-roadmap-pr`.
- Never infer invocation from a general request to discuss, review, or edit the repository.
- Work on exactly one item that exists in the current `docs/plan/pr-roadmap.md`.
- If the user names an obsolete or missing item, do not map it to a new item by guesswork. Report that
  it is no longer in the current roadmap and ask which current item to use.

Choose the endpoint from the user's request before taking action:

| Endpoint              | Allowed result                                                               |
| --------------------- | ---------------------------------------------------------------------------- |
| Named verification    | Run only the named check on the current checkout                             |
| Investigation         | Report the working contract and any blocking decision without changing files |
| Verified local change | Implement and verify, then stop before commit, push, and PR creation         |
| Pull request          | Implement, verify, commit, push, and create a non-draft PR; never merge it   |

When the user explicitly invokes the skill without choosing an earlier endpoint, the pull request is
the default. A narrower instruction always wins.

## Reload the current source of truth

Before selecting or implementing an item, read:

1. the repository-root `AGENTS.md`;
2. `docs/plan/README.md`;
3. `docs/plan/pr-roadmap.md`;
4. `docs/plan/cli-contract.md`;
5. `docs/plan/safety-and-verification.md`;
6. `docs/plan/scope-and-compatibility.md`;
7. item-specific documents referenced by those files; and
8. the pull request template, only when the selected endpoint can publish a PR.

Use the current files, not remembered content or example IDs in evals. Inspect implementation,
tests, dependencies, and Git history only to understand the present repository state. They cannot
override an approved product contract.

## Handle limited endpoints first

For a named verification, preserve the checkout and Git refs, run only the requested check, report the
result, and stop. Do not select a roadmap item, refresh the base, create a branch, or load unrelated
implementation context.

For investigation, keep all operations read-only. Do not fetch or update Git refs. Read-only remote or
GitHub queries are acceptable when freshness is necessary, but state that the local base was not
refreshed. Report the working contract and stop before implementation.

## Run preflight for implementation

1. Confirm that the working directory is the intended `skills-reconcile` repository.
2. Inspect the current branch, worktree status, remotes, recent history, and current roadmap.
3. Preserve unrelated user changes. Never stash, reset, discard, or absorb them. Stop if they prevent
   an isolated change.
4. Identify exactly one current roadmap item. If no single item follows unambiguously, stop and show
   the competing interpretations.
5. Confirm that every prerequisite named by the current roadmap is satisfied.
6. For a regular PR or the root of an explicitly requested stack, start from the latest successful
   `main`. Use a stack only when the user explicitly requests one. A later stack layer must use the
   preceding layer as its direct base.
7. Before implementing a user-facing interface, confirm that the current `cli-contract.md` contains
   the complete, approved contract required by `AGENTS.md`. A roadmap label is not a CLI contract.

Do not translate an old roadmap ID into a current one or assume that an unfinished contract item is
approval to choose product behavior.

## Establish one working contract

Before editing, provide a concise commentary update containing:

| Field          | Required content                                                     |
| -------------- | -------------------------------------------------------------------- |
| Item           | Exact current roadmap ID and responsibility                          |
| Scope          | One approved behavior or one bounded foundation change               |
| Out of scope   | Deferred behavior and unresolved future phases                       |
| Acceptance     | Observable success and explicit failures                             |
| Expected files | Source, tests, fixtures, configuration, and docs                     |
| Estimated size | Implementation, tests/fixtures, generated files, and docs separately |
| Verification   | Checks required by the current safety document and affected tooling  |

This update records the interpretation; it does not create a missing product contract.

## Stop for a material decision

Stop before implementation or publication when any of these is true:

- the selected item or its prerequisite is missing, obsolete, ambiguous, or incomplete;
- the current plan, CLI contract, repository instructions, implementation, or required dependency
  boundary disagree about user-visible behavior;
- completing the item requires selecting an undecided command, output, filesystem, TUI, copy, or
  compatibility policy;
- more than one roadmap responsibility is required for a buildable or reviewable change;
- the current line-count or change-size gates would be exceeded;
- real user data, installed Skills, credentials, or an unapproved write path would be required;
- unrelated worktree changes prevent an isolated diff;
- a required base, prerequisite PR, CI result, or exact contract approval cannot be established;
- public branch, commit, or PR content would expose private or machine-specific data; or
- the user's requested endpoint or authority does not permit the next action.

Report concrete evidence, the available interpretations, their effects, and one recommendation. Do
not invent missing decisions to keep the workflow moving.

Routine in-scope formatting, compilation, lint, or test failures are not product decisions. Fix the
mistake, rerun affected checks, and continue within the established contract.

## Implement the selected item

- Implement only the current item's approved responsibility.
- Keep future roadmap phases out of the design until their contracts are approved.
- For user-facing interfaces, implement only the exact grammar, outputs, statuses, and side effects in
  the current CLI contract.
- Add the behavior's tests and synthetic fixtures in the same change.
- Use only test-owned temporary directories and synthetic data where the current safety document
  requires isolation.
- Reject unsupported inputs explicitly when required by the current contract.
- Prefer the smallest reader-oriented design; do not add speculative abstractions or compatibility
  layers for discarded behavior.
- Keep every review boundary buildable and testable.
- Update roadmap completion records only when the current plan assigns that update to this item.

## Verify against the current plan

Derive the exact checks from the current item, `AGENTS.md`, and
`docs/plan/safety-and-verification.md`; do not rely on a fixed historical checklist embedded here.
At minimum when applicable:

1. run targeted tests during implementation and relevant full tests afterward;
2. run formatting, static analysis, documentation lint, and `git diff --check`;
3. run the currently required Go, Nix, container, or package smoke paths;
4. compare the final diff with the correct base and calculate the current line-count categories;
5. verify that tests use synthetic, isolated inputs and that read-only behavior performs no writes;
6. inspect the final status and diff for scope creep, accidental files, secrets, user paths, and
   stale terminology; and
7. record any planned check that is not yet available instead of inventing a substitute.

Do not run against real user state merely to strengthen a PR check. A manual real-state check is
allowed only when the current safety contract permits it and the user explicitly requests it.

For a verified local change, stop after successful verification unless the user separately requested
a local commit. Report the branch, worktree state, diff, checks, and the next unpublished action.

## Publish only at the PR endpoint

When all current gates pass and no material decision remains:

1. create a focused semantic commit;
2. push with the repository's approved regular or explicitly requested stacked workflow;
3. fill the current PR template with scope, exclusions, verification, actual line counts, and
   carry-over;
4. remove placeholders and private or machine-specific details from public text;
5. create a non-draft PR; and
6. verify the PR's base, head, URL, body, and draft state.

Never merge, enable auto-merge, or publish a release. If CI or review reveals a material product
decision, leave the PR open and report the evidence instead of broadening the change.

## Report the result

For every endpoint, state the selected current roadmap item when one was required, what changed or
remained unchanged, checks run, Git and PR state, and deliberately deferred scope. On a stop path,
state the exact decision needed to continue.
