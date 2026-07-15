# P10 — U8 multi-pass visual, responsive, and accessibility build loop

You are continuing Countershape U8 in a fresh chat after P09 has been committed, sealed, and strict-verified. Invoke the build-loop skill because visual feel and human comprehension are load-bearing here. Also read and use the in-app browser-control skill for live inspection of the real local application. This is not a one-screenshot review and not a unit-test substitute: repeatedly build, see, exercise, critique, rebuild, and re-run regressions until the objective bars pass or an explicit stop condition fires.

## Objective

Turn the functional secure studio into a reference-quality scientific comparison bench that makes one bounded product disagreement legible without turning candidates, counts, or visual hierarchy into a recommendation. Complete at least three full visual passes plus a final regression pass at 1440×900 and 375×812. Use a real model from a provider different from the builder for one screenshot-only critic pass. Treat its output as a fallible opinion, preserve it verbatim, and address or explicitly reject each material finding.

## Read before editing

Read completely:

1. `docs/CONCEPT_BRIEF.md`, especially Blind-first human protocol, trust copy, surface architecture, and human-surface acceptance.
2. `docs/SEMANTICS.md` and `docs/PROJECTION_ALGEBRA.md`; visual grouping and copy remain non-authoritative projections of server facts.
3. `research/deep-dive/04-product-dx.md`, `07-SYNTHESIS.md`, and `08-RED_TEAM.md`.
4. `docs/prompts/P09-U8-SECURE-STUDIO.md`, `docs/build-loop/U8-IMPLEMENTATION-LEDGER.md`, and `docs/HANDOFF_MODE_C.md`.
5. All current web components/styles, Playwright state fixtures, axe checks, and P09 screenshots.
6. The complete build-loop and browser-control skill files before tool use. In the ledger, record actions or pauses those skills caused.

Before editing, run the prior commit's `NO_COLOR=1 didrun verify --strict` and inspect `git status --short`. If the sealed base is not strict-clean, stop and hand off; visual polish cannot paper over a failed functional/security unit.

## Locked design direction and ownership

This unit owns `web/src/components/**`, `web/src/features/**`, `web/src/routes/**`, `web/src/styles/**`, `web/src/assets/**` when assets are necessary and local, `web/tests/visual/**`, `web/tests/accessibility/**`, `evidence/ui/**`, and `docs/build-loop/U8-VISUAL-LEDGER.md`. It may adjust P09 UI code to fix visual/accessibility defects, but must not change server truth, blind DTO content, ruling semantics, auth, or CAS without returning to the relevant P09 security tests and recording the cross-boundary change.

The feel is an exacting laboratory comparison bench: calm, tactile, evidence-first, typographically excellent, and useful without color. It is not mission control, a branch leaderboard, an AI-chat shell, a neon terminal, or a generic analytics dashboard. No AI gradients, glowing orbs, glassmorphism, fake activity, decorative charts, victory confetti, “winner” emphasis, or card size tied to support. Outcome cards receive equal visual weight. Candidate identity stays absent until reveal. Human-authored context and deterministic observed facts are visually distinct.

Desktop uses the brief's 1360px maximum canvas, 224px study rail, flexible decision canvas with a 640px floor, 352px evidence inspector, 64px top bar, and nonoverlapping 72px action bar. Mobile at 375px is a composed single-column decision surface: labeled outcome switching, Question/Evidence/Predicate access, full original/derivation/projection/nonasserted/reveal parity, and a safe-area-aware 64–88px action region. If full parity cannot be made clear, mobile becomes explicitly triage/defer-only; never hide trust evidence to fit.

## Establish the visual evidence protocol

Use the real embedded application and deterministic seed-state routes or fixture APIs, not static mockups. Create a reproducible capture matrix for both viewports covering at least: empty, preparing, active, partial, error, unstable, uncomparable, discovered, decision-ready, predicate editing, identity reveal, resolved, reject-all resolved, deferred, stale, invalidated, and completed. Capture consistent light/dark behavior only if both are actually supported; do not add a theme merely for screenshot variety.

For each pass, record in `U8-VISUAL-LEDGER.md`: commit/tree before the pass, exact commands, browser/viewport/state, screenshot paths, observed defect, severity, decision, changed files, regression result, and remaining risk. Inspect screenshots with vision at full useful detail; do not infer quality from pixels existing. Keep sanitized fixture screenshots only—no bearer token, fragment, private path, captured body, secrets, or user repository data.

## Pass 1 — information hierarchy and product truth

Capture the full matrix, then critique the first ten seconds of every state. A reviewer must immediately distinguish: study/candidate/Choicepoint jurisdiction; authored scenario versus observed facts; exact witness scope; observed and fresh-confirmation counts; reduction grade/budget/unresolved count; and the one safe next action. Failure/partial/stale states must receive at least as much design care as resolution.

Make reduced/original/derivation relationship unmistakable. Keep Captured → Projection → Operations one clear action away. Show all differing fields before assertion, nothing preselected, and `Not asserted` beside the exact tuple preview. Use precise trust copy from the brief. Remove ambiguous badges, ornamental metrics, misleading progress, hidden truncation, hover-only evidence, and visual cues that encode candidate order or support. Validate outcome permutations have equal dimensions, typography, emphasis, DOM metadata, and accessible naming.

