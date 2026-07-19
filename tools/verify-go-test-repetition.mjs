#!/usr/bin/env node

import { createHash } from "node:crypto";
import { arch, platform } from "node:os";
import { realpath, rm } from "node:fs/promises";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

import {
	acquireVerificationLock,
	admitTools,
	buildChildEnvironment,
	childResult,
	createPrivateRoots,
	finalizeVerificationResources,
	repositoryRoot,
	sensitiveGoPackages,
} from "./verify-current.mjs";

const modulePath = "github.com/nelsonwerd/countershape";
const allowedActions = new Set(["bench", "cont", "fail", "output", "pass", "pause", "run", "skip", "start"]);
const packageActions = new Set(["fail", "output", "pass", "start"]);
const testActions = new Set(["cont", "fail", "output", "pass", "pause", "run", "skip"]);
const expectedEntrypointPattern = /^(?:(?:Test|Fuzz)[A-Za-z0-9_]+|Example[A-Za-z0-9_]*)$/u;
const authorityNames = Object.freeze(["go", "node", "git", "sh", "cc", "cxx"]);

export class GoRepetitionError extends Error {
	constructor(code, detail) {
		super(`${code}: ${detail}`);
		this.code = code;
	}
}

function fail(code, detail) {
	throw new GoRepetitionError(code, detail);
}

function exactAlternation(names) {
	return names.length === 1 ? `^${names[0]}$` : `^(?:${names.join("|")})$`;
}

function qualificationCase({ id, packagePath, profile, count, expected, fullPackage = false }) {
	const sortedExpected = [...expected].sort();
	if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/u.test(id) || new Set(sortedExpected).size !== sortedExpected.length ||
		sortedExpected.length === 0 || sortedExpected.some((name) => !expectedEntrypointPattern.test(name))) {
		throw new Error(`invalid Go repetition qualification case: ${id}`);
	}
	const requiredProfile = sensitiveGoPackages.includes(packagePath) ? "sensitive" : "general";
	if (profile !== requiredProfile || !packagePath.startsWith(`${modulePath}/`) || !Number.isInteger(count) || count < 1 || count > 50) {
		throw new Error(`invalid Go repetition qualification case authority: ${id}`);
	}
	return Object.freeze({
		caseID: id,
		count,
		expected: Object.freeze(sortedExpected),
		packagePath,
		profile,
		qualification: true,
		run: fullPackage ? "^(?:Test|Fuzz|Example).*$" : exactAlternation(sortedExpected),
	});
}

