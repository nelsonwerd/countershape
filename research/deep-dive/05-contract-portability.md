# Deep dive lane 5: contract compiler, adapter semantics, and portability

- **Reviewer posture:** assume Countershape is deleted immediately after export
- **Date:** 2026-07-14
- **Target claim:** an exact-witness human ruling can become readable repository-native test code that runs without Countershape for both HTTP authorization and CLI configuration precedence
- **Verdict:** **MODIFY, THEN PROCEED**
- **Overall confidence:** **8/10** for the bounded contract described here; **3/10** for framework-neutral or generalized contracts

## Executive finding

Countershape can emit a genuinely standalone contract, but “standalone” needs a hard definition: after generation, a clean checkout with a supported Node.js runtime must be able to run the emitted test while Countershape is absent from `PATH`, `node_modules`, the filesystem, and the network. The test may use Node core modules and repository code. It may not import a Countershape package, call a Countershape daemon, resolve an external API, inspect a Wake log, or reconstruct a hidden normalizer.

That boundary is achievable if the product stops treating “the contract” as one artifact. Three objects have different truth jurisdictions:

1. The **Decision Record** is immutable historical provenance: which confirmed Choicepoint the human ruled on, which outcome fields they selected, what was allowed, and what was explicitly not established.
2. The **Generated Contract Bundle** is deterministic source: an inert manifest, fixture, readable predicate, ordinary `node:test` file, and a small vendored runner generated as source. It contains everything needed to execute the exact predicate, but not the local captured evidence store.
3. A **Contract Execution** is a new observation against the current checkout. It can conform, contradict, or fail to produce an eligible observation. It is not a replay of the original world and does not freshen or rewrite the historical Choicepoint.

The largest semantic risk is subtler than over-asserting fields. A selected field set can be too weak to represent the ruling. Suppose the human allows one observed outcome and rejects another, but both have status `404` and differ only in a selected-out-of-scope header. A status-only predicate would accept the rejected outcome. Compilation therefore needs a **separation obligation**: under the selected field projection, no disallowed observed outcome may equal any allowed observed outcome. If that obligation fails, the compiler returns `AMBIGUOUS_SCOPE`; it must not silently broaden the human's ruling.

The second necessary correction is that `REJECT_ALL` is not an executable oracle. It records that every observed outcome is unacceptable, but it does not say what future behavior is acceptable. Only a separately authored, typed `CUSTOM_EXPECTATION` can create a test for an outcome no candidate produced and legitimately demonstrate `none conforms`. An always-failing test generated from `REJECT_ALL` would preserve dissatisfaction, not intent.

With those constraints, the v1 compiler should emit only an exact set-membership predicate over explicitly selected, typed fields at one concrete witness. No regex, tolerance, wildcard, inferred invariant, generator quantifier, semantic matcher, inline JavaScript, or “apply to similar cases” toggle belongs in the reference path.

## Severity-ranked findings

