# U2 independent seal audit: six receipts that the happy path did not earn

- **Review target:** unsealed U2 Git publication and Darwin process-owner implementation
- **Reviewer posture:** read-only adversarial pass after the first complete Go-suite success and before any U2 claim
- **Date:** 2026-07-15
- **Verdict:** **BLOCK THE U2 SEAL UNTIL ALL SIX FINDINGS HAVE DEDICATED REGRESSIONS**
- **Receipt state:** `UNRECEIPTED`; diagnostic test success is not a claim, commit, seal, or grade

## Executive ruling

The implementation has no public capability-forgery finding, but its failure edges were materially weaker than its successful path. Three P0 defects could turn known uncertainty into a cleaner receipt: an escaped descendant did not set orphan risk, detailed materialization/start diagnostics were flattened away, and a failed durable-manifest write could leave an orphan manifest outside the cleanup flag. Three P1 defects left caller-controlled work or a process-group reuse window broader than the accepted contract.

U2 remains entirely unreceipted. The required response is to change the implementation and its tests, rerun every load-bearing command through root-serialized didrun, and replay the exact twenty-five-mutant matrix on the final tree. Passing the earlier suite or a narrower mutation run cannot waive these findings.

## Findings and mandatory dispositions

| Priority | Finding | False-clean risk | Required disposition before seal |
| --- | --- | --- | --- |
| **P0** | Incomplete pipe drains set teardown error but not `OrphanRisk` when an escaped `setsid`/`setpgid` descendant keeps inherited descriptors open and the original PGID probe is clean. | A demonstrably live escaped process can coexist with `OrphanRisk == false`. | Conservatively set orphan risk whenever either drain misses its deadline or cleanup certainty is otherwise incomplete. The escape fixture must assert the receipt flag and the secondary `ORPHAN_RISK` control while still cleaning the escaped PID out of band. |
| **P0** | `receiptInput.diagnostic` is supplied for materialization, marker, validation, and cancellation failures but is not incorporated into the receipt. | A coarse primary control erases the typed edge that explains why execution never became admissible. | Derive a stable closed diagnostic code from nested world/Git refusals and context sentinels, bind it into the receipt digest and accessor, and test representative publication-ambiguous, missing-object, marker, bound-materialization validation, and cancellation paths. Do not persist free-form host error text as stable identity. |
| **P0** | The exclusive manifest helper can create a file and then fail during write, chmod, sync, or close before `manifestCreated` becomes true. | Deferred cleanup can miss a partial/orphan adjacent manifest even though publication reports ordinary absence. | Make the helper prove cleanup and parent durability on every post-create failure. If absence cannot be proved, return a typed ambiguous-cleanup outcome. Add deterministic fault injection for write, chmod, file-sync, and close failures plus an absence assertion. |
| **P1** | Promisor-marker admission uses unbounded `os.ReadDir` on a caller-controlled pack directory. | Repository admission allocates and sorts work proportional to arbitrary directory cardinality before applying a closed source policy. | Iterate in bounded batches, declare a maximum pack-entry count, and reject over-limit directories before accepting the repository. Exercise a directory above the exact bound. |
| **P1** | `InstanceNonce` is checked only for nonemptiness before entering canonical identity and marker construction. | Unbounded or control-bearing caller text can consume memory/work and contaminate receipt text before spawn. | Admit only a bounded, valid UTF-8, control-free closed nonce profile before allocation; exercise empty, over-limit, invalid UTF-8, and control text. |
| **P1** | Teardown signals a previously measured PGID without first proving the original group is still observed at the TERM edge. | After ordinary child exit, a reused numeric PGID could theoretically direct TERM at an unrelated group. | Probe immediately before TERM; skip signaling on observed absence and classify probe uncertainty. Document that Darwin offers no atomic probe-and-signal authority, so PGID reuse between those calls remains a trusted-local residual boundary. |

## Seal obligations added by this audit

The final U2 receipt set must contain successful exact-tree events for focused audit regressions in addition to the pre-existing complete suite, race, stress, native SHA-1/SHA-256, policy sentinel, publication fault, fuzz, boundary, mutation, vet, native-facts, and diff/inventory checks. Any regression not executed through didrun is `UNRECEIPTED` even if its source exists.

The audit does not upgrade U2 into a sandbox or hostile-process boundary. An escaped descendant can deliberately close inherited pipes and evade both the original PGID probe and the drain symptom; this remains outside the containment claim. The new orphan flag is conservative evidence for observed uncertainty, not universal escape detection. Likewise, the pre-TERM probe narrows a reuse window but cannot make Darwin PGID signaling atomic.

## Second-pass seal audit: the brief changed again

After the six findings above were implemented, a fresh independent pass found additional proof gaps. None was waived as “already tested.” Each changed either the implementation, its adversarial gate, or the exact claim:

| Priority | Fresh finding | Disposition in the candidate tree |
| --- | --- | --- |
| **P1** | A successful rollback rename was reported as proven absence even when the second parent-directory sync failed. | Rollback resync failure is now `PUBLICATION_AMBIGUOUS`; a native fault-injection regression requires the visible destination to be gone while the receipt remains ambiguous. |
| **P1** | A waiter latched by the bottom `select` could be returned before repolling a newly ready deadline. | Terminal arbitration now exposes and tests an owner-turn priority poll; a latched wait remains provisional until output, cancellation, and deadline have been repolled. |
| **P1** | A non-`ExitError` return from `Wait` could set teardown error without orphan risk while `DirectChildWaited` remained true. | Only a nil wait error or ordinary `ExitError` completes the wait edge; unexpected wait failure keeps the edge incomplete and sets orphan risk. |
| **P1** | Structural materialization refusal, including publication ambiguity, could be flattened to cancellation when cancellation became ready concurrently. | Typed Git/materialization refusals outrank coincident context state; phase cancellation is used only when the returned error matches the context sentinel. |
| **P1** | The boundary analyzer counted direct calls but allowed function-value aliases, receiver decoys, selected-tree assignment, unparsed cgo authority, and drift in load-bearing helper programs. | The positive AST boundary now rejects selector/function aliases, freezes direct Git receiver/argv tuples and selected-tree mutation paths, freezes the native C preamble, and parses/fingerprints both helper programs. Hostile selftests exercise each bypass family. |
| **P1** | Proving an attempt parent empty used whole-directory `os.ReadDir`, allocating and sorting in proportion to arbitrary preexisting cardinality. | Parent admission now opens the directory, requests exactly one entry, fails closed on read/close/no-progress anomalies, and has counting-reader plus large physical-directory regressions. |

The first final-tree mutation replay then killed the audit itself: mutant `allow-destination-overwrite` survived because its test used a non-empty destination directory, which Darwin refuses even after exclusivity is removed. The fixture now uses an empty destination—the overwriteable race shape—and an exact single-mutant A/B/A run kills it. Full-matrix event `222` is permanent negative history and supports no capability.

Darwin also exposed a zombie-only group transition: `kill(-PGID, 0)` can transiently return a positive-presence observation with `EPERM`. The owner retries that observation until the grace deadline. A later observed absence is clean; persistent `EPERM` remains teardown/orphan uncertainty and never authorizes blind KILL. Pre-TERM `EPERM` remains fail-closed. This is a platform-behavior correction, not a broader containment claim.

The consolidated read-only audit found no remaining source-visible P0/P1 blocker after these dispositions. That is reviewer judgment, not a receipt. Every row remains `UNRECEIPTED` until the final tree passes the serialized didrun matrix, is claimed, committed, sealed, and strict-verified.
