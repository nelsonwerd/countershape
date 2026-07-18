# P07B-B terminal publication and retryable-materialization qualification

- **State:** implementation, red-team correction, and receipt reconciliation are sealed and strict-clean; the current maintenance boundary records the post-archive strict-verification dependency found during final audit and becomes durable only when its own commit receives a Git note and strict exit `0`
- **Prerequisites:** sealed P07B-A2.2 accepted source `6bf33350944dcdd56a4ceaa70f1d0836203ffb1f`, sealed receipt-document boundary `8f7663e2d29b63b1bb3ddca0a18bd5cfcf0fe4b1` at tree `e30957e807c9e8145a286cbb97f7cec309f7897c`, and sealed B0 controlling-spec commit `7f04b2a5129dcbee2fc991453aefdd6713281b86`
- **Receipt ceiling:** one exact terminal `ContractBundle` residue plus optional retryable exact six-file publication on the named Darwin/arm64/cgo tuple; no execution target, contract run, conformance classification, product surface, containment, or production claim
- **Controlling contract:** `docs/prompts/P07B-B-TERMINAL-PUBLICATION.md`
- **Source boundary:** commit `eb06bdcf18f8e14db1257e73733afdc05cac045e`, tree `d95c907754f3347bae885a64c98b30ad1ea00256`; Git note present; strict exit `0`; `16/16` claims recorded-exact
- **Receipt-document boundary:** commit `b4ac17bd66c6260fd4b12b9277ab5259659c55e1`, tree `7e9847d976d32a6307e7ccd5f2437391f0a4298a`; Git note present; strict exit `0` with its witness ledger live; `6/6` claims recorded-exact
- **Seal disclosure:** the source and receipt-document plain seals stopped on 39 and 27 aggregate high-entropy findings after their exact staged inventories and scoped named credential-pattern scans passed; both notes record `secrets_override: true` and strict reports `sealed with --allow-secrets (redacted export)`, which is neither a secret-absence result nor publication authorization
- **Evidence rule:** source grades below are copied verbatim from the source Git note; the receipt-document note and S6-13 maintenance evidence remain separate and cannot upgrade the product claim ceiling

## Implemented semantic boundary

P07B-B adds a terminal publication edge without turning parsed bundle bytes or
copied digests into authority:

- the sealed `PreparedBundle` retains the original private portable-ruling preparation;
- the node-only issuer creates an opaque publication capability bound to the exact study, prior head digest, DecisionRecord, Choicepoint, lineage root, bundle digest, and canonical bundle bytes;
- `store.AdvanceResidue` accepts only that leaf capability, validates every join, reconstructs one `ContractBundle` semantic object, and reaches the existing raw head owner exactly once;
- the complete expected `RULING` token is compared under the study transition exclusion before successor-object or head-temporary creation;
- the bundle object becomes durable before the terminal head can name it, while a pre-head interruption may honestly leave an unreachable immutable object;
- terminal durability is reconciled by a nonmutating study-lock, directory-sync, exact-head, and exact-object reopen path;
- `OpenResidue` reconstructs the terminal bundle plus its exact DecisionRecord -> Choicepoint -> FreshConfirmation predecessor chain without compiler or source inputs; and
- the resulting sealed `Residue` is bound to the store instance and must freshly reopen before every physical materialization edge.

Same-authority callers converge on one terminal revision: one constructor path
reports `PUBLISHED`, while a reconciled loser or replay reports
`ALREADY_CURRENT`. A different bundle or foreign store cannot become an
alternate terminal successor through this API.

## Native materializer boundary

`internal/contractmaterialize` is a cycle-free filesystem owner. Its exported
entrypoint accepts only a store-bound `node.Residue`; it exposes no raw bundle,
body, digest, callback, interface, or generic output API.

On the implemented Darwin/arm64/cgo path it:

1. validates a clean absolute destination and proves physical as well as lexical disjointness from the object-store namespace;
2. opens the destination parent with a component-wise no-follow descriptor walk, requires effective-UID ownership, owner `rwx`, no group/other write or special bits, retains that directory identity, and repeats the trust check before and after publication;
3. allocates a private same-parent mode-`0700` stage;
4. creates the fixed six mode-`0644` files exclusively and descriptor-relatively, handling short writes and zero progress;
5. syncs, closes, and independently reopens each effective-UID-owned member twice to verify type, link count, special bits, mode, size, digest, and exact bytes; the exact directory is likewise effective-UID-owned;
6. independently reads the complete six-name roster twice, rejecting extra and case-fold-equivalent aliases;
7. revalidates through the private store-bound source at the create or exact-existing acceptance boundary;
8. publishes only with `renameatx_np(RENAME_EXCL | RENAME_NOFOLLOW_ANY)` and has no ordinary-rename fallback;
9. syncs the parent and reopens the exact final directory before returning `CREATED`; and
10. treats an already-existing exact directory as `ALREADY_EXACT` without rewriting any member.

