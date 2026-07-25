#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { readFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import {
	collectC3PredecessorAuthority,
	collectC3PredecessorRecordAuthority,
	collectC2Facts,
	collectC3Facts,
	collectC4Facts,
	collectFacts,
	goJSONArguments,
	inspectC3DidrunNote,
	parseC4FinalRunbookClaimMap,
	parseC4StatusClaimMap,
	productionStoreOwnerReferenceCount,
	validateC2Facts,
	validateC3Facts,
	validateC4Facts,
	validateFacts,
	validateGoJSONTranscript,
} from "./check-p07b-c-architecture.mjs";

const selftestPath = fileURLToPath(import.meta.url);
const root = resolve(dirname(selftestPath), "..");
const checker = resolve(root, "tools/check-p07b-c-architecture.mjs");
const cases = Object.freeze([
	Object.freeze({ id: "production-files", code: "P07B_C1_PRODUCTION_TOPOLOGY" }),
	Object.freeze({ id: "test-files", code: "P07B_C1_TEST_TOPOLOGY" }),
	Object.freeze({ id: "test-file-rename", code: "P07B_C1_TEST_TOPOLOGY" }),
	Object.freeze({ id: "test-import", code: "P07B_C1_TEST_IMPORT_ROSTER" }),
	Object.freeze({ id: "test-symbol", code: "P07B_C1_TEST_SYMBOL_ROSTER" }),
	Object.freeze({ id: "model-extra-entry", code: "P07B_C1_MODEL_TOPOLOGY" }),
	Object.freeze({ id: "production-import", code: "P07B_C1_PRODUCTION_IMPORT_ROSTER" }),
	Object.freeze({ id: "dependency-closure", code: "P07B_C1_DEPENDENCY_CLOSURE" }),
	Object.freeze({ id: "ignored-production", code: "P07B_C1_PRODUCTION_TOPOLOGY" }),
	Object.freeze({ id: "foreign-build-input", code: "P07B_C1_PRODUCTION_TOPOLOGY" }),
	Object.freeze({ id: "premature-importer", code: "P07B_C1_IMPORTER_ROSTER" }),
	Object.freeze({ id: "schema-closure", code: "P07B_C1_SCHEMA_CLOSURE" }),
	Object.freeze({ id: "schema-required-roster", code: "P07B_C1_SCHEMA_REQUIRED_ROSTER" }),
	Object.freeze({ id: "target-root", code: "P07B_C1_OBJECT_ROSTER" }),
	Object.freeze({ id: "serialized-disposition", code: "P07B_C1_OBJECT_ROSTER" }),
	Object.freeze({ id: "result-roster", code: "P07B_C1_ENUM_ROSTER" }),
	Object.freeze({ id: "scope-domain", code: "P07B_C1_SCOPE_PROFILE" }),
	Object.freeze({ id: "limit-profile", code: "P07B_C1_LIMIT_PROFILE" }),
	Object.freeze({ id: "evidence-roster", code: "P07B_C1_EXAMPLE_REFERENCE_ROSTER" }),
	Object.freeze({ id: "example-graph", code: "P07B_C1_EXAMPLE_GRAPH" }),
	Object.freeze({ id: "c0-authority", code: "P07B_C1_C0_AUTHORITY" }),
	Object.freeze({ id: "topology", code: "P07B_C1_PRODUCTION_TOPOLOGY" }),
]);
const expectedRosterDigest = "061b6b305f56685e757864b6ed5fda600819a8ffa3b1f5a90f924e1eb0547b3b";
const c2Cases = Object.freeze([
	Object.freeze({ id: "store-production-files", code: "P07B_C2_STORE_TOPOLOGY" }),
	Object.freeze({ id: "store-directory-entry", code: "P07B_C2_STORE_TOPOLOGY" }),
	Object.freeze({ id: "production-import", code: "P07B_C2_IMPORT_ROSTER" }),
	Object.freeze({ id: "compiler-production-import", code: "P07B_C2_IMPORT_ROSTER" }),
	Object.freeze({ id: "new-production-export", code: "P07B_C2_EXPORTED_SURFACE" }),
	Object.freeze({ id: "object-store-export", code: "P07B_C2_EXPORTED_SURFACE" }),
	Object.freeze({ id: "compiler-parsed-surface", code: "P07B_C2_EXPORTED_SURFACE" }),
	Object.freeze({ id: "authority-shape", code: "P07B_C2_AUTHORITY_SHAPE" }),
	Object.freeze({ id: "test-symbol", code: "P07B_C2_TEST_SYMBOL_ROSTER" }),
	Object.freeze({ id: "test-file-roster", code: "P07B_C2_TEST_FILE_ROSTER" }),
	Object.freeze({ id: "forbidden-surface", code: "P07B_C2_FORBIDDEN_PRODUCTION_SURFACE" }),
	Object.freeze({ id: "namespace-identity", code: "P07B_C2_NAMESPACE_IDENTITY" }),
	Object.freeze({ id: "namespace-case-alias", code: "P07B_C2_NAMESPACE_IDENTITY" }),
	Object.freeze({ id: "relation-constants", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "relation-effects", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "relation-persistence-order", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "relation-open-convergence", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "run-manifest-gate", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "execution-profile-derived", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "relation-identity-guards", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "relation-identity-tests", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "boot-join", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "acquisition-order", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "release-order", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "reset-order", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "winner-freshness", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "start-claim-convergence", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "clear-receipt-transition", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "clear-receipt-convergence", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "clear-receipt-faults", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "interlock-identity-boundary", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "seal-ownership", code: "P07B_C2_SEAL_OWNERSHIP" }),
	Object.freeze({ id: "private-limits", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "private-kind-roster", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "private-state-roster", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "manifest-closure", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "manifest-open-convergence", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "private-arithmetic", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "availability-convergence", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "purge-order", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "purge-create-count", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "fresh-purge-order", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "missing-pack-gate", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "canonical-isolation", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "private-identity-guards", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "private-identity-tests", code: "P07B_C2_PRIVATE_EVIDENCE" }),
]);
const expectedC2RosterDigest = "9cffa462cff50a197afe3733529a0eba6ebc6d29a5399ea33ee6d33019fd885f";
const c3Cases = Object.freeze([
	Object.freeze({ id: "package-topology", code: "P07B_C3_PACKAGE_TOPOLOGY" }),
	Object.freeze({ id: "build-tags", code: "P07B_C3_BUILD_TAG_ROSTER" }),
	Object.freeze({ id: "source-import", code: "P07B_C3_IMPORT_ROSTER" }),
	Object.freeze({ id: "exported-surface", code: "P07B_C3_EXPORTED_SURFACE" }),
	Object.freeze({ id: "test-file-roster", code: "P07B_C3_TEST_FILE_ROSTER" }),
	Object.freeze({ id: "inherited-c2", code: "P07B_C3_INHERITED_C2" }),
	Object.freeze({ id: "store-bridge", code: "P07B_C3_STORE_BRIDGE" }),
	Object.freeze({ id: "git-target", code: "P07B_C3_GIT_TARGET" }),
	Object.freeze({ id: "host-epoch", code: "P07B_C3_HOST_EPOCH" }),
	Object.freeze({ id: "node-runtime", code: "P07B_C3_NODE_RUNTIME" }),
	Object.freeze({ id: "official-target", code: "P07B_C3_OFFICIAL_TARGET" }),
	Object.freeze({ id: "predecessor", code: "P07B_C3_PREDECESSOR" }),
]);
const expectedC3RosterDigest = "d4e3f7a08fadd7e09022d7113aec267cc7dd66de41829a2bbe687d4c86350740";
const c4Cases = Object.freeze([
	Object.freeze({ id: "inherited-c3", code: "P07B_C4_INHERITED_C3" }),
	Object.freeze({ id: "package-topology", code: "P07B_C4_PACKAGE_TOPOLOGY" }),
	Object.freeze({ id: "build-tags", code: "P07B_C4_BUILD_TAG_ROSTER" }),
	Object.freeze({ id: "test-file-roster", code: "P07B_C4_TEST_FILE_ROSTER" }),
	Object.freeze({ id: "mechanics-authority", code: "P07B_C4_MECHANICS_AUTHORITY" }),
	Object.freeze({ id: "mechanics-importers", code: "P07B_C4_MECHANICS_IMPORTERS" }),
	Object.freeze({ id: "world-adapter", code: "P07B_C4_WORLD_ADAPTER" }),
	Object.freeze({ id: "runner-surface", code: "P07B_C4_RUNNER_SURFACE" }),
	Object.freeze({ id: "admission-adjacency", code: "P07B_C4_ADMISSION_ADJACENCY" }),
	Object.freeze({ id: "execution-chronology", code: "P07B_C4_EXECUTION_CHRONOLOGY" }),
	Object.freeze({ id: "terminal-graph", code: "P07B_C4_TERMINAL_GRAPH" }),
	Object.freeze({ id: "evidence-scope", code: "P07B_C4_EVIDENCE_SCOPE" }),
	Object.freeze({ id: "scope-root", code: "P07B_C4_SCOPE_ROOT" }),
	Object.freeze({ id: "profile-command", code: "P07B_C4_PROFILE_COMMAND" }),
]);
const expectedC4RosterDigest = "671e79b65d43aec47c3dd14531f3e64180392ac73482c6698e2e809534c51670";

