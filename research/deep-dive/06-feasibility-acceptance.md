# Countershape feasibility, architecture cut, and acceptance gates

- **Lane:** implementation feasibility, package boundaries, performance, test strategy, and build sequencing
- **Date:** 2026-07-14
- **Verdict:** **MODIFY, THEN PROCEED**
- **Overall confidence:** **7/10** that the bounded reference system can be built; **4/10** that the current fifteen-minute and arbitrary-repository language can coexist with the required fresh-world semantics

## Executive ruling

Countershape is buildable as a reference-quality local system if the implementation is treated as a small experimental operating environment with an unusually strict evidence model, not as a general-purpose repository test runner. A Go core plus a TypeScript/React/Vite studio is an appropriate split. Go is well suited to deterministic typed state, Git plumbing, byte-bounded process I/O, POSIX lifecycle control, a loopback API, and single-binary delivery. React is appropriate for the decision bench because the interface must coordinate several evidence views and nontrivial states. Node remains only a build dependency for the studio, a measured fixture runtime, and the first standalone contract target; it is not the authority for Countershape state.

The architecture should share only the truth algebra across domains. CLI and HTTP must keep separate typed stimuli, captured channels, projections, and reducer neighborhoods. Forcing them through one generic JSON-shaped adapter would make the code superficially elegant and semantically dishonest. The reusable kernel is narrower: artifact identity, world/trial control, repeat classification, exact projection fingerprints, candidate-to-outcome maps, tri-valued reduction orchestration, immutable Choicepoint lineage, and deterministic compilation.

Four modifications are required before build:

1. Split the fifteen-minute success metric. It is a measured target for the two self-contained reference studies, not a promise for arbitrary 5k-100k LOC repositories with dependency installation. Imported repositories receive explicit wall/trial/byte budgets and may end at BEST_KNOWN or UNCOMPARABLE.
2. Make a content-addressed artifact graph, not a database event log, the persistence model. This preserves immutable evidence without recreating Wake's execution/replay center.
3. Narrow the first executable witness to one CLI invocation or one HTTP request after a declarative fixture seed. Arbitrary action sequences, database cloning, browser journeys, user code normalizers, plugins, and reusable prepared worlds are cuts.
4. Put the canonical model and hostile vectors ahead of process execution. If identity, projection equality, outcome-map preservation, state transitions, and compiler field selection are not proven first, a polished runner will only produce persuasive false evidence faster.

With those changes, the project remains intentionally heavy: it contains a strict compiler, filter-free Git materializer, controlled local runner, multi-candidate observation engine, reducer, immutable decision state, code generator, secure local API, responsive studio, report exporter, and two real end-to-end studies. The cut removes breadth, not the load-bearing system.

## Severity-ranked findings

