# P07B-C — exact target, finalized run, and immutable classification

**Created:** 2026-07-18
**Companion evidence:** `research/deep-dive/p07b-c/00-scope-and-method.md` through `10-executive-briefing.md`
**Platform scope:** Darwin/arm64/cgo, the exact exercised Node runtime tuple, logical `node`, one exact clean repository-relative JavaScript entrypoint, CLI plus locked child-bind HTTP
**Validated against:** commit `464e47adbf7f4497dfafa89938a9539239ffd41b` on `codex/countershape-autopilot`

This pack turns the planning-only P07B-C shapes into a capability-only, crash-honest execution spine: three immutable nonhead semantic objects plus one store-private operational interlock that can block spawn but can never author or select a result. Its source units, semantic prerequisites, and receipt descendants remain separate because target publication, cooperative spawn admission, physical run closure, standalone evidence, classification, and evidence reconciliation are different truths.

## How to use this pack

Execute the core source/receipt sequence `C0 → C1 → C1B → C2 → C2B → C3P → C3PB → C3 → C3B → C4 → C5 → C6`, inserting every separately declared maintenance boundary at its machine-ordered unit edge. The declared prerequisite edge is `C3B → C3D → C4V → C4`; C4 may begin only after C4V seals and verifies strictly. This is a permanent ordering contract, not the live phase cursor: the delimited capsule in `docs/HANDOFF_MODE_C.md` is the sole live phase/receipt cursor. Every arrow is a fresh didrun ledger or preserved unit-local ledger, exact staging review, commit, seal, Git-note inspection, and `NO_COLOR=1 didrun verify --strict` loop. When exact source-commit identity or grades must enter repository documentation, use a second receipt-document sub-boundary (`Ca` source, then `Cb` reconciliation) rather than creating a self-referential commit. Do not begin the next implementation unit until its last required sub-boundary exits strict with `0`.

The user has explicitly authorized commits at verified shippable boundaries. That authorization does not allow pushes, releases, external provider calls, or weakening a failed receipt.

## Rules every unit inherits

1. Read `docs/CONCEPT_BRIEF.md`, `docs/ARCHITECTURE.md`, `docs/SEMANTICS.md`, `docs/STATE_MACHINES.md`, `docs/THREAT_MODEL.md`, `docs/PROMPT_PACK.md`, `docs/HANDOFF_MODE_C.md`, this pack, and the P07B-C deep-dive package before editing.
2. Verify every file/line reference against current code. Current source wins over remembered locations and API facts; if it disagrees with a locked semantic ruling, stop and amend the ruling explicitly rather than silently letting stale code broaden authority.
3. The root is the sole repository writer and sole didrun/Git/seal operator. Parallel agents are read-only critics.
4. Every load-bearing command runs through `/opt/homebrew/bin/didrun run -- ...`; immediately claim each successful final event before any next event.
5. Through C4V, inherit sealed C1V exactly: `GOMAXPROCS=2`; direct build/vet and the exact general-package complement use `-p=2`; the declared sensitive packages (including `internal/store`) and every inherited/nested Go invocation use `-p=1`; fuzz uses `-parallel=1` with a count budget. C4 must preserve that stability envelope while relocating the three output/capture repetition cases to `internal/processmechanics` and classifying `internal/processmechanics`, `internal/contractexec/runner`, and `testkit/contractexec/cli` as sensitive. C4 freezes exact 54-case and dormant 56-case catalogs at `sha256:3be703fae10155c83b19be54ae7cd79cfc5b1bbaddfcf86bb2c9781ad93f853a` and `sha256:08db7c338eb3ba900cf6df2545c04f27bed16889c81dd16e5fb18197c4d52958`; each C4 case runs as its own `--case <id>` didrun receipt, and C5 later does the same for all 56 parent-sealed cases, including `testkit/contractexec/http`. No shared matrix receipt substitutes for per-case isolation.
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

- Generic CAS object existence proves canonical bytes only. A store-private typed publication witness and exact create-once relationship are necessary inert storage facts; C3/C4 alone rejoin them with live prerequisites and issue official target/run/execution authority.
- The closed mappings are `attempt digest -> target digest`, `target digest -> finalized-run digest`, and `(finalized-run digest, classifier-profile digest) -> execution digest`. Resolution requires the already-held typed parent key; no API accepts a search predicate or returns siblings/newest values.
- Semantic reopen is exact-digest/typed-parent-only. P07B-C adds no result listing, filesystem traversal, `latest`, mutable result/status record, or semantic execution head.
- Publication returns exact digests. Lost output is not solved inside C; a future immutable collection is a separate design.

### Cooperative spawn admission

- `ExecutionInterlock` is one store-private, store-wide operational CAS. It is either receipt-backed `CLEAR(boot-session,generation,revision)` or `HELD(target,attempt,boot-session,generation)`. It serializes candidate spawn only; it is neither canonical evidence nor a source of run/classification facts.
- A production acquisition requires an opaque C3-issued official target capability. C2 may implement/test storage mechanics but has no production issuer and cannot make a parsed C1 body official.
- A target-keyed `StartClaim` is durable intent and a permanent tombstone, never proof a child started.
- The C4 contract runner first durably acquires/reopens the interlock for the authentic target, then creates/reopens StartClaim. Only that winner receives one opaque process-local `RunPermit`. Generic publish/read, a losing publisher, copied owner data, a test fixture outside `_test.go`, or restart never recreates it.
- `SpawnObservation` separately records `Start` error or observed child PID. Missing observation is unknown.
- The honest guarantee is serialized cooperative at-most-once spawn admission per exact target, not exactly-once execution.
- Conclusive owner-produced terminal closure releases the interlock only after the private manifest, exact FCR, and target-to-run relationship are durable and reopened. An ambiguous operation returns no winner or permit and cannot itself reopen admission. Before `CLEAR`, ambiguity after StartClaim consumes the target and leaves the interlock held; after `CLEAR`, a missing or invalid receipt remains blocked without rolling back the FCR. Proven-no-claim, known-no-effect preclaim reconciliation does not consume the target.
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
- `internal/contractexec/model/` — C1's sole inert semantic authority for exact target/run/execution bodies; no store, Git, runtime, process, publication, or spawn edge.
- `spec/schema/v1/contract-execution-target.schema.json` — C1 closed target syntax projection; Go owns cross-field meaning.
- `spec/schema/v1/finalized-contract-run.schema.json` — C1 closed finalized-run syntax projection; Go owns process/scope/evidence correlations.
- `spec/schema/v1/contract-execution.schema.json` — C1 closed derived-classification syntax projection.
- `tools/validate-planning.mjs` — schema/example graph compatibility checks, not semantic or runtime authority.
- `tools/verify-current.mjs` — cumulative roster; C0's global serialization repair and C1V's qualified direct-general `p=2` receipt remain history, while C4V's recurrence-triggered direct-general `p=1` profile is current and every later unit enrolls its own gate before sealing.

