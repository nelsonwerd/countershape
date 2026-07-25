# Phase-3 focused verification

Three independent follow-ups tested the synthesis's remaining design judgments
against the current checker, phase specification, runbook, and sealed-history
precedents. No reviewer changed files, Git, didrun, notes, screens, or seals.

## 1. C5 claim partition

**Verdict: VERIFIED.**

The current C5 definition has exactly 79 paired records:

- 79 labels at `tools/check-p07b-c-plan.mjs:5559-5584`;
- 76 `tests-pass` and three `command-succeeded` types at line 5585;
- 79 command argv projections at lines 5590-5616;
- the renderer independently freezes 79 total and 76 test claims at lines
  7044-7045.

The proposed partition covers every global ordinal `0..78` exactly once:

| Shard | Ordinals | Count | Meaning |
| --- | --- | ---: | --- |
| `C5A` | `0,1,70,76,77` | 5 | Candidate/transition authority, sealed predecessor, staged scope, staged credential scan |
| `C5W01` | `2-13,71,72` | 14 | Product/profile checks, ancestry compatibility, self-tests |
| `C5W02` | `14-24` | 11 | Qualification cases 0-10 |
| `C5W03` | `25-44` | 20 | CLI physical qualifications |
| `C5W04` | `45-64` | 20 | HTTP physical qualifications |
| `C5W05` | `65-69` | 5 | Parity/full-package qualifications |
| `C5W06` | `73` | 1 | Cumulative pass one |
| `C5W07` | `74` | 1 | Cumulative pass two |
| `C5W08` | `75,78` | 2 | Cumulative pass three and global preseal reconciliation |

The tally is 76 test claims and three command claims. There are no missing,
duplicate, or out-of-range ordinals.

Ordinals 76 and 77 must remain in the source anchor because the source-final
gate requires a genuinely staged roster, stable index, clean cached diff, and
no unstaged/untracked paths (`tools/check-p07b-c-unit-scope.mjs:1371-1401`);
the credential gate scans those exact staged blobs (lines 1422-1433). The three
full cumulative passes are ordinals 73-75. Ordinal 78 is correctly last because
it must become global shard reconciliation.

The partition deliberately changes physical chronology while preserving the
logical global order and exact record bytes. C5P must migrate the current
single-live-ledger gate and preserve the embedded
`.countershape/p07bc-c5-final` argv bytes. This is implementation work, not a
partition discrepancy.

Confidence: 0.94 overall; cardinality and ordinal placement 0.99.

## 2. Owner-authorized C4H and predecessor-authorized C5P

**Verdict: PARTIALLY VERIFIED.**

The authority model can honestly support a post-C4 C4H whose governance source
is the owner and whose exact historical/evidence anchor is sealed C4. Existing
maintenance insertions establish precedent: C4V, C4M, C4N, and C4P inserted new
specification owners after exact sealed parents even when the prior phase table
did not name them. This was owner-rooted authority, not authority derived from
the predecessor.

Current v20 does not authorize either new unit. It hard-codes the unit order and
authority digest (`tools/check-p07b-c-unit-scope.mjs:13-18,449-475,587-588`);
topology owners are limited to C3D/C4V/C4M/C4N/C4P (lines 53-60). C4H must
therefore state:

- `authority_source = OWNER_OUT_OF_BAND`;
- `predecessor_predeclared_boundary = false`;
- `historical_anchor =` exact sealed C4;
- `topology_authority_derived_from_c4 = false`;
- classification `OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR`;
- no C4 product grade is inherited, rewritten, or upgraded.

This governance fact is not cryptographically authenticated unless the project
introduces a pre-existing trust key or external signed authorization artifact.
The status must disclose that limitation.

The synthesis did not yet fully make C5P non-self-authorizing. C4H and C5P both
own the specification and two checkers. Merely adding a C5P row would still let
candidate-owned code accept candidate-owned authority.

Before C4H seals, it must freeze:

- exact C5P parent, subject, profile, path/mode roster, and digest;
- exact claim labels/types/argv/cardinality/runbook/nonclaims;
- exact `TREE_WITNESS` and adapter profile semantics;
- exact C5/C6 topology rows and parents;
- expected post-C5P specification digest;
- exact checker digests or a narrowly enumerated checker transformation;
- an immutable bootstrap validator/authority manifest outside C5P's writable
  roster.

C5P admission must execute that C4H-sealed bootstrap consumer by reopening it
from C4H, or compare every candidate-owned authority byte to C4H-sealed
digests. Running only C5P's staged checker is insufficient.

Required hostiles include direct `C4 → C5P`, skipped C5P, wrong/missing notes,
unpredeclared rows/profiles/claims/commands, unauthorized checker/spec bytes,
candidate-only self-acceptance, and later non-owner changes.

No conceptual blocker remains. Full verification requires the sealed bootstrap
contract and explicit governance disclosure.

Confidence: 9/10; 15 machine/doc facts inspected, with owner provenance and the
bootstrap contract remaining open conditions.

## 3. `PROOF_ADAPTER` versus constrained `SOURCE_FULL`

**Verdict: PARTIALLY VERIFIED.**

A distinct `PROOF_ADAPTER` profile is preferable only if adapter boundaries
must avoid cumulative product replay. It is not intrinsically safer than
`SOURCE_FULL`.

Current C6M is the decisive control. It has an exact five-path, no-prefix
source-authority roster and no product/checker ownership, yet is
`SOURCE_FULL` (`spec/verification/p07b-c-unit-paths.json:855`;
`tools/check-p07b-c-unit-scope.mjs:265`). Its runbook includes verifier
self-test and cumulative verification
(`tools/check-p07b-c-plan.mjs:5660,5677-5687`) because current documentation
defines `SOURCE_FULL` to retain the complete cumulative command
(`docs/VERIFICATION.md:84-90`).

Therefore:

- If cumulative replay is acceptable, retain exact-roster `SOURCE_FULL`.
- If C5M/C6M are proof-composition-only, introduce `PROOF_ADAPTER`.
- Never keep the `SOURCE_FULL` label while silently deleting its cumulative
  commands.

`PROOF_ADAPTER` must be a strict source-bearing specialization:

- exact roster, no prefixes;
- at least one exact named authority-manifest path;
- no product/runtime/verifier/checker/didrun/specification paths;
- no receipt claims and no receipt-state change;
- parent-sealed declaration;
- independent note-derived manifest gate;
- exact staged source-final gate;
- exact staged credential scan;
- preseal ledger/chain integrity;
- no cumulative, qualification, product, runtime, or unchanged-behavior claim.

It must route through the same source-final implementation as `SOURCE_FULL`,
including exact roster, mode-100644 nonzero blobs, stable index, cached-diff
check, no unstaged/untracked paths, and repeated inventory
(`tools/check-p07b-c-unit-scope.mjs:1306,1346-1401`). Credential scanning must
also use the same staged-blob path (lines 1317,1330-1343,1422-1433).

Neither exact staged scope nor credentials prove correct manifest derivation.
C5M needs an independent gate that reconstructs exact anchor/witness identities
from Git and notes, analogous to current C6M source-authority admission
(`tools/check-p07b-c-unit-scope.mjs:1073,1107-1205,1436-1439`).

`RECEIPT_RECONCILIATION` is not an alternative: it permits only narrow
Markdown/named receipt declarations, carries fixed receipt claims, and is the
only profile allowed to advance receipt state (lines 481-505,580-585).

Confidence: 9/10; 11 of 13 load-bearing conclusions are directly verified.

## Revised synthesis ruling

The two-boundary sequence remains:

1. C4H repairs stderr, caller umask, symlink entry, subprocess lock evidence,
   operator state, and throughput provenance.
2. C4H additionally seals a complete immutable C5P bootstrap contract.
3. C5P introduces the verified 79-claim partition and a strict source-bearing
   `PROOF_ADAPTER` specialization.

This narrows the original synthesis in two ways:

- C5P is not considered predecessor-authorized until C4H seals exact candidate
  bytes/digests plus an immutable bootstrap consumer.
- `PROOF_ADAPTER` inherits the existing source-final and credential gates
  verbatim; it is not a reduced source-safety profile.
