# PHPX: Laravel First PHP Toolchain

## Document Status

**Working product name:** PHPX

**Status:** Draft, Go architecture and Laravel primary target selected, product necessity validation required before implementation

**Created:** July 15, 2026

**Last revised:** July 16, 2026

**Document purpose:** Define a Laravel first, Composer compatible, Go based PHP toolchain that can install and select PHP runtimes, synchronize Laravel environments, run many local applications without one container stack per project, execute isolated tools, and provide local development services through one command line interface.

The name PHPX is provisional. Public naming, package namespace availability, domain availability, and trademark review remain open decisions.

## 1. Product Definition

### 1.1 Goal

Create one fast, dependable command line tool that can take a Laravel project from source checkout to a working local development environment. The first product wedge is a lean, open, native Laravel environment that unifies the useful parts of Herd, Valet, Sail, DDEV, PHP version managers, Composer setup, and local service orchestration. Composer remains the canonical PHP package manager.

### 1.2 Product Promise

From a clean machine, a Laravel developer should eventually be able to run:

```bash
git clone git@github.com:company/laravel-app.git
cd laravel-app
phpx up
```

PHPX should then:

1. Detect a Laravel project, followed by classic WordPress, Bedrock, Composer based WordPress, Symfony, or generic PHP projects as those adapters become available.
2. Determine the required PHP version from project configuration and Composer metadata.
3. Install the correct PHP runtime when necessary.
4. Enable the PHP extensions required by Composer and the supported Laravel baseline.
5. Install missing dynamic extensions through a compatible extension provider.
6. Install Composer when needed and invoke the project supplied Artisan command through the selected runtime.
7. Run `composer install` without altering Composer semantics.
8. Prepare a reviewable local environment, allocate a database when requested, and preserve existing project secrets.
9. Start demand driven PHP workers without creating a dedicated container stack.
10. Serve the project through a local domain with trusted TLS.
11. Start only the database, cache, queue, scheduler, mail, log, and asset processes declared by the project or selected development profile.
12. Report one clear success state or one actionable failure state.

### 1.3 Problem Statement

Modern PHP package management is standardized around Composer, but Laravel development environments are split across Herd, Valet, Sail, DDEV, system package managers, database applications, and project specific shell instructions. Native tools are fast but can drift across machines. Container tools are reproducible but often create a web and service stack for every running project. On macOS, container file sharing may also add latency or require a synchronization layer.

The original scale problem remains a hard product constraint even though Laravel is now the primary market. PHPX must support approximately 50 registered PHP projects with 10 to 15 applications addressable during an active workday. A local environment should not become slow merely because many known projects are registered or several routes remain available on an Apple Silicon workstation.

Developers often need separate tools or manual instructions for:

1. Installing PHP.
2. Switching between PHP versions.
3. Matching command line PHP with web server PHP.
4. Installing native extensions.
5. Managing Composer itself.
6. Isolating global PHP tools.
7. Running local domains and TLS.
8. Preparing Laravel environment files, application keys, databases, queues, caches, mail capture, logs, and asset processes without unsafe automatic mutation.
9. Running Artisan through the correct PHP runtime and project environment.
10. Choosing between SQLite, MariaDB, MySQL, and PostgreSQL without maintaining separate setup documents.
11. Keeping dozens of known applications available without dozens of persistent web and database stacks.
12. Matching local command line PHP, PHP FPM, Composer, Artisan, and continuous integration.
13. Reproducing the same environment in continuous integration.
14. Diagnosing differences between machines.

This fragmentation creates slow onboarding, environment drift, global dependency conflicts, heavy resource use, brittle setup documentation, and a gap between local and automated environments.

### 1.4 Opportunity

PHPX can become the compatibility focused control plane around the existing PHP ecosystem. Its first job is to deliver a complete Laravel development environment without making containers the default unit of isolation. It does not need to replace PHP, Laravel, Artisan, Composer, PIE, PHPUnit, Pest, PHPStan, Rector, Mago, MariaDB, MySQL, PostgreSQL, Redis, or Node tooling. It needs to make those components installable, selectable, reproducible, and understandable through one coherent interface.

The strongest initial value proposition is not merely faster Composer execution. It is the ability to clone a Laravel application, resolve its PHP environment, install dependencies, prepare declared services, serve trusted HTTPS, and run its development processes through one reproducible contract with a small shared native footprint.

### 1.5 Target Users

1. Laravel developers working across multiple applications and PHP versions.
2. Laravel teams that need repeatable local and continuous integration environments.
3. Laravel agencies and product teams that want native performance without machine specific setup drift.
4. Open source Laravel package maintainers who need contributors productive with minimal setup.
5. WordPress developers and agencies managing classic WordPress, Bedrock, and Composer based sites.
6. Symfony developers working across multiple applications and PHP versions.
7. Framework neutral PHP developers and library maintainers.
8. New PHP developers who do not yet have a configured local environment.
9. Continuous integration maintainers who need deterministic PHP environments.
10. Teams replacing machine specific setup documents with executable project configuration.

### 1.6 Framework Priority

Development and product validation must follow this order:

1. Laravel.
2. WordPress.
3. Symfony.
4. Generic PHP sites and libraries.

Laravel is the first product market, the first complete application adapter, and the standard that defines the initial developer experience. The architecture must remain capable of supporting the remaining PHP ecosystem without forcing Laravel assumptions into the core runtime, artifact, proxy, service, or process managers.

### 1.7 Founding Scale Requirement

PHPX must be designed around the following real workload:

1. At least 50 registered PHP projects on one machine.
2. At least 15 Laravel applications available concurrently without 15 web containers and 15 database containers.
3. One shared router, one shared local certificate authority, and shared database engines where compatible.
4. No dedicated process for a registered application that has not received traffic and has no declared background process.
5. PHP workers created on demand and retired after an idle interval.
6. Asset watchers, queue workers, scheduler workers, and similar processes started only when explicitly declared.
7. Native host filesystem access with no duplicate project tree and no synchronization daemon required for normal macOS operation.

### 1.8 Why PHPX Should Exist

PHPX is not justified by a lack of PHP tools. The ecosystem already contains strong tools that solve most individual parts of local development:

1. Composer manages PHP project dependencies and exposes PHP and extension constraints, but it does not install PHP runtimes, configure local HTTPS, or manage application services.
2. Laravel Herd ships a native Laravel environment with PHP, Nginx, DNS, Composer, and Node.js. Herd Pro adds services and a committed `herd.yml` team configuration.
3. Laravel Valet provides a very small native macOS web environment with parked directories, trusted domains, and PHP selection, but it intentionally does not provide the complete service environment of Sail.
4. Laravel Sail provides a portable Docker based Laravel environment with PHP, Composer, Artisan, Node, databases, caches, mail, and optional services.
5. DDEV provides mature project configuration, broad PHP application support, services, local routing, and database workflows through project scoped containers.
6. PHP and general runtime managers can select language versions, but they do not understand Laravel application setup, services, routes, or process lifecycles.

The possible gap is therefore not any single missing feature. The gap is the intersection of an open source, headless, reproducible environment contract and a lean native runtime model.

PHPX should exist only to provide this distinct combination:

1. One command line control plane that works without a graphical application or preinstalled PHP runtime.
2. One committed environment declaration and exact artifact lock that can be enforced on developer machines and in continuous integration.
3. Native host filesystem performance and shared services without one persistent application stack per project.
4. Demand driven PHP FPM workers and explicit background processes so registered applications are cheap while idle.
5. Composer and Artisan compatibility through direct execution rather than replacement or partial emulation.
6. An open provider architecture for PHP runtimes, services, operating systems, and later application adapters.
7. Clear explanations, artifact provenance, frozen mode, offline mode, and machine readable behavior suitable for teams and automation.
8. An optional container compatibility path for applications whose infrastructure cannot be represented safely in native mode.

PHPX is not a Composer replacement and should not be described as one. The useful comparison with uv is that PHPX manages the runtime and reproducible project environment around the established package manager. Composer remains the dependency authority.

For a developer who only needs a polished Laravel environment on one Mac, Herd may already be the correct answer. For a team that prioritizes container isolation and service parity, Sail or DDEV may already be the correct answer. PHPX earns a place only if developers value the combined open, headless, deterministic, native, and resource efficient contract enough to switch or contribute.

### 1.9 Necessity Test and Stop Conditions

Milestone 0 is a product validation gate, not automatic permission to build the entire roadmap. Before the public MVP expands, PHPX must prove:

1. At least fifteen independent Laravel developers or five Laravel teams report a repeated problem that is not adequately solved by their current Herd, Valet, Sail, DDEV, or runtime manager workflow.
2. The same `phpx.toml` and `phpx.lock` can prepare a supported macOS environment and validate the project in Linux continuous integration without separate setup logic.
3. A clean clone reaches a working Laravel route and test command with fewer undocumented machine assumptions than the comparison workflows.
4. A measured 10 and 15 application workload uses materially fewer persistent processes and less memory than equivalent project scoped container stacks.
5. Exact runtime and service artifact resolution provides team value beyond the version declarations already available in `herd.yml`, Sail Compose files, or DDEV configuration.
6. PHPX can preserve Composer and Artisan behavior instead of creating ecosystem incompatibility.
7. Runtime and service artifacts can be sourced, verified, licensed, updated, and removed safely on every claimed platform.

The project should stop, narrow its scope, or contribute to an existing tool when any of the following becomes true:

1. The product reduces to a free reimplementation of Herd without a stronger headless, automation, or reproducibility contract.
2. Developers do not value exact environment locking or shared native services enough to change their workflow.
3. Benchmarks show no material resource or startup improvement over the environments PHPX intends to replace.
4. Secure runtime distribution cannot be sustained without depending on opaque or unreliable artifacts.
5. Native service sharing causes more compatibility, security, or support problems than it solves.
6. An existing open project can satisfy the validated requirements more effectively through contribution than a new ecosystem tool can through competition.

## 2. Product Principles

### 2.1 Laravel First, Ecosystem Compatible

Current Laravel applications must work as first class projects with Composer, Artisan, supported database drivers, trusted local HTTPS, and explicit development process orchestration. WordPress, Symfony, and generic PHP support must build on the same runtime, artifact, service, and process foundations.

### 2.2 Composer Remains Canonical When Present

`composer.json` and `composer.lock` remain the source of truth for PHP package dependencies, package resolution, autoloading, Composer plugins, Composer scripts, and declared PHP platform requirements.

PHPX must never create a parallel PHP package ecosystem or require projects to migrate their package metadata.

### 2.3 Existing Projects Work Without Migration

A Laravel or Composer project should receive useful PHPX behavior without restructuring its source tree. Optional PHPX configuration should unlock stronger reproducibility, local serving, services, and process orchestration.

### 2.4 Projects Remain Usable Without PHPX

Removing PHPX from a machine must not make the project structurally unusable. A developer with a compatible PHP installation, Composer, web server, database, and required project services must still be able to use normal Laravel and Composer workflows.

### 2.5 Native by Default, Containers by Exception

The default Laravel path must use native PHP, native filesystem access, shared native services, and a shared proxy. Containers may be supported as an optional compatibility backend for projects that genuinely require operating system isolation or custom infrastructure.

### 2.6 Shared Services, Isolated Project Data

Compatible projects should share a MariaDB, MySQL, PostgreSQL, Redis, or mail service process while receiving separate databases, users, credentials, namespaces, and backup histories where the upstream service supports those boundaries. A project may request a dedicated service instance when version or isolation requirements demand it.

### 2.7 Compatibility Before Native Replacement

PHPX must delegate to Composer whenever native behavior cannot preserve Composer semantics. Unsupported behavior must trigger a transparent fallback, never an approximation that appears successful.

### 2.8 Go Owns the Control Plane

Go is the implementation language for project discovery, artifact management, version selection, caching, process supervision, proxying, diagnostics, and the command line experience.

PHP remains the correct runtime for Composer plugins, Composer callbacks, application scripts, and PHP tools.

### 2.9 Deterministic by Default

When a lock file is present, two supported machines with the same target should resolve the same PHP patch version, Composer version, extension providers, extension versions, and tool versions.

### 2.10 Safe Global Coexistence

PHPX must coexist with system PHP, Homebrew PHP, Herd, Valet, Docker, Mise, and other version managers. It must not silently replace, unlink, delete, or modify external installations.

### 2.11 Framework Neutral Core

The core must understand PHP runtimes, artifacts, services, and executable project environments independently of any framework. Laravel receives the first and deepest application adapter. WordPress, Symfony, and generic PHP adapters follow without duplicating the core lifecycle.

### 2.12 Portfolio Scale Is a Core Feature

Registering a site must be cheap. A stopped or idle site must not retain a dedicated web server, database server, container, filesystem mirror, or file synchronization process. PHPX must measure and publish resource behavior across realistic portfolios.

### 2.13 Headless First

Every essential capability must work from a terminal and in continuous integration without a graphical application. A graphical interface may be added later as a client of the same stable core.

### 2.14 Explain Every Decision

PHPX must be able to explain why it selected a PHP version, extension provider, Composer version, runtime profile, service version, or fallback path.

## 3. Ownership Boundaries

### 3.1 Composer Owns

1. PHP package declarations.
2. Package version resolution.
3. `composer.lock`.
4. Package installation semantics during the compatibility path.
5. Composer plugins.
6. Composer scripts and callbacks.
7. Autoload generation during the compatibility path.
8. Package repository authentication semantics.

### 3.2 PHPX Owns

1. PHP runtime acquisition and selection.
2. Runtime artifact verification.
3. PHP configuration overlays.
4. Extension installation coordination.
5. Laravel project detection, environment planning, and Artisan execution through the selected runtime.
6. Composer acquisition and runtime selection.
7. Shared database, cache, and mail service lifecycle with project data isolation.
8. Database import, export, snapshot, and restore coordination.
9. WordPress project detection and WP CLI acquisition when the WordPress adapter is installed.
10. Environment synchronization.
11. Isolated tool environments.
12. Command execution within the selected environment.
13. Local proxy, DNS, TLS, and PHP FPM supervision.
14. Local development service lifecycle.
15. Environment diagnostics.
16. PHPX configuration and environment locking.
17. Shared artifact caching.
18. Registered site state and demand driven activation.

### 3.3 External Components Retain Their Responsibilities

1. PHP executes PHP applications.
2. Laravel and project supplied Artisan commands retain their normal application behavior.
3. WordPress core, plugins, themes, and WP CLI retain their normal behavior when the WordPress adapter is installed.
4. PIE resolves and builds supported dynamic PHP extensions.
5. Mago may provide formatting, linting, and static analysis integrations.
6. Existing tools such as PHPUnit, Pest, PHPStan, and Rector continue to execute as their own packages.
7. MariaDB, MySQL, PostgreSQL, Redis, and similar services continue to use their upstream engines.
8. Framework command line tools retain framework specific behavior.

## 4. Ecosystem Context

PHPX enters an ecosystem with several valuable adjacent projects.

### 4.1 Laravel Herd and Laravel Valet

Herd and Valet are the primary Laravel experience benchmarks. They establish familiar concepts such as parked directories, linked applications, trusted local domains, PHP isolation, and native service management. PHPX should preserve the speed and low friction of that experience while adding an open, headless, team reproducible environment contract.

### 4.2 Composer

Composer is the established package manager and defines the compatibility contract that PHPX must preserve.

### 4.3 PIE

PIE is the official PHP extension installer and the successor to PECL. PHPX should integrate with PIE instead of inventing a second extension package format.

### 4.4 Mago

