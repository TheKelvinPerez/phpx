# ADR 0001: Use Cobra As A Thin Command Adapter

## Status

Accepted

## Context

Elefante needs nested commands, global flags, help, shell completion, argument validation, context cancellation, and command tests. The core project model, resolver, providers, Composer integration, and executor must remain usable without a command framework.

## Decision

Elefante will use Cobra for command routing and presentation concerns only.

The root command will be created by `NewRootCommand(dependencies)`. Command construction will not use global mutable state. Command handlers will translate arguments into application requests, invoke application services, and render typed results.

Core packages will not import Cobra. Internal packages will not call `os.Exit`, `log.Fatal`, or global output streams. The process entry point alone maps the final result to a process exit code.

Elefante will not use the Cobra code generator. Command construction will remain explicit and reviewable.

## Consequences

Cobra becomes a reviewed runtime dependency. Command tests can supply arguments, context, input, output, and error streams without launching another process. Core behavior remains independent from command parsing and help generation.

## Alternatives Considered

1. Use the standard `flag` package, rejected because Elefante would need to build and maintain nested commands, persistent flags, help generation, and shell completion.
2. Put application behavior directly in Cobra handlers, rejected because command tests would lock behavior to one presentation framework and provider logic would become difficult to reuse.
3. Generate command files with Cobra tooling, rejected because explicit construction is easier to review and test.
