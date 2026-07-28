# P07B-C C4K — C3 Go test-timeout maintenance

This is a pre-seal source declaration. It defines the exact C4K repair and proof contract but cannot receipt its own result.

## State

- **Boundary:** active pre-seal `SOURCE_FULL` prerequisite after exact sealed C4I and before the unchanged parked C4J maintenance payload. Classification: `OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR`.
- **Authority:** `OWNER_OUT_OF_BAND`; the owner authorized continuation and blocking-defect repair, but sealed C4I did not predeclare C4K. `predecessor-predeclared = false`; no external signed instruction artifact exists, so this authority remains unevidenced beyond the owner-attributed session instruction and does not inherit or rewrite any C4I grade.
- **Parent:** sealed C4I commit `d784ace97dd03660c6c724e1cb8f838b869681ec`, tree `e38b702884b9df128b41bfa7710459dd92f901d7`, note blob `f1633147e2943e7272abf3b9b903ac72ab274494`, note-body SHA-256 `f221a99aaa1c0037d0de7ebe8c311bfd0ff1653434c4042f30b00a67d9c4e033`, subject `fix: stabilize verification surfaces`, `17/17 claims recorded-exact`, every grade `TREE-EXACT`, and strict exit `0`.
- **Commit subject:** `fix: bound C3 architecture test timeout`
- **Verification profile:** `SOURCE_FULL`; receipt C3P is `PRESENT`, receipt C3 is `PRESENT`, and receipt C6A is `ABSENT`.
- **Product authority:** none. C4K changes only the C3 architecture verification command, its defensive self-test, phase governance, and documentation; it changes no production Go implementation, generated Node product bytes, C4J timeout policy, C5 product behavior, C5 qualification case, C5 claim label/type/argv/order, package partition, parallelism, retry rule, or cumulative-row order.

Every intended C4K grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C4K itself.

## Trigger and classification

C4J final attempt 2 is permanent history at `.didrun-history/p07b-c-c4j-final-attempt-2-c3-go-timeout-6a6bf385eeed/.didrun/`, with private root `.countershape/p07bc-c4j-final-attempt-2-c3-go-timeout-6a6bf385eeed`. Its first cumulative verifier was green. Its second reached the C3 architecture self-test and Go terminated the exact official-target package after the implicit ten-minute default, while the outer architecture checker still had diagnostic time remaining. The attempt produced no commit, seal, note, strict result, HTML, or reusable C4J grade.

The repaired 19-path C4J payload is protected in stash `6357d7e039d79586e18210ebde283cc4ba0a28e0`. That checkpoint supplies no C4K or C4J grade. C4K is a separately scoped prerequisite because the C4J contract deliberately excludes the two architecture-checker files that own the defect.

## Exact repair

Only the exact `c3-official-target` Go JSON profile receives `-timeout=12m`; the other four C3 profiles and all six C4 profiles receive no inner timeout flag, the outer per-profile process ceiling remains `900000` milliseconds, the C3 clean-selftest ceiling is `1080000` milliseconds, and the cumulative verifier child ceiling remains `1200000` milliseconds.

The nested deadline ladder is therefore `720 s → 900 s → 1080 s → 1200 s`: Go package, architecture profile, clean C3 defensive self-test, then cumulative-verifier child. The argument and envelope self-tests prove the flag appears once on the official C3 profile and nowhere else, bind all four ceilings, and preserve real diagnostic margin at every nesting level. C4K adds no retry, changes no package partition or parallelism, removes no assertion, and preserves cumulative-row order. Twelve minutes is an evidence-based diagnostic margin over the observed approximately 8.5-minute green cluster and the ten-minute cutoff, not authority to keep extending deadlines when a real regression appears.

## Phase authority and parked C4J

The machine authority is `countershape/p07b-c-unit-paths/v23` with 25 ordered phase rows and authority SHA-256 `6996cec6d83a380b82c912aebea52797984fadacdc0c9a053cf69c79f90e0a45`. The plan transition matrix has 13 accepted and 380 rejected cases; the independently implemented unit-scope matrix has 13 accepted and 367 rejected cases. The current Markdown authority corpus has 61 paths at `sha256:8a2ffeb031c9073ae37163843cfe5e9fd8cf6b0dc0c280ff6839eed6a4bf0d5a`, with 945 positive-form rejections and 315 controls.

The phase table is `C4I → C4K → C5`.

The v23 table itself admits C5 after C4K; it does not encode the temporary operator checkpoint. C4J is not machine-declared in v23. Owner direction therefore parks C4J and blocks C5 operationally until C4K closes. After that boundary, C4J alone may restore its frozen payload, evolve v23 to v24, insert `C4K → C4J → C5`, and restart its ledger at command 1. This paragraph does not claim that machine authority enforces the parking decision.

## Exact C4K scope

C4K owns exactly these 11 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/THREAT_MODEL.md`, `docs/VERIFICATION.md`, `docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md`, `docs/status/P07B-C-C4K-C3-GO-TIMEOUT-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-architecture-selftest.mjs`, `tools/check-p07b-c-architecture.mjs`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Their sorted-newline roster digest is `sha256:9ab25f722aeaae583d08df271d6162c873254c51799b36b03fc85ff64d78567e`.

The source diff must contain one added status file and ten modified files, all mode `100644`, with no prefix-owned paths.

## Intended C4K claim map

| Claim | Type | Intended grade |
| --- | --- | --- |
| `P07B-C C4K candidate phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4K independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4K plan checker defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4K sealed-C4I Git-note and C4H C4 C4P C4N C4M C4V C3D ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4K unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4K historical C3 architecture compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4K repaired C3 architecture defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4K cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4K cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C4K exact eleven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C4K scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C4K sealed-C4I predecessor C4H C4 C4P C4N C4M C4V C3D ancestry and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Exact final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C4K`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C4K`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
4. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4k-sealed-c4i-note`
5. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
6. `/opt/homebrew/bin/node tools/check-p07b-c-architecture.mjs --c3`
7. `/opt/homebrew/bin/node tools/check-p07b-c-architecture-selftest.mjs --c3`
8. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
9. `/opt/homebrew/bin/node tools/verify-current.mjs`
10. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4K --source-final-gate`
11. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4K --credential-scan`
12. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4k-preseal-ledger`

The final root is `.countershape/p07bc-c4k-final`. The exact staged tree must exist before command 1. Every isolated shell fence generated for C4K sets `umask 077` before any didrun, claim, commit, seal, note, strict, HTML, or archive command; no persistent outer shell is assumed. Any nonzero command, commit/seal error, nonzero strict grade, or post-stage drift permanently fails that attempt; the failed ledger remains history and a repaired tree starts fresh evidence at command 1.

## Nonclaims and continuation to C4J

C4K establishes no product behavior, confidentiality, hostile same-UID resistance, HTTP/scope implementation, production-readiness, adoption, or maintainership claim. This source declaration does not assert a C4K commit, Git note, strict result, HTML report, or receipt grade. C4J remains parked until a future independently receipted C4K boundary permits its exact checkpoint to be restored and revalidated.
