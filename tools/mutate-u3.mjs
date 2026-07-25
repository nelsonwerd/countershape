#!/usr/bin/env node

import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import {
  MutationGateError,
  NamedTestOutcome,
  applyExactMutation,
  revalidateAdmittedGoExecutable,
} from "./mutate-u1.mjs";
import {
  admitU2DarwinToolchain,
  assertU2ManifestMatchesTree,
  assertU2SourceAndSeedUnchanged,
  cleanupU2SeedSnapshot,
  createU2SeedSnapshot,
  revalidateAdmittedDarwinCCompiler,
  runU2Mutant,
} from "./mutate-u2.mjs";

const modulePath = fileURLToPath(import.meta.url);
const repoRoot = resolve(dirname(modulePath), "..");

// This is a positive execution manifest, not a denylist. Mutation experiments
// copy only the exact transitive Go source/test closure required by the five
// named U3 package targets. A new file anywhere in a reviewed scope closes the
// gate until this list is deliberately updated. Private copies still retain
// the invoking user's host filesystem and network authority.
export const U3_SANDBOX_FILE_ALLOWLIST = Object.freeze([
  "go.mod",
  "internal/adapters/cli/bridge.go",
  "internal/adapters/cli/capture.go",
  "internal/adapters/cli/capture_test.go",
  "internal/adapters/cli/errors.go",
  "internal/adapters/cli/model/binding.go",
  "internal/adapters/cli/model/capture_policy.go",
  "internal/adapters/cli/model/fixture_recipe.go",
  "internal/adapters/cli/model/measure.go",
  "internal/adapters/cli/model/projection_authority.go",
  "internal/adapters/cli/model/projection_schema.go",
  "internal/adapters/cli/model/projection_schema_test.go",
  "internal/adapters/cli/model/stimulus.go",
  "internal/adapters/cli/model/stimulus_test.go",
  "internal/adapters/cli/neighbors.go",
  "internal/adapters/cli/neighbors_test.go",
  "internal/adapters/cli/projection.go",
  "internal/adapters/cli/projection_test.go",
  "internal/adapters/cli/stimulus.go",
  "internal/adapters/http/model/binding.go",
  "internal/adapters/http/model/capture_policy.go",
  "internal/adapters/http/model/digest.go",
  "internal/adapters/http/model/errors.go",
  "internal/adapters/http/model/measure.go",
  "internal/adapters/http/model/projection_authority.go",
  "internal/adapters/http/model/projection_schema.go",
  "internal/adapters/http/model/projection_schema_test.go",
  "internal/adapters/http/model/start.go",
  "internal/adapters/http/model/start_test.go",
  "internal/adapters/http/model/portable_start_test.go",
  "internal/adapters/http/model/stimulus.go",
  "internal/adapters/http/model/wire.go",
  "internal/adapters/http/model/wire_test.go",
  "internal/canon/canon_test.go",
  "internal/canon/digest.go",
  "internal/canon/doc.go",
  "internal/canon/encode.go",
  "internal/canon/error.go",
  "internal/canon/fuzz_test.go",
  "internal/canon/limits.go",
  "internal/canon/limits_test.go",
  "internal/canon/parse.go",
  "internal/canon/scan.go",
  "internal/canon/token.go",
  "internal/canon/typed.go",
  "internal/canon/value.go",
  "internal/compare/outcome_map.go",
  "internal/compare/outcome_map_test.go",
  "internal/compare/wire.go",
  "internal/domain/domain_test.go",
  "internal/domain/envelope.go",
  "internal/domain/errors.go",
  "internal/domain/identity.go",
  "internal/domain/projection.go",
  "internal/domain/receipt.go",
  "internal/domain/transitions.go",
  "internal/domain/world.go",
  "internal/gitobj/errors.go",
  "internal/gitobj/filesystem.go",
  "internal/gitobj/filesystem_darwin.go",
  "internal/gitobj/filesystem_unsupported.go",
  "internal/gitobj/fuzz_test.go",
  "internal/gitobj/git.go",
  "internal/gitobj/helpers_external_test.go",
  "internal/gitobj/identity_external_test.go",
  "internal/gitobj/inspect.go",
  "internal/gitobj/inspect_test.go",
  "internal/gitobj/materialize.go",
  "internal/gitobj/materialize_external_test.go",
  "internal/gitobj/mutation_contract_test.go",
  "internal/gitobj/publish_darwin.go",
  "internal/gitobj/publish_darwin_test.go",
  "internal/gitobj/publish_unsupported.go",
  "internal/gitobj/refusal_external_test.go",
  "internal/gitobj/single_target.go",
  "internal/gitobj/single_target_test.go",
  "internal/gitobj/types.go",
  "internal/gitobj/validate.go",
  "internal/observe/batch.go",
  "internal/observe/batch_test.go",
  "internal/observe/eligibility.go",
  "internal/observe/eligibility_test.go",
  "internal/observe/eligibilitycore/eligibility.go",
  "internal/observe/executed_link.go",
  "internal/observe/identity.go",
  "internal/observe/projection_evidence.go",
  "internal/observe/schedule.go",
  "internal/observe/schedule_test.go",
  "internal/processmechanics/capture.go",
  "internal/processmechanics/capture_darwin_test.go",
  "internal/processmechanics/process.go",
  "internal/processmechanics/process_darwin.go",
  "internal/processmechanics/process_darwin_test.go",
  "internal/processmechanics/process_unsupported.go",
  "internal/world/allocate.go",
  "internal/world/api.go",
  "internal/world/capture.go",
  "internal/world/cli_api.go",
  "internal/world/cli_bridge_darwin_test.go",
  "internal/world/cli_fixture.go",
  "internal/world/cli_invocation.go",
  "internal/world/errors.go",
  "internal/world/execute_darwin_test.go",
  "internal/world/finalize.go",
  "internal/world/fresh_confirmation.go",
  "internal/world/http_api.go",
  "internal/world/http_receipt.go",
  "internal/world/http_seed.go",
  "internal/world/http_service.go",
  "internal/world/http_service_darwin.go",
  "internal/world/http_service_darwin_test.go",
  "internal/world/http_portable_negative_darwin_test.go",
  "internal/world/http_service_unsupported.go",
  "internal/world/process.go",
  "internal/world/process_darwin.go",
  "internal/world/process_darwin_test.go",
  "internal/world/process_mutation_darwin_test.go",
  "internal/world/process_unsupported.go",
  "internal/world/receipt.go",
  "internal/world/tools.go",
  "testkit/gitrepo/gitrepo.go",
  "testkit/processfixture/main.go",
]);

