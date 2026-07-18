# Lane 1 — authority and data model

## Verdict

Keep three jurisdictions:

1. `ContractExecutionTarget` owns immutable pre-spawn intent and exact inputs.
2. `FinalizedContractRun` owns the closed physical facts.
3. `ContractExecution` owns the immutable classification conclusion.

The current planned shapes are not implementation-ready. `FinalizedContractRun` duplicates control-reason authority between disposition and observation, treats unobserved standalone facts as false, and exposes anonymous digest bags that a caller could pair. The target has redundant source fields and no absolute admitted runtime path. `ContractExecution` includes constant-false declarations that are not physical proof.

## Required invariants

- A parsed body is inert. Only an opaque store-issued capability may cross a stronger edge.
- One exact target binds one reopened residue/bundle, one pinned and inspected Git tree, one private materialization, one fresh attempt, one measured Darwin boot session, and one admitted Node runtime.
- The attempt marker predates target publication and spawn but does not self-reference the target.
- One target admits at most one cooperative spawn attempt, and one store-private boot-session interlock prevents a fresh target from bypassing unresolved prior-survivor uncertainty.
- A finalized run can be constructed only from the consumed target/run permit and terminal owner-produced facts.
- Process control and standalone-scope closure are separate axes.
- Classification accepts no requested tuple, reason, or class; it derives from the exact reopened graph.
- Repeated runs allocate distinct targets, runs, and executions even when their observed tuples match.
- None of the three objects advances or selects the study head.

## Corrections

### Target

Derive source/profile facts by reopening the exact `ContractBundle`; do not accept a second caller-authored source triple. Carry the measured Darwin boot-session identity, absolute admitted Node path, owned probe result (`process.execPath`, version/major, platform, architecture), and executable identity summaries. Keep its external artifact digest outside its own body.

### Finalized run

Replace independent disposition/observation reason ownership with a constructor-derived terminal algebra. Preserve one process-control owner containing at most one primary cause plus independent teardown/orphan controls, matching the sealed attempt model so cleanup cannot overwrite the first cause. Add a separate standalone-scope result (`COMPLETE`, `PARTIAL`, or `VIOLATED`) and make eligibility structurally require no process controls plus `COMPLETE` with no violation.

### Classification

Retain a separately persisted `ContractExecution` with a literal classifier profile such as `CONTRACT_EXECUTION_EXACT_TUPLE_V1`, exact target/run digests, and only `CONFORMS`, `CONTRADICTS`, or `INELIGIBLE_EXECUTION`. Remove tuple/reason duplication. Nonfreshening is proved by topology and head evidence, not by authored `false` values.

## Key evidence

- Jurisdiction split: `docs/ARCHITECTURE.md:45-47` and `docs/SEMANTICS.md:142-146`.
- Existing single-target materialization gap: `internal/gitobj/types.go:321-354,388-392`.
- Current duplicated run reasons: `spec/schema/v1/finalized-contract-run.schema.json:47-100`.
- Current narrow classification body: `spec/schema/v1/contract-execution.schema.json:20-50`.
- P08 consumes the distinct triple: `docs/prompts/P08-U7-CLI-REFERENCE-STUDIES.md:75`.

## Confidence

**8.7/10.** Ten authority facts were checked in current code/docs; the corrected canonical shapes remain prospective.
