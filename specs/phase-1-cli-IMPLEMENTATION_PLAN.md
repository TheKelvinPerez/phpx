# Elefante Phase 1 CLI Implementation Plan

## Plan Status

Updated: July 17, 2026

Source specification: `specs/phase-1-cli-technical-design.md`

Execution model: Test first, one behavior per red, green, refactor cycle

Current phase: Phase 8, Native Provider Doctor And Plan, complete

Next phase: Phase 9, DDEV Provider Doctor And Plan

Every phase must finish with a green focused suite, green repository suite, a clean diff review, and one intentional commit before the next phase begins.

## Collaboration Rules

### Single Contributor Workflow

One contributor completes the phase in task order. The contributor writes the failing behavior test, proves the failure, implements the smallest passing behavior, refactors while green, runs the complete phase verification, and commits the bounded result.

### Parallel Contributor Workflow

Parallel work is allowed only when file ownership and interfaces are already stable.

1. One contributor owns shared model or interface changes.
2. Other contributors may own independent fixtures, provider implementations, or golden cases after the shared contract lands.
3. Two contributors must not edit the same package contract, command tree, fixture generator, or protocol schema simultaneously.
4. Foundation and shared model changes land before provider or framework implementations.
5. The complete focused and repository verification runs after integrating parallel work.

## Phase 1: Go Module And Test Harness

Goal: Establish the buildable module, thin command adapter, dependency construction, and test first command harness.

Dependencies: None

### Tasks

1. Add a failing compiled binary test for `elefante --help` and `elefante version` behavior.
2. Initialize `go.mod` with module path `github.com/elefantephp/elefante`, the reviewed Go toolchain, and Cobra.
3. Implement `cmd/elefante`, `internal/cli`, `internal/app`, `internal/model`, and `internal/version` skeletons without domain behavior.
4. Add in process command test helpers with injected input, output, error, and context.
5. Add repository commands for unit tests, race detection, and static checks.

### TEST

1. Run `go test ./...` and confirm help and version behavior passes.
2. Run `go test -race ./...` and confirm no race reports.
3. Run `go vet ./...` and confirm no findings.
4. Build `./cmd/elefante` and execute `--help` from the compiled binary.

Completion Criteria:

* [x] The module builds on Darwin arm64.
* [x] The root command is constructed without global mutable state.
* [x] Tests can execute the full command tree in process.
* [x] Core packages do not import Cobra.

## Phase 2: Machine Event Protocol And Error Contract

Goal: Establish the stable machine protocol, human output boundary, typed errors, and exit mapping before domain commands emit results.

Dependencies: Phase 1 completed

### Tasks

1. Add failing golden tests for `started`, `result`, `error`, and `completed` event sequences.
2. Implement the versioned newline delimited JSON envelope with deterministic ordering and sequence numbers.
3. Implement typed diagnostics, stable Elefante error categories, and process exit mapping.
4. Add human renderer interfaces and minimal help compatible output without domain formatting.
5. Add tests proving `--json` owns standard output and malformed partial JSON is never emitted.

### TEST

1. Run `go test ./internal/output ./internal/cli ./internal/model`.
2. Run compiled binary tests for successful and failing commands in human and JSON modes.
3. Run `go test -race ./...`.

Completion Criteria:

* [x] Every machine output line is independently valid JSON.
* [x] Equivalent inputs produce identical canonical events.
* [x] Error events and process exits share one typed source.
* [x] Human output remains separate from protocol encoding.

## Phase 3: Project And Workspace Discovery

Goal: Discover repository, Composer root, application root, and workspace identity without executing project code.

Dependencies: Phase 2 completed

### Tasks

1. Add failing temporary filesystem tests for nearest Composer root selection, Git roots, worktrees, nongit projects, and ambiguous monorepos.
2. Implement safe path normalization, ancestor search, Git metadata inspection, and identity hashing.
3. Implement bounded JSON file reading with malformed input, duplicate key, type, size, and symlink boundary failures.
4. Add project input fingerprints and stable source references.
5. Add compiled `doctor --json` tests that stop after discovery with deterministic facts.

### TEST

1. Run `go test ./internal/discovery ./internal/paths`.
2. Run discovery fuzz tests.
3. Run compiled binary tests from repository roots, descendants, worktrees, and ambiguous fixtures.
4. Run `go test -race ./...`.

