# P07B-C C6M — sealed-source authority adapter maintenance

Classification: `DATA_ONLY_SOURCE_AUTHORITY_ADAPTER / PREDECLARED_SOURCE_FULL`

## Boundary

- **Active boundary:** `C6M`
- **Exact parent:** sealed `C6G`
- **Verification profile:** `SOURCE_FULL`
- **Commit subject:** `fix: bind sealed C6A source authority`
- **Receipt state:** C3P `PRESENT`; C3 `PRESENT`; C6A `ABSENT`
- **Current evidence:** all ten C6M grades are `UNRECEIPTED` until this exact unit commits, seals, and passes strict verification.

C6M is the parent-predeclared data-only adapter after sealed C6G and before future C6B receipt reconciliation. It creates a portable, byte-canonical source-authority record for already-sealed C6A across C6G and C6F's transparent sealed predecessor edges. It does not create a receipt, copy C6A, C6F, or C6G grades into C6M, or assert its own future seal.

## Exact ownership

C6M owns exactly five stage-zero mode-`100644` paths, with sorted-newline roster digest `sha256:5501a7e8c2d8f6d44392d101110edafe90a0348ef77f5ea24539f6a8ba969803`:

1. `docs/HANDOFF_MODE_C.md`
2. `docs/PROMPT_PACK.md`
3. `docs/VERIFICATION.md`
4. `docs/status/P07B-C-C6M-RECEIPT-ADAPTER-MAINTENANCE.md`
5. `spec/verification/p07b-c-c6a-source-authority.json`

There are no prefix scopes. C6M owns neither `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, `tools/check-p07b-c-unit-scope.mjs`, the cumulative verifier, product/runtime source, generated product artifacts, `docs/status/P07B-C-C6-EVIDENCE.md`, `docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md`, nor `spec/verification/p07b-c-c6a-receipt.json`. It cannot change the rules that admit or validate itself.

## Sealed C6A authority

Exact direct parent C6G is commit `3677b45d75204ce2ef7242abb5edd11cc049934e`, tree `520ad2cbf2d3da8354b02afd1cd2a7cd5df82d7d`, note blob `df26876b86d89689d9125531a816b92b7db97afb`, and note-body SHA-256 `2d781bd6373dc0d91b0683a5a0e046ada58522451e0d9f58a71d4df1eb115bc9`, with `10/10 claims recorded-exact`, every grade `TREE-EXACT`, strict exit `0`, and recorded `secrets_override: true`. C6G changed only historical phase-fixture, checker, and lifecycle-documentation authority; it issued no C6A receipt and left the canonical C6A source-authority manifest absent.

Exact sealed C6F beneath it is commit `dd2a011edff20d65c43934fc7c3e5a09b5829d2e`, tree `b0141ac2671df75e51c4fdb5b71c35cee5d0f4ea`, note blob `677e7dda3fc445bda07a7b473b0a40b9a18cea00`, with `10/10 claims recorded-exact`, every grade `TREE-EXACT`, strict exit `0`, and recorded `secrets_override: true`. C6F changed historical authority-fixture and frozen-replay verification machinery, issued no C6A receipt, and left the canonical C6A source-authority manifest absent. C6M must validate the exact `C6M → C6G → C6F → C6A` ancestry while binding the C6A source bytes below.

The canonical `countershape/p07b-c-c6a-source-authority/v1` manifest binds:

- source commit `3e9643f657248e0d5ff2c4bf0880218efbaeecd8`
- source tree `226a1e23da7eded933b3faabd4e8040576b2a2e0`
- single parent `e7f51c0a8fbd2d6fbe8eacb4a7f0d46f010af7ba`
- subject `test: close P07B-C cumulative evidence`
- didrun note ref `refs/notes/didrun`, blob `b110b998a56c89d77bec3f87a48f64f1f21a700e`
- note-body SHA-256 `096e9ae74e9a9ccfb984018f792a59bac3eb8ae24c550d83bb273e6b5ff064c7`
- disclosed `secrets_override: true`
- the exact eleven C6A labels/types, each recorded as `TREE-EXACT`

The plan and independent unit-scope validators reopen the commit, tree, parent, subject, exact C6A diff, note blob/body, complete eleven-event coverage, label/type/order, grade, and full position-specific redaction-tolerant hermetic argv previews. They then require the manifest to equal the canonical pretty-printed JSON plus one trailing newline. Redacted argv positions prove only their admitted structural projection, not the hidden values.

## Authority is not receipt

The manifest's presence does not change the C6A receipt state. During C6M:

- `spec/verification/p07b-c-c6a-receipt.json` remains absent;
- HANDOFF contains no `P07B-C-C6A-SOURCE-RECEIPTS` block;
- inherited `docs/status/P07B-C-C6-EVIDENCE.md` contains no such block and is outside C6M scope; and
- the visible lifecycle capsule continues to state C6A receipt `ABSENT`.

C6B alone may inherit the manifest byte-for-byte, add the separately declared receipt artifact, reconcile the tracked/local evidence declarations, and project the canonical receipt block. Even then, ignored local HTML/ledger availability remains a bounded local observation rather than portable strict witness.

## Exact final claim manifest

The canonical `--print-final-runbook C6M` renderer owns the shell details. Its immutable ten claim-bearing commands and labels are:

1. `P07B-C C6M data-driven phase plan coherence` — `tests-pass`
2. `P07B-C C6M independent candidate transition authority` — `tests-pass`
3. `P07B-C C6M source-authority defensive self-test` — `tests-pass`
4. `P07B-C C6M sealed-C6A source-authority conformance` — `tests-pass`
5. `P07B-C C6M unit-scope defensive self-test` — `tests-pass`
6. `P07B-C C6M cumulative verifier self-test` — `tests-pass`
7. `P07B-C C6M cumulative verification` — `tests-pass`
8. `P07B-C C6M exact five-path staged scope and diff integrity` — `command-succeeded`
9. `P07B-C C6M scoped staged credential-pattern scan` — `command-succeeded`
10. `P07B-C C6M sealed-C6G predecessor, sealed-C6F and sealed-C6A ancestry, and preceding didrun chain integrity` — `command-succeeded`

Claims are issued immediately after their matching zero exit. Any failure leaves the attempt permanent and unsealed. The unit is not complete until its exact commit has a didrun note, every verbatim grade is `TREE-EXACT`, `NO_COLOR=1 didrun verify --strict` exits `0`, an exact-commit HTML report exists, and the final ledger is retained. None of those future outcomes is asserted by this pre-seal document.

## Development launcher observation

The unclaimed development ledger eventually recorded three overlapping successful plan-self-test events after two attached Codex execution sessions yielded without a terminal result and process-name polling incorrectly treated them as gone. Didrun retained all three terminal events and its chain remained intact, so this is a Codex host-session/process-observation failure rather than a didrun or checker failure. Their overlap makes them unsuitable as single-load stability evidence and none is reused in the final ledger. Every subsequent long-running gate must use one tracked detached launcher and wait for its private terminal status before another load begins.

## Nonclaims

C6M adds no Countershape product behavior, compiler/runtime semantics, network capability, external API, adoption evidence, production readiness, cross-platform receipt, malicious same-user isolation, independent security review, release/signing, or maintainership result. The `SOURCE_FULL` battery re-executes the inherited product/checker baseline because that is the C6F-predeclared, C6G-inherited contract; a passing C6M receipt would record those commands against C6M's sealed tree, not establish correctness beyond their named scope.
