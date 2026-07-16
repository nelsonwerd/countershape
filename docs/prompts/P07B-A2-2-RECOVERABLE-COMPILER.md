# P07B-A2.2 — pure recoverable six-file Node compiler

## Role

You are implementing Countershape's deterministic source compiler and its second semantic implementation. Work as a language/runtime engineer who treats exact bytes, strict recovery, cross-process determinism, parser differentials, ceiling proofs, mutation quality, and generated-contract usability as load-bearing.

This prompt does not publish terminal `RESIDUE`, write a product output directory, execute a current repository target, mutate a study head, build the product studio, or claim runtime portability.

## Required starting state

P07B-A1 and P07B-A2.1 must each have their own committed, sealed, strict-clean local boundary. Do not begin from an uncommitted, stale, unknown, or nonzero A2.1 state.

Before editing, read:

1. `docs/CONCEPT_BRIEF.md`
2. `docs/SEMANTICS.md`
3. `docs/PROJECTION_ALGEBRA.md`
4. `docs/ARCHITECTURE.md`
5. `docs/STATE_MACHINES.md`
6. `docs/THREAT_MODEL.md`
7. `docs/status/P07B-A1-SOURCE.md`
8. the sealed A2.1 status/handoff and source diff
9. `docs/prompts/P07B-A2-COMPILATION-PLAN.md`
10. `research/deep-dive/11-p07-implementation-red-team.md`
11. current Go strict-canonical, adapter, world, observation, and A2.1 model code.

Run `git status --short --branch` before editing. Preserve unrelated changes. Root remains the sole writer and sole Git/didrun operator; parallel agents are read-only critics.

## Objective

Compile only the sealed A2.1 private sanitized input—authority-narrowed rather than content-redacted, free of added concrete candidate identity/provenance but still carrying exact authorized source/predicate content and opaque source/lineage identity digests—into one deterministic inert `ContractBundle` that:

- contains exactly six sorted nonempty regular files with fixed mode `100644`;
- recovers every exact file byte without compiler inputs;
- embeds the exact A1 `PortableSource` bytes as recoverable data;
- has a strict self-excluding companion manifest and external typed bundle digest;
- invents no typed concrete candidate identity/provenance, host/current/measured runtime fact, execution receipt, target, or publication authority; the declared source/runtime profiles and exact source runner/start authorities remain declarations, not execution evidence;
- includes a fixed Node-core second semantic implementation;
- produces byte-identical output across controlled process/root/environment variation;
- refuses malformed, noncanonical, incomplete, oversized, or tampered bundles; and
- agrees with Go on the checked-in normative semantic corpus.

## Authority boundary

```text
sealed A2.1 PreparedCompilation
  -> private sanitized compilation.Input
  -> pure deterministic Compile
  -> inert recoverable ContractBundle
  -> ParseContractBundle exact reconstruction

No filesystem write
No store publication
No study-head transition
No current target execution
```

`ContractBundle.Valid()` and successful parsing establish closed construction/recovery integrity only. They do not establish authorship, authenticity, confidentiality, currentness, runtime support, semantic equivalence, target absence, or execution success.

## Expected implementation surface

Preserve actual package ownership and extend rather than duplicate existing truth:

```text
internal/emit/node/
  service.go
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
    evaluate_test.go
spec/vectors/v1/
  contract-parity.jsonl
testkit/contracts/
  fixtures.go
  node-vector-runner.mjs
  compiler_probe_test.go
tools/
  check-p07b-a2-architecture.mjs
  check-p07b-a2-architecture-selftest.mjs
  mutate-p07b-a2-compiler.mjs
```

Use checked-in source assets, not generated-at-build-time files. Fix and verify their raw digests in Go. Do not add a package manager, `package.json`, dependency lock, transpiler, bundler, formatter dependency, or download step.

## Production API

The pure compiler is conceptually:

```go
func Compile(input compilation.Input) (model.ContractBundle, error)
```

Only packages under `internal/emit/node` may name the sealed internal input. The parent application service may expose:

```go
func CompilePrepared(prepared PreparedCompilation) (PreparedBundle, error)
```

`PreparedBundle` privately retains the A2.1 preparation for later P07B-B revalidation. Its bundle getter returns only an inert defensive value. No parsed bundle can recreate `PreparedCompilation`, a current ruling token, or publication authority.

## Exact bundle roster

Generate exactly these sorted paths, all nonempty regular files with mode `100644`:

1. `README.md`
2. `contract.test.mjs`
3. `decision.json`
4. `fixture.json`
5. `harness.mjs`
6. `manifest.json`

No directory entry, executable bit, alternate case, Unicode alias, path separator alias, symlink, hardlink, extra metadata file, source map, package file, cache, log, or receipt is allowed.

## File contracts

### `decision.json`

Emit strict canonical JSON plus one LF for a closed `CompiledDecision` containing:

- exact schema/kind/version;
- exact compilable action; and
- exact `one-of-exact/v1` predicate with ordered selected fields and canonical complete allowed tuples.

For each valid complete tuple, compute the frozen canonical `ExactTuple` body
bytes without a caller-supplied digest, deduplicate only byte-identical bodies,
and sort by unsigned lexicographic comparison of those canonical UTF-8 bytes.
Locale, UTF-16 ordering, map iteration, confirmation order, and caller insertion
order have no authority. Alternate insertion orders must compile to identical
`decision.json` and bundle bytes.

It is not another DecisionRecord. It contains no actor, annotation, Choicepoint body, candidate/outcome identity, proof, receipt, runtime, target, or provenance field. No tuple digest or caller-supplied artifact digest appears inside it.

### `fixture.json`

Emit strict canonical JSON plus one LF for a closed `PortableFixture` containing:

- exact PortableSource digest; and
- exact PortableSource canonical bytes as canonical padded base64.

Digest-only recovery is forbidden. Parsing must decode the bytes, strictly parse the source, recompute its digest, and require equality.

### `harness.mjs`

This fixed vendored Node-core program is the second semantic implementation. It owns full contract-data parsing, fixture reconstruction, supported CLI/HTTP execution, capture/projection, typed eligibility, and selected-tuple membership for only the sealed profiles. The fixed entrypoint separately owns the minimal strict manifest parser required before this module may be imported.

### `contract.test.mjs`

This fixed vendored entrypoint registers one test, uses its own bounded strict manifest-only parser, and verifies every protected companion before importing the harness inside that test. It cannot authenticate itself or `manifest.json`; outer bundle recovery owns those two bytes.

### `README.md`

Render fixed-LF human documentation containing at least:

- Countershape-generated experimental contract label;
- adapter and repository-relative entrypoint;
- exact decision action;
- selected fields in order;
- exact witnessed-stimulus scope;
- how to invoke with an explicit Node executable;
- trusted-code/full-user-permissions/host-network warning;
- selected fields are asserted while context-only fields may change;
- `CONFIDENTIALITY NOT ESTABLISHED`; and
- explicit nonclaims for authorship, correctness, portability, containment, and long-term maintainability.

Do not include candidate/ref names, support counts, absolute paths, current time, host runtime facts, receipts, or a bundle digest.