### C1 wire-resolution rulings

- The exact JSON spellings in the three C schemas and checked examples are frozen by C1. The Go constructors/parsers remain semantic authority; the schemas intentionally overapproximate cross-field and recursive rules, so schema-only acceptance grants no capability.
- `classifier_profile` is the literal `CONTRACT_EXECUTION_EXACT_TUPLE_V1`. `ClassifierProfileDigest()` derives the future C2 link key from the canonical schema/kind/literal-profile declaration. That digest is not serialized in `ContractExecution`, accepts no caller input, and commits no implementation bytes.
- The target's admitted absolute path, executable-byte digest, mode/count, measured `process.execPath`, version/major/platform/architecture, and probe facts establish checkpointed path/content identity only. They do not prove a later spawn consumed the same inode or descriptor and do not close coordinated replacement. C3/C4 own live revalidation and retain the residual risk.
- C1 validates the closed start-error codes and typed witness structure only. C4 owns whether referenced evidence has the asserted physical content and was recorded in the required chronology.

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
| C1B | narrow C1 source-receipt reconciliation | sealed C1 receipt only; no source behavior or target authority |
| C2 | nonhead storage mechanics, exact mappings, private evidence, interlock/claim substrate with test-only issuers; then a separate narrow C2B source-receipt reconciliation | persistence substrate only; no official target or production permit |
| C3P | pre-authority boot-session semantic correction, exact C3 scope, and receipt machinery; then separate C3PB source-receipt reconciliation | inert corrected profile only; no live measurement or target authority |
| C3 | direct single-target Git + Node/boot authority and target publication; then separate C3B source-receipt reconciliation | exact pre-spawn target authority only |
| C4V | executable C4/C5 scope, predecessor-note, qualification, and final-runbook contract repair | planning/checker authority only; no process execution |
| C4 | process mechanics plus contract runner, production interlock/permit, CLI-profile physical closure, FCR, classification | CLI-profile-only native physical slice |
| C5 | HTTP plus full five-domain standalone evidence | full P07B-C on exact exercised native tuple |
| C6 | C6a cumulative hostile/fault/parity/expert-surface closure, then C6b receipt reconciliation | exact sealed P07B-C claims only |

Every source boundary that must hand durable grades to a later unit is followed by its separately sealed receipt-reconciliation subunit. C1B, C2B, C3PB, and C3B may bind only their already-sealed source note, strict result, exact claim map, HTML snapshot, and ignored-ledger manifest; none may edit source/checker behavior or grade itself. The permanent prerequisite edge is `C3B → C3D → C4V → C4`; this paragraph does not declare the live phase. The delimited capsule in `docs/HANDOFF_MODE_C.md` remains the sole live phase/receipt cursor.

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
**Files:** `internal/contractexec/model/**`, the three C schemas/examples, planning generator/validator, the new C1 architecture checker, the inherited P07B-B architecture checker/self-test compatibility repair, semantic docs/tests.

## Goal

Implement strict codecs and constructors for the corrected target, finalized-run, closed-run witness, standalone scope, and classification bodies without filesystem or process authority.

## Exact changes

- Closed, versioned schemas with no self digests, redundant source DTO, duplicated reason, raw evidence, or constant-false proof fields.
- Separate process closure and standalone scope axes; the process sum preserves primary-plus-cleanup combinations and the constructor derives the final disposition.
- Typed refs, exact rosters, bounds, canonical ordering, and defensive copies.
- Literal classifier profile and exact truth table.
- Parsed values remain inert and cannot reach publication or spawn.
- The classifier-profile digest is a derived future relationship key, not an execution-body field or implementation digest.
- Checked examples must parse and rebuild byte-for-byte through the Go authority; schema-valid/runtime-invalid overapproximations must be explicit refusal tests.
- All three start-error codes receive exact wire round-trip coverage, while physical evidence content/chronology remains a C4 nonclaim.
- Runtime facts are documented as checkpointed path/content identity rather than inode- or descriptor-bound execution proof.
- Evolve the enrolled C0 plan gate before changing legacy fixture hashes: retain document/authority checks, replace C0-only pins with exact schema/runtime/example intersection, add the C1 architecture checker plus hostile self-test, and enroll both in `verify-current` before sealing.
- Evolve—not retire—the current P07B-B architecture gate. It must preserve every sealed B digest/import/API/publication/materialization invariant, admit C symbols only under the exact inert `internal/contractexec/model/` prefix, require the complete C1 architecture gate to pass, and continue rejecting every C surface elsewhere. Extend its in-memory defensive self-test for missing C1 closure, exact-prefix versus lookalike-prefix partitioning, and foreign C-surface injection. Composition is one-way `B -> C1`; C1 must never invoke B.
- Enumerate all nine production and five test files as exact C1 paths with no directory prefix allowance. Pin the exact test-import and top-level test/fuzz symbol rosters, and reject extra, missing, or renamed test topology.

## MUST NOT change

- No store writes, Git calls, runtime probes, process spawn, TAP parsing, head mutation, public CLI, or generic constructor from a body/digest bag.

## Tests

- Exhaustive cross-product of primary process cause, teardown/orphan controls, observation, scope, and classification, including simultaneous primary-plus-cleanup cases.
- Canonical goldens/runtime-schema intersection; invalid kind/profile/roster/order/duplicate/bound vectors.
- Properties and bounded fuzz for round-trip, typed-kind separation, tuple ordering, and mutation resistance.
- Architecture negative gate: C1 cannot import store/Git/world/process/CLI.
- Inherited-boundary compatibility: the full B gate and its self-test remain current, all B invariants still pass, and only the separately validated inert C1 package is removed from the former phase-specific “premature C” prohibition.
- Final targeted Go receipts consume `go test -json` through the C1 assertion profiles so a missing `-run` match or fuzz target fails instead of receiving a claim; fuzz uses `-parallel=1` and a fixed count budget.

## Gate

Machine-cleared inert semantic algebra only.

