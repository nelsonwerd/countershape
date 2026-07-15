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
- **Read-only diagnosis:** inspect the affected event in `.didrun/session.log`, resolve its `stdout_blob` and `stderr_blob` digests, and read the corresponding immutable objects under `.didrun/objects/<digest>`. This recovers the already-recorded wrapped output without manufacturing a second execution. Blob inspection is diagnosis only: it is not a verification event, receipt, claim, seal, or substitute for a successful didrun-wrapped rerun after a real fix.
- **Prohibited workaround:** never rerun a load-bearing command unwrapped merely to see its output. If didrun fails to retain or expose the referenced blob, record that as a didrun bug and keep the affected capability `UNRECEIPTED`.
- **Impact:** an agent cannot diagnose a recorded failure from the command response or `didrun show`; the captured blobs exist but the CLI exposes no safe output-view command.
- **Run policy:** diagnose only by inspecting the immutable blobs already referenced by the recorded event. After repairing the real fault, rerun the load-bearing command through didrun; an unwrapped diagnostic rerun is never used.
- **Suggested fix:** tee bounded stdout/stderr while capturing, or provide a redaction-aware `didrun show --event N --output` command.

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

## Evidence boundary

These are live product findings, not proof of adversarial tamper resistance, platform portability, or behavior outside this macOS session. The original broken ledger is retained as history rather than rewritten, repaired, or used to support a capability claim.
