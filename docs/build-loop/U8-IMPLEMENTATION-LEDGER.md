# U8 secure-studio implementation ledger

- **Unit:** U8 functional and security boundary (P09)
- **Base commit:** `9bf3b243d69f4253bfcdf706995e99e2edd63e67`
- **Base tree:** `725fbeeb685726ffe8b066d2b3962f8a7fa08a3d`
- **Base subject:** `docs: receipt U7 reference milestone`
- **Status:** functional/security candidate green in development; final didrun
  events, claims, commit, seal, strict result, and receipt do not yet exist
- **Runtime scope:** Darwin/arm64, local loopback, trusted local candidate code, host network allowed

## Prior-boundary admission

The checkout resumed clean at the sealed U7R commit with no live didrun, verifier,
Go-test, Vite, Playwright, or Countershape process. The sealed U7R note, archived
ledger, and HTML witness remain the source boundary. The post-rotation shorthand
and explicit-commit commands

```text
NO_COLOR=1 didrun verify --strict
NO_COLOR=1 didrun verify --commit 9bf3b243d69f4253bfcdf706995e99e2edd63e67 --strict
```

both exited `1` with `UNKNOWN` and `0/9 claims recorded-exact` because the live
ledger had already been rotated. They did not report a tree or claim mismatch.
This is a post-archive rehydration limitation, not a new U7R receipt. U8 will not
restore, copy, or rerun the U7R ledger; the existing commit-addressed note, HTML,
and private archive remain the prior strict witness.

## Locked functional acceptance

1. `countershape studio` binds only literal `127.0.0.1` on an ephemeral port.
2. The launch credential is a high-entropy URL fragment, transferred to
   `sessionStorage` and removed from the URL before any API request. It never
   enters a query, cookie, Referer, log, ordinary process output, or candidate
   path.
3. Every evidence/API read and every mutation requires exact Host and bearer
   authorization. Mutations additionally require exact Origin, JSON content
   type, session CSRF, and the current server revision digest. Static bootstrap
   assets contain no evidence and are the only unauthenticated content.
4. No permissive CORS header exists. CSP, `nosniff`, no-referrer, no-store, and
   same-origin resource policy are set on every response.
5. The decision-ready seed is parsed as an exact portable
   `choice.ChoicepointRecord`. Blind bytes come only from `choice.NewBlindView`
   through `choice.Session.BlindDTO`; reveal and final DecisionRecord bytes come
   only from the matching package session.
6. Blind serialized bytes contain no candidate key/ref/name, producer/provider
   metadata, total candidate count, per-outcome support count, hidden branch
   data, or cross-study alias. Distinct observed-outcome count remains visible.
7. Every differing field is explicitly acknowledged as `ASSERT` or `CONTEXT`.
   Nothing is preselected. The server checks exact acknowledgement coverage and
   requires the selected field set to equal the asserted field set before it
   calls `Session.Propose` or `Session.Revise`.
8. Required presentation visits are package-owned session facts. They establish
   presentation only, never comprehension. Provenance is unavailable before an
   explicit reveal transition.
9. Allow-one, allow-many, custom expectation, reject-all, and defer reach the
   typed choice validator. Allow-many remains complete tuples. Weak or empty
   selections fail closed. Reject/defer produce no executable artifact; U8 has
   no browser-facing emitter route.
10. Two-tab stale requests, replayed CSRF, guessed IDs, illegal methods/actions,
    form posts, `Origin: null`, hostile origins, and DNS-rebinding Host values
    fail with bounded typed errors and no mutation.
11. Candidate-controlled values remain inert text in JSON, DOM, accessibility
    state, and logs. Controls, ANSI/OSC, bidi, HTML-looking strings, path text,
    forged headings, and long tokens are visibly escaped/encoded where needed.
12. The exact trusted-code warning is visible before the decision surface:
    “Countershape will run trusted repository code with your user permissions
    and host network access. The temporary directory is for repeatability, not
    security isolation.”
13. Honest study/candidate/Choicepoint presentation states are renderable.
    Unimplemented durable workflows are visibly disabled and never presented as
    package authority.
14. Desktop 1440x900 and mobile 375x812 retain original/minimized/derivation,
    Captured -> Projection -> Operations, nonasserted fields, provenance reveal,
    exact scope, and a safe next action without page-level horizontal overflow.
15. Playwright exercises the real compiled binary and embedded assets. Axe has
    no serious or critical findings; keyboard completion, focusable mobile
    evidence scroll regions, hostile origin, stale CAS, and hostile content are
    load-bearing tests. U8 introduces no dialog or sheet, so dialog focus return
    is not an applicable P09 claim.

## Build-loop protocol

- Pass 1: server/session contract, refusal taxonomy, auth matrix, and exact
  blind-byte tests.
- Attack/fix cycle 1: hostile page, stale client, direct illegal-action API,
  token/storage/history inspection, and candidate-output injection.
- Pass 2: complete React flow, keyboard/mobile behavior, real-binary Playwright,
  accessibility, production embedding, and security regression.
- Attack/fix cycle 2: inspect network, serialized bytes, DOM/accessibility,
  storage, URL/history, and logs; add one regression per material finding.
- P10 begins only after this functional/security unit is committed, sealed, and
  strict-verified. P10 owns at least three visual passes plus the real Claude
  screenshot-only critic.

## Explicit ceilings

This unit does not establish hostile-code containment, browser-extension or
same-user-process resistance, anonymity, debiasing, comprehension, security
review completeness, candidate correctness, broad repository compatibility,
remote service safety, Linux/Windows behavior, or a receipt until its final
commands are run, claimed, committed, sealed, and strictly verified.

## Cross-unit changes

