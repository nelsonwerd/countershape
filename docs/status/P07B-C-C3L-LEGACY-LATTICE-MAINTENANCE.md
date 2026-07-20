# P07B-C C3L — legacy-lattice maintenance

## State

- **Boundary:** active pre-seal `SOURCE_FULL` maintenance after sealed C3A and before the frozen C3 source unit. Classification: `DEFECT_REPAIR`.
- **Parent:** sealed C3A commit `2b84d2841971568784d2ac955775b4a99ca7f0f6`, tree `b8d858033431fe51c262ce725b9d9b051ba9e8c9`, is note-present with note blob `3e516946f4b86e536ae9ea0124938bd534573983`, note-body SHA-256 `07a0d04dc667963ea16ec6bb458b929d3a7659637e705bb642b5267766609a8c`, `11/11 claims recorded-exact`, and strict exit `0`.
- **Scope:** nine exact mode-`100644` paths, empty prefixes, sorted-newline digest `sha256:b3624e1337681e0909e6c074246103e066118a5924e8f790e7d1f42359da9dc0`.
- **Commit subject:** `fix: admit exact C3 store model edge through U6`
- **Preserved C3 work:** stash object `fa737661e61c36c10ce793f64852836edec20065` contains the post-C3A C3 implementation/checker checkpoint and remains retained without dropping. Earlier stash `3e766715609680adaef4678fc9aeaa9e3a8bae09` remains retained as a second historical checkpoint. Neither stash supplies a C3L grade.

Every intended C3L grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3L itself.

## Trigger and defect classification

The resumed C3 build restored the intended inert store bridge. Its `internal/store/nonhead_contract.go` imports `internal/contractexec/model` so `PersistContractTargetRecord` accepts the closed `ContractExecutionTarget` type rather than generic bytes, a digest bag, or generic CAS authority. The cumulative verifier runs U6 before the later P07B-B/C checks. U6 still froze the pre-C3 store import lattice, so the exact new edge failed as `U6_INTERNAL_IMPORT_LATTICE` before those later checks could evaluate the bridge.

That failure is valid evidence: the cumulative compatibility assumption in C3A was incomplete. Removing U6/B from cumulative verification, invoking only B's exported helpers, widening the store import prefix, or redesigning the bridge around generic bytes would weaken the architecture to preserve a stale brief. C3L instead repairs the owning legacy boundary at a separate, review-visible source unit. The C3 unit entry's forty-path roster remains byte-for-byte unchanged.

## Exact repair

U6 retains its historical store import set as the only admitted pre-C3 state. It admits the one additional exact import `internal/contractexec/model` only when all of the following are simultaneously true:

- a fail-closed top-level Go source-shape index decodes both interpreted and raw import literals, and the sole decoded production importer is exactly `internal/store/nonhead_contract.go`;
- the decoded import alias is exactly `contractmodel`; import-like text inside a literal is inert;
- `ConformanceAttemptInput` is one non-alias package-level struct owned by that exact file and has exactly the four ordered, untagged, single-name digest fields `ContractBundleDigest`, `ResidueHeadDigest`, `TreeIdentityDigest`, and `MaterializationPolicyDigest`;
- across the production `store` package, `ConformanceAttemptRoots`, `ConformanceAttemptRecord`, and `ContractTargetRecord` are each declared exactly once as package-level structs in that same file; and
- the four `ObjectStore` methods `AllocateConformanceAttempt`, `OpenConformanceAttempt`, `PersistContractTargetRecord`, and `OpenContractTargetRecord` are each owned exactly once by that file and carry the expected typed parameter/result sequences, including `contractmodel.ContractExecutionTarget` at persistence. Receiver and parameter identifier spelling is not authority-bearing.

The conditional state is fail-closed. An incomplete bridge, tagged/hidden/extra field, signature or result drift, wrong alias, raw or escaped second model importer, local-shadow decoy, foreign owner, type alias, or duplicate method fails with `U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED`; all other unexpected imports still fail the exact U6 lattice. C3's own architecture and compiler-AST public-surface tests remain responsible for the deeper bridge semantics. C3L is compatibility admission, not proof that the future bridge implementation works.

