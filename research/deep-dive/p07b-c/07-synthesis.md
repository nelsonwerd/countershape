# P07B-C synthesis

## Initial synthesis ruling

The six lanes agreed on the physical spine but disagreed on classification persistence and restart semantics. The initial synthesis proposed:

- persisted `ContractExecutionTarget` and `FinalizedContractRun`;
- nonpersisted derived `ContractExecution`;
- typed private publication witnesses and exact-key links;
- durable start claim;
- embedded closed-run witness with standalone `COMPLETE | PARTIAL`;
- no mutable registry/head;
- a C0–C5 implementation sequence.

That synthesis correctly rejected generic CAS authority, caller-paired digest bags, TAP-derived classification, comparison-world reuse, unconditional absence booleans, and internal-object-first UX. It was wrong to remove the immutable classification and too imprecise about what a start claim proves.

The later post-write review found a second incompleteness: “new target after ambiguity” can overlap an unknown survivor. The final C0 ruling therefore adds one private boot-session interlock. This changes availability and store mechanics without adding semantic result history or process resume.

## Stable conclusions before red team

1. The current P07B-C schemas are planning authority only and need correction before implementation.
2. A direct `InspectedTree`-issued single-target materialization is mandatory.
3. Node admission must be explicit, probe-owned, and revalidated through spawn.
4. The existing Darwin process owner should be extracted beneath both comparison and contract domains.
5. Generic CAS bytes cannot reopen official execution authority without typed store-private witnesses.
6. Exact-digest semantic reopen is allowed; result list/latest/status mutation is not. The sole mutable exception is the private spawn interlock, which has no semantic authority.
7. Early or incomplete scope checks cannot appear as negative facts.
8. P08 should present one pinned run and a three-way behavioral result, not the authority graph.
9. Wake integration stops at immutable-ref handoff.
10. Every proposed behavior remains `UNRECEIPTED` until its own unit executes and seals.

## Questions deliberately handed to red team

- Does a nonpersisted classifier erase the historical conclusion?
- Is `start.claim` intent, liveness, or execution evidence?
- Are typed witnesses a hidden head or a legitimate exact-key capability resolver?
- Can one embedded witness remain bounded, private, and reopenable?
- Is `PARTIAL` structurally ineligible?
- Can the proposed subunits ship without temporary authority holes?
