# Countershape deep-dive synthesis: the controlling build contract

- **Inputs:** six specialist reports in `research/deep-dive/01-06` and the Phase 1 `docs/CONCEPT_BRIEF.md`
- **Date:** 2026-07-14
- **Decision:** **MODIFY, THEN PROCEED WITH THE BOUNDED REFERENCE SYSTEM**
- **Overall confidence:** 7/10 that the locked system is buildable and technically honest; 4/10 that it compresses real review work; 3/10 that it earns adoption
- **Authority:** this document resolves conflicts among the six lanes. The living brief must be revised to match it before prompt-pack or implementation.

## Executive ruling

Countershape survives, but its original novelty story and several broad product implications do not.

**Go:** build a repository-scale operationalization of interactive program disambiguation: exact Git object selections enter; trusted local CLI or HTTP candidates run in fresh, measured worlds; exact typed projections form an observed-stable candidate-to-outcome map; typed reducers locally reduce that same labeled map; a human rules on one exact witnessed stimulus without candidate or majority cues; and a deterministic standalone Node contract preserves only the fields the human explicitly selected.

**Modify:** call a Choicepoint Countershape's immutable, provenance-bound decision record, not a new semantic primitive. Split Git selection, source materialization, deterministic world configuration, measured world execution, captured evidence, lossy projection, human intent, emitted source, and current contract execution into different objects and truth jurisdictions. Replace `STABLE` with `OBSERVED_STABLE(k/k)`. Preserve the complete candidate-to-canonical-outcome map during reduction, not merely partition membership. Treat reduction as three-valued. Require a distinct fresh confirmation batch. Make exact selected-field predicates the only compilable v1 scope. Separate `REJECT_ALL` from `CUSTOM_EXPECTATION`. Add the field-separation obligation before compilation. Make blind-first adjudication and local-studio write security load-bearing.

**Kill:** kill the claim that Countershape invented candidate-derived intent questions, behavior clustering, human choice among implementation behaviors, feedback-to-test compilation, differential testing, shrinking, or approval contracts. Kill the phrase “finds the smallest behavior.” Kill an arbitrary-repository latency promise, any sandbox or network-denial implication, a universal adapter abstraction, and any event-log/replay/runtime expansion. If the build becomes a fixture diff plus an approval snapshot, uses checkout/archive as exact source materialization, reuses mutable worlds for evidentiary trials, hides normalization, compiles a ruling that cannot separate allowed from rejected observations, or needs Countershape to run the exported test, stop rather than relabel it complete.

The controlling product thesis is therefore:

> **Countershape compares exact repository candidates in declared fresh worlds, locally reduces an observed-stable behavioral split under named rules, lets a human state an exact witnessed expectation without branch or majority cues, and emits a standalone selected-field regression contract.**

This is an ambitious systems composition, not a new disambiguation algorithm. The closest semantic collision is Socrates; TiCoder, FlashProg, ARHF, SpecFix, xTestCluster, differential testing, delta debugging, property-based shrinking, approval testing, and Pact-style contracts establish the component lineage. The plausible differentiation is the complete operational lifecycle across real Git trees, measured worlds, instability, visible projections, strict reduction provenance, plural rulings, and standalone residue. That differentiation has only 6/10 confidence because a bounded search cannot prove absence.

## Hard scope lock

### The reference system includes

1. One **trusted local** Git repository and two to four refs resolved once to immutable commit and tree object IDs. Ref names are display provenance, never execution identity.
2. A filter-free Git-plumbing materializer for a declared subset: regular blobs, executable blobs, and lexically contained relative symlinks. It rejects gitlinks, LFS pointers, invalid or unrepresentable paths, symlink escapes, and filesystem collisions as typed `UNCOMPARABLE` conditions.
3. A Go core with strict canonical values, content-addressed immutable artifacts, atomic compare-and-swap head pointers, typed state machines, Git/process ownership, exact comparison, bounded reduction, deterministic emission, loopback API, and report export.
4. Two typed adapters over one truth kernel:
   - CLI: one ordered argv invocation with optional stdin, sparse declared environment, and bounded fixture files.
   - HTTP: one declarative fixture seed, one fresh local HTTP/1.1 service, one side-effect-free readiness check, and one method/path/ordered-query/header/body request.
