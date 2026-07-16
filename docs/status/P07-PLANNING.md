# P07 planning status — portable ruling and standalone authority lock

- **Unit:** P07 planning correction after implementation-time red team
- **Branch:** `codex/countershape-autopilot`
- **Implementation commit:** `e2cc2aa80f19d9a48f7265f20e8e0d46e4cd133a`
- **Implementation tree:** `875782ca8effaa4fc621fda065bb030910ba3955`
- **Receipt state of that commit:** failed; `13/28 claims recorded-exact`, with one `FAILED`, fourteen `STALE`, and thirteen semantically misbound positive rows after a didrun concurrent-append fork
- **Recovery boundary:** the follow-up commit carrying this file, `DIDRUN_BUGS.md`, and `HANDOFF_MODE_C.md` must be sealed and strict-clean from a fresh serialized ledger before P07A source work begins

## Why the original P07 prompt was rejected

Sealed U6 cannot honestly emit the originally proposed standalone contract:

1. every fresh Choicepoint uses the whole canonical projection, so its ruling cannot authorize selected fields such as `http.status` or `cli.stdout.json.mode`;
2. canonical stimuli retain content digests but not every source byte needed for later standalone execution;
3. changing historical adapter projection envelopes would invalidate already sealed fingerprints and evidence;
4. the inherited-listener HTTP profile is not reconstructible by a Node-only parent, and a high-level Node HTTP client normalizes bytes differently from the Go wire parser;
5. the original planning bundle put a digest inside its own preimage, omitted recoverable bytes, mixed later absence evidence into deterministic source, reused a 2–4-candidate historical world for a later one-target run, and allowed terminal residue to behave like a mutable execution head; and
6. the first repair still left run disposition and exact projected-tuple authority caller-pairable between target and execution objects.

The implementation-time deep dive changed the brief rather than rubber-stamping it. Independent Git-authority, schema-authority, and implementation-feasibility critics returned `CLEAN` only after the finalized run became the sole owner of terminal disposition and the exact projected tuple or control reason.

## Locked two-stage architecture

### P07A / U6b — adapter-bound portable ruling

P07A must preserve all historical CLI and HTTP projection bytes, fingerprints, and the existing 27-member Choicepoint body. Strict translators reopen roster-verified adapter projections under the exact projection binding and derive the unique closed portable profile for that exact adapter and binding. Every fresh public Choicepoint construction must use `ADAPTER_BOUND_PORTABLE_FIELDS_V1`, expose real selected/differing fields, and authorize selected-only custom expectations. Callers receive no legacy-mode switch. Legacy whole-projection rulings remain byte-identical and nonemittable.

Only `ALLOW_OBSERVED` and a separately reviewed selected-only `CUSTOM_EXPECTATION` may yield a compilable portable ruling. `REJECT_ALL`, `DEFER`, and `REFINE` remain noncompilable. P07B must reopen and revalidate that authority from the current store-bound ruling rather than trust serialized eligibility flags or tuples.

This is semantic ruling authority only. It is not standalone-source authority.

### P07B / U6c — byte-complete standalone residue

P07B requires both the current store-bound portable ruling and an exact live `PortableSource` preimage. The source reconstructs the exact plan, binding, minimized stimulus, and execution binding; it carries no confirmation tuple, projection, or proof bytes. Compilation independently reopens and retranslates the sealed confirmation proof bytes under the source-matched portable profile.

The initial launch family is only `NODE_REPO_SCRIPT_V1`: historical authority covers logical `node` plus an exact clean repository-relative `.js`, `.mjs`, or `.cjs` path, while the fresh target admits the current Node runtime. No claim equates that runtime with a historical absolute binary.

HTTP standalone execution uses raw `node:net` and requires a new physical `NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1` plan, observation, confirmation, portable Choicepoint, and selected-field ruling before emission. Existing inherited-listener P07A rulings remain structurally nonemittable and cannot be reused. Go/Node parity remains an external ship gate, never runtime transition authority.

