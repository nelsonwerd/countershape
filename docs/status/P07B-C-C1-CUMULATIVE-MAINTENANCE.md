# P07B-C C1 cumulative-verification maintenance

Status: source repair pending receipt and seal

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

## Intended receipts

- repeated exact independent-cap test;
- repeated mutation guard for distinct stdout/stderr limits;
- repeated simultaneous-overflow lifecycle test;
- complete `internal/world` package test; and
- cumulative repository verification after the maintenance commit is joined
  to the C1 branch.

Until the unit is committed, sealed, and strict-verified, all capabilities in
this status file are `UNRECEIPTED`.