5. Fresh materialization and explicit setup for every trial that contributes to discovery, shrink evaluation, final sweep, confirmation, or conformance. Execution is sequential by default with a recorded rotated candidate schedule.
6. A deterministic `WorldPlan` and measured `WorldInstance`, including explicit budgets, `HOST_ALLOWED` networking, sparse environment, private `HOME`/`TMPDIR`/state roots, toolchain facts, process-group lifecycle, and uncontrolled dimensions.
7. Immutable post-capture-policy `CapturedObservation` artifacts, versioned pure projections, strict typed canonical values, and byte equality only. Exact bytes may be described as raw only when the capture policy actually retained them unchanged.
8. Repeated classification into `OBSERVED_STABLE(k/k)`, `UNSTABLE`, `UNCOMPARABLE`, or `INCOMPLETE`. Infrastructure states remain reasons, never behavior clusters.
9. Deterministic candidate-to-outcome maps and typed, terminating reducers with `PRESERVES`, `CHANGES`, and `UNRESOLVED`; grades are `UNCHANGED`, `BEST_KNOWN`, or `ONE_MINIMAL_UNDER(reducer_set_digest)`.
10. Immutable Choicepoint lineages and the five human actions: allow one observed outcome, allow several observed outcomes, custom expectation, reject all observed outcomes, or defer. Refine derives a successor study; it is not an in-place ruling.
11. Exact-witness `one-of-exact/v1` predicates over explicitly selected portable fields. No assertion is preselected. Allow-many is a set of complete tuples, never a field cross-product.
12. A deterministic six-file Node 24 `node:test` bundle, using Node core plus repository code, that runs with Countershape absent. Node 22 is supported only if actually receipted. Darwin and Linux are claimed separately and only after native runtime receipts.
13. A monochrome-capable CLI, authenticated loopback studio, and escaped/redacted self-contained report. The React/Vite studio is embedded in the binary at runtime.
14. Two dependency-free three-candidate reference studies: HTTP cross-tenant authorization and CLI configuration precedence. At least one earns `ONE_MINIMAL_UNDER`; an unresolved case yields `BEST_KNOWN`; a deliberate flake becomes `UNSTABLE`; allow-many is exercised; `CUSTOM_EXPECTATION` produces none conforms.

### The reference system excludes

No agent launching, provider routing, worktree/task lifecycle, candidate ranking, merging, source synthesis, Wake event ingestion, run recovery, process resume, causal event viewer, or effect ledger. No didrun-like command recorder, claim evaluator, freshness grader, or evidence authority. Candidate provenance may mention Wake or any producer as inert metadata only.

No hostile-code containment, untrusted pull-request execution, filesystem sandbox, network denial, container promise, multi-user service, cloud runner, accounts, collaboration, billing, or production traffic. No Windows claim. No Docker golden path.

No arbitrary action-sequence DSL, browser adapter, GUI search, database cloning, external API, package installation in proof fixtures, automatic setup inference, persistent execution-evidence cache, prepared worlds, reset-hook correctness shortcut, general plugin protocol, in-process configuration code, or custom executable normalizer.

No fuzzy equality, tolerances, regex predicates, wildcards, similarity clustering, inferred invariants, semantic equivalence, universal quantification, equivalence-class scope, root-cause claim, global minimum, safety certification, or “best branch.” No model-generated probes, labels, explanations, rulings, or tests in the golden path. A real external-model critic may evaluate the visual build as a human-gated build tool; it is not product runtime.

The fifteen-minute target applies only to the two checked-in dependency-free studies on a named reference machine with fresh worlds and disclosed budgets. Imported repositories have explicit trial, wall, byte, and setup budgets and no latency or success guarantee. The 5k-100k LOC range remains a user hypothesis, not an execution guarantee.

## Integrated truth jurisdictions

| Question | Sole authority | What it does not establish |
| --- | --- | --- |
| Which source objects were selected? | `TreeIdentity`: repository fingerprint, object format, immutable commit/tree OIDs | bytes materialized, runtime dependencies, or behavior |
| Which supported source entries were written? | `MaterializationManifest` and portable tree digest | a complete Git checkout, submodule/LFS contents, or sandboxing |
| What execution was intended? | deterministic `WorldPlan` | actual ports, tools found, platform, timing, or cleanup |
| What execution occurred? | measured `WorldInstance` plus trial-control evidence | repeatability outside its declared compatibility boundary |
| What program data was durably captured? | `CapturedObservation` after its named capture/redaction policy | bytes discarded before persistence or semantic meaning |
| What values were compared? | `ProjectionDefinition` and `ProjectionResult` | correctness, safety, or equivalence beyond exact projected bytes |
| What disagreement was observed? | confirmed candidate-to-projection-fingerprint `OutcomeMap` | majority truth, determinism, or complete specification |
| What was reduced and with what grade? | original/minimized stimuli, reducer set, transcript, budgets, final sweep | global minimum, root cause, or preserved human comprehension |
| What behavior did the human intend here? | immutable `DecisionRecord` against one Choicepoint digest | broader intent, security approval, or implementation endorsement |
| What source was emitted? | byte digest of deterministic `ContractBundle` | current conformance, authenticity, or long-term maintainability |
| Does the current checkout satisfy the selected predicate now? | fresh `ContractExecution` | recreation of the historical WorldInstance or overall correctness |
| Did an external verification command run on a Git tree? | didrun's **verbatim** receipt grade, when present | product behavior beyond that grade; Countershape never upgrades it |

Git remains content authority. Countershape is authority only for the canonical bytes and transitions it creates. The human is authority only for the scope they explicitly rule on. didrun remains an external receipt authority: referenced grades are stored verbatim with their commit/tree association, or the capability is labeled `UNRECEIPTED`. A didrun receipt is never converted into `CONFORMS`, and a Countershape observation is never converted into a didrun grade.

