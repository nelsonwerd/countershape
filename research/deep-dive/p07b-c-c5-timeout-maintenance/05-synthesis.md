# C4J synthesis

## Decision

Create C4J as a separately sealed parent-owned verifier-authority boundary,
then resume C5:

```text
C4I → sealed C4K → C4J → C5
```

The operative short edge is `sealed C4K → C4J → C5`.

This is the smallest design that fixes the observed deadline defect without
making C5 its own execution authority or weakening its evidence.

## Chosen architecture

### Runtime

- Keep default child timeout exactly 20 minutes.
- Admit one exact optional 30-minute execution-policy record.
- Validate the record in the shared runtime before any authority I/O.
- Spawn once; never retry.
- Preserve every other spawn option and both tool revalidations.

### Policy

- Store exact two-row policy outside serialized `currentSteps`.
- Use a prototype-safe immutable record and own-property lookup.
- Elevate only:
  - `architecture-p07b-c-c5`
  - `architecture-p07b-c-c5-selftest`
- Derive policy from the canonical row ID inside the executor.
- Treat caller-supplied row metadata as inert.

### Governance

- C4J is owner-authorized. Sealed C4K predeclared checkpoint restoration in
  prose, while v23 did not machine-declare C4J.
- Phase authority becomes v24 with 26 receipt-phase rows.
- C5 becomes a direct child of C4J.
- Sealed C4K v23 and C4I v22 histories remain immutable and are checked from
  their sealed trees.
- C4J's direct parent is exact sealed C4K; C5's existing ancestry command
  gains `C4J → C4K → C4I` internally without argv churn.

### Evidence

- C4J has a 13-claim ledger.
- A dedicated runtime-policy selftest is mandatory.
- The unchanged repetition selftest directly exercises the changed shared
  runtime's default path.
- Three full cumulative passes exercise the C4J tree.
- Exact stage, credential, and preseal claims close the ledger.
- Commit, seal, Git-note inspection, strict verification, HTML, and archive
  follow the existing didrun loop.

### C5

- Retained checkpoint:
  `1c9dd947f61e3f1fd4c4c2578da0e104f1a02b56`.
- Reapply only after C4J later earns a seal and strict-green result.
- Split the combined sensitive row into exact 9+3 serial rows.
- Rotate the internal roster from 61 to 62.
- Keep all outer 79 claims byte-exact.
- Re-run all qualification and cumulative proof from command 1.

## Why policy is outside step metadata

Both options could be made safe with exact validation. The separate policy
table is chosen because it makes the authority boundary clearer:

- command rows state what runs;
- parent-frozen policy states which canonical row receives more time;
- the shared runtime states the absolute maximum and constructs process
  options.

This also lets C4J seal the policy before C5 activates the rows, while keeping
the C4J current-step roster unchanged. The future C5 roster rotates for one
honest reason only: the additional sensitive subgroup row.

## Deliberate scope

C4J owns 19 exact paths and no prefixes: five live docs, one normative status,
seven non-normative deep-dive files, the phase spec, two phase/plan checkers,
the verifier selftest, verifier, and runtime.

Tracking the research is a conscious expansion, not an accidental untracked
side channel. The project requires files as durable memory; exact scope and
credential checks enumerate the files; runtime and phase validators derive no
authority from their prose.

## Claims preserved

- No product capability is added by C4J.
- No HTTP, scope, process, store, target, permit, or classification behavior
  changes in C4J.
- No package, cache, parallelism, lock, ordering, inner timeout, or test
  assertion changes in C4J.
- No security review, production readiness, portability, performance, or
  adoption claim is made.
- C4J establishes a bounded policy mechanism, not that 30 minutes is always
  sufficient.

## Follow-up verification

The synthesis flags two claims that only real execution can settle:

1. Three C4J cumulative passes must show the unchanged default path remains
   operational across the complete parent-era roster.
2. Fresh C5 cumulative passes must show both aggregate rows actually return
   inside 30 minutes under seal conditions.

No additional speculative analysis can replace those receipts.

## Confidence

**8/10.** Ground-truth tally: 14 of 17 load-bearing conclusions are supported
by source, exact sealed C4K/C4I identities, exact receipt arrays, or captured execution.
Policy-table ergonomics, 30-minute sufficiency, and future atomic-ledger
stability remain design/operational judgments.
