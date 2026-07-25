# Maintenance inventory and ordering

## Ruling

One owner-authorized post-C4 `SOURCE_FULL` maintenance boundary is required
before C5. Call it `C4H`.

C4H should:

- bind exact sealed C4 as its historical/evidence anchor;
- reconcile the live HANDOFF/prompt cursor;
- add bounded fatal stderr diagnostics;
- make the planning fixture mode-exact under caller umask;
- replace the verifier symlink no-op with typed nonzero refusal;
- add real subprocess lock evidence without changing lock semantics;
- reparent C5 to C4H while retaining exact C4 ancestry;
- preserve C5's exact 79-claim monolithic contract.

Proof sharding is declined inside this longitudinal chain after adversarial
review. It would require a new phase/cursor model, receipt state, archive
registry, and claim vocabulary. A post-C6 Composite Proof Protocol v2 remains a
separate R&D direction.

## C4 evidence state

C4 is independently sealed despite its immutable preseal prose:

- commit `c4089ab31181c08dad880bf9e5d40c622c8c91c4`;
- tree `f427e23b6fe297a49a36a99fa037073b637d82ca`;
- 80 complete events and 80 claims;
- all grades `tree-exact`;
- exact HTML and commit-addressed local archive exist.

The `UNRECEIPTED` rows in C4 status are intentionally immutable declarations
that cannot self-receipt. The live operator cursor is now stale and must be
reconciled by a descendant.

## Defects in C4H

### Captured stderr

`gitOutput()` correctly rejects spawn error, signal, nonzero status, or any
stderr, but its failure drops the stderr bytes
(`tools/check-p07b-c-unit-scope.mjs:1241-1265`). The repair keeps stderr fatal
and adds a bounded control-safe byte prefix, byte count, and truncation flag.
It must not echo stdout or claim secret safety.

### Caller-umask fixture

The planning generator requests mode 0644 only during creation
(`tools/generate-p07-planning-example.mjs:152`). Caller umask 077 reduced those
files to 0600. The repair explicitly restores and verifies required directory,
bundle, and subject modes after creation, and tests both 077 and 022.

didrun's fingerprint still does not bind umask. That remains disclosed and
deferred to a future didrun-version boundary.

### Symlinked verifier entry

The direct-entry guard at `tools/verify-current.mjs:533` silently does nothing
when `process.argv[1]` is a symlink spelling of the verifier. Its self-test
currently freezes zero/no-output behavior
(`tools/verify-current-selftest.mjs:863-870`).

C4H preserves inert imports and canonical direct execution, while a
canonical-equivalent symlink path refuses nonzero with
`VERIFY_NONCANONICAL_ENTRY`.

### Real subprocess lock receipt

Production O_EXCL lock semantics already exist and remain unchanged. C4H adds a
black-box child-process scenario showing same-repository fail-fast,
different-repository independence, post-termination stale refusal, and
reviewed-cleanup recovery. `tools/verify-runtime-authority.mjs` is admitted only
if production lock behavior must change; test-only evidence belongs in
`verify-current-selftest.mjs`.

### C5 predecessor traversal

Current `--verify-c5-sealed-c4-note` treats HEAD itself as sealed C4
(`tools/check-p07b-c-plan.mjs:11671-11683`). Reparenting C5 to C4H would break
that command.

C4H must keep the argv and claim label unchanged while strengthening the
implementation to require:

`C5 parent = exact sealed C4H → C4H parent = exact sealed C4 → exact C4 historical chain`.

This adds a prerequisite and preserves the C4 anchor. It does not regrade or
rewrite any C4 claim.

## Proposal disposition