## Exact object model

Every persisted object has `schema_version`, `kind`, canonical bytes, and a domain-separated SHA-256 digest over `"countershape/v1/" + kind + NUL + bytes`. The strict value layer rejects duplicate keys, unsafe internal integers, negative zero, non-finite numbers, invalid UTF-8, and lone surrogates; it does not Unicode-normalize. Durations are integer milliseconds and external exact decimals are tagged canonical strings.

### Source and world

`TreeIdentity` contains `repository_fingerprint`, `object_format`, `commit_oid`, `tree_oid`, `ref_spelling_display_only`, and `git_version`. Two candidate slots with the same executable tree are rejected or explicitly coalesced.

`MaterializationManifest` contains `tree_identity_digest`, `materializer_version`, `policy_digest`, sorted entry records `(path_bytes_as_accepted_utf8, git_mode, blob_oid, portable_blob_digest, materialized_mode, symlink_target)`, explicit rejections, `portable_tree_digest`, and measured target-filesystem facts. Absolute root, current time, and random nonce are evidence in the WorldInstance, not stable manifest identity. It proves only that the supported entries were written and verified under that policy.

`WorldPlan` is deterministic and contains the candidate selection, materialization policy, adapter/runner digest, setup/start argv arrays, cwd policy, sparse environment names and nonsecret values, secret-slot presence policy, fixture/reset recipe, readiness recipe, network mode, capture policy, projection/comparator, repeat schedule, tool requirements, concurrency mode, and every time/trial/entry/byte/output budget with defaults materialized. It contains no shell strings, ambient interpolation, absolute output paths, allocated port, temp root, PID, or current time.

`WorldInstance` contains `world_plan_digest`, `instance_nonce`, actual owned roots and permissions, allocated port/resources, measured platform/kernel/architecture/filesystem, resolved tool paths/versions/digests, setup receipt, post-setup manifest, candidate order schedule, uncontrolled dimensions, start/finish facts, process-group lifecycle, and teardown result. A versioned compatibility predicate decides whether different instances may contribute to one observation claim. Digest equality is neither necessary nor sufficient for compatibility.

### Capture, projection, and outcome map

`CapturedObservation` contains `world_instance_digest`, `stimulus_digest`, `trial_index`, `trial_control`, `capture_policy_digest`, independently tagged channels (`Present`, `Absent(reason)`, or `Truncated(digest,count)`), HTTP/process metadata, private artifact digests, and teardown eligibility. Only `FINALIZED` execution with complete required capture and successful teardown may be projected. HTTP 500, CLI exit 2, and complete empty stdout can be ordinary behavior. Setup failure, readiness failure, transport failure, timeout, cancellation, output limit, projection rejection, or orphan risk cannot.

`ProjectionDefinition` contains the adapter domain, implementation and configuration digests, accepted channel contract, explicit operations, selected fields, header/query multimap rules, strict JSON rules, placeholder rules, and exact comparator version. `ProjectionResult` is `Ok(CanonicalProjection, fingerprint, source_links)` or `Rejected(reason)`. V1 equality is fingerprint equality over canonical projection bytes. Pairwise tolerances and order-dependent clustering do not exist.

A `StableBatch` binds one candidate execution key, stimulus, repeat policy, schedule, compatible WorldInstances, every capture/projection, and classification. It is `OBSERVED_STABLE(k/k,h)` only when all required fresh eligible trials yield `h`; multiple eligible fingerprints are `UNSTABLE`; ineligible world/control/projection states are `UNCOMPARABLE`; budget exhaustion before `k` is `INCOMPLETE`.

`OutcomeMap` is the canonical sorted map `candidate_execution_key -> projection_fingerprint` for the exact eligible candidate set, with excluded candidates and reasons listed separately. Its block IDs are fingerprints, not ordinals. A divergence requires at least two eligible candidates and at least two fingerprints. Candidate order, label, branch, producer, and group size do not alter its digest.

### Reduction

`ReductionRun` binds baseline `OutcomeMap`, original stimulus, reducer-set digest, well-founded measure, budgets, every proposed neighbor, fresh evaluation batches, tri-valued decisions, accepted path, and final direct-neighbor sweep. A proposed neighbor must be valid and strictly smaller. Evaluation is `PRESERVES` only when a fresh eligible batch yields the exact same candidate-to-fingerprint map; `CHANGES` only for another stable eligible map; everything else is `UNRESOLVED`.

The result grade is exactly one of:

- `UNCHANGED`: no accepted smaller stimulus;
- `BEST_KNOWN`: a smaller preserving stimulus exists, but the budget ended, the sweep was incomplete, or any direct neighbor was unresolved;
- `ONE_MINIMAL_UNDER(reducer_set_digest)`: every valid direct neighbor was freshly evaluated and classified `CHANGES`.