| Severity | Finding | Consequence | Required disposition |
| --- | --- | --- | --- |
| **P0** | Fresh materialization/setup for every evidentiary trial conflicts with a universal fifteen-minute promise. | Either the product reuses state and weakens its evidence, or routine repositories exceed the advertised budget. | Keep fresh worlds; scope the SLA to the two reference studies; report imported-repo budgets honestly. |
| **P0** | A generic adapter/projection/reducer interface can erase domain semantics. | CLI absence, signals, stderr, HTTP headers, JSON, transport failure, and fixture state collapse into one lossy blob. | Share orchestration interfaces only; keep domain models and reducers typed. |
| **P0** | Go's standard JSON decoder is not a canonical identity layer. | Duplicate keys, numeric coercion, negative zero, unsafe integers, and last-write-wins parsing can change digests or comparisons. | Build a strict parser plus one project-owned canonical encoder and vectors before persistence. |
| **P0** | REJECT_ALL cannot produce the required none-conforms executable artifact. | An always-failing generated test would encode dissatisfaction, not intended behavior. | Use CUSTOM_EXPECTATION for none-conforms; REJECT_ALL emits a decision record only. |
| **P1** | A mutable event database would overlap Wake and complicate authority. | Countershape drifts into run-history/recovery infrastructure and creates dual truth. | Use immutable content-addressed artifacts plus small atomic head pointers; no replay/resume runtime. |
| **P1** | Standalone tests can accidentally depend on Countershape or over-assert observed fields. | The residue is brittle, nonportable, or silently stronger than the ruling. | Emit deterministic node:test files from selected typed predicates; run with Countershape absent. |
| **P1** | POSIX process-group behavior is platform-sensitive and not containment. | Cross-compile may be mistaken for runtime proof; leaked children contaminate later worlds. | Build-tag Darwin/Linux lifecycle code; test both at runtime; retain trusted-local warning and ORPHAN_RISK. |
| **P1** | Reduction cost multiplies candidates, repeats, fresh setup, and final confirmation. | An apparently small neighborhood can consume hundreds of executions. | Compile hard trial/wall/shrink budgets into the plan; use BEST_KNOWN when exhausted; show the arithmetic. |
| **P1** | A local studio is a write-capable application, not a static viewer. | Host/Origin/CSRF or stale-tab failures can forge rulings. | Loopback-only bearer session, exact Host/Origin, no CORS, CSRF, and digest compare-and-swap. |
| **P2** | Vite introduces a second toolchain into an otherwise single-binary product. | Reproducible release builds and dependency maintenance become real work. | Pin Node/package manager/lockfile; embed built assets; keep runtime free of Node except emitted-test/fixture needs. |
| **P2** | General plugin or browser support would consume the verification budget. | Core correctness receives less adversarial testing while scope overlaps adjacent systems. | Do not create the protocol in this build; document extension seams only. |

## Recommended repository architecture

The repository should be a Go module with a deliberately thin web workspace:

~~~
cmd/countershape/                 CLI composition root
internal/domain/                  immutable value objects, sums, invariants
internal/canon/                   strict parser, canonical bytes, domain digests
internal/spec/                    inert source schema -> compiled StudyPlan
internal/gitobj/                  ref resolution and filter-free materialization
internal/world/                   fresh-world orchestration and lifecycle
internal/world/proc_darwin.go     Darwin process-group operations
internal/world/proc_linux.go      Linux process-group operations
internal/adapters/cli/            CLI stimulus, capture, projection, neighbors
internal/adapters/http/           HTTP stimulus, capture, projection, neighbors
internal/observe/                 repeat schedule, trial eligibility, classifier
internal/compare/                 exact fingerprints and labeled outcome map
internal/reduce/                  bounded tri-valued search and final sweep
internal/choice/                  lineage transitions and ruling validation
internal/emit/node/               deterministic standalone node:test emitters
internal/store/                   content-addressed artifacts and atomic heads
internal/server/                  authenticated loopback JSON/SSE API
internal/report/                  redacted self-contained report export
web/                              React/Vite decision bench
testkit/repos/                    generated Git repositories and hostile fixtures
testkit/contracts/                standalone emitted-test target fixtures
docs/                             contracts, threat model, operator workflows
~~~

Package direction must be acyclic. Domain and canon import only the standard library. Spec may depend on domain/canon. Git object and adapter packages implement narrow ports owned by application services. Observe depends on domain and a world-execution interface, not concrete adapters. Compare is pure. Reduce receives a typed neighbor provider and an evaluator; it must not know Git, processes, or HTTP. Choice consumes immutable artifact digests and validated domain values. Emitters consume a compiled ruling view, never the mutable study. Server is a delivery layer and cannot mutate the store except through Choice/application commands. The React bundle speaks only documented API DTOs; it never recomputes truth classifications in JavaScript.

Avoid a public Go plugin API in v1. The useful seams can remain internal interfaces:

- Materializer resolves one exact Candidate into one WorldInstance root.
- Adapter encodes a typed stimulus, runs one probe against a ready world, projects a CapturedObservation, and enumerates strictly smaller typed neighbors.
- TrialExecutor returns control evidence and captured channel digests.
- Evaluator returns PRESERVES, CHANGES, or UNRESOLVED.
- Emitter maps one exact ruling to sorted relative path/mode/bytes.

These interfaces are test seams, not an extension promise. A protocol without a real third adapter would be speculative maintenance surface.

