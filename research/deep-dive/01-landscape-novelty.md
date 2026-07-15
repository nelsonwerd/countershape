# Landscape and novelty audit: Countershape / Choicepoint

**Lane:** prior art, novelty, and Wake non-duplication
**Search date:** 2026-07-14
**Decision:** **MODIFY AND PROCEED**
**Overall confidence:** 8/10 on the landscape conclusion; 6/10 that the remaining end-to-end product boundary is publicly differentiated; 3/10 that “Choicepoint” is a new semantic primitive.

## Executive ruling

The current concept survives as a serious systems project, but not with its strongest novelty story intact.

The decisive collision is **Socrates / “Choose, Don’t Label: Multiple-Choice Query Synthesis for Program Disambiguation”** (PLDI 2026). Socrates starts from multiple candidate programs, finds regions in which their behavior differs, partitions the candidates into semantic behavior clusters, and asks a human to select among high-level behavior options. That is substantially the semantic heart currently attributed to a new “Choicepoint” operation. Earlier work such as FlashProg already used discriminating inputs and user choices to disambiguate candidate programs, while TiCoder turns user judgments about candidate tests into both candidate pruning and reusable regression tests. ARHF asks for desired behavior on ambiguous inputs. SpecFix treats implementation diversity as a sensor for underspecified requirements. Taken together, this prior art rules out claims that Countershape invented implementation-driven intent discovery, behavioral clustering for disambiguation, human choice among observed behaviors, or the conversion of that feedback into tests.

What remains genuinely interesting is the operational combination. I found no public source in this search that combines all of the following in one developer-facing system:

1. exact committed repository candidates rather than synthesized function bodies;
2. empirical black-box CLI and HTTP execution against a declared world;
3. world, runner, normalizer, and candidate fingerprints attached to every observation;
4. repeated execution and explicit stability classification before adjudication;
5. raw observations plus a declared normalization/projection boundary;
6. partition-preserving local reduction with a replayable shrink transcript;
7. a human ruling that can allow one outcome, multiple outcomes, no outcome, defer, or scope a predicate to one witnessed case; and
8. a deterministic, standalone repository test that no longer depends on Countershape.

That is a credible **repository-scale operationalization and synthesis of established techniques**, not a new disambiguation algorithm or a newly discovered human operation. The build should proceed if the brief is rewritten around that honest boundary. If the implementation collapses to static fixtures, a generic property-based shrinker, a JSON decision file, and an approval snapshot—without material environment identity, stability analysis, provenance, or standalone contract generation—it should be killed as an attractive composition demo.

## Scope and method

This was a public-source novelty landscape, not an exhaustive systematic review, patent claim chart, or legal freedom-to-operate opinion. I prioritized primary sources: papers and their artifacts, official documentation, official engineering posts, and project repositories. I searched the works named in the assignment and then followed conceptual neighbors that were closer than that initial list.

The audit distinguishes four questions that the current brief sometimes blends:

- **Algorithmic novelty:** Is candidate disagreement synthesized into a human-answerable query in a new way?
- **Artifact novelty:** Is the Choicepoint record itself a new semantic object?
- **Systems novelty:** Has the full repository/runtime/provenance/reduction/contract pipeline been publicly assembled before?
- **Product differentiation:** Is this usefully distinct from agent runners, branch comparison tools, contract-first systems, and testing libraries?

The answer is “no” to the broad algorithmic and semantic-object claims, “possibly, as a particular composition” to the systems claim, and “yes, if boundaries remain strict” to product differentiation.

## Severity-ranked findings

### P0 — Socrates directly occupies the proposed semantic center

