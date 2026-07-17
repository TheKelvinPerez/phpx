# PHPX: The Executable Environment for Composer Projects

## Document Status

**Version:** 3.0

**Updated:** July 16, 2026

**Status:** Approved product direction, Phase 1 specification ready for implementation planning

**Primary implementation language:** Go

**Primary Phase 1 market:** Laravel developers and teams

**Secondary Phase 1 markets:** Generic Composer projects, Laravel packages, Bedrock WordPress projects, Symfony projects, and PHP command line tools

**Phase 2 ambition:** A complete, open source, vertically integrated local PHP development platform that can replace Herd, Valet, Yerd, Lerd, Sail, DDEV, and fragmented local service tooling for supported workflows

**Long term ambition:** One coherent PHPX interface from project checkout through local development, continuous integration, build, deployment, health verification, and rollback

## 1. Product Decision

### 1.1 Vision

PHPX will become the one stop development and delivery platform for PHP.

It will reach that destination in deliberate layers. The first release will not attempt to own every local service. It will first own the path from project metadata to a correctly executed command.

The core product promise is:

> Composer describes what a PHP project needs. PHPX makes the environment satisfy those requirements, then runs the project correctly.

The initial experience should be:

```console
git clone <repository>
cd <repository>
phpx doctor
phpx sync
phpx run -- php artisan test
```

PHPX must preserve Composer as the canonical PHP package manager. PHPX manages the executable environment around Composer, including PHP selection, extension coordination, Composer acquisition, environment inspection, provider selection, command execution, diagnostics, isolated tools, and continuous integration parity.

### 1.2 Strategic Sequence

PHPX will use a land and expand strategy.

#### Phase 1: Project Execution Layer

PHPX provides one Composer aware interface across existing PHP environments.

Phase 1 includes:

1. Project discovery without executing project code.
2. PHP and extension requirement analysis.
3. Environment diagnostics and explainable plans.
4. Environment provider selection.
5. Runtime, extension, and Composer synchronization through providers.
6. Correct command execution inside the selected environment.
7. Isolated execution of PHP tools.
8. Laravel first convenience and validation.
9. Generic Composer compatibility.
10. Continuous integration execution using the same project interpretation.

Phase 1 succeeds when developers adopt PHPX as the command they use to understand, prepare, and run PHP projects.

#### Phase 2: Vertically Integrated Local Platform

PHPX adds a complete first party local environment provider and progressively replaces the external providers used during Phase 1.

Phase 2 includes:

1. Managed PHP runtime artifacts.
2. Managed PHP extensions.
3. A shared local web server and routing layer.
4. Local DNS and trusted TLS certificates.
5. Shared databases, caches, search, object storage, and mail capture.
6. Framework aware process supervision.
7. Multi project portfolio management.
8. Database import, export, snapshots, and restore.
9. Laravel, WordPress, Symfony, and generic PHP application adapters.
10. A graphical interface with complete command line and API parity.
11. Migration assistance from Herd, Valet, Yerd, Lerd, Sail, DDEV, and common manual environments.
12. Native local performance where safe, with explicit isolated fallbacks where required.

Phase 2 succeeds when a supported developer can install PHPX on a clean machine and use it as the only local PHP environment product.

#### Phase 3: Production Delivery Platform

PHPX extends the project contract from local and continuous integration environments into production delivery.

Phase 3 includes:

1. Reproducible build outputs.
2. Deployment planning and environment validation.
3. Provider adapters for managed platforms, self hosted servers, and container platforms.
4. Secrets boundaries and environment policy.
5. Database migration orchestration.
6. Health checks, release history, rollback, and auditability.
7. Production diagnostics using the same project model as local development.

Phase 3 succeeds when PHPX provides one coherent workflow from checkout to production without requiring PHPX to become the only possible hosting provider.

### 1.3 Why the Entry Wedge Changed

The original specification treated the complete local environment as the first product. Current market validation shows that this is no longer a sufficiently distinct entry wedge.

Herd now has a substantial command line interface and committed project configuration. DDEV already provides mature cross platform project configuration and automation. Yerd provides an open source native local environment. Lerd provides an open source shared Podman environment with committed configuration and broad framework support.

PHPX should not enter by recreating capabilities that these products already provide. It should enter at the layer they do not share: one Composer aware execution contract across environments.

The current market creates two opportunities:

1. A provider neutral project execution layer is not yet established as the common PHP workflow.
2. Once PHPX owns that workflow, it can introduce a first party provider without forcing users to learn a new interface or migrate every project at once.

### 1.4 Positioning

PHPX is:

1. The executable environment layer around Composer.
2. A Composer aware project resolver and runner.
3. A stable interface across local environment providers.
4. A path from fragmented environments toward one vertically integrated platform.
5. A command line first product that is safe for scripts, continuous integration, editors, and future graphical clients.

PHPX is not:

1. A Composer replacement.
2. A new PHP package ecosystem.
3. A free clone of Herd.
4. A mandatory container abstraction.
5. A new project manifest that every repository must adopt before receiving value.
6. A deployment platform during Phase 1.
7. A graphical application during Phase 1.

PHPX should not be marketed as “uv for PHP.” The useful lesson from uv is the integrated experience of install, sync, run, tools, caching, and compatibility. Composer already owns PHP package resolution and must remain the dependency authority.

### 1.5 Product Tagline

The recommended initial tagline is:

> Clone. Sync. Run.

The recommended explanatory line is:

> PHPX turns Composer requirements into an executable environment.

## 2. Problem Definition

### 2.1 The Missing Layer Around Composer

Composer standardizes PHP package dependencies, lock files, autoloading, plugins, scripts, and repository authentication.

Composer also exposes requirements for PHP and PHP extensions as platform packages. It validates those requirements, but it does not install the required PHP runtime, configure extensions, select a local server, or provision application services.

This creates a recurring gap between package intent and executable reality.

A repository may correctly declare:

1. A PHP version constraint.
2. Required extensions such as `ext-intl`, `ext-pdo`, or `ext-redis`.
3. Composer scripts.
4. Framework packages.
5. Development tools.

The repository still cannot guarantee that a developer has a compatible PHP binary, the correct extensions, the expected Composer executable, or the correct command environment.

### 2.2 Fragmented Environment Providers

PHP developers currently reach executable environments through products such as:

1. Herd.
2. Valet.
3. Yerd.
4. Lerd.
5. DDEV.
6. Sail.
7. Local.
8. ServBay.
9. Laragon.
10. mise, asdf, Devbox, or devenv.
11. System package managers and manual instructions.

These tools solve important problems, but each introduces its own configuration, capabilities, command surface, operating system support, and execution model.

There is no shared command that can reliably answer:

1. What does this Composer project require?
2. What does this machine currently provide?
3. Which compatible environment provider is available?
4. What will change if the project is synchronized?
5. How should this command be executed correctly?
6. Can the same interpretation be enforced in continuous integration?

### 2.3 Onboarding and Context Switching

The recurring job is not merely installing PHP once. Developers frequently move among projects with different PHP versions, extensions, tools, framework commands, and environment providers.

The target job is:

> When I clone or enter a PHP project, I want one command to understand its requirements, explain any mismatch, prepare a compatible environment, and run the requested command without undocumented machine setup.

### 2.4 Existing Strengths That PHPX Must Preserve

PHPX must build around the parts of the ecosystem that already work.

1. Composer remains the package dependency authority.
2. PIE remains the preferred extension installer where it is supported.
3. Artisan remains the Laravel application interface.
4. WP CLI remains the WordPress command interface.
5. Framework supplied test and development commands retain their semantics.
6. Existing local environment providers remain usable.
7. Existing projects remain usable without PHPX.

