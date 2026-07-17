# P07B-A2 plan — compilation authority and recoverable Node bundle

- **State:** locked implementation decomposition after sealed P07B-A1 source commit `1e56da3bfafc4cdb8abdca62f18fb3b1accea8c3`, tree `da0e13f37aacd0e7fb2b52b3aeac628f5558c11e`; A2.2 is additionally gated on sealed, strict-clean P07B-A2.1 and cumulative-verification maintenance boundaries
- **Scope:** two independently shippable source units; neither publishes `RESIDUE`, creates a product output directory, or constructs current execution evidence
- **Authority:** this plan narrows execution order without waiving any requirement in `P07-U6-STANDALONE-CONTRACT.md`

## Locked split

The cumulative-verification maintenance unit is an inserted nonfeature boundary between A2.1 and A2.2. A fresh A2.2 session must read `docs/VERIFICATION.md` and `docs/status/CUMULATIVE-VERIFICATION-MAINTENANCE.md`, confirm the maintenance didrun Git note plus strict exit `0`, and run `/opt/homebrew/bin/node tools/verify-current.mjs` before editing.

1. **A2.1 — compilation authority:** join the current store-bound portable ruling to the exact `PortableSource`, retranslate the retained confirmation proofs, and produce a sealed `PreparedCompilation` wrapper. It privately retains the original authority for P07B-B and separately owns a sanitized internal compiler input free of concrete candidate identity and provenance.
2. **A2.2 — recoverable compiler:** compile that capability into a deterministic, strictly recoverable six-file Node-core `ContractBundle` and prove Go/Node agreement over the checked-in semantic corpus.

Terminal publication and retryable materialization remain P07B-B. Pinned target, finalized run, classification, and physical absence/parity closure remain P07B-C.

## Why the split is load-bearing

The A1 tree leaves two deliberate implementation gaps:

- `promotion.PortableRulingPreparation` does not yet expose a current-validated snapshot containing the exact DecisionRecord, Choicepoint, and FreshConfirmation needed for the source/ruling join.
- `confirmation.Record` validates physical `FreshExecutionFact` bodies but discards their execution-binding digests, even though compilation must require the reconstructed source binding to equal every verified confirmation binding.

A2.1 repairs those authority seams without generating a bundle. A2.2's pure compiler consumes only the wrapper's private sanitized input; the outer wrapper continues to retain original authority for later publication revalidation.

```text
current promotion.Ruling
  + exact Choicepoint / FreshConfirmation
  + exact PortableSource
  + source-matched proof retranslation
       |
       v
sealed PreparedCompilation
  - privately retains original ruling preparation
  - publicly exposes none of that authority
  - owns sanitized compilation.Input:
      - no concrete candidate identity/provenance
      - no confirmation receipts or host/current/measured runtime facts outside authorized source content and the frozen declared emitter profile
  - no publication authority
       |
       v
pure deterministic compiler
       |
       v
inert recoverable ContractBundle
  - external typed digest
  - no self-digest
  - no head or store authority
```

Locked authority rules:

- `PreparedCompilation.Valid()` proves construction integrity only. It does not prove that the ruling remains current.
- A parsed `ContractBundle` is inert and cannot manufacture a `RESIDUE` capability.
- A2.1 preparation plus the A2.2 Go emission-time compiler/application path perform no store write, Git operation, filesystem materialization, process execution, clock read, randomness, or network access. The emitted fixed Node harness has only its separately frozen runtime authority to copy/overlay/clean a nonce-owned attempt root, generate its runtime attempt ID, spawn the explicit Node subject, and use literal loopback for the supported HTTP profile.
- P07B-B must revalidate the original preparation inside its owner-controlled transition before publishing any object.
- Concrete candidate keys, refs, aliases, producer metadata, support counts, confirmation receipts, and host/current/measured runtime or execution evidence remain inside the privately retained original preparation or may be used transiently during validation, but cannot become new structural authority in `compilation.Input` or the bundle. Authorized exact source and predicate content remains recoverable and may contain sensitive, host-looking, or candidate-looking bytes; the PortableSource also necessarily retains opaque source/lineage identity digests—including candidate-set, plan, envelope, materialization, fixture/capture, and projection identities—so do not misdescribe those as absent or redacted.

