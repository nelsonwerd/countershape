#!/usr/bin/env node

import { createHash } from "node:crypto";
import { arch, platform } from "node:os";
import { pathToFileURL } from "node:url";

import {
	acquireVerificationLock,
	admitTools,
	buildChildEnvironment,
	childResult,
	cleanupVerificationResources,
	createPrivateRoots,
	finalizeVerificationResources,
	repositoryRoot,
} from "./verify-runtime-authority.mjs";

const modulePath = "github.com/nelsonwerd/countershape";
const allowedActions = new Set(["bench", "cont", "fail", "output", "pass", "pause", "run", "skip", "start"]);
const packageActions = new Set(["fail", "output", "pass", "start"]);
const testActions = new Set(["cont", "fail", "output", "pass", "pause", "run", "skip"]);
const expectedEntrypointPattern = /^(?:(?:Test|Fuzz)[A-Za-z0-9_]+|Example[A-Za-z0-9_]*)$/u;
const authorityNames = Object.freeze(["go", "node", "git", "sh", "cc", "cxx"]);
const sensitiveGoPackages = Object.freeze([
	`${modulePath}/internal/contractexec/runner`,
	`${modulePath}/internal/emit/node/compiler`,
	`${modulePath}/internal/emit/node/program/v1`,
	`${modulePath}/internal/processmechanics`,
	`${modulePath}/internal/store`,
	`${modulePath}/internal/world`,
	`${modulePath}/testkit/contractexec/cli`,
	`${modulePath}/testkit/contractexec/http`,
	`${modulePath}/testkit/studies/cli_precedence`,
	`${modulePath}/testkit/studies/http_invoices`,
]);

// GO_REPETITION_PROFILE_CLASS was the sealed ad-hoc parser refusal. C4 replaces
// caller-supplied profiles with identity-selected, recursively frozen cases.

export class GoRepetitionError extends Error {
	constructor(code, detail) {
		super(`${code}: ${detail}`);
		this.code = code;
	}
}

function fail(code, detail) {
	throw new GoRepetitionError(code, detail);
}

