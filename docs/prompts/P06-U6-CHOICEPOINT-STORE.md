# P06 — U6 immutable Choicepoint store, physical confirmation, and ruling boundary

Implement the first half of Countershape U6 in a fresh chat. Begin by reading `docs/CONCEPT_BRIEF.md`, `research/deep-dive/02-architecture-correctness.md`, `research/deep-dive/03-execution-security.md`, `research/deep-dive/04-product-dx.md`, `research/deep-dive/07-SYNTHESIS.md`, `research/deep-dive/08-RED_TEAM.md`, and the current Mode C handoff. Confirm U5 is committed, sealed, and green under `NO_COLOR=1 didrun verify --strict`. If the repository or handoff disagrees with this prompt, the living brief and verified code win; document the discrepancy instead of silently inventing a third contract.

## Objective

Build a content-addressed immutable semantic artifact store, atomic study-head compare-and-swap, physical fresh-confirmation transition, immutable Choicepoint lineage, and blind-first ruling application boundary. At the end, only an exact freshly confirmed witness may become `CHOICEPOINT_READY`; only a ruling against the current exact Choicepoint digest may advance; stale clients and illegal actions must create no derived artifact or files.

This is not a Wake-style event ledger, process replay engine, command recorder, crash-resume system, or didrun substitute. Completed semantic objects are durable; in-flight progress is disposable. A crash may leave a typed partial attempt, which must be rerun in a new world rather than resumed.

## Locked authorities and invariants

Each persisted semantic object has a schema version, kind, strict canonical bytes, and the existing domain-separated SHA-256 digest. The store must not parse an object and reserialize it through a lossy ambient JSON path before identity. Object bytes are immutable after atomic publication.

The `OutcomeMap` field is the one canonical sorted `candidate_execution_key -> projection_fingerprint` map digest from `internal/compare`. Baseline, reduction, confirmation, Choicepoint, blind DTO, and ruling validation all bind that same typed digest. Display clusters and ordinal group IDs are derived views without identity.

Fresh confirmation is physical, not a metadata relabel. It requires, for every trial, an attempt artifact created before spawn, new owned roots and process lifecycle, new captured/projection evidence, a fixture-observed invocation fact outside the selected predicate, and a rotated candidate schedule. It may not reuse discovery or shrink execution evidence. Pure transformations of immutable captured bytes may be cached; executed observations may not.

The human can author `ALLOW_OBSERVED`, `CUSTOM_EXPECTATION`, `REJECT_ALL`, or `DEFER`; `REFINE` creates a successor study. Only allow-observed and custom-expectation are potentially compilable later. `REJECT_ALL`, `DEFER`, and `REFINE` must be impossible emitter inputs by type, not merely hidden in a UI.

## File and package ownership

Own `internal/store/**`, `internal/choice/**`, their tests, and narrowly required application-service wiring. Add domain values only where the locked schema needs construction-safe types. Add fixture instrumentation under `testkit/**`. If an API DTO package already exists, add server-independent DTO constructors there; do not build the HTTP server or React UI. Do not implement the Node bundle in this prompt.

Use a local layout equivalent to `.countershape/objects/sha256/<prefix>/<digest>`, per-study head records, and separately permissioned private captured artifacts. Exact paths may follow established repository conventions. Stable semantic objects contain no absolute root, current time, random title, branch nickname, or host formatting unless that fact belongs explicitly to measured evidence rather than identity.

## Implementation passes

