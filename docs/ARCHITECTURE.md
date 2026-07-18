# Countershape architecture contract

- **Contract version:** U0 / `architecture-v1`
- **Target:** narrowed Darwin reference instrument
- **Status:** controlling design; U1–U6, P07A, P07B-A1, P07B-A2.1, and accepted P07B-A2.2 receipts are enumerated without reinterpretation in their status files; P07B-B terminal publication/retryable native materialization is implemented on the working tree but remains unreceipted until its independent commit/seal/strict gate; target, execution, and product surfaces remain future
- **Authority order:** `CONCEPT_BRIEF.md`, `SEMANTICS.md`, `PROJECTION_ALGEBRA.md`, then the implementation-time correction in `research/deep-dive/11-p07-implementation-red-team.md`, then the earlier `research/deep-dive/08-RED_TEAM.md`, then this document

Countershape is a repository-scale operational composition for resolving one witnessed behavioral disagreement among exact repository candidates. It is not an agent runtime, candidate ranker, generalized workflow runner, correctness oracle, or new disambiguation algorithm. The reference instrument accepts trusted local code from a curated Git repository, performs finite fresh executions, compares exact projected bytes, records a local-caller-attributed selected-field ruling, and may preserve that ruling as standalone Node source only when a separate exact source preimage reconstructs the witnessed run.

This document fixes the component boundaries, dependency direction, truth ownership, persistence model, and U0–U9 growth path. Later units may narrow a capability when a gate fails. They may not merge truth jurisdictions or silently promote a weaker artifact to a stronger claim.

## Architectural invariants

1. **Different questions have different authorities.** Git selection, materialized bytes, declared execution, measured execution, captured evidence, projected values, observed disagreement, local-caller ruling, runnable source reconstruction, emitted source, current conformance, and external command receipts are separate objects.
2. **A comparison envelope is a declaration, not a proof.** It records which measured dimensions were required equal, tolerated, rejected, or left uncontrolled. Admission never establishes that tolerated placeholders were irrelevant to candidate control flow.
3. **Only exact canonical values compare.** V1 has no tolerance, similarity, wildcard, regex, inferred invariant, or order-dependent cluster identity.
4. **Comparability precedes labeled-map equality.** Reduction, confirmation, persistence, API, and studio first require the same plan, envelope, stimulus-independent comparison basis, expected roster, eligible set, exclusions, and exclusion classifications. Only then does equality of the canonical sorted `candidate_execution_key -> projection_fingerprint` map digest mean preservation. Display groups and naked digests carry no preservation authority.
5. **Control failure is not program behavior.** Only eligible finalized attempts can enter projection and a stable batch. Explicit setup, start, readiness, transport, timeout, output, projection, orphan, teardown, or cancellation failures remain typed controls.
6. **Every evidentiary execution is new.** Discovery, each reduction proposal, final sweep, confirmation, and current conformance allocate new attempt artifacts, private roots, and process lifecycles. Only pure transformations of immutable bytes may be memoized.
7. **Human scope is exact and narrow.** A compilable decision covers one witnessed stimulus and an explicit nonempty set of selected typed fields. Nonasserted fields remain visible and intentionally unconstrained.
8. **Portable ruling and runnable source are separate authorities.** Exact historical adapter projection bytes plus their exact binding may derive selected portable tuples. A separate byte-complete source preimage must reconstruct the same plan and minimized stimulus before standalone compilation.
9. **Standalone is a separate implementation.** The generated Node harness earns no parity claim until the same normative vectors pass in Go and Node and the bundle runs with Countershape absent from the exact tested target inventory.
10. **Persistence stores semantic artifacts, not runtime history.** Immutable content-addressed objects and atomic compare-and-swap heads are the durable model. Progress streams are disposable and cannot establish evidence.
11. **Failure narrows the product.** Unsupported source forms, uncontrolled variance, incomplete trials, unstable candidates, ambiguous selected fields, stale decisions, source-preimage mismatch, and unavailable runtimes refuse the stronger transition.

## Truth jurisdictions

