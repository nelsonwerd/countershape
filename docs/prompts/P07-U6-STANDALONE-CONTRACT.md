# P07B — U6 recoverable standalone contract residue

This is the controlling umbrella prompt. Do not execute it as one monolithic unit. P07B-A1 is sealed at source commit `1e56da3bfafc4cdb8abdca62f18fb3b1accea8c3`; its exact boundary and nonclaims live in `../status/P07B-A1-SOURCE.md`. Execute the locked decomposition in `P07B-A2-COMPILATION-PLAN.md`: A2.1 current-ruling compilation authority, A2.2 pure recoverable compiler, then independently gated B publication/materialization and C target/run/execution units. Every unit must be committed, sealed, and `NO_COLOR=1 didrun verify --strict` exit `0` before the next begins.

Read `docs/CONCEPT_BRIEF.md`, `docs/SEMANTICS.md`, `docs/PROJECTION_ALGEBRA.md`, `research/deep-dive/11-p07-implementation-red-team.md`, the prior portability/acceptance research, the A1/A2 status and prompt files, and the Mode C handoff. Inspect installed Node versions before naming support. Source intent is not a runtime receipt: every unexecuted Node/OS/architecture tuple remains `UNRECEIPTED`.

## Objective

Build a pure deterministic Go compiler from two independent exact authorities:

1. one current store-bound P07A portable `promotion.Ruling`; and
2. one byte-complete `PortableSource` reconstruction witness whose adapter constructors reproduce the exact plan, projection binding, and minimized stimulus bound by that ruling's Choicepoint.

Compile those inputs into one recoverable deterministic six-file Node-core ContractBundle, publish it as terminal `RESIDUE`, materialize it retryably, and publish later exact-pinned targets, finalized physical runs, and classifications as separate immutable nonhead objects. Prove the narrow Go and Node implementations agree on the exercised semantic profile and that CLI and HTTP bundles run from isolated target inventories without Countershape in the target source, dependencies, import graph, `PATH`, or service-call closure.

The five truth objects remain separate:

1. `DecisionRecord` is immutable, locally caller-attributed historical ruling provenance. It does not authenticate a human and does not contain runnable source bytes.
2. `ContractBundle` is deterministic recoverable source plus byte identity. Its external typed digest is the terminal residue address; no self-digest appears inside its body.
3. `ContractExecutionTarget` is immutable nonhead pre-spawn authority over the reopened bundle/source profile, one explicit Git-pinned inspected and verified materialization, one durably allocated fresh conformance attempt, and one admitted/revalidated Node runtime. Its external typed digest is not inside its body. It is not a historical `WorldInstance`, and a parsed body is inert.
4. `FinalizedContractRun` is immutable nonhead authority over exactly what happened to one target and its exact attempt: closed lifecycle, one constructor-derived `ELIGIBLE_CLEAN` or `INELIGIBLE_CONTROL(reason)` disposition, the exact projected tuple when present, and scoped standalone evidence. It references the target externally and cannot be paired with another target or attempt.
5. `ContractExecution` is only the derived conformance classification. It references exactly one target and that target's exact finalized run, and carries only `CONFORMS`, `CONTRADICTS`, or the ineligible class—never an independently supplied tuple or control reason. It carries no duplicate runtime, lifecycle, observation, or standalone authority. It neither recreates nor freshens the historical Choicepoint and never advances the terminal study head.

Do not describe the output as a universal specification, semantic equivalence, safety proof, framework-neutral test, authenticity proof, or portability beyond exact physically receipted runtime tuples.

## Locked compilation authority

The application service begins from the current store-bound `promotion.Ruling`, reopens and validates its exact DecisionRecord, Choicepoint, FreshConfirmation, and head token, and requests the sealed P07A portable compilation authority. Serialized `compilable` flags, profile digests, field names, tuples, or caller-authored DTOs can never reconstruct this authority.

Only `ALLOW_OBSERVED` and separately reviewed `CUSTOM_EXPECTATION` may compile. `REJECT_ALL`, `DEFER`, `REFINE`, legacy whole-projection Choicepoints, and stale rulings refuse before source or file publication. Exact refusals include:

```text
LEGACY_WHOLE_PROJECTION_NOT_PORTABLE
PORTABLE_SOURCE_REQUIRED
NONPORTABLE_START_PROFILE
STALE_CHOICEPOINT
```

The predicate remains `one-of-exact/v1` over the P07A closed portable value algebra. It contains complete selected-field tuples, never candidate identity, branch/ref/producer metadata, support counts, majority/first/name cues, per-field independent sets, or a cross-product. Selected-field separation is revalidated during compilation. `CUSTOM_EXPECTATION` retains its selected-only review authority and is the only compilable path where none of the confirmed candidates conforms.

## Byte-complete PortableSource

Sealed A1 owns `internal/contractsource/**`. Preserve its exact canonical bytes and strict boundary. A `PortableSource` is not an arbitrary byte slice plus claimed digests. Its closed adapter-specific constructors retain bounded exact inputs, reconstruct the adapter stimulus/execution binding themselves, and return an opaque source whose strict parser repeats the reconstruction. A2 must consume that authority rather than add an alternate source constructor.

It retains only what the standalone harness needs, including:

- exact adapter and closed launch profile;
- repository-relative entrypoint and explicit argv;
- sparse explicit environment and private-root policy;
- CLI stdin and fixture contents;
- HTTP request body, ordered query/headers, seed contents, readiness/start facts;
- capture, timeout, output, and teardown limits;
- exact plan, projection binding, and minimized-stimulus identities.

Before a compiler byte exists, require:

```text
source.plan_digest == choicepoint.world_plan_digest
source.adapter == choicepoint.world_plan.adapter
source.projection_binding_digest == choicepoint.projection_binding.digest
source.portable_profile_digest == ruling.portable_profile.digest
source.reconstructed_stimulus_bytes == choicepoint.minimized_stimulus_bytes
source.reconstructed_stimulus_digest == choicepoint.minimized_stimulus_digest
source.reconstructed_execution_binding_digest == every schedule-ordered strictly parsed FreshExecutionFact execution_binding_digest
project_selected(retranslate(full reopened FreshConfirmation projection roster, resolved source-matched profile)) == exact ruling allowed/disallowed selected-tuple partition
```

The application service owns the last equality: it reopens the Choicepoint and FreshConfirmation, verifies the complete proof roster, retranslates those exact projection bytes under the source-matched resolved profile, projects the full roster onto the ruling's selected fields, and exact-set compares the reconstructed allowed/disallowed partition. `PortableSource` contains no confirmation projection, tuple, or proof bytes and cannot author that equality.

Use exact canonical bytes and typed digests, not names or digest-only pairing. If raw bytes are unavailable, return `PORTABLE_SOURCE_REQUIRED`; do not guess, fetch, hardcode, or reverse a digest.

Use only `NODE_REPO_SCRIPT_V1`: historical authority establishes logical `node` plus one exact clean repository-relative `.js`, `.mjs`, or `.cjs` script. A fresh `ContractExecutionTarget` admits the target Node runtime; because the bundle itself runs under that runtime, its subject `{node}` resolves to the same `process.execPath`. This does not claim the target absolute Node binary is the historical binary. Arbitrary repository executables remain unsupported until a separately versioned physical lineage establishes their native-format, shebang, and interpreter authority.

No ambient executable lookup, absolute executable, shell, login profile, package manager, setup command, external host, or user configuration is allowed.

## Portable HTTP lineage

Do not translate the sealed inherited-listener HTTP world into a different standalone start mechanism. Preserve A1's sealed second explicit adapter start profile and its canonical bytes:

```text
NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1
```

The sealed A1 fixture child binds literal `127.0.0.1:0`, writes one closed bounded readiness frame containing the chosen port to a dedicated inherited pipe, closes the readiness pipe, serves exactly the declared request budget, and completes teardown. The Go runner owns deadlines, process group, readiness validation, probe, response caps, and post-exit orphan checks. Keep the historical inherited-listener profile valid but nonemittable. Do not rewrite the A1 fixture, profile, source, or receipt history.

