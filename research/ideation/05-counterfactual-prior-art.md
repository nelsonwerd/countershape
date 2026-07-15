# Counterfactual ambiguity lab: prior art, novelty boundary, and build gate

- **Research date:** 2026-07-14
- **Concept under review:** import 2-4 real Git refs or patches from any producer; run the same declared CLI, HTTP, or browser stimuli against every candidate; normalize observations; search for and shrink behavior that separates candidates; ask a human which outcome expresses intent; persist the answer as an executable contract; rerun and rank candidates by conformance.
- **Decision:** **PROCEED TO DEEP DIVE, 7/10 confidence.** This is a defensible product/system experiment, but not a new testing algorithm. Its only credible novelty is the human-facing loop from *candidate disagreement* to *minimal intent decision* to *durable executable contract* on real repository branches.
- **Automatic park condition:** if the implementation's value is mostly “run the tests on several worktrees,” “compare screenshots,” or “let an LLM choose a patch,” park it. Those categories are already occupied.

## Executive judgment

The idea survives the Wake collision and is more defensible than the behavior-to-code tracing option currently called Threadline. It has a different source of truth, user job, and aha moment from Wake:

- **Source of truth:** versioned, human-approved **Intent Decisions** and their executable witnesses—not an agent event log, trace history, or winning branch.
- **Primary user job:** discover tacit product decisions hidden inside several plausible implementations before choosing or merging one.
- **Aha:** four green-looking branches receive the same input, split into three observable behaviors, and the machine shrinks the split to one product question the user can answer in seconds. That answer immediately becomes a regression contract and regrades all four candidates.
- **Non-claim:** candidate agreement is not correctness; majority behavior is not intent; a human choice is not a proof of safety; observed equivalence is not semantic equivalence.

Threadline's “trace the product, not the agent” inversion is appealing, but its concrete surface is now visibly crowded. [Repro](https://repro.dev/) records clicks, DOM changes, console errors, and network requests and exposes the recording to coding agents through MCP. [Traceway](https://tracewayapp.com/product/traces) connects browser interactions and session replay to backend traces, exceptions, source maps, and database calls. [Highlight](https://highlight.io/) and [OpenReplay](https://docs.openreplay.com/) already join session replay with errors, logs, and backend context. Threadline could still add a code-slice compiler, but its flagship causal join increasingly looks like a product extension of an established observability stack.

The ambiguity lab instead compares **alternative futures under controlled stimuli**. Repro/APM products explain one observed execution. This system asks where several plausible implementations disagree, then turns the smallest disagreement into a durable decision. That is a cleaner open product job—even though almost every algorithmic ingredient has strong prior art.

## Why this is timely in July 2026

