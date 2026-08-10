# U7A reference CLI foundation

## Boundary identity

- **State:** source candidate; not committed, sealed, or receipted.
- **Boundary:** `U7A`.
- **Declared parent:** exact sealed `U7M`.
- **Subject:** `feat: add U7 reference CLI foundation`.
- **Profile:** `SOURCE_FULL`.
- **Product authority:** `U7_REFERENCE_APPLICATION`.
- **Product behavior:** `CANDIDATE_UNRECEIPTED`.
- **Grade transfer:** `NONE`.
- **U7D source receipt:** `ABSENT`.
- **Next boundary:** `U7B` only after exact U7A seals.

Exact parent U7M is commit `fc70bcaa51fec3991d139796c85495b6c1c75735`, tree `cfc84fca4e32b9d610f3b23458c30ab8dfcf8db1`, and subject `fix: adapt inherited P07 receipt verification`. Its local note blob is `e53cc6a9712b1ee1e064e43a53e2f355b7017788`; its commit-addressed archive is `.didrun-history/u7m-final-fc70bcaa51fe/.didrun`. Explicit-commit strict verification reported `13/13 claims recorded-exact`, and its 8,865-byte HTML witness has SHA-256 `a490d63f79da89f1e3616e2f30a451789d35b98100655e8e674f198b29687bef`.

U7A inherits no U7 product grade. Its own source and tests begin `UNRECEIPTED` and cannot grade themselves.

## Preserved failed attempt and fresh restart

The first U7A final attempt reached unclaimed event 8 on exact candidate tree `c65753ad3808c71eb8164b0761bf2dcf258226f7`. Cumulative verifier rows 1–61 passed and row 62 exposed the inherited C3P successor-subject defect that U7M subsequently repaired. That failed ledger is permanently preserved at `.didrun-history/u7a-precommit-failed-20260809T181519Z-cumulative-verifier/.didrun`; its private final root is preserved at `.countershape/u7a-final-precommit-failed-20260809T181519Z-cumulative-verifier`. No event 8 claim, commit, seal, note, strict result, HTML report, or reusable grade exists.

The ten product/test blobs were seeded byte-for-byte from checkpoint ref `refs/countershape/checkpoints/u7a-candidate-pre-u7m` at commit `91a8a74fb4be80582c8fe6f8c09ab705a98a8806`; the checkpoint's stale U7P-era HANDOFF and status blobs were not reused. A pre-ledger U7A review then repaired only the owned renderer and test bytes for hostile diagnostic-path terminal controls, as recorded below. This candidate regenerates both governance documents against sealed U7M. Its final evidence must start at fresh event zero and rerun all twelve commands and claims; no prior event, claim, ledger object, or grade may be reused.

## Locked product slice

U7A is an inert application foundation, not a study milestone. It implements:

- a thin `cmd/countershape` composition root that delegates once to `app.Run`;
- deterministic `help`, `validate`, and `preflight` command grammar;
- regular-file or bounded-stdin SourceSpec input with stable identity checks;
- strict authority-bearing validation through `internal/spec.ParseSource`;
- presentation projection only after accepted canonical bytes are reopened;
- one typed ordinary JSON envelope and a monochrome human renderer;
- a non-executing preflight report with the exact trusted-code warning, declared/core-default cost ceilings, and unresolved authority fields;
- stable refusal categories, stdout/stderr separation, and one safe next action for every non-help terminal state; and
- actual compiled-binary end-to-end tests plus reentrant race coverage.

U7A does **not** discover Git refs, materialize candidates, start a subject, create study evidence, observe or compare outcomes, reduce stimuli, create a Choicepoint, record a ruling, emit or run a contract, select a latest study, or resume a process. A preflight exit of zero means only that the report was produced; `execution_authorized` remains false.

## Exact CLI surface

```text
countershape help
countershape validate --spec <path|-> [--json]
countershape preflight --spec <path|-> [--json]
```

The ordinary machine envelope is `countershape-cli/v1`. It carries supplied typed facts and never creates semantic authority. Human errors go to stderr; ordinary JSON responses are exactly one JSON object plus LF. Neither mode emits ANSI/OSC control sequences. The exact warning is:

> Countershape will run trusted repository code with your user permissions and host network access. The temporary directory is for repeatability, not security isolation.

The warning is presentation in U7A because no execution-capable route exists. Before any later human execution it must precede the spawn. The frozen U7 study harness is a distinct machine protocol: later `study http --json` and `study cli --json` executions require one exact four-field domain-result JSON line and empty stderr. That protocol is intentionally not wrapped in the ordinary envelope; its process/artifact evidence and semantic ceiling remain governed by U7P's harness contract.

## Strict-input authority

Raw SourceSpec bytes are never admitted by ordinary JSON decoding. `internal/spec.ParseSource` is the first authority-bearing constructor. Only after it returns an opaque parsed capability does U7A reopen its accepted canonical bytes with the strict canonical parser to build an inert display summary. The summary uses the parser-issued source digest and exported core budget defaults; it cannot compile a plan or resolve a projection reference.

