# Current verification baseline

From a fresh shell at the repository root, run:

```sh
/opt/homebrew/bin/node tools/verify-current.mjs
```

This is the single cumulative baseline for the current Darwin/arm64 reference tree. It exits nonzero on the first failed current step and prints `RESULT PASS` only after all current steps pass. In an evidence-producing unit, wrap that exact command with `didrun run --`, claim the successful event immediately, and commit. In this managed environment, seal only with Git-note write authority, confirm `git notes --ref=didrun show HEAD` succeeds, and then require `NO_COLOR=1 didrun verify --strict` to exit `0`. A direct run is useful local evidence, but it has no didrun grade.

## What the command runs

The runner executes, in order:

1. a repository guard that rejects any `.DS_Store` outside private evidence/cache directories;
2. `go build`, `go vet`, and the complete Go suite;
3. the verification-runner hostile self-test;
4. the U5, U6/P07A-B, and P07B-A1 architecture checkers and hostile self-tests;
5. exact regeneration checks for the runtime bundle and planning examples plus the planning-validator hostile self-test, including real conforming execution, pre-import companion-tamper refusal, and eight byte-exact schema-valid/runtime-invalid cases crossed through the Go semantic parser;
6. fresh-process compiler-to-parser recovery;
7. the deterministic human-surface renderer self-test and exact six-scenario capture check;
8. the P07B-A2.2 layered architecture checker and its 117-case hostile self-test;
9. the P07B-B terminal-publication/materialization architecture checker and its defensive self-test;
10. the P07B-C C1 inert-model architecture checker and self-test—including exact production/test files, imports, top-level test/fuzz symbols, required rosters, and Go-JSON positive-execution parsing—the evolved C0-receipt/C1-projection plan self-test, and the machine-readable staged-unit-scope self-test; and
11. final revalidation of every admitted tool authority.

The current architecture chain is nested fail-closed in two directions from P07B-B: its sealed predecessor chain invokes P07B-A2.2, then P07B-A1, U6, and U5; its current compatibility edge invokes the complete C1 inert-model checker. Composition is one-way (`B -> C1`): C1 never invokes B, so there is no recursion. B and C1 retain explicit top-level rows and self-tests even though B also inherits C1 transitively. The evolved A2.2 checker retains the sealed A2.1 authority closure internally; there is no duplicate top-level A2.1 row pointing at the same file.

U1–U4 architecture checkers are `HISTORICAL-ONLY NOT-RUN`. They freeze the topology and, by U4, exact all-file aggregates of their sealed unit trees. The boundary evolved in two explicit steps: U5 moved typed adapter neighbor providers onto the pure reducer, then P07A introduced adapter-backed Choice test fixtures and U6's production-only import lattice. Under the current lattice, `internal/projectiontranslate` is the sole production generic-to-adapter branch allowed to import the full CLI/HTTP adapter packages, tests may reconstruct historical projections, and production Choice code still cannot import adapters. Running U3/U4 against today's larger tree would test a contract they explicitly do not own.

All mutation gates are also `HISTORICAL-ONLY NOT-RUN`. Each current script snapshot has a positive manifest and mutant roster bound to its latest sealed owner. Some were deliberately expanded and re-receipted by later units: U2/U3 at U4, and U6 through P07A/P07A-B. The current unit rewrites and shrinks none of them. The baseline prints every historical architecture and mutation row, but never spawns one, and no historical row is represented as passed on the current tree.

The proposed A2.2 source-rewrite driver never became a checked-in executable, completed run, claim, or seal. It therefore has no current or historical row and remains `UNRECEIPTED` design history. A2.2's replacement evidence is independently scoped across black-box/property/parity/recovery/fuzz/boundary checks and the hostile architecture self-test; the two bounded fuzz targets are separate final-unit didrun receipts and are not silently executed by this cumulative command. The human-surface rows verify deterministic rendered/captured bytes and normalized runtime records; they do not establish universal comprehension, semantic correctness, or a didrun grade for manual or different-model taste review.

The P07B-B defensive self-test likewise uses one clean canonical execution plus cloned metadata/topology negatives. It now also requires the clean C1 checker, partitions the exact `internal/contractexec/model/` prefix from lookalikes, and refuses every foreign C surface. It does not generate or rewrite production source copies into a recipe corpus and makes no mutation-completeness claim.

P07B-B source commit `eb06bdcf18f8e14db1257e73733afdc05cac045e`, tree `d95c907754f3347bae885a64c98b30ad1ea00256`, is sealed and strict-clean with `16/16` claims recorded-exact. Final B handoff commit `464e47adbf7f4497dfafa89938a9539239ffd41b`, tree `c7e80d6e6819d4e2c54aaedeb4dc3ef730de58c6`, independently verifies `5/5 TREE-EXACT` with its matching ledger. C0a commit `1c6d9fdef339314bfcb99d7d53f3a7c5040e5020` and C0b commit `47044e95f405b7252959415adb1aa0dbea8045ab` remain historical sealed authority. Narrow cumulative maintenance source `fa3d0c12b4c599744b666b2848e38a2499f33a89`, tree `c908c4e580144aa481f646090a1a21d92c86e1cf`, is independently sealed and strict-clean with `8/8 TREE-EXACT`; it removes a scheduler race from the stdout-overflow test premise by completing the exact sibling-channel fixture write before crossing the output cap, without changing capture or teardown behavior. Its one-file receipt reconciliation `3f31370a70979c4c5fe3a523be19e05f84271e96`, tree `29c5708b05c562984a0b7b66d6cfe3ac6e19c626`, is separately strict-clean with `6/6 TREE-EXACT`. C1 adds strict inert semantics and defensive projection gates only; every C1 row remains `UNRECEIPTED` until its exact final commit/seal/strict boundary, and it adds no store, Git, runtime, process, or publication authority.

