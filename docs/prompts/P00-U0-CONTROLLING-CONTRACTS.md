# P00 / U0 — Lock the controlling contracts

## Mission

Execute the first shippable unit of Countershape in a fresh chat. Establish the repository, lock the red-teamed contracts, and make the planning layer mechanically self-consistent. This is a documentation/specification unit: do not implement candidate execution, comparison, reduction, a server, or UI. The output is the exact contract all later units must obey, plus the first sealed didrun gate.

Work autonomously from the repository root. Preserve unrelated user changes. Give short phase updates, and use files as durable memory. The user has authorized the branch and commits named below.

## Required read set

Read these files completely, in this order:

1. `/Users/drewnelson/.claude/CLAUDE.md`
2. `docs/AUTOPILOT_KICKOFF.md`
3. `docs/PERSONA_SOREN_VALE.md`
4. `research/ideation/02-wake-overlap-audit.md`
5. `research/deep-dive/07-SYNTHESIS.md`
6. `research/deep-dive/08-RED_TEAM.md`
7. `docs/CONCEPT_BRIEF.md`
8. `docs/PROMPT_PACK.md` and every existing `docs/prompts/P*.md`
9. `docs/HANDOFF_MODE_C.md`

Treat the revised concept brief as product authority and the red team as the stricter interpretation when wording differs. Inspect `git status --short`, existing files, installed tool versions, and `didrun --help` without rewriting user work.

## Repository bootstrap

Before any build artifact, run `git init` even if the directory is already a repository; it is idempotent and satisfies the explicit bootstrap contract. Create or switch to `codex/countershape-autopilot` without destroying another branch. Do not reset, clean, amend away history, or stage `.didrun`.

Make `.gitignore` explicitly cover at least:

```text
.didrun/
.countershape/cache/
.countershape/*.db
.countershape/*.db-shm
.countershape/*.db-wal
```

Keep any existing valid ignores. `.didrun/` may contain secret-bearing evidence and must never be committed.

## Locked invariants and cuts

Write contracts for the narrowed Darwin reference instrument, not the broader synthesis draft. Keep all of these non-negotiable:

- Countershape is a repository-scale operational composition, not a novel disambiguation algorithm, semantic primitive, oracle, winner selector, or Wake adaptation.
- Only trusted local code, curated proof repositories, Darwin runtime when natively receipted, Git modes `100644` and `100755`, one CLI invocation, and one loopback HTTP/1.1 request are in the run claim.
- Git selection, materialized bytes, deterministic `WorldPlan`, measured `WorldInstance`, declared `ComparisonEnvelope`, captured evidence, projection, human ruling, emitted source, and current conformance are separate jurisdictions.
- A comparison envelope records measured, required-equal, tolerated, rejected, and uncontrolled dimensions. It never proves behavioral compatibility or placeholder irrelevance.
- Exact canonical projection bytes and the complete sorted `candidate_execution_key -> projection_fingerprint` map are authoritative. Group shape, ordinal cluster IDs, branch labels, order, and majority are not.
- Control failures never become behavior. Finite repeats produce only `OBSERVED_STABLE(k/k,h)`, `UNSTABLE`, `UNCOMPARABLE`, or `INCOMPLETE`.
- Reduction is tri-valued and can earn only `UNCHANGED`, `BEST_KNOWN`, or construction-safe `ONE_MINIMAL_UNDER(...)`; no “smallest” claim.
- Every evidentiary attempt is physically new. No prepared roots, reset hooks, snapshots, reflinks, or execution-evidence cache.
- Only exact witnessed, explicitly selected fields compile. Weak separation is `AMBIGUOUS_SCOPE`; allow-many is a set of complete tuples. `REJECT_ALL`, `DEFER`, and `REFINE` cannot emit code.
- Standalone Node semantics are a second implementation and remain unclaimed until one normative Go/Node corpus and an actual absence run pass.
- Reports are default-minimized local evidence exports with `CONFIDENTIALITY NOT ESTABLISHED`, never “safe” or “shareable.”
- Wake may provide inert candidate provenance only. didrun grades stay opaque and verbatim; missing evidence is `UNRECEIPTED`.

## File ownership and required artifacts

U0 owns only planning/specification surfaces:

```text
.gitignore
docs/ARCHITECTURE.md
docs/THREAT_MODEL.md
docs/CLAIM_VOCABULARY.md
docs/STATE_MACHINES.md
docs/status/U0.md
spec/schema/v1/*.schema.json
spec/examples/v1/*.json
spec/vectors/v1/*.jsonl
tools/validate-planning.mjs
docs/HANDOFF_MODE_C.md
```

