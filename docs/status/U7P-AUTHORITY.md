# U7P execution-authority bootstrap

## Boundary identity

- **State:** source candidate; not committed, sealed, or receipted.
- **Boundary:** `U7P`.
- **Declared parent:** exact sealed `C6B`.
- **Subject:** `chore: lock U7 execution authority`.
- **Profile:** `SOURCE_FULL`.
- **Product authority:** `NONE`.
- **Product behavior:** `INHERITED_UNREPROVEN`.
- **Grade transfer:** `NONE`.
- **U7D source receipt:** `ABSENT`.
- **Topology:** `C6B -> U7P -> U7A -> U7B -> U7C -> U7D -> U7R`.

U7P is a repository-authority bootstrap, not a product milestone. It freezes the machine-readable topology, exact path and command manifests, independent transition validators, source-architecture oracle, independent study-harness protocol, deterministic final runbook, and cumulative-verifier enrollment that later units must satisfy. It adds no reference-product source and cannot claim that inherited product behavior was re-proved.

## Transition authority and ceiling

The broad P08 roadmap already existed in sealed C6B, but the exact `U7P/U7A/U7B/U7C/U7D/U7R` topology, rosters, profiles, receipt transitions, and claim manifests did not. The transition is therefore classified `OWNER_AUTHORIZED_ROADMAP_ACTIVATION` with source `OWNER_OUT_OF_BAND`.

That source is topology authority only. Its provenance is `UNEVIDENCED` with disclosure `OWNER_ATTRIBUTED_SESSION_INSTRUCTION_ONLY_NO_QUALIFYING_PREEXISTING_ARTIFACT`; authentication is `NOT_ESTABLISHED`, and signed authorization is `NOT_IMPLEMENTED`. Sealed `docs/prompts/P08-U7-CLI-REFERENCE-STUDIES.md` is bound as `PROSE_ONLY` predecessor roadmap direction. It does not authenticate the later instruction or supply the absent exact topology.

No C6B grade or verifier outcome transfers across this edge. U7P records only the exact sealed parent evidence listed below and then re-runs its own declared current-tree verification before it may seal.

## Exact sealed C6B anchor

- Commit: `4cef12b38cfcd857593a21db5952f2dfb2dfc274`.
- Tree: `b50aee79f41c3f498524d89d425d6dd8f4009702`.
- Direct parent: `fafff150d23d6df211b4313e73ac7cf43f28b2d3`.
- Subject: `docs: receipt P07B-C contract execution`.
- Note blob: `a9f3f5484caf00f35e254f71549a754d3de3bd4b`.
- Note-body SHA-256: `355355cd57254802dbc661c4fc9faaf8c695dfbbc787c0175b085f2709f532b3`.
- Claims: `9/9`, each verbatim grade `tree-exact`.
- Strict exit: `0`.
- `secrets_override`: `true`.
- HTML: `.countershape/evidence/p07b-c-c6b-final-4cef12b38cfc.html`, SHA-256 `006e298a8cd8556bb32cee4229df60e3735b102a96c54878c3cf2205cbb48070`.
- Ledger archive: `.didrun-history/p07b-c-c6b-final-4cef12b38cfc/.didrun`.
- Archive closure: 14 regular files, 38,492 bytes, manifest SHA-256 `a09598654ee38382ea44201a7b4609b66c1bee192d1dcd8fc747508527122e37`, 10 events, 9 claims, and 1 seal.

The HTML is a `LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS`; the ledger is a `LOCAL_SECRET_BEARING_SNAPSHOT_NOT_PORTABLE`. Both U7 validators independently reopen the archived ledger, canonical JSONL, entry hash chain, object closure, exact note, seal, commit, tree, subject, and ancestry. They do not invoke live-ledger strict verification for an ancestor from inside a child ledger.

## Exact U7P scope

U7P must stage exactly these 17 paths, in this declared order, with no allowed-versus-required ambiguity:

1. `docs/ARCHITECTURE.md`
2. `docs/HANDOFF_MODE_C.md`
3. `docs/PROMPT_PACK.md`
4. `docs/STATE_MACHINES.md`
5. `docs/VERIFICATION.md`
6. `docs/prompts/P08-U7-CLI-REFERENCE-STUDIES.md`
7. `docs/status/U7P-AUTHORITY.md`
8. `spec/verification/u7-unit-paths.json`
9. `tools/check-u7-plan.mjs`
10. `tools/check-u7-scope.mjs`
11. `tools/check-u7-architecture.mjs`
12. `tools/check-u7-architecture-selftest.mjs`
13. `tools/check-u7-study-harness.mjs`
14. `tools/check-u7-study-harness-selftest.mjs`
15. `tools/print-u7-final-runbook.mjs`
16. `tools/verify-current.mjs`
17. `tools/verify-current-selftest.mjs`

No product, generated contract, fixture, study, or reference-application path is admitted. The candidate and final gates compare the staged index to the exact parent, require a stable clean worktree/index/spec/runtime snapshot, reject extra or missing paths, and scan only the exact staged bytes for credential patterns.

## Frozen continuation

The manifest freezes one direct-child chain:

| Unit | Role | Product ceiling | Receipt U7D |
| --- | --- | --- | --- |
| `U7P` | topology, validators, architecture oracle, runbook, verifier enrollment | `NONE / INHERITED_UNREPROVEN` | `ABSENT` |
| `U7A` | reference CLI foundation | `U7_REFERENCE_APPLICATION / CANDIDATE_UNRECEIPTED` | `ABSENT` |
| `U7B` | HTTP falsification study and its own three-run process/artifact protocol gate | `U7_REFERENCE_APPLICATION / CANDIDATE_UNRECEIPTED` | `ABSENT` |
| `U7C` | CLI decision/contract study and its own three-run process/artifact protocol gate | `U7_REFERENCE_APPLICATION / CANDIDATE_UNRECEIPTED` | `ABSENT` |
| `U7D` | cross-domain process/artifact closure | `U7_REFERENCE_APPLICATION / CANDIDATE_UNRECEIPTED` | `ABSENT` |
| `U7R` | exact source-receipt reconciliation only | `NONE / SOURCE_RECEIPT_RECONCILIATION` | `PRESENT` |

Every future source row has one exact roster, subject, parent, profile, claim order, command argv, final root, receipt state, and pathspec policy. A later candidate may satisfy those bytes but cannot widen or reinterpret them. U7R owns only its exact three receipt-projection paths and cannot change product or verification implementation.

## Runtime and recorder authority

The U7 runtime snapshot binds the current Darwin/arm64 platform and exact local entrypoint bytes for Node 25.2.1, Go 1.26.5, Git, shell, `gofmt`, clang/clang++, `/usr/bin/env`, and the local `/usr/bin/time` observer. It also binds the didrun entrypoint shim, resolved Python launcher, `pyvenv.cfg` with system site packages disabled, the editable import route, closed startup-file surface, recursive package manifest, repository Git configuration, and absent replace refs.

Every recorded child command and every recorder/control-plane invocation uses a fixed `/usr/bin/env -i` environment with private unit roots, fixed locale/time zone/path, disabled Node/Python startup overrides, disabled global/system Git configuration, and replace-object refusal. Tool/runtime bytes are admitted before their version/import probes execute and re-opened terminally.

This remains a local snapshot, not vendor authenticity. Dynamic libraries, SDKs, Python standard-library and third-party transitive dependencies, the dynamic loader, and hostile same-user swap-and-restore remain outside the claim.

## Study-harness authority and ceiling

U7P freezes `countershape/u7-study-harness/v1` before any product-owned study driver exists. The harness, not a later driver, chooses fresh private roots, admits exact tool bytes, inspects raw zero-parent Git fixture authority and exact worktree bytes, compiles the reference binary, creates the empty evidence root only after fixture preparation, directly launches each product binary under the admitted `/usr/bin/time`, reopens the exact artifact roster, and derives deterministic-equality and physical-digest-freshness facts from observed bytes. Its 70-case defensive self-test exercises the protocol and evidence-author separation without upgrading any product result.

