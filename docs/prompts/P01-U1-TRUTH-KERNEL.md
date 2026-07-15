# P01 / U1 — Build the truth kernel before any subprocess exists

## Mission

In a fresh chat, implement and seal Countershape’s pure Go authority: strict canonical identity, deterministic plans, eligibility, exact outcome maps, immutable state transitions, and ruling validation. This unit must make the dangerous states difficult or impossible to construct. It must contain no Git invocation, process spawn, socket, HTTP client, UI, emitter, or execution cache.

Begin only if U0’s strict didrun gate is green. Work autonomously, preserve user changes, narrate phases briefly, and use tracked status files as memory.

## Required read set

Read completely: `/Users/drewnelson/.claude/CLAUDE.md`, `docs/CONCEPT_BRIEF.md`, `docs/ARCHITECTURE.md`, `docs/THREAT_MODEL.md`, `docs/CLAIM_VOCABULARY.md`, `docs/STATE_MACHINES.md`, `research/deep-dive/07-SYNTHESIS.md`, `research/deep-dive/08-RED_TEAM.md`, `docs/status/U0.md`, `docs/HANDOFF_MODE_C.md`, and this prompt. Read every `spec/schema/v1`, `spec/examples/v1`, and `spec/vectors/v1` artifact. Confirm branch `codex/countershape-autopilot`, clean/understood status, U0 commit, and `NO_COLOR=1 didrun verify --strict` exit 0 before editing.

## Locked boundaries

- Go is the sole identity/classification authority. Do not use `encoding/json` on identity-bearing input before a strict token scanner has detected duplicate names, invalid UTF-8, lone surrogates, unsafe integers, `-0`, non-finite or unsupported numeric spellings.
- Never Unicode-normalize. Canonical object keys use the ordering U0 specifies; durations are integer milliseconds; exact decimals, if present, are tagged canonical strings. No float enters an identity object.
- Every semantic artifact has `schema_version`, `kind`, canonical bytes, and domain-separated SHA-256 over `countershape/v1/<kind>\x00<bytes>`.
- `WorldPlan` is deterministic and fully defaulted. It cannot contain allocated ports, temp roots, PIDs, clocks, random values, ambient environment interpolation, shell strings, or absolute output paths.
- `WorldInstance` and `ComparisonEnvelope` are types only in U1. Envelope admission is a declared policy over measured/tolerated/rejected/uncontrolled dimensions, never a proof of equivalent worlds.
- A control result cannot project or enter an `OutcomeMap`. HTTP/CLI success-shaped details do not exist in the generic kernel.
- `OutcomeMap` is one canonical sorted map from `candidate_execution_key` to projection fingerprint, with exclusions separate. Candidate order, display label, producer, support count, and group ordinals cannot affect its digest.
- Reduction outcomes are `PRESERVES`, `CHANGES`, or `UNRESOLVED`. `ONE_MINIMAL_UNDER` must be constructible only from a durable complete sweep containing the exact enumerated neighbor set and fresh `CHANGES` evidence for every neighbor. Cancellation, staleness, any unresolved neighbor, or exhausted budget blocks it.
- Rulings have no defaults. Allow-many is a canonical set of complete selected-field tuples. Missing differs from present-empty. `REJECT_ALL`, `DEFER`, and `REFINE` are noncompilable at the type boundary. U1 validates this but emits no source.
- didrun receipt references are opaque strings; unknown grades round-trip verbatim, absent evidence is `UNRECEIPTED`, and no product classification derives from them.

## Exact ownership

U1 owns:

```text
go.mod
internal/canon/**
internal/domain/**
internal/spec/**
internal/observe/**
internal/compare/**
internal/reduce/model.go
internal/choice/validation.go
testkit/vectors/**
tools/mutate-u1.mjs
docs/status/U1.md
docs/HANDOFF_MODE_C.md
```

Do not create `cmd/`, `internal/gitobj`, `internal/world`, either adapter, store, server, report, web, or Node contract harness. `internal/reduce/model.go` contains pure sums/invariants only; orchestration belongs to U5.

## Implementation sequence