export const U3_SOURCE_SCOPE_ROOTS = Object.freeze([
  "internal/adapters/cli",
  "internal/adapters/http/model",
  "internal/canon",
  "internal/compare",
  "internal/domain",
  "internal/gitobj",
  "internal/observe",
  "internal/processmechanics",
  "internal/world",
  "testkit/gitrepo",
  "testkit/processfixture",
]);

// IDs and tuples are independent frozen contracts. Deleting, reordering,
// renaming, or weakening one tuple fails before any repository test executes.
export const REQUIRED_U3_MUTANT_IDS = Object.freeze([
  "reverse-stimulus-argv",
  "collapse-absent-stdin-into-present-empty",
  "materialize-absent-environment-as-present-empty",
  "inherit-ambient-environment",
  "allow-runner-owned-environment-namespace",
  "bypass-plan-stimulus-prefix-binding",
  "invert-fixture-reopen-rehash-check",
  "accept-truncated-invocation-argv",
  "treat-nonzero-exit-as-invalid-behavior",
  "promote-output-limit-prefix-to-present-value",
  "prefer-latched-exit-over-timeout",
  "collapse-signal-field-to-missing",
  "swap-stdout-stderr-projection-sources",
  "hide-projection-operations",
  "promote-projection-rejection-to-eligible",
  "allow-prior-attempt-reuse",
  "majority-classify-alternating-candidate",
  "permit-partial-comparison-matrix",
  "include-support-artifact-in-preservation-identity",
  "drop-candidate-key-from-labeled-map-identity",
  "miswire-stderr-capture-and-report-requested-limit",
  "collapse-physical-present-empty-stdin",
  "substitute-same-length-physical-stdin",
  "allow-present-invocation-as-absent",
  "forge-preprocess-physical-entry",
  "allow-cross-paired-projection-rejection",
  "omit-probe-transport-from-arbitration-receipt",
  "bypass-reducible-direct-argv-grammar",
  "bypass-resolved-capture-policy",
  "bypass-resolved-adapter-projection",
  "erase-declared-plan-rotation",
  "bypass-observation-plan-schedule",
  "bypass-observation-wall-admission",
  "allow-reserved-git-fixture",
  "defer-physical-entry-until-start-success",
  "discard-late-overflow-earlier-primary",
  "bypass-environment-aggregate-limit",
  "trust-fixture-receipt-without-rehash",
  "misclassify-projection-resource-limit",
  "allow-node-inline-script-string",
  "publish-rejection-after-wall-budget",
  "forge-physical-entry-before-spawn-attempt",
  "overwrite-earlier-control-with-late-output-overflow",
]);