Completion Criteria:

* [x] Discovery never loads PHP or application bootstrap files.
* [x] Ambiguous roots produce candidate paths and a stable error code.
* [x] Worktrees receive distinct workspace identities.
* [x] Nongit Composer projects remain supported.

## Phase 4: Composer Metadata Parsing

Goal: Parse Composer project and lock metadata into immutable facts with precise source attribution.

Dependencies: Phase 3 completed

### Tasks

1. Add failing fixtures for root requirements, development requirements, conflicts, platform configuration, plugins, scripts, and locked package platform requirements.
2. Implement typed `composer.json` parsing without project execution.
3. Implement typed `composer.lock` parsing and content identity validation.
4. Normalize platform package names and preserve original constraint strings.
5. Add diagnostics for missing, stale, malformed, and internally inconsistent lock data.

### TEST

1. Run `go test ./internal/composer ./internal/discovery`.
2. Run parser fuzz tests against arbitrary JSON input.
3. Run fixture golden tests for source attribution and redacted diagnostics.
4. Run `go test -race ./...`.

Completion Criteria:

* [x] Composer facts include root and locked platform requirements.
* [x] Platform emulation is represented separately from actual runtime facts.
* [x] Plugins and scripts are discoverable before execution.
* [x] Parser failures never panic or leak sensitive content.

## Phase 5: Composer Platform Constraint Engine

Goal: Evaluate Composer compatible platform constraints without implementing dependency solving.

Dependencies: Phase 4 completed

### Tasks

1. Add failing tests for exact, comparison, AND, OR, wildcard, hyphen, tilde, caret, and stability cases.
2. Implement version normalization, constraint parsing, matching, and intersection behavior for platform packages.
3. Build an oracle harness that compares a broad corpus with official `composer/semver` behavior during conformance testing.
4. Add invalid and unsupported syntax blockers with original source references.
5. Add parser and matcher fuzz tests plus regression cases for every discovered mismatch.

### TEST

1. Run `go test ./internal/constraints`.
2. Run `go test ./internal/constraints -run TestComposerConformance`.
3. Run `go test -fuzz=Fuzz -fuzztime=30s ./internal/constraints`.
4. Run `go test -race ./...`.

Completion Criteria:

* [x] Supported constraint cases match the official oracle corpus.
* [x] Unsupported syntax blocks instead of approximating.
* [x] The engine evaluates PHP and all declared platform package kinds.
* [x] No package dependency solver behavior exists in Go.

## Phase 6: Framework Detection And Configuration

Goal: Detect supported project types and apply optional versioned Elefante policy without booting applications.

Dependencies: Phase 5 completed

### Tasks

1. Add failing fixtures for Laravel applications, Laravel packages, generic Composer projects, Bedrock WordPress, Symfony, and conflicting framework evidence.
2. Implement framework adapters that add facts and confidence evidence only.
3. Add failing tests for the version 1 `elefante.toml` schema, precedence, unknown fields, path escapes, task arrays, and prohibited secret values.
4. Implement TOML parsing, validation, and policy normalization.
5. Integrate `.php-version`, provider markers, and configuration fingerprints into discovery facts.

### TEST

1. Run `go test ./internal/frameworks ./internal/config`.
2. Run `doctor --json` against every framework fixture.
3. Run configuration fuzz and path boundary tests.
4. Run `go test -race ./...`.

Completion Criteria:

* [x] Framework detection never executes PHP.
* [x] Generic Composer remains the fallback adapter.
* [x] Configuration precedence is deterministic.
* [x] Task commands remain argument arrays without shell interpretation.

## Phase 7: Resolver, Plan Model, And Digest

Goal: Convert facts and policy into deterministic requirement resolutions, actions, diagnostics, and a content addressed plan.

Dependencies: Phase 6 completed

### Tasks

1. Add failing plan golden tests for compatible, incompatible, ambiguous, legacy, optional extension, offline, and frozen cases.
2. Implement requirement source precedence and conflict resolution.
3. Implement typed plan actions, effect classes, dependencies, trust requirements, and stable ordering.
4. Implement canonical digest input and SHA256 plan digest generation.
5. Add mutation free plan tests and digest sensitivity tests for every relevant input category.

