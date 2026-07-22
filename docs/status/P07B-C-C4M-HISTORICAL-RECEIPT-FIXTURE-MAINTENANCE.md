# P07B-C C4M — historical receipt-fixture maintenance

Classification: `DEFECT_REPAIR`

C4M owns exactly these 7 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C4M-HISTORICAL-RECEIPT-FIXTURE-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Their sorted-newline roster digest is `sha256:ec49dc68ce8fcfe92720fd2e8d6afa6d19d6d1c26aa9d27173dc2df736d06a87`.

- **Boundary:** active pre-seal `SOURCE_FULL` checker maintenance after sealed C4V and before C4. Classification: `DEFECT_REPAIR`.
- **Parent:** sealed C4V commit `861fd49a2b35b46c101dd6f18707f9c3525c6c7f`, tree `c2a560bf4c7bed6f93078af93714f979d06cf997`, is note-present with note blob `70c55639f4d0bbe031b3fd18166bde6162f35e3b`, note-body SHA-256 `b5f56269be341db27f7a0af32f158607ec306762d82c2529425f24cc1880a061`, `68/68 claims recorded-exact`, strict exit `0`, and recorded `secrets_override: true`.
- **Commit subject:** `fix: close descendant historical receipt fixtures`
- **Profile:** `SOURCE_FULL`
- **Receipts:** C3P `PRESENT`; C3 `PRESENT`; C6A `ABSENT`.
- **Product authority:** none.

Every intended C4M grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C4M itself.

## Why this boundary exists

C4 product work reached two clean development gates, then the full plan self-test exposed a checker defect before any C4 claim or commit. C3D separated the historical semantic phase from the live self-receipt pointer, but an omitted descendant corpus still partially fell back to historical bodies.

The historical C3U fixture supplied its historical `bodies` map while the self-receipt scanner correctly selected live C4. That partial map had no descendant-only C4 status body, so the checker stopped with:

```text
P07B-C C3 validated C3U-active absent fixture failed:
docs/status/P07B-C-C4-CLI-PROFILE.md: self-receipt corpus body is unavailable for C4
```

The failure is in fixture transport, not in the refusal policy and not in C4 product code. C4M therefore repairs the checker as an explicit interposed source boundary; it does not weaken a case, delete a claim, relabel a receipt, or reuse either successful development event.

## Preserved C4 checkpoint and permanent red evidence

- Development events 144 and 145 were unclaimed successful candidate-C4 and independent-scope gates.
- Event 146 is the permanent unclaimed nonzero plan-self-test result quoted above.
- Event 147 created the retained checkpoint.
- Stash object: `c419c38c608dc6bec7e23e0fc379df28d3cb2d47`.
- Stash message: `countershape-c4-pre-c4r-checkpoint-20260722`.
- Exact staged C4 tree: `3a8cdfd33168751fc38e219881fd874b0495a577`.
- Development ledger archive: `.didrun-history/p07b-c-c4-development-pre-c4m-c419c38c608d/.didrun/`.
- Byte-equal retained ledger original: `.didrun-history/p07b-c-c4-development-pre-c4m-c419c38c608d/.didrun-live-original/`.

These are development and checkpoint facts only. They support no C4 or C4M grade. C4 remains blocked until C4M seals and verifies strictly; only then may the retained C4 tree be reapplied and reconciled to its new parent.

## Complete descendant self-receipt corpus contract

A cross-boundary self-receipt scan consumes one complete captured live-boundary corpus; it never falls back to historical bodies or ambient descendant worktree bytes.

Explicit corpus keys must equal the complete declared Markdown roster: missing and foreign keys are rejected, explicit values take precedence, and ABSENT_FIXTURE_PATH means explicit absence rather than key omission.

The caller captures the live corpus once before constructing historical fixtures and carries a frozen versioned snapshot envelope through descendants under a private checker-owned key. Capture is canonical-root-bound, performs no-follow stable reads plus a second byte pass for every declared Markdown path, and requires HEAD, index-tree, and didrun-notes-tree identities to remain unchanged around the read set. Five injected hostiles cover foreign-root, byte, HEAD, index, and notes drift. Same-boundary scans continue to use the already-loaded phase bodies. Cross-boundary scans receive the captured total map explicitly; a partial map cannot borrow historical bodies, a missing key cannot become absence, and an explicitly absent existing path is tested against a detectable positive historical fallback trap. The fixed `C4V_COMPLETE_LIVE_SELF_RECEIPT_CORPUS` regression keeps an ordered 31-subcase manifest at `d7db6136c8c4bc4672bc6be374c5985ab4c5ad9990f48f4acde69f76d6bacc40`: exact C3U→C4 replay, cumulative C5, complete/missing/foreign/unavailable/explicit-absence states, malformed root/envelope/entry shapes, alias mutation, and explicit/embedded positive-descendant refusal. This remains a bounded sole-writer before/after observation, not an atomic snapshot or same-user ABA defense.

