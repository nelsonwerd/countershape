# P07B-C C6R — sealed C6N note-consumer maintenance

**State:** active pre-seal `SOURCE_FULL` defect repair. Every C6R grade is `UNRECEIPTED` until its exact staged tree completes its own ten-event didrun ledger, commit, seal, readable Git note, strict verification, HTML report, and ledger archive.

## Boundary and authority

C6R is the owner-authorized child of exact sealed C6N and the required parent of C6B. It repairs only the descendant consumer projection for C6N's immutable note bytes. It changes no Countershape product behavior, production Go source, cumulative-verifier implementation or row roster, sealed history, C6A source-authority manifest, or receipt state.

- **Commit subject:** `fix: bind sealed C6N note redaction projection`
- **Direct parent:** exact sealed C6N commit `096842954c9e6ae72b66010fff81e90cf0bd7a22`, tree `69845d09f85abf76b275b16c0d941fbd568ee907`
- **Parent note:** blob `b66c7b38bad6c8ea94288f2cd8afa87e76f0c831`, body SHA-256 `68abfb23fe35e2d2844e0a1622f7c4566f08789a29d16dd170c02b0e7046ef11`
- **Sealed C6M ancestor:** commit `b0de83dca3510102af4d1cddae2b2023bc88a87d`, tree `3d80a02443209f5d5649fcba92b79b72a2a04a15`
- **Sealed C6G ancestor:** commit `3677b45d75204ce2ef7242abb5edd11cc049934e`, tree `520ad2cbf2d3da8354b02afd1cd2a7cd5df82d7d`
- **Sealed C6F ancestor:** commit `dd2a011edff20d65c43934fc7c3e5a09b5829d2e`, tree `b0141ac2671df75e51c4fdb5b71c35cee5d0f4ea`
- **Sealed C6A ancestor:** commit `3e9643f657248e0d5ff2c4bf0880218efbaeecd8`, tree `226a1e23da7eded933b3faabd4e8040576b2a2e0`
- **Transition source:** `OWNER_OUT_OF_BAND`
- **Classification:** `OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR`
- **Predecessor declaration:** `NONE`
- **Provenance:** `UNEVIDENCED`, disclosed as `OWNER_ATTRIBUTED_SESSION_INSTRUCTION_ONLY_NO_QUALIFYING_PREEXISTING_ARTIFACT`
- **Authentication:** `NOT_ESTABLISHED`
- **Signed authorization:** `NOT_IMPLEMENTED`

The owner instruction was real, but no qualifying instruction artifact pre-existed in the direct-parent tree. This candidate-owned status cannot authenticate or retroactively evidence it. Receipt C6A remains `ABSENT`; the C6A source-authority manifest is inherited byte-identically, no receipt declaration or visible receipt projection is introduced, and no prior grade transfers into C6R.

The current machine authority is `countershape/p07b-c-unit-paths/v30`, with 44 ordered unit rows and 32 receipt-phase rows at receipt-phase authority digest `sha256:cd7bd450bec6d891ea66dd48408788d38aa7bbcee778912eabce7f45e8538f3e`.

## Observed defect and ruling

Exact sealed C6N is valid and immutable. Its ten note previews carry the visible GOCACHE form `«redacted:high-entropy».countershape/p07bc-c6n-final/gocache`. The pre-existing descendant consumer instead projected the prospective two-marker form and failed closed at `C6N: didrun note claim 1`. Rewriting C6N's source tree, note, grade, or seal would erase rather than repair the historical boundary.

The redaction-form difference is deterministic and content-sensitive, not temporal: the same didrun redactor independently redacts the C6M suffix but preserves the C6N suffix after the common root token is redacted.

The frozen source-authored C6N prospective contract remains `sha256:500955d69b63da43516312788befa6642ecbdcb6e7777ac014f5beeeb5beb41b` and retains policy `LEGACY_DOUBLE`; C6R does not reinterpret or retroactively rewrite that authored prediction. The separately named observed sealed-C6N projection is `sha256:cc851cc157036b09ae84739220db65bdc6593d0f78644d5a61fe789d6a93ffc4` and uses policy `SUFFIX_PRESERVING`. The prospective C6R contract is `sha256:bc4dffd3b0a393eefbb7d3947a1ad9ca7a391a7a2a332113a893e0682a88cfde`.

