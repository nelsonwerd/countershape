# Countershape

**A completed autonomous build.** An AI agent built this repository on its own from July 14 to August 14, 2026 — about a month, minus a several-day pause at usage limits — under a verification discipline where every claim has to be backed by a recorded command. All ten units of the roadmap are sealed, and so is the final evidence handoff. On one tested machine profile it builds and runs.

The repository was public from early on because the *process* is the interesting part, and process is only interesting if you can inspect it.

> ### Can you run it?
>
> **Yes — on the tested profile.** `make release` produces a single binary; `countershape studio` opens the blind decision screen in your browser; `countershape export` writes a local HTML report. It has only been exercised on macOS arm64 with Go 1.26.5 and Node 25.2.1. It is **not** packaged for distribution, **not** production-hardened, and its own self-verifier still won't run from a clone (two files hardcode absolute paths from the build machine — see [Known limitations](#known-limitations)).
>
> It is a research instrument, not a sandbox, CI service, agent runtime, or security product. Candidates run with your user's permissions and host network access.

---

## Quick start

Tested profile: macOS arm64, Go 1.26.5, Node 25.2.1, Apple Git 2.50.1, and the checked-in dependency tree.

```sh
make release
./dist/release/countershape-darwin-arm64/countershape --help
./dist/release/countershape-darwin-arm64/countershape studio
./dist/release/countershape-darwin-arm64/countershape export \
  --output "$PWD/countershape-report.html" \
  --acknowledge-confidentiality-not-established
```

The exported report is a local, inert, default-minimized presentation. **CONFIDENTIALITY IS NOT ESTABLISHED** — it is not sanitized, anonymous, or safe to share. Raw export additionally requires both `--include-raw` and `--i-understand-raw-may-contain-secrets`. The single binary embeds the studio assets and exposes help/version, `studio`, `export`, and the frozen HTTP/CLI study routes.

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
| Status | **Complete.** All ten units (0–9) of the roadmap are sealed and closed, and the final evidence handoff is sealed. The build is finished |
| Sealed commits | 90 |
| Verified claims | 1208 (1171 clean, 36 stale, 1 failed) |
| Go — product code | ~81,000 lines |
| Go — tests | ~68,000 lines |
| TypeScript — studio UI | ~1,000 lines |
| JavaScript — checkers | ~92,000 lines |
| **Runnable?** | **Yes — on one tested profile** (macOS arm64, Go 1.26.5, Node 25.2.1), from source. Not packaged for distribution; not production-hardened |
| Stats as of | 2026-08-15 |

The whole engine is sealed — exact Git materialization, isolated execution worlds, CLI and HTTP observation, the bounded reducer, the decision store, the ruling authority, the standalone-test compiler, publication, and execution classification. Units 7 through 9, the parts a human actually touches, are sealed too: the command-line entry point (`cmd/countershape`), the blind decision **studio** (a React/TypeScript screen on a local-loopback Go server), and export/packaging (a single embedded binary plus an HTML report). What it still is **not**, in the project's own words: finished-as-a-product, production-ready, hostile-code-safe, market-validated, independently secured, legally cleared, or maintainable without a maintainer. `docs/FINAL_HANDOFF.md` says exactly that.

---

## The receipts

This is the part worth your attention.

Every commit carries a cryptographic receipt — attached as a git note — recording every verification command that ran against that exact code, its real exit code, and the grade each claim earned. The grades are honest: `tree-exact` (recorded against this sealed tree), `scope-exact` (recorded against a declared subset of it), `stale` (the code changed after the check), `failed` (the command did not succeed).

**Git does not fetch notes by default.** To see them:

```bash
git fetch origin refs/notes/didrun:refs/notes/didrun
git notes --ref=didrun show HEAD
```

That prints the receipt for the latest commit. Swap `HEAD` for any commit hash to see its receipt.

