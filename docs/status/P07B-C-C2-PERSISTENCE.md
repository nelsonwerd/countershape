# P07B-C C2 persistence substrate status

- **Unit:** `C2` source receipt, reconciled by the active `C2B` working unit
- **Profile:** `RECEIPT_RECONCILIATION` for this working unit; the bound source used `SOURCE_FULL`
- **Source receipt:** C2 source commit `19c90786d9832e21ec86daea96ca514cc7304836`, tree `1557f1e26762d717327a2ec770f8153daa0baf3e`, is sealed, note-present, and strict-clean; every C2 source grade below is `TREE-EXACT`. This C2B receipt-document working unit binds only that existing source and remains `UNRECEIPTED` until its own commit, seal, note, and strict boundary.
- **Predecessor:** sealed C1B commit `46c48507fe998ea04e121470d6aa8ba0b38b3dae`, tree `6a6ee49048c2639d102641cecab4e6d99f31377e`
- **Scope:** 26 exact paths, no prefixes; sorted-newline roster `sha256:a463e4e9ad9830cb420bbad1da494c26d55e3f68b983abfd683a0c8cb1c3a522`

This is the separately enrolled receipt-document working unit for the already-existing C2 source boundary. It records only source facts admitted from Git, the didrun note, the exact-commit HTML report, and the ignored local ledger snapshot. It does not grade the working unit that adds this receipt.

## Sealed prerequisite

C1B is independently sealed at the commit and tree above with `7/7 claims recorded-exact`, every grade `TREE-EXACT`, strict exit `0`, and a present `refs/notes/didrun` note. Its exact report is `.countershape/evidence/p07b-c-c1b-final-46c48507fe99.html`, SHA-256 `ac9ff63000a3f4207df4a8909b42baf4fad80b635935606fc9d886038a4bc156`, 7002 bytes. The note blob is `4a45557bce22a7a8c93af9e3fa3d0ff1092654ea`; its body SHA-256 is `79d1e0feb01925f67ed7512c5577ab72df6c17ace567d77e4c0c79a3c72e7ac2`.

## Exact C2 boundary

C2 is the Darwin-private persistence substrate. C2 production never issues official target or permit authority; it adds storage mechanics only:

- three versioned typed relationships: conformance-attempt to target, target to finalized run, and `(run, derived classifier-profile)` to execution;
- generic object publication followed by an exact typed hard-link witness and an exact create-once relationship record;
- store-instance-bound inert records that cannot issue `OfficialTarget`, `RunPermit`, or any public semantic authority;
- one store-wide non-expiring `ExecutionInterlock`, a deterministic target-keyed `StartClaim`, and process-local winner authority that is never reconstructed from disk;
- exact mutation effects `KNOWN_NO_EFFECT`, `EXACT_CONVERGED`, and `AMBIGUOUS`;
- bounded private evidence manifests and packs with at most 16 blobs and 64 MiB aggregate content; and
- logical availability `RETAINED | PURGED | MISSING_UNEXPECTED`, separate from immutable canonical history.

Every exact-visible create-once artifact is converged by synchronizing its parent directory and reopening exact bytes before it can be promoted into a durable record. That rule covers typed relationships, StartClaims, private manifests, clear receipts, and purge intents. It closes the retry hole where a link could be visible after an `after-link` interruption but not yet directory-durable.

## Interlock and clear-receipt protocol

The only receiptless admission state is the first-ever absent interlock bootstrap. Acquisition requires the caller boot digest to equal the target body's `boot_session_binding.identity_digest`; the resulting HELD state and StartClaim repeat that exact boot identity. A previously persisted `CLEAR` is historical transition state, not a live-boot measurement and not a rule that freezes every future target to that historical boot. C3 must establish live boot authority, and C4 must remeasure it at admission.

Every persisted `CLEAR` is admission-valid only with one exact durable `ExecutionInterlockClearReceipt`. Its cause is closed to `KNOWN_NO_CLAIM`, `FINALIZED_RUN`, or `CHANGED_BOOT`; it reconstructs the exact previous HELD digest and fields plus the exact previous-to-CLEAR revision/generation transition. A visible receipt is synchronized and reopened before use. `CLEAR` without that receipt is ambiguous and blocks all later acquisition.

The receipt is a trusted-writer operational admission marker. It does not independently prove that a child never started, that a terminal run caused the transition, or that a prior child is absent. Its terminal-run digest is not a standalone semantic provenance proof. C4 owns physical chronology and terminal authority.

## Private evidence and purge

The private manifest joins one exact target, attempt, and StartClaim. It maps the closed logical evidence roster onto checked byte ranges and distinct private-byte digests in one exact pack. A finalized-run record refuses unless the manifest, pack, logical roster, and named target/run joins reopen exactly.