### 2.5 Claims That Still Require Validation

The following claims are plausible but not yet sufficiently proven:

1. A large population will switch products primarily for lower local memory use.
2. Exact runtime patch locking provides enough team value to justify another committed lock file.
3. A shared native database topology is preferred over isolated project services by most target teams.
4. Windows users will accept platform specific differences behind a common interface.
5. Deployment should become a first party PHPX engine instead of a provider integration layer.

These questions must be answered through prototypes, interviews, compatibility fixtures, and published benchmarks rather than assumption.

## 3. Product Principles

### 3.1 Composer Remains Canonical

`composer.json` and `composer.lock` remain the source of truth for PHP package dependencies, dependency resolution, autoloading, Composer plugins, Composer scripts, repository authentication, and package installation semantics.

PHPX must invoke the official Composer implementation during Phase 1. Native Composer acceleration may be explored later only when compatibility is measurable and transparent fallback remains available.

### 3.2 Useful Without Migration

A Composer project must receive useful PHPX diagnostics and execution behavior without adding a PHPX manifest or restructuring its repository.

PHPX must derive intent from established files before asking a project to declare new intent.

### 3.3 Projects Remain Usable Without PHPX

Removing PHPX from a machine must not make a project structurally unusable. A developer with a compatible PHP runtime, Composer, and required services must still be able to use normal project commands.

### 3.4 One Interface, Multiple Providers

The user facing interface must remain consistent while environment providers change.

Phase 1 integrates with providers. Phase 2 adds a first party provider. Phase 3 adds deployment providers.

Provider differences must be explained instead of hidden. PHPX must not claim two providers are equivalent when their isolation, operating system, services, or runtime semantics differ.

### 3.5 Vertical Integration in Layers

PHPX will vertically integrate through a staged ownership model.

1. Phase 1 owns interpretation, planning, synchronization, execution, and diagnostics.
2. Phase 2 owns the complete supported local environment.
3. Phase 3 owns the delivery workflow and may delegate infrastructure execution to production providers.

Later ambition must not expand the active implementation scope of an earlier phase.

### 3.6 Explain Every Decision

PHPX must be able to explain:

1. Which project root it selected.
2. Which framework or project type it detected.
3. Which PHP constraint it derived.
4. Which extensions it considers required.
5. Which environment provider it selected.
6. Why another provider was rejected.
7. Which Composer executable it will invoke.
8. Which files or commands caused each decision.
9. Which actions will mutate the machine or repository.

### 3.7 Plan Before Mutation

Read only discovery and planning must be available before PHPX modifies a machine, project, provider, service, or lock file.

Elevated operations and system trust changes require explicit approval. Noninteractive mutation requires an explicit confirmation flag.

### 3.8 Headless First

Every core operation must work through the command line without a graphical application.

Commands must provide stable exit codes, structured JSON, noninteractive behavior, and actionable errors.

The Phase 2 graphical application must consume the same daemon API and must not contain capabilities unavailable through the command line or API.

### 3.9 Compatibility Before Replacement

PHPX must delegate to existing implementations when it cannot preserve their semantics.

Unsupported behavior must fail clearly or use a declared fallback. It must never appear successful after performing only a partial approximation.

### 3.10 Secure Supply Chain

Downloaded runtimes, extensions, Composer executables, tools, services, and updates must have verifiable provenance and checksums.

Release signing, software bills of materials, minimal privileges, auditable artifact metadata, and rollback must be designed before PHPX distributes executable artifacts broadly.

### 3.11 Go Owns the Control Plane

Go is the default implementation language for the PHPX command line, resolver, provider coordination, structured output, caching, process execution, and future daemon.

PHP remains the correct runtime for Composer, Composer plugins, project scripts, Artisan, WP CLI, and PHP based tools.

The language is an implementation decision, not the product differentiation.

## 4. Target Users and Framework Strategy

### 4.1 Primary Phase 1 User

The primary user is a Laravel developer or team that works across multiple applications, PHP versions, extensions, and local environment products.

This user wants one reliable command surface without immediately abandoning the provider that already works on their machine.

### 4.2 Additional Users

1. Laravel package maintainers who test across PHP and Laravel versions.
2. Generic Composer library maintainers.
3. WordPress developers using Bedrock or Composer managed sites.
4. Symfony application developers.
5. Continuous integration maintainers.
6. Agencies managing many PHP projects.
7. Developers who want isolated PHP command line tools.

### 4.3 Framework Sequence

The implementation sequence is:

1. Laravel applications and Laravel packages.
2. Generic Composer applications and libraries.
3. Bedrock and Composer managed WordPress projects.
4. Symfony applications.
5. Classic WordPress through WP CLI and a capable local provider.

Laravel is the first complete user experience. It must not become a hard dependency of the core resolver or provider model.

### 4.4 Phase 2 Framework Expectations

The Phase 2 local platform must provide first class support for:

1. Laravel applications, queues, schedulers, Horizon, Octane where supported, Reverb, Vite processes, storage links, and Artisan workflows.
2. Classic WordPress, Bedrock, Composer managed WordPress, WP CLI, database import and export, search and replace, mail capture, and local TLS.
3. Symfony front controllers, console commands, workers, and common runtime layouts.
4. Generic PHP document roots, front controllers, command processes, and Composer scripts.

## 5. Phase 1 User Journeys

### 5.1 Inspect an Existing Project

```text
Developer enters repository
PHPX discovers project files without executing project code
PHPX derives PHP and extension requirements
PHPX inspects available providers
PHPX explains compatibility, conflicts, and missing capabilities
Developer receives an actionable plan
```

Primary command:

```console
phpx doctor
```

### 5.2 Preview Environment Preparation

```text
Developer requests a plan
PHPX chooses a provider according to explicit and discovered intent
PHPX lists runtime, extension, Composer, dependency, and command actions
PHPX identifies machine changes and project changes separately
No mutation occurs
```

Primary command:

```console
phpx plan
```

### 5.3 Synchronize a Project

```text
Developer approves synchronization
PHPX prepares or selects the provider environment
PHPX acquires or selects a compatible PHP runtime
PHPX coordinates required extensions
PHPX acquires or selects Composer
PHPX invokes Composer with normal semantics
PHPX verifies the resulting platform requirements
PHPX reports the synchronized environment
```

Primary command:

```console
phpx sync
```

### 5.4 Run a Project Command

```text
Developer supplies an argument vector
PHPX resolves the project and provider
PHPX verifies that the environment is usable
PHPX executes the command without shell string interpolation
PHPX streams output and returns the child exit code
```

Primary command:

```console
phpx run -- php artisan test
```

### 5.5 Run an Isolated Tool

```text
Developer names a Composer package that exposes an executable
PHPX resolves an isolated tool environment
PHPX reuses verified cached artifacts where possible
PHPX executes the tool without modifying project dependencies
PHPX removes disposable state according to cache policy
```

Primary command:

```console
phpx tool run phpstan/phpstan -- analyse
```

### 5.6 Use the Same Interpretation in Continuous Integration

```text
Continuous integration installs the PHPX binary
PHPX reads the same Composer metadata and optional PHPX configuration
PHPX selects the continuous integration provider
PHPX synchronizes in noninteractive mode
PHPX executes tests and returns stable exit codes
```

Primary commands:

```console
phpx sync --non-interactive --frozen
phpx run -- php artisan test
```

## 6. Phase 1 Command Line Interface

### 6.1 Essential Commands

The initial public command surface contains five primary commands.

#### `phpx doctor`

