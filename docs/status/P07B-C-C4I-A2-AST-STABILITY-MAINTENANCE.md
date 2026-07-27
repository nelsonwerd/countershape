# P07B-C C4I — verification-surface stability maintenance

Classification: `OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR`

C4I owns exactly these 14 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/THREAT_MODEL.md`, `docs/VERIFICATION.md`, `docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md`, `docs/status/P07B-C-C4I-A2-AST-STABILITY-MAINTENANCE.md`, `internal/world/process_darwin_test.go`, `spec/verification/p07b-c-unit-paths.json`, `testkit/processfixture/main.go`, `tools/capture-p07b-a2-human-surface.mjs`, `tools/check-p07b-a2-architecture-selftest.mjs`, `tools/check-p07b-a2-architecture.mjs`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Their sorted-newline roster digest is `sha256:522af4209bde3f769a12c3943f46bd83e566977ad8f275165310962c1e697585`.

- **Boundary:** active pre-seal `SOURCE_FULL` maintenance after exact sealed C4H and before the unchanged C5 product boundary. Classification: `OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR`.
- **Authority:** `OWNER_OUT_OF_BAND`; the owner authorized maintenance when a blocking verifier defect surfaced, but sealed C4H did not predeclare C4I. `predecessor-predeclared = false`; C4I inherits or rewrites no C4H grade, and the owner instruction is not cryptographically authenticated.
- **Parent:** sealed C4H commit `4b4686ea30e5ddc5043eee11ba24725bcce8bb94`, tree `2d02d220961f3fc41b2372df5eb8dc3aac47a529`, note blob `9532c545dd685046efd57e999613092f92e54e8d`, note-body SHA-256 `e94c4b8b48978f0208757f7f7707bde5c02b0cb7c91eebaa16873e9aaf1b5041`, subject `fix: harden verifier execution boundary`, `11/11 claims recorded-exact`, every grade `TREE-EXACT`, and strict exit `0`.
- **Commit subject:** `fix: stabilize verification surfaces`
- **Profile:** `SOURCE_FULL`; this unit edits inherited A2 verification surfaces, a Darwin world test and its process fixture, plus phase/checker authority.
- **Receipts:** C3P `PRESENT`; C3 `PRESENT`; C6A `ABSENT`.
- **Product authority:** none. C4I changes Go test and test-fixture bytes only; it changes no production Go implementation or generated Node product bytes, contract execution, target, permit, process, store, classification, runtime-authority helper, repetition helper, didrun installation, package partition, cache policy, parallelism, production or verifier timeout, or cumulative-row order.

Every intended C4I grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C4I itself.

The machine authority is `countershape/p07b-c-unit-paths/v22` with 24 ordered phase rows and authority SHA-256 `3176e9a6764203afeed0ae836bd1a4974ae63887dbee8f0c1de1f66f2c93f580`. The plan transition matrix has 12 accepted and 340 rejected cases; the independently implemented unit-scope matrix has 12 accepted and 327 rejected cases. The current Markdown corpus has 60 paths at `sha256:7bb2a0c222b4f568143c4bba9e3d99c8841ee0483fefc00ea3117d753be87dbc`, with 930 positive-form rejections and 310 controls.

## Blocking recurrence

The first repository-wide C5 development verifier passed rows `1–22/61`, including the `699103 ms` general-package row and `1135259 ms` twelve-package sensitive row, then exited `1` at inherited `architecture-p07b-a2-2-selftest` after `245811 ms`. The `example-generator-byte-drift` control appended one comment to a `4799`-byte profile. The checker had already accumulated its required `P07B_A2_EXAMPLE_GENERATOR_DIGEST` violation, but any one profile drift caused all six JavaScript profiles to enter one nested AST worker. That request included the unchanged `147788`-byte harness and prevented the known digest violation from reaching the self-test when the worker hit its fixed 60-second deadline.

This is the recurrence under `p=1` anticipated by sealed C4V. Earlier permanent reds reached the same inner deadline on different valid hostiles; later unchanged runs passed. A didrun-wrapped read-only benchmark on the C5 failure tree measured the exact all-profile worker at `118 ms` over `218577` input bytes and `462688` output bytes. The observed `SIGTERM` is therefore nondeterministic load sensitivity, not evidence that one normally valid AST computation needs a longer deadline. The failed C5 verifier and the benchmark harness's first diagnostic typo remain permanent unclaimed history; the corrected benchmark is development evidence only.

## Final-gate mode defect

The first C4I final attempt failed before any claim because the managed sandbox made Apple `/usr/bin/git` emit developer-tool diagnostics despite exit `0`; exact unchanged host execution later proved the same stripped command byte-clean. The second final attempt then recorded and claimed nine green events before its first cumulative verifier exited `1`. The second final attempt failed deterministically at `human-surface-p07b-a2-2-check`: `writeFileSync(..., { mode: 0o644 })` was reduced by caller `umask 077` to actual `0600`, after which the helper correctly rejected expected `0644` mode drift.

The third final attempt recorded and claimed events `0–10`, including one complete cumulative pass, then its second cumulative verifier failed at event `11`; that red event was not claimed. `TestProcessGroupAndSessionEscapesRemainExplicitExclusions/setpgid` started a nested child inside the owned group, but the fixed `100 ms` execution timer could trigger original-group teardown before the child ran `Setpgid` and published `escaped-group.pid`. The missing file proved that the test had not causally established its escape precondition; it did not prove a product cleanup defect. The permanent attempt lives at `.didrun-history/p07b-c-c4i-final-attempt-3-world-setpgid-pid-publication/.didrun/` with private root `.countershape/p07bc-c4i-final-attempt-3-world-setpgid-pid-publication`. All three failed attempts remain history; none produced a commit, seal, note, strict result, HTML, or reusable grade.

Accepting `0600`, weakening the assertion, removing `umask 077`, or skipping the row would contradict the exact `100644` ContractBundle and hide the defect. C4I instead expands its still-unsealed authority by one already-pinned A2 profile. `writeExactFile` retains exclusive `wx` creation, normalizes only the successfully created path to exact `0644`, then validates complete permission bits, regular/non-symlink type, link count, and bytes. Its synchronous self-test exercises caller umasks `077` and `022`; refuses and preserves preexisting regular files, final symlinks, and directories; restores the original caller mask through success and induced collision failure; and removes partial private roots.

## Repair and proof semantics

The checker still reads and hashes every exact JavaScript profile. Exact bytes still skip AST work because the reviewed SHA-256 pins already bind the complete source. When one or more profiles drift, only those drifted entries—the exact drifted profiles—enter the nested AST worker. Generic import, binding, dynamic-edge, and export comparisons consume only returned drifted shapes. Contract-entrypoint, harness/spawn/evaluator, parity-runner, and human-capture checks run only when their owning profile drifted. Every existing semantic hostile remains unchanged and must still produce its existing exact code.

The self-test now binds the drift-only dataflow and forbids the former all-profile call. The frozen 117/117 hostile roster remains pinned at `sha256:832d98ce1e73056e3133a9b6cc31d1ef3da4c88a451f85f122d35ab38b05c80e`; a separate 32/32 mutant-to-profile authority is pinned at `sha256:16bf1f1a46090abc88d87d27726aa182c0cbe3977a52042db1c829bb58a85442`. Clean and non-JavaScript cases require zero AST rows; every relevant mutant requires its exact singleton row; one three-profile hostile requires exact ordered AST rows; and two worker-infrastructure controls prove both synthetic precedence and end-to-end invalid-JavaScript refusal. `P07B_A2_JS_AST_QUERY_FAILED` is classified explicitly as an infrastructure failure before expected-hostile matching, so a timed-out or malformed worker can never be accepted merely because the checker had already accumulated the expected digest string.

C4I initially preserves cumulative verifier order. Moving A2 ahead of the long Go lanes would remove the post-load composition that exposed the defect and could mask rather than repair it. The fixed 60/90-second deadline hierarchy, `p=1`, `GOMAXPROCS=2`, test `-parallel=2`, fresh private caches, single-verifier lock, package roster, assertions, and fail-fast sequential execution remain exact. If drift-only analysis still reaches the deadline during a post-load cumulative pass, C4I remains open; the next repair must retain a post-heavy all-profile worker-health sentinel or remove the nested-worker architecture rather than reorder away the signal, retry a failed row, lengthen deadlines, or weaken the hostile roster.

The mode repair runs entirely beneath fresh mode-`0700` fixture roots before subject execution, so exact `0644` does not make those files cross-user traversable. Path-based chmod is sufficient for this named synchronous test helper, but it does not establish hostile same-UID swap resistance, ancestor no-follow traversal, a general materializer, product publication, or private-store mode policy. Descriptor-bound materialization remains a separate product authority.

The Darwin fixture repair replaces elapsed-time readiness with a test-only hash-bound protocol: the child starts its bounded self-lease, then atomically publishes a canonical mode-`0600` `PREPARED` record while still contained; the group-owned callback registers cleanup authority and independently observes that exact child in the original group before publishing `AUTHORIZED`; the child then changes group or session, emits exact stream markers, and publishes `READY`; after the unchanged product timeout and teardown receipt, the test independently re-observes PGID and, for `setsid`, SID before publishing `RELEASE`; the child validates that exact hash join and publishes `RELEASED` before voluntary exit. Each record binds the exact predecessor digest and private attempt identity. Hard-link publication makes complete bytes visible without replacement; link count `2` is treated only as an in-progress publication state until the private temporary name is removed, while every other mode, type, framing, or link-count mismatch fails closed.

The product execution timer still starts before the group-owned callback and remains exactly `100 ms`; the callback neither resets nor lengthens it. The callback can delay when physical teardown begins while it establishes the test precondition, so this test proves timeout classification and cleanup evidence after causal readiness—not a wall-clock enforcement bound. A `15 s` fixture self-lease beginning before `PREPARED` visibility plus bounded reap grace avoids identity-unsafe numeric-PID fallback; missing `RELEASED` is a test failure even when the lease later removes the child. Darwin PID/PGID reuse remains explicitly outside the containment claim. The existing `world-lifecycle-readiness-20` authority exercises the coordinated escape test and the rest of the sensitive world lifecycle roster twenty times. Development greens, including a focused 20-repeat run, support iteration only and are not reusable final grades.

The repair is justified optimization plus exact fixture construction, not a guaranteed scheduler cure or a new security boundary. Only one direct caller-umask self-test, three standalone A2 passes, the exact `world-lifecycle-readiness-20` qualification, and three complete cumulative passes on one final staged tree can support the intended stability claims.

## Intended C4I claim map

| # | Claim | Type | Intended grade |
| ---: | --- | --- | --- |
| 1 | `P07B-C C4I candidate phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| 2 | `P07B-C C4I independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| 3 | `P07B-C C4I plan checker defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 4 | `P07B-C C4I sealed-C4H Git-note and C4 C4P C4N C4M C4V C3D ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| 5 | `P07B-C C4I unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 6 | `P07B-C C4I human-surface caller-umask hermeticity` | `tests-pass` | `UNRECEIPTED` |
| 7 | `P07B-C C4I A2 architecture stability pass 1` | `tests-pass` | `UNRECEIPTED` |
| 8 | `P07B-C C4I A2 architecture stability pass 2` | `tests-pass` | `UNRECEIPTED` |
| 9 | `P07B-C C4I A2 architecture stability pass 3` | `tests-pass` | `UNRECEIPTED` |
| 10 | `P07B-C C4I coordinated world escape and lifecycle readiness stability` | `tests-pass` | `UNRECEIPTED` |
| 11 | `P07B-C C4I cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| 12 | `P07B-C C4I cumulative verification pass 1` | `tests-pass` | `UNRECEIPTED` |
| 13 | `P07B-C C4I cumulative verification pass 2` | `tests-pass` | `UNRECEIPTED` |
| 14 | `P07B-C C4I cumulative verification pass 3` | `tests-pass` | `UNRECEIPTED` |
| 15 | `P07B-C C4I exact fourteen-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| 16 | `P07B-C C4I scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| 17 | `P07B-C C4I sealed-C4H predecessor C4 C4P C4N C4M C4V C3D ancestry and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Exact final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C4I`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C4I`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
4. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4i-sealed-c4h-note`
5. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
6. `/opt/homebrew/bin/node tools/capture-p07b-a2-human-surface.mjs --self-test`
7. `/opt/homebrew/bin/node tools/check-p07b-a2-architecture-selftest.mjs`
8. `/opt/homebrew/bin/node tools/check-p07b-a2-architecture-selftest.mjs`
9. `/opt/homebrew/bin/node tools/check-p07b-a2-architecture-selftest.mjs`
10. `/opt/homebrew/bin/node tools/verify-go-test-repetition.mjs --case world-lifecycle-readiness-20`
11. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
12. `/opt/homebrew/bin/node tools/verify-current.mjs`
13. `/opt/homebrew/bin/node tools/verify-current.mjs`
14. `/opt/homebrew/bin/node tools/verify-current.mjs`
15. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4I --source-final-gate`
16. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4I --credential-scan`
17. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4i-preseal-ledger`

The final root is `.countershape/p07bc-c4i-final`. The exact staged tree must exist before command 1. Every isolated shell fence generated for C4I sets `umask 077` before any didrun, claim, commit, seal, note, strict, HTML, or archive command; no persistent outer shell is assumed. Any nonzero command, commit/seal error, nonzero strict grade, or post-stage drift permanently fails that attempt; the failed ledger remains history and a repaired tree starts fresh evidence at command 1.

## C5 continuation

C5 remains exactly 12 explicit paths plus three required prefixes, with the same subject, 79 labels/types/argv/order, 56 isolated qualification cases, and three cumulative passes. Only its parent/ancestry expands to `C4I → C4H → C4 → C4P → C4N → C4M → C4V → C3D`. Its checkpoint is not a commit, seal, receipt, or grade. After C4I seals and verifies strictly, the checkpoint must be reapplied without dropping, overlapping authority files reconciled manually, and every C5 development/final gate rerun on the new tree.
