# P09 — U8 secure local studio and blind-first decision bench

You are implementing Countershape U8 in a fresh chat. This is a security-sensitive human decision surface, not a generic dashboard. Work autonomously until the unit is honestly shippable or one of the stop conditions fires. Do not commit before the verification/claim boundary below.

## Objective

Build the authenticated loopback API and a complete functional React decision bench over the already-implemented U0–U7 truth objects. The browser must render server-computed facts without recomputing identity, grades, eligibility, outcome maps, reduction status, or ruling validity. The decisive flow is blind-first: inspect scenario and evidence, provisionally choose an exact witnessed behavior and explicit assertion fields, reveal provenance, then finalize through a fresh digest compare-and-swap.

This prompt establishes functional and security correctness. P10 performs the full visual craft loop, but P09 still requires two internal passes: first make the state/API contract correct, then attack it as a hostile page, stale client, keyboard user, and malicious candidate-output producer.

## Read before editing

Read completely:

1. `docs/CONCEPT_BRIEF.md`, especially Trusted-local threat boundary, Blind-first human protocol, State machines, and U8 acceptance rows.
2. `research/deep-dive/04-product-dx.md`, `07-SYNTHESIS.md`, and `08-RED_TEAM.md`.
3. `docs/prompts/P08-U7-CLI-REFERENCE-STUDIES.md` and the current `docs/HANDOFF_MODE_C.md`.
4. Existing `internal/domain`, `internal/store`, `internal/choice`, CLI presentation, generated fixtures, and web/build configuration. Inspect; do not create a parallel truth model.
5. The installed browser-control skill instructions before using the browser. State in the work log when that skill changes an action.

Confirm the previous sealed unit passes `NO_COLOR=1 didrun verify --strict` before editing. Inspect `git status --short`; preserve unrelated work. If U7 is not sealed and strict-clean, stop and emit Mode C rather than building on an unverified base.

## Locked cuts and exact ownership

This unit owns `internal/server/**`, `internal/server_test/**` if that convention exists, `web/src/api/**`, `web/src/app/**`, `web/src/features/**`, `web/src/routes/**`, the minimal `web/src/components/**` needed for semantics, `web/src/styles/foundation.css`, `web/tests/security/**`, `web/tests/flows/**`, and necessary web tooling/config files. Modify shared domain/store code only to expose an already-defined invariant; do not duplicate it in JavaScript. Record every cross-unit change in `docs/build-loop/U8-IMPLEMENTATION-LEDGER.md`.

Do not add agent orchestration, Wake ingestion, candidate ranking, model-generated labels, source-diff adjudication, arbitrary plugins, cloud/multi-user service, cookies, remote bind addresses, external APIs, telemetry, or hostile-code containment claims. Candidate programs are trusted local code with full user permissions and host network access. The studio protects local evidence and mutation integrity from ordinary web threats; it does not defend against browser extensions or another same-user process that steals the token.

## Pass 1 — server contract and refusal behavior

Bind only literal `127.0.0.1` on an ephemeral port. Generate a high-entropy launch token, place it in the initial URL fragment, move it into `sessionStorage`, and remove the fragment immediately. Never place credentials in query strings, Referer-visible paths, analytics, process output, or application logs.

Require bearer authorization for every read and write. Require exact `Host` for all requests. For mutations also require exact `Origin`, JSON content type, a session-bound CSRF value, and the current Choicepoint/head digest. Send no permissive CORS headers. Apply a restrictive CSP suitable for embedded static assets, `nosniff`, and an explicit referrer policy. Reject `Origin: null`, DNS-rebinding Host values, form posts, simple cross-origin requests, missing/wrong bearer, missing/wrong CSRF, stale digest, guessed IDs, and methods not in the closed route table. Return typed, non-secret errors.

Use separate blind and reveal DTOs. A blind response must not contain candidate ref/name, model/provider/producer provenance, total candidate count, per-outcome support count, hidden branch data, or a reversible cross-study alias. Distinct outcome count is necessarily visible. Derive persisted Choicepoint-scoped aliases and ordering independent of candidate order and support. Identity must be absent at the serialized-byte level, not hidden by CSS. Reveal only after recording the provisional action and field acknowledgements; record early reveal and any post-reveal change plus rationale.

All mutating application services revalidate state and CAS server-side. `REJECT_ALL`, `DEFER`, `REFINE`, stale Choicepoints, empty predicates, ambiguous field sets, and forged action variants cannot reach the emitter. Visit markers prove presentation only: original/derivation, projection operations, nonasserted fields, and reveal must be rendered/visited before finalization, but never label that as comprehension.

## Pass 2 — complete functional bench

Implement one renderer for all honest jurisdictions:

- study: `EMPTY`, `PREPARING`, `ACTIVE`, `PARTIAL`, `ERROR`, `COMPLETED`;
- candidate: `OBSERVED_STABLE(k/k)`, `UNSTABLE`, `UNCOMPARABLE`, `INCOMPLETE` with control reasons;
- Choicepoint: `DISCOVERED`, `DECISION_READY`, `RESOLVED`, `DEFERRED`, `STALE`, `INVALIDATED`.