The minimized witness receives a distinct fresh confirmation batch with a new nonce and rotated schedule before it can become decision-ready. Discovery or shrink evidence reuse never satisfies confirmation. The original, minimized stimulus, transcript, budgets, unresolved count, and grade remain permanently linked.

### Choicepoint, decision, contract, and current execution

`Choicepoint` is an immutable ready revision containing candidate/tree identities, WorldPlan and compatibility policy, original/minimized stimuli, capture and projection identities, the baseline and fresh-confirmed OutcomeMaps, `OBSERVED_STABLE(k/k)` evidence, excluded candidates, reduction grade/transcript, and evidence digests. Changing any semantic ancestor creates a new lineage and marks descendants stale; old bytes are never edited into freshness.

`DecisionRecord` contains exact `choicepoint_digest`, actor attribution, blind-view state, early-reveal event if any, provisional ruling, revealed provenance, any post-reveal change plus rationale, action, selected fields, explicit nonasserted fields, exact-witness scope, human annotation, and optional didrun references verbatim. Actions are `ALLOW_OBSERVED`, `CUSTOM_EXPECTATION`, `REJECT_ALL`, or `DEFER`; `REFINE` is a successor request. Only the first two are compilable.

The **field-separation obligation is mandatory**. For selected fields `F`, allowed outcomes `A`, and disallowed confirmed outcomes `D`:

```text
for every a in A and d in D: select_F(a) != select_F(d)
```

Missing remains tagged missing. If an allowed and disallowed outcome collide, compilation returns `AMBIGUOUS_SCOPE` and no files; it may suggest candidate-neutral differing fields but never auto-select them. Allow-many compiles a canonical set of selected-field tuples, not the cross-product of individual field values. An empty field set is forbidden. `REJECT_ALL` records dissatisfaction but has no positive oracle and emits no test. Only a separately reviewed typed `CUSTOM_EXPECTATION` can cause all candidates to contradict.

`ContractBundle` is a sorted artifact set containing `decision.json`, `fixture.json`, `contract.test.mjs`, `harness.mjs`, `README.md`, and `manifest.json`. It binds emitter/target versions and the `one-of-exact/v1` predicate AST. Its bytes contain no current time, random ID, absolute path, candidate name, branch name, secret, formatter drift, or Countershape import. It uses Node core only, performs its own narrow runner/projection semantics, and passes a clean-checkout test with Countershape absent from PATH, dependencies, filesystem, and network.

`ContractExecution` is new append-only evidence against the current checkout. It records current Git identity, bundle digest, runtime/world facts, captured/projection result, and one of `ELIGIBLE_OBSERVATION -> CONFORMS | CONTRADICTS` or `INELIGIBLE_EXECUTION -> typed reason`. It never rewrites or freshens the Choicepoint. Bundle SHA-256 proves byte integrity only, not authorship.

## State and refusal model

The artifact lineage is:

```text
SOURCE_SPEC -> COMPILED_PLAN -> MATERIALIZED_CANDIDATE_SET
-> BASELINE_OBSERVATION -> DIVERGENCE -> REDUCTION_RESULT
-> FRESH_CONFIRMATION -> CHOICEPOINT_READY -> RULING
-> CONTRACT_BUNDLE -> CONFORMANCE_REPLAY
```

Each arrow emits a new immutable object. `CANCELLED` and `PARTIAL` retain completed evidence but do not advance. Semantic changes make descendants `STALE`; explicit supersession creates `INVALIDATED(reason,replacement)`. A ruling before fresh confirmation, a contract from reject/defer/refine, a resumed reduction under a changed plan, or an added candidate under an old map is an illegal transition.

Study UI states are `EMPTY`, `PREPARING`, `ACTIVE`, `PARTIAL`, `ERROR`, and `COMPLETED`. Candidate eligibility is `OBSERVED_STABLE(k/k)`, `UNSTABLE`, `UNCOMPARABLE`, or `INCOMPLETE`, with typed control reasons beneath it. Choicepoints are `DISCOVERED`, `DECISION_READY`, `RESOLVED`, `DEFERRED`, `STALE`, or `INVALIDATED`. `COMPLETED` means declared budgets ended and all items have an honest disposition; it never means verified software.

Refusal is a feature. Unsupported Git forms, changed refs after resolution, incompatible worlds, inherited-secret requirements, unstable candidates, failed teardown, incomplete confirmation, unresolved direct neighbors, stale rulings, ambiguous selected fields, unsupported contract targets, and unreceipted platforms prevent the stronger claim rather than being coerced into a behavior or pass.

## Trusted-local threat boundary

V1 runs explicitly trusted repository and setup code with the operator's full user permissions and host network access. A disposable root, sparse environment, private directories, time/output caps, and POSIX process-group teardown reduce accidental contamination; they are not filesystem, network, CPU/memory, or hostile-code containment. A descendant that deliberately creates a new session can escape. Dependencies and plugins are executable code outside the promise.

