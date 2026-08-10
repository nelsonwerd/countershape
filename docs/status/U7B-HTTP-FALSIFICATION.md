# U7B HTTP falsification study

## Boundary identity

- **State:** source candidate; not committed, sealed, or receipted.
- **Namespace:** `countershape/u7-unit-paths/v2`.
- **Boundary:** `U7B`.
- **Declared parent:** exact sealed `U7A`.
- **Subject:** `feat: add U7 HTTP falsification study`.
- **Profile:** `SOURCE_FULL`.
- **Product authority:** `U7_REFERENCE_APPLICATION`.
- **Product behavior:** `CANDIDATE_UNRECEIPTED`.
- **Grade transfer:** `NONE`.
- **U7D source receipt:** `ABSENT`.
- **Final private root:** `.countershape/u7b-final`.
- **Next boundary:** `U7C` only after exact U7B commits, seals, exposes its note, and passes explicit-commit strict verification.

Exact parent U7A is commit `e178f1e7972017fb0cf8aab590d7f528a45f4bd9`, tree `5e51b2d048dc4e8e72aa0f9c5d2b5fcb86798067`, direct parent U7M `fc70bcaa51fec3991d139796c85495b6c1c75735`, and subject `feat: add U7 reference CLI foundation`. Its didrun note is blob `831856777bc831ea7ded4bc0300d3fe15717bab1`, 25,829 bytes, with note-body SHA-256 `bb907196a84312f5937a3f6ee930390fa58140225d0dfdf25d28d52e37a0ece1`. U7A closed `12/12` claims at `TREE-EXACT`, strict exit `0`, with the required high-entropy override recorded. Its local 8,434-byte HTML witness is `.countershape/evidence/u7a-final-e178f1e79720.html`, SHA-256 `5bbbfdf2c9804b819f29f1c1054218eace8e17fb32a18e77a191e44147166830`; its commit-addressed ledger archive is `.didrun-history/u7a-final-e178f1e79720/.didrun`.

Those identifiers admit U7A only as the exact predecessor. They do not transfer a U7B grade, prove this source candidate, or turn a local HTML or secret-bearing ledger snapshot into portable evidence. Every U7B claim below begins `UNRECEIPTED` and must be earned over the exact U7B tree in one fresh final ledger.

## Product boundary

U7B adds one narrow Darwin reference-machine route:

```text
countershape study http --json
```

The route is harness-only. Its successful process protocol is exactly one canonical JSON line with four fields and empty stderr:

```json
{"schema_version":"countershape/u7-study-domain-result/v1","domain":"http","ordinal":1,"status":"GREEN"}
```

The ordinal is supplied by the admitted harness environment and may be only canonical `1`, `2`, or `3`. The application refuses execution unless the requested domain, ordinal, exact working directory, distinct private scratch and empty evidence roots, and absolute Node/Git executables all pass admission. A human `study http` route does not invoke the handler; it returns a typed harness-only refusal and a safe next action.

The warning governing any execution-capable operator flow remains verbatim:

> Countershape will run trusted repository code with your user permissions and host network access. The temporary directory is for repeatability, not security isolation.

The four-field automation result intentionally does not repeat that prose. This status and the human preflight/operator flow carry the warning as an operator precondition; the frozen harness does not independently observe that a person read or understood it. The machine line is not a security, isolation, or safety verdict.

## Direct outer-fixture complete workflow

The outer U7P-frozen harness invokes `tools/run-u7-http-study.mjs` with exactly:

```text
--prepare --domain http --ordinal N --fixture-root ABSOLUTE_PATH
```

On success the driver emits no stdout or stderr and does not receive or access the evidence root. It creates one new private worktree with a clean, deterministic, parentless `HEAD`. That `HEAD` tracks every behavior-bearing portable HTTP candidate byte. Four deterministic parentless display refs name candidate-root trees for the `forbidden`, `conceal-not-found`, `metadata-disclosure`, and `alternating` roles. The ref trees are derived from those tracked bytes; they are not an untracked prepared cache. Fixed Git identities, dates, disabled signing/hooks/network protocol, and a closed environment make the fixture commit and trees repeatable across the three outer runs. Moving a display ref after a pin does not rewrite the already-pinned candidate.

The product opens that exact harness-prepared repository directly. It does not create a nested candidate repository. `testkit/reference` validates the private worktree roster, exact tracked portable bytes, domain marker, candidate refs, distinct commits/trees, and hardened Git inspection before returning the repository capability. It also refuses malformed, cross-domain, or process-local shared-root fixtures.

