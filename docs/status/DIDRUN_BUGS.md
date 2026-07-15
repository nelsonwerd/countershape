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

- **Observed:** `didrun run -- node tools/validate-planning.mjs` correctly returned exit code 1, but the terminal only printed didrun's event summary. It did not relay the wrapped command's stdout or stderr.
- **Impact:** an agent cannot diagnose a recorded failure from the command response or `didrun show`; the captured blobs exist but the CLI exposes no safe output-view command.
- **Run policy:** failure diagnosis is rerun directly only when needed and is labeled `UNRECEIPTED`; the repaired load-bearing command is always rerun through didrun before any claim.
- **Suggested fix:** tee bounded stdout/stderr while capturing, or provide a redaction-aware `didrun show --event N --output` command.

## S6-03 — Root `.didrun/` ignore defeats didrun's tree digest

- **Observed:** on the unborn repository, two successful wrapped validator commands recorded `tree (none)`. Reproducing didrun's temporary-index command showed `git add -A -- . ':(exclude,top).didrun' ':(exclude,top).didrun/**'` returning 1 with “The following paths are ignored: .didrun” when the root `.gitignore` contained `.didrun/`.
- **Control:** removing only that redundant root rule while retaining didrun's own `.didrun/.gitignore` changed the same probe to exit 0 and produce a tree id. `git status --ignored` and `git check-ignore` still show all ledger content ignored. didrun's source also excludes `.didrun` explicitly from every tree digest.
- **Impact:** the apparently recommended defense-in-depth ignore rule makes otherwise successful evidence unbindable (`unknown`) in didrun 0.1.0.
- **Run policy:** rely on didrun's generated self-ignore plus its explicit digest exclusion for the live ledger; keep archived ledgers under the separately ignored `.didrun-history/`. Before every commit, inspect the staged set and refuse any `.didrun` path.
- **Suggested fix:** make the temporary-index add tolerate an explicitly ignored excluded root, and add a fixture whose repository-level `.gitignore` contains `.didrun/`.

## Evidence boundary

These are live product findings, not proof of adversarial tamper resistance, platform portability, or behavior outside this macOS session. The original broken ledger is retained as history rather than rewritten, repaired, or used to support a capability claim.
