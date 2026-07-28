# Countershape threat model

- **Contract version:** U0 / `threat-model-v1`
- **Target:** single-user, trusted-local, narrowed Darwin reference instrument
- **Status:** design contract; P07B-C through C4I is sealed and strict-clean, C4K C3 Go-timeout maintenance is active and `UNRECEIPTED`, C4J remains parked, C5 remains blocked, and no security control is validated beyond its exact command/environment receipt
- **Review trigger:** update before any new source form, adapter, runtime OS, network mode, multi-user surface, or export field

The frozen C3-era status phrase “C3 exact-target publication is active” is retained only as historical checker compatibility; the current status immediately above governs.

## Security posture

Countershape executes repository code the operator has chosen to trust. That code runs with the operator's full user permissions and host network access. Fresh private roots, sparse environment, bounded capture, direct argv, and process-group cleanup are repeatability and damage-reduction measures. They are not hostile-code containment.

Before the first candidate execution, both CLI and studio must display this text without abbreviation:

> **Countershape will run trusted repository code with your user permissions and host network access. The temporary directory is for repeatability, not security isolation.**

The only v1 network mode is `HOST_ALLOWED`. No UI, log, report, or receipt may restate it as denial, filtering, isolation, or restricted egress. If the operator does not trust the repository and its declared runtime, the correct action is not to run Countershape.

## Security objectives

Within the stated boundary, Countershape is designed to:

1. execute exactly the supported regular/executable Git blobs selected by immutable object identity;
2. avoid ambient checkout filters, replacement objects, lazy network fetches, shell expansion, and inherited configuration in the reference studies;
3. keep detected infrastructure failures out of behavior outcomes;
4. make each evidentiary execution use new private roots and a new process lifecycle;
5. bound time and retained output and make uncertain process cleanup ineligible;
6. prevent an unrelated web origin from reading local evidence or forging a ruling through the loopback studio;
7. prevent candidate-controlled bytes from becoming active terminal, browser, accessibility, path, or report structure;
8. preserve blind-first separation between outcome facts and candidate/producer/support cues;
9. prevent stale, forged, noncompilable, or weakly scoped decisions from emitting source;
10. keep local evidence private by default and state that export confidentiality is unknown;
11. distinguish byte integrity, human attribution, current conformance, and external command receipts; and
12. fail closed when a boundary cannot be established.

These are design objectives, not a completed security review.

## Assets

| Asset | Why it matters | Authority |
| --- | --- | --- |
| Git object selection | determines which candidate bytes enter the experiment | Git + `TreeIdentity` |
| Materialized candidate root | contains executable repository bytes | `MaterializationManifest` for supported entries |
| Operator environment and host files | candidate code can access or mutate them | outside Countershape protection |
| Credentials and inherited secrets | accidental inheritance can create disclosure or behavior drift | sparse environment policy; not complete containment |
| Captured observations | can contain secrets, paths, customer-like data, and terminal controls | private `CapturedObservation` after capture policy |
| Projection definition and result | determines what values are compared and shown | versioned projection authority |
| Outcome map | determines the exact witnessed disagreement | canonical labeled-map digest |
| Decision record | records one locally caller-attributed exact-witness decision | strict session validation plus current Choicepoint CAS; actor authenticity is not established in U6 |
| Contract bundle | executable residue placed in a repository | deterministic emitter + bundle digest |
| Store heads | choose the current semantic lineage | authenticated atomic CAS |
| Loopback bearer and CSRF token | authorize local reads and mutations | server session boundary |
| didrun references | external evidence strings may affect human interpretation | didrun, never Countershape |

## Actors and assumptions

### In-boundary actors

- **Operator:** a local user who chooses the repository, refs, plan, projection, and final ruling. The operator can read the warning and is authorized to run the selected code.
- **Trusted candidate program:** may be buggy, flaky, noisy, stateful, or careless. It is not assumed to be well behaved, but it is not deliberately attacking the host boundary.
- **Reference fixture:** dependency-free checked-in Node-core code whose invocation and behavior can be instrumented for the two proof studies.
- **Local browser session:** receives a high-entropy capability token through a fragment and uses the exact loopback origin.
- **Unrelated web origin:** potentially malicious page attempting cross-origin reads, form posts, DNS rebinding, token leakage, or stale mutation.

### Explicitly excluded attackers

- repository code intentionally exploiting full same-user filesystem or network authority;
- a descendant deliberately escaping the process group through a new session or other host-specific technique;
- another process running as the same user and reading process state, local files, browser storage, or loopback traffic;
- browser extensions, developer tools, accessibility software, or malware with access to page contents or session storage;
- a compromised Git executable, Node runtime, OS kernel, filesystem, compiler, or package installed outside the checked-in reference fixtures;
- physical access, administrator/root compromise, memory forensics, side-channel attacks, denial of service, and supply-chain compromise of the toolchain;
- confidentiality of arbitrary captured content after the user exports or copies it; and
- authenticity of a human actor beyond local session attribution.

Exclusion does not mean harmless. It means Countershape must not claim protection against that actor.

## Trust boundaries

```text
Git object database
  -> filter-free materializer
  -> private attempt root
  -> full-permission candidate process on host network
  -> bounded capture/control gate
  -> private immutable evidence store
  -> authenticated loopback API
  -> inert studio renderer / explicit local export
  -> exact PortableSource reconstruction boundary
  -> deterministic standalone repository artifact
  -> exact-digest immutable nonhead target / finalized run / classification
```

Each arrow is a validation boundary, not a change of OS security principal. The materialized root, process, store, and server normally run as the same user. Content-addressed identity detects byte substitution inside Countershape's artifact model; it does not make the host trustworthy.

The blind-first boundary begins as a semantic information-separation boundary. U6 structurally omits candidate identity, producer/model provenance, total candidate count, per-outcome support counts, candidate ordering, and reveal material from the blind DTO. U8 must separately prove those fields do not enter the DOM, accessibility tree, visual metadata, or analytics labels. The number and facts of distinct outcomes necessarily cross because the caller must decide among them.

## Threat register

