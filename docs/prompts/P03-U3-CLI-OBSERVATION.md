# P03 / U3 — Prove the typed CLI observation spine

## Mission

In a fresh chat, build Countershape’s first complete observation domain: one typed CLI invocation per fresh attempt, bounded capture, explicit projection, rotated repeated trials, honest stability classification, and exact N-way outcome maps. Demonstrate the configuration-precedence domain without adding reduction, Choicepoint persistence, contract emission, end-user CLI commands, HTTP, server, report, or UI.

Proceed only from a strict-green U2. Preserve existing work, narrate compact phase updates, and keep status artifacts current.

## Required read set

Read fully: `/Users/drewnelson/.claude/CLAUDE.md`; `docs/CONCEPT_BRIEF.md`; `docs/ARCHITECTURE.md`; `docs/SEMANTICS.md`; `docs/PROJECTION_ALGEBRA.md`; `docs/THREAT_MODEL.md`; `docs/STATE_MACHINES.md`; `docs/CLAIM_VOCABULARY.md`; `research/deep-dive/02-architecture-correctness.md`; `research/deep-dive/04-product-dx.md`; `research/deep-dive/05-contract-portability.md`; `research/deep-dive/07-SYNTHESIS.md`; `research/deep-dive/08-RED_TEAM.md`; `docs/status/U1.md`; `docs/status/U2.md`; `docs/HANDOFF_MODE_C.md`; relevant U0 specs/vectors; and U1/U2 public types. Verify branch, working status, U2 seal, and `NO_COLOR=1 didrun verify --strict` before editing.

## Locked semantics and cuts

- CLI facts stay in `internal/adapters/cli`; do not coerce them into HTTP-shaped or generic JSON observation DTOs. Generic code may own only eligibility, scheduling/classification, exact-map comparison, and lineage-neutral orchestration.
- One stimulus is exactly one direct executable plus ordered argv, tagged stdin (`Absent` versus `Present(empty)` versus bytes), sparse tagged environment (`Absent` differs from `Present("")`), bounded regular fixture files, and a declared working-directory policy. No shell, script string, setup command, package install, arbitrary workflow, ambient HOME, or inherited environment.
- Every evidentiary trial uses U2’s new private roots and actual process lifecycle. No result cache or reuse. Fixture-observed invocation evidence is stored outside the selected behavior projection.
- Capture distinguishes normal exit and signal termination; stdout and stderr are distinct bounded byte channels. A nonzero exit and complete empty stdout can be eligible behavior. Timeout, cancellation, output cap, materialization/start/orphan/teardown failure, or projection rejection are controls and cannot enter an outcome.
- `CapturedObservation` means bytes after the declared capture policy. Call bytes raw only when unchanged and durably persisted. Projection is pure, versioned, visible, and returns exact canonical bytes plus source links or a typed rejection.
- Use a closed CLI projection field registry sufficient for the reference study: exit kind/code and the exact strict-JSON stdout fields locked by U0 (for example precedence source/value). Do not introduce arbitrary dotted paths, regex, tolerant comparison, implicit JSON coercion, or custom code.
- Repetition produces only `OBSERVED_STABLE(k/k,h)`, `UNSTABLE(histogram)`, `UNCOMPARABLE(reasons)`, or `INCOMPLETE`. Never say deterministic. A divergence contains at least two eligible candidates and two fingerprints.
- The complete `CandidateOutcomeMap` crosses the adapter boundary, including plan, comparison basis, phase, exclusions, shared per-repetition admission set, and globally unique evidence identities. Labels, refs, producer provenance, order, support counts, and ordinal clusters are display-only and cannot alter identity.
- `ComparisonEnvelope` is policy. U3 supplies complete `InstanceMeasurements`; `AssessComparison` persists one exact matrix admission per repetition or refuses before batching. A candidate key cannot substitute for its opaque binding and matching plan. Root/PID/timing placeholders in a projection never prove those dimensions irrelevant to prior control flow.

## Exact ownership

U3 owns:

```text
internal/adapters/cli/**
internal/observe/schedule.go
internal/observe/batch.go
testkit/clifixture/**
testkit/studies/cli_precedence/**
tools/check-u3-architecture.mjs
tools/mutate-u3.mjs
docs/status/U3.md
docs/HANDOFF_MODE_C.md
```

It may make additive corrections to U1/U2 interfaces only when required by the controlling contracts. No `cmd/`, HTTP adapter, reducer orchestration, store, choice emitter, Node harness, server, report, or web files.

## Implementation sequence

