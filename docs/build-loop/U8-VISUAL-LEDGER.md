# U8 visual build-loop ledger

- **Unit:** U8 visual/responsive/accessibility boundary (P10)
- **Base:** sealed P09 commit `69b1d38f7c0347429f7ecb842d3b8847a5989789`, tree `b4bf5ededc040ec646e2405f990e306510077ba5`
- **Required viewports:** desktop `1440×900`; mobile `375×812`; supplemental `1280×720`, `320×812`, and a `720` CSS-pixel viewport at device scale factor `2` as the 200%-equivalent layout check
- **Status:** four local visual passes completed; final-tree receipt pending; required different-provider model critic unavailable before model execution; `studio-complete` is not claimed

## Skill-controlled protocol

The build-loop skill required repeated build → see → exercise → check → critique → rebuild cycles instead of screenshot approval. The in-app browser-control skill caused a live real-binary inspection and prohibited direct cookie or storage reads; URL/token behavior therefore remains behavior-tested rather than manually exfiltrated. Browser inspection used the app browser's screenshot, computed-layout, and semantic DOM snapshot surfaces. Neither skill creates product or didrun authority.

Every load-bearing automated command in this unit ran through `didrun run --`. Manual pixel judgments below are local observations, not didrun grades. The 27-event, zero-claim development ledger permanently retains failed lint, launch-path, over-strict raster, and external-critic observations at `.didrun-history/u8-p10-development-external-critic-unavailable-46a07a0de484/.didrun`; none transfers a command grade. Only successful final-tree events may receive P10 claims.

## Pass 1 — hierarchy and product truth

- **Input:** the five exact P09 screenshots in `evidence/ui/p09-inputs/`.
- **Initial defects:** dashboard/glass styling; radial and panel gradients; glow/drop-shadow cues; 72px rather than 64px top bar; small 7–10px controls; action editor before authored question on mobile; all outcome cards stacked on mobile; no state-transition focus restoration; and no dedicated safe-area action navigation.
- **Decision:** preserve the dark single-theme instrument but rebuild it as a flat laboratory bench. Equal outcome cards retain identical structure and dimensions; no card weight derives from support or candidate order.
- **Changed files:** `web/src/styles/foundation.css`, `web/src/features/decision/DecisionBench.tsx`, `web/tests/accessibility/studio.spec.ts`, and `web/tests/visual/capture-matrix.spec.ts`.
- **Result:** typecheck, lint, unit, build, and the full real-binary Playwright suite exited zero on candidate snapshot `85ade2075202f8ce0d51a3442c2aa730d56cbec9`. Added checks later exited zero on `b5aeabab711ef0032c97548aaa327a13e419d094`.
- **Matrix:** the first provisional capture exposed a harness bug: `resolved` could photograph the loading h1. The capture now requires the terminal heading. The replacement complete matrix is `evidence/ui/p10-pass1/`, 18 states × 2 viewports = 36 PNGs, captured by the successful didrun event ending at tree `17a4c708c351a80259dd6fd8e558f1691ac5b635`.
- **Observed result:** desktop immediately distinguishes jurisdiction, package states, trust boundary, question, and ruling editor; mobile now presents question/evidence before the editor and offers equal labeled outcome switching plus a safe-area ruling jump. Error, partial, unstable, uncomparable, stale, and invalidated states receive explicit next-honest-action copy rather than optimistic progress.

## Pass 2 — responsive, keyboard, accessibility, and hostile content

- **Live browser finding:** at the in-app browser's `1280×720` viewport, the fixed `640 + 352 + gap + rail/padding` geometry overflowed horizontally by 5px and visibly clipped the ruling panel. This was a material responsive defect missed by the required 1440/mobile pair.
- **Fix:** raise the one-column collapse boundary from 1210px to 1320px and add an exact 1280px regression. The live semantic snapshot contained all three equal blind articles, all five evidence surfaces, authored question/scope/repeat facts, complete ruling controls, no provenance identity, and no support count. Computed inspection found zero gradient backgrounds and zero backdrop filters.
- **Automation:** `320px`, `1280px`, 200%-equivalent layout, forced colors, reduced motion, 44px visible button targets, no horizontal overflow, serious/critical axe filter, outcome switching, transition focus restoration, every closed state, and hostile inert evidence all exited zero on tree `7f5c97aaa06ed4c95cd9e2fbf1aa237daaf2354b`.
- **Lighthouse:** real local production binary; Lighthouse `13.4.1`; performance `95`, accessibility `100`, best practices `96`; diagnostic only. The successful didrun event is self-stable on tree `e758e40561eeb489398dddf52f7f125b742aa4ae`.
- **Matrix:** `evidence/ui/p10-pass2/`, 36 PNGs, ending at tree `e758e40561eeb489398dddf52f7f125b742aa4ae`.
- **Remaining limits:** no VoiceOver pass, independent accessibility audit, non-Chromium browser run, human comprehension study, or security review was performed.

