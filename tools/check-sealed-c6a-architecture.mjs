#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import {
	chmod,
	lstat,
	mkdir,
	mkdtemp,
	readFile,
	readdir,
	realpath,
	rm,
	symlink,
	writeFile,
} from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { TextDecoder } from "node:util";

const runnerPath = fileURLToPath(import.meta.url);
const repositoryRoot = resolve(dirname(runnerPath), "..");
const fatalUTF8 = new TextDecoder("utf-8", { fatal: true });

export const sealedC6AIdentity = Object.freeze({
	commit: "3e9643f657248e0d5ff2c4bf0880218efbaeecd8",
	manifest_bytes: 52_779,
	manifest_sha256: "a6c028ab3302ba215d16c90f3a27cb0583e631584382f7a70bfaed59de8cfe52",
	note_ref: "refs/notes/didrun",
	note_ref_commit: "ba1f89377012e444d30b0520e297241eabacfb48",
	regular_0644: 564,
	regular_0755: 3,
	total_blob_bytes: 12_773_420,
	tree: "226a1e23da7eded933b3faabd4e8040576b2a2e0",
});

export const sealedC6AArchitectureModes = Object.freeze([
	Object.freeze({
		argument: "--c6",
		child_arguments: Object.freeze(["--c6"]),
		child_path: "tools/check-p07b-c-architecture.mjs",
		marker: "P07B-C C6 cumulative architecture boundary OK",
		root_authority: "SEALED_C6A",
	}),
	Object.freeze({
		argument: "--c6-selftest",
		child_arguments: Object.freeze(["--c6"]),
		child_path: "tools/check-p07b-c-architecture-selftest.mjs",
		marker: "P07B-C C6 cumulative architecture defensive self-test OK (11 metadata cases; 22 Go JSON lifecycle cases)",
		root_authority: "SEALED_C6A",
	}),
]);

const productionLimits = Object.freeze({
	max_blob_bytes: 32 * 1024 * 1024,
	max_entries: 1_024,
	max_manifest_bytes: 2 * 1024 * 1024,
	max_path_bytes: 4_096,
	max_total_blob_bytes: 256 * 1024 * 1024,
});
const childTimeoutMS = 18 * 60 * 1_000;
const gitOutputLimit = 64 * 1024 * 1024;
const allowedModes = new Map([["100644", 0o644], ["100755", 0o755]]);

export class SealedC6AArchitectureError extends Error {
	constructor(code, detail) {
		super(`${code}: ${detail}`);
		this.code = code;
	}
}

function fail(code, detail) {
	throw new SealedC6AArchitectureError(code, detail);
}

function sha256(bytes) {
	return createHash("sha256").update(bytes).digest("hex");
}

export function gitBlobOID(bytes) {
	if (!Buffer.isBuffer(bytes)) fail("SEALED_C6A_BLOB_BYTES", typeof bytes);
	return createHash("sha1").update(`blob ${bytes.length}\0`).update(bytes).digest("hex");
}

function decodeRecord(bytes, index) {
	try {
		return fatalUTF8.decode(bytes);
	} catch (error) {
		fail("SEALED_C6A_MANIFEST_UTF8", `${index}:${error.message}`);
	}
}

function safePath(path, canonicalPaths) {
	if (path.length === 0 || path.startsWith("/") || path.endsWith("/") || path.includes("\\") ||
		/[\u0000-\u001f\u007f]/u.test(path) || path.normalize("NFC") !== path) {
		fail("SEALED_C6A_MANIFEST_PATH", JSON.stringify(path));
	}
	const components = path.split("/");
	if (components.some((component) => component.length === 0 || component === "." || component === ".." ||
		component.toLowerCase() === ".git")) {
		fail("SEALED_C6A_MANIFEST_PATH", JSON.stringify(path));
	}
	const canonical = path.toLowerCase();
	if (canonicalPaths.has(canonical)) fail("SEALED_C6A_MANIFEST_PATH_COLLISION", path);
	canonicalPaths.add(canonical);
	return components;
}