The first planning commit must also include the already-authored, reviewed baseline under `research/**`, `docs/AUTOPILOT_KICKOFF.md`, `docs/PERSONA_SOREN_VALE.md`, `docs/CONCEPT_BRIEF.md`, `docs/PROMPT_PACK.md`, and `docs/prompts/**`. Validate those files but do not overwrite them unless fixing a demonstrable contradiction. Do not stage scratch/raw secrets or product code. `tools/validate-planning.mjs` must use Node core only; it is a planning validator, not a competing runtime authority.

The architecture document must lock package direction, the generic/domain boundary, immutable artifact graph plus CAS heads, and absence of event-runtime/replay semantics. The threat model must include full-user-permission execution, `HOST_ALLOWED`, process-group cleanup limits, localhost forgery controls, candidate-output injection, report confidentiality limits, and excluded same-user/browser-extension attackers. The claim vocabulary must pair every allowed phrase with its prohibited upgrade and environment-specific receipt requirement. State machines must enumerate legal/illegal transitions and staleness.

Schemas/examples must cover at least `WorldPlan`, `WorldInstance`, `ComparisonEnvelope`, `CapturedObservation`, `StableBatch`, `OutcomeMap`, `ReductionRun`, `Choicepoint`, `DecisionRecord`, `ContractBundle`, and `ContractExecution`. Examples are inert contract examples, not proof that runtime canonicalization exists. Include malformed/refusal fixtures: duplicate candidate keys, ordinal group identity, control-as-outcome, `UNRESOLVED` one-minimality, reused confirmation evidence, empty fields, allow-many cross-product, and noncompilable emission.

## Implementation and review sequence

1. Reconcile every object/enum/budget across the brief and prompts; make a discrepancy ledger before editing.
2. Author the four controlling documents and schemas/examples with explicit schema versions and claim limits.
3. Implement the Node-core validator to parse every JSON/JSONL artifact, reject duplicate file names and missing fixture expectations, check Markdown relative links, ensure required refusal examples exist, and scan controlling docs for prohibited legacy terms such as unqualified `STABLE`, “compatible worlds,” “smallest behavior,” symlink support, safely shareable, or execution-cache reuse. Allow quoted/prohibited-vocabulary examples explicitly rather than using a brittle global ban.
4. Update `docs/status/U0.md` and `docs/HANDOFF_MODE_C.md` with what is locked, what remains unimplemented, exact next prompt, and all bets/UNRECEIPTED items.
5. Inspect the complete staged diff for accidental scope expansion and Wake/didrun overlap.

## Acceptance and negative checks

The planning validator must prove valid syntax, linked examples, enum agreement, required refusal cases, and no unresolved contradictions. Add negative self-tests that mutate a temporary copy: remove one required enum, change `UNRESOLVED` to a boolean false, introduce symlink acceptance, make `REJECT_ALL` compilable, or replace the labeled map with group IDs. Each mutation must make validation fail. A validator that passes without finding its mutation anchor is itself a failure.

Run every load-bearing command through didrun, including at minimum:

```text
didrun run -- node tools/validate-planning.mjs
didrun run -- node tools/validate-planning.mjs --self-test
didrun run -- git diff --check
```

Use `didrun show --session` to obtain exact event indices. Declare a scoped claim with commands of the form `didrun claim tests-pass --label "U0 planning contracts" --event N --path .`; attach a separate `lint-clean` or `command-succeeded` claim to the diff-check event. Then stage U0 plus the reviewed planning/research/prompt baseline named above, inspect every staged file, confirm `.didrun` and caches are absent, and commit as:

```text
docs: lock Countershape controlling contracts
```

Do not add an AI co-author trailer. Run `didrun seal --commit HEAD`, then gate on:

```text
NO_COLOR=1 didrun verify --strict
```

A nonzero exit means U0 is not complete. Fix the real defect, rerun the affected verification via `didrun run --`, make a new claim bound to the new event, commit the fix, seal again, and repeat strict verification until exit 0. Never weaken a test, delete/relabel a claim, or rewrite a failed old commit; those receipts are permanent history.

## Stop conditions and handoff

Stop before product code. If schemas cannot express the truth jurisdictions without a generic JSON observation or a boolean reduction result, keep U0 open and escalate the contradiction. If didrun malfunctions, record exact command, output, version, and workaround in `docs/status/U0.md`; do not invent a receipt.

Finish with a concise handoff containing branch, commit, strict-verification exit, verbatim claim grades, changed files, remaining bets, and `P01-U1-TRUTH-KERNEL.md` as the sole next implementation prompt. Anything not exercised through didrun must be labeled `UNRECEIPTED`.
