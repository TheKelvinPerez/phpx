# Elefante Phase 1 CLI Technical Design

## Document Status

Version: 1.0

Updated: July 17, 2026

Status: Approved implementation design

Primary implementation language: Go

Initial platform: Apple Silicon macOS

Initial supported execution matrix: PHP 8.3, PHP 8.4, PHP 8.5, Laravel 12, and Laravel 13

This document turns the approved product direction in `specs/elefante-project-toolchain.md` into an implementation contract for the complete Phase 1 command surface.

## 1. Objective

Phase 1 must produce one trustworthy executable that can understand, plan, synchronize, and execute Composer projects across native and isolated environments.

The complete Phase 1 command surface is:

```console
elefante doctor
elefante plan
elefante sync
elefante run -- <command> [arguments]
elefante tool run <package-constraint> -- [arguments]
```

Implementation will proceed incrementally, but the architecture and public contracts in this document apply to all five commands before the first implementation phase begins.

## 2. Scope Boundary

### 2.1 Included

1. Project and Composer root discovery without project code execution.
2. Laravel, generic Composer, Bedrock WordPress, and Symfony detection.
3. Composer compatible platform requirement evaluation in Go.
4. Native, DDEV, and Homebrew provider capabilities.
5. Official Composer acquisition and invocation.
6. Deterministic plans and content addressed plan approval.
7. Synchronization with explicit trust and mutation boundaries.
8. Argument safe child command execution.
9. Isolated Composer tool environments.
10. Optional versioned `elefante.toml` configuration.
11. Versioned local state and verified caches.
12. Human output and one newline delimited JSON machine protocol.
13. Offline, frozen, noninteractive, cancellation, and failure behavior.
14. Test first implementation with command contracts, fixtures, conformance suites, integration tests, fuzzing, and compiled binary tests.

### 2.2 Excluded

1. A persistent daemon.
2. A graphical client.
3. A first party PHP runtime distribution.
4. Web routing, local DNS, or trusted TLS.
5. Managed databases, cache, mail, search, or object storage.
6. First party process supervision.
7. Deployment and rollback.
8. A committed `elefante.lock`.
9. An external provider plugin protocol.
10. A replacement for Composer dependency solving.
11. Automatic telemetry or crash uploads.

## 3. Fixed Architecture Decisions

1. The Go module path is `github.com/elefantephp/elefante`.
2. The executable and root command are named `elefante`.
3. Cobra is the thin command adapter.
4. Core packages do not import Cobra.
5. Official Composer remains the package dependency authority.
6. Go evaluates Composer platform constraints only.
7. `elefante.toml` is optional, committed, and versioned.
8. Phase 1 does not create `elefante.lock`.
9. `--json` always emits newline delimited JSON events.
10. Executable plans receive a SHA256 digest.
11. Machine clients may approve an exact plan with `--approve-plan`.
12. Composer plugin and script approval is tied to content fingerprints.
13. Phase 1 collects no automatic telemetry.
14. `doctor` performs no network access.
15. The first provider sequence is native inspection, DDEV isolation, then Homebrew provisioning.
16. The initial supported execution matrix is PHP 8.3 through PHP 8.5 and Laravel 12 through Laravel 13.
17. PHP 8.2 and Laravel 11 remain legacy diagnostic fixtures.
18. Every behavior is implemented through a red, green, refactor loop.

## 4. Toolchain And Dependency Policy

### 4.1 Go Toolchain

The initial module will declare Go 1.26 and pin the reviewed patch toolchain used by the repository.

The initial local toolchain is Go 1.26.5 on Darwin arm64. The exact patch may move only through a reviewed dependency maintenance change.

### 4.2 Runtime Dependencies

The initial runtime dependency set is intentionally small:

1. `github.com/spf13/cobra` for the command adapter.
2. `github.com/pelletier/go-toml/v2` for `elefante.toml` parsing.
3. `golang.org/x/term` for terminal detection and safe interaction decisions.

The standard library owns JSON, hashing, HTTP, processes, logging, paths, archives, checksums, concurrency, and tests wherever practical.

No dependency injection framework, logging framework, assertion framework, shell framework, or dynamic plugin system will be introduced during the initial implementation.

### 4.3 Dependency Review

Every new dependency requires:

1. A direct need that cannot be met safely by the standard library.
2. License review.
3. Maintainer and release activity review.
4. Vulnerability review.
5. Transitive dependency review.
6. A pinned module version in `go.mod` and checksum coverage in `go.sum`.
7. An explanation in the implementing commit or an ADR when the choice is difficult to reverse.

## 5. Repository Layout

The implementation will use this initial layout:

```text
cmd/elefante/main.go
internal/app/
internal/cache/
internal/cli/
internal/composer/
internal/config/
internal/constraints/
internal/discovery/
internal/executor/
internal/frameworks/
internal/model/
internal/output/
internal/paths/
internal/plan/
internal/providers/
internal/providers/native/
internal/providers/ddev/
internal/providers/homebrew/
internal/security/
internal/state/
internal/tools/
internal/version/
testdata/fixtures/
testdata/golden/
```

