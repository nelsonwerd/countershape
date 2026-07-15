# Countershape red team: the reference system survives only as a narrower falsifiable instrument

- **Posture:** attempt to kill the controlling system, not improve its pitch
- **Inputs:** all six deep-dive lanes, `07-SYNTHESIS.md`, the current `CONCEPT_BRIEF.md`, and the Wake overlap audit
- **Date:** 2026-07-14
- **Verdict:** **NARROW, THEN PROCEED CONDITIONALLY**
- **Confidence:** 8/10 that the truth kernel is worth building; 6/10 that the narrowed local instrument is feasible; 3/10 that it improves real review; 2/10 that the complete U0-U9 system can be validated in this run without honest cuts

## Executive kill attempt

Countershape is not dead, but the synthesis still grants it three things it has not earned: a meaningful notion of compatible real worlds, a safe-enough imported-tree execution story, and a decisive demo that proves more than a carefully dressed fixture diff.

The semantic core is sound only when stated negatively. Countershape can record that, under a finite projection and a finite set of fresh executions, several exact tree candidates produced different exact selected values. It can locally reduce a typed stimulus while preserving the complete candidate-to-fingerprint map. A human can then author a selected-field predicate for that one witness. None of those operations proves that two worlds were behaviorally equivalent, that an unobserved dimension was irrelevant, that the reduced case captured the cause, or that the selected predicate expresses the user's broader intent.

The largest unresolved problem is world compatibility. A candidate can read its absolute working directory, allocated port, PID, clock, filesystem device, tool path, hostname, scheduler timing, or any host file. A projection can replace an ephemeral root in captured text, but it cannot prove that the root did not change a prior branch in the program. Two processes that print the same placeholder may have taken different internal paths. A `WorldInstance` compatibility predicate is therefore not a proof of behavioral compatibility. It is a declared **comparison envelope** over dimensions Countershape measured and chose to treat as non-disqualifying. That distinction must become public language and a typed limitation, not remain an implementation detail.

The second problem is duplicated semantics. The Go core owns strict parsing, capture eligibility, projection, and process lifecycle, while every six-file Node bundle reimplements a subset of parsing, HTTP/CLI execution, sparse environment, byte caps, process-group teardown, and projection. That is a second truth implementation. `JSON.parse` loses duplicate names before the contract can reject them; convenience HTTP APIs can join headers or follow redirects differently; Node and Go can disagree about invalid UTF-8, signals, and empty versus absent. "Countershape absent" is valuable, but absence does not imply semantic fidelity. Cross-language conformance vectors and mutation tests are a blocker, not polish.

The third problem is product proof. A page showing `403`, `404`, and `200`, followed by a status assertion, is an approval test with extra provenance unless the extra machinery prevents a wrong decision that a naive diff would have encouraged. The reference case must demonstrate that shared state, instability, hidden projection, or shape-only reduction produces a concrete false question, and that Countershape refuses or corrects it. Otherwise the lifecycle is technically ornate but product-irrelevant.

Finally, the proposed build is larger than its evidence budget. Native Linux behavior cannot be receipted from this macOS machine merely through cross-compilation. Human comprehension, bias reduction, maintainability, novelty, and adoption cannot be graded by didrun. Generic report redaction cannot make arbitrary captured output safely shareable. Imported-repository setup cannot be made reproducible by recording argv. These must be cuts or explicit bets, not delayed claims.

## Prioritized blocker table

