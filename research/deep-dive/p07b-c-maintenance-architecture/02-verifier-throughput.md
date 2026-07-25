# Verifier throughput and stability

## Ruling

The current verifier is slow but intentionally conservative. No cache or
parallelism change belongs in C4H.

| Proposal | Route |
| --- | --- |
| Persistent GOCACHE keyed only by Go | `DECLINED_WITH_REASON` |
| Broader persistent cache in this chain | `DEFERRED_WITH_REASON` |
| p8 | `DECLINED_WITH_REASON` |
| Restore p2 without current-tree requalification | `DECLINED_WITH_REASON` |
| Current p1 and nine-package sensitive roster | `INTENTIONAL` |
| Fail-fast single-verifier lock | `INTENTIONAL / ALREADY_SATISFIED` |
| Narrow receipt-only profile | `INTENTIONAL / ALREADY_SATISFIED` |
| Three final cumulative passes | `INTENTIONAL` |
| Historical baseline provenance wording | `DEFECT` |
| Real subprocess lock receipt | `DEFECT_IN_TEST_COVERAGE` |

This was a read-only review. No verifier, didrun, tests, seals, or mutating Git
commands were run.

## Current execution profile

The live cumulative verifier uses package p1 for direct build, vet, general
tests, sensitive tests, and inherited nested Go commands:

- `generalJobs = 1`;
- test `-parallel=2`;
- `GOMAXPROCS=2`;
- `GOFLAGS=-mod=readonly -buildvcs=false -p=1`;
- `CGO_ENABLED=1`.

See `tools/verify-current.mjs:20-64` and
`tools/verify-runtime-authority.mjs:475-505`.

“Sensitive serial” therefore means package-level p1, not globally
single-threaded execution. Packages may still use two parallel tests, internal
goroutines, or child processes.

The sensitive roster contains nine exact packages
(`tools/verify-current.mjs:38-48`):

1. `internal/contractexec/runner`
2. `internal/emit/node/compiler`
3. `internal/emit/node/program/v1`
4. `internal/processmechanics`
5. `internal/store`
6. `internal/world`
7. `testkit/contractexec/cli`
8. `testkit/studies/cli_precedence`
9. `testkit/studies/http_invoices`

The package partition is strongly checked. `go list` must be unique and
module-local; every sensitive package must exist; general is the exact dynamic
complement; both sets are disjoint; and a second list must reproduce the same
partition after testing (`tools/verify-current.mjs:349-396,484-502`).

## Parallelism history

Historical p4 passed both Go lanes and then caused two different later
JavaScript AST children to hit unchanged deadlines. This is direct evidence
that “all Go packages passed” is not enough; increased package concurrency can
alter ambient pressure seen by later Node work.

Historical p2 qualified on its old tree, but the present tree, package roster,
and verifier roster are materially larger. A later AST recurrence triggered the
predeclared rollback rule. C4V changed only direct/general jobs from two to one
while keeping assertions, timeouts, GOMAXPROCS, test parallelism, nested p1,
cache behavior, and order
(`docs/status/P07B-C-C4V-EXECUTION-CONTRACT-MAINTENANCE.md:22-28`).

Current C4 subsequently recorded three complete p1 cumulative passes:

- 5,252,318 ms (87.54 minutes)
- 4,791,163 ms (79.85 minutes)
- 4,728,284 ms (78.80 minutes)

All bind the same exact C4 tree and exited zero. These timings are local
observations, not portable guarantees.

Any future p2 proposal must re-audit current packages and rerun the complete
54-case qualification matrix, all 50x/20x and physical-shard cases, isolated
A2 hostiles, and three unchanged-tree cumulative p2 passes against three fresh
p1 baselines. Any timing, teardown, readiness, output, CAS, loopback, package
partition, or Node deadline failure rejects p2. p4/p8 remain out of scope until
p2 independently qualifies.

## Persistent cache

Every verifier invocation creates a fresh mode-0700 run root with distinct
GOCACHE, GOMODCACHE, GOPATH, temp, home, and authority-bin directories
(`tools/verify-runtime-authority.mjs:442-473`). Success removes the root before
releasing the lock.

A cache keyed only by the Go executable is unsound for this tree. Go's own
documentation states that its cache does not detect changes to C libraries
imported through cgo. Countershape has real cgo publication code. Hashing Go,
clang, and clang++ would still omit SDK headers, sysroot, libraries, linker
inputs, deployment target, helper files, and prior same-user writable-cache
mutation.

A broader immutable pure-Go seed plus a fresh cgo/reverse-dependent lane is
possible in principle, but it targets the wrong cost center. On current C4
receipts, build and vet together were about 0.4% of an average 82-minute pass.
The large rows are Go tests and repeated architecture/checker work. Persistent
cache research therefore remains a post-chain experiment.

## Lock

The current lock really is fail-fast. Acquisition occurs before plan
validation, tool admission, or private-root creation. It uses
`O_CREAT|O_EXCL|O_RDWR|O_NOFOLLOW`, mode 0600, bounded canonical content, repo
digest, and owner PID checks. A live or indeterminate PID returns
`VERIFY_ALREADY_RUNNING`; an absent PID returns `VERIFY_STALE_LOCK`
(`tools/verify-runtime-authority.mjs:144-211`).

Release and cleanup revalidate inode/path/content identity and preserve
uncertain replacements. PID reuse can false-busy and killed verification leaves
a review-required stale lock; these are intentional availability costs.

The self-test covers second acquisition, stale owner, different repository,
invalid content/mode, symlink, replacement, disappearance, same-inode drift,
and mode drift (`tools/verify-current-selftest.mjs:701-764`). The gap is that
both owners are calls inside one Node process. C4H should add a real child-
process contention receipt without changing production lock semantics.

## Receipt-only profile

`RECEIPT_RECONCILIATION` is already explicit and narrow. It permits an
empty-prefix exact roster of Markdown authority and/or named receipt
declarations, requires a frozen claim manifest, and refuses labels that imply
cumulative, suite, runtime, security, or unchanged-behavior evidence
(`tools/check-p07b-c-unit-scope.mjs:458-505`).

It is never inferred from filenames or a diff heuristic. Preserve it unchanged.

## Three cumulative passes

Focused cases and cumulative passes prove different things. Each isolated
qualification case gets its own lock, authority snapshot, private roots, Go
JSON lifecycle proof, cleanup, and marker. Full cumulative passes additionally
expose cross-row load, later Node/AST deadlines, ordering, and cleanup
accumulation. The p4 failures show why earlier green Go rows cannot substitute.

During development, focused cases and one rehearsal may be used diagnostically.
The final source/verifier ledger retains all three cumulative passes.

## Provenance erratum

`docs/status/P07B-C-VERIFICATION-THROUGHPUT.md:7` says the sealed C1M ledger
recorded both 976161 and 992765 ms. Raw history shows only 992765 ms was in the
sealed C1M final ledger; 976161 ms was a complete same-tree observation in a
later C1B attempt whose following event failed.

The faster observation still produced a stricter p2 threshold, so the
historical p2 acceptance is not invalidated. C4H should record the erratum in
new status prose without rewriting sealed history.

## Better future optimization target

The highest-leverage future target is repeated checker work: immutable
per-invocation source snapshots, content-addressed parse results, and removal of
redundant reparsing where reopening is not itself the asserted behavior. Any
such work must preserve mutation, race, source-reopen, and exact-byte hostiles.

Confidence: **9/10**. Ten of twelve load-bearing conclusions were verified from
current code, Git history, or local receipts; the remainder are design judgment.
