# Countershape architecture contract

- **Contract version:** U0 / `architecture-v1`
- **Target:** narrowed Darwin reference instrument
- **Status:** controlling design; U1 command receipts are enumerated without reinterpretation in `status/U1.md`, while later runtime and product capabilities remain `UNRECEIPTED`
- **Authority order:** `CONCEPT_BRIEF.md`, `SEMANTICS.md`, `PROJECTION_ALGEBRA.md`, then the stricter rulings in `research/deep-dive/08-RED_TEAM.md`, then this document

Countershape is a repository-scale operational composition for resolving one witnessed behavioral disagreement among exact repository candidates. It is not an agent runtime, candidate ranker, generalized workflow runner, correctness oracle, or new disambiguation algorithm. The reference instrument accepts trusted local code from a curated Git repository, performs finite fresh executions, compares exact projected bytes, and preserves a human-authored selected-field decision as standalone Node source.

This document fixes the component boundaries, dependency direction, truth ownership, persistence model, and U0–U9 growth path. Later units may narrow a capability when a gate fails. They may not merge truth jurisdictions or silently promote a weaker artifact to a stronger claim.

## Architectural invariants

1. **Different questions have different authorities.** Git selection, materialized bytes, declared execution, measured execution, captured evidence, projected values, observed disagreement, human intent, emitted source, current conformance, and external command receipts are separate objects.
2. **A comparison envelope is a declaration, not a proof.** It records which measured dimensions were required equal, tolerated, rejected, or left uncontrolled. Admission never establishes that tolerated placeholders were irrelevant to candidate control flow.
3. **Only exact canonical values compare.** V1 has no tolerance, similarity, wildcard, regex, inferred invariant, or order-dependent cluster identity.
4. **Comparability precedes labeled-map equality.** Reduction, confirmation, persistence, API, and studio first require the same plan, envelope, stimulus-independent comparison basis, expected roster, eligible set, exclusions, and exclusion classifications. Only then does equality of the canonical sorted `candidate_execution_key -> projection_fingerprint` map digest mean preservation. Display groups and naked digests carry no preservation authority.
5. **Control failure is not program behavior.** Only eligible finalized attempts can enter projection and a stable batch. Explicit setup, start, readiness, transport, timeout, output, projection, orphan, teardown, or cancellation failures remain typed controls.
6. **Every evidentiary execution is new.** Discovery, each reduction proposal, final sweep, confirmation, and current conformance allocate new attempt artifacts, private roots, and process lifecycles. Only pure transformations of immutable bytes may be memoized.
7. **Human scope is exact and narrow.** A compilable decision covers one witnessed stimulus and an explicit nonempty set of selected typed fields. Nonasserted fields remain visible and intentionally unconstrained.
8. **Standalone is a separate implementation.** The generated Node harness earns no parity claim until the same normative vectors pass in Go and Node and the bundle runs with Countershape absent.
9. **Persistence stores semantic artifacts, not runtime history.** Immutable content-addressed objects and atomic compare-and-swap heads are the durable model. Progress streams are disposable and cannot establish evidence.
10. **Failure narrows the product.** Unsupported source forms, uncontrolled variance, incomplete trials, unstable candidates, ambiguous selected fields, stale decisions, and unavailable runtimes refuse the stronger transition.

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
| What was locally reduced? | `ReductionRun` | typed neighbors, decreasing measure, fresh batches, transcript, budgets, complete sweep if present | root cause, global minimum, or comprehension |
| What is ready for a human decision? | `Choicepoint` | fresh-confirmed lineage and all governing artifact digests | a recommendation or branch endorsement |
| What did the human state? | `DecisionRecord` | exact Choicepoint, blind/reveal facts, action, selected and nonasserted fields | intent outside the witness or security approval |
| What source bytes were emitted? | `ContractBundle` | deterministic six-file set and bundle digest | authorship, current execution, or long-term fidelity |
| What happened against the current tree? | `ContractExecution` | current Git identity, fresh execution, bundle digest, eligible or ineligible result | recreation of historical evidence or overall correctness |
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
internal/reduce/           bounded tri-valued orchestration and complete sweep
internal/choice/           lineage, decision actions, field-separation validation
internal/emit/node/        deterministic six-file standalone source emitter
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

The HTTP adapter owns fixture-owned readiness, method/path, ordered query and header multimaps, request body, transport versus response, status, bounded response body, HTTP projection fields, and HTTP neighbors.

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

Countershape separates detected controls only. A declared setup program can exit zero after doing the wrong work, and a readiness mechanism can have hidden side effects. The two proof studies avoid setup and use fixture-owned readiness evidence.

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

The only reduction grades are `UNCHANGED`, `BEST_KNOWN`, and `ONE_MINIMAL_UNDER(reducer_set_digest)`. Construction of the last grade requires a durable-store authority over a complete final sweep whose enumerated valid direct-neighbor set equals its set of fresh `CHANGES` results. A cancelled sweep, exhausted budget, partial record, stale plan, changed eligibility, or any `UNRESOLVED` neighbor cannot construct that type. U1 deliberately exposes only logical neighbor/sweep evidence; the proof-bearing durable grade constructor does not exist until U5 adds the store-backed authority in the same package.

The minimized witness must pass a physically new confirmation batch with new attempt artifacts, invocation evidence outside the selected predicate, and a rotated schedule. A new nonce around copied evidence is invalid.

## Choicepoint and decision boundary

