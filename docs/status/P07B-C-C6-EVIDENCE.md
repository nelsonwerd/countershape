# P07B-C C6 — cumulative evidence closure

## Boundary and authority

This file is the frozen pre-seal source-era status for the `C6A_SOURCE_CANDIDATE` boundary after exact sealed C5V. Its frozen commit subject is `test: close P07B-C cumulative evidence`. C6A adds no product Go package, product API, semantic object, selector, or P08 surface. It closes cumulative architecture and developer-facing evidence over the already implemented C1–C5 behavior. Later operational state is selected only by the receipt-phase capsule in `docs/HANDOFF_MODE_C.md`.

Every C6A grade remains `UNRECEIPTED` until the exact staged tree completes the eleven-command final ledger, commits, seals, exposes a readable didrun Git note, passes `NO_COLOR=1 didrun verify --strict` with exit `0`, and emits its exact-commit HTML report. This source tree does not receipt itself. C6M will bind sealed C6A source authority; C6B will reconcile that authority into the handoff and this status.

## Exact sealed C5V parent

The direct parent is commit `e7f51c0a8fbd2d6fbe8eacb4a7f0d46f010af7ba`, tree `65bdfc39b48c3b9b1b832a12cc9be5e0083ea5c4`, subject `fix: harden final verification hygiene`, and direct parent `240060f018d5b0e86914bd27361c5899ac9ba1c0`. Its didrun note blob is `a6631221b39f1fa6c5d9d5d9042f93bde67a7be1`, with note-body SHA-256 `1bccdece04834c104e4a70c4ba832a24ad40a1baf5d0d911fb0456809f37f3a2`. All 12 sealed-note claims are `TREE-EXACT`; the previously sealed boundary also recorded an externally observed strict exit `0`. C6A's canonical evidence projects only the note-backed grade fact and does not manufacture a historical process-exit receipt. The note records `secrets_override: true`; that flag is a redacted-export disclosure only and establishes neither secret absence, confidentiality, publication authorization, nor a security review.

The exact-commit report is `.countershape/evidence/p07b-c-c5v-final-e7f51c0a8fbd.html`, 8,540 bytes at SHA-256 `1e7f90f4a9bb8d2dc809967d5152b60eb901606d492ee558df53e2c720b6915d`. The retained final ledger is `.didrun-history/p07b-c-c5v-final-e7f51c0a8fbd/.didrun/`; its complete 18-file descriptor is 176,204 bytes with 14 self-addressed objects, 13 session events, and manifest SHA-256 `8b314f001b6f2b714008d246ef6ed6e850bd1c78213a858e13ccba9c0de5d949`.

## Exact scope and ownership topology

C6A owns exactly these fourteen paths:

1. `docs/ARCHITECTURE.md`
2. `docs/CLAIM_VOCABULARY.md`
3. `docs/CONCEPT_BRIEF.md`
4. `docs/HANDOFF_MODE_C.md`
5. `docs/SEMANTICS.md`
6. `docs/STATE_MACHINES.md`
7. `docs/THREAT_MODEL.md`
8. `docs/VERIFICATION.md`
9. `docs/status/DIDRUN_BUGS.md`
10. `docs/status/P07B-C-C6-EVIDENCE.md`
11. `tools/check-p07b-c-architecture-selftest.mjs`
12. `tools/check-p07b-c-architecture.mjs`
13. `tools/verify-current-selftest.mjs`
14. `tools/verify-current.mjs`

It also admits realized regular source paths beneath exactly `docs/captures/p07b-c/`, `testkit/contractexec/`, and `tools/p07b-c/`. The testkit prefix is evidence input only in this candidate; C6A adds no testkit or product Go bytes. The C6 checker pins the exact direct rosters rather than trusting discovered-only snapshots.

`--c1` is the explicit historical C1 architecture boundary. `--c6` is the explicit cumulative C6 boundary. Exactly two intentional callers use the frozen no-argument compatibility path: the sealed B checker and C6A's frozen final-runbook architecture command. That path executes the same nonrecursive cumulative C6 implementation but emits the exact legacy C1 marker required by B. Every other current cumulative-verifier architecture row uses an explicit mode; C6 never calls inherited B and therefore cannot create `B -> C -> B` recursion.

