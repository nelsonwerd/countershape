# Founder falsification: kill the feature piles, keep one new operation

- **Lens:** brutal cofounder review in the Soren Vale persona
- **Date:** 2026-07-14
- **Material inspected:** every current ideation artifact, the grounding corpus, `docs/PERSONA_SOREN_VALE.md`, the live concept brief, current product/docs pages, primary papers, and source snapshots of Wake, didrun, Parallel Code, and Automagik Forge
- **Forced decision:** **Kill Threadline as a standalone system. Kill BranchLab as a branch tournament or candidate-selection product. Proceed only with the surgically narrower counterfactual kernel described here: _surface and resolve a Choicepoint_.**
- **Confidence:** 8/10 on the collision findings; 6/10 that the surviving product object is distinct; 4/10 that it will prove useful beyond a spectacular demo; 2/10 on adoption or demand, which is not validated.

## Executive verdict

Threadline is an observability stack with a useful context export. That is not a new category. [Repro](https://repro.dev/) already records clicks, DOM state, console errors, and network traffic for coding agents through MCP. [Highlight](https://highlight.io/docs/getting-started/frontend-backend-mapping) already connects a frontend session to backend errors and distributed traces. [OpenReplay](https://docs.openreplay.com/) already self-hosts session replay and joins it to application state and backend logs. [Traceway](https://tracewayapp.com/) already combines session replay, cross-service traces, source-mapped exceptions, and agent-facing debugging tools. [Trailblazer](https://doi.org/10.1145/3746059.3747652) already answers reachability questions with annotated, replayable program traces. “Trace the product, not the agent” is good product copy wrapped around a heavily occupied mechanism.

BranchLab is more defensible, but the version described in the architecture option is still a feature composition unless one sentence and one object are made load-bearing. Parallel Code already has an **AI Arena**, worktrees, diffs, and winner selection. Automagik Forge already has repeated attempts, side-by-side comparison, and a choose-the-best workflow. Switchman already scans parallel branches for semantic overlap and interface drift, emits merge-confidence states, and gates CI. Research already clusters patches by behavior, generates difference-exposing tests, searches for discriminating inputs, infers ambiguity from candidate dispersion, shrinks counterexamples, filters candidate code with static and dynamic evidence, and selects or fuses candidate patches.

The surviving operation is not “compare branches,” “find a difference,” “pick a winner,” or “write a contract.” It is:

> **Ask the code variants to reveal a product decision I failed to make; reduce that decision to one stable, inspectable case; then make my answer executable.**

The surviving semantic object is a **Choicepoint**:

> A versioned, stable, locally minimal stimulus that partitions exact candidate trees into distinct observed outcomes under one declared world and projection, plus a human-scoped adjudication and the exact executable predicate compiled from it.

If the product cannot make that operation feel categorically different from reviewing several diffs or approving a snapshot, kill it. There is no second-place version where a handsome dashboard rescues the concept.

## The prior-art floor is much higher than the current options admit

### Source snapshots used for the local systems comparison

- `nelsonwerd/wake` at `fff9bfa44a011b51c6fb7f188343a0352af1da79`
- installed didrun source at `6ca774a311ef74b1cfd1cd03e9e87023c04ba35e`
- `johannesjo/parallel-code` at `144535d7b5735c10175608531bc379b068aa28e6`
- `automagik-dev/forge` at `9ff6c06923fe653815ac95b93a2c10dd0351ce8a`

These are evidence of current implementation surfaces, not evidence of market demand or production maturity.

### Threadline's proposed stack already exists in pieces and in products

| Threadline claim | Current primary prior art | Falsifier's ruling |
| --- | --- | --- |
| Record a bug or journey as clicks, DOM state, logs, and requests | [Repro](https://repro.dev/) records exactly those inputs and exposes them to coding agents via MCP; Playwright Trace Viewer records action snapshots, source, console, and network | **Occupied** |
| Connect a browser action to backend execution | [Highlight full-stack mapping](https://highlight.io/docs/getting-started/frontend-backend-mapping) propagates request identity from frontend sessions through backend services and maps errors/logs to the session | **Occupied** |
| Self-host a replay and join it with backend context | [OpenReplay](https://docs.openreplay.com/) is an open-source self-hosted session replay stack with application-state plugins and backend-log integrations | **Occupied** |
| Join replay, traces, exceptions, and source mapping, then hand telemetry to an agent | [Traceway](https://tracewayapp.com/) advertises that complete loop, including an MCP server and agent skills | **Occupied** |
| Produce a precise code path that explains a behavior | [Trailblazer](https://andrewhead.info/assets/pdf/trailblazer.pdf) produces replayable annotated program traces for reachability questions, while classical slicing and dynamic tracing precede it | **Research-occupied** |
| Compare behavior before and after | Session replay, visual regression, distributed tracing, and regression testing all do variants of this | **Commodity composition** |
| Cut the trace into agent context | Potentially useful, but a formatter/exporter over an existing trace graph is not enough to carry a heavy new system | **Feature, not category** |

The only potentially sharp Threadline operation is “point at a visible state and ask which code causally produced it.” Even that is not enough. Repro starts from the recorded interaction, Highlight and Traceway traverse the full stack, and Trailblazer starts from a reachability question. Threadline would have to prove a far stronger property: that its behavior-to-source slice is precise enough to reduce agent context, stable enough to version as a contract, and materially better than exporting correlated telemetry plus a symbol index. That is three research risks stapled to an observability platform.

**Threadline kill verdict:** do not build it as the product. At most, keep “browser observation -> source-linked context” as a future adapter or import path for the counterfactual system. Do not build another recorder, APM, session store, trace viewer, or symbol graph in this run.

### Branch comparison and “which patch wins?” are already products

[Parallel Code](https://github.com/johannesjo/parallel-code) creates an isolated branch/worktree per agent, includes diff review and coverage badges, and explicitly ships an **AI Arena — race agents head-to-head** surface. [Automagik Forge](https://github.com/automagik-dev/forge) models multiple attempts per task, lets users compare attempts side by side, and asks them to choose one to merge. [Switchman](https://switchman.dev/) scans branches or worktrees for semantic overlap, interface drift, and green-looking incompatibilities; it emits green/amber/red/uncertain merge confidence and can gate CI.

That means all of these BranchLab pitches are dead on arrival:

- run several agents on the same task;
- put each candidate in a worktree;
- show the candidates side by side;
- compare diffs, tests, coverage, or performance;
- explain semantic overlap;
- rank the candidates;
- say whether they are safe to merge;
- merge or splice the winner.

Those can be inputs, integrations, or explicitly deferred conveniences. None can be the identity.

### Candidate-selection research has already crossed the obvious frontier

The research collision is worse than the product collision:

1. [xTestCluster](https://arxiv.org/abs/2207.11082) executes generated tests across plausible repair patches, clusters patches by dynamic behavior, and gives reviewers tests that expose behavioral differences. “Behavioral clusters plus a differentiating input” is prior art.
2. [SpecFix](https://arxiv.org/abs/2505.07270) samples multiple programs from one natural-language requirement, analyzes the induced behavior distribution as evidence of ambiguity, and repairs ambiguous requirements. “Implementation diversity reveals underspecified intent” is prior art.
3. [DiffSpec](https://arxiv.org/abs/2410.04249) uses natural-language specifications and code artifacts to generate targeted differential tests, producing confirmed bugs in eBPF and Wasm implementations. “Use prose and code to search for meaningful implementation differences” is prior art.
4. [Dynamic-Static Synergistic Selection](https://doi.org/10.1609/aaai.v40i38.40481) combines AST analysis with dynamic functional-consistency evidence to select candidate code while reducing reliance on flawed generated tests. “Static plus dynamic evidence picks a candidate” is prior art.
5. [Agentic Verifier](https://openreview.net/forum?id=8bHa5wc7X5) actively searches for discriminative inputs that expose behavioral discrepancies among candidate solutions. “A verifier should search for a separating input” is current research, not a future idea.
6. Team Atlanta's [2026 patching study](https://team-atlanta.github.io/blog/post-patch-2026-ensemble/) ran ten agent configurations, manually classified semantically wrong green patches, and found that ensemble selection usually matched or beat every component; three candidates were its practical sweet spot. “Contrast makes the better patch visible” is empirically active.
7. [PatchFusion](https://arxiv.org/abs/2607.01597) goes beyond selection and deterministically fuses repeated edit atoms across candidate pools. “Consensus across patches contains useful evidence” is also occupied, even though consensus is not intent.
8. [DDMT](https://arxiv.org/abs/2607.00929) combines delta debugging with metamorphic relations to preserve a property without a conventional oracle. Oracle-deficient reduction is itself active research.

The concept therefore cannot claim invention of differential execution, candidate clustering, ambiguity detection, counterexample search, shrinking, candidate selection, or patch fusion. It is an ambitious synthesis whose only credible novelty is the human decision lifecycle at repository scale.

### Contract-first tooling occupies the other half

The relevant “Pact” is not only the established consumer-contract ecosystem. [Pact Tools](https://pact.tools/docs/) is a 2026 plan-first agent framework whose pitch is “contracts before code, tests as law.” It decomposes work into typed contracts and tests, supports parallel and competitive agent implementations, scores winners by test pass rate and runtime, performs adversarial review and spec-compliance audits, carries durable task state across Claude and Codex, and keeps contract artifacts in the repository.

[VibeContract](https://arxiv.org/abs/2603.15691) likewise proposes intent decomposition, developer-validated task contracts, contract-guided generation, traceability to code, and contract-guided testing and runtime verification. The older [Pact consumer-contract ecosystem](https://docs.pact.io/) already persists executable interaction contracts and replays them against providers.

So BranchLab also cannot win on:

- compiling prose into contracts;
- making tests the judge;
- running competitive implementations against a contract;
- persisting a contract as a repository artifact;
- tracing intent to code and verification;
- using hidden tests to prevent Goodhart behavior.

The surviving arrow must run backward from implementations to a previously absent human decision:

```text
contract-first systems
  declared intent -> contract -> implementations -> conformance

surviving counterfactual system
  plausible implementations -> stable divergence -> missing decision
  -> human ruling -> exact-scope executable contract
```

If it starts from a polished spec and merely tests candidate compliance, Pact and VibeContract have already won.

### Wake and didrun close the remaining escape hatches

[Wake](https://github.com/nelsonwerd/wake) owns the local heterogeneous-agent runtime: subprocess protocol, crash-durable event log, scheduling, coordination, effect accounting, resume, replay, fork, and causal viewer. The proposed product must accept ordinary Git refs and must not contain an agent supervisor, fleet schema, recovery protocol, hash-chained run history, or agent timeline. It may launch a short-lived application command to observe behavior; that is test execution, not a durable agent runtime. The distinction must remain obvious in the architecture.

didrun owns deterministic evidence that a command ran against an exact Git tree, plus typed claims, commit-bound sealing, strict re-verification, grades, and an offline HTML report. The proposed product must not create a competing receipt ledger, redefine didrun grades, or imply that a successful probe proves the resulting choice is right. A Choicepoint may reference a didrun receipt verbatim. Its distinct truth is the observed N-way partition and the human ruling—not command freshness.

This also kills the assurance-graph finalist as a standalone product for this run. A claim/evidence graph would sit directly between didrun's receipts and Pact/VibeContract's contracts. Useful integration is not a sufficient new primitive.

## Attempt to kill BranchLab itself

Even after subtracting the prior art, the surviving kernel has six ways to be a sophisticated toy.

### 1. Candidate disagreement may be mostly junk

Variants differ in log wording, response order, generated ids, timestamps, dependency versions, race timing, error copy, or incidental markup. A normalizer can suppress that noise, but every suppression is also an opportunity to erase the behavior the user should have noticed. A tool that emits 40 harmless distinctions to find one meaningful choice does not compress review; it creates a new review queue.

### 2. A minimal witness may be less understandable than the original scenario

Delta debugging preserves a predicate, not meaning. Removing an authentication step, fixture row, or browser action can retain the same final partition for a different cause. A tiny JSON payload may be technically 1-minimal and product-incomprehensible. The UI must preserve the original scenario and the shrink derivation, and it must say “locally minimal under these reducers,” never “root cause.”

### 3. The human can choose the wrong thing

The system discovers a question; it does not know the answer. A vibe coder may prefer a result that leaks tenancy metadata, breaks compatibility, or violates a regulation. “Human-approved” is provenance, not correctness. Security-sensitive Choicepoints need explicit risk copy and may require an expert gate, but the system cannot invent one.

### 4. Exact witness decisions do not safely generalize

Choosing `404` for one cross-tenant invoice id does not imply `404` for every concealed resource. Compiling a broad predicate from one example silently strengthens the human's decision. The default contract scope must be **this exact witness**. Extending it to an equivalence class or generator domain must be a separate explicit operation with fresh counterexamples.

### 5. Meaningful probe generation may require the domain knowledge the product claims to discover

Without a schema, seed scenarios, or a real model, a generic search over a web application is hopeless. With a schema and seeds, the product may collapse to Schemathesis or fast-check plus a custom comparator. The golden path must be honest: a finite declared probe grammar is required input, and the product's value is the decision compression after execution—not magical test generation.

### 6. Running arbitrary candidate trees is a security problem, not a worktree problem

Postinstall scripts, build hooks, symlinks, fork bombs, credential reads, and network exfiltration all precede the interesting comparison. A worktree is not a sandbox. The first build can demonstrate disposable local worlds with no credentials, denied external network, resource limits, and process-group cleanup; it cannot claim hostile-code containment or production safety without independent review.

These risks are existential. If the build hides them behind a green status or one “best branch” score, the concept should be stopped even if every test passes.

## Forced reframe: the product is a Choicepoint compiler

### The new user operation

The one operation worth building is:

> **Resolve a Choicepoint:** take a stable, minimized divergence among plausible implementations, choose the allowed observed outcome set (or reject all / defer / demand another distinction), choose the scope of that ruling, inspect the exact predicate, and compile it into a replayable repository contract.

The system may perform many internal steps, but every surface must converge on that verb. The home screen is not a branch list or a run timeline. It is a queue of unresolved Choicepoints ordered by explicit, explainable factors such as boundary type and affected declared scenario—not an opaque risk score.

### The new semantic object

A **Choicepoint** is not a screenshot diff, test result, or selected branch. It is this exact object:

```text
Choicepoint
  identity
    choicepoint id
    schema version
    state: DISCOVERED | STABLE | RESOLVED | DEFERRED | INVALIDATED

  experiment jurisdiction
    candidate Git tree ids
    world/reset fingerprint
    runner + toolchain fingerprint
    probe grammar + seed
    normalizer/projection version

  divergence witness
    original stimulus
    locally minimized stimulus
    reducer set + attempted reductions
    repeated-run stability evidence
    raw observation digests
    normalized N-way outcome partition
    UNKNOWN / UNCOMPARABLE members

  human ruling
    allowed outcome groups | reject all | refine | defer
    scope: exact witness | explicit equivalence class | declared generator domain
    rationale + author + time
    warnings and requested expert gate

  executable residue
    exact predicate shown to the human
    deterministic contract digest
    rerun command
    per-candidate conformance observations
    receipt references, never upgraded
```

The state transition is the category-defining act:

```text
candidate trees
  -> DISCOVERED divergence
  -> STABLE Choicepoint
  -> human RESOLVED / DEFERRED / INVALIDATED
  -> deterministic contract
  -> fresh candidate conformance observations
```

The source-of-truth jurisdictions must remain separate:

| Fact | Authority |
| --- | --- |
| Repository content | Git tree id |
| A command ran and its evidence freshness | didrun receipt and verbatim grade |
| Agent execution history | Wake, if supplied; never required |
| Candidate behavior under this declared experiment | Choicepoint raw observations + pure normalizer |
| Which observed behavior the product owner wants | Human ruling |
| Whether a candidate satisfies the compiled exact predicate | Deterministic contract replay |
| Whether the software is correct, secure, maintainable, or production-ready | **Not established by this system** |

### What must be removed from the current BranchLab pitch

- Remove agent launching and “AI arena” language.
- Remove automatic winner selection and global candidate ranking.
- Remove patch splicing, cherry-picking, and merge orchestration.
- Remove a generalized assurance graph and competing receipt format.
- Remove model-generated intent or tests from the golden path.
- Remove browser exploration and screenshot search from the first proof; explicit browser replay can remain a later adapter.
- Remove “minimal semantic difference,” “equivalence,” “safe,” and “correct” from claims. Use observed partition, local minimality, conformance, and unknown.
- Remove an all-framework promise. Prove CLI and HTTP in one Node/TypeScript reference domain first.

This still leaves a heavy system: immutable candidate materialization, disposable worlds, a typed stimulus engine, repeated execution, raw and normalized observations, N-way partitioning, structure-aware shrinking, a stateful Choicepoint schema, deterministic contract compilation, a CLI, and a responsive visual decision bench. The scope is heavy because the semantic kernel is hard, not because every adjacent feature is included.

## Steelman: why the narrowed idea might matter

AI makes plausible implementations cheap and specifications expensive. Existing tools treat diversity as throughput (“run more agents”), selection evidence (“pick the best patch”), or defect evidence (“one implementation differs”). The strongest version of this product treats diversity as a **question generator for human intent**.

That creates a new habit for serious vibe coders:

1. Ask two or three agents—or humans—to implement one underspecified change.
2. Import the committed trees from any producer.
3. Let the system search a bounded, declared behavior space.
4. Receive one small question backed by exact outcomes, not three large diffs.
5. Answer the product question once.
6. Keep the answer as executable repository knowledge independent of every candidate and agent transcript.

The delight is not “candidate B won.” It is:

> “I did not realize I had to decide what cross-tenant lookup should reveal. The variants exposed it, the system reduced it to one request, and my answer now survives all of them.”

That is stranger and more durable than a branch dashboard. It is also honest about the machine's role: implementations nominate the question; a human owns the meaning; a deterministic check preserves the ruling.

## Falsifiable smallest proof

Build one vertical proof around two substantive local domains, with no model credential and no fake external API.

### Domain A: authorization semantics over HTTP

- Three committed candidate trees implement “support may view invoices but may not mutate payment methods.”
- Every candidate passes the existing visible test suite.
- A finite schema-derived grammar varies role, tenant relation, resource existence, read/mutate action, and a small request-body shape.
- Repeated identical runs distinguish stable code divergence from a deliberately flaky fixture.
- The system discovers at least one behavior not named in the tests, such as `403` vs `404` vs metadata-bearing `200` for a cross-tenant resource.
- The shrinker produces a materially smaller stable witness and preserves its reducer audit.
- The decision bench lets a user allow one observed outcome, allow several, reject all, defer, or request another distinction.
- The compiler defaults to an exact-witness predicate and shows every asserted field before save.
- Replay grades each candidate as conforming, nonconforming, unstable, or unobserved—never correct or safe.

### Domain B: configuration precedence over a CLI

- Three committed candidates merge argv, environment, and config-file values differently.
- Search distinguishes absent, empty, explicit-clear, and inherited values.
- Shrinking reduces a larger fixture to the smallest stable precedence Choicepoint.
- The same artifact and compiler work without HTTP, proving that the kernel is not an API-specific test viewer.

### Proof obligations

The smallest proof passes only if all are true:

1. The same committed candidate trees and declared world produce byte-identical normalized partitions and Choicepoint digests across three clean runs.
2. A deliberately flaky candidate becomes `UNSTABLE`, not a false code verdict.
3. A world mismatch becomes `UNCOMPARABLE`, not a divergence.
4. At least one Choicepoint is not already represented by the visible tests.
5. At least one seeded case produces `reject all` / `none conforms`.
6. The final witness is materially smaller than the original probe under an auditable reducer trace.
7. Contract compilation is deterministic and never asserts an unselected observation field.
8. The core works from plain Git refs and ordinary local commands; Wake and agent APIs are absent.
9. didrun grades, if imported, are displayed verbatim and never promoted into a semantic verdict.
10. The CLI and dashboard make the ruling possible without reading candidate source diffs or raw logs; raw evidence remains one action away.

## Success metric

For each of the two domains, on the reference machine:

- from three committed refs plus a declared finite probe grammar to the first stable Choicepoint in **under 10 minutes**;
- shrink at least one divergence by **50% or more** in structured stimulus size while preserving the same stable partition;
- reproduce the Choicepoint and contract digests across **three clean executions**;
- compile and rerun a human ruling against every candidate with no model or network dependency;
- in a later human test, a mid-level vibe coder correctly explains the choice and resolves it in **under 90 seconds without opening a diff**, while distinguishing “conforms to this ruling” from “correct.”

The last bullet is `PENDING HUMAN`; the autonomous run cannot mark it validated.

## Kill criterion

Park the concept—even if the implementation is beautiful—if any one of these holds after the vertical proof:

1. The top meaningful divergence in either domain is already expressed by the existing tests; the product discovered no missing decision.
2. Environmental noise or normalization disputes dominate the first ten stable-looking divergences.
3. A reviewer must inspect source diffs or raw logs to understand and resolve the Choicepoint.
4. The generated contract is an opaque snapshot or silently generalizes beyond the selected witness.
5. A small script combining Git worktrees, Schemathesis or fast-check, `jq`, and ApprovalTests reproduces the same comprehension and durable artifact with little lost value.
6. The product needs an LLM to invent useful probes or decide positive conformance in the golden path.
7. The build drifts toward agent launching, run replay, evidence grading, spec generation, merge safety, or winner selection.
8. The visual hero becomes branches, agents, timelines, or scores instead of one unresolved product choice.
9. The system cannot make `UNKNOWN`, `UNSTABLE`, `UNCOMPARABLE`, `DEFERRED`, and `REJECT ALL` first-class and understandable.
10. A real user sees the operation as “fancy differential tests” after the complete aha flow.

## Final forced choice

**Choose the Choicepoint compiler. Kill everything else for this build.**

- **Threadline:** killed as a standalone product; mature observability/session-replay systems and Trailblazer occupy its mechanism. Retain only as a possible future observation adapter.
- **Assurance graph / Outcome Circuit:** killed as the semantic center; it falls between didrun's receipts and Pact/VibeContract's contracts.
- **BranchLab as arena, evaluator, ranker, or merge tool:** killed; Parallel Code, Forge, Switchman, selection research, and patch-fusion research occupy it.
- **BranchLab's counterfactual kernel:** proceed only under the new operation and object above.

The steelman is strong enough for a deep dive and a hard prototype. The novelty claim is deliberately modest: this is not a new differential-testing algorithm. It is a proposed new human operation over real candidate software—**resolve a Choicepoint**—and a durable object that turns implementation diversity into executable product intent.

**Overall confidence: 6/10.** Architecture feasibility is 7/10 for CLI/HTTP in a bounded domain. Product distinctness is 6/10. Autonomous craft proof is achievable. Adoption, decision quality, and production safety remain human and security-review gates.
