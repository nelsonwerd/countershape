# U2 design red team: Git bytes and process cleanup need narrower authorities

- **Review target:** P02 / U2 Git-object materialization and fresh Darwin process substrate
- **Posture:** adversarial contract review before implementation authority is sealed
- **Date:** 2026-07-14
- **Verdict:** **PROCEED ONLY WITH THE LOCKS BELOW**
- **Confidence:** 8/10 in a bounded Git materializer; 7/10 in the Darwin process owner; 2/10 in any containment interpretation
- **Receipt state:** every U2 implementation claim is `UNRECEIPTED` at the time of this review

## Executive decision

U2 is feasible, but P02 currently compresses several different facts into words such as “repository fingerprint,” “verified tree,” “fresh,” and “cleanup.” That compression creates the easiest false-green path in the project. The implementation must keep four authorities separate:

1. a **local repository handle receipt** says which physical object database and Git executable were consulted during one operation;
2. a **portable pinned tree identity** says which object format, commit OID, and tree OID were selected;
3. a **materialization manifest** says which supported blob bytes and modes were written under one exact policy; and
4. a **process-lifecycle receipt** says what the Darwin owner observed while starting, draining, terminating, waiting, and probing one process group.

None of these establishes hostile-code containment. None may be reconstructed from a display ref, a caller-authored digest, a nonce, or a success boolean. U2 should stop rather than seal if any unsupported source can publish a partial root, if Git can consult replacements/promisors/alternates, if a cap can become truncated behavior, or if cleanup uncertainty can become eligible evidence.

## Implementation amendment after adversarial review

The implemented U2 reference profile deliberately narrows two recommendations from this review instead of pretending to satisfy them. First, topology reservation uses the actual target volume and exclusive placeholders, but canonical-path operations are not descriptor-relative and therefore do not resist a concurrent same-user path swap. Second, the process owner cannot recover a trustworthy cross-goroutine kernel-causal timestamp for cancellation, deadline, output, and wait readiness; it uses the fixed owner-observed priority `OUTPUT_LIMIT > CANCELLED > TIMEOUT > WAIT`. Both limits are recorded in ADR 0002, receipts, tests, and the nonclaim ledger. They are production-hardening work, not sealed capabilities.

A later independent pre-seal audit found six additional failure-edge defects; see `10-u2-seal-audit.md`. The implementation response bounds the pack-directory scan, makes manifest creation and every later prepublication failure prove durable manifest absence or return `MATERIALIZATION_PUBLICATION_AMBIGUOUS`, bounds structural instance nonces before allocation, retains closed detailed diagnostic codes in process receipts, treats incomplete waits/drains/probes as orphan risk, and probes the original PGID immediately before TERM. The last change narrows rather than eliminates PGID reuse: Darwin provides no atomic probe-and-signal capability, so the residual race remains an explicit receipt boundary and nonclaim.

## P0 and P1 traps

