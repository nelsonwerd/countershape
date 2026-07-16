# P07A — U6 adapter-bound portable ruling authority

Implement the prerequisite selected-field authority in a fresh chat. Read `docs/CONCEPT_BRIEF.md`, `docs/SEMANTICS.md`, `docs/PROJECTION_ALGEBRA.md`, `research/deep-dive/11-p07-implementation-red-team.md`, the prior deep-dive synthesis/red team, `docs/status/U6.md`, and the Mode C handoff. Verify the current U6 boundary is sealed and `NO_COLOR=1 didrun verify --strict` exits 0 before editing.

This unit repairs an implementation-time authority defect. It does not emit a standalone contract, expose `RULING -> RESIDUE`, add runnable source bytes, change either adapter's historical projection bytes, or begin the product CLI/server/studio.

## Objective

Replace fresh whole-projection Choicepoints with a construction-safe adapter-bound selected-field lineage while preserving every existing CLI/HTTP projection byte and fingerprint. Strictly translate exact confirmed adapter projections into one bounded portable value algebra, build candidate-neutral blind cards over complete portable tuples, and issue selected-field DecisionRecords for `ALLOW_OBSERVED` and separately reviewed `CUSTOM_EXPECTATION`.

Existing `WHOLE_EXACT_CANONICAL_PROJECTION_V1` Choicepoints and DecisionRecords remain byte-identical, strictly parseable historical objects. They are nonportable and must never be silently upgraded.

## Locked compatibility boundary

Do not refactor either adapter emitter onto a shared wire. `ProjectionFingerprint` is a digest of exact projection bytes; changing the CLI or HTTP envelope would invalidate sealed outcome maps, confirmations, Choicepoints, DecisionRecords, fixtures, and receipts.

Add exactly one fresh-construction mode:

```text
ADAPTER_BOUND_PORTABLE_FIELDS_V1
```

Keep the Choicepoint canonical body at the existing 27 members. The exact mode, embedded WorldPlan/projection binding, and embedded confirmation proofs already bind the interpretation. Do not add caller-supplied profile bytes or raw runnable fixture bytes to the Choicepoint.

Parsing dispatches by exact mode:

- `WHOLE_EXACT_CANONICAL_PROJECTION_V1` reconstructs only the sealed legacy whole-projection registry;
- `ADAPTER_BOUND_PORTABLE_FIELDS_V1` re-resolves the adapter profile and retranslates every confirmed proof;
- fresh public construction always emits the adapter-bound mode;
- an unknown mode refuses;
- no public option lets a caller choose legacy construction.

## Portable exact algebra

Own an adapter-neutral immutable algebra under `internal/portablevalue/**`:

```text
MISSING
NULL
BOOLEAN
INTEGER
STRING
BYTES
ORDERED_STRING_LIST
CANONICAL_JSON
```

`INTEGER` is limited to the existing safe-integer range and one canonical decimal spelling: no `-0`, leading zero, decimal, exponent, coercion, or float. `BYTES` retains exact bytes and uses canonical padded base64 only at wire boundaries. `ORDERED_STRING_LIST` preserves order, duplicates, `[]`, `["]`, and empty members. `CANONICAL_JSON` retains exact strict canonical bytes, not a digest substitute. Copy every caller-owned slice on input and output.

Enforce explicit per-value, list-count, aggregate-list, tuple, field-count, and total retained-byte ceilings that fit below the durable 1 MiB object ceiling after encoding. Missing, null, empty string, empty bytes, empty list, and a present list containing one empty string are all different identities.

Choice's historical four-member exact-value wire remains byte-compatible. Existing tags keep their existing slots. Encode new compatibility tags without changing old bodies:

- `BYTES`: canonical padded base64 in `text`, with the other payload slots at their exact zero values;
- `ORDERED_STRING_LIST`: base64 of the strict canonical JSON string-array bytes in `canonical_json_base64`, with the other payload slots at their exact zero values.

The future generated `CompiledDecision` may use a cleaner tagged DTO, but it must implement the same algebra exactly.

## Derived adapter profile and strict translation

Use construction-safe packages such as:

```text
internal/portablevalue
internal/projectionprofile
internal/projectiontranslate
```

`portablevalue` imports neither adapters nor Choice. `projectionprofile` owns the opaque derived field profile and its typed digest. `projectiontranslate` is the only generic-to-adapter branch and may import the CLI/HTTP adapters plus `projectionprofile`. `choice` consumes the translator/profile and never switches on adapter domain itself. Keep the exact import graph cycle-free and enforce it with the U6 architecture checker.

The derived profile identity binds:

- translator name/version;
- exact adapter domain;
- exact `ProjectionDefinitionBinding` digest and canonical bytes;
- exact ordered adapter-owned field descriptors, including channel, source path, value kind, and missing policy.

Its digest is a new downstream interpretation identity. It does not replace, equal, or relabel `ProjectionDefinitionBinding.FieldRegistryDigest`.

### CLI resolver

Resolve the active CLI definition only through adapter-owned construction. Enumerate the bounded 127 nonempty subsets of the seven canonical CLI field descriptors in registry order, reconstruct each with `cli.NewCLIProjectionDefinition`, and accept exactly one candidate whose full binding digest and canonical bytes equal the WorldPlan binding. Reject zero or multiple matches. Never resolve by adapter name, operation name, registry digest, or proof field roster alone.

Strictly parse the existing `cli-projection/v1` `CLIProjection` wire. Require exact canonical input; exact root members; exact active roster and registry order; exact field-object members; canonical base64; known tag/payload shapes; safe integers; and zero values in unused payload slots. Then translate to `portablevalue`.