In A2.1, use the sealed A1 profile/fixture to create the still-missing physical confirmation, portable Choicepoint, and selected-field ruling lineage; a fresh study necessarily creates its own plan and observations without altering A1 history. Only a current ruling joined to exact source under that lineage may authorize HTTP compilation. Projection bytes may remain identical when behavior is identical; plan, world, attempt, map, Choicepoint, and Decision identities correctly differ.

For HTTP, the source-reconstructed binding must expose exact start/readiness authority `NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1`, and its typed binding digest must equal every verified execution-binding digest in the reopened FreshConfirmation. Matching only the WorldPlan, argv, projection, or stimulus is insufficient. This equality makes an inherited-listener binding structurally nonemittable even when its observed projection bytes happen to match.

The generated HTTP harness must use `node:net`, send exact HTTP/1.1 request bytes, retain bounded exact response bytes, and implement the same strict parser as the Go adapter. High-level `http.request` is forbidden because it normalizes header whitespace and cannot establish raw-wire parity. Keep literal loopback only; no TLS, redirects, cookies, compression, proxies, WebSocket, streaming body, or external hostname.

## Pure compiler and package authority

Use a cycle-free layout such as:

```text
internal/emit/node/
  authority/
  internal/publication/
  model/
  compiler/
  materialize/
  runtime/
internal/contractexec/
testkit/contracts/
```

The pure compiler imports no store, filesystem, clock, process, Git, environment, or network package. It accepts only sealed sanitized compilation input, validates the fixed source profile and bounded portable predicate, and returns all six path/mode/content triples in memory. Compile repeatedly under different absolute roots, locale, timezone, umask, newline assumptions, map insertion order, and fresh processes; require identical paths, modes, and bytes.

The application service may import `promotion`, `store`, compiler, and materializer. `store` may import only a leaf nonforgeable publication authority, never the parent emitter service. An opened head token or a pure draft alone cannot manufacture a residue capability.

## Exact recoverable bundle

The checked-in `tools/generate-p07-planning-example.mjs` output is only an executable schema/validator fixture. It proves that the planned six-file shape can perform one local Node-core child check and that the paired example digest is coherent; it has no store-bound ruling/source authority and is not the P07B emitter. Replace planning-fixture assumptions with the construction-safe implementation below rather than promoting the generator into product authority.

Generate exactly these nonempty regular files, all mode `100644`:

```text
decision.json
fixture.json
contract.test.mjs
harness.mjs
README.md
manifest.json
```

`decision.json` is a `CompiledDecision`, not another DecisionRecord. `fixture.json` contains the inert recoverable PortableSource payload. `contract.test.mjs` is an ordinary `node:test` entry point. `harness.mjs` is the vendored versioned second implementation. `README.md` states exact scope and nonclaims.

Bootstrap integrity without self-reference:

1. generate the other five exact files;
2. hash those five raw byte strings;
3. generate `manifest.json` covering exactly those five sorted paths, modes, counts, and raw SHA-256 digests;
4. hash `manifest.json`;
5. create the ContractBundle canonical body containing semantic metadata plus path, mode, byte count, raw-byte digest, and exact canonical base64 contents for all six sorted files;
6. compute the domain-separated ContractBundle digest externally.

The manifest never lists itself. No generated file contains the bundle digest. The outer bundle covers the manifest. On restart the ContractBundle alone is sufficient to reproduce every file exactly.

Each allowed tuple carries its complete ordered field/value body and no unconstrained `tuple_digest` member. If an implementation needs tuple identity, derive it with one locked domain-separated codec from that canonical body and verify it rather than accepting caller-supplied authority.

Use exactly the runtime-generic declared semantic profile `countershape-node-core-exact/v1`; do not infer or embed the compiler host's current Node version as execution proof. The closed seven-member profile commits the adapter domain, `NODE_REPO_SCRIPT_V1`, exact repository-relative entrypoint, and adapter-specific start profile. Its typed digest is exactly `sha256("countershape/v1/ContractSourceProfile\x00" || exact_canonical_seven_member_body_bytes)` with no LF and is always recomputed rather than caller-paired. Exact Node version, major, OS, architecture, executable-byte digest, and owned-probe digest belong only to a physically admitted `ContractExecutionTarget`. The declared source profile does not receipt any runtime tuple.

