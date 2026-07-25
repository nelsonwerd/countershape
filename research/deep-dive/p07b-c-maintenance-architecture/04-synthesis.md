# Phase-2 synthesis: P07B-C maintenance architecture

> **Superseded after follow-up and different-model red-team.** This file
> preserves the Phase-2 recommendation as audit history. The operative ruling
> is `07-executive-briefing.md`: implement C4H and decline sharding inside the
> unchanged C5/C6 claim contract.

All C5P, witness-sharding, and “ten-path C4H” recommendations below are superseded audit history. The operative ruling is one 18-exact-path C4H followed by the unchanged monolithic C5 contract; proof sharding is declined and Composite Proof Protocol v2 is deferred.

## Bottom line

Use two sealed maintenance boundaries, in this order:

1. **`C4H` — verifier hermeticity and observability repair**
2. **`C5P` — proof-partition authority migration**

Then execute C5 as a source anchor plus same-tree evidence witnesses:

`C5P → C5A → C5W01 → … → C5W08 → C5M → C5B`

Apply the same structure, with fewer shards, to C6:

`C5B → C6A → C6W01 → C6W02 → C6M → C6MW01 → C6B`

This is the strongest reconciliation of the three specialist findings:

- The maintenance-order lane was correct that the immediate next boundary
  should contain only the already-proven verifier defects.
- The proof-semantics lane was correct that the 79-command C5 ledger should not
  remain one physical all-or-nothing transaction.
- The throughput lane was correct that cache and parallelism changes would
  alter admitted execution authority and should not be bundled into either
  repair.

Do not combine C4H and C5P. That would mix verifier runtime behavior, evidence
diagnostics, fixture modes, and a new proof calculus in one authority migration.
Do not finish C4H and then run monolithic C5 unchanged: C4's five late failures
and 14.6-hour first-to-last finalization window establish that the physical
transaction topology itself is now a material defect, not merely bad luck.

The present sealed C4 authority remains valid. The inspected note object
`e1e106d3e7cecf51cbf3a6b5752a5b5268e97427` binds exact commit
`c4089ab31181c08dad880bf9e5d40c622c8c91c4`, tree
`f427e23b6fe297a49a36a99fa037073b637d82ca`, 80 complete claims, exact
coverage, and all `tree-exact` grades. C4H must name that exact commit as its
parent. C4 itself did not predeclare C4H, so C4H must be described honestly as
an owner-authorized post-C4 maintenance migration based on the explicit
maintenance request, not as authority derived from C4.

Severity: **BLOCKER** for beginning C5 until C4H and C5P are sealed.

## Cross-lane evidence and contradictions

| Finding | Direct evidence | Resolution |
| --- | --- | --- |
| C4 exposed real late-tail defects, not a failed product | Five different late-run failures are recorded in `docs/status/P07B-C-C4-CLI-PROFILE.md:71-77` | C4 remains sealed; repair only surviving verifier/fixture defects in C4H |
| Current C5 is another monolithic source boundary | `spec/verification/p07b-c-unit-paths.json:797-824`; `tools/check-p07b-c-plan.mjs:5553-5623` | Split, but only after C4H predeclares C5P |
| Current preseal logic assumes one physical ledger | `tools/check-p07b-c-plan.mjs:23546-23653` | Replace only physical-ledger coupling; preserve exact labels, types, argv, global order, tree, and context equivalence |
| Strict didrun verification is not exact-commit proof | didrun `manifest.py:197-239,310-350`; `cli.py:118-130` | Require exact note on intended commit and `resolved-by commit`; strict remains necessary but insufficient |
| `tree-exact` is per event | didrun `claims.py:148-222` | Add a Countershape composite authority manifest and global reconstruction checker |
| General and sensitive execution are intentionally serialized | `tools/verify-current.mjs:20-64`; `tools/verify-runtime-authority.mjs:475-505`; `docs/VERIFICATION.md:446` | Preserve `-p=1`, `GOMAXPROCS=2`, `-parallel=2`, and current sensitive roster |
| Fresh caches are current hermetic authority | `tools/verify-runtime-authority.mjs:442-473` | Decline persistent Go-only cache for this phase |
| Fail-fast lock exists | `tools/verify-runtime-authority.mjs:182-225` | Preserve implementation; add missing real two-process evidence |
| Lock self-test is not process-level concurrency | `tools/verify-current-selftest.mjs:669-772` | Add a real child-process lock test in C4H |
| Symlinked direct entry is a false green | `tools/verify-current.mjs:533-535`; `tools/verify-current-selftest.mjs:863-870`; `docs/VERIFICATION.md:448` | Canonical-equivalent symlink invocation must refuse nonzero |
| Fatal Git stderr loses diagnostics | `tools/check-p07b-c-unit-scope.mjs:1241-1265` | Preserve fatal behavior; add bounded terminal-safe byte diagnostics |
| Fixture mode depends on caller umask | `tools/generate-p07-planning-example.mjs:152-160,187-224` | Restore and verify exact modes after creation under 077 and 022 |
| didrun fingerprint omits umask | didrun `capture.py:33-50` | Disclose; do not modify didrun mid-project |
| Phase fixtures already use a justified hybrid | `docs/VERIFICATION.md:374-376` | Keep semantic oracles; generate only repetitive topology cases |
| Throughput provenance is overstated | `docs/status/P07B-C-VERIFICATION-THROUGHPUT.md:7` and local history | Add an erratum in C4H status; do not rewrite sealed history |
| Live operator docs are stale | `docs/HANDOFF_MODE_C.md:15-34,1028-1035` | Reconcile in C4H |

