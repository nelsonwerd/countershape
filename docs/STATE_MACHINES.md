# Countershape state machines

- **Contract version:** U0 / `state-machines-v1`
- **Target:** narrowed Darwin reference instrument
- **Status:** normative separation of implemented and future transitions. U6a is sealed through `RULING`; P07A changes authority inside that transition; P07B-A1 and P07B-A2.1 are sealed substrate; accepted P07B-A2.2 seals pure compiler output; sealed P07B-B source implements terminal publication plus optional retryable native materialization; execution transitions remain future.

Countershape state is a set of immutable semantic artifacts connected by validated transitions. State names are not presentation copy. The Go domain model, JSON schemas, API DTOs, CLI, studio, generated residue, examples, and tests must agree on these names and preconditions.

## Global transition rules

Every semantic transition follows the same rules:

1. **Validate from immutable inputs.** The transition names every input artifact by domain-separated digest and validates its schema, kind, lineage, and construction invariants.
2. **Bind the current head.** If a mutable study head is involved, the request supplies the expected current digest. A stale compare-and-swap fails without creating the requested semantic object.
3. **Create, never rewrite.** A successful transition writes a new immutable object. It does not edit an ancestor or replace bytes behind a digest.
4. **Fail without partial semantic publication.** A validation or pre-transition emission failure may retain a typed nonadvancing failure artifact. It cannot publish a partially valid successor or a partial final contract directory. After a terminal bundle residue is durable, a separate materialization failure reports `RESIDUE_PERSISTED_EXPORT_INCOMPLETE` and remains retryable from that residue; it does not pretend the semantic transition rolled back.
5. **Advance only through a legal edge.** Presentation code cannot infer, skip, or manufacture an edge. The application service revalidates every browser/CLI request.
6. **Do not edit an ancestor into freshness.** The full product target models semantic changes through successor studies and derived stale/invalidation status. U6 implements no such status graph: a changed ancestor cannot advance the existing fixed head and must use a distinct study or a future successor service.
7. **Keep historical truth.** Stale, invalidated, cancelled, partial, deferred, and contradicted objects remain addressable and retain their original meaning.
8. **Do not upgrade external evidence.** didrun grades are opaque references. They do not cause a Countershape state transition.

`CANCELLED` and `PARTIAL` are honest nonadvancing dispositions. They retain completed evidence but never satisfy a missing precondition. A new run allocates fresh attempts; it does not resume a candidate process.

## Full-product immutable artifact lineage target

The full-product happy-path semantic lineage is shown below. Historical U6a could stop at a legacy whole-projection `RULING`. Sealed P07A/U6b adds portable selected-field ruling authority without a new head stage. Accepted P07B-A2.2 compiles the recoverable bundle in memory; sealed P07B-B source adds the terminal `CONTRACT_BUNDLE` residue and optional physical materialization. Later `CONTRACT_EXECUTION` is immutable nonhead evidence, not another lineage stage.

```text
SOURCE_SPEC
  -> COMPILED_PLAN
  -> MATERIALIZED_CANDIDATE_SET
  -> STRUCTURAL_WORLD_INSTANCES
  -> INSTANCE_MEASUREMENTS
  -> COMPARISON_ADMISSION | REJECTED_COMPARISON
  -> BASELINE_OBSERVATION
  -> DIVERGENCE
  -> REDUCTION_RESULT
  -> FRESH_CONFIRMATION
  -> CHOICEPOINT_READY
  -> RULING
  -> CONTRACT_BUNDLE

CONTRACT_BUNDLE
  -> CONTRACT_EXECUTION_TARGET (immutable nonhead)
  -> FINALIZED_CONTRACT_RUN (immutable nonhead)
  -> CONTRACT_EXECUTION (immutable nonhead)
```

Each name denotes a distinct immutable object kind, not a mutable phase field.

| From | To | Legal preconditions | Refusal/nonadvancing result |
| --- | --- | --- | --- |
| `SOURCE_SPEC` | `COMPILED_PLAN` | inert spec parses strictly; every default/budget is materialized; adapter and projection profiles are supported | typed spec refusal |
| `COMPILED_PLAN` | `MATERIALIZED_CANDIDATE_SET` | 2–4 refs pin to supported immutable trees; every required blob is verified; no candidate slot ambiguity | typed source/materialization refusal |
| `MATERIALIZED_CANDIDATE_SET` | `BASELINE_OBSERVATION` | opaque bindings plus matching plans allocate structural worlds; complete measurement matrices are admitted; peer candidate batches share the exact per-repetition admission set; all required batches reach an honest classification | rejected comparison, partial/control artifacts; no token, batch, or baseline as applicable |
| `BASELINE_OBSERVATION` | `DIVERGENCE` | at least two eligible candidates are `OBSERVED_STABLE(k/k,h)` and at least two fingerprints exist | no-divergence disposition |
| `DIVERGENCE` | `REDUCTION_RESULT` | typed reducer has a decreasing measure; every accepted proposal passes full-map comparability and preserves the exact eligible labeled map; U5 grade construction succeeds | cancelled/partial/best available grade according to rules |
| `REDUCTION_RESULT` | `FRESH_CONFIRMATION` | a crypto-random run challenge; challenge-and-ordinal-bound nonces; phase offset `1`; new attempts, worlds, process/root/invocation facts; and no prior-evidence reuse reproduce the exact comparable disposition | confirmation refusal |
| `FRESH_CONFIRMATION` | `CHOICEPOINT_READY` | exact eligible candidate-to-fingerprint map and exact exclusion key/classification map are preserved; same freshly executed exclusion is allowed, changed disposition is not; all semantic ancestors are current | stale/ineligible Choicepoint draft |
| `CHOICEPOINT_READY` | `RULING` | the local decision session completes against the exact current digest; action and selected/context fields are valid; caller attribution is recorded without human-authenticity proof | typed decision or durable-promotion refusal |
| `RULING` | `CONTRACT_BUNDLE` | current ruling uses the P07A adapter-bound portable mode; action is compilable; selected fields are nonempty and separate allowed/disallowed tuples; an exact PortableSource reconstructs the same plan/projection/minimized stimulus; the emitter creates one recoverable six-file body; the complete expected head token remains current | legacy/noncompilable/ambiguous/source/profile/stale refusal; no residue and no final directory |
| `CONTRACT_BUNDLE` | immutable nonhead `CONTRACT_EXECUTION_TARGET` | exact bundle/source reopen; one explicit Git ref is pinned, inspected, privately materialized, and reopened; one Node runtime is measured/revalidated; one fresh conformance marker is durable; target publishes and reopens before spawn | any authority/materialization/runtime/freshness failure refuses target publication; study head remains the same bundle residue |
| `CONTRACT_EXECUTION_TARGET` | immutable nonhead `FINALIZED_CONTRACT_RUN` | the C4 runner acquires the private boot-session interlock from opaque official-target authority, wins StartClaim/RunPermit once, records conclusive SpawnObservation, makes the full owner witness durable, and only then releases the interlock; process control, projection when present, teardown/drains/orphan checks, revalidation, and all five standalone domains close against the same target/attempt | no permit, missing observation, target/attempt mismatch, or incomplete terminal evidence refuses finalization; ambiguity consumes the target and holds the interlock on that boot; study head remains the same bundle residue |
| `FINALIZED_CONTRACT_RUN` | immutable nonhead `CONTRACT_EXECUTION` | exact official target/run reopen under one literal classifier profile; clean projected process closure plus `COMPLETE` unviolated scope derives conformance/contradiction, while either ineligible axis derives the ineligible class | target/run/profile or derived-result mismatch refuses classification publication; retry performs no spawn; study head remains the same bundle residue |

