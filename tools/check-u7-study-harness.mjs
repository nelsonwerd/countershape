#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { constants as fsConstants } from "node:fs";
import { chmod, lstat, mkdir, open, readdir, realpath, writeFile } from "node:fs/promises";
import { arch, cpus, release, totalmem } from "node:os";
import { dirname, isAbsolute, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { isDeepStrictEqual } from "node:util";

const modulePath = fileURLToPath(import.meta.url);
const repositoryRoot = resolve(dirname(modulePath), "..");
const specificationPath = resolve(repositoryRoot, "spec/verification/u7-unit-paths.json");
const harnessRelativePath = "tools/check-u7-study-harness.mjs";
const nodePath = "/opt/homebrew/bin/node";
const goPath = "/opt/homebrew/bin/go";
const gitPath = "/usr/bin/git";
const timePath = "/usr/bin/time";
const domainsByPhase = Object.freeze({
	U7B: Object.freeze(["http"]),
	U7C: Object.freeze(["cli"]),
	U7D: Object.freeze(["http", "cli"]),
});
const finalRootByPhase = Object.freeze({ U7B: ".countershape/u7b-final", U7C: ".countershape/u7c-final", U7D: ".countershape/u7d-final" });
const driverByDomain = Object.freeze({ http: "tools/run-u7-http-study.mjs", cli: "tools/run-u7-cli-study.mjs" });
const phaseBudgets = Object.freeze([
	Object.freeze({ id: "fixture", per_run_trials: 1 }),
	Object.freeze({ id: "compile", per_run_trials: 1 }),
	Object.freeze({ id: "search", per_run_trials: 80 }),
	Object.freeze({ id: "confirm", per_run_trials: 8 }),
	Object.freeze({ id: "contract", per_run_trials: 10 }),
]);
const deterministicArtifactPaths = Object.freeze({
	source_spec_sha256: "deterministic/source-spec.json",
	world_plan_sha256: "deterministic/world-plan.json",
	ruling_sha256: "deterministic/ruling.json",
	decision_record_sha256: "deterministic/decision-record.json",
	contract_bundle_sha256: "deterministic/contract-bundle.json",
});
const freshArtifactPaths = Object.freeze({
	world_instance_sha256: "fresh/world-instance.json",
	attempts_sha256: "fresh/attempts.json",
	measurements_sha256: "fresh/measurements.json",
	captures_sha256: "fresh/captures.json",
	confirmation_sha256: "fresh/confirmation.json",
	contract_execution_target_sha256: "fresh/contract-execution-target.json",
	finalized_contract_run_sha256: "fresh/finalized-contract-run.json",
	contract_execution_sha256: "fresh/contract-execution.json",
});
const productResultSchema = "countershape/u7-study-domain-result/v1";
const trialSchema = "countershape/u7-study-trial/v1";
const artifactSchema = "countershape/u7-study-artifact/v1";
const harnessProtocol = "countershape/u7-study-harness/v1";
const observationAuthority = "U7P_FROZEN_HARNESS_DIRECT_PROCESS_GIT_AND_ARTIFACT_OBSERVATION";
const semanticCeiling = "ARTIFACT_BYTES_AND_SUBJECT_PROCESS_TOPOLOGY_OBSERVED_PRODUCT_SEMANTICS_AND_FULL_HARNESS_RESOURCES_NOT_INDEPENDENTLY_ESTABLISHED";
const driverProtocol = "FIXTURE_ONLY_NO_EVIDENCE_ROOT";
const phaseVerdicts = Object.freeze({
	U7B: "LOCAL_HTTP_REFERENCE_FUNCTIONAL_GREEN_SUBJECT_RESOURCE_OBSERVED",
	U7C: "LOCAL_CLI_REFERENCE_FUNCTIONAL_GREEN_SUBJECT_RESOURCE_OBSERVED",
	U7D: "LOCAL_REFERENCE_MILESTONE_FUNCTIONAL_GREEN_FULL_STUDY_RESOURCE_UNRECEIPTED",
});
const maxOutputBytes = 2 * 1024 * 1024;
const maxArtifactBytes = 16 * 1024 * 1024;
const commandTimeoutMs = 20 * 60 * 1000;
const admittedRuntimeTools = Object.freeze({
	node: Object.freeze({ name: "node", path: nodePath, realpath: "/opt/homebrew/Cellar/node/25.2.1/bin/node", sha256: "87989003817c5347d6bad48e46897e1b6509328bb7e8295d83cb6b40af836b9c", version: "v25.2.1" }),
	go: Object.freeze({ name: "go", path: goPath, realpath: "/opt/homebrew/Cellar/go/1.26.5/libexec/bin/go", sha256: "3f947495f00cb7f8088a5cfd694da8dc43869b33f5e7377b048fb18922ffb7e0", version: "go version go1.26.5 darwin/arm64" }),
	git: Object.freeze({ name: "git", path: gitPath, realpath: gitPath, sha256: "179301dcb41ea78accc3fa0048a7e6f6710d891945a751a34addd622020c1818", version: "git version 2.50.1 (Apple Git-155)" }),
	time: Object.freeze({ name: "time", path: timePath, realpath: timePath, sha256: "d2210b72e8c978748a0f0ac7d0819dda4dd411bb7a2e9730476038b0d695e7a2", version: "NO_STABLE_VERSION_INTERFACE" }),
});

export const protocolAuthority = Object.freeze({
	harness_protocol: harnessProtocol,
	product_result_schema: productResultSchema,
	trial_schema: trialSchema,
	artifact_schema: artifactSchema,
	observation_authority: observationAuthority,
	semantic_ceiling: semanticCeiling,
	driver_protocol: driverProtocol,
	phase_verdicts: phaseVerdicts,
	observer_prefix: Object.freeze([timePath, "-p", "-l", "-o"]),
	domains_by_phase: domainsByPhase,
	driver_by_domain: driverByDomain,
	phase_budgets: phaseBudgets,
	deterministic_artifact_paths: deterministicArtifactPaths,
	fresh_artifact_paths: freshArtifactPaths,
});

class HarnessError extends Error {
	constructor(code, detail) {
		super(`U7_STUDY_HARNESS_${code}: ${detail}`);
		this.code = code;
	}
}

class HarnessUsageError extends Error {
	constructor(detail) {
		super(`U7_STUDY_HARNESS_USAGE: ${detail}`);
	}
}

function fail(code, detail) {
	throw new HarnessError(code, detail);
}

function sha256(bytes) {
	return createHash("sha256").update(bytes).digest("hex");
}

function exactKeys(value, keys) {
	return value !== null && typeof value === "object" && !Array.isArray(value) &&
		isDeepStrictEqual(Object.keys(value).sort(), [...keys].sort());
}

function sameStat(left, right) {
	return left.dev === right.dev && left.ino === right.ino && left.size === right.size && left.mode === right.mode &&
		left.mtimeMs === right.mtimeMs && left.nlink === right.nlink;
}

function sameDirectoryIdentity(left, right) {
	return left.isDirectory() && right.isDirectory() && !left.isSymbolicLink() && !right.isSymbolicLink() &&
		left.dev === right.dev && left.ino === right.ino && left.mode === right.mode && left.nlink >= 2 && right.nlink >= 2;
}

function decodeUTF8(bytes, label) {
	try {
		return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
	} catch {
		fail("INVALID_UTF8", label);
	}
}

function hasUTF8BOM(bytes) {
	return bytes.length >= 3 && bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf;
}

async function readRegularNoFollow(path, maximum = maxArtifactBytes, allowEmpty = false, allowHardlinks = false) {
	let handle;
	try {
		handle = await open(path, fsConstants.O_RDONLY | fsConstants.O_NOFOLLOW);
		const before = await handle.stat();
		if (!before.isFile() || before.nlink < 1 || (!allowHardlinks && before.nlink !== 1) || before.size > maximum || (!allowEmpty && before.size === 0)) {
			fail("REGULAR_FILE", `${path}: bytes=${before.size} links=${before.nlink}`);
		}
		const bytes = await handle.readFile();
		const after = await handle.stat();
		if (!sameStat(before, after) || bytes.length !== before.size) fail("FILE_DRIFT", path);
		return Object.freeze({ bytes, stat: before });
	} catch (error) {
		if (error instanceof HarnessError) throw error;
		fail("FILE_OPEN", `${path}: ${error?.code ?? error?.message}`);
	} finally {
		await handle?.close();
	}
}

async function lstatOptional(path) {
	try { return await lstat(path); }
	catch (error) { if (error?.code === "ENOENT") return undefined; throw error; }
}

async function requireAbsent(path) {
	const metadata = await lstatOptional(path);
	if (metadata !== undefined) fail("PATH_PREEXISTS", path);
}

async function requirePrivateDirectory(path) {
	const metadata = await lstat(path);
	if (!metadata.isDirectory() || metadata.isSymbolicLink() || metadata.nlink < 2 || (metadata.mode & 0o777) !== 0o700 || await realpath(path) !== path) {
		fail("PRIVATE_DIRECTORY", path);
	}
}

async function requireEmptyDirectory(path) {
	await requirePrivateDirectory(path);
	const entries = await readdir(path, { withFileTypes: true });
	if (entries.length !== 0) fail("DIRECTORY_NOT_EMPTY", path);
}

function stablePath(root, child) {
	const absolute = resolve(root, child);
	if (absolute === root || !absolute.startsWith(`${root}${sep}`)) fail("PATH_ESCAPE", child);
	return absolute;
}

function canonicalJSONLine(value) {
	return Buffer.from(`${JSON.stringify(value)}\n`, "utf8");
}

function parseCanonicalJSONLine(bytes, label) {
	if (hasUTF8BOM(bytes)) fail("CANONICAL_JSON", label);
	const text = decodeUTF8(bytes, label);
	if (!text.endsWith("\n") || text.slice(0, -1).includes("\n") || text.startsWith("\ufeff")) fail("CANONICAL_JSON", label);
	let value;
	try { value = JSON.parse(text); }
	catch { fail("CANONICAL_JSON", label); }
	if (!canonicalJSONLine(value).equals(bytes)) fail("CANONICAL_JSON", label);
	return value;
}

function validDigest(value) {
	return typeof value === "string" && /^[0-9a-f]{64}$/u.test(value) && !/^0+$/u.test(value);
}

function byteCompare(left, right) {
	return Buffer.compare(Buffer.from(left, "utf8"), Buffer.from(right, "utf8"));
}

function validRelativePath(path) {
	return typeof path === "string" && path.length > 0 && path.length <= 4096 && !isAbsolute(path) &&
		!path.startsWith("./") && !path.includes("\\") && !path.includes("//") &&
		!path.split("/").some((part) => part === "" || part === "." || part === "..") &&
		!/[\u0000-\u001f\u007f]/u.test(path);
}

export function parseTimeReport(bytes) {
	if (hasUTF8BOM(bytes)) fail("TIME_REPORT", "encoding or newline");
	const text = decodeUTF8(bytes, "time report");
	if (text.includes("\r") || text.startsWith("\ufeff") || !text.endsWith("\n")) fail("TIME_REPORT", "encoding or newline");
	const lines = text.slice(0, -1).split("\n");
	const labels = [
		"maximum resident set size", "average shared memory size", "average unshared data size", "average unshared stack size",
		"page reclaims", "page faults", "swaps", "block input operations", "block output operations", "messages sent",
		"messages received", "signals received", "voluntary context switches", "involuntary context switches",
		"instructions retired", "cycles elapsed", "peak memory footprint",
	];
	if (lines.length !== labels.length + 3 || !/^real [0-9]+\.[0-9]{2}$/u.test(lines[0]) ||
		!/^user [0-9]+\.[0-9]{2}$/u.test(lines[1]) || !/^sys [0-9]+\.[0-9]{2}$/u.test(lines[2])) fail("TIME_REPORT", "header or cardinality");
	const values = new Map();
	for (const [index, label] of labels.entries()) {
		const match = /^\s*([0-9]+)\s+(.+)$/u.exec(lines[index + 3]);
		if (match === null || match[2] !== label) fail("TIME_REPORT", `row ${index}`);
		const value = Number(match[1]);
		if (!Number.isSafeInteger(value) || value < 0) fail("TIME_REPORT", label);
		values.set(label, value);
	}
	const realSeconds = Number(lines[0].slice("real ".length));
	const peakRSS = values.get("maximum resident set size");
	if (!Number.isFinite(realSeconds) || realSeconds < 0 || !Number.isSafeInteger(peakRSS) || peakRSS <= 0) fail("TIME_REPORT", "real or RSS");
	return Object.freeze({ raw: text, wall_time_ms: Math.max(1, Math.ceil(realSeconds * 1000)), peak_rss_bytes: peakRSS });
}

function scrubbedEnvironment(base, overrides = {}) {
	const admitted = [
		"PATH", "LANG", "LC_ALL", "TZ", "NO_COLOR", "HOME", "TMPDIR", "GOTMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE",
		"GOFLAGS", "GOMAXPROCS", "CGO_ENABLED", "COUNTERSHAPE_NODE", "COUNTERSHAPE_GO", "COUNTERSHAPE_GIT",
		"COUNTERSHAPE_SH", "COUNTERSHAPE_GOFMT", "COUNTERSHAPE_CC", "COUNTERSHAPE_CXX", "CC", "CXX",
	];
	const environment = {};
	for (const name of admitted) if (typeof base[name] === "string") environment[name] = base[name];
	Object.assign(environment, {
		PATH: "/usr/bin:/bin:/opt/homebrew/bin", LANG: "C", LC_ALL: "C", TZ: "UTC", NO_COLOR: "1",
		NODE_OPTIONS: "", NODE_PATH: "", GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null",
		GIT_NO_LAZY_FETCH: "1", GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0", GIT_NO_REPLACE_OBJECTS: "1",
	}, overrides);
	return environment;
}

function spawnBytes(executable, args, options = {}) {
	const result = spawnSync(executable, args, {
		cwd: options.cwd ?? repositoryRoot,
		env: options.env ?? scrubbedEnvironment(process.env),
		encoding: null,
		timeout: options.timeout ?? commandTimeoutMs,
		maxBuffer: maxOutputBytes,
		shell: false,
	});
	if (result.error) fail("SPAWN", `${options.label ?? executable}: ${result.error.code ?? result.error.message}`);
	if (result.signal !== null) fail("SPAWN_SIGNAL", `${options.label ?? executable}: ${result.signal}`);
	return Object.freeze({ status: result.status, stdout: Buffer.from(result.stdout ?? []), stderr: Buffer.from(result.stderr ?? []) });
}

async function executableDigest(path, allowHardlinks = false) {
	const result = await readRegularNoFollow(await realpath(path), 64 * 1024 * 1024, false, allowHardlinks);
	return sha256(result.bytes);
}

const stdoutStore = new WeakMap();

async function timedSpawn({ name, executable, args, cwd, env, observationRoot }) {
	const reportPath = stablePath(observationRoot, `${name}.time`);
	await requireAbsent(reportPath);
	const subjectDigestBefore = await executableDigest(executable);
	const observerDigestBefore = await executableDigest(timePath);
	const result = spawnBytes(timePath, ["-p", "-l", "-o", reportPath, executable, ...args], { cwd, env, label: name });
	if (result.status !== 0) fail("SUBJECT_EXIT", `${name}: ${result.status}`);
	if (result.stderr.length !== 0) fail("SUBJECT_STDERR", `${name}: ${result.stderr.length}`);
	const report = await readRegularNoFollow(reportPath, 64 * 1024);
	if ((report.stat.mode & 0o777) !== 0o600) fail("TIME_REPORT_MODE", reportPath);
	const parsed = parseTimeReport(report.bytes);
	if (await executableDigest(executable) !== subjectDigestBefore || await executableDigest(timePath) !== observerDigestBefore) fail("EXECUTABLE_DRIFT", name);
	const environmentRows = Object.keys(env).sort(byteCompare).map((key) => `${key}=${env[key]}`);
	const observation = Object.freeze({
		name,
		observer_argv: Object.freeze([timePath, "-p", "-l", "-o", reportPath, executable, ...args]),
		cwd,
		executable_path: executable,
		executable_sha256: subjectDigestBefore,
		observer_path: timePath,
		observer_sha256: observerDigestBefore,
		environment_sha256: sha256(Buffer.from(`${environmentRows.join("\n")}\n`, "utf8")),
		wall_time_ms: parsed.wall_time_ms,
		peak_rss_bytes: parsed.peak_rss_bytes,
		time_report_path: relative(repositoryRoot, reportPath).split(sep).join("/"),
		time_report_bytes: report.bytes.length,
		time_report_sha256: sha256(report.bytes),
		time_report: parsed.raw,
		stdout_bytes: result.stdout.length,
		stdout_sha256: sha256(result.stdout),
	});
	stdoutStore.set(observation, result.stdout);
	return observation;
}

function checkedGit(fixtureRoot, args, env, label, allowEmpty = false) {
	const result = spawnBytes(gitPath, ["-C", fixtureRoot, "--no-replace-objects", ...args], { cwd: repositoryRoot, env, label });
	if (result.status !== 0 || result.stderr.length !== 0 || (!allowEmpty && result.stdout.length === 0)) fail("GIT", label);
	return result.stdout;
}

function nulRecords(bytes, label) {
	if (bytes.length === 0) return [];
	if (bytes.at(-1) !== 0) fail("FIXTURE_GIT_SHAPE", `${label}: terminal NUL`);
	return decodeUTF8(bytes.subarray(0, -1), label).split("\0");
}

async function inspectFixture(fixtureRoot, env, allowedExtraFiles = []) {
	await requirePrivateDirectory(fixtureRoot);
	const gitDirectory = stablePath(fixtureRoot, ".git");
	await requirePrivateDirectory(gitDirectory);
	for (const path of ["objects/info/alternates", "shallow", "info/grafts", "info/sparse-checkout"]) {
		if (await lstatOptional(stablePath(gitDirectory, path)) !== undefined) fail("FIXTURE_GIT_TOPOLOGY", path);
	}
	if (await lstatOptional(stablePath(fixtureRoot, ".countershape")) !== undefined) fail("FIXTURE_RESERVED_STATE", fixtureRoot);
	const unsafeConfig = spawnBytes(gitPath, ["-C", fixtureRoot, "config", "--local", "--get-regexp", "^(core\\.fsmonitor|core\\.sparseCheckout|core\\.sparseCheckoutCone|extensions\\.worktreeConfig)$"], { cwd: repositoryRoot, env, label: "fixture unsafe config" });
	if (unsafeConfig.status !== 1 || unsafeConfig.stdout.length !== 0 || unsafeConfig.stderr.length !== 0) fail("FIXTURE_GIT_TOPOLOGY", "unsafe local config");
	const replaceRefs = checkedGit(fixtureRoot, ["for-each-ref", "--format=%(refname)", "refs/replace"], env, "fixture replace refs", true);
	if (replaceRefs.length !== 0) fail("FIXTURE_GIT_TOPOLOGY", "replace refs");
	const gitDir = decodeUTF8(checkedGit(fixtureRoot, ["rev-parse", "--path-format=absolute", "--git-dir"], env, "fixture git dir"), "fixture git dir").trimEnd();
	const commonDir = decodeUTF8(checkedGit(fixtureRoot, ["rev-parse", "--path-format=absolute", "--git-common-dir"], env, "fixture common dir"), "fixture common dir").trimEnd();
	const topLevel = decodeUTF8(checkedGit(fixtureRoot, ["rev-parse", "--show-toplevel"], env, "fixture top level"), "fixture top level").trimEnd();
	const head = decodeUTF8(checkedGit(fixtureRoot, ["rev-parse", "--verify", "HEAD^{commit}"], env, "fixture head"), "fixture head").trimEnd();
	const tree = decodeUTF8(checkedGit(fixtureRoot, ["rev-parse", "--verify", "HEAD^{tree}"], env, "fixture tree"), "fixture tree").trimEnd();
	const count = decodeUTF8(checkedGit(fixtureRoot, ["rev-list", "--count", "HEAD"], env, "fixture count"), "fixture count").trimEnd();
	const statusBytes = checkedGit(fixtureRoot, ["status", "--porcelain=v2", "-z", "--untracked-files=all"], env, "fixture status", true);
	if (gitDir !== gitDirectory || commonDir !== gitDirectory || topLevel !== fixtureRoot || !/^[0-9a-f]{40}$/u.test(head) ||
		!/^[0-9a-f]{40}$/u.test(tree) || count !== "1" || statusBytes.length !== 0) fail("FIXTURE_AUTHORITY", fixtureRoot);

	const commitText = decodeUTF8(checkedGit(fixtureRoot, ["cat-file", "commit", head], env, "fixture raw commit"), "fixture raw commit");
	const headerEnd = commitText.indexOf("\n\n");
	const headers = headerEnd < 0 ? [] : commitText.slice(0, headerEnd).split("\n");
	if (headers[0] !== `tree ${tree}` || headers.some((line) => line.startsWith("parent "))) fail("FIXTURE_HISTORY", fixtureRoot);

	const treeRows = nulRecords(checkedGit(fixtureRoot, ["ls-tree", "-r", "-z", "--full-tree", "HEAD"], env, "fixture tree inventory"), "fixture tree inventory").map((row) => {
		const match = /^(100644|100755) blob ([0-9a-f]{40})\t(.+)$/u.exec(row);
		if (match === null || !validRelativePath(match[3])) fail("FIXTURE_TREE_SHAPE", row);
		return Object.freeze({ mode: match[1], oid: match[2], path: match[3] });
	}).sort((left, right) => byteCompare(left.path, right.path));
	if (treeRows.length === 0 || new Set(treeRows.map((row) => row.path)).size !== treeRows.length ||
		!isDeepStrictEqual(treeRows.map((row) => row.path), [...treeRows.map((row) => row.path)].sort(byteCompare))) fail("FIXTURE_TREE_SHAPE", "order or uniqueness");
	const indexRows = nulRecords(checkedGit(fixtureRoot, ["ls-files", "--stage", "-z"], env, "fixture index"), "fixture index").map((row) => {
		const match = /^(100644|100755) ([0-9a-f]{40}) 0\t(.+)$/u.exec(row);
		if (match === null || !validRelativePath(match[3])) fail("FIXTURE_INDEX_SHAPE", row);
		return Object.freeze({ mode: match[1], oid: match[2], path: match[3] });
	}).sort((left, right) => byteCompare(left.path, right.path));
	if (!isDeepStrictEqual(indexRows, treeRows)) fail("FIXTURE_INDEX_SHAPE", "index differs from HEAD");
	for (const flag of ["-v", "-f"]) {
		const flags = nulRecords(checkedGit(fixtureRoot, ["ls-files", flag, "-z"], env, `fixture index flags ${flag}`), `fixture index flags ${flag}`);
		if (!isDeepStrictEqual(flags, treeRows.map((row) => `H ${row.path}`))) fail("FIXTURE_INDEX_FLAGS", flag);
	}

	const trackedPaths = new Set(treeRows.map((row) => row.path));
	const allowed = new Set(allowedExtraFiles);
	if (allowed.size !== allowedExtraFiles.length || allowedExtraFiles.some((path) => !validRelativePath(path) || trackedPaths.has(path))) fail("FIXTURE_EXTRA_POLICY", "invalid allowed path");
	const expectedDirectories = new Set();
	for (const path of [...trackedPaths, ...allowed]) {
		const parts = path.split("/");
		for (let index = 1; index < parts.length; index += 1) expectedDirectories.add(parts.slice(0, index).join("/"));
	}
	const observedFiles = [];
	const observedDirectories = [];
	async function walkWorktree(current, prefix) {
		const entries = (await readdir(current, { withFileTypes: true })).sort((left, right) => byteCompare(left.name, right.name));
		for (const entry of entries) {
			if (prefix === "" && entry.name === ".git") continue;
			const path = prefix === "" ? entry.name : `${prefix}/${entry.name}`;
			if (!validRelativePath(path) || entry.isSymbolicLink()) fail("FIXTURE_WORKTREE_SHAPE", path);
			const absolute = stablePath(fixtureRoot, path);
			if (entry.isDirectory()) {
				const metadata = await lstat(absolute);
				if ((metadata.mode & 0o777) !== 0o700 || metadata.nlink < 2) fail("FIXTURE_WORKTREE_MODE", path);
				observedDirectories.push(path);
				await walkWorktree(absolute, path);
			} else if (entry.isFile()) observedFiles.push(path);
			else fail("FIXTURE_WORKTREE_SHAPE", path);
		}
	}
	await walkWorktree(fixtureRoot, "");
	const expectedFiles = [...trackedPaths, ...allowed].sort(byteCompare);
	observedFiles.sort(byteCompare);
	observedDirectories.sort(byteCompare);
	if (!isDeepStrictEqual(observedFiles, expectedFiles) || !isDeepStrictEqual(observedDirectories, [...expectedDirectories].sort(byteCompare))) {
		fail("FIXTURE_WORKTREE_ROSTER", `${observedFiles.length}/${observedDirectories.length}`);
	}
	for (const row of treeRows) {
		const worktree = await readRegularNoFollow(stablePath(fixtureRoot, row.path), maxArtifactBytes, true);
		const executable = (worktree.stat.mode & 0o111) !== 0;
		if (executable !== (row.mode === "100755")) fail("FIXTURE_WORKTREE_MODE", row.path);
		const blob = checkedGit(fixtureRoot, ["cat-file", "blob", row.oid], env, `fixture blob ${row.path}`, true);
		if (!worktree.bytes.equals(blob)) fail("FIXTURE_WORKTREE_BYTES", row.path);
	}
	for (const path of allowed) {
		const extra = await readRegularNoFollow(stablePath(fixtureRoot, path), 128 * 1024 * 1024);
		if ((extra.stat.mode & 0o111) === 0) fail("FIXTURE_EXTRA_MODE", path);
	}
	return Object.freeze({
		head_commit: head,
		tree,
		commit_count: 1,
		git_directory: relative(repositoryRoot, gitDirectory).split(sep).join("/"),
		git_common_directory: relative(repositoryRoot, commonDir).split(sep).join("/"),
		git_alternates: "ABSENT",
		clean_status_bytes: 0,
		clean_status_sha256: sha256(statusBytes),
	});
}

function expectedTrialPaths() {
	const paths = [];
	for (const phase of phaseBudgets.slice(2)) {
		for (let index = 1; index <= phase.per_run_trials; index += 1) paths.push(`phases/${phase.id}/trial-${String(index).padStart(3, "0")}.json`);
	}
	return Object.freeze(paths);
}

const trialPaths = expectedTrialPaths();
const expectedEvidenceFiles = Object.freeze([
	...Object.values(deterministicArtifactPaths),
	...Object.values(freshArtifactPaths),
	...trialPaths,
].sort(byteCompare));

function expectedEvidenceDirectories() {
	const directories = new Set();
	for (const path of expectedEvidenceFiles) {
		const parts = path.split("/");
		for (let index = 1; index < parts.length; index += 1) directories.add(parts.slice(0, index).join("/"));
	}
	return Object.freeze([...directories].sort(byteCompare));
}

const expectedEvidenceDirs = expectedEvidenceDirectories();

async function walkEvidence(root) {
	await requirePrivateDirectory(root);
	const files = [];
	const directories = [];
	async function walk(current, prefix) {
		const entries = (await readdir(current, { withFileTypes: true })).sort((left, right) => byteCompare(left.name, right.name));
		for (const entry of entries) {
			const path = prefix === "" ? entry.name : `${prefix}/${entry.name}`;
			if (entry.isSymbolicLink()) fail("EVIDENCE_NONREGULAR", path);
			const absolute = stablePath(root, path);
			if (entry.isDirectory()) {
				const metadata = await lstat(absolute);
				if ((metadata.mode & 0o777) !== 0o700 || metadata.nlink < 2) fail("EVIDENCE_DIRECTORY", path);
				directories.push(path);
				await walk(absolute, path);
			} else if (entry.isFile()) {
				const record = await readRegularNoFollow(absolute, maxArtifactBytes);
				if ((record.stat.mode & 0o777) !== 0o600) fail("EVIDENCE_MODE", path);
				files.push(Object.freeze({ path, mode: "0600", bytes: record.bytes.length, sha256: sha256(record.bytes), raw: record.bytes }));
			} else fail("EVIDENCE_NONREGULAR", path);
		}
	}
	await walk(root, "");
	if (!isDeepStrictEqual(files.map((entry) => entry.path), expectedEvidenceFiles) || !isDeepStrictEqual(directories, expectedEvidenceDirs)) {
		fail("EVIDENCE_ROSTER", `${files.length}/${directories.length}`);
	}
	return Object.freeze(files);
}

function validateTrial(value, domain, ordinal, path) {
	const match = /^phases\/(search|confirm|contract)\/trial-([0-9]{3})\.json$/u.exec(path);
	if (match === null || !exactKeys(value, ["schema_version", "domain", "ordinal", "phase", "trial", "status"]) ||
		value.schema_version !== trialSchema || value.domain !== domain || value.ordinal !== ordinal || value.phase !== match[1] ||
		value.trial !== Number(match[2]) || value.status !== "GREEN") fail("TRIAL_ARTIFACT", `${domain}:${ordinal}:${path}`);
}

function validateArtifact(value, domain, ordinal, field, deterministic) {
	const keys = deterministic ? ["schema_version", "domain", "artifact", "payload"] : ["schema_version", "domain", "ordinal", "artifact", "payload"];
	if (!exactKeys(value, keys) || value.schema_version !== artifactSchema || value.domain !== domain || value.artifact !== field ||
		(!deterministic && value.ordinal !== ordinal) || value.payload === null || typeof value.payload !== "object" || Array.isArray(value.payload)) {
		fail("SEMANTIC_ARTIFACT", `${domain}:${ordinal}:${field}`);
	}
}

export async function collectEvidence(root, domain, ordinal) {
	const files = await walkEvidence(root);
	const byPath = new Map(files.map((entry) => [entry.path, entry]));
	for (const path of trialPaths) validateTrial(parseCanonicalJSONLine(byPath.get(path).raw, path), domain, ordinal, path);
	const deterministic = {};
	for (const [field, path] of Object.entries(deterministicArtifactPaths)) {
		validateArtifact(parseCanonicalJSONLine(byPath.get(path).raw, path), domain, ordinal, field, true);
		deterministic[field] = byPath.get(path).sha256;
	}
	const fresh = {};
	for (const [field, path] of Object.entries(freshArtifactPaths)) {
		validateArtifact(parseCanonicalJSONLine(byPath.get(path).raw, path), domain, ordinal, field, false);
		fresh[field] = byPath.get(path).sha256;
	}
	const manifest = files.map(({ path, mode, bytes, sha256: digest }) => Object.freeze({ path, mode, bytes, sha256: digest }));
	return Object.freeze({
		manifest: Object.freeze(manifest),
		manifest_sha256: sha256(canonicalJSONLine(manifest)),
		deterministic: Object.freeze(deterministic),
		fresh: Object.freeze(fresh),
	});
}

export function validateProductResult(value, domain, ordinal) {
	if (!exactKeys(value, ["schema_version", "domain", "ordinal", "status"]) || value.schema_version !== productResultSchema ||
		value.domain !== domain || value.ordinal !== ordinal || value.status !== "GREEN") fail("PRODUCT_RESULT", `${domain}:${ordinal}`);
	return Object.freeze({ ...value });
}

async function loadSpecification() {
	const record = await readRegularNoFollow(specificationPath, 1024 * 1024);
	let specification;
	try { specification = JSON.parse(decodeUTF8(record.bytes, "U7 specification")); }
	catch { fail("SPECIFICATION", "JSON"); }
	const contract = specification?.receipt_contract;
	const harness = contract?.study_harness;
	if (!exactKeys(harness, ["protocol", "protocol_sha256", "path", "sha256", "product_result_schema", "trial_schema", "artifact_schema", "observation_authority", "semantic_ceiling", "driver_protocol", "phase_verdicts", "observer_prefix", "driver_by_domain", "phase_budgets", "deterministic_artifact_paths", "fresh_artifact_paths"]) ||
		harness.protocol !== harnessProtocol || harness.path !== harnessRelativePath || !validDigest(harness.sha256) ||
		!validDigest(harness.protocol_sha256) || harness.protocol_sha256 !== sha256(canonicalJSONLine(protocolAuthority)) ||
		harness.product_result_schema !== productResultSchema || harness.trial_schema !== trialSchema || harness.artifact_schema !== artifactSchema ||
		harness.observation_authority !== observationAuthority || harness.semantic_ceiling !== semanticCeiling || harness.driver_protocol !== driverProtocol ||
		!isDeepStrictEqual(harness.phase_verdicts, phaseVerdicts) ||
		!isDeepStrictEqual(harness.observer_prefix, [timePath, "-p", "-l", "-o"]) ||
		!isDeepStrictEqual(harness.driver_by_domain, driverByDomain) || !isDeepStrictEqual(harness.phase_budgets, phaseBudgets) ||
		!isDeepStrictEqual(harness.deterministic_artifact_paths, deterministicArtifactPaths) || !isDeepStrictEqual(harness.fresh_artifact_paths, freshArtifactPaths)) {
		fail("SPECIFICATION", "study harness contract");
	}
	const toolRows = specification.runtime_authority?.admitted_tools;
	const tools = new Map(Array.isArray(toolRows) ? toolRows.map((tool) => [tool.name, tool]) : []);
	if (!Array.isArray(toolRows) || tools.size !== toolRows.length) fail("SPECIFICATION", "runtime tool uniqueness");
	for (const [name, expected] of Object.entries(admittedRuntimeTools)) {
		const row = tools.get(name);
		if (!exactKeys(row, ["name", "path", "realpath", "sha256", "version"]) || !isDeepStrictEqual(row, expected) ||
			await realpath(row.path) !== row.realpath || await executableDigest(row.path, true) !== row.sha256) fail("RUNTIME_TOOL", name);
	}
	if (await realpath(process.execPath) !== admittedRuntimeTools.node.realpath || process.version !== admittedRuntimeTools.node.version) fail("RUNTIME_TOOL", "executed node");
	for (const [name, args] of [["node", ["--version"]], ["go", ["version"]], ["git", ["--version"]]]) {
		const row = admittedRuntimeTools[name];
		const probe = spawnBytes(row.path, args, { label: `${name} version` });
		if (probe.status !== 0 || probe.stderr.length !== 0 || decodeUTF8(probe.stdout, `${name} version`).trimEnd() !== row.version) fail("RUNTIME_TOOL", `${name} version`);
	}
	const harnessBytes = await readRegularNoFollow(modulePath, 4 * 1024 * 1024);
	if (sha256(harnessBytes.bytes) !== harness.sha256) fail("HARNESS_IDENTITY", harness.sha256);
	return Object.freeze({ value: specification, bytes: record.bytes, harness, runtime_sha256: sha256(canonicalJSONLine(specification.runtime_authority)) });
}

async function createPrivateDirectory(path) {
	await requireAbsent(path);
	await mkdir(path, { mode: 0o700 });
	await chmod(path, 0o700);
	await requirePrivateDirectory(path);
}

function phaseRowsForRun() {
	return Object.freeze(phaseBudgets.map((phase) => Object.freeze({ id: phase.id, trial_count: phase.per_run_trials, trial_budget: phase.per_run_trials })));
}

async function runOne({ phase, domain, ordinal, studiesRoot, specification }) {
	const runRoot = stablePath(studiesRoot, `${domain}/run-${ordinal}`);
	await mkdir(dirname(runRoot), { recursive: true, mode: 0o700 });
	await createPrivateDirectory(runRoot);
	const fixtureRoot = stablePath(runRoot, "fixture");
	const evidenceRoot = stablePath(runRoot, "evidence");
	const observationRoot = stablePath(runRoot, "observations");
	const privateHome = stablePath(runRoot, "home");
	const privateTmp = stablePath(runRoot, "tmp");
	for (const path of [observationRoot, privateHome, privateTmp]) await createPrivateDirectory(path);
	const environment = scrubbedEnvironment(process.env, {
		HOME: privateHome,
		TMPDIR: privateTmp,
		COUNTERSHAPE_STUDY_DOMAIN: domain,
		COUNTERSHAPE_STUDY_ORDINAL: String(ordinal),
	});
	const driver = resolve(repositoryRoot, driverByDomain[domain]);
	const driverRecord = await readRegularNoFollow(driver, 4 * 1024 * 1024);
	if ((driverRecord.stat.mode & 0o777) !== 0o644) fail("DRIVER_MODE", driver);
	const driverDigest = sha256(driverRecord.bytes);
	const prepareArgs = [driver, "--prepare", "--domain", domain, "--ordinal", String(ordinal), "--fixture-root", fixtureRoot];
	const prepare = await timedSpawn({ name: "prepare", executable: nodePath, args: prepareArgs, cwd: repositoryRoot, env: environment, observationRoot });
	if (prepare.stdout_bytes !== 0) fail("PREPARE_STDOUT", `${domain}:${ordinal}`);
	if (await executableDigest(driver) !== driverDigest) fail("DRIVER_DRIFT", `${domain}:${ordinal}`);
	const fixtureBeforeBuild = await inspectFixture(fixtureRoot, environment);
	const binary = stablePath(fixtureRoot, "countershape");
	await requireAbsent(binary);
	const buildArgs = ["build", "-trimpath", "-mod=readonly", "-buildvcs=false", "-o", binary, "./cmd/countershape"];
	const compile = await timedSpawn({ name: "compile", executable: goPath, args: buildArgs, cwd: repositoryRoot, env: environment, observationRoot });
	if (compile.stdout_bytes !== 0) fail("COMPILE_STDOUT", `${domain}:${ordinal}`);
	const binaryRecord = await readRegularNoFollow(binary, 128 * 1024 * 1024);
	if ((binaryRecord.stat.mode & 0o111) === 0) fail("BINARY_MODE", binary);
	const binaryDigest = sha256(binaryRecord.bytes);
	const execution = Object.freeze({
		cwd: fixtureRoot,
		argv: Object.freeze([binary, "study", domain, "--json"]),
		executable_path: binary,
		executable_sha256: binaryDigest,
	});
	await createPrivateDirectory(evidenceRoot);
	await requireEmptyDirectory(evidenceRoot);
	const evidenceDirectoryBefore = await lstat(evidenceRoot);
	const executeEnvironment = Object.freeze({ ...environment, COUNTERSHAPE_EVIDENCE_ROOT: evidenceRoot });
	const execute = await timedSpawn({ name: "execute", executable: binary, args: ["study", domain, "--json"], cwd: fixtureRoot, env: executeEnvironment, observationRoot });
	const evidenceDirectoryAfter = await lstat(evidenceRoot);
	if (!sameDirectoryIdentity(evidenceDirectoryBefore, evidenceDirectoryAfter)) fail("EVIDENCE_ROOT_IDENTITY", `${domain}:${ordinal}`);
	const productResultPath = stablePath(runRoot, "product.stdout.json");
	const productObject = validateProductResult(parseCanonicalJSONLine(await observationStdout(execute, productResultPath), `product ${domain}:${ordinal}`), domain, ordinal);
	const fixture = await inspectFixture(fixtureRoot, environment, ["countershape"]);
	if (!isDeepStrictEqual(fixture, fixtureBeforeBuild)) fail("FIXTURE_DRIFT", `${domain}:${ordinal}`);
	const initialEvidence = await collectEvidence(evidenceRoot, domain, ordinal);
	const terminalEvidence = await collectEvidence(evidenceRoot, domain, ordinal);
	if (!isDeepStrictEqual(initialEvidence, terminalEvidence) || await executableDigest(binary) !== binaryDigest) fail("TERMINAL_DRIFT", `${domain}:${ordinal}`);
	const processes = Object.freeze({ prepare, compile, execute });
	const observedWall = prepare.wall_time_ms + compile.wall_time_ms + execute.wall_time_ms;
	const observedPeak = Math.max(prepare.peak_rss_bytes, compile.peak_rss_bytes, execute.peak_rss_bytes);
	const runWallBudget = specification.value.receipt_contract.study_subject_process_wall_time_budget_ms_per_domain / specification.value.receipt_contract.study_run_count_per_domain;
	if (!Number.isSafeInteger(runWallBudget) || observedWall > runWallBudget || observedPeak > specification.value.receipt_contract.study_subject_process_peak_rss_budget_bytes_per_domain) {
		fail("RUN_BUDGET", `${domain}:${ordinal}`);
	}
	const deterministic = Object.freeze({ ...initialEvidence.deterministic, reference_binary_sha256: binaryDigest });
	const invocationDigest = sha256(canonicalJSONLine(execution));
	return Object.freeze({
		ordinal,
		trial_count: phaseBudgets.reduce((sum, item) => sum + item.per_run_trials, 0),
		trial_budget: specification.value.receipt_contract.study_trial_budget_per_domain / specification.value.receipt_contract.study_run_count_per_domain,
		observed_subject_process_wall_time_ms: observedWall,
		budget_subject_process_wall_time_ms: runWallBudget,
		observed_subject_process_peak_rss_bytes: observedPeak,
		budget_subject_process_peak_rss_bytes: specification.value.receipt_contract.study_subject_process_peak_rss_budget_bytes_per_domain,
		observation_authority: observationAuthority,
		fixture,
		driver_authority: Object.freeze({ path: driverByDomain[domain], sha256: driverDigest }),
		execution,
		processes,
		phases: phaseRowsForRun(),
		product_result: productObject,
		evidence_root: relative(repositoryRoot, evidenceRoot).split(sep).join("/"),
		evidence_manifest: initialEvidence.manifest,
		evidence_manifest_sha256: initialEvidence.manifest_sha256,
		deterministic_artifacts: deterministic,
		...initialEvidence.fresh,
		fixture_invocation_sha256: invocationDigest,
	});
}

export function validateGlobalStudies(studies) {
	if (studies.length < 2) return;
	const fixtureCommits = new Set(studies.map((study) => study.fixture_commit));
	const fixtureTrees = new Set(studies.map((study) => study.fixture_authority.tree));
	const gitDirectories = new Set(studies.flatMap((study) => study.fixture_authority.repository_git_directories));
	const repositoryCount = studies.reduce((sum, study) => sum + study.fixture_authority.repository_count, 0);
	if (fixtureCommits.size !== studies.length || fixtureTrees.size !== studies.length || gitDirectories.size !== repositoryCount) fail("CROSS_DOMAIN_FIXTURE", "commit, tree, or git-directory alias");
	const deterministic = new Set(studies.flatMap((study) => Object.values(study.deterministic_artifacts)));
	const fresh = new Set();
	for (const study of studies) {
		for (const run of study.runs) {
			for (const field of [...Object.keys(freshArtifactPaths), "fixture_invocation_sha256"]) {
				const digest = run[field];
				if (!validDigest(digest) || fresh.has(digest) || deterministic.has(digest)) fail("CROSS_DOMAIN_FRESHNESS", `${study.id}:${run.ordinal}:${field}`);
				fresh.add(digest);
			}
		}
	}
}

async function observationStdout(observation, destination) {
	// timedSpawn retains only the bounded digest/length in the public observation. The exact subject stdout is
	// re-executed nowhere; production timedSpawn stores it transiently through this private symbol-like map.
	const bytes = stdoutStore.get(observation);
	if (bytes === undefined) fail("STDOUT_AUTHORITY", observation.name);
	await requireAbsent(destination);
	await writeFile(destination, bytes, { flag: "wx", mode: 0o600 });
	const reopened = await readRegularNoFollow(destination, maxOutputBytes);
	if (!reopened.bytes.equals(bytes) || reopened.bytes.length !== observation.stdout_bytes || sha256(reopened.bytes) !== observation.stdout_sha256) fail("STDOUT_DRIFT", observation.name);
	return reopened.bytes;
}

function aggregateStudy(domain, runs, contract) {
	const deterministic = runs[0].deterministic_artifacts;
	if (runs.some((run) => !isDeepStrictEqual(run.deterministic_artifacts, deterministic))) fail("DETERMINISTIC_DRIFT", domain);
	const fixtureCommits = new Set(runs.map((run) => run.fixture.head_commit));
	const fixtureTrees = new Set(runs.map((run) => run.fixture.tree));
	const fixtureGitDirectories = new Set(runs.map((run) => run.fixture.git_directory));
	if (fixtureCommits.size !== 1 || fixtureTrees.size !== 1 || fixtureGitDirectories.size !== runs.length) fail("FIXTURE_DRIFT", domain);
	const fresh = new Set();
	for (const run of runs) {
		for (const field of [...Object.keys(freshArtifactPaths), "fixture_invocation_sha256"]) {
			const digest = run[field];
			if (!validDigest(digest) || fresh.has(digest) || Object.values(deterministic).includes(digest)) fail("FRESHNESS", `${domain}:${run.ordinal}:${field}`);
			fresh.add(digest);
		}
	}
	const phases = phaseBudgets.map((phase) => Object.freeze({
		id: phase.id,
		trial_count: phase.per_run_trials * runs.length,
		trial_budget: phase.per_run_trials * runs.length,
	}));
	const wall = runs.reduce((sum, run) => sum + run.observed_subject_process_wall_time_ms, 0);
	const peak = Math.max(...runs.map((run) => run.observed_subject_process_peak_rss_bytes));
	if (wall > contract.study_subject_process_wall_time_budget_ms_per_domain || peak > contract.study_subject_process_peak_rss_budget_bytes_per_domain) fail("DOMAIN_BUDGET", domain);
	return Object.freeze({
		id: domain,
		fixture_commit: runs[0].fixture.head_commit,
		fixture_authority: Object.freeze({
			head_commit: runs[0].fixture.head_commit,
			tree: runs[0].fixture.tree,
			clean_status_bytes: 0,
			clean_status_sha256: sha256(Buffer.alloc(0)),
			repository_count: runs.length,
			fresh_repository_per_run: true,
			repository_git_directories: Object.freeze(runs.map((run) => run.fixture.git_directory)),
		}),
		logical_product_command: Object.freeze(["countershape", "study", domain, "--json"]),
		run_count: runs.length,
		trial_count: runs.reduce((sum, run) => sum + run.trial_count, 0),
		budget_trial_count: contract.study_trial_budget_per_domain,
		observed_subject_process_wall_time_ms: wall,
		budget_subject_process_wall_time_ms: contract.study_subject_process_wall_time_budget_ms_per_domain,
		observed_subject_process_peak_rss_bytes: peak,
		budget_subject_process_peak_rss_bytes: contract.study_subject_process_peak_rss_budget_bytes_per_domain,
		observation_authority: observationAuthority,
		phases: Object.freeze(phases),
		deterministic_artifacts: deterministic,
		runs: Object.freeze(runs),
	});
}

function environmentEvidence(specification) {
	const versions = Object.fromEntries(specification.value.runtime_authority.admitted_tools.map((tool) => [tool.name, tool.version]));
	return Object.freeze({
		platform: process.platform,
		kernel_release: release(),
		architecture: arch(),
		git_version: versions.git,
		go_version: versions.go,
		node_version: versions.node,
		cpu_model: cpus()[0]?.model ?? "unknown CPU",
		logical_cpu_count: cpus().length,
		memory_bytes: totalmem(),
		runtime_authority_sha256: specification.runtime_sha256,
	});
}

export async function protocolCheck() {
	const specification = await loadSpecification();
	if (process.platform !== "darwin" || process.arch !== "arm64" || Number(process.versions.node.split(".")[0]) !== 25 ||
		sha256(canonicalJSONLine(protocolAuthority)) !== specification.value.receipt_contract.study_harness.protocol_sha256) {
		fail("PROTOCOL", "runtime or protocol digest");
	}
	if (await executableDigest(timePath) !== specification.value.runtime_authority.admitted_tools.find((tool) => tool.name === "time").sha256) fail("PROTOCOL", "observer bytes");
	process.stdout.write(`U7_STUDY_HARNESS_PROTOCOL_OK protocol=${harnessProtocol} digest=${specification.value.receipt_contract.study_harness.protocol_sha256}\n`);
}

export async function runPhase(phase) {
	if (!Object.hasOwn(domainsByPhase, phase)) throw new HarnessUsageError("--phase U7B|U7C|U7D or --protocol-check");
	const specification = await loadSpecification();
	if (process.platform !== "darwin" || process.arch !== "arm64" || Number(process.versions.node.split(".")[0]) !== 25) fail("RUNTIME", `${process.platform}/${process.arch}/${process.version}`);
	const expectedTmp = resolve(repositoryRoot, finalRootByPhase[phase], "tmp");
	if (process.env.TMPDIR !== expectedTmp) fail("TMPDIR", String(process.env.TMPDIR));
	await requirePrivateDirectory(expectedTmp);
	const studiesRoot = stablePath(expectedTmp, "studies");
	await createPrivateDirectory(studiesRoot);
	const studies = [];
	for (const domain of domainsByPhase[phase]) {
		const runs = [];
		for (let ordinal = 1; ordinal <= specification.value.receipt_contract.study_run_count_per_domain; ordinal += 1) {
			runs.push(await runOne({ phase, domain, ordinal, studiesRoot, specification }));
		}
		studies.push(aggregateStudy(domain, runs, specification.value.receipt_contract));
	}
	validateGlobalStudies(studies);
	const harnessBytes = await readRegularNoFollow(modulePath, 4 * 1024 * 1024);
	const evidence = Object.freeze({
		schema_version: specification.value.receipt_contract.study_evidence_schema,
		phase,
		harness_authority: Object.freeze({
			protocol: harnessProtocol,
			path: harnessRelativePath,
			sha256: sha256(harnessBytes.bytes),
			protocol_sha256: specification.value.receipt_contract.study_harness.protocol_sha256,
			observation_authority: observationAuthority,
			semantic_ceiling: semanticCeiling,
		}),
		environment: environmentEvidence(specification),
		studies: Object.freeze(studies),
		milestone_verdict: phaseVerdicts[phase],
		honest_fallback: specification.value.receipt_contract.honest_fallback,
		unreceipted: Object.freeze([...specification.value.receipt_contract.unreceipted]),
	});
	const terminalSpecification = await loadSpecification();
	if (!terminalSpecification.bytes.equals(specification.bytes) || terminalSpecification.runtime_sha256 !== specification.runtime_sha256 ||
		!isDeepStrictEqual(terminalSpecification.harness, specification.harness)) fail("SPECIFICATION_DRIFT", phase);
	process.stdout.write(canonicalJSONLine(evidence));
}

function parseArguments(argv) {
	if (isDeepStrictEqual(argv, ["--protocol-check"])) return Object.freeze({ mode: "protocol" });
	if (argv.length === 2 && argv[0] === "--phase" && Object.hasOwn(domainsByPhase, argv[1])) return Object.freeze({ mode: "phase", phase: argv[1] });
	throw new HarnessUsageError("check-u7-study-harness.mjs --protocol-check | --phase U7B|U7C|U7D");
}

async function main() {
	process.umask(0o077);
	const request = parseArguments(process.argv.slice(2));
	if (request.mode === "protocol") await protocolCheck();
	else await runPhase(request.phase);
}

async function dispatch() {
	if (process.argv[1] === undefined) return;
	let requested;
	try { requested = await realpath(resolve(process.argv[1])); }
	catch { return; }
	if (requested !== modulePath) return;
	if (resolve(process.argv[1]) !== modulePath) fail("NONCANONICAL_ENTRY", process.argv[1]);
	await main();
}

dispatch().catch((error) => {
	process.stdout.write("");
	process.stderr.write(`${error?.message ?? error}\n`);
	process.exitCode = error instanceof HarnessUsageError ? 2 : 1;
});
