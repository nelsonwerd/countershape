# Runtime-authority lane

## Verdict

The timeout repair belongs in the parent-owned
`tools/verify-runtime-authority.mjs`. C5 must not introduce a second
`spawnSync` wrapper or rebuild child options locally.

The narrow design is:

- default outer child deadline exactly `1_200_000 ms`;
- optional bounded execution policy exactly `1_800_000 ms`;
- an exact two-row policy owned and frozen before C5 activates;
- one spawn, no retry;
- unchanged executable admission, arguments, working directory, environment,
  encoding, output limit, result normalization, and pre/post executable
  revalidation.

## Existing authority

`childResult()` currently owns the full process boundary:

1. revalidate every declared tool;
2. select the admitted executable;
3. construct fixed spawn options;
4. execute once;
5. revalidate every declared tool again;
6. return a frozen status/signal/error/stdout/stderr record.

The 20-minute value is a private runtime constant. That is why a C5-local
wrapper would be structurally wrong: it would create another execution
authority whose environment, output ceiling, path admission, and revalidation
could diverge.

## Recommended runtime interface

Preserve the existing fourth argument, which carries test seams such as
injected arguments, revalidation, and spawn. Add a fifth argument:

```js
childResult(
  step,
  admitted,
  childEnvironment,
  dependencies = {},
  executionPolicy = undefined,
)
```

The fifth argument is not a dependency injection seam. It is a bounded policy
record consumed and validated by the runtime before authority I/O.

Accepted shapes:

- omitted fifth argument: exact default `1_200_000`;
- an exact data record containing only own `timeoutMS: 1_800_000`.

Rejected shapes include:

- `null`, explicit `undefined`, arrays, functions, nonordinary prototypes;
- inherited `timeoutMS`;
- accessors;
- unknown or extra keys;
- numeric strings, fractions, `NaN`, infinities, `BigInt`;
- zero, negatives, unsafe integers;
- any value other than the exact admitted extension.

The exact-value design is intentionally narrower than a continuous range. The
current need is one bounded extension, not a general timeout configuration
surface.

## Verifier policy

The verifier owns a frozen, prototype-safe two-key policy:

```text
architecture-p07b-c-c5          → 1_800_000
architecture-p07b-c-c5-selftest → 1_800_000
```

The lookup must use own-property semantics. A null-prototype record is
preferred; a `Map` is acceptable only if mutation is structurally prevented,
because `Object.freeze(new Map())` does not freeze its entries.

Policy stays outside `currentSteps`. Rows describe the commands whose evidence
is collected; they do not grant their own execution envelope. The executor
derives policy from the canonical enumerated row ID, not caller metadata.

The exact table is parent-governed even though it lives beside the verifier:
C4J's plan checker freezes its source shape, and C5 does not own the plan
checker. C5 later adds the canonical rows but cannot broaden the table without
failing the unchanged parent gate.

The current governance edge is `C4I → C4K → C4J → C5`. Active C4J owns this
two-row verifier policy after exact sealed C4K; C4K changed only the C3 Go
deadline and neither activated nor authorized C5 product work. Live phase
authority is v24 with 26 ordered receipt rows.

## Required self-test

The test must observe the actual `spawn` options rather than only inspect a
plan object.

Positive controls:

- omitted policy passes exactly `1_200_000`;
- each exact named-row policy passes exactly `1_800_000`;
- cwd, env object identity, encoding, max buffer, executable, and argv remain
  exact;
- result status, signal, error identity, stdout, and stderr remain exact;
- pre- and post-revalidation occur once each;
- spawn occurs once.

Negative controls:

- every malformed policy shape fails before spawn;
- `constructor`, `toString`, `__proto__`, typos, aliases, and unknown row IDs
  cannot elevate;
- a caller-owned `timeoutMS` property on a step is inert;
- no retry occurs after a simulated `ETIMEDOUT`;
- thrown-spawn plus failed post-revalidation remains an `AggregateError`;
- returned-timeout plus failed post-revalidation retains the existing
  diagnostic precedence rather than silently redesigning it.

The repetition helper continues to omit the fifth argument and therefore
remains at the 20-minute outer default. Its internal Go `-timeout=20m` remains
unchanged.

## Rejected alternatives

### Global 30-minute default

Rejected because it broadens every row and every shared runtime caller.

### C5-local child wrapper

Rejected because it recreates authority below the parent runtime.

### Step-owned timeout metadata

Technically viable with a strict allowlist, but declined here because it mixes
command description with execution authority and gives a row a place to
request its own privilege. A separately frozen policy table makes the trust
boundary clearer.

### Profile splitting or parallelism

Rejected. The aggregate checker deliberately encloses five profiles in one
before/after snapshot. Splitting changes the property proved; parallelism
changes load and ordering in an already load-sensitive Darwin suite.

### Retry

Rejected. A timeout remains a failed execution. The repair changes the maximum
allowance, not failure semantics.

## Findings

- **Blocker:** shared runtime must remain the sole spawn-options authority.
- **High:** policy shape must be exact and prototype-safe.
- **High:** the self-test must observe the actual timeout passed to spawn.
- **High:** both C5 aggregates need the extension.
- **Medium:** diagnostic precedence for a returned timeout plus failed
  post-revalidation should be characterized, not opportunistically changed.
- **Note:** a 30-minute cap still requires real-machine C5 receipts.

## Confidence

**8/10.** Ground-truth tally: 9 of 11 conclusions come directly from current
runtime/verifier source and captured durations. The policy-table placement and
30-minute adequacy remain design judgments until implementation and receipts.
