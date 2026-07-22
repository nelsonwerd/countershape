# P07B-C C4V — execution-contract maintenance

Classification: `DEFECT_REPAIR`

C4V owns exactly these 11 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/THREAT_MODEL.md`, `docs/VERIFICATION.md`, `docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md`, `docs/status/P07B-C-C4V-EXECUTION-CONTRACT-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, `tools/check-p07b-c-unit-scope.mjs`, `tools/verify-current-selftest.mjs`, and `tools/verify-current.mjs`. Their sorted-newline roster digest is `sha256:535a64fbb9f0d3c3672b4b8197fc7815da315117b1794a668e6565faa16df3f1`.

- **Boundary:** active pre-seal `SOURCE_FULL` execution-contract maintenance after sealed C3D and before C4. Classification: `DEFECT_REPAIR`.
- **Parent:** sealed C3D commit `62422a2400edd702c82fb680c6ef0c6923eb6cfb`, tree `7573f57017c6f56b434d1fbd5642ac585f7789c2`, is note-present with note blob `b9e6e4abe2e3e82a294f0cbfbb686b6a7d8d232e`, note-body SHA-256 `b78873dbc59a3c81ec14f5894bee090ed3c3e8dc868669fd25e557ea4cf9ae6c`, `10/10 claims recorded-exact`, strict exit `0`, and recorded `secrets_override: true`.
- **Commit subject:** `fix: declare executable C4 and C5 contracts`
- **Profile:** `SOURCE_FULL`; C4V edits checker/specification authority and therefore cannot use the receipt-only battery.
- **Receipts:** C3P `PRESENT`; C3 `PRESENT`; C6A `ABSENT`.
- **Product authority:** none. C4V changes no Go product source, process behavior, official target, permit, finalized run, classification, or didrun installation. It does change cumulative-verifier execution: direct build, vet, and general-package jobs revert from `p=2` to `p=1`; every other verifier authority remains fixed.

Every intended C4V grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C4V itself.

## Live S6 verifier-interruption evidence

The C4V development ledger retains the interruption honestly. Event 93 records a nonzero cumulative-verifier refusal `VERIFY_STALE_LOCK` for absent owner PID `12427` and supports no claim. Event 94 records only the didrun-wrapped exact removal `/bin/rm -- .countershape/verify-current/active.lock`; it is operational history, not capability evidence. A later verifier wrote a new mode-`0600`, current-uid lock for PID `21502` about 0.4 seconds after that recovery, but its outer wrapper was interrupted before didrun appended an event. PID `21502` is now absent and no open file holder was observed, so that run has no event, receipt, or grade. After an immediate process/holder recheck, development event 120 records the exact reviewed removal of only that stale lock. No residual `run-*` directory was bulk-deleted.

This live session shows both sides of the mechanism: the O_EXCL lock fails closed against overlapping ownership, while interrupted-owner recovery still needs manual review and an exact recorded cleanup. The final C4V ledger must be fresh and must complete its own cumulative verifier normally; events 93, 94, and 120 cannot support any final claim.

## Recurrent AST deadline and conservative verifier rollback

C4V final attempt 1 is permanent failed evidence at `.didrun-history/p07b-c-c4v-final-attempt-1-ast-sigterm/.didrun/`; its retained private root is `.countershape/p07bc-c4v-final-attempt-1-ast-sigterm`. It contains seven events and six claims. Unclaimed event 6 exited `1` in cumulative row `23/46`, `architecture-p07b-a2-2-selftest`, after `257205` ms: the `human-capture-byte-drift` run reached the unchanged 60-second inner AST-child deadline and surfaced `P07B_A2_JS_AST_QUERY_FAILED: SIGTERM`. The raw digest violation had already been detected; this is not evidence that the hostile survived. That attempt produced no commit, seal, Git note, strict result, HTML report, or C4V grade, and its events 0–5 are never reusable.

The earlier development event 24 failed the same late A2 row at a different hostile, while an unchanged later run passed. Two independent late occurrences fire the explicit recurrence clause sealed in `docs/status/P07B-C-VERIFICATION-THROUGHPUT.md`: current tiering is rejected. C4V therefore changes only `generalJobs` from two to one for direct build, vet, and general-package testing. It preserves `GOMAXPROCS=2`, test `-parallel=2`, sensitive/nested `p=1`, fresh cache, the exclusive lock, every timeout, package roster, ordering, and assertion. C1V's p2 receipt remains accurate historical evidence; this rollback is a policy-prescribed stability fallback, not proof that p2 caused either timeout. If the unchanged AST deadline recurs under p1, C4V stays open and the next repair must optimize or correctly classify AST work rather than lengthen deadlines or weaken the hostile roster.

The superseded development ledger remains `.didrun-history/p07b-c-c4v-development-c63712d69852-134/.didrun/`. `.countershape/c4v-a2-retry` is unclaimed diagnostic state and cannot support a final claim. The replacement final boundary restarts at event zero with 68 exact claims: 65 `tests-pass`, then three `command-succeeded`.