Receipts are produced by [didrun](https://github.com/nelsonwerd/didrun), a separate tool built for exactly this. Important: **`tree-exact` means "this command ran and exited zero against this exact tree." It does not mean the code is correct**, and it does not mean the check was a good check. It answers attribution, not correctness.

---

## Five things you'll find if you go looking

Better you hear them here than discover them and assume something was hidden.

**1. 89 of 90 seals record `secrets_override: true`.** This looks like the credential scanner was bypassed 89 times. It was — and every single firing was a false positive from one string. The scanner's OpenAI-key pattern is `/sk-(?:proj-)?[A-Za-z0-9_-]{20,}/`, unanchored, with the hyphen inside the character class. A diagnostic directory named `...c4-umask-diagnosis-red-<digest>` contains the substring `sk-diagnosis-red-...` — because **the word "umask" ends in "sk."** An independent audit replayed the exact regex over all 1,978 objects in the repository's history and found exactly two matches, both from that one directory name, and zero real credentials anywhere.

**2. One claim is graded `failed` and thirty-six are `stale`, across three commits.** Thirty of the stale grades and the single failure sit in two commits from the first 30 hours, when the harness was still being shaken out. The other six are `stale` grades inside the final packaging unit (commit `f49f4bf`), for an almost too-perfect reason: they are the performance-measurement claims, and the act of recording the measurements — writing the timing numbers into the evidence files — changed the tree the claims had just measured, so they graded stale against their own results. None of the thirty-seven was ever deleted, relabeled, or re-sealed; the status documents quote them verbatim and state plainly that none supports a capability. The other 87 commits are clean.

**3. There is roughly as much checking as there is product.** ~81,000 lines of product Go, against ~68,000 lines of tests and ~92,000 lines of JavaScript that exists solely to check the rest. Put another way: the checker JavaScript alone outweighs the product it verifies. There is also a five-day window in the history where ten milestones sealed and **zero lines of product code were written** — 88% of everything added in that stretch was checker JavaScript. (For the record: the studio UI that shipped in unit 8 is the run's first and only non-Go product code — about 1,000 lines of TypeScript, dwarfed by everything built to check the Go.) That is not a bug in the log; that is what happened, and it is one of the more interesting things this run has produced.

**4. There is a fully green seal that was thrown away.** On August 12 the run sealed the CLI decision milestone and verified it 12/12 `tree-exact` — every command passed, the witness report was generated. Then it voided the whole thing: one of the thirteen recorded events had run under a different recorder environment fingerprint than the other twelve. Not a code failure, not a test failure — an inconsistency in the evidence *about* the code. The passing seal was demoted to permanent failed-attempt history, and the milestone re-ran every event from zero against a rebuilt tree and re-sealed clean the next day. The discarded receipt is still in the notes ref, keyed to commit `9ed68ae2`, which this branch intentionally no longer contains — which is why a raw count of receipts comes to 91 while this README counts 90. The full account of both attempts is in `docs/status/U7C-CLI-DECISION-CONTRACT.md`.

**5. They pointed a real agent at it — and published how imperfectly the recorder caught it.** The final evidence includes two independent live sessions of a real coding agent (OpenAI Codex, `gpt-5.6-terra`) run in fresh throwaway clones and captured by didrun. The honest scorecard is right there in `evidence/s6/`: of the bare shell commands the agents actually ran, the recorder's PATH shim caught about 75% of the required ones and 50% across the full transcripts. A plain `git diff` and every absolute-path command (`/bin/pwd`) went uncaptured, and are labeled structural misses rather than quietly counted as hits. The tool documents the exact holes in its own evidence.

---

## What the boundaries mean

Countershape keeps a set of authorities strictly separate, and the code names them so they can't quietly blur together:

- A **tree identity** names exact admitted source bytes. It is not a branch name or a trust claim.
- **Materialized files** are a fresh execution root rebuilt from pinned Git objects.
- A **comparison envelope** states which dimensions are exact, which are recorded-only, and which are outside the claim.
- **Captured evidence** is private physical process/filesystem evidence; a **projection** is a typed, bounded view of it, never the capture itself.
- A **ruling** is a caller-attributed decision over one exact choicepoint.
- An **emitted bundle** is a standalone six-file contract residue, not an execution target.
- A **didrun receipt** records command evidence. Its grade is opaque and never upgraded into a conformance claim.

---

## Known limitations

- **The product runs; the self-verifier still won't run from a clone.** The `countershape` binary builds and runs on the tested profile, but two checker files hardcode absolute paths from the build machine (`tools/check-p07b-c-architecture.mjs`, `tools/check-p07b-c-plan.mjs`), so the full self-verification won't reproduce from a fresh clone. They were left untouched during the build on purpose — the run gated on an exact staged-file inventory, so editing anything mid-flight would have voided the unit being sealed — and are on the short post-run cleanup list. See [docs/LIMITATIONS.md](docs/LIMITATIONS.md).
- **macOS / Apple Silicon only.** Much of the execution substrate is Darwin-specific by design and says so.
- **It runs trusted code with your user permissions.** The isolated directories are for repeatability, not security. The project is explicit about this and never claims sandboxing. See [SECURITY.md](SECURITY.md) and [docs/THREAT_MODEL.md](docs/THREAT_MODEL.md).
- **Absolute paths from the build machine** appear in the docs and receipts. Scrubbing them would require rewriting history, which would orphan every sealed receipt — so they stay.

---

## How it was built

One prompt, then no human steering. The build was driven by [idea-to-ship](https://github.com/nelsonwerd/idea-to-ship-skills), a set of agent skills that takes a project from a fuzzy idea through research, planning, and building. Its `autopilot` mode runs the whole chain unattended — and because there is no human in the chair, it opens by generating a persona to *be* the person using it. That persona (documented in `docs/PERSONA_SOREN_VALE.md`) chose what to build. The operator never named the product.

The command-line units were driven by the receipt discipline above. The browser studio in unit 8 was built with a different skill — a build → see → exercise → check loop suited to UI — which is why that unit carries a Playwright and accessibility harness alongside its receipts. Verification throughout ran through [didrun](https://github.com/nelsonwerd/didrun).

The `docs/status/` directory contains a status document for every sealed milestone — 63 of them — each declaring exactly what it claims and what it explicitly does not. `docs/FINAL_HANDOFF.md` is the terminal handoff written when the run finished; `docs/HANDOFF_MODE_C.md` is the cursor at the moment it ended.

---

## Credits

Market grounding research in `research/grounding/` was gathered using [**last30days**](https://github.com/mvanhorn/last30days-skill) by [@mvanhorn](https://github.com/mvanhorn) (MIT) — an agent skill that researches a topic across Reddit, Hacker News, YouTube, GitHub and the web. `research/grounding/raw/` contains its raw output, including quoted public posts and transcript excerpts from their original authors, retained verbatim so the synthesis above it can be checked against its sources.

---

## License

Apache-2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE). No trademark, domain, or package-name clearance is claimed.

---

*A full writeup will follow, including the parts that do not flatter it.*
