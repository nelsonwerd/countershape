# P07B-C — exact target, finalized run, and immutable classification

**Created:** 2026-07-18
**Companion evidence:** `research/deep-dive/p07b-c/00-scope-and-method.md` through `10-executive-briefing.md`
**Platform scope:** Darwin/arm64/cgo, the exact exercised Node runtime tuple, logical `node`, one exact clean repository-relative JavaScript entrypoint, CLI plus locked child-bind HTTP
**Validated against:** commit `464e47adbf7f4497dfafa89938a9539239ffd41b` on `codex/countershape-autopilot`

This pack turns the planning-only P07B-C shapes into a capability-only, crash-honest execution spine: three immutable nonhead semantic objects plus one store-private operational interlock that can block spawn but can never author or select a result. It is intentionally split into seven implementation units because target publication, cooperative spawn admission, physical run closure, standalone evidence, and classification are different truths.

## How to use this pack

Execute `C0 → C1 → C2 → C3 → C4 → C5 → C6`, one unit at a time. Every arrow is a fresh didrun ledger or preserved unit-local ledger, exact staging review, commit, seal, Git-note inspection, and `NO_COLOR=1 didrun verify --strict` loop. When exact source-commit identity or grades must enter repository documentation, use a second receipt-document sub-boundary (`Ca` source, then `Cb` reconciliation) rather than creating a self-referential commit. Do not begin the next implementation unit until its last required sub-boundary exits strict with `0`.

The user has explicitly authorized commits at verified shippable boundaries. That authorization does not allow pushes, releases, external provider calls, or weakening a failed receipt.

## Rules every unit inherits

1. Read `docs/CONCEPT_BRIEF.md`, `docs/ARCHITECTURE.md`, `docs/SEMANTICS.md`, `docs/STATE_MACHINES.md`, `docs/THREAT_MODEL.md`, `docs/PROMPT_PACK.md`, `docs/HANDOFF_MODE_C.md`, this pack, and the P07B-C deep-dive package before editing.
2. Verify every file/line reference against current code. Current source wins over remembered locations and API facts; if it disagrees with a locked semantic ruling, stop and amend the ruling explicitly rather than silently letting stale code broaden authority.
3. The root is the sole repository writer and sole didrun/Git/seal operator. Parallel agents are read-only critics.
4. Every load-bearing command runs through `/opt/homebrew/bin/didrun run -- ...`; immediately claim each successful final event before any next event.
5. Use `GOMAXPROCS=2`; every Go build/vet/test/race/fuzz command uses `-p=1`, and fuzz uses `-parallel=1` with a count budget.
6. Stage only paths admitted by the machine-readable unit allowlist and fail unexpected paths. Never commit `.didrun/`, private captures, runtime roots, or `.countershape` artifacts. Never add AI co-author metadata.
7. After commit, seal, require the exact Git note, and loop on strict verification. A nonzero gate means the unit is unfinished. Never weaken tests, remove claims, relabel, or erase failed history.
8. Preserve the current didrun implementation for the longitudinal run. Record any misbehavior in `docs/status/DIDRUN_BUGS.md`.
9. Keep verification black-box/property/parity/recovery/boundary based. Do not recreate a dense recipe-level source-rewrite corpus.
10. Real external APIs are human-gated and are never simulated. P07B-C needs none.
11. Trusted local candidate code runs with the user’s permissions and host network. Private roots/process groups are repeatability and cleanup controls, not containment.
12. No public API may accept a semantic body/digest bag and turn it into live authority.

## Locked decisions

### Three immutable jurisdictions

- `ContractExecutionTarget`: pre-spawn authority over one exact reopened bundle/residue, one Git-pinned inspected/revalidated private materialization, one fresh durable attempt, and one admitted/revalidated Node runtime.
- `FinalizedContractRun`: exact closed physical facts for one consumed target/attempt, including a bounded closed-run witness and separate process/scope axes.
- `ContractExecution`: separately persisted classifier-profile-bound conclusion over the exact target/run pair. It owns no tuple, process reason, or scope reason.

All three are nonhead semantic objects. None is a mutable execution result, selector, list entry, latest pointer, or study-head stage. The sole mutable exception is the store-private `ExecutionInterlock` below: it is operational spawn exclusion, not semantic evidence or discovery authority.

### Official publication and discovery