### TEST

1. Run `go test ./internal/plan ./internal/model`.
2. Run plan canonicalization fuzz tests.
3. Run golden tests twice and compare byte identical JSON output.
4. Run `go test -race ./...`.

Completion Criteria:

* [x] Plan construction performs no I/O or mutation.
* [x] Equivalent facts and policy produce the same plan and digest.
* [x] Relevant input changes alter the digest.
* [x] Display wording does not alter the digest.

## Phase 8: Native Provider Doctor And Plan

Goal: Inspect and plan against host PHP and Composer executables without mutating system selection.

Dependencies: Phase 7 completed

### Tasks

1. Add the shared provider conformance harness with a deterministic fake provider.
2. Add failing native provider tests for missing PHP, compatible PHP, incompatible PHP, extensions, Composer, paths, and process errors.
3. Implement native executable discovery and safe PHP runtime inspection.
4. Implement native provider capabilities, observations, fingerprints, and execution specifications.
5. Complete `doctor` and `plan` command behavior for native fixtures in human and JSON modes.

### TEST

1. Run `go test ./internal/providers/... -run TestProviderConformance`.
2. Run `go test ./internal/providers/native ./internal/app ./internal/cli`.
3. Run compiled `doctor` and `plan` tests against the local PHP and Composer installation.
4. Run `go test -race ./...`.

Completion Criteria:

* [x] Native inspection reports PHP, extensions, Composer, and provenance.
* [x] Native planning never globally relinks or installs PHP.
* [x] Doctor performs no network access.
* [x] Human and machine output explain every provider decision.

## Phase 9: DDEV Provider Doctor And Plan

Goal: Inspect and plan the first isolated execution topology through documented DDEV interfaces.

Dependencies: Phase 8 completed

### Tasks

1. Add failing DDEV adapter tests using recorded structured command fixtures and process substitutes.
2. Implement DDEV availability, version, engine, project configuration, runtime, extension, and Composer inspection.
3. Implement deterministic DDEV plan actions for existing configuration, missing configuration, and environment start.
4. Add DDEV provider conformance coverage.
5. Add real DDEV doctor and plan integration tests against an isolated fixture through OrbStack.

### TEST

1. Run `go test ./internal/providers/ddev`.
2. Run `go test ./internal/providers/... -run TestProviderConformance`.
3. Run the documented DDEV integration command for the fixture.
4. Confirm integration cleanup affects only the fixture environment.

Completion Criteria:

* [ ] DDEV passes provider conformance.
* [ ] Missing DDEV configuration is a reviewable project mutation.
* [ ] Frozen mode blocks configuration creation.
* [ ] Real DDEV observations match adapter results.

## Phase 10: Trust, Network, State, And Locks

Goal: Establish safe approval, redaction, offline behavior, local state, action journals, and mutation concurrency controls.

Dependencies: Phase 9 completed

### Tasks

1. Add failing tests for Composer trust fingerprints, invalidation, noninteractive approval, `--yes`, and exact plan approval.
2. Implement secret redaction across diagnostics, events, logs, plans, URLs, headers, and environment summaries.
3. Implement command specific network policy with offline enforcement and deterministic network substitutes.
4. Implement operating system paths, atomic state writes, action journals, environment locks, and stale lock recovery.
5. Add compiled tests proving plan mismatch and approval failures perform no mutation.

### TEST

1. Run `go test ./internal/security ./internal/state ./internal/cache ./internal/paths`.
2. Run redaction fuzz and synthetic secret tests.
3. Run concurrent lock and race tests.
4. Run compiled offline, approval, and plan mismatch tests.

Completion Criteria:

* [ ] Raw synthetic secrets appear nowhere in captured output or state.
* [ ] Nonterminal execution never prompts.
* [ ] Plan mismatch exits before mutation.
* [ ] Interrupted state writes preserve the previous valid state.

## Phase 11: Official Composer Acquisition And Sync Engine

Goal: Acquire verified official Composer and apply journaled synchronization plans through provider actions.

Dependencies: Phase 10 completed

### Tasks

