# P07B-C C3S — cumulative U6 self-test maintenance

## State

- **Boundary:** active pre-seal `SOURCE_FULL` maintenance after sealed C3F and before the unchanged frozen C3 source unit. Classification: `DEFECT_REPAIR`.
- **Parent:** sealed C3F commit `63038644ba347d7b934a5557a490af95bb4428a4`, tree `3b4c39cfa907c2907d01dcd21399cafe52075872`, is note-present with note blob `098958ba7a9b2d9d237cee901a34279424b68128`, note-body SHA-256 `a23982f84573505fc966378dbcc4f825daf11da8a08b2bfd24f7f71de588d315`, `10/10 claims recorded-exact`, strict exit `0`, and a logged redacted-export override.
- **Scope:** eight exact mode-`100644` paths, empty prefixes; sorted-newline roster `sha256:1d5f793e70caea0ad7379e92bbf781da50846276c28a2aca2d3e7e2ed8c17a26`.
- **Commit subject:** `fix: make U6 C3 fixture phase-stable`
- **Preserved C3 work:** stash object `5d50090882888c115c54a5bee9edb9411c826832` contains the exact forty-path post-C3F C3 payload and remains retained without dropping. Earlier stash objects `f7160bbbc09245a1026eb6396ac76282ab913192`, `fa737661e61c36c10ce793f64852836edec20065`, and `3e766715609680adaef4678fc9aeaa9e3a8bae09` remain historical checkpoints. No stash supplies a C3S or C3 grade.

Every intended C3S grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3S itself.

## Trigger and classification

The preserved C3 build reached the full current-tree verifier after its focused profiles, race lane, architecture checks, and verifier self-test were green. The cumulative run then failed at current row 13 of 46, `architecture-u6-selftest`. Production U6 had already accepted C3's exact store-model bridge. Its hostile-fixture driver failed before executing a checker case because `installC3StoreBridge` assumed the historical import adjacency and always attempted to append a synthetic bridge. On the actual C3 tree that import already exists and the bridge types and methods already have real bodies, so the obsolete mutation anchor was absent. The permanent nonzero development receipt remains archived; it is evidence for this repair, not a receipt for C3S or C3.

This is a defect in cumulative test-fixture construction, not a reason to weaken U6, skip its self-test, delete the failing receipt, or broaden C3 from its sealed forty-path contract. C3S repairs the owning self-test in a separate review-visible boundary. The production U6 policy and verifier roster remain byte-for-byte unchanged.

## Exact repair

The U6 self-test now accepts exactly two starting phases:

1. On the historical C3F tree, absence of the exact `contractmodel` import is admitted only when every C3 bridge surface is also absent. The helper then installs the existing synthetic bridge and validates its complete exact shape.
2. On a C3 tree, the exact import may already exist. The helper then performs no source rewrite and instead requires one exact owner, the ordered four-field input, the non-alias roots and record structs, and all four typed `ObjectStore` method declarations.

Any partial historical surface, duplicate or missing baseline anchor, or ambiguous declaration fails the fixture before the checker result can be misread. The clean bridge case calls the helper twice, exercising install-then-present idempotence inside C3S. Hostile cases mutate unique full signatures and current gofmt field layouts instead of ambiguous substrings. The import-without-bridge case retains the import while removing one required owner type; local-shadow and foreign-owner cases preserve their original semantic attacks. The checksum-pinned 147-case roster, seven clean/comment controls, and 140 hostile cases are unchanged.

A development-only forward diagnostic also ran this repaired self-test against the real stashed C3 `internal/store/nonhead_contract.go` inside an ignored detached worktree and exited `0` through didrun. Because that fixture was outside the receipted source tree, it is supportive debugging evidence only. Exact current-tree compatibility must be re-established by C3's own cumulative claim after C3S seals and the stash is restored.

## Exact C3S scope

C3S owns exactly these 8 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C3S-CUMULATIVE-SELFTEST-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, `tools/check-p07b-c-unit-scope.mjs`, and `tools/check-u6-architecture-selftest.mjs`. Their sorted-newline roster digest is `sha256:1d5f793e70caea0ad7379e92bbf781da50846276c28a2aca2d3e7e2ed8c17a26`.

The source partition is one added status file and seven modified checker/declaration/documentation files. Schema v10 orders `C3P -> C3V -> C3M -> C3PB -> C3A -> C3L -> C3F -> C3S -> C3`. Prefixes are empty. C3 remains frozen to forty exact paths and digest `sha256:c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb`; C3S cannot absorb, edit, or relabel any C3 implementation path. After C3S seals, C3 must bind sealed C3S as its direct parent while retaining C3F, C3L, and C3A ancestry.

## Intended C3S claim map

| Exact claim label | Claim type | Pre-seal grade |
| --- | --- | --- |
| `P07B-C C3S cumulative-selftest plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3S cumulative-selftest plan defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3S historical U6 architecture compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3S U6 phase-stable fixture defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3S unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3S cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3S cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3S exact eight-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3S scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3S sealed-C3F predecessor and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

Only these ten labels may be claimed. Every command uses one exact `/usr/bin/env -i` prefix, pinned local tools, offline Go settings, disabled Node startup injection, and a dedicated private `.countershape/p07bc-c3s-final` root. The final chain gate binds the first nine events and its own live invocation/private-directory context; before C3S commits, it additionally requires current `HEAD` to equal sealed C3F and independently reopens C3F's exact commit, tree, parent, subject, `refs/notes/didrun` blob/type/body, ordered ten-claim semantics, and 10/10 event coverage.

## Nonclaims and gate back to C3

C3S can prove only that the historical U6 policy still passes and that its defensive fixture driver is stable across the absent-bridge and exact-present-bridge phases. Its predecessor check is point-in-time local Git object/note binding; it does not authenticate C3F, rerun C3F, or prove C3S's future seal. It proves no target composition, persistence, Git materialization, host measurement, Node runtime identity, attempt allocation, race freedom, process spawn, classification, confidentiality, hostile same-user boundary, production readiness, adoption, or maintainership.

After the multi-pass build loop, stage exactly the eight C3S paths. Archive the zero-claim development ledger, rotate the development-populated root, create a fresh ten-claim ledger plus private final root, run and immediately claim only the rows above, commit the exact tree, seal it, inspect the note, loop strict verification to exit `0`, emit the exact-commit HTML report, and archive the final ledger. Then revalidate and apply stash `5d50090882888c115c54a5bee9edb9411c826832` without dropping, preserve schema v10 plus the C3S declaration and repaired U6 self-test, bind sealed C3S as C3's direct predecessor, and resume the C3 multi-pass loop.