export const REVIEWED_U3_MUTANT_CONTRACT = Object.freeze([
  Object.freeze({
    id: "reverse-stimulus-argv",
    file: "internal/adapters/cli/model/stimulus.go",
    find: [
      "\t// MUTATION_ANCHOR: preserve-argv-order",
      "\tlogicalArgv = append(logicalArgv, argv...)",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: preserve-argv-order",
      "\tfor index := len(argv) - 1; index >= 0; index-- {",
      "\t\tlogicalArgv = append(logicalArgv, argv[index])",
      "\t}",
    ].join("\n"),
    package: "./internal/adapters/cli/model",
    testName: "TestCLIStimulusPreservesOrderAndTaggedAbsence",
  }),
  Object.freeze({
    id: "collapse-absent-stdin-into-present-empty",
    file: "internal/adapters/cli/model/stimulus.go",
    find: "func AbsentStdin() CLIStdin { return CLIStdin{presence: PresenceAbsent, bytes: []byte{}} }",
    replace: "func AbsentStdin() CLIStdin { return CLIStdin{presence: PresencePresent, bytes: []byte{}} }",
    package: "./internal/adapters/cli/model",
    testName: "TestReducerNeutralMeasurePreservesPresentEmptyUnit",
  }),
  Object.freeze({
    id: "materialize-absent-environment-as-present-empty",
    file: "internal/adapters/cli/model/stimulus.go",
    find: "func (b CLIEnvironmentBinding) Present() bool      { return b.presence == PresencePresent }",
    replace: "func (b CLIEnvironmentBinding) Present() bool      { return true }",
    package: "./internal/world",
    testName: "TestCLIEnvironmentIsSparseScheduledAndCollisionClosed",
  }),
  Object.freeze({
    id: "inherit-ambient-environment",
    file: "internal/world/allocate.go",
    find: "inheritAmbientEnvironment = false // MUTANT_U2_INHERIT_AMBIENT_ENVIRONMENT",
    replace: "inheritAmbientEnvironment = true // MUTANT_U2_INHERIT_AMBIENT_ENVIRONMENT",
    package: "./internal/world",
    testName: "TestBuildEnvironmentDoesNotInheritAmbientVariables",
  }),
  Object.freeze({
    id: "allow-runner-owned-environment-namespace",
    file: "internal/world/cli_api.go",
    find: [
      "\t// MUTATION_ANCHOR: cli-stimulus-must-not-control-runner-environment",
      "\treturn strings.HasPrefix(name, \"XDG_\") || strings.HasPrefix(name, \"COUNTERSHAPE_\") ||",
      "\t\tstrings.HasPrefix(name, \"DYLD_\")",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: cli-stimulus-must-not-control-runner-environment",
      "\treturn strings.HasPrefix(name, \"\\x00\")",
    ].join("\n"),
    package: "./internal/world",
    testName: "TestCLIEnvironmentIsSparseScheduledAndCollisionClosed",
  }),
  Object.freeze({
    id: "bypass-plan-stimulus-prefix-binding",
    file: "internal/adapters/cli/model/binding.go",
    find: "\tif !equalStrings(plan.StartArgv(), stimulus.BaseLogicalArgv()) {",
    replace: "\tif false && !equalStrings(plan.StartArgv(), stimulus.BaseLogicalArgv()) {",
    package: "./internal/adapters/cli/model",
    testName: "TestBindExecutionRequiresExactPlanPrefixAndRetainsPayload",
  }),
  Object.freeze({
    id: "invert-fixture-reopen-rehash-check",
    file: "internal/world/cli_fixture.go",
    find: [
      "\t// MUTATION_ANCHOR: fixture-finalized-by-reopen-and-rehash",
      "\tif actualDigest != expectedDigest || !bytes.Equal(actualBytes, contents) {",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: fixture-finalized-by-reopen-and-rehash",
      "\tif actualDigest == expectedDigest && bytes.Equal(actualBytes, contents) {",
    ].join("\n"),
    package: "./internal/world",
    testName: "TestCLIFixtureOverlayBindsRecipeAndReopensExactBytes",
  }),
  Object.freeze({
    id: "accept-truncated-invocation-argv",
    file: "internal/world/cli_invocation.go",
    find: [
      "\tif !attemptIsString || !argvIsArray || len(argvElements) != len(expectedArgv) {",
      "\t\treturn CLIInvocationSchemaMismatch, parsedDigest, int64(len(bytes)), false, false",
      "\t}",
      "\tattemptMatches := actualAttempt == expectedAttemptID",
      "\targvMatches := true",
      "\tfor index, element := range argvElements {",
      "\t\ttext, stringValue := element.Text()",
      "\t\tif !stringValue || text != expectedArgv[index] {",
      "\t\t\targvMatches = false",
      "\t\t}",
      "\t}",
    ].join("\n"),
    replace: [
      "\tif !attemptIsString || !argvIsArray || len(argvElements) > len(expectedArgv) {",
      "\t\treturn CLIInvocationSchemaMismatch, parsedDigest, int64(len(bytes)), false, false",
      "\t}",
      "\tattemptMatches := actualAttempt == expectedAttemptID",
      "\targvMatches := true",
      "\tfor _, element := range argvElements {",
      "\t\t_, stringValue := element.Text()",
      "\t\tif !stringValue {",
      "\t\t\targvMatches = false",
      "\t\t}",
      "\t}",
    ].join("\n"),
    package: "./internal/world",
    testName: "TestCLIInvocationEvidenceRequiresAttemptAndFullLogicalArgv",
  }),
  Object.freeze({
    id: "treat-nonzero-exit-as-invalid-behavior",
    file: "internal/adapters/cli/capture.go",
    find: [
      "\tif c.kind == CompletionExited {",
      "\t\treturn c.code >= 0 && c.code <= 255 && c.signal == \"\"",
      "\t}",
    ].join("\n"),
    replace: [
      "\tif c.kind == CompletionExited {",
      "\t\treturn c.code == 0 && c.signal == \"\"",
      "\t}",
    ].join("\n"),
    package: "./internal/adapters/cli",
    testName: "TestCLIProjectionNonzeroExitAndStrictJSONAreEligibleBehavior",
  }),
  Object.freeze({
    id: "promote-output-limit-prefix-to-present-value",
    file: "internal/adapters/cli/capture.go",
    find: "\t\treturn CLIChannelCapture{name: name, state: ChannelTruncated, bytes: append([]byte(nil), captured...), retainedDigest: digest, observedBytes: observed}, nil",
    replace: "\t\treturn CLIChannelCapture{name: name, state: ChannelPresent, bytes: append([]byte(nil), captured...), retainedDigest: digest, observedBytes: observed}, nil",
    package: "./internal/adapters/cli",
    testName: "TestCLIChannelCaptureDistinguishesExactCapOverflowAndEmpty",
  }),
  Object.freeze({
    id: "prefer-latched-exit-over-timeout",
    file: "internal/processmechanics/process_darwin.go",
    find: [
      "\tif deadlineObserved {",
      "\t\treturn terminalDecision{primary: ControlTimeout, stdin: stdin}, true, true",
      "\t}",
      "\tif waited != nil {",
      "\t\treturn terminalDecision{waited: waited, stdin: stdin}, true, false",
      "\t}",
    ].join("\n"),
    replace: [
      "\tif waited != nil {",
      "\t\treturn terminalDecision{waited: waited, stdin: stdin}, true, deadlineObserved",
      "\t}",
      "\tif deadlineObserved {",
      "\t\treturn terminalDecision{primary: ControlTimeout, stdin: stdin}, true, true",
      "\t}",
    ].join("\n"),
    package: "./internal/processmechanics",
    testName: "TestExecutionBudgetStartsAtPhysicalStartNotClose",
  }),
  Object.freeze({
    id: "collapse-signal-field-to-missing",
    file: "internal/adapters/cli/projection.go",
    find: [
      "\tcase CLIFieldExitSignal:",
      "\t\tif completion.kind != CompletionSignaled {",
      "\t\t\treturn exactMissing(operation, CLIChannelExit, field)",
      "\t\t}",
      "\t\treturn exactString(completion.signal, operation, CLIChannelExit, field)",
    ].join("\n"),
    replace: [
      "\tcase CLIFieldExitSignal:",
      "\t\treturn exactMissing(operation, CLIChannelExit, field)",
    ].join("\n"),
    package: "./internal/adapters/cli",
    testName: "TestCLIProjectionRetainsMissingPresentEmptyAndSignalDistinctions",
  }),
  Object.freeze({
    id: "swap-stdout-stderr-projection-sources",
    file: "internal/adapters/cli/projection.go",
    find: [
      "\tcase CLIFieldStdoutBytes:",
      "\t\treturn exactBytes(stdout, operation, CLIChannelStdout, field)",
      "\tcase CLIFieldStderrText:",
      "\t\treturn exactString(string(stderr), operation, CLIChannelStderr, field)",
    ].join("\n"),
    replace: [
      "\tcase CLIFieldStdoutBytes:",
      "\t\treturn exactBytes(stderr, operation, CLIChannelStderr, field)",
      "\tcase CLIFieldStderrText:",
      "\t\treturn exactString(string(stdout), operation, CLIChannelStdout, field)",
    ].join("\n"),
    package: "./internal/adapters/cli",
    testName: "TestCLIProjectionKeepsStdoutAndStderrIndependent",
  }),
  Object.freeze({
    id: "hide-projection-operations",
    file: "internal/adapters/cli/projection.go",
    find: [
      "\t\t// MUTATION_ANCHOR: hidden-projection-operation",
      "\t\ttranscript = append(transcript, CLIProjectionTraceEntry{operation: operation, sourceLinks: cloneSourceLinks(links)})",
    ].join("\n"),
    replace: [
      "\t\t// MUTATION_ANCHOR: hidden-projection-operation",
      "\t\t_ = links",
    ].join("\n"),
    package: "./internal/adapters/cli",
    testName: "TestCLIProjectionNonzeroExitAndStrictJSONAreEligibleBehavior",
  }),
  Object.freeze({
    id: "promote-projection-rejection-to-eligible",
    file: "internal/observe/eligibility.go",
    find: [
      "\tif fact.kind == trialControlled {",
      "\t\tdecision, err := eligibilitycore.Select(eligibilitycore.ControlIneligible, fact.controls)",
      "\t\tif err != nil {",
      "\t\t\treturn Eligibility{reasons: []domain.ControlReason{domain.ControlProjectionRejected}}",
      "\t\t}",
      "\t\treturn Eligibility{eligible: decision.IsEligible(), reasons: decision.Reasons()}",
      "\t}",
    ].join("\n"),
    replace: [
      "\tif fact.kind == trialControlled {",
      "\t\treturn Eligibility{eligible: true}",
      "\t}",
    ].join("\n"),
    package: "./internal/observe",
    testName: "TestProjectionRejectionIsPostAttemptControlNeverOutcome",
  }),
  Object.freeze({
    id: "allow-prior-attempt-reuse",
    file: "internal/observe/batch.go",
    find: [
      "\t// MUTATION_ANCHOR: prior-attempt-reuse-must-be-refused",
      "\tif _, reused := seenAttempts[attemptDigest]; reused {",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: prior-attempt-reuse-must-be-refused",
      "\tif _, reused := seenAttempts[attemptDigest]; false && reused {",
    ].join("\n"),
    package: "./internal/observe",
    testName: "TestRunObservationRefusesPriorAttemptReuseAcrossCandidates",
  }),
  Object.freeze({
    id: "majority-classify-alternating-candidate",
    file: "internal/observe/eligibility.go",
    find: [
      "\t} else if len(counts) >= 2 {",
      "\t\t// Once two eligible fingerprints exist, instability is established",
      "\t\t// evidence. A later orchestration stop can prevent stability from being",
      "\t\t// established, but cannot erase an observed disagreement.",
      "\t\tclassification.status = Unstable",
      "\t\tclassification.histogram = sortedHistogram(counts)",
    ].join("\n"),
    replace: [
      "\t} else if len(counts) >= 2 {",
      "\t\tclassification.status = ObservedStable",
      "\t\tfor fingerprint := range counts {",
      "\t\t\tclassification.fingerprint = fingerprint",
      "\t\t\tbreak",
      "\t\t}",
    ].join("\n"),
    package: "./internal/observe",
    testName: "TestRunObservationRotatesFreshAttemptsAndNeverMajorityClassifiesAlternation",
  }),
  Object.freeze({
    id: "permit-partial-comparison-matrix",
    file: "internal/observe/schedule.go",
    find: "\tif b.budget.MaxTotalTrials-b.startedTrials < candidateCount {",
    replace: "\tif b.budget.MaxTotalTrials-b.startedTrials < 1 {",
    package: "./internal/observe",
    testName: "TestBudgetTrackerStopsBeforeSplittingATrialBoundMatrix",
  }),
  Object.freeze({
    id: "include-support-artifact-in-preservation-identity",
    file: "internal/compare/outcome_map.go",
    find: [
      "\t\tidentity.Entries[index] = labeledEntry{",
      "\t\t\tCandidateExecutionKey: semanticCandidateKey(entry),",
      "\t\t\tProjectionFingerprint: entry.ProjectionFingerprint.String(),",
      "\t\t}",
    ].join("\n"),
    replace: [
      "\t\tidentity.Entries[index] = labeledEntry{",
      "\t\t\tCandidateExecutionKey: semanticCandidateKey(entry),",
      "\t\t\tProjectionFingerprint: entry.ProjectionFingerprint.String() + \"\\x00\" + entry.StableBatchDigest.String(),",
      "\t\t}",
    ].join("\n"),
    package: "./internal/compare",
    testName: "TestEligibilityChangeBlocksPreservationComparability",
  }),
  Object.freeze({
    id: "drop-candidate-key-from-labeled-map-identity",
    file: "internal/compare/outcome_map.go",
    find: [
      "\t// MUTATION_ANCHOR: candidate-key-is-part-of-labeled-map-identity",
      "\treturn entry.CandidateKey.String()",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: candidate-key-is-part-of-labeled-map-identity",
      "\treturn \"\"",
    ].join("\n"),
    package: "./internal/compare",
    testName: "TestCandidateKeyIsPartOfLabeledMapIdentity",
  }),
  Object.freeze({
    id: "miswire-stderr-capture-and-report-requested-limit",
    file: "internal/processmechanics/process_darwin.go",
    find: [
      "func newCapturePair(stdoutLimit, stderrLimit int64, overflowC chan<- struct{}) capturePair {",
      "\tstdout := newCappedCapture(stdoutLimit, overflowC)",
      "\tstderr := newCappedCapture(stderrLimit, overflowC) // MUTANT_U2_SHARE_OUTPUT_CAP",
      "\treturn capturePair{",
      "\t\tstdout: stdout, stderr: stderr,",
      "\t\tstdoutReceiptLimit: stdout.configuredLimit(), stderrReceiptLimit: stderr.configuredLimit(),",
      "\t}",
      "}",
    ].join("\n"),
    replace: [
      "func newCapturePair(stdoutLimit, stderrLimit int64, overflowC chan<- struct{}) capturePair {",
      "\tstdout := newCappedCapture(stdoutLimit, overflowC)",
      "\tstderr := newCappedCapture(stdoutLimit, overflowC) // MUTANT_U2_SHARE_OUTPUT_CAP",
      "\treturn capturePair{",
      "\t\tstdout: stdout, stderr: stderr,",
      "\t\tstdoutReceiptLimit: stdout.configuredLimit(), stderrReceiptLimit: stderrLimit,",
      "\t}",
      "}",
    ].join("\n"),
    package: "./internal/processmechanics",
    testName: "TestStdoutAndStderrLimitsAreIndependentMutationGuard",
  }),
  Object.freeze({
    id: "collapse-physical-present-empty-stdin",
    file: "internal/processmechanics/process_darwin.go",
    find: [
      "\tif invocation.stdin.presence == stdinPresent {",
      "\t\tstdinWriter, err = command.StdinPipe()",
    ].join("\n"),
    replace: [
      "\tif invocation.stdin.presence == stdinPresent && len(invocation.stdin.bytes) > 0 {",
      "\t\tstdinWriter, err = command.StdinPipe()",
    ].join("\n"),
    package: "./internal/processmechanics",
    testName: "TestPresentEmptyStdinRemainsPhysicallyDistinctFromAbsentStdin",
  }),
  Object.freeze({
    id: "substitute-same-length-physical-stdin",
    file: "internal/processmechanics/process_darwin.go",
    find: [
      "\t\t// MUTATION_ANCHOR: stdin-absence-must-not-collapse-to-present-empty",
      "\t\tstartExactStdinWriter(stdinWriter, invocation.stdin.bytes, stdinResultC)",
    ].join("\n"),
    replace: [
      "\t\t// MUTATION_ANCHOR: stdin-absence-must-not-collapse-to-present-empty",
      "\t\tstartExactStdinWriter(stdinWriter, make([]byte, len(invocation.stdin.bytes)), stdinResultC)",
    ].join("\n"),
    package: "./internal/processmechanics",
    testName: "TestPresentEmptyStdinRemainsPhysicallyDistinctFromAbsentStdin",
  }),
  Object.freeze({
    id: "allow-present-invocation-as-absent",
    file: "internal/world/cli_invocation.go",
    find: [
      "\tcase CLIInvocationAbsent:",
      "\t\treturn r.presence == CLIInvocationPresenceAbsent && !r.fileDigest.Valid() && r.bytes == 0 && noValidation",
    ].join("\n"),
    replace: [
      "\tcase CLIInvocationAbsent:",
      "\t\treturn r.presence == CLIInvocationPresencePresent && !r.fileDigest.Valid() && r.bytes == 0 && noValidation",
    ].join("\n"),
    package: "./internal/world",
    testName: "TestCLIInvocationReceiptRejectsSelfSealedStatusContradictions",
  }),
  Object.freeze({
    id: "forge-preprocess-physical-entry",
    file: "internal/world/receipt.go",
    find: [
      "\t\t// MUTATION_ANCHOR: preprocess-control-must-not-forge-physical-entry",
      "\t\tphysicalExecutionEntered: input.cli.physicalExecutionEntered,",
    ].join("\n"),
    replace: [
      "\t\t// MUTATION_ANCHOR: preprocess-control-must-not-forge-physical-entry",
      "\t\tphysicalExecutionEntered: true,",
    ].join("\n"),
    package: "./internal/world",
    testName: "TestCLIPreprocessReceiptCannotForgePhysicalExecutionMutationGuard",
  }),
  Object.freeze({
    id: "allow-cross-paired-projection-rejection",
    file: "internal/observe/eligibility.go",
    find: [
      "\tif !rejection.Valid() || rejection.worldDigest != world.Digest() ||",
      "\t\trejection.attemptArtifactDigest != attempt.ArtifactDigest() ||",
      "\t\trejection.candidateKey != world.CandidateKey() ||",
      "\t\trejection.capturePolicyDigest != world.CapturePolicyDigest() ||",
      "\t\trejection.definitionDigest != world.ProjectionDefinitionDigest() {",
    ].join("\n"),
    replace: "\tif !rejection.Valid() || rejection.definitionDigest != world.ProjectionDefinitionDigest() {",
    package: "./internal/observe",
    testName: "TestProjectionRejectedTrialRejectsEveryCrossPairedLineage",
  }),
  Object.freeze({
    id: "omit-probe-transport-from-arbitration-receipt",
    file: "internal/world/process.go",
    find: "const terminalArbitrationContract = \"OWNER_OBSERVED_PRIORITY_OUTPUT_PROBE_TRANSPORT_CANCEL_DEADLINE_WAIT\"",
    replace: "const terminalArbitrationContract = \"OWNER_OBSERVED_PRIORITY_OUTPUT_CANCEL_DEADLINE_WAIT\"",
    package: "./internal/world",
    testName: "TestTerminalArbiterUsesFixedOwnerObservedPriority",
  }),
  Object.freeze({
    id: "bypass-reducible-direct-argv-grammar",
    file: "internal/adapters/cli/model/stimulus.go",
    find: [
      "\t// MUTATION_ANCHOR: reducible-argv-must-use-domain-direct-grammar",
      "\tif err := domain.ValidateDirectArgv(logicalArgv); err != nil {",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: reducible-argv-must-use-domain-direct-grammar",
      "\tif err := domain.ValidateDirectArgv(logicalArgv); false && err != nil {",
    ].join("\n"),
    package: "./internal/adapters/cli/model",
    testName: "TestCLIStimulusRejectsDirectArgvGrammarBypasses",
  }),
  Object.freeze({
    id: "bypass-resolved-capture-policy",
    file: "internal/adapters/cli/model/binding.go",
    find: [
      "\t// MUTATION_ANCHOR: execution-binding-must-resolve-capture-policy",
      "\tif plan.CapturePolicyDigest() != capturePolicy.Digest() ||",
      "\t\tbudgets.StdoutBytes != capturePolicy.StdoutBytes() || budgets.StderrBytes != capturePolicy.StderrBytes() {",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: execution-binding-must-resolve-capture-policy",
      "\tif false && (plan.CapturePolicyDigest() != capturePolicy.Digest() ||",
      "\t\tbudgets.StdoutBytes != capturePolicy.StdoutBytes() || budgets.StderrBytes != capturePolicy.StderrBytes()) {",
    ].join("\n"),
    package: "./internal/adapters/cli/model",
    testName: "TestBindExecutionResolvesExactCaptureAndProjectionAuthorities",
  }),
  Object.freeze({
    id: "bypass-resolved-adapter-projection",
    file: "internal/adapters/cli/model/binding.go",
    find: [
      "\t// MUTATION_ANCHOR: execution-binding-must-resolve-adapter-projection",
      "\tif plan.ProjectionDefinitionDigest() != projection.Binding().Digest() {",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: execution-binding-must-resolve-adapter-projection",
      "\tif false && plan.ProjectionDefinitionDigest() != projection.Binding().Digest() {",
    ].join("\n"),
    package: "./internal/adapters/cli/model",
    testName: "TestBindExecutionResolvesExactCaptureAndProjectionAuthorities",
  }),
  Object.freeze({
    id: "erase-declared-plan-rotation",
    file: "internal/domain/world.go",
    find: [
      "\t\t\t// MUTATION_ANCHOR: world-plan-must-retain-declared-rotation",
      "\t\t\tRotation: string(config.RepeatSchedule.Rotation),",
    ].join("\n"),
    replace: [
      "\t\t\t// MUTATION_ANCHOR: world-plan-must-retain-declared-rotation",
      "\t\t\tRotation: \"NOT_ESTABLISHED_IN_U1\",",
    ].join("\n"),
    package: "./internal/domain",
    testName: "TestWorldPlanRetainsDeclaredSequentialRotationMutationGuard",
  }),
  Object.freeze({
    id: "bypass-observation-plan-schedule",
    file: "internal/observe/batch.go",
    find: [
      "\t// MUTATION_ANCHOR: observation-schedule-must-match-world-plan",
      "\tif schedule.CandidateCount() != config.Plan.Budgets().CandidateCount ||",
      "\t\tschedule.Repetitions() != expectedRepeats ||",
      "\t\tschedule.Rotation() != config.Plan.ScheduleRotation() {",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: observation-schedule-must-match-world-plan",
      "\tif false && (schedule.CandidateCount() != config.Plan.Budgets().CandidateCount ||",
      "\t\tschedule.Repetitions() != expectedRepeats ||",
      "\t\tschedule.Rotation() != config.Plan.ScheduleRotation()) {",
    ].join("\n"),
    package: "./internal/observe",
    testName: "TestRunObservationBindsScheduleAndProducedWorldToExactPlanMutationGuard",
  }),
  Object.freeze({
    id: "bypass-observation-wall-admission",
    file: "internal/observe/schedule.go",
    find: [
      "func (b *budgetTracker) withinWall(now time.Time) bool {",
      "\tif b.wallExpired(now) {",
      "\t\tb.exhaustion = budgetWallExhausted",
      "\t\treturn false",
      "\t}",
      "\treturn true",
      "}",
    ].join("\n"),
    replace: [
      "func (b *budgetTracker) withinWall(now time.Time) bool {",
      "\t_ = now",
      "\treturn true",
      "}",
    ].join("\n"),
    package: "./internal/observe",
    testName: "TestRunObservationWallBudgetCoversPostProducerAdmissionMutationGuard",
  }),
  Object.freeze({
    id: "allow-reserved-git-fixture",
    file: "internal/adapters/cli/model/stimulus.go",
    find: [
      "\t\t// MUTATION_ANCHOR: reserved-git-metadata-must-be-rejected-before-world",
      "\t\tif strings.EqualFold(segment, \".git\") {",
    ].join("\n"),
    replace: [
      "\t\t// MUTATION_ANCHOR: reserved-git-metadata-must-be-rejected-before-world",
      "\t\tif false && strings.EqualFold(segment, \".git\") {",
    ].join("\n"),
    package: "./internal/adapters/cli/model",
    testName: "TestCLIStimulusFixtureValidationIsStrict",
  }),
  Object.freeze({
    id: "defer-physical-entry-until-start-success",
    file: "internal/processmechanics/process_darwin.go",
    find: [
      "\tresult.PhysicalExecutionEntered = true",
      "\tresult.SpawnAttempted = true",
      "\tresult.StdoutCaptureLimit = captures.stdoutReceiptLimit",
      "\tresult.StderrCaptureLimit = captures.stderrReceiptLimit",
      "\tif err := command.Start(); err != nil {",
      "\t\t_ = stdoutPipe.Close()",
      "\t\t_ = stdoutWriter.Close()",
      "\t\t_ = stderrPipe.Close()",
      "\t\t_ = stderrWriter.Close()",
      "\t\tif stdinWriter != nil {",
      "\t\t\t_ = stdinWriter.Close()",
      "\t\t}",
      "\t\treturn SpawnObservation{}, nil, newStartError(\"SPAWN_FAILED\", err, result)",
      "\t}",
    ].join("\n"),
    replace: [
      "\tresult.SpawnAttempted = true",
      "\tresult.StdoutCaptureLimit = captures.stdoutReceiptLimit",
      "\tresult.StderrCaptureLimit = captures.stderrReceiptLimit",
      "\tif err := command.Start(); err != nil {",
      "\t\t_ = stdoutPipe.Close()",
      "\t\t_ = stdoutWriter.Close()",
      "\t\t_ = stderrPipe.Close()",
      "\t\t_ = stderrWriter.Close()",
      "\t\tif stdinWriter != nil {",
      "\t\t\t_ = stdinWriter.Close()",
      "\t\t}",
      "\t\treturn SpawnObservation{}, nil, newStartError(\"SPAWN_FAILED\", err, result)",
      "\t}",
      "\tresult.PhysicalExecutionEntered = true",
    ].join("\n"),
    package: "./internal/world",
    testName: "TestPhysicalStartFailureRecordsAttemptedButUnstartedExecutionMutationGuard",
  }),
  Object.freeze({
    id: "discard-late-overflow-earlier-primary",
    file: "internal/adapters/cli/capture.go",
    find: [
      "\tcase domain.ControlOutputLimit, domain.ControlTimeout, domain.ControlCancelled,",
      "\t\tdomain.ControlProbeTransportError, domain.ControlStartError:",
    ].join("\n"),
    replace: [
      "\tcase domain.ControlOutputLimit, domain.ControlTimeout, domain.ControlCancelled,",
      "\t\tdomain.ControlProbeTransportError:",
    ].join("\n"),
    package: "./internal/adapters/cli",
    testName: "TestLateOutputOverflowRetainsEarlierPrimaryControl",
  }),
  Object.freeze({
    id: "bypass-environment-aggregate-limit",
    file: "internal/adapters/cli/model/stimulus.go",
    find: "\tif totalBytes > maxEnvironmentTotalBytes {",
    replace: "\tif false && totalBytes > maxEnvironmentTotalBytes {",
    package: "./internal/adapters/cli/model",
    testName: "TestCLIStimulusEnvironmentAggregateMatchesCanonicalHeadroom",
  }),
  Object.freeze({
    id: "trust-fixture-receipt-without-rehash",
    file: "internal/world/cli_fixture.go",
    find: "\treturn err == nil && digest.String() == r.digest.String() && bytes.Equal(canonicalBytes, r.canonicalBytes)",
    replace: [
      "\t_ = digest",
      "\t_ = canonicalBytes",
      "\treturn err == nil && r.digest.Valid()",
    ].join("\n"),
    package: "./internal/world",
    testName: "TestCLIFixtureOverlayBindsRecipeAndReopensExactBytes",
  }),
  Object.freeze({
    id: "misclassify-projection-resource-limit",
    file: "internal/adapters/cli/projection.go",
    find: "\t\tCode: CodeProjectionResourceLimit, Operation: operation, Channel: channel,",
    replace: "\t\tCode: CodeProjectionInternal, Operation: operation, Channel: channel,",
    package: "./internal/adapters/cli",
    testName: "TestCLIProjectionResourceLimitsRejectWithoutPanicOrDataLoss",
  }),
  Object.freeze({
    id: "allow-node-inline-script-string",
    file: "internal/adapters/cli/model/stimulus.go",
    find: [
      "\t// MUTATION_ANCHOR: node-must-name-one-fixed-repository-program",
      "\tif executable != \"node\" || !fixedNodeProgram(baseArgv) {",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: node-must-name-one-fixed-repository-program",
      "\tif false && (executable != \"node\" || !fixedNodeProgram(baseArgv)) {",
    ].join("\n"),
    package: "./internal/adapters/cli/model",
    testName: "TestCLIStimulusRejectsDirectArgvGrammarBypasses",
  }),
  Object.freeze({
    id: "publish-rejection-after-wall-budget",
    file: "internal/observe/batch.go",
    find: "\t\t\tif !tracker.withinWall(rejectedAt) {",
    replace: "\t\t\tif false && !tracker.withinWall(rejectedAt) {",
    package: "./internal/observe",
    testName: "TestRunObservationWallBudgetCannotExpireWhilePublishingRejectedMatrix",
  }),
  Object.freeze({
    id: "forge-physical-entry-before-spawn-attempt",
    file: "internal/processmechanics/process.go",
    find: [
      "\t\t// MUTATION_ANCHOR: physical-entry-requires-spawn-attempt",
      "\t\tPhysicalExecutionEntered: false,",
    ].join("\n"),
    replace: [
      "\t\t// MUTATION_ANCHOR: physical-entry-requires-spawn-attempt",
      "\t\tPhysicalExecutionEntered: true,",
    ].join("\n"),
    package: "./internal/processmechanics",
    testName: "TestCopiedPreparedHandleCannotMultiplyStartAuthority",
  }),
  Object.freeze({
    id: "overwrite-earlier-control-with-late-output-overflow",
    file: "internal/world/process.go",
    find: "\tif outputOverflowIsPrimaryControl && result.primary == \"\" && (result.stdoutOverflow || result.stderrOverflow) {",
    replace: "\tif outputOverflowIsPrimaryControl && (result.stdoutOverflow || result.stderrOverflow) {",
    package: "./internal/world",
    testName: "TestTerminalArbiterUsesFixedOwnerObservedPriority",
  }),
]);

