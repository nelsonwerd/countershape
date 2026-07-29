# Countershape claim vocabulary

- **Contract version:** U0 / `claim-vocabulary-v1`
- **Scope:** source, CLI, studio, export, documentation, release notes, and handoff receipts
- **Status:** controlling language contract; receipt state is capability- and unit-specific, and only a sealed per-unit receipt map may state a didrun grade

Countershape's credibility depends on saying exactly what one finite experiment established and refusing every convenient upgrade. This vocabulary is an API: product code emits these terms, schemas constrain them, tests compare them, and public prose may not replace them with stronger synonyms.

## Claim construction

Every positive release claim must identify:

```text
category + subject + operation/result + finite scope + environment + evidence + limitation
```

For example:

> Observed capability: On the named Darwin host, candidate execution key `k1` produced projection fingerprint `h1` in 3 of 3 new eligible trials under WorldPlan `p`, ComparisonEnvelope `e`, and stimulus `s`; didrun reference grade: `<verbatim grade>`.

A claim that omits scope, environment, or evidence is incomplete. A heading, badge, screenshot, exit code, or polished UI does not fill missing fields by implication.

### Claim categories

| Category | Meaning | Evidence form | Mandatory limitation |
| --- | --- | --- | --- |
| `OBSERVED_CAPABILITY` | the named implementation performed a bounded operation | product artifact plus exact didrun reference when receipted | applies only to named tree, command, fixtures, environment, and budget |
| `PRODUCT_BOUNDARY` | the system intentionally excludes or refuses a capability | controlling docs, schema, and negative tests as applicable | does not establish safety outside the boundary |
| `NOVELTY_ASSERTION` | Countershape differs operationally from named closest work | cited comparison and bounded search | composition differentiation, not algorithmic priority |
| `ENGINEERING_INFERENCE` | a design is plausible but not yet exercised | reasoning and planned acceptance gate | `UNRECEIPTED`; must not read as shipped behavior |
| `HUMAN_OR_MARKET_BET` | comprehension, review compression, adoption, or maintainership hypothesis | future human/market study | didrun cannot validate it |

Every release claim is exactly one primary category. Supporting references do not promote one category into another. In particular, an `OBSERVED_CAPABILITY` cannot become a novelty assertion, and a verified command cannot turn a human bet into a product fact.

## Receipt vocabulary

Countershape stores a didrun reference as opaque data associated with the exact command and Git state named by didrun. The display contract is:

- show the grade string verbatim, without normalization or friendly summary;
- preserve unknown future grade strings unchanged;
- show `UNRECEIPTED` when the capability itself was not run through didrun;
- do not borrow a related command's receipt;
- do not aggregate several grades into a stronger umbrella grade;
- do not translate a didrun grade into `CONFORMS`, `OBSERVED_STABLE(...)`, security approval, or production readiness; and
- retain failed historical receipts rather than rewriting or hiding them.

`UNRECEIPTED` is not failure and not success. It means no qualifying didrun reference has been mapped to that exact capability/environment claim.

When several commands are required, list each command and its verbatim grade. The claim remains unreceipted if a load-bearing part has no qualifying reference.

### Verification-governance terms

`SOURCE_FULL` means the boundary ran its declared full source battery; it does not upgrade that battery into a security, production-readiness, or market claim. `RECEIPT_RECONCILIATION` means the boundary reconciles only previously sealed source evidence. `NON_PRODUCT_MAINTENANCE` names a prospective narrower nine-claim consumer predeclared by a sealed `SOURCE_FULL` parent. The token is dormant in v25, C4L does not consume it, and v25 provides grammar and partial scope primitives rather than an executable runbook/preseal profile. It never follows from a candidate diff or prose assertion.

For that dormant profile, `NONE` is the product-authority value, `PARENT_FROZEN` is a prospective fail-closed eligibility token whose future final gate must reopen the exact sealed parent and prove no tracked change outside the parent-predeclared roster relative to it, and `INHERITED_UNREPROVEN` says product behavior was not re-proved at the child. `PARENT_FROZEN` is not presently an enumerated product-projection digest. None of those tokens transfers a verifier result, didrun event or grade, product correctness, or a behavior claim. A child may claim only the exact narrow command it ran; an inherited grade or cumulative/product label is prohibited.

