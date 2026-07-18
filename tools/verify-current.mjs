#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { constants } from "node:fs";
import { chmod, lstat, mkdir, mkdtemp, open, readdir, realpath, rm, symlink } from "node:fs/promises";
import { arch, platform } from "node:os";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

export const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const verifierPath = resolve(repositoryRoot, "tools/verify-current.mjs");
const maxChildOutput = 64 * 1024 * 1024;
const childTimeoutMS = 20 * 60 * 1000;

export class VerificationError extends Error {
	constructor(code, detail) {
		super(`${code}: ${detail}`);
		this.code = code;
	}
}

export const toolSpecifications = Object.freeze([
	Object.freeze({ name: "go", variable: "COUNTERSHAPE_GO", fallback: "/opt/homebrew/bin/go" }),
	Object.freeze({ name: "node", variable: null, fallback: process.execPath }),
	Object.freeze({ name: "git", variable: "COUNTERSHAPE_GIT", fallback: "/usr/bin/git" }),
	Object.freeze({ name: "sh", variable: "COUNTERSHAPE_SH", fallback: "/bin/sh" }),
	Object.freeze({ name: "cc", variable: "COUNTERSHAPE_CC", fallback: "/usr/bin/clang" }),
	Object.freeze({ name: "cxx", variable: "COUNTERSHAPE_CXX", fallback: "/usr/bin/clang++" }),
]);

const goCommon = Object.freeze(["-mod=readonly", "-buildvcs=false"]);

export const currentSteps = Object.freeze([
	Object.freeze({ id: "workspace-no-ds-store", kind: "guard" }),
	Object.freeze({ id: "go-build", tool: "go", tools: Object.freeze(["go", "cc", "cxx"]), args: Object.freeze(["build", ...goCommon, "./..."]) }),
	Object.freeze({ id: "go-vet", tool: "go", tools: Object.freeze(["go", "cc", "cxx"]), args: Object.freeze(["vet", ...goCommon, "./..."]) }),
	Object.freeze({ id: "go-test-full", tool: "go", tools: Object.freeze(["go", "node", "git", "sh", "cc", "cxx"]), args: Object.freeze(["test", ...goCommon, "-count=1", "-p=2", "-timeout=20m", "./..."]) }),
	Object.freeze({ id: "verification-runner-selftest", tool: "node", tools: Object.freeze(["node"]), path: "tools/verify-current-selftest.mjs", marker: "verification runner self-test passed:" }),
	Object.freeze({ id: "architecture-u5", tool: "node", tools: Object.freeze(["node"]), path: "tools/check-u5-architecture.mjs", marker: "U5 architecture boundary OK" }),
	Object.freeze({ id: "architecture-u5-selftest", tool: "node", tools: Object.freeze(["node"]), path: "tools/check-u5-architecture-selftest.mjs", marker: "U5 architecture self-test passed:" }),
	Object.freeze({ id: "architecture-u6", tool: "node", tools: Object.freeze(["node"]), path: "tools/check-u6-architecture.mjs", marker: "U6 architecture boundary OK" }),
	Object.freeze({ id: "architecture-u6-selftest", tool: "node", tools: Object.freeze(["node"]), path: "tools/check-u6-architecture-selftest.mjs", marker: "U6 architecture checker self-test OK" }),
	Object.freeze({ id: "architecture-p07b-a1", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]), path: "tools/check-p07b-architecture.mjs", marker: "P07B source architecture boundary OK" }),
	Object.freeze({ id: "architecture-p07b-a1-selftest", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]), path: "tools/check-p07b-architecture-selftest.mjs", marker: "P07B architecture self-test OK" }),
	Object.freeze({
		id: "runtime-example-p07b-a2-2", tool: "node", tools: Object.freeze(["node", "go"]), path: "tools/generate-p07b-a2-runtime-example.mjs",
		args: Object.freeze(["--check"]), marker: "P07B A2.2 runtime ContractBundle example exact (",
	}),
	Object.freeze({
		id: "planning-example-p07b-a2-2", tool: "node", tools: Object.freeze(["node"]), path: "tools/generate-p07-planning-example.mjs",
		args: Object.freeze(["--exercise"]),
		marker: "P07 planning example: real A2.2 bundle conformed and intact entrypoint refused companion tamper before harness load",
	}),
	Object.freeze({
		id: "planning-validator-selftest", tool: "node", tools: Object.freeze(["node"]), path: "tools/validate-planning.mjs",
		args: Object.freeze(["--self-test"]), marker: "planning validator self-test: ok (",
	}),
	Object.freeze({
		id: "recovery-process-p07b-a2-2", tool: "node", tools: Object.freeze(["node", "go"]), path: "tools/verify-p07b-a2-recovery-process.mjs",
		marker: "P07B A2.2 fresh-process recovery OK",
	}),
	Object.freeze({
		id: "human-surface-p07b-a2-2-selftest", tool: "node", tools: Object.freeze(["node"]), path: "tools/capture-p07b-a2-human-surface.mjs",
		args: Object.freeze(["--self-test"]), marker: "P07B A2.2 human-surface renderer self-test OK",
	}),
	Object.freeze({
		id: "human-surface-p07b-a2-2-check", tool: "node", tools: Object.freeze(["node"]), path: "tools/capture-p07b-a2-human-surface.mjs",
		args: Object.freeze(["--check"]), marker: "P07B A2.2 human-surface capture exact",
	}),
	Object.freeze({ id: "architecture-p07b-a2-2", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]), path: "tools/check-p07b-a2-architecture.mjs", marker: "P07B A2.2 architecture boundary OK" }),
	Object.freeze({ id: "architecture-p07b-a2-2-selftest", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]), path: "tools/check-p07b-a2-architecture-selftest.mjs", marker: "P07B A2.2 architecture defensive self-test OK" }),
	Object.freeze({ id: "architecture-p07b-b", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]), path: "tools/check-p07b-b-architecture.mjs", marker: "P07B B architecture boundary OK" }),
	Object.freeze({ id: "architecture-p07b-b-selftest", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]), path: "tools/check-p07b-b-architecture-selftest.mjs", marker: "P07B B architecture defensive self-test OK" }),
	Object.freeze({ id: "authority-revalidation", kind: "authority-guard" }),
]);

