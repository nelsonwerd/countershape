# P07B-C C2M receipt-phase checker maintenance

**Boundary state:** C2M is a `SOURCE_FULL` checker-maintenance unit after the sealed C2 source and before the unchanged C2B receipt reconciliation. Classification: `DEFECT_REPAIR`. This tracked document is pre-seal authority only; every intended C2M row remains `UNRECEIPTED` until its own commit, didrun seal, independently present Git note, and strict exit `0`.

## Trigger and evidence disposition

The C2 source is immutable at commit `19c90786d9832e21ec86daea96ca514cc7304836`, tree `1557f1e26762d717327a2ec770f8153daa0baf3e`. Its final source ledger, Git note, strict result, and exact-commit HTML remain unchanged. C2M neither relabels that source nor changes the retained C2B stash object. The live handoff is intentionally shared by the C2M and C2B rosters: C2M edits its maintenance state now, then C2B will deliberately merge that state with the stashed receipt block.

The first C2B development ledger is preserved intact at `.didrun-history/2026-07-19-p07b-c-c2b-build-loop-pre-maintenance/.didrun/`. It contains three events and zero claims. Event `0` is a permanent nonzero plan check caused by temporarily compressing three required historical C1V archive paths; the draft repaired that documentation and event `1` passed. Event `2` is a permanent nonzero receipt-present plan self-test with the exact first diagnostic `P07B-C evolved plan checker self-test false negative: C2 handoff source digest drift`. None of those events supports a C2B grade.

The exact three-path C2B draft is preserved without dropping at stash object `229a60c0630e296b67faa3c81b91fac73f9f63ff`. Address it by full object identity, never by a mutable stash ordinal. The non-overlap C2 status and receipt-declaration bytes remain future C2B bytes and are not C2M scope. The stashed handoff bytes are retained input to the later deliberate merge, not the live handoff authority for C2M.

## Root cause

`checkC2ReceiptPhase` intentionally has two closed phases. With no C2 receipt declaration, it requires the pending source status, the exact C2 and C2B scope digests in the handoff, fourteen `UNRECEIPTED` rows, and no source-receipt block. With a declaration, it instead requires the admitted source commit/tree/note, fourteen `TREE-EXACT` rows, local-evidence disclosures, and the delimited source-receipt block.

The checker implementation was correct, but its top-level hostile self-test mixed those phases. It always loaded the live handoff and C2 status, then ran four pre-receipt-only hostile mutations. Once a valid C2B draft introduced the receipt declaration, the live checker correctly selected the receipt branch: two handoff digest mutations were no longer phase requirements, the status-digest mutation produced only an unrelated generic boundary error, and replacing `UNRECEIPTED` became a no-op because the live row was already `TREE-EXACT`. The first deterministic false negative stopped C2B before freeze. This is a Countershape self-test lifecycle defect, not a didrun defect, source-receipt defect, or C2 runtime defect.

## Repair

The self-test now exercises both phases on every run without weakening either:

1. The ordinary top-level `checkPlan()` baseline still validates the actual working tree. In C2B that is the receipt-present working-tree baseline.
2. The existing synthetic receipt-phase suite still rejects declaration, authority, Git parent/diff, didrun note/argv, local archive, status, handoff, and self-receipt substitutions. A receipt-present synthetic full-plan baseline plus one hostile handoff-identity mutation also prove that top-level `checkPlan()` wires that branch.
3. A private absence sentinel makes the receipt read fail with a synthetic, `readFile`-equivalent `ENOENT` signal; empty or malformed receipt JSON is not treated as absence.
4. The sealed C2 commit and tree are reopened with config-isolated Git, and the exact sealed C2 status blob supplies the pre-receipt rows. The current handoff is used after removing at most one exact C2 source-receipt block, preserving all later maintenance history while selecting the old phase.
5. Full `checkPlan()` must accept that sealed-source fixture before mutations. The same four pre-receipt-only hostile mutations then run through the full plan wiring with exact expected phase errors.
6. Every mutation anchor must occur exactly once. A missing or duplicated anchor is a test-fixture failure, never a counted rejection.

