---
name: skills-reconcile-roadmap-pr
description: Advance exactly one skills-reconcile migration roadmap item from preflight through a verified regular pull request. Use only when explicitly invoked in this repository; stop for unresolved scope, specification, safety, or verification decisions.
---

<!-- markdownlint-disable MD013 -->

# Skills Reconcile Roadmap PR

## Outcome

Advance exactly one item from `docs/plan/pr-roadmap.md` on top of the latest successful `main`.
On the green path, investigate the source and destination, implement and verify the item, commit and
push the branch, and create a regular pull request. Never merge the pull request.

## Interpret the invocation

- Run this workflow only after explicit invocation. `agents/openai.yaml` disables implicit use.
- Treat an item ID named by the user as the target. Otherwise determine the single next item from
  roadmap order, repository state, and merged pull requests.
- Unless the user limits the task, explicit invocation authorizes the complete green path through a
  regular pull request. Do not pause for routine implementation choices that the existing contract
  already resolves.
- Honor narrower requests such as investigation only, implementation without publication, or a
  named verification step. For an investigation-only request, keep preflight read-only and do not
  create a branch.
- Never merge, enable auto-merge, publish a release, or run the migrated CLI against real data.

## Load the source of truth

Before changing files, read:

1. the repository-root `AGENTS.md`;
2. `docs/plan/README.md`;
3. `docs/plan/pr-roadmap.md`;
4. `docs/plan/safety-and-verification.md`;
5. `docs/plan/scope-and-compatibility.md`;
6. the repository pull request template; and
7. any item-specific documentation or instructions discovered from those files.

Treat the migration source and its pinned revision exactly as `AGENTS.md` defines them. Inspect that
source read-only. Never copy Skill bodies, real manifests, machine state, credentials, or
user-specific configuration into the destination.

## Run preflight

1. Confirm that the working directory belongs to `skills-reconcile` and inspect the current branch,
   worktree status, remotes, and recent history.
2. Preserve unrelated user changes. Do not stash, reset, discard, or incorporate them. Stop if they
   prevent an isolated change.
3. For a full workflow, fetch the remote and confirm the local base equals the latest successful
   `main`. For an investigation-only request, do not fetch or update Git refs; use read-only remote
   or GitHub API queries when freshness is needed, and report that the local base was not refreshed.
4. Confirm the preceding roadmap item is merged and there is no open feature pull request on which
   this item would depend.
5. Identify exactly one roadmap item. Stop if repository history and the roadmap do not identify one
   unambiguous next item.
6. Create a new, non-stacked branch from that base.

If `main` advances before publication, rebase onto the new latest successful `main` and rerun all
verification when the rebase is conflict-free. Stop and report evidence if it conflicts or changes
the selected item's contract.

## Establish the contract before implementation

Inspect the relevant source implementation, source tests, source documentation, and current
destination code. Then state a compact working contract in a commentary update with:

| Field          | Required content                                                     |
| -------------- | -------------------------------------------------------------------- |
| Scope          | One user-visible behavior or one roadmap responsibility              |
| Out of scope   | Deferred flags, commands, fixes, refactors, and dependencies         |
| Acceptance     | Observable success and explicit failure conditions                   |
| Expected files | Source, tests, fixtures, configuration, and documentation            |
| Estimated size | Implementation, tests/fixtures, generated files, and docs separately |
| Verification   | Targeted, full, Nix, container, policy, and manual checks that apply |

Continue without waiting when the contract is supported consistently, fits the roadmap item, and
meets all gates below. The commentary update is an audit trail, not a request for another approval.

## Stop for a material decision

Stop before commit, push, or pull request creation when any of these conditions appears:

- source code, tests, or documentation disagree about the same input;
- behavior appears defective or requires a bug fix, specification change, general refactor, or
  unrelated dependency update;
- more than one roadmap item is needed for a buildable or testable change;
- non-test implementation is estimated above 450 changed lines or actually exceeds 500;
- total hand-written implementation, tests, and fixtures exceeds 1,000 changed lines without a
  reviewed split decision;
- real HOME, XDG state, installed Skills, manifests, credentials, Docker socket, or other real user
  or workspace data would be required;
- destructive ownership, deletion targets, or required test isolation cannot be proven;
- the previous feature pull request is unmerged, `main` is failing, or a required verification path
  remains unavailable or unresolved;
- the outgoing branch, commit, or pull request would expose private or machine-specific data; or
- a choice would materially alter the approved scope or user-visible contract.

Provide the smallest reproduction or exact evidence, the competing interpretations, their effects,
whether they can be separated within the line gate, and a recommended option. Do not assume the
answer.

Routine formatting, compilation, lint, or test failures caused by an in-scope mistake are not
material decisions. Fix them, rerun the affected checks, and continue. Retry a transient command
failure when doing so cannot change the contract or touch real data.

## Implement one item

- Port behavior, not whole files. Preserve the destination CLI/module names and the dependency order
  in the plan.
- Add synthetic fixtures and failure-path tests with the behavior. Keep unsupported inputs and flags
  explicit rather than silently accepting them.
- Make the smallest reader-oriented change that satisfies the established contract. Do not add
  speculative abstractions or compatibility aliases.
- Keep the migration source read-only and the destination buildable at every review boundary.
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

## Publish the green path

Only after every applicable local check passes and no material decision remains:

1. create a focused semantic commit;
2. push the non-stacked branch;
3. fill the repository pull request template, including the contract, deliberate exclusions,
   verification results, actual line-count categories, and any carry-over;
4. remove placeholders and private or machine-specific details from all public text;
5. create a regular pull request without a draft flag; and
6. query the created pull request and confirm its base, head, URL, body, and `isDraft: false`.

If CI or review later exposes a routine in-scope defect, fix it on the same branch and reverify. If it
exposes a material decision, leave the regular pull request open, do not merge it, and present the
same evidence and options required by the stop gate.

## Report the result

On success, report the pull request URL, branch and commit, selected roadmap item, changed files,
actual line-count categories, checks run, and deliberately deferred scope. On a stop path, report
what remains unchanged or unpublished and the exact user decision needed to continue.
