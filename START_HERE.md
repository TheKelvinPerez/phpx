# Elefante Project Context

Last updated: July 17, 2026

## Start Here

This repository is the planning home for Elefante. The project was previously developed under a temporary placeholder name.

Before making product or implementation decisions, read these files in order:

1. `START_HERE.md`
2. `specs/elefante-project-toolchain.md`
3. `ELEFANTE_BRAND_SETUP.md`

The specification is the canonical product direction. This file preserves the founding context, the decisions made during product discovery, and the exact restart point.

## Current State

There is no product implementation yet. The repository currently contains product strategy, architecture, acceptance criteria, branding infrastructure, and restart context.

Current local project path:

`/Users/0xaquawolf/Projects/elefante`

Current working branch:

`agent/laravel-primary-target`

The GitHub remote still uses the original placeholder repository:

`git@github.com:TheKelvinPerez/phpx.git`

The local folder and current documentation have been renamed to Elefante. Do not rename or transfer the GitHub repository until that external migration is explicitly requested and the destination repository strategy is confirmed.

## Tmux Resume Setup

Tmux session:

`elefante`

Tmux window:

`elefante`

Session mark:

`Slot 6, Elefante, /Users/0xaquawolf/Projects/elefante`

The tmux sessionizer discovers the repository dynamically from the new folder.

Resume through the sessionizer:

```console
tmux-sessionizer switch /Users/0xaquawolf/Projects/elefante
```

Resume through the saved mark:

```console
tmux-session-marks go 6
```

The configured home row shortcut for slot 6 is `Alt+q`.

After entering the session, open `START_HERE.md` and continue from the Immediate Restart Checklist.

## The Product in One Sentence

Elefante is the local development runtime and control plane for PHP projects. Composer resolves PHP dependencies. Elefante understands the project, prepares the compatible runtime and services, isolates every active environment, and runs the application correctly.

Short description:

`The local development runtime for PHP.`

Initial product expression:

> Composer describes what a PHP project needs. Elefante makes the environment satisfy those requirements, then runs the project correctly.

## Why This Product Exists

The product comes from a real recurring workflow problem.

One dashboard repository can have four features in progress through four Git worktrees. Each worktree needs its own application environment, frontend process, database, logs, and runtime state. The worktrees may share one PostgreSQL server, but they must use four isolated databases. Ports must not conflict. Starting, finding, switching, repairing, and cleaning those environments should not require custom scripts and terminal memory.

The same problem becomes more chaotic across a large portfolio. A previous WordPress workflow had approximately fifteen projects open at once. Databases, local domains, processes, service versions, and terminal sessions became difficult to understand and clean up. Orphaned databases and broken local state were a normal consequence of the tooling, not an unusual mistake.

The current workflow uses tmux, worktrees, shell scripts, database cloning, dynamic port allocation, and custom automation to make parallel development usable. Elefante should productize that operating model.

This is not a small convenience wrapper. It is intended to demonstrate senior level product engineering across runtime management, process execution, operating system integration, developer experience, distribution, security, and brand ownership.

## The Core Product Decision

Composer remains the PHP package dependency authority.

Elefante does not replace Composer's solver, package graph, lock file, scripts, or ecosystem. Elefante invokes the official Composer implementation and owns the executable environment around it.

Elefante is responsible for:

1. Discovering the project and framework.
2. Reading PHP and extension requirements.
3. Selecting or acquiring a compatible PHP runtime.
4. Coordinating extensions.
5. Selecting or provisioning the local environment provider.
6. Invoking Composer with its normal semantics.
7. Provisioning application services.
8. Isolating projects, branches, and worktrees.
9. Allocating domains, ports, databases, storage, and processes.
10. Running commands inside the correct environment.
11. Explaining failures and repair plans.
12. Cleaning up resources Elefante created.

## What “The uv of PHP” Means

The uv comparison describes the experience, not a plan to replace Composer.

Elefante should feel like one fast, coherent tool that installs, synchronizes, runs, diagnoses, caches, and manages development environments. It extends beyond uv because a typical PHP application also needs a web server, database, cache, mail capture, frontend process, local domain, trusted TLS, queues, and framework specific commands.

The useful comparison is:

`uv style integration + Composer compatibility + Herd or DDEV environment ownership + worktree isolation`

Elefante can be described as the uv of PHP as long as the explanation immediately says that Composer remains the dependency authority.

## Comparable Tools in Other Ecosystems

