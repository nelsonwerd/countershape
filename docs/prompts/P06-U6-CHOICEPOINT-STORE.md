# P06 — U6 immutable Choicepoint store, physical confirmation, and ruling boundary

Implement the first half of Countershape U6 in a fresh chat. Begin by reading `docs/CONCEPT_BRIEF.md`, `docs/SEMANTICS.md`, `docs/PROJECTION_ALGEBRA.md`, `research/deep-dive/02-architecture-correctness.md`, `research/deep-dive/03-execution-security.md`, `research/deep-dive/04-product-dx.md`, `research/deep-dive/07-SYNTHESIS.md`, `research/deep-dive/08-RED_TEAM.md`, and the current Mode C handoff. Confirm U5 is committed, sealed, and green under `NO_COLOR=1 didrun verify --strict`. If the repository or handoff disagrees with this prompt, the living brief and verified code win; document the discrepancy instead of silently inventing a third contract.

## Objective

Extend U5's narrow content-addressed sweep store into a bounded v1 semantic object store and one fixed Darwin study-head spine; add atomic compare-and-swap, physical fresh-confirmation transition, immutable Choicepoint lineage, and a blind-first ruling application boundary. At the end, only an exact freshly confirmed witness may become `CHOICEPOINT_READY`; only typed transitions carrying live or opaque predecessor-bound authority may publish sensitive stages; stale clients and illegal actions must create no derived artifact or files.

This is not a Wake-style event ledger, process replay engine, command recorder, crash-resume system, or didrun substitute. Completed semantic objects are durable; in-flight progress is disposable. A crash may leave already-finalized immutable objects or previously persisted nonadvancing evidence. U6 writes no crash-time `PARTIAL` or status object and cannot resume; a retry allocates an entirely new confirmation run.

### Implementation-time corrections (locked)

The verified implementation and strict runtime parsers supersede six planning phrases in the original prompt:

1. The empty-field refusal is `EMPTY_SELECTED_FIELDS`, not `NO_SELECTED_FIELDS`.
2. `CUSTOM_EXPECTATION` must differ from every confirmed selected-field tuple. An expectation equal to an observed tuple is refused as `CUSTOM_EXPECTATION_ALREADY_OBSERVED`; use `ALLOW_OBSERVED` for an observed tuple.
3. Fresh confirmation requires the exact labeled eligible map and the exact exclusion key/classification map. A freshly executed excluded candidate may remain excluded with the same classification—for example HTTP candidate D may again be freshly `UNSTABLE`. A changed excluded key or classification is incomparable and blocks readiness.
4. The durable readiness stage is `CHOICEPOINT_READY`; `DECISION_READY` is not a U6 store stage.
5. Immutable object publication on the receipted Darwin path uses a same-directory hard-link create-if-absent operation followed by reopen/rehash. Replacing rename is reserved for the mutable head file.
6. The required U6 mutation manifest is the frozen 17-mutant list below. Adding or removing a mutant is a contract change, not test cleanup.

## Locked authorities and invariants

Each persisted semantic object has a schema version, kind, strict canonical bytes, and the existing domain-separated SHA-256 digest. The store must not parse an object and reserialize it through a lossy ambient JSON path before identity. Object bytes are immutable after atomic publication.

Baseline, reduction, confirmation, Choicepoint, blind DTO, and ruling validation bind the full typed `CandidateOutcomeMap` or an opaque comparability result. They first require the same plan, envelope, stimulus-independent comparison basis, roster, eligible set, and exclusions; only then may the exact sorted eligible candidate-to-projection digest decide preservation. Concrete admission matrices and evidence differ across fresh phases. A naked digest, display cluster, or ordinal group ID has no comparison authority.

Fresh confirmation is physical, not a metadata relabel. It requires, for every trial, an attempt artifact created before spawn, new owned roots and process lifecycle, new captured/projection evidence, a fixture-observed invocation fact outside the selected predicate, and a rotated candidate schedule. It may not reuse discovery or shrink execution evidence. Pure transformations of immutable captured bytes may be cached; executed observations may not.

