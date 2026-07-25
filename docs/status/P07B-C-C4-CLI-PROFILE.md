# P07B-C C4 — CLI-profile physical execution and classification

- **Boundary:** active pre-seal `SOURCE_FULL` child of sealed C4P.
- **Parent:** sealed C4P commit `bbda662314fa3584859ad38f1baeb86ac1f09b5f`, tree `2819dfb90d23da0e89a17f92528338930b73f32c`, direct parent `b915d43cced850936c46b52620654f7506bdb993`.
- **Parent receipt:** didrun note blob `0dbc8868d07eac3fa19a5cfab2a9b584d07209e8`, note-body SHA-256 `4a022906c3d4d4be4a6f32396537cd5c8fac0e2072f06b093b40f7fc8d0e3e75`, `10/10 TREE-EXACT`, strict exit `0`, `secrets_override: true`.
- **Commit subject:** `feat: finalize CLI contract executions`.
- **Receipt state:** C3P `PRESENT`; C3 `PRESENT`; C6A `ABSENT`.
- **Exact scope:** 38 exact paths at sorted-newline `sha256:99bda0d3e5c2f2a2ee4f354945188bf2e5563716c9e26511211a4bfc9197bf56`, plus required realized prefixes `internal/contractexec/runner/`, `internal/processmechanics/`, and `testkit/contractexec/cli/` at `sha256:f5f7b8cdfeba3581e8632c0d6686287f5afa00aef224bb4babe39ee1fdc7f4b2`. The staged realization is 53 paths total: 38 exact plus 15 prefix-owned files split 7 runner, 6 process mechanics, and 2 CLI fixture files.
- **Final evidence root:** `.countershape/p07bc-c4-final`.
- **Final ledger:** exactly 80 commands and 80 immediate claims: 77 `tests-pass`, then three `command-succeeded`.

Every intended C4 grade remains `UNRECEIPTED` until one exact staged tree completes the parent-rendered final runbook, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. Focused development passes cannot receipt this status or the final boundary.

## Implemented physical slice

C4 extracts copy-safe, one-shot Darwin mechanics into `internal/processmechanics`. That package owns argv/environment/capture/process-group mechanics but imports no store, target, semantic model, or permit authority. The existing world edge remains a compatibility adapter. `internal/contractexec/runner` is the sole production consumer of official-target admission and the sole production caller of `AcquireContractRunOwner`; a repository-wide code-only identifier inventory admits exactly the store definition and runner reference. This is a cooperating-source topology claim, not complete dataflow analysis or a security boundary against arbitrary same-user code.

The runner reopens and revalidates target/runtime state across preparation and terminal closure. After durable owner acquisition it creates a detached bounded closure context and performs the final full `ReopenOfficialTarget` in one exact six-statement guarded window: reopen assignment, terminal error guard, exact-freshness guard, target replacement, deferred close, then `consumeAndStart`. Preparation freezes one inert defensive identity from the first fresh target. Every later full reopen first validates its retained predecessor, reconstructs the complete physical graph, and then compares the newly returned canonical model and seven roots with that frozen identity before replacement. Cached bundle/model/path values are used only after that equality edge; no reopen, Node probe, Git/materialization check, boot measurement, or spawn-adjacent check is skipped. Permit consumption remains immediately adjacent to `Prepared.Start`. Post-admission target drift leaves the durable StartClaim and interlock held, leaves RunPermit/start authority unconsumed, and performs no spawn.

One physical enrolled reference target closes clean/projected/complete and remains `CONTRADICTS`. A separate real selected-field ruling is compiled over the same exact enrolled two-file reference tree, executed through the same runner, and yields `CONFORMS` with exactly two profile-ordered UTF-8 fields: `cli.stdout.json.mode=argv` and `cli.stdout.json.source=argv`. Ineligible process/scope controls remain a third distinct class.

## Evidence, durability, and recovery

The CLI fixture writes one canonical invocation receipt bounded to 64 KiB. C4 validates a fresh random attempt ID and exact logical argv, retains a digest-backed validated summary, and retires the child receipt before terminal target revalidation. This is cooperating candidate-written evidence, not hostile-process attestation.

