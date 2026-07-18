# ADR 0002 — U2 admits bytes and processes only through measured edges

- **Status:** accepted for U2 implementation
- **Date:** 2026-07-14
- **Scope:** Git object admission, candidate binding, fresh attempt allocation, and Darwin process lifecycle
- **Later amendment:** Decision 17's initial positive-presence `EPERM` rule is current behavior from the P07B-C C1B pre-enrollment maintenance boundary; it does not relabel or extend the sealed historical U2 receipt.

## Context

U1 made semantic authority unavailable until a pure relationship could prove it. U2 is the first impure boundary: Countershape must decide whether repository bytes and a host process are the exact things named by those pure identities. Filesystem paths, display refs, successful exit codes, and random nonces are not sufficient evidence.

The substrate is intentionally smaller than a checkout engine or a sandbox. It runs trusted local code with the operator's full permissions and host network. It must not inherit Git policy, execute hooks or filters, hide missing objects behind a fetch, reuse a prepared world, or claim containment of hostile descendants.

## Decisions

### Repository and object identity

1. A private repository-handle receipt identifies one local object database operation. It binds canonical private paths, measured device/inode facts, object format, exact Git executable bytes/version, local-config bytes, and the closed policy profile. Moving, copying, or changing that substrate forces re-admission. This receipt is deliberately absent from portable tree identity, the selected-tree-set digest, `WorldPlan`, candidate keys, and stable manifests. Git alternates and partial/promisor repositories are rejected in v1. Pack-directory classification is paged and refuses above 8,192 entries; it never uses an unbounded directory read.
2. A display ref is resolved exactly once to a commit OID and then to a tree OID. All later reads use the pinned OIDs. Replacement-object interpretation, lazy promisor fetch, terminal prompting, helpers, and permitted network protocols are disabled.
3. Git is an absolute, resolved tool capability with a closed operation set. There is no public arbitrary-command runner. System/global configuration is neutralized; only the repository-local configuration needed to interpret repository extensions and object format remains visible.
4. Tree entries are parsed from NUL-delimited plumbing bytes. Only blob modes `100644` and `100755` are admitted. Symlinks, gitlinks, unknown modes, invalid UTF-8, control text, absolute/dot/traversal paths, `.git` aliases, prefix conflicts, LFS pointers, missing objects, and all configured budget excesses are typed refusals.
5. Each blob is independently rehashed as `blob <decimal-length>\0<bytes>` using the repository's SHA-1 or SHA-256 format and compared to the requested OID. A separate SHA-256 byte digest supplies portable identity. Skipping this check is an accepted mutant, not an optimization.

### Candidate-set and plan dependency

6. Ref pinning produces portable tree identities containing only object format plus exact commit/tree OIDs. A pre-plan `SelectedTreeSet` digest covers the sorted unique pinned-tree identity digests and count. It contains no repository receipt, policy, blob result, plan digest, or candidate key. The materialization-policy digest remains a separate `WorldPlan` field.
7. Inspection independently rehashes the selected commits, every reachable tree, and every blob; after rejecting duplicate portable executable trees, the exact selected set and policy may bind the compiled plan into opaque `CandidateExecutionBinding` values. This one-way sequence avoids a candidate-set/plan digest cycle:

   ```text
   repository receipt -> pinned portable trees -> selected-tree-set declaration
                                               -> WorldPlan
   selected trees + policy -> inspected trees -> opaque execution bindings
   ```

8. Duplicate executable portable trees are rejected in U2 even when they came from different refs, repositories, or object formats. Labels and provenance cannot manufacture distinct candidates.

### Materialization

9. Entry counts, individual sizes, total bytes, raw path rules, prefix conflicts, LFS detection, object hashes, and deterministic ordering are checked before candidate bytes are written. The private mode-0700 publication parent is classified with one bounded `ReadDir(1)` request; any preexisting entry, read/close error, or no-progress result refuses before staging. A private placeholder probe on the actual target volume detects case and Unicode aliases under that filesystem's behavior before publication. U2 uses canonical paths plus `Lstat`/exclusive creation; it does not claim resistance to a concurrent same-user path swap. Descriptor-relative `openat`/`O_NOFOLLOW` hardening remains a production-security tail.
10. Candidate bytes are streamed again into a private mode-0700 staging root using exclusive regular-file creation and explicit `0644`/`0755` modes. Files and required directories are synchronized, reopened, and rehashed. A canonical mode-0600 manifest is persisted beside the candidate. Manifest write, chmod, file-sync, close, and first parent-sync failures remove the created path, synchronize its parent again, and prove absence; failure to prove durable absence is typed `PUBLICATION_AMBIGUOUS`. Every later prepublication failure repeats that manifest-cleanup proof. Publication is one exclusive rename only after the complete staged tree and absence of `.git` have been verified. A post-rename parent-sync failure is rolled back only when the rollback rename and its second parent sync both succeed; either failure is typed `PUBLICATION_AMBIGUOUS`, never ordinary absence.
11. A manifest's stable identity contains the pinned-tree identity, object format, sorted path/mode/portable-blob records, portable tree digest, and policy digest. Absolute roots, allocation nonce, clocks, display provenance, physical repository receipt, and target-filesystem facts are excluded. Target volume and publication-path facts live only in the materialization receipt.

