# P01 / U1 — Build the truth kernel before any subprocess exists

## Mission

In a fresh chat, implement and seal Countershape’s pure Go authority: strict canonical identity, deterministic plans, eligibility, exact outcome maps, immutable state transitions, and ruling validation. This unit must make the dangerous states difficult or impossible to construct. It must contain no Git invocation, process spawn, socket, HTTP client, UI, emitter, or execution cache.

Begin only if U0’s strict didrun gate is green. Work autonomously, preserve user changes, narrate phases briefly, and use tracked status files as memory.

## Required read set

Read completely: `/Users/drewnelson/.claude/CLAUDE.md`, `docs/CONCEPT_BRIEF.md`, `docs/ARCHITECTURE.md`, `docs/SEMANTICS.md`, `docs/PROJECTION_ALGEBRA.md`, `docs/THREAT_MODEL.md`, `docs/CLAIM_VOCABULARY.md`, `docs/STATE_MACHINES.md`, `research/deep-dive/07-SYNTHESIS.md`, `research/deep-dive/08-RED_TEAM.md`, `docs/status/U0.md`, `docs/HANDOFF_MODE_C.md`, and this prompt. Read every `spec/schema/v1`, `spec/examples/v1`, and `spec/vectors/v1` artifact. Confirm branch `codex/countershape-autopilot`, clean/understood status, U0 commit, and `NO_COLOR=1 didrun verify --strict` exit 0 before editing.

## Locked boundaries

