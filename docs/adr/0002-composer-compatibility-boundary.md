# ADR 0002: Preserve Composer Authority While Evaluating Platform Constraints In Go

## Status

Accepted

## Context

Elefante must diagnose a project when PHP or Composer is missing or incompatible. Delegating all requirement interpretation to Composer would prevent diagnosis in exactly that condition. Reimplementing Composer dependency solving would create a large compatibility and maintenance risk.

## Decision

Elefante will parse `composer.json` and `composer.lock` as data without loading project code.

The Go resolver will implement Composer compatible parsing and evaluation only for platform requirements, including PHP, PHP subtypes, extensions, system libraries, Composer, the Composer plugin API, and the Composer runtime API.

Elefante will not implement Composer package dependency solving. Official Composer will perform dependency installation, plugin execution, script execution, lock handling, and final platform verification.

The platform constraint evaluator will be validated against a conformance corpus derived from official Composer behavior. Unknown or unsupported constraint syntax will produce a blocker instead of an approximation.

## Consequences

`doctor` and `plan` can work without a usable PHP runtime. Elefante assumes responsibility for a narrow but correctness sensitive compatibility layer. Constraint behavior requires focused tests, fuzzing, and comparison against official Composer semantics.

## Alternatives Considered

1. Invoke Composer for every constraint decision, rejected because Composer itself requires a usable PHP runtime.
2. Implement the complete Composer solver in Go, rejected because Composer remains the dependency authority.
3. Use a generic semantic version library without conformance tests, rejected because Composer constraint grammar and stability behavior are not interchangeable with a generic semantic version interpretation.
