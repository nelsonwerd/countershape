# P07B-A1 source and portable-start substrate receipt

- **State:** source implementation sealed and strict-clean at commit `1e56da3bfafc4cdb8abdca62f18fb3b1accea8c3`, tree `da0e13f37aacd0e7fb2b52b3aeac628f5558c11e`
- **Prerequisite:** P07A-B source commit `65005e498b05f3da62d6e30e108fb1cdba68d2af`, tree `18f52daa4f7b25b71d7fe3fda70c2aba561d29c9`, is sealed and strict-clean
- **Strict didrun verdict:** `ALL RECORDED-EXACT`; `15/15 claims recorded-exact`; every grade below is copied verbatim as `TREE-EXACT`
- **Seal disclosure:** the plain seal stopped on `105 likely secret(s) found (high-entropy)`. After the exact 34-path staged inventory, clean staged diff, scoped structured credential-pattern scan, and manual review, the commit sealed with didrun's logged `--allow-secrets` redacted-export override. This is not a secret-free finding or a security review.
- **Receipt-document boundary:** the follow-up documentation commit carrying this file and the controlling handoff is accepted only when its own fresh serialized ledger is chain-intact, sealed, and strict-clean

## Implemented boundary

P07B-A1 implements the source and portable-start substrate:

- closed model-level CLI and HTTP projection reconstruction without changing the historical projection bytes;
- exact portable-profile parsing and typed regeneration;
- a bounded opaque `PortableSource` that reconstructs and byte-matches its supplied `WorldPlan`, projection binding and definition, portable profile, adapter stimulus, capture/start/readiness authorities, runner profile, and CLI/HTTP execution binding;
- a distinct HTTP start authority, `NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1`;
- an exact FD3 readiness grammar, `COUNTERSHAPE_READY_V1 <port>\n` followed by EOF;
- a reviewed Node-core fixture that binds literal `127.0.0.1:0`, reports the selected port through FD3, and receives the request only after the frame is accepted;
- finalized readiness-only rejection receipts for malformed, incomplete, timed-out, or prematurely exited child processes; and
- a physical four-trial Darwin `Observation` under the child-bind lineage while the sealed inherited-listener fixture remains byte-identical.

This is a nonhead, nontransition substrate. It does not authorize compilation or change the study head.

## Exact physical and compatibility facts

- Sealed inherited-listener fixture: 18,343 bytes, SHA-256 `d476c70a9c8e1605fc65335044afddaa9f56f007d1fcbc70ea87c0b747c72caa`.
- New reviewed child-bind fixture: 19,264 bytes, SHA-256 `8cb5dfc1d5276e9f6157c127339e6e6c6404f525a3e8f9bec4f3eb1ad84a9932`.
- Claim-bearing mutation event admitted Node `v25.2.1` at the recorded path fingerprint and Apple Git `2.50.1 (Apple Git-155)` at the recorded path fingerprint.
- Go verification used Go `1.26.5` on the current Darwin arm64 host with CGO enabled and the exact paths recorded in didrun.
- The mutation seed manifest was `sha256:b3cecc2bb22af9abb5f034b09fa9578e701d4502b5664e21f38c9e707ed1de51`; the frozen 30-ID roster digest is `sha256:79036a0a86685d70d62c3239ce1488d16ccfd9738277c199a7685f547f6f369a`.
- Private mutation copies explicitly retained the invoking user's host filesystem and network authority.

These are exact receipt facts for the named environment. Linux, Windows, other architectures, and other Node versions or majors are `UNRECEIPTED`.

## Verbatim receipt map

A receipt records what ran against the sealed tree. It does not prove correctness, completeness, hostile containment, or market value.

| Event | Claimed capability | didrun claim label | Verbatim grade |
| ---: | --- | --- | --- |
| 89 | exact source staging set and whitespace/diff integrity | `P07B A1 exact 34-path staged inventory and diff check` | `TREE-EXACT` |
| 90 | checked-in Node fixtures and P07B verification tools parse | `P07B A1 Node fixtures and verification tools parse` | `TREE-EXACT` |
| 91 | source/process dependency closure and exact architectural anchors | `P07B A1 exact source and process architecture boundary` | `TREE-EXACT` |
| 92 | 48 hostile architecture copies plus clean/comment controls | `P07B A1 architecture checker hostile-copy self-test` | `TREE-EXACT` |
| 93 | mutation harness roster, receipt binding, restoration, and tamper self-test | `P07B A1 mutation harness receipt and tamper self-test` | `TREE-EXACT` |
| 94 | thirty required mutants killed with thirty fresh A/B/A receipts | `P07B A1 thirty-mutant fresh A/B/A closure` | `TREE-EXACT` |
| 95 | complete Go repository test suite | `P07B A1 complete Go repository suite` | `TREE-EXACT` |
| 96 | complete Go vet pass | `P07B A1 complete Go vet` | `TREE-EXACT` |
| 97 | complete Go race suite | `P07B A1 complete Go race suite` | `TREE-EXACT` |
| 98 | bounded one-worker PortableSource authority-reconstruction fuzzing | `P07B A1 PortableSource authority reconstruction fuzz` | `TREE-EXACT` |
| 99 | bounded one-worker child-reported port-frame grammar fuzzing | `P07B A1 child-reported port-frame grammar fuzz` | `TREE-EXACT` |
| 100 | tenfold portable-readiness rejection and lifecycle stress | `P07B A1 tenfold portable readiness negative-path stress` | `TREE-EXACT` |
| 101 | threefold inherited-listener and child-bind physical studies | `P07B A1 threefold legacy and child-bind physical studies` | `TREE-EXACT` |
| 102 | focused closed source, projection-profile, adapter-model, and runner-profile suites | `P07B A1 closed PortableSource projection profile and runner model suites` | `TREE-EXACT` |
| 103 | scoped structured credential-pattern scan over the exact staged blobs | `P07B A1 scoped staged structured credential-pattern scan` | `TREE-EXACT` |