Purge writes and durably reopens an exact no-serve intent before unlinking a pack. A retry converges an already-visible exact intent before cleanup. `PURGED` is logical unavailability, not confidentiality or secure erasure. Missing pack bytes without a valid purge intent produce `MISSING_UNEXPECTED`. Neither state rewrites target, run, execution, relationship, study-head, or retention-at-finalization bytes.

## Multi-pass corrections

The build loop materially changed the first draft:

- separated C2 inert records from C3 official-target issuance and C4 physical permit/run authority;
- made ambiguity dominate joined mutation errors after any durable partial effect;
- closed relation algebra, typed-parent, cross-store, wrong-kind, wrong-profile, alternate-link, and raw-key substitutions;
- bounded parser arithmetic before addition, conversion, allocation, and pack slicing;
- made manifest identity and logical roster prerequisites of finalization, availability, and purge;
- added receipt-backed CLEAR transitions rather than treating a bare CLEAR file as released;
- joined target and StartClaim boot identities exactly at acquisition, restart reopen, and downstream private/run persistence without overclaiming historical CLEAR as live boot authority;
- required exact-visible convergence before any create-once record is promoted after restart;
- reasserted retained relation, publication, StartClaim, private-manifest, pack, purge, and fixed-root directory identities at later authority and mutation boundaries, with both symlink and real mode-`0700` exact-content replacement-refusal coverage;
- made canonical case spelling a checked path invariant for fixed/dynamic directories and exact relationship, claim, manifest, pack, purge-intent, and interlock leaves; and
- replaced the exported-surface text approximation with compiler-parsed Go AST enumeration of declarations and every named or embedded struct field, while freezing the compiler-derived production import roster and executing the public-surface profile inside the cumulative architecture boundary;
- bound each future C2 source grade to one index-specific command identity, with raw argv revalidation from the ignored ledger and an explicit structural-only ceiling for any didrun-redacted Git-note arguments;
- required the future source receipt to reopen one exact parent and a raw Git diff of seven added plus nineteen modified mode-`100644` blobs over the exact 26-path roster; and
- repaired the cumulative verifier's stale C1-only plan-self-test marker after the plan checker became a joint C1/C2 boundary. The first development receipt remains permanently failed and unclaimed, while its repaired development rerun passed all 37 rows at exit `0`; that rerun is still unclaimed and cannot substitute for the final frozen-tree ledger.

## Exact targeted profiles

The architecture transcript parser requires one run and one pass for every exact top-level symbol and fails on missing matches, skips, duplicates, foreign packages, unexpected symbols, or package failure.

| Profile | Exact top-level tests |
| --- | ---: |
| `c2-nonhead-persistence` | 6 |
| `c2-interlock` | 7 |
| `c2-private-evidence` | 6 |
| `c2-public-surface` | 2 |

The C2 architecture checker composes `C2 -> B -> C1`. Its C2 hostile self-test has a frozen 46-case metadata roster plus seven Go-JSON parser cases. The inherited B self-test admits only the three exact C2 symbols in `internal/store/nonhead_contract.go`; lookalike names and files remain forbidden. The exported-surface proof and process-capability import refusal are compiler-derived and load-bearing rather than regex-authoritative.

## C2 source receipt map

- C2 source strict exit: `0`
- C2 source strict claims: `14/14 claims recorded-exact`

This receipt binds only the already-existing C2 source commit. C2B cannot name or grade its own commit, tree, Git note, or strict result.

C2 source commit `19c90786d9832e21ec86daea96ca514cc7304836`, tree `1557f1e26762d717327a2ec770f8153daa0baf3e`, is sealed, note-present, and strict-clean with `14/14 claims recorded-exact`.

The C2 seal records `secrets_override: true` after 130 local scanner findings with kinds `high-entropy`; this is not evidence of secret absence.

The separately claimed C2 structured staged credential scan reported `0` findings; its provenance is sealed supporting event `12`, not the Git note alone.

Local ignored C2 ledger archive `.didrun-history/2026-07-19-p07b-c-c2-source/.didrun/` contains `14` sealed events and `15` archived session events; post-seal events, if any, are outside the sealed manifest.

C2 ledger manifests use `sorted-relative-posix-path-tab-size-tab-sha256-newline/v1`; all-files SHA-256 is `4d63558b56a04383f348e972966e05e2fb653d45b81e3cd2a902d4b8a2f2c017` and objects-only SHA-256 is `295870bbd68a312920a50ec3a488d9016dc3ae63a435ddd9e4a04dcae092f61f`.

