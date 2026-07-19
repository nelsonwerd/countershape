# P07B-C verification-throughput maintenance ruling

**Boundary state:** C1V is a source-bearing verifier-maintenance unit between the sealed C1M prerequisite and the data-only C1B receipt reconciliation. It changes no Countershape product semantics. It remains ungraded in this tracked document; its commit, Git note, didrun grades, and strict result must be inspected independently after sealing.

## Measured before-state

The sealed C1M final ledger recorded two clean cold cumulative runs on the same staged tree at `976161 ms` and `992765 ms`. The latter spent `632845 ms` in the former single `go-test-full` row. Its largest package times were `155794 ms` for the CLI physical study, `125103 ms` for the HTTP physical study, `121919 ms` for Node parity, `91226 ms` for `internal/world`, and `36848 ms` for the generated-contract compiler/runtime package. These are local Darwin/arm64 observations, not portable performance guarantees.

## Proposal routing

| Item | Ruling | Reason |
| --- | --- | --- |
| Persistent Go-only build cache | `DECLINED_WITH_REASON` | Admitted Go 1.26.5 states that its build cache does not detect changes to C libraries. This tree has real Darwin/cgo publication code. A namespace keyed only by the Go executable digest omits C/C++ authorities, SDK/libSystem state, and accidental cross-run cache mutation, so “results cannot differ” is false here. |
| Tiered parallelism | `DEFECT_CONDITIONAL_REPAIR` | Global package serialization unnecessarily covered pure build/vet and the complete general package complement. Direct general work may use two package jobs only while the frozen sensitive packages and every nested tool retain the historical serial environment. Acceptance requires the stability and benchmark gates below. |
| Single-verifier exclusion | `DEFECT_REPAIR` | Two cumulative runners can contend and recreate the load conditions that the serial test profile is intended to avoid. A private fail-fast ownership lock is part of the verifier contract. |
| Receipt-only boundaries | `DEFECT_NARROW_REPAIR` | An exact data-only receipt reconciliation should prove its source note/declaration/status/handoff agreement and boundary integrity, not claim unrelated runtime or full-suite behavior. Eligibility is explicit in the unit specification and is never inferred from filename extensions or a diff heuristic. |

## Cache ruling

`GOCACHE` remains fresh below each mode-`0700` `run-*` root and is deleted with that run root. `HOME`, `TMPDIR`, `GOTMPDIR`, `GOPATH`, `GOMODCACHE`, and the admitted-authority bin remain fresh as well. No tracked claim describes compilation as warm, cached, or portable. A future persistent-cache proposal must bind a broader compile-host authority and carry an independently fresh cgo lane; C1V does neither and therefore preserves the cold-cache evidence boundary.

## Closed resource profiles

The ambient child profile remains `GOFLAGS=-mod=readonly -buildvcs=false -p=1`, `GOMAXPROCS=2`, offline module resolution, fixed locale/timezone/color, and the sparse admitted-authority `PATH`. This keeps every nested Go command serial.

Only the verifier's direct `go build`, `go vet`, and general-package `go test` commands override package jobs to `-p=2`. The test command retains `-count=1`, `-timeout=20m`, and explicit `-parallel=2`. Before either test tier runs, admitted `go list ./...` is parsed as a strict, unique package roster. Its exact complement is tested once in the general tier; these six whole packages are removed from that tier and tested once afterward with `-p=1`, `GOMAXPROCS=2`, and `-parallel=2`:

- `github.com/nelsonwerd/countershape/internal/emit/node/compiler`
- `github.com/nelsonwerd/countershape/internal/emit/node/program/v1`
- `github.com/nelsonwerd/countershape/internal/store`
- `github.com/nelsonwerd/countershape/internal/world`
- `github.com/nelsonwerd/countershape/testkit/studies/cli_precedence`
- `github.com/nelsonwerd/countershape/testkit/studies/http_invoices`

