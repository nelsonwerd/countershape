# U7N CLI contract and milestone authority alignment

## Boundary identity

- **State:** active replacement source candidate; not committed, sealed, or receipted. One earlier unsealed failed-attempt commit is retained separately below.
- **Namespace:** `countershape/u7-unit-paths/v3`.
- **Boundary:** `U7N`.
- **Declared parent:** exact sealed `U7B`.
- **Subject:** `fix: reconcile U7 CLI and milestone authority`.
- **Profile:** `SOURCE_FULL`.
- **Product authority:** `NONE`.
- **Product behavior:** `INHERITED_UNREPROVEN`.
- **Grade transfer:** `NONE`.
- **U7D source receipt:** `ABSENT`.
- **Final private root:** `.countershape/u7n-final`.
- **Next boundary:** `U7C` only after exact U7N commits, seals, exposes its note, passes explicit-commit strict verification, emits its commit-addressed HTML, and closes its ledger archive.

Exact parent U7B is commit `7ae0d4847b4479e3d85b979ab26825f49638db54`, tree `805e38b30c91315e962abb69c1e015d6278f6882`, direct parent U7A `e178f1e7972017fb0cf8aab590d7f528a45f4bd9`, and subject `feat: add U7 HTTP falsification study`. Its didrun note is blob `987dd6f56507ea22a398010b79118780acda99b3`, 28,030 bytes, with note-body SHA-256 `484fd34d321a80400c6cbe22c3c802e1deb24767ce5c413cd99f67128cdc23c7`. U7B closed `12/12` claims at `TREE-EXACT`, strict exit `0`, with the high-entropy override recorded. Its local 8,570-byte HTML witness is `.countershape/evidence/u7b-final-7ae0d4847b44.html`, SHA-256 `1bdfa09d1ef80c3eda2da1a27951c3e638a90a4794fb83f1738119b9ca897f85`; its commit-addressed ledger archive is `.didrun-history/u7b-final-7ae0d4847b44/.didrun`.

Those identifiers admit U7B only as the exact predecessor. They do not transfer a U7N grade, prove this control candidate, or turn the local HTML or secret-bearing ledger snapshot into portable evidence. Every U7N claim below begins `UNRECEIPTED` and must be earned over one exact U7N tree in one fresh final ledger.

## Owner-selected authority amendment

U7N is the smallest zero-product repair for two contradictions found only after U7B sealed. It is a second explicit topology amendment with these exact machine facts:

- source `OWNER_OUT_OF_BAND`;
- classification `OWNER_AUTHORIZED_CLI_AND_MILESTONE_AUTHORITY_RECONCILIATION`;
- provenance kind `UNEVIDENCED` with disclosure `OWNER_ATTRIBUTED_SESSION_INSTRUCTION_ONLY_NO_QUALIFYING_PREEXISTING_ARTIFACT`;
- authentication `NOT_ESTABLISHED`;
- signed authorization `NOT_IMPLEMENTED`;
- predecessor declaration kind `NONE`;
- product authority `NONE`;
- product behavior `INHERITED_UNREPROVEN`;
- grade transfer `NONE`;
- defect `U7C_EXIT2_AND_EDGE_CLASSIFICATION_CONFLICTS_WITH_EXACT_ENROLLED_CLI_STIMULUS_AND_U7D_OVERSTATES_ABSENT_HTTP_RUN_CLASSIFICATION_AUTHORITY`.

This authority may amend future topology, protocol identity, and claim wording. It cannot authenticate the owner, transfer U7B grades, prove inherited behavior, change product/runtime facts, or create a U7C/U7D result. U7N changes no `cmd/**`, `internal/**`, `testkit/**`, P07 checker/spec, runner, fixture, adapter, or evidence-engine byte.

The exact source chain is now:

```text
C6B -> U7P -> U7M -> U7A -> U7B -> U7N -> U7C -> U7D -> U7R
```

Only U7C's immediate parent changes, from U7B to U7N. U7D remains a direct child of U7C and U7R remains a direct child of U7D. U7B and every earlier commit, note, archive, grade, and historical statement remain immutable.

