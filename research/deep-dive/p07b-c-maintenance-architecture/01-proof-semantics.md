# Proof semantics: C5/C6 transaction splitting

## Verdict

Choose option (b) as the target architecture: partition independently
verifiable claims into separately sealed, same-tree witness boundaries, then
admit them through a short machine-checked adapter and receipt closure.

This is proof-sound, but not directly authorized by the current sealed phase
table. A new `C5P` proof-partition maintenance boundary must be explicitly
approved as an authority migration. Current C5 is frozen as one `SOURCE_FULL`
boundary, owns neither phase checker nor phase specification, and has an exact
79-claim runbook (`spec/verification/p07b-c-unit-paths.json:797-824`;
`tools/check-p07b-c-unit-scope.mjs:218-241,547-552`;
`tools/check-p07b-c-plan.mjs:5559-5616`). It cannot safely rewrite its own
admission rules.

If an explicit authority migration is unacceptable, decline the split and
execute the existing C5 contract. Do not disguise a self-authorized C5 checker
change as uninterrupted parent-sealed evidence.

The review was read-only. No didrun, tests, seals, or state-changing Git
commands were run. The frozen C4 note was inspected through Git object plumbing
and contains the expected exact commit/tree, 80 claims, 80 complete events, and
80 lowercase `tree-exact` grades.

## Findings

### 1. Validated: `TREE-EXACT` does not inherently require one ledger

Didrun grades each claim from its supporting event. A claim is `tree-exact`
when that event is self-stable and its tree equals the sealed tree; the claim
type only says a declared command exited zero and supplies no extra semantics
(`/Users/drewnelson/autopilot-dev-stress-test/src/didrun/claims.py:29-31,148-222`;
`ledger.py:107-117`). Nothing in that grading rule requires 79 or 80
otherwise-independent events to share one session.

The single-ledger requirement is Countershape policy implemented by
`verifyLivePresealLedger`: one exact cardinality, one hash chain, one tree, one
wrapper fingerprint, one ordered claim roster, and one staged tree
(`tools/check-p07b-c-plan.mjs:23546-23653`). Therefore it may be replaced only
by an equivalent composition proof, not simply omitted.

### 2. Blocker: raw strict verification accepts the wrong same-tree commit

Didrun first resolves the exact commit note, but if absent scans all notes and
accepts the first manifest with the same tree. Verification then regrades that
manifest's own claims against its own tree. `--strict` exits zero solely from
`report.all_verified`; it does not require `resolved_by === "commit"` or
`report.commit === requested_commit` (`manifest.py:197-239,310-350`;
`cli.py:118-130`).

That behavior is deliberate rebase support, but it is unsafe for a chain of
multiple same-tree witness commits. A missing note on `C5W03` could otherwise
be “verified” by the note on `C5W01`.

Every shard and closure must therefore require all of:

- one note blob from `refs/notes/didrun` attached to the exact commit;
- parsed `note.commit` equals that exact commit;
- parsed `note.tree` equals both the commit tree and source-anchor tree;
- strict output resolves by `commit`, never `tree-fallback`;
- the note blob OID and body SHA-256 remain stable before and after reading;
- the exact note roster, argv, grades, coverage, and indexes validate
  independently.

Current Countershape consumers already implement the stronger pattern: exact
commit reopen, note listing, blob type, body digest, stable note ref, stable
HEAD, subject, ancestry, and source diff
(`tools/check-p07b-c-plan.mjs:10856-10915`;
`tools/check-p07b-c-unit-scope.mjs:948-986`). Reuse that machinery.

### 3. Blocker: the split needs an explicit authority-migration boundary

The sealed v20 table has the direct edge `C4 → C5`, and current checker code
hard-freezes C5's 12 exact paths, three prefixes, and `SOURCE_FULL` profile. C5
may not own the phase specification, phase checkers, repetition helper, runtime
authority, or B checker (`spec/verification/p07b-c-unit-paths.json:797-824`;
`docs/VERIFICATION.md:412-418`;
`tools/check-p07b-c-unit-scope.mjs:218-241,547-552`).

