# P07B-C C3M — C3P receipt-phase checker maintenance

## State

- **Boundary:** active pre-seal `SOURCE_FULL` checker maintenance after sealed C3V and before unchanged C3PB. Classification: `DEFECT_REPAIR`.
- **Parent:** sealed C3V commit `725933fa3c7826a281e84b51f729736cbcfac6ec`, tree `21f0e1bf46e3776e1b496a9c2783f549923c4510`, is note-present and strict-clean with `8/8 claims recorded-exact` and strict exit `0`.
- **Source under receipt:** C3P remains immutable at commit `f7b6e6bda7a8864969415ab8636c495902e78dd9`, tree `2d5555db63d46c837cf4e12f2dab2d04a41f4654`.
- **Preserved draft:** the blocked exact C3PB receipt draft is retained without dropping at stash object `5eeb848c334ad3a8a44e4cf61fa298e452188ab1`; it is not live authority and supplies no grade.
- **Scope:** seven exact mode-`100644` paths, empty prefixes, sorted-newline digest `sha256:58e1f6fba95cf3b4312136eea5bb5c64af89a6958711aca84181a1874aa03240`.
- **Commit subject:** `fix: preserve interposed receipt-phase authority`

Every intended C3M grade remains `UNRECEIPTED` until the exact staged tree commits, didrun seals it, the Git note is independently readable, and `NO_COLOR=1 didrun verify --strict` exits `0`. This status cannot receipt C3M itself.

## Trigger and evidence disposition

The source-receipt-field-validated C3PB draft exposed a checker lifecycle defect before any C3PB path was staged and before any C3PB didrun ledger, claim, commit, or seal existed. Its source receipt fields were independently reconstructed against the sealed C3P note, archive, and exact-commit HTML, but that evidence cannot excuse a phase-unstable plan self-test or make the draft globally valid. The draft was stashed by full object identity so the checker can be repaired at a separate boundary without widening or silently rewriting the frozen three-path receipt unit.

No C3P receipt declaration or C3P source-receipt block is live during C3M. The stash is preserved input for later deliberate restoration only. It is not proof of C3PB correctness, and no command from the blocked draft supports a grade.

## Root cause

`runC3PReceiptSelfTest()` correctly validates the actual working-tree phase and also constructs an absent-receipt phase. When a receipt declaration became present, however, `syntheticC3PAbsentPhaseFixture()` reopened both the C3P status and the C3P-era handoff from the sealed source commit. The status reopening was correct; the handoff reopening was not. That historical handoff predates the required C3V throughput disposition and this C3M maintenance authority, so the synthetic phase discarded interposed descendants and could not pass the current full plan.

The checker must never reopen the C3P-era handoff. This is a plan self-test lifecycle defect, not a C3P semantic defect, a receipt-evidence defect, a verifier failure, a didrun failure, or a runtime failure.

## Repair and hostile coverage

The repair keeps both receipt phases closed on every run:

