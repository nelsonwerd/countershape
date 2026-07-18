# P07B-C C1 cumulative-verification maintenance

Status: source repair sealed and strict-clean; receipt-document correction in progress

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

## Scope

This maintenance unit repairs one timing-sensitive assertion in the existing
Darwin process-world test fixture. It does not change capture limits, output
control precedence, process teardown, receipts, or P07B-C execution semantics.

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

P07B-C C1 semantic capabilities are a separate unit and remain `UNRECEIPTED`
until the exact C1 commit is sealed, its Git note exists, and strict verification
exits `0`.