The local caller can author `ALLOW_OBSERVED`, `CUSTOM_EXPECTATION`, `REJECT_ALL`, `DEFER`, or a semantic `REFINE` request. U6 records caller-asserted operator attribution but does not authenticate a human actor. `REFINE` durable promotion is refused with `REFINE_REQUIRES_SUCCESSOR_STUDY` because construction-safe successor-study creation is not implemented. Only `ALLOW_OBSERVED` and separately reviewed `CUSTOM_EXPECTATION` may yield U6 `CompilableRuling` authority; P07 owns consuming that authority for emission. `REJECT_ALL`, `DEFER`, and `REFINE` must be impossible emitter inputs by type, not merely hidden in a UI.

## File and package ownership

Own `internal/store/**`, `internal/choice/**`, their tests, and narrowly required application-service wiring. Add domain values only where the locked schema needs construction-safe types. Add fixture instrumentation under `testkit/**`. If an API DTO package already exists, add server-independent DTO constructors there; do not build the HTTP server or React UI. Do not implement the Node bundle in this prompt.

Use a local layout equivalent to `.countershape/objects/sha256/<prefix>/<digest>`, per-study head records, and separately permissioned private captured artifacts. Exact paths may follow established repository conventions. Stable semantic objects contain no absolute root, current time, random title, branch nickname, or host formatting unless that fact belongs explicitly to measured evidence rather than identity.

## Implementation passes

1. Generalize the already-receipted U5 write-once sweep publication primitive: validate canonical bytes and expected digest, create an owned private same-directory temporary file, write completely, sync as supported, publish with hard-link create-if-absent, unlink the temporary name, then reopen/verify. Existing identical bytes are idempotent; an existing path with different bytes is corruption and fails closed. Reject traversal, symlink parents, case/Unicode path collisions, wrong modes, and digest aliases. Do not fork a second storage implementation.
2. Implement atomic head compare-and-swap with `study_id`, expected revision/digest, next digest, lineage relation, and typed stale/corrupt errors. The raw CAS primitive is unexported. Exported head methods accept only exact stage-specific semantic inputs or opaque predecessor-bound publication capabilities; serialized `FreshConfirmation`, `Choicepoint`, or `DecisionRecord` bytes cannot mint transition authority. A changed ancestor cannot advance the existing fixed spine and must use a distinct study or future construction-safe successor service. Concurrent writers yield exactly one winner.
3. Encode the bounded stage table `SOURCE_PLAN -> BASELINE -> DIVERGENCE -> REDUCTION -> CONFIRMATION -> CHOICEPOINT_READY -> RULING -> RESIDUE`, while exposing public typed transitions only through `RULING` in U6a. `RESIDUE` is the reserved P07 terminal slot and must have no generic/public U6a advance method. `PARTIAL` and `CANCELLED` are semantic dispositions but do not advance this head. Branching successor lineages, archived head history, durable `STALE`/`INVALIDATED` records, deferred reopening, deletion, recovery, and retention remain future work and must be called `UNRECEIPTED`. Added candidates, changed plan/projection/repeat/reducer, old confirmation, or a ruling against a superseded digest must reject.
4. Add the confirmation service. It rotates the schedule relative to discovery, obtains new attempt/evidence links from the existing world/observe services, verifies fixture invocation evidence, and applies the full-map comparability operation before requiring the confirmed eligible labeled-map digest to equal the reduced baseline. Concrete admission matrices and phase differ; the comparison basis does not. Any eligible-candidate instability, uncomparable or incomplete eligible evidence, changed eligible/excluded disposition, changed exclusion classification, stale authority, or reused trial prevents readiness. A freshly executed exclusion may remain excluded only with the exact same key and classification. A new nonce without new execution facts is insufficient.
5. Construct immutable Choicepoints containing exact tree/candidate identities, `WorldPlan` and declared comparison-envelope identity, original/minimized stimuli, projection/capture identity, baseline and confirmed maps, observed-stable counts, exclusions, reduction transcript/grade, and evidence links. Avoid labels implying compatible or deterministic worlds.
6. Build a server-independent **blind semantic DTO kernel** from canonical truth. It includes caller-authored scenario text, exact scope, original/minimized stimuli, bounded reduction facts, configured discovery/confirmation repeats, projection operations, deterministic candidate-neutral outcome cards, and trust warnings. It must omit system-supplied candidate identities, ref/branch names, producer/model provenance, total candidate count, per-outcome support count, support-derived ordering, DOM hints, and reveal data. Distinct outcome count is necessarily visible. This is not yet the complete U8 presentation payload: comparison-envelope summary, actual observed counts, reducer names/proposal derivation, trusted-code execution warning, DOM/accessibility behavior, and comprehension are `UNRECEIPTED`. Free-form scenario and behavior bytes may themselves reveal identity; the claim is structural omission of system-supplied identity/support metadata, not anonymity or content-taint safety.
7. Build reveal and ruling commands separately. Record provisional action, selected versus context-only fields, required evidence-view interaction facts, explicit reveal/early-reveal fact, provenance reveal, post-reveal change and rationale, and final action. Visited facts prove presentation only, never comprehension or debiasing.
8. Enforce the selected-field **separation obligation** before marking an allow-observed ruling compilable: for every allowed confirmed outcome and disallowed confirmed outcome, their complete selected-field typed tuples must differ. Missing and present-empty are distinct. Allow-many is a canonical set of complete tuples, never a cross-product. No fields means `EMPTY_SELECTED_FIELDS`; collision means `AMBIGUOUS_SCOPE`; suggestions may identify candidate-neutral differing fields but cannot auto-select them. Custom expectations use the same closed typed registry, require a separate review record, and must not duplicate a confirmed observed selected-field tuple.
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
- pure session finalization covers allow-one, allow-many, custom, reject-all, defer, refine, early reveal, and post-reveal-change semantics exactly; durable promotion accepts the four non-`REFINE` actions and must refuse `REFINE` without publication;
- weak fields return `AMBIGUOUS_SCOPE` with no compilable artifact; allow-many never generates a field cross-product; no unselected field enters the predicate view;
- `REJECT_ALL`, `DEFER`, and `REFINE` cannot be converted to a compilable ruling even through direct application-service calls;
- unknown didrun strings round-trip byte-for-byte and missing receipt remains `UNRECEIPTED`.