### T1 — Source selection and Git policy injection

**Threat.** A moving ref changes after display; a replacement object changes interpretation; a partial clone lazily fetches over the network; checkout/archive attributes or filters transform bytes; a hook or executable configuration runs; a blob OID and streamed bytes disagree.

**Required controls.** Resolve refs once to commit and tree OIDs. Record display spelling only as provenance. Use Git plumbing with replacement interpretation and lazy promisor fetch disabled. Never use checkout, worktree, or archive for evidence. Do not invoke filters or hooks. Prohibit network during materialization. Stream and rehash every blob using the repository object format. Pin and record the Git executable version. A changed display ref does not alter the pinned selection.

**Refusal.** Missing promisor objects, object mismatch, unsupported object format, changed selection before pinning, Git command policy uncertainty, or any network attempt yields typed source ineligibility.

**Residual risk.** A compromised Git executable or object database can lie. The repository may depend on content not represented by supported blobs. Runtime code can read arbitrary host files later.

### T2 — Path and filesystem escape

**Threat.** Absolute paths, `..`, NUL, invalid UTF-8, normalization/case aliases, reserved names, path collisions, symbolic-link chains, submodules, LFS indirection, or executable-mode drift cause writes outside the root or materialize different content than represented.

**Required controls.** Accept only Git modes `100644` and `100755`. Reject symbolic links, gitlinks, LFS pointer blobs, invalid/non-UTF-8 paths, unsafe components, duplicates, target-filesystem aliases/collisions, unsupported modes, and missing objects. Create a new nonce-owned mode-0700 root before materialization. Create regular files without following candidate paths and set only the supported executable distinction. Verify the resulting regular file bytes and manifest record.

**Refusal.** Any unsupported entry refuses the candidate; omission is not allowed to create a partial executable tree.

**Residual risk.** After execution begins, trusted candidate code can rename or replace files and roots using its user authority. Cleanup of a deliberately manipulated tree is outside containment guarantees.

### T3 — Ambient environment contamination and secret inheritance

**Threat.** Candidate behavior or disclosure changes because it inherits user configuration, credentials, proxy variables, locale, shell startup, `HOME`, temp state, or tool paths.

**Required controls.** Direct argv execution with no shell or command wrapper. Argv zero is one bare declared tool name; the remaining v1 logical arguments use a bounded closed grammar, while the resolved executable path is a later measured process fact. The declared `WorldPlan` environment is closed to exact public literals: `LANG=C`, `LC_ALL=C`, `TZ=UTC`, `NO_COLOR=1`, and `NODE_NO_WARNINGS=1`; runtime-owned private `HOME`, `TMPDIR`, and state-root values belong to `InstanceMeasurements` or process/materialization receipts, not stable plan bytes or structural `WorldInstance`. Record secret-slot presence without placing secret bytes in canonical plans. This profile avoids claiming that arbitrary caller-authored values are nonsecret. Reference fixtures require no inherited secrets, setup, package installation, registry, or external hostname. Resolved tool paths and versions belong to measurement artifacts and may participate in comparison assessment. Candidate count remains a declared budget input until U2 binds it to an exact roster.

**Refusal.** A required inherited secret or undeclared ambient dependency excludes the reference claim. Imported trusted setup remains a future surface and cannot inherit proof-fixture guarantees.

**Residual risk.** Environment sparsity does not stop access to host files, keychains, agents, sockets, or network credentials available to the same user. Candidate code can inspect host facts Countershape does not measure.

### T4 — Process escape, resource abuse, and orphaning

**Threat.** Candidate code hangs, emits unlimited bytes, forks descendants, ignores termination, keeps pipes open, or leaves a background process that contaminates later evidence.

**Required controls.** On Darwin, spawn direct argv in a new POSIX process group. Apply declared readiness, probe, total, output, and teardown budgets. Bound retained stdout, stderr, and HTTP body independently while safely draining or terminating. On stop, observed absence skips TERM and only clean observed presence authorizes it. The exact positive-presence `EPERM` state is retried within a bounded sub-budget; persistence or every other probe error retains uncertainty and authorizes no blind signal. After TERM, KILL requires clean continued presence after bounded grace. The owner then waits for the direct child, closes bounded drains, and probes the group. Sequential scheduling and new roots prevent known ordinary survivors from joining later trials.

**Refusal.** Timeout, output overflow, cancellation, incomplete drains, failed direct-child wait, uncertain group state, or teardown failure yields an ineligible control such as `TIMEOUT`, `OUTPUT_LIMIT`, `ORPHAN_RISK`, or `TEARDOWN_ERROR`. It never becomes an outcome value.

**Residual risk.** Process groups are cooperative cleanup, not CPU/memory/filesystem/network isolation. A descendant can deliberately create a new session and escape. The OS may retain resources not visible to the group probe. No Linux behavior is implied by a Darwin receipt.

### T5 — Network access and service confusion

**Threat.** Candidate code reaches external services, exfiltrates data, binds broadly, follows redirects, or mistakes another local service for the fixture.

**Required controls.** Publicly label the mode `HOST_ALLOWED`. Proof fixtures use no external hostname, TLS, redirect, proxy, cookie jar, compression, WebSocket, streaming body, or database. The inherited-listener fixture binds loopback and produces its existing fixture-owned readiness signal. Sealed P07B-A1 adds a child-bind profile that accepts exactly `COUNTERSHAPE_READY_V1 <port>\n` over inherited FD 3 followed by EOF, then uses the reported literal-loopback endpoint for the request. The allocated/reported endpoint and readiness/runtime facts remain process artifacts; `WorldInstance` retains only structural identity.

**Refusal.** A proof requiring external access, ambiguous readiness, or a response not tied to the owned fixture falls outside the reference study.

**Residual risk.** Candidate code retains normal host network access. Loopback does not prevent contact with other local services. A1's `child_reported_port` proves the accepted child-process-tree frame and subsequent endpoint use, not that the reporting PID owns the listener; arbitrary trusted code could report a decoy local service. Countershape cannot establish readiness noninterference for arbitrary imported services.

### T6 — Control failure laundering

