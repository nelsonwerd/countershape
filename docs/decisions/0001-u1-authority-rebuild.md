# ADR 0001 — U1 authority must be unavailable before it is provable

- **Status:** accepted during U1 red-team rebuild
- **Date:** 2026-07-14
- **Scope:** truth-kernel construction APIs and prompt-pack unit boundaries

## Context

The first U1 draft compiled and passed its preliminary unit/race/fuzz/mutation runs, but two independent static audits constructed false evidence entirely through public APIs. Examples included a repaired-invalid-UTF-8 `WorldPlan` collision, a control fact promoted to eligible behavior, a roster-omitting map, arbitrary `(stage,digest)` lineage advancement, caller-declared sweep completeness, and an `ALLOW_OBSERVED` tuple not present in confirmation.

These were not test-count problems. Each API accepted a raw record, Boolean, enum, list, or digest where the stronger claim required an authority produced by an earlier semantic operation.

## Decision

Countershape follows one rule across the build:

> If the unit that can establish an authority does not exist yet, the constructor that consumes that authority does not exist yet either.

Consequences for U1:

1. Identity-bearing typed values are validated before any ordinary JSON encoder can repair them.
2. Candidate maps cover an exact expected roster; a divergent baseline is a sealed refinement of a general complete candidate map.
3. Evidence-bearing outcome identity and preservation-map identity are distinct types.
4. Captured behavior and detected controls are disjoint values, not compatible flags.
5. Speculative future artifact-stage constructors are removed. Later units introduce transitions only when their semantic predecessor and guard exist.
6. Logical reduction evidence exists in U1, but durable `ONE_MINIMAL_UNDER` authority does not. U5 now owns the minimal durable sweep store required to construct it.
7. A ruling universe is derived from the exact observed/confirmed eligible-entry universe. Callers cannot author the disallowed complement.
8. Envelope policy, measured instance facts, and the result of applying the policy are separate objects. A caller-selected `ADMITTED` enum is not evidence.

## Prompt-pack correction

U5, not U6, introduces the narrow content-addressed durable-sweep authority. U6 extends that storage primitive for full semantic objects and study-head compare-and-swap. This preserves dependency direction: `internal/reduce` validates an opaque store-issued authority; it never imports the store implementation.

## Nonclaims

Opaque types do not prove that host measurements or subprocess facts are truthful. U1 establishes strict structure and pure relationships only. Physical freshness, measured-world provenance, durable publication, and current-head authority remain unavailable until their owning units land and are receipted.
