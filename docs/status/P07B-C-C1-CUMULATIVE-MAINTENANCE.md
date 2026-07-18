# P07B-C C1 cumulative-verification maintenance

Status: source repair and receipt reconciliation sealed and strict-clean; C1 source is separately sealed and strict-clean, and tracked C1 grades require a separately sealed C1B receipt boundary

## Sealed source boundary

- Commit: `fa3d0c12b4c599744b666b2848e38a2499f33a89`
- Tree: `c908c4e580144aa481f646090a1a21d92c86e1cf`
- Strict result: `8/8 claims recorded-exact`; exit `0`
- Git note: present under `refs/notes/didrun`
- Seal disclosure: `sealed with --allow-secrets (redacted export)`
- Ledger archive: `.didrun-history/2026-07-18-p07b-c-c1-output-cap-maintenance/.didrun/`

The plain seal stopped on five aggregate high-entropy findings. A masked
classification located four verifier-generated historical script/status
identifiers and the exact staged status-file path; the exact committed delta
then passed the named structured credential-pattern scan. This is not a
secret-absence finding or permission to publish the local ledger.

The development history permanently retains one unclaimed green scan that ran
after the index was cleared and one unclaimed nonzero scan with an incorrectly
guessed full commit hash. The corrected scan binds the exact full commit and
three-path delta. Neither earlier event supports a capability.

| Claim label | Verbatim grade |
| --- | --- |
| `P07B-C C1 maintenance exact output caps stable 50x` | `TREE-EXACT` |
| `P07B-C C1 maintenance independent limit mutation guard stable 20x` | `TREE-EXACT` |
| `P07B-C C1 maintenance simultaneous overflow lifecycle stable 20x` | `TREE-EXACT` |
| `P07B-C C1 maintenance world package` | `TREE-EXACT` |
| `P07B-C C1 maintenance cumulative verifier` | `TREE-EXACT` |
| `P07B-C C1 maintenance staged diff clean` | `TREE-EXACT` |
| `P07B-C C1 maintenance exact three-path scope` | `TREE-EXACT` |
| `P07B-C C1 maintenance exact commit structured credential scan` | `TREE-EXACT` |

## Sealed receipt reconciliation

- Commit: `3f31370a70979c4c5fe3a523be19e05f84271e96`
- Tree: `29c5708b05c562984a0b7b66d6cfe3ac6e19c626`
- Strict result: `6/6 claims recorded-exact`; exit `0`
- Git note: present under `refs/notes/didrun`
- Seal disclosure: `sealed with --allow-secrets (redacted export)`
- Ledger archive: `.didrun-history/2026-07-18-p07b-c-c1-output-cap-maintenance-receipt/.didrun/`

| Receipt claim label | Verbatim grade |
| --- | --- |
| `P07B-C C1 maintenance receipt reconciliation` | `TREE-EXACT` |
| `P07B-C C1 maintenance receipt cumulative verifier` | `TREE-EXACT` |
| `P07B-C C1 maintenance receipt staged diff clean` | `TREE-EXACT` |
| `P07B-C C1 maintenance receipt exact one-path scope` | `TREE-EXACT` |
| `P07B-C C1 maintenance receipt staged credential scan` | `TREE-EXACT` |
| `P07B-C C1 maintenance receipt didrun chain intact` | `TREE-EXACT` |

The receipt commit reconciles this status file to the sealed source note and
re-runs the then-current 24-step cumulative verifier. Its own plain seal stopped
on seven aggregate high-entropy path/identifier findings after the staged
structured credential scan passed; the logged redacted-export override remains
visible. This is not a secret-absence finding.

## Sealed output-cap scope

The sealed output-cap unit above repairs one timing-sensitive assertion in the
existing Darwin process-world test fixture. It does not change capture limits,
output control precedence, process teardown, receipts, or P07B-C execution
semantics.

## Ruling

`OUTPUT_LIMIT` deliberately starts process-group teardown as soon as either
capture crosses its limit. The old stdout-overflow stimulus wrote stdout first
and stderr second, then required all nine stderr bytes to have been observed.
That requirement depended on the child reaching its second write before the
owner delivered `SIGTERM`; the scheduler is not part of that contract.

The repair keeps every exact-cap assertion and the immediate teardown policy.
It adds a test-fixture mode that completes stderr first, then crosses stdout's
limit by one byte. The sibling-channel evidence is therefore causally prior to
the control event instead of racing it. The existing stderr-overflow stimulus
already has the corresponding order: exact stdout first, overflowing stderr
second.

## Evidence boundary

The eight source claims above establish only the exact repaired test stimulus,
its repeated local Darwin behavior, the complete world package, the then-current
24-step repository verifier, and exact source-unit bookkeeping on this macOS
session. They do not establish scheduler independence on every platform,
production readiness, secret absence, security review, or future-regression
freedom.

P07B-C C1 semantic source commit `2fceacecbacb89fd7650f1570b2af33e6ea25ed3`
is separately sealed, note-present, and strict-clean with `19/19` claims. Its
tracked grades are authoritative only through an agreeing receipt declaration,
source note, status, and handoff plus a separately sealed C1B boundary.

## Separate C1B pre-enrollment lifecycle amendment

The later C1B pre-enrollment maintenance boundary is separate from the sealed
output-cap repair and does change one current Darwin teardown edge. An initial
positive-presence `EPERM` observation is consistent with a transient zombie-only
group state but does not authorize signaling. The owner retries only that exact
state within one quarter of the declared teardown budget, clipped to the overall
deadline. A later absence closes cleanly without a signal; later clean presence
alone authorizes TERM; persistent EPERM and every other error remain fail-closed
with teardown uncertainty and orphan risk.

The final-ledger rehearsal that exposed this edge remains archived as negative
history and supports no claim. The repair decomposes its earlier timer-sensitive
fixture into a causally synchronized late-EOF proof and a live inherited-writer
teardown check. It also adds fake-clock deadline controls so an observation at or
after the retry deadline cannot become signal authority. This paragraph is a
semantic disclosure, not a self-receipt: the exact maintenance commit, didrun
note, and strict result must exist before C1B opens, and C1B cannot relabel this
historical boundary.