**Threat.** Setup, start, readiness, transport, timeout, capture, projection, output, orphan, or teardown failure is encoded as HTTP/CLI program output and appears in an outcome card.

**Required controls.** Before any trial exists, `AssessComparison` consumes the complete envelope-bound measurement matrix. It persists `ComparisonAdmission` with the exact candidate roster and digest-sorted measurement set or persists `RejectedComparison`. Rejection creates no row token, trial, stable batch, or outcome map. After admission, only the centralized eligibility service can create a `StableBatch`; adapters cannot create a `CandidateOutcomeMap`. Trial control is a closed typed sum separate from domain capture. Only finalized attempts with required complete capture and successful teardown can project. Mutation tests must attempt to route both pre-batch rejection and every admitted-trial control variant into a projected value and fail.

**Refusal.** A rejected comparison is a pre-batch study refusal. Detected failures after admission classify the tagged trial/batch as `UNCOMPARABLE` or contribute to `INCOMPLETE` according to the declared batch contract.

**Residual risk.** A setup command can exit zero while semantically failing, or readiness can mutate hidden state without detection. Countershape separates detected controls, not every causal failure.

### T7 — Evidence substitution and false freshness

**Threat.** Old captured bytes are copied behind a new nonce, a prior process result is reused for confirmation, or a mutable file changes after hashing.

**Required controls.** No executed-observation reuse path exists in v1. Create an attempt artifact before spawn, allocate new roots, record a new lifecycle, and require fixture-observed invocation evidence outside the selected predicate. Confirmation rotates the schedule. Finalized artifact writes are reopened and rehashed. Pure canonical/projection transforms may reuse results only when the immutable input and implementation/configuration digests match. `duplicate_evidence_within_batch:false`, globally unique world/attempt digests, nonces, and admission-token digests establish structural identity checks only; none proves a physically new process, filesystem, host, cache, or network interaction.

**Refusal.** Repeated attempt IDs, roots, process receipts, capture digests without new invocation evidence, or stale ancestor digests prevent confirmation and Choicepoint readiness.

**Residual risk.** A malicious candidate can forge its own application-level echo. The curated fixture counter is proof-fixture evidence, not general process attestation.

### T8 — Projection manipulation and information loss

**Threat.** A projection hides a security-bearing field, ordinary parsers lose duplicate names or numeric distinctions, volatile placeholders erase a prior behavior difference, or Go and Node interpret bytes differently.

**Required controls.** Projection operations are pure, versioned, named, visible, and linked to captured source channels. Strict parsers reject duplicate names, invalid UTF-8, lone surrogates, negative zero, unsafe integers, and unsupported forms before typed values. A mutation that removes HTTP status or disclosure-bearing metadata in the flagship study must fail. The UI presents Captured-to-Projection operations one action away. A projection change creates a new semantic lineage.

**Refusal.** Parser rejection, missing required capture, unsupported field, or cross-language vector disagreement prevents the stronger compare or standalone claim.

**Residual risk.** Visible projection cannot prove a tolerated root, port, time, or host fact did not change earlier candidate control flow. The selected projection is finite and can omit future-important fields.

### T9 — Labeled-map degradation during reduction

**Threat.** A cache, reducer, API, or UI preserves only group count or membership while candidate outputs change, yielding a false local-reduction grade.

**Required controls.** Store, reducer, confirmation, API, and UI carry full typed `CandidateOutcomeMap` values or an opaque result from `ComparableForPreservation`, never a naked `PreservationMapDigest`. Comparability requires equal plan, envelope, stimulus-independent comparison basis, expected roster, eligible set, exclusions, and exclusion classifications. Only comparable maps use the complete sorted candidate-key-to-fingerprint digest: equality is `PRESERVES`, inequality is `CHANGES`; every comparability failure is `UNRESOLVED`. The digest is distinct from the evidence-bearing `OutcomeArtifactDigest`. Concrete per-repetition admission artifacts may differ across stimuli/phases, while each map requires one exact admission-digest set shared by all candidate batches. Display aliases and groups have no semantic identity. Construction of `ONE_MINIMAL_UNDER(...)` requires a durable-store authority over a complete direct-neighbor sweep whose result set is entirely typed `CHANGES` with the required evidence/content-digest nonoverlap; the grade itself does not establish physical execution or freshness. This constructor is absent before U5.

**Refusal.** Any unresolved neighbor, exhausted budget, cancellation, partial sweep, changed plan, changed eligibility, or absent durable completion record limits the grade to `BEST_KNOWN` or `UNCHANGED` as defined by the run.

**Residual risk.** A truthful local grade says nothing about global minimum, causal explanation, or human comprehension.

### T10 — Blind-step information leak and decision steering

**Threat.** Branch/model identity, producer provenance, candidate support counts, majority, or ordering leaks through payloads, DOM attributes, accessible names, card size, typography, analytics labels, or stable aliases and steers the ruling.

**Required controls.** The U6 blind DTO is a core semantic type, not yet an authenticated server response. It structurally omits candidate/ref/producer identity, total candidate count, per-outcome support, source order, and reveal material. Aliases bind the Choicepoint digest and exact projection fingerprint and expand all supporting outcomes. Deterministic order hashes only the unexposed projection fingerprint and therefore cannot read support count or candidate order. U8 must add equal visual weight, accessibility-tree parity, analytics discipline, complete warning copy, and the authenticated server boundary.

**Refusal.** Payload or accessibility-tree leakage, majority-coded order/weight, or a finalization path that skips provenance reveal blocks studio completion.

**Residual risk.** Behavior facts can reveal authorship. Projection-fingerprint-derived order can remain stable across Choicepoints containing the same facts; it is support-neutral, not anonymous or unlinkable. Distinct outcome count is visible. Visit records prove presentation only.

### T11 — Weak, forged, or stale rulings

**Threat.** A stale tab finalizes against an old Choicepoint; a malicious page forges a request; fields default to asserted; allow-many becomes independent value sets; selected fields fail to distinguish allowed from disallowed outcomes; or a noncompilable action reaches the emitter.

