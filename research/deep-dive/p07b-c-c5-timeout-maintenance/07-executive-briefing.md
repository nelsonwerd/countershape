# Executive briefing: C5 timeout maintenance

## TL;DR

C5 did not reveal a failed product assertion. Its third full verification
attempt reached a hard 20-minute wrapper deadline after the same aggregate had
already passed twice in about 18.7–18.8 minutes. Separately, the verifier still
runs all twelve timing-sensitive packages in one row even though the live
status promises a safer serial 9+3 split.

The correct response is to restore C4J after exact sealed C4K and seal it as a
separate maintenance boundary:

1. C4J keeps 20 minutes as the universal default.
2. It permits 30 minutes only for the two named C5 aggregate rows.
3. The existing shared runtime remains the only process-execution authority.
4. C4J inserts itself after sealed C4K in the machine phase chain and seals.
5. C5 resumes, implements the promised 9+3 split, and starts all evidence over.

Nothing in this repair lowers a test count, removes a check, retries a failed
command, parallelizes load-sensitive work, or changes C5's frozen 79-command
receipt.

## What the evidence actually says

- Two complete C5 cumulative runs passed all 61 internal rows.
- Their C5 aggregate rows used more than 93% of the outer deadline.
- A third run passed rows 1–48 and then ended at `1,200,027 ms` with
  `ETIMEDOUT`.
- The combined sensitive row had just passed in `1,161,075 ms`.
- No C5 assertion failure was reported in that attempt.

This makes the 20-minute wrapper an under-budgeted liveness boundary. It does
not prove that 30 minutes will always be sufficient; only fresh execution can
prove that on this machine.

## Load-bearing design

The runtime accepts one exact bounded policy while retaining all other child
controls. A parent-frozen, prototype-safe table grants that policy only to:

- `architecture-p07b-c-c5`
- `architecture-p07b-c-c5-selftest`

Every other caller stays at 20 minutes. Inner Go/checker deadlines remain
unchanged. A timeout still fails immediately; there is no retry.

C4J migrates live phase authority to v24/26, establishing:

```text
C4I → sealed C4K → active C4J → C5
```

C4I's and C4K's sealed documents and grades remain historical facts. C5's
existing ancestry command keeps its exact name and argv but dynamically
traverses `C4J → C4K → C4I`.

## What remains unchanged

- C5 outer ledger: 79 commands, 76 test claims, three command claims.
- C5 labels, types, argument vectors, and order.
- C5 qualification catalog and digest.
- C5 source-path and prefix digests.
- C6's frozen 11/10/9 receipt arrays.
- All Go `-timeout=20m` values.
- `p=1`, `GOMAXPROCS=2`, `-parallel=2`.
- Fresh private caches, exclusive lock, serial order, and all assertions.

## What intentionally changes

- New C4J phase/path/narrative/source identities.
- Shared runtime/verifier source digest.
- After C4J, C5's internal verifier grows from 61 to 62 rows because one
  sensitive row becomes exact inherited-nine plus exact C5-three.

## Stop conditions

Do not resume C5 unless C4J:

- validates exact sealed C4K as its direct parent and proves the C4K-to-C4I
  link plus lower ancestry;
- passes its targeted runtime-policy selftest;
- passes the unchanged repetition-runner selftest;
- passes three complete cumulative verifications;
- has exact 19-path scope;
- commits and seals;
- exposes the expected Git note;
- passes `NO_COLOR=1 didrun verify --strict`;
- produces HTML evidence and a retained ledger archive.

Do not seal C5 using any pre-C4J event. All 79 commands start again.

## Honest verdict

Proceed. The repair is bounded, preserves evidence strength, and places
authority in the right layer. The remaining bet is operational rather than
conceptual: whether 30 minutes provides enough real load headroom and whether
the long atomic C5 ledger reveals another unrelated late defect.

## Confidence

**8/10.** Ground-truth tally: 13 of 16 load-bearing conclusions are supported
by exact source, sealed identities, captured transcripts, or frozen receipt
arrays. The 30-minute sufficiency, future ledger stability, and absence of a
shared-model blind spot remain unvalidated.
