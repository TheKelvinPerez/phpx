# Elefante Phase 1 Implementation Handoff

Last updated: July 17, 2026

## Purpose

Resume Elefante from its completed planning state, land the canonical documentation safely, then begin only Phase 1, Go Module And Test Harness, through a strict red, green, refactor loop.

Do not recreate the product strategy or technical design. The repository documents are the source of truth.

## Required Reading

Read these files completely before making implementation or architecture decisions:

1. `AGENTS.md`
2. `START_HERE.md`
3. `specs/elefante-project-toolchain.md`
4. `ELEFANTE_BRAND_SETUP.md`
5. `CONTEXT.md`
6. `specs/phase-1-cli-technical-design.md`
7. `specs/phase-1-cli-IMPLEMENTATION_PLAN.md`
8. `docs/adr/0001-thin-cobra-adapter.md`
9. `docs/adr/0002-composer-compatibility-boundary.md`
10. `docs/adr/0003-machine-protocol-and-plan-digests.md`

Treat the product toolchain specification as the canonical product direction. Treat the Phase 1 CLI technical design as the canonical implementation contract. Treat the implementation plan as the required phase order and verification contract.

## Current Verified State

1. The local repository path is `/Users/0xaquawolf/Projects/elefante`.
2. The external GitHub remote still uses the retired temporary repository identity.
3. The Elefante documentation exists on the current local documentation branch and has not been merged or pushed to `main`.
4. The design commit immediately before this handoff is `a88545e41a82619df349a47c7e780d3eff6866b1` with subject `docs: define Phase 1 CLI implementation`.
5. No Go module or application code exists yet.
6. The technical design covers all five Phase 1 commands.
7. The implementation plan contains eighteen incremental phases.
8. Remaining brand reservations are recorded but do not block local CLI work.
9. No repository transfer, package publication, container publication, or deployment has been approved.

Do not trust this state without verifying it through Git at the start of the session.

## First Response Before Editing

Inspect and report:

1. Current branch.
2. Current HEAD commit and subject.
3. Working tree status.
4. `origin/main` commit after fetching.
5. Commits and diff scope between `origin/main` and the current documentation branch.
6. Whether the documentation branch contains only the Elefante planning and brand work.
7. The exact Phase 1 task found in the implementation plan.
8. The first failing test that will prove the implementation path.

Do not begin runtime code while the canonical documentation remains absent from verified `origin/main`.

## Documentation Landing Gate

The current documentation cannot be bypassed by creating an implementation branch from the older `main`.

Perform this gate in order:

1. Fetch `origin/main`.
2. Verify local `main` has not diverged from `origin/main`.
3. Verify the documentation branch contains no unrelated runtime code or unrelated commits.
4. Request explicit approval to merge the documentation branch into `main` and push the documentation only result to `origin/main`.
5. Do not interpret approval to start coding as permission to push `main`.
6. After explicit approval, merge with plain `git merge` from `main`.
7. Push `main` only when that push is explicitly approved.
8. Fetch again and prove local `main` equals `origin/main` at the documentation commit.

If fetch fails, branches diverge, unrelated commits appear, or push approval is absent, stop before creating implementation work.

## Implementation Branch Gate

After the documentation is present on verified `origin/main`:

1. Check out local `main`.
2. Fetch `origin main`.
3. Prove local `main` equals `origin/main`.
4. Create a normal feature branch named `feat/phase-1-foundation` from that verified base.
5. Do not push the feature branch unless explicitly requested.
6. Keep the diff limited to Phase 1 of the implementation plan.

Elefante is not under `~/Projects/lightcodelabs/`, so the Lightcodelabs worktree workflow does not apply.

## Active Task

Implement only `Phase 1: Go Module And Test Harness` from `specs/phase-1-cli-IMPLEMENTATION_PLAN.md`.

The phase goal is to establish the buildable Go module, thin Cobra command adapter, dependency construction, and test first command harness.

Required outcomes are:

1. A failing compiled binary behavior test exists before application behavior.
2. The test covers `elefante --help` and `elefante version`.
3. `go.mod` uses module path `github.com/elefantephp/elefante`.
4. The reviewed Go toolchain is declared.
5. Cobra remains isolated inside the CLI adapter.
6. The initial packages are `cmd/elefante`, `internal/cli`, `internal/app`, `internal/model`, and `internal/version`.
7. Tests can construct and execute the complete command tree in process with injected input, output, error, and context.
8. Repository verification commands cover normal tests, race detection, static checks, and a compiled binary smoke.

Do not implement discovery, constraints, providers, Composer acquisition, synchronization, execution, tools, configuration, state, or machine event behavior during this phase.

## Test First Execution Loop

For each behavior:

1. Red, add one focused behavior test through the public command boundary.
2. Run the focused test and prove it fails for the expected missing behavior.
3. Green, add the smallest implementation that passes.
4. Run the focused test and prove it passes.
5. Refactor only while the focused test remains green.
6. Run the full phase verification before committing.

Do not write every test first. Complete one vertical behavior cycle at a time.

## Required Phase Verification

Run and report:

```console
go test ./...
go test -race ./...
go vet ./...
go build -o ./tmp/elefante-phase-1 ./cmd/elefante
./tmp/elefante-phase-1 --help
./tmp/elefante-phase-1 version
```

Remove the temporary compiled binary before committing unless the repository establishes an ignored build output directory during the phase.

## Phase Completion Gate

Phase 1 is complete only when:

1. The module builds on Darwin arm64.
2. Root command construction uses no global mutable state.
3. Tests can execute the complete command tree in process.
4. Core packages do not import Cobra.
5. Focused tests, full tests, race detection, static checks, and compiled binary smoke all pass.
6. The diff contains only Phase 1 work.
7. The implementation plan marks Phase 1 complete and Phase 2 next.
8. The phase receives one clear local commit.
9. Nothing is pushed, merged, published, or released without explicit approval.

## Stop Conditions

Stop and request direction when:

1. The documentation branch contains unrelated commits or runtime code.
2. Local `main` is ahead of, behind, or diverged from `origin/main` in a way that cannot be resolved by a verified fast forward.
3. The technical design and implementation plan conflict.
4. Phase 1 requires a new runtime dependency beyond Cobra.
5. A proposed change crosses into Phase 2 or a later implementation phase.
6. A test cannot be written through the public command boundary without changing the accepted architecture.
7. A command would rename or transfer the external repository, publish a package, publish a container, or push without explicit approval.

## Closeout Report

At phase completion, report:

1. Behaviors implemented.
2. Focused and complete verification results.
3. Important behavior intentionally deferred.
4. Files changed.
5. Commit hash and subject.
6. Working tree status.
7. Branch relation to `origin/main`.
8. The exact Phase 2 starting point.

Do not begin Phase 2 until Phase 1 is committed and the next phase is explicitly requested.