## Why this maintenance boundary exists

The frozen C4 row could not honestly implement or verify its promised source boundary:

1. The store bridge needed to change the frozen public API test, but `internal/store/public_api_test.go` was outside the old roster.
2. A real process-mechanics extraction needed the owning `internal/world` implementation and tests; copying mechanics would have created two physical authorities.
3. The inherited B and U6 architecture gates rejected the new packages and surfaces unless their exact checkers and hostile self-tests evolved in the same unit.
   C4 is the final writer of B before C5. C4V must first seal an independent schema/scanner and the exact historical inventory; C4 then creates a canonical manifest binding its real delta and predeclaring C5 rows, while C5 owns neither manifest nor checker. Separate runner/HTTP/scope topology and exact row matching reject C5-without-C4, partial/mixed/lookalike states, missing or extra reserved tokens, and schema/digest/order drift.
4. The U2/U3 mutation drivers pinned the old world-process source and would otherwise silently stop testing the relocated owner.
5. C4 changes load-sensitive package ownership and the repetition roster, so the frozen C1V 52-case matrix is insufficient; the C4 contract must requalify the expanded 54-case matrix.
6. C4 had no executable final runbook, sealed-predecessor note gate, or preseal ledger contract.
7. C5 owns its C architecture checker and self-test, but owns neither the B future-surface manifest/checker, phase plan/scope checker, phase specification, repetition helper, nor runtime authority; it may only satisfy the parent-sealed rows/topology. Without a C4V-sealed 79-command runbook, sealed-C4 consumer, and C4-provided dormant 56-case catalog it would have no non-circular way to close.
8. The repetition helper imported lock, tool-admission, private-root, child-environment, execution, revalidation, and cleanup functions from C5-owned `tools/verify-current.mjs`. That made the claimed C4-sealed C5 qualification runtime mutable through an indirect import even though C5 could not edit the helper itself.

These are contract defects, not reasons to weaken an existing gate. C4V repairs the declaration before product source starts.

## Receipt-machinery proposal disposition

The later receipt-phase machinery proposal is routed without another boundary:

- `DEFECT_ALREADY_REPAIRED`: sealed C3D commit `62422a2400edd702c82fb680c6ef0c6923eb6cfb` is the requested `SOURCE_FULL` phase-table/generic-fixture consolidation.
- `INTENTIONAL_HYBRID`: structural cases are generated from the table; four named sealed-history regressions remain explicit semantic oracles.
- `DECLINED_WITH_REASON`: a pure boundary/parent/profile/receipt-state row cannot encode sealed-note argv projection, C3Q's outer-versus-historical Markdown-scanner asymmetry, C3T's validated absent-state lifecycle, or C3U's real legacy-caller terminal-byte preservation. Moving that logic into more table columns would disguise, not remove, bespoke semantic authority.

The retained fixed regressions are `C3R_NOTE_PREVIEW`, `C3Q_EXTERNAL_SELF_RECEIPT`, `C3T_VALIDATED_ABSENT_BASELINE`, and `C3U_TERMINAL_RECEIPT_LAYERING`. Explicit forward-horizon allowlists also remain independently implemented in both checkers so a staged table cannot authorize its own new boundary.

## v17 phase authority and fixed acceptance

The current schema is `countershape/p07b-c-unit-paths/v17`. It contains 19 ordered rows and has complete authority digest `sha256:11f96a7809d845ef21671d8f847bd3b9f324e84c539cd2f93ab2bc14929ecd46`.

- Generic phase catalog: 722 cases, exactly 361 accepted controls and 361 hostile rejections; digest `c31ebebeac2fdc5d522b466ab082bcd60c5da5b488b7c5be96b7c6cfe1d15aac`.
- Combined executable catalog: 726 cases after the four fixed regressions; digest `bb433b824bc306bb02138122e1e563f2584c4862766d9721b4881d5f1aa23132`.
- Combined acceptance split: exactly 365 accepted controls and 361 hostile rejections.
- Plan candidate-transition matrix: exactly 7 accepted and 41 rejected.
- Independent unit-scope transition matrix: exactly 7 accepted and 24 rejected.
- Frozen historical self-receipt matrix: 1,320 rejections, 440 core accepted controls, eight supplemental aliases, and two strict-zero prohibition controls.

C3Q's current-surface oracle uses the active phase dynamically, but its historical C3B external-Markdown replay remains fixed. The sealed C3D v16 specification, 666-case generic catalog, 670-case combined catalog, status, commit, tree, note, and historical digests remain immutable evidence.