Before first execution, CLI and studio must state: **“Countershape will run trusted repository code with your user permissions and host network access. The temporary directory is for repeatability, not security isolation.”** Network mode is `HOST_ALLOWED`, never “best-effort blocked.” Setup is explicit argv, loud, and absent in the proof fixtures.

The process owner uses no shell, bounded byte streams, a new POSIX process group, TERM/grace/KILL, waits for the direct child, probes the group, and returns `ORPHAN_RISK` if cleanup is uncertain. Cross-platform support requires native Darwin and Linux receipts; cross-compilation is not runtime validation.

The mutable studio binds literal `127.0.0.1`, opens with a high-entropy fragment token moved to session storage, requires bearer authorization on reads and writes, exact Host and Origin, no permissive CORS, CSRF on mutations, and compare-and-swap against the current Choicepoint digest. Bundled assets use restrictive CSP and inert rendering. Private captured artifacts default to mode 0600; shareable reports omit captured bodies by default, escape all candidate content, apply versioned export redaction, and require explicit `--include-raw`. Redacted bytes are never called raw.

## Blind-first decision protocol

The human surface is a five-stage protocol, not a branch picker:

1. Show human-authored scenario context, exact witness scope, experiment jurisdiction, observed/confirmation counts, and security boundary.
2. Show minimized and original stimuli together with derivation, reducer names, grade, budgets, and unresolved-neighbor count.
3. Show deterministic, equally weighted outcome facts while candidate identity, producer/model provenance, and group cardinality are absent from the payload, not CSS-hidden. Ordering derives from a persisted Choicepoint-scoped hash.
4. Let the user allow one, allow several, author a custom expectation, reject all, or defer. Nothing is pre-asserted. Every differing projected field is marked `Assert` or `Context only`; preview shows the exact tuple set and `Not asserted` fields.
5. Reveal candidate identities and implications before finalization. Early reveal is recorded. A changed post-reveal ruling preserves both choices and requires a rationale.

The UI says “candidate identity hidden in this view,” never anonymous, unbiased, recommended, winner, or most candidates. Deterministic labels contain typed facts such as `HTTP 404 · JSON object (2 fields)`, never invented semantic adjectives. Human-authored context is visibly separate. The Captured -> Projection -> Operations boundary is one action away, and a projection change creates a new lineage.

Desktop at 1440px uses a study rail, decision canvas, evidence inspector, and nonoverlapping action bar. Mobile at 375px shows one complete outcome card at a time with labeled switching and full access to original, derivation, projection operations, nonasserted fields, and identity reveal. If evidence parity fails, mobile is triage/defer-only. WCAG 2.2 AA, keyboard completion, screen-reader semantics, 200% zoom, forced colors, reduced motion, inert candidate output, and no horizontal page overflow are gates, not polish.

## Architecture and U0-U9 sequence

The core is Go; the studio is React/Vite and embedded. Persistence is a content-addressed artifact graph plus atomic head pointers, not a mutable event database. Disposable SSE progress is not evidence. Pure packages own canon/domain/spec/compare/reduce/choice/emit; Git, world/process, typed CLI/HTTP adapters, store, server, and report remain at the edges. JavaScript displays precomputed truth; it does not recompute digests or state classifications.

Units are ordered and gated as follows. Each is claimable only after its load-bearing commands run through didrun, claims are declared, the commit is sealed, and `NO_COLOR=1 didrun verify --strict` exits zero.

1. **U0 — controlling contracts and threat model:** revise the living brief; lock schemas, state machines, budgets, refusal semantics, fixture specs, package directions, claim language, and machine-readable examples.
2. **U1 — truth kernel:** strict canonical parser/encoder, domain digests, plan compilation, exact comparison, outcome maps, ruling validation, vectors, properties, model-state tests, and required mutants. No subprocess exists.
3. **U2 — source/process substrate:** immutable ref resolution, filter-free materializer, private fresh worlds, sparse environment, byte caps, process groups, teardown, and hostile fixtures. Claim only natively tested operating systems.
4. **U3 — CLI observation spine:** typed one-shot CLI stimulus/capture/projection, repeat scheduler, rotated order, control taxonomy, observed-stable and flake fixtures, and baseline comparison.
5. **U4 — HTTP observation spine:** typed local HTTP start/readiness/request/capture/projection and declarative seed; prove the same truth algebra without domain coercion.
6. **U5 — bounded reducer:** typed neighbors, well-founded measures, tri-valued evaluations, full labeled-map preservation, budgets/transcripts, final sweep, same-shape/different-outcome trap, `BEST_KNOWN`, and `ONE_MINIMAL_UNDER`.
7. **U6 — Choicepoint and residue:** artifact graph/head CAS, immutable lineage, fresh confirmation, blind ruling data contract, separation obligation, deterministic six-file Node bundle, and fresh conformance. Prove Countershape absence.
8. **U7 — complete CLI/reference studies:** safe next-action CLI, both three-candidate studies, allow-many, custom none-conforms, unstable and uncomparable fixtures, three clean reproductions, and named-machine timing.
9. **U8 — secure studio/decision bench:** authenticated loopback API, all states, desktop/mobile flows, a minimum of three build-see-exercise-critique-rebuild passes, Playwright/axe/manual accessibility, and a real different-model visual critic.
10. **U9 — report, packaging, and hardening:** escaped/redacted report, single-binary packaging, docs/license/contributor path, performance and limitation ledgers, final adversarial suite, final strict verify, and didrun HTML evidence.