## A2.1 — compilation authority

### Existing authority seams

Add defensive inert inspection only:

```go
func (r ChoicepointRecord) WorldPlan() domain.WorldPlan
func (r ChoicepointRecord) MinimizedStimulus() CanonicalArtifact

func (r FreshExecutionRecord) ExecutionBindingDigest() domain.Digest
func (r Record) ExecutionBindingDigests() []domain.Digest
```

`confirmation.Record` retains the exact schedule-ordered binding-digest roster already validated from its embedded physical facts. This changes no historical canonical bytes or member counts.

Add a current-validated promotion snapshot:

```go
type PortableCompilationSnapshot struct { /* private */ }

func OpenPortableCompilationSnapshot(
    ctx context.Context,
    objectStore *store.ObjectStore,
    preparation PortableRulingPreparation,
) (PortableCompilationSnapshot, error)
```

The constructor must revalidate the complete current ruling capability, reparse the DecisionRecord/Choicepoint/FreshConfirmation, verify their exact durable joins again, and return only defensive inert records. It exposes no `HeadToken` and promises currentness only at construction.

### New emitter model

```text
internal/emit/node/
  service.go
  model/
    predicate.go
    source_profile.go
  internal/compilation/
    input.go
```

`model` owns closed pure values:

- `ExactValue`
- `ExactField`
- `ExactTuple`
- `Predicate`
- `SourceProfile`

The exact-value wire matches `common.schema.json`:

- `MISSING` and `NULL`: tag only;
- `STRING`: exact UTF-8 value;
- `INTEGER`: canonical safe-integer spelling;
- `BOOLEAN`: exact Boolean;
- `BYTES`: canonical padded base64;
- `ORDERED_STRING_LIST`: ordered values preserving duplicates and empties; and
- `CANONICAL_JSON`: exact canonical bytes as base64.

No tuple accepts a caller-supplied `tuple_digest`.

`internal/compilation.Input` is sealed and contains only DecisionRecord digest, Choicepoint digest, decision action, exact `PortableSource`, exact derived emitter `SourceProfile`, selected fields, and exact allowed tuple set. “Sanitized” means authority-narrowed, not content-redacted: exact source and predicate values may contain sensitive, host-looking, or candidate-looking behavior bytes. It adds no concrete candidate identity/provenance, confirmation receipt, or host runtime/execution evidence beyond authorized source content and opaque source/lineage identity digests. Complete tuple bodies are canonicalized through the frozen `ExactTuple` wire, deduplicated only by byte identity, and sorted by unsigned lexicographic canonical-byte order; locale, UTF-16, map iteration, confirmation order, and caller insertion order have no authority.

Do not conflate the two profile types. `source.Profile()` is the P07A portable projection profile used for retranslation and selected-field order. The emitter `SourceProfile` is a distinct exact seven-member declaration derived from A1: `runtime_family = NODE`, `semantic_profile = countershape-node-core-exact/v1`, adapter domain, `launch_profile = NODE_REPO_SCRIPT_V1`, exact repository-relative entrypoint, the closed adapter start profile, and `scope = DECLARED_SOURCE_PROFILE_NOT_EXECUTION_EVIDENCE`. CLI pairs only with `DIRECT_CHILD_V1`; HTTP pairs only with `NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1`. It contains no host Node/OS/architecture/path fact and receipts no runtime tuple. Its typed identity is exactly `sha256("countershape/v1/ContractSourceProfile\x00" || exact_canonical_seven_member_body_bytes)` with no LF; same bytes under any other domain, a noncanonical body, an LF suffix, or a caller-paired digest are refused.

```go
func PrepareCompilation(
    ctx context.Context,
    objectStore *store.ObjectStore,
    preparation promotion.PortableRulingPreparation,
    source contractsource.PortableSource,
) (PreparedCompilation, error)
```

`PreparedCompilation` privately retains the original preparation for P07B-B and the sanitized input for A2.2. Its public inspection is inert and defensive.

### Exact preparation algorithm