No single existing tool is an exact equivalent. Elefante combines responsibilities that are fragmented elsewhere.

### Python

uv combines Python project setup, dependency workflows, tool execution, caching, and runtime management behind one fast command surface.

Elefante borrows the coherent experience, but delegates PHP package resolution to Composer and adds application runtime, services, routing, workspaces, and local infrastructure.

### JavaScript

The comparable experience is spread across Node version managers, PNPM, process runners, monorepo tools, local proxies, Docker Compose, and framework development servers.

Elefante aims to make those categories feel like one project aware runtime for PHP applications.

### Rust

Rust developers receive a coherent relationship between rustup, Cargo, toolchains, builds, tests, and executable tools.

Elefante wants a similarly trustworthy command surface while also managing the service infrastructure required by web applications.

### General Environment Tools

Nix, Devbox, devenv, mise, and container based development environments can prepare reproducible tools and services. Elefante is narrower in ecosystem scope and deeper in PHP semantics, framework workflows, local routing, databases, and worktree lifecycle.

### Existing PHP Tools

Composer owns dependencies.

Herd, Valet, Yerd, and ServBay own parts of the native local runtime experience.

DDEV, Sail, and Lerd own container or shared service environments.

Local provides an approachable WordPress environment.

ngrok, Cloudflare Tunnel, and Expose provide public tunnels.

Elefante's opportunity is one Composer aware project contract that can initially coordinate existing providers, then progressively own the complete supported local environment.

## Full Vision and MVP Sequence

The full vision includes replacing fragmented local development products for supported workflows. Herd, Valet, DDEV, Sail, Local, separate database applications, manual certificates, and custom process scripts should not remain permanent requirements.

That replacement goal is not being removed from the product. It is sequenced.

### Phase 1

Build the trusted project execution layer:

1. Discover a Composer project without executing it.
2. Understand PHP and extension requirements.
3. Inspect the current machine and available providers.
4. Produce explainable diagnostics and plans.
5. Synchronize through existing providers.
6. Run commands correctly.
7. Support isolated PHP tools.
8. Prove Laravel first workflows.
9. Preserve generic Composer and WordPress compatibility.
10. Provide stable structured output for automation.

Representative commands:

```console
elefante doctor
elefante plan
elefante sync
elefante run -- php artisan test
elefante tool run phpstan/phpstan -- analyse
```

### Phase 2

Vertically integrate the complete local environment:

1. Managed PHP runtimes and extensions.
2. Web routing and PHP processes.
3. Local domains and trusted TLS.
4. Shared databases, cache, mail, search, and object storage.
5. Project and worktree isolation.
6. Framework process supervision.
7. Database snapshots, cloning, import, export, and restore.
8. Laravel, WordPress, Symfony, and generic adapters.
9. Multi project portfolio management.
10. Graphical client with command line and API parity.
11. Migration from existing local environment tools.
12. Project aware sharing and tunnels.

Representative commands:

```console
elefante up
elefante open
elefante status
elefante workspaces
elefante services
elefante db snapshot
elefante share
```

### Phase 3

Carry the same project interpretation into builds and production delivery:

1. Reproducible build outputs.
2. Deployment plans.
3. Provider adapters.
4. Environment and secrets boundaries.
5. Migration orchestration.
6. Health checks.
7. Release history.
8. Rollback.

Production delivery is part of the long term vertical integration goal, but it is not the MVP.

## Founding Workflow Requirements

### Multiple Projects

Elefante must support many registered projects without running one complete stack per project. Compatible projects should share efficient infrastructure while preserving isolated project data.

The target is at least fifty registered projects and at least fifteen locally addressable projects during an active workday.

### Git Worktrees

Git worktrees are first class environments, not unusual edge cases.

For every active worktree, Elefante must understand:

1. Repository identity.
2. Worktree path.
3. Branch identity.
4. Environment identity.
5. Runtime and service requirements.
6. Assigned domains and ports.
7. Database and credentials.
8. Processes and logs.
9. Ownership of generated resources.

Four worktrees from one repository must be able to run simultaneously without manual conflict resolution.

### Ports and Domains

Elefante must allocate collision free ports for PHP, Vite, websockets, debuggers, tunnel clients, and auxiliary services.

People should normally use stable local domains instead of remembering ports. Port allocation still matters internally and must remain visible through status and diagnostics.

### Databases

Compatible environments may share one PostgreSQL, MySQL, or MariaDB engine. Every project or worktree receives its own isolated database, user, schema, credentials, and lifecycle.

