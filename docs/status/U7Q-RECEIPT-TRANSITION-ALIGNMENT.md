# U7Q receipt-transition and recorder-runtime control alignment

## Boundary identity

- **State:** active source candidate; not committed, sealed, or receipted.
- **Namespace:** `countershape/u7-unit-paths/v3`.
- **Boundary:** `U7Q`.
- **Declared parent:** exact sealed `U7D` commit `b6ea9abf3d036a7fbda5c579a4bcfebd22b2d7e3`.
- **Parent tree:** `5b24536a793a407b833419e048c143e94778280b`.
- **Parent subject:** `test: close U7 reference study evidence`.
- **Subject:** `fix: reconcile U7 receipt transition controls`.
- **Profile:** `SOURCE_FULL`.
- **Product authority:** `NONE`.
- **Product behavior:** `INHERITED_UNREPROVEN`.
- **U7D source receipt:** `ABSENT`.
- **Final private root:** `.countershape/u7q-final`.
- **Next boundary:** `U7R` only after U7Q closes its own fresh 13-command ledger, seal, note, strict HTML witness, archive, and terminal audit.

Exact sealed U7D carries note blob
`a75a0c78490ad4ab12b4e1e8938ddef0c3705aa8`, 29,099 bytes, with
note-body SHA-256
`5618203268e90d173fd7ee6fb843cd657855b41cc42b653033fce2666075a847`.
It closed `13/13` claims at `TREE-EXACT`, strict exit `0`, with
`secrets_override: true`. Its local HTML is
`.countershape/evidence/u7d-final-b6ea9abf3d03.html`, 8,854 bytes, SHA-256
`b93573200a6175ee34a1450d3e916096f8b4fbc99f806dc96a73c6fe749f44a5`;
its archive is `.didrun-history/u7d-final-b6ea9abf3d03/.didrun`. These facts
admit U7D as predecessor and future receipt-source authority only. They transfer
no U7Q grade.

## Defect and repair boundary

The frozen U7R contract admitted no possible candidate. Its phase gate required
HANDOFF to carry the canonical `U7R` / `RECEIPT_CANDIDATE` / `PRESENT` capsule,
while its source-receipt transform independently required staged HANDOFF to equal
its parent with only the pending receipt block replaced. Sealed U7D necessarily
carried the `U7D` / `SOURCE_CANDIDATE` / `ABSENT` capsule. Source validation also
treated ambient `HEAD` as U7D, which cannot remain true after an interposed repair
boundary. The old self-test exercised capsule parsing and receipt replacement in
separate fixtures and therefore missed the contradictory conjunction.

During the first U7Q attempt, an external Homebrew transaction also upgraded and
removed the declared Didrun Python launcher epoch while cumulative verification
was active. The reinstalled Python 3.14.5 launcher remains at the declared
realpath but has SHA-256
`a533f0d1060b48834eaf7bb41a77d71c9c1f611b31a0a3997ee5a521314bda9d`.
Sealed U7D's source study evidence remains bound to source runtime-authority
SHA-256 `3d99267f2063b9108467f5149233efc64624804fdecb7aa8dd658dbc292bbbe7`.
Those active and source epochs must remain distinct; U7Q cannot reinterpret
sealed evidence under the current recorder epoch.

U7Q repairs exactly those control predicates:

1. U7R becomes a direct child of sealed U7Q.
2. Sealed U7D remains U7Q's exact parent and the immutable receipt source.
3. U7R HANDOFF must differ from sealed U7Q by exactly the canonical
   U7Q-to-U7R capsule substitution and the pending-to-receipt block substitution.
4. `docs/status/U7D-EVIDENCE.md` may differ from sealed U7D only by the exact
   pending-to-receipt block substitution.
5. Both independent validators derive U7D through U7Q's exact parent edge rather
   than ambient `HEAD`.
6. Combined hostile self-tests exercise the capsule and receipt transforms in one
   candidate and refuse every additional, missing, reordered, or stale mutation.
7. Independent runtime authorities bind the active reinstalled launcher while
   receipt validation preserves the exact sealed-U7D source runtime digest.

The exact transition authority is `OWNER_OUT_OF_BAND`, classification
`OWNER_AUTHORIZED_RECEIPT_TRANSITION_AND_RECORDER_RUNTIME_EPOCH_CONTROL_REPAIR`, provenance `UNEVIDENCED`,
authentication `NOT_ESTABLISHED`, signature `NOT_IMPLEMENTED`, and grade transfer
`NONE`. Its exact defect token is
`U7R_PHASE_CAPSULE_TRANSITION_CONFLICTS_WITH_EXACT_PARENT_RELATIVE_PENDING_BLOCK_REPLACEMENT_RECEIPT_SOURCE_VALIDATION_ASSUMES_U7D_IS_AMBIENT_HEAD_AND_EXTERNAL_HOMEBREW_UPGRADE_INVALIDATED_DECLARED_DIDRUN_PYTHON_LAUNCHER_EPOCH`.
Its projection policy is
`HANDOFF_EXACT_PARENT_RELATIVE_U7Q_TO_U7R_CAPSULE_AND_PENDING_BLOCK_REPLACEMENT_STATUS_EXACT_PARENT_RELATIVE_PENDING_BLOCK_REPLACEMENT`.

U7Q creates no product, study, run, artifact, semantic, resource, security,
deployment, receipt, or grade authority. It changes no sealed U7D product or
evidence byte and cannot reinterpret any U7D fact. The U7D receipt stays absent.

## Exact owned roster

All 15 paths are required mode-`100644` tracked blobs:

1. `docs/ARCHITECTURE.md`
2. `docs/CONCEPT_BRIEF.md`
3. `docs/HANDOFF_MODE_C.md`
4. `docs/PROMPT_PACK.md`
5. `docs/STATE_MACHINES.md`
6. `docs/VERIFICATION.md`
7. `docs/prompts/P08-U7-CLI-REFERENCE-STUDIES.md`
8. `docs/status/U7Q-RECEIPT-TRANSITION-ALIGNMENT.md`
9. `spec/verification/u7-unit-paths.json`
10. `tools/check-u7-plan.mjs`
11. `tools/check-u7-scope.mjs`
12. `tools/check-u7-architecture.mjs`
13. `tools/check-u7-architecture-selftest.mjs`
14. `tools/verify-current.mjs`
15. `tools/verify-current-selftest.mjs`

No other path is authorized. In particular U7Q owns no product, study, adapter,
fixture, harness, generic semantic package, contract runner, receipt declaration,
or sealed U7D status byte.

## Exact claim manifest

Every row remains `UNRECEIPTED` in this source candidate:

| Event | Type | Exact label | Grade |
| ---: | --- | --- | --- |
| 0 | tests-pass | U7Q candidate plan and sealed-U7D parent authority | `UNRECEIPTED` |
| 1 | tests-pass | U7Q independent candidate transition and exact staged authority | `UNRECEIPTED` |
| 2 | tests-pass | U7Q inherited P07 receipt successor compatibility | `UNRECEIPTED` |
| 3 | tests-pass | U7Q plan contract defensive self-test | `UNRECEIPTED` |
| 4 | tests-pass | U7Q independent staged-scope defensive self-test | `UNRECEIPTED` |
| 5 | tests-pass | U7Q final runbook renderer defensive self-test | `UNRECEIPTED` |
| 6 | tests-pass | U7Q zero-product-surface architecture conformance | `UNRECEIPTED` |
| 7 | tests-pass | U7Q architecture authority defensive self-test | `UNRECEIPTED` |
| 8 | tests-pass | U7Q cumulative verifier defensive self-test | `UNRECEIPTED` |
| 9 | tests-pass | U7Q cumulative verification on the exact staged maintenance candidate | `UNRECEIPTED` |
| 10 | command-succeeded | U7Q exact fifteen-path staged scope and diff integrity | `UNRECEIPTED` |
| 11 | command-succeeded | U7Q scoped staged credential-pattern scan | `UNRECEIPTED` |
| 12 | command-succeeded | U7Q sealed-U7D predecessor and preceding didrun chain integrity | `UNRECEIPTED` |

## First permanent failed precommit attempt

The first U7Q attempt used tree
`d9a87940aafea5342698477a9a32d3ba9442347b`. Events `0`–`8` exited zero and
were claimed. While intended event `9` cumulative verification was active, an
external `brew install azure-cli` transaction upgraded and removed the declared
Python 3.14.5 launcher epoch. Its wrapper never appended or claimed event `9`;
unreferenced ledger object
`c72b451064b1b89d46cc750744a299964ae73012b7f6dee01c29d50dd3821a08`
contains a `59/70` progression followed by row-60 runtime-path refusal and
`RESULT FAIL`.

A batched operator sequence then launched the intended event `10` before the
cumulative wrapper was established terminal and continued after nonzero. The
intended events `10`/`11`/`12` were recorded unclaimed as ledger events
`9`/`10`/`11`, each refusing the changed Didrun Python runtime at
`U7_SCOPE_DIDRUN_PYTHON_PATH` or `U7_PLAN_DIDRUN_PYTHON_PATH`. Their recorded
intervals are nonoverlapping, but the unclosed-wrapper launch and continued
batched fences permanently invalidate the attempt.

Exact recovery preserves the 12-event/9-claim ledger at
`.didrun-history/u7q-precommit-failed-20260813T224902Z-runtime-drift-unclosed-wrapper-batched-fences/.didrun`
and private root at
`.countershape/u7q-final-precommit-failed-20260813T224902Z-runtime-drift-unclosed-wrapper-batched-fences`.
It created no commit, seal, note, strict result, HTML, normal archive,
product/checker/scope/credential/preseal authority, grade, or receipt. None of
its events or claims transfer to the fresh attempt.

## Verification and nonclaims

The generated U7Q runbook is authoritative for exact argv, environment,
zero-based event/claim ordering, pathspecs, commit, seal, strict report, and
archive rotation. A nonzero evidence command permanently ends that attempt; no
U7D event, claim, or grade transfers. The plain-seal high-entropy exception
remains review-gated and any override is disclosure, not evidence of secret
absence.

U7Q establishes no Linux or Windows behavior, other Node majors, broad imported
repository behavior, hostile containment, network denial, comprehension, review
compression, adoption, maintainability, production readiness, security review,
external-platform behavior, imported-repository timing, full-study resource
bound, HTTP finalized-run/classification authority, two-domain
target/run/classification reproduction, generalized CLI official execution, U7D
source receipt, or U7R self-receipt. It does not rerun either study or the U7D
six-process harness evidence; cumulative verification is control compatibility,
not a new product milestone.

## Continuation

After U7Q commits, seals, exposes its note, passes explicit-commit strict
verification, emits its commit-addressed HTML, rotates its ledger, and passes a
terminal audit, U7R may create exactly three receipt paths. U7R must treat U7Q as
its direct parent and sealed U7D as its receipt source. It may project only the
already-sealed U7D facts and explicit nonclaims, may change no product or control
implementation byte, and cannot receipt itself.
