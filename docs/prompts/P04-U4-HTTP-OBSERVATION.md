# P04 / U4 — Prove HTTP observation and the falsification fixture

## Mission

In a fresh chat, add Countershape’s second typed observation domain: one fresh local cleartext HTTP/1.1 service, a fixture-owned readiness signal, one exact request, typed bounded capture/projection, repeated classification, and the flagship cross-tenant invoice baseline. Demonstrate why shared roots, one-shot observation, and hidden projection are unsafe. Do not add shrinking, Choicepoints, contract emission, end-user CLI, server/studio, report export, or external APIs.

Begin only after U3’s seal verifies strict-green. Preserve other work, give concise phase narration, and keep durable status/handoff files current.

## Required read set

Read completely: `/Users/drewnelson/.claude/CLAUDE.md`; `docs/CONCEPT_BRIEF.md`; `docs/ARCHITECTURE.md`; `docs/THREAT_MODEL.md`; `docs/STATE_MACHINES.md`; `docs/CLAIM_VOCABULARY.md`; `research/deep-dive/02-architecture-correctness.md`; `research/deep-dive/03-execution-security.md`; `research/deep-dive/04-product-dx.md`; `research/deep-dive/05-contract-portability.md`; `research/deep-dive/07-SYNTHESIS.md`; `research/deep-dive/08-RED_TEAM.md`; `docs/status/U2.md`; `docs/status/U3.md`; `docs/HANDOFF_MODE_C.md`; relevant U0 schemas/vectors; and the U1–U3 source/API surfaces. Verify branch, working status, U3 commit/seal, and `NO_COLOR=1 didrun verify --strict` before editing.

## Locked semantics and scope cuts

- HTTP stimulus, capture, projection, and future neighbors stay in `internal/adapters/http`. Generic identity/observe/compare code must not import HTTP, branch on HTTP fields, or coerce CLI into an HTTP/JSON shape.
- The proof fixture has no setup command or package install. Declarative seed files are written into each fresh world before start. The service signals readiness over a dedicated inherited pipe with a bounded fixed protocol; readiness is not an HTTP request and cannot warm application state.
- One trial starts one local service and performs exactly one cleartext HTTP/1.1 request to literal `127.0.0.1` on the allocated port. No external host, proxy, TLS, redirect, cookie jar, compression, chunked/streaming body, WebSocket, DNS, credential, or retry.
- Preserve method, path, ordered query multimap, ordered request-header multimap, and tagged absent/present-empty/body bytes. Use a constrained deterministic wire encoder/parser or equally strict implementation that cannot silently follow redirects, join relevant duplicate fields, auto-decompress, or use proxy environment.
- Capture transport-versus-response separately. Any complete status—including `500`—may be eligible behavior. Start/readiness/transport/timeout/output/protocol/projection/orphan/teardown failures are ineligible controls and cannot become status codes or outcome cards.
- Captured evidence retains the volatile request ID and scratch-root value. A visible, versioned projection removes exactly those declared noise fields while preserving status and disclosure-bearing metadata. Placeholder replacement proves only projected equality, never that root/port differences did not change earlier control flow.
- Reuse U3’s fresh rotated scheduler and U1’s exact `OutcomeMap`. No cache, shared-world mode, shape-only summary, or adapter-local classifier exists in product code.
- Scope is the curated Node-core fixture and `HOST_ALLOWED` trusted local execution. Do not claim imported-repository comparability, containment, network denial, general REST support, or semantic equality.

## Exact ownership

U4 owns:

```text
internal/adapters/http/**
testkit/httpfixture/**
testkit/studies/http_invoices/**
tools/check-u4-architecture.mjs
tools/mutate-u4.mjs
docs/status/U4.md
docs/HANDOFF_MODE_C.md
```

It may add domain-neutral hooks to U2/U3 only where the existing lifecycle/scheduler contract already calls for them. It must not add HTTP switches or fields to `internal/domain`, `internal/observe`, `internal/compare`, `internal/reduce`, or `internal/choice`. Do not create `cmd/`, store, emitter/Node harness, server, report, or web code.

## Implementation sequence

