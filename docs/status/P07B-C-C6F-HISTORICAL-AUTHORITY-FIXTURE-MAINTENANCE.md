# P07B-C C6F — historical C6 authority-fixture maintenance

**State:** active pre-seal `SOURCE_FULL` defect repair. Every C6F grade is `UNRECEIPTED` until the exact staged tree completes its own ten-event didrun ledger, commit, seal, readable Git note, strict verification, HTML report, and ledger archive.

## Boundary and authority

C6F is the owner-authorized child of exact sealed C6A and the required parent of C6M. It repairs two related ambient-phase defects discovered sequentially: a historical C3Q fixture that leaked future C6 files, and frozen C6A architecture rows that read descendant governance bytes as though they still belonged to C6A. It neither changes nor republishes Countershape product behavior.

- **Commit subject:** `fix: isolate historical C6 authority fixtures`
- **Direct parent:** exact sealed C6A commit `3e9643f657248e0d5ff2c4bf0880218efbaeecd8`, tree `226a1e23da7eded933b3faabd4e8040576b2a2e0`
- **Parent note:** blob `b110b998a56c89d77bec3f87a48f64f1f21a700e`, body SHA-256 `096e9ae74e9a9ccfb984018f792a59bac3eb8ae24c550d83bb273e6b5ff064c7`
- **Transition source:** `OWNER_OUT_OF_BAND`
- **Classification:** `OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR`
- **Predecessor declaration:** `NONE`
- **Provenance:** `UNEVIDENCED`, disclosed as `OWNER_ATTRIBUTED_SESSION_INSTRUCTION_ONLY_NO_QUALIFYING_PREEXISTING_ARTIFACT`
- **Authentication:** `NOT_ESTABLISHED`
- **Signed authorization:** `NOT_IMPLEMENTED`

The owner instruction was real, but no qualifying instruction artifact pre-existed in the direct-parent tree. This candidate-owned file cannot authenticate or retroactively evidence it. C6A source boundary is sealed, while `Receipt C6A` remains `ABSENT`; C6F cannot receipt, transfer, relabel, renew, or strengthen any C6A grade.

The current machine authority is `countershape/p07b-c-unit-paths/v27`, with 41 ordered unit rows and 29 receipt-phase rows at receipt-phase authority digest `sha256:ba3e4836c50f6a4d7f4f5ef8563ba8fb8e1b0ab6d602bab6017656daa2d57faa`.

## Historical-fixture repair

The fixed `C3Q_EXTERNAL_SELF_RECEIPT` regression reconstructs a C3B-era plan surface. Its HANDOFF was explicitly rewritten, but the fixture still allowed later C6 files to fall through to the ambient worktree. Once C6M staged `spec/verification/p07b-c-c6a-source-authority.json`, that historical replay observed a future manifest and correctly refused. Unstaging the manifest would invalidate C6M; changing its phase capsule would invalidate C6M; and validating only sealed C6A would not validate the candidate tree.

C6F makes that historical fixture explicit-total for every later C6 authority it may query. Its exact ordered absence authority is:

1. `docs/captures/p07b-c/c6a-evidence-summary.json`
2. `docs/status/P07B-C-C6-EVIDENCE.md`
3. `spec/verification/p07b-c-c6a-receipt.json`
4. `spec/verification/p07b-c-c6a-source-authority.json`

The ordered JSON-lines path digest is `36d29096c536a24bd5882c4dd2c7e305bae19510f23033897a9ccb4a8cff1218`. Every entry must be the exact `ABSENT_FIXTURE_PATH` sentinel. The defensive manifest executes 11 ordered cases—one baseline, four missing-path mutations, four non-sentinel substitutions, one foreign path, and one ordering mutation—at digest `983fed85578bef0dc1eb9c3ce1b9b6622557083d71c0a767edd48bc5565c9683`. The fixed C3Q integration separately requires its complete override roster to be exact HANDOFF plus those four sentinel entries before historical `checkPlan()` execution, so deleting the installation cannot leave the isolated helper test green. No omitted key may reopen descendant worktree bytes.

The live-plan semantics, receipt projections, and C6A source contract remain unchanged. The v27 generated phase catalog, transition hostiles, and C6F integration controls expand only for the new C6F row and this isolation oracle; no historical case or rejection is removed.

## Frozen-C6A replay repair