## Exact CLI semantic correction

The sealed runner enrolls one exact CLI candidate inventory together with one exact reference stimulus. That reference stimulus supplies no behavior override, so the fixture selects its default normal exit `0`. The fixture's only separate explicit nonzero behavior exits `7`. Therefore a new exit-`2` fixture/stimulus cannot become the enrolled official reference execution without changing protected authority outside U7C.

The future U7C contract is consequently bounded as follows:

1. the one official target-to-run-to-classification proof uses the exact enrolled reference stimulus and normal exit `0`;
2. exit `7` remains fixture/substrate coverage, not an enrolled official-reference proof;
3. a clean signal is an eligible completion whose selected exit-code field is `MISSING`, so it contradicts the normal-exit tuple rather than becoming ineligible;
4. an invalid or traversal overlay is rejected before spawn and creates no execution;
5. timeout, malformed selected JSON, and output-limit cases may prove adapter/world behavior, but changed stimuli are scope-unenrolled and cannot be relabeled as official reference classifications;
6. runtime escaped-write behavior is unproven in this unit; and
7. `CLI_OFFICIAL_EXECUTION_GENERALIZATION_UNVALIDATED` remains explicit.

The generic post-spawn application failure prose may still say that no candidate ran even after an attempted run. U7N owns no application byte, so this operator-UX defect is excluded from the U7C/U7D milestone and requires a separately authorized product repair if release-quality interactive failure reporting becomes mandatory.

## Harness v2 and bounded U7D authority

Changing the future U7D verdict changes harness bytes, so the old v1/U7P-frozen label cannot remain. U7N establishes an explicit U7P-origin/U7N-amended epoch:

- protocol `countershape/u7-study-harness/v2`;
- execution authority `U7P_ORIGIN_U7N_AMENDED_HARNESS_V2_DIRECT_PROCESS_STUDY_EXECUTION`;
- observation authority `U7P_ORIGIN_U7N_AMENDED_HARNESS_V2_DIRECT_PROCESS_GIT_AND_ARTIFACT_OBSERVATION`;
- protocol-authority SHA-256 `f9e5c6580c85d6603b3064ecd03de2328a8a8bced5107bf11456a07b2c9c40f1`;
- harness-file SHA-256 `ba8ed823b8c3740a6af0baf124a32c8663cef68483c1e940b5e1d8ecdb1416fa`;
- defensive self-test: 83 cases at digest `088df37bd893da368108693845852a608fee2e5f75da814d0dc47535621fc81b`.

The v2 protocol binds the exact authority strings, domain/driver/budget/artifact topology, bounded U7D verdict, honest fallback, and required limitation suffix. U7N itself runs only `--protocol-check` and the synthetic defensive self-test; it is not a product-study phase.

The exact U7D milestone is:

```text
LOCAL_TWO_DOMAIN_PROCESS_ARTIFACT_GREEN_CLI_OFFICIAL_EXECUTION_HTTP_DIRECT_PROCESS_ONLY_FULL_STUDY_RESOURCE_UNRECEIPTED
```

The exact fallback is:

```text
ONE_DOMAIN_OFFICIAL_CONTRACT_EXECUTION_PLUS_HTTP_DIRECT_PROCESS_OBSERVATION_ONLY
```

Sealed U7B created ten fresh official HTTP targets and ten app-observed direct bundle-process facts. It explicitly did not create target-bound HTTP `FinalizedContractRun` or classification-only `ContractExecution` authority. Therefore the receipt contract must retain these exact limitations:

- `HTTP_FINALIZED_CONTRACT_RUN_AUTHORITY_ABSENT`;
- `HTTP_CONTRACT_EXECUTION_CLASSIFICATION_AUTHORITY_ABSENT`;
- `TWO_DOMAIN_TARGET_RUN_CLASSIFICATION_REPRODUCTION_UNMET`;
- `CLI_OFFICIAL_EXECUTION_GENERALIZATION_UNVALIDATED`;
- `FULL_STUDY_RESOURCE_BOUND_UNVALIDATED`;
- `U7R_SELF_RECEIPT_ABSENT`.