Package responsibilities are:

1. `app` coordinates command use cases.
2. `cli` constructs Cobra commands and maps flags to application requests.
3. `model` contains provider independent facts, requirements, plans, diagnostics, events, and results.
4. `discovery` reads project files and repository metadata.
5. `constraints` parses and evaluates Composer platform constraints.
6. `frameworks` adds framework facts without booting applications.
7. `plan` resolves facts and policy into deterministic actions.
8. `providers` defines provider contracts and shared conformance tests.
9. Provider subpackages implement one execution topology each.
10. `composer` acquires, verifies, and invokes official Composer.
11. `executor` runs argument vectors and preserves process semantics.
12. `tools` manages isolated Composer tool environments.
13. `config` parses and validates committed and user policy.
14. `state` persists local observations, approvals, action journals, and successful results.
15. `cache` stores verified artifacts and content addressed tool environments.
16. `security` owns redaction, checksums, trust fingerprints, and sensitive value policy.
17. `output` renders human output and machine events.
18. `paths` maps logical state locations to operating system paths.
19. `version` exposes build version, commit, platform, and protocol versions.

## 6. Dependency Direction

The dependency direction is strict:

```text
cmd → cli → app → model and service interfaces
app → discovery, plan, composer, executor, tools, state
plan → model and provider interfaces
provider implementations → model, executor, and provider contracts
output → model
```

The following imports are prohibited:

1. `model` importing providers, frameworks, output, or Cobra.
2. Providers importing CLI or output packages.
3. Framework adapters bypassing discovery or the resolver.
4. Command handlers calling operating system processes directly.
5. Application services writing directly to global terminal streams.

Package cycle tests and normal Go compilation enforce this boundary.

## 7. Application Construction

`main` performs only process level setup:

1. Construct platform paths.
2. Construct the local state and cache stores.
3. Construct network clients and process execution boundaries.
4. Register compiled provider implementations.
5. Construct application services.
6. Build the root Cobra command with `NewRootCommand(dependencies)`.
7. Execute the root command with a signal aware context.
8. Map the final typed result to a process exit code.

No package initialization function performs I/O, network access, registration, or global mutation.

## 8. Core Model

The final Go names may change for clarity during implementation, but these types and relationships are required.

### 8.1 Project Identity

```go
type ProjectIdentity struct {
    RepositoryRoot string
    ComposerRoot   string
    ApplicationRoot string
    WorkspaceRoot  string
    GitCommonDir   string
    Branch         string
    HeadCommit     string
    IdentityKey    string
}
```

`IdentityKey` is a local stable hash derived from normalized repository identity, Composer root relative path, and workspace identity. Secret remote credentials are removed before hashing.

### 8.2 Discovery Facts

```go
type ProjectFacts struct {
    Identity          ProjectIdentity
    Composer          ComposerFacts
    Frameworks        []FrameworkFact
    VersionFiles      []VersionFileFact
    ProviderMarkers   []ProviderMarker
    Configuration     ConfigFacts
    Diagnostics       []Diagnostic
    InputFingerprints []InputFingerprint
}
```

Every fact includes a source path and a source kind. Facts are immutable after discovery.

### 8.3 Requirements

```go
type Requirement struct {
    Name       string
    Kind       RequirementKind
    Constraint string
    Optional   bool
    Sources    []RequirementSource
}
```

Requirement kinds include PHP, PHP subtype, extension, system library, Composer, Composer plugin API, Composer runtime API, provider capability, and command capability.

### 8.4 Provider Observation

```go
type ProviderObservation struct {
    Provider     string
    Available    bool
    Version      string
    Capabilities []Capability
    Runtimes     []RuntimeObservation
    Composer     []ComposerObservation
    Extensions   []ExtensionObservation
    Diagnostics  []Diagnostic
    Fingerprint  string
}
```

The fingerprint covers provider facts that can change plan execution.

### 8.5 Plan

```go
type Plan struct {
    SchemaVersion string
    Operation     Operation
    Project       ProjectIdentity
    Provider      ProviderSelection
    Requirements  []RequirementResolution
    Actions       []PlanAction
    Diagnostics   []Diagnostic
    Trust         []TrustRequirement
    Inputs        []InputFingerprint
    Digest        string
}
```

### 8.6 Plan Action

```go
type PlanAction struct {
    ID               string
    Kind             ActionKind
    Summary          string
    Effect           EffectClass
    Network          NetworkRequirement
    Trust            TrustClass
    Reversible       bool
    Inputs           []ActionInput
    ExpectedOutputs  []ActionOutput
    Dependencies     []string
}
```

Effect classes are read, cache mutation, local Elefante state mutation, provider mutation, machine mutation, project mutation, and project code execution.

Action IDs are deterministic within the plan. They are derived from the ordered action kind and normalized inputs.

### 8.7 Diagnostics

```go
type Diagnostic struct {
    Code       string
    Severity   Severity
    Message    string
    Detail     string
    Hint       string
    Sources    []SourceReference
    Provider   string
    Retryable  bool
}
```