1. Open the current-validated promotion snapshot. Preserve corruption and wrong-store failures distinctly. Retain a typed `STALE_CHOICEPOINT` mapping for a future legally superseded ruling, but do not claim an A2.1 stale-loser test while `RULING` has no successor transition.
2. Require a valid, exactly reparsed `PortableSource`.
3. Require source/Choicepoint equality for exact WorldPlan bytes/digest—including its adapter domain and projection binding—and minimized-stimulus kind/bytes/digest. Do not invent a Choicepoint profile-byte or runner/start accessor.
4. Require the source execution-binding digest to equal every schedule-ordered digest retained by FreshConfirmation.
5. Rebuild the confirmed projection roster from the strict confirmed map.
6. Convert retained confirmation proofs to `projectiontranslate.ProjectionProof`.
7. Run `TranslateConfirmed` using the source-matched exact binding.
8. Require translated portable projection profile bytes/digest to equal `source.Profile()` and the prepared ruling's portable projection profile. This step owns the first full profile-byte join; an earlier preparation digest is only a cheap precheck. Separately construct/revalidate the distinct seven-member emitter `SourceProfile` from exact source authority. Step 4's full execution-binding equality, not a new Choicepoint accessor, binds source runner/entrypoint/start/readiness/capture authority to every confirmation execution.
9. Revalidate the ruling: only `ALLOW_OBSERVED` or `CUSTOM_EXPECTATION`; selected fields in resolved profile order; allowed/disallowed refs partition the translated roster; tuple sets equal the corresponding projections; exact tuple bodies are byte-deduplicated and unsigned-lexicographically sorted by their canonical `ExactTuple` bytes; custom expectation contains exactly one reviewed selected-only tuple and matches no confirmed candidate; every allowed/disallowed tuple pair remains separated.
10. Discard all concrete candidate and outcome identity outside the exact source's opaque plan/materialization digests.
11. Construct and revalidate the sealed sanitized input.
12. Return no partial value on any failure.

The explicit application-service retranslation remains required even though strict Choicepoint parsing also retranslates. A deletion-only mutant may be observationally equivalent for constructible records; enforce this defense through architecture anchors unless a genuinely distinguishing hostile fixture exists.

### A2.1 required tests

Positive cases:

- current CLI `ALLOW_OBSERVED`;
- CLI allow-many with correlated tuples;
- portable child-bind HTTP `ALLOW_OBSERVED`;
- HTTP `CUSTOM_EXPECTATION`, such as selected status `401`, with no confirmed candidate conforming;
- restart/reopen preserves the compilation-input digest plus every exposed deterministic identity, selected field, and exact tuple byte; the private sanitized input body remains unexposed, so direct cross-process body-byte equality is `UNRECEIPTED`; and
- defensive getter mutation cannot alter authority.

Negative cases:

- wrong store and invalid/foreign preparation; preserve the future stale mapping, but defer the physical stale-loser receipt to P07B-B where a legal `RULING` successor exists;
- a foreign source whose exact plan, binding, profile, stimulus, runner/start, or execution bytes differ; accept an independently reconstructed byte-identical source regardless of provenance;
- plan, adapter, binding, profile, stimulus, start-profile, or execution-binding-roster mismatch;
- inherited-listener HTTP source;
- missing source bytes;
- proof/profile retranslation mismatch where a sealed cross-pair is constructible;
- selected-field reorder or widening at its owning upstream refusal boundary plus a downstream architecture anchor;
- allowed/disallowed partition disagreement at its owning upstream refusal boundary plus downstream exact-set revalidation;
- per-field Cartesian-product authorization;
- custom expectation with an unselected field or one matching a confirmed candidate;
- legacy whole-projection ruling at the existing portable-preparation refusal;
- `REJECT_ALL`, `DEFER`, or `REFINE` at their existing upstream promotion refusals; and
- every error returns zero sanitized authority and creates no object, temp path, or output.

### A2.1 mutation direction

Freeze only non-equivalent mutants with named killers. Suitable behaviors include:

1. admit a foreign mismatching source by dropping the full source/ruling join;
2. retain candidate keys in sanitized input;
3. treat all translated confirmed tuples as allowed;
4. keep only the first allowed tuple;
5. synthesize a Cartesian product;
6. sort selected fields lexically instead of profile order;
7. append an unselected field;
8. collapse `MISSING` into empty string;
9. convert opaque bytes through UTF-8;
10. sort or deduplicate an ordered string list;
11. parse/stringify canonical JSON instead of retaining exact bytes;
12. treat `CUSTOM_EXPECTATION` as `ALLOW_OBSERVED`;
13. preserve tuple insertion/confirmation order; and
14. use UTF-16 or locale tuple ordering instead of unsigned canonical-byte order.

Collision-only byte-versus-digest mutants and redundant-check deletion mutants do not enter the behavioral roster unless a constructible test distinguishes them. Enforce those requirements in the architecture checker. Do not add backdoor constructors or noncompiling/equivalent mutants to simulate unreachable invalid sealed values.

## A2.2 — recoverable compiler

### Proposed layout

```text
internal/emit/node/
  compiler/
    compiler.go
    compiler_test.go
  model/
    bundle.go
    bundle_test.go
  program/v1/
    assets.go
    contract.test.mjs
    harness.mjs
  parity/
    evaluate.go
spec/vectors/v1/
  contract-parity.jsonl
testkit/contracts/
  fixtures.go
  node-vector-runner.mjs
  compiler_probe_test.go
tools/
  check-p07b-a2-architecture.mjs
  check-p07b-a2-architecture-selftest.mjs
  mutate-p07b-a2-authority.mjs
  mutate-p07b-a2-compiler.mjs
```

```go
func Compile(input compilation.Input) (model.ContractBundle, error)
```

Only packages under `internal/emit/node` can name `compilation.Input`. A parent `CompilePrepared` service may return a `PreparedBundle` that retains the original preparation privately for P07B-B and exposes only an inert defensive bundle.

### Exact six-file bundle

Generate exactly these sorted nonempty regular files, all mode `100644`:

1. `README.md`
2. `contract.test.mjs`
3. `decision.json`
4. `fixture.json`
5. `harness.mjs`
6. `manifest.json`

`decision.json` contains the closed action and `one-of-exact/v1` selected-field predicate. Allowed complete tuples are byte-deduplicated and sorted by unsigned lexicographic order of their exact canonical `ExactTuple` bodies before rendering. It is not another DecisionRecord and contains no actor, candidate, receipt, or provenance fields.

`fixture.json` contains the exact PortableSource digest and exact canonical bytes as base64. Digest-only recovery is forbidden.

`contract.test.mjs` verifies all five protected companion files before importing `harness.mjs`. `harness.mjs` is the fixed vendored second implementation. `README.md` records adapter, repository-relative entrypoint, action, selected fields, witnessed-stimulus scope, and explicit nonclaims. `manifest.json` covers exactly the other five files and never itself.

The outer runtime codec implements the exact 20-member roster and constants in `spec/schema/v1/contract-bundle.schema.json`. Its determinism scope is exactly `EMITTER_INVENTED_STRUCTURAL_FACTS_EXCLUDING_AUTHORIZED_INPUT_CONTENT`, with all seven `emitter_introduces_*` flags false. Those flags constrain typed structure invented by the emitter; they do not content-scan exact source, predicate values, digest links, or rendered human text. `CUSTOM_EXPECTATION` has exactly one allowed tuple. A2.2 updates the planning example/generator/validator to the implemented runtime bodies and gates schema required fields, runtime canonical body, strict parser, valid example, and generated decision/fixture/manifest rosters on exact parity. The self-contained A2.2 prompt freezes those values; a different wire requires a new version rather than silent drift.

All six files are UTF-8, BOM/raw-NUL/CR-free, LF-only, and end in exactly one LF; generated JSON is exact canonical body plus that LF. PortableSource uses the typed `PortableSource` digest domain, file/manifest entries use raw SHA-256 including the terminal LF, and the external bundle uses the typed `ContractBundle` digest domain over canonical body bytes. Parity vectors must kill raw/typed/base64/LF-boundary substitutions.

### Compiler algorithm

