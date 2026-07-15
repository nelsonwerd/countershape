# P05 — U5 bounded, tri-valued reducer

You are implementing Countershape U5 in a fresh chat. Work in the repository root. Do not redesign the product from memory: first read `docs/CONCEPT_BRIEF.md`, `docs/SEMANTICS.md`, `docs/PROJECTION_ALGEBRA.md`, `research/deep-dive/02-architecture-correctness.md`, `research/deep-dive/06-feasibility-acceptance.md`, `research/deep-dive/07-SYNTHESIS.md`, `research/deep-dive/08-RED_TEAM.md`, and the prior unit handoff. Confirm that U4 is committed, sealed, and passes `NO_COLOR=1 didrun verify --strict`; if it does not, stop with an honest prerequisite handoff. Preserve user changes and do not amend or rewrite prior receipts.

## Objective

Build the pure bounded reduction orchestrator and the typed CLI and HTTP neighbor providers. A reduction may accept a smaller stimulus only when a physically fresh eligible evaluation is comparable to the full baseline map and then reproduces the exact canonical sorted `candidate_execution_key -> projection_fingerprint` map. Comparability requires the same plan, envelope, stimulus-independent comparison basis, expected roster, eligible set, exclusions, and exclusion classifications; the per-run admission artifact and purpose may differ. Partition shape, member sets, outcome count, support count, cluster ordinal, a naked digest, or a display DTO are never preservation authorities.

The unit must produce replayable provenance and only these grades:

- `UNCHANGED`: no smaller preserving stimulus was accepted;
- `BEST_KNOWN`: a preserving reduction exists, but a budget ended, the final sweep is incomplete, or any direct neighbor is unresolved; and
- `ONE_MINIMAL_UNDER(reducer_set_digest)`: every valid direct neighbor of the final stimulus was freshly evaluated and the durable result for each is `CHANGES`.

Do not claim a smallest behavior, global minimum, root cause, semantic equivalence, or correctness.

## Locked truth boundaries

`internal/compare` remains the authority for `CandidateOutcomeMap`, its typed `PreservationMapDigest`, and the opaque comparability-first result. The reducer consumes full typed maps or that opaque result; it must not compare a naked digest, reconstruct maps from display clusters, require identical per-run admission artifacts across different stimuli, or substitute `OutcomeArtifactDigest`. CLI and HTTP keep their own typed stimuli, well-founded measures, and deterministic neighbor functions. `internal/reduce` owns only search order, budgets, tri-valued orchestration, transcript construction, and validation of a store-issued final-sweep authority. It must not import Git, process, HTTP, CLI, store, or server implementations. A narrow `internal/store` edge slice persists the sweep and imports the inward contracts; dependency direction never reverses.

An evaluation is exactly `PRESERVES`, `CHANGES`, or `UNRESOLVED`. First assess full-map comparability. Incomparable maps are `UNRESOLVED`; comparable maps with equal exact labeled-map digests are `PRESERVES`; comparable maps with different exact labeled-map digests are `CHANGES`. Every unstable, incomplete, cancelled, timed-out, stale, or budget-precluded result is also `UNRESOLVED`. Never encode this as a Boolean.

No execution evidence may be cached or reused. Pure parsing and canonical transforms of immutable bytes may be memoized, but each evaluated neighbor needs new attempt artifacts, worlds, processes, and captures supplied by the evaluator. The reducer records evidence links and never asserts freshness from a nonce alone.

## File and package ownership

Primary ownership is `internal/reduce/**` and its tests. Add the minimal content-addressed reduction-sweep completion authority under `internal/store/**`; P06 will extend this substrate for study heads and Choicepoints. Add typed neighbor implementations and tests only under `internal/adapters/cli/**` and `internal/adapters/http/**`. Add reusable generated fixtures under `testkit/**` when necessary. Pure compare/domain changes are permitted only to expose an already-specified typed value; do not create a parallel digest or generic JSON stimulus. Do not touch the contract emitter, server, or web UI in this unit.

## Implementation passes

