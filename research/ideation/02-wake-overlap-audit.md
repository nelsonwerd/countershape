# Wake overlap audit: the runtime already exists

- **Repository:** [`nelsonwerd/wake`](https://github.com/nelsonwerd/wake)
- **Immutable snapshot audited:** [`fff9bfa44a011b51c6fb7f188343a0352af1da79`](https://github.com/nelsonwerd/wake/commit/fff9bfa44a011b51c6fb7f188343a0352af1da79), tagged [`v0.1.0`](https://github.com/nelsonwerd/wake/tree/v0.1.0), 2026-07-14
- **Audit date:** 2026-07-14
- **Scope inspected:** README and complete reference docs; repository/package structure; core event, store, reducer, ledger, supervisor, protocol, adapters, CLI, viewer, demo, tests, CI, tags, commit history, public issues/PRs/releases
- **Method note:** source and public CI were inspected; Wake's test suite and live-model protocols were **not** rerun for this audit. The latest public CI run at the audited head reported success ([Actions run 29376309855](https://github.com/nelsonwerd/wake/actions/runs/29376309855)).
- **Confidence:** 9/10 on source-level overlap and boundaries; 6/10 on runtime maturity because broad live use is intentionally not claimed by Wake and was not independently exercised here.

## Executive verdict

**The current `project-aperture` placeholder overlaps Wake approximately 9/10 at the runtime layer. Building it as presently sketched would mostly rebuild the user's newly completed project.**

Wake is not a terminal recorder with a hash bolted on. Its actual primitive is: **a run is one append-only, hash-chained data structure that is simultaneously the execution substrate, coordination bus, recovery state, effect ledger, audit record, and replay/fork lineage.** It already supervises heterogeneous unmodified subprocess agents, ships real Claude Code and Codex adapter work, survives application crashes, resumes, reduces events deterministically, verifies chain integrity, forks timelines, exposes a stable CLI/JSON surface, and serves a read-only causal viewer. The implementation is a substantial correctness-oriented Go system, not a demo shell around logs.

The current concept brief's candidate spine—local event journal, adapter boundary, isolated execution, evidence/receipts, causal run graph, resume/handoff, CLI, and dashboard—names almost the same system. A different language, prettier interface, renamed event types, or added Git metadata would not create an honest new concept.

There is, however, a clean and valuable layer Wake does **not** own:

> **Compile fuzzy human change intent into an artifact-scoped contract; map obligations onto a repository; reconcile changed code and verification evidence against those obligations; and emit a model-portable change packet for review and continuation.**

That system's primary data structure would be an **intent / artifact / obligation graph**, not an agent event log. It would treat Wake, OpenTelemetry, Git, didrun, CI, and ordinary shell sessions as optional evidence providers. It would not supervise agents, recover their processes, classify their effects, or implement replay/fork. This is the strongest non-duplicative direction remaining in the current concept family.

An explicitly more user-friendly Wake application is also legitimate, but only if it is honest about being **Wake Studio**, reuses Wake as the engine, and creates substantial new user value above the runtime. Reimplementing the engine behind a friendlier UI is not enough.

## What Wake actually is

### The core primitive

Wake's manual states its load-bearing idea in six words: **"the run is a data structure."** The source of truth is a linear event sequence. Each event carries its predecessor hash, sequence, closed kind, source agent, causal parent, deterministic timestamp field, and payload digest. State is a pure reduction of that sequence; SQLite tables outside events/blobs are rebuildable indexes or caches ([concepts, lines 3-39](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/concepts.md#L3-L39)).

This is more specific than "agent observability":

- The **supervisor** is the sole writer, process manager, scheduler, and coordination bus.
- Agents are subprocesses speaking a seven-frame NDJSON protocol over stdin/stdout.
- The log is inside the execution path rather than a PostToolUse hook beside it.
- Nondeterministic or externally visible operations are modeled as effects with durable intent-before-I/O semantics.
- Recovery re-reduces the same log, fences prior process epochs, and resumes from recorded state.
- Replay constructs no executor; fork structurally shares the parent prefix.
- The CLI and viewer are read projections of the same source of truth.

The README's own sharp positioning is that the record is produced by the runtime the agents execute within, and that the log is at once execution substrate, crash-resume spine, coordination bus, and audit record ([README, lines 5-25](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/README.md#L5-L25)). That is the exact territory the provisional Aperture architecture was about to claim.

### The user job

Wake serves a **single local operator running a heterogeneous fleet**. The operator:

1. Builds or installs one Go binary and adapter binaries.
2. Declares agent `argv`, scheduling mode, spawn limits, environment allowlists, and effect classes in a fleet JSON file.
3. Runs the fleet with `wake run`.
4. Uses `status`, `log`, and `view` to understand the run.
5. Uses `gate answer` for an approval, `resume` after interruption/crash, `verify` for chain/state integrity, and `replay` / `fork` / `diff` for historical analysis.

The quickstart demonstrates this complete job ([README, lines 121-166](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/README.md#L121-L166)); the CLI exposes explicit machine-safe dispositions and stable JSON for those verbs ([CLI reference, lines 1-35](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/cli.md#L1-L35)).

Wake's present aha moment is therefore:

> Kill a live multi-agent fleet, resume it to an honest terminal state, verify and reconstruct the causal record, then inspect or branch the timeline without re-firing recorded effects.

The narrated demo is expressly organized around that experience ([demo README, lines 1-33](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/demo/README.md#L1-L33)).

### Architecture and core source abstractions

| Layer | What Wake implements | Evidence |
|---|---|---|
| Deterministic event codec | Closed event envelope, content/payload hashes, stable IDs, strict chain linkage, O(1) fork marker | [`internal/chain/chain.go`, lines 9-117](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/internal/chain/chain.go#L9-L117), [lines 154-226](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/internal/chain/chain.go#L154-L226) |
| Durable store | SQLite event rows plus blob CAS; runs and epochs are derived/rebuildable; WAL/full-sync write path; one run family per store | [`internal/store/schema.sql`, lines 1-52](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/internal/store/schema.sql#L1-L52) |
| Pure reducer | Closed run/agent/gate/effect/budget state; order-insensitive across agents but trajectory-sensitive within an agent | [`internal/reduce/state.go`, lines 35-78](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/internal/reduce/state.go#L35-L78), [lines 126-144](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/internal/reduce/state.go#L126-L144) |
| Effect ledger | `brokered` / `gated` / `observed` tiers and `P` / `E` / `R` / `X` retry-risk classes; intent-before-I/O, in-doubt dispatch, witness accounting | [`docs/manual/effects.md`, lines 11-80](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/effects.md#L11-L80), [lines 82-105](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/effects.md#L82-L105) |
| Supervisor | Sole writer; lease and epoch fencing; spawn/reap; free or deterministic fleet scheduling; effect execution; gates; event/spawn budgets | [`internal/supervisor/supervisor.go`, lines 1-31](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/internal/supervisor/supervisor.go#L1-L31), [lines 135-230](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/internal/supervisor/supervisor.go#L135-L230) |
| Adapter protocol | Five-frame mandatory core plus effect request/control capabilities; at-least-once ordered delivery; acknowledgements and idempotent emission | [`docs/PROTOCOL.md`, lines 1-22](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/PROTOCOL.md#L1-L22), [lines 233-258](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/PROTOCOL.md#L233-L258) |
| Real harness bridges | Claude stream-json / permission callback bridge and Codex app-server / JSON-RPC approval bridge, with fake-protocol and real-binary tiers kept distinct from human live-model validation | [`docs/ADAPTER_CLAUDE.md`, lines 1-29](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/ADAPTER_CLAUDE.md#L1-L29), [`docs/ADAPTER_CODEX.md`, lines 1-24](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/ADAPTER_CODEX.md#L1-L24) |
| Time travel | Executor-free reduction, O(1) parent-pointer fork, event/state/effect diff | [`docs/manual/time-travel.md`, lines 1-37](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/time-travel.md#L1-L37), [lines 66-77](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/time-travel.md#L66-L77) |
| Human surfaces | Monochrome-aware CLI, stable JSON, status diagnosis, event log, chain verifier, lineage operations, embedded read-only timeline/viewer | [`docs/manual/cli.md`, lines 80-164](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/cli.md#L80-L164), [`internal/view/model.go`, lines 91-162](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/internal/view/model.go#L91-L162) |

This is a compact codebase but not a small idea: the audited snapshot contains 199 Go files and 106 Go test files, plus a demo/selftest and crash-injection sweeps. The commit history was built in ordered architectural slices—codec/chain, store, reducer/ledger, crash census, protocol, supervisor, fleet, adapters, time travel, CLI, viewer, hardening—rather than as one generated drop ([commit history](https://github.com/nelsonwerd/wake/commits/main/)).

## Where Wake is genuinely strong

### 1. It has an actual semantic kernel

Wake's event log is not merely storage for arbitrary trace spans. A small closed event vocabulary, a reducer, an epoch-fenced append chokepoint, effect lifecycle rules, agent delivery cursors, and fork lineage all agree on one model. The design is therefore falsifiable in code. That makes it considerably harder to differentiate from than an observability dashboard.

### 2. Correctness claims have unusually careful boundaries

Wake's normative guarantee document separates:

- executor-free replay determinism;
- application-crash resume with externally witnessed accounting for **brokered** effects;
- state-hash equivalence only for deterministic test fleets and deterministic executors.

It expressly denies power-loss coverage and real-model hash equivalence ([guarantees, lines 13-31](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/guarantees.md#L13-L31)). It also distinguishes corruption evidence from tamper evidence and explains that external anchoring is deferred ([security, lines 7-25](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/security.md#L7-L25)). This honesty is part of the design, not just disclaimer text.

### 3. It uses containment as a systems wedge

The adapters are subprocesses under a supervisor, not SDK functions inserted into one agent framework. This gives Wake a plausible way to coordinate Claude, Codex, scripted workers, and arbitrary binaries under one transport without making each agent's application state into a LangGraph or Temporal workflow.

The claim remains tier-scoped: real harnesses can auto-execute some operations and observed actions are not upgraded into brokered effects. The Claude adapter explicitly describes where permission interception exists and where it does not ([Claude adapter, lines 65-97](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/ADAPTER_CLAUDE.md#L65-L97)). That nuance makes the architecture more credible, not less overlapping.

### 4. Crash behavior is a first-class engineering subject

Leases, process-group reaping, PID reuse guards, epoch fencing, idempotency keys, crash-site census, in-doubt effect dispatch, and an external witness are all explicit. The demo exercises fixed and random kill points, gate survival, and Class-X halt behavior ([demo selftest, lines 79-145](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/demo/selftest.sh#L79-L145)). A new runtime would need comparable rigor merely to reach parity.

### 5. The read surface preserves forensic constraints

The viewer opens the store read-only, binds localhost, validates host/origin, applies a strict CSP, renders payloads as text, embeds its assets, and has no write endpoint ([security, lines 63-87](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/security.md#L63-L87)). The UI may be first-draft aesthetically, but it is not a casual admin panel.

## Wake's honest boundaries

These are boundaries to respect, not invitations to build a near-identical competitor by checking the missing boxes.

### Product and validation boundaries

- `v0.1.0` is a first tag, not a mature distribution. At the audit snapshot the public repository reported no users/traction, stars, forks, issues, or PR activity; there was a tag but no GitHub Release artifact. Wake itself says demand is unvalidated and the surfaces are first-draft ([README, lines 251-263](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/README.md#L251-L263)).
- One real Claude Opus fleet reportedly survived a double crash, but that run used bypass permissions and exercised zero gate events. Broad live validation is light. The Codex live-model leg is still human-gated ([README, lines 197-231](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/README.md#L197-L231)).
- Runtime validation is macOS/arm64. Linux is cross-build/vet only; Windows is out of scope ([README, lines 232-241](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/README.md#L232-L241)).

### Security and operating boundaries

- It is single-operator and local-first, with no authentication, authorization, or multi-tenancy.
- Run logs may durably contain raw prompts, effect arguments, results, and secrets. There is no redaction/export path or at-rest encryption ([security, lines 27-61](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/security.md#L27-L61)).
- Its chain proves internal consistency, not an adversary-resistant history. External anchoring is deferred.
- Multi-machine execution and remote collaboration are deferred.

### Functional boundaries most relevant to the new concept

1. **No repository-semantic model.** Wake events know run/agent/message/effect/gate lifecycle. `artifact` is an agent emit label that maps into the generic `msg.emitted` chain kind; Wake does not define path identity, symbol/module dependency, AST semantics, ownership, or a relationship from a requirement to a changed file ([protocol, lines 109-131](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/PROTOCOL.md#L109-L131)).
2. **No intent compiler or executable change contract.** Fleet JSON declares processes, scheduling, budgets, and effect policy; it does not compile a fuzzy product outcome into invariants, proof obligations, acceptance criteria, or a scoped change program ([fleet spec, lines 1-42](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/fleet-spec.md#L1-L42)).
3. **No software-quality verdict.** `wake verify` verifies the event chain and reduced state hash. It does not claim tests passed, behavior meets intent, code is secure, or a patch is reviewable ([CLI, lines 94-106](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/cli.md#L94-L106)).
4. **No proof-carrying change artifact.** The README explicitly says no receipt artifact ships for Wake's own claims; its evidence is rerunnable test/sweep commands ([README, lines 168-189](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/README.md#L168-L189)). It has effect-result receipts inside a run, not a portable, artifact-scoped review packet.
5. **No arbitrary cross-tool semantic continuation.** Resume recovers the original fleet spec and process adapter. Claude can use a stored session ID; Codex thread resume is not wired. Wake does not compile completed decisions, current artifact state, open obligations, and minimal context into a tool-independent handoff for a different agent to continue. The fleet spec is deliberately copied so resume recovers the exact `argv` ([fleet spec, lines 1-9](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/fleet-spec.md#L1-L9)); Codex thread resume remains unimplemented ([Codex adapter, lines 251-267](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/ADAPTER_CODEX.md#L251-L267)).
6. **No general task/workflow graph by design.** Wake explicitly tells graph-shaped systems to use LangGraph and platform-backed durable work to use Temporal ([README, lines 79-95](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/README.md#L79-L95)).
7. **Fork execution is not wired through the CLI.** Branches can be created, reduced, inspected, and diffed, but `run` / `resume` cannot execute a fork branch; recorded `live` / `stub` policy is not read ([time travel, lines 39-64](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/time-travel.md#L39-L64), [lines 79-94](https://github.com/nelsonwerd/wake/blob/fff9bfa44a011b51c6fb7f188343a0352af1da79/docs/manual/time-travel.md#L79-L94)). This is an obvious Wake roadmap item, not a clean independent wedge.

## Exact overlap with the provisional concept

| Provisional Aperture element | Wake coverage | Overlap verdict |
|---|---|---|
| Local event journal | Append-only SQLite log + blob CAS is the sole truth | **Exact duplicate** |
| Vendor-neutral adapter boundary | Five-frame NDJSON subprocess protocol; Claude and Codex adapters | **Exact duplicate** |
| Isolated/durable execution | Supervisor owns subprocesses, leases, epochs, crash recovery, workdir | **Core duplicate**; Wake's isolation guarantees are narrower than a full OS sandbox, but the product category is the same |
| Multi-agent orchestration | Fleet specs, free/deterministic schedulers, spawning and budgets | **Exact duplicate** |
| Effect capture / approval | Effect tiers/classes, gates, witness, retry/halt semantics | **Exact duplicate** |
| Resumability | `kill -9` recovery and adapter/session recovery | **Exact at process-run level** |
| Replay/fork | Executor-free replay, parent-pointer fork, diff | **Exact duplicate** |
| Corruption-evident evidence | Hash chain and `verify` | **Exact duplicate** |
| Causal run graph | Causal parents, per-agent lanes, event timeline, inspector | **Exact at run-event level** |
| CLI | Run/status/log/verify/replay/fork/diff/gate/view + JSON | **Exact duplicate** |
| Mission-control dashboard | Embedded read-only lineage/timeline/effect/gate viewer | **Strong duplicate** |
| Intent/task contract | Fleet spec is operational, not a change contract | **Open** |
| Artifact-level causal graph | Generic artifact emit only; no file/symbol/requirement semantics | **Open** |
| Verification claims tied to intent | Chain/state integrity only | **Open** |
| Different-agent semantic handoff | Same fleet/adapter recovery; no portable change packet | **Partially open** |

The current success metric contains both categories. "Interrupt and resume a multi-agent change" is Wake. "Resume through a different agent adapter" and "select a changed artifact and explain which intent, evidence, and unresolved obligations produced it" are not Wake—but they must become the **center**, not add-ons around a cloned runtime.

## Red lines: what this run must not build

Unless the project is explicitly branded and architected as an application of Wake, do **not** implement any of the following as a new semantic kernel:

1. Another append-only per-agent event journal presented as the source of truth.
2. Another local daemon/supervisor that owns Claude, Codex, or arbitrary agent subprocesses.
3. Another generic stdin/stdout adapter protocol for heterogeneous agents.
4. Another crash lease / epoch fence / process reaper / run-resume system.
5. Another `P/E/R/X`-like effect safety taxonomy, approval gate ledger, or external-effect witness.
6. Another hash-chain verifier for agent activity.
7. Another executor-free run replay or event-granularity timeline fork.
8. Another fleet JSON format for agent argv, budgets, spawning, or scheduling.
9. Another hero UI whose main object is a vertical timeline of run events, agent lanes, effect rows, and chain status.
10. Another command family substantially equivalent to `run`, `resume`, `status`, `log`, `verify`, `replay`, `fork`, `diff`, `gate`, `view`.
11. A TypeScript/Rust rewrite, hosted version, multi-user version, or prettier skin offered as a separate novel system. Those are adaptations/extensions of Wake's category.
12. Fork-branch execution as the new product wedge. It is an acknowledged missing Wake wire-up and belongs upstream first.

Renaming `Event` to `Observation`, `Effect` to `Action`, `Run` to `Session`, or `fork` to `branch` does not cross the boundary.

## Legitimate adjacent space

### Recommended: a software change compiler above runtimes

The cleanest adjacent concept is a **proof-carrying change compiler for AI-built software**.

Its source language is a human's fuzzy outcome plus repository policy. Its intermediate representation is not a run log but a typed graph:

- **Intent:** desired user-visible outcome and prohibited regressions.
- **Invariant:** behavior that must remain true.
- **Artifact:** file, symbol, schema, route, component, configuration, or document.
- **Obligation:** a checkable claim connecting intent/invariant to artifacts.
- **Evidence requirement:** what class of observation could support the obligation.
- **Decision:** a durable choice with rationale and supersession rules.
- **Risk boundary:** security/data/compatibility/operational blast radius.
- **Handoff slice:** minimal verified context and remaining obligations for the next actor.

The "compiler" would:

1. Index the repository and construct a semantic artifact/dependency graph.
2. Compile an outcome into explicit obligations and unknowns, with human approval for ambiguous/high-risk branches.
3. Create narrow context slices for any agent without owning that agent's process.
4. Ingest Git changes plus evidence from ordinary commands, didrun, CI, Wake, or OpenTelemetry.
5. Attribute changed artifacts and evidence to obligations; never infer more certainty than the evidence supports.
6. Detect uncovered changes, orphan obligations, contradicted invariants, and stale decisions.
7. Emit a portable **change packet**: intent, decisions, relevant artifacts, evidence grades, unresolved risks, and the next safe action.
8. Render an artifact/obligation map whose hero question is "what remains unproven?", not "what event happened next?"

Wake could be an excellent optional provenance adapter: its event family may enrich attribution and process history. The compiler must remain useful with a plain Git diff and verification receipts. Replacing Wake with another execution source should reduce provenance resolution, not break the product.

This preserves the original ambitious idea—portable, trustworthy agent work—while moving the novelty to a layer Wake intentionally does not model.

### Other adjacent ideas that remain distinct if kept narrow

1. **Review-compression engine for AI changes.** Turn a large generated diff into obligation-centered review slices, risk-ranked reading order, hidden coupling warnings, and explicit evidence gaps. Do not supervise the generator.
2. **Model-portable semantic handoff compiler.** Build the smallest continuation packet that lets a different tool resume work from repository facts and decisions rather than a transcript. Do not call process recovery "handoff" and do not recreate event replay.
3. **Executable intent IDE.** An interactive environment where non-experts see an outcome decomposed into invariants, trade-offs, and tests before an agent writes code, then watch the contract reconcile against the evolving patch. Its main visual is an obligation map, not an agent timeline.
4. **Evidence-policy layer.** Define which evidence is acceptable for different classes of claims, ingest receipts from didrun/CI/security tools, and prevent a model from upgrading an observation into a correctness claim. This must be claim-semantic policy, not another command recorder.
5. **Wake adapter/application.** A serious product layer that makes Wake usable by less systems-oriented builders. This is valid only under the adaptation bar below.

### Spaces that look adjacent but are too close

- OS-level containment and syscall capture for agent effects: technically deeper, but still squarely extends Wake's containment/effect-record thesis.
- Hosted team Wake, remote fleets, RBAC, multi-tenancy, and chain anchoring: legitimate future products, but direct extensions of Wake.
- Executable Wake branches and visual timeline comparison: direct Wake roadmap.
- A generalized durable agent workflow engine: Wake intentionally declines this, but Temporal/LangGraph/DBOS already own it; it is not a clean novel answer.

## If the project is a user-friendly Wake adaptation

A derivative route is legitimate because the user expressly allowed it, but it needs an unusually high honesty and value bar.

### Required architectural posture

- Name Wake as the engine in the one-line promise and architecture diagram.
- Execute the audited Wake binary or use an upstreamed Wake library/API. Do not clone the log, reducer, supervisor, chain, effect taxonomy, or time-travel implementation.
- Keep Wake run families as runtime truth; application data may add user-facing metadata without claiming to replace or strengthen Wake's guarantees.
- Contribute necessary correctness changes upstream when possible.
- Attribute Apache-2.0 code and preserve Wake's vocabulary for guarantees and caveats.
- Call the product an application such as **Wake Studio** rather than presenting it as a newly invented durable-agent runtime.

### Required new user value

A justified adaptation should deliver most of the following:

1. Zero-fleet-JSON onboarding: detect installed agents, validate versions, choose a safe throwaway workdir, and generate an inspectable spec.
2. A plain-language trust model: "safe to resume," "decision required," "cannot know whether this applied," and "history internally consistent" without overstating tamper or correctness guarantees.
3. Guided approval and Class-X resolution flows, including exact content and consequences.
4. Repository-aware artifact views and diffs above Wake's generic events.
5. Secrets preflight and loud sharing warnings; never pretend the immutable log can be redacted.
6. A cross-agent handoff experience that packages decisions and repository state, not just Wake events.
7. Accessible responsive UI, empty/loading/error/parked/halted/completed states, keyboard workflows, and excellent CLI parity.
8. Real human-gated validation against Claude and Codex where external accounts are required; unvalidated paths stay labeled.
9. A five-minute first run and a fifteen-minute substantive run for the target vibe coder.

A restyled event viewer or wizard that emits the same JSON but leaves the operator to understand epochs, effect tiers, gates, and run families is not enough.

## Non-duplication acceptance test

Apply this test before locking the new brief and again before each build unit.

### Automatic fail: any two mean the concept is still Wake-shaped

- [ ] The primary persistent object is an agent run/event log.
- [ ] The core process owns, schedules, resumes, or reaps agent subprocesses.
- [ ] The central guarantee is crash durability, deterministic reduction, replay, fork, or chain integrity.
- [ ] The system defines a generic agent transport or fleet specification.
- [ ] The system classifies and brokers external effects or human tool approvals.
- [ ] The primary CLI verbs resemble `run/resume/replay/fork/verify`.
- [ ] The dashboard hero is an agent/event timeline with run status and effect ledger.
- [ ] The smallest proof requires killing an agent process and recovering it.

### Must pass: all required for the independent concept

- [ ] The one-line promise makes sense without the words durable, replay, fork, chain, fleet, supervisor, or event log.
- [ ] The semantic kernel is an intent/artifact/obligation graph grounded in repository facts.
- [ ] The smallest proof works with a plain Git repository, one ordinary agent or human edit, and real verification receipts; Wake is optional.
- [ ] The primary output is a portable, artifact-scoped change/review/handoff packet.
- [ ] The main aha is seeing what changed, why it satisfies or violates intent, what evidence supports each claim, and what remains unproven.
- [ ] Cross-tool continuation reconstructs work from checked artifacts, decisions, and obligations—not from process/session replay.
- [ ] Wake and OpenTelemetry can be ingestion adapters without changing the core state machine.
- [ ] No code reimplements Wake's chain, supervisor, protocol, effect ledger, witness, crash recovery, or time-travel.
- [ ] Each flagship feature maps to a specific absent Wake capability documented above.

### Adaptation exception

If the project deliberately fails the independent test because it is a Wake application, it must instead pass all of these:

- [ ] The product explicitly says it is powered by Wake.
- [ ] Wake remains the runtime and source of run truth.
- [ ] No core Wake mechanism is independently reimplemented.
- [ ] The new value is predominantly onboarding, repository semantics, approvals, review, handoff, and human-centered UX.
- [ ] Wake's caveats appear intact wherever evidence or guarantees are explained.

## Recommended concept correction

Replace the current candidate one-line promise:

> Make multi-agent software work portable, inspectable, resumable, and provable across tools.

with something that cannot be mistaken for Wake:

> **Compile a fuzzy software change into an artifact-scoped contract, keep every code change and evidence receipt reconciled against that contract, and emit a reviewable packet any human or coding agent can continue without the original transcript.**

Then make these decisions explicit in `docs/CONCEPT_BRIEF.md`:

- **LOCKED:** Wake owns durable heterogeneous agent execution; this project will not recreate it.
- **LOCKED:** Git owns repository content; didrun/CI/tool outputs own verification receipts; the new system owns only semantic reconciliation and evidence policy.
- **LOCKED:** cross-agent continuation means a semantic handoff into a fresh tool, not live process recovery.
- **LOCKED:** the primary UI is an obligation-to-artifact map and review queue, not a run timeline.
- **LOCKED:** Wake is optional input in the independent product; if it becomes required, reclassify the project as a Wake application.
- **KILL:** if the prototype's value disappears when agent event history is replaced by a Git diff plus verification receipts, or if the demo's aha is still kill/resume/replay, park it as a duplicate.

That correction preserves the user's ambition while respecting what Wake has already accomplished.
