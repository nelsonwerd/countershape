# P07B-B — terminal ContractBundle publication and retryable materialization

> **Controlling scope lock:** This prompt narrows the P07B umbrella's
> publication/materialization section into one independently shippable unit.
> It does not weaken `P07-U6-STANDALONE-CONTRACT.md`. Where an implementation
> detail was previously implicit, the authority, durability, result-state,
> and native-filesystem rulings below control P07B-B.

## Role

Act as a storage and filesystems engineer. Exact authority flow, compare-before-
publish ordering, restart reconstruction, crash-state honesty, no-follow
filesystem behavior, idempotent retry, and operator-facing error semantics are
load-bearing. Preserve A2.2's pure compiler and inert `ContractBundle` model.

Root is the sole writer and sole Git/didrun operator. Parallel agents are
read-only spec, architecture, and adversarial critics. Use `GOMAXPROCS=2` and
serial Go package execution where practical so this unit can coexist with the
owner's other local work.

## Required starting state

Do not edit until all of these are true:

1. P07B-A1, P07B-A2.1, cumulative-verification maintenance, P07B-A2.2 source,
   its corrected descendant, and its receipt-document boundary are committed,
   sealed, Git-note-present, and strict-clean.
2. `git status --short --branch` is clean on the expected branch.
3. `/opt/homebrew/bin/node tools/verify-current.mjs` passes through
   `didrun run --` on the starting tree, or an existing sealed exact-tree
   receipt covers that tree and a non-load-bearing read-only inspection shows
   no drift.
4. The exact A2.2 `PreparedBundle` and `ContractBundle` APIs are understood;
   no raw bundle parser is treated as publication authority.

Read before implementation:

- `docs/CONCEPT_BRIEF.md`
- `docs/ARCHITECTURE.md`
- `docs/SEMANTICS.md`
- `docs/STATE_MACHINES.md`
- `docs/THREAT_MODEL.md`
- `docs/VERIFICATION.md`
- `docs/prompts/P07-U6-STANDALONE-CONTRACT.md`
- `docs/prompts/P07B-A2-COMPILATION-PLAN.md`
- `docs/prompts/P07B-A2-2-RECOVERABLE-COMPILER.md`
- `docs/status/P07B-A2-2-COMPILER.md`
- `docs/HANDOFF_MODE_C.md`
- current `internal/store`, `internal/choice/promotion`,
  `internal/emit/node`, and Darwin Git publication code.

## Objective

Implement exactly this authority flow:

```text
sealed PreparedBundle
  -> retained original PortableRulingPreparation
  -> exact current RULING HeadToken
  -> opaque node-issued residue-publication authority
  -> store full-token CAS before object/temp creation
  -> durable exact ContractBundle object reopen
  -> terminal RESIDUE head publication and reopen
  -> store-bound sealed Residue
  -> fresh terminal-residue reopen
  -> optional retryable exact six-file materialization
```

P07B-B ends at the materialized six-file bundle. It creates no Git execution
target, admitted Node runtime, conformance attempt, physical subject run,
`FinalizedContractRun`, `ContractExecution`, product CLI/server/studio, or
adoption claim. Those remain P07B-C or later work.

## Locked authority design

### Prepared authority stays private

`PreparedBundle` continues to retain its exact `PreparedCompilation`, which
continues to retain the original `PortableRulingPreparation`. Do not expose
that preparation, its raw head token, the private compiler input, or a generic
publication constructor.

The application edge lives in package `internal/emit/node`, where it can see
the retained preparation without weakening the public A2.2 boundary. A parsed
or copied `ContractBundle`, canonical body, digest, DecisionRecord digest, or
study ID is inert and cannot reach production materialization.

### Opaque leaf authority

Follow the established hidden-issuer/public-alias pattern:

```text
internal/emit/node/internal/publication/
internal/emit/node/authority/
```

Only the node subtree may invoke the issuer. The store imports only the public
alias and validates its sealed construction. The authority binds, at minimum:

- exact predecessor DecisionRecord digest;
- exact `ContractBundle` digest and canonical bytes; and
- the fixed `ContractBundle` kind.

The architecture checker must prove no other production package imports the
hidden issuer and no generic constructor, mutable callback, interface, map, or
raw byte carrier can mint the authority.

### Typed store transition

Add only this semantic transition shape:

```go
func (*store.ObjectStore) AdvanceResidue(
    context.Context,
    store.HeadToken,
    nodeauthority.Publication,
) (store.HeadToken, error)
```

The method must require:

- a valid opaque node-issued publication authority;
- expected stage `RULING` and kind `DecisionRecord`;
- expected current digest exactly equal to the authority predecessor;
- strict reparse of the exact authorized bundle bytes at the authorized
  external digest; and
- one call into the existing raw owner transition for `StageResidue`.

The existing `advanceHead` lock and full-token equality remain the commit
point. The complete expected token is compared while holding the per-study
transition exclusion and before `publishLocked`, object temporary creation, or
head temporary creation. `AdvanceHead`, `ReplaceHead`, `ForceHead`, a generic
residue draft, and an opened-token-only transition remain forbidden.

### Publication application service

The production application API is conceptually:

```go
type PublicationDisposition string

const (
    PublicationCreated PublicationDisposition = "PUBLISHED"
    PublicationAlreadyCurrent PublicationDisposition = "ALREADY_CURRENT"
)

type PublicationResult struct {
    Residue Residue
    Disposition PublicationDisposition
}

func PublishPrepared(
    context.Context,
    *store.ObjectStore,
    PreparedBundle,
) (PublicationResult, error)

func OpenResidue(
    context.Context,
    *store.ObjectStore,
    store.StudyID,
) (Residue, error)
```

`PublishPrepared` must:

1. reject nil context/store or invalid prepared authority with no partial
   value;
2. compile nothing and write no output directory;
3. revalidate the retained original portable-ruling preparation;
4. open the exact study head named by that retained authority;
5. accept an already-terminal exact same residue only by calling the full
   restart `OpenResidue` path;
6. otherwise require the exact current `RULING` token, exact predecessor
   DecisionRecord, and exact prepared bundle joins;
7. issue the opaque leaf authority;
8. call `AdvanceResidue`; and
9. return only the fully reopened terminal residue plus the constructor-derived
   `PUBLISHED` or `ALREADY_CURRENT` disposition.

If `AdvanceResidue` reports a post-commit ambiguity, reconcile by reopening the
head and exact object. Return a valid residue only when the exact terminal
state reconstructs. If the exact ruling remains current, report publication
failure with no residue. If neither state can be proven, report a distinct
residue-publication ambiguity. Never infer rollback from an error alone.

Two callers with the same prepared authority may converge on the same exact
terminal residue; exactly one reports `PUBLISHED` and a reconciled loser or
replay reports `ALREADY_CURRENT`. A replay must not create revision 9. A
different or foreign authority never becomes an alternate terminal successor.

## Restart reconstruction contract

`OpenResidue` does more than parse the current object. It must freshly verify:

1. exact study head at terminal revision/stage/kind;
2. exact current `ContractBundle` object and store authority;
3. strict `ContractBundle` recovery at the head's external digest;
4. head previous-object digest equals the bundle's DecisionRecord digest;
5. exact predecessor DecisionRecord object exists and strictly parses;
6. exact Choicepoint object named by both the DecisionRecord and bundle exists
   and strictly parses;
7. exact FreshConfirmation object named/embedded by that Choicepoint exists and
   agrees byte-for-byte; and
8. every reopened object authority belongs to the supplied store instance.

The resulting sealed `Residue` is bound to the store instance, study,
predecessor DecisionRecord, bundle digest/body, and complete terminal head
token. `Residue.Valid()` proves sealed construction only. Every materialization
attempt freshly reopens the residue and requires exact equality before it
touches a destination.

Removing, replacing, corrupting, or cross-pairing the current bundle or any
predecessor-chain object makes reopen and materialization refuse.

