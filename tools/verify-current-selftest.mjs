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
	childArguments,
	currentSteps,
	executeCurrentPlan,
	historicalOnly,
	packageArguments,
	partitionGoPackages,
	revalidatePackagePartition,
	rosterDigest,
	sensitiveGoPackages,
	validateRepositoryPlan,
} from "./verify-current.mjs";
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
const authorityNames = Object.freeze(["go", "node", "git", "sh", "cc", "cxx"]);
const sealedC4VParentRosterDigest = "7cf294fcbe8247e7a4de2d8996fb6b960780a35d018a8d00069af15940f0efb8";
const expectedRosterDigest = "0fab61318d55314e3accaeabbaf56cc049aae5755f3c3f41d6a36eba1a6c7af5";

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
	expect(digest !== sealedC4VParentRosterDigest, "VERIFY_SELFTEST_C4_ROSTER_NOT_EVOLVED", digest);
	const currentIDs = currentSteps.map((step) => step.id);
	expect(new Set(currentIDs).size === currentIDs.length, "VERIFY_SELFTEST_DUPLICATE_CURRENT", currentIDs.join(","));
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
	const historicalIDs = historicalOnly.map((row) => row.id);
	expect(new Set(historicalIDs).size === historicalIDs.length, "VERIFY_SELFTEST_DUPLICATE_HISTORICAL", historicalIDs.join(","));
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
		await expectCode(validateRepositoryPlan(fixture, [], []), "VERIFY_PLAN_FILE_MISSING");
		outside = await realpath(await mkdtemp(join(tmpdir(), "countershape-verify-selftest-outside-")));
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
	const [source, runtimeSource] = await Promise.all([readFile(verifierPath, "utf8"), readFile(runtimePath, "utf8")]);
	expect(!source.includes("...process.env") && !runtimeSource.includes("...process.env"), "VERIFY_SELFTEST_PROCESS_ENV_SPREAD", "verification runtime modules");
	let importFixture = await realpath(await mkdtemp(join(tmpdir(), "countershape-verify-import-selftest-")));
	try {
		const toolsDirectory = join(importFixture, "tools");
		await mkdir(toolsDirectory, { mode: 0o700 });
		const copiedRuntime = join(toolsDirectory, "verify-runtime-authority.mjs");
		const copiedVerifier = join(toolsDirectory, "verify-current.mjs");
		await Promise.all([
			writeFile(copiedRuntime, runtimeSource, { mode: 0o600 }),
			writeFile(copiedVerifier, source, { mode: 0o600 }),
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
	await inspectFilesystemGuards();
	await inspectVerificationLock();
	await inspectVerificationLockSubprocesses();
	await inspectResourceFinalization();
	await inspectSourceAndArguments();
	const sourceDigest = createHash("sha256")
		.update(await readFile(verifierPath))
		.update(await readFile(runtimePath))
		.digest("hex");
	process.stdout.write(`verification runner self-test passed: exact C4 rosters/env/package partition, fail-closed status/signal/error/marker, bounded tool admission, framed child output, canonical plan/tool paths, private root isolation, exclusive lock integrity plus real subprocess contention/stale recovery, inert verifier imports, noncanonical-entry refusal, cleanup aggregation, historical nonexecution, and artifact refusal (sources sha256:${sourceDigest})\n`);
}

main().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
