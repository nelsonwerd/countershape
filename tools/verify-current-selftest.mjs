#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { chmod, lstat, mkdir, mkdtemp, readFile, readdir, realpath, rm, symlink, unlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import {
	VerificationError,
	acquireVerificationLock,
	admitTool,
	admitTools,
	assertNoDSStore,
	buildChildEnvironment,
	childArguments,
	childResult,
	childToolNames,
	createPrivateBase,
	currentSteps,
	executeCurrentPlan,
	finalizeVerificationResources,
	historicalOnly,
	packageArguments,
	partitionGoPackages,
	repositoryRoot,
	revalidatePackagePartition,
	revalidateStepTools,
	revalidateTool,
	rosterDigest,
	sensitiveGoPackages,
	toolSpecifications,
	validateRepositoryPlan,
} from "./verify-current.mjs";

const selftestPath = fileURLToPath(import.meta.url);
const verifierPath = resolve(dirname(selftestPath), "verify-current.mjs");
const expectedRosterDigest = "7cf294fcbe8247e7a4de2d8996fb6b960780a35d018a8d00069af15940f0efb8";

function fail(code, detail) {
	throw new Error(`${code}: ${detail}`);
}

function expect(condition, code, detail) {
	if (!condition) fail(code, detail);
}

async function expectCode(promise, code) {
	try {
		await promise;
	} catch (error) {
		if (error instanceof VerificationError && error.code === code) return;
		fail("VERIFY_SELFTEST_WRONG_ERROR", `${code}: ${error.stack ?? error}`);
	}
	fail("VERIFY_SELFTEST_FALSE_NEGATIVE", code);
}

function fakeAuthorities() {
	return Object.freeze(Object.fromEntries(toolSpecifications.map((tool) => [
		tool.name,
		Object.freeze({ name: tool.name, path: `/authority/${tool.name}`, sha256: tool.name.padEnd(64, "0").slice(0, 64) }),
	])));
}

