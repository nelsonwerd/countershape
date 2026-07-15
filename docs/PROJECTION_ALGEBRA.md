# Countershape projection and preservation algebra

- **Contract version:** `projection-algebra-v1`
- **Status:** controlling pure semantics; physical freshness and durable grade authority are separate

This document defines the only generic equality and reduction relations Countershape v1 may use. Display clusters, support counts, aliases, ordering, and outcome-card layout are not semantic inputs.

## Exact projection identity

A successful projection ends in strict canonical bytes. Its `ProjectionFingerprint` is the domain-separated digest of those exact bytes. Equality is exact fingerprint equality under the same full projection definition; v1 has no tolerance, wildcard, regular expression, similarity, inferred invariant, or browser-produced identity.

The full definition is not a caller-paired digest and label. Its opaque binding commits the adapter domain, implementation and configuration, accepted channel set, ordered operation pipeline, exact comparator, and separately typed field-registry digest. A `WorldPlan` accepts this binding and derives the digest/domain from it; a bare wire digest has no execution authority.

`ProjectionResult` is deliberately not the fingerprint. It binds one exact fingerprint and canonical projection to its world, attempt, captured observation, capture policy, and definition lineage. Two runs producing the same bytes share an equality fingerprint while retaining distinct result identities.

`ProjectionFingerprint` is not correctness or semantic equivalence. It states only that the selected projection operation produced the same canonical bytes.

## CandidateOutcomeMap identities

One `CandidateOutcomeMap` has two deliberately different addresses:

1. `OutcomeArtifactDigest` addresses the full evidence-bearing outcome body: plan, stimulus, envelope, exact per-repetition admission-digest set shared across candidate batches, stimulus-independent comparison basis, phase, expected roster, eligible entries, exclusions, contributing batch links, and global attempt/world nonduplication facts.
2. `PreservationMapDigest` addresses only the canonical sorted eligible `candidate_execution_key -> projection_fingerprint` entries.

The second digest excludes stimulus and evidence links so a smaller stimulus with physically new evidence can preserve the same observed split. It also excludes exclusions by design: an eligibility change is not a product behavior change and must become `UNRESOLVED`, not `CHANGES`.

`preservation_map_digest` is a derived companion in the wire envelope. It is recomputed from the canonical eligible entries and is not trusted as caller input.

## Comparability before equality

A naked `PreservationMapDigest` never establishes preservation. First apply `ComparableForPreservation` to the full typed maps.

Two maps are comparable only when all are equal:

- world plan digest;
- comparison envelope digest;
- comparison basis digest, which identifies the exact plan/envelope/roster/equality basis while excluding stimulus and purpose;
- exact expected candidate roster;
- eligible candidate set;
- excluded candidate set; and
- each excluded candidate's classification.

The stimulus, per-run admitted-comparison digest set, attempt/world evidence digests, batch digests, and phase may differ because reduction and confirmation require new evidence under distinct purposes. Comparing the per-run admission artifact digests would incorrectly make every baseline/neighbor pair incomparable.

The complete reduction relation is:

```text
if !ComparableForPreservation(baseline, observed):
    UNRESOLVED("CANDIDATE_ELIGIBILITY_OR_ADMISSION_CHANGED")
else if baseline.PreservationMapDigest == observed.PreservationMapDigest:
    PRESERVES("EXACT_PRESERVATION_MAP_MATCH")
else:
    CHANGES("EXACT_PRESERVATION_MAP_CHANGED")
```

The reducer and durable store must consume either the full typed maps or an opaque result produced by this operation. They must never reconstruct the decision from a display DTO or compare a naked digest.

## Divergence and confirmation refinements

`CandidateOutcomeMap` is the general complete-roster object. It may be nondivergent and may contain exclusions. A nondivergent but comparable neighbor is a valid `CHANGES` result.

`DivergentBaseline` is a sealed refinement requiring at least two eligible candidates and two distinct projection fingerprints.

`ConfirmedOutcomeMap` is a separate sealed refinement requiring the confirmation purpose and a still-divergent complete map. In U1 this is structural binding only. Physical fresh-confirmation authority requires U2 execution evidence and the later durable Choicepoint transition.

## Reduction evidence and grades

U1 owns only:

- strictly decreasing logical neighbors;
- `PRESERVES`, `CHANGES`, and `UNRESOLVED` evaluations; and
- an in-memory logical complete-sweep relation.

U1 owns no reduction grade. In particular it cannot construct `ONE_MINIMAL_UNDER`.

U5 introduces the durable-sweep authority. `internal/reduce` produces a pure logical completion draft and never imports the store. `internal/store` imports that draft, atomically publishes and reopens its exact canonical bytes, and then returns an opaque capability with no public constructor. The thin outward `internal/reduction` service imports both and is the only package permitted to construct `ONE_MINIMAL_UNDER(reducer_set_digest)`. It may do so only when the durably reopened enumerated direct-neighbor set exactly equals the set of new `CHANGES` evaluation records. Any omitted, duplicate, stale, preserving, unresolved, cancelled, post-budget, or non-fresh member prevents that grade. An empty set qualifies only when the empty enumeration itself was completed and durably reopened.

The reducer-set digest binds the finite typed rules, their versions, the adapter-level well-founded measure, and the exact fixed-versus-reducible scope. The local grade never quantifies over stimuli outside that recorded neighborhood. Adapter-level reduction measures are new U5 artifacts; they do not change the sealed CLI or HTTP stimulus digests.

The U5 schemas mirror the inert canonical bodies for `ReductionRun`, `ReductionTranscript`, `CompletedSweepDraft`, and outward `ReductionGrade`. Shape validation is not authority: it cannot establish cross-artifact lineage, set equality, physical freshness, durable publication, or local minimality. The runtime parser and opaque store capability remain required.

## Nonclaims

The algebra does not establish determinism, semantic equivalence, correctness, safety, root cause, global minimum, comprehension, or intent. Finite agreement remains `OBSERVED_STABLE(k/k,h)` under the named plan, envelope, stimulus, schedule, and evidence.
