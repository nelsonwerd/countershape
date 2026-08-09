# U7M P07 receipt-successor compatibility maintenance

## Boundary

- **State:** `SOURCE_CANDIDATE`; every row below is `UNRECEIPTED` until the exact candidate commits, seals, exposes its didrun note, passes explicit-commit strict verification, emits the HTML witness, and closes its commit-addressed ledger archive.
- **Parent:** exact sealed U7P commit `05ade5ea9af2ededbc91118c8952631c68e43f36`, tree `372ac1b48b239f6680f8159b7155628d61faaded`.
- **Subject:** `fix: adapt inherited P07 receipt verification`.
- **Profile:** `SOURCE_FULL`.
- **Product authority:** `NONE`.
- **Product behavior:** `INHERITED_UNREPROVEN`.
- **Receipt U7D:** `ABSENT`.
- **Grade transfer:** `NONE`.
- **Topology:** `C6B -> U7P -> U7M -> U7A -> U7B -> U7C -> U7D -> U7R`.

## Why this boundary exists

The first U7A final attempt exposed a deterministic verifier defect rather than a product failure. Current row `architecture-p07b-c-c3p-receipt` invokes the frozen P07 full plan with no injected C6A authority. Its C6B authority resolver accepts only an ambient C6R preseal subject or C6B sealed-current subject. Exact sealed U7P is neither, and no honest U7 descendant can be either. U7P's precommit verifier passed only because ambient HEAD was still exact sealed C6B.

The U7A attempt reached event 8 on exact candidate tree `c65753ad3808c71eb8164b0761bf2dcf258226f7`. Verifier rows 1–61 passed; row 62 exited nonzero with the successor-subject mismatch. No event 8 claim ran. The ledger is permanently preserved at `.didrun-history/u7a-precommit-failed-20260809T181519Z-cumulative-verifier/.didrun`, the final root is preserved at `.countershape/u7a-final-precommit-failed-20260809T181519Z-cumulative-verifier`, and the candidate is anchored by `refs/countershape/checkpoints/u7a-candidate-pre-u7m` at commit `91a8a74fb4be80582c8fe6f8c09ab705a98a8806`, tree `c65753ad3808c71eb8164b0761bf2dcf258226f7`. None of that failed-attempt evidence is reusable.

No legal U7A-roster-only repair exists: product bytes cannot change ambient HEAD authority, while changing the frozen P07 capsule or any of the four protected P07 paths would violate exact sealed-C6B compatibility. U7M therefore interposes one non-product control-plane boundary before U7A resumes.

## Authority classification

U7M is selected by an owner-attributed out-of-band session instruction. The exact amendment was not machine-predeclared, has no qualifying pre-existing artifact, and is therefore:

- `OWNER_OUT_OF_BAND`;
- `OWNER_AUTHORIZED_AUTHORITY_MIGRATION_DEFECT_REPAIR`;
- `UNEVIDENCED` with disclosure `OWNER_ATTRIBUTED_SESSION_INSTRUCTION_ONLY_NO_QUALIFYING_PREEXISTING_ARTIFACT`;
- authentication `NOT_ESTABLISHED`;
- signed authorization `NOT_IMPLEMENTED`;
- product authority `NONE`;
- product behavior `INHERITED_UNREPROVEN`;
- grade transfer `NONE`.

This authority may repair verification topology only. It cannot relabel the failed U7A event, change a product claim, upgrade a receipt, or transfer U7P/C6B grades.

## Exact repair

The frozen P07 checker bytes and all 11 P07 blocks remain unchanged. The raw `architecture-p07b-c-c3p-receipt` row moves intact to historical-only, after the two already-retired successor-incompatible P07 self-test rows. The cumulative verifier retains 70 current rows and 15 historical-only rows.

The current `architecture-p07b-c-inherited-compatibility` row becomes the enforcement-equivalent descendant adapter. It must, in order:

1. bracket the entire operation with exact stable U7 HEAD, index, worktree, specification, and notes authority;
2. run current U7 candidate/ancestry validation with nested success narration suppressed;
3. require all 11 visible P07 blocks and the four protected P07 checker/spec paths to equal exact sealed C6B;
4. bind the tracked C6A source-authority manifest, receipt declaration, evidence summary, and C6 evidence projection to exact sealed-C6B bytes;
5. reopen the declared C6A commit/tree/one-parent/subject and exact didrun note blob/body digest;
6. require the tracked evidence summary at current index/worktree, sealed C6B, and sealed C6A source tree to be the same bytes, compensating for the legacy injected-authority shortcut;
7. execute the unchanged full P07 plan with explicit sealed C6A authority and require an empty error array;
8. rerun terminal U7 candidate validation and require the complete bracketed authority unchanged;
9. emit exactly one success line only after all checks pass.

The one admitted line is:

`U7 inherited P07 compatibility exact: parent=C6B blocks=11 protected_paths=4 c3p_plan=FULL_EXPLICIT_C6A_AUTHORITY legacy_live_entrypoints=RETIRED_SUCCESSOR_INCOMPATIBLE`

The verifier additionally requires empty stderr, exit zero, no signal, and no extra stdout. This establishes successor-compatible full P07 plan validation under exact sealed C6A authority. It does not make any historical failure green, modify the historical checker, or confer a product grade.

## Exact fourteen-path roster

1. `docs/ARCHITECTURE.md`
2. `docs/HANDOFF_MODE_C.md`
3. `docs/PROMPT_PACK.md`
4. `docs/STATE_MACHINES.md`
5. `docs/VERIFICATION.md`
6. `docs/prompts/P08-U7-CLI-REFERENCE-STUDIES.md`
7. `docs/status/U7M-P07-RECEIPT-SUCCESSOR-COMPATIBILITY.md`
8. `spec/verification/u7-unit-paths.json`
9. `tools/check-u7-architecture.mjs`
10. `tools/check-u7-architecture-selftest.mjs`
11. `tools/check-u7-plan.mjs`
12. `tools/check-u7-scope.mjs`
13. `tools/verify-current.mjs`
14. `tools/verify-current-selftest.mjs`

No product, harness, runbook-renderer, P07 checker, receipt declaration, sealed note, or sealed history path is owned by U7M.

## Ordered claim manifest

| Event | Type | Label | State |
|---:|---|---|---|
| 0 | `tests-pass` | U7M candidate plan and sealed-U7P parent authority | `UNRECEIPTED` |
| 1 | `tests-pass` | U7M independent candidate transition and exact staged authority | `UNRECEIPTED` |
| 2 | `tests-pass` | U7M inherited P07 receipt successor compatibility | `UNRECEIPTED` |
| 3 | `tests-pass` | U7M plan contract defensive self-test | `UNRECEIPTED` |
| 4 | `tests-pass` | U7M independent staged-scope defensive self-test | `UNRECEIPTED` |
| 5 | `tests-pass` | U7M final runbook renderer defensive self-test | `UNRECEIPTED` |
| 6 | `tests-pass` | U7M zero-product-surface architecture conformance | `UNRECEIPTED` |
| 7 | `tests-pass` | U7M architecture authority defensive self-test | `UNRECEIPTED` |
| 8 | `tests-pass` | U7M cumulative verifier defensive self-test | `UNRECEIPTED` |
| 9 | `tests-pass` | U7M cumulative verification on the exact staged maintenance candidate | `UNRECEIPTED` |
| 10 | `command-succeeded` | U7M exact fourteen-path staged scope and diff integrity | `UNRECEIPTED` |
| 11 | `command-succeeded` | U7M scoped staged credential-pattern scan | `UNRECEIPTED` |
| 12 | `command-succeeded` | U7M sealed-U7P predecessor and preceding didrun chain integrity | `UNRECEIPTED` |

Every claim binds its matching zero-based event and all fourteen paths in declared order. A command's green output is not a receipt until the complete final closure succeeds.

## Continuation and nonclaims

U7M changes no product behavior. It does not prove adoption, comprehension, maintainability, production readiness, security review, hostile containment, network denial, Linux/Windows behavior, other Node majors, or any U7 study result. Its architecture phase has zero governed product files.

After exact sealed U7M, restore the ten checkpointed U7A product/test paths byte-for-byte, regenerate only U7A's HANDOFF and status projections for parent U7M, and rerun all twelve U7A events/claims from event zero. U7B, U7C, U7D, and U7R remain unchanged except for their new transitive U7M ancestry.
