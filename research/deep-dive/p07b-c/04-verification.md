# Lane 4 — verification architecture

## Verdict

P07B-C must be receipted as multiple independently sealed units. Schema-valid examples, planning validators, generated TAP, or direct `t.TempDir` fixtures cannot substitute for the live authority chain.

## Acceptance layers

| Layer | Required proof | Primary false-green to kill |
| --- | --- | --- |
| Semantic model | strict runtime/schema intersection, canonical goldens, exhaustive outcome/standalone cross-product, property/fuzz vectors | caller-paired reason, tuple, scope, or classification survives |
| Nonhead store/interlock | generic CAS/C2 fixtures cannot mint official authority; typed witness/link/interlock corruption, race, crash, boot-reset, and ambiguity matrix | parsed object or losing publisher receives a permit; fresh target bypasses unknown survivor |
| Pre-spawn target | exact Git pin/inspect/materialize, explicit Node admission, fresh attempt, target publish/reopen | dirty tree, PATH runtime, fake comparison authority, copied digest reaches spawn |
| Physical run | Go-owned CLI/HTTP path, terminal lifecycle, private evidence, COMPLETE/PARTIAL/VIOLATED detectors | TAP becomes authority; eligible result precedes teardown or absence closure |
| Classification | exact target/run/profile join and constructor-derived result | caller-selected class, run/target cross-pair, ineligible collapsed to contradiction |
| Cumulative | architecture hostile self-test, full native matrix, race/vet/fuzz, repeat under stock TMPDIR, exact receipts | focused suite passes while cumulative roster omits it or binds stale tree |

## Test families

- Exhaustive state and tagged-sum enumeration.
- Canonical round-trip, strict field roster, defensive-copy, and typed-kind properties.
- Store winner/loser concurrency and create/sync/reopen fault phases.
- Git moving-ref, copied-capability, mutation, and no-dirty-worktree boundaries.
- Runtime admission/revalidation and owned-probe mismatches.
- Process start/timeout/cap/cancel/TERM/KILL/drain/orphan cases, including every retained primary-plus-cleanup combination.
- Standalone positive controls: each detector must reject one planted forbidden condition.
- Differential parity between direct generated Node behavior and Go-owned result without trusting TAP.
- Restart and ambiguity matrices for target, boot-session interlock, start intent, terminal witness, run, and classification publication; same-boot ambiguity blocks all later subject spawn.
- Repeated-run uniqueness and byte-for-byte unchanged study head.

Keep this resumed proof black-box/property/parity/recovery/boundary based. Do not recreate the abandoned dense source-rewrite recipe corpus.

## Resource discipline

Use `GOMAXPROCS=2`, Go `-p=1`, race `-p=1`, and fuzz `-parallel=1 -p=1` with count budgets. Development can run focused smoke checks; each boundary must run its full named matrix before seal, and cumulative closure reruns the frozen tree.

## didrun ceiling

Each load-bearing command runs through `didrun run --`; each successful final event is claimed immediately. Claims remain narrow: semantic algebra, nonhead persistence, target authority, runtime admission, lifecycle, standalone scope, classification, architecture, race, fuzz, cumulative. A single “P07B-C works” claim is prohibited.

## Confidence

**8.8/10** in the verification architecture; executed P07B-C validation gates in this review: zero.