The emitter invents no current time, random value, absolute path, user name, temporary port, concrete candidate/ref/branch identity, declared secret value, host formatter drift, host-runtime fact, execution receipt, Wake artifact, or didrun interpretation as a new typed structural fact. Generated harness code adds no Countershape import/service binding, package install, shell command, registry dependency, or configured external-service dependency. Authorized PortableSource bytes, predicate values, and source-derived human text are retained as data and may contain sensitive, host-looking, or candidate-looking content; do not content-scan them into a redaction, secret-free, or security claim. Record exact construction facts such as sparse environment and absent ambient lookup; retain `confidentiality_established = false`. `external_service_binding = NONE` applies only to the generated harness; trusted subject code retains host-network authority, and HTTP may use loopback to that subject.

Keep the canonical body below the store's 1 MiB ceiling after base64. Lock explicit file, per-payload, list, tuple, allowed-tuple, aggregate raw-byte, and canonical-body ceilings. Refuse overflow before publication.

## Terminal publication and retryable materialization

Compile all six files in memory before the store transition. `AdvanceResidue` must:

1. acquire the exact study transition authority;
2. revalidate the complete expected `RULING` head token before creating an object or temporary file;
3. while holding the transition exclusion, durably publish and reopen the exact ContractBundle object;
4. advance the head to terminal `RESIDUE` only after the object is durable;
5. return a sealed residue bound to the store instance, study, predecessor DecisionRecord, bundle digest, and exact head token.

A stale loser creates no object, temporary file, or final output. A crash after content-addressed object publication but before head publication may leave an unreachable immutable object; document that exact possibility rather than calling it atomic rollback. A head must never reference a missing object.

No executable output directory exists before terminal residue publication. Materialization is a separate retryable operation from the reopened bundle:

- validate a clean explicit destination;
- create a private same-parent temporary directory;
- create each fixed file without following links;
- reopen and verify mode, count, and hash;
- atomically create-new rename the complete directory;
- fsync only to the exact natively tested policy;
- accept an already-existing exact directory idempotently;
- refuse partial/mismatched output without overwrite.

If materialization fails after residue publication, return `RESIDUE_PERSISTED_EXPORT_INCOMPLETE` or an equally precise typed result. The durable change remains true and a reopen may retry without recompilation or head mutation.

## Runtime target, finalized run, and immutable classification

Top-level runtime categories are disjoint:

- `ELIGIBLE_OBSERVATION` contains a projected exact tuple and becomes `CONFORMS` or `CONTRADICTS`;
- start, readiness, transport, timeout, output, parse/projection, orphan, and teardown failures become typed `INELIGIBLE_EXECUTION`.

HTTP `500`, CLI exit `2`, an absent optional field, or empty complete stdout may be eligible behavior. A harness failure and a contradiction may both make `node --test --test-reporter=tap` nonzero, but TAP diagnostics and closed machine codes must distinguish them. The A2.2 contract freezes one stderr record, `COUNTERSHAPE_RESULT_V1\t<outcome>\t<reason>\n`, with closed outcomes `CONFORMS`, `CONTRADICTS`, `INELIGIBLE_EXECUTION`, `MALFORMED_CONTRACT`, `TAMPER_DETECTED`, and `HARNESS_FAILURE`; only conformance exits zero, while stdout remains explicitly selected TAP. Do not substitute free-form error text for that result channel. This direct channel is inert operator UX, never target/run/execution evidence.

Never reuse the comparison-only `WorldInstance` chain here. Its authority begins with a 2–4-member historical `SelectedTreeSet`, exact `WorldPlan`, and opaque `CandidateExecutionBinding`; an arbitrary later target cannot enter that chain without copied-digest theater.

Create `ContractExecutionTarget` through this capability-only edge:

1. reopen and exact-verify the typed terminal `ContractBundle`, derive its PortableSource/profile/launch facts, and accept no caller-paired digest DTO;
2. pin one explicit Git ref once, inspect it under the exact source-derived policy, then materialize and reopen it through a new single-target path that consumes the opaque `InspectedTree` directly and verifies every path, mode, and byte;
3. admit one explicit absolute, symlink-resolved Node executable without `PATH`; hash/revalidate its bytes and execute one owned bounded probe for `process.execPath`, version, platform, and architecture;
4. durably allocate one fresh `CONFORMANCE` attempt marker whose nonce commits the bundle/source/tree/runtime authorities before spawn;
5. construct, publish, and reopen the target body only from those live capabilities; then execute only the private pinned materialization with the admitted runtime.

The target body carries the exact bundle digest; PortableSource/profile/source-profile digests; pinned object format and commit/tree OIDs plus recomputed `PinnedTreeIdentity`; portable-tree, materialization-policy, and materialization-manifest digests; fresh attempt facts; and admitted runtime facts. It has no self digest and no didrun or execution receipt. A parsed target is inert. Dirty working-tree bytes are never executed: run the once-pinned object materialization or refuse. A crash may leave an unreachable immutable target, never a study-head transition.

Run the process only from the reopened target capability. After materialization revalidation, process completion, teardown, and orphan inspection are terminal, construct and publish one `FinalizedContractRun`. Its canonical body has no self digest. It references the exact external target digest and repeats the target's exact attempt-artifact digest as a checked join; it owns the closed lifecycle digests, observation sum, exact projected tuple when present, and target-inventory/child-binding/import/service evidence. Its opaque constructor derives exactly one terminal disposition: `ELIGIBLE_CLEAN` only after clean lifecycle plus exact projection, otherwise `INELIGIBLE_CONTROL(reason)` from the exact closed control authority. A finalized run for another target or attempt is a hard `TARGET_RUN_MISMATCH`. No classification is eligible unless this object's disposition is `ELIGIBLE_CLEAN` and its observation is exactly `PROJECTED`.

`ContractExecution` is an immutable separately published classification. Its canonical body has no self `artifact_digest` and no duplicate bundle/tree/attempt/runtime, lifecycle, observation, tuple, control-reason, or standalone fields. It references only the external typed `contract_execution_target_digest`, the exact typed `finalized_contract_run_digest`, the derived conformance or ineligible class, and these nonfreshening facts:

```text
historical_execution_evidence_reused = false
choicepoint_freshened = false
study_head_advanced = false
```

The outer Go runtime service constructs the finalized run only from the reopened target capability and its own exact finalized attempt/process/capture/projection authority, then constructs the classification only from the reopened matching target and finalized-run capabilities. It independently executes the recovered PortableSource through the owner-controlled Go target path; it never parses the direct Node TAP/stderr ABI into authority. Process and observation evidence bind the target digest, never `world_instance_digest`. Git pinning, inspection, verified materialization, runtime admission, target publication, run finalization, or target/run matching failure refuses classification publication. Repeated runs allocate distinct target, finalized-run, and execution digests while the same study remains at the exact ContractBundle residue. Direct `<explicit-node> --test --test-reporter=tap <absolute-bundle-root>/contract.test.mjs` from the target-source cwd remains ordinary and requires no Countershape process or target object.

## Normative Go/Node parity corpus

Create one checked-in byte/result corpus consumed by Go and the generated Node implementation. Cover strict parsing, portable profile, selected-field extraction, tuple canonicalization, missing/empty distinctions, CLI completion, raw HTTP status/header/body projection, eligible/ineligible taxonomy, and expected errors.

Include duplicate JSON names, unsafe integers, `-0`, exponent/decimal forms, lone surrogates, invalid UTF-8, NUL/truncation, JSON key and array reordering, duplicate headers, absent and present-empty fields, empty bytes/list, exit `2`, natural signal termination, HTTP `500`, refused connection, hung readiness, output cap, malformed response separators/OWS, and teardown/orphan risk. A strict driver alone reads expected results; it passes only closed semantic input to oracle-blind Go/Node evaluators, with poisoned-expected and duplicate-input metamorphic tests killing any expected-echo path. Natural subject signal completion is eligible and distinct from exit; owner-induced timeout/cap/cancel termination is ineligible. If signal names are target-specific, keep that field outside portable support or bind it to an exact target-specific codec.

