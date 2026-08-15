# S6 live-agent result

- **Verdict:** `PARK`
- **Provider/model:** OpenAI `gpt-5.6-terra`
- **CLI:** `/Applications/ChatGPT.app/Contents/Resources/codex`, `codex-cli 0.147.0-alpha.6.5`
- **Capture:** `/opt/homebrew/bin/didrun`, `didrun 0.1.0`, installed from user-local source `$USER/autopilot-dev-stress-test/src/didrun`; Tier-1 PATH shim
- **Fixture commit:** `8c711915bf5dd8a59542744d6f7bc65e09bf6d51`
- **Frozen prompt SHA-256:** `b6fbe3b0718904a65c3f7bfb6a59f9b6a77c67f173a27b10cdd725165db5c4f6`

Two independent real Codex sessions ran in fresh disposable Git clones outside the Countershape checkout. Both changed exactly `message.txt` and `count.txt`, ran the successful fixture test twice, ran and continued after the intentional exit-7 failure, invoked `/bin/pwd`, inspected the diff, and returned normally. The sessions used `--ephemeral`, ignored user/project rules, emitted JSON tool-call transcripts, used the workspace-write sandbox, and did not copy Countershape source, user data, credentials, or a Countershape ledger into the fixtures.

The first attempted launcher combined mutually exclusive Codex flags `--sandbox` and `--approve-for-me`. Codex rejected it at argument parsing with exit 2 before creating an agent session. It is retained as a failed P12 wrapper event, excluded from the two-session denominator, and not called an S6 result.

## Recall matrix

| Session | Meaningful two-file change | Required bare commands captured | Full-transcript bare commands captured | Intentional failure | Absolute path |
| --- | --- | ---: | ---: | --- | --- |
| 1 | exact | 3/4 | 3/8 | transcript 7; capture 7 | 0/1 captured; 1 documented structural miss |
| 2 | exact | 3/4 | 3/4 | transcript 7; capture 7 | 0/1 captured; 1 documented structural miss |
| Total | two independent exact changes | 6/8 = 75% | 6/12 = 50% | 2/2 honest | 0/2 captured; 2/2 explicitly classified |

The custom `fixture-test` executions were captured four times and the custom `fixture-fail` executions were captured twice with the correct nonzero exit. The standard bare `git diff` command was absent from both shim ledgers even though a `git` shim was installed at the front of the inherited PATH. Session 1 also used spontaneous bare `rg`, `ls`, and two `sed` invocations; none appeared in the shim ledger. Both `/bin/pwd` calls were absent as the installed didrun protocol predicts for absolute paths and are classified as structural misses rather than silently counted as captured.

The precommitted pass bar was at least 95% bare-name recall, 80–95% was a tier downgrade, and below 80% or any silent class miss was `PARK`. The required-command recall is 75% and the full transcript recall is 50%, so the only honest result is `PARK`.

## Finding and limits

`S6-P12-01 — live Codex shell command resolution bypasses incomplete Tier-1 shimming`: custom unique shims survived, but a standard installed `git` shim did not intercept the agent's bare `git diff`, and unshimmed spontaneous commands were invisible. This is a live-agent capture limitation, not evidence that the agent failed its fixture and not a defect in Countershape's directly wrapped Tier-0 receipts.

The audit checked the ignored raw transcripts and child ledgers for named credential patterns before producing this summary. That bounded scan is not secret-absence authority. Raw JSONL transcripts, final messages, stderr, and child ledgers remain only under ignored private `.didrun/s6/` and are not committed. The deterministic comparison is reproduced by `tools/final-audit/s6-audit.mjs`; the machine matrix is `evidence/s6/s6-matrix.json`.