`OWNER_OUT_OF_BAND` names only why an otherwise unpredeclared repository boundary is admitted. `CITED_UNAUTHENTICATED` requires a literal canonical `OWNER_INSTRUCTION_ARTIFACT` path and raw-byte SHA-256 for exactly one nonempty, at-most-64-KiB valid-UTF-8 mode-`100644` regular blob already present in the direct parent's tree. `UNEVIDENCED` requires the exact disclosure `OWNER_ATTRIBUTED_SESSION_INSTRUCTION_ONLY_NO_QUALIFYING_PREEXISTING_ARTIFACT`. Both retain `authentication: "NOT_ESTABLISHED"` and `signed_authorization: "NOT_IMPLEMENTED"`; citation is not authentication, and a same-unit artifact is not pre-existing authority.

## Canonical evidence terms

### Source and world

| Use this term | Exact meaning | Required qualifiers |
| --- | --- | --- |
| `TreeIdentity` | content-portable object-format/commit/tree identity | object format and commit/tree OIDs; repository fingerprint, Git version, and display ref are separate local provenance only |
| `supported entries materialized and verified` | named `100644`/`100755` blob bytes were written and rehashed under the policy | manifest, policy digest, portable tree digest, target-filesystem facts |
| `WorldPlan` | deterministic declared execution configuration | plan digest and fully materialized budgets |
| `CandidateExecutionKey` | stable wire/display reference for one candidate identity | exact key syntax; never evidence-allocation authority |
| `CandidateExecutionBinding` | opaque full candidate execution identity | tree, materialization, exact plan, adapter, runner, projection; actual matching plan required at allocation |
| `WorldInstance` | structural declared-consistency identity for one attempted subject | plan, candidate binding, stimulus, attempt artifact, purpose, nonce, ordinal; no runtime or freshness claim |
| `InstanceMeasurements` | one complete envelope-bound runtime row reported for a structural instance | envelope, instance, exact values and sources; no honest-host or admission claim by itself |
| `comparison matrix admitted under envelope E` | `AssessComparison` admitted the exact candidate matrix and digest-sorted measurement set | plan, candidate roster, measurement digests, equality basis, stimulus-independent comparison-basis digest |
| `comparison rejected before batching` | `AssessComparison` refused the exact matrix | measurement digests and typed reasons; no token, trial, batch, or outcome |
| `HOST_ALLOWED` | candidate retained ordinary host network access | trusted-local/full-user-permission warning |

Admission under an envelope never claims behavioral equivalence or physical freshness. Each concrete repetition has its own admission artifact; deterministic row tokens are internal and ephemeral. Placeholder projection establishes only a transformation of captured values, not that the placeholder had no earlier effect on control flow.

### Capture, projection, and observation

| Use this term | Exact meaning | Never infer |
| --- | --- | --- |
| `CapturedObservation after capture policy C` | bytes/channels retained after C, with tagged absent/truncated states | bytes discarded before C |
| `raw captured bytes` | bytes persisted unchanged by capture policy | redacted, normalized, decoded, or truncated bytes |
| `ProjectionResult` | result of named pure visible operations | the projected fields are complete or correct |
| `exact projection equality` | canonical projection bytes have the same fingerprint | semantic equivalence |
| `OBSERVED_STABLE(k/k,h)` | all `k` required new eligible trials yielded `h` | determinism or future repetition |
| `UNSTABLE(histogram)` | at least two eligible fingerprints were observed | which behavior is intended |
| `UNCOMPARABLE(reasons)` | after admission, source, control, or projection was ineligible | program contradiction; envelope rejection is a pre-batch `RejectedComparison`, not this classification |
| `INCOMPLETE` | budget ended before required eligible trials | equality, instability, or pass |
| `no observed difference under N named finite probes` | eligible projections agreed for exactly those probes | equivalence outside them |
| `eligible divergence` | at least two eligible candidates produced at least two fingerprints | correctness, recommendation, or majority truth |

An HTTP `500`, CLI exit `2`, or complete empty stdout may be eligible behavior. Detected materialization, setup, start, readiness, transport, timeout, cancellation, output, projection, orphan, and teardown failures remain control reasons.

### Exact map and reduction

