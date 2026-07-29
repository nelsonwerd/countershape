# P07B-C C4L non-product maintenance authority bootstrap

## Boundary

- **State:** active pre-seal; every C4L grade below is `UNRECEIPTED`.
- **Classification:** `OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR`.
- **Subject:** `fix: bootstrap non-product maintenance authority`.
- **Parent:** exact sealed C4J commit `c12c927d94e6c56529a2fb90148674b4b4e731a5`, tree `c99c1f394e02f17f8cf9de6b18f1a949bcaa07a8`.
- **Verification profile:** `SOURCE_FULL`. C4L introduces but cannot consume `NON_PRODUCT_MAINTENANCE`.
- **Product authority:** none. C4L changes governance, phase/checker authority, and truth-model documentation only. It changes no production Go, generated Node product, schema/example product contract, product predicate, qualification catalog, runtime/verifier implementation, package partition, timeout, cache, parallelism, retry rule, or cumulative row.
- **Product result:** `product_behavior: "INHERITED_UNREPROVEN"` and `product_projection: "PARENT_FROZEN"`. The latter is a prospective fail-closed eligibility token, not a current receipt: a future consumer's final gate must reopen its exact sealed parent and prove no tracked change outside the parent-predeclared roster relative to that parent. V25 defines no enumerated product-projection digest and inherits no verifier result, product-correctness fact, or grade.
- **Transition authority:** `OWNER_OUT_OF_BAND`. C4J did not predeclare C4L. No qualifying pre-existing owner-instruction artifact exists, so the authority is explicitly `UNEVIDENCED`; owner authentication and signed authorization are not established.

C4L is one coherent bootstrap after C4J and before the unchanged frozen C5 product unit:

```text
C4K -> C4J -> C4L -> C5
```

The unit advances the phase specification to `countershape/p07b-c-unit-paths/v25`, retains C3P and C3 receipts as `PRESENT`, retains C6A as `ABSENT`, and leaves every historical receipt grade and claim label unchanged.

The machine authority is `countershape/p07b-c-unit-paths/v25` with 27 ordered phase rows and authority SHA-256 `189e1fe1a69e3937a769867d2e0d82c7b8b432da1de6c78382e97edd6aa5a796`. The plan transition matrix has 15 accepted and 466 rejected cases; the independently implemented unit-scope matrix has 15 accepted and 453 rejected cases. The current Markdown authority corpus has 70 paths at `sha256:5894c9e05456cacff20e2f0095547b032befb13e7aea850583df3972f10fbc07`, with 1080 positive-form rejections and 360 controls.

## Rulings

| Item | Route | Ruling |
| --- | --- | --- |
| Verification cost for productless maintenance | `DEFECT_REPAIR`, prospectively accepted with a narrower claim ceiling | The two-profile model conflates product-bearing source authority with predeclared verifier/checker/documentation mechanics. C4L adds `NON_PRODUCT_MAINTENANCE`, but only a sealed parent may predeclare a consumer's exact row, roster, claim manifest, and profile. |
| C4L self-using the new profile | `DECLINED_WITH_REASON` | Candidate-owned schema and checker bytes cannot authorize their own reduced battery. C4L remains `SOURCE_FULL` and runs the complete three-pass cumulative gate. |
| Surprise maintenance self-insertion | `DECLINED_WITH_REASON` | An unpredeclared repair must first use a separate `SOURCE_FULL` authority-migration boundary. A narrow unit may not edit its declaring phase specification or either generic phase/profile checker. |
| Parent product grades at a narrow child | `INTENTIONAL` non-transfer | Exact parent commit/tree/note are topology prerequisites only. The child must say `INHERITED_UNREPROVEN` and cannot copy, relabel, satisfy, or re-use a parent grade or event. |
| `OWNER_OUT_OF_BAND` provenance | `DEFECT_REPAIR` | Required prose did not distinguish a cited artifact from an unevidenced owner-attributed sentence. The v25 phase graph binds a closed provenance union and includes it in the phase-authority digest. |
| Signed owner authorization | `INTENTIONAL / NOT_IMPLEMENTED` | Citation is not authentication. A signed authorization scheme needs a pre-existing trust key, signer policy, expiry/revocation semantics, and independent verification; C4L invents none of those. |
| C4J sealed-note rejection by frozen C5 | `DEFECT_REPAIR` | C4J sealed cleanly, but its installed didrun export used the legacy double-redacted GOCACHE preview while the future consumer admitted only suffix-preserving projection. C4L pins the exact historical identity and admits that legacy projection only there. |