The allowed fallback products are explicit: stop at truth-kernel prototype if canonical/state mutations survive; observation-only if fresh worlds cannot protect behavior classification; one adapter if the second needs a second truth model; decision JSON only if standalone emission fails; CLI-only if studio security/accessibility fails. A cut is not promoted to the full claim.

## Exact acceptance matrix

| Capability | Required evidence and pass bar | Failure disposition |
| --- | --- | --- |
| Canonical identity | all malformed/golden vectors and properties pass; required canon/order/domain mutants die | stop before subprocess code |
| Source identity/materialization | blob/mode round-trip; moving ref remains pinned; filters/hooks/archive attributes never run; unsupported forms reject | remove imported-tree claim |
| World/process lifecycle | every evidentiary run has fresh roots/env/ports/setup; ordinary descendants die; byte caps hold; control failures remain typed | unsupported OS or observation-only cut |
| Stability | constant, alternating, timeout, projection failure, and incomplete fixtures classify exactly with counts | no divergence readiness |
| Exact N-way compare | candidate permutation invariance and same-shape/different-output trap pass | kill Choicepoint claim |
| Reduction | smaller witness in both domains; full labeled map preserved; unresolved yields `BEST_KNOWN`; one complete sweep earns local grade | observation-only product |
| Rulings/field separation | legal state transitions exact; no defaults; one/many tuple semantics; weak fields return `AMBIGUOUS_SCOPE`; reject/defer emit none | no executable residue |
| Standalone bundle | byte-identical across clean roots; Countershape physically absent; Node core only; selected fields only; custom none-conforms works | decision record only |
| Two-domain kernel | both studies share identity/observe/compare/reduce/choice services while preserving typed adapters | narrow to one domain |
| Secure studio | missing/wrong auth, Host/Origin, CSRF, stale CAS fail; blind payload omits identity/cardinality; candidate content remains inert | CLI-only |
| Human surface | all required states at 1440x900 and 375x812; three visual passes; keyboard/axe/zoom/forced-colors/reduced-motion; different-model critic findings addressed or logged | no studio completion claim |
| Report | default self-contained export is escaped/redacted, excludes private captured bytes, and labels every receipt verbatim or `UNRECEIPTED` | no shareable-report claim |
| Reproduction/performance | three clean study runs keep Plan/Choicepoint/contract bytes stable while new Instance/evidence stays linked; both proof studies meet 15 minutes on named machine | revise metric, never hide reuse |
| Cross-platform | complete native process/standalone matrix on each claimed OS/Node version | unrun platform is `UNRECEIPTED` |

Automated UI gates do not substitute for visual inspection. Unit tests do not substitute for the multi-pass build loop. A different-model opinion does not substitute for deterministic acceptance. didrun receipts do not substitute for the underlying test, and an unreceipted manual observation remains labeled as such.

## Rejected claims and required public language

Allowed language: “repository-scale operationalization,” “observed stable k/k,” “no observed difference under named finite probes,” “exact projection equality,” “best known under named reducers and budgets,” “1-minimal under the recorded reducer set,” “human-scoped exact-witness ruling,” and “standalone selected-field test.”

Prohibited language: invented interactive disambiguation; new Choicepoint primitive; first implementation-derived intent question; first behavior clustering or feedback-to-test loop; oracle-free verification; deterministic world; raw after redaction; smallest/root cause/global minimum; semantic equivalence; correctness/safety/security approval; winner/best branch; complete intent; universal portable contract; sandboxed; network blocked; reproducible without named boundaries; or didrun evidence upgraded beyond its verbatim grade.

Wake's boundary is exact: Wake may produce immutable candidate refs and inert provenance. Countershape does not launch agents, consume Wake event history, resume/replay/fork runs, manage effects, recover processes, or become a friendlier Wake UI. Its content-addressed evidence graph stores semantic artifacts, not a command/event runtime.

## Evidence tally and confidence ledger

This synthesis deliberately deduplicates overlapping lane findings into **23 confirmed constraints, 18 strong engineering inferences, and 12 unverified bets**.