1. Validate the sanitized input.
2. Generate canonical `decision.json` plus one LF.
3. Generate canonical `fixture.json` plus one LF.
4. Render fixed-LF README text.
5. Load the fixed embedded `contract.test.mjs` and `harness.mjs` assets.
6. Enforce per-payload, tuple, selected-field, allowed-tuple, per-file, and aggregate raw ceilings.
7. Hash the five nonmanifest files over raw bytes.
8. Generate canonical `manifest.json` plus one LF covering those five sorted paths.
9. Hash the manifest raw bytes.
10. Build the outer ContractBundle with path, mode, count, raw SHA-256, and canonical base64 for all six files.
11. Enforce the final canonical body ceiling.
12. Compute the domain-separated bundle digest externally.
13. Run `ParseContractBundle` and require exact reconstruction before returning.

No generated file contains the bundle digest.

### Ceiling proof

Do not infer a sub-1-MiB bundle from a `640 KiB` per-file limit. `fixture.json` base64-wraps PortableSource and the outer bundle base64-wraps `fixture.json` again.

For file lengths `nᵢ`, prove:

```text
sum(4 * ceil(nᵢ / 3)) + maximum_bounded_metadata <= 1 MiB
```

Establish a reviewed maximum metadata envelope, choose the aggregate raw ceiling only after that proof, retain the final exact canonical-length check, test the exact accepted boundary and boundary-plus-one, refuse rather than truncate, and keep the A1 PortableSource ceiling independently enforced. Roughly `700 KiB` raw may be a starting estimate; it is not a locked constant until the proof and tests support it.

### Strict bundle parser

`ParseContractBundle` rejects unknown/missing/duplicate/noncanonical members; invalid base64; text-envelope drift; file roster/order/mode drift; raw count/hash mismatch; manifest self-coverage or omission; root/decision action or predicate mismatch; predicate stimulus disagreement with the exact stimulus reconstructed from source; root/predicate/source portable-profile mismatch; root/fixture/recomputed typed-source mismatch; a declared emitter profile not derived from exact source adapter/runner/entrypoint/start authority; selected-field/tuple order or membership drift; any exact value incompatible with its recovered field descriptor; `CUSTOM_EXPECTATION` cardinality other than one; changed fixed assets; a README that is not the exact rerender; bundle/self/caller-supplied artifact digests or tuple digests; and digest-only recovery. Closed-roster checks reject added concrete candidate/provenance/support/receipt/host-runtime/target/run/execution structure without scanning authorized source/predicate/text content for forbidden-looking bytes.

A successful parse reconstructs decision, fixture, manifest, and README; compares both fixed assets; and recovers all six exact files plus the exact `PortableSource` without compiler inputs.

### Generated Node implementation

`contract.test.mjs` registers one test, then uses its own bounded strict manifest-only parser inside the callback. It rejects a missing/symlinked/nonregular, wrong-mode where the named platform supports that check, or noncanonical manifest; requires the exact five protected paths in order; rejects nonregular or symlinked companions; checks mode under the named runtime profile; verifies byte counts and raw hashes; and imports `harness.mjs` only after verification. The entrypoint, harness, and Go consume one shared manifest corpus; no hidden seventh parser module exists. The entrypoint cannot authenticate the manifest: manifest-plus-companion coordinated replacement and same-UID verification/import TOCTOU remain explicit nonclaims even when outer parsing ran earlier. Outer parsing compares against the exact outer body/external digest supplied at that time; a completely replaced internally valid bundle plus its newly computed digest is another inert valid bundle, not a detectable forgery.

The import rosters are exact: entrypoint static imports `node:crypto`, `node:fs`, `node:path`, `node:test`, and `node:url` plus the sole post-verification dynamic `./harness.mjs`; harness static imports `node:child_process`, `node:crypto`, `node:fs`, `node:net`, `node:os`, and `node:path`, with no dynamic edge. The harness uses `process.execPath`, `shell:false`, an environment built from an empty object, literal loopback, and `node:net` rather than `node:http`. It contains no package manager, external hostname, Countershape/Wake import, didrun call, or configured service binding.