Preparation admits private-evidence capacity before owner acquisition. Two 16 MiB channel ceilings, 1 MiB capture-frame overhead, one 3 MiB projection frame, and twelve 1 MiB canonical-summary ceilings produce a maximum 48 MiB unique-body envelope. Stdout/stderr occupy one binary length frame shared by the logical Drain and Captured references, whose persisted digests must match; projection bytes and the exact tuple occupy a second raw frame. Every draft is capped at 15 logical kinds and 14 unique blobs. The full projected child reaches both caps; controlled or start-error drafts may be smaller. Its actual unique-byte total is revalidated before private-manifest persistence, below the unchanged store ceiling of 16 blobs/64 MiB.

Standalone scope closes exactly five domains: target inventory, child binding, import resolution, service binding, and named sentinel inheritance. Clean inventory is available only for the exact enrolled cooperating two-file fixture. Both inventory passes use Darwin all-component no-follow opens, traverse directory descriptors in 128-entry batches, and stream exactly each admitted file size under ceilings of 20,000 entries, 32 MiB cumulative relative-path bytes, depth 128, 64 MiB per file, and 64 MiB aggregate; every limit or identity failure refuses. Target mutation is a hard refusal before classification. Import/service evidence counts connections to named Unix-socket canaries. Sentinel evidence proves omission only of named environment keys. Static metadata, TAP, or declarations cannot manufacture `COMPLETE`.

Evidence, scope, short-socket, attempt, and attempt-evidence directories use retained identities and descriptor rosters capped at exact expected cardinality plus one. Post-child production cleanup never walks a foreign tree or calls recursive removal: it retires only the retained invocation receipt, sockets, module, and alias, then removes roots proven empty. Deep foreign evidence hard-refuses; deep foreign scope residue or same-size module mutation marks the scope ambiguous; deep foreign attempt/evidence residue blocks reopen. Behavioral tests require each sentinel to remain untouched; that requirement remains `UNRECEIPTED` until the final ledger. These are bounded-work and fail-closed properties, not hostile same-UID containment, secure deletion, or closure of the final identity-check-to-remove race.

The target/attempt/StartClaim-bound private manifest preserves the sorted logical-reference sequence with multiplicity, checks distinct byte ranges/digests, and enforces 16 blobs and 64 MiB. Duplicate logical references fail exact-roster comparison. Private placement, retention-at-finalization, purge, and default omission establish neither confidentiality nor secure deletion.

FCR publication requires closed process/scope facts, exact private manifest/items, and the target-to-run relationship. Release requires exact durable `CLEAR` plus its receipt. Immediately before `ContractExecution` persistence, the store reopens the exact StartClaim and revalidates the finalized-run release relationship; deleting or corrupting that relation blocks classification even when an earlier in-memory release bit exists. `ResumeCLIClassification` is the only C4 “resume” operation. It reuses durable semantic authority and may reopen/revalidate the owned runtime/target graph and converge terminal release/classification, including after private evidence is purged, but it has no preparation, subject-admission, permit, start, capture, or subject-rerun path. Its full live target reopen executes two bounded owned Node runtime probe processes.

## Architecture and verifier authority

C4 is the last writer of the inherited B checker before C5. It creates canonical `spec/verification/p07b-b-future-surface-authority.json`, binds the independently scanned C4 production reserved-family delta, and predeclares a nonempty seven-row C5 HTTP runner skeleton. The canonical 20-row instance is 2,248 bytes at `sha256:2e7c0e9fc530d75e5b8c6cd1cab5d270d5e8ce3e260cda37c4363b3e6c02f49a`; the C4V-sealed schema digest is unchanged. C4 requires runner topology present and HTTP/scope topology absent. C5 must preserve this manifest and the C4P-sealed v20 phase specification and plan/scope checkers byte-for-byte.

The six C4 Go-JSON profiles are:

| Profile | Exact top-level passes | Skips |
| --- | ---: | ---: |
| `c4-processmechanics-parity` | 12 | 0 |
| `c4-admission-permit` | 7 | 0 |
| `c4-cli-closure` | 4 | 0 |
| `c4-finalized-run-release` | 5 | 0 |
| `c4-classification-recovery` | 2 | 0 |
| `c4-authority-race` | 4 | 0 |