Harness v2 still observes only fixture Git/worktree authority, direct prepare/compile/execute topology, executable identities, artifact paths/modes/bytes, deterministic-wrapper repetition, ordinal-bearing fresh-wrapper nonaliasing, and direct subject-process wall/RSS. It does not independently recompute product semantics, raw product-computed authority freshness, HTTP target/run/classification, per-logical-phase resources, or total parent-harness resources.

## Exact owned roster

All 16 paths are required mode-`100644` tracked blobs. No other staged path is authorized:

1. `docs/ARCHITECTURE.md`
2. `docs/HANDOFF_MODE_C.md`
3. `docs/PROMPT_PACK.md`
4. `docs/STATE_MACHINES.md`
5. `docs/VERIFICATION.md`
6. `docs/prompts/P08-U7-CLI-REFERENCE-STUDIES.md`
7. `docs/status/U7N-CLI-CONTRACT-ALIGNMENT.md`
8. `spec/verification/u7-unit-paths.json`
9. `tools/check-u7-plan.mjs`
10. `tools/check-u7-scope.mjs`
11. `tools/check-u7-architecture.mjs`
12. `tools/check-u7-architecture-selftest.mjs`
13. `tools/check-u7-study-harness.mjs`
14. `tools/check-u7-study-harness-selftest.mjs`
15. `tools/verify-current.mjs`
16. `tools/verify-current-selftest.mjs`

The HANDOFF edit must remain outside every frozen P07 marked block. `tools/print-u7-final-runbook.mjs` remains unchanged and derives U7N dynamically from the specification. U7N may not add a U7D pending/source receipt projection, product path, new study driver, or product output.

## Exact claim manifest

Every grade remains `UNRECEIPTED` in this source candidate:

| Event | Type | Exact claim label | Current grade |
|---:|---|---|---|
| 0 | tests-pass | U7N candidate plan and sealed-U7B parent authority | `UNRECEIPTED` |
| 1 | tests-pass | U7N independent candidate transition and exact staged authority | `UNRECEIPTED` |
| 2 | tests-pass | U7N inherited P07 receipt successor compatibility | `UNRECEIPTED` |
| 3 | tests-pass | U7N plan contract defensive self-test | `UNRECEIPTED` |
| 4 | tests-pass | U7N independent staged-scope defensive self-test | `UNRECEIPTED` |
| 5 | tests-pass | U7N final runbook renderer defensive self-test | `UNRECEIPTED` |
| 6 | tests-pass | U7N zero-product-surface architecture conformance | `UNRECEIPTED` |
| 7 | tests-pass | U7N architecture authority defensive self-test | `UNRECEIPTED` |
| 8 | tests-pass | U7N amended study-harness v2 protocol conformance | `UNRECEIPTED` |
| 9 | tests-pass | U7N amended study-harness v2 protocol defensive self-test | `UNRECEIPTED` |
| 10 | tests-pass | U7N cumulative verifier defensive self-test | `UNRECEIPTED` |
| 11 | tests-pass | U7N cumulative verification on the exact staged maintenance candidate | `UNRECEIPTED` |
| 12 | command-succeeded | U7N exact sixteen-path staged scope and diff integrity | `UNRECEIPTED` |
| 13 | command-succeeded | U7N scoped staged credential-pattern scan | `UNRECEIPTED` |
| 14 | command-succeeded | U7N sealed-U7B predecessor and preceding didrun chain integrity | `UNRECEIPTED` |

## Permanent failed postcommit attempt

The first U7N final attempt used exact tree `6b892e9479aec0a927214cdb0acd7578ca8fc44e` and reached unsealed commit `0e90217a22bdd3a05fbe3211655241c5636395c0`, directly after exact U7B `7ae0d4847b4479e3d85b979ab26825f49638db54`. Its one-writer ledger is complete at 15 events/15 claims, but the mandatory plain seal failed closed on 190 high-entropy findings. It created no note, strict result, HTML witness, normal commit-addressed archive, or U7N grade.