1. The ordinary `checkPlan()` baseline still validates the actual working tree, whether the receipt declaration is absent during C3M or present during C3PB.
2. `readSealedC3PStatusFixture()` uses config-isolated Git with replacement objects disabled, re-proves the pinned C3P tree, and reopens only the bounded canonical C3P status blob.
3. `parseC3PReceiptBlock()` counts raw start and end markers independently. It admits only `0/0` or exactly `1/1`, requires both markers to be standalone LF lines in forward order, extracts one exact block, and returns the byte-exact remainder. Nested/duplicate markers, partial cardinality, reversed order, inline markers, or trailing marker-line content fail.
4. `c3pReceiptHandoffBlock()` defines the one canonical receipt payload and order. Receipt-present checking compares the extracted block byte-for-byte, requires the coherent C3PB-active operational phase, and rejects every exact receipt identity, disclosure, C3P-specific heading, or row that survives outside the block. The generic Markdown table header is deliberately not treated as C3P-specific residue because historical receipt tables use it elsewhere.
5. `withoutC3PReceiptBlock()` derives the synthetic absent-phase handoff from the current handoff by consuming only that parsed block. `syntheticC3PAbsentPhaseFixture()` combines the sealed source status with the byte-exact remainder and an absence sentinel. Empty or malformed JSON is never reclassified as absence.
6. The full absent-phase `checkPlan()` baseline must pass after stripping. Marker-free identity, one-block removal, and append/remove byte round-trip are positive cases; every malformed topology is a negative case.
7. C3V and C3M authority survives the absent-phase reconstruction. The suite proves both carried anchors in absent and receipt-present fixtures, then mutates each authority independently in both phases and requires full-plan rejection.
8. The synthetic receipt-present handoff transitions to explicit C3PB-active state and models C3M as a closed interposed prerequisite. Its six operational state, dirty-scope, ledger, verification-battery, heading, and receipt-presence transitions are one exact phase tuple: coherent all-C3M input transitions once, coherent all-C3PB input is byte-identical on rerun, and every mixed, missing, duplicated, or stale tuple fails. A second full receipt-present reconstruction must reproduce the same three overridden files byte-for-byte and pass `checkPlan()` again. Existing declaration, source-diff, parent, Git-note, argv, grade, coverage, local-snapshot, documentation, no-recursion, and self-receipt hostile cases remain required, with additional externalized-payload, receipt-residue, and stale-operational-state negatives.

The verifier and repetition-runner files remain byte-for-byte frozen at the C3V-pinned digests. C3M changes plan/scope checker authority only; it adds no product or runtime behavior.

## Exact C3M scope

C3M owns exactly these 7 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C3P-RECEIPT-PHASE-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Their sorted-newline roster digest is `sha256:58e1f6fba95cf3b4312136eea5bb5c64af89a6958711aca84181a1874aa03240`.

Prefixes are empty. Schema v6 orders `C3P -> C3V -> C3M -> C3PB -> C3`. C3M is `SOURCE_FULL` because it changes checker and unit-scope authority. C3PB remains unchanged: its exact three-path `RECEIPT_RECONCILIATION` roster has digest `sha256:531cbe600cf2747752749890ea9a15c3d0126e498da6838d4f6faeb1a0d7e885`, its seven labels and types remain frozen, and it cannot run until C3M strict-closes.

## Intended C3M claim map

| Exact claim label | Claim type | Pre-seal grade |
| --- | --- | --- |
| `P07B-C C3M receipt-phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3M pre-receipt and receipt-phase checker self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3M unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3M cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3M cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3M exact seven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3M scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3M preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

Only these eight labels may be claimed. The final chain gate pins the first seven events' absolute argv, label/type, index, empty pathspec, complete wrapper capture, green exit, and one unchanged staged tree; it also requires no premature seal and separately proves its own live invocation before becoming event eight.

## Throughput authority remains closed

C3M does not reopen C3V's proposal dispositions. The private Go cache stays fresh for this Darwin/cgo tree; qualified direct general work remains `-p=2`; sensitive and nested Go work remains serialized; the fail-fast O_EXCL verifier lock retains review-first stale handling; and narrow receipt units retain their explicit seven-claim profile. C3M edits none of those implementations or settings and claims no new qualification or performance result.

## Nonclaims and gate back to C3PB

C3M changes no Countershape runtime, target, storage semantics, Git materialization, Node authority, host measurement, attempt authority, process execution, result classification, CLI, dashboard, external API, or existing receipt grade. It does not prove secret absence, security review, portability, production readiness, adoption, or maintainership. It does not grade the stashed C3PB draft or make local HTML and ignored-ledger snapshots portable.

After a genuine multi-pass build loop and independent read-only review, stage exactly the seven C3M paths. Archive the zero-claim development ledger, start a fresh final ledger, run and immediately claim only the eight rows above, commit the exact staged tree, seal it, inspect the Git note, loop strict verification to exit `0`, generate an exact-commit HTML report, and archive the final ledger. Only then revalidate stash object `5eeb848c334ad3a8a44e4cf61fa298e452188ab1`, apply it without dropping, preserve the non-overlap C3PB bytes, and deliberately merge the handoff so it retains C3V/C3M history while adding the one exact C3P source-receipt block.
