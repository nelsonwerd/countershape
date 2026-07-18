# Lane 5 — operator DX and Wake fit

## Verdict

P07B-C is necessary truth plumbing, not yet the user-facing aha. P08 must compress it into one action:

> Check this recorded behavior against this exact candidate, using a fresh copy, and tell me whether it matched, differed, or could not be judged.

The default vocabulary budget is five product concepts: **contract, candidate, pinned commit, checked fields, result**. Target/run/publication objects and evidence digests belong under details/JSON.

## Future P08 result projection

| Core result | Human headline | Meaning |
| --- | --- | --- |
| `CONFORMS` | Matches | Checked fields matched one allowed exact combination. |
| `CONTRADICTS` | Differs | The run was usable, but checked fields differed from every allowed exact combination. |
| `INELIGIBLE_EXECUTION` | Could not judge | Countershape did not obtain a clean enough run to judge the checked behavior. |

Pre-target refusal, contract invalidity, usage error, and internal failure are not behavioral results. A future versioned CLI result contract should reserve distinct exit codes and make JSON/human/`NO_COLOR` modes agree.

The generated six-file contract remains the expert Countershape-absent escape hatch. Its manual three-path invocation and TAP output are coherent but are not the ordinary operator path.

## Wake boundary

Wake owns agent fleet execution, event history, coordination, crash resume, replay, fork, effects, and gates. Countershape owns exact Git candidate behavior comparison and persisted selected-field contracts. Legitimate integration is only:

```text
display label + repository + immutable ref + inert provenance
```

Countershape still pins and inspects the Git object itself. Countershape does not consume Wake sessions, events, gates, or runtime authority; it must not launch coding agents, reconstruct Wake timelines, or provide agent-run resume/replay.

## Still bets

- Builders will tolerate the pinned-contract workflow.
- Matches/differs/could-not-judge compresses review better than diff-first work.
- Users understand that standalone is not sandboxed or portable.
- A generic immutable-ref handoff is sufficient integration.

The brief’s real comprehension and review-compression gates remain human/market work.

## Confidence

**7/10.** Eleven of thirteen conclusions were checked in Countershape/Wake source; adoption and wording remain judgment.
