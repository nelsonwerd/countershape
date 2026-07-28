#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { createHash, randomBytes } from "node:crypto";
import { constants } from "node:fs";
import { chmod, lstat, mkdir, mkdtemp, open, realpath, rm, symlink, unlink } from "node:fs/promises";
import { platform } from "node:os";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");

const maxChildOutput = 64 * 1024 * 1024;
const childTimeoutMS = 20 * 60 * 1000;
const defaultChildTimeoutMS = childTimeoutMS;
const extendedChildTimeoutMS = 30 * 60 * 1000;
const maxAdmittedToolBytes = 512 * 1024 * 1024;
const toolReadChunkBytes = 1024 * 1024;
const lockRecordMaxBytes = 4096;
const lockSchemaVersion = "countershape/verify-current-lock/v1";
const toolSpecifications = Object.freeze([
	Object.freeze({ name: "go", variable: "COUNTERSHAPE_GO", fallback: "/opt/homebrew/bin/go" }),
	Object.freeze({ name: "node", variable: null, fallback: null }),
	Object.freeze({ name: "git", variable: "COUNTERSHAPE_GIT", fallback: "/usr/bin/git" }),
	Object.freeze({ name: "sh", variable: "COUNTERSHAPE_SH", fallback: "/bin/sh" }),
	Object.freeze({ name: "cc", variable: "COUNTERSHAPE_CC", fallback: "/usr/bin/clang" }),
	Object.freeze({ name: "cxx", variable: "COUNTERSHAPE_CXX", fallback: "/usr/bin/clang++" }),
]);

export class VerificationRuntimeError extends Error {
	constructor(code, detail) {
		super(`${code}: ${detail}`);
		this.code = code;
	}
}

function sameOpenedFile(left, right) {
	return left.dev === right.dev && left.ino === right.ino && left.size === right.size &&
		left.mode === right.mode && left.mtimeMs === right.mtimeMs && left.ctimeMs === right.ctimeMs;
}

function sameFileIdentity(left, right) {
	return left.dev === right.dev && left.ino === right.ino;
}

function effectiveUID() {
	if (typeof process.geteuid !== "function") {
		throw new VerificationRuntimeError("VERIFY_EFFECTIVE_UID_UNAVAILABLE", platform());
	}
	return process.geteuid();
}

async function requireOwnedDirectory(path, code, { exactMode, writableMode = 0 } = {}) {
	let handle;
	try {
		handle = await open(path, constants.O_RDONLY | (constants.O_DIRECTORY ?? 0) | (constants.O_NOFOLLOW ?? 0));
		const opened = await handle.stat();
		const atPath = await lstat(path);
		const canonical = await realpath(path);
		if (!opened.isDirectory() || atPath.isSymbolicLink() || !sameFileIdentity(opened, atPath) || canonical !== path ||
			opened.uid !== effectiveUID() || (exactMode !== undefined && (opened.mode & 0o777) !== exactMode) ||
			(writableMode !== 0 && (opened.mode & writableMode) !== 0)) {
			throw new VerificationRuntimeError(code, path);
		}
	} catch (error) {
		if (error instanceof VerificationRuntimeError) throw error;
		throw new VerificationRuntimeError(code, `${path}: ${error.code ?? error.message}`);
	} finally {
		await handle?.close();
	}
}

export async function createPrivateBase(root = repositoryRoot) {
	const absoluteRoot = resolve(root);
	const canonicalRoot = await realpath(absoluteRoot);
	if (absoluteRoot !== canonicalRoot) {
		throw new VerificationRuntimeError("VERIFY_REPOSITORY_ROOT_NOT_CANONICAL", `${absoluteRoot} != ${canonicalRoot}`);
	}
	const artifactRoot = join(canonicalRoot, ".countershape");
	try {
		await mkdir(artifactRoot, { mode: 0o700 });
	} catch (error) {
		if (error.code !== "EEXIST") {
			throw new VerificationRuntimeError("VERIFY_ARTIFACT_ROOT_CREATE_FAILED", `${artifactRoot}: ${error.code ?? error.message}`);
		}
	}
	await requireOwnedDirectory(artifactRoot, "VERIFY_ARTIFACT_ROOT_INVALID", { writableMode: 0o022 });
	const base = join(artifactRoot, "verify-current");
	try {
		await mkdir(base, { mode: 0o700 });
	} catch (error) {
		if (error.code !== "EEXIST") {
			throw new VerificationRuntimeError("VERIFY_PRIVATE_BASE_CREATE_FAILED", `${base}: ${error.code ?? error.message}`);
		}
	}
	await requireOwnedDirectory(base, "VERIFY_PRIVATE_BASE_INVALID", { exactMode: 0o700 });
	return base;
}

