#!/usr/bin/env node

import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import {
  chmod,
  lstat,
  mkdir,
  mkdtemp,
  readFile,
  readdir,
  realpath,
  rm,
  symlink,
  unlink,
  writeFile,
} from "node:fs/promises";
import { tmpdir } from "node:os";
import {
  dirname,
  isAbsolute,
  join,
  relative as relativePath,
  resolve,
  sep,
} from "node:path";
import { fileURLToPath } from "node:url";

import {
  MutationGateError,
  NamedTestOutcome,
  applyExactMutation,
  copyRegularAllowlist,
  mutateRegularFileNoFollow,
  readAllowlistManifest,
  revalidateAdmittedGoExecutable,
} from "./mutate-u1.mjs";
import {
  admitU2DarwinToolchain,
  minimalU2GoEnvironment,
  revalidateAdmittedDarwinCCompiler,
  runNamedU2GoTest,
  runU2Mutant,
} from "./mutate-u2.mjs";
import {
  U4_REVIEWED_NON_GO_FILES,
  U4_SANDBOX_FILE_ALLOWLIST,
  U4_SOURCE_SCOPE_ROOTS,
} from "./mutate-u4.mjs";

const modulePath = fileURLToPath(import.meta.url);
const repoRoot = resolve(dirname(modulePath), "..");

// U1-U4 remain byte-for-byte independent contracts. U5 composes the exported
// U4 positive closure and explicitly admits only the new reducer packages,
// typed neighbor files, and physical reduction studies required by U5 tests.
export const U5_ADDITIONAL_FILES = Object.freeze([
  "internal/adapters/cli/neighbors.go",
  "internal/adapters/cli/neighbors_test.go",
  "internal/adapters/http/neighbors.go",
  "internal/adapters/http/neighbors_test.go",
  "internal/reduce/draft.go",
  "internal/reduce/engine.go",
  "internal/reduce/engine_test.go",
  "internal/reduce/identity.go",
  "internal/reduce/model.go",
  "internal/reduce/model_test.go",
  "internal/reduce/wire.go",
  "internal/reduce/wire_test.go",
  "internal/reduction/finalize.go",
  "internal/reduction/finalize_test.go",
  "internal/store/reduction_sweep.go",
  "internal/store/reduction_sweep_test.go",
  "testkit/clifixture/fixture.go",
  "testkit/clifixture/fixture.mjs",
  "testkit/clifixture/fixture_test.go",
  "testkit/studies/cli_precedence/reduction_darwin_test.go",
  "testkit/studies/cli_precedence/study.go",
  "testkit/studies/cli_precedence/study_darwin_test.go",
  "testkit/studies/http_invoices/reduction_darwin_test.go",
]);

export const U5_SANDBOX_FILE_ALLOWLIST = Object.freeze([
  ...U4_SANDBOX_FILE_ALLOWLIST,
  ...U5_ADDITIONAL_FILES,
]);

export const U5_SOURCE_SCOPE_ROOTS = Object.freeze([
  ...U4_SOURCE_SCOPE_ROOTS,
  "internal/reduce",
  "internal/reduction",
  "internal/store",
  "testkit/clifixture",
  "testkit/studies/cli_precedence",
]);

export const U5_REVIEWED_NON_GO_FILES = Object.freeze([
  ...U4_REVIEWED_NON_GO_FILES,
  "testkit/clifixture/fixture.mjs",
]);

export const U5_REVIEWED_EMBED_BINDINGS = Object.freeze([
  Object.freeze({
    owner: "testkit/httpfixture/fixture.go",
    asset: "testkit/httpfixture/server.mjs",
    directive: "//go:embed server.mjs",
  }),
  Object.freeze({
    owner: "testkit/clifixture/fixture.go",
    asset: "testkit/clifixture/fixture.mjs",
    directive: "//go:embed fixture.mjs",
  }),
]);

export const REQUIRED_U5_MUTANT_IDS = Object.freeze([
  "replace-full-map-preservation-with-partition-shape",
  "collapse-unresolved-into-changes",
  "skip-fresh-final-sweep",
  "ignore-parent-cancellation",
  "ignore-parent-cancellation-during-final-enumeration",
  "promote-wall-budget-fencepost",
  "promote-proposal-budget-fencepost",
  "promote-trial-budget-fencepost",
  "accept-map-backed-trial-count-mismatch",
  "drop-evaluator-error-trial-accounting",
  "drop-observed-preservation-map-digest-from-wire",
  "accept-unknown-canonical-transcript-fields",
  "allow-midstream-terminal-protocol-refusal",
  "allow-disconnected-transcript-parent-chain",
  "allow-replay-baseline-evidence-reuse",
  "bind-transcript-identity-to-wall-progress",
  "accept-nondecreasing-neighbor",
  "allow-reused-evaluation-evidence",
  "treat-group-ordinal-as-preservation-identity",
  "replace-declared-rule-priority-with-lexical-order",
  "replace-transform-priority-with-child-digest-order",
  "omit-reducer-set-digest-from-strong-grade",
]);