### HTTP resolver

Reconstruct the fixed `http.NewHTTPProjectionDefinition` and require its complete binding digest and canonical bytes to equal the WorldPlan binding. Strictly parse the existing capitalized `HTTPProjection` wire exactly as emitted today: exact root names; exact four-field roster/order; exact six-member field objects; exact tags; zero values in unused payload slots; canonical JSON bytes; and ordered duplicate content-type values. Freeze this awkward historical wire as compatibility authority rather than normalizing it.

### Proof-first construction

For every confirmed candidate:

1. verify the candidate and original projection bytes against the confirmation's exact projection roster;
2. resolve one exact profile from the WorldPlan binding;
3. strictly translate those same verified bytes;
4. seal the complete portable tuple plus the original fingerprint authority.

No public constructor may accept a caller-authored tuple paired with a projection digest. The original adapter bytes remain the historical observation; the tuple is only their exact profile-bound interpretation.

## Choice, blind view, and ruling corrections

Derive a Choice field registry only from the sealed profile. Bind its typed identity to the profile digest, projection-definition digest, and exact ordered definitions. Extend Choice values with bytes and ordered lists without weakening existing constructors or fuzz limits.

Preserve these blind properties:

- full original projection fingerprints determine cards and aliases;
- distinct fingerprints remain distinct cards even if selected tuples match;
- aliases and order ignore candidate identity, source order, and support count;
- cards expose complete portable tuples, not candidate membership.

For a portable Choicepoint:

- `selectable_fields` is the complete profile roster in registry order;
- `differing_fields` contains exactly fields with more than one exact value across blind cards, also in registry order;
- selecting a nondiffering field is allowed as an explicit choice but fails separation with `AMBIGUOUS_SCOPE` when it cannot distinguish allowed from disallowed outcomes;
- equality uses exact value identity bytes, never display strings or digest-only comparison.

Correct `CUSTOM_EXPECTATION`: its expectation contains exactly one value for every selected field and no unselected field. Introduce a sealed selected-tuple type rather than requiring a registry-complete tuple. Registry-order canonicalization, distinct-from-every-confirmed-selected-tuple review, reviewer/evidence binding, and `CUSTOM_EXPECTATION_ALREADY_OBSERVED` remain mandatory. Legacy whole-projection behavior remains valid because its selected set has one whole field.

`ALLOW_OBSERVED` compiles only canonical complete selected tuples derived from confirmed outcomes. Allow-many never becomes independent per-field sets or a cross-product. `REJECT_ALL`, `DEFER`, and `REFINE` remain noncompilable.

## Legacy and future-emission boundary

Add a closed query or sealed preparation seam that lets P07B distinguish portable from legacy authority without parsing strings in the emitter. A legacy current ruling must produce exactly:

```text
LEGACY_WHOLE_PROJECTION_NOT_PORTABLE
```

This unit may test that refusal but must not yet accept source bytes, compile files, publish a ContractBundle, materialize output, or advance the store to `RESIDUE`.

## Required physical studies

Exercise real CLI and HTTP flows through fresh confirmation, Choicepoint, blind session, selected-field DecisionRecord, store promotion, restart, and reopen:

- CLI selects `cli.stdout.json.mode` while at least one diagnostic/context field remains unasserted;
- HTTP selects `http.status`, preserves ordered content-type evidence in complete tuples, and separately reviews custom status `401` using a selected-only expectation.

The current inherited-listener HTTP profile may prove P07A selected-field authority but remains nonemittable. P07B owns a new portable-start HTTP lineage.

## Verification and mutation gate

Run all load-bearing commands through `didrun run --`. At minimum:

- strict malformed/golden/property tests for every portable value;
- all 127 CLI profile subsets resolve uniquely;
- HTTP exact binding resolves and every near-match fails;
- both historical projection parsers reject noncanonical bytes, unknown members, roster/order drift, bad unused slots, invalid base64/JSON, unsafe integers, and resource overflow;
- legacy Choicepoint and DecisionRecord fixtures parse and rebuild byte-identically;
- fresh CLI/HTTP Choicepoints emit only the portable mode;
- bytes and ordered lists round-trip with empty/non-UTF-8/duplicate cases;
- selectable/differing fields and fingerprint-derived aliases are exact;
- selected-only custom expectation succeeds; missing, extra, unselected, or already-observed values fail;
- promotion/store restart reconstructs both modes without upgrading legacy history;
- the physical CLI and HTTP studies complete;
- fuzz both historical translators;
- full repository tests, race tests, vet, architecture checker, hostile checker self-test, and a genuine A/B/A semantic mutation suite pass.

Required mutants include resolution by domain alone, resolution by digest alone, registry inferred from proof bytes, skipped exact-canonical check, ignored CLI roster/order, ignored HTTP unused payload slots, list sort/join/deduplication, bytes through UTF-8, missing-to-empty collapse, all selectable fields hardcoded as differing, unselected custom-field smuggling, new construction falling back to whole projection, and legacy parsing rebuilt under portable authority.

## Stop conditions and handoff

Stop without P07B authority if exact binding resolution, historical fingerprint preservation, selected-field separation, legacy byte compatibility, physical selected-field promotion, or mutation closure fails. A schema that merely accepts the new tag is not runtime authority.

Stage only P07A paths, declare narrow claims from successful didrun events, commit without an AI co-author trailer, seal, and loop `NO_COLOR=1 didrun verify --strict` until exit 0. Update `docs/status/` and Mode C with exact event indexes, claims, verbatim grades, commit/tree, legacy nonclaims, and P07B's first command.