## v18 phase authority and fixed acceptance

The current schema is `countershape/p07b-c-unit-paths/v18`: 20 ordered phase rows at authority digest `sha256:dd134e870c11ed4e80eb8335680b59a0ed1553c45d04746fd2fdd722f3f975a4`. It inserts only `C4M` between C4V and C4; every other phase contract remains exact.

- The plan transition matrix admits exactly 8 accepted and 45 rejected snapshots.
- The independent scope transition matrix admits exactly 8 accepted and 28 rejected snapshots.
- The generated structural catalog contains 780 cases at digest `9933fe888704b2a8aaac645e9a362a86b2e662b18efc2d632bf064858763b825`.
- The executable catalog contains 785 cases at digest `1dfd07994201ce139426bd6daad625dc40d4f32b60e4c5022fa8c8cd6fda1496`: 405 accepted controls and 380 hostile rejections.
- Its fixed-regression manifest digest is `c2ab6de5fda27074dd1b9492fd10464c76f943c5913d07a037c6fe5fb58cbd78`.
- The cumulative plan-authority corpus contains 47 sorted Markdown paths at `sha256:3170c03ba6f8cda2910cd3a4f3437b88f6a30d0de991ca62484426d750af376e`.
- The active C4M self-receipt extension requires 735 rejections and 245 controls.
- Forward self-receipt coverage requires 5,355 rejections and 1,785 controls, plus seven full-plan controls.

Sealed C4V's v17 table, 722/726 catalogs, and 46-path corpus remain labeled historical. Sealed C3U/C3B's literal 44-path roster remains pinned at `sha256:1c5bf120f1ab948586bfa594e6ae634fc6acf7a5544febcccb2c4dc475a59a49`; its 15-form/five-control plus supplemental manifest is pinned at `ef4b8250cb0fbb666598b1926d3d17349b48ef913adced41831776821e99703c`, and its 1,320/440/8/2 matrix cannot expand with the current corpus. C4M changes none of their committed bytes or past receipts.

## Sealed-C4V and C3D ancestry

The C4M predecessor validator pins the exact sealed C4V commit, tree, source partition, subject, Git-note blob/body, 68-claim order, verbatim grades, hermetic argv projection, and strict disclosure. It then reopens sealed C3D through the C4V parent without allowing the outer HEAD to change between loads. The shared pure chain validators exercise C4M → C4V → C3D and C4 → C4M → C4V → C3D baselines plus C3D identity, note-object, and outer-HEAD hostiles; the existing authority tests retain raw and redaction-tolerant positives and reject diff, argv, grade, coverage, and disclosure drift.

C4's future validator is rewired to C4M → C4V → C3D. C5's future validator is rewired to C4 → C4M → C4V → C3D. Neither future contract becomes evidence before its own exact tree executes and seals.

## Intended claim map

| Claim | Type | Grade |
|---|---|---|
| `P07B-C C4M candidate phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4M independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4M generated fixture and descendant-corpus defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4M sealed-C4V Git-note and C3D ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4M unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4M cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4M cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4M exact seven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C4M scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C4M sealed-C4V predecessor and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

The private final root is `.countershape/p07bc-c4m-final`. Labels, types, order, hermetic argv, and ledger cardinality are checker-owned.

## Final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C4M`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C4M`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
4. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4m-sealed-c4v-note`
5. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
6. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
7. `/opt/homebrew/bin/node tools/verify-current.mjs`
8. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4M --source-final-gate`
9. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4M --credential-scan`
10. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4m-preseal-ledger`

These are command tails, not bare receipt commands. Each must execute only through the checker-rendered hermetic `didrun run --` invocation and be claimed immediately with its matching label and type.

## Nonclaims and handoff

C4M adds no product behavior, Go source, external API, runtime capability, security boundary, sandbox, performance result, or production/adoption claim. It does not prove arbitrary hostile JavaScript safety or same-user isolation. It repairs one phase-fixture authority seam and preserves the full existing gate.

After C4M independently commits, seals, exposes its note, and passes strict verification, reapply the retained C4 checkpoint without dropping it. Reconcile C4's parent, v18 phase authority, live command names, status, and documentation, then execute C4 from a fresh ledger. If any C4 check fails, that red remains permanent and C4 remains unfinished.
