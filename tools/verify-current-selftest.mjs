#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawn, spawnSync } from "node:child_process";
import { chmod, lstat, mkdir, mkdtemp, open, readFile, readdir, realpath, rm, symlink, unlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import {
	VerificationError,
	assertNoDSStore,
	c5SensitiveGoPackages,
	childArguments,
	childExecutionPolicyForStepID,
	currentSteps,
	currentStepChildResult,
	dispatchCurrentStep,
	exactU7ArchitectureOutput,
	exactU7ArchitectureSelftestOutput,
	exactU7PhaseContracts,
	executeCurrentPlan,
	historicalOnly,
	loadU7PhaseCapsuleFromIndex,
	packageArguments,
	parseU7PhaseCapsule,
	partitionGoPackages,
	revalidatePackagePartition,
	rootAuthorityForStep,
	rosterDigest,
	sealedC6AArchitectureStepIDs,
	sensitiveGoPackages,
	validateU7PhaseCapsule,
	validateRepositoryPlan,
} from "./verify-current.mjs";
import { parsePhaseCapsule as parsePlanPhaseCapsule } from "./check-u7-plan.mjs";
import {
	VerificationRuntimeError,
	acquireVerificationLock,
	admitTool,
	admitTools,
	buildChildEnvironment,
	childResult,
	cleanupVerificationResources,
	createPrivateBase,
	createPrivateRoots,
	finalizeVerificationResources,
	processLiveness,
	repositoryRoot,
} from "./verify-runtime-authority.mjs";

const selftestPath = fileURLToPath(import.meta.url);
const verifierPath = resolve(dirname(selftestPath), "verify-current.mjs");
const runtimePath = resolve(dirname(selftestPath), "verify-runtime-authority.mjs");
const sealedC6ARunnerPath = resolve(dirname(selftestPath), "check-sealed-c6a-architecture.mjs");
const u7PlanPath = resolve(dirname(selftestPath), "check-u7-plan.mjs");
const authorityNames = Object.freeze(["go", "node", "git", "sh", "cc", "cxx"]);
const sealedC4VHistoricalRosterDigest = "7cf294fcbe8247e7a4de2d8996fb6b960780a35d018a8d00069af15940f0efb8";
const sealedC4HParentRosterDigest = "0fab61318d55314e3accaeabbaf56cc049aae5755f3c3f41d6a36eba1a6c7af5";
const sealedCombinedC5RosterDigest = "3821722f937f647ec98f03170cf9efb1e9a51a98e63d361d38dae58ade9ffaed";
const expectedRosterDigest = "ffc33c263c231c5ea2f3d427f271d45ab4a7afa445f57d19ef954086c702e51c";
const expectedHistoricalDigest = "0de1d88c9674a87b096ecdc772d9cc2a0b96fff2cda9fa9ed3eefe78ccec16e2";
const exactC5StepIDs = Object.freeze([
	"architecture-p07b-c-c5",
	"architecture-p07b-c-c5-selftest",
	"go-json-p07b-c-c5-http-behavior",
	"go-json-p07b-c-c5-readiness-teardown",
	"go-json-p07b-c-c5-scope-closure",
	"go-json-p07b-c-c5-cross-profile-parity",
	"go-json-p07b-c-c5-http-authority-race",
]);
const exactC6StepIDs = Object.freeze([
	"architecture-p07b-c-c6",
	"architecture-p07b-c-c6-selftest",
]);
const exactU7StepIDs = Object.freeze([
	"architecture-u7",
	"architecture-u7-plan-selftest",
	"architecture-u7-scope-selftest",
	"architecture-u7-study-harness-protocol",
	"architecture-u7-study-harness-selftest",
	"u7-final-runbook-selftest",
	"architecture-u7-selftest",
]);
const exactExtendedChildTimeoutStepIDs = Object.freeze([
	"architecture-p07b-c-c5",
	"architecture-p07b-c-c5-selftest",
]);
const exactInheritedSensitivePackages = Object.freeze([
	"github.com/nelsonwerd/countershape/internal/contractexec/runner",
	"github.com/nelsonwerd/countershape/internal/emit/node/compiler",
	"github.com/nelsonwerd/countershape/internal/emit/node/program/v1",
	"github.com/nelsonwerd/countershape/internal/processmechanics",
	"github.com/nelsonwerd/countershape/internal/store",
	"github.com/nelsonwerd/countershape/internal/world",
	"github.com/nelsonwerd/countershape/testkit/contractexec/cli",
	"github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
	"github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
]);
const exactC5SensitivePackages = Object.freeze([
	"github.com/nelsonwerd/countershape/internal/contractexec/http",
	"github.com/nelsonwerd/countershape/internal/contractexec/scope",
	"github.com/nelsonwerd/countershape/testkit/contractexec/http",
]);

function fail(code, detail) {
	throw new Error(`${code}: ${detail}`);
}

function expect(condition, code, detail) {
	if (!condition) fail(code, detail);
}

function occurrenceCount(body, snippet) {
	return body.split(snippet).length - 1;
}

function expectPlainCode(invoke, code) {
	try {
		invoke();
	} catch (error) {
		expect(
			typeof error?.message === "string" && error.message.startsWith(`${code}:`),
			"VERIFY_SELFTEST_WRONG_PLAIN_ERROR",
			`${code}: ${error?.stack ?? error}`,
		);
		return;
	}
	fail("VERIFY_SELFTEST_FALSE_NEGATIVE", code);
}

function assertC5Roster(ids, sensitive, c5Sensitive) {
	if (exactC5StepIDs.some((id) => !ids.includes(id))) {
		fail("VERIFY_SELFTEST_C5_BLOCK_OMISSION", ids.join(","));
	}
	const c5Set = new Set(exactC5StepIDs);
	const actualC5Order = ids.filter((id) => c5Set.has(id));
	if (JSON.stringify(actualC5Order) !== JSON.stringify(exactC5StepIDs)) {
		fail("VERIFY_SELFTEST_C5_BLOCK_ORDER", actualC5Order.join(","));
	}
	const start = ids.indexOf(exactC5StepIDs[0]);
	const positions = exactC5StepIDs.map((id) => ids.indexOf(id));
	if (start !== ids.indexOf("go-json-p07b-c-c4-authority-race") + 1 ||
		positions.some((position, index) => position !== start + index) ||
		ids[start + exactC5StepIDs.length] !== exactC6StepIDs[0]) {
		fail("VERIFY_SELFTEST_C5_BLOCK_CONTIGUITY", ids.join(","));
	}
	if (JSON.stringify(sensitive) !== JSON.stringify(exactInheritedSensitivePackages)) {
		fail("VERIFY_SELFTEST_INHERITED_SENSITIVE_ROSTER", sensitive.join(","));
	}
	if (JSON.stringify(c5Sensitive) !== JSON.stringify(exactC5SensitivePackages)) {
		fail("VERIFY_SELFTEST_C5_SENSITIVE_ROSTER", c5Sensitive.join(","));
	}
	const inheritedSensitiveRow = ids.indexOf("go-test-sensitive-serial");
	const c5SensitiveRow = ids.indexOf("go-test-sensitive-c5-serial");
	if (inheritedSensitiveRow < 0 || c5SensitiveRow !== inheritedSensitiveRow + 1 ||
		ids[c5SensitiveRow + 1] !== "go-package-partition-revalidation") {
		fail("VERIFY_SELFTEST_SENSITIVE_ROW_ADJACENCY", ids.join(","));
	}
}

function assertC6Roster(ids) {
	for (const id of exactC6StepIDs) {
		if (ids.filter((candidate) => candidate === id).length > 1) {
			fail("VERIFY_SELFTEST_C6_BLOCK_DUPLICATION", ids.join(","));
		}
	}
	if (exactC6StepIDs.some((id) => !ids.includes(id))) {
		fail("VERIFY_SELFTEST_C6_BLOCK_OMISSION", ids.join(","));
	}
	const c6Set = new Set(exactC6StepIDs);
	const actualC6Order = ids.filter((id) => c6Set.has(id));
	if (JSON.stringify(actualC6Order) !== JSON.stringify(exactC6StepIDs)) {
		fail("VERIFY_SELFTEST_C6_BLOCK_ORDER", actualC6Order.join(","));
	}
	const start = ids.indexOf(exactC6StepIDs[0]);
	const positions = exactC6StepIDs.map((id) => ids.indexOf(id));
	if (start !== ids.indexOf(exactC5StepIDs.at(-1)) + 1 ||
		positions.some((position, index) => position !== start + index) ||
		ids[start + exactC6StepIDs.length] !== "architecture-p07b-c-inherited-compatibility") {
		fail("VERIFY_SELFTEST_C6_BLOCK_CONTIGUITY", ids.join(","));
	}
	if (ids.length !== 71) fail("VERIFY_SELFTEST_CURRENT_STEP_COUNT", String(ids.length));
}

function assertU7Roster(ids) {
	for (const id of exactU7StepIDs) {
		if (ids.filter((candidate) => candidate === id).length !== 1) {
			fail("VERIFY_SELFTEST_U7_CARDINALITY", `${id}:${ids.join(",")}`);
		}
	}
	if (ids[0] !== "workspace-no-ds-store" || ids[1] !== "architecture-u7" || ids[2] !== "go-package-partition") {
		fail("VERIFY_SELFTEST_U7_EARLY_PLACEMENT", ids.slice(0, 4).join(","));
	}
	const late = exactU7StepIDs.slice(1);
	const lateStart = ids.indexOf(late[0]);
	if (lateStart !== ids.indexOf("architecture-p07b-c-c3p-receipt") + 1 ||
		late.some((id, index) => ids[lateStart + index] !== id) ||
		ids[lateStart + late.length] !== "workspace-no-ds-store-terminal") {
		fail("VERIFY_SELFTEST_U7_LATE_BLOCK", ids.join(","));
	}
}

function assertRetiredP07Rows(currentIDs, historical = historicalOnly) {
	const expectedRows = [
		{ id: "architecture-p07b-c-plan-selftest", scripts: ["tools/check-p07b-c-plan.mjs"], status: "docs/status/U7P-AUTHORITY.md" },
		{ id: "architecture-p07b-c-c3p-receipt-selftest", scripts: ["tools/check-p07b-c-c3p-receipt.mjs"], status: "docs/status/U7P-AUTHORITY.md" },
	];
	const historicalIDs = historical.map((row) => row.id);
	for (const expected of expectedRows) {
		const matches = historical.filter((candidate) => candidate.id === expected.id);
		if (matches.length !== 1) fail("VERIFY_SELFTEST_RETIRED_P07_ROW_CARDINALITY", `${expected.id}:${matches.length}`);
	}
	for (const expected of expectedRows) {
		const row = historical.find((candidate) => candidate.id === expected.id);
		if (JSON.stringify(row.scripts) !== JSON.stringify(expected.scripts) || row.status !== expected.status) {
			fail("VERIFY_SELFTEST_RETIRED_P07_ROW_DRIFT", expected.id);
		}
		if (currentIDs.includes(expected.id)) fail("VERIFY_SELFTEST_RETIRED_P07_ROW_REINTRODUCED", expected.id);
	}
	if (JSON.stringify(historicalIDs.slice(-expectedRows.length)) !== JSON.stringify(expectedRows.map((row) => row.id))) {
		fail("VERIFY_SELFTEST_RETIRED_P07_ROW_ORDER", historicalIDs.join(","));
	}
	if (currentIDs.filter((id) => id === "architecture-p07b-c-c3p-receipt").length !== 1) {
		fail("VERIFY_SELFTEST_LIVE_C3P_RECEIPT_MISSING", currentIDs.join(","));
	}
}

function assertFinderGuardRoster(ids) {
	const exactGuardIDs = ["workspace-no-ds-store", "workspace-no-ds-store-terminal"];
	const actualGuardIDs = ids.filter((id) => exactGuardIDs.includes(id));
	if (actualGuardIDs.length !== exactGuardIDs.length ||
		new Set(actualGuardIDs).size !== exactGuardIDs.length) {
		fail("VERIFY_SELFTEST_FINDER_GUARD_CARDINALITY", actualGuardIDs.join(","));
	}
	if (JSON.stringify(actualGuardIDs) !== JSON.stringify(exactGuardIDs)) {
		fail("VERIFY_SELFTEST_FINDER_GUARD_ORDER", actualGuardIDs.join(","));
	}
	if (ids[0] !== exactGuardIDs[0] ||
		ids.at(-3) !== exactGuardIDs[1] ||
		ids.at(-2) !== "authority-revalidation" ||
		ids.at(-1) !== "verification-resource-finalization") {
		fail("VERIFY_SELFTEST_FINDER_GUARD_PLACEMENT", ids.join(","));
	}
}

async function expectCode(promise, code) {
	try {
		await promise;
	} catch (error) {
		if ((error instanceof VerificationError || error instanceof VerificationRuntimeError) && error.code === code) return;
		fail("VERIFY_SELFTEST_WRONG_ERROR", `${code}: ${error.stack ?? error}`);
	}
	fail("VERIFY_SELFTEST_FALSE_NEGATIVE", code);
}

function capturedChild(command, args, options) {
	const child = spawn(command, args, { ...options, stdio: ["pipe", "pipe", "pipe"] });
	const capture = {
		child, stdout: "", stderr: "", stdoutBytes: 0, stderrBytes: 0,
		failure: null, didClose: false, closed: null, completed: null,
	};
	let resolveClosed;
	capture.closed = new Promise((resolvePromise) => {
		resolveClosed = resolvePromise;
	});
	const rejectCapture = (code, detail) => {
		if (capture.failure !== null || capture.didClose) return;
		capture.failure = new Error(`${code}: ${detail}`);
		if (child.exitCode === null && child.signalCode === null) child.kill("SIGKILL");
	};
	const append = (name, chunk) => {
		const bytesName = `${name}Bytes`;
		capture[bytesName] += chunk.length;
		capture[name] += chunk.toString("utf8");
		if (capture[bytesName] > 64 * 1024) rejectCapture("VERIFY_SELFTEST_CHILD_OUTPUT_LIMIT", name);
	};
	child.stdout.on("data", (chunk) => append("stdout", chunk));
	child.stderr.on("data", (chunk) => append("stderr", chunk));
	child.once("error", (error) => rejectCapture("VERIFY_SELFTEST_CHILD_SPAWN", error.message));
	child.stdout.once("error", (error) => rejectCapture("VERIFY_SELFTEST_CHILD_STDOUT", error.message));
	child.stderr.once("error", (error) => rejectCapture("VERIFY_SELFTEST_CHILD_STDERR", error.message));
	child.stdin.once("error", (error) => rejectCapture("VERIFY_SELFTEST_CHILD_STDIN", error.message));
	child.once("close", (status, signal) => {
		capture.didClose = true;
		resolveClosed({ status, signal });
	});
	capture.completed = capture.closed.then((result) => {
		if (capture.failure !== null) throw capture.failure;
		return result;
	});
	void capture.completed.catch(() => {});
	return capture;
}

function waitForChildMarker(capture, marker, timeoutMS = 10_000) {
	if (capture.stdout.includes(marker)) return Promise.resolve();
	return new Promise((resolvePromise, rejectPromise) => {
		let finished = false;
		const timeout = setTimeout(() => {
			finish(rejectPromise, new Error(`timed out waiting for ${marker}: ${capture.stdout}${capture.stderr}`));
		}, timeoutMS);
		const onData = () => {
			if (!capture.stdout.includes(marker)) return;
			finish(resolvePromise);
		};
		const cleanup = () => {
			clearTimeout(timeout);
			capture.child.stdout.off("data", onData);
		};
		const finish = (settle, value) => {
			if (finished) return;
			finished = true;
			cleanup();
			settle(value);
		};
		capture.child.stdout.on("data", onData);
		capture.completed.then(
			({ status, signal }) => finish(
				rejectPromise,
				new Error(`child closed before ${marker}: status=${status} signal=${signal}: ${capture.stdout}${capture.stderr}`),
			),
			(error) => finish(rejectPromise, error),
		);
	});
}

async function waitForChildExit(capture, timeoutMS = 10_000) {
	let timeout;
	try {
		return await Promise.race([
			capture.completed,
			new Promise((_, rejectPromise) => {
				timeout = setTimeout(() => {
					rejectPromise(new Error(`timed out waiting for child close: ${capture.stdout}${capture.stderr}`));
				}, timeoutMS);
			}),
		]);
	} finally {
		clearTimeout(timeout);
	}
}

function fakeAuthorities() {
	return Object.freeze(Object.fromEntries(authorityNames.map((name) => [
		name,
		Object.freeze({ name, path: `/authority/${name}`, sha256: name.padEnd(64, "0").slice(0, 64) }),
	])));
}

async function inspectRosters() {
	const digest = rosterDigest();
	expect(digest === expectedRosterDigest, "VERIFY_SELFTEST_ROSTER_DRIFT", `${digest} != ${expectedRosterDigest}`);
	expect(
		createHash("sha256").update(JSON.stringify(historicalOnly)).digest("hex") === expectedHistoricalDigest,
		"VERIFY_SELFTEST_HISTORICAL_ROSTER_DRIFT",
		JSON.stringify(historicalOnly),
	);
	expect(digest !== sealedC4VHistoricalRosterDigest, "VERIFY_SELFTEST_C4V_ROSTER_NOT_EVOLVED", digest);
	expect(digest !== sealedC4HParentRosterDigest, "VERIFY_SELFTEST_C5_ROSTER_NOT_EVOLVED", digest);
	expect(digest !== sealedCombinedC5RosterDigest, "VERIFY_SELFTEST_C5_SENSITIVE_SPLIT_NOT_EVOLVED", digest);
	const currentIDs = currentSteps.map((step) => step.id);
	expect(new Set(currentIDs).size === currentIDs.length, "VERIFY_SELFTEST_DUPLICATE_CURRENT", currentIDs.join(","));
	assertC5Roster(currentIDs, sensitiveGoPackages, c5SensitiveGoPackages);
	assertC6Roster(currentIDs);
	assertU7Roster(currentIDs);
	assertFinderGuardRoster(currentIDs);
	const sealedRows = currentSteps.filter((step) => rootAuthorityForStep(step) === "SEALED_C6A");
	expect(
		JSON.stringify(sealedRows.map((step) => step.id)) === JSON.stringify(sealedC6AArchitectureStepIDs) &&
		sealedRows.every((step) => step.path === "tools/check-sealed-c6a-architecture.mjs") &&
		currentSteps.filter((step) => step.path === "tools/check-sealed-c6a-architecture.mjs").length === 2,
		"VERIFY_SELFTEST_ROOT_AUTHORITY_ROSTER",
		sealedRows.map((step) => `${step.id}:${step.path}`).join(","),
	);
	const sealedBaseline = sealedRows[0];
	for (const [name, hostile] of [
		["missing authority", { ...sealedBaseline, rootAuthority: undefined }],
		["wrong authority", { ...sealedBaseline, rootAuthority: "LIVE" }],
		["wrong path", { ...sealedBaseline, path: "tools/check-p07b-c-architecture.mjs" }],
		["wrong mode", { ...sealedBaseline, args: ["--c6-selftest"] }],
	]) {
		expectPlainCode(() => rootAuthorityForStep(hostile), "VERIFY_ROOT_AUTHORITY_CONTRACT");
		void name;
	}
	expectPlainCode(
		() => rootAuthorityForStep({ ...currentSteps.find((step) => step.id === "go-build"), rootAuthority: "SEALED_C6A" }),
		"VERIFY_ROOT_AUTHORITY_UNEXPECTED",
	);
	expectPlainCode(
		() => assertC5Roster(
			currentIDs.filter((id) => id !== "go-json-p07b-c-c5-scope-closure"),
			sensitiveGoPackages,
			c5SensitiveGoPackages,
		),
		"VERIFY_SELFTEST_C5_BLOCK_OMISSION",
	);
	const reorderedC5 = [...currentIDs];
	const readinessIndex = reorderedC5.indexOf("go-json-p07b-c-c5-readiness-teardown");
	const scopeIndex = reorderedC5.indexOf("go-json-p07b-c-c5-scope-closure");
	[reorderedC5[readinessIndex], reorderedC5[scopeIndex]] =
		[reorderedC5[scopeIndex], reorderedC5[readinessIndex]];
	expectPlainCode(
		() => assertC5Roster(reorderedC5, sensitiveGoPackages, c5SensitiveGoPackages),
		"VERIFY_SELFTEST_C5_BLOCK_ORDER",
	);
	const interposedC5 = [...currentIDs];
	const interposedIndex = interposedC5.indexOf("architecture-p07b-c-unit-scope-selftest");
	const [interposed] = interposedC5.splice(interposedIndex, 1);
	interposedC5.splice(interposedC5.indexOf("go-json-p07b-c-c5-scope-closure"), 0, interposed);
	expectPlainCode(
		() => assertC5Roster(interposedC5, sensitiveGoPackages, c5SensitiveGoPackages),
		"VERIFY_SELFTEST_C5_BLOCK_CONTIGUITY",
	);
	expectPlainCode(
		() => assertC6Roster(currentIDs.filter((id) => id !== exactC6StepIDs[1])),
		"VERIFY_SELFTEST_C6_BLOCK_OMISSION",
	);
	const reorderedC6 = [...currentIDs];
	const c6Index = reorderedC6.indexOf(exactC6StepIDs[0]);
	const c6SelftestIndex = reorderedC6.indexOf(exactC6StepIDs[1]);
	[reorderedC6[c6Index], reorderedC6[c6SelftestIndex]] =
		[reorderedC6[c6SelftestIndex], reorderedC6[c6Index]];
	expectPlainCode(
		() => assertC6Roster(reorderedC6),
		"VERIFY_SELFTEST_C6_BLOCK_ORDER",
	);
	const duplicatedC6 = [...currentIDs];
	duplicatedC6.splice(duplicatedC6.indexOf(exactC6StepIDs[0]), 0, exactC6StepIDs[0]);
	expectPlainCode(
		() => assertC6Roster(duplicatedC6),
		"VERIFY_SELFTEST_C6_BLOCK_DUPLICATION",
	);
	const interposedC6 = [...currentIDs];
	const c6InterpositionIndex = interposedC6.indexOf("architecture-p07b-c-unit-scope-selftest");
	const [c6Interposition] = interposedC6.splice(c6InterpositionIndex, 1);
	interposedC6.splice(interposedC6.indexOf(exactC6StepIDs[1]), 0, c6Interposition);
	expectPlainCode(
		() => assertC6Roster(interposedC6),
		"VERIFY_SELFTEST_C6_BLOCK_CONTIGUITY",
	);
	expectPlainCode(
		() => assertC6Roster([...currentIDs, "unadmitted-extra-stage"]),
		"VERIFY_SELFTEST_CURRENT_STEP_COUNT",
	);
	for (const id of exactU7StepIDs) {
		expectPlainCode(
			() => assertU7Roster(currentIDs.filter((candidate) => candidate !== id)),
			"VERIFY_SELFTEST_U7_CARDINALITY",
		);
		const duplicated = [...currentIDs];
		duplicated.splice(duplicated.indexOf(id), 0, id);
		expectPlainCode(() => assertU7Roster(duplicated), "VERIFY_SELFTEST_U7_CARDINALITY");
	}
	const lateReordered = [...currentIDs];
	const planIndex = lateReordered.indexOf("architecture-u7-plan-selftest");
	const scopeIndexU7 = lateReordered.indexOf("architecture-u7-scope-selftest");
	[lateReordered[planIndex], lateReordered[scopeIndexU7]] = [lateReordered[scopeIndexU7], lateReordered[planIndex]];
	expectPlainCode(() => assertU7Roster(lateReordered), "VERIFY_SELFTEST_U7_LATE_BLOCK");
	const earlyMoved = [...currentIDs];
	const [earlyU7] = earlyMoved.splice(earlyMoved.indexOf("architecture-u7"), 1);
	earlyMoved.splice(earlyMoved.indexOf("go-build") + 1, 0, earlyU7);
	expectPlainCode(() => assertU7Roster(earlyMoved), "VERIFY_SELFTEST_U7_EARLY_PLACEMENT");
	const lateInterposed = [...currentIDs];
	const [interposition] = lateInterposed.splice(lateInterposed.indexOf("architecture-u6"), 1);
	lateInterposed.splice(lateInterposed.indexOf("architecture-u7-study-harness-protocol"), 0, interposition);
	expectPlainCode(() => assertU7Roster(lateInterposed), "VERIFY_SELFTEST_U7_LATE_BLOCK");
	expectPlainCode(
		() => assertC5Roster(currentIDs, sensitiveGoPackages.slice(1), c5SensitiveGoPackages),
		"VERIFY_SELFTEST_INHERITED_SENSITIVE_ROSTER",
	);
	const reorderedSensitive = [...sensitiveGoPackages];
	[reorderedSensitive[0], reorderedSensitive[1]] = [reorderedSensitive[1], reorderedSensitive[0]];
	expectPlainCode(
		() => assertC5Roster(currentIDs, reorderedSensitive, c5SensitiveGoPackages),
		"VERIFY_SELFTEST_INHERITED_SENSITIVE_ROSTER",
	);
	expectPlainCode(
		() => assertC5Roster(currentIDs, sensitiveGoPackages, c5SensitiveGoPackages.slice(1)),
		"VERIFY_SELFTEST_C5_SENSITIVE_ROSTER",
	);
	const reorderedC5Sensitive = [...c5SensitiveGoPackages];
	[reorderedC5Sensitive[0], reorderedC5Sensitive[1]] =
		[reorderedC5Sensitive[1], reorderedC5Sensitive[0]];
	expectPlainCode(
		() => assertC5Roster(currentIDs, sensitiveGoPackages, reorderedC5Sensitive),
		"VERIFY_SELFTEST_C5_SENSITIVE_ROSTER",
	);
	expectPlainCode(
		() => assertC5Roster(
			currentIDs.filter((id) => id !== "go-test-sensitive-c5-serial"),
			sensitiveGoPackages,
			c5SensitiveGoPackages,
		),
		"VERIFY_SELFTEST_SENSITIVE_ROW_ADJACENCY",
	);
	const interposedSensitiveRows = [...currentIDs];
	const generalRow = interposedSensitiveRows.splice(interposedSensitiveRows.indexOf("go-test-general"), 1)[0];
	interposedSensitiveRows.splice(interposedSensitiveRows.indexOf("go-test-sensitive-c5-serial"), 0, generalRow);
	expectPlainCode(
		() => assertC5Roster(interposedSensitiveRows, sensitiveGoPackages, c5SensitiveGoPackages),
		"VERIFY_SELFTEST_SENSITIVE_ROW_ADJACENCY",
	);
	expectPlainCode(
		() => assertFinderGuardRoster(currentIDs.filter((id) => id !== "workspace-no-ds-store-terminal")),
		"VERIFY_SELFTEST_FINDER_GUARD_CARDINALITY",
	);
	const reorderedFinderGuards = [...currentIDs];
	const openingFinderIndex = reorderedFinderGuards.indexOf("workspace-no-ds-store");
	const terminalFinderIndex = reorderedFinderGuards.indexOf("workspace-no-ds-store-terminal");
	[reorderedFinderGuards[openingFinderIndex], reorderedFinderGuards[terminalFinderIndex]] =
		[reorderedFinderGuards[terminalFinderIndex], reorderedFinderGuards[openingFinderIndex]];
	expectPlainCode(
		() => assertFinderGuardRoster(reorderedFinderGuards),
		"VERIFY_SELFTEST_FINDER_GUARD_ORDER",
	);
	expectPlainCode(
		() => assertFinderGuardRoster([...currentIDs, "workspace-no-ds-store-terminal"]),
		"VERIFY_SELFTEST_FINDER_GUARD_CARDINALITY",
	);
	const interposedFinderGuard = [...currentIDs];
	const interposedFinderRow = interposedFinderGuard.splice(
		interposedFinderGuard.indexOf("architecture-p07b-c-c3p-receipt"),
		1,
	)[0];
	interposedFinderGuard.splice(interposedFinderGuard.indexOf("authority-revalidation"), 0, interposedFinderRow);
	expectPlainCode(
		() => assertFinderGuardRoster(interposedFinderGuard),
		"VERIFY_SELFTEST_FINDER_GUARD_PLACEMENT",
	);
	const finalizationSteps = currentSteps.filter((step) => step.kind === "finalization-guard");
	expect(
		finalizationSteps.length === 1 && currentSteps.at(-1) === finalizationSteps[0],
		"VERIFY_SELFTEST_FINALIZATION_NOT_UNIQUE_LAST",
		currentIDs.join(","),
	);
	const authoritySteps = currentSteps.filter((step) => step.kind === "authority-guard");
	expect(
		authoritySteps.length === 1 && currentSteps.at(-2) === authoritySteps[0],
		"VERIFY_SELFTEST_AUTHORITY_NOT_UNIQUE_PENULTIMATE",
		currentIDs.join(","),
	);
	const artifactGuardSteps = currentSteps.filter((step) => step.kind === "guard");
	expect(
		artifactGuardSteps.length === 2 &&
			artifactGuardSteps[0].id === "workspace-no-ds-store" &&
			artifactGuardSteps[1].id === "workspace-no-ds-store-terminal",
		"VERIFY_SELFTEST_FINDER_GUARD_KIND_DRIFT",
		artifactGuardSteps.map((step) => step.id).join(","),
	);
	const historicalIDs = historicalOnly.map((row) => row.id);
	expect(new Set(historicalIDs).size === historicalIDs.length, "VERIFY_SELFTEST_DUPLICATE_HISTORICAL", historicalIDs.join(","));
	assertRetiredP07Rows(currentIDs);
	for (const id of ["architecture-p07b-c-plan-selftest", "architecture-p07b-c-c3p-receipt-selftest"]) {
		expectPlainCode(() => assertRetiredP07Rows(currentIDs, historicalOnly.filter((row) => row.id !== id)), "VERIFY_SELFTEST_RETIRED_P07_ROW_CARDINALITY");
		expectPlainCode(() => assertRetiredP07Rows(currentIDs, [...historicalOnly, historicalOnly.find((row) => row.id === id)]), "VERIFY_SELFTEST_RETIRED_P07_ROW_CARDINALITY");
		expectPlainCode(() => assertRetiredP07Rows([...currentIDs, id]), "VERIFY_SELFTEST_RETIRED_P07_ROW_REINTRODUCED");
	}
	const wrongRetiredScript = historicalOnly.map((row) => row.id === "architecture-p07b-c-c3p-receipt-selftest" ? { ...row, scripts: ["tools/check-p07b-c-plan.mjs"] } : row);
	expectPlainCode(() => assertRetiredP07Rows(currentIDs, wrongRetiredScript), "VERIFY_SELFTEST_RETIRED_P07_ROW_DRIFT");
	const wrongRetiredStatus = historicalOnly.map((row) => row.id === "architecture-p07b-c-plan-selftest" ? { ...row, status: "docs/status/U1.md" } : row);
	expectPlainCode(() => assertRetiredP07Rows(currentIDs, wrongRetiredStatus), "VERIFY_SELFTEST_RETIRED_P07_ROW_DRIFT");
	const reorderedRetired = [...historicalOnly];
	const penultimateRetired = reorderedRetired.length - 2;
	[reorderedRetired[penultimateRetired], reorderedRetired[penultimateRetired + 1]] = [reorderedRetired[penultimateRetired + 1], reorderedRetired[penultimateRetired]];
	expectPlainCode(() => assertRetiredP07Rows(currentIDs, reorderedRetired), "VERIFY_SELFTEST_RETIRED_P07_ROW_ORDER");
	expectPlainCode(() => assertRetiredP07Rows(currentIDs.filter((id) => id !== "architecture-p07b-c-c3p-receipt")), "VERIFY_SELFTEST_LIVE_C3P_RECEIPT_MISSING");
	const knownTools = new Set(authorityNames);
	for (const step of currentSteps.filter((candidate) => candidate.tool)) {
		const names = step.tools;
		expect(Array.isArray(names) && names.length > 0, "VERIFY_SELFTEST_STEP_TOOL_ROSTER", step.id);
		expect(names[0] === step.tool, "VERIFY_SELFTEST_PRIMARY_TOOL_MISMATCH", step.id);
		expect(new Set(names).size === names.length, "VERIFY_SELFTEST_DUPLICATE_STEP_TOOL", `${step.id}: ${names.join(",")}`);
		expect(names.every((name) => knownTools.has(name)), "VERIFY_SELFTEST_UNKNOWN_STEP_TOOL", `${step.id}: ${names.join(",")}`);
	}
	const exactStep = (id, expected) => {
		const step = currentSteps.find((candidate) => candidate.id === id);
		expect(step !== undefined, "VERIFY_SELFTEST_C1_STEP_MISSING", id);
		for (const [field, value] of Object.entries(expected)) {
			expect(JSON.stringify(step[field]) === JSON.stringify(value), "VERIFY_SELFTEST_C1_STEP_DRIFT", `${id}:${field}`);
		}
	};
	exactStep("go-package-partition", {
		kind: "package-guard",
		tool: "go",
		tools: ["go"],
		args: ["list", "-mod=readonly", "-buildvcs=false", "./..."],
		marker: "PACKAGE_PARTITION exact",
	});
	exactStep("go-build", {
		tool: "go",
		tools: ["go", "cc", "cxx"],
		args: ["build", "-mod=readonly", "-buildvcs=false", "-p=1", "./..."],
	});
	exactStep("go-vet", {
		tool: "go",
		tools: ["go", "cc", "cxx"],
		args: ["vet", "-mod=readonly", "-buildvcs=false", "-p=1", "./..."],
	});
	exactStep("go-test-general", {
		tool: "go",
		tools: ["go", "node", "git", "sh", "cc", "cxx"],
		packageClass: "general",
		args: ["test", "-mod=readonly", "-buildvcs=false", "-p=1", "-parallel=2", "-count=1", "-timeout=20m"],
	});
	exactStep("go-test-sensitive-serial", {
		tool: "go",
		tools: ["go", "node", "git", "sh", "cc", "cxx"],
		packageClass: "sensitive",
		args: ["test", "-mod=readonly", "-buildvcs=false", "-p=1", "-parallel=2", "-count=1", "-timeout=20m"],
	});
	exactStep("go-test-sensitive-c5-serial", {
		tool: "go",
		tools: ["go", "node", "git", "sh", "cc", "cxx"],
		packageClass: "c5Sensitive",
		args: ["test", "-mod=readonly", "-buildvcs=false", "-p=1", "-parallel=2", "-count=1", "-timeout=20m"],
	});
	exactStep("go-package-partition-revalidation", {
		kind: "package-revalidation",
		tool: "go",
		tools: ["go"],
		args: ["list", "-mod=readonly", "-buildvcs=false", "./..."],
		marker: "PACKAGE_PARTITION_REVALIDATED exact",
	});
	exactStep("go-repetition-runner-selftest", {
		tool: "node",
		tools: ["node"],
		path: "tools/verify-go-test-repetition.mjs",
		args: ["--self-test"],
		marker: "Go repetition verifier self-test passed:",
	});
	exactStep("verification-resource-finalization", { kind: "finalization-guard" });
	expect(JSON.stringify(sensitiveGoPackages) === JSON.stringify([
		"github.com/nelsonwerd/countershape/internal/contractexec/runner",
		"github.com/nelsonwerd/countershape/internal/emit/node/compiler",
		"github.com/nelsonwerd/countershape/internal/emit/node/program/v1",
		"github.com/nelsonwerd/countershape/internal/processmechanics",
		"github.com/nelsonwerd/countershape/internal/store",
		"github.com/nelsonwerd/countershape/internal/world",
		"github.com/nelsonwerd/countershape/testkit/contractexec/cli",
		"github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
		"github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
	]), "VERIFY_SELFTEST_SENSITIVE_PACKAGE_ROSTER", sensitiveGoPackages.join(","));
	expect(JSON.stringify(c5SensitiveGoPackages) === JSON.stringify([
		"github.com/nelsonwerd/countershape/internal/contractexec/http",
		"github.com/nelsonwerd/countershape/internal/contractexec/scope",
		"github.com/nelsonwerd/countershape/testkit/contractexec/http",
	]), "VERIFY_SELFTEST_C5_SENSITIVE_PACKAGE_ROSTER", c5SensitiveGoPackages.join(","));
	exactStep("planning-example-p07b-a2-2", {
		tool: "node",
		tools: ["node", "go"],
		path: "tools/generate-p07-planning-example.mjs",
		args: ["--exercise"],
	});
	exactStep("planning-validator-selftest", {
		tool: "node",
		tools: ["node", "go"],
		path: "tools/validate-planning.mjs",
		args: ["--self-test"],
		marker: "planning validator self-test: ok (",
	});
	exactStep("architecture-p07b-b", {
		tool: "node",
		tools: ["node", "go", "cc", "cxx"],
		path: "tools/check-p07b-b-architecture.mjs",
		marker: "P07B B architecture boundary OK",
	});
	exactStep("architecture-p07b-b-selftest", {
		tool: "node",
		tools: ["node", "go", "cc", "cxx"],
		path: "tools/check-p07b-b-architecture-selftest.mjs",
		marker: "P07B B architecture defensive self-test OK",
	});
	exactStep("architecture-p07b-c-c1", {
		tool: "node",
		tools: ["node", "go"],
		path: "tools/check-p07b-c-architecture.mjs",
		args: ["--c1"],
		marker: "P07B-C C1 architecture boundary OK",
	});
	exactStep("architecture-p07b-c-c1-selftest", {
		tool: "node",
		tools: ["node", "go"],
		path: "tools/check-p07b-c-architecture-selftest.mjs",
		args: ["--c1"],
		marker: "P07B-C C1 architecture defensive self-test OK",
	});
	exactStep("architecture-p07b-c-c2", {
		tool: "node",
		tools: ["node", "go", "cc", "cxx"],
		path: "tools/check-p07b-c-architecture.mjs",
		args: ["--c2"],
		marker: "P07B-C C2 cumulative architecture boundary OK",
	});
	exactStep("architecture-p07b-c-c2-selftest", {
		tool: "node",
		tools: ["node", "go", "cc", "cxx"],
		path: "tools/check-p07b-c-architecture-selftest.mjs",
		args: ["--c2"],
		marker: "P07B-C C2 cumulative architecture defensive self-test OK",
	});
	for (const [id, profile, count] of [
		["go-json-p07b-c-c2-nonhead", "c2-nonhead-persistence", 6],
		["go-json-p07b-c-c2-interlock", "c2-interlock", 7],
		["go-json-p07b-c-c2-private-evidence", "c2-private-evidence", 6],
		["go-json-p07b-c-c2-public-surface", "c2-public-surface", 2],
	]) {
		exactStep(id, {
			tool: "node",
			tools: ["node", "go", "cc", "cxx"],
			path: "tools/check-p07b-c-architecture.mjs",
			args: ["--run-go-json", profile],
			marker: `P07B-C C2 Go JSON target execution OK (${profile}: ${count} passed, 0 skipped)`,
		});
	}
	exactStep("architecture-p07b-c-c4", {
		tool: "node",
		tools: ["node", "go", "git", "sh", "cc", "cxx"],
		path: "tools/check-p07b-c-architecture.mjs",
		args: ["--c4"],
		marker: "P07B-C C4 cumulative architecture boundary OK",
	});
	exactStep("architecture-p07b-c-c4-selftest", {
		tool: "node",
		tools: ["node", "go", "git", "sh", "cc", "cxx"],
		path: "tools/check-p07b-c-architecture-selftest.mjs",
		args: ["--c4"],
		marker: "P07B-C C4 cumulative architecture defensive self-test OK (14 metadata cases; 42 Go JSON parser cases; 7 command cases; 5 owner-reference parser cases)",
	});
	for (const [id, profile, count] of [
		["go-json-p07b-c-c4-processmechanics-parity", "c4-processmechanics-parity", 12],
		["go-json-p07b-c-c4-admission-permit", "c4-admission-permit", 7],
		["go-json-p07b-c-c4-cli-closure", "c4-cli-closure", 4],
		["go-json-p07b-c-c4-finalized-run-release", "c4-finalized-run-release", 5],
		["go-json-p07b-c-c4-classification-recovery", "c4-classification-recovery", 2],
		["go-json-p07b-c-c4-authority-race", "c4-authority-race", 4],
	]) {
		exactStep(id, {
			tool: "node",
			tools: ["node", "go", "git", "sh", "cc", "cxx"],
			path: "tools/check-p07b-c-architecture.mjs",
			args: ["--run-go-json", profile],
			marker: `P07B-C C4 Go JSON target execution OK (${profile}: ${count} passed, 0 skipped)`,
		});
	}
	exactStep("architecture-p07b-c-c5", {
		tool: "node",
		tools: ["node", "go", "git", "sh", "cc", "cxx"],
		path: "tools/check-p07b-c-architecture.mjs",
		args: ["--c5"],
		marker: "P07B-C C5 cumulative architecture boundary OK",
	});
	exactStep("architecture-p07b-c-c5-selftest", {
		tool: "node",
		tools: ["node", "go", "git", "sh", "cc", "cxx"],
		path: "tools/check-p07b-c-architecture-selftest.mjs",
		args: ["--c5"],
		marker: "P07B-C C5 cumulative architecture defensive self-test OK (14 metadata cases; 60 Go JSON parser cases; 6 command cases)",
	});
	for (const [id, profile, count] of [
		["go-json-p07b-c-c5-http-behavior", "c5-http-behavior", 4],
		["go-json-p07b-c-c5-readiness-teardown", "c5-readiness-teardown", 9],
		["go-json-p07b-c-c5-scope-closure", "c5-scope-closure", 9],
		["go-json-p07b-c-c5-cross-profile-parity", "c5-cross-profile-parity", 22],
		["go-json-p07b-c-c5-http-authority-race", "c5-http-authority-race", 5],
	]) {
		exactStep(id, {
			tool: "node",
			tools: ["node", "go", "git", "sh", "cc", "cxx"],
			path: "tools/check-p07b-c-architecture.mjs",
			args: ["--run-go-json", profile],
			marker: `P07B-C C5 Go JSON target execution OK (${profile}: ${count} passed, 0 skipped)`,
		});
	}
	exactStep("architecture-p07b-c-c6", {
		tool: "node",
		tools: ["node", "go", "git", "sh", "cc", "cxx"],
		path: "tools/check-sealed-c6a-architecture.mjs",
		args: ["--c6"],
		rootAuthority: "SEALED_C6A",
		marker: "P07B-C C6 cumulative architecture boundary OK",
	});
	exactStep("architecture-p07b-c-c6-selftest", {
		tool: "node",
		tools: ["node", "go", "git", "sh", "cc", "cxx"],
		path: "tools/check-sealed-c6a-architecture.mjs",
		args: ["--c6-selftest"],
		rootAuthority: "SEALED_C6A",
		marker: "P07B-C C6 cumulative architecture defensive self-test OK (11 metadata cases; 22 Go JSON lifecycle cases)",
	});
	exactStep("architecture-p07b-c-c3", {
		tool: "node",
		tools: ["node", "go", "git", "sh", "cc", "cxx"],
		path: "tools/check-p07b-c-architecture.mjs",
		args: ["--c3"],
		marker: "P07B-C C3 cumulative architecture boundary OK",
	});
	exactStep("architecture-p07b-c-c3-selftest", {
		tool: "node",
		tools: ["node", "go", "git", "sh", "cc", "cxx"],
		path: "tools/check-p07b-c-architecture-selftest.mjs",
		args: ["--c3"],
		marker: "P07B-C C3 cumulative architecture defensive self-test OK (12 metadata cases; 35 Go JSON parser cases; 58 raw predecessor/parser cases)",
	});
	for (const [id, profile, count] of [
		["go-json-p07b-c-c3-official-target", "c3-official-target", 10],
		["go-json-p07b-c-c3-single-target", "c3-single-target", 7],
		["go-json-p07b-c-c3-hostepoch", "c3-hostepoch", 8],
		["go-json-p07b-c-c3-noderuntime", "c3-noderuntime", 12],
		["go-json-p07b-c-c3-store-bridge", "c3-store-bridge", 6],
	]) {
		exactStep(id, {
			tool: "node",
			tools: ["node", "go", "git", "sh", "cc", "cxx"],
			path: "tools/check-p07b-c-architecture.mjs",
			args: ["--run-go-json", profile],
			marker: `P07B-C C3 Go JSON target execution OK (${profile}: ${count} passed, 0 skipped)`,
		});
	}
	exactStep("architecture-p07b-c-inherited-compatibility", {
		tool: "node",
		tools: ["node", "go", "git", "sh", "cc", "cxx"],
		path: "tools/check-u7-plan.mjs",
		args: ["--verify-inherited-p07-compatibility"],
		marker: "U7 inherited P07 compatibility exact:",
		exactStdout: "U7 inherited P07 compatibility exact: parent=C6B blocks=11 protected_paths=4 legacy_live_selftest=RETIRED_SUCCESSOR_INCOMPATIBLE\n",
	});
	exactStep("architecture-p07b-c-c3p-receipt", {
		tool: "node",
		tools: ["node", "git"],
		path: "tools/check-p07b-c-c3p-receipt.mjs",
		marker: "P07B-C C3P receipt check passed: phase-specific source/receipt, scope, Git-note, and documentation authority are coherent",
	});
	exactStep("architecture-u7", {
		kind: "u7-phase-architecture",
		tool: "node",
		tools: ["node"],
		path: "tools/check-u7-architecture.mjs",
		marker: "U7_ARCHITECTURE_OK phase=",
	});
	exactStep("architecture-u7-plan-selftest", {
		tool: "node",
		tools: ["node", "go", "git", "sh", "cc", "cxx"],
		path: "tools/check-u7-plan.mjs",
		args: ["--self-test"],
		marker: "U7 plan contract defensive self-test passed:",
	});
	exactStep("architecture-u7-scope-selftest", {
		tool: "node",
		tools: ["node", "go", "git", "sh", "cc", "cxx"],
		path: "tools/check-u7-scope.mjs",
		args: ["--self-test"],
		marker: "U7 independent staged-scope defensive self-test passed:",
	});
	exactStep("architecture-u7-study-harness-protocol", {
		tool: "node",
		tools: ["node", "go", "git"],
		path: "tools/check-u7-study-harness.mjs",
		args: ["--protocol-check"],
		marker: "U7_STUDY_HARNESS_PROTOCOL_OK protocol=countershape/u7-study-harness/v1 digest=",
	});
	exactStep("architecture-u7-study-harness-selftest", {
		tool: "node",
		tools: ["node", "go", "git", "sh", "cc", "cxx"],
		path: "tools/check-u7-study-harness-selftest.mjs",
		args: ["--phase", "U7P"],
		marker: "U7 study harness defensive self-test passed: active=U7P cases=",
	});
	exactStep("u7-final-runbook-selftest", {
		tool: "node",
		tools: ["node", "go", "git", "sh", "cc", "cxx"],
		path: "tools/print-u7-final-runbook.mjs",
		args: ["--self-test"],
		marker: "U7 final runbook renderer defensive self-test passed:",
	});
	exactStep("architecture-u7-selftest", {
		kind: "u7-phase-architecture-selftest",
		tool: "node",
		tools: ["node"],
		path: "tools/check-u7-architecture-selftest.mjs",
		marker: "U7 architecture defensive self-test passed: active=",
	});

	const toolEntries = (await readdir(resolve(repositoryRoot, "tools"))).sort();
	const mutationFiles = toolEntries.filter((name) => /^(?:mutate|test-mutate)-.+\.mjs$/u.test(name)).map((name) => `tools/${name}`);
	const admittedMutations = historicalOnly.flatMap((row) => row.scripts).filter((path) => /\/(?:mutate|test-mutate)-/u.test(path)).sort();
	expect(
		JSON.stringify(mutationFiles) === JSON.stringify(admittedMutations),
		"VERIFY_SELFTEST_ORPHAN_MUTATION_GATE",
		`actual=${mutationFiles.join(",")} admitted=${admittedMutations.join(",")}`,
	);

	const architectureFiles = toolEntries.filter((name) => /^check-.+\.mjs$/u.test(name)).map((name) => `tools/${name}`);
	const admittedArchitecture = [...new Set([
		...currentSteps.filter((step) => step.id.startsWith("architecture-")).map((step) => step.path),
		...historicalOnly.filter((row) => row.id.startsWith("architecture-")).flatMap((row) => row.scripts),
	])].sort();
	expect(
		JSON.stringify(architectureFiles) === JSON.stringify(admittedArchitecture),
		"VERIFY_SELFTEST_ORPHAN_ARCHITECTURE_GATE",
		`actual=${architectureFiles.join(",")} admitted=${admittedArchitecture.join(",")}`,
	);
}

function inspectEnvironment() {
	const admitted = fakeAuthorities();
	const roots = {
		home: "/private/home", tmp: "/private/tmp", gotmp: "/private/gotmp", gocache: "/private/gocache",
		gopath: "/private/gopath", gomodcache: "/private/gomodcache", authorityBin: "/private/authority-bin",
	};
	const environment = buildChildEnvironment(admitted, roots);
	const expected = {
		HOME: roots.home, TMPDIR: roots.tmp, GOTMPDIR: roots.gotmp, GOCACHE: roots.gocache, GOPATH: roots.gopath,
		GOMODCACHE: roots.gomodcache, GOENV: "off", GOWORK: "off", GOTOOLCHAIN: "local", GOPROXY: "off",
		GOSUMDB: "off", GOVCS: "*:off", GOFLAGS: "-mod=readonly -buildvcs=false -p=1", CGO_ENABLED: "1",
		CC: admitted.cc.path, CXX: admitted.cxx.path, GOMAXPROCS: "2", LANG: "C", LC_ALL: "C", TZ: "UTC",
		NO_COLOR: "1", PATH: `${roots.authorityBin}:/usr/bin:/bin`, COUNTERSHAPE_GO: admitted.go.path,
		COUNTERSHAPE_NODE: admitted.node.path, COUNTERSHAPE_GIT: admitted.git.path, COUNTERSHAPE_SH: admitted.sh.path,
		COUNTERSHAPE_CC: admitted.cc.path, COUNTERSHAPE_CXX: admitted.cxx.path,
	};
	expect(JSON.stringify(environment) === JSON.stringify(expected), "VERIFY_SELFTEST_CHILD_ENVIRONMENT_DRIFT", JSON.stringify(environment));
}

async function inspectPackagePartition() {
	const general = "github.com/nelsonwerd/countershape/internal/canon";
	const completeSensitive = [...sensitiveGoPackages, ...c5SensitiveGoPackages].sort();
	const shuffled = [
		c5SensitiveGoPackages[1],
		sensitiveGoPackages[3],
		general,
		...completeSensitive.filter((path) =>
			path !== c5SensitiveGoPackages[1] && path !== sensitiveGoPackages[3]),
	];
	const partition = partitionGoPackages(`${shuffled.join("\n")}\n`);
	expect(partition.general.length === 1 && partition.general[0] === general, "VERIFY_SELFTEST_PACKAGE_GENERAL", partition.general.join(","));
	expect(
		JSON.stringify(partition.sensitive) === JSON.stringify(sensitiveGoPackages) &&
			JSON.stringify(partition.c5Sensitive) === JSON.stringify(c5SensitiveGoPackages) &&
			partition.all.length === completeSensitive.length + 1 &&
			new Set([...partition.general, ...partition.sensitive, ...partition.c5Sensitive]).size === partition.all.length,
		"VERIFY_SELFTEST_PACKAGE_EXACT_UNION",
		JSON.stringify(partition),
	);
	const generalStep = currentSteps.find((step) => step.id === "go-test-general");
	const sensitiveStep = currentSteps.find((step) => step.id === "go-test-sensitive-serial");
	const c5SensitiveStep = currentSteps.find((step) => step.id === "go-test-sensitive-c5-serial");
	expect(
		JSON.stringify(packageArguments(generalStep, partition)) === JSON.stringify([...generalStep.args, general]),
		"VERIFY_SELFTEST_GENERAL_PACKAGE_DISPATCH",
		JSON.stringify(packageArguments(generalStep, partition)),
	);
	expect(
		JSON.stringify(packageArguments(sensitiveStep, partition)) === JSON.stringify([...sensitiveStep.args, ...sensitiveGoPackages]),
		"VERIFY_SELFTEST_SENSITIVE_PACKAGE_DISPATCH",
		JSON.stringify(packageArguments(sensitiveStep, partition)),
	);
	expect(
		JSON.stringify(packageArguments(c5SensitiveStep, partition)) ===
			JSON.stringify([...c5SensitiveStep.args, ...c5SensitiveGoPackages]),
		"VERIFY_SELFTEST_C5_SENSITIVE_PACKAGE_DISPATCH",
		JSON.stringify(packageArguments(c5SensitiveStep, partition)),
	);
	expect(
		JSON.stringify(packageArguments({ args: ["list", "./..."] }, partition)) === JSON.stringify(["list", "./..."]),
		"VERIFY_SELFTEST_NONPARTITION_ARGUMENT_DISPATCH",
		"non-partition step changed",
	);
	revalidatePackagePartition(
		partition,
		partitionGoPackages(`${[general, ...completeSensitive].join("\n")}\n`),
	);
	await expectCode(
		Promise.resolve().then(() => packageArguments({ id: "missing", packageClass: "general", args: ["test"] })),
		"VERIFY_PACKAGE_PARTITION_UNAVAILABLE",
	);
	await expectCode(
		Promise.resolve().then(() => packageArguments({ id: "unknown", packageClass: "all", args: ["test"] }, partition)),
		"VERIFY_PACKAGE_PARTITION_UNAVAILABLE",
	);
	for (const [name, changed] of [
		["all", [...partition.all, "github.com/nelsonwerd/countershape/new-package"]],
		["general", [...partition.general, "github.com/nelsonwerd/countershape/new-package"]],
		["sensitive", sensitiveGoPackages.slice(1)],
		["c5Sensitive", c5SensitiveGoPackages.slice(1)],
	]) {
		await expectCode(
			Promise.resolve().then(() => revalidatePackagePartition(partition, {
				...partition,
				[name]: changed,
			})),
			"VERIFY_PACKAGE_PARTITION_CHANGED",
		);
	}
	const reorderedSensitive = [...sensitiveGoPackages];
	[reorderedSensitive[0], reorderedSensitive[1]] =
		[reorderedSensitive[1], reorderedSensitive[0]];
	const reorderedC5Sensitive = [...c5SensitiveGoPackages];
	[reorderedC5Sensitive[0], reorderedC5Sensitive[1]] =
		[reorderedC5Sensitive[1], reorderedC5Sensitive[0]];
	const swappedSensitive = [...sensitiveGoPackages];
	const swappedC5Sensitive = [...c5SensitiveGoPackages];
	[swappedSensitive[0], swappedC5Sensitive[0]] =
		[swappedC5Sensitive[0], swappedSensitive[0]];
	swappedSensitive.sort();
	swappedC5Sensitive.sort();
	const overlappingC5Sensitive = [
		c5SensitiveGoPackages[0],
		sensitiveGoPackages[0],
		c5SensitiveGoPackages[1],
	].sort();
	for (const [name, invoke, code] of [
		[
			"missing-inherited-package",
			() => partitionGoPackages(`${[general, ...completeSensitive.filter((path) => path !== sensitiveGoPackages[0])].join("\n")}\n`),
			"VERIFY_SENSITIVE_PACKAGE_ROSTER_INVALID",
		],
		[
			"missing-c5-package",
			() => partitionGoPackages(`${[general, ...completeSensitive.filter((path) => path !== c5SensitiveGoPackages[0])].join("\n")}\n`),
			"VERIFY_C5_SENSITIVE_PACKAGE_ROSTER_INVALID",
		],
		[
			"reordered-inherited",
			() => partitionGoPackages(`${[general, ...completeSensitive].join("\n")}\n`, reorderedSensitive, c5SensitiveGoPackages),
			"VERIFY_SENSITIVE_PACKAGE_ROSTER_INVALID",
		],
		[
			"reordered-c5",
			() => partitionGoPackages(`${[general, ...completeSensitive].join("\n")}\n`, sensitiveGoPackages, reorderedC5Sensitive),
			"VERIFY_C5_SENSITIVE_PACKAGE_ROSTER_INVALID",
		],
		[
			"duplicate-inherited",
			() => partitionGoPackages(
				`${[general, ...completeSensitive].join("\n")}\n`,
				[sensitiveGoPackages[0], ...sensitiveGoPackages.slice(0, -1)].sort(),
				c5SensitiveGoPackages,
			),
			"VERIFY_SENSITIVE_PACKAGE_ROSTER_INVALID",
		],
		[
			"duplicate-c5",
			() => partitionGoPackages(
				`${[general, ...completeSensitive].join("\n")}\n`,
				sensitiveGoPackages,
				[c5SensitiveGoPackages[0], c5SensitiveGoPackages[0], c5SensitiveGoPackages[1]].sort(),
			),
			"VERIFY_C5_SENSITIVE_PACKAGE_ROSTER_INVALID",
		],
		[
			"swapped-subgroups",
			() => partitionGoPackages(
				`${[general, ...completeSensitive].join("\n")}\n`,
				swappedSensitive,
				swappedC5Sensitive,
			),
			"VERIFY_SENSITIVE_PACKAGE_ROSTER_INVALID",
		],
		[
			"overlapping-subgroups",
			() => partitionGoPackages(
				`${[general, ...completeSensitive].join("\n")}\n`,
				sensitiveGoPackages,
				overlappingC5Sensitive,
			),
			"VERIFY_SENSITIVE_PACKAGE_GROUPS_OVERLAP",
		],
		[
			"foreign",
			() => partitionGoPackages(`${[general, ...completeSensitive, "example.invalid/foreign"].join("\n")}\n`),
			"VERIFY_PACKAGE_LIST_INVALID",
		],
		[
			"duplicate-go-list",
			() => partitionGoPackages(`${[general, general, ...completeSensitive].join("\n")}\n`),
			"VERIFY_PACKAGE_LIST_INVALID",
		],
		[
			"missing-final-lf",
			() => partitionGoPackages([general, ...completeSensitive].join("\n")),
			"VERIFY_PACKAGE_LIST_INVALID",
		],
	]) {
		await expectCode(Promise.resolve().then(invoke), code);
		expect(name.length > 0, "VERIFY_SELFTEST_PACKAGE_CASE_NAME", name);
	}
}

async function inspectChildArguments() {
	expect(
		JSON.stringify(childArguments({ path: "tools/example.mjs", args: ["--check"] })) ===
			JSON.stringify([resolve(repositoryRoot, "tools/example.mjs"), "--check"]),
		"VERIFY_SELFTEST_PATH_ARGUMENTS_DROPPED",
		"path-backed step did not retain its explicit mode",
	);
	expect(
		JSON.stringify(childArguments({ args: ["test", "./..."] })) === JSON.stringify(["test", "./..."]),
		"VERIFY_SELFTEST_TOOL_ARGUMENTS_DRIFT",
		"tool-only arguments changed",
	);
	let overriddenArguments;
	let defaultSpawnOptions;
	await childResult(
		{ id: "argument-override", tool: "go", tools: ["go"] },
		fakeAuthorities(),
		{},
		{
			args: ["test", "example.invalid/package"],
			revalidate: async () => {},
			spawn: (_executable, args, options) => {
				overriddenArguments = args;
				defaultSpawnOptions = options;
				return { status: 0, signal: null, error: null, stdout: "", stderr: "" };
			},
		},
	);
	expect(
		JSON.stringify(overriddenArguments) === JSON.stringify(["test", "example.invalid/package"]),
		"VERIFY_SELFTEST_CHILD_ARGUMENT_OVERRIDE",
		JSON.stringify(overriddenArguments),
	);
	expect(
		defaultSpawnOptions.cwd === repositoryRoot && defaultSpawnOptions.encoding === "utf8" &&
			defaultSpawnOptions.env !== undefined && defaultSpawnOptions.timeout === 1_200_000 &&
			defaultSpawnOptions.maxBuffer === 64 * 1024 * 1024,
		"VERIFY_SELFTEST_CHILD_DEFAULT_OPTIONS",
		JSON.stringify(defaultSpawnOptions),
	);
	const exactExtendedChildTimeoutStepIDSet = new Set(exactExtendedChildTimeoutStepIDs);
	for (const step of currentSteps) {
		const policy = childExecutionPolicyForStepID(step.id);
		expect(
			(policy !== undefined) === exactExtendedChildTimeoutStepIDSet.has(step.id),
			"VERIFY_SELFTEST_CURRENT_TIMEOUT_POLICY_SELECTION",
			`${step.id}: ${JSON.stringify(policy)}`,
		);
	}
	for (const id of ["constructor", "toString", "__proto__", "architecture-p07b-c-c5-alias", "", null, undefined]) {
		expect(
			childExecutionPolicyForStepID(id) === undefined,
			"VERIFY_SELFTEST_TIMEOUT_POLICY_LOOKUP",
			String(id),
		);
	}
	for (const id of exactExtendedChildTimeoutStepIDs) {
		const policy = childExecutionPolicyForStepID(id);
		expect(
			policy !== undefined && Object.isFrozen(policy) &&
				Object.getPrototypeOf(policy) === null &&
				JSON.stringify(Object.keys(policy)) === JSON.stringify(["timeoutMS"]) &&
				policy.timeoutMS === 1_800_000,
			"VERIFY_SELFTEST_TIMEOUT_POLICY_EXACT",
			`${id}: ${JSON.stringify(policy)}`,
		);
		let extendedSpawnOptions;
		let extendedSpawns = 0;
		let extendedRevalidations = 0;
		await currentStepChildResult(
			{ id, tool: "node", tools: ["node"] },
			fakeAuthorities(),
			{},
			{
				revalidate: async () => { extendedRevalidations += 1; },
				spawn: (_executable, _args, options) => {
					extendedSpawns += 1;
					extendedSpawnOptions = options;
					return { status: 0, signal: null, error: null, stdout: "", stderr: "" };
				},
			},
		);
		expect(
			extendedSpawns === 1 && extendedRevalidations === 2 &&
				extendedSpawnOptions.timeout === 1_800_000 &&
				extendedSpawnOptions.cwd === repositoryRoot &&
				extendedSpawnOptions.encoding === "utf8" &&
				extendedSpawnOptions.maxBuffer === 64 * 1024 * 1024,
			"VERIFY_SELFTEST_TIMEOUT_POLICY_PROPAGATION",
			`${id}: spawns=${extendedSpawns} revalidations=${extendedRevalidations} options=${JSON.stringify(extendedSpawnOptions)}`,
		);
	}
	let inertMetadataTimeout;
	await currentStepChildResult(
		{ id: "metadata-cannot-elevate", tool: "node", tools: ["node"], timeoutMS: 1_800_000 },
		fakeAuthorities(),
		{},
		{
			revalidate: async () => {},
			spawn: (_executable, _args, options) => {
				inertMetadataTimeout = options.timeout;
				return { status: 0, signal: null, error: null, stdout: "", stderr: "" };
			},
		},
	);
	expect(inertMetadataTimeout === 1_200_000, "VERIFY_SELFTEST_STEP_METADATA_ELEVATED", String(inertMetadataTimeout));
	const exactExtendedPolicy = Object.freeze({ timeoutMS: 1_800_000 });
	let directExtendedTimeout;
	await childResult(
		{ id: "direct-extended", tool: "node", tools: ["node"] },
		fakeAuthorities(),
		{},
		{
			revalidate: async () => {},
			spawn: (_executable, _args, options) => {
				directExtendedTimeout = options.timeout;
				return { status: 0, signal: null, error: null, stdout: "", stderr: "" };
			},
		},
		exactExtendedPolicy,
	);
	expect(directExtendedTimeout === 1_800_000, "VERIFY_SELFTEST_DIRECT_EXTENDED_TIMEOUT", String(directExtendedTimeout));
	const inheritedPolicy = Object.freeze(Object.create(Object.freeze({ timeoutMS: 1_800_000 })));
	const accessorPolicy = {};
	Object.defineProperty(accessorPolicy, "timeoutMS", { enumerable: true, get: () => 1_800_000 });
	Object.freeze(accessorPolicy);
	const symbolPolicy = Object.freeze({ timeoutMS: 1_800_000, [Symbol("extra")]: true });
	for (const [name, policy] of [
		["explicit-undefined", undefined],
		["null", null],
		["array", Object.freeze([1_800_000])],
		["unfrozen", { timeoutMS: 1_800_000 }],
		["inherited", inheritedPolicy],
		["accessor", accessorPolicy],
		["extra", Object.freeze({ timeoutMS: 1_800_000, extra: true })],
		["symbol", symbolPolicy],
		["string", Object.freeze({ timeoutMS: "1800000" })],
		["fraction", Object.freeze({ timeoutMS: 1_800_000.5 })],
		["nan", Object.freeze({ timeoutMS: Number.NaN })],
		["infinity", Object.freeze({ timeoutMS: Number.POSITIVE_INFINITY })],
		["bigint", Object.freeze({ timeoutMS: 1_800_000n })],
		["zero", Object.freeze({ timeoutMS: 0 })],
		["negative", Object.freeze({ timeoutMS: -1 })],
		["default-explicit", Object.freeze({ timeoutMS: 1_200_000 })],
		["under", Object.freeze({ timeoutMS: 1_799_999 })],
		["over", Object.freeze({ timeoutMS: 1_800_001 })],
	]) {
		let spawned = false;
		let revalidated = false;
		await expectCode(
			childResult(
				{ id: `invalid-policy-${name}`, tool: "node", tools: ["node"] },
				fakeAuthorities(),
				{},
				{
					revalidate: async () => { revalidated = true; },
					spawn: () => {
						spawned = true;
						return { status: 0, signal: null, error: null, stdout: "", stderr: "" };
					},
				},
				policy,
			),
			"VERIFY_CHILD_EXECUTION_POLICY_INVALID",
		);
		expect(!spawned && !revalidated, "VERIFY_SELFTEST_INVALID_POLICY_USED_AUTHORITY", name);
	}
	const authorities = fakeAuthorities();
	const recordStepTools = async (step, _admitted, events) => {
		for (const name of step.tools) events.push(name);
	};
	for (const scenario of [
		{
			name: "nonzero",
			spawnResult: { status: 23, signal: null, error: null, stdout: "", stderr: "injected nonzero" },
		},
		{
			name: "spawn-error",
			spawnResult: { status: null, signal: null, error: new Error("injected spawn error"), stdout: "", stderr: "" },
		},
		{
			name: "timeout",
			spawnResult: Object.assign(
				{ status: null, signal: "SIGTERM", stdout: "partial-out", stderr: "partial-err" },
				{ error: Object.assign(new Error("injected timeout"), { code: "ETIMEDOUT" }) },
			),
		},
	]) {
		const events = [];
		const result = await childResult(
			{ id: scenario.name, tool: "node", tools: ["node", "go"], path: "tools/fixture.mjs" },
			authorities,
			{},
			{
				revalidate: async (step, admitted) => recordStepTools(step, admitted, events),
				spawn: () => { events.push("spawn"); return scenario.spawnResult; },
			},
		);
		expect(
			result.status === scenario.spawnResult.status && result.signal === scenario.spawnResult.signal &&
				result.error === scenario.spawnResult.error && result.stdout === scenario.spawnResult.stdout &&
				result.stderr === scenario.spawnResult.stderr,
			"VERIFY_SELFTEST_CHILD_RESULT_IDENTITY",
			scenario.name,
		);
		expect(
			JSON.stringify(events) === JSON.stringify(["node", "go", "spawn", "node", "go"]),
			"VERIFY_SELFTEST_CHILD_REVALIDATION_ORDER",
			`${scenario.name}: ${events.join(",")}`,
		);
	}
	const thrownEvents = [];
	try {
		await childResult(
			{ id: "spawn-throw", tool: "node", tools: ["node", "go"], path: "tools/fixture.mjs" },
			authorities,
			{},
			{
				revalidate: async (step, admitted) => recordStepTools(step, admitted, thrownEvents),
				spawn: () => { thrownEvents.push("spawn"); throw new Error("injected spawn throw"); },
			},
		);
		fail("VERIFY_SELFTEST_FALSE_NEGATIVE", "spawn throw");
	} catch (error) {
		expect(error.message === "injected spawn throw", "VERIFY_SELFTEST_WRONG_SPAWN_THROW", error.stack ?? error);
	}
	expect(
		JSON.stringify(thrownEvents) === JSON.stringify(["node", "go", "spawn", "node", "go"]),
		"VERIFY_SELFTEST_CHILD_THROW_REVALIDATION_ORDER",
		thrownEvents.join(","),
	);
	let aggregateRevalidations = 0;
	try {
		await childResult(
			{ id: "spawn-and-revalidation-fail", tool: "node", tools: ["node"] },
			authorities,
			{},
			{
				revalidate: async () => {
					aggregateRevalidations += 1;
					if (aggregateRevalidations === 2) throw new VerificationRuntimeError("VERIFY_TOOL_REVALIDATION_FAILED", "injected");
				},
				spawn: () => { throw new Error("injected spawn failure"); },
			},
		);
		fail("VERIFY_SELFTEST_FALSE_NEGATIVE", "spawn and authority revalidation aggregation");
	} catch (error) {
		expect(
			error instanceof AggregateError && error.errors.length === 2 &&
				error.errors[0].message === "injected spawn failure" && error.errors[1].code === "VERIFY_TOOL_REVALIDATION_FAILED",
			"VERIFY_SELFTEST_CHILD_FAILURE_NOT_AGGREGATED",
			error.stack ?? error,
		);
	}
	await expectCode(
		childResult({ id: "unadmitted", tool: "node", tools: ["node"] }, Object.freeze({}), {}),
		"VERIFY_TOOL_NOT_ADMITTED",
	);
	await expectCode(
		childResult({ id: "missing-roster", tool: "node" }, authorities, {}),
		"VERIFY_PLAN_TOOL_ROSTER_REQUIRED",
	);
	await expectCode(
		childResult({ id: "wrong-primary", tool: "node", tools: ["go", "node"] }, authorities, {}),
		"VERIFY_PLAN_PRIMARY_TOOL_MISMATCH",
	);
	await expectCode(
		childResult({ id: "duplicate", tool: "node", tools: ["node", "node"] }, authorities, {}),
		"VERIFY_PLAN_TOOL_DUPLICATE",
	);
	await expectCode(
		childResult({ id: "unknown", tool: "node", tools: ["node", "python"] }, authorities, {}),
		"VERIFY_PLAN_TOOL_UNKNOWN",
	);
}

async function inspectFailClosedExecution() {
	const authorities = fakeAuthorities();
	const successfulCalls = [];
	let successfulOutput = "";
	let clockValue = 0;
	const successfulStatus = await executeCurrentPlan({
		admitted: authorities,
		childEnvironment: {},
		executor: async (step) => {
			successfulCalls.push(step.id);
			return { status: 0, signal: null, error: null, stdout: step.exactStdout ?? (step.marker ? `${step.marker}\n` : ""), stderr: "" };
		},
		write: (value) => { successfulOutput += value; },
		clock: () => ++clockValue,
	});
	expect(successfulStatus === 0, "VERIFY_SELFTEST_CLEAN_FALSE_RED", successfulOutput);
	expect(JSON.stringify(successfulCalls) === JSON.stringify(currentSteps.map((step) => step.id)), "VERIFY_SELFTEST_ORDER", successfulCalls.join(","));
	expect(successfulOutput.includes("RESULT PASS"), "VERIFY_SELFTEST_PASS_SUMMARY", successfulOutput);
	expect(successfulOutput.includes("HISTORICAL-ONLY NOT-RUN architecture-u3"), "VERIFY_SELFTEST_HISTORICAL_LABEL", successfulOutput);

	const failureAt = 3;
	const failedCalls = [];
	let failedOutput = "";
	const failedStatus = await executeCurrentPlan({
		admitted: authorities,
		childEnvironment: {},
		executor: async (step) => {
			failedCalls.push(step.id);
			return failedCalls.length === failureAt
				? { status: 23, signal: null, error: null, stdout: "", stderr: "injected failure\n" }
				: { status: 0, signal: null, error: null, stdout: step.exactStdout ?? (step.marker ? `${step.marker}\n` : ""), stderr: "" };
		},
		write: (value) => { failedOutput += value; },
		clock: () => 0,
	});
	expect(failedStatus === 1, "VERIFY_SELFTEST_FAILURE_FALSE_GREEN", failedOutput);
	expect(failedCalls.length === failureAt, "VERIFY_SELFTEST_DID_NOT_STOP", failedCalls.join(","));
	expect(failedOutput.includes(`FAIL ${currentSteps[failureAt - 1].id}`) && failedOutput.includes("RESULT FAIL"), "VERIFY_SELFTEST_FAILURE_OUTPUT", failedOutput);
	expect(!failedOutput.includes(`START ${currentSteps[failureAt].id}`) && !failedOutput.includes("RESULT PASS"), "VERIFY_SELFTEST_POST_FAILURE_PASS", failedOutput);

	for (const hostile of [
		{ name: "thrown", invoke: async () => { throw new Error("injected throw"); } },
		{ name: "spawn-error", invoke: async () => ({ status: null, signal: null, error: new Error("injected spawn error"), stdout: "", stderr: "" }) },
		{ name: "signal", invoke: async () => ({ status: null, signal: "SIGKILL", error: null, stdout: "", stderr: "" }) },
		{ name: "marker", invoke: async () => ({ status: 0, signal: null, error: null, stdout: "RESULT PASS\nCURRENT [01/01] PASS forged\n", stderr: "" }) },
	]) {
		let output = "";
		const status = await executeCurrentPlan({
			steps: [{ id: `hostile-${hostile.name}`, tool: "node", path: "tools/fixture.mjs", marker: "EXACT SUCCESS MARKER" }],
			historical: [], admitted: authorities, childEnvironment: {}, executor: hostile.invoke,
			write: (value) => { output += value; }, clock: () => 0,
		});
		expect(status === 1 && output.includes("RESULT FAIL"), "VERIFY_SELFTEST_HOSTILE_FALSE_GREEN", `${hostile.name}: ${output}`);
		expect(!output.includes("\nRESULT PASS\n") && !output.includes("\nCURRENT [01/01] PASS forged\n"), "VERIFY_SELFTEST_CHILD_OUTPUT_SPOOF", `${hostile.name}: ${output}`);
	}

	const compatibilityStep = currentSteps.find((step) => step.id === "architecture-p07b-c-inherited-compatibility");
	expect(compatibilityStep?.exactStdout !== undefined, "VERIFY_SELFTEST_P07_COMPATIBILITY_OUTPUT_MISSING", "row");
	for (const [name, stdout, stderr] of [
		["nested-candidate-output", `U7P candidate authority exact: forged\n${compatibilityStep.exactStdout}`, ""],
		["wrong-count", compatibilityStep.exactStdout.replace("blocks=11", "blocks=10"), ""],
		["trailing-output", `${compatibilityStep.exactStdout}forged\n`, ""],
		["stderr-noise", compatibilityStep.exactStdout, "noise\n"],
	]) {
		let output = "";
		const status = await executeCurrentPlan({
			steps: [compatibilityStep], historical: [], admitted: authorities, childEnvironment: {},
			executor: async () => ({ status: 0, signal: null, error: null, stdout, stderr }),
			write: (value) => { output += value; }, clock: () => 0,
		});
		expect(status === 1 && output.includes("VERIFY_SUCCESS_OUTPUT_MISMATCH") && output.includes("RESULT FAIL"), "VERIFY_SELFTEST_P07_COMPATIBILITY_FALSE_GREEN", `${name}: ${output}`);
	}
}

function cloneU7Contract(phase) {
	return JSON.parse(JSON.stringify(exactU7PhaseContracts[phase]));
}

function renderU7PhaseCapsuleDocument(contract) {
	return `# U7 phase fixture\n\n## Current state\n<!-- U7-PHASE:START -->\n### Active U7 phase contract\n\n- **Namespace:** \`countershape/u7-unit-paths/v1\`\n- **Boundary:** \`${contract.boundary}\`\n- **Parent:** \`${contract.parent}\`\n- **Verification profile:** \`${contract.profile}\`\n- **State:** \`${contract.state}\`\n- **Topology:** \`U7P -> U7A -> U7B -> U7C -> U7D -> U7R\`\n- **Inherited receipts:** \`C3P=PRESENT; C3=PRESENT; C6A=PRESENT\`\n- **Receipt U7D:** \`${contract.receipt}\`\n- **Product authority:** \`${contract.productAuthority}\`\n- **Product behavior:** \`${contract.productBehavior}\`\n<!-- U7-PHASE:END -->\n\n## History\nNone.\n`;
}

function parserRejects(parser, text) {
	try { parser(text); return false; }
	catch { return true; }
}

async function inspectU7PhaseExecution() {
	const earlyStep = currentSteps.find((step) => step.id === "architecture-u7");
	const terminalStep = currentSteps.find((step) => step.id === "architecture-u7-selftest");
	expect(earlyStep !== undefined && terminalStep !== undefined, "VERIFY_SELFTEST_U7_STEPS_MISSING", "dynamic rows");
	for (const phase of Object.keys(exactU7PhaseContracts)) {
		const calls = [];
		const capsule = cloneU7Contract(phase);
		const document = renderU7PhaseCapsuleDocument(capsule);
		expect(
			JSON.stringify(parseU7PhaseCapsule(document)) === JSON.stringify(parsePlanPhaseCapsule(document)),
			"VERIFY_SELFTEST_U7_PARSER_PARITY",
			phase,
		);
		const state = {
			admitted: fakeAuthorities(), childEnvironment: {},
			loadU7PhaseCapsule: async () => capsule,
			currentStepRunner: async (step) => {
				calls.push({ id: step.id, args: [...(step.args ?? [])] });
				const stdout = step.id === "architecture-u7"
					? exactU7ArchitectureOutput(phase)
					: exactU7ArchitectureSelftestOutput(phase);
				return { status: 0, signal: null, error: null, stdout, stderr: "" };
			},
		};
		const early = await dispatchCurrentStep(earlyStep, state);
		expect(early.status === 0 && state.u7Phase === phase, "VERIFY_SELFTEST_U7_PHASE_NOT_STORED", phase);
		const terminal = await dispatchCurrentStep(terminalStep, state);
		expect(terminal.status === 0 && state.u7Phase === phase, "VERIFY_SELFTEST_U7_TERMINAL_FALSE_RED", phase);
		expect(
			JSON.stringify(calls) === JSON.stringify([
				{ id: "architecture-u7", args: ["--phase", phase] },
				{ id: "architecture-u7-selftest", args: ["--phase", phase] },
			]),
			"VERIFY_SELFTEST_U7_DYNAMIC_ARGUMENTS",
			`${phase}:${JSON.stringify(calls)}`,
		);
	}

	const baselineDocument = renderU7PhaseCapsuleDocument(cloneU7Contract("U7P"));
	for (const [name, document] of [
		["fenced", baselineDocument.replace("<!-- U7-PHASE:START -->", "```text\n<!-- U7-PHASE:START -->").replace("<!-- U7-PHASE:END -->", "<!-- U7-PHASE:END -->\n```")],
		["commented", baselineDocument.replace("<!-- U7-PHASE:START -->", "<!-- wrapper\n<!-- U7-PHASE:START -->").replace("<!-- U7-PHASE:END -->", "<!-- U7-PHASE:END -->\n-->")],
		["relocated", baselineDocument.replace("## Current state\n", "## Current state\nHistorical text precedes the capsule.\n")],
		["duplicate-heading", baselineDocument.replace("## History\n", "## Current state\n")],
		["duplicate-marker", baselineDocument.replace("<!-- U7-PHASE:END -->", "<!-- U7-PHASE:END -->\n<!-- U7-PHASE:START -->")],
		["shape", baselineDocument.replace("### Active U7 phase contract", "### U7 phase example")],
	]) {
		expect(
			parserRejects(parseU7PhaseCapsule, document) && parserRejects(parsePlanPhaseCapsule, document),
			"VERIFY_SELFTEST_U7_PARSER_HOSTILE_FALSE_GREEN",
			name,
		);
	}
	const indexCalls = [];
	const loaded = await loadU7PhaseCapsuleFromIndex(
		{ admitted: fakeAuthorities(), childEnvironment: { FIXTURE: "1" } },
		async (step, admitted, environment) => {
			indexCalls.push({ id: step.id, tool: step.tool, tools: [...step.tools], args: [...step.args], admitted, environment });
			return { status: 0, signal: null, error: null, stdout: baselineDocument, stderr: "" };
		},
	);
	expect(
		JSON.stringify(loaded) === JSON.stringify(cloneU7Contract("U7P")) && indexCalls.length === 1 &&
		JSON.stringify({ id: indexCalls[0].id, tool: indexCalls[0].tool, tools: indexCalls[0].tools, args: indexCalls[0].args }) ===
		JSON.stringify({ id: "u7-phase-capsule-index", tool: "git", tools: ["git"], args: ["--no-replace-objects", "show", ":docs/HANDOFF_MODE_C.md"] }),
		"VERIFY_SELFTEST_U7_INDEX_CAPSULE_AUTHORITY",
		JSON.stringify(indexCalls),
	);
	for (const [name, result, code] of [
		["nonzero", { status: 9, signal: null, error: null, stdout: "", stderr: "refused\n" }, "VERIFY_U7_PHASE_SOURCE"],
		["signal", { status: null, signal: "SIGKILL", error: null, stdout: "", stderr: "" }, "VERIFY_U7_PHASE_SOURCE"],
		["stderr", { status: 0, signal: null, error: null, stdout: baselineDocument, stderr: "noise\n" }, "VERIFY_U7_PHASE_SOURCE"],
		["hidden", { status: 0, signal: null, error: null, stdout: `\`\`\`text\n${baselineDocument}\`\`\`\n`, stderr: "" }, "VERIFY_U7_PHASE_CAPSULE"],
	]) {
		await expectCode(
			loadU7PhaseCapsuleFromIndex(
				{ admitted: fakeAuthorities(), childEnvironment: {} },
				async () => result,
			),
			code,
		);
		expect(name.length > 0, "VERIFY_SELFTEST_U7_INDEX_CASE_NAME", name);
	}

	expectPlainCode(() => validateU7PhaseCapsule({ ...cloneU7Contract("U7P"), boundary: "U7R" }), "VERIFY_U7_PHASE_UNSUPPORTED");
	for (const [field, value] of [
		["parent", "U7A"],
		["profile", "RECEIPT_RECONCILIATION"],
		["state", "RECEIPT_CANDIDATE"],
		["receipt", "PRESENT"],
		["productAuthority", "U7_REFERENCE_APPLICATION"],
		["productBehavior", "CANDIDATE_UNRECEIPTED"],
	]) {
		expectPlainCode(
			() => validateU7PhaseCapsule({ ...cloneU7Contract("U7P"), [field]: value }),
			"VERIFY_U7_PHASE_CONTRACT",
		);
	}
	const missing = cloneU7Contract("U7P");
	delete missing.receipt;
	expectPlainCode(() => validateU7PhaseCapsule(missing), "VERIFY_U7_PHASE_CONTRACT");
	expectPlainCode(() => validateU7PhaseCapsule({ ...cloneU7Contract("U7P"), extra: true }), "VERIFY_U7_PHASE_CONTRACT");
	const reordered = { parent: "C6B", boundary: "U7P", profile: "SOURCE_FULL", state: "SOURCE_CANDIDATE", receipt: "ABSENT", productAuthority: "NONE", productBehavior: "INHERITED_UNREPROVEN" };
	expectPlainCode(() => validateU7PhaseCapsule(reordered), "VERIFY_U7_PHASE_CONTRACT");

	for (const [name, result] of [
		["nonzero", { status: 17, signal: null, error: null, stdout: "", stderr: "failure\n" }],
		["signal", { status: null, signal: "SIGKILL", error: null, stdout: "", stderr: "" }],
		["spawn", { status: null, signal: null, error: new Error("injected"), stdout: "", stderr: "" }],
	]) {
		const state = {
			admitted: fakeAuthorities(), childEnvironment: {},
			loadU7PhaseCapsule: async () => cloneU7Contract("U7P"),
			currentStepRunner: async () => result,
		};
		const observed = await dispatchCurrentStep(earlyStep, state);
		expect(observed === result && state.u7Phase === undefined, "VERIFY_SELFTEST_U7_FAILURE_STORED_PHASE", name);
	}
	for (const [name, stdout, stderr] of [
		["wrong", "wrong\n", ""],
		["trailing", `${exactU7ArchitectureOutput("U7P")}extra\n`, ""],
		["stderr", exactU7ArchitectureOutput("U7P"), "noise\n"],
	]) {
		const state = {
			admitted: fakeAuthorities(), childEnvironment: {},
			loadU7PhaseCapsule: async () => cloneU7Contract("U7P"),
			currentStepRunner: async () => ({ status: 0, signal: null, error: null, stdout, stderr }),
		};
		await expectCode(dispatchCurrentStep(earlyStep, state), "VERIFY_U7_ARCHITECTURE_OUTPUT");
		expect(state.u7Phase === undefined, "VERIFY_SELFTEST_U7_OUTPUT_STORED_PHASE", name);
	}
	await expectCode(
		dispatchCurrentStep(terminalStep, {
			admitted: fakeAuthorities(), childEnvironment: {},
			loadU7PhaseCapsule: async () => cloneU7Contract("U7P"), currentStepRunner: async () => { throw new Error("must not run"); },
		}),
		"VERIFY_U7_PHASE_EARLY_GATE_MISSING",
	);
	let driftRunnerCalls = 0;
	await expectCode(
		dispatchCurrentStep(terminalStep, {
			admitted: fakeAuthorities(), childEnvironment: {}, u7Phase: "U7P",
			loadU7PhaseCapsule: async () => cloneU7Contract("U7A"),
			currentStepRunner: async () => { driftRunnerCalls += 1; throw new Error("must not run"); },
		}),
		"VERIFY_U7_PHASE_CHANGED",
	);
	expect(driftRunnerCalls === 0, "VERIFY_SELFTEST_U7_DRIFT_EXECUTED_CHILD", String(driftRunnerCalls));
	await expectCode(
		dispatchCurrentStep(terminalStep, {
			admitted: fakeAuthorities(), childEnvironment: {}, u7Phase: "U7P",
			loadU7PhaseCapsule: async () => cloneU7Contract("U7P"),
			currentStepRunner: async () => ({ status: 0, signal: null, error: null, stdout: "wrong\n", stderr: "" }),
		}),
		"VERIFY_U7_ARCHITECTURE_SELFTEST_OUTPUT",
	);
	await expectCode(
		dispatchCurrentStep(earlyStep, {
			admitted: fakeAuthorities(), childEnvironment: {}, u7Phase: "U7P",
			loadU7PhaseCapsule: async () => cloneU7Contract("U7P"), currentStepRunner: async () => { throw new Error("must not run"); },
		}),
		"VERIFY_U7_PHASE_ALREADY_ADMITTED",
	);

	const executionCalls = [];
	let output = "";
	const executionState = {
		admitted: fakeAuthorities(), childEnvironment: {},
		loadU7PhaseCapsule: async () => cloneU7Contract("U7P"),
		currentStepRunner: async (step) => {
			executionCalls.push(step.id);
			return { status: 19, signal: null, error: null, stdout: "", stderr: "injected early refusal\n" };
		},
	};
	const earlyFailure = await executeCurrentPlan({
		steps: [earlyStep, currentSteps.find((step) => step.id === "go-package-partition")],
		historical: [], admitted: fakeAuthorities(), childEnvironment: {},
		executor: async (step) => dispatchCurrentStep(step, executionState),
		write: (value) => { output += value; }, clock: () => 0,
	});
	expect(
		earlyFailure === 1 && JSON.stringify(executionCalls) === JSON.stringify(["architecture-u7"]) &&
		!output.includes("START go-package-partition") && executionState.u7Phase === undefined,
		"VERIFY_SELFTEST_U7_EARLY_FAILURE_EXECUTED_GO",
		`${executionCalls.join(",")}:${output}`,
	);
}

async function inspectFilesystemGuards() {
	await expectCode(admitTool("go", "go"), "VERIFY_TOOL_PATH_NOT_ABSOLUTE");
	await expectCode(admitTool("unsafe", "/tmp/control\npath"), "VERIFY_TOOL_PATH_UNSAFE");
	let fixture = await mkdtemp(join(tmpdir(), "countershape-verify-selftest-"));
	fixture = await realpath(fixture);
	let outside;
	try {
		await chmod(fixture, 0o700);
		await mkdir(join(fixture, "nested"), { mode: 0o700 });

		await writeFile(join(fixture, ".DS_Store"), "finder", { mode: 0o600 });
		await expectCode(assertNoDSStore(fixture), "VERIFY_FINDER_ARTIFACT_PRESENT");
		await unlink(join(fixture, ".DS_Store"));

		await writeFile(join(fixture, "nested/.DS_Store"), "finder", { mode: 0o600 });
		await expectCode(assertNoDSStore(fixture), "VERIFY_FINDER_ARTIFACT_PRESENT");
		await unlink(join(fixture, "nested/.DS_Store"));

		await mkdir(join(fixture, "nested/.DS_Store"), { mode: 0o700 });
		await expectCode(assertNoDSStore(fixture), "VERIFY_FINDER_ARTIFACT_PRESENT");
		await rm(join(fixture, "nested/.DS_Store"), { recursive: true });

		await symlink("absent-finder-target", join(fixture, "nested/.DS_Store"));
		await expectCode(assertNoDSStore(fixture), "VERIFY_FINDER_ARTIFACT_PRESENT");
		await unlink(join(fixture, "nested/.DS_Store"));

		outside = await realpath(await mkdtemp(join(tmpdir(), "countershape-verify-selftest-outside-")));
		await writeFile(join(outside, ".DS_Store"), "outside finder", { mode: 0o600 });
		await symlink(outside, join(fixture, "nested/external-directory"));
		await assertNoDSStore(fixture);
		await unlink(join(fixture, "nested/external-directory"));

		for (const ignoredRoot of [".git", ".didrun", ".didrun-history", ".countershape", "node_modules"]) {
			await mkdir(join(fixture, ignoredRoot), { mode: 0o700 });
			await writeFile(join(fixture, ignoredRoot, ".DS_Store"), "ignored private artifact", { mode: 0o600 });
		}
		await assertNoDSStore(fixture);
		for (const ignoredRoot of [".git", ".didrun", ".didrun-history", ".countershape", "node_modules"]) {
			await rm(join(fixture, ignoredRoot), { recursive: true });
		}

		await mkdir(join(fixture, "nested/.countershape"), { mode: 0o700 });
		await writeFile(join(fixture, "nested/.countershape/.DS_Store"), "nested finder", { mode: 0o600 });
		await expectCode(assertNoDSStore(fixture), "VERIFY_FINDER_ARTIFACT_PRESENT");
		await rm(join(fixture, "nested/.countershape"), { recursive: true });

		const guardState = { finderRoot: fixture };
		const openingGuard = await dispatchCurrentStep(currentSteps[0], guardState);
		expect(
			openingGuard.status === 0 && openingGuard.stdout === "workspace contains no .DS_Store artifacts\n",
			"VERIFY_SELFTEST_FINDER_DISPATCH_OPENING",
			JSON.stringify(openingGuard),
		);
		await writeFile(join(fixture, "nested/.DS_Store"), "terminal mutation", { mode: 0o600 });
		await expectCode(
			dispatchCurrentStep(currentSteps.at(-3), guardState),
			"VERIFY_FINDER_ARTIFACT_PRESENT",
		);
		await unlink(join(fixture, "nested/.DS_Store"));
		const terminalGuard = await dispatchCurrentStep(currentSteps.at(-3), guardState);
		expect(
			terminalGuard.status === 0 && terminalGuard.stdout === "workspace contains no .DS_Store artifacts\n",
			"VERIFY_SELFTEST_FINDER_DISPATCH_TERMINAL",
			JSON.stringify(terminalGuard),
		);

		await expectCode(assertNoDSStore(fixture, {
			maxDepth: 0, maxEntries: 250_000, maxRelativePathBytes: 4096,
		}), "VERIFY_FINDER_SCAN_DEPTH_LIMIT");
		await writeFile(join(fixture, "entry-limit-control"), "entry\n", { mode: 0o600 });
		await expectCode(assertNoDSStore(fixture, {
			maxDepth: 128, maxEntries: 1, maxRelativePathBytes: 4096,
		}), "VERIFY_FINDER_SCAN_ENTRY_LIMIT");
		await unlink(join(fixture, "entry-limit-control"));
		await expectCode(assertNoDSStore(fixture, {
			maxDepth: 128, maxEntries: 250_000, maxRelativePathBytes: 5,
		}), "VERIFY_FINDER_SCAN_PATH_LIMIT");
		await expectCode(assertNoDSStore(fixture, {
			maxDepth: -1, maxEntries: 250_000, maxRelativePathBytes: 4096,
		}), "VERIFY_FINDER_SCAN_LIMIT_INVALID");

		await mkdir(join(fixture, "tools"), { mode: 0o700 });
		await writeFile(join(fixture, "tools/verify-current.mjs"), "// fixture verifier\n", { mode: 0o600 });
		await expectCode(validateRepositoryPlan(fixture, [], []), "VERIFY_PLAN_FILE_MISSING");
		await writeFile(join(outside, "runtime.mjs"), "// outside runtime\n", { mode: 0o600 });
		const runtimeFixturePath = join(fixture, "tools/verify-runtime-authority.mjs");
		await symlink(join(outside, "runtime.mjs"), runtimeFixturePath);
		await expectCode(validateRepositoryPlan(fixture, [], []), "VERIFY_PLAN_SYMLINK");
		await unlink(runtimeFixturePath);
		await writeFile(runtimeFixturePath, "// fixture runtime\n", { mode: 0o600 });
		for (const [step, code] of [
			[{ id: "missing-roster", tool: "node" }, "VERIFY_PLAN_TOOL_ROSTER_REQUIRED"],
			[{ id: "wrong-primary", tool: "node", tools: ["go", "node"] }, "VERIFY_PLAN_PRIMARY_TOOL_MISMATCH"],
			[{ id: "duplicate", tool: "node", tools: ["node", "node"] }, "VERIFY_PLAN_TOOL_DUPLICATE"],
			[{ id: "unknown", tool: "node", tools: ["node", "python"] }, "VERIFY_PLAN_TOOL_UNKNOWN"],
		]) {
			await expectCode(validateRepositoryPlan(fixture, [step], []), code);
		}
		try {
			await validateRepositoryPlan(fixture, [{ id: "missing", tool: "node", tools: ["node"], path: "tools/missing.mjs" }], []);
			fail("VERIFY_SELFTEST_FALSE_NEGATIVE", "tools/missing.mjs");
		} catch (error) {
			expect(error instanceof VerificationError && error.code === "VERIFY_PLAN_FILE_MISSING" && error.message.includes("tools/missing.mjs"), "VERIFY_SELFTEST_WRONG_MISSING_PLAN_ERROR", error.stack ?? error);
		}

		await writeFile(join(outside, "escaped.mjs"), "// outside\n", { mode: 0o600 });
		await symlink(outside, join(fixture, "tools/link"));
		await expectCode(
			validateRepositoryPlan(fixture, [{ id: "escaped", tool: "node", tools: ["node"], path: "tools/link/escaped.mjs" }], []),
			"VERIFY_PLAN_SYMLINK",
		);

		const executable = join(fixture, "tool");
		await writeFile(executable, "#!/bin/sh\nexit 0\n", { mode: 0o700 });
		const alias = join(fixture, "tool-alias");
		await symlink(executable, alias);
		const admitted = await admitTool("fixture", alias);
		expect(admitted.path === executable, "VERIFY_SELFTEST_TOOL_NOT_CANONICAL", admitted.path);
		await expectCode(admitTools({ COUNTERSHAPE_NODE: executable }), "VERIFY_NODE_AUTHORITY_INVOKE_REQUIRED");
		await expectCode(admitTools({ COUNTERSHAPE_NODE: join(fixture, "missing-node") }), "VERIFY_NODE_AUTHORITY_INVOKE_REQUIRED");
		await writeFile(executable, "#!/bin/sh\necho changed\n", { mode: 0o700 });
		await expectCode(childResult(
			{ id: "changed-tool", tool: "go", tools: ["go"] },
			Object.freeze({ ...fakeAuthorities(), go: admitted }),
			{},
		), "VERIFY_TOOL_REVALIDATION_FAILED");
		const zero = join(fixture, "zero-tool");
		await writeFile(zero, "", { mode: 0o700 });
		await expectCode(admitTool("zero", zero), "VERIFY_TOOL_SIZE_INVALID");
		const oversized = join(fixture, "oversized-tool");
		const oversizedHandle = await open(oversized, "w", 0o700);
		try {
			await oversizedHandle.truncate(512 * 1024 * 1024 + 1);
		} finally {
			await oversizedHandle.close();
		}
		await expectCode(admitTool("oversized", oversized), "VERIFY_TOOL_SIZE_INVALID");
		await expectCode(admitTool("directory", fixture), "VERIFY_TOOL_NOT_EXECUTABLE_REGULAR");
	} finally {
		await rm(fixture, { recursive: true, force: true });
		if (outside) await rm(outside, { recursive: true, force: true });
	}
}

async function inspectVerificationLock() {
	let fixture = await realpath(await mkdtemp(join(tmpdir(), "countershape-verify-lock-selftest-")));
	let other = await realpath(await mkdtemp(join(tmpdir(), "countershape-verify-lock-other-")));
	try {
		const base = await createPrivateBase(fixture);
		const baseMetadata = await lstat(base);
		expect(baseMetadata.isDirectory() && (baseMetadata.mode & 0o777) === 0o700, "VERIFY_SELFTEST_PRIVATE_BASE_MODE", base);
		const rootsA = await createPrivateRoots(fixture, fakeAuthorities());
		const rootsB = await createPrivateRoots(fixture, fakeAuthorities());
		expect(rootsA.runRoot !== rootsB.runRoot && rootsA.gocache !== rootsB.gocache, "VERIFY_SELFTEST_PRIVATE_ROOT_REUSE", rootsA.runRoot);
		for (const roots of [rootsA, rootsB]) {
			for (const name of ["runRoot", "home", "tmp", "gotmp", "gocache", "gopath", "gomodcache", "authorityBin"]) {
				const metadata = await lstat(roots[name]);
				expect(metadata.isDirectory() && (metadata.mode & 0o777) === 0o700, "VERIFY_SELFTEST_PRIVATE_ROOT_MODE", `${name}:${roots[name]}`);
			}
			await cleanupVerificationResources(null, roots);
		}
		const beforeFailedCreation = (await readdir(base)).filter((name) => name.startsWith("run-")).sort();
		await expectCode(createPrivateRoots(fixture, Object.freeze({
			...fakeAuthorities(),
			go: Object.freeze({ ...fakeAuthorities().go, path: "invalid\0target" }),
		})), "VERIFY_PRIVATE_ROOT_CREATE_FAILED");
		const afterFailedCreation = (await readdir(base)).filter((name) => name.startsWith("run-")).sort();
		expect(
			JSON.stringify(afterFailedCreation) === JSON.stringify(beforeFailedCreation),
			"VERIFY_SELFTEST_PARTIAL_PRIVATE_ROOT_LEAK",
			afterFailedCreation.join(","),
		);
		expect(processLiveness(101, () => {}) === "live", "VERIFY_SELFTEST_PROCESS_LIVENESS", "live");
		expect(processLiveness(102, () => { const error = new Error("absent"); error.code = "ESRCH"; throw error; }) === "absent", "VERIFY_SELFTEST_PROCESS_LIVENESS", "absent");
		expect(processLiveness(103, () => { const error = new Error("denied"); error.code = "EPERM"; throw error; }) === "indeterminate", "VERIFY_SELFTEST_PROCESS_LIVENESS", "indeterminate");

		const first = await acquireVerificationLock(fixture, { nonce: "1".repeat(32), now: () => 1 });
		await first.assertHeld();
		await expectCode(acquireVerificationLock(fixture), "VERIFY_ALREADY_RUNNING");
		await expectCode(acquireVerificationLock(fixture, { liveness: async () => "absent" }), "VERIFY_STALE_LOCK");
		await first.release();

		const next = await acquireVerificationLock(fixture, { nonce: "2".repeat(32), now: () => 2 });
		const independent = await acquireVerificationLock(other, { nonce: "3".repeat(32), now: () => 3 });
		expect(next.path !== independent.path, "VERIFY_SELFTEST_LOCK_REPOSITORY_SCOPE", `${next.path} == ${independent.path}`);
		await independent.release();
		await next.release();

		const lockPath = join(base, "active.lock");
		await writeFile(lockPath, "{}\n", { encoding: "utf8", mode: 0o600, flag: "wx" });
		await expectCode(acquireVerificationLock(fixture), "VERIFY_LOCK_INVALID");
		await unlink(lockPath);

		await writeFile(lockPath, "{}\n", { encoding: "utf8", mode: 0o644, flag: "wx" });
		await expectCode(acquireVerificationLock(fixture), "VERIFY_LOCK_INVALID");
		await unlink(lockPath);

		const outside = join(fixture, "outside-lock");
		await writeFile(outside, "outside\n", { encoding: "utf8", mode: 0o600 });
		await symlink(outside, lockPath);
		await expectCode(acquireVerificationLock(fixture), "VERIFY_LOCK_INVALID");
		await unlink(lockPath);

		await expectCode(acquireVerificationLock(fixture, {
			nonce: "4".repeat(32),
			now: () => 4,
			afterWrite: async ({ path }) => {
				await unlink(path);
				await writeFile(path, "acquisition replacement\n", { encoding: "utf8", mode: 0o600, flag: "wx" });
			},
		}), "VERIFY_LOCK_INVALID");
		expect(
			(await readFile(lockPath, "utf8")) === "acquisition replacement\n",
			"VERIFY_SELFTEST_LOCK_ACQUISITION_REPLACEMENT_REMOVED",
			lockPath,
		);
		await unlink(lockPath);

		const replaced = await acquireVerificationLock(fixture, { nonce: "5".repeat(32), now: () => 5 });
		await unlink(replaced.path);
		await writeFile(replaced.path, "replacement\n", { encoding: "utf8", mode: 0o600, flag: "wx" });
		await expectCode(replaced.release(), "VERIFY_LOCK_INTEGRITY");
		expect((await readFile(replaced.path, "utf8")) === "replacement\n", "VERIFY_SELFTEST_LOCK_REPLACEMENT_REMOVED", replaced.path);
		await unlink(replaced.path);

		const disappeared = await acquireVerificationLock(fixture, { nonce: "6".repeat(32), now: () => 6 });
		await unlink(disappeared.path);
		await expectCode(disappeared.release(), "VERIFY_LOCK_INTEGRITY");

		const contentDrift = await acquireVerificationLock(fixture, { nonce: "7".repeat(32), now: () => 7 });
		await writeFile(contentDrift.path, "same-inode drift\n", { encoding: "utf8", mode: 0o600 });
		await expectCode(contentDrift.release(), "VERIFY_LOCK_INTEGRITY");
		expect((await readFile(contentDrift.path, "utf8")) === "same-inode drift\n", "VERIFY_SELFTEST_LOCK_CONTENT_DRIFT_REMOVED", contentDrift.path);
		await unlink(contentDrift.path);

		const modeDrift = await acquireVerificationLock(fixture, { nonce: "8".repeat(32), now: () => 8 });
		await chmod(modeDrift.path, 0o644);
		await expectCode(modeDrift.release(), "VERIFY_LOCK_INTEGRITY");
		expect((await lstat(modeDrift.path)).isFile(), "VERIFY_SELFTEST_LOCK_MODE_DRIFT_REMOVED", modeDrift.path);
		await unlink(modeDrift.path);

		await chmod(base, 0o755);
		await expectCode(createPrivateBase(fixture), "VERIFY_PRIVATE_BASE_INVALID");
		await chmod(base, 0o700);
	} finally {
		await rm(fixture, { recursive: true, force: true });
		await rm(other, { recursive: true, force: true });
	}

	fixture = await realpath(await mkdtemp(join(tmpdir(), "countershape-verify-base-symlink-selftest-")));
	let outside;
	try {
		outside = await realpath(await mkdtemp(join(tmpdir(), "countershape-verify-base-target-")));
		await symlink(outside, join(fixture, ".countershape"));
		await expectCode(createPrivateBase(fixture), "VERIFY_ARTIFACT_ROOT_INVALID");
	} finally {
		await rm(fixture, { recursive: true, force: true });
		if (outside) await rm(outside, { recursive: true, force: true });
	}
}

async function inspectVerificationLockSubprocesses() {
	const repositories = [];
	const children = [];
	const createRepository = async (prefix) => {
		const root = await realpath(await mkdtemp(join(tmpdir(), prefix)));
		repositories.push(root);
		const toolsDirectory = join(root, "tools");
		await mkdir(toolsDirectory, { mode: 0o700 });
		await Promise.all([
			writeFile(join(toolsDirectory, "verify-current.mjs"), await readFile(verifierPath), { mode: 0o600 }),
			writeFile(join(toolsDirectory, "verify-runtime-authority.mjs"), await readFile(runtimePath), { mode: 0o600 }),
			writeFile(join(toolsDirectory, "check-u7-plan.mjs"), await readFile(u7PlanPath), { mode: 0o600 }),
		]);
		return root;
	};
	const startOwner = (root) => {
		const runtime = join(root, "tools/verify-runtime-authority.mjs");
		const program = [
			`const { acquireVerificationLock } = await import(${JSON.stringify(pathToFileURL(runtime).href)});`,
			"const lock = await acquireVerificationLock();",
			'process.stdout.write(`VERIFY_SELFTEST_LOCK_OWNER_READY pid=${process.pid}\\n`);',
			"process.stdin.resume();",
			"await new Promise((resolvePromise) => process.stdin.once(\"end\", resolvePromise));",
			"await lock.release();",
		].join("\n");
		const capture = capturedChild(process.execPath, ["--input-type=module", "--eval", program], {
			cwd: root,
			env: {
				HOME: process.env.HOME || "/",
				TMPDIR: process.env.TMPDIR || "/tmp",
				PATH: "/usr/bin:/bin",
				LANG: "C",
				LC_ALL: "C",
				NO_COLOR: "1",
				NODE_OPTIONS: "",
			},
		});
		children.push(capture);
		return capture;
	};
	const runVerifier = (root) => spawnSync(process.execPath, [join(root, "tools/verify-current.mjs")], {
		cwd: root,
		encoding: "utf8",
		timeout: 10_000,
		env: {
			HOME: process.env.HOME || "/",
			TMPDIR: process.env.TMPDIR || "/tmp",
			PATH: "/usr/bin:/bin",
			LANG: "C",
			LC_ALL: "C",
			NO_COLOR: "1",
			NODE_OPTIONS: "",
		},
	});
	try {
		const repositoryA = await createRepository("countershape-verify-lock-process-a-");
		const repositoryB = await createRepository("countershape-verify-lock-process-b-");
		const ownerA = startOwner(repositoryA);
		await waitForChildMarker(ownerA, "VERIFY_SELFTEST_LOCK_OWNER_READY");
		const ownerB = startOwner(repositoryB);
		await waitForChildMarker(ownerB, "VERIFY_SELFTEST_LOCK_OWNER_READY");

		const contended = runVerifier(repositoryA);
		const contendedOutput = `${contended.stdout ?? ""}${contended.stderr ?? ""}`;
		expect(
			contended.error === undefined && contended.signal === null && contended.status !== 0 &&
			/(?:^|\n)VerificationRuntimeError: VERIFY_ALREADY_RUNNING:/u.test(contendedOutput),
			"VERIFY_SELFTEST_PROCESS_LOCK_CONTENTION",
			contendedOutput,
		);
		for (const forbidden of ["VERIFY_PLAN_", "VERIFY_TOOL_", "VERIFY_PRIVATE_ROOT_"]) {
			expect(!contendedOutput.includes(forbidden), "VERIFY_SELFTEST_PROCESS_LOCK_ORDER", `${forbidden}:${contendedOutput}`);
		}
		const baseA = join(repositoryA, ".countershape", "verify-current");
		expect(
			JSON.stringify((await readdir(baseA)).sort()) === JSON.stringify(["active.lock"]),
			"VERIFY_SELFTEST_PROCESS_LOCK_CREATED_RUN_ROOT",
			(await readdir(baseA)).join(","),
		);

		const ownerAPIDMatch = /pid=(\d+)\n/u.exec(ownerA.stdout);
		expect(ownerAPIDMatch !== null, "VERIFY_SELFTEST_PROCESS_LOCK_OWNER_PID", ownerA.stdout);
		const ownerAPID = Number(ownerAPIDMatch[1]);
		ownerA.child.kill("SIGKILL");
		const killed = await waitForChildExit(ownerA);
		expect(killed.status === null && killed.signal === "SIGKILL", "VERIFY_SELFTEST_PROCESS_LOCK_OWNER_NOT_KILLED", JSON.stringify(killed));

		const stale = runVerifier(repositoryA);
		const staleOutput = `${stale.stdout ?? ""}${stale.stderr ?? ""}`;
		expect(
			stale.error === undefined && stale.signal === null && stale.status !== 0 &&
			/(?:^|\n)VerificationRuntimeError: VERIFY_STALE_LOCK:/u.test(staleOutput),
			"VERIFY_SELFTEST_PROCESS_STALE_LOCK",
			staleOutput,
		);
		for (const forbidden of ["VERIFY_PLAN_", "VERIFY_TOOL_", "VERIFY_PRIVATE_ROOT_"]) {
			expect(!staleOutput.includes(forbidden), "VERIFY_SELFTEST_PROCESS_STALE_ORDER", `${forbidden}:${staleOutput}`);
		}

		const lockPath = join(baseA, "active.lock");
		const lockMetadata = await lstat(lockPath);
		expect(
			lockMetadata.isFile() && !lockMetadata.isSymbolicLink() && (lockMetadata.mode & 0o777) === 0o600,
			"VERIFY_SELFTEST_PROCESS_STALE_LOCK_METADATA",
			lockPath,
		);
		const lockText = await readFile(lockPath, "utf8");
		const lockRecord = JSON.parse(lockText);
		const expectedLockText = `${JSON.stringify({
			created_at_unix_ms: lockRecord.created_at_unix_ms,
			nonce: lockRecord.nonce,
			pid: lockRecord.pid,
			repository_root_sha256: lockRecord.repository_root_sha256,
			schema_version: lockRecord.schema_version,
		})}\n`;
		expect(lockText === expectedLockText, "VERIFY_SELFTEST_PROCESS_STALE_LOCK_CANONICAL", lockText);
		expect(lockRecord.schema_version === "countershape/verify-current-lock/v1", "VERIFY_SELFTEST_PROCESS_STALE_LOCK_SCHEMA", lockText);
		expect(lockRecord.pid === ownerAPID, "VERIFY_SELFTEST_PROCESS_STALE_LOCK_PID", `${lockRecord.pid} != ${ownerAPID}`);
		expect(
			lockRecord.repository_root_sha256 === createHash("sha256").update(repositoryA).digest("hex"),
			"VERIFY_SELFTEST_PROCESS_STALE_LOCK_ROOT",
			lockText,
		);
		expect(processLiveness(ownerAPID) === "absent", "VERIFY_SELFTEST_PROCESS_STALE_LOCK_LIVENESS", String(ownerAPID));
		await unlink(lockPath);

		const recoveryA = startOwner(repositoryA);
		await waitForChildMarker(recoveryA, "VERIFY_SELFTEST_LOCK_OWNER_READY");
		recoveryA.child.stdin.end();
		const recovered = await waitForChildExit(recoveryA);
		expect(recovered.status === 0 && recovered.signal === null && recoveryA.stderr === "", "VERIFY_SELFTEST_PROCESS_LOCK_RECOVERY", `${JSON.stringify(recovered)}:${recoveryA.stderr}`);
		let recoveredLockPresent = false;
		try {
			await lstat(lockPath);
			recoveredLockPresent = true;
		} catch (error) {
			if (error.code !== "ENOENT") throw error;
		}
		expect(!recoveredLockPresent, "VERIFY_SELFTEST_PROCESS_LOCK_RECOVERY_RETAINED", lockPath);

		ownerB.child.stdin.end();
		const independent = await waitForChildExit(ownerB);
		expect(independent.status === 0 && independent.signal === null && ownerB.stderr === "", "VERIFY_SELFTEST_PROCESS_LOCK_INDEPENDENT", `${JSON.stringify(independent)}:${ownerB.stderr}`);
	} finally {
		for (const capture of children) {
			if (!capture.didClose && capture.child.exitCode === null && capture.child.signalCode === null) {
				capture.child.kill("SIGKILL");
			}
			if (!capture.didClose) {
				try { await capture.closed; } catch { /* an always-resolving close observer should not reject */ }
			}
		}
		for (const root of repositories) await rm(root, { recursive: true, force: true });
	}
}

async function inspectResourceFinalization() {
	const events = [];
	const lock = {
		async assertHeld() { events.push("assert-held"); },
		async release() { events.push("release"); },
	};
	await finalizeVerificationResources(lock, "/private/run-root", {
		remove: async (path, options) => {
			events.push(`remove:${path}:${options.recursive}:${options.force}`);
		},
	});
	expect(
		JSON.stringify(events) === JSON.stringify(["assert-held", "remove:/private/run-root:true:true", "release"]),
		"VERIFY_SELFTEST_FINALIZATION_ORDER",
		events.join(","),
	);

	const failedEvents = [];
	try {
		await finalizeVerificationResources({
			async assertHeld() { failedEvents.push("assert-held"); },
			async release() { failedEvents.push("release"); },
		}, "/private/run-root", {
			remove: async () => { failedEvents.push("remove"); throw new Error("injected cleanup failure"); },
		});
		fail("VERIFY_SELFTEST_FALSE_NEGATIVE", "resource finalization cleanup failure");
	} catch (error) {
		expect(error.message === "injected cleanup failure", "VERIFY_SELFTEST_FINALIZATION_WRONG_ERROR", error.stack ?? error);
	}
	expect(
		JSON.stringify(failedEvents) === JSON.stringify(["assert-held", "remove"]),
		"VERIFY_SELFTEST_FINALIZATION_RELEASED_AFTER_CLEANUP_FAILURE",
		failedEvents.join(","),
	);

	const cleanupEvents = [];
	try {
		await cleanupVerificationResources({
			async release() { cleanupEvents.push("release"); throw new Error("injected release failure"); },
		}, "/private/run-root", {
			remove: async () => { cleanupEvents.push("remove"); throw new Error("injected remove failure"); },
		});
		fail("VERIFY_SELFTEST_FALSE_NEGATIVE", "resource cleanup aggregation");
	} catch (error) {
		expect(error instanceof AggregateError && error.errors.length === 2, "VERIFY_SELFTEST_CLEANUP_NOT_AGGREGATED", error.stack ?? error);
	}
	expect(JSON.stringify(cleanupEvents) === JSON.stringify(["remove", "release"]), "VERIFY_SELFTEST_CLEANUP_ORDER", cleanupEvents.join(","));
}

async function inspectSourceAndArguments() {
	const [source, runtimeSource, sealedRunnerSource, u7PlanSource] = await Promise.all([
		readFile(verifierPath, "utf8"), readFile(runtimePath, "utf8"), readFile(sealedC6ARunnerPath, "utf8"), readFile(u7PlanPath, "utf8"),
	]);
	expect(!source.includes("...process.env") && !runtimeSource.includes("...process.env"), "VERIFY_SELFTEST_PROCESS_ENV_SPREAD", "verification runtime modules");
	expect(!sealedRunnerSource.includes("...process.env") && !sealedRunnerSource.includes("checkout") &&
		!sealedRunnerSource.includes("worktree") && !sealedRunnerSource.includes("git archive"),
	"VERIFY_SELFTEST_SEALED_RUNNER_SOURCE_AUTHORITY", "sealed runner must remain raw-blob and allowlisted-env only");
	expect(
		occurrenceCount(sealedRunnerSource, 'runGit(git, repositoryRoot, ["cat-file", "--batch"], {') === 1 &&
		occurrenceCount(sealedRunnerSource, 'runGit(git, repositoryRoot, ["hash-object", "--stdin-paths", "--no-filters"], {') === 1 &&
		occurrenceCount(sealedRunnerSource, "loadGitBlobBatch(") === 2 &&
		occurrenceCount(sealedRunnerSource, "validateSnapshotGitHashes(") === 2 &&
		sealedRunnerSource.includes("admittedDarwinGitStderr") &&
		sealedRunnerSource.includes("describeRejectedGitResult") &&
		sealedRunnerSource.includes("stderr_prefix_hex=") &&
		sealedRunnerSource.includes("stderr_truncated=") &&
		!sealedRunnerSource.includes('["cat-file", "blob", entry.oid]') &&
		!sealedRunnerSource.includes('["hash-object", "--stdin"]'),
		"VERIFY_SELFTEST_SEALED_RUNNER_BATCH_CARDINALITY",
		"sealed runner must use exactly one bounded blob batch and exactly one no-filter path-hash batch with bounded binary-safe rejection diagnostics",
	);
	const sealedRunnerSelftest = spawnSync(process.execPath, [sealedC6ARunnerPath, "--self-test"], {
		cwd: repositoryRoot, encoding: "utf8", timeout: 30_000, maxBuffer: 4 * 1024 * 1024,
	});
	expect(
		sealedRunnerSelftest.status === 0 && sealedRunnerSelftest.signal === null && sealedRunnerSelftest.stderr === "" &&
		sealedRunnerSelftest.stdout === "sealed C6A architecture runner self-test passed: bounded Git blob/hash batches and Darwin diagnostics, raw blobs, path/mode authority, literal-filter immunity, and two-mode routing\n",
		"VERIFY_SELFTEST_SEALED_RUNNER_SELFTEST",
		`${sealedRunnerSelftest.status ?? sealedRunnerSelftest.signal}:${sealedRunnerSelftest.stderr || sealedRunnerSelftest.stdout || sealedRunnerSelftest.error}`,
	);
	let importFixture = await realpath(await mkdtemp(join(tmpdir(), "countershape-verify-import-selftest-")));
	try {
		const toolsDirectory = join(importFixture, "tools");
		await mkdir(toolsDirectory, { mode: 0o700 });
		const copiedRuntime = join(toolsDirectory, "verify-runtime-authority.mjs");
		const copiedVerifier = join(toolsDirectory, "verify-current.mjs");
		await Promise.all([
			writeFile(copiedRuntime, runtimeSource, { mode: 0o600 }),
			writeFile(copiedVerifier, source, { mode: 0o600 }),
			writeFile(join(toolsDirectory, "check-u7-plan.mjs"), u7PlanSource, { mode: 0o600 }),
		]);
		const inert = spawnSync(process.execPath, [
			"--input-type=module", "--eval",
			`await import(${JSON.stringify(pathToFileURL(copiedVerifier).href)}); process.stdout.write("verifier import inert\\n");`,
		], { encoding: "utf8", timeout: 10_000 });
		expect(inert.status === 0 && inert.stdout === "verifier import inert\n" && inert.stderr === "", "VERIFY_SELFTEST_VERIFIER_IMPORT_EFFECT", `${inert.stdout ?? ""}${inert.stderr ?? ""}`);
		let artifactPresent = false;
		try {
			await lstat(join(importFixture, ".countershape"));
			artifactPresent = true;
		} catch (error) {
			if (error.code !== "ENOENT") throw error;
		}
		expect(!artifactPresent, "VERIFY_SELFTEST_RUNTIME_IMPORT_ARTIFACT", importFixture);
	} finally {
		await rm(importFixture, { recursive: true, force: true });
	}
	const result = spawnSync(process.execPath, [verifierPath, "--unexpected"], { encoding: "utf8", timeout: 10_000 });
	const output = `${result.stdout ?? ""}${result.stderr ?? ""}`;
	expect(
		result.error === undefined && result.signal === null && Number.isInteger(result.status) && result.status !== 0 &&
		/(?:^|\n)VerificationError: VERIFY_ARGUMENTS:/u.test(output),
		"VERIFY_SELFTEST_ARGUMENTS_FALSE_GREEN",
		output,
	);
	let fixture = await realpath(await mkdtemp(join(tmpdir(), "countershape-verify-symlink-main-")));
	try {
		const alias = join(fixture, "verify-current-link.mjs");
		await symlink(verifierPath, alias);
		expect(await realpath(alias) === verifierPath, "VERIFY_SELFTEST_SYMLINK_REALPATH", alias);
		const linked = spawnSync(process.execPath, [alias], { encoding: "utf8", timeout: 10_000 });
		const linkedOutput = `${linked.stdout ?? ""}${linked.stderr ?? ""}`;
		expect(
			linked.status !== 0 && linked.signal === null && linked.stdout === "" &&
			/(?:^|\n)VerificationError: VERIFY_NONCANONICAL_ENTRY:/u.test(linkedOutput),
			"VERIFY_SELFTEST_NONCANONICAL_ENTRY_FALSE_GREEN",
			linkedOutput,
		);
		for (const forbidden of ["VERIFY_ARGUMENTS", "VERIFY_PLATFORM_", "VERIFY_ALREADY_RUNNING", "VERIFY_STALE_LOCK", "VERIFY_PLAN_", "VERIFY_TOOL_"]) {
			expect(!linkedOutput.includes(forbidden), "VERIFY_SELFTEST_NONCANONICAL_ENTRY_ORDER", `${forbidden}:${linkedOutput}`);
		}
	} finally {
		await rm(fixture, { recursive: true, force: true });
	}
}

async function main() {
	if (process.argv.length !== 2) fail("VERIFY_SELFTEST_ARGUMENTS", "no arguments are accepted");
	await inspectRosters();
	inspectEnvironment();
	await inspectPackagePartition();
	await inspectChildArguments();
	await inspectFailClosedExecution();
	await inspectU7PhaseExecution();
	await inspectFilesystemGuards();
	await inspectVerificationLock();
	await inspectVerificationLockSubprocesses();
	await inspectResourceFinalization();
	await inspectSourceAndArguments();
	const sourceDigest = createHash("sha256")
		.update(await readFile(verifierPath))
		.update(await readFile(runtimePath))
		.update(await readFile(sealedC6ARunnerPath))
		.update(await readFile(u7PlanPath))
		.digest("hex");
	process.stdout.write(`verification runner self-test passed: exact C5/U7 rosters/env/package partition, visible-capsule U7P-D phase authority with pre-Go admission and terminal live recheck, explicit live/sealed-C6A root authority, bounded-batch raw-blob sealed runner self-test with narrow Darwin diagnostics, fail-closed status/signal/error/marker, bounded tool admission, framed child output, canonical plan/tool paths, private root isolation, exclusive lock integrity plus real subprocess contention/stale recovery, inert verifier imports, noncanonical-entry refusal, cleanup aggregation, historical nonexecution, and artifact refusal (sources sha256:${sourceDigest})\n`);
}

main().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