function rosterDigest(roster = cases) {
	const hash = createHash("sha256");
	for (const entry of roster) hash.update(entry.id).update("\0").update(entry.code).update("\0");
	return hash.digest("hex");
}

function fail(code, detail) { throw new Error(`${code}: ${detail}`); }

function requireViolation(facts, code, id) {
	const problems = validateFacts(facts);
	if (!problems.some((problem) => problem.code === code)) {
		fail("P07B_C1_SELFTEST_FALSE_NEGATIVE", `${id}:${problems.map((problem) => problem.code).join(",")}`);
	}
}

function requireC2Violation(facts, code, id) {
	const problems = validateC2Facts(facts);
	if (!problems.some((problem) => problem.code === code)) {
		fail("P07B_C2_SELFTEST_FALSE_NEGATIVE", `${id}:${problems.map((problem) => problem.code).join(",")}`);
	}
}

function requireC3Violation(facts, code, id) {
	const problems = validateC3Facts(facts);
	if (!problems.some((problem) => problem.code === code)) {
		fail("P07B_C3_SELFTEST_FALSE_NEGATIVE", `${id}:${problems.map((problem) => problem.code).join(",")}`);
	}
}

function requireC4Violation(facts, code, id) {
	const problems = validateC4Facts(facts);
	if (!problems.some((problem) => problem.code === code)) {
		fail("P07B_C4_SELFTEST_FALSE_NEGATIVE", `${id}:${problems.map((problem) => problem.code).join(",")}`);
	}
}

function requireC4ParserRefusal(callback, id) {
	let refused = false;
	try {
		callback();
	} catch (error) {
		refused = error?.code === "P07B_C4_PROFILE_COMMAND";
	}
	if (!refused) fail("P07B_C4_SELFTEST_PARSER_FALSE_NEGATIVE", id);
}

function renderC4RunbookSource() {
	const result = spawnSync(process.execPath, [resolve(root, "tools/check-p07b-c-plan.mjs"), "--print-final-runbook", "C4"], {
		cwd: root,
		encoding: "utf8",
		timeout: 30_000,
		maxBuffer: 32 * 1024 * 1024,
		env: process.env,
	});
	if (result.error || result.signal || result.status !== 0 || result.stderr !== "") {
		fail("P07B_C4_SELFTEST_RUNBOOK_SOURCE", `${result.status ?? result.signal}: ${result.stderr || result.stdout || result.error}`);
	}
	return result.stdout;
}

function runCleanChecker(phase = "c1") {
	const args = phase === "c4" ? [checker, "--c4"] :
		phase === "c3" ? [checker, "--c3"] : phase === "c2" ? [checker, "--c2"] : [checker];
	const marker = phase === "c4" ? "P07B-C C4 cumulative architecture boundary OK" :
		phase === "c3" ? "P07B-C C3 cumulative architecture boundary OK" :
		phase === "c2" ? "P07B-C C2 cumulative architecture boundary OK" : "P07B-C C1 architecture boundary OK";
	const result = spawnSync(process.execPath, args, {
		cwd: root,
		encoding: "utf8",
		timeout: phase === "c4" ? 1_800_000 : phase === "c3" ? 1_200_000 : phase === "c2" ? 420_000 : 180_000,
		maxBuffer: 32 * 1024 * 1024,
		env: {
			...process.env,
			COUNTERSHAPE_GO: process.env.COUNTERSHAPE_GO ?? "/opt/homebrew/bin/go",
			GOMAXPROCS: "2",
			GOFLAGS: "-mod=readonly -buildvcs=false -p=1",
		},
	});
	if (result.error || result.signal || result.status !== 0 || result.stderr !== "" || result.stdout.trim() !== marker) {
		fail(phase === "c4" ? "P07B_C4_SELFTEST_CLEAN_CHECKER" :
			phase === "c3" ? "P07B_C3_SELFTEST_CLEAN_CHECKER" :
			phase === "c2" ? "P07B_C2_SELFTEST_CLEAN_CHECKER" : "P07B_C1_SELFTEST_CLEAN_CHECKER",
			`${result.status ?? result.signal}: ${result.stderr || result.stdout || result.error}`);
	}
}

function inspectGoJSONTranscriptParser() {
	const packagePath = "github.com/nelsonwerd/countershape/internal/contractexec/model";
	const first = "TestClassifierTruthTableUsesOnlyEligibleCompleteTupleMembership";
	const second = "TestClosedRunAlgebraExhaustiveCrossProduct";
	const clean = [
		{ Action: "start", Package: packagePath },
		{ Action: "run", Package: packagePath, Test: first },
		{ Action: "pass", Package: packagePath, Test: first },
		{ Action: "run", Package: packagePath, Test: second },
		{ Action: "pass", Package: packagePath, Test: second },
		{ Action: "pass", Package: packagePath },
	];
	const encode = (events) => Buffer.from(`${events.map((event) => JSON.stringify(event)).join("\n")}\n`, "utf8");
	validateGoJSONTranscript("exhaustive-algebra", encode(clean));
	const hostile = [
		clean.filter((event) => !(event.Action === "pass" && event.Test === second)),
		clean.map((event) => event.Action === "pass" && event.Test === first ? { ...event, Action: "skip" } : event),
		[...clean.slice(0, -1),
			{ Action: "run", Package: packagePath, Test: "TestUnexpected" },
			{ Action: "pass", Package: packagePath, Test: "TestUnexpected" },
			clean.at(-1)],
		[...clean.slice(0, 3), clean[2], ...clean.slice(3)],
		clean.map((event) => event.Action === "pass" && !event.Test ? { ...event, Action: "fail" } : event),
		clean.map((event, index) => index === 0 ? { ...event, Package: "example.invalid/foreign" } : event),
	];
	for (const [index, events] of hostile.entries()) {
		let rejected = false;
		try {
			validateGoJSONTranscript("exhaustive-algebra", encode(events));
		} catch {
			rejected = true;
		}
		if (!rejected) fail("P07B_C1_SELFTEST_GO_JSON_FALSE_NEGATIVE", index + 1);
	}
	return hostile.length + 1;
}

