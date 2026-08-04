# P07B-C C6N — sealed C6M note-redaction maintenance

**State:** active pre-seal `SOURCE_FULL` defect repair. Every C6N grade is `UNRECEIPTED` until its exact staged tree completes its own ten-event didrun ledger, commit, seal, readable Git note, strict verification, HTML report, and ledger archive.

## Boundary and authority

C6N is the owner-authorized child of exact sealed C6M and the required parent of C6B. It repairs a representation mismatch between the sealed C6M note bytes and the future consumer contract, and it repairs a final-ledger stability defect in the descendant sealed-C6A replay runner. It changes no Countershape product behavior, production Go source, cumulative-verifier row roster, sealed history, or C6A source-authority manifest.

- **Commit subject:** `fix: bind C6M note redaction epoch`
- **Direct parent:** exact sealed C6M commit `b0de83dca3510102af4d1cddae2b2023bc88a87d`, tree `3d80a02443209f5d5649fcba92b79b72a2a04a15`
- **Parent note:** blob `d08739e2f5bb3d761d5962081242924ae9cb7f36`, body SHA-256 `1bcf1b4563561d6a07db9394bb6b25b2e7d2ca126df18b98f0c8596cc960a487`
- **Sealed C6G ancestor:** commit `3677b45d75204ce2ef7242abb5edd11cc049934e`, tree `520ad2cbf2d3da8354b02afd1cd2a7cd5df82d7d`
- **Sealed C6F ancestor:** commit `dd2a011edff20d65c43934fc7c3e5a09b5829d2e`, tree `b0141ac2671df75e51c4fdb5b71c35cee5d0f4ea`
- **Sealed C6A ancestor:** commit `3e9643f657248e0d5ff2c4bf0880218efbaeecd8`, tree `226a1e23da7eded933b3faabd4e8040576b2a2e0`
- **Transition source:** `OWNER_OUT_OF_BAND`
- **Classification:** `OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR`
- **Predecessor declaration:** `NONE`
- **Provenance:** `UNEVIDENCED`, disclosed as `OWNER_ATTRIBUTED_SESSION_INSTRUCTION_ONLY_NO_QUALIFYING_PREEXISTING_ARTIFACT`
- **Authentication:** `NOT_ESTABLISHED`
- **Signed authorization:** `NOT_IMPLEMENTED`

The owner instruction was real, but no qualifying instruction artifact pre-existed in the direct-parent tree. This candidate-owned status cannot authenticate or retroactively evidence it. Receipt C6A remains `ABSENT`; the C6A source-authority manifest is inherited byte-identically, no receipt declaration or visible receipt projection is introduced, and no prior grade transfers into C6N.

The current machine authority is `countershape/p07b-c-unit-paths/v29`, with 43 ordered unit rows and 31 receipt-phase rows at receipt-phase authority digest `sha256:30ac9d08842915a5f3dfe5aca79c7fa94b02add8dc9785226ca1741411b66bb5`.

## Observed defect and repair

The sealed C6M Git note is valid and immutable. Its ten command previews preserve exact tails and bounded positional redaction, but the GOCACHE position uses the two-marker form actually emitted by that didrun epoch. The prospective consumer expected only the later marker-plus-path-suffix form and therefore rejected the exact parent note before C6B could begin. Rewriting or resealing C6M would erase rather than repair the historical boundary.

C6N makes the representation epoch an explicit input to both independent validators. C6A, C6F, and C6G remain pinned to `SUFFIX_PRESERVING`; exact sealed C6M and prospective C6N use `LEGACY_DOUBLE`. Raw exact previews remain valid. Neither redacted form is accepted generically: the policy is selected by the exact boundary contract, and the C6M form additionally requires its pinned commit, tree, parent, subject, note blob, and note-body digest. Cross-epoch hostile cases prove that substituting the other form is refused. Redacted positions establish only the admitted visible structure, not the hidden bytes.

The independent contract digests are `sha256:a7f3696229c63afe11911980d7b21873b21d5c06e50207f8d0883b2379e707ea` for exact sealed C6M and `sha256:500955d69b63da43516312788befa6642ecbdcb6e7777ac014f5beeeb5beb41b` for prospective C6N. Synthetic note parity is a checker contract, not evidence that the future C6N note exists.