## Persistence without a Wake collision

Countershape needs durable provenance but does not need an execution event store. Use a local content-addressed artifact graph:

~~~
.countershape/
  objects/sha256/ab/<digest>      canonical immutable bytes
  studies/<study-id>/head.json    current lineage digest and revision
  studies/<study-id>/private/     mode-0600 captured byte artifacts
  exports/                        generated decision/test/report outputs
~~~

Each object is written to an owned temporary file, flushed, atomically renamed, and then treated as immutable. A head update includes expected prior revision and uses compare-and-swap semantics. The artifact itself links predecessor and semantic ancestors by digest. Superseding a plan creates a new branch of the artifact graph and marks the old derived head stale; it never rewrites old evidence.

This is intentionally not an append-only command/event ledger, process replay engine, crash-resume protocol, fleet history, or causal viewer. Transient progress may stream to the studio over server-sent events, but those notifications are disposable and non-authoritative. After a crash, completed immutable objects remain inspectable; an in-flight attempt becomes PARTIAL and is rerun from a fresh world. Countershape does not resume a process or reproduce Wake's runtime semantics.

## Canonical model and identity risks

The Go standard library is safe for transport after validation, but encoding/json must not define identity. It accepts duplicate object names with last-value behavior and normally decodes numbers through floating point when targeting interface values. Map iteration and HTML escaping choices also make ambient serializer behavior the wrong artifact contract.

R0 should therefore implement one restricted canonical value algebra:

- null, Boolean, UTF-8 string without lone surrogate;
- safe-range signed integer for Countershape specifications;
- externally observed decimal as a tagged canonical decimal string, parsed from sign/coefficient/base-ten exponent without binary floating-point;
- byte string as an explicit base64-tagged channel value;
- array retaining order; and
- object with unique UTF-8 keys sorted by raw Unicode scalar sequence under the documented encoder.

Reject negative zero, duplicate keys, non-finite values, unsafe internal integers, invalid UTF-8, and unsupported numeric forms before a typed object exists. Do not normalize Unicode. Domain-separated SHA-256 covers kind plus canonical bytes. Candidate IDs include repository fingerprint, Git object format, commit/tree IDs, portable materialized-tree digest, and materializer policy. Instance allocations such as port and path never enter the deterministic WorldPlan digest; they live in WorldInstance evidence and pass through a named comparability function.

Observed HTTP JSON receives an explicit projection policy. It may either preserve body bytes exactly or parse through the strict external JSON algebra. A rejected body is projection rejection, not empty JSON. Built-in normalizer operations are finite and visible: select status, selected headers under declared multimap rules, selected JSON paths, exit status, stdout/stderr channels, remove one declared scratch-root placeholder, and optionally normalize one declared line-ending policy. There is no regex code, arbitrary JavaScript, fuzzy comparison, tolerance, or hidden date stripping in v1.

Every digest needs golden vectors consumed by both Go and TypeScript DTO tests. JavaScript must display precomputed identifiers and may validate DTO shape; it must not independently produce canonical artifact digests. This avoids a second canonical implementation becoming accidental authority.

## State machines

Three related state machines should be explicit rather than one oversized status enum.

### Artifact lineage

~~~
SOURCE_SPEC
  -> COMPILED_PLAN
  -> MATERIALIZED_CANDIDATE_SET
  -> BASELINE_OBSERVATION
  -> DIVERGENCE
  -> REDUCTION_RESULT
  -> FRESH_CONFIRMATION
  -> CHOICEPOINT_READY
  -> RULING
  -> CONTRACT_SET
  -> CONFORMANCE_REPLAY
~~~

Each arrow creates a new immutable artifact. CANCELLED and PARTIAL retain eligible prior artifacts but do not advance. Changing candidate set, plan, capture policy, projection, comparator, repeat policy, reducer set, ruling, or emitter makes descendants stale. Ruling is legal only from a freshly confirmed witness. CONTRACT_SET is legal only for ALLOW_OBSERVED or CUSTOM_EXPECTATION. REJECT_ALL, DEFER, and REFINE stop without executable output; REFINE creates a new source-spec lineage.