- Generic CAS object existence proves canonical bytes only. Official target/run/execution reopen requires a store-private typed publication witness and an exact create-once relationship.
- The closed mappings are `attempt digest -> target digest`, `target digest -> finalized-run digest`, and `(finalized-run digest, classifier-profile digest) -> execution digest`. Resolution requires the already-held typed parent key; no API accepts a search predicate or returns siblings/newest values.
- Semantic reopen is exact-digest/typed-parent-only. P07B-C adds no result listing, filesystem traversal, `latest`, mutable result/status record, or semantic execution head.
- Publication returns exact digests. Lost output is not solved inside C; a future immutable collection is a separate design.

### Cooperative spawn admission

- `ExecutionInterlock` is one store-private, store-wide operational CAS. It is either `CLEAR` for a measured Darwin boot-session identity or `HELD(target, boot-session)`. It serializes candidate spawn only; it is neither canonical evidence nor a source of run/classification facts.
- A production acquisition requires an opaque C3-issued official target capability. C2 may implement/test storage mechanics but has no production issuer and cannot make a parsed C1 body official.
- A target-keyed `StartClaim` is durable intent and a permanent tombstone, never proof a child started.
- The C4 contract runner first durably acquires/reopens the interlock for the authentic target, then creates/reopens StartClaim. Only that winner receives one opaque process-local `RunPermit`. Generic publish/read, a losing publisher, copied owner data, a test fixture outside `_test.go`, or restart never recreates it.
- `SpawnObservation` separately records `Start` error or observed child PID. Missing observation is unknown.
- The honest guarantee is serialized cooperative at-most-once spawn admission per exact target, not exactly-once execution.
- Conclusive owner-produced terminal closure releases the interlock only after the terminal witness is durable. Any crash/ambiguity after intent permanently consumes the target and leaves the interlock held on that boot session. A fresh target identity alone is insufficient retry authority.
- Clearing an ambiguous interlock requires explicit operator reset plus a measured Darwin boot-session identity demonstrably distinct from the held identity. If that fence cannot be proven, P07B-C stops before every further subject spawn. It never claims the prior child absent on the same boot.

### Closed run and scope

- One process-control owner preserves at most one primary cause plus independent teardown/orphan controls; cleanup cannot overwrite the primary cause.
- Standalone scope is a separate `COMPLETE | PARTIAL | VIOLATED` sum over exactly five checks: target inventory, child bindings, import resolution, service bindings, and named parent-secret sentinel inheritance.
- Only clean projected process closure plus standalone `COMPLETE` without violation is eligible.
- `PARTIAL` and `VIOLATED` force ineligible classification without being relabeled as a process `ControlReason`.
- A bounded canonical `ClosedRunWitness` contains typed refs and summaries. Raw/large evidence remains private behind a manifest, explicit ceilings, default omission, and a confidentiality nonclaim.

### Classification

- Classifier profile v1 is exact complete-tuple membership under the bundle’s exact selected-field roster.
- An eligible tuple in the allowed set derives `CONFORMS`; an eligible tuple outside derives `CONTRADICTS`; every ineligible run derives `INELIGIBLE_EXECUTION`.
- Classification accepts no requested class/tuple/reason. A run may publish successfully while classification fails; classification-only retry never reruns a process.
- A future classifier profile creates a distinct object/link without rewriting target or run.

### Human projection reserved for P08

- `CONFORMS` → **Matches**.
- `CONTRADICTS` → **Differs**.
- `INELIGIBLE_EXECUTION` → **Could not judge**.
- Refusal, contract invalidity, usage error, and internal failure stay outside behavioral results.
- The default jargon budget is contract, candidate, pinned commit, checked fields, result. Digests live in details/JSON.

### Wake boundary

Wake integration is an immutable-ref handoff only. Countershape may receive a display label, repository, immutable Git ref, and inert provenance, then independently pin and inspect the Git object. It never consumes Wake sessions/events/effects/gates, resumes/replays/forks agent runs, or supervises coding agents.

## Architecture map

