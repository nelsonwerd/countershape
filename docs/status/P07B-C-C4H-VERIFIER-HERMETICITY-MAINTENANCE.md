# P07B-C C4H — verifier hermeticity and operator-state maintenance

C4H owns exactly these 18 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C4H-VERIFIER-HERMETICITY-MAINTENANCE.md`, `research/deep-dive/p07b-c-maintenance-architecture/00-scope-and-method.md`, `research/deep-dive/p07b-c-maintenance-architecture/01-proof-semantics.md`, `research/deep-dive/p07b-c-maintenance-architecture/02-verifier-throughput.md`, `research/deep-dive/p07b-c-maintenance-architecture/03-maintenance-order.md`, `research/deep-dive/p07b-c-maintenance-architecture/04-synthesis.md`, `research/deep-dive/p07b-c-maintenance-architecture/05-follow-up-verification.md`, `research/deep-dive/p07b-c-maintenance-architecture/06-different-model-red-team.md`, `research/deep-dive/p07b-c-maintenance-architecture/07-executive-briefing.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, `tools/check-p07b-c-unit-scope.mjs`, `tools/generate-p07-planning-example.mjs`, `tools/verify-current-selftest.mjs`, and `tools/verify-current.mjs`. Their sorted-newline roster digest is `sha256:1adc913036078ad4efdf418e78581d81369aa630d245a5b493c60a69a0478532`.

- **Boundary:** active pre-seal `SOURCE_FULL` maintenance after exact sealed C4 and before C5. Classification: `OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR`.
- **Historical anchor:** exact sealed C4 commit `c4089ab31181c08dad880bf9e5d40c622c8c91c4`, tree `f427e23b6fe297a49a36a99fa037073b637d82ca`, with note blob `e1e106d3e7cecf51cbf3a6b5752a5b5268e97427`, note-body SHA-256 `63a19a8bd7ddf446f7d53873e107d47c4deaa5f9c8ef741b5c892d964f38db17`, `80/80 claims recorded-exact`, all 80 grades `TREE-EXACT`, and strict exit `0`.
- **Commit subject:** `fix: harden verifier execution boundary`
- **Private final root:** `.countershape/p07bc-c4h-final`

Every intended C4H grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C4H itself.

## Authority and topology

| Authority fact | Value |
| --- | --- |
| Authority source | `OWNER_OUT_OF_BAND` |
| predecessor-predeclared boundary | `false` |
| Topology authority derived from C4 | `false` |
| Historical anchor | exact sealed C4 |
| Grade inheritance or rewriting | `false` |

The owner explicitly authorized this maintenance boundary after C4 sealed. That instruction supplies process authority for C4H but is not cryptographically authenticated. C4 did not predeclare C4H, so this unit does not claim that its topology came from C4, does not inherit a C4 receipt, and does not rewrite or relabel any frozen C4 grade. C4's 80 `UNRECEIPTED` status rows are immutable pre-seal declarations whose separately sealed Git-note grades remain authoritative historical evidence.

`countershape/p07b-c-unit-paths/v21` has 23 ordered phase rows at `sha256:66f568efb313303c92b23a51b743200f15b47a451a9b3424aaa251e0228e7928`. It inserts exactly `C4 → C4H → C5`; the plan transition matrix has 11 accepted and 302 rejected cases, and the independently implemented unit-scope matrix has 11 accepted and 289 rejected cases. C4H's Markdown authority surface has 59 paths, 915 hostile rejections, and 305 controls. The exact structural catalog contains 966 cases; the executable catalog contains 972 after the same six independent fixed regressions. No product Go path or prefix belongs to C4H.

## Defects repaired