| Severity | Finding | Failure if unchanged | Required disposition |
| --- | --- | --- | --- |
| **P0** | Selected fields may not distinguish allowed from disallowed observed outcomes. | The checked-in test accepts behavior the human explicitly rejected. | Enforce the separation obligation; fail `AMBIGUOUS_SCOPE` and return to field selection. |
| **P0** | A historical Choicepoint, emitted source, and later test run are being described as one “contract.” | Stale evidence appears refreshed, or a current failure appears to invalidate history. | Model `DecisionRecord`, `ContractBundle`, and `ContractExecution` separately. |
| **P0** | `REJECT_ALL` has no positive acceptance condition. | Compiler emits a permanently red test or invents the desired result. | Emit a non-executable decision record only; require typed `CUSTOM_EXPECTATION` for a test. |
| **P0** | Hidden normalizer/runner behavior can survive export as an undeclared dependency. | Test passes in Countershape but cannot reproduce the predicate independently. | Vendor the narrow runner/projection source into the bundle; test with Countershape physically unavailable. |
| **P1** | Infrastructure failures can be flattened into assertion mismatches. | A startup error is reported as product contradiction, corrupting the ruling's meaning. | Keep harness eligibility errors disjoint from `CONTRADICTS`. |
| **P1** | “Portable” can imply cross-platform, cross-runtime, or framework-neutral behavior. | Users trust a contract outside the environment its process semantics support. | Claim Node 24 LTS on macOS/Linux for v1; fail closed elsewhere. |
| **P1** | Generated code can freeze machine paths, timestamps, candidate names, or formatter drift. | Same ruling emits different bytes or leaks local provenance. | Pure compiler, repo-relative paths, LF, sorted artifacts, pinned emitter, cross-root byte-identity gate. |
| **P1** | Ruling emission can race a changed Choicepoint or projection. | Contract encodes an obsolete witness while the UI shows a newer one. | Digest-addressed immutable lineage and compare-and-swap emission from a freshly confirmed revision. |
| **P1** | A standalone HTTP test can accidentally use higher-level client semantics that differ from discovery. | Redirects, decompression, header joining, or body decoding alter the predicate. | Use a generated Node core `http.request` runner with explicit byte and header behavior. |
| **P1** | Repository test execution can inherit ambient secrets and `NODE_OPTIONS`. | A regression contract leaks credentials or changes behavior by machine. | Generated sparse environment, explicit pass-through list, private temp state, loud unsupported-secret error. |
| **P2** | Artifact integrity can be confused with authenticity. | A SHA-256 manifest is presented as proof of authorship or untampered Git history. | Call it byte-integrity only; Git review/signing remains outside Countershape. |
| **P2** | A deterministic emitter is not automatically a maintainable test. | Repository APIs move and the test becomes orphaned despite valid bytes. | Emit readable source and a decision README; integration and long-term ownership remain human work. |

## Exact compilation boundary

### Preconditions

`compile(choicepoint, ruling, emitter, target)` is pure and refuses input unless all of the following hold:

- the Choicepoint is an immutable `CHOICEPOINT_READY` revision derived from a distinct fresh confirmation batch;
- every included candidate is `OBSERVED_STABLE(k/k)` for that batch and excluded candidates are listed with reasons;
- the Choicepoint binds the exact candidate-to-canonical-outcome map, projection configuration, adapter, World Plan, reducer grade, and captured-observation digests;
- the ruling's `choicepoint_digest` equals the ready revision, with no superseding semantic ancestor;
- the action is `ALLOW_OBSERVED` or `CUSTOM_EXPECTATION`;
- every selected field is in the adapter's v1 portable field registry;
- selected expected values are representable in the exact typed predicate language;
- allowed and disallowed observed outcomes are separable under those fields; and
- target paths, command templates, fixtures, and environment bindings are inert schema-validated data with no secret value or absolute machine path.

The fresh confirmation batch cannot be the shrink batch or execution-evidence reuse. Its purpose is to show that the reduced witness still produced the same outcome-labeled map in new World Instances immediately before the human rules. Finite repeats remain `OBSERVED_STABLE(k/k)`, never deterministic.

### Ruling semantics

| Human action | Durable record | Executable bundle | Exact semantics |
| --- | --- | --- | --- |
| `ALLOW_OBSERVED` one | yes | yes | Accept the selected-field tuple for one confirmed observed outcome. |
| `ALLOW_OBSERVED` many | yes | yes | Accept the set of selected-field tuples for the chosen confirmed outcomes; order and duplicates have no meaning. |
| `REJECT_ALL` | yes | **no** | No observed tuple is acceptable; desired behavior remains unspecified. |
| `DEFER` | yes | **no** | Preserve the unresolved question without an oracle. |
| `REFINE` | yes, plus successor request | **no** | Start a new lineage with changed stimulus, fields, world, or scope. |
| `CUSTOM_EXPECTATION` | yes | yes | Accept a typed exact tuple entered and reviewed by the human even if no candidate produced it. |

