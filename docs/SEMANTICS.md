# Countershape semantic authority contract

- **Contract version:** `semantics-v1`
- **Status:** controlling model; runtime truth and durable publication remain `UNRECEIPTED` until their owning units are sealed
- **Purpose:** separate structural identity, measured facts, comparison assessment, observed behavior, and storage/wire projection

Countershape does not treat a digest-shaped string, a status enum, or a JSON object as evidence merely because it has the right fields. Each stronger state is available only through the operation that can establish it. JSON schemas describe closed storage/wire projections; production constructors must rebuild the corresponding opaque authority from stricter inputs.

## Canonical body and wire envelope

Every persisted semantic artifact has a canonical body containing `schema_version`, `kind`, and its identity-bearing semantic fields. Its domain-separated digest is computed over that body and accompanies it in the storage/wire envelope as `artifact_digest`; the digest is not part of its own hash preimage.

Some envelopes also carry a separately derived companion digest. In particular, `preservation_map_digest` addresses the eligible labeled map derived from a `CandidateOutcomeMap`; it is not part of the enclosing outcome artifact's hash preimage and is never sufficient by itself to classify preservation.

Schema acceptance proves only that a wire value has the declared closed shape. It does not prove strict canonical parsing, semantic construction, source provenance, physical freshness, host truth, comparison admission, durability, or current-head authority. Strings and collections may have stricter UTF-8 byte caps in Go than JSON Schema's code-point-oriented `maxLength` can express.

U1 does not yet supply the explicit runtime-to-wire codecs for every target schema. Until U6 implements and receipts those codecs, example/schema validation is planning-artifact validation only and runtime/schema round-trip parity is `UNRECEIPTED`.

## Projection definition and result authority

A `FieldRegistry` has its own domain-separated identity over the exact closed field IDs, paths, types, and missing/null policy. A full `ProjectionDefinition` is a different object. It commits the adapter domain, implementation and configuration digests, accepted channel set, ordered operation pipeline, exact comparator, and field-registry digest. Channel order is normalized; operation order is semantic and preserved.

Construction yields an opaque `ProjectionDefinitionBinding` that retains the declaration whose digest it exposes. `WorldPlan` consumes that binding and requires its adapter domain to equal the plan adapter. A source document may contain a projection-definition digest reference, but source compilation also requires the resolved binding and refuses a zero, mismatched, or differently domained capability. There is no compile-from-bare-digest path.

`ProjectionFingerprint` is only the domain-separated digest of exact canonical projection bytes and is used for equality grouping. `ProjectionResult` is a separate lineage-bearing artifact derived from those bytes plus world, finalized attempt, captured observation, capture policy, and full projection definition. Byte-identical results in different attempts share a fingerprint but never a result digest.

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

Opaque Go types are the in-process authority boundary. Schemas and examples are wire projections, never alternate constructors. When a future store reopens an object it must:

1. verify the external artifact digest against the exact canonical body and kind domain;
2. parse through the strict canonical profile without information loss;
3. validate all semantic references and predecessor authorities; and
4. reconstruct the opaque type through the owning package.

A DTO Boolean such as a derived `compilable` display field, a string status such as `ADMITTED`, or a digest copied from another object is not accepted directly by an emitter, reducer, or state transition.

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