- `internal/store/object_store.go` — generic canonical CAS; not sufficient as official C authority.
- `internal/emit/node/publication.go` — opaque residue publication/reopen precedent.
- `internal/gitobj/types.go`, `materialize.go`, `validate.go` — live Git capabilities and current comparison-only materialization seam.
- `internal/world/tools.go` — current generic tool admission to narrow into Node-specific authority.
- `internal/world/process*.go` — direct Darwin process mechanics to extract into import-restricted `internal/processmechanics`.
- `internal/contractexec/runner` — the only C package above store/runtime authority that consumes `RunPermit`; low-level process mechanics never imports store or sees the permit.
- `internal/domain/transitions.go` — existing process/control reason roster; standalone failures must not enter it.
- `internal/portablevalue/value.go` — exact tuple limits and canonical selected-field values.
- `internal/adapters/{cli,http}` — authoritative capture/projection components, never TAP parsing.
- `spec/schema/v1/contract-execution-target.schema.json` — current planning target shape, replaced in C1.
- `spec/schema/v1/finalized-contract-run.schema.json` — current false-green scope/reason shape, replaced in C1.
- `spec/schema/v1/contract-execution.schema.json` — current planning classification shape, narrowed in C1.
- `tools/validate-planning.mjs` — current fixture-graph checks, not runtime authority.
- `tools/verify-current.mjs` — cumulative roster; C0 repairs all direct and inherited Go package parallelism to `-p=1`, and every later unit enrolls its own gate before sealing.

## Locked v1 semantic tables

C1 implements these tables literally. Changing a field roster, precedence rule, or ceiling requires a new profile/version and a separately receipted planning correction.

| Body | Identity-bearing fields | Forbidden inputs |
| --- | --- | --- |
| `ContractExecutionTarget` | schema/kind/version; exact bundle and terminal-residue binding; pinned object format/commit/tree and recomputed tree identity; materialization-policy/manifest identity; fresh attempt artifact/nonce; measured boot-session identity; admitted absolute Node path, executable identity, owned-probe digest, measured `process.execPath`, version/major/platform/architecture | self digest, result, caller-authored source duplicate, didrun grade, standalone declaration, mutable status, raw argv, or proof Boolean |
| `FinalizedContractRun` | schema/kind/version; exact official target/attempt/StartClaim; one conclusive `SpawnObservation`; bounded `ClosedRunWitness`; process closure; scope closure; capture/projection sum; exact private-manifest summary and retention-at-finalization fact | self digest, classifier result/profile, current blob-availability status, raw evidence bytes, caller-selected reason/disposition, or proof Boolean |
| `ContractExecution` | schema/kind/version; literal `CONTRACT_EXECUTION_EXACT_TUPLE_V1`; exact official target and finalized-run digests; derived result only | tuple, process/scope reason, lifecycle/runtime/source duplicate, requested class, self digest, current/latest flag, or didrun grade |

`SpawnObservation` is exactly `START_ERROR(code)` or `CHILD_PID_OBSERVED(pid)`. Missing/unknown observation is an operationally incomplete consumed target and cannot enter an FCR. A conclusive start error may close an ineligible FCR; a PID observation requires complete owner closure.

Process closure is exactly:

- `PROCESS_CLEAN`, with no primary or cleanup control; or
- `PROCESS_CONTROLLED`, with optional primary control plus independent `teardown_error` and `orphan_risk` facts, requiring at least one of those three. The primary roster excludes `TEARDOWN_ERROR` and `ORPHAN_RISK`; cleanup ordering is fixed as teardown then orphan. Cleanup never overwrites the primary cause.

Capture/projection is exactly `NO_CAPTURE`, `CAPTURED_UNPROJECTED(typed_ref)`, or `PROJECTED(typed_ref, exact_tuple)`. Tuple presence is equivalent to `PROJECTED`, even when a later process-cleanup or scope failure makes the run ineligible.

Standalone scope is exactly:

- `COMPLETE`: all five named checks are present and clean;
- `PARTIAL`: no typed violation is present and the missing-check roster is nonempty; or
- `VIOLATED`: at least one typed violation is present, with every other check explicitly present-clean or missing.

`VIOLATED` outranks `PARTIAL`; missing checks never erase an observed violation. The five-domain order is target inventory, child bindings, import resolution, service bindings, named parent-secret sentinel inheritance.

The constructor-derived disposition table is closed:

| Process axis | Scope axis | Disposition | Tuple membership allowed |
| --- | --- | --- | --- |
| `PROCESS_CLEAN` | `COMPLETE` | `ELIGIBLE_CLEAN` only when observation is `PROJECTED` | yes |
| `PROCESS_CONTROLLED` | `COMPLETE` | `INELIGIBLE_CONTROL` | no |
| `PROCESS_CLEAN` | `PARTIAL` or `VIOLATED` | `INELIGIBLE_STANDALONE` | no |
| `PROCESS_CONTROLLED` | `PARTIAL` or `VIOLATED` | `INELIGIBLE_CONTROL_AND_STANDALONE` | no |

