# PHPX: WordPress First PHP Toolchain

## Document Status

**Working product name:** PHPX

**Status:** Draft, ready for structured refinement

**Created:** July 15, 2026

**Document purpose:** Define a WordPress first, Composer compatible, Rust based PHP toolchain that can install and select PHP runtimes, run many local sites without one container stack per site, synchronize project environments, execute isolated tools, and provide local development services through one command line interface.

The name PHPX is provisional. Public naming, package namespace availability, domain availability, and trademark review remain open decisions.

## 1. Product Definition

### 1.1 Goal

Create one fast, dependable command line tool that can take a WordPress or PHP project from source checkout to a working local development environment. The first product wedge is a lean native alternative to DDEV for developers who manage many WordPress sites. Composer remains the canonical PHP package manager whenever a project uses Composer.

### 1.2 Product Promise

From a clean machine, a WordPress developer should eventually be able to run:

```bash
git clone git@github.com:company/wordpress-site.git
cd wordpress-site
phpx up
```

PHPX should then:

1. Detect a classic WordPress, Bedrock, Composer based WordPress, Laravel, Symfony, or generic PHP project.
2. Determine the required PHP version from project configuration, Composer metadata, or WordPress compatibility data.
3. Install the correct PHP runtime when necessary.
4. Enable the PHP extensions expected by WordPress and any explicit project requirements.
5. Install missing dynamic extensions through a compatible extension provider.
6. Install and invoke WP CLI and Composer when needed.
7. Run `composer install` without altering Composer semantics when Composer metadata is present.
8. Allocate a project database inside a shared native database service by default.
9. Start demand driven PHP workers without creating a dedicated container stack.
10. Serve the project through a local domain with trusted TLS.
11. Start only the additional services and processes declared by the project.
12. Report one clear success state or one actionable failure state.

### 1.3 Problem Statement

Modern PHP package management is standardized around Composer, but WordPress development environments remain fragmented and frequently depend on a container stack for each site. DDEV provides a strong and familiar experience, but its architecture normally creates one web container and one database container per running project. On macOS, container file sharing can also require a synchronization layer to recover acceptable filesystem performance.

That architecture becomes expensive for a WordPress developer responsible for dozens of client sites. A representative workload for PHPX is approximately 50 registered sites with 10 to 15 sites available during an active workday. The current pain is severe machine slowdown even on an Apple Silicon workstation with 64 GB of memory.

Developers often need separate tools or manual instructions for:

1. Installing PHP.
2. Switching between PHP versions.
3. Matching command line PHP with web server PHP.
4. Installing native extensions.
5. Managing Composer itself.
6. Isolating global PHP tools.
7. Running local domains and TLS.
8. Starting databases, queues, caches, mail capture, and asset processes.
9. Importing, exporting, snapshotting, and restoring WordPress databases.
10. Running WP CLI against the correct site and runtime.
11. Supporting classic WordPress, Bedrock, and WordPress multisite layouts.
12. Keeping dozens of known sites available without dozens of persistent web and database stacks.
13. Reproducing the same environment in continuous integration.
14. Diagnosing differences between machines.

This fragmentation creates slow onboarding, environment drift, global dependency conflicts, heavy resource use, brittle setup documentation, and a gap between local and automated environments.

### 1.4 Opportunity

PHPX can become the compatibility focused control plane around the existing PHP ecosystem. Its first job is to deliver a leaner DDEV style experience for WordPress without making containers the default unit of isolation. It does not need to replace PHP, WordPress, WP CLI, Composer, PIE, PHPUnit, Pest, PHPStan, Rector, Mago, MariaDB, MySQL, PostgreSQL, Redis, or framework tooling. It needs to make those components installable, selectable, reproducible, and understandable through one coherent interface.

The strongest initial value proposition is not merely faster Composer execution. It is the ability to keep a large WordPress portfolio locally available with a small shared native footprint, then reproduce each project environment from source and configuration.

### 1.5 Target Users

1. WordPress developers managing many client sites locally.
2. WordPress agencies that need a repeatable team environment without one container stack per site.
3. Classic WordPress, Bedrock, and Composer based WordPress teams.
4. Laravel developers working across multiple applications and PHP versions.
5. Symfony developers working across multiple applications and PHP versions.
6. Framework neutral PHP developers and library maintainers.
7. New PHP developers who do not yet have a configured local environment.
8. Continuous integration maintainers who need deterministic PHP environments.
9. Open source maintainers who want contributors productive with minimal setup.
10. Teams replacing machine specific setup documents with executable project configuration.

### 1.6 Framework Priority

Development and product validation must follow this order:

1. WordPress.
2. Laravel.
3. Symfony.
4. Generic PHP sites and libraries.

WordPress is not merely another framework adapter. It is the first product market, the first local environment workflow, and the source of the initial resource efficiency requirements. The architecture must remain capable of supporting the remaining PHP ecosystem without forcing WordPress assumptions into the core runtime manager.

### 1.7 Founding Scale Requirement

PHPX must be designed around the following real workload:

1. At least 50 registered WordPress projects on one machine.
2. At least 15 sites available concurrently without 15 web containers and 15 database containers.
3. One shared router, one shared local certificate authority, and shared database engines where compatible.
4. No dedicated process for a registered site that has not received traffic and has no declared background process.
5. PHP workers created on demand and retired after an idle interval.
6. Asset watchers, queue workers, cron runners, and similar processes started only when explicitly declared.
7. Native host filesystem access with no duplicate project tree and no synchronization daemon required for normal macOS operation.

## 2. Product Principles

### 2.1 WordPress First, Ecosystem Compatible

Classic WordPress projects without `composer.json` must work as first class projects. Bedrock and other Composer based WordPress projects must receive the full Composer integration. Laravel, Symfony, and generic PHP support must build on the same runtime, artifact, service, and process foundations.

### 2.2 Composer Remains Canonical When Present

`composer.json` and `composer.lock` remain the source of truth for PHP package dependencies, package resolution, autoloading, Composer plugins, Composer scripts, and declared PHP platform requirements.

PHPX must never create a parallel PHP package ecosystem or require projects to migrate their package metadata.

### 2.3 Existing Projects Work Without Migration

A WordPress or Composer project should receive useful PHPX behavior without restructuring its source tree. Optional PHPX configuration should unlock stronger reproducibility, local serving, services, and process orchestration.

### 2.4 Projects Remain Usable Without PHPX

Removing PHPX from a machine must not make the project structurally unusable. A developer with a compatible PHP installation, web server, database, WP CLI, and Composer when applicable must still be able to use the normal project workflow.

### 2.5 Native by Default, Containers by Exception

The default WordPress path must use native PHP, native filesystem access, shared native database services, and a shared proxy. Containers may be supported as an optional compatibility backend for projects that genuinely require operating system isolation or custom infrastructure.

### 2.6 Shared Services, Isolated Project Data

Compatible WordPress projects should share one MariaDB or MySQL server process while receiving separate databases, users, credentials, import history, and backups. A project may request a dedicated service instance when version or isolation requirements demand it.

### 2.7 Compatibility Before Native Replacement

PHPX must delegate to Composer whenever native behavior cannot preserve Composer semantics. Unsupported behavior must trigger a transparent fallback, never an approximation that appears successful.

### 2.8 Rust Owns the Control Plane

Rust should be the default implementation language for project discovery, artifact management, version selection, caching, process supervision, proxying, diagnostics, and the command line experience.

PHP remains the correct runtime for Composer plugins, Composer callbacks, application scripts, and PHP tools.

### 2.9 Deterministic by Default

When a lock file is present, two supported machines with the same target should resolve the same PHP patch version, Composer version, extension providers, extension versions, and tool versions.

### 2.10 Safe Global Coexistence

PHPX must coexist with system PHP, Homebrew PHP, Herd, Valet, Docker, Mise, and other version managers. It must not silently replace, unlink, delete, or modify external installations.

### 2.11 Framework Neutral Core

The core must understand PHP runtimes, artifacts, services, and executable project environments independently of any framework. WordPress receives the first and deepest application adapter. Laravel, Symfony, and generic PHP adapters follow without duplicating the core lifecycle.

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
5. WordPress project detection and local environment adaptation.
6. WP CLI acquisition and runtime selection.
7. Composer acquisition and runtime selection.
8. Shared database engine lifecycle and project database isolation.
9. WordPress database import, export, snapshot, and restore coordination.
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
2. WordPress core, plugins, and themes retain their normal application behavior.
3. WP CLI performs WordPress administration commands.
4. PIE resolves and builds supported dynamic PHP extensions.
5. Mago may provide formatting, linting, and static analysis integrations.
6. Existing tools such as PHPUnit, Pest, PHPStan, and Rector continue to execute as their own packages.
7. MariaDB, MySQL, PostgreSQL, Redis, and similar services continue to use their upstream engines.
8. Framework command line tools retain framework specific behavior.

## 4. Ecosystem Context

PHPX enters an ecosystem with several valuable adjacent projects.

### 4.1 DDEV

DDEV is the primary experience benchmark and the clearest architectural contrast. DDEV uses one web container and one database container per running project, plus shared router and SSH agent containers. On macOS and Windows, DDEV commonly uses Mutagen to work around container filesystem performance limitations. PHPX should preserve DDEV strengths such as project configuration, local URLs, database import and export, WP CLI integration, web server choice, and broad PHP version support while replacing the default per project container topology with shared native processes.

### 4.2 Composer