| Priority | Trap | False result | Locked disposition |
| --- | --- | --- | --- |
| **P0** | Physical repository fingerprint enters `TreeIdentity` or `CandidateExecutionKey` | identical clones produce different semantic candidates; absolute paths leak into stable identity | physical repository facts are receipt-only; portable source identity is object format + immutable OIDs |
| **P0** | `CandidateSetDigest` contains candidate keys, while candidate keys contain `WorldPlanDigest` | an unconstructable digest cycle or a placeholder digest receives authority | pre-plan tree-set digest contains only sorted portable tree identities; executable roster is a distinct post-plan projection |
| **P0** | replacement refs, promisor fetch, or alternates remain reachable | bytes other than the pinned primary object database silently enter the world | replacements disabled, promisor/partial repositories refused, alternates refused, any network sentinel activation is terminal |
| **P0** | object header/OID is trusted without independent streaming rehash | corrupt or substituted bytes receive Git identity | rehash `type SP size NUL bytes` with the declared Git algorithm and compare before publication |
| **P0** | target-filesystem aliases are checked only with string normalization | case/Unicode aliases overwrite or merge entries on the actual volume | reserve the entire topology with exclusive descriptor-relative creates on the target filesystem before writing blob bytes |
| **P0** | final path is visible before all blobs, modes, manifest, and directory topology verify | another stage can execute a partial candidate | private same-filesystem staging plus exclusive atomic rename; failure leaves destination absent |
| **P0** | output limit returns retained prefix as behavior | infrastructure failure becomes a candidate outcome | byte `limit + 1` yields `OUTPUT_LIMIT`; retained prefixes are diagnostic-only and ineligible |
| **P0** | PID is killed instead of the owned process group, or final probe is omitted | descendants contaminate later trials while teardown appears clean | verified PGID, TERM/grace/KILL to negative PGID, direct-child wait, bounded drains, final group probe |
| **P1** | Git SHA-1 and SHA-256 use one assumed OID width/hasher | SHA-256 repositories misparse or SHA-1 identities are mislabeled | discover exact storage format; closed 40/64-lowercase profiles; two explicit hash implementations |
| **P1** | `ls-tree` is parsed as lines or quoted text | newline, tab, escape, or invalid-byte names alter the roster | `-r -z --full-tree` byte parser; the first NUL is the only record delimiter |
| **P1** | stdout and stderr share one cap or one blocking reader | a quiet channel loses budget or a noisy channel deadlocks teardown | independent concurrent drains, independent caps, bounded discard after overflow |
| **P1** | cancellation races timeout/overflow through “first goroutine wins” | repeated runs record different primary controls | a single coordinator derives primary cause from recorded event facts and the precedence below |
| **P1** | `setsid` is used as if it strengthens cleanup | the direct child moves to a different session/process-group topology than the owner expects | U2 uses `setpgid`, never `setsid`; a descendant-created session is an explicit exclusion fixture |

## Locked source-identity graph

The only acyclic construction order is:

```text
RepositoryHandleReceipt                 (operation-local, nonportable)
  -> PinnedTreeIdentity[]               (portable object format + commit/tree OIDs)
  -> SelectedTreeSetDigest              (pre-plan, sorted unique tree identities)
  -> WorldPlanDigest                    (binds selected set + materialization policy + execution declaration)
  -> CandidateExecutionBinding[]        (tree + policy + actual plan + adapter/runner/projection)
  -> CandidateExecutionKey[]            (post-plan wire references)
  -> ExecutableRosterDigest             (post-plan, if a later artifact needs it)
```

`SelectedTreeSetDigest` is the value that the current `WorldPlan.CandidateSetDigest` must match. It is computed from a canonical sorted array of portable `PinnedTreeIdentityDigest` values and a count. It contains no display ref, branch, producer, repository path, remote URL, repository handle fingerprint, materialization root, `WorldPlanDigest`, or `CandidateExecutionKey`.

`PinnedTreeIdentity` binds:

```text
object_format = SHA1 | SHA256
commit_oid     = exact lowercase OID in that format
tree_oid       = exact lowercase OID in that format
```

The commit is provenance; the tree is executable content identity. Two slots with the same `(object_format, tree_oid)` are rejected before plan binding even if their commit OIDs differ. After materialization, duplicate `PortableTreeDigest` values are also rejected, which catches identical executable trees arriving through different Git object formats. Coalescing is not implemented in U2.

The plan’s declared candidate count must equal the exact selected-tree count. The supplied `CandidateSetDigest` must equal the derived `SelectedTreeSetDigest`. A generic digest that merely has valid `sha256:` syntax is not a candidate-set authority. A later post-plan roster digest must use a different type and domain separator; it must never be fed back into `WorldPlan`.

## Repository fingerprint semantics

Git has no universal repository UUID. A remote URL is mutable, optional, credential-bearing, and not identity. A filesystem path is clone-specific. Therefore **repository fingerprint must not be a portable semantic identity**.