The current positive-receipt refusal corpus contains 46 sorted Markdown authorities at `sha256:b480bb8e93d9ddd6075a57bdf8dbe99cb663eb1f0a11228af31698b54a5dd2f2`. Both the sealed C3D status and active C4V status are required. The first 45-path development draft omitted C3D and the cumulative forward-surface self-test rejected it; C4V repairs the corpus rather than weakening that regression. The sealed C3U/C3B 44-path historical matrix remains byte-for-byte unchanged.

## Corrected C4 ownership

C4 owns these 38 exact paths:

1. `docs/ARCHITECTURE.md`
2. `docs/CLAIM_VOCABULARY.md`
3. `docs/HANDOFF_MODE_C.md`
4. `docs/PROMPT_PACK.md`
5. `docs/SEMANTICS.md`
6. `docs/STATE_MACHINES.md`
7. `docs/THREAT_MODEL.md`
8. `docs/VERIFICATION.md`
9. `docs/status/P07B-C-C4-CLI-PROFILE.md`
10. `internal/store/contract_run_bridge.go`
11. `internal/store/contract_run_bridge_test.go`
12. `internal/store/execution_interlock.go`
13. `internal/store/execution_interlock_test.go`
14. `internal/store/nonhead_contract.go`
15. `internal/store/nonhead_contract_test.go`
16. `internal/store/private_contract_run.go`
17. `internal/store/private_contract_run_test.go`
18. `internal/store/public_api_test.go`
19. `internal/world/capture.go`
20. `internal/world/process.go`
21. `internal/world/process_darwin.go`
22. `internal/world/process_darwin_test.go`
23. `internal/world/process_mutation_darwin_test.go`
24. `internal/world/process_unsupported.go`
25. `spec/verification/p07b-b-future-surface-authority.json`
26. `tools/check-p07b-b-architecture-selftest.mjs`
27. `tools/check-p07b-b-architecture.mjs`
28. `tools/check-p07b-c-architecture-selftest.mjs`
29. `tools/check-p07b-c-architecture.mjs`
30. `tools/check-u6-architecture-selftest.mjs`
31. `tools/check-u6-architecture.mjs`
32. `tools/mutate-u2.mjs`
33. `tools/mutate-u3.mjs`
34. `tools/test-mutate-u2.mjs`
35. `tools/verify-current-selftest.mjs`
36. `tools/verify-current.mjs`
37. `tools/verify-go-test-repetition.mjs`
38. `tools/verify-runtime-authority.mjs`

The sorted-newline exact-path digest is `sha256:99bda0d3e5c2f2a2ee4f354945188bf2e5563716c9e26511211a4bfc9197bf56`.

C4 additionally owns these three directory prefixes:

- `internal/contractexec/runner/`
- `internal/processmechanics/`
- `testkit/contractexec/cli/`

Their sorted-newline prefix digest is `sha256:f5f7b8cdfeba3581e8632c0d6686287f5afa00aef224bb4babe39ee1fdc7f4b2`.

## Frozen B future-surface contract

C4V freezes the independent production-Go scanner and manifest schema at `sha256:91c4ca13a4b30a683b82066fdcdc1a7892de2679a2e5419a57142e36b22f5937`, including a canonical baseline-policy projection independently recomputed at `sha256:5a1c3e2f33969143765612f4934ce4ca41cc1e3df95383b8d7780354461e3399`, the exact live-and-strict-scanned 27-row sealed-C3D historical inventory at `sha256:901d7cb5ca65cb9cb1ca7932ece1741fbfc82df5f9e375d43fadd6e057615960`, allowed C4/C5 locations, and required C4 bridge row. It does not claim to know C5's final row inventory.

The C4V plan gate scans that baseline before the future manifest exists: ordinary coherence uses stable no-follow live reads, while the candidate gate uses only the pinned full-index snapshot. The self-test carries separate positive live and strict-snapshot baselines plus exactly 42 direct refusal controls, including wrong mode, missing blob, fatal UTF-8, historical deletion, extra reserved token, premature topology, malformed non-string path, and C5 parent-manifest preservation cases. Four additional C4/C5 full-plan hostiles mutate manifest and private fixture-projected observed-row inputs through the outer `checkPlan()` route, so a disconnected validator call cannot leave the direct controls green; those forward source rows remain synthetic and are not future-source conformance evidence.

The named admitted states remain `C4-present/C5-absent` and `C4+C5-present`. C4 creates canonical `spec/verification/p07b-b-future-surface-authority.json`. The C4V-owned plan checker independently scans mode-`100644`, fatal-UTF-8, production non-test Go files below `internal/`, subtracts only the frozen historical rows, and requires the remaining reserved-family tokens to equal sorted unique `{boundary,path,symbol}` manifest rows. At C4, observed rows equal the C4 rows, runner topology is present, and HTTP/scope topology is absent. C4 predeclares exact C5 rows, which may be empty. At C5 and later, the manifest must remain byte-for-byte equal to sealed C4, observed rows equal the combined C4+C5 rows, and runner, HTTP, and scope topology are all present. C5 owns neither manifest, B checker/self-test, nor plan checker. Historical deletion, C5-without-C4, partial/mixed topology, missing/extra/relocated/lookalike tokens, wrong row boundary/order/schema/digest, and prefix lookalikes fail.