function canonicalLockRecord(record) {
	return `${JSON.stringify({
		created_at_unix_ms: record.created_at_unix_ms,
		nonce: record.nonce,
		pid: record.pid,
		repository_root_sha256: record.repository_root_sha256,
		schema_version: record.schema_version,
	})}\n`;
}

async function readBoundedHandle(handle, code, path) {
	const before = await handle.stat();
	if (!before.isFile() || before.size <= 0 || before.size > lockRecordMaxBytes) {
		throw new VerificationRuntimeError(code, `${path}: invalid byte length ${before.size}`);
	}
	const buffer = Buffer.alloc(before.size);
	let offset = 0;
	while (offset < buffer.length) {
		const { bytesRead } = await handle.read(buffer, offset, buffer.length - offset, offset);
		if (bytesRead === 0) throw new VerificationRuntimeError(code, `${path}: unexpected EOF at ${offset}/${buffer.length}`);
		offset += bytesRead;
	}
	const after = await handle.stat();
	if (!sameOpenedFile(before, after)) throw new VerificationRuntimeError(code, `${path}: changed while reading`);
	return buffer.toString("utf8");
}

function validateLockRecord(text, expectedRootDigest, path) {
	let record;
	try {
		record = JSON.parse(text);
	} catch (error) {
		throw new VerificationRuntimeError("VERIFY_LOCK_INVALID", `${path}: ${error.message}`);
	}
	const expectedKeys = ["created_at_unix_ms", "nonce", "pid", "repository_root_sha256", "schema_version"];
	if (!record || typeof record !== "object" || Array.isArray(record) ||
		JSON.stringify(Object.keys(record)) !== JSON.stringify(expectedKeys) ||
		record.schema_version !== lockSchemaVersion || !Number.isSafeInteger(record.pid) || record.pid <= 0 ||
		!Number.isSafeInteger(record.created_at_unix_ms) || record.created_at_unix_ms < 0 ||
		!(/^[0-9a-f]{32}$/u.test(record.nonce)) || record.repository_root_sha256 !== expectedRootDigest ||
		canonicalLockRecord(record) !== text) {
		throw new VerificationRuntimeError("VERIFY_LOCK_INVALID", path);
	}
	return Object.freeze(record);
}

