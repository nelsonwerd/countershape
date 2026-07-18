#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { chmod, mkdir, mkdtemp, readFile, readdir, realpath, rm, symlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import {
	VerificationError,
	admitTool,
	admitTools,
	assertNoDSStore,
	buildChildEnvironment,
	childArguments,
	childResult,
	childToolNames,
	currentSteps,
	executeCurrentPlan,
	historicalOnly,
	repositoryRoot,
	revalidateStepTools,
	revalidateTool,
	rosterDigest,
	toolSpecifications,
	validateRepositoryPlan,
} from "./verify-current.mjs";

const selftestPath = fileURLToPath(import.meta.url);
const verifierPath = resolve(dirname(selftestPath), "verify-current.mjs");
const expectedRosterDigest = "d94d54d15bf4a5cd0e0429aa1f8d0de0af0fdb0b14a46577102a9d0e9029f2b6";

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
	const historicalIDs = historicalOnly.map((row) => row.id);
	expect(new Set(historicalIDs).size === historicalIDs.length, "VERIFY_SELFTEST_DUPLICATE_HISTORICAL", historicalIDs.join(","));
	const knownTools = new Set(toolSpecifications.map((tool) => tool.name));
	for (const step of currentSteps.filter((candidate) => candidate.tool)) {
		const names = childToolNames(step);
		expect(names[0] === step.tool, "VERIFY_SELFTEST_PRIMARY_TOOL_MISMATCH", step.id);
		expect(new Set(names).size === names.length, "VERIFY_SELFTEST_DUPLICATE_STEP_TOOL", `${step.id}: ${names.join(",")}`);
		expect(names.every((name) => knownTools.has(name)), "VERIFY_SELFTEST_UNKNOWN_STEP_TOOL", `${step.id}: ${names.join(",")}`);
	}

	const toolEntries = (await readdir(resolve(repositoryRoot, "tools"))).sort();
	const mutationFiles = toolEntries.filter((name) => /^(?:mutate|test-mutate)-.+\.mjs$/u.test(name)).map((name) => `tools/${name}`);
	const admittedMutations = historicalOnly.flatMap((row) => row.scripts).filter((path) => /\/(?:mutate|test-mutate)-/u.test(path)).sort();
	expect(
		JSON.stringify(mutationFiles) === JSON.stringify(admittedMutations),
		"VERIFY_SELFTEST_ORPHAN_MUTATION_GATE",
		`actual=${mutationFiles.join(",")} admitted=${admittedMutations.join(",")}`,
	);

	const architectureFiles = toolEntries.filter((name) => /^check-.+\.mjs$/u.test(name)).map((name) => `tools/${name}`);
	const admittedArchitecture = [
		...currentSteps.filter((step) => step.id.startsWith("architecture-")).map((step) => step.path),
		...historicalOnly.filter((row) => row.id.startsWith("architecture-")).flatMap((row) => row.scripts),
	].sort();
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
				: { status: 0, signal: null, error: null, stdout: "", stderr: "" };
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
	await inspectChildArguments();
	await inspectFailClosedExecution();
	await inspectFilesystemGuards();
	await inspectSourceAndArguments();
	const sourceDigest = createHash("sha256").update(await readFile(verifierPath)).digest("hex");
	process.stdout.write(`verification runner self-test passed: exact rosters/env, fail-closed status/signal/error/marker, framed child output, canonical plan/tool paths, symlink main, historical nonexecution, and artifact refusal (source sha256:${sourceDigest})\n`);
}

main().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
