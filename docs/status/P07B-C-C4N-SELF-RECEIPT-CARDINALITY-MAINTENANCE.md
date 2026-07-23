# P07B-C C4N — self-receipt cardinality maintenance

Classification: `DEFECT_REPAIR`

C4N owns exactly these 7 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C4N-SELF-RECEIPT-CARDINALITY-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Their sorted-newline roster digest is `sha256:95de7e71ca2df4a57d0544c3197bb7d917359bb3d9ecfc51a2130973faf36e6e`.

- **Boundary:** active pre-seal `SOURCE_FULL` receipt-phase machinery maintenance after sealed C4M and before C4. Classification: `DEFECT_REPAIR`.
- **Parent:** sealed C4M commit `d87394799d2229bd6e6d342318f446272685a86f`, tree `afd0157303e9025a490b2b6ac991f5f15734137f`, is note-present with note blob `af7eec1ff78c6ceec40e39dffebdfe97ed4da62c`, note-body SHA-256 `68fa26fcb59c9cb3c6066d28459b3f4ec4df596812f1d55ae7b23f3f5ea7be48`, `10/10 claims recorded-exact`, strict exit `0`, and recorded `secrets_override: true`.
- **Commit subject:** `fix: derive live self-receipt cardinality from phase authority`
- **Profile:** `SOURCE_FULL`
- **Receipts:** C3P `PRESENT`; C3 `PRESENT`; C6A `ABSENT`.
- **Product authority:** none.

Every intended C4N grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C4N itself.

## Defect and routing

C4M correctly sealed a 47-path live refusal corpus. Under sealed v18, the next declared C4 phase added one status path, so a frozen active-only cardinality literal rejected the otherwise-valid 48-path C4 successor surface. Current v19 inserts C4N as the 48th path and derives later C4 reentry as 49 paths. That is a checker-extension defect, not a product failure and not a reason to weaken any refusal, claim, or sealed receipt.

The external consolidation proposal is accepted as a `SOURCE_FULL` simplification boundary. Structural phase behavior now comes from one validated phase table, one forward-candidate allowlist derived from unit order, and generic fixtures. The fixed C3R, C3Q, C3T, C3U, C4V, and C4M regressions remain separate semantic oracles because row topology cannot encode their note, lifecycle, corpus-transport, and sealed-history meanings without hiding bespoke behavior in the table.

## v19 phase authority

The machine scope is `countershape/p07b-c-unit-paths/v19`: 21 ordered phase rows at authority digest `sha256:8f237c7d883c212167e5a04e5dd24d6efdacf4a9ce5492215fd06892a8237c4a`. It inserts only `C4N` between C4M and C4 and changes no product roster.

- The plan transition matrix has 9 accepted and 232 rejected structural, authority, and exhaustive predecessor cases.
- The independently implemented scope transition matrix has 9 accepted and 219 rejected cases.
- Every candidate is exercised against explicit absence plus all 21 declared receipt-phase predecessors; only its table-declared edge is accepted.
- C4V, C4M, and C4N are the only post-C3D specification owners. Every non-owner candidate must preserve its parent specification bytes exactly.

## Generated self-receipt catalog and frozen history

The generic catalog contains 840 cases—441 accepted controls and 399 hostile rejections—at digest `a414d55463202a2edea6b132eb1259e83430ad10c0695ecbb729ac3441c3c19c`. Six fixed regressions extend it to 846 executable cases—447 accepted controls and 399 hostile rejections—at digest `0aa97e46bd3a24c9238b6bd46a574234ea9b798d72175a46d4190049a5910595`. Their manifest digest is `145421ce5d9b7eedbd852142febd89ecde5786e4224243bdbd7ea2e0e1e55ef5`.

The sixth oracle, `C4M_FROZEN_SELF_RECEIPT_CARDINALITY`, pins sealed C4M commit `d87394799d2229bd6e6d342318f446272685a86f`, checker blob `ea2f8a8e74c886e931af19c0dac0190c9e4d8da3`, checker SHA-256 `0d5943cfbd48e1d0cf7da00389eab569c08b5bcfa26dad8b90528e0e9a5e822d`, and the distinction between a `47-path frozen C4M` corpus and the `48-path active C4N` corpus. It preserves C4M's 735 positive-form rejections and 245 controls while C4N derives 750 rejections and 250 controls. C4 reentry derives 765 rejections and 255 controls. The eight-phase forward aggregate is therefore 6,180 rejections and 2,060 controls plus eight full-plan controls; the eight-row extension catalog is pinned separately at `c5e5d421e0dab5709ef8356dc8c25421a0effff8f2698efea0da9e9d74d23761`. The count is derived from per-phase rosters and does not leak C4N backward into C4M history.

The earlier 31-subcase complete-descendant corpus oracle remains pinned at `d7db6136c8c4bc4672bc6be374c5985ab4c5ad9990f48f4acde69f76d6bacc40`. The frozen C3U/C3B 44-path form/control authority remains unchanged. C4N adds no reinterpretation of any sealed grade.

## Predecessor and future ancestry

C4N's predecessor gate reopens exact sealed C4M commit/tree/source partition, subject, Git-note blob/body, ten-claim order, hermetic argv projection, grades, coverage, and disclosure. It independently reopens exact C4V and C3D beneath C4M under one stable outer HEAD.

The future C4 chain is `C4 → C4N → C4M → C4V → C3D`; the future C5 chain prepends C5's sealed-C4 check. These are validator contracts only until the future source trees execute their own complete runbooks and seal.

## Preserved C4 checkpoint

C4 product work remains retained in stash object `c419c38c608dc6bec7e23e0fc379df28d3cb2d47`, message `countershape-c4-pre-c4r-checkpoint-20260722`, with exact staged tree `3a8cdfd33168751fc38e219881fd874b0495a577`. Its development ledger archive is `.didrun-history/p07b-c-c4-development-pre-c4m-c419c38c608d/.didrun/`. Those facts support no C4 or C4N claim. C4 remains blocked until C4N seals; only then may the retained tree be reapplied and reconciled to the new parent and v19 authority.

## Intended claim map

| Claim | Type | Grade |
|---|---|---|
| `P07B-C C4N candidate phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4N independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4N generated phase cardinality and frozen-history defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4N sealed-C4M Git-note and C4V C3D ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4N unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4N cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4N cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4N exact seven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C4N scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C4N sealed-C4M predecessor and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

The private final root is `.countershape/p07bc-c4n-final`. Labels, types, order, hermetic argv, and exact ten-event ledger cardinality are checker-owned.

## Final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C4N`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C4N`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
4. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4n-sealed-c4m-note`
5. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
6. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
7. `/opt/homebrew/bin/node tools/verify-current.mjs`
8. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4N --source-final-gate`
9. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4N --credential-scan`
10. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4n-preseal-ledger`

These are command tails. Each final event must run only through the checker-rendered hermetic `didrun run --` invocation and be claimed immediately with its matching label and type.

## Nonclaims and handoff

C4N adds no product behavior, Go source, external API, runtime capability, sandbox, security boundary, performance result, or adoption claim. It proves the bounded declared table/catalog/checker contracts on the receipted tree, not arbitrary JavaScript safety, same-user isolation, or an atomic filesystem snapshot.

After C4N independently closes, preserve its exact receipt, reapply the retained C4 checkpoint without dropping the stash, and reconcile C4's parent, v19 phase authority, live ancestry command, status, and documentation. Any nonzero C4 event remains permanent evidence and keeps C4 unfinished.