The deterministic six-file bundle embeds every byte. The manifest covers the other five files; an outer bundle digest covers all six, avoiding self-reference. Staleness is checked before object or temporary-file creation, the bundle is durable before the terminal `RESIDUE` head advances, and post-residue materialization failure remains retryable. Later target, run, and execution objects are immutable nonhead objects.

## Final authority split

The corrected planning layer contains 21 closed object schemas and examples. The load-bearing later-execution objects are:

| Object | Sole authority | Forbidden authority |
| --- | --- | --- |
| `ContractExecutionTarget` | exact reopened bundle/source profile, one explicitly pinned and inspected private materialization, one fresh durable conformance attempt, and one admitted/revalidated Node runtime | self-digest, dirty worktree authority, arbitrary executable family, or tuple/result authority |
| `FinalizedContractRun` | exact target and attempt, closed lifecycle, constructor-derived `ELIGIBLE_CLEAN` or `INELIGIBLE_CONTROL(reason)`, and the exact projected tuple when present | caller-selected disposition, mismatched target/attempt, or projected-ineligible smuggling |
| `ContractExecution` | immutable classification derived from the exact target and exact finalized run | independently authored tuple, control reason, runtime, attempt, or result authority |

Two refusal vectors are mandatory: copied target authority and target/run mismatch. Validators freeze exact root and nested rosters, the Node-only source profile, target/run/execution joins, safe integer bounds, recursive receipt-cycle rejection, exact six-file bytes, tuple membership, and source/confirmation binding. Hostile mutants cover authority resurrection, runtime-family widening, target/run/attempt substitution, tuple and reason substitution, eligible-without-projection, projected-ineligible smuggling, vector shrinkage, and source decoys.

## didrun incident and receipt boundary

The implementation tree itself passed the final recorded commands at stored events `891`–`904`: generator syntax, deterministic generation, executable fixture plus tamper refusal, validator syntax, planning validation, hostile validator self-test, architecture gate and hostile self-test, full Go suite, Go vet, staged whitespace, no-unstaged-change, exact 29-path inventory, and scoped credential-prefix scan all recorded exit `0`.

Those raw events are not usable receipts. Earlier concurrent syntax checks forked the ledger at stored index `865`. Explicit claim numbers thereafter addressed physical list positions rather than the displayed stored indexes, shifting each label backward. The implementation commit's strict report happens to exit `1` because the first misbound final claim resolves to recorded exit `2`; strict itself does not detect the broken chain. All 28 claims in that manifest support no capability. The failed report and complete ledger remain under `.didrun-history/2026-07-15-p07-concurrent/.didrun/`; see S6-07.

The recovery is a new follow-up commit on a fresh ledger. Every command and claim must run serially, each successful command must be claimed immediately without `--event`, `didrun show --session` must exit `0` with an intact chain before acceptance, the exact commit must be sealed, and `NO_COLOR=1 didrun verify --strict` must exit `0`. Both gates are required because strict alone does not inspect chain integrity. The old commit must never be resealed because didrun replaces a commit's note with `git notes add -f`.

## Nonclaims

This planning unit does not implement portable selected-field translation, selected-field Choicepoint construction, portable rulings, source reconstruction, bundle compilation, residue publication, materialization, standalone execution, Go/Node parity, the product CLI, server, studio, visual design, packaging, security, production readiness, adoption, or maintainership. It receipts no Node/OS/runtime tuple, cross-platform or framework portability, actor authenticity, confidentiality, external-network denial, or entrypoint/manifest self-authentication. The generated six-file fixture exercises deterministic planning/example coherence only; it is not runtime emission or store-bound authority.

## Next boundary

P07A/U6b is the next bounded source unit: unchanged historical adapter bytes, strict portable translators, `ADAPTER_BOUND_PORTABLE_FIELDS_V1`, selected-field Choicepoints/rulings, real selectable/differing fields, selected-only custom expectations, and legacy nonemittable preservation. `PortableSource`, HTTP child-bind readiness, bundle emission, `RESIDUE`, and standalone execution remain P07B work.
