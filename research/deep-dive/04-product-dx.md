# Product and DX deep dive: making an empirical Choicepoint honestly decidable

- **Lane:** user comprehension, CLI, responsive studio, accessibility, and trust language
- **Date:** 2026-07-14
- **Recommendation:** **MODIFY, THEN PROCEED**
- **Confidence:** 8/10 on the interaction architecture; 6/10 that deterministic presentation is sufficient for both reference domains; 3/10 that it compresses real review work until a comparative user test is run

## Executive finding

Countershape can give a serious non-expert a real decision surface without a model, source diff, or raw-log archaeology, but the current brief does not yet specify the protocol tightly enough. A conventional dashboard would quietly turn candidate order, group size, and branch names into recommendations; one-click approval would freeze incidental fields into brittle tests.

Treat **resolving a Choicepoint as a staged, auditable protocol**, not selecting a card:

1. establish the human-authored scenario and the experiment jurisdiction;
2. show the reduced witness beside its original and reduction grade;
3. compare deterministic, identity-blinded observed outcomes with equal visual weight;
4. choose one/many/custom/reject/defer and explicitly select every asserted field;
5. reveal candidate identities and implications, then finalize or record why the ruling changed.

The first four stages hide branch names, model/provider provenance, and outcome cardinality. The fifth reveals them because permanent anonymity blocks legitimate implementation context and is not credible: behavior may reveal authorship. Say **“candidate identity hidden in this view,” never “anonymous” or “unbiased.”** Google’s field experiment across 5,217 reviews and 300 engineers found that reviewers often guessed hidden authors, while anonymity reduced attention to power dynamics but hindered high-bandwidth conversation. That supports staged blinding, not erased provenance. See the primary [Google Research paper](https://research.google/pubs/engineering-impacts-of-anonymous-author-code-review-a-field-experiment/).

Countershape must distinguish semantic context from deterministic description. It may display user-authored `Cross-tenant invoice fetch` and mechanically label `HTTP 404 · JSON object (2 fields) · state unchanged`; it may not generate “secure concealment” without human interpretation. The fallback is typed facts plus optional authored annotation that never affects equality or compilation.

Without those changes, kill the dashboard and ship only the truth-model prototype.

## The P0 interaction hazards

### 1. Candidate identity and majority can become an accidental judge

Showing `claude-opus`, `codex`, `main`, `senior-engineer`, or “3 of 4 candidates” before the user states the desired behavior introduces provenance, authority, incumbency, and majority cues. None is evidence of intent. Research has found apparent authorship can bias code review even when code quality is controlled; see the controlled study reported at [ESEC/FSE 2020](https://2020.esec-fse.org/details/fse-2020-papers/114/Biases-and-Differences-in-Code-Review-using-Medical-Imaging-and-Eye-Tracking-Genders). More generally, empirical work on [automation bias in AI decision support](https://pubmed.ncbi.nlm.nih.gov/39234734/) supports avoiding an automated recommendation that a hurried user can rubber-stamp.

Countershape must not show a recommended outcome, winning branch, confidence score, popularity count, “most candidates,” or preselected radio. Each outcome receives the same width, typography, and action affordance. Initial ordering is derived from a Choicepoint-scoped hash, not candidate order, status code, or cluster size, and is persisted for audit. Stable opaque aliases derive from projection fingerprints, for example `Outcome 7A92`, rather than ordinal `Option A` or quality-coded names.

### 2. “Choose this outcome” can silently over-assert

An observed outcome may include status, headers, body fields, database delta, latency, and capture metadata. Clicking its card must not mean “assert all of this forever.” After selecting allowed outcome fingerprints, the user enters a predicate stage where every discriminating projected field is explicitly marked **Assert** or **Context only**. Nothing is pre-asserted. Save remains disabled until at least one field is asserted and every visible discriminating field has been acknowledged.

For several allowed outcomes, the preview is a disjunction of selected tuples, not an accidental cross-product. The UI should render:

```text
(status = 403 AND state_delta = none)
OR
(status = 404 AND state_delta = none)
```

and list `body`, `headers`, and `latency` under **Not asserted**. The compiler preview and generated test diff are visible before finalization. This is the only credible defense against a one-click approval test that freezes incidental output.

### 3. Projection can become an invisible editorial opinion

The decision card uses projected values, but the right-hand evidence inspector must keep the capture boundary visible:

```text
Program emission
  -> capture policy (possibly redacted before persistence)
  -> Captured Observation
  -> projection operations
  -> canonical value used for equality
```

Use **Captured**, not **Raw**, unless exact bytes were actually retained. Every outcome has one action to open `Captured / Projection / Operations`. The operations view says exactly what was dropped, replaced, sorted, or parsed. Changing projection settings does not live-update the current Choicepoint; it creates a new lineage and marks dependent artifacts stale.

### 4. A reduced witness can be technically valid and humanly misleading

The main surface pins the reduced witness, but never presents it alone. Above it remain the human-authored scenario name, original intent text, and declared fixture purpose. Beside it is a persistent reduction receipt:

```text
Reduced 14 fields -> 3
BEST KNOWN under json.remove-key@1 and json.shrink-scalar@1
73/100 trial budget used · 2 direct neighbors unresolved
```

or, where earned, `1-MINIMAL UNDER 2 NAMED REDUCERS`. Copy never says “smallest,” “minimal behavior,” or “root cause.” Desktop offers `Reduced | Original | Derivation` as tabs; mobile presents the same content in stacked sections, with Original one tap away and never behind an unlabeled icon. The derivation shows stimulus changes, not source changes.

## One state model, three levels

The current vocabulary mixes study lifecycle, Choicepoint lifecycle, and candidate eligibility. That will produce contradictory banners unless the jurisdictions are explicit.

### Study-level states

- `EMPTY`: no compiled study exists.
- `PREPARING`: parsing, resolving refs, materializing, setup, or readiness is in progress.
- `ACTIVE`: eligible trials, reduction, or fresh confirmation are running.
- `PARTIAL`: work stopped or was cancelled after preserving some evidence; no derived decision is promoted.
- `ERROR`: a terminal study-level fault prevents continuing under the current plan.
- `COMPLETED`: the declared budgets ended and every discovered item is resolved, deferred, stale, or explicitly incomplete. It does not mean the software is correct.

### Candidate/trial eligibility states

- `OBSERVED_STABLE(k/k)`: required fresh eligible trials produced one projection fingerprint.
- `UNSTABLE`: at least two complete projection fingerprints were observed for one candidate.
- `UNCOMPARABLE`: materialization, setup, readiness, world compatibility, projection, timeout, output, or teardown made the candidate ineligible.
- `INCOMPLETE`: the repeat budget ended before enough eligible trials existed.

`ERROR`, `TIMEOUT`, `MISSING`, and `OUTPUT_LIMIT` are reasons under trial control, not outcome clusters. An HTTP `500` or CLI exit `2` may still be ordinary observed behavior when the adapter completed.

### Choicepoint-level states

- `DISCOVERED`: at least two observed outcome fingerprints exist, but reduction or confirmation is unfinished.
- `DECISION_READY`: the user-facing name for a freshly confirmed, observed-stable Choicepoint. The machine record may remain `CONFIRMED`; do not show bare `STABLE` as if determinism were proved.
- `RESOLVED`: a ruling is recorded. Resolution kind is `ALLOW_OBSERVED`, `CUSTOM_EXPECTATION`, or `REJECT_ALL`.
- `DEFERRED`: no contract was emitted; rationale and revisit condition are visible.
- `STALE`: a semantic ancestor changed. The old bytes remain inspectable, but no action can mutate them into current evidence.
- `INVALIDATED`: an explicit reason supersedes the record.

`UNSTABLE` and `UNCOMPARABLE` are not weak Choicepoints. They are evidence about why a candidate is excluded from one. The queue can show them as blockers alongside a Choicepoint, but cannot invite resolution over their outputs.

## Exact CLI information architecture

The CLI should be a legible state machine, not a collection of low-level engines. It has one primary discovery command and one primary decision command:

```text
countershape next [--study <id>]
countershape resolve <choicepoint-id>
```

The v1 surface is:

```text
countershape init
countershape study inspect --file countershape.study.json
countershape study run --file countershape.study.json
countershape study status <study-id>
countershape next [--study <study-id>]
countershape choicepoint show <id> [--view question|evidence|predicate]
countershape choicepoint reveal <id>
countershape resolve <id>                    # interactive, blind-first
countershape ruling preview <id> --file ruling.json
countershape ruling apply <id> --file ruling.json --expect-digest <digest>
countershape evidence show <id> --captured|--projection|--derivation
countershape studio <study-id>
countershape report <study-id> --output report.html
```

There is no `--winner`, `--best`, `--approve branch-a`, or direct `resolve --status 404 --yes`. Scripted resolution is two-phase: `ruling preview` prints selected fields, non-asserted fields, scope, artifact paths, and Choicepoint digest; `ruling apply` requires that digest. This prevents stale clients applying a ruling to changed evidence.

The interactive flow is concise:

```text
CHOICEPOINT cp_91e4 · DECISION READY
Observed stable 5/5 · fresh confirmation 5/5
Scope: this exact witness

Scenario (human-authored)
Cross-tenant invoice fetch

Reduced witness · BEST KNOWN
GET /invoices/inv_7
role=support · actor_tenant=blue · invoice_tenant=red

Candidate identities and outcome counts are hidden in this view.

Outcome 7A92  HTTP 404 · JSON object (1 field) · state unchanged
Outcome C144  HTTP 403 · text body (18 B) · state unchanged
Outcome F801  HTTP 200 · JSON object (4 fields) · state unchanged

Resolution: allow one / allow several / custom / reject all / defer
```

After an outcome selection, the terminal asks which projected fields to assert, renders the exact predicate, lists everything not asserted, then records a provisional choice. Only then does it reveal candidate refs, provenance, and per-candidate implications. If the user changes the choice after reveal, Countershape records both rulings and requires a rationale; it does not block the change or pretend the result stayed blinded.

Status output always ends with one safe next action. `ACTIVE` offers `watch` or `cancel`; `PARTIAL` offers `resume` or `inspect`; `UNSTABLE` offers `increase trials` or `inspect histogram`; `STALE` offers `derive a new study`; `COMPLETED` offers `report`. No progress animation or color is required to understand it, and `NO_COLOR` remains fully usable.

## Responsive studio information architecture

### Desktop at 1440px

Use a restrained 1360px maximum canvas with a 12-column grid:

- **224px study rail:** study identity, queue filters, counts by honest state, and one-line next action. No branch list.
- **minmax(640px, 1fr) decision canvas:** scenario, witness, equal-weight outcome cards, resolution, and predicate editor.
- **352px evidence inspector:** jurisdiction, capture/projection boundary, reduction grade, freshness, and provenance reveal. It is sticky but never overlays the final action.
- **64px top bar:** repository, study digest, state, and local-only/security boundary.
- **72px sticky decision bar:** Back, Defer, Reject all, and Continue/Finalize. Destructive or semantically distinct actions do not share a color-only distinction.

The default route is `/studies/:study/choicepoints/:id/question`; siblings are `/evidence`, `/predicate`, and `/result`. Browser Back preserves the draft. The decision canvas has this fixed order:

1. state and jurisdiction banner;
2. human-authored scenario context;
3. reduced witness with Original/Derivation access;
4. observed outcomes, identity-blinded;
5. resolution vocabulary;
6. exact predicate builder;
7. identity reveal and final confirmation;
8. result/contract residue.

The outcome area is not a branch spreadsheet. Each card shows differing projected fields plus explicitly selected unchanged state facts. Content is never truncated merely to preserve alignment; an evidence-only semantic table supports row/column comparison and export.

### Mobile at 375px

Mobile is a complete decision surface, not a shrunk desktop matrix:

- a 48px top bar with Back, `Choicepoint n of m`, and state;
- a two-line scenario header and exact-witness scope;
- one outcome card at a time, switched with a labeled segmented control or native radio list (`Outcome 7A92`, `C144`, `F801`), not an unlabeled swipe carousel;
- persistent `Question`, `Evidence`, and `Predicate` tabs using the WAI-ARIA tabs keyboard model where tabs are used;
- a 64-88px bottom action area with Defer, Reject all, and Continue, respecting safe-area insets;
- identity reveal as a full-height sheet with a visible Close button and focus return, never a hover tooltip.

All content fits at 320-375 CSS pixels without horizontal page scrolling. Structured JSON wraps or opens in a full-screen code sheet with line wrapping on by default. The predicate preview is prose first and generated test code second. `Finalize` is unavailable until the user has visited the predicate summary; this is stateful workflow validation, not a disabled button with no explanation.

Mobile may be used for finalization only when Captured/Projection operations, original witness, not-asserted fields, and candidate reveal are all available. If those cannot be made usable, mobile becomes triage/defer-only. Hiding critical evidence to make the design fit is not an acceptable responsive strategy.

## Screen behavior for every required state

| State | Primary copy and content | Available action | Forbidden implication |
| --- | --- | --- | --- |
| `EMPTY` | “No study yet. Compare 2-4 committed candidates under one declared plan.” Show a real local sample and exact commands. | Create sample; inspect study file | “Connect your AI” or fake live data |
| `PREPARING` | Exact current boundary: resolving refs, materializing, setup, readiness. Preview commands/environment/network policy before first execution. | Cancel; inspect plan | Candidate quality or progress race |
| `ACTIVE` | Trials completed/required, current phase, budget, elapsed time, candidate order schedule. Do not render provisional outcome winners. | Watch; cancel preserving evidence | Early majority or “likely stable” |
| `PARTIAL` | Completed evidence, missing work, why stopped, and which derived claims are ineligible. | Resume same plan; inspect; derive new plan | Treating partial as a Choicepoint |
| `ERROR` | Name the failing jurisdiction and whether any candidate behavior was observed. | Retry infrastructure; edit plan; export diagnostics | “Candidate failed” when runner failed |
| `UNSTABLE` | Per-candidate outcome histogram and trial identifiers; candidate excluded from stable map. | Add declared trials; inspect captured/projection; defer study | Choosing one flaky sample as behavior |
| `UNCOMPARABLE` | Materialization/world/control reason and affected candidate; stable candidates remain inspectable but no N-way claim is inflated. | Repair plan; rerun all required candidates | Empty output or contradiction |
| `DISCOVERED` | Original divergence and current reduction/confirmation state. | Inspect; continue; cancel | Resolve before fresh confirmation |
| `DECISION_READY` | Observed-stable counts, fresh confirmation, reduction grade, blind outcomes, exact scope. | Resolve; reject all; defer | Determinism, safety, or majority truth |
| `RESOLVED` | Human ruling, before/after reveal record, exact predicate, non-asserted fields, contract digest, conformance replay. | Open test; rerun; supersede | “Correct branch” |
| `DEFERRED` | Rationale, author, time, evidence freshness, revisit condition. No executable contract. | Reopen under same digest; derive current revision | Quietly treating defer as allow |
| `STALE` | Frozen old ruling and exact ancestor change that invalidated currency. | Inspect history; derive new Choicepoint | Editing history into freshness |
| `COMPLETED` | Counts of resolved/deferred/stale/unstable/uncomparable/unknown plus budgets and standalone residue. | Export report; run contracts | “Project verified” |

## Deterministic labels without a model

Countershape cannot honestly translate arbitrary program behavior into product language without domain semantics. It can make typed evidence readable through a strict label grammar:

- **HTTP:** method/path from the stimulus; status; body kind and size/shape; only distinguishing selected JSON paths; declared state-delta summary; truncation/absence tags.
- **CLI:** exit or signal; stdout/stderr present/absent/bytes; only distinguishing parsed fields; declared filesystem-delta summary.
- **Fallback:** `Projection <fingerprint> · 4 differing fields · captured artifact 1,842 B`.

Labels are pure functions of adapter version, projection schema, and canonical value. They are snapshot-tested, candidate-order invariant, locale-independent in v1, and never include branch/model identity. They use no evaluative adjectives such as secure, clean, expected, suspicious, or leaking. HTTP reason phrases may be shown only as protocol vocabulary and not as interpretation.

Human-authored semantic context appears in a visually distinct block labeled **Scenario context · authored**, while tool labels say **Observed outcome · generated deterministically**. The user may add a human label or rationale, but it is annotation, never equality input. If the typed projection still requires logs or source to understand in either reference fixture, the comprehension gate fails; the remedy is a better declared projection or seed description, not an invented explanation.

## Resolution vocabulary and scope

The exact v1 options are:

- **Allow one observed outcome**
- **Allow several observed outcomes**
- **Custom expectation**
- **Reject all observed outcomes**
- **Defer**

`Custom expectation` uses typed field/operator/value controls supported by the adapter and is the only route to an executable contract where none of the candidates conforms. `Reject all` records that implementation work is needed and emits no always-failing test. `Defer` emits no contract. “Refine” is an action that derives a new study/probe lineage, not a resolution that mutates the current Choicepoint.

The only compilable scope in v1 is displayed as a fixed, non-dropdown value:

```text
Scope: this exact witnessed stimulus
Not claimed: all cross-tenant requests, this endpoint family, or the generator domain
```

Selected assertion fields narrow what is promised about this witness; they do not broaden the quantifier domain. Do not show disabled “equivalence class” or “all generated cases” options that advertise an unsafe near-future affordance. Broader scope requires a separately designed operation with its own counterexamples.

## Trust copy dictionary

| Use | Never use |
| --- | --- |
| `Observed stable 5/5; confirmed in a fresh 5/5 batch` | `Deterministic`, `proven stable` |
| `No observed difference under 84 declared probes` | `Same`, `equivalent` |
| `BEST KNOWN under named reducers` | `Minimal`, `smallest`, `root cause` |
| `1-MINIMAL UNDER json.remove-key@1` | `The minimum` |
| `Conforms to this exact predicate` | `Correct`, `safe`, `best` |
| `Contradicts this exact predicate` | `Wrong branch` |
| `Candidate identity hidden in this view` | `Anonymous`, `bias-free` |
| `Captured after policy redacted 2 headers` | `Raw` |
| `Human ruling recorded` | `Approved safe` |
| `Study completed with 2 unknowns` | `Verified` |

If a study is explicitly tagged by its author as security-sensitive, show: **“This ruling records desired behavior; it is not a security review. An allowed outcome can still be unsafe.”** The system must not infer the tag from a `403` or `404` response.

## Accessibility, keyboard, motion, and evidence safety

The build target is WCAG 2.2 AA, including visible/unobscured focus, logical focus order, status messages, and minimum target sizing; the normative source is the [W3C WCAG 2.2 Recommendation](https://www.w3.org/TR/WCAG22/). Prefer native radio groups, checkboxes, buttons, details, tables, and dialogs. Where tabs are used, implement the keyboard and role model from the [WAI-ARIA Authoring Practices tabs pattern](https://www.w3.org/WAI/ARIA/apg/patterns/tabs/); APG is informative guidance, not itself conformance.

- Every state uses text, structure, and a symbol in addition to color.
- Focus never moves because polling updates. A single polite live region announces phase changes and completion, not every trial.
- Cancel, reject, defer, and finalize have distinct labels and confirmation semantics; no icon-only irreversible action.
- Normal Tab order is complete. Optional shortcuts are discoverable in `?`, disabled in text fields, and never the only route. Suggested desktop chords are `g q` queue, `g e` evidence, `[`/`]` previous/next outcome, and `Ctrl/Command+Enter` continue; users can turn them off.
- Dialogs/sheets trap focus, close with Escape when safe, restore focus to the invoker, and never hide evidence only on hover.
- Touch targets are at least 44px in the product design even though WCAG 2.2 AA’s minimum criterion is smaller.
- At 200% zoom and 375px width, content reflows without horizontal page scroll or obscured sticky controls.
- Motion only explains a transition from original to reduced or running to ready. Under `prefers-reduced-motion`, it becomes an instant state change with identical information.
- Candidate output is escaped and presented as inert text. ANSI/OSC, control characters, invalid UTF-8, and HTML-looking strings never become UI chrome or accessible names.

## Why adjacent products are insufficient references

[GitHub pull-request review](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/reviewing-changes-in-pull-requests) is organized around base/head branches, files, and diffs. Countershape must invert that hierarchy: scenario and outcome first, candidates after a provisional ruling.

[Chromatic](https://www.chromatic.com/docs/branching-and-baselines/) maintains an accepted “known good” branch baseline. [Playwright](https://playwright.dev/docs/test-snapshots) compares against a reference and warns that rendering varies by environment. Countershape has no trusted baseline: it compares peers, permits plural/reject/custom rulings, and emits selected-field predicates rather than a blessed snapshot.

[Socrates / Choose, Don’t Label](https://arxiv.org/abs/2604.08792) is the closest interaction ancestor: behavior clusters become a multiple-choice disambiguation query. Countershape’s product work is empirical repository evidence with world identity, instability, visible projection, plural rulings, and standalone exact-witness residue.

## Acceptance tests and build-loop gates

### Deterministic product tests

1. Candidate permutation, branch names, model/provider metadata, and group cardinality do not change aliases, label bytes, card weight, or blind-step ordering.
2. Blind endpoints and accessibility trees do not contain candidate identity or group size; identity is not merely hidden with CSS.
3. Revealing before a provisional ruling records `UNBLINDED_BEFORE_RULING`; changing after reveal preserves both choices and requires rationale.
4. Every outcome label is a pure snapshot-tested function; unknown/binary/oversized observations use the factual fallback.
5. No assertion is preselected. Finalization fails if no field is asserted or a differing field is unacknowledged.
6. Allow-many compiles a disjunction of tuples, not a cross-product. Unselected headers/body/latency never appear in emitted assertions.
7. `REJECT_ALL` and `DEFER` emit no test; `CUSTOM_EXPECTATION` can produce `none conforms`.
8. Projection changes make descendants stale and cannot overwrite the prior ruling.
9. `UNSTABLE`, `UNCOMPARABLE`, trial errors, and incomplete evidence never appear as selectable outcomes.
10. Mobile and CLI expose every trust boundary available on desktop, or mobile finalization is disabled with an explanation.

### Visual and accessibility loop

Capture and inspect, at both 1440×900 and 375×812, at least: empty, preparing, active, partial, error, unstable, uncomparable, discovered, decision-ready, predicate editing, identity reveal, resolved, reject-all resolved, deferred, stale, and completed. Run at least three visual passes, not one screenshot ceremony: information hierarchy; responsive/focus/accessibility; then independent critic and regression pass.

Automated gates include Playwright flows for blind-to-final, early reveal, custom-none-conforms, reject/defer, stale CAS, and mobile evidence inspection; axe with no serious/critical violations; keyboard-only completion; reduced-motion snapshots; 200% zoom; monochrome/forced-colors inspection; and no horizontal page overflow. A different-model visual critic should receive screenshots without implementation commentary and grade state clarity, equal outcome weight, trust-copy prominence, and mobile action safety.

### Human comprehension and bias test — still unvalidated

Recruit at least five builders matching the beachhead, none familiar with the fixture implementation. For each reference domain:

1. Without source diffs or raw logs, at least 4/5 can state the concrete difference among outcomes and identify exactly which fields their drafted predicate would and would not assert within three minutes.
2. At least 4/5 correctly answer that observed stability is not determinism, conformance is not correctness, Captured may be post-redaction, and reject-all emits no test.
3. In a counterbalanced repeat with outcome order, candidate identity, and candidate counts changed but behavior fixed, at least 4/5 retain the semantic ruling or give an explicit context-based reason for changing it.
4. Against a branch/diff-first baseline, the median time to an accurate decision record falls by at least 30% without a higher predicate-error rate.
5. VoiceOver on macOS and one nonvisual screen-reader pass can complete allow-one, custom expectation, and defer with no hidden state.

These are acceptance criteria, not results. Until this study is run, the claim that Countershape compresses review remains a 3/10 product bet.

## Severity-ranked findings

| Severity | Finding | Failure if unchanged | Required change |
| --- | --- | --- | --- |
| **P0** | Branch/model identity and group size appear before intent selection | provenance and majority become implicit oracle | blind facts and cardinality through provisional ruling; reveal before finalization with audit |
| **P0** | Outcome click implies full projection approval | brittle contract silently asserts incidental fields | explicit per-field Assert/Context; no defaults; exact preview |
| **P0** | Projection boundary is secondary or hidden | lossy normalizer becomes invisible product judgment | persistent Captured/Projection/Operations inspector and stale-on-change lineage |
| **P0** | Tool invents semantic labels without a model/domain rule | deterministic evidence is laundered into an explanation | typed factual label grammar; authored context clearly separated |
| **P1** | `STABLE`, errors, and Choicepoint states share one vocabulary | users resolve flaky or infrastructure output | three-level state model and `DECISION_READY` user label |
| **P1** | Reduced witness replaces original | a locally valid shrink asks a humanly different question | retain scenario, original, derivation, and grade on every decision surface |
| **P1** | Reject-all and custom expectation are conflated | impossible always-failing tests or inability to express none-conforms | separate operations; only custom expectation compiles |
| **P1** | Scope dropdown generalizes from one example | one ruling silently becomes a universal contract | exact-witness-only fixed scope in v1 |
| **P1** | Blindness is claimed as absolute | behavior leaks identity; legitimate context is unavailable | say identity-hidden; staged reveal; record early reveal/change |
| **P1** | Mobile hides evidence to fit | phone becomes a dangerous approval remote | full evidence parity or triage-only mobile |
| **P2** | Active UI shows provisional clusters/counts | early outcome becomes an anchor | progress and budgets only until decision-ready |
| **P2** | Dense custom widgets replace native controls | keyboard/screen-reader path becomes fragile | native semantics; APG only where needed; full keyboard gate |

## Exact brief changes required

The synthesis should edit `docs/CONCEPT_BRIEF.md` as follows before prompt-pack:

1. Replace the loose “human actions” line with the five-stage blind-first protocol: context, witness/original, blind outcomes, selected-field predicate, provenance reveal/finalization.
2. State that candidate identity, producer/model provenance, and outcome cardinality are absent from the blind payload, not CSS-hidden. Early reveal and post-reveal changes are audited; no claim of anonymity or bias elimination is allowed.
3. Add deterministic label rules and explicitly prohibit tool-authored semantic/evaluative outcome language in the no-model golden path.
4. Split state vocabulary into Study, candidate/trial eligibility, and Choicepoint lifecycle. Change user-facing `STABLE` to `DECISION READY` with observed and confirmation counts.
5. Require original scenario, reduced witness, derivation, reducer set, grade, budget, and unresolved-neighbor count on the decision surface.
6. Require an always-visible Captured -> Projection boundary with operation list; projection change creates a new lineage and stale descendants.
7. Specify that no projected field is pre-asserted. Every differing field must be explicitly Assert or Context; preview and Not asserted list precede save.
8. Define allow-many compilation as a disjunction of selected-field tuples.
9. Keep exact-witness as the only v1 scope and remove broader-scope controls, including disabled teaser controls.
10. Separate `CUSTOM_EXPECTATION`, `REJECT_ALL`, `DEFER`, and `REFINE`: only the first can compile a none-conforms contract; reject/defer emit none; refine derives a new study.
11. Add the exact 1440px three-column and 375px single-card information architectures, with full evidence parity or triage-only mobile.
12. Add the trust-copy dictionary as a product invariant and include it in snapshot tests.
13. Add WCAG 2.2 AA, keyboard-only, screen-reader, 200% zoom, forced-colors, reduced-motion, and candidate-output-inertness gates.
14. Add the comparative human-comprehension test to the success metric; do not claim review compression before it passes.
15. Add a product kill condition: stop the studio if users need source/raw-log archaeology for either reference case, if blind ordering/provenance materially changes rulings, or if emitted predicates over-assert.

## Ground-truth tally

### Directly supported by project artifacts or primary sources: 10

1. The current brief requires no model in the golden path, source-diff-free adjudication, raw/captured evidence access, mobile/desktop surfaces, and explicit unstable/uncomparable states.
2. The architecture lane requires outcome-labeled partitions, tri-valued reduction, fresh confirmation, exact selected-field predicates, exact-witness scope, immutable lineage, and separate reject/custom actions.
3. Socrates already presents behavioral clusters as multiple-choice program-disambiguation queries.
4. Google’s 5,217-review field experiment shows author hiding is penetrable and changes social dynamics; it is not a universal unbiased-review mechanism.
5. Controlled code-review research reports effects from apparent authorship with code quality controlled.
6. Empirical automation-bias research makes automated recommendation cues a legitimate human-factors risk.
7. GitHub review is explicitly base/head and source-diff centered.
8. Chromatic explicitly uses accepted branch baselines; Playwright documents environment-sensitive snapshots.
9. WCAG 2.2 specifies focus, reflow, status, contrast, and target requirements; WAI-ARIA APG documents expected tabs interaction.
10. No current artifact contains evidence that representative users can resolve either fixture accurately without source/log review.

### Strong design inferences: 9

1. Staged blinding is safer than either immediate provenance or permanent anonymity.
2. Outcome cardinality is a majority cue and should be hidden with identity during semantic selection.
3. A deterministic typed label is more honest than generated semantic prose in a no-model system.
4. No-default field selection is worth the added friction because contract over-assertion is irreversible residue.
5. Original and reduced stimuli must coexist for product comprehension.
6. Three-level states prevent infrastructure failures from becoming selectable behavior.
7. Mobile finalization is safe only with evidence parity.
8. A fixed exact-witness scope communicates the v1 truth better than a generalization dropdown.
9. A two-phase noninteractive ruling with digest compare-and-swap is the appropriate CLI safety boundary.

### Unvalidated bets: 7

1. Typed HTTP and CLI projections are understandable enough without semantic explanation.
2. Users will tolerate staged blinding and explicit assertion selection within the 15-minute target.
3. Hiding group counts improves decisions more than it frustrates users.
4. Original-versus-reduced context prevents causal misinterpretation in real cases.
5. A complete mobile decision flow is useful rather than merely possible.
6. The studio beats diff-first review on speed without increasing predicate errors.
7. Teams value an audited pre/post-reveal decision record instead of treating it as process theater.

## Confidence and verdict

- **State and information architecture:** 9/10. The jurisdictions follow the underlying artifact model and are testable.
- **Bias controls:** 7/10. Staged blinding removes obvious cues but cannot hide behavior-derived authorship or eliminate human bias.
- **Predicate safety:** 9/10 if field selection has no defaults and compiler previews are load-bearing; 3/10 for one-click outcome approval.
- **No-model comprehension:** 6/10 for structured HTTP; 5/10 for structured CLI; 2/10 for arbitrary opaque output.
- **Accessibility feasibility:** 8/10 with native controls and a narrow interaction set.
- **Mobile finalization:** 6/10; make it triage-only if evidence parity compromises clarity.
- **Review compression/adoption:** 3/10 until the human gate runs.

**Verdict: MODIFY, THEN PROCEED.** Countershape should build the CLI and studio only after the truth model exposes exactly the facts this protocol needs. The decisive product claim is not “we explain which implementation is right.” It is:

> Countershape lets a human inspect one empirically bounded disagreement, state an exact witnessed expectation without branch or model cues, see what the resulting test will and will not assert, then reveal provenance and preserve the ruling honestly.

Kill or narrow the human surface if either reference fixture requires source diffs or raw-log archaeology to understand, if candidate identity/group size changes rulings without an acknowledged reason, if mobile hides a trust boundary, or if a selected outcome silently freezes unselected fields. Those are not polish defects. They invalidate the promised operation.