**Required controls.** U6 supplies strict session transitions, no default-selected fields, exact missing/empty separation, complete-tuple allow-many, custom-review binding, sealed compile eligibility, full store-token CAS, and stale-ready refusal. Its actor fields are local caller assertions. Bearer, Host, Origin, content-type, CSRF, and browser-server controls are U8 targets and remain `UNRECEIPTED` before that unit. Emitter input remains a future construction-safe compilable decision authority available only for `ALLOW_OBSERVED` or separately reviewed `CUSTOM_EXPECTATION`.

**Refusal.** Stale or foreign store authority, missing reveal, omitted or empty selections, `EMPTY_SELECTED_FIELDS`, invalid field path, `AMBIGUOUS_SCOPE`, tuple cross-product, observed-tuple custom expectation, or noncompilable emitter coercion is refused. `REFINE` specifically returns `REFINE_REQUIRES_SUCCESSOR_STUDY` before semantic-object publication or head mutation. `REJECT_ALL` and `DEFER` may create noncompilable DecisionRecords but cannot reach an emitter.

**Residual risk.** The exact selected fields intentionally permit changes in context-only fields. The ruling does not express broader intent or approve a candidate implementation.

### T12 — Loopback disclosure and request forgery

**Threat.** A malicious site reads evidence through permissive CORS, posts a form, exploits DNS rebinding, uses `Origin: null`, guesses an identifier, replays a stale mutation, or obtains a token from URLs/logs.

**Required controls.** Bind literal `127.0.0.1`; require exact `Host` including the allocated port. Deliver a high-entropy token in a URL fragment, transfer it to session storage, remove the fragment, and never place it in query strings or logs. Require bearer authorization for every read and write. Require exact Origin, JSON content type, CSRF, and CAS for mutations. Emit no permissive CORS response. Candidate content never becomes a navigable URL. Use restrictive CSP for bundled assets.

**Negative cases.** Missing/wrong bearer, form POST, failed preflight, foreign Origin, `Origin: null`, rebinding Host, guessed study ID, stale digest, wrong content type, and replayed CSRF all fail before reading or mutating semantic state.

**Residual risk.** Same-user processes and browser extensions can steal tokens or read storage and are excluded. A compromised browser can act as the operator.

### T13 — Candidate-output injection

**Threat.** Captured bytes contain ANSI/OSC terminal controls, bidi controls, invalid text, HTML, `</script>`, forged Markdown headings, path separators, huge strings, or text that contaminates accessible names and reports.

**Required controls.** CLI previews visibly encode terminal controls and never write candidate bytes as live control sequences. Browser surfaces use inert text nodes, bounded rendering, and explicit visible encodings for controls/bidi where necessary. Candidate values are not interpolated into HTML, URLs, CSS, DOM IDs, script blocks, Markdown structure, file names, logs, or accessible control names. Exports use structural serializers and restrictive CSP. Byte limits apply before UI/report rendering.

**Refusal.** Invalid textual forms remain byte evidence or typed rejection according to the capture profile; they are not repaired into identity-bearing strings.

**Residual risk.** Copying candidate text into another tool may reactivate controls. Accessibility testing cannot prove every assistive technology treats encoded content identically.

### T14 — Export disclosure

**Threat.** A token appears under an unknown key, in free text, split or encoded, or in a field the user selected. Rule-based redaction misses it and an export leaks it.

**Required controls.** The default local export omits captured bodies and private artifact bytes. It contains only fields the user previewed, records all minimization/redaction operations, structurally escapes content, and prominently displays `CONFIDENTIALITY NOT ESTABLISHED`. An export preview is mandatory. `--include-raw` is a separate explicit dangerous action and may include unchanged captured bytes only when they actually are unchanged.

**Refusal.** Failure to render the warning, unknown export field, structural injection, or accidental private-artifact inclusion blocks export creation.

**Residual risk.** Known-secret fixtures establish only fixture coverage. No generic redaction can establish absence of arbitrary secrets. The operator owns handling after export.

### T15 — Standalone semantic drift and artifact substitution

C2 storage mechanics cannot issue this authority. That invariant applies to every target, interlock, and private-evidence record described below.

C1's `internal/contractexec/model` is the sole semantic parser/constructor authority for the three execution bodies. Its JSON schemas are closed syntax projections and intentionally cannot express every evidence-role, process/spawn/capture, scope-precedence, aggregate, and recursive-ceiling rule. Schema-only acceptance therefore grants nothing. C1 stores the admitted absolute Node path, executable-byte digest, mode/count, measured `process.execPath`, and runtime tuple as checkpointed path/content identity; this is not descriptor-bound or inode-bound execution proof and does not close coordinated same-user replacement between revalidation and spawn. C1 likewise validates only the closed start-error structure and typed evidence references; C4 must establish physical evidence content and chronology.

**Threat.** A legacy whole-projection ruling is mislabeled as selected-field authority; caller-paired bytes/digests substitute for exact runnable source; schema-valid C1 bytes are treated as semantically valid authority despite cross-field disagreement; an inherited-listener HTTP observation is silently translated into child-bind execution; Node accepts bytes Go rejected, normalizes or joins headers differently, conflates missing/empty, follows redirects, changes signal semantics, asserts a context-only field, depends on Countershape or a registry, or the bundle/runtime is changed after its checkpoint.

**Required controls.** P07A preserves exact historical adapter wires and derives portable tuples only after roster verification and complete projection-binding resolution. Sealed P07B-A1 provides a closed byte-complete source constructor that reconstructs its supplied exact plan, minimized stimulus, profile, runner/start/readiness/capture authorities, and execution binding; it also reaches a physical HTTP Observation under the new child-bind/pipe-readiness profile. A1 does not join that object to the current ruling/confirmation, and its `closed_facts` are declarations rather than static analysis, secret scan, dependency absence, network denial, confidentiality, or containment. Sealed A2.1 reopens and revalidates current store authority, freshly reinspects the reparsed ruling/profile, compares every confirmation execution binding, independently retranslates the reopened proofs, revalidates the selected-tuple partition, and produces only a private authority-narrowed compiler input. Accepted A2.2 deterministically compiles a recoverable six-file ContractBundle and exercises the normative Go/Node corpus for generated strict parsing, field selection, CLI/raw-HTTP capture, projection, and error taxonomy; `manifest.json` covers the other five files while the outer bundle covers all six. Sealed P07B-B source adds an opaque node-issued terminal store transition and a store-bound, no-raw-input native materializer with exclusive no-follow publication, exact-existing convergence, immutable mismatch refusal, and typed ambiguity; only its named receipt rows inherit grades, and hostile same-user containment remains a nonclaim. Initial launch authority remains only logical `node` plus one exact repository-relative JavaScript entrypoint; generic repository executables are excluded.