`ALLOW_OBSERVED` may select one or several outcomes, not candidate names. Candidate identity is provenance only. If two candidates share an allowed fingerprint, the record says the behavior was allowed; it does not endorse either implementation.

`CUSTOM_EXPECTATION` uses the same field registry and exact value language as observed rulings. It cannot contain executable source. A human may choose status `401` when confirmed candidates produced `403`, `404`, and `200`; all candidates can then honestly `CONTRADICT` the contract. This is the only v1 route to the required `none conforms` example.

### Separation obligation

Let `select_F(o)` be the typed tuple obtained by projecting canonical outcome `o` onto selected fields `F`; missing values remain tagged missing. For an observed ruling with allowed set `A` and disallowed set `D`, compilation requires:

```text
for every a in A and d in D: select_F(a) != select_F(d)
```

The compiler also canonicalizes and deduplicates the allowed tuples. Selecting two outcomes that collapse to one tuple is legal only if no disallowed outcome has that tuple; the preview must say that two historical outcomes compile to one executable alternative. If a disallowed outcome collides, the compiler returns `AMBIGUOUS_SCOPE` with the colliding outcome digests and candidate-neutral field differences that could separate them. It does not auto-select those fields.

For a custom expectation, compilation additionally shows how its tuple relates to every observed tuple. It may equal an observed tuple only after the user confirms that this is intentionally equivalent to allowing that behavior; otherwise `CUSTOM_DUPLICATES_OBSERVED` prevents accidental use of the wrong action.

## Concrete bundle format

The reference emitter writes one repo-relative directory such as `countershape/contracts/cs-8bf29ad14c1e/`:

```text
decision.json          immutable, deterministic historical record
fixture.json           exact stimulus and runner target, no code
contract.test.mjs      readable node:test entry point
harness.mjs            vendored, versioned core-only runner/projection
README.md              human summary, run command, limitations
manifest.json          sorted artifact paths, modes, SHA-256 byte digests
```

The identifier derives from the Choicepoint and ruling digests, not a mutable title. A user-facing slug may appear in the README but never controls identity. The manifest covers every file except itself and is not a signature. The test verifies the referenced decision, fixture, and harness bytes before running; Git remains the review/history mechanism.

`decision.json` should contain schema and digest versions, the Choicepoint and ruling digests, original and minimized stimulus digests, reduction grade, fresh confirmation batch digest and `k/k` counts, candidate-neutral observed alternatives, excluded-state summaries, the selected field paths, the compiled predicate, emitter identity, target profile, and any didrun receipt references verbatim. It must not contain absolute paths, raw secret-bearing bodies, candidate source, or a claim that a receipt proves the product behavior. Missing receipts are recorded as `UNRECEIPTED`; grades are never rewritten.

The executable predicate is a deliberately small AST, not generated assertion source:

```json
{
  "kind": "one-of-exact/v1",
  "domain": "http/v1",
  "fields": [
    {"path": [{"field": "status"}], "type": "safe-integer"}
  ],
  "alternatives": [
    [{"type": "safe-integer", "value": "404"}]
  ],
  "origin": "allow-observed"
}
```