This is `U7P_FROZEN_HARNESS_DIRECT_PROCESS_GIT_AND_ARTIFACT_OBSERVATION`. The harness records trial-only logical phase counts and independently observed prepare/compile/execute subject-process wall time and peak RSS. It does not independently recompute the HTTP or CLI semantics expressed by product artifacts, attribute time or memory to each logical study phase, or observe total parent-harness resource use. Those ceilings are machine-labeled `ARTIFACT_BYTES_AND_SUBJECT_PROCESS_TOPOLOGY_OBSERVED_PRODUCT_SEMANTICS_AND_FULL_HARNESS_RESOURCES_NOT_INDEPENDENTLY_ESTABLISHED`; `FULL_STUDY_RESOURCE_BOUND_UNVALIDATED` remains an explicit nonclaim.

## Architecture authority

U7P freezes an independent phase-aware source oracle. At U7P it requires all future `cmd/countershape`, `internal/reference`, `testkit/reference`, and product-owned U7 study-runner surfaces to be absent. Its hostile self-test freezes the future U7A–U7D cumulative surface rules before those files exist: exact governed inventories, package declarations, phase-specific composition imports, matching-domain adapter/fixture imports, typed-result reproduction edges, byte-exact inherited semantic packages/oracles, parsed JavaScript with a closed module grammar and enumerated runtime-resolver refusals, selected raw process/network ownership, cross-domain identifier/import refusals, and selected authoritative Go type/constant name ownership across production and governed tests.

Passing this source oracle is not a Go type-system proof, proof that an imported fixture is used correctly, proof that no equivalent second truth model exists under arbitrary names, proof that every reflective JavaScript capability is absent beyond the enumerated resolver set, runtime containment proof, network-denial proof, product receipt, or security review. It is deterministic source/import/name-ownership evidence over the exact admitted tree.

## Exact U7P claim manifest

Each claim binds the zero-based successful event index and all 17 exact pathspecs in declared order. The final ledger must contain exactly these 13 command/claim pairs:

| Event | Type | Exact label | Command |
| ---: | --- | --- | --- |
| 0 | `tests-pass` | `U7P candidate plan and sealed-C6B parent authority` | `node tools/check-u7-plan.mjs --check-candidate U7P` |
| 1 | `tests-pass` | `U7P independent candidate transition and exact staged authority` | `node tools/check-u7-scope.mjs --unit U7P --candidate-phase` |
| 2 | `tests-pass` | `U7P plan contract defensive self-test` | `node tools/check-u7-plan.mjs --self-test` |
| 3 | `tests-pass` | `U7P independent staged-scope defensive self-test` | `node tools/check-u7-scope.mjs --self-test` |
| 4 | `tests-pass` | `U7P final runbook renderer defensive self-test` | `node tools/print-u7-final-runbook.mjs --self-test` |
| 5 | `tests-pass` | `U7P dormant future-surface architecture conformance` | `node tools/check-u7-architecture.mjs --phase U7P` |
| 6 | `tests-pass` | `U7P architecture authority defensive self-test` | `node tools/check-u7-architecture-selftest.mjs --phase U7P` |
| 7 | `tests-pass` | `U7P study-harness protocol defensive self-test` | `node tools/check-u7-study-harness-selftest.mjs --phase U7P` |
| 8 | `tests-pass` | `U7P cumulative verifier defensive self-test` | `node tools/verify-current-selftest.mjs` |
| 9 | `tests-pass` | `U7P cumulative verification on the exact staged candidate` | `node tools/verify-current.mjs` |
| 10 | `command-succeeded` | `U7P exact seventeen-path staged scope and diff integrity` | `node tools/check-u7-scope.mjs --unit U7P --final-gate` |
| 11 | `command-succeeded` | `U7P scoped staged credential-pattern scan` | `node tools/check-u7-scope.mjs --unit U7P --credential-scan` |
| 12 | `command-succeeded` | `U7P sealed-C6B predecessor and preceding didrun chain integrity` | `node tools/check-u7-plan.mjs --verify-preseal U7P` |

The table abbreviates only the absolute executables and common hermetic prefix for readability; `spec/verification/u7-unit-paths.json` is the command authority. The generated runbook renders every full argv, event index, and pathspec exactly.

## Receipt state