Diagnostic codes are stable public identifiers. Human wording may improve without changing code meaning.

## 9. Global Command Contract

### 9.1 Global Flags

The primary commands support these flags where meaningful:

1. `--json` emits the machine event protocol.
2. `--non-interactive` prohibits prompts.
3. `--yes` approves the freshly computed plan in one invocation.
4. `--approve-plan <digest>` approves only an exact previously reviewed plan.
5. `--provider <name>` requests a provider explicitly.
6. `--offline` prohibits Elefante initiated network access.
7. `--frozen` prohibits project file changes.
8. `--verbose` adds human diagnostic detail or machine debug events.
9. `--quiet` suppresses nonessential human output.
10. `--project <path>` selects the discovery starting path.
11. `--config <path>` selects an explicit Elefante configuration file.

### 9.2 Flag Rules

1. `--approve-plan` and `--yes` are mutually exclusive.
2. `--quiet` affects human mode only.
3. `--json` owns standard output completely.
4. When standard input is not a terminal, prompts are disabled even without `--non-interactive`.
5. A required approval in noninteractive mode produces an approval required error before mutation.
6. Explicit command flags override committed project policy.
7. Committed project policy overrides user preferences.
8. User preferences override automatic provider selection.

### 9.3 Help And Completion

Help output must include mutation, trust, network, and project code execution implications for each command.

Shell completion may suggest command names, provider names, task names, and safe file paths. Completion must not trigger network access, project code, provider mutation, or secret reads.

## 10. Command Semantics

### 10.1 `elefante doctor`

`doctor` is read only and performs no network access.

It must:

1. Discover the project and selected Composer root.
2. Parse Composer metadata and recognized intent files.
3. Detect the project type without booting application code.
4. Resolve PHP and extension requirements.
5. Inspect local providers.
6. Explain conflicts, missing capabilities, unsupported versions, and likely blockers.
7. Recommend the next command.

`doctor` does not acquire Composer, install dependencies, start DDEV, invoke project PHP, or write local trust approval.

Warnings produce a successful process exit when the project remains actionable. Blocking diagnostics produce the relevant Elefante exit category.

### 10.2 `elefante plan`

`plan` is read only. It may use read only network metadata unless `--offline` is active.

It must:

1. Perform all doctor analysis.
2. Resolve one deterministic provider selection.
3. Resolve runtime, extension, Composer, dependency, tool, and execution actions.
4. Separate cache, local state, provider, machine, project, and code execution effects.
5. Identify required approvals.
6. Produce a canonical plan digest.

The default operation is the plan for `sync`. Future operation selection may be added through an explicit flag without changing the plan model.

### 10.3 `elefante sync`

`sync` applies an approved synchronization plan.

It must:

1. Recompute discovery facts and provider observations.
2. Build the executable plan.
3. Validate `--approve-plan` when supplied.
4. Acquire all required locks.
5. Confirm trust for Composer plugins and scripts.
6. Apply runtime, extension, provider, Composer, and dependency actions in order.
7. Invoke official Composer with normal semantics.
8. Run official platform verification.
9. Persist a successful local environment record.
10. Report partial state and recovery guidance after failure.

`sync` does not claim transactional rollback for external package managers or Composer. Every applied action is journaled so the next invocation can diagnose partial progress.

### 10.4 `elefante run -- <command>`

`run` executes one argument vector inside the selected environment.

It must:

1. Require the command separator before the child argument vector.
2. Resolve the project, provider, and environment.
3. Refuse execution when the environment has unresolved blocking requirements.
4. Construct the working directory and environment overlay.
5. Execute without shell interpolation.
6. Stream input, output, and errors according to the selected output mode.
7. Forward cancellation and supported signals.
8. Preserve the child exit code after the child starts.

Shell semantics require an explicit shell executable in the supplied vector.

### 10.5 `elefante tool run <package-constraint>`

`tool run` creates or reuses an isolated Composer project outside the target project.

It must:

1. Parse an explicit package name and optional version constraint.
2. Resolve a compatible PHP runtime and Composer executable.
3. Plan package downloads, plugins, scripts, and executable discovery.
4. Apply the same trust, network, offline, and approval policies as synchronization.
5. Install into a content addressed tool environment.
6. Discover package binaries through Composer metadata and `vendor/bin`.
7. Execute one unambiguous binary or require an explicit binary selection.
8. Preserve the tool exit code.
9. Never modify project Composer files or dependencies.

## 11. Project Discovery

### 11.1 Starting Path

Discovery begins at `--project` when supplied, otherwise at the current working directory.

The path is converted to an absolute cleaned path. Symlinks are resolved for identity while the user supplied path remains available for display.

### 11.2 Root Selection

Discovery searches ancestors for Git and Composer boundaries.

Selection rules are:

1. If the starting path is inside exactly one Composer project, select the nearest ancestor `composer.json`.
2. If committed Elefante configuration selects a Composer root, validate and use it.
3. If the repository contains several Composer roots and the starting path does not identify one, return an ambiguity diagnostic with every candidate.
4. Do not silently select a nested project when several candidates are equally valid.
5. A project outside Git remains supported using its Composer root and normalized path identity.