- Go is the sole identity/classification authority. Do not use `encoding/json` on identity-bearing input before a strict token scanner has detected duplicate names, invalid UTF-8, lone surrogates, unsafe integers, `-0`, non-finite or unsupported numeric spellings.
- Never Unicode-normalize. Canonical object keys use the ordering U0 specifies; durations are integer milliseconds; exact decimals, if present, are tagged canonical strings. No float enters an identity object.
- Every semantic artifact has a canonical body containing `schema_version`, `kind`, and all identity-bearing semantic fields. Domain-separated SHA-256 addresses that body. U0 schemas describe target storage/wire envelopes, but U1 must not claim runtime/schema parity: explicit codecs and schema round trips belong to U6.
- `WorldPlan` is deterministic and fully defaulted. It cannot contain allocated ports, temp roots, PIDs, clocks, random values, ambient environment interpolation, wrappers, shell strings, or host paths. Argv zero is a bare declared logical tool; the remaining arguments use the closed v1 profile. Tool resolution is measured later.
- `CandidateExecutionKey` is only a wire/display reference. An opaque `CandidateExecutionBinding` retains the full tree/materialization/plan/adapter/runner/projection identity, and allocating a `WorldInstance` requires both that binding and the actual matching `WorldPlan`.
- `WorldInstance` is structural declared-consistency identity for plan, binding, stimulus, attempt artifact, purpose, nonce, and ordinal. `InstanceMeasurements` is the complete policy-bound measured row. `AssessComparison` alone yields persisted `ComparisonAdmission` or `RejectedComparison`; only admission yields deterministic internal per-row tokens. Rejection is a pre-batch refusal and cannot become a control trial, batch, or outcome map.
- A control result cannot project or enter an `OutcomeMap`. HTTP/CLI success-shaped details do not exist in the generic kernel.
- `CandidateOutcomeMap` covers the exact expected candidate roster with one canonical sorted eligible map plus explicit exclusions and binds plan, the exact shared ordered per-repetition admission-digest set, stimulus-independent comparison basis, phase, and globally unique attempt/world evidence. Candidate order, display label, producer, support count, and group ordinals cannot affect its typed preservation-map digest. Preservation first requires full-map comparability on plan, envelope, comparison basis, roster, eligible set, and exclusions; only then may exact labeled-map digest equality yield `PRESERVES`. A naked digest never suffices. `DivergentBaseline` is a sealed refinement requiring two eligible candidates and two fingerprints.
- Reduction outcomes are `PRESERVES`, `CHANGES`, or `UNRESOLVED`. U1 may model logical evaluation/sweep evidence but must expose no public constructor for `ONE_MINIMAL_UNDER`: durable store authority does not exist until U5. Cancellation, staleness, any unresolved neighbor, or exhausted budget must remain structurally incapable of acquiring it later.
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
internal/reduce/**
internal/choice/**
tools/validate-planning.mjs
tools/mutate-u1.mjs
tools/test-mutate-u1.mjs
tools/check-u1-boundary.mjs
tools/u1boundary/**
spec/schema/v1/**
spec/examples/v1/**
spec/vectors/v1/**
docs/*.md
docs/decisions/**
docs/prompts/**
docs/status/DIDRUN_BUGS.md
docs/status/U1.md
```

Do not create `cmd/`, `internal/gitobj`, `internal/world`, either adapter, store, server, report, web, or Node contract harness. `internal/reduce/model.go` contains pure sums/invariants only; orchestration belongs to U5.

## Implementation sequence

1. Initialize the smallest local Go module consistent with U0. Keep dependencies at zero unless a dependency is justified in writing and vendoring/licensing implications are recorded; standard library is preferred.
2. Build a strict JSON/token reader and canonical encoder. Parse first into project-owned lossless tokens, validate, then construct typed values. Return typed errors with stable machine codes and byte offsets, never partial identity.
3. Implement domain-separated digests and golden vectors. Include nested ordering, escaped keys, missing/present-empty, boundary integers, invalid UTF-8, duplicate names, `-0`, exponent/decimal rejections, lone surrogates, and kind-domain separation.
4. Implement deterministic `WorldPlan` compilation from inert spec: cap source bytes and every collection/string before authority, materialize every default and budget, preserve ordered argv/multimaps, sort only fields declared unordered, and choose multi-error refusals deterministically. A source may name a projection-definition digest, but compilation requires the resolved opaque full definition and refuses digest or adapter-domain mismatch; a bare digest can never produce a plan. Admit only exact adapter/version/shape/readiness combinations, a bare required tool plus the bounded closed argument profile, and the five exact public environment literals. Reject wrapper/shell/host-path syntax, secret values in stable bytes, unknown nested keys, impossible budgets, NUL/control values, and ambient interpolation. Treat candidate count as a declared budget input until U2 explicitly binds it to the selected roster.
5. Implement execution-attempt and artifact-lineage sum types plus edge-specific guarded constructors. Do not expose a generic `(stage,digest)` artifact or adjacency constructor. Staleness creates a new status object; it never mutates old bytes. Illegal paths—ruling before confirmation, reuse under changed plan, emission request from a noncompilable action—must be rejected before I/O.
6. Implement the sealed chain `WorldPlan + CandidateExecutionBinding + stimulus + attempt identity -> WorldInstance -> InstanceMeasurements -> AssessComparison`. Admission tokens remain deterministic, internal, and ephemeral. Implement centralized eligibility and `StableBatch` classification over tagged captured-or-controlled trial facts. Exact counts and histograms are preserved. Finite agreement is named only `OBSERVED_STABLE(k/k,h)`; identity-level nonduplication is not physical freshness.
7. Implement exact `candidate:<64 lowercase hex>` `CandidateExecutionKey`, `ProjectionFingerprint`, complete-roster `CandidateOutcomeMap`, and its `DivergentBaseline` refinement. Derive display clusters only as identity-free views. Name the evidence address `OutcomeArtifactDigest` and the comparison identity `PreservationMapDigest`; types must prevent substitution. Expose one typed comparability-first operation over full maps; reducers must never compare a naked preservation digest.
8. Implement ruling validation and the selected-field separation obligation over a closed typed field identifier supplied by a domain. `ALLOW_OBSERVED` consumes a sealed confirmed-outcome set and selects observed IDs; allowed/disallowed tuples are derived as a complete partition. Only separately reviewed `CUSTOM_EXPECTATION` may author a tuple. Empty fields, unknown fields, omissions, collisions, or cross-product allow-many return typed refusal. Keep compile eligibility as a sealed sum usable by U6, not a boolean.
9. Add property, fuzz, model-state, semantic canonicalization, and permutation tests. Keep target wire codecs explicitly `UNRECEIPTED` for U6. Update U1 status/handoff with exact nonclaims.

## Acceptance, negative cases, and required mutants

Tests must establish deterministic bytes/digests across repeated runs; parse/encode idempotence; order independence only where declared; candidate permutation invariance; label/producer/support-count irrelevance; explicit exclusion identity; all eligibility classifications; illegal lineage transitions; receipt string round-trip; and separation for allow-one, tuple-set allow-many, custom expectation, empty, missing, and present-empty.

Create a dependency-free mutation driver that copies only an explicit regular-file allowlist into a private temporary directory, refuses symlinks/nonregular inputs, uses a minimal private environment, verifies every exact mutation anchor once, establishes that the exact named test passes unmutated, applies one mutation, and requires a JSON test event showing that exact test failed. Infrastructure failure is not a killed mutant. The exact eighteen unique IDs are fixed: `remove-digest-kind-separator`, `bypass-duplicate-key-rejection`, `accept-negative-zero`, `drop-candidate-key-from-map-identity`, `replace-labeled-map-with-partition-shape`, `turn-control-failure-eligible`, `collapse-unresolved-into-changes`, `permit-incomplete-one-minimal-sweep`, `allow-empty-field-selection`, `compile-allow-many-as-independent-sets`, `bypass-programmatic-array-profile`, `bypass-world-plan-binding`, `drop-measurement-set-from-admission`, `ignore-trial-stimulus-binding`, `ignore-shared-admission-set`, `ignore-reduction-phase-binding`, `allow-reused-captured-observation`, and `ignore-projection-definition-binding`. A missing, duplicate, replaced, or multiply anchored ID fails the run. Add synthetic driver self-tests for secret exclusion, symlink refusal, anchor errors, missing tests, baseline failure, and infrastructure failure.

All load-bearing commands go through didrun. At minimum run:

```text
didrun run -- /usr/bin/env -i HOME="$PWD/.countershape/verify/home" TMPDIR="$PWD/.countershape/verify/tmp" LANG=C TZ=UTC NO_COLOR=1 PATH=/opt/homebrew/bin:/usr/bin:/bin /opt/homebrew/bin/node tools/validate-planning.mjs
didrun run -- /usr/bin/env -i HOME="$PWD/.countershape/verify/home" TMPDIR="$PWD/.countershape/verify/tmp" LANG=C TZ=UTC NO_COLOR=1 PATH=/opt/homebrew/bin:/usr/bin:/bin /opt/homebrew/bin/node tools/validate-planning.mjs --self-test
didrun run -- /usr/bin/env -i HOME="$PWD/.countershape/verify/home" TMPDIR="$PWD/.countershape/verify/tmp" LANG=C TZ=UTC NO_COLOR=1 PATH=/opt/homebrew/Cellar/go/1.26.5/libexec/bin:/usr/bin:/bin GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOFLAGS= GOCACHE="$PWD/.countershape/verify/gocache" GOMODCACHE="$PWD/.countershape/verify/gomodcache" GOPATH="$PWD/.countershape/verify/gopath" CGO_ENABLED=1 /opt/homebrew/Cellar/go/1.26.5/libexec/bin/go test ./...
didrun run -- /usr/bin/env -i HOME="$PWD/.countershape/verify/home" TMPDIR="$PWD/.countershape/verify/tmp" LANG=C TZ=UTC NO_COLOR=1 PATH=/opt/homebrew/Cellar/go/1.26.5/libexec/bin:/usr/bin:/bin GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOFLAGS= GOCACHE="$PWD/.countershape/verify/gocache" GOMODCACHE="$PWD/.countershape/verify/gomodcache" GOPATH="$PWD/.countershape/verify/gopath" CGO_ENABLED=1 /opt/homebrew/Cellar/go/1.26.5/libexec/bin/go test -race ./...
didrun run -- /usr/bin/env -i HOME="$PWD/.countershape/verify/home" TMPDIR="$PWD/.countershape/verify/tmp" LANG=C TZ=UTC NO_COLOR=1 PATH=/opt/homebrew/Cellar/go/1.26.5/libexec/bin:/usr/bin:/bin GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOFLAGS= GOCACHE="$PWD/.countershape/verify/gocache" GOMODCACHE="$PWD/.countershape/verify/gomodcache" GOPATH="$PWD/.countershape/verify/gopath" CGO_ENABLED=1 /opt/homebrew/Cellar/go/1.26.5/libexec/bin/go vet ./...
didrun run -- /usr/bin/env -i HOME="$PWD/.countershape/verify/home" TMPDIR="$PWD/.countershape/verify/tmp" LANG=C TZ=UTC NO_COLOR=1 PATH=/opt/homebrew/bin:/usr/bin:/bin COUNTERSHAPE_GO=/opt/homebrew/Cellar/go/1.26.5/libexec/bin/go /opt/homebrew/bin/node tools/check-u1-boundary.mjs --self-test
didrun run -- /usr/bin/env -i HOME="$PWD/.countershape/verify/home" TMPDIR="$PWD/.countershape/verify/tmp" LANG=C TZ=UTC NO_COLOR=1 PATH=/opt/homebrew/bin:/usr/bin:/bin COUNTERSHAPE_GO=/opt/homebrew/Cellar/go/1.26.5/libexec/bin/go /opt/homebrew/bin/node tools/check-u1-boundary.mjs
didrun run -- /usr/bin/env -i HOME="$PWD/.countershape/verify/home" TMPDIR="$PWD/.countershape/verify/tmp" LANG=C TZ=UTC NO_COLOR=1 PATH=/opt/homebrew/bin:/usr/bin:/bin COUNTERSHAPE_GO=/opt/homebrew/Cellar/go/1.26.5/libexec/bin/go /opt/homebrew/bin/node tools/mutate-u1.mjs
didrun run -- /usr/bin/env -i HOME="$PWD/.countershape/verify/home" TMPDIR="$PWD/.countershape/verify/tmp" LANG=C TZ=UTC NO_COLOR=1 PATH=/opt/homebrew/bin:/usr/bin:/bin COUNTERSHAPE_GO=/opt/homebrew/Cellar/go/1.26.5/libexec/bin/go /opt/homebrew/bin/node --test tools/test-mutate-u1.mjs
didrun run -- /usr/bin/env -i HOME="$PWD/.countershape/verify/home" TMPDIR="$PWD/.countershape/verify/tmp" LANG=C TZ=UTC NO_COLOR=1 PATH=/usr/bin:/bin /usr/bin/git diff HEAD --check
# after staging the exact reviewed U1 set:
didrun run -- /usr/bin/env -i HOME="$PWD/.countershape/verify/home" TMPDIR="$PWD/.countershape/verify/tmp" LANG=C TZ=UTC NO_COLOR=1 PATH=/usr/bin:/bin /usr/bin/git diff --cached --check
didrun run -- /usr/bin/env -i HOME="$PWD/.countershape/verify/home" TMPDIR="$PWD/.countershape/verify/tmp" LANG=C TZ=UTC NO_COLOR=1 PATH=/usr/bin:/bin /usr/bin/git diff --cached --name-only
```

Create the private HOME/TMP/cache directories first through didrun. These fixed profiles are part of the verification contract: `env -i` deliberately omits `NODE_OPTIONS`, `NODE_PATH`, ambient Go configuration, shell aliases, and user `PATH` entries. If an absolute toolchain path changes, stop and update the declared profile plus evidence; never fall back silently to ambient `go` or `node`.

Run every Go fuzz target separately by exact name for at least 20 seconds through didrun and claim each relevant event; this avoids silently omitting a package or target. Run a deterministic repeat check against fresh temporary caches if the suite supplies one. Do not claim Linux runtime from cross-compilation.

Use `didrun show --session`, then attach explicit claims to every exact successful load-bearing event: planning normal and self-test; Go normal, race, vet, and each exact fuzz target; boundary normal and self-test; mutation-driver selftests and all eighteen A/B/A experiments; full candidate diff; staged diff; and staged-name inventory. Anything omitted remains `UNRECEIPTED`. Stage only U1 files, inspect the staged diff, and commit `feat: establish the Countershape truth kernel`. No AI co-author trailer. Seal HEAD and loop on `NO_COLOR=1 didrun verify --strict` until exit 0. On failure, fix the real issue, rerun via didrun, make new claims, create a new fix commit if the failed commit was sealed, reseal, and reverify. Never weaken tests, remove/relabel claims, or erase failed receipt history.

## Stop/fallback and handoff

Stop before subprocess work if any canonical, map-identity, state-machine, eligibility, separation, or required mutation gate fails. The honest fallback is a still-open truth-kernel prototype, not U1 complete. If strict parsing cannot support a proposed portable value, cut that value from v1 and revise the controlling spec explicitly; do not silently use ordinary JSON semantics.

Finish with updated `docs/status/U1.md` and Mode C handoff, then report branch, commit(s), exact didrun grades, strict exit, tests/mutants run, environment claims, remaining UNRECEIPTED items, and `P02-U2-GIT-WORLD-SUBSTRATE.md` as the next prompt.
