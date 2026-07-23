# P07B-C C4P — future-surface phase maintenance

Classification: `DEFECT_REPAIR`.

C4P owns exactly these 7 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C4P-FUTURE-SURFACE-PHASE-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Their sorted-newline roster digest is `sha256:5dc083fadb60003b0ba96516d0a01f91daa88b7298b142018b5eda00e48ab59b`.

- **Boundary:** active pre-seal `SOURCE_FULL` future-surface phase maintenance after sealed C4N and before C4. Classification: `DEFECT_REPAIR`.
- **Parent:** sealed C4N commit `b915d43cced850936c46b52620654f7506bdb993`, tree `3ea928443ecb32d6457babe2bce02ddec1d9632f`, is note-present with note blob `949b8d2fbad1cce74e92971a30fdd31fae62f256`, note-body SHA-256 `f5151d5a92de5fbdb3a2a7f5faa8e0e6b89afddb72d6c21d9ad643f4f8fbee6a`, `10/10 claims recorded-exact`, strict exit `0`, and recorded `secrets_override: true`.
- **Commit subject:** `fix: make B future-surface self-test phase-aware`

Every intended C4P grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C4P itself.

## Trigger and ruling

The cumulative development pass reached the plan self-test only after the C4 product, architecture, and behavior suites were green. The self-test then unconditionally applied the sealed-C3D baseline scanner to the ambient staged C4 runner topology. That was a checker false red: the direct baseline oracle was valid, but its ambient caller ignored the live phase.

The repair is table-relative. `countershape/p07b-c-unit-paths/v20` contains 22 ordered phase rows at `sha256:6d7e41d5ff22fc73118ceb0ae9f27a2151e7121d3a91c8945a391b3f42868f7d`. Ordered anchors select exactly four modes:

- `NONE` before C4V;
- `BASELINE` from C4V through every maintenance row before C4;
- `C4` from C4 through every maintenance row before C5;
- `C5` from C5 onward.

One selector serves both the ambient plan gate and the future-surface self-test, but it is judged against a separately frozen 22-record phase-to-mode oracle at `sha256:e042018eedc9ff7b78abc79361d6a824d0d4b4080e510f5b76336a3b39c8ff34`. That oracle enumerates every v20 boundary independently of the selector thresholds. Unknown rows fail. Each declared row gets a routed positive control, each enforcing row gets a wrong-epoch rejection, and C5-without-runner plus HTTP-without-scope remain explicit negatives. Every `BASELINE` forward fixture injects the exact frozen historical 27-row inventory and production-path projection, so restored ambient C4 source cannot contaminate C4V/C4M/C4N/C4P history. The strict synthetic sealed-C3D baseline and all 42 direct hostile cases stay independent and unchanged.

The resulting candidate-transition matrix has 10 accepted and 266 rejected cases. The independent scope implementation has 10 accepted and 253 rejected cases. C4P owns the v20 specification mutation; C4 and every later row must preserve its parent bytes.

Predecessor replay also exposed a bounded didrun redaction epoch difference: sealed C4V/C4M notes use the exact double high-entropy marker at hermetic GOCACHE position 6, while sealed C4N uses one marker plus its exact `.countershape/.../gocache` suffix. Acceptance is epoch-scoped: C4V/C4M validators allow only raw or legacy-double position 6, while C4N/C4P/C4 validators allow only raw or exact suffix-preserving position 6. At GOCACHE position 6, the other epoch's form, a bare marker, or a foreign suffix fails; other prefix positions retain their own fixed projections, and redacted command tails fail. This preserves real sealed history without letting a new receipt hide the cache-root suffix behind an older opaque projection.

The v20 generic structural catalog contains 902 cases at `d52bdf89855bb306c844103851452a439534e3042e4741bed8787386da529770`, split into 484 accepted controls and 418 hostile rejections; six fixed semantic regressions make 908 executable cases at `5c2b8ecb64483ed999b817c74fe6751b2dbd689edf3c0740fbceb920c6f7f203`, split into 490 accepted controls and 418 hostile rejections. The separate nine-row phase-extension cardinality catalog is pinned at `6ac375496ed3a160375e5f55258299daad807d9047bc2af4b9be46d2058c4e8c` and preserves sealed C4M's exact 47-path digest while deriving C4N/C4P/C4 as 48/49/50 paths. C4P's exact 49-path Markdown authority is `sha256:d901a41a7b01febb9bf6f0f9d4823056e9fdb1109746a5adf132877b5ae791be`, with 765 rejections and 255 controls; C4 derives the 50-path authority at `sha256:3375ec014b2c977d8d27f052b0eb516b119e00b47c775afaf6c33ff757336c9e`, with 780 rejections and 260 controls. Across all nine forward phases, the aggregate is 7,020 rejections and 2,340 controls plus nine full-plan controls.

## Consolidation proposal disposition

- `DEFECT_ALREADY_REPAIRED`: sealed C3D/C4N already replaced bespoke structural boundary tuples with the data-driven phase table and generic absent/present/hostile fixture generator.
- `INTENTIONAL_HYBRID`: exhaustive structural predecessors and cardinalities remain generated, while fixed C3R/C3Q/C3T/C3U/C4V/C4M semantic regressions remain independent executable oracles.
- `DECLINED_WITH_REASON`: a pure four-field replacement cannot honestly express note projection, outer/historical scan asymmetry, validated absence, terminal-byte preservation, complete-corpus transport, or frozen historical cardinality, so it would weaken rather than consolidate those checks.

C4P repairs only future-surface phase selection and deletes no prior hostile.

## Preserved C4 checkpoint

C4 product work remains immutable in stash `90d3c5010b44002fee80811966cc3c577a016d98`, message `countershape-c4-pre-c4p-checkpoint-20260722`, with identical main/index tree `96c3d6bf25f5994da35ca004d447870b8075adb9`. Its mixed 68-event development ledger is archived at `.didrun-history/p07b-c-c4-development-pre-c4p-96c3d6bf25f5/.didrun`; it supports no C4P or C4 capability claim. Earlier stash `c419c38c608dc6bec7e23e0fc379df28d3cb2d47` also remains retained. Neither checkpoint is dropped before C4 later closes independently.

## Intended claim map

| Claim | Type | Intended grade |
| --- | --- | --- |
| `P07B-C C4P candidate phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4P independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4P phase-aware future-surface defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4P sealed-C4N Git-note and C4M C4V C3D ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4P unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4P cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4P cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4P exact seven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C4P scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C4P sealed-C4N predecessor and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C4P`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C4P`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
4. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4p-sealed-c4n-note`
5. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
6. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
7. `/opt/homebrew/bin/node tools/verify-current.mjs`
8. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4P --source-final-gate`
9. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4P --credential-scan`
10. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4p-preseal-ledger`

The runbook is a declaration, not evidence. C4 remains blocked until this boundary independently closes.
