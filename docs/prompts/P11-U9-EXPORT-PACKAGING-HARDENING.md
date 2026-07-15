# P11 — U9 local evidence export, packaging, documentation, and hardening

You are implementing Countershape U9 in a fresh chat after both U8 prompts are committed, sealed, and strict-verified. Work autonomously, in the Soren Vale posture: evidence boundaries are product features, refusal is preferable to a polished lie, and the local artifact should feel unusually complete without pretending to be production-hardened.

## Objective

Ship a coherent Darwin reference package around the verified truth spine: a default-minimized, inert local evidence export; an embedded-studio single binary; clean-fixture installation and smoke paths; public-facing documentation, license, contribution/security boundaries; adversarial export/package tests; and a measured performance/limitations ledger. This prompt does not produce the final didrun HTML or final receipts handoff—that is P12—but it must leave one sealed U9 commit that P12 can audit without implementation changes.

Run at least three passes: export confidentiality/injection correctness; clean-root packaging and documentation DX; adversarial/performance regression. Tests are load-bearing but do not replace opening and visually inspecting the exported artifact or exercising the installed binary.

## Read before editing

Read completely:

1. `docs/CONCEPT_BRIEF.md`, especially Standalone contract boundary, Execution and security boundary, Budgets, Acceptance matrix, and prohibited claim language.
2. `docs/SEMANTICS.md` and `docs/PROJECTION_ALGEBRA.md`; exports must preserve their distinctions without recomputing authority.
3. `research/deep-dive/03-execution-security.md`, `04-product-dx.md`, `06-feasibility-acceptance.md`, `07-SYNTHESIS.md`, and `08-RED_TEAM.md`.
4. P09 and P10, both U8 ledgers, the current `docs/HANDOFF_MODE_C.md`, and every existing README/security/packaging file.
5. The actual CLI, report/store, embedded web asset, fixture, contract-bundle, and didrun-receipt integration code. Preserve their truth jurisdictions.

Run the prior sealed commit's `NO_COLOR=1 didrun verify --strict` before changes and inspect `git status --short`. If it is nonzero, do not start U9; repair or emit Mode C for the owning unit.

## Locked cuts and exact ownership

This unit owns `internal/report/**`, `internal/assets/**`, report/export wiring under `cmd/countershape/**`, packaging scripts/config under `scripts/**` and `packaging/**`, root build metadata needed for distribution, `testkit/export/**`, `testkit/package/**`, `evidence/performance/**`, `docs/PERFORMANCE.md`, `docs/LIMITATIONS.md`, `docs/THREAT_MODEL.md`, `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `LICENSE`, and `docs/build-loop/U9-HARDENING-LEDGER.md`. It may repair earlier packages only for a failing adversarial or packaging invariant and must record the cross-unit edit and rerun that unit's suites.

Use Apache-2.0 for this open-source infrastructure reference unless an already-committed repository decision says otherwise. Do not imply trademark/domain/package-name clearance. Do not publish, push, create a release, sign binaries, contact users, or call an external service. Build local release-shaped artifacts only.

The full runtime claim is Darwin on the exact natively tested host. A Linux cross-build is compilation evidence only and remains `UNRECEIPTED` as Linux runtime. Claim only the Node major actually exercised. No Docker golden path, remote execution, hostile candidate isolation, cloud service, safely shareable report, security certification, universal repository support, or review-compression claim.

## Pass 1 — default-minimized local evidence export

Implement a deterministic, self-contained local report/export from server-owned immutable artifacts. The default includes study/Choicepoint identities, named comparison envelope, measured counts, original and minimized stimulus previews, reduction grade/budgets/unresolved count, visible Captured → Projection operation descriptions, eligible outcome facts, exclusions with typed reasons, human ruling, selected and explicitly nonasserted fields, contract digest/current conformance evidence, and receipt references exactly as stored.

Default export must omit captured bodies, private attempt roots, bearer/CSRF tokens, environment values, secret slots, raw process streams, user home paths, and candidate producer/model provenance not included by the selected reveal policy. Include only fields shown in a mandatory preview, and record the versioned minimization/redaction operations. Put this warning prominently in the artifact and CLI confirmation: **`CONFIDENTIALITY NOT ESTABLISHED`**. Never call the default safe, sanitized, anonymous, or shareable.

`--include-raw` is an explicit dangerous opt-in with a strong warning and preview/confirmation behavior appropriate to a local CLI. If noninteractive use is supported, require an unambiguous danger flag rather than silently assuming consent. Redacted data is never called raw. Captured private artifacts remain mode 0600 where the platform supports it; report/output creation is atomic and refuses unsafe paths or overwrite races.

Render all untrusted candidate and receipt text as inert text. Escape HTML/SVG/script contexts structurally; use a no-network/no-script CSP where HTML is the format; include no external fonts, scripts, images, trackers, or links that auto-fetch. Visibly encode ANSI/OSC, control and bidi characters. Receipt grades round-trip verbatim—even an unfamiliar value—and absent evidence is exactly `UNRECEIPTED`. Countershape never parses a didrun grade into conformance or a stronger label.

Build adversarial fixtures with `</script>`, SVG/onload, style escapes, ANSI/OSC links, bidi overrides, forged section headings, huge/unbroken values, path traversal, duplicate-looking IDs, unknown grade strings, token-shaped secrets in selected and nonselected fields, and output-size limits. Assert zero execution/network on file open, no raw secret fixture in default output, all required minimum facts present, deterministic bytes across clean roots, bounded file size or explicit truncation with digest/count, print usability, responsive 375px layout, and a clear distinction between omission, truncation, redaction, absent, and empty. Fixture secrecy tests prove fixture coverage only; docs must say they do not establish arbitrary-secret absence.

Open the generated report through the browser skill under `file://` or the actual supported local route, inspect it at 1440×900, 375×812, print preview, forced colors, and with hostile payloads. Record screenshots and findings. This is one visual hardening pass, not a substitute for U8's loop.