async function inspectRosters() {
	const digest = rosterDigest();
	expect(digest === expectedRosterDigest, "VERIFY_SELFTEST_ROSTER_DRIFT", `${digest} != ${expectedRosterDigest}`);
	const currentIDs = currentSteps.map((step) => step.id);
	expect(new Set(currentIDs).size === currentIDs.length, "VERIFY_SELFTEST_DUPLICATE_CURRENT", currentIDs.join(","));
	const finalizationSteps = currentSteps.filter((step) => step.kind === "finalization-guard");
	expect(
		finalizationSteps.length === 1 && currentSteps.at(-1) === finalizationSteps[0],
		"VERIFY_SELFTEST_FINALIZATION_NOT_UNIQUE_LAST",
		currentIDs.join(","),
	);
	const historicalIDs = historicalOnly.map((row) => row.id);
	expect(new Set(historicalIDs).size === historicalIDs.length, "VERIFY_SELFTEST_DUPLICATE_HISTORICAL", historicalIDs.join(","));
	const knownTools = new Set(toolSpecifications.map((tool) => tool.name));
	for (const step of currentSteps.filter((candidate) => candidate.tool)) {
		const names = childToolNames(step);
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
		"github.com/nelsonwerd/countershape/internal/emit/node/compiler",
		"github.com/nelsonwerd/countershape/internal/emit/node/program/v1",
		"github.com/nelsonwerd/countershape/internal/store",
		"github.com/nelsonwerd/countershape/internal/world",
		"github.com/nelsonwerd/countershape/testkit/studies/cli_precedence",
		"github.com/nelsonwerd/countershape/testkit/studies/http_invoices",
	]), "VERIFY_SELFTEST_SENSITIVE_PACKAGE_ROSTER", sensitiveGoPackages.join(","));
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
		marker: "P07B-C C1 architecture boundary OK",
	});
	exactStep("architecture-p07b-c-c1-selftest", {
		tool: "node",
		tools: ["node", "go"],
		path: "tools/check-p07b-c-architecture-selftest.mjs",
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
	exactStep("architecture-p07b-c-plan-selftest", {
		tool: "node",
		tools: ["node", "git"],
		path: "tools/check-p07b-c-plan.mjs",
		args: ["--self-test"],
		marker: "P07B-C evolved plan checker self-test passed:",
	});
	exactStep("architecture-p07b-c-c3p-receipt", {
		tool: "node",
		tools: ["node", "git"],
		path: "tools/check-p07b-c-c3p-receipt.mjs",
		marker: "P07B-C C3P receipt check passed: phase-specific source/receipt, scope, Git-note, and documentation authority are coherent",
	});
	exactStep("architecture-p07b-c-c3p-receipt-selftest", {
		tool: "node",
		tools: ["node", "git"],
		path: "tools/check-p07b-c-c3p-receipt.mjs",
		args: ["--self-test"],
		marker: "P07B-C C3P receipt checker self-test passed:",
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
	const shuffled = [sensitiveGoPackages[3], general, ...sensitiveGoPackages.slice(0, 3), ...sensitiveGoPackages.slice(4)];
	const partition = partitionGoPackages(`${shuffled.join("\n")}\n`);
	expect(partition.general.length === 1 && partition.general[0] === general, "VERIFY_SELFTEST_PACKAGE_GENERAL", partition.general.join(","));
	expect(
		JSON.stringify(partition.sensitive) === JSON.stringify(sensitiveGoPackages) &&
			partition.all.length === sensitiveGoPackages.length + 1,
		"VERIFY_SELFTEST_PACKAGE_EXACT_UNION",
		JSON.stringify(partition),
	);
	const generalStep = currentSteps.find((step) => step.id === "go-test-general");
	const sensitiveStep = currentSteps.find((step) => step.id === "go-test-sensitive-serial");
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
		JSON.stringify(packageArguments({ args: ["list", "./..."] }, partition)) === JSON.stringify(["list", "./..."]),
		"VERIFY_SELFTEST_NONPARTITION_ARGUMENT_DISPATCH",
		"non-partition step changed",
	);
	revalidatePackagePartition(partition, partitionGoPackages(`${[general, ...sensitiveGoPackages].join("\n")}\n`));
	await expectCode(
		Promise.resolve().then(() => packageArguments({ id: "missing", packageClass: "general", args: ["test"] })),
		"VERIFY_PACKAGE_PARTITION_UNAVAILABLE",
	);
	await expectCode(
		Promise.resolve().then(() => packageArguments({ id: "unknown", packageClass: "all", args: ["test"] }, partition)),
		"VERIFY_PACKAGE_PARTITION_UNAVAILABLE",
	);
	await expectCode(
		Promise.resolve().then(() => revalidatePackagePartition(partition, {
			...partition,
			all: [...partition.all, "github.com/nelsonwerd/countershape/new-package"],
		})),
		"VERIFY_PACKAGE_PARTITION_CHANGED",
	);
	for (const [name, invoke, code] of [
		["missing-sensitive", () => partitionGoPackages(`${[general, ...sensitiveGoPackages.slice(1)].join("\n")}\n`), "VERIFY_SENSITIVE_PACKAGE_ROSTER_INVALID"],
		["duplicate", () => partitionGoPackages(`${[general, general, ...sensitiveGoPackages].join("\n")}\n`), "VERIFY_PACKAGE_LIST_INVALID"],
		["foreign", () => partitionGoPackages(`${[general, ...sensitiveGoPackages, "example.invalid/foreign"].join("\n")}\n`), "VERIFY_PACKAGE_LIST_INVALID"],
		["missing-final-lf", () => partitionGoPackages([general, ...sensitiveGoPackages].join("\n")), "VERIFY_PACKAGE_LIST_INVALID"],
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
	await childResult(
		{ id: "argument-override", tool: "go", tools: ["go"] },
		fakeAuthorities(),
		{},
		{
			args: ["test", "example.invalid/package"],
			revalidate: async () => {},
			spawn: (_executable, args) => {
				overriddenArguments = args;
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
		JSON.stringify(childToolNames({ id: "nested", tool: "node", tools: ["node", "go"] })) === JSON.stringify(["node", "go"]),
		"VERIFY_SELFTEST_NESTED_TOOL_ROSTER",
		"nested admitted tool roster changed",
	);
	const authorities = fakeAuthorities();
	const visited = [];
	await revalidateStepTools(
		{ id: "nested", tool: "node", tools: ["node", "go"] },
		authorities,
		async (authority) => { visited.push(authority.name); },
	);
	expect(JSON.stringify(visited) === JSON.stringify(["node", "go"]), "VERIFY_SELFTEST_NESTED_TOOL_REVALIDATION", visited.join(","));
	for (const scenario of [
		{
			name: "nonzero",
			spawnResult: { status: 23, signal: null, error: null, stdout: "", stderr: "injected nonzero" },
		},
		{
			name: "spawn-error",
			spawnResult: { status: null, signal: null, error: new Error("injected spawn error"), stdout: "", stderr: "" },
		},
	]) {
		const events = [];
		const result = await childResult(
			{ id: scenario.name, tool: "node", tools: ["node", "go"], path: "tools/fixture.mjs" },
			authorities,
			{},
			{
				revalidate: async (step, admitted) => revalidateStepTools(
					step,
					admitted,
					async (authority) => { events.push(authority.name); },
				),
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
				revalidate: async (step, admitted) => revalidateStepTools(
					step,
					admitted,
					async (authority) => { thrownEvents.push(authority.name); },
				),
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
	await expectCode(
		Promise.resolve().then(() => childToolNames({ id: "missing-roster", tool: "node" })),
		"VERIFY_PLAN_TOOL_ROSTER_REQUIRED",
	);
	await expectCode(
		Promise.resolve().then(() => childToolNames({ id: "wrong-primary", tool: "node", tools: ["go", "node"] })),
		"VERIFY_PLAN_PRIMARY_TOOL_MISMATCH",
	);
	await expectCode(
		Promise.resolve().then(() => childToolNames({ id: "duplicate", tool: "node", tools: ["node", "node"] })),
		"VERIFY_PLAN_TOOL_DUPLICATE",
	);
	await expectCode(
		Promise.resolve().then(() => childToolNames({ id: "unknown", tool: "node", tools: ["node", "python"] })),
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
			return { status: 0, signal: null, error: null, stdout: step.marker ? `${step.marker}\n` : "", stderr: "" };
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
				: { status: 0, signal: null, error: null, stdout: step.marker ? `${step.marker}\n` : "", stderr: "" };
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
		await writeFile(join(fixture, "nested/.DS_Store"), "finder", { mode: 0o600 });
		await expectCode(assertNoDSStore(fixture), "VERIFY_FINDER_ARTIFACT_PRESENT");
		await rm(join(fixture, "nested/.DS_Store"));
		await assertNoDSStore(fixture);

		await mkdir(join(fixture, "tools"), { mode: 0o700 });
		await writeFile(join(fixture, "tools/verify-current.mjs"), "// fixture verifier\n", { mode: 0o600 });
		try {
			await validateRepositoryPlan(fixture, [{ id: "missing", tool: "node", tools: ["node"], path: "tools/missing.mjs" }], []);
			fail("VERIFY_SELFTEST_FALSE_NEGATIVE", "tools/missing.mjs");
		} catch (error) {
			expect(error instanceof VerificationError && error.code === "VERIFY_PLAN_FILE_MISSING" && error.message.includes("tools/missing.mjs"), "VERIFY_SELFTEST_WRONG_MISSING_PLAN_ERROR", error.stack ?? error);
		}

		outside = await realpath(await mkdtemp(join(tmpdir(), "countershape-verify-selftest-outside-")));
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
		await writeFile(executable, "#!/bin/sh\necho changed\n", { mode: 0o700 });
		await expectCode(revalidateTool(admitted), "VERIFY_TOOL_REVALIDATION_FAILED");
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
}

async function inspectSourceAndArguments() {
	const source = await readFile(verifierPath, "utf8");
	expect(!source.includes("...process.env"), "VERIFY_SELFTEST_PROCESS_ENV_SPREAD", "verify-current.mjs");
	const result = spawnSync(process.execPath, [verifierPath, "--unexpected"], { encoding: "utf8", timeout: 10_000 });
	const output = `${result.stdout ?? ""}${result.stderr ?? ""}`;
	expect(result.status !== 0 && output.includes("VERIFY_ARGUMENTS"), "VERIFY_SELFTEST_ARGUMENTS_FALSE_GREEN", output);
	let fixture = await realpath(await mkdtemp(join(tmpdir(), "countershape-verify-symlink-main-")));
	try {
		const alias = join(fixture, "verify-current-link.mjs");
		await symlink(verifierPath, alias);
		const linked = spawnSync(process.execPath, [alias, "--unexpected"], { encoding: "utf8", timeout: 10_000 });
		const linkedOutput = `${linked.stdout ?? ""}${linked.stderr ?? ""}`;
		expect(linked.status !== 0 && linkedOutput.includes("VERIFY_ARGUMENTS"), "VERIFY_SELFTEST_SYMLINK_MAIN_SKIPPED", linkedOutput);
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
	await inspectFilesystemGuards();
	await inspectVerificationLock();
	await inspectResourceFinalization();
	await inspectSourceAndArguments();
	const sourceDigest = createHash("sha256").update(await readFile(verifierPath)).digest("hex");
	process.stdout.write(`verification runner self-test passed: exact rosters/env/package partition, fail-closed status/signal/error/marker, framed child output, canonical plan/tool paths, exclusive lock integrity, symlink main, historical nonexecution, and artifact refusal (source sha256:${sourceDigest})\n`);
}

main().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
