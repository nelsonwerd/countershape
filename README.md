# Countershape

**A live, unfinished autonomous build.** An AI agent has been building this repository on its own since July 14 — nearly four weeks, minus a several-day pause at usage limits — under a verification discipline where every claim has to be backed by a recorded command. It is running again as you read this. The first CLI code sealed August 10, but there is still nothing you can install and no interface you can use.

The repository is public early because the *process* is the interesting part, and process is only interesting if you can inspect it.

> ### Should you clone this?
>
> **Not to run it — not yet.** Nothing works from a clone today: the CLI is only just taking shape, and two files still hardcode absolute paths from the machine it's being built on. Clone it to *read* — the code, the 63 status documents, and the receipts are all worth a look right now.
>
> **When the build finishes**, the hardcoded paths get fixed, the command-line interface and decision screen land, and this README is replaced with real install and usage instructions. Watch the repo if you want to know when that happens.

---

## What Countershape is meant to be

When several AI agents write the same feature, you get branches that all compile and all pass the tests — and quietly disagree about things nobody specified. Does an unauthorized request return `403` or `404`? Which config source wins? What happens on malformed input? Nothing catches these, because nothing was ever written down about them.

Countershape treats those candidates as **sensors, not voters**. It:

1. Runs one declared input against each candidate branch in a fresh, isolated environment
2. Strips volatile noise (timestamps, request IDs, scratch paths) down to exact canonical bytes
3. Produces an exact map of which candidate produced which outcome
4. Shrinks the input to the smallest case that still reproduces the disagreement
5. Re-runs everything fresh to confirm the split is real
6. Shows a human the differences **blind** — no branch names, no vote counts, so you can't rubber-stamp the majority
7. Emits a standalone test encoding your ruling, which runs with Countershape uninstalled

It never picks a winner and has no opinion about correctness. Its strongest possible claim is: *under this one finite experiment, these candidates produced these exact outputs, and a human ruled on one case.*

The bet is that generating code got cheap and **deciding which behavior you actually wanted** became the expensive part. The project's own concept brief rates adoption odds at 3 out of 10.

---

## Current state — honestly

| | |
|---|---|
| Started | 2026-07-14 |
| Status | **Still building.** Units 0–6 of a 10-unit roadmap sealed and closed; unit 7 — the CLI, the first thing a human will touch — opened August 9 and has sealed three product milestones since, most recently the CLI decision flow on August 13 |
| Sealed commits | 85 |
| Verified claims | 1170 (1139 clean, 30 stale, 1 failed) |
| Go — product code | ~79,000 lines |
| Go — tests | ~67,000 lines |
| JavaScript — checkers | ~92,000 lines |
| **Runnable?** | **No.** No CLI, no UI, no install path |
| Stats as of | 2026-08-14 |

The core engine exists and is sealed: exact Git materialization, isolated execution worlds, CLI and HTTP observation, a bounded reducer, the decision store, the ruling authority, the standalone-test compiler, publication, and execution classification. What does *not* exist yet is everything a human would touch — the command line, the blind decision screen, and packaging. Those are units 7 through 9, and unit 7 is well underway: the run has sealed the CLI's entry point and application shell (`cmd/countershape`), an HTTP falsification study, and — on August 13 — the CLI decision flow, the first sealed code on the path to the blind decision screen. It is still a foundation, not a usable tool.

If you clone this expecting to run something, you will be disappointed. Come back later.

---

## The receipts

This is the part worth your attention.

Every commit carries a cryptographic receipt — attached as a git note — recording every verification command that ran against that exact code, its real exit code, and the grade each claim earned. The grades are honest: `tree-exact` (recorded against this sealed tree), `stale` (the code changed after the check), `failed` (the command did not succeed).

**Git does not fetch notes by default.** To see them:

```bash
git fetch origin refs/notes/didrun:refs/notes/didrun
git notes --ref=didrun show HEAD
```

That prints the receipt for the latest commit. Swap `HEAD` for any commit hash to see its receipt.