Both independent checkers must derive the observed projection only after pinning C6N's exact commit, tree, parent, subject, note blob, and note-body digest. Raw exact previews remain admissible only under that exact authority. Cross-form substitution, bare GOCACHE values, positional drift, claim or command drift, and foreign note bytes remain refused. The observed projection is a descendant-consumer fact about sealed bytes, not an amendment to C6N and not evidence that a C6R note already exists.

## Scope

C6R owns exactly these seven mode-`100644` paths:

1. `docs/HANDOFF_MODE_C.md`
2. `docs/PROMPT_PACK.md`
3. `docs/VERIFICATION.md`
4. `docs/status/P07B-C-C6R-NOTE-CONSUMER-MAINTENANCE.md`
5. `spec/verification/p07b-c-unit-paths.json`
6. `tools/check-p07b-c-plan.mjs`
7. `tools/check-p07b-c-unit-scope.mjs`

The sorted-newline roster digest is `sha256:3f815ac91f817fba618f24d307919dd9e97aec0f142a5cee1c9e86b284e06169`.

Prefixes are empty. The inherited C6A manifest, C6 receipt declaration and evidence, product/runtime source, cumulative verifier implementation and roster, sealed C6N/C6M/C6G/C6F/C6A commits and notes, and all failed-attempt archives are outside this roster and remain byte-stable.

## C6R final manifest

| # | Claim | Type | Intended grade |
|---:|---|---|---|
| 1 | `P07B-C C6R data-driven phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| 2 | `P07B-C C6R independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| 3 | `P07B-C C6R sealed-C6N note-redaction compatibility` | `tests-pass` | `UNRECEIPTED` |
| 4 | `P07B-C C6R sealed-C6N Git-note and sealed-C6M, sealed-C6G, sealed-C6F, and sealed-C6A ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| 5 | `P07B-C C6R unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 6 | `P07B-C C6R cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| 7 | `P07B-C C6R cumulative verification with sealed-C6A replay` | `tests-pass` | `UNRECEIPTED` |
| 8 | `P07B-C C6R exact seven-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| 9 | `P07B-C C6R scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| 10 | `P07B-C C6R sealed-C6N predecessor, sealed-C6M, sealed-C6G, sealed-C6F, and sealed-C6A ancestry, and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

## Exact final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C6R`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C6R`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
4. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c6r-sealed-c6n-note`
5. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
6. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
7. `/opt/homebrew/bin/node tools/verify-current.mjs`
8. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C6R --source-final-gate`
9. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C6R --credential-scan`
10. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c6r-preseal-ledger`

Run each command once in the generated C6R runbook and claim it immediately; any nonzero command stops that ledger, supports no claim, and requires a repaired fresh boundary.

The first seven prove the phase transition, the observed sealed-C6N suffix-preserving note projection and frozen prospective mismatch, exact C6N/C6M/C6G/C6F/C6A ancestry, both generic checker self-tests, the cumulative-verifier self-test, and unchanged cumulative verification with sealed-C6A replay. The last three prove exact staged ownership, structured credential-pattern absence over the seven blobs, and final ledger/ancestry coherence. No C6R claim grades product behavior independently of that cumulative command.

## Claim ceiling and continuation

C6R may claim only the bounded sealed-C6N note-consumer compatibility repair, exact parent and ancestor authority, current checker and cumulative-verifier coherence, exact staged scope, and exact didrun chain integrity after its own receipt exists. It does not claim hidden-value recovery, product change, C6A receipt, renewal or correction of any sealed ancestor, portable host stability, hostile same-UID isolation, secret absence, confidentiality, security certification, production readiness, adoption, or maintainership.

The next permitted edge is C6B only after this boundary independently closes. C6B must inherit the C6A source-authority manifest byte-for-byte, add only its predeclared receipt surfaces, and validate exact C6B-to-C6R-to-C6N-to-C6M-to-C6G-to-C6F-to-C6A ancestry. No grade crosses that edge automatically.