| Question | Sole authority | Bound by | Does not establish |
| --- | --- | --- | --- |
| Which source was selected? | `TreeIdentity` | repository fingerprint, object format, immutable commit/tree OIDs | materialized bytes, dependencies, or behavior |
| Which supported entries were written? | `MaterializationManifest` | policy digest, sorted blob/mode records, byte verification, portable tree digest | a complete checkout, expanded Git forms, or containment |
| What execution was declared? | `WorldPlan` | deterministic argv, roots policy, sparse environment, fixture/readiness recipes, capture/projection, schedules, budgets | actual host resources or runtime facts |
| What structural attempt was declared consistent? | `WorldInstance` | exact plan and candidate binding, stimulus, attempt artifact, purpose, nonce, ordinal | process execution, host facts, freshness, admission, or equivalence |
| What runtime dimensions were reported for that attempt? | `InstanceMeasurements` and edge receipts | exact structural instance, envelope, complete policy-bound row, measurement provenance | honest host measurement or comparison admission |
| Why may one exact measurement matrix proceed? | `ComparisonAdmission` produced by `AssessComparison` | plan, envelope, exact digest-sorted measurement set, candidate roster, equality basis, derived comparison-basis digest | behavioral equivalence, physical freshness, or placeholder irrelevance |
| Why did a matrix stop before batching? | `RejectedComparison` produced by `AssessComparison` | exact measurement digests and typed reasons | any token, trial, batch, or outcome |
| What bytes or channels were retained? | `CapturedObservation` | capture policy, tagged channels, control result, bounded artifact digests | discarded pre-capture bytes or semantic meaning |
| What values compared equal? | `ProjectionDefinition` and `ProjectionResult` | visible versioned operations and exact canonical bytes | correctness, safety, or semantic equivalence |
| What finite split was observed? | `StableBatch` and `CandidateOutcomeMap` | tagged trials, shared per-repetition admission set, exact labeled map, exclusions, and global evidence identities | determinism, physical freshness, majority truth, or complete intent |
| What was locally reduced? | `ReductionRun` | typed neighbors, decreasing measure, nonoverlapping declared evidence identities, transcript, budgets, complete logical sweep if present | physical freshness, root cause, global minimum, or comprehension |
| What is ready for a human decision? | `Choicepoint` | fresh-confirmed lineage and all governing artifact digests | a recommendation or branch endorsement |
| What ruling did the local caller record? | `DecisionRecord` | exact Choicepoint, blind/reveal presentation facts, caller-asserted actor, action, selected and nonasserted fields | actor authenticity, comprehension, intent outside the witness, security approval, or runnable source recovery |
| Which supplied bytes form a self-consistent runnable reconstruction? | live `PortableSource` reconstruction witness | closed adapter constructor, exact supplied plan/projection/stimulus/profile/runner/start/execution-binding equality | equality to the current store-bound ruling or confirmation, compile authority, proof that source was historically retained, or hostile-code containment |
| What source bytes were emitted? | `ContractBundle` | deterministic recoverable six-file set at an external typed digest | authorship, current execution, receipt status, or long-term fidelity |
| What exact target may one conformance attempt run? | immutable nonhead `ContractExecutionTarget` | reopened bundle/source profile, explicit pinned and verified Git materialization, fresh durable attempt, admitted/revalidated Node runtime | process execution, historical `WorldInstance`, dirty working-tree bytes, or caller-paired digests |
| What physically finalized against that target? | immutable nonhead `FinalizedContractRun` | exact target and attempt, closed lifecycle, constructor-derived clean/ineligible disposition, exact projected tuple when present, scoped standalone evidence | conformance classification, another target/attempt, host-wide absence, or confidentiality |
| How is that exact finalized run classified? | immutable nonhead `ContractExecution` | exact target digest, exact matching finalized-run digest, derived conformance or ineligible class | independent tuple/reason authorship, recreation of historical evidence, study-head advancement, or overall correctness |
| Did an external command run on a Git tree? | didrun's referenced verbatim grade, when present | its exact command and Git association | any Countershape classification or stronger unreceipted claim |

Git remains the content authority for selected objects. Countershape is authoritative only for canonical objects and transitions it actually creates. The human is authoritative only for the fields and witness explicitly ruled on. Wake may be inert producer provenance. didrun remains an external receipt authority.

## Canonical identity boundary

Every persisted semantic object has a canonical semantic body and an external storage/wire envelope:

```text
schema_version
kind
semantic_fields
canonical_body_bytes = canonical(schema_version, kind, semantic_fields)
artifact_digest = SHA-256("countershape/v1/" + kind + NUL + canonical_body_bytes)
```

The digest accompanies the body in the wire envelope and is never part of its own hash preimage. Derived companions such as `preservation_map_digest` are recomputed from their named semantic projection and never trusted as caller input. The Go identity layer must reject information loss before a typed value is constructed: duplicate object names, invalid UTF-8, lone surrogates, non-finite numbers, negative zero, unsafe internal integers, and unsupported numeric forms. It never performs Unicode normalization. Durations are integer milliseconds. Exact external decimals, if a later profile admits them, are tagged canonical strings rather than binary floating-point values.

The studio never computes identity-bearing bytes or digests. JSON serialization from a browser is transport only. Server application services parse typed requests, validate the current head with compare-and-swap, and create the canonical object.

The Node contract harness is not allowed to substitute ordinary `JSON.parse` for an identity-sensitive profile that Go parses strictly. Either the exercised profile has a matching strict Node parser and vector receipt, or that shape is absent from the standalone profile.

## Package and dependency contract

The implementation layout is fixed at the architectural level:

```text
cmd/countershape/          composition root and CLI presentation
internal/canon/            strict tokens, canonical values, domain digests
internal/domain/           immutable sums, value objects, construction invariants
internal/spec/             inert JSON specification -> deterministic WorldPlan
internal/gitobj/           immutable ref selection and supported blob materialization
internal/world/            fresh attempts and Darwin process ownership
internal/observe/          eligibility, schedules, repeated classification
internal/compare/          projection fingerprints, comparison basis, and complete CandidateOutcomeMap
internal/reduce/           pure bounded tri-valued search and logical sweep drafts
internal/reduction/        outward reducer/store composition and authoritative grades
internal/portablevalue/    closed adapter-neutral exact value algebra
internal/projectionprofile/ opaque derived adapter-bound profile identity
internal/projectiontranslate/ strict historical CLI/HTTP wire translation
internal/choice/           lineage, decision actions, field-separation validation
internal/contractsource/   byte-complete runnable-source reconstruction
internal/runnerprofile/    closed logical-Node runner and launch profile identity
internal/emit/node/        deterministic six-file standalone source emitter
internal/contractexec/     immutable nonhead current execution publication
internal/store/            content-addressed objects and CAS heads
internal/server/           authenticated loopback transport and typed DTOs
internal/report/           default-minimized local evidence export
internal/adapters/cli/     CLI stimulus, capture, projection, fields, neighbors
internal/adapters/http/    HTTP stimulus, capture, projection, fields, neighbors
web/                       React/Vite renderer; no truth recomputation
testkit/                   generated Git repositories, process fixtures, vectors
spec/                      schemas, examples, and cross-language normative corpus
```