An otherwise clean run without `PROJECTED` is `PROCESS_CONTROLLED` under the existing exact capture/projection primary reason; it cannot manufacture `ELIGIBLE_CLEAN`. `CONFORMS`/`CONTRADICTS` are exact complete-tuple membership only for `ELIGIBLE_CLEAN`; all other dispositions derive `INELIGIBLE_EXECUTION` without copying their reasons into `ContractExecution`.

Locked ceilings: every canonical C body remains within the existing 1 MiB object parser ceiling; `ClosedRunWitness` is at most 256 KiB; it contains at most 16 typed refs and exactly five standalone domains; private evidence is at most 16 blobs and 64 MiB aggregate per finalized run. Existing per-stream CLI and HTTP capture ceilings remain authoritative. Canonical bodies contain summaries/digests only. Private evidence may contain secrets; confidentiality is not established.

## Execution order and claim ceilings

| Unit | Shippable result | Maximum honest claim |
| --- | --- | --- |
| C0 | deep-dive corrections, controlling docs, pack, handoff | planning/scope lock only |
| C1 | strict semantic model, schemas, examples, exhaustive algebra | inert canonical semantics only |
| C2 | nonhead storage mechanics, exact mappings, private evidence, interlock/claim substrate with test-only issuers | persistence substrate only; no official target or production permit |
| C3 | direct single-target Git + Node authority and target publication | exact pre-spawn target authority only |
| C4 | process mechanics plus contract runner, production interlock/permit, CLI-profile physical closure, FCR, classification | CLI-profile-only native physical slice |
| C5 | HTTP plus full five-domain standalone evidence | full P07B-C on exact exercised native tuple |
| C6 | C6a cumulative hostile/fault/parity/expert-surface closure, then C6b receipt reconciliation | exact sealed P07B-C claims only |

## What this pack does not cover

- Product CLI composition/status/quickstart (P08), studio/visual UX (P09/P10), export/package/release (P11), or final whole-system dogfood (P12).
- Wake sessions/events/effects/gates/resume/replay/fork or coding-agent supervision.
- Arbitrary executables, package installation, setup commands, browsers, databases, containers, external providers, hostile repositories, or other OS/runtime tuples.
- Host-wide absence, network/registry denial, containment, confidentiality, listener ownership, coordinated same-user replacement resistance, production readiness, adoption, or maintainership.

---

# C0 — lock the corrected P07B-C authority contract

**Risk:** medium-high. This changes future design authority and must not retroactively inflate sealed B claims.
**Files:** P07B-C research package; this prompt; controlling concept/architecture/semantics/state/threat/claim-vocabulary/pack/handoff/status docs; the machine-readable C0 authority declaration and C0–C6 path allowlist; both C0 checkers; the cumulative verifier and its self-test.

## Goal

Make the deep-dive corrections durable and self-contained before production C types exist. Mark the old three planning schemas as superseded for C1 rather than silently treating them as runtime-ready.

## Exact changes

- Record the six lanes, synthesis, different-model red team, follow-ups, and executive ruling.
- Update all controlling docs to the locked decisions above.
- Link this pack from `docs/PROMPT_PACK.md` and identify C1 as the next implementation unit.
- Add a metadata-only checker plus mutation self-test that requires the three-object split, private operational interlock, intent-only StartClaim, process-local RunPermit, separate SpawnObservation, COMPLETE/PARTIAL/VIOLATED scope, bounded private evidence, no semantic execution head, ordered single C0–C6 headings, substantive eleven-file research package, and Wake boundary.
- Add an exact machine-readable C0 authority declaration for the semantic/operational object rosters, issuer and package edges, boot-reset fence, five scope domains/states, P08 reservation, Wake boundary, and C0–C6 order. Mutants that add authority, move issuance earlier, weaken reset, remove scope, or relabel receipt state must fail.
- Add a machine-readable C0–C6 path allowlist plus a staged-scope checker/self-test. Every later prompt must update its own allowlist intentionally before an unexpected path can enter its receipt.
- Repair `tools/verify-current.mjs` and its self-test so `GOFLAGS` plus direct build/vet/test arguments enforce `-p=1`; enroll the C0 plan/scope self-tests in the current roster.
- Treat the C0 checker as an intentional phase gate: it pins legacy C fixtures only through C0. C1 must evolve that same enrolled gate before replacing the fixtures, retain every semantic/document invariant, and replace hash pins with strict runtime/schema/example intersection. Every C1–C5 unit enrolls its new checker/test runner and hostile self-test before its own seal.
- Update `docs/HANDOFF_MODE_C.md` with exact P07B-B sealed state plus the provisional C0 boundary.
- Add a provisional `docs/status/P07B-C-C0-AUTHORITY.md`. The enrolled checker must accept exactly two states: C0a requires the complete eight-row `UNRECEIPTED` map with no receipt declaration; after C0a seals, C0b adds `spec/verification/p07b-c-c0-receipt.json` with the exact source commit/tree, strict counts, ordered claim labels, and verbatim grades. The checker reopens that real ancestor commit/tree and its didrun Git note through admitted Git, requires exact eight-event/claim coverage, and requires status plus a delimited handoff receipt table to agree. Wrong commit, tree, note grade, count, status, or handoff must fail. Close C0b as a separate receipt-document commit.

