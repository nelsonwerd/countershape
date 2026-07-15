# Finalist duel: choose the decision witness, not another trust dashboard

- **Judge:** skeptical systems/product review in the Soren Vale frame
- **Date:** 2026-07-14
- **Decision:** **Proceed with B, the counterfactual ambiguity lab, under a deliberately narrow semantic contract. Park A as a standalone product. Park C as a future probe-planning module.**
- **Confidence:** 8/10 that B is the cleanest non-duplicate; 6/10 that the full product can make its minimization and isolation claims reference-quality in this build; 4/10 on adoption, which this exercise cannot validate.

## Executive ruling

The attractive answer from the first ideation lane—an assurance graph and proof-carrying change—does not survive contact with the **actual installed didrun**, not merely its one-line pitch. didrun already records commands and exact Git tree state, declares structured claims, grades their binding as `TREE-EXACT`, `SCOPE-EXACT`, `STALE`, `UNKNOWN`, or `FAILED`, attaches a commit-bound manifest, exports a bundle, and renders a self-contained HTML reviewer artifact. Its source is unusually explicit that it records command success without inferring whether the command mattered. An “assurance graph” that adds claim-to-file edges, policies, and a prettier unresolved-risk view would be useful, but as a new heavyweight system it is uncomfortably close to **didrun plus semantic metadata**.

The repository flight simulator is ambitious and visually strong, but its promise is too broad to state honestly. Dependency queries, affected-test selection, static/data-flow analysis, runtime traces, browser session replay, and visual regression already cover most of its ingredients. The remaining claim—predicting a change's true behavioral blast radius before it exists—is precisely where false negatives become dangerous and framework-specific analysis explodes the scope.

The counterfactual ambiguity lab has the strongest distinct semantic center:

> **Given several candidate Git trees that claim to implement the same intent, execute identical probes, partition the candidates by observed behavior, minimize the inputs or action sequence that distinguishes each partition, ask the human only for the unresolved semantic decision, and compile that decision into a rerunnable contract.**

Its primitive is not a run, receipt, branch, task, diff, trace, screenshot, or quality score. It is a **decision witness**: the smallest reproducible observation that exposes a meaningful behavioral choice among plausible implementations.

This is differential testing as an intellectual ancestor, not a claim of a novel testing algorithm. The new systems proposition is the product lifecycle around it: arbitrary candidate patches in, oracle-free behavioral partitions, minimized witnesses, explicit human adjudication, executable decision contracts, and a new comparison run that shows which candidates satisfy the now-specified behavior. That remains valuable if every candidate came from a human and if Wake, didrun, Parallel Code, and Forge are absent.

## The expanded non-duplication gate

The Wake-focused automatic rejection tests in [`00-selection-rubric.md`](./00-selection-rubric.md) are necessary but no longer sufficient. The user has now revealed two adjacent systems, Wake and didrun, and the current market has mature branch managers, spec workflows, impact analyzers, and replay tools. A finalist needs a different answer in every row below.

| System | Source of truth | Primary job | Aha moment | What this project must not duplicate |
| --- | --- | --- | --- | --- |
| Wake | Hash-chained run event sequence | Execute, coordinate, recover, replay, and fork an agent fleet | Kill and safely recover a fleet, then inspect its causal history | Supervisor, agent protocol, event log, replay/fork runtime, effect broker, causal run viewer |
| didrun | Commit/tree-bound manifest over recorded command events and declared claims | Show what command actually ran against which Git tree and how fresh that evidence is | “The agent said tests passed; here is the exact recorded grade” | Command recorder, generic success claims, tree/scope/staleness grading, evidence bundle/report |
| Parallel Code / Automagik Forge | Tasks/attempts plus Git branches and worktrees | Run agents in isolated worktrees and compare/merge their output | See several attempts side by side and choose a winner | Agent launcher, task board, worktree manager, side-by-side diff selector, merge UI |
| Playwright-style visual regression | Committed golden snapshot plus current rendering | Detect pixels or serialized output that changed from a baseline | See the baseline/actual/diff triptych | Screenshot-baseline manager or generic snapshot approval tool |
| Differential testing | Divergent outputs from multiple implementations under common inputs | Find inconsistencies without a trusted oracle | One input makes implementations disagree | Merely reporting raw output differences |
| VibeContract / Spec Kit | Human-approved specification, task artifacts, and contracts | Turn intent into structured implementation and QA work | The agent follows an explicit spec/contract instead of a vague prompt | Spec generator, plan/task workflow, static checklist, generic traceability map |
| Candidate B | **Behavior matrix plus adjudicated decision contracts** | Discover what the intent failed to specify by comparing plausible implementations | A tiny, reproducible case reveals the decision the human did not know they needed to make | Any of the above substrate jobs |