### Fresh worlds and tools

12. Every evidentiary attempt allocates new candidate, `HOME`, `TMPDIR`, state, and evidence roots. No cache, prepared root, clone, snapshot, reflink, reset hook, or prior-attempt artifact can satisfy this authority. The caller's structural instance nonce must be 1–256 UTF-8 bytes, already trim-stable, and free of Unicode control characters before allocation. An attempt marker is durably created before spawn.
13. Runtime tool names declared in a plan are resolved to explicit absolute executable capabilities before the attempt. Spawn replaces the bare `argv[0]` with that exact path. It never searches ambient `PATH`, invokes a shell, or accepts a generic environment map.
14. The child environment is rebuilt from nothing: runtime-owned fresh roots, a closed public-literal subset from the plan, fixed locale/time/color controls, and only explicitly provisioned secret slots in later units. U2 refuses required secrets and does not implement setup commands.

### Darwin lifecycle and bounded capture

15. U2 implements one-shot CLI process mechanics, not CLI observation semantics. The child becomes a new Darwin POSIX process-group leader. `stdout` and `stderr` are drained separately with independent exact byte budgets; a cap-plus-one observation is `OUTPUT_LIMIT`, never successful truncation.
16. Timeout, cancellation, overflow, start failure, and materialization failure are primary controls. A typed structural materialization refusal outranks coincident cancellation; phase cancellation is recorded only when the returned error matches the context sentinel. Cleanup findings are additive and cannot overwrite the primary cause. A closed diagnostic code retains the exact Git/world refusal or lifecycle edge in the process-receipt digest without admitting free-form host error text. Once multiple asynchronous events are ready, U2 applies the closed owner-observed priority `OUTPUT_LIMIT > CANCELLED > TIMEOUT > WAIT`; a wait selected on the prior owner turn is provisional until every higher-priority fact is repolled. This is deterministic process-owner arbitration, not a claim to recover kernel-causal timestamps.
17. Teardown resolves the originally measured group immediately before `TERM`. Observed absence skips the signal; clean observed presence alone authorizes it. Darwin's exact `(present=true, EPERM)` observation is retried for at most one quarter of the declared teardown budget, clipped to the overall teardown deadline. A later absence closes cleanly; a later clean presence authorizes `TERM`; persistent `EPERM` and every other probe error retain uncertainty and authorize no blind signal. After `TERM`, the same transient state is retried during bounded grace; persistent uncertainty cannot authorize blind `KILL`. The owner then waits for the direct child, completes bounded pipe drains, and performs a final group-existence probe. A successful direct-child wait is not by itself cleanup authority. Only nil wait error or ordinary `ExitError` completes the direct wait edge; any unexpected wait failure, incomplete drain, or final-probe uncertainty conservatively sets orphan risk even if the original PGID is later absent.
18. A descendant can call `setsid` or move itself into another process group and escape the owned group. Adversarial fixtures prove and explicitly clean these exclusions; Countershape never reports them as containment. U2 establishes cooperative cleanup for ordinary descendants only. Linux behavior remains `UNRECEIPTED` unless executed natively.

## Consequences

- The materializer must read admitted blobs twice: once before any candidate-path write and once into a fresh staging tree. The cost is accepted because bytes, not Git's exit status, are the authority.
- Repository relocation invalidates the private handle receipt and forces physical re-admission while portable tree/candidate identity survives. This is deliberate separation between source authority and cross-repository executable equivalence.
- A target filesystem that aliases two declared paths is refused rather than normalized. Countershape does not choose a winner.
- Trusted-local execution assumes no concurrent same-user mutation of admitted repositories, manifests, or candidate paths during the narrowed revalidation-to-spawn interval. U2 detects many changes but does not establish a hostile-filesystem boundary.
- Darwin exposes no atomic authority that binds a zero-signal group probe to the immediately following signal. The pre-TERM probe narrows accidental PGID reuse but cannot eliminate reuse between calls; every process receipt carries that residual exclusion explicitly.
- U2 process receipts remain edge evidence. They do not become `CapturedObservation`, `StableBatch`, divergence, or eligibility authority until an adapter unit interprets them.

## Exact mutation obligations

The U2 mutation gate freezes and kills twenty-five reviewed semantic edits: replacement refs; lazy fetch; alternate object authority; skipped streamed rehash; wrong SHA-256 OID width; symlink admission; executable-mode collapse; LFS admission; path validation after blob consumption; lexical-only collision detection; duplicate portable trees; destination overwrite; ambient environment inheritance; ambient `PATH` lookup; shell spawn; skipped marker gate; shared stream cap; overflow-as-success; direct-PID signaling; missing KILL escalation; drain-timeout-as-complete; skipped final group probe; late control overriding output; process escape promoted to containment; and physical/post-plan authority polluting the pre-plan digest. The exact IDs, tuples, named killing tests, and infrastructure-failure taxonomy are locked in `docs/status/U2.md` and `tools/mutate-u2.mjs`.

## Nonclaims

U2 does not establish sandboxing, confidentiality, malicious-code containment, network isolation, remote-repository acquisition, worktree semantics, Git filter compatibility, escaped-session cleanup, Linux lifecycle parity, semantic output equivalence, or contract correctness.