Inspects project intent, the current machine, available providers, runtime compatibility, extensions, Composer, and likely execution blockers.

`doctor` is read only.

Required output includes:

1. Detected project root and type.
2. Composer files used.
3. PHP constraints and their sources.
4. Required extensions and their sources.
5. Available providers and capabilities.
6. Selected provider or reason no provider can be selected.
7. Composer availability.
8. Blocking conflicts.
9. Recommended next command.

#### `phpx plan`

Produces the ordered action plan that `sync` would execute.

`plan` is read only.

The plan must distinguish:

1. Machine mutations.
2. Provider mutations.
3. Project file mutations.
4. Dependency installation.
5. Commands that may execute project supplied code.
6. Cache reads and downloads.

#### `phpx sync`

Brings the selected executable environment into agreement with project requirements.

`sync` may:

1. Select or install a runtime through a provider.
2. Select or install extensions through a provider or PIE.
3. Select or install Composer.
4. Invoke `composer install`.
5. Verify installed platform requirements.
6. Write PHPX local state.
7. Write an optional PHPX lock only when explicitly requested or required by repository policy.

#### `phpx run -- <command>`

Executes a command inside the selected project environment.

The separator is required whenever command arguments could be confused with PHPX flags.

The initial implementation must preserve the child process exit code and signal behavior where the operating system permits it.

#### `phpx tool run <package> -- <arguments>`

Runs a Composer distributed command line tool in an isolated cached environment without adding it to project dependencies.

The package name must be explicit during the first release. Friendly aliases may be introduced after collision and trust policies are defined.

### 6.2 Global Flags

All applicable commands must support:

1. `--json` for versioned structured output.
2. `--non-interactive` to prohibit prompts.
3. `--yes` to approve declared mutations in noninteractive workflows.
4. `--provider <name>` to choose a provider explicitly.
5. `--offline` to prohibit network access.
6. `--frozen` to prohibit lock and project configuration changes.
7. `--verbose` for additional diagnostic context.
8. `--quiet` for minimal output while preserving errors.

### 6.3 Convenience Commands

The following shortcuts may be added after the five primary commands are stable:

```console
phpx composer <arguments>
phpx artisan <arguments>
phpx wp <arguments>
phpx test
```

Each shortcut must compile to the same execution path as `phpx run`. Shortcuts must not introduce separate environment logic.

## 7. Phase 1 Project Discovery

### 7.1 Discovery Inputs

PHPX must recognize, when present:

1. `composer.json`.
2. `composer.lock`.
3. `.php-version`.
4. `artisan`.
5. Laravel bootstrap and public front controller files.
6. Symfony console and front controller files.
7. Bedrock structure.
8. WordPress core and WP CLI markers.
9. `.ddev/config.yaml`.
10. `herd.yml`.
11. `.lerd.yaml`.
12. Yerd site information when exposed through a stable command or documented interface.
13. `docker-compose.yml` and `compose.yaml`.
14. Laravel Sail files.
15. `mise.toml` and `.tool-versions`.
16. Optional PHPX configuration.

### 7.2 Discovery Safety

Discovery must not:

1. Execute project supplied PHP.
2. Load application bootstrap files.
3. Execute Composer plugins or scripts.
4. Read secret values into normal output.
5. Modify provider state.
6. Modify the repository.

### 7.3 Project Root Selection

PHPX must distinguish:

1. Repository root.
2. Composer project root.
3. Application root.
4. Public document root.
5. Tool working directory.
6. Monorepo or workspace boundaries.

Ambiguous roots must be explained and require an explicit choice. PHPX must not silently select a nested project when multiple valid roots exist.

### 7.4 Intent Precedence

The initial precedence is:

1. Explicit command line arguments.
2. Optional committed PHPX configuration.
3. Composer requirements and lock metadata.
4. Standard version files.
5. Recognized provider configuration.
6. Framework conventions.
7. Current environment as a final fallback.

Conflicts must be reported with the files and values involved.

## 8. Phase 1 Environment Model

### 8.1 Normalized Execution Plan

The resolver must produce a normalized internal plan containing:

1. Project identity and type.
2. Working directory.
3. PHP version constraint.
4. Selected PHP runtime when known.
5. Required and optional PHP extensions.
6. Composer version policy.
7. Composer install mode and flags.
8. Selected environment provider.
9. Provider capabilities used.
10. Environment variables with secret values redacted.
11. Requested command and arguments.
12. Network requirements.
13. Planned mutations.
14. Cache inputs and outputs.
15. Warnings, blockers, and fallback decisions.

The normalized plan is the stable center of the architecture. Human output, JSON output, providers, continuous integration, and the future GUI must consume this model.

### 8.2 PHP Requirement Resolution

PHPX must consider:

1. Root Composer PHP constraints.
2. Locked package PHP constraints.
3. Composer platform configuration.
4. `.php-version`.
5. Optional PHPX policy.
6. Provider availability.
7. Operating system and architecture.

PHPX must distinguish an actual project requirement from a Composer platform emulation setting.

PHPX must never choose a runtime that violates locked package requirements merely because the root requirement appears compatible.

### 8.3 Extension Requirement Resolution

PHPX must identify required `ext-*` platform packages from Composer metadata.

For each extension, the plan must report:

1. Requirement source.
2. Whether the extension is bundled, dynamically installed, or provided externally.
3. Provider support.
4. Version information when discoverable.
5. Whether installation requires compilation, elevated privileges, or a restart.

PIE should be used where it is compatible and provides the required package. Provider supplied extensions remain valid when their provenance and runtime compatibility are known.

### 8.4 Composer Selection

PHPX must use the official Composer executable.

The selection policy must support:

1. A project pinned Composer policy when explicitly declared.
2. A verified managed Composer executable.
3. A compatible provider supplied Composer executable.
4. A compatible system Composer executable as a fallback.

PHPX must expose the exact Composer executable and version used.

### 8.5 Configuration and Locking

Phase 1 must not require `phpx.toml` or `phpx.lock` for basic use.

An optional `phpx.toml` may be introduced for intent that established project files cannot express, including:

1. Allowed or preferred providers.
2. PHPX task aliases.
3. Project specific environment policy.
4. Continuous integration policy.
5. Optional extension policy.
6. Monorepo project selection.

An optional `phpx.lock` may be introduced only when it locks information not already owned by `composer.lock`, including:

1. PHP runtime artifact identity and checksum.
2. Composer executable version and checksum.
3. Extension provider and artifact identity.
4. Isolated tool package versions.
5. Provider protocol version when required for reproducibility.

`phpx.lock` must never duplicate the Composer package dependency graph.

Exact runtime locking must earn adoption through user validation before it becomes a default repository artifact.

## 9. Phase 1 Provider Architecture

### 9.1 Provider Purpose

An environment provider turns a normalized execution plan into an executable process environment.

Providers let PHPX deliver one interface before PHPX owns the complete local stack.

### 9.2 Required Provider Capabilities

A provider may expose capabilities for:

1. Inspecting available PHP runtimes.
2. Installing or selecting PHP.
3. Inspecting and enabling extensions.
4. Providing Composer.
5. Constructing command environment variables and paths.
6. Executing a command.
7. Reporting runtime identity and provenance.
8. Reporting whether network access or elevated privileges are required.

Phase 2 extends the same capability model with serving, DNS, TLS, services, processes, and graphical management.

### 9.3 Initial Provider Set

The technical proof must support two different execution topologies:

1. A host or native provider.
2. An isolated provider such as DDEV or another container based environment.