Composer is the established package manager and defines the compatibility contract that PHPX must preserve.

### 4.3 PIE

PIE is the official PHP extension installer and the successor to PECL. PHPX should integrate with PIE instead of inventing a second extension package format.

### 4.4 Mago

Mago provides Rust based formatting, linting, and static analysis. PHPX should evaluate direct integration or managed execution rather than recreating those capabilities.

### 4.5 Laravel Valet and Laravel Herd

Valet and Herd establish familiar concepts such as parked directories, linked sites, trusted local domains, PHP isolation, and service management. PHPX should preserve the best parts of that user experience while remaining framework neutral and headless.

### 4.6 Yerd

Yerd already provides a Rust based local PHP environment with PHP version management, local domains, TLS, services, and Composer support. A technical and product comparison is required before PHPX builds its server layer. Integration, shared components, or contribution may be better than duplicating mature functionality.

### 4.7 StaticPHP

StaticPHP provides an important possible source of portable PHP artifacts for early development. Artifact provenance, supported features, licenses, extension coverage, PHP FPM availability, and long term reliability must be reviewed before production dependence.

### 4.8 Libretto

Libretto explores Composer compatible package installation in Rust. Its compatibility boundaries, cache model, resolver, autoload generation, and project health should be evaluated before PHPX attempts any native Composer acceleration.

## 5. User Personas

### 5.1 WordPress Portfolio Developer

Maintains approximately 50 client sites and may need 10 to 15 available during a workday. Wants instant local URLs, correct PHP versions, databases, WP CLI, mail capture, and imports without keeping a container pair alive for every site.

### 5.2 WordPress Agency Maintainer

Needs repeatable configuration for classic WordPress, Bedrock, and multisite projects across a team. Needs fast onboarding, database and upload workflows, and predictable local domains.

### 5.3 Application Developer

Works on several PHP applications with different runtime and extension requirements. Needs automatic project switching without changing the global system runtime.

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

### 6.1 Existing WordPress Site on a Clean Machine

```text
Clone or copy WordPress site
    ↓
Run phpx init or phpx up
    ↓
PHPX detects classic WordPress, Bedrock, or Composer based WordPress
    ↓
PHPX installs a compatible runtime, WP CLI, and Composer when present
    ↓
PHPX installs or validates extensions
    ↓
PHPX creates an isolated database and imports a configured dump when present
    ↓
PHPX registers the local domain and trusted TLS
    ↓
Developer opens the site or runs phpx wp
```

### 6.2 WordPress Portfolio Workday

```text
PHPX knows 50 registered sites
    ↓
Developer opens or requests 15 sites during the day
    ↓
One shared proxy routes every domain
    ↓
Compatible sites use one shared database engine with isolated databases
    ↓
PHP FPM workers start on request and retire after idle timeout
    ↓
Sites remain addressable without 15 persistent container stacks
```

### 6.3 DDEV Project Import

```text
Developer enters a project containing .ddev/config.yaml
    ↓
Run phpx import ddev
    ↓
PHPX maps project type, document root, PHP version, database engine, hostnames, and upload directories
    ↓
PHPX reports custom containers, add ons, and hooks that cannot be mapped
    ↓
Developer reviews and writes phpx.toml
    ↓
Original .ddev configuration remains untouched
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

### 6.5 Running WP CLI

```text
Run phpx wp plugin status
    ↓
PHPX selects the site PHP runtime and project database environment
    ↓
PHPX invokes a managed WP CLI version at the detected WordPress path
    ↓