C2 archive core SHA-256 values are session `3c1ee199346c09162e157bd0a8c42a975c6fe02f4a2834a6ea03382c26b69183`, claims `8bfff82872a82d0bf1acfc0dc0fd87e2bd0483a6fa15a28d72fec2a81bccac63`, seals `f017ac5ae0c08fd8424710611992f0193698b3f49c0064c08756a8233b06495a`, and .gitignore `cdbcae15105d6b781e620813c79c7e868740d4e9cc53ce6f5fcbbc12387adf4b`.

The local C2 HTML snapshot is `.countershape/evidence/p07b-c-c2-final-19c90786d983.html`, SHA-256 `a91395284ae54487d34482c29143893bdb19cebc71afb8a4116ffec7ad3fd00f`, 9063 bytes; it is not a portable strict witness.

The C2 Git note blob is `0e43a1920c7432150167ca3756fbab3cb67e0a00` with body SHA-256 `52f50ced8db8d7873970c83f32537ae118200aa41eca8c6058ec275875489fe4`.

| Claim label | Claim type | Verbatim source grade |
| --- | --- | --- |
| `P07B C2 exact typed nonhead persistence profile` | `tests-pass` | `TREE-EXACT` |
| `P07B C2 interlock StartClaim and receipt-backed clear profile` | `tests-pass` | `TREE-EXACT` |
| `P07B C2 private evidence lifecycle profile` | `tests-pass` | `TREE-EXACT` |
| `P07B C2 exported authority and discovery absence` | `tests-pass` | `TREE-EXACT` |
| `P07B C2 focused store race suite` | `tests-pass` | `TREE-EXACT` |
| `P07B C2 cumulative architecture boundary` | `tests-pass` | `TREE-EXACT` |
| `P07B C2 cumulative architecture defensive self-test` | `tests-pass` | `TREE-EXACT` |
| `P07B C2 predecessor B compatibility` | `tests-pass` | `TREE-EXACT` |
| `P07B C2 predecessor B defensive self-test` | `tests-pass` | `TREE-EXACT` |
| `P07B C2 cumulative verifier self-test` | `tests-pass` | `TREE-EXACT` |
| `P07B C2 cumulative verification` | `tests-pass` | `TREE-EXACT` |
| `P07B C2 exact 26-path staged scope and diff integrity` | `command-succeeded` | `TREE-EXACT` |
| `P07B C2 scoped staged credential-pattern scan` | `command-succeeded` | `TREE-EXACT` |
| `P07B C2 preceding didrun chain integrity` | `command-succeeded` | `TREE-EXACT` |

Development ledgers contain superseded successes and permanent nonzero events. They are intentionally unclaimed and support none of the rows above. The grades above come only from the sealed source ledger and Git note declared by this receipt.

## Separately enrolled C2B receipt boundary

The C2 source commit deliberately cannot contain its own future commit, tree, Git note, strict verdict, HTML digest, or ignored-ledger manifest. After C2 independently seals and verifies strictly, C2B must reconcile those then-existing facts before C3 begins. C2B is an exact three-path, no-prefix `RECEIPT_RECONCILIATION` unit over `docs/HANDOFF_MODE_C.md`, this status file, and the then-created `spec/verification/p07b-c-c2-receipt.json`; its sorted-newline roster digest is `sha256:5e30e009bb29672d585ece9893bd7c8db198915335a73ded3f3c790ad18f665b`.

C2B may run only its seven machine-declared claims: source receipt reconciliation, defensive receipt-checker self-test, declared local source-evidence snapshot match, receipt-only Go build, exact three-path staged scope/diff integrity, scoped staged credential-pattern scan, and preceding didrun-chain integrity. It cannot claim a cumulative suite, runtime behavior, security, or unchanged behavior, and it cannot name or grade its own future commit, tree, note, or strict result. `spec/verification/p07b-c-c2-receipt.json` must remain absent from the C2 source commit.

## Explicit nonclaims and gate before C3

C2 proves no official target, production spawn admission, process spawn, physical run, child absence, semantic classification, result discovery, list/latest/status API, study-head transition, clone resistance, confidentiality, secure deletion, hostile same-user containment, network-filesystem behavior, forced-power-loss recovery, or classifier implementation provenance.

C3 remains blocked until C2's exact 26-path source unit commits, seals, has a present didrun note, passes `NO_COLOR=1 didrun verify --strict` with every intended row recorded-exact, has an exact-commit HTML report, and the separate exact three-path C2B reconciliation also commits, seals, proves its own note, and verifies strictly. C3 must then establish live Git/materialization/runtime/attempt/boot authority; it may not treat this C2 substrate as official execution authority.