Mago provides Rust based formatting, linting, and static analysis. Because PHPX uses Go, managed executable integration is preferred over linking to internal Mago libraries. PHPX should not recreate those capabilities.

### 4.5 Laravel Sail and DDEV

Sail and DDEV are the primary reproducibility and resource comparison points. Their container based environments provide important service isolation and team consistency, but a project stack can carry persistent web, database, cache, and supporting processes. PHPX should preserve declarative configuration, local URLs, database workflows, service versions, and predictable onboarding while replacing the default per project container topology with shared native processes.

### 4.6 Yerd

Yerd already provides a Rust based local PHP environment with PHP version management, local domains, TLS, services, and Composer support. A technical and product comparison is required before PHPX builds its server layer. PHPX should reuse proven behavior and protocols where practical, but the Go core must not depend on unstable internal Rust library interfaces.

### 4.7 StaticPHP

StaticPHP provides an important possible source of portable PHP artifacts for early development. Artifact provenance, supported features, licenses, extension coverage, PHP FPM availability, and long term reliability must be reviewed before production dependence.

### 4.8 Libretto

Libretto explores Composer compatible package installation in Rust. Its compatibility boundaries, cache model, resolver, autoload generation, and project health should be evaluated before PHPX attempts any native Composer acceleration. Reuse would require a stable executable or language neutral protocol rather than direct library integration.

## 5. User Personas

### 5.1 Laravel Application Developer

Works across Laravel applications with different PHP versions, databases, queues, caches, mail, and frontend processes. Wants one command to reach a correct local environment without globally relinking PHP or keeping a container stack alive for every application.

### 5.2 Laravel Team Maintainer

Needs a reviewable environment contract for a Laravel team. Wants new contributors, local machines, and continuous integration to agree on PHP, Composer, extensions, services, URLs, and development processes.

### 5.3 WordPress Portfolio Developer

Maintains many classic WordPress, Bedrock, or Composer based sites. Needs the same lean shared infrastructure after the Laravel adapter establishes the core.

### 5.4 New Contributor

Clones an unfamiliar repository and wants to run tests quickly without following a long machine setup document.

### 5.5 Library Maintainer

Needs to execute a library test suite across multiple supported PHP versions and dependency configurations.

### 5.6 Team Maintainer

Needs one committed environment declaration that keeps every developer and continuous integration aligned.

### 5.7 Continuous Integration Maintainer

Needs deterministic, cacheable, noninteractive commands with stable exit codes and machine readable output.

### 5.8 Framework Developer

Wants framework aware conveniences without making the underlying tool exclusive to one framework.

## 6. Core User Journeys

### 6.1 Existing Laravel Application on a Clean Machine

```text
Clone a Laravel application
    ↓
Run phpx init or phpx up
    ↓
PHPX detects Laravel from Composer metadata, Artisan, bootstrap files, and the public front controller
    ↓
PHPX installs a compatible PHP runtime and Composer
    ↓
PHPX installs or validates extensions
    ↓
PHPX proposes a safe local environment plan and allocates configured services
    ↓
PHPX registers the local domain and trusted TLS
    ↓
Developer opens the application or runs phpx artisan
```

### 6.2 Laravel Portfolio Workday

```text
PHPX knows 50 registered PHP projects
    ↓
Developer opens or requests 15 Laravel applications during the day
    ↓
One shared proxy routes every domain
    ↓
Compatible applications reuse shared database, cache, and mail services with isolated project data
    ↓
PHP FPM workers start on request and retire after idle timeout
    ↓
Applications remain addressable without 15 persistent container stacks
```

### 6.3 Preparing the Laravel Environment

```text
Developer enters a Laravel project with .env.example and no local .env
    ↓
Run phpx init
    ↓
PHPX inspects Composer, Artisan, environment examples, database configuration, and declared processes
    ↓
PHPX proposes runtime, service, local URL, and environment changes without exposing secrets
    ↓
Developer reviews and writes phpx.toml
    ↓
PHPX creates only the approved local state and leaves existing environment files intact
```

### 6.4 Switching Between Projects

```text
Enter Project A requiring PHP 8.2
    ↓
Run phpx run php artisan test
    ↓
PHPX selects managed PHP 8.2 only for that process
    ↓
Enter Project B requiring PHP 8.5
    ↓
Run the same command
    ↓
PHPX selects managed PHP 8.5 without global relinking
```

### 6.5 Running Artisan

```text
Run phpx artisan migrate:status
    ↓
PHPX selects the application PHP runtime and project environment
    ↓
PHPX invokes the project supplied Artisan command
    ↓
The command operates on the intended application and database without a container shell
```

### 6.6 Running an Isolated Tool

```text
Run phpx tool run phpstan/phpstan analyse
    ↓
PHPX resolves a compatible isolated tool environment
    ↓
PHPX installs it once in the shared tool store
    ↓
PHPX runs it against the current project
    ↓
Project dependencies remain unchanged
```

### 6.7 Starting the Full Development Environment

```text
Run phpx up
    ↓
PHPX synchronizes runtime and dependencies
    ↓
PHPX starts or reuses compatible shared services
    ↓
PHPX starts PHP FPM and declared queue, scheduler, log, mail, and asset processes
    ↓
PHPX routes the configured local domain
    ↓
PHPX displays URLs, health, and process logs
```

### 6.8 Reproducing the Environment in Continuous Integration

```text
Install the PHPX binary
    ↓
Run phpx sync --frozen --no-interaction
    ↓
PHPX restores verified artifacts from cache when available
    ↓
PHPX fails if committed locks would change
    ↓
Run phpx run php artisan test
```

## 7. Success Criteria

### 7.1 Technical Proof Criteria

* [ ] A macOS arm64 machine with no usable PHP, Composer, web server, or local database can run a current Laravel fixture through PHPX.
* [ ] PHPX installs a managed PHP runtime without changing the system PHP installation.
* [ ] PHPX installs and invokes Composer through the managed runtime.
* [ ] PHPX invokes the project supplied Artisan command through the managed runtime.
* [ ] A current Laravel fixture loads through trusted local HTTPS and can access its configured database.
* [ ] Laravel front controller routing works through the default server adapter.
* [ ] A fresh fixture can prepare approved local environment values without overwriting an existing `.env` file or secret.
* [ ] `php artisan migrate` and `php artisan test` complete through PHPX.
* [ ] Repeating `phpx up` without project changes is idempotent.
* [ ] Interrupted runtime downloads cannot leave an installation marked complete.
* [ ] Every downloaded artifact is verified before execution.
* [ ] `phpx doctor` reports enough information to diagnose the tested failure cases.
* [ ] The architecture demonstrates 15 registered Laravel fixtures available through one shared proxy and one shared compatible database engine.

### 7.2 Laravel Public MVP Criteria

* [ ] macOS arm64 is supported and benchmarked on Apple Silicon.
* [ ] The Laravel versions declared by the support policy are covered by end to end fixtures.
* [ ] PHP version selection honors PHPX configuration, `.php-version`, and Composer requirements.
* [ ] PHPX manages supported PHP branches and exact locked patch versions.
* [ ] Required built in extensions are detected and validated.
* [ ] Supported dynamic extensions can be installed through PIE.
* [ ] `phpx artisan` runs against the detected application with the selected PHP runtime and project environment.
* [ ] Composer plugins and scripts behave exactly as they do through direct Composer invocation.
* [ ] Isolated Composer tool execution works without modifying the project.
* [ ] `phpx.lock` can be committed and enforced with frozen mode.
* [ ] Local applications resolve through configured `.test` domains and trusted TLS.
* [ ] SQLite works without a server process, and at least one shared SQL engine can host isolated databases and users for compatible projects.
* [ ] Database create, import, export, snapshot, restore, and remove operations are available.
* [ ] A shared Redis provider is available for projects that explicitly request cache, session, or queue support.
* [ ] A shared mail capture service can receive mail from multiple applications while preserving project identity.
* [ ] Laravel front controller routing and public static assets work without Apache or Nginx.
* [ ] Environment preparation, application key generation, and local service credentials require reviewable rules and never overwrite existing secrets silently.
* [ ] Queue workers, the scheduler, log viewing, and asset processes start only through explicit configuration or a selected development profile.
* [ ] Commands required by local automation and continuous integration support noninteractive execution.
* [ ] A Linux x86_64 continuous integration fixture can enforce the same committed PHPX lock for runtime, Composer, Artisan, and test commands without claiming full Linux local environment support.
* [ ] Machine readable output is available for status, resolution, and diagnostics.
* [ ] The project publishes clear security, support, and artifact provenance policies.
* [ ] Fifty projects can remain registered without one persistent process, container, or database engine per project.
* [ ] Fifteen Laravel applications can remain addressable concurrently without 15 web containers and 15 database containers.
* [ ] A published benchmark compares PHPX with Herd or Valet and with a container workflow such as Sail or DDEV on equivalent Laravel fixtures.

### 7.3 Portfolio Performance Criteria

* [ ] Registering an idle application adds no dedicated long running process.
* [ ] An idle application has no PHP worker after the configured idle timeout.
* [ ] Compatible Laravel applications reuse shared services rather than creating one engine process per project.
* [ ] All applications reuse one proxy, one DNS integration, one local certificate authority, and one mail capture service.
* [ ] Laravel code executes directly from the host filesystem with no duplicate synchronized project tree.
* [ ] Opening an idle registered application creates required PHP workers automatically.
* [ ] Closing the browser does not keep unnecessary PHP workers alive indefinitely.
* [ ] PHP command line and web requests use the same project PHP version.
* [ ] Local service ports never collide silently.
* [ ] `phpx status` accurately reports every registered application, active worker, managed service, and endpoint.
* [ ] The benchmark records idle memory, active memory, process count, time to first response, steady response time, and disk use.

### 7.4 Adoption Criteria

* [ ] Existing Laravel projects can try PHPX without restructuring the application.
* [ ] Existing Composer projects can try PHPX without converting dependency files.
* [ ] A common Laravel project can generate a reviewable PHPX configuration through `phpx init`.
* [ ] A project can stop using PHPX without reversing package metadata changes.
* [ ] Setup documentation for a representative Laravel application can be reduced to PHPX installation, initialization, and one startup command.
* [ ] The compatibility policy and fallback behavior are documented and tested.

## 8. Scope Boundaries

### 8.1 Technical Proof Scope

1. macOS arm64.
2. Go command line application.
3. Current Laravel project discovery.
4. Composer PHP requirement resolution.
5. Managed PHP installation from a vetted artifact source.
6. Managed Composer installation.
7. Managed Artisan execution from the project.
8. Shared Go proxy and trusted local TLS.
9. SQLite plus one shared SQL engine with isolated project databases.
10. Demand driven PHP FPM pools.
11. `init`, `up`, `down`, `status`, `artisan`, `sync`, `run`, `composer`, `db`, and `doctor` commands.
12. Artifact checksums and atomic installs.
13. Current Laravel application fixtures.
14. A 15 application scale fixture derived from the founding portfolio resource constraint.

### 8.2 Public MVP Scope

1. macOS arm64 for the complete local environment.
2. Linux x86_64 with glibc for frozen synchronization and Laravel test commands in continuous integration.
3. Supported Laravel application versions.
4. Project configuration and environment lock files.
5. PHP runtime lifecycle management.
6. Managed Composer and project supplied Artisan execution.
7. Shared proxy, DNS integration, TLS, SQL database, Redis, and mail capture providers.
8. Demand driven PHP FPM.
9. Laravel front controller and public asset behavior.
10. Standard extension validation.
11. PIE integration for a documented supported set.
12. Laravel environment preparation and database workflows.
13. Explicit queue, scheduler, log, mail, and frontend process profiles.
14. Isolated PHP tools.
15. Continuous integration support for package, Artisan, and tool commands.
16. Shared artifact cache.
17. Stable human and machine readable output.
18. Signed releases and a documented update path.
19. Published Laravel resource and workflow benchmarks.

### 8.3 Later Scope

1. Additional operating systems and architectures.
2. WordPress adapter for classic WordPress, Bedrock, and Composer based WordPress.
3. Symfony adapter.
4. Generic PHP site adapter.
5. Laravel Sail and DDEV migration assistance beyond basic discovery and coexistence.
6. Additional native service providers and dedicated service modes.
7. Managed JavaScript runtime acquisition if research proves it belongs inside PHPX.
8. WordPress multisite and Apache compatibility after the WordPress adapter ships.
9. Graphical clients.
10. Native locked Composer installation acceleration.
11. Shared remote caches and enterprise mirrors.

### 8.4 Explicitly Out of Scope

1. Reimplementing the PHP interpreter.
2. Creating a replacement for Packagist.
3. Creating a new PHP dependency manifest.
4. Breaking or partially emulating Composer plugins.
5. Reimplementing PostgreSQL, MySQL, Redis, or similar engines.
6. Replacing every PHP formatter, test runner, or analyzer.
7. Production deployment in the first public MVP.
8. Modifying operating system PHP installations without explicit user action.
9. Requiring a graphical interface.
10. Silently executing privileged operations.
11. Full compatibility with arbitrary Sail, DDEV, or Docker Compose overrides in the public MVP.
12. Replacing the selected JavaScript package manager or runtime during the first public MVP.
13. Starting Node, asset watchers, scheduler loops, or queue workers for every registered project by default.

## 9. Functional Requirements

### 9.1 Project Discovery

**FR 1.1:** PHPX must discover the current project from the current working directory or an explicit path.

**FR 1.2:** Discovery must recognize `phpx.toml`, `phpx.lock`, `composer.json`, `composer.lock`, `.php-version`, `artisan`, `bootstrap/app.php`, `public/index.php`, Laravel package metadata, common Sail files, `.ddev/config.yaml`, and the repository root.

**FR 1.3:** Directory traversal must stop at the selected project boundary and must not accidentally inherit configuration from an unrelated parent project.

**FR 1.4:** Commands must accept an explicit project path for automation.

**FR 1.5:** Running PHPX outside a project must either perform a valid global operation or return an actionable message.

**FR 1.6:** PHPX must expose the discovered project root through `phpx status` and machine readable output.

**FR 1.7:** PHPX must distinguish the repository root, Laravel application root, PHP working directory, public document root, storage directory, and writable bootstrap cache directory.

**FR 1.8:** PHPX must detect Laravel without executing project supplied PHP during discovery.

**FR 1.9:** PHPX must determine the installed Laravel framework version from Composer metadata without booting the application.

**FR 1.10:** A Laravel package repository without a complete application must be identified as a library project rather than incorrectly served as an application.

**FR 1.11:** Parked directory discovery must avoid descending into `vendor`, `node_modules`, `.git`, `storage`, backups, and known generated directories.

**FR 1.12:** The detected project type and supporting evidence must be available through `phpx explain project`.

### 9.2 PHP Requirement Resolution

**FR 2.1:** PHPX must read the root PHP constraint from `composer.json` when present.

**FR 2.2:** PHPX must account for locked package platform requirements when choosing an actual runtime.

**FR 2.3:** PHPX must distinguish Composer `config.platform.php` from the actual runtime requirement.

**FR 2.4:** PHPX must support Composer version constraint syntax needed for PHP runtime selection.

**FR 2.5:** Constraint behavior must be tested against the official Composer Semver behavior and fixture corpus.

**FR 2.6:** PHPX must reject contradictory PHP requirements with an explanation showing each source.

**FR 2.7:** PHPX must not automatically cross to a different PHP minor branch when the project has pinned a branch.

**FR 2.8:** PHPX may update to a newer patch within an allowed branch only when lock and command policy allow it.

**FR 2.9:** Laravel automatic detection must require credible Composer or source evidence. A manually selected Laravel adapter must not invent a framework or PHP version when that evidence is absent.