## Final-ledger sealed replay stability repair

C6N final attempt 1 never reached product or checker semantics: the restarted Codex sandbox denied `.git/index.lock` creation during command 1. Its single unclaimed event is retained at `.didrun-history/p07b-c-c6n-final-attempt-1-sandbox-git-lock/.didrun/`. Final attempt 2 passed and immediately claimed commands 1–6, then its unclaimed cumulative command failed at row 57 when one of 567 per-blob `git hash-object --stdin` subprocesses reached the runner's unchanged 60-second deadline. That seven-event ledger is retained at `.didrun-history/p07b-c-c6n-final-attempt-2-c6-hash-object-timeout/.didrun/`. Neither attempt produced a commit, seal, note, strict result, HTML report, or reusable C6N grade.

The observed runner launched 567 `cat-file` processes and 567 `hash-object` processes for each sealed row. C6N replaces that fan-out with exactly one bounded `git cat-file --batch` extraction and one bounded `git hash-object --stdin-paths --no-filters` parity pass. The extraction parser requires one exact ordered `<oid> blob <size> LF bytes LF` frame per manifest row, enforces the existing entry/blob/total ceilings, rejects missing, reordered, malformed, truncated, oversized, or trailing data, and recomputes every Git blob SHA-1 in process. Every hash-batch entry must carry components that exactly reconstruct its already-safe manifest path; absolute, traversal, `.git`, control-bearing, normalization-unstable, duplicate, case-colliding, and directory/file-colliding entries are rejected before a path is built. The hash request uses canonical absolute snapshot paths, has an 8 MiB input ceiling, and accepts only exactly one lowercase 40-hex-plus-LF result per ordered entry. `--no-filters` is load-bearing because the sealed snapshot includes `.gitattributes`. The snapshot is inspected before Git parity, after parity, and after the frozen child. Git still runs once per batch with the same 60-second deadline; no retry or deadline increase is added.

The runner also adopts the already sealed C6F three-form Apple diagnostic classifier for its own status-zero Git calls: empty stderr or at most 16 KiB of fatal-UTF-8, final-LF, no-CR lines matching only the pinned `DARWIN_USER_TEMP_DIR`, FSEvents, or `DARWIN_USER_CACHE_DIR` forms. BOMs, foreign or malformed lines, missing LF, CRLF, invalid UTF-8, over-bound bytes, nonzero exit, signal, spawn error, output drift, or batch framing drift remain fatal. Rejection reports project only a 256-byte hexadecimal stderr prefix plus total byte count and a truncation bit; they never decode the potentially 256 MiB batch stderr buffer. The built-in hostile self-test exercises accepted, nonzero, signaled, spawn-error, foreign-stderr, non-buffer, oversized-diagnostic, and path-contract outcomes, while the verifier self-test pins exactly one production call site for each Git batch. This is bounded host-noise classification, not general stderr suppression.

The unclaimed thirteen-event repair ledger is preserved at `.didrun-history/p07b-c-c6n-batch-repair-diagnostic-2dd3ee57e772/.didrun/`. Its deliberate development reds caught noncanonical tool input, sandbox-induced Apple diagnostics inside the frozen child, an undersized Node `maxBuffer`, and relative-path resolution against the parent worktree. On exact repair tree `2dd3ee57e772`, the built-in hostile self-test passed; the real `--c6` row then passed in 10,133 ms and the real `--c6-selftest` row passed in 13,451 ms outside the app sandbox under the standing owner authorization. Those events validate the focused implementation but remain unclaimed development evidence; they do not replace C6N's fresh ten-command final ledger.

## Plan self-test throughput repair

The first complete cumulative restart after the power loss is permanent red development evidence at `.didrun-history/p07b-c-c6n-development-red-plan-selftest-timeout-0f2ce2b787a9/.didrun/`. The interrupted power-loss process itself produced no terminal didrun event and is not a result. The complete restart retained 39 events on tree `0f2ce2b787a92c849a38d0f76b3cc2c6b64cad10`: rows 1–58 passed, then unclaimed event 38 exited `1` when row 59, `architecture-p07b-c-plan-selftest`, reached the unchanged `1,200,000` ms outer child bound after `1,200,016` ms and reported `spawnSync ... ETIMEDOUT`. Its C3P and C3 completion markers had printed, the evolved-plan completion marker had not, and rows 60–65 plus all historical rows were not run. The event supports no C6N claim, seal, grade, or partial cumulative result.