The command operates on the intended site without a container shell
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
PHPX starts PHP FPM and declared processes
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
Run phpx run composer test
```

## 7. Success Criteria

### 7.1 Technical Proof Criteria

* [ ] A macOS arm64 machine with no usable PHP, Composer, WP CLI, web server, or local database can run a classic WordPress site through PHPX.
* [ ] PHPX installs a managed PHP runtime without changing the system PHP installation.
* [ ] PHPX installs and invokes WP CLI through the managed runtime.
* [ ] PHPX installs and invokes Composer through the managed runtime for a Bedrock fixture.
* [ ] A classic WordPress fixture loads through trusted local HTTPS and can access its database.
* [ ] A Bedrock fixture loads through trusted local HTTPS and can access its database.
* [ ] WordPress permalinks work through the default server adapter.
* [ ] Database import and export round trips preserve the technical proof fixture.
* [ ] Repeating `phpx up` without project changes is idempotent.
* [ ] Interrupted runtime downloads cannot leave an installation marked complete.
* [ ] Every downloaded artifact is verified before execution.
* [ ] `phpx doctor` reports enough information to diagnose the tested failure cases.
* [ ] The architecture demonstrates 15 registered WordPress fixtures available through one shared proxy and one shared compatible database engine.

### 7.2 WordPress Public MVP Criteria

* [ ] macOS arm64 is supported and benchmarked on Apple Silicon.
* [ ] Classic WordPress, Bedrock, and Composer based WordPress projects are supported.
* [ ] PHP version selection honors PHPX configuration, `.php-version`, Composer requirements when present, and detected WordPress compatibility.
* [ ] PHPX manages supported PHP branches and exact locked patch versions.
* [ ] Required built in extensions are detected and validated.
* [ ] Supported dynamic extensions can be installed through PIE.
* [ ] WP CLI runs against the detected WordPress root with the selected PHP runtime and database environment.
* [ ] Composer plugins and scripts behave exactly as they do through direct Composer invocation.
* [ ] Isolated Composer tool execution works without modifying the project.
* [ ] `phpx.lock` can be committed and enforced with frozen mode.
* [ ] Local sites resolve through configured `.test` domains and trusted TLS.
* [ ] One shared MariaDB or MySQL engine can host isolated databases and users for compatible projects.
* [ ] Database create, import, export, snapshot, restore, and remove operations are available.
* [ ] A shared mail capture service can receive mail from multiple sites while preserving site identity.
* [ ] Standard single site WordPress rewrites and permalinks work without Apache or Nginx.
* [ ] Custom `.htaccess` behavior is detected and either mapped, warned about, or routed through an explicit compatibility backend.
* [ ] Common DDEV WordPress configuration can be imported without modifying `.ddev` files.
* [ ] Commands required by local automation and continuous integration support noninteractive execution.
* [ ] Machine readable output is available for status, resolution, and diagnostics.
* [ ] The project publishes clear security, support, and artifact provenance policies.
* [ ] Fifty sites can remain registered without one persistent process, container, or database engine per site.
* [ ] Fifteen sites can remain addressable concurrently without 15 web containers and 15 database containers.
* [ ] A published benchmark compares PHPX and DDEV on the same WordPress portfolio workload.

### 7.3 Portfolio Performance Criteria

* [ ] Registering an idle site adds no dedicated long running process.
* [ ] An idle site has no PHP worker after the configured idle timeout.
* [ ] Compatible WordPress sites reuse a shared database engine rather than creating one engine per site.
* [ ] All sites reuse one proxy, one DNS integration, one local certificate authority, and one mail capture service.
* [ ] WordPress code executes directly from the host filesystem with no duplicate synchronized project tree.
* [ ] Opening an idle registered site creates required PHP workers automatically.
* [ ] Closing the browser does not keep unnecessary PHP workers alive indefinitely.
* [ ] PHP command line and web requests use the same project PHP version.
* [ ] Local service ports never collide silently.
* [ ] `phpx status` accurately reports every registered site, active worker, managed service, and endpoint.
* [ ] The benchmark records idle memory, active memory, process count, time to first response, steady response time, and disk use.

### 7.4 Adoption Criteria

* [ ] Existing WordPress projects can try PHPX without restructuring the WordPress installation.
* [ ] Existing Composer projects can try PHPX without converting dependency files.
* [ ] A common DDEV WordPress project can generate a reviewable PHPX configuration through one import command.
* [ ] A project can stop using PHPX without reversing package metadata changes.
* [ ] Setup documentation for a representative WordPress site can be reduced to installation, import or initialization, and one startup command.
* [ ] The compatibility policy and fallback behavior are documented and tested.

## 8. Scope Boundaries

### 8.1 Technical Proof Scope

1. macOS arm64.
2. Rust command line application.
3. Classic WordPress and Bedrock project discovery.
4. WordPress and Composer PHP requirement resolution.
5. Managed PHP installation from a vetted artifact source.
6. Managed WP CLI installation.
7. Managed Composer installation for Composer based projects.
8. Shared Rust proxy and trusted local TLS.
9. Shared MariaDB or MySQL engine with isolated project databases.
10. Demand driven PHP FPM pools.
11. `init`, `up`, `down`, `status`, `wp`, `sync`, `run`, `composer`, `db`, and `doctor` commands.
12. Artifact checksums and atomic installs.
13. Classic WordPress and Bedrock fixtures.
14. A 15 site scale fixture derived from the representative portfolio workload.

### 8.2 Public MVP Scope

1. macOS arm64.
2. Classic WordPress, Bedrock, and Composer based WordPress.
3. Project configuration and environment lock files.
4. PHP runtime lifecycle management.
5. Managed WP CLI and Composer.
6. Shared proxy, DNS integration, TLS, database, and mail capture.
7. Demand driven PHP FPM.
8. Standard WordPress rewrite behavior.
9. Standard extension validation.
10. PIE integration for a documented supported set.
11. WordPress database and upload directory workflows.
12. Common DDEV configuration import.
13. Isolated PHP tools.
14. Continuous integration support for package and tool commands.
15. Shared artifact cache.
16. Stable human and machine readable output.
17. Signed releases and a documented update path.
18. Published portfolio resource benchmarks.

### 8.3 Later Scope

1. Additional operating systems and architectures.
2. Laravel adapter.
3. Symfony adapter.
4. Generic PHP site adapter.
5. WordPress multisite with subdirectory and subdomain modes.
6. Native Apache compatibility mode for projects dependent on custom `.htaccess` behavior.
7. Broader DDEV add on and hook migration.
8. Additional native service providers.
9. Project process graphs beyond the WordPress essentials.
10. Graphical clients.
11. Native locked Composer installation acceleration.
12. Shared remote caches and enterprise mirrors.

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
11. Full compatibility with arbitrary Docker Compose overrides in the public MVP.
12. Parsing and emulating every possible Apache `.htaccess` directive in the Rust proxy.
13. Starting Node, asset watchers, cron loops, or queue workers for every registered site by default.

## 9. Functional Requirements

### 9.1 Project Discovery

**FR 1.1:** PHPX must discover the current project from the current working directory or an explicit path.

**FR 1.2:** Discovery must recognize `phpx.toml`, `phpx.lock`, `composer.json`, `composer.lock`, `.php-version`, `.ddev/config.yaml`, `wp-config.php`, `wp-load.php`, `wp-settings.php`, `wp-content`, Bedrock structure, and the repository root.

**FR 1.3:** Directory traversal must stop at the selected project boundary and must not accidentally inherit configuration from an unrelated parent project.

**FR 1.4:** Commands must accept an explicit project path for automation.

**FR 1.5:** Running PHPX outside a project must either perform a valid global operation or return an actionable message.

**FR 1.6:** PHPX must expose the discovered project root through `phpx status` and machine readable output.

**FR 1.7:** PHPX must distinguish the repository root, PHP working directory, web document root, WordPress root, WordPress content directory, and upload directory.

**FR 1.8:** PHPX must detect classic WordPress and Bedrock without executing project supplied PHP during discovery.

**FR 1.9:** PHPX must read the installed WordPress core version from source metadata without booting WordPress.

**FR 1.10:** A WordPress plugin or theme repository without a complete WordPress installation must be identified as an extension project rather than incorrectly served as a site.

**FR 1.11:** Parked directory discovery must avoid descending into `vendor`, `node_modules`, `.git`, upload archives, backups, and known generated directories.

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

**FR 2.9:** For WordPress without Composer metadata, PHPX must combine explicit project configuration, detected WordPress core compatibility, current PHP support status, and the PHPX recommended default.

**FR 2.10:** WordPress compatibility and PHP security support must be reported separately. Compatibility with an end of life PHP version must never be presented as a security recommendation.

**FR 2.11:** PHPX must not infer plugin and theme compatibility with a newer PHP version when reliable metadata is unavailable.

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

**FR 3.11:** WordPress public MVP runtime artifacts must contain the PHP extensions in the documented WordPress baseline or report an unsupported baseline before serving the site.

**FR 3.12:** Legacy WordPress runtimes may be installable for client maintenance, but PHPX must distinguish installability from active security support.

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

**FR 5.2:** The plan must resolve the project adapter, runtime, runtime configuration, extensions, WP CLI, Composer when present, isolated tools required by configuration, database allocation, local site registration, and package installation action.

**FR 5.3:** `phpx sync` must use `composer install` when `composer.lock` exists.

**FR 5.4:** `phpx sync` must not run `composer update` implicitly.

**FR 5.5:** `phpx sync --frozen` must fail when `phpx.lock` is absent, stale, incompatible with the current target, or would change.

**FR 5.6:** `phpx sync --offline` must refuse network access and explain every missing cached artifact.

**FR 5.7:** A successful synchronization must finish with Composer platform requirement validation when Composer metadata exists, followed by adapter specific validation.

**FR 5.8:** Repeated synchronization without input changes must be idempotent.

**FR 5.9:** Partial failure must preserve the last known valid environment.

**FR 5.10:** PHPX must distinguish resolution, download, installation, Composer, script, and validation failures.

**FR 5.11:** A classic WordPress project without Composer must synchronize successfully without creating Composer files.

**FR 5.12:** `phpx up` must synchronize prerequisites before exposing the local route, but it must not update WordPress core, plugins, themes, or Composer packages implicitly.

### 9.6 Extension Management

**FR 6.1:** PHPX must distinguish built in PHP extensions from separately installed dynamic extensions.

**FR 6.2:** PHPX must infer required extension names from Composer platform requirements and a versioned WordPress extension baseline.

**FR 6.3:** A broad runtime artifact may contain common built in extensions while project configuration controls which optional extensions are enabled.

**FR 6.4:** PHPX must integrate with PIE for supported dynamic extension installation.

**FR 6.5:** When one extension name has multiple possible providers, PHPX must request an explicit selection and lock it.

**FR 6.6:** Dynamic extension artifacts must be isolated by operating system, architecture, PHP version, PHP API, thread model, build flags, and extension version.

**FR 6.7:** PHPX must never load an extension built for an incompatible PHP ABI.

**FR 6.8:** PHPX must expose the active INI scan directory and every PHPX managed INI fragment.

**FR 6.9:** Development extensions such as Xdebug must support project scoped activation without modifying the base runtime.

**FR 6.10:** Compiling an extension must display required native build dependencies before requesting privileged installation.

**FR 6.11:** The WordPress baseline must distinguish required, recommended, and optional extensions.

**FR 6.12:** Site Health and PHPX diagnostics should agree on loaded extension visibility wherever WordPress exposes equivalent checks.

### 9.7 Isolated Tool Management

**FR 7.1:** `phpx tool run <package>` must run a Composer distributed tool without adding it to the current project.

**FR 7.2:** Each tool must receive an isolated Composer home and dependency environment.

**FR 7.3:** Tools must share immutable downloaded artifacts where safe without sharing mutable vendor state.

**FR 7.4:** Tool runtime selection must satisfy both the tool and current project context.

**FR 7.5:** `phpx tool install`, `list`, `update`, and `remove` must manage persistent tool installations.

**FR 7.6:** A tool may expose one or more executable aliases through PHPX managed shims.

**FR 7.7:** Tool resolution must be lockable for continuous integration and team usage.

**FR 7.8:** Project installed binaries in `vendor/bin` must take precedence when the user explicitly requests the project tool.

### 9.7.1 WordPress and WP CLI

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

**FR 8.2:** The child path must place the selected PHP runtime, selected WP CLI and Composer executables, PHPX tool shims, and `vendor/bin` in documented order.

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

**FR 9.9:** WordPress database credentials generated by PHPX must live in uncommitted local state or an operating system credential store.

**FR 9.10:** PHPX must support Bedrock environment files without committing local secrets.

**FR 9.11:** For classic WordPress, PHPX must offer reviewable local configuration strategies and must not silently rewrite `wp-config.php`.

**FR 9.12:** A generated `wp-config-phpx.php` integration file must be local, clearly marked, and included only after explicit approval or an existing project convention allows it.

### 9.10 Local Site Management

**FR 10.1:** The local server layer must support WordPress document roots first, followed by Laravel, Symfony, and generic PHP adapters.

**FR 10.2:** Each site must use its selected PHP version for both command line and web execution.

**FR 10.3:** The preferred architecture is a Rust reverse proxy forwarding to a site pool managed by the PHP FPM master for the selected PHP version.

**FR 10.4:** `phpx link` must register the current project explicitly.

**FR 10.5:** `phpx park <directory>` must discover eligible child projects without registering unrelated directories.

**FR 10.6:** `phpx secure` must issue a trusted local certificate after the local certificate authority is installed.

**FR 10.7:** Binding privileged ports, installing resolver configuration, and trusting a local certificate authority must occur through explicit setup actions.

**FR 10.8:** Normal site operation must not require repeated privilege elevation.

**FR 10.9:** Sites must bind to loopback by default.

**FR 10.10:** Port and socket allocation must be deterministic and collision aware.

**FR 10.11:** One PHP FPM master may serve multiple site pools using the same PHP runtime version.

**FR 10.12:** WordPress site pools must use the FPM `ondemand` process manager so idle pools can have zero child workers.

**FR 10.13:** PHPX must enforce a configurable global PHP worker limit and a configurable per site worker limit.

**FR 10.14:** An idle timeout must retire unused PHP workers without unregistering the site route.

**FR 10.15:** Standard WordPress front controller rewrites, static assets, uploads, and pretty permalinks must work through the default proxy adapter.

**FR 10.16:** PHPX must inspect `.htaccess` and report directives outside the supported WordPress rewrite subset.

**FR 10.17:** Projects that depend on arbitrary Apache behavior must be able to select a documented compatibility backend when one becomes available.

**FR 10.18:** The shared proxy must wake or route an idle registered site without requiring a manual start command for ordinary web requests.

**FR 10.19:** WordPress multisite subdomain mode must be able to register wildcard local DNS and certificates before it is declared supported.

**FR 10.20:** Site routes must remain cheap registry entries when no PHP worker or project process is active.

### 9.11 Development Services

**FR 11.1:** PHPX must manage upstream MariaDB or MySQL and mail capture for the WordPress public MVP. It may later manage PostgreSQL, Redis, search, storage, and additional engines.

**FR 11.2:** Every service must have an explicit version, data path, configuration path, port policy, and health check.

**FR 11.3:** Compatible WordPress projects should share a database engine process while using separate databases, users, credentials, and backup histories.

**FR 11.4:** Services must bind to loopback by default.

**FR 11.5:** `phpx services up`, `down`, `status`, `logs`, and `remove` must expose lifecycle operations.

**FR 11.6:** Removal of service data must require explicit confirmation or a force flag.

**FR 11.7:** A container backend may be offered as an adapter, but containers must not be required for core runtime management.

**FR 11.8:** A project requiring a different database engine version may use another shared versioned engine instance.

**FR 11.9:** A project may request a dedicated database engine for compatibility or isolation, but dedicated mode is not the default.

**FR 11.10:** Stopping a site must not delete its database or uploads.

**FR 11.11:** The shared mail service must tag or filter messages by originating site.

**FR 11.12:** Redis and other optional services must not start merely because PHPX knows about a site.

### 9.11.1 WordPress Database and File Workflows

**FR 11W.1:** `phpx db create` must allocate a project database and least privilege project user in the selected shared engine.

**FR 11W.2:** `phpx db import` must support common plain and compressed SQL dump formats.

**FR 11W.3:** `phpx db export` must create a portable logical dump without stopping unrelated sites.

**FR 11W.4:** `phpx db snapshot` and `phpx db restore` must maintain project scoped local recovery points.

**FR 11W.5:** Database deletion must display the site, engine, database name, and backup state before confirmation.

**FR 11W.6:** URL search and replace must be an explicit WP CLI backed operation and must not happen silently during import.

**FR 11W.7:** PHPX must support configurable WordPress upload directories for classic WordPress and Bedrock.

**FR 11W.8:** File import and export must preserve the source project tree unless the user chooses a managed external upload store.

**FR 11W.9:** Database credentials must be unique per project by default.

**FR 11W.10:** Project database isolation is an organizational and access boundary, not a substitute for hostile code isolation.

### 9.12 Process Supervision

**FR 12.1:** `phpx up` must start declared project processes in dependency order.

**FR 12.2:** Processes must support health checks, restart policy, working directory, environment variables, and dependency declarations.

**FR 12.3:** PHPX must prefix or structure logs so concurrent process output remains understandable.

**FR 12.4:** Interruption must trigger graceful shutdown followed by bounded forced termination when required.

**FR 12.5:** `phpx status` must distinguish starting, healthy, unhealthy, stopped, and failed processes.

**FR 12.6:** An optional user scoped daemon may retain local site and service state, but core package and runtime commands must not require it.

**FR 12.7:** Registered sites must not automatically start asset watchers, Node processes, cron loops, or queue workers.

**FR 12.8:** A global scheduler must cap total PHP workers across every site and report queueing caused by the cap.

**FR 12.9:** PHPX must be able to sleep a site by stopping its optional project processes while leaving its route and data registered.

### 9.13 Diagnostics

**FR 13.1:** `phpx doctor` must inspect WordPress discovery, WordPress root, runtime selection, artifact integrity, PHP configuration, extensions, WP CLI, Composer when present, database access, rewrite compatibility, permissions, proxy state, DNS, TLS, services, ports, and process state as applicable.

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

### 9.17 DDEV Import and Coexistence

**FR 17.1:** `phpx import ddev` must read `.ddev/config.yaml` without modifying it.

**FR 17.2:** The importer must map project name, project type, document root, PHP version, database type and version, primary URL, additional hostnames, additional domains, upload directories, web server type, and common environment values where safe.

**FR 17.3:** Custom Docker Compose files, custom images, add ons, hooks, web build files, and custom Nginx or Apache configuration must be inventoried and classified as mapped, ignored by choice, or unsupported.

**FR 17.4:** Unsupported DDEV behavior must block automatic cutover when it may affect application behavior.

**FR 17.5:** PHPX must generate a reviewable proposed `phpx.toml` before writing it.

**FR 17.6:** An optional migration command may ask DDEV to export the project database and then import it into PHPX local state.

**FR 17.7:** URL replacement from `.ddev.site` to `.test` must require explicit approval and use WP CLI aware replacement.

**FR 17.8:** PHPX must not stop, delete, or alter the DDEV project until the user explicitly requests cleanup after validation.

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

**FR 18.9:** Resource benchmarks must use the founding workload of 50 registered sites and 15 concurrently addressable WordPress sites.

### 9.19 Framework Rollout

**FR 19.1:** WordPress requirements and fixtures must reach public MVP quality before Laravel becomes an official application adapter.

**FR 19.2:** Laravel support must reuse runtime, Composer, database, proxy, TLS, process, and diagnostics foundations.

**FR 19.3:** Symfony support must follow Laravel and reuse the same foundations.

**FR 19.4:** Generic PHP site support must allow an explicit document root and front controller without framework detection.

**FR 19.5:** Adding later adapters must not weaken WordPress portfolio performance guarantees.

## 10. Command Line Contract

### 10.1 Initial Command Surface

```text
phpx init
phpx import ddev
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

