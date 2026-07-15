# P12 — final evidence, live-agent S6 dogfood, receipts, and Mode C handoff

You are closing the Countershape autopilot run in a fresh chat. This is an evidence and honesty unit, not an opportunity to add features. Audit the sealed U0–U9 system, exercise didrun under a real nested-agent session, produce the final commit-bound strict/HTML evidence, and leave a self-contained handoff that separates observed capabilities from product, market, security, and maintainership bets.

Do not declare the active goal complete merely because time or context is low. Complete it only after every required gate below passes or after an honest scope cut has been committed, receipted, and named. If context runs short, update Mode C and stop at a resumable boundary.

## Read before acting

Read completely:

1. `docs/CONCEPT_BRIEF.md`, `docs/SEMANTICS.md`, `docs/PROJECTION_ALGEBRA.md`, `docs/PROMPT_PACK.md`, every `docs/prompts/P00…P11` file, and `docs/HANDOFF_MODE_C.md`.
2. `research/deep-dive/07-SYNTHESIS.md` and `08-RED_TEAM.md`.
3. Every `docs/build-loop/*LEDGER.md`, `docs/PERFORMANCE.md`, `docs/LIMITATIONS.md`, `docs/THREAT_MODEL.md`, `SECURITY.md`, and the current README.
4. didrun's installed `--help`, global agent discipline, current session/event display, claim types, seal behavior, strict semantics, and HTML option. Do not assume syntax when the installed binary can tell you.
5. The original didrun S6 protocol at its installed/source location if available. Record the didrun executable path, reported version, package/source provenance, and any version mismatch as an observation—not a guessed resolution.

Inspect `git log --oneline --decorate`, `git status --short`, tracked files under `.didrun`, and didrun notes/seals. `.didrun/` is secret-bearing and must never be committed. Confirm each previous shippable unit has a commit, a seal, and `NO_COLOR=1 didrun verify --strict` exit 0 before continuing. A missing or nonzero prior gate returns to the owning prompt; do not manufacture a final umbrella receipt.

## Exact ownership and prohibited actions

P12 owns `evidence/s6/**` sanitized summaries, `evidence/final/**` non-secret inventories, `docs/FINAL_HANDOFF.md`, `docs/RECEIPTS.md`, the final update to `docs/HANDOFF_MODE_C.md`, and narrowly scoped audit helpers under `tools/final-audit/**` if needed. The final HTML lives under ignored `.didrun/reports/countershape-final.html` so report generation does not dirty the sealed tree. Do not edit product code in this prompt. A product defect sends work back to its owning unit for a new verified commit, then P12 restarts its audit.

Do not push, publish, open a PR, create a hosted service, sign/release an artifact, invoke a fake external model/agent, or claim human/market/security validation. Preserve every didrun grade verbatim. `tree-exact` means a self-stable command was recorded against the sealed tree; it does not mean the claim is true or the software is correct. Anything not executed through didrun is `UNRECEIPTED`, even if it was manually observed or another tool said “pass.”

## Audit pass 1 — capability and receipt inventory

Build a machine-auditable inventory of U0–U9 commits, trees, seals, claims, event references, commands, exits, and verbatim grades. Cross-check against the brief's acceptance matrix. For each capability, identify the narrowest actual evidence command and exact environment (OS/architecture, Go, Node, browser/viewport, model critic where applicable). Never assign one receipt to a different capability or infer Linux/browser/runtime coverage from a build.

Run focused negative audits through `didrun run --`: prohibited product/claim vocabulary; tracked `.didrun` or credentials/private roots; Wake/runtime overlap; unknown-grade round-trip; default export warning; forbidden remote bind/CORS; generated bundle Countershape imports; report external assets; and package manifest leaks. A text scan is supporting evidence only—retain behavioral tests as the load-bearing evidence.

If the inventory exposes an unreceipted claim in README/docs, either remove/narrow that claim in the owning unit and redo its verification, or mark it `UNRECEIPTED`. Never create a claim after the fact without a recorded run.

## Audit pass 2 — formal live-agent S6 protocol