export function parseTreeManifest(bytes, limits = productionLimits) {
	if (!Buffer.isBuffer(bytes) || !limits || typeof limits !== "object" ||
		!Number.isSafeInteger(limits.max_manifest_bytes) || bytes.length === 0 ||
		bytes.length > limits.max_manifest_bytes || bytes.at(-1) !== 0) {
		fail("SEALED_C6A_MANIFEST_FRAMING", Buffer.isBuffer(bytes) ? String(bytes.length) : typeof bytes);
	}
	const entries = [];
	const paths = new Set();
	const canonicalPaths = new Set();
	let start = 0;
	let previousPathBytes;
	let totalPathBytes = 0;
	for (let cursor = 0; cursor < bytes.length; cursor += 1) {
		if (bytes[cursor] !== 0) continue;
		if (cursor === start) fail("SEALED_C6A_MANIFEST_EMPTY_RECORD", String(entries.length));
		if (entries.length >= limits.max_entries) fail("SEALED_C6A_MANIFEST_ENTRY_LIMIT", String(entries.length + 1));
		const recordBytes = bytes.subarray(start, cursor);
		const record = decodeRecord(recordBytes, entries.length);
		const match = /^(\d{6}) ([a-z]+) ([0-9a-f]{40})\t([\s\S]+)$/u.exec(record);
		if (!match) fail("SEALED_C6A_MANIFEST_RECORD", `${entries.length}:${JSON.stringify(record.slice(0, 160))}`);
		const [, mode, type, oid, path] = match;
		if (!allowedModes.has(mode) || type !== "blob") fail("SEALED_C6A_MANIFEST_OBJECT", `${mode}:${type}:${path}`);
		const pathBytes = Buffer.from(path, "utf8");
		totalPathBytes += pathBytes.length;
		if (pathBytes.length === 0 || pathBytes.length > limits.max_path_bytes || totalPathBytes > limits.max_manifest_bytes) {
			fail("SEALED_C6A_MANIFEST_PATH_LIMIT", path);
		}
		if (previousPathBytes !== undefined && Buffer.compare(previousPathBytes, pathBytes) >= 0) {
			fail("SEALED_C6A_MANIFEST_ORDER", path);
		}
		previousPathBytes = pathBytes;
		const components = safePath(path, canonicalPaths);
		if (paths.has(path)) fail("SEALED_C6A_MANIFEST_DUPLICATE", path);
		paths.add(path);
		entries.push(Object.freeze({ components: Object.freeze(components), mode, oid, path }));
		start = cursor + 1;
	}
	if (start !== bytes.length || entries.length === 0) fail("SEALED_C6A_MANIFEST_FRAMING", String(start));
	for (const entry of entries) {
		for (let index = 1; index < entry.components.length; index += 1) {
			const prefix = entry.components.slice(0, index).join("/");
			if (paths.has(prefix)) fail("SEALED_C6A_MANIFEST_DF_COLLISION", `${prefix}:${entry.path}`);
		}
	}
	return Object.freeze(entries);
}

function gitEnvironment() {
	return Object.freeze({
		GIT_CONFIG_GLOBAL: "/dev/null",
		GIT_CONFIG_NOSYSTEM: "1",
		GIT_NO_LAZY_FETCH: "1",
		GIT_OPTIONAL_LOCKS: "0",
		GIT_TERMINAL_PROMPT: "0",
		HOME: "/",
		LANG: "C",
		LC_ALL: "C",
		NO_COLOR: "1",
		PATH: "/usr/bin:/bin",
		TZ: "UTC",
	});
}

function runGit(git, cwd, args, options = {}) {
	const result = spawnSync(git, ["--no-replace-objects", ...args], {
		cwd,
		encoding: null,
		env: gitEnvironment(),
		input: options.input,
		maxBuffer: options.maxBuffer ?? gitOutputLimit,
		timeout: options.timeout ?? 60_000,
	});
	if (result.error || result.signal || result.status !== 0 || (result.stderr?.length ?? 0) !== 0) {
		fail(
			"SEALED_C6A_GIT",
			`${args[0]} status=${result.status} signal=${result.signal} error=${result.error?.message ?? "none"} stderr=${JSON.stringify((result.stderr ?? Buffer.alloc(0)).toString("utf8").slice(0, 512))}`,
		);
	}
	return result.stdout ?? Buffer.alloc(0);
}