const qualificationDefinitions = Object.freeze([
	{
		id: "world-output-caps-50", packagePath: `${modulePath}/internal/world`, profile: "sensitive", count: 50,
		expected: ["TestStdoutAndStderrHaveIndependentExactCaps"],
	},
	{
		id: "world-output-independence-20", packagePath: `${modulePath}/internal/world`, profile: "sensitive", count: 20,
		expected: ["TestStdoutAndStderrLimitsAreIndependentMutationGuard"],
	},
	{
		id: "world-simultaneous-overflow-20", packagePath: `${modulePath}/internal/world`, profile: "sensitive", count: 20,
		expected: ["TestSimultaneousChannelOverflowRetainsIndependentFacts"],
	},
	{
		id: "world-lifecycle-readiness-20", packagePath: `${modulePath}/internal/world`, profile: "sensitive", count: 20,
		expected: [
			"TestExecuteBuildsFreshWorldsAndPublishesMarkerBeforeSpawn",
			"TestFinalGroupProbeRequiresObservedAbsence",
			"TestHTTPPortableEarlyExitRetainsCausallyLaterReadinessEOF",
			"TestHTTPPortableReadinessFailuresRemainFinalizedReceipts",
			"TestPreTermProbeControlsWhetherTheOriginalGroupIsSignaled",
			"TestProcessGroupAndSessionEscapesRemainExplicitExclusions",
			"TestProcessLifecycleControlsAndCleansDescendants",
			"TestToolVersionProbeCleansDescendantHeldPipesWithinItsBound",
			"TestUnexpectedWaitFailureIsNotACompletedCleanupEdge",
		],
	},
	{
		id: "compiler-generated-runtime-20", packagePath: `${modulePath}/internal/emit/node/compiler`, profile: "sensitive", count: 20,
		expected: [
			"TestGeneratedCLIContractDistinguishesAbsentAndPresentEmptyStdin",
			"TestGeneratedCLIContractRunsFromPrivateTargetInventory",
			"TestGeneratedContractReportsHarnessFailureForVerifiedLoaderFailure",
			"TestGeneratedHTTPContractClassifiesProbeTimeout",
			"TestGeneratedHTTPContractClassifiesSocketReset",
			"TestGeneratedHTTPContractRunsFromPrivateTargetInventory",
		],
	},
	{
		id: "program-lifecycle-20", packagePath: `${modulePath}/internal/emit/node/program/v1`, profile: "sensitive", count: 20,
		expected: ["TestCopiedHarnessLifecycleStateMachines"],
	},
	{
		id: "store-cross-process-cas-20", packagePath: `${modulePath}/internal/store`, profile: "sensitive", count: 20,
		expected: ["TestStudyHeadCrossProcessCASHasExactlyOneWinner"],
	},
	...Array.from({ length: 20 }, (_, index) => ({
		id: `cli-physical-reducer-${String(index + 1).padStart(2, "0")}-of-20`,
		packagePath: `${modulePath}/testkit/studies/cli_precedence`,
		profile: "sensitive",
		count: 1,
		expected: ["TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence"],
	})),
	...Array.from({ length: 20 }, (_, index) => ({
		id: `http-physical-reducer-${String(index + 1).padStart(2, "0")}-of-20`,
		packagePath: `${modulePath}/testkit/studies/http_invoices`,
		profile: "sensitive",
		count: 1,
		expected: ["TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence"],
	})),
	{
		id: "parity-evaluator-20", packagePath: `${modulePath}/internal/emit/node/parity`, profile: "general", count: 20,
		expected: ["TestGoAndNodeParityEvaluatorsMatchLiteralOracle"],
	},
	{
		id: "parity-framing-20", packagePath: `${modulePath}/internal/emit/node/parity`, profile: "general", count: 20,
		expected: [
			"TestNodeParityRunnerAcceptsExactWholeWireCap",
			"TestNodeParityRunnerRejectsInvalidFramesAtomically",
			"TestNodeParityRunnerRejectsMissingFinalLFWithoutStderr",
			"TestParityFramingRejectsExpandedSemanticResultAtomically",
			"TestParityResponseFramingExactBodyBoundary",
		],
	},
	{
		id: "parity-full-package-3", packagePath: `${modulePath}/internal/emit/node/parity`, profile: "general", count: 3, fullPackage: true,
		expected: [
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
		],
	},
	{
		id: "cli-physical-full-package-3", packagePath: `${modulePath}/testkit/studies/cli_precedence`, profile: "sensitive", count: 3, fullPackage: true,
		expected: [
			"TestCLICompilationFreshProcessRestartHelper",
			"TestCLIMaterializationControlCrossesAdapterAndExcludedMap",
			"TestCLIPhysicalReducerBudgetFenceRetainsOnlyBestKnown",
			"TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
			"TestCLIP07BBPublicationHelper",
			"TestCLIPresentBytesStdinCrossesWorldAndAdapter",
			"TestCLIPresentEmptyStdinCrossesWorldAndAdapter",
			"TestCLIReferenceControlClassificationMatrix",
			"TestCLIReferenceDisplayPermutationPreservesMapWithFreshAttempts",
			"TestCLIReferencePrecedenceStudy",
			"TestCLIStudyUsesStrictSourceCompilerAndPlanBoundSchedule",
			"TestOptionalReceiptMeasurementKeepsAbsenceExplicit",
		],
	},
	{
		id: "http-physical-full-package-3", packagePath: `${modulePath}/testkit/studies/http_invoices`, profile: "sensitive", count: 3, fullPackage: true,
		expected: [
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
		],
	},
]);

export const qualificationCases = Object.freeze(Object.fromEntries(
	qualificationDefinitions.map((definition) => {
		const specification = qualificationCase(definition);
		return [specification.caseID, specification];
	}),
));
export const qualificationCaseIDs = Object.freeze(Object.keys(qualificationCases));

export function qualificationMatrixDigest() {
	return createHash("sha256").update(JSON.stringify(qualificationCases)).digest("hex");
}