README content is a closed deterministic render from parsed frozen metadata and source-derived safe fields, not caller-authored free text. `ParseContractBundle` must rerender it and require byte equality. The invocation contract is exact: use a prepared target-source inventory as cwd and run `<explicit-node> --test --test-reporter=tap <absolute-bundle-root>/contract.test.mjs`; the Node and root placeholders must be replaced by explicit absolute values, and the bundle directory and target-source root must be distinct and non-nested. The warning must say that direct invocation establishes no Git target identity and that trusted subject code still has the user's permissions and host network.

### Direct invocation result protocol

The generated entrypoint registers exactly one test and writes exactly one
bounded ASCII machine record to stderr:

```text
COUNTERSHAPE_RESULT_V1\t<outcome>\t<reason>\n
```

The closed outcomes are `CONFORMS`, `CONTRADICTS`,
`INELIGIBLE_EXECUTION`, `MALFORMED_CONTRACT`, `TAMPER_DETECTED`, and
`HARNESS_FAILURE`. The last three are direct standalone bootstrap/data/runtime
diagnostics, not
`ContractExecution` classification values and not evidence that an attacker or
author was identified.
`CONFORMS` pairs only with `NONE`; `CONTRADICTS` pairs only with
`PREDICATE_MISMATCH`; `MALFORMED_CONTRACT` pairs only with
`CONTRACT_DATA_INVALID`; and `TAMPER_DETECTED` pairs only with
`COMPANION_INTEGRITY_MISMATCH`. `HARNESS_FAILURE` pairs only with
`INTERNAL_INVARIANT_FAILED`. `INELIGIBLE_EXECUTION` pairs with exactly one
of this closed reason roster:

```text
CAPTURE_FAILED
CLEANUP_FAILED
ENVIRONMENT_INVALID
EXECUTION_ROOT_FAILED
FIXTURE_OVERLAY_FAILED
ORPHAN_RISK
OUTPUT_LIMIT
PROJECTION_FAILED
READINESS_FAILED
RESPONSE_PARSE_FAILED
SOURCE_COPY_FAILED
SOURCE_INVENTORY_INVALID
START_FAILED
TEARDOWN_FAILED
TIMEOUT
TRANSPORT_FAILED
```

The reason roster is canonical UTF-8 byte order. The record contains no captured or
caller-controlled text. Stdout remains Node's TAP stream; the subject's
stdout/stderr are bounded captured data and are never forwarded as a second
machine record. `CONFORMS` is the only outcome that permits process exit `0`;
every other outcome produces the ordinary nonzero
`node --test --test-reporter=tap` result while
retaining its distinct outcome/reason record. Missing, duplicate, malformed,
or contradictory records are harness failure, never conformance. Test exact
records for conformance, contradiction, every ineligible reason, malformed
contract data, companion tamper, and internal harness failure.

Register exactly one `node:test` case before performing any contract-controlled
bootstrap. Inside that test callback, run manifest parsing and companion
verification, dynamically import the harness, execute, fully tear down, and
clean up. Route every caught contract-controlled path through one guarded
single-use result emitter after cleanup, then return only for conformance or
throw a bounded non-caller-controlled error so every other outcome remains TAP
failure. A caught error with no exact typed mapping becomes only
`HARNESS_FAILURE/INTERNAL_INVARIANT_FAILED`; it may never masquerade as
contradiction or ineligibility. Node loader failure before the fixed entrypoint
can register its test, operator signal/process kill, OOM, or failure of the
result write itself may produce no record and remains an explicit runtime-level
nonclaim. Direct invocation defines no cancellation trigger or record guarantee;
cancellation remains in the semantic parity corpus and later owner-controlled
Go execution path, not this stderr ABI.

Use this total stage mapping. A missing, symlinked, nonregular, wrong-mode, or
expected-unreadable (`ENOENT`, `EACCES`, or `EPERM`) manifest, and any manifest
envelope/grammar/roster failure before companion comparison, is
`MALFORMED_CONTRACT/CONTRACT_DATA_INVALID`; an unexpected filesystem/runtime
error remains harness failure. After a manifest parses, a missing, symlinked,
nonregular, wrong-mode, expected-unreadable (`ENOENT`, `EACCES`, or `EPERM`),
wrong-count, or wrong-hash companion is
`TAMPER_DETECTED/COMPANION_INTEGRITY_MISMATCH`; an unexpected companion
filesystem/runtime error remains harness failure. Malformed verified decision or
fixture data is `MALFORMED_CONTRACT/CONTRACT_DATA_INVALID`; every recognized
setup/execution/control failure maps to its closed ineligible reason. Invalid
supplied-inventory structure is `SOURCE_INVENTORY_INVALID`; input/temp/attempt
root topology, canonicalization, allocation, reopen, or execution-subroot
construction failure is `EXECUTION_ROOT_FAILED`; bounded copy failure is
`SOURCE_COPY_FAILED`; fixture/seed construction failure is
`FIXTURE_OVERLAY_FAILED`; and an environment collision, reserved-name breach,
or attempt/schedule/environment grammar failure is `ENVIRONMENT_INVALID`;
an eligible complete tuple maps only by exact predicate membership; and every
otherwise-unmapped caught exception is harness failure. When several ineligible
facts survive, select exactly the first present reason in this precedence:

```text
ORPHAN_RISK
CLEANUP_FAILED
TEARDOWN_FAILED
OUTPUT_LIMIT
TIMEOUT
TRANSPORT_FAILED
CAPTURE_FAILED
RESPONSE_PARSE_FAILED
PROJECTION_FAILED
READINESS_FAILED
START_FAILED
ENVIRONMENT_INVALID
FIXTURE_OVERLAY_FAILED
SOURCE_COPY_FAILED
EXECUTION_ROOT_FAILED
SOURCE_INVENTORY_INVALID
```

This direct profile orders output-limit before deadline, while teardown/orphan
and post-run cleanup can prevent an otherwise eligible result. The shared
semantic corpus separately retains the sealed owner-event cancellation case.
Test every direct reason's stage mapping, all adjacent priority pairs,
concurrent output/timeout, primary failure
plus teardown/orphan, success plus cleanup failure, and three-fault
combinations. No later failure may be silently discarded merely because an
earlier behavioral tuple exists.

Test the exact record for every manifest path-kind/mode/read failure separately,
not only malformed JSON and companion mismatch.

This direct stderr/TAP protocol is standalone operator UX only. It is inert and
must never construct, populate, or be parsed as authority for a
`ContractExecutionTarget`, `FinalizedContractRun`, or `ContractExecution`.
P07B-C independently executes the recovered `PortableSource` through its
owner-controlled Go target capability and derives the exact lifecycle, observed
tuple, standalone evidence, and closed `common.ControlReason` from that physical
authority. No direct Node evidence channel exists in A2.2.

### `manifest.json`

Emit strict canonical JSON plus one LF. Cover exactly the first five sorted paths with their fixed mode, raw byte count, and lowercase SHA-256 over raw bytes. Never cover `manifest.json` itself. Never include the outer bundle digest.

## ContractBundle model

The outer canonical body covers all six files and stores for each:

- exact sorted path;
- fixed mode;
- exact raw byte count;
- lowercase raw-byte SHA-256; and
- canonical padded base64 of exact raw contents.

Its typed artifact digest is domain-separated and external to its canonical body. No generated file embeds it. The model must defensively copy all bytes and reconstruct every invariant on parse.

The emitter must not invent these typed structural facts:

- absolute root, cwd, HOME, TMPDIR, locale, timezone, umask, clock, random ID, OS, architecture, installed Node fact, or other host/current/measured runtime evidence;
- concrete candidate/ref/producer/support/order identity;
- process receipt, didrun event/grade/claim, target, run, classification, or materialization path authority;
- declared secret binding or a claim that source is secret-free;
- Countershape runtime import, service endpoint, package-manager or registry binding; or
- Wake event/session/runtime/artifact hook.

This is a structural authority rule, never a content scan. Exact PortableSource bytes, predicate values, source-derived README text, and typed authority-digest links remain authorized input data and may contain sensitive, host-looking, candidate-looking, receipt-looking, or order-bearing content. `confidentiality_established = false` is mandatory.

## Canonical wire compatibility lock

`spec/schema/v1/contract-bundle.schema.json` is the controlling outer body-field roster. The runtime model and strict parser must implement exactly its 20 root members, with no omission, alias, or addition. The strict canonical encoder orders object names by the existing UTF-8 key rule; the list below is a roster, not an alternate serialization order:

```text
schema_version
kind
bundle_version
decision_record_digest
choicepoint_digest
portable_source_digest
portable_profile_digest
decision_action
emitter_version
source_profile
predicate
files
manifest_policy
runtime_dependency_profile
countershape_runtime_binding
package_registry_binding
environment_profile
external_service_binding
determinism_profile
confidentiality_established
```

Freeze these exact constants:

```text
schema_version = countershape/v1
kind = ContractBundle
bundle_version = node-core-contract-bundle/v1
emitter_version = node-exact-emitter/v1
manifest_policy = COVERS_OTHER_FIVE_EXCLUDES_SELF_V1
runtime_dependency_profile = NODE_CORE_ONLY_V1
countershape_runtime_binding = ABSENT_BY_CONSTRUCTION
package_registry_binding = NONE
environment_profile = EXPLICIT_SPARSE_ALLOWLIST_V1
external_service_binding = NONE
confidentiality_established = false
```

`external_service_binding = NONE` means only that the generated harness has no configured dependency on an external service. The trusted subject remains ordinary local code with the user's filesystem and host-network authority; it may itself contact external services, and the HTTP profile uses loopback to the locally started subject. This field is not network-denial or containment evidence.

`source_profile` has exactly seven members:

```text
runtime_family = NODE
semantic_profile = countershape-node-core-exact/v1
adapter_domain = CLI | HTTP
launch_profile = NODE_REPO_SCRIPT_V1
subject_entrypoint = exact source entrypoint
start_profile = DIRECT_CHILD_V1 | NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1
scope = DECLARED_SOURCE_PROFILE_NOT_EXECUTION_EVIDENCE
```

The adapter/start pair is closed: CLI requires `DIRECT_CHILD_V1`; HTTP requires `NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1`.

The outer `predicate` has exactly `kind`, `scope`, `stimulus_digest`, `portable_profile_digest`, `selected_fields`, and `allowed_tuples`. Fix `kind = one-of-exact/v1` and `scope = EXACT_WITNESSED_STIMULUS`. The profile digest must equal the root profile digest; every tuple has only its exact ordered `fields`, and every field has only `field_id` plus the closed exact `value`.

Every outer file entry has exactly `path`, `mode`, `byte_count`, `byte_sha256`, and `content_base64`. `determinism_profile` has exactly this eight-member roster; canonical object ordering remains the existing UTF-8 key order: `scope = EMITTER_INVENTED_STRUCTURAL_FACTS_EXCLUDING_AUTHORIZED_INPUT_CONTENT`, plus seven false flags named `emitter_introduces_time`, `emitter_introduces_random_id`, `emitter_introduces_absolute_path`, `emitter_introduces_concrete_candidate_identity`, `emitter_introduces_declared_secret_value`, `emitter_introduces_host_runtime_fact`, and `emitter_introduces_execution_receipt`. These are scoped structural nonclaims, not scans or assertions about authorized source, predicate, digest-link, or human-text content. In particular, the candidate flag forbids newly emitted typed concrete candidate keys/refs/aliases/provenance; it does not erase candidate-looking behavior bytes or opaque source/lineage identities. `confidentiality_established = false` remains mandatory.

Freeze the generated JSON bodies too:

- `decision.json` has exactly `schema_version = countershape-contract/v1`, `kind = CompiledDecision`, `decision_action`, and the exact same six-member predicate object as the outer bundle.
- `fixture.json` has exactly `schema_version = countershape-contract/v1`, `kind = PortableFixture`, `fixture_version = portable-source-fixture/v1`, `portable_source_digest`, and `portable_source_base64`. The digest is independently recomputed from decoded strict source bytes; it is not caller-paired.
- `manifest.json` has exactly `schema_version = countershape-contract/v1`, `kind = IntegrityManifest`, `manifest_version = countershape-manifest/v1`, and `files`; each of its five entries has exactly `path`, `mode`, `byte_count`, and `byte_sha256`.

All six generated files are UTF-8 text with no BOM, raw NUL, or CR byte; they use LF-only line endings and end in exactly one ASCII LF. The file envelope for each generated JSON file is stricter: canonical JSON body bytes followed by exactly one LF. The strict file-envelope parser must require that LF, strip only it, strict-parse and canonical-compare the preceding bytes, and reject missing LF, CRLF, multiple terminal LFs, leading whitespace, or any other trailing whitespace. Escaped whitespace or `\u0000` inside JSON strings remains ordinary canonical JSON data.

Freeze all digest domains and byte boundaries:

- `fixture.json.portable_source_digest` is the A1 typed digest `sha256("countershape/v1/PortableSource\x00" || exact_canonical_source_bytes)` and must be recomputed after strict source parsing;
- the declared emitter-profile identity is `sha256("countershape/v1/ContractSourceProfile\x00" || exact_canonical_seven_member_source_profile_body_bytes)` with no terminal LF; it is recomputed from the closed outer `source_profile` body whenever compared or exposed for later target binding, never supplied as an independent bundle field, and same bytes under any other domain are unequal;
- every outer `byte_sha256` and every manifest `byte_sha256` is raw SHA-256 over exact file bytes, including the one terminal LF, with no Countershape domain prefix;
- the external bundle digest is `sha256("countershape/v1/ContractBundle\x00" || exact_canonical_bundle_body_bytes)` and never hashes pretty JSON, a terminal LF, base64 text, or itself.

Add parity vectors that distinguish typed source/profile/bundle digests from raw file hashes and kill the wrong profile domain, hashing of base64 text, decoded text, or LF-stripped bytes.

