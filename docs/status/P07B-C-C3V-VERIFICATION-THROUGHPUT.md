# P07B-C C3V — verification-throughput proposal reconciliation

## State

- **State:** active pre-seal `SOURCE_FULL` maintenance; every C3V grade below is `UNRECEIPTED`
- **Parent:** sealed C3P commit `f7b6e6bda7a8864969415ab8636c495902e78dd9`, tree `2d5555db63d46c837cf4e12f2dab2d04a41f4654`
- **Scope:** 7 exact mode-`100644` paths, no prefixes; sorted-newline roster `sha256:6cc5d8fe929f0f1dfa6ffc223d2e8bcee5ba4596a09caa9dc032103bbb5bafbd`
- **Commit subject:** `docs: reconcile verifier throughput proposal`
- **Execution/product delta:** none; C3V changes no verifier, repetition runner, Go package, runtime, product, or didrun implementation. Its only behavior change is verification-scope/checker authority: unit-scope schema v5 admits and orders this exact C3V boundary.

## Trigger

An external observer proposed a persistent private Go build cache, direct `p=8` general work with a serialized sensitive lane, a fail-fast single-verifier lock, and a narrower docs-only receipt battery. The proposal's snapshot predates sealed C1V. C3V routes each item against the current tree and the already-sealed qualification evidence rather than silently repeating or overriding that work.

This is a reconciliation of maintenance intent, not a claim that the proposal was wrong in the abstract. Several observations were valid; their repairs already exist. Where the proposed premise is unsafe for this Darwin/cgo tree or exceeds the qualified host-sharing profile, C3V preserves the earlier decline with its reason.

## Sealed C1V authority used for the ruling

C1V commit `88e62acb3023dcd6ee51950c2ad24bbcc2ca8900`, tree `6c98b3c384f01b63d1a00a002303041576896609`, is note-present and strict-clean with `10/10 claims recorded-exact`; all ten verbatim grades are `TREE-EXACT`. Its final ledger closed the exact 52-case focused qualification matrix and three complete cumulative verifier passes at `814384`, `815886`, and `813933` milliseconds, each below the fixed `878544` millisecond ceiling derived from the faster sealed `976161` millisecond C1M baseline.

That evidence is local Darwin/arm64 qualification for the sealed C1V tree, not a universal benchmark or future-regression guarantee. C3V does not relabel it, rerun only a favorable subset, or treat documentation as stronger authority than the C1V Git note and strict result.

## Item-by-item disposition

| Proposal item | Disposition | Ruling |
| --- | --- | --- |
| 1. Persistent content-addressed private `GOCACHE` keyed by admitted Go authority | `DECLINED_WITH_REASON` | Go's cache does not detect imported C-library changes. Countershape has real Darwin/cgo code; a Go-only key omits clang, SDK/header, libSystem, and prior writable-cache authority. `GOCACHE` therefore remains fresh below each mode-`0700` run root and is removed at finalization. |
| 2. Direct `p=8` general lane plus serialized sensitive lane | `INTENTIONAL; P8_DECLINED_WITH_REASON` | C1V already replaced global package serialization with exact tiering: direct build/vet/general tests use `-p=2`, while six frozen sensitive packages run separately at `-p=1` and every nested command inherits `-p=1`. The `p=4` candidate passed Go packages but caused two permanent later AST-timeout events; `p=8` was therefore declined for this shared host. The accepted `p=2` profile passed the historical 50x/20x matrix and three cumulative ceilings. |
| 3. Exclusive single-verifier lock | `INTENTIONAL` | The verifier already acquires `.countershape/verify-current/active.lock` with exclusive creation before authority admission or work. A second live/indeterminate owner fails fast. An absent PID produces review-first `VERIFY_STALE_LOCK`; automatic stale deletion remains declined because two recovery contenders can race and remove a new owner's lock. |
| 4. Narrow docs-only receipt boundary | `INTENTIONAL` | `RECEIPT_RECONCILIATION` already admits only an exact empty-prefix Markdown/receipt-declaration roster with seven machine-declared claims: reconciliation, checker self-test, local snapshot match, receipt-only Go build, exact staged scope/diff, credential scan, and chain integrity. It cannot claim the cumulative suite, runtime, security, or unchanged behavior. Source units remain `SOURCE_FULL`. |