**FR 2.10:** Laravel compatibility and PHP security support must be reported separately. Compatibility with an end of life PHP version must never be presented as a security recommendation.

**FR 2.11:** PHPX must not infer application or package compatibility with a newer PHP version when Composer constraints and reliable metadata are unavailable.

**FR 2.12:** Legacy PHP selection must require an explicit project request or explicit command confirmation and must persist a visible warning state.

### 9.3 PHP Runtime Management

**FR 3.1:** `phpx php install <request>` must install a matching managed PHP runtime.

**FR 3.2:** `phpx php list` must distinguish managed, system, active, supported, security only, and end of life runtimes.

**FR 3.3:** `phpx php pin <request>` must record the project runtime request without silently modifying Composer requirements.

**FR 3.4:** `phpx php remove <version>` must refuse removal while the runtime is actively used unless the user explicitly confirms a forced operation.

**FR 3.5:** Runtime installations must use temporary directories followed by an atomic promotion into the managed store.

**FR 3.6:** Concurrent requests for the same runtime must coordinate through a file or process lock.

**FR 3.7:** The runtime store must support multiple patch versions and architectures without relinking the operating system PHP executable.

**FR 3.8:** A project may explicitly request managed PHP, system PHP, or prefer managed PHP with system fallback.

**FR 3.9:** System PHP discovery must never imply permission to modify that installation.

**FR 3.10:** PHPX must expose the exact selected executable and loaded configuration files.

**FR 3.11:** Laravel public MVP runtime artifacts must contain the PHP extensions in the documented Laravel baseline or report an unsupported baseline before serving the application.

**FR 3.12:** Legacy project runtimes may be installable for maintenance, but PHPX must distinguish installability from active security support.

### 9.4 Composer Management

**FR 4.1:** PHPX must download Composer from an approved source and verify its integrity when the project or requested command requires Composer.

**FR 4.2:** PHPX must support project locking of the Composer major, minor, or exact version.

**FR 4.3:** `phpx composer <arguments>` must execute Composer using the selected project PHP runtime.

**FR 4.4:** PHPX must preserve Composer exit codes, standard input, standard output, standard error, signals, terminal behavior, and interactive prompts.

**FR 4.5:** Composer authentication files and environment variables must follow Composer behavior without copying secrets into PHPX project configuration.

**FR 4.6:** Composer plugins, custom installers, scripts, and callbacks must execute through Composer itself during the compatibility path.

**FR 4.7:** PHPX must not silently add `--no-plugins`, `--no-scripts`, or ignored platform requirements.

**FR 4.8:** `phpx add` and `phpx remove` may provide convenience commands, but they must delegate package mutation to Composer.

**FR 4.9:** Direct Composer usage must remain possible for developers who prefer it.

**FR 4.10:** PHPX should offer an optional shell shim so direct `composer` selects the project runtime, but installing the shim must be explicit.

### 9.5 Environment Synchronization

**FR 5.1:** `phpx sync` must produce and display a deterministic execution plan.

**FR 5.2:** The plan must resolve the project adapter, runtime, runtime configuration, extensions, Composer, Artisan path, isolated tools required by configuration, database allocation, local route registration, declared services, declared processes, and package installation action.

**FR 5.3:** `phpx sync` must use `composer install` when `composer.lock` exists.

**FR 5.4:** `phpx sync` must not run `composer update` implicitly.

**FR 5.5:** `phpx sync --frozen` must fail when `phpx.lock` is absent, stale, incompatible with the current target, or would change.

**FR 5.6:** `phpx sync --offline` must refuse network access and explain every missing cached artifact.

**FR 5.7:** A successful synchronization must finish with Composer platform requirement validation when Composer metadata exists, followed by adapter specific validation.

**FR 5.8:** Repeated synchronization without input changes must be idempotent.

**FR 5.9:** Partial failure must preserve the last known valid environment.

**FR 5.10:** PHPX must distinguish resolution, download, installation, Composer, script, and validation failures.

**FR 5.11:** A Laravel project must synchronize through its existing Composer metadata without creating a second dependency manifest or changing constraints implicitly.

**FR 5.12:** `phpx up` must synchronize prerequisites before exposing the local route, but it must not update Laravel, project packages, database schema, or application code implicitly.

### 9.6 Extension Management

**FR 6.1:** PHPX must distinguish built in PHP extensions from separately installed dynamic extensions.

**FR 6.2:** PHPX must infer required extension names from Composer platform requirements and a versioned Laravel extension baseline.

**FR 6.3:** A broad runtime artifact may contain common built in extensions while project configuration controls which optional extensions are enabled.

**FR 6.4:** PHPX must integrate with PIE for supported dynamic extension installation.

**FR 6.5:** When one extension name has multiple possible providers, PHPX must request an explicit selection and lock it.

**FR 6.6:** Dynamic extension artifacts must be isolated by operating system, architecture, PHP version, PHP API, thread model, build flags, and extension version.

**FR 6.7:** PHPX must never load an extension built for an incompatible PHP ABI.

**FR 6.8:** PHPX must expose the active INI scan directory and every PHPX managed INI fragment.

**FR 6.9:** Development extensions such as Xdebug must support project scoped activation without modifying the base runtime.

**FR 6.10:** Compiling an extension must display required native build dependencies before requesting privileged installation.

**FR 6.11:** The Laravel baseline must distinguish required, recommended, and optional extensions.

**FR 6.12:** PHPX diagnostics and Laravel runtime inspection should agree on loaded extension visibility wherever the framework exposes equivalent checks.

### 9.7 Isolated Tool Management

**FR 7.1:** `phpx tool run <package>` must run a Composer distributed tool without adding it to the current project.

**FR 7.2:** Each tool must receive an isolated Composer home and dependency environment.

**FR 7.3:** Tools must share immutable downloaded artifacts where safe without sharing mutable vendor state.

**FR 7.4:** Tool runtime selection must satisfy both the tool and current project context.

**FR 7.5:** `phpx tool install`, `list`, `update`, and `remove` must manage persistent tool installations.

**FR 7.6:** A tool may expose one or more executable aliases through PHPX managed shims.

**FR 7.7:** Tool resolution must be lockable for continuous integration and team usage.

**FR 7.8:** Project installed binaries in `vendor/bin` must take precedence when the user explicitly requests the project tool.

### 9.7.1 Laravel and Artisan

**FR 7L.1:** PHPX must detect the project supplied `artisan` executable and must never download or substitute a different Artisan implementation.

**FR 7L.2:** `phpx artisan <arguments>` must execute Artisan with the selected PHP runtime, application root, local URL, and project service environment.

**FR 7L.3:** PHPX must preserve Artisan arguments, exit codes, standard streams, interactive prompts, signal behavior, and framework bootstrap behavior.

**FR 7L.4:** `phpx artisan` must fail clearly when discovery identifies a Laravel package rather than a complete application.

**FR 7L.5:** PHPX must not run migrations, seeders, key generation, storage linking, cache clearing, or other mutating Artisan commands implicitly during ordinary synchronization.

**FR 7L.6:** PHPX may offer named, reviewable setup actions that invoke standard Artisan commands after explicit approval.

**FR 7L.7:** Queue, scheduler, Reverb, Horizon, Octane, and similar framework processes must be represented as explicit project process declarations or supported profiles.

**FR 7L.8:** Laravel version support must be defined by tested application fixtures and published support policy, not only by package constraint resolution.

### 9.7.2 WordPress and WP CLI

**FR 7W.1:** PHPX must install a verified managed WP CLI release.

**FR 7W.2:** `phpx wp <arguments>` must execute WP CLI with the selected site PHP runtime, WordPress root, local URL, and project database environment.

**FR 7W.3:** PHPX must preserve WP CLI arguments, exit codes, standard streams, interactive prompts, package behavior, and configuration discovery.

**FR 7W.4:** PHPX must support classic WordPress and Bedrock path conventions.

**FR 7W.5:** WordPress multisite commands must not be exposed as fully supported until both subdirectory and subdomain routing fixtures pass.

**FR 7W.6:** PHPX must not update WordPress core, plugins, themes, or WP CLI packages implicitly during environment synchronization.

**FR 7W.7:** `phpx wp` must fail clearly when discovery identifies only a plugin or theme repository without a configured host WordPress installation.

**FR 7W.8:** PHPX must support an explicit WordPress path override for unusual layouts.

### 9.8 Command Execution

**FR 8.1:** `phpx run <command>` must execute inside the resolved project environment.

**FR 8.2:** The child path must place the selected PHP runtime, Composer executable, project `vendor/bin`, and PHPX tool shims in documented order. Adapter specific tools such as WP CLI may extend that path when installed.

**FR 8.3:** The child process must inherit the terminal, working directory, environment, signals, and exit status unless explicitly overridden.

**FR 8.4:** PHPX must set `PHP_BINARY` and related environment values consistently where required.

**FR 8.5:** PHPX must support `phpx run -- <command>` for commands whose arguments overlap with PHPX flags.

**FR 8.6:** `phpx shell` may open a subshell with the environment active, but core workflows must not require shell activation.

### 9.9 Configuration and Locking

**FR 9.1:** `phpx.toml` must describe only environment requirements not already owned by Composer.

**FR 9.2:** `phpx.lock` must be generated, deterministic, reviewable, and intended for version control.

**FR 9.3:** Lock data must support target specific artifacts without requiring separate lock files for each operating system.

**FR 9.4:** The lock must record exact versions, source identifiers, hashes, and compatibility metadata.

**FR 9.5:** PHPX must provide a schema version and explicit migration behavior.

**FR 9.6:** Unknown configuration fields must produce an error by default to prevent misspelled settings from being ignored.

**FR 9.7:** Secrets must not be stored in `phpx.toml` or `phpx.lock`.

**FR 9.8:** `phpx init` must inspect the project and propose configuration before writing it.

**FR 9.9:** Laravel database and service credentials generated by PHPX must live in uncommitted local state or an operating system credential store.

**FR 9.10:** PHPX must understand Laravel `.env` and `.env.example` conventions without committing local secrets.

**FR 9.11:** PHPX must never overwrite an existing Laravel `.env` file or replace an existing `APP_KEY` silently.

**FR 9.12:** PHPX may inject local values through the supervised process environment or write an approved local environment file, but the chosen strategy must be visible through `phpx explain environment`.

**FR 9.13:** Copying `.env.example`, generating `APP_KEY`, or changing `APP_URL` and service variables must require an explicit setup plan and must preserve a recoverable record of approved local changes.

### 9.10 Local Site Management

**FR 10.1:** The local server layer must support Laravel public document roots first, followed by WordPress, Symfony, and generic PHP adapters.

**FR 10.2:** Each site must use its selected PHP version for both command line and web execution.

**FR 10.3:** The preferred architecture is a Go reverse proxy forwarding to a site pool managed by the PHP FPM master for the selected PHP version.

**FR 10.4:** `phpx link` must register the current project explicitly.

**FR 10.5:** `phpx park <directory>` must discover eligible child projects without registering unrelated directories.

**FR 10.6:** `phpx secure` must issue a trusted local certificate after the local certificate authority is installed.

**FR 10.7:** Binding privileged ports, installing resolver configuration, and trusting a local certificate authority must occur through explicit setup actions.

**FR 10.8:** Normal site operation must not require repeated privilege elevation.

**FR 10.9:** Sites must bind to loopback by default.

**FR 10.10:** Port and socket allocation must be deterministic and collision aware.

**FR 10.11:** One PHP FPM master may serve multiple site pools using the same PHP runtime version.

**FR 10.12:** Laravel application pools must use the FPM `ondemand` process manager so idle pools can have zero child workers.

**FR 10.13:** PHPX must enforce a configurable global PHP worker limit and a configurable per site worker limit.

**FR 10.14:** An idle timeout must retire unused PHP workers without unregistering the site route.

**FR 10.15:** Laravel front controller routing, public static assets, and ordinary streamed responses must work through the default proxy adapter.

**FR 10.16:** The proxy must prevent accidental static exposure of `.env`, storage internals, source files, and other paths outside the selected public document root.

**FR 10.17:** Projects that depend on custom Nginx, Apache, Octane, or container behavior must receive a blocking compatibility explanation or select a documented backend when available.

**FR 10.18:** The shared proxy must wake or route an idle registered site without requiring a manual start command for ordinary web requests.

**FR 10.19:** Applications that explicitly request wildcard subdomains must be able to register matching local DNS and certificates before that behavior is declared supported.

**FR 10.20:** Site routes must remain cheap registry entries when no PHP worker or project process is active.

### 9.11 Development Services

**FR 11.1:** PHPX must support SQLite, at least one upstream shared SQL engine, Redis, and mail capture for the Laravel public MVP. Additional SQL engines, search, storage, and specialized services may follow.

**FR 11.2:** Every service must have an explicit version, data path, configuration path, port policy, and health check.

**FR 11.3:** Compatible Laravel projects should share a database or cache engine process while using separate databases, users, credentials, namespaces, and backup histories where supported.

**FR 11.4:** Services must bind to loopback by default.

**FR 11.5:** `phpx services up`, `down`, `status`, `logs`, and `remove` must expose lifecycle operations.

**FR 11.6:** Removal of service data must require explicit confirmation or a force flag.

**FR 11.7:** A container backend may be offered as an adapter, but containers must not be required for core runtime management.

**FR 11.8:** A project requiring a different database engine version may use another shared versioned engine instance.

**FR 11.9:** A project may request a dedicated database engine for compatibility or isolation, but dedicated mode is not the default.

**FR 11.10:** Stopping a site must not delete its database or uploads.

**FR 11.11:** The shared mail service must tag or filter messages by originating site.

**FR 11.12:** Redis and other optional services must not start merely because PHPX knows about a site.

### 9.11.1 Laravel Database and Storage Workflows

**FR 11L.1:** `phpx db create` must create a project SQLite database or allocate a project database and least privilege user in the selected shared engine.

**FR 11L.2:** `phpx db import` must support common plain and compressed SQL dump formats for documented server engines.

**FR 11L.3:** `phpx db export` must create a portable logical dump without stopping unrelated applications.

**FR 11L.4:** `phpx db snapshot` and `phpx db restore` must maintain project scoped local recovery points.

**FR 11L.5:** Database deletion must display the application, engine, database name, and backup state before confirmation.

**FR 11L.6:** Database migrations, seeders, pruning, and reset commands must be explicit Artisan operations and must not happen silently during import or synchronization.

**FR 11L.7:** PHPX must preserve Laravel storage paths and writable directory requirements without relocating project data silently.

**FR 11L.8:** Creating the public storage link must be an explicit setup action and must use Laravel behavior.

**FR 11L.9:** Database credentials must be unique per project by default.

**FR 11L.10:** Project database and Redis isolation are organizational and access boundaries, not substitutes for hostile code isolation.

### 9.12 Process Supervision

**FR 12.1:** `phpx up` must start declared project processes in dependency order.

**FR 12.2:** Processes must support health checks, restart policy, working directory, environment variables, and dependency declarations.

**FR 12.3:** PHPX must prefix or structure logs so concurrent process output remains understandable.

**FR 12.4:** Interruption must trigger graceful shutdown followed by bounded forced termination when required.

**FR 12.5:** `phpx status` must distinguish starting, healthy, unhealthy, stopped, and failed processes.

**FR 12.6:** An optional user scoped daemon may retain local site and service state, but core package and runtime commands must not require it.

**FR 12.7:** Registered applications must not automatically start asset watchers, Node processes, scheduler loops, queue workers, Horizon, Reverb, or Octane.