| Claimed capability | Grade |
| --- | --- |
| U7P candidate plan and sealed-C6B parent authority | `UNRECEIPTED` |
| U7P independent candidate transition and exact staged authority | `UNRECEIPTED` |
| U7P plan contract defensive self-test | `UNRECEIPTED` |
| U7P independent staged-scope defensive self-test | `UNRECEIPTED` |
| U7P final runbook renderer defensive self-test | `UNRECEIPTED` |
| U7P dormant future-surface architecture conformance | `UNRECEIPTED` |
| U7P architecture authority defensive self-test | `UNRECEIPTED` |
| U7P study-harness protocol defensive self-test | `UNRECEIPTED` |
| U7P cumulative verifier defensive self-test | `UNRECEIPTED` |
| U7P cumulative verification on the exact staged candidate | `UNRECEIPTED` |
| U7P exact seventeen-path staged scope and diff integrity | `UNRECEIPTED` |
| U7P scoped staged credential-pattern scan | `UNRECEIPTED` |
| U7P sealed-C6B predecessor and preceding didrun chain integrity | `UNRECEIPTED` |

Development greens, generated runbook text, the sealed parent receipt, and this status prose cannot substitute for the exact final ledger. No U7P grade exists until its own boundary closes.

## Inherited P07 verifier-epoch repair

Routing: `DEFECT_OBSERVED_IN_INHERITED_VERIFIER_EPOCH`.

The first U7P cumulative development run reached the inherited P07 plan self-test only after 59 earlier rows had passed, then failed on the initial verifier-integration and descendant-document assumptions. After those defects were repaired, focused event 101 exposed the separate historical-fixture contradiction. Sealed C6B used the nine-claim receipt-reconciliation profile and therefore did not run either `tools/check-p07b-c-plan.mjs --self-test` or `tools/check-p07b-c-c3p-receipt.mjs --self-test` against its own post-receipt HANDOFF. The latter transitively invokes the same fixed historical phase-table replay. That replay's C3Q fixture deletes C6 source authority and evidence while retaining the live sealed C6A receipt projection; the checker correctly rejects the resulting impossible state. Running either self-test unchanged on an honest descendant cannot establish the row it names.

U7P does not weaken either self-test, edit protected P07 bytes, delete permanent development failures, or relabel either row as passing. Both rows are retained adjacently in the verifier's historical-only roster; the ordinary live C3P receipt check remains current. The resulting roster is exactly 71 current rows and 14 historical-only rows. Their live successor is explicitly narrower: `tools/check-u7-plan.mjs --verify-inherited-p07-compatibility` reruns the current U7 candidate authority, extracts exactly 11 visible P07 receipt/phase blocks from staged HANDOFF and exact sealed C6B, requires byte equality for every block and the complete P07 marker roster, and requires byte equality for `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-c3p-receipt.mjs`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Nested candidate validation remains load-bearing but suppresses its own success narration in this composite command. The cumulative verifier requires the one exact compatibility stdout line, exact counts/state token, empty stderr, status zero, and no signal. Its success cannot be read as a grade for either retired self-test.

## Final-ledger lifecycle repair

Routing: `DEFECT_OBSERVED_IN_U7P_RUNBOOK_RECOVERY`.

The first rendered U7 runbook created a private final root that its own fresh-attempt preflight later required absent, yet did not provide an executable preservation/cleanup transition after a failed command. The repaired runbook owns two disjoint restartable state machines. Normal success requires an exact clean committed workspace, no failed ref/archive/final-root marker, same-device no-follow destination-absent renames, and terminal reopening of the complete archived note, chain, object closure, seal, and claims. Its verified rotation command is terminal in the fence.

Any nonzero permanently ends the ledger. Before commit, a unique UTC/reason recovery preserves an optional ledger plus the hermetic final root. At or after commit, exact-full-commit recovery normalizes a live, partially normal-archived, or fully normal-archived ledger into failed paths; preserves the final root; creates or revalidates `refs/countershape/failed-attempts/<unit>/<commit>`; and compare-and-swaps HEAD from the failed commit to the declared parent while preserving the failed tree in the index. Both commands preclassify their whole transition before mutation and are rerunnable after interruption. Same-UID races, directory-fsync guarantees, and power-loss durability beyond the observed rename/ref transitions remain unreceipted.

## Development evidence and S6 finding