## Preserved current verifier contract

C3V intentionally leaves `tools/verify-current.mjs`, `tools/verify-current-selftest.mjs`, and `tools/verify-go-test-repetition.mjs` byte-for-byte unchanged. The current contract remains:

- fresh private `HOME`, `TMPDIR`, `GOTMPDIR`, `GOCACHE`, `GOPATH`, and `GOMODCACHE` beneath one run root;
- ambient `GOFLAGS=-mod=readonly -buildvcs=false -p=1`, `GOMAXPROCS=2`, offline module resolution, fixed locale/timezone/color, and admitted tool authorities;
- direct two-job build, vet, and general-package tests;
- six exact sensitive packages tested once afterward at `-p=1`, with exact package-partition revalidation;
- one fail-fast O_EXCL verifier owner and conservative review-first stale-lock handling; and
- explicit `SOURCE_FULL` versus `RECEIPT_RECONCILIATION` profiles selected by machine declaration, never diff heuristics.

Unit-scope schema v5 adds only the exact C3V ordering and roster authority between C3P and C3PB. It does not relabel any historical profile.

The three deliberately unchanged verifier authorities are pinned to their C3P-tree byte identities: `tools/verify-current.mjs` is `sha256:ce76506d67c8d79b6359f5825dcc85abbc60b87344c8fb4884335faa03babbc3`; `tools/verify-current-selftest.mjs` is `sha256:b18771c16cb85b5a85fdadb0ee0aa39d7357217e302ea3b0b690ba56cfc2d558`; and `tools/verify-go-test-repetition.mjs` is `sha256:1321fb1381f1f74d0959b9ceb40b4dcc4abc637bb281ff2af759863ede3bfd5a`. Byte identity is narrower than a universal unchanged-behavior claim.

Because C3V changes no execution setting, the proposal's conditional 50x/20x requalification trigger does not fire. C3V may cite the sealed C1V qualification but may not claim a new qualification or a new performance result.

If a later unit changes any of those bytes or settings, this no-delta ruling no longer applies. That unit must expand its scope to the verifier authorities and repetition helper, rerun the load-sensitive qualification matrix, and close its own commit/seal/strict boundary.

## Live S6 didrun bug finding

`C3V-S6-DIDRUN-INTERRUPT-1`: during an unclaimed development run, operator SIGINT caused didrun to exit `130` with a Python `KeyboardInterrupt` traceback before it appended an event for the in-flight cumulative verifier; process inspection found no surviving verifier child. It left the mode-`0600` lock for PID `84735` in place; review proved the PID absent, no process held the file, and the owner, repository-root digest, and mode matched before one explicit didrun-wrapped recovery event removed only that lock. The interrupted command therefore has no didrun receipt and supports no claim. This is a didrun interruption-capture bug finding, not a verifier failure or a sealed-tree result; final evidence must complete normally in a fresh ledger.

## Planned claim map

| Claim label | Claim type | Pre-seal grade |
| --- | --- | --- |
| `P07B-C C3V throughput proposal disposition coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3V disposition and frozen-verifier defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3V unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3V cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3V cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3V exact seven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3V scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3V preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Gate and nonclaims

C3V is complete only after all eight commands run through didrun on one unchanged staged tree, the exact seven paths commit, the commit seals, its Git note is independently readable, and `NO_COLOR=1 didrun verify --strict` exits `0`. This pre-seal status cannot name or grade C3V's future commit. C3PB remains blocked until C3V closes; C3V does not create, edit, or grade the C3P receipt declaration.

C3V proves no throughput improvement beyond the historical C1V receipts, no portable performance, warm-cache correctness, p8 stability, secret absence, security review, production readiness, adoption, or maintainership. It changes no Countershape runtime capability and creates no live host, Git, Node, target, admission, process, result, or classifier authority.