## MUST NOT change

- No P07B-C runtime types, store layout, schemas/examples, Git/process behavior, or public CLI.
- No change to any sealed P07B-B receipt/grade.
- No claim that the deep dive executed or validated production behavior.

## Verification

1. Metadata-only C0 plan checker passes.
2. Existing planning validator still passes and the old C schemas are explicitly identified as superseded planning fixtures pending C1.
3. C0 plan-checker and staged-scope self-tests reject their bounded mutation matrices.
4. The repaired serial-Go current cumulative verifier passes with no C runtime claim.
5. Machine-readable staged paths and scoped credential-prefix scan pass.

## Gate

After C0a and C0b each seal/strict-clean, machine-cleared for document consistency only. Every P07B-C runtime capability remains `UNRECEIPTED`.

## Commit

Source: `docs: lock P07B-C execution authority`
Receipt reconciliation: `docs: receipt P07B-C C0 authority lock`

---

# C1 — implement strict inert semantic algebra

**Risk:** high. Canonical bytes become the identity contract for later physical objects.
**Files:** `internal/contractexec/model/**`, the three C schemas/examples, planning generator/validator, architecture checker, semantic docs/tests.

## Goal

Implement strict codecs and constructors for the corrected target, finalized-run, closed-run witness, standalone scope, and classification bodies without filesystem or process authority.

## Exact changes

- Closed, versioned schemas with no self digests, redundant source DTO, duplicated reason, raw evidence, or constant-false proof fields.
- Separate process closure and standalone scope axes; the process sum preserves primary-plus-cleanup combinations and the constructor derives the final disposition.
- Typed refs, exact rosters, bounds, canonical ordering, and defensive copies.
- Literal classifier profile and exact truth table.
- Parsed values remain inert and cannot reach publication or spawn.
- Evolve the enrolled C0 plan gate before changing legacy fixture hashes: retain document/authority checks, replace C0-only pins with exact schema/runtime/example intersection, add the C1 architecture checker plus hostile self-test, and enroll both in `verify-current` before sealing.

## MUST NOT change

- No store writes, Git calls, runtime probes, process spawn, TAP parsing, head mutation, public CLI, or generic constructor from a body/digest bag.

## Tests

- Exhaustive cross-product of primary process cause, teardown/orphan controls, observation, scope, and classification, including simultaneous primary-plus-cleanup cases.
- Canonical goldens/runtime-schema intersection; invalid kind/profile/roster/order/duplicate/bound vectors.
- Properties and bounded fuzz for round-trip, typed-kind separation, tuple ordering, and mutation resistance.
- Architecture negative gate: C1 cannot import store/Git/world/process/CLI.

## Gate

Machine-cleared inert semantic algebra only.

## Commit

`feat: define P07B-C execution semantics`

---

# C2 — add nonhead persistence and private interlock substrate

**Risk:** high. Winner/loser and crash semantics are authority-bearing.
**Files:** the exact `internal/store/execution_interlock.go`, `nonhead_contract.go`, and `private_contract_run.go` families admitted by the unit allowlist; private manifests/attachments; store tests; architecture gates/self-tests; cumulative roster; status docs. Any additional file must first be named explicitly in the staged allowlist.

## Goal

Add the low-level storage mechanics for typed official-publication records, exact-key relationships, the private operational interlock, start-intent records, private attempt attachments, and durable terminal manifests without any production issuer that can make a C1 body official or mint a RunPermit.