| Item | Status | Decision |
| --- | --- | --- |
| C4 source receipt | Satisfied externally | Consume, never edit/relabel |
| Captured stderr | Live defect | Repair in C4H |
| Caller umask | Live defect | Repair in C4H |
| Symlink false-green | Live defect | Repair in C4H |
| Process-level lock receipt | Coverage defect | Add in C4H |
| Throughput provenance | Documentation defect | Erratum in C4H status |
| Live HANDOFF/prompt cursor | Stale | Reconcile in C4H |
| Phase-fixture consolidation | Already satisfied | Preserve hybrid |
| Pure row-only fixtures | Declined | Fixed semantic oracles are load-bearing |
| Persistent Go cache | Declined | Authority closure incomplete and gain unproven |
| p8/p2 now | Declined | Prior load sensitivity; needs separate qualification |
| Exclusive lock | Satisfied | Preserve |
| Receipt-only profile | Satisfied | Preserve |
| Proof sharding in C5/C6 | Declined with reason | Conflicts with exact current terminal predicate and phase cursor |
| Composite Proof Protocol v2 | Deferred | Post-C6 new vocabulary/authority project |
| didrun changes | Deferred | Keep longitudinal tool version frozen |

## Phase-fixture ruling

C3D already implemented the correct consolidation:

- structural lifecycle tuples from one phase table;
- family-specific semantics as independent oracles;
- wholesale row-only replacement declined.

C3Q's outer-versus-historical scanner asymmetry, C3T's validated absent
baseline, and C3U's exact legacy callers cannot safely become table rows.
Adding C4H requires one structural row and regenerated topology fixtures while
retaining every fixed regression.

## C4H roster

The core exact roster is:

1. `docs/HANDOFF_MODE_C.md`
2. `docs/PROMPT_PACK.md`
3. `docs/VERIFICATION.md`
4. `docs/status/P07B-C-C4H-VERIFIER-HERMETICITY-MAINTENANCE.md`
5. `spec/verification/p07b-c-unit-paths.json`
6. `tools/check-p07b-c-plan.mjs`
7. `tools/check-p07b-c-unit-scope.mjs`
8. `tools/generate-p07-planning-example.mjs`
9. `tools/verify-current.mjs`
10. `tools/verify-current-selftest.mjs`

The deep-dive research package is also admitted as exact documentation source
so it can remain durable and the source-final gate sees no untracked files.
There are no product Go prefixes.

## C4H final claim shape

An 11-command ledger is proportionate:

1. candidate phase-plan coherence;
2. independent candidate-transition authority;
3. plan-checker defensive self-test;
4. exact sealed-C4 note and parent conformance;
5. unit-scope defensive self-test;
6. planning-example caller-umask hermeticity;
7. verifier self-test with symlink and process-lock cases;
8. one cumulative verification;
9. exact admitted staged scope and diff integrity;
10. staged credential scan;
11. exact preceding-chain integrity.

The first eight are `tests-pass`, the last three `command-succeeded`.

## Cheap preflight policy

Before the fresh C5/C6 final ledger, run failure-prone cheap checks as
unclaimed diagnostics on the reviewed staged tree:

- predecessor-note consumer;
- exact source/scope inventory;
- credential scan;
- umask fixture;
- symlink refusal;
- checker/launcher validation;
- any other cheap late-tail command.

These diagnostics confer no grade, do not make any later event reusable, and
cannot replace or skip a final command. The final ledger retains every existing
command, order, immediate claim, source-final gate, credential scan, and
monolithic preseal predicate.

## Cost

Measured local anchors:

| Boundary | Command time | Wall time |
| --- | ---: | ---: |
| C4M | 1.102 h | 1.206 h |
| C4N | 1.107 h | 1.198 h |
| C4P | 1.104 h | 1.216 h |
| C4V | 4.547 h | 4.644 h |
| C4 final | 6.410 h | 14.617 h first-to-last |

A clean C4H is estimated, not receipted, at roughly 1.75-2.25 wall-hours. C5's
79 commands and three cumulative passes should be treated as comparable to or
greater than C4's 6.41 command-hours.

Confidence: **9/10**. C4H scope and the sharding decline are grounded in current
validators, sealed history, and a different-model adversarial review.