### One execution attempt

~~~
ALLOCATED -> MATERIALIZING -> SETTING_UP -> STARTING
          -> READY -> PROBING -> CAPTURING -> TEARING_DOWN -> FINALIZED
~~~

Every phase can terminate in a typed control state. Only FINALIZED with complete capture and successful teardown is eligible for projection and stable classification. Teardown failure becomes UNCOMPARABLE in v1. Timeout and cancellation initiate TERM/grace/KILL/group-probe and retain bounded partial byte digests. No result becomes complete merely because the direct child exited.

### Reduction

~~~
BASELINE_CONFIRMED
  -> SEARCHING
  -> BEST_CANDIDATE
  -> EXHAUSTIVE_DIRECT_NEIGHBOR_SWEEP
  -> UNCHANGED | BEST_KNOWN | ONE_MINIMAL_UNDER
~~~

Each candidate evaluation is tri-valued. Any UNRESOLVED direct neighbor, exhausted trial budget, cancelled sweep, or ineligible candidate forbids ONE_MINIMAL_UNDER. Search order and accepted transcript are recorded. The preservation predicate is exact equality of the candidate-execution-ID to canonical-projection-fingerprint map, not merely equal cluster membership.

## Execution and adapter cut

The golden path supports Darwin and Linux and explicitly rejects Windows at build/run time. It invokes Git as an argv-based subprocess for ls-tree and cat-file plumbing; writing a Git object parser is unnecessary risk. Materialization accepts regular/executable blobs and constrained internal symlinks, rejects gitlinks/LFS/path collisions, verifies the result, and never provides .git.

World execution is sequential by default. Each evidentiary trial gets a fresh materialization, private HOME/TMP/XDG/state roots, sparse environment, loopback port, and fresh explicit setup. Network is labeled HOST_ALLOWED. Commands use exec.Cmd with no shell and platform build-tag process-group helpers. A Context cancellation is only the trigger; the lifecycle owner still performs TERM, bounded drain, KILL, wait, and group probe.

The CLI adapter supports one argv vector, optional stdin bytes or strict JSON, declared environment bindings, fixture files, and the captured tuple of exit/signal/stdout/stderr. The HTTP adapter supports one fixture seed plus one method/path/ordered query/header multimap/body request to a freshly started service, explicit readiness, and the captured tuple of status/header multimap/body. HTTP transport/readiness errors stay in TrialControl. This is sufficient for authorization and configuration-precedence proofs while avoiding a general workflow language.

Stateful multi-request action sequences are cut. The authorization demo may establish users/tenants/resources through declarative fixture files or setup argv, then issue one cross-tenant request. If the product cannot express that without a bespoke database integration, the demo is invalid and the concept should narrow rather than hide the setup in application-specific code.

## Performance, termination, and resource budgets

Every StudyPlan compiles explicit finite budgets. Recommended reference defaults are:

| Budget | Default | Hard reference ceiling |
| --- | ---: | ---: |
| candidates | 2-4 | 4 |
| discovery repeats per candidate/stimulus | 3 | 5 |
| fresh confirmation repeats | 3 | 5 |
| materialized entries | 25,000 | 100,000 |
| materialized bytes per world | 256 MiB | 1 GiB |
| single blob | 32 MiB | 128 MiB |
| setup | 15 s | 120 s |
| service readiness | 5 s | 30 s |
| one probe | 3 s | 30 s |
| teardown grace plus kill | 3 s | 10 s |
| stdout or stderr captured bytes | 1 MiB | 16 MiB |
| HTTP compressed/uncompressed body | 1 MiB | 16 MiB |
| shrink stimuli proposed | 40 | 200 |
| total candidate trials | 300 | 2,000 |
| shrink wall time | 10 min | 60 min |

The defaults are product policy, not security containment. Disk availability is checked before each fresh world, and the owned root is removed only after a nonce/containment check. HTTP decompression, if supported at all, applies both compressed and expanded byte caps. Output is hashed while streamed, with bounded head/tail previews.