`REJECT_ALL` and `DEFER` may create a final noncompilable `RULING`/`DecisionRecord`, but neither has an edge to `CONTRACT_BUNDLE`; U6 does not reopen a deferred head. A semantic `REFINE` DecisionRecord may be constructed, but durable promotion returns `REFINE_REQUIRES_SUCCESSOR_STUDY` before object publication or head mutation.

## Future semantic ancestry and staleness target

The following changes are semantic and always create a successor lineage:

- candidate commit/tree or candidate execution key;
- materialization policy or verified manifest;
- `WorldPlan`, adapter, runner, comparison envelope policy, or budgets;
- original or reduced stimulus;
- capture policy, projection definition, or canonical profile;
- repeat policy, exact outcome map, eligibility, or exclusions;
- reducer set, well-founded measure, accepted path, or final sweep;
- fresh-confirmation evidence;
- selected fields, allowed/custom tuples, or exact witness scope; and
- contract emitter or predicate profile.

A display-only change—such as ref spelling, caller annotation, or nonsemantic layout preference—must be stored outside identity-bearing objects or explicitly modeled so it cannot silently alter semantic digests.

Staleness rules:

```text
CURRENT ancestor + semantic change
  -> new ancestor revision
  -> old descendants remain immutable
  -> old descendants receive/derive STALE in the new lineage
  -> no STALE object can be made current again
```

To continue, reconstruct the necessary descendants from the new ancestor. Revalidation cannot flip an old byte-identical object from `STALE` to fresh; it creates a new artifact linked to new evidence.

`INVALIDATED(reason,replacement?)` is a future explicit terminal disposition for a known defect, policy withdrawal, or supersession. U6 implements neither this object nor successor lineage construction.

## Store-head state machine

CAS heads are mutable selectors, not evidence and not a history log.

```text
ABSENT --create(expected=ABSENT,new=A)--> HEAD(A)
HEAD(A) --advance(expected=A,new=B)----> HEAD(B)
HEAD(A) --advance(expected=X,new=B)----> CAS_CONFLICT   when X != A
```

Rules:

- `next` is a valid in-memory `SemanticObject` of the exact kind permitted at the next fixed revision.
- The complete opaque expected token binds store instance, study, revision, stage, current kind/digest, immediate previous object, lineage root, and head digest; it must match the reopened current head.
- CAS comparison and legal-edge validation occur before successor object or temporary-file creation.
- `CAS_CONFLICT` and `ILLEGAL_LINEAGE` publish no successor object and do not advance the head.
- A successful advance publishes/reopens the object, atomically replaces `head.json`, and reopens both head and object.
- Only the current head body is retained. Previous head/object digests are identity commitments, not a persisted navigable history.
- Deletion, rewind, branch, successor-study, invalidation, administrative repair, and progress SSE are outside U6.

## Execution-attempt state machine

Every evidentiary discovery, reduction proposal, final-sweep neighbor, confirmation trial, and current conformance run creates a physically new attempt.

```text
ALLOCATED
  -> MATERIALIZING
  -> STARTING
  -> READY
  -> PROBING
  -> CAPTURING
  -> TEARING_DOWN
  -> FINALIZED
```

The phases have these entry and exit contracts:

| Phase | Entry fact | Success edge | Detected failure handling |
| --- | --- | --- | --- |
| `ALLOCATED` | attempt artifact exists before spawn; nonce and owned roots are new | private roots/resources created | set pending allocation/control reason; finalize without behavior |
| `MATERIALIZING` | exact pinned candidate and empty owned root | supported blobs verified and written | set `MATERIALIZATION_ERROR`; route through cleanup |
| `STARTING` | direct argv, sparse environment, cwd, and budgets fixed | direct child/process group created | set `START_ERROR`; route through cleanup |
| `READY` | CLI one-shot is ready immediately, or fixture-owned HTTP readiness evidence is present | probe may begin | set `READINESS_ERROR`; route through cleanup |
| `PROBING` | exact typed stimulus bound to attempt | domain capture completes within caps | set transport/timeout/cancel/output control; route through cleanup |
| `CAPTURING` | channels and control metadata are bounded | immutable captured artifact prepared | set capture/projection-precondition control; route through cleanup |
| `TEARING_DOWN` | process/resources may exist; pending control retained | spend one bounded cooperative-exit interval; resolve the pre-TERM owned-group state, retrying only `(present=true, EPERM)` within its bounded sub-budget; skip TERM on absence, signal only after clean presence, and retain persistent/other uncertainty; after TERM use bounded grace/KILL only from clean continued presence; finally require child wait, pipe drains, and observed group absence | set/augment `TEARDOWN_ERROR` or `ORPHAN_RISK` |
| `FINALIZED` | immutable attempt and teardown facts published | eligibility service may inspect | terminal |

For the U4 HTTP reference profile, readiness evidence is one dedicated inherited pipe carrying exactly byte `0x01` followed by EOF within the declared readiness budget (`ONE_BYTE_0X01_THEN_EOF_V1`). It is never an HTTP request. Empty EOF, a different or extra byte, missing EOF, timeout, cancellation, or service exit before the complete signal cannot advance `STARTING -> READY`.