**FR 12.8:** A global scheduler must cap total PHP workers across every site and report queueing caused by the cap.

**FR 12.9:** PHPX must be able to sleep a site by stopping its optional project processes while leaving its route and data registered.

### 9.13 Diagnostics

**FR 13.1:** `phpx doctor` must inspect Laravel discovery, framework and application roots, public document root, runtime selection, artifact integrity, PHP configuration, extensions, Composer, Artisan, environment readiness, writable paths, application key state, database access, proxy state, DNS, TLS, services, ports, and process state as applicable.

**FR 13.2:** Every failed check must include evidence and a proposed next command.

**FR 13.3:** Doctor output must support human, JSON, and quiet status formats.

**FR 13.4:** Diagnostics must redact credentials, tokens, cookies, and sensitive environment values.

**FR 13.5:** `phpx explain` must describe resolution decisions without mutating the environment.

### 9.14 Continuous Integration

**FR 14.1:** Essential commands must support `--no-interaction`, `--frozen`, `--offline`, `--json`, and stable exit codes where applicable.

**FR 14.2:** PHPX must document cache directories and safe cache keys.

**FR 14.3:** A first party setup action or portable installer may be provided after the CLI contract stabilizes.

**FR 14.4:** Continuous integration must be able to verify locks without starting local site infrastructure.

**FR 14.5:** CI output must avoid animated terminal behavior when no interactive terminal is present.

### 9.15 Updates and Removal

**FR 15.1:** PHPX must support stable, preview, and development release channels.

**FR 15.2:** Self update must verify the new executable before replacement.

**FR 15.3:** Failed self update must preserve the previous executable.

**FR 15.4:** Runtime, extension, tool, cache, and service data removal must be separately addressable.

**FR 15.5:** PHPX must provide a dry run for cache cleanup and destructive removal.

### 9.16 Privacy

**FR 16.1:** Core operation must not require an account.

**FR 16.2:** Usage telemetry must be absent by default unless a later public decision explicitly introduces opt in telemetry.

**FR 16.3:** Crash reports must never upload automatically.

**FR 16.4:** Diagnostic bundles must be locally reviewable before sharing.

### 9.17 Existing Environment Import and Coexistence

**FR 17.1:** PHPX import commands must inspect common Laravel Sail and DDEV project configuration without modifying it.

**FR 17.2:** The importer must map project name, project type, document root, PHP version, database type and version, cache service, mail service, primary URL, additional hostnames, process commands, web server type, and common environment values where safe.

**FR 17.3:** Custom Docker Compose files, custom images, add ons, hooks, web build files, and custom Nginx or Apache configuration must be inventoried and classified as mapped, ignored by choice, or unsupported.

**FR 17.4:** Unsupported DDEV behavior must block automatic cutover when it may affect application behavior.

**FR 17.5:** PHPX must generate a reviewable proposed `phpx.toml` before writing it.

**FR 17.6:** An optional migration command may ask the existing environment to export the project database and then import it into PHPX local state.

**FR 17.7:** Changes from an existing local URL to a PHPX `.test` URL must require explicit approval and must not mutate application data automatically.

**FR 17.8:** PHPX must not stop, delete, or alter Sail, DDEV, or other existing project infrastructure until the user explicitly requests cleanup after validation.

**FR 17.9:** A project may retain both configurations during migration.

### 9.18 Portfolio Management and Resource Scheduling

**FR 18.1:** `phpx sites` must list at least 50 registered sites quickly without probing every project through PHP execution.

**FR 18.2:** Site status must distinguish registered, sleeping, warming, available, busy, unhealthy, and disabled.

**FR 18.3:** Site metadata must include project path, URL, adapter, PHP version, database engine, database name, last access time, active workers, and optional processes.

**FR 18.4:** PHPX must support site groups or tags for client, maintenance state, PHP version, and user defined organization.

**FR 18.5:** Bulk commands must require an explicit selection query and display affected sites before mutation.

**FR 18.6:** The scheduler must prioritize interactive web requests over optional background development processes by default.

**FR 18.7:** PHPX must expose global resource limits and current consumption through `phpx status`.

**FR 18.8:** An idle registered site must consume registry data only, apart from resources shared globally or by runtime version.

**FR 18.9:** Resource benchmarks must use the founding workload of 50 registered projects and 15 concurrently addressable Laravel applications.

### 9.19 Framework Rollout

**FR 19.1:** Laravel requirements and fixtures must reach public MVP quality before another application adapter becomes official.

**FR 19.2:** WordPress support must reuse runtime, Composer, database, proxy, TLS, process, and diagnostics foundations while adding classic project discovery and WP CLI behavior.

**FR 19.3:** Symfony support must follow the Laravel public MVP and reuse the same foundations.

**FR 19.4:** Generic PHP site support must allow an explicit document root and front controller without framework detection.

**FR 19.5:** Adding later adapters must not weaken Laravel workflow correctness or portfolio performance guarantees.

## 10. Command Line Contract

### 10.1 Initial Command Surface

```text
phpx init
phpx sync
phpx up
phpx down
phpx restart
phpx logs
phpx open
phpx run
phpx shell
phpx status
phpx explain
phpx doctor

phpx link
phpx unlink
phpx park
phpx unpark
phpx secure
phpx unsecure
phpx sites
phpx site sleep
phpx site wake

phpx artisan

phpx db create
phpx db import
phpx db export
phpx db snapshot
phpx db restore
phpx db shell
phpx db remove

phpx mail open
phpx mail status

phpx services up
phpx services down
phpx services status
phpx services logs
phpx services remove

phpx php install
phpx php pin
phpx php list
phpx php current
phpx php update
phpx php remove

phpx composer
phpx add
phpx remove

phpx ext add
phpx ext list
phpx ext enable
phpx ext disable
phpx ext remove

phpx tool run
phpx tool install
phpx tool list
phpx tool update
phpx tool remove

phpx cache status
phpx cache clean
phpx self update
```

### 10.2 Later Command Surface

```text
phpx import sail
phpx import ddev

phpx wp

phpx adapter wordpress
phpx adapter symfony
phpx adapter generic

phpx ddev compare
phpx ddev cleanup
```

### 10.3 Global Flags

```text
--project <path>
--config <path>
--json
--quiet
--verbose
--no-color
--no-interaction
--offline
--frozen
--dry-run
```

### 10.4 Exit Code Policy

1. `0` means the requested operation completed successfully.
2. `1` means a general operational failure.
3. `2` means invalid command or configuration usage.
4. Dedicated documented codes may represent frozen lock failure, network unavailability, integrity failure, platform incompatibility, and child process failure.
5. Child command execution should preserve the child exit code whenever doing so is unambiguous.

## 11. Proposed Project Configuration

The following example demonstrates the intended Laravel boundary. Composer requirements remain in `composer.json`. PHPX configuration describes the local and reproducible environment around Laravel and those package requirements.

```toml
schema = 1

[project]
type = "laravel"
root = "."
document_root = "public"

[php]
version = "8.4"
source = "managed"

[laravel]
artisan = "artisan"

[composer]
version = "2"

[extensions.dev]
xdebug = "^3.4"

[site]
domain = "example-app.test"
tls = true
front_controller = "public/index.php"

[database]
engine = "mariadb"
version = "10.11"
mode = "shared"

[workers]
mode = "ondemand"
max_children = 2
idle_timeout = "10s"

[processes.queue]
command = "php artisan queue:listen --tries=1"
autostart = false

[processes.scheduler]
command = "php artisan schedule:work"
autostart = false

[processes.assets]
command = "pnpm dev"
autostart = false

[profiles.dev]
processes = ["queue", "scheduler", "assets"]

[tools]
phpstan = "phpstan/phpstan:^2"
```

### 11.1 Configuration Rules

1. `php.version` expresses an allowed request, not necessarily an exact patch.
2. The resolved exact patch belongs in `phpx.lock`.
3. Composer `require.php` must also allow the selected runtime when Composer metadata exists.
4. The supported Laravel and Composer compatibility ranges must allow the selected runtime.
5. Composer `ext-*` requirements are inferred and must not need duplication.
6. A versioned Laravel extension baseline is inferred without duplication.
7. `extensions.dev` contains optional development extensions or explicit provider choices.
8. Shared database mode shares the engine process, never the project database or credentials.
9. Database credentials are generated into uncommitted local state.
10. Worker limits are project limits beneath a global resource ceiling.
11. Optional processes do not start unless `autostart` is true or the user requests them.
12. Tool aliases are local PHPX aliases and do not alter project packages.
13. Process commands execute through the selected project environment.

## 12. Resolution and Synchronization Algorithm

### 12.1 Resolution Precedence

For PHP runtime selection, PHPX should use the following precedence:

1. An exact compatible artifact from a valid frozen `phpx.lock`.
2. The runtime request in `phpx.toml`.
3. An existing `.php-version` file.
4. The root PHP requirement in `composer.json` when present.
5. The PHP compatibility range for the detected Laravel framework version when Composer metadata does not already provide an equivalent constraint.
6. The latest PHP branch that is supported by PHP and compatible with the detected application.

Higher precedence never permits violation of a lower level compatibility constraint. A pinned PHPX version that violates Composer or detected Laravel requirements must fail. PHPX must explain when a Laravel application remains compatible with a PHP branch that PHP itself no longer supports.

### 12.2 Synchronization Steps

1. Discover project files, application adapter, application root, public document root, and target platform.
2. Parse PHPX configuration and validate its schema.
3. Read Laravel and Composer metadata without executing project code.
4. Parse Composer platform requirements.
5. Load and validate the PHPX lock when present.
6. Resolve a supported and application compatible PHP runtime.
7. Resolve the runtime artifact and verify target compatibility.
8. Resolve built in and dynamic extension requirements.
9. Resolve Composer and the project supplied Artisan path.
10. Resolve configured isolated tools.
11. Resolve database engine allocation and project database identity.
12. Resolve local domain, TLS, front controller adapter, PHP FPM pool, and worker policy.
13. Produce an immutable execution plan.
14. Acquire per artifact and shared service locks.
15. Download missing artifacts into temporary storage.
16. Verify hashes, signatures, sizes, and manifests.
17. Atomically install artifacts into the managed store.
18. Generate project scoped PHP and PHP FPM configuration overlays.
19. Execute `composer install` through the selected runtime according to lock and command policy.
20. Execute Composer platform requirement validation.
21. Validate Artisan discovery, writable paths, approved environment readiness, and configured database connectivity without running application migrations implicitly.
22. Persist a new lock only when command policy allows changes.
23. Report the selected environment and next useful commands.

`phpx up` extends synchronization by ensuring shared services are healthy, registering the route, loading the PHP FPM pool, and starting only declared automatic processes.

### 12.3 Failure Guarantees

1. A failed download never replaces a valid artifact.
2. A failed extension installation never corrupts the base runtime.
3. A failed Composer operation preserves Composer output and exit status.
4. A failed lock update does not leave a partially written lock file.
5. A failed process startup shuts down only processes started for that operation.
6. Every failure identifies the stage and evidence.

## 13. Technical Architecture

### 13.1 High Level Components

```text
PHPX CLI
    ↓
Project Resolver
    ↓
Environment Planner
    ├── Application Adapter Registry
    ├── Laravel Adapter
    ├── PHP Runtime Manager
    ├── Composer Adapter
    ├── Artisan Runner
    ├── Extension Adapter
    ├── Database Allocator
    ├── Tool Manager
    ├── Artifact Store
    └── Lock Manager
    ↓
Command Runner

Laravel Local Control Plane
    ├── User Daemon
    ├── Go HTTP and TLS Proxy
    ├── DNS Integration
    ├── PHP FPM Supervisor
    ├── Shared Database Supervisor
    ├── Resource Scheduler
    ├── Site Registry
    ├── Shared Mail Capture
    ├── Service Supervisor
    └── Process Log Router
```

### 13.2 Proposed Go Module

```text
cmd/
    phpx/
internal/
    app/
    project/
    adapter/
    laravel/
    constraints/
    runtime/
    artifact/
    composer/
    extension/
    tool/
    database/
    ddevimport/
    scheduler/
    process/
    proxy/
    service/
    daemon/
    platform/
    testkit/
testdata/
    fixtures/
go.mod
go.sum
```

The first release should prefer one `phpx` executable that can run normal commands and launch its daemon mode. A separate daemon executable should be introduced only if operating system lifecycle requirements make it necessary.

### 13.3 Dependency Direction

1. Domain packages define shared types and errors without importing command line or daemon delivery packages.
2. Project, adapter, Laravel, constraint, runtime, artifact, Composer, extension, database, and tool packages depend inward on narrow domain contracts.
3. Laravel depends on stable adapter interfaces and must not place application assumptions inside runtime or artifact packages.
4. Proxy, database, scheduler, and service packages depend on runtime and process abstractions, not the command line parser.
5. The `cmd/phpx` package composes capabilities but contains minimal business logic.
6. The daemon exposes a versioned local protocol and uses the same core services as the CLI.
7. Interfaces should normally be declared by the consuming package and remain narrow enough to support deterministic fakes.
8. The `internal` boundary prevents application internals from becoming an accidental public Go API.
9. Platform specific implementations use focused files and build constraints behind shared interfaces.
10. Test support provides fixtures and fake artifact repositories without entering production dependencies.

### 13.4 Managed Store Layout

```text
~/.local/share/phpx/
    runtimes/
        php/
            <version>/
                <target>/
    composer/
        <version>/
    wp-cli/
        <version>/
    extensions/
        <php-api>/
            <target>/
    tools/
        <tool-id>/
            <version>/
    services/
        <service>/
            <version>/
    data/
        databases/
            <engine>/
                <version>/
        mail/
    cache/
        artifacts/
        metadata/
        downloads/
    state/
        projects/
        sites/
        databases/
        processes/
        locks/
    logs/
```

Platform appropriate base directories must be used on Windows and macOS when required. The displayed path is the Unix conceptual layout.

### 13.5 Project Scoped State

Generated state that should not be committed may live beneath:

```text
.phpx/
    php.ini.d/
    php-fpm.d/
    database/
    sockets/
    logs/
    run/
```

The project should recommend ignoring `.phpx/`. The committed `phpx.toml` and `phpx.lock` remain at the project root unless later usability research favors a dedicated directory.

### 13.6 Daemon Model

1. Runtime installation, Composer execution, tool execution, and synchronization must work without a daemon.
2. The Laravel local environment uses one user scoped daemon for persistent routes, PHP FPM pools, resource scheduling, and shared services.
3. The daemon must never run as root.
4. Privileged setup must be handled by a narrowly scoped helper or explicit operating system command.
5. CLI and graphical clients must communicate with the daemon through a versioned authenticated local socket.

### 13.7 Laravel Portfolio Process Topology

The target topology for 50 registered projects is:

```text
One PHPX user daemon
    ├── One Go HTTP and TLS proxy
    ├── One site registry and resource scheduler
    ├── One shared mail capture process
    ├── Shared versioned SQL and cache engines for compatible applications
    ├── Additional shared database engines only when required
    ├── One PHP FPM master for each active PHP version
    │       ├── One configured pool per registered application using that version
    │       └── Zero child workers for each idle pool
    └── Optional project processes started only by declaration
```

The PHP FPM `ondemand` process manager allows a configured pool to have no child worker until a request arrives. PHPX must also set a global process ceiling so traffic across many sites cannot create an unbounded number of PHP workers.

PHP FPM pools are not a hostile code security boundary and may share an OPcache instance within one PHP FPM master. A future dedicated runtime mode may start a separate master for projects requiring stronger local separation.

### 13.8 Shared Database Topology