The bundle directory comes from module URL and the prepared target-source inventory comes from initial `process.cwd()`; they must be distinct and non-nested. Before any write, canonicalize the existing runtime temporary parent and reject it only when equal to or inside an input root; preexisting ancestry of either input by a common temp parent is allowed. Allocate the fresh `0700` attempt root as a create-new direct child, reopen it, and require that actual root to remain a direct child and be pairwise distinct/non-nested with both inputs. Temp-parent-to-attempt and attempt-to-subroot are the only newly created execution-path nestings; the accepted common-ancestor case requires the actual attempt to be a verified sibling of the input. Candidate/fixture/HOME/TMP/XDG/state/evidence are intentional distinct direct subroots beneath the attempt root. Then boundedly no-follow copy a source-only inventory with no `.git`, symlinks, hardlinks, or special entries, write exact fixtures/seeds create-new, verify copied/written bytes, and run only in that copy. Ordinary source files may share names such as `README.md`; the harness never discovers or loads contract members from the target inventory. Before/after manifests plus native watchers must detect injected transient-write cases and remain quiet in the named native run; notification silence is scoped platform evidence, not proof of absolute nonoccurrence. It inherits no parent environment and always tears down processes before deleting only its nonce-owned root. Direct invocation proves no Git identity; P07B-C supplies the verified pinned inventory. Topology/root, inventory/copy, environment, overlay, and cleanup failure each map to the exact A2.2 ineligible reason and never to contradiction.

The child environment is an exact name-sorted union, built from empty: source plan literals; the private HOME/TMPDIR/XDG/state/evidence/fixture bindings; one independently fresh runtime-owned `COUNTERSHAPE_ATTEMPT_ID` matching exactly `^attempt:[0-9a-f]{64}$`; `COUNTERSHAPE_SCHEDULE_ORDINAL=0`; and `COUNTERSHAPE_SCHEDULE_REPETITION=0`. The Node attempt value is not equal to or authority for any Go attempt; only the candidate-visible grammar matches. CLI adds only present source-carried stimulus environment values. HTTP instead adds the exact typed `COUNTERSHAPE_HTTP_STIMULUS_DIGEST` and `COUNTERSHAPE_HTTP_READINESS_FD=3`, with no HTTP listener FD/port. Duplicates, reserved-name collisions, inherited variables, wrong attempt grammar, or the old `COUNTERSHAPE_REPETITION` alias are refused and parity-tested.

README invocation is exactly `<explicit-node> --test --test-reporter=tap <absolute-bundle-root>/contract.test.mjs` from the prepared target cwd. The contract emits one stderr record `COUNTERSHAPE_RESULT_V1\t<outcome>\t<reason>\n`; closed outcomes are `CONFORMS`, `CONTRADICTS`, `INELIGIBLE_EXECUTION`, `MALFORMED_CONTRACT`, `TAMPER_DETECTED`, and `HARNESS_FAILURE`, with the exact stage mapping/reason roster/precedence frozen by the A2.2 prompt. The last three are standalone bootstrap/data/runtime diagnostics, not `ContractExecution` classification values or attacker attribution. All contract-controlled work runs inside one registered test and one guarded result emitter. Stdout remains explicitly selected TAP. Only conformance exits zero; every other outcome remains nonzero but machine-distinct. This direct protocol is inert operator UX and never feeds Target/FinalizedRun/Execution; P07B-C independently executes source through its owner-controlled Go target capability and derives exact lifecycle/tuple/evidence/control authority.

For HTTP, reproduce A1's exact `NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1` and `ASCII_COUNTERSHAPE_READY_V1_SPACE_PORT_LF_THEN_EOF_V1` wire: signal `ready-port-frame`, FD 3, `COUNTERSHAPE_READY_V1 ` prefix, canonical decimal port `1..65535`, 32-byte ceiling, exactly one LF then EOF, and literal `127.0.0.1` only. Every alias, extra/missing byte, timeout, early exit, or descriptor-holder case is typed ineligible.

