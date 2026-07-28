# C5 cumulative-verifier failure: empirical lane

## Question

Does C5 have a product-correctness failure, a verifier deadline defect, a
package-lane composition defect, or some combination of those?

This lane is deliberately narrow. It uses the staged-tree identity, the three
captured cumulative-verifier transcripts, and the current verifier/docs. It
does not infer product correctness from a timeout, and it does not reuse any
development event as seal evidence.

## Fixed input

- Sealed parent at the time of the captured C5 attempts: C4I commit
  `d784ace97dd03660c6c724e1cb8f838b869681ec`, tree
  `e38b702884b9df128b41bfa7710459dd92f901d7`.
- C5 staged tree:
  `20bc8691770467320b6ddadbebe4ebfeefe07564`.
- C5 checkpoint after this analysis:
  `1c9dd947f61e3f1fd4c4c2578da0e104f1a02b56`.
- Frozen outer C5 ledger: 79 commands, 76 `tests-pass`, then three
  `command-succeeded`.
- The three transcripts are diagnostic development evidence only. They are not
  didrun claims for the future C5 seal.
- Current context: C4K later sealed between C4I and active C4J; C5 remains
  blocked until C4J seals.

## Observed results

### Pass 1

- Transcript:
  `/tmp/countershape-c5-cumulative-pass-1-20bc86917704.log`
- SHA-256:
  `cac3d71349caa037971416b86256c6f9300358f786c4b3430bdbaa883bb49660`
- Result: `61/61` current rows passed.
- Total observed wall time: `8,892,618 ms`.
- `architecture-p07b-c-c5`: `1,128,605 ms`.
- `architecture-p07b-c-c5-selftest`: `1,128,693 ms`.

### Pass 2

- Transcript:
  `/tmp/countershape-c5-cumulative-pass-2-20bc86917704.log`
- SHA-256:
  `e682a467bcf26771504ec3de334098d068603d259423608062f8688bfdee197f`
- Result: `61/61` current rows passed.
- Total observed wall time: `8,869,921 ms`.
- `architecture-p07b-c-c5`: `1,120,532 ms`.
- `architecture-p07b-c-c5-selftest`: `1,123,459 ms`.

### Pass 3

- Transcript:
  `/tmp/countershape-c5-cumulative-pass-3-20bc86917704.log`
- SHA-256:
  `149d344188f4afce77aa6b4aae8d3c370402cea027015abdc712860d50dccb38`
- Rows 1 through 48 passed.
- The combined twelve-package sensitive row passed in `1,161,075 ms`.
- Row 49, `architecture-p07b-c-c5`, failed after `1,200,027 ms`.
- Failure detail:
  `spawnSync /opt/homebrew/Cellar/node/25.2.1/bin/node ETIMEDOUT`.
- Rows 50 through 61 did not run.

The two successful C5 aggregate executions consumed about 94.1% and 93.4% of
the current 1,200,000 ms outer child bound. The third attempt ended almost
exactly at that bound. This is direct evidence that the outer allowance lacks
ordinary load-variance headroom. It is not evidence that an inner semantic
deadline fired, that an assertion failed, or that the product result was
incorrect.

## Independent implementation/documentation defect

`docs/status/P07B-C-C5-HTTP-SCOPE.md` says C4I split the twelve sensitive
packages into an inherited-nine row and a C5-three row. The staged
`tools/verify-current.mjs` still has one `go-test-sensitive-serial` row over
all twelve packages. This mismatch is a defect even though the combined row
passed twice and passed again immediately before the pass-3 timeout.

The intended C5-specific subgroup is:

1. `github.com/nelsonwerd/countershape/internal/contractexec/http`
2. `github.com/nelsonwerd/countershape/internal/contractexec/scope`
3. `github.com/nelsonwerd/countershape/testkit/contractexec/http`

The inherited subgroup is the other nine packages. Both must remain serial,
retain `-p=1`, `-parallel=2`, `-count=1`, and Go's own `-timeout=20m`, and be
proven sorted, unique, disjoint, and exact-union complete.

## Findings

### Blocker

The shared runtime has a single global 20-minute outer child deadline. It can
terminate a valid C5 aggregate before the aggregate returns.

### High

The sensitive-lane implementation contradicts the live C5 status. The repair
must split one internal verifier row into two without deleting coverage.

### High

Both the C5 architecture row and its aggregate self-test need the bounded
extension. The two successful self-test durations are also within roughly
80 seconds of the current ceiling.

### High

The pre-C4K/pre-C4J pass-1 and pass-2 greens become stale when runtime
authority, phase ancestry, or verifier composition changes. They may motivate
the repair but cannot be reused in the final C5 ledger.

### Note

A 30-minute maximum is proportionate and finite, but sufficiency is still an
operational bet until both aggregate rows return under fresh didrun evidence.
If either reaches the new ceiling, the next step is diagnosis rather than
another automatic increase.

## Confidence

**8/10.** Ground-truth tally: 7 of 8 load-bearing conclusions are backed by
captured transcripts, exact source, or exact documentation; the remaining
conclusion is the unproven operational adequacy of a 30-minute maximum.
