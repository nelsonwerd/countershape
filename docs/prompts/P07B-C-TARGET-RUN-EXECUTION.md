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
5. Inherit sealed C1V exactly: `GOMAXPROCS=2`; direct build/vet and the exact general-package complement use `-p=2`; the six declared sensitive packages (including `internal/store`) and every inherited/nested Go invocation use `-p=1`; fuzz uses `-parallel=1` with a count budget.
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
- `tools/verify-current.mjs` — cumulative roster; C0's global serialization repair is preserved as history, C1V's qualified general-`p=2`/sensitive-`p=1` tier is current, and every later unit enrolls its own gate before sealing.

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
| C2 | nonhead storage mechanics, exact mappings, private evidence, interlock/claim substrate with test-only issuers; then a separate narrow C2B source-receipt reconciliation | persistence substrate only; no official target or production permit |
| C3 | direct single-target Git + Node authority and target publication | exact pre-spawn target authority only |
| C4 | process mechanics plus contract runner, production interlock/permit, CLI-profile physical closure, FCR, classification | CLI-profile-only native physical slice |
| C5 | HTTP plus full five-domain standalone evidence | full P07B-C on exact exercised native tuple |
| C6 | C6a cumulative hostile/fault/parity/expert-surface closure, then C6b receipt reconciliation | exact sealed P07B-C claims only |

Every source boundary that must hand durable grades to a later unit is followed by its separately sealed receipt-reconciliation subunit. C2B may bind only the already-sealed C2 source note, strict result, exact claim map, HTML snapshot, and ignored-ledger manifest; it cannot edit source/checker behavior or grade itself. C3 remains blocked until C2B also seals and verifies strictly.

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

# C3 — publish one exact pre-spawn target

**Risk:** high. This joins live Git, private materialization, runtime, attempt, and store authority.
**Files:** `internal/gitobj/single_target.go`, `internal/noderuntime/**`, `internal/hostepoch/**`, `internal/contractexec/target*.go`, the exact C2 nonhead-store files named in the C3 allowlist, production target-publication owner, tests/checkers/self-tests, cumulative roster, docs/status.

## Goal

Make C3 the sole producer of an opaque `OfficialTarget`: one C2-reopened inert target record rejoined with the reopened terminal residue, one live pinned and verified Git materialization, one fresh attempt, one live measured boot session, and one admitted Node runtime. C3 publishes no run, acquires no interlock, and never mints a permit.

## Exact changes

- Materialize/revalidate directly from opaque `InspectedTree`; share only private mechanics with comparison materialization.
- Admit one explicit absolute Node path with owned probe and exact identity summaries, plus the stable Darwin boot-session identity used by later admission/reset.
- Allocate/sync a fresh CONFORMANCE attempt attachment. Construct the C1 target only from C3-owned live prerequisites, persist it through C2's inert-record mechanics, then reopen the exact typed relationship.
- Return only sealed `OfficialTarget`. It privately carries the exact C2 record, but that record remains mechanically inert when obtained outside C3. No public C3 API accepts copied target bytes, a digest/OID/runtime-string bag, generic CAS authority, or a prior process-local capability as a substitute.
- Reopening official authority remeasures and revalidates the full C3 prerequisite graph, including live boot identity. C4 must measure boot again; a stale-boot target is never spawnable.
- Enroll the C3 target gate and hostile self-test in the cumulative verifier, and assert study-head inode/bytes remain unchanged.

## MUST NOT change

- No fake one-member `WorldPlan`, `BoundCandidate`, or `WorldInstance`; no PATH lookup, dirty worktree, arbitrary executable, candidate/subject spawn, run, classification, TAP, interlock acquisition, StartClaim, RunPermit, FCR, execution status, or head advance. One bounded owner-controlled Node runtime-probe subprocess is required and is not a subject run.

## Tests

- Moving-ref pin, copied OID/body/generic-record refusal, dirty-worktree exclusion, unsupported tree forms, target-link mutation, runtime symlink/byte/probe mismatch, attempt reuse, boot mismatch, closed capability, restart full-live-authority rebuild, and pre-spawn fault matrix. Offline store clones remain an explicit nonclaim.

## Gate

Machine-cleared pre-spawn target authority only. Architecture evidence proves C3 is the only production `OfficialTarget` issuer while C2 remains inert. This is the minimum honest cut if later physical evidence cannot be proven.

## Commit

`feat: publish exact contract execution targets`

---

# C4 — close a CLI candidate-profile run and publish its classification

**Risk:** high. First subject execution and first finalized/classified evidence.
**Files:** `internal/processmechanics/**`, `internal/contractexec/runner/**`, the exact C2 store files named in the C4 allowlist, CLI candidate-profile service, production admission/run/classification publication, private capture manifest, tests/checkers/self-tests, cumulative roster, docs/status.

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

## MUST NOT change

- No TAP-derived authority, raw target/body/digest admission, raw argv public edge, direct low-level-mechanics permit/semantic-body edge, process resume, same-target retry, lease/PID takeover, same-boot reset after ambiguity, automatic reset, FCR mutation after purge/loss, clone-resistance, confidentiality, secure-erasure claim, HTTP, product CLI, full two-profile P07B-C claim, result list/status, or head mutation.

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
