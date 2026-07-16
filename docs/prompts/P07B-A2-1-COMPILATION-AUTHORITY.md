# P07B-A2.1 — current-ruling compilation authority

## Role

You are implementing one narrow authority seam in Countershape. Work as a rigorous Go systems engineer who treats canonical compatibility, current-store authority, negative construction paths, mutation closure, and developer ergonomics as load-bearing.

This is not a compiler prompt. It produces no generated files, bundle, residue, output directory, target, process run, UI, or report.

## Required starting state

The repository must contain and locally verify the sealed P07B-A1 source boundary:

- source commit `1e56da3bfafc4cdb8abdca62f18fb3b1accea8c3`;
- tree `da0e13f37aacd0e7fb2b52b3aeac628f5558c11e`;
- strict didrun result `15/15 claims recorded-exact`;
- every final A1 claim verbatim `TREE-EXACT`.

The later documentation receipt may be a descendant. Do not rewrite, amend, reseal, or reinterpret A1 history.

Before editing, read in this order:

1. `docs/CONCEPT_BRIEF.md`
2. `docs/SEMANTICS.md`
3. `docs/PROJECTION_ALGEBRA.md`
4. `docs/ARCHITECTURE.md`
5. `docs/STATE_MACHINES.md`
6. `docs/THREAT_MODEL.md`
7. `docs/status/P07B-A1-SOURCE.md`
8. `docs/prompts/P07B-A2-COMPILATION-PLAN.md`
9. `research/deep-dive/11-p07-implementation-red-team.md`
10. the relevant current Go packages and their tests.

Run `git status --short --branch` before edits. Preserve unrelated/user changes. Root is the sole writer and sole Git/didrun operator. Read-only agents may inspect in parallel; they may not edit, run Git, or write the didrun ledger.

## Objective

Create one opaque `PreparedCompilation` only when all of these agree exactly:

- the complete current store-bound portable ruling capability;
- its reparsed DecisionRecord;
- its exact Choicepoint;
- its exact FreshConfirmation;
- one valid reparsed A1 `PortableSource`;
- every schedule-ordered confirmation execution-binding digest;
- the exact source-matched projection profile;
- independently reopened and translated confirmation proofs;
- the allowed/disallowed selected-tuple partition; and
- the selected-field separation obligation.

The returned capability must privately retain what later publication needs and keep an inert, defensive sanitized compiler input hidden inside the node package. That input is free of concrete candidate keys, refs, aliases, producer metadata, and support counts; its exact PortableSource necessarily retains opaque source/lineage identity digests, including candidate-set, plan, envelope, materialization, fixture/capture, and projection identities. No caller may construct the input from copied digests, JSON, public fields, or a validity Boolean.

## Locked authority model

```text
current promotion.Ruling capability
  + exact DecisionRecord / Choicepoint / FreshConfirmation
  + exact A1 PortableSource
  + source-matched proof retranslation
  + ruling partition and separation revalidation
       |
       v
sealed PreparedCompilation
  - privately retains original ruling preparation
  - publicly exposes none of that authority
  - owns private compilation.Input with no concrete candidate identity/provenance
  - no generated files
  - no store publication
  - no durable currentness claim after return
```

`PreparedCompilation.Valid()` may prove construction integrity only. It must not claim that the ruling remains current after the operation returns. P07B-B will revalidate the retained original preparation inside its owner-controlled publication transition.

## Owned implementation surface

Inspect current names and preserve package ownership. The expected minimal change surface is:

```text
internal/choice/                     defensive Choicepoint accessors if absent
internal/world/                      defensive FreshExecution binding accessor
internal/confirmation/               retained schedule-ordered binding roster
internal/choice/promotion/service.go current-validated portable compilation snapshot
internal/emit/node/model/            closed pure predicate/source-profile values
internal/emit/node/internal/compilation/ sealed sanitized input
internal/emit/node/                  PrepareCompilation application service
tools/check-p07b-a2-architecture.mjs
tools/check-p07b-a2-architecture-selftest.mjs
tools/mutate-p07b-a2-authority.mjs
```

If an existing owner already has a more precise file, extend it instead of duplicating truth. Do not move private canonical parsing into an application package.

## Required authority seam changes

Add only defensive inert inspection needed to avoid parsing another package's private JSON:

