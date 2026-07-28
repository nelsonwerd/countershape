# Governance and phase-machine lane

## Verdict

Restore one owner-authorized `SOURCE_FULL` maintenance boundary directly after
exact sealed C4K:

```text
C4I → C4K → C4J → C5
```

The live topology is `C4I → C4K → C4J → C5`.

C4J is `OWNER_OUT_OF_BAND` and
`OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR`.
Sealed C4K authorized restoring the exact checkpoint in prose, while its v23
machine table did not contain C4J: `predecessor-predeclared = true (sealed
prose only)` and `machine-predeclared = false`. C4J cannot amend, relabel, or
inherit any C4K grade.

## Phase migration

The live phase specification becomes:

- schema `countershape/p07b-c-unit-paths/v24`;
- 38 total units;
- 26 ordered receipt-phase rows;
- C4K parent `C4I`;
- C4J parent `C4K`;
- C5 parent `C4J`;
- C4J profile `SOURCE_FULL`;
- no C4J prefixes;
- C3P receipt `PRESENT`, C3 receipt `PRESENT`, C6A receipt `ABSENT`.

The direct suffix is:

```text
C4H → C4I → C4K → C4J → C5 → C6A → C6M → C6B
```

This requires coordinated updates to the schema, ordered-unit rosters,
specification-owner tables, phase digest, transition matrices, absent/present
and hostile fixtures, Markdown corpus, final-runbook registry, boundary
cardinality maps, candidate-transition rules, and C5/C6 future fixtures.

The final implementation measures and pins authority digest
`30530a2e05fa6b7fb74983756c24f22a898f4a6da6df75d447df17278b0152db`,
plan matrix `14/422`, independent scope matrix `14/409`, and a 69-path
Markdown corpus at
`sha256:a53acdeef0e86b4865caab7822ea06cfa62742794ff1aaa5816023ef172cf901`
with 1,065 hostile rejections and 355 controls. These values are derived
outputs; the process rule remains never to forecast and then force a digest.

## Sealed parent

C4J binds this immutable direct-parent C4K identity:

- commit `8c1ee2df955055ee40eec1f14f0c4d96f97d5361`;
- tree `99fe7082d05c96647a144785e0ddc5af1a09f52f`;
- subject `fix: bound C3 architecture test timeout`;
- note blob `2d039109e269968e2314934fb4e2c4cccb31a097`;
- note-body SHA-256
  `cc276493b0d67bb9903d30591aaa99ed29df09d33154bed2f81faef717435766`;
- 12 of 12 `TREE-EXACT`;
- strict exit zero;
- `secrets_override: true`.

The next lower immutable ancestor remains C4I:

- commit `d784ace97dd03660c6c724e1cb8f838b869681ec`;
- tree `e38b702884b9df128b41bfa7710459dd92f901d7`;
- subject `fix: stabilize verification surfaces`;
- note blob `f1633147e2943e7272abf3b9b903ac72ab274494`;
- note-body SHA-256
  `f221a99aaa1c0037d0de7ebe8c311bfd0ff1653434c4042f30b00a67d9c4e033`;
- 17 of 17 `TREE-EXACT`;
- strict exit zero.

The C4J predecessor validator must traverse:

```text
C4K → C4I → C4H → C4 → C4P → C4N → C4M → C4V → C3D
```

After C4J seals, C5's existing command spelling
`--verify-c5-sealed-c4-note` remains byte-exact but its implementation starts
at C4J and traverses the longer chain.

## Historical preservation

`docs/status/P07B-C-C4I-A2-AST-STABILITY-MAINTENANCE.md` remains byte-exact.
Its v22 schema, 24-row count, C4I-active wording, 17-claim ledger, and
no-timeout-change statement are sealed facts.

The sealed C4K v23 table, 25-row count, C4K-active/parked-C4J wording, and
12-claim ledger are likewise frozen historical facts. The live v24
specification cannot honestly retain either historical schema merely to
satisfy a mutable-file assertion. C4J validates C4K's v23 and C4I's v22
authority from their sealed Git trees and validates only v24 against the live
file.

All live documentation changes must be additive and section-bounded:

- current cursor C4J, parent C4K;
- C5 blocked until C4J seals;
- v24 and 26 receipt-phase rows;
- one bounded outer-child policy extension;
- unchanged nested deadlines, package coverage, ordering, cache, lock, and
  parallelism;
- C4J has no product, HTTP, scope, process, store, classification, didrun, or
  security-review authority.

## Exact C4J scope

C4J deliberately owns 19 exact paths and no prefixes:

1. `docs/HANDOFF_MODE_C.md`
2. `docs/PROMPT_PACK.md`
3. `docs/THREAT_MODEL.md`
4. `docs/VERIFICATION.md`
5. `docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md`
6. `docs/status/P07B-C-C4J-C5-VERIFIER-TIMEOUT-MAINTENANCE.md`
7. `research/deep-dive/p07b-c-c5-timeout-maintenance/01-empirical-failure.md`
8. `research/deep-dive/p07b-c-c5-timeout-maintenance/02-runtime-authority.md`
9. `research/deep-dive/p07b-c-c5-timeout-maintenance/03-governance-phase-machine.md`
10. `research/deep-dive/p07b-c-c5-timeout-maintenance/04-receipt-runbook-impact.md`
11. `research/deep-dive/p07b-c-c5-timeout-maintenance/05-synthesis.md`
12. `research/deep-dive/p07b-c-c5-timeout-maintenance/06-red-team.md`
13. `research/deep-dive/p07b-c-c5-timeout-maintenance/07-executive-briefing.md`
14. `spec/verification/p07b-c-unit-paths.json`
15. `tools/check-p07b-c-plan.mjs`
16. `tools/check-p07b-c-unit-scope.mjs`
17. `tools/verify-current-selftest.mjs`
18. `tools/verify-current.mjs`
19. `tools/verify-runtime-authority.mjs`

The research files are explicitly non-normative evidence and durable working
memory. Validators derive no runtime, phase, or receipt authority from their
prose. They are nevertheless enumerated so staged-scope and credential gates
remain exact.

Recommended subject:

```text
fix: bound C5 architecture verification
```

Recommended private root:

```text
.countershape/p07bc-c4j-final
```

Every generated C4J shell fence sets `umask 077`.

## Required hostiles

- C4J skipping C4K;
- C4K skipping C4I;
- C5 skipping C4J;
- C6A skipping C5 for C4J;
- merge/wrong parent;
- wrong commit, tree, subject, note blob, or note-body digest;
- source path/order/status/mode/identity drift;
- claim label/type/argv/order/grade/coverage drift;
- stale outer head, wrong exact C4K link, or lower-chain substitution;
- unknown phase owner;
- v22 or v23 live-schema fallback;
- extra, missing, substituted, symlinked, or unstaged C4J path.

## Findings

- **Blocker:** the whole phase corpus must migrate together.
- **Blocker:** sealed C4K and C4I history must be read from immutable trees,
  not rewritten in live v24 authority.
- **High:** C5's frozen command name and labels stay unchanged while dynamic
  ancestry gains C4J.
- **High:** C4J must prove exact sealed-parent identity before any self-receipt.
- **Medium:** research is safe to track only because it is explicitly scoped
  and denied normative authority.

## Confidence

**8/10.** Ground-truth tally: 12 of 14 load-bearing conclusions are grounded
in the current schema, checkers, and exact sealed C4K/C4I identities. The
remaining uncertainty is whether the pinned candidate values survive the full
multi-pass receipt loop on the final staged tree.
