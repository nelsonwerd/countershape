# didrun live-session findings

This file records didrun behavior observed during a real multi-agent build. It is not a replacement for the final S6 report. Raw ledgers and captured command output remain local and ignored because didrun evidence is secret-bearing by construction.

## S6-01 — Concurrent writers fork one session chain

- **Observed:** the pre-commit session contained 14 JSONL entries. Structural inspection found duplicate indices `2` and `7`; `didrun show --session` reported `chain BROKEN at index 2`.
- **Preserved evidence:** the original `session.log` SHA-256 is `623be68cc7a55fcf1245a41ed138a86798a2abc744093ed8a8acb08041125e0e`. The complete original `.didrun` directory is preserved locally under `.didrun-history/2026-07-14-precommit-concurrent/` and is ignored from Git.
- **Mechanism:** source inspection of the installed editable package shows `Session.append()` reads the current tail with `_last()` and then opens the log in append mode, with no process lock or atomic compare-and-append. The duplicate indices are consistent with two agents reading the same tail before either append became authoritative.
- **Impact:** the original ledger is not valid evidence. No claim will be declared over it and it will never be sealed.
- **Run policy:** all remaining didrun writes are serialized by the primary agent. A fresh ledger is used after preserving the failed history. Subagents may edit in parallel, but may not execute didrun concurrently.
- **Suggested fix:** take an interprocess lock around tail read plus append, then re-read the tail after acquiring it. Add a stress test that launches concurrent writers and requires one contiguous, intact chain.

## S6-02 — Failed wrapper output is not surfaced by the CLI

- **Observed (historical pre-sparse invocation):** the early command `didrun run -- node tools/validate-planning.mjs` correctly returned exit code 1, but the terminal only printed didrun's event summary. It did not relay the wrapped command's stdout or stderr. The same happened on successful wrapper event 18 while calculating an independent SHA-256 golden: the event was recorded with exit 0, but the digest text was not visible through either `didrun run` or `didrun show --session`. The ambient command spelling is preserved only as history and must not be copied into a current verification profile.
- **Observed under long real verification:** U4 mutation event `415` ran for 201.95 seconds and repeated-study event `416` ran for 332.42 seconds. Final-tree repeated-study event `459` ran for 447.77 seconds, inherited U2 mutation event `460` for 366.42 seconds, inherited U3 mutation event `461` for 530.21 seconds, and U4 mutation event `463` for 270.12 seconds. None surfaced bounded child progress while running; only the durable wrapper summary appeared after completion. This is safe for evidence retention but poor for a human supervising a long agent session, because silence is indistinguishable from a stuck verifier until the final event arrives.
- **Read-only diagnosis:** inspect the affected event in `.didrun/session.log`, resolve its `stdout_blob` and `stderr_blob` digests, and read the corresponding immutable objects under `.didrun/objects/<digest>`. This recovers the already-recorded wrapped output without manufacturing a second execution. Blob inspection is diagnosis only: it is not a verification event, receipt, claim, seal, or substitute for a successful didrun-wrapped rerun after a real fix.
- **Object-layout footnote:** this didrun workspace mixes Git-style fanout directories and full-digest flat wrapper blobs under `.didrun/objects`. For a wrapper event's `stdout_blob`/`stderr_blob`, the observed retained output path is the flat full SHA-256 name. A future inspection command should hide this implementation detail rather than require agents to infer the object layout.
- **Prohibited workaround:** never rerun a load-bearing command unwrapped merely to see its output. If didrun fails to retain or expose the referenced blob, record that as a didrun bug and keep the affected capability `UNRECEIPTED`.
- **Impact:** an agent cannot diagnose a recorded failure from the command response or `didrun show`; the captured blobs exist but the CLI exposes no safe output-view command. Long successful checks also provide no progressive liveness signal to the supervising human.
- **Run policy:** diagnose only by inspecting the immutable blobs already referenced by the recorded event. After repairing the real fault, rerun the load-bearing command through didrun; an unwrapped diagnostic rerun is never used.
- **Suggested fix:** tee bounded redaction-aware progress while still capturing immutable output, and provide `didrun show --event N --output` plus an explicit running-event heartbeat.

## S6-03 — Root `.didrun/` ignore defeats didrun's tree digest