U2 may create an opaque `RepositoryHandleReceipt` for anti-substitution within one operation. Its fingerprint is a domain-separated SHA-256 over measured local facts such as:

- canonical absolute Git common-directory and primary object-directory paths;
- device/inode identity for both directories on Darwin;
- discovered object format;
- absolute Git executable path, portable executable-byte digest, and exact version output; and
- the exact repository-policy profile version.

Those path and inode facts are private receipt data. They do not enter `TreeIdentity`, `SelectedTreeSetDigest`, `WorldPlan`, `CandidateExecutionBinding`, or a portable manifest digest. Re-discovery before each Git subprocess must match the open operation’s handle receipt. A mismatch aborts that operation. This is a TOCTOU consistency check in the trusted-local model, not proof against a malicious same-user process or compromised filesystem.

Display refs are provenance only. U2 accepts either an exact full lowercase OID or a closed full-ref spelling (`refs/heads/...` or `refs/tags/...`) that passes Git ref-name validation; arbitrary revision expressions are refused. Resolve once, require a commit, peel an annotated tag explicitly, derive its tree, and retain the immutable OIDs. A later moving-ref change cannot alter the pinned object IDs; observing the new ref creates a successor selection.

## Git invocation and policy refusal

Every Git subprocess uses one absolute, measured Git executable and a fresh environment built from zero. Repository discovery is separated from object reads. Object commands address the resolved Git directory directly; they do not rediscover through a working tree.

The fixed policy is:

- system and global configuration disabled;
- private `HOME`, `TMPDIR`, and `XDG_CONFIG_HOME`;
- `LANG=C`, `LC_ALL=C`, `TZ=UTC`, `NO_COLOR=1`;
- terminal and credential prompting disabled with rejecting askpass helpers;
- replacement interpretation disabled by the Git global option and `GIT_NO_REPLACE_OBJECTS=1`;
- lazy fetch disabled with `GIT_NO_LAZY_FETCH=1`;
- optional locks disabled;
- all protocols set to `never` for the object-read commands;
- no inherited `GIT_*`, proxy, credential, SSH-agent, cloud, or toolchain variables; and
- no checkout, switch, worktree, restore, archive, submodule, fetch, remote, hook, filter, or working-tree command.

Repository-local config is not silently treated as harmless. U2 preflights it under the same neutral outer configuration and refuses:

- `extensions.partialClone`, any promisor remote, or any partial-clone filter;
- a nonempty `objects/info/alternates` file;
- any alternate/object-directory environment mechanism (the sparse environment omits all of them);
- an unsupported repository extension or object format; and
- a Git-dir indirection or config uncertainty the implementation cannot classify.

Replacement refs may exist because the fixture must prove they are ignored. The resolved commit, tree, and blobs must be the unreplaced objects. Promisor/partial repositories and alternates are refused in U2 even when all currently requested objects happen to be local. That is intentionally stricter than “do not fetch”: it removes an unreceipted provenance branch from the reference instrument.

Network denial is not an OS security property here. A fixture must install transport/askpass/proxy/remote-helper sentinels and Git tracing, then prove none was invoked. The only allowed public statement is “the named commands did not invoke the named network sentinels.” `HOST_ALLOWED` remains the process network boundary.

## SHA-1 and SHA-256 object verification

Discover the repository’s **storage** object format before accepting any OID. Only exact `sha1` and `sha256` are supported. SHA-1 OIDs are 40 lowercase hex characters; SHA-256 OIDs are 64. Mixed-format compatibility output is not silently accepted as storage identity.

For selected commits, trees, and every blob, the cat-file protocol is parsed as bytes with bounded ASCII headers. The requested OID, returned OID, type, and nonnegative decimal size must match expectations. Missing, ambiguous, malformed, delta-leaked, or unexpected-type responses are typed refusals.

The verifier independently hashes:

```text
<type> + " " + base10(size) + NUL + exact_object_bytes
```