The frozen behavioral suite covers duplicate keys, negative zero, unsafe integers, lone surrogates, invalid UTF-8, trailing data, unknown members, missing/duplicate/unknown CLI arguments, the 1 MiB input limit, symlink refusal, source-byte non-echo, typed refusal codes, exact warning text, JSON framing, no terminal controls, supplied-fact preservation, broken writers, and concurrent independent invocations.

## Exact owned roster

All twelve paths are required, regular mode-`100644` tracked blobs and no other staged path is authorized:

1. `cmd/countershape/main.go`
2. `internal/reference/app/command.go`
3. `internal/reference/app/envelope.go`
4. `internal/reference/app/render.go`
5. `internal/reference/app/workspace.go`
6. `internal/reference/app/spec.go`
7. `internal/reference/app/validate.go`
8. `internal/reference/app/preflight.go`
9. `internal/reference/app/app_test.go`
10. `internal/reference/app/cli_e2e_test.go`
11. `docs/HANDOFF_MODE_C.md`
12. `docs/status/U7A-CLI-FOUNDATION.md`

The HANDOFF edit is outside every frozen P07 marked block. No U7D source-receipt marker or pending receipt block exists.

## Claim manifest

Every grade remains `UNRECEIPTED` in this source candidate:

| Event | Type | Exact claim label | Current grade |
|---:|---|---|---|
| 0 | tests-pass | U7A candidate plan and sealed-U7M parent authority | `UNRECEIPTED` |
| 1 | tests-pass | U7A independent candidate transition and exact staged authority | `UNRECEIPTED` |
| 2 | tests-pass | U7A plan contract defensive self-test | `UNRECEIPTED` |
| 3 | tests-pass | U7A independent staged-scope defensive self-test | `UNRECEIPTED` |
| 4 | command-succeeded | U7A exact-owned Go formatting | `UNRECEIPTED` |
| 5 | tests-pass | U7A reference CLI foundation tests | `UNRECEIPTED` |
| 6 | tests-pass | U7A reference CLI race tests | `UNRECEIPTED` |
| 7 | command-succeeded | U7A reference CLI vet | `UNRECEIPTED` |
| 8 | tests-pass | U7A cumulative verification | `UNRECEIPTED` |
| 9 | command-succeeded | U7A exact staged scope and diff integrity | `UNRECEIPTED` |
| 10 | command-succeeded | U7A scoped staged credential-pattern scan | `UNRECEIPTED` |
| 11 | command-succeeded | U7A sealed-U7M predecessor and preceding didrun chain integrity | `UNRECEIPTED` |

The final attempt must use only the generated U7A runbook and its exact zero-based event/pathspec binding. Any nonzero permanently ends that ledger and uses the generated matching recovery fence.

## Red-team dispositions and ceilings

1. **Phase-only foundation:** accepted and implemented. Production app code imports only inert standard-library facilities plus the strict `canon/spec/domain` authority needed to validate and classify refusals. It does not use the effectful packages that the sealed source oracle could syntactically admit. Built-binary tests exercise the actual thin main delegation.
2. **Renderer recomputation:** defect risk addressed in product tests. Both renderers consume one precomputed `ResponseEnvelope`; a deliberately inconsistent supplied envelope must be preserved rather than “corrected.” The sealed architecture oracle remains intentionally bounded source/name evidence, not complete Go behavior or dataflow proof.
3. **Status semantics:** checker limitation disclosed. The status file is not content-parsed by the current sealed-U7M validator set, so this substantive contract and its test evidence are load-bearing craft, not a machine-proven document-truth claim.
4. **Parent strict/HTML transfer:** intentional operational ceiling. Descendant admission independently validates the parent commit, note, scope, and archived ledger rather than transferring a local HTML grade. Current U7M strict/HTML witnesses are named above; their continued local presence is not cryptographically established by the U7A ancestry edge.
5. **Harness warning contradiction:** deferred to the already-frozen protocol boundary. Ordinary/human commands retain warning and safe-next-action rules. The exact harness JSON route is a narrow automation protocol whose surrounding U7B/U7C operator flow, not its four-field stdout, must carry the human warning. No U7A execution makes this exception live.
6. **Replay-safe next action:** defect found and repaired. A validated regular-file path is single-quoted as one shell argument, including embedded quotes; stdin explicitly says its bytes are not retained and requires saving the same bytes before preflight. Display paths preserve literal spacing and refuse control, bidi-format, and Unicode line/paragraph separator code points.
7. **One supplied preflight view:** defect found and repaired. Human execution-started, execution-authorized, and unresolved-authority lines are projected from the same `PreflightView` serialized by JSON. A deliberately inconsistent transport fixture requires the human renderer to preserve supplied `yes` values and only the supplied unresolved authority rather than substituting U7A literals.
8. **Strict option/value boundary:** defect found and repaired. `--spec --json` and `--spec --spec` are argument refusals rather than dash-leading filenames; an intentional dash-leading filename must use an explicit `./` path.
9. **Human diagnostic terminal controls:** defect found before the fresh ledger and repaired. An unknown SourceSpec member name can carry JSON-escaped control runes in the strict parser's typed diagnostic path. Machine JSON keeps its original typed value under encoder escaping; the human renderer now visibly escapes all control and format runes before wrapping any externally derived field. A non-JSON hostile with ESC, OSC/BEL, and C1 CSI bytes requires the typed refusal, empty stdout, visible escapes, and no raw terminal-control rune.