The first repaired cumulative pass reached the live B architecture row only after all product rows had passed. B delegated to the C checker with no phase argument; that legacy entrypoint intentionally ran the full frozen C6A contract, which then rejected C6F's sole live HANDOFF capsule. Reintroducing the old C6A capsule would violate live-cursor cardinality, and changing `tools/check-p07b-c-architecture.mjs` would rewrite a sealed C6A source input.

C6F instead makes the authority split explicit. Live B now delegates to exact `--c1`, so B and its defensive self-test continue to scan the current tree. All Go, B, C1–C5, plan, scope, receipt, and terminal-authority rows remain `LIVE`. Exactly two verifier rows—`architecture-p07b-c-c6` and `architecture-p07b-c-c6-selftest`—declare `SEALED_C6A` and invoke `tools/check-sealed-c6a-architecture.mjs`.

The sealed runner pins commit `3e9643f657248e0d5ff2c4bf0880218efbaeecd8`, tree `226a1e23da7eded933b3faabd4e8040576b2a2e0`, notes-ref commit `ba1f89377012e444d30b0520e297241eabacfb48`, and a 567-entry raw-tree manifest at `sha256:a6c028ab3302ba215d16c90f3a27cb0583e631584382f7a70bfaed59de8cfe52`. It materializes literal blobs without invoking `git checkout` or `git archive`, without linked-worktree registration, and without applying attributes or filters; independently checks every Git blob identity; rejects unsafe modes, paths, collisions, and framing; supplies a private detached Git control directory with the pinned notes ref and content-addressed object alternate; rechecks the snapshot after the child; and removes it after every handled terminal outcome. It does create a private non-bare working directory; abrupt process termination or failure before cleanup protection is established may leave that root behind for operator removal. Its marker names `root_authority=SEALED_C6A`. Those two rows prove sealed-C6A architecture/evidence compatibility under the current admitted toolchain, not current C6F governance-byte conformance.

This is a checker/verifier-authority repair, not a product, evidence, or receipt repair. Because C6F changes the phase specification, both generic phase checkers, the live B inheritance seam, and the cumulative verifier, the narrower `NON_PRODUCT_MAINTENANCE` profile is inapplicable. The boundary is a twelve-path `SOURCE_FULL` unit and runs the full 65-row verifier once with 63 live rows and exactly two labeled sealed-C6A replays.

## Final-ledger Darwin diagnostic repair

The first C6F final attempt ran on staged tree `f6c30a8f641a0c6b747a5fc2b4c3415461af7632`. Commands 1–7 passed and were immediately claimed; the cumulative verifier event completed all 65 rows in 8,993,502 ms. Unclaimed command 8 then failed closed before scope evaluation: its first Git inventory returned status zero but carried 344 stderr bytes from Apple developer tooling—an FSEvents startup diagnostic and a `DARWIN_USER_CACHE_DIR` `confstr(3)` fallback diagnostic. That eight-event attempt is preserved byte-for-byte at `.didrun-history/p07b-c-c6f-final-f6c30a8f641a/.didrun/`. It produced no commit, seal, note, strict result, HTML report, or reusable C6F grade.

This observed host-boundary defect is routed `DEFECT / REPAIRED_IN_C6F`. The scope checker now independently implements the same narrow Darwin diagnostic grammar already used by the C6 evidence reader: stderr is admitted only when it is at most 16 KiB, has no leading UTF-8 BOM, is otherwise fatal UTF-8, is LF-terminated with no CR, and has every complete line exactly matching one of the three pinned Apple `confstr`/`xcodebuild` forms. Empty stderr remains accepted. Unknown, mixed, malformed, BOM-prefixed, control-bearing, over-bound, or differently framed bytes remain fatal; nonzero status, signal, spawn error, non-buffer streams, and stdout handling are unchanged. The defensive self-test covers clean and four admitted layouts plus sixteen refusal/result cases. This is not general warning suppression, host-tool attestation, or permission to discard unclassified stderr.

## Exact sealed C6A authority

The parent source contract is reopened for compatibility only. C6A's exact subject is `test: close P07B-C cumulative evidence`; its direct parent is sealed C5V commit `e7f51c0a8fbd2d6fbe8eacb4a7f0d46f010af7ba`. The note records eleven ordered `TREE-EXACT` grades, strict exit `0`, and `secrets_override: true`. C6F validates that immutable commit, tree, parent, subject, note object, note-body digest, labels, types, and hermetic argv previews before it can proceed. Those facts remain C6A facts only.

