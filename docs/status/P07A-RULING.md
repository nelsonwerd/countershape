# P07A portable-ruling status

- **Boundary:** P07A-B / U6b, adapter-bound portable Choicepoint and selected-field ruling authority
- **Source commit:** `65005e498b05f3da62d6e30e108fb1cdba68d2af`
- **Source tree:** `18f52daa4f7b25b71d7fe3fda70c2aba561d29c9`
- **Strict didrun verdict:** `ALL RECORDED-EXACT`; `17/17 claims recorded-exact`
- **Reference host:** Darwin arm64, Go 1.26.5, CGO enabled; the physical study used the installed Node runtime on this host
- **Receipt-document boundary:** the follow-up documentation commit carrying this file and the controlling handoff is accepted only when its fresh serialized ledger is chain-intact, sealed, and strict-clean

## Implemented authority

Fresh public Choicepoint construction now uses `ADAPTER_BOUND_PORTABLE_FIELDS_V1`. It verifies every original projection proof against the confirmed roster before translation, resolves the one exact adapter-owned profile from the complete projection binding, and derives a complete portable tuple without changing the historical CLI or HTTP projection bytes or fingerprints. The profile and expectation-domain rosters freeze all 127 nonempty CLI field subsets plus the fixed HTTP profile.

Choice derives its field registry only from the sealed profile. Blind cards retain complete tuples and fingerprint-derived aliases; `selectable_fields` follows profile order and `differing_fields` contains exactly the fields whose exact values differ across cards. `ALLOW_OBSERVED` retains complete selected tuples rather than independent per-field sets. A sealed `SelectedTuple` lets `CUSTOM_EXPECTATION` contain exactly the selected fields, in profile order, with separate reviewer/evidence binding and no unselected context.

The expectation-domain check establishes only this narrow statement: the selected values are the restriction of at least one complete tuple in the exact adapter projection model. It does not establish that a current source, candidate, fixture, or runtime can emit them. CLI exit-kind/code/signal and selected stdout/JSON coherence are checked; HTTP status, body, metadata-object, and ordered content-type shapes are checked. The exact profile roster, expectation-domain roster, portable value ceilings, selection rule, and proof-first rule are bound into the portable-mode semantic digest.

Legacy `WHOLE_EXACT_CANONICAL_PROJECTION_V1` Choicepoint and DecisionRecord fixtures reopen and rebuild byte-identically. They remain valid, addressable, parseable nonportable history. Portable preparation refuses them with `LEGACY_WHOLE_PROJECTION_NOT_PORTABLE`; callers cannot request fresh legacy construction. The store-backed preparation seam rechecks the current `RULING` head and carries no source bytes, bundle bytes, or residue authority.

Resource admission is structural, not a hostile-input DoS proof. Candidate rosters remain bounded to two through four members. Alias, projection-proof, Choicepoint-receipt, and DecisionRecord-receipt sizes are admitted before the reviewed copy/sort sites. Portable proposals conservatively materialize a prospective DecisionRecord below a 768 KiB base ceiling and reserve 256 KiB for late actor, annotation, and receipt attribution; a later larger attribution may still refuse. Annotations remain exact inert durable bytes in this unit, including control and bidirectional characters. Every future CLI, dashboard, report, and generated README must escape them for its output context.

## Physical study boundary

The Darwin CLI study selects `cli.stdout.json.mode` while leaving diagnostic/context fields unasserted, promotes the ruling, restarts the store, reopens it, and obtains current portable preparation. The Darwin HTTP study selects `http.status`, retains ordered duplicate content-type evidence in complete tuples, separately reviews selected-only custom status `401`, promotes, restarts, reopens, and prepares the ruling.

Those studies do not establish that every portable value was promoted end to end. Custom CLI `BYTES` and HTTP ordered-list Decisions were exact canonical round-tripped, not promoted. Each physical package ran once on this host. The inherited-listener HTTP lineage remains structurally nonemittable and cannot satisfy P07B.

## Exact receipt map

Every grade below is copied verbatim from the source commit's strict report. A receipt records what ran against the sealed tree; it does not prove correctness or completeness.