When multiple readiness-terminal facts are owner-visible together, U4 applies fixed precedence: output limit, cancellation, deadline, child exit, then readiness. An owned clean cooperative exit still closes the pre-TERM probe result as `ABSENT`, including when a transient positive-presence `EPERM` observation resolves to absence; `NOT_APPLICABLE` is not a clean owned-HTTP lifecycle result.

Any detected failure routes to `TEARING_DOWN` when owned resources may exist. It does not jump directly to behavior classification. A service that independently exits nonzero or by signal after writing a response is a transport control; only owner-issued TERM/KILL during cleanup is excluded from that rule. If teardown itself fails, the teardown/orphan fact is retained even when an earlier control reason already exists.

An attempt is **eligible for projection** only when all are true:

```text
phase == FINALIZED
trial_control == BEHAVIOR_CAPTURED
required_capture == COMPLETE
teardown == CLEAN
admission_token == PRESENT_FROM_ASSESS_COMPARISON
```

Every other finalized attempt is ineligible. `FINALIZED` means the attempt record is complete; it does not mean the candidate passed.

Illegal attempt transitions include starting before materialization verification, probing before ready, projecting before finalization, skipping teardown, converting a cap/timeout into application output, and reusing a finalized attempt in another evidentiary batch.

## Stable-batch and candidate-eligibility state machine

For one candidate binding/key, plan, stimulus, envelope, stimulus-independent comparison basis, exact admission roster, ordered per-repetition admission-digest set, repeat policy, purpose, and schedule:

```text
BATCH_CREATED
  -> BATCH_SCHEDULED
  -> BATCH_RUNNING
  -> BATCH_CLASSIFIED
```

The batch schedules exactly `k` required new trials within declared budgets. Classification is construction-safe and mutually exclusive:

1. `UNCOMPARABLE(reasons)` if an already-admitted required trial has a detected source/control/projection result that cannot contribute. Envelope/matrix rejection happened earlier and cannot create a batch.
2. `UNSTABLE(histogram)` as soon as at least two eligible fingerprints have occurred and no ineligible control has taken precedence. Every trial actually run remains in the histogram; the batch need not spend the remaining repeat budget merely to establish finite disagreement.
3. `INCOMPLETE` if the budget/cancellation ends before `k` eligible trials, only one eligible fingerprint has occurred, and no stronger explicit source/envelope/control refusal already determines `UNCOMPARABLE`.
4. `OBSERVED_STABLE(k/k,h)` if exactly `k` required new trials are eligible and every fingerprint is `h`.

Before `BATCH_CREATED`, `AssessComparison` must have persisted one admission matrix per repetition. A rejected matrix is terminal for that study path and produces no row token or batch. The classification records every tagged captured-or-controlled trial, so a UI cannot hide a control or incomplete trial behind an aggregate. Extra opportunistic trials do not repair a failed required batch; a new repeat policy creates a new batch.

Only `OBSERVED_STABLE(k/k,h)` candidates enter a `CandidateOutcomeMap`. The others remain explicitly excluded with their classification and reasons. Construction requires:

```text
eligible_candidate_count >= 2
entries union exclusions == exact expected candidate roster
one canonical sorted candidate_execution_key -> projection_fingerprint map
same ordered comparison_admission_digest set in every candidate batch
globally unique world_instance_digest and attempt_artifact_digest evidence
plan + comparison_basis + phase bound explicitly
```

A nondivergent neighbor may still construct a `CandidateOutcomeMap` so the comparability-first reduction operation can classify a collapsed split as `CHANGES`. Entering the baseline divergence/Choicepoint path requires the sealed `DivergentBaseline` refinement with `distinct_projection_fingerprint_count >= 2`. Rejecting a comparable nondivergent map at base construction would misclassify a real collapse as `UNRESOLVED`.

Candidate label/order/producer/support does not enter the map digest. Display groups are recomputed views and cannot cross back into semantic input.

## Projection state machine

Projection never repairs control evidence:

```text
ELIGIBLE_CAPTURE
  -> PROJECTION_VALIDATING
  -> PROJECTION_OK(canonical_bytes,fingerprint,source_links)
     or PROJECTION_REJECTED(reason)
```

`PROJECTION_REJECTED` makes the trial ineligible. It is not an output fingerprint. Each operation must be named, deterministic, and linked to captured input. Changing an operation or field registry creates a new `ProjectionDefinition`. In U6 the old fixed spine simply cannot advance with that changed ancestor; an independently named study is unlinked, while construction-safe successor lineage is a future service.

The only equality edge is exact canonical-byte/fingerprint equality under the same projection definition. No display formatting, locale order, ordinal group number, or browser serializer participates.

## Reduction-run state machine

The run-level phases are:

```text
REDUCTION_CREATED
  -> BASELINE_BOUND
  -> SEARCHING
  -> FINAL_SWEEP
  -> GRADED
```

`BASELINE_BOUND` fixes the full typed baseline `CandidateOutcomeMap`, original stimulus, plan, envelope, stimulus-independent comparison basis, reducer-set digest, strictly decreasing measure, budgets, exact candidate roster, eligible set, and exclusions. It retains the baseline preservation digest as a derived value, not a standalone predicate. Concrete admission matrices and purpose change for each newly measured neighbor; requiring their artifact digests to equal the baseline would be a category error.

Each proposed neighbor has its own state:

```text
PROPOSED
  -> VALIDATED_SMALLER
  -> FRESH_BATCH_RUNNING
  -> PRESERVES | CHANGES | UNRESOLVED
```

An invalid or nondecreasing proposal is recorded as `REJECTED_NEIGHBOR` and never executes. After execution, compare full typed maps first. A plan, envelope, comparison-basis, roster, eligibility, or exclusion mismatch is `UNRESOLVED`; comparable maps with equal exact eligible labeled-map digests are `PRESERVES`, and comparable maps with different digests are `CHANGES`. A proposal is accepted into the path only from `PRESERVES`. `CHANGES` and `UNRESOLVED` are never interchangeable, and a naked digest never chooses among them.

Grade construction is exact:

| Grade | Construction requirements | What it does not claim |
| --- | --- | --- |
| `UNCHANGED` | no smaller proposal was accepted | that all neighbors were resolved or that the original is locally minimal |
| `BEST_KNOWN` | at least one smaller proposal was accepted, and budget ended, sweep was incomplete, or at least one direct neighbor is `UNRESOLVED` | local minimality |
| `ONE_MINIMAL_UNDER(reducer_set_digest)` | current witness is valid; final neighbor enumeration is complete and durable; every enumerated direct neighbor has a typed result with required evidence/content-digest nonoverlap; the enumerated set exactly equals the typed `CHANGES` set | physical freshness, global minimum, cause, or comprehension |