| Use this term | Exact meaning | Required evidence |
| --- | --- | --- |
| `CandidateOutcomeMap` | complete roster, eligible labeled map, exclusions, plan/basis/phase, shared admission set, and globally unique evidence identities | full typed map plus distinct outcome-artifact and derived preservation-map digests |
| `PRESERVES` | full typed maps were comparable and their exact complete eligible labeled-map digests matched | opaque comparability result plus new attempt references |
| `CHANGES` | full typed maps were comparable and their exact complete eligible labeled-map digests differed | opaque comparability result plus new attempt references |
| `UNRESOLVED` | full maps were not comparable or no complete eligible map existed | typed basis/roster/eligibility/exclusion/control reasons and budgets |
| `UNCHANGED` | no accepted smaller typed stimulus | reducer set and transcript |
| `BEST_KNOWN under R and budgets B` | a smaller preserving stimulus exists but the final proof is incomplete | unresolved count, budget/sweep facts, reducer digest |
| `ONE_MINIMAL_UNDER(R)` | every enumerated constructible direct neighbor in the complete durable sweep has a typed `CHANGES` result with the required same-domain evidence/content-digest nonoverlap | exact enumerated neighbor set and sweep completion artifact; the grade itself does not establish physical execution or freshness |
| `locally reduced witness` | the recorded accepted path preserved the exact map | original/reduced stimuli, measure, transcript |

A naked `PreservationMapDigest`, display grouping, support count, or per-run admission artifact is never enough to claim preservation. `ComparisonBasisDigest` is stable across stimuli/repetitions only because it covers plan, envelope, roster, and equality basis rather than concrete measurement rows. `ONE_MINIMAL_UNDER(R)` is a local construction-grade label. It does not establish global minimum, root cause, semantic necessity, or human comprehension. Any `UNRESOLVED` direct neighbor, incomplete sweep, cancellation, budget exhaustion, changed ancestor, or reused executed observation makes that grade impossible to construct.

### Human decision and standalone residue