## Commit

`feat: define P07B-C execution semantics`

---

# C2 — add nonhead persistence and private interlock substrate

**Risk:** high. Winner/loser and crash semantics are authority-bearing.
**Files:** the exact `internal/store/execution_interlock.go`, `nonhead_contract.go`, `object_store.go`, and `private_contract_run.go` families admitted by the unit allowlist; the controlling prompt/docs; architecture gates/self-tests; cumulative roster; status docs. Any additional file must first be named explicitly in the staged allowlist.

## Goal

Add only the Darwin private persistence substrate for exact nonhead relationships, bounded private evidence, and cooperative spawn exclusion. C2 may persist and reopen inert store records, but it cannot return an official target, admit a subject run, mint a `RunPermit`, spawn a process, or make a semantic conclusion.

## Exact changes

- Define versioned, namespace-specific private link records. Each canonically repeats its relation, typed parent key, typed child key, and relationship-specific joins: `CONFORMANCE_ATTEMPT -> ContractExecutionTarget`; `ContractExecutionTarget -> FinalizedContractRun`; and `(FinalizedContractRun, derived classifier-profile digest) -> ContractExecution`. The target's attempt, the run's target/attempt/StartClaim, and the execution's target/run/profile must join exactly. Link paths come from a domain-separated canonical typed-key preimage, never concatenated caller strings or untyped digest path fragments.
- C2 returns only store-instance-bound inert records. None is an official-target capability or permit. C3 alone may wrap a reopened target record plus live Git/runtime/boot prerequisites into `OfficialTarget`; C4 alone may wrap a fresh successful admission into a process-local `RunPermit`. Files deliberately reopened by those descendants are named in their unit allowlists now.
- Generic CAS object existence remains insufficient. C2 may persist model objects only as mechanical storage; no C2 API accepting a body, digest, or generic object may return authority consumable by admission or spawn. Test fixture issuers exist only in `_test.go`.
- Persist one store-wide `ExecutionInterlock` under a Darwin cooperative cross-process lock. Its durable state is only `CLEAR(boot-session)` or `HELD(target,attempt,boot-session,generation)`. The generation is an opaque compare fence, not PID, liveness, result, lease expiry, discovery, or resume authority. The caller boot must equal the target's serialized boot binding; a historical `CLEAR` boot is transition history, not C3's live-boot authority.
- Acquire the interlock before StartClaim. StartClaim is one deterministic target/attempt/boot-bound durable tombstone. Only a newly durable claim created beneath one fresh lease may later feed C4 permit minting. Existing claim, reopened held state, copied generation, restart, losing racer, or ambiguous create result grants no winner authority.
- Classify persistence mutations as `KNOWN_NO_EFFECT`, `EXACT_CONVERGED`, or `AMBIGUOUS`. Only an exact held generation with proven no claim and a known-no-effect preclaim result may reconcile to clear. Before `CLEAR`, ambiguity after StartClaim remains held; after `CLEAR`, a missing or invalid receipt remains blocked. The first-ever absent interlock is the only receiptless bootstrap. Every persisted `CLEAR` requires an exact durable clear receipt with one closed cause (`KNOWN_NO_CLAIM`, `FINALIZED_RUN`, or `CHANGED_BOOT`) that reconstructs the previous HELD state and exact HELD-to-CLEAR transition. Receiptless or invalid `CLEAR` blocks all later acquisition but cannot roll back an already durable FCR.
- Treat every exact-visible create-once record as potentially unsynced until its parent directory is synchronized and the exact bytes reopen. Apply that convergence rule to typed relationships, StartClaims, private manifests, clear receipts, and purge intents before returning durable records or deleting a pack.
- Define a private manifest bound to the exact target, attempt, and StartClaim. It maps every logical witness reference to checked private blob bytes/ranges and a distinct private-byte digest, enforces the closed logical roster, and enforces the 16-blob/64-MiB ceilings. A finalized run may be stored only after its manifest and every referenced private item reopen exactly; C4, not C2, later owns the claim that those bytes substantiate a semantic evidence reference.
- `retention_at_finalization` means the complete manifest was retained and reopened at finalization time; it is not a current-availability claim. Availability is separately `RETAINED | PURGED | MISSING_UNEXPECTED`. `PURGED` requires a durable no-serve tombstone before best-effort cleanup and means neither secure erasure nor confidentiality. Purge/loss never rewrites target, run, relationship, or classification bytes.
- Cross-store protection is live capability/store-instance binding plus exact reopen in the original root. It does not detect or prevent a complete offline filesystem clone. The classifier profile digest identifies the declaration, not classifier implementation bytes.

## MUST NOT change

- No C2 production issuer of `OfficialTarget`, `FinalizedContractRun`, `ContractExecution`, or `RunPermit`; no path from body/digest/generic CAS bytes to admission or spawn authority.
- No process spawn, Git materialization, runtime/boot measurement, semantic result derivation, list/latest/traversal/status API, study-head mutation, lease expiry, PID takeover, automatic same-boot reset, clone-resistance claim, classifier-implementation provenance claim, or confidentiality/secure-erasure claim.

## Tests

- Multi-process interlock races prove one fresh lease and one StartClaim creator; no C2 production value is spawn-capable.
- Typed-key mutations reject wrong relation, parent/child kind, parent digest, target/attempt join, StartClaim join, classifier-profile key, alternate link, generic object, copied body, and raw-digest lookup.
- Fault-inject create/link/rename/sync/reopen boundaries. Only exact convergence may continue; an ambiguous operation cannot mint a permit or reopen admission, pre-`CLEAR` ambiguity after StartClaim remains held, and post-`CLEAR` receipt ambiguity remains blocked.
- Reconciliation proves that only a held generation with a proven no-claim, known-no-effect preclaim path can clear. Same-boot reset always refuses; changed-boot reset remains fenced test mechanics pending C3's live witness and C4's operator action.
- Clear-receipt tests cover missing, corrupt, wrong-cause, wrong-previous, every temporary/link/directory-sync/reopen fault, restart admission, and receiptless fail-closed behavior. A receipt is only the trusted writer's operational admission marker; it is not independent proof of terminal causation or child absence.
- After-link interruption tests directly restart and reopen exact relationships, StartClaims, private manifests, clear receipts, and purge intents; no merely visible file is promoted without durability convergence.
- Cross-store live-capability misuse refuses. A copied on-disk store is labeled as an explicit nonclaim, not a passed anti-clone control.
- Private-manifest tests cover missing, extra, wrong-role, wrong-range, wrong-hash, oversized, cross-run, and postpublication-loss cases. Tombstone-first purge reports `PURGED` without serving blobs; unexpected loss reports `MISSING_UNEXPECTED`; finalized-run and classification bytes remain unchanged.