| Priority | Classification | Blocker | Concrete failure | Required disposition |
| --- | --- | --- | --- | --- |
| 1 | **BLOCKER** | World compatibility overclaims | Candidate branches on `cwd` hash or allocated port before projection replaces it; equal projected output does not prove equivalent execution conditions. | Rename it a declared comparison envelope; reject undeclared variance; make curated fixtures the only full claim. |
| 2 | **BLOCKER** | The flagship case is still a dressed fixture diff | A user could choose `404` from three printed lines; world identity, trials, reduction, and lineage do not change the answer. | Require the falsifiable stateful proof below or call the result an observation/approval prototype. |
| 3 | **BLOCKER** | Go discovery and Node residue can disagree | Go rejects duplicate JSON names while generated `harness.mjs` uses `JSON.parse` and accepts the last value. | One normative vector corpus must kill cross-language parser/runner/projection mutants before standalone is claimed. |
| 4 | **BLOCKER** | Accepted symlinks are unsafe under a lexical-only rule | `a -> b`, `b -> ../../victim` passes per-link lexical checks but resolves outside; cleanup can race a path changed by executed code. | Reject every symlink in this run. Reintroduce only with graph resolution and an explicit non-hostile cleanup boundary. |
| 5 | **BLOCKER** | Fresh confirmation can be counterfeited by metadata | A new nonce and rotated schedule can wrap reused captures or a cached process result. | Remove all execution-evidence reuse in v1; require new attempt artifacts and instrumented fixture execution counts. |
| 6 | **BLOCKER** | Report confidentiality is not establishable | A token appears in an allowed JSON value or prose not matched by redaction rules and enters a self-contained report. | Do not call reports safe or shareable; omit captured bodies by default, preview selected projection data, and label sensitivity unknown. |
| 7 | **BLOCKER** | Local-minimality can leak through summary paths | Core preserves the labeled map, but a cache/UI/final sweep compares only block membership or count. | A single map-digest type must cross persistence, cache, reducer, API, and UI; no ordinal cluster identity exists. |
| 8 | **REQUIRED CHANGE** | Readiness/setup success can masquerade as behavior | Readiness warms an auth cache; setup exits zero after failing to seed one tenant; the later `500` is classified as eligible behavior. | Say Countershape separates explicit control failures only. Use a fixture-owned readiness signal in the proof and trust declared setup semantics elsewhere. |
| 9 | **REQUIRED CHANGE** | The run claims too many environments | Node 24, Linux process groups, all visual states, and security behavior may not all execute here. | Claim only exact natively receipted runtime/OS/state combinations; everything else is `UNRECEIPTED`. |
| 10 | **ACCEPTED BET** | Blind-first may add ceremony without improving decisions | A user clicks every evidence tab, then still follows the likely branch after reveal. | Keep staged capture, but make comprehension/bias reduction a human-study bet, not a shipped capability. |

## Answers to the 18 controlling questions

### 1. Compatible worlds — **BLOCKER**

No general implementation can prove that two different real worlds omit no behavior-changing dimension. Suppose candidate A computes `if hash(cwd) % 2 == 0`, or the service uses the chosen port in a cache key. Countershape can normalize `/tmp/cs-123` to `{ROOT}` in the output; it cannot erase the earlier control-flow difference. The same applies to scheduler order, filesystem inode allocation, DNS, kernel state, and ambient files.

The implementation may establish only: "these instances satisfy comparison-envelope version E, which measured fields X, required equal fields Y, and explicitly tolerated fields Z." Ephemeral root and port differences are safe placeholders only for a typed projection of captured values, never proven semantically irrelevant. The proof fixtures may additionally attest through source-independent metamorphic checks that substituting roots/ports does not change the selected projection. Imported repositories receive no stronger claim. Replace `compatible worlds` with `instances admitted by a declared comparison envelope` everywhere.

### 2. Canonical authority bypass — **BLOCKER**

The highest-risk bypass is not JavaScript display code; it is the standalone Node harness. Go's `encoding/json` and Node's `JSON.parse` both destroy information before a strict identity layer can inspect it. Duplicate keys, `-0`, exponent spelling, unsafe integers, lone surrogates, invalid UTF-8 replacement, map order, locale sorting, and header joining all create divergence.

No unvalidated serializer may consume identity-bearing bytes. Go needs a strict token scanner before typed values. Node needs either a matching strict parser or a contract profile that does not parse those values. The shared vector corpus must run through both implementations and include byte-level expected results and expected rejections. TypeScript may display precomputed IDs only. Any successful contract test using a different parse/projection result than discovery is a blocker, even if both results look reasonable.

### 3. Materializer escape and hidden Git policy — **BLOCKER**

`ls-tree` plus `cat-file` avoids checkout filters, hooks, and archive attributes, but the invocation still needs replacement refs and partial-clone lazy fetch disabled. Otherwise a local replace ref can change object interpretation or a missing promisor object can trigger network access. Raw blob bytes must be rehashed against the named Git object format after streaming.

Lexically contained individual symlinks are insufficient because symlink chains can escape or cycle. Target-filesystem case and Unicode aliasing also vary. For this run, accept only `100644` and `100755`; reject all symlinks, gitlinks, LFS pointers, invalid paths, and collisions. Materialize files before execution and never traverse candidate paths during cleanup. Cleanup against code that deliberately renames/replaces the owned root is outside trusted-local guarantees; do not claim otherwise.