function inspectC2GoJSONTranscriptParser() {
	const packagePath = "github.com/nelsonwerd/countershape/internal/store";
	const first = "TestC2StoreExportsNoOfficialIssuerOrRunPermit";
	const second = "TestC2StoreExportsNoListLatestTraversalStatusOrHeadMutationSurface";
	const clean = [
		{ Action: "start", Package: packagePath },
		{ Action: "run", Package: packagePath, Test: first },
		{ Action: "pass", Package: packagePath, Test: first },
		{ Action: "run", Package: packagePath, Test: second },
		{ Action: "pass", Package: packagePath, Test: second },
		{ Action: "pass", Package: packagePath },
	];
	const encode = (events) => Buffer.from(`${events.map((event) => JSON.stringify(event)).join("\n")}\n`, "utf8");
	validateGoJSONTranscript("c2-public-surface", encode(clean));
	const hostile = [
		clean.filter((event) => !(event.Action === "pass" && event.Test === second)),
		clean.map((event) => event.Action === "pass" && event.Test === first ? { ...event, Action: "skip" } : event),
		[...clean.slice(0, -1),
			{ Action: "run", Package: packagePath, Test: "TestC2Unexpected" },
			{ Action: "pass", Package: packagePath, Test: "TestC2Unexpected" },
			clean.at(-1)],
		[...clean.slice(0, 3), clean[2], ...clean.slice(3)],
		clean.map((event) => event.Action === "pass" && !event.Test ? { ...event, Action: "fail" } : event),
		clean.map((event, index) => index === 0 ? { ...event, Package: "example.invalid/foreign" } : event),
	];
	for (const [index, events] of hostile.entries()) {
		let rejected = false;
		try {
			validateGoJSONTranscript("c2-public-surface", encode(events));
		} catch {
			rejected = true;
		}
		if (!rejected) fail("P07B_C2_SELFTEST_GO_JSON_FALSE_NEGATIVE", index + 1);
	}
	return hostile.length + 1;
}

const c3ParserProfiles = Object.freeze([
	Object.freeze({
		name: "c3-official-target",
		packagePath: "github.com/nelsonwerd/countershape/internal/contractexec",
		tests: Object.freeze([
			"TestC3OfficialTargetAttemptsAreFreshAndDistinct",
			"TestC3OfficialTargetClosedCapabilityAndDefensiveGetters",
			"TestC3OfficialTargetInvalidPreSpawnInputsReturnNoAuthority",
			"TestC3OfficialTargetLinkedRelationshipMutationRefusesReopen",
			"TestC3OfficialTargetMaterializationMutationRefusesReopen",
			"TestC3OfficialTargetMovingRefCannotRetargetReopen",
			"TestC3OfficialTargetPublicSurfaceAndSoleIssuer",
			"TestC3OpenOfficialTargetRebuildsFullAuthorityAcrossRestart",
			"TestC3OpenOfficialTargetLiveParentMatrix",
			"TestC3PublishOfficialTargetJoinsExactLivePrerequisiteGraph",
		]),
	}),
	Object.freeze({
		name: "c3-single-target",
		packagePath: "github.com/nelsonwerd/countershape/internal/gitobj",
		tests: Object.freeze([
			"TestC3SingleTargetAmbiguityRequiresExplicitReopen",
			"TestC3SingleTargetCannotPublishTwiceIntoOnePrivateParent",
			"TestC3SingleTargetExcludesDirtyWorktreeBytes",
			"TestC3SingleTargetMaterializesDirectlyFromInspectedAuthority",
			"TestC3SingleTargetMovingRefCannotChangePinnedBytes",
			"TestC3SingleTargetRefusesUnsupportedTreeFormsBeforePublication",
			"TestC3SingleTargetReopenRefusesEveryPublishedMutation",
		]),
	}),
	Object.freeze({
		name: "c3-hostepoch",
		packagePath: "github.com/nelsonwerd/countershape/internal/hostepoch",
		tests: Object.freeze([
			"TestC3HostEpochCanonicalMeasurement",
			"TestC3HostEpochConcurrentMeasurementsNeverCache",
			"TestC3HostEpochContextAndForgedAuthorityRefuse",
			"TestC3HostEpochDarwinLiveMeasurement",
			"TestC3HostEpochFaultAndInstabilityRefuse",
			"TestC3HostEpochMalformedSamplesRefuse",
			"TestC3HostEpochPublicSurfaceIsClosed",
			"TestC3HostEpochRevalidationRequiresSameFreshMeasurement",
		]),
	}),
	Object.freeze({
		name: "c3-noderuntime",
		packagePath: "github.com/nelsonwerd/countershape/internal/noderuntime",
		tests: Object.freeze([
			"TestC3NodeRuntimeCopiedCapabilitiesSerializeRevalidation",
			"TestC3NodeRuntimeDarwinLiveAdmissionAndRevalidation",
			"TestC3NodeRuntimeDarwinRejectsNonNodeExecutable",
			"TestC3NodeRuntimeDarwinRejectsNonPrivateProbeParent",
			"TestC3NodeRuntimeFaultMatrixRefusesAuthority",
			"TestC3NodeRuntimeMeasureProbeMeasureAdmission",
			"TestC3NodeRuntimeProbeParserIsClosed",
			"TestC3NodeRuntimeProbeProgramDigestIsExact",
			"TestC3NodeRuntimeProbeWritersDrainAfterBounds",
			"TestC3NodeRuntimePublicSurfaceAndSoleSpawnEdge",
			"TestC3NodeRuntimeRejectsAmbientOrForgedInputs",
			"TestC3NodeRuntimeRevalidationIsFreshAndExact",
		]),
	}),
	Object.freeze({
		name: "c3-store-bridge",
		packagePath: "github.com/nelsonwerd/countershape/internal/store",
		tests: Object.freeze([
			"TestC3ConformanceAttemptConcurrentValidationAndReopenAreRaceFree",
			"TestC3ConformanceAttemptIsFreshDurableAndRestartReopenable",
			"TestC3ConformanceAttemptMarkerMutationRefusesReopen",
			"TestC3ConformanceAttemptRejectsCrossStoreAndRootReplacement",
			"TestC3ContractTargetBridgeConvergesExactAndRejectsReuse",
			"TestC3StoreBridgeExportsOnlyInertAttemptAndTargetRecords",
		]),
	}),
]);

