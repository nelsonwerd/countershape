# Focused follow-up verification

## 1. Classification persistence

**Verdict: partially verified; retain the separate object.** Classification is mathematically derivable from immutable inputs, but independent persistence is the stronger audit/product boundary. `FinalizedContractRun` owns physical truth; classification also consumes the exact bundle predicate and classifier profile. Separate publication lets the physical run survive classifier failure, supports classification-only retry, and permits a future profile to produce a distinct conclusion without rewriting the run.

Minimal `ContractExecution`:

```text
schema_version
kind = ContractExecution
classifier_profile = CONTRACT_EXECUTION_EXACT_TUPLE_V1
contract_execution_target_digest
finalized_contract_run_digest
result = CONFORMS | CONTRADICTS | INELIGIBLE_EXECUTION
```

The result is constructor-derived. It contains no tuple or control/scope reason.

## 2. Start claim

**Verdict: corrected after post-write review; name the narrow guarantee.** A deterministic target-keyed StartClaim alone enforces at most once only for that target. The final design additionally uses one store-private boot-session `ExecutionInterlock`. Only the C4 runner may acquire it from C3-issued official-target authority, then win StartClaim and receive a nonserializable, nonreopenable, single-consumption RunPermit. Generic publish/read, C2 fixtures, or a losing concurrent creator must never mint the permit.

StartClaim records intent and permanent non-reissuance. SpawnObservation separately records `START_ERROR` or an observed child PID. Missing observation remains unknown, cannot enter an FCR, and keeps the interlock held on that boot. Any ambiguity permanently tombstones the target; a fresh target/attempt alone is not retry authority. Reset requires explicit operator action plus a measured boot-session identity demonstrably distinct from the held identity; otherwise C refuses every further subject spawn.

## 3. Closed-run and standalone evidence

**Verdict: partially verified; foundations exist, implementation does not.** Use bounded canonical summaries and typed refs in the run; those summaries are sufficient for immutable FCR/classification authority. Keep raw/large evidence private with an exact manifest, count/byte ceilings, retention-at-finalization status, explicit purge, and default omission. Before FCR publication, every required blob must revalidate. After publication, current availability is separately reported as `RETAINED`, `PURGED`, or `MISSING_UNEXPECTED`; it never mutates the FCR or historical classification and must never misreport missing bytes as retained.

Standalone scope is:

```text
COMPLETE  — all five required checks have typed evidence and no violation
PARTIAL   — missing checks are explicit and nonempty
VIOLATED  — at least one typed violation is present; other checks may remain missing
```

The process axis remains separate and preserves at most one primary cause plus independent teardown/orphan controls under one owner. Cleanup findings never overwrite the primary cause. The derived terminal disposition is one of:

- `ELIGIBLE_CLEAN`;
- `INELIGIBLE_CONTROL`;
- `INELIGIBLE_STANDALONE`;
- `INELIGIBLE_CONTROL_AND_STANDALONE`.

Only clean process closure plus a projected tuple plus standalone `COMPLETE` is eligible for tuple membership. A projected tuple may be retained under standalone ineligibility but cannot produce `CONFORMS` or `CONTRADICTS`.

## Recommended ceilings for C1 validation

- Canonical `FinalizedContractRun`: existing 1 MiB parser ceiling.
- `ClosedRunWitness`: 256 KiB.
- Typed witness references: 16 total, exactly five standalone check domains.
- Private evidence: 16 blobs / 64 MiB aggregate per finalized run.
- Existing CLI capture: 16 MiB per stdout/stderr stream.
- Existing HTTP capture limits remain the source of truth.

These numerical choices are design recommendations, not evidence of acceptable production performance.