1. Define typed `CLIStimulus`, `CLICapturePolicy`, `CLICapturedObservation`, `CLIProjectionDefinition`, and reducer-neutral stimulus measure data. Preserve argv order and all absent/empty distinctions in canonical identity. Validate fixture paths before U2 receives them.
2. Compile the CLI portion of an inert source spec into a deterministic `WorldPlan` containing only logical tool names and repository-relative argv. Allocate structural `WorldInstance` from the opaque candidate binding plus actual matching plan. Resolve the actual executable and private HOME/TMPDIR/state paths per attempt into `InstanceMeasurements` or U2 process receipts; never rewrite those host facts into stable plan or structural-world identity. Display the trusted-code/`HOST_ALLOWED` warning in plan metadata.
3. Adapt U2 lifecycle evidence into CLI capture without losing exit-versus-signal, stdout/stderr, byte counts, truncation/control facts, attempt marker, and fixture invocation receipt. Only the central U1 eligibility service may admit projection.
4. Implement visible pure projection operations: channel requirement, UTF-8 validation where declared, strict JSON token parsing through `internal/canon`, closed field extraction with tagged missing/present-empty, and canonical result encoding. Record every operation and source link. Do not silently delete stderr, paths, or volatile fields.
5. Implement deterministic sequential schedules with candidate rotation between repeats. Schedule order belongs to evidence but not outcome identity. Enforce disclosed trial/wall/output budgets before and during execution.
6. Assess one complete candidate measurement matrix per repetition. A rejection stops before batching. Assemble admitted tagged trials into `StableBatch`, require peer batches to share the exact ordered admission set, and then construct U1 `CandidateOutcomeMap`; adapters never construct a looser group representation. Preserve exclusions and reasons separately.
7. Build dependency-free Node-core CLI fixture candidates for config precedence: one favors config, one environment, one argv. A private HOME must neutralize an ambient user-config sentinel. Include constant, deliberately alternating, timeout, output-limit, signal, malformed-projection, empty-output, and incomplete-budget fixtures. Invocation instrumentation must prove a process actually ran and remain outside projection.
8. Produce a test-only study result that makes the precedence split and Captured-to-Projection operations inspectable. This is observation evidence, not a human Choicepoint or contract.
9. Update status/handoff with the exact fields supported and all nonclaims.

## Acceptance, negative cases, and mutants

Tests must cover: canonical stimulus identity; argv order; stdin/environment absent versus empty; fixture order/path validation; sparse environment and private HOME; fresh unique roots and invocation receipts; exit 0/nonzero versus signal; independent stdout/stderr; exact output-cap boundaries; eligible empty output under a bytes projection; rejected empty output under strict-JSON projection; timeout/cancel/output/start/teardown controls; `k/k` constant classification; alternating histogram; projection `UNCOMPARABLE`; budget `INCOMPLETE`; rotated schedule; permutation-invariant exact map; and exclusions outside the map.

The precedence study must prove ambient HOME cannot change a result and all three eligible candidates yield distinct exact projected values. Re-running with labels, producer metadata, and candidate order permuted must preserve the map digest while creating fresh attempts.

The architecture checker must fail if generic packages import the CLI adapter, switch on CLI field names, or define a generic HTTP/CLI JSON capture. The mutation driver must anchor and kill at least: reorder argv; collapse absent/empty stdin; inherit ambient environment; swap stdout/stderr; classify nonzero exit as infrastructure failure; accept output-limit as a value; turn timeout into exit code; classify an alternating candidate by majority/last trial; reuse a prior attempt; hide a projection operation; and include label/support count in outcome identity. Missing anchors fail the mutation run.

Wrap every load-bearing command through didrun, including at minimum:

```text
didrun run -- go test ./internal/adapters/cli ./internal/observe ./testkit/clifixture ./testkit/studies/cli_precedence
didrun run -- go test -race ./internal/adapters/cli ./internal/observe
didrun run -- go test ./testkit/studies/cli_precedence -count=10
didrun run -- node tools/check-u3-architecture.mjs
didrun run -- node tools/mutate-u3.mjs
didrun run -- git diff --check
```

Add a didrun-wrapped golden study command if the test package exposes one, and record actual trial counts/timing without upgrading it to the final 15-minute two-study claim.

Use `didrun show --session`; attach explicit claims with commands of the form `didrun claim tests-pass --label "U3 CLI observation" --event N --path .` to exact normal/race/repeat/architecture/mutation events, plus a scoped diff-check claim. Stage only U3 files, inspect, commit `feat: add typed CLI observation`, seal HEAD, and loop `NO_COLOR=1 didrun verify --strict` until exit 0. Any failure requires a real fix, new wrapped run, new event-bound claim, a new fix commit if the failed commit was sealed, reseal, and reverify. Never weaken tests, relabel/delete claims, or rewrite failed receipt history.

## Stop/fallback and handoff

Stop if any control can project, fresh execution is not physically demonstrated, CLI facts leak into generic truth, map identity depends on grouping/order/metadata, or required mutants survive. The honest fallback is an unsealed observation prototype; do not claim divergence readiness. If didrun misbehaves, log the exact S6 finding and mark affected claims `UNRECEIPTED`.

Update `docs/status/U3.md` and Mode C, then report commit(s), verbatim receipt grades, strict exit, exact Darwin/Node version actually exercised, supported fields, classification matrix, timings, nonclaims, and `P04-U4-HTTP-OBSERVATION.md` as the sole next prompt.