## Deterministic evidence artifacts

The evidence schema is `countershape/p07b-c-c6a-expert-evidence/v4`; the compact receipt-consumer summary remains `countershape/p07b-c-c6a-evidence-summary/v1`; the terminal grammar is `countershape/p07b-c-c6a-terminal-evidence/v2`. Expert v4 places a closed `artifact_state` object immediately after the admission digest with exact boundary `C6A`, document epoch `C6A_SOURCE_CANDIDATE`, receipt `ABSENT`, and state `UNRECEIPTED`; the terminal banner derives from that source-era object. The summary preserves the predeclared C6B v1 six-key consumer shape while its already-admitted bounded private-evidence note begins `C6A source-era state is UNRECEIPTED` as a compatibility disclosure, not lifecycle or receipt authority. Rich didrun finding details live only in expert evidence. The pure library and bounded entrypoint generate and verify exactly these tracked artifacts:

- `docs/captures/p07b-c/c6a-expert-evidence.json`
- `docs/captures/p07b-c/c6a-evidence-summary.json`
- `docs/captures/p07b-c/c6a-diagnostics-60.txt`
- `docs/captures/p07b-c/c6a-diagnostics-80.txt`
- `docs/captures/p07b-c/c6a-diagnostics-120.txt`

The JSON documents are canonical, LF-terminated, byte-bounded, schema-closed, and derived from a fixed source-input roster. The evidence entrypoint independently reopens the exact C5V commit/tree/note/report/archive, requires full archive object and hash-chain closure plus exact event argv/claim/seal semantics, reopens the four C5V-sealed testkit inputs byte-for-byte, admits and revalidates fixed front-door tool authority under a sterile execution profile, derives and compares the selected-profile in-module compiled/test/embed source closure, explicitly binds the runtime-read choicepoint example and Node parity corpus, requires `go.sum` absent, and reruns exactly the selected `c5-cross-profile-parity` and `c5-http-behavior` Go-JSON profiles. The admission digest binds front-door execution authority, sealed-parent evidence, exact pure profile descriptors, the frozen present/absent source-closure descriptor, and every measured source byte descriptor. Each profile row records the admitted Go executable, exact argv, invocation digest, repository-root working-directory contract, and sterile environment contract. Their admitted surface is 4 CLI tests, 5 HTTP tests across the selected profiles, and 17 generated-Node parity tests, with no top-level or nested skips.

The three text captures are ASCII-only projections of the canonical JSON at exact 60, 80, and 120 column ceilings. They are developer diagnostics, not a product CLI ABI. One exclusive writer lock protects a four-state `prepared -> backed_up -> activated -> committed` v3 journal. The writer creates and syncs an exact sibling generation, records its device/inode plus manifest, rechecks the complete admission snapshot immediately before activation, identity-validates the generation immediately before and after activation rename, validates the active generation before commit, and syncs each directory transition. Destructive cleanup first renames an exact journal-owned generation to a deterministic journal-derived quarantine; a later recovery may resume deletion from a partially cleaned quarantine only when its retained root device/inode still matches the journal. Pre-commit interruption therefore restores the prior generation (or removes the first uncommitted generation) across an additional cleanup interruption; the defensive cases exercise both branches, and once `committed` is durable, cleanup failure retains the already validated new generation. Startup recovery fails closed on unjournaled, foreign, identity-drifted, or ambiguous residue, so no partial mixed generation is admitted. A crash before the first durable journal or during its temporary replacement remains an operator-inspected fail-closed case, not automatically inferred recovery. This protocol is crash-recovery logic under sole-writer filesystem assumptions; path mutations by hostile same-UID actors remain outside the claim, and it does not assert that every storage device honors power-loss durability identically.

## Expert-surface loop and final-freeze protocol

Eight distinct pre-seal passes inspected both canonical JSON documents and the rendered 60/80/120-column surfaces, then exercised the writer/verifier rather than treating a unit test as a substitute:

1. **Pass 1 — parent and generation closure:** the five surfaces were readable, but source/evidence review exposed incomplete C5V archive closure and a non-atomic five-file install. Both were accepted as defects and repaired before regeneration.
2. **Pass 2 — hostile path and transcript closure:** the regenerated surfaces remained unclipped, while exercise exposed ancestor-symlink traversal, Git replace-ref influence, decoded-before-validation Go-JSON, and incomplete package/test lifecycle grammar. Each was repaired and the five surfaces regenerated.
3. **Pass 3 — display and schema closure:** all three widths remained within their exact ceilings, but Unicode display-width ambiguity survived the earlier renderer contract. The renderer was narrowed to deterministic ASCII and the canonical JSON/text generation was repeated.
4. **Pass 4 — post-red-team authority and terminal hierarchy:** after the later authority/transaction repairs, source review caught the frozen C6B summary-v1 consumer mismatch and restored its compact `{id,status}` contract while retaining rich findings in expert evidence. Regenerated terminal review then exposed a shell-looking argv dump whose hard wraps corrupted copy/paste, buried limitations, and reduced one permanent negative to a count. Exact argv remained in expert JSON; the diagnostics gained an early evidence-boundary block, a deliberately non-copyable replay summary, and the negative's exact identifier/effect.
5. **Pass 5 — narrow-width polish:** two fresh reviews found only long-token wrapping in finding locators and the closure-derivation identifier. The terminal projection replaced both with readable bounded references while preserving their exact values in expert JSON, removed a redundant spacer, regenerated all five artifacts, and rechecked semantic parity across 60/80/120 columns. The 60-column surface became 159 lines rather than the pre-repair 180 and remained exactly width-bounded.
6. **Pass 6 — post-cumulative parser closure:** cumulative row 50 exposed the false requirement for every slash-delimited test-name prefix to have its own lifecycle. The repair first aligned the architecture parser and its positive/hostile controls; independent review then caught the same assumption in the evidence parser and narrowed both implementations to exact top-level-root enclosure because Go-JSON carries no parent identifier. A first repaired generation and then the post-red-team generation each completed. The latter passed all three receipted render entrypoints plus an independent static surface check: the 60/80/120 files had 159/133/112 lines, exact maximum widths 60/80/120, ASCII/LF framing, normalized cross-width parity, v2/v1 schema agreement, and the six-finding projection. A fresh sanitized Terra-family critic reported `0 High / 0 Medium / 0 Low`.
7. **Pass 7 — structured nonproof state:** the first source-frozen repeat passed generation, three renders, static parity, C6 architecture/self-test, verifier self-test, and evidence self-test, but its fresh blind critic found `0 High / 1 Medium / 0 Low`: JSON-only readers lacked the terminal's primary `UNRECEIPTED` guard. The expert/terminal portion was accepted as a defect: expert evidence moved from v2 to v3 with a closed top-level source-era state and the terminal now renders from it. A seventh top-level summary key was `DECLINED_WITH_REASON` because the parent-owned C6B v1 consumer exact-closes six root keys and no remaining unit owns that checker. The summary's existing bounded note instead carries the explicit source-era compatibility disclosure without changing that key shape; it does not become lifecycle authority. Pure-library and independent-architecture hostiles reject forged expert state and removed summary disclosure.
8. **Pass 8 — machine-readable prominence and replay safety:** the v3 source-frozen repeat passed the full surface protocol, but its fresh critic reported `0 High / 1 Medium / 1 Low`: canonical key order buried the expert state below the source inventory, and exact argv arrays lacked a structured non-standalone warning. Both expert findings were accepted. V4 replaces the late scalar with an early closed `artifact_state` object including explicit receipt absence, and every profile carries exact `replay_policy` booleans stating that argv is not standalone, is not copy/paste-safe, and requires the declared environment contract. The summary top-level-key portion remains the documented out-of-scope decline with its v1-compatible source-era disclosure mitigation.

The later load-bearing red team opened further authority, source-closure, transaction-recovery, nested-skip, post-profile parent-reopen, ledger-chain, C5V-testkit-byte, document-epoch, slash-bearing-lifecycle, structured-nonproof-state, state-prominence, and replay-safety defects. Passes 4–8 reran the regenerate/inspect/exercise loop after successive repairs; the earlier passes did not pre-receipt changed bytes.