export const historicalOnly = Object.freeze([
	Object.freeze({ id: "architecture-u1", scripts: Object.freeze(["tools/check-u1-boundary.mjs"]), status: "docs/status/U1.md" }),
	Object.freeze({ id: "architecture-u2", scripts: Object.freeze(["tools/check-u2-boundary.mjs"]), status: "docs/status/U2.md" }),
	Object.freeze({ id: "architecture-u3", scripts: Object.freeze(["tools/check-u3-architecture.mjs"]), status: "docs/status/U3.md" }),
	Object.freeze({ id: "architecture-u4", scripts: Object.freeze(["tools/check-u4-architecture.mjs"]), status: "docs/status/U4.md" }),
	Object.freeze({ id: "mutation-u1", scripts: Object.freeze(["tools/mutate-u1.mjs", "tools/test-mutate-u1.mjs"]), status: "docs/status/U1.md" }),
	Object.freeze({ id: "mutation-u2", scripts: Object.freeze(["tools/mutate-u2.mjs", "tools/test-mutate-u2.mjs"]), status: "docs/status/U4.md" }),
	Object.freeze({ id: "mutation-u3", scripts: Object.freeze(["tools/mutate-u3.mjs"]), status: "docs/status/U4.md" }),
	Object.freeze({ id: "mutation-u4", scripts: Object.freeze(["tools/mutate-u4.mjs"]), status: "docs/status/U4.md" }),
	Object.freeze({ id: "mutation-u5", scripts: Object.freeze(["tools/mutate-u5.mjs", "tools/test-mutate-u5.mjs"]), status: "docs/status/U5.md" }),
	Object.freeze({ id: "mutation-u6-p07a-b", scripts: Object.freeze(["tools/mutate-u6.mjs"]), status: "docs/status/P07A-RULING.md" }),
	Object.freeze({ id: "mutation-p07b-a1", scripts: Object.freeze(["tools/mutate-p07b.mjs"]), status: "docs/status/P07B-A1-SOURCE.md" }),
	Object.freeze({ id: "mutation-p07b-a2-1", scripts: Object.freeze(["tools/mutate-p07b-a2-authority.mjs"]), status: "docs/status/P07B-A2-1-AUTHORITY.md" }),
]);

function sameOpenedFile(left, right) {
	return left.dev === right.dev && left.ino === right.ino && left.size === right.size &&
		left.mode === right.mode && left.mtimeMs === right.mtimeMs;
}