The cumulative checker self-test closes 14 metadata cases, 42 Go JSON parser cases, seven command cases, and five owner-reference parser cases. The live C4 qualification matrix contains 54 isolated cases at `sha256:3be703fae10155c83b19be54ae7cd79cfc5b1bbaddfcf86bb2c9781ad93f853a`; the C4-owned dormant C5 matrix contains 56 at `sha256:08db7c338eb3ba900cf6df2545c04f27bed16889c81dd16e5fb18197c4d52958`. Dormant C5 identities exercise parser/selector/builder/runtime lifecycle without running absent packages. The current verifier self-test roster digest is `0fab61318d55314e3accaeabbaf56cc049aae5755f3c3f41d6a36eba1a6c7af5`.

The 20 C4 claim labels named `http-physical-reducer-*` exercise the pre-existing `testkit/studies/http_invoices` study package. They do not implement or validate the absent C5 contract-execution HTTP runner or scope packages.

The verifier has nine whole sensitive packages and retains C4V's direct/general `p=1`, nested/sensitive `p=1`, `GOMAXPROCS=2`, test `-parallel=2`, fresh private caches, and fail-fast exclusive lock. The throughput proposal is already reconciled: persistent Go-only cache and p8 are `DECLINED_WITH_REASON`; the O_EXCL lock and explicit receipt-only roster are `INTENTIONAL / ALREADY SATISFIED`. Current v20 has no unconsumed pre-C4 maintenance row; any future reconsideration requires a separately scoped verifier-authority migration and full requalification. Independent review also confirmed a canonical-entry DX defect: invoking `tools/verify-current.mjs` through a symlink exits successfully without running the verifier. C4 records this as `DEFECT / DECLINED_WITH_REASON`: sealed C4P requires the exact awaited canonical direct-entry guard, so changing it here would violate the parent-sealed source contract. Operators must use the tracked canonical path and require the terminal verification marker; a later verifier-authority migration must replace the silent no-op with an explicit refusal.

The C4V-sealed execution prompt and the concept brief retain older broad “no process rerun” / “never spawns” shorthand. Both files are outside C4's admitted scope. C4 therefore routes their wording as `DEFERRED_AUTHORITY_CORRECTION`, not as current truth: the C4-owned semantics, architecture, threat model, state, handoff, verification, tests, and this status govern the narrower no-contract-subject-rerun claim until C6B or another explicitly admitted authority migration updates those historical surfaces.

## Development evidence and S6

Permanent nonzero didrun events remain non-evidence. The managed sandbox denied the global Go cache, one early hermetic command omitted GOPATH, one shell expansion targeted `/gocache`, and sandbox-injected Git `confstr()` stderr was correctly rejected. Repository-private roots plus approved real host/Git semantics then passed. A proposed reference-digest/private-blob-digest equality broke sealed 16-blob C2 behavior and was reverted; the real multiplicity defect was fixed instead. Documentation reconciliation exposed and repaired a stale cumulative-verifier marker after the hostile checker added five owner-reference parser cases. None of these developmental events is a final C4 receipt.

The first fresh C4 final-ledger attempt passed and immediately claimed commands 1 through 22, then command 23 (`contract-cli-standalone-closure-20`) reached the unchanged 1,200-second child limit and exited nonzero before any claim 23. Its ledger and final root are archived intact as permanent failed history. The cause was throughput amplification, not an awaited 20-second sleep: one repetition performed three physical subject executions plus 13 full target reopens, while redundant defensive getters repeatedly reparsed the complete contract bundle around those reopens. C4 now compares each newly reopened target with the inert prepared identity and reuses content-addressed immutable Git source objects inside the test process; the cache's commit metadata derives from that same content key, making commit identity independent of caller order. Fresh materialization, runtime admission, attempt, target record, interlock, and physical execution remain per call.