The user explicitly human-gated real external agent API use for this run and requested live S6 evidence. Execute a real nested coding-agent CLI; do not simulate an agent with a shell script or ask the current model to role-play its output. Prefer the installed Codex CLI for direct relevance, or Claude CLI if the exact installed/authenticated setup is more reliable. Record provider, model, CLI path/version, and invocation flags. If authentication, quota, or CLI availability prevents a real run, record `S6 UNRECEIPTED` plus the exact blocker; never fabricate success.

Use a disposable directory outside the Countershape repository. Initialize a fresh Git repository with a tiny deterministic test fixture and baseline commit. Copy no Countershape source, credentials, user data, or `.didrun` ledger into it. Freeze and hash an agent task that requires the agent itself to:

1. make meaningful changes to at least two fixture files;
2. run at least one successful test command by bare name;
3. run at least one intentional failing command and continue after observing the nonzero exit;
4. run at least one command by absolute path;
5. inspect its diff and rerun the passing test.

Run **two independent real agent sessions** against fresh copies of that fixture so the result is not one lucky shell. Launch each actual agent under didrun capture using the installed documented mechanism; preserve the agent's machine-readable tool-call transcript separately from didrun's ledger. Use fixed non-secret prompts and bounded agent settings. Do not let either agent touch the Countershape checkout, network resources beyond its human-gated model provider, or unrelated files.

For each session, construct a row-by-row cross-check between the agent's own command tool calls and didrun events:

- every bare-name command must appear with argv and exit status;
- the intentional failure must be recorded as failing, never success;
- the absolute-path command must either be captured with honest argv/exit or explicitly graded as the documented structural miss—never silently counted as captured;
- the didrun shim/trap path must survive the agent's shell-snapshot/PATH initialization;
- session ordering and coverage limitations must be explicit.

Compute defect-class recall against the frozen transcript: success/bare-name, failing/bare-name, and absolute-path/structural-miss classes. The S6 pass bar is at least 95% defect-class recall across both real sessions with no silent class miss. A 80–95% result is a tier-downgrade finding; below 80%, wrapper incompatibility, shim loss, false success, or silent absolute-path loss is **S6 PARK**. Do not change the fixture, denominator, or threshold after seeing results.

Store raw transcripts and ledgers only in ignored `.didrun/s6/`. Run a secret/token/private-path scan before extracting a sanitized summary to `evidence/s6/S6-LIVE-AGENT-RESULT.md` and a deterministic matrix to `evidence/s6/s6-matrix.json`. The summary includes prompt/fixture digests, exact commands, sessions, expected versus observed rows, recall arithmetic, limitations, and one of `PASS`, `TIER-DOWNGRADE`, `PARK`, or `UNRECEIPTED`. It must not include credentials, raw private transcripts, or an upgraded euphemism.

Run every load-bearing S6 launch and comparison/audit command through `didrun run --`. If recursive didrun introspection cannot be safely wrapped, record that step as an `UNRECEIPTED` manual diagnostic and do not use it to support a capability claim. A failure in didrun itself is a live S6 bug finding, not permission to stop the Countershape audit. Log reproduction steps, installed version/source mismatch, expected/actual behavior, severity, and whether direct `didrun run --` unit receipts remain usable. Continue honestly under the user's instruction: S6 failure limits claims about didrun live-agent capture; it does not retroactively turn explicitly wrapped Countershape commands into unrun commands.

## Audit pass 3 — final comprehensive verification

From a clean tree, run the full final matrix through `didrun run --`: formatting/lint; all Go tests and race tests; required mutation/property/state-machine suites; web typecheck/unit/build; Playwright functional/security/visual assertions; axe; export injection/CSP checks; clean-package smoke; both standalone-contract absence/parity studies; three clean reference-study reproductions; performance budget check; secret/package inventory; and any final audit helper. Use only commands actually supported by the repository. Do not omit a slow suite merely because it passed in an earlier unit.

