# Different-model red team

## Verdict

**Structural rework before implementation.** The red team found two blockers in the initial synthesis and several required corrections.

## Blocker 1 — immutable classification was lost

A later derivation can change if tuple codecs, membership rules, missing/empty treatment, or classifier code changes. Without a persisted profile-bound conclusion, Countershape can answer only “what does this binary conclude now,” not “what did it conclude then.” The original three-object jurisdiction was sound.

Correction: preserve `ContractExecution` as a separately published immutable classification with a literal classifier profile and no independent tuple/reason authorship. A crash after run publication permits classification-only retry, never a physical rerun.

## Blocker 2 — start intent is not spawn evidence

No store/process transaction can atomically mean both “intent was durable” and “the OS child exists.” A crash may occur before, during, or after `Start` while recovery sees only the intent.

Correction: `StartClaim` is a permanent intent tombstone. A specialized winner-only create primitive mints one process-local `RunPermit`; a distinct `SpawnObservation` records what the owner observed. Missing observation is unknown. The target is never reusable after intent. A lease may improve availability but cannot prove liveness or authorize same-target takeover.

## High corrections

- Typed publication witnesses require exact-digest resolution and cannot mutate or select current state.
- A closed-run witness needs explicit body/reference/blob ceilings, raw-evidence retention policy, and default-export confidentiality fences.
- Standalone scope must be a closed `COMPLETE | PARTIAL | VIOLATED` axis. Partial/violated structurally force ineligibility and cannot be smuggled into process `ControlReason`.
- C subunits need authority-negative tests so early parsers/store layers cannot accidentally expose spawn or classification.
- P08’s future CLI ABI needs a versioned JSON/result/exit-code contract; scripts must never parse prose or TAP.
- Exact-digest-only inspection is coherent but product-hostile if output is lost. C must print digests; a future immutable collection is a separate design, not a hidden traversal/latest pointer.

## Post-write closure correction

The later coherence review found one issue the initial red team had not closed: a permanent target tombstone prevents duplicate spawn for that target but does not prevent a fresh target from overlapping an unknown survivor. The accepted correction adds one store-private boot-session `ExecutionInterlock`. Only the higher C4 runner may acquire it from C3-issued official-target authority; missing observation or incomplete terminal closure keeps it held. A fresh target is insufficient. Reset requires explicit operator action plus a measured boot-session identity demonstrably distinct from the held identity; otherwise no further subject spawn is admitted. The interlock can block spawn only and is never semantic evidence, result status, history, or a current selector.

## Final C0 different-model closure loop

A separate different-model critic then reviewed the written pack and checkers in three read-only rounds. It first rejected the staged-path gate because deletions/rename sources were omitted and lexical prefixes crossed directory boundaries; it also required an exact structured authority declaration and uniform boot-reset language. After those fixes, it rejected a C0a-only receipt checker that would make truthful C0b reconciliation fail. After the two-phase state was added, it rejected self-consistent invented receipt identities and an unchecked handoff. The final correction now reopens the declared C0a ancestor commit/tree and didrun Git note through admitted Git, matches the exact eight-claim/event roster, and cross-checks delimited status/handoff tables. Wrong-commit, wrong-tree, wrong-grade, and handoff-mismatch controls join the mutation matrix.

The critic's final verdict was **READY** with no remaining high-confidence blocker, high, or medium finding. This is qualitative design/checker criticism only: the critic edited nothing and ran no test, build, didrun, or Git command, so the verdict remains `UNRECEIPTED` and carries no Countershape semantic authority.

## Acceptance attacks

The final pack must kill, among others:

- parsed target reaching spawn;
- second permit after crash or lease expiry;
- historic target/run changing classification under a new binary;
- partial scope plus tuple becoming conforming;
- standalone failure relabeled as generic process control;
- raw secret-bearing evidence entering canonical run bytes;
- missing private evidence reopening as complete;
- P08 status inventing latest via traversal;
- a shared process owner accepting raw argv or a target digest instead of an opaque permit.

## Confidence

**9/10.** The initial two blockers and four high findings were grounded in current schemas, state contracts, store behavior, and process boundaries; the final closure loop dispositioned every later Medium+ finding. No critic-lane runtime gate was executed.