Before spawn, the nonhead `ContractExecutionTarget` is constructed only from the reopened bundle/residue, one live Git-issued pinned/inspected/verified private materialization, a fresh durable conformance attempt, an opaque measured Darwin boot session, and an admitted/revalidated explicit Node runtime. The only admitted boot profile is `DARWIN_KERN_BOOTSESSIONUUID_V1`: read read-only `kern.bootsessionuuid` twice through a bounded fixed-name Darwin system-call edge, validate each 36-character hyphenated nonzero UUID, lowercase each sample, and compare the normalized values. Raw UUID/digest/timestamp constructors, caller-authored boot values, the retired clock-derived profile, `kern.boottime`, clock/uptime/PID/file/process-local fallbacks, `/usr/sbin/sysctl`, PATH, malformed or unstable data, and unsupported platforms fail closed. The digest is only an OS-reported boot-epoch checkpoint, not host identity, a security principal, physical-reboot proof, prior-child absence, VM non-rollback, kernel trust, clone resistance, or collision impossibility. C2's canonical object, typed private witness, and exact relationship produce only a durable inert storage record; C3 alone may rejoin it with the live prerequisite graph and return `OfficialTarget`. The C4 contract runner alone may consume that authority after a fresh independent boot measurement, durably acquire a store-private boot-session `ExecutionInterlock`, create a new target-keyed StartClaim beneath the same fresh generation, and receive one nonreopenable process-local RunPermit. An existing claim/held state, losing racer, copied generation, restart, or ambiguous mutation grants none. Low-level process mechanics never sees that permit or semantic body. Interlock/claim state proves no spawn, exit, cleanup, or survivor absence; a separate SpawnObservation records `START_ERROR` or observed child PID. Missing observation cannot enter an FCR and leaves the interlock held on that boot. A fresh target does not clear ambiguity; reset requires explicit operator action plus a differently measured boot-session identity, or no further subject spawn is admitted, and the difference still does not establish survivor absence. Every persisted `CLEAR` must reopen one exact clear receipt; a split where CLEAR is durable but its receipt is missing or invalid remains ambiguous and blocks admission. The receipt reconstructs the trusted writer's state transition but does not prove child absence or independently establish the physical cause/run. The five-domain standalone run measures target inventory, child bindings, import resolution, service bindings, and named parent-secret sentinel inheritance as COMPLETE, PARTIAL, or VIOLATED. It establishes no host-wide absence, network/registry denial, listener ownership, containment, or confidentiality. After terminal process and scope closure, a separate `FinalizedContractRun` binds the exact target/attempt to bounded typed summaries and private-evidence refs. The private manifest, exact FCR, target-to-run record, CLEAR state, and clear receipt must be durable and reopened before interlock release is complete. A separately persisted classifier-profile-bound `ContractExecution` then derives `CONFORMS`, `CONTRADICTS`, or `INELIGIBLE_EXECUTION` without accepting a tuple or process/scope reason, never starts or reruns the contract subject or consumes subject admission, and never advances the terminal study head. Classification recovery still performs live target validation, including two bounded owned Node runtime probe processes. Direct generated Node TAP remains independent black-box/operator evidence.

C4 narrows the revalidation-to-spawn window after owner acquisition: it creates a detached bounded closure context, performs the final full `ReopenOfficialTarget`, requires exact freshness, replaces the retained handle, installs cleanup, and only then consumes the permit and starts. A repository-wide code-only inventory permits exactly the store definition and the runner's production `AcquireContractRunOwner` reference. Physical tests retain the historical real `CONTRADICTS` branch and add a separate real-ruling `CONFORMS` branch with an exact two-field `argv` tuple. These are cooperating-source and fixture controls, not resistance to arbitrary same-user code.

CLI detector authority is also explicit. Clean target inventory is available only for the exact enrolled two-file fixture. Both inventory passes use Darwin all-component no-follow directory/file opens, 128-entry directory batches, exact-admitted-size streamed SHA-256, and before/open/after identity checks; they refuse beyond 20,000 entries, 32 MiB cumulative relative-path bytes, depth 128, 64 MiB per file, or 64 MiB aggregate. This prevents a child-created huge/deep tree from driving unbounded post-run reads, but does not close same-UID swap-and-restore races between observations. Child binding uses a candidate-written canonical receipt bounded to 64 KiB and joined to a fresh attempt ID plus exact logical argv; import/service evidence counts connections to named Unix-socket canaries; sentinel evidence proves omission only of named environment keys. Target mutation is a hard refusal rather than a positive detector result. Private manifests preserve logical-reference multiplicity and enforce 16 blobs/64 MiB. Classification reopens the exact StartClaim and revalidates durable finalized-run release immediately before persistence, so deleting that relation blocks publication despite a cached release bit.

The evidence envelope is admitted before owner acquisition rather than discovered after a child runs. At the maximum legal 16 MiB stdout and stderr caps, C4 reserves 48 MiB: one shared raw capture frame with 1 MiB overhead, one 3 MiB raw projection frame, and twelve canonical summaries capped at 1 MiB each. Drain and Captured must share byte-identical content and one stored digest; the complete draft is uniquely re-counted before persistence. Evidence, scope, attempt, and short-socket direct rosters are read only to expected cardinality plus one through retained descriptors. Production cleanup deletes only identity-checked named leaves and proven-empty roots; it does not walk or recursively sweep foreign subtrees. Deep foreign residue is preserved on refusal/ambiguity. This bounds Countershape's own work but does not stop a same-UID process from creating residue, reading private bytes, or racing the ordinary final `Lstat`-to-`Remove` interval.