Its byte-oriented JSON authority has two APIs. Canonicalization matches Go `canon.Canonicalize`: it accepts legal whitespace, member reordering, and escape aliases, emits exact canonical bytes, and rejects invalid UTF-8, duplicate decoded names, lone surrogates, raw unescaped controls, unsafe integers, `-0`, decimals/exponents, and truncation. Strict canonical parsing additionally compares input with those canonical bytes, so it rejects whitespace, escape aliases, and noncanonical UTF-8 key order; JSON file parsing strips exactly one required LF first. Escaped `\u0000` remains valid data. Node retains ordered key/value pairs or uses null-prototype objects with explicit own data properties; it never performs setter-bearing assignment, and all duplicate/roster/lookup checks are own-property-only. The parity corpus covers `__proto__`, `constructor`, and `prototype` at root and nested positions, including duplicate decoded escape aliases. Runtime classification preserves natural CLI exit versus signal, missing versus present-empty, exact opaque bytes, ordered duplicate headers/list members, eligible HTTP `500`, contradiction versus typed ineligibility, and owner-induced timeout/cap/cancel as ineligible. Cancellation is shared-corpus and later owner-controlled Go behavior only; direct standalone invocation has no cancellation trigger or machine-record guarantee.

### Shared Go/Node parity corpus

`spec/vectors/v1/contract-parity.jsonl` is consumed by one strict test driver, not directly by either semantic evaluator. The driver parses each closed vector, strips identifiers/descriptions/tags/expected fields, and passes only its closed semantic operation/input payload to each evaluator; evaluator production APIs and subprocess payloads have no corpus path or expected-result field. Only the driver compares independently returned actual results with expected data. Duplicate-input/different-label metamorphic cases, deliberately poisoned expectations, and an expected-echo mutant prove oracle separation. The corpus covers strict JSON/canonicalization; every portable value tag; missing/empty distinctions; tuple order/duplicates/omissions/unselected fields/cross-products; exact membership; CLI exit `2` and signal; HTTP `500`; duplicate headers; separators/OWS; content-length/transfer/content encoding failures; refused connection; hung readiness; output cap; timeout/cancel; teardown/orphan risk; and eligible versus ineligible classification.

Any Go/Node disagreement blocks A2.2.

### A2.2 required tests

- deterministic repeated compilation in one process;
- subprocess compilation across cwd, absolute root, HOME/TMP, locale, timezone, umask, and construction order;
- byte-identical paths, modes, contents, bundle bytes, and digest;
- parse/recovery after compiler inputs are discarded;
- defensive-copy and six-file tamper tests;
- manifest bootstrap/self-exclusion, entrypoint/harness/Go manifest-parser parity, companion-before-import refusal, fixed-original-outer rejection of manifest-plus-companion replacement, deterministic README rerender, and fixed program-asset digests;
- strict parser hostile corpus and Go/Node vector parity;
- exact digest-domain and all-six envelope parity;
- generated CLI and child-bind HTTP smoke;
- pairwise non-nested bundle/target/actual-attempt roots while admitting preexisting temp-parent ancestry of an input only when the attempt is its verified sibling, with temp-parent-to-attempt and attempt-to-subroot as the only newly created execution-path containment, bounded no-follow source copy, exact fixture/seed overlay, exact per-adapter sparse environment including `^attempt:[0-9a-f]{64}$`, before/after manifests plus named-platform transient-write watchers, source immutability, and cleanup faults;
- allow-many tuple preservation and custom none-conforms behavior;
- selected-field change contradicts while unselected-field change conforms;
- exact machine-record/TAP/exit separation, total reason stage/precedence faults, exact A1 readiness grammar, and Node syntax checks for both emitted modules; and
- architecture proof that compiler production code performs no filesystem or process operation.

### A2.2 mutation direction

Candidate behavioral/architecture mutants include raw/typed/base64/LF-boundary or wrong-`ContractSourceProfile`-domain hash confusion, digest-only fixture, omitted recovery member, manifest self-coverage/omission, fixed-original-outer manifest-plus-companion replacement, README/fixed-asset drift, early harness import, permissive entrypoint manifest parsing, wrong text envelope/mode, host runtime fact, legacy whole-bundle `contains_*` scanning, last-key-wins or prototype-sensitive JSON construction, conflated canonicalize/strict-parse behavior, unsafe numeric coercion, UTF-16 key or tuple ordering, tuple insertion-order retention, expected-result echo/corpus access by an evaluator, omitted predicate/source stimulus or descriptor/value join, substring membership, custom-expectation multi-tuple acceptance, missing/empty collapse, header normalization, high-level HTTP, malformed readiness acceptance, shell execution, direct/nested user-inventory execution, symlink traversal, ambient/wrong schedule/wrong attempt-ID/omitted HTTP-binding environment, transient input-root write, Countershape/Wake/package-manager/service injection, false egress-denial interpretation, and untyped/wrong-precedence machine-result collapse.