A second exact count-20 development run proved that the earlier count-two warmup sample was not predictive: the child again reached 1,200 seconds while its retained JSON tail showed only passing tests, with closure around 57–60 seconds and forbidden controls around 28–29 seconds. A temporary receipted timing probe localized roughly 14.1 seconds to each public target publication and 1.9 seconds to each full reopen; getters were only about 0.1 seconds and preparation about 0.36 seconds. The probe was removed. An intermediate topology that overlapped only reference and conforming then passed the exact count-20 helper in 1,162.043 seconds. That is valid development evidence, but a 37.957-second margin below the immutable child deadline was not accepted as stable enough for finalization.

The current topology leaves all three unchanged physical workflows behind Go's frozen `-parallel=2` ceiling and the test-only capacity-two semaphore. Closure and forbidden are parallel top-level tests; closure retains parallel reference and conforming subtests. This changes only within-iteration pairing: Go and the repetition parser still forbid overlap between separate count iterations. Reference, conforming, forbidden, and ordinary hostile controls use pointer- and canonical-root-distinct stores. Control and forbidden must have the exact reference bundle digest; conforming must differ because it contains the real selected-field ruling that yields `CONFORMS`. The target-mutation control receives a fresh one-shot exact-reference store because its expected target-drift refusal leaves admission held. A selected two-count run passed in 95.051 seconds, the complete four-test package passed twice in 283.550 seconds including two independent mutation refusals, and the unchanged exact count-20 helper then passed in 1,022.843 seconds with 177.157 seconds of child-deadline margin. The checker locks the capacity, role routing, fixed role paths, root/store separation, exact bundle relationships, hostile call-site roster, scheduling topology, and all three qualification workflow bodies. No publication, attempt, subject execution, reopen, recovery edge, or assertion was removed. Claim 23 remains `UNRECEIPTED` until the frozen count-20 command passes on the final staged tree.

An additive focused `-race` run emitted no detector report but exited nonzero in 272.937 seconds when both overlapping branches refused at finalized-run persistence/reopen after instrumentation load exceeded the current pre-seal 32.3-second terminal-closure deadline. It is not a race receipt, and full overlapping CLI-workflow race freedom remains `UNRECEIPTED`. C4 did not enlarge or refresh the product deadline to turn that diagnostic green.

The second C4 final attempt reached 77/80 immediate claims on exact staged tree `aca5f714fc9aa6b29afffc58fe21c2a9bdeb883f`. Runbook command 78 then stopped before scope evaluation: its first exact Git inventory child returned status zero, null signal, and no spawn error, but emitted nonempty stderr, which the gate intentionally treats as fatal. That checker version did not retain the nested stderr bytes, so the exact warning and deterministic cause are unknowable from this event. The already-observed managed-sandbox Apple `confstr()` warning is the strongest supported hypothesis, not a confirmed attribution. The ledger and private root remain intact under their `p07b-c-c4-final-failed-aca5f714fc9a-attempt2` archives, and none of the 77 green events is reusable. The parent-owned gate is outside C4's frozen scope and remains unchanged; bounded captured-stderr diagnostics are queued for the post-C4 verifier-maintenance boundary, where they must remain fail-closed and be labeled terminal-safe but not secret-safe. Final C4 evidence restarts at command 1 under approved host Git execution semantics.

The third C4 final attempt ran on exact staged tree `ee87b43f51aa4f95a27e3a96abe37faf507bf84f`. Commands 1–77 exited zero and were immediately claimed under didrun `env_fingerprint` `3112b643b594e807`. After a Codex UI interruption, archive inspection established that cumulative pass three had completed as zero-based event 76 (runbook command 77) and claim 77 was already bound. The resumed zero-based events 77 and 78—runbook command 78, the exact source-final gate, and command 79, the scoped credential scan—also exited zero and were immediately claimed, but under `env_fingerprint` `588aa17f6ef49b56`. Zero-based event 79, runbook command 80 (`--verify-c4-preseal-ledger`), then exited `1` with exact stderr `P07B-C C4 live event wrapper context mismatch` and received no claim. The preseal gate correctly refused the split wrapper context: the attempt contains 80 events but only 79 claims, and none of its successful events or claims is reusable. The ledger and private root are preserved unchanged at `.didrun-history/p07b-c-c4-final-failed-ee87b43f51aa-attempt3-wrapper-context-split/.didrun/` and `.countershape/p07bc-c4-final-failed-ee87b43f51aa-attempt3-wrapper-context-split/`. C4 remains `UNRECEIPTED`; final evidence must restart at command 1 with a fresh ledger and private root.

