# P07B-C C3P — runtime-epoch semantic prerequisite

## State

- **Unit:** closed `C3P` source receipt, reconciled by the independently sealed `C3PB` descendant
- **Profile:** historical `SOURCE_FULL` source with a closed `RECEIPT_RECONCILIATION` descendant
- **Source receipt:** C3P source commit `f7b6e6bda7a8864969415ab8636c495902e78dd9`, tree `2d5555db63d46c837cf4e12f2dab2d04a41f4654`, is sealed, note-present, and strict-clean; every C3P source grade below is `TREE-EXACT`. This source-receipt record binds only that existing C3P source; the independently sealed C3PB descendant is outside these C3P grades.
- **Parent:** sealed C2B commit `ab5e2dd5b66702a1d1cd13eb0047b57a48469487`, tree `02fd396858ca41cff6d0ee561dce7ee65a3f94d0`
- **Scope:** 25 exact mode-`100644` paths, no prefixes; sorted-newline roster `sha256:8b047704df047cb96f1c5d19418c4cfe8502e4b4746bb4f24c0851dac8d018c0`
- **Commit subject:** `fix: use stable Darwin boot-session identity`
- **Closed receipt unit:** C3PB owns exactly three paths with roster `sha256:531cbe600cf2747752749890ea9a15c3d0126e498da6838d4f6faeb1a0d7e885`; it closed at commit `13369122ba7d5557eba1949095c1135a41843070`, tree `d710d9b249786f3fb648f566ae542f1d9ddec180`, with `7/7 claims recorded-exact` and strict exit `0`

## Defect and correction

`kern.boottime` is adjusted when Darwin's calendar time changes, so `DARWIN_KERN_BOOTTIME_V1` cannot identify one stable boot session. C3P replaces it with the exact `DARWIN_KERN_BOOTSESSIONUUID_V1` contract before any production `OfficialTarget` exists. The three joined examples are regenerated because the target digest change propagates into the finalized-run and execution references.

Future `internal/hostepoch` owns the live measurement. It will read only `kern.bootsessionuuid` twice through a bounded Darwin system-call edge, validate each 36-character hyphenated nonzero UUID, lowercase each sample, compare the normalized values, and issue an opaque capability over the typed digest. It has no public raw-value constructor or fallback clock. C3 and C4 remeasure it at their own physical boundaries.

This correction does not claim that a changed UUID proves child absence, a physical reboot, VM non-rollback, kernel trust, or safe reset. The historical C1/C2 receipts remain evidence for their historical trees; C3P supersedes only the future boot-profile meaning.

## Scope corrections for C3

- C3 now owns an exact forty-path, empty-prefix `SOURCE_FULL` roster with digest `sha256:c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb`.
- `internal/store/public_api_test.go` is included because the narrow inert attempt/target bridge must be compiler-visible and frozen rather than hidden from C2's public-surface gate.
- Direct single-target materialization accepts opaque `gitobj.InspectedTree`; it never creates a fake one-member world/candidate binding.
- C3B is declared as a separate exact three-path receipt unit with digest `sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67`.
- The dedicated receipt checker was originally enrolled for C3P/C3PB source reconciliation and remains in the cumulative verifier with its hostile self-test. Its current phase contract has evolved after sealed C3PB: both synthetic receipt branches now require the sole active C3A descendant tuple, while the C3P grades themselves remain phase-neutral.

## C3P source receipt map

- C3P source strict exit: `0`
- C3P source strict claims: `12/12 claims recorded-exact`

This receipt binds only the already-existing C3P source commit. The independently sealed C3PB descendant is outside this C3P source grade map, and the active C3A unit cannot grade itself here.

C3P source commit `f7b6e6bda7a8864969415ab8636c495902e78dd9`, tree `2d5555db63d46c837cf4e12f2dab2d04a41f4654`, is sealed, note-present, and strict-clean with `12/12 claims recorded-exact`.

