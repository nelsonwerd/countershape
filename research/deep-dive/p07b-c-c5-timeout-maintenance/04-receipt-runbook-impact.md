# Receipt and runbook impact lane

## Verdict

C4J and the C5 sensitive-lane split can land without changing the frozen outer
C5 receipt:

- exactly 79 commands;
- exactly 76 `tests-pass`;
- exactly three `command-succeeded`;
- identical labels, types, argv, and order;
- identical qualification case catalog and digest;
- identical C6 11/10/9 receipt shapes.

The final three C5 test claims already invoke `node tools/verify-current.mjs`.
An internal verifier change from 61 to 62 rows changes what those commands
execute, not their outer argv.

## C4J claim profile

C4J uses 13 claims: ten `tests-pass`, then three `command-succeeded`.

1. C4J candidate phase-plan coherence
2. C4J independent candidate-transition authority
3. C4J plan-checker defensive self-test
4. C4J sealed-C4I note and lower-ancestry compatibility
5. C4J unit-scope defensive self-test
6. C4J runtime/verifier timeout-policy defensive self-test
7. C4J repetition-runner defensive self-test against the changed runtime
8. C4J cumulative verification pass 1
9. C4J cumulative verification pass 2
10. C4J cumulative verification pass 3
11. C4J exact 19-path staged scope and diff integrity
12. C4J scoped staged credential-pattern scan
13. C4J sealed-C4I predecessor and preceding didrun-chain integrity

Commands 11 through 13 are the terminal command claims. C4J preseal validates
the first 12 events plus exact label/type/argv/order/coverage before its own
claim can be recorded.

Claims 4 and 13 deliberately retain their frozen `sealed-C4I` wording. They
name the C4I-and-lower compatibility subchain, not C4J's direct parent. The
implementation first binds exact sealed C4K, proves `C4K → C4I`, and only then
traverses the named lower ancestry. The same compatibility rule preserves
`--verify-c4j-sealed-c4i-note`.

The three cumulative passes do not replace claim 6. The targeted test is the
direct proof that malformed policies cannot spawn and that the actual spawn
receives the intended bound.

## Digest effects

Expected:

- C4J phase authority: v24 with 26 ordered receipt rows at
  `30530a2e05fa6b7fb74983756c24f22a898f4a6da6df75d447df17278b0152db`;
- C4J exact-path digest: new;
- live Markdown corpus: 69 paths at
  `sha256:a53acdeef0e86b4865caab7822ea06cfa62742794ff1aaa5816023ef172cf901`,
  with 1,065 hostile rejections and 355 controls;
- runtime/verifier source-closure digest: rotate;
- C4J current-step roster digest: unchanged, because timeout policy remains
  outside `currentSteps` and the sensitive split is not part of C4J;
- C5 current-step roster digest: rotates when its internal rows become 62;
- C5 qualification digest: unchanged;
- C5 exact-path and prefix digests: unchanged;
- C5 outer claim labels/types/argv/order: unchanged;
- C6 outer receipt arrays: unchanged.

An unexpected digest change is a defect to diagnose, not something to explain
away in prose.

## C5 activation after C4J

After C4J later earns a seal and strict-green result:

1. reapply the retained C5 checkpoint;
2. resolve only expected overlaps in verifier selftest and live docs;
3. keep the exact parent C4J;
4. activate the already parent-frozen two-row timeout policy;
5. split the sensitive set into exact inherited-nine and C5-three rows;
6. prove sorted/unique/disjoint/exact-union composition and revalidation;
7. retain `-p=1`, `-parallel=2`, `-count=1`, `-timeout=20m`;
8. run fresh qualification and aggregate evidence;
9. start the frozen 79-command final ledger again at command 1.

The prior two cumulative greens and one timeout are permanent diagnostic
history only.

## C5 ancestry compatibility

Do not rename:

```text
--verify-c5-sealed-c4-note
```

Do not edit its frozen claim label merely to enumerate C4J or C4K. The
implementation loads sealed C4J, requires its exact direct parent C4K, proves
`C4K → C4I`, and then traverses the existing lower chain. The stable umbrella
wording already permits maintenance boundaries without churning the receipt
vocabulary.

## Sensitive split acceptance

- inherited list exact nine;
- C5 list exact three;
- each sorted and unique;
- sets disjoint;
- union exact original twelve;
- no package remains in general;
- missing, foreign, duplicate, reordered, or swapped subgroup fails;
- both rows adjacent and serial;
- package-list revalidation catches subgroup drift;
- terminal result reports `62/62`;
- C5 seven-row product block remains contiguous.

## Findings

- **Blocker:** C5 must not create a second child-execution authority.
- **High:** frozen outer C5 receipt arrays remain byte-exact.
- **High:** qualification digest remains unchanged.
- **High:** internal verifier roster intentionally rotates only at the 9+3
  split.
- **High:** all pre-C4K/pre-C4J C5 events are stale for sealing.
- **Medium:** C6 keeps its existing direct C5/C6A/C6M relationships.
- **Note:** the new deadline reduces one known atomic-ledger failure mode but
  does not prove the full long ledger cannot expose another late defect.

## Confidence

**8/10.** Ground-truth tally: 10 of 12 load-bearing conclusions are directly
encoded in current claim arrays, runbook commands, digest functions, and phase
relationships. Operational 30-minute sufficiency and future full-ledger
stability remain bets.