The first host provider may use an existing system runtime or a documented provider interface. Selection among Herd, Yerd, Lerd, mise, system packages, and an early PHPX artifact provider requires a focused artifact and integration spike.

The proof must not depend on undocumented private APIs.

### 9.4 Provider Selection

Provider selection follows this order:

1. Explicit `--provider` argument.
2. Committed PHPX provider policy when present.
3. Recognized project provider configuration.
4. Compatible provider already associated with the project.
5. Configured user default.
6. Best compatible discovered provider.

Automatic selection must be explainable and deterministic for the same machine state.

### 9.5 Provider Conformance

Every supported provider must pass a conformance suite that verifies:

1. Capability inspection.
2. PHP identity reporting.
3. Extension reporting.
4. Environment construction.
5. Argument safe command execution.
6. Exit code propagation.
7. Structured error translation.
8. Secret redaction.
9. Cancellation and signal handling.
10. Offline behavior where supported.

### 9.6 Provider Extension Boundary

Phase 1 should begin with providers compiled into the Go binary.

An external executable provider protocol may be introduced after the internal interface and JSON schema stabilize. Dynamic in process plugins are not required and increase supply chain and compatibility risk.

## 10. Phase 1 Synchronization and Execution

### 10.1 Synchronization Sequence

`phpx sync` must execute an explicit state transition:

```text
Discover project
Resolve requirements
Select provider
Inspect provider state
Build mutation plan
Request approval when required
Prepare runtime
Prepare extensions
Prepare Composer
Invoke Composer install
Run Composer platform verification
Record local state
Report final environment
```

### 10.2 Composer Semantics

PHPX must preserve:

1. Composer authentication.
2. Composer plugins.
3. Composer scripts.
4. Composer environment variables.
5. Composer configuration precedence.
6. Lock file semantics.
7. Exit codes.
8. Standard flags such as `--no-dev`, `--no-scripts`, and `--prefer-dist`.

PHPX must clearly state when `composer install` can execute project supplied or dependency supplied code.

### 10.3 Command Execution

PHPX must pass commands as argument vectors, not interpolated shell strings.

The execution layer must define:

1. Working directory.
2. Executable path.
3. Arguments.
4. Environment overlay.
5. Standard input behavior.
6. Standard output and error streaming.
7. Signal forwarding.
8. Exit code propagation.
9. Timeout and cancellation behavior.

Shell execution must be explicit when a user requests shell semantics.

### 10.4 Isolated Tool Execution

Tool environments must be separate from project dependencies.

The tool runner must:

1. Resolve packages through Composer semantics.
2. Store each resolved environment under a content derived identity.
3. Reuse compatible environments safely.
4. Expose the exact package and version used.
5. Support version constraints.
6. Avoid global Composer dependency mutation.
7. Verify the selected runtime satisfies the tool.

Example:

```console
phpx tool run "phpstan/phpstan:^2" -- analyse
```

### 10.5 Cache Model

Phase 1 may cache:

1. Verified Composer executables.
2. Provider metadata.
3. Runtime metadata.
4. Downloaded artifacts when licensing permits.
5. Isolated tool environments.
6. Parsed project metadata keyed by file content.

Cache correctness must never depend only on modification time.

The cache must support inspection, pruning, offline use, and safe recovery from partial writes.

## 11. Phase 1 Framework Adapters

### 11.1 Laravel Adapter

The Laravel adapter must:

1. Detect Laravel without booting the application.
2. Distinguish an application from a Laravel package.
3. Identify the project supplied `artisan` executable.
4. Run Artisan through the selected environment.
5. Detect common test commands.
6. Identify Sail configuration without requiring Sail.
7. Explain common PHP and extension conflicts.
8. Preserve normal Composer and Artisan behavior.

Local routing, queues, schedulers, databases, mail, Vite, Horizon, Reverb, and Octane are Phase 2 responsibilities unless an existing provider already exposes them during Phase 1.

### 11.2 Generic Composer Adapter

The generic adapter must:

1. Work without framework markers.
2. Discover Composer binaries and scripts.
3. Run project test, analysis, formatting, and build commands.
4. Support library repositories with no web document root.

### 11.3 Bedrock WordPress Adapter

The initial WordPress adapter must:

1. Detect Bedrock and Composer managed WordPress structures.
2. Resolve Composer requirements normally.
3. Run WP CLI through an isolated tool or project supplied executable.
4. Avoid assuming that WordPress core lives at the repository root.

Database lifecycle, domain replacement, and web serving depend on provider capabilities in Phase 1 and become first party capabilities in Phase 2.

### 11.4 Symfony Adapter

The Symfony adapter must:

1. Detect Symfony applications and console commands.
2. Run project supplied console commands through the selected environment.
3. Preserve Composer scripts and Symfony Runtime behavior.

## 12. Phase 1 Security Requirements

### 12.1 Trust Boundaries

PHPX must distinguish:

1. Trusted PHPX code.
2. Verified PHPX artifacts.
3. Provider executables.
4. Composer and Composer plugins.
5. Project supplied code and scripts.
6. Dependency supplied code.
7. User supplied shell commands.

### 12.2 Read Only Inspection

`doctor` and `plan` must not execute project code.

If a provider cannot inspect state without mutation or project execution, PHPX must report that limitation before calling it.

### 12.3 Artifact Verification

Every managed executable artifact must include:

1. Source URL or registry identity.
2. Version.
3. Platform and architecture.
4. Cryptographic checksum.
5. Signature or attestation where available.
6. License metadata.
7. Installation time.

Downloads must use atomic installation and must not become executable before verification succeeds.

### 12.4 Secret Handling

PHPX must redact likely secrets from human and JSON diagnostic output.

PHPX must not copy production secrets into local configuration or committed project files.

Provider commands that may expose secrets require output filtering and conformance tests.

### 12.5 Privilege Handling

Phase 1 must not require a persistent privileged daemon.

Any elevated package installation must be visible in the plan and explicitly approved. PHPX must prefer user scoped installation where practical.

## 13. Phase 1 Functional Requirements

### 13.1 Discovery

**FR 1.1:** PHPX must discover a Composer project from any descendant directory.

**FR 1.2:** PHPX must detect Laravel applications and Laravel packages without booting them.

**FR 1.3:** PHPX must support generic Composer repositories without framework assumptions.

**FR 1.4:** PHPX must report ambiguous project roots instead of choosing silently.

**FR 1.5:** Discovery must not execute project supplied code.

### 13.2 Resolution

**FR 2.1:** PHPX must derive PHP constraints from Composer metadata.

**FR 2.2:** PHPX must derive required extensions from Composer metadata.

**FR 2.3:** PHPX must detect conflicts among Composer metadata, version files, PHPX policy, and provider state.

**FR 2.4:** Every selected value must include an explanation source.

**FR 2.5:** The resolver must produce versioned structured output.

### 13.3 Providers

**FR 3.1:** PHPX must select a provider explicitly or deterministically.

**FR 3.2:** PHPX must report provider capabilities before mutation.

**FR 3.3:** The technical proof must execute through at least one host topology and one isolated topology.

**FR 3.4:** Provider failures must map to stable PHPX error categories while retaining useful provider context.

### 13.4 Synchronization

**FR 4.1:** `phpx plan` must show the actions `phpx sync` would perform.

**FR 4.2:** `phpx sync` must use the official Composer implementation.

**FR 4.3:** `phpx sync --frozen` must not change Composer or PHPX lock files.

**FR 4.4:** `phpx sync --offline` must fail before network access when required artifacts are missing.

**FR 4.5:** Repeating `phpx sync` against an already synchronized project must be safe and materially faster than initial synchronization.

### 13.5 Execution

