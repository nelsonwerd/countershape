# P07B-C C3R — sealed-note preview compatibility maintenance

Classification: `DEFECT_REPAIR`

- **Boundary:** active pre-seal `SOURCE_FULL` checker maintenance after sealed C3 and before the unchanged frozen C3B receipt unit. Classification: `DEFECT_REPAIR`.
- **Parent:** sealed C3 commit `029c5cf43853eb3cb46520f6e0dea8431a3de5c0`, tree `de3e02a343f8ecfb344bdb28d014342f5b423959`, is note-present with note blob `9ad8a864911ca30914463c61d3447a88beeda927`, note-body SHA-256 `f5eaa66d0cae152a9edc7f4ff2d49ba66c0d1eb8f3459171a2888daaaaf80fa8`, `18/18 claims recorded-exact`, strict exit `0`, and a logged redacted-export override.
- **Commit subject:** `fix: validate sealed C3 note previews`
- **Scope profile:** exact paths only, no prefixes, `SOURCE_FULL`.
- **Frozen descendants:** C3B remains exactly three receipt-only paths at `sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67`; its seven claim labels, types, and command argv remain unchanged.
- **Preserved draft:** C3B draft stash object `fc02f6722716f0600b1e1d7af28cb91925d1047c` remains retained without dropping.
- **Permanent red history:** `.didrun-history/2026-07-20-p07b-c-c3b-build-loop-pre-c3r/.didrun/` contains the first failed C3B plan event and zero claims. It supports no capability claim and is not reused by C3R.

C3R owns exactly these 7 paths: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C3R-NOTE-PREVIEW-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Their sorted-newline roster digest is `sha256:fe14d9f9985771cce0525edfca47c001b5912dacc4c2d6fef182458d3b44f58e`.

Every intended C3R grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3R itself.

## Trigger and exact defect

The first C3B plan command rejected the already-sealed C3 Git note at claim eight's argv preview. The local ignored C3 ledger contains the exact raw race selector:

`^(TestC3OfficialTargetClosedCapabilityAndDefensiveGetters|TestC3HostEpochConcurrentMeasurementsNeverCache|TestC3NodeRuntimeCopiedCapabilitiesSerializeRevalidation|TestC3ConformanceAttemptConcurrentValidationAndReopenAreRaceFree)$`

didrun's exported note projection preserved the other three alternatives but replaced only the first long identifier with `«redacted:high-entropy»`. The exporter also applied the already-admitted redactions at common hermetic-prefix positions. The prior checker accepted those exact prefix projections but required every tail argument to equal the raw local argv, so the one valid note projection failed.

This is a checker compatibility defect, not evidence corruption. The ignored ledger remains the byte-exact raw witness. The Git note and HTML are scrubbed projections. `TREE-EXACT` establishes that the recorded successful command was self-stable against the sealed tree; it does not promise that exported `argv_preview` bytes are unredacted.

## Repair and defensive boundary

The repaired predicate accepts one exact sealed-note argv projection only for C3 claim index seven (human claim eight): the complete canonical raw tail with exactly the first race-test alternative replaced by `«redacted:high-entropy»`. It still requires the full canonical argv length, exact allowed prefix positions, and byte equality everywhere else.

Hostile cases reject a marker on another claim, whole-selector replacement, a second alternative replacement, marker suffixes, tail drift, unrelated tail redaction, and any attempt to use the note projection as byte-exact raw local-ledger argv. The dedicated live integration gate reopens sealed C3 commit, tree, single parent, subject, note blob/type/body digest, all eighteen note claims, exact projection, coverage, and current-HEAD identity.

The phase repair also keeps the receipt fixtures stable across the interposed boundary. With no C3 receipt declaration, the only admissible current tuple is `C3R_ACTIVE`. Once both the exact C3 receipt and independently sealed C3R authority exist, the only admissible tuple is `C3B_ACTIVE`. Older complete tuples remain diagnostic-only; mixed, partial, duplicated, relocated, commented, fenced, raw-HTML-wrapped, or receipt-incoherent states fail.

C3R's operator seal procedure is outcome-dependent: run the plain seal first. If it succeeds, the note must truthfully record `secrets_override: false`; only after plain-seal refusal on reviewed aggregate high-entropy findings and the claimed scoped staged scan may the operator use a logged `--allow-secrets` retry that produces `secrets_override: true`. Portable C3R authority does not prove plain-seal-first sequencing, refusal, or review: its Git note records only the resulting boolean. C3B's canonical didrun metadata must render that exact recorded seal disclosure from the validated note, while explicitly preserving the same limitation. Neither value proves secret absence, a general credential audit, or publication authority.

## Intended claim map

| Claim label | Claim type | Grade before seal |
| --- | --- | --- |
| `P07B-C C3R note-preview plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3R note-preview and dual-phase defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3R sealed-C3 Git-note argv-preview compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3R unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3R cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3R cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3R exact seven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3R scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3R sealed-C3 predecessor and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Nonclaims

C3R changes no Countershape product behavior, target composition, store format, Git materialization, host/runtime measurement, verifier roster, cache/parallelism setting, repetition semantics, lock semantics, didrun installation, sealed C3 commit/note, or C3B source scope. It establishes no process-start authority, runtime freshness, secret absence, generic correctness of didrun's redaction heuristics, hostile-code containment, security approval, production readiness, cross-platform behavior, adoption, or maintainership capacity.

The logged C3 `secrets_override: true` means only that a redacted export override was used after the separately scoped review. It is not a secret-free finding. C3R must use its own fresh nine-event ledger; no event from C3 or the failed C3B development history may support a C3R claim.