Permanent unclaimed development event 94 ran the full cumulative verifier on staged tree `dae4eec5900a5ae25babc4d174749c5cf77a7b73` and exited `1` after surfacing verifier/U7-integration documentation defects; those bytes were repaired rather than replayed or relabeled. Permanent unclaimed event 101 ran the inherited P07 plan self-test on staged tree `76a3fc0c4528ecfa5ffd83b06c3db26b00469368` and exited `1` on the successor-incompatible fixed C3Q projection described above. Event 102 independently ran the inherited P07 scope self-test on that tree and exited `0`; it is development evidence only and carries no U7P grade. Permanent unclaimed development event 139 ran the 72-row cumulative verifier on staged tree `9efdb98c572e24be17699707ca9c781b151f89b1`, passed rows 1 through 59, and exited `1` at row 60 because the verifier admitted only the compatibility line while the composite checker also narrated its nested candidate success. The repair keeps nested candidate validation load-bearing while suppressing only that nested success narration, restoring the future-phase-safe one-line compatibility contract; the failed receipt remains permanent and is not relabeled.

The later pre-interval-repair cumulative event ran on exact staged tree `036b1c97533bc2de21befba96e00e1892429452b`, passed rows 1–62 of the then-72-current-row roster, and exited `1` at row 63, `architecture-p07b-c-c3p-receipt-selftest`, proving that the C3P wrapper was a second transitive entrypoint into the same successor-incompatible replay. Its complete unclaimed ledger is preserved at `.didrun-history/u7p-development-cumulative-pre-interval-repair-036b1c97533b/.didrun`. U7P retains the ordinary live C3P receipt check, moves only the incompatible self-test row to historical-only, and does not reuse any earlier green.

The first focused retry after that repair is preserved at `.didrun-history/u7p-development-red-verifier-selftest-syntax-c904201a7c30/.didrun`. Four syntax events passed on staged tree `c904201a7c30a55c05e4e95af61150bd0407986f`; the fifth exited `1` because the newly added historical-row reorder hostile used method-call expressions as destructuring assignment targets. The source was corrected to swap array elements by numeric indexes. All five events remain unclaimed and nonreusable.

The next focused ledger is preserved at `.didrun-history/u7p-development-red-verifier-roster-hostile-order-b277173f233b/.didrun`. Its first 15 events passed on staged tree `b277173f233b8b97bad02ea3938a2e8939da77e3`; event 15 exited `1` because removing the second retired row reached the adjacency diagnostic before the hostile's expected cardinality diagnostic. The verifier self-test now establishes both retired-row cardinalities before exact-row and adjacency checks. Every event in the failed ledger remains unclaimed and nonreusable.

Exact staged tree `39be44b29e59e045dfb17e8f0e8c7c12b49e483f` subsequently passed a 16-event focused ledger and a separate full cumulative event. The focused chain is preserved at `.didrun-history/u7p-development-focused-green-39be44b29e59/.didrun`. The cumulative chain is preserved at `.didrun-history/u7p-development-cumulative-green-39be44b29e59/.didrun` and recorded `RESULT PASS current=71/71 historical_not_run=14`, exit `0`, in `8,205.490` seconds. Both remain unclaimed development evidence; neither substitutes for final command 9 on the eventual exact final tree.

The first generated final preparation attempt produced no didrun ledger, event, or claim. The prepare control environment placed `HOME` and `TMPDIR` beneath the absent final root; before the absence assertion, the admitted Go version probe created `home/Library/Application Support/go`, causing the gate to refuse exactly. The no-ledger residue is preserved at `.countershape/u7p-preflight-bootstrap-failed-home-probe-39be44b29e59`. Routing: `DEFECT_OBSERVED_IN_U7P_RUNBOOK_BOOTSTRAP`. The repaired renderer runs prepare from the stable `.countershape` recovery/control environment and pins both the required command and the absence of the final-root-prefixed form before creating the unit root.

That repaired preflight succeeded and then exposed a second no-ledger bootstrap defect: `install -d` named only the six private child paths, so it created their intermediate unit root with caller-default mode `0755`; the U7 recovery machine correctly requires exact mode `0700`. The unchanged residue is preserved at `.countershape/u7p-prepare-parent-mode-failed-9fe8c61cb68f`. Routing: `DEFECT_OBSERVED_IN_U7P_PRIVATE_ROOT_CREATION`. The renderer now names the parent before all children in the exact mode-`0700` installation command, self-tests that command, and the success archive preflight independently enforces exact parent mode.