The repair preserves every checker case, diagnostic contract, boundary, receipt state, claim, verifier row, and deadline. After one ordinary uncached baseline passes, the self-test captures an invocation-scoped authority snapshot by performing the real Git/note loads, then reuses an authority only when exact canonical root, active boundary, receipt bytes, C6 manifest bytes, Git and Node executable identities, `HEAD`, index tree, and notes tree still match. The epoch additionally requires configured Git to resolve to the `/usr/bin/git` used by sealed-C6 derivation. Explicit fixture injections always win; a strict-index snapshot, byte mismatch, alternate root, alternate boundary, absent input, or read error bypasses reuse and follows the existing validation path. The complete authority snapshot is independently recaptured and compared in a terminal finalizer even when the test body fails. Mutable loaded objects are cloned and recursively frozen before reuse.

The three qualification modules and exact verifier-entry fixture are parsed once per exact ordered byte set and exact Node executable identity inside that same invocation. The cache verifies full path-to-source equality after its typed digest lookup and stores successful analyses only; parse or closure failures are never retained. The epoch wrapper is intentionally limited to the current C6M/C6N/C6B horizon where all five historical receipts and the C6 manifest exist; other phases retain the ordinary uncached path. There is no timeout increase, retry, case removal, case sharding, semantic relaxation, claim change, receipt change, or verifier-row change. This is count-preserving elimination of repeated immutable authority work, not substitution of inherited proof.

The first repaired diagnostic is preserved at `.didrun-history/p07b-c-c6n-plan-throughput-green-26a91ed4bb6c/.didrun/`. On exact staged tree `26a91ed4bb6ca1a13b75d7934975f7e6e75a4859`, events 0–1 passed staged-diff and syntax checks, and event 2 ran the complete plan self-test in `164.872573` seconds with exit `0`, empty stderr, and stdout object `sha256:816de4d8a7b18bca21169624b22662a3cd515f31267ebb49e0cd42c7f0d64e5a`. Its exact public markers remain C3P `1926/968`, C3 `205` plus two inert decoys, and evolved aggregate `25225`. This is an approximately 86.3% wall-clock reduction from the unchanged 1,200-second ceiling that the prior attempt exhausted. Because recording this diagnostic changes the candidate documentation tree, the ledger is deliberately unclaimed and cannot substitute for the repaired final-tree self-test or cumulative verifier.

The next seven-event ledger is permanent red at `.didrun-history/p07b-c-c6n-development-red-bare-sealed-c6-tool-authority-c804f6ed8a00/.didrun/`. On staged tree `c804f6ed8a003a097499b3ed6295a9187fc39e45`, events 0–5 passed final-tree diff, syntax, complete plan self-test, scope self-test, exact sealed-C6M note/ancestry, and verifier self-test gates. Event 6 then exited `1` after `0.073660` seconds because a manually selected bare `tools/check-sealed-c6a-architecture.mjs --c6` invocation omitted the cumulative verifier's required hermetic `COUNTERSHAPE_NODE` and tool/root environment, producing exact code `SEALED_C6A_TOOL_AUTHORITY`. This is an operator command-selection defect, not evidence against the runner or product, but the whole ledger remains unclaimed and no green event is reusable. The correction is to let the unchanged cumulative verifier construct and exercise both sealed-C6A rows under its already qualified environment; the gate is not weakened and the environment is not approximated by hand.

## Scope

C6N owns exactly these nine mode-`100644` paths:

1. `docs/HANDOFF_MODE_C.md`
2. `docs/PROMPT_PACK.md`
3. `docs/VERIFICATION.md`
4. `docs/status/P07B-C-C6N-NOTE-REDACTION-MAINTENANCE.md`
5. `spec/verification/p07b-c-unit-paths.json`
6. `tools/check-p07b-c-plan.mjs`
7. `tools/check-p07b-c-unit-scope.mjs`
8. `tools/check-sealed-c6a-architecture.mjs`
9. `tools/verify-current-selftest.mjs`