## Publication crash and concurrency semantics

The only honest durable-state model is:

| Boundary | Allowed durable state |
| --- | --- |
| before object publication | exact `RULING`; no successor object |
| object durable, before head publication | exact `RULING` plus a possibly unreachable immutable bundle object |
| terminal head durable | exact `RESIDUE` naming an existing exact bundle |
| post-head error | ambiguous until exact reopen reconciliation |

Do not call this an atomic transaction or claim object rollback. A head must
never name a missing bundle. A stale loser reaches the full-token comparison
before successor object or temporary creation and creates no output directory.

Test both in-process and cross-process store-instance races. Deterministic
compilation means same-ruling competitors normally name the same bundle; pair
that convergence test with the lower-level distinct-successor CAS test so the
no-losing-object claim is not inferred from content identity alone.

## Locked materialization design

### Structural gate

Only a freshly reopened terminal `Residue` may enter the production
materializer. No exported materializer accepts `model.ContractBundle`,
canonical bytes, digest, path roster, callback, or interface. Keep the
filesystem owner in the cycle-free sibling package:

```text
internal/contractmaterialize/
```

That package may import the node residue API and store, while the node package
must not import it. Its unexported filesystem core may consume only the exact
bundle defensively recovered from a freshly reopened residue. This keeps OS,
path, syscall, and cgo authority out of the compiler/application package and
prevents any raw bundle API from reaching the official output edge.

The public application edge is conceptually:

```go
type Disposition string

const (
    Created Disposition = "CREATED"
    AlreadyExact Disposition = "ALREADY_EXACT"
)

func contractmaterialize.Materialize(
    context.Context,
    *store.ObjectStore,
    node.Residue,
    string,
) (contractmaterialize.MaterializedContract, error)
```

`MaterializedContract` is an opaque live receipt with a constructor-derived
`CREATED` or `ALREADY_EXACT` disposition, not a seventh bundle member, not a
durable semantic object, and not execution authority.

### Destination contract

For the native claimed path:

- destination is non-root, clean, absolute, and contains no control text;
- its parent exists, is a real directory, is already symlink-resolved, and
  retains the same opened identity through publication;
- destination and parent do not overlap the object store's owned root in either
  direction; the store exposes only the narrow path-overlap refusal needed to
  enforce this, not a mutable root or path-construction API;
- the parent may be nonempty, but staging is a new private mode-`0700`
  same-parent directory;
- final destination is absent or one exact real mode-`0700` directory;
- the final directory contains exactly the six fixed flat names and no extras;
- each member is a real nonsymlink regular file, has one link, physical mode
  `0644`, exact byte count, exact raw SHA-256, and exact content;
- no receipt, lock, temporary, cache, metadata, or marker file enters the final
  directory; and
- alternate case names and filesystem aliases refuse.

An already-existing exact directory is success without rewrite: preserve every
inode, byte, and mode. A partial, changed, extra, linked, aliased, wrong-mode,
or wrong-kind destination is an immutable refusal; never repair, merge,
delete, rename aside, or overwrite it.

### Staging and publication

For a missing destination:

1. freshly reopen and verify the residue;
2. retain the parent directory identity;
3. allocate a private same-parent staging directory;
4. create every fixed file exclusively without following links;
5. write all bytes with short-write/zero-progress handling;
6. apply physical mode `0644`;
7. sync and close every file;
8. reopen and verify type, link count, mode, count, hash, and bytes;
9. verify the exact six-entry stage roster and sync the stage directory;
10. freshly reopen the residue again immediately before publication;
11. revalidate the parent identity and destination state;
12. publish with Darwin+cgo `RENAME_EXCL | RENAME_NOFOLLOW_ANY` only;
13. sync the parent directory; and
14. reopen the exact final directory before returning success.

Never fall back to ordinary rename on an unsupported platform or toolchain.
Only Darwin/arm64 with the exercised cgo policy may receive the P07B-B native
receipt; every other tuple returns a typed unsupported/unreceipted refusal.