The UI must expose the multiplication before run. Four candidates, three repeats, and forty attempted shrink stimuli already permit 480 shrink trials, before the baseline, fresh confirmation, accepted-step rechecks, or final sweep. The compiled total-trial cap is therefore authoritative even when the stimulus cap has not been reached. The engine stops at whichever bound is hit first. Budget exhaustion is an ordinary result with consumed/remaining counts, never a failed hidden retry.

The reference fifteen-minute metric applies only to the checked-in HTTP and CLI studies on a documented baseline machine, with no dependency installation and small source trees. A real imported repository gets no latency guarantee until measured. Prepared roots, dependency caches, reflink snapshots, and reset hooks are deferred optimization modes because they change the world model. If fresh worlds make real studies unusable, that is a product-learning result, not permission to silently reuse state.

## Test and verification pyramid

### Level 0: schema vectors and pure unit tests

Run on every change. Cover strict parsing, canonical bytes, domain separation, candidate ordering, Plan/Instance separation, every tagged sum, transition guards, selected-field compilation, and deterministic path sorting. Golden vectors include duplicate keys, negative zero, unsafe integers, invalid UTF-8, Unicode composition pairs, absent versus empty, truncation, and Git SHA-1/SHA-256 tags.

### Level 1: property, fuzz-seed, and model tests

Use Go property tests over generated canonical values, candidate permutations, projection partitions, typed reducer neighborhoods, and state-machine command sequences. Every neighbor must be valid and strictly decrease the well-founded measure. Replaying a transcript reconstructs proposals, while execution remains fresh. Go fuzz corpora are checked in; ordinary CI runs the seeds, while bounded scheduled fuzzing extends them.

### Level 2: mutation tests

Mutation is load-bearing for the truth kernel, not an optional coverage number. Pin one Go mutation tool or a checked-in deterministic mutation runner and target canon, compare, reduce, choice, and emit. Required killed mutants include: removed domain separator; duplicate-key last wins; missing equals empty; pairwise epsilon equality; candidate input order changes cluster IDs; partition-shape-only preservation; UNRESOLVED treated as CHANGES; final sweep skipped; timestamp/candidate name/unselected field injected into generated tests; and REJECT_ALL permitted to emit.

The build should report survived mutants individually. Do not reduce the mutation set to make a gate green; either strengthen tests or explicitly cut the capability before its first claim.

### Level 3: component and hostile integration tests

Generate local Git repositories containing attributes, executable modes, symlinks, LFS pointers, gitlinks, path collisions, moving refs, and malicious-looking output. Exercise sparse environment, fresh roots, fixed-port theft, hangs, grandchild-held pipes, ignored TERM, output floods, invalid UTF-8, teardown failure, and alternating results. HTTP and CLI fixtures must classify every infrastructure state distinctly.

Run process lifecycle tests natively on Darwin and Linux. GOOS cross-compilation proves only compilation and is never a runtime receipt. If this build runs only on macOS, Linux remains UNRECEIPTED in the handoff.

### Level 4: end-to-end semantic studies

The CLI configuration-precedence study and HTTP cross-tenant authorization study each require three committed candidates. All candidate native tests pass first. Countershape must discover at least one missing behavioral decision, reduce it, show captured-to-projected evidence, create a ruling, emit a standalone node:test artifact, and fresh-replay all candidates. A deliberate flake becomes UNSTABLE. A same-partition/different-output trap is rejected. One fixture earns ONE_MINIMAL_UNDER; another may honestly stop at BEST_KNOWN. CUSTOM_EXPECTATION produces none conforms. ALLOW_OBSERVED with multiple outcomes is also exercised.

### Level 5: local API, UI, accessibility, and report

API tests cover bearer, Host, Origin, CORS absence, CSRF, stale digest CAS, CSP, path handling, and inert rendering. Playwright covers keyboard-only blind-first resolution, no-default field assertions, desktop and mobile layouts, all eight major states, captured/projection switching, original/minimized comparison, provenance reveal, reject/defer/refine, and stale-tab handling. Axe and Lighthouse are recorded, but neither substitutes for manual screen inspection and a different-model visual critic. Report tests inject closing script tags, terminal escapes, fake secrets, and very large content and prove the default export is escaped/redacted with local captured artifacts excluded.

