# P07B-C C4J — bounded C5 verifier timeout maintenance

C4J owns exactly these 19 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/THREAT_MODEL.md`, `docs/VERIFICATION.md`, `docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md`, `docs/status/P07B-C-C4J-C5-VERIFIER-TIMEOUT-MAINTENANCE.md`, `research/deep-dive/p07b-c-c5-timeout-maintenance/01-empirical-failure.md`, `research/deep-dive/p07b-c-c5-timeout-maintenance/02-runtime-authority.md`, `research/deep-dive/p07b-c-c5-timeout-maintenance/03-governance-phase-machine.md`, `research/deep-dive/p07b-c-c5-timeout-maintenance/04-receipt-runbook-impact.md`, `research/deep-dive/p07b-c-c5-timeout-maintenance/05-synthesis.md`, `research/deep-dive/p07b-c-c5-timeout-maintenance/06-red-team.md`, `research/deep-dive/p07b-c-c5-timeout-maintenance/07-executive-briefing.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, `tools/check-p07b-c-unit-scope.mjs`, `tools/verify-current-selftest.mjs`, `tools/verify-current.mjs`, and `tools/verify-runtime-authority.mjs`. Their sorted-newline roster digest is `sha256:c0920287c9dc9998f0cae11f8beced9d134c74da786a10f1886b03b2a1a50239`.

- **Boundary:** active pre-seal `SOURCE_FULL` maintenance after exact sealed C4K and before the unchanged C5 product boundary. Classification: `OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR`.
- **Authority:** `OWNER_OUT_OF_BAND`; the owner authorized maintenance when repeated C5 aggregate verification reached the inherited outer deadline. Sealed C4K predeclared restoration of the exact C4J checkpoint in prose, but v23 did not machine-declare C4J: `predecessor-predeclared = true (sealed prose only)` and `machine-predeclared = false`. No external signed instruction artifact exists, so this authority remains unevidenced beyond the owner-attributed session instruction; C4J inherits or rewrites no C4K grade.
- **Parent:** sealed C4K commit `8c1ee2df955055ee40eec1f14f0c4d96f97d5361`, tree `99fe7082d05c96647a144785e0ddc5af1a09f52f`, note blob `2d039109e269968e2314934fb4e2c4cccb31a097`, note-body SHA-256 `cc276493b0d67bb9903d30591aaa99ed29df09d33154bed2f81faef717435766`, subject `fix: bound C3 architecture test timeout`, `12/12 claims recorded-exact`, every grade `TREE-EXACT`, strict exit `0`, and `secrets_override: true`.
- **Commit subject:** `fix: bound C5 architecture verification`
- **Product authority:** none. C4J changes only verifier/runtime execution authority, phase governance, documentation, and explicitly non-normative research; it changes no production Go implementation, generated Node product bytes, C5 product behavior, qualification case, C5 claim label/type/argv/order, Go inner timeout, package partition, parallelism, retry rule, or cumulative-row order.

Every intended C4J grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C4J itself.

## Defect and empirical ruling

The third C5 cumulative attempt reached `ETIMEDOUT` at `1,200,027 ms` on `architecture-p07b-c-c5` after rows 1–48 passed. Two prior unsealed development aggregates completed in `1,128,605 ms` and `1,120,532 ms`; the adjacent C5 architecture self-test also historically runs near that ceiling. This is evidence of an insufficient outer scheduling margin, not proof of product correctness and not a reusable C5 grade.

The failed 59-event C5 ledger remains permanent history at `.didrun-history/p07b-c-c5-cumulative-pass-3-timeout-20bc86917704/.didrun`. The exact C5 staged tree is retained in stash `1c9dd947f61e3f1fd4c4c2578da0e104f1a02b56`. Earlier C5 passes are stale after C4J and cannot be imported into a seal.

## Runtime authority

The child runtime default remains exactly `1200000` milliseconds. Only canonical step IDs `architecture-p07b-c-c5` and `architecture-p07b-c-c5-selftest` receive the exact parent-owned `1800000`-millisecond policy; the policy is outside `currentSteps`, cannot be supplied by a row, permits no retry, and changes no inner semantic deadline.

The runtime source retains exact `const childTimeoutMS = 20 * 60 * 1000;` as the historical C3 timeout-authority marker and derives `defaultChildTimeoutMS` from that single value. The extended constant remains separate. This preserves C4K's inherited C3 defensive inspection while allowing C4J's policy selector to name the default explicitly; it does not duplicate or relax the 20-minute authority.

The optional fifth argument to `childResult` is either omitted or an exact frozen ordinary/null-prototype record with one own data property, `timeoutMS`, at the sole allowed extended value. Arrays, accessors, inherited properties, extra string or symbol keys, unfrozen records, non-integers, coercible values, default-as-explicit values, and out-of-range values fail before revalidation or spawn. The runtime still constructs executable, argv, cwd, environment, encoding, output ceiling, and timeout; performs one pre-spawn and one post-spawn tool revalidation; executes exactly one child; and preserves status, signal, stdout, stderr, and error identity.

