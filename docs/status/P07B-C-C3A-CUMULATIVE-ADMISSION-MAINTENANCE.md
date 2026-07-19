# P07B-C C3A — cumulative-admission maintenance

## State

- **Boundary:** active pre-seal `SOURCE_FULL` maintenance after sealed C3PB and before the frozen C3 source unit. Classification: `DEFECT_REPAIR`.
- **Parent:** sealed C3PB commit `13369122ba7d5557eba1949095c1135a41843070`, tree `d710d9b249786f3fb648f566ae542f1d9ddec180`, is note-present with note blob `29c2d16db73d82c970b2eba6a838ec400a6f0139`, note-body SHA-256 `fa8cbe6a18e83f72343948dba511a08033d2bbba749696404ec450304a6094b1`, `7/7 claims recorded-exact`, and strict exit `0`.
- **Scope:** thirteen exact mode-`100644` paths, empty prefixes, sorted-newline digest `sha256:cc9bb98ceeb2f281336d3e235377d3cdb8920664f833baea830a84b847621188`.
- **Commit subject:** `fix: align pre-C3 architecture and runtime bounds`
- **Preserved C3 work:** stash object `3e766715609680adaef4678fc9aeaa9e3a8bae09` remains retained without dropping and supplies no C3A grade.

Every intended C3A grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3A itself.

## Trigger and defect classification

The preserved C3 build exposed two predecessor-contract mismatches before its frozen forty-path unit could be sealed:

1. P07B-B's cumulative architecture gate classified every scanned non-test Go mention under `internal/` of `ContractExecutionTarget` outside the C1 model and three C2 storage symbols as foreign. C3's sole intended issuer, `internal/contractexec/target.go:ContractExecutionTarget`, therefore could not pass the inherited gate even though the C0 authority contract assigns production issuance to C3. This is a checker-admission defect, not permission for a directory prefix or a new issuer class.
2. The JSON Schema already limits the ASCII Node runtime version to 128 characters, and C3's live probe enforces 128 bytes, but `internal/contractexec/model` accepted an unbounded regex match. Because the grammar is ASCII-only, the schema character bound and Go byte bound must coincide. This is a canonical-model implementation defect; examples, schemas, digests, and schema-conforming wire bytes do not change.

C3A repairs both mismatches at a distinct source boundary. It does not widen C3's frozen roster, rewrite any C3 stash byte, or turn predecessor compatibility into target authority.

## Exact repairs

### Inherited architecture admission

The B checker admits one new exact path-symbol pair: `internal/contractexec/target.go:ContractExecutionTarget`. Absence remains valid before C3 is restored. The admission is not a prefix and does not admit a renamed file, sibling package, runner package, prefixed identifier, suffixed identifier, or the tested Unicode-suffixed identifier. The pre-existing exact support pair `internal/store/nonhead_contract.go:ContractExecutionClassifierProfile` is now inventoried explicitly alongside the three exact C2 nonhead object-kind pairs; the same spelling at another path and any longer spelling remain foreign.

The collector conservatively scans word-shaped raw-source tokens in non-test Go files under `internal/`, including tokens in comments and string literals, for all three reserved family roots. Its controls distinguish each exact family from prefix, suffix, and tested Unicode-suffix variants such as `ContractExecutionTargetβ`, while the exact partition still refuses reserved spellings at hostile paths outside the intentionally admitted C1 model prefix and exact C2/C3 pairs. This is intentionally an internal production-Go token boundary, not a repository-wide scan, Go-AST identifier claim, runtime-string detector, or general Unicode-confusable detector. The checker executable guard compares real paths, and the self-test requires exact stdout plus empty stderr from both direct and symlink invocation, so an alias cannot silently bypass or noisily counterfeit the gate.

This repair addresses the predecessor B gate only. Once the preserved target files return, C3 still must evolve its own already-frozen architecture checker so the inherited historical-C1 topology admits exactly the intended target sibling pair while C3 mode validates its full issuer contract. Arbitrary or partial `internal/contractexec` topology remains forbidden.

### Canonical runtime bound

Target construction now enforces the exact 128-byte runtime-version ceiling before applying the existing version grammar. The existing primitive-bound test proves that a 128-byte ASCII version constructs and parses, while a 129-byte version fails both direct construction and a coherently rehashed parser path with `INVALID_CONTRACT_EXECUTION_TARGET`.

No schema, example, generated JSON validator, or canonical fixture changes. The repair intentionally rejects values that the Go implementation previously accepted even though they violated the sealed schema; every previously schema-conforming target remains unchanged.

### Receipt-phase succession

