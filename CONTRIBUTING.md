# Contributing

Countershape changes are invariant-first. State the authority being consumed, the narrower authority being produced, the refusal behavior, and the resource ceiling before changing code.

1. Preserve typed adapter boundaries. HTTP and CLI facts do not become generic strings simply because their shapes look alike.
2. Add hostile fixtures and mutants for every new seam: aliasing, stale lineage, same-shape/different-map, noncanonical bytes, unsafe filesystem topology, cancellation, and output bounds.
3. Keep physical execution and semantic authority separate. Tests that exercise DTOs do not replace fresh process evidence.
4. Treat didrun as an append-only evidence ledger. Run load-bearing checks through the owning unit, declare only supported claims, and never erase or relabel failed evidence.
5. Do not weaken a test or cache physical evidence to meet a timing target.

## Adding an adapter

Define a domain-specific stimulus, capture, and projection contract; bind exact executable/materialization authority; declare comparison dimensions and resource budgets; preserve fresh-run lineage; and expose only typed facts to choice/reduction. Add package tests, cross-boundary hostiles, and a dependency-free fixture. Do not coerce a new domain into HTTP or CLI vocabulary.

Run `make test`, the relevant race/mutation suites, `make release`, and `make smoke`. Record unsupported tooling as `UNRECEIPTED` rather than inventing a result.