The earlier C4H sharding refusal remains correct. `NON_PRODUCT_MAINTENANCE` does not split, reuse, or substitute events inside a proof chain. It changes the claim manifest for a separately sealed, strictly narrower boundary whose parent already authorized that exact profile.

The new profile is deliberately dormant in v25: no current row consumes it. That is the cost of refusing self-authorization. V25 installs grammar and partial scope primitives only; it does not install a generic narrow runbook, profile-specific preseal reconciliation, or an end-to-end candidate/stage fixture. A later `SOURCE_FULL` specification owner must predeclare one concrete child and add those exact authorities before the narrower profile can reduce wall time. Verifier/runtime implementation, phase-checker, product-predicate, or core truth-document edits remain `SOURCE_FULL`. At the current observed roughly 18.7 minutes per cumulative pass, the six named remaining product/source units C5, C6, U7, U8, U9, and P12 have a three-pass floor of about 5.6 hours before growth and retry costs. That is a planning forecast, not a didrun receipt or a reason to weaken any gate.

## `NON_PRODUCT_MAINTENANCE` contract

A narrow row is legal only when all of the following hold:

1. The exact row already exists in the sealed direct parent's unit specification. Eligibility never comes from prose, a filename, or the candidate diff.
2. The row declares the narrow profile, no product authority, inherited-unproven product behavior, the prospective `PARENT_FROZEN` eligibility token, empty prefixes, an exact sorted path roster, an exact ordered nine-claim manifest, unchanged receipt states, and the exact predecessor unit.
3. The candidate leaves `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-c-plan.mjs`, `tools/check-p07b-c-unit-scope.mjs`, and the generic profile contract byte- and mode-identical to the parent.
4. The candidate owns only its predeclared stage-zero nonempty `100644` added/modified blobs. Deletion, rename, executable, symlink, gitlink, intent-to-add, unmerged entry, extra path, prefix, and unchanged modified row all refuse.
5. Product/runtime source, generated product artifacts, product schemas/examples, dependency manifests, product-semantic truth documents, product predicate tests, qualification catalogs, phase/profile authority, and cumulative/product claim labels are forbidden.
6. The closed path ceiling admits only `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, one exact status file, files below `docs/prompts/` or `research/`, plus a boundary-specific `tools/check-*-maintenance.mjs`, `tools/verify-*-maintenance.mjs`, or matching `-selftest` file. `tools/verify-current.mjs`, `tools/verify-current-selftest.mjs`, `tools/verify-runtime-authority.mjs`, and all other verifier/runtime implementation are forbidden and remain `SOURCE_FULL`.
7. A future final gate must reopen the exact sealed parent and prove the exact staged diff, exact path roles, and no tracked change outside the parent-predeclared roster relative to that parent. That is an identity fact, not an enumerated product-projection or product-correctness grade. Any claim about verifier steps, package partitions, assertions, qualification digests, or current outcomes remains unproven.

The exact narrow manifest contains six `tests-pass` claims followed by three `command-succeeded` claims:

1. candidate phase plan coherence;
2. independent candidate transition authority;
3. parent-sealed plan-checker defensive self-test;
4. exact sealed-parent note and ancestry compatibility;
5. parent-sealed unit-scope defensive self-test;
6. boundary-specific maintenance and predicate-projection self-test;
7. exact staged maintenance scope, diff, and product-projection identity;
8. scoped staged credential-pattern scan; and
9. exact parent and preceding ledger-chain integrity.

Narrow labels may not state or imply cumulative verification, full-suite coverage, product verification, runtime correctness, security, unchanged behavior, inherited grade, or reused witness. No narrow command invokes `tools/verify-current.mjs`. Product-source boundaries—including C5—remain `SOURCE_FULL`.

## Operator-transition authority

`OWNER_OUT_OF_BAND` is a boundary-governance token, not Countershape product authority. It explains why an otherwise unpredeclared phase edge is admitted. Its machine record has a closed classification, predecessor-declaration state, authentication ceiling, signed-authorization state, and provenance union.

The two provenance states are:

- `CITED_UNAUTHENTICATED`: an exact qualifying owner-instruction artifact already existed as exactly one nonempty, at-most-64-KiB valid-UTF-8 mode-`100644` regular blob in the direct parent's Git tree (`DIRECT_PARENT_TREE`) and is named by literal canonical repository path plus raw-byte SHA-256. Executable blobs, pathspec interpretation, symlinks, trees, gitlinks, empty bytes, oversize bytes, and undecodable bytes refuse. An instruction arriving after that parent requires a separate full-authority ingestion boundary before a later child can cite it. The citation proves only that the named bytes were referenced; it does not prove owner identity, authorship, authorization, freshness, non-revocation, or signature.
- `UNEVIDENCED`: the exact disclosure `OWNER_ATTRIBUTED_SESSION_INSTRUCTION_ONLY_NO_QUALIFYING_PREEXISTING_ARTIFACT` records that no qualifying artifact is available. It does not deny that a real instruction occurred.

A same-unit companion file cannot upgrade its own authority. C4H, C4I, C4K, C4J, and C4L remain historically real owner-attributed authorizations, but none has a qualifying original instruction artifact. C4J also has a sealed predecessor's prose-only record of owner-attributed direction in the C4K status; that paraphrase is not the original instruction and therefore does not change C4J from `UNEVIDENCED`.

The complete v1 machine record for C4L is operative visible authority, not an example:

- `source = OWNER_OUT_OF_BAND`
- `classification = OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR`
- `predecessor_declaration.kind = NONE`
- `provenance.kind = UNEVIDENCED`
- `provenance.disclosure = OWNER_ATTRIBUTED_SESSION_INSTRUCTION_ONLY_NO_QUALIFYING_PREEXISTING_ARTIFACT`
- `authentication = NOT_ESTABLISHED`
- `signed_authorization = NOT_IMPLEMENTED`

Git identity, a didrun receipt, a predecessor paraphrase, or a matching Markdown sentence cannot authenticate the owner.

C4J has the same `UNEVIDENCED` provenance disclosure. Its separate
`predecessor_declaration` is `PROSE_ONLY`:

```text
path = docs/status/P07B-C-C4K-C3-GO-TIMEOUT-MAINTENANCE.md
role = PREDECESSOR_RECORD_OF_OWNER_ATTRIBUTED_DIRECTION
preexistence = DIRECT_PARENT_TREE
digest = sha256:fc80148ddedddb925962a15dacb20dad8979426aa3f3ef4f8fef12df9c614d6b
```

That blob proves only that the predecessor contained the paraphrase. It is
not the original instruction artifact and does not authenticate the owner.

`PARENT_FROZEN` is a prospective fail-closed identity condition, not inherited
verification and not a v25 receipt. Before a future consumer can use it, that
consumer's final gate must reopen the exact sealed parent and prove no tracked
change outside the parent-predeclared roster relative to that parent. V25 does
not enumerate or hash a product-authoritative projection. A missing exact
parent, extra changed path, or mismatched predeclared roster makes the narrow
profile ineligible and requires `SOURCE_FULL`. Even when identity passes,
`product_behavior` remains `INHERITED_UNREPROVEN`; no parent event, result, or
grade is copied, reused, or re-proven.

## Exact sealed C4J compatibility repair

Sealed C4J remains valid and immutable:

- commit `c12c927d94e6c56529a2fb90148674b4b4e731a5`;
- tree `c99c1f394e02f17f8cf9de6b18f1a949bcaa07a8`;
- subject `fix: bound C5 architecture verification`;
- note blob `f76ca73665f8cd3f19fdb9c7d4537462390a2200`;
- note-body SHA-256 `a6b6191547efccf168d18932d559cba5a896c8dcde1f64ede72a27844f495cf1`;
- `13/13` complete `TREE-EXACT` claims; and
- strict exit `0`; and
- `secrets_override: true`.

Its GOCACHE argv preview is exactly `«redacted:high-entropy».«redacted:high-entropy»`.

C4L admits that legacy projection only when all pinned immutable C4J identities above, its exact source partition, exact claim roster, complete coverage, disclosure, direct C4K parent, and lower ancestry agree. Generic, lookalike, or future notes retain their boundary-declared projection. This repairs consumer compatibility without rewriting C4J's note, grade, status, or failed history.

## Scope

C4L owns exactly thirteen paths and no prefix:

```text
docs/ARCHITECTURE.md
docs/CLAIM_VOCABULARY.md
docs/HANDOFF_MODE_C.md
docs/PROMPT_PACK.md
docs/SEMANTICS.md
docs/STATE_MACHINES.md
docs/THREAT_MODEL.md
docs/VERIFICATION.md
docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md
docs/status/P07B-C-C4L-NON-PRODUCT-MAINTENANCE-BOOTSTRAP.md
spec/verification/p07b-c-unit-paths.json
tools/check-p07b-c-plan.mjs
tools/check-p07b-c-unit-scope.mjs
```

Sorted-newline roster digest: `sha256:76c8ecfba55dd8833333c0f6180bf1df38f1a7cba65373feaea281add963b1b3`.

The core truth/security documents are bootstrap-only scope. They do not become generally editable by `NON_PRODUCT_MAINTENANCE`.

## Frozen C5 contract

C5's product contract remains byte-exact:

- 79 labels and command order;
- 76 `tests-pass` plus three `command-succeeded` types;
- 56 isolated qualification cases and their canonical digests;
- three cumulative passes;
- 12 exact paths and three product prefixes; and
- the historical `--verify-c5-sealed-c4-note` argv spelling.

Only phase topology and the implementation behind that frozen compatibility spelling change. C5 starts from dynamic sealed C4L, proves C4L's exact `SOURCE_FULL` shape and note, then exact sealed C4J and the existing `C4K -> C4I -> C4H -> C4 -> C4P -> C4N -> C4M -> C4V -> C3D` chain under one stable outer HEAD. C4L's event cannot satisfy any C5 event.

The C5 frozen claim-contract projection is `sha256:d158e73bf4ddc659cf2cec153037b2ef2d41b07df5d72790b096ede9e6cab9a4` over its 79 labels, 79 types, 79 command tails, 56 qualification IDs, qualification-matrix digest, 12 exact paths, and three prefixes.

## C4L final manifest

The final ledger uses fresh private root `.countershape/p07bc-c4l-final`, sets `umask 077` in every fence, and declares these exact claims:

| # | Label | Type | Intended grade |
| ---: | --- | --- | --- |
| 1 | `P07B-C C4L candidate phase plan coherence` | `tests-pass` | `UNRECEIPTED` |
| 2 | `P07B-C C4L independent candidate transition authority` | `tests-pass` | `UNRECEIPTED` |
| 3 | `P07B-C C4L plan checker defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 4 | `P07B-C C4L transition-authority and maintenance-profile defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 5 | `P07B-C C4L sealed-C4J Git-note and lower ancestry compatibility` | `tests-pass` | `UNRECEIPTED` |
| 6 | `P07B-C C4L unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 7 | `P07B-C C4L cumulative verifier defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| 8 | `P07B-C C4L cumulative verification pass 1` | `tests-pass` | `UNRECEIPTED` |
| 9 | `P07B-C C4L cumulative verification pass 2` | `tests-pass` | `UNRECEIPTED` |
| 10 | `P07B-C C4L cumulative verification pass 3` | `tests-pass` | `UNRECEIPTED` |
| 11 | `P07B-C C4L exact thirteen-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| 12 | `P07B-C C4L scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| 13 | `P07B-C C4L sealed-C4J predecessor and preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

