# P07B-A2.2 preimplementation rulings

- **Status:** controlling correction to
  `P07B-A2-2-RECOVERABLE-COMPILER.md`
- **Reason:** independent spec, code, and test red teams found claims whose
  evidence was ambiguous or not constructible without production fault hooks
- **Scope:** A2.2 only; no publication, materialization, target, run, or study
  transition authority is added

These rulings narrow ambiguity without reducing the six-file compiler,
strict-recovery, Go/Node parity, generated CLI/HTTP, determinism,
black-box/property/boundary evidence, or human-surface goals. If this file and
the parent prompt disagree, this file controls A2.2. A later wire or authority
change requires a new reviewed version.

## R1 — parser identity and coherent replacement

The external typed `ContractBundle` digest is required parser input:

```go
func ParseContractBundle(
    exactCanonicalBody []byte,
    expectedDigest domain.Digest,
) (ContractBundle, error)
```

The parser first requires strict canonical body bytes, recomputes the exact
`ContractBundle` domain digest, and compares it with `expectedDigest` before it
returns an inert value. There is no exported parse-without-expected-identity
path. The compiler computes the digest from its newly assembled body and calls
this same parser before returning.

The required tests distinguish four cases:

1. any mutation under the original expected digest fails;
2. recomputed outer metadata with an inconsistent decision, fixture, manifest,
   README, fixed asset, source, profile, stimulus, descriptor, or predicate join
   fails;
3. a fully coherent rewritten bundle with a newly computed external digest may
   parse as a different inert bundle; and
4. neither success case recreates ruling, publication, authorship,
   authenticity, or currentness authority.

“Tamper detected” always means disagreement with a fixed expected identity or
one of the closed internal joins. It never means that a coherent alternate
bundle can be identified as a forgery.

## R2 — A2.2 is a partial compiler with a mechanical ceiling

`Compile` is intentionally partial over otherwise valid sealed A2.1 inputs.
Oversize input returns typed `COMPILER_LIMIT_EXCEEDED` and the zero
`ContractBundle`; it never truncates, emits a partial file roster, or relaxes an
upstream PortableSource/value limit.

Freeze these A2.2 limits unless the mechanical proof changes before the unit is
sealed:

```text
file count                         = 6
per-file raw bytes                 <= 640 KiB
aggregate raw bytes over six files <= 640 KiB
outer non-content canonical bytes  <= 160 KiB
final canonical body bytes         <= canon.MaxInputBytes (1 MiB)
```

`outer non-content canonical bytes` is the exact final canonical-body length
minus the sum of the six canonical padded-base64 content string lengths. It
therefore includes the complete outer `predicate`, `source_profile`, all file
metadata, fixed constants, braces, quotes, commas, and count spellings. It is
not an estimate.

For six file lengths `n_i`, the static proof is:

```text
sum(4 * ceil(n_i / 3))
  <= 4 * ceil(MaxAggregateRawFileBytes / 3) + 4 * (FileCount - 1)

encoded_content_maximum
  = 4 * ceil(MaxAggregateRawFileBytes / 3) + 4 * (FileCount - 1)

encoded_content_maximum + MaxOuterNonContentBytes
  < canon.MaxInputBytes
```

The final exact length check remains mandatory even after that proof. Separate
limits for generated decision, fixture, README, program assets, manifest,
selected fields, tuple count, and value payloads may be stricter, but none may
make the equation less conservative. Tests cover every independently reachable
limiter with an exact accepted boundary and boundary plus one, plus a
multidimensional case where independently valid source and tuples do not fit
together. A guard that is mechanically dominated by another invariant is not
given a dishonest black-box boundary fixture: the per-file 640 KiB guard is
dominated by six required non-empty files plus the aggregate 640 KiB guard, and
the final 1 MiB check is dominated by the static encoded-content proof. Those
defense-in-depth guards are instead exercised through checked arithmetic,
hostile-copy/parser fixtures, and architecture self-tests that make the guards
observable without pretending their boundaries are reachable from a valid
compiler input.

## R3 — exact parity corpus and subprocess protocol

`spec/vectors/v1/contract-parity.jsonl` is canonical JSON Lines:

- one strict canonical JSON object plus one LF per line;
- no BOM, CR, blank line, trailing whitespace, or missing final LF;
- at most 512 vectors, 65,536 whole bytes per line including its terminal LF,
  and 1 MiB total including every terminal LF;