If no smaller proposal was accepted and the run is partial/unresolved, the truthful grade is `UNCHANGED` with transcript/budget limitations; it is not promoted to `BEST_KNOWN` merely to sound more informative.

The following edges to `ONE_MINIMAL_UNDER(...)` are illegal: from `SEARCHING`; from an incomplete/cancelled sweep; with an exhausted budget before durable completion; with any `UNRESOLVED` neighbor; with a stale baseline; with changed candidate eligibility; with an execution reused from another evaluation; when only display-group shape or a naked digest was compared; or when concrete admission matrices were incorrectly required to match across different stimuli/phases.

U1 can model logical proposals, evaluations, and sweep completeness, but cannot construct this proof-bearing grade. U5 adds the durable store and the only constructor that can consume its completion authority. A caller-authored boolean, digest, or list is never a substitute for that authority.

## Fresh-confirmation state machine

```text
CONFIRMATION_NOT_STARTED
  -> CONFIRMATION_RUNNING
  -> CONFIRMED_EXACT_MAP | CONFIRMATION_REFUSED(reason)
```

Entry requires a U5-graded reduction result. Each confirmation run receives a crypto-random challenge; each instance nonce binds that challenge and its schedule ordinal. The schedule is phase-bound and begins at offset `1` from discovery. Every trial references an attempt artifact created before spawn, new private roots, a new process lifecycle, fixture-observed invocation evidence outside the selected predicate, and evidence identities absent from discovery/reduction. The minimized `CandidateOutcomeMap` is reconstructed from the new batches and must pass full-map comparability with the reduced baseline before matching the exact eligible labeled-map digest. Exact exclusion keys/classifications are part of comparability, but their fresh batch/evidence identities are expected to differ: the same candidate may, for example, remain freshly `UNSTABLE`. A changed eligible/excluded disposition or exclusion classification refuses readiness.

`CONFIRMATION_REFUSED` is terminal for that run. A retry allocates an entirely new confirmation run and attempts; U6 creates no successor-study/status object for the refusal. A new nonce attached to copied capture, projection, or process evidence is an illegal transition.

Only `CONFIRMED_EXACT_MAP` can construct `CHOICEPOINT_READY`.

## Sealed U6 Choicepoint/ruling spine

```text
CONFIRMATION
  -> CHOICEPOINT_READY
  -> RULING
```

`CHOICEPOINT_READY` is a store-head stage and live `promotion.Ready` capability, not a mutable field inside `ChoicepointRecord`.

- `ALLOW_OBSERVED`, `CUSTOM_EXPECTATION`, `REJECT_ALL`, and `DEFER` can produce strict DecisionRecords and may advance the fixed head to `RULING`.
- Only the first two may yield `CompilableRuling`.
- `DEFER` records a noncompilable decision; U6 does not reopen the same head into a new session.
- `REFINE` remains a valid noncompilable semantic decision but cannot advance the U6 head; promotion returns `REFINE_REQUIRES_SUCCESSOR_STUDY` without publication.
- U6 has no durable `DISCOVERED`, `RESOLVED`, `DEFERRED`, `STALE`, or `INVALIDATED` state object.

## Future product lifecycle target

`DISCOVERED`, derived `RESOLVED`/`DEFERRED` presentation status, deferred-session reopening, successor studies, and explicit `STALE`/`INVALIDATED` status remain future application-service work. Those states must not be inferred from UI flags or added to Choicepoint canonical bytes. If implemented, they require new immutable objects and construction-safe transitions with their own receipts.

## Blind-first decision-session state machine

The interaction state is separate from Choicepoint truth:

```text
BLIND_OPEN
  -> PROVISIONAL_RECORDED
  -> REVEALED
  -> POST_REVEAL_RECORDED
  -> FINALIZED
```

U6's closed blind semantic kernel carries scenario, exact scope, original and minimized stimulus bytes, bounded reduction facts, configured repeat counts, projection operations, deterministic cards, and trust warnings. It structurally omits system-supplied candidate/ref/producer identity, total candidate count, per-outcome support, source order, and reveal material. Behavior and caller-authored scenario text may themselves reveal provenance; DOM/accessibility omission and the complete comparison-envelope/security/derivation presentation remain U8 work.

The semantic session records visits to exact named surfaces rather than claiming that it can inspect a future UI. `ORIGINAL_WITNESS`, `MINIMIZED_WITNESS`, `REDUCTION_DERIVATION`, `PROJECTION_OPERATIONS`, and `NONASSERTED_FIELDS` must be recorded before a standard-path proposal. `PROVENANCE_REVEAL` becomes visitable only after reveal.

`PROVISIONAL_RECORDED` records an action, complete tuple choice or custom tuple as applicable, an explicit disposition for every differing field, and the exact preview. It is not a ruling.

`REVEALED` is mandatory before finalization. A subsequent `POST_REVEAL_RECORDED` state preserves both provisional and final values; a changed draft requires a rationale.

An early reveal is legal but explicit:

```text
BLIND_OPEN -> REVEALED -> POST_REVEAL_RECORDED -> FINALIZED
```

This path records that no pre-reveal provisional ruling exists. It cannot be presented as completion of the standard blind-first sequence. The exact required surfaces on both paths are `ORIGINAL_WITNESS`, `MINIMIZED_WITNESS`, `REDUCTION_DERIVATION`, `PROJECTION_OPERATIONS`, `NONASSERTED_FIELDS`, and `PROVENANCE_REVEAL`. In the standard path the first five must be visited before the provisional ruling; both paths require all six before finalization.

View/visit facts prove presentation only. They are not comprehension state.

### Final action validation

| Action | U6 semantic result | Durable U6 behavior | Compilable? |
| --- | --- | --- | ---: |
| `ALLOW_OBSERVED` | exact observed tuple-set DecisionRecord | may advance current head to `RULING` | yes |
| `CUSTOM_EXPECTATION` | one reviewed tuple distinct from every confirmed selected tuple | may advance current head to `RULING` | yes |
| `REJECT_ALL` | dissatisfaction record without positive oracle | may advance to noncompilable `RULING` | no |
| `DEFER` | noncompilable defer record | may advance to `RULING`; reopening is future work | no |
| `REFINE` | semantic successor-study request record | promotion returns `REFINE_REQUIRES_SUCCESSOR_STUDY`; no object/head mutation | no |