- `cmd/countershape/main.go` adds only the `studio` dispatch and keeps the
  existing machine-study path unchanged.
- `.gitignore` now admits exactly `web/dist/**`. Without this exception the Go
  embed compiled only on a dirty machine that happened to retain ignored build
  output; a clean checkout would lack the production bundle. `web/embed_test.go`
  pins the three-file embedded roster and refuses seeded evidence strings in
  unauthenticated bootstrap bytes.

## Attack/fix cycles

1. **Closed result-array shape.** The first real-browser path exposed `null`
   result arrays that crashed React. The server now emits explicit empty arrays
   and the state tests bind that shape.
2. **No implied human choice.** Manual browser inspection found the first
   outcome and every differing field preselected. Outcome selection and field
   acknowledgement now begin empty; all ruling paths explicitly select an
   outcome or authored value and mark every field Assert or Context.
3. **Presentation-boundary clarity.** Manual inspection found missing authored
   scenario/jurisdiction/budget context and controls that were inert but not
   visibly encoded. The UI now shows scenario, exact scope, discovery and fresh
   confirmation counts, reduction budgets, and the explicit Captured →
   Projection → Operations boundary. Candidate controls, bidi markers, and
   replacement characters are rendered as visible `\<U+XXXX\>` evidence text.
4. **Closed server diagnostics.** The HTTP audit found Go's default protocol
   error logger as an unnecessary output channel. The server binds it to
   `io.Discard`, and the hostile HTTP test verifies that exact writer along with
   duplicate-header, unknown-field, oversized-body, query, Host, Origin, CSRF,
   content-type, and stale-CAS refusals without state mutation.
5. **Stable stale-tab seam.** A browser test appeared to hang in cleanup after a
   UI label changed. Trace inspection showed the click locator consumed the
   entire timeout; cleanup was innocent. Evidence surfaces now expose a static
   package-surface identifier, and the real two-tab stale replay passed three
   repetitions followed by the full suite.
6. **Mobile keyboard access.** The first mobile axe pass found a serious
   `scrollable-region-focusable` violation on original/minimized witness panes.
   Those panes are now focusable with fixed, non-candidate accessible names.
   The focused mobile axe test and full real-binary suite pass.
7. **Clean-checkout embedding.** Final source audit found `web/dist` hidden by
   the repository-wide ignore rule even though Go embeds it. The narrow
   unignore plus embedded-roster test closes that machine-local false green.

## Manual browser inspection

The installed in-app browser-control skill required inspection through the
signed-in app browser instead of ad hoc HTTP simulation, and prohibited direct
reads of cookies or browser storage. The rebuilt real binary was inspected at
`1440×900`; the fragment disappeared before the first API view, no token was
visible in the document, zero outcome and acknowledgement buttons were pressed,
the exact two-field pending count was visible, the Captured → Projection →
Operations label was present, page-level horizontal overflow was absent, and
the browser log was empty. The accessibility snapshot contained the authored
question, exact scope, 3 discovery and 2 fresh-confirmation trials per
candidate, three equal-weight blind cards, all five evidence surfaces, fixed
ruling controls, and all four authority nonclaims. Behavior-based tests—not a
forbidden direct storage read—prove that a fresh context without the fragment
is unauthenticated.

## Screenshot inputs to P10

`web/capture/p09.spec.ts` runs a real embedded binary and captures five
sanitized inputs under `evidence/ui/p09-inputs/`: desktop and mobile
decision-ready, desktop identity-reveal, desktop resolved, and mobile error.
The exact sizes and SHA-256 values are in that directory's `MANIFEST.md`.
Two independent captures were byte-identical. These are inputs only; their
existence is not counted as a visual pass or human-comprehension evidence.

## Development command record

Development commands are diagnostics and receive no claim. Material terminal
results on the candidate were:

| Command / observation | Terminal result |
| --- | --- |
| `go test -mod=readonly -buildvcs=false -count=1 ./internal/server ./cmd/countershape` | exit `0`; server `10.622s`; command package compiled |
| `go test -race -mod=readonly -buildvcs=false -count=1 ./internal/server` | exit `0`; `70.651s` |
| `npm --prefix web run typecheck` | exit `0` |
| `npm --prefix web run lint` | exit `0` |
| `npm --prefix web run test` | exit `0`; 2 files / 3 tests |
| `npm --prefix web run build` | exit `0`; exact three-file production bundle |
| real-binary Playwright before the mobile axe repair | 12 passed / 1 failed on serious keyboard access |
| focused mobile axe after repair | exit `0`; 1 passed |
| full real-binary Playwright after repair | exit `0`; 13 passed in `39.6s` |
| `npm --prefix web run capture:p09` | exit `0`; 1 capture test / 5 images |
| independent current-binary recapture plus `cmp` | exit `0`; all five PNGs byte-identical |
| `go test -mod=readonly -buildvcs=false -count=1 ./web ./cmd/countershape` | exit `0`; embedded roster green; command package compiled |
| archive of `git write-tree` followed by offline `go test -mod=readonly -buildvcs=false -count=1 ./web ./cmd/countershape` | exit `0`; the exact staged tree compiles without ignored `web/dist` state |

## Final evidence plan

Only final-tree `didrun run --` events may receive claims. The intended closed
set is formatting/diff integrity, normal Go tests, race tests, web
typecheck/lint/unit/build, real-binary Playwright/axe/security flows,
byte-repeating screenshot capture, a committed-tree clean-checkout Go build,
and a bounded credential-pattern scan. Every claim will name its exact event
and P09 pathspecs. The tracked commit will remain honestly candidate-shaped;
the external didrun note, strict result, HTML, and archive are the post-commit
receipt and cannot be self-authored into the same commit.
