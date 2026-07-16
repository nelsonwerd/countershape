# P07 implementation-time red team — portable rulings before standalone residue

**Date:** 2026-07-15
**Decision:** conditional GO after a mandatory P07A/P07B split
**Confidence:** 9/10 on the authority correction; 7/10 on eventual standalone parity until the physical Node corpus passes

## Executive ruling

The original P07 prompt cannot honestly compile the sealed U6 ruling into a selected-field standalone contract. U6 fresh construction records `WHOLE_EXACT_CANONICAL_PROJECTION_V1`, so the only selectable field is `countershape.exact_projection`. A contract emitted from that authority would assert every projected field while claiming that unselected fields may vary. The implementation would look polished but its predicate would be false to its provenance.

The correction is not a scope cut. P07 becomes two independently sealed units:

1. **P07A — adapter-bound portable ruling.** Preserve every existing CLI/HTTP projection byte and fingerprint. Strictly translate those historical adapter wires into one bounded portable value algebra, create new selected-field Choicepoints and DecisionRecords under `ADAPTER_BOUND_PORTABLE_FIELDS_V1`, preserve legacy whole-projection records as parseable nonportable history, and correct custom expectations to contain exactly the selected fields.
2. **P07B — standalone residue.** Require both a current store-bound portable ruling and a byte-complete live source-reconstruction witness. Add the HTTP child-bind readiness profile, compile one recoverable six-file Node-core bundle, advance `RULING -> RESIDUE` only after a full-token stale check, materialize retryably, and publish later capability-constructed targets, target-bound finalized runs, and classifications as separate immutable nonhead evidence without changing the terminal study head.

This is the smallest architecture that preserves the sealed U3–U6 evidence while making the standalone claim constructible.

## Findings by lane

### 1. Ruling authority

`internal/choice/choicepoint.go` always builds `NewWholeProjectionRegistry`. Existing DecisionRecords therefore authorize only whole-projection equality. They cannot be reinterpreted as authority over `http.status`, `cli.stdout.json.mode`, or any other selected field.

Locked response:

- widen the existing Choicepoint mode with `ADAPTER_BOUND_PORTABLE_FIELDS_V1` without changing the 27-member body;
- make fresh public construction use the portable mode;
- keep the old mode reachable only through strict legacy parsing;
- derive tuples only by verifying the original confirmed projection proof and then translating its exact adapter wire;
- refuse legacy whole-projection emission with `LEGACY_WHOLE_PROJECTION_NOT_PORTABLE`.

### 2. Fingerprint stability

The CLI and HTTP adapters intentionally emit different strict wire shapes. Replacing them with a shared envelope would change every projection fingerprint, outcome map, confirmation, Choicepoint, DecisionRecord, fixture, and sealed receipt downstream of U3/U4. That migration buys no necessary authority.

Locked response: preserve the existing adapter emitters byte-for-byte. The portable tuple is a versioned downstream interpretation, not a replacement historical observation. Its profile identity binds the exact projection-definition binding plus the ordered adapter-owned field descriptors. It never replaces or masquerades as `ProjectionDefinitionBinding.FieldRegistryDigest`.

### 3. Runnable source reconstruction

CLI and HTTP canonical stimuli retain lengths and digests for stdin, body, fixture, and seed content, but not all raw bytes. A ruling cannot reverse those digests into a runnable fixture. Hardcoded fixture values would silently invent evidence.

Locked response: P07B accepts a second authority, `PortableSource`, whose closed constructor retains exact bounded bytes and reconstructs the adapter stimulus, execution binding, projection binding, and launch facts. Before any compiler output exists it must exact-match the ruling's Choicepoint plan, minimized stimulus, adapter, and derived portable profile. Missing source returns `PORTABLE_SOURCE_REQUIRED` with zero artifacts. Successful residue embeds all recoverable source bytes.

### 4. HTTP portability and parity