The source-input roster includes this status, so narrating a final capture repeat would itself change the admission digest. The non-self-invalidating freeze rule is therefore: after the last source/document edit, rerun the host-authority writer, all three tracked render entrypoints, the static width/schema/parity check, and a fresh sanitized different-model review; then make no source edit merely to report that repeat. Deterministic results remain in the chain-intact development ledger, qualitative review remains `UNRECEIPTED`, and the final exact-evidence command reopens the frozen bytes in the eleven-command C6A ledger.

## Blind different-model critic

Every blind review used a fresh Terra-family critic and only the five sanitized tracked captures plus a bounded rubric. The first historical pass reported `0 High / 2 Medium / 1 Low`; all three findings were accepted and repaired by adding execution context, full didrun finding projections, and explicit `secrets_override` limits. After the later red team, a fresh pass reported `1 High / 2 Medium / 0 Low`: hard-wrapped shell-looking argv was unsafe to copy, proof limitations were too late in the hierarchy, and the retained termination was only a count. All were accepted and repaired. Two subsequent fresh passes each reported `0 High / 0 Medium / 1 Low`, first for wrapped full finding locators and then for a wrapped opaque derivation identifier; both were repaired without removing exact machine values from expert JSON. The next fresh pass and the post-parser-repair fresh pass each reported `0 High / 0 Medium / 0 Low`. The first source-frozen repeat reported `0 High / 1 Medium / 0 Low` for missing structured `UNRECEIPTED` state; v3 addressed it. The v3 source-frozen repeat then reported `0 High / 1 Medium / 1 Low` for late state placement, the intentionally unchanged summary root shape, and machine-readable replay safety; those results produced v4 plus the explicit summary disposition before the freeze protocol restarted. No critic received a raw didrun object, private root, environment dump, credential value, or external API authority.

Deterministic capture commands may be receipted; the blind critic's qualitative judgment, severity assignment, and disposition advice remain `UNRECEIPTED`, carry no Countershape semantic authority, and establish neither comprehension, taste, correctness, nor security.

## Development evidence and permanent negatives

The chain-intact C6A development ledger retains every nonzero and successful iteration. Early evidence generation first exposed host-epoch unavailability in the managed sandbox; the unchanged host-authority run then exposed an exact generated-Node fuzz-name admission defect. Later defensive passes found and repaired incomplete C5V archive closure, five-file non-atomic installation, ancestor-symlink traversal, Git replace-ref influence, decoded-before-validation Go-JSON output, incomplete package/test lifecycle grammar, Unicode display-width ambiguity, summary/consumer schema drift, transaction cleanup reentry and foreign-residue gaps, activation identity gaps, no-prior-generation coverage gaps, a lifecycle-marker cardinality drift, unsafe/opaque terminal rendering, structured nonproof-state omission, late machine-readable state placement, and absent structured replay safety. The first post-capture cumulative development pass completed rows `1–49/65` and then failed row `50/65`, `architecture-p07b-c-c5`: the parser had incorrectly required a lifecycle event for every slash-delimited textual test-name prefix. A separately receipted raw profile control exited `0` and showed Go directly emitting names such as `TestC5InventoryBindsReferenceFixtureAndRejectsSymlink/setgid/nested-directory` without an intermediate `.../setgid` lifecycle. The repair treats slash-bearing names as opaque identities and requires the exact admitted top-level root lifecycle to enclose each nested interval; it does not invent intermediate parents or weaken exact roster, terminal, skip, output, or pause/continue enforcement. Because Go-JSON carries no explicit parent identifier, prefix-colliding intermediate ancestry is not claimed. The failed cumulative event and two subsequent pre-regeneration self-test failures—managed-sandbox Git stderr, then correctly detected source-input drift—remain permanent and unclaimed. Failed events remain permanent, unclaimed development history and support no C6A grade.

The evidence writer is intentionally host-bound because the selected physical profiles measure Darwin boot-session authority. A sandbox refusal is an honest environmental failure; there is no fallback or synthetic success path.