const c4ParserProfiles = Object.freeze([
	Object.freeze({
		name: "c4-processmechanics-parity",
		packagePath: "github.com/nelsonwerd/countershape/internal/processmechanics",
		packageArgument: "./internal/processmechanics",
		race: true,
		tests: Object.freeze([
			"TestStdoutAndStderrHaveIndependentExactCaps",
			"TestStdoutAndStderrLimitsAreIndependentMutationGuard",
			"TestSimultaneousChannelOverflowRetainsIndependentFacts",
			"TestPreTermRetryNeverUsesPostDeadlineProbeAsSignalAuthority",
			"TestPreTermRetryWaitOvershootDoesNotConsumeAnotherProbe",
			"TestPreparedProcessStartsOnceAndClosesWithParentObservedFacts",
			"TestCopiedPreparedHandleCannotMultiplyStartAuthority",
			"TestCopiedRunningHandleSharesOneTerminalClosure",
			"TestPresentEmptyStdinRemainsPhysicallyDistinctFromAbsentStdin",
			"TestExecutionBudgetStartsAtPhysicalStartNotClose",
			"TestPostPermitCancellationStillProducesAChildObservation",
			"TestSpawnObservationPersistenceAbortIsClosedAndTerminal",
		]),
	}),
	Object.freeze({
		name: "c4-admission-permit",
		packagePath: "github.com/nelsonwerd/countershape/internal/contractexec/runner",
		packageArgument: "./internal/contractexec/runner",
		race: false,
		tests: Object.freeze([
			"TestConcurrentAdmissionProducesExactlyOneStart",
			"TestRunPermitConsumptionIsSingleUseAndAdjacentToStart",
			"TestStartErrorClosesDurableRunAndClassificationWithoutChild",
			"TestPreparedCLIEnvironmentUsesFreshAttemptEvidenceRootAndShortCanaries",
			"TestParentSentinelEnvironmentIsOmittedFromRealChild",
			"TestSameTargetRetryRefusesAfterTerminalClosure",
			"TestCallerCancellationAfterAdmissionStillClosesTerminalFacts",
		]),
	}),
	Object.freeze({
		name: "c4-cli-closure",
		packagePath: "github.com/nelsonwerd/countershape/testkit/contractexec/cli",
		packageArgument: "./testkit/contractexec/cli",
		race: false,
		tests: Object.freeze([
			"TestCLIContractExecutionClosesStandaloneScope",
			"TestCLIContractExecutionForbiddenPositiveControls",
			"TestCLIContractExecutionChildBindingEvidenceStates",
			"TestCLIContractExecutionTargetMutationBlocksFinalization",
		]),
	}),
	Object.freeze({
		name: "c4-finalized-run-release",
		packagePath: "github.com/nelsonwerd/countershape/internal/store",
		packageArgument: "./internal/store",
		race: false,
		tests: Object.freeze([
			"TestC4ContractRunBridgePersistsClosesClassifiesAndReopens",
			"TestC4FinalizedReleaseConvergesReceiptWithoutRewritingClear",
			"TestC4FinalizedClearReceiptWithoutRunLinkCannotReadmit",
			"TestC4FinalizedRunRefusesMissingSpawnObservationBeforePublication",
			"TestC4TerminalClosureRequiresDurableSpawnObservationButClassificationDoesNot",
		]),
	}),
	Object.freeze({
		name: "c4-classification-recovery",
		packagePath: "github.com/nelsonwerd/countershape/internal/store",
		packageArgument: "./internal/store",
		race: false,
		tests: Object.freeze([
			"TestC4ClassificationRecoveryUsesHistoricalRunReleaseLink",
			"TestC4ClassificationRecoveryDoesNotRequireRetainedPrivatePack",
		]),
	}),
	Object.freeze({
		name: "c4-authority-race",
		packagePath: "github.com/nelsonwerd/countershape/internal/store",
		packageArgument: "./internal/store",
		race: true,
		tests: Object.freeze([
			"TestC4ContractRunBridgeRefusesSkippedAndMismatchedEdges",
			"TestC4SpawnObservationPersistsEveryClosedStartErrorExactly",
			"TestC4PrivateManifestDerivesRefsGroupsBodiesAndCannotFork",
			"TestC4StoreRunBridgeExportsOnlyOpaqueTypedAuthority",
		]),
	}),
]);

function inspectC3GoJSONTranscriptParser() {
	const encode = (events) => Buffer.from(`${events.map((event) => JSON.stringify(event)).join("\n")}\n`, "utf8");
	let count = 0;
	for (const profile of c3ParserProfiles) {
		const clean = [
			{ Action: "start", Package: profile.packagePath },
			...profile.tests.flatMap((Test) => [
				{ Action: "run", Package: profile.packagePath, Test },
				{ Action: "pass", Package: profile.packagePath, Test },
			]),
			{ Action: "pass", Package: profile.packagePath },
		];
		validateGoJSONTranscript(profile.name, encode(clean));
		count += 1;
		const first = profile.tests[0];
		const last = profile.tests.at(-1);
		const hostile = [
			clean.filter((event) => !(event.Action === "pass" && event.Test === last)),
			clean.map((event) => event.Action === "pass" && event.Test === first ? { ...event, Action: "skip" } : event),
			[...clean.slice(0, -1),
				{ Action: "run", Package: profile.packagePath, Test: "TestC3Unexpected" },
				{ Action: "pass", Package: profile.packagePath, Test: "TestC3Unexpected" },
				clean.at(-1)],
			[...clean.slice(0, 3), clean[2], ...clean.slice(3)],
			clean.map((event) => event.Action === "pass" && !event.Test ? { ...event, Action: "fail" } : event),
			clean.map((event, index) => index === 0 ? { ...event, Package: "example.invalid/foreign" } : event),
		];
		for (const [index, events] of hostile.entries()) {
			let rejected = false;
			try {
				validateGoJSONTranscript(profile.name, encode(events));
			} catch {
				rejected = true;
			}
			if (!rejected) fail("P07B_C3_SELFTEST_GO_JSON_FALSE_NEGATIVE", `${profile.name}:${index + 1}`);
			count += 1;
		}
	}
	return count;
}

function inspectC4GoJSONTranscriptParser() {
	const encode = (events) => Buffer.from(`${events.map((event) => JSON.stringify(event)).join("\n")}\n`, "utf8");
	let count = 0;
	for (const profile of c4ParserProfiles) {
		const clean = [
			{ Action: "start", Package: profile.packagePath },
			...profile.tests.flatMap((Test) => [
				{ Action: "run", Package: profile.packagePath, Test },
				{ Action: "pass", Package: profile.packagePath, Test },
			]),
			{ Action: "pass", Package: profile.packagePath },
		];
		validateGoJSONTranscript(profile.name, encode(clean));
		count += 1;
		const first = profile.tests[0];
		const last = profile.tests.at(-1);
		const hostile = [
			clean.filter((event) => !(event.Action === "pass" && event.Test === last)),
			clean.map((event) => event.Action === "pass" && event.Test === first ? { ...event, Action: "skip" } : event),
			[...clean.slice(0, -1),
				{ Action: "run", Package: profile.packagePath, Test: "TestC4Unexpected" },
				{ Action: "pass", Package: profile.packagePath, Test: "TestC4Unexpected" },
				clean.at(-1)],
			[...clean.slice(0, 3), clean[2], ...clean.slice(3)],
			clean.map((event) => event.Action === "pass" && !event.Test ? { ...event, Action: "fail" } : event),
			clean.map((event, index) => index === 0 ? { ...event, Package: "example.invalid/foreign" } : event),
		];
		for (const [index, events] of hostile.entries()) {
			let rejected = false;
			try {
				validateGoJSONTranscript(profile.name, encode(events));
			} catch {
				rejected = true;
			}
			if (!rejected) fail("P07B_C4_SELFTEST_GO_JSON_FALSE_NEGATIVE", `${profile.name}:${index + 1}`);
			count += 1;
		}
	}
	return count;
}