### Dependency direction

- `canon` is the lowest project-owned identity layer and depends only on the Go standard library.
- `domain` may depend on `canon`; it exposes construction-safe immutable values, not mutable records with validity booleans.
- `spec`, `observe`, `compare`, `reduce`, `choice`, and the byte-producing portion of `emit/node` depend inward on `domain` and `canon`. They do not import process, Git, HTTP, CLI, server, browser, report, storage, or filesystem-publication implementations.
- `gitobj`, `world`, `store`, typed adapters, `server`, and `report` are edge packages. They may depend on inward contracts. Inward packages never import them. Atomic publication of an emitted bundle is an edge operation; `emit/node` itself returns a deterministic artifact set without performing I/O.
- `cmd/countershape` is the composition root. It wires concrete edges to pure application services and contains no alternate truth rules.
- `web` consumes typed server DTOs and opaque digests. It never imports Go-generated algorithms recreated in TypeScript.

Cycles across these layers are prohibited. Convenience may not move domain-specific facts into a generic record.

### Generic kernel boundary

The generic kernel may own only:

- content identity and canonical references;
- detected-control eligibility;
- repeated-batch classification;
- exact labeled-map comparison;
- tri-valued reduction orchestration over a typed neighbor provider;
- immutable lineage and staleness;
- decision transition and field-separation validation; and
- deterministic artifact-set construction.

The CLI adapter owns ordered argv, `Absent` versus `Present("")`, sparse environment, bounded fixture files, exit versus signal, stdout, stderr, CLI projection fields, and CLI neighbors.

The HTTP adapter owns fixture-owned readiness, method/path, ordered query and header multimaps, request body, transport versus response, status, bounded response body, HTTP projection fields, and HTTP neighbors. Its cycle-free model authority is the only HTTP package the impure world edge may import. A thin `ExecuteHTTP` edge may reuse world allocation, materialization, process-group ownership, and teardown while one private HTTP-specific world sequencer owns inherited readiness plus one direct exchange. This is not a neutral or exported arbitrary service runner, and it does not move HTTP field truth into generic packages. The reference readiness wire is exactly `ONE_BYTE_0X01_THEN_EOF_V1` on a dedicated inherited pipe.

There is no generic JSON observation inside the truth kernel. The adapters may both project to canonical selected-field tuples, but neither is coerced into the other's capture model. Adding HTTP branches to `compare`, `reduce`, or `choice`, or coercing CLI capture into an HTTP-shaped record, fails the two-domain claim and narrows the instrument to one adapter.

## Source identity and materialization

The source path is Git-object based, not working-tree based:

1. Resolve each display ref once to immutable commit and tree OIDs.
2. Record the repository fingerprint, object format, Git version, and display spelling.
3. Enumerate the selected tree through filter-free plumbing.
4. Accept only Git modes `100644` and `100755`.
5. Stream each blob without checkout, worktree, or archive policy; disable replacement-object interpretation and lazy promisor fetching.
6. Rehash streamed bytes against the repository object format and verify the named object.
7. Refuse symbolic links, gitlinks, LFS pointers, unsafe or non-UTF-8 paths, path aliases/collisions, unsupported modes, and missing objects.
8. Write only into a new mode-0700 attempt-owned root, preserving the supported executable bit.

Materialization is offline. A missing promisor object is a refusal, never permission to fetch. The portable tree digest excludes absolute root, current time, random attempt identity, and target-specific inode facts. Attempt allocation and host facts belong to U2 measurement/process artifacts, not the structural `WorldInstance` identity.

Two candidate slots resolving to the same executable tree are rejected or explicitly coalesced before observation. Branch names and producer labels never become execution identity.

## World plan, instance, and comparison envelope

`SourceSpec` is a strict inert input, not an evidence artifact. Its only constructor tokenizes through the lossless canonical parser and applies a closed nested field grammar; ordinary JSON decoding and exported Go structs are not production inputs. Source bytes, strings, argv, environment entries, secret slots, and required tools all have pre-execution bounds. Missing optional budgets/repeat counts are materialized by compilation, while explicit zero remains explicit and is refused where invalid. Unknown keys and semantically impossible adapter/readiness/version combinations fail before a usable `ParsedSource` capability or `WorldPlan` can exist. Multiple refusals are selected in a fixed field order.

`WorldPlan` contains every deterministic choice that should survive clean reproduction: candidate set digest, declared candidate count, materialization policy, adapter and runner digests, direct logical argv arrays, cwd policy, a closed reference-environment profile (`LANG=C`, `LC_ALL=C`, `TZ=UTC`, `NO_COLOR=1`, `NODE_NO_WARNINGS=1`), secret-slot presence policy, private-root rules, fixture/readiness recipes, `HOST_ALLOWED`, capture and projection definitions, repeat counts, schedule, tool requirements, concurrency mode, and all budgets with defaults materialized. V1 permits only a bare declared tool as argv zero and a bounded closed argument grammar; resolution to an absolute executable is a later measured tool/process fact. It refuses other environment names or values rather than pretending a name heuristic establishes nonsecret content. It contains no wrapper or shell string, current time, host path, temp path, allocated port, PID, or ambient interpolation. The declared candidate count is budget input only; equality with the exact selected roster is deferred to the U2 candidate-set gate and is never inferred from the opaque set digest.

