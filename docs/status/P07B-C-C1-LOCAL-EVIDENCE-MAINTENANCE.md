# P07B-C C1E local-evidence checker maintenance

**Boundary state:** C1E is a `SOURCE_FULL` checker-maintenance unit after sealed C1V and before the unchanged data-only C1B receipt reconciliation. Classification: `DEFECT_REPAIR`. This tracked document is pre-seal authority only; every intended C1E row remains `UNRECEIPTED` until its own commit, didrun seal, independently present Git note, and strict exit `0`.

## Trigger and ruling

The first C1B development pass correctly stopped before freeze. Its four-event, zero-claim ledger has two permanent nonzero events: event `0` rejected a stale phase-specific handoff phrase during checker development, and event `3` then made the real defect visible when `--verify-c1-local-evidence` rejected 57 archived paths as content-address mismatches. The ledger is preserved at `.didrun-history/2026-07-18-p07b-c-c1b-development-local-evidence-namespace-defect/.didrun/`; it supports no C1B capability.

C1E's 21-event, zero-claim, zero-seal build-loop ledger is archived intact at `.didrun-history/2026-07-18-p07b-c-c1e-build-loop/.didrun/`. Its permanent nonzero events are `0`, `9`, `10`, and `11`: the initial stale prompt-pack requirement failure followed by the directory-aware fixture failures that exposed unpruned fan-out residue. The corrected fixture prunes only empty former parents, revalidates a clean baseline after restoration, and never converts those failed receipts into catches or passes.

Independent read-only recomputation showed that the pinned C1 archive, counts, byte totals, and both outer SHA-256 manifests are exact. Installed didrun intentionally multiplexes two disjoint stores beneath one physical `objects/` directory:

- 20 flat didrun capture blobs use `objects/<64-lowercase-hex>` and are named by SHA-256 of their raw stored bytes.
- 57 redirected Git loose objects use `objects/<2-lowercase-hex>/<38-lowercase-hex>` in this SHA-1 repository. Their stored bytes are zlib streams; their joined fan-out name is SHA-1 of the inflated canonical Git object.
- The existing objects-only manifest binds the path, stored byte count, and raw-file SHA-256 of all 77 combined object files. SHA-1 here is Git object-format identity, not the archive-integrity hash.

The old checker applied the flat capture-blob rule to both namespaces. It therefore had no accepting path for the real archive even though the archive matched the installed producer. That is a Countershape checker defect, not evidence of archive corruption and not a didrun runtime defect. The exact 20/57 observation is local C1 archive evidence, not a portable didrun-format guarantee.

## Repair

The local-evidence checker now partitions every `objects/` descendant by exact path and rejects every third shape:

1. A flat lowercase 64-hex path must equal SHA-256 of its raw bytes.
2. A lowercase two-plus-38-hex fan-out path must inflate within the fixed bound as one complete zlib stream with no trailing input, remain at most 4 MiB inflated, contain exactly one canonical `blob | tree | commit | tag` header with canonical decimal length, match its payload byte count, and recompute to the path's Git SHA-1 identity.
3. Before content reads, a bounded iterative traversal enforces the declared file/byte ceilings. The directory set must equal `objects/` plus the nonempty two-hex parents implied by admitted loose objects; empty `pack`/`info`, alternates, uppercase identifiers, temporary files, foreign directories, extra depth, malformed compression, trailing bytes, noncanonical headers, size drift, digest drift, symlinks, and nonregular entries fail closed.
4. Every non-null session stdout, stderr, and transcript reference must be one flat SHA-256 identifier, and the referenced set must equal the flat capture-blob set. Missing and unreferenced capture blobs both fail.

The outer all-files and objects-only manifests remain unchanged and continue to bind stored bytes. The repair adds semantic validation inside that already-bound inventory; it does not weaken, rewrite, flatten, or migrate the archive.

The synthetic self-test positively crosses valid `blob`, `tree`, `commit`, and `tag` loose objects; accepts the exact ceiling and one byte below it; and rejects one byte above it, flat-byte tampering, Git identity substitution, malformed zlib, trailing compressed data, noncanonical headers, declared-size drift, every foreign/empty/extra-depth directory shape, unrecognized object paths, manifest/count/core-digest drift, missing references, and extra unreferenced capture blobs. `--verify-sealed-c1-local-evidence` requires the C1 receipt declaration to remain absent while it exercises the real ignored C1 archive from pinned sealed-C1 constants. C1B later reruns the declaration-bound `--verify-c1-local-evidence` mode, which requires that declaration to be present, against its own exact four-path snapshot. That mode reads one bounded no-follow UTF-8 receipt snapshot and consumes the same bytes it validates; concurrent replacement cannot substitute a second unvalidated declaration.

## Exact C1E scope

C1E owns exactly these 7 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C1-LOCAL-EVIDENCE-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Their sorted-newline roster digest is `sha256:9a6422b45c82a44e6171ae9351468df5f6ece2190d6ab47f26aec6f84fd7ccde`.

Prefixes are empty. C1E uses the full source profile because it changes tracked checker authority. C1B remains unchanged: its four paths, sorted-newline digest `sha256:f8d1fb96d36f7e0cfde99bb73c7726297763c66bafd21aeb5c1fb1e249bdd119`, seven ordered receipt labels/types, and `RECEIPT_RECONCILIATION` profile are not widened or relabeled.

## Intended C1E receipt map

| Exact claim label | Claim type | Pre-seal grade |
| --- | --- | --- |
| `P07B-C C1E checker syntax and evolved plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C1E mixed-namespace local-evidence checker self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C1E sealed-C1 local archive object-namespace match` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C1E unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C1E cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C1E exact seven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C1E scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C1E preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Nonclaims

C1E changes no Countershape semantic model, product runtime, storage behavior, Git publication behavior, process control, standalone execution, external API, or existing/product-capability grade. It may earn only its eight maintenance grades. It does not prove the archive portable across didrun versions or Git object formats. It does not make the local HTML or ignored ledger a portable strict witness, establish secret absence or security review, authorize publication, or grade the stashed C1B work. Its static local-snapshot traversal assumes the enforced sole-writer protocol and does not claim hostile same-user isolation against concurrent directory namespace replacement.

## Gate back to C1B

Finish targeted multi-pass repair and independent adversarial review, stage exactly the seven paths, archive all development ledgers, and start the final ledger at event zero. Run each intended row through didrun and claim it immediately. Commit the exact snapshot, seal it, independently inspect `refs/notes/didrun`, and loop `NO_COLOR=1 didrun verify --strict` until exit `0`; generate the exact-commit HTML and archive the complete ledger only after that gate closes.

Then verify `git stash show --include-untracked --name-only 1d2c14f160ff42d509d96219512fa413830a9b1d` still reports the exact four-path inventory and recorded pre-apply blob/SHA-256 identities. The main stash tree carries the three tracked paths; the untracked receipt is specifically `1d2c14f160ff42d509d96219512fa413830a9b1d^3:spec/verification/p07b-c-c1-receipt.json`. Apply the stash without `--index` and without dropping it. Require the three non-overlap files—`docs/status/DIDRUN_BUGS.md`, `docs/status/P07B-C-C1-SEMANTICS.md`, and `spec/verification/p07b-c-c1-receipt.json`—to match their stashed bytes exactly; deliberately merge `docs/HANDOFF_MODE_C.md` so it retains C1E's sealed receipt while adding C1B's pre-seal state. Validate both authorities and the unchanged four-path C1B roster, then restart C1B from a fresh ledger. C2 remains blocked throughout.