`tools/verify-current.mjs` owns a frozen null-prototype two-key policy and resolves it with `Object.hasOwn` from the canonical enumerated step ID inside the executor. C4J does not put timeout metadata in `currentSteps`, so its current verifier roster remains unchanged. C5 later activates the two already-sealed keys and separately owns the promised sensitive 9+3 row split; that later split rotates the internal roster from 61 to 62 without changing the frozen outer 79-command C5 receipt.

## Machine authority and evidence contract

The machine authority is `countershape/p07b-c-unit-paths/v24` with 26 ordered phase rows and authority SHA-256 `30530a2e05fa6b7fb74983756c24f22a898f4a6da6df75d447df17278b0152db`. The plan transition matrix has 14 accepted and 422 rejected cases; the independently implemented unit-scope matrix has 14 accepted and 409 rejected cases. The current Markdown authority corpus has 69 paths at `sha256:a53acdeef0e86b4865caab7822ea06cfa62742794ff1aaa5816023ef172cf901`, with 1065 positive-form rejections and 355 controls.

C4J is an explicit `SOURCE_FULL` row with exact parent C4K, no prefixes, C3P/C3 receipts present, and C6A absent. C5's parent becomes C4J. Hostile transition fixtures reject C4J skipping C4K, C4K skipping C4I, C5 skipping C4J, merge parents, stale outer heads, source-table reuse, and C6A skipping C5.

The frozen C4J claim labels and `--verify-c4j-sealed-c4i-note` command spelling remain byte-exact compatibility names. The implementation first proves exact sealed C4K as C4J's direct parent, then proves the named sealed-C4I and lower chain transitively; the spelling does not assert that C4I is C4J's direct parent.

Sealed C4K's exported note establishes a boundary-local legacy double-marker GOCACHE projection: all 12 `argv_preview` records use the exact double marker at hermetic position 6, while the retained raw final ledger records the canonical `.countershape/p07bc-c4k-final/gocache` assignment. Only the exact pinned C4K historical-identity path admits the raw token or that exact double marker at position 6; generic C4K-shaped candidates and C4J's future note remain suffix-preserving. The validator still requires the pinned note blob/body digest, full argv cardinality/order, fixed projections at every other admitted position, byte-exact command tails, and exact labels/types/grades. Suffix-preserving, bare, foreign, shifted, or tail-redacted forms fail on the exact historical path. This is a descendant-validator assumption defect and observed didrun export-compatibility finding whose didrun root cause is not established, not hidden-value attestation or gate weakening.

The seven tracked deep-dive files are non-normative evidence. They explain the empirical failure, alternatives, synthesis, and adversarial corrections, but no runtime or phase validator imports them as executable authority.

## Intended C4J claim map

| # | Claim | Type | Intended grade |
| ---: | --- | --- | --- |
| 1 | `P07B-C C4J candidate phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| 2 | `P07B-C C4J independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| 3 | `P07B-C C4J plan checker defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 4 | `P07B-C C4J sealed-C4I Git-note and C4H C4 C4P C4N C4M C4V C3D ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| 5 | `P07B-C C4J unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 6 | `P07B-C C4J verifier runtime timeout policy defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 7 | `P07B-C C4J repetition-runner defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 8 | `P07B-C C4J cumulative verification pass 1` | `tests-pass` | `UNRECEIPTED` |
| 9 | `P07B-C C4J cumulative verification pass 2` | `tests-pass` | `UNRECEIPTED` |
| 10 | `P07B-C C4J cumulative verification pass 3` | `tests-pass` | `UNRECEIPTED` |
| 11 | `P07B-C C4J exact nineteen-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| 12 | `P07B-C C4J scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| 13 | `P07B-C C4J sealed-C4I predecessor C4H C4 C4P C4N C4M C4V C3D ancestry and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Exact final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C4J`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C4J`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
4. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4j-sealed-c4i-note`
5. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
6. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
7. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --self-test`
8. `/opt/homebrew/bin/node tools/verify-current.mjs`
9. `/opt/homebrew/bin/node tools/verify-current.mjs`
10. `/opt/homebrew/bin/node tools/verify-current.mjs`
11. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4J --source-final-gate`
12. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4J --credential-scan`
13. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4j-preseal-ledger`

The final root is `.countershape/p07bc-c4j-final`. The exact staged tree must exist before command 1. Every isolated shell fence generated for C4J sets `umask 077` before any didrun, claim, commit, seal, note, strict, HTML, or archive command; no persistent outer shell is assumed. Any nonzero command, commit/seal error, nonzero strict grade, or post-stage drift permanently fails that attempt; the failed ledger remains history and a repaired tree starts fresh evidence at command 1.

## Nonclaims and continuation

C4J does not prove the 30-minute bound sufficient under future load, C5 product correctness, the sensitive 9+3 split, HTTP behavior, Node authenticity, production readiness, hostile same-UID isolation, containment, confidentiality, adoption, or maintainership. It proves only the bounded execution authority and regression surface after three complete current-tree cumulative passes.

C5 remains blocked until C4J seals. After strict-zero C4J closure, reapply the retained C5 checkpoint without dropping it, reconcile the 19 authority paths deliberately, activate the two exact policy keys, implement the 9+3 sensitive split, and generate fresh repetition and aggregate evidence. The frozen C5 79 labels/types/argv/order and `--verify-c5-sealed-c4-note` spelling do not change.