`CandidateExecutionKey` is the exact wire/display reference. It is not allocation authority. `CandidateExecutionBinding` is opaque and retains immutable tree identity, materialization policy, the exact `WorldPlan`, adapter, runner, and projection definition. Allocating structural evidence requires that binding plus the actual matching plan; a key or copied digest fields cannot substitute.

`WorldInstance` records declared structural consistency only: plan digest, candidate key and binding digest, stimulus digest, attempt-artifact digest, purpose, nonce, and ordinal. It has no roots, ports, PIDs, clocks, platform/tools, lifecycle, teardown, measurement values, or admission status. U2 edges record those runtime observations in `InstanceMeasurements` and process/materialization receipts.

`ComparisonEnvelopePolicy` declares which dimensions must be measured and how variance is treated. `InstanceMeasurements` is one exact complete policy-bound row for one structural instance. `AssessComparison` is the sole constructor that evaluates the complete candidate matrix and persists either `ComparisonAdmission` or `RejectedComparison`. An admitted artifact includes the exact candidate roster and exact digest-sorted measurement set. Each concrete repetition therefore has a unique admission artifact. A derived `ComparisonBasisDigest` covers only plan, envelope, roster, and the required/rejected equality basis, deliberately excluding stimulus, purpose, and measurement rows so distinct runs remain comparable.

The policy and assessment expose five explicit dimension sets:

- `measured`: facts observed for each instance;
- `required_equal`: dimensions whose variance rejects comparison;
- `tolerated`: measured dimensions permitted to differ for this finite experiment;
- `rejected_variance`: differences that actually excluded instances; and
- `uncontrolled`: known or classed dimensions not controlled or fully measured.

Ephemeral roots, ports, PIDs, and timing may be measured and listed as tolerated, while visible occurrences are transformed by a named projection operation. That transformation establishes only equality of projected captured values. It does not establish that those values had no earlier effect on program branches. Curated fixtures may add metamorphic checks; imported repositories receive no stronger language.

An admitted matrix creates deterministic internal, ephemeral row tokens bound to the concrete admission, measurement row, stimulus, and purpose. Persisted admission and measurement artifacts can reconstruct them. A rejected matrix is a pre-batch refusal and creates no token, trial, stable batch, or outcome map. V1 executes sequentially with a recorded, rotated candidate schedule. Concurrency would introduce a new envelope and is out of the reference claim.

## Observation, projection, and exact comparison

An execution attempt passes through the centralized eligibility gate. An application response such as HTTP `500`, CLI exit `2`, or complete empty stdout may be eligible behavior. `MATERIALIZATION_ERROR`, `SETUP_ERROR`, `START_ERROR`, `READINESS_ERROR`, `PROBE_TRANSPORT_ERROR`, `TIMEOUT`, `CANCELLED`, `OUTPUT_LIMIT`, `PROJECTION_REJECTED`, `ORPHAN_RISK`, and `TEARDOWN_ERROR` are controls and cannot be represented as projected outcomes.

Countershape separates detected controls only. A declared setup program can exit zero after doing the wrong work, and a readiness mechanism can have hidden side effects. Both proof studies avoid setup. CLI is one-shot-ready; HTTP alone uses fixture-owned inherited-pipe readiness evidence and never an HTTP warm-up request.

`CapturedObservation` is post-capture-policy evidence. Every channel is tagged `Present`, `Absent(reason)`, or `Truncated(digest,count)`. The word “raw” is reserved for persisted bytes unchanged by capture policy. `ProjectionDefinition` is pure, visible, versioned, and names each operation. A successful result ends in strict canonical bytes and a fingerprint; a rejection is an ineligible control.

For `k` required new eligible trials, the only batch classifications are:

- `OBSERVED_STABLE(k/k,h)` when every eligible projection fingerprint is `h`;
- `UNSTABLE(histogram)` when at least two eligible fingerprints occur;
- `UNCOMPARABLE(reasons)` when an admitted trial is later ineligible because its source, control, or projection cannot contribute; and
- `INCOMPLETE` when the declared budget ends before `k` eligible trials.

Envelope assessment happens before batching. `RejectedComparison` is a study refusal, not `UNCOMPARABLE` behavior: it creates no admission token, tagged trial, batch, or map. For an admitted study, `StableBatch` preserves each captured-or-controlled trial tag, plan, candidate binding, stimulus, comparison basis, exact admission roster, repeat policy, phase, and ordered per-repetition admission digests. Peer candidate batches may join one map only when those admission-digest sets match exactly. Finite agreement is never promoted to determinism. A divergence requires at least two eligible candidates and at least two projection fingerprints.

`CandidateOutcomeMap` is a complete canonical sorted map plus explicit exclusions for the exact candidate roster bound by the plan. It also binds stimulus, envelope, stimulus-independent comparison basis, exact shared per-repetition admission-digest set, phase, contributing batches, and globally unique attempt/world evidence identities. `CandidateExecutionKey` supplies only the stable label inside the map; `CandidateExecutionBinding` retains executable identity. Candidate order, producer, ref spelling, support count, and display grouping do not affect the preservation-map digest. A `DivergentBaseline` is a sealed refinement requiring at least two eligible candidates and two distinct fingerprints; a nondivergent comparable neighbor remains a valid `CandidateOutcomeMap` for a `CHANGES` result.