## Exact changes

- Namespace-specific witness/mapping bytes and exact-existing convergence; generic CAS bytes alone remain unofficial.
- Closed create-once mappings for attempt→target, target→run, and `(run,classifier-profile)`→execution, each resolvable only with its exact typed parent key.
- Store-private `ExecutionInterlock` durability/CAS/reconcile/reset mechanics and target-keyed StartClaim persistence. C2 exports no production semantic issuer and returns no production RunPermit.
- Any C2 issuer used to exercise mechanics exists only in `_test.go`; production official-target publication arrives in C3, and production interlock/StartClaim acquisition plus permit minting arrives in C4.
- StartClaim remains deterministic intent/tombstone; the interlock has only `CLEAR(boot-session)` or `HELD(target,boot-session)`, with no PID, liveness, result, latest pointer, or process-resume authority.
- Bounded private evidence manifest with retention-at-finalization semantics and explicit purge nonmutation rule. Canonical FCR summaries are sufficient semantic authority; private-blob availability is a separately reported operational state (`RETAINED | PURGED | MISSING_UNEXPECTED`), never rewritten into the FCR.
- Exact existing convergence, mismatch refusal, no-effect/unknown-effect ambiguity, restart reopen, store-instance binding, safe no-claim reconciliation, held-on-ambiguity behavior, and explicit changed-boot-session reset.

## MUST NOT change

- No official target/run/execution issuer, production RunPermit, process spawn, Git materialization, runtime admission, result list/latest/traversal/status API, study head, or semantic conclusion from the interlock.

## Tests

- Multi-process interlock/claim races; exactly one storage winner, with no production permit.
- Parsed/copied target bodies cannot reach the C2 mechanics through production APIs; all fixture issuers are `_test.go`-only and an architecture gate kills any non-test issuer.
- Fault matrix at create/sync/link/reopen; corruption, cross-store, wrong-kind, and alternate-link refusal.
- Interlock acquisition-before-claim, safe release only from a terminal-closure test capability, same-boot ambiguity refusal, and changed-boot-session reset matrix.
- Private manifest count/size/roster/retention/purge boundaries; missing evidence before FCR publication refuses publication, while postpublication purge or unexpected loss changes only honestly reported blob availability and never historical classification.

## Gate

Machine-cleared persistence/interlock substrate only. Official semantic publication, production permit issuance, and spawn remain absent.

## Commit

`feat: add contract execution persistence substrate`

---

# C3 — publish one exact pre-spawn target

**Risk:** high. This joins live Git, private materialization, runtime, attempt, and store authority.
**Files:** `internal/gitobj/single_target.go`, `internal/noderuntime/**`, `internal/hostepoch/**`, `internal/contractexec/target*.go`, production target-publication owner, tests/checkers/self-tests, cumulative roster, docs/status.

## Goal

Publish/reopen a `ContractExecutionTarget` from one exact current residue and live physical authorities before any subject spawn.

## Exact changes

- Materialize/revalidate directly from opaque `InspectedTree`; share only private mechanics with comparison materialization.
- Admit one explicit absolute Node path with owned probe and exact identity summaries.
- Admit the exact Darwin boot-session identity used by later interlock acquisition/reset.
- Allocate/sync a fresh CONFORMANCE attempt attachment.
- Construct/publish/reopen the target plus its typed witness and attempt→target relationship only through the live C3 issuer; retain exact private revalidation closure and return an opaque official-target capability.
- Enroll the C3 target gate and hostile self-test in the cumulative verifier before sealing.
- Assert study head inode and bytes remain unchanged.

## MUST NOT change

- No fake one-member `WorldPlan`, `BoundCandidate`, or `WorldInstance`; no PATH lookup, dirty worktree, arbitrary executable, candidate/subject spawn, run, classification, TAP, interlock acquisition, StartClaim, RunPermit, or head advance. One bounded owner-controlled Node runtime-probe subprocess is required and is not a subject run.

## Tests

- Moving-ref pin, copied OID/body refusal, dirty-worktree exclusion, unsupported tree forms, target mutation, runtime symlink/byte/probe mismatch, closed capability, restart exact-digest reopen, and pre-spawn fault matrix.

## Gate

Machine-cleared pre-spawn target authority only. This is the minimum honest cut if later physical evidence cannot be proven.

## Commit

`feat: publish exact contract execution targets`

---

