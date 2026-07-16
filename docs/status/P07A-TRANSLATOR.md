# P07A translator-substrate status

- **Boundary:** P07A-A, the proof-first portable translation substrate; P07A/U6b Choice integration remains open.
- **Implementation commit:** `d274588b3aba26d2c29258dd42e167294d7c5a0f`
- **Implementation tree:** `2f2924e20dbe23559af755e475d353381fa08327`
- **Implementation-manifest result:** permanent strict failure, `15/31 claims recorded-exact`; the fifteen final rows are `TREE-EXACT`, while sixteen earlier exploratory claims are `STALE`.
- **Recovery boundary:** the follow-up documentation commit carrying this status file, `DIDRUN_BUGS.md`, `CONCEPT_BRIEF.md`, and `HANDOFF_MODE_C.md` is the accepted boundary only when its fresh serialized ledger has an intact chain and `NO_COLOR=1 didrun verify --strict` exits `0`.

## What the implementation establishes

The source commit adds three reviewed packages without changing either historical adapter projection wire:

- `internal/portablevalue` owns an immutable bounded algebra for `MISSING`, `NULL`, `BOOLEAN`, safe canonical `INTEGER`, exact UTF-8 `STRING`, exact `BYTES`, duplicate/order-preserving `ORDERED_STRING_LIST`, and strict exact `CANONICAL_JSON`;
- `internal/projectionprofile` derives an inert ordered interpretation identity from translator name/version, exact projection-binding digest and bytes, adapter domain, and exact adapter-owned descriptor semantics; and
- `internal/projectiontranslate` is the sole full-adapter branch. It freezes the seven-field CLI registry and resolves all 127 nonempty registry-order subsets by exact binding digest plus exact canonical bytes, reconstructs the fixed four-field HTTP definition, strictly parses both unchanged historical wires, and translates only copied proof bytes that all passed confirmation-roster verification first.

`projectiontranslate.ConfirmedTranslations` seals candidate identity, original projection fingerprint, copied original projection bytes, exact profile-bound tuple, confirmed outcome-map digest, and preservation-map digest. A caller cannot pair a bare profile or authored tuple with a projection digest and obtain that authority.

The architecture checker now includes all three package roots, enforces their exact import lattice, permits full CLI/HTTP adapter imports only in `projectiontranslate`, preserves World's model-only adapter imports, and admits exactly one external `projectionprofile.NewDerived` reference inside `newResolvedProfile`. Its self-test has 53 checksum-pinned clean/hostile cases.

The mutation gate expands from 17 inherited U6 faults to 29 reviewed faults. New semantic mutants cover domain-only and digest-only resolution, skipped exact-canonical input, ignored CLI roster order, ignored HTTP unused slots, sorted/joined/deduplicated ordered lists, bytes through UTF-8, missing-to-empty collapse, unknown missing-policy acceptance, and translation before complete proof verification. Every mutant must compile and receive a fresh baseline/mutant/post-control run; compilation failures are infrastructure failures, never semantic kills.

## Recovery receipt map

The fresh follow-up manifest is authoritative. These labels are intentionally narrow; each accepted row must display the verbatim grade below.

| Claimed capability | didrun claim label | Verbatim grade |
| --- | --- | --- |
| Exact 29-fault inherited-plus-translator A/B/A closure | `P07A translator recovery exact 29-fault A/B/A mutation closure` | `TREE-EXACT` |
| Full repository Go tests | `P07A translator recovery full Go repository suite` | `TREE-EXACT` |
| Race-enabled portable/profile/translator/Choice boundary tests | `P07A translator recovery race-enabled boundary suites` | `TREE-EXACT` |
| Full repository Go vet | `P07A translator recovery full Go vet` | `TREE-EXACT` |
| Active mixed historical-wire fuzzing | `P07A translator recovery strict mixed historical-wire fuzz` | `TREE-EXACT` |
| Active exact CLI byte fuzzing | `P07A translator recovery exact CLI bytes fuzz` | `TREE-EXACT` |
| Active ordered HTTP list fuzzing | `P07A translator recovery ordered HTTP list fuzz` | `TREE-EXACT` |
| Reviewed package/import/construction architecture | `P07A translator recovery architecture gate` | `TREE-EXACT` |
| Architecture hostile self-test | `P07A translator recovery architecture hostile self-test` | `TREE-EXACT` |
| Mutation-driver contract self-test | `P07A translator recovery mutation driver self-test` | `TREE-EXACT` |
| Staged whitespace | `P07A translator recovery staged whitespace` | `TREE-EXACT` |
| No unstaged source changes | `P07A translator recovery no unstaged changes` | `TREE-EXACT` |
| Exact four-path documentation inventory | `P07A translator recovery exact four-path documentation inventory` | `TREE-EXACT` |
| Scoped staged added-line credential-prefix scan | `P07A translator recovery scoped credential-prefix scan` | `TREE-EXACT` |
| Serialized didrun chain integrity | `P07A translator recovery didrun chain intact` | `TREE-EXACT` |

The source commit's successful final events are still mechanically exact, but its strict result is nonzero and the commit is not relabeled as strict-clean. Its complete chain-intact ledger, claims, seal, and failed strict state are preserved under `.didrun-history/2026-07-15-p07a-translator-stale/.didrun/`. The plain source seal stopped on 189 aggregate entropy findings; the logged redacted-export override followed an exact 18-path inventory and scoped credential-prefix scan. Neither that scan nor the override is a secret-free finding or security review.

## Permanent negative history

Failed source-ledger events remain unclaimed and support no capability:

- mutation self-test failures exposed exact-anchor defects in the new mutant definitions;
- the first full mutation invocation omitted the mandatory absolute Go/Clang declarations;
- two later mutation runs correctly rejected subtest-based kill targets and noncompiling mutants as infrastructure failures;
- an overescaped staged-inventory `awk` command failed before the corrected direct digest comparison; and
- the first strict verification of the source commit rejected sixteen stale exploratory claims.

Those failures were repaired by changing the harness or invocation, never by weakening a property, deleting a claim, shrinking a roster, or accepting compilation failure as a kill.

## Explicit nonclaims

This boundary does **not** make fresh Choicepoints portable. Production Choice still emits `WHOLE_EXACT_CANONICAL_PROJECTION_V1`, branches on adapter stimulus kind, uses the legacy whole-projection registry, and has no selected-only custom expectation. It does not implement real selectable/differing fields, portable DecisionRecords, the legacy nonportable preparation refusal, physical selected-field promotion studies, source reconstruction, standalone emission, `RESIDUE`, execution, CLI, server, studio, visual design, packaging, cross-platform behavior, hostile-code security, production readiness, adoption, or maintainership.

The next source boundary is P07A-B: preserve legacy Choicepoint/DecisionRecord bytes while making fresh construction consume only sealed `ConfirmedTranslations`, derive the Choice registry in profile order, expose exact differing fields, add selected-only custom expectations, remove adapter branching from Choice, survive store restart, and pass the required physical CLI/HTTP studies plus Choice-specific mutants.