The sealed HTTP world starts a child from an inherited listener descriptor. A Node-only parent has no reference-quality portable public API for creating and exporting that listener descriptor. Separately, Node's high-level `http` client normalizes header whitespace and therefore is not the same wire authority as Countershape's strict Go HTTP/1.1 parser.

Locked response:

- keep inherited-listener history valid but nonemittable;
- add a new exact portable profile in P07B: the child binds literal `127.0.0.1:0`, reports the allocated port through one bounded inherited readiness pipe, closes that pipe, serves one request, and tears down;
- execute a new confirmation and ruling under that profile before HTTP emission;
- generate the standalone HTTP harness with `node:net`, exact request bytes, bounded response bytes, and the same strict HTTP/1.1 grammar as Go.

### 5. Object identity and recoverability

The planning `ContractBundle` and `ContractExecution` schemas placed `artifact_digest` inside the body being hashed. That is circular and contradicts Countershape's external typed-digest rule. The bundle retained file digests but not file bytes, so a reopened terminal residue could not rematerialize its own output. An absence receipt inside deterministic source is also causally circular because it can exist only after the source runs. The later implementation-feasibility pass found a second authority flaw: the first correction attempted to reuse a historical 2–4-candidate `WorldInstance` for an arbitrary later single target and admitted a generic repository executable never authorized by the CLI lineage.

Locked response:

- keep bundle, execution-target, finalized-run, and execution digests external to their canonical bodies;
- embed exact base64 contents, path, mode, byte count, and raw-byte digest for all six files in the bundle;
- have `manifest.json` cover the other five files and exclude itself; have the outer ContractBundle cover all six;
- store no execution or didrun receipt inside deterministic bundle source;
- treat integrity as tamper detection, never authorship or authenticity.

### 6. Publication, materialization, and later executions

`RESIDUE` is terminal. Repeated conformance runs cannot mutate it or advance to another stage without contradicting the fixed study spine.

Locked response:

- compile all six files in memory;
- under the store's exclusive study transition, revalidate the complete expected head token before creating any object or temporary file, durably publish the bundle, then advance the head; a stale loser publishes nothing, while a crash after object publication but before head publication may leave only an unreachable content-addressed object;
- create no final executable directory before the terminal residue is durable;
- materialize from the committed bundle through a private same-parent temporary directory and exact reopen/hash checks, then create-new rename;
- publish/reopen a separate immutable `ContractExecutionTarget` before spawn, constructed only from the reopened bundle/source, one live Git-issued pinned/inspected/verified private materialization, one fresh durable conformance attempt, and one admitted/revalidated Node runtime;
- after the physical attempt closes, publish/reopen a separate immutable `FinalizedContractRun` bound to that exact target and attempt, with closed lifecycle, constructor-derived clean/ineligible disposition, exact projected tuple when present, and scoped standalone evidence;
- make final `ContractExecution` reference only that target plus the exact target-bound finalized run and derived result class, independently author neither tuple nor control reason, never duplicate caller-pairable bundle/tree/attempt/runtime/lifecycle/observation fields, and never change the study head;
- keep `WorldInstance` historical and comparison-only; a parsed target, copied OIDs/digests, a fake one-member candidate set, or dirty working-tree bytes remain inert;
- initially support only the antecedent-backed logical `node` plus exact repository-relative JavaScript entrypoint. A generic repository executable requires a new physical lineage.

## Locked portable value profile

The shared semantic algebra is deliberately small:

- `MISSING`
- `NULL`
- `BOOLEAN`
- `INTEGER` with safe range and canonical decimal spelling
- `STRING`
- `BYTES` with exact canonical padded base64 on the wire
- `ORDERED_STRING_LIST`, preserving order, duplicates, empty list, and empty members
- `CANONICAL_JSON`, retaining exact canonical bytes rather than only a digest

Missing, empty string, empty bytes, empty list, and a present list containing one empty string are distinct. No regex, tolerance, case folding, substring, majority, first-outcome, name-based, or cross-product semantics enter the predicate.