using SHA-1 or SHA-256 according to the discovered storage format. It reads exactly `size` bytes, requires the protocol terminator, rejects trailing/short content, and compares the computed lowercase OID to the requested OID with no abbreviation. Blob hashing and portable hashing occur in the same bounded stream:

- Git-format hash establishes agreement with the named Git object under Git’s format;
- `BlobPortableDigest = SHA-256(domain || mode || byte_count || exact_blob_bytes)` supplies Countershape’s portable blob identity; and
- `PortableTreeDigest` is derived from the canonical sorted list of raw path bytes, exact mode, byte count, and portable blob digest.

The mode is outside Git’s blob OID and therefore must enter the portable manifest/tree digest. Executable-bit changes must change `PortableTreeDigest`.

SHA-1 support is compatibility with SHA-1 Git identity, not a claim of SHA-1 collision resistance. The extra portable SHA-256 records the exact bytes Countershape consumed; it does not repair a malicious collision in Git history. A compromised Git binary or object database remains outside the threat boundary.

Native SHA-256 repository integration may be claimed only if the installed measured Git can create and read the fixture. Otherwise algorithm vectors may be unit-receipted while the native SHA-256 Git integration capability remains `UNRECEIPTED`.

## Raw tree paths and target-filesystem collisions

`ls-tree -r -z --full-tree` output is parsed as raw bytes. A record is exactly:

```text
mode SP type SP oid TAB path NUL
```

No C-style unquoting, newline splitting, locale conversion, or shell interpolation is permitted. The parser bounds header length, path length, entry count, and cumulative path bytes before allocating proportionally.

Before converting a path to a Go string, U2 requires valid UTF-8. Unicode is **not normalized for identity**. Split only on the literal `/` byte and reject:

- empty path or component, leading/trailing slash, repeated slash;
- `.` or `..` components;
- any NUL or invalid UTF-8;
- an absolute path;
- any component whose ASCII case-fold is `.git`;
- duplicate raw path bytes;
- a file/directory prefix conflict; and
- any path exceeding the closed component/path bounds.

Newlines, tabs, combining characters, and bidi controls are not made safe by lossy rewriting. If admitted, they remain exact bytes and every diagnostic uses an escaped byte representation. Future UI/report inertness is a separate obligation.

Case and Unicode aliasing are properties of the actual target volume. String lowercasing, NFC, or NFD alone is not an adequate oracle. After all raw lexical checks and budgets pass, U2 creates the entire empty directory/file topology in the private unpublished staging directory using descriptor-relative, no-follow operations and exclusive creation. Directory traversal is through already-open directory descriptors. If the actual filesystem maps two distinct raw names to one entry, the second exclusive reservation fails and the whole candidate is `PATH_COLLISION`. A file/directory prefix conflict fails likewise. Blob bytes are not written until the complete topology has been reserved successfully.

Only modes `100644` and `100755`, both with Git type `blob`, are accepted. Symbolic links (`120000`), gitlinks (`160000`), trees presented as files, unknown modes, and special files refuse the candidate. A blob beginning with the exact Git LFS pointer signature is refused before publication. Unsupported content is never omitted.

## Budgets and atomic publication

Metadata preflight happens before blob reads: exact entry count, supported modes/types, raw paths, duplicate OIDs, and declared size headers must fit the plan’s entry, single-blob, and total-byte budgets using overflow-checked arithmetic. Cat-file sizes are rechecked while streaming; a lying header cannot bypass the cap.

Publication uses this sequence:

1. allocate a private mode-0700 attempt directory and persist the attempt marker;
2. create an unpredictable private staging directory and an absent final materialization name under the same opened parent directory;
3. validate metadata and reserve the complete path topology;
4. stream, verify, write, `chmod` to exact `0644`/`0755`, and sync every regular file;
5. write and sync the canonical sorted manifest;
6. rehash every staged regular file through held/no-follow descriptors and verify the manifest;
7. sync created directories bottom-up and the staging root;
8. publish with Darwin’s exclusive no-replace atomic rename on the same filesystem; and
9. sync the parent directory, then return the published root and manifest receipt.