A copied canonical body, copied C2 relationship file, or complete filesystem copy does not carry a live in-memory capability. The current local store nevertheless cannot detect or prevent an offline copy from being opened independently as a clone. Likewise, private placement and a durable no-serve purge tombstone establish neither confidentiality nor secure deletion, and `ClassifierProfileDigest()` identifies the declaration rather than classifier implementation bytes. These are explicit nonclaims, not controls inferred from successful reopen tests.

**Refusal.** Legacy ruling, source/world/profile mismatch, nonportable start profile, generic repository executable, vector disagreement, stale publication, unrecoverable file body, manifest mismatch, copied or parsed target authority, OID/tree-identity mismatch, missing verified materialization, dirty working-tree execution, reused attempt, unadmitted runtime, target/run mismatch, forbidden dependency resolution, an observed forbidden binding under the named service-binding detector, or harness/control ambiguity blocks the applicable target, execution, or standalone claim. Post-residue materialization failure remains a typed retryable export failure and cannot be reported as if no residue was published.

**Residual risk.** Hashes prove byte integrity, not authorship, authenticity, confidentiality, or long-term maintainability. `OfficialTarget.Valid()` is structural, not fresh; C4's spawn-adjacent reopen reduces but does not eliminate coordinated same-UID substitution/TOCTOU. The fixed runtime probe proves the explicitly selected cooperating executable reported the admitted tuple, not that it is vendor-authentic Node or unable to emulate the response. A conflicting same-attempt target race is first-durable-writer-wins and has no semantic arbitration. The target-inventory check is not host-wide absence, network denial, registry denial, or containment. Candidate-written invocation evidence is not hostile-process attestation. Node runtime compromise, repository integration changes, and coordinated path/object substitution remain explicit nonclaims.

### T16 — Receipt or provenance overstatement

**Threat.** Wake runtime history becomes product state; a didrun grade is parsed or summarized upward; a green command is assigned to another capability or OS; missing evidence is inferred from UI success.

**Required controls.** Wake data is restricted to inert candidate provenance. Countershape has no Wake events, effects, epochs, gates, session recovery, replay, fork, or scheduling state. didrun reference fields are opaque strings and round-trip unknown values verbatim. Every claimed OS, Node major, viewport/state, security behavior, and timed metric maps to its own exact receipt. Missing evidence is `UNRECEIPTED`.

**Residual risk.** Receipts do not validate novelty, comprehension, bias reduction, market adoption, maintainability, or review completeness.

### T17 — Verifier concurrency and cross-run cache substitution

**Threat.** Two cumulative verifiers overlap and create the same load-sensitive timing failures that the serial profile is meant to prevent, or a persistent compilation cache is described as fresh authority even though Go does not detect imported C-library changes.

**Required controls.** The Darwin verifier acquires one private mode-`0600` O_EXCL ownership record before plan/tool/cache work, fails fast for a live or indeterminate owner, treats an absent owner as a review-required stale refusal, revalidates exact descriptor/path metadata and canonical content before success and release, and never race-deletes an uncertain or stale record. PID reuse may conservatively refuse availability. Current general direct package jobs are closed at one; nine frozen whole sensitive packages plus every nested Go command remain serial, and the exact `go list ./...` union is proved before tests and re-proved unchanged after both lanes. Every other initially discovered admitted-module package is intentionally general rather than “unclassified.” C1V's sealed two-job result remains historical evidence, but its explicit recurrence refusal fired after C4V observed a second independent late A2 AST deadline; C4V therefore reverts direct build, vet, and general testing to `p=1` without extending deadlines, removing tests, or changing `GOMAXPROCS=2` and test `-parallel=2`. Focused stability evidence uses a tracked named-case matrix, exact Test/Fuzz/Example state parsing, and explicit admitted-tool path/digest rows; ad hoc regex runs cannot qualify. The last current row must remove the private run root and release the exact lock before terminal PASS. `GOCACHE` stays fresh per run because current Countershape includes cgo; C1V and C4V make no warm-cache or automatic crash-recovery claim.

**C4I recurrence repair.** A later post-load A2 AST worker deadline recurrence exposed a narrower defect: one JavaScript digest drift sent all six profiles, including a large unchanged harness, through one nested worker even when the expected violation was already known. C4I still reads and hashes all six sources, but only drifted profiles enter the worker. The worker must return exact ordered AST rows matching the requested path list; malformed, duplicate, missing, foreign, or out-of-order rows and `P07B_A2_JS_AST_QUERY_FAILED` are fatal infrastructure failures before expected-hostile matching. Clean and non-JavaScript controls require zero rows, each of 32 pinned relevant mutants requires one exact singleton, and a three-profile hostile proves exact ordered batching. This removes the all-profile path from ordinary drift analysis without changing the fixed 60/90-second deadline hierarchy, `p=1`, `GOMAXPROCS=2`, test `-parallel=2`, fresh caches, lock, package roster, assertions, or cumulative verifier order. Qualification requires three standalone A2 passes plus three complete post-load cumulative passes on one staged tree; the optimization is not a scheduler cure, product authority, or security authority.

The same final gate exposed a deterministic test-fixture mode defect: an exclusive create requested `0644`, caller `umask 077` produced `0600`, and the human-surface checker refused the mismatch. C4I now creates below fresh mode-`0700` directories, changes only the successfully and exclusively created path to exact `0644`, verifies its type/link-count/mode/bytes, preserves preexisting regular/symlink/directory entries, restores caller umasks `077` and `022`, and cleans partial roots. The enclosing private directories prevent cross-user traversal during this fixture construction, but the path-based chmod does not establish hostile same-UID replacement resistance, a general materializer, or a product security boundary. Product materialization, private store mode-`0600` policy, ancestor no-follow guarantees, and coordinated replacement remain unchanged.