Malformed, duplicated, foreign, or missing frozen-sensitive package rows fail before tests. Every other valid package under the admitted module path belongs to the dynamic general complement, so a package already present at the first listing is exercised rather than rejected as “unclassified.” The admitted package closure is listed again after both lanes and must equal the original `all`, `general`, and `sensitive` sets byte-for-byte; any addition, removal, or reclassification during the run fails before later proof rows. `internal/emit/node/parity` remains in the general tier only if its repeated Node framing/evaluator canaries and three complete cumulative passes stay clean.

The first `-p=4`, `GOMAXPROCS=2` candidate is rejected. Its two unchanged-tree cumulative development runs passed every Go package but later failed the A2.2 hostile architecture gate when the JavaScript AST subprocess reached its unchanged 60-second deadline on two different mutants. The same 77-case gate passed in an isolated faithful verifier environment, so this is load sensitivity rather than one invalid mutant; it is still disqualifying stability evidence. Those failed didrun events remain permanent. C1V does not lengthen that deadline, retry the failed row, or grade either run. `-p=8` is also declined for this host-sharing session. The remaining `-p=2` candidate aligns package jobs with `GOMAXPROCS=2` and leaves deliberate headroom for the user's separate workload; any recurrence rejects tiering entirely. Its first pre-finalization development pass completed all 30 then-current rows in `859342 ms`, versus the sealed cold C1M observations of `976161 ms` and `992765 ms`; that is a local provisional improvement of `116819–133423 ms` (`11.97–13.44%`), not a qualifying final receipt.

Tiering is accepted only if all focused 50x/20x lifecycle/output/readiness/CAS/runtime receipts pass, both physical study packages pass three complete serialized repetitions, Node parity passes its repeated canaries, and three complete cumulative runs pass. Every one of those three candidate runs must complete in at most `878544 ms`: against the faster sealed C1M baseline (`976161 ms`), that is at least `97617 ms` and at least 10 percent faster, so it also exceeds the 60-second floor. No median, best-run selection, or comparison only to the slower baseline is allowed. Any Darwin teardown, readiness, pipe, EPIPE, CAS, Node deadline, loopback, partition, coverage, or duration failure reverts the candidate without weakened assertions or longer deadlines.

The focused Go repetition runner makes those positive receipts closed and machine-auditable. It accepts one full admitted-module package, enforces that package's declared `general` or `sensitive` classification, accepts a count from 1 through 50, an anchored test regex, and the exact expected top-level Test/Fuzz/Example roster. It holds the same single-verifier lock, creates the same fresh private roots, preserves the 20-minute command timeout, parses `go test -json`, and requires a well-ordered package/test lifecycle plus the exact run/pass count for every expected entrypoint while rejecting skips, failures, foreign packages, and foreign entrypoints. A full-package run must enumerate ordinary tests, fuzz-seed entrypoints, and examples. Cleanup and lock release precede its success marker. Its hostile self-test is cumulative; each real focused run remains a distinct didrun event.

The helper exports the exact qualification matrix below: 52 ordered named executions across 14 families. Its canonical JSON digest is `sha256:bf5803a752ffe6c2833e7869086ec6a4919e15c5e49e8cd012a5c8c16879fc37`; the plan checker pins both that digest and every ordered case ID. Only `--case <id>` uses these exact package/profile/count/regex/expected-entrypoint bytes and emits `qualification=true`. Explicit package/regex mode remains useful for development but emits `qualification=false` and cannot satisfy the aggregate stability gate. Every real receipt first emits the ordered admitted Go/Node/Git/sh/CC/CXX paths and SHA-256 rows, then binds their aggregate digest into the terminal success record.