The C3P seal records `secrets_override: true` after 46 local scanner findings with kinds `high-entropy`; this is not evidence of secret absence.

The separately claimed C3P structured staged credential scan reported `0` findings; its provenance is sealed supporting event `10`, not the Git note alone.

Local ignored C3P ledger archive `.didrun-history/2026-07-19-p07b-c-c3p-final/.didrun/` contains `12` sealed events and `12` archived session events; post-seal events, if any, are outside the sealed manifest.

C3P ledger manifests use `sorted-relative-posix-path-tab-size-tab-sha256-newline/v1`; all-files SHA-256 is `ec95dd6e19d5eeaec9786e0c5e8a3d51dd8a18f569324c75ddb58c3e54185dc2` and objects-only SHA-256 is `6926948d1012d1442b01f8b30eb2a946d81dbe51af1db25177db899e19141104`.

C3P archive core SHA-256 values are session `9bbb9ec54885026e6d2fab4d8cb9f0a4c466c2b222e9817a83f043fb4255a2f2`, claims `23d492233f974cabfab1479fe915009f7c0cb4ea4762912ad270848bf980a59c`, seals `f8b9f1fc3f9fbfe76bbb11ce885009414780d1c28bfe5ab16baa892fd4f5e1ab`, and .gitignore `cdbcae15105d6b781e620813c79c7e868740d4e9cc53ce6f5fcbbc12387adf4b`.

The local C3P HTML snapshot is `.countershape/evidence/p07b-c-c3p-final-f7b6e6bda7a8.html`, SHA-256 `7ffcf52106b90746638d9df7b3edadf994e667954ebdcde3510938a981fdb148`, 8483 bytes; it is not a portable strict witness.

The C3P Git note blob is `09dd345d428554b84da592841e48bdbcc42304c3` with body SHA-256 `06237c6d9f60d4d8f9a5c0edd971e6b0bb2bf7c977d45e89119c427b09000d11`.

| Claim label | Claim type | Verbatim source grade |
| --- | --- | --- |
| `P07B-C C3P planning and scope coherence` | `tests-pass` | `TREE-EXACT` |
| `P07B-C C3P legacy clock-derived profile refusal` | `tests-pass` | `TREE-EXACT` |
| `P07B-C C3P joined semantic example regeneration` | `tests-pass` | `TREE-EXACT` |
| `P07B-C C3P contract model profile` | `tests-pass` | `TREE-EXACT` |
| `P07B-C C3P cumulative C1 architecture compatibility` | `tests-pass` | `TREE-EXACT` |
| `P07B-C C3P architecture defensive self-test` | `tests-pass` | `TREE-EXACT` |
| `P07B-C C3P unit-scope defensive self-test` | `tests-pass` | `TREE-EXACT` |
| `P07B-C C3P cumulative verifier self-test` | `tests-pass` | `TREE-EXACT` |
| `P07B-C C3P cumulative verification` | `tests-pass` | `TREE-EXACT` |
| `P07B-C C3P exact twenty-five-path staged scope and diff integrity` | `command-succeeded` | `TREE-EXACT` |
| `P07B-C C3P scoped staged credential-pattern scan` | `command-succeeded` | `TREE-EXACT` |
| `P07B-C C3P preceding didrun chain integrity` | `command-succeeded` | `TREE-EXACT` |

## Gate

The C3P source, intervening C3V/C3M maintenance, and C3PB receipt boundary are sealed and strict-clean. The active C3A maintenance unit owns only its separately declared architecture/model/checker correction and remains `UNRECEIPTED`. The grades above come only from the sealed C3P source note and its receipt declaration; none grades C3PB, C3A, or later C3 behavior. C3 remains blocked until C3A independently closes.

No C3P result proves a live boot measurement, runtime admission, Git materialization, attempt allocation, target publication, `OfficialTarget`, interlock acquisition, permit, subject spawn, finalized run, classification, product UX, security review, production readiness, adoption, or maintainership.