## Tri-valued reduction

Each typed adapter supplies a deterministic finite neighbor function and a strictly decreasing well-founded measure. The reducer orchestrator records every proposal and obtains new eligible batches. A proposal is:

- `UNRESOLVED` when the full typed maps differ in plan, envelope, comparison basis, expected roster, eligible set, exclusions, or exclusion classifications, or when no complete eligible map exists;
- `PRESERVES` when comparable maps have equal complete sorted eligible candidate-to-projection digests; or
- `CHANGES` when comparable maps have different complete sorted eligible candidate-to-projection digests.

Membership shape, number of groups, ordinal group IDs, and a naked `PreservationMapDigest` are never comparison inputs. Persistence, reducer, confirmation, API, and UI carry full typed maps or an opaque result from the comparability-first operation. The evidence-bearing outcome artifact has a distinct `OutcomeArtifactDigest` and cannot be substituted.

The preservation digest is the complete sorted candidate-key-to-fingerprint map, not the enclosing `CandidateOutcomeMap` artifact digest: enclosing artifacts also bind the particular stimulus, concrete admission matrices, phase, and evidence batches, which must change on a physically fresh shrink evaluation. `ComparisonBasisDigest` remains stable across those runs. Comparing per-run admission artifacts, putting stimulus inside `CandidateExecutionKey`, or comparing full artifact digests would make every nontrivial shrink incomparable or changed by construction and is forbidden.

The only reduction grades are `UNCHANGED`, `BEST_KNOWN`, and `ONE_MINIMAL_UNDER(reducer_set_digest)`. Construction of the last grade requires a live run using the exact compiled-`WorldPlan`-bound shrink budget, no recorded limitation, a complete logical final sweep whose enumerated valid direct-neighbor set equals its set of typed `CHANGES` evaluations, and a matching durable-store authority. Complete map-backed results must report a trial count equal to their typed attempt-evidence cardinality. Evaluation construction proves same-domain content-digest nonoverlap with the current baseline and earlier evaluations; it does not prove physical execution, hidden work, or freshness. Unresolved/error-only counts remain a trusted evaluator-edge assertion. The reference process edges are responsible for enforcing new attempts, process lifecycles, and child deadlines. A cancelled sweep, exhausted budget, partial record, ad-hoc unbound budget, stale plan, changed eligibility, or any `UNRESOLVED` neighbor cannot construct the strong type. U1 deliberately exposes only logical neighbor/sweep evidence. U5 keeps `internal/reduce` pure: `internal/store` imports its completion draft and returns an opaque, unconstructible capability only after atomic publication and reopen; the thin outward `internal/reduction` service imports both packages and is the sole grade-construction edge. This avoids both an import cycle and a forgeable reducer-owned store interface.

The reducer-set identity binds rule names and versions, the adapter-owned reduction-measure definition, and a policy-scope digest: fixed semantic anchors, pin sets, execution shape, and enabled transform classes. It does not bind the current values or membership of reducible argv, environment, query, header, body, or unpinned fixture/seed members; the run's original/current stimulus digests and proposal chain bind those exact values. `ONE_MINIMAL_UNDER` therefore means one-minimal only among the final stimulus's direct neighbors enumerated by that recorded policy, never under every constructible stimulus. Reduction measures are adapter-level canonical tuples and do not alter the already-sealed CLI or HTTP stimulus identities.

A serialized reduction transcript is inert intent, not executable evidence. It binds the baseline outcome/preservation identities and sorted four-domain evidence declarations, but `ExecuteReplay` additionally requires the exact opaque compare-owned `DivergentBaseline` and refuses any mismatch before a callback runs. Replay receipts record only structurally valid same-domain digest noncollision against that typed baseline, the archive, and earlier replay returns. They do not prove physical execution, freshness, proposal correspondence, historical-decision reproduction, or artifact authorship.

The minimized witness must pass a physically new confirmation batch with new attempt artifacts, invocation evidence outside the selected predicate, and a rotated schedule. A new nonce around copied evidence is invalid.

## Choicepoint and decision boundary

A `Choicepoint` is an immutable `CHOICEPOINT_READY` revision. It binds original and reduced stimuli, exact candidates, plan, envelope, projection identity, fresh-confirmed exact map and exclusions, bounded reduction facts, and evidence references. The U6 store implements one fixed linear head spine only; branching successor lineages and durable `STALE`/`INVALIDATED` status objects are target architecture, not current authority. Historical sealed U6 construction used `WHOLE_EXACT_CANONICAL_PROJECTION_V1`. Sealed P07A preserves those records byte-for-byte while making new public construction use `ADAPTER_BOUND_PORTABLE_FIELDS_V1` over the same 27-member body.

The U6 blind semantic DTO kernel contains caller-authored scenario text, exact scope, original/reduced stimuli, bounded reduction facts, configured repeat counts, projection operations, deterministic outcome facts, and distinct outcome count. It omits system-supplied candidate identity, producer metadata, total candidates, per-outcome support counts, candidate source order, and reveal data in its closed canonical shape. Ordering is fingerprint-only and aliases are Choicepoint-scoped. Free-form scenario text and behavior bytes may themselves reveal identity; U6 does not establish anonymity, content-taint safety, DOM/accessibility omission, or the complete U8 presentation payload.