| Qualification case | Package | Profile / count | Closed coverage |
| --- | --- | --- | --- |
| `world-output-caps-50` | `internal/world` | sensitive / 50 | exact independent stdout/stderr cap test |
| `world-output-independence-20` | `internal/world` | sensitive / 20 | cap-independence mutation guard |
| `world-simultaneous-overflow-20` | `internal/world` | sensitive / 20 | simultaneous-channel overflow facts |
| `world-lifecycle-readiness-20` | `internal/world` | sensitive / 20 | exact nine-test process, teardown, readiness, and late-EOF roster |
| `compiler-generated-runtime-20` | `internal/emit/node/compiler` | sensitive / 20 | exact six-test generated CLI/HTTP runtime roster |
| `program-lifecycle-20` | `internal/emit/node/program/v1` | sensitive / 20 | copied harness lifecycle state machines |
| `store-cross-process-cas-20` | `internal/store` | sensitive / 20 | cross-process study-head CAS winner |
| `cli-physical-reducer-01-of-20` … `cli-physical-reducer-20-of-20` | `testkit/studies/cli_precedence` | sensitive / 1 each × 20 | twenty separately locked fresh-root physical CLI reducer receipts |
| `http-physical-reducer-01-of-20` … `http-physical-reducer-20-of-20` | `testkit/studies/http_invoices` | sensitive / 1 each × 20 | twenty separately locked fresh-root physical HTTP reducer receipts |
| `parity-evaluator-20` | `internal/emit/node/parity` | general / 20 | Go/Node literal-oracle evaluator |
| `parity-framing-20` | `internal/emit/node/parity` | general / 20 | exact five-test framing/cap roster |
| `parity-full-package-3` | `internal/emit/node/parity` | general / 3 | all 16 ordinary tests plus the fuzz-seed entrypoint |
| `cli-physical-full-package-3` | `testkit/studies/cli_precedence` | sensitive / 3 | exact complete 12-test package roster |
| `http-physical-full-package-3` | `testkit/studies/http_invoices` | sensitive / 3 | exact complete 13-test package roster |

### Failed aggregate physical receipt and bounded repair

The first frozen final-ledger attempt passed its helper gate and the first seven named families, then the old aggregate `cli-physical-reducer-20` process reached the unchanged 20-minute child deadline. didrun event 8 is permanently failed with `ETIMEDOUT`, `SIGTERM`, and no Go test failure; its retained stdout tail shows a `166.32s` iteration pass followed by the next iteration's run event. That ledger remains intact at `.didrun-history/2026-07-18-p07b-c-c1v-final-attempt-1-physical-aggregate-timeout/.didrun/`.

This was a receipt-shape defect, not evidence that p2 general-package work destabilized a sensitive lane: the helper was already running the physical package at `-p=1`. C1V does not increase the 20-minute deadline, reuse the partial event, or reduce the required 20 repetitions. It replaces each physical reducer aggregate with twenty canonical count-1 cases, each with its own lock, admitted authorities, fresh private roots/cache, exact JSON lifecycle proof, cleanup, and didrun event. HTTP uses the same shape before encountering the equivalent avoidable aggregate deadline.

Qualification starts only after independent code/docs review has closed every tracked edit. The exact eleven C1V paths are then staged, this development ledger is archived intact, and one fresh final ledger records the helper gates, every focused repetition, all three cumulative runs, exact staged-scope/diff integrity, the scoped named-pattern credential scan, and preceding didrun-chain integrity against that unchanged snapshot. A scope event from the archived development ledger is admission evidence only. Any tracked edit after the freeze invalidates the whole qualifying matrix; the superseded ledger is archived and the matrix restarts from event zero. Only that unchanged snapshot and its fresh-ledger boundary events may be claimed, committed, sealed, note-checked, and strictly verified.

## Fail-fast verifier ownership

The runner acquires `.countershape/verify-current/active.lock` before plan validation, tool admission, per-run-root creation, build, or test work. The private base necessarily exists before its lock can be created. The mode-`0600` regular file contains a bounded canonical schema, PID, random nonce, creation time, and repository-root digest. Exclusive creation never queues. A live or indeterminate owner fails with `VERIFY_ALREADY_RUNNING`; an absent owner fails with `VERIFY_STALE_LOCK` and a review-first recovery instruction.

