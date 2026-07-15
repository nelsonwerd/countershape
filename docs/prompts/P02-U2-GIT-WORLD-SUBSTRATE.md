# P02 / U2 — Build exact Git materialization and fresh Darwin worlds

## Mission

In a fresh chat, add Countershape’s first impure substrate: resolve immutable Git identities, materialize only the supported blob subset without checkout policy, and run one trusted local program in a physically fresh, measured Darwin attempt with bounded capture and cooperative process-group cleanup. Do not add a CLI/HTTP observation adapter, comparison workflow, reduction, persistence graph, server, or UI.

Start only after U1 is sealed and strict-green. Preserve user work, report short phase updates, and keep the handoff current.

## Required read set

Read completely: `/Users/drewnelson/.claude/CLAUDE.md`; `docs/CONCEPT_BRIEF.md`; `docs/ARCHITECTURE.md`; `docs/THREAT_MODEL.md`; `docs/STATE_MACHINES.md`; `docs/CLAIM_VOCABULARY.md`; `research/deep-dive/02-architecture-correctness.md`; `research/deep-dive/03-execution-security.md`; `research/deep-dive/07-SYNTHESIS.md`; `research/deep-dive/08-RED_TEAM.md`; `docs/status/U0.md`; `docs/status/U1.md`; `docs/HANDOFF_MODE_C.md`; and all U0 schemas/vectors relevant to `TreeIdentity`, `MaterializationManifest`, `WorldPlan`, `WorldInstance`, `ComparisonEnvelope`, capture, and attempt control. Inspect U1 source rather than recreating its types. Verify branch, status, and `NO_COLOR=1 didrun verify --strict` before edits.

## Non-negotiable boundary

- This executes trusted local code with the operator’s full permissions and host network. Temporary roots improve repeatability; they are not containment. Preserve the exact warning in public-facing data.
- Resolve each display ref once to immutable commit and tree OIDs. Ref spelling never becomes execution identity. Reject duplicate executable trees unless the spec explicitly coalesces them.
- Materialization may use only filter-free Git plumbing such as `ls-tree`/`cat-file`. Never call checkout, switch, worktree, archive, restore, smudge/clean filters, hooks, or a working-tree path.
- Invoke Git with system/global config neutralized, replacement interpretation disabled, lazy promisor fetch disabled, prompts disabled, and no permitted network fallback. Treat a missing object as a typed refusal.
- Support only modes `100644` and `100755`. Reject all symlinks, gitlinks, LFS pointers, unsupported modes, invalid UTF-8, absolute/dot/traversal/unsafe paths, aliases/collisions on the measured target filesystem, missing objects, oversize entries, and budget exhaustion.
- Stream every blob, verify its Git object hash using the repository object format, compute a portable SHA-256 digest, and write into a private mode-0700 root without `.git`. A manifest proves only supported files written under this policy.
- Every evidentiary attempt allocates new materialization, `HOME`, `TMPDIR`, and state roots and creates an attempt artifact before spawn. There is no execution cache, prepared root, clone, snapshot, reflink, reset hook, or evidence reuse path.
- Use absolute argv execution with no shell and a sparse allowlist environment. Record measured facts separately from deterministic `WorldPlan`; allocated root, port, PID, clock, nonce, platform, and cleanup belong to `WorldInstance`.
- On Darwin, own a new POSIX process group; use TERM, bounded grace, KILL, bounded drains, direct-child wait, and a final group probe. An adversarial descendant may `setsid` and escape; cleanup is cooperative, not sandboxing.
- Setup/readiness/probe semantics are not implemented here. Only detected lifecycle failures receive typed controls; never coerce failure into candidate output.

## Exact ownership

U2 owns:

```text
internal/gitobj/**
internal/world/**
testkit/gitrepo/**
testkit/processfixture/**
tools/mutate-u2.mjs
docs/status/U2.md
docs/HANDOFF_MODE_C.md
```

It may add narrowly required constructors/interfaces to `internal/domain` or `internal/spec`, but must not weaken U1 invariants. Do not create `cmd/`, adapters, comparison orchestration, reducers, store, emitter, server, report, or web code.

## Implementation sequence