```go
func (r ChoicepointRecord) WorldPlan() domain.WorldPlan
func (r ChoicepointRecord) MinimizedStimulus() CanonicalArtifact

func (r FreshExecutionRecord) ExecutionBindingDigest() domain.Digest
func (r Record) ExecutionBindingDigests() []domain.Digest
```

Use the actual current type names after inspection. Getter results must be defensive. Do not change any already sealed canonical member roster, ordering, bytes, digest, fixture, or member count.

`confirmation.Record` must retain the exact schedule-ordered execution-binding digest roster that it already validates from embedded physical facts. Parsing and construction must reconstruct the same roster without changing its 38-member canonical body. Repeated binding digests are expected across trials and must remain in order; missing, reordered, incomplete, zero, or foreign values are refused under existing authority rules.

Prefer one private `physicalWireSummary` result from physical-wire validation over adding another positional return to its existing parallel slices. Update parse, clone, and `Valid()` together so an in-memory record with an absent or misaligned roster is invalid while every strict historical parse reconstructs it. Add no field to the canonical identity.

Add a current-validated snapshot owned by promotion, conceptually:

```go
type PortableCompilationSnapshot struct { /* private */ }

func OpenPortableCompilationSnapshot(
    ctx context.Context,
    objectStore *store.ObjectStore,
    preparation PortableRulingPreparation,
) (PortableCompilationSnapshot, error)
```

The operation must:

1. revalidate the complete current ruling capability;
2. reopen and strictly reparse the DecisionRecord, Choicepoint, and FreshConfirmation;
3. recheck their exact durable joins;
4. return only defensive inert records; and
5. expose no `HeadToken` or durable-currentness claim.

Keep this snapshot in the existing promotion service unless the inherited U6 architecture contract is deliberately updated: it freezes the `PortableRulingPreparation` reference surface. Avoid adding another portable-ruling inspector call when the retained preparation already supplies the owned reopen path.

Wrong store, foreign preparation, corruption, and illegal lineage must remain distinct where existing typed errors distinguish them. Preserve a `STALE_CHOICEPOINT` mapping for future legal supersession, but do not claim a physical A2.1 stale-loser case while `RULING` has no successor transition. Legacy and noncompilable actions are refused at the existing portable-preparation boundary.

## Closed pure predicate model

Create construction-safe pure values under `internal/emit/node/model`:

- `ExactValue`
- `ExactField`
- `ExactTuple`
- `Predicate`
- `SourceProfile`

Use the existing `portablevalue.Value` algebra as the semantic core. Reconstruct through its tag-specific public constructors so byte/list storage is copied, ceilings and canonical rules stay single-owned, and the emitter does not invent a parallel value language.

The exact-value wire must agree with the closed portable algebra and `common.schema.json`:

- `MISSING` and `NULL`: tag only;
- `STRING`: exact valid UTF-8 value;
- `INTEGER`: canonical safe-integer spelling, with no float, exponent, plus, leading zero, or `-0` alias;
- `BOOLEAN`: exact Boolean;
- `BYTES`: canonical padded base64;
- `ORDERED_STRING_LIST`: ordered values preserving duplicates, empty list, and empty members;
- `CANONICAL_JSON`: exact strict canonical bytes encoded as canonical padded base64.

No tuple accepts a caller-supplied tuple digest. No model accepts a generic map that loses profile order. Missing and present-empty remain distinct.

The closed predicate version is `one-of-exact/v1`. It carries the exact ordered selected fields and a canonical set of complete selected-field tuples. For each valid tuple, compute the frozen canonical `ExactTuple` body bytes with no caller-supplied digest, deduplicate only byte-identical bodies, and sort the set by unsigned lexicographic comparison of those canonical UTF-8 bytes. Construction order is never observable. It never represents per-field allowed values and never forms a Cartesian product.

Keep two different profile authorities explicitly separate:

- `PortableSource.Profile()` is the P07A portable **projection profile** used to translate and order selected fields. Its canonical bytes/digest must equal the reopened prepared ruling's portable projection profile.
- `model.SourceProfile` is the runtime-generic **declared emitter profile**. Freeze exactly seven members: `runtime_family = NODE`, `semantic_profile = countershape-node-core-exact/v1`, source-derived `adapter_domain`, `launch_profile = NODE_REPO_SCRIPT_V1`, exact repository-relative `subject_entrypoint`, source-derived `start_profile`, and `scope = DECLARED_SOURCE_PROFILE_NOT_EXECUTION_EVIDENCE`. CLI pairs only with `DIRECT_CHILD_V1`; HTTP pairs only with `NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1`.

Construct `model.SourceProfile` only from the exact A1 source's adapter, logical-Node runner, entrypoint, and start authorities. It declares the generated semantic/runtime family but contains no host Node version, executable path/digest, OS, architecture, or execution receipt. Do not compare its bytes to the distinct P07A projection profile.

Freeze its typed identity as
`sha256("countershape/v1/ContractSourceProfile\x00" || exact_canonical_seven_member_source_profile_body_bytes)`.
The canonical body has no terminal LF. Construct the body first and recompute this
digest internally; accept neither caller-supplied profile bytes nor a paired
digest. A same-body digest under `SourceProfile`, `PortableProfile`, or any other
domain is unequal and refused. A2.1 tests recomputation and defensive-copy
stability; A2.2 codec parity and later target binding must use this exact domain.

## Sanitized compilation input

Create a sealed internal input that contains only:

- DecisionRecord digest;
- Choicepoint digest;
- decision action;
- exact `PortableSource`;
- exact derived declared emitter `model.SourceProfile`;
- selected fields in profile order; and
- exact canonical allowed tuple set.

“Sanitized” means authority-narrowed, not content-redacted. The exact PortableSource and selected predicate values remain recoverable and may contain sensitive, host-looking, or candidate-looking behavior bytes; this input establishes no confidentiality. Apart from that authorized content and the frozen declared emitter profile, it must add no:

- candidate key, alias, ref, branch, commit, tree, producer, model, source order, or support count;
- confirmation proof bytes or receipt;
- duplicated or host-derived process/runtime evidence, PID, allocated port, host root or absolute path, clock, platform measurement, or execution result; the exact source may still carry its declared logical runner, repository-relative entrypoint, and start/readiness/capture authorities;
- didrun event, grade, claim, manifest, or interpretation;
- target, finalized run, execution classification, materialization path, or head token; or
- outward public generic constructor from wire data.

The value must be defensively copied and revalidated. A constructor exported only inside Go's `internal/emit/node` subtree is acceptable when the architecture gate freezes exactly one call site in the application service. A valid serialized DecisionRecord or copied digest collection cannot construct it.

## Application service

Implement the conceptual operation using current package naming:

```go
func PrepareCompilation(
    ctx context.Context,
    objectStore *store.ObjectStore,
    preparation promotion.PortableRulingPreparation,
    source contractsource.PortableSource,
) (PreparedCompilation, error)
```

`PreparedCompilation` privately retains the original preparation for P07B-B and the sanitized input for A2.2. It may expose a digest and safe inert summaries, but no public method may return the internal compiler input, expose concrete candidate identity, or create publication authority. A later same-package A2.2 service consumes the private input.

## Exact preparation algorithm

Perform these checks in this order and return no partial capability on error:

1. Open the current-validated promotion snapshot.
2. Strictly parse `source.CanonicalBytes()` and require an exact byte/digest match with the supplied source.
3. Require source and Choicepoint equality for:
   - exact WorldPlan canonical bytes and digest;
   - the adapter domain and projection-binding canonical bytes/digest already contained in that exact WorldPlan; and
   - minimized stimulus kind, canonical bytes, and digest.