[Socrates](https://arxiv.org/abs/2604.08792), published at PLDI 2026 under the title *Choose, Don’t Label: Multiple-Choice Query Synthesis for Program Disambiguation*, is the closest public work found. Given a finite hypothesis space of candidate programs consistent with a high-level specification, it discovers preconditions under which candidates differ, clusters candidates by semantic behavior, and presents a small multiple-choice query. Each answer corresponds to a Hoare triple describing a behavior cluster. The user’s choice eliminates inconsistent candidates. The system also optimizes the balance and interpretability of the question, and evaluates symbolic and neuro-symbolic variants across table, JSON, image-editing, and image-search domains. The accompanying artifact is publicly archived at [Zenodo](https://doi.org/10.5281/zenodo.19052770).

This is not merely adjacent terminology. It instantiates the same core loop: implementation set → discriminating situation → behavior partition → human intent selection → narrower acceptable set. Therefore these current or implied claims are untenable:

- “resolve a Choicepoint” is a new human operation;
- candidate implementations exposing a semantic choice is a new primitive;
- presenting N behavior classes rather than code diffs is novel;
- a human selecting the intended behavior among candidate-derived choices is novel.

The important residual distinction is execution domain and residue. Socrates reasons over formal or synthesized program hypotheses and emits information that prunes that hypothesis space. Countershape proposes arbitrary Git trees, empirical stateful black-box runners, named environmental context, flake/stability handling, raw observation provenance, reducer traces, and a standalone repository test. Those are meaningful engineering differences, but they are an operational extension of the interaction model, not a clean-room invention of it.

**Confidence: 10/10.** This finding must change the concept brief, positioning, and documentation.

### P0 — The end-to-end arrow is assembled from established components

Socrates is not the only collision. [TiCoder](https://arxiv.org/abs/2208.05950) generates candidate implementations and tests, asks whether a proposed test matches intent using `YES`, `NO`, or `UNDEFINED`, uses the answer to prune and rank implementations, and returns user-approved tests that can serve future debugging and regression. [ARHF](https://arxiv.org/abs/2508.14114) identifies ambiguity in code-writing tasks, proposes specific ambiguous inputs, and asks for limited feedback on the desired behavior. Microsoft Research’s [FlashProg user-interaction work](https://www.microsoft.com/en-us/research/publication/user-interaction-models-for-disambiguation-in-programming-by-example/) used active-learning-style inputs on which candidate programs disagree, showed alternatives, and treated the user’s selection as another specification example. [Synthesis with Abstract Examples](https://www.sri.inf.ethz.ch/publications/drachsler2017synthesis) similarly lets users accept a generalized behavior region or provide a counterexample.

On the output side, approval testing already makes human acceptance durable: the [ApprovalTests.cpp tutorial](https://approvaltestscpp.readthedocs.io/en/latest/generated_docs/Tutorial.html) describes comparing received output to a checked-in approved result, while [ApprovalTests.Net](https://github.com/approvals/ApprovalTests.Net) provides the mature repository implementation. Consumer-driven contract systems such as [Pact](https://docs.pact.io/getting_started/how_pact_works) already serialize expected interactions into executable artifacts and replay them against providers under a versioned [Pact specification](https://docs.pact.io/implementation_guides/pact_specification).

No one item on that list is the proposed product. Collectively, however, they establish nearly every abstract arrow in “implementations disagree → human rules → executable artifact persists.” Countershape’s defensible claim is that it makes this loop work across real candidate trees and real runners with reproducibility metadata—not that the arrow itself did not exist.

**Confidence: 9/10.** The risk is not inability to build; it is overclaiming composition as a new primitive.

### P1 — Repository candidates and differentiating execution are active territory

[xTestCluster](https://arxiv.org/abs/2207.11082) clusters 902 plausible automated-program-repair patches using generated tests and dynamic behavior, then uses differentiating tests to reduce patch-review burden. [DiffSpec](https://arxiv.org/abs/2410.04249) generates differential tests from specifications and code artifacts and applies them to real eBPF and WebAssembly implementations. The [Agentic Verifier](https://openreview.net/forum?id=8bHa5wc7X5) actively searches for highly discriminative inputs across candidate competitive-programming solutions to improve selection. [Agent-CoEvo](https://arxiv.org/abs/2604.04580) co-evolves repository-level code and test populations, executes a code-by-test matrix in isolated environments, and treats tests as dynamic behavioral constraints. [R2E](https://github.com/r2e-project/r2e) turns arbitrary GitHub repositories into executable Docker environments and builds equivalence harnesses with setup, files, and database state; its broader repository-execution contribution is documented in the [2026 Berkeley dissertation record](https://www2.eecs.berkeley.edu/Pubs/TechRpts/2026/EECS-2026-158.html).

These works invalidate novelty claims around patch clustering, discriminating-input discovery, execution matrices, containerized repository execution, or differentiating tests in isolation. The remaining gap is that they generally use a known implementation, test consensus, functional consistency, or automated fitness as the oracle. Countershape deliberately stops where those systems rank or select, exposes the stable partition to a human, and preserves a scoped ruling.

**Confidence: 9/10.** Repository scale is a differentiator only as part of the full lifecycle, not by itself.

### P1 — Reduction, replay, flake handling, and contracts are substrates, not inventions

[DDMT](https://arxiv.org/abs/2607.00929) combines delta debugging with metamorphic testing to minimize failures without a conventional expected-output oracle. [fast-check](https://fast-check.dev/docs/introduction/) generates and shrinks property-based counterexamples; its [model-based testing documentation](https://fast-check.dev/docs/advanced/model-based-testing/) covers command sequences and replay paths. [Hypothesis](https://hypothesis.readthedocs.io/en/latest/tutorial/replaying-failures.html) persists and replays failures and explicitly documents [flaky or unstable execution effects](https://hypothesis.readthedocs.io/en/latest/explanation/example-count.html). [Schemathesis](https://schemathesis.readthedocs.io/en/latest/explanations/data-generation/) applies generation and shrinking to API schemas. [libFuzzer](https://llvm.org/docs/LibFuzzer.html) maintains and minimizes a reproducer corpus.

Countershape can legitimately define a specialized preservation predicate: a reduction is accepted only if repeated trials retain the same declared candidate partition under the same runner, world, and normalizer fingerprints. It cannot call the reduction algorithm new merely because its predicate is product-specific. “Smallest behavior” is also too strong: common shrinkers are local and strategy-dependent, and environmental nondeterminism makes global minimality especially implausible.

Similarly, the standalone contract is a valuable product invariant but not a novel artifact class. Approval tests, Pact files, generated tests, and user-approved TiCoder tests all establish durable executable residue. The differentiated requirement is narrower: generated residue should encode a human-scoped predicate for one minimized witnessed divergence and run without Countershape.

**Confidence: 9/10.** Treat all of these as credited foundations.

### P1 — Candidate-agent and branch tools define a bright product boundary

[Parallel Code](https://github.com/johannesjo/parallel-code) launches multiple agents in Git worktrees, presents diffs and an “AI Arena,” and lets users merge a winning branch. [Automagik Forge](https://github.com/automagik-dev/forge) similarly manages agent attempts in worktrees and supports side-by-side comparison and best-attempt selection. [Switchman](https://switchman.dev/) analyzes branches for semantic overlap and interface drift and returns merge-confidence categories. [Pact Tools](https://pact.tools/docs/) decomposes contract-first work, coordinates agents, and selects successful implementations against tests and runtime evidence.

Countershape remains differentiated only if it refuses their ownership surfaces. It should consume exact Git references produced anywhere, but not spawn agents, manage worktrees, plan tasks, rank coding agents, choose a “winner,” estimate merge safety, or merge code. It asks a different question: “What behavioral contract should survive regardless of which implementation is chosen?”

**Confidence: 9/10.** Expanding into orchestration would blur the concept and collide with both current tools and Wake.

### P1 — Wake is not a collision, but integration language can create one

The audited [Wake repository](https://github.com/nelsonwerd/wake) owns the local agent runtime: process supervision, adapters, event history, crash recovery, replay/fork semantics, effect records, and a run viewer. Countershape’s proposed core owns candidate behavioral execution, observation normalization, stable partitioning, reduction, human semantic rulings, and contract export. Those centers of gravity are distinct.

The safe relationship is one-way and optional:

- Wake, another agent runner, or a human may produce candidate Git references and provenance.
- Countershape receives those immutable references as inputs.
- Countershape does not interpret or replay Wake’s event history and does not require Wake to execute contracts.

Do not add agent launching, attempt dashboards, run replay, crash resume, effect ledgers, agent adapters, or branch lifecycle management to make Countershape “more complete.” Even a friendlier UI over those concepts would be a Wake application, not this product. A small adapter that imports candidate refs plus source provenance is appropriate; an orchestration layer is not.

**Confidence: 10/10.** The current concept can avoid duplicating Wake with disciplined scope.

### P2 — Patent landscaping reinforces caution, but is not a legal conclusion

A limited primary keyword search found [US11513773B2, “Feedback-driven semi-supervised synthesis of program transformations”](https://patents.google.com/patent/US11513773), which describes selecting fruitful disambiguation inputs and incorporating direct, implicit, or automatic user feedback into continuing synthesis. Broader candidate-program/specification patents include [US11132180B2](https://patents.google.com/patent/US11132180B2/en) and [US11934801B2](https://patents.google.com/patent/US11934801B2/en). These are not asserted as blocking patents, and this report does not interpret their claims. They simply reinforce that candidate disambiguation and user feedback are established terrain. A production company would need counsel-led claim analysis and a wider jurisdictional search.

**Confidence: 7/10** that this is a useful caution; **0/10** as an FTO opinion, because none was attempted.

## Evidence matrix

| Work / primary source | What it establishes | Claim it invalidates or narrows | Remaining Countershape boundary |
|---|---|---|---|
| [Socrates / Choose, Don’t Label](https://arxiv.org/abs/2604.08792) | Candidate programs → discriminating region → semantic clusters → human multiple choice | New human operation; new behavior-choice primitive; first N-way intent partition | Empirical repository runners, environment/stability provenance, reducer trace, standalone scoped contract |
| [TiCoder](https://arxiv.org/abs/2208.05950) | Human labels candidate-generated tests; feedback prunes code; accepted tests persist | First feedback-to-regression-test loop | Real Git trees, black-box state, N-way observations, declared world |
| [ARHF](https://arxiv.org/abs/2508.14114) | Asks desired behavior for ambiguous inputs | First ambiguous-input clarification for code | Repository/runtime operationalization and durable contract |
| [FlashProg interaction models](https://www.microsoft.com/en-us/research/publication/user-interaction-models-for-disambiguation-in-programming-by-example/) | Active query where candidate programs disagree; user selects | First implementation-derived disambiguation input | Non-PBE, empirical stateful systems, provenance and residue |
| [Synthesis with Abstract Examples](https://www.sri.inf.ethz.ch/publications/drachsler2017synthesis) | User accepts a generalized behavior region or gives counterexample | First scoped/generalized human behavior judgment | Concrete repo workflow and contract generation |
| [SpecFix](https://arxiv.org/abs/2505.07270) / [artifact](https://github.com/msv-lab/SpecFix) | Samples programs, clusters behavior, diagnoses and repairs ambiguous NL specs | Implementations as a new sensor for underspecification | Human ruling; exact Git candidates; controlled worlds; standalone test |
| [xTestCluster](https://arxiv.org/abs/2207.11082) | Generated tests dynamically cluster plausible patches and distinguish them | First patch clustering or reviewer-facing differentiating tests | No automated correctness ranking; semantic adjudication and contract residue |
| [DiffSpec](https://arxiv.org/abs/2410.04249) | Differential tests distinguish real eBPF/Wasm implementations | First real-system differentiating input | Human intent, stable world identity, reduction transcript, scoped contract |
| [Agentic Verifier](https://openreview.net/forum?id=8bHa5wc7X5) | Searches discriminative inputs across candidate solutions | First active discriminating-input search | Human semantic choice instead of candidate ranking |
| [Agent-CoEvo](https://arxiv.org/abs/2604.04580) | Repository code/test populations, execution matrix, isolated runs | First repo-level candidate × test execution | Human oracle and durable witnessed contract |
| [R2E](https://github.com/r2e-project/r2e) | Executable repo environments and stateful equivalence harnesses | Repository harness/environment novelty | Competing candidates with no trusted original; human adjudication |
| [DDMT](https://arxiv.org/abs/2607.00929) | Oracle-light minimization via metamorphic preservation | Oracle-deficient reduction novelty | Partition-preservation predicate plus full provenance |
| [fast-check](https://fast-check.dev/docs/introduction/) / [Hypothesis](https://hypothesis.readthedocs.io/en/latest/tutorial/replaying-failures.html) | Generation, shrinking, replay, stateful sequences, flake awareness | Generic shrink/replay novelty; global “smallest” wording | Named local reducers and reproducible multi-candidate predicate |
| [PatchFusion](https://arxiv.org/abs/2607.01597) | Deterministic evidence fusion over candidate patch pools | Candidate-pool evidence aggregation novelty | It changes/selects code; Countershape records human semantics |
| [Dynamic–static selection](https://ojs.aaai.org/index.php/AAAI/article/view/40481) | Static filtering plus dynamic functional consistency selects generated code | Mixed evidence for candidate selection novelty | Countershape refuses correctness selection |
| [Diffy](https://blog.x.com/engineering/en_us/a/2015/diffy-testing-services-without-writing-tests) | Same requests across service versions; learns noisy differences from known-good replicas | Service differential execution and noise-handling novelty | No known-good baseline; human judgment of a stable partition |
| [Parallel Code](https://github.com/johannesjo/parallel-code) | Multi-agent worktrees, diff arena, merge winner | Agent arena / branch management novelty | Candidate refs are immutable inputs, not managed attempts |
| [Automagik Forge](https://github.com/automagik-dev/forge) | Agent attempts and side-by-side best-attempt workflow | Friendlier multi-attempt agent dashboard novelty | Behavioral contract decision after candidate production |
| [Switchman](https://switchman.dev/) | Branch overlap, drift, and merge-confidence analysis | Merge-risk or semantic-branch scoring novelty | Concrete executed divergence, no merge-safety claim |
| [Pact Tools](https://pact.tools/docs/) | Contract-first agent planning and competitive implementation | Contract-first competitive-agent novelty | Post-implementation discovery; does not orchestrate agents |
| [Pact specification](https://docs.pact.io/implementation_guides/pact_specification) | Serialized executable interactions replayed across providers | Executable contract artifact novelty | Human-selected predicate induced from candidate disagreement |
| [VibeContract](https://arxiv.org/abs/2603.15691) | Intent → validated contracts → code/tests/runtime traceability | Broad “intent contract” category novelty | Reverse arrow: observed competing implementations → scoped contract |
| [ApprovalTests](https://approvaltests.com/) | Human-approved output becomes checked-in regression baseline | Human approval-to-durable-test novelty | N-way candidate partition and narrow predicate rather than snapshot acceptance |
| [Wake](https://github.com/nelsonwerd/wake) | Local agent supervision, replay/fork, effects, event history | Any agent-runtime or attempt-viewer expansion | Optional candidate/provenance producer only |

## What is established composition, and what may still be differentiated

### Established ingredients

The build should assume the following are public foundations, not claims:

- differential execution across candidate versions;
- generated inputs that separate implementations;
- dynamic clustering of patches or programs by observed behavior;
- candidate-program disambiguation through human questions;
- multiple-choice semantic behavior clusters;
- active learning to choose informative questions;
- treating implementation diversity as evidence of specification ambiguity;
- property-based generation, shrinking, replay, and state-machine commands;
- delta-debugging under a custom preservation predicate;
- repeated execution and noise filtering;
- human approval of expected outputs;
- executable contract serialization and conformance replay;
- repository isolation in containers; and
- multi-agent branch production, comparison, and winner selection.

### The potentially differentiated system contract

The strongest defensible object is not “a new kind of question.” It is a **reproducible decision record whose identity binds code, environment, observation semantics, evidence, and executable residue**.

A Countershape Choicepoint can still be a useful named record format if it binds:

- exact candidate tree identifiers and dirty-tree rejection policy;
- runner and world content digests, including declared external dependencies;
- normalizer identity and a retained raw result for audit;
- trial counts, per-candidate observations, instability classification, and refusal rules;
- a stable N-way equivalence partition rather than a winner score;
- the initial stimulus, every reducer step, and the final 1-minimal case under named reducers;
- the human actor, decision vocabulary, scope, rationale, and explicit uncertainty;
- the generated predicate and the standalone test’s content digest; and
- later conformance results without rewriting the historical observation.

That record is valuable because it makes a semantic decision inspectable across time and tools. Calling it “Countershape’s durable record format” or “a repository artifact” is defensible. Calling it a new semantic primitive is not.

## Claim language: allowed, qualified, and prohibited

### Strong language that the evidence supports

- “Countershape brings interactive program disambiguation to real Git candidate trees.”
- “Compare exact candidate trees in one declared world, reduce a stable observed split to a reviewable case, and preserve the ruling as a standalone repository test.”
- “A repository-scale operationalization of differential testing, interactive disambiguation, reduction, and contract generation.”
- “An empirically observed partition under named candidate, world, runner, and normalizer fingerprints.”
- “Locally 1-minimal under the recorded reducer set and preservation predicate.”
- “A human-scoped ruling: allow one, allow several, reject all, defer, or refine.”
- “The exported test is standalone; the evidence ledger remains auditable.”
- “Distinct from Wake: Wake runs and replays agents; Countershape consumes candidate refs and records behavioral decisions.”
- “In this public-source search, we did not find one tool combining exact repository candidates, empirical world identity, stability-aware partition reduction, plural human rulings, and standalone contract residue.”

### Language that requires an immediate qualifier

- “Deterministic” must refer to serialization, fingerprints, or replay mechanics—not an unconstrained external world.
- “Minimal” must be “1-minimal under named reducers,” never globally smallest.
- “Equivalent” must mean equivalent under the declared normalizer for the recorded stimulus and trials, not semantically equivalent programs.
- “Contract” must mean a scoped executable predicate, not a complete specification.
- “Reproducible” must name the captured boundary and disclose undeclared external dependencies.
- “Asks what you meant” is acceptable product copy only if the technical narrative credits prior interactive-disambiguation work and does not frame the interaction as invented here.

### Claims to prohibit

- invented program or intent disambiguation;
- first to use implementation diversity to expose ambiguity;
- first to ask a human to choose among candidate behaviors;
- first to turn such feedback into tests or contracts;
- “resolve a Choicepoint” as a newly invented human operation;
- new algorithm, new testing paradigm, or oracle-free verification;
- finds “the smallest behavior” without the local-reducer qualification;
- proves semantic equivalence, correctness, safety, compatibility, or mergeability;
- identifies the best implementation or the winner;
- eliminates ambiguity in a whole specification from one witness;
- “deterministic world” when network, clock, scheduler, model, or other external state remains uncontrolled; and
- a novel “contract compiler” category claim without explicitly narrowing it to candidate-induced, witnessed repository contracts.

## Required changes to the concept brief

1. **Replace the novelty thesis.** Remove “new human operation,” “new primitive,” and any implication that candidate behavior choice is unprecedented. Define Choicepoint as Countershape’s durable, provenance-bound decision record.
2. **Rewrite the one-line promise.** Replace “finds the smallest behavior that makes them disagree” with “locally reduces a stable observed disagreement under named rules.” A concise candidate is: **“Countershape compares exact candidate trees in one declared world, reduces a stable behavioral split to a reviewable case, and preserves the human ruling as a standalone repository test.”**
3. **Add an explicit prior-art lineage.** Credit Socrates, TiCoder, FlashProg/interactive PBE, SpecFix, xTestCluster, differential testing, property-based shrinking, approval testing, and Pact-style executable contracts. Explain what Countershape operationalizes beyond each.
4. **Make provenance load-bearing in the MVP.** Candidate, runner, world, and normalizer digests; raw observations; trial evidence; stability status; and reducer transcript cannot be postponed as polish. Without them, the remaining differentiation disappears.
5. **Keep plural rulings.** “Allow one” alone degenerates into winner selection. Allow-many, reject-all, defer, and refine/scoped-predicate behavior are central to the non-ranking philosophy and should have first-class tests.
6. **Define exact refusal semantics.** The system must refuse to adjudicate an unstable split, a dirty or moving candidate reference, a changed normalizer, an unidentifiable world, or a reduction that cannot reproduce the original partition. Refusal is part of correctness.
7. **Narrow the contract guarantee.** The output contract protects the chosen predicate for the witnessed stimulus class; it does not prove full behavioral equivalence or complete intent.
8. **Keep Wake and agent orchestration out.** Support a tiny provenance import interface, including Wake-produced candidate refs, but do not own processes, worktrees, agent sessions, effects, replay, recovery, or merging.
9. **Add a composition kill gate.** Before calling the first build concept-complete, demonstrate one stateful or environment-sensitive case where fingerprints, repeated trials, raw-versus-normalized evidence, and the reduction transcript materially change the decision. If all demos are pure fixture comparisons, stop or rescope.
10. **Add a novelty claim test to release review.** Every public claim should be classified as observed capability, product boundary, or novelty assertion. Any novelty assertion must cite the specific closest work it improves upon.

## Ground-truth tally

This audit reviewed **24 named primary-source families** in the evidence matrix:

- **5 direct interactive-disambiguation precedents:** Socrates, TiCoder, ARHF, FlashProg interaction models, and Synthesis with Abstract Examples;
- **6 candidate/differential execution precedents:** SpecFix, xTestCluster, DiffSpec, Agentic Verifier, Agent-CoEvo, and R2E;
- **5 testing/reduction/selection substrates:** DDMT, fast-check/Hypothesis, PatchFusion, dynamic–static selection, and Diffy;
- **4 contract/intent precedents:** Pact, Pact Tools, VibeContract, and ApprovalTests; and
- **4 product-boundary references:** Parallel Code, Automagik Forge, Switchman, and Wake.

Of those, **five** directly invalidate broad semantic novelty claims, **fifteen** establish major ingredients that must be described as composition, and **four** define adjacent product territory the build should avoid. The patent search adds caution but is excluded from that tally because it was not a claim-level legal analysis.

No searched source was found to expose the exact complete Countershape lifecycle as a coherent open-source developer product. Absence from this bounded search is not proof of global novelty. It supports qualified differentiation language only.

## Confidence ledger

| Finding | Confidence | Why |
|---|---:|---|
| Candidate-program disagreement → human behavior choice is established | 10/10 | Socrates is an exact, peer-reviewed semantic collision, reinforced by older interactive synthesis |
| Feedback → reusable tests/contracts is established | 9/10 | TiCoder, approval testing, and Pact cover the relevant arrows in different settings |
| Repository execution and discriminating tests are established | 9/10 | xTestCluster, DiffSpec, Agent-CoEvo, and R2E provide direct evidence |
| Generic reduction/replay/stability ingredients are established | 9/10 | Mature property/fuzz tooling and DDMT establish the substrate |
| The exact eight-part lifecycle was not found in one public tool | 6/10 | Broad targeted search supports it, but non-discovery cannot prove novelty |
| Countershape is distinct from Wake under the proposed boundary | 10/10 | Their state ownership, primary artifacts, and user decisions are materially different |
| The concept can become a compelling reference-quality system | 7/10 | Architecture has a real integration burden, but value still depends on proving a nontrivial repository case |
| Any patent freedom-to-operate conclusion | 0/10 | Outside the method and scope of this report |

## Final recommendation

**Modify and proceed.** Do not kill the build merely because its ingredients have precedent; reference-quality infrastructure often matters because it makes fragmented research and testing ideas usable together. Do kill the “new Choicepoint primitive” thesis. The honest ambitious thesis is:

> Countershape operationalizes interactive program disambiguation for exact repository candidates, binding empirical evidence and human scope to a standalone contract that survives the tool.

The project earns its ambition only if the repository/runtime boundary is real. Its decisive demo should include state, environmental identity, an unstable first observation that becomes classifiable through controlled trials, a normalization choice visible beside raw evidence, a partition-preserving reduction, an allow-many or reject-all human ruling, and an exported test that passes without Countershape. That would be meaningfully more than a test-library mashup, while remaining honest about the deep lineage it builds on.