### Level 6: deterministic and performance reproduction

Compile plans and contracts twice in clean paths and compare bytes. Run each reference study three clean times; Plan, Choicepoint schema/ruling, and contract bytes must be stable where the same human fixture ruling is supplied, while WorldInstance and observation evidence remain new and linked. Benchmark materialization, peak memory under output flood, baseline/reduction trial counts, and full fifteen-minute target on the declared reference machine.

Every load-bearing command in Levels 0-6 must be executed through didrun. A test log without a didrun receipt remains UNRECEIPTED even when it looks green.

## Acceptance matrix

| Capability | Required proof | Pass bar | If it fails |
| --- | --- | --- | --- |
| Canonical identity | vectors, properties, mutants | all vectors/properties pass; every required mutant killed | stop before execution code |
| Git materialization | hostile generated repos | supported blobs/modes round-trip; unsupported forms reject; no filters/hooks | remove imported-tree claim |
| Fresh world lifecycle | Darwin/Linux process fixtures | roots/env/ports differ; ordinary descendants killed; typed failures preserved | platform stays unsupported or runner stops |
| Stability classification | constant/alternating/timeout fixtures | exact OBSERVED_STABLE(k/k), UNSTABLE, UNCOMPARABLE/INCOMPLETE outputs | no divergence or reduction claims |
| Exact N-way compare | permutation properties and trap | candidate order invariant; complete labeled map preserved | kill Choicepoint readiness |
| Tri-valued reduction | HTTP/CLI e2e, unresolved neighbor, final sweep | smaller witness in both domains; unresolved yields BEST_KNOWN; one valid local-minimal receipt | ship observation-only prototype |
| Ruling semantics | state-machine/model tests | one/many/custom/reject/defer/refine legal edges exact | do not emit contracts |
| Standalone contract | byte determinism and isolated node:test | selected fields only; no Countershape import/PATH; one none-conforms case | keep decision JSON only |
| Secure local studio | API adversarial suite and browser flows | unauthorized/cross-origin/stale writes fail; blind-first decision states work at 1440/375 | CLI-only release |
| Evidence report | injection/redaction fixtures | self-contained, escaped, redacted by default, captured bytes excluded | no shareable report claim |
| Two-domain kernel | full studies | both use same truth/reduction/choice services without domain coercion | narrow product to one adapter |
| Fifteen-minute aha | timed clean reference runs | both checked-in studies finish under target on named machine | revise metric, never hide cache reuse |
| Cross-platform support | native receipts | Darwin and Linux runtime suites pass | name only the validated OS |

## Phased shippable units

Each unit is independently claimable, committed, sealed, and strict-verified before the next. A later failure creates a new correcting commit and receipt; it never rewrites an earlier claim.

1. **U0 — contracts and threat model.** Lock schemas, state machines, budgets, refusal semantics, package dependency rules, and fixture specifications. Validate documentation links and machine-readable examples.
2. **U1 — truth kernel.** Implement domain/canon/spec/compare/choice pure packages, vectors, properties, state-machine tests, and required truth-kernel mutants. No subprocess exists yet.
3. **U2 — source and process substrate.** Implement immutable ref resolution, filter-free materializer, private world allocation, byte-bounded process groups, teardown, and hostile fixtures. Claim Darwin only unless Linux is actually run.
4. **U3 — CLI observation spine.** Add CLI stimulus/capture/projection, repeat scheduler, labeled partitions, and the flake/control-state fixtures. Deliver a CLI-only baseline comparison before reduction.
5. **U4 — HTTP observation spine.** Add service start/readiness/request/capture/projection with one declarative seed plus one request. Prove the same observation algebra without forcing CLI types through HTTP.
6. **U5 — bounded reducer.** Add typed neighbor providers, tri-valued evaluator, budgets, transcripts, final neighbor sweep, the same-shape/different-output trap, and both reduction grades.
7. **U6 — Choicepoint and residue.** Add artifact store/head CAS, immutable lineage, plural rulings, deterministic node:test emitters, and fresh conformance replay. Prove absent Countershape dependency.
8. **U7 — complete CLI and reference studies.** Expose safe next actions, budgets, evidence paths, two demos, clean reproduction scripts, and timed metrics. This is the first semantic product milestone.
9. **U8 — local API and decision bench.** Embed the Vite build, secure the loopback API, implement full responsive/a11y states, and perform at least three build-see-exercise-critique-rebuild passes plus a different-model critic.
10. **U9 — report, packaging, and hardening.** Add safe report export, install/release packaging, license/contributor docs, performance ledger, threat-model limitations, final adversarial run, and final didrun HTML evidence.

