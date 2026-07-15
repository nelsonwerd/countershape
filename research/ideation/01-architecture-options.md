# Architecture divergence: systems that do not rebuild Wake

- **Author lens:** Soren Vale architecture/novelty lane
- **Date:** 2026-07-14
- **Status:** ideation evidence, not a build decision
- **Wake baseline inspected:** `nelsonwerd/wake` at `fff9bfa44a011b51c6fb7f188343a0352af1da79` (2026-07-14)

## Executive cut

The provisional “agent-native build runtime” in the concept brief is no longer defensible as written. [Wake already runs heterogeneous, unmodified subprocess agents inside a local, corruption-evident, crash-durable event log](https://github.com/nelsonwerd/wake): its log is the execution substrate, coordination bus, replay source, and fork spine. Rebuilding a local event journal, subprocess adapters, crash-resume, replay, effect receipts, and a causal viewer would be the same core system with fewer correctness miles.

That is useful pressure, not bad news. It moves the search away from “make agent execution durable” toward higher-order objects that Wake does not model:

1. the **behavior of the software being changed**;
2. **counterfactual candidate worlds** and their trade-off frontier;
3. **semantic changes** that survive file movement;
4. **least-authority plans** inferred before execution;
5. **human outcome contracts** with explicit proof gaps; and
6. a deliberately user-friendly **application layer over Wake**, if the project chooses leverage over greenfield novelty.

My strongest recommendation is **Threadline**, a local behavior-to-code causal workbench. Its key inversion is memorable: *trace the product, not the agent*. A vibe coder demonstrates the behavior in the running application; Threadline follows that interaction through UI state, network calls, server spans, data effects, and source symbols, turns the causal slice into an agent-readable change capsule, then replays the same behavior and shows the before/after evidence. This is ambitious infrastructure with a human surface at its center, and it remains useful regardless of which coding agent produced the patch.

The runner-up is **BranchLab**, a counterfactual patch tournament. It is more immediately legible and easier to demo, but parallel-agent products are already crowding around worktree orchestration. Its novelty survives only if the product is about *behavioral differential comparison and splicing*, not launching more agents.

## Evidence frame and non-duplication bar

Several 2026 substrates are already real:

- [OpenAI's Codex app](https://openai.com/index/introducing-the-codex-app/) already manages parallel agents in isolated worktrees and exposes diffs. “Run several agents” is not a wedge.
- [GitHub Copilot hooks](https://docs.github.com/en/copilot/reference/hooks-reference) already provide pre/post tool, permission, stop, subagent, and error lifecycle points. “Add hooks and log calls” is not a wedge.
- [OpenTelemetry](https://opentelemetry.io/blog/2026/genai-observability/) already models agent/model/tool traces, while its [HTTP semantic conventions](https://opentelemetry.io/docs/specs/semconv/http/) and trace context provide the substrate for cross-process causality. “Show an agent trace” is not a wedge.
- [Playwright Trace Viewer](https://playwright.dev/docs/trace-viewer) already records action-level DOM snapshots, screenshots, source locations, console output, and network traffic. “Record browser tests” is not a wedge.
- [Microsoft Research's RPG-Encoder](https://www.microsoft.com/en-us/research/publication/closing-the-loop-universal-repository-representation-with-rpg-encoder-2/?lang=en) and 2026 code-graph work make static repository structure a crowded field. “Put the AST in a graph and expose MCP” is not a wedge.
- [VibeContract](https://arxiv.org/abs/2603.15691) already decomposes natural-language intent into task contracts tied to code and runtime verification. “Generate a spec and tests” is not a wedge.
- Wake already owns the combination of a harness-agnostic subprocess protocol, hash-chained event log, effect ledger, crash recovery, replay, and event-granularity forking. “Proof-carrying agent runs” is not a new concept in this workspace.

The options below are therefore judged on whether their **primary semantic object** remains valuable if Wake, Git, OTel, Playwright, and an existing agent UI are all available underneath.

## Comparative scorecard

Scores are directional, 1–5. “Proofability” means a local, honest proof without a gated model API; it does not mean production readiness.

| Rank | Concept | Primary object | Eyebrow factor | User legibility | Proofability | Architecture depth | Wake non-duplication | Verdict |
|---:|---|---|---:|---:|---:|---:|---:|---|
| 1 | **Threadline** | Behavior thread | 5 | 5 | 4 | 5 | 5 | Pursue |
| 2 | **BranchLab** | Minimal behavioral divergence | 5 | 5 | 4 | 5 | 5 | Keep alive |
| 3 | **Patchwork Calculus** | Semantic change program | 5 | 3 | 3 | 5 | 5 | High-risk research bet |
| 4 | **Capability Flight Plan** | Least-authority manifest | 4 | 4 | 2 | 5 | 4 | Security-heavy, valuable |
| 5 | **Outcome Circuit** | Executable outcome contract | 3 | 5 | 4 | 4 | 5 | Useful but adjacent-heavy |
| 6 | **Wake Studio** | Human mission projection | 3 | 5 | 5 | 3 | 5 if it imports Wake; 1 if it reimplements it | Pragmatic leverage option |

---

## Option 1 — Threadline: point at behavior, reveal the code that makes it true

### One-line promise

Perform a behavior in your local app, and Threadline turns it into a replayable causal thread from pixels and gestures through requests, runtime spans, data effects, source symbols, and verification—then gives any coding agent only the slice it needs to change that behavior safely.

### User and pain

The beachhead user has a working TypeScript web app and can describe what they see, but cannot reliably answer:

- Which code actually produced this UI state?
- What backend path and data effects did this click trigger?
- What else shares that path?
- Did the agent change only the intended behavior?
- Can I replay the exact journey and compare before versus after?

File trees and diffs are implementation-shaped. The user's intent is behavior-shaped. Agents spend tokens rediscovering the bridge on every run, and the user reviews output at the wrong abstraction level.

### System insight

The missing object is a **Behavior Thread**:

```text
user action
  -> visible state transition
  -> browser/network event
  -> server/runtime spans
  -> data or external effect
  -> executing source symbols
  -> asserted outcome
```

The thread is neither an agent trace nor a static code graph. It is an executable, versioned slice of product behavior. It can become all of the following without changing identity:

- a context boundary for an agent;
- a regression scenario;
- a blast-radius map;
- a before/after comparator;
- a human-readable explanation of a feature; and
- a durable handoff artifact.

The inversion is important: agent actions are incidental. The application behavior is the stable truth the human cares about.

### Architecture

1. **Interaction recorder.** A Playwright-backed local browser records gestures, accessibility locators, DOM snapshots, screenshots, console messages, and network events. Playwright already proves these can be captured and inspected locally; Threadline's work begins where Trace Viewer stops.
2. **Trace stitcher.** Each browser action receives a thread id propagated through HTTP headers into OTel spans. Async boundaries retain causal links rather than relying only on timestamps. Unsupported breaks are shown as breaks, never guessed into continuity.
3. **Runtime-to-source resolver.** Source maps, stack frames, route metadata, and an incremental AST/symbol index map spans to source symbols and call sites. Dynamic evidence and static reachability are stored separately so “observed” never silently becomes “all possible.”
4. **Local graph store.** SQLite stores append-only observations plus rebuildable projections: Action, ViewState, Request, Span, Effect, Symbol, Artifact, Assertion, and Gap. Blob content is content-addressed; secrets and payload values are opt-in and redacted before persistence.
5. **Scenario compiler.** A recorded journey becomes editable steps and explicit assertions. The compiler flags weak or accidental assertions—timestamps, random ids, layout noise—instead of presenting brittle replay as proof.
6. **Context cutter.** CLI and MCP read APIs emit a bounded capsule containing the observed symbol slice, nearby static dependents, behavior contract, current gaps, and replay command. This is a query tool for any agent, not a model gateway.
7. **Behavioral differ.** Before/after runs compare visible states, accessibility trees, request/response schemas, trace topology, data-effect classes, latency envelopes, and code-symbol reach. It reports changed evidence and missing evidence separately.
8. **Mission-control UI.** The main canvas is a spatial thread, not a chat: behavior at left, implementation at right, gaps and assertions between. A user can scrub time, select a pixel or request, reveal source, pin an invariant, and export a change capsule.

### Why now

[Playwright now exposes an agent-oriented CLI and local trace artifacts](https://playwright.dev/agent-cli/commands/tracing), while OTel provides cross-service trace context and common HTTP semantics. At the same time, current research is converging on structure-aware repository representations rather than blind file retrieval. The ingredients exist; the opportunity is a product-level causal join between *what the user did* and *what the code did*.

### Existing substitutes and the non-duplication case

- **Playwright Trace Viewer** understands a test journey but does not resolve a browser gesture into a cross-process source-symbol slice, compile that slice for coding agents, or compare semantic topology before and after.
- **APM/OTel backends** understand runtime spans but are operations-shaped, often cloud-first, and rarely retain DOM/action context or emit a bounded code-change capsule.
- **Static code graphs** show possible relationships but not which path produced the behavior the user just demonstrated.
- **IDE “go to definition”** starts from a symbol; Threadline starts from the product.
- **Agent traces** explain what an agent did. Threadline explains what the application did, so it survives model and tool churn.

This is still a positioning hypothesis. A deep dive must look specifically for observability vendors, browser-to-backend debugging products, and “feature map” code-intelligence systems that already join these layers.

| Wake overlap | Material difference | Safe relationship |
|---|---|---|
| Both produce causal views and local artifacts. | Wake's subject is the **agent fleet execution**; Threadline's subject is the **application behavior under change**. Threadline does not implement agent supervision, crash-resume, hash chains, effect replay, or timeline forks. | A Threadline capsule could be input/output evidence inside a Wake run. Wake is optional infrastructure, not copied code. |

### Offline/local proof path

Build one instrumented local TypeScript application with a browser, API server, and SQLite data layer. Record a three-step user journey; show an action propagating into an API span and database operation; resolve the observed path to exact source symbols; emit a context capsule through both CLI and local MCP; make a known patch; replay; and show a truthful before/after graph with one deliberately broken trace edge labeled as a gap. No model call is required to prove the spine.

### Heavy-but-coherent MVP

- TypeScript/JavaScript monorepos only.
- Chromium + Playwright recorder.
- Node HTTP instrumentation and SQLite adapter; generic OTLP ingest for other local services.
- Source-map and Tree-sitter/TypeScript symbol resolution.
- Scenario/assertion DSL with record, replay, and compare.
- Redaction rules and capture budget.
- CLI, read-only MCP server, and responsive visual thread explorer.
- Demo application with active, completed, failure, missing-instrumentation, and redaction states.
- Contract tests for causal propagation, source resolution, redaction, deterministic projection, and honest gap handling.

### Key failure mode

If useful causal continuity requires invasive manual instrumentation, or if source attribution is too noisy to reduce agent context materially, the concept collapses into “Playwright trace plus Jaeger plus a code graph.” The smallest proof must show that selecting a visible behavior yields a precise, actionable code slice and that replay catches a meaningful behavioral delta. A pretty graph alone fails.

**Architecture confidence: 7/10.** The substrate is buildable; the usefulness of the join and the accuracy of source resolution need adversarial proof.

---

## Option 2 — BranchLab: a counterfactual ambiguity laboratory for software changes

### One-line promise

Import several plausible Git branches, drive them with identical inputs, and shrink every normalized behavioral disagreement into the smallest human intent decision—then preserve that decision as an executable contract instead of merely choosing a branch.

### User and pain

Parallel agents increase solution throughput but leave the user with parallel diffs and duplicated review work. More subtly, two candidates can both pass the written tests while making different product decisions the human never knew they needed to specify: whether a modal preserves draft input, whether a retry duplicates a toast, whether sorting is stable, whether an empty state is explanatory or silent. The user cannot easily discover these latent ambiguities, let alone turn the chosen behavior into durable intent. Existing multi-agent interfaces mostly optimize launch and isolation; selection remains intuition plus test status.

### System insight

Treat a set of branches as a **counterfactual experiment**. Each branch receives the same fixtures, clocks, random seeds, browser actions, requests, and environment. Observations are normalized into comparable behavior rather than raw pixels or timestamps. The system then performs delta-debugging over input sequences, fixture state, viewport, timing envelope, and observation fields to find a **minimal behavioral divergence**: the smallest reproducible case in which plausible implementations disagree.

That divergence is presented as an intent question the human can actually answer:

> After saving a renamed filter, should Back restore the old draft or the persisted name?

The answer becomes an **Intent Decision** containing the minimized fixture, input sequence, accepted outcome predicate, rejected alternatives, and provenance to the branches that exposed it. The decision compiles into an executable regression contract. BranchLab's lasting product is therefore not a winning branch or a score; it is a growing, executable boundary around the human's previously tacit product intent.

Pareto comparison still matters for performance, accessibility, dependency churn, and scope, but it is secondary. The non-obvious wedge is *ambiguity discovery through counterfactual execution*.

### Architecture

1. **Branch importer.** Accept arbitrary local Git refs or worktrees. Candidate creation by an agent is optional and out of the semantic core; branches can come from humans, agents, previous experiments, or upstream PRs.
2. **Controlled-world fabric.** Materialize one isolated world per branch with shared fixture declarations, deterministic clock/random providers where the application supports them, and explicit environment fingerprints. Environmental differences are evidence, not noise to hide.
3. **Identical-input driver.** Replay browser actions, HTTP/CLI requests, data fixtures, and event sequences against every world with synchronized checkpoints.
4. **Observation normalizer.** Convert DOM/a11y trees, screenshots, API shapes, database deltas, emitted events, logs, performance envelopes, and runtime traces into typed observations. Rules separately label ignored nondeterminism, tolerances, redactions, and lossy normalization.
5. **Divergence detector.** Align observations by semantic anchors and find the first disagreement, including a “cannot align” state. It never treats an absent observation as equality.
6. **Counterexample shrinker.** Delta-debug action sequences, fixture rows, request fields, timing windows, viewport/state dimensions, and observation projections until it finds a locally minimal reproducer. “Minimal” is relative to declared reducers and is not a global proof.
7. **Intent-decision compiler.** Present the minimized A/B/N-way outcome in product language. Human choices are `prefer A`, `prefer B`, `either is acceptable`, `neither`, or `needs a new distinction`. The accepted decision compiles into an outcome predicate and replay fixture, with the rejected observations retained.
8. **Assurance/receipt bridge.** Import test and command receipts—including didrun grades—verbatim. A receipt can establish that a probe ran on a given tree; BranchLab never upgrades that into correctness or invents a competing grade.
9. **Trade-off frontier.** Compare hard gates and measurements without collapsing them into a hidden scalar “AI score.” Subjective critic notes and missing evidence remain separate lanes.
10. **Decision-splice layer.** Cherry-pick commits where possible and otherwise create an explicit repair task with candidate evidence and Intent Decisions attached—never pretending semantic merge is automatic.
11. **Visual ambiguity bench.** Candidates are synchronized columns; the minimized input is a scrub-able timeline; conflicting observations are overlaid; and the center rail captures the human decision and generated contract.

### Why now

[Codex already makes parallel worktrees a mainstream agent interaction](https://openai.com/index/introducing-the-codex-app/), and current agent users run many workers at once. Once parallel generation becomes cheap, controlled comparison becomes more valuable than another launcher.

### Existing substitutes and the non-duplication case

- Codex and other multi-agent command centers isolate and review work, but do not define a same-intent experiment, normalize behavioral probes, or expose a trade-off frontier.
- CI matrices evaluate a known implementation across environments; BranchLab compares unknown implementations against a shared outcome contract.
- Agent benchmarks compare systems globally; BranchLab compares candidate patches for one user's live repository.
- Git worktrees provide isolation, not experimental semantics.
- Differential and property-based testing shrink failures against a known oracle. BranchLab uses disagreement between plausible implementations to *discover a missing oracle*, then asks the human to supply it.

### Compared with an assurance graph

An assurance graph and BranchLab answer different questions:

| System | Core question | Blind spot | Relationship |
|---|---|---|---|
| **Assurance graph** | “What evidence supports each declared claim?” | Two branches can satisfy every declared claim while making different, unstated product decisions. | Intent Decisions become new claim nodes; minimized runs become probes/evidence. |
| **BranchLab** | “Where do plausible implementations behave differently under the same inputs, and which difference reflects intent?” | Agreement is not correctness; all candidates can share the same bug, and a human can choose the wrong behavior. | Consume assurance state and add ambiguity-discovery evidence; never replace assurance. |
| **didrun** | “Did the claimed command/check run on the sealed tree, and what exact receipt grade did it earn?” | It intentionally does not decide whether the command is a sufficient product oracle. | Import verbatim grades. Do not recreate command receipts, claim grading, seals, or strict verification. |

This distinction is load-bearing. “Both candidates passed” is often the beginning of BranchLab's work, not the conclusion. Conversely, a minimized divergence with no receipted execution is an unverified observation, not a contract.

| Wake overlap | Material difference | Safe relationship |
|---|---|---|
| None is required. Importing Git branches and running application probes does not need agent-fleet semantics. | Wake makes agent execution durable/forkable. BranchLab's core is **N candidate code worlds + normalized identical-input execution + counterexample shrinking + human Intent Decisions**. It must not build a durable event log, process supervisor, effect ledger, replay engine, or agent-timeline fork. | Keep Wake entirely out of the MVP. A future adapter may ingest Wake-produced Git refs exactly as it ingests any other branch, but Wake internals remain off-limits. |

### Offline/local proof path

Create three honest local Git branches of a demo application, each implementing the same underspecified feature with a different edge-case behavior. Import the refs; boot identical isolated worlds; replay one recorded browser/API journey; normalize the results; shrink the failure from a long journey and populated database to the minimal action sequence and fixture row; ask the human to choose among the three visible outcomes; compile that choice into a Playwright/API contract; run it against all branches; and import the exact didrun receipt for the winning tree. No model call, simulated agent API, or Wake component is required.

### Heavy-but-coherent MVP

- Git-ref importer; world lifecycle, cancellation, and cleanup.
- Fixture/environment manifest with deterministic clock/random hooks where supported.
- Identical-input drivers for browser, HTTP, and CLI behavior.
- Typed observation schema for DOM/a11y, screenshot, API, data delta, event/log, trace, and performance evidence.
- Normalization policy with explicit lossy/ignored/redacted fields.
- Alignment engine and delta-debugging shrinker with local-minimality receipts.
- Intent Decision schema, human choice flow, and Playwright/HTTP/CLI contract compiler.
- Assurance/didrun receipt importer that preserves exact grades.
- Secondary evidence frontier with no hidden scalar score.
- Commit-level transplant plus explicit repair workflow.
- Monochrome CLI and responsive ambiguity bench.
- Cost/time budgets and a no-network local mode.

### Key failure mode

Nondeterminism can manufacture false divergences, while aggressive normalization can erase the difference that matters. Shrinking stateful UI/database behavior is computationally expensive and can create a smaller case with a different cause. Agreement cannot establish correctness, all candidates may share one bug, and a human can codify the wrong preference. If the system cannot turn a real, previously unstated disagreement into a small comprehensible decision and a stable executable contract, it is expensive branch theater. The concept becomes ordinary immediately if marketed as “run three agents and pick one.”

**Architecture confidence: 7/10.** Highly demoable; novelty depends on normalized counterfactual execution, useful shrinking, and intent compilation—not agent launching.

---

## Option 3 — Patchwork Calculus: semantic changes as executable programs

### One-line promise

Replace fragile byte patches with typed, inspectable change programs that can be composed, rebased, inverted, and repaired across an evolving codebase.

### User and pain

A vibe coder experiences a change as “add optimistic undo to saved filters,” while Git stores line differences. When another agent refactors the code, the human intent and the patch's structural assumptions separate. Cherry-picks conflict, follow-up agents rediscover intent, and a textual diff cannot say whether two changes commute.

### System insight

Make the first-class artifact a **Change Program**:

```text
preconditions
  + typed semantic operations
  + invariants and behavior probes
  + postconditions
  + provenance
```

Operations target symbols and relationships—“split function,” “introduce state transition,” “replace callers,” “add route guarded by invariant”—rather than line coordinates. A patch can then report whether it applies exactly, applies with a repaired binding, conflicts semantically, or violates its postconditions.

### Architecture

1. Lossless parser and symbol graph for one language family (TypeScript/TSX first).
2. Typed operation IR with stable symbol anchors, structural preconditions, scope/effect summaries, and reversible operations where honest.
3. Compiler from operation IR to concrete edits with formatter integration and generated source map from operation to bytes.
4. Algebra for composition, commutativity checks, inversion, rebase, and conflict explanation.
5. Observation bridge: behavior probes and runtime traces can become pre/postconditions but are not silently promoted into semantic proof.
6. Agent protocol: agents may propose plain diffs, but the system attempts to lift them into a change program and shows unlifted residue. Native agents can emit IR directly through MCP/CLI.
7. Visual change editor displaying intent blocks, dependencies, affected symbols, byte diff, and replayable operations.

### Why now

The industry is moving from text retrieval toward structured action spaces. [CODESTRUCT](https://aclanthology.org/2026.acl-long.607/) frames codebases as named AST entities with syntax-validating reads and edits; [RPG-Encoder](https://www.microsoft.com/en-us/research/publication/closing-the-loop-universal-repository-representation-with-rpg-encoder-2/?lang=en) explicitly treats generation and comprehension as inverse directions over a repository representation. The opportunity is not another code graph; it is version-control semantics over executable transformations.

### Existing substitutes and the non-duplication case

- Codemod engines and OpenRewrite perform semantic transformations but generally treat recipes as developer-authored migration tools, not the durable representation of every AI-generated change.
- Git/Sapling/Jujutsu improve history and branch ergonomics while remaining byte/tree based.
- Structural diff viewers explain changes but do not make them composable programs.
- AST editing tools validate syntax, not user-level intent or change algebra.

| Wake overlap | Material difference | Safe relationship |
|---|---|---|
| Both make a run/change more inspectable and forkable in a broad sense. | Wake's state is an event log for agent execution. Patchwork's state is a **typed program transformation and its applicability semantics**. There is no subprocess fleet, effect ledger, crash recovery, or hash-chain claim. | Wake may record who executed a Change Program; Patchwork defines what the code change means. |

### Offline/local proof path

For TypeScript only, encode six operation kinds; lift a real diff into the IR; apply it to revision A; refactor/move the target symbols in revision B; rebind and reapply; invert a reversible subset; compose two independent changes; and reject one semantic conflict with a useful explanation. Expose the same artifact in CLI and a visual operation timeline.

### Heavy-but-coherent MVP

- TypeScript/TSX parser, type checker integration, stable symbol identity strategy.
- Operation IR and JSON schema.
- Apply/validate/invert/compose/rebase commands.
- Lift-from-diff with explicit “unmodeled bytes” residue.
- Property tests and mutation corpus for formatting/comments/renames/moves.
- Agent MCP surface and visual change-program editor.
- Git interoperability: every application still produces an ordinary commit.

### Key failure mode

General program semantics are undecidable, dynamic behavior defeats static anchors, and seemingly simple refactors destroy identity. If most real agent diffs lift into a large opaque “replace node” operation, the algebra is decorative. The proof must demonstrate repair across meaningful code motion, not curated renames.

**Architecture confidence: 5/10.** Potentially category-defining, but the research risk is much higher than the build budget suggests.

---

## Option 4 — Capability Flight Plan: compile intent into least authority before an agent flies

### One-line promise

Dry-run an agent task in a disposable local world, turn observed and declared needs into a reviewable capability plan, then enforce that plan across real execution and surface deviations as small, intelligible permission diffs.

### User and pain

Today's permission prompts are atomized: one shell command, path, MCP tool, or network request at a time. Beginners either approve blindly or drown in prompts. Advanced users adopt broad bypass modes to keep agents moving. Neither sees the task's proposed blast radius as one coherent object before execution.

### System insight

The missing object is a **Capability Flight Plan**: a versioned, task-scoped manifest of allowed reads, writes, executables, network destinations, secret handles, MCP resources, budgets, and irreversible effects. The plan is produced from three sources kept visibly distinct:

1. human-declared scope;
2. static prediction from the repository/task; and
3. observed needs from a disposable preflight.

Real execution is a plan-conformance problem. A deviation is a proposed plan diff with causal context, not another generic “allow bash?” dialog.

### Architecture

1. Portable capability schema with deny-by-default semantics and explicit unknowns.
2. Preflight runner in a disposable container/VM where syscall, filesystem, process, network, and tool requests can be observed without granting access to real secrets or external write APIs.
3. Policy compiler targeting OS/container enforcement plus available agent hooks. Hook policy is advisory unless the containment boundary proves completeness.
4. Secret broker exposing opaque, scoped handles rather than raw environment variables where integrations support it.
5. Effect classifier for idempotent, reversible, expensive, and irreversible operations; classifications remain human-reviewable claims.
6. Approval UI showing capability groups, reason chains, data sensitivity, and diffs from known-good plans.
7. Runtime auditor that fails closed at the strongest available boundary and labels weaker “observed only” platforms honestly.

### Why now

[GitHub's current hooks](https://docs.github.com/en/copilot/reference/hooks-reference) can allow, deny, or ask around tool use, but their own fail behavior varies by command, HTTP mode, error, and timeout. The [MCP authorization specification](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization) hardens token audience and transport, but it does not express one task's minimal behavioral authority. Agent power is increasing faster than a human's ability to reason about a stream of prompts.

### Existing substitutes and the non-duplication case

- Agent permission systems match tool/command rules but do not infer and present a whole-task authority budget.
- Devcontainers and hosted sandboxes isolate environments but do not produce a portable semantic plan or human-scale deviation review.
- Enterprise policy systems enforce organizational baselines; Flight Plan specializes those constraints for one intent.
- MCP authorization answers who may access a server, not which resources and effects this change should need.

| Wake overlap | Material difference | Safe relationship |
|---|---|---|
| Wake has effect tiers/classes, human gates, and containment-oriented execution. | Flight Plan's core is **pre-execution least-authority inference, OS-level enforcement, and human-readable capability diffs**. It must not reproduce Wake's log, supervisor, replay, or effect ledger. Wake currently does not claim a cross-platform security sandbox. | Wake events can be one evidence source and Wake can run inside the enforced environment. Never claim Wake alone is the security boundary. |

### Offline/local proof path

On one supported platform, preflight a local task against a demo repo with fake-but-clearly-local resources (files, localhost service, test secret handle); generate a plan; approve it; rerun under enforcement; block an undeclared path and domain; show a minimal plan diff; and prove that a crashing/failed hook cannot bypass the OS boundary. No real credential or external API is involved.

### Heavy-but-coherent MVP

- Linux-first container/namespace enforcement; macOS support explicitly observational unless a reviewed boundary is available.
- Capability schema and policy compiler.
- File/process/network observer and enforcer.
- Local secret-handle broker.
- GitHub/Claude/Codex hook exporters labeled by coverage.
- CLI and visual blast-radius planner.
- Adversarial bypass suite and redaction tests.

### Key failure mode

This is a security product. A friendly graph cannot compensate for an incomplete boundary. Cross-platform subprocess behavior, shell indirection, DNS, inherited file descriptors, local sockets, and secret exfiltration create a long tail. If enforcement is hook-level or fail-open, the product must call itself a planner/auditor rather than a sandbox. Production use requires independent security review.

**Architecture confidence: 5/10.** Real value, but correctness and platform scope are load-bearing enough to overwhelm a broad first build.

---

## Option 5 — Outcome Circuit: compile human intent into explicit proof obligations

### One-line promise

Turn “make this work like this” into a small, editable circuit of observable outcomes, forbidden regressions, confidence gaps, and executable probes that stays attached to the code after the agent leaves.

### User and pain

The beachhead user can state desired behavior but does not know what test suite is sufficient. An agent can generate both implementation and flattering tests, producing circular confidence. Existing specs are verbose documents or task checklists that drift from the shipped behavior.

### System insight

An **Outcome Circuit** is a typed graph, not prose:

- stimuli and fixtures;
- visible and machine-observable outcomes;
- invariants and forbidden outcomes;
- environmental assumptions;
- probe implementations;
- evidence receipts; and
- uncovered/ambiguous clauses.

Natural language helps draft the circuit, but the graph's truth status comes from human approval and independent execution. Every claim is one of `observed`, `inferred`, `unexercised`, or `contradicted`. The product is valuable precisely because it shows what remains unproved.

### Architecture

1. Outcome DSL/schema with browser, HTTP, CLI, data-state, performance-envelope, and manual observation nodes.
2. Intent parser that proposes a circuit but never silently accepts it.
3. Ambiguity linter that finds subjective adjectives, missing fixtures, hidden actors, and unverifiable claims.
4. Probe compiler to Playwright, command checks, API assertions, and custom adapters.
5. Independence modes: implementation agent, verifier agent, and deterministic checks are labeled separately; shared model lineage is visible.
6. Coverage graph mapping each outcome clause to probes and current receipts.
7. Change-impact watch that marks contracts stale when linked symbols/routes/schema change.
8. Human UI optimized for accepting, splitting, rejecting, or weakening a claim with consequences shown.

### Why now

The verification burden is now empirically visible, but the adjacent field is active. [VibeContract](https://arxiv.org/abs/2603.15691) already links decomposed task contracts, code, and runtime verification. Outcome Circuit is only defensible if it wins on non-expert legibility, multimodal behavior probes, explicit epistemic states, and durable post-build maintenance—not on “AI writes a spec.”

### Existing substitutes and the non-duplication case

- Spec-driven-development tools generate requirements and tasks; this system's atomic unit is an observable claim with a proof state.
- BDD/Cucumber expresses behavior but assumes an author who knows how to formalize it and rarely visualizes ambiguity or evidence independence.
- Test management tools track cases but do not compile human intent into a living causal contract for coding agents.
- VibeContract is direct adjacent prior art. The deep dive should kill this option unless a source/code read confirms a meaningful gap.

| Wake overlap | Material difference | Safe relationship |
|---|---|---|
| Wake records receipts/effects during execution. | Outcome Circuit defines **what human outcome must be established and what remains unknown**. It does not supervise agents, persist a runtime event log, resume, replay agent effects, or hash-chain evidence. | Wake or didrun receipts can satisfy circuit nodes through adapters; the circuit must preserve their original grade and scope. |

### Offline/local proof path

Record a user journey in a demo app; draft an outcome circuit; deliberately leave one ambiguous clause; approve the rest; compile browser/API/CLI probes; run them before and after a seeded regression; and show the exact clause transitions without any model API. A human-authored JSON fixture proves the engine; optional model drafting stays gated.

### Heavy-but-coherent MVP

- Typed outcome schema and validator.
- Visual circuit editor plus concise YAML/JSON form.
- Browser, HTTP, and CLI probe compiler.
- Receipt import with grade preservation.
- Ambiguity/staleness linter.
- Coverage and epistemic-state UI.
- Git integration and agent-facing MCP queries.

### Key failure mode

The oracle problem does not disappear: a plausible contract can encode the wrong behavior, generated probes can share the implementation's blind spot, and non-experts may approve polished nonsense. If the system implies correctness rather than making uncertainty legible, it is actively harmful. It may also be a feature of Threadline rather than a standalone platform.

**Architecture confidence: 6/10 as a subsystem; 4/10 as the whole product.**

---

## Option 6 — Wake Studio: the human application layer Wake deliberately has not finished

### One-line promise

Turn Wake's rigorous fleet log into a calm, comprehensible mission surface where a serious vibe coder can understand what changed, why it changed, what was actually verified, where execution is in doubt, and what the next safe action is.

### User and pain

Wake's correctness vocabulary is sophisticated: epochs, cursors, effect tiers/classes, fork lineage, receipts, in-doubt dispatch, state hashes, and anomaly detection. That vocabulary is appropriate for the substrate and too expensive for the target user. Wake's own README says its surfaces are a first draft, its human-approval flow is not converged, and broad live validation is light.

### System insight

Do not hide the rigor; **project it into task-shaped human concepts**:

- Mission and intended outcome
- Workers and responsibilities
- Artifacts and causal origins
- Checks and their exact receipt grades
- Decisions waiting on the human
- Effects that are safe, risky, or in doubt
- Timeline branches and what would be inherited or re-fired
- Handoff readiness and unresolved uncertainty

The Studio is explicitly an application over Wake, not an alternate runtime. Its primary object is a **Mission Projection**, a read-only/review projection built from Wake's canonical log plus user-authored intent metadata.

### Architecture

1. Version-pinned Wake protocol/CLI adapter; prefer an upstream export/API over reading private SQLite tables.
2. Pure projection layer from Wake events into human mission concepts, preserving links back to raw events and exact receipt wording.
3. Intent sidecar that stores goal, acceptance criteria, artifact labels, and decision rationale without entering Wake's correctness claims.
4. Local daemon serving a responsive dashboard and monochrome CLI summaries.
5. Artifact causality explorer, effect/gate inbox, crash/resume explainer, fork preview, and handoff bundle generator.
6. Adapter health surface that distinguishes protocol fixture, real binary/credential-free, and live-model validation exactly as Wake does.
7. Optional behavior evidence import from Threadline/Playwright/didrun without upgrading its grade.

### Why now

The infrastructure is already here in the user's repository, while Wake states honestly that design taste and the human-approval experience remain open. A focused application could provide immediate leverage and serve as a brutal real-user test of Wake's protocol semantics.

### Existing substitutes and the non-duplication case

- Wake's current viewer is a read-only first draft; Studio is a full task/decision/artifact UX.
- General trace explorers expose event detail but do not translate Wake's exact correctness boundaries for a vibe coder.
- Agent command centers show threads and diffs but do not inherit Wake's durable substrate or effect semantics.

| Wake overlap | Material difference | Safe relationship |
|---|---|---|
| Deliberately high at the product boundary: same runs, agents, events, effects, receipts, and branches. | Deliberately zero at the semantic kernel: **no new supervisor, log, reducer, effect ledger, crash recovery, fork implementation, or competing protocol**. Studio consumes Wake as the only execution source of truth and adds intent metadata + human projection. | This is the one permitted “same space” option because it is explicitly a user-friendly application layer. If implementation starts recreating Wake internals, stop. |

### Offline/local proof path

Build against Wake's bundled deterministic demo and locally produced run families. Render clean run, crash/resume, chain failure, pending gate, X-class halt, branch preview, and unreceipted adapter states. Every UI claim deep-links to the raw Wake event or exact CLI output. The proof needs no model access and must remain useful when the agent is a scripted worker.

### Heavy-but-coherent MVP

- Supported Wake version matrix and import contract.
- Mission projection schema and deterministic projection tests.
- CLI `mission`, `explain`, `handoff`, and `open` commands.
- Responsive mission dashboard with keyboard navigation.
- Gate/effect explanation UX; actual approval remains Wake-owned until an upstream-safe command exists.
- Artifact and receipt views, fork preview, and handoff bundle.
- Full empty/loading/error/active/crashed/resumed/completed/corrupt states.
- Usability tests against people who have not read Wake's manual.

### Key failure mode

This may be excellent product work without being a new systems idea. It also inherits a zero-adoption experimental dependency and can easily misstate Wake's nuanced guarantees. If Studio cannot make a new user complete crash/recovery, gate interpretation, artifact inspection, and handoff without reading the manual—while preserving exact caveats—it is decoration.

**Architecture confidence: 8/10 buildability, 5/10 standalone novelty.**

---

## Recommendation and forced choice

### Choose Threadline for the deep dive

Threadline best satisfies the stated ambition because it changes the unit of interaction from files/chat/agent sessions to **observable product behavior**. It is a real infrastructure project—trace propagation, source resolution, graph projection, redaction, scenario replay, semantic diff, MCP/CLI interfaces—with a human experience that is inseparable from correctness. It can be proven locally without pretending to integrate a model provider. It also composes cleanly with Wake and didrun instead of cloning either:

```text
human demonstrates behavior
        |
        v
Threadline records behavior thread + gaps
        |
        +----> agent context capsule (CLI/MCP)
        |
        v
agent executes in any tool (optionally Wake-supervised)
        |
        v
Threadline replays behavior + compares evidence
        |
        v
didrun or other verifier grades build claims without grade inflation
```

The architecture deep dive should attempt to kill five load-bearing claims:

1. **Cross-layer correlation:** Can a browser action be joined to backend/data activity without invasive application rewrites?
2. **Source precision:** Does runtime + static resolution produce an agent context slice materially smaller and more relevant than ordinary repository search?
3. **Behavioral fidelity:** Can replay distinguish intended change from regression without pretending a screenshot is correctness?
4. **Privacy:** Can useful evidence survive default redaction of headers, bodies, secrets, and user data?
5. **Non-duplication:** Does any existing observability/code-intelligence product already perform this behavior-to-code-to-agent-capsule loop?

### Keep BranchLab as the fallback

If source attribution or trace continuity fails, BranchLab is the most coherent fallback. It can incorporate Threadline-style observations later, but its v1 must be framed as a counterfactual **ambiguity-discovery and intent-compilation system**, not a multi-agent launcher, assurance dashboard, didrun replacement, or Wake derivative.

### Treat Wake Studio as a deliberate strategic fork

If the founder decides the goal is to maximize immediate utility for the existing Wake asset rather than create a separate novel kernel, Wake Studio is valid. Make that choice explicitly. Do not accidentally drift there while claiming a new runtime.

## Global kill lines

Park any option that crosses one of these lines:

- It needs to reimplement Wake's supervisor, event log, crash-resume, replay, fork, effect ledger, or hash-chain spine to feel complete.
- Its “agent integration” is a simulated external API rather than a real local interface or a clearly labeled fixture.
- Its primary demo is a graph with no user action that becomes newly possible.
- It converts model narration into a stronger truth grade than the underlying deterministic evidence.
- It cannot show missing instrumentation, ambiguity, redaction, or unsupported behavior as first-class states.
- Its useful proof reduces to Git + Playwright Trace Viewer + Jaeger + a README after the novelty is removed.

## Sources used for architectural claims

- [Wake repository and current README](https://github.com/nelsonwerd/wake)
- [Wake prior-art analysis](https://github.com/nelsonwerd/wake/blob/main/docs/PRIOR_ART.md)
- [OpenAI: Introducing the Codex app](https://openai.com/index/introducing-the-codex-app/)
- [GitHub Copilot hooks reference](https://docs.github.com/en/copilot/reference/hooks-reference)
- [OpenTelemetry: GenAI observability](https://opentelemetry.io/blog/2026/genai-observability/)
- [OpenTelemetry HTTP semantic conventions](https://opentelemetry.io/docs/specs/semconv/http/)
- [Playwright Trace Viewer](https://playwright.dev/docs/trace-viewer)
- [Playwright agent CLI tracing](https://playwright.dev/agent-cli/commands/tracing)
- [Microsoft Research: RPG-Encoder](https://www.microsoft.com/en-us/research/publication/closing-the-loop-universal-repository-representation-with-rpg-encoder-2/?lang=en)
- [CODESTRUCT: Code Agents over Structured Action Spaces](https://aclanthology.org/2026.acl-long.607/)
- [VibeContract](https://arxiv.org/abs/2603.15691)
- [Model Context Protocol authorization specification](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization)