“Forge” is overloaded in current tooling. This review uses **Automagik Forge**, the relevant open-source Kanban/worktree product whose docs explicitly describe multiple attempts, side-by-side comparison, and winner selection. The unrelated terminal MCP and enterprise-governance products with the same name do not change this verdict.

## Actual didrun inspection: why A is much closer than it first appears

I inspected the globally installed `didrun 0.1.0`, its CLI help, and the editable source checkout at commit [`6ca774a`](https://github.com/nelsonwerd/didrun/tree/6ca774a311ef74b1cfd1cd03e9e87023c04ba35e). The relevant behavior is not speculative:

- `didrun run -- <cmd>` captures argv, output, exit code, timing, and Git tree state.
- `didrun claim` declares a typed claim and binds it to a recorded event. The current claim vocabulary is deliberately small: `command-succeeded`, `tests-pass`, and `lint-clean`.
- The deterministic grader distinguishes exact tree binding, claimant-declared path scope, staleness with a concrete delta, absent evidence, and a witnessed failed command. See [`claims.py`](https://github.com/nelsonwerd/didrun/blob/6ca774a311ef74b1cfd1cd03e9e87023c04ba35e/src/didrun/claims.py).
- `didrun seal` attaches a manifest to the commit and can write an exportable bundle; `verify --strict` regrades it; `verify --html` emits the offline report. See [`manifest.py`](https://github.com/nelsonwerd/didrun/blob/6ca774a311ef74b1cfd1cd03e9e87023c04ba35e/src/didrun/manifest.py) and the [README trust boundary](https://github.com/nelsonwerd/didrun/blob/6ca774a311ef74b1cfd1cd03e9e87023c04ba35e/README.md).
- didrun explicitly kills a proposed `diff-exercised` claim because it would be coverage instrumentation without a deterministic core implementation. Path relevance is never inferred; the claimant supplies pathspecs.
- didrun's trust model is careful: it attests that a command ran and exited with a given result against a tree, not that the command was meaningful or the code correct.

That last boundary leaves genuine research space, but it does not automatically justify another product around the same nouns. The honest extension seam is a separate decision system that **references didrun receipts** without regrading them. Candidate B uses that seam cleanly. Candidate A mostly expands it.

## Finalist A — assurance graph / proof-carrying change

### Strongest form

The strongest version models:

```text
human promise -> artifact scope -> evidence requirement -> receipt -> support state
```

It distinguishes `SUPPORTED`, `CONTRADICTED`, `UNKNOWN`, `STALE`, and `WAIVED`; ingests Git, didrun, CI, browser, and human receipts; and emits a portable reviewer package. Its best UX is an unresolved-human-attention queue rather than a generic score.

### Collision ruling

| Neighbor | Collision | Judgment |
| --- | --- | --- |
| Wake | Low | Different source of truth and job; Wake could supply provenance. |
| didrun | **Severe** | Claims, evidence binding, Git-tree freshness, commit manifests, portable bundle, strict gate, and HTML review are already didrun's center. |
| Parallel Code / Forge | Low | A begins after candidate work exists. |
| Visual regression | Low to medium | A can ingest a snapshot receipt, but must not rebuild the runner. |
| Differential testing | Low | A does not generate disagreements; it classifies evidence. |
| VibeContract / Spec Kit | **High** | Intent decomposition, contracts, traceability, and QA are directly adjacent. VibeContract's stated paradigm already links tasks, contracts, code, testing, runtime verification, and debugging. |

It passes the Wake-specific rejection list, but fails the expanded product test: **replace most of the implementation with didrun plus a small declarative intent map and a custom HTML view, and most of the v1 value survives.** That is an integration or didrun roadmap, not the best independent heavy system for this run.

### Strongest falsifiable v1

On two substantively different real changes, define five to eight falsifiable claims, ingest actual Git and didrun evidence, and reveal one meaningful unsupported or contradicted claim that a green test command hides. Every status must reproduce offline from the package. A second reviewer must identify the top unresolved risk faster from the package than from the diff and raw didrun report.

**Kill it** if any of these are true:

1. A YAML file plus `didrun verify --html` preserves the same insight.
2. The graph needs an LLM to determine a positive support state.
3. A claim-to-artifact edge is presented as factual when it is merely author-declared or inferred.
4. The output is a global confidence/quality score.
5. The portable package duplicates didrun's bundle rather than referencing it.

### Hardest correctness risks

- **Semantic relevance is not deterministic.** A green test receipt can be real yet irrelevant to the human promise. Automating that edge either overclaims or reintroduces an LLM into the trust path.
- **Circular evidence.** An agent can write the implementation, the claim, and a narrow test that “supports” it. The graph must represent provenance/independence without pretending independence proves quality.
- **False aggregation.** Different claims require incomparable evidence; one roll-up grade hides the weakest obligation.
- **Policy authorship.** Evidence sufficiency is a human policy decision. Defaults can quietly become bogus certification.
- **Artifact drift and refactors.** Claim-to-symbol/file mappings decay even when behavior remains correct.
- **Secret-bearing receipts.** Importing raw tool output widens the leakage surface didrun already treats carefully.

### Verdict

**PARK as the semantic center.** Keep only a compact `decision contract` and receipt-reference format inside B. If the project later proves that users need a broader change assurance object, build it as an explicit didrun companion or contribution rather than pretending the overlap is accidental.

## Finalist B — counterfactual ambiguity lab

### Lock the one semantic center

The center is **minimal behavioral disagreement turned into a durable human decision**.

Inputs:

- one frozen base revision;
- one intent envelope containing the same outcome and hard constraints for every candidate;
- two or more already-existing candidate Git trees;
- a versioned local runner and a finite probe grammar;
- optional receipt references from didrun.

Core derivation:

```text
candidate trees
      |
identical, isolated probes
      |
normalized observations
      |
behavioral equivalence partitions
      |
minimized distinguishing witness
      |
human semantic decision
      |
executable decision contract
      |
re-run -> conforming / nonconforming / unstable / unobserved
```

The source of truth is a **behavior matrix** keyed by candidate tree, probe definition, runner/environment digest, normalizer version, and raw observation digest, plus **human-adjudicated decision contracts**. Git remains content truth. didrun remains receipt truth. No event history is needed to derive the decision state.

The product must use epistemically bounded copy:

- `NO OBSERVED DIFFERENCE`, never “equivalent”;
- `DIVERGENCE`, never “one branch is wrong”;
- `DECIDED FOR THIS WITNESS`, never “specification complete”;
- `UNSTABLE` when repeated runs disagree;
- `UNCOMPARABLE` for setup, timeout, schema, or infrastructure failures;
- `UNKNOWN` for unprobed behavior.

### Why it is not just Parallel Code or Forge

[Parallel Code](https://github.com/johannesjo/parallel-code) creates worktrees, spawns agents, shows diffs, and includes an “AI Arena” to race agents head to head. [Automagik Forge](https://docs.namastex.ai/forge/troubleshooting/faq) similarly creates task attempts in worktrees, compares side-by-side diffs, and asks the user to choose a winner.

B does none of their execution job. It accepts Git refs produced by those tools, Wake, CI, a human, or a fixture. It does not ask “which diff looks best?” It asks:

> **Which smallest observable case makes these implementations disagree, and what behavior do you actually want in that case?**

The human chooses a semantic outcome, not an agent or branch. The choice becomes a regression contract; candidates are then evaluated against it. A branch may win one decision and lose another, and “reject all” is first-class.

### Why it is more than differential testing

Differential testing is the closest prior art and must be credited, not hand-waved. It finds discrepancies by applying the same inputs to multiple implementations when no trusted oracle exists. [DiffSpec](https://arxiv.org/abs/2410.04249) goes further: it uses natural-language specifications and code artifacts to generate targeted differential tests, and reported differentiating tests and confirmed bugs in eBPF and Wasm implementations.

Therefore the project cannot claim “we invented running two versions and diffing outputs.” Its defensible new system is the **decision-witness lifecycle**:

1. candidate patches rather than independent mature implementations;
2. typed multi-modal observation adapters rather than one raw stdout comparator;
3. repeated execution and explicit unstable/uncomparable partitions;
4. deterministic witness minimization within a declared finite grammar;
5. a human adjudication surface that frames a product/security decision, not a test failure;
6. compilation of that decision into a repository-native executable contract;
7. a rerun that may accept one, several, or none of the candidates;
8. a portable record that says exactly what was and was not observed.

The algorithmic pieces are established. The eyebrow-raising contribution is making **underspecification discoverable from implementation diversity** and converting it into durable software knowledge.

### Why it is not visual regression

[Playwright visual comparisons](https://playwright.dev/docs/test-snapshots) compare a current screenshot or serialized value to a committed golden and explicitly warn that rendering varies across OS, version, settings, and hardware. Visual regression is one possible observation adapter, not the kernel.

B initially has no golden and no presumed winner. It compares N candidate behaviors under one probe, finds partitions, and asks the user to set the missing oracle. A later DOM/visual adapter should prefer accessibility trees and semantic state before pixels, retain raw artifacts, and label environmental noise. A screenshot approval UI must never become the hero surface.

### Why it is not VibeContract or Spec Kit

[VibeContract](https://arxiv.org/abs/2603.15691) starts from natural-language intent, decomposes it into tasks and task-level contracts, and maintains traceability into generated code, tests, runtime verification, and debugging. [Spec Kit](https://github.github.com/spec-kit/index.html) offers the established `Spec -> Plan -> Tasks -> Implement` workflow across many agents.

B runs the arrow in the opposite direction where those workflows are weakest:

```text
several plausible implementations
        -> observed disagreement
        -> missing human decision
        -> minimized witness
        -> executable contract
```

It can import a Spec Kit scenario or VibeContract requirement as probe context, but it must remain valuable when the original request was one sentence. It discovers **which part of the intent was underdetermined**; it does not generate the project plan.

### Why it is not Wake or didrun

- Wake can produce candidate branches or provenance, but B does not run, schedule, resume, replay, or fork an agent process. A plain list of Git refs is sufficient.
- didrun can receipt each probe invocation and bind it to a candidate tree. B should reference those exact grades verbatim, never copy or strengthen them. didrun cannot compare several candidates, construct behavioral partitions, minimize distinguishing input, or encode a human semantic choice.
- The behavior matrix is a derived comparison artifact, not an event log or alternate command receipt ledger.

### Strongest falsifiable v1

The proof should be narrow enough to be correct and rich enough not to be a toy.

#### Domain boundary

Support local Node/TypeScript programs through two real adapters:

1. **HTTP adapter:** launch each candidate with a declared command; issue method/path/header/body probes; normalize status, selected headers, canonical JSON, and declared state observations.
2. **CLI adapter:** run identical argv/stdin/env fixtures; normalize exit code, stdout/stderr through explicit user-reviewed rules, and declared filesystem outputs.

No external model API participates. Candidate branches are honest checked-in fixtures or user-supplied refs, never presented as live AI output. A future agent integration is human-gated.

#### Two substantive demonstrations

1. **Authorization semantics:** three green-test candidate branches implement a support-agent invoice-view feature. Identical probes expose at least two distinctions, such as cross-tenant access and whether a concealed resource returns `403` or `404`. The tool minimizes each witness to the smallest role/tenant/action tuple that preserves the partition.
2. **Configuration precedence:** three green-test candidates implement CLI/env/file configuration merging. Identical probes expose a distinction between absent, empty, and explicitly cleared values. The tool minimizes the argv/env/config input that preserves the partition.

For each demonstration the system must:

- ingest at least three candidate Git trees;
- prove the runner executed the exact same versioned probe set and declared environment policy for each candidate;
- repeat probes enough to detect a deliberately flaky fixture as `UNSTABLE` rather than a branch verdict;
- produce deterministic partitions from canonical observations;
- minimize at least one input/action sequence and state honestly that the result is **1-minimal within the declared grammar**, not globally minimal;
- let the human choose among observed outcomes or reject all;
- generate a readable decision file and an executable regression test without an LLM in the compilation step;
- rerun all candidates and show which satisfy that decision;
- reproduce the same matrix and contract digest offline from the committed fixtures;
- show one desktop and one mobile decision surface where the next action is unmistakable.

#### Go/kill line

**Proceed** only if both demonstrations complete the full loop and a reviewer can explain the missing semantic decision without reading raw logs or source diffs.

**Kill or radically narrow** if any of these occur:

1. The system only surfaces differences the fixture author already encoded as expected tests.
2. The “minimizer” merely truncates a JSON object without preserving the behavioral partition.
3. Candidate execution leaks state across runs or uses unequal setup.
4. A setup failure is shown as behavioral divergence.
5. The human ends up choosing “branch A” rather than a semantic outcome.
6. The generated contract cannot be run without the product.
7. The hero view is a side-by-side diff, run timeline, or generic green/red test table.
8. The demo cannot repeat on the second problem family.

### Hardest correctness risks

1. **Observational incompleteness.** Agreement on the probes says nothing about unobserved behavior. The UI and file format must never shorten `NO OBSERVED DIFFERENCE` to `SAME`.
2. **Environment parity.** Time, random seeds, locale, ports, process scheduling, filesystem state, dependency caches, and network can manufacture differences. Every comparison needs a versioned environment policy, unique scratch state, deterministic seeds where supported, and an explicit list of uncontrolled dimensions.
3. **Side-effect isolation.** Candidate code is arbitrary code. Shared databases, files, or ports can contaminate later candidates. V1 should be trusted-local-repository only, disable outbound network where practical, use per-run temp roots, enforce teardown, and report cleanup failure as `UNCOMPARABLE`.
4. **Flakiness.** One run cannot establish a partition. Repeat counts and stability thresholds must be explicit; inconsistent outcomes create their own unstable class and are never fed to minimization as stable facts.
5. **Normalizer unsoundness.** Removing timestamps or sorting arrays may erase real differences; retaining every byte may amplify noise. Normalizers must be versioned, narrow, user-inspectable, and preserve raw observations by digest.
6. **Minimization claims.** Delta debugging generally finds a 1-minimal witness relative to its reduction operators and test predicate, not the global smallest semantic case. Cache keys must include candidate trees, runner, normalizer, environment, and probe definition.
7. **Stateful action sequences.** Reducing a sequence can change setup, preconditions, or identity. A witness must include the smallest valid setup prefix, not only the final request.
8. **Correlated blind spots.** Three candidates can agree on the same bug, especially when generated from the same model or examples. Consensus is not correctness and must not be promoted to support.
9. **Human codification error.** A user can deliberately choose an insecure behavior. The system records the decision and consequence; it does not certify the decision. Security-sensitive decisions should require a visible rationale and remain reviewable.
10. **Probe-generation trust.** An LLM may later propose probes, but proposal is outside the truth path. Only deterministic execution and comparison create observations; generated probes remain labeled by provenance.
11. **Untrusted-code execution.** Comparing a pull request executes its code. The security model must say “trusted local candidates” for v1 and defer untrusted PR execution until real sandboxing receives independent review.
12. **Contract portability.** If the compiled regression only runs through this tool, the decision is hostage to the tool. Emit ordinary test code plus a small declarative decision record.

### Verdict

**PROCEED, conditional.** B has one new user job, one crisp data structure, a dramatic local aha, and a falsifiable correctness envelope. It is heavy because isolation, canonical observations, stability classification, minimization, contract generation, CLI, and decision UX are all load-bearing. It is coherent because every subsystem serves the same question: **what did the alternative implementations force the human to decide?**

## Finalist C — repository flight simulator / impact hypothesis

### Strongest form

Before a change, construct an impact hypothesis from repository structure, routes, schemas, tests, configuration, and observed runtime edges. Generate probes over the predicted radius. After the patch, overlay actual changes and observations, then highlight surprises: unpredicted dependencies, unexercised paths, state drift, and behavior outside the declared radius.

The honest phrase is **impact hypothesis**, not digital twin or simulation. Anything stronger overclaims.

### Collision ruling

| Neighbor | Collision | Judgment |
| --- | --- | --- |
| Wake | Low | Different job; Wake could supply observed provenance. |
| didrun | Low | didrun can receipt probes; it does not predict impact. |
| Parallel Code / Forge | Medium | If C compares branches, their worktree/diff UI already covers the visible shell. |
| Visual regression | **High for UI changes** | Baseline/actual/diff and change approval are mature; C needs cross-layer causality to be distinct. |
| Differential testing | Medium to high | Comparing predicted and actual outcomes or multiple candidate branches converges on differential analysis. |
| VibeContract / Spec Kit | Medium | Declared change scope, invariants, and acceptance criteria overlap; C's distinct value must come from measured repository structure. |
| Static/impact tools | **Severe** | Bazel reverse dependencies, Nx `affected`, CodeQL control/data-flow graphs, and test-impact analysis already own large parts of the model. |
| Session replay / agent bug repro | **Severe if user-journey centered** | Repro.dev already records clicks, errors, requests, and DOM state for coding agents via MCP; Highlight and OpenReplay already join replay, errors, network, traces/logs, and source maps. |

Official examples make the prior-art density concrete:

- [Bazel query](https://bazel.build/query/guide) can trace dependencies and reverse dependencies to ask what code a change may break.
- [Nx affected](https://nx.dev/docs/features/ci-features/affected) uses Git plus the project graph to determine the minimum affected projects and tasks.
- [CodeQL](https://codeql.github.com/docs/codeql-overview/about-codeql/) builds AST, control-flow, and data-flow representations for queryable program analysis.
- [Repro.dev](https://repro.dev/) captures interactions, errors, network traffic, replay, and DOM state and exposes the recording to coding agents through MCP.
- [Highlight](https://docs.highlight.io/session-replay) joins a button click to server-side logs/errors; [OpenReplay](https://docs.openreplay.com/en/session-replay/) captures journeys, WebSockets, errors, and can turn sessions into E2E tests.

The remaining distinct join—source-symbol impact **plus counterfactual alternatives plus executable human contracts**—is essentially B with an impact-analysis probe planner. That is a reason to defer C as a module, not launch it as a second center.

### Strongest falsifiable v1

Lock to one ecosystem and two cross-cutting changes. Before revealing each patch, construct a typed impact hypothesis from static imports, route definitions, tests, and user-declared invariants. Require it to predict a measured set of affected surfaces and generate real probes. After revealing the patch, score precision and recall against hidden fixture instrumentation and show at least one useful surprise without missing a designated critical edge.

**Kill it** if:

1. the output is a dependency graph with AI prose;
2. warnings are generic enough to apply to every change;
3. one critical dynamic edge is silently omitted;
4. runtime recording or session replay provides the aha by itself;
5. the demo only works because the fixture's architecture was hand-encoded;
6. supporting a second framework requires a second analyzer core.

### Hardest correctness risks

- **False negatives look like safety.** A missing dynamic edge is much worse than an extra warning when the UI calls the result a flight simulator.
- **Framework semantics dominate.** Routes, dependency injection, ORM relations, build plugins, reflection, generated code, and runtime configuration require ecosystem-specific models.
- **Static and dynamic evidence have incompatible bounds.** A static “may flow” and a runtime “was observed” cannot be merged into one confidence number.
- **State-space explosion.** “Blast radius” includes combinations of identity, data, timing, cache, queue, and external service behavior.
- **Baseline drift.** The repository model and observed traces go stale; rebuilding them may be expensive and non-hermetic.
- **Fixture overfitting.** A hand-designed demo can make the graph appear clairvoyant.
- **Replay prior art.** If the system starts from a captured bug session and hands it to an agent, Repro.dev and observability vendors already do the obvious version.
- **Dangerous naming.** “Simulator,” “twin,” and “predicted safe” imply coverage the system cannot establish.

### Verdict

**PARK as a standalone v1.** Preserve a modest future component: an impact hypothesis may prioritize B's probe search or explain why a decision witness matters. It must retain source labels such as `STATIC`, `OBSERVED`, `DECLARED`, `INFERRED`, and `UNKNOWN`, and it must never gate correctness on presumed graph completeness.

## Weighted score against `00-selection-rubric.md`

Scores are integers from 1 to 5. The total is normalized to 100 by multiplying each criterion's weight by `score / 5`.

| Criterion | Weight | A: assurance graph | B: ambiguity lab | C: flight simulator |
| --- | ---: | ---: | ---: | ---: |
| New systems primitive | 22 | 2 | 4 | 2 |
| Eyebrow-raising demo | 16 | 3 | 5 | 5 |
| 2026 grounding | 14 | 5 | 4 | 4 |
| Vibe-coder legibility | 12 | 4 | 5 | 4 |
| Engineering depth | 12 | 3 | 5 | 5 |
| Offline proofability | 10 | 5 | 5 | 3 |
| Coherent heavy scope | 8 | 4 | 4 | 2 |
| Open-source extension surface | 6 | 4 | 4 | 4 |
| **Weighted total / 100** | **100** | **70.4** | **90.0** | **71.6** |

### Score rationale

- **A wins grounding and proofability** because the review-evidence pain is explicit and didrun already demonstrates the local workflow. It loses novelty and depth precisely because the installed tool has already implemented the hard receipt mechanics and refuses the semantic overclaims A would be tempted to make.
- **B wins the duel** because the demo is immediate, the semantic center remains distinct under every substitution test, the hard parts are real deterministic systems work, and it can prove its bounded claim offline. It does not receive a 5 for novelty because differential testing and DiffSpec are genuine ancestors, or for OSS extension because adapter proliferation could fragment comparability.
- **C is the most cinematic idea** and contains deep engineering, but “repository behavior” is not one coherent analyzable domain. Its score is dragged down by dense prior art, weak universal proofability, and scope explosion.

## Proceed / park recommendation

### Proceed: B, with these locks

1. **LOCKED source of truth:** normalized behavior matrix plus adjudicated decision contracts. No agent event log.
2. **LOCKED primary job:** expose underspecified behavior among plausible implementations and turn the human's decision into a runnable contract.
3. **LOCKED producer boundary:** candidates are Git trees. The product does not spawn agents, manage worktrees, recover sessions, or merge branches.
4. **LOCKED receipt boundary:** didrun grades are imported/referenced verbatim. The product never rebrands them as proof or defines a competing tree-freshness grade.
5. **LOCKED comparison boundary:** equivalence is only observational over a finite, versioned probe set and environment policy.
6. **LOCKED minimization boundary:** claim only 1-minimality relative to named reducers and a stable partition predicate.
7. **LOCKED decision boundary:** the human selects behavior or rejects all, never selects an agent as the semantic operation.
8. **LOCKED output boundary:** generated decisions compile to ordinary tests that run without the product.
9. **LOCKED security boundary:** trusted local candidates only in v1; untrusted code isolation is deferred and visibly human-gated.
10. **LOCKED UI axis:** decision witnesses and behavioral partitions, not chronology, agent lanes, side-by-side source diffs, or global quality scores.

### Park: A

Retain only receipt references, a small decision schema, and perhaps an export view. Revisit the broader assurance graph only after B proves that users repeatedly preserve discovered decisions—and then position it as a didrun companion or upstream extension.

### Park: C

Retain impact hypotheses as a future probe-prioritization adapter. Do not call the product a simulator or twin. Any browser journey input must differentiate explicitly from Repro.dev, Highlight, and OpenReplay by driving cross-candidate decision witnesses rather than replaying one failure.

## Deep-dive attack agenda

The next phase should try to kill B on the following load-bearing claims before the concept brief locks architecture:

1. Can a runner make environment parity inspectable without pretending hermeticity?
2. Which observation algebra handles HTTP and CLI while preserving raw evidence and preventing normalizer overreach?
3. Can a partition-preserving reducer produce honest 1-minimal witnesses for both unordered structured input and ordered stateful actions?
4. How are unstable and uncomparable outcomes represented so they cannot masquerade as divergence?
5. Can generated regression tests remain framework-native, readable, and independent of the tool?
6. Can the UI lead a non-expert to choose behavior rather than branch identity, while explaining that consensus is not correctness?
7. Can the system execute trusted candidates safely enough for a local v1, and is the boundary to untrusted PRs unmistakable?
8. Can the second fixture family pass without a new bespoke core?
9. Does didrun integration preserve its exact grades and secret-handling boundaries without copying its manifest?
10. Does any implementation step accidentally recreate Parallel Code/Forge worktree management, Wake execution, Spec Kit planning, Playwright snapshot approval, or Repro-style session replay?

## Final judge's note

The assurance graph is the responsible idea. The flight simulator is the spectacular idea. The ambiguity lab is the idea with a chance to be both.

Its memorable claim is not “AI code you can trust.” That phrase is too broad and already crowded. It is:

> **When several plausible implementations all look finished, make them reveal the decision your prompt forgot to specify—then turn that decision into a test the repository remembers.**

That is eyebrow-raising, technically honest, usable without a model credential, distinct from Wake and didrun, and sharp enough to fail.