function parseRunArguments(argv) {
	if (argv.length === 2 && argv[0] === "--case") {
		const specification = qualificationCases[argv[1]];
		if (!specification) fail("GO_REPETITION_QUALIFICATION_CASE", String(argv[1]));
		return specification;
	}
	if (argv.length < 10 || argv.length % 2 !== 0 || argv[0] !== "--package" || argv[2] !== "--profile" ||
		argv[4] !== "--count" || argv[6] !== "--run") {
		fail("GO_REPETITION_ARGUMENTS", "expected --case ID or --package P --profile general|sensitive --count N --run REGEX --expect ENTRY [...]");
	}
	const packagePath = argv[1];
	const profile = argv[3];
	const count = Number(argv[5]);
	const run = argv[7];
	const expected = [];
	for (let index = 8; index < argv.length; index += 2) {
		if (argv[index] !== "--expect") fail("GO_REPETITION_ARGUMENTS", `unexpected token ${argv[index]}`);
		expected.push(argv[index + 1]);
	}
	if (typeof packagePath !== "string" || !packagePath.startsWith(`${modulePath}/`) ||
		!(/^github\.com\/nelsonwerd\/countershape(?:\/[A-Za-z0-9_.-]+)+$/u.test(packagePath)) || packagePath.split("/").includes("..")) {
		fail("GO_REPETITION_PACKAGE", String(packagePath));
	}
	if (!(profile === "general" || profile === "sensitive")) fail("GO_REPETITION_PROFILE", String(profile));
	const requiredProfile = sensitiveGoPackages.includes(packagePath) ? "sensitive" : "general";
	if (profile !== requiredProfile) {
		fail("GO_REPETITION_PROFILE_CLASS", `${packagePath}: supplied=${profile} required=${requiredProfile}`);
	}
	if (!Number.isSafeInteger(count) || count < 1 || count > 50) fail("GO_REPETITION_COUNT_ARGUMENT", String(argv[5]));
	if (typeof run !== "string" || run.length < 3 || run.length > 4096 || !run.startsWith("^") || !run.endsWith("$") ||
		/[\u0000-\u001f\u007f]/u.test(run)) fail("GO_REPETITION_RUN_PATTERN", String(run));
	if (expected.length === 0 || new Set(expected).size !== expected.length ||
		expected.some((name) => !expectedEntrypointPattern.test(name))) fail("GO_REPETITION_EXPECTED_ROSTER", expected.join(","));
	return Object.freeze({
		caseID: null,
		count,
		expected: Object.freeze([...expected].sort()),
		packagePath,
		profile,
		qualification: false,
		run,
	});
}

