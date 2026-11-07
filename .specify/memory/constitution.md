<!--
Sync Impact Report
Version change: uninitialized scaffold → 1.0.0
Modified principles: none (initial adoption)
Added sections: Core Principles, Project Constraints, Development Workflow
Removed sections: none
Follow-up TODOs: Confirm the original ratification date.
-->
# Chess Clone Constitution

## Core Principles

### I. Test-Driven Development (NON-NEGOTIABLE)
Every behavior change MUST begin with a focused automated test that fails for the
intended reason. Implement the minimum change needed to pass, verify the full
relevant test suite, and refactor only while tests remain green (Red–Green–Refactor).
This sequence provides evidence that tests detect regressions and keeps implementation
aligned with expected behavior. Exceptions require explicit approval before work begins.

### II. Chess Rules Correctness
Game behavior MUST follow the applicable chess rules, including legal move validation,
turn order, check, checkmate, and draw conditions. Rule-related changes MUST include
tests for the changed behavior and relevant boundary cases. Correctness is foundational
to a trustworthy chess experience.

### III. Integration Confidence
Changes crossing component, state, or interface boundaries MUST be covered by
integration tests in addition to focused unit tests where practical. Tests MUST verify
observable behavior rather than internal implementation details. This catches failures
that isolated tests cannot reveal.

### IV. Simplicity and Maintainability
Implementations MUST favor the simplest design that satisfies tested requirements.
Avoid speculative features and unnecessary abstractions; explain unavoidable complexity
in the change review. Readable, focused code makes future rule and interface changes safer.

### V. Consistent Player Experience
User-visible behavior MUST communicate game state and available actions clearly and
consistently. Changes to interaction or presentation MUST preserve a usable experience
across the supported project interfaces and be verified with appropriate tests.

## Project Constraints

The project MUST preserve a clear separation between chess rules, game state, and
presentation wherever the existing architecture supports it. New dependencies and
architectural layers MUST have a documented need; avoid introducing them for hypothetical
future use.

## Development Workflow

For each behavior change, developers MUST record the failing test result before
implementation, the passing result afterward, and any relevant regression-test results.
Before review, run the focused tests and the broader available test suite. Reviews MUST
check rule correctness, test quality, and consistency with these principles.

## Governance

This constitution governs project development practices. Amendments MUST be reviewed
and documented with their rationale. Changes to principles or other governance
requirements MUST increment the MAJOR version when they remove or materially redefine
existing obligations, MINOR when adding principles or materially expanding guidance,
and PATCH for clarifications or non-semantic edits. Every amendment MUST update the
version and last-amended date. Contributors and reviewers MUST check proposed work for
compliance; deviations require explicit justification and approval before implementation.

**Version**: 1.0.0 | **Ratified**: TODO(RATIFICATION_DATE): confirm original adoption date | **Last Amended**: 2026-11-07