// This is a second immutable contract, intentionally separate from the ID
// list and the executable clone below. One changed byte closes the gate.
export const REVIEWED_U5_MUTANT_CONTRACT = Object.freeze([
  Object.freeze({
    id: "replace-full-map-preservation-with-partition-shape",
    file: "internal/reduce/model.go",
    find: [
      "\tassessment := compare.AssessPreservation(baseline, observed)",
      "\tif !assessment.Valid() {",
    ].join("\n"),
    replace: [
      "\tassessment := compare.AssessPreservation(baseline, observed)",
      "\tif len(baseline.DisplayGroups()) == len(observed.DisplayGroups()) {",
      "\t\tevaluation.decision = Preserves",
      "\t\tevaluation.reasonCode = \"PARTITION_SHAPE_MATCH\"",
      "\t\treturn evaluation, nil",
      "\t}",
      "\tif !assessment.Valid() {",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestClassifyNeighborIsTriValuedAndUsesExactMap",
  }),
  Object.freeze({
    id: "collapse-unresolved-into-changes",
    file: "internal/reduce/model.go",
    find: [
      "\t// MUTATION_ANCHOR: unresolved-is-not-changes",
      "\treturn Unresolved",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: unresolved-is-not-changes",
      "\treturn Changes",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestUnresolvedNeverCollapsesIntoChanges",
  }),
  Object.freeze({
    id: "skip-fresh-final-sweep",
    file: "internal/reduce/engine.go",
    find: [
      "\tif !sameProposalSet(searchProposals, proposals) {",
      "\t\ts.limit(\"FINAL_SWEEP_ENUMERATION_STALE\")",
      "\t\treturn",
      "\t}",
      "\tneighbors := make([]Neighbor, 0, len(proposals))",
    ].join("\n"),
    replace: [
      "\tif !sameProposalSet(searchProposals, proposals) {",
      "\t\ts.limit(\"FINAL_SWEEP_ENUMERATION_STALE\")",
      "\t\treturn",
      "\t}",
      "\t// Mutant: erase every known direct neighbor and certify an empty sweep.",
      "\tproposals = nil",
      "\tneighbors := make([]Neighbor, 0, len(proposals))",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestBoundedReducerAcceptsExactMapAndBuildsFreshCompleteDraft",
  }),
  Object.freeze({
    id: "ignore-parent-cancellation",
    file: "internal/reduce/engine.go",
    find: [
      "\tcase <-s.parentContext.Done():",
      "\t\t// MUTATION_ANCHOR: parent-cancellation-prevents-completion",
      "\t\treturn ReasonReductionCancelled",
    ].join("\n"),
    replace: [
      "\tcase <-s.parentContext.Done():",
      "\t\t// MUTATION_ANCHOR: parent-cancellation-prevents-completion",
      "\t\treturn UnresolvedReason(\"\")",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestCancellationDuringEmptySearchEnumerationCannotCompleteSweep",
  }),
  Object.freeze({
    id: "ignore-parent-cancellation-during-final-enumeration",
    file: "internal/reduce/engine.go",
    find: [
      "func (s *runState[S]) finalSweepRun(ctx context.Context, searchProposals []TypedProposal[S]) {",
      "\ts.finalState = FinalSweepIncomplete",
      "\tproposals, err := s.enumerate(ctx)",
      "\tif reason := s.stopReason(ctx); reason != \"\" {",
      "\t\ts.limit(string(reason))",
      "\t\treturn",
      "\t}",
    ].join("\n"),
    replace: [
      "func (s *runState[S]) finalSweepRun(ctx context.Context, searchProposals []TypedProposal[S]) {",
      "\ts.finalState = FinalSweepIncomplete",
      "\tproposals, err := s.enumerate(ctx)",
      "\t// Mutant: launder both cancellation sources after final enumeration.",
      "\ts.parentContext, ctx = context.Background(), context.Background()",
      "\tif reason := s.stopReason(ctx); reason != \"\" {",
      "\t\ts.limit(string(reason))",
      "\t\treturn",
      "\t}",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestCancellationDuringFinalEmptyEnumerationCannotCompleteSweep",
  }),
  Object.freeze({
    id: "promote-wall-budget-fencepost",
    file: "internal/reduce/engine.go",
    find: [
      "\t// MUTATION_ANCHOR: wall-budget-fencepost-is-exhausted",
      "\tif !s.clock.Now().Before(s.deadline) {",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: wall-budget-fencepost-is-exhausted",
      "\tif s.clock.Now().After(s.deadline) {",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestCancellationBudgetFencepostAndReusedEvidenceNeverProduceDraft",
  }),
  Object.freeze({
    id: "promote-proposal-budget-fencepost",
    file: "internal/reduce/engine.go",
    find: [
      "\t// MUTATION_ANCHOR: proposal-budget-equality-is-exhausted",
      "\tif s.proposalsUsed >= s.input.Budget.proposalLimit {",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: proposal-budget-equality-is-exhausted",
      "\tif s.proposalsUsed > s.input.Budget.proposalLimit {",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestProposalBudgetFencepostCannotBePromoted",
  }),
  Object.freeze({
    id: "promote-trial-budget-fencepost",
    file: "internal/reduce/engine.go",
    find: [
      "\t// MUTATION_ANCHOR: candidate-trial-budget-equality-is-exhausted",
      "\tif s.trialsUsed >= s.input.Budget.trialLimit {",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: candidate-trial-budget-equality-is-exhausted",
      "\tif s.trialsUsed > s.input.Budget.trialLimit {",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestCandidateTrialBudgetFencepostCannotBePromoted",
  }),
  Object.freeze({
    id: "accept-map-backed-trial-count-mismatch",
    file: "internal/reduce/engine.go",
    find: [
      "\t// MUTATION_ANCHOR: map-backed-trial-count-matches-attempt-evidence",
      "\tif observation.OutcomeMap != nil &&",
      "\t\tobservation.CandidateTrials != uint64(len(observation.OutcomeMap.EvidenceAttemptDigests())) {",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: map-backed-trial-count-matches-attempt-evidence",
      "\tif false && observation.OutcomeMap != nil &&",
      "\t\tobservation.CandidateTrials != uint64(len(observation.OutcomeMap.EvidenceAttemptDigests())) {",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestMapBackedEvaluationRejectsReportedTrialCountMismatch",
  }),
  Object.freeze({
    id: "drop-evaluator-error-trial-accounting",
    file: "internal/reduce/engine.go",
    find: [
      "\t\tobservation = EvaluationObservation{",
      "\t\t\tUnresolvedReason: ReasonEvaluatorError, UnresolvedEvidence: evidence,",
      "\t\t\tCandidateTrials: observation.CandidateTrials,",
      "\t\t}",
    ].join("\n"),
    replace: [
      "\t\t// Mutant: preserve refusal evidence but erase trials charged before the error.",
      "\t\tobservation = EvaluationObservation{",
      "\t\t\tUnresolvedReason: ReasonEvaluatorError, UnresolvedEvidence: evidence,",
      "\t\t}",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestEvaluatorErrorChargesReportedCandidateTrials",
  }),
  Object.freeze({
    id: "drop-observed-preservation-map-digest-from-wire",
    file: "internal/reduce/engine.go",
    find: "\t\t\tObservedOutcomeMapDigest: observedOutcome.String(), ObservedPreservationMapDigest: observedPreservation.String(),",
    replace: "\t\t\tObservedOutcomeMapDigest: observedOutcome.String(), ObservedPreservationMapDigest: observedPreservation.String()[:0],",
    package: "./internal/reduce",
    testName: "TestReductionWireRoundTripRetainsExactMapDigestsAndRepeatedFinalProposal",
  }),
  Object.freeze({
    id: "accept-unknown-canonical-transcript-fields",
    file: "internal/reduce/wire.go",
    find: [
      "\tif !bytes.Equal(canonicalBytes, exactCanonicalBytes) {",
      "\t\treturn TranscriptRecord{}, &domain.Error{Code: \"UNKNOWN_OR_NONEXACT_REDUCTION_TRANSCRIPT_FIELD\"}",
      "\t}",
    ].join("\n"),
    replace: [
      "\tif false && !bytes.Equal(canonicalBytes, exactCanonicalBytes) {",
      "\t\treturn TranscriptRecord{}, &domain.Error{Code: \"UNKNOWN_OR_NONEXACT_REDUCTION_TRANSCRIPT_FIELD\"}",
      "\t}",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestReductionWireRejectsUnknownCanonicalFields",
  }),
  Object.freeze({
    id: "allow-midstream-terminal-protocol-refusal",
    file: "internal/reduce/wire.go",
    find: [
      "\t\tif terminalProtocolRefusal &&",
      "\t\t\t(decision != Unresolved || outcome != nil || index != len(identity.Entries)-1 ||",
    ].join("\n"),
    replace: [
      "\t\tif terminalProtocolRefusal &&",
      "\t\t\t(decision != Unresolved || outcome != nil || false /* Mutant: permit a refusal before the final entry. */ ||",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestTranscriptParserRejectsMidstreamTerminalProtocolRefusal",
  }),
  Object.freeze({
    id: "allow-disconnected-transcript-parent-chain",
    file: "internal/reduce/wire.go",
    find: "\t\tif haveExpectedParent && (parent != expectedParent || before.digest != expectedMeasure.digest) {",
    replace: "\t\tif false && haveExpectedParent && (parent != expectedParent || before.digest != expectedMeasure.digest) {",
    package: "./internal/reduce",
    testName: "TestStandaloneTranscriptParserRejectsDisconnectedParentChain",
  }),
  Object.freeze({
    id: "allow-replay-baseline-evidence-reuse",
    file: "internal/reduce/engine.go",
    find: [
      "\t\tbaseline.BatchDigests(), baseline.EvidenceAttemptDigests(),",
      "\t\tbaseline.EvidenceWorldDigests(), baseline.EvidenceObservationDigests(),",
    ].join("\n"),
    replace: "\t\tnil, nil, nil, nil, // Mutant: omit the trusted baseline from replay nonreuse.",
    package: "./internal/reduce",
    testName: "TestSerializedTranscriptReplayRequiresTrustedBaselineAndNewNoncollidingEvidence",
  }),
  Object.freeze({
    id: "bind-transcript-identity-to-wall-progress",
    file: "internal/reduce/engine.go",
    find: [
      "\ttranscript, err := newTranscript(",
      "\t\ts.input.Baseline.OutcomeMap(), s.input.ReducerSet, s.input.Budget,",
      "\t\ts.entries, s.accepted, s.limitations, s.finalState, finalDigests,",
      "\t)",
    ].join("\n"),
    replace: [
      "\t// Mutant: leak host wall progress into otherwise logical transcript identity.",
      "\tnondeterministicBudget := s.input.Budget",
      "\tnondeterministicBudget.wallLimit += time.Duration(s.clock.Now().UnixNano())",
      "\ttranscript, err := newTranscript(",
      "\t\ts.input.Baseline.OutcomeMap(), s.input.ReducerSet, nondeterministicBudget,",
      "\t\ts.entries, s.accepted, s.limitations, s.finalState, finalDigests,",
      "\t)",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestTranscriptIdentityExcludesNondeterministicWallProgress",
  }),
  Object.freeze({
    id: "accept-nondecreasing-neighbor",
    file: "internal/reduce/model.go",
    find: "\t\t!input.CurrentMeasure.Valid() || !input.Measure.Valid() || !strictlyDecreases(input.CurrentMeasure, input.Measure) ||",
    replace: "\t\t!input.CurrentMeasure.Valid() || !input.Measure.Valid() || false ||",
    package: "./internal/reduce",
    testName: "TestNewNeighborRejectsNondecreasingMeasure",
  }),
  Object.freeze({
    id: "allow-reused-evaluation-evidence",
    file: "internal/reduce/engine.go",
    find: "\tif !s.ledger.admit(evaluation) {",
    replace: "\tif false && !s.ledger.admit(evaluation) {",
    package: "./internal/reduce",
    testName: "TestCrossProposalEvidenceReuseIsRecordedAsTerminalRefusal",
  }),
  Object.freeze({
    id: "treat-group-ordinal-as-preservation-identity",
    file: "internal/reduce/model.go",
    find: [
      "\tassessment := compare.AssessPreservation(baseline, observed)",
      "\tif !assessment.Valid() {",
    ].join("\n"),
    replace: [
      "\tassessment := compare.AssessPreservation(baseline, observed)",
      "\tbaselineGroups, observedGroups := baseline.DisplayGroups(), observed.DisplayGroups()",
      "\tsameOrdinalShape := len(baselineGroups) == len(observedGroups)",
      "\tfor index := 0; sameOrdinalShape && index < len(baselineGroups); index++ {",
      "\t\tsameOrdinalShape = len(baselineGroups[index].Members) == len(observedGroups[index].Members)",
      "\t}",
      "\tif sameOrdinalShape {",
      "\t\tevaluation.decision = Preserves",
      "\t\tevaluation.reasonCode = \"GROUP_ORDINAL_SHAPE_MATCH\"",
      "\t\treturn evaluation, nil",
      "\t}",
      "\tif !assessment.Valid() {",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestClassifyNeighborIsTriValuedAndUsesExactMap",
  }),
  Object.freeze({
    id: "replace-declared-rule-priority-with-lexical-order",
    file: "internal/reduce/engine.go",
    find: [
      "\t\tleftPriority, _ := s.input.ReducerSet.rulePriority(l.rule)",
      "\t\trightPriority, _ := s.input.ReducerSet.rulePriority(r.rule)",
      "\t\tif leftPriority != rightPriority {",
      "\t\t\treturn leftPriority < rightPriority",
      "\t\t}",
    ].join("\n"),
    replace: [
      "\t\t// Mutant: lexical spelling overrides the declared reducer-set priority.",
      "\t\tif l.rule.name != r.rule.name {",
      "\t\t\treturn l.rule.name < r.rule.name",
      "\t\t}",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestReducerSetPriorityIsCanonicalAndOutranksLexicalRuleOrder",
  }),
  Object.freeze({
    id: "replace-transform-priority-with-child-digest-order",
    file: "internal/reduce/engine.go",
    find: [
      "\t\tif l.transformPriority != r.transformPriority {",
      "\t\t\treturn l.transformPriority < r.transformPriority",
      "\t\t}",
      "\t\treturn l.stimulusDigest.String() < r.stimulusDigest.String()",
    ].join("\n"),
    replace: [
      "\t\t// Mutant: child digest outranks the declared transform priority.",
      "\t\treturn l.stimulusDigest.String() < r.stimulusDigest.String()",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestReducerTransformPriorityOutranksChildDigest",
  }),
  Object.freeze({
    id: "omit-reducer-set-digest-from-strong-grade",
    file: "internal/reduction/finalize.go",
    find: "\t\tReducerSetDigest:     reducerSet.String(),",
    replace: "\t\tReducerSetDigest:     \"\",",
    package: "./internal/reduction",
    testName: "TestStrongGradeCanonicalIdentityBindsReducerSetDigest",
  }),
]);

export const U5_MUTANTS = Object.freeze(
  REVIEWED_U5_MUTANT_CONTRACT.map((mutant) => Object.freeze({ ...mutant })),
);

const tupleFields = Object.freeze(["file", "find", "id", "package", "replace", "testName"]);

function sameArray(left, right) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

function exactOccurrenceCount(source, anchor) {
  if (typeof anchor !== "string" || anchor.length === 0) return 0;
  return source.split(anchor).length - 1;
}

function codeIs(expected) {
  return (error) => error instanceof MutationGateError && error.code === expected;
}

export function assertU5MutantDefinitionSet(
  mutants = U5_MUTANTS,
  requiredIDs = REQUIRED_U5_MUTANT_IDS,
  reviewedContract = REVIEWED_U5_MUTANT_CONTRACT,
) {
  const requiredCount = 22;
  if (!Object.isFrozen(requiredIDs) || requiredIDs.length !== requiredCount || new Set(requiredIDs).size !== requiredCount) {
    throw new MutationGateError("U5_INVALID_REQUIRED_ID_SET", `U5 requires exactly ${requiredCount} unique frozen IDs`);
  }
  if (!Object.isFrozen(reviewedContract) || reviewedContract.length !== requiredCount ||
      reviewedContract.some((tuple) => !Object.isFrozen(tuple))) {
    throw new MutationGateError("U5_INVALID_REVIEWED_MUTANT_CONTRACT", "U5 reviewed tuples must be recursively frozen");
  }
  if (mutants.length !== requiredCount) {
    throw new MutationGateError("U5_MUTANT_SET_MISMATCH", `defined ${mutants.length} U5 mutants, want ${requiredCount}`);
  }
  const ids = mutants.map((mutant) => mutant.id);
  if (new Set(ids).size !== ids.length || !sameArray(ids, requiredIDs) ||
      !sameArray(reviewedContract.map((tuple) => tuple.id), requiredIDs)) {
    throw new MutationGateError("U5_MUTANT_SET_MISMATCH", "U5 mutant IDs, order, or independent tuple contract changed");
  }
  const sourceMutations = mutants.map((mutant) => `${mutant.file}\0${mutant.find}\0${mutant.replace}`);
  if (new Set(sourceMutations).size !== sourceMutations.length) {
    throw new MutationGateError("U5_DUPLICATE_SOURCE_MUTATION", "U5 requires one distinct source fault injection per mutant ID");
  }
  const allowedFiles = new Set(U5_SANDBOX_FILE_ALLOWLIST);
  const reviewedByID = new Map(reviewedContract.map((tuple) => [tuple.id, tuple]));
  for (const mutant of mutants) {
    const reviewed = reviewedByID.get(mutant.id);
    if (!reviewed || !sameArray(Object.keys(mutant).sort(), tupleFields)) {
      throw new MutationGateError("U5_MUTANT_CONTRACT_MISMATCH", `${mutant.id} differs from the reviewed tuple shape`);
    }
    for (const field of tupleFields) {
      if (mutant[field] !== reviewed[field]) {
        throw new MutationGateError("U5_MUTANT_CONTRACT_MISMATCH", `${mutant.id}.${field} differs from the reviewed tuple`);
      }
    }
    if (!allowedFiles.has(mutant.file)) {
      throw new MutationGateError("U5_MUTANT_FILE_NOT_ALLOWLISTED", `${mutant.id} targets ${mutant.file}`);
    }
    if (typeof mutant.find !== "string" || mutant.find.length === 0 || mutant.find === mutant.replace) {
      throw new MutationGateError("U5_INVALID_MUTATION", `${mutant.id} is not one exact effective replacement`);
    }
    if (!/^\.\/internal\/(?:reduce|reduction)$/u.test(mutant.package)) {
      throw new MutationGateError("U5_INVALID_MUTATION_PACKAGE", `${mutant.id} has invalid package ${mutant.package}`);
    }
    if (!/^Test[A-Za-z0-9_]+$/u.test(mutant.testName)) {
      throw new MutationGateError("U5_INVALID_MUTATION_TEST", `${mutant.id} has invalid named test ${mutant.testName}`);
    }
  }
  if (!Object.isFrozen(mutants) || mutants.some((tuple) => !Object.isFrozen(tuple))) {
    throw new MutationGateError("U5_MUTANT_SET_NOT_IMMUTABLE", "U5 executable tuples must be recursively frozen");
  }
}

function assertSafeRelativePath(value, code = "U5_UNSAFE_PATH") {
  if (typeof value !== "string" || value.length === 0 || isAbsolute(value) ||
      value.split(/[\\/]/u).some((part) => part === "" || part === "." || part === "..")) {
    throw new MutationGateError(code, `unsafe relative path ${JSON.stringify(value)}`);
  }
}

function slash(value) {
  return value.split(sep).join("/");
}

function assertContained(root, candidate, code) {
  const fromRoot = relativePath(root, candidate);
  if (fromRoot === ".." || fromRoot.startsWith(`..${sep}`) || isAbsolute(fromRoot)) {
    throw new MutationGateError(code, `${candidate} escapes ${root}`);
  }
}

async function enumerateU5SourceScope(sourceRoot, scope, reviewedNonGo) {
  assertSafeRelativePath(scope, "U5_UNSAFE_SCOPE_PATH");
  const absoluteRoot = resolve(sourceRoot, scope);
  assertContained(sourceRoot, absoluteRoot, "U5_MANIFEST_SCOPE_ESCAPE");
  let rootMetadata;
  try {
    rootMetadata = await lstat(absoluteRoot);
  } catch (error) {
    throw new MutationGateError("U5_MANIFEST_SCOPE_MISSING", `${scope}: ${error.code ?? error.message}`);
  }
  if (rootMetadata.isSymbolicLink() || !rootMetadata.isDirectory()) {
    throw new MutationGateError("U5_MANIFEST_SCOPE_NONREGULAR", `${scope} is not one real directory`);
  }

  const files = [];
  async function visit(directory) {
    let entries;
    try {
      entries = await readdir(directory, { withFileTypes: true });
    } catch (error) {
      throw new MutationGateError(
        "U5_MANIFEST_ENUMERATION_FAILED",
        `${slash(relativePath(sourceRoot, directory))}: ${error.code ?? error.message}`,
      );
    }
    entries.sort((left, right) => left.name.localeCompare(right.name, "en"));
    for (const entry of entries) {
      const absolute = join(directory, entry.name);
      assertContained(sourceRoot, absolute, "U5_MANIFEST_TREE_ESCAPE");
      const relative = slash(relativePath(sourceRoot, absolute));
      let metadata;
      try {
        metadata = await lstat(absolute);
      } catch (error) {
        throw new MutationGateError("U5_MANIFEST_LSTAT_FAILED", `${relative}: ${error.code ?? error.message}`);
      }
      if (metadata.isSymbolicLink()) {
        throw new MutationGateError("U5_MANIFEST_TREE_SYMLINK", `${relative} is a symbolic link`);
      }
      if (metadata.isDirectory()) {
        await visit(absolute);
      } else if (!metadata.isFile()) {
        throw new MutationGateError("U5_MANIFEST_TREE_NONREGULAR", `${relative} is not a regular file`);
      } else if (!relative.endsWith(".go") && !reviewedNonGo.has(relative)) {
        throw new MutationGateError("U5_MANIFEST_TREE_UNREVIEWED_NON_GO", `${relative} is not an admitted source asset`);
      } else {
        files.push(relative);
      }
    }
  }
  await visit(absoluteRoot);
  return files;
}

export async function assertU5ManifestMatchesTree(
  sourceRoot = repoRoot,
  allowlist = U5_SANDBOX_FILE_ALLOWLIST,
  scopeRoots = U5_SOURCE_SCOPE_ROOTS,
  reviewedNonGoFiles = U5_REVIEWED_NON_GO_FILES,
  embedBindings = U5_REVIEWED_EMBED_BINDINGS,
) {
  const resolvedSource = resolve(sourceRoot);
  let sourceMetadata;
  try {
    sourceMetadata = await lstat(resolvedSource);
  } catch (error) {
    throw new MutationGateError("U5_MANIFEST_ROOT_MISSING", `${resolvedSource}: ${error.code ?? error.message}`);
  }
  if (sourceMetadata.isSymbolicLink() || !sourceMetadata.isDirectory()) {
    throw new MutationGateError("U5_MANIFEST_ROOT_INVALID", `${resolvedSource} is not one real source directory`);
  }
  if (allowlist.length === 0 || new Set(allowlist).size !== allowlist.length) {
    throw new MutationGateError("U5_INVALID_FILE_ALLOWLIST", "U5 file allowlist must be nonempty and unique");
  }
  if (scopeRoots.length === 0 || new Set(scopeRoots).size !== scopeRoots.length) {
    throw new MutationGateError("U5_INVALID_SOURCE_SCOPES", "U5 source scopes must be nonempty and unique");
  }
  if (new Set(reviewedNonGoFiles).size !== reviewedNonGoFiles.length) {
    throw new MutationGateError("U5_INVALID_NON_GO_ALLOWLIST", "U5 reviewed non-Go paths must be unique");
  }
  for (const value of allowlist) assertSafeRelativePath(value, "U5_UNSAFE_ALLOWLIST_PATH");
  for (const value of scopeRoots) assertSafeRelativePath(value, "U5_UNSAFE_SCOPE_PATH");
  for (const value of reviewedNonGoFiles) assertSafeRelativePath(value, "U5_UNSAFE_NON_GO_PATH");

  const reviewedNonGo = new Set(reviewedNonGoFiles);
  const declaredNonGo = allowlist.filter((file) => file !== "go.mod" && !file.endsWith(".go")).sort();
  if (!sameArray(declaredNonGo, [...reviewedNonGoFiles].sort())) {
    throw new MutationGateError(
      "U5_NON_GO_ALLOWLIST_MISMATCH",
      `declared non-Go files [${declaredNonGo.join(",")}] differ from the reviewed closed set`,
    );
  }
  const isScoped = (file) => scopeRoots.some((scope) => file.startsWith(`${scope}/`));
  const support = allowlist.filter((file) => !isScoped(file)).sort();
  if (!sameArray(support, ["go.mod"])) {
    throw new MutationGateError("U5_MANIFEST_SUPPORT_SET_MISMATCH", `support files [${support.join(",")}] differ from go.mod`);
  }
  for (const file of reviewedNonGoFiles) {
    if (!isScoped(file)) throw new MutationGateError("U5_NON_GO_OUTSIDE_SCOPE", `${file} is outside every source scope`);
  }

  const actual = [];
  for (const scope of scopeRoots) actual.push(...await enumerateU5SourceScope(resolvedSource, scope, reviewedNonGo));
  actual.sort((left, right) => left.localeCompare(right, "en"));
  if (new Set(actual).size !== actual.length) {
    throw new MutationGateError("U5_OVERLAPPING_SOURCE_SCOPES", "U5 source scopes enumerate at least one file twice");
  }
  const expected = allowlist.filter((file) => file !== "go.mod").sort();
  const actualSet = new Set(actual);
  const expectedSet = new Set(expected);
  const unlisted = actual.filter((file) => !expectedSet.has(file));
  const missing = expected.filter((file) => !actualSet.has(file));
  if (unlisted.length > 0 || missing.length > 0) {
    throw new MutationGateError("U5_MANIFEST_TREE_MISMATCH", `unlisted=[${unlisted.join(",")}] missing=[${missing.join(",")}]`);
  }

  if (embedBindings.length !== reviewedNonGoFiles.length ||
      new Set(embedBindings.map((binding) => binding.asset)).size !== embedBindings.length ||
      !sameArray(embedBindings.map((binding) => binding.asset).sort(), [...reviewedNonGoFiles].sort())) {
    throw new MutationGateError("U5_EMBED_BINDING_SET_MISMATCH", "reviewed non-Go files differ from exact go:embed bindings");
  }
  for (const binding of embedBindings) {
    if (!binding || Object.keys(binding).sort().join(",") !== "asset,directive,owner") {
      throw new MutationGateError("U5_EMBED_BINDING_SHAPE", "embed binding has an unreviewed shape");
    }
    assertSafeRelativePath(binding.owner, "U5_UNSAFE_EMBED_OWNER");
    assertSafeRelativePath(binding.asset, "U5_UNSAFE_EMBED_ASSET");
    if (!expectedSet.has(binding.owner) || !expectedSet.has(binding.asset)) {
      throw new MutationGateError("U5_EMBED_BINDING_UNLISTED", `${binding.owner} -> ${binding.asset} is not fully allowlisted`);
    }
    const owner = await readFile(resolve(resolvedSource, binding.owner), "utf8");
    if (exactOccurrenceCount(owner, binding.directive) !== 1) {
      throw new MutationGateError("U5_EMBED_DIRECTIVE_COUNT", `${binding.owner} does not bind ${binding.asset} exactly once`);
    }
  }
  return readAllowlistManifest(resolvedSource, allowlist);
}

function manifestEntriesEqual(left, right) {
  return left.file === right.file && left.bytes === right.bytes && left.sha256 === right.sha256;
}

function requireU5ManifestEqual(actual, expected, code, detail) {
  if (actual.digest !== expected.digest || actual.entries.length !== expected.entries.length ||
      actual.entries.some((entry, index) => !manifestEntriesEqual(entry, expected.entries[index]))) {
    throw new MutationGateError(code, `${detail}: ${actual.digest} != ${expected.digest}`);
  }
}

async function validateU5TreeAndManifest(root, allowlist, scopeRoots, reviewedNonGoFiles, embedBindings) {
  const first = await assertU5ManifestMatchesTree(root, allowlist, scopeRoots, reviewedNonGoFiles, embedBindings);
  const second = await assertU5ManifestMatchesTree(root, allowlist, scopeRoots, reviewedNonGoFiles, embedBindings);
  requireU5ManifestEqual(second, first, "U5_MANIFEST_CHANGED_DURING_DOUBLE_READ", "tree changed between complete manifest reads");
  return second;
}

export async function assertU5PrivateCopy(root, allowlist = U5_SANDBOX_FILE_ALLOWLIST) {
  const resolvedRoot = resolve(root);
  let rootMetadata;
  try {
    rootMetadata = await lstat(resolvedRoot);
  } catch (error) {
    throw new MutationGateError("U5_PRIVATE_ROOT_MISSING", error.code ?? error.message);
  }
  if (rootMetadata.isSymbolicLink() || !rootMetadata.isDirectory() || (rootMetadata.mode & 0o777) !== 0o700) {
    throw new MutationGateError("U5_PRIVATE_ROOT_INVALID", `${resolvedRoot} is not one private real directory`);
  }
  const canonicalRoot = await realpath(resolvedRoot);
  const files = [];
  async function visit(directory) {
    const entries = await readdir(directory, { withFileTypes: true });
    entries.sort((left, right) => left.name.localeCompare(right.name, "en"));
    for (const entry of entries) {
      const absolute = join(directory, entry.name);
      assertContained(resolvedRoot, absolute, "U5_PRIVATE_COPY_ESCAPE");
      const relative = slash(relativePath(resolvedRoot, absolute));
      const metadata = await lstat(absolute);
      if (metadata.isSymbolicLink()) throw new MutationGateError("U5_PRIVATE_COPY_SYMLINK", relative);
      if (metadata.isDirectory()) {
        if ((metadata.mode & 0o777) !== 0o700) throw new MutationGateError("U5_PRIVATE_DIRECTORY_MODE", relative);
        await visit(absolute);
        continue;
      }
      if (!metadata.isFile() || metadata.nlink !== 1) {
        throw new MutationGateError("U5_PRIVATE_COPY_NONREGULAR", relative);
      }
      if ((metadata.mode & 0o777) !== 0o600) throw new MutationGateError("U5_PRIVATE_FILE_MODE", relative);
      const canonical = await realpath(absolute);
      assertContained(canonicalRoot, canonical, "U5_PRIVATE_COPY_REALPATH_ESCAPE");
      files.push(relative);
    }
  }
  await visit(resolvedRoot);
  files.sort();
  const expected = [...allowlist].sort();
  if (!sameArray(files, expected)) {
    throw new MutationGateError("U5_PRIVATE_COPY_FILE_SET", `actual=[${files.join(",")}] expected=[${expected.join(",")}]`);
  }
}

function normalizeSeedOptions({
  sourceRoot = repoRoot,
  allowlist = U5_SANDBOX_FILE_ALLOWLIST,
  scopeRoots = U5_SOURCE_SCOPE_ROOTS,
  reviewedNonGoFiles = U5_REVIEWED_NON_GO_FILES,
  embedBindings = U5_REVIEWED_EMBED_BINDINGS,
} = {}) {
  return Object.freeze({
    sourceRoot: resolve(sourceRoot),
    allowlist: Object.freeze([...allowlist]),
    scopeRoots: Object.freeze([...scopeRoots]),
    reviewedNonGoFiles: Object.freeze([...reviewedNonGoFiles]),
    embedBindings: Object.freeze(embedBindings.map((binding) => Object.freeze({ ...binding }))),
  });
}

export async function createU5SeedSnapshot(options = {}) {
  const contract = normalizeSeedOptions(options);
  const sourceManifest = await validateU5TreeAndManifest(
    contract.sourceRoot,
    contract.allowlist,
    contract.scopeRoots,
    contract.reviewedNonGoFiles,
    contract.embedBindings,
  );
  const parent = await mkdtemp(join(tmpdir(), "countershape-u5-seed-"));
  await chmod(parent, 0o700);
  const root = join(parent, "repo");
  try {
    await copyRegularAllowlist(contract.sourceRoot, root, contract.allowlist);
    await assertU5PrivateCopy(root, contract.allowlist);
    const manifest = await validateU5TreeAndManifest(
      root,
      contract.allowlist,
      contract.scopeRoots,
      contract.reviewedNonGoFiles,
      contract.embedBindings,
    );
    requireU5ManifestEqual(manifest, sourceManifest, "U5_SEED_COPY_MISMATCH", "immutable seed differs from admitted source");
    const sourceAfterCopy = await validateU5TreeAndManifest(
      contract.sourceRoot,
      contract.allowlist,
      contract.scopeRoots,
      contract.reviewedNonGoFiles,
      contract.embedBindings,
    );
    requireU5ManifestEqual(sourceAfterCopy, sourceManifest, "U5_ADMITTED_SOURCE_CHANGED", "source changed while seed was copied");
    const seedAfterSourceRead = await validateU5TreeAndManifest(
      root,
      contract.allowlist,
      contract.scopeRoots,
      contract.reviewedNonGoFiles,
      contract.embedBindings,
    );
    requireU5ManifestEqual(seedAfterSourceRead, manifest, "U5_SEED_SNAPSHOT_CHANGED", "seed changed during source revalidation");
    return Object.freeze({ ...contract, parent, root, sourceManifest, manifest });
  } catch (error) {
    await rm(parent, { recursive: true, force: true });
    throw error;
  }
}

export async function cleanupU5SeedSnapshot(seed) {
  await rm(seed.parent, { recursive: true, force: true });
}

export async function assertU5SourceAndSeedUnchanged(seed) {
  await assertU5PrivateCopy(seed.root, seed.allowlist);
  const sourceManifest = await validateU5TreeAndManifest(
    seed.sourceRoot,
    seed.allowlist,
    seed.scopeRoots,
    seed.reviewedNonGoFiles,
    seed.embedBindings,
  );
  const seedManifest = await validateU5TreeAndManifest(
    seed.root,
    seed.allowlist,
    seed.scopeRoots,
    seed.reviewedNonGoFiles,
    seed.embedBindings,
  );
  requireU5ManifestEqual(sourceManifest, seed.sourceManifest, "U5_ADMITTED_SOURCE_CHANGED", "admitted source changed");
  requireU5ManifestEqual(seedManifest, seed.manifest, "U5_SEED_SNAPSHOT_CHANGED", "immutable seed changed");
}

function manifestEntryMap(manifest) {
  return new Map(manifest.entries.map((entry) => [entry.file, entry]));
}

export async function assertExactU5MutantDelta(sandboxRoot, seed, mutant) {
  await assertU5PrivateCopy(sandboxRoot, seed.allowlist);
  const actual = await validateU5TreeAndManifest(
    sandboxRoot,
    seed.allowlist,
    seed.scopeRoots,
    seed.reviewedNonGoFiles,
    seed.embedBindings,
  );
  const expectedEntries = manifestEntryMap(seed.manifest);
  const actualEntries = manifestEntryMap(actual);
  const changed = [];
  for (const [file, expected] of expectedEntries) {
    const candidate = actualEntries.get(file);
    if (!candidate || !manifestEntriesEqual(candidate, expected)) changed.push(file);
  }
  for (const file of actualEntries.keys()) if (!expectedEntries.has(file)) changed.push(file);
  changed.sort();
  if (changed.length !== 1 || changed[0] !== mutant.file) {
    throw new MutationGateError("U5_MUTANT_DELTA_SCOPE", `${mutant.id}: changed [${changed.join(",")}], want exactly ${mutant.file}`);
  }
  const seedBytes = await readFile(resolve(seed.root, mutant.file));
  const expectedBytes = Buffer.from(applyExactMutation(seedBytes.toString("utf8"), mutant), "utf8");
  const actualBytes = await readFile(resolve(sandboxRoot, mutant.file));
  if (!actualBytes.equals(expectedBytes)) {
    throw new MutationGateError("U5_MUTANT_DELTA_BYTES", `${mutant.id}: target differs from the reviewed exact replacement`);
  }
  return actual;
}

export async function assertReviewedU5Anchors(sourceRoot = repoRoot, mutants = U5_MUTANTS) {
  assertU5MutantDefinitionSet(mutants);
  await assertU5ManifestMatchesTree(sourceRoot);
  for (const mutant of mutants) {
    const source = await readFile(resolve(sourceRoot, mutant.file), "utf8");
    const count = exactOccurrenceCount(source, mutant.find);
    if (count !== 1) {
      throw new MutationGateError("U5_MUTATION_ANCHOR_COUNT", `${mutant.id}: exact anchor count was ${count}, want 1`);
    }
    const mutated = applyExactMutation(source, mutant);
    if (mutated === source || exactOccurrenceCount(mutated, mutant.find) !== 0) {
      throw new MutationGateError("U5_MUTATION_REPLACEMENT_INVALID", `${mutant.id}: replacement is not exact and singular`);
    }
  }
}

export async function assertReviewedU5NamedTests(sourceRoot = repoRoot, mutants = U5_MUTANTS) {
  assertU5MutantDefinitionSet(mutants);
  await assertU5ManifestMatchesTree(sourceRoot);
  for (const mutant of mutants) {
    const packageRoot = mutant.package.slice(2);
    const testFiles = U5_SANDBOX_FILE_ALLOWLIST.filter(
      (file) => file.startsWith(`${packageRoot}/`) && file.endsWith("_test.go"),
    );
    const pattern = new RegExp(`\\bfunc\\s+${mutant.testName}\\s*\\(`, "gu");
    let definitions = 0;
    for (const file of testFiles) {
      const source = await readFile(resolve(sourceRoot, file), "utf8");
      definitions += source.match(pattern)?.length ?? 0;
    }
    if (definitions !== 1) {
      throw new MutationGateError(
        "U5_NAMED_TEST_DEFINITION_COUNT",
        `${mutant.id}: ${mutant.package} ${mutant.testName} has ${definitions} admitted definitions, want 1`,
      );
    }
  }
}

function fakeClassification(outcome) {
  return { classification: { outcome, detail: outcome }, output: "" };
}

async function prepareU5CacheRoot(cacheRoot) {
  await mkdir(cacheRoot, { mode: 0o700 });
  for (const name of ["home", "tmp", "go-tmp", "go-path", "go-build", "go-mod"]) {
    const directory = join(cacheRoot, name);
    await mkdir(directory, { recursive: true, mode: 0o700 });
    await chmod(directory, 0o700);
  }
}

function revalidateU5Toolchain(toolchain) {
  revalidateAdmittedGoExecutable(toolchain.executable);
  revalidateAdmittedDarwinCCompiler(toolchain.compiler);
}

async function prepareU5Experiment({ mutant, toolchain, phase, seed, phaseRecords }) {
  await assertU5SourceAndSeedUnchanged(seed);
  const sandboxParent = await mkdtemp(join(tmpdir(), `countershape-u5-${phase}-`));
  await chmod(sandboxParent, 0o700);
  const sandbox = join(sandboxParent, "repo");
  const cacheRoot = join(sandboxParent, "cache");
  try {
    await copyRegularAllowlist(seed.root, sandbox, seed.allowlist);
    await assertU5PrivateCopy(sandbox, seed.allowlist);
    const sourceManifest = await validateU5TreeAndManifest(
      sandbox,
      seed.allowlist,
      seed.scopeRoots,
      seed.reviewedNonGoFiles,
      seed.embedBindings,
    );
    requireU5ManifestEqual(
      sourceManifest,
      seed.manifest,
      "U5_EXPERIMENT_SEED_MISMATCH",
      `${mutant.id} ${phase} copy differs from the immutable seed`,
    );
    await prepareU5CacheRoot(cacheRoot);
    revalidateU5Toolchain(toolchain);
    const environment = minimalU2GoEnvironment(cacheRoot, toolchain);
    revalidateU5Toolchain(toolchain);
    return {
      sandboxParent,
      sandbox,
      cacheRoot,
      environment,
      toolchain,
      phase,
      sourceManifest,
      u5Seed: seed,
      phaseRecords,
    };
  } catch (error) {
    await rm(sandboxParent, { recursive: true, force: true });
    throw error;
  }
}

async function cleanupU5Experiment(context) {
  await rm(context.sandboxParent, { recursive: true, force: true });
}

async function expectedU5ExperimentManifest(context, mutant) {
  if (context.phase === "mutant") {
    return assertExactU5MutantDelta(context.sandbox, context.u5Seed, mutant);
  }
  await assertU5PrivateCopy(context.sandbox, context.u5Seed.allowlist);
  const manifest = await validateU5TreeAndManifest(
    context.sandbox,
    context.u5Seed.allowlist,
    context.u5Seed.scopeRoots,
    context.u5Seed.reviewedNonGoFiles,
    context.u5Seed.embedBindings,
  );
  requireU5ManifestEqual(
    manifest,
    context.u5Seed.manifest,
    "U5_CONTROL_SOURCE_CHANGED",
    `${mutant.id} ${context.phase} source differs from the immutable seed`,
  );
  return manifest;
}

async function runNamedU5GoTest(context, mutant, toolchain) {
  await assertU5SourceAndSeedUnchanged(context.u5Seed);
  const before = await expectedU5ExperimentManifest(context, mutant);
  revalidateU5Toolchain(toolchain);
  let result;
  try {
    result = runNamedU2GoTest({
      sandbox: context.sandbox,
      mutant,
      environment: context.environment,
      toolchain,
    });
  } finally {
    revalidateU5Toolchain(toolchain);
  }
  const after = await expectedU5ExperimentManifest(context, mutant);
  requireU5ManifestEqual(
    after,
    before,
    "U5_TEST_CHANGED_SOURCE",
    `${mutant.id} ${context.phase} named test changed admitted source`,
  );
  await assertU5SourceAndSeedUnchanged(context.u5Seed);
  context.phaseRecords.push(Object.freeze({
    phase: context.phase,
    manifest: before.digest,
    sandbox: context.sandbox,
  }));
  return result;
}

export function exactU5ABADigest(mutant, phaseRecords, seed) {
  const reviewed = REVIEWED_U5_MUTANT_CONTRACT.find((tuple) => tuple.id === mutant?.id);
  if (!Object.isFrozen(mutant) || !reviewed || tupleFields.some((field) => mutant[field] !== reviewed[field])) {
    throw new MutationGateError("U5_ABA_RECEIPT_INVALID", `${mutant?.id ?? "unknown"} is not one frozen reviewed U5 tuple`);
  }
  if (!Array.isArray(phaseRecords)) {
    throw new MutationGateError("U5_ABA_RECEIPT_INVALID", `${mutant?.id ?? "unknown"} lacks an A/B/A record array`);
  }
  if (phaseRecords.some((record) => !record || !sameArray(Object.keys(record).sort(), ["manifest", "phase", "sandbox"]))) {
    throw new MutationGateError("U5_ABA_RECEIPT_INVALID", `${mutant?.id ?? "unknown"} has an unreviewed A/B/A record shape`);
  }
  const phases = phaseRecords.map((record) => record.phase);
  const manifests = phaseRecords.map((record) => record.manifest);
  const sandboxes = phaseRecords.map((record) => record.sandbox);
  const validDigest = (value) => /^sha256:[0-9a-f]{64}$/u.test(value);
  if (
    !mutant || typeof mutant.id !== "string" ||
    !seed?.manifest || !validDigest(seed.manifest.digest) ||
    phaseRecords.length !== 3 || !sameArray(phases, ["baseline", "mutant", "post-control"]) ||
    manifests.some((digest) => !validDigest(digest)) ||
    sandboxes.some((sandbox) => typeof sandbox !== "string" || !isAbsolute(sandbox)) ||
    new Set(sandboxes).size !== 3 ||
    manifests[0] !== seed.manifest.digest || manifests[2] !== seed.manifest.digest ||
    manifests[1] === seed.manifest.digest
  ) {
    throw new MutationGateError("U5_ABA_RECEIPT_INVALID", `${mutant?.id ?? "unknown"} lacks exact fresh A/B/A closure facts`);
  }
  const canonical = JSON.stringify({
    schema_version: "countershape.u5.mutation-receipt/v1",
    mutant_tuple: {
      file: mutant.file,
      find: mutant.find,
      id: mutant.id,
      package: mutant.package,
      replace: mutant.replace,
      test_name: mutant.testName,
    },
    phases: phaseRecords,
    seed_manifest: seed.manifest.digest,
  });
  return `sha256:${createHash("sha256").update(canonical, "utf8").digest("hex")}`;
}

async function withU5TemporaryRoot(prefix, callback) {
  const root = await mkdtemp(join(tmpdir(), prefix));
  await chmod(root, 0o700);
  try {
    return await callback(root);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
}

async function writeSyntheticU5Source(root) {
  const allowlist = Object.freeze([
    "go.mod",
    "pkg/fixture.go",
    "pkg/fixture.mjs",
    "pkg/guard.go",
  ]);
  const scopeRoots = Object.freeze(["pkg"]);
  const reviewedNonGoFiles = Object.freeze(["pkg/fixture.mjs"]);
  const embedBindings = Object.freeze([
    Object.freeze({
      owner: "pkg/fixture.go",
      asset: "pkg/fixture.mjs",
      directive: "//go:embed fixture.mjs",
    }),
  ]);
  await mkdir(join(root, "pkg"), { recursive: true });
  await writeFile(join(root, "go.mod"), "module example.invalid/u5\n", "utf8");
  await writeFile(
    join(root, "pkg", "fixture.go"),
    "package fixture\n\nimport _ \"embed\"\n\n//go:embed fixture.mjs\nvar fixture string\n",
    "utf8",
  );
  await writeFile(join(root, "pkg", "fixture.mjs"), "export const fixture = true;\n", "utf8");
  await writeFile(join(root, "pkg", "guard.go"), "package fixture\nconst guarded = true // SYNTHETIC_U5_ANCHOR\n", "utf8");
  return Object.freeze({ allowlist, scopeRoots, reviewedNonGoFiles, embedBindings });
}

async function exerciseHostileU5ManifestTripwires() {
  await withU5TemporaryRoot("countershape-u5-hostile-", async (root) => {
    const contract = await writeSyntheticU5Source(root);
    const inspect = () => assertU5ManifestMatchesTree(
      root,
      contract.allowlist,
      contract.scopeRoots,
      contract.reviewedNonGoFiles,
      contract.embedBindings,
    );
    await assert.doesNotReject(inspect());

    const unlistedGo = join(root, "pkg", "unlisted.go");
    await writeFile(unlistedGo, "package fixture\n", "utf8");
    await assert.rejects(inspect(), codeIs("U5_MANIFEST_TREE_MISMATCH"));
    await unlink(unlistedGo);

    const unreviewedAsset = join(root, "pkg", "secret.txt");
    await writeFile(unreviewedAsset, "must not enter the execution closure\n", "utf8");
    await assert.rejects(inspect(), codeIs("U5_MANIFEST_TREE_UNREVIEWED_NON_GO"));
    await unlink(unreviewedAsset);

    const linkedGo = join(root, "pkg", "linked.go");
    await symlink(join(root, "pkg", "guard.go"), linkedGo);
    await assert.rejects(inspect(), codeIs("U5_MANIFEST_TREE_SYMLINK"));
    await unlink(linkedGo);

    await assert.rejects(
      assertU5ManifestMatchesTree(
        root,
        [...contract.allowlist, "../escape.go"],
        contract.scopeRoots,
        contract.reviewedNonGoFiles,
        contract.embedBindings,
      ),
      codeIs("U5_UNSAFE_ALLOWLIST_PATH"),
    );

    await mkdir(join(root, "pkg", "nested"));
    await writeFile(join(root, "pkg", "nested", "nested.go"), "package nested\n", "utf8");
    await assert.rejects(
      assertU5ManifestMatchesTree(
        root,
        [...contract.allowlist, "pkg/nested/nested.go"],
        ["pkg", "pkg/nested"],
        contract.reviewedNonGoFiles,
        contract.embedBindings,
      ),
      codeIs("U5_OVERLAPPING_SOURCE_SCOPES"),
    );

    const linkedRoot = join(dirname(root), `${root.split(sep).at(-1)}-linked-root`);
    await symlink(root, linkedRoot);
    try {
      await assert.rejects(
        assertU5ManifestMatchesTree(
          linkedRoot,
          [...contract.allowlist, "pkg/nested/nested.go"],
          contract.scopeRoots,
          contract.reviewedNonGoFiles,
          contract.embedBindings,
        ),
        codeIs("U5_MANIFEST_ROOT_INVALID"),
      );
    } finally {
      await unlink(linkedRoot);
    }
  });
}

async function exerciseU5PrivateSeedAndDeltaTripwires() {
  await withU5TemporaryRoot("countershape-u5-seed-selftest-", async (root) => {
    const source = join(root, "source");
    await mkdir(source, { mode: 0o700 });
    const contract = await writeSyntheticU5Source(source);
    const seed = await createU5SeedSnapshot({ sourceRoot: source, ...contract });
    try {
      await assertU5SourceAndSeedUnchanged(seed);
      const sourceGuard = join(source, "pkg", "guard.go");
      const originalSource = await readFile(sourceGuard);
      await writeFile(sourceGuard, "package fixture\nconst guarded = false // SYNTHETIC_U5_ANCHOR\n", "utf8");
      await assert.rejects(assertU5SourceAndSeedUnchanged(seed), codeIs("U5_ADMITTED_SOURCE_CHANGED"));
      await writeFile(sourceGuard, originalSource);
      await assertU5SourceAndSeedUnchanged(seed);

      const seedGuard = join(seed.root, "pkg", "guard.go");
      const originalSeed = await readFile(seedGuard);
      await writeFile(seedGuard, "package fixture\nconst guarded = false // SYNTHETIC_U5_ANCHOR\n", "utf8");
      await assert.rejects(assertU5SourceAndSeedUnchanged(seed), codeIs("U5_SEED_SNAPSHOT_CHANGED"));
      await writeFile(seedGuard, originalSeed);
      await assertU5SourceAndSeedUnchanged(seed);

      const sandbox = join(root, "sandbox");
      await copyRegularAllowlist(seed.root, sandbox, seed.allowlist);
      const mutant = {
        id: "synthetic-u5-mutant",
        file: "pkg/guard.go",
        find: "const guarded = true // SYNTHETIC_U5_ANCHOR",
        replace: "const guarded = false // SYNTHETIC_U5_ANCHOR",
      };
      await mutateRegularFileNoFollow(sandbox, mutant);
      await assertExactU5MutantDelta(sandbox, seed, mutant);
      await writeFile(join(sandbox, "go.mod"), "module changed.invalid/u5\n", "utf8");
      await assert.rejects(assertExactU5MutantDelta(sandbox, seed, mutant), codeIs("U5_MUTANT_DELTA_SCOPE"));
    } finally {
      await cleanupU5SeedSnapshot(seed);
    }
  });
}

async function exerciseSyntheticU5ABA() {
  const phases = [];
  const phaseRecords = [];
  let invocation = 0;
  const seedDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
  const mutantDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
  const seed = { manifest: { digest: seedDigest } };
  const result = await runU2Mutant(U5_MUTANTS[0], {}, {
    async prepareExperiment({ phase }) {
      phases.push(phase);
      return { phase, sandbox: `/fresh/u5/${phase}`, sandboxParent: `/fresh/u5/${phase}` };
    },
    async cleanupExperiment() {},
    async applyMutation() {},
    async runNamedTest(context) {
      phaseRecords.push(Object.freeze({
        phase: context.phase,
        manifest: context.phase === "mutant" ? mutantDigest : seedDigest,
        sandbox: context.sandbox,
      }));
      const outcomes = [NamedTestOutcome.Pass, NamedTestOutcome.Failure, NamedTestOutcome.Pass];
      return fakeClassification(outcomes[invocation++]);
    },
  });
  assert.match(result, /^KILLED replace-full-map-preservation-with-partition-shape/u);
  assert.deepEqual(phases, ["baseline", "mutant", "post-control"]);
  assert.equal(invocation, 3);
  assert.match(exactU5ABADigest(U5_MUTANTS[0], phaseRecords, seed), /^sha256:[0-9a-f]{64}$/u);
  assert.throws(
    () => exactU5ABADigest(
      U5_MUTANTS[0],
      phaseRecords.map((record) => ({ ...record, sandbox: "/fresh/u5/reused" })),
      seed,
    ),
    codeIs("U5_ABA_RECEIPT_INVALID"),
  );
}

export async function selfTest() {
  assertU5MutantDefinitionSet();
  const selfManifest = await validateU5TreeAndManifest(
    repoRoot,
    U5_SANDBOX_FILE_ALLOWLIST,
    U5_SOURCE_SCOPE_ROOTS,
    U5_REVIEWED_NON_GO_FILES,
    U5_REVIEWED_EMBED_BINDINGS,
  );
  for (const file of U5_REVIEWED_NON_GO_FILES) {
    const entry = selfManifest.entries.find((candidate) => candidate.file === file);
    assert.ok(entry && entry.bytes > 0 && /^[0-9a-f]{64}$/u.test(entry.sha256), `${file} lacks exact manifest bytes`);
  }
  for (const forbidden of [".env", ".git/config", ".didrun/claims.json", "credentials.json", "CLAUDE.md"]) {
    assert.equal(U5_SANDBOX_FILE_ALLOWLIST.includes(forbidden), false, `${forbidden} entered the U5 execution closure`);
  }
  assert.equal(U5_SANDBOX_FILE_ALLOWLIST.some((file) => file.startsWith(".git/") || file.startsWith(".didrun/")), false);
  await assertReviewedU5Anchors(repoRoot);
  await assertReviewedU5NamedTests(repoRoot);

  assert.throws(() => assertU5MutantDefinitionSet(U5_MUTANTS.slice(0, -1)), codeIs("U5_MUTANT_SET_MISMATCH"));
  const reordered = [...U5_MUTANTS];
  [reordered[0], reordered[1]] = [reordered[1], reordered[0]];
  assert.throws(() => assertU5MutantDefinitionSet(reordered), codeIs("U5_MUTANT_SET_MISMATCH"));
  const tampered = U5_MUTANTS.map((mutant) => ({ ...mutant }));
  tampered[0].replace += "\n// unreviewed weakening";
  assert.throws(() => assertU5MutantDefinitionSet(tampered), codeIs("U5_MUTANT_CONTRACT_MISMATCH"));
  const extraField = U5_MUTANTS.map((mutant) => ({ ...mutant }));
  extraField[0].unreviewed = true;
  assert.throws(() => assertU5MutantDefinitionSet(extraField), codeIs("U5_MUTANT_CONTRACT_MISMATCH"));
  const mutableExactClone = U5_MUTANTS.map((mutant) => ({ ...mutant }));
  assert.throws(() => assertU5MutantDefinitionSet(mutableExactClone), codeIs("U5_MUTANT_SET_NOT_IMMUTABLE"));

  await exerciseHostileU5ManifestTripwires();
  await exerciseU5PrivateSeedAndDeltaTripwires();
  await exerciseSyntheticU5ABA();
  return `U5 mutation self-test passed: ${U5_MUTANTS.length}/${U5_MUTANTS.length} frozen tuples; 1/1 deterministic synthetic A/B/A runner proof; hostile manifest, path, seed, and delta tripwires closed.`;
}

export async function main(arguments_ = process.argv.slice(2)) {
  if (arguments_.length === 1 && arguments_[0] === "--self-test") {
    process.stdout.write(`${await selfTest()}\n`);
    return;
  }
  if (arguments_.length !== 0) {
    throw new MutationGateError("U5_UNSUPPORTED_ARGUMENT", "usage: node tools/mutate-u5.mjs [--self-test]");
  }

  assertU5MutantDefinitionSet();
  await validateU5TreeAndManifest(
    repoRoot,
    U5_SANDBOX_FILE_ALLOWLIST,
    U5_SOURCE_SCOPE_ROOTS,
    U5_REVIEWED_NON_GO_FILES,
    U5_REVIEWED_EMBED_BINDINGS,
  );
  await assertReviewedU5Anchors(repoRoot);
  await assertReviewedU5NamedTests(repoRoot);
  const toolchain = await admitU2DarwinToolchain();
  const seed = await createU5SeedSnapshot({ sourceRoot: repoRoot });
  const results = [];
  try {
    for (const mutant of U5_MUTANTS) {
      const phaseRecords = [];
      const result = await runU2Mutant(mutant, toolchain, {
        prepareExperiment({ mutant: currentMutant, toolchain: currentToolchain, phase }) {
          return prepareU5Experiment({
            mutant: currentMutant,
            toolchain: currentToolchain,
            phase,
            seed,
            phaseRecords,
          });
        },
        cleanupExperiment: cleanupU5Experiment,
        applyMutation: mutateRegularFileNoFollow,
        runNamedTest(context, currentMutant) {
          return runNamedU5GoTest(context, currentMutant, toolchain);
        },
      });
      const receipt = exactU5ABADigest(mutant, phaseRecords, seed);
      results.push(Object.freeze({ mutant: mutant.id, result, receipt }));
    }
    await assertU5SourceAndSeedUnchanged(seed);
  } finally {
    await cleanupU5SeedSnapshot(seed);
  }
  revalidateU5Toolchain(toolchain);

  process.stdout.write(`Trusted Go realpath: ${toolchain.executable.path}\n`);
  process.stdout.write(`Trusted Go sha256: ${toolchain.executable.sha256}\n`);
  process.stdout.write(`Trusted Go version: ${toolchain.version.line}\n`);
  process.stdout.write(`Trusted C compiler realpath: ${toolchain.compiler.path}\n`);
  process.stdout.write(`Trusted C compiler sha256: ${toolchain.compiler.sha256}\n`);
  process.stdout.write(`Closed Darwin CGO facts digest: ${toolchain.cgo.digest}\n`);
  process.stdout.write(`Immutable U5 seed manifest: ${seed.manifest.digest}\n`);
  for (const file of U5_REVIEWED_NON_GO_FILES) {
    const entry = seed.manifest.entries.find((candidate) => candidate.file === file);
    if (!entry) throw new MutationGateError("U5_NON_GO_MANIFEST_ENTRY_MISSING", `${file} is absent from the final seed manifest`);
    process.stdout.write(`Reviewed U5 non-Go source: ${entry.file} bytes=${entry.bytes} sha256=${entry.sha256}\n`);
  }
  process.stdout.write("Containment notice: private mutation copies retain the invoking user's host filesystem and network authority.\n");
  for (const result of results) {
    process.stdout.write(`${result.result}; exact fresh A/B/A closure digest ${result.receipt}\n`);
  }
  process.stdout.write(
    `U5 mutation gate: ${results.length}/${REQUIRED_U5_MUTANT_IDS.length} required mutants killed; ` +
    `${results.length}/${REQUIRED_U5_MUTANT_IDS.length} fresh A/B/A receipts.\n`,
  );
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(modulePath)) {
  main().catch((error) => {
    process.stderr.write(`${error.stack ?? error}\n`);
    process.exitCode = 1;
  });
}