The fourth C4 final attempt used one detached host-side driver on exact staged tree `55c208c220f56aec6827f39884e6439fdaaf12df`. Runbook commands 1–74 exited zero and were immediately claimed; zero-based event 74, command 75's first cumulative verification pass, exited `1` and received no claim. All 75 events retained the same tree and `env_fingerprint` `9d275669725f08c3`. The cumulative verifier passed rows 1–16, including all general and sensitive Go packages, then row 17 (`planning-example-p07b-a2-2`) exercised a materialized bundle whose intact entrypoint returned `MALFORMED_CONTRACT|CONTRACT_DATA_INVALID`. The driver had forced inherited process `umask 077`; the planning generator requests regular-file mode `0644` at creation but does not restore the exact mode after POSIX umask reduction, so its bundle files were created as `0600` and the entrypoint correctly rejected the manifest before harness import. An isolated didrun command under explicit `077` reproduced the same diagnostic on the same tree, while the paired explicit-`022` control passed both conformance and pre-import tamper refusal. Those paired causal ledgers remain preserved under their original red and green caller-mode diagnosis archive directories; neither upgrades a C4 grade. The failed final ledger, driver log/status, and private root are preserved at `.didrun-history/p07b-c-c4-final-failed-55c208c220f5-attempt4-cumulative-planning-example/` and `.countershape/p07bc-c4-final-failed-55c208c220f5-attempt4-cumulative-planning-example/`. This is a final-driver context defect plus an out-of-scope planning-fixture hermeticity gap, not evidence of a C4 product regression. didrun's environment fingerprint does not bind process umask, which is retained as an S6 evidence-model finding. C4 cannot edit the C1-owned generator or didrun installation under its sealed roster; the replacement driver keeps its log/status and `.didrun` root private while fixing `022` for command execution, and a later verifier-authority maintenance boundary must make the fixture mode-exact independent of caller umask and decide whether umask belongs in admitted wrapper context. All 74 green events are nonreusable, every C4 grade remains `UNRECEIPTED`, and final evidence restarts at command 1.

Before the fifth final ledger began, its first reviewed detached launch produced a dead screen socket before the worker marker; the precreated log remained empty, and no status, final root, `.didrun`, runbook block, or evidence event existed. That no-ledger runtime failure is preserved at `.didrun-history/p07b-c-c4-final-launcher-failed-709199497939-attempt5-no-ledger/`, and the launcher then gained a private startup marker plus a live-session handshake. The resulting attempt 5b ran on exact staged tree `7091994979392222d0c04fca82c157aa7f59c919` with wrapper fingerprint `9d275669725f08c3`: zero-based events 0–77 exited zero and were immediately claimed, including all three cumulative verification passes and the exact source-final gate. Zero-based event 78, runbook block 80 / evidentiary command 79, then exited `1` during the scoped credential scan and received no claim. The scanner reported its OpenAI-key pattern in exactly the three newly amended documentation files; no credential existed. Each file named the same caller-mode diagnostic archive whose `umask` basename ending sat directly before the separate `-diagnosis-` suffix, producing the credential-shaped substring. The scanner correctly failed closed and remains unchanged; the documentation must avoid that literal concatenation. The permanent 79-event/78-claim ledger, runtime artifacts, launcher, and diagnosis are preserved at `.didrun-history/p07b-c-c4-final-failed-709199497939-attempt5b-credential-scan-path-substring/`, with its private verifier root at `.countershape/p07bc-c4-final-failed-709199497939-attempt5b-credential-scan-path-substring/`. All 78 green events are nonreusable, every C4 grade remains `UNRECEIPTED`, and final evidence restarts at command 1 on a new staged tree.