Inspect the final UI/report screenshots rather than mass-accepting snapshots. Confirm the real different-model critic identity and findings are preserved, but do not convert its opinion into a deterministic grade. Confirm the exact host is the only native runtime claim and every unrun OS, Node major, browser, viewport/state, VoiceOver path, security review, human study, and adoption claim is marked `UNRECEIPTED` or a bet.

If any load-bearing command fails, fix in the owning unit, run the genuine loop there, commit/seal/strict-verify, then rerun P12. Never weaken tests, lower thresholds, remove negative fixtures, delete claims, or relabel permanent failed receipts.

## Final handoff and receipts contract

Write `docs/RECEIPTS.md` with a table containing exactly:

`Capability | Scope/environment | Load-bearing command | Event/claim reference | Verbatim didrun grade | What the grade does not establish`

Include at least canonical identity; Git materialization; fresh Darwin process lifecycle; CLI and HTTP stability; exact OutcomeMap; reducer grades and shape trap; fresh confirmation; ruling/separation; standalone bundle/parity/absence; two-domain kernel; CLI/reference studies; studio security; human-surface automation; visual manual/critic observations; local export; packaging; reproduction/performance; S6; and each environment claim. Put `UNRECEIPTED` literally wherever no didrun-backed evidence exists. Do not summarize `tree-exact`/`scope-exact` as verified, proven, certified, or passed by an independent authority.

Write `docs/FINAL_HANDOFF.md` as a complete Mode C handoff with:

- product thesis, exact scope, final commit/tree, artifact paths, and quickest real demo;
- what is technically validated, with links to receipts and decisive negative proofs;
- what was manually observed but unreceipted;
- explicit fallback/relabel, if any;
- S6 live-agent result and didrun bug findings;
- known limitations and security exclusions;
- a `Validated vs. bet` ledger separating engineering observations, inference, human-study hypotheses, novelty, and adoption;
- the user's tail: adoption research, comparative comprehension study, production hardening, independent security review, packaging/signing/release, legal/name clearance, cross-platform native matrices, maintenance policy, contributor governance, and ongoing adapter stewardship;
- no claim that the product is finished, market validated, hostile-code safe, production ready, or maintainable without a maintainer.

Update `docs/HANDOFF_MODE_C.md` to point to the final handoff and state whether continuation is required. Run documentation/link/claim-language checks through didrun, stage only sanitized final artifacts, review the staged diff and secret scan, and commit without an AI co-author trailer.

## Final seal, strict gate, and HTML evidence

After the final documentation commit, seal it with didrun. Generate the final report against that exact commit using the installed syntax equivalent to:

```text
NO_COLOR=1 didrun verify <FINAL_COMMIT> --strict --html .didrun/reports/countershape-final.html
```

The exact flag order comes from installed help. Exit must be 0. If it is nonzero, the final unit is not done: fix the real stale/failed/unknown condition, rerun affected verification through didrun, declare truthful new claims, commit, reseal, and regenerate against the new final commit. Old failed receipts remain permanent. Never edit the HTML or hand-author a green report.

Open the generated HTML locally and confirm it renders, is self-contained, names the exact final commit/tree, contains every final claim and verbatim grade, and has no credentials/private transcript. Record its SHA-256 and absolute path in the final handoff response. The ignored report may remain outside Git; its binding is the sealed commit and didrun manifest, not an added post-seal tree change.

Only now, if the scoped objective is genuinely achieved and no required work remains, mark the active goal complete. If the goal is token-budgeted, report the final usage returned by the goal tool. Otherwise leave it active and emit Mode C; never use goal completion to hide a missing S6, strict, HTML, or receipt gate.

## Stop conditions and honest continuation

Stop and hand off if a prior unit is unsealed, final strict remains nonzero, the HTML cannot bind the exact final commit, raw S6 material cannot be sanitized, or a product defect requires implementation. A real S6 `PARK` is not itself a reason to falsify Countershape completion: record the didrun capture limitation, retain only directly wrapped receipt claims, and continue. Market adoption, human comprehension, review compression, legal clearance, independent security review, production operations, cross-platform native support, and maintainership are always the human tail.
