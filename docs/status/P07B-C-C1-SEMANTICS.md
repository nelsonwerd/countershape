# P07B-C C1 inert-semantics status

**Source state:** C1 implementation working tree. Every C1 capability below remains `UNRECEIPTED` in source until the exact C1 commit is sealed, its didrun Git note is present, and `NO_COLOR=1 didrun verify --strict` exits `0`.

**Immediate sealed predecessor:** cumulative output-cap maintenance source `fa3d0c12b4c599744b666b2848e38a2499f33a89`, tree `c908c4e580144aa481f646090a1a21d92c86e1cf`, is note-present and strict-clean with `8/8 TREE-EXACT`; its receipt reconciliation `3f31370a70979c4c5fe3a523be19e05f84271e96`, tree `29c5708b05c562984a0b7b66d6cfe3ac6e19c626`, is note-present and strict-clean with `6/6 TREE-EXACT`. The source changes only one Darwin test stimulus, its fixture, and its status file; the receipt changes only that status file. Neither changes runtime control or a C1 semantic path.

## Boundary

C1 implements the strict, inert canonical algebra for three immutable nonhead bodies:

- `ContractExecutionTarget` contains canonical fields representing one exact bundle/residue graph, pinned-tree identity, conformance-attempt identity, boot-session identity, and a Node runtime tuple. C1 checks their closed shape and internal correlations; future C3/C4 authority must establish the represented Git, boot, and runtime facts physically.
- `FinalizedContractRun` contains canonical target/attempt/StartClaim references plus one closed witness with spawn, process-control, five-domain standalone-scope, and bounded private-manifest summary fields. C1 checks semantic correlations only; future C4 owns the represented physical observations and chronology.
- `ContractExecution` binds the exact target/run plus the literal `CONTRACT_EXECUTION_EXACT_TUPLE_V1` profile and encodes only the constructor-derived `CONFORMS`, `CONTRADICTS`, or `INELIGIBLE_EXECUTION` result. C1 does not persist or publish it.

The only production package is `internal/contractexec/model`. Values can be constructed, strictly parsed, canonically encoded, domain-hashed, compared, and classified from already-held semantic values. The package has no store, Git, materialization, runtime, process, network, clock, randomness, publication, spawn, CLI, server, or study-head authority. No production package imports it during C1. The staged C1 boundary names all nine production and five test files exactly; prefixes are empty, so an extra, missing, or renamed model test cannot hide behind a directory allowance.

## Locked semantics

- Process closure is either clean or controlled. Controlled closure owns at most one primary reason plus independent teardown and orphan controls; cleanup cannot erase the primary cause.
- Capture is exactly `NO_CAPTURE`, `CAPTURED_UNPROJECTED`, or `PROJECTED`. Tuple presence is equivalent to `PROJECTED`, even if another axis later makes the run ineligible.
- Standalone scope closes exactly target inventory, child bindings, import resolution, service bindings, and named parent-secret sentinel inheritance as `COMPLETE`, `PARTIAL`, or `VIOLATED`. An observed violation outranks missing evidence.
- Only clean projected process closure plus complete unviolated scope is eligible for exact complete-tuple membership. Every other legal closed run derives `INELIGIBLE_EXECUTION`.
- The checked classifier-profile digest is `sha256:789b54457f24e19a181da1e0d44f45a491d21c5550b622de0f0ef19ec15412f4`. It is the future C2 relationship key derived from the canonical schema/kind/literal-profile declaration; it is not serialized in `ContractExecution`, caller-supplied authority, or an implementation-byte commitment.
- Target runtime facts are checkpointed path/content identity: absolute admitted path, executable-byte digest, executable mode/count, measured `process.execPath`, version/major, platform/architecture, and owned-probe facts. They do not prove which inode or descriptor a later spawn executes and do not close coordinated same-user substitution.
- All three closed start-error codes round-trip with exact ineligible witness structure. C1 validates code/reference shape only; C4 owns physical evidence content and chronology.
- Canonical object bodies remain within the existing 1 MiB parser ceiling. A closed-run witness is at most 256 KiB and 16 typed refs; private evidence is summarized as at most 16 blobs and 64 MiB aggregate, retained at finalization and omitted by default.

