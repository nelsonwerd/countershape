# P07B-C C0 authority-lock status

**State:** provisional C0a source document; every C0 row below is `UNRECEIPTED` until the source commit is claimed, committed, sealed, note-checked, and strict-clean. A later C0b receipt-document commit must replace this paragraph with the exact source commit/tree and verbatim grades and add `spec/verification/p07b-c-c0-receipt.json`; the enrolled checker validates either the complete provisional state or the complete reconciled state, never a mixture.

## Sealed predecessor

- Final P07B-B handoff commit: `464e47adbf7f4497dfafa89938a9539239ffd41b`
- Tree: `c7e80d6e6819d4e2c54aaedeb4dc3ef730de58c6`
- Independently reproduced predecessor gate: `5/5 TREE-EXACT`
- Exact predecessor HTML: `.countershape/evidence/p07b-b-final-464e47adbf7f.html`
- HTML SHA-256: `c7e7c66bdc2d8a608dcc2a1dfc3ba6ef90e2373071c3f9ac841b7d72fdd1a62c`
- Preserved predecessor ledger: `.didrun-history/2026-07-18-p07b-b-final-handoff-live/.didrun/`

The C ledger must be fresh. Restoring an old ledger is required only to reproduce that old exact strict result; an HTML snapshot does not remove didrun’s measured local-ledger dependency.

## C0 ruling

C0 locks design authority only:

1. Three immutable nonhead semantic jurisdictions remain distinct: exact pre-spawn `ContractExecutionTarget`, physical `FinalizedContractRun`, and classifier-profile-bound `ContractExecution`.
2. Generic CAS bytes are never official authority. Exact typed publication and held-parent relationships are required; result listing, traversal, mutable status, and latest/current selectors remain absent.
3. One store-private boot-session `ExecutionInterlock` is the sole mutable operational exception. It may block candidate spawn but cannot author, discover, select, or resume a result.
4. C2 supplies only low-level persistence/interlock mechanics with `_test.go` issuers. C3 alone may issue an official live target. The higher C4 contract runner alone may consume that target, acquire the interlock, win StartClaim, and receive the process-local RunPermit. Low-level process mechanics never sees semantic authority.
5. StartClaim is intent, not spawn evidence. SpawnObservation is exactly start error or observed child PID. Missing observation cannot enter an FCR and leaves the interlock held on that boot.
6. A fresh target cannot bypass prior-survivor uncertainty. Ambiguous reset requires explicit operator action plus a measured Darwin boot-session identity demonstrably distinct from the held identity; otherwise C refuses every further subject spawn.
7. One process-control owner preserves at most one primary cause plus independent teardown/orphan controls. Standalone scope is a separate `COMPLETE | PARTIAL | VIOLATED` axis over exactly five named domains.
8. A projected tuple may be retained under later ineligibility. Only no process control plus projected observation plus complete unviolated scope reaches exact tuple membership.
9. Canonical FCR summaries are sufficient historical authority. Private blobs are bounded, may contain secrets, and establish no confidentiality. Prepublication missing blobs refuse FCR; postpublication `RETAINED | PURGED | MISSING_UNEXPECTED` availability never rewrites FCR/classification bytes.
10. Direct generated Node/TAP remains black-box/operator evidence only. Wake remains immutable-ref/inert-provenance handoff only. didrun remains external receipt authority only.

## Deep-dive effect

The six lanes did not rubber-stamp the initial plan. Synthesis first removed persisted classification; the different-model red team restored it. The post-write coherence reviews then found the target-local survivor-overlap gap, process-primary/cleanup regression, premature C2 issuer, process-layering cycle, C4/C5 scope contradiction, C6 self-reference/live-ledger error, private-purge ambiguity, P08 surface leakage, and cumulative `-p=2` conflict. A final different-model closure loop additionally forced deletion/rename-safe staged scope, boundary-safe prefixes, an exact structured authority declaration, coherent boot reset, two-phase C0 receipts, real Git/didrun source binding, and status/handoff agreement before returning `READY`. All critic judgments remain `UNRECEIPTED`; the controlling pack incorporates the corrections.

## C0 files and gates

- Research package: `research/deep-dive/p07b-c/00-scope-and-method.md` through `10-executive-briefing.md`.
- Controlling pack: `docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md`.
- Machine-readable authority summary: `spec/verification/p07b-c-c0-authority.json`.
- Future C0b receipt state: `spec/verification/p07b-c-c0-receipt.json` (correctly absent during C0a). Its source identity and eight grades must reopen from the real ancestor commit/tree and didrun Git note, and its status/handoff tables must match.
- Machine-readable unit scope: `spec/verification/p07b-c-unit-paths.json`.
- C0 plan/structure checker: `tools/check-p07b-c-plan.mjs` with bounded mutation self-test.
- Staged-scope checker: `tools/check-p07b-c-unit-scope.mjs` with strict config/path self-test.
- Cumulative verifier: repaired to direct and inherited Go `-p=1`; C0 self-tests enrolled before C0 seals.
- Proof style: architecture/metadata, black-box, property, parity, recovery, and boundary evidence. The abandoned dense source-rewrite recipe corpus remains unclaimed and is not recreated.

## Provisional receipt map

| Intended C0a capability | Intended claim label | Current grade |
| --- | --- | --- |
| C0 authority/document/section consistency | `P07B C0 authority plan coherence` | `UNRECEIPTED` |
| Plan-checker bounded mutation resistance | `P07B C0 plan checker self-test` | `UNRECEIPTED` |
| Unit-scope checker bounded mutation resistance | `P07B C0 unit scope self-test` | `UNRECEIPTED` |
| Inherited planning compatibility | `P07B C0 inherited planning validation` | `UNRECEIPTED` |
| Serial-Go cumulative repository baseline | `P07B C0 cumulative verification` | `UNRECEIPTED` |
| Exact C0A staged path/diff boundary | `P07B C0 exact staged scope and diff` | `UNRECEIPTED` |
| Scoped structured credential-prefix scan | `P07B C0 scoped staged credential scan` | `UNRECEIPTED` |
| Serialized didrun chain | `P07B C0 didrun chain intact` | `UNRECEIPTED` |

## Explicit nonclaims

C0 implements no C runtime type, schema replacement, store layout, target issuer, Git materializer, boot-session admission, Node admission, interlock, StartClaim, RunPermit, SpawnObservation, process mechanics, standalone detector, FCR, classification, product CLI, server, dashboard, or report. It validates no runtime capability. Linux, Windows, other Node/platform tuples, host-wide absence, network/registry denial, listener ownership, containment, confidentiality, hostile same-user resistance, descriptor-bound execution, security review, production readiness, comprehension, adoption, and maintainership remain `UNRECEIPTED` human/engineering tail.

## Next boundary

Finish C0a only: run the final C0 checks through a fresh didrun ledger, claim each successful event immediately, stage only the `C0A` allowlist, commit, seal, require the Git note, and strict-verify until exit `0`. Then create C0b by adding the strict receipt declaration and replacing provisional receipt state with the exact source commit/tree and verbatim grades, independently claim/commit/seal/note/strict it, and only then begin C1.