No row becomes a grade until the exact staged tree passes every command through a fresh didrun ledger, commits with the frozen subject, seals to an independently present Git note, exits `NO_COLOR=1 didrun verify --strict` with zero, and renders its HTML evidence.

## Exact final command order

1. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --check-candidate-phase C4L`
2. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --candidate-phase C4L`
3. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --self-test`
4. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --authority-profile-self-test`
5. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4l-sealed-c4j-note`
6. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --self-test`
7. `/opt/homebrew/bin/node tools/verify-current-selftest.mjs`
8. `/opt/homebrew/bin/node tools/verify-current.mjs`
9. `/opt/homebrew/bin/node tools/verify-current.mjs`
10. `/opt/homebrew/bin/node tools/verify-current.mjs`
11. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4L --source-final-gate`
12. `/opt/homebrew/bin/node tools/check-p07b-c-unit-scope.mjs --unit C4L --credential-scan`
13. `/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --verify-c4l-preseal-ledger`

The exact staged tree must exist before command 1. Every isolated shell fence generated for C4L sets `umask 077` before any didrun, claim, commit, seal, note, strict, HTML, or archive command; no persistent outer shell is assumed. Any nonzero command, commit/seal error, nonzero strict grade, or post-stage drift permanently fails that attempt; the failed ledger remains history and a repaired tree starts fresh evidence at command 1.

## Residual limits

- The model remains a trusted-candidate, sole-writer governance system. It does not resist a malicious same-user maintainer.
- Exact product-byte/projection identity is narrower than product semantic correctness.
- Citation remains unauthenticated until a separately designed trust-root and signature protocol exists.
- didrun's secret override is a disclosure, not proof that source or output contains no secret. The independent staged named-pattern scan remains required.
- Historical failed ledgers remain permanent evidence and are never reused.