4. Require the source execution-binding digest to equal every schedule-ordered execution-binding digest retained by FreshConfirmation.
5. Rebuild the confirmed projection roster from the strict confirmation map.
6. Convert retained confirmation proof records through their owning package into `projectiontranslate.ProjectionProof` values.
7. Run the existing proof-first `TranslateConfirmed` under the exact source-matched binding.
8. Require the translated portable projection profile canonical bytes and digest to equal both `source.Profile()` and the prepared ruling's portable projection profile. This is the first full profile-byte join; a preparation profile digest may be used only as an earlier cheap precheck. Separately derive and revalidate the seven-member emitter `model.SourceProfile` from the exact source; never compare these two profile types as if they shared a wire. The step-4 full execution-binding equality is what binds the source's logical runner, repository-relative entrypoint, and closed start/readiness/capture authorities to every confirmation execution—not an invented Choicepoint accessor.
9. Revalidate the ruling partition:
   - action is only `ALLOW_OBSERVED` or `CUSTOM_EXPECTATION`;
   - selected fields are nonempty, unique, and exactly in resolved profile order;
   - allowed and disallowed outcome references partition the translated confirmed roster;
   - their selected tuple sets equal the corresponding translated projections;
   - allow-many remains a set of complete correlated tuples;
   - tuple bodies are byte-deduplicated and unsigned-lexicographically sorted by exact canonical `ExactTuple` bytes, independent of confirmation or construction order;
   - `CUSTOM_EXPECTATION` retains exactly one reviewed selected-only tuple;
   - no confirmed candidate conforms to a custom expectation; and
   - every allowed/disallowed tuple pair differs in at least one selected field.
10. Discard every concrete candidate and outcome identity outside the exact source's opaque source/lineage identity digests.
11. Construct and independently revalidate the sanitized input.
12. Return the sealed preparation with no store write or external side effect.

The explicit application-service proof retranslation is required even if strict Choicepoint parsing also translates. If deletion of that defense is observationally equivalent for every constructible current record, enforce the defense through architecture anchors rather than pretending an equivalent mutant is behavioral.

## No-side-effect contract

Outside the owning store's required current-object reopen, production A2.1 code may not perform:

- direct filesystem reads/writes, path materialization, or filesystem-package imports;
- process execution;
- Git invocation or repository inspection;
- network or loopback access;
- clock reads, sleeps, randomness, environment reads, or runtime/platform inspection;
- object publication, head mutation, temporary creation, or materializer work; or
- generated source assembly.

Store reads required to reopen current authority belong only to the application/promotion edge and remain subject to the store's existing strict parser/reopen rules. The service may call that authority but may not perform direct path I/O. The pure model and sanitized input closure must not import store, world/process, confirmation, promotion, projection translation, Git, filesystem, network, runtime, clock, or randomness packages.

## Positive test matrix

At minimum prove:

1. current CLI `ALLOW_OBSERVED`;
2. CLI allow-many with correlated tuples that would fail under a Cartesian product;
3. portable child-bind HTTP `ALLOW_OBSERVED` after building the required fresh confirmation/Choicepoint/ruling lineage in test fixtures;
4. HTTP `CUSTOM_EXPECTATION`, such as selected status `401`, with no confirmed candidate conforming;
5. process restart/reopen returns byte-identical sanitized input;
6. defensive getter mutation cannot change the preparation;
7. absent and present-empty values remain different; and
8. legacy P07A fixture bytes and digests remain byte-identical.

## Negative test matrix

At minimum refuse:

- invalid, foreign, or wrong-store preparation;
- legacy whole-projection ruling at the existing portable-preparation refusal boundary;
- `REJECT_ALL`, `DEFER`, or `REFINE` at their existing upstream promotion refusal boundaries;
- a foreign source whose exact plan, binding, profile, stimulus, runner/start, or execution bytes differ even when adapter/profile shape matches; an independently reconstructed byte-identical source is the same canonical authority and must be accepted regardless of construction provenance;
- plan, adapter, projection binding, profile, stimulus, runner, entrypoint, start, readiness, or capture mismatch;
- inherited-listener HTTP source;
- missing or changed exact source payload bytes;
- zero, incomplete, reordered, or differing confirmation binding roster while preserving expected repeated digests in schedule order;
- proof-roster, proof-byte, translator, or profile mismatch;
- selected-field reorder, omission, duplication, or widening;
- allowed/disallowed reference or tuple partition disagreement;
- independent per-field values or Cartesian-product authorization;
- custom tuple with an unselected field;
- custom tuple matching any confirmed selected tuple;
- any construction error returning a nonzero sanitized capability; and
- any error creating an object, temp path, generated file, or head mutation.

Cross-pair fixtures must combine individually valid source, preparation, Choicepoint, confirmation, profile, and proof objects from different studies. Shape-invalid-only cases are insufficient.

