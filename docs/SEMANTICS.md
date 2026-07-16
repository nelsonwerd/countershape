# Countershape semantic authority contract

- **Contract version:** `semantics-v1`
- **Status:** controlling model; runtime authority is receipted through P07A plus the P07B-A1 source/child-bind Observation substrate; current-ruling compilation authority, compiler, and standalone publication remain future
- **Purpose:** separate structural identity, measured facts, comparison assessment, observed behavior, and storage/wire projection

Countershape does not treat a digest-shaped string, a status enum, or a JSON object as evidence merely because it has the right fields. Each stronger state is available only through the operation that can establish it. Some schemas describe exact canonical artifact bodies; others describe external storage or wire envelopes. Production constructors must rebuild the corresponding opaque authority from stricter inputs.

## Canonical body and wire envelope

Every persisted semantic artifact has a canonical body containing `schema_version`, `kind`, and its identity-bearing semantic fields. Its domain-separated digest is computed over that body and addresses it externally in the object path, head, typed reference, or an envelope that explicitly defines `artifact_digest`; the digest is not part of its own hash preimage. The U6 `choicepoint.schema.json` and `decision-record.schema.json` files describe exact canonical bodies and therefore intentionally reject a self-referential `artifact_digest` member. The corrected P07 `ContractBundle`, `ContractExecutionTarget`, `FinalizedContractRun`, and `ContractExecution` bodies follow the same rule; none may contain its own artifact digest.

Some envelopes also carry a separately derived companion digest. In particular, `preservation_map_digest` addresses the eligible labeled map derived from a `CandidateOutcomeMap`; it is not part of the enclosing outcome artifact's hash preimage and is never sufficient by itself to classify preservation.

Schema acceptance proves only that a wire value has the declared closed shape. It does not prove strict canonical parsing, semantic construction, source provenance, physical freshness, host truth, comparison admission, durability, or current-head authority. Strings and collections may have stricter UTF-8 byte caps in Go than JSON Schema's code-point-oriented `maxLength` can express.

U1 did not supply explicit runtime-to-wire codecs for every target schema. Sealed U6 adds strict runtime codecs, exact canonical body-field sets, and one valid runtime/example intersection for `Choicepoint` and `DecisionRecord`; the exact receipt boundary and grades live in `status/U6.md`. This is not equivalence between the JSON-Schema and strict-Go acceptance languages: schemas intentionally overapproximate some UTF-8 byte caps and semantic authority. Future standalone, studio, and export schemas remain planning authorities until their owning units implement and receipt their named parity gates.

## Projection definition and result authority

A `FieldRegistry` has its own domain-separated identity over the exact closed field IDs, paths, types, and missing/null policy. A full `ProjectionDefinition` is a different object. It commits the adapter domain, implementation and configuration digests, accepted channel set, ordered operation pipeline, exact comparator, and field-registry digest. Channel order is normalized; operation order is semantic and preserved.

Construction yields an opaque `ProjectionDefinitionBinding` that retains the declaration whose digest it exposes. `WorldPlan` consumes that binding and requires its adapter domain to equal the plan adapter. A source document may contain a projection-definition digest reference, but source compilation also requires the resolved binding and refuses a zero, mismatched, or differently domained capability. There is no compile-from-bare-digest path.

`ProjectionFingerprint` is only the domain-separated digest of exact canonical projection bytes and is used for equality grouping. `ProjectionResult` is a separate lineage-bearing artifact derived from those bytes plus world, finalized attempt, captured observation, capture policy, and full projection definition. Byte-identical results in different attempts share a fingerprint but never a result digest.

## Portable selected-field interpretation

Historical sealed U6 Choicepoints use `WHOLE_EXACT_CANONICAL_PROJECTION_V1`; their single field is the entire exact projection body. Those records remain valid parseable ruling artifacts but do not authorize a future emitter to assert that only a subset of HTTP or CLI fields matters. Sealed P07A fresh public construction instead uses only `ADAPTER_BOUND_PORTABLE_FIELDS_V1`.

P07A adds that mode without changing either adapter's projection bytes or the existing 27-member Choicepoint body. Construction first verifies the original exact projection proof against the confirmation roster, then resolves the complete adapter definition from the WorldPlan's exact `ProjectionDefinitionBinding`, then strictly translates the verified adapter wire into a complete portable tuple. The portable profile identity binds translator version, adapter domain, complete projection binding, and ordered adapter-owned field descriptors. It is a downstream interpretation digest, not another name for the adapter field-registry digest.