- **Observed:** on the unborn repository, two successful wrapped validator commands recorded `tree (none)`. Reproducing didrun's temporary-index command showed `git add -A -- . ':(exclude,top).didrun' ':(exclude,top).didrun/**'` returning 1 with “The following paths are ignored: .didrun” when the root `.gitignore` contained `.didrun/`.
- **Control:** removing only that redundant root rule while retaining didrun's own `.didrun/.gitignore` changed the same probe to exit 0 and produce a tree id. `git status --ignored` and `git check-ignore` still show all ledger content ignored. didrun's source also excludes `.didrun` explicitly from every tree digest.
- **Impact:** the apparently recommended defense-in-depth ignore rule makes otherwise successful evidence unbindable (`unknown`) in didrun 0.1.0.
- **Run policy:** rely on didrun's generated self-ignore plus its explicit digest exclusion for the live ledger; keep archived ledgers under the separately ignored `.didrun-history/`. Before every commit, inspect the staged set and refuse any `.didrun` path.
- **Suggested fix:** make the temporary-index add tolerate an explicitly ignored excluded root, and add a fixture whose repository-level `.gitignore` contains `.didrun/`.

## S6-04 — Wrapped commands inherit false-green environment controls

- **Observed:** `didrun run --` records the child argv and result but does not establish a clean child environment. Ambient Go controls such as `GOFLAGS=-run=^$` or `GOFLAGS=-exec=/usr/bin/true`, and Node preload controls such as `NODE_OPTIONS`/`NODE_PATH`, can change or bypass the named verifier while leaving a superficially successful wrapper event.
- **Impact:** an absolute executable alone is insufficient. A command receipt can describe exactly what didrun launched without proving that inherited tool-specific controls were absent.
- **Run policy:** every load-bearing U1 Go/Node command uses `/usr/bin/env -i`, absolute tool paths, fixed HOME/TMP/cache roots, `GOENV=off`, `GOWORK=off`, `GOTOOLCHAIN=local`, `GOPROXY=off`, explicit empty `GOFLAGS`, and no `NODE_OPTIONS` or `NODE_PATH`. The mutation driver additionally fingerprints its operator-declared Go executable. No ambient receipt is claimed.
- **Suggested fix:** support a declared clean-environment mode in didrun and include the effective allowlisted environment in the event body, with secret values structurally excluded.

## S6-05 — Timed Go fuzz teardown can record a false-red

- **Observed:** serialized wrapper event `101` ran `FuzzObservedSelectionAlwaysDerivesTheExactComplement` for the declared 20 seconds, completed baseline coverage and 22,907 fuzz executions, reported no panic, assertion, crashing input, or failing corpus, then exited 1 at 20.11 seconds with only `context deadline exceeded`.
- **Diagnosis boundary:** this is consistent with a Go 1.26.5 timed-fuzz coordinator cancellation race during deadline teardown; it is not evidence that the property passed, and it is not attributed to didrun. The failed event remains permanent and unclaimed.
- **Repair and run policy:** fixture construction was moved outside the per-input callback, the exact-complement assertion was strengthened from cardinality to exact outcome-ID set equality, and the same target must pass again for at least 20 seconds through didrun. If native parallel timed fuzzing repeats the teardown race, retain the false-red and rerun with declared `GOMAXPROCS=1` plus `-parallel=1`; this changes scheduling pressure, not the property or duration. Never relabel the failed receipt as success.
- **Upstream tail:** a production evidence policy should pin a Go release with a verified coordinator fix or prefer an execution-count fuzz budget whose completion does not race a parent deadline. Countershape does not claim to repair the Go toolchain.

## S6-06 — Entropy scanning blocks ordinary cryptographic fixtures

- **Observed:** the first U1 seal attempt for commit `13282ed` stopped with `export blocked: 104 likely secret(s) found (high-entropy)`. U1 intentionally contains many fixed SHA-256 digests, candidate keys, canonical golden values, and evidence identifiers; didrun 0.1.0 reported only the aggregate count and no safe path/line classification with which to distinguish those fixtures from an actual credential.
- **Response:** the blocked attempt remains part of the live session history. After manually reviewing the staged inventory and repository for credential-bearing files, the exact commit was sealed with `didrun seal --commit 13282ed --allow-secrets`. The resulting strict report truthfully displays `sealed with --allow-secrets (redacted export)`; Countershape does not reinterpret that notice as a secret-free finding.
- **Continued live evidence:** later plain seals were blocked by 127 findings at U2, 86 at U3, 121 at U4, and 66 at U5. Each unit first passed an exact staged inventory, staged diff check, and scoped credential-prefix scan, then used the logged redacted-export override. The variation tracks ordinary source/digest/mutant content and does not make the aggregate detector output more actionable.
- **Impact:** entropy alone produces high false-positive pressure in cryptographic and receipt-heavy repositories, while the aggregate message provides too little locality for a precise repair. The override also coarsens the final evidence statement: it says export redaction was requested, not whether every flagged value was benign.
- **Run policy:** never use `--allow-secrets` automatically. First inspect the exact staged path inventory and obvious credential filenames/patterns; then record the override verbatim in status and handoff. Public export or repository publication still requires a dedicated human secret scan outside didrun.
- **Suggested fix:** emit a redacted finding list with file, line, detector kind, and a stable finding fingerprint; support explicit allowlisted fixture regions or semantic digest types without displaying the suspected value.

## Evidence boundary

These are live product findings, not proof of adversarial tamper resistance, platform portability, or behavior outside this macOS session. The original broken ledger is retained as history rather than rewritten, repaired, or used to support a capability claim.