# C4 — close a CLI candidate-profile run and publish its classification

**Risk:** high. First subject execution and first finalized/classified evidence.
**Files:** `internal/processmechanics/**`, `internal/contractexec/runner/**`, CLI candidate-profile service, production admission/run/classification publication, private capture manifest, tests/checkers/self-tests, cumulative roster, docs/status.

## Goal

Extract the low-level Darwin mechanics and make the higher contract runner the sole consumer of one authentic target-bound RunPermit. Perform a Go-owned CLI candidate-profile run, close physical evidence, durably release or preserve the interlock as appropriate, publish `FinalizedContractRun`, then publish `ContractExecution`.

## Exact changes

- Exact import graph: `processmechanics` imports no store/contract/domain package; `contractexec/runner` imports official-target/admission/runtime plus `processmechanics`; only existing `world` and the contract runner may call raw mechanics; no low-level mechanics API accepts a permit, target digest, semantic body, or public raw argv.
- The runner reopens the official target, durably acquires/reopens `ExecutionInterlock`, creates/reopens StartClaim, and only then receives the single-consumption process-local RunPermit. It consumes the permit itself, derives the private mechanics request, and revalidates target/runtime immediately before subject spawn.
- Persist a separate SpawnObservation. Missing/unknown observation or incomplete owner closure cannot publish an FCR or classification; the target remains consumed and the interlock stays held for that boot session.
- Close capture, process, wait, drains, teardown, final group probe, materialization/runtime revalidation, and all five standalone domains for the CLI profile.
- Persist bounded summaries/typed refs; raw evidence remains private. Make the terminal witness durable before releasing the interlock. Only a conclusive terminal-closure capability can release it.
- Publish/reopen FCR, derive/publish/reopen classifier-profile-bound execution.
- Classification-only retry performs no spawn.
- Revalidation proves checkpoint-bound identity plus child self-report only. Preserve the revalidation-to-`execve` same-user substitution residual; no descriptor-bound or coordinated-replacement claim is added.
- Enroll runner/architecture gates and hostile self-tests in the cumulative verifier before sealing.

## MUST NOT change

- No TAP-derived authority, raw argv public edge, process resume, same-target retry, same-boot reset after ambiguity, HTTP, product CLI, full two-profile P07B-C claim, result list/status, or head mutation.

## Tests

- Conforms/contradicts/ineligible; selected/unselected field behavior; every primary-plus-cleanup combination; all five CLI scope detectors and their forbidden-positive controls; partial/violated scope; poisoned TAP independence; target/run/profile mismatch; interlock-before-claim ordering; missing observation; crash at every admission/terminal/release boundary; same-boot retry refusal; changed-boot explicit reset; classification-only retry.

## Gate

Machine-cleared CLI candidate-profile-only slice on the exact exercised native runtime. This is not the P08 product CLI. HTTP and full P07B-C remain `UNRECEIPTED`.

## Commit

`feat: finalize CLI contract executions`

---

# C5 — add HTTP and complete standalone-scope evidence

**Risk:** high. Adds readiness/service/import evidence and completes the narrow standalone claim.
**Files:** HTTP contract execution service, five scope detectors, fixtures/tests/checkers/docs.

## Goal

Add the locked child-bind/raw-HTTP profile, complete all five HTTP scope domains, and close the combined CLI/HTTP scope matrix without redefining the C4 CLI evidence.

## Exact changes

- Reuse adapter-owned raw HTTP parsing/projection; no high-level normalization in generic layers.
- Measure the exact loopback child-reported service binding without claiming listener ownership.
- Complete target inventory, child binding, import resolution, service binding, and sentinel inheritance evidence for HTTP; reopen and regression-check the already complete C4 CLI witness.
- Publish correct PARTIAL/VIOLATED combinations and derive ineligible classification.
- Enroll the HTTP/scope gates and hostile self-tests in the cumulative verifier before sealing.

## MUST NOT change

- No inherited-listener substitution, generic HTTP/CLI coercion, network denial, host-wide absence, containment, registry claim, arbitrary service discovery, or new runtime/platform claim.

## Tests

- Conforming/differing/custom-401/allow-many; decoy readiness, wrong port, partial/malformed capture, timeout, teardown/orphan; each detector’s forbidden-positive control; cross-target/run evidence; CLI regression and Go/Node parity.

## Gate

Machine-cleared full P07B-C only for the exact exercised CLI/HTTP native tuple.