| Use this term | Exact meaning | Required qualifiers |
| --- | --- | --- |
| `Choicepoint` | immutable strict archive for one exact fresh-confirmed witness | current decision publication additionally requires the exact store-bound `CHOICEPOINT_READY` capability |
| `identity-hidden blind step` | the U6 DTO omits candidate/ref/producer/count/support/order/reveal fields | behavior can reveal provenance; distinct outcome count remains visible; no anonymity claim |
| `local-caller-attributed exact-witness ruling record` | a DecisionRecord applies only to one Choicepoint and explicit fields | `LOCAL_CALLER_ASSERTED_OPERATOR`; `AUTHENTICITY_NOT_ESTABLISHED_IN_U6`; presentation is not comprehension |
| `adapter-bound portable ruling` | roster-verified unchanged adapter projection bytes were strictly interpreted under their exact projection binding into complete portable tuples | exact translator/profile version; not runnable-source authority; legacy whole-projection records are nonemittable |
| `PortableSource reconstruction witness` | a closed adapter constructor retained runnable bytes and self-consistently reconstructed its supplied exact plan, projection binding/definition, minimized stimulus, portable profile, runner/start/readiness/capture authorities, and one execution binding; HTTP additionally requires the child-bind start authority | A1 construction authority only; equality to the current Choicepoint and every schedule-ordered FreshConfirmation execution binding is A2-only; contains no confirmation proof/ruling-tuple bytes, is not a ruling, and does not prove the raw bytes were historically stored in the DecisionRecord |
| `ALLOW_OBSERVED` | one or more complete confirmed observed tuples are positively allowed | exact alias expansion, tuple set, and separation proof |
| `CUSTOM_EXPECTATION` | one reviewed authored tuple containing exactly the selected fields and absent from every confirmed selected-field tuple is positively expected | no unselected field; matching an observed tuple is `CUSTOM_EXPECTATION_ALREADY_OBSERVED` |
| `REJECT_ALL` | dissatisfaction without a positive oracle | strict noncompilable DecisionRecord |
| `DEFER` | finalized noncompilable defer DecisionRecord | U6 may store it at `RULING`; no executable residue or deferred-session reopening |
| `REFINE` | semantic request for a successor study | U6 promotion returns `REFINE_REQUIRES_SUCCESSOR_STUDY`; no publication or head advance |
| `AMBIGUOUS_SCOPE` | selected fields fail to separate allowed and disallowed confirmed tuples | no compilable DecisionRecord or emitted files |
| `standalone selected-field test` | a recoverable deterministic six-file Node bundle compiled from a current portable ruling plus an exact source witness and run under exact target/runtime/parity evidence with all five standalone domains closed | exact target/runtime, predicate/source profile, selected fields, and scope status; no host-wide absence, listener ownership, confidentiality, or containment claim |
| `ContractExecutionTarget` | immutable nonhead pre-spawn authority constructed from the reopened bundle/residue, a live Git-issued explicit pinned and verified materialization, a fresh durable conformance attempt, opaque `DARWIN_KERN_BOOTSESSIONUUID_V1` authority, and an admitted/revalidated Node runtime | `Valid` is structural; C4 requires immediate physical reopen and performs the final full reopen on a detached bounded closure context after owner acquisition in the exact guarded window immediately before permit consumption/`Start`; the fixed probe is cooperating-executable profile evidence, not Node vendor authenticity; parsed bytes, copied fields, and caller-authored boot values are inert; no process-result or study-head claim |
| `ExecutionInterlock` | store-private operational CAS serializes subject admission for one OS-reported boot-session UUID and may block work after ambiguity | UUID equality/change is only a cooperative epoch fence; no semantic/result/discovery authority, liveness fact, physical-reboot proof, survivor-absence proof, process resume, or same-boot ambiguous reset; reset requires explicit operator action plus a differently measured UUID and still claims no prior-child absence |
| `StartClaim` / `RunPermit` | durable target-keyed intent permanently consumes one target; only the C4 runner that wins both authentic-target interlock and claim acquisition receives one process-local, nonreopenable, single-consumption permit | serialized cooperative at-most-once spawn admission only; no proof a child started, stopped, or is absent; fresh target alone cannot bypass an ambiguous held interlock |
| `FinalizedContractRun` | immutable nonhead physical-run authority bound to one exact target/attempt/start intent and one bounded closed-run witness with separate process-control and `COMPLETE | PARTIAL | VIOLATED` scope axes | no conformance classification, another-target pairing, host-wide absence, listener ownership, confidentiality, or study-head claim |
| `ContractExecution` | immutable nonhead classifier-profile-bound conclusion over one exact official target and that target's exact official finalized run | derived class only; no independently supplied tuple/process/scope reason, duplicate physical authority, process rerun, historical freshening, or head advance |
| `CLI invocation evidence` | canonical child-written receipt, at most 64 KiB, validated against one fresh runtime attempt ID and the exact logical argv, retained by digest-backed summary and retired before finalization | cooperating-candidate evidence only; not hostile-process attestation, provenance, authenticity, or proof against same-UID replacement |
| `exact private manifest` | target/attempt/StartClaim-bound sorted logical-reference sequence mapped to distinct checked private-byte ranges and digests under the 16-blob/64-MiB ceilings | multiplicity is preserved; duplicate logical references fail exact-roster comparison; private placement and default omission do not establish confidentiality or secure deletion |
| `C4 private-evidence envelope` | before owner acquisition, reserves at most 48 MiB of unique bodies from two 16 MiB channels, shared capture framing, projection framing, and twelve summary ceilings; every actual draft is capped at 15 logical kinds and 14 unique blobs and is re-counted before persistence | the full projected child reaches those caps, while controlled/start-error drafts may be smaller; no expansion of the store's 64 MiB/16-blob ceiling and no confidentiality or availability claim; Drain and Captured are different typed references to one persisted raw frame |
| `C4 bounded transient closure` | retained directory identities plus expected-cardinality-plus-one descriptor rosters; exact closure removes only named retained leaves and proven-empty roots, while foreign evidence refuses and foreign scope residue becomes ambiguous | no recursive cleanup, hostile-code containment, secure deletion, same-UID TOCTOU resistance, or guarantee that unknown residue is removed |
| `durable release revalidation` | classification reopens the exact StartClaim and validates the finalized-run release relationship immediately before persistence | a cached in-memory release bit is insufficient; missing/corrupt release blocks classification, while coordinated same-UID mutation between checks remains outside the guarantee |
| `C1 inert parsed model` | strict canonical construction, parsing, hashing, equality, and derived classification over already-held target/run/bundle semantic values | no store, Git, runtime, process, publication, spawn, evidence-content, chronology, or study-head authority; `UNRECEIPTED` until the exact C1 gate seals |
| `classifier profile link key` | canonical digest derived from the closed schema/kind/literal-profile declaration for the exact run/profile relationship | not serialized in `ContractExecution`, not caller-supplied classification authority, and no implementation-byte commitment |
| `C1 schema projection` | closed JSON syntax projection whose checked example intersects the strict Go model | schema acceptance alone is never semantic acceptance or authority; cross-field and recursive constraints remain Go-owned |
| `CONFORMS` | eligible clean observation under complete unviolated scope matches one allowed selected-field tuple | this exact target/run/classifier profile only |
| `CONTRADICTS` | eligible clean observation under complete unviolated scope matches no allowed selected-field tuple | this exact target/run/classifier profile only |
| `INELIGIBLE_EXECUTION` | process control and/or partial/violated standalone scope prevented an eligible behavioral conclusion | exact reasons remain in `FinalizedContractRun`; not a contradiction |