The semantic session records a provisional action and explicit field decisions, then records provenance reveal before finalization. Early reveal and post-reveal changes are facts, not proof of understanding. No field is preselected. Every differing projected field is explicitly `Assert` or `Context only`. Historical U6 records expose only the whole projection; sealed P07A implements the strict profile-bound translation that makes adapter fields genuinely selectable without changing historical projection fingerprints. A server rendering and transporting this session remains U8 work.

Compilable actions are `ALLOW_OBSERVED` and `CUSTOM_EXPECTATION`. For selected fields `F`, allowed tuples `A`, and disallowed confirmed tuples `D`, compilation requires:

```text
for every a in A and d in D: select_F(a) != select_F(d)
```

Missing and present-empty values remain distinct. The portable algebra additionally distinguishes null, empty bytes, empty ordered list, and a present ordered list containing an empty string. Allow-many is a canonical set of complete selected-field tuples derived from a sealed confirmed-outcome set. It is never a cross-product. Empty selection returns `EMPTY_SELECTED_FIELDS`; failed separation returns `AMBIGUOUS_SCOPE`. A custom expectation contains exactly the selected fields, contains no unselected field, and differs from every confirmed selected-field tuple; equality returns `CUSTOM_EXPECTATION_ALREADY_OBSERVED`. These failures create no compilable authority. `REJECT_ALL`, `DEFER`, and `REFINE` are noncompilable by construction. U6 can construct a semantic `REFINE` DecisionRecord, but durable promotion refuses it until successor-study creation exists.

## Portable ruling, A1 source substrate, standalone residue, and Go/Node parity

Sealed P07A derives one closed profile from the exact WorldPlan projection binding and translates only roster-verified existing CLI/HTTP projection bytes. It never rewrites the adapter wire. The profile binds translator version, adapter, complete binding, ordered adapter-owned descriptors, and an existential model-shape expectation domain. Legacy whole-projection records remain valid history and return `LEGACY_WHOLE_PROJECTION_NOT_PORTABLE` at the current portable-preparation seam.

Sealed P07B-A1 implements the opaque byte-complete `PortableSource` substrate. Its closed adapter constructor reparses and exact-matches the supplied WorldPlan, projection binding/definition, minimized-stimulus canonical bytes/digest, P07A profile, logical-Node runner/start/readiness/capture authorities, and reconstructed execution binding. It preserves absent versus present-empty payloads and refuses inherited-listener HTTP source. This is a self-consistency boundary over supplied construction authorities, not yet an equality join to the current store-bound `Ruling`, `Choicepoint`, or `FreshConfirmation`.

P07B-A1 also implements `NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1`: the child reports `COUNTERSHAPE_READY_V1 <port>\n` over inherited FD 3 and closes it before the parent performs the exact raw HTTP exchange. The resulting A1 receipt reaches `Observation` only. `child_reported_port` proves the accepted child-process-tree frame and its use for the subsequent request; it is not kernel evidence that the reporting PID owns the listener, and arbitrary trusted code could report a decoy loopback service. Trusted code retains the user's filesystem and network authority.

Sealed P07B-A2.1 separately opens and revalidates the current ruling/Choicepoint/FreshConfirmation, requires the source execution binding to equal every retained confirmation binding, independently retranslates reopened projection proofs under the source-matched profile, and revalidates the selected-tuple partition before compiler bytes exist. It returns only a sealed preparation with a private authority-narrowed input. Source retains no proof or ruling tuple bytes. A current ruling alone cannot recover raw fixture, seed, stdin, or body bytes retained historically only by digest.

The historical inherited-listener HTTP profile remains nonemittable. A1 exercised the new child-bind profile through a physical Observation; A2.1 exercises its fresh confirmation, portable Choicepoint, selected-field ruling, and compilation join. Accepted A2.2's generated HTTP harness uses raw `node:net` bytes and the exact Go HTTP/1.1 grammar; high-level normalizing HTTP calls are outside the profile.

The deterministic output directory contains exactly:

```text
README.md
contract.test.mjs
decision.json
fixture.json
harness.mjs
manifest.json
```

The closed predicate profile is `one-of-exact/v1`. The ContractBundle embeds exact base64 contents, modes, counts, and raw-byte hashes for all six files. `manifest.json` covers only the other five; the outer bundle covers all six. The bundle's typed digest stays external and no generated file embeds it. The determinism profile is scoped to typed structural facts invented by the emitter, excluding authorized input content: it introduces no current time, random ID, absolute path, concrete candidate/ref identity, declared secret value, host-runtime fact, or execution receipt as new authority. Generated harness code also adds no Countershape dependency, package install, shell, registry call, or configured external-service dependency. This is not a content scan or confidentiality claim; exact PortableSource bytes, predicate values, and source-derived human text may themselves contain sensitive, host-looking, or candidate-looking data. `external_service_binding = NONE` describes only the generated harness: trusted subject code retains the user's filesystem and host-network authority, and the HTTP contract uses local loopback to that subject.