Architecture-oracle false positives, transitive effect absence beyond the exact reviewed source, hostile same-UID input replacement, filesystem containment, confidentiality, network denial, candidate safety, semantic-study correctness, full CLI ABI stability, Linux/Windows/other Node majors, independent security review, comprehension, adoption, production hardening, and maintainership are not established.

## Multi-pass craft record

All commands below were development-only didrun events under the U7A private environment. They support repair decisions, not a final claim or product grade.

1. **Pass 1 — build and narrow-terminal inspection.** The real `cmd/countershape` binary was built and exercised with `COLUMNS=60`, `NO_COLOR=1`, and the exact `help`, valid `validate`, valid `preflight`, and ordinary JSON routes. The machine envelope was canonical and one-line, but human prose, warning, digest, and next-action lines ignored the terminal width. The repair added bounded `40..240` column admission, deterministic prose wrapping, and literal-field wrapping. The post-repair test first failed only because an old assertion compared an intentionally wrapped sentence byte-contiguously. That nonzero permanently ended its ledger; it is preserved at `.didrun-history/u7a-development-red-width-20260809T153733Z/.didrun` with private root `.countershape/u7a-development-red-width-20260809T153733Z`.
2. **Pass 2 — three-width and refusal inspection.** A rebuilt real binary exercised help at 60 columns, valid validation at 80, valid preflight at 120, a 60-column unknown-command refusal through an expected-exit wrapper, and a wrong-artifact JSON refusal through an expected-exit wrapper. Every human line stayed within its admitted width; JSON remained one object plus LF with empty stderr; the wrong artifact refused at `$.artifact_digest` without echoing source bytes. A fresh `gpt-5.6-terra` critic received only the sanitized captures and staged twelve-path source. It found no Blocker or High and reported three Medium defects: unsafe/non-replayable `--spec` reinstruction, human preflight literals diverging from the supplied view, and option tokens consumed as spec filenames. All three were repaired as dispositions 6–8. The first hostile regression run then failed because its test incorrectly inferred JSON output from the rejected value token; production correctly emitted the usage refusal on stderr. That nonzero permanently ended its ledger at `.didrun-history/u7a-development-red-critic-hostile-20260809T154724Z/.didrun`, with root `.countershape/u7a-development-red-critic-hostile-20260809T154724Z`.
3. **Pass 3 — critic-repair rebuild.** The repaired focused suite passed, the actual binary rebuilt, regular-file validation at 80 columns emitted a single-quoted replay-safe path, stdin validation at 60 columns stated that bytes are not retained, and preflight at 120 columns retained the exact warning, `Execution started: no`, `Execution authorized: no`, and the three supplied unresolved authorities. Unit hostiles separately cover embedded quotes and shell metacharacters, repeated path spacing, bidi/line-separator refusal, both option-as-value forms, and deliberately inconsistent human/JSON preflight facts.
4. **Pass 4 — pre-ledger terminal-control red-team.** A read-only specialist review found that the strict SourceSpec parser can include a decoded unknown-member name in its typed diagnostic path, while the human renderer had emitted that path without presentation escaping. The repaired focused app suite ran once through didrun and passed, including the new human-mode ESC/OSC-BEL/C1 hostile and the strengthened all-control/all-format-rune assertion. Its unclaimed development ledger is preserved at `.didrun-history/u7a-development-terminal-controls-20260810T001138Z/.didrun`, with private root `.countershape/u7a-development-terminal-controls-20260810T001138Z`. It supports the repair decision only; no event, object, or result is reused in the final ledger.

An earlier development ledger is permanently preserved at `.didrun-history/u7a-development-red-focused-20260809T152738Z/.didrun` with root `.countershape/u7a-development-red-focused-20260809T152738Z`: its focused test exposed only two test-harness defects (JSON refusals were correctly on stdout, and the e2e repository path was joined twice). Both were repaired before any succeeding development event. No red event is reused. Every development ledger named above is archived before the final run and supports no final grade.

The different-model judgment and all CLI taste/comprehension findings remain `UNRECEIPTED`; the machine evidence can establish only the exact commands, bytes, exits, and checks it records.