**FR 5.1:** `phpx run` must execute the supplied argument vector inside the selected environment.

**FR 5.2:** `phpx run` must preserve the child exit code.

**FR 5.3:** PHPX must stream command output without withholding normal interactive feedback.

**FR 5.4:** PHPX must support cancellation and signal forwarding on supported operating systems.

### 13.6 Tools

**FR 6.1:** `phpx tool run` must not modify project Composer dependencies.

**FR 6.2:** Tool package versions and runtime requirements must be visible.

**FR 6.3:** Cached tool environments must be content addressed or equivalently collision safe.

### 13.7 Automation

**FR 7.1:** Every primary command must support noninteractive operation.

**FR 7.2:** Every primary command must support structured JSON where meaningful.

**FR 7.3:** JSON schemas must be versioned before public stability is promised.

**FR 7.4:** Errors must produce stable nonzero exit codes.

## 14. Phase 1 Nonfunctional Requirements

### 14.1 Performance

1. The PHPX binary should start fast enough that wrapper use does not make common commands feel materially slower.
2. Warm discovery should reuse content validated metadata.
3. No daemon is required for Phase 1.
4. Repeated commands must not scan unrelated parent directories or entire home directories.
5. Downloads and tool environments should use deduplicated caches where safe.

Numeric thresholds must be established from baseline measurements before the public beta.

### 14.2 Reliability

1. Interrupted downloads and installations must recover safely.
2. Partial provider mutations must be diagnosed on the next run.
3. Local PHPX state must use atomic writes.
4. A failed command must not be reported as a successful synchronization.
5. Provider output changes must be covered by fixtures or version checks.

### 14.3 Portability

The architecture must support macOS, Linux, and Windows.

The technical proof may begin on macOS arm64, followed by Linux continuous integration. Public cross platform claims require real conformance testing on each claimed platform.

Identical user facing commands do not require identical operating system internals.

### 14.4 Maintainability

1. Core project discovery must remain separate from framework adapters.
2. Resolution must remain separate from provider mutation.
3. Human rendering must remain separate from structured results.
4. Providers must pass shared contract tests.
5. Composer behavior must not be reimplemented casually in Go.

## 15. Phase 1 Scope

### 15.1 Technical Proof Scope

1. Go command line application.
2. Project root discovery.
3. Composer metadata inspection.
4. Laravel application and package detection.
5. PHP and extension requirement planning.
6. `doctor` and `plan` human output.
7. Versioned JSON output.
8. One host execution provider.
9. One isolated execution provider.
10. Managed or selected official Composer execution.
11. `sync` for a locked project.
12. `run` with exit code propagation.
13. Representative fixtures.

### 15.2 Public Phase 1 Scope

1. The five primary commands.
2. Laravel first workflows.
3. Generic Composer workflows.
4. Initial Bedrock WordPress support.
5. Initial Symfony support.
6. Isolated tool execution.
7. Provider conformance tests.
8. Continuous integration guidance.
9. Secure update and artifact metadata.
10. Published compatibility and performance results.

### 15.3 Explicitly Outside Phase 1

1. A PHPX web server.
2. A PHPX DNS service.
3. PHPX managed TLS certificates.
4. PHPX managed databases, cache, mail, search, or object storage.
5. A persistent PHPX daemon.
6. A graphical interface.
7. First party queue and scheduler supervision.
8. Deployment and rollback.
9. Mandatory PHPX project configuration.
10. A native Composer replacement.
11. Complete Windows local environment ownership.

These are deferred product capabilities, not rejected ambitions.

## 16. Phase 1 Acceptance Criteria

### 16.1 Technical Proof

* [ ] A Laravel repository can be inspected without executing project code.
* [ ] PHPX explains the PHP requirement and required extensions with source files.
* [ ] `phpx plan` reports the provider, runtime, Composer, and dependency actions before mutation.
* [ ] The same Laravel test command runs through a host provider and an isolated provider.
* [ ] The official Composer implementation performs dependency installation.
* [ ] Child process exit codes are preserved.
* [ ] JSON output is deterministic for the same project and provider state.
* [ ] Secrets are redacted from diagnostics.
* [ ] Provider contract tests pass for both proof providers.

### 16.2 Public Beta

* [ ] Representative Laravel applications and packages complete `doctor`, `sync`, and `run` workflows.
* [ ] A generic Composer library completes the same core workflow.
* [ ] A Bedrock WordPress project completes the Composer workflow and can invoke WP CLI.
* [ ] A Symfony project completes the Composer workflow and can invoke its console.
* [ ] `phpx tool run` executes at least three representative tools without modifying project dependencies.
* [ ] Noninteractive and JSON workflows are documented and tested.
* [ ] Offline and frozen failure behavior is tested.
* [ ] Installation artifacts include checksums and release provenance.
* [ ] Published benchmarks compare direct commands with PHPX warm and cold command overhead.

### 16.3 Adoption Gate for Phase 2

Phase 2 implementation should begin only after Phase 1 demonstrates:

1. Repeated use by developers across multiple projects.
2. Demand for one first party provider rather than only a universal wrapper.
3. At least fifteen independent developers or five teams identifying local environment ownership as the next valuable expansion.
4. Evidence that provider fragmentation creates meaningful ongoing friction.
5. A viable, supportable runtime artifact strategy.
6. A provider model stable enough that the first party backend does not require rewriting Phase 1 commands.

## 17. Phase 2 Product Definition

### 17.1 Goal

Phase 2 makes PHPX a complete local PHP development environment.

The target experience is:

```console
phpx install
cd <project>
phpx sync
phpx up
phpx open
```

After PHPX installation, a supported project should not require Herd, Valet, Yerd, Lerd, DDEV, Sail, Local, ServBay, Laragon, a separate database application, a separate certificate application, or manual web server configuration.

### 17.2 Phase 2 Ownership

PHPX owns:

1. PHP runtime acquisition and selection.
2. Runtime artifact verification and updates.
3. PHP configuration overlays.
4. Extension installation and activation.
5. Local site registration.
6. Web routing and static file behavior.
7. PHP process lifecycle.
8. Local DNS or hosts integration.
9. Local certificate authority and trusted certificates.
10. Shared service lifecycle.
11. Project data isolation.
12. Database import, export, snapshot, and restore.
13. Framework process supervision.
14. Local logs and diagnostics.
15. Multi project state.
16. The local daemon API.
17. The graphical interface.
18. Project environment locking.

Composer continues to own PHP package dependencies.

### 17.3 Replacement Standard

Phase 2 should be evaluated as a replacement, not merely another option.

For supported workflows, PHPX must match or exceed the important capabilities users rely on in:

1. Herd and Valet for native speed, site linking, PHP selection, domains, and TLS.
2. Yerd for open source native services and efficient local operation.
3. Lerd for shared infrastructure, broad framework support, and committed setup.
4. DDEV for reproducibility, database workflows, project configuration, and cross platform team use.
5. Sail for Laravel supplied service conventions.
6. Local for approachable WordPress site and database workflows.

PHPX does not need identical implementation internals. It must satisfy the underlying jobs with a more coherent interface.

## 18. Phase 2 Architecture

### 18.1 Components

Phase 2 consists of:

1. The existing Go command line.
2. A user scoped Go daemon.
3. A versioned local API.
4. A first party local environment provider.
5. Managed runtime and service artifacts.
6. A shared routing layer.
7. Framework adapters.
8. An optional graphical client.

The command line must remain usable if the graphical client is not installed.

### 18.2 Daemon Responsibilities

The daemon may own:

1. Registered project state.
2. Routing state.
3. PHP worker and pool lifecycle.
4. Shared service lifecycle.
5. Process supervision.
6. Certificate renewal.
7. Log collection.
8. Health monitoring.
9. Local API requests.

The daemon must run without persistent root privileges. Privileged system changes must use a minimal, audited helper or an explicit one time setup action.

### 18.3 Runtime Distribution

The first party provider must support:

1. Multiple PHP minor versions side by side.
2. Exact patch identities when artifacts permit.
3. Per project PHP selection.
4. Command line and web runtime agreement.
5. Verified artifacts and atomic updates.
6. Rollback after a broken update.
7. Architecture specific builds.
8. Clear extension ABI compatibility.

The runtime strategy may use PHPX built artifacts, verified upstream artifacts, or a combination. No artifact source may be treated as permanent until licensing, provenance, update cadence, and security response are validated.

### 18.4 Extension Management

Phase 2 must coordinate:

1. Bundled extensions.
2. PIE installed extensions.
3. Verified prebuilt extension artifacts.
4. Runtime specific extension configuration.
5. Project extension profiles.
6. Upgrade and rollback.

Extension activation must be isolated by runtime version and must not mutate unrelated system PHP installations.

### 18.5 Web Server and PHP Execution

The local provider must support:

1. Framework front controllers.
2. Static file delivery.
3. Configurable document roots.
4. PHP FPM or an equivalent supported execution adapter.
5. Per project PHP selection.
6. Request logs and PHP error logs.
7. Websocket and development server proxying.
8. Custom ports and hostnames.
9. Project activation on demand.

The exact server implementation is a Phase 2 architecture decision. PHP FPM behind a Go controlled proxy and FrankenPHP are candidates. Compatibility, memory behavior, Windows support, and extension support must determine the choice.

### 18.6 DNS and Domains

PHPX should use `.test` as the default local top level domain.

The platform must support:

1. Parked project directories.
2. Explicit project links.
3. Aliases.
4. Custom local domains.
5. Operating system appropriate DNS integration.
6. A hosts file fallback when necessary.
7. Conflict detection with Herd, Valet, DDEV, Yerd, Lerd, VPN software, and system resolvers.

### 18.7 Trusted TLS

PHPX must provide:

1. A local certificate authority.
2. Explicit trust installation.
3. Per domain certificate issuance.
4. Certificate renewal.
5. Certificate removal.
6. Keychain or trust store integration per operating system.
7. Clear warnings before system trust changes.
8. No transmission of private local certificate keys.

### 18.8 Shared Services

The target service catalog includes:

1. MySQL.
2. MariaDB.
3. PostgreSQL.
4. Redis compatible cache.
5. Mail capture.
6. Meilisearch or another declared search provider.
7. S3 compatible object storage.
8. Optional project specific custom services.

Compatible projects should share service engines while receiving isolated databases, users, credentials, namespaces, ports, and storage paths.

Projects requiring incompatible versions or stronger isolation may receive a separate service instance or an explicit container fallback.

### 18.9 Database Workflows

PHPX must support:

1. Create and drop project databases.
2. Project scoped users and credentials.
3. Import and export.
4. Compressed backups.
5. Snapshots and restore.
6. Database version visibility.
7. WordPress URL search and replace through WP CLI.
8. Laravel migration commands through Artisan.
9. Safety confirmation for destructive operations.
10. Clear local and remote environment boundaries.

### 18.10 Process Supervision

Phase 2 must supervise declared development processes such as:

1. Laravel queues.
2. Laravel scheduler workers.
3. Horizon.
4. Reverb.
5. Vite and other frontend development servers.
6. Octane when explicitly supported.
7. Symfony workers.
8. WordPress frontend build processes.
9. User declared commands.

Processes must support logs, restart policy, health, resource visibility, and clean shutdown.

### 18.11 Multi Project Portfolio

The founding scale target remains:

1. At least fifty registered projects.
2. At least fifteen locally addressable projects during an active workday.
3. No requirement for fifteen complete web and database container stacks.
4. Demand driven PHP workers where practical.
5. Shared compatible services.
6. Fast project switching.
7. Clear per project resource visibility.

Performance claims require published comparisons using equivalent Laravel, WordPress, Symfony, and generic fixtures.

### 18.12 Graphical Interface

The Phase 2 graphical interface must provide:

1. Project registration and status.
2. Start, stop, open, and remove actions.
3. PHP and extension selection.
4. Service management.
5. Database workflows.
6. Process and log views.
7. Domain and certificate management.
8. Diagnostics and repair plans.
9. Update management.
10. Import from recognized environment configurations.

The graphical interface must be a client of the same local API used by the command line. It must not introduce a second control path or hide core functionality behind a paid tier.

### 18.13 Project Contract

Phase 2 may make a committed `phpx.toml` the preferred environment contract after Phase 1 validates the required fields.

The contract may declare:

1. PHP policy.
2. Extension policy.
3. Local domain and TLS intent.
4. Document root.
5. Services and versions.
6. Database initialization.
7. Development processes.
8. Framework adapter settings.
9. Environment variable names without committed secrets.
10. Provider compatibility policy.

`phpx.lock` may lock exact runtime, extension, service, Composer, and tool artifacts without duplicating `composer.lock`.

### 18.14 Migration and Coexistence

PHPX must detect and safely coexist with existing tools.

Migration assistance should read recognized configuration from:

1. Herd.
2. Valet.
3. Yerd.
4. Lerd.
5. DDEV.
6. Sail.
7. Local where accessible through supported exports.
8. Common Docker Compose environments.

Import must produce a reviewable plan. PHPX must not disable or uninstall another environment product without explicit permission.

## 19. Phase 2 Commands

Phase 2 extends the command line without changing the Phase 1 meanings.

```console
phpx up
phpx down
phpx restart
phpx status
phpx open
phpx sites
phpx site add
phpx site remove
phpx php list
phpx php install
phpx php use
phpx extensions
phpx services
phpx service start
phpx service stop
phpx db create
phpx db import
phpx db export
phpx db snapshot
phpx db restore
phpx cert trust
phpx cert secure
phpx cert unsecure
phpx logs
phpx processes
phpx daemon status
phpx gui
```

Command naming must remain coherent and scriptable. New commands require structured output and noninteractive behavior before they are considered complete.

## 20. Phase 2 Acceptance Criteria

### 20.1 Complete Local Environment

* [ ] A clean supported machine can install PHPX without an existing PHP runtime.
* [ ] A current Laravel fixture can synchronize, serve through trusted TLS, connect to a managed database, run a queue, and execute tests.
* [ ] A Bedrock WordPress fixture can synchronize, serve through trusted TLS, import a database, run WP CLI, and capture mail.
* [ ] A Symfony fixture can synchronize, serve, run console commands, and supervise a worker.
* [ ] A generic PHP fixture can declare a document root and run locally.
* [ ] Command line PHP and web PHP use the selected compatible runtime.
* [ ] The graphical interface and command line report the same project state.

### 20.2 Portfolio Scale

* [ ] Fifty projects can remain registered without one persistent application stack per project.
* [ ] Fifteen compatible sites can remain addressable using shared infrastructure.
* [ ] Idle and active resource use is published against equivalent Herd or Yerd and DDEV or Lerd workflows.
* [ ] Project activation and first request latency are measured on macOS, Linux, and Windows where supported.
* [ ] A project can be removed without affecting unrelated project data.

### 20.3 Reproducibility and Safety