The contradiction resolves by sequencing: **C4H first** satisfies the
conservative authority requirement; **C5P second** prevents the transaction-loss
pattern from recurring.

## Option comparison

| Option | Advantages | Failure mode | Ruling |
| --- | --- | --- | --- |
| C4H only, then monolithic C5 | Small immediate repair | Leaves 79 commands, 56 qualifications, three cumulative passes, and late preseal checks in one loss domain | Declined as final architecture |
| Combined C4H/C5P | One transition and one cumulative seal | Mixes runtime and proof semantics; a red in either invalidates both | Declined |
| C4H, then C5P | Concrete defects first; C4H predeclares C5P; proof migration gets an independent seal | Two maintenance seals before product resumes | Selected |

The extra maintenance seal is justified. C4's first final command set consumed
about 6.4 command-hours; its attempts spanned about 14.6 wall-hours, and each
late defect voided every earlier green event. The selected structure may add
clean-run overhead but sharply reduces replay loss and gives every new proof
rule an independently reviewable parent.

# Boundary 1: C4H

## Identity and roster

- Unit: `C4H`
- Profile: `SOURCE_FULL`
- Parent: exact sealed C4 commit
  `c4089ab31181c08dad880bf9e5d40c622c8c91c4`
- Subject: `fix: harden verifier execution boundary`
- Purpose: post-C4 verifier hermeticity, diagnostics, and operator-state repair
- Next source child: explicitly predeclared `C5P`

Exact admitted roster, no prefixes:

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

Do not edit sealed throughput status to rewrite history. Carry its provenance
erratum in the new C4H status.

## C4H claim shape

A compact 11-command source ledger is sufficient:

1. Candidate phase-plan coherence
2. Independent candidate-transition authority
3. Plan-checker defensive self-test
4. Exact sealed-C4 commit-note and parent conformance
5. Unit-scope defensive self-test, including captured-stderr hostiles
6. Planning-example caller-umask hermeticity
7. Verifier defensive self-test, including symlink and subprocess-lock cases
8. One complete cumulative verification pass
9. Exact ten-path staged scope and diff integrity
10. Scoped staged credential-pattern scan
11. Exact predecessor and preceding didrun-chain integrity

The first eight are `tests-pass`; the last three are `command-succeeded`.

## Required repairs

### Captured stderr

`gitOutput()` remains fail-closed for spawn error, signal, nonzero status, or
any stderr. Its failure additionally carries total stderr byte count, a bounded
prefix (256 bytes), control-safe lowercase hex, and a truncation flag. It must
not assume UTF-8, print raw controls, reflect stdout, or claim secret safety.

Hostiles cover clean success, spawn error, signal, nonzero exit, stderr-only,
mixed stdout/stderr, 256/257 bytes, invalid UTF-8, NUL/control bytes, and
truncation.

### Mode-exact planning fixture

After creation, explicitly set and verify every contract mode. Prove caller
umask 077 and 022, exact directory and file modes, conforming execution in both
cases, unchanged pre-import tamper refusal, and exact subject-file mode.
Document that didrun's current context fingerprint does not bind umask.

### Canonical entrypoint refusal

Preserve inert imports. Canonical tracked direct invocation runs normally. A
symlink whose real target is the tracked verifier exits nonzero with exact code
`VERIFY_NONCANONICAL_ENTRY`. The former zero/no-output path is forbidden.

### Real subprocess lock evidence

Retain all existing lock/tamper tests and add:

1. Process A acquires the real repository lock and signals readiness.
2. Same-repository B fails fast with `VERIFY_ALREADY_RUNNING` before tool
   admission or run-root creation.
3. Different-repository process succeeds concurrently.
4. After A is terminated, the next same-repository invocation returns
   `VERIFY_STALE_LOCK`.
5. Only reviewed cleanup permits acquisition again.

Automatic stale-lock deletion remains forbidden.

# Boundary 2: C5P

## Identity and roster