Kill mutations for last-key-wins JSON, numeric coercion, UTF-16 rather than UTF-8 key ordering, missing equals empty, signal equals exit, header trim/join/sort/deduplication, majority/name/first tuple choice, substring/regex/tolerance instead of exact membership, allow-many cross-product, assertion of an unselected field, high-level HTTP normalization, shell execution, Countershape import/service call, package-manager/external-host use, manifest bypass, bundle self-digest, digest-only file recovery, receipt-grade interpretation, and ineligibility flattened to contradiction. Any Go/Node disagreement blocks the P07B seal and capability claim; it does not authorize or refuse residue construction or publication.

## Physical standalone acceptance

For both CLI and portable-start HTTP bundles, use two distinct, non-nested roots: one exact six-file materialized bundle root and one prepared target-source inventory containing only the subject fixture code. The target inventory must not contain the exact generated contract subtree, but ordinary source files may share common names such as `README.md`. Countershape source, binary, package, dependency, import, and service-call path must be absent from both exercised closures, and the explicit Node executable must not be resolved through `PATH`. Use private HOME/TMP/state roots and an explicit sparse child environment. Pass named fake AWS/npm/SSH/Git/cloud sentinel values to the parent and prove they do not reach the child.

From the target-source inventory as cwd, run directly as:

```text
<explicit-node> --test --test-reporter=tap <absolute-bundle-root>/contract.test.mjs
```

Loopback remains available only for the HTTP subject. Do not claim host-wide Countershape absence, hostile containment, package-registry denial, network denial, confidentiality, or sandboxing unless an independent mechanism was actually run and receipted. The outer materializer verifies all six exact bundle members before publishing the directory. An intact `contract.test.mjs` entrypoint can verify the protected companion files before subject spawn, but cannot authenticate itself or the manifest; coordinated post-materialization replacement is explicitly outside the claim.

Prove: byte identity across roots; outer six-file materialization rejects any changed member; an intact entrypoint rejects protected companion-file inconsistency before subject spawn; relocation works; unselected field changes still conform; every selected-field change contradicts; allow-many remains tuple-based; custom `401` makes every eligible original HTTP candidate contradict; CLI selected precedence output conforms while unasserted diagnostics may vary; `REJECT_ALL`/defer/refine/legacy/stale/mismatched-source inputs create no output; materialization retries after an injected post-residue failure; and repeated ContractExecutions leave the head unchanged.

Run the physical matrix only for installed Node majors and native operating systems, naming exact version/OS/architecture separately. Cross-compilation and a generic source profile are not runtime evidence.

## didrun, commit, and handoff

Every load-bearing Go test, Node vector run, mutation suite, standalone inventory run, relocation, outer six-file tamper rejection, intact-entrypoint companion-tamper-before-spawn check, named-secret noninheritance, fault-injection, concurrency, race/vet/lint command, and runtime matrix entry must be `didrun run -- <command>`. Bind narrow claims to exact successful events. Entrypoint/manifest self-authentication, coordinated post-materialization replacement, unrun Node majors, Linux, Windows, external-network denial, hostile security, authenticity, confidentiality, maintainability, and framework portability remain `UNRECEIPTED` or explicit nonclaims.

Stage only P07B paths, commit without an AI co-author trailer, seal, and loop `NO_COLOR=1 didrun verify --strict` until exit 0. On failure, repair the actual defect, rerun affected commands through didrun, declare new claims, commit, reseal, and reverify. Never delete or weaken a test, vector, mutant, claim, threshold, or old receipt.

Update the status file and Mode C handoff with bundle schema, source profile, executed runtime tuples, target-inventory absence method, exact commands/events/claims/verbatim grades, commit/tree, materialization failure semantics, and remaining risks. Stop at selected-field DecisionRecord-only residue if source reconstruction, portable HTTP lineage, parity, separation, deterministic recoverable bytes, stale-before-publication, typed ineligibility, retryable materialization, or physical target-inventory absence fails.