1. **Bounded fatal Git diagnostics.** A Git child that exits cleanly with empty stderr remains accepted. Error, signal, nonzero, stderr-only, and mixed-output cases remain fatal. Diagnostics report total stderr bytes, at most the first 256 raw bytes as lowercase hex, and an exact truncation boolean. The 256-byte case is not truncated; the 257-byte case is. NUL, invalid UTF-8, and every C0 control are represented only as hex. Stdout is never reflected. This makes the diagnostic terminal-safe, not secret-safe.
2. **Caller-umask-exact planning fixtures.** The real planning example now exercises caller umask `077` and `022`, produces exact mode-`0700` directories and mode-`0644` files, validates no-follow descriptor identity before replacement mutation, and restores the prior caller umask through nested success and failure cleanup.
3. **Canonical verifier entry.** Importing `tools/verify-current.mjs` remains inert and the canonical direct path executes normally. A symlink or other noncanonical entry spelling exits nonzero with `VERIFY_NONCANONICAL_ENTRY`; it cannot silently no-op.
4. **Real subprocess lock and recovery evidence.** A second verifier in the same repository fails before later stages with `VERIFY_ALREADY_RUNNING`; a verifier in a distinct repository remains independent. A SIGKILL leaves a canonical stale lock classified as `VERIFY_STALE_LOCK`; reviewed cleanup and a subsequent owner run prove recovery and lock release after actual process close.

The subprocess self-test keeps an always-resolving close observer separate from its failure-aware completion promise. Output, spawn, stdio, or stdin failures record the failure, kill if necessary, wait for the actual `close` event, and only then surface the failure or remove test repositories. This prevents a test-only false green caused by observing `exitCode` before stdio closure.

## Verification and throughput ruling

C4H changes no Go or Node product code, compiler/runtime contract, qualification catalog, or didrun installation. It changes no cache lifetime, package partition, parallelism, timeout, or ordering semantics. Direct build, vet, general tests, sensitive tests, and nested work remain at `p=1`; `GOMAXPROCS=2`, test `-parallel=2`, the nine sensitive packages, fresh private caches, the fail-fast O_EXCL production lock, and all fixed deadlines remain exact.

The throughput provenance is corrected without changing any sealed result. The sealed C1M ledger contained only the `992765 ms` cold baseline. The faster `976161 ms` value was a later same-tree observation whose following C1B-attempt event failed. C1V used the faster value, which produced a stricter threshold; this erratum neither weakens nor invalidates its sealed timing evidence.

C4H requires one C4H cumulative pass. C5 still requires three cumulative passes, all 56 isolated qualification cases, and the exact existing 79 labels, types, argv, and order. The spelling `--verify-c5-sealed-c4-note` remains byte-exact while its internal ancestry walk becomes `C4H → C4 → C4P → C4N → C4M → C4V → C3D`.

The operator proposal to shard C5/C6 proof ledgers is `DECLINED_WITH_REASON`: C5's frozen terminal predicate requires one exact 79-event ledger, and the current authority has no legal receipt transition that can replace any of those events with a witness boundary without weakening or relabeling the claim. Composite Proof Protocol v2 is `DEFERRED_WITH_REASON` until after C6B, when it can be designed prospectively. No `C5P`, `TREE_WITNESS`, `PROOF_ADAPTER`, or `C5B` boundary is introduced.

## Claim and nonclaim boundary

C4H proves only the four bounded repair families above, the unchanged current cumulative baseline on its staged tree, exact 18-path/no-prefix scope, the staged credential-pattern scan, and sealed-predecessor/ledger-chain integrity. It does not prove credential or secret absence, hostile same-UID isolation, continuous executable immutability, universal survivor absence, arbitrary target containment, C5 HTTP semantics, production readiness, adoption, security-review completion, or maintainership.

## Intended claim map

| Claim | Type | Intended grade |
| --- | --- | --- |
| `P07B-C C4H candidate phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4H independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4H plan checker defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4H sealed-C4 Git-note and C4P C4N C4M C4V C3D ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4H unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4H planning-example caller-umask hermeticity` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4H verifier defensive self-test with symlink and subprocess-lock cases` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4H cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4H exact eighteen-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C4H scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C4H sealed-C4 predecessor C4P C4N C4M C4V C3D ancestry and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Exact final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C4H`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C4H`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
4. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4h-sealed-c4-note`
5. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
6. `/opt/homebrew/bin/node tools/generate-p07-planning-example.mjs --exercise`
7. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
8. `/opt/homebrew/bin/node tools/verify-current.mjs`
9. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4H --source-final-gate`
10. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4H --credential-scan`
11. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4h-preseal-ledger`

The runbook is a declaration, not evidence. C5 remains blocked until C4H independently commits, seals, and verifies strictly.