export async function admitTool(name, suppliedPath) {
	if (!isAbsolute(suppliedPath)) {
		throw new VerificationError("VERIFY_TOOL_PATH_NOT_ABSOLUTE", `${name}: ${suppliedPath}`);
	}
	if (/[\u0000-\u001f\u007f]/u.test(suppliedPath) || suppliedPath.includes(":")) {
		throw new VerificationError("VERIFY_TOOL_PATH_UNSAFE", name);
	}
	let canonical;
	try {
		canonical = await realpath(suppliedPath);
	} catch (error) {
		throw new VerificationError("VERIFY_TOOL_REALPATH_FAILED", `${name}: ${suppliedPath}: ${error.code ?? error.message}`);
	}
	let before;
	try {
		before = await lstat(canonical);
	} catch (error) {
		throw new VerificationError("VERIFY_TOOL_LSTAT_FAILED", `${name}: ${canonical}: ${error.code ?? error.message}`);
	}
	if (!before.isFile() || before.isSymbolicLink() || (before.mode & 0o111) === 0) {
		throw new VerificationError("VERIFY_TOOL_NOT_EXECUTABLE_REGULAR", `${name}: ${canonical}`);
	}
	let handle;
	try {
		handle = await open(canonical, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
		const opened = await handle.stat();
		if (!opened.isFile() || !sameOpenedFile(before, opened)) {
			throw new VerificationError("VERIFY_TOOL_CHANGED", `${name}: ${canonical}`);
		}
		const bytes = await handle.readFile();
		const after = await handle.stat();
		if (!sameOpenedFile(opened, after)) {
			throw new VerificationError("VERIFY_TOOL_CHANGED", `${name}: ${canonical}`);
		}
		return Object.freeze({
			name,
			path: canonical,
			sha256: createHash("sha256").update(bytes).digest("hex"),
			device: opened.dev,
			inode: opened.ino,
			size: opened.size,
		});
	} catch (error) {
		if (error instanceof VerificationError) throw error;
		throw new VerificationError("VERIFY_TOOL_OPEN_FAILED", `${name}: ${canonical}: ${error.code ?? error.message}`);
	} finally {
		await handle?.close();
	}
}

export async function admitTools(environment = process.env) {
	if (environment.COUNTERSHAPE_NODE) {
		if (!isAbsolute(environment.COUNTERSHAPE_NODE)) {
			throw new VerificationError("VERIFY_NODE_AUTHORITY_INVOKE_REQUIRED", "COUNTERSHAPE_NODE must name the Node runtime that invoked this verifier");
		}
		const [requested, running] = await Promise.all([realpath(environment.COUNTERSHAPE_NODE), realpath(process.execPath)]);
		if (requested !== running) {
			throw new VerificationError("VERIFY_NODE_AUTHORITY_INVOKE_REQUIRED", `invoke ${environment.COUNTERSHAPE_NODE} tools/verify-current.mjs instead of overriding a running verifier`);
		}
	}
	const admitted = {};
	for (const specification of toolSpecifications) {
		const supplied = specification.variable ? (environment[specification.variable] || specification.fallback) : specification.fallback;
		admitted[specification.name] = await admitTool(specification.name, supplied);
	}
	return Object.freeze(admitted);
}

export async function revalidateTool(admitted) {
	const current = await admitTool(admitted.name, admitted.path);
	if (current.path !== admitted.path || current.sha256 !== admitted.sha256 || current.device !== admitted.device ||
		current.inode !== admitted.inode || current.size !== admitted.size) {
		throw new VerificationError("VERIFY_TOOL_REVALIDATION_FAILED", admitted.name);
	}
}

export async function revalidateTools(admitted) {
	for (const specification of toolSpecifications) await revalidateTool(admitted[specification.name]);
}

export function childToolNames(step) {
	if (!step || typeof step !== "object" || typeof step.tool !== "string") {
		throw new VerificationError("VERIFY_PLAN_PRIMARY_TOOL_REQUIRED", step?.id ?? "unnamed step");
	}
	if (!Array.isArray(step.tools) || step.tools.length === 0) {
		throw new VerificationError("VERIFY_PLAN_TOOL_ROSTER_REQUIRED", step.id ?? step.tool);
	}
	if (step.tools[0] !== step.tool) {
		throw new VerificationError("VERIFY_PLAN_PRIMARY_TOOL_MISMATCH", `${step.id ?? step.tool}: ${step.tools[0]} != ${step.tool}`);
	}
	const known = new Set(toolSpecifications.map((specification) => specification.name));
	const seen = new Set();
	for (const name of step.tools) {
		if (typeof name !== "string" || !known.has(name)) {
			throw new VerificationError("VERIFY_PLAN_TOOL_UNKNOWN", `${step.id ?? step.tool}: ${String(name)}`);
		}
		if (seen.has(name)) {
			throw new VerificationError("VERIFY_PLAN_TOOL_DUPLICATE", `${step.id ?? step.tool}: ${name}`);
		}
		seen.add(name);
	}
	return [...step.tools];
}

export async function revalidateStepTools(step, admitted, revalidate = revalidateTool) {
	for (const name of childToolNames(step)) {
		if (!admitted?.[name]) throw new VerificationError("VERIFY_TOOL_NOT_ADMITTED", `${step.id ?? step.tool}: ${name}`);
		await revalidate(admitted[name]);
	}
}

function slash(value) {
	return value.split(sep).join("/");
}

async function requireRegularRepositoryFile(root, relativePath) {
	const absolute = resolve(root, relativePath);
	const fromRoot = relative(root, absolute);
	if (fromRoot === ".." || fromRoot.startsWith(`..${sep}`)) {
		throw new VerificationError("VERIFY_PLAN_PATH_ESCAPE", relativePath);
	}
	const components = slash(relativePath).split("/");
	let cursor = root;
	for (let index = 0; index < components.length; index += 1) {
		const component = components[index];
		if (!component || component === "." || component === "..") {
			throw new VerificationError("VERIFY_PLAN_PATH_INVALID", relativePath);
		}
		cursor = join(cursor, component);
		let metadata;
		try {
			metadata = await lstat(cursor);
		} catch (error) {
			throw new VerificationError("VERIFY_PLAN_FILE_MISSING", `${relativePath}: ${error.code ?? error.message}`);
		}
		if (metadata.isSymbolicLink()) throw new VerificationError("VERIFY_PLAN_SYMLINK", relativePath);
		const final = index === components.length - 1;
		if ((!final && !metadata.isDirectory()) || (final && !metadata.isFile())) {
			throw new VerificationError("VERIFY_PLAN_FILE_NONREGULAR", relativePath);
		}
	}
}

export async function validateRepositoryPlan(root = repositoryRoot, steps = currentSteps, historical = historicalOnly) {
	const absoluteRoot = resolve(root);
	const canonicalRoot = await realpath(absoluteRoot);
	if (absoluteRoot !== canonicalRoot) throw new VerificationError("VERIFY_REPOSITORY_ROOT_NOT_CANONICAL", `${absoluteRoot} != ${canonicalRoot}`);
	const paths = new Set(["tools/verify-current.mjs"]);
	for (const step of steps) {
		if (step.tool) childToolNames(step);
		else if (Object.hasOwn(step, "tools")) throw new VerificationError("VERIFY_PLAN_PRIMARY_TOOL_REQUIRED", step.id ?? "unnamed step");
		if (step.path) paths.add(step.path);
	}
	for (const row of historical) {
		for (const path of row.scripts) paths.add(path);
		paths.add(row.status);
	}
	for (const path of [...paths].sort()) await requireRegularRepositoryFile(canonicalRoot, path);
}

const ignoredArtifactRoots = new Set([".git", ".didrun", ".didrun-history", ".countershape", "node_modules"]);

export async function assertNoDSStore(root = repositoryRoot) {
	const findings = [];
	async function walk(directory, relativeDirectory) {
		const entries = await readdir(directory, { withFileTypes: true });
		entries.sort((left, right) => left.name.localeCompare(right.name, "en"));
		for (const entry of entries) {
			const relativePath = relativeDirectory ? `${relativeDirectory}/${entry.name}` : entry.name;
			if (entry.name === ".DS_Store") findings.push(relativePath);
			if (entry.isDirectory() && !(relativeDirectory === "" && ignoredArtifactRoots.has(entry.name))) {
				await walk(join(directory, entry.name), relativePath);
			}
		}
	}
	await walk(root, "");
	if (findings.length > 0) {
		throw new VerificationError("VERIFY_FINDER_ARTIFACT_PRESENT", findings.map(slash).join(","));
	}
}

async function requirePrivateDirectory(path, code) {
	const canonical = await realpath(path);
	const metadata = await lstat(canonical);
	if (canonical !== path || !metadata.isDirectory() || metadata.isSymbolicLink() || (metadata.mode & 0o777) !== 0o700) {
		throw new VerificationError(code, path);
	}
}

export async function createPrivateRoots(root = repositoryRoot, admitted) {
	const base = resolve(root, ".countershape/verify-current");
	await mkdir(base, { recursive: true, mode: 0o700 });
	await chmod(base, 0o700);
	await requirePrivateDirectory(base, "VERIFY_PRIVATE_BASE_INVALID");
	const runRoot = await mkdtemp(join(base, "run-"));
	await chmod(runRoot, 0o700);
	const paths = { runRoot };
	for (const name of ["home", "tmp", "gotmp", "gocache", "gopath", "gomodcache", "authorityBin"]) {
		paths[name] = join(runRoot, name);
		await mkdir(paths[name], { mode: 0o700 });
		await requirePrivateDirectory(paths[name], "VERIFY_PRIVATE_DIRECTORY_INVALID");
	}
	for (const [name, target] of [
		["go", admitted.go.path], ["node", admitted.node.path], ["git", admitted.git.path], ["sh", admitted.sh.path],
		["cc", admitted.cc.path], ["c++", admitted.cxx.path],
	]) await symlink(target, join(paths.authorityBin, name));
	return Object.freeze(paths);
}

export function buildChildEnvironment(admitted, roots) {
	return Object.freeze({
		HOME: roots.home,
		TMPDIR: roots.tmp,
		GOTMPDIR: roots.gotmp,
		GOCACHE: roots.gocache,
		GOPATH: roots.gopath,
		GOMODCACHE: roots.gomodcache,
		GOENV: "off",
		GOWORK: "off",
		GOTOOLCHAIN: "local",
		GOPROXY: "off",
		GOSUMDB: "off",
		GOVCS: "*:off",
		GOFLAGS: "-mod=readonly -buildvcs=false",
		CGO_ENABLED: "1",
		CC: admitted.cc.path,
		CXX: admitted.cxx.path,
		GOMAXPROCS: "2",
		LANG: "C",
		LC_ALL: "C",
		TZ: "UTC",
		NO_COLOR: "1",
		PATH: `${roots.authorityBin}:/usr/bin:/bin`,
		COUNTERSHAPE_GO: admitted.go.path,
		COUNTERSHAPE_NODE: admitted.node.path,
		COUNTERSHAPE_GIT: admitted.git.path,
		COUNTERSHAPE_SH: admitted.sh.path,
		COUNTERSHAPE_CC: admitted.cc.path,
		COUNTERSHAPE_CXX: admitted.cxx.path,
	});
}

export function childArguments(step, root = repositoryRoot) {
	const suffix = [...(step.args ?? [])];
	return step.path ? [resolve(root, step.path), ...suffix] : suffix;
}

export async function childResult(step, admitted, childEnvironment, dependencies = {}) {
	const executable = admitted[step.tool].path;
	const args = childArguments(step);
	const revalidate = dependencies.revalidate ?? revalidateStepTools;
	const spawn = dependencies.spawn ?? spawnSync;
	await revalidate(step, admitted);
	let result;
	let spawnFailure;
	try {
		result = spawn(executable, args, {
			cwd: repositoryRoot,
			encoding: "utf8",
			env: childEnvironment,
			timeout: childTimeoutMS,
			maxBuffer: maxChildOutput,
		});
	} catch (error) {
		spawnFailure = error;
	}
	await revalidate(step, admitted);
	if (spawnFailure) throw spawnFailure;
	return {
		status: result.status,
		signal: result.signal,
		error: result.error,
		stdout: result.stdout ?? "",
		stderr: result.stderr ?? "",
	};
}

function writeChildFrames(write, step, stream, source) {
	if (!source) return;
	const lines = source.split(/\r?\n/u);
	if (lines.at(-1) === "") lines.pop();
	for (const line of lines) write(`CHILD ${step.id} ${stream} ${JSON.stringify(line)}\n`);
}

export function rosterDigest(steps = currentSteps, historical = historicalOnly) {
	return createHash("sha256").update(JSON.stringify({ steps, historical })).digest("hex");
}

export async function executeCurrentPlan({
	steps = currentSteps,
	historical = historicalOnly,
	admitted,
	childEnvironment,
	executor,
	write = (value) => process.stdout.write(value),
	clock = () => performance.now(),
}) {
	write(`COUNTERSHAPE_VERIFY_V1 platform=${platform()}/${arch()} jobs=2 roster_sha256:${rosterDigest(steps, historical)}\n`);
	for (const specification of toolSpecifications) {
		const tool = admitted[specification.name];
		write(`AUTHORITY ${specification.name} path=${JSON.stringify(tool.path)} sha256:${tool.sha256}\n`);
	}
	write(`ROSTER CURRENT ${steps.map((step) => step.id).join(",")}\n`);
	for (const row of historical) {
		write(`HISTORICAL-ONLY NOT-RUN ${row.id} scripts=${row.scripts.join(",")} status=${row.status}\n`);
	}
	let passed = 0;
	for (let index = 0; index < steps.length; index += 1) {
		const step = steps[index];
		const ordinal = String(index + 1).padStart(2, "0");
		const total = String(steps.length).padStart(2, "0");
		write(`CURRENT [${ordinal}/${total}] START ${step.id}\n`);
		const started = clock();
		let result;
		try {
			result = await executor(step, { admitted, childEnvironment });
		} catch (error) {
			result = { status: 1, signal: null, error, stdout: "", stderr: "" };
		}
		writeChildFrames(write, step, "STDOUT", result.stdout);
		writeChildFrames(write, step, "STDERR", result.stderr);
		if (!result.error && !result.signal && result.status === 0 && step.marker && !result.stdout.includes(step.marker)) {
			result = { ...result, status: 1, error: new VerificationError("VERIFY_SUCCESS_MARKER_MISSING", `${step.id}: ${step.marker}`) };
		}
		const duration = Math.max(0, Math.round(clock() - started));
		if (result.error || result.signal || result.status !== 0) {
			const outcome = result.error?.message ?? (result.signal ? `signal=${result.signal}` : `exit=${result.status}`);
			write(`CURRENT [${ordinal}/${total}] FAIL ${step.id} duration_ms=${duration} detail=${JSON.stringify(outcome)}\n`);
			write(`RESULT FAIL current=${passed}/${steps.length} historical_not_run=${historical.length}\n`);
			return 1;
		}
		passed += 1;
		write(`CURRENT [${ordinal}/${total}] PASS ${step.id} duration_ms=${duration}\n`);
	}
	write(`RESULT PASS current=${passed}/${steps.length} historical_not_run=${historical.length}\n`);
	return 0;
}

async function main() {
	if (process.argv.length !== 2) {
		throw new VerificationError("VERIFY_ARGUMENTS", "no arguments are accepted");
	}
	if (platform() !== "darwin" || arch() !== "arm64") {
		throw new VerificationError("VERIFY_PLATFORM_UNSUPPORTED", `${platform()}/${arch()}; Darwin reference baseline required`);
	}
	await validateRepositoryPlan();
	const admitted = await admitTools();
	const roots = await createPrivateRoots(repositoryRoot, admitted);
	try {
		const childEnvironment = buildChildEnvironment(admitted, roots);
		const status = await executeCurrentPlan({
			admitted,
			childEnvironment,
			executor: async (step) => {
				if (step.kind === "guard") {
					await assertNoDSStore();
					return { status: 0, signal: null, error: null, stdout: "workspace contains no .DS_Store artifacts\n", stderr: "" };
				}
				if (step.kind === "authority-guard") {
					await revalidateTools(admitted);
					return { status: 0, signal: null, error: null, stdout: "all admitted tool authorities revalidated\n", stderr: "" };
				}
				return await childResult(step, admitted, childEnvironment);
			},
		});
		process.exitCode = status;
	} finally {
		await rm(roots.runRoot, { recursive: true, force: true });
	}
}

const invokedModule = process.argv[1] ? await realpath(resolve(process.argv[1])) : "";
if (invokedModule === verifierPath) {
	main().catch((error) => {
		process.stderr.write(`${error.stack ?? error}\n`);
		process.exitCode = 1;
	});
}