The claim-bearing mutation output contains exactly thirty `KILLED` rows and ends with `P07B mutation gate: 30/30 required mutants killed; 30/30 fresh A/B/A receipts.` Seeded mutation closure is a tested guard, not mutation completeness.

## Permanent negative history

Events `1`, `5`, `7`, `11`, `13`, `17`, `18`, `22`, `28`, `29`, `32`, `34`, `35`, `37`, `39`, `45`, `51`, `52`, `59`, `60`, `62`, `64`, `66`, and `71` exited nonzero during development. They preserve, respectively, evolving model/test-contract defects, architecture-checker and hostile-selftest gaps, physical-readiness assertion gaps, mutation-harness and equivalent/noncompiling-mutant defects, the transitive-dependency import-cycle defect, and the expected canonical-golden failure used to regenerate `child_reported_port`'s digest. None supports a capability.

The first plain seal's high-entropy refusal is also permanent operational evidence. It was not weakened or deleted. The source ledger is preserved locally under `.didrun-history/2026-07-16-p07b-a1-source/.didrun/`.

## Explicit nonclaims

P07B-A1 does **not** establish:

- equality to the current store-bound `Ruling`, `Choicepoint`, or `FreshConfirmation`;
- independent retranslation of reopened confirmation proofs or compile authorization;
- a FreshConfirmation, portable Choicepoint, or selected-field ruling under the new HTTP lineage—the physical A1 study reaches `Observation` only;
- a live materialization-policy capability join; reconstructed plan digests and budgets are data, not target/application verification;
- a compiler, six-file bundle, Go/Node parity, terminal `RESIDUE`, publication transition, materializer, target, finalized run, classification, product CLI, server, dashboard, report, or UI;
- kernel evidence that the reporting PID owns the listener. `child_reported_port` proves the exact child-process-tree frame and its use in the subsequent request. The reviewed fixture itself binds loopback before reporting, but an arbitrary trusted script could report a decoy local service;
- static analysis, secret scanning, dependency absence, network denial, confidentiality, or containment from `closed_facts`;
- exact execution of the opened Node/Git inode. The mutation gate honestly reports an admitted path fingerprint and retains the path hash/execute time-of-check/time-of-use boundary;
- hostile-code isolation. Trusted repository code still runs with the user's full permissions and host network access;
- cross-platform/runtime portability, production readiness, adoption, human comprehension, legal clearance, independent security review, release readiness, or maintainership; or
- any didrun conclusion stronger than the verbatim grades above.

The scoped credential-pattern scan covered a named pattern set over the exact staged blobs. It is not a complete credential audit and does not authorize publication of the secret-bearing didrun ledger.

## Pre-A2 planning-wire correction

Follow-up documentation red-team work found an unsound planning-only
determinism shape before any runtime `ContractBundle` codec or compiler existed.
The old whole-bundle `contains_*` flags could not truthfully be false because
exact `PortableSource`, predicate values, digest-linked authority data, and
source-derived human text may themselves contain sensitive, host-looking,
candidate-looking, receipt-looking, or order-bearing bytes.

The planning schema, example, generator, validator, and controlling prompt pack
now use the exact scope
`EMITTER_INVENTED_STRUCTURAL_FACTS_EXCLUDING_AUTHORIZED_INPUT_CONTENT` and seven
`emitter_introduces_*` flags. All seven flags constrain only typed structural
facts invented by the emitter; they are not a content scan, redaction claim,
secret-absence claim, or security result. `confidentiality_established` remains
false. The corrected planning wire also makes `CUSTOM_EXPECTATION` carry exactly
one allowed tuple, bounds `source_profile.subject_entrypoint` to 4,096 ASCII
bytes, and rejects a root-leading hyphen exactly as the sealed shared
`runnerprofile` grammar does. The validator now locks the exact outer,
predicate, file, dependency, determinism, and confidentiality constants rather
than only their property rosters.

This correction changes only future planning fixtures and their derived
digests. It does not change any sealed A1 canonical byte, digest, fixture,
receipt, or claim, and it does not create compilation authority or compiler
capability. The documentation/spec commit carrying this correction requires its
own independent didrun receipt.

## Sole next source boundary

P07B-A2.1 is next. It must reopen and revalidate the current store-bound portable ruling, retain the exact FreshConfirmation execution-binding roster, independently retranslate the reopened projection proofs under the source-matched profile, revalidate the allowed/disallowed selected-tuple partition, and produce only a sealed wrapper that privately retains original authority and owns a separate sanitized internal compiler input. It creates no generated files, store object, temporary output, or head transition.

Only after A2.1 is independently committed, sealed, and strict-clean may A2.2 build the pure deterministic recoverable six-file compiler and Go/Node corpus. Terminal publication/materialization and target/run/execution remain separate later boundaries. See `../prompts/P07B-A2-COMPILATION-PLAN.md`.