export const U3_MUTANTS = Object.freeze(
  REVIEWED_U3_MUTANT_CONTRACT.map((mutant) => Object.freeze({ ...mutant })),
);

const tupleFields = Object.freeze(["file", "find", "id", "package", "replace", "testName"]);

function sameArray(left, right) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

function exactOccurrenceCount(source, anchor) {
  if (typeof anchor !== "string" || anchor.length === 0) return 0;
  return source.split(anchor).length - 1;
}

export function assertU3MutantDefinitionSet(
  mutants = U3_MUTANTS,
  requiredIDs = REQUIRED_U3_MUTANT_IDS,
  reviewedContract = REVIEWED_U3_MUTANT_CONTRACT,
) {
  const requiredCount = 43;
  if (requiredIDs.length !== requiredCount || new Set(requiredIDs).size !== requiredCount) {
    throw new MutationGateError("INVALID_REQUIRED_ID_SET", `U3 requires exactly ${requiredCount} unique frozen IDs`);
  }
  if (mutants.length !== requiredCount) {
    throw new MutationGateError("MUTANT_SET_MISMATCH", `defined ${mutants.length} U3 mutants, want ${requiredCount}`);
  }
  const ids = mutants.map((mutant) => mutant.id);
  if (new Set(ids).size !== ids.length) {
    throw new MutationGateError("DUPLICATE_MUTANT_ID", "U3 mutant IDs must be unique");
  }
  if (!sameArray(ids, requiredIDs)) {
    throw new MutationGateError("MUTANT_SET_MISMATCH", "U3 mutant IDs or reviewed order changed");
  }
  if (reviewedContract.length !== requiredCount || !sameArray(reviewedContract.map((tuple) => tuple.id), requiredIDs)) {
    throw new MutationGateError("INVALID_REVIEWED_MUTANT_CONTRACT", "the independent U3 tuple contract changed shape or order");
  }
  const allowedFiles = new Set(U3_SANDBOX_FILE_ALLOWLIST);
  const reviewedByID = new Map(reviewedContract.map((tuple) => [tuple.id, tuple]));
  for (const mutant of mutants) {
    const reviewed = reviewedByID.get(mutant.id);
    if (!reviewed || !sameArray(Object.keys(mutant).sort(), tupleFields)) {
      throw new MutationGateError("MUTANT_CONTRACT_MISMATCH", `${mutant.id} differs from the reviewed tuple shape`);
    }
    for (const field of tupleFields) {
      if (mutant[field] !== reviewed[field]) {
        throw new MutationGateError("MUTANT_CONTRACT_MISMATCH", `${mutant.id}.${field} differs from the reviewed tuple`);
      }
    }
    if (!allowedFiles.has(mutant.file)) {
      throw new MutationGateError("MUTANT_FILE_NOT_ALLOWLISTED", `${mutant.id} targets ${mutant.file}`);
    }
    if (mutant.find.length === 0 || mutant.find === mutant.replace) {
      throw new MutationGateError("INVALID_MUTATION", `${mutant.id} is not one exact effective replacement`);
    }
    if (!/^\.\/internal\/(?:adapters\/cli(?:\/model)?|compare|domain|observe|processmechanics|world)$/u.test(mutant.package)) {
      throw new MutationGateError("INVALID_MUTATION_PACKAGE", `${mutant.id} has invalid package ${mutant.package}`);
    }
    if (!/^Test[A-Za-z0-9_]+$/u.test(mutant.testName)) {
      throw new MutationGateError("INVALID_MUTATION_TEST", `${mutant.id} has invalid named test ${mutant.testName}`);
    }
  }
}