function decodeGitLine(bytes, operation) {
	let text;
	try {
		text = fatalUTF8.decode(bytes);
	} catch (error) {
		fail("SEALED_C6A_GIT_OUTPUT", `${operation}:${error.message}`);
	}
	if (!text.endsWith("\n") || text.includes("\r") || text.slice(0, -1).includes("\n")) {
		fail("SEALED_C6A_GIT_OUTPUT", operation);
	}
	return text.slice(0, -1);
}

async function requireCanonicalTool(name, value) {
	if (typeof value !== "string" || !isAbsolute(value) || value.includes("\n") || value.includes("\r")) {
		fail("SEALED_C6A_TOOL_AUTHORITY", name);
	}
	const [canonical, metadata] = await Promise.all([realpath(value), lstat(value)]);
	if (canonical !== value || metadata.isSymbolicLink() || !metadata.isFile()) {
		fail("SEALED_C6A_TOOL_AUTHORITY", `${name}:${value}`);
	}
	return canonical;
}

async function admittedTools(environment = process.env) {
	const [node, git] = await Promise.all([
		requireCanonicalTool("COUNTERSHAPE_NODE", environment.COUNTERSHAPE_NODE),
		requireCanonicalTool("COUNTERSHAPE_GIT", environment.COUNTERSHAPE_GIT),
	]);
	if (node !== await realpath(process.execPath)) fail("SEALED_C6A_NODE_AUTHORITY", `${node} != ${process.execPath}`);
	return Object.freeze({ git, node });
}

function childEnvironment(environment = process.env) {
	const required = [
		"CC", "CGO_ENABLED", "COUNTERSHAPE_CC", "COUNTERSHAPE_CXX", "COUNTERSHAPE_GIT", "COUNTERSHAPE_GO",
		"COUNTERSHAPE_NODE", "COUNTERSHAPE_SH", "CXX", "GOCACHE", "GOENV", "GOFLAGS", "GOMAXPROCS",
		"GOMODCACHE", "GOPATH", "GOPROXY", "GOSUMDB", "GOTMPDIR", "GOTOOLCHAIN", "GOVCS", "GOWORK",
		"HOME", "LANG", "LC_ALL", "NO_COLOR", "PATH", "TMPDIR", "TZ",
	];
	const output = Object.create(null);
	for (const name of required) {
		const value = environment[name];
		if (typeof value !== "string" || value.length === 0 || value.includes("\0") || value.includes("\n") || value.includes("\r")) {
			fail("SEALED_C6A_CHILD_ENVIRONMENT", name);
		}
		output[name] = value;
	}
	Object.assign(output, {
		GIT_CONFIG_GLOBAL: "/dev/null",
		GIT_CONFIG_NOSYSTEM: "1",
		GIT_NO_LAZY_FETCH: "1",
		GIT_OPTIONAL_LOCKS: "0",
		GIT_TERMINAL_PROMPT: "0",
		NODE_OPTIONS: "",
	});
	return Object.freeze(output);
}

async function canonicalObjectDirectory(git) {
	const raw = decodeGitLine(runGit(git, repositoryRoot, ["rev-parse", "--git-path", "objects"]), "git object path");
	const supplied = isAbsolute(raw) ? raw : resolve(repositoryRoot, raw);
	const canonical = await realpath(supplied);
	const metadata = await lstat(canonical);
	if (!metadata.isDirectory() || canonical.includes("\n") || canonical.includes("\r")) {
		fail("SEALED_C6A_OBJECT_DIRECTORY", canonical);
	}
	return canonical;
}

