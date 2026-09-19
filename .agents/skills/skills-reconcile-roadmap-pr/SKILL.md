---
name: skills-reconcile-roadmap-pr
description: Advance exactly one skills-reconcile roadmap item from preflight through a verified pull request. Use only when explicitly invoked in this repository; stop for unresolved scope, specification, safety, or verification decisions.
---

<!-- markdownlint-disable MD013 -->

# Skills Reconcile Roadmap PR

## Outcome

Start work on exactly one item in `docs/plan/pr-roadmap.md`. For a regular pull request or the root of
an explicitly requested stack, start from the latest successful `main`; for a later stack layer, start
from the preceding item's branch in that same stack.
On the green path, establish the destination contract, implement and verify the item, commit and
push the branch, and create a pull request with the approved branch workflow. Never merge the pull request.

## Interpret the invocation

- Run this workflow only after explicit invocation. `agents/openai.yaml` disables implicit use.
- Treat an item ID named by the user as the target. Otherwise determine the single next item from
  roadmap order, repository state, and merged pull requests, including open preceding layers when the
  user explicitly requested a stack.
- Choose the requested endpoint before entering the roadmap workflow:
  - a named verification runs only that check on the current checkout;
  - investigation stops after establishing and reporting the working contract;
  - a verified local change stops after implementation and verification; and
  - a pull request is the default when the user does not set an earlier endpoint.
- Do not proceed beyond the chosen endpoint. Explicit invocation authorizes the complete green path
  only when the user has not limited it, and routine implementation choices already follow from the
  established contract.
- Never merge, enable auto-merge, publish a release, or run the migrated CLI against real data.

For a named verification, preserve the current checkout and Git refs, run only the named check,
report its result, and stop without entering the roadmap workflow below.

## Load the source of truth

Before continuing with the roadmap workflow, read:

1. the repository-root `AGENTS.md`;
2. `docs/plan/README.md`;
3. `docs/plan/pr-roadmap.md`;
4. `docs/plan/cli-contract.md`;
5. `docs/plan/safety-and-verification.md`;
6. `docs/plan/scope-and-compatibility.md`;
7. the repository pull request template; and
8. any item-specific documentation or instructions discovered from those files.

Treat the approved plan and the pinned dependency's public boundary as `AGENTS.md` defines them.
Inspect the migration inventory read-only to locate candidate capabilities and dependencies. Never use
its implementation, tests, or output to justify a contract, and never copy Skill bodies, real manifests,
machine state, credentials, or user-specific configuration into the destination.

## Run preflight

1. Confirm that the working directory belongs to `skills-reconcile` and inspect the current branch,
   worktree status, remotes, and recent history.
2. Preserve unrelated user changes. Do not stash, reset, discard, or incorporate them. Stop if they
   prevent an isolated change.
3. When implementing an item, fetch the remote. For a regular pull request or stack root, confirm the
   base equals the latest successful `main`. For a later layer in an explicitly requested stack,
   confirm the stack root has that base and the immediate base is the preceding item's branch.
4. In a regular workflow, confirm the preceding roadmap item is merged and there is no open feature
   pull request on which this item would depend. In an explicitly requested stack, confirm each open
   dependency is an earlier layer of that same stack and follows roadmap order. Before implementing a
   user-facing command against an open contract layer, also confirm a maintainer explicitly approved
   that contract PR's exact current head and record the commit and approval evidence in the dependent
   PR body. Review completion, resolved threads, and stack position are not approval; a new contract
   commit requires approval again.
5. When implementing or investigating an item, identify exactly one roadmap item. Stop if repository
   history and the roadmap do not identify one unambiguous next item.
6. When implementing an item, create a new branch from that base. Use a stack only when the user has
   explicitly requested one, and keep every layer buildable and reviewable on its immediate base.

For an investigation endpoint, do not fetch or update Git refs; use read-only remote or GitHub
API queries when freshness is needed, and report that the local base was not refreshed.

## Establish the contract before implementation

Inspect the approved product contract, the current destination code, and the public boundary of any
dependency involved. Use the migration inventory only to locate candidate scope. Then state a compact
working contract in a commentary update with:

| Field          | Required content                                                     |
| -------------- | -------------------------------------------------------------------- |
| Scope          | One user-visible behavior or one roadmap responsibility              |
| Out of scope   | Deferred flags, commands, fixes, refactors, and dependencies         |
| Acceptance     | Observable success and explicit failure conditions                   |
| Expected files | Source, tests, fixtures, configuration, and documentation            |
| Estimated size | Implementation, tests/fixtures, generated files, and docs separately |
| Verification   | Targeted, full, Nix, container, policy, and manual checks that apply |