export async function assertU3ManifestMatchesTree(sourceRoot = repoRoot) {
  return assertU2ManifestMatchesTree(sourceRoot, U3_SANDBOX_FILE_ALLOWLIST, U3_SOURCE_SCOPE_ROOTS);
}

export async function assertReviewedU3Anchors(sourceRoot = repoRoot, mutants = U3_MUTANTS) {
  assertU3MutantDefinitionSet(mutants);
  for (const mutant of mutants) {
    const source = await readFile(resolve(sourceRoot, mutant.file), "utf8");
    const count = exactOccurrenceCount(source, mutant.find);
    if (count !== 1) {
      throw new MutationGateError("MUTATION_ANCHOR_COUNT", `${mutant.id}: exact anchor count was ${count}, want 1`);
    }
    const mutated = applyExactMutation(source, mutant);
    if (mutated === source || exactOccurrenceCount(mutated, mutant.find) !== 0) {
      throw new MutationGateError("MUTATION_REPLACEMENT_INVALID", `${mutant.id}: replacement is not exact and singular`);
    }
  }
}

function fakeClassification(outcome) {
  return { classification: { outcome, detail: outcome }, output: "" };
}

export async function selfTest() {
  assertU3MutantDefinitionSet();
  await assertU3ManifestMatchesTree(repoRoot);
  await assertReviewedU3Anchors(repoRoot);

  assert.throws(
    () => assertU3MutantDefinitionSet(U3_MUTANTS.slice(0, -1)),
    (error) => error instanceof MutationGateError && error.code === "MUTANT_SET_MISMATCH",
  );
  const reordered = [...U3_MUTANTS];
  [reordered[0], reordered[1]] = [reordered[1], reordered[0]];
  assert.throws(
    () => assertU3MutantDefinitionSet(reordered),
    (error) => error instanceof MutationGateError && error.code === "MUTANT_SET_MISMATCH",
  );
  const tampered = U3_MUTANTS.map((mutant) => ({ ...mutant }));
  tampered[0].replace += "\n// unreviewed weakening";
  assert.throws(
    () => assertU3MutantDefinitionSet(tampered),
    (error) => error instanceof MutationGateError && error.code === "MUTANT_CONTRACT_MISMATCH",
  );

  const phases = [];
  let invocation = 0;
  const killed = await runU2Mutant(U3_MUTANTS[0], {}, {
    async prepareExperiment({ phase }) {
      phases.push(phase);
      return { sandbox: `/fresh/u3/${phase}`, sandboxParent: `/fresh/u3/${phase}` };
    },
    async cleanupExperiment() {},
    async applyMutation() {},
    async runNamedTest() {
      const outcomes = [NamedTestOutcome.Pass, NamedTestOutcome.Failure, NamedTestOutcome.Pass];
      return fakeClassification(outcomes[invocation++]);
    },
  });
  assert.match(killed, /^KILLED reverse-stimulus-argv/u);
  assert.deepEqual(phases, ["baseline", "mutant", "post-control"]);
  assert.equal(invocation, 3);
  return `U3 mutation self-test passed: ${U3_MUTANTS.length} frozen tuples, exact manifest, exact anchors, fresh A/B/A control.`;
}