### 11.3 Safe File Reading

Discovery may read recognized metadata files only.

It must not:

1. Include PHP files.
2. Load `vendor/autoload.php`.
3. Execute Composer plugins or scripts.
4. Run Artisan, WP CLI, Symfony Console, or application bootstrap files.
5. Read `.env` values into normal output.
6. follow a symlink outside the selected repository without reporting the boundary.

JSON readers reject malformed input, duplicate object keys, unexpected types, and configured file size limits.

### 11.4 Git Identity

When Git is available, discovery may use read only Git commands to determine repository root, common directory, worktree root, branch, and head commit.

Remote URLs are normalized with credentials removed. A missing remote does not block project use.

## 12. Framework Detection

Framework detection produces facts and confidence evidence. It never changes the core resolver.

### 12.1 Laravel Application

Strong evidence includes:

1. A root requirement on `laravel/framework`.
2. A project supplied `artisan` file.
3. Laravel bootstrap markers.
4. A conventional public front controller.

### 12.2 Laravel Package

A Laravel package contains Laravel or Illuminate requirements without application bootstrap markers. Package detection must not require a runnable Artisan file.

### 12.3 Bedrock WordPress

Detection uses Composer package requirements and Bedrock directory conventions. It must not assume WordPress core is at the repository root.

### 12.4 Symfony

Detection uses Symfony package requirements, console markers, and front controller conventions.

### 12.5 Generic Composer

Every valid Composer project receives the generic adapter even when no framework adapter reaches high confidence.

Conflicting framework evidence produces diagnostics instead of arbitrary selection.

## 13. Composer Metadata And Platform Resolution

### 13.1 Inputs

The resolver reads:

1. Root `composer.json` requirements and conflicts.
2. Root `composer.lock` package requirements and conflicts.
3. Composer platform configuration.
4. Composer plugin API and runtime API requirements.
5. `.php-version` when present.
6. Optional Elefante policy.
7. Provider runtime and extension observations.

### 13.2 Platform Emulation

Composer platform configuration is represented separately from the actual runtime requirement and actual provider observation.

Elefante reports when platform emulation can allow installation on a runtime that would fail at execution. Synchronization always concludes with official platform verification against the actual selected runtime.

### 13.3 Constraint Grammar

The Go evaluator supports Composer behavior required by platform packages:

1. Exact versions.
2. Comparison operators.
3. Logical AND and OR.
4. Wildcards.
5. Hyphen ranges.
6. Tilde ranges.
7. Caret ranges.
8. Stability suffixes and flags where platform evaluation requires them.
9. Normalization of common PHP version forms.

Unsupported syntax is a blocker with the original source preserved.

### 13.4 Conformance

Constraint tests include:

1. Golden cases derived from official Composer documentation.
2. A broad corpus compared with official `composer/semver` behavior.
3. Invalid constraint cases.
4. Fuzz tests for parser safety and deterministic results.
5. Regression fixtures for every discovered mismatch.

## 14. Optional `elefante.toml`

### 14.1 Location And Discovery

Elefante searches for `elefante.toml` at the repository root and selected Composer root. Two files at different roots are an ambiguity unless the repository file explicitly selects the project file.

An explicit `--config` path takes precedence.

### 14.2 Version 1 Schema

```toml
schema_version = 1

[project]
composer_root = "."

[providers]
preferred = ["native"]
allowed = ["native", "ddev", "homebrew"]
denied = []

[composer]
constraint = "^2"

[extensions]
optional = ["ext-xdebug"]

[tasks.test]
command = ["php", "artisan", "test"]
working_directory = "."

[ci]
provider = "ddev"
frozen = true
```

### 14.3 Schema Rules

1. `schema_version` is required when the file exists.
2. Unknown fields are errors for synchronization and warnings with blockers for doctor and plan.
3. Paths are relative to the configuration file and cannot escape the repository without explicit future support.
4. Task commands are argument arrays, never shell strings.
5. Task names must be unique and must not shadow primary Elefante commands.
6. Environment secret values are prohibited.
7. Composer requirements, dependencies, scripts, and repositories remain owned by Composer files.
8. Provider specific configuration remains owned by the provider unless Elefante explicitly creates a reviewed file.

### 14.4 Configuration Creation

Phase 1 may add a future explicit `elefante config init` convenience command after the five primary commands are stable. Until then, doctor provides a documented snippet without writing the file.

## 15. Provider Architecture

### 15.1 Provider Contract

The internal provider interface exposes:

```go
type Provider interface {
    Name() string
    Inspect(context.Context, InspectRequest) (ProviderObservation, error)
    Plan(context.Context, ProviderPlanRequest) (ProviderPlan, error)
    Apply(context.Context, ProviderAction, ActionRuntime) (ActionResult, error)
    ExecutionSpec(context.Context, ExecutionRequest) (ExecutionSpec, error)
}
```