Consequently, neither option (a) nor (b) can be introduced honestly by
executing modified candidate C5 checkers that accept their own expanded scope.
The old sealed checker would reject the extra phase/checker paths; the new
checker accepting itself is not parent authorization.

The next boundary must be explicitly classified as `AUTHORITY_MIGRATION`,
proposed here as `C5P`. It should state that C4 is the exact historical anchor
but did not predeclare C5P. If the owner insists that every phase be predeclared
by its machine-checked predecessor, the only honest ruling is to decline the
split.

### 4. High: a shard chain must reconstruct everything the one ledger proves

The current preseal gate proves more than “all commands were green.” It
enforces:

- exact event/claim cardinality;
- exact event and claim order;
- hash linkage and indexes;
- one stable tree;
- one stable wrapper fingerprint;
- exact argv and admitted redaction positions;
- immediate claim/event correspondence;
- complete coverage;
- exact qualification stdout blobs and executable authority;
- no prior seal;
- equality between the event tree and `git write-tree`.

See `tools/check-p07b-c-plan.mjs:23512-23543,23546-23653`. The final runbook
also fixes command/claim adjacency and seal → note → strict → HTML → archive
order (`tools/check-p07b-c-plan.mjs:6960-7034,7109-7119`).

A compositional replacement needs a parent-sealed partition manifest whose
ordered concatenation exactly reconstructs the old claim vocabulary:

- every original label appears exactly once, unchanged;
- every original type appears exactly once;
- every expected argv is exact except the existing admitted redaction
  projections;
- every original grade remains lowercase `tree-exact` in the Git note; if a
  tracked receipt schema uses uppercase `TREE-EXACT`, that spelling remains
  exact in that schema;
- no missing, extra, duplicate, overlapping, or reordered range;
- local declared/supporting/event indexes are exact;
- each shard has complete local coverage;
- aggregate original-claim coverage equals the frozen original cardinality;
- new shard-integrity claims are separately named metadata claims and may not
  replace or regrade an original product claim.

The opaque wrapper fingerprint should either be equal across every shard or be
superseded by a stronger canonical context commitment that binds the admitted
environment, tool roster, and umask. Merely checking fingerprint stability
within each short ledger would weaken the previous all-events invariant.

### 5. High: scope, credential, and chain claims are source-local

C5's present final three commands are source-final scope, staged credential
scan, and whole-ledger preseal reconciliation
(`tools/check-p07b-c-plan.mjs:5612-5614`). The source gate requires the exact
staged roster, modes, cached diff, no unstaged or untracked files, and stable
inventory; the credential gate scans those staged blobs
(`tools/check-p07b-c-unit-scope.mjs:1346-1401,1422-1433`).

These claims cannot simply be moved after an ordinary source commit while
keeping their existing semantics. Recommended treatment:

- run exact staged-scope and credential claims in `C5A`, while the source
  changes are genuinely staged;
- run independent product, qualification, and cumulative claims in subsequent
  same-tree witnesses;
- place the original “preceding didrun chain integrity” label in the final
  witness, but strengthen its command to validate the complete chain of local
  ledgers, notes, direct parents, partition ranges, context commitments, and
  source tree;
- have `C5P` explicitly freeze this new boundary order.

This changes the old single-session chronology, but does not relabel a claim or
weaken its predicate. If byte-for-byte preservation of the old 79-event
chronology is additionally required, ordinary commit-based sharding is
incompatible with the staged-scope command. A low-level delayed-reference
transaction could preserve it, but introduces unreachable-object and
branch-transaction risks and is not recommended.

### 6. High: same-tree witnesses need stronger commit and immutability checks

A `TREE_WITNESS` commit is accepted only if it is an explicitly declared,
allow-empty, one-parent commit:

- its parent is the immediately preceding declared shard;
- its tree equals the source-anchor tree exactly;
- its diff is empty;
- its subject and ordinal are exact;
- it has its own exact-commit note and strict result;
- it is present exactly once in the aggregate manifest;
- it is an ancestor through the declared linear chain, not merely any ancestor
  of closure HEAD.