### 4. Infrastructure failures in outcome clusters — **REQUIRED CHANGE**

A centralized eligibility gate can prevent explicit `SETUP_ERROR`, `READINESS_ERROR`, `PROBE_TRANSPORT_ERROR`, `TIMEOUT`, `OUTPUT_LIMIT`, and `TEARDOWN_ERROR` from reaching projection. No adapter should construct an `OutcomeMap` directly. A mutation that maps any control state to a value must die.

But a setup command can exit zero while doing the wrong thing, and a readiness request can mutate application state while returning 200. A later application `500` is then an eligible observation because Countershape cannot infer the hidden cause. The brief must say it separates **detected control failures**, not all environment/setup failures. The HTTP proof should use a fixture-owned readiness signal or a demonstrably passive protocol, not a request whose non-mutating nature is merely asserted.

### 5. Same membership, different output — **BLOCKER**

The trap must be rejected in every representation, not only reducer code. Consider baseline `A=403, B=404, C=404` and neighbor `A=200, B=500, C=500`. A block-count cache, UI DTO using ordinal group IDs, or final sweep keyed by member sets will accept the wrong question.

Persist and pass one canonical `candidate_execution_key -> projection_fingerprint` map digest. Derived display groups are views and have no identity. Cache keys, reduction transcripts, confirmation, Choicepoint readiness, and UI summaries must all bind this digest. Tests must deserialize old artifacts and exercise the trap through the API and UI, not just a pure comparator.

### 6. False `ONE_MINIMAL_UNDER` — **BLOCKER**

Any unresolved direct neighbor, exhausted trial or wall budget, cancelled or partial sweep, stale plan, changed candidate eligibility, or reused execution evidence forbids the label. This must be a construction invariant: the `OneMinimal` type is creatable only from a complete final-sweep artifact whose enumerated valid neighbor set equals the set of fresh `CHANGES` results. A boolean `preserves=false` API is unacceptable because it collapses `CHANGES` and `UNRESOLVED`.

Tests must inject cancellation after the last apparent neighbor, timeout the smallest neighbor, and exhaust the budget exactly at sweep completion. Every case remains `BEST_KNOWN` unless the durable completion record exists.

### 7. Physical freshness — **BLOCKER**

A new random nonce is not evidence that a process executed. A cache can copy old captures into a new batch and assign new metadata. The narrow reference build should have no execution-evidence cache or reuse path at all. Each confirmation trial references a newly allocated root, a new attempt artifact created before spawn, a new process lifecycle receipt, and a fixture-observed invocation counter or nonce echo held outside the projected predicate. Confirmation also uses a different candidate rotation.

Pure canonical/projection caches remain allowed because they transform immutable bytes. Executed observations do not. Reuse can be redesigned later under a visibly weaker acceleration mode.

### 8. Field-selection soundness — **BLOCKER**

The separation formula is necessary and must be enforced over every eligible confirmed allowed and disallowed outcome. Field paths must come from a closed typed registry; no unchecked dotted path, implicit parent selection, or default field is legal. Missing and present-empty remain different. Allow-many is a canonical set of complete tuples, never independent value sets. Empty selection emits nothing.

The formula does not prove unseen future outputs are acceptable. A later output may match selected status `404` while changing an unselected leak-bearing body. That is intentional selected-field scope, not semantic equivalence. The UI and README must state this consequence directly. `CUSTOM_EXPECTATION` that collapses to an observed tuple requires an explicit action change or confirmation.

### 9. Noncompilable actions — **BLOCKER**

`REJECT_ALL`, `DEFER`, and `REFINE` must be impossible inputs to the emitter at the type/application-service boundary, not merely hidden buttons. The server repeats validation under CAS even if a stale or hostile client submits an old `ALLOW` payload. A stale Choicepoint, early preview, or replayed HTTP request cannot create files. Tests should call the emitter and API directly for every illegal state and assert an empty output directory.

### 10. Standalone Node residue — **BLOCKER**

The bundle must run with Countershape binaries, source, packages, services, and package registry unavailable. For HTTP, "network unavailable" must mean no external network; local loopback is required and must be stated. A clean invocation must distinguish an eligible selected-tuple mismatch from startup, readiness, timeout, output, projection, and teardown errors in TAP diagnostics and machine codes.

Physical absence is only half the gate. The vendored harness must pass the same vectors as Go for the exact contract profile, and selected-field mutations must die. Claim only the exact Node major and native OS actually run. Byte-identical emission does not establish semantic parity or maintainability.