- unique ASCII IDs matching `^[a-z0-9][a-z0-9._-]{0,95}$`;
- a closed root roster: `id`, `description`, `tags`, `operation`, `input`, and
  `expected`;
- tags are unique and unsigned-UTF-8 sorted; and
- only the driver may observe `id`, `description`, `tags`, or `expected`.

The closed operation roster is:

```text
CANONICALIZE_JSON
PARSE_CANONICAL_JSON
PARSE_MANIFEST_ENVELOPE
PARSE_READY_FRAME
PARSE_HTTP_RESPONSE
PROJECT_CLI_OBSERVATION
PROJECT_HTTP_OBSERVATION
EVALUATE_EXACT_PREDICATE
SELECT_DIRECT_RESULT
SELECT_OWNER_ELIGIBILITY
```

Each operation receives an exact closed `input` object and returns one exact
closed result:

```text
{"status":"OK","value":<operation-owned closed value>}
{"status":"REFUSED","code":<closed operation-owned code>}
```

The detailed per-operation field rosters live next to the Go evaluator and are
locked by schema/roster tests before vectors are admitted. Changing a roster or
operation name is a corpus-version change, not a permissive parser update.
`SELECT_DIRECT_RESULT` deliberately has no cancellation input because the
standalone ABI owns no cancellation trigger. `SELECT_OWNER_ELIGIBILITY` is the
separate pure semantic adapter over the existing owner-event control roster and
includes `CANCELLED`; it creates no direct-runtime injection surface and is not
called by `runContract`. This separation resolves the parent brief's otherwise
contradictory requirements to retain cancellation in semantic parity while
forbidding cancellation in the direct invocation contract.

The Go evaluator is an input-only thin adapter over the existing Go
canonicalization, adapter wire/readiness, projection, exact-value/predicate,
and result-selection owners. It may not read the corpus or duplicate an
existing parser merely to agree with Node. The Node evaluator is exported by
the fixed harness and receives the same input-only operation object. Neither
evaluator accepts an expected value, vector ID, description, tag, corpus path,
or filesystem lookup.

The Node runner protocol is one canonical request plus one LF on stdin and one
canonical response plus one LF on stdout, with empty stderr, no extra frame,
and a 65,536-byte whole-wire cap in each direction. The one required terminal
LF is part of that cap, so the canonical request or response body is at most
65,535 bytes. A framing failure exits `1` with empty stdout and stderr and no
partial response. In particular, a pure semantic result larger than the
response body cap is a protocol failure, not semantic
`REFUSED/OUTPUT_LIMIT`; semantic evaluation remains independent of the runner
envelope. One fresh process evaluates one request.
The exact harness export roster is:

```text
canonicalizeJSON
parseCanonicalJSON
evaluateParityOperation
runContract
```

No other export is allowed. Poisoned expectations, shuffled vector order,
duplicate exact inputs under different private labels, and static rejection of
driver-only metadata in evaluator inputs remain required oracle-separation
gates.

`PARSE_MANIFEST_ENVELOPE` runs independently against all three parsers: the Go
parser, the harness evaluator parser, and a copied/instrumented entrypoint asset
that exposes only its otherwise-private minimal parser at the test edge. Each
is compared with the vector's literal expected result; pairwise agreement alone
is insufficient. The copied entrypoint layer is labeled parser-corpus evidence,
not intact production-asset direct-execution evidence, and the production
entrypoint gains no extra export or test hook.

## R4 — exhaustive pure result evidence versus physical fault evidence

No production environment variable, file, argument, exported mutable hook, or
test-only branch may inject a harness fault.

Evidence is split into three named layers:

1. the production pure result selector is exhaustively tested for every closed
   reason, every adjacent ineligible-precedence pair, concurrent
   output-limit/timeout, primary-plus-teardown/orphan, success-plus-cleanup,
   and the required three-fault combinations;
2. intact generated assets are physically invoked for naturally constructible
   conformance, contradiction, malformed data, companion tamper, source
   inventory, root topology, start, readiness, transport, timeout, output,
   teardown, and cleanup paths available on the named reference runtime; and
3. copied, explicitly instrumented test assets/adapters exercise otherwise
   unconstructible filesystem/process failure paths. Those copies prove the
   mapping and test harness, not byte-identical production-asset behavior.