phpx wp

phpx db create
phpx db import
phpx db export
phpx db snapshot
phpx db restore
phpx db shell
phpx db remove

phpx mail open
phpx mail status

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
phpx services up
phpx services down
phpx services status
phpx services logs
phpx services remove

phpx adapter laravel
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

The following example demonstrates the intended WordPress boundary. Composer requirements remain in `composer.json` when the project uses Composer. PHPX configuration describes the local and reproducible environment around WordPress and those package requirements.

```toml
schema = 1

[project]
type = "wordpress"
root = "."
wordpress_root = "."

[php]
version = "8.4"
source = "managed"

[wordpress]
wp_cli = "2"

[composer]
version = "2"

[extensions.dev]
xdebug = "^3.4"

[site]
domain = "client-site.test"
tls = true
rewrite = "wordpress"

[database]
engine = "mariadb"
version = "10.11"
mode = "shared"

[files]
uploads = "wp-content/uploads"

[workers]
mode = "ondemand"
max_children = 2
idle_timeout = "10s"

[processes.assets]
command = "pnpm dev"
autostart = false

[tools]
phpstan = "phpstan/phpstan:^2"
```

### 11.1 Configuration Rules

1. `php.version` expresses an allowed request, not necessarily an exact patch.
2. The resolved exact patch belongs in `phpx.lock`.
3. Composer `require.php` must also allow the selected runtime when Composer metadata exists.
4. Detected WordPress compatibility must allow the selected runtime.
5. Composer `ext-*` requirements are inferred and must not need duplication.
6. A versioned WordPress extension baseline is inferred without duplication.
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
5. The PHP compatibility range for the detected WordPress core version.
6. The latest PHP branch that is supported by PHP and compatible with the detected application.

Higher precedence never permits violation of a lower level compatibility constraint. A pinned PHPX version that violates Composer or detected WordPress requirements must fail. PHPX must explain when WordPress remains compatible with a PHP branch that PHP itself no longer supports.

### 12.2 Synchronization Steps

1. Discover project files, application adapter, document root, WordPress root, and target platform.
2. Parse PHPX configuration and validate its schema.
3. Read WordPress core metadata without executing project code.
4. Parse Composer platform requirements when Composer metadata is present.
5. Load and validate the PHPX lock when present.
6. Resolve a supported and application compatible PHP runtime.
7. Resolve the runtime artifact and verify target compatibility.
8. Resolve built in and dynamic extension requirements.
9. Resolve WP CLI and Composer when required.
10. Resolve configured isolated tools.
11. Resolve database engine allocation and project database identity.
12. Resolve local domain, TLS, rewrite adapter, PHP FPM pool, and worker policy.
13. Produce an immutable execution plan.
14. Acquire per artifact and shared service locks.
15. Download missing artifacts into temporary storage.
16. Verify hashes, signatures, sizes, and manifests.
17. Atomically install artifacts into the managed store.
18. Generate project scoped PHP and PHP FPM configuration overlays.
19. Execute `composer install` through the selected runtime when `composer.lock` exists.
20. Execute Composer platform requirement validation when Composer metadata exists.
21. Validate WP CLI bootstrap and database connectivity for WordPress.
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
    ├── WordPress Adapter
    ├── PHP Runtime Manager
    ├── WP CLI Adapter
    ├── Composer Adapter
    ├── Extension Adapter
    ├── Database Allocator
    ├── Tool Manager
    ├── Artifact Store
    └── Lock Manager
    ↓
Command Runner

WordPress Local Control Plane
    ├── User Daemon
    ├── Rust HTTP and TLS Proxy
    ├── DNS Integration
    ├── PHP FPM Supervisor
    ├── Shared Database Supervisor
    ├── Resource Scheduler
    ├── Site Registry
    ├── Shared Mail Capture
    ├── Service Supervisor
    └── Process Log Router
```

### 13.2 Proposed Rust Workspace

```text
crates/
    phpx-cli
    phpx-core
    phpx-project
    phpx-adapters
    phpx-wordpress
    phpx-constraints
    phpx-runtime
    phpx-artifacts
    phpx-wp-cli
    phpx-composer
    phpx-extensions
    phpx-tools
    phpx-database
    phpx-ddev-import
    phpx-scheduler
    phpx-process
    phpx-server
    phpx-services
    phpx-daemon
    phpx-test-support