## Gate

Machine-cleared persistence substrate only. It proves exact typed storage, fixture-driven interlock/StartClaim and receipt-backed-CLEAR mechanics, and bounded private-manifest retention at inert finalized-run persistence. It proves no official target, production permit or spawn admission, process spawn, physical run/finalization chronology, child absence, clone resistance, confidentiality, secure deletion, classifier implementation provenance, or semantic classification.

## Commit

`feat: add contract execution persistence substrate`

---

# C3P — correct the runtime epoch before live authority

**Risk:** high semantic prerequisite. The former `DARWIN_KERN_BOOTTIME_V1` label treated a calendar-adjustable value as one stable boot session.

**Exact source scope:** 25 mode-`100644` paths, no prefixes, sorted-newline digest `sha256:8b047704df047cb96f1c5d19418c4cfe8502e4b4746bb4f24c0851dac8d018c0`. The machine roster in `spec/verification/p07b-c-unit-paths.json` is authoritative. The additional verifier pair enrolls the dedicated C3P/C3PB receipt checker in cumulative verification instead of leaving an orphan gate.

## Goal

Freeze the future target profile as `DARWIN_KERN_BOOTSESSIONUUID_V1` before any production `OfficialTarget` exists. C3P changes inert model/schema/example/planning bytes and controlling design only. It performs no live measurement and issues no boot, runtime, Git, attempt, store, target, admission, or process authority.

## Exact semantics

- The private C3 host-epoch owner calls a bounded fixed-name Darwin system-call edge for read-only `kern.bootsessionuuid`; it does not spawn `/usr/sbin/sysctl` and never consults PATH.
- Parse each sample as exactly 36 ASCII UUID characters with the four fixed hyphens, reject nonhex/all-zero/embedded/trailing data, lowercase its hexadecimal digits, then compare two consecutive canonical values. Sample disagreement refuses.
- Hash only the canonical UUID bytes under the typed `DarwinBootSessionIdentity` domain and retain only the digest in the canonical target.
- Expose no public constructor from UUID, digest, timestamp, cached state, or satisfied Boolean. Never fall back to `kern.boottime`, wall clock, uptime, PID, filesystem or process-local state. Unsupported platforms refuse explicitly.
- C3 remeasures on construction/reopen and C4 independently remeasures immediately before admission. Equality/change means only that the OS reported the same/different UUID. It proves no physical reboot, prior-child absence, VM non-rollback, kernel trust, clone resistance, collision impossibility, or safe reset.

The target schema and all three joined examples change together; the parser and planning hostile cases must reject the retired profile even after an attacker coherently recomputes dependent digests. Historical C1/C2 receipts remain exact evidence for their historical trees and are not rewritten.

## Scope corrections carried into C3

- Replace C3's directory prefixes with an exact forty-path, empty-prefix `SOURCE_FULL` roster at `sha256:c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb` so source-final and credential gates are exact.
- Materialize and reopen the single target directly from opaque `gitobj.InspectedTree`; never manufacture `WorldPlan`, `BoundCandidate`, or `WorldInstance` authority.
- Include `internal/store/public_api_test.go`; the narrow inert C3 bridge must be compiler-visible and frozen, never hidden behind generic bytes or a private-surface bypass.
- Declare C3B before C4 as a separate exact three-path `RECEIPT_RECONCILIATION` unit with digest `sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67`.
- C3's in-scope `tools/check-p07b-c-plan.mjs` must implement both absent-receipt and receipt-present C3B validation before the C3 source seals. C3B has no separate checker file and may not invent one outside its exact three paths.

## Evidence boundary

C3P's exact twelve-claim final ledger is declared in its status and checker. Commit as `fix: use stable Darwin boot-session identity`, seal, require the Git note, loop strict verification to exit `0`, emit the exact-commit HTML report, and archive the ledger. Then C3PB may bind only those already-sealed source receipts through its exact three paths, roster digest `sha256:531cbe600cf2747752749890ea9a15c3d0126e498da6838d4f6faeb1a0d7e885`, and standard seven narrow claims. C3PB cannot name or grade itself. C3 remains blocked until both boundaries and any intervening separately declared maintenance unit are strict-clean.

---

# C3 — publish one exact pre-spawn target

**Risk:** high. This joins live Git, private materialization, runtime, attempt, and store authority.
**Files:** the exact forty-path, empty-prefix machine roster only; no directory wildcard or inferred descendant is admitted.

## Goal

Make C3 the sole producer of an opaque `OfficialTarget`: one C2-reopened inert target record rejoined with the reopened terminal residue, one live pinned and verified Git materialization, one fresh attempt, one live measured boot session, and one admitted Node runtime. C3 publishes no run, acquires no interlock, and never mints a permit.

## Exact changes

- Materialize/revalidate directly from opaque `InspectedTree`; share only private mechanics with comparison materialization.
- Treat content-portable tree identity as object format plus immutable commit/tree OIDs. Keep repository fingerprint, Git version, and display ref as local provenance, open source and materialization repositories independently, and reject symlink or noncanonical roots.
- Admit one explicit absolute Node path with owned probe and exact identity summaries. Measure `DARWIN_KERN_BOOTSESSIONUUID_V1` with the C3P parser/canonicalization contract and opaque capability; no caller-authored boot material enters.
- Allocate/sync a fresh CONFORMANCE attempt attachment. Construct the C1 target only from C3-owned live prerequisites, persist it through C2's inert-record mechanics, then reopen the exact typed relationship.
- Return only sealed `OfficialTarget`. It privately carries the exact C2 record, but that record remains mechanically inert when obtained outside C3. No public C3 API accepts copied target bytes, a digest/OID/runtime-string bag, generic CAS authority, or a prior process-local capability as a substitute.
- Reopening official authority remeasures and revalidates the full C3 prerequisite graph, including live boot identity. C4 must independently measure boot again; a stale-boot target is never spawnable, while a changed UUID still makes no survivor-absence claim.
- `OfficialTarget.Valid()` is structural only. A fresh `ReopenOfficialTarget` must refuse target/runtime/link/executable/materialization/boot drift and is the only authority C4 may consume adjacent to physical admission. The owned probe binds the cooperating executable's reported tuple and probe digest, not Node vendor authenticity.
- The attempt-to-target relationship is create-once and first-durable-writer-wins. C3 allocates one attempt per publication; restart rebuilds every live capability instead of trusting stored locator fields. Exercise a 24-phase pre-spawn fault seam without exporting it as production authority.
- Enroll the C3 target gate and hostile self-test in the cumulative verifier, and assert study-head inode/bytes remain unchanged.
- Bind the exact C3S/C3F/C3L/C3A predecessor authority manifest, including parents, trees, subjects, note blob types/identities/raw-body digests, ordered claims, argv previews, and complete coverage.