The evolved plan self-test preserves exactly two historical C0 receipt phases while adding the C1 inert projection. C0a requires all eight status rows to remain `UNRECEIPTED` while `spec/verification/p07b-c-c0-receipt.json` is absent. C0b requires that file to contain the exact C0a source commit/tree, strict `8/8` counts, ordered claim roster, and verbatim `TREE-EXACT` grades. Through its declared admitted Git authority, the checker requires the source commit to be a real ancestor, recomputes its tree, parses its didrun Git note, and matches all eight successful tree-exact note claims and event coverage; status and a delimited handoff table must match every field. C1 additionally pins the three closed schema/object projections, their required rosters, canonical example graph, exact source receipt noninflation, and unit allowlist. Hostile self-tests cover wrong C0 commit/tree/grade/handoff and C1 authority, schema, documentation, and scope drift. Neither phase logic nor C1 projection checking implies physical capability.

## Hermetic child environment

The runner accepts no arguments, root override, skip flag, or historical-run mode. It creates a private mode-`0700` run root below ignored `.countershape/verify-current/`, then gives children fresh `HOME`, `TMPDIR`, `GOTMPDIR`, `GOCACHE`, `GOPATH`, and `GOMODCACHE` directories. It sets `GOENV=off`, `GOWORK=off`, `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`, `GOVCS=*:off`, `GOFLAGS=-mod=readonly -buildvcs=false -p=1`, `CGO_ENABLED=1`, `GOMAXPROCS=2`, fixed locale/timezone/color values, and a sparse `PATH`. Direct Go build/vet/test rows also carry `-p=1`, so nested and direct package parallelism obey the same resource ceiling. Ambient `GOFLAGS`, Node preload paths, proxy variables, and credentials are not copied.

The locked Darwin defaults are:

- `COUNTERSHAPE_GO=/opt/homebrew/bin/go`
- Node authority equal to the canonical `process.execPath` already running the verifier
- `COUNTERSHAPE_GIT=/usr/bin/git`
- `COUNTERSHAPE_SH=/bin/sh`
- `COUNTERSHAPE_CC=/usr/bin/clang`
- `COUNTERSHAPE_CXX=/usr/bin/clang++`

An explicit Go/Git/shell/C/C++ override must be an absolute path. Node is not overridable after startup: invoke the verifier with the desired Node executable; a conflicting `COUNTERSHAPE_NODE` fails closed. Every admitted tool alias is resolved to a regular executable, opened without following the final component, checked for replacement during each admission/fingerprint read, SHA-256 fingerprinted, and printed as an `AUTHORITY` row. Every executable roster row declares its complete admitted tool set, including tools reached by a nested checker, and the verifier re-opens and re-fingerprints each declared authority immediately before and after the row's direct spawn. All authorities are revalidated again before the final result. A private `authority-bin` exposes only the admitted `go`, `node`, `git`, `sh`, `cc`, and `c++` aliases ahead of `/usr/bin:/bin`; admitting one tool does not add its sibling directory to `PATH`.

Each JavaScript checker must also emit its exact success marker; zero exit without that marker is failure. Child output is JSON-framed under `CHILD` rows so it cannot counterfeit the verifier's reserved `CURRENT` or `RESULT` records. These controls reduce accidental/path-replacement ambiguity but do not establish hostile same-user isolation or eliminate the residual same-user check-to-exec race between the last pre-spawn fingerprint and a nested process launch.

The cumulative command uses its resolved private `TMPDIR` for deterministic operation. Separately, the repository's tests must also pass twice under the login shell's stock macOS `TMPDIR`; test-only helpers canonicalize `t.TempDir()` before it enters a strict object-store or Darwin no-follow publication boundary. Production store/publication code still rejects symlinked ancestors and never canonicalizes a caller's root for them.

## Output contract

The stable output classes are:

```text
COUNTERSHAPE_VERIFY_V1 ...
AUTHORITY ...
ROSTER CURRENT ...
HISTORICAL-ONLY NOT-RUN ...
CURRENT [NN/NN] START ...
CHILD ...
CURRENT [NN/NN] PASS ...
RESULT PASS current=N/N historical_not_run=M
```

On failure, the failing row is `CURRENT ... FAIL`, no later current row is spawned, and the summary is `RESULT FAIL`. `HISTORICAL-ONLY NOT-RUN` means exactly that; it is not a current-tree compatibility claim.