Private `Symbol`-keyed forward fixtures cannot coexist with a strict staged snapshot and are unreachable from the production CLI. They prove validator/state-machine refusals only, never future source existence or conformance. The single known required-row digest `sha256:4ebfb5ae2608f0cdaa2e316e087fd8c8e479a891e7e52d0a7aab7f9a45541807` is explicitly `TEST_VECTOR_ONLY/INCOMPLETE`; C4's future concrete manifest, not that vector, becomes instance authority.

## C4 architecture and correctness direction

The low-level boundary is staged:

```text
Prepare(request) -> Prepared
runner reopens target and runtime
runner consumes private one-shot RunPermit
Prepared.Start() -> StartError | (SpawnObservation, Running)
runner persists SpawnObservation
Running.Close() -> wait/drain/teardown/final-probe facts
```

`processmechanics` receives no permit, target, target digest, semantic body, or public raw argv. `world` becomes a compatibility adapter over neutral mechanics. Only the higher runner consumes the store admission edge immediately adjacent to `Start`.

The store bridge uses opaque values with private fields: `ContractRunOwner`, `PrivateRunManifest`, `FinalizedRunRecord`, `TerminalClosure`, and `ContractExecutionRecord`. Model conversion remains in `nonhead_contract.go`. Release must also converge the exact deterministic `FINALIZED_RUN` receipt when durable `CLEAR` exists but receipt publication failed; it may reconstruct only the immediately prior exact `HELD` state and must refuse every mismatch. Changed-boot reset remains a separate explicit operation and is unreachable from ordinary run, same-boot retry, timeout, or takeover logic.

No arbitrary Node execution becomes `COMPLETE` from declarations, static absence, generated metadata, TAP, preload hooks, or child-authored claims. Noncooperative targets may honestly yield `MISSING` or `PARTIAL`; reference fixtures must exercise genuine clean and forbidden-positive detector paths.

## Frozen qualification runtime authority

C4 adds `tools/verify-runtime-authority.mjs` as an exact added source path. That module imports Node built-ins only and owns the shared lock/tool admission, fresh private roots, hermetic child environment, bounded execution, authority revalidation, failure cleanup, and success-finalization primitives. Both `tools/verify-current.mjs` and `tools/verify-go-test-repetition.mjs` have exactly one relative named-import edge, to that runtime. The repetition helper may not import C5-owned `verify-current.mjs` directly or indirectly, and it owns a separate qualification-sensitive package roster rather than accepting caller-controlled classification.

The plan checker has no top-level repetition-helper import. Through C4V it reads the stage-equal mode-`100644` helper bytes and requires exact legacy SHA-256 `1321fb1381f1f74d0959b9ceb40b4dcc4abc637bb281ff2af759863ede3bfd5a` before lazy import. At live C4 and C5 it reads pinned stage-equal helper, runtime, and verifier bytes and runs a bounded sterile Acorn check. At live C4/C5, `qualificationStaticAuthority` decodes the recursively frozen literal catalog into a checker-owned projection and never imports or evaluates future candidate helper bytes. The static gate does not execute candidate parser, selector, validator, or builder exports; actual execution evidence is the separate helper self-test, per-case receipts, and preseal reconciliation. The helper's sole relative named-import edge must be `./verify-runtime-authority.mjs`; the runtime admits only frozen Node built-ins and no relative edge; the verifier's sole relative named-import edge is the runtime. Re-exports, dynamic import, CommonJS/dynamic-evaluation loader forms including `process.getBuiltinModule`, side-effect/aliased imports, effectful module initialization, protected-binding drift, authority-wrapper forwarding, and noncanonical direct entry fail. This is static source/import conformance, not a sandbox or confinement claim.

The helper exports recursively frozen literal `qualificationMatrices` C4 and C5 records, their exact digests, exact C4 identity aliases, and stable `qualificationCaseForID(caseID)`. Its exact route is parse → canonicalize → validate → build → run. `parseRunArguments(argv)` begins with the exact `argv.length === 2 && argv[0] === "--case"` selector branch, then an exact throwing ad-hoc preamble for minimum/even cardinality and `--package`/`--profile`/`--count`/`--run` positions. `validateExecutionSpecification(specification)` starts with named selector canonicalization and identity refusal. `buildGoTestArguments(specification)` returns the exact frozen 11-element Go argv: `test -json -mod=readonly -buildvcs=false`, profile-bound `-p`, `-parallel=2`, exact count, `-timeout=20m`, run, and package. `runRepetition(specification, dependencies = {})` begins with `const executionSpecification = validateExecutionSpecification(specification)`, builds only from that object, and passes the same argv to its dependency-bound child. Async `main()` ends with `await runRepetition(parseRunArguments(process.argv.slice(2)))` before the final exact Node-import-bound guard awaits `main()`. The checker statically validates this source route and its checker-owned projection; its C4V self-test rejects module-closure, catalog, alias, selector, parser, validator, builder, child-route, and direct-entry drift.