The C4 physical fixture exercises both eligible result branches without relabeling history: the enrolled reference target remains `CONTRADICTS`; a second real ruling compiled over the same exact two-file reference tree yields `CONFORMS` with the profile-ordered values `cli.stdout.json.mode=argv` and `cli.stdout.json.source=argv`. These are CLI-profile observations only. HTTP/two-profile completion remains absent.

Visited UI panels prove presentation only. They do not prove reading, comprehension, reduced bias, or review compression.

## Permitted and prohibited claims

<!-- countershape-validator: allow-prohibited-terms begin -->
| Permitted bounded phrase | Prohibited upgrade | Why the upgrade is false |
| --- | --- | --- |
| “repository-scale operational composition” after the decisive proof | “new disambiguation algorithm” or “new Choicepoint primitive” | Socrates, TiCoder, ARHF, FlashProg, differential testing, shrinking, approval tests, and contracts establish the semantic lineage |
| “comparison matrix admitted under envelope E” | “compatible worlds” or “reproducible environment” | unmeasured or tolerated dimensions may affect earlier control flow |
| `OBSERVED_STABLE(3/3,h)` | unqualified `STABLE`, “deterministic,” or “reliable” | finite repetitions do not prove future behavior |
| “exact projection equality” | “semantic equivalence” or “same behavior” | equality covers named projected bytes only |
| “no observed difference under N named finite probes” | “equivalent implementations” | unprobed behavior remains unknown |
| “locally reduced witness” | “smallest behavior,” “root cause,” or “global minimum” | reduction is local to named typed neighbors and budgets |
| `BEST_KNOWN under R and B` | “minimal” | the final sweep was unresolved or incomplete |
| `ONE_MINIMAL_UNDER(R)` | “the minimum case” | only constructible direct neighbors under R were ruled out |
| “local-caller-attributed exact-witness ruling record” | “complete intent,” “authenticated human action,” or “specification discovered” | the record covers only selected fields for one witnessed stimulus and does not establish actor authenticity |
| “selected-field contract” | “correctness proof,” “safety proof,” or “security approval” | nonasserted and unobserved behavior remains unconstrained |
| “candidate outcomes” | “winner,” “best branch,” “recommended implementation,” or “majority answer” | candidates are sensors, not voters or judges |
| “trusted local execution with `HOST_ALLOWED`” | “sandboxed,” “isolated,” “network blocked,” or “safe for untrusted PRs” | code has the user's permissions and host network |
| “regular/executable blobs accepted; links refused” | “symlink support” or “complete checkout” | v1 accepts only Git modes `100644` and `100755` |
| “new attempt roots and process lifecycles” | “execution-evidence cache/reuse with equivalent freshness” | v1 has no executed-observation reuse path |
| “default-minimized local evidence export” | “safe report” or “safely shareable” | arbitrary secret absence is not established |
| “fixture-known secrets omitted” | “secret-free export” | tests cover only named fixtures |
| “bundle hash matched” | “authentic” or “human-approved source” | hashing establishes bytes, not authorship |
| “standalone on Node X / OS Y under receipt Z” | “universally portable” | only the exercised runtime/profile is known |
| “localhost negative suite passed” | “web-secure” or “production-secure” | same-user/browser-extension attackers and review completeness are excluded |
| “didrun grade `<verbatim>`” | “verified,” “green,” or any upward summary not present in the grade | Countershape has no authority to reinterpret didrun |
| `UNRECEIPTED` | “implicitly covered” | nearby evidence cannot be assigned to an unrun item |
<!-- countershape-validator: allow-prohibited-terms end -->