- **Confirmed constraints (23):** Socrates collides with the semantic novelty claim; feedback-to-test and differential/reduction substrates pre-exist; Wake owns agent-runtime/replay territory; didrun owns receipt grading; Git checkout/worktree/archive can apply policy; submodules/LFS exceed a parent tree; temporary roots are not sandboxes; POSIX groups are cleanup, not containment; loopback writes need auth/Origin/CSRF; Git identity differs from materialized bytes and world; Plan differs from Instance; captured may be post-redaction; exact projection equality is the safe v1 rule; finite repeats do not prove determinism; infrastructure states differ from behavior; reduction must preserve outcome labels; reduction is tri-valued; unresolved blocks local minimality; fresh confirmation differs from discovery; reject-all has no positive oracle; field selection must separate allowed/disallowed outcomes; exported source differs from current execution; candidate identity/cardinality are decision cues.
- **Strong engineering inferences (18):** Go core plus embedded React; strict project-owned canonicalization; filter-free plumbing materializer; fresh world per evidentiary trial; sequential rotated schedule; content-addressed artifacts instead of an event DB; typed CLI/HTTP adapters sharing only algebra; one invocation/request as the proof shape; exact selected-field AST; six-file Node-core bundle; immutable lineage and CAS; three-level UI state vocabulary; blind-first staged reveal; deterministic factual labels; evidence-parity-or-triage mobile; authenticated localhost app; dependency-free proof studies; CLI milestone before studio.
- **Unverified bets (12):** repeated materialization is fast enough; three-to-five repeats are useful; strict labeled-map reduction still shrinks real cases; typed projections are understandable without model prose; users tolerate explicit field selection; both domains fit one kernel; Node bundles remain maintainable; process cleanup behaves equivalently on supported hosts; default redaction is useful; both demos meet fifteen minutes; the bench beats diff-first review; builders retain enough simultaneous candidates for adoption.

| Area | Confidence | Reason |
| --- | ---: | --- |
| Novelty boundary | 9/10 negative, 6/10 differentiated composition | direct PLDI collision; bounded non-discovery cannot prove uniqueness |
| Canonical truth/observation algebra | 9/10 | pure, narrow, property- and mutation-testable |
| Filter-free source materialization | 8/10 | Git plumbing is strong; edge paths/performance need implementation |
| Fresh-world process correctness on Darwin | 7/10 | coherent POSIX contract; no containment and daemon escape remain |
| Linux runtime parity | 5/10 before native receipts | design inference only until run |
| Exact labeled reduction | 8/10 truthfulness, 5/10 usefulness | semantics are crisp; compression is unproven |
| Contract fidelity | 9/10 after separation obligation | pure exact predicate; imported-repo integration remains harder |
| Blind-first DX/accessibility | 7/10 feasibility, 3/10 review compression | interaction is testable; comparative users are absent |
| Fifteen-minute proof fixtures | 6/10 | plausible only under the locked dependency-free profile |
| Arbitrary repository usability | 3/10 | setup, state, and cost are uncontrolled |
| Security containment | 1/10 | intentionally not provided |
| Adoption/maintainership | 3/10 | no market or maintainer evidence |

## Questions the red team must answer exactly

1. Can two different real worlds be called compatible without omitting a dimension that changes behavior, and can the implementation prove why ephemeral port/root differences are safe placeholders?
2. Does any Go or JavaScript serializer, map iteration, float conversion, duplicate-key behavior, negative zero, locale, or Unicode normalization bypass the canonical authority?
3. Can the materializer be tricked into filters, hooks, archive attributes, path aliasing, symlink-parent traversal, submodule/LFS hydration, or cleanup outside its nonce-owned root?
4. Can setup/readiness/transport/timeout/output/teardown failure enter an outcome cluster through any adapter path?
5. Does the same-membership/different-output trap get rejected everywhere, including caches, final sweep, UI summaries, and persisted lineages?
6. Can any `UNRESOLVED` neighbor, exhausted budget, reused batch, or cancelled sweep still produce `ONE_MINIMAL_UNDER`?
7. Is fresh confirmation physically new execution, with different instance nonce and schedule, or only a renamed cache hit?
8. Can field selection allow a disallowed confirmed outcome, create a cross-product for allow-many, assert an unselected field, or compile an empty predicate?
9. Can `REJECT_ALL`, `DEFER`, or `REFINE` produce executable code through any API or stale UI path?
10. Does the emitted bundle run after Countershape is removed from PATH, source, dependencies, services, and network, and does it preserve harness failure versus contradiction?
11. Does the blind API/accessibility tree leak branch/model identity or group cardinality, and does ordering or visual weight encode majority?
12. Can a user finalize without viewing original/derivation, projection operations, every nonasserted field, and the provenance reveal?
13. Can a hostile page or stale tab read evidence or forge a ruling through Host, Origin, CORS, bearer, CSRF, or missing CAS checks?
14. Can candidate bytes create terminal control, HTML/script, accessible-name, report, or path injection, or leak captured secrets into the default report?
15. Does either adapter force domain facts into a generic JSON model, or require bespoke truth/reduction/choice services?
16. Does the decisive stateful case prove that world identity, repeated trials, visible capture/projection, and shrink transcript change the human decision, rather than decorating a fixture diff?
17. Has any runtime/event/replay/agent concern crossed the Wake boundary, or any external didrun grade been interpreted, summarized upward, or assigned to an unreceipted capability?
18. Are every claimed OS, Node version, visual state, timed metric, and security behavior backed by its own load-bearing didrun receipt, with everything else explicitly `UNRECEIPTED`?