The U6 defensive self-test now checksum-pins 147 cases: seven clean/comment controls and 140 hostile cases. Its C3-specific slice has four clean controls (ordinary bridge, raw-literal bridge, renamed receiver/parameters, and raw-string camouflage) plus seventeen hostile controls covering absent/incomplete shape, wrong aliases, parameter/result/field/tag/order drift, hidden or exported fifth fields, decoded raw/escaped foreign imports, local shadows, foreign aliases, duplicate top-level types, and duplicate methods. The historical clean tree still passes, so C3L proves both sides of the phase boundary rather than making the future import mandatory before C3 returns.

## Exact C3L scope

C3L owns exactly these 9 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C3L-LEGACY-LATTICE-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, `tools/check-p07b-c-unit-scope.mjs`, `tools/check-u6-architecture-selftest.mjs`, and `tools/check-u6-architecture.mjs`. Their sorted-newline roster digest is `sha256:b3624e1337681e0909e6c074246103e066118a5924e8f790e7d1f42359da9dc0`.

Schema v8 orders `C3P -> C3V -> C3M -> C3PB -> C3A -> C3L -> C3`. Prefixes are empty. C3 remains frozen to forty exact paths and digest `sha256:c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb`; C3L cannot absorb or relabel any C3 implementation path. C3 must bind the sealed C3L identity as its direct parent while retaining the sealed C3A evidence as C3L's parent.

## Intended C3L claim map

| Exact claim label | Claim type | Pre-seal grade |
| --- | --- | --- |
| `P07B-C C3L legacy-lattice plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3L legacy-lattice plan defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3L historical U6 architecture compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3L U6 phase-admission defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3L cumulative B architecture compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3L B defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3L unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3L cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3L cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3L exact nine-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3L scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3L sealed-C3A predecessor and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

Only these twelve labels may be claimed. Every command uses one exact `/usr/bin/env -i` prefix, pinned local tools, offline Go settings, disabled Node startup injection, and a dedicated private final root. The final chain gate binds the first eleven events and checks its own live invocation/private-directory context under the same non-self-proving boundary used by C3A. It also binds current `HEAD` to sealed C3A's exact commit/tree and exact `refs/notes/didrun` blob/body/11-claim coverage before C3L can commit.

## Throughput proposal remains closed

C3L does not reopen C3V's routed verification-throughput proposal. The Darwin/cgo build cache remains fresh per verifier run; qualified direct general work remains `-p=2`; sensitive and nested work remains serialized; the fail-fast verifier lock remains review-first; and receipt-only units retain their narrow seven-claim profile. C3L changes no verifier parallelism, cache, lock, or receipt-battery setting.

## Nonclaims and gate back to C3

C3L proves only that the historical U6/B chain recognizes exactly the old store lattice or the specifically shaped future C3 bridge edge. Its predecessor check is point-in-time local Git object/note binding; it does not authenticate C3A, rerun C3A, or prove C3L's future seal. It proves no live target, target persistence behavior, Git materialization, host measurement, Node probe, attempt allocation, runtime stability, interlock, process spawn, classification, confidentiality, security boundary, production readiness, adoption, or maintainership. The source-shape admission is repository conformance evidence, not a hostile same-user boundary and not a substitute for C3's compiler, runtime, recovery, race, or composition tests.

After the full C3L build loop, stage exactly the nine paths, archive the development ledger, create a fresh twelve-claim final ledger and private run root, run and immediately claim only the declared rows, commit, seal, inspect the note, loop strict verification to exit `0`, emit the exact-commit HTML report, and archive the final ledger. Then revalidate and apply stash `fa737661e61c36c10ce793f64852836edec20065` without dropping, replace C3's direct predecessor with the sealed C3L identity, and resume the C3 build loop.
