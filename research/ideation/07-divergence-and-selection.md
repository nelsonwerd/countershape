# Divergence and selection — from durable agents to executable ambiguity

- **Founder:** Soren Vale
- **Date:** 2026-07-14
- **Decision:** advance the counterfactual ambiguity laboratory into deep dive
- **Working product name:** **Countershape**
- **Confidence:** 7/10 on concept coherence; 6/10 on technical proofability; 3/10 on adoption because no user study has happened

## The decision in one paragraph

Build a local laboratory that accepts several committed Git trees claiming to implement the same intent, drives them with identical finite probes, groups them by normalized observed behavior, reduces a stable disagreement to a locally minimal witness, asks a human which behavior they actually meant, and compiles that choice into an ordinary executable contract. The lasting object is a **Choicepoint**—the concrete Counterfactual Intent Decision—not an agent run, winning branch, score, trace, or screenshot. Wake may produce candidate commits; didrun may receipt commands; neither is reimplemented.

The project is a systems/product synthesis, not a claim to have invented differential testing, delta debugging, approval testing, or executable specifications. Its experimental claim is narrower: alternative implementations can act as sensors for tacit intent, and a disciplined local tool can compress their disagreement into a decision a serious non-expert can make and preserve.

## What changed the direction

### 1. The initial runtime thesis failed the Wake test

