# P07B-C C5V — verifier-preflight hygiene

**State:** active pre-seal `SOURCE_FULL` defect repair. Every C5V grade is `UNRECEIPTED` until the exact staged tree completes its own twelve-event didrun ledger, commit, seal, readable Git note, strict verification, HTML report, and ledger archive.

## Boundary and authority

C5V is the owner-authorized child of exact sealed C5 and the required parent of C6A. Its exact commit subject is `fix: harden final verification hygiene`. Its classification is `OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR`; transition source is `OWNER_OUT_OF_BAND`; `predecessor_declaration.kind` is `NONE`; provenance is `UNEVIDENCED` with disclosure `OWNER_ATTRIBUTED_SESSION_INSTRUCTION_ONLY_NO_QUALIFYING_PREEXISTING_ARTIFACT`; authentication is `NOT_ESTABLISHED`; and signed authorization is `NOT_IMPLEMENTED`. The owner instruction was real, but no qualifying instruction artifact pre-existed in the direct-parent tree. This candidate-owned status cannot authenticate or retroactively evidence that instruction.

The current machine authority is `countershape/p07b-c-unit-paths/v26`, with 40 ordered unit rows and 28 receipt-phase rows at receipt-phase authority digest `sha256:68ab9eebe4b471eaab828f5d89f270478d1bca434b2ca4ac737641dcf15709a2`.

C5V is a nine-path SOURCE_FULL unit:

1. `docs/HANDOFF_MODE_C.md`
2. `docs/PROMPT_PACK.md`
3. `docs/VERIFICATION.md`
4. `docs/status/P07B-C-C5V-VERIFIER-PREFLIGHT-HYGIENE.md`
5. `spec/verification/p07b-c-unit-paths.json`
6. `tools/check-p07b-c-plan.mjs`
7. `tools/check-p07b-c-unit-scope.mjs`
8. `tools/verify-current-selftest.mjs`
9. `tools/verify-current.mjs`

The sorted-newline roster digest is `sha256:42d2961220c8a2d06d70fe5fa9627f220569a74b3b6d2df9a26858f770481031`. There are no prefix-owned paths. C5V changes generic verifier, checker, specification, and operator-documentation bytes, so the dormant `NON_PRODUCT_MAINTENANCE` profile is inapplicable; three full cumulative passes remain mandatory.

## Exact sealed parent

The direct parent is sealed C5:

- commit `240060f018d5b0e86914bd27361c5899ac9ba1c0`
- tree `4906ff4af407fa4e48cbf569f7e456694660ca06`
- direct parent C4L `32911730afb7cc717b8e0a164b9ea0038d83f3fa`
- subject `feat: complete standalone contract execution`
- didrun note blob `2fc791cfd4c7c6279f9691d97e472b859ee21fdd`
- note-body SHA-256 `de453a2780bbc7b80fe1091d029cd18d602529bc4f49027da1f7a8667ceb7d0d`
- `79/79` ordered grades, each `TREE-EXACT`
- `secrets_override: true`

C5 passed strict verification twice. Its exact-commit HTML report is `.countershape/evidence/p07b-c-c5-final-240060f018d5.html`, SHA-256 `ba4f8d9f6da9b382617c463b7ab24794f9137770efe7c3fd7c599993b5315068`; its sealed final ledger is `.didrun-history/p07b-c-c5-final-240060f018d5/.didrun/`. C5V reopens those local chain facts. It does not cryptographically authenticate the owner, issue a new C5 product grade, or substitute for C5's sealed product evidence.

## Defects and live S6 evidence

Two independent late-run failures made the generated operator path weaker than the verifier it launches:

- C5 final attempt 3 reached event 15 before a shutdown-left `.countershape/verify-current/active.lock` caused the verifier to refuse. The failed ledger remains `.didrun-history/p07b-c-c5-final-attempt-3-event15-failed/.didrun/`; the reviewed one-file recovery evidence remains `.didrun-history/p07b-c-c5-attempt3-reviewed-stale-lock-recovery-pid88658/.didrun/`. The runtime O_EXCL lock behaved correctly. The generator defect was failing to expose the residue before event 1.
- C5 final attempt 4 let cumulative pass two return green after a `.DS_Store` appeared later than that pass's opening check; pass three then refused immediately. The failed ledger remains `.didrun-history/p07b-c-c5-final-attempt-4-event76-failed/.didrun/`, with recovery evidence at `.didrun-history/p07b-c-c5-attempt4-finder-artifact-recovery/.didrun/`. The verifier needed a terminal workspace recheck, and the generated runbook needed the same Finder-artifact preflight before starting an atomic ledger.

These are verification-hygiene defects, not product-runtime defects.

## Repair

Every generated preflight stage first requires any visible repository-root `.countershape/verify-current` parent to be a canonical, current-uid-owned, exact-mode-`0700` real directory and requires its `active.lock` to be absent by no-follow `lstat`; only `ENOENT` passes that absence check. Preparation additionally requires repository-root `.didrun` and the boundary's exact projected final root to be absent before any evidence directory, final root, didrun event, claim, commit, seal, or archive mutation. Archive preflight instead requires repository-root `.didrun` and every boundary final directory to exist as canonical, current-uid-owned, exact-mode-`0700` real directories, then requires only the exact commit-addressed archive root to be absent before the move. A regular file, malformed object, directory where absence is required, symlink, dangling symlink, wrong owner/mode, or traversable intermediate alias refuses. The generator never rotates or deletes residue.