## MUST NOT change

- No fake one-member `WorldPlan`, `BoundCandidate`, or `WorldInstance`; no PATH lookup, `sysctl` subprocess, clock/uptime/PID/file/process-local boot fallback, raw boot constructor, dirty worktree, arbitrary executable, candidate/subject spawn, run, classification, TAP, interlock acquisition, StartClaim, RunPermit, FCR, execution status, or head advance. One bounded owner-controlled Node runtime-probe subprocess is required and is not a subject run.

## Tests

- Moving-ref pin, copied OID/body/generic-record refusal, dirty-worktree exclusion, unsupported tree forms, target-link mutation, runtime symlink/byte/probe mismatch, attempt reuse, boot mismatch, closed capability, restart full-live-authority rebuild, and pre-spawn fault matrix. Offline store clones remain an explicit nonclaim.
- The live-parent matrix snapshots every scoped store path before refusal and proves exact names, modes, inode identities, ownership, link counts, sizes, and regular-file hashes are unchanged; an independently opened exact-content repository is the positive portable-identity control.
- `OfficialTarget.Valid()` is snapshot-only and performs no storage repair. Hostile materialization and relationship mutations leave structural capability validity distinct from durable `ContractTargetRecord.Valid()`, while only physical reopen may refresh or refuse the graph.
- Run exactly `TestC3OfficialTargetClosedCapabilityAndDefensiveGetters`, `TestC3HostEpochConcurrentMeasurementsNeverCache`, `TestC3NodeRuntimeCopiedCapabilitiesSerializeRevalidation`, and `TestC3ConformanceAttemptConcurrentValidationAndReopenAreRaceFree` under `-race`, `-p=1`, and `-timeout=20m` across their four owning packages. This is the direct-goroutine/sync/atomic C3 test-source roster; the five complete non-race authority profiles retain every other C3 behavior, including Git materialization.
- The sole-issuer check uses the current Darwin/arm64/cgo Go build selection and import-qualified module-wide named-type and function graphs. It carries source-ordered lexical block/function/receiver generic scopes and structurally substitutes binding-sensitive instantiated arguments through output-bearing positions, including union terms and dependent/renamed receiver constraints. Known generic function call/value results and direct or shallowest-promoted generic method call/value/expression results are checked positionally; parenthesized callee and built-in forms normalize before resolution; omitted inferred output parameters fail closed; indexed module function/method call multi-results are tuple-positioned; and input-only generic consumers remain allowed. Hostile cases cover exported variables plus nested/indexed/shallowest-promoted selected fields, inferred conversion/make/new/identifier initializers, multiple dot imports, function-local and cross-package wrappers (including true declared import names), repeated nested generic instantiations, and held factories; unresolved external generic calls with target-bearing type arguments remain conservatively refused. Local value/function shadowing and non-call comma-ok tuple projection are not modeled exactly, and the check remains an explicit cooperating-code shape policy rather than whole-program dataflow proof.
- Treat C3's verifier and verifier-self-test edits as roster-only enrollment of the seven C3 steps, not as an execution-setting change: cache lifetime, parallelism, package classification, child environment, lock semantics, and the repetition helper remain unchanged, while sequential verification work materially increases. Record this as an explicit superseding exception to C3V's blanket future-byte trigger, not literal satisfaction. The 52-case C1V load-sensitive qualification matrix is not rerun; the dedicated verifier self-test plus one cumulative pass establish only the evolved roster and one composition snapshot, not repeated load-flake stability or a new performance ceiling. Any later change to those execution authorities must rerun the complete matrix.
- The final source boundary uses one private eighteen-claim source ledger. The later exact three-path boundary uses a separate seven-claim C3B receipt ledger and validates both receipt-absent C3 and receipt-present C3B documentation without reusing a source event.

## Gate

Machine-cleared pre-spawn target authority only. Architecture evidence proves C3 is the only production `OfficialTarget` issuer while C2 remains inert. This is the minimum honest cut if later physical evidence cannot be proven.

C3 does not hand durable grades to C4 until the separate exact three-path C3B receipt boundary binds the sealed source note, claim map, HTML snapshot, and ignored ledger, then independently seals and verifies strictly.

## Commit

`feat: publish exact contract execution targets`

---

# C4V — repair the execution-unit verification contract

**Risk:** high. This boundary edits the parent-sealed plan/scope authority that must constrain both remaining product units.
**Files:** exactly `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/THREAT_MODEL.md`, `docs/VERIFICATION.md`, `docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md`, `docs/status/P07B-C-C4V-EXECUTION-CONTRACT-MAINTENANCE.md`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, `tools/check-p07b-c-unit-scope.mjs`, `tools/verify-current-selftest.mjs`, and `tools/verify-current.mjs`; sorted-newline digest `sha256:535a64fbb9f0d3c3672b4b8197fc7815da315117b1794a668e6565faa16df3f1`.

## Goal

Insert `C3D → C4V → C4`, preserve C3D's generated structural fixtures plus four fixed semantic regressions, and freeze executable C4 and C5 source/evidence contracts before either product boundary may start. C4V changes no Go product source, but it conservatively reverts the cumulative verifier's direct build, vet, and general-package jobs from `p=2` to `p=1` after the second independent late A2 AST deadline fired C1V's declared recurrence refusal.

## Exact changes

