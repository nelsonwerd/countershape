# P07B-C deep dive — scope and method

**Date:** 2026-07-18
**Reviewed boundary:** commit `464e47adbf7f4497dfafa89938a9539239ffd41b` on `codex/countershape-autopilot`
**Mode:** six read-only specialist lanes, synthesis, different-model red team, three single-claim follow-ups, and two post-write adversarial coherence reviews

## Question

What is the smallest sound P07B-C design that can publish an exact pre-spawn target, close one physical contract run, and persist its classification without reusing historical comparison authority, executing dirty working-tree bytes, turning metadata into evidence, or creating a mutable execution head?

## Constraints

- Specialists were read-only. They did not edit the repository, run tests/builds/didrun, or spawn a subject.
- The root remained the sole repository, Git, and didrun writer.
- The review considered current code and controlling documents, not chat memory, authoritative.
- The existing P07B-C schemas were treated as planning artifacts open to correction.
- Countershape remains a local trusted-code instrument, not a sandbox or process-resume system.
- Wake may hand off immutable Git refs and inert provenance only; Countershape does not consume Wake sessions, event logs, gates, replay, or agent lifecycle.
- The generated Node test remains an independent direct-run surface. Its TAP output never becomes Go execution authority.

## Lanes

1. Immutable authority and canonical data model.
2. Runtime admission, process lifecycle, and scoped standalone evidence.
3. Store publication, crash recovery, and nonhead persistence.
4. Verification architecture and didrun claim ceilings.
5. Operator/DX coherence and Wake non-duplication.
6. Alternative architecture and adversarial prompt interpretation.

The synthesis initially proposed two persisted semantic objects and a derived classification view. A different-model red team rejected that simplification because it erased the immutable historical conclusion. Focused verification restored the three-object split. The post-write reviews then found that target-local at-most-once admission could still overlap an unknown survivor on a fresh target, so the final design adds one private boot-session `ExecutionInterlock`. It is the only mutable operational exception, can only block spawn, and owns no semantic result/discovery authority.

## Evidence status

This package is a design review. Repository facts were checked against the sealed tree; all proposed P07B-C behavior remains `UNRECEIPTED` until its owning implementation unit runs through didrun, commit, seal, Git-note inspection, and strict verification.