export async function main(arguments_ = process.argv.slice(2)) {
  if (arguments_.length === 1 && arguments_[0] === "--self-test") {
    process.stdout.write(`${await selfTest()}\n`);
    return;
  }
  if (arguments_.length !== 0) {
    throw new MutationGateError("UNSUPPORTED_ARGUMENT", "usage: node tools/mutate-u3.mjs [--self-test]");
  }

  assertU3MutantDefinitionSet();
  await assertU3ManifestMatchesTree(repoRoot);
  await assertReviewedU3Anchors(repoRoot);
  const toolchain = await admitU2DarwinToolchain();
  const seed = await createU2SeedSnapshot({
    sourceRoot: repoRoot,
    allowlist: U3_SANDBOX_FILE_ALLOWLIST,
    scopeRoots: U3_SOURCE_SCOPE_ROOTS,
  });
  const results = [];
  try {
    for (const mutant of U3_MUTANTS) {
      results.push(await runU2Mutant(mutant, toolchain, { seedSnapshot: seed }));
    }
    await assertU2SourceAndSeedUnchanged(seed);
  } finally {
    await cleanupU2SeedSnapshot(seed);
  }
  revalidateAdmittedGoExecutable(toolchain.executable);
  revalidateAdmittedDarwinCCompiler(toolchain.compiler);
  process.stdout.write(`Trusted Go realpath: ${toolchain.executable.path}\n`);
  process.stdout.write(`Trusted Go sha256: ${toolchain.executable.sha256}\n`);
  process.stdout.write(`Trusted Go version: ${toolchain.version.line}\n`);
  process.stdout.write(`Trusted C compiler realpath: ${toolchain.compiler.path}\n`);
  process.stdout.write(`Trusted C compiler sha256: ${toolchain.compiler.sha256}\n`);
  process.stdout.write(`Closed Darwin CGO facts digest: ${toolchain.cgo.digest}\n`);
  process.stdout.write(`Immutable U3 seed manifest: ${seed.manifest.digest}\n`);
  process.stdout.write("Containment notice: private mutation copies retain the invoking user's host filesystem and network authority.\n");
  for (const result of results) process.stdout.write(`${result}\n`);
  process.stdout.write(`U3 mutation gate killed ${results.length}/${REQUIRED_U3_MUTANT_IDS.length} exact required mutants.\n`);
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(modulePath)) {
  main().catch((error) => {
    process.stderr.write(`${error.stack ?? error}\n`);
    process.exitCode = 1;
  });
}