function validateExecutionSpecification(specification) {
	const exactKeys = ["caseID", "count", "expected", "packagePath", "profile", "qualification", "run"];
	if (!specification || typeof specification !== "object" || Array.isArray(specification) ||
		JSON.stringify(Object.keys(specification).sort()) !== JSON.stringify([...exactKeys].sort()) ||
		!Array.isArray(specification.expected)) {
		fail("GO_REPETITION_SPECIFICATION", "exact execution specification shape required");
	}
	if (typeof specification.caseID === "string") {
		const canonical = qualificationCases[specification.caseID];
		if (!canonical || canonical.count !== specification.count || canonical.packagePath !== specification.packagePath ||
			canonical.profile !== specification.profile || canonical.qualification !== specification.qualification ||
			canonical.run !== specification.run || JSON.stringify(canonical.expected) !== JSON.stringify(specification.expected)) {
			fail("GO_REPETITION_SPECIFICATION", `qualification case mismatch: ${specification.caseID}`);
		}
		return canonical;
	}
	if (specification.caseID !== null || specification.qualification !== false) {
		fail("GO_REPETITION_SPECIFICATION", "ad hoc execution cannot claim qualification");
	}
	const argv = [
		"--package", specification.packagePath,
		"--profile", specification.profile,
		"--count", String(specification.count),
		"--run", specification.run,
		...specification.expected.flatMap((name) => ["--expect", name]),
	];
	const canonical = parseRunArguments(argv);
	if (JSON.stringify(canonical) !== JSON.stringify(specification)) {
		fail("GO_REPETITION_SPECIFICATION", "ad hoc execution specification is not canonical");
	}
	return canonical;
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
		if (error instanceof GoRepetitionError && error.code === code) return;
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
	const platformName = dependencies.platform ?? platform();
	const architecture = dependencies.arch ?? arch();
	if (platformName !== "darwin" || architecture !== "arm64") fail("GO_REPETITION_PLATFORM", `${platformName}/${architecture}`);
	const acquire = dependencies.acquireVerificationLock ?? acquireVerificationLock;
	const admit = dependencies.admitTools ?? admitTools;
	const createRoots = dependencies.createPrivateRoots ?? createPrivateRoots;
	const makeEnvironment = dependencies.buildChildEnvironment ?? buildChildEnvironment;
	const executeChild = dependencies.childResult ?? childResult;
	const finalize = dependencies.finalizeVerificationResources ?? finalizeVerificationResources;
	const remove = dependencies.remove ?? rm;
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
			const cleanup = [];
			if (roots) {
				try { await remove(roots.runRoot, { recursive: true, force: true }); } catch (error) { cleanup.push(error); }
			}
			try { await lock.release(); } catch (error) { cleanup.push(error); }
			if (cleanup.length > 0) failure = new AggregateError([...(failure ? [failure] : []), ...cleanup], "Go repetition verification cleanup failed");
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
	if (success <= cursor || !joined.includes("qualification=false") || !joined.includes("authorities_sha256:")) {
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
			async remove() { cleanup.push("remove"); },
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
		async acquireVerificationLock() { return { async release() { throw new Error("release failed"); } }; },
		async admitTools() { return admitted; },
		async createPrivateRoots() { return roots; },
		buildChildEnvironment() { return {}; },
		async childResult() { return { status: 1, signal: null, error: null, stdout: "", stderr: "" }; },
		async remove() { throw new Error("remove failed"); },
		write() {},
	}), (error) => error instanceof AggregateError && error.errors.length === 3, "aggregate cleanup failure");
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
	const parsed = parseRunArguments([
		"--package", `${modulePath}/internal/example`, "--profile", "general", "--count", "2",
		"--run", "^(ExampleAlpha|FuzzAlpha|TestAlpha)$", "--expect", "TestAlpha", "--expect", "FuzzAlpha", "--expect", "ExampleAlpha",
	]);
	if (parsed.expected.join(",") !== "ExampleAlpha,FuzzAlpha,TestAlpha") fail("GO_REPETITION_SELFTEST_ARGUMENTS", parsed.expected.join(","));
	if (parseRunArguments(["--case", "parity-full-package-3"]) !== qualificationCases["parity-full-package-3"] ||
		!qualificationCases["parity-full-package-3"].expected.includes("FuzzParseContractParityCorpusLine")) {
		fail("GO_REPETITION_SELFTEST_QUALIFICATION_CASE", "parity-full-package-3");
	}
	const cliPhysicalShards = qualificationCaseIDs.filter((id) => /^cli-physical-reducer-[0-9]{2}-of-20$/u.test(id));
	const httpPhysicalShards = qualificationCaseIDs.filter((id) => /^http-physical-reducer-[0-9]{2}-of-20$/u.test(id));
	if (qualificationCaseIDs.length !== 52 || cliPhysicalShards.length !== 20 || httpPhysicalShards.length !== 20 ||
		[...cliPhysicalShards, ...httpPhysicalShards].some((id) => qualificationCases[id].count !== 1)) {
		fail("GO_REPETITION_SELFTEST_QUALIFICATION_SHARDS", JSON.stringify({ cliPhysicalShards, httpPhysicalShards }));
	}
	expectCode(() => parseRunArguments(["--case", "unknown"]), "GO_REPETITION_QUALIFICATION_CASE");
	expectCode(() => parseRunArguments([
		"--package", `${modulePath}/internal/world`, "--profile", "general", "--count", "2",
		"--run", "^TestAlpha$", "--expect", "TestAlpha",
	]), "GO_REPETITION_PROFILE_CLASS");
	expectCode(() => parseRunArguments([
		"--package", `${modulePath}/internal/emit/node/parity`, "--profile", "sensitive", "--count", "2",
		"--run", "^TestAlpha$", "--expect", "TestAlpha",
	]), "GO_REPETITION_PROFILE_CLASS");
	await expectAsyncFailure(() => runRepetition({ ...specification, qualification: true }, {
		platform: "darwin",
		arch: "arm64",
		async acquireVerificationLock() { throw new Error("forged qualification reached lock acquisition"); },
	}), (error) => error instanceof GoRepetitionError && error.code === "GO_REPETITION_SPECIFICATION", "forged ad hoc qualification");
	await expectAsyncFailure(() => runRepetition({ ...qualificationCases["world-output-caps-50"], count: 1 }, {
		platform: "darwin",
		arch: "arm64",
		async acquireVerificationLock() { throw new Error("mutated qualification reached lock acquisition"); },
	}), (error) => error instanceof GoRepetitionError && error.code === "GO_REPETITION_SPECIFICATION", "mutated named qualification");
	await compositionSelfTest(specification, cleanText);
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

const invoked = process.argv[1] ? await realpath(resolve(process.argv[1])) : "";
if (invoked === fileURLToPath(import.meta.url)) {
	main().catch((error) => {
		process.stderr.write(`${error.stack ?? error}\n`);
		process.exitCode = 1;
	});
}