function inspectC4GoJSONArguments() {
	for (const profile of c4ParserProfiles) {
		const expected = [
			"test", ...(profile.race ? ["-race"] : []),
			"-mod=readonly", "-buildvcs=false", "-p=1", "-count=1", "-json", "-run",
			`^(?:${profile.tests.join("|")})$`, profile.packageArgument,
		];
		const actual = goJSONArguments(profile.name);
		if (JSON.stringify(actual) !== JSON.stringify(expected)) {
			fail("P07B_C4_SELFTEST_GO_JSON_ARGUMENTS", `${profile.name}:${JSON.stringify(actual)}`);
		}
	}
	let rejected = false;
	try {
		goJSONArguments("c4-unregistered-profile");
	} catch {
		rejected = true;
	}
	if (!rejected) fail("P07B_C4_SELFTEST_GO_JSON_ARGUMENTS", "unknown C4 profile was accepted");
	return c4ParserProfiles.length + 1;
}

function inspectC4OwnerReferenceParser() {
	const cases = [
		{ source: "package store\nfunc AcquireContractRunOwner() {}\n", want: 1 },
		{ source: "package runner\nvar acquire = vault.AcquireContractRunOwner\n", want: 1 },
		{ source: "package runner\nfunc f() { AcquireContractRunOwner() }\n", want: 1 },
		{ source: "package runner\n// AcquireContractRunOwner\nvar text = `AcquireContractRunOwner`\n", want: 0 },
		{ source: "package runner\nvar a = store.AcquireContractRunOwner\nvar b = store.AcquireContractRunOwner\n", want: 2 },
	];
	for (const [index, test] of cases.entries()) {
		const actual = productionStoreOwnerReferenceCount(test.source);
		if (actual !== test.want) fail("P07B_C4_SELFTEST_OWNER_REFERENCE_PARSER", `${index}:${actual} != ${test.want}`);
	}
	return cases.length;
}

function inspectC3PredecessorAuthorityParser() {
	const cleanNotes = new Map();
	for (const unit of ["C3S", "C3F", "C3L", "C3A"]) {
		collectC3PredecessorRecordAuthority(unit, (args, result) => {
			if (args[0] === "cat-file" && args[1] === "blob") {
				cleanNotes.set(unit, Buffer.from(result.stdout));
			}
			return result;
		});
	}
	for (const unit of ["C3S", "C3F", "C3L", "C3A"]) {
		if (!Buffer.isBuffer(cleanNotes.get(unit))) fail("P07B_C3_SELFTEST_PREDECESSOR_FIXTURE", `${unit} note body was not captured`);
		inspectC3DidrunNote(unit, cleanNotes.get(unit));
	}
	const cleanC3SNote = cleanNotes.get("C3S");
	let count = 8;
	const requireRejected = (name, action) => {
		let rejected = false;
		try { action(); } catch { rejected = true; }
		if (!rejected) fail("P07B_C3_SELFTEST_PREDECESSOR_FALSE_NEGATIVE", name);
		count += 1;
	};
	const rawMutation = (name, matches, stdout) => requireRejected(name, () => {
		collectC3PredecessorRecordAuthority("C3S", (args, result) => matches(args)
			? { ...result, stdout: typeof stdout === "function" ? stdout(result.stdout) : Buffer.from(stdout) }
			: result);
	});
	const line = (value) => Buffer.from(`${value}\n`, "utf8");
	rawMutation("invalid UTF-8 scalar", (args) => args[0] === "show" && args.includes("--format=%s"), Buffer.from([0xff, 0x0a]));
	rawMutation("commit identity", (args) => args[0] === "rev-parse" && args.at(-1).endsWith("^{commit}"), line("0".repeat(40)));
	rawMutation("tree identity", (args) => args[0] === "rev-parse" && args.at(-1).endsWith("^{tree}"), line("0".repeat(40)));
	rawMutation("wrong parent", (args) => args[0] === "show" && args.includes("--format=%P"), line("0".repeat(40)));
	rawMutation("merge parent cardinality", (args) => args[0] === "show" && args.includes("--format=%P"), line(`${"0".repeat(40)} ${"1".repeat(40)}`));
	rawMutation("subject identity", (args) => args[0] === "show" && args.includes("--format=%s"), line("altered subject"));
	rawMutation("subject trailing space", (args) => args[0] === "show" && args.includes("--format=%s"), line("fix: make U6 C3 fixture phase-stable "));
	rawMutation("subject extra LF", (args) => args[0] === "show" && args.includes("--format=%s"), Buffer.from("fix: make U6 C3 fixture phase-stable\n\n", "utf8"));
	rawMutation("subject CRLF", (args) => args[0] === "show" && args.includes("--format=%s"), Buffer.from("fix: make U6 C3 fixture phase-stable\r\n", "utf8"));
	rawMutation("parent repeated separator", (args) => args[0] === "show" && args.includes("--format=%P"), line(" 63038644ba347d7b934a5557a490af95bb4428a4"));
	rawMutation("note blob identity", (args) => args[0] === "notes", line("0".repeat(40)));
	rawMutation("note object type", (args) => args[0] === "cat-file" && args[1] === "-t", line("tree"));
	rawMutation("raw note body digest", (args) => args[0] === "cat-file" && args[1] === "blob", (body) => Buffer.concat([body, Buffer.from(" ")]));

	requireRejected("invalid UTF-8 note", () => inspectC3DidrunNote("C3S", Buffer.from([0xff])));
	requireRejected("malformed note JSON", () => inspectC3DidrunNote("C3S", Buffer.from("{", "utf8")));
	const mutateNote = (name, mutate) => requireRejected(name, () => {
		const note = JSON.parse(cleanC3SNote.toString("utf8"));
		mutate(note);
		inspectC3DidrunNote("C3S", Buffer.from(`${JSON.stringify(note)}\n`, "utf8"));
	});
	mutateNote("note version", (note) => { note.version = 2; });
	mutateNote("note commit", (note) => { note.commit = "0".repeat(40); });
	mutateNote("note tree", (note) => { note.tree = "0".repeat(40); });
	mutateNote("note secrets disclosure", (note) => { note.secrets_override = false; });
	mutateNote("note claim count", (note) => { note.claims.pop(); });
	mutateNote("note claim order", (note) => { [note.claims[0], note.claims[1]] = [note.claims[1], note.claims[0]]; });
	mutateNote("note claim label", (note) => { note.claims[0].claim.label += " altered"; });
	mutateNote("note claim type", (note) => { note.claims[0].claim.ctype = "command-succeeded"; });
	mutateNote("note claim index", (note) => { note.claims[0].claim.declared_at_index = 1; });
	mutateNote("note root field roster", (note) => { note.extra = true; });
	mutateNote("note coverage field roster", (note) => { note.coverage.extra = true; });
	mutateNote("note recorded claim field roster", (note) => { note.claims[0].extra = true; });
	mutateNote("note claim field roster", (note) => { note.claims[0].claim.extra = true; });
	mutateNote("note event indices", (note) => { note.claims[0].claim.event_indices = [1]; });
	mutateNote("note supporting event index", (note) => { note.claims[0].supporting_event_index = 1; });
	mutateNote("note pathspecs", (note) => { note.claims[0].claim.pathspecs = ["."]; });
	mutateNote("note delta", (note) => { note.claims[0].delta = ["changed"]; });
	mutateNote("note exit code", (note) => { note.claims[0].exit_code = 1; });
	mutateNote("note grade", (note) => { note.claims[0].grade = "scope-exact"; });
	mutateNote("note reason", (note) => { note.claims[0].reason = "changed"; });
	mutateNote("empty note argv", (note) => { note.claims[0].claim.argv_preview = []; });
	mutateNote("non-string note argv", (note) => { note.claims[0].claim.argv_preview[0] = 7; });
	mutateNote("note argv command insertion", (note) => { note.claims[0].claim.argv_preview.splice(39, 0, "/usr/bin/true"); });
	mutateNote("note argv cache prefix", (note) => { note.claims[0].claim.argv_preview[6] += "-altered"; });
	mutateNote("note argv tool binding", (note) => { note.claims[0].claim.argv_preview[31] += "-altered"; });
	mutateNote("note argv tail substitution", (note) => { note.claims[0].claim.argv_preview[note.claims[0].claim.argv_preview.length - 1] += "-altered"; });
	mutateNote("note argv appended argument", (note) => { note.claims[0].claim.argv_preview.push("--extra"); });
	mutateNote("note total coverage", (note) => { note.coverage.total_events -= 1; });
	mutateNote("note complete coverage", (note) => { note.coverage.by_coverage.complete -= 1; });
	mutateNote("note extra coverage class", (note) => { note.coverage.by_coverage.partial = 1; });
	const chainMutation = (name, mutate) => requireRejected(name, () => {
		collectC3PredecessorAuthority((args, result) => args[0] === "merge-base" ? mutate(result) : result);
	});
	chainMutation("ancestry false status", (result) => ({ ...result, status: 1 }));
	chainMutation("ancestry unexpected status", (result) => ({ ...result, status: 2 }));
	chainMutation("ancestry nonempty stdout", (result) => ({ ...result, stdout: Buffer.from("unexpected\n", "utf8") }));
	chainMutation("ancestry stderr", (result) => ({ ...result, stderr: Buffer.from("unexpected\n", "utf8") }));
	chainMutation("ancestry malformed stdout", (result) => ({ ...result, stdout: "" }));
	return count;
}