Kill exactly these required mutants: publish before CAS compare; last-writer-wins head update; mutable object overwrite; nonce-only freshness; discovery-evidence reuse; partition-shape rather than exact labeled-map confirmation; stale head digest accepted; candidate identity placed in the blind DTO; blind cards ordered by support; default-first selected field; allow-many cross-product; missing collapsed into present-empty; `REJECT_ALL` cast to compilable; receipt-grade normalization; alias bound to the first supporting member; run challenge omitted from confirmation nonce; and zero confirmation schedule offset. Do not delete or rename a mutation because it survives.

## didrun unit boundary

Wrap every load-bearing Go test, race test, property/model suite, mutation runner, lint/vet command, and fresh-confirmation fixture in `didrun run --`. Use `didrun show --session` to capture event indexes. Declare narrow event-bound claims with relevant paths and exact language; never claim confidentiality, hostile-code safety, comprehension, bias reduction, Linux, Node, or UI behavior here.

After review, stage only this prompt's files, commit the verified U6 store/Choicepoint unit without an AI co-author trailer, then run `didrun seal --commit HEAD` followed by `NO_COLOR=1 didrun verify --strict`. Nonzero means the unit is unfinished. Fix the underlying issue, rerun affected verification through didrun, add claims, create a correcting commit, reseal, and repeat until strict exits 0. Prior failed receipts remain history; never weaken tests/claims or relabel scope to pass.

## Handoff and honest fallback

Update the Mode C handoff with commit, exact grades, event indexes, strict verdict, store format/version, the fixed linear head stages, freshness proof, blind-kernel omissions, caller-attribution/authenticity nonclaim, receipt behavior, resource ceilings, permanent failed events, and P07 as next. Anything not run through didrun is `UNRECEIPTED`.

Stop before contract emission if immutable CAS lineage is not construction-safe, physical confirmation can be forged from old evidence, exact map identity is lost, blind DTOs leak identities/support, or selected fields can accept a confirmed disallowed outcome. The honest fallback is an observation/reduction artifact store with no decision-ready or executable-residue claim.