The third C4I final attempt exposed a different verification-fixture race: the original owned group could reach its unchanged `100 ms` execution timeout before a nested child changed process group and published the evidence that the test immediately required. The repair uses canonical mode-`0600`, hash-bound `PREPARED → AUTHORIZED → READY → RELEASE → RELEASED` records below one private state root. The child starts its self-lease before publishing `PREPARED` while still contained; the test registers cleanup authority and independently observes original-group membership before `AUTHORIZED`; exact stdout/stderr markers, a final record, and independent PGID plus `setsid` SID observations establish `READY`; `RELEASED` acknowledges the exact `RELEASE` hash join before voluntary exit. The timer still starts before the group-owned callback and is never reset, but the callback can delay physical teardown while it establishes this test precondition. A `15 s` self-lease and bounded reap observation avoid identity-unsafe numeric-PID signaling after failure. This proves causal test construction and exact cleanup evidence only; it does not improve product containment, close Darwin PID/PGID reuse, resist hostile same-UID state replacement, or establish a wall-clock enforcement guarantee.

**C4K nested Go test-deadline repair.** Only exact `c3-official-target` receives `-timeout=12m`, giving the bounded `720 s → 900 s → 1080 s → 1200 s` ladder from inner Go test through outer profile process and clean C3 defensive self-test to the cumulative-verifier child ceiling. The other four C3 profiles and all six C4 profiles receive no inner timeout flag. The 18-minute clean-selftest ceiling is deliberately below the unchanged 20-minute cumulative parent so each layer retains diagnostic margin. C4K adds no retry and changes no package partition, parallelism, assertion, cache policy, cumulative-row order, product behavior, or security authority. The larger inner deadline addresses one observed implicit-default cutoff; another overrun is a diagnostic failure, not automatic authority to lengthen it again.

**Refusal.** Concurrent owner, malformed/replaced/disappeared lock, stale lock, malformed/duplicate/foreign package row, missing frozen-sensitive package, during-run package drift, nested parallel inheritance, cleanup/release failure, or any repeated timing failure blocks the verifier. Except for the exact reviewed C4K nested Go deadline above, no timeout, assertion, package roster, or receipt label is weakened to recover throughput.

**Residual risk.** O_EXCL is not a kernel lease. Abrupt termination can leave a stale file requiring inspected manual removal, same-UID replacement remains outside containment, PID reuse can false-busy, and the one-job current profile is only a conservative local Darwin/arm64 stability ruling—not proof that two-job execution caused either late AST timeout. A future persistent cache needs a broader compiler/SDK/C-library authority plus fresh cgo evidence.

### T18 — Store tamper, filesystem semantics, and durability overclaim

**Threat.** A stale or forged token wins, a loser publishes an orphan successor, a same-UID process edits object or head bytes, filesystem semantics differ from local Darwin, or fsync usage is described as proof of power-loss recovery.

**U6/C2 controls.** Private mode-0700 directories; mode-0600 object, head, relationship, interlock, receipt, manifest, pack, and purge-intent files; path, symlink, case-alias, kind, canonical-byte, and digest validation; retained directory identity reasserted at relevant later authority and mutation boundaries; Darwin hard-link create-if-absent object/typed-witness publication; per-study and store-wide contract `flock`; full-token comparison before successor-object creation; exact typed relationship joins; store-instance-bound inert records; receipt-backed CLEAR; atomic head/interlock replacement; and parent-directory synchronization plus exact reopen before any exact-visible create-once record is promoted.

**Refusal.** Foreign, incomplete, or stale token; illegal revision, stage, kind, relation, or predecessor; substituted directory; malformed or oversized bytes; mismatched kind/digest/parent/profile/boot; nonprivate mode; alias; receiptless CLEAR; wrong manifest range/hash/roster; unexpected private loss; or post-publication corruption fails closed.

**Residual risk.** These controls govern conforming Countershape writers on the exercised local Darwin filesystem. Mode `0600` does not stop the same UID. There is no hostile same-user race defense, NFS or network-filesystem claim, forced-power-loss receipt, archived head chain, garbage collection, deletion or retention guarantee, or administrative recovery protocol.

## Local server control contract

| Request class | Bearer | Exact Host | Exact Origin | JSON content type | CSRF | CAS current digest |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Static embedded asset | no | yes | n/a | n/a | n/a | n/a |
| Semantic read | yes | yes | n/a | n/a | n/a | n/a |
| Progress stream | yes | yes | n/a | n/a | n/a | n/a |
| Provisional decision write | yes | yes | yes | yes | yes | yes |
| Reveal / finalize / emit | yes | yes | yes | yes | yes | yes |

Exact Origin means the literal server-created `http://127.0.0.1:{port}` origin. `localhost`, alternate IP spellings, wildcard ports, `null`, or absent Origin on a browser mutation are rejected. API clients outside the browser require a separately explicit CLI authorization path; weakening browser checks is not the compatibility mechanism.

## Data handling contract

- Private store directories are mode 0700 and private artifact files are mode 0600 where supported on the receipted Darwin filesystem.
- C2 uses `private-captures/contract-runs/{manifests,packs,purge-intents}` for bounded operational evidence retention. It is not a public artifact API, confidentiality boundary, or stable migration format.
- Secret values never enter `WorldPlan`, canonical digests intended for display, URLs, routine logs, or receipts. A secret-slot policy records presence without the bytes.
- Captured evidence can still contain secrets emitted by the program. It is private by default and subject to byte caps.
- Projection is not redaction unless an operation is explicitly named as redaction. Transformed bytes are not called raw.
- The studio shows only server-approved typed previews; it does not fetch arbitrary artifact paths.
- Export paths are created by the operator under a safe tool-owned name. Candidate bytes cannot choose a path.
- A valid purge intent makes bytes logically unavailable before best-effort pack cleanup. This is neither secure deletion nor proof that copies, filesystem snapshots, or same-UID readers lack the bytes. Retention at finalization is not perpetual availability.

## Security refusal policy

Security-sensitive uncertainty lowers capability rather than changing labels:

- unsupported Git content -> candidate `UNCOMPARABLE`;
- uncertain process cleanup -> `ORPHAN_RISK`, ineligible evidence;
- incomplete capture or output cap -> typed control, not truncated behavior equality;
- unauthorized/stale write -> no semantic object;
- ambiguous selected fields -> `AMBIGUOUS_SCOPE`, no bundle;
- parity or absence failure -> DecisionRecord-only residue;
- server security or evidence-parity failure -> CLI-only surface;
- export structural or warning failure -> no export;
- runtime not natively exercised -> exact environment item `UNRECEIPTED`.

No fallback may silently keep the stronger public claim.

## Verification obligations by unit

| Unit | Load-bearing threat work | Required negative focus |
| --- | --- | --- |
| U0 | this boundary, schemas, refusal examples | planning mutation checks |
| U1 | strict canonicalization, construction-safe sums | duplicate keys, numeric/Unicode loss, illegal transitions, required mutants |
| U2 | Git policy and Darwin process lifecycle | replacements, fetch/network, paths/modes, byte caps, descendants, orphan risk |
| U3 | CLI eligibility and sparse environment | control laundering, alternating output, inherited HOME, terminal bytes |
| U4 | HTTP ownership and readiness | wrong service, transport/response split, body cap, contamination fixture |
| U5 | exact-map reduction | same-shape/different-map trap, cancellation, unresolved final neighbor |
| U6a / P06 | CAS, physical confirmation, Choicepoint, blind semantic kernel, rulings | stale/forged authority, reused evidence, blind identity/support/order leak, cross-product, context assertion, noncompilable promotion |
| U6b / P07A | exact projection-wire translation and selected-field ruling authority | profile substitution, fingerprint migration, malformed historical wire, bytes/list collapse, selected-field widening, legacy upgrade |
| U6c / P07B-A1 | self-consistent source reconstruction, logical-Node runner profile, child-ready frame grammar, portable HTTP Observation | source/authority mismatch, normalized alias, missing bytes, inherited-listener substitution, malformed/extra readiness frames, readiness-only finalized controls; no listener-ownership or compilation claim |
| U6c / P07B-A2.1 | current-ruling/source/Choicepoint/confirmation/proof join, selected-tuple partition, private authority-narrowed input, fresh-process preparation reconstruction | cross-paired source authority, wrong store/predecessor, binding-roster drift, tuple widening/cross-product, public authority exposure, store write |
| U6c / P07B-A2.2 | raw-wire Go/Node parity and strict recoverable bundle compilation | normalized HTTP, unrecoverable/partial bundle, semantic cross-pair, generated-asset tamper, oracle leakage |
| U6c / P07B-B | terminal recoverable publication and retryable exact native materialization | stale/full-token publication, missing predecessor, foreign store, store/path alias, symlink/hardlink/case alias, partial output, fallback rename, sync/close ambiguity |
| U6c / P07B-C | capability-only nonhead target, normalized Darwin boot-session UUID, intent-only at-most-once spawn admission, bounded finalized run with separate five-domain scope, and classifier-profile-bound execution | retired/malformed/unstable/caller-authored boot profile or fallback, generic executable, copied/parsed target, dirty worktree, permit reissue, ambiguous-spawn reuse, runtime substitution, metadata-shaped absence, forbidden dependency, contradiction/ineligibility collapse, private-evidence leak, or hidden execution head |
| U7 | decisive studies | false one-shot result, fresh invocation evidence, all decision actions |
| U8 | loopback and blind renderer | auth/Host/Origin/CORS/CSRF/CAS, identity/support leak, injection, mobile parity |
| U9 | export and packaging | secret fixtures, structural injection, raw opt-in, absent warnings, final receipt map |

Security checks are native-environment claims. A Darwin result does not receipt Linux. A screenshot does not establish request forgery controls. Automated checks do not replace visual inspection or independent security review.

## Security claim limits

Permitted, when exactly receipted, are narrow statements such as:

- “The tested Darwin materializer accepted and verified only Git modes `100644` and `100755` in the named fixtures.”
- “The named localhost negative suite rejected the tested missing/wrong bearer, Host, Origin, CSRF, content type, and stale-CAS requests.”
- “The tested default export omitted the named fixture secrets and rendered the named injection corpus inert.”
- “The tested process owner returned `ORPHAN_RISK` for the named uncertain-cleanup fixture.”

These do not establish hostile-code containment, comprehensive secret removal, complete web security, production readiness, or security-review completeness.

<!-- countershape-validator: allow-prohibited-terms begin -->
The phrases “sandboxed,” “network blocked,” “safe to run untrusted PRs,” “safely shareable,” and “security approved” are prohibited upgrades. So are claims that symlink support exists in this run or that cached/reused execution evidence proves freshness.
<!-- countershape-validator: allow-prohibited-terms end -->

## Required human tail

Before production use, the owner must commission an independent security review covering every implemented component at that time, including the source parser/materializer and process lifecycle plus the localhost application, artifact permissions, export pipeline, generated harness, dependency/release chain, and platform-specific behavior if and when those later surfaces are built. They must also define vulnerability intake, supported versions, patch SLAs, artifact retention/deletion, release signing, reproducible builds, and maintainer ownership.

This U0 model is a falsifiable boundary for implementation. It is not that review.

## C4J bounded verifier timeout authority

C4J repairs a verifier scheduling ceiling, not a product security boundary. The child runtime remains parent-owned: callers may omit policy or pass one exact frozen ordinary/null-prototype record with the sole own data key `timeoutMS`. Validation rejects inherited keys, accessors, extra string/symbol keys, unfrozen or exotic records, coercion, and every numeric value except the admitted `1800000`; omission preserves the `1200000` default.

The verifier policy is a frozen null-prototype record outside `currentSteps`, looked up with `Object.hasOwn`. Its only keys are `architecture-p07b-c-c5` and `architecture-p07b-c-c5-selftest`. The runtime performs both authority revalidations and exactly one spawn; there is no retry, fallback, clamp, or changed inner deadline. The remaining risk is explicit: 30 minutes may still be insufficient under future load, and successful C4J cumulative runs do not prove C5 behavior.