If selected fields fail the separation formula, finalization returns `AMBIGUOUS_SCOPE`; no DecisionRecord with compile eligibility and no files are created.

The table's “yes” is semantic compile eligibility, not standalone portability. Sealed P07A fresh Choicepoints now select adapter-bound fields, while historical U6 whole-projection rulings remain valid, addressable, and parseable but receive `LEGACY_WHOLE_PROJECTION_NOT_PORTABLE` from portable preparation. P07B must preserve that refusal even when a legacy DecisionRecord carries the derived `compilable` fact.

## Implemented P07A/U6b portable-ruling state machine

P07A changes the construction authority inside `CHOICEPOINT_READY -> RULING`; it adds no head stage:

```text
EXACT_CONFIRMED_PROJECTION_PROOFS
  -> PROJECTION_BINDING_RESOLVED
  -> HISTORICAL_ADAPTER_WIRES_STRICTLY_TRANSLATED
  -> PORTABLE_COMPLETE_TUPLES_SEALED
  -> PORTABLE_CHOICEPOINT_READY
  -> SELECTED_FIELD_RULING
```

Alternative results are exact nonadvancing refusals:

```text
PROJECTION_PROFILE_UNRESOLVED
PROJECTION_WIRE_NONEXACT
CONFIRMED_PROOF_MISMATCH
EMPTY_SELECTED_FIELDS
AMBIGUOUS_SCOPE
CUSTOM_EXPECTATION_ALREADY_OBSERVED
```

Legacy whole-projection Choicepoints remain valid and addressable and may retain historical rulings. They do not enter the portable sequence; the current preparation seam returns `LEGACY_WHOLE_PROJECTION_NOT_PORTABLE`, and P07B must preserve that refusal before any compiler or file construction.

## Implemented P07B-A1 source/process substrate

A1 adds no study-head stage and performs no `RULING -> CONTRACT_BUNDLE` transition. Its two closed, independently useful paths are:

```text
EXACT_SUPPORTED_SOURCE_INPUTS
  -> CLOSED_ADAPTER_AND_RUNNER_AUTHORITIES_RECONSTRUCTED
  -> PORTABLE_SOURCE_CANONICALIZED
  -> PORTABLE_SOURCE_REPARSED_AND_BYTE_MATCHED
```

and, for the portable HTTP fixture:

```text
CHILD_PROCESS_STARTED_WITH_FD3
  -> EXACT_READY_FRAME_AND_EOF_ACCEPTED
  -> CHILD_REPORTED_LOOPBACK_ENDPOINT_USED
  -> RAW_HTTP_EXCHANGE
  -> OBSERVATION_FINALIZED
```

The first path establishes self-consistency over supplied A1 inputs only. The second reaches `Observation` only. Neither path opens a current ruling, creates a FreshConfirmation, creates a portable Choicepoint/ruling under the child-bind lineage, generates a file, publishes an object, or advances a head. The reported port is not kernel listener-ownership evidence; arbitrary trusted code can report a decoy local service.

## P07B/U6c contract-emission state machine (A2.2 sealed; B qualification pending)

```text
DECISION_FINALIZED
  -> CURRENT_PORTABLE_RULING_REOPENED
  -> A1_PORTABLE_SOURCE_JOINED_TO_CURRENT_RULING_CONFIRMATION_AND_PROOFS
  -> PURE_SIX_FILE_BUNDLE_COMPILED_IN_MEMORY
  -> EXPECTED_RULING_TOKEN_REVALIDATED
  -> CONTRACT_BUNDLE_OBJECT_PUBLISHED_AND_REOPENED
  -> TERMINAL_RESIDUE_HEAD_PUBLISHED
  -> MATERIALIZATION_PENDING
  -> MATERIALIZED | RESIDUE_PERSISTED_EXPORT_INCOMPLETE
```

Alternative terminal results from validation are:

```text
NONCOMPILABLE_ACTION(action)
AMBIGUOUS_SCOPE
LEGACY_WHOLE_PROJECTION_NOT_PORTABLE
PORTABLE_SOURCE_REQUIRED
PORTABLE_SOURCE_MISMATCH
NONPORTABLE_START_PROFILE
STALE_CHOICEPOINT
```

The emitter accepts a construction-safe current portable ruling plus an opaque exact source-reconstruction witness, not a generic DecisionRecord plus a Boolean or caller-paired digests. It creates all six path/mode/content triples in memory. The ContractBundle embeds every exact file byte; `manifest.json` covers the other five and excludes itself. The store revalidates the complete expected head before object/temp creation, durably publishes and reopens the bundle, and only then advances to terminal residue.

A stale loser creates no object, temporary file, or final output. A crash after object publication but before head publication may leave an unreachable immutable object; a head cannot name a missing object. Final filesystem materialization begins only after residue and uses a private same-parent temporary directory plus create-new rename. An existing exact directory is idempotent; a partial/mismatched directory is never overwritten. Post-residue failure leaves recoverable semantic authority and an absent or refused final destination.

Go/Node parity disagreement blocks the external P07B ship-and-claim gate, not this semantic transition. didrun grades remain opaque external evidence and never authorize residue publication. A qualifying parity receipt maps to a claim without editing the bundle or study head; an absent receipt leaves the capability `UNRECEIPTED` rather than inventing a runtime transition result.

## Corrected P07B/U6c contract-execution state machine

C1 implements only the inert canonical object algebra at the right side of this future physical sequence. It can build and strictly reopen exact target, finalized-run, and execution bodies from already-held semantic inputs; it owns no transition, persistence, filesystem, runtime, spawn, or publication authority. The checked-in schemas are syntax projections rather than alternate state constructors. In particular, a schema-valid body may still fail Go-owned process/spawn/capture, evidence-role, scope-order, aggregate, or private-manifest correlations.

Historical evidence and later exact-target conformance are never merged:

```text
EXPLICIT_TARGET_REF_PINNED
  -> TARGET_INSPECTED
  -> TARGET_PRIVATELY_MATERIALIZED_AND_REOPENED
  -> NODE_RUNTIME_ADMITTED_AND_REVALIDATED
  -> DARWIN_BOOT_SESSION_ADMITTED
  -> CONFORMANCE_ATTEMPT_MARKER_DURABLE
  -> CONTRACT_EXECUTION_TARGET_PUBLISHED_AND_REOPENED
  -> EXECUTION_INTERLOCK_HELD_FOR_TARGET_AND_BOOT_SESSION
  -> START_INTENT_CLAIMED_AND_RUN_PERMIT_MINTED
  -> SPAWN_ATTEMPTED
  -> START_ERROR_OBSERVED | CHILD_PID_OBSERVED | SPAWN_OUTCOME_UNKNOWN
  -> PROCESS_AND_STANDALONE_AXES_CLOSED
  -> TERMINAL_WITNESS_DURABLE
  -> EXECUTION_INTERLOCK_RELEASED
  -> FINALIZED_CONTRACT_RUN_PUBLISHED_AND_REOPENED
  -> CONTRACT_EXECUTION_CLASSIFIED_AND_REOPENED

SPAWN_OUTCOME_UNKNOWN
  -> TARGET_PERMANENTLY_CONSUMED_AND_INTERLOCK_HELD
  -> NO_FCR_OR_CLASSIFICATION

PROCESS_CLEAN + PROJECTED(tuple) + STANDALONE_COMPLETE
  -> ELIGIBLE_CLEAN -> CONFORMS | CONTRADICTS

PROCESS_CONTROL(primary?, teardown_error?, orphan_risk?) and/or STANDALONE_PARTIAL | STANDALONE_VIOLATED
  -> INELIGIBLE_CONTROL | INELIGIBLE_STANDALONE | INELIGIBLE_CONTROL_AND_STANDALONE
  -> INELIGIBLE_EXECUTION
```

`ContractExecutionTarget` is a nonhead pre-spawn object, not proof that a process ran. It is constructed only from the reopened bundle/residue, opaque live `InspectedTree` plus its exact verified materialization, owned admitted Node capability, and fresh durable attempt allocation. C1's stored path, byte digest, mode/count, and measured `process.execPath` identify a checkpointed runtime tuple; they do not prove which filesystem object a later spawn consumed. It is official only through its typed store-private publication witness and exact link. It never routes through the comparison-only `SelectedTreeSet -> WorldPlan -> CandidateExecutionBinding -> WorldInstance` chain. A parsed target, copied OIDs/digests/runtime strings, dirty working-tree bytes, a generic CAS object, or a fake one-member candidate set cannot create execution authority.

One store-private `ExecutionInterlock` is the sole mutable operational exception to the nonhead semantic model. It serializes subject spawn for a measured boot session but cannot author, discover, or select a run/result. Only the C4 contract runner may acquire it from an opaque C3-issued official-target capability. The runner then creates the target-keyed durable `StartClaim`; only the combined durable winner receives one opaque, process-local, single-consumption `RunPermit`. A generic read/publish, C2 test fixture, losing creator, copied owner epoch, lease expiry, or restart never receives another. `SpawnObservation` separately records `START_ERROR` or an observed child PID. C1 closes and round-trips the three start-error code shapes only; C4 must prove their evidence content and chronology. Missing observation is `UNKNOWN`, never `NOT_STARTED`, and cannot enter an FCR. A conclusive terminal witness becomes durable before the interlock releases. Any ambiguity permanently consumes that target and leaves the interlock held on the same boot; a fresh target/attempt is not sufficient retry authority. Reset requires explicit operator action plus a measured boot-session identity demonstrably distinct from the held identity; otherwise no further subject spawn is admitted. This is serialized cooperative at-most-once spawn admission, not exactly-once execution or process-resume authority.

`FinalizedContractRun` is the nonhead proof boundary for one physical run, not predicate classification. It binds the exact target, attempt, start intent, and owner-produced terminal facts to a bounded canonical `ClosedRunWitness`. One process-control owner preserves at most one primary cause plus independent teardown/orphan controls; cleanup findings cannot overwrite the primary cause, and any retained control makes that axis ineligible. Standalone scope independently closes exactly five domains as `COMPLETE`, `PARTIAL`, or `VIOLATED`; partial/violated scope is ineligible even when a projected tuple exists and is never smuggled into the process `ControlReason` roster. A run from another target/attempt cannot be paired into classification even when tuple bytes match. Raw/large evidence remains private and bounded; default output never implies confidentiality.

- `CONFORMS`: clean projected selected-field tuple under standalone `COMPLETE` is a member of the allowed complete-tuple set.
- `CONTRADICTS`: clean projected selected-field tuple under standalone `COMPLETE` is not a member.
- `INELIGIBLE_EXECUTION`: process control and/or partial/violated standalone scope prevented an eligible conclusion; exact reasons remain only in the finalized run.

Both contradiction and ineligibility produce a nonzero ordinary direct Node test, but TAP diagnostics and machine codes remain distinct. Direct generated execution remains a separately exercised operator/black-box path and never becomes Go target/run/classification authority. `ContractExecution` binds the exact official target/run plus literal classifier profile and persists only the derived class. A classification-publication retry performs no spawn. A target execution never freshens, modifies, or reclassifies the historical Choicepoint, even when its pinned tree equals a historical candidate tree. Each target/run/classification triple is separate immutable nonhead evidence; package topology and before/after head evidence prove no head advance rather than an authored Boolean.

## Future product study-presentation lifecycle target

The following coarse application states are not U6 durable objects or head stages. U6 implements only the fixed typed head spine described above; it does not persist `COMPLETED`, `PARTIAL`, or `ERROR` study-status records and does not create successor runs.

Study presentation state is deliberately coarser than semantic artifact state:

```text
EMPTY
  -> PREPARING
  -> ACTIVE
  -> COMPLETED | PARTIAL | ERROR
```

| State | Meaning | Legal next action |
| --- | --- | --- |
| `EMPTY` | no compiled plan exists | author/compile source spec |
| `PREPARING` | plan/source validation and candidate preparation are in progress | enter `ACTIVE`, or finish `PARTIAL`/`ERROR` with evidence |
| `ACTIVE` | finite declared work is executing | finish `COMPLETED`, `PARTIAL`, or `ERROR` |
| `COMPLETED` | declared budgets ended and every item has an honest disposition | future application may create a construction-safe successor; no in-place resume |
| `PARTIAL` | cancellation/crash/budget left declared work without all dispositions | future application may inspect and create a fresh successor run |
| `ERROR` | a nonrecoverable study-level control prevented honest completion | future application may create a corrected successor spec/run |

`COMPLETED` never means verified software, all candidates agree, or every Choicepoint is resolved. It means the finite study ledger has no undisposed item.