Update `spec/examples/v1/contract-bundle.valid.json`, `tools/generate-p07-planning-example.mjs`, and `tools/validate-planning.mjs` from planning-fixture shapes to the implemented runtime shapes. The checked-in valid example must be produced from or exact-round-trip through the runtime codec; validator acceptance alone is not enough. Add body-field parity tests that compare the schema required roster, runtime canonical body, strict parser, valid example, and generated `decision.json`/`fixture.json`/`manifest.json` rosters. Any intentional wire change requires a new version and controlling-contract review; silently adapting the runtime to a different shape is forbidden.

## Deterministic compiler algorithm

Perform these operations in fixed order:

1. validate the sealed sanitized input;
2. generate canonical `decision.json` plus exactly one LF;
3. generate canonical `fixture.json` plus exactly one LF;
4. render fixed-LF `README.md` with no platform formatter;
5. load the exact embedded `contract.test.mjs` and `harness.mjs` assets;
6. enforce per-value, selected-field, tuple, allowed-tuple, per-file, and aggregate raw ceilings;
7. hash the five nonmanifest files over raw bytes;
8. generate canonical `manifest.json` plus exactly one LF covering those five sorted paths;
9. hash the raw manifest bytes;
10. construct the outer body for all six sorted files with path/mode/count/raw digest/base64;
11. enforce the final canonical-body ceiling;
12. compute the external domain-separated bundle digest; and
13. invoke `ParseContractBundle` and require exact byte-for-byte reconstruction before returning.

No map iteration order, cwd, platform newline, locale, timezone, process environment, clock, random source, temp path, host runtime, or construction history may influence bytes.

## Ceiling proof

Do not assume a per-file ceiling implies the existing 1 MiB canonical-object ceiling. `fixture.json` base64-wraps PortableSource and the outer bundle base64-wraps `fixture.json` again.

For exact file lengths `n_i`, prove in code review and tests:

```text
sum(4 * ceil(n_i / 3)) + maximum_bounded_metadata <= MaxCanonicalObjectBytes
```

The proof must include:

- a reviewed maximum metadata envelope for paths, modes, counts, hashes, schema members, commas, quotes, and outer structure;
- the nested base64 expansion of PortableSource through `fixture.json` and the outer bundle;
- independent existing PortableSource ceiling;
- selected-field, tuple-count, value-size, README, program-asset, per-file, raw-aggregate, and final-canonical checks;
- exact accepted-boundary and boundary-plus-one tests; and
- refusal without truncation or partial output.

A rough aggregate estimate is not a locked constant. Choose constants only after the mechanical proof and tests support them. Keep the final exact canonical-length check even after static bounds are established.

## Strict bundle parser

`ParseContractBundle` must perform strict canonical parsing and reject:

- empty, oversized, unknown, missing, duplicate, reordered, or noncanonical members;
- invalid UTF-8, duplicate JSON names, lone surrogates, unsafe numbers, `-0`, decimals, exponents, or noncanonical key order;
- invalid, unpadded, alternate, or noncanonical base64;
- path traversal, absolute paths, separators/aliases, wrong roster/order/mode, duplicate or extra files;
- raw byte-count or digest mismatch;
- manifest self-coverage, omission, extra path, reordering, or mismatch;
- changed fixed program assets or a README that is not the exact deterministic rerender;
- root/`decision.json` action disagreement, or any outer/decision predicate byte disagreement;
- outer/decision `predicate.stimulus_digest` disagreement with the exact typed adapter stimulus digest reconstructed from the parsed `PortableSource`;
- root/predicate/translated-source portable-profile digest disagreement;
- root/fixture/recomputed typed PortableSource digest disagreement, or fixture/source byte disagreement;
- a declared emitter `source_profile` not exactly derived from the parsed source adapter, logical-Node runner, entrypoint, and closed start authority;
- selected fields absent from or out of order relative to the portable projection profile, tuple field roster/order disagreement, an `ExactValue` tag/body incompatible with its recovered field descriptor, duplicate tuple bodies, allowed tuples not in unsigned lexicographic exact-canonical-byte order, or `CUSTOM_EXPECTATION` with anything other than exactly one tuple;
- any bundle/self/caller-supplied artifact digest or tuple digest inside generated semantic files; `fixture.json`'s independently recomputed exact PortableSource digest is required;
- additional concrete candidate/ref/producer/support/receipt/target/run/execution members outside the frozen outer and generated semantic rosters; closed-roster checks apply structurally and must not scan opaque PortableSource content as field syntax;
- injected host Node/OS/architecture/executable-path/time/random evidence outside the exact PortableSource and frozen declared source profile; required `source_profile.runtime_family`, `source_profile.semantic_profile`, `runtime_dependency_profile`, and exact source runner/start authorities remain valid declarations rather than execution evidence;
- digest-only file or source recovery; and
- any outer body that cannot recover all six exact file bytes after compiler inputs are discarded.

Successful parse must reconstruct and compare `decision.json`, `fixture.json`, `manifest.json`, and `README.md`, compare both fixed program assets, then return defensive exact files and exact PortableSource. Mutating returned slices must not affect the bundle.

## Node entrypoint bootstrap

`contract.test.mjs` must, before importing the harness:

1. locate its directory from its own module URL without cwd dependence;
2. `lstat` and reject a missing, symlinked, nonregular, or wrong-mode `manifest.json` where the named platform supports the mode check, then open and strict-envelope-parse its exact closed profile;
3. require the exact five protected paths in exact order;
4. reject missing, nonregular, or symlinked companions;
5. verify fixed mode where the named exercised platform/runtime supports it;
6. verify exact byte count and raw SHA-256 for all five; and
7. only then dynamically import the fixed relative `./harness.mjs`.

The entrypoint implements a bounded byte-oriented manifest-only parser itself;
ordinary `JSON.parse` is insufficient. It requires the exact canonical-JSON-
plus-one-LF envelope, closed manifest roster, duplicate-name rejection, UTF-8
ordering, canonical safe integers, and exact value ceilings before trusting any
declared companion metadata. With only six files, it may not import a hidden
shared parser before verification. Generate or duplicate this reviewed minimal
parser from the same locked profile and run one manifest corpus against it, the
harness strict parser, and Go; any disagreement blocks A2.2. No other dynamic
import is allowed. Companion verification must be complete before any subject
code is started.

An intact entrypoint cannot authenticate `manifest.json`: coordinated replacement of the manifest plus any protected companion can pass its local check without replacing `contract.test.mjs`. Same-UID replacement between `lstat`/read/verification and dynamic import is also a TOCTOU residual. Outer bundle parse/recovery detects disagreement against the exact outer body and external digest supplied at the time it runs; it does not establish authorship/currentness or create hostile same-UID replacement resistance. A completely replaced, internally valid bundle accompanied by its newly computed external digest is another inert valid bundle, not a detectable forgery. State all four limits; do not narrow the nonclaim to replacement of entrypoint plus manifest.

## Node runtime restrictions

The generated harness must use Node core only:

- spawn only `process.execPath`, never ambient `node` or PATH lookup;
- `shell: false`;
- construct a sparse environment from an empty object, never spread `process.env`;
- use only literal loopback for the supported HTTP subject;
- use `node:net` raw bytes for HTTP, never `node:http`, `node:https`, `fetch`, redirects, proxy discovery, DNS, TLS, package resolution, or external hosts;
- perform no Countershape or Wake import/call;
- invoke no didrun, Git, shell, package manager, registry, compiler, or external service; and
- make every timeout, byte cap, readiness, teardown, and classification path explicit.