async function inspectExistingLock(path, expectedRootDigest) {
	let handle;
	try {
		const before = await lstat(path);
		if (!before.isFile() || before.isSymbolicLink() || before.uid !== effectiveUID() || (before.mode & 0o777) !== 0o600) {
			throw new VerificationRuntimeError("VERIFY_LOCK_INVALID", path);
		}
		handle = await open(path, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
		const opened = await handle.stat();
		if (!opened.isFile() || !sameFileIdentity(before, opened)) {
			throw new VerificationRuntimeError("VERIFY_LOCK_INVALID", path);
		}
		const text = await readBoundedHandle(handle, "VERIFY_LOCK_INVALID", path);
		const finalOpened = await handle.stat();
		const after = await lstat(path);
		if (after.isSymbolicLink() || !after.isFile() || after.uid !== effectiveUID() || (after.mode & 0o777) !== 0o600 ||
			!sameOpenedFile(finalOpened, after)) {
			throw new VerificationRuntimeError("VERIFY_LOCK_INVALID", path);
		}
		return validateLockRecord(text, expectedRootDigest, path);
	} catch (error) {
		if (error instanceof VerificationRuntimeError) throw error;
		throw new VerificationRuntimeError("VERIFY_LOCK_INVALID", `${path}: ${error.code ?? error.message}`);
	} finally {
		await handle?.close();
	}
}

export function processLiveness(pid, signal = process.kill) {
	try {
		signal(pid, 0);
		return "live";
	} catch (error) {
		if (error.code === "ESRCH") return "absent";
		return "indeterminate";
	}
}

export async function acquireVerificationLock(root = repositoryRoot, dependencies = {}) {
	const canonicalRoot = await realpath(resolve(root));
	const base = await createPrivateBase(canonicalRoot);
	const path = join(base, "active.lock");
	const repositoryRootDigest = createHash("sha256").update(canonicalRoot).digest("hex");
	const record = Object.freeze({
		created_at_unix_ms: dependencies.now?.() ?? Date.now(),
		nonce: dependencies.nonce ?? randomBytes(16).toString("hex"),
		pid: dependencies.pid ?? process.pid,
		repository_root_sha256: repositoryRootDigest,
		schema_version: lockSchemaVersion,
	});
	const serialized = canonicalLockRecord(record);
	validateLockRecord(serialized, repositoryRootDigest, path);
	let handle;
	try {
		handle = await open(path, constants.O_CREAT | constants.O_EXCL | constants.O_RDWR | (constants.O_NOFOLLOW ?? 0), 0o600);
	} catch (error) {
		if (error.code !== "EEXIST") {
			throw new VerificationRuntimeError("VERIFY_LOCK_CREATE_FAILED", `${path}: ${error.code ?? error.message}`);
		}
		const existing = await inspectExistingLock(path, repositoryRootDigest);
		const state = await (dependencies.liveness?.(existing.pid) ?? processLiveness(existing.pid));
		if (state === "absent") {
			throw new VerificationRuntimeError(
				"VERIFY_STALE_LOCK",
				`${path}: pid=${existing.pid}; inspect and unlink only after confirming no verifier is active`,
			);
		}
		throw new VerificationRuntimeError("VERIFY_ALREADY_RUNNING", `${path}: pid=${existing.pid}; liveness=${state}`);
	}
	try {
		const opened = await handle.stat();
		if (!opened.isFile() || opened.uid !== effectiveUID() || (opened.mode & 0o777) !== 0o600) {
			throw new VerificationRuntimeError("VERIFY_LOCK_INVALID", path);
		}
		await handle.writeFile(serialized, { encoding: "utf8" });
		await handle.sync();
		await dependencies.afterWrite?.({ path, record });
		const atPath = await lstat(path);
		if (atPath.isSymbolicLink() || !sameFileIdentity(opened, atPath)) {
			throw new VerificationRuntimeError("VERIFY_LOCK_INVALID", path);
		}
	} catch (error) {
		let created;
		try { created = await handle.stat(); } catch { /* preserve uncertain ownership */ }
		try {
			const atPath = await lstat(path);
			if (created && !atPath.isSymbolicLink() && sameFileIdentity(created, atPath)) await unlink(path);
		} catch { /* preserve the primary failure and uncertain path */ }
		let closeFailure;
		try { await handle.close(); } catch (failure) { closeFailure = failure; }
		if (closeFailure) throw new AggregateError([error, closeFailure], "lock acquisition and handle close both failed");
		if (error instanceof VerificationRuntimeError) throw error;
		throw new VerificationRuntimeError("VERIFY_LOCK_WRITE_FAILED", `${path}: ${error.code ?? error.message}`);
	}
	let released = false;
	const assertHeld = async () => {
		if (released) throw new VerificationRuntimeError("VERIFY_LOCK_INTEGRITY", `${path}: already released`);
		const opened = await handle.stat();
		const before = await lstat(path);
		if (!opened.isFile() || opened.uid !== effectiveUID() || (opened.mode & 0o777) !== 0o600 ||
			before.isSymbolicLink() || !before.isFile() || before.uid !== effectiveUID() || (before.mode & 0o777) !== 0o600 ||
			!sameFileIdentity(opened, before)) {
			throw new VerificationRuntimeError("VERIFY_LOCK_INTEGRITY", path);
		}
		const text = await readBoundedHandle(handle, "VERIFY_LOCK_INTEGRITY", path);
		const finalOpened = await handle.stat();
		const after = await lstat(path);
		if (after.isSymbolicLink() || !after.isFile() || after.uid !== effectiveUID() || (after.mode & 0o777) !== 0o600 ||
			!sameOpenedFile(finalOpened, after) || text !== serialized) {
			throw new VerificationRuntimeError("VERIFY_LOCK_INTEGRITY", path);
		}
	};
	return Object.freeze({
		path,
		pid: record.pid,
		assertHeld,
		async release() {
			if (released) return;
			let failure;
			try {
				await assertHeld();
				await unlink(path);
			} catch (error) {
				failure = error instanceof VerificationRuntimeError
					? error
					: new VerificationRuntimeError("VERIFY_LOCK_INTEGRITY", `${path}: ${error.code ?? error.message}`);
			}
			try {
				await handle.close();
			} catch (error) {
				failure ??= new VerificationRuntimeError("VERIFY_LOCK_CLOSE_FAILED", `${path}: ${error.code ?? error.message}`);
			}
			released = true;
			if (failure) throw failure;
		},
	});
}

async function digestOpenedTool(handle, opened, name, path) {
	if (!Number.isSafeInteger(opened.size) || opened.size <= 0 || opened.size > maxAdmittedToolBytes) {
		throw new VerificationRuntimeError("VERIFY_TOOL_SIZE_INVALID", `${name}: ${path}: ${opened.size}`);
	}
	const digest = createHash("sha256");
	const buffer = Buffer.alloc(Math.min(toolReadChunkBytes, opened.size));
	let offset = 0;
	while (offset < opened.size) {
		const count = Math.min(buffer.length, opened.size - offset);
		const { bytesRead } = await handle.read(buffer, 0, count, offset);
		if (bytesRead !== count) {
			throw new VerificationRuntimeError("VERIFY_TOOL_READ_INCOMPLETE", `${name}: ${path}: ${offset}/${opened.size}`);
		}
		digest.update(buffer.subarray(0, bytesRead));
		offset += bytesRead;
	}
	return digest.digest("hex");
}

export async function admitTool(name, suppliedPath) {
	if (!isAbsolute(suppliedPath)) {
		throw new VerificationRuntimeError("VERIFY_TOOL_PATH_NOT_ABSOLUTE", `${name}: ${suppliedPath}`);
	}
	if (/[\u0000-\u001f\u007f]/u.test(suppliedPath) || suppliedPath.includes(":")) {
		throw new VerificationRuntimeError("VERIFY_TOOL_PATH_UNSAFE", name);
	}
	let canonical;
	try {
		canonical = await realpath(suppliedPath);
	} catch (error) {
		throw new VerificationRuntimeError("VERIFY_TOOL_REALPATH_FAILED", `${name}: ${suppliedPath}: ${error.code ?? error.message}`);
	}
	let before;
	try {
		before = await lstat(canonical);
	} catch (error) {
		throw new VerificationRuntimeError("VERIFY_TOOL_LSTAT_FAILED", `${name}: ${canonical}: ${error.code ?? error.message}`);
	}
	if (!before.isFile() || before.isSymbolicLink() || (before.mode & 0o111) === 0) {
		throw new VerificationRuntimeError("VERIFY_TOOL_NOT_EXECUTABLE_REGULAR", `${name}: ${canonical}`);
	}
	if (!Number.isSafeInteger(before.size) || before.size <= 0 || before.size > maxAdmittedToolBytes) {
		throw new VerificationRuntimeError("VERIFY_TOOL_SIZE_INVALID", `${name}: ${canonical}: ${before.size}`);
	}
	let handle;
	try {
		handle = await open(canonical, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
		const opened = await handle.stat();
		if (!opened.isFile() || !sameOpenedFile(before, opened)) {
			throw new VerificationRuntimeError("VERIFY_TOOL_CHANGED", `${name}: ${canonical}`);
		}
		const sha256 = await digestOpenedTool(handle, opened, name, canonical);
		const after = await handle.stat();
		if (!sameOpenedFile(opened, after)) {
			throw new VerificationRuntimeError("VERIFY_TOOL_CHANGED", `${name}: ${canonical}`);
		}
		return Object.freeze({
			name,
			path: canonical,
			sha256,
			device: opened.dev,
			inode: opened.ino,
			size: opened.size,
		});
	} catch (error) {
		if (error instanceof VerificationRuntimeError) throw error;
		throw new VerificationRuntimeError("VERIFY_TOOL_OPEN_FAILED", `${name}: ${canonical}: ${error.code ?? error.message}`);
	} finally {
		await handle?.close();
	}
}

function sameAdmittedTool(left, right) {
	return left?.name === right?.name && left?.path === right?.path && left?.sha256 === right?.sha256 &&
		left?.device === right?.device && left?.inode === right?.inode && left?.size === right?.size;
}

export async function admitTools(environment = process.env, expected = null) {
	if (environment.COUNTERSHAPE_NODE) {
		if (!isAbsolute(environment.COUNTERSHAPE_NODE)) {
			throw new VerificationRuntimeError(
				"VERIFY_NODE_AUTHORITY_INVOKE_REQUIRED",
				"COUNTERSHAPE_NODE must name the Node runtime that invoked this verifier",
			);
		}
		let requested;
		let running;
		try {
			[requested, running] = await Promise.all([realpath(environment.COUNTERSHAPE_NODE), realpath(process.execPath)]);
		} catch (error) {
			throw new VerificationRuntimeError(
				"VERIFY_NODE_AUTHORITY_INVOKE_REQUIRED",
				`${environment.COUNTERSHAPE_NODE}: ${error.code ?? error.message}`,
			);
		}
		if (requested !== running) {
			throw new VerificationRuntimeError(
				"VERIFY_NODE_AUTHORITY_INVOKE_REQUIRED",
				`invoke ${environment.COUNTERSHAPE_NODE} instead of overriding a running verifier`,
			);
		}
	}
	const admitted = {};
	for (const specification of toolSpecifications) {
		const fallback = specification.name === "node" ? process.execPath : specification.fallback;
		const supplied = expected?.[specification.name]?.path ??
			(specification.variable ? (environment[specification.variable] || fallback) : fallback);
		admitted[specification.name] = await admitTool(specification.name, supplied);
		if (expected && !sameAdmittedTool(admitted[specification.name], expected[specification.name])) {
			throw new VerificationRuntimeError("VERIFY_TOOL_REVALIDATION_FAILED", specification.name);
		}
	}
	return Object.freeze(admitted);
}

function childToolNames(step) {
	if (!step || typeof step !== "object" || typeof step.tool !== "string") {
		throw new VerificationRuntimeError("VERIFY_PLAN_PRIMARY_TOOL_REQUIRED", step?.id ?? "unnamed step");
	}
	if (!Array.isArray(step.tools) || step.tools.length === 0) {
		throw new VerificationRuntimeError("VERIFY_PLAN_TOOL_ROSTER_REQUIRED", step.id ?? step.tool);
	}
	if (step.tools[0] !== step.tool) {
		throw new VerificationRuntimeError("VERIFY_PLAN_PRIMARY_TOOL_MISMATCH", `${step.id ?? step.tool}: ${step.tools[0]} != ${step.tool}`);
	}
	const known = new Set(toolSpecifications.map((specification) => specification.name));
	const seen = new Set();
	for (const name of step.tools) {
		if (typeof name !== "string" || !known.has(name)) {
			throw new VerificationRuntimeError("VERIFY_PLAN_TOOL_UNKNOWN", `${step.id ?? step.tool}: ${String(name)}`);
		}
		if (seen.has(name)) {
			throw new VerificationRuntimeError("VERIFY_PLAN_TOOL_DUPLICATE", `${step.id ?? step.tool}: ${name}`);
		}
		seen.add(name);
	}
	return [...step.tools];
}

async function revalidateStepTools(step, admitted) {
	for (const name of childToolNames(step)) {
		if (!admitted?.[name]) {
			throw new VerificationRuntimeError("VERIFY_TOOL_NOT_ADMITTED", `${step.id ?? step.tool}: ${name}`);
		}
		const current = await admitTool(admitted[name].name, admitted[name].path);
		if (!sameAdmittedTool(current, admitted[name])) {
			throw new VerificationRuntimeError("VERIFY_TOOL_REVALIDATION_FAILED", name);
		}
	}
}

async function requirePrivateDirectory(path, code) {
	const canonical = await realpath(path);
	const metadata = await lstat(canonical);
	if (canonical !== path || !metadata.isDirectory() || metadata.isSymbolicLink() || metadata.uid !== effectiveUID() ||
		(metadata.mode & 0o777) !== 0o700) {
		throw new VerificationRuntimeError(code, path);
	}
}

export async function createPrivateRoots(root = repositoryRoot, admitted) {
	const base = await createPrivateBase(root);
	let runRoot;
	try {
		runRoot = await mkdtemp(join(base, "run-"));
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
		]) {
			await symlink(target, join(paths.authorityBin, name));
		}
		return Object.freeze(paths);
	} catch (error) {
		const failure = error instanceof VerificationRuntimeError
			? error
			: new VerificationRuntimeError("VERIFY_PRIVATE_ROOT_CREATE_FAILED", error.code ?? error.message);
		if (!runRoot) throw failure;
		try {
			await rm(runRoot, { recursive: true, force: true });
		} catch (cleanupFailure) {
			throw new AggregateError([failure, cleanupFailure], "private root creation and cleanup both failed");
		}
		throw failure;
	}
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
		GOFLAGS: "-mod=readonly -buildvcs=false -p=1",
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

function childArguments(step, root = repositoryRoot) {
	const suffix = [...(step.args ?? [])];
	return step.path ? [resolve(root, step.path), ...suffix] : suffix;
}

function childTimeoutForExecutionPolicy(executionPolicy, supplied) {
	if (!supplied) return defaultChildTimeoutMS;
	if (executionPolicy === null || (typeof executionPolicy !== "object") ||
		Array.isArray(executionPolicy) || !Object.isFrozen(executionPolicy)) {
		throw new VerificationRuntimeError("VERIFY_CHILD_EXECUTION_POLICY_INVALID", "exact frozen record required");
	}
	const prototype = Object.getPrototypeOf(executionPolicy);
	const keys = Reflect.ownKeys(executionPolicy);
	const descriptor = Object.getOwnPropertyDescriptor(executionPolicy, "timeoutMS");
	if ((prototype !== Object.prototype && prototype !== null) ||
		keys.length !== 1 || keys[0] !== "timeoutMS" ||
		descriptor === undefined || !Object.hasOwn(descriptor, "value") ||
		descriptor.get !== undefined || descriptor.set !== undefined ||
		!Number.isSafeInteger(descriptor.value) || descriptor.value !== extendedChildTimeoutMS) {
		throw new VerificationRuntimeError(
			"VERIFY_CHILD_EXECUTION_POLICY_INVALID",
			`timeoutMS=${String(descriptor?.value)}`,
		);
	}
	return descriptor.value;
}

export async function childResult(step, admitted, childEnvironment, dependencies = {}, executionPolicy) {
	const childTimeoutMS = childTimeoutForExecutionPolicy(executionPolicy, arguments.length >= 5);
	const args = dependencies.args ?? childArguments(step);
	const revalidate = dependencies.revalidate ?? revalidateStepTools;
	const spawn = dependencies.spawn ?? spawnSync;
	await revalidate(step, admitted);
	const executable = admitted[step.tool].path;
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
	let revalidationFailure;
	try {
		await revalidate(step, admitted);
	} catch (error) {
		revalidationFailure = error;
	}
	if (spawnFailure && revalidationFailure) {
		throw new AggregateError([spawnFailure, revalidationFailure], "child spawn and authority revalidation both failed");
	}
	if (spawnFailure) throw spawnFailure;
	if (revalidationFailure) throw revalidationFailure;
	return Object.freeze({
		status: result.status,
		signal: result.signal,
		error: result.error,
		stdout: result.stdout ?? "",
		stderr: result.stderr ?? "",
	});
}

export async function finalizeVerificationResources(lock, runRoot, dependencies = {}) {
	const remove = dependencies.remove ?? rm;
	await lock.assertHeld();
	await remove(runRoot, { recursive: true, force: true });
	await lock.release();
}

export async function cleanupVerificationResources(lock, roots, dependencies = {}) {
	const remove = dependencies.remove ?? rm;
	const failures = [];
	const runRoot = typeof roots === "string" ? roots : roots?.runRoot;
	if (runRoot) {
		try {
			await remove(runRoot, { recursive: true, force: true });
		} catch (error) {
			failures.push(error);
		}
	}
	if (lock) {
		try {
			await lock.release();
		} catch (error) {
			failures.push(error);
		}
	}
	if (failures.length === 1) throw failures[0];
	if (failures.length > 1) throw new AggregateError(failures, "verification resource cleanup failed");
}