Unsupported tuples return a typed incomplete/unsupported result. A mismatched,
partial, linked, aliased, wrong-mode, wrong-kind, or extra-member destination is
immutable refusal: it is never repaired, merged, renamed aside, or overwritten.
Failures before publication clean only the call-owned private stage when exact
ownership and removal can be established. Failures after the exclusive rename
boundary are reported as ambiguous unless exact reconciliation succeeds.

## Exercised implementation matrix

The sealed-source suites include:

- real CLI `ALLOW_OBSERVED` and child-bind HTTP `CUSTOM_EXPECTATION` publication;
- in-process and fresh-process/store publication convergence;
- cancelled and foreign-store publication inventory preservation;
- replay without revision or object-inventory change;
- direct typed-store transition closure for exact, zero, cancelled, foreign, mismatched-head, and stale calls, plus nonmutating terminal confirmation under exact, cancelled, invalid, missing-object, corrupt-head, and replaced-store states;
- restart reconstruction without compiler, corpus, or source inputs;
- refusal after removing the bundle, DecisionRecord, Choicepoint, or FreshConfirmation;
- two independent CLI destinations, exact-existing inode-preserving retry, immutable tamper refusal, and unchanged terminal head;
- sequential and two-goroutine HTTP same-bundle convergence, plus a genuine two-residue different-bundle destination race with exactly one creator and one immutable mismatch refusal, with no surviving private stage;
- descriptor-relative exact publication, retained-parent replacement/trust drift, intermediate-symlink replacement, case aliases, same-inode mutation between independent reads, special bits, full file-kind/content/mode/link/roster mismatch, stage allocation/cleanup uncertainty, roster-close failure, short/zero writes, existing-output observation failure, destination appearance at rename, post-rename sync/close/reopen faults, rename-outcome ambiguity, and cleanup faults; and
- cancellation after the successful pre-rename stage sync with exact cleanup, plus cancellation immediately after successful exclusive rename with uninterrupted durability reconciliation; and
- lexical and deterministic physical root, descendant, and parent-alias object-store/output overlap refusal.

These are implementation inventory statements. Only the specifically named
commands in the receipt map inherit their verbatim didrun grades; the inventory
does not create broader security, portability, or production claims.

## Architecture and cumulative-evidence design

The current P07B-B checker retains the inherited A2 boundary and checks the
exact owner topology, hidden issuer, store edge, complete head-token equality,
nonmutating reconciliation path, restart ordering, one public-entry reopen plus
create/exact-existing branch revalidation through the private store-bound
source, no existing-output mutation edge, observation-ambiguity versus physical-
mismatch classification, repeated exact member/roster reads, parent trust and
output ownership structure, descriptor walk, full alias scan, native flags/no
fallback, rename reconciliation, forbidden imports, and absence of every
P07B-C target/run/classification owner.

Its defensive self-test intentionally uses clean execution plus cloned
metadata-fact negatives. It does not create or rewrite production source
copies. This replaces an abandoned recipe-level source-rewrite approach with
black-box boundary evidence and makes no mutation-completeness claim. The
cumulative verifier explicitly adds both B gates after A2; inherited current
checkers remain active and historical-only gates remain labeled rather than
silently executed or claimed on the evolved tree.

## Qualification receipt map

These grades are copied verbatim from the sealed Git note on source commit
`eb06bdcf18f8e14db1257e73733afdc05cac045e`, tree
`d95c907754f3347bae885a64c98b30ad1ea00256`. The serialized ledger contains
exactly 16 complete events and 16 claims; its independent closure reports
`16 events chain intact`.