async function writeSnapshotFile(root, entry, bytes) {
	if (gitBlobOID(bytes) !== entry.oid) fail("SEALED_C6A_BLOB_OID", entry.path);
	const absolute = join(root, ...entry.components);
	const directory = dirname(absolute);
	await mkdir(directory, { recursive: true, mode: 0o755 });
	await writeFile(absolute, bytes, { flag: "wx", mode: allowedModes.get(entry.mode) });
	await chmod(absolute, allowedModes.get(entry.mode));
}

async function inspectSnapshotFiles(root, entries) {
	const observed = [];
	async function walk(directory, prefix) {
		const directoryEntries = await readdir(directory, { withFileTypes: true });
		directoryEntries.sort((left, right) => left.name.localeCompare(right.name, "en"));
		for (const entry of directoryEntries) {
			const path = prefix ? `${prefix}/${entry.name}` : entry.name;
			if (prefix === "" && entry.name === ".git") continue;
			const absolute = join(directory, entry.name);
			const metadata = await lstat(absolute);
			if (metadata.isSymbolicLink()) fail("SEALED_C6A_SNAPSHOT_SYMLINK", path);
			if (metadata.isDirectory()) await walk(absolute, path);
			else if (metadata.isFile()) observed.push(path);
			else fail("SEALED_C6A_SNAPSHOT_OBJECT", path);
		}
	}
	await walk(root, "");
	const expectedPaths = entries.map((entry) => entry.path).sort();
	observed.sort();
	if (JSON.stringify(observed) !== JSON.stringify(expectedPaths)) fail("SEALED_C6A_SNAPSHOT_ROSTER", observed.join(","));
	let total = 0;
	for (const entry of entries) {
		const absolute = join(root, ...entry.components);
		const before = await lstat(absolute);
		if (!before.isFile() || before.isSymbolicLink() || before.nlink !== 1 ||
			(before.mode & 0o777) !== allowedModes.get(entry.mode)) {
			fail("SEALED_C6A_SNAPSHOT_MODE", entry.path);
		}
		const bytes = await readFile(absolute);
		const after = await lstat(absolute);
		if (before.dev !== after.dev || before.ino !== after.ino || before.size !== after.size || before.mtimeMs !== after.mtimeMs ||
			gitBlobOID(bytes) !== entry.oid) {
			fail("SEALED_C6A_SNAPSHOT_DRIFT", entry.path);
		}
		total += bytes.length;
	}
	return total;
}

async function createMinimalGitDirectory(root, objectDirectory) {
	const gitDirectory = join(root, ".git");
	await mkdir(join(gitDirectory, "objects/info"), { recursive: true, mode: 0o700 });
	await mkdir(join(gitDirectory, "refs/notes"), { recursive: true, mode: 0o700 });
	const files = new Map([
		["HEAD", `${sealedC6AIdentity.commit}\n`],
		["config", "[core]\n\trepositoryformatversion = 0\n\tbare = false\n\tfilemode = true\n\tlogallrefupdates = false\n"],
		["objects/info/alternates", `${objectDirectory}\n`],
		[sealedC6AIdentity.note_ref, `${sealedC6AIdentity.note_ref_commit}\n`],
	]);
	for (const [path, body] of files) {
		const absolute = join(gitDirectory, ...path.split("/"));
		await writeFile(absolute, body, { flag: "wx", mode: 0o400 });
		await chmod(absolute, 0o400);
	}
	for (const directory of ["objects/info", "objects", "refs/notes", "refs", "."]) {
		await chmod(resolve(gitDirectory, directory), 0o700);
	}
	return Object.freeze({ files, gitDirectory });
}