Providers return typed data and errors. They do not prompt, render terminal output, select themselves, or mutate outside an approved action.

### 15.2 Capabilities

Initial capabilities include:

1. Inspect runtime.
2. Inspect extensions.
3. Inspect Composer.
4. Install runtime.
5. Install extension.
6. Start environment.
7. Execute command.
8. Provide network isolation information.
9. Provide platform and architecture identity.

### 15.3 Provider Selection

Selection order is:

1. Explicit `--provider`.
2. Committed Elefante policy.
3. Recognized provider configuration in the project.
4. Provider previously associated with the workspace when still compatible.
5. User default.
6. Deterministic best compatible provider.

Rejections include a stable reason code.

### 15.4 Native Provider

The native provider inspects executables already available through the process environment.

It may execute safe runtime inspection commands such as PHP version and extension reporting. It does not load project code.

The native provider does not install or globally relink PHP. When the current runtime is compatible, it supports Composer and child command execution.

### 15.5 DDEV Provider

DDEV is the first isolated provider.

The adapter uses documented DDEV commands and structured output where available. It must support:

1. Version and engine inspection.
2. Project configuration recognition.
3. PHP version and extension inspection.
4. Environment start planning.
5. Composer execution inside DDEV.
6. Argument safe project command execution.
7. Exit code and cancellation propagation.

When DDEV configuration is absent, Elefante may propose creating a minimal configuration only as an explicit project mutation. Frozen mode blocks that action.

### 15.6 Homebrew Provider

Homebrew provisioning follows native and DDEV proof work.

The adapter uses documented Homebrew structured metadata. It may install a compatible versioned PHP formula after approval.

It must not globally relink the user’s active PHP. Elefante constructs a process environment using the selected keg paths.

Extension installation uses bundled extensions, provider packages, or PIE according to an explicit plan. Unsupported extensions produce blockers rather than ad hoc compilation.

The adapter never uninstalls formulas during Phase 1.

### 15.7 Provider Conformance Suite

Every provider runs the same contract tests for:

1. Stable identity.
2. Capability inspection.
3. Runtime reporting.
4. Extension reporting.
5. Composer reporting.
6. Argument safe execution.
7. Working directory behavior.
8. Environment overlay behavior.
9. Exit code propagation.
10. Cancellation.
11. Offline behavior.
12. Secret redaction.
13. Typed errors.

## 16. Composer Acquisition And Invocation

### 16.1 Selection Order

Composer selection follows:

1. An explicit committed Composer constraint.
2. A verified Elefante managed Composer executable.
3. A compatible provider supplied Composer executable.
4. A compatible system Composer executable.

The exact selected executable, version, source, checksum when managed, and provider are visible in plans and results.

### 16.2 Managed Composer

Managed Composer downloads use official distribution sources and verification metadata.

The acquisition flow is:

1. Resolve an exact Composer version.
2. Download into a temporary cache path.
3. Verify expected cryptographic identity.
4. Record source and license metadata.
5. Atomically move the verified file into the content addressed cache.
6. Mark it executable only after verification succeeds.

Partial downloads never become executable.

### 16.3 Invocation

Composer is invoked as an argument vector with an explicit working directory and environment overlay.

Elefante preserves normal Composer authentication, plugins, scripts, lock behavior, and supported flags after trust approval.

Sensitive authentication values are inherited without being copied into plans, local state, or output.

### 16.4 Trust Fingerprint

The Composer execution fingerprint includes:

1. The selected `composer.json` content digest.
2. The selected `composer.lock` content digest when present.
3. The relevant Elefante configuration digest.
4. The selected Composer executable identity.
5. The plugin package identities discoverable before execution.
6. The script definitions discoverable before execution.

Changing a relevant input invalidates local trust approval.

## 17. Plan Construction And Digest

### 17.1 Plan Purity

Plan construction is a pure transformation after facts and provider observations are collected.

The planner does not perform downloads, start providers, write files, prompt, or invoke Composer.

### 17.2 Ordering

Actions form a directed acyclic graph and receive one deterministic execution order.

The default order is:

1. Cache preparation.
2. Runtime preparation.
3. Extension preparation.
4. Composer preparation.
5. Provider start or preparation.
6. Composer dependency installation.
7. Platform verification.
8. Local state recording.

### 17.3 Canonical Digest Input

The digest model uses fixed structs and sorted slices, not unordered maps.

It includes:

1. Protocol and plan schema versions.
2. Requested operation and flags that change behavior.
3. Project input fingerprints.
4. Configuration fingerprints.
5. Provider observation fingerprints.
6. Selected runtime, extensions, Composer, and tool identities.
7. Ordered actions and their normalized inputs.
8. Trust requirements.
9. Offline and frozen policy.

Display wording, timestamps, progress, local log paths, and terminal formatting are excluded.

The digest format is `sha256:<lowercase hexadecimal>`.

### 17.4 Apply Validation

`--approve-plan` triggers this sequence:

1. Rediscover project facts.
2. Reinspect relevant providers.
3. Rebuild the plan.
4. Recompute the digest.
5. Compare using constant time equality.
6. Exit without mutation when the digest differs.