async function runC2Selftest() {
	const digest = rosterDigest(c2Cases);
	if (digest !== expectedC2RosterDigest) fail("P07B_C2_SELFTEST_ROSTER_DRIFT", `${digest} != ${expectedC2RosterDigest}`);
	runCleanChecker("c2");
	const clean = await collectC2Facts();
	const cleanProblems = validateC2Facts(clean);
	if (cleanProblems.length > 0) fail("P07B_C2_SELFTEST_CLEAN_FACTS", cleanProblems.map((problem) => problem.code).join(","));

	for (const test of c2Cases) {
		const facts = structuredClone(clean);
		switch (test.id) {
		case "store-production-files": facts.package.productionFiles.pop(); break;
		case "store-directory-entry": facts.directoryEntries.push("foreign.go:file"); facts.directoryEntries.sort(); break;
		case "production-import": facts.imports["execution_interlock.go"].push("net/http"); facts.imports["execution_interlock.go"].sort(); break;
		case "compiler-production-import": facts.package.productionImports.push("example.invalid/process-capability"); facts.package.productionImports.sort(); break;
		case "new-production-export": facts.newProductionExports["internal/store/nonhead_contract.go"].push("OfficialTarget"); break;
		case "object-store-export": facts.objectStoreExports.pop(); break;
		case "compiler-parsed-surface": facts.compilerParsedSurface = false; break;
		case "authority-shape": facts.structs.interlockLease.pop(); break;
		case "test-symbol": facts.testSymbols.pop(); break;
		case "test-file-roster": facts.testFiles["internal/store/execution_interlock_test.go"].pop(); break;
		case "forbidden-surface": facts.forbiddenSurface.push("process-start"); break;
		case "namespace-identity": facts.namespaces.retained = false; break;
		case "namespace-case-alias": facts.namespaces.caseAliasGuard = false; break;
		case "relation-constants": facts.relations.constants = false; break;
		case "relation-effects": facts.relations.effects.pop(); break;
		case "relation-persistence-order": facts.relations.persistenceOrder = false; break;
		case "relation-open-convergence": facts.relations.openConvergence = false; break;
		case "run-manifest-gate": facts.relations.runManifestGate = false; break;
		case "execution-profile-derived": facts.relations.executionProfileDerived = false; break;
		case "relation-identity-guards": facts.relations.identityGuards = false; break;
		case "relation-identity-tests": facts.relations.identityReplacementTests = false; break;
		case "boot-join": facts.interlock.bootJoin = false; break;
		case "acquisition-order": facts.interlock.acquisitionOrder = false; break;
		case "release-order": facts.interlock.releaseOrder = false; break;
		case "reset-order": facts.interlock.resetOrder = false; break;
		case "winner-freshness": facts.interlock.winnerFreshAndConsumed = false; break;
		case "start-claim-convergence": facts.interlock.startClaimConvergence = false; break;
		case "clear-receipt-transition": facts.interlock.clearReceiptTransition = false; break;
		case "clear-receipt-convergence": facts.interlock.clearReceiptConvergence = false; break;
		case "clear-receipt-faults": facts.interlock.clearReceiptFaults = false; break;
		case "interlock-identity-boundary": facts.interlock.identityBoundary = false; break;
		case "seal-ownership": facts.interlock.productionSeals.lease += 1; break;
		case "private-limits": facts.privateEvidence.limits = false; break;
		case "private-kind-roster": facts.privateEvidence.kinds.pop(); break;
		case "private-state-roster": facts.privateEvidence.states.pop(); break;
		case "manifest-closure": facts.privateEvidence.manifestClosure = false; break;
		case "manifest-open-convergence": facts.privateEvidence.manifestOpenConvergence = false; break;
		case "private-arithmetic": facts.privateEvidence.arithmeticBounded = false; break;
		case "availability-convergence": facts.privateEvidence.availabilityJoins = false; break;
		case "purge-order": facts.privateEvidence.purgeOrder = false; break;
		case "purge-create-count": facts.privateEvidence.purgeCreateCount = false; break;
		case "fresh-purge-order": facts.privateEvidence.freshPurgeOrder = false; break;
		case "missing-pack-gate": facts.privateEvidence.missingPackGate = false; break;
		case "canonical-isolation": facts.privateEvidence.noCanonicalMutation = false; break;
		case "private-identity-guards": facts.privateEvidence.identityGuards = false; break;
		case "private-identity-tests": facts.privateEvidence.identityReplacementTests = false; break;
		default: fail("P07B_C2_SELFTEST_UNKNOWN_CASE", test.id);
		}
		requireC2Violation(facts, test.code, test.id);
	}
	const goJSONCases = inspectC2GoJSONTranscriptParser();
	process.stdout.write(`P07B-C C2 cumulative architecture defensive self-test OK (${c2Cases.length} metadata cases; ${goJSONCases} Go JSON parser cases)\n`);
}