- Require exact sealed C3D commit `62422a2400edd702c82fb680c6ef0c6923eb6cfb`, tree `7573f57017c6f56b434d1fbd5642ac585f7789c2`, one parent, subject, source diff, note blob/body, ten claims, argv, grades, coverage, and current-HEAD identity.
- Evolve only the phase table to `countershape/p07b-c-unit-paths/v17`: 19 rows, authority `sha256:11f96a7809d845ef21671d8f847bd3b9f324e84c539cd2f93ab2bc14929ecd46`; 722 generic cases at `c31ebebeac2fdc5d522b466ab082bcd60c5da5b488b7c5be96b7c6cfe1d15aac`; 726 combined cases at `bb433b824bc306bb02138122e1e563f2584c4862766d9721b4881d5f1aa23132`; exact 365/361 acceptance split; plan transitions 7/41 and independent transitions 7/24.
- Preserve C1V's sealed p2 result as historical evidence while applying its live recurrence refusal: direct build, vet, and general testing use `p=1`; `GOMAXPROCS=2`, test `-parallel=2`, sensitive/nested serialization, fresh private cache, exclusive lock, every deadline, package roster, ordering, and assertion remain unchanged. C4 and C5 inherit this current p1 setting unless a later independently requalified verifier boundary supersedes it. Do not lengthen the 60/90-second A2 timeout hierarchy as a throughput repair.
- Retain the permanent failed final attempt at `.didrun-history/p07b-c-c4v-final-attempt-1-ast-sigterm/.didrun/` and `.countershape/p07bc-c4v-final-attempt-1-ast-sigterm`: seven events, six claims, unclaimed event 6 exit `1` at cumulative row `23/46`, `architecture-p07b-a2-2-selftest`, after `257205` ms. It yielded no commit, seal, note, strict result, HTML report, or grade; none of its events may support the replacement final ledger.
- Freeze the current positive-receipt refusal corpus to 46 sorted Markdown authorities at `sha256:b480bb8e93d9ddd6075a57bdf8dbe99cb663eb1f0a11228af31698b54a5dd2f2`, including both sealed C3D and active C4V status surfaces; retain the sealed historical 44-path corpus unchanged.
- Route the machinery proposal `DEFECT_ALREADY_REPAIRED` and `INTENTIONAL_HYBRID`; retain fixed oracles `C3R_NOTE_PREVIEW`, `C3Q_EXTERNAL_SELF_RECEIPT`, `C3T_VALIDATED_ABSENT_BASELINE`, and `C3U_TERMINAL_RECEIPT_LAYERING`. Decline a pure four-field replacement because those cases encode note projection, outer/historical scanner asymmetry, validated absent lifecycle, and terminal legacy-byte preservation.
- Freeze C4 to 38 exact paths at `sha256:99bda0d3e5c2f2a2ee4f354945188bf2e5563716c9e26511211a4bfc9197bf56` plus required realized prefixes `internal/contractexec/runner/`, `internal/processmechanics/`, and `testkit/contractexec/cli/` at `sha256:f5f7b8cdfeba3581e8632c0d6686287f5afa00aef224bb4babe39ee1fdc7f4b2`. Added exact paths include C4-authored `spec/verification/p07b-b-future-surface-authority.json` and C4-sealed shared execution authority `tools/verify-runtime-authority.mjs`; neither is owned by C5.
- Freeze C5 to 12 exact paths at `sha256:d16ba74582e53dc5c791b8c34fbcd6200338dc066ec13230c2908eb6ba257bf5` plus required realized prefixes `internal/contractexec/http/`, `internal/contractexec/scope/`, and `testkit/contractexec/http/` at `sha256:a9212680066fdbddc8f25eef04a41c264ac9148a506238dd92c39978e3e35f84`. Its exact roster includes `docs/VERIFICATION.md` because C5 evolves the current verifier; C6A owns that manual for the same reason. C5 owns its C architecture checker and self-test, but owns neither `spec/verification/p07b-b-future-surface-authority.json`, the B checker/self-test, the phase plan/scope checker, the phase specification, repetition helper, nor runtime authority; it may only satisfy the parent-sealed rows/topology.
- Require C4, the last pre-C5 writer of the inherited B checker, to preserve all sealed B authorities. C4V seals the independent production-Go scanner/manifest schema `sha256:91c4ca13a4b30a683b82066fdcdc1a7892de2679a2e5419a57142e36b22f5937`, canonical baseline policy `sha256:5a1c3e2f33969143765612f4934ce4ca41cc1e3df95383b8d7780354461e3399`, and exact live-and-strict-scanned 27-row historical inventory `sha256:901d7cb5ca65cb9cb1ca7932ece1741fbfc82df5f9e375d43fadd6e057615960`; C4 authors canonical `spec/verification/p07b-b-future-surface-authority.json`, binds its real C4 delta, and predeclares exact C5 rows, which may be empty. C4 requires runner present and HTTP/scope absent; C5 preserves manifest/checkers unchanged, matches the combined rows, and realizes all three. Reject historical deletion, C5-without-C4, partial/mixed topology, missing/extra/relocated/lookalike rows or tokens, and schema/digest/order drift. Private forward fixtures prove validator/state-machine behavior only—not future source existence or conformance. The one-known-row `sha256:4ebfb5ae2608f0cdaa2e316e087fd8c8e479a891e7e52d0a7aab7f9a45541807` is `TEST_VECTOR_ONLY/INCOMPLETE`.
- Freeze C4's exact 80-command/claim runbook, including 54 separate qualification-case receipts, sealed-C4V consumer, private root `.countershape/p07bc-c4-final`, and subject `feat: finalize CLI contract executions`.
- Freeze C5's exact 79-command/claim runbook, including 56 separate qualification-case receipts, sealed-C4 consumer plus direct C4V/C3D ancestry validation, private root `.countershape/p07bc-c5-final`, and subject `feat: complete standalone contract execution`.
- Future Git-note argv previews require exact length, exact tail, raw prefix tokens or only the position-specific didrun projections: double redaction at zero-based argv indexes 2/4/5/6/7/8 and bare redaction at zero-based indexes 31/32/33/35/36. `secrets_override` is a Boolean observation, not a forced value.
- Preserve the live 52-case C1V qualification through C4V and require legacy helper SHA-256 `1321fb1381f1f74d0959b9ceb40b4dcc4abc637bb281ff2af759863ede3bfd5a` before its lazy import. Freeze C4's exact 54-case catalog at `sha256:3be703fae10155c83b19be54ae7cd79cfc5b1bbaddfcf86bb2c9781ad93f853a` and its dormant exact 56-case C5 catalog at `sha256:08db7c338eb3ba900cf6df2545c04f27bed16889c81dd16e5fb18197c4d52958`. The helper exports recursively frozen `qualificationMatrices`, exact C4 identity aliases, the two digests, and `qualificationCaseForID(caseID)`. Its exact route is parse → canonicalize → validate → build → run: `parseRunArguments(argv)` accepts the exact `--case <id>` branch then has the exact throwing ad-hoc preamble; `validateExecutionSpecification(specification)` reselects the named identity and refuses copies; `buildGoTestArguments(specification)` returns the exact frozen 11-element Go argv; `runRepetition(specification, dependencies = {})` begins with canonical validation and builds/executes only from `executionSpecification`; `main()` ends with `await runRepetition(parseRunArguments(process.argv.slice(2)))`. At live C4/C5, `qualificationStaticAuthority` decodes the recursively frozen literal catalog into a checker-owned projection and never imports or evaluates future candidate helper bytes. The static gate does not execute candidate parser, selector, validator, or builder exports; actual execution evidence is the separate helper self-test, per-case receipts, and preseal reconciliation. C4's self-test drives both dormant C5-only identities through parser, selector, validator, exact builder, and a dependency-injected run entry without runtime acquisition or absent-package execution; malformed argv, unknown IDs, copied specifications, raw lookup, and builder drift fail before child execution.