The architecture hostile-copy gate rejects any production fault hook. Receipt
labels and the handoff keep all three evidence layers distinct.

## R5 — total direct-result mapping

Before attempt allocation, bootstrap/data outcomes retain the parent prompt's
stage mapping: unexpected runtime/invariant failure is harness failure, a
recognized companion mismatch is tamper, and a recognized grammar/join failure
is malformed contract data. Once runtime work begins, families are selected in
this order after all cleanup and watcher facts are collected:

```text
1. INELIGIBLE_EXECUTION / ORPHAN_RISK
2. INELIGIBLE_EXECUTION / CLEANUP_FAILED
3. INELIGIBLE_EXECUTION / TEARDOWN_FAILED
4. TAMPER_DETECTED / COMPANION_INTEGRITY_MISMATCH
5. HARNESS_FAILURE / INTERNAL_INVARIANT_FAILED
6. first present remaining INELIGIBLE_EXECUTION reason in the parent order,
   beginning with OUTPUT_LIMIT
7. CONFORMS / NONE or CONTRADICTS / PREDICATE_MISMATCH
```

The three safety-finalization facts stay visible even when an internal error
also survives. A recognized post-bootstrap bundle integrity mismatch is next.
An unexpected or unmapped internal/runtime error then overrides every ordinary
control or behavioral fact; it may never masquerade as contradiction,
conformance, timeout, or another typed reason. Test internal error paired with
each safety fact, integrity fact, ordinary ineligible reason, and behavioral
result in both discovery orders and in three-fault combinations.

Freeze these previously ambiguous stage mappings:

- readiness budget expiry, early child exit before the accepted frame, FD3
  error, incomplete frame, or a holder that prevents required EOF is
  `READINESS_FAILED`;
- `TIMEOUT` begins only after readiness for HTTP, or after successful spawn for
  CLI, and represents the bounded subject/probe deadline;
- a missing declared entrypoint in the supplied inventory is
  `SOURCE_INVENTORY_INVALID`; disappearance or mismatch after a valid source
  manifest was copied is `SOURCE_COPY_FAILED`;
- inability to establish the required named-platform watcher contract, or a
  later watcher-facility error without an observed input mutation, is
  `ENVIRONMENT_INVALID`; an impossible watcher state or unmapped exception is
  harness failure;
- a target-inventory watcher event is `SOURCE_INVENTORY_INVALID`;
- a bundle-root watcher integrity event is `TAMPER_DETECTED`; and
- recognized attempt-root cleanup refusal is `CLEANUP_FAILED`, while an
  unexpected cleanup invariant/error is harness failure.

Transport/capture/projection boundaries are exact: connect, write, socket, or
natural HTTP child exit/signal failure is `TRANSPORT_FAILED`; complete retained
HTTP bytes rejected by the exact response grammar are
`RESPONSE_PARSE_FAILED`; capture/drain/evidence-assembly failure is
`CAPTURE_FAILED`; and valid retained CLI/HTTP observation bytes rejected by
selected-field extraction or exact-value construction are
`PROJECTION_FAILED`. Adjacent cross-fault tests prove the frozen precedence.

Pure readiness vectors cover accepted ports `1` and `65535`. Physical HTTP
smoke uses an OS-assigned literal-loopback port; it does not claim physical
ability to bind either grammar boundary port.

## R6 — source-inventory and watcher contract

The source inventory mirrors the existing Git materializer's path language:

- path identity and sorting use raw valid UTF-8 bytes with unsigned byte order;
- no Unicode normalization or locale collation occurs;
- a path is relative, slash-separated, at most 4,096 bytes, and each component
  is at most 255 bytes, with at most 128 components;
- no empty, `.`, `..`, absolute, backslash-containing, volume, control-text, or
  DEL component is admitted; control text includes U+0000..U+001F and
  U+007F..U+009F;
- every ASCII-case spelling of `.git` is refused;
- only directories and regular files are admitted, files have exact logical
  mode `100644` or `100755`, and regular files must have link count one;
- empty directories are refused because they have no Git-materialized source
  identity;
- the source-carried entry count counts regular files; directory count and
  depth are separately bounded at 8,192 directories and 128 components;
- directory names are read as bytes, admitted only when strict UTF-8
  decode/re-encode reproduces the same bytes, and never passed through a lossy
  string decode before validation;