This is stronger than “same tree anywhere.” A same-tree commit on another
branch, an unlisted same-tree commit, a merge, an interposed empty commit, or a
note borrowed through tree fallback must fail.

Adapter and receipt commits necessarily have different trees because they add
authority manifests and receipt documentation. Across those commits, every
product, verifier, checker, runtime-authority, and qualification-authority blob
must remain byte-identical to the source anchor. Their changed-path rosters must
equal the narrow adapter or receipt rosters. Current C6 code supplies a useful
model: exact one-parent source derivation, source note validation, inherited
manifest identity, exact adapter/receipt rosters, and byte-identical source
artifacts (`tools/check-p07b-c-unit-scope.mjs:998-1097`;
`tools/check-p07b-c-plan.mjs:14843-14937,14974-15046`).

### 7. High: note attachment and mutation must fail closed

Didrun's note attachment discards the `git notes add` process result
(`manifest.py:301-307`). The generated runbook partly compensates by
immediately showing the exact commit's note before strict
(`tools/check-p07b-c-plan.mjs:6976-7015`). With same-tree commits, this check
becomes mandatory: silent attachment failure plus tree fallback is an otherwise
plausible false green.

The aggregate adapter must reopen every note twice, require the same blob OID
both times, hash its raw bytes, and include that exact OID/hash in the canonical
authority manifest. Future closure verification must recheck the live note ref
against those recorded identities. A note changed after closure invalidates
later verification.

This still does not defeat a coordinated same-user change-and-restore between
observations. That remains outside the current threat claim; sole-writer
operation is required.

### 8. High: HTML and ignored archives cannot repair missing authority

Current C6 receipt semantics correctly label HTML as
`LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS` and the archive as
`LOCAL_IGNORED_ARCHIVE_NOT_GIT_AUTHORITY`
(`tools/check-p07b-c-plan.mjs:14580-14608`). The runbook generates HTML and
archives only after strict succeeds, while retaining the live `.didrun` because
strict does not auto-discover an archive
(`tools/check-p07b-c-plan.mjs:7007-7032`).

Keep that separation. A closure must fail if an exact Git note, exact claim,
coverage record, or declared shard is missing even when an HTML file and
byte-complete archive exist. Archives may support local replay and the adapter
may record their hashes and availability, but neither can be substituted for
the source note or aggregate manifest.

### 9. Medium: failed attempts remain permanent and non-composable

C4's history shows failures after 77 claims, after a wrapper-context split, at
cumulative command 75, and at the credential scan; each attempt correctly made
all earlier green events nonreusable
(`docs/status/P07B-C-C4-CLI-PROFILE.md:71-77`). Sharding reduces the amount
rerun after a late failure, but it must not permit harvesting green claims from
a failed shard.

A shard is admitted only after commit, seal, exact note validation, and strict
exit zero. Any unsealed, partial, failed, wrong-context, or later-repaired
ledger is permanent negative history and contributes zero claims to the
aggregate. A repaired attempt receives a new ledger root, commit, note, and
manifest entry.

### 10. Validated: C6A/C6M/C6B is a strong composition template

The repository already has most receipt-composition primitives:

- a canonical source-authority record binding commit, tree, parent, subject,
  note ref/type/blob/body hash, secrets flag, labels, types, and grades
  (`tools/check-p07b-c-plan.mjs:14430-14461`);
- an exact source note validator for identity, argv, grades, indexes, and
  coverage (`tools/check-p07b-c-plan.mjs:14773-14807`);
- exact source derivation through C6M/C6B ancestry
  (`tools/check-p07b-c-plan.mjs:14843-14937`);
- exact tracked source-summary identity and separate local HTML/archive labels
  (`tools/check-p07b-c-plan.mjs:14540-14616,14974-15046`);
- hostiles for altered commit, digest, label, grade, strict exit, archive, HTML,
  argv injection, redaction, and coverage
  (`tools/check-p07b-c-plan.mjs:15055-15187`).

Extend this design from one source note to an ordered witness-set manifest. Do
not replace it with Markdown saying “the heavy verifier passed.”