1. A database engine process is keyed by engine type, engine version, target, and relevant configuration profile.
2. Every project receives a separate database and database user.
3. Project credentials are stored locally and injected through an approved Laravel environment strategy.
4. Stopping or sleeping a site never stops a shared engine that serves another project.
5. Removing a site registration never drops its database automatically.
6. A database engine may stop only when no registered project depends on it or the user explicitly requests a global service stop.
7. Engine upgrades require logical export and verified restore when binary storage formats are not compatible.

### 13.9 Native Filesystem Model

PHPX executes Laravel directly from the host project path. It does not copy the project into a virtual machine or container volume and does not require a file synchronization daemon for ordinary operation. Application storage remains on normal host paths unless the project explicitly selects a managed external path.

This model removes the container bind mount and synchronization costs that can affect DDEV performance on macOS. It also means PHPX does not provide container level filesystem isolation by default.

### 13.10 Implementation Language Decision

Go is the selected implementation language for the durable PHPX control plane. This includes the command line executable, user daemon, local proxy, resource scheduler, artifact verifier, runtime installer, process supervisor, and operating system integration.

This is not a goal to rewrite the PHP ecosystem in Go. PHP remains the application language and the compatibility authority for Composer, WordPress, WP CLI, Laravel, Symfony, and project supplied scripts. PHPX should invoke those tools through the selected PHP runtime instead of approximating their behavior.

**Decision status:** Accepted on July 15, 2026.

Go was selected because PHPX is primarily a web adjacent orchestration product. Its control plane coordinates HTTP, TLS, local sockets, downloads, files, databases, PHP FPM, and child processes. Go provides native distribution and strong concurrency while remaining close to the founding maintainer's web and backend experience. This improves the probability of reaching a useful Laravel release and sustaining the project afterward.

The product must not claim that Go is categorically faster than PHP or Rust. Go is expected to provide fast startup and efficient resident control plane behavior, but every performance claim remains subject to the published PHPX benchmarks. Laravel application execution remains PHP execution regardless of the control plane language.

#### 13.10.1 Why the Control Plane Should Not Depend on PHP

PHPX has a bootstrap constraint that normal PHP applications do not have. It must be able to discover a project, download and verify a PHP runtime, install that runtime, and diagnose a broken PHP installation on a machine where no usable PHP executable exists yet.

A core written in PHP creates a circular dependency. A PHAR provides convenient single file application packaging, but the machine still needs a compatible PHP interpreter to execute it. Shipping a private PHP interpreter beside the PHAR can solve the bootstrap problem, but it turns the bootstrap runtime into another platform specific artifact that the launcher must locate and manage.

PHP is still the right language in several parts of the system:

1. Composer remains the package operation authority.
2. Project supplied Artisan remains the Laravel command authority.
3. WP CLI remains the WordPress command authority when that adapter is installed.
4. Laravel, WordPress, and Symfony continue to execute as PHP applications.
5. Small framework probes may execute inside the selected PHP runtime when application boot behavior is the only reliable source of truth.
6. Compatibility fixtures may be written in PHP to prove that PHPX preserves ecosystem behavior.

PHP could operate a long lived daemon through existing event loop and process libraries. The objection is not that PHP is incapable or always slow. The stronger objections are the bootstrap dependency, the need to bundle an interpreter, the resident runtime cost, and the weaker fit for a cross platform process and service supervisor that must remain available while PHP installations are being changed.

#### 13.10.2 Why Go Fits This Product

1. Go can produce a native executable that starts without a separately installed PHP, Composer, Node.js, Bun, Java, or .NET runtime.
2. The standard library directly covers HTTP, TLS, sockets, archive formats, hashing, structured data, child processes, signals, filesystem access, testing, profiling, and cross platform primitives.
3. Goroutines and channels fit concurrent proxy requests, downloads, process event streams, health checks, log routing, and bounded background work without requiring a complex asynchronous type system.
4. The Go toolchain provides integrated formatting, testing, fuzzing, race detection, profiling, dependency management, and cross compilation.
5. Garbage collection removes most manual memory lifetime management and lets the initial team focus on Laravel behavior, artifact safety, process cleanup, and user experience.
6. The small language and fast compiler reduce the time required to understand, review, build, and change the control plane.
7. Go's common use in web services, command line tools, proxies, cloud infrastructure, and platform engineering makes PHPX concepts transferable to a broad contributor and maintainer audience.
8. Direct compilation to macOS, Linux, and Windows targets is straightforward while the core avoids C dependencies.
9. Native startup and low overhead are expected to support 50 registered sites without one heavy environment per site.
10. Built in profiling makes heap growth, garbage collection, goroutine leaks, CPU use, blocking, and contention measurable during the portfolio benchmark.

Go is most valuable here around PHP, not instead of PHP. The Go control plane performs orchestration. PHP continues to perform PHP ecosystem work.

#### 13.10.3 Go Costs and Failure Modes

Go also creates material costs:

1. Every standard Go executable includes a runtime and garbage collector.
2. Garbage collection reduces direct control over memory and can increase resident memory or introduce latency when allocation behavior is poor.
3. The compiler does not prevent all data races. Race testing observes only code paths that actually execute.
4. Goroutines can leak when ownership, cancellation, and channel shutdown are unclear.
5. Nil values, unchecked errors, panics, and partially initialized structs can produce runtime failures that a more expressive type system might prevent.
6. `cgo` complicates builds, cross compilation, distribution, and process isolation.
7. Go binaries may be larger than equivalent carefully optimized native binaries because they include the runtime and standard library code they use.
8. External modules create supply chain and maintenance risk even when the standard library covers much of the product.
9. Go does not solve PHP binary distribution, database distribution, certificates, DNS, or Laravel compatibility by itself.
10. A poorly designed Go daemon can still leak memory, leak goroutines, deadlock, consume excessive CPU, corrupt state through logic errors, or expose insecure local interfaces.

The team must account for these costs through resource budgets, profiling, cancellation rules, race testing, failure injection, dependency review, and narrow operating system providers.

#### 13.10.4 Why Not C or C++

C and C++ can meet the native startup, low level control, and low memory requirements. They also have mature platform APIs and broad library access.

They are not the preferred default because the control plane is both privileged in effect and exposed to complex inputs. It downloads artifacts, verifies metadata, extracts archives, edits configuration, removes files, routes HTTP requests, and supervises processes. Memory corruption, use after free behavior, undefined behavior, and data races create unnecessary risk in that boundary. Modern C++ reduces some of this risk through disciplined ownership types, but it does not enforce the same safety baseline across the program.

C++ also brings more variation in build systems, dependency management, compiler behavior, and safe coding conventions. Go provides the memory safety and development speed this product needs without taking on that complexity. PHPX may still call established C or C++ executables through narrow reviewed process interfaces when reimplementation would be wasteful.

#### 13.10.5 Why Not Bun and TypeScript

Bun and TypeScript offer fast product iteration, familiar asynchronous programming, a large package ecosystem, and the ability to compile an application into a distributable executable. Bun documents that such an executable includes a copy of the Bun runtime.

That model can remove a separate install step, but it does not remove the runtime. Go also embeds a runtime, so the presence of a runtime is not the deciding factor. Go is preferred because its static native toolchain, process model, mature cross platform support, and standard library fit a long lived local control plane better. TypeScript checks disappear at runtime, so data crossing process, filesystem, and network boundaries still requires careful runtime validation.

Bun may be a good future choice for a graphical client, web interface, documentation tooling, or development scripts. It is not selected for the first control plane because PHPX benefits more from Go's static compilation, standard library, explicit error handling, and established infrastructure ecosystem.

#### 13.10.6 Why Not Rust

Rust is the strongest alternative to Go for PHPX. It offers finer memory control, no garbage collector, stronger compile time protection against data races and invalid memory access, excellent C interoperability, and the possibility of lower idle memory.

Rust is not selected for the initial implementation because most PHPX work is orchestration around HTTP, files, downloads, and processes rather than an embedded runtime, database engine, or latency critical data plane. The largest resource improvement comes from shared native services and demand driven PHP workers, not from removing the Go garbage collector. Go also aligns more closely with the founding maintainer's web background and offers a shorter path to a maintainable Laravel release.

Mago, Yerd, and Libretto remain valuable adjacent Rust projects. PHPX should integrate through stable executable or language neutral interfaces instead of adopting a second implementation language. Rust may be reconsidered for an isolated component only when profiling proves that Go cannot meet a published resource, latency, security, or native integration requirement. The public MVP must not become a mixed Go and Rust codebase by default.

#### 13.10.7 Why Not Zig or Another Native Language

Zig has appealing properties for this product, including no garbage collector, strong C interoperability, native binaries, and cross compilation support. Its ecosystem for a complete cross platform daemon, proxy, package client, certificate manager, and process supervisor is less established than the Rust and Go options. It also places more memory management responsibility on the application.

Swift would fit a macOS only product well, but PHPX intends to support Linux and eventually Windows. Java, Kotlin, C Sharp, and similar managed platforms provide mature libraries and productive development, but add a runtime and garbage collector to a tool whose central promise includes low idle overhead and simple bootstrap distribution.

The comparison is therefore not Go against every language in the abstract. It is which language best satisfies native bootstrap, acceptable resident cost, practical concurrency, cross platform system control, contributor sustainability, and time to a reliable Laravel release.

#### 13.10.8 Why Go Is the Right First Implementation

Go is selected through a product tradeoff rather than a claim that it wins every technical category:

1. PHPX is a web adjacent control plane, not a new PHP interpreter or operating system service manager for hostile multiuser workloads.
2. The product needs HTTP, TLS, process management, downloads, files, concurrency, and cross platform releases more than manual memory control.
3. Shared services, native filesystem access, and demand driven PHP FPM workers produce the primary resource savings.
4. A smaller language and fast build loop improve the chance that a small team ships and maintains the complete Laravel workflow.
5. Go expands the contributor and maintainer path into backend, platform, cloud, and infrastructure engineering while remaining approachable from PHP.
6. Rust remains available later if measured evidence identifies an isolated problem that Go cannot solve within the resource budget.

Go is not selected merely because it is native or commonly used for infrastructure. The Milestone 0 proof must still demonstrate that the daemon, proxy, and scheduler meet PHPX resource and reliability targets.

#### 13.10.9 Language Boundary Rules

1. Do not reimplement Composer in the Go core.
2. Do not reimplement Artisan or framework commands in the Go core.
3. Keep framework specific application execution inside the selected PHP runtime.
4. Prefer stable subprocess contracts before linking deeply into fast moving external codebases.
5. Keep operating system specific code behind narrow provider interfaces.
6. Prefer the Go standard library before adding an external module.
7. Commit `go.mod` and `go.sum`, pin module versions, audit advisories and licenses, and review every new transitive dependency.
8. Forbid `unsafe` in normal application packages. Any unavoidable use requires isolation, written invariants, focused tests, and review.
9. Avoid `cgo` in the control plane. Any exception requires a documented need and a release plan for every supported target.
10. Every goroutine must have an owner, cancellation path, bounded work policy, and observable shutdown behavior.
11. Propagate `context.Context` through cancellable network, process, and filesystem operations without storing it in long lived domain structs.
12. Do not use panic for expected project, network, process, or user input failures.
13. Execute subprocesses with explicit argument vectors and context cancellation. Do not construct shell command strings when direct execution is possible.
14. Treat all project files, downloaded metadata, archives, daemon messages, and subprocess output as untrusted input.
15. Require `gofmt`, `go vet`, `go test`, race testing, vulnerability scanning, and the selected static analyzer in continuous integration.
16. Keep runtime installation and Composer execution usable without the daemon so a daemon failure cannot prevent recovery.
17. Add a language neutral local protocol only when a graphical client or external integration actually needs it.

#### 13.10.10 Milestone 0 Language Proof

Milestone 0 must build a narrow Go proof containing project discovery, one verified download, one managed child process, a local daemon connection, and one proxied PHP request. It must record:

1. Cold command startup time.
2. Idle daemon resident memory.
3. Idle memory with 50 registered routes and no PHP workers.
4. Memory and latency with 15 addressable Laravel applications and a bounded number of active requests.
5. Proxy overhead relative to direct PHP FPM access.
6. Binary size and installation size.
7. Build time and cross target release complexity.
8. Behavior after a killed child process, stale process identifier, corrupt download, interrupted extraction, and daemon restart.
9. Heap size, garbage collection frequency, garbage collection CPU cost, and memory limit behavior.
10. Goroutine count before and after repeated site start, stop, request, and failure cycles.
11. Race detector results under realistic concurrent workloads.
12. The amount of operating system specific, `cgo`, and unsafe code required.
13. Whether release targets build with `CGO_ENABLED=0`.
14. Contributor setup time on a clean machine.

PHP should also be measured as the compatibility baseline for project discovery and command execution. Go remains the accepted implementation language when it meets the published resource budgets and the implementation complexity remains supportable. If the proof misses a budget, the team must first profile allocations, goroutine lifecycle, proxy behavior, and process topology. Rust evaluation is warranted only when evidence shows that the Go runtime or language model is the limiting factor rather than the surrounding architecture.

## 14. PHP Runtime Distribution

### 14.1 Distribution Challenge

The PHP project publishes source releases for Unix platforms but does not publish official Linux or macOS binaries. PHPX therefore needs a trusted artifact strategy.

### 14.2 Initial Artifact Strategy

1. Evaluate StaticPHP and other reputable build sources.
2. Record upstream release, build configuration, target, extension set, and checksums.
3. Mirror artifacts only when licensing and redistribution terms permit it.
4. Verify artifacts before making them available through PHPX metadata.
5. Treat third party artifacts as replaceable providers behind one artifact interface.

### 14.3 Long Term Build Pipeline

The production pipeline should build and test at least:

1. macOS arm64.
2. macOS x86_64 while supported.
3. Linux x86_64 with glibc.
4. Linux arm64 with glibc.
5. Linux x86_64 with musl.
6. Windows x86_64.

Additional targets require demonstrated demand and sustainable test coverage.

### 14.4 Runtime Contents

A standard runtime artifact should include:

1. PHP CLI.
2. PHP FPM when supported on the target.
3. `php-config` and `phpize` when extension building is supported.
4. A documented broad set of common built in extensions.
5. Development and production INI templates.
6. License notices for PHP and bundled libraries.
7. Build metadata and a software bill of materials.

### 14.5 Artifact Profiles

Avoid a large matrix of binaries for every extension combination. Prefer one broad target artifact where practical, with project scoped INI fragments controlling extension activation.

If size requires profiles, begin with:

1. `laravel`, containing the documented required and recommended Laravel baseline.
2. `full`, including additional common database, internationalization, image, archive, and general application capabilities.
3. `dev`, adding headers and build tools needed for dynamic extensions.

The number of profiles must remain intentionally small.

### 14.6 Artifact Manifest Requirements

Every artifact record must include:

1. PHP version.
2. Target triple.
3. Thread model.
4. Debug status.
5. PHP API identifier.
6. Build flags.
7. Included extensions.
8. Linked native library versions.
9. Artifact size.
10. SHA256 digest.
11. Signature or provenance reference.
12. Source commit or source archive digest.
13. Build pipeline version.
14. Support and withdrawal status.

## 15. Composer Compatibility Policy

### 15.1 Compatibility Path

The first production path must always execute the official Composer application through the selected PHP runtime.

PHPX may prepare the runtime and environment, but Composer remains responsible for package operations.

### 15.2 Required Compatibility Cases

The test suite must cover:

1. Normal package installation from Packagist.
2. Private Composer repositories.
3. VCS repositories.
4. Path repositories.
5. Composer plugins.
6. Custom installers.
7. Root package scripts.
8. PHP callbacks.
9. `@php` and `@composer` scripts.
10. Authentication through supported Composer mechanisms.
11. Custom vendor directories.
12. Platform configuration.
13. Autoload customization.
14. Package binaries.
15. Projects with no lock file.

### 15.3 Future Native Locked Install Path

A native Go installation path may be introduced only for explicitly supported locked installations.

Eligibility detection must account for:

1. Composer plugins.
2. Custom package installers.
3. Installer path configuration.
4. Root scripts.
5. Nonstandard repositories.
6. Custom vendor directories.
7. Autoload modes.
8. Package types.
9. Authentication requirements.
10. Platform constraints.

If any behavior cannot be proven compatible, PHPX must invoke Composer instead.

### 15.4 Fallback Invariants

1. Fallback is automatic unless strict native mode was explicitly requested.
2. Fallback is displayed clearly.
3. Fallback does not change dependency resolution.
4. The same `composer.lock` remains authoritative.
5. Native mode never ignores scripts or plugins merely to remain eligible.

## 16. Security Requirements

### 16.1 Artifact Security

1. Every executable artifact must be verified before execution.
2. Release metadata must be signed.
3. Artifact downloads must use authenticated HTTPS.
4. Installation must verify expected size and digest.
5. Withdrawn artifacts must remain identifiable in existing locks with a clear warning.
6. The build pipeline must publish provenance and a software bill of materials.
7. Reproducible builds should be pursued where toolchains permit them.

### 16.2 Update Security

1. Self update must verify release metadata and executable integrity.
2. The previous executable must remain available until the new version starts successfully.
3. Update channels must not cross silently.
4. Security updates may be announced prominently, but automatic installation requires an explicit policy.

### 16.3 Local Environment Security

1. Web services bind to loopback by default.
2. Database and cache services bind to loopback by default.
3. Public sharing requires an explicit command and visible active status.
4. The local certificate authority private key must use operating system appropriate restrictive permissions.
5. Privileged helpers must expose the smallest possible operation set.
6. PHPX must never request broad persistent administrator access.

### 16.4 Secret Handling

1. Composer credentials remain in Composer supported locations.
2. Tokens must be redacted from logs and diagnostic output.
3. Environment values matching known secret patterns must be redacted.
4. PHPX lock and configuration files must reject secret fields in documented locations.
5. Optional operating system keychain integration may be added for PHPX specific credentials.

### 16.5 Executable Package Behavior

Composer plugins and scripts execute project supplied code. PHPX must preserve this behavior while making the trust boundary visible during first use or through a security policy command. It must not describe a Composer project installation as sandboxed when it is not.

Laravel applications, Composer packages, Artisan commands, service providers, migrations, seeders, and project scripts can execute arbitrary PHP. The default shared native topology is intended for trusted local projects, not hostile code isolation. Separate database users and PHP FPM pools reduce accidental crossover but do not create a security sandbox. The same warning applies to WordPress code when that adapter is installed. An optional dedicated or container backend should be recommended for untrusted projects.

### 16.6 Go Supply Chain Security

1. Prefer the Go standard library when it provides a maintainable implementation.
2. Commit `go.mod` and `go.sum` and build releases with a documented Go toolchain version.
3. Run Go vulnerability analysis for every release and in continuous integration.
4. Review module licenses, maintainers, update history, transitive dependencies, and security posture before adoption.
5. Do not add direct branch, local replacement, or unreviewed fork dependencies to release builds.
6. Treat `cgo` and `unsafe` usage as security exceptions requiring written justification and focused review.
7. Record the Go toolchain and module graph in release provenance and the software bill of materials.

## 17. Performance and Reliability Targets

These are engineering targets, not claims until benchmarks are published.

### 17.1 Command Responsiveness

1. Local informational commands should start and render within 100 milliseconds on a representative modern machine.
2. Project discovery should complete within 50 milliseconds for ordinary repositories.
3. Lock validation should avoid network access when metadata is current.
4. Warm synchronization overhead before Composer execution should remain below 500 milliseconds for an unchanged project.

### 17.2 Artifact Efficiency

1. Identical immutable artifacts should be stored once per machine.
2. Downloads should support resumption when the server supports byte ranges.
3. Independent downloads should execute concurrently within a configurable limit.
4. Cache cleanup must preserve artifacts referenced by active locks unless explicitly forced.

### 17.3 Process Reliability

1. The local daemon should target less than 50 MB idle memory before managed services.
2. PHP FPM and service crashes must update status promptly.
3. Process identifiers must be validated before termination to avoid killing unrelated processes.
4. Stale sockets and state files must be recoverable through normal startup or `doctor`.

### 17.4 Laravel Portfolio Resource Budget

The benchmark machine for the first published portfolio result must include an Apple Silicon Mac. Results must state the exact hardware, macOS version, PHP versions, Laravel fixtures, package set, service configuration, database contents, process profile, and measurement method.

Engineering targets:

1. Fifty registered projects create no dedicated long running process per project.
2. Fifteen Laravel applications remain addressable concurrently through the shared proxy.
3. An idle application returns to zero PHP child workers after its configured idle timeout.
4. One PHP FPM master is shared by application pools using the same PHP version unless dedicated mode is selected.
5. The global PHP worker count remains bounded even when requests arrive for every application.
6. The default global worker budget is derived from available processors and memory, can be overridden, and is visible in status output.
7. The PHPX daemon and proxy together should target less than 100 MB idle resident memory before database and mail services.
8. The default shared database engine should target less than 512 MB idle resident memory under the public MVP fixture workload.
9. PHPX creates no duplicate synchronized copy of Laravel project files.
10. No Node process, asset watcher, queue worker, or scheduler loop starts for an application unless requested.
11. `phpx sites` should render 50 registry entries within 200 milliseconds without booting Laravel.
12. A registered application should answer its first request within two seconds when its runtime and shared services are already installed and healthy.
13. Published comparison must include a native Laravel environment such as Herd or Valet and a container environment such as Sail or DDEV with the same 50 registered and 15 available application scenario where practical.
14. Published daemon measurements must separate total resident memory, Go heap, goroutine count, garbage collection frequency, garbage collection CPU time, and non Go child processes.
15. Repeating application start, request, sleep, wake, and stop cycles must not produce unbounded goroutine or heap growth.

The product must report actual resource consumption. It must not claim a fixed per application memory number because Laravel packages, boot behavior, PHP settings, active processes, and traffic change resource use substantially.

### 17.5 Atomicity

1. Runtime installation is atomic.
2. Composer installation is atomic.
3. Tool installation is atomic.
4. Lock file replacement is atomic.
5. Self update is atomic.
6. Service data deletion is never part of ordinary synchronization.

## 18. User Experience Requirements

### 18.1 Output

1. Lead with the result or current action.
2. Show resolved versions without overwhelming routine output.
3. Make verbose evidence available on demand.
4. Use stable wording for errors consumed by documentation and support.
5. Disable animation and color automatically when output is not interactive.
6. Provide JSON for automation without mixing human text into standard output.

### 18.2 Error Messages

Every actionable error should answer:

1. What failed.
2. Which project and target were involved.
3. What evidence PHPX observed.
4. Why PHPX could not continue safely.
5. Which command or file should be used next.

Example:

```text
PHP 8.2 cannot satisfy this project.

phpx.toml requests: 8.2
composer.json requires: ^8.4

Update phpx.toml or run:
phpx php pin 8.4
```

### 18.3 Destructive Actions

1. Display the exact paths and resources affected.
2. Require confirmation in an interactive terminal.
3. Require an explicit force flag without a terminal.
4. Support dry run where practical.
5. Never combine cache cleanup with project or service data deletion.

## 19. Testing Strategy

### 19.1 Unit Tests

1. Composer constraint parsing.
2. Version precedence.
3. Target selection.
4. Artifact manifest validation.
5. Hash verification.
6. Lock serialization.
7. Configuration schema validation.
8. Path and environment construction.
9. Error redaction.
10. Process identity validation.
11. Laravel application and package repository detection.
12. Laravel application root and public document root selection.
13. Laravel and PHP compatibility lookup.
14. Laravel front controller and public asset routing.
15. Laravel environment preparation planning.
16. Shared database allocation.
17. Site worker scheduling and global limits.
18. Portfolio registry queries.

### 19.2 Compatibility Tests

1. Compare constraint results against Composer Semver fixtures.
2. Compare direct Composer and PHPX wrapped Composer behavior.
3. Cover plugins, scripts, custom installers, repositories, and authentication stubs.
4. Verify platform requirements through actual selected runtimes.
5. Test supported PHP patch releases and API identifiers.
6. Compare direct Artisan execution with PHPX wrapped Artisan behavior.
7. Validate Laravel environment and service injection strategies.
8. Validate Laravel front controller routing through the Go proxy.
9. Validate that only the public document root is exposed as static content.
10. Compare imported Sail and DDEV configuration with the generated PHPX plan when those importers are enabled.

### 19.3 End to End Fixtures

Maintain representative fixtures for:

1. Current Laravel application using SQLite.
2. Current Laravel application using the first supported shared SQL engine.
3. Supported Laravel application using Redis and mail capture.
4. Laravel application with queue and scheduler process declarations.
5. Laravel application with an explicit frontend asset process.
6. Laravel application with a subdirectory or monorepo project root.
7. Laravel application with existing local environment secrets that PHPX must preserve.
8. Laravel application using common Sail configuration.
9. Laravel application using common DDEV configuration.
10. Laravel package repository that must not be served as an application.
11. Portfolio fixture containing 50 registered projects and 15 routed Laravel applications.
12. Current classic WordPress site after the Laravel public MVP.
13. Current Bedrock site after the Laravel public MVP.
14. Current Symfony application after WordPress support.
15. Generic PHP site and Composer library after Symfony support.
16. Project with a Composer plugin.
17. Project with a dynamic extension requirement.
18. Project with a custom vendor directory.
19. Project with an unsupported custom container service.

### 19.4 Failure Injection

Test:

1. Network interruption.
2. Invalid digest.
3. Truncated archive.
4. Disk full condition.
5. Permission denial.
6. Concurrent installation.
7. Process crash.
8. Port collision.
9. Stale lock.
10. Invalid certificate state.
11. Shared database engine crash while multiple sites are registered.
12. PHP worker exhaustion across 15 requested sites.
13. Corrupt Laravel database import.
14. Sail or DDEV import containing unsupported custom Docker configuration.
15. Site removal requested while no current database snapshot exists.

### 19.5 Platform Matrix

Every supported tier requires automated testing on each supported architecture. A platform cannot be labeled supported solely because the Go binary compiles.

### 19.6 Performance Tests

Publish repeatable benchmarks for:

1. CLI startup.
2. Project discovery.
3. Cold runtime installation.
4. Warm synchronization.
5. Isolated tool startup.
6. Fifty site registry listing.
7. Fifteen site addressability with idle workers.
8. Fifteen site concurrent request bursts under a global worker cap.
9. Shared database idle and active memory.
10. First response after a site pool has returned to zero workers.
11. Artifact cache size and reuse.
12. Disk usage compared with native and container Laravel environments using equivalent projects.
13. Total process count compared with native and container Laravel environments using equivalent projects.
14. Go heap, total resident memory, goroutine count, and garbage collection behavior while idle and under the 15 site request fixture.
15. Goroutine and heap stability across at least 1,000 repeated site lifecycle cycles.
16. Race detector coverage for concurrent route, process, database allocation, and shutdown operations.

## 20. Platform Support Policy

### 20.1 Proposed Tiers

**Tier 1:** Guaranteed through continuous testing and release blocking failures.

**Tier 2:** Expected to work and tested regularly, but may not block every release.

**Tier 3:** Community supported and compiled when feasible.

### 20.2 Initial Targets

1. Technical proof Tier 1: macOS arm64.
2. Laravel public MVP Tier 1: macOS arm64.
3. Laravel public MVP automation target: Linux x86_64 with glibc for frozen synchronization and project commands in continuous integration, without local DNS, TLS, proxy, or native service support.
4. Later full Tier 1 candidate: Linux x86_64 with glibc.
5. Later Tier 2 candidate: Linux arm64 with glibc.
6. Later Tier 1 candidate: Windows x86_64.
7. Later Tier 2 candidate: macOS x86_64 while the platform remains viable.
8. Later Tier 2 candidate: Linux x86_64 with musl.

### 20.3 PHP Support

1. Supported PHP branches receive normal support.
2. Security only PHP branches may remain installable with visible warnings.
3. End of life branches require an explicit legacy policy.
4. PHPX must never imply security support beyond the PHP project policy.
5. Legacy installation may be useful for maintenance, but it must be clearly separated from recommended defaults.

## 21. Release and Distribution

### 21.1 Initial Distribution

1. Signed release archives.
2. A reviewed shell installer for macOS during the first public MVP.
3. Direct manual archive installation.
4. `go install` for contributors, not as the primary user installation path.

### 21.2 Later Distribution

1. Homebrew formula.
2. Linux installer and packages for selected distributions.
3. Windows installer and package manager support.
4. Continuous integration setup integration.
5. Optional graphical client.

### 21.3 Release Channels

1. `stable` for supported production releases.
2. `preview` for release candidates and compatibility testing.
3. `nightly` for development builds without stability guarantees.

### 21.4 Versioning

PHPX should use semantic versioning after the configuration, lock, and command contracts stabilize. Before 1.0, compatibility expectations must still be documented per release.

## 22. Open Source and Sustainability

The recommended starting position is an open source core because ecosystem infrastructure depends on trust, inspectable artifact handling, and broad compatibility contributions.

The following must remain open specifications even if hosted products are introduced later:

1. `phpx.toml` schema.
2. `phpx.lock` schema.
3. Artifact manifest schema.
4. Runtime provider interface.
5. Extension provider interface.
6. Local daemon protocol required by third party clients.

Possible later sustainability paths include:

1. Hosted signed artifact mirrors.
2. Shared team caches.
3. Private runtime and extension mirrors.
4. Enterprise policy and audit features.
5. Supported graphical clients.
6. Commercial support.

Licensing and governance remain open decisions.

## 23. Implementation Milestones

### Milestone 0: Laravel Product and Architecture Validation

Deliverables:

1. Confirm the working product and repository name.
2. Complete the necessity interviews and comparison requirements from Section 1.9.
3. Publish a capability matrix for Herd Basic, Herd Pro, Valet, Sail, DDEV, PHP version managers, and PHPX.
4. Measure representative Herd or Valet, Sail or DDEV, and PHPX proof workloads on an Apple Silicon Mac with 1, 10, and 15 available Laravel applications.
5. Define the one workflow PHPX must perform materially better, using the same locked environment locally and in Linux continuous integration as the leading candidate.
6. Compare Yerd, StaticPHP, Herd, Valet, Sail, DDEV, Mise, and existing PHP version managers before duplicating their components.
7. Select the first PHP runtime artifact provider and verify PHP CLI and PHP FPM support.
8. Select SQLite and the first shared SQL, Redis, and mail providers for the proof.
9. Define artifact, runtime, database, cache, application adapter, and process provider interfaces.
10. Define Laravel application and package repository detection without executing project code.
11. Define Composer and Artisan execution policies.
12. Define safe Laravel environment preparation and secret handling.
13. Import or recreate a Composer Semver conformance corpus for Laravel and Composer projects.
14. Build the narrow Go language proof described in Section 13.10.10.
15. Select licenses for the code and documentation.
16. Produce security threat models for artifacts, shared native services, Laravel executable code, project secrets, and local TLS.

Exit criteria:

* [ ] The necessity test in Section 1.9 passes with recorded evidence.
* [ ] No foundational component is being duplicated without an explicit reason.
* [ ] The first PHP runtime artifact can be verified and legally redistributed or fetched.
* [ ] The runtime includes usable PHP CLI and PHP FPM binaries.
* [ ] Native SQL, Redis, and mail distribution paths are understood for the proposed MVP scope.
* [ ] The first Laravel comparison workload and measurement method are recorded.
* [ ] Laravel environment preparation and secret handling have an approved direction.
* [ ] A current Laravel fixture can be discovered and served through the proof without changing the system PHP installation.
* [ ] The project has a documented stop or narrowing decision if the evidence does not justify a new tool.

### Milestone 1: Native Laravel Vertical Slice

Deliverables:

1. Go module and command line shell.
2. User scoped daemon.
3. Laravel application and package repository discovery.
4. Managed PHP installation for macOS arm64.
5. Managed Composer and project supplied Artisan execution.
6. One managed PHP FPM master with demand driven application pools.
7. Go HTTP and TLS proxy.
8. Explicit local DNS and certificate setup.
9. SQLite plus one shared SQL engine with isolated project databases and users.
10. Reviewable Laravel local environment preparation.
11. `phpx init`, `up`, `down`, `status`, `artisan`, `composer`, `run`, `db create`, and `doctor`.
12. Current Laravel fixtures for SQLite and the selected shared SQL engine.
13. A 15 application routing and worker scheduling fixture.

Exit criteria:

* [ ] A clean macOS arm64 environment can install dependencies and serve the Laravel fixture through trusted HTTPS.
* [ ] Artisan operates through the selected runtime with direct command compatibility.
* [ ] SQLite and shared SQL fixtures connect through approved environment strategies.
* [ ] Laravel front controller routing and public assets work.
* [ ] Fifteen registered application routes can remain available while idle pools have zero PHP child workers.
* [ ] The system PHP installation remains unchanged.
* [ ] Interrupted installation recovery is proven.

### Milestone 2: Laravel Public MVP

Deliverables:

1. `phpx.toml` and `phpx.lock` version 1.
2. Portfolio registry supporting at least 50 projects.
3. Link, unlink, park, unpark, sleep, wake, open, logs, and site status workflows.
4. PHP runtime list, pin, update, and remove operations.
5. Supported PHP branches plus a documented Laravel and legacy PHP policy.
6. Versioned Laravel extension baseline.
7. PIE integration for selected dynamic extensions.
8. Database create, import, export, snapshot, restore, shell, and remove operations.
9. Shared Redis and mail capture with project isolation and filtering.
10. Laravel environment preparation with existing secret preservation.
11. Explicit queue, scheduler, log, and frontend process profiles.
12. Frozen, offline, noninteractive, and JSON modes.
13. Continuous integration validation using the committed lock.
14. Signed macOS release and installer.
15. Published PHPX, Herd or Valet, and Sail or DDEV workflow and portfolio benchmarks.

Exit criteria:

* [ ] Every Laravel public MVP success criterion passes on macOS arm64.
* [ ] Fifty projects remain registered without dedicated per project background processes.
* [ ] Fifteen Laravel applications remain addressable without 15 web and database container pairs.
* [ ] Database and local service data survive stop, sleep, restart, and PHPX upgrades.
* [ ] Existing `.env` files and application secrets survive initialization and synchronization unchanged unless the developer explicitly approves a mutation.
* [ ] The same committed PHPX lock validates locally and in the supported continuous integration fixture.
* [ ] Security, support, legacy PHP, and artifact provenance policies are published.

### Milestone 3: Laravel Workflow Maturity

Deliverables:

1. Additional SQL provider support selected by validated demand.
2. Laravel Sail and DDEV import, coexistence, and migration diagnostics.
3. Supported profiles for Horizon, Reverb, queue workers, the scheduler, Pail, and frontend development.
4. An explicit compatibility strategy for Octane and custom web server requirements.
5. Testing database allocation that cannot overwrite development data.
6. Team environment validation and deeper continuous integration integration.
7. Environment diff and explanation commands.
8. Resource history and portfolio diagnostics.
9. A completed decision on managed Node acquisition versus integration with existing Node managers.
10. Additional macOS reliability and upgrade testing.

Exit criteria:

* [ ] Imported Sail and DDEV projects receive a reviewable plan and unsafe automatic cutover remains blocked.
* [ ] Laravel package repositories cannot be misidentified and served as full applications.
* [ ] Custom web server requirements produce a supported backend or a blocking explanation.
* [ ] Development profiles start only the processes explicitly selected by the project or command.
* [ ] Team locks are enforceable locally and in continuous integration.
* [ ] Portfolio performance remains within published budgets.

### Milestone 4: WordPress Adapter

Deliverables:

1. Classic WordPress, Bedrock, and Composer based WordPress discovery.
2. WordPress root, content directory, upload directory, and document root selection.
3. Managed WP CLI acquisition and execution.
4. WordPress database import, export, snapshot, restore, and explicit URL migration workflows.
5. Standard WordPress front controller and permalink routing.
6. Classic WordPress and Bedrock local environment strategies.
7. Versioned WordPress extension baseline.
8. Common DDEV WordPress import and coexistence.
9. Current classic WordPress, Bedrock, and Composer based fixtures.

Exit criteria:

* [ ] Current classic WordPress and Bedrock fixtures reach working local HTTPS environments.
* [ ] WP CLI operates through the selected project runtime and cannot target the wrong site silently.
* [ ] Standard WordPress permalinks work through the default proxy.
* [ ] Database and upload workflows preserve project data and never run URL replacement implicitly.
* [ ] Laravel workflow and portfolio benchmarks do not regress beyond the published tolerance.

### Milestone 5: WordPress Portfolio Maturity

Deliverables:

1. WordPress multisite subdirectory support.
2. WordPress multisite subdomain support with wildcard local routing and TLS.
3. Plugin and theme project workflows using an explicit host WordPress site.
4. Richer DDEV import coverage and migration diagnostics.
5. An explicit Apache compatibility backend or documented alternative for sites dependent on custom `.htaccess` behavior.
6. Optional WordPress Redis and object cache integration.
7. Environment diff and portfolio diagnostics for mixed Laravel and WordPress projects.

Exit criteria:

* [ ] Both supported multisite modes pass routing, WP CLI, database, upload, and domain fixtures.
* [ ] Plugin and theme repositories cannot accidentally operate against the wrong host site.
* [ ] Custom web server requirements produce a supported backend or a blocking explanation.
* [ ] WordPress portfolio behavior remains within the shared resource budgets.

### Milestone 6: Symfony and Generic PHP Adapters

Deliverables:

1. Symfony project discovery and public directory selection.
2. Symfony Console execution and common process presets.
3. Generic PHP document root and front controller configuration.
4. Generic Composer library mode without a web server.
5. Adapter authoring documentation.
6. Current Symfony, generic site, and generic library fixtures.

Exit criteria:

* [ ] A current Symfony application reaches a working local HTTPS environment.
* [ ] A generic PHP site works with explicit routing configuration.
* [ ] A generic Composer library can synchronize and test without starting local services.
* [ ] New adapters use shared core services rather than duplicating lifecycle logic.

### Milestone 7: Additional Platforms

Deliverables:

1. Linux x86_64 with glibc support.
2. Linux native service integration and package testing.
3. Windows architecture spike and PHP FastCGI strategy.
4. Additional targets promoted only after full workflow testing.
5. Portable continuous integration setup.

Exit criteria:

* [ ] Linux Tier 1 workflows pass the same application and artifact standards as macOS.
* [ ] Platform differences are explicit in configuration and diagnostics.
* [ ] Windows receives no support tier until local sites, PHP execution, databases, and cleanup are tested end to end.

### Milestone 8: Composer Installation Acceleration

Deliverables:

1. Eligibility analyzer for native locked installation.
2. Content addressed package cache.
3. Parallel package download and extraction.
4. Supported autoload generation.
5. Automatic Composer fallback.
6. Compatibility and performance benchmarks.

Exit criteria:

* [ ] Every unsupported case falls back safely.
* [ ] Native output matches Composer for the supported fixture corpus.
* [ ] Published benchmarks demonstrate meaningful value in cold and continuous integration scenarios.

## 24. Acceptance Scenarios

### Scenario 1: Clean Machine

**Given** a supported macOS arm64 machine without usable PHP, Composer, web server, or database

**When** the developer runs `phpx up` in a supported Laravel application

**Then** PHPX installs verified managed artifacts, installs Composer dependencies, prepares approved local environment values, allocates the configured database, registers trusted HTTPS, serves Laravel, and leaves the system runtime unchanged.

### Scenario 2: Multiple PHP Versions

**Given** Laravel Application A and Laravel Application B require different supported PHP branches

**When** the developer runs a PHP command in each project through PHPX

**Then** each command uses the correct runtime without global relinking.

### Scenario 3: Laravel Composer Plugin

**Given** a Laravel application depends on a Composer plugin

**When** the developer synchronizes the project

**Then** PHPX invokes Composer and preserves the normal plugin authorization and execution flow.

### Scenario 4: Missing Extension

**Given** Laravel or Composer requires a supported dynamic extension that is absent

**When** PHPX resolves the environment

**Then** PHPX identifies a PIE provider, requests any required explicit selection, installs it for the correct ABI, locks it, and reruns platform validation.

### Scenario 5: Existing Environment Preservation

**Given** a Laravel application already contains a local `.env` file, application key, and database credentials

**When** the developer runs `phpx init` or `phpx up`

**Then** PHPX reports any incompatible values and leaves the existing file and secrets unchanged unless the developer approves a specific mutation.

### Scenario 6: Frozen Continuous Integration

**Given** committed Composer and PHPX lock files

**When** continuous integration runs `phpx sync --frozen --no-interaction`

**Then** no lock changes occur and any mismatch fails before tests run.

### Scenario 7: Offline Operation

**Given** all required artifacts are cached

**When** the developer runs `phpx sync --offline`

**Then** synchronization succeeds without network access.

### Scenario 8: Missing Offline Artifact

**Given** one required artifact is not cached

**When** the developer runs offline synchronization

**Then** PHPX fails and names the exact missing artifact without attempting a connection.

### Scenario 9: Interrupted Download

**Given** a runtime download is interrupted

**When** the developer retries installation

**Then** PHPX resumes or safely restarts the download and never treats the partial archive as installed.

### Scenario 10: Laravel Portfolio

**Given** 50 registered PHP projects containing Laravel applications that use several PHP versions and compatible service versions

**When** 15 sites are requested during the same work period

**Then** each application routes to its matching PHP FPM pool, compatible applications share engine processes, the global worker cap is respected, and idle pools return to zero PHP child workers.

### Scenario 11: Destructive Database Removal

**Given** a Laravel project database contains local data and no current snapshot

**When** the developer requests database removal

**Then** PHPX displays the application, engine, database, user, and backup state and refuses deletion without explicit confirmation.

### Scenario 12: Shared Database Engine

**Given** 12 Laravel applications request the same supported SQL engine and version in shared mode

**When** all sites are registered

**Then** PHPX runs one compatible engine process and creates 12 separate databases and users.

### Scenario 13: Idle Application Wake

**Given** a registered Laravel application has no PHP child worker

**When** its local URL receives a request

**Then** the selected PHP FPM master creates a worker on demand and the proxy completes the request without a manual site start command.

### Scenario 14: Explicit Development Profile

**Given** a Laravel application declares queue, scheduler, log, and frontend processes but does not enable automatic startup

**When** the developer runs the ordinary web profile

**Then** PHPX serves the application without starting those optional processes, and `phpx up --profile dev` starts only the processes named by the development profile.

### Scenario 15: Laravel Package Repository

**Given** a repository contains Laravel packages but no complete Laravel application or public front controller

**When** PHPX discovers and synchronizes the repository

**Then** PHPX treats it as a Composer library, supports package and test commands, and does not register a local web route automatically.

### Scenario 16: Site Sleep

**Given** a Laravel application has an asset watcher, queue worker, and PHP workers running

**When** the developer runs `phpx site sleep`

**Then** PHPX stops optional project processes, allows PHP workers to retire, and preserves the route, database, storage, configuration, and future wake behavior.

### Scenario 17: Later WordPress Adapter

**Given** the WordPress adapter milestone is complete and a classic WordPress project contains a database dump and standard permalink rules

**When** the developer initializes and starts the project

**Then** PHPX reuses the Laravel proven runtime, proxy, database, and scheduling core while adding WordPress discovery, WP CLI, import, and permalink behavior.

## 25. Risks and Mitigations

### 25.1 Runtime Artifact Trust

**Risk:** PHPX depends on binaries not published by the PHP project for macOS and Linux.

**Mitigation:** Start with vetted providers behind an abstraction, verify every artifact, publish provenance, and build an owned reproducible pipeline before claiming broad production reliability.

### 25.2 Composer Semantic Complexity

**Risk:** A native implementation may break plugins, installers, scripts, repositories, or autoload behavior.

**Mitigation:** Delegate all package operations to Composer first. Add native acceleration only through strict eligibility and transparent fallback.

### 25.3 Extension Matrix Complexity

**Risk:** PHP version, API, target, native libraries, flags, and extension versions create a large compatibility matrix.

**Mitigation:** Ship broad standard runtimes, integrate PIE, isolate by ABI, limit the supported extension set initially, and publish exact compatibility data.

### 25.4 Scope Expansion

**Risk:** Runtime management, Composer, local serving, services, tools, and acceleration could become several products at once.

**Mitigation:** Enforce the necessity gate and Laravel milestone boundaries. The first vertical slice includes only the local capabilities required to prove one complete Laravel workflow and the lean shared topology. Defer WordPress, Symfony, generic PHP, broad service matrices, graphical clients, and native Composer installation until their named milestones.

### 25.5 Existing Competitors

**Risk:** Herd, Valet, Sail, DDEV, Yerd, Mise, or another project may already solve enough of the product that a new tool creates fragmentation instead of value.

**Mitigation:** Apply the Section 1.9 stop conditions. Treat integration and contribution as preferred options when they preserve the product promise. Compete only on a validated open, headless, deterministic, native environment contract, not on duplicating local HTTPS, PHP switching, or service buttons.

### 25.6 Privileged Operating System Integration

**Risk:** DNS, trusted certificates, and privileged ports can require administrator access and create security concerns.

**Mitigation:** Use one explicit setup operation, a minimal helper, loopback defaults, transparent changes, and complete uninstall instructions.

### 25.7 Cross Platform Drift

**Risk:** A command may compile everywhere while behaving differently across platforms.

**Mitigation:** Define support tiers by full workflow tests, not compilation. Keep provider interfaces platform aware and publish known differences.

### 25.8 Ecosystem Trust

**Risk:** Developers may resist a tool that appears to replace Composer or hide environment behavior.

**Mitigation:** Preserve Composer files, expose underlying commands, publish open formats, explain decisions, and make fallback visible.

### 25.9 Artifact Hosting Cost

**Risk:** PHP runtimes, services, extensions, and caches can create substantial bandwidth and storage costs.

**Mitigation:** Begin with upstream fetch providers, use content addressed caching, measure real demand, and design mirrors as optional providers.

### 25.10 Working Name Conflict

**Risk:** PHPX may conflict with an existing product, package, command, domain, or trademark.

**Mitigation:** Treat PHPX as a working name and complete naming research before the first public release.

### 25.11 Shared Service Failure Radius

**Risk:** One shared database, cache, or PHP FPM master failure can affect several applications at once.

**Mitigation:** Use health checks, bounded restart policy, versioned engine instances, durable project data, logical snapshots, clear dependency reporting, and optional dedicated mode for sensitive projects.

