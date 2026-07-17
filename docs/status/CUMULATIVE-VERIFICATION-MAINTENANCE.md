# Cumulative-verification maintenance boundary

This standalone unit runs after sealed P07B-A2.1 and before P07B-A2.2. It changes test setup, verification orchestration, documentation, and Finder-artifact hygiene only. It does not change the truth kernel, store policy, ruling authority, compilation authority, or any A2.2 compiler behavior.

## Finding rulings

### 1. Stock macOS `TMPDIR` — defect in tests, not store policy

The locked store and Darwin publication policy intentionally reject symlinked ancestors. macOS's default temporary alias contains `/var -> /private/var`, while several tests passed raw `t.TempDir()` paths into those strict boundaries. There was no wrapper-only ruling, and sibling tests already canonicalized their caller-owned root.

The repair is test-only: affected store, reduction, Darwin publisher, CLI study, and HTTP study helpers call `filepath.EvalSymlinks` after `t.TempDir()` creates the directory, require a clean absolute result, and pass that canonical caller-owned root onward. Negative symlink tests now begin below a canonical parent so they reach the deliberately inserted malicious edge. Production `ensurePrivateRoot`, `OpenObjectStore`, reduction-store publication, and no-follow rename behavior are unchanged.

Route: **defect**. Acceptance requires two consecutive complete repository suites with the login shell's stock `TMPDIR` untouched.

### 2. U3/U4 architecture chain — intentional historical boundary, defective legibility

U4 says its exact all-file U3 aggregates are sealed and that a future unit must replace the contract rather than grow exceptions. The boundary then evolved in two recorded units: U5 intentionally added typed CLI/HTTP neighbor providers that depend inward on `internal/reduce`, and P07A later added adapter-backed Choice test fixtures while U6 established the replacement production-only import lattice. The current U6 checker makes `internal/projectiontranslate` the sole production generic-to-adapter branch allowed to import the full CLI/HTTP adapter packages and allows `_test.go` fixtures without granting production Choice code adapter authority. The U3/U4 failure on those later sources is therefore expected historical-tree drift, not a present production-boundary violation.

The current chain is `P07B-A2.1 -> P07B-A1 -> U6/P07A-B -> U5`. U1–U4 architecture gates are historical-only and never silently invoked by the current baseline. The runner names both groups, runs every current checker and current hostile self-test, and prints each historical row as `HISTORICAL-ONLY NOT-RUN`.

Route: **intentional**, with a **defect** in orchestration/legibility repaired additively. U3/U4 were not weakened, rewritten, or incorrectly chained into a tree they do not own.

### 3. Finder artifacts and frozen mutation manifests — cleanup plus intentional historical seals

The repository already ignored `.DS_Store`; six ignored Finder artifacts had nevertheless accumulated and were deleted. The current baseline now refuses their return outside private evidence/cache roots.

Mutation drivers intentionally bind a frozen positive manifest and fixed mutant roster to their latest sealed owner. Retrofitting every old manifest to today's larger tree would create a new mutation program and weaken exact historical meaning. All existing rosters remain byte-for-byte historical authority. The top-level runner lists U1, the U4-receipted U2/U3 gates, U4, U5, the U6/P07A-B gate, P07B-A1, and P07B-A2.1 mutation gates as historical-only and does not spawn them. Its self-test fails if any current `mutate-*`, `test-mutate-*`, or `check-*` script becomes orphaned from an explicit classification.

Route: `.DS_Store` cleanup is **unconditional**; mutation behavior is **intentional** and is now explicit.

### 4. One current entry point and environment contract — defect