## Exact edits required to the living brief

Before prompt-pack, `docs/CONCEPT_BRIEF.md` must be materially rewritten, not appended with caveats:

1. Change state to “deep dive complete; bounded reference spine approved subject to U0 gates.” Replace the one-line promise with the controlling thesis from this synthesis.
2. Credit Socrates explicitly and redefine Choicepoint as Countershape's provenance-bound record format, not a new human operation or semantic primitive.
3. Change the user/repository language so 5k-100k LOC is an audience hypothesis; scope the fifteen-minute metric to the two dependency-free named studies and named machine.
4. Replace `committed tree contents only` with `TreeIdentity` plus supported filter-free `MaterializationManifest`; list default rejection of gitlinks, LFS, unsafe symlinks/paths, and collisions.
5. Replace singular world/manifest language with deterministic `WorldPlan`, measured `WorldInstance`, and a versioned compatibility result.
6. Replace `raw observations` with `CapturedObservation after declared capture policy`; reserve raw for unchanged persisted bytes and separate report redaction.
7. Define exact canonical projection byte equality as the only v1 comparator; explicitly cut fuzzy/tolerant/similarity/custom-code normalizers.
8. Replace all `STABLE` claims with `OBSERVED_STABLE(k/k)` and require a separate fresh confirmation batch before decision readiness.
9. Replace partition preservation with exact candidate-to-projection-fingerprint map preservation; add `PRESERVES`, `CHANGES`, `UNRESOLVED` and all three reduction grades.
10. Add the exact artifact, execution-attempt, study, candidate-eligibility, Choicepoint, and reduction state machines and their illegal transitions.
11. Replace the candidate object list with the exact objects in this synthesis, including `DecisionRecord`, `ContractBundle`, and `ContractExecution` as separate authorities.
12. Add immutable content-addressed persistence and atomic head CAS; explicitly prohibit an event-log/replay/resume runtime and preserve the Wake boundary.
13. Lock Go core plus embedded React/Vite and typed CLI/HTTP adapters sharing only truth orchestration. Limit shapes to one CLI invocation and one seeded HTTP request.
14. Add default/hard execution budgets and pre-run trial multiplication disclosure. Fresh worlds and setup remain the reference path; no hidden cache reuse.
15. Replace human actions with the five-stage blind-first protocol; remove identity/cardinality from blind payload; audit early reveal and post-reveal changes.
16. Add deterministic factual labels, original/reduced/derivation parity, permanent Captured->Projection operations, no-default field selection, explicit nonasserted fields, and tuple-set allow-many.
17. Fix v1 scope to the exact witnessed stimulus. Separate custom expectation, reject all, defer, and refine; only allow/custom compile.
18. Add the field-separation formula and `AMBIGUOUS_SCOPE`; prohibit compilation when selected fields cannot distinguish allowed from disallowed outcomes.
19. Define the six-file Node 24 bundle and standalone absence test; add eligible `CONFORMS|CONTRADICTS` versus typed ineligible execution; keep current execution separate from historical evidence.
20. Replace the security paragraph with the exact trusted-local/full-user-permissions/`HOST_ALLOWED` warning, process-group limitation, native-OS receipt boundary, studio controls, and private-versus-shareable evidence policy.
21. Replace the roadmap with U0-U9 exactly, including truth-kernel mutation gates before subprocesses, CLI semantic milestone before studio, three visual passes, real different-model critic, and final didrun HTML.
22. Replace the acceptance section with the matrix in this synthesis and name honest fallback products. Add the composition kill gate and comparative five-person comprehension/bias test as unvalidated human acceptance.
23. Add prohibited claim language and the prior-art lineage; require every release claim to be classified as observed capability, product boundary, or novelty assertion with a cited closest work.
24. Preserve didrun grades verbatim in every DecisionRecord/report receipt reference and label anything not executed through didrun `UNRECEIPTED`.

## Final decision

**Proceed, after the brief changes.** The system is worth building because its refusal-oriented integration is technically substantial and unusual enough to make an eyebrow-raising reference artifact. But the build earns that description only if provenance, instability, projection visibility, strict reduction, selected-field compilation, standalone absence, security boundaries, and blind human adjudication remain load-bearing.

What is validated is the conceptual coherence of a narrow architecture and the fact that its abstract ingredients have deep prior art. What remains a bet is almost everything users would care about after the demo: whether real repositories can run fresh worlds cheaply, whether strict reduction finds comprehensible cases, whether the decision bench beats diff review, whether contracts survive refactors, whether nonexperts interpret the evidence correctly, and whether anyone maintains adapters or adopts the workflow. The build can produce evidence for the technical spine. It cannot manufacture market validation, hostile-code security, production operations, cross-platform receipts it did not run, legal clearance, or maintainership.