## Schema, examples, and proof shape

The three JSON Schemas are closed syntax projections. `internal/contractexec/model` is semantic authority. JSON Schema intentionally overapproximates evidence-kind/domain pairing, process/spawn/capture correlations, derived scope aggregates, violation precedence, and recursive ceilings; schema-only acceptance therefore grants no semantic capability.

The checked target/run/execution examples are canonicalized, strictly parsed with exact external typed digests and parents, and rebuilt byte-for-byte. The planning generator delegates semantic construction to an opt-in Go example probe instead of reimplementing the model in JavaScript. The fixed eight-body schema/runtime overapproximation corpus is `sha256:78f5a2c76d38dc2de96b11cf0439f33f73427160fc441ba89f9f39897dab6d4b`.

The C1 verification design uses:

- exhaustive 52,488-case process × cleanup × capture × scope algebra;
- canonical goldens, defensive-copy, typed-kind, mismatch, bounds, and invalid-combination tests;
- a byte-identical eight-case joint probe that first proves each inert mutant schema-valid and then proves the Go semantic parser refuses it;
- all-three start-error wire coverage;
- checked-example parse/rebuild parity; and
- three independently bounded single-worker parser fuzz targets; and
- an exact `go test -json` parser that requires every claimed suite/test/fuzz target to emit one matching run plus pass record, with the two opt-in probes explicitly skipped only in the full model-suite profile.

The source-rewriting recipe corpus abandoned during planning is not recreated. C1 architecture defense uses Go package/dependency/importer metadata and in-memory fact mutations. Planning-validator mutations operate on inert parsed values rather than emitting a catalog of source-edit recipes.

## Multi-pass adversarial changes

The different-model critic changed the source boundary rather than rubber-stamping it:

1. classifier-profile linkage is now documented as a derived literal-profile key, not a hidden body field or implementation digest;
2. schema overapproximation is explicit and covered by runtime refusal tests;
3. runtime identity is narrowed to checkpointed path/content facts with the substitution race preserved for C3/C4;
4. all three start-error codes receive wire-level coverage while physical content/chronology stays a C4 nonclaim;
5. checked examples cross the actual Go parser/rebuilder; and
6. each parser receives a bounded fuzz pass with `GOMAXPROCS=2`, Go `-p=1`, and fuzz `-parallel=1`; and
7. the first cumulative rehearsal exposed P07B-B's obsolete total-absence C premise. The B checker remains current and keeps every sealed B invariant, but now admits C symbols only inside the exact inert model prefix after the complete C1 architecture gate passes; missing C1 closure, lookalike prefixes, and foreign C surfaces remain hostile self-test cases. Composition is one-way `B -> C1`, so C1 remains the single exact topology authority and recursion is forbidden;
8. every closed schema object now has an exact required-member roster, with independent architecture, plan, and nested-schema hostile deletions; and
9. the schema/runtime overapproximation claim now crosses the actual schema validator and an opt-in Go probe over the same eight canonical bodies, rather than inferring schema acceptance from runtime refusal alone;
10. the broad model-directory allowlist is gone: the exact 43-path roster has no prefixes and hashes to `sha256:6755b26a51dc71c4565fb1c600ba6282e63f44bff3521cecc1c8ba463c2882fe`; the added exact path is the sealed-maintenance status that C1 must reconcile rather than leave one receipt phase behind; and
11. the architecture boundary pins the five test filenames, test-import closure, and 22 top-level test/fuzz symbols, while final targeted receipts are rejected unless Go JSON proves the selected names actually ran and passed.

Every critic report and development pass is `UNRECEIPTED` unless it becomes the exact event immediately supporting a declared claim in the final C1 ledger.

## Intended C1 receipt map