async function inspectMinimalGitDirectory(git, root, control) {
	for (const [path, body] of control.files) {
		const absolute = join(control.gitDirectory, ...path.split("/"));
		const metadata = await lstat(absolute);
		if (!metadata.isFile() || metadata.isSymbolicLink() || metadata.nlink !== 1 || (metadata.mode & 0o777) !== 0o400 ||
			(await readFile(absolute, "utf8")) !== body) {
			fail("SEALED_C6A_GIT_CONTROL_DRIFT", path);
		}
	}
	const checks = [
		[["rev-parse", "--verify", "HEAD^{commit}"], sealedC6AIdentity.commit],
		[["rev-parse", "--verify", "HEAD^{tree}"], sealedC6AIdentity.tree],
		[["rev-parse", "--verify", `${sealedC6AIdentity.note_ref}^{commit}`], sealedC6AIdentity.note_ref_commit],
		[["rev-parse", "--show-object-format"], "sha1"],
	];
	for (const [args, expected] of checks) {
		const actual = decodeGitLine(runGit(git, root, args), args.join(" "));
		if (actual !== expected) fail("SEALED_C6A_GIT_CONTROL_AUTHORITY", `${args.join(" ")}:${actual}`);
	}
}

async function materializeSealedTree(git, root) {
	for (const [args, expected] of [
		[["rev-parse", "--verify", `${sealedC6AIdentity.commit}^{commit}`], sealedC6AIdentity.commit],
		[["rev-parse", "--verify", `${sealedC6AIdentity.commit}^{tree}`], sealedC6AIdentity.tree],
		[["rev-parse", "--verify", `${sealedC6AIdentity.note_ref_commit}^{commit}`], sealedC6AIdentity.note_ref_commit],
		[["rev-parse", "--show-object-format"], "sha1"],
	]) {
		const actual = decodeGitLine(runGit(git, repositoryRoot, args), args.join(" "));
		if (actual !== expected) fail("SEALED_C6A_SOURCE_AUTHORITY", `${args.join(" ")}:${actual}`);
	}
	const manifest = runGit(git, repositoryRoot, ["ls-tree", "-r", "-z", "--full-tree", sealedC6AIdentity.commit], {
		maxBuffer: productionLimits.max_manifest_bytes,
	});
	if (manifest.length !== sealedC6AIdentity.manifest_bytes || sha256(manifest) !== sealedC6AIdentity.manifest_sha256) {
		fail("SEALED_C6A_MANIFEST_AUTHORITY", `${manifest.length}:sha256:${sha256(manifest)}`);
	}
	const entries = parseTreeManifest(manifest);
	const modeCounts = new Map([["100644", 0], ["100755", 0]]);
	let total = 0;
	for (const entry of entries) {
		modeCounts.set(entry.mode, modeCounts.get(entry.mode) + 1);
		const bytes = runGit(git, repositoryRoot, ["cat-file", "blob", entry.oid], { maxBuffer: productionLimits.max_blob_bytes });
		if (bytes.length > productionLimits.max_blob_bytes || total + bytes.length > productionLimits.max_total_blob_bytes ||
			gitBlobOID(bytes) !== entry.oid) {
			fail("SEALED_C6A_BLOB_AUTHORITY", entry.path);
		}
		const gitOID = decodeGitLine(runGit(git, repositoryRoot, ["hash-object", "--stdin"], { input: bytes }), "hash-object");
		if (gitOID !== entry.oid) fail("SEALED_C6A_BLOB_HASH_PARITY", entry.path);
		await writeSnapshotFile(root, entry, bytes);
		total += bytes.length;
	}
	if (entries.length !== sealedC6AIdentity.regular_0644 + sealedC6AIdentity.regular_0755 ||
		modeCounts.get("100644") !== sealedC6AIdentity.regular_0644 || modeCounts.get("100755") !== sealedC6AIdentity.regular_0755 ||
		total !== sealedC6AIdentity.total_blob_bytes || await inspectSnapshotFiles(root, entries) !== total) {
		fail("SEALED_C6A_SNAPSHOT_AUTHORITY", `${entries.length}:${modeCounts.get("100644")}:${modeCounts.get("100755")}:${total}`);
	}
	return entries;
}

function modeForArgument(argument) {
	const matches = sealedC6AArchitectureModes.filter((mode) => mode.argument === argument);
	if (matches.length !== 1) fail("SEALED_C6A_ARGUMENTS", String(argument));
	return matches[0];
}