The product then executes one complete typed workflow before publication:

1. pin all four display refs once, inspect their exact Git trees under one materialization policy, compile strict source into a typed `WorldPlan`, and retain the exact comparison envelope;
2. run 12 decisive discovery trials: three fresh repetitions for each candidate, requiring A (`403`), B (`404`), and C (metadata-bearing `200`) to be `OBSERVED_STABLE(3/3)` while D produces both projected outcomes and is excluded as `UNSTABLE`;
3. run two additional 12-trial discovery batches whose complete labeled maps equal the decisive map while their world, attempt, observation, and map authorities are physically new;
4. run four reference plus four tenantless-seed trials and require the same request shape to produce a different complete fingerprint map, refusing shape-only preservation;
5. run two separately fresh 12-trial noisy-baseline batches, require exact-map equivalence without canonical-byte reuse, then run one 12-trial reducer evaluation that removes only the irrelevant seed and preserves the whole labeled map;
6. record the honest weak `BEST_KNOWN` result, publish and reopen a complete durable sweep, and require `ONE_MINIMAL_UNDER` only from that sweep authority;
7. run eight physical confirmation trials under the core-owned rotated schedule, require exact-map reproduction, and retain eight valid fresh confirmation facts;
8. persist the confirmed study, promote a blind Choicepoint, prove a content-type-only proposal returns `AMBIGUOUS_SCOPE` without emitting an object, then finalize a separate status-only `404` ruling after reveal and provenance review;
9. construct and strictly reopen the portable HTTP source, compile and publish the selected-field six-file Node contract bundle, and retain its residue;
10. materialize and terminally reopen the exact six-file bundle, allocate ten separate fresh official targets for the pinned `404` candidate, immediately reopen each pre-spawn target, run its `contract.test.mjs` through the app-owned bounded Node test boundary, require the exact TAP diagnostic `CONFORMS|NONE`, and terminally reopen the same target and bundle; and
11. derive exactly 80 search, eight confirmation, and ten contract GREEN protocol rows only from the retained physical attempt, confirmation, and direct-process facts. No caller supplies phase status or phase hashes. The ten direct process facts do not create a sealed HTTP-runner permit, finalized-run, resume, or classification authority.

The completed value retains the admitted ordinal and a recomputed physical-run binding. Its 80 search worlds must carry the exact `u7-http-N-` ordinal prefix, and that product-computed digest binds those world identities plus 80 attempt digests, eight confirmation-fact digests, and ten direct-process-fact digests. Publication derives the ordinal from that sealed value—there is no second caller-supplied publication ordinal—and every fresh protocol slot carries the same binding, including the two explicit absence projections.

Successful publication creates exactly 111 private mode-`0600` JSON files beneath the already-admitted evidence root: five deterministic wrappers, eight ordinal-bearing fresh wrappers, 80 search trial records, eight confirmation trial records, and ten contract trial records. Event 5's current-tree compiled-product E2E independently parses the payloads, requires 88 world/attempt/measurement/capture authorities, verifies all 80/8/10 phase-receipt joins, and checks each fresh target -> direct process-fact association plus the explicit absence of sealed finalized-run and classification authority. The frozen filenames `fresh/finalized-contract-run.json` and `fresh/contract-execution.json` are protocol slots: their payloads say `availability: ABSENT`; they do not impersonate the missing core objects. All evidence directories are mode `0700`; publication is no-overwrite and terminally reopened under the declared sole-writer model. The application emits the four-field green line only after the handler returns, the evidence workspace closes, and working/scratch/evidence/tool identities are rechecked.

Use `OBSERVED_STABLE(3/3)` only for the three discovery candidates after final event 5 records its complete conjunctive product/race command. That observation classification is established by the full compiled-product child and its exact typed assertions, not by treating that child as race-instrumented or by relabeling the direct bundle diagnostic as a sealed contract classifier result. Event 7 observes process and artifact bytes but does not independently prove either semantic linkage. Do not abbreviate the result to deterministic or generally stable. The alternating candidate remains excluded rather than decided by majority or last observation.

## Direct product evidence and event-6 substrate regressions

The direct product route above makes the complete happy-path discovery, reduction, confirmation, ruling, emission, fresh-target allocation, and direct standalone bundle-test chain load-bearing. A discovery-only `Result` cannot authorize evidence: only the package-sealed `CompletedHTTPStudy` issued after every terminal validator passes can reach production publication. This is not the P08 target-bound finalized-run/classification chain.