Subject code remains trusted and has the user's permissions and host network. Sparse environment and loopback fixture behavior are not containment or egress denial.

Freeze the JavaScript import graph by module specifier:

- `contract.test.mjs` static imports exactly `node:crypto`, `node:fs`, `node:path`, `node:test`, and `node:url`, then has the sole dynamic edge `./harness.mjs` after companion verification;
- `harness.mjs` static imports exactly `node:child_process`, `node:crypto`, `node:fs`, `node:net`, `node:os`, and `node:path`, with no dynamic import.

No transitive package import exists. Adding another built-in is a reviewed profile change, not an implementation convenience.

## Standalone execution-root contract

Keep the contract root, target-source inventory, and fresh execution root distinct:

1. `contract.test.mjs` derives the bundle directory only from `import.meta.url`. It captures the target-source inventory as the initial `process.cwd()` and passes both roots explicitly to the verified harness.
2. Require both as existing canonical private-or-operator-selected directories that are distinct and non-nested. The target inventory must contain no `.git`, symlink, hardlink, socket, device, FIFO, or other special entry. Ordinary source files may share names such as `README.md`; no contract member is discovered or loaded from this root. Direct users must prepare this source-only inventory; the later P07B-C service supplies its already Git-pinned and verified private target materialization.
3. Resolve and canonicalize the existing runtime temporary parent before any write. Reject if it equals or is inside either input root; it may be a preexisting common ancestor of an input because the nonce attempt child can still be a safe sibling. Allocate the mode-`0700` attempt root as one create-new direct child of that exact parent, reopen/canonicalize it, require it to remain a strict direct child, and require the actual attempt root to be pairwise distinct and non-nested with the bundle and target roots before copying or overlay. On rejection, remove only a nonce-owned empty attempt root. Create candidate-copy, fixture, HOME, TMP, XDG config/cache/data/state, state, and evidence as distinct direct subroots beneath the attempt root with create-new/no-follow operations. Temp-parent to attempt and attempt to those direct subroots are the only newly created execution-path nesting relations; preexisting temp-parent ancestry of either input is allowed only when the actual attempt root is the verified non-nested sibling described above.
4. Copy the target inventory into the candidate root with a bounded sorted no-follow traversal. Admit only directories and regular `100644`/`100755` files, enforce the source-carried entry/aggregate/single-file ceilings, reopen and raw-hash every destination, and compare a runtime-built manifest before proceeding. Direct invocation proves only copy agreement with the supplied inventory, not Git identity or historical candidate identity.
5. Resolve the repository-relative subject entrypoint lexically beneath the private candidate copy and spawn only `process.execPath` plus the exact source argv with child cwd equal to that copy. Never execute from or write into the supplied target inventory.
6. Recreate exact source-carried CLI fixtures or HTTP seeds only beneath the separate fixture root using sorted create-new/no-follow writes, exact mode/count/raw-hash reopen checks, and collision refusal. Apply exact stdin/body bytes and start/readiness/capture limits without normalization.
7. Build the child environment from an empty object and emit entries sorted by exact name. Refuse every duplicate or collision with runner-owned names. Common entries are the source plan's exact public literals; `HOME`, `TMPDIR`, `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`, `XDG_DATA_HOME`, and `XDG_STATE_HOME` bound to their private roots; `COUNTERSHAPE_STATE_ROOT`, `COUNTERSHAPE_EVIDENCE_ROOT`, and `COUNTERSHAPE_FIXTURE_ROOT` bound to their private roots; one independently fresh runtime-owned `COUNTERSHAPE_ATTEMPT_ID` matching exactly `^attempt:[0-9a-f]{64}$`; and exactly `COUNTERSHAPE_SCHEDULE_ORDINAL=0` plus `COUNTERSHAPE_SCHEDULE_REPETITION=0`. The Node attempt value intentionally differs from historical/current Go attempts and grants no target or study authority; only the candidate-visible grammar is shared. CLI additionally includes only present source-carried stimulus environment bindings, omitting absent bindings. HTTP includes no CLI stimulus bindings and additionally includes `COUNTERSHAPE_HTTP_STIMULUS_DIGEST=<exact typed source stimulus digest>` and `COUNTERSHAPE_HTTP_READINESS_FD=3`; FD 3 is the sole owned readiness pipe, while `COUNTERSHAPE_HTTP_LISTEN_FD` and `COUNTERSHAPE_HTTP_PORT` are absent. Inherit no parent variable, `PATH`, proxy, credential, Node option, or loader setting.
8. On every success or failure, finish process-group teardown first and then remove only the nonce-owned attempt root. Copy, overlay, reopen, cleanup, or uncertain-orphan failure is typed ineligible and can never become contradiction or conformance.

Test with bundle and target roots on different paths, spaces in parent paths, changed cwd, source collisions, symlink/hardlink/special-entry attacks, oversized copies, fixture/seed collisions, injected parent secrets/proxies/loaders, cleanup faults, and a source-root before/after manifest proving the supplied inventory was not mutated.

Also test runtime temporary parents equal to, inside, and symlink-aliased to
each input root as refusals; a common-ancestor parent producing a safe sibling
attempt as an accepted case; plus a post-create canonical-path substitution. A native
read-only watcher on both input roots must remain quiet for the complete run so
a transient create/delete cannot hide behind equal before/after manifests.
Watcher availability and semantics are named platform evidence, not a portable
containment claim.

## Exact portable HTTP readiness wire

The second semantic runtime must reproduce A1's exact start/readiness profile,
not merely wait for arbitrary FD 3 text. Require:

```text
start authority: NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1
protocol: ASCII_COUNTERSHAPE_READY_V1_SPACE_PORT_LF_THEN_EOF_V1
signal name: ready-port-frame
inherited readiness descriptor: 3
frame prefix: COUNTERSHAPE_READY_V1<single ASCII space>
maximum complete frame: 32 bytes
endpoint after acceptance: literal 127.0.0.1:<decoded port>
```

After spawn, read at most 32 bytes under the exact source readiness budget. The
only accepted payload is the ASCII prefix, one canonical unsigned decimal port
in `1..65535` with no sign, leading zero, whitespace, or non-ASCII digit,
exactly one LF, and immediate EOF. Do not connect until the entire frame and EOF
are accepted. Then make one raw `node:net` connection to literal `127.0.0.1` and
that port; perform no DNS, retry, redirect, proxy, or alternate-address lookup.
Reject empty, partial, oversized, wrong-prefix, zero/out-of-range, leading-zero,
signed, spaced, CRLF, doubled-LF, NUL/non-ASCII, extra-byte, missing-EOF,
multiple-frame, early-exit, timeout, and holder-process cases as typed
ineligible readiness/control results. Parity vectors and generated HTTP smoke
must cover both accepted boundary ports and every direct rejection class.
Cancellation is covered only by the shared semantic corpus and later
owner-controlled Go execution; direct standalone invocation defines no
cancellation trigger or machine-record guarantee.