The canonical expert evidence projects these six unresolved didrun findings without upgrading them into product or security facts; the compact summary carries only their exact `id` and `status` for the frozen C6B receipt consumer:

- `S6-01` — severity `UNASSESSED`; scope: concurrent ledger writers; effect: forked session indices invalidate the ledger; reference: `docs/status/DIDRUN_BUGS.md#s6-01--concurrent-writers-fork-one-session-chain`.
- `S6-02` — severity `UNASSESSED`; scope: wrapped-command output and liveness; effect: retained output is not safely surfaced for diagnosis or progress; reference: `docs/status/DIDRUN_BUGS.md#s6-02--failed-wrapper-output-is-not-surfaced-by-the-cli`.
- `S6-06` — severity `UNASSESSED`; scope: entropy scanning; effect: ordinary digest fixtures force a coarse redacted-export override; reference: `docs/status/DIDRUN_BUGS.md#s6-06--entropy-scanning-blocks-ordinary-cryptographic-fixtures`.
- `S6-07` — severity `UNASSESSED`; scope: concurrent append and claim addressing; effect: duplicate indices can misbind claims and evade strict chain refusal; reference: `docs/status/DIDRUN_BUGS.md#s6-07--a-second-concurrent-append-breaks-claim-address-stability-and-can-evade-strict`.
- `S6-10` — severity `UNASSESSED`; scope: Git-note publication; effect: a note-write failure can leave local seal state advanced; reference: `docs/status/DIDRUN_BUGS.md#s6-10--git-note-attachment-failure-advances-local-seal-state-fail-open`.
- `S6-13` — severity `UNASSESSED`; scope: archived-ledger verification; effect: strict grading becomes `UNKNOWN` when the sealed witness ledger leaves the live path; reference: `docs/status/DIDRUN_BUGS.md#s6-13--strict-verification-is-not-self-contained-after-a-sealed-ledger-moves`.

## Intended C6A claim map

| # | Claim | Type | Intended grade |
| ---: | --- | --- | --- |
| 1 | `P07B-C C6A candidate phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| 2 | `P07B-C C6A independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| 3 | `P07B-C C6A architecture conformance` | `tests-pass` | `UNRECEIPTED` |
| 4 | `P07B-C C6A architecture defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 5 | `P07B-C C6A cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| 6 | `P07B-C C6A cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| 7 | `P07B-C C6A final evidence defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 8 | `P07B-C C6A exact CLI HTTP Node and documentation evidence closure` | `tests-pass` | `UNRECEIPTED` |
| 9 | `P07B-C C6A exact staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| 10 | `P07B-C C6A scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| 11 | `P07B-C C6A declared C5V parent edge and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Generated final-runbook authority

The sole command-order authority is the generated artifact from `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --print-final-runbook C6A`. It must contain exactly the eleven commands corresponding positionally to the claim table above; every command runs once in generated order through its emitted hermetic environment and receives its matching immediate claim only after exit `0`. This status deliberately does not duplicate the generated argv roster.

Any nonzero command remains permanent failed evidence. Repair the real defect, preserve the failed ledger, start a fresh exact-tree final ledger at command 1, re-claim only successful commands, commit, seal, independently read the note, and loop `NO_COLOR=1 didrun verify --strict` until exit `0` without weakening or relabeling a claim.

## Nonclaims and next boundaries

C6A does not establish:

- adoption or market demand
- human comprehension or taste
- maintainership
- production hardening or cross-platform support
- security review or hostile containment

It also does not establish vendor authenticity, complete Go toolchain/standard-library/SDK/dynamic-library or host provenance, host-wide absence, listener ownership, same-UID containment, confidentiality, secure deletion, physical reboot, survivor absence, process resume, universal power-loss recovery, or a supported product CLI. C6M is the next exact five-path source-authority adapter after C6A seals: `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C6M-RECEIPT-ADAPTER-MAINTENANCE.md`, and `spec/verification/p07b-c-c6a-source-authority.json`. C6B is the separate exact five-path receipt reconciliation after C6M seals. External APIs remain human-gated and are neither called nor faked.
