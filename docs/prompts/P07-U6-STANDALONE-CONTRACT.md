# P07 — U6 deterministic standalone contract residue

Implement the second half of Countershape U6 in a fresh chat. Read `docs/CONCEPT_BRIEF.md`, `research/deep-dive/05-contract-portability.md`, `research/deep-dive/06-feasibility-acceptance.md`, `research/deep-dive/07-SYNTHESIS.md`, `research/deep-dive/08-RED_TEAM.md`, and the Mode C handoff. Verify that P06's Choicepoint/store commit is sealed and `NO_COLOR=1 didrun verify --strict` exits 0. Inspect the installed Node versions before naming support. A design target is not a runtime receipt: if Node 24 is not physically run, Node 24 remains `UNRECEIPTED` even if the bundle compiles or another major passes.

## Objective

Build a pure deterministic Go compiler from one current, freshly confirmed, compilable ruling to an exact six-file Node-core bundle, plus append-only/current `ContractExecution` evidence. Prove the narrow Go and Node implementations agree on the exercised semantic profile and that the emitted test runs from a clean target checkout with Countershape absent from `PATH`, source, dependencies, services, and package registry.

The three authorities remain separate:

1. `DecisionRecord` is immutable historical human provenance for one Choicepoint digest.
2. `ContractBundle` is deterministic emitted source and byte identity.
3. `ContractExecution` is new evidence against the current checkout; it neither recreates nor freshens the historical world.

Do not describe the output as a universal specification, semantic equivalence, safety proof, framework-neutral test, or portable beyond the exact natively receipted Node/OS profiles.

## Locked compilation boundary

The emitter accepts only a construction-safe `CompilableRuling`: `ALLOW_OBSERVED` or separately reviewed `CUSTOM_EXPECTATION`. `REJECT_ALL`, `DEFER`, and `REFINE` cannot reach it through type assertions, API coercion, or stale replay. Emission checks the current Choicepoint/head digest with compare-and-swap semantics; a changed projection, confirmation, candidate set, selected fields, target, or ruling returns `STALE_CHOICEPOINT` and writes nothing.

The only executable predicate is `one-of-exact/v1` over a closed typed portable field registry. Allow-observed compiles selected behavior tuples, never candidate identities, branch names, producer provenance, first/majority outcomes, or support counts. Allow-many is a canonical set of complete tuples, never a cross-product. Before bytes exist, enforce the separation obligation: every allowed confirmed outcome's selected tuple differs from every disallowed confirmed outcome's tuple. Weak fields return `AMBIGUOUS_SCOPE` and zero artifacts. Empty fields are forbidden. Custom expectation is the only path to an executable `none conforms`; `REJECT_ALL` emits a record only.

## Owned paths and exact bundle

Own `internal/emit/node/**`, focused application-service wiring, a normative shared corpus under `testkit/contracts/**`, and standalone fixture/test scripts under `testkit/**`. Reuse the existing strict Go canonical/domain authority and Choicepoint/store types; do not fork them. Do not build the server or studio.

The compiler returns a sorted list of `(repo-relative path, mode, content bytes)` for exactly:

```text
decision.json
fixture.json
contract.test.mjs
harness.mjs
README.md
manifest.json
```

`manifest.json` covers the other five files with sorted paths, modes, and SHA-256 byte digests; it is byte integrity, not authenticity. Output is LF UTF-8 with all defaults materialized and contains no current time, random value, absolute path, user name, temp port, candidate/ref/branch name, secret, host formatter drift, Countershape import, Wake artifact, didrun interpretation, package install, shell command, or external service. Receipt references are inert verbatim strings; missing is `UNRECEIPTED`.

## Implementation passes