1. Add failing tests for Composer selection precedence, cache hits, verified downloads, checksum failure, partial downloads, and offline misses.
2. Implement official Composer metadata resolution, download, verification, atomic cache promotion, and executable selection.
3. Add failing sync engine tests for ordered actions, journal checkpoints, failure reporting, retry, and safe compensation.
4. Implement the provider independent synchronization application service.
5. Implement Composer install and platform verification actions with trust policy and normal argument semantics.

### TEST

1. Run `go test ./internal/composer ./internal/cache ./internal/app`.
2. Run integration tests against an official Composer executable and a local fixture.
3. Simulate interrupted downloads and action failures.
4. Run `go test -race ./...`.

Completion Criteria:

* [ ] Unverified Composer artifacts never execute.
* [ ] Offline cache hits work and misses fail before mutation.
* [ ] Sync journals every completed action.
* [ ] Composer remains responsible for installation and final platform checks.

## Phase 12: Native Sync And Run

Goal: Synchronize and execute supported projects through the native provider with exact process semantics.

Dependencies: Phase 11 completed

### Tasks

1. Add failing command tests for native sync approval, Composer scripts, plugins, frozen mode, and platform verification.
2. Implement native provider apply actions and environment construction.
3. Add failing executor tests for argument safety, working directories, environment overlays, input, output, errors, cancellation, signals, and exit codes.
4. Implement `elefante run --` with raw human streams and encoded machine stream events.
5. Add compiled binary tests using commands with zero, nonzero, signaled, large output, and binary output behavior.

### TEST

1. Run `go test ./internal/executor ./internal/providers/native ./internal/app`.
2. Run compiled native `sync` and `run` tests against fixtures.
3. Run signal and cancellation tests on Darwin.
4. Run `go test -race ./...`.

Completion Criteria:

* [ ] Native sync completes a locked supported fixture.
* [ ] Run preserves every child exit code after start.
* [ ] Arguments never pass through implicit shell interpolation.
* [ ] Machine stream events reconstruct child output correctly.

## Phase 13: DDEV Sync And Run

Goal: Synchronize and execute the same supported project behavior through DDEV.

Dependencies: Phase 12 completed

### Tasks

1. Add failing DDEV apply and execution tests for start, Composer install, platform verification, command execution, and cancellation.
2. Implement DDEV provider apply actions and execution specifications.
3. Run the shared command contract through native and DDEV providers.
4. Add real DDEV synchronization and test command integration fixtures.
5. Add cleanup verification that removes only test fixture resources.

### TEST

1. Run `go test ./internal/providers/ddev ./internal/app`.
2. Run the complete provider conformance suite.
3. Run real DDEV `sync` and `run -- php artisan test` integration tests.
4. Compare native and DDEV machine result contracts.

Completion Criteria:

* [ ] The representative Laravel test command runs through both topologies.
* [ ] DDEV child exits and cancellation propagate correctly.
* [ ] Provider differences remain visible in plans and results.
* [ ] Integration cleanup is environment scoped.

## Phase 14: Homebrew Runtime Provisioning

Goal: Plan and install compatible host PHP runtimes through Homebrew without globally relinking the active PHP.

Dependencies: Phase 13 completed

### Tasks

1. Add failing Homebrew metadata and plan tests for available formulae, installed kegs, unsupported versions, offline mode, and approval.
2. Implement documented Homebrew structured inspection and versioned PHP formula selection.
3. Implement approved formula installation actions without global relinking or automatic uninstall.
4. Implement process environment construction from selected keg paths.
5. Add extension planning through bundled extensions, provider packages, and PIE capability boundaries.

### TEST

1. Run `go test ./internal/providers/homebrew`.
2. Run provider conformance for supported Homebrew capabilities.
3. Run plan tests against the local Homebrew metadata.
4. Run an explicitly approved isolated formula integration test when a safe formula target is available.

Completion Criteria:

* [ ] Homebrew plans are deterministic and approval gated.
* [ ] Elefante never changes the global PHP link.
* [ ] Selected keg paths produce the intended PHP executable.
* [ ] Unsupported extension installation produces a blocker.

## Phase 15: Isolated Tool Environments

Goal: Resolve, cache, and execute Composer distributed tools without project dependency mutation.

Dependencies: Phase 14 completed

### Tasks