- Unit: `C5P`
- Profile: `SOURCE_FULL`
- Parent: exact sealed C4H
- Subject: `refactor: partition C5 and C6 evidence transactions`
- Product/runtime effect: none

Exact admitted roster, no prefixes:

1. `docs/HANDOFF_MODE_C.md`
2. `docs/PROMPT_PACK.md`
3. `docs/VERIFICATION.md`
4. `docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md`
5. `docs/status/P07B-C-C5P-PROOF-PARTITION-MAINTENANCE.md`
6. `spec/verification/p07b-c-unit-paths.json`
7. `spec/verification/p07b-c-proof-partitions.json`
8. `tools/check-p07b-c-plan.mjs`
9. `tools/check-p07b-c-unit-scope.mjs`

C5P must not alter C5 product paths, Go code, `verify-current*`,
`verify-runtime-authority.mjs`, `verify-go-test-repetition.mjs`, or didrun.

## Profiles

Add two explicit profiles:

- `TREE_WITNESS`
  - exact empty roster and prefixes;
  - allow-empty, one-parent commit;
  - exact source-anchor tree, subject, and ordinal;
  - no staged changes;
  - exact commit note required.

- `PROOF_ADAPTER`
  - narrow manifest/document roster;
  - no product or verifier paths;
  - reconstructs already-sealed anchor and witness notes;
  - never calls its own changed tree the C5 product tree;
  - no cumulative product claims.

Keep `RECEIPT_RECONCILIATION` unchanged.

## Phase topology

`C5P → C5A → C5W01 → C5W02 → C5W03 → C5W04 → C5W05 → C5W06 → C5W07 → C5W08 → C5M → C5B → C6A → C6W01 → C6W02 → C6M → C6MW01 → C6B`

Every edge is exact and one-parent. Merges, interposed commits, or foreign-branch
substitutes fail.

## C5 source anchor

`C5A` remains the logical C5 source boundary with the existing 12 exact paths,
the three required prefixes `internal/contractexec/http/`,
`internal/contractexec/scope/`, and `testkit/contractexec/http/`, and subject
`feat: complete standalone contract execution`.

C5's existing 79 labels, types, and argv remain byte-for-byte preserved. `C5`
is the logical claim family and `C5A` its physical source boundary. The
candidate, source-final, and preseal argv keep their existing C5 spelling; the
checker must explicitly define and hostile-test that alias.

## Exact C5 partition

Using zero-based global ordinals:

- `C5A`: `{0,1,70,76,77}`
- `C5W01`: `{2–13,71,72}`
- `C5W02`: `{14–24}`
- `C5W03`: `{25–44}`
- `C5W04`: `{45–64}`
- `C5W05`: `{65–69}`
- `C5W06`: `{73}`
- `C5W07`: `{74}`
- `C5W08`: `{75,78}`

This covers `0..78` exactly once. W06–W08 carry the three cumulative passes;
W08 also carries strengthened global chain reconciliation. Each witness uses a
fresh ledger/root, immediate claims, commit, seal, exact-note check, strict,
HTML, and archive. Failed ledgers remain permanent and contribute nothing.

## Adapter and receipt

`C5M`, `PROOF_ADAPTER`, owns:

1. `docs/HANDOFF_MODE_C.md`
2. `docs/PROMPT_PACK.md`
3. `docs/VERIFICATION.md`
4. `docs/status/P07B-C-C5M-COMPOSITE-AUTHORITY.md`
5. `spec/verification/p07b-c-c5-composite-authority.json`

`C5B`, `RECEIPT_RECONCILIATION`, owns:

1. `docs/HANDOFF_MODE_C.md`
2. `docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md`
3. `docs/status/P07B-C-C5-EVIDENCE.md`
4. `spec/verification/p07b-c-c5-receipt.json`

C5M binds exact commits, trees, subjects, parents, note blobs/body digests,
global ordinals, labels, types, argv, grades, coverage, and context
fingerprints. C5B inherits those bytes; it does not regenerate a plausible
summary.

## C6 partition

Preserve existing labels and argv:

- `C6A`: original ordinals `{0,1,8,9}`
- `C6W01`: `{2,3,4,6,7}`
- `C6W02`: `{5,10}`
- `C6M`: original C6M ordinals `{0,1,2,3,4,7,8}`
- `C6MW01`: `{5,6,9}`

Keep current source/adapter/receipt rosters and subjects; only parent edges and
physical proof ownership change.

## Proof acceptance criteria

### Exact commit note

Every anchor, witness, adapter, and receipt must have an exact note at
`refs/notes/didrun`; stable blob OID and body digest; manifest commit/tree equal
independently derived Git identity; exact subject/parent; and strict resolution
by `commit`. Removing the intended note while leaving another same-tree note
must fail Countershape even if native strict verification tree-falls back.