async function runC3Selftest() {
	const digest = rosterDigest(c3Cases);
	if (digest !== expectedC3RosterDigest) fail("P07B_C3_SELFTEST_ROSTER_DRIFT", `${digest} != ${expectedC3RosterDigest}`);
	runCleanChecker("c3");
	const clean = await collectC3Facts();
	const cleanProblems = validateC3Facts(clean);
	if (cleanProblems.length > 0) fail("P07B_C3_SELFTEST_CLEAN_FACTS", cleanProblems.map((problem) => problem.code).join(","));

	for (const test of c3Cases) {
		const facts = structuredClone(clean);
		switch (test.id) {
		case "package-topology":
			facts.packages["github.com/nelsonwerd/countershape/internal/contractexec"].go.pop();
			break;
		case "build-tags":
			facts.buildTags["internal/contractexec/target_test.go"] = "darwin";
			break;
		case "source-import":
			facts.sourceImports["internal/contractexec/target.go"].push("net/http");
			facts.sourceImports["internal/contractexec/target.go"].sort();
			break;
		case "exported-surface":
			facts.sourceSurfaces["internal/contractexec/target.go"].pop();
			break;
		case "test-file-roster":
			facts.testFiles["internal/contractexec/target_test.go"].pop();
			break;
		case "inherited-c2":
			facts.c2Problems.push({ code: "P07B_C2_SELFTEST_SENTINEL", detail: "forced" });
			break;
		case "store-bridge":
			facts.storeBridge.noIssuerOrProcessEdge = false;
			break;
		case "git-target":
			facts.gitTarget.reopenOrder = false;
			break;
		case "host-epoch":
			facts.hostEpoch.noFallbackOrProcess = false;
			break;
		case "node-runtime":
			facts.nodeRuntime.spawnOwners.push("internal/contractexec/target.go");
			facts.nodeRuntime.spawnOwners.sort();
			break;
		case "official-target":
			facts.officialTarget.faultCoverage = false;
			break;
		case "predecessor": {
			facts.predecessor.source_parent.commit = "0".repeat(40);
			requireC3Violation(facts, test.code, test.id);
			const authorityFacts = structuredClone(clean);
			authorityFacts.predecessorAuthority.source_parent.note_blob = "0".repeat(40);
			requireC3Violation(authorityFacts, "P07B_C3_PREDECESSOR_AUTHORITY", "predecessor-authority");
			const chainFacts = structuredClone(clean);
			chainFacts.predecessorAuthority.chain.source_parent_is_ancestor_of_head = false;
			requireC3Violation(chainFacts, "P07B_C3_PREDECESSOR_CHAIN", "predecessor-chain");
			continue;
		}
		default:
			fail("P07B_C3_SELFTEST_UNKNOWN_CASE", test.id);
		}
		requireC3Violation(facts, test.code, test.id);
	}
	const goJSONCases = inspectC3GoJSONTranscriptParser();
	const predecessorCases = inspectC3PredecessorAuthorityParser();
	process.stdout.write(`P07B-C C3 cumulative architecture defensive self-test OK (${c3Cases.length} metadata cases; ${goJSONCases} Go JSON parser cases; ${predecessorCases} raw predecessor/parser cases)\n`);
}

async function runC4Selftest() {
	const digest = rosterDigest(c4Cases);
	if (digest !== expectedC4RosterDigest) fail("P07B_C4_SELFTEST_ROSTER_DRIFT", `${digest} != ${expectedC4RosterDigest}`);
	runCleanChecker("c4");
	const clean = await collectC4Facts();
	const cleanProblems = validateC4Facts(clean);
	if (cleanProblems.length > 0) fail("P07B_C4_SELFTEST_CLEAN_FACTS", cleanProblems.map((problem) => problem.code).join(","));
	const statusSource = await readFile(resolve(root, "docs/status/P07B-C-C4-CLI-PROFILE.md"), "utf8");
	const runbookSource = renderC4RunbookSource();

	for (const test of c4Cases) {
		const facts = structuredClone(clean);
		switch (test.id) {
		case "inherited-c3":
			facts.c3Problems.push({ code: "P07B_C3_SELFTEST_SENTINEL", detail: "forced" });
			break;
		case "package-topology":
			facts.packages["github.com/nelsonwerd/countershape/internal/processmechanics"].go.pop();
			break;
		case "build-tags":
			facts.buildTags["internal/processmechanics/process_darwin.go"] = "darwin && arm64";
			break;
		case "test-file-roster":
			facts.testFiles["internal/contractexec/runner/runner_darwin_test.go"].pop();
			break;
		case "mechanics-authority":
			facts.mechanics.copySafeState = false;
			break;
		case "mechanics-importers":
			facts.mechanics.importers.push("github.com/nelsonwerd/countershape/testkit/contractexec/cli");
			facts.mechanics.importers.sort();
			break;
		case "world-adapter":
			facts.worldAdapter.noDuplicateSpawn = false;
			break;
		case "runner-surface":
			facts.runner.surface.pop();
			break;
		case "admission-adjacency":
			facts.runner.spawnAdjacentRevalidation = false;
			facts.runner.soleOwnerAcquirer.push("internal/store/alternate_owner.go:1");
			break;
		case "execution-chronology":
			for (const field of [
				"physicalConforms", "executionChronology", "startErrorChronology", "immutableSourceCacheIsolation",
				"boundedPhysicalTestConcurrency",
				"preOwnerEvidenceCapacity", "actualDraftCapacityGate",
			]) {
				const hostile = structuredClone(clean);
				hostile.runner[field] = false;
				requireC4Violation(hostile, test.code, `${test.id}-${field}`);
			}
			continue;
		case "terminal-graph":
			facts.terminalGraph.releaseOrder = false;
			break;
		case "evidence-scope":
			for (const field of [
				"invocationClosure", "boundedCandidateInventory", "rawFramedSingleCopy", "boundedStoreRosters",
			]) {
				const hostile = structuredClone(clean);
				hostile.evidence[field] = false;
				requireC4Violation(hostile, test.code, `${test.id}-${field}`);
			}
			continue;
		case "scope-root":
			for (const field of [
				"attemptPrivateRoot", "shortPrivateAlias", "boundedDescriptorRoster",
				"identityBoundTerminalCleanup", "residueHostiles",
			]) {
				const hostile = structuredClone(clean);
				hostile.scopeProbe[field] = false;
				requireC4Violation(hostile, test.code, `${test.id}-${field}`);
			}
			continue;
		case "profile-command": {
			const commandHostile = structuredClone(clean);
			const args = commandHostile.profiles["c4-authority-race"];
			args.splice(args.indexOf("-race"), 1);
			requireC4Violation(commandHostile, test.code, `${test.id}-go-argv`);

			const firstLabel = clean.claimMap.status[0].label;
			const driftLabel = `${firstLabel} drift`;
			const statusLabelToken = `| 1 | \`${firstLabel}\` |`;
			const statusDriftSource = statusSource.replace(statusLabelToken, `| 1 | \`${driftLabel}\` |`);
			if (statusDriftSource === statusSource) fail("P07B_C4_SELFTEST_SOURCE_FIXTURE", "status label token");
			const parsedStatusDrift = parseC4StatusClaimMap(statusDriftSource);
			if (parsedStatusDrift[0].label !== driftLabel) fail("P07B_C4_SELFTEST_PARSER_HARDCODED", "status label");
			const statusSourceHostile = structuredClone(clean);
			statusSourceHostile.claimMap.status = parsedStatusDrift;
			requireC4Violation(statusSourceHostile, test.code, `${test.id}-status-source`);
			requireC4ParserRefusal(() => parseC4StatusClaimMap(statusSource.replace(
				"| ---: | --- | --- | --- |", "| --- | --- | --- | --- |",
			)), `${test.id}-status-delimiter`);
			requireC4ParserRefusal(() => parseC4StatusClaimMap(statusSource.replace(
				"| `UNRECEIPTED` |", "| `TREE-EXACT` |",
			)), `${test.id}-status-grade`);
			const statusSection = statusSource.slice(statusSource.indexOf("## Intended C4 claim map\n"));
			requireC4ParserRefusal(
				() => parseC4StatusClaimMap(`${statusSource}\n${statusSection}`),
				`${test.id}-status-duplicate-section`,
			);

			const headingToken = `# 1. ${firstLabel}`;
			const claimToken = `--label '${firstLabel}'`;
			const runbookDriftSource = runbookSource.replace(headingToken, `# 1. ${driftLabel}`)
				.replace(claimToken, `--label '${driftLabel}'`);
			if (runbookDriftSource === runbookSource) fail("P07B_C4_SELFTEST_SOURCE_FIXTURE", "runbook label tokens");
			const parsedRunbookDrift = parseC4FinalRunbookClaimMap(runbookDriftSource);
			if (parsedRunbookDrift[0].label !== driftLabel) fail("P07B_C4_SELFTEST_PARSER_HARDCODED", "runbook label");
			const runbookSourceHostile = structuredClone(clean);
			runbookSourceHostile.claimMap.runbook = parsedRunbookDrift;
			requireC4Violation(runbookSourceHostile, test.code, `${test.id}-runbook-source`);
			requireC4ParserRefusal(
				() => parseC4FinalRunbookClaimMap(runbookSource.replace(headingToken, `# 1. ${driftLabel}`)),
				`${test.id}-runbook-heading-claim-disagreement`,
			);

			const normalizedHostile = structuredClone(clean);
			normalizedHostile.claimMap.status[0].label = driftLabel;
			requireC4Violation(normalizedHostile, test.code, `${test.id}-normalized-validator`);
			continue;
		}
		default:
			fail("P07B_C4_SELFTEST_UNKNOWN_CASE", test.id);
		}
		requireC4Violation(facts, test.code, test.id);
	}
	const goJSONCases = inspectC4GoJSONTranscriptParser();
	const argumentCases = inspectC4GoJSONArguments();
	const ownerReferenceCases = inspectC4OwnerReferenceParser();
	process.stdout.write(`P07B-C C4 cumulative architecture defensive self-test OK (${c4Cases.length} metadata cases; ${goJSONCases} Go JSON parser cases; ${argumentCases} command cases; ${ownerReferenceCases} owner-reference parser cases)\n`);
}