```

### 13.3 Dependency Direction

1. `phpx-core` defines shared domain types and errors.
2. Project, adapter, WordPress, constraint, runtime, artifact, WP CLI, Composer, extension, database, and tool crates depend on core abstractions.
3. WordPress depends on stable adapter interfaces and must not place application assumptions inside the runtime or artifact crates.
4. Server, database, scheduler, and service components depend on runtime and process abstractions, not the command line parser.
5. The command line crate composes capabilities but contains minimal business logic.
6. The daemon exposes a versioned local protocol and uses the same core services as the CLI.
7. Test support provides fixtures and fake artifact repositories without entering production dependencies.

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
2. The WordPress local environment uses one user scoped daemon for persistent routes, PHP FPM pools, resource scheduling, and shared services.
3. The daemon must never run as root.
4. Privileged setup must be handled by a narrowly scoped helper or explicit operating system command.
5. CLI and graphical clients must communicate with the daemon through a versioned authenticated local socket.

### 13.7 WordPress Portfolio Process Topology

The target topology for 50 registered sites is:

```text
One PHPX user daemon
    ├── One Rust HTTP and TLS proxy
    ├── One site registry and resource scheduler
    ├── One shared mail capture process
    ├── One shared MariaDB 10.11 engine for compatible sites
    ├── Additional shared database engines only when required
    ├── One PHP FPM master for each active PHP version
    │       ├── One configured pool per registered site using that version
    │       └── Zero child workers for each idle pool
    └── Optional project processes started only by declaration