The future C6F sealed-note consumer is also frozen before C6F seals. The plan and scope checkers independently project C6F's subject, ten labels/types, ten command tails, final-root identity, and position-specific redaction policy to the same sealed-source contract digest `sha256:84227e43b4534cc6977d5b5d274ee415f118af98c36e3e390c86d585d11aa5fb`. Each checker drives a synthetic C6F note through its own generic note validator and rejects command-tail drift. This proves checker parity for the declared projection; it is not a receipt or proof of the future Git note's existence.

## Scope

C6F owns exactly these twelve mode-`100644` paths:

1. `docs/HANDOFF_MODE_C.md`
2. `docs/PROMPT_PACK.md`
3. `docs/VERIFICATION.md`
4. `docs/status/P07B-C-C6F-HISTORICAL-AUTHORITY-FIXTURE-MAINTENANCE.md`
5. `spec/verification/p07b-c-unit-paths.json`
6. `tools/check-p07b-b-architecture-selftest.mjs`
7. `tools/check-p07b-b-architecture.mjs`
8. `tools/check-p07b-c-plan.mjs`
9. `tools/check-p07b-c-unit-scope.mjs`
10. `tools/check-sealed-c6a-architecture.mjs`
11. `tools/verify-current-selftest.mjs`
12. `tools/verify-current.mjs`

The sorted-newline roster digest is `sha256:6f4d8c5b02df2d2e0df6e8f4b91b66c9f1465bce427939e8ccc1442d64b85c9c`. Prefixes are empty. C6A's evidence artifacts, frozen C6 architecture checker and runtime-authority implementation, product/runtime source, receipt declaration, and source-authority manifest are outside this roster and remain byte-stable.

## C6F final manifest

| # | Claim | Type | Intended grade |
|---:|---|---|---|
| 1 | `P07B-C C6F candidate phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| 2 | `P07B-C C6F independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| 3 | `P07B-C C6F historical C6 authority fixture defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 4 | `P07B-C C6F sealed-C6A Git-note source authority compatibility` | `tests-pass` | `UNRECEIPTED` |
| 5 | `P07B-C C6F unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 6 | `P07B-C C6F cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| 7 | `P07B-C C6F live-current verification and sealed-C6A architecture replay` | `tests-pass` | `UNRECEIPTED` |
| 8 | `P07B-C C6F exact twelve-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| 9 | `P07B-C C6F scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| 10 | `P07B-C C6F sealed-C6A predecessor and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Exact final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C6F`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C6F`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
4. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c6f-sealed-c6a-note`
5. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
6. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
7. `/opt/homebrew/bin/node tools/verify-current.mjs`
8. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C6F --source-final-gate`
9. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C6F --credential-scan`
10. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c6f-preseal-ledger`

Run each command once in the generated C6F runbook and claim it immediately; any nonzero command stops that ledger, supports no claim, and requires a repaired fresh boundary.

The first seven prove the phase transition, the exact four-entry/11-case historical-fixture authority plus its C3Q installation invariant, the cross-checkable future C6F note contract, exact sealed-C6A authority, both generic checker self-tests, and the full verifier roster with live current-tree rows plus exactly two sealed-C6A architecture replays. Claim 3 is bounded to those named paths, cases, and integration rule; it is not a general fixture sandbox. Claim 7 does not relabel the two sealed replays as current C6F architecture evidence. The last three prove exact staged ownership, structured credential-pattern absence over the twelve blobs, and final ledger/ancestry coherence.

## Claim ceiling and continuation

C6F may claim explicit-total isolation of the named four-path historical C6 authority fixture, exact sealed-C6A compatibility replay, current phase/checker/verifier coherence, live current-tree verification for every row not explicitly labeled `SEALED_C6A`, exact staged scope, and exact chain integrity. Other historical `checkPlan()` inputs remain ambient and outside that four-path claim. C6F does not claim that the two frozen replay rows validate current C6F governance bytes, a C6 product change, a C6A receipt, portable local-evidence availability, secret absence, confidentiality, hostile same-UID isolation, security certification, production readiness, adoption, or maintainership.

The next permitted edge is to C6M only after this boundary's own exact seal. C6M may then restore its parked five-path payload and bind sealed-C6A source facts in the canonical source-authority manifest while reopening C6F as its direct predecessor. C6B remains a later separate receipt-reconciliation boundary. The intended topology is `C6A → C6F → C6M → C6B`; no grade crosses those edges automatically.