* [ ] A committed PHPX contract can prepare equivalent supported local environments.
* [ ] Runtime, extension, Composer, and service artifacts can be locked and verified.
* [ ] Secrets remain outside committed configuration.
* [ ] Destructive database operations require explicit project and action confirmation.
* [ ] PHPX can coexist with another local environment without silently taking over its domains, ports, or services.

### 20.4 Replacement Validation

* [ ] Users can migrate representative Herd, DDEV, Lerd, and manual projects through reviewable plans.
* [ ] At least one Laravel team and one WordPress team can use PHPX as their only supported local environment during an extended pilot.
* [ ] The command line covers every core graphical action.
* [ ] No essential local capability requires a proprietary paid tier.

## 21. Phase 3 Production Delivery

### 21.1 Goal

Phase 3 carries the same project interpretation into build and deployment while preserving clear differences between local development and production infrastructure.

The target interface may include:

```console
phpx deploy plan production
phpx deploy production
phpx releases production
phpx rollback production <release>
phpx doctor production
```

### 21.2 Production Provider Model

PHPX should support providers for:

1. Managed Laravel platforms.
2. Generic Linux servers over a constrained deployment protocol.
3. OCI image based platforms.
4. Self hosted application platforms.
5. Static build outputs where applicable.

The provider model should allow PHPX to become the one user interface without requiring PHPX to own every server or cloud account.

### 21.3 Production Boundaries

PHPX must not assume that local shared services are appropriate production architecture.

Production delivery must define:

1. Immutable build inputs.
2. Runtime and extension requirements.
3. Composer install policy.
4. Asset build policy.
5. Secrets injection.
6. Database migration policy.
7. Queue and scheduler processes.
8. Health checks.
9. Release activation.
10. Rollback behavior.
11. Audit events.

### 21.4 Phase 3 Entry Gate

Phase 3 implementation should begin only when:

1. Phase 2 has a stable project contract.
2. Local and continuous integration execution are trusted.
3. Build outputs can be reproduced and verified.
4. At least two production providers can share a meaningful deployment model.
5. Security review covers secrets, remote execution, artifact provenance, rollback, and database migrations.

## 22. Technical Architecture

### 22.1 Go Module Boundaries

The initial codebase should separate:

1. `discovery`, established file and project type inspection.
2. `model`, normalized project, requirement, plan, and result types.
3. `resolver`, PHP, extension, Composer, and provider decisions.
4. `providers`, provider contracts and implementations.
5. `composer`, official Composer acquisition and invocation.
6. `executor`, argument safe process execution.
7. `tools`, isolated tool environments.
8. `cache`, verified artifacts and metadata.
9. `output`, human and structured rendering.
10. `config`, optional PHPX policy and local state.
11. `frameworks`, Laravel, WordPress, Symfony, and generic adapters.
12. `security`, checksums, provenance, redaction, and trust policy.

Phase 2 may add:

1. `daemon`.
2. `server`.
3. `dns`.
4. `certs`.
5. `services`.
6. `processes`.
7. `sites`.
8. `api`.
9. `gui` client contracts.

### 22.2 Dependency Direction

The normalized model must not import provider implementations or framework adapters.

Discovery produces facts. Resolution turns facts and policy into a plan. Providers execute approved plan steps. Output renders results.

Framework adapters may add facts and convenience commands, but they must not bypass the core resolver or executor.

### 22.3 State Locations

PHPX must separate:

1. Committed project intent.
2. Committed resolution locks.
3. Uncommitted project local state.
4. User preferences.
5. Global verified cache.
6. Phase 2 daemon state.
7. Logs and temporary process state.

The exact operating system paths must follow platform conventions and remain inspectable through a PHPX command.

### 22.4 Versioned Interfaces

The following interfaces require explicit versions before stability promises:

1. Structured command output.
2. Provider protocol.
3. Project configuration schema.
4. PHPX lock schema.
5. Phase 2 local API.
6. Phase 3 deployment provider protocol.

## 23. Test and Benchmark Strategy

### 23.1 Fixture Matrix

The fixture suite should include:

1. Current supported Laravel applications.
2. A supported older Laravel application with an older PHP constraint.
3. A Laravel package repository.
4. A generic Composer library.
5. A generic Composer application.
6. A Bedrock WordPress project.
7. A Symfony project.
8. A project with conflicting PHP declarations.
9. A project with required extensions.
10. A project containing Composer plugins and scripts.
11. A monorepo with multiple Composer roots.

### 23.2 Phase 1 Verification

1. Golden discovery fixtures.
2. Composer constraint fixtures.
3. Extension discovery fixtures.
4. Provider conformance tests.
5. Command argument and exit code tests.
6. Structured output schema tests.
7. Redaction tests.
8. Offline and partial download tests.
9. Cross platform path and signal tests.
10. End to end test commands through two provider topologies.

### 23.3 Phase 1 Benchmarks

Measure:

1. Cold and warm PHPX startup.
2. Cold and warm project discovery.
3. PHPX wrapper overhead compared with direct command execution.
4. Initial and repeated synchronization.
5. Isolated tool cold and warm execution.
6. Cache size and deduplication.

### 23.4 Phase 2 Benchmarks

Measure equivalent fixtures against appropriate competitors:

1. Clean installation time.
2. Clone to working route time.
3. Cold and warm site activation.
4. First request and repeated request latency.
5. Idle memory and process count for one, ten, and fifteen addressable sites.
6. Active memory and CPU under representative requests.
7. Disk use.
8. PHP version switching.
9. Database import and export.
10. Framework queue and asset process startup.

No benchmark result may be claimed without published fixtures, machine details, commands, warmup policy, and raw measurements.

## 24. Milestones

### Milestone 0: Architecture and Risk Spikes

1. Confirm normalized plan model.
2. Prototype Composer metadata inspection without project execution.
3. Test PHP constraint resolution strategy.
4. Compare host provider options.
5. Prove one isolated provider adapter.
6. Define structured output conventions.

Exit condition: PHPX can explain a representative Laravel project and show a credible provider plan.

### Milestone 1A: Doctor and Plan

1. Implement project discovery.
2. Implement Laravel and generic Composer detection.
3. Implement requirement resolution.
4. Implement provider inspection.
5. Implement human and JSON output.

Exit condition: `phpx doctor` and `phpx plan` produce trusted, read only results on the fixture suite.

### Milestone 1B: Sync and Run

1. Implement official Composer acquisition and invocation.
2. Implement one host provider.
3. Implement one isolated provider.
4. Implement synchronization state transitions.
5. Implement command execution and exit propagation.

Exit condition: representative Laravel and generic projects synchronize and run tests through both provider topologies.

### Milestone 1C: Isolated Tools and Continuous Integration

1. Implement isolated tool environments.
2. Implement verified caches.
3. Add Bedrock WordPress and Symfony fixtures.
4. Add continuous integration examples.
5. Publish compatibility and performance results.

Exit condition: the complete Phase 1 command surface is ready for public beta.

### Milestone 1D: Phase 1 Adoption

1. Recruit Laravel application and package users.
2. Recruit Composer library maintainers.
3. Validate repeated multi project use.
4. Record provider gaps and requested first party capabilities.
5. Decide whether optional PHPX configuration and locking have earned inclusion.

Exit condition: the Phase 2 adoption gate is satisfied or the product remains focused on the execution layer.

### Milestone 2A: First Party Local Provider Proof

1. Establish secure PHP runtime artifacts.
2. Implement the user scoped daemon.
3. Implement site registration and local routing.
4. Implement PHP execution.
5. Implement local TLS.
6. Benchmark representative sites.

Exit condition: one Laravel and one WordPress fixture run through a PHPX owned local provider.

### Milestone 2B: Shared Services and Processes

