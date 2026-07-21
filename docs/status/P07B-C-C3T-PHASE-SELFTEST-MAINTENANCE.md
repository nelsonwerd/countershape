# P07B-C C3T — receipt-phase self-test maintenance

**State:** active pre-seal checker maintenance; every intended C3T grade is `UNRECEIPTED`

- **Boundary:** active pre-seal `SOURCE_FULL` checker maintenance after sealed C3Q and before the unchanged frozen C3B receipt unit. Classification: `DEFECT_REPAIR`.
- **Parent:** sealed C3Q commit `818ed90ebbf99ae38916307d9f2342921d38ebbf`, tree `157241ac2ea7c390b2f67a788b26b9f7666b2893`, is note-present with note blob `06cd81c05ce2e6f2348d0765298846ac048caf7e`, note-body SHA-256 `b61f53351614361f3646186827ed96e7aace8980bcd3be52f470a366a59fe855`, `9/9 claims recorded-exact`, strict exit `0`, and recorded `secrets_override: true`.
- **Commit subject:** `fix: make C3 receipt self-test phase-stable`

Every intended C3T grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3T itself.

## Trigger and permanent red evidence

The first post-C3Q C3B build-loop rehearsal reached its outer receipt-phase self-test with the production plan event already green. The next wrapped event exited `1` at `P07B-C phase fixture C3 absent downgrade active state anchor count 0`. The test driver had reused the live C3B receipt-present handoff while trying to exercise an absent-receipt downgrade, so the phase-owned anchor was correctly absent from that document. A later operator shell-control mistake declared the failed event rather than its predecessor; that declaration is retained as an operator finding and supports no product or maintenance claim.

The permanent red ledger is `.didrun-history/2026-07-20-p07b-c-c3b-build-loop-post-c3q-selftest-red/.didrun/`. Its populated hermetic run root is archived at `.countershape/history/p07bc-c3b-post-c3q-selftest-red-root`. Neither is rewritten, retried in place, relabeled, or counted toward C3T. The exact post-C3Q C3B draft is retained without dropping at stash object `eab9839c902abf52e9729144a2244740228a41eb`; a stash is recovery material, not evidence.

The failure narrows sealed C3Q's evidence honestly. C3Q proved its production active-unit refusal helper and its direct C3Q/C3B renderer cases on the sealed 42-path corpus. The later failure disproved the broader claim that every outer-driver absent mutation was already phase-neutral. C3T repairs only that fixture boundary and does not weaken, delete, relabel, or replace any sealed C3Q receipt.

## Exact scope

C3T owns exactly these 7 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C3T-PHASE-SELFTEST-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Their sorted-newline roster digest is `sha256:496788860db6c42f5acc50de0de02be8a63bcb8e11f99fb24d59d9c0e016477e`.

The machine scope advances to `countershape/p07b-c-unit-paths/v14`, inserting exact empty-prefix `SOURCE_FULL` unit C3T between sealed C3Q and frozen C3B. The source-final gate requires this one added status file and six modified mode-`100644` regular files, with no missing path, extra path, prefix, rename, deletion, type change, unchanged modified row, or receipt-profile substitution.

C3B stays frozen to exactly three paths at `sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67`; its `RECEIPT_RECONCILIATION` profile and seven labels/types do not change. C3T changes no verifier roster.

## Repair contract

The self-test first reconstructs one validated C3T-active absent fixture from the live lifecycle. When the checkout is already receipt-present, the inverse renderer removes only the lifecycle-owned C3B subsection and metadata, the exact C3 receipt projection, and the terminal receipt declaration/block. It then requires a clean full-plan baseline before any hostile mutation runs. Every absent-phase mutation starts from a fresh copy of that validated C3T-active absent fixture; it never depends on the current handoff already containing an absent-phase anchor.

The existing forward renderer turns that fixture into the canonical C3B-active receipt-present projection. The driver proves absent → present, present → absent, and present → absent → present byte equality over the three controlled receipt paths. Exact-one anchors and visible-Markdown topology remain mandatory, so comment, fence, raw-HTML, duplicate, missing, or relocated authority cannot make a transition look successful.

Receipt-present admission requires the canonical one-line receipt declaration and exact terminal handoff block. Pretty, reordered, duplicate-key, nonterminal, or trailing-content forms fail rather than being silently normalized. When a live C3 receipt exists, the regenerated status, handoff, and declaration must equal the captured live bytes. Ordinary sealed-descendant plan coherence accepts exact C3T ancestry after C3B commits; only the dedicated C3B preseal chain gate requires C3T to be current `HEAD`.

The current active-unit corpus has 43 exact Markdown paths at sorted-newline digest `sha256:52e78638d70f4e229e0b54c0a8311b6e0d1d2acc0d4e439cacf10cde65c62959`. Its declared matrix is 2 phases × 43 paths × 15 positive forms: 1,290 required rejections, plus 430 pending, modal, future, failed-development, or prohibition controls that must remain accepted. These counts describe bounded positive-receipt-form refusal, not general Markdown understanding or a security audit.

## Fixed predecessor authority

C3T binds sealed C3Q as the exact current parent: commit `818ed90ebbf99ae38916307d9f2342921d38ebbf`, tree `157241ac2ea7c390b2f67a788b26b9f7666b2893`, direct parent `424273b000f87925a87f0b3041b2e1e8c5750d39`, subject `fix: close C3B external self-receipt checks`, note blob `06cd81c05ce2e6f2348d0765298846ac048caf7e`, note-body SHA-256 `b61f53351614361f3646186827ed96e7aace8980bcd3be52f470a366a59fe855`, exact seven-path source diff, and nine ordered claims with complete event coverage. C3T pre-seal validation requires that identity to remain current `HEAD`. The later C3B preseal chain gate requires one exact sealed C3T child as current `HEAD`; after C3B commits, ordinary plan coherence instead requires that exact C3T child to remain an ancestor.

## Intended claim map

| Claim label | Claim type | Intended grade |
| --- | --- | --- |
| `P07B-C C3T phase-selftest plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3T receipt-phase fixture defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3T sealed-C3Q Git-note and ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3T unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3T cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3T cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3T exact seven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3T scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3T sealed-C3Q predecessor and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Final command order

The final ledger must execute these command tails in this exact order. Each tail is prefixed by the checker-owned hermetic `/usr/bin/env -i` argv rooted at `.countershape/p07bc-c3t-final`, launched through `/opt/homebrew/bin/didrun run --`, and immediately followed by only its matching declared claim.

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs`
2. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c3t-sealed-c3q-note`
4. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
5. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
6. `/opt/homebrew/bin/node tools/verify-current.mjs`
7. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C3T --source-final-gate`
8. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C3T --credential-scan`
9. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c3t-preseal-ledger`

The first six commands are `tests-pass`; the final three are `command-succeeded`. Event nine may bind the exact sealed C3Q predecessor, the preceding eight live events, one exact staged tree, and an absent seal. Because didrun appends an event only after the command exits, it cannot prove its own outer wrapper.

## Nonclaims

C3T changes no product source, target, store, Git materialization, runtime behavior, process edge, verifier roster, cache or parallelism setting, sensitive-package classification, repetition semantics, lock semantics, didrun installation, sealed C3/C3R/C3Q bytes, or frozen C3B contract. It proves no process start, runtime freshness, secret absence, general security property, production readiness, adoption, or maintainership. All such outcomes remain outside this maintenance boundary and `UNRECEIPTED`.