C3PB is an independently closed predecessor, not the active working unit. The plan checker therefore requires an exact `C3A_ACTIVE` operational tuple across both its synthetic receipt-absent and receipt-present reconstructions. Each of the four metadata keys must have exactly one exact canonical line inside `## Current state`; the maintenance heading and receipt-presence field must occur inside the matching active subsection. Moving correct bytes into quoted or historical text, adding a contradictory duplicate metadata key, or embedding the right tuple as historical text on an otherwise contradictory line does not satisfy the phase. Complete C3M-active and C3PB-active tuples remain recognizable only as stale historical diagnostics; neither is admitted as a current phase. A C3A reconstruction preserves the C3P receipt payload byte-for-byte, pins C3PB's closed identity, and rejects missing, duplicated, relocated, embedded, mixed, or coherent whole-tuple downgrade transitions. Canonical C3A receipt markers and grade forms are forbidden across all six C3A-owned Markdown files. The C3P source status is phase-neutral: its grades bind C3P only, while the closed C3PB identity and this active C3A boundary remain outside that source grade map.

## Exact C3A scope

C3A owns exactly these 13 paths: `docs/ARCHITECTURE.md`, `docs/HANDOFF_MODE_C.md`, `docs/PROMPT_PACK.md`, `docs/VERIFICATION.md`, `docs/status/P07B-C-C3A-CUMULATIVE-ADMISSION-MAINTENANCE.md`, `docs/status/P07B-C-C3P-RUNTIME-EPOCH.md`, `internal/contractexec/model/codec_test.go`, `internal/contractexec/model/target.go`, `spec/verification/p07b-c-unit-paths.json`, `tools/check-p07b-b-architecture-selftest.mjs`, `tools/check-p07b-b-architecture.mjs`, `tools/check-p07b-c-plan.mjs`, and `tools/check-p07b-c-unit-scope.mjs`. Their sorted-newline roster digest is `sha256:cc9bb98ceeb2f281336d3e235377d3cdb8920664f833baea830a84b847621188`.

Schema v7 orders `C3P -> C3V -> C3M -> C3PB -> C3A -> C3`. Prefixes are empty. C3 remains frozen to forty exact paths and digest `sha256:c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb`; C3A cannot absorb or relabel any C3 implementation path.

## Intended C3A claim map

| Exact claim label | Claim type | Pre-seal grade |
| --- | --- | --- |
| `P07B-C C3A cumulative-admission plan coherence` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3A cumulative-admission plan defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3A runtime-version exact 128-byte model boundary` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3A cumulative B architecture compatibility` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3A B admission and entrypoint defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3A unit-scope defensive self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3A cumulative verifier self-test` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3A cumulative verification` | `tests-pass` | `UNRECEIPTED` |
| `P07B-C C3A exact thirteen-path staged scope and diff integrity` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3A scoped staged credential-pattern scan` | `command-succeeded` | `UNRECEIPTED` |
| `P07B-C C3A preceding didrun chain integrity` | `command-succeeded` | `UNRECEIPTED` |

Only these eleven labels may be claimed. The declared manifest requires one exact `/usr/bin/env -i` prefix for all eleven commands, with pinned local tool paths, disabled Node startup injection, offline Go settings, and a dedicated final root. The final chain gate pins the first ten events' absolute argv, repository cwd, stable didrun wrapper-environment fingerprint, label/type, index, empty pathspec, complete wrapper capture, green exit, and one unchanged staged tree; it requires the C3P receipt declaration to remain present and no premature seal. Because its own wrapper event is appended only after it exits, event eleven can prove only its live cwd, Node realpath, invocation tail, effective child bindings, and exact mode-`0700` nonsymlink run directories—not its outer `env` argv self-referentially. didrun records that eleventh argv afterward, but the preceding-chain claim does not upgrade it to a self-proved fact. The fingerprint is same-wrapper provenance only, not full child-environment attestation.

## Throughput proposal remains closed

C3A does not reopen C3V's already-routed maintenance proposal. The Darwin/cgo build cache remains fresh per verifier run; qualified direct general work remains `-p=2`; sensitive and nested work remains serialized; the fail-fast verifier lock remains review-first; and exact receipt units retain their narrow seven-claim profile. C3A changes no verifier implementation or execution setting and claims no new throughput result.

## Nonclaims and gate back to C3

C3A proves neither a live target nor a production issuer. It adds no Git materialization, host measurement, Node probe, attempt allocation, official-target capability, interlock acquisition, process spawn, classification, CLI, dashboard, external API, security review, production readiness, adoption, or maintainership. The B checker is a repository architecture assertion, not a hostile same-user security boundary. The model-bound test proves parity at the checked tree only.

After a multi-pass build loop and independent review, stage exactly the thirteen C3A paths. Archive the zero-claim development ledger, rotate the development-populated C3A run root, recreate the root plus `home`, `tmp`, `gotmp`, `gocache`, `gopath`, and `gomodcache` as owned mode-`0700` nonsymlink directories, then start a fresh eleven-claim ledger. Run and immediately claim only the rows above, commit the exact tree, seal it, inspect the note, loop strict verification to exit `0`, generate the exact-commit HTML report, and archive the final ledger. Only then revalidate stash object `3e766715609680adaef4678fc9aeaa9e3a8bae09` and apply it without dropping. Any overlap must preserve the sealed C3A repairs, and the post-apply inventory must remain inside C3's frozen forty-path roster. C3 must enroll the sealed C3A identity in its predecessor manifest and solve the separate inherited-C1 topology transition inside its existing frozen checker paths.