A `Choicepoint` is an immutable, decision-ready revision. It binds original and reduced stimuli, exact candidates, plan, envelope, capture and projection definitions, baseline and fresh-confirmed maps, exclusions, reduction transcript and grade, and evidence digests. Any semantic ancestor change creates a new lineage. Existing descendants become `STALE`; explicit replacement creates `INVALIDATED(reason,replacement)`.

The blind-first DTO contains scenario context, exact scope, envelope summary, observation counts, original/reduced stimuli and derivation, projection operations, deterministic outcome facts, distinct outcome count, and reduction evidence. It must omit candidate identity, producer/model provenance, total candidates, per-outcome support counts, candidate order, and any hidden DOM/accessibility metadata encoding those facts.

The server records a provisional action and explicit field decisions, then reveals provenance before finalization. Early reveal and post-reveal changes are facts, not proof of understanding. No field is preselected. Every differing projected field is explicitly `Assert` or `Context only`.

Compilable actions are `ALLOW_OBSERVED` and `CUSTOM_EXPECTATION`. For selected fields `F`, allowed tuples `A`, and disallowed confirmed tuples `D`, compilation requires:

```text
for every a in A and d in D: select_F(a) != select_F(d)
```

Missing and present-empty values remain distinct. Allow-many is a canonical set of complete selected-field tuples derived from a sealed confirmed-outcome set. It is never a cross-product. Empty selection returns `EMPTY_SELECTED_FIELDS`; failed separation returns `AMBIGUOUS_SCOPE`. Both create no files. `REJECT_ALL`, `DEFER`, and `REFINE` are noncompilable by construction; refine requests a successor study.

## Standalone residue and Go/Node parity

The deterministic output directory contains exactly:

```text
decision.json
fixture.json
contract.test.mjs
harness.mjs
README.md
manifest.json
```

The closed predicate profile is `one-of-exact/v1`. Bundle bytes contain no current time, random ID, absolute path, candidate/ref name, secret, Countershape import, package install, registry call, or external service. The HTTP contract may use local loopback only for its test subject.

`DecisionRecord`, `ContractBundle`, and `ContractExecution` are separate truth objects. Bundle hashing proves byte integrity, not authorship or conformance. A current eligible selected-tuple result is `CONFORMS` or `CONTRADICTS`; startup, readiness, timeout, capture, projection, output, or teardown failure is `INELIGIBLE_EXECUTION`, even though contradiction and ineligibility both cause an ordinary test failure.

One normative corpus must exercise Go and Node strict parsing, field selection, missing/empty distinctions, CLI and HTTP capture profiles, projection bytes, and eligible/ineligible errors. Mutations that accept duplicate names, use independent allow-many value sets, assert a context-only field, or collapse ineligibility into contradiction must fail in both authorities. Standalone remains `UNRECEIPTED` until this corpus and a physical absence run pass on the exact Node major and native OS named in the receipt.

## Content-addressed semantic artifact graph

The durable store has two concepts:

1. **Immutable objects:** canonical semantic artifacts addressed by domain-separated digest. A digest collision or mismatched kind/bytes is fatal. Objects are written to a private temporary file, fsynced as required by the platform policy, renamed atomically, reopened, and rehashed before publication.
2. **CAS heads:** small mutable selectors such as a study's current lineage. A write supplies the expected old digest and desired new digest. Missing or stale expectations fail without creating a semantic transition.

Graph edges are digest references inside immutable objects. A new revision never edits an ancestor. Partial and cancelled evidence may be retained as immutable nonadvancing objects. Staleness is derived from semantic ancestry and recorded in a successor status object; old bytes are never rewritten as fresh.

The store is explicitly not a command journal, fleet history, agent session, effects ledger, process-resume mechanism, fork/replay engine, or causal timeline. SSE may report best-effort in-flight progress to the local studio. A crash can preserve finalized immutable objects and an honest partial attempt. It cannot resume the process; the next evidentiary trial starts in a new world.

## Local API and studio boundary

The local server binds literal `127.0.0.1`. A high-entropy fragment token is transferred to session storage and removed from the visible URL. Reads and writes require bearer authorization. Writes additionally require exact Host and Origin, JSON content type, CSRF, and the expected current digest for CAS. There is no permissive CORS.

The server computes all classifications, digests, transition availability, blind/reveal separation, and compile eligibility. The React/Vite client renders inert typed facts and sends human inputs. Candidate content never becomes HTML, a URL, a class name, an accessible label without encoding, or executable script. Desktop and mobile render the same trust-boundary facts; if evidence parity is absent, mobile is triage/defer-only.

The report edge creates a default-minimized local evidence export. It omits captured bodies, contains only previewed selected projection data, structurally escapes candidate text, records minimization/redaction operations, and displays `CONFIDENTIALITY NOT ESTABLISHED`. `--include-raw` is an explicit dangerous path and does not rename transformed bytes as raw.

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
| U6 | artifact graph/CAS, confirmation, decision records, Go/Node corpus, deterministic residue | DecisionRecord only unless parity and absence gates pass |
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
- allows a stale head to finalize a decision;
- treats an emitted bundle as current conformance;
- adds event-runtime, replay, recovery, agent, ranking, or merge semantics; or
- upgrades an external receipt or an unexecuted environment claim.

<!-- countershape-validator: allow-prohibited-terms begin -->
## Explicitly rejected architecture descriptions

The public and internal architecture must not describe Countershape as offering “compatible worlds,” finding the “smallest behavior,” selecting a “winner” or “best branch,” supporting a “safely shareable” report, accepting symlink support in this run, or using execution-evidence cache/reuse. Those are prohibited upgrades, not roadmap shorthand.
<!-- countershape-validator: allow-prohibited-terms end -->
