# P07B-C C3U — cross-phase receipt-fixture maintenance

**State:** active pre-seal checker maintenance; every intended C3U grade is `UNRECEIPTED`

- **Boundary:** active pre-seal `SOURCE_FULL` checker maintenance after sealed C3T and before the unchanged frozen C3B receipt unit. Classification: `DEFECT_REPAIR`.
- **Parent:** sealed C3T commit `9d8430aa70b1536a85dd7352a3d339667f2c1e88`, tree `3be4d7cde4d043b87438506483ffb8c256719df2`, is note-present with note blob `d9e371310b360b183d890ed401cb36bd877becda`, note-body SHA-256 `05914bf85ce1d3bc48f5debf902babb526faf2500af318c35f08ffdf071ff735`, `9/9 claims recorded-exact`, strict exit `0`, and recorded `secrets_override: true`.
- **Commit subject:** `fix: preserve terminal C3 receipts in legacy fixtures`

Every intended C3U grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3U itself.

## Trigger and permanent red evidence

The first post-C3T C3B build-loop rehearsal passed the live plan and then failed its receipt self-test at `docs/HANDOFF_MODE_C.md: C3 source receipt block must be the exact terminal handoff block`. The permanent zero-claim ledger is `.didrun-history/2026-07-20-p07b-c-c3b-build-loop-post-c3t-c2-terminal-red/.didrun/`. It remains unchanged and supports no C3B or C3U claim.

The canonical post-C3T C3B draft is retained without dropping at stash object `cca3045b55ddc00e011c4331f4731bd1a433ea36`; earlier drafts remain retained separately. A stash is recovery material, not evidence. The red event isolated a fixture-composition defect: the live plan was already valid, but the C2 legacy fixture renderer appended a historical receipt after the already-canonical terminal C3 receipt block. This boundary repairs the test driver, not the C3 receipt payload or C3B source-receipt declaration.

## Exact scope

C3U owns exactly these 7 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C3U-CROSS-PHASE-RECEIPT-FIXTURE-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Their sorted-newline roster digest is `sha256:7af7456929b0db08abc2a31bd1cb62d84002c1a6bcdda3f4a8b4896f6b8c30cd`.

The machine scope advances to `countershape/p07b-c-unit-paths/v15`, inserting exact empty-prefix `SOURCE_FULL` unit C3U between sealed C3T and frozen C3B. The source-final gate requires this one added status file and six modified mode-`100644` regular files, with no missing path, extra path, prefix, rename, deletion, type change, unchanged modified row, or receipt-profile substitution.

C3B stays frozen to exactly three paths at `sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67`; its `RECEIPT_RECONCILIATION` profile and seven labels/types do not change. C3U changes no verifier roster.

## Repair contract

`insertLegacyReceiptBlockBeforeOptionalTerminalC3` is the one fail-closed composition helper used by the synthetic C0, C1, C2, and C3P receipt renderers. It validates the requested mode before inspecting the document. A marker-free handoff retains the historical direct or trimmed-blank byte behavior. If one already-canonical exact terminal C3 block exists, the helper captures its complete bytes, inserts the requested legacy receipt immediately before it, and reattaches the C3 authority byte-for-byte without parsing or regenerating its payload.

The helper refuses a partial marker pair, reversed markers, duplicated blocks, inline markers, comment- or fence-hidden authority, a nonterminal block, trailing content, non-LF C3 marker/terminal framing, or an unsupported mode. Exact visible Markdown topology remains load-bearing. The actual C0, C1, C2, and C3P fixture callers each run as a full-plan terminal-C3 control: stripping the inserted legacy receipt must recover that caller's exact prior handoff and preserve the exact terminal C3 block byte-for-byte. All four callers retain their prior hostile cases.

The current active-unit corpus has 44 exact Markdown paths at sorted-newline digest `sha256:1c5bf120f1ab948586bfa594e6ae634fc6acf7a5544febcccb2c4dc475a59a49`. Its declared matrix is 2 phases × 44 paths × 15 positive forms: 1,320 required rejections, plus 440 core accepted pending, modal, future, or failed-development controls. Eight supplemental natural aliases must reject, while two explicit strict-zero prohibition controls must remain accepted. These are bounded positive-receipt-form checks, not general Markdown understanding or a security audit.

## Fixed predecessor authority

C3U binds sealed C3T as the exact current parent: commit `9d8430aa70b1536a85dd7352a3d339667f2c1e88`, tree `3be4d7cde4d043b87438506483ffb8c256719df2`, direct parent `818ed90ebbf99ae38916307d9f2342921d38ebbf`, subject `fix: make C3 receipt self-test phase-stable`, note blob `d9e371310b360b183d890ed401cb36bd877becda`, note-body SHA-256 `05914bf85ce1d3bc48f5debf902babb526faf2500af318c35f08ffdf071ff735`, exact seven-path source diff, and nine ordered claims with complete event coverage. C3U pre-seal validation requires that identity to remain current `HEAD`. The later C3B preseal chain gate must resolve one exact sealed C3U child as current `HEAD`; after C3B commits, ordinary plan coherence accepts the exact sealed descendant ancestry.

## Intended claim map

| Claim label | Claim type | Intended grade |
| --- | --- | --- |
| `P07B-C C3U cross-phase receipt-fixture plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3U legacy receipt fixture defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3U sealed-C3T Git-note and ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3U unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3U cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3U cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3U exact seven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3U scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3U sealed-C3T predecessor and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Final command order

The final ledger must execute these command tails in this exact order. Each tail is prefixed by the checker-owned hermetic `/usr/bin/env -i` argv rooted at `.countershape/p07bc-c3u-final`, launched through `/opt/homebrew/bin/didrun run --`, and immediately followed by only its matching declared claim.

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs`
2. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c3u-sealed-c3t-note`
4. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
5. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
6. `/opt/homebrew/bin/node tools/verify-current.mjs`
7. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C3U --source-final-gate`
8. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C3U --credential-scan`
9. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c3u-preseal-ledger`

The first six commands are `tests-pass`; the final three are `command-succeeded`. Event nine may bind the exact sealed C3T predecessor, the preceding eight live events, one exact staged tree, and an absent seal. Because didrun appends an event only after the command exits, it cannot prove its own outer wrapper.

## Nonclaims

C3U changes no product source, target, store, Git materialization, runtime behavior, process edge, verifier roster, cache or parallelism setting, sensitive-package classification, repetition semantics, lock semantics, didrun installation, sealed C3/C3R/C3Q/C3T bytes, or frozen C3B contract. It proves no C3B source receipt, process start, runtime freshness, executable authenticity, secret absence, general security property, C4 authority, production readiness, adoption, or maintainership. All such outcomes remain outside this maintenance boundary and `UNRECEIPTED`.