Event 5 is one exact command with two separately bounded surfaces. The race-instrumented parent synchronously builds `cmd/countershape` from the current tree with exact `-trimpath -mod=readonly -buildvcs=false -p=1`, clears ambient `GOFLAGS`, disables the writable Go environment and toolchain switching, reopens the child build information, and refuses a child carrying `-race=true`. That verified non-race binary then runs the complete 80/8/10 product route from the driver fixture under a six-minute test-only deadline; the parent retains the exact four-field stdout, empty-stderr, 111-file semantic/phase-link, and unchanged-fixture assertions. Separately, the race-instrumented process runs a real two-repeat/eight-trial HTTP world-observe-compare slice, concurrent application/workspace isolation, concurrent fixture-copy close/reopen authority, concurrent three-root publication, the inherited application concurrency cases, fixture admission tests, and the remaining bounded package tests. This topology does not establish that the full reducer, rotated confirmation, blind ruling, bundle, or ten direct bundle tests ran under the race detector. `FULL_80_8_10_WORKFLOW_RACE_INSTRUMENTATION_UNVALIDATED` remains an exact nonclaim, and silence from the detector on the bounded cases is not race freedom. The nested build and its test-only deadline are event-5 assertions, not event-7 process/resource authority; the frozen product-harness budget is unchanged.

The deterministic `ruling`, `decision-record`, and `contract-bundle` files are explicitly labeled `SEMANTIC_REGRESSION_PROJECTION`. They contain invariant selected-field/predicate/grade facts derived only after strict typed records and the six-file bundle validate; they are not alternate constructors and do not claim to be the underlying canonical DecisionRecord or ContractBundle bytes. The fresh confirmation payload carries its typed digest and eight physical facts. The target payload carries ten real pre-spawn target/attempt projections. The finalized-run-named payload carries only direct process facts and declares finalized-run authority absent; the execution-named payload carries only `STANDALONE_BUNDLE_TEST_PASSED`, process links, and classification authority absent. The frozen harness checks paths, schemas, exact wrapper bytes, deterministic-wrapper repetition, and ordinal-bearing fresh-wrapper digest separation. It does not independently infer semantic linkage, compare fresh payloads with their ordinal removed, or prove literal cross-run equality of the underlying canonical DecisionRecord/ContractBundle objects.

Event 6 is a separate claim over the pre-existing decisive HTTP substrate suite:

```text
/opt/homebrew/bin/go test -mod=readonly -buildvcs=false -p=1 -count=1 -timeout=15m ./testkit/studies/http_invoices
```

That separate regression surface exercises the named deliberately invalid shared-root contamination negative, fresh-root recovery, visible projection sensitivity, exact tenantless `200 | 500 | 500` changed-map trap, same-partition/different-fingerprint refusal, unresolved reduction and both grade boundaries, copied-capture refusal, B conformance, A/C contradiction, current-evidence handling for D, custom `401`, and noncompilable `REJECT_ALL` behavior. Those broader hostiles remain separately load-bearing even where the direct route exercises the matching positive or a narrower in-route hostile.

An event-6 green grade would establish only that exact regression command on its recorded tree and runtime. It cannot substitute for the U7B application route's own completed workflow, and it cannot be merged into event 5 or event 7. Conversely, event 5 or the harness event cannot substitute for event 6. The final U7B ledger requires all three distinct surfaces.

## Frozen harness ceiling

U7B inherits the U7P-frozen protocol `countershape/u7-study-harness/v1` and observation authority:

```text
U7P_FROZEN_HARNESS_DIRECT_PROCESS_GIT_AND_ARTIFACT_OBSERVATION
```

The driver protocol is `FIXTURE_ONLY_NO_EVIDENCE_ROOT`. For U7B, the harness prepares three fresh repositories, compiles the reference binary, invokes the exact machine route once per repository, and directly observes fixture Git/worktree authority, exact driver/binary/tool identities, prepare/compile/execute process topology, timing transcripts, the exact closed artifact roster, within-domain deterministic-wrapper byte repetition, and ordinal-bearing fresh-wrapper digest nonaliasing. Product code places the same product-computed physical-run binding in every fresh protocol slot; that digest is derived from ordinal-bound world nonces, 80 search-attempt facts, eight confirmation facts, and ten direct standalone process facts. The frozen harness does not parse or independently compare those underlying facts.