export const qualificationMatrices = Object.freeze({
	"C4": Object.freeze({
		"caseIDs": Object.freeze([
			"processmechanics-output-caps-50",
			"processmechanics-output-independence-20",
			"processmechanics-simultaneous-overflow-20",
			"world-lifecycle-readiness-20",
			"contract-runner-admission-50",
			"contract-cli-standalone-closure-20",
			"compiler-generated-runtime-20",
			"program-lifecycle-20",
			"store-cross-process-cas-20",
			"cli-physical-reducer-01-of-20",
			"cli-physical-reducer-02-of-20",
			"cli-physical-reducer-03-of-20",
			"cli-physical-reducer-04-of-20",
			"cli-physical-reducer-05-of-20",
			"cli-physical-reducer-06-of-20",
			"cli-physical-reducer-07-of-20",
			"cli-physical-reducer-08-of-20",
			"cli-physical-reducer-09-of-20",
			"cli-physical-reducer-10-of-20",
			"cli-physical-reducer-11-of-20",
			"cli-physical-reducer-12-of-20",
			"cli-physical-reducer-13-of-20",
			"cli-physical-reducer-14-of-20",
			"cli-physical-reducer-15-of-20",
			"cli-physical-reducer-16-of-20",
			"cli-physical-reducer-17-of-20",
			"cli-physical-reducer-18-of-20",
			"cli-physical-reducer-19-of-20",
			"cli-physical-reducer-20-of-20",
			"http-physical-reducer-01-of-20",
			"http-physical-reducer-02-of-20",
			"http-physical-reducer-03-of-20",
			"http-physical-reducer-04-of-20",
			"http-physical-reducer-05-of-20",
			"http-physical-reducer-06-of-20",
			"http-physical-reducer-07-of-20",
			"http-physical-reducer-08-of-20",
			"http-physical-reducer-09-of-20",
			"http-physical-reducer-10-of-20",
			"http-physical-reducer-11-of-20",
			"http-physical-reducer-12-of-20",
			"http-physical-reducer-13-of-20",
			"http-physical-reducer-14-of-20",
			"http-physical-reducer-15-of-20",
			"http-physical-reducer-16-of-20",
			"http-physical-reducer-17-of-20",
			"http-physical-reducer-18-of-20",
			"http-physical-reducer-19-of-20",
			"http-physical-reducer-20-of-20",
			"parity-evaluator-20",
			"parity-framing-20",
			"parity-full-package-3",
			"cli-physical-full-package-3",
			"http-physical-full-package-3",
		]),
		"cases": Object.freeze({
			"processmechanics-output-caps-50": Object.freeze({
				"caseID": "processmechanics-output-caps-50",
				"count": 50,
				"expected": Object.freeze([
					"TestStdoutAndStderrHaveIndependentExactCaps",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/processmechanics",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestStdoutAndStderrHaveIndependentExactCaps$",
			}),
			"processmechanics-output-independence-20": Object.freeze({
				"caseID": "processmechanics-output-independence-20",
				"count": 20,
				"expected": Object.freeze([
					"TestStdoutAndStderrLimitsAreIndependentMutationGuard",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/processmechanics",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestStdoutAndStderrLimitsAreIndependentMutationGuard$",
			}),
			"processmechanics-simultaneous-overflow-20": Object.freeze({
				"caseID": "processmechanics-simultaneous-overflow-20",
				"count": 20,
				"expected": Object.freeze([
					"TestSimultaneousChannelOverflowRetainsIndependentFacts",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/processmechanics",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestSimultaneousChannelOverflowRetainsIndependentFacts$",
			}),
			"world-lifecycle-readiness-20": Object.freeze({
				"caseID": "world-lifecycle-readiness-20",
				"count": 20,
				"expected": Object.freeze([
					"TestExecuteBuildsFreshWorldsAndPublishesMarkerBeforeSpawn",
					"TestFinalGroupProbeRequiresObservedAbsence",
					"TestHTTPPortableEarlyExitRetainsCausallyLaterReadinessEOF",
					"TestHTTPPortableReadinessFailuresRemainFinalizedReceipts",
					"TestPreTermProbeControlsWhetherTheOriginalGroupIsSignaled",
					"TestProcessGroupAndSessionEscapesRemainExplicitExclusions",
					"TestProcessLifecycleControlsAndCleansDescendants",
					"TestToolVersionProbeCleansDescendantHeldPipesWithinItsBound",
					"TestUnexpectedWaitFailureIsNotACompletedCleanupEdge",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/world",
				"profile": "sensitive",
				"qualification": true,
				"run": "^(?:TestExecuteBuildsFreshWorldsAndPublishesMarkerBeforeSpawn|TestFinalGroupProbeRequiresObservedAbsence|TestHTTPPortableEarlyExitRetainsCausallyLaterReadinessEOF|TestHTTPPortableReadinessFailuresRemainFinalizedReceipts|TestPreTermProbeControlsWhetherTheOriginalGroupIsSignaled|TestProcessGroupAndSessionEscapesRemainExplicitExclusions|TestProcessLifecycleControlsAndCleansDescendants|TestToolVersionProbeCleansDescendantHeldPipesWithinItsBound|TestUnexpectedWaitFailureIsNotACompletedCleanupEdge)$",
			}),
			"contract-runner-admission-50": Object.freeze({
				"caseID": "contract-runner-admission-50",
				"count": 50,
				"expected": Object.freeze([
					"TestConcurrentAdmissionProducesExactlyOneStart",
					"TestRunPermitConsumptionIsSingleUseAndAdjacentToStart",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/contractexec/runner",
				"profile": "sensitive",
				"qualification": true,
				"run": "^(?:TestConcurrentAdmissionProducesExactlyOneStart|TestRunPermitConsumptionIsSingleUseAndAdjacentToStart)$",
			}),
			"contract-cli-standalone-closure-20": Object.freeze({
				"caseID": "contract-cli-standalone-closure-20",
				"count": 20,
				"expected": Object.freeze([
					"TestCLIContractExecutionClosesStandaloneScope",
					"TestCLIContractExecutionForbiddenPositiveControls",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/contractexec/cli",
				"profile": "sensitive",
				"qualification": true,
				"run": "^(?:TestCLIContractExecutionClosesStandaloneScope|TestCLIContractExecutionForbiddenPositiveControls)$",
			}),
			"compiler-generated-runtime-20": Object.freeze({
				"caseID": "compiler-generated-runtime-20",
				"count": 20,
				"expected": Object.freeze([
					"TestGeneratedCLIContractDistinguishesAbsentAndPresentEmptyStdin",
					"TestGeneratedCLIContractRunsFromPrivateTargetInventory",
					"TestGeneratedContractReportsHarnessFailureForVerifiedLoaderFailure",
					"TestGeneratedHTTPContractClassifiesProbeTimeout",
					"TestGeneratedHTTPContractClassifiesSocketReset",
					"TestGeneratedHTTPContractRunsFromPrivateTargetInventory",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/emit/node/compiler",
				"profile": "sensitive",
				"qualification": true,
				"run": "^(?:TestGeneratedCLIContractDistinguishesAbsentAndPresentEmptyStdin|TestGeneratedCLIContractRunsFromPrivateTargetInventory|TestGeneratedContractReportsHarnessFailureForVerifiedLoaderFailure|TestGeneratedHTTPContractClassifiesProbeTimeout|TestGeneratedHTTPContractClassifiesSocketReset|TestGeneratedHTTPContractRunsFromPrivateTargetInventory)$",
			}),
			"program-lifecycle-20": Object.freeze({
				"caseID": "program-lifecycle-20",
				"count": 20,
				"expected": Object.freeze([
					"TestCopiedHarnessLifecycleStateMachines",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/emit/node/program/v1",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCopiedHarnessLifecycleStateMachines$",
			}),
			"store-cross-process-cas-20": Object.freeze({
				"caseID": "store-cross-process-cas-20",
				"count": 20,
				"expected": Object.freeze([
					"TestStudyHeadCrossProcessCASHasExactlyOneWinner",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/store",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestStudyHeadCrossProcessCASHasExactlyOneWinner$",
			}),
			"cli-physical-reducer-01-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-01-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-02-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-02-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-03-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-03-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-04-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-04-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-05-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-05-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-06-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-06-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-07-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-07-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-08-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-08-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-09-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-09-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-10-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-10-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-11-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-11-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-12-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-12-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-13-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-13-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-14-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-14-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-15-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-15-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-16-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-16-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-17-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-17-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-18-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-18-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-19-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-19-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-20-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-20-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"http-physical-reducer-01-of-20": Object.freeze({
				"caseID": "http-physical-reducer-01-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-02-of-20": Object.freeze({
				"caseID": "http-physical-reducer-02-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-03-of-20": Object.freeze({
				"caseID": "http-physical-reducer-03-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-04-of-20": Object.freeze({
				"caseID": "http-physical-reducer-04-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-05-of-20": Object.freeze({
				"caseID": "http-physical-reducer-05-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-06-of-20": Object.freeze({
				"caseID": "http-physical-reducer-06-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-07-of-20": Object.freeze({
				"caseID": "http-physical-reducer-07-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-08-of-20": Object.freeze({
				"caseID": "http-physical-reducer-08-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-09-of-20": Object.freeze({
				"caseID": "http-physical-reducer-09-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-10-of-20": Object.freeze({
				"caseID": "http-physical-reducer-10-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-11-of-20": Object.freeze({
				"caseID": "http-physical-reducer-11-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-12-of-20": Object.freeze({
				"caseID": "http-physical-reducer-12-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-13-of-20": Object.freeze({
				"caseID": "http-physical-reducer-13-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-14-of-20": Object.freeze({
				"caseID": "http-physical-reducer-14-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-15-of-20": Object.freeze({
				"caseID": "http-physical-reducer-15-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-16-of-20": Object.freeze({
				"caseID": "http-physical-reducer-16-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-17-of-20": Object.freeze({
				"caseID": "http-physical-reducer-17-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-18-of-20": Object.freeze({
				"caseID": "http-physical-reducer-18-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-19-of-20": Object.freeze({
				"caseID": "http-physical-reducer-19-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-20-of-20": Object.freeze({
				"caseID": "http-physical-reducer-20-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"parity-evaluator-20": Object.freeze({
				"caseID": "parity-evaluator-20",
				"count": 20,
				"expected": Object.freeze([
					"TestGoAndNodeParityEvaluatorsMatchLiteralOracle",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/emit/node/parity",
				"profile": "general",
				"qualification": true,
				"run": "^TestGoAndNodeParityEvaluatorsMatchLiteralOracle$",
			}),
			"parity-framing-20": Object.freeze({
				"caseID": "parity-framing-20",
				"count": 20,
				"expected": Object.freeze([
					"TestNodeParityRunnerAcceptsExactWholeWireCap",
					"TestNodeParityRunnerRejectsInvalidFramesAtomically",
					"TestNodeParityRunnerRejectsMissingFinalLFWithoutStderr",
					"TestParityFramingRejectsExpandedSemanticResultAtomically",
					"TestParityResponseFramingExactBodyBoundary",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/emit/node/parity",
				"profile": "general",
				"qualification": true,
				"run": "^(?:TestNodeParityRunnerAcceptsExactWholeWireCap|TestNodeParityRunnerRejectsInvalidFramesAtomically|TestNodeParityRunnerRejectsMissingFinalLFWithoutStderr|TestParityFramingRejectsExpandedSemanticResultAtomically|TestParityResponseFramingExactBodyBoundary)$",
			}),
			"parity-full-package-3": Object.freeze({
				"caseID": "parity-full-package-3",
				"count": 3,
				"expected": Object.freeze([
					"FuzzParseContractParityCorpusLine",
					"TestContractParityCorpus",
					"TestContractParityCorpusExactSizeBoundaries",
					"TestContractParityCorpusIsOrderAndOracleIndependent",
					"TestContractParityCorpusParserRejectsEnvelopeAliases",
					"TestContractParityCorpusRejectsClosedSchemaDrift",
					"TestContractParityManifestMatchesCopiedEntrypointParser",
					"TestContractParityOracleLeakMutantsAreKilledByRequestRoster",
					"TestDirectResultSelectorExhaustiveGoNodeMatrix",
					"TestGoAndNodeParityEvaluatorsMatchLiteralOracle",
					"TestNodeParityRunnerAcceptsExactWholeWireCap",
					"TestNodeParityRunnerRejectsInvalidFramesAtomically",
					"TestNodeParityRunnerRejectsMissingFinalLFWithoutStderr",
					"TestOwnerEligibilitySelectorExhaustiveGoNodeMatrix",
					"TestParityFramingRejectsExpandedSemanticResultAtomically",
					"TestParityResponseFramingExactBodyBoundary",
					"TestResultFrameBytesExactBoundaries",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/emit/node/parity",
				"profile": "general",
				"qualification": true,
				"run": "^(?:Test|Fuzz|Example).*$",
			}),
			"cli-physical-full-package-3": Object.freeze({
				"caseID": "cli-physical-full-package-3",
				"count": 3,
				"expected": Object.freeze([
					"TestCLICompilationFreshProcessRestartHelper",
					"TestCLIMaterializationControlCrossesAdapterAndExcludedMap",
					"TestCLIP07BBPublicationHelper",
					"TestCLIPhysicalReducerBudgetFenceRetainsOnlyBestKnown",
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
					"TestCLIPresentBytesStdinCrossesWorldAndAdapter",
					"TestCLIPresentEmptyStdinCrossesWorldAndAdapter",
					"TestCLIReferenceControlClassificationMatrix",
					"TestCLIReferenceDisplayPermutationPreservesMapWithFreshAttempts",
					"TestCLIReferencePrecedenceStudy",
					"TestCLIStudyUsesStrictSourceCompilerAndPlanBoundSchedule",
					"TestOptionalReceiptMeasurementKeepsAbsenceExplicit",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^(?:Test|Fuzz|Example).*$",
			}),
			"http-physical-full-package-3": Object.freeze({
				"caseID": "http-physical-full-package-3",
				"count": 3,
				"expected": Object.freeze([
					"TestHTTPCompilationFreshProcessRestartHelper",
					"TestHTTPInvoiceAlternatingCandidateUsesAllTrialsNeverMajority",
					"TestHTTPInvoiceOutcomeMapUsesExactCandidateLabelsNotGroupShape",
					"TestHTTPInvoicePermutationPreservesExactMapWithFreshEvidence",
					"TestHTTPInvoicePortableChildBindPhysicalLineage",
					"TestHTTPInvoiceReferenceStudy",
					"TestHTTPInvoiceRequestFactsDrivePhysicalPolicy",
					"TestHTTPInvoiceStudyRecordsTimingAndTrialMultiplication",
					"TestHTTPInvoiceStudyUsesStrictSourceCompilerAndPlanBoundSchedule",
					"TestHTTPInvoiceTenantSeedShapeTrapIsPhysicalAndLabelSensitive",
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
					"TestHTTPPhysicalTenantSeedNeighborChangesExactLabeledMapWithStableRoster",
					"TestNegativeSharedRootContaminationCreatesFalseEquality",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^(?:Test|Fuzz|Example).*$",
			}),
		}),
		"digest": "3be703fae10155c83b19be54ae7cd79cfc5b1bbaddfcf86bb2c9781ad93f853a",
	}),
	"C5": Object.freeze({
		"caseIDs": Object.freeze([
			"processmechanics-output-caps-50",
			"processmechanics-output-independence-20",
			"processmechanics-simultaneous-overflow-20",
			"world-lifecycle-readiness-20",
			"contract-runner-admission-50",
			"contract-cli-standalone-closure-20",
			"contract-http-readiness-50",
			"contract-http-teardown-20",
			"compiler-generated-runtime-20",
			"program-lifecycle-20",
			"store-cross-process-cas-20",
			"cli-physical-reducer-01-of-20",
			"cli-physical-reducer-02-of-20",
			"cli-physical-reducer-03-of-20",
			"cli-physical-reducer-04-of-20",
			"cli-physical-reducer-05-of-20",
			"cli-physical-reducer-06-of-20",
			"cli-physical-reducer-07-of-20",
			"cli-physical-reducer-08-of-20",
			"cli-physical-reducer-09-of-20",
			"cli-physical-reducer-10-of-20",
			"cli-physical-reducer-11-of-20",
			"cli-physical-reducer-12-of-20",
			"cli-physical-reducer-13-of-20",
			"cli-physical-reducer-14-of-20",
			"cli-physical-reducer-15-of-20",
			"cli-physical-reducer-16-of-20",
			"cli-physical-reducer-17-of-20",
			"cli-physical-reducer-18-of-20",
			"cli-physical-reducer-19-of-20",
			"cli-physical-reducer-20-of-20",
			"http-physical-reducer-01-of-20",
			"http-physical-reducer-02-of-20",
			"http-physical-reducer-03-of-20",
			"http-physical-reducer-04-of-20",
			"http-physical-reducer-05-of-20",
			"http-physical-reducer-06-of-20",
			"http-physical-reducer-07-of-20",
			"http-physical-reducer-08-of-20",
			"http-physical-reducer-09-of-20",
			"http-physical-reducer-10-of-20",
			"http-physical-reducer-11-of-20",
			"http-physical-reducer-12-of-20",
			"http-physical-reducer-13-of-20",
			"http-physical-reducer-14-of-20",
			"http-physical-reducer-15-of-20",
			"http-physical-reducer-16-of-20",
			"http-physical-reducer-17-of-20",
			"http-physical-reducer-18-of-20",
			"http-physical-reducer-19-of-20",
			"http-physical-reducer-20-of-20",
			"parity-evaluator-20",
			"parity-framing-20",
			"parity-full-package-3",
			"cli-physical-full-package-3",
			"http-physical-full-package-3",
		]),
		"cases": Object.freeze({
			"processmechanics-output-caps-50": Object.freeze({
				"caseID": "processmechanics-output-caps-50",
				"count": 50,
				"expected": Object.freeze([
					"TestStdoutAndStderrHaveIndependentExactCaps",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/processmechanics",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestStdoutAndStderrHaveIndependentExactCaps$",
			}),
			"processmechanics-output-independence-20": Object.freeze({
				"caseID": "processmechanics-output-independence-20",
				"count": 20,
				"expected": Object.freeze([
					"TestStdoutAndStderrLimitsAreIndependentMutationGuard",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/processmechanics",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestStdoutAndStderrLimitsAreIndependentMutationGuard$",
			}),
			"processmechanics-simultaneous-overflow-20": Object.freeze({
				"caseID": "processmechanics-simultaneous-overflow-20",
				"count": 20,
				"expected": Object.freeze([
					"TestSimultaneousChannelOverflowRetainsIndependentFacts",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/processmechanics",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestSimultaneousChannelOverflowRetainsIndependentFacts$",
			}),
			"world-lifecycle-readiness-20": Object.freeze({
				"caseID": "world-lifecycle-readiness-20",
				"count": 20,
				"expected": Object.freeze([
					"TestExecuteBuildsFreshWorldsAndPublishesMarkerBeforeSpawn",
					"TestFinalGroupProbeRequiresObservedAbsence",
					"TestHTTPPortableEarlyExitRetainsCausallyLaterReadinessEOF",
					"TestHTTPPortableReadinessFailuresRemainFinalizedReceipts",
					"TestPreTermProbeControlsWhetherTheOriginalGroupIsSignaled",
					"TestProcessGroupAndSessionEscapesRemainExplicitExclusions",
					"TestProcessLifecycleControlsAndCleansDescendants",
					"TestToolVersionProbeCleansDescendantHeldPipesWithinItsBound",
					"TestUnexpectedWaitFailureIsNotACompletedCleanupEdge",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/world",
				"profile": "sensitive",
				"qualification": true,
				"run": "^(?:TestExecuteBuildsFreshWorldsAndPublishesMarkerBeforeSpawn|TestFinalGroupProbeRequiresObservedAbsence|TestHTTPPortableEarlyExitRetainsCausallyLaterReadinessEOF|TestHTTPPortableReadinessFailuresRemainFinalizedReceipts|TestPreTermProbeControlsWhetherTheOriginalGroupIsSignaled|TestProcessGroupAndSessionEscapesRemainExplicitExclusions|TestProcessLifecycleControlsAndCleansDescendants|TestToolVersionProbeCleansDescendantHeldPipesWithinItsBound|TestUnexpectedWaitFailureIsNotACompletedCleanupEdge)$",
			}),
			"contract-runner-admission-50": Object.freeze({
				"caseID": "contract-runner-admission-50",
				"count": 50,
				"expected": Object.freeze([
					"TestConcurrentAdmissionProducesExactlyOneStart",
					"TestRunPermitConsumptionIsSingleUseAndAdjacentToStart",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/contractexec/runner",
				"profile": "sensitive",
				"qualification": true,
				"run": "^(?:TestConcurrentAdmissionProducesExactlyOneStart|TestRunPermitConsumptionIsSingleUseAndAdjacentToStart)$",
			}),
			"contract-cli-standalone-closure-20": Object.freeze({
				"caseID": "contract-cli-standalone-closure-20",
				"count": 20,
				"expected": Object.freeze([
					"TestCLIContractExecutionClosesStandaloneScope",
					"TestCLIContractExecutionForbiddenPositiveControls",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/contractexec/cli",
				"profile": "sensitive",
				"qualification": true,
				"run": "^(?:TestCLIContractExecutionClosesStandaloneScope|TestCLIContractExecutionForbiddenPositiveControls)$",
			}),
			"contract-http-readiness-50": Object.freeze({
				"caseID": "contract-http-readiness-50",
				"count": 50,
				"expected": Object.freeze([
					"TestHTTPChildReportedReadinessBindsExactService",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/contractexec/http",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPChildReportedReadinessBindsExactService$",
			}),
			"contract-http-teardown-20": Object.freeze({
				"caseID": "contract-http-teardown-20",
				"count": 20,
				"expected": Object.freeze([
					"TestHTTPEarlyExitAndTeardownRetainCausalFacts",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/contractexec/http",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPEarlyExitAndTeardownRetainCausalFacts$",
			}),
			"compiler-generated-runtime-20": Object.freeze({
				"caseID": "compiler-generated-runtime-20",
				"count": 20,
				"expected": Object.freeze([
					"TestGeneratedCLIContractDistinguishesAbsentAndPresentEmptyStdin",
					"TestGeneratedCLIContractRunsFromPrivateTargetInventory",
					"TestGeneratedContractReportsHarnessFailureForVerifiedLoaderFailure",
					"TestGeneratedHTTPContractClassifiesProbeTimeout",
					"TestGeneratedHTTPContractClassifiesSocketReset",
					"TestGeneratedHTTPContractRunsFromPrivateTargetInventory",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/emit/node/compiler",
				"profile": "sensitive",
				"qualification": true,
				"run": "^(?:TestGeneratedCLIContractDistinguishesAbsentAndPresentEmptyStdin|TestGeneratedCLIContractRunsFromPrivateTargetInventory|TestGeneratedContractReportsHarnessFailureForVerifiedLoaderFailure|TestGeneratedHTTPContractClassifiesProbeTimeout|TestGeneratedHTTPContractClassifiesSocketReset|TestGeneratedHTTPContractRunsFromPrivateTargetInventory)$",
			}),
			"program-lifecycle-20": Object.freeze({
				"caseID": "program-lifecycle-20",
				"count": 20,
				"expected": Object.freeze([
					"TestCopiedHarnessLifecycleStateMachines",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/emit/node/program/v1",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCopiedHarnessLifecycleStateMachines$",
			}),
			"store-cross-process-cas-20": Object.freeze({
				"caseID": "store-cross-process-cas-20",
				"count": 20,
				"expected": Object.freeze([
					"TestStudyHeadCrossProcessCASHasExactlyOneWinner",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/store",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestStudyHeadCrossProcessCASHasExactlyOneWinner$",
			}),
			"cli-physical-reducer-01-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-01-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-02-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-02-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-03-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-03-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-04-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-04-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-05-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-05-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-06-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-06-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-07-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-07-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-08-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-08-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-09-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-09-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-10-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-10-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-11-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-11-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-12-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-12-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-13-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-13-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-14-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-14-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-15-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-15-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-16-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-16-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-17-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-17-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-18-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-18-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-19-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-19-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"cli-physical-reducer-20-of-20": Object.freeze({
				"caseID": "cli-physical-reducer-20-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence$",
			}),
			"http-physical-reducer-01-of-20": Object.freeze({
				"caseID": "http-physical-reducer-01-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-02-of-20": Object.freeze({
				"caseID": "http-physical-reducer-02-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-03-of-20": Object.freeze({
				"caseID": "http-physical-reducer-03-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-04-of-20": Object.freeze({
				"caseID": "http-physical-reducer-04-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-05-of-20": Object.freeze({
				"caseID": "http-physical-reducer-05-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-06-of-20": Object.freeze({
				"caseID": "http-physical-reducer-06-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-07-of-20": Object.freeze({
				"caseID": "http-physical-reducer-07-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-08-of-20": Object.freeze({
				"caseID": "http-physical-reducer-08-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-09-of-20": Object.freeze({
				"caseID": "http-physical-reducer-09-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-10-of-20": Object.freeze({
				"caseID": "http-physical-reducer-10-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-11-of-20": Object.freeze({
				"caseID": "http-physical-reducer-11-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-12-of-20": Object.freeze({
				"caseID": "http-physical-reducer-12-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-13-of-20": Object.freeze({
				"caseID": "http-physical-reducer-13-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-14-of-20": Object.freeze({
				"caseID": "http-physical-reducer-14-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-15-of-20": Object.freeze({
				"caseID": "http-physical-reducer-15-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-16-of-20": Object.freeze({
				"caseID": "http-physical-reducer-16-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-17-of-20": Object.freeze({
				"caseID": "http-physical-reducer-17-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-18-of-20": Object.freeze({
				"caseID": "http-physical-reducer-18-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-19-of-20": Object.freeze({
				"caseID": "http-physical-reducer-19-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"http-physical-reducer-20-of-20": Object.freeze({
				"caseID": "http-physical-reducer-20-of-20",
				"count": 1,
				"expected": Object.freeze([
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence$",
			}),
			"parity-evaluator-20": Object.freeze({
				"caseID": "parity-evaluator-20",
				"count": 20,
				"expected": Object.freeze([
					"TestGoAndNodeParityEvaluatorsMatchLiteralOracle",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/emit/node/parity",
				"profile": "general",
				"qualification": true,
				"run": "^TestGoAndNodeParityEvaluatorsMatchLiteralOracle$",
			}),
			"parity-framing-20": Object.freeze({
				"caseID": "parity-framing-20",
				"count": 20,
				"expected": Object.freeze([
					"TestNodeParityRunnerAcceptsExactWholeWireCap",
					"TestNodeParityRunnerRejectsInvalidFramesAtomically",
					"TestNodeParityRunnerRejectsMissingFinalLFWithoutStderr",
					"TestParityFramingRejectsExpandedSemanticResultAtomically",
					"TestParityResponseFramingExactBodyBoundary",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/emit/node/parity",
				"profile": "general",
				"qualification": true,
				"run": "^(?:TestNodeParityRunnerAcceptsExactWholeWireCap|TestNodeParityRunnerRejectsInvalidFramesAtomically|TestNodeParityRunnerRejectsMissingFinalLFWithoutStderr|TestParityFramingRejectsExpandedSemanticResultAtomically|TestParityResponseFramingExactBodyBoundary)$",
			}),
			"parity-full-package-3": Object.freeze({
				"caseID": "parity-full-package-3",
				"count": 3,
				"expected": Object.freeze([
					"FuzzParseContractParityCorpusLine",
					"TestContractParityCorpus",
					"TestContractParityCorpusExactSizeBoundaries",
					"TestContractParityCorpusIsOrderAndOracleIndependent",
					"TestContractParityCorpusParserRejectsEnvelopeAliases",
					"TestContractParityCorpusRejectsClosedSchemaDrift",
					"TestContractParityManifestMatchesCopiedEntrypointParser",
					"TestContractParityOracleLeakMutantsAreKilledByRequestRoster",
					"TestDirectResultSelectorExhaustiveGoNodeMatrix",
					"TestGoAndNodeParityEvaluatorsMatchLiteralOracle",
					"TestNodeParityRunnerAcceptsExactWholeWireCap",
					"TestNodeParityRunnerRejectsInvalidFramesAtomically",
					"TestNodeParityRunnerRejectsMissingFinalLFWithoutStderr",
					"TestOwnerEligibilitySelectorExhaustiveGoNodeMatrix",
					"TestParityFramingRejectsExpandedSemanticResultAtomically",
					"TestParityResponseFramingExactBodyBoundary",
					"TestResultFrameBytesExactBoundaries",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/internal/emit/node/parity",
				"profile": "general",
				"qualification": true,
				"run": "^(?:Test|Fuzz|Example).*$",
			}),
			"cli-physical-full-package-3": Object.freeze({
				"caseID": "cli-physical-full-package-3",
				"count": 3,
				"expected": Object.freeze([
					"TestCLICompilationFreshProcessRestartHelper",
					"TestCLIMaterializationControlCrossesAdapterAndExcludedMap",
					"TestCLIP07BBPublicationHelper",
					"TestCLIPhysicalReducerBudgetFenceRetainsOnlyBestKnown",
					"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
					"TestCLIPresentBytesStdinCrossesWorldAndAdapter",
					"TestCLIPresentEmptyStdinCrossesWorldAndAdapter",
					"TestCLIReferenceControlClassificationMatrix",
					"TestCLIReferenceDisplayPermutationPreservesMapWithFreshAttempts",
					"TestCLIReferencePrecedenceStudy",
					"TestCLIStudyUsesStrictSourceCompilerAndPlanBoundSchedule",
					"TestOptionalReceiptMeasurementKeepsAbsenceExplicit",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
				"profile": "sensitive",
				"qualification": true,
				"run": "^(?:Test|Fuzz|Example).*$",
			}),
			"http-physical-full-package-3": Object.freeze({
				"caseID": "http-physical-full-package-3",
				"count": 3,
				"expected": Object.freeze([
					"TestHTTPCompilationFreshProcessRestartHelper",
					"TestHTTPInvoiceAlternatingCandidateUsesAllTrialsNeverMajority",
					"TestHTTPInvoiceOutcomeMapUsesExactCandidateLabelsNotGroupShape",
					"TestHTTPInvoicePermutationPreservesExactMapWithFreshEvidence",
					"TestHTTPInvoicePortableChildBindPhysicalLineage",
					"TestHTTPInvoiceReferenceStudy",
					"TestHTTPInvoiceRequestFactsDrivePhysicalPolicy",
					"TestHTTPInvoiceStudyRecordsTimingAndTrialMultiplication",
					"TestHTTPInvoiceStudyUsesStrictSourceCompilerAndPlanBoundSchedule",
					"TestHTTPInvoiceTenantSeedShapeTrapIsPhysicalAndLabelSensitive",
					"TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
					"TestHTTPPhysicalTenantSeedNeighborChangesExactLabeledMapWithStableRoster",
					"TestNegativeSharedRootContaminationCreatesFalseEquality",
				]),
				"packagePath": "github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
				"profile": "sensitive",
				"qualification": true,
				"run": "^(?:Test|Fuzz|Example).*$",
			}),
		}),
		"digest": "08db7c338eb3ba900cf6df2545c04f27bed16889c81dd16e5fb18197c4d52958",
	}),
});
export const qualificationCaseIDs = qualificationMatrices.C4.caseIDs;
export const qualificationCases = qualificationMatrices.C4.cases;

export function qualificationMatrixDigest(boundary = "C4") {
	if (boundary === "C4" || boundary === "C5") return qualificationMatrices[boundary].digest;
	throw new Error("GO_REPETITION_QUALIFICATION_MATRIX");
}

export function qualificationCaseForID(caseID) {
	if (Object.hasOwn(qualificationMatrices.C4.cases, caseID)) return qualificationMatrices.C4.cases[caseID];
	if (Object.hasOwn(qualificationMatrices.C5.cases, caseID)) return qualificationMatrices.C5.cases[caseID];
	throw new Error("GO_REPETITION_QUALIFICATION_CASE");
}

export function parseRunArguments(argv) {
	if (argv.length === 2 && argv[0] === "--case") {
		return qualificationCaseForID(argv[1]);
	}
	if (argv.length < 10 || argv.length % 2 !== 0 || argv[0] !== "--package" || argv[2] !== "--profile" ||
		argv[4] !== "--count" || argv[6] !== "--run") {
		throw new Error("GO_REPETITION_ARGUMENTS");
	}
	throw new Error("GO_REPETITION_ARGUMENTS");
}

export function validateExecutionSpecification(specification) {
	if (typeof specification.caseID === "string") {
		const canonical = qualificationCaseForID(specification.caseID);
		if (canonical !== specification) {
			throw new Error("GO_REPETITION_SPECIFICATION");
		}
		return canonical;
	}
	throw new Error("GO_REPETITION_SPECIFICATION");
}

export function buildGoTestArguments(specification) {
	const jobs = specification.profile === "general" ? 2 : 1;
	return Object.freeze([
		"test", "-json", "-mod=readonly", "-buildvcs=false", `-p=${jobs}`, "-parallel=2",
		`-count=${specification.count}`, "-timeout=20m", "-run", specification.run, specification.packagePath,
	]);
}

export function assertGoTestJSON(stdout, specification) {
	if (typeof stdout !== "string" || stdout.length === 0 || !stdout.endsWith("\n") || stdout.includes("\r") || stdout.includes("\0")) {
		fail("GO_REPETITION_JSON_FRAMING", "stdout must be nonempty LF-delimited JSON");
	}
	const events = stdout.slice(0, -1).split("\n").map((line, index) => {
		try {
			const event = JSON.parse(line);
			if (!event || typeof event !== "object" || Array.isArray(event)) throw new Error("not an object");
			return event;
		} catch (error) {
			fail("GO_REPETITION_JSON_FRAMING", `line ${index + 1}: ${error.message}`);
		}
	});
	const expected = new Set(specification.expected);
	const states = new Map(specification.expected.map((name) => [name, { pass: 0, phase: "idle", run: 0, top: true }]));
	let packagePass = 0;
	let packageStart = 0;
	if (events[0]?.Action !== "start" || events[0]?.Test !== undefined ||
		events.at(-1)?.Action !== "pass" || events.at(-1)?.Test !== undefined) {
		fail("GO_REPETITION_PACKAGE_LIFECYCLE", "package start must be first and package pass must be last");
	}
	for (const [index, event] of events.entries()) {
		if (event.Package !== specification.packagePath) fail("GO_REPETITION_PACKAGE_EVENT", `line ${index + 1}: ${String(event.Package)}`);
		if (typeof event.Action !== "string" || !allowedActions.has(event.Action)) {
			fail("GO_REPETITION_ACTION", `line ${index + 1}: ${String(event.Action)}`);
		}
		if (event.Action === "fail") fail("GO_REPETITION_FAIL_EVENT", `line ${index + 1}: ${event.Test ?? "package"}`);
		if (event.Action === "skip") fail("GO_REPETITION_SKIP_EVENT", `line ${index + 1}: ${event.Test ?? "package"}`);
		if (event.Test === undefined) {
			if (!packageActions.has(event.Action)) fail("GO_REPETITION_PACKAGE_ACTION", `line ${index + 1}: ${event.Action}`);
			if (event.Action === "start") packageStart += 1;
			if (event.Action === "pass") packagePass += 1;
			continue;
		}
		if (typeof event.Test !== "string" || event.Test.length === 0) fail("GO_REPETITION_TEST_EVENT", `line ${index + 1}`);
		if (!testActions.has(event.Action)) fail("GO_REPETITION_TEST_ACTION", `line ${index + 1}: ${event.Action}`);
		const top = event.Test.split("/", 1)[0];
		if (!expected.has(top)) fail("GO_REPETITION_FOREIGN_TEST", `${event.Test}`);
		const topState = states.get(top);
		if (event.Test !== top && topState.phase !== "active") {
			fail("GO_REPETITION_PARENT_LIFECYCLE", `${event.Test}: top ${top} is ${topState.phase} at line ${index + 1}`);
		}
		let state = states.get(event.Test);
		if (!state) {
			state = { pass: 0, phase: "idle", run: 0, top: false };
			states.set(event.Test, state);
		}
		if (event.Action === "run") {
			if (state.phase !== "idle") fail("GO_REPETITION_TEST_LIFECYCLE", `${event.Test}: overlapping run at line ${index + 1}`);
			state.phase = "active";
			state.run += 1;
		} else if (event.Action === "pause") {
			if (state.phase !== "active") fail("GO_REPETITION_TEST_LIFECYCLE", `${event.Test}: pause from ${state.phase} at line ${index + 1}`);
			state.phase = "paused";
		} else if (event.Action === "cont") {
			if (state.phase !== "paused") fail("GO_REPETITION_TEST_LIFECYCLE", `${event.Test}: cont from ${state.phase} at line ${index + 1}`);
			state.phase = "active";
		} else if (event.Action === "output") {
			if (!(state.phase === "active" || state.phase === "paused")) {
				fail("GO_REPETITION_TEST_LIFECYCLE", `${event.Test}: output from ${state.phase} at line ${index + 1}`);
			}
		} else if (event.Action === "pass") {
			if (state.phase !== "active") fail("GO_REPETITION_TEST_LIFECYCLE", `${event.Test}: pass from ${state.phase} at line ${index + 1}`);
			if (event.Test === top) {
				const activeDescendant = [...states.entries()].find(([name, candidate]) =>
					name.startsWith(`${top}/`) && candidate.phase !== "idle",
				);
				if (activeDescendant) {
					fail("GO_REPETITION_PARENT_LIFECYCLE", `${top}: descendant ${activeDescendant[0]} is ${activeDescendant[1].phase} at line ${index + 1}`);
				}
			}
			state.phase = "idle";
			state.pass += 1;
		}
	}
	if (packageStart !== 1 || packagePass !== 1) fail("GO_REPETITION_PACKAGE_COUNTS", `start=${packageStart} pass=${packagePass}`);
	for (const [name, state] of states) {
		if (state.phase !== "idle" || state.run !== state.pass) {
			fail("GO_REPETITION_TEST_LIFECYCLE", `${name}: phase=${state.phase} run=${state.run} pass=${state.pass}`);
		}
		if (state.top && (state.run !== specification.count || state.pass !== specification.count)) {
			fail("GO_REPETITION_TEST_COUNTS", `${name}: run=${state.run} pass=${state.pass} expected=${specification.count}`);
		}
	}
	return Object.freeze({ events: events.length, tests: specification.expected.length, repetitions: specification.count });
}

function authorityRoster(admitted) {
	const entries = authorityNames.map((name) => {
		const authority = admitted[name];
		if (!authority || typeof authority.path !== "string" || !/^[0-9a-f]{64}$/u.test(authority.sha256)) {
			fail("GO_REPETITION_AUTHORITY", name);
		}
		return Object.freeze({ name, path: authority.path, sha256: authority.sha256 });
	});
	const digest = createHash("sha256").update(JSON.stringify(entries)).digest("hex");
	return Object.freeze({ digest, entries: Object.freeze(entries) });
}

function authorityText(roster) {
	return roster.entries.map((entry) =>
		`GO_REPETITION_AUTHORITY name=${entry.name} path=${JSON.stringify(entry.path)} sha256:${entry.sha256}\n`,
	).join("");
}

function encoded(events) {
	return `${events.map((event) => JSON.stringify(event)).join("\n")}\n`;
}

function cleanEvents(specification) {
	const events = [{ Action: "start", Package: specification.packagePath }];
	for (let repetition = 0; repetition < specification.count; repetition += 1) {
		for (const name of specification.expected) {
			events.push({ Action: "run", Package: specification.packagePath, Test: name });
			if (name === specification.expected[0]) {
				events.push({ Action: "run", Package: specification.packagePath, Test: `${name}/child` });
				events.push({ Action: "pause", Package: specification.packagePath, Test: `${name}/child` });
				events.push({ Action: "cont", Package: specification.packagePath, Test: `${name}/child` });
				events.push({ Action: "pass", Package: specification.packagePath, Test: `${name}/child` });
			}
			events.push({ Action: "pass", Package: specification.packagePath, Test: name });
		}
	}
	events.push({ Action: "pass", Package: specification.packagePath });
	return events;
}

function expectCode(invoke, code) {
	try {
		invoke();
	} catch (error) {
		if ((error instanceof GoRepetitionError && error.code === code) || error.message === code) return;
		throw error;
	}
	fail("GO_REPETITION_SELFTEST_FALSE_NEGATIVE", code);
}

async function expectAsyncFailure(invoke, predicate, detail) {
	try {
		await invoke();
	} catch (error) {
		if (predicate(error)) return;
		throw error;
	}
	fail("GO_REPETITION_SELFTEST_FALSE_NEGATIVE", detail);
}

export async function runRepetition(specification, dependencies = {}) {
	const executionSpecification = validateExecutionSpecification(specification);
	if ((dependencies.platform ?? platform()) !== "darwin" || (dependencies.arch ?? arch()) !== "arm64") {
		fail("GO_REPETITION_PLATFORM", `${dependencies.platform ?? platform()}/${dependencies.arch ?? arch()}`);
	}
	const acquire = dependencies.acquireVerificationLock ?? acquireVerificationLock;
	const executeChild = dependencies.childResult ?? childResult;
	const admit = dependencies.admitTools ?? admitTools;
	const createRoots = dependencies.createPrivateRoots ?? createPrivateRoots;
	const makeEnvironment = dependencies.buildChildEnvironment ?? buildChildEnvironment;
	const finalize = dependencies.finalizeVerificationResources ?? finalizeVerificationResources;
	const remove = dependencies.remove ?? cleanupVerificationResources;
	const write = dependencies.write ?? ((value) => process.stdout.write(value));
	const lock = await acquire();
	let roots;
	let finalized = false;
	let failure;
	try {
		const admitted = await admit();
		roots = await createRoots(repositoryRoot, admitted);
		const childEnvironment = makeEnvironment(admitted, roots);
		const authorities = authorityRoster(admitted);
		write(authorityText(authorities));
		const step = Object.freeze({ id: "go-test-repetition", tool: "go", tools: authorityNames });
		const args = buildGoTestArguments(executionSpecification);
		const result = await executeChild(step, admitted, childEnvironment, { args });
		if (result.error || result.signal || result.status !== 0 || result.stderr !== "") {
			const detail = `${result.error?.message ?? ""} signal=${result.signal ?? "none"} status=${result.status} ` +
				`stderr=${JSON.stringify(result.stderr.slice(-4096))} stdout_tail=${JSON.stringify(result.stdout.slice(-4096))}`;
			fail("GO_REPETITION_CHILD", detail);
		}
		const summary = assertGoTestJSON(result.stdout, executionSpecification);
		await finalize(lock, roots.runRoot);
		finalized = true;
		write(
			`Go repetition verification passed: case=${executionSpecification.caseID ?? "AD_HOC"} qualification=${executionSpecification.caseID !== null} ` +
			`package=${executionSpecification.packagePath} profile=${executionSpecification.profile} count=${summary.repetitions} ` +
			`tests=${executionSpecification.expected.join(",")} events=${summary.events} authorities_sha256:${authorities.digest}\n`,
		);
	} catch (error) {
		failure = error;
	} finally {
		if (!finalized) {
			try {
				await remove(lock, roots);
			} catch (error) {
				failure = new AggregateError([...(failure ? [failure] : []), error], "Go repetition verification cleanup failed");
			}
		}
	}
	if (failure) throw failure;
}

async function compositionSelfTest(specification, cleanStdout) {
	const calls = [];
	const output = [];
	let released = false;
	const lock = {
		async assertHeld() { calls.push("lock.assertHeld"); if (released) throw new Error("released"); },
		async release() { calls.push("lock.release"); released = true; },
	};
	const admitted = Object.fromEntries(authorityNames.map((name, index) => [name, {
		path: `/authority/${name}`,
		sha256: String(index + 1).padStart(64, "0"),
	}]));
	const roots = { runRoot: "/private/run" };
	let capturedArgs;
	await runRepetition(specification, {
		platform: "darwin",
		arch: "arm64",
		async acquireVerificationLock() { calls.push("lock.acquire"); return lock; },
		async admitTools() { calls.push("tools.admit"); return admitted; },
		async createPrivateRoots(root, actual) {
			calls.push("roots.create");
			if (root !== repositoryRoot || actual !== admitted) throw new Error("root/admitted drift");
			return roots;
		},
		buildChildEnvironment(actual, actualRoots) {
			calls.push("environment.build");
			if (actual !== admitted || actualRoots !== roots) throw new Error("environment authority drift");
			return Object.freeze({ PRIVATE: "1" });
		},
		async childResult(step, actual, environment, options) {
			calls.push("child.execute");
			if (step.tool !== "go" || JSON.stringify(step.tools) !== JSON.stringify(authorityNames) || actual !== admitted ||
				environment.PRIVATE !== "1") throw new Error("child composition drift");
			capturedArgs = options.args;
			return { status: 0, signal: null, error: null, stdout: cleanStdout, stderr: "" };
		},
		async finalizeVerificationResources(actualLock, runRoot) {
			calls.push("finalize.begin");
			if (actualLock !== lock || runRoot !== roots.runRoot) throw new Error("finalization authority drift");
			await actualLock.assertHeld();
			calls.push("root.remove");
			await actualLock.release();
			calls.push("finalize.end");
		},
		write(value) { output.push(value); },
	});
	const expectedCalls = [
		"lock.acquire", "tools.admit", "roots.create", "environment.build", "child.execute", "finalize.begin",
		"lock.assertHeld", "root.remove", "lock.release", "finalize.end",
	];
	if (JSON.stringify(calls) !== JSON.stringify(expectedCalls) ||
		JSON.stringify(capturedArgs) !== JSON.stringify(buildGoTestArguments(specification))) {
		fail("GO_REPETITION_SELFTEST_COMPOSITION", JSON.stringify({ calls, capturedArgs }));
	}
	const joined = output.join("");
	let cursor = -1;
	for (const name of authorityNames) {
		const next = joined.indexOf(`GO_REPETITION_AUTHORITY name=${name} `);
		if (next <= cursor) fail("GO_REPETITION_SELFTEST_AUTHORITY_ORDER", name);
		cursor = next;
	}
	const success = joined.indexOf("Go repetition verification passed:");
	if (success <= cursor || !joined.includes("qualification=true") || !joined.includes("authorities_sha256:")) {
		fail("GO_REPETITION_SELFTEST_SUCCESS_ORDER", joined);
	}

	for (const [name, child, finalizeFailure, expectedCleanup] of [
		["child", { status: 1, signal: null, error: null, stdout: "", stderr: "" }, false, ["remove", "release"]],
		["parser", { status: 0, signal: null, error: null, stdout: "bad\n", stderr: "" }, false, ["remove", "release"]],
		["finalize", { status: 0, signal: null, error: null, stdout: cleanStdout, stderr: "" }, true, ["finalize", "remove", "release"]],
	]) {
		const cleanup = [];
		const failureLock = { async release() { cleanup.push("release"); } };
		const failureOutput = [];
		await expectAsyncFailure(() => runRepetition(specification, {
			platform: "darwin",
			arch: "arm64",
			async acquireVerificationLock() { return failureLock; },
			async admitTools() { return admitted; },
			async createPrivateRoots() { return roots; },
			buildChildEnvironment() { return {}; },
			async childResult() { return child; },
			async finalizeVerificationResources() {
				cleanup.push("finalize");
				if (finalizeFailure) throw new Error("injected finalization failure");
			},
			async remove(actualLock, actualRoots) {
				if (actualRoots !== roots) throw new Error("cleanup roots drift");
				cleanup.push("remove");
				await actualLock.release();
			},
			write(value) { failureOutput.push(value); },
		}), () => true, name);
		if (JSON.stringify(cleanup) !== JSON.stringify(expectedCleanup) ||
			failureOutput.join("").includes("Go repetition verification passed:")) {
			fail("GO_REPETITION_SELFTEST_FAILURE_CLEANUP", `${name}: ${JSON.stringify(cleanup)}`);
		}
	}

	await expectAsyncFailure(() => runRepetition(specification, {
		platform: "darwin",
		arch: "arm64",
		async acquireVerificationLock() { return {}; },
		async admitTools() { return admitted; },
		async createPrivateRoots() { return roots; },
		buildChildEnvironment() { return {}; },
		async childResult() { return { status: 1, signal: null, error: null, stdout: "", stderr: "" }; },
		async remove() {
			throw new AggregateError([new Error("remove failed"), new Error("release failed")], "cleanup failed");
		},
		write() {},
	}), (error) => error instanceof AggregateError && error.errors.length === 2 &&
		error.errors[1] instanceof AggregateError && error.errors[1].errors.length === 2, "aggregate cleanup failure");
}

export async function dormantQualificationSelfTest(caseID) {
	const parsed = parseRunArguments(["--case", caseID]);
	const canonical = validateExecutionSpecification(parsed);
	const args = buildGoTestArguments(canonical);
	if (!Object.isFrozen(args)) {
		throw new Error("dormant qualification argv is not frozen");
	}
	await runRepetition(canonical, {
		platform: "darwin",
		arch: "arm64",
		acquireVerificationLock: async () => Object.freeze({}),
		admitTools: async () => Object.freeze({
			go: Object.freeze({ path: "/authority/go", sha256: "0000000000000000000000000000000000000000000000000000000000000001" }),
			node: Object.freeze({ path: "/authority/node", sha256: "0000000000000000000000000000000000000000000000000000000000000002" }),
			git: Object.freeze({ path: "/authority/git", sha256: "0000000000000000000000000000000000000000000000000000000000000003" }),
			sh: Object.freeze({ path: "/authority/sh", sha256: "0000000000000000000000000000000000000000000000000000000000000004" }),
			cc: Object.freeze({ path: "/authority/cc", sha256: "0000000000000000000000000000000000000000000000000000000000000005" }),
			cxx: Object.freeze({ path: "/authority/cxx", sha256: "0000000000000000000000000000000000000000000000000000000000000006" }),
		}),
		createPrivateRoots: async () => Object.freeze({ runRoot: "/private/dormant" }),
		buildChildEnvironment: () => Object.freeze({}),
		childResult: async () => {
			return Object.freeze({ status: 0, signal: null, error: null, stdout: encoded(cleanEvents(canonical)), stderr: "" });
		},
		finalizeVerificationResources: async () => {},
		remove: async () => {},
		write: () => {},
	});
}

async function selfTest() {
	const specification = Object.freeze({
		caseID: null,
		count: 2,
		expected: Object.freeze(["TestAlpha", "TestBeta"]),
		packagePath: `${modulePath}/internal/example`,
		profile: "general",
		qualification: false,
		run: "^(?:TestAlpha|TestBeta)$",
	});
	const clean = cleanEvents(specification);
	const cleanText = encoded(clean);
	const summary = assertGoTestJSON(cleanText, specification);
	if (summary.tests !== 2 || summary.repetitions !== 2) fail("GO_REPETITION_SELFTEST_CLEAN", JSON.stringify(summary));
	const beforeTerminal = (event) => encoded([...clean.slice(0, -1), event, clean.at(-1)]);
	const insideTop = (event) => encoded([
		{ Action: "start", Package: specification.packagePath },
		{ Action: "run", Package: specification.packagePath, Test: "TestAlpha" },
		event,
		{ Action: "pass", Package: specification.packagePath, Test: "TestAlpha" },
		{ Action: "pass", Package: specification.packagePath },
	]);
	const childBeforeParent = encoded([
		{ Action: "start", Package: specification.packagePath },
		{ Action: "run", Package: specification.packagePath, Test: "TestAlpha/early" },
		{ Action: "pass", Package: specification.packagePath, Test: "TestAlpha/early" },
		...clean.slice(1),
	]);
	const parentBeforeChild = encoded([
		{ Action: "start", Package: specification.packagePath },
		{ Action: "run", Package: specification.packagePath, Test: "TestAlpha" },
		{ Action: "run", Package: specification.packagePath, Test: "TestAlpha/late" },
		{ Action: "pass", Package: specification.packagePath, Test: "TestAlpha" },
		{ Action: "pass", Package: specification.packagePath, Test: "TestAlpha/late" },
		{ Action: "pass", Package: specification.packagePath },
	]);
	for (const [name, value, code] of [
		["missing-pass", encoded(clean.filter((event, index) => !(event.Action === "pass" && event.Test === "TestBeta" && index > 8))), "GO_REPETITION_TEST_LIFECYCLE"],
		["test-fail", beforeTerminal({ Action: "fail", Package: specification.packagePath, Test: "TestAlpha" }), "GO_REPETITION_FAIL_EVENT"],
		["package-fail", beforeTerminal({ Action: "fail", Package: specification.packagePath }), "GO_REPETITION_FAIL_EVENT"],
		["skip", beforeTerminal({ Action: "skip", Package: specification.packagePath, Test: "TestAlpha" }), "GO_REPETITION_SKIP_EVENT"],
		["foreign-test", beforeTerminal({ Action: "run", Package: specification.packagePath, Test: "TestForeign" }), "GO_REPETITION_FOREIGN_TEST"],
		["foreign-package", encoded(clean.map((event, index) => index === 0 ? { ...event, Package: `${modulePath}/foreign` } : event)), "GO_REPETITION_PACKAGE_EVENT"],
		["package-run", beforeTerminal({ Action: "run", Package: specification.packagePath }), "GO_REPETITION_PACKAGE_ACTION"],
		["test-start", beforeTerminal({ Action: "start", Package: specification.packagePath, Test: "TestAlpha" }), "GO_REPETITION_TEST_ACTION"],
		["test-bench", beforeTerminal({ Action: "bench", Package: specification.packagePath, Test: "TestAlpha" }), "GO_REPETITION_TEST_ACTION"],
		["pause-without-run", insideTop({ Action: "pause", Package: specification.packagePath, Test: "TestAlpha/late" }), "GO_REPETITION_TEST_LIFECYCLE"],
		["cont-without-pause", insideTop({ Action: "cont", Package: specification.packagePath, Test: "TestAlpha/late" }), "GO_REPETITION_TEST_LIFECYCLE"],
		["subtest-pass-without-run", insideTop({ Action: "pass", Package: specification.packagePath, Test: "TestAlpha/late" }), "GO_REPETITION_TEST_LIFECYCLE"],
		["subtest-output-without-run", insideTop({ Action: "output", Package: specification.packagePath, Test: "TestAlpha/late" }), "GO_REPETITION_TEST_LIFECYCLE"],
		["child-before-parent", childBeforeParent, "GO_REPETITION_PARENT_LIFECYCLE"],
		["parent-before-child-terminal", parentBeforeChild, "GO_REPETITION_PARENT_LIFECYCLE"],
		["invalid-json", "not-json\n", "GO_REPETITION_JSON_FRAMING"],
		["missing-final-lf", cleanText.slice(0, -1), "GO_REPETITION_JSON_FRAMING"],
		["trailing-package-output", encoded([...clean, { Action: "output", Package: specification.packagePath, Output: "late\n" }]), "GO_REPETITION_PACKAGE_LIFECYCLE"],
	]) {
		expectCode(() => assertGoTestJSON(value, specification), code);
		if (!name) fail("GO_REPETITION_SELFTEST_CASE", code);
	}
	if (parseRunArguments(["--case", "parity-full-package-3"]) !== qualificationCases["parity-full-package-3"] ||
		!qualificationCases["parity-full-package-3"].expected.includes("FuzzParseContractParityCorpusLine")) {
		fail("GO_REPETITION_SELFTEST_QUALIFICATION_CASE", "parity-full-package-3");
	}
	const cliPhysicalShards = qualificationCaseIDs.filter((id) => /^cli-physical-reducer-[0-9]{2}-of-20$/u.test(id));
	const httpPhysicalShards = qualificationCaseIDs.filter((id) => /^http-physical-reducer-[0-9]{2}-of-20$/u.test(id));
	if (qualificationCaseIDs.length !== 54 || qualificationMatrices.C5.caseIDs.length !== 56 ||
		qualificationMatrixDigest() !== "3be703fae10155c83b19be54ae7cd79cfc5b1bbaddfcf86bb2c9781ad93f853a" ||
		qualificationMatrixDigest("C5") !== "08db7c338eb3ba900cf6df2545c04f27bed16889c81dd16e5fb18197c4d52958" ||
		cliPhysicalShards.length !== 20 || httpPhysicalShards.length !== 20 ||
		[...cliPhysicalShards, ...httpPhysicalShards].some((id) => qualificationCases[id].count !== 1)) {
		fail("GO_REPETITION_SELFTEST_QUALIFICATION_SHARDS", JSON.stringify({ cliPhysicalShards, httpPhysicalShards }));
	}
	expectCode(() => parseRunArguments(["--case", "unknown"]), "GO_REPETITION_QUALIFICATION_CASE");
	expectCode(() => parseRunArguments([
		"--package", `${modulePath}/internal/world`, "--profile", "general", "--count", "2",
		"--run", "^TestAlpha$", "--expect", "TestAlpha",
	]), "GO_REPETITION_ARGUMENTS");
	await expectAsyncFailure(() => runRepetition({ ...specification, qualification: true }, {
		platform: "darwin",
		arch: "arm64",
		async acquireVerificationLock() { throw new Error("forged qualification reached lock acquisition"); },
	}), (error) => error.message === "GO_REPETITION_SPECIFICATION", "forged ad hoc qualification");
	await expectAsyncFailure(() => runRepetition({ ...qualificationCases["processmechanics-output-caps-50"], count: 1 }, {
		platform: "darwin",
		arch: "arm64",
		async acquireVerificationLock() { throw new Error("mutated qualification reached lock acquisition"); },
	}), (error) => error.message === "GO_REPETITION_SPECIFICATION", "mutated named qualification");
	const compositionSpecification = qualificationCases["parity-evaluator-20"];
	await compositionSelfTest(compositionSpecification, encoded(cleanEvents(compositionSpecification)));
	await dormantQualificationSelfTest("contract-http-readiness-50");
	await dormantQualificationSelfTest("contract-http-teardown-20");
	process.stdout.write(
		`Go repetition verifier self-test passed: matrix_sha256:${qualificationMatrixDigest()} exact JSON/state framing, ` +
		"profile classification, Test/Fuzz/Example rosters, authority binding, exact child composition, finalization ordering, and failure cleanup\n",
	);
}

async function main() {
	if (process.argv.length === 3 && process.argv[2] === "--self-test") {
		await selfTest();
		return;
	}
	await runRepetition(parseRunArguments(process.argv.slice(2)));
}

if (process.argv[1] !== undefined && pathToFileURL(process.argv[1]).href === import.meta.url) {
	await main();
}
