# Adversarial red-team

## Initial verdict

The synthesis was rejected as underspecified, then accepted conditionally
after the corrections below were incorporated into the C4J contract.

## Blockers and resolutions

### 1. Execution policy could have accepted ambiguous JavaScript shapes

**Attack:** a naïve `policy?.timeoutMS ?? default` accepts inherited values,
accessors, extra keys, coercible strings, or explicit invalid values and can
silently fall back.

**Resolution:** runtime accepts no fifth argument or one exact ordinary/null-
prototype data record with only own `timeoutMS: 1_800_000`. No coercion,
clamping, accessors, inherited values, extra keys, or alternate numbers.
Invalid input fails before spawn.

### 2. A frozen plain object can still inherit dangerous lookup keys

**Attack:** `constructor`, `toString`, or `__proto__` can resolve through the
prototype; `Object.freeze(new Map())` does not freeze map entries.

**Resolution:** exact two-key null-prototype record, own-property lookup,
canonical enumerated step IDs, and exhaustive hostile lookup controls. Step
metadata cannot elevate.

### 3. Three cumulative passes could obscure missing targeted proof

**Attack:** the full verifier may stay green without demonstrating that the
new fifth argument reaches `spawn` or that malformed policy fails before
execution.

**Resolution:** claim 6 is a dedicated runtime/verifier policy selftest. It
observes actual spawn options, invalid no-spawn behavior, one-spawn/no-retry,
result identity, and both revalidations. Cumulative passes are additional
regression evidence only.

### 4. C4J changes every verifier-control layer at once

**Attack:** runtime, verifier, selftest, plan checker, scope checker, and phase
spec could bless one another.

**Resolution:** bind exact sealed C4K as the direct parent before candidate
acceptance; independently prove `C4K → C4I` and the lower chain; keep old and
new authority identities explicit; exercise hostile fixtures for every exact
C4K identity/link/outer-HEAD substitution and each changed validator; require
exact 19-path stage and reject additions, omissions, substitutions, symlinks,
and unstaged drift; validate C4I independently; make C4J preseal enforce exact
labels/types/argv/grades/order/coverage.

### 5. A single phase-row insertion is insufficient

**Attack:** C5 or C6 fixtures could still accept the old ancestry even when the
visible phase table says C4J exists.

**Resolution:** migrate schema, order, parent rules, phase digest, transition
matrices, generated fixtures, Markdown corpus, runbook registry, cardinality
maps, C5 dynamic ancestry, and C6 future controls together to v24/26 and exact
`C4J → C4K → C4I`. Add skip, merge, stale-head, and substitution hostiles.
Preserve the frozen C4J claim labels and CLI spellings as compatibility names
instead of rewriting their receipt vocabulary.

### 6. Digest churn could be rationalized after the fact

**Attack:** an unexpected qualification or outer-receipt change might be
documented as harmless.

**Resolution:** declare expected rotations before implementation:

- C4J phase/path/corpus/source-closure identities rotate;
- C4J step-roster digest stays exact;
- C5 step-roster rotates only when 61 becomes 62;
- C5 qualification/path/prefix/outer-ledger identities stay exact;
- C6 11/10/9 arrays stay exact.

Unexpected movement is a defect.

### 7. Sensitive splitting could leak into C4J

**Attack:** pre-landing the split in C4J would entangle product and parent
authority and make stale C5 evidence tempting to reuse.

**Resolution:** C4J retains 61 rows. C5 alone lands the 9+3 split after C4J
seals, and all C5 evidence restarts.

### 8. Research files could become accidental authority

**Attack:** untracked or separately committed research could violate direct
ancestry; tracked prose might enlarge normative authority invisibly.

**Resolution:** the seven files are explicitly included in C4J's 19-path
scope, labeled non-normative, and committed atomically with C4J. Validators
derive no execution or phase decisions from them.

## Residual risks

- The same model family produced specialist, synthesis, and red-team work;
  shared blind spots remain possible.
- The 30-minute maximum is evidence-informed but not yet receipted.
- Full-ledger duration remains long enough that unrelated latent defects can
  still surface.
- The phase checker is large; only exhaustive selftests and sealed ancestry
  protect against a missed C4K/C4J fixture.

## Final verdict

**Approve after all eight resolutions are implemented and tested.** Do not
proceed to C5 on a merely green targeted test; C4J must commit, seal, expose
its note, pass strict verification, render HTML, and archive its ledger.

## Confidence

**7/10.** Ground-truth tally: 11 of 15 load-bearing red-team conclusions are
anchored in current source, receipt structures, or captured execution.
Prototype-safe implementation details and the runtime adequacy claims remain
unproven until tests and real runs complete.