Qualification remains case-isolated. Each of C4's 54 and C5's 56 cases runs in a separate didrun event with its own immediate claim, lock, admitted tool snapshot, fresh private roots/environment, bounded child execution, Go JSON validation, cleanup/finalization, and terminal evidence. There is no shared session, shared root, aggregate matrix receipt, or cross-case failure policy. The C4/C5 preseal gate reopens each didrun stdout blob, verifies its content digest, the independently derived exact ordered `[go,node,git,sh,cc,cxx]` roster plus the SHA-256 digest of UTF-8 `JSON.stringify(roster)`, and the exact terminal case/package/profile/count/test result. During C4, the helper self-test drives both dormant C5-only identities—`contract-http-readiness-50` and `contract-http-teardown-20`—through the exported parser, selector, `validateExecutionSpecification(specification)`, and `buildGoTestArguments(specification)`, asserts stable identity plus exact frozen argv, and exercises `runRepetition(specification, dependencies = {})` with injected dependencies that acquire no runtime authority and spawn no absent C5 package. Unknown IDs, wrong flags, extra arguments, copied specifications, raw alias lookup, and builder drift fail before child execution. Runtime lifecycle correctness remains in the C4-owned runtime and repetition-helper self-tests, while the checker proves the stable source/catalog route without pretending to sandbox code.

At preseal reconciliation, the checker independently derives exact ordered `[go,node,git,sh,cc,cxx]` from frozen `COUNTERSHAPE_GO/NODE/GIT/SH/CC/CXX`; realpaths each; requires executable regular non-symlink files; performs bounded no-follow raw reads up to 512 MiB with before/after identity; SHA-256 fingerprints each; and requires Node realpath to equal the invoking checker. Every 54/56 stdout blob must match that admitted roster exactly; the terminal digest is SHA-256 of UTF-8 `JSON.stringify(roster)`; the roster is reopened after all blobs; and a self-consistent foreign stdout roster fails. This is bounded preseal-time independent admission plus post-blob re-admission and recorded per-event output, not proof of continuous executable-byte identity between event execution and reconciliation, across events, or against same-uid swap-and-restore/TOCTOU.

## C4V intended claim map

The replacement ledger is exact: 68 claims, with 65 `tests-pass` claims followed by three `command-succeeded` claims. Every qualification, A2 stability pass, and cumulative pass is a separate event and immediate claim.

| Claim | Type | Intended grade |
| --- | --- | --- |
| `P07B-C C4V execution-contract phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V execution-contract defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V sealed-C3D Git-note and ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V focused repetition helper self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification world-output-caps-50` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification world-output-independence-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification world-simultaneous-overflow-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification world-lifecycle-readiness-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification compiler-generated-runtime-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification program-lifecycle-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification store-cross-process-cas-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-01-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-02-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-03-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-04-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-05-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-06-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-07-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-08-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-09-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-10-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-11-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-12-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-13-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-14-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-15-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-16-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-17-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-18-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-19-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-reducer-20-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-01-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-02-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-03-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-04-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-05-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-06-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-07-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-08-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-09-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-10-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-11-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-12-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-13-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-14-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-15-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-16-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-17-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-18-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-19-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-reducer-20-of-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification parity-evaluator-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification parity-framing-20` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification parity-full-package-3` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification cli-physical-full-package-3` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V qualification http-physical-full-package-3` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V A2 architecture stability pass 1` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V A2 architecture stability pass 2` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V A2 architecture stability pass 3` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V cumulative verification pass 1` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V cumulative verification pass 2` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V cumulative verification pass 3` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4V exact eleven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C4V scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C4V sealed-C3D predecessor and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Final command order