`DecisionRecord`, `ContractBundle`, `ContractExecutionTarget`, `FinalizedContractRun`, and `ContractExecution` are separate truth objects. Bundle hashing proves byte integrity, not authorship or conformance. The store revalidates the complete expected `RULING` token before any object/temp creation, publishes and reopens the bundle before advancing to terminal `RESIDUE`, and materializes only afterward from recoverable bytes. Materialization failure after residue is retryable and does not imply rollback. A later target is published and reopened before spawn from one explicit pinned Git materialization, the reopened bundle/source profile, a fresh conformance attempt, and an admitted Node runtime; it is not a `WorldInstance` and never advances the study head. After the physical attempt closes, `FinalizedContractRun` binds that exact target and attempt to lifecycle, the constructor-derived terminal disposition, exact projected tuple or control reason, and scoped standalone facts. A clean target-eligible selected tuple is `CONFORMS` or `CONTRADICTS`; startup, readiness, timeout, capture, projection, output, or teardown failure is `INELIGIBLE_EXECUTION`, even though contradiction and ineligibility both cause an ordinary test failure. `ContractExecution` references only the exact target plus its exact finalized run and derived result class; it independently authors neither tuple nor reason and is published separately.

One normative corpus must exercise Go and Node strict parsing, field selection, missing/empty distinctions, CLI and raw HTTP capture profiles, projection bytes, and eligible/ineligible errors. Mutations that accept duplicate names, use independent allow-many value sets, assert a context-only field, normalize HTTP wire differences, or collapse ineligibility into contradiction must fail in both authorities. Standalone remains `UNRECEIPTED` until this corpus and a physical target-inventory absence run pass on the exact Node version, major, native OS, and architecture named in the receipt. A runtime-generic bundle profile alone receipts none of those tuples.

## Bounded semantic object store and fixed study-head spine

The durable store has two concepts:

1. **Immutable objects:** canonical semantic artifacts addressed by domain-separated digest. A digest collision or mismatched kind/bytes is fatal. Objects are written to a private temporary file, fsynced as required by the platform policy, published with an atomic create-if-absent operation, reopened, and rehashed before authority is returned. The Darwin U5 sweep store uses a same-directory hard link rather than a replacing rename, then unlinks the temporary name and fsyncs the directory.
2. **CAS heads:** small mutable selectors such as a study's current lineage. A write supplies the expected old digest and desired new digest. Missing or stale expectations fail without creating a semantic transition.

The raw head-advance primitive is unexported. Public baseline, divergence, and reduction transitions consume live exact typed values; confirmation, Choicepoint, and ruling transitions consume opaque publication capabilities bound to the exact predecessor. The confirmation issuer is confined to the confirmation subtree, and Choicepoint/ruling issuers to the store-validating promotion subtree. Strict serialized records remain inert and cannot recreate those capabilities. `REFINE` is denied before ruling authority issuance and again by the store's closed ruling-action set.

Object edges are digest references inside immutable bodies. A new revision never edits an ancestor. U6a's public typed spine advances through `SOURCE_PLAN -> BASELINE -> DIVERGENCE -> REDUCTION -> CONFIRMATION -> CHOICEPOINT_READY -> RULING`. The internal stage table reserves the terminal `RESIDUE` slot; P07A does not expose it, and P07B owns construction-safe `ContractBundle` publication. ContractExecutionTarget, FinalizedContractRun, and ContractExecution are nonhead objects, never stages. `PARTIAL` and `CANCELLED` do not advance the head. Only the current head plus immediate predecessor object/head digests are retained in `head.json`; there is no archived head-body history, durable staleness/invalidation record, branching, deletion, garbage collection, retention, or administrative recovery.

The receiptable U6 storage authority is deliberately narrow: conforming Countershape writers on the natively exercised local Darwin filesystem. Darwin `flock`, same-directory hard-link create-if-absent publication, and atomic head rename protect cooperating writers. Direct same-UID mutation remains possible despite `0600` object modes. NFS/network/synchronizing filesystems, hostile same-UID races, forced power-loss recovery, and crash consistency beyond the exercised reopen checks are not established. `private-captures/` is a validated reserved `0700` directory; U6 does not yet expose a specialized captured-body publication API.

The store is explicitly not a command journal, fleet history, agent session, effects ledger, process-resume mechanism, fork/replay engine, or causal timeline. Future SSE may report best-effort in-flight progress to the local studio. A crash may leave already-finalized immutable objects or previously persisted nonadvancing evidence; U6 writes no crash-time `PARTIAL` or status object and cannot resume. A retry allocates an entirely new confirmation run.

## Future U8/U9 local API, studio, and report targets

The future local server will bind literal `127.0.0.1`. A high-entropy fragment token will be transferred to session storage and removed from the visible URL. Reads and writes will require bearer authorization. Writes will additionally require exact Host and Origin, JSON content type, CSRF, and the expected current digest for CAS. There will be no permissive CORS.

The future server will consume core-computed classifications, digests, transition availability, blind/reveal separation, and compile eligibility. The React/Vite client will render inert typed facts and send caller inputs. Candidate content must never become HTML, a URL, a class name, an accessible label without encoding, or executable script. Desktop and mobile must render the same trust-boundary facts; if evidence parity is absent, mobile becomes triage/defer-only.

The future report edge will create a default-minimized local evidence export. It must omit captured bodies, contain only previewed selected projection data, structurally escape candidate text, record minimization/redaction operations, and display `CONFIDENTIALITY NOT ESTABLISHED`. `--include-raw` is an explicit dangerous future path and must not rename transformed bytes as raw.

## Wake and didrun boundaries