## Strict Node JSON/canonical implementation

Identity-sensitive input must not rely on ordinary `JSON.parse` alone. Implement two bounded byte-oriented APIs matching the checked-in Go authority:

1. `canonicalizeJSON` matches `canon.Canonicalize`: it admits valid surrounding/inter-token whitespace, reordered object members, and valid escape aliases such as `\u0061` or `\/`, then returns the exact Go canonical bytes. It rejects invalid UTF-8, duplicate decoded object names, lone surrogates/invalid escapes, raw unescaped control bytes including raw NUL, unsafe integers, `-0`, decimal/exponent spellings, truncation/trailing non-whitespace data, and excess depth/members/bytes/tokens. Escaped `\u0000` is valid data and canonicalizes normally.
2. `parseCanonicalJSON` calls that canonicalizer and additionally requires byte-for-byte equality between its input and the returned canonical bytes. It therefore rejects whitespace, noncanonical escapes, and noncanonical UTF-8 object-key order. JSON file-envelope parsing first requires and strips exactly one LF, then applies this strict canonical-body API.

Parity vectors must include both normalization-accepted inputs and identity-rejected aliases; never make Go `canon.Canonicalize` stricter merely to simplify Node. If Node cannot match a Go-accepted shape exactly, narrow the standalone supported semantic input profile explicitly and update both implementations; never silently normalize or broaden an identity boundary.

Both Node APIs must construct objects without prototype-sensitive assignment:
retain ordered key/value pairs or use null-prototype objects plus explicit own
data-property definition, never `object[key] = value` on a setter-bearing
prototype. Duplicate detection, roster checks, and lookups are own-property-
only. The shared Go/Node corpus and mutation suite cover `__proto__`,
`constructor`, and `prototype` at root and nested positions, including duplicate
decoded names formed through escape aliases.

## Execution and classification semantics

The second implementation must preserve:

- natural CLI exit code versus natural signal;
- absent versus present-empty stdin/body/config/value;
- exact opaque stdout/stderr/body bytes;
- ordered duplicate HTTP headers and ordered string-list members;
- raw response status, including eligible HTTP `500`;
- exact response separator and OWS rules from the supported Go grammar;
- typed transport/readiness/capture/projection/teardown ineligibility;
- owner-induced timeout, output cap, cancel, and cleanup control as ineligible, never contradiction;
- `ELIGIBLE_OBSERVATION` before predicate membership;
- complete-tuple exact membership, never substring/regex/tolerance/per-field product; and
- unselected-field variation remaining conforming when selected fields are unchanged.

High-level HTTP normalization is prohibited even if ordinary fixtures happen to agree.

## Shared Go/Node normative corpus

Create `spec/vectors/v1/contract-parity.jsonl` as a bounded strict line-oriented corpus consumed only by a strict test driver. The corpus is data, not an alternate authority constructor. Neither semantic evaluator may read the corpus directly.

The driver parses a closed vector, keeps its identifier, description, tags, and
expected result private, and sends each Go or Node evaluator only an exact
closed operation plus semantic input payload. Production evaluator APIs and
Node subprocess payloads contain no expected/answer/result-oracle field and no
case identifier that can encode one. Evaluator code has no corpus path or
filesystem lookup. Only the driver compares independently returned actual
results to the vector's expected result. Duplicate exact inputs under different
driver labels must produce identical actual results; changing only a cloned
vector's expected result must leave evaluator output unchanged and make the
driver fail. Include poisoned-expected, shuffled-vector-order, and
duplicate-input metamorphic tests, plus an expected-echo/case-ID switch mutant
with a named architecture killer.

Cover at least:

- strict JSON and canonicalization acceptance/rejection;
- every portable value tag;
- missing/null/empty-string/empty-bytes/empty-list/list-with-empty distinctions;
- ordered duplicates and list order;
- tuple field order, omissions, duplicates, unselected fields, allow-many correlation, and Cartesian-product traps;
- exact membership versus substring/regex/tolerance traps;
- CLI exit `0`, clean nonzero such as `2`, and natural signal completion;
- CLI stdout/stderr opaque bytes and JSON projection paths;
- HTTP status including `500`;
- duplicate headers and order;
- response separators and OWS;
- content-length, transfer-encoding, content-encoding, truncation, and extra-byte failures;
- connection refusal and transport close;
- malformed/missing/extra readiness frames and hung readiness;
- output cap, timeout, cancel, teardown, and orphan-risk classification;
- eligible versus ineligible observation; and
- selected-field conforming/contradicting cases.

Each vector must have one closed expected result. The driver must refuse malformed vector schema before constructing either input-only evaluator call. Any Go/Node disagreement or oracle-separation failure blocks A2.2.

## Positive verification matrix

Prove:

1. repeated compilation in one process is byte-identical;
2. subprocess compilation is byte-identical across different cwd, absolute repository root, HOME/TMPDIR, locale, timezone, umask, environment order, and model construction order;
3. exact paths, modes, file bytes, manifest, outer bytes, and typed digest agree;
4. parsing recovers all six exact files and PortableSource after discarding compiler inputs;
5. defensive copies cannot mutate bundle identity;
6. generated CLI contract smoke passes for allow-one and allow-many;
7. generated child-bind raw-HTTP contract smoke passes;
8. custom none-conforms behavior works;
9. selected-field change contradicts while context-only change conforms;
10. intact entrypoint rejects each changed companion before harness import;
11. outer parsing rejects tamper of each of the six files;
12. both emitted modules pass Node syntax checking;
13. fixed assets and their exact static/dynamic import rosters match reviewed digests;
14. compiler production code performs no filesystem, process, Git, clock, randomness, environment, runtime, or network operation;
15. typed PortableSource/ContractSourceProfile/ContractBundle digests and raw per-file hashes agree with independent vectors and disagree with wrong-domain/base64/text/LF-stripped hash mutants;
16. all-six text envelopes and the three canonical-JSON-plus-one-LF envelopes round-trip exactly and reject every newline/BOM/raw-control alias;
17. bundle root, target source, and the actual direct-child attempt root satisfy the exact pairwise non-nesting contract—with preexisting temp-parent ancestry of an input admitted only when the attempt is its verified sibling, and only temp-parent-to-attempt plus attempt-to-subroot as newly created execution-path containment—while the subject runs from the private copy with exact fixtures/seeds, the exact per-adapter sparse environment roster, quiet input-root watchers, source-root immutability, and complete cleanup;
18. with the original outer body and external digest fixed, manifest-plus-companion replacement fails outer count/hash/content joins; README must equal its deterministic rerender and fixed program assets must retain reviewed digests even when inner or outer metadata is recomputed;
19. exact root/action/predicate/profile/stimulus/source joins, per-field descriptor/value compatibility, and the one-tuple `CUSTOM_EXPECTATION` cardinality survive schema/runtime/example/generated-body parity;
20. alternate allowed-tuple insertion orders produce the same unsigned-byte canonical order and exact bundle bytes;
21. the parity driver alone owns expected results, input-only Go/Node evaluators remain invariant under poisoned expected data, duplicate inputs, and vector order, and an expected-echo/case-ID mutant is killed;
22. direct `<explicit-node> --test --test-reporter=tap` runs emit the exact single machine record and exit/TAP separation for conformance, contradiction, every typed ineligible reason and precedence combination, malformed data, tamper, and harness failure; and
23. the exact child-bind readiness grammar agrees with A1 on accepted boundary ports and every malformed/timeout/early-exit/holder negative.