The error includes the expected digest, actual digest, and categories of changed inputs without exposing secrets.

## 18. Synchronization Execution

### 18.1 Environment Lock

Only one mutating Elefante operation may act on one environment identity at a time.

Locks are user scoped files created atomically with owner process metadata. Stale lock recovery validates whether the owning process still exists before removal.

Read only doctor operations may run concurrently. Plan operations may run concurrently unless a provider documents an unsafe inspection boundary.

### 18.2 Action Journal

Before the first mutation, Elefante writes an atomic local action journal containing the approved plan digest and pending actions.

Each completed action updates the journal atomically.

After success, the journal becomes the environment’s last successful synchronization record. After failure, it remains available for diagnosis and safe retry.

### 18.3 Failure And Compensation

Actions declare whether they are reversible and may provide a compensation action.

Elefante automatically compensates only when the provider contract proves the action is safe to reverse. It does not uninstall Homebrew formulas, delete user caches, or reverse Composer changes automatically during Phase 1.

A failed synchronization reports:

1. Completed actions.
2. Failed action.
3. Actions not attempted.
4. Observed partial state.
5. Safe retry command.
6. Manual recovery guidance when required.

## 19. Process Execution

### 19.1 Execution Specification

```go
type ExecutionSpec struct {
    Executable       string
    Arguments        []string
    WorkingDirectory string
    Environment      []string
    InputMode        InputMode
    OutputMode       OutputMode
}
```

The executable and each argument remain separate through the complete stack.

### 19.2 Environment Construction

Providers return an environment overlay. The executor combines it with the permitted parent environment using documented precedence.

Plans expose environment variable names and redacted change categories, not secret values.

### 19.3 Signals And Cancellation

The root context responds to operating system interruption and termination signals.

When a child starts, supported signals are forwarded. On cancellation, Elefante allows a bounded graceful period before forced termination when the platform supports it.

Unix signal exits use the conventional `128 + signal` process status when available.

### 19.4 Exit Preservation

After a child starts, `run` and `tool run` return the child exit code exactly.

If Elefante fails before the child starts, it returns an Elefante exit category. The machine completion event identifies whether the exit originated from Elefante or the child.

## 20. Machine Event Protocol

### 20.1 Envelope

Every JSON line uses this envelope:

```json
{
  "schema": "elefante.events/v1",
  "sequence": 1,
  "command": "doctor",
  "type": "started",
  "payload": {}
}
```

### 20.2 Event Types

Phase 1 event types are:

1. `started`.
2. `fact`.
3. `diagnostic`.
4. `plan`.
5. `approval_required`.
6. `progress`.
7. `stdout`.
8. `stderr`.
9. `result`.
10. `error`.
11. `completed`.

### 20.3 Determinism

Canonical events exclude timestamps and random operation identifiers.

Sequences begin at one and increase without gaps. Arrays use documented stable ordering. Paths use normalized absolute or repository relative forms according to the field contract.

### 20.4 Child Streams

Valid UTF 8 child chunks use:

```json
{"encoding":"utf8","data":"..."}
```

Other bytes use:

```json
{"encoding":"base64","data":"..."}
```

Chunk boundaries are transport details. Clients must concatenate events by stream and sequence when exact output reconstruction is required.

### 20.5 Protocol Compatibility

New optional fields may be added within a schema version. Removing fields, changing field meaning, changing types, or changing required event ordering requires a new schema version.

## 21. Human Output

Human output is optimized for direct terminal use.

It must:

1. Lead with the selected project and provider.
2. Separate facts, warnings, blockers, planned mutations, trust requirements, and next actions.
3. Use color only when the output stream supports it.
4. Remain understandable without color.
5. Avoid spinners when output is redirected.
6. Stream child output without prefixing every line.
7. Show exact commands only when they are safe to display.
8. Redact sensitive values consistently with machine output.

Golden tests cover human output structure without locking incidental whitespace more tightly than necessary.

## 22. Error And Exit Contract

### 22.1 Stable Error Categories

Phase 1 defines:

1. `ELEFANTE_USAGE`, exit 2.
2. `ELEFANTE_DISCOVERY`, exit 3.
3. `ELEFANTE_REQUIREMENTS`, exit 4.
4. `ELEFANTE_PROVIDER`, exit 5.
5. `ELEFANTE_APPROVAL_REQUIRED`, exit 6.
6. `ELEFANTE_PLAN_MISMATCH`, exit 7.
7. `ELEFANTE_NETWORK`, exit 8.
8. `ELEFANTE_TRUST`, exit 9.
9. `ELEFANTE_SYNC`, exit 10.
10. `ELEFANTE_ARTIFACT`, exit 11.
11. `ELEFANTE_STATE`, exit 12.
12. `ELEFANTE_INTERNAL`, exit 70.

Exit zero means the requested Elefante operation completed successfully.

### 22.2 Child Exit Codes

Child exit codes take precedence after the child starts. The machine completion event includes:

