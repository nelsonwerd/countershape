# Executive briefing: P07B-C maintenance architecture

## Bottom line

**Proceed with C4H. Decline C5/C6 sharding inside the current claim contract.**

C4 is valid, sealed, and strict-clean. The next unit should repair the concrete
verifier/hermeticity defects that made its finalization fragile, reconcile the
operator cursor, and reparent C5 through C4H while preserving exact C4
ancestry. Then C5 and C6 retain their current monolithic final ledgers.

The proposed sharding architecture initially looked proof-sound because didrun
grades individual claims against trees. Deep review found that the surrounding
Countershape contract makes one crucial terminal predicate irreducibly
single-ledger: C5's final chain claim requires one live 79-event ledger, one
linked event chain, one tree, and one wrapper context. Replacing it with
multi-ledger reconciliation changes the claim's meaning. The user explicitly
forbade claim relabeling or gate weakening, so sharding must be declined here.

This is not a general claim that composite receipts are impossible. It means
they require a new proof vocabulary and authority model. A future Composite
Proof Protocol v2 remains a strong post-C6 experiment.

## What is validated

- Commit `c4089ab31181c08dad880bf9e5d40c622c8c91c4` for C4 binds 80 complete
  `tree-exact` claims to tree
  `f427e23b6fe297a49a36a99fa037073b637d82ca`.
- The product is not the C4 blocker; finalization exposed verifier/fixture and
  transaction-tail defects.
- Fatal Git stderr loses diagnostic bytes.
- Planning fixture modes vary under caller umask.
- A symlinked verifier entry can exit zero without executing.
- The production lock is fail-fast, but its self-test lacks a real competing
  process.
- Current p1, fresh per-run caches, O_EXCL lock, narrow receipt profile, and
  three cumulative passes are intentional.
- Persistent Go-only cache and p8/p2 changes are not justified in this unit.
- C5's existing terminal preseal predicate cannot be preserved by sharding.
- Same-tree phase witnesses conflict with the tracked HANDOFF cursor.
- Current receipt state has no legal C5B transition.

## C4H must fix

1. Add bounded terminal-safe stderr bytes while keeping any stderr fatal.
2. Restore and verify exact fixture modes after creation under umask 077/022.
3. Refuse canonical-equivalent symlink verifier entry with
   `VERIFY_NONCANONICAL_ENTRY`.
4. Add real subprocess lock contention/stale-owner evidence.
5. Record the historical throughput-provenance erratum without rewriting
   sealed history.
6. Reconcile HANDOFF and prompt-pack state.
7. Reparent C5 to sealed C4H, then exact sealed C4, while preserving all 79 C5
   labels, types, argv, command order, and the one-ledger preseal predicate.
8. Preserve p1, GOMAXPROCS=2, test parallel=2, fresh caches, timeouts,
   qualification counts, and didrun version.

## C5/C6 operating mitigation

Run cheap failure-prone commands as unclaimed preflight diagnostics on the
fully reviewed staged tree before starting the fresh final didrun ledger.
Preflight never confers a grade, never makes green events reusable, never skips
a final command, and never alters permanent failed history.

The final ledger remains exact and monolithic.

## Deferred Composite Proof Protocol v2

A later composite protocol needs:

- separate source-cursor and witness-identity planes;
- an explicit C5 receipt state;
- a predecessor-sealed immutable bootstrap consumer;
- a new composite claim vocabulary distinguishing physical execution order
  from logical ordinals;
- exact-commit note admission that rejects didrun tree fallback;
- a deterministic commit-addressed archive registry;
- portable note authority separated from local context/archive attestation;
- first-class C5 and C6 composite schemas and hostile fixtures.

That project is large and potentially valuable. It is outside the unchanged
C5/C6 receipt contract.

## Honest confidence

**9/10.** The final ruling rests on current validator code, exact C4 receipts,
three specialist lanes, focused follow-up verification, and a different-model
red-team. The remaining uncertainty is implementation risk in C4H and the
unreceipted wall-time estimate, not the reason for declining sharding.

Twenty-six of twenty-nine load-bearing conclusions are grounded in code, Git,
local receipts, or mechanically checked ordinal coverage. Three are design
judgments: the best future composite vocabulary, the eventual value of that
protocol, and C4H's expected wall time.

## Proceed?

**Yes.** Seal C4H first. Begin no C5 source work until C4H's exact ledger,
commit, note, strict verification, HTML, and archive are complete.