A crash may leave finalized immutable semantic objects and nonadvancing partial-attempt evidence, but U6 does not publish a `PARTIAL` head or study-status object. It may not resume a candidate process or treat pre-crash executed observations as fresh confirmation. A future successor run may reuse pure canonical transformations of immutable bytes, but every evidentiary attempt must be new.

## Legal transition summary

| Transition | Why legal |
| --- | --- |
| eligible HTTP `500` -> projected value | transport/control succeeded; status is program behavior |
| candidate -> `UNSTABLE(histogram)` | all required trials eligible and at least two fingerprints observed |
| admitted candidate -> `UNCOMPARABLE(reasons)` | source/control/projection prevents its tagged trials from contributing |
| measurement matrix -> `RejectedComparison` | envelope assessment refused before tokens or batches existed |
| neighbor -> `UNRESOLVED` | no complete map exists or the full maps fail plan/basis/roster/eligibility/exclusion comparability |
| incomplete final sweep -> `BEST_KNOWN` | a smaller preserving witness exists, but local proof is incomplete |
| complete all-`CHANGES` sweep -> `ONE_MINIMAL_UNDER(R)` | construction invariant has all required durable evidence |
| confirmed Choicepoint -> noncompilable `DEFER` DecisionRecord | local caller deferred; U6 may advance to fixed `RULING`, but no source or reopening authority exists |
| semantic `REFINE` request -> nonadvancing promotion refusal | `REFINE_REQUIRES_SUCCESSOR_STUDY`; no object is published and the current Choicepoint/head remains unchanged |
| confirmed Choicepoint -> noncompilable `REJECT_ALL` DecisionRecord | local caller recorded dissatisfaction; U6 may advance to fixed `RULING`, but does not create a `RESOLVED` object or prove human authenticity |
| custom tuple matching no observed candidate -> bundle | separately authored positive oracle passes typing/separation |
| current eligible mismatch -> `CONTRADICTS` | harness ran correctly and selected tuple was outside allowed set |

## Illegal transition table

| Attempted transition | Required refusal | Reason |
| --- | --- | --- |
| moving display ref changes pinned candidate | keep original identity or create new lineage | ref spelling is not execution identity |
| `CandidateExecutionKey` or copied digests -> evidence allocation | typed binding refusal | allocation requires opaque `CandidateExecutionBinding` plus actual matching `WorldPlan` |
| copied target/tree/source/runtime fields, parsed `ContractExecutionTarget`, fake one-member candidate set, or historical `WorldInstance` -> official target/run | target-authority refusal | only the C3 live issuer may return official-target authority, and only the C4 runner may consume it |
| generic CAS read, C2 fixture, existing StartClaim, losing interlock/claimant, copied owner epoch, lease expiry, or restart -> RunPermit | `TARGET_START_ALREADY_CLAIMED` or interlock refusal; no permit | only the C4 combined interlock/claim durable winner may receive the one process-local permit |
| StartClaim -> child started/stopped/absent fact | preserve intent only; spawn outcome remains unknown without owner observation | store intent is not an OS-process transaction |
| ambiguous/crashed target -> FCR, interlock release, or any same-boot target spawn | refuse; keep interlock held | fresh target identity does not clear survivor uncertainty; only explicit operator reset plus a measured boot-session identity demonstrably distinct from the held identity can reopen admission |
| finalized run for target/attempt A -> classification for target/attempt B | `TARGET_RUN_MISMATCH`; no classification | lifecycle and observation authority cannot be caller-paired across physical attempts |
| standalone `PARTIAL` or `VIOLATED` plus projected tuple -> eligible conformance | `INELIGIBLE_EXECUTION` | missing or forbidden scope evidence cannot become a behavior conclusion |
| standalone-scope failure -> process `ControlReason` | preserve a separate scope axis | process control and absence measurement have different truth jurisdictions |
| unsupported Git entry omitted and remaining tree runs | `UNCOMPARABLE` | partial tree would misrepresent source |
| `STARTING` before verified materialization | attempt control failure | execution source is not established |
| retained projection before terminal closure -> tuple membership/classification | refuse until FCR closes clean process plus `COMPLETE` scope | useful projected bytes may be retained, but behavior classification cannot outrun lifecycle/scope evidence |
| timeout/output/teardown reason -> outcome fingerprint | reject construction | controls are not behavior values |
| `RejectedComparison` -> token/trial/batch/map | pre-batch study refusal | assessment did not admit the matrix |
| `UNSTABLE`, `UNCOMPARABLE`, or `INCOMPLETE` candidate -> eligible map entry | list as excluded | only `OBSERVED_STABLE(k/k,h)` enters the eligible labeled map |
| candidate batches with different admission-digest sets -> one map | reject construction | all candidates must share one exact matrix per repetition |
| duplicate world/attempt evidence identities across candidate batches -> one map | reject construction | outcome-map evidence identities are globally unique |
| ordinal group ID, membership shape, or naked preservation digest -> preservation | reject comparison | full-map comparability precedes exact labeled-map equality |
| boolean false substitutes for `CHANGES`/`UNRESOLVED` | reject schema/type | unresolved evidence blocks local grade |
| any unresolved neighbor -> `ONE_MINIMAL_UNDER(...)` | grade at most `BEST_KNOWN` when a smaller witness exists | final sweep is incomplete |
| copied capture + new nonce -> fresh confirmation | `CONFIRMATION_REFUSED` | metadata is not execution |
| changed plan/projection/candidate under old reduction | reject the old fixed-spine authority; use a distinct unlinked StudyID until a successor service exists | U6 publishes no `STALE` object and cannot advance changed ancestry |
| add a candidate under an existing `CandidateOutcomeMap` | new baseline lineage | exact candidate set is map identity |
| ruling before exact fresh confirmation | no `CHOICEPOINT_READY` authority | historical/discovery evidence is insufficient |
| finalization before required presentation/reveal | decision refusal | local decision protocol is incomplete |
| stale tab writes a ruling | `CAS_CONFLICT` | current lineage changed |
| empty or weak selected fields -> compile | `AMBIGUOUS_SCOPE`; no files | allowed/disallowed tuples are not separated |
| allow-many fields expanded independently | reject tuple construction | cross-product adds unauthored outcomes |
| context-only field emitted as assertion | emitter/vector failure | field was not selected |
| legacy whole-projection ruling -> portable preparation or later standalone compiler | `LEGACY_WHOLE_PROJECTION_NOT_PORTABLE`; no files | whole equality cannot authorize selected-field scope |
| caller-paired raw bytes/digests -> PortableSource | source reconstruction refusal | a digest-shaped pair is not a typed adapter preimage |
| inherited-listener HTTP ruling -> child-bind standalone source | `NONPORTABLE_START_PROFILE`; no files | the execution world changed and requires a new physical lineage |
| `REJECT_ALL`, `DEFER`, or `REFINE` -> emitter | `NONCOMPILABLE_ACTION`; no files | no positive exact oracle |
| partial six-file output -> publish | no residue and no final directory | bundle is one recoverable semantic body |
| post-residue materialization failure -> claim no durable change | `RESIDUE_PERSISTED_EXPORT_INCOMPLETE`; retry from bundle | source publication and filesystem export are separate truths |
| bundle digest -> `CONFORMS` without execution | no transition | source bytes are not current behavior |
| target conformance -> historical Choicepoint freshening or head advance | immutable nonhead ContractExecutionTarget, FinalizedContractRun, and ContractExecution only | truth jurisdictions differ |
| exact nonhead target/run/execution -> filesystem traversal, latest/current selector, or mutable status | reject field/API | P07B-C is exact-digest-only and does not add an execution head |
| stale CAS head -> last-write-wins | `CAS_CONFLICT` | concurrent/stale clients cannot overwrite intent |
| crash -> process resume/replay | no resume; any already-persisted nonadvancing evidence stays inert and a retry allocates a new world | persistence is not an execution runtime and U6 writes no crash-time status object |
| Wake session/effect/gate -> Countershape state | reject field/object | Wake is inert candidate provenance only |
| didrun grade -> Countershape classification | round-trip only | external receipt authority is separate |