```json
{
  "exit": {
    "origin": "child",
    "code": 1
  }
}
```

This resolves ambiguity when a child uses the same numeric code as an Elefante category.

### 22.3 Error Content

Every public error contains:

1. Stable code.
2. Category.
3. Human message.
4. Actionable hint when available.
5. Relevant source references.
6. Retryability.
7. Redacted structured details.

## 23. Local Paths And State

Logical paths use operating system conventions through `paths`.

On macOS the initial mapping is:

```text
User configuration: ~/Library/Application Support/Elefante/
User cache:         ~/Library/Caches/Elefante/
User logs:          ~/Library/Logs/Elefante/
```

State is organized beneath the user configuration root:

```text
config.toml
state/projects/<identity>/environment.json
state/projects/<identity>/trust.json
state/projects/<identity>/journal.json
locks/<identity>.lock
```

Cache organization is:

```text
composer/<content-identity>/
tools/<content-identity>/
downloads/<content-identity>/
metadata/<source-identity>/
```

Rules are:

1. Directories containing trust or state use user only permissions.
2. Sensitive files use user read and write permissions.
3. Writes use temporary files, flush, and atomic rename.
4. Cache identity uses content, not modification time alone.
5. Project repositories receive no local state files by default.
6. State schemas are versioned.
7. Unsupported future state versions fail safely without deletion.

## 24. Network Policy

1. `doctor` performs no network access.
2. `plan` may resolve read only artifact metadata.
3. `sync` accesses only sources represented in its approved plan.
4. `run` initiates no network access beyond child behavior.
5. `tool run` accesses the network only on a cache miss or explicit refresh.
6. `--offline` blocks all Elefante initiated network requests.
7. Missing offline artifacts fail before mutation.
8. HTTP clients use explicit connection, response, and total request timeouts.
9. Requests carry an Elefante version and platform user agent.
10. Redirects crossing an origin boundary are validated.
11. Download size limits and checksums are enforced.
12. Network metadata records source and retrieval identity without creating telemetry.

## 25. Security And Privacy

### 25.1 No Automatic Telemetry

Phase 1 does not upload usage, command, project, crash, or diagnostic data.

Any future diagnostic report is generated locally, redacted, and reviewed before manual sharing.

### 25.2 Secret Redaction

Redaction covers:

1. Common token and password variable names.
2. Composer authentication values.
3. Credentials embedded in URLs.
4. Private registry and repository credentials.
5. Authorization and cookie headers.
6. Sensitive query parameters.
7. Values explicitly marked secret by a provider.

Redaction tests use synthetic secrets and verify that raw values never appear in human output, machine events, logs, plans, state, or errors.

### 25.3 Project Code Trust

`doctor` and `plan` never execute project code.

Composer plugins and scripts require content based approval before execution. Noninteractive execution requires `--yes`, a matching plan digest, or explicit `--no-plugins` and `--no-scripts` behavior that removes the trust requirement.

### 25.4 Privilege

Elefante does not run a privileged daemon and does not invoke `sudo` automatically.

An action requiring elevated privileges becomes a blocker with a reviewable manual prerequisite unless a future audited privilege mechanism is specified.

## 26. Offline And Frozen Semantics

### 26.1 Offline

Offline mode permits cached verified artifacts and local provider operations. It blocks DNS, HTTP, registry, package metadata, and download access initiated by Elefante.

The plan states which actions are satisfied by cache and which are unavailable.

### 26.2 Frozen

Frozen mode prohibits changes to:

1. `composer.json`.
2. `composer.lock`.
3. `elefante.toml`.
4. Recognized provider configuration.
5. Other committed project files.

Frozen mode still permits approved cache, local state, provider, and machine mutations when they do not require project file changes.

Composer receives flags that preserve lock intent. A missing lock file for an operation that requires frozen dependency identity is a blocker.

## 27. Tool Environment Design

The tool cache identity includes:

1. Package name.
2. Resolved package version.
3. Composer lock content.
4. Selected PHP runtime identity.
5. Platform and architecture.
6. Composer identity.
7. Relevant plugin and script policy.

Tool environments are prepared in temporary directories and atomically promoted after successful installation and verification.

Concurrent callers share one content lock. A failed preparation never replaces a valid cached environment.

Cache pruning is not part of the five primary commands. The state and cache packages must expose internal inventory and removal operations so a later public cache command can be added without redesign.

## 28. Supported And Legacy Matrices

### 28.1 Public Execution Support

1. PHP 8.3.
2. PHP 8.4.
3. PHP 8.5.
4. Laravel 12.
5. Laravel 13.

### 28.2 Legacy Diagnostics

1. PHP 8.2.
2. Laravel 11.
3. Projects with unsupported PHP constraints.
4. Projects with conflicting root and lock requirements.

Legacy projects may execute when a provider supports them, but Elefante reports that they fall outside the supported matrix.

The matrix is versioned data. Framework detection and requirement resolution must not hardcode only the initial versions.

## 29. Test Strategy

### 29.1 Test First Rule

Every behavior follows:

1. Red, add one failing behavior test through the nearest public interface.
2. Green, implement the smallest behavior that passes.
3. Refactor, improve design only while the focused suite remains green.
4. Repeat for the next behavior.

Implementation commits must not contain large untested behavior batches.

### 29.2 Test Layers

1. Pure unit tests for parsers, normalization, canonicalization, redaction, and resolution.
2. In process command contract tests using constructed root commands and temporary filesystems.
3. Provider conformance tests shared by every provider.
4. Compiled binary tests that execute the real process.
5. Integration tests with official Composer.
6. DDEV integration tests through OrbStack on the initial machine.
7. Homebrew plan and controlled integration tests.
8. Fuzz tests for untrusted parsers and canonicalization.
9. Golden tests for human output and machine events.

### 29.3 Test Substitution Boundaries

Tests may substitute:

1. Filesystem access behind focused readers when a real temporary filesystem is not the better seam.
2. Process execution.
3. Provider implementations.
4. Network clients.
5. Clock only for local logs and cache freshness.
6. Terminal detection.

The normalized model, planner, event encoder, and command application remain real in command contract tests.

### 29.4 Required Verification Commands

The initial repository verification is:

```console
go test ./...
go test -race ./...
go vet ./...
go test ./internal/constraints -run TestComposerConformance
go test ./internal/providers/... -run TestProviderConformance
```

Compiled binary and integration suites receive explicit commands during their implementing phase.

### 29.5 Coverage Policy

There is no global percentage target that encourages low value tests.

The following correctness critical packages require branch focused coverage and explicit edge case inventories:

1. Composer constraints.
2. Plan canonicalization and digest.
3. Secret redaction.
4. Project discovery.
5. Provider selection.
6. Exit code mapping.
7. Trust fingerprinting.
8. State atomicity and lock recovery.

## 30. Fixture Matrix

The repository will contain generated or minimal purpose built fixtures for:

1. Laravel 13 on PHP 8.5.
2. Laravel 13 on PHP 8.4.
3. Laravel 12 on PHP 8.3.
4. Laravel 11 as a legacy diagnostic case.
5. A Laravel package.
6. A generic Composer library.
7. A generic Composer application.
8. A Bedrock WordPress project.
9. A Symfony project.
10. Multiple Composer roots in one repository.
11. Conflicting `.php-version` and Composer requirements.
12. Composer platform emulation.
13. Required and optional extensions.
14. Composer plugins and scripts.
15. Missing lock, stale lock, and malformed lock files.
16. DDEV configuration.
17. Unknown Elefante configuration fields.
18. Secret shaped values for redaction tests.

Fixtures do not require network access during normal unit and command contract tests.

## 31. Performance And Reliability Budgets

Initial budgets are measured before public claims.

The implementation must support:

1. Fast process startup without a daemon.
2. Discovery limited to relevant ancestors and selected repository files.
3. Content validated metadata reuse.
4. Bounded memory for streamed child output.
5. Atomic cache promotion.
6. Safe recovery from interrupted downloads.
7. Safe recovery from interrupted synchronization journals.
8. No whole home directory scans.

Benchmarks will measure cold and warm startup, discovery, planning, command wrapper overhead, synchronization, and tool execution.

## 32. Build And Release Contract

The initial release pipeline will eventually produce signed checksummed binaries for Darwin arm64.

Linux and Windows builds may compile before they are publicly supported, but they do not receive support claims until their provider and process conformance suites run on real systems.

Release metadata includes:

1. Semantic version.
2. Git commit.
3. Go toolchain.
4. Target operating system and architecture.
5. Checksums.
6. License and provenance information.

Container publication is not required for Phase 1 implementation and must receive a separate reviewed image design before use.

## 33. Completion Criteria

Phase 1 is implementation complete when:

1. All five primary commands satisfy their command contracts.
2. Human and machine output contracts pass golden tests.
3. Native and DDEV providers pass conformance tests.
4. Homebrew provisioning passes plan tests and approved integration coverage.
5. Official Composer performs synchronization and platform verification.
6. Child exit codes and signals are preserved.
7. Offline, frozen, noninteractive, trust, and plan digest behavior pass compiled binary tests.
8. Tool environments execute representative analysis, test, and formatting packages without project mutation.
9. Supported Laravel and generic Composer fixtures pass through both native and isolated topologies where compatible.
10. Bedrock and Symfony fixtures complete their Phase 1 workflows.
11. Redaction and artifact verification tests pass.
12. Race detection and static checks pass.
13. Compatibility and performance measurements are recorded with reproducible commands.

## 34. Deferred Decisions

The following decisions remain intentionally outside this technical design:

1. First party PHP runtime artifact sources.
2. Phase 2 daemon protocol.
3. PHP FPM versus FrankenPHP.
4. Local routing, DNS, and TLS implementation.
5. Managed service topology.
6. Graphical client technology.
7. Committed `elefante.lock` schema.
8. External provider executable protocol.
9. Production deployment providers.
10. Container image contents and release policy.

None of these decisions may expand Phase 1 implementation scope without a specification change.
