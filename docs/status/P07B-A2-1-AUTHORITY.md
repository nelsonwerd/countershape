# P07B-A2.1 current-ruling compilation-authority receipt

- **State:** implemented as one bounded A2.1 authority unit; the commit containing this file is accepted only when its didrun chain is intact, its commit is sealed, and `NO_COLOR=1 didrun verify --strict` exits `0`
- **Prerequisite:** sealed P07B-A1 source commit `1e56da3bfafc4cdb8abdca62f18fb3b1accea8c3`, tree `da0e13f37aacd0e7fb2b52b3aeac628f5558c11e`
- **Receipt ceiling:** current-ruling/source/confirmation/proof compilation preparation only; no compiler output, generated file, bundle, residue, publication, materialization, execution, or product surface
- **Grade rule:** the table below copies didrun's verbatim `TREE-EXACT` grade only after strict verification of the sealed commit; a receipt records what ran and is not a proof of correctness or completeness

## Implemented boundary

P07B-A2.1 adds the narrow authority seam that A1 intentionally omitted:

- `ChoicepointRecord` exposes defensive exact `WorldPlan` and minimized-stimulus views without changing its canonical body;
- `FreshExecutionRecord` and `confirmation.Record` retain defensive execution-binding digests, with the confirmation roster reconstructed in schedule order from the existing 38-member wire and checked again by `Valid()`;
- promotion opens a current-revalidated inert `PortableCompilationSnapshot` that reparses and rejoins the current DecisionRecord, Choicepoint, FreshConfirmation, fresh ruling inspection, and exact portable-profile bytes without exposing a head token;
- the new pure node model owns exact portable values, ordered fields, complete correlated tuples, an unsigned-canonical tuple set, and a seven-member declared emitter `SourceProfile` under the exact `ContractSourceProfile` digest domain;
- private `internal/emit/node/internal/compilation.Input` is sealed and can be constructed from exactly one production call site;
- `PrepareCompilation` exact-reparses the A1 source, joins its plan/projection/stimulus and every retained confirmation binding, independently retranslates reopened proof records, revalidates the ruling partition and selected-field separation, and returns one sealed `PreparedCompilation`;
- the wrapper privately retains the original portable ruling preparation for later P07B-B revalidation while exposing only defensive inert summaries; and
- production A2.1 performs no publication, head transition, direct filesystem I/O, process execution, Git operation, clock/environment/runtime inspection, randomness, network access, or generated-source assembly beyond the owning store's required current-object reopen.

“Sanitized” means authority-narrowed, not content-redacted. The exact `PortableSource` and predicate values remain recoverable and may contain sensitive or identity-looking behavior bytes. The input adds no concrete candidate keys, refs, aliases, producer metadata, support counts, confirmation receipts, or injected host-runtime facts outside the exact source and frozen declared profile.

## Exact physical and compatibility scope

- CLI `ALLOW_OBSERVED` prepares exactly the correlated tuples `(argv, argv)` and `(config, config)` over `cli.stdout.json.mode` plus `cli.stdout.json.source`; it does not authorize their Cartesian cross-pairs or `cli.stdout.bytes`.
- HTTP exercises both one reviewed custom status `401` tuple and a separate `ALLOW_OBSERVED` status/body-kind tuple under the child-bind lineage.
- Both studies capture the pre-operation head and a device/inode/mode/size/modtime/content inventory, then require the full success/refusal/restart matrix to leave the store unchanged. This is scoped native evidence plus an architecture prohibition, not proof that an arbitrary filesystem can never hide a transient event.
- Both restart checks launch the current test executable in a fresh process, with a distinct working directory and one closed path-only environment entry. The child reparses the source, reopens the store, reissues the preparation, and returns exact compilation/DecisionRecord/Choicepoint/source/profile identities plus ordered tuple bytes through a private bounded protocol.
- The behavioral cross-study source matrix contains two independently strict sources: a same-plan/projection/profile source whose minimized stimulus and execution binding differ, and a same-adapter/runner/projection/profile/stimulus/capture source whose WorldPlan schedule/budget authority and consequent execution binding differ. Both return exact `SOURCE_RULING_MISMATCH`, a zero prepared capability, and no store change. Wrong-store and predecessor authority are separately refused with the existing typed store error.
- A same-adapter different-runner source is not constructible because the closed source constructor admits only the adapter's required runner. Profile, entrypoint/start, HTTP readiness/capture, proof, selected-field, and partition drift remain coupled upstream-constructor plus exact-architecture obligations rather than independently behaviorally receipted branches; no backdoor constructor was added to fake them.
- Legacy Choicepoint, DecisionRecord, confirmation, projection, and A1 source canonical bodies remain unchanged. `PreparedCompilation.Valid()` proves construction integrity only; it does not promise the retained ruling is still current after return.

