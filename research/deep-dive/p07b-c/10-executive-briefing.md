# P07B-C executive briefing

## Bottom line

P07B-C is feasible, but the original planning schemas were too optimistic. The corrected design keeps the three immutable objects while sharply changing the run and crash semantics:

- `ContractExecutionTarget` is official only through a typed store witness and exact private attachment.
- One private boot-session `ExecutionInterlock` serializes subject spawn; the C4 runner then uses StartClaim intent to mint one process-local permit and permanently consumes the target.
- `FinalizedContractRun` owns a bounded closed physical witness with a primary-plus-cleanup process axis and a separate standalone-scope axis.
- `ContractExecution` remains a separately persisted, classifier-profile-bound historical conclusion.
- No mutable semantic execution head, list, latest pointer, process resume, or Wake-like event log is introduced; the private interlock is the sole mutable operational exception and owns no result/history authority.

This is a design correction, not implementation evidence. Current production P07B-C capability remains zero and every proposed row is `UNRECEIPTED`.

## What is now locked

1. Three-object jurisdiction stays.
2. A direct single-target Git capability replaces any temptation to fake a one-candidate comparison world.
3. An explicit Node path and owned probe stay live through spawn.
4. Go owns the authoritative physical run; TAP stays independent.
5. Serialized cooperative at-most-once spawn admission is the claim ceiling; same-boot ambiguity blocks every later subject spawn until explicit operator reset plus a measured boot-session identity demonstrably distinct from the held identity.
6. Scope is `COMPLETE | PARTIAL | VIOLATED`, separate from process control.
7. Raw evidence is private, bounded, omitted by default, and never equated with confidentiality; its current availability is reported separately from immutable FCR/classification truth.
8. Exact-digest reopen is supported; browse/latest/status is not.
9. P08 later projects “Matches / Differs / Could not judge,” with refusals kept outside behavior.
10. Wake integration is an immutable-ref handoff only.

## What changed because of red team

The first synthesis tried to eliminate persisted `ContractExecution`. The different-model critic showed that this would erase the historical conclusion across classifier-version changes. Follow-up verification restored it.

The first synthesis also treated a start claim too broadly. The corrected design calls it a permanent intent tombstone and adds a nonreopenable run permit plus a distinct spawn observation. It does not claim exactly-once execution, child absence after crash, or safe lease takeover.

## Delivery sequence

`C0 scope lock → C1 semantic model → C2 nonhead/interlock substrate → C3 target authority → C4 CLI-profile physical run → C5 HTTP/full scope → C6 cumulative/receipt closure`

Each arrow is a separate commit/seal/strict-clean boundary. C3 is a valid pre-spawn milestone. C4 may ship only as a CLI-only native slice. Full P07B-C requires C5 and C6.

## Go / stop ruling

**GO after the C0 corrections.** Stop at the pre-spawn milestone rather than weakening the label if direct single-target materialization, opaque runtime admission, winner-only permit issuance, or observer-produced scope evidence cannot be proven.

## Confidence and ground-truth tally

**8.3/10 on the architecture direction.** Twelve load-bearing current-repository facts were independently checked; eight implementation decisions remain bets. Executed P07B-C tests/builds/didrun events in the review: zero.

## Human tail

Still outside this run: comprehension/adoption proof, independent security review, production operations, hostile-code containment, cross-platform/runtime matrices, release/signing, legal/name clearance, governance, and maintainership.