Every mutant has a named killer and fresh A/B/A restoration.

## Architecture gates

The layered A2 checker first runs the sealed A1 checker.

The Go compiler dependency closure contains only pure emitter model/sanitized-input/PortableSource/canonicalization dependencies. Forbid `os`, `io/fs`, `path/filepath`, `time`, `runtime`, randomness, `os/exec`, `net`, `net/http`, store, Git, world/process, confirmation, comparison, projection translation, and promotion. The root application service may import authority packages but no filesystem, process, Git, clock, randomness, or network packages. JavaScript assets use only their separately frozen built-in rosters.

The JS allowlist equals the exact per-asset rosters and rejects high-level HTTP/HTTPS/DNS/TLS, package imports, any other dynamic import, `eval`, `Function`, `fetch`, shell execution, environment spreading, ambient PATH, and external hosts. It requires fixed asset digests, companion verification before harness import, no self-digest, no generic public compilation constructor, no added candidate/provenance/support/receipt or host-runtime authority outside authorized content/profiles, no Wake integration, and no didrun-grade interpretation.

The hostile-copy self-test covers comment/string spoofing, transitive forbidden dependencies, import-order drift, high-level HTTP, shell/environment drift, self-digest insertion, and capability opening.

## Commit and didrun gates

At each unit, run every load-bearing command through `didrun run --`, claim each final successful command immediately without explicit event addressing, stage only the exact unit, and commit. In this managed environment, seal only with Git-note write authority, require `git notes --ref=didrun show HEAD` to succeed, and loop `NO_COLOR=1 didrun verify --strict` until exit `0`.

A2.1 requires formatting, focused/full Go tests, selected race, vet, architecture/self-test, mutation closure, source/ruling negative matrix, and restart/reopen tests. A2.2 requires formatting, focused/full/race/vet, schema/runtime/example/generated-body parity, architecture/self-test, mutation closure, oracle-blind Go/Node corpus plus poisoned-expected/metamorphic/expected-echo gates, Node syntax, generated CLI/HTTP smoke, deterministic subprocess matrix, recovery/tamper, and overflow boundaries.

Do not begin A2.2 while A2.1 is nonzero, stale, unknown, or failed. Only after A2.2 is strict-clean may P07B-B begin terminal publication and retryable materialization.

## Wake divergence

Countershape A2 is a deterministic selected-field contract compiler. It does not supervise or launch coding agents, consume Wake events/epochs/effects/gates/sessions/timelines, resume/replay/fork/recover agent work, manage fleets, become a Wake UI, or embed Wake artifacts/runtime hooks. A Wake-produced immutable Git ref may later be an ordinary external candidate input.

## Explicit nonclaims

A2 establishes no terminal residue, stale-safe publication, product materialization, target-inventory absence, subject dependency/package absence, current runtime tuple, portability, host-wide absence, network denial, sandboxing, hostile containment, confidentiality, secret-free source, authenticity, authorship, entrypoint/manifest self-authentication, coordinated-replacement resistance, human authenticity, semantic equivalence, universal specification status, future Node maintainability, or receipt stronger than its verbatim grade. A fully replaced internally valid bundle accompanied by its newly computed external digest is another inert valid bundle; detecting that substitution requires external expected-digest/current-authority, not self-authentication.

`NODE_CORE_ONLY_V1`, `package_registry_binding = NONE`, and `external_service_binding = NONE` describe only dependencies configured by the generated harness. `countershape_runtime_binding = ABSENT_BY_CONSTRUCTION` means only that the generated runtime has no Countershape import/service binding. None constrains trusted subject code's host network. Physical target-inventory absence remains P07B-C.

All emitted source retains `confidentiality_established = false`; exact fixture, stdin, body, seed, or environment bytes may still be confidential.
