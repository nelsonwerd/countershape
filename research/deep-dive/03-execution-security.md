# Deep dive lane 3: execution, security, and process lifecycle

- **Reviewer posture:** adversarial implementation and threat-model review
- **Date:** 2026-07-14
- **Target claim:** a trusted-local Countershape v1 can materialize and execute 2-4 exact committed Git trees honestly on macOS and Linux
- **Verdict:** **MODIFY, THEN PROCEED**
- **Overall confidence:** **7/10** after the modifications below; **3/10** if “exact tree,” “isolated,” or “reset” remain undefined

## Executive finding

Countershape can honestly compare 2-4 committed candidates, but only if it stops using one phrase—“exact committed tree”—for three different things:

1. **Tree identity:** the immutable Git commit and tree object IDs.
2. **Materialized source:** the supported tree entries written byte-for-byte into a disposable directory under an explicit materializer policy.
3. **Execution world:** materialized source plus platform, toolchain, dependency setup, environment, ports, clock policy, and mutable runtime state.

Git can answer the first question exactly. A deliberately small materializer can answer the second for a declared subset. Countershape can only fingerprint and control parts of the third. It cannot make arbitrary local code safe merely by placing it under a temporary directory.

The strongest bounded implementation is therefore not `git worktree`, `git archive | tar`, or `git clone`. It is a filter-free Git-object materializer built on `git ls-tree -rz` and `git cat-file --batch`, followed by a fresh-world-per-trial runner. It resolves each ref once, rejects unsupported tree entries instead of guessing, uses no shell, gives every candidate a private `HOME` and `TMPDIR`, defaults to sequential execution, and labels host networking as enabled. On POSIX systems, it owns an ordinary process tree through a new process group and escalates `SIGTERM` to `SIGKILL`, while expressly declining to contain a child that deliberately daemonizes into a new session.

That yields a defensible v1 for trusted, self-contained fixtures and cooperative local applications. It does **not** yield an untrusted-PR sandbox, network denial, filesystem confinement, complete descendant reaping, dependency reproducibility, or multi-user localhost service. Those remain human-gated hardening work.

## Primary-source facts that change the design

### Git porcelain is not a neutral byte copier

`git worktree add` creates a linked working tree that shares repository data and administrative state. Git's own manual says multiple checkout remains experimental and submodule support is incomplete. More importantly, checkout is policy-bearing: Git attributes may transform blob contents through text/ident/smudge filters, and an external smudge command can run during checkout. A repository's `post-checkout` hook also runs after `git worktree add` unless `--no-checkout` is used. A worktree is convenient developer UX, but it is the wrong correctness primitive for Countershape's supposedly filter-free candidate source. Sources: [git-worktree](https://git-scm.com/docs/git-worktree.html), [gitattributes checkout filters](https://git-scm.com/docs/gitattributes), and [githooks post-checkout](https://git-scm.com/docs/githooks/2.46.0.html).