### Retry, cancellation, and ambiguity

Before exclusive rename, cancellation or failure cleans only the current
call's private stage. If cleanup durability cannot be established, report it
without changing the terminal head or touching an existing destination.

Once exclusive rename begins, finish the exact parent-sync/reopen reconciliation
despite cancellation. A parent-sync or final-reopen failure after rename is
`EXPORT_PUBLICATION_AMBIGUOUS`, not presumed absence. A retry reopens the
terminal residue and destination: exact existing output may be re-synced and
accepted; absence may be rebuilt; mismatch remains immutable refusal.

The closed publication/export outcomes distinguish at least:

```text
RESIDUE_PERSISTED_EXPORT_CREATED
RESIDUE_PERSISTED_EXPORT_ALREADY_EXACT
RESIDUE_PERSISTED_EXPORT_INCOMPLETE
RESIDUE_PERSISTED_EXPORT_AMBIGUOUS
RESIDUE_PUBLICATION_AMBIGUOUS
```

A combined convenience operation may return a valid persisted residue beside
an export error, but the state tag must make that durable truth explicit. The
retry path is always `OpenResidue` plus `MaterializeResidue`: no recompilation
and no head mutation.

## Required implementation surface

Expected ownership, adjusted only when the architecture critic proves a
better cycle-free topology:

```text
internal/choice/promotion/
  residue.go
internal/emit/node/
  publication.go
  authority/authority.go
  internal/publication/authority.go
internal/contractmaterialize/
  materialize.go
  publish_darwin.go
  publish_unsupported.go
internal/store/
  head.go
  public_api_test.go
tools/
  check-p07b-b-architecture.mjs
  check-p07b-b-architecture-selftest.mjs
```

Keep the A2.2 compiler, bundle schema, generated six files, Node assets, parity
corpus, and human-surface capture byte-for-byte unchanged unless a separately
documented correctness defect requires a new independently receipted correction.

## Required positive matrix

Publication and reopen:

- real CLI `ALLOW_OBSERVED` prepared authority;
- real child-bind HTTP `CUSTOM_EXPECTATION` prepared authority;
- exact bundle object durable before terminal head;
- exact terminal head and full predecessor-chain restart reconstruction;
- same prepared authority replay returns the same residue without revision or
  object-inventory change;
- two store instances and two subprocesses converge on one terminal residue;
- caller mutation of returned bytes or getters cannot alter authority; and
- full restart needs no compiler input, corpus, source file, or product output.

Materialization:

- CLI and HTTP residues produce the same fixed six-path policy and their own
  exact bytes at two nonnested destinations;
- newly created output has exact root/member modes, counts, hashes, bytes,
  roster, and no extras;
- exact-existing retry preserves root and member inodes;
- injected post-residue pre-export failure, store/process restart, exact reopen,
  successful retry, unchanged terminal head;
- concurrent same-residue/same-destination calls converge on one creator plus
  exact-existing successes;
- same residue at different destinations succeeds independently; and
- a destination-appears race reconciles only through exact verification.

## Required refusal and fault matrix

Authority/publication refusals:

- zero/invalid `PreparedBundle`, parsed-only bundle, raw body/digest, foreign
  store, foreign study, cross-paired predecessor, wrong stage/kind, stale head,
  cancelled context, and mismatched bundle joins;
- each expected-head field mutation at the store boundary;
- full-token change after application revalidation but before store CAS;
- no successor object/temp/output on pre-CAS refusal;
- missing/corrupt bundle, DecisionRecord, Choicepoint, or FreshConfirmation on
  restart; and
- exact classification of object/head absent, durable, and ambiguous fault
  states without claiming rollback.

Filesystem refusals:

- relative/unclean/root/control-text destination;
- missing, replaced, symlinked, or non-directory parent;
- final root as file, symlink, FIFO, socket, device, or wrong-mode directory;
- missing/extra/alternate-case member;
- member symlink, hardlink, directory, FIFO/socket/device, wrong mode, wrong
  count, wrong digest, or changed bytes;