Parallel candidate production is becoming ordinary. [OpenAI's Codex app](https://openai.com/index/introducing-the-codex-app/) runs agents concurrently in isolated worktrees and lets users review each diff. [Worktrunk](https://github.com/max-sixty/worktrunk) makes worktree creation, switching, hooks, previews, merging, and cleanup convenient for 5-10 parallel agent sessions. [Hive](https://github.com/morapelker/hive) puts Claude, Codex, and OpenCode worktrees, sessions, diffs, and Git operations in one visual application.

Those products reduce the cost of obtaining branches. They do not reduce several branches to *behavioral decision points*. Their review object remains a thread, worktree, or diff.

Patch selection is also an active frontier rather than a hypothetical future problem:

- [Agentless](https://github.com/OpenAutoCoder/Agentless) samples multiple patches, filters them with regression and generated reproduction tests, then reranks survivors. Its selection pipeline demonstrates that “generate many, submit one” is already standard in software-agent research.
- The July 2026 [PatchFusion paper](https://arxiv.org/abs/2607.01597) names the pass@k-to-pass@1 gap directly and deterministically fuses repeated edit atoms across candidate pools. Its benchmark result is evidence that candidate diversity contains useful signal. It does **not** establish that consensus edits express a product owner's intent.
- The Codex/worktree trend plus PatchFusion's post-generation framing creates the opening: once branches are cheap, the scarce resource is a good human decision about their meaning.

This is adoption pull for candidate comparison, not purchase-intent validation.

## The closest prior art—and exactly what it invalidates

### 1. SpecFix is the sharpest conceptual collision

[SpecFix](https://github.com/msv-lab/SpecFix), the artifact for the ASE 2025 paper [“Automated Repair of Ambiguous Problem Descriptions for LLM-Based Code Generation”](https://arxiv.org/abs/2505.07270), already:

1. samples many programs from one natural-language requirement;
2. clusters candidates by behavior under generated inputs;
3. interprets behavioral dispersion as specification ambiguity; and
4. refines the requirement to reduce that uncertainty.

The paper's key abstraction is the distribution of programs induced by a description. That is uncomfortably close to “use alternative implementations to reveal underspecified intent.” Any claim that the lab invents ambiguity discovery through candidate disagreement would be false.

The remaining boundary is product and systems scope, not algorithmic priority. SpecFix operates on code-generation benchmark problems, requires supported chat-completion APIs, generates candidate programs itself, and automatically repairs natural-language descriptions. The proposed lab would instead:

- import already-existing, repository-level Git branches from any human or tool;
- treat the candidate producer as irrelevant;
- execute full applications through CLI, HTTP, and bounded browser drivers;
- preserve raw observations and explicit normalizer provenance;
- ask a human to decide among concrete outcomes, including `either`, `neither`, and `needs another distinction`;
- compile the decision into a local executable contract; and
- rerun the entire candidate set against the accumulated contract corpus.

That is meaningfully different in use, but it should cite SpecFix prominently and avoid claiming an original ambiguity-detection method.

### 2. xTestCluster already clusters patches using difference-exposing tests

[xTestCluster](https://arxiv.org/abs/2207.11082) analyzes patches from one or more automated repair tools, generates tests with EvoSuite and Randoop, clusters patches by which generated tests they fail, and hands reviewers the inputs that expose behavioral differences. Across 902 plausible patches from 21 Java repair tools, its authors reported a median 50% reduction in patches needing review.

This invalidates two easy novelty claims:

- clustering syntactically different patches by observed behavior is not new; and
- giving a reviewer the input that separates patch behavior is not new.

The lab must earn its existence after clustering: a multimodal repository harness, structure-aware shrinking, a legible intent-decision interaction, contract compilation, and longitudinal conformance. If it merely displays one patch per behavior cluster, it is a general-purpose xTestCluster application.

### 3. Difference-exposing-test search is a mature research problem

[Mokav](https://www.sciencedirect.com/science/article/pii/S0164121225002407) takes two programs plus one example input and iteratively searches for a difference-exposing test using execution feedback. It reported finding such tests for 81.7% of 1,535 curated Python program pairs, versus lower baselines. The approach is LLM-driven and function/program-pair scoped, but the central operation—search for an input where implementations differ—is exactly prior art.

The lab should therefore implement a deterministic, typed generator/shrinker core first. Optional model-assisted proposal can be added only behind a user-supplied real credential and cannot be necessary for the golden path.

### 4. Differential service testing already sends identical HTTP traffic to variants

[Twitter's Diffy](https://blog.x.com/engineering/en_us/a/2015/diffy-testing-services-without-writing-tests) multicasts the same request to a candidate and two known-good service instances, compares responses, and learns likely nondeterministic noise from disagreement between the two known-good copies. It already covers same-request execution, response normalization, aggregation, and a browser triage surface.

The ambiguity lab is not “Diffy for agents.” Its distinction must be:

- no trusted baseline is assumed;
- 2-4 peer candidates are partitioned by behavior;
- the system actively searches and shrinks separating stimuli rather than only replaying supplied traffic;
- the output is a human intent question, not a regression alert; and
- the accepted result becomes an executable contract that can prefer one behavior, allow several, or reject all candidates.

Diffy's twin-baseline noise technique is also a correctness lesson: one run per candidate is insufficient. The lab needs repeated same-tree executions to estimate flakiness before attributing a difference to code.

### 5. Shrinking is established infrastructure, including oracle-deficient shrinking

Delta debugging has minimized failure-inducing inputs and interaction sequences for decades. The original [ddmin work](https://www.st.cs.uni-saarland.de/papers/tse2002/) explicitly reduced a 95-action browser failure to three actions and 896 lines of HTML to one line. [libFuzzer](https://llvm.org/docs/LibFuzzer.html) minimizes crashing inputs and reduces corpus entries while preserving coverage features. [fast-check](https://fast-check.dev/docs/advanced/model-based-testing/) generates and efficiently shrinks command sequences, preserves replay seeds/paths, and reports minimal counterexamples. [Schemathesis](https://schemathesis.readthedocs.io/en/stable/explanations/data-generation/) generates schema-derived HTTP requests, exercises stateful API sequences, and automatically shrinks failures.

Even the claimed oracle-free angle is current prior art. The July 2026 [DDMT paper](https://arxiv.org/abs/2607.00929) combines delta debugging with metamorphic relations so reduction can preserve a property without a traditional correctness oracle.

The lab can contribute a typed cross-candidate predicate—“the normalized partition of candidate observations remains the same”—and reducers for product inputs, but must call its output **locally minimal under declared reducers**, never globally minimal or semantically complete.

### 6. Visual review already turns human approval into baselines

[Playwright](https://playwright.dev/docs/test-snapshots) compares screenshots and arbitrary snapshot values, with explicit warnings that rendering varies by OS, browser, settings, and hardware. [Chromatic](https://www.chromatic.com/docs/branching-and-baselines/) compares branch UI snapshots, routes changes to humans, and persists accepted branch-specific baselines. [Lost Pixel](https://github.com/lost-pixel/lost-pixel) provides an open-source visual regression engine; its managed surface adds screenshot approval/rejection.

[ApprovalTests](https://approvaltests.com/) has long captured complex output, asked a human to approve it, and stored the approved artifact as a future regression oracle. The basic arc “show difference -> human approves -> save expected result” is therefore not new.

The lab's browser surface must go beyond baseline screenshot approval:

- compare multiple peer futures rather than head versus one approved past;
- search over fixtures, viewports, and action sequences;
- normalize DOM/accessibility/network observations separately from pixels;
- shrink the *stimulus*, not only crop or highlight the output difference; and
- express the decision as an outcome predicate with explicit tolerated variation, not an opaque blessed PNG.

The current July 2026 [PlayCoder repository](https://github.com/Tencent/PlayCoder) is another warning. It evaluates generated GUI patches through real interactions and screenshots because compile/unit-test success misses silent behavioral failures. Behavioral GUI execution itself is not a wedge.

### 7. Specification and contract tooling occupies both sides of the loop

[GitHub Spec Kit](https://github.com/github/spec-kit) makes specs, plans, tasks, checklists, and cross-artifact analysis first-class across more than 30 coding agents. [VibeContract](https://arxiv.org/abs/2603.15691) proposes decomposing high-level intent into task contracts linked to code, testing, runtime verification, and debugging. [Pact](https://docs.pact.io/implementation_guides/pact_specification) persists and replays consumer-provider interaction contracts.

The lab cannot win on “turn intent into executable specs.” Its distinct direction is **discover intent from empirical disagreement among plausible implementations**. Traditional spec tools move intent toward code; this loop lets code variants expose where intent was missing, then sends the resulting decision back into the executable specification.

## Novelty ledger

### Established composition

These pieces are prior art and should be described as implementation choices:

- Git refs/worktrees as isolated candidate content;
- parallel agent/worktree dashboards;
- executing the same input against several implementations;
- dynamic behavior clustering and test-equivalence classes;
- difference-exposing-test generation;
- property-based, fuzz, metamorphic, and delta-debug input search;
- HTTP shadow traffic and response normalization;
- screenshot/DOM snapshot comparison and approval baselines;
- generated tests, BDD, consumer contracts, and spec-driven workflows;
- test-based candidate filtering, majority voting, LLM judges, or consensus patch fusion.

### Genuinely differentiated system proposition

The defensible new primitive is a **Counterfactual Intent Decision**:

```text
candidate Git trees
  + shared controlled world
  + one minimized, replayable stimulus
  + raw and normalized N-way outcomes
  + explicit human choice (one / many / none / refine)
  + scope and provenance
  = durable executable decision contract
```

The candidates are used as an **oracle-free question generator**, not as voters. Their disagreement reveals a boundary the original request failed to specify. The human decides the product meaning. The system makes that choice durable and immediately measures every candidate against it.

This is novel as a coherent local tool for real branches and a non-expert-facing workflow. It is probably not publishable as a new testing algorithm without further research. Honest positioning: **a counterfactual product-decision laboratory assembled from differential testing, shrinking, approval testing, and executable contracts**.

### Why it is more defensible than Threadline

Threadline's remaining novelty is a precise behavior-to-source context cutter; most of its capture and causal-join stack is already visible in Repro, Traceway, Highlight, and OpenReplay. The ambiguity lab has formidable academic neighbors, but no inspected product turns arbitrary Git candidate branches into minimized human product decisions and accumulates those decisions as a branch-neutral executable contract corpus.

Confidence in that negative claim is **6/10**, not certainty. A deep dive should search patents, research prototypes, and internal developer tools specifically for “N-version branch comparison + counterexample generation + human intent capture.”

## Strongest truthful demo

Use a real local SaaS fixture and four prebuilt Git branches for one underspecified request:

> “Let support agents view customer invoices, but never let them change payment methods.”

Every branch builds and passes the existing suite. The lab imports all four refs and runs a declared OpenAPI/JSON-state search in isolated worlds. It finds several differences, then shrinks them to three short decisions:

1. **Same-tenant support read:** one branch returns a fully masked card, one returns the last four digits, and one denies access.
2. **Cross-tenant invoice id:** branches split between `403`, `404`, and a `200` metadata leak.
3. **Mutation attempt:** one branch returns `403` without effects; another returns `200` but silently no-ops.

The visual bench shows one stimulus on the left and outcome clusters in synchronized columns—not source diffs. The user chooses:

- same-tenant support may see last four digits;
- cross-tenant reads must return `404` to conceal resource existence; and
- mutation attempts must return `403` and produce no data delta.

Each choice becomes an executable contract with the minimized fixture, request sequence, status/body/data predicates, allowed nondeterminism, candidate commit ids, environment fingerprint, and raw observations. Rerun produces:

```text
candidate-a  2/3 decisions satisfied
candidate-b  3/3 decisions satisfied
candidate-c  0/3 decisions satisfied · security-relevant contradiction
candidate-d  1/3 decisions satisfied
```

The result does **not** say candidate B is correct or safe. It says B conforms to all three decisions and the existing declared gates. A secondary browser scenario can show the same machinery at 375px, but HTTP should carry the proof because its state and observations are easier to isolate and shrink honestly.

## Heavy-but-coherent v1

### Scope in

1. **Candidate import:** 2-4 refs from one local Git repository; committed trees only for receipted comparison. Patches may be materialized into temporary commits with provenance.
2. **Controlled worlds:** one disposable sandbox/worktree per candidate, unique ports, explicit environment allowlist, timeout/CPU/memory/process limits, network denied by default, and a declarative reset recipe between cases.
3. **Drivers:**
   - one-shot CLI processes with argv/stdin/fixture files;
   - HTTP services with readiness probes and JSON request/state generators;
   - Playwright scenario replay for declared browser action sequences.
4. **Search:** seed corpus plus typed generators. HTTP JSON Schema/OpenAPI and a small built-in primitive schema are enough for v1. No claim of useful blind fuzzing without an input model.
5. **Observations:** raw exit/stdout/stderr/file delta; HTTP status/headers/JSON/body/data delta; browser DOM/accessibility tree/screenshot/console/network. Every lossy transform is separately labeled.
6. **Comparison:** canonical observation fingerprints, N-way equivalence partitions, repeated-run flake classification, and `UNALIGNED` / `UNRESOLVED` states.
7. **Shrinking:** structure-aware JSON/string/sequence reducers that preserve the same candidate partition; local-minimality receipt with attempted reductions and replay seed.
8. **Intent Decisions:** `prefer outcome`, `allow outcomes`, `reject all`, `defer`, and `add distinction`; no forced winner.
9. **Contract compiler:** deterministic JSON/YAML artifact plus native replay command. A contract pins the stimulus and predicates; it never embeds a model judgment.
10. **Conformance frontier:** hard contract/test results, measurements, and unknowns in separate columns. No universal quality score and no majority-as-truth.
11. **Human surfaces:** a monochrome CLI and responsive comparison bench with raw/normalized toggles, first-divergence focus, shrink history, decision capture, and exact next action.
12. **Portable report:** self-contained HTML plus machine-readable contract corpus.

### Named deferrals

- launching or supervising coding agents;
- importing Wake internals or recreating its event/runtime machinery;
- hosted execution, distributed runners, accounts, RBAC, billing, or collaboration;
- production traffic shadowing;
- automatic database cloning/reset for arbitrary stacks;
- semantic equivalence proofs;
- automatic patch merging, splicing, or synthesis;
- model-generated intent, tests, explanations, or patch judgments in the golden path;
- unconstrained browser exploration or general GUI action generation;
- performance benchmarking beyond explicit envelopes;
- Windows isolation and a production-grade hostile-code sandbox;
- more than four simultaneous candidates.

Browser **search** should be deferred even if browser **replay** ships. Stateful UI sequence generation and shrink correctness are enough of a research project by themselves. The first reference-quality spine should be Git + sandbox + HTTP/CLI search + decision contracts; Playwright proves multimodal extensibility through explicit scenarios.

## Reference architecture

```text
Git refs / patch files
        |
        v
candidate materializer -----> immutable candidate manifest
        |
        v
isolated world factory ------> environment + reset fingerprint
        |
        v
typed stimulus engine -------> seed / generator / replay path
        |
        +----> candidate A driver ----> raw observation A
        +----> candidate B driver ----> raw observation B
        +----> candidate C driver ----> raw observation C
        +----> candidate D driver ----> raw observation D
                                      |
                                      v
versioned normalizers -------> N-way observation partition
                                      |
                         search + structure-aware shrink
                                      |
                                      v
                            minimal divergence case
                                      |
                                      v
human decision UI -----------> Intent Decision contract
                                      |
                                      v
                             replay + conformance frontier
```

The durable objects should be small and explicit:

- `Candidate`: Git tree id, materialization recipe, build provenance.
- `World`: image/toolchain, env allowlist, reset recipe, resource policy.
- `Stimulus`: typed value or action sequence, seed, validity predicate.
- `Observation`: raw bytes/artifacts plus typed projections.
- `Normalizer`: versioned pure transform, redaction/ignore/tolerance declarations.
- `Partition`: candidate groups under one observation equivalence relation.
- `Divergence`: stimulus, partition, stability, shrink proof, unknowns.
- `IntentDecision`: accepted/allowed/rejected outcomes, rationale, scope, author/time.
- `Contract`: replayable fixture plus predicates derived from the decision.
- `ConformanceResult`: exact contract execution against an exact candidate tree.

## Correctness and security traps

### 1. “Same input” is meaningless without the same starting world

Candidate behavior depends on database rows, filesystem state, process order, locale, timezone, dependency lockfiles, clocks, randomness, and downstream services. Each trial needs a reset protocol and environment fingerprint. The UI must distinguish `CODE_DIVERGENCE` from `WORLD_MISMATCH`; it cannot normalize the latter away.

### 2. Imported branches are untrusted code

Build steps can run postinstall hooks, steal host credentials, read SSH agents, fork-bomb, persist daemons, or exfiltrate source. Worktrees are content isolation, not security isolation. The golden path needs a disposable OS/container boundary, no host secrets, read-only source input, a writable scratch volume, no network by default, resource caps, process-group cleanup, and loud platform caveats. A path-traversing patch or symlink must never escape the candidate root.

### 3. Normalization can erase the exact thing the user needed to decide

Timestamp/UUID/header/order removal is convenient and dangerous. Preserve raw observations permanently; make every ignore/tolerance rule visible and versioned; show a raw-versus-normalized toggle; and rerun identical code to measure natural variance. “Normalized equal” means equal under that named projection, nothing more.

### 4. Flakiness creates fake ambiguity

One candidate may alternate outcomes because of races. Each discovered divergence needs replay quorum and a stability label. If the same candidate produces multiple partitions, surface `FLAKY` instead of attributing the split to different code. Timing-sensitive cases need explicit schedules or remain unresolved.

### 5. Stateful shrinking can change the cause

Deleting an earlier browser/API action may alter authentication, IDs, cache state, or transaction boundaries while coincidentally preserving the final output split. Reducers need validity checks, dependency repair, reset between attempts, and causal fingerprints where possible. Report 1-minimality under the configured reducer—not “the root cause.”

### 6. Alignment is a semantic problem

Candidate A may emit an extra event, redirect, or request. Zip-by-index comparison will pair unrelated observations. Align by declared checkpoints and semantic keys; when alignment fails, produce `UNALIGNED` rather than a fabricated diff.

### 7. Majority is not an oracle

Three agents can copy the same misconception. Candidate count must never create confidence. Rank only on explicit contracts, independently meaningful tests, and measurements. Show ties and “none conforms.” PatchFusion-style consensus may be useful secondary evidence, never product intent.

### 8. A single chosen example does not justify a broad rule

The human can decide that one concrete response is desirable without intending it for every input in the generator domain. Contract capture must ask scope: exact witness, equivalence class, or user-edited predicate. Generalization is a separate explicit act with fresh counterexamples.

### 9. Side effects make parallel comparison unsafe

HTTP `POST`, browser flows, emails, queues, and payment SDKs can mutate shared systems. Every candidate needs isolated data and ports; external network calls are denied unless a user explicitly supplies a safe local substitute. Never fake a real API and label it integrated.

### 10. Observation storage can capture secrets and personal data

Headers, bodies, DOM, screenshots, logs, and database deltas are high-risk. Redaction must occur before persistence, raw-secret capture should be opt-in, reports need sharing warnings, and the local store needs retention/deletion controls. A “raw” toggle cannot mean “secret-bearing by default.”

### 11. Contract compilation can silently strengthen the human's decision

If the user chooses “return 404,” the generated assertion must not also freeze headers, body wording, latency, or database internals unless selected. Show the exact predicate before saving. The contract compiler should be deterministic and snapshot-tested independently.

### 12. Ranking can hide incomparable trade-offs

Accessibility, correctness contracts, performance, dependency churn, and maintainability are not safely reducible to one number. Use a Pareto frontier and explicit hard gates. Subjective design criticism is a separate human/model note and is never promoted to a deterministic receipt.

## Smallest falsifiable proof

Proceed from deep dive into a full build only if a vertical prototype can satisfy all of these without a model credential:

1. Import at least three committed candidate branches for a substantive local application.
2. Materialize and reset their worlds reproducibly without shared mutable state.
3. Discover a behavior split not already named by the existing tests.
4. Shrink it to a materially smaller, stable HTTP or CLI witness in under 15 minutes on the reference machine.
5. Present the alternatives so a mid-level vibe coder can make the product decision without reading the source diff or raw logs.
6. Compile the choice into a human-readable executable contract.
7. Rerun that contract against every candidate and reproduce the same conformance result from a clean checkout.
8. Preserve raw observations, normalizer versions, candidate tree ids, environment fingerprints, and unknowns.
9. Produce `none conforms` on a seeded case where all candidates are wrong.
10. Show that removing the comparison/shrink/decision loop collapses the experience—proving it is not just a polished test matrix.

Kill or park if divergences are dominated by environmental noise, if meaningful inputs require an LLM to invent them, if users still need to inspect all diffs to decide, if generated contracts are brittle snapshots, or if the demo cannot beat a script combining worktrees, Schemathesis, and ApprovalTests in comprehension or repeatability.

## Blunt verdict

**Proceed to deep dive at 7/10.** The user-facing primitive is memorable, the offline proof is real, the engineering is load-bearing, and the concept is cleanly separate from Wake. It is also more defensible than Threadline against currently visible commercial and open-source products.

The confidence is not higher because SpecFix and xTestCluster already own most of the conceptual academic territory, Diffy owns same-traffic service comparison and noise handling, and approval/visual/property-based tools own the major mechanics. The project should be presented as an ambitious synthesis with one new product object—**the Counterfactual Intent Decision**—not as the invention of differential testing, shrinking, behavioral clustering, or executable specifications.

The deep dive should try to kill three remaining claims:

1. **Usefulness:** do candidate disagreements reveal high-value tacit intent rather than mostly harmless implementation variance?
2. **Correctness:** can isolation, normalization, repeated execution, and structure-aware shrinking make the decisions stable enough to trust as questions?
3. **Compression:** can a serious non-expert resolve the ambiguity materially faster and more safely than reviewing the branches and tests directly?

If those survive, this is a rare heavy build whose interface, algorithms, and systems correctness all express the same idea.