1. Add database and cache service management.
2. Add mail capture.
3. Add project data isolation.
4. Add database workflows.
5. Add process supervision.
6. Validate portfolio scale.

Exit condition: PHPX supports complete daily local workflows without an external environment product.

### Milestone 2C: Graphical Interface and Migration

1. Stabilize the local API.
2. Build the graphical client.
3. Add import plans for established tools.
4. Add update, repair, and rollback workflows.
5. Run Laravel and WordPress team pilots.

Exit condition: the Phase 2 replacement criteria are satisfied.

### Milestone 3: Production Delivery

Milestone 3 begins only after its entry gate. It must receive a separate detailed specification before implementation.

## 25. Risks and Mitigations

### 25.1 PHPX Becomes Only a Wrapper

**Risk:** Provider delegation produces little value beyond shell aliases.

**Mitigation:** Own the normalized Composer aware plan, deterministic resolution, diagnostics, tool isolation, conformance suite, continuous integration interpretation, and stable execution interface.

### 25.2 Provider Drift

**Risk:** External provider commands and output change.

**Mitigation:** Prefer documented interfaces, pin supported versions, maintain provider fixtures, expose capabilities, and fail clearly when compatibility is unknown.

### 25.3 Configuration Fragmentation

**Risk:** PHPX adds another manifest before proving distinct intent.

**Mitigation:** Derive from Composer and established files first. Add optional configuration only for PHPX specific policy. Do not duplicate dependency or provider configuration unnecessarily.

### 25.4 Artifact Supply Chain Burden

**Risk:** Owning PHP and extension binaries becomes a permanent security and maintenance commitment.

**Mitigation:** Delay broad artifact ownership until Phase 2, validate sources, automate reproducible builds where possible, sign releases, publish provenance, and support rollback.

### 25.5 Composer Semantic Mismatch

**Risk:** PHPX approximates Composer behavior and breaks real projects.

**Mitigation:** Invoke official Composer, preserve arguments and environment, maintain plugin and script fixtures, and avoid native replacement in Phase 1.

### 25.6 Phase 2 Scope Leaks Into Phase 1

**Risk:** Proxy, certificate, service, GUI, or deployment work delays the entry wedge.

**Mitigation:** Enforce the Phase 1 outside scope list. Provider capabilities may be inspected or invoked, but PHPX does not own those systems until the Phase 2 gate.

### 25.7 Cross Platform Claims Exceed Reality

**Risk:** One interface is mistaken for identical operating system behavior.

**Mitigation:** Publish platform capability matrices, test each claimed platform, allow provider differences, and avoid unsupported parity claims.

### 25.8 First Party Platform Arrives Too Late

**Risk:** PHPX gains users but external providers absorb the execution layer.

**Mitigation:** Design the provider contract so the first party backend can begin immediately after the adoption gate. Retain the same commands and normalized model throughout the transition.

### 25.9 Open Source Sustainability

**Risk:** Runtime builds, services, operating system integrations, and security response exceed maintainer capacity.

**Mitigation:** Keep the core narrow during Phase 1, use a conformance driven provider ecosystem, automate releases, establish governance, and fund maintenance before expanding supported matrices.

## 26. Governance and Licensing

### 26.1 Core License

The recommended core license is Apache 2.0.

The core command line, provider contracts, first party local provider, daemon API, and essential graphical functionality should remain truly open source.

### 26.2 Contribution Model

The project should support contributions through:

1. Framework adapters.
2. Environment providers.
3. Production providers.
4. Runtime and extension build recipes.
5. Compatibility fixtures.
6. Documentation and migration guides.

Provider and adapter acceptance requires ownership, tests, security review, and a maintenance policy.

### 26.3 Sustainability

Possible funding paths include:

1. Sponsorship.
2. Paid support.
3. Hosted artifact mirrors.
4. Team policy and audit services.
5. Managed deployment services in Phase 3.

Core local capabilities must not be intentionally crippled to manufacture a paid replacement for essential open source functionality.

## 27. Open Decisions

The following decisions require spikes or user validation before implementation reaches the affected milestone:

1. Which host provider will power the Phase 1 technical proof?
2. How much Composer constraint logic should Go implement before delegating to Composer?
3. What exact data belongs in optional `phpx.toml` during Phase 1?
4. Should `phpx.lock` ship during Phase 1 or wait for Phase 2 artifact ownership?
5. Which PHP versions and Laravel versions form the initial support policy?
6. Which provider becomes the first isolated topology?
7. Which runtime artifact strategy can be sustained for Phase 2?
8. Should the Phase 2 web execution path use PHP FPM, FrankenPHP, or multiple adapters?
9. How should Windows DNS, trust, process, and runtime behavior differ from macOS and Linux?
10. Which graphical client technology best preserves local API and command line parity?
11. Which production providers justify Phase 3?

Each decision must be resolved before its dependent implementation begins. None of these questions block Milestone 1A discovery, doctor, and plan work.

## 28. Research Basis

This direction was validated against current primary documentation available on July 16, 2026.

The most important references are:

1. Composer defines PHP and `ext-*` requirements as platform packages that it validates but does not install. [Composer platform packages](https://getcomposer.org/doc/01-basic-usage.md#platform-packages)
2. Composer remains the established dependency manager and project package authority. [Composer introduction](https://getcomposer.org/doc/00-intro.md)
3. PIE is the official PHP extension installer and still depends on an available PHP runtime and platform prerequisites. [PIE documentation](https://php.github.io/pie/)
4. Herd provides command line automation and committed project configuration. [Herd command line](https://herd.laravel.com/docs/windows/advanced-usage/command-line) [Herd project configuration](https://herd.laravel.com/docs/macos/sites/herd-yaml)
5. DDEV provides mature project configuration and a Docker based cross platform architecture, with documented filesystem synchronization tradeoffs on macOS and Windows. [DDEV architecture](https://docs.ddev.com/en/stable/users/usage/architecture/) [DDEV performance](https://docs.ddev.com/en/stable/users/install/performance/)
6. Yerd provides an open source native local PHP environment with runtime, domain, TLS, service, Laravel, and WordPress capabilities. [Yerd introduction](https://yerd.app/guide/introduction) [Yerd PHP versions](https://yerd.app/guide/php-versions)
7. Lerd provides an open source shared Podman environment with committed project configuration and broad framework support. [Lerd comparison](https://lerd.sh/getting-started/comparison) [Lerd configuration](https://lerd.sh/configuration)
8. mise demonstrates exact cross platform tool artifact locking, checksums, and continuous integration synchronization. [mise lock files](https://mise.jdx.dev/dev-tools/mise-lock.html)
9. uv demonstrates an integrated install, sync, run, tool, runtime, and cache experience while preserving familiar Python workflows. [uv documentation](https://docs.astral.sh/uv/)
10. Vite demonstrates the value of entering through one acute workflow problem, preserving ecosystem integration, and expanding from a focused layer into shared infrastructure. [Why Vite](https://vite.dev/guide/why)

External capabilities and licensing can change. Competitor claims must be revalidated before implementing provider adapters, migration tools, comparison pages, or public positioning.

## 29. Final Product Boundary

The product direction is intentionally ambitious and intentionally sequenced.

Phase 1 establishes PHPX as the trusted way to understand, synchronize, and execute Composer projects.

Phase 2 uses that position to deliver a complete open source local PHP environment with first party runtimes, extensions, routing, certificates, services, processes, portfolio management, and a graphical interface.

Phase 3 carries the same contract into build, deployment, health verification, and rollback.

The long term destination is one vertically integrated PHP platform. The first indispensable job is one correctly executed command.