Negating a prohibited phrase is not preferred public copy. Use the positive bounded phrase first, then the precise limitation. The table above exists so validators can permit intentional examples without globally allowing those words.

## Environment-specific receipt rules

Environment is part of the claim, never a footnote.

| Claim dimension | Minimum qualifying evidence | What remains `UNRECEIPTED` |
| --- | --- | --- |
| Darwin process lifecycle | native Darwin load-bearing suite through didrun on named architecture/kernel/tool versions | Linux, Windows, other architectures, different lifecycle code |
| Linux compilation | cross-build command receipt | Linux runtime behavior, process groups, permissions, filesystem semantics |
| Node standalone runtime | actual five-domain scope/parity/classification run on the exact Node path/version/major, native OS, and architecture | other runtime tuples; host-wide absence; network/registry denial; listener ownership; confidentiality; containment |
| Git materialization | exact Git version/object-format fixture suite with network/replacement/fetch negatives | other Git versions/object formats or arbitrary repository success |
| Reference-study timing | complete fresh study command on named machine and disclosed budgets | imported repository timing or other machines |
| CLI surface | exact CLI workflow tests on current tree | studio behavior |
| Studio state | exact state/viewport exercise plus the required visual inspection record | other states, viewport widths, browsers, forced-color/zoom modes not exercised |
| Accessibility automation | exact axe/keyboard/semantic command receipt | human usability, all assistive technologies, WCAG certification |
| Localhost request control | named auth/Host/Origin/content-type/CSRF/CAS negative suite | excluded same-user or browser-extension attackers, complete security review |
| Candidate-output inertness | named terminal/browser/report injection corpus | all encodings and downstream copy/paste targets |
| Export omission | named secret fixtures and structural checks | arbitrary confidentiality |
| Deterministic bundle bytes | clean-root byte comparison on the exact tree/toolchain | semantic parity, authorship, or future reproducibility |
| Different-model visual critique | exact invoked critic/model/input record if run through didrun | deterministic correctness, user comprehension, market judgment |

Cross-compilation is compilation evidence only. A browser screenshot is evidence that pixels were produced, not that a control works. A test on one viewport/state does not receipt another. An umbrella “cross-platform,” “accessible,” “secure,” or “fast” grade is prohibited unless each stated dimension has its own qualifying evidence and the phrase remains no stronger than those receipts.

## Claim record requirements

Every handoff/release capability row must carry:

- stable claim identifier;
- primary category;
- exact bounded statement;
- Git commit/tree identity;
- component and semantic artifact digests when applicable;
- fixture/probe/reducer/budget scope;
- OS, architecture, runtime, tool, viewport/state dimensions as applicable;
- exact didrun reference(s) and verbatim grade(s), or `UNRECEIPTED`;
- known limitations and excluded attackers/environments;
- for novelty assertions, closest credited work and why the composition differs; and
- for bets, the future human/market gate.

A receipt for a command that merely lists files cannot support runtime behavior. A unit test cannot substitute for the build-loop visual record. A model critic cannot substitute for deterministic tests. didrun receipts do not substitute for the underlying command's relevance.

## U0–U9 claim ceilings

| Unit | Permitted exit language | Prohibited promotion |
| --- | --- | --- |
| U0 | “planning artifacts passed the named planning validator” with exact receipt | implemented/runtime behavior |
| U1 | “truth-kernel vectors/properties/state mutants passed on the named tree,” only after the root-serialized didrun gate | subprocess, Git import, physical freshness, comparison runtime, reduction-grade, or standalone claims; until then all U1 capability is `UNRECEIPTED` |
| U2 | “named source/process fixtures passed natively on Darwin” | hostile containment or other-OS runtime |
| U3 | “named CLI fixtures received exact batch classifications and map” | HTTP/two-domain or decision claims |
| U4 | “named HTTP and CLI fixtures reuse the bounded truth kernel” | arbitrary adapter/repository support |
| U5 | exact `BEST_KNOWN` or `ONE_MINIMAL_UNDER(...)` with reducer/budget evidence | causal/global-minimum language |
| U6a | strict Choicepoint/Decision codecs, physical confirmation, blind identity/support omission, and fixed Darwin CAS spine only when mapped to exact receipts | authenticated-human authorship, complete studio payload, successor/stale lifecycle, REFINE successor creation, deferred reopening, Node residue, other filesystems, power-loss recovery, or production security |
| U6b / P07A | adapter-bound portable selected-field ruling only after unchanged-wire/profile, legacy-compatibility, physical study, and mutation receipts | runnable source, standalone bundle, residue, current conformance, or broad adapter portability |
| U6c / P07B | recoverable deterministic residue plus only the separately receipted C target/run/classification layers: typed publication, private boot-session interlock, serialized cooperative admission, exact runtime, closed process evidence, and five-domain scope | current correctness, exactly-once execution, same-boot child absence after ambiguity, host-wide absence, listener ownership, broad portability, containment, confidentiality, or authenticated-human authorship |
| U7 | “narrowed Darwin reference studies passed the named acceptance gates” | arbitrary-repository success or human benefit |
| U8 | exact authenticated/blind/viewport/accessibility observations | comprehension, bias reduction, production web security |
| U9 | per-capability final receipt map and minimized local export | umbrella completion, market validation, maintainability |