The unit-scope checker also advances the closed unit schema to v4, enrolls C2M between C2 and C2B, and admits the exact source-final and staged credential gates only for a declared empty-prefix, exact-roster `SOURCE_FULL` unit. Prefix-bearing source units and receipt profiles remain ineligible for these exact-roster gates.

## Multi-pass development evidence

The C2M development ledger will be preserved at `.didrun-history/2026-07-19-p07b-c-c2m-build-loop/.didrun/` with zero claims. Event `1` permanently records the first hostile-suite failure after the schema validator rejected a prefix-bearing C2M mutation earlier and more strictly than the test expected. Event `7` permanently records the receipt-present full-plan fixture refusing a two-occurrence tree anchor rather than mutating ambiguously. Both defects were repaired: the expectation now names the actual rejection layer, and the hostile receipt mutation anchors the unique source-identity sentence. Events `5` and `11` are complete cumulative passes on superseded development trees and remain unclaimed.

Event `16` is a deliberate negative exercise of the final chain mode against this multi-event, zero-claim development ledger. The live Homebrew Node symlink identity passed; the mode then rejected the ledger because it was not the exact seven-event final prefix. This expected nonzero event proves no grade and cannot substitute for the fresh final-ledger chain gate. Later green development events likewise remain rehearsals only.

## Exact C2M scope

C2M owns exactly these 7 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C2-RECEIPT-PHASE-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Their sorted-newline roster digest is `sha256:75dcb6a88a9faec8aa89cfbe67c0870e88d95709a1bf07e2fb494e92362688cd`.

Prefixes are empty. C2M is `SOURCE_FULL` because it changes checker and unit-scope authority. C2B remains exact and stashed: its three paths, sorted-newline digest `sha256:5e30e009bb29672d585ece9893bd7c8db198915335a73ded3f3c790ad18f665b`, seven ordered labels/types, and `RECEIPT_RECONCILIATION` profile are not widened, reordered, or relabeled.

## Intended C2M receipt map

| Exact claim label | Claim type | Pre-seal grade |
| --- | --- | --- |
| `P07B-C C2M receipt-phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C2M pre-receipt and receipt-phase checker self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C2M unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C2M cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C2M cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C2M exact seven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C2M scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C2M preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

Only those eight labels may be claimed. The final chain gate pins the first seven events' index-specific absolute argv identities, claim types, labels, event indices, empty pathspecs, one unchanged staged tree, wrapper-complete green state, and absence of a premature seal; it separately asserts its own live absolute Node/script/mode identity before returning success as event eight.

## Throughput proposal remains closed

C2M does not reopen the sealed C1V throughput ruling. The persistent Go-only cache remains `DECLINED_WITH_REASON` because this Darwin tree contains cgo and Go's cache does not bind imported C-library authority. Qualified direct general work remains at `-p=2`; six whole sensitive packages and nested Go remain at `-p=1`; the failed p4 AST-timeout events and host-sharing p8 decline remain permanent. The fail-fast O_EXCL single-verifier lock and narrow receipt profile are already implemented and sealed. C2M changes none of those settings.

## Nonclaims and gate back to C2B

C2M changes no Countershape runtime, storage semantics, execution authority, source receipt, product surface, external API, or existing grade. It does not prove C2B correct, grade the stashed receipt draft, establish secret absence, replace independent review, or make local HTML/ledger evidence portable.

After multi-pass targeted and cumulative verification plus independent read-only review, stage exactly the seven paths, archive the development ledger, and start the final ledger at event zero. Run and immediately claim only the eight rows above, commit the exact snapshot, seal it, independently inspect `refs/notes/didrun`, and loop `NO_COLOR=1 didrun verify --strict` until exit `0`. Generate an exact-commit HTML report and archive the final ledger only after the strict gate closes.

Then revalidate stash object `229a60c0630e296b67faa3c81b91fac73f9f63ff`, apply it without dropping, keep the two non-overlap C2B paths byte-exact, and deliberately merge only `docs/HANDOFF_MODE_C.md` so it retains the C2M boundary while adding the C2 source-receipt block. Restart C2B from a fresh ledger; the archived three-event development ledger remains permanent history and supports nothing.
