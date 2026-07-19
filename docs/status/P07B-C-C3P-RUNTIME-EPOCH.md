# P07B-C C3P — runtime-epoch semantic prerequisite

## State

- **State:** active pre-seal `SOURCE_FULL` prerequisite; every C3P grade below is `UNRECEIPTED`
- **Parent:** sealed C2B commit `ab5e2dd5b66702a1d1cd13eb0047b57a48469487`, tree `02fd396858ca41cff6d0ee561dce7ee65a3f94d0`
- **Scope:** 25 exact mode-`100644` paths, no prefixes; sorted-newline roster `sha256:8b047704df047cb96f1c5d19418c4cfe8502e4b4746bb4f24c0851dac8d018c0`
- **Commit subject:** `fix: use stable Darwin boot-session identity`
- **Next receipt unit:** C3PB owns exactly three paths with roster `sha256:531cbe600cf2747752749890ea9a15c3d0126e498da6838d4f6faeb1a0d7e885`; it may bind only the already-sealed C3P source evidence and cannot grade itself

## Defect and correction

`kern.boottime` is adjusted when Darwin's calendar time changes, so `DARWIN_KERN_BOOTTIME_V1` cannot identify one stable boot session. C3P replaces it with the exact `DARWIN_KERN_BOOTSESSIONUUID_V1` contract before any production `OfficialTarget` exists. The three joined examples are regenerated because the target digest change propagates into the finalized-run and execution references.

Future `internal/hostepoch` owns the live measurement. It will read only `kern.bootsessionuuid` twice through a bounded Darwin system-call edge, validate each 36-character hyphenated nonzero UUID, lowercase each sample, compare the normalized values, and issue an opaque capability over the typed digest. It has no public raw-value constructor or fallback clock. C3 and C4 remeasure it at their own physical boundaries.

This correction does not claim that a changed UUID proves child absence, a physical reboot, VM non-rollback, kernel trust, or safe reset. The historical C1/C2 receipts remain evidence for their historical trees; C3P supersedes only the future boot-profile meaning.

## Scope corrections for C3

- C3 now owns an exact forty-path, empty-prefix `SOURCE_FULL` roster with digest `sha256:c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb`.
- `internal/store/public_api_test.go` is included because the narrow inert attempt/target bridge must be compiler-visible and frozen rather than hidden from C2's public-surface gate.
- Direct single-target materialization accepts opaque `gitobj.InspectedTree`; it never creates a fake one-member world/candidate binding.
- C3B is declared as a separate exact three-path receipt unit with digest `sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67`.
- The dedicated C3P/C3PB receipt checker is enrolled in the cumulative verifier with both its phase-coherence gate and hostile self-test; the verifier and its self-test therefore belong to this exact source roster.

## Planned source claim map

| Claim label | Claim type | Pre-seal grade |
| --- | --- | --- |
| `P07B-C C3P planning and scope coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3P legacy clock-derived profile refusal` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3P joined semantic example regeneration` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3P contract model profile` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3P cumulative C1 architecture compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3P architecture defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3P unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3P cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3P cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3P exact twenty-five-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3P scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3P preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Gate

C3P is complete only after the exact source roster passes every planned command through didrun, the commit seals, its Git note is present, and `NO_COLOR=1 didrun verify --strict` exits `0`. Its own status remains pre-seal and cannot name or grade its future commit. At that boundary, the queued verifier-throughput proposal is routed in a separately scoped maintenance unit whose name and roster are intentionally undecided until C3P closes; it may not mutate this unit mid-seal. C3 remains blocked until that maintenance closes and a later C3PB receipt declaration independently binds the C3P source note, status, handoff, HTML snapshot, and ignored ledger, then itself commits, seals, and verifies strictly.

No C3P result proves a live boot measurement, runtime admission, Git materialization, attempt allocation, target publication, `OfficialTarget`, interlock acquisition, permit, subject spawn, finalized run, classification, product UX, security review, production readiness, adoption, or maintainership.