### 11. Blind payload leakage — **REQUIRED CHANGE**

The number of distinct observed outcomes is necessarily visible; what must be absent is candidate identity, producer/model provenance, candidate count per outcome, and total support/majority cues. The synthesis's phrase "outcome cardinality" is ambiguous and should become "per-outcome candidate support cardinality." Opaque aliases must be Choicepoint-scoped so they cannot correlate behavior across studies. Outcome order, card dimensions, typography, accessible names, DOM attributes, analytics labels, and hidden text must be independent of support count and candidate order.

Behavior may reveal authorship, so this remains identity-hidden presentation, not anonymity or debiasing.

### 12. Required evidence viewing — **TEST-ONLY RISK**

The UI can require that original/derivation, projection operations, nonasserted fields, and provenance reveal were rendered and visited before finalize. It cannot prove the user read or understood them. Keyboard, mobile, and screen-reader flows must exercise the gate; stale tabs must fail CAS. The DecisionRecord should record visit/reveal events as interaction facts, not evidence of comprehension.

Whether the protocol improves decisions remains an accepted human-study bet. Do not replace that study with click telemetry or call the flow bias-resistant.

### 13. Localhost forgery and disclosure — **BLOCKER**

Reads and writes require the bearer; writes additionally require exact Origin, JSON content type, CSRF, and current digest CAS. Host is literal `127.0.0.1:port`, CORS is absent, tokens never enter query strings or logs, and the initial fragment is removed after transfer to session storage. Candidate content is never used in URLs. A malicious page's form POST, fetch preflight, DNS-rebinding Host, `Origin: null`, stale tab, and guessed study ID must all fail.

This does not defend against browser extensions or another same-user process that steals the token. The threat model must preserve that exclusion.

### 14. Injection and report confidentiality — **BLOCKER**

Candidate bytes can contain ANSI/OSC control sequences, bidi controls, HTML, `</script>`, forged headings, path separators, and strings that become accessible names. All views need inert text rendering and visible encodings for controls. Static reports need structural escaping and CSP without embedding candidate values in script contexts.

Generic redaction cannot prove confidentiality. Secrets can appear under unknown keys, be encoded, split, hashed, or be the very field the user selected. Therefore the product may produce a **default-minimized export**, not a safely shareable report. It omits Captured bodies, includes only previewed projected fields, records redaction operations, and displays `CONFIDENTIALITY NOT ESTABLISHED`. `--include-raw` is an explicit dangerous export. Security tests prove known fixtures are omitted and payloads inert; they do not prove absence of secrets.

### 15. Two-domain kernel — **REQUIRED CHANGE**

Only identity, eligibility, stable-batch classification, exact map comparison, tri-valued reduction orchestration, lineage, and ruling validation may be generic. CLI keeps exit-versus-signal, stdout/stderr, argv/env/fixtures, and its own reducers. HTTP keeps transport-versus-response, ordered query/header multimaps, readiness, body rules, and its own reducers. No generic JSON observation DTO should become internal truth.

The acceptance test is architectural as well as behavioral: adding HTTP must not add HTTP branches to compare/reduce/choice packages, and CLI types must not be coerced into HTTP-shaped fields. If that fails, this run narrows to the domain that has the coherent model.

### 16. Decisive case versus dressed diff — **BLOCKER**

The current `403/404/200` story does not yet prove the system. A shell script can print those three outputs, and a human can write `assert status == 404`. The decisive case must show a naive one-shot/shared-world or shape-only reducer producing the wrong decision surface, then show Countershape refusing or correcting it.

Use the full falsifiable proof below. If the build cannot demonstrate that fresh worlds, repeated trials, visible projection, and labeled-map reduction each alter eligibility or the question, the honest output is a differential observation plus approval-test generator—not the full Choicepoint lifecycle claim.

### 17. Wake and didrun boundaries — **REQUIRED CHANGE**

Candidate refs and opaque producer provenance may mention Wake. No Wake events, epochs, effects, gates, sessions, process recovery, fleet scheduling, replay, fork, or timeline become Countershape state. The candidate runner owns a short-lived program world only; it does not own coding agents. Reduction transcripts describe stimuli, not execution-history replay.