async function main() {
	if (process.argv[2] === "--c4") {
		if (process.argv.length !== 3) fail("P07B_C4_SELFTEST_ARGUMENTS", "--c4 accepts no other arguments");
		await runC4Selftest();
		return;
	}
	if (process.argv[2] === "--c3") {
		if (process.argv.length !== 3) fail("P07B_C3_SELFTEST_ARGUMENTS", "--c3 accepts no other arguments");
		await runC3Selftest();
		return;
	}
	if (process.argv[2] === "--c2") {
		if (process.argv.length !== 3) fail("P07B_C2_SELFTEST_ARGUMENTS", "--c2 accepts no other arguments");
		await runC2Selftest();
		return;
	}
	if (process.argv.length !== 2) fail("P07B_C1_SELFTEST_ARGUMENTS", "no arguments accepted");
	const digest = rosterDigest();
	if (digest !== expectedRosterDigest) fail("P07B_C1_SELFTEST_ROSTER_DRIFT", `${digest} != ${expectedRosterDigest}`);
	runCleanChecker();
	const clean = await collectFacts();
	const cleanProblems = validateFacts(clean);
	if (cleanProblems.length > 0) fail("P07B_C1_SELFTEST_CLEAN_FACTS", cleanProblems.map((problem) => problem.code).join(","));

	for (const test of cases) {
		const facts = structuredClone(clean);
		switch (test.id) {
		case "production-files":
			facts.package.productionFiles.pop();
			break;
		case "test-files":
			facts.package.testFiles.pop();
			break;
		case "test-file-rename":
			facts.package.testFiles[0] = "algebra_renamed_test.go";
			facts.package.testFiles.sort();
			break;
		case "test-import":
			facts.package.testImports.push("net/http");
			facts.package.testImports.sort();
			break;
		case "test-symbol":
			facts.package.testSymbols.pop();
			break;
		case "model-extra-entry":
			facts.topology.modelEntries.push("extra_test.go:file");
			facts.topology.modelEntries.sort();
			break;
		case "production-import":
			facts.package.productionImports.push("os");
			facts.package.productionImports.sort();
			break;
		case "dependency-closure":
			facts.localDependencies.push("github.com/nelsonwerd/countershape/internal/store");
			facts.localDependencies.sort();
			break;
		case "ignored-production":
			facts.package.ignoredGoFiles.push("runtime_other.go");
			break;
		case "foreign-build-input":
			facts.package.nonGoBuildFiles.push("bridge.c");
			break;
		case "premature-importer":
			facts.productionImporters.push("github.com/nelsonwerd/countershape/cmd/countershape");
			break;
		case "schema-closure":
			facts.schema.unclosedObjects.push("run/$defs/ClosedRunWitness");
			break;
		case "schema-required-roster":
			facts.schema.incompleteRequiredObjects.push("execution/");
			break;
		case "target-root":
			facts.schema.targetRoot.push("result");
			break;
		case "serialized-disposition":
			facts.schema.forbiddenMembers.push("run/properties/terminal_disposition");
			break;
		case "result-roster":
			facts.schema.results.pop();
			break;
		case "scope-domain":
			facts.schema.scopeDomains.pop();
			break;
		case "limit-profile":
			facts.schema.limits.privateByteMax += 1;
			break;
		case "evidence-roster":
			facts.example.witnessReferenceKinds.pop();
			facts.example.witnessReferenceCount -= 1;
			break;
		case "example-graph":
			facts.example.graph.runToExecution = false;
			break;
		case "c0-authority":
			facts.c0.objects.pop();
			break;
		case "topology":
			facts.topology.contractexecEntries.push("http:directory");
			facts.topology.contractexecEntries.sort();
			break;
		default:
			fail("P07B_C1_SELFTEST_UNKNOWN_CASE", test.id);
		}
		requireViolation(facts, test.code, test.id);
	}
	const goJSONCases = inspectGoJSONTranscriptParser();
	process.stdout.write(`P07B-C C1 architecture defensive self-test OK (${cases.length} metadata cases; ${goJSONCases} Go JSON parser cases)\n`);
}

main().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