`tools/verify-current.mjs` is the single fresh-shell command. It admits and fingerprints fixed absolute tool authorities, exposes only reviewed aliases through a private authority bin, constructs a sparse hermetic child environment, runs build/vet/full-test, the runner self-test, every current architecture checker/self-test, requires exact checker success markers, frames child output, and revalidates every tool before the final result. `docs/VERIFICATION.md` is the operator contract. The hostile runner self-test pins both rosters and the exact environment, stops on nonzero/signal/spawn/throw/missing-marker failures, proves historical rows are never spawned, rejects Finder artifacts, ancestor-symlink plan escapes, misleading Node authority, tool replacement, and symlink-invocation main skips, and prevents silent architecture/mutation gate orphaning.

Route: **defect**.

## Negative evidence retained

The complete mixed development ledger is preserved unchanged at `.didrun-history/2026-07-16-cumulative-maintenance-development/.didrun/`. It supports no sealed capability because later critic-driven runner fixes superseded its source trees. Its negative events remain useful debugging history:

- event `0`: the complete stock-`TMPDIR` Go suite exposed the unresolved test roots and Darwin no-follow publication aliases;
- event `1`: U3 rejected later adapter-backed test fixtures and later CLI ownership, confirming that its sealed all-file contract is historical; and
- event `2`: the U1 mutation self-test stopped on a current-tree manifest artifact, confirming that the frozen U1 manifest is not cumulative.

Events `3`, `4`, and `8` respectively record an incomplete hand-built A2 checker environment, the first verifier pass refusing the two deliberately changed exact test AST seals, and a managed-sandbox denial of the ambient Go build cache. Events `5`, `6`, `7`, `9`, `10`, and `11` have honest development claims for the then-current trees, including the second cumulative pass, but the ledger was archived rather than sealed after the independent runner critic found fail-closed defects. Those development claims are not copied into the final receipt table.

A second one-event smoke ledger is preserved at `.didrun-history/2026-07-16-cumulative-maintenance-critic-smoke/.didrun/`. Its claimed syntax/self-test pass confirmed the first critic repair, then a final self-test-hardening edit superseded that tree; it too was archived unsealed and supports no final capability.

No failed event is relabeled as a catch. Only successful events from the fresh final-tree ledger receive the labels below.

## Final receipt labels

The sealed manifest is authoritative. Each row below must display the verbatim grade shown before this unit may open A2.2.

| Claimed capability | didrun claim label | Verbatim grade |
| --- | --- | --- |
| Go formatting and Node verifier syntax | `maintenance formatting and verifier syntax` | `TREE-EXACT` |
| Canonical test roots plus named no-follow negative controls | `maintenance canonical temp roots and no-follow controls` | `TREE-EXACT` |
| Verification-runner hostile self-test | `maintenance verification runner hostile self-test` | `TREE-EXACT` |
| Complete stock-`TMPDIR` suite, first consecutive pass | `maintenance stock macOS TMPDIR full suite pass one` | `TREE-EXACT` |
| Complete stock-`TMPDIR` suite, second consecutive pass | `maintenance stock macOS TMPDIR full suite pass two` | `TREE-EXACT` |
| One-command cumulative build/vet/test/current-architecture baseline | `maintenance one-command cumulative verification baseline` | `TREE-EXACT` |
| Exact staged diff and maintenance inventory | `maintenance exact staged inventory and diff` | `TREE-EXACT` |
| Scoped staged structured credential-pattern scan | `maintenance scoped staged credential-pattern scan` | `TREE-EXACT` |
| Serialized didrun chain integrity | `maintenance didrun chain intact` | `TREE-EXACT` |

## Limits and next boundary

This unit does not claim that historical gates pass on today's tree, mutation completeness, cross-platform behavior, hostile same-user isolation, secret absence, production readiness, security review, adoption, or maintainership. The cumulative runner is an operator baseline, not a receipt system; only its didrun-wrapped event plus the sealed strict manifest has a grade.

After this unit's commit is sealed and `NO_COLOR=1 didrun verify --strict` exits `0`, the sole next feature boundary is P07B-A2.2 in `docs/prompts/P07B-A2-2-RECOVERABLE-COMPILER.md` under the constraints in `docs/prompts/P07B-A2-COMPILATION-PLAN.md`.