These are command tails, not bare receipt commands. Each runs only through `didrun run --` with the checker-owned hermetic prefix and is claimed immediately with the corresponding label/type above. Any nonzero event archives the complete attempt and requires a fresh restart from command 1.

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C4V`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C4V`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
4. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4v-sealed-c3d-note`
5. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
6. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
7. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --self-test`
8. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case world-output-caps-50`
9. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case world-output-independence-20`
10. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case world-simultaneous-overflow-20`
11. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case world-lifecycle-readiness-20`
12. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case compiler-generated-runtime-20`
13. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case program-lifecycle-20`
14. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case store-cross-process-cas-20`
15. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-01-of-20`
16. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-02-of-20`
17. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-03-of-20`
18. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-04-of-20`
19. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-05-of-20`
20. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-06-of-20`
21. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-07-of-20`
22. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-08-of-20`
23. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-09-of-20`
24. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-10-of-20`
25. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-11-of-20`
26. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-12-of-20`
27. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-13-of-20`
28. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-14-of-20`
29. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-15-of-20`
30. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-16-of-20`
31. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-17-of-20`
32. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-18-of-20`
33. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-19-of-20`
34. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-20-of-20`
35. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-01-of-20`
36. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-02-of-20`
37. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-03-of-20`
38. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-04-of-20`
39. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-05-of-20`
40. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-06-of-20`
41. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-07-of-20`
42. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-08-of-20`
43. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-09-of-20`
44. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-10-of-20`
45. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-11-of-20`
46. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-12-of-20`
47. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-13-of-20`
48. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-14-of-20`
49. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-15-of-20`
50. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-16-of-20`
51. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-17-of-20`
52. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-18-of-20`
53. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-19-of-20`
54. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-20-of-20`
55. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case parity-evaluator-20`
56. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case parity-framing-20`
57. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case parity-full-package-3`
58. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-full-package-3`
59. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-full-package-3`
60. `/opt/homebrew/bin/node tools/check-p07b-a2-architecture-selftest.mjs`
61. `/opt/homebrew/bin/node tools/check-p07b-a2-architecture-selftest.mjs`
62. `/opt/homebrew/bin/node tools/check-p07b-a2-architecture-selftest.mjs`
63. `/opt/homebrew/bin/node tools/verify-current.mjs`
64. `/opt/homebrew/bin/node tools/verify-current.mjs`
65. `/opt/homebrew/bin/node tools/verify-current.mjs`
66. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4V --source-final-gate`
67. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4V --credential-scan`
68. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4v-preseal-ledger`

The private final root is `.countershape/p07bc-c4v-final`. The final operator uses `--print-final-runbook C4V`; the renderer fixes repository-root entry, isolated `set -eu` fences, exact hermetic argv, immediate claims, commit subject, plain-seal-first order, reviewed secrets override only at a manual checkpoint, readable note inspection, strict verification, HTML generation, and commit-addressed ledger archiving. Before any `mkdir`, `chmod`, or archive copy, preparation and archive fences set `umask 077` and invoke `--verify-final-runbook-parent-authority` through sterile `/usr/bin/env -i`; the guard requires the canonical repository root plus existing `.countershape` and `.didrun-history` parents to be real directories, non-symlinks, owned by the current uid, and not group/other writable, and an existing `.countershape/evidence` to have exact mode `0700`.

## Frozen C4 final verification contract

C4's generated runbook has exactly 80 commands: 77 `tests-pass` claims followed by three `command-succeeded` claims. It includes five focused functional profiles, one exact race/boundary-fault profile, C/B/U6 architecture plus self-tests, U2/U3 mutation-driver compatibility, repetition-helper self-test, 54 separate qualification-case receipts, sealed-C4V/C3D ancestry, scope/verifier self-tests, three complete cumulative passes, exact staged scope, credential scan, and preseal chain integrity.

The 54-case matrix preserves every frozen C1V semantic case while relocating the three world output/capture cases to `internal/processmechanics`, retaining `world-lifecycle-readiness-20`, and adding `contract-runner-admission-50` plus `contract-cli-standalone-closure-20`. Its exact digest is `sha256:3be703fae10155c83b19be54ae7cd79cfc5b1bbaddfcf86bb2c9781ad93f853a`. C4 may not relabel the compatibility-only U2/U3 checks as full mutation-kill evidence. The machine-audited C4 race profile must retain the exact declared passing roster and race instrumentation.

### C4 exact final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C4`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C4`
3. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --run-go-json c4-processmechanics-parity`
4. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --run-go-json c4-admission-permit`
5. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --run-go-json c4-cli-closure`
6. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --run-go-json c4-finalized-run-release`
7. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --run-go-json c4-classification-recovery`
8. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --run-go-json c4-authority-race`
9. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --c4`
10. `/opt/homebrew/bin/node tools/check-p07b-c-architecture-selftest.mjs --c4`
11. `/opt/homebrew/bin/node tools/check-u6-architecture.mjs`
12. `/opt/homebrew/bin/node tools/check-u6-architecture-selftest.mjs`
13. `/opt/homebrew/bin/node tools/check-p07b-b-architecture.mjs`
14. `/opt/homebrew/bin/node tools/check-p07b-b-architecture-selftest.mjs`
15. `/opt/homebrew/bin/node --test tools/test-mutate-u2.mjs`
16. `/opt/homebrew/bin/node tools/mutate-u3.mjs --self-test`
17. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --self-test`
18. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case processmechanics-output-caps-50`
19. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case processmechanics-output-independence-20`
20. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case processmechanics-simultaneous-overflow-20`
21. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case world-lifecycle-readiness-20`
22. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case contract-runner-admission-50`
23. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case contract-cli-standalone-closure-20`
24. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case compiler-generated-runtime-20`
25. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case program-lifecycle-20`
26. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case store-cross-process-cas-20`
27. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-01-of-20`
28. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-02-of-20`
29. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-03-of-20`
30. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-04-of-20`
31. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-05-of-20`
32. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-06-of-20`
33. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-07-of-20`
34. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-08-of-20`
35. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-09-of-20`
36. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-10-of-20`
37. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-11-of-20`
38. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-12-of-20`
39. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-13-of-20`
40. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-14-of-20`
41. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-15-of-20`
42. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-16-of-20`
43. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-17-of-20`
44. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-18-of-20`
45. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-19-of-20`
46. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-20-of-20`
47. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-01-of-20`
48. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-02-of-20`
49. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-03-of-20`
50. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-04-of-20`
51. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-05-of-20`
52. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-06-of-20`
53. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-07-of-20`
54. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-08-of-20`
55. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-09-of-20`
56. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-10-of-20`
57. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-11-of-20`
58. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-12-of-20`
59. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-13-of-20`
60. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-14-of-20`
61. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-15-of-20`
62. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-16-of-20`
63. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-17-of-20`
64. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-18-of-20`
65. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-19-of-20`
66. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-20-of-20`
67. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case parity-evaluator-20`
68. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case parity-framing-20`
69. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case parity-full-package-3`
70. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-full-package-3`
71. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-full-package-3`
72. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4-sealed-c4v-note`
73. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
74. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
75. `/opt/homebrew/bin/node tools/verify-current.mjs`
76. `/opt/homebrew/bin/node tools/verify-current.mjs`
77. `/opt/homebrew/bin/node tools/verify-current.mjs`
78. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4 --source-final-gate`
79. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4 --credential-scan`
80. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4-preseal-ledger`

## Frozen C5 source and final verification contract

C5 owns exactly these 12 paths: `docs/ARCHITECTURE.md`, `docs/CLAIM_VOCABULARY.md`, `docs/HANDOFF_MODE_C.md`, `docs/SEMANTICS.md`, `docs/STATE_MACHINES.md`, `docs/THREAT_MODEL.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C5-HTTP-SCOPE.md`, `tools/check-p07b-c-architecture-selftest.mjs`, `tools/check-p07b-c-architecture.mjs`, `tools/verify-current-selftest.mjs`, and `tools/verify-current.mjs`. Their sorted-newline roster digest is `sha256:d16ba74582e53dc5c791b8c34fbcd6200338dc066ec13230c2908eb6ba257bf5`. `docs/VERIFICATION.md` is deliberately owned because C5 evolves the current verifier; C6A owns that manual for the same reason.

Its three required, realized prefixes are `internal/contractexec/http/`, `internal/contractexec/scope/`, and `testkit/contractexec/http/`, with sorted-newline digest `sha256:a9212680066fdbddc8f25eef04a41c264ac9148a506238dd92c39978e3e35f84`. C5 owns its C architecture checker and self-test, but owns neither `spec/verification/p07b-b-future-surface-authority.json`, the B checker/self-test, the phase plan/scope checker, phase specification, repetition helper, nor `tools/verify-runtime-authority.mjs`; it may only satisfy the parent-sealed rows/topology. Its exact commit subject is `feat: complete standalone contract execution`, direct parent is sealed C4, and private final root is `.countershape/p07bc-c5-final`.

C4 must seal a dormant exact 56-case catalog and stable selector. The catalog is the C4 54-case catalog plus, immediately after `contract-cli-standalone-closure-20`, `contract-http-readiness-50` in `testkit/contractexec/http` with count 50 and exact test `TestHTTPChildReportedReadinessBindsExactService`, then `contract-http-teardown-20` with count 20 and exact test `TestHTTPEarlyExitAndTeardownRetainCausalFacts`. Its canonical JSON digest is `sha256:08db7c338eb3ba900cf6df2545c04f27bed16889c81dd16e5fb18197c4d52958`. The C4 helper self-test must prove the dormant C5-only identities through parser/selector/validator/builder plus a no-authority dependency-injected run; C5 later executes all 56 identities with separate `--case <id>` receipts without owning the helper bytes.

C5's generated runbook has exactly 79 commands: 76 `tests-pass` claims followed by three `command-succeeded` claims. `--verify-c5-sealed-c4-note` requires the current one-parent C4 commit, exact subject, frozen source A/M partition, all three realized prefixes, exact 80-claim note and argv/grade/coverage, and direct C4V/C3D ancestry with one stable outer HEAD. `--verify-c5-preseal-ledger` binds that authority to the exact fresh C5 ledger and unchanged C3P/C3-present/C6A-absent receipt state.

### C5 exact final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C5`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C5`
3. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --run-go-json c5-http-behavior`
4. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --run-go-json c5-readiness-teardown`
5. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --run-go-json c5-scope-closure`
6. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --run-go-json c5-cross-profile-parity`
7. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --run-go-json c5-http-authority-race`
8. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --c5`
9. `/opt/homebrew/bin/node tools/check-p07b-c-architecture-selftest.mjs --c5`
10. `/opt/homebrew/bin/node tools/check-u6-architecture.mjs`
11. `/opt/homebrew/bin/node tools/check-u6-architecture-selftest.mjs`
12. `/opt/homebrew/bin/node tools/check-p07b-b-architecture.mjs`
13. `/opt/homebrew/bin/node tools/check-p07b-b-architecture-selftest.mjs`
14. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --self-test`
15. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case processmechanics-output-caps-50`
16. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case processmechanics-output-independence-20`
17. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case processmechanics-simultaneous-overflow-20`
18. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case world-lifecycle-readiness-20`
19. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case contract-runner-admission-50`
20. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case contract-cli-standalone-closure-20`
21. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case contract-http-readiness-50`
22. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case contract-http-teardown-20`
23. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case compiler-generated-runtime-20`
24. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case program-lifecycle-20`
25. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case store-cross-process-cas-20`
26. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-01-of-20`
27. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-02-of-20`
28. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-03-of-20`
29. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-04-of-20`
30. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-05-of-20`
31. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-06-of-20`
32. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-07-of-20`
33. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-08-of-20`
34. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-09-of-20`
35. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-10-of-20`
36. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-11-of-20`
37. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-12-of-20`
38. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-13-of-20`
39. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-14-of-20`
40. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-15-of-20`
41. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-16-of-20`
42. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-17-of-20`
43. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-18-of-20`
44. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-19-of-20`
45. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-reducer-20-of-20`
46. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-01-of-20`
47. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-02-of-20`
48. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-03-of-20`
49. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-04-of-20`
50. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-05-of-20`
51. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-06-of-20`
52. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-07-of-20`
53. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-08-of-20`
54. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-09-of-20`
55. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-10-of-20`
56. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-11-of-20`
57. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-12-of-20`
58. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-13-of-20`
59. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-14-of-20`
60. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-15-of-20`
61. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-16-of-20`
62. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-17-of-20`
63. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-18-of-20`
64. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-19-of-20`
65. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-reducer-20-of-20`
66. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case parity-evaluator-20`
67. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case parity-framing-20`
68. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case parity-full-package-3`
69. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case cli-physical-full-package-3`
70. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case http-physical-full-package-3`
71. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c5-sealed-c4-note`
72. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
73. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
74. `/opt/homebrew/bin/node tools/verify-current.mjs`
75. `/opt/homebrew/bin/node tools/verify-current.mjs`
76. `/opt/homebrew/bin/node tools/verify-current.mjs`
77. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C5 --source-final-gate`
78. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C5 --credential-scan`
79. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c5-preseal-ledger`