### Same-tree witness

Each witness has one exact parent, subject, source-anchor tree, empty diff,
ordinal, partition identity, and note. Merges, interposed commits, unlisted
witnesses, or protected-path changes fail.

### Global reconstruction

The final checker reconstructs exactly 79 logical C5 claims with exact labels,
types, argv, ordinals, local indexes, coverage, immediate binding, and grades.
There are no gaps, overlaps, duplicates, reorders, failed/partial contributors,
or meta-claim substitutions. Scope and credential events must come from the
genuine staged C5A tree. Archived shard ledgers are reopened and compared to
event identity, argv, tree, context, exit, output digest, claim binding, and
note content.

### Protected-source and profile preservation

C5A is the sole product tree. Witnesses preserve it exactly; adapter/receipt
commits change only admitted rosters. All three cumulative passes and all
qualification counts/timeouts/assertions remain real and unchanged. The four
profiles have disjoint semantics and hostile mismatch tests.

### Phase fixtures

Use data rows for topology only. Retain fixed semantic oracles for the
C3R/C3Q/C3T asymmetries. A purely generated row system remains declined.

## Proposal routing

### Defect

- Fatal nested Git stderr loses its diagnostic bytes.
- Planning fixture modes vary with caller umask.
- Symlinked verifier direct entry exits zero without work.
- Lock behavior lacks actual two-process evidence.
- Throughput provenance wording is overstated.
- Live handoff is stale.
- C5 evidence can only compose through one failure-domain ledger.

### Intentional

- Any Git stderr remains fatal.
- Go package work remains `-p=1`; `GOMAXPROCS=2`; test `-parallel=2`.
- Private runtime roots and caches remain fresh per run.
- O_EXCL lock and review-first stale refusal remain.
- Three cumulative passes remain distinct.
- Receipt-only work keeps an explicit narrow profile.
- Phase fixtures remain generated topology plus fixed semantic hostiles.
- Failed receipts and old commits remain permanent.

### Declined with reason

- Persistent Go-only cache.
- p8 or p2 without a separate full requalification.
- Pure row-only fixtures.
- Combined C4H/C5P.
- C4H followed by monolithic C5 as final architecture.
- Treating native strict zero as exact-commit proof.
- Relabeling or deleting any C5 claim.

### Deferred with reason

- didrun installation/fingerprint changes.
- broader persistent-cache research.
- future p2 requalification.
- broader legacy prose outside admitted rosters.
- cross-platform production/security hardening beyond the Darwin reference.

## Required execution order

1. Implement and focus-test C4H through didrun.
2. Run the complete current verifier through didrun.
3. Declare, commit, seal, exact-note check, strict-verify, HTML, and archive C4H.
4. Implement C5P only after the C4H strict-verification gate returns zero.
5. Exercise exact-note fallback rejection and full 79-claim reconstruction.
6. Seal C5P independently.
7. Build and seal C5A, then W01–W08 sequentially.
8. Seal C5M and C5B.
9. Repeat the smaller C6 partition.
10. Preserve every red ledger; never reuse green events from a failed shard.

Root remains the sole Git/didrun writer. Read-only criticism can run in
parallel; commits, seals, notes, strict checks, and archives cannot.

## Rough cost

Measured anchors:

- C4M/C4N/C4P: about 1.1 command-hours and 1.2 wall-hours each.
- C4V: about 4.55 command-hours and 4.64 wall-hours.
- C4: 6.41 command-hours and 14.62 first-to-last wall-hours.
- Recent cumulative pass: roughly 79–88 minutes.

Planning estimates, not receipts:

- C4H clean path: 1.75–2.25 wall-hours.
- C5P implementation/hostiles/seal: 2–4.5 wall-hours.
- Sharded C5 clean path: 7.5–10 wall-hours.
- Maximum replay loss after sharding: about one qualification shard or one
  80–90-minute cumulative witness rather than the whole logical ledger.
- C5M/C5B: likely under 0.5 hour each if profiles stay narrow.

The sharded clean path may be slower than a hypothetical first-try monolithic
green. Its value is bounded failure loss, exact attribution, and a proof
structure that survives observed run conditions.

## Confidence

Confidence: **9/10**.

Twenty-two load-bearing conclusions were directly verified from current
repository bytes, sealed C4 identity, or didrun source. Four design judgments
remain to be proven: exact shard sizing, `PROOF_ADAPTER` versus constrained
`SOURCE_FULL`, sufficiency of the owner-authorized C4H-to-C5P migration, and
wall-time estimates.

The key bet is not whether didrun can record 79 commands. It can. The bet is
whether Countershape can compose several exact commit-bound notes into one
logical receipt without losing any claim, argv, tree, event, context, or
ancestry invariant. C5P must prove that adversarially before C5 begins.