## Verbatim receipt map

The sealed manifest is authoritative. Anything outside this table is `UNRECEIPTED` for A2.1.

| Claimed capability | didrun claim label | Verbatim grade |
| --- | --- | --- |
| Go source formatting | `P07B A2.1 Go formatting` | `TREE-EXACT` |
| Node verification-tool syntax | `P07B A2.1 verification tool syntax` | `TREE-EXACT` |
| Focused authority, model, roster, and legacy compatibility suites | `P07B A2.1 focused authority compatibility suites` | `TREE-EXACT` |
| Physical source/ruling cross-pair, no-side-effect, and fresh-process restart matrix | `P07B A2.1 physical source ruling cross-pair and fresh-process restart matrix` | `TREE-EXACT` |
| Complete Go repository suite | `P07B A2.1 complete Go repository suite` | `TREE-EXACT` |
| Complete Go vet | `P07B A2.1 complete Go vet` | `TREE-EXACT` |
| Complete race-enabled Go repository suite | `P07B A2.1 complete Go race suite` | `TREE-EXACT` |
| Layered A1 plus A2.1 dependency/API/dataflow architecture boundary | `P07B A2.1 exact architecture boundary` | `TREE-EXACT` |
| Clean controls plus 57 defensive architecture copies | `P07B A2.1 architecture 57-case defensive self-test` | `TREE-EXACT` |
| Mutation roster, module closure, receipt binding, restoration, and tamper self-test | `P07B A2.1 mutation harness receipt and tamper self-test` | `TREE-EXACT` |
| Fourteen required non-equivalent faults caught with fresh baseline/change/post-control copies | `P07B A2.1 fourteen-fault fresh A/B/A closure` | `TREE-EXACT` |
| Bounded one-worker exact-value construction fuzzing | `P07B A2.1 bounded exact construction fuzz` | `TREE-EXACT` |
| Exact staged path inventory plus staged whitespace/diff integrity | `P07B A2.1 exact staged inventory and diff check` | `TREE-EXACT` |
| Named structured credential-pattern scan over the exact staged blobs | `P07B A2.1 scoped staged structured credential-pattern scan` | `TREE-EXACT` |
| Serialized didrun ledger chain integrity | `P07B A2.1 didrun chain intact` | `TREE-EXACT` |

The claim-bearing mutation run admits explicit Go, C compiler, Node, and Git executables; uses V8 module parsing plus the exact fingerprinted Node runtime's bundled Acorn AST to bind every declared static local ESM edge, reject local side-effect imports outside the frozen module roster, reject dynamic imports, and reject unsupported local extensions; binds that complete declared runner-module closure into its seed; requires exact replacement anchors and one named killer; uses three distinct private copies per fault; requires `PASS / NAMED_TEST_FAILURE / PASS`; and prints bounded per-phase manifest, command, and environment digests. Its terminal line is `P07B A2.1 mutation gate: 14/14 required mutants killed; 14/14 fresh A/B/A receipts.` This is a tested fault roster, not mutation completeness or hostile-code containment. The private copies and trusted physical studies retain the invoking user's host filesystem and network authority.

## Permanent negative history