Legacy parsing reconstructs only the whole-projection registry. Fresh public construction uses only the adapter-bound profile. A legacy DecisionRecord offered to the store-bound portable-preparation seam returns `LEGACY_WHOLE_PROJECTION_NOT_PORTABLE`; canonical bytes are never migrated or reinterpreted in place.

The closed portable algebra distinguishes:

- tagged `MISSING` and `NULL`;
- `BOOLEAN`;
- safe `INTEGER` with canonical decimal spelling;
- exact `STRING` and `BYTES`;
- `ORDERED_STRING_LIST`, preserving order, duplicates, empty list, and empty members; and
- exact strict `CANONICAL_JSON` bytes.

These semantics do not imply that the historical CLI and HTTP wire codecs are identical. Their exact existing shapes remain fingerprint authority. The translator is the only versioned bridge. No caller-authored tuple plus a digest can substitute for verified projection bytes.

The sealed expectation domain validates existential model shape only: the selected values must be the restriction of at least one complete tuple admitted by the exact adapter projection model. It does not prove that a current source, candidate, fixture, or runtime can emit the values. That stronger source authority belongs to P07B.

A P07A portable ruling is selected-field semantic authority, not runnable-source authority. Historical stimuli intentionally retain some source content only by length and digest. Sealed P07B-A1 therefore adds a separate opaque `PortableSource` preimage witness. Its closed adapter constructor retains exact bounded bytes and rebuilds and exact-matches its supplied adapter stimulus, WorldPlan, projection binding/definition, portable profile, logical-Node runner/start/readiness/capture authorities, and execution binding. HTTP additionally requires the exact child-bind/pipe-readiness start authority.

### P07B-A1 authority ceiling

`PortableSource.Valid()` proves closed reconstruction integrity over the supplied A1 authorities only. It does not prove equality to a current store-bound `Ruling`, `Choicepoint`, or `FreshConfirmation`; it does not retain reopened projection proofs or ruling tuples; and it does not authorize compilation. Its `closed_facts` are construction declarations, not static analysis, dependency inspection, secret scanning, network denial, confidentiality, or containment.

The A1 child-bind lineage accepts exactly `COUNTERSHAPE_READY_V1 <port>\n` on inherited FD 3 followed by EOF, then uses the reported literal-loopback endpoint for the request. This yields a physical `Observation`, not a FreshConfirmation, portable Choicepoint, or ruling. `child_reported_port` is not kernel evidence that the reporting PID owns the listener; an arbitrary trusted script could report a decoy local service.

P07B-A2 must open and revalidate the current ruling/Choicepoint/FreshConfirmation, compare the source execution binding with every retained schedule-ordered confirmation binding, independently retranslate the reopened confirmation projection proofs under the source-matched profile, and compare the result with the exact selected-tuple ruling partition. The source cannot change the ruling; the ruling cannot recover missing source. That application authority produces a sealed wrapper that privately retains the original preparation for later publication revalidation and separately owns a sanitized internal compiler input with no concrete candidate keys, refs, aliases, producer metadata, or support counts. The exact PortableSource still transitively contains opaque source/lineage identity digests, including candidate-set, plan, envelope, materialization, fixture/capture, and projection identities.

## Candidate reference and binding

`CandidateExecutionKey` is the exact `candidate:<64 lowercase hex>` wire and display reference. It intentionally excludes stimulus, display label, producer, branch spelling, order, support count, and group identity.

The key is not allocation authority. `CandidateExecutionBinding` is an opaque value produced from the full candidate execution identity:

- immutable tree identity;
- materialization policy;
- exact `WorldPlan`;
- adapter identity;
- runner identity; and
- projection definition.

Allocating evidence requires both the opaque binding and the actual `WorldPlan`. The constructor verifies that their plan and candidate references agree. A parsed key or a caller-authored collection of matching digest strings cannot substitute for the binding.

## World, measurement, and assessment chain

The v1 authority chain is:

```text
WorldPlan + CandidateExecutionBinding + stimulus + attempt identity
  -> WorldInstance

WorldInstance + ComparisonEnvelope + complete measured dimension row
  -> InstanceMeasurements

ComparisonEnvelope + one complete row for every candidate in the exact roster
  -> AdmittedComparison + per-row AdmissionToken
     or RejectedComparison
```

`WorldInstance` is structural declared-consistency identity only. It binds the exact plan, candidate binding, stimulus, attempt artifact, purpose, nonce, and schedule ordinal. It contains no roots, ports, PIDs, clocks, platform claims, tool claims, lifecycle result, teardown result, or admission enum. Its digest does not establish that a process ran or that any supplied host fact is truthful.

`InstanceMeasurements` is one complete row under one `ComparisonEnvelope`, bound to one `WorldInstance`. U2 edge code supplies the values and their measurement provenance. U1 establishes coverage, source tags, canonical identity, and policy linkage only; opaque construction does not prove an edge measured the host honestly.

`AssessComparison` is the sole admission operation. It yields:

- a persisted `AdmittedComparison` when the exact candidate roster is covered and every required-equal/reject-on-variance dimension agrees; or
- a persisted `RejectedComparison` with the exact measurement rows and reason codes.

An admitted result carries the plan, envelope, exact candidate roster, exact sorted measurement-digest matrix, and required/rejected equality basis. Every repetition therefore has its own concrete admission artifact. It also derives a `comparison_basis_digest` from only the plan, envelope, roster, and equality basis. That basis deliberately excludes stimulus, attempt purpose, measurement rows, and the per-run admission artifact, so baseline and neighbor studies can be comparable without pretending they are the same run.

Its per-row `AdmissionToken` is internal and ephemeral, deterministically derived from the per-run admitted comparison plus one exact measurement row, stimulus, and attempt purpose. Persisted comparison and measurement artifacts are sufficient to recompute it.

A rejected comparison is a pre-batch study refusal. It produces no admission token, `TrialFact`, `StableBatch`, or `CandidateOutcomeMap`. `ENVELOPE_REJECTED` is therefore not a behavior-shaped batch member. If a later product needs per-candidate pre-admission disposition, that requires a new sealed sum; it must not manufacture admission.

## Trial and batch authority

Every `TrialFact` has exactly one disposition:

- `BEHAVIOR_CAPTURED`, with a captured-observation digest, projection-result digest, and exact projection fingerprint; or
- `CONTROL_INELIGIBLE`, with one or more detected control reasons.

Both variants bind the structural world, finalized attempt artifact, and a deterministic admission-token digest. Captured behavior additionally binds the capture-policy and full projection-definition digests. Captured behavior and control ineligibility cannot coexist.

A `StableBatch` binds one candidate, plan, stimulus, comparison envelope, the exact ordered set of per-repetition admitted-comparison digests, stimulus-independent comparison basis, exact admission roster, attempt purpose, applied schedule digest, `ROTATE_START_BY_REPETITION_V1` policy, schedule ordinals, and tagged trials. Each trial names the admission matrix and row token used for that repetition. Within-batch attempt and world digests must be globally unique. U1 supplied structural ordinal checks; U3 adds the separate deterministic rotated-schedule evidence and refuses any mismatch with the plan-declared sequential policy. The wire fact `duplicate_evidence_within_batch:false` states only structural nonduplication inside that batch; it does not by itself establish physical freshness.

Peer candidate batches may form one `CandidateOutcomeMap` only when they carry the exact same ordered admission-digest set: one shared comparison matrix per repetition. Only an `OBSERVED_STABLE(k/k,h)` batch contributes an eligible map entry. `UNSTABLE`, `UNCOMPARABLE`, and `INCOMPLETE` batches remain explicit exclusions. A `CandidateOutcomeMap` covers the exact expected roster and also binds the plan, stimulus, envelope, admission set, comparison basis, phase, all contributing batches, and the globally unique attempt/world evidence set across those batches.

In U1, “expected roster” means the exact caller-declared structural roster supplied to the constructor. U1 proves entries plus exclusions cover that input; it does not prove Git candidate-set membership or equality with `Budgets.CandidateCount`. U2 must derive and bind the selected roster before any runtime study treats it as candidate-set completeness.

## Persistence and reconstruction

Opaque Go types are the in-process authority boundary. Schemas and examples are wire projections, never alternate constructors. When the sealed U6 store reopens a bounded semantic object it must:

1. verify the external artifact digest against the exact canonical body and kind domain;
2. parse through the strict canonical profile without information loss;
3. validate all semantic references and predecessor authorities; and
4. reconstruct the opaque type through the owning package.

A DTO Boolean such as a derived `compilable` display field, a string status such as `ADMITTED`, or a digest copied from another object is not accepted directly by an emitter, reducer, or state transition.

## Contract residue and execution authority

`ContractBundle` is a canonical semantic body whose external typed digest becomes the terminal `RESIDUE` reference. It embeds the exact recoverable bytes of all six generated files with fixed paths, modes, counts, and raw-byte digests. `manifest.json` covers the other five files and excludes itself; the outer bundle covers all six. Neither the manifest nor a generated file embeds the bundle digest. A didrun receipt or standalone execution fact cannot enter deterministic bundle source because it exists only after that source is built and run.

The store checks the complete expected `RULING` token under the exclusive transition before creating an object or temporary file. It durably publishes and reopens the bundle before advancing the head. A stale loser publishes nothing. A crash after immutable object publication but before head publication may leave an unreachable object; a head must never refer to a missing one. Final output materialization happens only after residue publication and is retryable from the reopened bundle. A post-residue materialization failure reports that durable publication already occurred.

`ContractExecutionTarget` is a separate immutable nonhead pre-spawn authority. It binds one reopened bundle and exact source profile, one explicit Git-pinned inspected and verified private materialization, one fresh durably allocated conformance attempt, and one measured/revalidated Node executable and owned runtime probe. It is constructed only from retained live capabilities; a parsed body, copied OIDs, or copied digests are inert. It does not reuse `WorldInstance`, whose authority remains closed over the historical 2–4-candidate comparison plan. Dirty working-tree bytes are never represented as the pinned target or executed by this edge.

`FinalizedContractRun` is a separate immutable nonhead physical-run authority. It references one exact external `ContractExecutionTarget` digest, repeats that target's exact attempt-artifact digest as a checked join, and owns the closed lifecycle digests, the exact projected tuple when present, scoped target-inventory/child-binding/import/service evidence, and one constructor-derived terminal disposition. `ELIGIBLE_CLEAN` requires clean lifecycle plus exact `PROJECTED` observation; every control path becomes `INELIGIBLE_CONTROL(reason)` from the same closed authority. It is published only after materialization revalidation, process completion, teardown, orphan inspection, and finalization are terminal. A target or attempt mismatch refuses classification.

`ContractExecution` is a separate immutable content-addressed classification object, not another study-head stage and not an append-only mutable execution head. Its body references one exact external `ContractExecutionTarget` digest and that target's exact external `FinalizedContractRun` digest, plus only the derived conformance or ineligible class and explicit nonfreshening facts. The exact tuple or control reason lives only in the finalized run; the classification cannot independently author or echo either. It does not duplicate caller-pairable bundle, tree, attempt, runtime, lifecycle, observation, or standalone fields. Git pin/inspection/materialization, runtime admission, target publication, run finalization, or target/run matching failure refuses classification publication; direct Node execution remains ordinary. Repeated runs allocate distinct target, finalized-run, and execution objects while the study head remains the same terminal bundle residue.

The bundle declares only a runtime-generic Node-core source profile. It is not execution evidence. Exact Node version, major, OS, architecture, executable bytes, and probe program belong to a physically admitted `ContractExecutionTarget`; physical result authority belongs to the matching `FinalizedContractRun`; predicate classification belongs to `ContractExecution`; any didrun grade remains external. Outer materialization verifies all six bundle members. With an intact `contract.test.mjs` entrypoint, direct execution verifies the protected companion-file manifest before subject spawn. Neither layer establishes authorship, authenticity, confidentiality, entrypoint self-authentication, or resistance to coordinated post-materialization replacement.

## Explicit U1 nonclaims

U1 establishes strict structure and pure relationships only. It does not establish:

- that a repository object was materialized;
- that a host fact was honestly measured;
- that an attempt was physically new;
- that a process ran, was contained, or was cleaned up;
- that execution evidence is fresh across batches;
- that a semantic object was durably published or reopened;
- that a reduction grade was earned; or
- that a current study head authorizes a transition.

Those authorities arrive only in their named later units and remain `UNRECEIPTED` until exercised through didrun on the exact claimed tree.