1. Define the closed predicate AST and typed values used by both proof domains: tagged missing/absent, null, Boolean, safe integer encoded canonically, exact supported strings/bytes, ordered lists/multimaps, and only the strict JSON paths actually exercised. Preserve absent versus present-empty, empty bytes versus missing, exit versus signal, and ordered duplicate headers. Defer unsupported decimals/nesting rather than relying on `JSON.parse` coercion.
2. Implement a pure compiler that validates readiness, freshness, current lineage, target, selected fields, separation, path containment, secret/external bindings, and deterministic artifact naming before returning any bytes. Compile twice in different absolute roots, locale/timezone/newline/umask settings and require identical path/mode/bytes.
3. Generate readable `decision.json`, inert `fixture.json`, a small ordinary `node:test` entry point, a vendored versioned `harness.mjs`, limitations/run instructions, and the integrity manifest. All runtime reads resolve relative to `import.meta.url` and the configured repository-root relation. Moving both repository and bundle must not change semantics.
4. Implement the CLI harness with explicit argv arrays, `shell: false`, sparse environment, private `HOME`/`TMPDIR`/state/fixture roots, bounded stdin/stdout/stderr, direct repo-relative executable resolution, and disjoint exit/signal results. `{node}` may resolve only to `process.execPath`; no ambient executable lookup, login shell, globbing, user config, or package manager.
5. Implement the HTTP harness with one repo-relative local service, literal `127.0.0.1`, unique port, fixture-owned readiness, one cleartext HTTP/1.1 request through Node core `http.request`, explicit ordered header multimap/body caps, and no TLS, redirect, cookies, compression, proxy, WebSocket, streaming body, external hostname, or third-party API. Teardown completes before eligibility.
6. Keep top-level runtime categories disjoint. `ELIGIBLE_OBSERVATION` becomes `CONFORMS` or `CONTRADICTS`. Start/readiness/transport/timeout/output/projection/orphan/teardown failures become typed `INELIGIBLE_EXECUTION`. HTTP `500`, CLI exit `2`, absent optional field, or empty complete stdout may be eligible. Both contradiction and harness failure may make `node --test` nonzero, but TAP diagnostics and stable machine codes must distinguish them.
7. Append a new `ContractExecution` artifact for the current checkout with bundle digest, measured runtime facts, eligible result or typed ineligibility, and optional current Git identity. Never edit the DecisionRecord or confirmation evidence.

## Normative Go/Node parity gate

Create one checked-in byte/result vector corpus consumed by both implementations. It must cover strict parsing, selected-field extraction, exact tuple canonicalization, missing/empty, CLI completion, HTTP status/header multimap/body projection, eligible/ineligible taxonomy, and expected errors. Include duplicate JSON names, unsafe integers, negative zero, lone surrogates, invalid UTF-8, NUL/truncation, JSON key reordering, array reordering, duplicate headers, absent/empty fields, exit `2`, signal termination, HTTP `500`, refused connection, hung readiness, output cap, and teardown/orphan risk.

Kill mutations for: `JSON.parse` last-key-wins; numeric coercion; missing equals empty; signal equals exit; duplicate headers joined; allowed tuple chosen by majority/name/first; substring/regex/tolerance instead of exact membership; allow-many cross-product; assertion of an unselected header/body/stderr/file; shell execution; Countershape import/service call; package-manager or external-host access; manifest bypass; receipt-grade interpretation; and harness failure flattened to contradiction. Any Go/Node disagreement blocks standalone.

## Physical standalone and end-to-end acceptance

For both CLI and HTTP generated bundles, use a clean, separately created target checkout containing only repository fixture code plus the six files. Countershape binaries/source/packages/services and `node_modules` must be absent; remove Countershape from `PATH`; configure package/registry access unavailable; run directly as `node --test <contract.test.mjs>`. Loopback remains available only for the HTTP subject. Do not call this hostile containment or network blocking unless an independent mechanism was actually receipted.

Prove: byte identity across roots; integrity tampering stops before spawn; relocation works; fake AWS/npm/SSH/Git/cloud parent secrets are not inherited; unselected field changes still conform; each selected field change contradicts; HTTP `404` conforms while `403`/metadata `200` contradict; a custom `401` expectation makes every eligible original candidate contradict; CLI selected precedence output conforms while diagnostics may vary; `REJECT_ALL`/defer/refine/stale inputs create no executable file. Run the full matrix only for installed Node majors and native operating systems, naming each separately. Cross-compilation is not runtime evidence.

## didrun, commit, and handoff

Every load-bearing Go test, Node vector run, mutation suite, standalone absence run, relocation, secret-environment check, race/vet/lint command, and target matrix entry must be `didrun run -- <command>`. Inspect indexes with `didrun show --session`; bind narrow claims to exact successful events and paths. Unrun Node majors, Linux, Windows, external-network denial, hostile security, maintainability, and framework portability are `UNRECEIPTED` or explicit nonclaims.

Stage only U6 residue changes, commit without an AI co-author trailer, `didrun seal --commit HEAD`, and loop on `NO_COLOR=1 didrun verify --strict` until exit 0. On failure, repair the real defect, rerun affected commands through didrun, re-claim, commit, reseal, and reverify. Never delete claims, weaken vectors/mutants, or upgrade a grade.

Update the Mode C handoff with bundle schema, executed Node/OS matrix, physical-absence method, exact commands/events/grades, commit, strict verdict, and risks. Stop at `DecisionRecord`-only residue if parity, separation, deterministic bytes, typed ineligibility, CAS freshness, or physical absence fails. A pretty generated test is not standalone evidence.