## Verification

C4V uses exactly 68 claims: 65 `tests-pass`, then three `command-succeeded`. In order: candidate C4V plan, independent transition, plan self-test, sealed-C3D note, scope self-test, verifier self-test, repetition-helper self-test, all 52 frozen C1V `--case <id>` qualifications as isolated events, three unchanged 117-case A2 architecture self-tests, three complete 46-row cumulative verifiers, C4V exact eleven-path source-final gate, credential scan, and C4V preseal ledger. Render the exact instructions with `--print-final-runbook C4V`; all commands run through `didrun run --` and are claimed immediately. Any nonzero event archives the entire attempt and restarts from event zero.

## Gate

Commit exactly `fix: declare executable C4 and C5 contracts`, seal, inspect the note, and loop `NO_COLOR=1 didrun verify --strict` until exit zero on a new repaired commit if necessary. C4 remains blocked until this boundary is sealed and strict-clean. Every C4V/C4/C5 grade is `UNRECEIPTED` before its own execution.

---

# C4 — close a CLI candidate-profile run and publish its classification

**Risk:** high. First subject execution and first finalized/classified evidence.
**Files:** exactly the parent-sealed 38-path roster at `sha256:99bda0d3e5c2f2a2ee4f354945188bf2e5563716c9e26511211a4bfc9197bf56` plus at least one regular source path under each of `internal/contractexec/runner/`, `internal/processmechanics/`, and `testkit/contractexec/cli/` at prefix digest `sha256:f5f7b8cdfeba3581e8632c0d6686287f5afa00aef224bb4babe39ee1fdc7f4b2`. No other path is admitted.

## Goal

Extract the low-level Darwin mechanics and make the higher contract runner the sole consumer of one authentic target-bound RunPermit. Perform a Go-owned CLI candidate-profile run, close physical evidence, publish and reopen the exact `FinalizedContractRun` plus target-to-run relationship, durably release or preserve the interlock as appropriate, then publish `ContractExecution`.

## Exact changes

- Exact import graph: `processmechanics` imports no store/contract/domain package; `contractexec/runner` imports official-target/admission/runtime plus `processmechanics`; only existing `world` and the contract runner may call raw mechanics; no low-level mechanics API accepts a permit, target digest, semantic body, or public raw argv.
- The runner accepts only sealed C3 `OfficialTarget`, reopens it, compares its boot binding with a newly measured live boot identity, and revalidates target/runtime immediately before subject spawn.
- Admission is ordered and fenced: acquire one fresh `HELD(target,boot,generation)` lease; create/reopen the deterministic StartClaim with that exact fresh lease; only a claim newly created and reopened by that same live operation lets the runner construct one private, nonserializable, single-use `RunPermit`; consume it inside the runner immediately before `Start`. Existing claim/lease, restart, copied generation, losing racer, ambiguous persistence, or stale boot returns no permit and performs no spawn.
- Persist a separate SpawnObservation. Missing/unknown observation or incomplete owner closure cannot publish an FCR or classification; the target remains consumed and the interlock stays held for that boot session.
- Close capture, process, wait, drains, teardown, final group probe, materialization/runtime revalidation, and all five standalone domains for the CLI profile.
- Persist every private evidence item, exact private manifest, and closed terminal facts. Publish and reopen the exact FCR and target-to-run relationship before writing `CLEAR`; only C4's conclusive terminal-closure capability may initiate release. Spawn/private-evidence/FCR ambiguity before `CLEAR` leaves the target consumed and held. A later `CLEAR` or receipt ambiguity preserves the durable FCR but blocks admission until the exact receipt independently converges.
- Derive/publish/reopen classifier-profile-bound execution only after FCR durability. Classification-only retry never reruns a process.
- A changed-boot reset requires explicit operator action plus the exact C2 CAS fence. It is not a same-boot retry and does not claim prior-child absence.
- Revalidation proves checkpoint-bound identity plus child self-report only. Preserve the revalidation-to-`execve` same-user substitution residual; no descriptor-bound or coordinated-replacement claim is added.
- Enroll runner/architecture gates and hostile self-tests in the cumulative verifier before sealing.
- Evolve the inherited B checker/self-test in C4, its last owning boundary before C5, into the C4V-sealed manifest/state gate. Preserve every sealed B digest/import/API/publication/materialization and the exact historical 27-row C1/C2/C3 inventory. Create canonical `spec/verification/p07b-b-future-surface-authority.json`; independently scan production non-test Go tokens; bind observed C4 rows and predeclare exact C5 rows. Exercise the named states `C4-present/C5-absent` and `C4+C5-present`; use the real C4 tree, use checker-owned fixtures only for validator/topology hostiles, and reject C5-without-C4, partial/mixed/relocated/lookalike states, missing/extra rows or tokens, historical deletion, and manifest/schema drift. C5 must preserve manifest and checker bytes and pass the real combined tree.
- Move lock/tool admission, private-root construction, hermetic child environment, bounded execution, authority revalidation, failure cleanup, and success finalization into added `tools/verify-runtime-authority.mjs`. It imports Node built-ins only and performs no authority action at import time. Both `verify-current.mjs` and the repetition helper have exactly one relative named-import edge, to that runtime; the helper must not import C5-owned `verify-current.mjs` directly or indirectly. Re-exports, dynamic import, CommonJS/dynamic-evaluation loader forms including `process.getBuiltinModule`, aliased imports, top-level imported-authority aliases, IIFEs, and invoked local authority wrappers are forbidden. This is static closure conformance, not a sandbox claim.
- Replace the helper's live 52-case matrix with a separately classified, literal exact 54-case C4 matrix, including the relocated process-mechanics cases and new runner/CLI sensitive cases. Its digest must be `sha256:3be703fae10155c83b19be54ae7cd79cfc5b1bbaddfcf86bb2c9781ad93f853a`.
- In the same parent-owned helper, export recursively frozen literal `qualificationMatrices` C4/C5 records, exact digests, C4 identity aliases, and stable `qualificationCaseForID(caseID)`. Implement the exact parse → canonicalize → validate → build → run contract: `parseRunArguments(argv)` has the exact selector branch plus throwing ad-hoc preamble; `validateExecutionSpecification(specification)` begins with exact selector-based named identity refusal; `buildGoTestArguments(specification)` returns `test -json -mod=readonly -buildvcs=false`, profile-bound `-p`, `-parallel=2`, exact count, `-timeout=20m`, run, and package in one frozen 11-element array; `runRepetition(specification, dependencies = {})` validates first and passes only that built argv to the dependency-bound child. Async `main()` ends in the exact awaited parse-to-run route before the final canonical `pathToFileURL`-bound direct entry awaits `main()`. The helper and verifier use exact nonaliased Node URL bindings and inert module initialization. The C4 helper self-test drives the dormant C5-only identities through parser/selector/validator/builder and a no-authority dependency-injected run without executing absent C5 packages. The C4V-sealed checker validates source shape plus a checker-owned literal projection, never imports candidate future bytes, and C4/C5 preseal reconciliation verifies each case event's stdout blob and terminal authority/result line. Each of the 54 C4 cases executes through a distinct didrun event and immediate claim, with a separate lock, admitted tool snapshot, fresh private roots/environment, bounded child result plus Go JSON validation, cleanup/finalization, and terminal evidence. There is no shared qualification session or aggregate matrix receipt.