| Intended capability | Exact claim label | Source grade |
| --- | --- | --- |
| Strict inert target/run/execution model | `P07B C1 inert semantic model` | `UNRECEIPTED` |
| Exhaustive closed-run/classification algebra | `P07B C1 exhaustive algebra` | `UNRECEIPTED` |
| Checked schema/runtime/example intersection and overapproximation refusals | `P07B C1 schema runtime example intersection` | `UNRECEIPTED` |
| Target parser bounded fuzz | `P07B C1 target parser bounded fuzz` | `UNRECEIPTED` |
| Finalized-run parser bounded fuzz | `P07B C1 finalized-run parser bounded fuzz` | `UNRECEIPTED` |
| Execution parser bounded fuzz | `P07B C1 execution parser bounded fuzz` | `UNRECEIPTED` |
| Exact production package/dependency/importer boundary | `P07B C1 architecture boundary` | `UNRECEIPTED` |
| Architecture checker defensive self-test | `P07B C1 architecture checker self-test` | `UNRECEIPTED` |
| Preserved P07B-B invariants plus exact C1 compatibility | `P07B C1 predecessor architecture compatibility` | `UNRECEIPTED` |
| Inherited B checker defensive self-test | `P07B C1 predecessor architecture self-test` | `UNRECEIPTED` |
| Go-authoritative checked-example generator | `P07B C1 planning generator parity` | `UNRECEIPTED` |
| Planning-validator compatibility and defensive mutations | `P07B C1 planning validator compatibility` | `UNRECEIPTED` |
| Durable C0 authority plus exact C1 projection coherence | `P07B C1 evolved plan coherence` | `UNRECEIPTED` |
| Evolved plan-checker defensive self-test | `P07B C1 plan checker self-test` | `UNRECEIPTED` |
| Cumulative-verifier orchestration self-test | `P07B C1 cumulative verifier self-test` | `UNRECEIPTED` |
| Exact staged C1 path and diff boundary | `P07B C1 exact staged scope and diff` | `UNRECEIPTED` |
| Scoped staged credential-pattern scan | `P07B C1 scoped staged credential scan` | `UNRECEIPTED` |
| Serial-Go cumulative repository verification | `P07B C1 cumulative verification` | `UNRECEIPTED` |
| Serialized didrun ledger chain | `P07B C1 didrun chain intact` | `UNRECEIPTED` |

## Explicit nonclaims

C1 does not prove official publication, store restart semantics, live Git identity, materialization, runtime admission/revalidation, executed-object identity, fresh attempts, boot-session truth, interlock correctness, at-most-once spawn admission, process lifecycle, start-error evidence content/chronology, teardown, orphan absence, five-domain physical scope, private-evidence confidentiality, classification publication, direct Node parity, product CLI, server, dashboard, or report.

Linux, Windows, other Node/platform tuples, host-wide absence, network/registry denial, listener ownership, descriptor-bound execution, coordinated same-user replacement resistance, containment, confidentiality, production readiness, independent security review, comprehension, adoption, and maintainership remain `UNRECEIPTED` human/engineering tail.

## Gate before C2

Archive the C1 development ledger intact. From a fresh ledger, run and immediately claim every final row above. The model-suite and exhaustive events must pipe `go test -json` through `--assert-go-json model-suite` and `--assert-go-json exhaustive-algebra`. Each fuzz event must use its exact parser profile, `-parallel=1`, and `-fuzztime=10000x`. The schema/runtime claim attaches to a full cross-language planning-validator self-test; planning-validator compatibility attaches to a separate rerun, never a second claim on the first event. Stage exactly 43 individually named paths, require the sorted newline roster digest `sha256:6755b26a51dc71c4565fb1c600ba6282e63f44bff3521cecc1c8ba463c2882fe`, and keep `prefixes` empty. The chain event validates the preceding 18 wrapper events; its immediately attached claim is the nineteenth and is closed only by the later seal/note/strict gate. Commit `feat: define P07B-C execution semantics`, seal the commit, require its didrun Git note, and loop `NO_COLOR=1 didrun verify --strict` until exit `0`. A nonzero strict gate leaves C1 unfinished and cannot be repaired by deleting, weakening, or relabeling a test or claim.