Database cloning must be explicit and fast. A worktree should be able to clone a known local source database without naming collisions or manual SQL administration.

Elefante must know which environment owns every generated database. Removing one worktree must never delete another worktree's data.

### Processes

Queues, schedulers, Horizon, Reverb, Vite, frontend watchers, WordPress build tools, Symfony workers, and declared commands must be supervised per environment.

Status, logs, restart behavior, health, and shutdown must be visible and deterministic.

### Cleanup and Repair

Elefante must detect orphaned databases, ports, routes, processes, storage, and environment records.

Cleanup must begin with a reviewable plan. It must identify ownership and avoid removing unrelated resources.

### Tmux and Automation Workflows

Elefante should work naturally inside tmux and other terminals without requiring a specific terminal manager.

Human readable output is required for direct use. Stable structured output is required so scripts and parallel automation can inspect environments, allocate resources, start processes, and clean up safely.

Elefante can eventually replace much of the custom worktree setup currently encoded in shell scripts and tmux conventions.

## Tunneling Direction

Sharing a local application on a phone, with a collaborator, or with an external webhook should be a normal project command.

The first version should integrate existing tunnel providers rather than operate a global relay network immediately. Candidate providers include ngrok, Cloudflare Tunnel, and Expose.

Elefante should own:

1. Selecting the project and active workspace.
2. Resolving the correct local target.
3. Starting and stopping the tunnel process.
4. Displaying the public URL.
5. Showing status and logs.
6. Cleaning up tunnel state.

The long term vision may include an Elefante tunnel network. A Laravel based relay service could become a separate project once demand, authentication, abuse prevention, bandwidth policy, operations, and cost are understood.

The local command contract should remain stable whether the transport comes from an integration or a future first party network.

## Technical Direction

### Implementation Language

Go is the default implementation language.

The intended distribution is a single native binary with fast startup, straightforward cross compilation, structured concurrency, process control, and a future user scoped daemon.

Composer remains an external official executable during the initial phases. Elefante coordinates it rather than translating its dependency behavior into Go.

### Platform Order

1. Apple Silicon on macOS, beginning on an M1 Max.
2. Linux.
3. Windows.

The public architecture is cross platform from the start. Platform claims are made only after real conformance testing.

### Runtime Shape

The expected architecture grows into:

1. A Go command line.
2. A user scoped Go daemon.
3. A versioned local API.
4. Provider adapters.
5. A first party local environment provider.
6. Managed and verified artifacts.
7. A shared routing layer.
8. Framework adapters.
9. An optional graphical client using the same API.

The daemon must not require persistent root privileges.

## Product Principles

1. Composer remains canonical for dependencies.
2. Existing projects remain usable without Elefante.
3. Read only diagnosis and planning come before mutation.
4. Provider differences remain visible.
5. Every mutation has ownership and a cleanup path.
6. Structured output is a first class interface.
7. The command line, local API, and graphical client share one control path.
8. Native performance is preferred where safe.
9. Isolated fallbacks remain available where compatibility requires them.
10. The product must coexist safely with existing local tools during migration.
11. No essential local capability should require a proprietary paid tier.
12. Security, artifact provenance, signing, rollback, and minimal privileges are product requirements.

## Brand Decision

Public brand:

`Elefante`

Canonical namespace:

`elefantephp`

Domain:

`elefantephp.com`

The elephant connection comes directly from PHP's visual identity. Elefante reflects the founder's Spanish language connection while remaining recognizable and pronounceable internationally.

Use English for product concepts and interface labels. Examples include Projects, Workspaces, Environments, Services, Share, and Tunnels.

Use Elefante as the display name everywhere. Use `elefantephp` only when a unique infrastructure identifier is required.

## Brand Infrastructure Status

Confirmed as completed:

* Purchased `elefantephp.com` through Porkbun
* Enabled auto renewal
* Enabled the domain security lock
* Enabled WHOIS privacy
* Enabled Porkbun API access
* Created the GitHub organization `elefantephp`
* Created the npm organization `elefantephp`
* Reserved the npm scope `@elefantephp`
* Created the forwarding address `accounts@elefantephp.com`
* Forwarded `accounts@elefantephp.com` to `thekelvinperez@gmail.com`

The complete reservation checklist lives in `ELEFANTE_BRAND_SETUP.md`.

## Account Ownership Strategy

The canonical registration address is:

`accounts@elefantephp.com`