`git archive` is not exact either. `export-ignore` intentionally omits entries and `export-subst` rewrites placeholders. These attributes are taken from the archived tree by default. An archive is therefore a release projection, not a byte-faithful tree materialization. Source: [git-archive and archive attributes](https://git-scm.com/docs/git-archive.html).

Git plumbing exposes the needed lower-level truth. `git ls-tree -z` emits verbatim NUL-terminated paths plus object modes/types/IDs; `git cat-file` returns raw uncompressed blob contents and supports batch operation. Git trees encode regular files, executable files, symlinks, trees, and submodule gitlinks; Git only tracks the executable permission bit for regular files. Sources: [git-ls-tree](https://git-scm.com/docs/git-ls-tree.html), [git-cat-file](https://git-scm.com/docs/git-cat-file.html), and the [Git data model](https://git-scm.com/docs/gitdatamodel.html).

Submodules and LFS prove why tree identity and execution world must remain separate. A submodule tree entry stores a commit object ID, not the submodule's file contents. Git LFS stores a small pointer in Git and the large payload elsewhere, commonly on a remote server. Silently initializing submodules or hydrating LFS would add repositories, credentials, network, and mutable local configuration that the parent tree ID does not capture. Sources: [gitsubmodules](https://git-scm.com/docs/gitsubmodules) and [Git LFS](https://git-lfs.com/).

### “Temporary directory” is not a sandbox

A child process runs with the user's OS identity. A changed `cwd`, sparse environment, and private temp root reduce accidental contamination; they do not prevent the child from reading `~/.ssh`, opening `/etc`, writing another repository, binding a public interface, forking indefinitely, or making network requests. There is no portable Node API that applies a macOS-and-Linux filesystem/network sandbox. Linux namespaces/cgroups/seccomp and macOS sandbox/VM mechanisms are different products and require their own threat review.

Dependency setup is code execution too. Current npm documentation says `npm ci` removes an existing `node_modules` and installs from a lockfile, but install lifecycle scripts run unless policy disables or approves them; npm's current script controls explicitly describe unrestricted scripts as dangerous. A frozen lockfile improves dependency selection, not host containment or offline reproducibility. Sources: [npm ci](https://docs.npmjs.com/cli/v11/commands/npm-ci/) and [npm install-script policy](https://docs.npmjs.com/cli/v11/commands/npm-install-scripts/).

### POSIX process groups help, but do not form a containment boundary

Node documents that `spawn(..., { detached: true })` makes the child leader of a new process group and session on non-Windows platforms. POSIX `kill` semantics allow a negative PID to signal the matching process group on Linux and macOS. Node's AbortSignal and timeout options signal the spawned child; they do not promise descendant-tree cleanup. A descendant can also call `setsid`, enter a different group/session, and escape a group kill. Sources: [Node child_process](https://nodejs.org/api/child_process.html), [Node process.kill](https://nodejs.org/api/process.html), [Linux kill(2)](https://man7.org/linux/man-pages/man2/kill.2.html), and [macOS kill(2)](https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man2/kill.2.html).

### Loopback binding is necessary, not sufficient

A browser can emit cross-origin writes even when same-origin policy prevents the attacking page from reading the response. MDN recommends an unguessable CSRF token for cross-origin write protection. Host headers are attacker-controlled unless checked; OWASP documents the risks of trusting them. Therefore a mutable local comparison studio still needs a session token, exact Host/Origin policy, no CORS, and injection-safe rendering even when it binds only to `127.0.0.1`. Sources: [MDN same-origin policy](https://developer.mozilla.org/en-US/docs/Web/Security/Defenses/Same-origin_policy) and [OWASP Host header testing](https://owasp.org/www-project-web-security-testing-guide/v42/4-Web_Application_Security_Testing/07-Input_Validation_Testing/17-Testing_for_Host_Header_Injection).

## Bounded v1 threat model

### Assets

- The user's source repository, refs, object database, and visible working tree.
- Files and credentials elsewhere in the user's account.
- Candidate integrity: which exact blobs and modes were tested.
- Observation integrity: raw captured bytes, projections, partitions, shrink attempts, and rulings.
- Decision integrity: a browser page must not forge or overwrite a Choicepoint ruling.
- Availability: Countershape should stop cooperative descendants, cap output, and clean only roots it owns.
- Confidentiality of captured output and exported reports.

### Trusted

- One local operator and the Countershape installation.
- Candidate repositories and explicit setup/start commands are **trusted not to be intentionally hostile**. They may still be buggy, hang, leak output, ignore signals, or write unexpected files.
- Built-in CLI and HTTP adapters.
- Git and the explicitly selected runtime/toolchain binaries.

### Untrusted or malformed even in trusted-local mode

- Git path bytes, symlink targets, modes, LFS pointer files, and gitlinks until validated.
- Candidate stdout/stderr/HTTP bodies and headers.
- Browser requests arriving at the loopback server.
- Report data rendered into HTML.
- Adapter/plugin messages if an extension protocol is later enabled.
- Dependency packages and install scripts are outside the trusted boundary unless the operator separately reviews and approves them.

### Defended in v1

- Ref movement after study creation, checkout filters/hooks, tool-authored path traversal, unsafe symlink materialization, case-collision aliasing, accidental secret-environment inheritance, predictable temp paths, ordinary child/grandchild leaks, infinite output, simple timeout hangs, cross-candidate state/port reuse, cross-origin browser writes, Host-header confusion, and HTML/terminal injection.

### Explicitly not defended in v1

- A malicious candidate, dependency, adapter, or plugin.
- Host filesystem reads/writes performed by candidate code, network access/exfiltration, fork bombs, CPU/memory/disk exhaustion beyond best-effort budgets, `setsid`/double-fork daemon escape, kernel exploits, debugger/ptrace attacks, or attacks by another process running as the same user.
- Power loss, runner crash recovery, remote/multi-user operation, or hostile pull-request execution.
- Identical behavior across macOS and Linux. Cross-platform support means the same bounded protocol runs on both and records the platform; it does not mean their worlds are equivalent.

The UI and CLI must show this sentence before the first executable command: **“Countershape will run trusted repository code with your user permissions and host network access. The temporary directory is for repeatability, not security isolation.”**

## Recommended execution architecture

### 1. Resolve immutable identities once

For each user-provided ref, resolve and persist:

```text
repository identity
object format (sha1 or sha256)
commit oid
tree oid
ref spelling (display only)
Git version
```

All later operations address the commit/tree OIDs, never the mutable ref. Ref movement is informative metadata, not a reason to silently change the candidate. A missing/pruned object is `MATERIALIZATION_ERROR`, not candidate behavior.

### 2. Materialize through Git objects, not checkout/archive

Use `git ls-tree -rz --full-tree <tree>` to enumerate entries and `git cat-file --batch` to stream raw blobs. Do not pass `--filters`, do not run a checkout, and do not place `.git` in the candidate root.

The first implementation should support only:

- `100644` regular blobs;
- `100755` executable blobs, followed by an explicit chmod and mode check; and
- `120000` symlinks only when the target is relative and lexically remains inside the candidate root.

Reject `160000` gitlinks as `UNCOMPARABLE_SUBMODULE`. Detect the canonical Git LFS pointer header and reject it as `UNCOMPARABLE_LFS_POINTER` by default. Hydration can later become an opt-in expanded-world adapter whose LFS object IDs, local availability, network use, and git-lfs version join the world manifest.

Parse paths as bytes, require valid UTF-8 for v1, reject NUL/control characters, absolute paths, empty/`.`/`..` components, duplicate entries, and platform-unrepresentable names. Before writing, calculate every destination from the owned root and verify containment. Create ordinary directories/files first and symlinks last, so the materializer never traverses a candidate symlink while writing. Detect case-fold and Unicode-normalization collisions on the target filesystem; a tree containing both `A.ts` and `a.ts` must become `UNCOMPARABLE_FILESYSTEM` on a case-insensitive volume, not silently collapse.

After materialization, stream-hash every regular file back to its blob OID equivalent, verify executable bits, lstat links without following them, and emit a `MaterializationManifest`. Git does not version mtimes, ownership, extended attributes, ACLs, directory entries, or filesystem case semantics. Set a deterministic mtime policy where practical and include that policy—plus OS, architecture, filesystem sensitivity probe, umask, and materializer version—in the world identity.

This supports the honest sentence: **“The supported Git blobs and modes were materialized byte-for-byte from tree X under materializer policy Y.”** It does not support “this directory is the only possible checkout of X.”

### 3. Make execution-world identity first-class

The existing semantic cache-key invariant should become a structured `WorldIdentity`:

```text
candidate commit/tree
materialization manifest digest
OS/kernel/arch and filesystem policy
Countershape/adapter/materializer versions
absolute toolchain binary paths and versions
setup argv plus setup receipt/result
environment policy and nonsecret values
secret-presence tokens, never raw secret values
network mode (HOST_ALLOWED in v1)
reset strategy
port allocation and readiness policy
clock/locale/timezone policy
stimulus, normalizer, comparator, reducer versions
```

No `STABLE` partition or cache hit may cross differing world identities. Similar platform fingerprints can be displayed together, but equality must be exact over the declared fields.

### 4. Default to no setup; make setup loud

The two reference demonstrations should be self-contained Node programs with no package installation. This proves Countershape's semantics without smuggling registry state into its correctness story.

For imported repositories, setup is an explicit argv array shown before execution, never a shell string and never auto-inferred. Default environment:

- private mode-0700 `HOME`, `TMPDIR`, and XDG/cache directories per trial;
- a declared minimal `PATH` or absolute executable paths;
- stable `TZ`, locale, `CI=1`, and `NO_COLOR=1` where supported;
- no inherited cloud tokens, `SSH_AUTH_SOCK`, `NODE_OPTIONS`, package-manager auth, Git credential variables, or arbitrary parent environment;
- opt-in `passEnv` names with a loud secret warning.

Do not persist opted-in secret values in the world manifest. A per-study secret-presence identifier can show that two trials used the same operator-provided slot, but it must not be presented as independently reproducible evidence.

Setup runs inside the same threat boundary as the candidate. Network is **allowed**, not “unknown” or “best effort blocked.” If `npm ci` is offered as a helper, display whether lifecycle scripts are disabled/approved and record the exact npm version and flags. A setup failure is `SETUP_ERROR`; it never normalizes to an application response.

### 5. Fresh worlds are the correctness default

Every repeated trial and shrink attempt that contributes to a stable partition should start from a fresh materialization and repeat the declared setup. For HTTP, start a fresh service for each whole stimulus/action sequence, not necessarily each request inside the sequence. Give every world its own root, HOME, temp directory, state directory, and port.

Copy-on-write snapshots, prepared `node_modules`, container layers, and reset hooks are performance features, not the reference truth path. Defer them until each has a content manifest and cross-contamination tests. If a future fast mode reuses a prepared world, label it `DECLARED_RESET` and forbid it from sharing cache entries with `FRESH_WORLD` results.

Run candidates sequentially by default. Parallel execution changes CPU scheduling, port pressure, shared service contention, and timeout behavior. Parallel mode can be opt-in and must join the world identity. Randomize or balance candidate order across repetitions so warm caches and first-run work do not systematically favor one candidate.

### 6. Own cooperative processes on macOS/Linux

Spawn commands directly with `shell: false`, explicit argv, explicit cwd/env, piped stdio, and `detached: true` on POSIX. Treat the resulting PID as the process-group ID. On completion, timeout, cancellation, readiness failure, or output overflow:

1. stop accepting new observations;
2. signal `SIGTERM` to `-pgid`;
3. drain bounded pipes during a short grace period;
4. signal `SIGKILL` to `-pgid` if anything remains;
5. wait for the direct child and test group existence with signal 0;
6. record exit code, signal, deadlines, escalation, final group probe, and partial-output digests.

Do not rely on Node's timeout/AbortSignal alone, and do not call `unref()`. A leader may exit while a grandchild keeps stdout open, so completion needs its own wall-clock deadline independent of the child's `exit` event and stream closure.

If the final group probe still finds members, classify the trial `ORPHAN_RISK` and stop the study; do not emit a behavioral verdict. This catches ordinary mistakes. It cannot find a descendant that deliberately created another session, and the report must say so. Linux cgroups could later improve Linux-only ownership; they would not justify the same macOS guarantee.

### 7. Bound bytes, not lines or decoded strings

Stream stdout and stderr independently as bytes. Hash while draining, retain a configurable head/tail sample, and cap both per-stream and combined bytes. Once the cap is crossed, keep draining or terminate the group so the child cannot deadlock on a full pipe; mark `OUTPUT_LIMIT`. Never buffer an unbounded body or parse JSON before applying the byte cap.

Raw bytes are the observation. Text normalization requires declared UTF-8 validity/replacement behavior; binary results use base64 or a typed binary comparator. Terminal views strip or visibly encode ANSI/OSC/control sequences. HTML always escapes candidate content and never inserts it with `innerHTML`.

### 8. Make port/readiness behavior observable

Allocate a distinct loopback port per candidate world. The common “bind port 0, close it, then give the number to a child” method has a race; generic programs rarely accept an inherited listening socket. Accept that race, retry only on a diagnosed bind collision, and record the attempts. Do not reinterpret a start timeout or fixed-port collision as a behavioral divergence.

Pass `HOST=127.0.0.1` and the chosen port to cooperative fixtures, but acknowledge that trusted arbitrary code may ignore them and bind `0.0.0.0`. Readiness is an explicit bounded probe. Stop the process group on readiness failure before beginning another candidate.

### 9. Secure the local comparison studio as a mutable app

- Bind only `127.0.0.1`, never `0.0.0.0`, and generate a fresh high-entropy session token.
- Open `http://127.0.0.1:<port>/#<token>`; the app moves the token to `sessionStorage` and removes it from the visible URL. Fragments are not sent in HTTP requests.
- Require `Authorization: Bearer` on every API request, including reads; require JSON and the exact current Choicepoint digest on writes.
- Reject any `Host` except the literal bound IP/port and any present `Origin` except the exact studio origin. Emit no permissive CORS headers.
- Add a CSRF nonce for mutations as defense in depth. Use atomic compare-and-swap semantics so a stale tab cannot overwrite a later ruling.
- Serve bundled assets only. Use a restrictive CSP, `frame-ancestors 'none'`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, and `Cross-Origin-Resource-Policy: same-origin`.
- Never map arbitrary URL paths to candidate filesystem paths.

This is single-operator defense in depth, not authentication against another process owned by the same user.

### 10. Separate local raw evidence from shareable reports

Store study directories mode 0700 and files mode 0600. The local raw store can preserve exact bytes and digests because the trusted operator needs replayable evidence. The default self-contained HTML report must include escaped/redacted projections, digests, sizes, truncation state, and links/names for local raw artifacts—not raw bodies by default.

`report --include-raw` should require explicit confirmation and list the selected artifacts. Structured redaction rules (headers/JSON keys/env values plus declared patterns) are versioned and applied on export. The report states that redacted evidence cannot reconstruct the raw observation. Do not claim “raw” after redaction, do not print raw bytes to a terminal, and do not embed candidate strings in executable script contexts.

### 11. Defer third-party plugins until the built-ins prove the kernel

A loaded JavaScript module is arbitrary code in Countershape's process. Do not auto-load `countershape.config.js`, repository plugins, package-manager plugins, or editor extensions in the golden path. Configuration should be inert schema-validated data.

If an adapter protocol is added in R5, use an explicitly selected executable, not dynamic import. Give it a version/capability handshake, NUL-safe or length-prefixed frames, maximum frame count/bytes, schema rejection, deadlines, and no authority to choose host paths. Run it under the same process-group and environment policy as candidate code. State plainly that a plugin is trusted executable code. The protocol transports typed stimuli/observations only; it must not become a Wake-like agent transport, supervisor, or effect broker.

## Severity-ranked findings

| Severity | Finding | Consequence if unchanged | Required disposition |
| --- | --- | --- | --- |
| **Critical** | “Exact committed tree” conflates Git identity, checkout projection, and mutable execution world. | False reproducibility and misleading cache hits. | Split `TreeIdentity`, `MaterializationManifest`, and `WorldIdentity`; reject unsupported trees. |
| **Critical** | Temp roots/environment scrubbing do not sandbox local code or network. | Users may run hostile code believing it is confined. | Trusted-code consent; label `HOST_ALLOWED`; no hostile-PR claim. |
| **High** | Worktree/archive paths can apply filters, hooks, archive attributes, or incomplete submodule behavior. | Candidate bytes differ from tree or execute host config. | Use Git plumbing materializer; no checkout/archive. |
| **High** | Reusing prepared roots or reset hooks can fabricate stable partitions through shared state. | False divergence or false agreement. | Fresh materialization/setup for every evidentiary trial. |
| **High** | Direct-child timeout leaves grandchildren and held pipes. | Orphans contaminate later trials and hang the runner. | POSIX process group, escalation, deadline, final probe, `ORPHAN_RISK`. |
| **High** | Mutable localhost UI without token/Host/Origin checks is cross-origin writable. | Forged rulings or evidence disclosure. | Session bearer, CSRF, literal Host, no CORS, CSP, CAS writes. |
| **High** | Raw outputs and reports can contain credentials or active payloads. | Local secret leak or report XSS. | Sparse env, private files, export redaction, escaped static report. |
| **Medium** | Submodule/LFS hydration silently expands the candidate beyond its tree. | Network/credential drift and non-reproducible worlds. | Reject by default; explicit future expanded-world adapters. |
| **Medium** | Port allocation and service readiness are racy. | Infrastructure noise mislabeled as branch behavior. | Unique ports, diagnosed retry, explicit readiness state. |
| **Medium** | Parallel execution introduces contention and shared-global effects. | Order-dependent partitions and flaky timeouts. | Sequential default; parallelism joins world identity. |
| **Medium** | Unbounded/decoded output can exhaust memory or inject terminals/HTML. | Runner DoS and unsafe evidence views. | Byte caps, streaming hash, termination, typed encoding/escaping. |
| **Medium** | In-process plugin loading destroys the boundary. | Plugin compromise of all studies and overlap with Wake. | Defer; executable protocol only with explicit trust. |

## Adversarial test suite required before R1 is green

### Git/materialization

1. Resolve a moving branch, move it, and prove the stored commit/tree remain the candidate.
2. A repository with `.gitattributes` smudge, `export-ignore`, and `export-subst` proves no filter/hook/archive policy runs.
3. Regular and executable blobs round-trip to their object IDs and modes.
4. Internal relative symlink succeeds; absolute, escaping, chained-escape, and symlink-parent writes reject before output.
5. Gitlink returns `UNCOMPARABLE_SUBMODULE`; canonical LFS pointer returns `UNCOMPARABLE_LFS_POINTER` without network.
6. `A.ts`/`a.ts`, Unicode-normalization collision, newline/control path, invalid UTF-8 path, duplicate path, and `..` component all reject deterministically where unrepresentable.
7. Cleanup refuses a directory without the owned-root nonce and never follows a candidate symlink.

### Environment/world/reset

8. A fixture prints every environment key; seeded fake AWS/npm/SSH/Git tokens in the parent never appear.
9. HOME/TMP/state/port values differ across every candidate and repeated trial.
10. Candidate A writes state with a common filename; B and the next A trial cannot observe it.
11. A setup command hangs, fails, and mutates source; each becomes setup evidence, never an application outcome.
12. Platform/toolchain/setup/reset differences change the world digest and prevent cache reuse.
13. Parallel mode is absent from the reference path or demonstrably changes the world identity.

### Process lifecycle and output

14. Direct child exits normally while a grandchild holds stdout; wall deadline still completes cleanup.
15. Child and grandchild ignore `SIGTERM`; group escalation removes both and records `SIGKILL`.
16. A descendant calls `setsid`; test documents the expected limitation and produces `ORPHAN_RISK` when observable—never a containment claim.
17. Timeout during setup, readiness, request, and teardown all kill the owned group.
18. Infinite stdout/stderr stays under bounded resident memory, terminates, and becomes `OUTPUT_LIMIT`.
19. Invalid UTF-8, NULs, ANSI/OSC sequences, `</script>`, oversized JSON, and gzip bombs render inertly or reject before parsing.
20. Port theft between allocation and bind retries as infrastructure failure; it never changes a behavioral partition.

### Local API/report/plugin

21. Requests with missing/wrong bearer, hostile Host, hostile/`null` Origin on mutation, form content type, missing CSRF, and stale Choicepoint digest all reject.
22. Studio assets expose no CORS wildcard, external script/font, candidate path traversal, or inline candidate markup; CSP/security headers are asserted.
23. Report export removes configured fake secrets, records redaction metadata, escapes active content, and does not include raw bodies by default.
24. If plugins exist, malformed version, unknown capability, oversized frame, excessive frames, slow response, path request, and child process leak all fail closed.

These tests should be load-bearing didrun-recorded commands in the build phase. The escaped-session limitation is a documented negative capability, not a test to relabel as passed containment.

## Ground-truth tally

### Confirmed by primary documentation: 12

1. Worktrees are linked to repository state and submodule multiple-checkout support is incomplete.
2. Worktree add can invoke `post-checkout`.
3. Checkout filters can transform stored blobs through external commands.
4. Archive attributes can omit or rewrite tree content.
5. Git plumbing can enumerate NUL-terminated paths/modes/OIDs and return raw blobs.
6. Submodule entries identify another commit, not an embedded filesystem.
7. Git LFS stores pointer blobs while payloads live outside ordinary Git objects.
8. Node inherits `process.env` unless `env` is explicitly supplied.
9. Node detached POSIX children lead a new process group/session.
10. Negative-PID signaling targets a process group on macOS/Linux.
11. npm installation may execute lifecycle scripts unless disabled/approved.
12. Same-origin read restrictions do not replace CSRF and Host validation for local writes.

### Strong engineering inferences: 10

1. A Git-plumbing materializer is the narrowest filter-free cross-platform source path.
2. Tree and execution-world identity must be separate cache jurisdictions.
3. Fresh-world-per-trial is the only simple reset story strong enough for witness minimization.
4. Sequential default execution reduces contamination and timing noise.
5. Process groups are adequate for cooperative descendant cleanup but not containment.
6. Output caps must apply before decoding/parsing.
7. A bearer in the URL fragment plus exact Host/Origin checks is a practical local-app defense.
8. Raw local evidence and shareable report content need different disclosure defaults.
9. In-process plugins are incompatible with the desired trust boundary.
10. The reference demos should avoid dependency installation so the core claim is actually falsifiable.

### Still unverified implementation bets: 7

1. Materializing a 100k-LOC repository from Git plumbing per trial is fast enough for the 15-minute metric.
2. Repeating setup per trial is usable on nontrivial imported repositories.
3. Node's cross-platform filesystem APIs preserve every accepted path/mode case as designed on the supported macOS/Linux filesystems.
4. Group cleanup reliably detects all ordinary descendant patterns in the chosen Node baseline.
5. Proposed output limits are sufficient for real CLI/HTTP fixtures without masking useful evidence.
6. Operators will understand and respect the trusted-code/network warning.
7. Export redaction rules cover enough real secrets to make a default report safely shareable.

### Confidence by subsystem

| Subsystem | Confidence | Reason |
| --- | ---: | --- |
| Git identity and filter-free blob materialization | 9/10 | Directly supported by stable Git plumbing; edge-path implementation still needs hostile fixtures. |
| Tree-to-world semantics | 8/10 | Conceptual boundary is sound; performance is untested. |
| Fresh-world correctness | 8/10 | Strong, simple semantics; expensive setup is the risk. |
| Cooperative process lifecycle | 7/10 | POSIX groups are well-defined; daemon escape/PID races cap the claim. |
| Host containment/network denial | 1/10 | Intentionally not provided. |
| Loopback studio defense | 8/10 | Standard controls; must be tested as a write-capable local app. |
| Report confidentiality | 6/10 | Safe defaults are clear; robust generic redaction remains a bet. |
| macOS/Linux parity | 6/10 | Same protocol is feasible; filesystem/process details require both-host CI. |

## Exact changes required in `docs/CONCEPT_BRIEF.md`

1. **Replace “committed tree contents only” language** with:

   > One trusted local Git repository; 2-4 refs resolved once to immutable commit/tree IDs. Countershape materializes only a declared supported subset of Git entries directly from blobs and modes, without checkout filters/hooks or archive attributes. Unsupported submodules, LFS pointers, path encodings, symlinks, or filesystem collisions become `UNCOMPARABLE`, never silent approximations.

2. **Split source truth in the jurisdiction table**:

   > Git commit/tree IDs answer what source objects were selected. A materialization manifest answers which supported blobs/modes were written. A world manifest answers which platform, toolchain, setup, environment, network, reset, and adapter policy executed them.

3. **Replace “independent temp roots, ports, sanitized environment”** with:

   > Fresh mode-0700 materialization, HOME, TMP, state, and loopback port per candidate trial; sparse allowlisted environment; sequential execution by default; fresh setup for every evidentiary repeat/shrink; `HOST_ALLOWED` network mode. These controls reduce accidental contamination and are not a sandbox.

4. **Add execution states** `MATERIALIZATION_ERROR`, `SETUP_ERROR`, `START_ERROR`, `READINESS_ERROR`, `OUTPUT_LIMIT`, and `ORPHAN_RISK`. None may be normalized into a candidate behavior, successful empty output, or `MISSING`.

5. **Strengthen the cache invariant**:

   > A stable observation/cache key includes commit/tree, materialization manifest, platform/filesystem/toolchain fingerprint, setup argv/result, nonsecret environment policy, network mode, reset/concurrency/port/readiness policy, adapter, stimulus, normalizer, comparator, and reducer versions. Secret-bearing runs are marked operator-supplied and never claimed independently reproducible.

6. **Strengthen the fresh-world invariant**:

   > Every repeated trial and reduction attempt that supports a stable partition starts from a fresh materialization and declared setup. Reused snapshots/reset hooks are a later separately keyed mode and cannot support the reference correctness claim until adversarially validated.

7. **Replace the security-boundary paragraph** with:

   > V1 runs trusted local code with the operator's full user permissions and host network access. The disposable root, sparse environment, byte/output/time budgets, and POSIX process-group teardown are repeatability and accident-containment controls—not filesystem, network, resource, or hostile-code isolation. Descendants that daemonize into another session may escape. Untrusted PRs, dependencies, plugins, remote users, and sandbox claims remain out of scope pending a dedicated reviewed boundary.

8. **Add local-studio requirements**: loopback literal binding, per-session bearer, exact Host/Origin, no CORS, CSRF/CAS writes, bundled assets/CSP, and inert evidence rendering.

9. **Add report disclosure requirements**: private raw store; redacted/escaped report by default; explicit `--include-raw`; redaction metadata; never call redacted output raw.

10. **Move plugin protocol to R5** and say it transports observation adapter messages only, runs as explicit trusted executable code under the same lifecycle policy, and is not an agent adapter/supervisor.

11. **Change R1 acceptance** to require adversarial tests 1-20 above on macOS plus Linux CI before the build claims cross-platform support. Cross-build/typecheck alone is not runtime validation.

12. **Add a kill condition**:

   > Kill or narrow the imported-repository promise if the reference implementation silently uses Git checkout/archive semantics, reuses mutable worlds for stable partitions, advertises a sandbox/network block, cannot stop ordinary grandchildren, or cannot keep infrastructure/setup failures out of behavioral partitions.

## Final ruling

**Proceed with a narrowed reference spine.** Countershape's semantic product remains distinct from Wake: it does not supervise agents, recover fleets, or build an event ledger. The execution substrate here should be deliberately boring, disposable, and subordinate to behavior comparison.

The v1 claim should be:

> Countershape resolves immutable Git trees, materializes a validated filter-free subset, runs cooperative trusted local CLI/HTTP candidates in fresh fingerprinted worlds, bounds and classifies their processes/output honestly, and never upgrades setup/isolation noise into product behavior.

The claim should not be:

> Countershape safely sandboxes arbitrary branches or reproduces any repository exactly across machines.

That distinction is not disclaimer polish. It is the line between a reference-quality differential system and a dangerous local code runner with an attractive UI.