The first cold C4P final-gate attempt inside the managed sandbox also failed honestly before any product assertion: `/usr/bin/git` exited zero but emitted Apple toolchain-cache warnings that the strict stderr policy rejected; its warm replay then reached `git write-tree` and failed because the sandbox denied `.git/index.lock`. The exact unchanged command passed with approved host Git authority. This is retained as an S6 execution-authority mismatch, not a Countershape product defect or a reusable C4P receipt.

didrun continued to retain failed exits and record successful wrapper exits plus tree identities during this real agent session. The final S6 judgment must still come from the fresh sealed 80-event ledger, strict verification, and exact-commit HTML report.

Late independent review found two genuine pre-seal blockers: dual legal output caps could be represented more than once and exceed private evidence storage, and candidate-writable post-child directories were eagerly enumerated or recursively removed. The implementation, checker, nested behavioral tests, and documentation were rebuilt around the bounded shared persisted-body design above. A first bounded-roster draft also assumed one positive `ReadDir(n)` call must return EOF; development receipts exposed that portability error, and the fixed expected-plus-one loop now permits partial progress, refuses zero progress, and rechecks descriptor/path identity. Targeted package, cumulative architecture, and independently hostile self-test passes are development evidence only.

## Intended C4 claim map

| # | Claim | Type | Intended grade |
| ---: | --- | --- | --- |
| 1 | `P07B-C C4 candidate phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| 2 | `P07B-C C4 independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| 3 | `P07B-C C4 Darwin process-mechanics extraction and world-parity profile` | `tests-pass` | `UNRECEIPTED` |
| 4 | `P07B-C C4 fresh admission StartClaim and single-use RunPermit profile` | `tests-pass` | `UNRECEIPTED` |
| 5 | `P07B-C C4 CLI capture and standalone-scope closure profile` | `tests-pass` | `UNRECEIPTED` |
| 6 | `P07B-C C4 finalized-run durability and interlock-release profile` | `tests-pass` | `UNRECEIPTED` |
| 7 | `P07B-C C4 classifier publication and classification-only recovery profile` | `tests-pass` | `UNRECEIPTED` |
| 8 | `P07B-C C4 exact authority race and boundary-fault profile` | `tests-pass` | `UNRECEIPTED` |
| 9 | `P07B-C C4 cumulative architecture boundary` | `tests-pass` | `UNRECEIPTED` |
| 10 | `P07B-C C4 architecture defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 11 | `P07B-C C4 predecessor U6 compatibility` | `tests-pass` | `UNRECEIPTED` |
| 12 | `P07B-C C4 predecessor U6 defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 13 | `P07B-C C4 predecessor B compatibility` | `tests-pass` | `UNRECEIPTED` |
| 14 | `P07B-C C4 predecessor B defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 15 | `P07B-C C4 inherited U2 mutation-driver compatibility self-test` | `tests-pass` | `UNRECEIPTED` |
| 16 | `P07B-C C4 inherited U3 mutation-driver compatibility self-test` | `tests-pass` | `UNRECEIPTED` |
| 17 | `P07B-C C4 repetition-profile relocation defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 18 | `P07B-C C4 isolated qualification case processmechanics-output-caps-50` | `tests-pass` | `UNRECEIPTED` |
| 19 | `P07B-C C4 isolated qualification case processmechanics-output-independence-20` | `tests-pass` | `UNRECEIPTED` |
| 20 | `P07B-C C4 isolated qualification case processmechanics-simultaneous-overflow-20` | `tests-pass` | `UNRECEIPTED` |
| 21 | `P07B-C C4 isolated qualification case world-lifecycle-readiness-20` | `tests-pass` | `UNRECEIPTED` |
| 22 | `P07B-C C4 isolated qualification case contract-runner-admission-50` | `tests-pass` | `UNRECEIPTED` |
| 23 | `P07B-C C4 isolated qualification case contract-cli-standalone-closure-20` | `tests-pass` | `UNRECEIPTED` |
| 24 | `P07B-C C4 isolated qualification case compiler-generated-runtime-20` | `tests-pass` | `UNRECEIPTED` |
| 25 | `P07B-C C4 isolated qualification case program-lifecycle-20` | `tests-pass` | `UNRECEIPTED` |
| 26 | `P07B-C C4 isolated qualification case store-cross-process-cas-20` | `tests-pass` | `UNRECEIPTED` |
| 27 | `P07B-C C4 isolated qualification case cli-physical-reducer-01-of-20` | `tests-pass` | `UNRECEIPTED` |
| 28 | `P07B-C C4 isolated qualification case cli-physical-reducer-02-of-20` | `tests-pass` | `UNRECEIPTED` |
| 29 | `P07B-C C4 isolated qualification case cli-physical-reducer-03-of-20` | `tests-pass` | `UNRECEIPTED` |
| 30 | `P07B-C C4 isolated qualification case cli-physical-reducer-04-of-20` | `tests-pass` | `UNRECEIPTED` |
| 31 | `P07B-C C4 isolated qualification case cli-physical-reducer-05-of-20` | `tests-pass` | `UNRECEIPTED` |
| 32 | `P07B-C C4 isolated qualification case cli-physical-reducer-06-of-20` | `tests-pass` | `UNRECEIPTED` |
| 33 | `P07B-C C4 isolated qualification case cli-physical-reducer-07-of-20` | `tests-pass` | `UNRECEIPTED` |
| 34 | `P07B-C C4 isolated qualification case cli-physical-reducer-08-of-20` | `tests-pass` | `UNRECEIPTED` |
| 35 | `P07B-C C4 isolated qualification case cli-physical-reducer-09-of-20` | `tests-pass` | `UNRECEIPTED` |
| 36 | `P07B-C C4 isolated qualification case cli-physical-reducer-10-of-20` | `tests-pass` | `UNRECEIPTED` |
| 37 | `P07B-C C4 isolated qualification case cli-physical-reducer-11-of-20` | `tests-pass` | `UNRECEIPTED` |
| 38 | `P07B-C C4 isolated qualification case cli-physical-reducer-12-of-20` | `tests-pass` | `UNRECEIPTED` |
| 39 | `P07B-C C4 isolated qualification case cli-physical-reducer-13-of-20` | `tests-pass` | `UNRECEIPTED` |
| 40 | `P07B-C C4 isolated qualification case cli-physical-reducer-14-of-20` | `tests-pass` | `UNRECEIPTED` |
| 41 | `P07B-C C4 isolated qualification case cli-physical-reducer-15-of-20` | `tests-pass` | `UNRECEIPTED` |
| 42 | `P07B-C C4 isolated qualification case cli-physical-reducer-16-of-20` | `tests-pass` | `UNRECEIPTED` |
| 43 | `P07B-C C4 isolated qualification case cli-physical-reducer-17-of-20` | `tests-pass` | `UNRECEIPTED` |
| 44 | `P07B-C C4 isolated qualification case cli-physical-reducer-18-of-20` | `tests-pass` | `UNRECEIPTED` |
| 45 | `P07B-C C4 isolated qualification case cli-physical-reducer-19-of-20` | `tests-pass` | `UNRECEIPTED` |
| 46 | `P07B-C C4 isolated qualification case cli-physical-reducer-20-of-20` | `tests-pass` | `UNRECEIPTED` |
| 47 | `P07B-C C4 isolated qualification case http-physical-reducer-01-of-20` | `tests-pass` | `UNRECEIPTED` |
| 48 | `P07B-C C4 isolated qualification case http-physical-reducer-02-of-20` | `tests-pass` | `UNRECEIPTED` |
| 49 | `P07B-C C4 isolated qualification case http-physical-reducer-03-of-20` | `tests-pass` | `UNRECEIPTED` |
| 50 | `P07B-C C4 isolated qualification case http-physical-reducer-04-of-20` | `tests-pass` | `UNRECEIPTED` |
| 51 | `P07B-C C4 isolated qualification case http-physical-reducer-05-of-20` | `tests-pass` | `UNRECEIPTED` |
| 52 | `P07B-C C4 isolated qualification case http-physical-reducer-06-of-20` | `tests-pass` | `UNRECEIPTED` |
| 53 | `P07B-C C4 isolated qualification case http-physical-reducer-07-of-20` | `tests-pass` | `UNRECEIPTED` |
| 54 | `P07B-C C4 isolated qualification case http-physical-reducer-08-of-20` | `tests-pass` | `UNRECEIPTED` |
| 55 | `P07B-C C4 isolated qualification case http-physical-reducer-09-of-20` | `tests-pass` | `UNRECEIPTED` |
| 56 | `P07B-C C4 isolated qualification case http-physical-reducer-10-of-20` | `tests-pass` | `UNRECEIPTED` |
| 57 | `P07B-C C4 isolated qualification case http-physical-reducer-11-of-20` | `tests-pass` | `UNRECEIPTED` |
| 58 | `P07B-C C4 isolated qualification case http-physical-reducer-12-of-20` | `tests-pass` | `UNRECEIPTED` |
| 59 | `P07B-C C4 isolated qualification case http-physical-reducer-13-of-20` | `tests-pass` | `UNRECEIPTED` |
| 60 | `P07B-C C4 isolated qualification case http-physical-reducer-14-of-20` | `tests-pass` | `UNRECEIPTED` |
| 61 | `P07B-C C4 isolated qualification case http-physical-reducer-15-of-20` | `tests-pass` | `UNRECEIPTED` |
| 62 | `P07B-C C4 isolated qualification case http-physical-reducer-16-of-20` | `tests-pass` | `UNRECEIPTED` |
| 63 | `P07B-C C4 isolated qualification case http-physical-reducer-17-of-20` | `tests-pass` | `UNRECEIPTED` |
| 64 | `P07B-C C4 isolated qualification case http-physical-reducer-18-of-20` | `tests-pass` | `UNRECEIPTED` |
| 65 | `P07B-C C4 isolated qualification case http-physical-reducer-19-of-20` | `tests-pass` | `UNRECEIPTED` |
| 66 | `P07B-C C4 isolated qualification case http-physical-reducer-20-of-20` | `tests-pass` | `UNRECEIPTED` |
| 67 | `P07B-C C4 isolated qualification case parity-evaluator-20` | `tests-pass` | `UNRECEIPTED` |
| 68 | `P07B-C C4 isolated qualification case parity-framing-20` | `tests-pass` | `UNRECEIPTED` |
| 69 | `P07B-C C4 isolated qualification case parity-full-package-3` | `tests-pass` | `UNRECEIPTED` |
| 70 | `P07B-C C4 isolated qualification case cli-physical-full-package-3` | `tests-pass` | `UNRECEIPTED` |
| 71 | `P07B-C C4 isolated qualification case http-physical-full-package-3` | `tests-pass` | `UNRECEIPTED` |
| 72 | `P07B-C C4 sealed-C4P Git-note and C4N C4M C4V C3D ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| 73 | `P07B-C C4 unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 74 | `P07B-C C4 cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| 75 | `P07B-C C4 cumulative verification pass one` | `tests-pass` | `UNRECEIPTED` |
| 76 | `P07B-C C4 cumulative verification pass two` | `tests-pass` | `UNRECEIPTED` |
| 77 | `P07B-C C4 cumulative verification pass three` | `tests-pass` | `UNRECEIPTED` |
| 78 | `P07B-C C4 exact admitted staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| 79 | `P07B-C C4 scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| 80 | `P07B-C C4 sealed-C4P predecessor C4N C4M C4V C3D ancestry and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Nonclaims and next gate

C4 is CLI-profile-only. HTTP, the two-profile standalone claim, P08 product CLI, studio, report/export, packaging, release, and external API integration are absent. The runtime probe is a cooperating-executable tuple, not Node-vendor authenticity. C4 proves no arbitrary-target inventory completeness, host-wide absence, network/registry denial, listener ownership, same-UID replacement resistance, containment, confidentiality, secure deletion, physical reboot, survivor absence, exactly-once execution, process resume, power-loss recovery, production security/readiness, adoption, comprehension, or maintainership.

C4 creates immutable nonhead target/run/execution evidence and does not advance or freshen the study head/current/latest selectors. Direct Node TAP remains independent operator evidence. C5 may begin only after C4's exact 80-claim commit is sealed, its didrun note is readable, strict verification exits `0`, and the exact-commit HTML/ledger handoff is complete.