- duplicate regular-file `(device,inode)` identities, file/directory prefix
  conflicts, and destination-volume case or Unicode alias collisions are
  refused;
- exact file bytes, counts, modes, and raw SHA-256 values form the sorted source
  manifest; and
- create-new copy plus reopen/hash/mode checks must reproduce that manifest.

On the named Darwin evidence path, logical modes map only from exact
`stat.mode & 0777 == 0644` or `0755`. Each source file crosses
`lstat -> O_NOFOLLOW open -> fstat`, with device, inode, type, link count, and
mode equality before the full read and a second `fstat` agreement after the
read. A swap during open/read refuses. Modes such as `0640`, `0664`, and `0744`
are not rounded or inferred from only an executable bit.

The current filesystem is allowed to determine whether two distinct byte names
can coexist; copy-time `O_EXCL` collision refusal catches target-volume aliases.
No cross-platform Unicode/case-equivalence claim is made.

Both input-root watchers are established and error-checked before the first
inventory read, allocation, copy, overlay, or spawn. They remain active through
process teardown and attempt-root cleanup, are drained for two event-loop
turns, and are closed only after their terminal facts are collected. Root
device/inode/canonical identity is checked before and after. A positive-control
test must prove the exact watcher observes a create/delete event before any
quiet-run assertion can carry evidence. Watcher silence is named Darwin
evidence, not proof that every filesystem exposes every transient event.

## R7 — exercised runtime and machine-diagnostic scope

A2.2 physically receipts only the admitted local Darwin/arm64 Node tuple.
Generated source remains runtime-generic declaration data, but direct execution
on a platform where the required mode/watcher/process-group contract is
unavailable returns `INELIGIBLE_EXECUTION/ENVIRONMENT_INVALID`. The bundle
contains no host tuple.

The entrypoint itself emits exactly one native `node:test` diagnostic beginning
`COUNTERSHAPE_RESULT_V1|` after registration for every caught
contract-controlled path. Under the explicitly selected TAP reporter, its
outer wire is exactly one `# COUNTERSHAPE_RESULT_V1|<outcome>|<reason>` line on
stdout. Tests invoke Node from a controlled sparse parent environment, require
empty outer stderr, and reject missing or duplicate Countershape prefixes.
They do not claim that stderr contains no earlier Node loader/runtime diagnostic
under an operator-controlled ambient environment; pre-registration loader,
signal, OOM, and diagnostic-emission failures remain no-diagnostic nonclaims.

## R8 — verifier evolution and script classification

A2.2 must update:

- `tools/verify-current.mjs` and its hostile self-test;
- `docs/VERIFICATION.md`;
- the layered A2 architecture checker and self-test; and
- the A2.2 status/handoff.

The current architecture row becomes P07B-A2.2 and keeps A1/A2.1 closure. The
proposed source-rewrite driver was never implemented, checked in, claimed, or
sealed, so it is `UNRECEIPTED` design history and must not appear in either the
current or historical executable roster. The exact current/historical roster
digest and orphan hostile tests are repinned. No historical script is called a
current pass.

## R9 — black-box, fuzz, and human-render closure

The recipe-level source-rewrite lane is superseded and `UNRECEIPTED`. A2.2
instead closes its semantic surface through black-box conformance and negative
matrices, oracle-separation properties, Go/Node differential parity, coherent
recovery/tamper cases, bounded fuzzing, exact ceiling boundaries, deterministic
subprocess checks, and the retained hostile architecture self-test. No one
category may be inferred from another, and no source-rewrite completeness claim
is made.

Parser negatives must remain non-vacuous. Identity cases retain the original
expected digest and prove mismatch. File, manifest, and semantic join cases use
a coherent alternate outer body and expected typed digest so the identity gate
passes before the named downstream join is refused. Each case asserts its
prerequisite joins before the intended refusal. A fully coherent alternate body
plus its new digest succeeds as a different inert bundle. Bundle fuzz seeds
include coherent bundles and exercise internal joins rather than stopping only
at the first digest comparison.

The bounded active fuzz gate is fixed to the bundle parser and strict corpus
line parser. Each runs with the checked-in seed corpus, one worker,
`-parallel=1`, at least `-fuzztime=10000x`, and no network. The final command may
raise but not lower that deterministic iteration floor. The status and receipt
label report the exact count; “fuzzed” is not generalized beyond those targets
and iterations.