Wake may create immutable candidate refs and provide opaque inert provenance. Countershape does not consume Wake events, sessions, epochs, effects, gates, timelines, replay, fork, recovery, or scheduling. Countershape's process owner exists for one short behavior trial and does not supervise coding agents.

didrun references are opaque strings associated with their exact external command and Git state. Countershape round-trips unknown grades verbatim. It never parses a grade into `CONFORMS`, aggregates several grades into a stronger label, or assigns a receipt to a capability that was not run. Missing evidence is `UNRECEIPTED`.

## U0–U9 architectural boundaries

| Unit | Architectural addition | Claim ceiling at exit |
| --- | --- | --- |
| U0 | controlling documents, schemas, examples, vectors, planning validator | internally checked planning contracts only |
| U1 | canon/domain/spec/compare/choice truth kernel, properties, state models, required mutants; no subprocess | truth-kernel prototype on the exact tested tree |
| U2 | Git-object materializer and fresh Darwin process substrate | supported-source and native process-substrate observations only |
| U3 | typed CLI observation, rotation, classification, exact map | CLI observation spine |
| U4 | typed HTTP observation and fixture-owned readiness | two typed observation spines if generic boundaries remain clean |
| U5 | typed neighbors, tri-valued orchestration, transcript, shape trap, minimal content-addressed durable-sweep authority | local reduction grades under named rules and budgets |
| U6a / P06 | fixed linear artifact store/CAS, physical confirmation, Choicepoint, blind semantic kernel, and DecisionRecord/ruling promotion through `RULING` | may claim only the exact source/store boundary mapped to P06 receipts; no public residue transition |
| U6b / P07A | unchanged historical projection wires, strict adapter-bound portable interpretation, selected-field Choicepoint/ruling, and legacy nonportable preservation | selected-field semantic authority only; no runnable source or residue claim |
| U6c / P07B-A1 | byte-complete self-consistent source reconstruction and portable HTTP child-bind Observation substrate | sealed source/process substrate only; no current-ruling join, compiler, confirmation/ruling under the new lineage, residue, or execution claim |
| U6c / P07B-A2.1 | current-revalidated portable-ruling snapshot, exact source/Choicepoint/confirmation/proof join, pure exact predicate/profile model, and sealed authority-narrowed compilation input | current-ruling compilation preparation only; no generated files, compiler output, store writes, residue, materialization, or execution |
| U6c / P07B-A2.2 | pure recoverable six-file compiler, strict bundle parser, fixed Node-core runtime assets, and Go/Node semantic corpus | sealed compiler/source boundary only; no residue, product materializer, target, or current execution |
| U6c / P07B-B | opaque node-issued terminal publication, complete predecessor restart, and retryable exact native six-file materialization | current working-tree candidate; `UNRECEIPTED`; no target-inventory, runtime, run, classification, or product claim |
| U6c / P07B-C | capability-only execution target, admitted runtime, finalized run, target-inventory absence, and immutable nonhead current execution | remains future until its independent runtime, absence, lifecycle, and classification gates pass |
| U7 | complete CLI and both reference studies, repeated clean runs and timing | narrowed Darwin reference studies only |
| U8 | authenticated decision bench, blind-first flow, visual/accessibility loops | studio-complete only if every security and evidence-parity gate passes |
| U9 | minimized export, packaging, adversarial hardening, final evidence and handoff | only the capabilities mapped to verbatim receipts |

No unit inherits a future unit's claim. Honest fallback milestones are truth-kernel prototype, observation-only instrument, one-domain instrument, DecisionRecord-only residue, and CLI-only surface.

## Fitness checks for later units

Later implementation review must reject any change that:

- gives the browser canonical or state-transition authority;
- allows an adapter to construct `CandidateOutcomeMap` without assessment, centralized eligibility, shared admission-set, and global evidence-identity gates;
- represents reduction as a boolean or compares group shape;
- uses checkout, worktree, archive, replacement objects, lazy fetch, or network to materialize evidence;
- accepts source modes outside `100644` and `100755`;
- reuses executed observations for confirmation or a final sweep;
- lets a noncompilable action reach the emitter;
- lets a legacy whole-projection ruling or caller-paired tuple/profile reach the emitter;
- compiles without a byte-complete source preimage that reconstructs the exact plan and minimized stimulus;
- allows a stale head to finalize a decision;
- creates a final output directory before terminal residue publication;
- routes a single later target through the 2–4-candidate `WorldInstance` chain, accepts a parsed target/copy of its digests as authority, or executes dirty working-tree bytes;
- admits a repository executable outside the antecedent-backed logical-Node-plus-repository-script profile;
- spawns before the exact ContractExecutionTarget is durably published and reopened;
- pairs a FinalizedContractRun with any target or attempt other than the exact one it names;
- lets ContractExecution duplicate target/runtime/lifecycle/observation authority or become a study-head transition;
- treats an emitted bundle as current conformance;
- adds event-runtime, replay, recovery, agent, ranking, or merge semantics; or
- upgrades an external receipt or an unexecuted environment claim.

<!-- countershape-validator: allow-prohibited-terms begin -->
## Explicitly rejected architecture descriptions

The public and internal architecture must not describe Countershape as offering “compatible worlds,” finding the “smallest behavior,” selecting a “winner” or “best branch,” supporting a “safely shareable” report, accepting symlink support in this run, or using execution-evidence cache/reuse. Those are prohibited upgrades, not roadmap shorthand.
<!-- countershape-validator: allow-prohibited-terms end -->