Receipts are produced by [didrun](https://github.com/nelsonwerd/didrun), a separate tool built for exactly this. Important: **`tree-exact` means "this command ran and exited zero against this exact tree." It does not mean the code is correct**, and it does not mean the check was a good check. It answers attribution, not correctness.

---

## Four things you'll find if you go looking

Better you hear them here than discover them and assume something was hidden.

**1. 84 of 85 seals record `secrets_override: true`.** This looks like the credential scanner was bypassed 84 times. It was — and every single firing was a false positive from one string. The scanner's OpenAI-key pattern is `/sk-(?:proj-)?[A-Za-z0-9_-]{20,}/`, unanchored, with the hyphen inside the character class. A diagnostic directory named `...c4-umask-diagnosis-red-<digest>` contains the substring `sk-diagnosis-red-...` — because **the word "umask" ends in "sk."** An independent audit replayed the exact regex over all 1,978 objects in the repository's history and found exactly two matches, both from that one directory name, and zero real credentials anywhere.

**2. One claim is graded `failed` and thirty are `stale`**, all inside two commits from the first 30 hours. They were never deleted, relabeled, or re-sealed. The status documents quote them verbatim and state plainly that none of those claims supports a capability. The other 83 commits are clean.

**3. There is roughly twice as much checking as there is product.** ~79,000 lines of product Go, against ~67,000 lines of tests and ~92,000 lines of JavaScript that exists solely to check the rest. Put another way: the checker JavaScript alone outweighs the product it verifies. There is also a five-day window in the history where ten milestones sealed and **zero lines of product code were written** — 88% of everything added in that stretch was checker JavaScript. That is not a bug in the log; that is what happened, and it is one of the more interesting things this run has produced.

**4. There is a fully green seal that was thrown away.** On August 12 the run sealed the CLI decision milestone and verified it 12/12 `tree-exact` — every command passed, the witness report was generated. Then it voided the whole thing: one of the thirteen recorded events had run under a different recorder environment fingerprint than the other twelve. Not a code failure, not a test failure — an inconsistency in the evidence *about* the code. The passing seal was demoted to permanent failed-attempt history, and the milestone re-ran every event from zero against a rebuilt tree and re-sealed clean the next day. The discarded receipt is still in the notes ref, keyed to commit `9ed68ae2`, which this branch intentionally no longer contains — which is why a raw count of receipts comes to 82 while this README counts 81. The full account of both attempts is in `docs/status/U7C-CLI-DECISION-CONTRACT.md`.

---

## Known limitations

- **The verifier will not run from a clone — this is temporary.** Two files hardcode absolute paths from the build machine (`tools/check-p07b-c-architecture.mjs`, `tools/check-p07b-c-plan.mjs`). They can't be touched while the build is live: the run gates on an exact staged-file inventory, so editing anything mid-flight would void the unit currently being sealed. **Both are on the post-run cleanup list**, along with the CLI, packaging, and install instructions. Nothing about this is a design decision — it's the cost of publishing a repository while an agent is still writing to it.
- **macOS / Apple Silicon only.** Much of the execution substrate is Darwin-specific by design and says so.
- **It runs trusted code with your user permissions.** The isolated directories are for repeatability, not security. The project is explicit about this and never claims sandboxing.
- **Absolute paths from the build machine** appear in the docs and receipts. Scrubbing them would require rewriting history, which would orphan every sealed receipt — so they stay.

---

## How it's being built

One prompt, then no human steering. The build is driven by [idea-to-ship](https://github.com/nelsonwerd/idea-to-ship-skills), a set of agent skills that takes a project from a fuzzy idea through research, planning, and building. Its `autopilot` mode runs the whole chain unattended — and because there is no human in the chair, it opens by generating a persona to *be* the person using it. That persona (documented in `docs/PERSONA_SOREN_VALE.md`) chose what to build. The operator never named the product.

Verification runs through [didrun](https://github.com/nelsonwerd/didrun).

The `docs/status/` directory contains a status document for every sealed milestone — 63 of them — each declaring exactly what it claims and what it explicitly does not. `docs/HANDOFF_MODE_C.md` is the live cursor showing where the run currently is.

---

## Credits

Market grounding research in `research/grounding/` was gathered using [**last30days**](https://github.com/mvanhorn/last30days-skill) by [@mvanhorn](https://github.com/mvanhorn) (MIT) — an agent skill that researches a topic across Reddit, Hacker News, YouTube, GitHub and the web. `research/grounding/raw/` contains its raw output, including quoted public posts and transcript excerpts from their original authors, retained verbatim so the synthesis above it can be checked against its sources.

---

## License

MIT. See [LICENSE](LICENSE).

---

*A full writeup will follow when the run completes, including the parts that do not flatter it.*