Its semantic ceiling is exact and must remain visible:

```text
ARTIFACT_BYTES_AND_SUBJECT_PROCESS_TOPOLOGY_OBSERVED_PRODUCT_SEMANTICS_AND_FULL_HARNESS_RESOURCES_NOT_INDEPENDENTLY_ESTABLISHED
```

The three-run HTTP budget is 300 logical protocol trials: fixture `3`, compile `3`, search `240`, confirm `24`, and contract `30`. In the frozen protocol, the last 30 GREEN rows mean ten direct bundle-test process facts per run; they are not sealed HTTP-runner classification grades. The subject-process ceiling is the sum of the three directly observed prepare/compile/execute spans at 900,000 ms and the maximum observed subject-process RSS at 4,294,967,296 bytes. Those values exclude parent-harness orchestration, Git inspection, artifact walking/hashing, and parent Node memory. Therefore `FULL_STUDY_RESOURCE_BOUND_UNVALIDATED` remains an explicit nonclaim even if event 7 passes.

The phase verdict token `LOCAL_HTTP_REFERENCE_FUNCTIONAL_GREEN_SUBJECT_RESOURCE_OBSERVED` may be recorded only by a successful final event 7. It means exactly the frozen process/artifact observation above. It does not mean independently recomputed semantic reproduction, a total full-study resource bound, production readiness, or a U7D/U7R receipt.

## Exact owned roster

All 16 paths are required mode-`100644` tracked blobs. No other staged path is authorized:

1. `cmd/countershape/main.go`
2. `internal/reference/app/command.go`
3. `internal/reference/app/envelope.go`
4. `internal/reference/app/render.go`
5. `internal/reference/app/workspace.go`
6. `internal/reference/app/run_darwin.go`
7. `internal/reference/httpstudy/study.go`
8. `internal/reference/httpstudy/reduction.go`
9. `internal/reference/httpstudy/result.go`
10. `internal/reference/httpstudy/study_test.go`
11. `internal/reference/httpstudy/e2e_darwin_test.go`
12. `testkit/reference/http.go`
13. `testkit/reference/http_test.go`
14. `tools/run-u7-http-study.mjs`
15. `docs/HANDOFF_MODE_C.md`
16. `docs/status/U7B-HTTP-FALSIFICATION.md`

The HANDOFF edit must remain outside every frozen P07 marked block. U7B may not add a U7D receipt projection, pending U7R receipt block, or extra product path.

## Exact claim manifest

Every grade remains `UNRECEIPTED` in this source candidate:

| Event | Type | Exact claim label | Current grade |
|---:|---|---|---|
| 0 | tests-pass | U7B candidate plan and sealed-U7A parent authority | `UNRECEIPTED` |
| 1 | tests-pass | U7B independent candidate transition and exact staged authority | `UNRECEIPTED` |
| 2 | tests-pass | U7B plan contract defensive self-test | `UNRECEIPTED` |
| 3 | tests-pass | U7B independent staged-scope defensive self-test | `UNRECEIPTED` |
| 4 | command-succeeded | U7B exact-owned Go formatting | `UNRECEIPTED` |
| 5 | tests-pass | U7B Darwin Node 25 HTTP product orchestration and race tests | `UNRECEIPTED` |
| 6 | tests-pass | U7B Darwin Node 25 HTTP decisive substrate regressions | `UNRECEIPTED` |
| 7 | tests-pass | U7B Darwin Node 25 harness-observed three-run HTTP process/artifact protocol and byte-repetition evidence | `UNRECEIPTED` |
| 8 | tests-pass | U7B cumulative verification | `UNRECEIPTED` |
| 9 | command-succeeded | U7B exact staged scope and diff integrity | `UNRECEIPTED` |
| 10 | command-succeeded | U7B scoped staged credential-pattern scan | `UNRECEIPTED` |
| 11 | command-succeeded | U7B sealed-U7A predecessor and preceding didrun chain integrity | `UNRECEIPTED` |

The final attempt must use only the generated U7B runbook, whose zero-based event index, command, immediate claim, and exact 16-path pathspec binding are authoritative. A successful command without its immediate exact claim remains unclaimed. Any nonzero command permanently ends that ledger; preserve it through the generated matching recovery fence, repair from exact U7A, and restart at event zero. No development result may be copied, replayed, or relabeled into the final ledger.