Generated smoke may materialize into test-owned temporary directories at the test edge. Production compiler code may not.

## Negative verification matrix

Include:

- every strict parser rejection category;
- every roster/order/path/mode/base64/count/hash/manifest mismatch;
- self-digest and manifest-self-coverage attempts;
- changed fixed program asset;
- decision action/predicate, fixture/source, predicate/source stimulus, portable-profile, per-field descriptor/value tag, and declared-source-profile cross-pairs;
- `CUSTOM_EXPECTATION` with zero or multiple allowed tuples;
- digest-only fixture recovery;
- raw-versus-typed, base64-text, decoded-text, or LF-stripped digest substitution;
- missing LF, CRLF, doubled terminal LF, BOM, raw NUL/control, and other all-six text-envelope drift;
- manifest-plus-companion replacement against the fixed original outer body/digest, README drift from deterministic rerender, fixed-program-asset drift, and semantic cross-pairs even when an attacker recomputes available inner or outer metadata;
- the legacy whole-bundle `contains_*` determinism shape, or content scanning that rejects authorized source/predicate/text bytes merely because they look like paths, candidates, receipts, or secrets;
- concrete candidate/provenance/receipt or host-runtime-evidence field injection into the closed outer/generated semantic rosters, without treating opaque PortableSource content as injectable structure;
- cwd/PATH/environment dependence; bundle/target overlap; a temp parent equal to or inside either input; an attempt root that is not a strict direct child or is equal/nested with an input; execution-subroot escape/collision; transient input-root writes; direct execution from the supplied inventory; source mutation; symlink/hardlink/special-entry traversal; fixture/seed collision; wrong-prefix/length/uppercase attempt ID; wrong/missing `COUNTERSHAPE_SCHEDULE_REPETITION`; missing/wrong HTTP stimulus digest or FD3 readiness binding; inherited extra variables; or cleanup escape;
- `shell:true`, high-level HTTP, external host, package import, or package-manager insertion;
- treating `external_service_binding = NONE` as egress denial for trusted subject code;
- manifest verification after harness import;
- per-file and aggregate boundary-plus-one;
- oversized valid source/value/tuple/README metadata without truncation;
- missing versus empty collapse;
- tuple Cartesian product;
- header join/sort/deduplication;
- normal exit versus signal collapse;
- HTTP `500` treated as transport failure;
- malformed child-bind readiness grammar or readiness without exact LF-plus-EOF;
- direct timeout/output/teardown treated as contradiction or selected with the wrong exact precedence, plus cancellation collapsed incorrectly in the shared semantic corpus; and
- any error returning partial files or a nonzero bundle.

## Architecture gate

The layered A2 checker must first run the strict A1 checker and retain A2.1 closure. Close the Go compiler/model transitive dependency set over only the pure emitter model, fixed program assets, sanitized input, PortableSource, canonicalization, and their required pure dependencies. Check JavaScript assets against their separately frozen built-in import rosters.

Forbid in the Go compiler/model production closure:

- `os`, `io/fs`, `path/filepath`;
- `time`, `runtime`, environment inspection, randomness;
- `os/exec`, `syscall`, process packages;
- `net`, `net/http`, DNS, TLS;
- store, Git, world/process, confirmation, comparison, projection translation, promotion; and
- server, report, browser, didrun, Wake, materializer, target, or execution packages.

The JavaScript import allowlist must equal the exact per-asset rosters above. Reject high-level HTTP/HTTPS, DNS, TLS, package imports, any other dynamic import, `eval`, `Function`, `fetch`, shell execution, environment spreading, ambient PATH lookup, external hosts, Countershape/Wake imports, service calls, and package managers.

Require fixed asset digests, verification-before-import ordering, no self-digest, no public generic compiler-input constructor, no concrete candidate/provenance/support or receipt fields and no injected host-runtime evidence outside the exact PortableSource/frozen declared profiles, and no didrun-grade interpretation. Evaluator production APIs and Node process payloads must be input-only: reject corpus paths, vector identifiers/tags/descriptions, expected-result fields, expected-field reads, or switches keyed to driver-only case identity.

The hostile-copy self-test must catch comment/string spoofing, transitive forbidden imports, per-asset import-roster drift, a second dynamic edge, high-level HTTP, shell/environment drift, self-digest insertion, manifest self-coverage, verification-after-import, fixed-asset drift, direct-target-root execution, expected-result/corpus access or case-ID branching in an evaluator, and capability opening.

## Mutation gate

Use fresh A/B/A copies and freeze only non-equivalent mutants with named killers. Include strong candidates such as:

1. hash base64 text instead of raw file bytes;
2. retain only PortableSource digest;
3. omit one file from outer recovery;
4. include manifest in itself;
5. omit one protected companion from manifest;
6. import harness before companion verification;
7. emit mode `100755`;
8. insert host Node/OS/architecture/time/path fact;
9. accept last-key-wins duplicate JSON;
10. accept unsafe numeric coercion or `-0`;
11. compare UTF-16 rather than UTF-8 key order;
12. use substring or regex membership;
13. collapse missing and empty;
14. sort/join/deduplicate ordered headers;
15. use `node:http` or `fetch`;
16. use `shell:true`;
17. inherit ambient environment or PATH;
18. inject a Countershape, Wake, package-manager, registry, or service dependency;
19. treat natural signal as ordinary exit;
20. treat HTTP `500` as transport failure;
21. flatten ineligible execution into contradiction;
22. raw-hash PortableSource or typed-hash a file;
23. accept missing/CRLF/double-LF JSON envelopes;
24. scan authorized predicate/source text for forbidden-looking content;
25. allow multiple custom-expectation tuples;
26. skip deterministic README regeneration;
27. accept manifest-plus-companion replacement against the fixed original outer body/digest or accept changed fixed program assets/README rerender;
28. run the subject in the supplied source inventory or write fixtures/seeds into it;
29. follow a source/fixture symlink or inherit a parent credential/proxy/loader variable;
30. interpret `external_service_binding = NONE` as subject network containment;
31. preserve allowed-tuple insertion order or use UTF-16/locale ordering;
32. emit `COUNTERSHAPE_REPETITION`, use a non-`attempt:`/non-64-lowercase-hex attempt ID, omit `COUNTERSHAPE_HTTP_STIMULUS_DIGEST`, or omit/misbind `COUNTERSHAPE_HTTP_READINESS_FD=3`;
33. collapse the machine records, omit harness failure, or let contradiction/ineligibility share an untyped diagnostic;
34. omit the predicate/source stimulus join or accept a value tag incompatible with its recovered field descriptor;
35. accept a noncanonical/extra/missing-EOF child readiness frame;
36. allocate the attempt root inside an input root or ignore a transient input-root write;
37. hash the emitter profile under `SourceProfile` or another wrong domain, trust a caller-paired profile digest, or include a terminal LF; and
38. pass expected results/case identifiers into an evaluator, read the corpus from evaluator code, or echo/switch on the expected answer.