Exact generated postcommit recovery retained that commit at `refs/countershape/failed-attempts/u7n/0e90217a22bdd3a05fbe3211655241c5636395c0`, moved the complete failed ledger to `.didrun-history/u7n-final-failed-0e90217a22bdd3a05fbe3211655241c5636395c0/.didrun`, preserved the private root at `.countershape/u7n-final-failed-0e90217a22bdd3a05fbe3211655241c5636395c0`, and restored `HEAD` to sealed U7B. Every event and claim is permanent nonreusable failed-attempt history. The active replacement candidate retains no event, claim, seal, or grade from that failed tree and must start again at event zero on its new exact tree.

Read-only seal review used the installed didrun detector and the exact 16-path inventory without opening the archived ledger payload. The detector skips hex-only Git and SHA values. Reconstructing the claimed command argv identifies 180 of the 190 aggregate findings as the twelve sterile private path/tool environment assignments repeated across 15 events. The exact nine-pattern staged credential scanner reports zero findings, and review of the fixed command-output emitters found only protocol/authority markers, counts, paths, and OIDs rather than credential-shaped values. The remaining ten findings were not individually mapped from the forbidden ledger payload, so this is bounded manual review, not proof of secret absence.

The replacement final attempt must use only the generated U7N runbook, whose zero-based event index, command, immediate claim, and exact 16-path pathspec binding are authoritative. A successful command without its immediate exact claim remains unclaimed. A nonzero evidentiary command, claim, commit/preflight step, non-entropy seal failure, or later closure permanently ends that ledger; preserve it through the exact generated recovery fence, repair from sealed U7B, and restart at event zero. The sole exception is a plain seal refusal solely on didrun's high-entropy heuristic. Only after the replacement's freshly claimed exact-inventory credential scan again reports zero and deliberate review confirms the same bytes may the operator run the generated commented `--allow-secrets` checkpoint. That redacted seal must disclose `secrets_override: true` and does not establish secret absence. No development or failed-attempt result may be copied, replayed, or relabeled into the replacement ledger.

## Verification and nonclaims

The two transition validators independently pin the v3 eight-unit topology, both ordered amendment records, exact U7N roster and claim manifest, only-U7C reparenting, v2 receipt contract, sealed U7B-to-C6B ancestry, exact archive/note authority, and one-writer interval discipline. The architecture oracle admits U7N as a zero-new-product phase over the 19-file cumulative U7B surface and permits exactly U7P and U7M/U7N control ownership according to the pinned owner table. The cumulative verifier retains 70 current rows and 15 historical-only rows; U7N changes current phase selection and harness epoch without deleting, relabeling, or executing the three successor-incompatible historical P07 entrypoints.

U7N establishes no product behavior, CLI result, HTTP result, cross-domain result, new runtime, new fixture, security boundary, containment, network denial, cross-platform behavior, imported-repository behavior, adoption, maintainability, production readiness, independent security review, resource result, or receipt reconciliation. Its architecture, harness, verifier, and self-test greens, including those recorded in the failed ledger, prove only their exact commands on that failed tree. They do not transfer to the replacement tree, receipt U7N, or prove future U7C/U7D behavior.

## Development-only record

The U7C deep-dive and U7N development checks are diagnostics only. Static review identified the enrolled-stimulus, signal, overlay, escaped-write, HTTP-authority, and harness-epoch mismatches described above. Development syntax checks, the 100-case architecture self-test, 70/15 verifier self-test, plan self-test, independent scope self-test, and harness defensive self-test may guide repair, but no development result supports a U7N claim.

Proceed to U7C only after a fresh exact U7N ledger records all 15 commands and claims, the declared commit is sealed, the note is inspected, explicit-commit strict verification exits `0`, the commit-addressed HTML exists, normal ledger rotation succeeds, and a terminal clean-state audit reopens the same authority.