The sequencing deliberately proves an end-to-end CLI product before visual polish. It also prevents the studio from inventing client-side statuses that the kernel cannot support.

## Explicit cuts and stop conditions

Cut from this build: browser adapter; generated GUI exploration; model-generated probes, explanations, or tests; arbitrary JavaScript normalizers; fuzzy/tolerant equality; user plugins; persistent execution-evidence cache; prepared snapshots/reflinks; Docker; hostile-code containment; agent launching; worktree management; Wake history import; merge/rank; automatic setup inference; multi-step workflow DSL; database cloning; equivalence-class generalization; hosted service; Windows; and crash-resume.

Stop at the truth kernel if canonical mutations survive or state-machine commands can create illegal evidence. Stop at observation-only if fresh worlds cannot keep setup/control failures out of candidate behavior. Stop at one domain if the second requires a separate truth model rather than a typed adapter. Stop before contracts if selected-field emitters are not deterministic and standalone. Ship CLI-only if the mutable studio fails the security or accessibility gates. Never substitute Docker while its daemon is unavailable, and never simulate an external model/API to satisfy a visual or integration requirement.

## Exact changes required in the living brief

1. Replace the one-line “smallest behavior” language with “locally reduces an observed-stable behavioral split under named reducers and budgets.”
2. Define the fifteen-minute metric as a measured target for the two dependency-free reference studies on a named machine. Give imported repositories explicit budgets and no blanket latency guarantee.
3. Replace broad 5k-100k LOC execution implication with “bounded supported Git trees under entry/byte/setup budgets”; retain that size range as a target audience hypothesis only.
4. Name Go core plus embedded React/Vite studio and the package split above as the reference architecture.
5. Add the content-addressed immutable artifact graph and atomic head pointer; explicitly reject an event-log/replay/resume runtime to preserve the Wake boundary.
6. Define one-invocation CLI and one-seeded-request HTTP as the golden-path stimulus shapes. Move arbitrary action sequences out.
7. Define strict canonical values and state that encoding/json output is not itself identity; external decimals are tagged canonical strings and duplicate keys/negative zero reject.
8. Make the three state machines—artifact lineage, execution attempt, and reduction—explicit acceptance contracts.
9. Add the default/hard budgets table and require pre-run trial-cost disclosure.
10. State Darwin/Linux support separately and require native receipts; cross-compile does not count.
11. Make the standalone first emitter node:test, exact-witness and selected-field only, with byte-deterministic output and no Countershape dependency.
12. Make the two-domain criterion “shared truth/observation/reduction/choice services plus typed adapters,” not one universal stimulus or reducer.
13. Add CLI-only and observation-only fallback products as honest failure cuts.
14. Move browser, plugin protocol, execution caching, prepared worlds, generalization, and arbitrary stateful workflows beyond the reference roadmap.

## Ground-truth tally

### Confirmed from the current brief and completed deep-dive lanes: 16

1. The intended scope is 2-4 immutable committed candidates in one trusted local repository.
2. CLI and HTTP are the two required domains.
3. Wake owns agent supervision, execution history, replay/fork, and effect-ledger territory.
4. Docker is unavailable as a local golden-path dependency.
5. V1 cannot claim hostile-code or network containment.
6. Git checkout/worktree/archive semantics are insufficient for byte-faithful materialization.
7. WorldPlan and WorldInstance must be distinct.
8. Captured evidence is post-capture-policy and pre-projection, not necessarily raw.
9. Exact canonical projection equality is the safe v1 comparator.
10. Stable means only observed stable across declared repeats.
11. Reduction must preserve the full candidate-to-outcome map.
12. Reduction is tri-valued and unresolved neighbors block local-minimality.
13. Fresh confirmation cannot be satisfied from discovery evidence.
14. REJECT_ALL differs from CUSTOM_EXPECTATION.
15. Choicepoint artifacts require immutable revision/staleness semantics.
16. Local studio writes require loopback security and compare-and-swap.