1. Add failing tests for package constraints, runtime compatibility, cache identities, trust, ambiguous binaries, and project immutability.
2. Implement temporary isolated Composer project creation and verified atomic cache promotion.
3. Implement package binary discovery and explicit binary selection behavior.
4. Implement `elefante tool run` human and machine command contracts.
5. Add representative tool integrations for analysis, testing, and formatting packages.

### TEST

1. Run `go test ./internal/tools ./internal/cache ./internal/app`.
2. Snapshot project Composer files before and after every tool integration.
3. Run cold and warm tool execution tests.
4. Run concurrent cache preparation tests with race detection.

Completion Criteria:

* [ ] Tool execution never changes project dependencies.
* [ ] Compatible cached environments are reused safely.
* [ ] Ambiguous package binaries require explicit selection.
* [ ] Tool child exit codes are preserved.

## Phase 16: Framework And Workspace Completion

Goal: Complete public Phase 1 behavior for generic Composer, Bedrock WordPress, Symfony, Laravel packages, monorepos, and Git worktrees.

Dependencies: Phase 15 completed

### Tasks

1. Add end to end fixtures and command contracts for every supported framework adapter.
2. Implement WP CLI and Symfony Console execution through the common tool and run paths.
3. Complete monorepo explicit root selection and configuration behavior.
4. Complete workspace local state isolation for branches and Git worktrees.
5. Add legacy diagnostics for PHP 8.2, Laravel 11, and unsupported constraints.

### TEST

1. Run command contracts across the complete fixture matrix.
2. Run two worktrees from one repository and verify distinct local state and locks.
3. Run Bedrock WP CLI and Symfony Console representative commands.
4. Run `go test -race ./...`.

Completion Criteria:

* [ ] Every public Phase 1 project type completes its declared workflow.
* [ ] Worktree state cannot overwrite another workspace.
* [ ] Legacy status is visible without preventing provider supported execution.
* [ ] Framework shortcuts do not bypass the common executor.

## Phase 17: Hardening And Failure Matrix

Goal: Prove behavior under malformed input, interrupted operations, network failures, provider drift, and hostile output.

Dependencies: Phase 16 completed

### Tasks

1. Build a failure matrix covering discovery, constraints, provider inspection, downloads, trust, locks, Composer, child processes, state, and output encoding.
2. Add regression tests for every uncovered failure branch.
3. Run sustained fuzzing for JSON, TOML, constraints, redaction, canonical plans, and event encoding.
4. Add provider output version guards and unknown version failure behavior.
5. Run compiled binary tests in redirected, nonterminal, offline, frozen, and interrupted environments.

### TEST

1. Run `go test ./...`.
2. Run `go test -race ./...`.
3. Run `go vet ./...`.
4. Run the complete fuzz corpus and failure matrix suite.
5. Run real native and DDEV smoke tests after hardening changes.

Completion Criteria:

* [ ] Every known failure produces a typed actionable error.
* [ ] No failure path reports false success.
* [ ] Partial state remains diagnosable and retryable.
* [ ] Provider drift fails clearly.

## Phase 18: Performance, Compatibility, And Release Proof

Goal: Produce the evidence required for a Phase 1 public beta decision.

Dependencies: Phase 17 completed

### Tasks

1. Add reproducible benchmarks for startup, discovery, planning, wrapper overhead, synchronization, and tool execution.
2. Run the supported PHP and Laravel fixture matrix through native and DDEV providers where compatible.
3. Record compatibility results, known limitations, and legacy diagnostics.
4. Implement deterministic version metadata and Darwin arm64 release builds with checksums.
5. Complete documentation for installation, command behavior, machine protocol, security, and provider support.

### TEST

1. Run the complete repository, race, static, conformance, integration, and compiled binary suites.
2. Build a clean Darwin arm64 release artifact and verify its checksum.
3. Install the artifact into a clean temporary path and run all five primary command smoke tests.
4. Reproduce benchmark results from documented commands.

Completion Criteria:

* [ ] All Phase 1 technical design completion criteria pass.
* [ ] Compatibility claims are backed by recorded fixture results.
* [ ] Performance claims include machine details and reproducible commands.
* [ ] Release artifacts contain version, commit, platform, checksum, and provenance metadata.
* [ ] No container image, package publication, repository transfer, or external release occurs without explicit approval.