didrun references are opaque verbatim records bound to the commit/tree and command receipt they name. Countershape never parses a grade into `CONFORMS`, aggregates it into a stronger label, or assigns it to another capability. A weird or unknown grade string must round-trip unchanged. Missing evidence is `UNRECEIPTED`, not inferred from a green UI.

### 18. Receipt completeness — **BLOCKER**

Every claimed native OS, Node major, visual state, security behavior, timed metric, and deterministic build must have its own load-bearing command through didrun and exact tree freshness. On this host, Darwin can be claimed if run; Linux remains `UNRECEIPTED` unless a real Linux environment runs the same suite. Cross-build is compilation evidence only. Node 24 is unclaimed unless Node 24 actually runs. Screenshot presence is not visual correctness; manual and different-model findings remain separately labeled observations.

didrun cannot receipt novelty, user comprehension, bias reduction, maintainability, security review completeness, or adoption. Those remain bets even if every command gets an exact grade.

## Scope cuts necessary for this run

The synthesis's U0-U9 architecture is a long-term reference roadmap, not an honest single-run completion promise. Preserve the heavy semantic spine while cutting claims and dangerous breadth:

1. **Darwin-only runtime claim.** Cross-compile Linux if useful, but label it unreceipted runtime support.
2. **Curated repositories only for the full proof.** Do not claim arbitrary 5k-100k LOC imports. A separate hostile materializer suite may establish supported blob handling.
3. **Regular and executable blobs only.** Reject all symlinks, submodules, LFS, non-UTF-8 paths, and target-filesystem collisions.
4. **No setup command and no package installation in proof studies.** Reference fixtures use Node core only. Imported setup remains a future trusted adapter surface.
5. **No execution-evidence reuse, prepared roots, snapshots, or reset hooks.** Fresh execution is simpler and falsifiable.
6. **One invocation and one request only.** No arbitrary workflows, database cloning, cookies, redirects, compression, TLS, external host, or browser replay.
7. **Narrow portable field registry.** Support only fields exercised and cross-language receipted by the two studies. Defer arbitrary nested JSON/decimal semantics if the strict dual implementation is not complete.
8. **One actually executed Node major.** Other supported targets are `UNRECEIPTED` until their standalone matrix runs.
9. **Report is a local evidence export, not safely shareable.** Default-minimized and previewed; confidentiality unknown.
10. **Studio completion is conditional.** If authenticated writes, blind payload tests, full evidence parity, keyboard/mobile safety, or visual loops fail, ship the CLI surface and seed-state renderer without a studio-complete claim.

These cuts still leave a substantial system: strict identity/canonical values, Git object materialization, fresh process worlds, typed CLI and HTTP observations, repeated classification, exact N-way maps, tri-valued reduction, immutable Choicepoints, selected-field compilation, standalone execution, CLI, decision bench, and evidence export.

## Smallest falsifiable full-system proof

The minimum proof that deserves **proceed** uses four HTTP candidates in a dependency-free committed fixture and must defeat a naive comparator:

1. Candidate A returns stable `403`; B stable `404`; C stable metadata-bearing `200`; D alternates `404` and `200` across fresh invocations.
2. Every response also contains a volatile request ID and scratch-root path. Captured evidence preserves them; the declared projection visibly removes only those fields. A mutation that hides status or metadata fails.
3. A deliberately naive shared-root run causes contamination that makes B and C appear equal. Fresh roots eliminate the contamination. The report shows the naive result only as a negative fixture, never product evidence.
4. Repetition classifies D as `UNSTABLE` and excludes it; it never becomes an outcome card.
5. A reducer proposal deleting tenant seed data preserves membership shape but changes labeled outputs from `403|404|200` to `200|500|500`; Countershape rejects it. A later typed reduction preserves the exact map and earns either honest `BEST_KNOWN` or a fully swept `ONE_MINIMAL_UNDER`.
6. A physically new confirmation batch, verified by external invocation counters and new attempt artifacts, reproduces the labeled map under a rotated schedule.
7. Blind-first ruling allows `404`, selects status only, displays metadata body as nonasserted, then reveals candidate provenance. A weak field selection that collapses allowed and rejected outcomes returns `AMBIGUOUS_SCOPE`.
8. The emitted Node bundle runs with Countershape removed. B conforms; A and C contradict; D either contradicts/conforms per a fresh eligible run or becomes an ineligible/unstable diagnostic according to the contract profile—never historical truth rewritten.
9. A custom `401` expectation makes every eligible stable candidate contradict. `REJECT_ALL` emits no code.