### Strong engineering inferences: 14

1. Go is the lowest-risk core for this local binary/process/artifact system.
2. React is justified by the density and number of decision states.
3. Typed adapters should share orchestration, not data shapes.
4. A filesystem artifact graph is sufficient and avoids a Wake-like event runtime.
5. The first generated target should be node:test because both required user domains can be exercised through standard Node libraries.
6. One request/invocation is enough to prove the two-domain engine.
7. Strict canonicalization must precede all I/O work.
8. Progress notifications need not be durable evidence.
9. Reference fixtures should require no dependency installation.
10. Trial multiplication must be shown before execution.
11. Runtime OS validation is categorically different from cross-compilation.
12. Mutation tests are necessary to validate negative correctness claims.
13. A CLI semantic milestone reduces UI-driven scope risk.
14. CLI-only, observation-only, and one-domain outcomes are legitimate cuts rather than disguised completion.

### Unverified implementation and product bets: 12

1. Git-object materialization remains fast enough when repeated hundreds of times.
2. Three discovery repeats and three confirmation repeats balance latency and useful flake detection.
3. The recommended byte and time defaults fit realistic local applications.
4. Strict outcome-labeled preservation still permits useful shrinking.
5. A declarative fixture seed covers the authorization demo without bespoke database logic.
6. Node test emitters remain readable in repositories with different test conventions.
7. Users understand selected-field exact-witness scope.
8. The React bench improves decisions beyond a well-designed CLI report.
9. Darwin and Linux process-group behavior can meet the same cooperative cleanup contract.
10. Default report redaction is useful without implying secret completeness.
11. Both reference studies can hit the fifteen-minute target without evidence reuse.
12. Real users have enough simultaneous candidate trees for the workflow to matter.

## Confidence ledger and final verdict

| Area | Confidence | Basis |
| --- | ---: | --- |
| Go truth/kernel architecture | 9/10 | Small pure packages, strict types, standard crypto, and property/state testing are tractable. |
| Git/process substrate on macOS | 8/10 | Clear plumbing and POSIX design; implementation fixtures still required. |
| Linux runtime parity | 5/10 | Design is portable, but only native execution can validate it. |
| Two typed adapters over one kernel | 8/10 | Shared algebra is crisp if stimulus/projection types remain separate. |
| Tri-valued reduction | 7/10 | Termination is specifiable; useful compression and runtime cost remain bets. |
| Standalone node:test residue | 8/10 | Exact witness and selected fields keep the compiler bounded. |
| Secure local studio | 7/10 | Standard controls are known; visual/a11y and write-path testing are substantial. |
| Fifteen-minute reference metric | 6/10 | Credible for dependency-free fixtures, not imported repositories. |
| Arbitrary 5k-100k LOC repository usability | 3/10 | Fresh setup/materialization cost and uncontrolled dependencies are unmeasured. |
| Adoption value | 4/10 | Technical usefulness does not establish workflow frequency or maintenance pull. |

**Verdict: MODIFY, THEN PROCEED.** Build the full bounded reference spine, not the broad platform. The elegant architecture is a deterministic Go evidence compiler surrounding fresh trusted-local executions, with typed CLI/HTTP edges and a React decision bench. Its most important engineering feature is refusal: unsupported trees, uncontrolled worlds, unstable outcomes, unresolved reductions, stale rulings, and unverified platforms remain explicit terminal states. If the implementation preserves that refusal behavior while producing two genuinely useful standalone decision contracts, Countershape earns the “reference system” description. Everything beyond that—real-repository speed, broad adapters, sandboxing, collaboration, and adoption—is a later human and market tail, not something this build can honestly declare.