| Claimed capability | didrun claim label | Verbatim grade |
| --- | --- | --- |
| Go formatting and Node verifier syntax | `P07B B formatting and verifier syntax` | `TREE-EXACT` |
| Focused publication, reopen, store, and authority suites | `P07B B focused publication reopen and store suites` | `TREE-EXACT` |
| Materializer durability and fault matrix | `P07B B materializer durability and fault matrix` | `TREE-EXACT` |
| Real CLI terminal publication, restart, retry, and refusal matrix | `P07B B CLI publication restart and materialization matrix` | `TREE-EXACT` |
| Real HTTP publication and concurrent materialization convergence | `P07B B HTTP publication and concurrent materialization matrix` | `TREE-EXACT` |
| Focused race-enabled terminal publication/materialization packages | `P07B B focused race suite` | `TREE-EXACT` |
| Store public API and typed transition closure | `P07B B store public API closure` | `TREE-EXACT` |
| Exact P07B-B architecture boundary | `P07B B exact architecture boundary` | `TREE-EXACT` |
| Defensive metadata-fact architecture self-test | `P07B B architecture defensive self-test` | `TREE-EXACT` |
| First complete repository suite under stock macOS temporary-root policy | `P07B B complete Go repository suite pass one` | `TREE-EXACT` |
| Second consecutive complete repository suite | `P07B B complete Go repository suite pass two` | `TREE-EXACT` |
| Complete Go vet | `P07B B complete Go vet` | `TREE-EXACT` |
| One-command cumulative current baseline | `P07B B cumulative verification baseline` | `TREE-EXACT` |
| Exact staged inventory and diff integrity | `P07B B exact staged inventory and diff check` | `TREE-EXACT` |
| Scoped staged structured credential-pattern scan | `P07B B scoped staged structured credential-pattern scan` | `TREE-EXACT` |
| Serialized didrun ledger chain integrity | `P07B B didrun chain intact` | `TREE-EXACT` |

The later, post-spec user requirement for an exact-final-B-commit HTML report
is part of this boundary, even though the earlier sealed controlling prompt
deferred HTML to final closure. Generate it outside Git after the receipt-
document commit is sealed and strict-clean; it is a closure artifact, not a
substitute receipt-table claim. Overall project closure must generate a new
report for its own final commit; a B report cannot be reused for a later tree.
The source-bound report is preserved locally at
`.countershape/evidence/p07b-b-source-eb06bdcf18f8.html` with SHA-256
`df743fc83943eb3b4e903698860a5658ade4b29ccffe01aa2eb453a7d387ddfb`.
The receipt-document report is preserved at
`.countershape/evidence/p07b-b-final-b4ac17bd66c6.html` with SHA-256
`09f870a05eb9726d27fa2db13064fe5cb4b8ca182708b5a4ddeeb047cfc8414b`.
Final audit then proved that strict verification exits `1` with all claims
`UNKNOWN` when the matching ledger is only archived, and returns to `0` after
an identical live copy is restored; `docs/status/DIDRUN_BUGS.md` S6-13 records
the exact boundary. Because the maintenance record changes HEAD, generate a
new exact-final-commit HTML after that maintenance commit seals.

## Permanent negative history

The archived development ledger preserves expected stale architecture pins,
checker-development failures, and a materializer test that initially assumed a
fault-injected handle remained open after the corrected cleanup path had
already closed it. Those events support no capability and must never be
relabelled. Successful development events also belong to superseded tree
identities and support no final claim. The complete 69-event development ledger
is preserved at `.didrun-history/2026-07-17-p07b-b-development/.didrun/`; the
separate 16-event source ledger is preserved at
`.didrun-history/2026-07-17-p07b-b-source/.didrun/`.

Platform generation refusals encountered while drafting an abandoned dense
source-rewrite driver are session diagnostics, not didrun evidence, a caught
security defect, or a product failure. No such driver was completed, claimed,
or sealed. The checked-in replacement self-test is metadata-only.

## Explicit limits and nonclaims

Even after every planned gate passes, P07B-B does **not** establish:

- an atomic object/head transaction, rollback, unreachable-object/stage garbage collection, or general power-loss proof;
- NFS, remote, case-sensitive-volume-general, Linux, Windows, non-arm64, non-cgo, or other filesystem/toolchain behavior;
- hostile same-user or same-UID isolation, extended-ACL interpretation, coordinated-replacement resistance, sandboxing, confidentiality, authenticity, authorship, secret absence, external-network denial, or a security review;
- elimination of residual same-UID cleanup/name races or coordinated ancestor/name replacement windows; effective-UID ownership, final-parent mode checks, retained descriptors, and repeated identity checks narrow them but do not establish hostile containment;
- a `ContractExecutionTarget`, admitted runtime, fresh finalized run, conformance classification, execution receipt, or standalone target-inventory absence;
- a product CLI, server, dashboard, report, studio, installer, migration contract, release/signing process, or operational support policy; or
- production readiness, adoption, demand, human comprehension, legal/trademark clearance, or maintainership.

The strongest allowed sentence after exact qualification remains the claim
ceiling in the controlling prompt; a didrun grade never upgrades that scope.

## Sole next feature boundary

After the implementation and receipt-document commits are each sealed,
Git-note-present, and strict-clean, the sole next feature unit is P07B-C:
immutable `ContractExecutionTarget`, `FinalizedContractRun`, and derived
`ContractExecution` authority plus the fresh standalone conformance matrix.
Product CLI/server/studio work remains later and independently gated.