The decision route follows this fixed sequence: authored scenario and exact-witness jurisdiction; minimized and original stimuli plus derivation/grade/budgets/unresolved count; equal-weight blind outcome facts; allow-one/allow-many/custom/reject/defer; explicit `Assert` or `Context only` acknowledgement for every differing field with nothing preselected; exact tuple-set preview and `Not asserted`; provenance reveal; final confirmation/result. `ALLOW_OBSERVED` and `CUSTOM_EXPECTATION` alone may compile. Allow-many previews complete tuples, never a cross-product. A weak field set shows `AMBIGUOUS_SCOPE` and emits nothing.

Render Captured → Projection → Operations as an explicit boundary. Say `Captured` unless unchanged bytes were retained. Candidate output is always inert text. Visibly encode ANSI/OSC and dangerous controls; escape HTML, SVG, closing tags, bidi controls, forged headings, path separators, and strings that resemble UI. Candidate bytes never enter URLs, raw HTML, accessible names, CSS, scripts, or status copy.

Include the exact execution warning before first run: **“Countershape will run trusted repository code with your user permissions and host network access. The temporary directory is for repeatability, not security isolation.”** Use `HOST_ALLOWED`; never imply a sandbox or network denial. Use only the trust vocabulary allowed by the brief: observed-stable, exact projection, named local reduction grade, scoped ruling, conforms/contradicts. Never write winner, best branch, deterministic, proven safe, anonymous, unbiased, smallest, root cause, equivalent, or verified software.

## Required attacks and exercised flows

Create deterministic server/integration tests for the full auth matrix, exact Host/Origin, no-CORS behavior, content type, CSRF, CAS, token leakage, blind DTO byte inspection, illegal emitter actions, and stale-tab replay. Use fixture payloads containing `<script>`, `</script>`, event-handler HTML, ANSI/OSC, bidi overrides, invalid-looking UTF-8 replacements, fake PASS headings, secrets, path traversal text, and very long values. Assert that the payload remains data in API, DOM, accessibility tree, and logs.

Use Playwright against the real local binary/API, not mocked success responses, for: blind-to-final allow-one; allow-many tuple semantics; custom none-conforms; reject-all and defer with no files; early reveal and post-reveal rationale; weak-field refusal; stale CAS in two tabs; every lifecycle state; keyboard-only completion; mobile evidence inspection at 375×812; desktop at 1440×900; and a hostile-origin attempt. Run axe on the core routes and accept no serious or critical violations. Assert no page-level horizontal overflow and that focus is restored after reveal dialogs/sheets. Save screenshots as inputs to P10, but do not call screenshot existence a visual pass.

Run at least two attack/fix cycles. After the first green suite, inspect network requests, serialized blind payloads, DOM/accessibility snapshots, browser storage, URL/history, and logs manually. Add a regression test for every material finding. Do not weaken security checks, reduce fixtures, suppress axe rules, hide content on mobile, or substitute mocks to make the gate pass.

## Verification, didrun boundary, and commit

Every load-bearing command must be executed through `didrun run --`, including formatting checks, Go tests/race tests, web typecheck/lint/unit tests, production build, Playwright, axe, and the security/injection matrix. Use actual repository commands discovered from the build; do not invent a green command. Record commands and outcomes verbatim in `docs/build-loop/U8-IMPLEMENTATION-LEDGER.md`.

At the shippable boundary:

1. Ensure all relevant suites pass through `didrun run --` against the final tree and inspect the recorded events.
2. Declare only supported, truthful didrun claims (`tests-pass`, `lint-clean`, or `command-succeeded`) bound to the successful events. Do not claim manual taste, comprehension, security completeness, or an unrun environment.
3. Stage the exact U8 files, review the staged diff and secret scan, then commit with no AI co-author trailer.
4. Seal that commit with didrun.
5. Run `NO_COLOR=1 didrun verify --strict` as a loop. Nonzero means U8 is not done: read the verbatim grade/reason, fix the real issue, rerun affected verification through didrun, re-claim, commit, re-seal, and re-verify. Never delete or relabel an old claim, weaken a test, or conceal a permanent failed receipt.
6. Update `docs/HANDOFF_MODE_C.md` with commit, sealed claims/grades, screenshots, open risks, and the exact next prompt.

## Stop and fallback

Stop with Mode C instead of overclaiming if the domain/store API cannot supply blind facts without JavaScript truth recomputation; if auth/Host/Origin/CSRF/CAS attacks remain open; if candidate identity/support leaks in blind bytes or accessibility state; if illegal actions can emit code; or if mobile must hide a trust boundary. The allowed fallback is the already-verified CLI plus a read-only seed-state renderer, explicitly **not** a studio-complete claim. Mark all unrun viewports, browsers, operating systems, manual observations, and human-comprehension claims `UNRECEIPTED`.