### 11. Medium: current C6M is still a source-style proof ledger

The phase table calls C6M `SOURCE_FULL`, and its ten-command runbook reruns the
plan self-test, scope self-test, verifier self-test, and cumulative verifier
before sealing (`spec/verification/p07b-c-unit-paths.json:855-871`;
`tools/check-p07b-c-plan.mjs:5660-5688`). That is safe but undermines the
intended short-adapter architecture.

Under the partition model, the heavy cumulative claim remains verbatim but
lives in a declared `TREE_WITNESS` shard. C6M should validate only exact
source/witness authority, canonical manifest bytes, protected-tree
immutability, its exact adapter scope, credentials, and its local chain. It must
not silently discard the existing cumulative claim.

## Recommended exact state machine

`C5P` must materialize concrete witness counts and claim partitions before
`C5A` begins. `N` and `M` below are not runtime wildcards: the next machine
authority must contain explicit rows, exact subjects, exact claim memberships,
and direct parents.

| Phase | Profile | Exact parent | Role |
| --- | --- | --- | --- |
| `C5P` | `SOURCE_FULL` maintenance | sealed C4 | Explicit proof-partition authority migration; no product source |
| `C5A` | `SOURCE_FULL` | sealed C5P | Commit exact C5 product/source tree; source-local staged scope and credential claims |
| `C5W01` | new `TREE_WITNESS` | sealed C5A | First exact claim shard; allow-empty, same tree |
| `C5W02…C5WNN` | `TREE_WITNESS` | immediately prior witness | Remaining exact shards; direct linear same-tree chain |
| `C5M` | narrow `SOURCE_FULL` adapter | sealed last witness | Add canonical C5 source/witness authority manifest only |
| `C5B` | `RECEIPT_RECONCILIATION` | sealed C5M | Short C5 receipt/handoff closure |
| `C6A` | `SOURCE_FULL` | sealed C5B | Commit exact C6 source/evidence tree |
| `C6W01…C6WMM` | `TREE_WITNESS` | C6A then prior witness | Exact C6 heavy/final witness shards |
| `C6M` | narrow `SOURCE_FULL` adapter | sealed last witness | Add canonical composite C6 authority manifest |
| `C6B` | `RECEIPT_RECONCILIATION` | sealed C6M | Existing short receipt closure |

The single new profile is `TREE_WITNESS`. C5M/C6M can remain constrained
`SOURCE_FULL` units to avoid another profile, provided their exact rosters and
protected-path rules are explicit.

Proposed C5P roster:

- `docs/HANDOFF_MODE_C.md`
- `docs/PROMPT_PACK.md`
- `docs/VERIFICATION.md`
- `docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md`
- `docs/status/P07B-C-C5P-PROOF-PARTITION-MAINTENANCE.md`
- `spec/verification/p07b-c-unit-paths.json`
- new `spec/verification/p07b-c-proof-partitions.json`
- `tools/check-p07b-c-plan.mjs`
- `tools/check-p07b-c-unit-scope.mjs`

It must not own C5 product code, `verify-current`, the repetition helper,
runtime authority, or didrun.

## Canonical aggregate manifest

At minimum, the adapter manifest should bind:

- schema/version and exact partition-authority digest;
- source commit, tree, parent, subject, exact changed-path roster, modes, and
  roster digest;
- ordered witness records: phase, ordinal, commit, tree, direct parent, subject,
  empty-diff assertion, note blob, note-body SHA-256, `secrets_override`, local
  claim count, local coverage, wrapper/context commitment;
- exact original claim records: global ordinal, unchanged label/type, expected
  argv projection, owning shard/local index, exact grade, exit zero;
- exact meta-claim records, kept disjoint from original product claims;
- protected-path manifest and source-to-adapter/source-to-receipt blob equality;
- aggregate cardinalities and exact partition proof;
- explicit declaration that HTML and ignored archives are nonauthority.

`C5B` and `C6B` must require that manifest bytes are inherited unchanged from
the adapter.

## Required hostile fixtures

The split is not ready until fixtures reject:

1. **Identity substitution:** stale tree, wrong tree, same-tree wrong commit,
   missing exact note with successful tree fallback, wrong note ref, nonblob
   note, changed note OID/body, foreign branch, merge commit, interposed empty
   commit, wrong parent, or unlisted same-tree commit.
2. **Claim substitution:** missing, extra, duplicated, overlapping, reordered,
   partial, or cross-shard-reused claims; wrong label/type/case/argv/redaction/
   index/exit/reason/delta/grade; incomplete coverage or wrong aggregate
   cardinality.
3. **Tree mutation:** any source, product, verifier, checker, repetition,
   runtime-authority, or qualification-authority byte changing after the source
   anchor; unauthorized adapter/receipt path or mode; staged/worktree mismatch.
4. **Context substitution:** different wrapper fingerprint without an admitted
   stronger context record, wrong tool roster, wrong umask/context commitment,
   shard-local chain damage, or a prior seal in a supposedly fresh ledger.
5. **Artifact substitution:** HTML/archive present while note is missing;
   copied archive from another commit; local receipt claims used to fill a
   missing source grade; failed/partial attempt listed as a witness.

## Alternatives and tradeoffs

Option (a) is a safe interim pattern: source/heavy seal → narrow adapter →
receipt closure. Existing C6A/C6M/C6B largely implements it. It reduces late
receipt-document failures, but it does not achieve the broader goal if the
source ledger still contains dozens of independent, long-running
qualifications.

Option (b) adds more commits, notes, manifests, and fixture complexity, but it
is the only proposal that reduces the loss radius of independent C5
qualifications and cumulative passes. The proof overhead is justified only if
measured shard durations are used to freeze a bounded schedule. If one
individual command itself exceeds the duration target, sharding cannot fix that
without changing and requalifying the command.

Decline is required if either:

- the owner will not explicitly authorize the C5P authority migration; or
- the old single-session chronology and staged-scope placement are declared
  irreducibly coatomic.

Neither didrun grading nor the product semantics currently demonstrate that all
79 claims genuinely require one ledger.

## Executive summary

Countershape can safely split C5 and C6 evidence, but only by replacing the
current one-ledger proof with an equally strict composition proof.
`TREE-EXACT` is a per-claim relation between a self-stable event and a sealed
tree; it does not inherently require 79 claims in one session. Option (b) is
recommended: one exact source anchor, a linear chain of explicitly declared
same-tree witness commits, a narrow source-authority adapter, and a short
receipt closure.

Two conditions are load-bearing. First, didrun's strict verifier accepts a note
found through same-tree fallback. Therefore every witness must independently
prove that the exact commit has its own exact note and that strict resolved by
commit. Second, current sealed C4 authority predeclares only the existing C5
boundary. C5 cannot safely edit the phase table or checkers that admit C5. A new
C5P boundary must be explicitly authorized as an authority migration;
otherwise the honest decision is to decline the split and run the existing
79-claim C5 contract.

The adapter must reconstruct everything currently supplied by one ledger:
exact labels, types, argv, order, grades, coverage, event correspondence,
context, source tree, direct parents, and protected-path immutability. Same-tree
commits are acceptable only as declared, allow-empty, one-parent witnesses in
a linear chain. HTML and ignored archives remain local snapshots, never
substitutes for an exact Git note. Failed or partial shards remain permanent
negative history and contribute no reusable claims. Existing C6A/C6M/C6B
source-receipt code is the correct foundation to generalize.

## Top five concerns and validated points

1. Didrun tree fallback makes exact-commit note admission mandatory.
2. C5 cannot self-author the required phase/checker migration.
3. Composition must preserve global order, coverage, context, and claim
   uniqueness.
4. Product, verifier, and checker bytes must remain immutable after the source
   anchor.
5. Validated: one ledger is policy, not an inherent `TREE-EXACT` semantic
   requirement.

Confidence: **8/10**. Two blocker, five high, two medium, and two validated
findings are grounded in source and current notes. Residual uncertainty is
operational shard sizing and whether the owner accepts an explicit post-C4
authority migration.