C1V does not automatically delete stale O_EXCL locks. Two stale-recovery contenders can otherwise race and one can remove the other's newly acquired lock. Acquisition-error cleanup unlinks only after re-proving that the path still names the descriptor-created inode. Normal checks consume the exact bounded file, compare before/after descriptor and path metadata, and require its canonical content; replacement, disappearance, content drift, or mode drift fails with `VERIFY_LOCK_INTEGRITY` and does not remove the uncertain path. PID reuse may conservatively cause a false busy result, which is an availability cost rather than concurrent evidence. Automatic crash recovery would require a separately admitted kernel-lock authority and is outside this narrow repair.

The ignored `.countershape` parent is validated as an owned, non-symlink, non-group/world-writable directory before use. The private verifier base is validated as an owned, canonical, mode-`0700` directory before any chmod or child creation, removing the previous validate-after-chmod symlink side effect.

## Machine-declared verification profiles

`spec/verification/p07b-c-unit-paths.json` v2 assigns every unit one closed profile:

- `SOURCE_FULL` for source, verifier, schema, generated-artifact, or runtime units. These units retain the cumulative baseline.
- `RECEIPT_RECONCILIATION` only for exact empty-prefix Markdown/receipt-declaration rosters. This profile cannot support a cumulative, full-suite, runtime, security, or unchanged-behavior claim.

Historical receipts are not relabeled. C0B remains machine-declared `SOURCE_FULL` because that is the larger battery it actually ran. Future C1B and C6B may use the narrow profile. Each narrow entry carries an ordered seven-claim label/type manifest; `check-p07b-c-unit-scope.mjs --unit <unit> --receipt-manifest` prints that exact authority. The cumulative verifier does not auto-select the narrow profile: the unit owner must run and claim the declared roster explicitly through didrun. Both staged modes additionally require every receipt path to exist in the index as exactly one stage-zero regular mode-`100644` nonzero blob, rejecting deletion, symlink, executable, submodule, intent-to-add, and unmerged entries. C1B's four paths and sorted-newline digest remain exactly unchanged: `docs/HANDOFF_MODE_C.md`, `docs/status/DIDRUN_BUGS.md`, `docs/status/P07B-C-C1-SEMANTICS.md`, and `spec/verification/p07b-c-c1-receipt.json`; digest `sha256:f8d1fb96d36f7e0cfde99bb73c7726297763c66bafd21aeb5c1fb1e249bdd119`.

The C1B receipt-only roster may claim only source receipt reconciliation, defensive receipt-checker self-test, declared local source-evidence snapshot match, receipt-only Go build, exact four-path staged scope/diff integrity, scoped named credential-pattern scan, and preceding didrun-chain integrity. Its pre-seal tracked text cannot present C1B as already graded; only the later didrun/seal result can do that. Any path, mode, manifest, or profile drift fails closed; it does not silently widen or downgrade the gate.

## Exact C1V scope

C1V owns exactly these 11 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/THREAT_MODEL.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-VERIFICATION-THROUGHPUT.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, `tools/check-p07b-c-unit-scope.mjs`, `tools/verify-current-selftest.mjs`, `tools/verify-current.mjs`, and `tools/verify-go-test-repetition.mjs`. Their sorted-newline roster digest is `sha256:6032cc515978947743e1bfea72340e1065d77473ccc27b3b0fc036ebd4ceb2a5`.

This declaration does not receipt itself. C1V must complete its full didrun build loop, commit, seal, note check, and strict verification before any C1B receipt declaration, receipt block, or reconciliation claim is introduced. `docs/HANDOFF_MODE_C.md` intentionally belongs to both unit rosters and may carry only C1V state during this boundary.