Add a checked-in Node-core deterministic README renderer/capture tool. It
accepts only widths `60`, `80`, and `120`, emits no ANSI or active control
sequence, performs no network or package lookup, and wraps the fixed Markdown
subset by Unicode code points under one versioned capture grammar. The three
captures and structured CLI records—not an unstated terminal application—are
the human-touch review authority.

## R10 — implementation surface correction

The subprocess compiler probe that needs private `compilation.Input` lives
under `internal/emit/node`, where Go's internal visibility permits it.
`testkit/contracts` may provide public-service fixtures and Node runners but may
not bypass the internal-package rule.

Self-contained recovery also crosses a fresh-process boundary. The compiler
probe emits only the exact outer canonical body and external digest. A separate
parser subprocess—with no compiler input, corpus path, or repository lookup and
with varied cwd and sparse environment—must recover all six files and the exact
PortableSource. This is process-local reconstruction evidence, not crash
persistence or storage durability.

In addition to the parent prompt's surface, A2.2 may add the bounded renderer,
strict corpus schema/parser tests, A2.2 status file, and current-verifier updates
required by these rulings. It may not add a package manager, external service,
production fault hook, publication/materialization path, or target/execution
authority.

## R11 — standalone invocation facts are harness-owned

The generated direct harness does not require a subject-authored
`cli-invocation.json` or any other Go-world receipt. Existing Go execution
requires that evidence because its stronger capture object proves a separate
historical fixture invocation. A2.2 instead constructs the exact logical argv,
private candidate copy, and fixture overlay itself and retains those facts only
inside the current direct run. Requiring the subject to echo them would make an
arbitrary generated contract depend on an undeclared Countershape protocol and
would turn standalone UX into receipt authority. The harness must still verify
its own overlay bytes, argv, cwd, environment, and source-copy manifest before
spawn; this ruling removes no such check and creates no durable execution
evidence.

## R12 — the machine channel is a native TAP diagnostic

The originally proposed outer-stderr ABI is impossible under the simultaneously
required invocation `node --test --test-reporter=tap`. Node runs the test file
in an isolated child, consumes that child's stdout and stderr as test events,
and renders both through the selected TAP reporter on the reporter destination.
Writing to `process.stderr` or file descriptor `2` inside the registered test
therefore becomes a TAP comment on outer stdout; it cannot remain an independent
outer-stderr record. Selecting stderr as the reporter destination would move the
entire TAP stream and still would not create the promised separation. A parent
launcher or custom reporter could create such a channel, but would change the
locked six-file bundle and exact invocation.

The corrected public direct-result wire is one native test diagnostic, emitted
through the registered test context, rendered by the explicitly selected TAP
reporter as:

```text
# COUNTERSHAPE_RESULT_V1|<outcome>|<reason>\n
```

The pipe delimiter is intentional: Node's TAP escaping rewrites tabs and
backslashes, while the closed outcome and reason alphabet cannot contain a
pipe. Controlled sparse-environment runs require exactly one such prefix on
stdout and empty outer stderr. Subject stdout/stderr remain captured data and
are never forwarded. Only `CONFORMS|NONE` permits exit zero; every other exact
diagnostic accompanies the ordinary nonzero TAP result. This correction changes
only standalone operator UX and does not create Target, FinalizedRun,
ContractExecution, receipt, or publication authority.

## R13 — inert HTTP seed paths retain their sealed constructor domain

The sealed HTTP seed constructor admits strict UTF-8 clean relative slash paths
but, unlike the CLI fixture constructor, does not exclude U+0000 or other
control code points. A2.2 recovery must not narrow that already-valid inert
`PortableSource` domain and mislabel such a source as malformed contract data.
The bundle parser therefore mirrors the sealed HTTP constructor exactly. If the
local filesystem API cannot materialize one of those admitted names, the
attempt refuses before spawn as `FIXTURE_OVERLAY_FAILED`.

This is an intentionally narrow compatibility ruling, not an endorsement of
control-bearing filenames and not a portability claim. Pure parity vectors pin
the constructor distinction; physical tests pin the local materialization
refusal. A later authority revision may close the HTTP seed-path language, but
A2.2 does not silently rewrite sealed A1 validity.