If a gate fails, use the honest milestone name: truth-kernel prototype, observation-only instrument, one-domain instrument, DecisionRecord-only residue, or CLI-only surface. A fallback never inherits the full reference-instrument claim.

## Public examples

### Acceptable

> Under comparison envelope `e`, candidates `k1`, `k2`, and `k3` were each observed at the recorded projection fingerprint in 3 of 3 new eligible trials. Candidate `k4` produced two eligible fingerprints and is `UNSTABLE`; it was excluded from the Choicepoint.

> The reduced CLI stimulus is `BEST_KNOWN` under reducer set `r` and the recorded 40-proposal/10-minute budgets. One direct neighbor remained `UNRESOLVED`.

> This decision allows the complete selected-field tuple `{status: 404}` for this exact request. The displayed body is context only and may change without making the generated contract fail.

> The bundle produced identical bytes in the named clean-root runs. Standalone Node semantics remain `UNRECEIPTED` until the parity and physical-absence commands run on the stated Node major.

> The default export omitted the named captured bodies and showed `CONFIDENTIALITY NOT ESTABLISHED`. This does not establish that arbitrary secrets are absent.

### Rejected and corrected

<!-- countershape-validator: allow-prohibited-terms begin -->
- Reject: “Countershape found the correct branch.” Correct: “The human allowed one selected-field outcome for the exact witnessed stimulus; no branch was endorsed.”
- Reject: “These worlds are compatible.” Correct: “These instances were admitted by comparison envelope `e`; tolerated and uncontrolled dimensions remain listed.”
- Reject: “The case is minimal.” Correct: use the exact reduction grade and reducer-set digest.
- Reject: “The contract proves the app is secure.” Correct: “The current eligible observation matched the exact selected-field tuple; nonasserted behavior remains unconstrained.”
- Reject: “The report is safe to share.” Correct: “The export is default-minimized, previewed, and labeled `CONFIDENTIALITY NOT ESTABLISHED`.”
- Reject: “didrun says this feature works.” Correct: quote the exact grade beside the exact command and state what that command exercised.
<!-- countershape-validator: allow-prohibited-terms end -->

## Wake and didrun wording

Permitted Wake wording:

> Candidate provenance identifies Wake session metadata as inert producer context. Countershape selected the immutable Git tree independently.

Wake provenance does not authorize Countershape to describe agent scheduling, recovery, effects, gates, sessions, timelines, replay, or fork as its state.

Permitted didrun wording:

> Command `<exact command>` at `<commit/tree>` has didrun grade `<verbatim grade>`.

Countershape never translates that grade into its own observation or contract classification. If the referenced command did not exercise the capability/environment, the capability remains `UNRECEIPTED`.

## Human and market claims

The following remain `HUMAN_OR_MARKET_BET` even after every technical unit passes:

- builders understand the projection and nonasserted-field boundary;
- blind-first staging changes or improves decisions;
- the bench reduces accurate decision time by at least 30% versus diff-first review;
- strict labeled-map reduction produces comprehensible witnesses in real repositories;
- generated contracts remain readable and maintained through refactors;
- imported repositories can expose useful fixtures economically;
- builders retain enough simultaneous candidates to adopt the workflow;
- contributors maintain adapters and security boundaries; and
- a hosted or paid layer is warranted.

Only the counterbalanced human study and actual adoption/maintainership evidence can move these claims. didrun, screenshots, telemetry visits, or a different-model critic cannot.