1. Initialize the smallest local Go module consistent with U0. Keep dependencies at zero unless a dependency is justified in writing and vendoring/licensing implications are recorded; standard library is preferred.
2. Build a strict JSON/token reader and canonical encoder. Parse first into project-owned lossless tokens, validate, then construct typed values. Return typed errors with stable machine codes and byte offsets, never partial identity.
3. Implement domain-separated digests and golden vectors. Include nested ordering, escaped keys, missing/present-empty, boundary integers, invalid UTF-8, duplicate names, `-0`, exponent/decimal rejections, lone surrogates, and kind-domain separation.
4. Implement deterministic `WorldPlan` compilation from inert spec: materialize every default and budget, preserve ordered argv/multimaps, sort only fields declared unordered, reject secret values in stable bytes, shell strings, unknown keys, impossible budgets, and ambient interpolation.
5. Implement execution-attempt and artifact-lineage sum types plus legal transition constructors. Staleness creates a new status object; it never mutates old bytes. Illegal paths—ruling before confirmation, reuse under changed plan, emission request from a noncompilable action—must be rejected before I/O.
6. Implement centralized eligibility and `StableBatch` classification over abstract trial/projection facts. Exact counts and histograms are preserved. Finite agreement is named only `OBSERVED_STABLE(k/k,h)`.
7. Implement `CandidateExecutionKey`, `ProjectionFingerprint`, and `OutcomeMap`. Require at least two eligible candidates and fingerprints for divergence. Derive display clusters only as identity-free views.
8. Implement ruling validation and the selected-field separation obligation over a closed typed field identifier supplied by a domain. Empty fields, unknown fields, allowed/disallowed collisions, or cross-product allow-many return typed refusal. Keep compile eligibility as a sealed sum usable by U6, not a boolean.
9. Add property, fuzz, model-state, serialization round-trip, and permutation tests. Update U1 status/handoff with exact nonclaims.

## Acceptance, negative cases, and required mutants

Tests must establish deterministic bytes/digests across repeated runs; parse/encode idempotence; order independence only where declared; candidate permutation invariance; label/producer/support-count irrelevance; explicit exclusion identity; all eligibility classifications; illegal lineage transitions; receipt string round-trip; and separation for allow-one, tuple-set allow-many, custom expectation, empty, missing, and present-empty.

Create a dependency-free mutation driver that copies the repository to a temporary directory, verifies each exact anchor exists, applies one mutation, runs the targeted tests, and requires failure. Required killed mutants: remove the digest kind separator; bypass duplicate-key rejection; accept negative zero; drop a candidate key from map identity; replace labeled-map equality with partition shape; turn a control failure into an eligible observation; collapse `UNRESOLVED` into `CHANGES`; permit incomplete sweep to create one-minimal; allow empty field selection; and compile allow-many as independent value sets. A missing mutation anchor is a failed mutation run.

All load-bearing commands go through didrun. At minimum run:

```text
didrun run -- go test ./...
didrun run -- go test -race ./...
didrun run -- go test ./internal/canon ./internal/domain ./internal/observe ./internal/compare ./internal/choice -fuzz=Fuzz -fuzztime=20s
didrun run -- node tools/mutate-u1.mjs
didrun run -- git diff --check
```

If the Go version requires one fuzz target per invocation, run each separately through didrun and claim each relevant event. Run a deterministic repeat check against fresh temporary caches if the suite supplies one. Do not claim Linux runtime from cross-compilation.

Use `didrun show --session`, then attach explicit claims with commands of the form `didrun claim tests-pass --label "U1 truth kernel" --event N --path .` for the exact normal/race/fuzz/mutation events and a scoped lint/command claim for the diff check. Stage only U1 files, inspect the staged diff, and commit `feat: establish the Countershape truth kernel`. No AI co-author trailer. Seal HEAD and loop on `NO_COLOR=1 didrun verify --strict` until exit 0. On failure, fix the real issue, rerun via didrun, make new claims, create a new fix commit if the failed commit was sealed, reseal, and reverify. Never weaken tests, remove/relabel claims, or erase failed receipt history.

## Stop/fallback and handoff

Stop before subprocess work if any canonical, map-identity, state-machine, eligibility, separation, or required mutation gate fails. The honest fallback is a still-open truth-kernel prototype, not U1 complete. If strict parsing cannot support a proposed portable value, cut that value from v1 and revise the controlling spec explicitly; do not silently use ordinary JSON semantics.

Finish with updated `docs/status/U1.md` and Mode C handoff, then report branch, commit(s), exact didrun grades, strict exit, tests/mutants run, environment claims, remaining UNRECEIPTED items, and `P02-U2-GIT-WORLD-SUBSTRATE.md` as the next prompt.