Typed values include `absent`, `null`, `boolean`, safe integer as a canonical decimal string, exact decimal string, UTF-8 string, bytes as base64, ordered string list, and strict canonical JSON. They do not include JavaScript numbers outside the safe integer range. Duplicate JSON names, lone surrogates, non-finite values, and negative zero reject before a typed value exists. This follows the interoperability constraints and cautions in [RFC 8785](https://www.rfc-editor.org/rfc/rfc8785.html) and its verified [`-0` erratum](https://www.rfc-editor.org/errata/rfc8785), without claiming JCS itself provides semantic identity.

Field paths are typed segments, not dot strings or unchecked JSON Pointer text. Supported HTTP fields are response status, an explicitly named response-header multimap, and explicitly selected strict-JSON body paths or complete bounded body bytes. Supported CLI fields are completion kind, exit code or signal as disjoint variants, stdout bytes/text, stderr bytes/text, and explicitly declared output-file bytes/strict-JSON paths inside the fixture root. Latency, PID, port, timestamps, absolute paths, log order, and undeclared filesystem state are not compilable fields.

## Standalone execution semantics

The emitted test targets Node 24 LTS on supported macOS and Linux. Node's current release table identifies Node 24 as LTS and Node 20 as end-of-life in 2026; the reference artifact should therefore not establish a new baseline on Node 20. See the official [Node release schedule](https://nodejs.org/en/about/previous-releases). Node 22 LTS can be an additional tested target, but Node 24 is the reference receipt environment. Unsupported majors and Windows produce `CONTRACT_ENV_UNSUPPORTED`, a test failure with a diagnostic, never a skip or pass.

`node --test countershape/contracts/<id>/contract.test.mjs` must work without package installation. The bundle uses only `node:test`, `node:assert`, `node:child_process`, `node:http`, `node:fs`, `node:path`, `node:os`, and `node:crypto`. Node's test runner gives each test file process isolation by default but inherits some process options, so the generated runner must not mistake test-process isolation for candidate isolation and must explicitly control candidate environment. See the official [`node:test` execution model](https://nodejs.org/api/test.html).

Before execution, the test validates schema versions, manifest digests, target runtime/OS, repo-relative path containment, and absence of forbidden environment bindings. It creates a mode-private temp root, `HOME`, `TMPDIR`, and state directory. It uses explicit argv arrays and `shell: false`; Node documentation distinguishes direct spawning from shell execution and warns that killing a direct parent does not necessarily kill descendants. See [`child_process`](https://nodejs.org/api/child_process.html). The generated POSIX helper uses a new process group, bounded byte capture, timeout escalation, and a final group probe. This is cooperative cleanup, not hostile-code containment.

Every run yields one of two top-level categories:

```text
ELIGIBLE_OBSERVATION -> CONFORMS | CONTRADICTS
INELIGIBLE_EXECUTION -> typed harness/projection error
```

A CLI exit code `2`, an HTTP `500`, empty stdout, an absent optional header, or invalid business response can be eligible product behavior if the declared adapter completed and can project it. `START_ERROR`, `READINESS_ERROR`, `TIMEOUT`, `OUTPUT_LIMIT`, `ORPHAN_RISK`, missing required channel, or invalid strict JSON is not a product contradiction. The ordinary test process exits nonzero in both contradiction and harness-error cases, but TAP diagnostics and machine-readable error codes preserve the distinction.

### HTTP authorization profile

The generated HTTP harness launches one declared local service with a repo-relative command, sparse environment, `HOST=127.0.0.1`, a unique port, and private state paths. The proof profile is local cleartext HTTP only: no TLS, proxy, redirect following, cookie jar, HTTP/2, WebSocket, streaming response, external hostname, or third-party API. It uses Node core `http.request`, not `fetch`, so redirect and automatic content behavior remain explicit. It caps bytes before decoding, preserves response status, models headers as a declared multimap from raw header pairs, and applies exactly the emitted projection operations.

Readiness is a separate, declared, bounded request. The proof fixture must guarantee it is side-effect-free; a general repository cannot receive that guarantee by assertion. After readiness, the exact minimized request is sent once in a fresh service world. Teardown occurs before the observation is eligible. A failed teardown becomes `ORPHAN_RISK`, not a pass.

The authorization fixture is dependency-free and deliberately concrete: a local Node server receives one support-role request for a cross-tenant invoice. Candidate trees return `403`, `404`, or a metadata-bearing `200`. A ruling selecting only status `404` compiles to one exact alternative. A second confirmed Choicepoint can use `CUSTOM_EXPECTATION` status `401` against candidates that produce none of it, proving `none conforms` without inventing an external identity provider or API.

### CLI configuration-precedence profile

The CLI harness spawns one declared repo-relative executable directly. The exact stimulus binds ordered argv, stdin bytes, a sparse environment with `Absent` distinct from `Present("")`, a fresh fixture directory, and bounded declared input files. No shell expansion, globbing, command substitution, user config directory, login shell, or ambient `PATH` lookup is permitted. A `{node}` token may resolve only to `process.execPath`; other executables resolve to validated files under the current checkout.

The deterministic fixture writes three conflicting values for the same setting: a config file value, an environment value, and a command-line flag. Candidate trees implement different precedence orders. The minimized witness retains only the three sources necessary for disagreement. An exact selected stdout JSON path such as `effectiveMode`, plus exit code if the human selects it, becomes the contract. Changes to unselected diagnostics must not fail the test. A signal, timeout, malformed selected JSON, or write outside the fixture root is ineligible, not equivalent to any precedence outcome.

Neither fixture installs dependencies or calls a real external API. The demonstration repositories should be self-contained Node core programs so the contract claim is falsifiable independently of package registries, credentials, Docker, Wake, or hosted services.

## Compiler and runtime error taxonomy

Compile-time errors are stable machine codes with candidate-neutral explanations:

- `UNCONFIRMED_CHOICEPOINT`: no distinct fresh confirmation batch;
- `STALE_CHOICEPOINT`: a semantic ancestor or ruling target was superseded;
- `NONCOMPILABLE_RULING`: `REJECT_ALL`, `DEFER`, or `REFINE` requested code emission;
- `NO_SELECTED_FIELDS`: an empty predicate would accept everything;
- `UNSUPPORTED_FIELD` or `UNSUPPORTED_PREDICATE`: target cannot reproduce the requested semantics;
- `AMBIGUOUS_SCOPE`: allowed and disallowed observed outcomes collide under selected fields;
- `CUSTOM_DUPLICATES_OBSERVED`: custom value is actually an observed value without explicit confirmation;
- `SECRET_BOUND_TARGET`: target embeds or requires a secret value;
- `EXTERNAL_DEPENDENCY`: target refers to a non-loopback service, package install, or undeclared executable;
- `UNSAFE_TARGET_PATH`: absolute, traversal, symlink escape, case collision, or reserved output path;
- `ARTIFACT_PATH_COLLISION`: two emitted files normalize to one path; and
- `NONDETERMINISTIC_EMISSION`: repeated compilation does not produce the same sorted bytes.

Runtime ineligibility codes include `CONTRACT_ENV_UNSUPPORTED`, `BUNDLE_INTEGRITY_ERROR`, `MATERIALIZATION_ERROR`, `START_ERROR`, `READINESS_ERROR`, `PROBE_TRANSPORT_ERROR`, `TIMEOUT`, `OUTPUT_LIMIT`, `ORPHAN_RISK`, `PROJECTION_REJECTED`, `MISSING_REQUIRED_CHANNEL`, and `TEARDOWN_ERROR`. `ASSERTION_MISMATCH` is reserved for an eligible canonical tuple outside the allowed exact set. The diagnostic includes expected and received selected fields, never unselected secret-bearing output by default.

## Determinism, staleness, and compatibility

The compiler returns a sorted list of `(relative_path, mode, content_bytes)`. Every default is materialized. Output contains LF newlines, stable UTF-8, no current time, random value, current username, absolute path, temp port, candidate nickname, branch name, or host formatter output. Compile twice in different absolute directories and under different locale/timezone/newline settings; every path, mode, and byte must match.

Changing candidate trees, World Plan, adapter, projection, repeat policy, reducer set, selected fields, ruling, emitter, or target profile creates a new immutable lineage. Existing bundles remain historical and are marked superseded by a new record; they are not edited in place by Countershape. Moving the generated directory within the repository must not change execution because all reads are relative to `import.meta.url` and the configured repository root relation.

A later test run does not require the current checkout to equal an original candidate tree. That is the point of a regression contract. It records the current Git identity in optional execution evidence, but it does not alter the Decision Record or claim to recreate the original World Instance. The checked-in test's current result is repository truth for current conformance; the Choicepoint remains historical evidence about why that test exists.

Portability is intentionally narrow:

- **supported:** dependency-free generated Node source, Node 24 LTS reference and tested Node 22 LTS, macOS/Linux, UTF-8 repository paths, one-shot CLI, one local HTTP/1.1 service, exact typed projection;
- **not claimed:** Windows process semantics, Bun/Deno/browser runtimes, Jest/Vitest-native APIs, arbitrary frameworks, containers, package installs, TLS, external services, databases, parallel tests, hostile candidates, custom plugin code, tolerant/fuzzy comparisons, property quantification, or universal reproducibility;
- **human integration tail:** add the direct `node --test` invocation to the repository's normal suite, select supported CI OS/Node versions, maintain launch commands when the application moves, and review deliberate contract supersession.

This does not overlap Wake. The bundle launches one application solely for one ordinary test, keeps no agent event ledger, supervises no fleet, offers no resume/fork/replay UI, and consumes no Wake runtime state. Wake or any other tool may have produced the candidate commits; that provenance is inert metadata only.

## Required adversarial and mutation gates

### Compiler properties

1. Compile the same ruling twice in two clean absolute roots, locales, timezones, and umasks; require identical path/mode/byte sets.
2. Permute candidates and allowed-outcome selection order; require identical predicate and bundle bytes.
3. Select a weak field that makes one allowed and one disallowed outcome collide; require `AMBIGUOUS_SCOPE` and no files.
4. Select fields that distinguish them; require successful emission and assertions over only those fields.
5. `REJECT_ALL`, `DEFER`, and `REFINE` emit no executable file. `CUSTOM_EXPECTATION` emits one and can make all original candidates contradict.
6. Supersede the projection, fresh batch, candidate set, or ruling target between preview and emission; compare-and-swap must return `STALE_CHOICEPOINT`.
7. Attempt timestamps, absolute paths, candidate labels, secret values, shell strings, traversal, symlink output parents, Unicode/case collisions, and unsupported fields; all reject before writing.
8. Remove Countershape from `PATH`, rename its source directory, clear `node_modules`, and deny network. Both emitted proof contracts still execute.

### Predicate and adapter mutations

9. Mutate the emitter to assert an unselected header, body field, stderr, latency, or file. A fixture changes only that field; the correct contract passes and the mutant fails.
10. Mutate exact set membership to majority choice, candidate-name choice, first alternative, substring, regex, numeric coercion, or tolerant comparison. Targeted cases kill every mutant.
11. Collapse `Absent` into empty, signal into exit code, empty bytes into missing, invalid JSON into empty object, or duplicate headers into a comma-joined string. Each mutant is killed.
12. Reorder JSON object keys without changing meaning under the declared strict projection; equality remains invariant. Reorder arrays or duplicate headers where order is declared significant; equality changes.
13. Feed duplicate JSON names, unsafe integers, negative zero, lone surrogates, invalid UTF-8, NUL bytes, and truncated output. Every case reaches the specified typed result or projection rejection—never a silent coercion.
14. Return HTTP `500` after a complete request and prove it is eligible observed behavior. Refuse connection, hang readiness, exceed output, and leak a grandchild; each is a distinct ineligible harness error.
15. Run a CLI that exits `2` normally and prove it can conform when selected. Kill it by signal and prove the result cannot equal exit `2`.

### Standalone and lifecycle gates

16. Run every emitted bundle from a clean checkout using only `node --test <file>` on macOS and Linux CI for Node 24; run Node 22 only if it is claimed supported.
17. Move the repository and contract directory to different absolute paths; results and generated bytes remain unaffected.
18. Seed fake AWS, npm, SSH, Git, and cloud tokens in the parent environment; the candidate cannot observe them unless a test explicitly declares a nonsecret pass-through.
19. Change unselected HTTP body/header fields and CLI diagnostics; conformance remains. Change each selected field; contradiction is exact and readable.
20. Run the contract concurrently despite the unsupported profile; it must allocate independent state/ports or fail closed, never cross-contaminate and pass.
21. Tamper with decision, fixture, or harness bytes without updating the manifest; the test stops at `BUNDLE_INTEGRITY_ERROR` before candidate execution.
22. Kill mutants that import Countershape, contact localhost Countershape services, invoke package managers, use a shell, resolve an external hostname, interpret didrun grades, or read Wake artifacts.

Every load-bearing command for these gates belongs in didrun during the build. A code snapshot is insufficient: tests 8, 16, 17, and 18 must physically exercise absence, relocation, clean runtime, and sparse environment.

## Exact required changes to `docs/CONCEPT_BRIEF.md`

1. Replace singular “contract” wording with three explicit objects: immutable `DecisionRecord`, deterministic `ContractBundle`, and append-only/current `ContractExecution` evidence.
2. Define standalone as: Countershape absent from `PATH`, dependencies, filesystem, and network; only Node core plus repository code; direct clean-checkout invocation succeeds.
3. Add the selected-field separation obligation. Compilation must reject `AMBIGUOUS_SCOPE` if any allowed and disallowed confirmed outcome collapse to the same selected tuple.
4. Limit v1 predicates to `one-of-exact/v1` over a typed portable field registry. Defer regex, tolerance, wildcards, arbitrary code, generators, inferred invariants, and scope generalization.
5. State that `ALLOW_OBSERVED` compiles selected behavior tuples, never candidate identities or majority. Multiple allowed outcomes compile as a canonical set.
6. Separate `REJECT_ALL` from `CUSTOM_EXPECTATION`. The former emits only a record; the latter is the only path to an executable `none conforms` case.
7. Require a distinct fresh confirmation batch before ruling and compare-and-swap emission against the exact Choicepoint digest. Semantic changes create new immutable lineages and stale descendants.
8. Add the six-file deterministic bundle and require no absolute paths, timestamps, random values, secrets, branch names, or machine formatting. Cross-directory byte identity is a release gate.
9. Make Node 24 LTS on macOS/Linux the reference target. Claim Node 22 only after the full standalone matrix; explicitly exclude Windows and alternate runtimes from v1.
10. Specify HTTP v1 as one local cleartext HTTP/1.1 service using explicit Node core request semantics; no external API, TLS, redirects, cookies, streaming, WebSockets, or third-party dependency.
11. Specify CLI v1 as direct argv/stdin/sparse-env execution with no shell or ambient user config; preserve exit-vs-signal and absent-vs-empty distinctions.
12. Add the two-level runtime result: eligible observations become `CONFORMS` or `CONTRADICTS`; setup, transport, timeout, output, projection, and teardown states remain ineligible typed failures.
13. Clarify that current contract execution tests the current tree and does not freshen historical evidence or recreate an original World Instance.
14. State that embedded receipt references are verbatim provenance only and may be `UNRECEIPTED`; the standalone runner neither interprets nor upgrades them.
15. Add a kill condition: narrow or kill standalone export if either proof domain needs Countershape at runtime, hidden normalizer behavior, package/network access, a shell, framework-specific magic, or assertions broader than the human-selected exact fields.

## Ground-truth tally

### Confirmed by primary documentation or repository artifacts: 9

1. The current brief requires standalone tests, exact selected fields, HTTP and CLI proof domains, fresh conformance, and no Countershape runtime dependency.
2. The architecture lane requires a fresh confirmation batch and separates `REJECT_ALL` from `CUSTOM_EXPECTATION`.
3. The architecture lane limits v1 to exact canonical equality and records missing/control states distinctly.
4. The security lane requires direct argv execution, sparse environment, byte caps, POSIX process-group cleanup, and no sandbox claim.
5. Node 24 is LTS in July 2026; Node 20 is end-of-life.
6. Node's built-in test runner can execute ordinary JavaScript test files without a third-party package.
7. Node child processes do not automatically provide descendant containment and shell execution worsens process ownership.
8. RFC 8785 requires duplicate-name rejection and unchanged Unicode strings for its canonical form, while verified errata warns about negative-zero collapse.
9. Wake owns agent runtime/event/replay concerns, which this generated one-test bundle does not need.

### Strong engineering inferences: 12

1. Historical decision, emitted source, and current execution need separate identities and state transitions.
2. Field selection must separate allowed from disallowed observed outcomes to faithfully encode the ruling.
3. A constrained predicate AST is safer and more portable than emitted arbitrary assertion code.
4. A vendored core-only harness is the simplest way to remove a hidden Countershape dependency while preserving adapter semantics.
5. An exact allowed set handles one-or-many rulings without majority or candidate identity.
6. `REJECT_ALL` cannot produce a useful positive oracle.
7. Node core `http.request` gives a narrower explicit HTTP profile than a convenience client for the proof fixture.
8. Runtime ineligibility must remain distinct from product contradiction even though both fail an ordinary test command.
9. Byte-integrity manifests help detect accidental drift but do not establish authenticity.
10. Current-tree conformance should not require matching an original candidate tree.
11. Package-free proof fixtures are necessary to test the product claim independently of registries and secrets.
12. Node 24/macOS/Linux is a credible reference profile; “framework-neutral portable contract” is not.

### Unverified implementation and adoption bets: 8

1. The six-file bundle remains readable enough that maintainers will keep it rather than rewrite it.
2. A vendored harness can stay small while matching discovery projection semantics exactly.
3. Real TypeScript repositories can expose startup commands and state paths without custom code.
4. Fresh local HTTP startup is fast enough for ordinary regression suites.
5. Node 22 and Node 24 produce identical relevant header/process behavior for claimed cases.
6. Selected exact fields capture useful intent without becoming brittle snapshots.
7. Users understand why `REJECT_ALL` cannot emit a red test and will provide a custom positive expectation when needed.
8. Teams will integrate the direct test command into CI and maintain it across repository restructuring.

## Confidence and verdict

- **Ruling-to-predicate fidelity:** 9/10 after the separation obligation; 4/10 without it.
- **Deterministic compilation:** 9/10; it is pure, bounded, and mechanically testable.
- **CLI standalone proof:** 8/10 for dependency-free local fixtures; 5/10 for imported repositories.
- **HTTP standalone proof:** 7/10 for one local HTTP/1.1 service; 3/10 for arbitrary frameworks and state.
- **Cross-machine portability:** 6/10 for the stated Node/macOS/Linux matrix; 2/10 if phrased universally.
- **Long-term maintainability:** 5/10 until users maintain generated contracts across real refactors.
- **No-Wake-overlap boundary:** 9/10 if the bundle remains a one-test runner with inert candidate provenance.

**Verdict: MODIFY, THEN PROCEED.** Countershape should build the contract compiler only after the brief adopts the three-object model, field-separation obligation, noncompilable `REJECT_ALL`, typed exact predicate AST, and narrow Node 24/macOS/Linux target. The honest flagship claim is:

> From one freshly confirmed Choicepoint, Countershape deterministically emits a readable exact-field regression bundle that runs against the current checkout with Node core alone and preserves whether a later eligible observation belongs to the human-approved set.

It does **not** emit a universal specification, prove the desired behavior safe, reproduce the original machine, generalize one witness, guarantee framework portability, or make execution receipts mean more than their verbatim grades.