## Commit

`feat: complete standalone contract execution`

---

# C6 — cumulative closure, expert surfaces, and receipts

**Risk:** high. Final P07B-C evidence boundary and cumulative-regression risk.
**Files:** cumulative verifier/selftest, architecture hostile gate, docs/status/handoff, deterministic developer-fixture diagnostics/JSON/docs captures under the admitted `docs/captures/p07b-c/` and `tools/p07b-c/` directories, evidence report.

## Goal

Use C6a to freeze and prove all C layers plus inherited boundaries and run a genuine multi-pass expert diagnostic/JSON/docs review. Use C6b only after C6a seals to reconcile exact source receipts and handoff state without self-reference.

## Exact changes

- Replace B’s “C absent” check only with exact C ownership/import/API topology; preserve every other B gate.
- Add hostile metadata/selftests and full cumulative ordered roster.
- Run at least three build/see/exercise/critique/rebuild passes on developer-fixture diagnostics, strict JSON evidence, and C operator documentation at 60/80/120 columns. Do not use the P08 Matches/Differs/Could-not-judge headlines or define a product CLI ABI.
- Run a blind different-model critic on sanitized expert captures; disposition every Medium+ finding. Deterministic capture commands may be receipted; critic judgment remains explicitly `UNRECEIPTED` and has no semantic authority.
- C6a records implementation-owned native facts and then commits/seals/strict-verifies the frozen source tree.
- C6b records C6a’s exact commit/tree, timings, permanent negatives, didrun bugs, claims/grades, nonclaims, current private-evidence availability, and handoff; it receives its own fresh receipt-doc ledger, commit, seal, note check, and strict gate.
- Generate the exact C6b strict HTML and SHA-256 outside Git after C6b strict succeeds. Preserve an archival byte-identical copy of the final ledger, but retain the matching `.didrun` live; if it is ever rotated, restore it before strict/HTML reproduction. HTML is not a ledger substitute.

## MUST NOT change

- No grade inflation, generalized platform/runtime/security claim, deleted negative history, semantic execution index/status selector, P08 implementation/headlines/ABI, external provider call without human authorization, hand-edited HTML, or removal of the final matching live ledger.

## Verification

- Full repo twice under stock macOS TMPDIR; Go `-p=1`; race/vet/count-fuzz; architecture/checker selftests; exact physical CLI/HTTP matrix; direct generated Node parity; unchanged head; staged scope/credential-prefix checks.
- Each final command receives an immediate narrow didrun claim.
- C6a source and C6b receipt commits each seal, their Git notes validate, and strict exits `0`; exact HTML is generated for C6b only after that exact commit exists.

## Gate

Machine-cleared exact P07B-C capabilities only. Human comprehension, taste, adoption, security review, production hardening, cross-platform support, and maintainership remain human work.

## Commit

C6a: `test: close P07B-C cumulative evidence`
C6b: `docs: receipt P07B-C contract execution`

## Combined falsification matrix

The pack is not complete unless the final gate rejects all of these classes:

1. Parsed/copied target reaches a production publication/admission issuer or spawn.
2. Generic CAS, C2 test fixture, losing interlock/StartClaim caller, or existing record mints live authority.
3. One target crosses the spawn-attempt boundary twice, including after crash or lease fiction.
4. Missing SpawnObservation becomes “not started,” publishes an FCR, or releases the interlock.
5. A fresh target starts on the same boot while an ambiguous prior target may survive.
6. Dirty or moving-ref bytes execute instead of the pin.
7. Runtime changes at a controlled checkpoint evade refusal, or the revalidation-to-`execve` residual is omitted/overclaimed.
8. TAP changes Go classification.
9. Partial/violated scope plus projected tuple becomes eligible.
10. Standalone failure enters process `ControlReason` or cleanup overwrites the primary cause.
11. Raw/oversized/private evidence enters canonical bytes or default export.
12. A prepublication required blob is missing yet FCR publication succeeds, or a postpublication blob is purged/missing but details still report it as retained/available; immutable FCR/classification bytes themselves must not change.
13. Run A classifies against target B or a foreign classifier profile.
14. Classifier retry spawns a process or historic conclusion changes under the same profile.
15. Any C object advances/rewrites the terminal study head, or the private interlock becomes semantic result/discovery authority.
16. A result traversal/latest/status registry silently appears.
17. A Wake session/event/gate or didrun grade becomes Countershape semantic authority.