The same preflight scans the repository recursively for `.DS_Store` entries while ignoring only the root-level private trees `.git`, `.didrun`, `.didrun-history`, `.countershape`, and `node_modules`. It detects the basename regardless of file type, uses no-follow metadata for traversal, and never follows a directory symlink. A nested directory merely named like an ignored root is still scanned.

The cumulative verifier adds `workspace-no-ds-store-terminal` immediately before authority revalidation and resource finalization. The opening and terminal rows use the same bounded no-follow scan. Self-tests pin exactly two rows, their order and placement, root and nested regular entries, a directory, a dangling symlink, all five root-only ignore controls, a symlink-directory no-follow control, nested ignored-name refusal, and mutation after the opening observation.

## Operator response

If preflight reports `active.lock`, stop; do not rerun the ledger. Inspect the no-follow object type first. A valid regular lock records a PID that can be checked against process liveness and the current file holder; a malformed regular file, directory, symlink, or dangling symlink has no trusted recorded PID. Only after confirming that no live verifier owns the reviewed object may the operator remove that one `active.lock`. Preserve every `run-*` directory, failed ledger, and archive.

If preflight reports `.DS_Store`, stop; do not rerun the ledger. Remove the visible Finder artifact named by the diagnostic, not ignored/private evidence, then start a fresh ledger. Never weaken the scan or add a newly discovered path to the ignore set merely to obtain green evidence.

Every absence observation is point-in-time under the documented sole-writer discipline. It is not hostile same-UID isolation and cannot prevent another local actor from mutating and restoring paths between observations.

## C5V final manifest

| # | Claim | Type | Intended grade |
|---:|---|---|---|
| 1 | `P07B-C C5V candidate phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| 2 | `P07B-C C5V independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| 3 | `P07B-C C5V plan checker defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 4 | `P07B-C C5V sealed-C5 Git-note and lower ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| 5 | `P07B-C C5V unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 6 | `P07B-C C5V cumulative verifier defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 7 | `P07B-C C5V cumulative verification pass 1` | `tests-pass` | `UNRECEIPTED` |
| 8 | `P07B-C C5V cumulative verification pass 2` | `tests-pass` | `UNRECEIPTED` |
| 9 | `P07B-C C5V cumulative verification pass 3` | `tests-pass` | `UNRECEIPTED` |
| 10 | `P07B-C C5V exact nine-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| 11 | `P07B-C C5V scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| 12 | `P07B-C C5V sealed-C5 predecessor and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Exact final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C5V`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C5V`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
4. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c5v-sealed-c5-note`
5. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
6. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
7. `/opt/homebrew/bin/node tools/verify-current.mjs`
8. `/opt/homebrew/bin/node tools/verify-current.mjs`
9. `/opt/homebrew/bin/node tools/verify-current.mjs`
10. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C5V --source-final-gate`
11. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C5V --credential-scan`
12. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c5v-preseal-ledger`

Run each command once in the generated C5V runbook and claim it immediately; any nonzero command stops that ledger, supports no claim, and requires a repaired fresh boundary.

The first six prove phase, exact-parent, checker, and hygiene mechanisms. The three cumulative passes prove the complete current tree under the unchanged serialized verifier policy. The last three prove exact staged ownership, structured credential-pattern absence over those nine blobs, and final ledger/ancestry coherence.

## Proposal routing

- Persistent Go-only build cache: `DECLINED_WITH_REASON`; the admitted authority does not bind the full cgo, SDK, and imported-library surface needed to call cross-run reuse hermetic.
- Higher general Go parallelism: `DECLINED_WITH_REASON`; the load-sensitive Darwin recurrence remains stronger evidence than the throughput projection. Global `p=1`, `GOMAXPROCS=2`, and test `-parallel=2` remain intentional.
- Exclusive single-verifier lock: `INTENTIONAL / ALREADY SATISFIED`; C5V repairs fail-early operator visibility, not runtime lock semantics.
- Docs-only receipt profile: `INTENTIONAL / ALREADY SATISFIED` as `RECEIPT_RECONCILIATION`. The newer `NON_PRODUCT_MAINTENANCE` grammar stays dormant because no safe concrete consumer exists and it cannot cover generic verifier/checker edits.
- Receipt-phase fixture consolidation: `DEFECT_ALREADY_REPAIRED` by sealed C3D/C4M/C4N; C5V adds a table row and generic hostiles rather than another hand-modeled lifecycle.
- Proof-chain splitting: `DECLINED_WITH_REASON`; sealed C5 successfully completed its exact monolithic 79-event contract, and no legal claim transition authorizes witness substitution.
- Owner instruction provenance: `PARTIALLY SATISFIED`; the closed vocabulary and citation practice exist, but this out-of-band instruction remains honestly `UNEVIDENCED`. Citation would not be authentication.

## Claim ceiling

C5V may claim fail-closed preflight behavior, no-follow artifact scanning, exact row placement, exact sealed-C5 chain validation, staged-scope integrity, and full current-tree verification. It does not change C5 product behavior, cache or parallelism policy, lock acquisition/release semantics, product deadlines, external APIs, proof-chain atomicity, or historical grades. It does not prove automatic stale-lock recovery, arbitrary filesystem stability, hostile same-UID resistance, security certification, production readiness, adoption, or maintainership.