1. Implement write-once object publication: validate canonical bytes and expected digest, create an owned private temporary file, write completely, sync as supported, atomically rename, then reopen/verify. Existing identical bytes are idempotent; an existing path with different bytes is corruption and fails closed. Reject traversal, symlink parents, case/Unicode path collisions, wrong modes, and digest aliases.
2. Implement atomic head compare-and-swap with `study_id`, expected revision/digest, next digest, lineage relation, and typed stale/corrupt errors. A semantic ancestor change creates a successor lineage and makes old descendants stale; no old artifact is edited into freshness. Concurrent writers yield exactly one winner.
3. Encode legal lineage transitions from source plan through baseline, divergence, reduction result, fresh confirmation, `CHOICEPOINT_READY`, ruling, and later residue. `CANCELLED`/`PARTIAL` retain inspectable evidence without advancing. Added candidates, changed plan/projection/repeat/reducer, old confirmation, or ruling against a superseded digest must reject.
4. Add the confirmation service. It rotates the schedule relative to discovery, obtains new attempt/evidence links from the existing world/observe services, verifies fixture invocation evidence, and requires the exact confirmed `OutcomeMap` to equal the reduced baseline map. Any unstable, uncomparable, incomplete, stale, or reused trial prevents readiness. A new nonce without new execution facts is insufficient.
5. Construct immutable Choicepoints containing exact tree/candidate identities, `WorldPlan` and declared comparison-envelope identity, original/minimized stimuli, projection/capture identity, baseline and confirmed maps, observed-stable counts, exclusions, reduction transcript/grade, and evidence links. Avoid labels implying compatible or deterministic worlds.
6. Build a **blind DTO** from server-computed truth. It includes human-authored scenario, exact witness, original/minimized stimuli, derivation, reduction grade/budgets/unresolved count, observed and confirmation counts, projection operations, deterministic candidate-neutral outcome facts, differing fields, and explicit trust warnings. It must omit candidate identities, ref/branch names, producer/model provenance, total candidate count, per-outcome support count, support-derived ordering, DOM hints, and reveal data. Distinct outcome count is necessarily visible. Omission must occur in DTO construction, not CSS.
7. Build reveal and ruling commands separately. Record provisional action, selected versus context-only fields, required evidence-view interaction facts, explicit reveal/early-reveal fact, provenance reveal, post-reveal change and rationale, and final action. Visited facts prove presentation only, never comprehension or debiasing.
8. Enforce the selected-field **separation obligation** before marking an allow-observed ruling compilable: for every allowed confirmed outcome and disallowed confirmed outcome, their complete selected-field typed tuples must differ. Missing and present-empty are distinct. Allow-many is a canonical set of complete tuples, never a cross-product. No fields means `NO_SELECTED_FIELDS`; collision means `AMBIGUOUS_SCOPE`; suggestions may identify candidate-neutral differing fields but cannot auto-select them. Custom expectations use the same closed typed registry and require explicit confirmation when duplicating an observed tuple.
9. Store didrun references as opaque verbatim strings. Unknown grades round-trip unchanged; absence is `UNRECEIPTED`. Countershape never interprets a grade as conformance or promotes it.

## Required properties, attacks, and mutations

Add deterministic unit, property, model/state-machine, filesystem, race, and integration tests proving:

- canonical write/read identity, idempotence, corruption detection, safe path handling, private modes, and no mutation after publication;
- two concurrent CAS writers have one winner; stale tabs, stale digests, replayed commands, and changed ancestors produce no ruling/derived files;
- every illegal lineage edge rejects, including ruling before distinct confirmation and resuming reduction under a changed plan;
- copied old captures with new nonces fail physical freshness; new attempts plus fixture-observed invocation facts and rotated schedules pass;
- confirmation must reproduce the exact labeled map, not membership shape;
- blind DTO serialized bytes contain no candidate/ref/producer identity, support counts, majority cues, or hidden reveal material, including accessibility/metadata-shaped fields;
- blind order is deterministic and independent of input candidate order/support count;
- allow-one, allow-many, custom, reject-all, defer, refine, early reveal, and post-reveal-change transitions are exact;
- weak fields return `AMBIGUOUS_SCOPE` with no compilable artifact; allow-many never generates a field cross-product; no unselected field enters the predicate view;
- `REJECT_ALL`, `DEFER`, and `REFINE` cannot be converted to a compilable ruling even through direct application-service calls;
- unknown didrun strings round-trip byte-for-byte and missing receipt remains `UNRECEIPTED`.

Kill required mutants for: last-writer-wins head updates; mutable object overwrite; nonce-only freshness; discovery-batch reuse; map-shape confirmation; stale CAS accepted; identity merely CSS-hidden; support-count ordering; default-selected field; allow-many cross-product; missing equals empty; `REJECT_ALL` cast to compilable; and receipt grade normalization. Do not delete a mutation because it survives.

## didrun unit boundary

Wrap every load-bearing Go test, race test, property/model suite, mutation runner, lint/vet command, and fresh-confirmation fixture in `didrun run --`. Use `didrun show --session` to capture event indexes. Declare narrow event-bound claims with relevant paths and exact language; never claim confidentiality, hostile-code safety, comprehension, bias reduction, Linux, Node, or UI behavior here.

After review, stage only this prompt's files, commit the verified U6 store/Choicepoint unit without an AI co-author trailer, then run `didrun seal --commit HEAD` followed by `NO_COLOR=1 didrun verify --strict`. Nonzero means the unit is unfinished. Fix the underlying issue, rerun affected verification through didrun, add claims, create a correcting commit, reseal, and repeat until strict exits 0. Prior failed receipts remain history; never weaken tests/claims or relabel scope to pass.

## Handoff and honest fallback

Update the Mode C handoff with commit, exact grades, event indexes, strict verdict, store format/version, state transitions, freshness proof, blind DTO omissions, receipt behavior, remaining risks, and P07 as next. Anything not run through didrun is `UNRECEIPTED`.

Stop before contract emission if immutable CAS lineage is not construction-safe, physical confirmation can be forged from old evidence, exact map identity is lost, blind DTOs leak identities/support, or selected fields can accept a confirmed disallowed outcome. The honest fallback is an observation/reduction artifact store with no decision-ready or executable-residue claim.