- parent identity change and destination-appears races;
- different residue racing for one destination; and
- unsupported platform/toolchain with no ordinary-rename fallback.

Use unexported operation tables only below the low-level materializer to inject
create, short-write, zero-progress, chmod, file-sync, close, reopen, stage-sync,
exclusive-rename, parent-sync, final-reopen, and cleanup failures. Production
callers receive no fault hook.

After every refusal, snapshot and compare the relevant store/output inventory.
Do not call a refusal a caught fault unless the exact adverse condition was
physically exercised.

## Architecture and verification gates

Add a P07B-B checker and hostile self-test that preserve the inherited pure
compiler boundary while proving:

- only the node application owner can issue publication authority;
- store imports only the leaf alias, never node service/materializer;
- `AdvanceResidue` is the only new typed head edge and performs exactly one raw
  transition after strict authority checks;
- full-token comparison still precedes object/temp creation;
- no output API is reachable from a raw/parsed bundle;
- production materialization always performs two fresh residue reopens and
  rejects a supplied residue that differs from either reopened authority;
- low-level materializer imports no process, Git, network, runtime execution,
  package-manager, or product-surface package;
- Darwin exclusive/no-follow publication has no fallback;
- existing exact output is verified, never rewritten;
- P07B-C target/run/classification packages remain absent; and
- no current checker or mutation driver is silently orphaned.

Evolve the cumulative verifier explicitly. If an older architecture checker
owns an exact historical package map that legitimately predates B, preserve its
historical script/receipt in Git and make the new current B checker carry the
replacement boundary. Do not merely suppress, skip, or relabel a failing gate.

Run through `didrun run --`, with immediate successful-event claims:

- formatting and syntax;
- focused publication/reopen suites;
- focused materializer and durability fault matrix;
- real CLI and HTTP authority integration;
- restart and cross-process races;
- race-enabled focused packages;
- store public-API closure;
- P07B-B architecture checker and hostile self-test;
- complete Go suite twice under stock macOS `TMPDIR`;
- Go vet;
- one-command cumulative verification;
- exact staged inventory/diff;
- scoped structured credential-pattern scan; and
- serialized didrun chain integrity.

At the shippable boundary, declare exact claims, commit, seal, independently
confirm the didrun Git note, and loop `NO_COLOR=1 didrun verify --strict` until
exit `0`. Never delete, weaken, rename, or reframe a failed claim. Generate the
required exact-final-commit HTML report later at the final closure boundary.

## Claim ceiling

If all gates pass, the strongest allowed P07B-B sentence is:

> On the named local Darwin/arm64+cgo tuple, Countershape durably reopened one
> exact terminal ContractBundle residue and created or retried its exact six
> files under the exercised exclusive no-follow filesystem policy.

P07B-B does not establish an atomic object/head transaction, garbage collection
of unreachable objects/stages, power-loss behavior beyond the exercised sync
policy, NFS or other filesystem semantics, Linux/Windows support, hostile
same-user isolation, authenticity, authorship, confidentiality, coordinated-
replacement resistance, sandboxing, external-network denial, runtime
conformance, standalone target absence, product usability, production
readiness, security review, adoption, release/signing, or maintainership.

## Stop conditions

Stop and narrow honestly if any of these becomes true:

- the store transition requires a generic head mutation or raw bundle authority;
- output can be created before terminal residue reopen;
- restart cannot reconstruct the full predecessor chain;
- exclusive no-follow create-new publication is unavailable on the claimed
  native tuple;
- a sync/cancellation error must be mislabeled as rollback or absence;
- existing mismatched output must be overwritten to make a test pass;
- the current architecture boundary can pass only by disabling inherited
  checks; or
- strict didrun verification remains nonzero.

In that case, commit and receipt only the narrowest true fallback, update Mode
C with exact remaining work, and leave every stronger capability
`UNRECEIPTED`.
