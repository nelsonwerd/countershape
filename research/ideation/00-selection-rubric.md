# Phase 1 selection rubric

This is the convergence gate after the Wake collision. It prevents the most polished option from winning merely because it resembles an already-built system.

## Automatic rejection tests

Reject an option before scoring if any answer is yes:

1. Is its source of truth an agent run/event log?
2. Is its primary job supervising, resuming, replaying, or forking an agent fleet?
3. Is its aha moment inspecting what agents did in a causal timeline?
4. Would replacing its implementation with Wake plus a thinner UI preserve most of the value?
5. Does its offline proof require pretending a model, hosted sandbox, or third-party API exists?
6. Is the "new primitive" actually a prompt convention, task board, trace dashboard, or fixed test runner?

## Weighted score

Score each surviving concept 1-5, then apply the weight.

| Criterion | Weight | What earns a 5 |
| --- | ---: | --- |
| New systems primitive | 22 | A crisp source of truth and operation that existing agent tools do not already expose. |
| Eyebrow-raising demo | 16 | One local session makes a previously awkward or impossible action feel obvious. |
| 2026 grounding | 14 | Directly answers a visible workflow or correctness failure without inventing demand. |
| Vibe-coder legibility | 12 | A serious non-expert can make a safer decision without becoming a platform engineer. |
| Engineering depth | 12 | Correctness depends on real semantics, isolation, deterministic machinery, or hard evidence—not UI volume. |
| Offline proofability | 10 | The load-bearing claim can be exercised locally with real commands and no fake APIs. |
| Coherent heavy scope | 8 | Large enough to stress the build, small enough to have one load-bearing spine. |
| Open-source extension surface | 6 | Adapters or protocols create adoption pull without owning hosted execution. |

## Tie-breakers

Prefer the concept that:

- treats nondeterministic agents as optional producers rather than trusted judges;
- creates new evidence instead of repackaging logs;
- can consume ordinary branches, patches, or artifacts from any tool;
- distinguishes disagreement from failure and evidence from truth;
- has a useful golden path before any model credential is supplied;
- would compose with Wake cleanly while remaining valuable without it.

## Required convergence output

The chosen concept must lock all of these in `docs/CONCEPT_BRIEF.md`:

- source of truth;
- primary user job;
- one-sentence promise;
- first 15-minute aha moment;
- smallest falsifiable proof;
- why Git/worktrees, parallel-agent dashboards, visual regression, tests, and Wake do not already provide it;
- exact v1 scope and named deferrals;
- validated facts, technical inferences, and unresolved bets kept separate.