Re-run the functional Playwright flows and security suite through didrun before beginning Pass 2. A visually attractive semantic regression is a failed pass.

## Pass 2 — responsive, keyboard, accessibility, and hostile content

Exercise the complete allow-one, allow-many, custom, reject, defer, early-reveal, and stale-CAS paths using keyboard only. Verify logical focus order, visible and unobscured focus, focus trap/escape/return for dialogs and sheets, one polite live region, native control semantics, at least 44px designed touch targets, and no icon-only irreversible actions. Use WAI-ARIA tabs behavior only where tabs remain.

At 1440×900, 375×812, 320px width, and 200% browser zoom, confirm no page-level horizontal scrolling, hidden action, inspector overlap, clipped JSON, or covered focus. Inspect `prefers-reduced-motion`, forced colors, monochrome/NO_COLOR-equivalent comprehension, long localized-looking strings, and hostile candidate text containing HTML, ANSI/OSC, bidi controls, giant unbroken tokens, absent/empty fields, and forged UI labels. Candidate output must remain inert visible evidence, never chrome or an accessible name.

Run Playwright, axe (no serious or critical findings), accessibility-tree snapshots, production build, and a Lighthouse pass against the local production surface. Treat Lighthouse as diagnostic evidence, not a substitute for keyboard/manual inspection and not a license to remove evidence for score. Add regression coverage for every material defect and recapture the affected states at both viewports.

## Pass 3 — real different-model screenshot critic

The user has human-gated a real external-model visual critic for this build. Use an actually installed, authenticated model CLI from a provider different from the current builder—prefer the available Claude CLI only after confirming its exact version/model flags. Do not fake, role-play, synthesize, or paraphrase a critic run. If no real different-model tool can execute, record the blocker and stop short of a studio-complete claim.

Give the critic only a curated, sanitized screenshot set with neutral state labels and this rubric, without source code, implementation commentary, desired praise, prior findings, or candidate identity: (1) state and jurisdiction clarity; (2) equal outcome weight and absence of majority cues; (3) trust-boundary prominence; (4) understanding reduced versus original and projection operations; (5) asserted versus nonasserted clarity; (6) mobile final-action safety; (7) keyboard/focus cues visible where screenshots permit; (8) any language that overstates certainty. Ask for severity-ranked findings, evidence by screenshot, and a ship/block verdict. Record the exact CLI, provider, model identifier, tool version, prompt digest, image manifest, exit status, and verbatim response in `evidence/ui/different-model-critic/`; run the external critic command through `didrun run --` because it is a required build observation. Never include credentials or count the opinion as deterministic verification.

Triage every finding in the ledger as accepted/fixed, rejected with concrete evidence, or open blocker. Fix all blocker/high findings that are in scope, add tests where mechanizable, then recapture and—when material layout changed—run one focused second critic comparison. Do not blindly obey taste advice that violates the brief or security model.

## Pass 4 — regression and honest completion

Run the entire state matrix once more from a clean production build. Diff screenshots intentionally; inspect all changed states, not only golden updates. Re-run Playwright functional/security flows, axe, keyboard scripts, viewport/overflow checks, reduced motion, forced colors, production embedding, Go tests affected by web assets, and secret/token scans. Screenshot-golden changes require a written reason; never mass-update snapshots to hide a regression.

The visual loop is complete only when all required states were actually inspected at both viewports, no blocker/high critic finding remains unexplained, no serious/critical axe finding remains, core paths complete by keyboard, mobile has evidence parity or is honestly triage-only, hostile content is inert, and functional/security regressions remain green. Report VoiceOver or another nonvisual pass as a manual observation only if it was actually performed; it remains `UNRECEIPTED` unless a concrete supporting command was captured, and otherwise belongs wholly to the human tail.

## didrun unit boundary, commit, and strict loop

Wrap every load-bearing build, test, capture, axe, Lighthouse, browser automation, critic, and regression command with `didrun run --`. Manual screenshot judgments remain manual observations and must not be upgraded into a command grade. At the boundary, bind truthful supported claims to the successful final events, stage only the U8 visual unit, inspect the diff and screenshot/privacy manifest, commit without an AI co-author trailer, seal, and loop on `NO_COLOR=1 didrun verify --strict` until exit 0.

Any nonzero strict result means the unit is unfinished. Fix the underlying stale/failed/unknown evidence, rerun through didrun, declare new honest claims, commit, reseal, and reverify. Never delete, weaken, relabel, or overwrite an earlier failed receipt. Update `docs/HANDOFF_MODE_C.md` with the final commit, verbatim grades, visual matrix, critic identity/findings, exact unreceipted observations, and P11 as the next prompt.

## Stop and fallback

Stop with Mode C if functional truth must be distorted for aesthetics; the blind view leaks provenance/support; mobile needs hidden evidence; keyboard finalization cannot be safe; hostile output becomes executable/presentational chrome; the required real critic cannot run; or the loop reaches three rebuilds with the same blocker and no new evidence. The honest fallback is CLI-complete with a read-only/triage studio. Human comprehension, bias reduction, review compression, security-review completeness, cross-browser parity, and market value remain bets regardless of a green U8.
