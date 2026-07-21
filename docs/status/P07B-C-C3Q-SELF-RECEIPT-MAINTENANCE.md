# P07B-C C3Q — active-unit self-receipt maintenance

**State:** active pre-seal checker maintenance; every intended C3Q grade is `UNRECEIPTED`

- **Boundary:** active pre-seal `SOURCE_FULL` checker maintenance after sealed C3R and before the unchanged frozen C3B receipt unit. Classification: `DEFECT_REPAIR`.
- **Parent:** sealed C3R commit `424273b000f87925a87f0b3041b2e1e8c5750d39`, tree `0e4694362d0242a3b3da0ed2f8f78971ee595f6c`, is note-present with note blob `7d0c90aaec4f1a36d8603e5c1ff6195a02221314`, note-body SHA-256 `dcd8991ef2446da78137319d7069d89999eda9748338fbdb5f8927bb4460e09d`, `9/9 claims recorded-exact`, strict exit `0`, and recorded `secrets_override: true`.
- **Commit subject:** `fix: close C3B external self-receipt checks`

Every intended C3Q grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3Q itself.

## Trigger and permanent red history

The resumed three-path C3B draft reached its receipt-present plan self-test only after sealed C3R repaired note-preview compatibility. One hostile case inserted a positive active-descendant result into `docs/VERIFICATION.md`. The receipt-present checker branch inspected the C3 target status and handoff for active-descendant self-receipts, but not that external plan authority. The case therefore returned no diagnostic and the self-test correctly failed.

This is phase-asymmetric self-receipt enforcement, not a flaky command, didrun failure, security-policy workaround, or permission problem. C3B cannot own the repair: its exact `RECEIPT_RECONCILIATION` contract contains only the handoff, C3 target status, and C3 receipt declaration, while checker and unit-specification changes require `SOURCE_FULL`.

The failed C3B development ledger is permanent at `.didrun-history/2026-07-20-p07b-c-c3b-build-loop-pre-c3q/.didrun/`. It supports no C3B or C3Q claim. The current merged C3B draft remains preserved without dropping at stash object `1d677db4b9049084ea066263b1bb6c25ab62e5f0`; the earlier pre-C3R checkpoint `fc02f6722716f0600b1e1d7af28cb91925d1047c` also remains retained. Neither stash is evidence.

The first C3Q final-ledger attempt is also permanent at `.didrun-history/2026-07-20-p07b-c-c3q-final-attempt-1-ds-store/.didrun/` and supports no C3Q claim. Its first five commands exited `0`; cumulative event 5 exited `1` after macOS Finder metadata appeared during the run. The initial workspace artifact check had already passed, but inherited C1 topology validation later refused `internal/contractexec/.DS_Store` together with the expected production entries. The exact staged tree remained unchanged. The operator removed only the untracked `.DS_Store` artifacts, retained the failed receipt, and restarted from a fresh ledger instead of retrying, deleting, or relabeling the red event.

## Exact scope

C3Q owns exactly these 7 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C3Q-SELF-RECEIPT-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Their sorted-newline roster digest is `sha256:f5f27e661c1ace6516aba4b9d5766f4c6ab50b6602dbcb101bff9b6a4ebfe6fc`.

The machine scope advances to `countershape/p07b-c-unit-paths/v13`, inserting exact empty-prefix `SOURCE_FULL` unit C3Q between sealed C3R and frozen C3B. The source-final gate requires one added status file and six modified mode-`100644` regular files, with no missing path, extra path, rename, deletion, type change, unchanged modified row, prefix, or receipt-profile substitution.

C3B stays byte-contract-frozen to exactly:

- `docs/HANDOFF_MODE_C.md`
- `docs/status/P07B-C-C3-TARGET.md`
- `spec/verification/p07b-c-c3-receipt.json`

Its sorted-newline digest remains `sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67`, and its seven narrow claim labels/types remain unchanged.

## Repair contract

The plan checker freezes the exact plan-authority Markdown corpus from every Markdown body it loads through the required-text maps and status validation, plus the C1 status, C2 status, and didrun-status authorities that are read by phase helpers. The sorted 42-path registry has digest `sha256:16d63155a00c9e87b7eec79707ad6d47d6f5d70205d18243b8bc687e401e4b66`; a path cannot silently fall outside the scan merely because a helper reads it separately. One shared active-unit refusal function scans that corpus after phase detection:

- with no C3 receipt declaration, the sole live tuple is `C3Q_ACTIVE` and C3Q is the active unit;
- with the exact C3 receipt declaration, sealed-C3R ancestry, and one exact sealed C3Q child, the sole live tuple is `C3B_ACTIVE` and C3B is the active unit;
- complete older tuples remain recognizable only for precise stale-state diagnostics;
- missing, duplicated, partial, mixed, relocated, commented, fenced, raw-HTML-wrapped, or coherent downgrade tuples fail.

The refusal matrix covers positive active-unit commit/tree identities; sealed/strict-clean/verified/complete/closed status words; zero strict-result forms; recorded-exact claim counts and verdicts; evidence, grade, receipt, or result associations with positive grades; positive claim-table rows independent of column order; passed-verification statements; and active-unit receipt-marker forms. It preserves legitimate `UNRECEIPTED`, pending, blocked, future, modal, failed-development, and sealed-predecessor text.