## Pass 2 — one-binary package and first-run DX

Embed the production React/Vite assets into the Go binary with an explicit build step and stale-asset detection. The installed binary must run `--help`, version/about, CLI reference studies, local studio, and report export without a checkout, `node_modules`, Vite dev server, or package registry. Be precise: Node remains the fixture/generated-contract runtime for the exercised studies, not an invisible Countershape runtime dependency.

Create a clean temporary Git repository/package test that copies only the release-shaped binary and declared fixture inputs. Verify first-run warning, literal loopback bind, fragment-token flow, static assets, CLI next action, report generation, and generated standalone bundle. Inspect binary metadata and packaged file manifest for absolute build paths, credentials, `.didrun`, private evidence, source maps, dev endpoints, and accidental user files. Rebuild from two clean roots and compare every artifact that is claimed deterministic; if the Go binary itself is not byte-reproducible, document that and narrow the deterministic claim to the exact objects that are byte-identical.

Keep the quickstart real and short: use checked-in dependency-free reference fixtures, show the exact trusted-code warning, demonstrate one disagreement and selected-field residue, and state what the output does not prove. Documentation must distinguish `TreeIdentity`, materialized files, comparison envelope, captured evidence, projection, ruling, emitted bundle, current execution, and didrun receipt. Include Wake's boundary: refs/provenance may come from Wake, but Countershape is not an agent runtime, replay system, fleet manager, or friendlier Wake UI.

Write `THREAT_MODEL.md` with trusted local/full-user permissions/host-network reality, process-group cleanup limitation, localhost auth exclusions, captured/export sensitivity, generated-contract duplication risk, and no hostile-code containment. `SECURITY.md` defines private reporting guidance and supported scope without claiming an audit. `CONTRIBUTING.md` explains invariant-first changes, typed adapter boundaries, tests/mutants/fixtures, didrun discipline, and how to add an adapter without domain coercion. `LIMITATIONS.md` contains every cut and fallback product. `PERFORMANCE.md` states machine, OS, architecture, tool versions, exact fixtures, budgets, repetitions, and no imported-repository extrapolation.

## Pass 3 — adversarial, reproduction, and performance closure

Run the full Go/web/Playwright/axe/security/export/standalone suites plus race tests and relevant mutation tests. Exercise malformed store objects, stale CAS, illegal ruling-to-emitter calls, same-shape/different-map propagation through report DTOs, unstable/ineligible candidates, output limits, cancellation at sweep completion, physical fresh confirmation, and Countershape-absent Node bundles. No report view may reinterpret a domain state.

Run both dependency-free reference studies three times from clean roots with fresh evidence. The stable objects claimed deterministic—compiled Plan, DecisionRecord, and ContractBundle—must retain their expected bytes/digests; WorldInstance and captured/confirmation artifacts must remain physically new and linked. Record wall time and trial multiplication for each run on the named machine. The two studies must meet the brief's under-15-minute target or the metric must be honestly revised before commit. Never add caches, prepared roots, reset hooks, reduced repeats, or hidden retries to hit the number.

Run a dependency/license inventory and secret scan using locally available tools. Do not download a scanner merely to satisfy a checkbox. Log unsupported checks as `UNRECEIPTED`; do not fabricate an SBOM, signature, provenance attestation, security review, or cross-platform matrix.

## didrun unit boundary and strict loop

Every load-bearing format, lint, test, race, mutation, web build, browser/report inspection command, package smoke, clean-root reproduction, timing run, license inventory, and secret scan must run through `didrun run --`. Manual visual judgments and legal/security/market conclusions remain explicitly outside command grades.

At the boundary, inspect events and declare only truthful supported claims. Stage the exact U9 scope, inspect the staged diff and artifact manifest, commit without an AI co-author trailer, seal the commit, and loop on `NO_COLOR=1 didrun verify --strict` until exit 0. A nonzero exit means U9 is not done: fix the underlying problem, rerun affected commands through didrun, declare new honest claims, commit, reseal, and reverify. Never weaken a test, delete a claim, relabel an old receipt, or treat old failed receipts as erasable history.

Update `docs/HANDOFF_MODE_C.md` with the commit, exact grades, release-shaped artifact paths/digests, report screenshots, performance table, platform/runtime matrix, known findings, and P12 as the next prompt.

## Stop and fallback

Stop and hand off if default export can execute or auto-fetch; a known fixture secret leaks; unknown receipt grades are upgraded; packaging needs dev services; prior truth invariants regress; or the reference studies need evidence reuse to meet timing. Permitted fallbacks are no-export, checkout-required developer build, CLI-only, DecisionRecord-only residue, one-domain instrument, or observation/approval prototype. Name the cut; never inherit the full-system claim.