The independent CLI precedence study then proves the second typed adapter with a smaller bar: private HOME prevents ambient config contamination; argv/env/config precedence remains visible; selected stdout field and exit semantics compile; unselected diagnostics do not fail. If either domain requires a separate compare/reduce/choice kernel, narrow to one.

## Exact brief edits beyond the synthesis

In addition to all 24 synthesis edits, the living brief must make these further changes:

1. Replace every implication that world compatibility is discovered or proven with **declared comparison envelope** language. List tolerated dimensions, measured dimensions, uncontrolled dimensions, and the fact that placeholder projection cannot prove control-flow irrelevance.
2. Remove symlink support from this run. The supported materialization subset is regular/executable blobs only; all links and expanded Git forms are typed refusals.
3. Require Git invocations to disable replacement-object interpretation and lazy promisor fetch, and require raw object rehash verification. No materialization step may trigger network.
4. State that only detected control failures are separated from behavior. Setup exit success and readiness noninterference remain trusted assumptions; the proof fixtures avoid both ambiguities.
5. Delete execution-evidence reuse from v1. Fresh confirmation is enforced by new attempt creation and observed invocation, not a nonce alone.
6. Name the Go/Node semantic duplication as a first-class risk. The standalone claim requires one normative cross-language vector corpus for parser, field selection, HTTP/CLI capture, projection, and error taxonomy.
7. Clarify blind leakage language: distinct outcome count is visible; candidate identities and per-outcome support counts are absent. No claim of anonymity or debiasing.
8. Replace `shareable report` with `default-minimized local evidence export`; add `CONFIDENTIALITY NOT ESTABLISHED` and a preview gate. Redaction tests are fixture coverage, not a secrecy guarantee.
9. Replace the generic first aha with the stateful falsification proof above. The brief must name which naive result is wrong and which Countershape invariant prevents it.
10. Split completion claims by environment. Darwin, Linux, Node major, browser viewport/state, and security control each require an exact receipt; no umbrella cross-platform grade.
11. State that visited evidence views prove presentation only, never comprehension. Review compression and bias effects remain human acceptance criteria.
12. Add a run-level stop condition: if the build reaches only curated outputs without the contamination, instability, visible projection, shape-trap, fresh-confirmation, separation, and absent-runtime proofs, rename the milestone **observation/approval prototype** rather than full reference system.

## Final verdict and confidence ledger

**Stop/go: conditional go for a narrowed Darwin reference instrument; no-go for the synthesis's broad imported-repository, cross-platform, safely shareable, review-compressing system claim.**

The truth kernel deserves implementation because its refusal semantics are unusually crisp and falsifiable. The complete product deserves the name Countershape only if the stateful proof demonstrates that the machinery prevents a false question, not merely records a verbose route to an obvious answer. If that proof fails, retain the useful kernel and standalone emitter but stop calling it a repository-scale operationalization.

| Claim | Confidence | Ruling |
| --- | ---: | --- |
| Strict canonical identity can be implemented in Go | 9/10 | proceed after mutation/vector gate |
| Cross-language standalone semantics can match | 6/10 | blocker until vectors and absence run |
| Fresh local Darwin worlds classify curated fixtures honestly | 8/10 | proceed with trusted-code boundary |
| General world compatibility for imported repositories | 2/10 | reject claim |
| Labeled-map tri-valued reduction is truthful | 9/10 | proceed; usefulness only 4/10 |
| Materializer for regular/executable blobs is safe enough | 8/10 | proceed; reject links/expanded forms |
| Two typed adapters share one kernel | 7/10 | test; cut second domain if coercion appears |
| Blind-first studio captures a scoped ruling faithfully | 7/10 | proceed conditionally |
| Blind-first studio improves judgment | 3/10 | human-study bet |
| Default report is confidential/shareable | 2/10 | reject claim |
| Linux/Node 24 portability without native receipts | 1/10 | `UNRECEIPTED` |
| Countershape is a novel algorithm or semantic primitive | 1/10 | reject |
| Countershape is a differentiated reference composition if the decisive proof passes | 6/10 | accepted bet |
| Adoption and maintainership | 3/10 | human/market tail |

This red team does not validate demand, production safety, hostile-code isolation, secret completeness, legal clearance, cross-platform behavior, or maintainership. It approves only a narrower experiment whose strongest result is an auditable refusal to turn unstable, contaminated, weakly scoped, or semantically changed evidence into a human contract.