Some defenses are intentionally redundant behind sealed upstream constructors. A legally superseded A2.1 preparation is not currently constructible because `RULING` has no successor transition; its stale-loser receipt belongs in P07B-B. If proof/profile drift, selected-field reorder, partition tampering, or a noncompilable action cannot be constructed through the current public capability graph, do not add a backdoor test constructor or fake a behavioral mutant. Prove the upstream refusal in its owning package, retain the downstream check, and use an architecture anchor/hostile-copy self-test to prevent deletion. Record the constructibility limit explicitly in the status ledger.

## Physical integration map

Use existing study infrastructure rather than adding a second runner:

- In the HTTP invoices physical reduction path, enable its existing `PortableStart` configuration so baseline, reduction, confirmation, portable Choicepoint, and ruling use the real A1 child-bind lineage. Build the exact minimized source from that confirmed study and profile. Exercise custom `401`, wrong-store, original/noisy-stimulus mismatch, retained binding equality, and restart-equivalent preparation. A separate allow-observed ruling may use a second store study because one current head finalizes only once.
- In the CLI precedence physical reduction path, make the strongest current ruling an allow-many correlated predicate over `cli.stdout.json.mode` and `cli.stdout.json.source`, for example `(config,config)` and `(argv,argv)`. This must kill cross-product and keep-first-tuple faults while leaving `cli.stdout.bytes` unasserted. Exercise exact minimized source, wrong-store, foreign/original-stimulus mismatch, restart equivalence, and defensive getters.
- In confirmation wire tests, parse an existing sealed fixture and assert roster cardinality/order/repeated values/defensive copying without changing its bytes or digest.
- In Choicepoint/session round-trip tests, assert both legacy and fresh records expose byte-identical WorldPlan/minimized-stimulus values and returned-byte mutation cannot alter the record.

## Compatibility and restart gates

Before claiming this unit:

- compare all sealed legacy Choicepoint/DecisionRecord/confirmation fixtures byte-for-byte and digest-for-digest;
- prove strict parsing reconstructs the new in-memory binding roster without changing canonical bodies;
- reopen the store in a fresh process and obtain the same sanitized body;
- prove public APIs do not expose a constructor that can mint `PreparedCompilation` or sanitized input from copied wire values; and
- keep A1 source canonical bytes/digests unchanged.

Any required canonical change to a sealed historical body is a stop condition. Do not regenerate history to make the test pass.

## Architecture gate

Extend a layered A2 checker that first runs the sealed A1 checker. It must close transitive production dependencies, reject forbidden imports through aliases or helper packages, and detect source-spoofing in comments/strings.

The pure model/input closure may depend only on canonical/domain/portable-value/profile/source pure authorities needed for construction. The root preparation service may depend on store/promotion/confirmation/choice/translation, but not on filesystem, process, Git, clock, randomness, environment, runtime, or network packages.

Require:

- no public generic input constructor;
- no concrete candidate/provenance/support fields, confirmation receipts, or injected host-runtime evidence in sanitized input outside the exact PortableSource and frozen declared source profile;
- no Wake runtime/artifact integration;
- no didrun grade interpretation;
- no generated file roster or bundle type in A2.1; and
- no store write or head-transition call.

The hostile-copy self-test must demonstrate that the checker catches transitive forbidden dependencies, constructor exposure, comment/string spoofing, candidate field insertion, store-write insertion, and removal of the explicit retranslation anchor.

## Mutation gate

Freeze only non-equivalent mutants with named killers and fresh A/B/A restoration. Candidate behavioral mutants include:

1. admit a foreign mismatching source by dropping the source/ruling join;
2. compare only source digests while ignoring exact canonical bytes where a valid collision-independent mismatch is constructible;
3. retain candidate keys in sanitized input;
4. treat every translated confirmed tuple as allowed;
5. retain only the first allowed tuple;
6. synthesize a Cartesian product;
7. sort selected fields lexically instead of profile order;
8. append an unselected field;
9. collapse `MISSING` into empty string;
10. convert opaque bytes through UTF-8;
11. sort or deduplicate an ordered string list;
12. parse/stringify canonical JSON instead of retaining exact bytes;
13. treat `CUSTOM_EXPECTATION` as `ALLOW_OBSERVED`;
14. accept a mismatched confirmation execution-binding roster;
15. preserve tuple insertion/confirmation order instead of canonical-byte order; and
16. compare tuple bodies with UTF-16 or locale collation instead of unsigned bytes.