```

The PHP FPM `ondemand` process manager allows a configured pool to have no child worker until a request arrives. PHPX must also set a global process ceiling so traffic across many sites cannot create an unbounded number of PHP workers.

PHP FPM pools are not a hostile code security boundary and may share an OPcache instance within one PHP FPM master. A future dedicated runtime mode may start a separate master for projects requiring stronger local separation.

### 13.8 Shared Database Topology

1. A database engine process is keyed by engine type, engine version, target, and relevant configuration profile.
2. Every project receives a separate database and database user.
3. Project credentials are stored locally and injected through an approved WordPress configuration strategy.
4. Stopping or sleeping a site never stops a shared engine that serves another project.
5. Removing a site registration never drops its database automatically.
6. A database engine may stop only when no registered project depends on it or the user explicitly requests a global service stop.
7. Engine upgrades require logical export and verified restore when binary storage formats are not compatible.

### 13.9 Native Filesystem Model

PHPX executes WordPress directly from the host project path. It does not copy the project into a virtual machine or container volume and does not require a file synchronization daemon for ordinary operation. Upload directories remain normal host paths unless the project explicitly selects a managed external path.

This model removes the container bind mount and synchronization costs that can affect DDEV performance on macOS. It also means PHPX does not provide container level filesystem isolation by default.

### 13.10 Implementation Language Decision

Rust is the preferred implementation language for the durable PHPX control plane. This includes the command line executable, user daemon, local proxy, resource scheduler, artifact verifier, runtime installer, process supervisor, and operating system integration.

This is not a goal to rewrite the PHP ecosystem in Rust. PHP remains the application language and the compatibility authority for Composer, WordPress, WP CLI, Laravel, Symfony, and project supplied scripts. PHPX should invoke those tools through the selected PHP runtime instead of approximating their behavior.

The choice of Rust must be justified by the product constraints and validated in Milestone 0. Rust is not a brand requirement. If a measured prototype shows that another language delivers the resource, safety, distribution, and maintenance goals more effectively, the architecture must be reconsidered before the public interface hardens.

#### 13.10.1 Why the Control Plane Should Not Depend on PHP

PHPX has a bootstrap constraint that normal PHP applications do not have. It must be able to discover a project, download and verify a PHP runtime, install that runtime, and diagnose a broken PHP installation on a machine where no usable PHP executable exists yet.

A core written in PHP creates a circular dependency. A PHAR provides convenient single file application packaging, but the machine still needs a compatible PHP interpreter to execute it. Shipping a private PHP interpreter beside the PHAR can solve the bootstrap problem, but it turns the bootstrap runtime into another platform specific artifact that the launcher must locate and manage.

PHP is still the right language in several parts of the system:

1. Composer remains the package operation authority.
2. WP CLI remains the WordPress command authority.
3. WordPress, Laravel, and Symfony continue to execute as PHP applications.
4. Small framework probes may execute inside the selected PHP runtime when application boot behavior is the only reliable source of truth.
5. Compatibility fixtures may be written in PHP to prove that PHPX preserves ecosystem behavior.

PHP could operate a long lived daemon through existing event loop and process libraries. The objection is not that PHP is incapable or always slow. The stronger objections are the bootstrap dependency, the need to bundle an interpreter, the resident runtime cost, and the weaker fit for a cross platform process and service supervisor that must remain available while PHP installations are being changed.

#### 13.10.2 Why Rust Fits This Product

1. Rust can produce a native executable that starts without PHP, Composer, Node.js, Bun, a Java Virtual Machine, or a .NET runtime.
2. Rust has no garbage collector. This removes a managed garbage collection runtime from the resident daemon and gives PHPX direct control over allocation and object lifetime. It does not guarantee low memory use, so the daemon still requires measurement and memory budgets.
3. Ownership and type checking prevent broad classes of memory lifetime, data race, and invalid state errors before release. This matters in a daemon that handles concurrent requests, process identities, filesystem mutations, downloaded archives, credentials, and shared service state.
4. Rust provides low level access to sockets, process groups, signals, filesystem APIs, native libraries, and platform services without requiring the entire application to use unsafe memory operations.
5. Async networking and structured concurrency libraries are suitable for the proxy, downloader, local protocol, health checks, and process event streams.
6. Cargo gives the project one integrated build, test, dependency, workspace, and release workflow.
7. The emerging PHP systems tooling represented by Mago, Yerd, and Libretto creates potential for shared Rust libraries, subprocess integration, or upstream collaboration.
8. Native startup and predictable idle behavior support the specific goal of keeping 50 sites registered while avoiding one heavy environment per site.

Rust is most valuable here around PHP, not instead of PHP. The control plane performs systems work. PHP continues to perform PHP ecosystem work.

#### 13.10.3 Rust Costs and Failure Modes

Rust also creates material costs:

1. Initial development can be slower for contributors who are new to ownership, lifetimes, traits, and async Rust.
2. Compile times and cross compilation can become release bottlenecks.
3. Async code, cancellation, process cleanup, and operating system abstraction remain difficult even with memory safety.
4. Rust has fewer likely contributors in the WordPress community than PHP or JavaScript.
5. Crate dependencies create their own supply chain risk and maintenance burden.
6. Unsafe code, native libraries, shell commands, and operating system APIs can reintroduce safety problems.
7. Rust does not solve PHP binary distribution, database distribution, certificates, DNS, or WordPress compatibility by itself.
8. A poorly designed Rust daemon can still leak memory, deadlock, consume excessive CPU, corrupt state through logic errors, or expose insecure local interfaces.
9. Choosing Rust for components that only express WordPress policy can make the project harder to contribute to without producing a meaningful performance or safety benefit.

The team must account for these costs in sequencing, contributor documentation, testing, and the public extension model.

#### 13.10.4 Why Not C or C++

C and C++ can meet the native startup, low level control, low memory, and no garbage collector requirements. They also have mature platform APIs and broad library access.

They are not the preferred default because the control plane is both privileged in effect and exposed to complex inputs. It downloads artifacts, verifies metadata, extracts archives, edits configuration, removes files, routes HTTP requests, and supervises processes. Memory corruption, use after free behavior, undefined behavior, and data races create unnecessary risk in that boundary. Modern C++ reduces some of this risk through disciplined ownership types, but it does not enforce the same safety baseline across the program.

C++ also brings more variation in build systems, dependency management, compiler behavior, and safe coding conventions. PHPX may still call established C or C++ libraries through narrow reviewed interfaces when reimplementation would be wasteful.

#### 13.10.5 Why Not Bun and TypeScript

Bun and TypeScript offer fast product iteration, familiar asynchronous programming, a large package ecosystem, and the ability to compile an application into a distributable executable. Bun documents that such an executable includes a copy of the Bun runtime.

That model can remove a separate install step, but it does not remove the runtime. The control plane would still carry JavaScriptCore garbage collection, dynamic language behavior, Bun release compatibility, and a larger runtime surface inside every binary. TypeScript checks disappear at runtime, so data crossing process, filesystem, and network boundaries still requires careful runtime validation.

Bun may be a good future choice for a graphical client, web interface, documentation tooling, or development scripts. It is less aligned with the smallest possible always available daemon, direct operating system control, and compile time enforcement of shared mutable state.

#### 13.10.6 Why Not Go

Go is the strongest alternative to Rust for PHPX. It has excellent networking and concurrency support, straightforward cross platform builds, fast compilation, simple deployment, and a much gentler learning curve. A capable version of this product could absolutely be built in Go.

The Rust preference comes from the strict idle resource goal, absence of a garbage collector, finer control over process and memory behavior, stronger compile time protection around shared state, and possible reuse within the growing Rust based PHP tooling ecosystem.

Those advantages are not free. Go may produce a working and maintainable public tool sooner, especially for a small team. Its garbage collector is engineered for low latency and provides tuning controls, so it cannot be rejected through a generic claim that garbage collection is bad. Rust should remain the choice only if the Milestone 0 prototype demonstrates an important resource or correctness advantage without creating unacceptable development and release complexity.

#### 13.10.7 Why Not Zig or Another Native Language

Zig has appealing properties for this product, including no garbage collector, strong C interoperability, native binaries, and cross compilation support. Its ecosystem for a complete cross platform daemon, proxy, package client, certificate manager, and process supervisor is less established than the Rust and Go options. It also places more memory management responsibility on the application.

Swift would fit a macOS only product well, but PHPX intends to support Linux and eventually Windows. Java, Kotlin, C Sharp, and similar managed platforms provide mature libraries and productive development, but add a runtime and garbage collector to a tool whose central promise includes low idle overhead and simple bootstrap distribution.

The comparison is therefore not Rust against every language in the abstract. It is which language best satisfies native bootstrap, low resident cost, safe concurrency, cross platform system control, contributor sustainability, and time to a reliable WordPress release.

#### 13.10.8 Why Developer Tools Are Moving Toward Rust

Not everything is moving to Rust. Go, C++, JavaScript, Python, PHP, Java, and C Sharp remain better choices for many products. Rust is unusually visible in command line and developer infrastructure rewrites because those tools benefit from a combination that used to require a difficult tradeoff:

1. Native startup speed and compact distribution.
2. Performance close to C and C++ for CPU and input or output intensive work.
3. Memory and thread safety checked by the compiler.
4. No garbage collector or required language runtime.
5. A modern dependency manager and build tool through Cargo.
6. Practical libraries for command interfaces, networking, parsing, serialization, cryptography, and asynchronous work.

There is also visibility bias. A Rust rewrite is easier to market than a careful improvement to an existing tool. Rewriting a mature PHP component solely because Rust is fashionable would add compatibility risk. PHPX has a stronger case because it needs a new native control plane that exists before and around the PHP runtime.

#### 13.10.9 Language Boundary Rules

1. Do not reimplement Composer in the Rust core.
2. Do not reimplement WP CLI commands in the Rust core.
3. Keep framework specific application execution inside the selected PHP runtime.
4. Prefer stable subprocess contracts before linking deeply into fast moving external codebases.
5. Keep operating system specific code behind narrow provider interfaces.
6. Forbid unsafe Rust in normal workspace crates. Any unavoidable unsafe module requires isolation, written invariants, focused tests, and review.
7. Pin dependencies, retain lockfiles, audit advisories and licenses, and minimize crates that execute build scripts.
8. Treat all project files, downloaded metadata, archives, daemon messages, and subprocess output as untrusted input.
9. Keep runtime installation and Composer execution usable without the daemon so a daemon failure cannot prevent recovery.
10. Add a language neutral local protocol only when a graphical client or external integration actually needs it.

#### 13.10.10 Milestone 0 Language Proof

Before the architecture is considered settled, Milestone 0 must build a narrow Rust proof containing project discovery, one verified download, one managed child process, a local daemon connection, and one proxied PHP request. It must record:

1. Cold command startup time.
2. Idle daemon resident memory.
3. Idle memory with 50 registered routes and no PHP workers.
4. Memory and latency with 15 addressable WordPress sites and a bounded number of active requests.
5. Proxy overhead relative to direct PHP FPM access.
6. Binary size and installation size.
7. Build time and cross target release complexity.
8. Behavior after a killed child process, stale process identifier, corrupt download, interrupted extraction, and daemon restart.
9. The amount of operating system specific and unsafe code required.
10. Contributor setup time on a clean machine.

A smaller Go comparison must exercise the resident daemon, route registry, and child process supervisor using the same fixtures. PHP should also be measured as the compatibility baseline for project discovery and command execution. Rust remains the preferred language when it meets the published resource budgets and the implementation complexity remains supportable. The decision must be recorded before the WordPress vertical slice expands.

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

1. `wordpress`, containing the documented required and recommended WordPress baseline.
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

A native Rust installation path may be introduced only for explicitly supported locked installations.

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

WordPress core, plugins, themes, must use plugins, and WP CLI packages can also execute arbitrary PHP. The default shared native topology is intended for trusted local client projects, not hostile code isolation. Separate database users and PHP FPM pools reduce accidental crossover but do not create a security sandbox. An optional dedicated or container backend should be recommended for untrusted projects.

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

### 17.4 WordPress Portfolio Resource Budget

The benchmark machine for the first published portfolio result must include an Apple Silicon Mac. Results must state the exact hardware, macOS version, PHP versions, WordPress fixtures, plugin set, database contents, and measurement method.

Engineering targets:

1. Fifty registered sites create no dedicated long running process per site.
2. Fifteen sites remain addressable concurrently through the shared proxy.
3. An idle site returns to zero PHP child workers after its configured idle timeout.
4. One PHP FPM master is shared by site pools using the same PHP version unless dedicated mode is selected.
5. The global PHP worker count remains bounded even when requests arrive for every site.
6. The default global worker budget is derived from available processors and memory, can be overridden, and is visible in status output.
7. The PHPX daemon and proxy together should target less than 100 MB idle resident memory before database and mail services.
8. The default shared database engine should target less than 512 MB idle resident memory under the public MVP fixture workload.
9. PHPX creates no duplicate synchronized copy of WordPress project files.
10. No Node process, asset watcher, queue worker, or cron loop starts for a site unless requested.
11. `phpx sites` should render 50 registry entries within 200 milliseconds without booting WordPress.
12. A registered site should answer its first request within two seconds when its runtime and shared services are already installed and healthy.
13. Published comparison must include DDEV with the same 50 registered and 15 available site scenario where practical.

The product must report actual resource consumption. It must not claim a fixed per site memory number because WordPress plugins, themes, PHP settings, and traffic change worker memory substantially.

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
11. Classic WordPress and Bedrock detection.
12. WordPress root and document root selection.
13. WordPress PHP compatibility lookup.
14. Standard WordPress rewrite routing.
15. DDEV configuration mapping.
16. Shared database allocation.
17. Site worker scheduling and global limits.
18. Portfolio registry queries.

### 19.2 Compatibility Tests

1. Compare constraint results against Composer Semver fixtures.
2. Compare direct Composer and PHPX wrapped Composer behavior.
3. Cover plugins, scripts, custom installers, repositories, and authentication stubs.
4. Verify platform requirements through actual selected runtimes.
5. Test supported PHP patch releases and API identifiers.
6. Compare WP CLI direct execution with PHPX wrapped WP CLI behavior.
7. Validate classic WordPress and Bedrock database configuration strategies.
8. Validate standard WordPress permalinks through the Rust proxy.
9. Detect unsupported custom `.htaccess` behavior.
10. Compare imported DDEV configuration with the generated PHPX plan.

### 19.3 End to End Fixtures

Maintain representative fixtures for:

1. Current classic WordPress single site.
2. Current Bedrock site.
3. Composer based WordPress site with plugins managed through Composer.
4. Legacy WordPress site requiring an end of life PHP branch.
5. WordPress site with a custom content directory.
6. WordPress site with a custom uploads directory.
7. WordPress site imported from common DDEV configuration.
8. WordPress site with unsupported DDEV custom services.
9. WordPress site with standard pretty permalinks.
10. WordPress site with custom `.htaccess` behavior.
11. WordPress multisite in subdirectory mode before that mode is released.
12. WordPress multisite in subdomain mode before that mode is released.
13. Portfolio fixture containing 50 registered sites and 15 routed sites.
14. Current Laravel application after the WordPress public MVP.
15. Current Symfony application after Laravel support.
16. Generic PHP site and Composer library after Symfony support.
17. Project with a Composer plugin.
18. Project with a dynamic extension requirement.
19. Project with a custom vendor directory.

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
13. Corrupt WordPress database import.
14. DDEV import containing unsupported custom Docker configuration.
15. Site removal requested while no current database snapshot exists.

### 19.5 Platform Matrix

Every supported tier requires automated testing on each supported architecture. A platform cannot be labeled supported solely because the Rust binary compiles.

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
12. Disk usage compared with DDEV and Mutagen using equivalent projects.
13. Total process count compared with DDEV using equivalent projects.

## 20. Platform Support Policy

### 20.1 Proposed Tiers

**Tier 1:** Guaranteed through continuous testing and release blocking failures.

**Tier 2:** Expected to work and tested regularly, but may not block every release.

**Tier 3:** Community supported and compiled when feasible.

### 20.2 Initial Targets

1. Technical proof Tier 1: macOS arm64.
2. WordPress public MVP Tier 1: macOS arm64.
3. Later Tier 1 candidate: Linux x86_64 with glibc.
4. Later Tier 2 candidate: Linux arm64 with glibc.
5. Later Tier 1 candidate: Windows x86_64.
6. Later Tier 2 candidate: macOS x86_64 while the platform remains viable.
7. Later Tier 2 candidate: Linux x86_64 with musl.

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
4. Cargo installation for contributors, not as the primary user requirement.

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

### Milestone 0: WordPress and DDEV Foundation Research

Deliverables:

1. Confirm the working product and repository name.
2. Measure the founding DDEV workload on an Apple Silicon Mac with a representative subset of 50 sites and 15 available sites.
3. Document DDEV project configuration, database, router, Mutagen, WP CLI, mail, import, export, and add on behavior relevant to WordPress migration.
4. Compare Yerd, PVM, StaticPHP, Herd, Valet, Mise, and existing PHP version managers.
5. Select the first PHP runtime artifact provider and verify PHP FPM support.
6. Select the first MariaDB or MySQL artifact provider and define shared engine lifecycle.
7. Define artifact, runtime, database, application adapter, and process provider interfaces.
8. Define the classic WordPress, Bedrock, and Composer based WordPress detection contract.
9. Define the supported WordPress rewrite subset and explicit `.htaccess` boundary.
10. Define WP CLI acquisition and execution policy.
11. Import or recreate a Composer Semver conformance corpus for Bedrock and Composer based sites.
12. Select licenses for the code and documentation.
13. Produce security threat models for artifacts, shared native services, WordPress executable code, and local TLS.

Exit criteria:

* [ ] No foundational component is being duplicated without an explicit reason.
* [ ] The first PHP runtime artifact can be verified and legally redistributed or fetched.
* [ ] The runtime includes usable PHP CLI and PHP FPM binaries.
* [ ] A native database engine distribution path is understood.
* [ ] The first DDEV comparison workload and measurement method are recorded.
* [ ] WordPress configuration injection and secret handling have an approved direction.

### Milestone 1: Native WordPress Vertical Slice

Deliverables:

1. Rust workspace and command line shell.
2. User scoped daemon.
3. Classic WordPress and Bedrock discovery.
4. Managed PHP installation for macOS arm64.
5. One managed PHP FPM master with demand driven site pools.
6. Rust HTTP and TLS proxy.
7. Explicit local DNS and certificate setup.
8. One shared MariaDB or MySQL engine with isolated project databases and users.
9. Managed WP CLI.
10. Managed Composer for Bedrock.
11. `phpx init`, `up`, `down`, `status`, `wp`, `composer`, `run`, `db import`, `db export`, and `doctor`.
12. Classic WordPress and Bedrock fixtures.
13. A 15 site routing and worker scheduling fixture.

Exit criteria:

* [ ] A clean macOS arm64 environment can serve the classic WordPress fixture through trusted HTTPS.
* [ ] The Bedrock fixture installs Composer packages and serves correctly.
* [ ] WP CLI operates on both fixtures through their selected runtime.
* [ ] Both fixtures use separate databases and users within one shared database engine.
* [ ] Standard WordPress permalinks work.
* [ ] Fifteen registered site routes can remain available while idle pools have zero PHP child workers.
* [ ] The system PHP installation remains unchanged.
* [ ] Interrupted installation recovery is proven.

### Milestone 2: WordPress Public MVP

Deliverables:

1. `phpx.toml` and `phpx.lock` version 1.
2. Portfolio registry supporting at least 50 sites.
3. Link, unlink, park, unpark, sleep, wake, open, logs, and site status workflows.
4. PHP runtime list, pin, update, and remove operations.
5. Supported PHP branches plus a documented legacy WordPress policy.
6. Versioned WordPress extension baseline.
7. PIE integration for selected dynamic extensions.
8. Database create, import, export, snapshot, restore, shell, and remove operations.
9. Shared mail capture with site filtering.
10. Classic WordPress and Bedrock local configuration strategies.
11. Common DDEV WordPress configuration import.
12. WordPress upload directory import and export.
13. Frozen, offline, noninteractive, and JSON modes.
14. Signed macOS release and installer.
15. Published PHPX and DDEV portfolio benchmark.

Exit criteria:

* [ ] Every WordPress public MVP success criterion passes on macOS arm64.
* [ ] Fifty sites remain registered without dedicated per site background processes.
* [ ] Fifteen sites remain addressable without 15 web and database container pairs.
* [ ] Database and upload data survive stop, sleep, restart, and PHPX upgrades.
* [ ] DDEV import refuses unsafe automatic cutover when unsupported custom behavior exists.
* [ ] Security, support, legacy PHP, and artifact provenance policies are published.

### Milestone 3: WordPress Portfolio Maturity

Deliverables:

1. WordPress multisite subdirectory support.
2. WordPress multisite subdomain support with wildcard local routing and TLS.
3. Plugin and theme project workflows using an explicit host WordPress site.
4. Richer DDEV import coverage and migration diagnostics.
5. An explicit Apache compatibility backend or a documented alternative for sites dependent on custom `.htaccess` behavior.
6. Optional Redis and object cache support.
7. Team environment validation and continuous integration integration.
8. Environment diff and explanation commands.
9. Resource history and portfolio diagnostics.
10. Additional macOS reliability and upgrade testing.

Exit criteria:

* [ ] Both supported multisite modes pass routing, WP CLI, database, upload, and domain fixtures.
* [ ] Plugin and theme repositories cannot accidentally operate against the wrong host site.
* [ ] Custom web server requirements produce a supported backend or a blocking explanation.
* [ ] Team locks are enforceable locally and in continuous integration.
* [ ] Portfolio performance remains within published budgets.

### Milestone 4: Laravel Adapter

Deliverables:

1. Laravel project discovery and document root selection.
2. Composer first synchronization.
3. Managed Artisan execution.
4. Laravel database presets for MariaDB, MySQL, PostgreSQL, and SQLite.
5. Optional Redis, queue, scheduler, mail, and asset process declarations.
6. Laravel specific diagnostics.
7. Current Laravel fixture and compatibility tests.

Exit criteria:

* [ ] A current Laravel application reaches a working local HTTPS environment from a clean supported machine.
* [ ] Composer plugins and scripts match direct Composer behavior.
* [ ] Laravel background processes start only when configured.
* [ ] WordPress portfolio benchmarks do not regress beyond the published tolerance.

### Milestone 5: Symfony and Generic PHP Adapters

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

### Milestone 6: Additional Platforms

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

### Milestone 7: Composer Installation Acceleration

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

**Given** a supported macOS arm64 machine without usable PHP, WP CLI, Composer, web server, or database

**When** the developer runs `phpx up` in a classic WordPress project

**Then** PHPX installs verified managed artifacts, allocates a database, registers trusted HTTPS, serves WordPress, and leaves the system runtime unchanged.

### Scenario 2: Multiple PHP Versions

**Given** WordPress Site A requires PHP 8.2 and WordPress Site B requires PHP 8.5

**When** the developer runs a PHP command in each project through PHPX

**Then** each command uses the correct runtime without global relinking.

### Scenario 3: Bedrock Composer Plugin

**Given** a Bedrock project depends on a Composer plugin

**When** the developer synchronizes the project

**Then** PHPX invokes Composer and preserves the normal plugin authorization and execution flow.

### Scenario 4: Missing Extension

**Given** WordPress or Composer requires a supported dynamic extension that is absent

**When** PHPX resolves the environment

**Then** PHPX identifies a PIE provider, requests any required explicit selection, installs it for the correct ABI, locks it, and reruns platform validation.

### Scenario 5: DDEV Import

**Given** a WordPress project contains common `.ddev/config.yaml` settings and an existing DDEV database

**When** the developer runs `phpx import ddev`

**Then** PHPX proposes mapped PHPX configuration, identifies unsupported customization, offers an explicit database migration, and leaves the DDEV environment intact.

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

### Scenario 10: WordPress Portfolio

**Given** 50 registered WordPress sites using three PHP versions and two compatible database engine versions

**When** 15 sites are requested during the same work period

**Then** each site routes to its matching PHP FPM pool, compatible sites share engine processes, the global worker cap is respected, and idle pools return to zero PHP child workers.

### Scenario 11: Destructive Database Removal

**Given** a WordPress project database contains local data and no current snapshot

**When** the developer requests database removal

**Then** PHPX displays the site, engine, database, user, and backup state and refuses deletion without explicit confirmation.

### Scenario 12: Shared Database Engine

**Given** 12 WordPress sites request MariaDB 10.11 in shared mode

**When** all sites are registered

**Then** PHPX runs one MariaDB 10.11 engine and creates 12 separate databases and users.

### Scenario 13: Idle Site Wake

**Given** a registered WordPress site has no PHP child worker

**When** its local URL receives a request

**Then** the selected PHP FPM master creates a worker on demand and the proxy completes the request without a manual site start command.

### Scenario 14: Custom Apache Behavior

**Given** a WordPress project contains `.htaccess` directives outside the supported WordPress rewrite subset

**When** PHPX plans the local environment

**Then** PHPX reports the unsupported directives and refuses to claim full compatibility without an approved server backend.

### Scenario 15: WordPress URL Migration

**Given** a database contains a production or `.ddev.site` URL

**When** the database is imported

**Then** PHPX preserves the imported values until the developer explicitly approves a WP CLI aware search and replace operation.

### Scenario 16: Site Sleep

**Given** a WordPress site has an asset watcher and PHP workers running

**When** the developer runs `phpx site sleep`

**Then** PHPX stops optional project processes, allows PHP workers to retire, preserves the route, database, uploads, configuration, and future wake behavior.

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

**Mitigation:** Enforce WordPress milestone gates. The first vertical slice includes only the local capabilities required to prove the lean WordPress topology. Defer Laravel, Symfony, generic PHP, broad services, graphical clients, and native Composer installation until their named milestones.

### 25.5 Existing Competitors

**Risk:** DDEV, Yerd, Herd, PVM, Mise, or another project may already solve important portions of the product.

**Mitigation:** Treat integration and contribution as preferred options when they preserve the product promise. Compete on the missing project and automation contract, not on duplicating every local environment feature.

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

**Risk:** One shared database or PHP FPM master failure can affect several sites at once.

**Mitigation:** Use health checks, bounded restart policy, versioned engine instances, durable project data, logical snapshots, clear dependency reporting, and optional dedicated mode for sensitive projects.

### 25.12 Shared Native Trust Boundary

**Risk:** A compromised or intentionally hostile WordPress plugin may access resources available to the local user and may attack shared local services.

**Mitigation:** State clearly that native mode is for trusted local projects, use separate database users, restrict services to loopback, minimize credentials exposed to each process, and offer a future dedicated or container backend for untrusted code.

### 25.13 WordPress Configuration Diversity

**Risk:** Classic WordPress projects define database and URL constants through arbitrary PHP in `wp-config.php`, while Bedrock and custom stacks use different environment conventions.

**Mitigation:** Detect known strategies, propose reviewable local integration, never rewrite configuration silently, keep generated credentials out of version control, and block startup when PHPX cannot prove which database the site will use.

### 25.14 Apache Compatibility

**Risk:** WordPress plugins and legacy sites may depend on custom `.htaccess` behavior that a Rust front controller adapter cannot reproduce.

**Mitigation:** Support the standard WordPress rewrite subset, inspect custom directives, publish the boundary, and add an explicit Apache compatibility backend instead of pretending all directives work.

### 25.15 Legacy Client Sites

**Risk:** A WordPress portfolio may contain sites requiring PHP branches or database versions that are no longer supported upstream.

**Mitigation:** Separate compatibility from security support, require explicit legacy selection, preserve visible warnings, isolate legacy engines by version, and provide upgrade diagnostics without forcing application changes.

### 25.16 Native Database Distribution

**Risk:** Portable MariaDB and MySQL acquisition, upgrades, data directories, and macOS architecture support create another artifact supply chain.

**Mitigation:** Begin with one reviewed MariaDB version, keep the database provider replaceable, prefer logical backups across incompatible versions, verify artifacts, and delay broad engine matrices until real projects require them.

### 25.17 Performance Claim Credibility

**Risk:** A leaner than DDEV claim may become vague or misleading when plugin sets and traffic differ.

**Mitigation:** Publish exact fixtures, commands, hardware, idle and active states, process counts, memory, disk, and response measurements. Keep claims limited to reproducible benchmark results.

### 25.18 Rust Delivery Complexity

**Risk:** Rust learning cost, async complexity, platform integration, and release engineering may delay the WordPress product enough to erase the benefit of the language choice.

**Mitigation:** Prove the smallest control plane in Milestone 0, keep PHP ecosystem behavior in PHP, compare the resident core with Go, isolate operating system providers, measure contributor setup, and reconsider the language before public interfaces harden if the evidence does not support Rust.

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

**Recommended direction:** Evaluate reuse or integration with Yerd before implementing a new proxy, DNS service, and runtime manager.

**Still required:** Compare daemon boundaries, licenses, project configuration, platform support, and contribution feasibility.

### 26.6 Code Quality Integration

**Recommended direction:** Integrate or execute Mago instead of creating another parser, formatter, linter, and analyzer.

**Still required:** Determine whether stable library interfaces exist or a managed executable is safer.

### 26.7 Composer Acceleration

**Recommended direction:** Defer native installation and evaluate Libretto when the compatibility contract is established.

**Still required:** Audit correctness, project maturity, licensing, and shared component opportunities.

### 26.8 Service Backend

**Recommended direction:** Begin with one shared native MariaDB 10.11 provider for WordPress, then add versioned MariaDB or MySQL providers based on real portfolio requirements. Keep an optional container adapter for exceptional projects.

**Still required:** Decide service scope, licensing, update sources, and data migration behavior.

### 26.9 Windows Timing

**Recommended direction:** Design portable abstractions from the beginning, but do not block the macOS technical proof on Windows support.

**Still required:** Recruit real Windows testing before assigning Tier 1 status.

### 26.10 Telemetry

**Recommended direction:** No telemetry by default.

**Still required:** Decide whether explicit opt in aggregate usage data would ever provide enough value to justify the trust cost.

### 26.11 PHP FPM Sharing

**Recommended direction:** One PHP FPM master per PHP runtime version, one pool per site, `ondemand` workers, and one global worker ceiling.

**Still required:** Benchmark OPcache behavior, configuration reload cost, failure radius, process limits, and dedicated mode.

### 26.12 Classic WordPress Configuration

**Recommended direction:** Detect existing environment aware configuration first. Otherwise propose a local `wp-config-phpx.php` include through an explicit reviewable change.

**Still required:** Test common agency conventions, local ignored config files, Bedrock environment files, DDEV generated includes, and hardcoded production constants.

### 26.13 Default Local Domain

**Recommended direction:** Use `.test` by default and support explicit aliases.

**Still required:** Decide wildcard resolver implementation, multisite domain mapping, certificate scope, and conflict behavior with Valet, Herd, DDEV, and Yerd.

### 26.14 DDEV Migration Depth

**Recommended direction:** Import common WordPress configuration and data first. Inventory unsupported custom behavior without translating arbitrary Docker Compose.

**Still required:** Define the exact supported `.ddev/config.yaml` fields and determine whether PHPX may invoke DDEV export commands during an approved migration.

### 26.15 WordPress Default PHP Policy

**Recommended direction:** Select a currently supported PHP branch that official WordPress compatibility data accepts, while preserving explicit project pins and warnings for legacy sites.

**Still required:** Decide whether the default favors the newest compatible branch or a more conservative compatibility branch for third party plugins and themes.

### 26.16 Untrusted Project Isolation

**Recommended direction:** Document native mode as trusted local development and design a future dedicated or container backend behind the same project interface.

**Still required:** Define the threat model, dedicated database behavior, filesystem boundaries, and user experience for switching backends.

### 26.17 Core Implementation Language

**Recommended current decision:** Rust for the native control plane, PHP for Composer, WordPress, WP CLI, framework execution, and compatibility probes.

**Still required:** Complete the Milestone 0 Rust proof, compare the resident core with a narrow Go implementation, publish the measurements, and record the final architecture decision before Milestone 1 expands.

## 27. Required Research Before Implementation

1. Inventory a representative anonymized subset of the founding 50 site WordPress portfolio.
2. Record PHP versions, WordPress versions, database engines, document roots, Composer usage, DDEV customizations, upload paths, and required services.
3. Benchmark DDEV with representative 1 site, 10 site, and 15 site available workloads on the Apple Silicon reference machine.
4. Review DDEV architecture, WordPress quickstart, database workflows, performance modes, custom web server configuration, and add on behavior.
5. Review official WordPress runtime, database, HTTPS, extension, rewrite, and multisite requirements.
6. Review WP CLI distribution, update, package, path, multisite, database, and search and replace behavior.
7. Prototype PHP FPM `ondemand` pools across 50 configured sites and multiple PHP versions.
8. Measure one master per PHP version against one master per site.
9. Prototype a global PHP worker ceiling and first request wake behavior.
10. Audit native MariaDB and MySQL artifacts, licenses, configuration, logical backup, and upgrade paths on macOS arm64.
11. Define classic WordPress and Bedrock local configuration strategies.
12. Define the supported WordPress rewrite subset and `.htaccess` detection method.
13. Review Composer source and documented plugin contracts.
14. Build a Composer Semver conformance suite for PHP runtime requests.
15. Inspect Composer lock platform metadata across representative Bedrock and WordPress projects.
16. Audit StaticPHP artifacts, manifests, build recipes, and licenses.
17. Audit PIE target selection and noninteractive workflows.
18. Review Yerd architecture and identify reusable components or collaboration paths.
19. Review Mago library boundaries and executable integration.
20. Review Libretto compatibility tests and cache architecture.
21. Compare existing `.php-version` behavior across version managers.
22. Define runtime artifact and shared native service threat models.
23. Measure common PHP runtime and extension combinations across the WordPress portfolio and public Composer projects.
24. Validate classic WordPress and Bedrock fixtures on a clean macOS arm64 environment.
25. Build and measure the Milestone 0 Rust control plane proof.
26. Build a narrow Go comparison for daemon memory, route registration, child process supervision, and release complexity.
27. Measure a PHP based discovery and command runner as the ecosystem compatibility baseline.
28. Record the core language architecture decision with benchmark evidence before Milestone 1 expands.

## 28. Reference Sources

Composer documentation

https://getcomposer.org/doc

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

https://laravel.com/docs/valet

Laravel Herd

https://herd.laravel.com/docs

uv

https://docs.astral.sh/uv

Rust language overview

https://rust-lang.org

Rust ownership

https://doc.rust-lang.org/book/ch04-01-what-is-ownership.html

Rust concurrency

https://doc.rust-lang.org/book/ch16-00-concurrency.html

Go garbage collector guide

https://go.dev/doc/gc-guide

Bun standalone executables

https://bun.sh/docs/bundler/executables

PHP PHAR documentation

https://www.php.net/manual/en/book.phar.php

DDEV architecture

https://ddev.readthedocs.io/en/stable/users/usage/architecture/

DDEV performance on macOS

https://ddev.readthedocs.io/en/stable/users/install/performance/

DDEV WordPress quickstart

https://ddev.readthedocs.io/en/stable/users/quickstart/

DDEV configuration

https://ddev.readthedocs.io/en/stable/users/configuration/config/

WordPress requirements

https://wordpress.org/about/requirements/

WordPress PHP compatibility

https://make.wordpress.org/hosting/handbook/server-environment/

WP CLI command reference

https://developer.wordpress.org/cli/commands/

PHP FPM configuration

https://www.php.net/manual/en/install.fpm.configuration.php

## 29. Refinement Readiness

This draft is ready for a dedicated refinement pass focused on product positioning, runtime artifact strategy, Composer compatibility, public MVP boundaries, configuration format, platform priorities, and open source governance.

The specification should not be converted into implementation phases until the Milestone 0 research decisions have been reviewed, because runtime provenance and existing project integration can materially change the architecture.