| Event | Claimed capability | didrun claim label | Verbatim grade |
|---:|---|---|---|
| 112 | Mutation-driver contract self-test | `P07A-B 55-case mutation driver self-test` | `TREE-EXACT` |
| 113 | Fifty-five singular seeded faults distinguished by named tests with fresh A/B/A controls | `P07A-B exact reviewed 55-fault A/B/A mutation closure` | `TREE-EXACT` |
| 114 | Full Go repository suite under the recorded hermetic module mode | `P07A-B full Go repository suite Darwin arm64 hermetic module mode` | `TREE-EXACT` |
| 115 | Race-enabled domain, compare, portable-value, profile, translator, Choice, and promotion packages | `P07A-B race-enabled semantic boundary package suites` | `TREE-EXACT` |
| 116 | Full Go vet under the recorded hermetic module mode | `P07A-B full Go vet hermetic module mode` | `TREE-EXACT` |
| 117 | Ten-second strict mixed historical-projection translation fuzz campaign | `P07A-B bounded strict mixed historical projection translation fuzz` | `TREE-EXACT` |
| 118 | Ten-second exact CLI stdout-byte translation fuzz campaign | `P07A-B bounded exact CLI stdout bytes translation fuzz` | `TREE-EXACT` |
| 119 | Ten-second fixed-HTTP ordered content-type list fuzz campaign | `P07A-B bounded HTTP ordered content-type list translation fuzz` | `TREE-EXACT` |
| 120 | Darwin Node-fixture CLI and HTTP selected-field studies | `P07A-B Darwin physical CLI and HTTP selected-field studies` | `TREE-EXACT` |
| 121 | Static manifest over 105 exact Go files and five exact JSON authorities | `P07A-B reviewed 105-Go 5-JSON architecture manifest` | `TREE-EXACT` |
| 122 | Architecture checker over 120 checksum-pinned hostile synthetics and three clean/comment controls | `P07A-B architecture checker 120 hostile cases and 3 controls` | `TREE-EXACT` |
| 124 | Tracked candidate diff whitespace check | `P07A-B tracked candidate diff whitespace clean` | `TREE-EXACT` |
| 125 | Complete staged source diff whitespace check | `P07A-B staged source diff whitespace clean` | `TREE-EXACT` |
| 126 | Staged source path inventory | `P07A-B staged source path inventory captured` | `TREE-EXACT` |
| 127 | No unstaged tracked source changes | `P07A-B no unstaged tracked source changes` | `TREE-EXACT` |
| 128 | No untracked nonignored source paths | `P07A-B no untracked nonignored source paths` | `TREE-EXACT` |
| 129 | Staged tree equals the fully verified source tree | `P07A-B staged source tree equals verified tree 18f52daa4f7b` | `TREE-EXACT` |

The plain seal stopped on 60 aggregate high-entropy findings. A local masked-source classification found 39 workspace/toolchain path tokens in recorded argv and 21 checked-in mutation identifiers in event 113 output; didrun reported no structured credential-pattern finding. The source commit then sealed with the logged `--allow-secrets` redacted-export override. This classification is not a secret-free finding or a security review. Public export still requires a dedicated human secret scan.

## Permanent negative history

Failed mutation events `104`, `108`, and `111` remain unclaimed and support no capability. They exposed, respectively, a macOS private-root canonicalization mismatch, an oversized `%#v` diagnostic that exhausted the wrapper output channel, and the inherited store tests' `/var` symlink assumption. The harness and test diagnostics were repaired; the fault roster, properties, and kill requirements were not weakened. Earlier exploratory failures remain in the archived source ledger as engineering history.

## Explicit nonclaims

This boundary is not standalone emission. It does not accept runnable source bytes, construct `PortableSource`, add the child-bind HTTP lineage, compile or materialize a ContractBundle, publish `RESIDUE`, run Go/Node parity, or create ContractExecutionTarget, FinalizedContractRun, or ContractExecution. It adds no product CLI, server, dashboard, visual renderer, report, packaging, installer, external API, or model integration.

The receipts do not establish mutation completeness, broad future migration compatibility, whole-repository race freedom, comprehensive fuzz safety, repeatable physical behavior, cross-OS or cross-architecture portability, hostile-code isolation, confidentiality, production readiness, adoption, or maintainership. The architecture checker is a static lexical guard with tested hostile synthetics, not proof that no bypass exists.

## Next boundary

P07B / U6c is the only next implementation unit. Begin by running:

```text
NO_COLOR=1 /opt/homebrew/bin/didrun verify --strict
```

against the sealed documentation boundary, then read `docs/prompts/P07-U6-STANDALONE-CONTRACT.md`, this file, `docs/SEMANTICS.md`, `docs/PROJECTION_ALGEBRA.md`, and `research/deep-dive/11-p07-implementation-red-team.md`. P07B must consume the store-bound portable preparation seam; it may not reconstruct authority from a serialized DecisionRecord or weaken the legacy refusal.