## Corrected object graph

```text
sealed adapter projection bytes + exact projection binding
  -> strict versioned translator
  -> portable profile + complete portable tuples
  -> new adapter-bound Choicepoint
  -> selected-field DecisionRecord
  -> current store-bound promotion.Ruling

promotion.Ruling + exact PortableSource reconstruction witness
  -> pure in-memory compiler
  -> recoverable ContractBundle
  -> terminal RESIDUE head
  -> retryable materialization

ContractBundle + live single-target Git capability + fresh attempt + admitted Node
  -> immutable ContractExecutionTarget (published/reopened before spawn)
  -> exact FinalizedContractRun (published/reopened after lifecycle closure)
  -> immutable ContractExecution classification (never a head transition)
```

## Mandatory negative cases

P07A must kill profile substitution, digest-only resolution, proof-derived field invention, ignored CLI roster/order, ignored HTTP unused slots, bytes-through-UTF-8 coercion, ordered-list sorting/deduplication, missing/empty collapse, caller-authored tuple injection, selected-field widening, unselected custom-expectation smuggling, portable records falling back to whole projection, and legacy records rebuilding under portable authority.

P07B must kill source bytes replaced by digests, omitted fixture/seed/body bytes, plan/stimulus/profile mismatch, inherited HTTP mode labeled portable, high-level HTTP normalization, stale publication that writes an object or output directory, manifest self-coverage, digest-only unrecoverable files, outer-bundle acceptance of any changed six-file member before publication, intact-entrypoint acceptance of a companion-file mismatch against an unchanged manifest before subject spawn, ambient executable lookup, shell/package-manager/external-host use, generic repository-executable resurrection, copied/recomputed target digests, OID/tree-identity disagreement, missing or substituted materialization authority, parsed-target execution, reused/nonconformance attempts, runtime executable/probe substitution, dirty working-tree execution, named parent-secret inheritance, target/run or attempt/run substitution, ineligibility flattened to contradiction, and execution evidence that advances or freshens the terminal study. Entrypoint/manifest self-authentication and coordinated post-materialization replacement remain explicit nonclaims.

## Stop conditions

Stop at a selected-field DecisionRecord without executable residue if any of these remains true:

- a portable tuple is not bound to exact historical projection bytes and the exact projection definition;
- raw runnable source cannot reconstruct the exact minimized stimulus;
- a legacy whole-projection ruling reaches compilation;
- Go and Node disagree on bytes, tuple, or classification;
- Node HTTP bytes do not obey the Go wire profile;
- a stale publication creates a final output;
- the terminal bundle cannot recover all six exact files after restart;
- an eligible result can exist without a projected observation;
- materialization failure is not explicitly retryable from the durable residue;
- an arbitrary later target can be routed through `WorldInstance`, constructed from copied fields, or spawned before its immutable target is published and reopened;
- initial P07B accepts anything broader than the antecedent-backed Node repository-script launch;
- later execution mutates the terminal study head.

## Honest confidence

- **9/10 — authority split.** Multiple independent code, schema, compiler, Node, and acceptance reviews converged after explicitly resolving their initial disagreement about emitter migration.
- **9/10 — preservation of sealed evidence.** Keeping historical adapter bytes and the 27-member Choicepoint shape prevents gratuitous fingerprint invalidation.
- **8/10 — store/materialization direction.** The stale-before-publication and durable-object-before-head ordering is coherent, but crash/fault injection must validate the exact implementation.
- **7/10 — Go/Node parity feasibility.** The narrow profile is implementable, but raw HTTP parsing, signal taxonomy, and strict JSON edge cases remain empirical blockers.
- **3/10 — cross-platform support today.** Only physically executed Node/OS tuples may be receipted. Linux, Windows, and uninstalled Node majors remain `UNRECEIPTED` regardless of source intent.