Continue without waiting when the contract is supported consistently, fits the roadmap item, and
meets all gates below. For a user-facing command in an open stack, this includes exact-head contract
approval from a maintainer. The commentary update is an audit trail, not approval evidence.

## Stop for a material decision

Stop before commit, push, or pull request creation when any of these conditions appears:

- approved product contracts, current destination behavior, or a dependency's public boundary
  disagree about the same input;
- behavior appears defective or requires a bug fix, specification change, general refactor, or
  unrelated dependency update;
- more than one roadmap item is needed for a buildable or testable change;
- non-test implementation is estimated or actually measured above 500 changed lines;
- total hand-written implementation, tests, and fixtures exceeds 1,000 changed lines without a
  reviewed split decision;
- real HOME, XDG state, installed Skills, manifests, credentials, Docker socket, or other real user
  or workspace data would be required;
- destructive ownership, deletion targets, or required test isolation cannot be proven;
- in a regular workflow the previous feature pull request is unmerged, or in an explicitly requested
  stack the root or immediate-base relationship fails the preflight rules;
- an open contract layer required by a user-facing command lacks explicit maintainer approval for its
  current head, or the dependent PR body lacks the commit and approval evidence;
- `main` is failing or a required verification path remains unavailable or unresolved;
- the outgoing branch, commit, or pull request would expose private or machine-specific data; or
- a choice would materially alter the approved scope or user-visible contract.

Provide the smallest reproduction or exact evidence, the competing interpretations, their effects,
whether they can be separated within the line gate, and a recommended option. Do not assume the
answer.

Routine formatting, compilation, lint, or test failures caused by an in-scope mistake are not
material decisions. Fix them, rerun the affected checks, and continue. Retry a transient command
failure when doing so cannot change the contract or touch real data.

At an investigation endpoint, report the working contract, findings, and any decision needed, then
stop before implementation. Do not create a branch or change files, commits, or pull requests.

## Implement one item

- Implement the approved behavior in the destination's responsibility boundary. Preserve the
  destination CLI/module names and the dependency order in the plan.
- For a user-facing command, implement only grammar, flags, status semantics, output, and side effects
  already approved in `docs/plan/cli-contract.md`. A roadmap item name is not an interface contract.
- Add synthetic fixtures and failure-path tests with the behavior. Keep unsupported CLI commands,
  positional arguments, and flags explicit rather than silently accepting them.
- Make the smallest reader-oriented change that satisfies the established contract. Do not add
  speculative abstractions or compatibility aliases.
- Keep the migration inventory read-only and the destination buildable at every review boundary.
- Do not update roadmap completion records before the feature pull request is merged unless the plan
  explicitly assigns that documentation to the selected item.

## Verify before publication

Run the checks required by the selected phase and the repository's current tooling. At minimum:

1. run targeted tests while implementing, then the applicable full Go, Nix, and container checks;
2. run write-capable integration scenarios only inside the isolated container with synthetic HOME,
   XDG directories, fixtures, and fake external processes;
3. run formatting, static analysis, repository security/policy checks, and `git diff --check`;
4. compare the final diff to the merge base and calculate the plan's separate line-count categories;
5. confirm no prohibited paths, Skill bodies, real state, credentials, absolute user paths, or build
   artifacts are tracked; and
6. inspect the final status and diff for scope, test coverage, accidental files, and public-output
   privacy.

Never substitute a host-side write test for the required container boundary. If a phase does not yet
provide one of the planned check paths, record that fact accurately rather than inventing a command.

At a verified local change endpoint, honor an explicitly requested local commit only after the
checks pass; otherwise leave the verified diff in the worktree. Report the branch, worktree and
commit state, diff summary, checks run, and the next action that would require authorization, then
stop before publication.

## Publish the green path

Only the pull request endpoint enters this sequence. Proceed when every applicable local
check passes and no material decision remains:

1. create a focused semantic commit;
2. push the branch with the approved regular or stacked workflow;
3. fill the repository pull request template, including the contract, deliberate exclusions,
   verification results, actual line-count categories, and any carry-over;
4. remove placeholders and private or machine-specific details from all public text;
5. create a pull request without a draft flag; and
6. query the created pull request and confirm its base, head, URL, body, and `isDraft: false`.

If CI or review later exposes a routine in-scope defect, fix it on the same branch and reverify. If it
exposes a material decision, leave the pull request open, do not merge it, and present the
same evidence and options required by the stop gate.

## Report the result

Use the endpoint-specific reports above for named verification, investigation, and verified local
changes. On published success, report the pull request URL, branch and commit, selected roadmap item,
changed files, actual line-count categories, checks run, and deliberately deferred scope. On a
material stop path, report what remains unchanged or unpublished and the exact user decision needed
to continue.