## Architecture and nonclaims

The U7 architecture boundary requires a thin composition root importing only `app` and `httpstudy`; domain-neutral app code cannot import the HTTP study, adapters, fixture package, or contract machinery. The HTTP study owns the app/HTTP-adapter/reference-fixture edges and cannot import the later CLI or reproduction layers. The fixture package cannot import `internal/reference`. The driver is mode `100644`, uses only the admitted static Node built-ins, has no relative/dynamic/CommonJS loader, and has no HTTP/network built-in or fetch/WebSocket path. No governed Go file has a build tag, and the U7A-only strict parser/validation/preflight/test files remain frozen outside U7B ownership.

The architecture checker is bounded source/import/name evidence. It is not a compiler proof, dataflow proof, sandbox, transitive dependency audit, or same-user isolation boundary. Candidate code runs with the user's permissions and host network access. Private temporary directories support freshness and repeatability; they do not contain malicious code, deny the network, protect secrets from the same user, or prove external repository safety. Evidence publication performs exact no-follow creation, roster walking, and terminal reopen under the required sole-writer execution model; it does not resist a malicious same-UID actor that renames and restores paths between observations.

U7B does not establish Linux or Windows behavior, unrun Node majors, broad imported-repository behavior, hostile containment, network denial, total full-study resource use, full-workflow race instrumentation or race freedom, independent product-semantic linkage, harness-independent raw cross-run core-authority nonaliasing, literal underlying DecisionRecord/ContractBundle byte repetition, comprehension, review compression, adoption, maintainability, production readiness, an independent security review, external-platform behavior, imported-repository timing, the later CLI study, cross-domain closure, or a U7D/U7R receipt. It also creates no sealed HTTP-runner interlock, start claim, run permit, target-bound finalized-run, classification-only resume, or classifier authority. The generated harness starts its subject detached: natural successful return includes the harness's own teardown, but an outer timeout/cancel/overflow cannot prove that detached subject absent; such a failure is nonpublishable and requires external process-tree audit before any normal retry. `P08_TARGET_RUN_CLASSIFICATION_REPRODUCTION_UNMET`, `FULL_80_8_10_WORKFLOW_RACE_INSTRUMENTATION_UNVALIDATED`, and `U7R_SELF_RECEIPT_ABSENT` remain true.

## Development-only record

Development checks are not final evidence. Earlier superseded-candidate ordinary in-process E2E runs exercised the real driver-prepared ordinal-1 fixture through `app.Run(["study", "http", "--json"], Handler())`; the later run completed in 242.988 seconds but still used the now-retired sealed-runner target/run/classification route. After the conjunctive event-5 repair, a compiled-product E2E on that superseded candidate passed in 230.933 seconds, and the race-instrumented eight-trial physical slice passed in 7.742 seconds. After the authority-absence narrowing and exact attempt-partition repair, the current direct-process compiled-product E2E passed in 169.14 seconds and a terminal process audit found no matching Go test, product, Node contract, fixture-server, didrun, or verifier process. That run was not race-instrumented, was not the final event-5 command, and carries no final claim. Separate ordinary focused tests exercised the human route, invalid environment, special-mode, terminal-mode-drift, fixture-ref/commit, concurrent application/workspace, concurrent publication, publication refusals, escaped-request revocation, and canceled-runner poisoning. The reference-fixture package tests and U7B architecture checker also passed during development.

Three development-only attempts to run the complete workflow inside the race-instrumented test process ended with opaque exit `70` at 327.55, 720.79, and 861.63 seconds under test-only five-, twelve-, and fourteen-minute contexts. They emitted no race detector report, are consistent with context exhaustion but were not more narrowly classified, support no race or event-5 finding, and motivated the explicit conjunctive boundary above. The frozen product-harness five-minute subject-process budget was never changed. A first bounded-slice diagnostic also refused an unresolved Node symlink before executing a trial; passing the same canonical tool identity admitted by the application repaired only that test wiring.

None of these commands ran as a U7B final didrun event or received an exact claim. The diagnostics were neither the final event-5 conjunction nor the frozen outer harness's three-run prepare/compile/execute sequence. They supply no final timing, RSS, deterministic three-run equality, cross-run freshness, event-6 substrate, race-freedom, cumulative-verifier, staged-scope, credential-scan, ancestry, commit, seal, note, strict, HTML, archive, or receipt grade. No final three-run metric is reported here because none has yet been recorded by the required event 7.