Direct inspection of [`nelsonwerd/wake`](https://github.com/nelsonwerd/wake) showed that the provisional local agent runtime was essentially already built: heterogeneous subprocess fleets, a crash-durable hash-chained event log, coordination, effect accounting, replay/fork, resume, CLI, and a causal viewer. Rebuilding it under a friendlier name would be a weaker copy. The first concept was killed.

### 2. The assurance-graph thesis failed the didrun test

The installed didrun already captures exact commands and Git tree state, binds structured claims, grades freshness, seals commit manifests, exports bundles, and emits HTML. A broader intent/claim graph could be useful, but its hard v1 would be `didrun + semantic metadata + another report`. That is a companion or future didrun direction, not the strongest new infrastructure project.

### 3. Behavior-to-code tracing failed the crowded-substrate test

The inversion “trace the product, not the agent” remains useful. Yet [Repro](https://repro.dev/) already captures browser interactions, DOM state, errors, and network activity for coding agents through MCP; [Highlight](https://www.highlight.io/), [OpenReplay](https://openreplay.com/), and related observability products already connect frontend replay to backend evidence. A source-symbol context cutter might still differentiate, but too much of the flagship demo is established session-replay/APM behavior.

### 4. Counterfactual intent survived only after losing its inflated novelty claim

[SpecFix](https://arxiv.org/abs/2505.07270) already uses behavioral dispersion among sampled programs to identify ambiguous requirements. [xTestCluster](https://arxiv.org/abs/2207.11082) already clusters plausible patches using difference-exposing tests. [Diffy](https://blog.x.com/engineering/en_us/a/2015/diffy-testing-services-without-writing-tests) already sends the same traffic to service variants and filters noise. Property-based tools and delta debugging already generate and shrink counterexamples; approval-testing systems already turn human choices into baselines.

Those findings remove any honest claim to a new algorithm. They leave one differentiated lifecycle object:

```text
exact candidate trees
  + versioned controlled world
  + stable locally-minimal stimulus
  + raw and normalized N-way outcomes
  + explicit human semantic choice
  + exact scope and provenance
  = Choicepoint (Counterfactual Intent Decision)
```

The human is not asked to choose a model or branch. They **resolve a Choicepoint**: choose an outcome, allow several outcomes, reject all, defer, or request a finer distinction. The ruling becomes a repository-native contract; every candidate is rerun against it.

## The seven options that competed

Scores use the weighted rubric in `00-selection-rubric.md`. They are directional judgments, not market measurements.

| Rank | Concept | Semantic object | Weighted read | Decision |
| ---: | --- | --- | ---: | --- |
| 1 | **Countershape / Choicepoint compiler** | Choicepoint | **90.0/100** | Advance |
| 2 | Behavior-to-code causal workbench | Behavior Thread | 79.2/100 after prior-art penalty | Park as possible observation adapter |
| 3 | Capability Flight Plan | Least-authority task manifest | 78.0/100 | Park; security scope overwhelms this build |
| 4 | Patchwork Calculus | Typed semantic change program | 76.8/100 | Research bet; general lifting/rebase risk too high |
| 5 | Repository flight simulator | Counterfactual impact hypothesis | 71.6/100 | Park; dense prior art and dangerous false negatives |
| 6 | Assurance graph / proof-carrying change | Claim-to-evidence graph | 70.4/100 | Park as didrun companion, not a new center |
| 7 | Wake Studio | Human projection of a Wake mission | 63.2/100 | Legitimate application path, not this greenfield systems bet |

### Why Countershape won

- **Different source of truth:** a behavior matrix plus explicit human decisions, not an event history, claim ledger, task board, baseline screenshot, or spec document.
- **Different user operation:** adjudicate a hidden semantic choice exposed by code variants.
- **Dramatic offline proof:** three green-looking implementations split on one minimized input; one click creates a test that reclassifies all three.
- **Agents are optional producers:** fixtures, humans, CI, Codex, Claude, or Wake can create commits; the comparison semantics do not change.
- **Reference-quality hard parts:** environment parity, process lifecycle, canonical observation algebra, stability classification, N-way partitioning, structure-aware shrinking, deterministic contract compilation, and a truthful decision UI.
- **Useful failure:** `UNSTABLE`, `UNCOMPARABLE`, `NO OBSERVED DIFFERENCE`, `REJECT ALL`, and `UNKNOWN` are first-class results.

## Locked semantic center

### One-line promise

**When several plausible implementations all look finished, Countershape finds the smallest behavior that makes them disagree, asks what you meant, and turns the answer into a test the repository remembers.**

### Primary user

A serious non-expert or mid-level builder maintaining a 5k-100k LOC TypeScript application, already comfortable asking coding agents for changes and using Git, but unable to economically review several implementations or enumerate every edge-case decision in advance.

### Primary job

Discover and resolve tacit product or security semantics before selecting or merging an AI-assisted implementation. The product verb is **resolve a Choicepoint**, never “pick a winner.”

### Source-of-truth jurisdictions

| Truth | Authority |
| --- | --- |
| Repository content | Git tree id |
| What a candidate emitted for a probe | raw observation bytes/artifacts plus the runner/environment manifest |
| Which observations compare equal | the named, versioned normalizer and comparator |
| Which stimulus is locally minimal | the shrink transcript, declared reducers, and stable partition predicate |
| What behavior the human intended | the signed/attributed Choicepoint ruling |
| Whether a command was receipted against a tree | didrun's exact verbatim grade, when referenced |
| Whether the software is correct or safe | **no authority in this system**; remains an open-world judgment |

### Aha moment

Three branches that all pass their existing tests disagree on cross-tenant invoice access. Countershape reduces the difference to one role/tenant/resource tuple and shows three outcomes: `403`, `404`, and a metadata-leaking `200`. The user chooses “conceal existence with `404`,” sees the exact predicate, saves it, and immediately watches the three candidates become `CONFORMS`, `CONTRADICTS`, and `CONTRADICTS`—without reading their diffs.

### Success metric

In under 15 minutes on each of two problem families, a mid-level builder can import three committed candidate trees, discover at least one stable behavioral decision absent from the existing tests, understand and adjudicate the minimized witness without reading source diffs or raw logs, compile the decision into an ordinary standalone test, and reproduce the same conformance result from a clean local checkout.

### Kill criterion

Park or radically narrow the concept if any of these survive the build loop:

1. Most reported differences are environment noise or harmless formatting.
2. The system cannot distinguish setup failure, flakiness, and genuine code divergence.
3. “Shrinking” does not preserve the exact candidate partition or cannot state its limited minimality honestly.
4. The user still has to review every source diff to understand the decision.
5. The human interaction is effectively “pick branch A.”
6. The generated contract only runs through Countershape or silently freezes more output than the user selected.
7. The second fixture family requires a bespoke comparison engine.
8. A shell script combining established tools produces the same comprehension and reproducibility.

## Smallest falsifiable proof

The deep dive may narrow implementation details but must not weaken this loop:

1. Import three exact committed candidate trees from one local repository.
2. Materialize independent temporary source roots without changing the user's branches.
3. Run the same versioned probe and declared environment policy against each.
4. Repeat enough to mark one deliberately flaky candidate `UNSTABLE`.
5. Preserve raw observations before applying any normalization.
6. Form deterministic N-way behavior partitions.
7. Reduce one structured stimulus to a 1-minimal witness relative to named reducers while preserving that partition.
8. Present a product decision, not a code diff or model ranking.
9. Compile the chosen outcome to an ordinary test and a readable decision record.
10. Rerun all candidates, including one case where `none conforms` is correct.
11. Reproduce the artifact and digest offline.
12. Exercise both an HTTP authorization domain and a CLI configuration-precedence domain.

No model credential, hosted sandbox, Wake component, or simulated external API is permitted in the proof.

## Provisional product boundary before deep dive

### In

- One local Git repository, 2-4 committed refs, committed content only.
- Safe content materialization into disposable roots; no agent launching or task/worktree UI.
- Two deterministic real adapters in the golden path: long-running HTTP service and one-shot CLI process.
- Explicit seed corpus and finite typed input grammar; useful blind generation is not promised.
- Repeated trials and per-candidate stability classification.
- Raw observation preservation plus versioned HTTP/JSON and CLI normalizers.
- N-way partitions with missing/error/timeout/setup states kept outside behavioral equality.
- Structure-aware reducers and a replayable shrink transcript.
- Intent choices: prefer one observed outcome, allow several, reject all, defer, or refine scope.
- Deterministic contract compiler targeting ordinary repository tests plus a machine-readable decision record.
- Monochrome CLI, responsive local studio, and self-contained report.
- A documented adapter protocol after two first-party adapters prove the abstraction.

### Out

- Agent orchestration, scheduling, process resumption, replay/fork of agent runs, or agent timelines.
- Generic Git worktree management, merge automation, patch synthesis, or “best branch” ranking.
- didrun-compatible claim grading implemented independently.
- Model-generated probes, explanations, decisions, or tests in the golden path.
- External model/provider APIs, fake integrations, hosted execution, collaboration accounts, billing, or cloud control plane.
- Untrusted pull-request execution, production traffic shadowing, or claims of a hardened sandbox.
- General database cloning, arbitrary distributed systems, or unconstrained GUI exploration.
- Semantic-equivalence or global-minimality claims.
- Universal quality/confidence scores.

### Deferred experiment, not a v1 promise

An explicit Playwright scenario adapter may replay a user-authored journey and compare DOM/accessibility/network observations. Browser input generation and stateful sequence shrinking remain out until the HTTP/CLI spine is correct.

## Provisional architecture hypotheses for deep dive

The deep dive must challenge, not ratify, these:

1. **Immutable import:** resolve each ref to commit and tree ids, extract the committed tree without checking it out into a user-visible branch, and record unsupported submodule/LFS cases.
2. **World manifest:** hash tool versions, candidate tree, setup/launch argv, sanitized environment policy, fixture/reset recipe, adapter, normalizer, and probe grammar.
3. **Process boundary:** argv arrays rather than shell strings, minimal inherited environment, per-world temp directories and ports, output/time limits, process-group teardown, and no claim of hostile-code isolation.
4. **Observation algebra:** raw bytes/artifacts are immutable; typed projections are versioned pure functions; absent and failed observations never equal successful empty values.
5. **Partition stability:** equality must be deterministic for canonical projections; unstable candidates create a separate class and are not minimized as stable code differences.
6. **Shrinking:** each reducer is typed; every attempt starts from a fresh declared world; cache keys include all semantic inputs; final claim is 1-minimal only under those reducers.
7. **Decision scope:** exact witness, user-edited predicate, and broader generalized rule are different acts. Compilation never silently generalizes.
8. **Portable contract:** generated tests use ordinary HTTP/CLI libraries and run without the Countershape binary.
9. **Read model:** the studio renders immutable experiment artifacts plus explicit decisions; chronology is an inspector detail, not the primary axis.

## Design direction

The metaphor is a **scientific comparison bench**, not agent mission control.

- Candidate identity is visually secondary to observed outcome clusters.
- The witness remains pinned while outcomes occupy aligned columns.
- Raw versus normalized evidence is always one action away.
- The UI says `NO OBSERVED DIFFERENCE`, never `SAME`; `DECIDED FOR THIS WITNESS`, never `CORRECT`.
- Color never carries trust alone. Shape, status text, and placement must survive monochrome and color-vision differences.
- Desktop at 1440px supports the full matrix, shrink trace, and predicate editor.
- Mobile at 375px supports one decision card, outcome switching, raw evidence, and defer/reject actions without horizontal panic.
- Empty state creates a real local example study. Error state preserves partial evidence and identifies the failed comparison boundary.
- Motion may reveal alignment and reduction, but reduced-motion mode loses no meaning.

## Provisional roadmap

Roadmap follows the semantic dependency, not UI convenience:

1. **R0 — truth model:** schemas, canonicalization, observation states, partitions, fixture vectors, and threat model.
2. **R1 — real execution:** immutable Git import, CLI adapter, HTTP adapter, reset/teardown, stability classification.
3. **R2 — counterexample engine:** finite generators, partition-preserving reducers, deterministic cache, replay/shrink receipts.
4. **R3 — intent compiler:** decision schema, predicate editor, ordinary-test emitters, conformance replay, `none conforms`.
5. **R4 — human bench:** CLI next action, local API, responsive studio, self-contained report, full visual loop.
6. **R5 — extension proof:** second domain, adapter conformance kit, explicit Playwright replay if the core remains green.
7. **R6 — hardening:** adversarial inputs, cancellation/cleanup, secrets/redaction, packaging, docs, install path, performance budgets.

No phase advances on compilation alone. Each shippable unit must pass its objective gates, declare didrun claims, commit, seal, and loop on strict verification before the next unit.

## Validated facts, technical inferences, and bets

### Verified locally or from primary sources

- Wake occupies the original runtime architecture in this workspace.
- didrun occupies deterministic command receipt, Git-tree binding, claim grading, commit manifests, and HTML evidence.
- Current agent products make parallel Git candidate generation increasingly ordinary.
- Differential testing, patch clustering, difference-exposing tests, shrinking, and approval baselines are established prior art.
- Repro/APM/session-replay tools substantially occupy browser-to-backend causal capture.
- Recent 2026 research documents an experience-dependent verification gap and higher review burden for novice vibe coders.

### Technical inferences to test

- A finite HTTP/CLI grammar can expose meaningful disagreements in repository-level candidate patches.
- Repeated runs plus explicit world fingerprints can separate enough noise from code behavior for a useful local experiment.
- Partition-preserving structured reduction can produce a much smaller decision witness within the time budget.
- A deterministic compiler can express a human-selected outcome without freezing accidental details.
- A decision-first UI can compress review more than a branch/diff-first UI.

### Unvalidated bets

- Serious vibe coders will pay the compute/time cost to produce or compare multiple candidates.
- Disagreements will frequently reveal valuable tacit intent rather than inconsequential implementation variance.
- Humans will make better decisions from outcome witnesses than from source review.
- Decision contracts will become a repeated maintenance habit instead of another artifact that drifts.
- Open-source adapter pull will exist.
- Any hosted or paid layer is justified.

## Handoff to deep dive

The concept is coherent enough to attack, not approved to build unchanged. Deep dive must independently investigate:

- prior-art/patent/non-duplication risk around SpecFix, xTestCluster, service diffing, and branch comparison;
- observation algebra and normalizer unsoundness;
- reproducible world materialization and honest local security limits;
- partition-preserving reduction and state-reset semantics;
- contract portability and accidental over-assertion;
- two-domain feasibility within a reference-quality build;
- human-comprehension and visual decision design;
- exact acceptance tests and cut lines.

The brief in `docs/CONCEPT_BRIEF.md` must be revised after synthesis and red-team. Any deep-dive finding that merely says “looks good” has failed its job.
