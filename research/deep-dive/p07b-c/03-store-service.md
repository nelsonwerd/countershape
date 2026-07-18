# Lane 3 — store, service, and crash recovery

## Verdict

The generic content-addressed store can persist canonical bytes, but object existence cannot prove official capability-only construction. P07B-C therefore needs typed, store-private publication witnesses and create-once exact-key links around the CAS.

No mutable semantic execution registry is allowed. Exact-digest reopening is deliberate; C does not implement result listing, `latest`, mutable result status, process resume, or an execution head. The final post-write correction permits exactly one mutable store-private `ExecutionInterlock`: it is the sole mutable operational exception, serializes spawn for a measured boot session, and is not canonical evidence, history, discovery, or result authority.

## Proposed topology

```text
objects/sha256/...                         canonical semantic bodies
nonheads/v1/publications/<kind>/<digest>   typed official-publication witness
nonheads/v1/links/target-by-attempt/...    create-once target link
nonheads/v1/links/run-by-target/...        create-once run link
nonheads/v1/links/execution-by-run-profile/... create-once classifier link
private-contract-runs/v1/<attempt>/        private operational attachment
private-contract-runs/v1/interlock         sole store-private boot-session spawn interlock
```

All operational directories/files are owned, real, no-follow, and mode `0700`/`0600`. Create-once links never change and select no newest value.

## Start boundary

Only C3 live Git/runtime construction can issue official-target authority; C2 store fixtures remain `_test.go`-only. In C4, the higher contract runner durably acquires/reopens `ExecutionInterlock` for that official target and boot session, then creates/reopens the target-keyed immutable StartClaim. Only the combined winner receives an opaque process-local RunPermit. Generic publish/read, existing-claim reopen, copied epochs, C2 fixtures, or restart may never recreate a permit, and low-level process mechanics never sees it.

The guarantee is **serialized cooperative at-most-once spawn admission per exact target**, not exactly-once execution. A distinct `SpawnObservation` records `START_ERROR` or an observed child PID. Missing observation is unknown and cannot enter an FCR. A conclusive terminal witness becomes durable before releasing the interlock. Any crash or ambiguity after intent permanently consumes the target and holds the interlock on that boot: fresh target alone is insufficient, and another target cannot spawn merely because its identity is fresh. Reset requires explicit operator action plus a measured boot-session identity demonstrably distinct from the held identity; otherwise no further subject spawn is admitted. No lease or interlock is upgraded into process-existence evidence.

## Crash rules

- Target bytes without typed witness are unofficial and inert.
- A fully published target without a claim may be reopened only through its exact private attachment and revalidated authorities.
- Once a claim may exist, the same target is never spawned again; missing terminal closure also holds the store-private interlock for that boot session.
- Owner loss never implies the child did or did not start, exit, or clean up; no same-boot fresh-target bypass is permitted.
- A durable terminal witness may support finalized-run publication retry without a physical rerun only while every required prepublication private blob is present and revalidated.
- A finalized run may support classification-only retry.
- After finalized-run publication, explicit purge or unexpected blob loss changes only separately reported evidence availability; canonical summaries and the historical classification remain immutable, and missing bytes are never reported as retained.
- The terminal study residue never rolls back or advances.

## Key evidence

- Generic publish/read behavior: `internal/store/object_store.go:56-103,179-240,285-316`.
- Existing publication idempotence: `internal/store/object_store_test.go:128-167`.
- Existing residue capability precedent: `internal/emit/node/publication.go:34-45,156-212`.
- Spawn attempted versus started: `internal/world/process_darwin.go:195-244` and `internal/world/receipt.go:409-411,470-471`.
- No process resume: `docs/ARCHITECTURE.md:267-271`.

## Confidence

**8.1/10.** The CAS and process facts are checked; the typed witness namespace and winner-only claim primitive remain unimplemented.