## Pass 3 — different-provider critic attempt and local state review

- **Curated input:** seven sanitized pass-2 PNGs, neutral state labels, exact byte/hash manifest, and the rubric required by P10. The critic root contained no source, prior finding, desired praise, bearer, URL fragment, private path, captured body, or implementation commentary.
- **Exact tool request:** Claude Code `2.1.210`, provider Anthropic, `opus` selector, high effort, safe mode, `Read` only, no session persistence, JSON output. The command ran through didrun.
- **Terminal result:** exit `1`; the existing OAuth session was expired and could not refresh. The response has empty model usage and the provider API was not called. Exact records are in `evidence/ui/different-model-critic/`.
- **Disposition:** `OPEN BLOCKER — EXTERNAL_CRITIC_UNAVAILABLE`. No model judgment, finding, ship verdict, or second comparison exists. A same-provider or role-played substitute is explicitly rejected. `STUDIO_COMPLETE_UNMET` remains exact.
- **Local third pass:** the complete 36-image matrix at `evidence/ui/p10-pass3/` was recaptured and inspected across empty, preparing, active, partial, error, unstable, uncomparable, discovered, decision-ready, predicate-editing, identity-reveal, resolved, reject-all, deferred, stale, invalidated, incomplete, and completed states. It ended at tree `815b156ad43a77ca510a0b7426e30df64791649f`.
- **Observed result:** failure and historical states carry the same typographic care and trust copy as favorable states; no decorative status chart, hidden trust evidence, winner cue, or candidate-count cue was found. The 375px deferred state retains the exact question before evidence and decision detail. This is manual local judgment, not independent criticism or comprehension evidence.

## Pass 4 — regression and honest completion

- **Production recapture:** the embedded bundle was rebuilt and all 18 states were recaptured at desktop and mobile into `evidence/ui/p10-final/`. Its strict roster manifest is 8,155 bytes with SHA-256 `20ab27c82ae9e9de9efdc230426887429b66b2b5039582b81e270fb20899a056`.
- **Intentional diff:** Pass 3 versus final retained identical dimensions and content geometry. Six of 36 PNGs contained 464 changed raster pixels in aggregate, all limited to at most `2/255` in one color channel around antialiased progress-rail edges. An independent same-binary recapture affected 506 of 18,009,000 pixels across eight PNGs under the same bound. `web/tests/visual/compare_matrices.go` refuses any dimension change, channel delta above `2/255`, or per-image change above `0.1%`; its two comparisons exited zero. This is bounded pixel stability, not byte identity or a visual-quality claim.
- **Final inspection:** decision-ready, error, identity reveal, deferred, resolved, and invalidated representatives were re-opened at full useful detail. No hierarchy, copy, state, trust-boundary, candidate-weight, clipping, or mobile-action change was observed. The judgment remains a local manual observation.
- **Regression boundary:** the final receipt must still rebuild the bundle, recapture all 36 screenshots to a private root and compare them with the tracked final matrix, run every functional/security/accessibility/hostile flow, Go embed/command tests, typecheck/lint/unit/build, axe, reduced motion, forced colors, viewport/zoom checks, and a bounded credential scan against one exact staged tree. Only successful self-stable final events may receive claims.

The admissible outcome is:

`VISUALLY_HARDENED_LOCAL_TRIAGE_STUDIO_EXTERNAL_CRITIC_UNAVAILABLE_STUDIO_COMPLETE_UNMET`

Exact nonclaims remain: `EXTERNAL_CRITIC_UNAVAILABLE`, `STUDIO_COMPLETE_UNMET`, `HUMAN_COMPREHENSION_UNRECEIPTED`, `BIAS_REDUCTION_UNRECEIPTED`, `INDEPENDENT_VISUAL_REVIEW_UNRECEIPTED`, `VOICEOVER_UNRUN`, `NON_CHROMIUM_BROWSERS_UNRUN`, `CROSS_PLATFORM_UNRUN`, `SECURITY_REVIEW_NOT_PERFORMED`, `PRODUCTION_READINESS_UNVALIDATED`, and `MARKET_VALUE_UNVALIDATED`.