At preseal reconciliation, independently derive exact ordered `[go,node,git,sh,cc,cxx]` from frozen `COUNTERSHAPE_GO/NODE/GIT/SH/CC/CXX`; realpath each; require executable regular non-symlink files; bounded no-follow raw reads up to 512 MiB with before/after identity; SHA-256 fingerprints; and Node realpath equal to the invoking checker. Every 54/56 stdout blob must match that roster exactly; the terminal digest is SHA-256 of UTF-8 `JSON.stringify(roster)`; reopen the roster after all blobs; reject a self-consistent foreign roster. This is bounded preseal-time admission plus post-blob re-admission and recorded event output, not continuous executable-byte identity between execution and reconciliation, across events, or against same-uid swap-and-restore/TOCTOU.

## MUST NOT change

- No TAP-derived authority, raw target/body/digest admission, raw argv public edge, direct low-level-mechanics permit/semantic-body edge, process resume, same-target retry, lease/PID takeover, same-boot reset after ambiguity, automatic reset, FCR mutation after purge/loss, clone-resistance, confidentiality, secure-erasure claim, HTTP, product CLI, full two-profile P07B-C claim, result list/status, or head mutation.

## Tests

- Conforms/contradicts/ineligible; selected/unselected field behavior; every primary-plus-cleanup combination; all five CLI scope detectors and their forbidden-positive controls; partial/violated scope; poisoned TAP independence; target/run/profile mismatch; interlock-before-claim ordering; missing observation; crash at every admission/terminal/release boundary; same-boot retry refusal; changed-boot explicit reset; classification-only retry.

## Gate

Machine-cleared CLI candidate-profile-only slice on the exact exercised native runtime. This is not the P08 product CLI. HTTP and full P07B-C remain `UNRECEIPTED`.

The final ledger is exactly 80 commands and 80 immediate claims: 77 `tests-pass`, then three `command-succeeded`. It includes the six focused C4 profiles, C/B/U6 architecture and self-tests, U2/U3 compatibility, repetition self-test plus 54 separate exact `--case <id>` qualification receipts, `--verify-c4-sealed-c4v-note`, scope/verifier self-tests, three cumulative passes, source-final gate, credential scan, and `--verify-c4-preseal-ledger`. Each qualification receipt preserves its own lock and fresh private execution roots. Use private root `.countershape/p07bc-c4-final` and generate the exact sequence with `--print-final-runbook C4`. Keep `docs/PROMPT_PACK.md` phase-independent and update `docs/VERIFICATION.md` for the C5/C6A current-verifier ownership rule.

## Commit

`feat: finalize CLI contract executions`

---

# C5 — add HTTP and complete standalone-scope evidence

**Risk:** high. Adds readiness/service/import evidence and completes the narrow standalone claim.
**Files:** exactly 12 paths at `sha256:d16ba74582e53dc5c791b8c34fbcd6200338dc066ec13230c2908eb6ba257bf5` plus at least one regular source path under each of `internal/contractexec/http/`, `internal/contractexec/scope/`, and `testkit/contractexec/http/` at prefix digest `sha256:a9212680066fdbddc8f25eef04a41c264ac9148a506238dd92c39978e3e35f84`. The exact roster includes `docs/VERIFICATION.md`; C5 may not edit either phase checker, the specification, the repetition helper, or `tools/verify-runtime-authority.mjs`.

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

The final ledger is exactly 79 commands and 79 immediate claims: 76 `tests-pass`, then three `command-succeeded`. It includes five focused C5 profiles, C/B/U6 architecture and self-tests, repetition self-test plus 56 separate parent-sealed `--case <id>` qualification receipts, `--verify-c5-sealed-c4-note`, scope/verifier self-tests, three cumulative passes, source-final gate, credential scan, and `--verify-c5-preseal-ledger`. Each qualification receipt preserves its own lock and fresh private execution roots. Use private root `.countershape/p07bc-c5-final` and generate the exact sequence with `--print-final-runbook C5`. The sealed-C4 consumer must validate C4's one-parent exact A/M source diff, all three realized prefixes, 80-claim note, full argv/grade/coverage, and the direct C4V/C3D chain under one stable outer HEAD.

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

- Full repo twice under stock macOS TMPDIR; the exact sealed C1V general-`p=2`/sensitive-`p=1` profile; race/vet/count-fuzz; architecture/checker selftests; exact physical CLI/HTTP matrix; direct generated Node parity; unchanged head; staged scope/credential-prefix checks.
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