async function runSealedMode(mode) {
	const tools = await admittedTools();
	const environment = childEnvironment();
	const suppliedTemp = process.env.TMPDIR;
	if (typeof suppliedTemp !== "string" || !isAbsolute(suppliedTemp)) fail("SEALED_C6A_TMP_AUTHORITY", String(suppliedTemp));
	const canonicalTemp = await realpath(suppliedTemp);
	const root = await realpath(await mkdtemp(join(canonicalTemp, "countershape-sealed-c6a-")));
	await chmod(root, 0o700);
	let primaryFailure;
	let entries;
	let control;
	try {
		entries = await materializeSealedTree(tools.git, root);
		const objectDirectory = await canonicalObjectDirectory(tools.git);
		control = await createMinimalGitDirectory(root, objectDirectory);
		await inspectMinimalGitDirectory(tools.git, root, control);
		const child = spawnSync(tools.node, [resolve(root, mode.child_path), ...mode.child_arguments], {
			cwd: root,
			encoding: "utf8",
			env: environment,
			maxBuffer: gitOutputLimit,
			timeout: childTimeoutMS,
		});
		if (child.stdout) process.stdout.write(child.stdout);
		if (child.stderr) process.stderr.write(child.stderr);
		if (child.error || child.signal || child.status !== 0 || child.stderr !== "" || child.stdout !== `${mode.marker}\n`) {
			fail("SEALED_C6A_CHILD", `${mode.argument}:status=${child.status} signal=${child.signal} error=${child.error?.message ?? "none"}`);
		}
		await inspectSnapshotFiles(root, entries);
		await inspectMinimalGitDirectory(tools.git, root, control);
		process.stdout.write(
			`SEALED_C6A_ARCHITECTURE root_authority=${mode.root_authority} commit=${sealedC6AIdentity.commit} tree=${sealedC6AIdentity.tree} manifest_sha256:${sealedC6AIdentity.manifest_sha256} mode=${mode.argument}\n`,
		);
	} catch (error) {
		primaryFailure = error;
		throw error;
	} finally {
		try {
			await rm(root, { recursive: true, force: true });
		} catch (cleanupFailure) {
			if (primaryFailure) throw new AggregateError([primaryFailure, cleanupFailure], "sealed C6A execution and cleanup both failed");
			throw cleanupFailure;
		}
	}
}

function manifestBytes(records) {
	return Buffer.from(`${records.join("\0")}\0`, "utf8");
}

function expectCode(invoke, code) {
	try {
		invoke();
	} catch (error) {
		if (error instanceof SealedC6AArchitectureError && error.code === code) return;
		throw error;
	}
	fail("SEALED_C6A_SELFTEST_FALSE_NEGATIVE", code);
}