1. Implement repository fingerprint/object-format discovery and ref pinning. Store Git version and display provenance separately. Make moving-ref tests prove later changes do not alter a pinned identity.
2. Parse `ls-tree -rz --full-tree` as bytes, not newline text. Validate every entry and all cross-entry collision conditions before writing anything. Reject LFS pointer content explicitly. Preflight entry, single-blob, and total-byte budgets.
3. Stream blobs with `cat-file --batch` or an equivalently policy-free plumbing path. Independently recompute `blob <length>\0<bytes>` using SHA-1 or SHA-256 as declared by the repository, compare the requested OID, then compute the portable digest. Use exclusive creation, explicit `0644`/`0755`, `fsync` where the contract requires, and atomic publication only after the complete tree verifies.
4. Return a sorted immutable manifest with policy digest, blob/mode records, portable tree digest, target-filesystem facts, and typed rejection. Do not include absolute roots, time, or nonce in stable manifest identity.
5. Implement fresh attempt allocation and the execution state machine `ALLOCATED -> MATERIALIZING -> STARTING -> READY -> PROBING -> CAPTURING -> TEARING_DOWN -> FINALIZED`, while U2 exercises only applicable lifecycle phases. Illegal transitions and partial/cancelled paths retain evidence without becoming eligible.
6. Implement sparse environment construction, direct spawn, separate bounded stdout/stderr drains, timeout/cancel handling, process-group termination, and teardown classification. Output cap is `OUTPUT_LIMIT`, not a successful truncated behavior. Capture exit-versus-signal and orphan risk as typed lifecycle evidence for later adapters.
7. Record an attempt marker before spawn and a process-lifecycle receipt after wait/teardown. Prove unique owned roots and actual fixture invocation; a fresh nonce alone is insufficient.
8. Update status/handoff with exact Darwin claim, cleanup limitations, unsupported Git forms, and Linux as `UNRECEIPTED` unless natively run.

## Acceptance, adversarial fixtures, and mutants

Build generated Git repositories covering SHA-1 and, only when the installed Git supports it, SHA-256; moving refs; executable-bit distinction; replacement refs; filter and hook sentinels; archive attributes; missing/promisor objects with a network-attempt sentinel; symlink, gitlink, LFS, non-UTF-8, traversal-like, case/Unicode collision, oversized blob, and missing object refusals. Assert unsupported input leaves no published partial tree. Assert materialized bytes/modes rehash exactly and `.git` is absent.

Process fixtures must cover stdout/stderr at exact and over-limit boundaries, empty output, nonzero exit, signal death, timeout, cancellation, ordinary child/grandchild cleanup, TERM refusal requiring KILL, teardown/group-probe failure, environment leakage sentinels, private HOME/TMPDIR/state, and unique roots/attempt files across repeated runs. Test an escaping `setsid` fixture only to document the exclusion and clean it up explicitly; never turn that into a passing containment claim.

The mutation driver must fail closed when anchors are absent and require tests to kill at least: replacement refs enabled; lazy fetch allowed; blob rehash skipped; symlink accepted; `100755` collapsed into `100644`; path validation performed after writes; inherited environment passed through; shell spawn substituted; attempt marker created after spawn; output-limit mapped to success; and final group probe removed.

Run load-bearing checks through didrun, at minimum:

```text
didrun run -- go test ./internal/gitobj ./internal/world ./testkit/...
didrun run -- go test -race ./internal/gitobj ./internal/world
didrun run -- go test ./internal/world -run 'TestProcess|TestFresh|TestTeardown' -count=20
didrun run -- node tools/mutate-u2.mjs
didrun run -- git diff --check
```

Add a separate didrun-wrapped command that asserts no forbidden Git subcommand or network sentinel ran. Do not call cross-compilation native evidence.

Use `didrun show --session`; bind explicit claims with commands of the form `didrun claim tests-pass --label "U2 Git and Darwin worlds" --event N --path .` to exact unit/race/stress/adversarial/mutation events, plus a diff-check claim. Stage only U2 changes, inspect them, commit `feat: add exact Git and fresh Darwin worlds`, seal HEAD, and loop `NO_COLOR=1 didrun verify --strict` to exit 0. A failure requires a real fix, new didrun run, new claim, new fix commit if already sealed, reseal, and reverify. Never weaken tests or erase/relabel prior claims or commits.

## Stop/fallback and handoff

If materialization can trigger policy/network, unsupported entries can publish, blob verification is incomplete, ordinary descendants survive, byte caps deadlock, control results can appear eligible, or required mutants survive, stop. Remove the source-import or Darwin observation claim rather than hand-waving it. didrun malfunction is a logged bug with exact output and affected items marked `UNRECEIPTED`.

Update `docs/status/U2.md` and Mode C, then report commits, verbatim grades, strict exit, exact OS/Git/object formats exercised, refusal matrix, cleanup exclusions, remaining bets, and `P03-U3-CLI-OBSERVATION.md` as the only next prompt.