1. Define typed `HTTPStimulus`, declarative seed, readiness contract, start specification, capture policy, response/control sums, projection definition, and a closed field registry sufficient for the invoice proof. Preserve all ordered multimaps and absent/empty distinctions in canonical identity.
2. Compile the HTTP inert spec into deterministic `WorldPlan` while leaving actual port, roots, process facts, and readiness receipt in `WorldInstance`. Allocate loopback resources only per attempt. Make the comparison envelope list measured, required-equal, tolerated, rejected, and uncontrolled dimensions explicitly.
3. Add fixture-owned readiness via a dedicated inherited descriptor/pipe. Accept exactly the bounded protocol specified by U0, close it after success, and classify EOF, malformed, extra bytes, timeout, and process exit as readiness controls. Do not probe the service to decide readiness.
4. Implement one constrained request. Prefer direct TCP bytes for this intentionally narrow profile. Bound status line, headers, and body; require the declared content-length/body profile; preserve response metadata needed by the closed registry; reject unsupported transfer/content encodings and protocol shapes. Always perform U2 teardown, even after transport or parse failure.
5. Convert only a complete response plus successful teardown into typed HTTP captured evidence. Run visible projection operations through strict canonical parsing and extraction. Store the pre-projection captured body privately and link every projected value to its operation/source.
6. Reuse centralized eligibility, repeated classification, rotated schedules, and exact map creation. Candidate order and allocated port/root may change evidence but cannot alter map identity.
7. Build four dependency-free Node-core candidates: A observed-stable `403`; B observed-stable `404`; C observed-stable metadata-bearing `200`; D alternates `404`/`200` across actual invocations. Every response includes volatile request ID and scratch root. External fixture instrumentation must prove invocation and drive only the deliberate D fixture; it is excluded from selected projection.
8. Build a deliberately naive **test-only** shared-root runner whose state contamination makes B and C appear equal. Label its result `NEGATIVE_FIXTURE_NON_PRODUCT`; ensure no production API can enable shared roots or import this helper. Then prove fresh product worlds restore the actual B/C split.
9. Expose a test-readable Captured-to-Projection transcript. Mutating the projection to remove status or disclosure-bearing metadata must change/fail the expected map; volatile-only removal must preserve it.
10. Update status/handoff with the observed baseline, negative fixture meaning, exact readiness/wire profile, and unresolved proof work for U5+.

## Acceptance, negative cases, and mutants

Tests must cover method/path/query/header order and duplicate semantics; absent versus empty body/header value; exact request bytes; no proxy/DNS/redirect/compression/cookie behavior; readiness success and every malformed/timeout/early-exit path; status `200/403/404/500` eligibility; connection refusal/partial response/bad status/header overflow/body overflow/content-length mismatch/unsupported transfer encoding controls; fresh ports/roots/attempt markers; ordinary descendant teardown; projection visibility; strict JSON failures; and candidate permutation invariance.

The invoice study must run enough fresh trials to classify A/B/C as `OBSERVED_STABLE(k/k,h)` and D as `UNSTABLE`, exclude D from the exact stable map, retain volatile fields in captured evidence, remove them only through named projection, and yield three fingerprints for A/B/C. The naive helper must demonstrate the wrong B/C equality while product execution cannot select that path. This is negative evidence, never part of a Choicepoint.

Architecture checks must prove generic packages contain no HTTP imports/field names/type switches, CLI types remain unchanged and typed, and only one canonical outcome-map digest reaches study output. The mutation driver must anchor and kill at least: readiness-by-HTTP request; shared-root product option; proxy environment use; redirect following; status `500` as infrastructure failure; transport failure as status; teardown failure admitted; volatile removal hidden from transcript; status removal accepted; disclosure metadata removal accepted; D classified by majority; and map comparison reduced to group shape. Missing anchors are failures.

Wrap all load-bearing commands through didrun, at minimum:

```text
didrun run -- go test ./internal/adapters/http ./internal/observe ./internal/compare ./testkit/httpfixture ./testkit/studies/http_invoices
didrun run -- go test -race ./internal/adapters/http ./internal/observe
didrun run -- go test ./testkit/studies/http_invoices -count=10
didrun run -- node tools/check-u4-architecture.mjs
didrun run -- node tools/mutate-u4.mjs
didrun run -- git diff --check
```

Add a didrun-wrapped study command/assertion that records trial multiplication and actual timing. Do not yet claim the final under-15-minute two-study metric, reduction, decision readiness, or contract residue.

Use `didrun show --session`; bind explicit claims with commands of the form `didrun claim tests-pass --label "U4 HTTP observation" --event N --path .` to the exact normal/race/repeat/architecture/mutation/study events, plus a scoped diff-check claim. Stage only U4 changes, inspect, commit `feat: add typed HTTP observation`, seal HEAD, and loop `NO_COLOR=1 didrun verify --strict` to exit 0. A failed gate means U4 is unfinished: fix the real defect, rerun via didrun, create new event-bound claims, add a new fix commit if already sealed, reseal, and reverify. Never weaken tests, delete/relabel claims, or rewrite failed receipt history.

## Stop/fallback and handoff

Stop if readiness can mutate the application, an unsupported protocol is silently accepted, any control becomes an outcome, shared execution enters product evidence, projection hides a decision-bearing field, generic truth gains HTTP branches, the B/C contamination proof is not materially corrected by fresh worlds, or required mutants survive. The honest fallback is a one-domain CLI instrument or observation-only prototype, not a full reference system.

If didrun misbehaves, log exact command/output/version as live S6 evidence and mark affected capability `UNRECEIPTED`. Update `docs/status/U4.md` and Mode C, then report commit(s), verbatim grades, strict exit, native OS/Node exercised, classification and negative-proof results, timing, nonclaims, and the exact U5 reducer prompt from `docs/PROMPT_PACK.md` as the next unit.