The harness must self-test mutant application, named killer mapping, exact restoration, path containment, and tamper detection. Do not count noncompiling or equivalent mutants as killed semantic obligations.

## Multi-pass build loop

This unit is not one-and-done and unit tests are not a substitute for the generated artifact.

Perform at least these passes:

1. **Model pass:** compile/parse/recover exact bundle; review canonical bytes and ceilings.
2. **Runtime pass:** materialize test copies and exercise real generated CLI and child-bind HTTP contracts.
3. **Adversarial pass:** strict parser corpus, per-file tamper, bootstrap ordering, mutation closure, and architecture hostile self-test.
4. **Determinism pass:** repeat across processes, roots, environment variants, and construction order.
5. **Human-touch pass 1 — source and 80-column render:** inspect README Markdown source plus a deterministic no-network terminal rendering at 80 columns. Exercise CLI success, contradiction, ineligible control, malformed bundle, and tamper diagnostics. Check hierarchy, line wrapping, warning prominence, exact scope, exit guidance, copy/paste safety, and absence of ANSI/control activation.
6. **Human-touch pass 2 — narrow and wide:** repeat the generated README/CLI capture at 60 and 120 columns, with `NO_COLOR=1`, long entrypoints/field names, missing/empty values, and the largest bounded diagnostic. Fix clipping, ambiguous indentation, unstable wrapping, jargon, and buried trust warnings.
7. **Human-touch pass 3 — task flow:** from a clean materialized test copy, have a fresh read-only critic follow only the generated README to invoke the contract and interpret conforming, contradicting, and ineligible results. Disposition every comprehension/DX finding and recapture all three widths.
8. **Different-model critic:** after the local passes are stable, pause immediately before any real external-provider call and request human authorization. If authorized, provide only sanitized captured README/CLI artifacts, ask a different model to critique hierarchy, warnings, terminology, recovery guidance, and novice/mid-level usability, and disposition every finding without granting semantic authority. Never substitute the same model, simulate a response, or send source/evidence that has not been cleared. If authorization or a provider is unavailable, record this gate `UNRECEIPTED` and stop the full A2.2 handoff rather than claiming it passed.
9. **Regression pass:** rerun the complete functional, parity, determinism, parser, mutation, architecture, three-width human-surface, and final didrun gate after all critic-driven edits.

Visual browser work belongs to U8, but generated README/CLI feel is already load-bearing. Check in sanitized deterministic captures or structured expectations only when the owning prompt explicitly requires them; never retain secrets or machine-specific paths. Record manual/different-model taste observations as such; they are not didrun grades unless the exact external command is receipted, and even then the receipt proves only that command ran.

## didrun and resource discipline

Every load-bearing command runs through `didrun run --`. Serialize every didrun operation. Use at most two test workers where configurable and one worker for fuzz/mutation loops unless measured evidence justifies more, leaving host headroom for other work.

Final gates must include:

- exact staged inventory/diff check;
- format and Node syntax checks;
- focused compiler/model/parser tests;
- exact schema/runtime/example/generated-body field-roster and constant parity;
- full `go test ./...`;
- full `go vet ./...`;
- selected/full race coverage;
- architecture checker and hostile self-test;
- mutation harness self-test and complete non-equivalent mutant closure;
- shared Go/Node corpus parity plus poisoned-expected, duplicate-input, shuffled-order, and expected-echo oracle-separation gates;
- real generated CLI and HTTP smoke;
- deterministic subprocess/root/environment matrix;
- recovery and six-file tamper matrix;
- exact ceiling/boundary/overflow tests;
- bounded active fuzzing of bundle/parser/corpus seams;
- generated README/CLI human-touch critic disposition check; and
- scoped staged structured credential-pattern scan.

Immediately after each final successful verification command, declare the exact truthful claim before recording another event. Do not bind claims to older explicit event numbers. Failed development events are permanent history and support no capability.

Then:

1. finish and immediately claim the non-staged final gates;
2. stage only the exact A2.2 files;
3. run the exact staged inventory/diff check through didrun and immediately claim it;
4. run the scoped structured credential-pattern scan over the exact staged blobs through didrun and immediately claim only that named scope;
5. inspect the staged diff, private artifacts, and obvious credential exposure;
6. commit without an AI co-author trailer;
7. run `NO_COLOR=1 didrun seal --commit HEAD`;
8. investigate any aggregate entropy failure before using the logged redacted override;
9. loop `NO_COLOR=1 didrun verify --strict` until exit `0`, fixing and re-receipting real failures; and
10. archive the exact uncommitted ledger under a named `.didrun-history/` boundary.

Never weaken a parser, ceiling, test, vector, mutant, architecture rule, claim, receipt, or old commit to open the gate.

## Honest claim vocabulary

Permitted narrow claims, when directly receipted:

- pure compiler produced and reparsed the exact six-file bundle in the named tests;
- checked-in Go and Node implementations agreed on the named corpus;
- generated CLI/HTTP smoke passed on the explicitly named local runtime tuple;
- deterministic matrix produced byte-identical output across its named variations;
- tamper/overflow/architecture/mutation gates passed their named cases; and
- exact test/race/vet/fuzz commands succeeded on the named tree.

Do not generalize a local Node/Darwin smoke into portability. Do not claim terminal residue, stale-safe publication, product materialization, target-inventory absence, package-free subject code, host-wide absence, network denial, sandboxing, hostile containment, confidentiality, authenticity, authorship, coordinated-replacement resistance, semantic equivalence, production readiness, security approval, adoption, or long-term maintainability.

`NODE_CORE_ONLY_V1`, `package_registry_binding = NONE`, and `external_service_binding = NONE` describe only dependencies configured by the generated harness. `countershape_runtime_binding = ABSENT_BY_CONSTRUCTION` means only that the generated runtime has no Countershape import/service binding. None constrains trusted subject code's host network. Physical target-inventory absence remains P07B-C.

## Stop conditions

Stop and write an honest handoff without beginning P07B-B if:

- exact six-file recovery requires retained compiler inputs;
- manifest bootstrap cannot verify all protected companions before harness import;
- any generated file must embed its own bundle digest;
- the ceiling proof does not fit the existing canonical-object maximum;
- Go/Node corpus results disagree or an evaluator can observe corpus metadata, case identity, or expected results;
- generated HTTP must use a normalizing high-level client;
- production compiler code needs filesystem/process/network/clock/random/runtime authority;
- deterministic bytes vary across the required matrix;
- a required non-equivalent mutant survives;
- a load-bearing final command remains nonzero; or
- strict didrun verification does not exit `0`.

## Done condition

A2.2 is done only when its independently committed implementation, vectors, generated assets, tests, status, and handoff are sealed; the exact final claims show verbatim grades; the didrun chain is intact; and `NO_COLOR=1 didrun verify --strict` exits `0`.

Only then may a fresh unit implement terminal publication and retryable materialization. A parsed bundle alone must remain inert.