The first focused rerun under mandatory caller `umask 077` then stopped at zero-based event 12: the synthetic study drivers requested creation mode `0644`, became `0600`, and were correctly refused with `U7_STUDY_HARNESS_DRIVER_MODE`. The terminal 13-event ledger is preserved unclaimed at `.didrun-history/u7p-development-focused-red-caller-mode-driver-acadee5b3c99/.didrun`. Routing: `DEFECT_OBSERVED_IN_U7_STUDY_SELFTEST_CALLER_UMASK`. The self-test now chmods only each newly created synthetic driver to exact `0644` before the pre-existing `driver-mode` hostile deliberately returns one to `0600`; mandatory caller `077` and the harness gate are not weakened.

The next focused run passed zero-based events 0–16 and then the scoped credential scan correctly rejected the former archive basename in three staged documents because adjacent basename characters matched the OpenAI-key detector. The terminal ledger is preserved unclaimed at `.didrun-history/u7p-development-focused-red-credential-doc-path-2e4347e68f3c/.didrun`; the earlier archive was identity-preservingly renamed to the caller-mode spelling above, and the credential scanner is unchanged. Routing: `DEFECT_OBSERVED_IN_U7_DOCUMENTATION_CREDENTIAL_PATTERN`.

The same review found that the claimed exact-mode final-root admission used `mode & 0777`, which could admit sticky, setgid, or setuid variants. Routing: `DEFECT_OBSERVED_IN_U7_PRIVATE_MODE_AUTHORITY`. The shared success/recovery helper now compares `mode & 07777` to exact `0700`; its self-test rejects `0755`, `01700`, `02700`, and `04700` before accepting one exact private directory.

One development mistake launched two didrun writers concurrently. Both child commands exited zero, but didrun 0.1.0 assigned the same event index to both appenders, producing a broken ledger chain. The raw ledger is preserved without edits at `.didrun-history/u7p-development-concurrent-writer-chain-broken-20260808/.didrun` and supports no claim or grade.

A later long study-harness self-test exposed a separate app-session failure mode: its command cell returned without surfacing the still-live unified command session. Later development writers overlapped that wrapper, and its delayed event recorded `tree_before=4386736aa9eb79f5dd55dae6abc070190adee6ac` and `tree_after=d76919341fd5027fcfd12e6b7e79a247a39bdff6`. The chain remained linked, but tree drift and writer overlap make the whole ledger non-authoritative. It is preserved without edits at `.didrun-history/u7p-development-concurrent-tree-drift-20260808/.didrun` and supports no claim or grade.

Routing: `DEFECT_OBSERVED_IN_TOOLING` and `DEFECT_OBSERVED_IN_U7P_LEDGER_INTERVAL_AUTHORITY`. The working mitigation is one root-owned didrun/repository/claim/Git/seal/recovery/archive writer at a time; read-only critics may run concurrently, but no subagent writes Git or didrun evidence. A yielded long command remains active until both its didrun wrapper and relevant child process are confirmed terminal; an empty app-cell result is not terminal evidence. The plan and independent scope validators now also require finite, nonreversed, ledger-ordered, nonoverlapping recorded child intervals for U7 sessions; U7 archive/local-evidence validation includes the final unclaimed note-inspection event, while sealed C6B is not retroactively reprofiled. This establishes only ordering of didrun's recorded wall-clock intervals around top-level children. It does not establish wrapper, claim, Git, seal, recovery, archive, or detached-descendant nonoverlap; clock monotonicity/authenticity; a cross-process lock; or same-UID ABA resistance. These remain live S6 findings that didrun 0.1.0 lacks a cross-process writer lock and that the app can lose the continuation handle for a still-running nested command. U7P does not upgrade didrun mid-boundary and does not claim the mitigation repairs either implementation.

## Honest nonclaims

U7P does not establish product behavior, study execution, product-semantic reproduction, whole-harness resource bounds, determinism, physical freshness, hostile containment, network denial, confidentiality, human comprehension, review compression, maintainability, adoption, production readiness, external-platform behavior, Linux/Windows behavior, unrun Node majors, broad imported-repository behavior, security review, or the authenticity of the owner-attributed topology instruction. Those remain explicit later work or owner/market obligations.