The domain address should remain the identity attached to external services because Elefante owns the domain and can change mailbox providers later.

The planned private inbox is:

`elefantephp@proton.me`

Once created and secured, Porkbun forwarding should change so `accounts@elefantephp.com` delivers to Proton instead of the personal Gmail account.

The Google strategy is:

1. Try to claim `elefantephp@gmail.com` defensively.
2. Use it only for Google services and YouTube.
3. If it is unavailable, create a Google Account using `accounts@elefantephp.com`.
4. Create the YouTube business channel as `Elefante`.
5. Claim `@elefantephp`.
6. Add the personal Google Account as a backup owner or manager.
7. Use account permissions instead of sharing passwords.

Every account should use a unique password, two factor authentication or a passkey, saved recovery codes, and a trusted backup owner when supported.

## Namespace Plan

Website:

`https://elefantephp.com`

GitHub organization:

`github.com/elefantephp`

Future canonical repository:

`github.com/elefantephp/elefante`

npm:

`@elefantephp`

X:

`@elefantephp`

YouTube:

`@elefantephp`

Docker Hub:

`elefantephp`

Canonical container image:

`ghcr.io/elefantephp/elefante`

Docker Hub mirror:

`docker.io/elefantephp/elefante`

Homebrew tap:

`elefantephp/homebrew-tap`

Go module:

`github.com/elefantephp/elefante`

Composer vendor:

`elefantephp/*`

Packagist does not reserve an empty vendor. Publish the first legitimate package when it exists. Do not publish a meaningless placeholder.

## Immediate Restart Checklist

Brand reservation comes before implementation.

1. Create and secure `elefantephp@proton.me`.
2. Redirect `accounts@elefantephp.com` to Proton.
3. Test the forward from an unrelated email account.
4. Claim `elefantephp@gmail.com` if available.
5. Create the YouTube channel `Elefante`.
6. Claim the YouTube handle `@elefantephp`.
7. Claim the X username `elefantephp`.
8. Claim the Docker ID `elefantephp`.
9. Harden GitHub and npm ownership and recovery.
10. Continue with LinkedIn and Bluesky after the core developer surfaces are protected.

After those reservations, return to implementation planning for the Phase 1 technical proof.

## First Implementation Planning Target

The first technical proof should be narrow enough to build but structurally honest about the full vision.

It should:

1. Run on Apple Silicon macOS.
2. Be implemented in Go.
3. Discover a Laravel Composer project without booting application code.
4. Explain PHP and extension requirements.
5. Inspect at least one host provider and one isolated provider.
6. Produce a deterministic plan.
7. Invoke official Composer.
8. Run a representative Artisan command and preserve its exit code.
9. Return human readable and structured output.
10. Establish interfaces that can later own runtimes, services, workspaces, and tunnels.

WordPress remains an important product target. Laravel is the primary Phase 1 proof because it is the founder's strongest current ecosystem and provides a demanding application workflow. Bedrock WordPress and generic Composer fixtures should remain in the conformance suite.

## Decisions That Remain Open

1. The first host provider adapter.
2. The first isolated provider adapter.
3. The runtime artifact source for the first party provider.
4. PHP FPM behind a Go controlled proxy versus FrankenPHP.
5. The exact initial configuration schema.
6. Whether `elefante.lock` earns inclusion during Phase 1.
7. The exact database cloning contract.
8. The first tunnel provider integration.
9. The boundary between the command line and the future daemon.
10. The timing of the GitHub repository rename and transfer.

These are implementation decisions, not reasons to weaken the product vision.

## Repository Rules for the Next Session

1. Treat Elefante as the current product name.
2. Do not reintroduce the retired placeholder name into current files.
3. Do not rewrite Git history solely to remove the old placeholder name.
4. Do not rename or transfer the GitHub repository without explicit approval.
5. Preserve Composer as the dependency authority.
6. Preserve the complete local platform vision.
7. Treat worktrees and multi project isolation as core requirements.
8. Treat the current phase boundaries as sequencing, not permanent product limits.
9. Keep the architecture cross platform while implementing Apple Silicon first.
10. Do not push or publish anything without explicit approval.

## What Was Last Completed

The latest work established the Elefante brand, purchased and secured the domain, reserved the GitHub and npm namespaces, configured the ownership email forward, created the brand reservation checklist, renamed the local project identity, expanded the specification with parallel workspace and tunneling requirements, and prepared this restart context.

No application code has been written yet.