The sorted-newline roster digest is `sha256:012c7559efd28dce5dac7fb56881bbddd8f326f89eac4d32a6773ad8cfe856af`.

Prefixes are empty. The C6A manifest, receipt declaration, C6 evidence, cumulative verifier entrypoint and row roster, product/runtime source, frozen C6 architecture child, and all sealed notes and commits are outside this roster and remain byte-stable.

## C6N final manifest

| # | Claim | Type | Intended grade |
|---:|---|---|---|
| 1 | `P07B-C C6N data-driven phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| 2 | `P07B-C C6N independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| 3 | `P07B-C C6N note-redaction defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 4 | `P07B-C C6N sealed-C6M Git-note and sealed-C6G, sealed-C6F, and sealed-C6A ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| 5 | `P07B-C C6N unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 6 | `P07B-C C6N cumulative verifier and sealed-C6A replay runner self-test` | `tests-pass` | `UNRECEIPTED` |
| 7 | `P07B-C C6N cumulative verification with sealed-C6A replay` | `tests-pass` | `UNRECEIPTED` |
| 8 | `P07B-C C6N exact nine-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| 9 | `P07B-C C6N scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| 10 | `P07B-C C6N sealed-C6M predecessor, sealed-C6G, sealed-C6F, and sealed-C6A ancestry, and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Exact final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C6N`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C6N`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
4. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c6n-sealed-c6m-note`
5. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
6. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
7. `/opt/homebrew/bin/node tools/verify-current.mjs`
8. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C6N --source-final-gate`
9. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C6N --credential-scan`
10. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c6n-preseal-ledger`

Run each command once in the generated C6N runbook and claim it immediately; any nonzero command stops that ledger, supports no claim, and requires a repaired fresh boundary.

The first seven prove the phase transition, the exact boundary-scoped note-redaction epoch and its defensive oracle, exact sealed-C6M/C6G/C6F/C6A ancestry, both generic checker self-tests, the bounded-batch sealed replay runner, and the unchanged cumulative verifier roster. The last three prove exact staged ownership, structured credential-pattern absence over the nine blobs, and final ledger/ancestry coherence.

## Claim ceiling and continuation

C6N may claim only the bounded note-representation compatibility repair, bounded-batch descendant replay stability under the admitted toolchain, exact parent and ancestor authority, current checker/verifier coherence, exact staged scope, and exact didrun chain integrity after its own receipt exists. It does not claim hidden-value recovery, product change, C6A receipt, renewal of any sealed ancestor, portable host stability, hostile same-UID isolation, secret absence, confidentiality, security certification, production readiness, adoption, or maintainership.

The next permitted edge is C6B only after this boundary independently closes. C6B must inherit the C6A source-authority manifest byte-for-byte, add only its predeclared receipt surfaces, and validate exact `C6B → C6N → C6M → C6G → C6F → C6A` ancestry. No grade crosses that edge automatically.

## Current development state — 2026-08-03

Exact sealed C6M remains `HEAD` at commit `b0de83dca3510102af4d1cddae2b2023bc88a87d`, tree `3d80a02443209f5d5649fcba92b79b72a2a04a15`. C6N remains uncommitted, unsealed, and `UNRECEIPTED`. Its earlier complete development battery is preserved at `.didrun-history/p07b-c-c6n-development-green-972ba020bd0d/.didrun/`, but the expanded nine-path repair invalidates that tree as current proof. The two failed final attempts, focused batch-repair diagnostic, and 39-event plan-self-test timeout ledger remain permanent history as described above; none is reusable on the repaired tree.

The next valid development boundary must stage exactly the nine-path roster, rerun the repaired plan self-test, both independent checker self-tests, the exact sealed-C6M note validator, the strengthened verifier self-test, both focused real sealed-C6A rows, and the complete cumulative verifier through didrun on one final source tree. Preserve that development ledger before creating a fresh canonical C6N final root. No event or claim from either failed final attempt or the red `0f2ce2b787a9` development ledger may be reused; the final ledger must restart at command 1.