The final destination is never created early and is never overwritten. On any pre-rename error, close descriptors, remove only the unpublished staging name, and leave the final name absent. A crash may leave an unpublished staging directory; its name and missing publication receipt make it ineligible. Atomic visibility is claimable after a native receipt. Crash durability across hardware failure is not claimed merely because `fsync` calls returned.

The stable `MaterializationManifest` excludes absolute paths, device/inode, staging/final root spelling, time, nonce, PID, Git display ref, and repository handle fingerprint. It includes policy version/digest, pinned portable tree identity, object format, canonical sorted entry records, portable blob digests, exact modes/counts, and portable tree digest. Host and publication facts live in a separate receipt.

## Sparse candidate tool environment

Git’s environment and the candidate’s environment are different closed profiles. Candidate execution never inherits the parent process environment.

The candidate environment contains only:

- exact declared public entries already admitted by `WorldPlan` (`LANG=C`, `LC_ALL=C`, `TZ=UTC`, `NO_COLOR=1`, `NODE_NO_WARNINGS=1`, when present);
- runtime-owned absolute `HOME`, `TMPDIR`, `XDG_CACHE_HOME`, `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, and `XDG_STATE_HOME`, each beneath new private attempt-owned roots; and
- a narrowly named attempt-state directory variable only if it is part of the locked runner profile.

No inherited `PATH`, `PWD`, shell, proxy, credential, agent, package-manager, compiler, `DYLD_*`, `NODE_OPTIONS`, Git, cloud, or locale variable is allowed. The child’s cwd is set by the spawn API, not by trusting a `PWD` value.

`argv[0]` remains the bare logical tool name in `WorldPlan`, but U2 resolves it through an operator-supplied closed tool registry to one absolute executable. It never calls `LookPath` on the ambient environment. The exact executable path, portable byte digest, version output, and resolution profile are `InstanceMeasurements`/tool-receipt facts and may participate in envelope admission. The direct spawn receives the absolute executable separately from the logical argv. No shell and no `/usr/bin/env` wrapper are used.

The reference fixtures require no setup command, package installation, registry, external hostname, or inherited secret. If a candidate needs any of those, it is outside the U2 reference claim.

## Attempt allocation and authority handoff

An attempt marker and all owned root names are created before spawn. Creating a nonce in memory is insufficient. Allocation accepts the opaque `CandidateExecutionBinding` and the actual matching `WorldPlan`; it revalidates their U1 relationship. A parsed `CandidateExecutionKey`, copied digest fields, or a materialization manifest alone cannot allocate a `WorldInstance`.

U2 must not put roots, paths, device/inode, Git version, tool path/version, PID/PGID, clocks, exit status, stream counts, or teardown facts into the stable U1 `WorldInstance`. Those belong in edge receipts and the policy-complete `InstanceMeasurements` row. The attempt artifact digest is created before spawn and becomes one input to `WorldInstance`; the process receipt links back to both.

U2 drives the U1 attempt state machine rather than creating a parallel status enum. A lifecycle or materialization refusal records its detailed edge code, then maps to the existing closed control such as `MATERIALIZATION_ERROR`, `START_ERROR`, `CANCELLED`, `TIMEOUT`, `OUTPUT_LIMIT`, `ORPHAN_RISK`, or `TEARDOWN_ERROR`. Git-specific detail remains in the source/materialization receipt; it is not projected behavior.

## Darwin process-group lifecycle

The only claimable U2 owner is native Darwin. It uses direct `execve` semantics with `SysProcAttr.Setpgid=true` and `Pgid=0`. It does **not** set `Setsid`. Immediately after start, the owner records the direct child PID, queries the PGID, and requires `PGID == child PID` before any negative-PGID signal is allowed. PID/PGID values must be positive and pass defensive range checks.

The locked lifecycle is:

1. create manual stdout/stderr pipes and attempt receipt before spawn;
2. spawn the absolute executable with the exact sparse environment and materialized cwd;
3. close parent copies of write ends and start independent drain goroutines immediately;
4. run one direct-child waiter and one centralized event coordinator;
5. on normal completion or a terminal control, begin teardown while retaining the primary fact;
6. signal `TERM` to `-PGID` when the group still exists;
7. wait only the declared grace interval while drains continue;
8. signal `KILL` to `-PGID` if the group remains or its state is uncertain;
9. wait for the direct child within the teardown deadline;
10. wait for both drains to reach EOF within their bounded drain deadline, then close them;
11. probe `kill(-PGID, 0)` after wait/drain; and
12. classify `ESRCH` as no observed group, while `0`, `EPERM`, or any unexpected error is `ORPHAN_RISK`/teardown uncertainty.

The direct child’s status is recorded as exactly `EXITED(code)` or `SIGNALED(signal)`. A nonzero application exit is not intrinsically a U2 infrastructure error; later CLI eligibility decides what it means. A failed start has no assumed group and still produces a finalized typed control receipt.

No signal is ever sent to a negative ID until the owned group relationship was measured. No success path skips teardown because the direct child exited. Ordinary descendants can outlive the direct child while retaining its group and pipes.

Process groups are cooperative cleanup. A descendant can call `setsid`, create a new session/group, close or redirect its pipes, and escape the owner’s group probe. The hostile `setsid` fixture exists only to demonstrate this exclusion and must be explicitly located and killed by the test harness outside the product result. It must never be counted as a passing containment or descendant-cleanup case.

## Stdout/stderr limits and cancellation precedence

Stdout and stderr have independent limits from `WorldPlan`, independent buffers, independent total-seen counters, and concurrent readers. Exactly `N` bytes followed by EOF is complete. Reading byte `N+1` before teardown begins records overflow. The reader retains at most `N` bytes, switches to bounded discard/drain mode, and requests teardown. The retained prefix may appear only in private diagnostic evidence; it cannot be tagged `Present` or projected as behavior.

Counters use overflow-safe saturating arithmetic. Drain loops use fixed-size buffers and cannot allocate in proportion to candidate output. A process that fills one pipe while the other is quiet cannot block the other reader. Empty output is a valid complete channel when the process and teardown otherwise qualify.

Primary runtime control is derived centrally, not by whichever goroutine wins a `select`. The decision is:

1. a materialization/start failure established before execution dominates later cancellation;
2. an output overflow observed before teardown was requested is `OUTPUT_LIMIT`;
3. otherwise, an explicit cancellation request timestamped before the execution deadline is `CANCELLED`;
4. otherwise, expiration of the execution deadline is `TIMEOUT`;
5. otherwise, a detected pipe/wait/lifecycle failure receives its typed control;
6. direct-child exit or signal is application process status, not a control by itself; and
7. teardown/orphan findings are always appended and never overwrite the primary control.

Bytes observed only while draining after cancellation/timeout remain channel receipt facts; they do not retroactively change the already selected primary control. If both stdout and stderr crossed their caps before teardown, primary remains `OUTPUT_LIMIT` and both channel overflow facts are retained. A user cancellation cannot launder an already observed overflow, and a deadline cannot be relabeled cancellation by a late request.

U1 currently permits only teardown/orphan secondary controls. Any richer U2 channel-fact list must remain in the edge receipt rather than weakening that invariant.

## Required negative fixtures

### Git/source fixtures

- moving branch and annotated tag after pinning; original commit/tree remain selected;
- local replacement ref that would change the commit/tree/blob if replacement interpretation were enabled;
- promisor/partial-clone metadata, both with a local object and with a missing object; both refuse before network;
- nonempty `objects/info/alternates` pointing to an otherwise valid missing blob; refuse rather than consume it;
- network, remote-helper, askpass, filter, hook, archive-export, and smudge sentinels; none may execute;
- SHA-1 repository and native SHA-256 repository when the measured Git supports it;
- corrupt/short/long cat-file frames, wrong object type, wrong returned OID, and requested object missing;
- same tree under two commits and identical portable trees across distinct selections; reject duplicate executable content;
- exact `100644`/`100755` pair with identical blob bytes; manifest/tree identities must differ;
- `120000`, `160000`, malformed/unknown mode, LFS pointer, and special-object attempts;
- invalid UTF-8, absolute/dot/dotdot/empty components, `.git` aliases, prefix conflict, duplicate raw path, newline/tab path, and overlong path;
- case-only, composed/decomposed Unicode, and other actual-volume alias pairs, exercised on the staging filesystem;
- entry-count, per-blob, total-byte, integer-overflow, and late cat-file size mismatch cases; and
- failure at every staging step; final destination absent and no partial manifest accepted.

### Process fixtures

- empty stdout/stderr, exactly-at-limit output, limit-plus-one on each channel, and simultaneous overflow;
- one channel flooding while the other writes a final marker, proving concurrent drains;
- zero and nonzero exit, signal death, natural direct-child exit with live grandchild, and waiter error;
- timeout, cancellation before deadline, cancellation after deadline, overflow-before-cancel, and cancel-before-overflow-during-drain;
- child/grandchild ordinary group cleanup, TERM-cooperative exit, TERM refusal requiring KILL, inherited-pipe descendant, failed/uncertain final probe, and bounded drain timeout;
- `setsid` escape as an exclusion-only fixture with explicit out-of-band cleanup;
- inherited-environment sentinels for `PATH`, `NODE_OPTIONS`, proxies, credentials, agent sockets, `DYLD_*`, and package-manager variables;
- private `HOME`, `TMPDIR`, all XDG roots, cwd, exact resolved executable, and environment cardinality recorded by the fixture;
- two repeated runs proving different attempt files/root device paths/process lifecycles plus fixture-observed invocation, not merely different nonces; and
- cancellation/failure at every U1 attempt phase proving no eligible `FinalizedAttempt` can be projected.

## Exact U2 mutation contract

The mutation driver must use an immutable manifest of source-file hashes, apply one named edit at a time, run the exact killing test in a private copy, restore, and prove A/B/A source identity. Missing or multiply matching anchors are driver failures. A mutant killed by compilation or an unrelated assertion does not count unless that is the intended invariant.

At minimum these mutants must die:

1. enable replacement refs;
2. remove `GIT_NO_LAZY_FETCH` / permit promisor fetch;
3. accept an alternate object directory;
4. skip streamed Git-object rehash;
5. hard-code SHA-1 or accept the wrong OID width;
6. accept symlink mode `120000`;
7. collapse `100755` into `100644`;
8. accept an LFS pointer as an ordinary blob;
9. validate paths only after blob bytes are written;
10. replace target-filesystem exclusive reservation with lexical lowercasing only;
11. omit the duplicate portable-tree refusal;
12. publish staging before final verification or allow destination overwrite;
13. inherit the parent environment;
14. resolve the executable through ambient `PATH`;
15. substitute shell execution for direct spawn;
16. create the attempt marker after spawn;
17. share one stdout/stderr cap;
18. map output overflow to successful truncated output;
19. signal the direct PID rather than the negative PGID;
20. remove TERM-to-KILL escalation;
21. remove the bounded direct-child wait or bounded drains;
22. remove the final process-group probe;
23. let cancellation overwrite an already observed output overflow or elapsed timeout;
24. treat `setsid` escape as clean group containment; and
25. include `WorldPlanDigest`/candidate keys in the pre-plan candidate-set digest.

P02’s original eleven mutants are a floor, not the U2 gate. The driver’s own self-tests must prove missing-anchor, duplicate-anchor, no-op, wrong-test, nonrestoration, tool failure, timeout, and source-manifest drift all fail closed.

## Exact receipt expectations

All load-bearing checks must run through root-serialized didrun using fixed absolute tools and an `env -i` profile. No ambient `GOFLAGS`, `GOENV`, `GOWORK`, `GOTOOLCHAIN`, `NODE_OPTIONS`, `NODE_PATH`, Git config, proxy, or credential variable may enter a receipt. Ordinary diagnostic successes from a dirtier environment do not qualify.

The future U2 seal requires separate successful didrun events for:

1. focused Git-object/materializer unit and hostile-fixture tests;
2. focused Darwin world/process tests;
3. race tests for `internal/gitobj` and `internal/world`;
4. repeated process lifecycle/teardown stress (`-count=20` or stronger) without cached results;
5. SHA-1 native repository integration;
6. SHA-256 native repository integration, or an explicit `UNRECEIPTED` capability if the measured Git cannot create it;
7. replacement/promisor/alternate/filter/hook/network sentinel assertions;
8. exact/over-limit stdout and stderr boundary tests plus cancellation-precedence tests;
9. path/case/Unicode collision fixtures on the actual measured target filesystem;
10. atomic-publication fault injection proving the destination remains absent;
11. mutation-driver self-tests;
12. the full U2 mutation matrix with every required mutant killed for its intended reason;
13. `go vet` and the repository boundary/import check under the same sparse toolchain;
14. `git diff --check`; and
15. a final native-environment fact event recording Darwin version/architecture, filesystem format and case-sensitivity behavior, Git path/version/object-format support, Go path/version, and Node path/version used by mutation tooling.

Claims must bind to the smallest exact events. “U2 passes” is too broad. At minimum, Git selection/materialization, SHA-1, SHA-256 (if exercised), Darwin fresh-attempt allocation, process cleanup, cap/control taxonomy, forbidden-command/network sentinels, race/stress, mutation completeness, and diff cleanliness receive distinct claims. Their grades are copied verbatim. A successful unit test cannot receipt native SHA-256 Git support, filesystem alias behavior, Darwin cleanup stress, or absence of network unless that exact path ran.

Before the U2 commit, inspect the didrun session for sparse command environments, intended test selection, noncached execution, and the exact tree. Then claim, commit, seal, and loop `NO_COLOR=1 didrun verify --strict` until exit zero. A failing receipt is repaired with a new event and claim; old failure history remains. The final handoff maps every U2 capability to its exact verbatim grade or `UNRECEIPTED`.

## Seal blockers and claim limits

U2 is not shippable if any of these remains true:

- repository-local policy, replacement, promisor, or alternate behavior is ambiguous;
- a selected commit/tree/blob is not independently rehashed in its declared object format;
- a SHA-1 result is described as collision-resistant;
- any unsupported entry is omitted while the remainder executes;
- distinct raw paths can alias on the measured target filesystem without refusal;
- blob bytes are written before full topology reservation or final output appears before complete verification;
- candidate-set construction depends on `WorldPlanDigest` or candidate keys;
- the candidate inherits ambient variables or uses ambient executable lookup;
- attempt allocation can proceed from a key/digest copy rather than the binding plus actual plan;
- byte overflow, timeout, cancellation, wait, drain, or teardown uncertainty can become behavior;
- ordinary descendants survive a passing cleanup fixture;
- the `setsid` fixture is counted as containment evidence;
- a required mutant survives or dies for an unrelated reason; or
- strict didrun verification is nonzero.

Even after a green seal, the honest claim is narrow: on the exact receipted Darwin/Git/filesystem/toolchain profile, the named fixtures selected immutable Git objects, materialized only verified regular/executable blobs into unpublished private staging before atomic publication, constructed sparse fresh attempts, bounded retained output, and returned typed cleanup controls for the exercised failures. It does not establish sandboxing, blocked network, safety for untrusted code, complete descendant containment, crash-proof durability, arbitrary-repository compatibility, Linux behavior, or production security.