async function runSelfTest() {
	const literal = Buffer.from("$Format:%H$\n", "utf8");
	const executable = Buffer.from("#!/bin/sh\nexit 0\n", "utf8");
	const records = [
		`100644 blob ${gitBlobOID(literal)}\t.gitattributes`,
		`100755 blob ${gitBlobOID(executable)}\ttools/literal.sh`,
	];
	const limits = { ...productionLimits, max_entries: 8, max_manifest_bytes: 4_096, max_path_bytes: 128, max_total_blob_bytes: 4_096 };
	const valid = manifestBytes(records);
	const parsed = parseTreeManifest(valid, limits);
	if (parsed.length !== 2 || parsed[0].path !== ".gitattributes" || parsed[1].mode !== "100755" ||
		JSON.stringify(sealedC6AArchitectureModes.map((mode) => mode.argument)) !== JSON.stringify(["--c6", "--c6-selftest"]) ||
		sealedC6AArchitectureModes.some((mode) => mode.root_authority !== "SEALED_C6A") ||
		modeForArgument("--c6-selftest").child_path !== "tools/check-p07b-c-architecture-selftest.mjs") {
		fail("SEALED_C6A_SELFTEST_BASELINE", JSON.stringify(parsed));
	}
	expectCode(() => parseTreeManifest(valid.subarray(0, -1), limits), "SEALED_C6A_MANIFEST_FRAMING");
	expectCode(() => parseTreeManifest(manifestBytes([records[0], records[0]]), limits), "SEALED_C6A_MANIFEST_ORDER");
	expectCode(() => parseTreeManifest(manifestBytes([...records].reverse()), limits), "SEALED_C6A_MANIFEST_ORDER");
	expectCode(() => parseTreeManifest(manifestBytes([`120000 blob ${"0".repeat(40)}\tlink`]), limits), "SEALED_C6A_MANIFEST_OBJECT");
	expectCode(() => parseTreeManifest(manifestBytes([`100644 tree ${"0".repeat(40)}\tdirectory`]), limits), "SEALED_C6A_MANIFEST_OBJECT");
	expectCode(() => parseTreeManifest(manifestBytes([`100644 blob ${"0".repeat(40)}\t../escape`]), limits), "SEALED_C6A_MANIFEST_PATH");
	expectCode(() => parseTreeManifest(manifestBytes([`100644 blob ${"0".repeat(40)}\t.GIT/config`]), limits), "SEALED_C6A_MANIFEST_PATH");
	expectCode(() => parseTreeManifest(manifestBytes([
		`100644 blob ${"0".repeat(40)}\tA`, `100644 blob ${"1".repeat(40)}\ta`,
	]), limits), "SEALED_C6A_MANIFEST_PATH_COLLISION");
	expectCode(() => parseTreeManifest(manifestBytes([
		`100644 blob ${"0".repeat(40)}\ta`, `100644 blob ${"1".repeat(40)}\ta/b`,
	]), limits), "SEALED_C6A_MANIFEST_DF_COLLISION");
	expectCode(() => modeForArgument("--c5"), "SEALED_C6A_ARGUMENTS");

	const fixture = await realpath(await mkdtemp(join(tmpdir(), "countershape-sealed-c6a-selftest-")));
	try {
		await chmod(fixture, 0o700);
		await writeSnapshotFile(fixture, parsed[0], literal);
		await writeSnapshotFile(fixture, parsed[1], executable);
		if (await inspectSnapshotFiles(fixture, parsed) !== literal.length + executable.length ||
			(await readFile(join(fixture, ".gitattributes"))).equals(literal) !== true ||
			(await readFile(join(fixture, "tools/literal.sh"))).equals(executable) !== true) {
			fail("SEALED_C6A_SELFTEST_LITERAL_BLOBS", fixture);
		}
		await rm(join(fixture, "tools/literal.sh"));
		await symlink("../.gitattributes", join(fixture, "tools/literal.sh"));
		try {
			await inspectSnapshotFiles(fixture, parsed);
			fail("SEALED_C6A_SELFTEST_FALSE_NEGATIVE", "snapshot symlink");
		} catch (error) {
			if (!(error instanceof SealedC6AArchitectureError) || error.code !== "SEALED_C6A_SNAPSHOT_SYMLINK") throw error;
		}
	} finally {
		await rm(fixture, { recursive: true, force: true });
	}
	process.stdout.write("sealed C6A architecture runner self-test passed: raw blobs, path/mode authority, literal-filter immunity, and two-mode routing\n");
}

async function classifyEntry(entry) {
	if (entry === undefined) return Object.freeze({ kind: "IMPORTED" });
	const source = await realpath(runnerPath);
	let supplied;
	try {
		supplied = resolve(entry);
		if (await realpath(supplied) !== source) return Object.freeze({ kind: "IMPORTED" });
	} catch {
		return Object.freeze({ kind: "IMPORTED" });
	}
	if (supplied === source) return Object.freeze({ kind: "CANONICAL" });
	return Object.freeze({ kind: "NONCANONICAL", source, supplied });
}

const entry = await classifyEntry(process.argv[1]);
if (entry.kind === "NONCANONICAL") fail("SEALED_C6A_NONCANONICAL_ENTRY", `${entry.supplied} != ${entry.source}`);
if (entry.kind === "CANONICAL") {
	if (process.argv.length !== 3) fail("SEALED_C6A_ARGUMENTS", "one exact mode argument required");
	if (process.argv[2] === "--self-test") await runSelfTest();
	else await runSealedMode(modeForArgument(process.argv[2]));
}
