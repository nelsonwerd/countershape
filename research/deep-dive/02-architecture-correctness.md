# Countershape architecture and correctness model

- **Lane:** architecture, observation algebra, reduction, contracts, and reproducibility
- **Date:** 2026-07-14
- **Recommendation:** **PROCEED ONLY WITH MATERIAL MODIFICATIONS**
- **Confidence:** 8/10 on the model and risks below; 6/10 that the bounded implementation can satisfy it in this run; 4/10 that real repositories will yield enough comprehensible witnesses

## Executive finding

Countershape has a defensible semantic kernel, but the current brief is still too loose at exactly the places where a polished implementation could tell a false story. Five changes are prerequisites, not refinements:

1. A "world manifest" must split into a deterministic **World Plan** and a measured **World Instance**. Random ports and temporary roots cannot simultaneously be execution facts and inputs to a reproducible digest.
2. The persisted source evidence is not literally raw if secrets are redacted before storage. Call it a **Captured Observation**: pre-projection, post-capture-policy, immutable, and always linked to its capture-policy digest.
3. Shrinking cannot preserve only an unlabeled partition of candidate membership. `A|BC` can remain `A|BC` while the outcomes change from `403|404` to `200|500`; that is a different human question. The safe v1 predicate preserves the complete **outcome-labeled partition**: candidate-to-canonical-observation fingerprints, not merely coalition shape.
4. Reduction is three-valued: `PRESERVES`, `CHANGES`, or `UNRESOLVED`. A timeout, unstable replay, setup failure, exhausted repeat budget, or projection rejection is never evidence that a reduction failed. Any unresolved direct neighbor prevents a 1-minimality claim.
5. "Stable" must mean only **observed stable in k declared trials**. Finite repetition cannot prove determinism. A final witness needs a fresh confirmation batch, not just cached shrink observations.

With those changes, the core can be stated precisely: compile one strict study specification; materialize exact candidate contents into fresh worlds; classify infrastructure separately from product observations; apply one exact, versioned projection equality; form a deterministic outcome-labeled N-way partition; reduce only under a terminating typed neighborhood while preserving that labeled partition; let a human issue an exact-scope ruling; compile only the selected fields into deterministic standalone artifacts.

This is achievable for deliberately constrained CLI and HTTP fixtures. It is not yet credible for arbitrary repositories, fuzzy normalizers, probabilistic clustering, stateful browser journeys, or hostile candidate code.

## What established work actually licenses

Countershape should inherit the limits of its ancestors, not only their vocabulary.

