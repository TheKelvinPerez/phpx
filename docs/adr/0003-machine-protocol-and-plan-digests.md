# ADR 0003: Use One Event Protocol And Content Addressed Plan Approval

## Status

Accepted

## Context

Elefante is expected to serve automation clients more often than direct human operation. Read only commands can return a final result immediately, while synchronization and child commands must stream progress and output. Machine driven mutations also need protection against the reviewed plan changing before execution.

## Decision

The `--json` flag will produce one versioned newline delimited JSON event protocol for every primary command.

Each line will be independently valid JSON. Stable event types will represent lifecycle, results, progress, child streams, warnings, errors, and completion. Standard output will contain protocol events only while JSON mode is active.

Canonical protocol events will omit random identifiers and timestamps so equivalent inputs can produce deterministic output.

Every executable plan will receive a SHA256 digest calculated from a canonical representation of the requested operation, project inputs, provider observations, selected artifacts, trust requirements, and ordered actions.

Noninteractive callers may execute the reviewed plan by supplying `--approve-plan <digest>`. Elefante will recompute the plan immediately before mutation and fail without mutation when the digest differs.

## Consequences

Automation clients need one streaming parser for every command. Human mode remains separate and optimized for terminal use. Plan execution gains an explicit review and apply boundary. Protocol schemas and digest inputs become compatibility contracts that require versioning and golden tests.

## Alternatives Considered

1. Return one JSON document from every command, rejected because long running commands would need to buffer child output and progress.
2. Use different unrelated JSON formats for read only and streaming commands, rejected because automation clients would need separate lifecycle models.
3. Allow `--yes` as the only noninteractive approval mechanism, rejected because it cannot prove that the applied plan matches a previously reviewed plan.