1. Define construction-safe value types for reducer-set identity, strictly decreasing measure, budgets, proposal identity, tri-valued evaluation, transcript entry, accepted path, final sweep, and result grade. Materialize every default budget in the compiled input. Reject zero/negative/overflowing limits and non-decreasing or duplicate proposals with typed errors.
2. Implement a deterministic search. Given identical typed neighbors and evaluation results, proposal order, accepted path, consumed counts, and transcript bytes must match. Every proposal records parent stimulus digest, neighbor digest, reducer name/version, before/after measure, evaluation batch digest, exact observed map digest when available, result, and budget accounting.
3. Implement authoritative wall, proposal, and total-candidate-trial budgets. Stop at the first exhausted bound. Budget exhaustion is a result, not a hidden retry. Cancellation immediately prevents a strong grade even if the final apparent neighbor just completed.
4. Implement the final direct-neighbor sweep as a durable certificate authority, not a Boolean flag or caller-authored digest. Atomically persist the baseline/reducer/measure binding, exact enumerated neighbor set, and complete fresh evaluation objects; reopen and verify the content-addressed record before producing the opaque authority consumed by `internal/reduce`. `ONE_MINIMAL_UNDER` must be constructible only when the enumerated valid neighbor set equals the set of fresh completed `CHANGES` records, with no unresolved, omitted, duplicate, stale, or post-budget member. An empty direct-neighbor set may certify only if enumeration itself completed durably. Crash-before-publication and corrupted-record tests must fail closed.
5. Implement finite deterministic typed neighbor providers for both domains. CLI must preserve ordered argv, `Absent` versus `Present("")`, sparse environment, fixture files, stdin, and exit-versus-signal semantics. HTTP must preserve method, path, ordered query/header multimaps, body, fixture seed, and the one-request profile. Each neighbor must be valid and strictly decrease the domain measure. Do not coerce either domain into generic JSON.
6. Produce compact CLI-readable transcript rendering, but keep rendering non-authoritative. Use the canonical map digest in every persisted/API-shaped reducer fact; ordinal cluster IDs are forbidden.

## Exact verification gates

Add table, property, state/model, integration, and deterministic mutation tests. At minimum prove:

- candidate input permutations yield the same baseline map digest and reducer decision;
- each emitted neighbor is valid, deterministic, unique, and strictly smaller;
- the synthetic same-partition baseline `A=403,B=404,C=404` and neighbor `A=200,B=500,C=500` is `CHANGES`, even though both partition as `{A}|{B,C}`;
- the physical U4 seed-authority neighbor keeps identical request bytes and the same eligible A/B/C roster but changes `403|404|200` to `200|500|500`; it is also `CHANGES`, while its partition is intentionally not claimed identical;
- candidate labels, order, producer, and support counts cannot change preservation;
- unstable, incomplete, teardown-error, cancelled, and stale evaluations are `UNRESOLVED` and force `BEST_KNOWN`;
- exhaustion immediately before, during, and immediately after the apparent last sweep item never fabricates `ONE_MINIMAL_UNDER`;
- a complete sweep with every direct neighbor freshly `CHANGES` earns the exact reducer-set-bound grade;
- a logical complete list, arbitrary digest, copied certificate, partial write, or corrupted durable record cannot earn that grade;
- transcript replay reconstructs proposals and decisions but triggers fresh execution rather than replaying captures;
- both CLI and HTTP produce a smaller preserving witness in checked-in fixtures;
- one fixture ends honestly at `BEST_KNOWN`, and another earns `ONE_MINIMAL_UNDER`;
- serialized/deserialized transcript and API-facing data retain the exact map digest and reject the shape trap.

Kill, by deterministic checked-in mutation cases, at least: full-map equality replaced by partition-shape equality; `UNRESOLVED` treated as `CHANGES`; final sweep skipped; cancellation ignored; budget fencepost promoted; map digest dropped during serialization; accepted non-decreasing neighbor; reused evaluation batch accepted as fresh; cluster ordinal treated as identity; and reducer-set digest omitted from the grade. Surviving required mutants block the unit. Strengthen implementation/tests; never delete, weaken, or relabel a mutant to open the gate.

## didrun, commit, seal, strict loop

Every load-bearing test, property suite, mutation suite, race check, vet/lint command, and end-to-end fixture command must run as `didrun run -- <command>`. Record exact commands and event indexes in the handoff. Inspect them with `didrun show --session`, then declare narrowly scoped `tests-pass`, `lint-clean`, or `command-succeeded` claims bound to the successful events and relevant paths. Do not claim a platform, domain, or grade that was not actually exercised.

When the unit is green: review `git diff`, stage only U5-owned changes, create an intentional U5 commit with no AI co-author trailer, then `didrun seal --commit HEAD`. Run `NO_COLOR=1 didrun verify --strict`. A nonzero exit means U5 is not done: diagnose the real issue, make a correcting change, rerun all affected load-bearing verification through didrun, declare new claims, commit, reseal, and repeat strict verification until exit 0. Never delete old claims or receipts, weaken tests, or relabel scope. Do not begin U6 before the strict gate is green.

## Handoff and stop conditions

Update the Mode C handoff with the commit, exact strict verdict, commands/event indexes, verbatim claim grades, changed paths, result-grade examples, remaining risks, and the next prompt. Label any capability not exercised through didrun `UNRECEIPTED`.

Stop at the prior observation-only milestone if exact map identity cannot cross persistence/serialization intact, if either adapter requires domain branches inside the generic reducer, if required mutants survive, or if fresh evidence cannot be distinguished from reused captures. A useful reducer that cannot finish a complete sweep may ship only with `BEST_KNOWN`; do not manufacture local minimality.
