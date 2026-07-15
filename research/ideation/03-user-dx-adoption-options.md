# User, DX, and open-source adoption lane

**Authorial stance:** Soren Vale's user/DX and adoption specialist
**Research date:** 2026-07-14
**Wake snapshot inspected:** [`fff9bfa`](https://github.com/nelsonwerd/wake/tree/fff9bfa44a011b51c6fb7f188343a0352af1da79)
**Confidence:** 8/10 on the Wake collision; 7/10 on the user-job diagnosis; 5/10 on which product form will earn a habit. None of this establishes demand.

## Executive read: the provisional runtime direction collides with Wake

The current concept brief's candidate spine—local event journal, heterogeneous agent adapters, crash-safe continuation, replay/fork, causal artifact inspection, an effect ledger, and a mission-control viewer—is not merely similar to Wake. It is substantially the product Wake already is.

Wake's stated load-bearing idea is that **the run is a data structure**: a hash-chained event log acts simultaneously as orchestration state, coordination record, audit record, and time machine. It runs unmodified Claude Code, Codex, scripted workers, and arbitrary subprocesses through an NDJSON protocol; it implements crash resume, executor-free replay, O(1) fork lineage, effect tiers/classes, human gates, and a read-only causal timeline viewer. Its own positioning is "the recorder is the runtime." See Wake's [README](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/README.md), [concept model](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/concepts.md), [time-travel contract](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/time-travel.md), and [viewer surface](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/internal/view/static/index.html).

That collision is useful. It removes the temptation to spend this stress test rebuilding a less mature event runtime. There are only two honest paths:

1. Build a system with a different semantic center—most promisingly, **assurance about a software change** rather than durable execution of an agent run.
2. Explicitly build a humane application layer **on top of Wake**, contributing adapters or APIs upstream instead of copying its runtime. The user allowed this exception, but it must be described as a Wake application, not disguised as a new substrate.

My recommendation is path 1. The strongest system concept from the user/DX lane is an **assurance graph and proof-carrying change package**: a local-first tool that connects human intent, falsifiable claims, changed artifacts, evidence requirements, actual receipts, remaining uncertainty, and reviewer decisions. Wake and didrun can be evidence providers. Git remains content truth. The new system does not supervise agents, own effects, resume processes, or replay timelines.

## What Wake makes off-limits

| Proposed primitive | Wake already ships or explicitly owns | Decision here |
|---|---|---|
| Append-only event journal as run truth | Hash-chained SQLite event log; pure reducer | Reject as the new product's core |
| Heterogeneous local agent runtime | Unmodified subprocess agents over NDJSON | Reject |
| Crash resume | Supervisor epochs, process fencing, redelivery, effect recovery | Reject |
| Replay and fork | Executor-free replay; O(1) timeline fork; branch diff | Reject |
| Effect ledger / approval gates | Brokered, gated, and observed tiers; P/E/R/X classes; gate events | Reject |
| Causal run dashboard | Run picker, run/agent/effect/gate summaries, causal timeline, event inspector | Reject a generic version |
| Corruption-evident provenance | Hash chain, state hash, verification | Reject |
| Portable work handoff | Wake does not yet own a semantic change-assurance package | Open, if it is not process resumption |
| Intent-to-evidence coverage | Wake records what happened but does not model whether the change's intended claims are adequately evidenced | Open |
| Pre-change behavioral rehearsal | Wake can fork a run, but does not predict a code change's blast radius or compare predicted vs actual impact | Open |
| Human comprehension / ownership transfer | Wake exposes exact events, not whether a non-expert can safely maintain the resulting code | Open |

Wake is especially explicit about its honesty boundaries: a chain is corruption-evident rather than tamper-evident; logs can contain secrets; a branch can be inspected but is not yet runnable through the CLI; one real-model fleet run is not broad validation. Those boundaries are strengths, not openings to reproduce the same features under friendlier names. See its [security posture](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/security.md) and [honest prior-art map](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/PRIOR_ART.md).

## The real user jobs in 2026

The beachhead is not "anyone who uses AI." It is a serious non-expert or mid-level builder with a 5k–100k LOC application, at least one real user or operational consequence, and enough AI fluency to run multiple tools. They can make software work. Their missing capability is **responsible confidence**.

Two current studies put shape around that gap. In 22,953 pull requests, lower-experience vibe coders changed more files, received 4.52x more review comments, had 31% lower acceptance, and remained open 5.16x longer than higher-experience peers; the authors describe verification burden shifting to maintainers ([Asdaque et al.](https://arxiv.org/abs/2602.23905)). A survey of 162 vibe coders found risk awareness across experience groups but an experience-dependent ability to evaluate, debug, and verify—the "perception-action gap" ([Fawzy, Tahir, and Blincoe](https://arxiv.org/abs/2605.24521)). Those are not product-demand studies, but they strongly argue that generation is no longer the only bottleneck.

The jobs are:

1. **Before an agent edits:** turn "make auth safer" or "redesign onboarding" into a small set of observable promises, non-negotiable invariants, and explicit unknowns without learning formal methods first.
2. **While work is happening:** know where human attention is useful. The user does not want to spectate token streams; they want to see a constraint violated, a risky area touched, or a proof gap opened.
3. **Before shipping:** answer "What exactly do I believe about this change, and what evidence supports each belief?" A green test suite is evidence, not a complete answer.
4. **When something fails later:** recover the intended behavior, assumptions, and known blind spots—not the entire chat transcript.
5. **When asking for expert help:** deliver a compact review object that makes the expert's scarce time land on unresolved risks rather than archaeology.
6. **When switching tools:** keep intent and assurance state portable without pretending model sessions themselves are deterministic or interchangeable.
7. **While learning:** close the perception-action gap by explaining *why a check matters* and what remains untested, without turning the product into a condescending course.

### The emotional job

The user wants to move from "the app seems to work" to "I know what I checked, what I did not check, and what would change my mind." The product must create warranted calm, not a confidence score that launders uncertainty.

### Trust language

The UI should never flatten state to red/green. A useful assurance vocabulary is:

- **SUPPORTED:** named evidence currently satisfies the claim's declared policy.
- **CONTRADICTED:** at least one valid receipt falsifies the claim.
- **UNKNOWN:** required evidence has not run or cannot be observed.
- **STALE:** evidence predates a relevant artifact or policy change.
- **WAIVED:** a human knowingly accepts the unresolved risk, with reason and scope.
- **UNRECEIPTED:** a narrator or agent says something passed, but no accepted receipt supports it.

Every status should expose the exact evidence and policy that produced it. "Supported" must never be rendered as "correct."

## Adoption principles: earn permission gradually

Open-source adoption pull is plausible only if the first run is useful before the user restructures their workflow.

1. **Read-only first contact.** Point the tool at an existing branch plus a plain-language goal. It should produce a useful map without installing hooks, starting a daemon, or uploading code.
2. **A valuable residue.** The output should remain useful if the user uninstalls immediately: a portable JSON package and a self-contained HTML review artifact.
3. **No model required for the golden path.** A model may propose claims or explain evidence only when the user supplies one. Deterministic schema validation, Git inspection, test receipt ingestion, and rendering must work offline.
4. **Unknown by default.** Missing adapters or evidence create visible unknowns, never synthetic green checks.
5. **Meet existing practice.** Ingest Git diffs, JUnit/TAP, SARIF, coverage, didrun receipts, OpenTelemetry spans, CI metadata, and optionally Wake runs. Do not require a new executor. OpenTelemetry already provides a GenAI trace substrate ([OpenTelemetry](https://opentelemetry.io/blog/2026/genai-observability/)); GitHub exposes agent lifecycle hooks for validation, audit, and policy ([GitHub Docs](https://docs.github.com/en/enterprise-cloud@latest/copilot/concepts/agents/hooks)). The opportunity is semantic composition, not another proprietary log.
6. **Progressive rigor.** Day one: three claims and a branch. Week two: repo policies, reusable evidence rules, reviewer attestations, and CI gating.
7. **CLI/UI parity.** The monochrome CLI must give a decisive next action. The visual surface earns its existence by revealing many-to-many relationships among claims, files, and evidence—not by animating agents.
8. **Local and inspectable.** No account, no mandatory cloud, no opaque scoring model, no source upload. A single command should open the review surface on localhost.
9. **Extension seams that matter.** The natural OSS contribution surfaces are analyzers, receipt importers, policy packs, and renderers—not an endless agent-logo matrix.
10. **Do not make a standard the activation gate.** A portable change package can become a protocol after users value the product. Leading with "a new spec" is a reliable way to build an elegant empty room.

## Candidate forms and wedges

Scores are directional, not pseudo-precision. `Novelty` means the composition feels meaningfully different in use; `Habit` means a serious builder could adopt it repeatedly; `Proof` means the core can be demonstrated locally without fake APIs; `Collision` is reverse-scored (10 means safely far from Wake).

| Rank | Candidate | Novelty | Habit | Proof | Collision | Verdict |
|---:|---|---:|---:|---:|---:|---|
| 1 | Assurance graph + proof-carrying change | 8 | 9 | 9 | 9 | **Advance** |
| 2 | Repository flight simulator | 9 | 8 | 6 | 9 | **Advance, tightly scoped** |
| 3 | Wake Studio, a humane application on Wake | 7 | 8 | 8 | 10 if it imports Wake; 1 if reimplemented | **Viable exception** |
| 4 | Agent conformance range + capability passports | 8 | 6 | 8 | 8 | **Keep as infra option** |
| 5 | Reverse onboarding / ownership-transfer compiler | 7 | 8 | 7 | 10 | **Keep as a mode, not the spine** |
| 6 | Human attention router for AI changes | 7 | 9 | 6 | 9 | **Fold into candidate 1** |
| 7 | PATCH/1 portable change capsule | 8 | 5 | 9 | 9 | **Make an internal format first** |

### 1. Assurance graph + proof-carrying change

**One-line promise:** Every AI-assisted change carries a legible map from what the human intended to what was changed, what was checked, what was falsified, and what remains a bet.

**Semantic center:**

```text
Claim -> Evidence requirement -> Receipt -> Coverage decision
   \             |                  |              /
    -> Artifact scope -> Risk -> Human attestation
```

The unit of truth is not a run event; it is a **falsifiable claim about a change**. Examples: "unauthenticated users cannot read `/billing/history`," "migration rollback preserves existing rows," or "the redesigned route remains usable at 375px." Each claim names its artifact scope, falsifier, required evidence kinds, freshness rule, and risk. Receipts can arrive from didrun, CI, local commands, browser checks, a human review, Wake, or other providers. A deterministic evaluator computes the trust state. An agent can propose claims, but cannot declare them supported.

**Why it is different:** Wake answers "what did this fleet do, and is the record internally consistent?" This system answers "what do we believe about the resulting software change, and is the evidence sufficient under an explicit policy?" Git, Wake, didrun, OTel, and CI become inputs. None owns the semantic assurance graph.

**Adjacent competition:** VibeContract proposes decomposing natural-language intent into tasks and contracts linked to code and runtime verification ([Wang](https://arxiv.org/abs/2603.15691)). GitHub Spec Kit turns scenarios into specs, plans, tasks, checklists, and implementation workflows ([Spec Kit](https://github.com/github/spec-kit)). A credible build must go beyond generating contracts: it must ingest real heterogeneous receipts, evaluate freshness and contradictions deterministically, preserve unknowns, and export a review object that survives the agent workflow.

**One-session demo:**

1. Run `assure open --base main --goal docs/change.md` on a nontrivial example branch.
2. The CLI reports: `8 claims · 3 supported · 1 contradicted · 3 unknown · 1 stale` and prints the safest next action.
3. The browser shows a three-column scarlet-thread view: **intent/claims -> changed artifacts -> evidence/unknowns**.
4. Select `auth/session.ts`; the UI explains which claims it can affect, the exact receipts supporting them, and one contradicted cookie-policy test.
5. Run the suggested test through didrun. The receipt enters the graph, but a second claim remains unknown because no browser check exists.
6. Export `change.assurance.json` plus a self-contained review HTML. No API credentials and no fake agent are involved.

**Two-week useful habit:** Start each meaningful branch with 3–8 claims; let agents and checks attach evidence during work; run `assure gate` before merge; send the compact unresolved-risk view to a human reviewer; archive the package with the release. Reusable repo policies gradually reduce setup.

**Aha:** clicking any changed file reveals *why it changed, what promise it participates in, what check bears on that promise, and what nobody has checked*. The delight comes from collapsing an hour of archaeology into ten seconds.

**Risks:** contract authoring tax; false precision; semantic mappings that look smarter than they are; adjacent VibeContract/Spec Kit functionality; noisy receipt adapters. Mitigation is a tiny manual schema, deterministic evaluation, explicit `UNKNOWN`, no global quality score, and a read-only first-run inference labeled as hypotheses.

### 2. Repository flight simulator

**One-line promise:** Rehearse a proposed AI change against a locally constructed behavioral/structural twin, then compare predicted blast radius with what the patch actually touched and broke.

**Product form:** Build a repository model from imports, routes, schemas, tests, runtime traces, configuration, and user-declared invariants. Before edits, a plan names intended surfaces. The simulator produces an inspectable impact hypothesis and a set of local probes. After edits, it overlays actual file/test/runtime effects and highlights surprises: unpredicted dependencies, unexercised paths, schema drift, and changed behavior outside the declared radius.

This must not claim to simulate arbitrary software perfectly. The honest primitive is a **counterfactual impact hypothesis with executable probes**, not a digital twin oracle.

**One-session demo:**

1. Open an example SaaS repo and propose: "support team members can view invoices but never change payment methods."
2. The pre-change view highlights the auth policy, two routes, a service, a cache key, three tests, and a database read path. It labels inferred edges separately from observed runtime edges.
3. Two local worktree implementations are provided as prebuilt fixtures, not AI-generated fakes. The simulator runs the same probes against each.
4. Variant A unexpectedly invalidates an admin cache; Variant B touches fewer surfaces but lacks a tenancy guard. The UI shows predicted vs actual impact and why each surprise matters.
5. Export the safer experiment as an assurance claim set for candidate 1.

**Two-week useful habit:** Use `flight plan` before cross-cutting changes or large agent jobs; accept/edit the impact hypothesis; use `flight compare` after each substantial patch; add one missing invariant or probe when a surprise appears. The repo model becomes more useful through ordinary use.

**Aha:** "I can see the part of the system the agent did not realize it was changing before I merge it."

**Risks:** enormous analyzer surface, language/framework coupling, false-negative danger, expensive runtime capture, and a demo that overfits a fixture. The v1 must lock to one ecosystem and show epistemic labels: `STATIC`, `OBSERVED`, `DECLARED`, `INFERRED`, `UNKNOWN`. It is the most eyebrow-raising option but the least likely to reach reference quality in one stress-test run unless scoped brutally.

### 3. Wake Studio: a humane application on Wake

**One-line promise:** Turn Wake's durable fleet substrate into a guided creative-engineering workspace a serious vibe coder can operate without learning event sourcing, effect classes, or fleet protocols.

**Boundary:** This is legitimate only if it consumes Wake as the runtime and says so prominently. It must not recreate Wake's log, reducer, subprocess protocol, crash recovery, fork semantics, effect ledger, or chain verifier. Any missing core capability becomes a Wake contribution or an explicit limitation.

**Product form:** A project-oriented desktop/local web app over Wake runs. The user's objects are outcomes, constraints, branches, review questions, and safe next actions. Wake events remain available through progressive disclosure. The app can add opinionated templates, human gates in plain language, a visual branch comparator, evidence annotations, and a one-click handoff bundle.

**One-session demo:**

1. `studio open .` detects Wake and imports an existing completed, resumed run family.
2. A guided "new mission" flow generates a transparent fleet spec and shows exactly what commands/processes will run.
3. The user kills the demo fleet, reopens Studio, and sees: "Work stopped after planner output; two workers have unconsumed events; resume is safe because no Class X effect is in doubt."
4. The user resumes, compares root vs fork in a visual lane, and exports a handoff.

**Two-week useful habit:** Start and monitor every multi-agent change from Studio, approve risky effects, resume interrupted work, and compare alternative branches—while learning Wake's underlying vocabulary only as needed.

**Aha:** a non-runtime engineer survives a crash and understands the next safe action without reading a raw log.

**Risks:** Wake currently has no adoption signal and calls its surfaces first draft. A wrapper cannot validate demand the substrate does not have. Product/application coupling may constrain iteration. The idea is also less novel as infrastructure; its value is exceptional craft and accessibility. If selected, the brief must explicitly say **"Wake application"** and coordinate with its API and roadmap.

### 4. Agent conformance range + capability passports

**One-line promise:** Before trusting a new coding agent, model, permission profile, or upgrade on a real repo, run it through repo-specific adversarial drills and get a portable capability passport.

The system defines behavioral scenarios: preserve an invariant, handle a failing test without weakening it, refuse a secret, recover from misleading context, make a minimal diff, explain uncertainty, and leave a handoff. It drives any user-supplied real agent interface, records outcomes, and grades only machine-observable properties. External model/API legs remain human-gated; deterministic local worker fixtures validate the harness, not model quality.

**Aha:** two agents both say "done," but the passport shows one weakened a test and the other preserved the invariant. The user selects an executor and permission profile based on evidence rather than reputation.

**Habit/adoption:** run on agent upgrades, new repo onboarding, and before granting higher permissions. OSS pull could come from shared conformance scenarios and adapter contributions. Daily habit is weaker than candidate 1, and benchmarks invite gaming, so this is a strong infrastructure project but a narrower user product.

### 5. Reverse onboarding / ownership-transfer compiler

**One-line promise:** Turn an AI-built subsystem into an interactive maintenance map its human owner can actually operate when the agent is gone.

Inputs are code, Git history, tests, runbooks, and change packages. Output is not generated documentation alone: it is a compact architecture/invariant map plus executable failure drills. The user rehearses "session cache is stale," "migration halfway failed," or "provider timeout" and learns the real recovery path. The tool records what remains undocumented or unexercised.

This serves the perception-action gap directly and is safely orthogonal to Wake. However, as a standalone product it risks becoming an AI explainer or tutorial generator. The stronger use is a **Maintainer Mode** in the assurance system: every high-risk change must leave behind an ownership-transfer view and at least one runnable failure drill.

### 6. Human attention router for AI changes

**One-line promise:** Spend the human's next ten minutes only where their judgment can change the outcome.

The router consumes diffs, claim coverage, static-analysis findings, test receipts, ownership, and agent uncertainty. It emits a ranked queue of concrete decisions: "Confirm this authorization behavior," "No evidence covers rollback after row 10,000," or "Two implementations disagree on tenancy semantics." It is not another alert feed; every item includes why human judgment is necessary, what decision is requested, and what will happen next.

The UX job is powerful and habitual, but a standalone prioritizer cannot be trusted without the assurance graph beneath it. Fold it into candidate 1 as the home screen and CLI next-action engine. Never use an opaque risk score as the primary explanation.

### 7. PATCH/1 portable change capsule

**One-line promise:** Make an AI-assisted software change a portable, inspectable artifact—intent, patch, policies, receipts, unresolved risks, and attestations—rather than a transient chat session.

The container could use content-addressed blobs plus a canonical manifest. It can reference Git objects, didrun receipts, Wake run heads, CI artifacts, SARIF, and human decisions without copying secret-bearing logs by default. A verifier checks internal consistency and missing blobs; it does not claim the contents are true or complete.

This is clean systems work and safely distinct from Wake: Wake's durable artifact is a run family; PATCH/1's artifact is a reviewable software change and its assurance state. But "a new standard" has no inherent user pull. Build the format as candidate 1's internal/export schema, publish it only after the product proves interchange value.

## Explicitly rejected concepts

These should not survive convergence unless the brief says they are direct Wake contributions:

1. **A vendor-neutral local agent build runtime.** That is Wake's category and architecture.
2. **An append-only causal run journal with hash chains.** Wake implements it deeply; OTel and multiple recorders cover adjacent observability.
3. **Cross-agent crash resume through a subprocess adapter SDK.** Wake's supervisor/protocol already does this.
4. **A time-travel UI for agent runs.** Wake ships replay/fork/diff semantics and a causal family viewer. Better visual polish alone does not justify a second runtime.
5. **An effect broker with approval gates and exactly-once language.** Wake has a detailed tier/class model and a witnessed-effect claim with explicit caveats.
6. **A generic "mission control for agents."** This is a crowded surface and usually optimizes for watching work, not making a better decision.
7. **A context capsule whose primary promise is resuming another agent.** Without a new semantic object it is a handoff file; with durable runtime semantics it collides with Wake.
8. **A prompt/spec generator.** Spec Kit and many workflows already cover spec/plan/task generation; VibeContract is directly adjacent on intent/contracts/verification.
9. **A universal quality score for AI code.** It would collapse incompatible evidence and hide uncertainty—the exact opposite of trustworthy UX.

## Substitute and competition map

| User reaches for | What it already does | Remaining opening |
|---|---|---|
| Git + PR | Content history, diff, review conversation, merge gate | Intent and evidence are fragmented; no explicit claim-to-proof semantics |
| Test runner / CI | Executes checks and reports pass/fail | Does not establish which user promises a check bears on or what remains unknown |
| didrun | Receipts and claim verification around concrete commands | Excellent evidence provider; the higher-level change assurance graph is still open |
| Wake | Durable, corruption-evident local fleet execution, resume/replay/fork, causal inspection | Semantic assurance of the resulting software and novice-safe ownership transfer |
| OpenTelemetry | Interoperable traces/spans and GenAI observability | A trace says what happened, not whether intended change claims are adequately supported |
| GitHub agent hooks | Policy, lifecycle validation, audit hooks | Platform-bound event points; no portable assurance object |
| Spec Kit | Spec, plan, task, checklist, implementation workflow across many agents | Strong intent workflow; weaker heterogeneous receipt evaluation and post-change proof object |
| VibeContract | Research vision for tasks, contracts, traceability, testing, and runtime verification | Close conceptual neighbor; differentiation requires shipped receipt semantics and an honest user-facing uncertainty model |
| Static analysis / SARIF | Concrete findings with location/severity | One evidence type; no connection to the user's complete change intent |
| AI PR reviewer | Comments and suggested defects | Another probabilistic opinion unless grounded in independently checkable evidence |

## Recommended product direction from this lane

Advance **candidate 1: Assurance graph + proof-carrying change**, while borrowing two components rather than expanding the product indiscriminately:

- Candidate 6 becomes the primary interaction: the home screen is the next human decision, not a run dashboard.
- Candidate 7 becomes the internal/export format: every change is a portable package, but the project does not lead with standards work.
- Candidate 5 becomes one completed-state view: "Could a human maintain this tomorrow?"
- Candidate 2 remains a later, high-ambition module after the assurance semantics are proven.
- Wake is a first-class **optional evidence importer**, alongside didrun and Git, never a dependency for the initial local proof.

### Hard architecture boundary

The new system must not contain a process supervisor, fleet scheduler, effect executor, crash-resume engine, event replay engine, or hash-chained run log. If any appears in the architecture, stop and ask whether it belongs upstream in Wake.

### Revised smallest proof

The existing success metric and kill criterion are contaminated by Wake because they center cross-adapter interruption/resumption. Replace them for this direction:

- **Success:** In under 15 minutes on a nontrivial example branch, a mid-level vibe coder can declare or accept 5–8 falsifiable change claims, ingest real Git/didrun/test evidence, find at least one meaningful unsupported or contradicted claim that a green-looking diff obscures, run the suggested local check, and export a portable reviewer package whose every trust state is reproducible from its contents.
- **Aha:** select any changed artifact and see the scarlet thread from human promise to artifact to exact evidence to remaining unknown—then get one safe next action.
- **Kill:** park if the product cannot catch a meaningful proof gap on at least two substantively different changes, if users still need to read raw test/log output to understand the verdict, if the deterministic evaluator cannot reproduce all statuses offline, or if the experience collapses to a prettier checklist plus CI links.

### One crucial design choice

Do not use chronological events as the primary visual axis. Wake already does that well. Use a spatial assurance graph:

```text
PROMISES                 SOFTWARE                 EVIDENCE

What must remain true -> What can affect it    -> What actually tested it
What is newly true      -> What changed        -> What contradicted it
What is out of scope    -> What was surprised  -> What nobody checked
```

Time and agent identity remain available in the inspector. The default question is not "What are the agents doing?" It is "What do I have reason to believe?"

## Adoption experiments before broadening scope

These are human gates, not things an autonomous build can validate:

1. Give five serious vibe coders an existing risky branch and compare their time-to-identify the most important proof gap with and without the tool.
2. Observe whether they can explain `SUPPORTED`, `UNKNOWN`, and `WAIVED` without coaching—and whether any interpret supported as guaranteed correct.
3. See whether the exported package reduces an experienced reviewer's archaeology time or merely adds another artifact to read.
4. Measure week-two return: did users create a second claim set without prompting?
5. Ask maintainers whether a package changes their willingness to review AI-assisted contributions. Do not infer this from stars.
6. Test the read-only first run on unfamiliar repos. If useful output requires a day of policy authoring, the wedge is wrong.

## Bottom line

Wake eliminates the provisional runtime concept as a clean-sheet build. That is a gift: the higher-value, more user-legible problem remains unsolved. Build the layer that turns intent and raw execution evidence into an honest, portable assurance object. Let Wake prove what happened inside a fleet. Let didrun prove what commands were receipted. Let Git prove the content. The new system's job is to show, without theater, what those facts do and do not justify believing about the software change.