### 25.12 Shared Native Trust Boundary

**Risk:** A compromised or intentionally hostile Laravel application, Composer package, Artisan command, migration, or project script may access resources available to the local user and may attack shared local services.

**Mitigation:** State clearly that native mode is for trusted local projects, use separate database users, restrict services to loopback, minimize credentials exposed to each process, and offer a future dedicated or container backend for untrusted code.

### 25.13 Laravel Environment Mutation

**Risk:** Automatically copying or editing `.env`, generating `APP_KEY`, or changing database and service values may destroy local secrets, disconnect existing data, or make the project behave differently from its documented workflow.

**Mitigation:** Generate a reviewable plan, preserve existing files by default, store generated credentials outside version control, expose every injected value through explanation commands, and require explicit approval for persistent mutation.

### 25.14 Laravel Server Modes

**Risk:** Applications may depend on Octane, FrankenPHP, RoadRunner, custom Nginx behavior, websocket servers, or container networking that PHP FPM behind the default proxy cannot reproduce.

**Mitigation:** Make PHP FPM the documented first backend, detect known alternative server declarations, refuse unsupported claims, and add explicit backend adapters only after end to end fixtures prove them.

### 25.15 Development Process Sprawl

**Risk:** Queue workers, the scheduler, Horizon, Reverb, Pail, Vite, Octane, and custom project commands can recreate the same resource problem as per project containers when started automatically.

**Mitigation:** Keep web routing separate from development process profiles, default optional processes to stopped, cap global resources, and show every active process and its owner in status output.

### 25.16 Native Service Distribution

**Risk:** Portable SQL, Redis, mail, search, and storage service acquisition, upgrades, data directories, and platform support create several artifact supply chains.

**Mitigation:** Begin with SQLite and the minimum validated shared providers, keep every service provider replaceable, prefer logical backups across incompatible versions, verify artifacts, and delay broad engine matrices until real projects require them.

### 25.17 Performance Claim Credibility

**Risk:** A leaner than container environments claim may become vague or misleading when package sets, enabled processes, services, and traffic differ.

**Mitigation:** Publish exact fixtures, commands, hardware, idle and active states, process counts, memory, disk, and response measurements. Keep claims limited to reproducible benchmark results.

### 25.18 Go Runtime and Concurrency Discipline

**Risk:** Garbage collection, goroutine leaks, data races, `cgo` dependencies, or careless process cancellation may undermine the lean resource promise or create unreliable shutdown behavior.

**Mitigation:** Prove the smallest Go control plane in Milestone 0, profile the real portfolio fixture, establish goroutine ownership and cancellation rules, run race tests, avoid `cgo`, isolate operating system providers, and enforce explicit heap and goroutine stability budgets.

### 25.19 Managed JavaScript Boundary

**Risk:** Managing Node.js, Bun, Corepack, and JavaScript package managers can double the product scope and duplicate mature version managers, while ignoring them can prevent a clean Laravel clone from serving its frontend.

**Mitigation:** Supervise declared frontend commands in the first MVP, detect and explain missing runtimes, compare integration with existing managers, and add managed JavaScript acquisition only after Laravel workflow research proves it is necessary.

### 25.20 Later WordPress Complexity

**Risk:** Classic WordPress configuration, WP CLI, multisite, custom content paths, legacy PHP, `.htaccess`, and plugin behavior can overwhelm the Laravel first roadmap if they leak into the core too early.

**Mitigation:** Keep WordPress behind its application adapter and named milestones. Preserve the detailed WordPress requirements as later acceptance gates without allowing them to delay the Laravel public MVP.

## 26. Open Decisions

### 26.1 Product Name

**Recommended current decision:** Continue using PHPX internally only.

**Required before public release:** Search command names, package registries, repositories, domains, and trademarks.

### 26.2 License

**Recommended direction:** A permissive open source license for the core and open specifications.

**Still required:** Legal review of runtime, library, service, and extension redistribution obligations.

### 26.3 First Runtime Provider

**Recommended direction:** Evaluate StaticPHP for the technical proof while keeping the provider replaceable.

**Still required:** Confirm PHP FPM, development tools, artifact provenance, extension coverage, licenses, and release reliability.

### 26.4 Configuration Filename

**Recommended direction:** `phpx.toml` and `phpx.lock`.

**Still required:** Confirm that a new top level file is preferable to Composer `extra`, and validate naming after the public product name is selected.

### 26.5 Server Implementation

**Recommended direction:** Study Yerd's behavior, boundaries, and protocols before implementing the Go proxy, DNS integration, and runtime manager. Interoperate only through a stable executable or language neutral contract.

**Still required:** Compare daemon boundaries, licenses, project configuration, platform support, protocol stability, and contribution feasibility.

### 26.6 Code Quality Integration

**Recommended direction:** Execute Mago as a managed tool instead of creating another parser, formatter, linter, and analyzer.

**Still required:** Define the managed executable version, capability discovery, configuration, output, and failure contract.

### 26.7 Composer Acceleration

**Recommended direction:** Defer native installation and evaluate Libretto as a compatibility and performance reference when the Composer contract is established. Any in process native implementation belongs in the Go core.

**Still required:** Audit correctness, project maturity, licensing, executable interoperability, and reusable conformance fixtures.

### 26.8 Service Backend

**Recommended direction:** Support SQLite without a daemon, then select one shared SQL provider plus Redis and mail capture from validated Laravel demand. Keep an optional container adapter for exceptional projects.

**Still required:** Choose the first SQL engine, decide exact service scope, review licensing and update sources, and define backup and data migration behavior.

### 26.9 Windows Timing

**Recommended direction:** Design portable abstractions from the beginning, but do not block the macOS technical proof on Windows support.

**Still required:** Recruit real Windows testing before assigning Tier 1 status.

### 26.10 Telemetry

**Recommended direction:** No telemetry by default.

**Still required:** Decide whether explicit opt in aggregate usage data would ever provide enough value to justify the trust cost.

### 26.11 PHP FPM Sharing

**Recommended direction:** One PHP FPM master per PHP runtime version, one pool per application, `ondemand` workers, and one global worker ceiling.

**Still required:** Benchmark OPcache behavior, configuration reload cost, failure radius, process limits, and dedicated mode.

### 26.12 Laravel Environment Strategy

**Recommended direction:** Preserve an existing `.env` file, prefer supervised process environment injection for generated local credentials, and offer `.env.example` copying or persistent edits only through a reviewable approved setup action.

**Still required:** Test Laravel environment precedence, application key handling, encrypted environment files, testing environments, team conventions, Sail and DDEV values, and interactions with cached configuration.

### 26.13 Default Local Domain

**Recommended direction:** Use `.test` by default and support explicit aliases.

**Still required:** Decide wildcard resolver implementation, multisite domain mapping, certificate scope, and conflict behavior with Valet, Herd, DDEV, and Yerd.

### 26.14 Sail and DDEV Migration Depth

**Recommended direction:** Import common Laravel service, runtime, URL, and process configuration first. Inventory unsupported custom behavior without translating arbitrary Docker Compose.

**Still required:** Define the exact supported Sail Compose and `.ddev/config.yaml` fields and determine whether PHPX may invoke existing export commands during an approved migration.

### 26.15 Laravel Support Policy

**Recommended direction:** Begin with the current maintained Laravel release and the PHP branches permitted by its Composer constraints, then add older Laravel releases only through complete fixtures.

**Still required:** Define the exact Laravel release window, patch update policy, package compatibility warnings, and behavior for applications on unsupported framework or PHP versions.

### 26.16 Untrusted Project Isolation

**Recommended direction:** Document native mode as trusted local development and design a future dedicated or container backend behind the same project interface.

**Still required:** Define the threat model, dedicated database behavior, filesystem boundaries, and user experience for switching backends.

### 26.17 Core Implementation Language, Resolved

**Accepted decision:** Go for the native control plane. PHP remains responsible for Composer, Artisan, Laravel, WordPress, WP CLI, framework execution, project scripts, and compatibility probes.

**Validation required:** Complete the Milestone 0 Go proof and publish its resource measurements before Milestone 1 expands. Reconsider the language only if profiling shows that Go itself prevents PHPX from meeting a published requirement.

### 26.18 Product Differentiation

**Recommended direction:** Position PHPX as an open, headless, locked, native Laravel environment control plane that uses Composer and Artisan directly. Do not position it as another local HTTPS switcher or Composer replacement.

**Still required:** Pass the Section 1.9 necessity test and show a material workflow or resource advantage over Herd, Valet, Sail, and DDEV.

### 26.19 Managed JavaScript Runtime

**Recommended direction:** Detect and supervise project declared frontend commands while initially integrating with an existing Node or Bun installation selected by the project.

**Still required:** Determine whether clean machine Laravel onboarding requires PHPX to acquire Node.js and Corepack, or whether integration with existing version managers provides a smaller and more compatible boundary.

### 26.20 Later WordPress Configuration

**Recommended direction:** When the WordPress adapter begins, detect existing environment aware configuration first. Otherwise propose an explicit reviewable local integration without silently rewriting `wp-config.php`.

**Still required:** Test common agency conventions, local ignored configuration files, Bedrock environment files, DDEV generated includes, hardcoded production constants, and legacy PHP policies during the WordPress milestones.

## 27. Required Research Before Implementation

1. Interview at least fifteen independent Laravel developers or five Laravel teams using Herd, Valet, Sail, DDEV, or custom local environments.
2. Record the repeated problems, current workarounds, switching barriers, required platforms, team size, services, and willingness to adopt an open headless tool.
3. Build a current capability matrix for Herd Basic, Herd Pro, Valet, Sail, DDEV, Yerd, Mise, and PHPX using official documentation and hands on verification.
4. Inventory representative Laravel applications across current framework versions, PHP branches, Composer features, SQL engines, Redis, queues, mail, frontend tooling, and custom processes.
5. Benchmark native and container workflows with representative 1 application, 10 application, and 15 application workloads on the Apple Silicon reference machine.
6. Measure time from clean clone to trusted local route and passing test command for each comparison workflow.
7. Prove or reject the hypothesis that one locked PHPX environment can serve both local macOS setup and Linux continuous integration validation.
8. Review Laravel installation, configuration, Artisan, queue, scheduler, cache, database, testing, Vite, Sail, Octane, Reverb, Horizon, Pail, and package development behavior.
9. Prototype PHP FPM `ondemand` pools across 50 configured projects and multiple PHP versions.
10. Measure one PHP FPM master per PHP version against one master per application.
11. Prototype a global PHP worker ceiling and first request wake behavior.
12. Define Laravel application, package repository, application root, and public document root detection without executing project code.
13. Define safe `.env`, `.env.example`, `APP_KEY`, service credential, and configuration cache behavior.
14. Audit SQLite and candidate SQL, Redis, and mail artifacts, licenses, configuration, logical backup, and upgrade paths on macOS arm64.
15. Decide whether frontend runtime acquisition belongs inside PHPX or should integrate with existing Node and Bun managers.
16. Review Composer source and documented plugin contracts.
17. Build a Composer Semver conformance suite for PHP runtime requests.
18. Inspect Composer lock platform metadata across representative Laravel projects.
19. Audit StaticPHP artifacts, manifests, build recipes, and licenses.
20. Audit PIE target selection and noninteractive workflows.
21. Review Yerd architecture, behavior, protocols, and collaboration paths without assuming direct Rust library reuse.
22. Define a managed Mago executable integration contract.
23. Review Libretto compatibility tests, cache architecture, and executable interoperability as references for a possible future Go implementation.
24. Compare existing `.php-version` behavior across version managers.
25. Define runtime artifact, project execution, secret handling, and shared native service threat models.
26. Measure common PHP runtime and extension combinations across the Laravel fixture set and public Composer projects.
27. Validate current Laravel fixtures on a clean macOS arm64 environment.
28. Build and measure the Milestone 0 Go control plane proof.
29. Profile daemon memory, garbage collection, goroutine lifecycle, route registration, child process supervision, and release complexity against the published budgets.
30. Measure a PHP based discovery and command runner as the ecosystem compatibility baseline.
31. Record the Go validation evidence before Milestone 1 expands and investigate Rust only if the Go runtime is proven to be the limiting factor.
32. Make and record an explicit build, narrow, contribute elsewhere, or stop decision before Milestone 1.

## 28. Reference Sources

Composer documentation

https://getcomposer.org/doc

Composer introduction and scope

https://getcomposer.org/doc/00-intro.md

Composer platform dependencies

https://getcomposer.org/doc/articles/composer-platform-dependencies.md

Composer scripts

https://getcomposer.org/doc/articles/scripts.md

Composer plugins

https://getcomposer.org/doc/articles/plugins.md

PHP downloads and binary availability

https://www.php.net/downloads.php

PHP supported versions

https://www.php.net/supported-versions.php

PIE documentation

https://php.github.io/pie

Mago

https://github.com/carthage-software/mago

Yerd

https://github.com/forjedio/yerd

StaticPHP

https://static-php.dev

Libretto

https://github.com/libretto-pm/libretto

Laravel Valet

https://laravel.com/docs/13.x/valet

Laravel Sail

https://laravel.com/docs/13.x/sail

Laravel Herd

https://herd.laravel.com/docs

Laravel Herd installation and bundled tools

https://herd.laravel.com/docs/macos/getting-started/installation

Laravel Herd project configuration

https://herd.laravel.com/docs/macos/sites/herd-yaml

Laravel Herd services

https://herd.laravel.com/docs/macos/herd-pro-services/services

Laravel documentation

https://laravel.com/docs/13.x

uv

https://docs.astral.sh/uv

Go documentation

https://go.dev/doc/

2025 Go Developer Survey

https://go.dev/blog/survey2025

Go module reference

https://go.dev/ref/mod

Effective Go

https://go.dev/doc/effective_go

Go garbage collector guide

https://go.dev/doc/gc-guide

Go race detector

https://go.dev/doc/articles/race_detector

Go fuzzing

https://go.dev/doc/security/fuzz/

Go diagnostics

https://go.dev/doc/diagnostics

Go vulnerability management

https://go.dev/doc/security/vuln/

Rust documentation for alternative evaluation

https://doc.rust-lang.org/book/

Bun standalone executables

https://bun.sh/docs/bundler/executables

PHP PHAR documentation

https://www.php.net/manual/en/book.phar.php

DDEV architecture

https://docs.ddev.com/en/stable/users/usage/architecture/

DDEV performance on macOS

https://docs.ddev.com/en/stable/users/install/performance/

DDEV WordPress quickstart

https://docs.ddev.com/en/stable/users/quickstart/

DDEV configuration

https://docs.ddev.com/en/stable/users/configuration/config/

WordPress requirements

https://wordpress.org/about/requirements/

WordPress PHP compatibility

https://make.wordpress.org/hosting/handbook/server-environment/

WP CLI command reference

https://developer.wordpress.org/cli/commands/

PHP FPM configuration

https://www.php.net/manual/en/install.fpm.configuration.php

## 29. Refinement Readiness

This draft is ready for a dedicated refinement pass focused first on Laravel product necessity, competitive differentiation, and the one workflow PHPX must perform materially better. Technical refinement should then cover Go package boundaries, daemon protocol design, runtime artifact strategy, Composer compatibility, Laravel environment safety, public MVP boundaries, configuration format, platform priorities, and open source governance.

The specification should not be converted into implementation phases until the Section 1.9 necessity test and Milestone 0 research decisions have been reviewed. A build, narrow, contribute elsewhere, or stop decision must be recorded first because existing Laravel tools already cover much of the visible feature surface.