<!-- countershape-validator: allow-prohibited-terms begin -->
Additional prohibited transitions include promoting an unqualified `STABLE` label, declaring “compatible worlds,” calling a reduced witness the “smallest behavior,” selecting a “winner” or “best branch,” publishing a “safely shareable” report, accepting symlink support, or treating execution-evidence cache/reuse as fresh evidence.
<!-- countershape-validator: allow-prohibited-terms end -->

## U0–U9 transition ownership

| Unit | State-machine authority added | Gate before next unit |
| --- | --- | --- |
| U0 | normative enums, legal/illegal edges, schemas, examples, planning validator | all controlling artifacts agree; negative planning mutations fail |
| U1 | canonical object construction, binding/structural-world/measurement/admission sums, logical map/reduction/decision invariants, model state tests; no grade constructor | required truth-kernel mutants die under the root-serialized didrun gate; until then U1 is `UNRECEIPTED`; no subprocess exists |
| U2 | attempt phases through `FINALIZED`, Git/materialization refusals, Darwin teardown controls | native hostile fixtures and freshness facts pass |
| U3 | CLI projection and stable-batch classifications | constant/alternating/control/incomplete fixtures classify exactly |
| U4 | HTTP phases/readiness/capture using the same eligibility service | no HTTP branch enters generic compare/reduce/choice truth |
| U5 | reduction proposal/run/final-sweep/grade transitions | shape trap, cancellation, budget edge, unresolved neighbor, both grades pass |
| U6a / P06 | CAS heads, physical confirmation, Choicepoint, blind semantic session, DecisionRecords, ruling promotion through `RULING` | stale/forged/reused-evidence/noncompilable/weak-field/blind-leak negatives pass; no public residue edge |
| U6b / P07A | adapter-bound portable tuple interpretation and selected-field rulings; no new head stage | unchanged historical bytes, exact profile/wire translation, selected-only custom expectation, legacy refusal, and physical CLI/HTTP studies pass |
| U6c / P07B-A1 | nontransition PortableSource reconstruction and portable child-bind HTTP Observation substrate | sealed source/process gates pass; no current-ruling join, compiler, residue, or execution state is inferred |
| U6c / P07B-A2.1 | nontransitioning current-ruling compilation preparation | source/ruling/Choicepoint/confirmation/proof equality and selected-tuple partition gates pass; no head advance or durable currentness after return |
| U6c / P07B-A2.2 | nontransitioning recoverable bundle compilation and strict Go/Node semantic evaluation | sealed parity, recovery, generated-runtime, and pure-boundary gates pass; no head advance |
| U6c / P07B-B (complete sealed boundary) | exact `RULING -> RESIDUE` terminal transition plus nonsemantic retryable physical materialization | final handoff, receipt reconciliation, and exact-boundary HTML are sealed and strict-clean; no C execution state is inferred |
| U6c / P07B-C (C0 sealed; C1 inert semantics current) | C1 strict nonhead target/run/classification algebra; later one private boot-session interlock, serialized intent-only at-most-once spawn admission, and separate process/scope axes | C1's own strict gate can establish inert semantics only; C2–C6 store/interlock, Git/runtime, lifecycle, five-domain scope, classification-publication, and cumulative gates remain separate, the interlock has no result authority, and study head remains terminal residue |
| U7 | complete study lifecycle and both decisive reference lineages | three clean runs preserve semantic bytes while all attempts are new |
| U8 | authenticated transport and full renderer state matrix | blind leakage, request forgery, presentation obligations, visual/accessibility gates pass |
| U9 | export/packaging/final claim mapping | exact environment receipts, final strict verify, HTML evidence, honest handoff |

No unit may manufacture a later state as fixture output and claim the transition implementation. U0 examples are inert contracts. U1 model tests do not establish subprocess behavior. U8 screenshots do not establish state-transition correctness without server/domain evidence.

## Model-testing obligations

At U1 and again when each edge gains I/O, model/state tests must generate legal and illegal command sequences and assert:

- every legal command produces exactly one permitted successor or typed nonadvancing result;
- every illegal command produces no semantic successor and leaves the head unchanged;
- terminal/stale/invalidated objects cannot reenter a current lineage;
- cancellation at every phase cannot produce eligibility or local-minimality proof;
- serialization round-trips preserve exact variants without unknown-to-known coercion;
- candidate permutation never changes the derived preservation-map digest of a semantically identical `CandidateOutcomeMap`;
- action/field variants cannot reach emitter types through generic decoding;
- CAS conflicts are observable and side-effect free; and
- didrun/Wake strings round-trip without affecting transitions.

Required mutants include: control-as-outcome, group-shape preservation, `UNRESOLVED` as false, reused confirmation, empty field acceptance, allow-many cross-product, context-only assertion, noncompilable emission, stale-CAS acceptance, and contract ineligibility collapsed into contradiction. A surviving load-bearing mutant keeps the owning unit open.