`--print-final-runbook C4V`, `--print-final-runbook C4`, and `--print-final-runbook C5` render these exact contracts. Every C4V, C4, and C5 grade remains `UNRECEIPTED` until its own command/claim sequence, commit, seal, note, strict result, HTML report, and archived ledger exist.

## Unclaimed development evidence

Development-ledger event `24` ran the cumulative verifier for roughly 21 minutes and then recorded a load-sensitive `SIGTERM` in the inherited P07B-A2.2 architecture self-test's `runner-powerful-harness-import` hostile; the entire `architecture-p07b-a2-2-selftest` verifier row reported `duration_ms=217480`. The unchanged focused A2.2 gate had already passed. Events `25` and `26` were operator-launcher errors that omitted required hermetic variables (`COUNTERSHAPE_GO`, then `GOCACHE`) and support no product or verifier claim. Event `27` recreated the checker-owned isolated environment and passed the unchanged focused hostile. Final-attempt event 6 later reproduced the same late-row mechanism on `human-capture-byte-drift`, making the timing manifestation recurrent and firing the declared rollback rule. None of those events is claimed, sealed, or evidence for a C4V grade; all remain permanent negative/diagnostic history rather than substitutes for the replacement 68-event boundary.

## Nonclaims and handoff

If C4V seals and verifies strictly, it will establish only that the executable C4 and C5 ownership, architecture direction, transition authority, predecessor-note consumers, final evidence plans, historical receipt-fixture contract, current direct `p=1` verifier profile, 52 isolated qualification cases, three unchanged A2 self-tests, and three cumulative compositions are coherent on that sealed tree. Those local repeated receipts are not a performance ceiling, a causal attribution of the old failures to `p=2`, or a cross-platform stability proof. Before then every C4V grade remains `UNRECEIPTED`. C4V proves no subject spawn, runtime freshness, permit use, interlock release, FCR durability, classification, arbitrary-Node completeness, confinement, confidentiality, secret absence, general security, cross-platform behavior, adoption, production readiness, or maintainership.

C4 begins only after C4V independently closes. If a new receipt-machinery defect is discovered after C4V seals, an explicit interposed maintenance unit is required; no future source unit may silently edit the C4V-sealed phase table or checker authority.