- Zeller and Hildebrandt define `ddmin` around a test predicate and 1-minimality: no single input entity can be removed while retaining the failure. Their paper also explicitly permits `UNRESOLVED` test outcomes and discusses grammar-aware reductions; it does not make a semantic-root-cause or global-minimum claim. The celebrated Mozilla example took 139 runs, which is a useful warning about the multiplication caused here by candidates and repeat trials. See the primary [delta debugging paper](https://www.st.cs.uni-saarland.de/publications/files/zeller-tse-2002.pdf).
- Property-based systems treat a seed as only one part of replay. Current [fast-check model-based testing documentation](https://fast-check.dev/docs/advanced/model-based-testing/) records `seed`, shrink `path`, and a separate `replayPath` for executed command history. [Hypothesis documentation](https://hypothesis.readthedocs.io/en/latest/reference/api.html) warns that a fixed seed reproduces examples only absent timing, hash-randomization, and external-state nondeterminism, and explicitly says its example database is a cache that must not be relied upon for correctness. Countershape has more external state than either library, so a seed cannot be its receipt.
- Differential testing compares implementations but does not make the majority an oracle. McKeeman's original [Differential Testing for Software](https://www.cs.swarthmore.edu/~bylvisa1/cs97/f13/Papers/DifferentialTestingForSoftware.pdf) also separates crashes, loops, diagnostics, and outputs into different result categories. That supports Countershape's refusal to equate infrastructure failure, absence, and successful empty output.
- Metamorphic testing creates relations between source and follow-up executions when a single expected output is unavailable; the early [HKUST primary record](https://researchportal.hkust.edu.hk/en/publications/application-of-metamorphic-testing-in-numerical-analysis/) is useful precedent. Countershape may later use declared metamorphic relations for probe discovery, but a relation is another user-authored oracle. It cannot be silently inferred from candidate agreement.
- Flakiness is not cured by a magic retry count. Google's [flake-aware culprit-finding research](https://research.google/pubs/flake-aware-culprit-finding/) describes rerun cost and models uncertainty over more than 13,000 breakages. Countershape v1 should use conservative categorical language, not counterfeit statistical confidence from three repetitions.
- RFC 8785 provides deterministic property sorting and ECMAScript primitive serialization for I-JSON, but its [verified `-0` erratum](https://www.rfc-editor.org/errata/rfc8785) shows why "canonical JSON" is not synonymous with "semantic identity." [RFC 8259](https://www.rfc-editor.org/rfc/rfc8259.html) notes that duplicate object names are unpredictable across implementations and that exact numeric interoperability is bounded. Countershape must validate before parsing away those distinctions.
- Git content identity is repository-format-aware. Git's own [hash transition design](https://git-scm.com/docs/hash-function-transition) permits SHA-1 and SHA-256 object formats and explains that commit/tree representations refer to object names in the active format. A bare hexadecimal object ID without algorithm and repository identity is not a portable candidate identifier.

## Proposed algebra

### 1. Domain-separated immutable identities

Every persisted value has a `schema_version`, a kind, canonical bytes, and a domain-separated digest:

```text
digest(kind, bytes) = SHA-256(
  UTF8("countershape/v1/") || UTF8(kind) || 0x00 || bytes
)
```

Do not use a generic `sha256(JSON.stringify(x))`. Domain separation prevents the same bytes from being mistaken for different object kinds. Canonical bytes are produced by one pinned implementation tested against vectors. Internal specification numbers are safe-range integers only; durations are integer milliseconds; external decimals that must survive exactly are tagged canonical decimal strings. Reject duplicate JSON keys, `NaN`, infinities, unsafe integers, lone surrogates, and `-0` before the typed value exists. Do not Unicode-normalize: composed and decomposed strings may be distinct product behavior.

Each digest names exact bytes, not truth. A cryptographic match means "same encoded artifact under this scheme," never "same semantic world."

### 2. Candidate identity

```text
Candidate = {
  repository_fingerprint,
  object_format: "sha1" | "sha256",
  commit_oid,
  tree_oid,
  portable_tree_digest,
  materializer_version,
  submodule_policy,
  lfs_policy
}
```

`repository_fingerprint` identifies the trusted local object database without leaking its absolute path. `portable_tree_digest` is Countershape's SHA-256 over a sorted stream of entry type, normalized repository-relative byte path, executable mode, symlink target or blob bytes, and explicit submodule/LFS sentinel. It protects cross-object-format reports and catches materializer defects; Git remains the repository's content authority. Commit identity is provenance. Execution identity is primarily tree content because `.git` is absent from a world. If a candidate can inspect commit metadata, that fact must be declared and commit identity becomes semantic input.

Two refs resolving to the same executable tree must be rejected as duplicate candidate slots or explicitly coalesced. Allowing both fabricates N-way support without new behavior.

### 3. World Plan versus World Instance

The current singular world manifest hides an incompatibility. A plan can be deterministic while an instance necessarily contains allocated ports, PID values, wall-clock times, and temporary paths.

```text
WorldPlan = {
  candidate,
  adapter_and_runner_digest,
  materialization_policy,
  setup_argv,
  launch_argv,
  cwd_policy,
  environment_allowlist_and_bindings,
  network_policy_claim,
  fixture_and_reset_recipe,
  readiness_recipe,
  timeout_and_output_budgets,
  toolchain_requirements,
  capture_policy,
  projection_and_comparator,
  repeat_policy
}

WorldInstance = {
  world_plan_digest,
  instance_nonce,
  measured_tool_versions,
  measured_platform,
  resolved_executable_digests,
  allocated_resources,
  post_setup_manifest,
  uncontrolled_dimensions,
  start_and_finish_times,
  teardown_result
}
```

`WorldPlan` is compiled deterministically from a strict source spec with all defaults materialized. `WorldInstance` is evidence. A versioned compatibility predicate decides whether instances are comparable; instance digest equality is neither required nor sufficient. For example, different port numbers are comparable only if all request and observation paths represent them through declared placeholders, while different Node major versions are uncomparable in v1.

The source parser must reject YAML aliases, merge keys, duplicate keys, implicit environment interpolation, shell strings, absolute output paths, traversal, and symlink escapes. Commands are argv arrays. Relative paths are normalized once against a declared repository or world root. A compiled plan contains no ambient defaults.

### 4. Trial envelope and Captured Observation

Runner state and program behavior are separate sums:

```text
TrialControl =
  COMPLETE
  | SETUP_ERROR | LAUNCH_ERROR | READINESS_ERROR
  | PROBE_TRANSPORT_ERROR | TIMEOUT | CANCELLED
  | OUTPUT_LIMIT | POLICY_VIOLATION | TEARDOWN_ERROR

Channel<T> = Present<T> | Absent<declared_reason> | Truncated<digest, byte_count>

CapturedObservation = {
  trial_control: COMPLETE,
  capture_policy_digest,
  channels,
  process_or_http_metadata,
  persisted_artifact_digests
}
```

HTTP `404`, CLI exit code `2`, and an empty stdout are ordinary observed values when the adapter completed. A runner timeout is not. Absent stderr is not present empty stderr. A required channel that is absent causes projection rejection; an optional absence remains a tagged value. Truncation never compares equal to a complete prefix. If teardown fails after capture, the trial stays inspectable but is ineligible for stable reduction until the engine proves the next world is independent; v1 should simply classify it `UNCOMPARABLE` and stop that search branch.

"Raw" should be removed from user-visible claims unless bytes are truly retained. Persisted observations are after a declared capture/redaction policy so secrets can be removed before disk. Projection is the next, separately versioned lossy step. The artifact graph is:

```text
program emission
  -> capture policy (may redact; unpersisted bytes die here)
  -> immutable Captured Observation
  -> pure ProjectionResult
  -> canonical projection fingerprint
```

### 5. Projection and equality

A projection is a total, versioned function over accepted captures:

```text
N : CapturedObservation -> Ok<CanonicalProjection> | Rejected<Reason>
```

Its implementation digest and configuration digest are both semantic inputs. `Rejected` is not an empty projection. The v1 comparator is only byte equality of canonical projections. This gives an equivalence relation by construction. User-selected fields belong in the projection config, not an ad hoc pairwise comparator.

Do not ship numeric tolerances, fuzzy strings, edit-distance similarity, or order-dependent clustering in v1. `abs(a-b) <= epsilon` is not transitive: `a≈b` and `b≈c` can hold while `a≈c` does not, so candidate iteration order can change clusters. If tolerant comparison arrives later, it must first map each value into a deterministic bucket with visible boundaries; users must accept the collision semantics.

HTTP projection needs a declared multimap model. Header names are case-insensitive, but values and duplicate order can matter (`Set-Cookie` is an obvious trap). Query parameters are an ordered multimap until a projection explicitly chooses otherwise. JSON body parsing rejects duplicate keys before object construction. CLI projection keeps argv order, environment `Absent` distinct from `Present("")`, stdout and stderr as distinct byte channels, and exit status distinct from signal termination. Trimming whitespace, replacing scratch paths, sorting arrays, dropping dates, or case-folding are visible normalizer operations with vectors.

### 6. Repeated-run classification and N-way partitions

For candidate `c`, stimulus `s`, plan `w`, and repeat policy `r`, obtain `k` fresh trials. Each complete projected trial has signature `h`; non-complete trials have explicit control states.

```text
classify(c, s) =
  OBSERVED_STABLE(k, h)    if k required trials all produce h
  UNSTABLE(histogram)      if two or more complete h values occur
  UNCOMPARABLE(states)     if any world/trial/projection eligibility fails
  INCOMPLETE               if the repeat budget ends before k eligible trials
```

The UI may shorten the first label to `STABLE (3/3 observed)` but must never imply proven determinism or show an unsupported probability. `TIMEOUT`, `ERROR`, `MISSING`, and `UNKNOWN` remain detailed reasons under `UNCOMPARABLE`/`INCOMPLETE`, not candidate behavior blocks.

Let `E` be candidates with `OBSERVED_STABLE(k,h)`. A divergence exists only if `|E| >= 2`, every included candidate shares compatible world instances, and there are at least two distinct `h` values. The outcome-labeled partition is a canonical sorted map:

```text
L(s) = sorted(candidate_execution_id -> projection_fingerprint)
```

The ordinary mathematical partition can be derived by grouping identical fingerprints. Its block ID is the projection fingerprint, never ordinal `cluster-1`; candidate input order cannot change it. Unstable candidates are displayed alongside but excluded from `L`. A Choicepoint names the exact eligible candidate set and all exclusions.

Run candidates sequentially in a deterministic rotated order for each repeat to expose order-sensitive host effects without multiplying simultaneous contention. Record the schedule. A fresh confirmation batch must use a distinct instance nonce and a different rotation. Shared database, fixed port, home directory, package cache mutation, wall clock, locale, timezone, random device, DNS, and process leaks are named uncontrolled dimensions unless actually controlled.

## Partition-preserving reduction

### The dangerous counterexample

Suppose the baseline outputs are:

```text
candidate A -> 403
candidate B -> 404
candidate C -> 404
```

Deleting authentication setup produces:

```text
candidate A -> 200
candidate B -> 500
candidate C -> 500
```

Both have membership shape `{A}|{B,C}`. The second is smaller and "partition-preserving" in the unlabeled sense, but it has changed the decision from concealment semantics to missing-auth/setup behavior. A beautiful dashboard could now invite the wrong ruling.

Therefore v1 preservation is:

```text
preserves(s') iff L(s') == L(s_baseline)
```

That is strict. It may prevent shrinking when responses echo reducible input. The honest remedy is a narrow declared projection that excludes an irrelevant echo, not weakening preservation invisibly. A future `MEMBERSHIP_ONLY` experiment may preserve only the equivalence relation, but must be labeled as hypothesis discovery, show both outcome maps, and require independent confirmation before it can create a Choicepoint.

### Reducer contract

Each adapter supplies a finite, deterministic neighbor function `R_v(s)` and a well-founded measure:

```text
mu(s) = (typed_node_count, semantic_token_count, canonical_byte_length, canonical_bytes)
```

Every neighbor must be schema-valid and strictly smaller under `mu`. Reducers have names and versions: remove JSON property, remove array element, shrink scalar toward declared sentinels, remove CLI flag/value pair, remove environment binding, remove fixture entry, or remove a stateful action whose dependency constraints remain satisfied. Generic byte deletion is not valid for structured inputs unless the adapter explicitly owns a bytes grammar.

Evaluation is:

```text
evaluate(s') =
  PRESERVES    if a fresh required batch yields exactly baseline L
  CHANGES      if a fresh required batch yields another stable eligible L
  UNRESOLVED   otherwise
```

`UNRESOLVED` includes instability, infrastructure failure, projection rejection, candidate eligibility changes, timeout, cancellation, and insufficient budget. It is never cached as `CHANGES`.

The search may use ddmin-style chunk attempts plus typed scalar shrinking, but its final grade is independent of the algorithm:

- `UNCHANGED`: no accepted reduction.
- `BEST_KNOWN`: smaller preserving stimulus found, but budget ended or a direct neighbor is unresolved.
- `ONE_MINIMAL_UNDER(reducer_set_digest)`: every valid direct neighbor under the named reducers was freshly evaluated and classified `CHANGES`.

The last label requires an exhaustive final neighbor sweep. It is local to the declared neighborhood, measure, projection, candidates, world compatibility policy, and repeat policy. It says nothing about global minimum, root cause, semantic simplicity, or human comprehensibility. Always retain the original stimulus and full accepted/rejected/unresolved shrink transcript.

### Search budget and cache

Budgets are part of the compiled Study Plan: maximum candidate trials, wall time, shrink candidates, output bytes, and retries. Budget exhaustion is a result, not success. The UI must state, for example, `BEST KNOWN: 73/100 trial budget consumed; 2 smaller neighbors unresolved`.

Use two cache classes:

1. **Pure cache:** plan compilation, canonicalization, neighbor enumeration, projection of an immutable capture. Safe under exact semantic key.
2. **Execution evidence reuse:** previously captured trial batches. Never call this a correctness cache. The key includes candidate execution identity, plan, stimulus, capture/projection/comparator versions, measured compatibility class, repeat policy, schedule, and runner version. Reuse carries original instance IDs and does not count as a fresh confirmation.

Final witness confirmation and post-ruling conformance replay always execute afresh. This follows the useful Hypothesis distinction: a database accelerates discovery but is not correctness evidence.

## Choicepoint and contract state machine

Do not mutate one JSON object through vague states. Persist immutable revisions connected by explicit predecessor digests:

```text
StudySpec
  -> COMPILED(plan_digest)
  -> MATERIALIZED(candidate_set_digest)
  -> OBSERVED(batch_digest)
  -> DIVERGENCE(outcome_labeled_partition_digest)
  -> REDUCTION(result_grade, transcript_digest)
  -> CONFIRMED(fresh_batch_digest)
  -> CHOICEPOINT_READY(choicepoint_digest)
  -> RULING(ruling_digest)
  -> CONTRACT_EMITTED(contract_set_digest)
  -> CONFORMANCE_REPLAY(result_digest)
```

At any execution edge, `CANCELLED` or `PARTIAL` preserves completed evidence but cannot masquerade as the next state. Derived artifacts become `STALE` when any semantic ancestor changes; they are not overwritten or deleted. `INVALIDATED` is an explicit superseding record with reason and replacement digest. Changing candidates, projection, world plan, reducer set, repeat policy, ruling scope, or compiler version creates a new lineage.

State-machine hazards to reject:

- ruling before fresh witness confirmation;
- contract emission from `DEFER`, `REFINE`, or infrastructure-failed states;
- resuming reduction under a changed plan while retaining old 1-minimality;
- adding/removing a candidate without recomputing the N-way baseline;
- reprojecting captures under a new normalizer while showing the old decision as current;
- reporting cancellation as timeout or timeout as behavior;
- marking a trial complete before process-group teardown and artifact persistence are durable;
- retrying only the outlier candidate, which gives candidates unequal observation opportunity.

### Decision scope

The default ruling is an exact witness plus an explicit predicate over selected projection fields:

```text
Ruling =
  ALLOW_OBSERVED { allowed_fingerprints, selected_fields }
  | CUSTOM_EXPECTATION { typed_predicate }
  | REJECT_ALL
  | DEFER
  | REFINE
```

`ALLOW_OBSERVED` may select one or several stable outcomes. The predicate preview must show precisely what fields become assertions. `CUSTOM_EXPECTATION` is the only honest path to the required `none conforms` demonstration: the human may specify an expected `401` even when candidates produced `403`, `404`, and `200`, then all candidates contradict the resulting contract. `REJECT_ALL` does **not** compile an always-failing test; it records that no observed outcome is acceptable and returns the Choicepoint to implementation work. `DEFER` and `REFINE` emit no contract.

Generalizing from this witness to an equivalence class or generator domain is a separate future operation. It needs a declared quantifier domain and new counterexamples. V1 must not offer a scope toggle that turns one example into a universal rule.

### Deterministic compilation

```text
compile(choicepoint, ruling, emitter_version, target_config)
  -> sorted(path, mode, content_bytes)[]
```

The compiler is pure. Generated bytes contain no current time, absolute path, random identifier, candidate label, machine-specific newline, or formatter-version drift. Provenance may live in a separate deterministic decision record; the executable assertion includes only selected fields. Artifact-set digest covers normalized relative path, mode, and bytes. Compile twice in two clean directories and require byte identity.

The generated HTTP and CLI tests may use the repository's ordinary test runner and standard libraries, but must not import Countershape. A deterministic emitter cannot guarantee a durable test: repository APIs and fixtures can later change. The decision record remains historical evidence, while the checked-in test's ordinary suite result determines current conformance.

## Property, state-machine, and mutation test plan

### Canonicalization and identity properties

- Object insertion order, JSON whitespace, and legal escape spelling produce one canonical digest after strict parse.
- Duplicate keys, unsafe integers, `-0`, non-finite numbers, invalid UTF-8, and lone surrogates are rejected before hashing.
- NFC and NFD strings remain distinct; `Present("")` differs from `Absent`; complete empty bytes differ from truncation.
- Git SHA-1 and SHA-256 object IDs are tagged and cannot cross-parse; the same portable tree bytes produce one portable digest.
- Candidate labels, ref names, source checkout path, temp root, port, and input candidate order do not affect Study Plan or partition digest.
- Mutations: remove kind domain separation, use locale sort, use default `JSON.stringify`, accept duplicate keys, coerce `-0`, truncate a Git OID. Every mutant must be killed.

### Projection and partition properties

- Exact equality is reflexive, symmetric, and transitive across generated projections.
- Grouping is invariant under every candidate permutation; block IDs remain observation fingerprints.
- Any trial-control failure, rejected projection, or unstable histogram excludes that candidate from stable blocks.
- Two identical candidate trees are rejected/coalesced before partitioning.
- Raw/captured artifact links survive every successful projection; no projected field exists without declared source channels.
- Mutation: replace exact equality with epsilon pairwise comparison and require a generated `a,b,c` transitivity counterexample; collapse missing to empty; treat HTTP `500` as runner failure; sort `Set-Cookie`; drop exit status. Each mutant must be killed.

### Repetition and world properties

- Deliberately alternating fixture becomes `UNSTABLE`, never a block verdict.
- Setup failure, readiness failure, timeout, output limit, projection failure, and teardown leak produce distinct reasons.
- Candidate schedule rotation cannot change canonical result when fixtures are isolated.
- A fixture that writes to `$HOME`, inherits a secret, reuses a fixed port, leaves a grandchild, or shares state is caught by policy/conformance tests.
- Changing Node major, locale, timezone, normalizer config, capture policy, readiness recipe, or tool digest invalidates reuse.
- Mutation: reuse one world across shrink attempts, skip teardown wait, inherit the whole environment, or retry only one candidate. Tests must detect unequal/fresh-world violations.

### Reducer properties

- Every neighbor is valid and strictly decreases `mu`; cycles are impossible.
- Accepted steps preserve the complete candidate-to-fingerprint map, not only block cardinalities or membership.
- The authentication-removal counterexample above is rejected despite identical coalition shape.
- Any unresolved direct neighbor forces `BEST_KNOWN`; budget exhaustion can never yield 1-minimal.
- `ONE_MINIMAL_UNDER` implies an auditable final sweep over all direct neighbors with `CHANGES` receipts.
- Search transcript replay from original stimulus deterministically reconstructs every proposed stimulus and decision, although executions themselves remain fresh evidence.
- Reducer-order permutations may reach different local minima; both are honest if their named transcript and final sweep pass. This must be shown in a fixture so the UI never says "the minimum."
- Mutation: treat `UNRESOLVED` as false, compare only number of blocks, omit reducer version from keys, accept equal-measure neighbors, or stop without final sweep. Each mutant must be killed.

### State machine and compiler properties

- Property-based command sequences cover every valid transition and reject every invalid edge; replay data includes the executed command history, following fast-check's model-based precedent.
- Superseding a semantic ancestor makes all descendants stale while retaining inspectable bytes.
- Compile twice across directories/platform newline settings to byte identity; formatter and emitter versions are pinned.
- Selecting status alone cannot accidentally assert body, headers, stderr, latency, or state snapshots.
- `REJECT_ALL`, `DEFER`, and `REFINE` cannot emit executable tests; `CUSTOM_EXPECTATION` may legitimately produce `none conforms`.
- Standalone emitted tests run with Countershape absent from `PATH` and dependencies.
- Mutation: inject timestamp, absolute path, candidate name, an unselected field, or Countershape import. Snapshot/property tests must kill all.

## Severity-ranked risks and counterexamples

| Severity | Risk | Failure mode | Required response |
| --- | --- | --- | --- |
| **P0** | Unlabeled partition preservation | shrink asks a different question while retaining `A|BC` | preserve complete candidate-to-outcome fingerprints in v1 |
| **P0** | Unresolved treated as false | flaky/failed smaller case is used to claim local minimality | tri-valued evaluation; unresolved blocks 1-minimal |
| **P0** | Shared or unequal world | candidate order, readiness, or prior probe manufactures divergence | fresh world per trial; rotated schedule; explicit compatibility gate |
| **P0** | Lossy projection becomes hidden oracle | normalizer erases the very behavior under review | narrow visible ops, captured evidence link, exact equality only |
| **P0** | Contract over-assertion | user chooses `404`; test freezes body, headers, and time | selected-field typed predicate and compiler mutation tests |
| **P1** | Finite trials called deterministic | three matching runs conceal low-rate flake | `observed stable k/k` language and fresh confirmation |
| **P1** | Canonical numeric collapse | `-0`, large integer, float, or duplicate key changes identity | strict pre-parse validation and tagged exact decimals |
| **P1** | World digest paradox | random allocations destroy reproducibility or are omitted as if irrelevant | split Plan from Instance and define compatibility |
| **P1** | Cleanup race/process leak | next trial observes prior server or occupied resource | owned process group, teardown wait, leak conformance fixture |
| **P1** | Readiness mutates application state | health probe warms cache or writes state before stimulus | declared side-effect-free readiness or post-readiness reset |
| **P1** | Cache laundered as fresh evidence | old stable samples hide current nondeterminism | evidence reuse carries instance IDs; final confirmation always fresh |
| **P1** | `REJECT_ALL` emits impossible test | permanent red suite encodes no intended behavior | only custom expectation emits `none conforms` contract |
| **P2** | Reducer-local witness is incomprehensible | technically small input loses product narrative | retain original, derivation, and human comprehension gate |
| **P2** | Same-tree duplicate candidates | false appearance of consensus | reject/coalesce duplicate execution identities |
| **P2** | Adapter abstraction overclaims unity | HTTP and CLI semantics leak into generic kernel | share trial algebra; keep encoders, projections, and reducers typed per adapter |

## Exact brief and build acceptance changes

The synthesis should make these edits to `docs/CONCEPT_BRIEF.md` before prompt-pack:

1. Replace `WorldPolicy`/singular manifest language with `WorldPlan`, `WorldInstance`, and a versioned comparability result.
2. Replace "immutable raw observations" with "immutable pre-projection Captured Observations after the declared capture/redaction policy." Reserve `raw` for bytes actually persisted unchanged.
3. Define v1 comparison as exact equality over canonical typed projections. Explicitly defer fuzzy, tolerance, similarity, and order-dependent clustering.
4. Change the shrink obligation from "same candidate partition" to "same complete candidate-to-canonical-outcome fingerprint map." Membership-only reduction is out of the golden path.
5. Add tri-valued shrink outcomes and the three reduction grades `UNCHANGED`, `BEST_KNOWN`, and `ONE_MINIMAL_UNDER`.
6. State that any unresolved direct neighbor or exhausted confirmation budget forbids the 1-minimal label.
7. Change `STABLE` copy to `OBSERVED_STABLE(k/k)` and require a distinct fresh confirmation batch after shrinking.
8. Make exact-witness selected-field predicates the only v1 compilable scope. Remove equivalence-class/broader-generalization UI from v1.
9. Separate `REJECT_ALL` from `CUSTOM_EXPECTATION`; only the latter can yield the required `none conforms` executable contract.
10. Make Choicepoints and derived artifacts immutable revision chains with explicit stale/invalidation edges.
11. State that the shared engine is trial/control/identity algebra; HTTP and CLI retain typed stimulus, projection, and reducer modules. Do not require one universal reducer.
12. Put the truth model before Git/process execution in the build plan. R0 must pass vectors, properties, mutants, and a model-based state machine before R1 starts.

The minimum build gates should be:

- **Identity gate:** 100% of canonicalization vectors pass, all malformed/ambiguous encodings reject, and domain/candidate mutations are killed.
- **Algebra gate:** comparator laws and candidate-order invariance pass over generated cases; all control states remain disjoint from observed values.
- **World gate:** fresh roots, sanitized env, distinct ports, rotated schedule, teardown, and leak fixtures pass for both adapters.
- **Stability gate:** constant, alternating, delayed-timeout, and one-in-ten scripted fixtures classify exactly; product copy includes observed trial counts.
- **Reduction gate:** both HTTP and CLI produce a smaller witness; accepted steps preserve the full labeled map; a same-shape/different-output trap is rejected; an unresolved neighbor yields `BEST_KNOWN`; a separate finite fixture earns `ONE_MINIMAL_UNDER` after a complete sweep.
- **Compiler gate:** exact-field snapshots, cross-directory byte identity, no Countershape runtime dependency, `REJECT_ALL` no-emission, and custom `none conforms` all pass.
- **Reproduction gate:** three clean study executions produce identical Plan, Choicepoint, and contract digests; World Instance and run evidence digests are expected to differ and remain linked. This distinction must appear in the report.
- **Comprehension gate:** a user can view original stimulus, reduced stimulus, complete outcome map, excluded unstable candidates, capture-to-projection diff, reducer limits, exact predicate, and freshness without opening source diffs.

Cut browser replay, persistent execution caching, user-defined normalizer code, fuzzy equality, equivalence-class scope, and arbitrary multi-step stateful reducers if these gates threaten reference quality. They are not required to prove the Choicepoint lifecycle.

## Ground-truth tally

### Directly supported by primary sources or local artifacts

1. The brief and ideation files require exact Git trees, repeated trials, versioned projection, N-way partitions, named reducers, 1-minimality, exact-scope contracts, and CLI/HTTP adapters.
2. `ddmin` defines 1-minimality relative to removable entities and works with pass/fail/unresolved outcomes; grammar-aware reduction is established.
3. Property-based/stateful tools persist more than a random seed and explicitly limit what replay/cache means.
4. JSON permits interoperability hazards around duplicate names and numeric range; JCS uses I-JSON/ECMAScript constraints and collapses negative zero unless rejected per verified erratum.
5. Git object IDs are algorithm/repository-format sensitive.
6. Flaky observations make repeated diagnosis costly and uncertain; finite repetition does not establish determinism.

### Engineering inferences with strong confidence

1. Exact canonical projection equality is the only low-risk v1 clustering rule.
2. Plan/Instance separation is necessary to reconcile deterministic compilation with measured ephemeral execution.
3. Unresolved shrink attempts must block local-minimality claims.
4. A fresh confirmation batch is stronger than reusing shrink evidence.
5. Immutable artifact lineage is safer and simpler to audit than mutating state in place.
6. A selected-field predicate is necessary to prevent accidental contract strengthening.

### Unvalidated project bets

1. Strict outcome-labeled preservation still permits a materially smaller witness in realistic HTTP and CLI cases.
2. Three to five observed trials plus fixtures are an acceptable latency/cost trade for local use.
3. Users understand `BEST_KNOWN` versus `ONE_MINIMAL_UNDER` and value the honesty.
4. Narrow visible normalizers remove enough noise without erasing meaningful behavior.
5. The original-to-reduced derivation preserves human comprehension even when reduction changes input context.
6. Ordinary generated tests can remain readable and maintainable across real repository evolution.

## Confidence and verdict

- **Observation/partition algebra:** 9/10. Exact canonical fingerprints and explicit control states are tractable and highly testable.
- **Reduction truthfulness:** 8/10 if the labeled-map and tri-valued modifications land; 4/10 under the current ambiguous partition language.
- **World reproducibility:** 6/10 for curated local fixtures; 3/10 for arbitrary repositories with installers, network, databases, or background processes.
- **Contract correctness:** 8/10 for exact selected-field HTTP/CLI assertions; 3/10 for generalized rules.
- **Two-domain kernel:** 7/10 if only the algebra is shared and adapters stay typed; 4/10 if one generic stimulus/reducer abstraction is forced.
- **Human usefulness:** 5/10 until a different lane demonstrates comprehension without source/log archaeology.

**Verdict: MODIFY, THEN PROCEED.** Countershape should survive the deep dive, but its flagship claim must become stricter: it discovers an **observed-stable, outcome-labeled, locally reduced decision witness under a declared world and projection**. It does not discover the root cause, prove determinism, infer semantic equivalence, or certify the chosen behavior. If the build cannot preserve that distinction in schemas, CLI copy, dashboard states, contracts, and tests, stop at the truth-model prototype rather than shipping a persuasive false oracle.
