# Different-model adversarial review

## Ruling: REVISE

The two-boundary decision has a sound goal, but the proposed sharding protocol
is not executable without changing several existing guarantees. Three design
blockers prevent implementation as written.

This review was read-only and used a different model. It changed no files, Git
state, didrun state, notes, screens, or seals.

## Proven baseline

- The current phase machine is a single ordered list ending
  `C4 → C5 → C6A → C6M → C6B`; it accepts only `SOURCE_FULL` and
  `RECEIPT_RECONCILIATION`
  (`tools/check-p07b-c-unit-scope.mjs:13-18,53-60`).
- Every forward phase carries a tracked `docs/HANDOFF_MODE_C.md` cursor, from
  which the active boundary is derived
  (`tools/check-p07b-c-unit-scope.mjs:469-478`;
  `tools/check-p07b-c-plan.mjs:208-219`).
- Current C5 preseal requires one 79-event/79-claim ledger, one tree, and one
  wrapper-context fingerprint
  (`tools/check-p07b-c-plan.mjs:24118-24135,23589-23617`).
- Native didrun strict can resolve a requested commit through another note with
  the same tree, and its exit status tests only the final grade set (didrun
  `manifest.py:197-239,310-350`; `cli.py:118-130`).
- didrun's event fingerprint omits umask and executable bytes
  (`capture.py:33-50,53-89`).
- The current C6 adapter logic is hard-wired to C6M/C6B and direct C6A source
  derivation; it is not a generic witness-set implementation
  (`tools/check-p07b-c-unit-scope.mjs:1040-1099,1436-1439`).

## Blockers

### B1 — Same-tree witnesses cannot advance the tracked phase cursor

The proposed `C5Wxx`/`C6Wxx` witnesses are allow-empty commits with the source
anchor's exact tree. The current machine requires each phase to change the
checked-in HANDOFF cursor to its own row
(`tools/check-p07b-c-unit-scope.mjs:469-478`;
`tools/check-p07b-c-plan.mjs:208-219`).

Changing HANDOFF changes the tree. Not changing it leaves the active boundary
at C5A/C6A, so current phase-gated commands reject the witness or interpret it
as the source anchor. Candidate transition also requires staged HANDOFF and
specification authority (`tools/check-p07b-c-unit-scope.mjs:1107-1162`).

A future composite protocol would need two authority planes: tracked phase
cursor only for source-bearing boundaries, and non-phase witness identity from
a sealed partition manifest plus exact commit subject/parent/tree/note. Without
that new model, same-tree phase witnesses are impossible.

### B2 — C5B cannot keep current receipt semantics unchanged

Current receipt state contains only `C3P`, `C3`, and `C6A`. Every
`RECEIPT_RECONCILIATION` row must advance exactly one state from `ABSENT` to
`PRESENT` (`tools/check-p07b-c-unit-scope.mjs:14-18,481-505,568-585`).

A C5B row changes no current key and fails. Advancing C6A would falsely receipt
C6 early. A future protocol needs an explicit `C5_SOURCE` receipt state or a
different profile and state semantics, presealed across every row and fixture.

### B3 — The proposed C5P bootstrap remains circular

C4H and C5P both propose to own the phase specification and the two main
checkers. Hashes in a C4H status file do not solve circularity if C5P's staged
checker decides whether those hashes are acceptable.

A future protocol needs a dedicated C4H-sealed bootstrap consumer excluded from
C5P forever. It must run from exact historical C4H bytes, bind C4H identity and
note, inspect candidate index bytes, and compare every C5P-owned blob to a
C4H-sealed canonical manifest. C5P's checker cannot be its own admission
authority.

## High findings

### H1 — Original preseal claim semantics cannot survive sharding unchanged

The existing terminal command `--verify-c5-preseal-ledger` proves that current
`.didrun` contains exactly the preceding 78 events/claims in one linked chain
and one tree (`tools/check-p07b-c-plan.mjs:5612-5616,23589-23617`).

Changing it into global shard reconciliation proves a different predicate.
That may be a worthwhile stronger composite claim, but it is a claim-vocabulary
migration. If labels, types, argv, and predicate must remain unchanged, sharding
must be declined.

### H2 — Physical order and logical ordinal are different

C5A would execute global ordinals `0,1,70,76,77` before W01 executes
`2-13,71,72`. Chronological concatenation cannot equal `0..78`. A future
composite protocol needs explicit `execution_order` and `logical_ordinal`, with
an exact bijection from logical ordinal to shard/local index. It cannot say
physical order equals legacy order.

### H3 — Context and umask are not portable from current notes

Current note validation contains note identity, labels, types, grades, indexes,
and argv previews, but not event `env_fingerprint`
(`tools/check-p07b-c-unit-scope.mjs:896-934`). The local ledger contains the
fingerprint, which itself omits umask.

A future design must separate portable composite authority from local
shard-context attestation. A tracked adapter may attest that it locally checked
archives, but downstream receipt prose cannot claim independent note-derived
proof of historical umask/tool context.

### H4 — Archive discovery is under-specified

Strict verification does not auto-discover archives, and current archives are
explicitly non-Git authority (`tools/check-p07b-c-plan.mjs:7007-7032,
14587-14608`).

A future protocol needs a canonical commit-addressed archive registry, format
and root hashes, exact commit/tree/note binding, and fail-closed missing/extra/
relocated handling. It must never use globbing, “latest,” or live-ledger
fallback.

### H5 — C6 needs a first-class composite schema

Current C6M/C6B derives direct C6A source and C6M note specifically
(`tools/check-p07b-c-unit-scope.mjs:1057-1071,1436-1439`). Adding witnesses
requires a complete C6 schema and hostiles, not parent substitution.

### H6 — Adapter safety needs a dedicated gate

A future `PROOF_ADAPTER` must not be a broadened generic source predicate. Its
gate needs historical bootstrap verification, canonical manifest derivation,
protected-tree comparison, exact staged roster/modes, cached-diff and clean
worktree, credential scan, and adapter-local chain verification.

## Medium findings

- C4H governance may be explicit but remains owner-rooted, not authenticated by
  sealed C4. Its receipt must say so.
- A “fresh” shard root with byte-preserved argv can only mean deleting and
  recreating the same fixed path between shards; unique suffixes change argv.
- Exact-note checks must be primary; native strict is secondary because note
  attachment errors can be ignored and strict can tree-fallback.
- Sharding creates roughly 19 post-C4 boundaries and may be slower on a
  first-try green. Its only demonstrated value is bounded replay loss, not
  throughput.

## Required revision gate for any future composite protocol

Before sharding could be implemented, a predecessor-sealed package would need:

1. A historical immutable bootstrap consumer.
2. A two-plane source-cursor/witness-identity model.
3. An explicit C5 receipt-state transition.
4. A declared semantic migration from single-ledger to logical-ordinal claims.
5. A deterministic local archive registry and portable/local proof boundary.
6. A complete C6 witness/adapter schema.
7. Hostiles for same-tree HANDOFF mutation, missing receipt state,
   candidate-only bootstrap acceptance, physical/logical reorder, archive
   substitution, and C6 witness substitution.

## Final decision

**REVISE.**

If literal preservation of the old one-ledger C5/C6 chain predicate and event
order is required, the ruling becomes **DECLINE SHARDING**. No commit-based
witness protocol can satisfy both.

Confidence: **9/10**. Twenty implementation facts were observed directly, five
mechanical contradictions derived from them, and four points were explicitly
model judgment. Findings: three Blocker, six High, four Medium, two Low.