The direct regression remains an end-to-end external-surface mutation in each phase. The declared all-surface/all-form cross-product is 2 phases × 42 paths × 15 positive forms: 1,260 required rejections, plus 420 legitimate pending/future/failed controls that must remain accepted. Eight supplemental C3Q/C3B source-commit/tree, verification-passed, and natural claim-count aliases must map to their exact stable diagnostics; two explicit strict-zero prohibition controls must remain accepted. It calls the same production refusal function over every exact corpus path, so a future new controlling Markdown body is not silently protected only in one phase. This is a bounded lexical grammar, not arbitrary sentiment or paraphrase inference.

The owning C3Q status must expose its required authority as visible Markdown; whole-document HTML comments, fences, indented code, and raw-HTML wrappers are hostile cases. The future C3Q-to-C3B renderer derives raw byte ranges from the same visible-Markdown projection used by lifecycle classification, removes inert comment/fence heading decoys with the entire old subsection, and then requires the canonical C3B subsection byte-for-byte. A receipt-present inert heading followed by stale C3Q planned prose must fail rather than truncate validation.

## Checker ancestry

C3R remains an exact sealed ancestor. Its fixed authority is:

- commit `424273b000f87925a87f0b3041b2e1e8c5750d39`;
- tree `0e4694362d0242a3b3da0ed2f8f78971ee595f6c`;
- direct parent `029c5cf43853eb3cb46520f6e0dea8431a3de5c0`;
- subject `fix: validate sealed C3 note previews`;
- note blob `7d0c90aaec4f1a36d8603e5c1ff6195a02221314`;
- note-body SHA-256 `dcd8991ef2446da78137319d7069d89999eda9748338fbdb5f8927bb4460e09d`;
- nine ordered claims, exact event indices, command projections, grades, and complete coverage;
- exact source diff of one added and six modified paths.

During C3Q pre-seal verification, C3R must be current `HEAD`. During later C3B verification, C3R must remain the exact direct child of C3, C3Q must be the unique direct child of C3R, and that C3Q child must be current `HEAD`. Merely dropping the historical C3R-current-head check would admit an arbitrary descendant and is explicitly rejected.

The C3B handoff renderer will bind C3Q's exact validated note blob OID and note-body SHA-256 as well as both predecessor notes' actual `secrets_override` booleans. Valid-shaped alternate C3Q note identities and outcome substitutions must fail against the already-rendered handoff. Each field is only an exact recorded note disclosure; it does not prove plain-seal-first sequencing, refusal, human review, secret absence, a general credential audit, or publication authority.

## Intended claim map

| Claim label | Claim type | Intended grade |
| --- | --- | --- |
| `P07B-C C3Q self-receipt plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3Q dual-phase and authority-surface defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3Q sealed-C3R Git-note compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3Q unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3Q cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3Q cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3Q exact seven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3Q scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3Q sealed-C3R predecessor and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Final command order

The final ledger must execute these command tails in this exact order. Each tail is prefixed by the checker-owned hermetic `/usr/bin/env -i` argv rooted at `.countershape/p07bc-c3q-final`, launched through `/opt/homebrew/bin/didrun run --`, and immediately followed by only its matching declared claim.

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs`
2. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c3q-sealed-c3r-note`
4. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
5. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
6. `/opt/homebrew/bin/node tools/verify-current.mjs`
7. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C3Q --source-final-gate`
8. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C3Q --credential-scan`
9. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c3q-preseal-ledger`

The first six commands are `tests-pass`; the final three are `command-succeeded`. Every command must use the same exact hermetic `/usr/bin/env -i` prefix rooted at `.countershape/p07bc-c3q-final`. The final chain event validates the preceding eight events, current staged tree, absent seal, and sealed C3R authority; because didrun appends its own event only after the command exits, it cannot self-prove its outer launcher.

The frozen corpus is inspectable without source archaeology via `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --print-plan-authority-corpus`. Report mode recomputes the sorted-path digest and refuses success if it differs from the declared constant; its deterministic output names all 42 paths, the computed exact digest, and the bounded footer `scope: positive-receipt-form refusal only; not a general Markdown or security audit`. This read-only inventory command is not part of the nine-event final ledger and earns no didrun grade by itself.

## Nonclaims and operator findings

C3Q changes no verifier roster, product source, target, store, Git materialization, host/runtime measurement, process behavior, cache lifetime, parallelism, sensitive-package classification, repetition semantics, lock semantics, didrun installation, sealed C3/C3R bytes, or frozen C3B contract. It proves only bounded positive-receipt refusal over the exact loaded Markdown corpus. It does not prove arbitrary-repository content linting, secret absence, security review, runtime correctness, production readiness, adoption, or maintainership.

One read-only critic mistakenly invoked didrun's help-only parser while mapping an earlier boundary. No event, claim, seal, note, lock, Git, index, stash, or tracked-file change resulted. That remains an operator-discipline defect and `UNRECEIPTED` process finding; it is not presented as tool or product evidence.

During the C3Q development loop, the root operator accidentally launched the unit-scope self-test and sealed-C3R compatibility check as concurrent didrun wrappers. Both commands exited `0`; didrun appended them as ordered events 21 and 22, and a subsequent wrapped `Session.verify_chain()` check exited `0`, so no development-ledger chain corruption was observed. Their `started_at` intervals overlap, however, proving the current installation does not prevent concurrent verifier work. These events remain development-only, will not enter the fresh final ledger, and are live S6 evidence for evaluating the proposed fail-fast single-verifier lock at the next unit boundary.

Historical didrun interruption findings remain unchanged: interrupted wrapper commands emitted tracebacks, appended no event, and required explicit review of stale lock state. C3Q relies only on normally completed wrapper events in a fresh final ledger.