Do not pad the roster with noncompiling, equivalent, or architecture-only mutants. Architecture requirements belong in the architecture gate unless a constructible behavioral test distinguishes them.

The mutation tool must self-test its own mutation application, named killer mapping, fresh-copy containment, digest restoration, and tamper detection. Retained copies are not hostile-code containment; trusted tests retain host authority.

## didrun and commit discipline

Every load-bearing command runs through `didrun run --`. Keep resource pressure bounded: serialize didrun, use at most two test workers where configurable, and use one worker for fuzz/mutation loops unless a measured reason justifies more.

Required final gates include:

- exact staged inventory and diff check;
- Go formatting check;
- focused A2.1 tests;
- full `go test ./...`;
- full `go vet ./...`;
- selected/full race coverage appropriate to runtime;
- architecture checker and hostile self-test;
- mutation harness self-test and final mutant closure;
- source/ruling cross-pair negative matrix;
- restart/reopen compatibility;
- bounded active fuzzing for the new parser/construction seams; and
- scoped staged structured credential-pattern scan.

Immediately after each final successful command, declare the exact truthful claim before recording another event. Do not address an older event explicitly. Development failures remain permanent history and support no capability.

At the boundary:

1. finish and immediately claim the non-staged final gates;
2. stage only the exact A2.1 files;
3. run the exact staged inventory/diff check through didrun and immediately claim it;
4. run the scoped structured credential-pattern scan over the exact staged blobs through didrun and immediately claim only that named scope;
5. inspect the staged diff, private artifacts, and obvious credential exposure;
6. commit without an AI co-author trailer;
7. run `NO_COLOR=1 didrun seal --commit HEAD`;
8. if the seal reports aggregate high entropy, inspect exact staged scope and structured patterns before deciding whether the logged redacted override is justified;
9. run `NO_COLOR=1 didrun verify --strict`;
10. on any nonzero result, fix the real issue, rerun affected verification through didrun, immediately re-claim, create a new commit, reseal, and repeat until exit `0`; and
11. archive the exact ledger under a named `.didrun-history/` boundary without committing it.

Never weaken a test, threshold, mutant, claim, receipt, or old commit to open the gate.

## Honest claim vocabulary

Permitted claims, only when directly receipted, are narrow:

- current-ruling snapshot revalidation passed the named matrix;
- exact PortableSource/ruling/confirmation/proof join passed the named matrix;
- private `compilation.Input` excluded concrete candidate identity/provenance, confirmation receipts, and injected host-runtime facts outside its exact PortableSource and frozen declared source profile under the named architecture and mutation gates, while the wrapper retained original authority without public exposure;
- legacy canonical compatibility passed the named fixtures;
- exact test/race/vet/fuzz/mutation commands succeeded on the named tree.

Do not claim compiler output, source emission, residue, materialization, runtime portability, target absence, network denial, sandboxing, confidentiality, authenticity, security review, production readiness, or market value.

## Stop conditions

Stop and write an honest handoff without starting A2.2 if:

- any exact join requires parsing another package's private canonical JSON instead of using an owning accessor;
- the confirmation execution-binding roster cannot be reconstructed without changing sealed canonical bytes;
- a current-ruling snapshot cannot distinguish stale/foreign/corrupt authority under existing store semantics;
- the private compiler input adds concrete candidate identity/provenance, confirmation receipts, or host-runtime facts outside its exact PortableSource and frozen declared source profile, or the outer wrapper publicly exposes its retained authority or publication power;
- declared emitter profile identity uses any domain other than `ContractSourceProfile`, hashes a noncanonical or LF-suffixed body, or trusts a caller-paired digest;
- the architecture gate cannot prove the intended dependency closure;
- a required non-equivalent mutant survives;
- any load-bearing verification remains nonzero; or
- strict didrun verification does not exit `0`.

## Done condition

A2.1 is done only when one local commit contains the narrow authority seam, its status/handoff documentation states exact nonclaims, its final claims have verbatim grades, its ledger chain is intact, and `NO_COLOR=1 didrun verify --strict` exits `0`.

Only then may a fresh context execute `P07B-A2-2-RECOVERABLE-COMPILER.md`.