Nonzero development events remain in the complete unsealed development ledger preserved at `.didrun-history/2026-07-16-p07b-a2-1-preseal-mixed/.didrun/` and support no capability. They record, among other repairs, incomplete accessors and tests, architecture-checker dataflow/roster gaps, physical-test permission and golden mismatches, absent or unsuitable C-compiler admission, one noncompiling changed build, stale frozen roster digests, and the full-loop `ENOBUFS` caused by an unbounded private-structure failure diagnostic. The latter was repaired by retaining the assertion while replacing whole-graph printing with bounded validity, authority-digest, action, count, preview, and tuple-hash diagnostics; both unchanged physical controls and the complete fresh-copy regression gate then passed. The first complete race event produced no race report but exposed an inadequate 12-second child-process timeout in both physical restart helpers; the semantic assertions remained unchanged while the harness stayed bounded at a 90-second child timeout beneath a two-minute parent guard, and the focused race-instrumented physical matrix then passed. The first complete 14-fault command was mistakenly launched without loopback escalation: its first thirteen non-listener faults proceeded, while the final HTTP baseline was denied its local listener and therefore was not a caught mutant. The map assertion remained exact, bounded per-trial control diagnostics were added, ten consecutive approved-loopback HTTP owner tests passed with the unchanged declared readiness/probe/teardown budgets, and the complete closure was rerun through the approved local-loopback path. No failed receipt was deleted, reclassified as a catch, or used as positive evidence.

One earlier zero-exit development event directly selected the two private restart-helper test names without supplying their closed protocol environment, so both helpers intentionally returned without exercising restart behavior. Its overbroad claim label is immutable archived-ledger history and supports no row in the receipt map; only the final-ledger owner-test event bearing the exact mapped label is positive physical evidence.

The pre-seal audit found that the chain-intact development ledger had no prior seal watermark and contained claims across superseded tree identities. Sealing it would therefore have included honestly stale claims in the A2.1 manifest. The whole ledger was preserved unchanged at the path above before a fresh serialized final ledger reran every receipt row; no event or claim was copied, deleted, relabeled, or promoted. This was an operator ledger-lifecycle correction caught before sealing, not a didrun grade override. After the shutdown/resume boundary, the archived ledger also records one focused-suite event that exited before tests because the reboot had removed its named private `/private/tmp` scratch directories; recreating only those ignored scratch directories let the unchanged command pass, and no code, assertion, or timeout was changed.

## Explicit nonclaims

P07B-A2.1 does **not** establish:

- compiler correctness, compiler output, source emission, Go/Node parity, a six-file recoverable `ContractBundle`, manifest recovery, or generated-program behavior;
- `RESIDUE`, store publication, a terminal head transition, stale-before-publication closure, retryable materialization, target inventory, a finalized run, execution classification, or Countershape absence;
- currentness after `PrepareCompilation` returns, global semantic equivalence, completeness of the architecture or mutation rosters, or that every impossible cross-pair was made behaviorally constructible;
- byte-for-byte exposure or cross-process comparison of the private sanitized-input body; restart evidence compares its digest plus every exposed deterministic identity, selected field, and exact tuple byte;
- secret absence, content redaction, confidentiality, sandboxing, hostile-code isolation, network denial, listener ownership, exact post-fingerprint executable inode use, or an independent security review;
- Linux, Windows, other architectures, other toolchain versions, cross-platform runtime portability, production readiness, adoption, human comprehension, legal/trademark clearance, release readiness, or maintainership; or
- any didrun conclusion stronger than the verbatim grade recorded above.

The structured staged scan covers only its named pattern set and does not authorize publication of the secret-bearing didrun ledger or replace a human release secret review.

## Mandatory maintenance, then sole next feature boundary

Only after this A2.1 commit is sealed, its chain is intact, and strict verification exits `0` may a fresh context execute the operator-supplied cumulative-verification maintenance request at `/Users/drewnelson/.codex/attachments/fac7f825-1946-4bcc-880a-1feb4d1b7a40/pasted-text.txt`. That maintenance work is a separate exact-scope commit and must itself be sealed and strict-clean. Only then may P07B-A2.2 execute from `../prompts/P07B-A2-2-RECOVERABLE-COMPILER.md`. A2.2 may consume the private sanitized input to build the pure deterministic recoverable compiler and Go/Node corpus. It still may not publish residue or materialize a target. P07B-B publication/materialization and P07B-C target/run/execution remain later independently committed, sealed, strict-clean units.
