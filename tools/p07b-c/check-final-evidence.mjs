#!/usr/bin/env node

import { createHash, randomBytes } from "node:crypto";
import { spawnSync } from "node:child_process";
import { constants } from "node:fs";
import {
	chmod, lstat, mkdir, mkdtemp, open, readFile, readdir, realpath, rename, rm, symlink, unlink, writeFile,
} from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

import {
	goJSONProfileDescriptor,
} from "../check-p07b-c-architecture.mjs";
import {
	admitTools,
	buildChildEnvironment,
	cleanupVerificationResources,
	createPrivateRoots,
} from "../verify-runtime-authority.mjs";
import {
	assertProfileRunsMatchDescriptors,
	buildEvidenceDocument,
	buildProfileRun,
	buildSourceClosure,
	buildSummary,
	c5vAuthority,
	c5vClaims,
	c6AbsentSourceInputPaths,
	c6ArtifactPaths,
	c6ExecutionEnvironmentContract,
	c6GoListArguments,
	c6ProfileDescriptors,
	c6SelectedProfiles,
	c6SourceClosurePackages,
	c6Widths,
	canonicalCompact,
	collectSourceInputs,
	computeAdmissionSha256,
	c6SourceInputPaths,
	describeDirectory,
	expectedArtifactBytes,
	readTrackedEvidence,
	renderEvidence,
	resolveNoFollow,
	runPureSelfTest,
	validateRenderBytes,
	validateGoJSONTranscript,
} from "./final-evidence-lib.mjs";

const checkerPath = fileURLToPath(import.meta.url);
const repositoryRoot = resolve(dirname(checkerPath), "../..");
const gitExecutable = "/usr/bin/git";
const fixedToolPaths = Object.freeze({
	cc: "/usr/bin/clang",
	cxx: "/usr/bin/clang++",
	git: "/usr/bin/git",
	go: "/opt/homebrew/bin/go",
	node: "/opt/homebrew/bin/node",
	sh: "/bin/sh",
});
const toolEnvironmentVariables = Object.freeze({
	COUNTERSHAPE_CC: fixedToolPaths.cc,
	COUNTERSHAPE_CXX: fixedToolPaths.cxx,
	COUNTERSHAPE_GIT: fixedToolPaths.git,
	COUNTERSHAPE_GO: fixedToolPaths.go,
	COUNTERSHAPE_NODE: fixedToolPaths.node,
	COUNTERSHAPE_SH: fixedToolPaths.sh,
});
const sourceFileFields = Object.freeze([
	"GoFiles", "CgoFiles", "CFiles", "CXXFiles", "MFiles", "HFiles", "FFiles", "SFiles", "SwigFiles",
	"SwigCXXFiles", "SysoFiles", "TestGoFiles", "XTestGoFiles", "EmbedFiles", "TestEmbedFiles", "XTestEmbedFiles",
]);
const explicitSourceInputs = Object.freeze([
	"docs/status/DIDRUN_BUGS.md",
	"go.mod",
	"internal/emit/node/parity/runner.mjs",
	"spec/examples/v1/choicepoint.valid.json",
	"spec/vectors/v1/contract-parity.jsonl",
	"tools/check-p07b-c-architecture.mjs",
	"tools/p07b-c/check-final-evidence.mjs",
	"tools/p07b-c/final-evidence-lib.mjs",
	"tools/p07b-c/profile-authority.mjs",
	"tools/p07b-c/source-closure.mjs",
	"tools/verify-runtime-authority.mjs",
]);
const c5vTestkitPaths = Object.freeze([
	"testkit/contractexec/cli/fixture_darwin.go",
	"testkit/contractexec/cli/runner_darwin_test.go",
	"testkit/contractexec/http/fixture_darwin.go",
	"testkit/contractexec/http/runner_darwin_test.go",
]);
const artifactLockName = ".p07b-c.lock";
const artifactJournalName = ".p07b-c.transaction.json";
const artifactTransactionSchema = "countershape/p07b-c-artifact-transaction/v3";
const artifactLockSchema = "countershape/p07b-c-artifact-lock/v1";
const failedAttemptPath = ".didrun-history/p07b-c-c5v-final-failed-20260731-operator-terminated/session.log";
const expectedArchiveCore = Object.freeze({
	claims: "030b105794ff9c087de3446585dff490c70b3722a554b5f6bb288168448df87e",
	gitignore: "cdbcae15105d6b781e620813c79c7e868740d4e9cc53ce6f5fcbbc12387adf4b",
	seals: "d8eacf6539d7cb353d4db7340e30768bb614b77f0026a218bc3d4dc4652f61a2",
	session: "bc3c8b58d2d7b97491f773e5e7fa6d73de8e7334e6e1c6fa8729c019ac3b1aad",
});
const expectedArchiveChainAnchors = Object.freeze({
	first: "b20527ca0eebd1c44c30fdb4bc5948500836c196808aaf5680487451ced55221",
	last: "9e5237aff5f93310fb53f27e357330ae65c7d5a265bca3039fe7d9b82d6de421",
});
const expectedC5VCommandTails = Object.freeze([
	Object.freeze(["tools/check-p07b-c-plan.mjs", "--check-candidate-phase", "C5V"]),
	Object.freeze(["tools/check-p07b-c-unit-scope.mjs", "--candidate-phase", "C5V"]),
	Object.freeze(["tools/check-p07b-c-plan.mjs", "--self-test"]),
	Object.freeze(["tools/check-p07b-c-plan.mjs", "--verify-c5v-sealed-c5-note"]),
	Object.freeze(["tools/check-p07b-c-unit-scope.mjs", "--self-test"]),
	Object.freeze(["tools/verify-current-selftest.mjs"]),
	Object.freeze(["tools/verify-current.mjs"]),
	Object.freeze(["tools/verify-current.mjs"]),
	Object.freeze(["tools/verify-current.mjs"]),
	Object.freeze(["tools/check-p07b-c-unit-scope.mjs", "--unit", "C5V", "--source-final-gate"]),
	Object.freeze(["tools/check-p07b-c-unit-scope.mjs", "--unit", "C5V", "--credential-scan"]),
	Object.freeze(["tools/check-p07b-c-plan.mjs", "--verify-c5v-preseal-ledger"]),
]);
const c5vPrivateRoot = join(repositoryRoot, ".countershape/p07bc-c5v-final");
const expectedC5VCommandPrefix = Object.freeze([
	"/usr/bin/env", "-i",
	`HOME=${join(c5vPrivateRoot, "home")}`,
	`PWD=${repositoryRoot}`,
	`TMPDIR=${join(c5vPrivateRoot, "tmp")}`,
	`GOTMPDIR=${join(c5vPrivateRoot, "gotmp")}`,
	`GOCACHE=${join(c5vPrivateRoot, "gocache")}`,
	`GOPATH=${join(c5vPrivateRoot, "gopath")}`,
	`GOMODCACHE=${join(c5vPrivateRoot, "gomodcache")}`,
	"GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off",
	"GOFLAGS=-mod=readonly -buildvcs=false -p=1", "CGO_ENABLED=1", "GOMAXPROCS=2",
	"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "NODE_OPTIONS=", "NODE_PATH=",
	"PATH=/opt/homebrew/bin:/usr/bin:/bin", "SHELL=/bin/sh",
	"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_LAZY_FETCH=1",
	"GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0",
	"COUNTERSHAPE_GO=/opt/homebrew/bin/go", "COUNTERSHAPE_NODE=/opt/homebrew/bin/node",
	"COUNTERSHAPE_GIT=/usr/bin/git", "COUNTERSHAPE_SH=/bin/sh", "COUNTERSHAPE_CC=/usr/bin/clang",
	"COUNTERSHAPE_CXX=/usr/bin/clang++", "CC=/usr/bin/clang", "CXX=/usr/bin/clang++",
	"/opt/homebrew/bin/node",
]);
const sessionEntryKeys = Object.freeze(["entry_hash", "event", "index", "prev_hash"]);
const sessionEventKeys = Object.freeze([
	"argv", "coverage", "cwd", "ended_at", "env_fingerprint", "exit_code", "observed_via", "started_at",
	"stderr_blob", "stdout_blob", "submodule_dirty", "transcript_blob", "tree_after", "tree_before",
]);
const archiveClaimKeys = Object.freeze(["ctype", "declared_at_index", "event_indices", "label", "pathspecs"]);
const archiveSealKeys = Object.freeze(["claims_watermark", "commit", "tree"]);
const didrunBugIDs = Object.freeze(["S6-01", "S6-02", "S6-06", "S6-07", "S6-10", "S6-13"]);

class FinalEvidenceError extends Error {
	constructor(code, detail) {
		super(`${code}: ${detail}`);
		this.code = code;
	}
}

function fail(code, detail) {
	throw new FinalEvidenceError(code, detail);
}

function digest(bytes) {
	return createHash("sha256").update(bytes).digest("hex");
}

function exact(left, right) {
	return JSON.stringify(left) === JSON.stringify(right);
}

function sorted(values) {
	return [...values].sort((left, right) => Buffer.compare(Buffer.from(left), Buffer.from(right)));
}

function exactKeys(value, keys, code) {
	if (!value || typeof value !== "object" || Array.isArray(value) || !exact(Object.keys(value), keys)) {
		fail(code, JSON.stringify(value));
	}
	return value;
}

function pythonJSONString(value) {
	return JSON.stringify(value).replace(/[\u007f-\uffff]/gu, (character) => {
		const units = [];
		for (let index = 0; index < character.length; index += 1) {
			units.push(`\\u${character.charCodeAt(index).toString(16).padStart(4, "0")}`);
		}
		return units.join("");
	});
}

function pythonCanonicalJSON(value) {
	if (value === null || typeof value === "boolean" || typeof value === "string") {
		return Buffer.from(typeof value === "string" ? pythonJSONString(value) : JSON.stringify(value), "ascii");
	}
	if (typeof value === "number") {
		if (!Number.isFinite(value)) fail("P07B_C6_ARCHIVE_CANONICAL_NUMBER", String(value));
		const encoded = JSON.stringify(value);
		return Buffer.from(encoded.includes("e+") ? encoded.replace("e+", "e") : encoded, "ascii");
	}
	if (Array.isArray(value)) {
		return Buffer.concat([
			Buffer.from("["),
			...value.flatMap((entry, index) => [index === 0 ? Buffer.alloc(0) : Buffer.from(","), pythonCanonicalJSON(entry)]),
			Buffer.from("]"),
		]);
	}
	if (value && typeof value === "object") {
		const keys = Object.keys(value).sort();
		return Buffer.concat([
			Buffer.from("{"),
			...keys.flatMap((key, index) => [
				index === 0 ? Buffer.alloc(0) : Buffer.from(","),
				Buffer.from(`${pythonJSONString(key)}:`, "ascii"),
				pythonCanonicalJSON(value[key]),
			]),
			Buffer.from("}"),
		]);
	}
	fail("P07B_C6_ARCHIVE_CANONICAL_TYPE", typeof value);
}

function didrunEntryHash(index, previous, event) {
	return digest(Buffer.concat([
		Buffer.from(previous, "ascii"),
		pythonCanonicalJSON({ event, index, prev: previous }),
	]));
}

function confinedRelativePath(absolute) {
	const candidate = relative(repositoryRoot, absolute);
	if (candidate.length === 0 || isAbsolute(candidate) || candidate === ".." || candidate.startsWith(`..${sep}`)) return null;
	return candidate.split(sep).join("/");
}

function isWithin(root, absolute) {
	const candidate = relative(root, absolute);
	return candidate === "" || (!isAbsolute(candidate) && candidate !== ".." && !candidate.startsWith(`..${sep}`));
}

function jsonLines(bytes, label, maximum = 20_000) {
	if (!Buffer.isBuffer(bytes) || bytes.length === 0 || bytes.at(-1) !== 0x0a || bytes.includes(0x0d)) {
		fail("P07B_C6_JSONL_ENVELOPE", label);
	}
	const lines = bytes.subarray(0, -1).toString("utf8").split("\n");
	if (lines.length > maximum || lines.some((line) => line.length === 0 || Buffer.byteLength(line) > 8 * 1024 * 1024)) {
		fail("P07B_C6_JSONL_BOUND", label);
	}
	return lines.map((line, index) => {
		try { return JSON.parse(line); }
		catch (error) { fail("P07B_C6_JSONL_PARSE", `${label}:${index + 1}:${error.message}`); }
	});
}

async function readLocalRegular(relativePath, maximum = 128 * 1024 * 1024, minimum = 1) {
	const absolute = await resolveNoFollow(repositoryRoot, relativePath, { kind: "file" });
	const before = await lstat(absolute);
	if (!before.isFile() || before.isSymbolicLink() || before.nlink !== 1 || (before.mode & 0o022) !== 0 ||
		before.size < minimum || before.size > maximum) fail("P07B_C6_LOCAL_EVIDENCE_MODE", relativePath);
	const bytes = await readFile(absolute);
	const after = await lstat(absolute);
	if (!after.isFile() || after.isSymbolicLink() || before.dev !== after.dev || before.ino !== after.ino ||
		before.size !== after.size || before.mtimeMs !== after.mtimeMs) fail("P07B_C6_LOCAL_EVIDENCE_CHANGED", relativePath);
	return bytes;
}

async function readCurrentDescriptor(relativePath, maximum = 128 * 1024 * 1024, minimum = 1) {
	const absolute = await resolveNoFollow(repositoryRoot, relativePath, { kind: "file" });
	const before = await lstat(absolute);
	if (!before.isFile() || before.isSymbolicLink() || before.nlink !== 1 || (before.mode & 0o022) !== 0 ||
		before.size < minimum || before.size > maximum) fail("P07B_C6_LOCAL_EVIDENCE_MODE", relativePath);
	const bytes = await readFile(absolute);
	const after = await lstat(absolute);
	if (!after.isFile() || after.isSymbolicLink() || before.dev !== after.dev || before.ino !== after.ino ||
		before.size !== after.size || before.mtimeMs !== after.mtimeMs) fail("P07B_C6_LOCAL_EVIDENCE_CHANGED", relativePath);
	return Object.freeze({
		bytes: bytes.length,
		mode: `100${(after.mode & 0o777).toString(8).padStart(3, "0")}`,
		path: relativePath,
		sha256: digest(bytes),
		value: bytes,
	});
}

async function createExecutionRuntime() {
	for (const [variable, fixed] of Object.entries(toolEnvironmentVariables)) {
		if (Object.hasOwn(process.env, variable) && process.env[variable] !== fixed) {
			fail("P07B_C6_INHERITED_TOOL_AUTHORITY", `${variable}=${process.env[variable]}`);
		}
	}
	const admitted = await admitTools(toolEnvironmentVariables);
	for (const [name, supplied] of Object.entries(fixedToolPaths)) {
		const canonical = await realpath(supplied);
		if (admitted[name]?.path !== canonical) fail("P07B_C6_FIXED_TOOL_AUTHORITY", `${name}:${admitted[name]?.path}`);
	}
	const roots = await createPrivateRoots(repositoryRoot, admitted);
	const environment = buildChildEnvironment(admitted, roots);
	let closed = false;
	const revalidate = async () => {
		if (closed) fail("P07B_C6_RUNTIME_CLOSED", "tool revalidation");
		await admitTools(toolEnvironmentVariables, admitted);
	};
	const run = async (tool, args, code, { maximum = 64 * 1024 * 1024, timeout = 30 * 60 * 1000 } = {}) => {
		await revalidate();
		const result = spawnSync(admitted[tool].path, args, {
			cwd: repositoryRoot,
			encoding: null,
			env: environment,
			maxBuffer: maximum,
			timeout,
		});
		let revalidationFailure;
		try { await revalidate(); }
		catch (error) { revalidationFailure = error; }
		if (result.error || result.signal || result.status !== 0 || !Buffer.isBuffer(result.stdout) ||
			!Buffer.isBuffer(result.stderr) || result.stderr.length !== 0 || result.stdout.length > maximum) {
			const failure = new FinalEvidenceError(code, `${tool} status=${result.status ?? result.signal ?? "spawn"}`);
			if (revalidationFailure) throw new AggregateError([failure, revalidationFailure], `${code}: execution and revalidation failed`);
			throw failure;
		}
		if (revalidationFailure) throw revalidationFailure;
		return result.stdout;
	};
	const close = async () => {
		if (closed) return;
		closed = true;
		await cleanupVerificationResources(null, roots);
	};
	return Object.freeze({ admitted, close, environment, revalidate, roots, run });
}

function runGit(args, { input, maximum = 16 * 1024 * 1024 } = {}) {
	if (!isAbsolute(gitExecutable) || gitExecutable !== "/usr/bin/git") fail("P07B_C6_GIT_AUTHORITY", gitExecutable);
	const result = spawnSync(gitExecutable, ["--no-replace-objects", ...args], {
		cwd: repositoryRoot,
		encoding: null,
		env: {
			GIT_CONFIG_GLOBAL: "/dev/null",
			GIT_CONFIG_NOSYSTEM: "1",
			GIT_NO_LAZY_FETCH: "1",
			GIT_NO_REPLACE_OBJECTS: "1",
			GIT_OPTIONAL_LOCKS: "0",
			GIT_TERMINAL_PROMPT: "0",
			HOME: "/",
			LANG: "C",
			LC_ALL: "C",
			PATH: "/usr/bin:/bin",
			TZ: "UTC",
		},
		input,
		maxBuffer: maximum,
		timeout: 60_000,
	});
	if (result.error || result.signal || result.status !== 0 || !admittedDarwinGitStderr(result.stderr) || result.stdout.length > maximum) {
		fail("P07B_C6_GIT_COMMAND", `${args[0]} status=${result.status ?? result.signal ?? "spawn"}`);
	}
	return result.stdout;
}

function admittedDarwinGitStderr(bytes) {
	if (!Buffer.isBuffer(bytes) || bytes.length > 16 * 1024) return false;
	if (bytes.length === 0) return true;
	const source = bytes.toString("utf8");
	if (!source.endsWith("\n") || source.includes("\r")) return false;
	return source.slice(0, -1).split("\n").every((line) =>
		/^git: warning: confstr\(\) failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using \/tmp instead$/u.test(line) ||
		/^20[0-9]{2}-[0-9]{2}-[0-9]{2} [0-9:.]+ xcodebuild\[[0-9]+:[0-9]+\]  DVTFilePathFSEvents: Failed to start fs event stream\.$/u.test(line) ||
		/^20[0-9]{2}-[0-9]{2}-[0-9]{2} [0-9:.]+ xcodebuild\[[0-9]+:[0-9]+\] \[MT\] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr\(3\), error = Error Domain=NSPOSIXErrorDomain Code=5 "Input\/output error"\. Using NSCachesDirectory instead\.$/u.test(line));
}

function oneLine(bytes, label) {
	if (!Buffer.isBuffer(bytes) || bytes.length < 2 || bytes.at(-1) !== 0x0a || bytes.includes(0x0d) ||
		bytes.subarray(0, -1).includes(0x0a)) fail("P07B_C6_GIT_FRAME", label);
	return bytes.subarray(0, -1).toString("utf8");
}

function validateNote(noteBytes) {
	if (digest(noteBytes) !== c5vAuthority.note_body_sha256) fail("P07B_C6_NOTE_BODY", "sha256");
	let note;
	try { note = JSON.parse(noteBytes.toString("utf8")); }
	catch (error) { fail("P07B_C6_NOTE_JSON", error.message); }
	if (note.commit !== c5vAuthority.commit || note.tree !== c5vAuthority.tree || note.secrets_override !== true ||
		note.version !== 1 || note.coverage?.total_events !== 12 || note.coverage?.by_coverage?.complete !== 12 ||
		Object.keys(note.coverage?.by_coverage ?? {}).length !== 1 || !Array.isArray(note.claims) || note.claims.length !== 12) {
		fail("P07B_C6_NOTE_ROOT", "identity, coverage, or seal disclosure");
	}
	for (let index = 0; index < c5vClaims.length; index += 1) {
		const row = note.claims[index];
		if (row?.claim?.label !== c5vClaims[index].label || row.claim.ctype !== c5vClaims[index].type ||
			row.claim.declared_at_index !== index || !exact(row.claim.event_indices, [index]) ||
			!exact(row.claim.pathspecs, []) || row.grade !== "tree-exact" || row.supporting_event_index !== index ||
			row.exit_code !== 0 || row.reason !== "self-stable command ran against the sealed tree") {
			fail("P07B_C6_NOTE_CLAIM", String(index));
		}
	}
	return note;
}

async function inspectArchive() {
	const descriptor = await describeDirectory(repositoryRoot, c5vAuthority.ledger_path);
	if (descriptor.file_count !== c5vAuthority.archive_file_count ||
		descriptor.object_count !== c5vAuthority.archive_object_count ||
		descriptor.total_bytes !== c5vAuthority.archive_total_bytes ||
		descriptor.manifest_sha256 !== c5vAuthority.archive_manifest_sha256) {
		fail("P07B_C6_ARCHIVE_DESCRIPTOR", JSON.stringify(descriptor));
	}
	const corePaths = {
		claims: `${c5vAuthority.ledger_path}/claims.jsonl`,
		gitignore: `${c5vAuthority.ledger_path}/.gitignore`,
		seals: `${c5vAuthority.ledger_path}/seals.jsonl`,
		session: `${c5vAuthority.ledger_path}/session.log`,
	};
	const core = Object.fromEntries(await Promise.all(Object.entries(corePaths).map(async ([name, path]) => [name, await readLocalRegular(path)])));
	for (const [name, expected] of Object.entries(expectedArchiveCore)) {
		if (digest(core[name]) !== expected) fail("P07B_C6_ARCHIVE_CORE", name);
	}
	const objectRoot = await resolveNoFollow(repositoryRoot, `${c5vAuthority.ledger_path}/objects`, { kind: "directory" });
	const objects = await readdir(objectRoot, { withFileTypes: true });
	if (objects.length !== c5vAuthority.archive_object_count ||
		objects.some((entry) => !entry.isFile() || !/^[0-9a-f]{64}$/u.test(entry.name))) {
		fail("P07B_C6_ARCHIVE_OBJECT_ROSTER", String(objects.length));
	}
	for (const entry of objects) {
		const bytes = await readLocalRegular(`${c5vAuthority.ledger_path}/objects/${entry.name}`, 128 * 1024 * 1024, 0);
		if (digest(bytes) !== entry.name) fail("P07B_C6_ARCHIVE_OBJECT_DIGEST", entry.name);
	}
	const session = jsonLines(core.session, "C5V session", 32);
	if (session.length !== c5vAuthority.archive_session_event_count) {
		fail("P07B_C6_ARCHIVE_SESSION_COUNT", String(session.length));
	}
	const blobReferences = session.flatMap((entry) =>
		[entry?.event?.stdout_blob, entry?.event?.stderr_blob, entry?.event?.transcript_blob]);
	if (blobReferences.some((name) => name !== null &&
		(typeof name !== "string" || !/^[0-9a-f]{64}$/u.test(name)))) {
		fail("P07B_C6_ARCHIVE_OBJECT_REFERENCE", "non-null blob reference shape");
	}
	const referencedObjects = sorted([...new Set(blobReferences.filter((name) => typeof name === "string"))]);
	if (
		!exact(referencedObjects, sorted(objects.map((entry) => entry.name)))) {
		fail("P07B_C6_ARCHIVE_OBJECT_CLOSURE", JSON.stringify(referencedObjects));
	}
	for (let index = 0; index < session.length; index += 1) {
		const entry = session[index];
		const expectedPrevious = index === 0 ? "0".repeat(64) : session[index - 1].entry_hash;
		exactKeys(entry, sessionEntryKeys, "P07B_C6_ARCHIVE_SESSION_SCHEMA");
		exactKeys(entry.event, sessionEventKeys, "P07B_C6_ARCHIVE_EVENT_SCHEMA");
		if (entry.index !== index || entry.prev_hash !== expectedPrevious || !/^[0-9a-f]{64}$/u.test(entry.entry_hash ?? "") ||
			entry.entry_hash !== didrunEntryHash(index, expectedPrevious, entry.event) ||
			entry.event.exit_code !== 0 || entry.event.coverage !== "complete" || entry.event.observed_via !== "wrapper" ||
			entry.event.cwd !== repositoryRoot || entry.event.env_fingerprint !== "ed9d01ef6528c9d4" ||
			entry.event.stderr_blob !== digest(Buffer.alloc(0)) || entry.event.transcript_blob !== null ||
			entry.event.submodule_dirty !== false || typeof entry.event.started_at !== "number" ||
			typeof entry.event.ended_at !== "number" || entry.event.ended_at < entry.event.started_at ||
			entry.event.tree_before !== c5vAuthority.tree || entry.event.tree_after !== c5vAuthority.tree) {
			fail("P07B_C6_ARCHIVE_SESSION_ENTRY", String(index));
		}
		if (index < 12) {
			const tail = expectedC5VCommandTails[index];
			if (!exact(entry.event.argv, [...expectedC5VCommandPrefix, ...tail])) {
				fail("P07B_C6_ARCHIVE_EVENT_ARGV", String(index));
			}
		} else if (!exact(entry.event.argv, [
			"/usr/bin/git", "notes", "--ref=didrun", "show", c5vAuthority.commit,
			]) || entry.event.stdout_blob !== c5vAuthority.note_body_sha256) fail("P07B_C6_ARCHIVE_NOTE_EVENT", "12");
		}
		if (session[0].entry_hash !== expectedArchiveChainAnchors.first ||
			session.at(-1).entry_hash !== expectedArchiveChainAnchors.last) {
			fail("P07B_C6_ARCHIVE_CHAIN_ANCHOR", `${session[0].entry_hash}/${session.at(-1).entry_hash}`);
		}
		const claims = jsonLines(core.claims, "C5V claims", 32);
	if (claims.length !== c5vClaims.length) fail("P07B_C6_ARCHIVE_CLAIM_COUNT", String(claims.length));
	for (let index = 0; index < claims.length; index += 1) {
		const claim = claims[index];
		exactKeys(claim, archiveClaimKeys, "P07B_C6_ARCHIVE_CLAIM_SCHEMA");
		if (claim?.ctype !== c5vClaims[index].type || claim.label !== c5vClaims[index].label ||
			claim.declared_at_index !== index || !exact(claim.event_indices, [index]) || !exact(claim.pathspecs, [])) {
			fail("P07B_C6_ARCHIVE_CLAIM", String(index));
		}
	}
	const seals = jsonLines(core.seals, "C5V seals", 4);
	if (seals.length !== 1) fail("P07B_C6_ARCHIVE_SEAL_COUNT", String(seals.length));
	exactKeys(seals[0], archiveSealKeys, "P07B_C6_ARCHIVE_SEAL_SCHEMA");
	if (seals[0]?.commit !== c5vAuthority.commit || seals[0]?.tree !== c5vAuthority.tree ||
		seals[0]?.claims_watermark !== 12) fail("P07B_C6_ARCHIVE_SEAL", JSON.stringify(seals[0]));
	const elapsedMS = [6, 7, 8].map((index) => {
		const elapsed = Math.round((session[index].event.ended_at - session[index].event.started_at) * 1000);
		if (!Number.isSafeInteger(elapsed) || elapsed < 1 || elapsed > 86_400_000) fail("P07B_C6_ARCHIVE_TIMING", String(index));
		return elapsed;
	});
	return Object.freeze({
		descriptor: Object.freeze({
			file_count: descriptor.file_count,
			manifest_sha256: descriptor.manifest_sha256,
			object_count: descriptor.object_count,
			path: descriptor.path,
			session_event_count: session.length,
			total_bytes: descriptor.total_bytes,
		}),
		elapsedMS: Object.freeze(elapsedMS),
	});
}

async function inspectPermanentNegative() {
	const rows = jsonLines(await readLocalRegular(failedAttemptPath), "C5V failed attempt", 16);
	if (rows.length !== 3 || rows[2]?.index !== 2 || rows[2]?.event?.exit_code !== -15 ||
		!exact(rows[2]?.event?.argv?.slice(-3), ["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"])) {
		fail("P07B_C6_PERMANENT_NEGATIVE", failedAttemptPath);
	}
}

async function inspectDidrunBugStatuses() {
	const source = (await readLocalRegular("docs/status/DIDRUN_BUGS.md", 4 * 1024 * 1024)).toString("utf8");
	for (const id of didrunBugIDs) {
		const start = source.indexOf(`## ${id} —`);
		const end = source.indexOf("\n## ", start + 4);
		const section = start >= 0 ? source.slice(start, end < 0 ? source.length : end) : "";
		if (start < 0 || !section.includes("- **C6A machine status:** `OPEN`")) fail("P07B_C6_DIDRUN_STATUS", id);
	}
}

async function inspectC5VTestkitAuthority() {
	const rows = [];
	for (const path of c5vTestkitPaths) {
		const listing = runGit(["ls-tree", "-z", c5vAuthority.tree, "--", path]);
		const match = /^(100644) blob ([0-9a-f]{40})\t([^\u0000]+)\u0000$/u.exec(listing.toString("utf8"));
		if (!match || match[3] !== path) fail("P07B_C6_C5V_TESTKIT_TREE", path);
		const sealedBytes = runGit(["cat-file", "blob", `${c5vAuthority.tree}:${path}`], { maximum: 32 * 1024 * 1024 });
		const measuredBlob = oneLine(runGit(["hash-object", "--stdin"], { input: sealedBytes }), `testkit blob ${path}`);
		const current = await readCurrentDescriptor(path, 32 * 1024 * 1024);
		if (measuredBlob !== match[2] || current.mode !== match[1] || current.sha256 !== digest(sealedBytes) ||
			!current.value.equals(sealedBytes)) {
			fail("P07B_C6_C5V_TESTKIT_DRIFT", path);
		}
		rows.push(Object.freeze({
			blob: match[2], bytes: current.bytes, mode: match[1], path, sha256: current.sha256,
		}));
	}
	return Object.freeze(rows);
}

async function inspectParentEvidence() {
	const tree = oneLine(runGit(["rev-parse", `${c5vAuthority.commit}^{tree}`]), "parent tree");
	const ancestry = oneLine(runGit(["rev-list", "--parents", "-n", "1", c5vAuthority.commit]), "parent ancestry").split(" ");
	const subject = oneLine(runGit(["show", "-s", "--format=%s", c5vAuthority.commit]), "parent subject");
	if (tree !== c5vAuthority.tree || !exact(ancestry, [c5vAuthority.commit, c5vAuthority.parent]) || subject !== c5vAuthority.subject) {
		fail("P07B_C6_PARENT_GIT", `${tree}/${ancestry.length}/${subject}`);
	}
	const noteBytes = runGit(["notes", "--ref=didrun", "show", c5vAuthority.commit]);
	validateNote(noteBytes);
	const noteBlob = oneLine(runGit(["hash-object", "--stdin"], { input: noteBytes }), "note blob");
	if (noteBlob !== c5vAuthority.note_blob) fail("P07B_C6_NOTE_BLOB", noteBlob);
	const html = await readLocalRegular(c5vAuthority.html_path, 16 * 1024 * 1024);
	if (html.length !== c5vAuthority.html_bytes || digest(html) !== c5vAuthority.html_sha256) {
		fail("P07B_C6_PARENT_HTML", `${html.length}/${digest(html)}`);
	}
	const archive = await inspectArchive();
	await inspectPermanentNegative();
	await inspectDidrunBugStatuses();
	const testkit = await inspectC5VTestkitAuthority();
	return Object.freeze({
		parentEvidence: Object.freeze({
			archive: archive.descriptor,
			claims: Object.freeze(c5vClaims.map((claim, index) => Object.freeze({
				grade: "TREE-EXACT", label: claim.label, supporting_event_index: index, type: claim.type,
			}))),
			commit: c5vAuthority.commit,
			html: Object.freeze({ bytes: html.length, path: c5vAuthority.html_path, sha256: digest(html) }),
			note_blob: noteBlob,
			note_body_sha256: digest(noteBytes),
			parent: c5vAuthority.parent,
			secrets_override: true,
			strict_grade_projection: "ALL_TREE_EXACT_FROM_SEALED_NOTE",
			subject: c5vAuthority.subject,
			tree: c5vAuthority.tree,
		}),
		elapsedMS: archive.elapsedMS,
		testkit,
	});
}

function parseConcatenatedJSON(bytes, label) {
	if (!Buffer.isBuffer(bytes) || bytes.length === 0 || bytes.length > 64 * 1024 * 1024 || bytes.includes(0x00)) {
		fail("P07B_C6_GO_LIST_ENVELOPE", label);
	}
	const source = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
	const values = [];
	let start = -1;
	let depth = 0;
	let inString = false;
	let escaped = false;
	for (let index = 0; index < source.length; index += 1) {
		const character = source[index];
		if (start < 0) {
			if (/\s/u.test(character)) continue;
			if (character !== "{") fail("P07B_C6_GO_LIST_FRAME", `${label}:${index}`);
			start = index;
			depth = 1;
			continue;
		}
		if (inString) {
			if (escaped) escaped = false;
			else if (character === "\\") escaped = true;
			else if (character === "\"") inString = false;
			continue;
		}
		if (character === "\"") inString = true;
		else if (character === "{") depth += 1;
		else if (character === "}") {
			depth -= 1;
			if (depth === 0) {
				try { values.push(JSON.parse(source.slice(start, index + 1))); }
				catch (error) { fail("P07B_C6_GO_LIST_PARSE", `${label}:${error.message}`); }
				start = -1;
			}
		}
	}
	if (start >= 0 || inString || depth !== 0 || values.length === 0) fail("P07B_C6_GO_LIST_TRUNCATED", label);
	return values;
}

function selectedProfilePackages(descriptors) {
	const packages = sorted(new Set(descriptors.flatMap((descriptor) => descriptor.targets.map((target) => target.packageArgument))));
	if (!exact(packages, c6SourceClosurePackages)) fail("P07B_C6_SOURCE_PACKAGE_ROSTER", JSON.stringify(packages));
	return packages;
}

async function deriveSelectedProfileSourcePaths(runtime, descriptors) {
	const packages = selectedProfilePackages(descriptors);
	if (!exact(c6GoListArguments, ["list", "-deps", "-test", "-json", ...packages])) {
		fail("P07B_C6_GO_LIST_ARGUMENT_AUTHORITY", JSON.stringify(c6GoListArguments));
	}
	const output = await runtime.run(
		"go",
		[...c6GoListArguments],
		"P07B_C6_GO_LIST_RUN",
		{ timeout: 10 * 60 * 1000 },
	);
	const paths = new Set(explicitSourceInputs);
	for (const record of parseConcatenatedJSON(output, "selected profiles")) {
		if (!record || typeof record !== "object" || Array.isArray(record)) {
			fail("P07B_C6_GO_LIST_RECORD", JSON.stringify(record));
		}
		if (record.Module?.Main !== true) continue;
		if (typeof record.Dir !== "string") fail("P07B_C6_GO_LIST_RECORD", JSON.stringify(record));
		for (const field of sourceFileFields) {
			const names = record[field] ?? [];
			if (!Array.isArray(names) || names.some((name) => typeof name !== "string" || name.length === 0)) {
				fail("P07B_C6_GO_LIST_FILES", `${record.ImportPath ?? record.Dir}:${field}`);
			}
			for (const name of names) {
				const absolute = isAbsolute(name) ? resolve(name) : resolve(record.Dir, name);
				if (isWithin(runtime.roots.runRoot, absolute)) continue;
				const candidate = confinedRelativePath(absolute);
				if (candidate === null) continue;
				const admitted = await resolveNoFollow(repositoryRoot, candidate, { kind: "file" });
				const info = await lstat(admitted);
				if (!info.isFile() || info.isSymbolicLink() || info.nlink !== 1) {
					fail("P07B_C6_GO_LIST_SOURCE_MODE", candidate);
				}
				paths.add(candidate);
			}
		}
	}
	const derived = sorted(paths);
	if (!exact(derived, c6SourceInputPaths)) {
		const missing = c6SourceInputPaths.filter((path) => !paths.has(path));
		const extra = derived.filter((path) => !c6SourceInputPaths.includes(path));
		fail("P07B_C6_SOURCE_CLOSURE", `missing=${JSON.stringify(missing)} extra=${JSON.stringify(extra)}`);
	}
	return Object.freeze({ packages: Object.freeze(packages), paths: Object.freeze(derived) });
}

function environmentContract() {
	return c6ExecutionEnvironmentContract;
}

function exactOutputLines(bytes, expected, label) {
	if (!Buffer.isBuffer(bytes) || bytes.length < 2 || bytes.at(-1) !== 0x0a || bytes.includes(0x0d) || bytes.includes(0x00)) {
		fail("P07B_C6_TOOL_OUTPUT_FRAME", label);
	}
	const lines = bytes.subarray(0, -1).toString("utf8").split("\n");
	if (lines.length !== expected || lines.some((line) => line.length === 0)) fail("P07B_C6_TOOL_OUTPUT_LINES", label);
	return lines;
}

async function collectExecutionAuthority(runtime) {
	const [goarch, goos, goroot, goversion] = exactOutputLines(
		await runtime.run("go", ["env", "GOARCH", "GOOS", "GOROOT", "GOVERSION"], "P07B_C6_GO_ENV"),
		4,
		"go env",
	);
	const [versionOutput] = exactOutputLines(
		await runtime.run("go", ["version"], "P07B_C6_GO_VERSION"),
		1,
		"go version",
	);
	if (versionOutput !== `go version ${goversion} ${goos}/${goarch}`) {
		fail("P07B_C6_GO_VERSION_VALUE", versionOutput);
	}
	return Object.freeze({
		authority_scope: "FRONT_DOOR_EXECUTABLES_AND_DECLARED_GO_ENVIRONMENT_ONLY",
		environment_contract: environmentContract(),
		go_env: Object.freeze({ goarch, goos, goroot, goversion }),
		tools: Object.freeze(["cc", "cxx", "git", "go", "node", "sh"].map((name) => Object.freeze({
			bytes: runtime.admitted[name].size,
			name,
			path: runtime.admitted[name].path,
			sha256: runtime.admitted[name].sha256,
		}))),
		version_output: versionOutput,
	});
}

function sourceClosureFor(derived) {
	if (!exact(derived.packages, c6SourceClosurePackages) || !exact(derived.paths, c6SourceInputPaths) ||
		!exact(c6AbsentSourceInputPaths, ["go.sum"])) {
		fail("P07B_C6_SOURCE_CLOSURE_BUILD", "derived closure differs from frozen authority");
	}
	return buildSourceClosure();
}

async function collectAdmissionSnapshot(runtime, descriptors) {
	const executionAuthority = await collectExecutionAuthority(runtime);
	const parent = await inspectParentEvidence();
	const derived = await deriveSelectedProfileSourcePaths(runtime, descriptors);
	const sourceClosure = sourceClosureFor(derived);
	const sourceInputs = await collectSourceInputs(repositoryRoot);
	await runtime.revalidate();
	const admissionSha256 = computeAdmissionSha256(
		executionAuthority, sourceClosure, parent.parentEvidence, sourceInputs,
	);
	return Object.freeze({
		admissionSha256,
		executionAuthority,
		parent,
		sourceClosure,
		sourceInputs,
		testkit: parent.testkit,
	});
}

async function runSelectedProfile(runtime, descriptor) {
	const output = await runtime.run(
		"go",
		descriptor.arguments,
		"P07B_C6_GO_JSON_RUN",
		{ timeout: 30 * 60 * 1000 },
	);
	return validateGoJSONTranscript(descriptor, output);
}

async function buildCurrentEvidence(runtime) {
	const descriptors = c6SelectedProfiles.map(goJSONProfileDescriptor);
	if (!exact(descriptors, c6ProfileDescriptors)) {
		fail("P07B_C6_PROFILE_AUTHORITY_DRIFT", "architecture descriptor differs from pure C6 authority");
	}
	const before = await collectAdmissionSnapshot(runtime, descriptors);
	const runs = [];
	for (const descriptor of descriptors) {
		runs.push(buildProfileRun(descriptor, await runSelectedProfile(runtime, descriptor), before.executionAuthority));
	}
	const after = await collectAdmissionSnapshot(runtime, descriptors);
	if (!exact(before, after)) fail("P07B_C6_ADMISSION_CHANGED", "selected profile execution");
	assertProfileRunsMatchDescriptors(runs, descriptors);
	const evidence = buildEvidenceDocument(before.parent.parentEvidence, runs, before.sourceInputs, {
		executionAuthority: before.executionAuthority,
		sourceClosure: before.sourceClosure,
	});
	const summary = buildSummary(before.parent.elapsedMS);
	return Object.freeze({ admission: before, evidence, summary });
}

async function readExactTrackedEvidence() {
	const tracked = await readTrackedEvidence(repositoryRoot);
	const descriptors = c6SelectedProfiles.map(goJSONProfileDescriptor);
	assertProfileRunsMatchDescriptors(tracked.evidence.profile_runs, descriptors);
	return tracked;
}

async function exists(path) {
	try { await lstat(path); return true; }
	catch (error) { if (error?.code === "ENOENT") return false; throw error; }
}

function effectiveUID() {
	return typeof process.geteuid === "function" ? process.geteuid() : -1;
}

async function syncPath(path) {
	const handle = await open(path, "r");
	try { await handle.sync(); }
	finally { await handle.close(); }
}

function artifactRows(artifacts) {
	const prefix = "docs/captures/p07b-c/";
	const names = sorted([...artifacts.keys()].map((path) => {
		if (!path.startsWith(prefix) || path.slice(prefix.length).length === 0 || path.slice(prefix.length).includes("/")) {
			fail("P07B_C6_ARTIFACT_INSTALL_PATH", path);
		}
		return path.slice(prefix.length);
	}));
	const expectedNames = sorted(Object.values(c6ArtifactPaths).map((path) => path.slice(prefix.length)));
	if (!exact(names, expectedNames)) fail("P07B_C6_ARTIFACT_INSTALL_ROSTER", JSON.stringify(names));
	return Object.freeze(names.map((name) => {
		const bytes = artifacts.get(`${prefix}${name}`);
		if (!Buffer.isBuffer(bytes) || bytes.length === 0) fail("P07B_C6_ARTIFACT_INSTALL_BYTES", name);
		return Object.freeze({ bytes: bytes.length, name, sha256: digest(bytes) });
	}));
}

async function requireExactArtifactDirectory(path, rows) {
	const rootInfo = await lstat(path);
	if (!rootInfo.isDirectory() || rootInfo.isSymbolicLink() || (rootInfo.mode & 0o7777) !== 0o755 ||
		(effectiveUID() >= 0 && rootInfo.uid !== effectiveUID())) fail("P07B_C6_ARTIFACT_DIRECTORY_MODE", path);
	if (!exact(sorted(await readdir(path)), rows.map((row) => row.name))) {
		fail("P07B_C6_ARTIFACT_DIRECTORY_ROSTER", path);
	}
	for (const row of rows) {
		const target = join(path, row.name);
		const before = await lstat(target);
		const bytes = await readFile(target);
		const after = await lstat(target);
		if (!before.isFile() || before.isSymbolicLink() || before.nlink !== 1 || (before.mode & 0o7777) !== 0o644 ||
			before.dev !== after.dev || before.ino !== after.ino || before.size !== after.size ||
			before.mtimeMs !== after.mtimeMs || bytes.length !== row.bytes || digest(bytes) !== row.sha256) {
			fail("P07B_C6_ARTIFACT_DIRECTORY_BYTES", row.name);
		}
	}
}

function validateArtifactGenerationRecord(value, code) {
	exactKeys(value, ["artifact_manifest_sha256", "artifacts", "device", "inode"], `${code}_SCHEMA`);
	if (!Number.isSafeInteger(value.device) || value.device < 0 || !Number.isSafeInteger(value.inode) || value.inode <= 0 ||
		!/^[0-9a-f]{64}$/u.test(value.artifact_manifest_sha256) || !Array.isArray(value.artifacts) ||
		value.artifacts.length !== Object.keys(c6ArtifactPaths).length) fail(`${code}_VALUE`, JSON.stringify(value));
	const rows = value.artifacts.map((row) => {
		exactKeys(row, ["bytes", "name", "sha256"], `${code}_ROW_SCHEMA`);
		if (!Number.isSafeInteger(row.bytes) || row.bytes <= 0 || row.bytes > 8 * 1024 * 1024 ||
			!/^[0-9a-f]{64}$/u.test(row.sha256) || typeof row.name !== "string" || row.name.includes("/") ||
			row.name.includes("\\")) fail(`${code}_ROW`, JSON.stringify(row));
		return Object.freeze({ bytes: row.bytes, name: row.name, sha256: row.sha256 });
	});
	const expectedNames = sorted(Object.values(c6ArtifactPaths).map((path) => path.slice("docs/captures/p07b-c/".length)));
	if (!exact(rows.map((row) => row.name), expectedNames) ||
		digest(canonicalCompact(rows)) !== value.artifact_manifest_sha256) fail(`${code}_MANIFEST`, JSON.stringify(value));
	return Object.freeze({
		artifact_manifest_sha256: value.artifact_manifest_sha256,
		artifacts: Object.freeze(rows),
		device: value.device,
		inode: value.inode,
	});
}

async function inspectArtifactGeneration(path) {
	const rootInfo = await lstat(path);
	if (!rootInfo.isDirectory() || rootInfo.isSymbolicLink() || (rootInfo.mode & 0o7777) !== 0o755 ||
		(effectiveUID() >= 0 && rootInfo.uid !== effectiveUID()) || !Number.isSafeInteger(rootInfo.dev) ||
		!Number.isSafeInteger(rootInfo.ino)) fail("P07B_C6_ARTIFACT_OLD_MODE", path);
	const names = sorted(await readdir(path));
	const expectedNames = sorted(Object.values(c6ArtifactPaths).map((entry) => entry.slice("docs/captures/p07b-c/".length)));
	if (!exact(names, expectedNames)) fail("P07B_C6_ARTIFACT_OLD_ROSTER", JSON.stringify(names));
	const rows = [];
	for (const name of names) {
		const target = join(path, name);
		const before = await lstat(target);
		if (!before.isFile() || before.isSymbolicLink() || before.nlink !== 1 || (before.mode & 0o7777) !== 0o644 ||
			before.size <= 0 || before.size > 8 * 1024 * 1024 ||
			(effectiveUID() >= 0 && before.uid !== effectiveUID())) fail("P07B_C6_ARTIFACT_OLD_FILE", name);
		const bytes = await readFile(target);
		const after = await lstat(target);
		if (!after.isFile() || after.isSymbolicLink() || before.dev !== after.dev || before.ino !== after.ino ||
			before.size !== after.size || before.mtimeMs !== after.mtimeMs) fail("P07B_C6_ARTIFACT_OLD_CHANGED", name);
		rows.push(Object.freeze({ bytes: bytes.length, name, sha256: digest(bytes) }));
	}
	return validateArtifactGenerationRecord({
		artifact_manifest_sha256: digest(canonicalCompact(rows)),
		artifacts: rows,
		device: rootInfo.dev,
		inode: rootInfo.ino,
	}, "P07B_C6_ARTIFACT_OLD_GENERATION");
}

async function requireArtifactGeneration(path, generation, code) {
	const expected = validateArtifactGenerationRecord(generation, code);
	const info = await lstat(path);
	if (!info.isDirectory() || info.isSymbolicLink() || info.dev !== expected.device || info.ino !== expected.inode) {
		fail(`${code}_IDENTITY`, path);
	}
	await requireExactArtifactDirectory(path, expected.artifacts);
}

function artifactJournalRecord(state, transaction) {
	return Object.freeze({
		artifact_manifest_sha256: transaction.artifactManifestSha256,
		artifacts: transaction.rows,
		backup: transaction.backupName,
		had_old: transaction.hadOld,
		new_generation: transaction.newGeneration,
		old_generation: transaction.oldGeneration,
		schema_version: artifactTransactionSchema,
		stage: transaction.stageName,
		state,
	});
}

function validateJournalRecord(value, capturesRoot) {
	exactKeys(value, [
		"artifact_manifest_sha256", "artifacts", "backup", "had_old", "new_generation", "old_generation",
		"schema_version", "stage", "state",
	], "P07B_C6_ARTIFACT_JOURNAL_SCHEMA");
	if (value.schema_version !== artifactTransactionSchema ||
		!["prepared", "backed_up", "activated", "committed"].includes(value.state) ||
		typeof value.had_old !== "boolean" || !/^[0-9a-f]{64}$/u.test(value.artifact_manifest_sha256) ||
		!/^\.p07b-c\.stage-[0-9a-f]{24}$/u.test(value.stage) ||
		!/^\.p07b-c\.backup-[0-9a-f]{24}$/u.test(value.backup) ||
		!Array.isArray(value.artifacts) || value.artifacts.length !== Object.keys(c6ArtifactPaths).length) {
		fail("P07B_C6_ARTIFACT_JOURNAL_VALUE", capturesRoot);
	}
	const rows = value.artifacts.map((row) => {
		exactKeys(row, ["bytes", "name", "sha256"], "P07B_C6_ARTIFACT_JOURNAL_ROW_SCHEMA");
		if (!Number.isSafeInteger(row.bytes) || row.bytes <= 0 || !/^[0-9a-f]{64}$/u.test(row.sha256) ||
			typeof row.name !== "string" || row.name.includes("/") || row.name.includes("\\")) {
			fail("P07B_C6_ARTIFACT_JOURNAL_ROW", JSON.stringify(row));
		}
		return Object.freeze({ bytes: row.bytes, name: row.name, sha256: row.sha256 });
	});
	const expectedNames = sorted(Object.values(c6ArtifactPaths).map((path) => path.slice("docs/captures/p07b-c/".length)));
	if (!exact(rows.map((row) => row.name), expectedNames) || digest(canonicalCompact(rows)) !== value.artifact_manifest_sha256) {
		fail("P07B_C6_ARTIFACT_JOURNAL_MANIFEST", capturesRoot);
	}
	let oldGeneration = null;
	const newGeneration = validateArtifactGenerationRecord(value.new_generation, "P07B_C6_ARTIFACT_JOURNAL_NEW");
	if (newGeneration.artifact_manifest_sha256 !== value.artifact_manifest_sha256 ||
		!exact(newGeneration.artifacts, rows)) {
		fail("P07B_C6_ARTIFACT_JOURNAL_NEW_VALUE", capturesRoot);
	}
	if (value.had_old) {
		oldGeneration = validateArtifactGenerationRecord(value.old_generation, "P07B_C6_ARTIFACT_JOURNAL_OLD");
	} else if (value.old_generation !== null) {
		fail("P07B_C6_ARTIFACT_JOURNAL_OLD_VALUE", capturesRoot);
	}
	return Object.freeze({
		...value,
		artifacts: Object.freeze(rows),
		new_generation: newGeneration,
		old_generation: oldGeneration,
	});
}

async function writeJournal(capturesRoot, record) {
	validateJournalRecord(record, capturesRoot);
	const journal = join(capturesRoot, artifactJournalName);
	const temporary = join(capturesRoot, `.p07b-c.journal-write-${randomBytes(12).toString("hex")}`);
	const handle = await open(temporary, constants.O_CREAT | constants.O_EXCL | constants.O_WRONLY |
		(constants.O_NOFOLLOW ?? 0), 0o600);
	try {
		await handle.writeFile(canonicalCompact(record));
		await handle.sync();
	} finally { await handle.close(); }
	try {
		await rename(temporary, journal);
		await syncPath(capturesRoot);
	} catch (error) {
		try { await unlink(temporary); } catch {}
		throw error;
	}
}

async function readJournal(capturesRoot) {
	const path = join(capturesRoot, artifactJournalName);
	if (!await exists(path)) return null;
	const before = await lstat(path);
	if (!before.isFile() || before.isSymbolicLink() || before.nlink !== 1 || (before.mode & 0o7777) !== 0o600 ||
		before.size <= 0 || before.size > 1024 * 1024) fail("P07B_C6_ARTIFACT_JOURNAL_MODE", path);
	const bytes = await readFile(path);
	const after = await lstat(path);
	if (before.dev !== after.dev || before.ino !== after.ino || before.size !== after.size || before.mtimeMs !== after.mtimeMs) {
		fail("P07B_C6_ARTIFACT_JOURNAL_CHANGED", path);
	}
	let value;
	try { value = JSON.parse(bytes.toString("utf8")); }
	catch (error) { fail("P07B_C6_ARTIFACT_JOURNAL_PARSE", error.message); }
	if (!canonicalCompact(value).equals(bytes)) fail("P07B_C6_ARTIFACT_JOURNAL_CANONICAL", path);
	return validateJournalRecord(value, capturesRoot);
}

function pidLiveness(pid) {
	try { process.kill(pid, 0); return "live"; }
	catch (error) { return error?.code === "ESRCH" ? "absent" : "indeterminate"; }
}

async function acquireArtifactLock(capturesRoot, root, { liveness = pidLiveness } = {}) {
	const path = join(capturesRoot, artifactLockName);
	const record = Object.freeze({
		nonce: randomBytes(16).toString("hex"),
		pid: process.pid,
		repository_root_sha256: digest(Buffer.from(resolve(root))),
		schema_version: artifactLockSchema,
	});
	const serialized = canonicalCompact(record);
	let handle;
	try {
		handle = await open(path, constants.O_CREAT | constants.O_EXCL | constants.O_RDWR |
			(constants.O_NOFOLLOW ?? 0), 0o600);
	} catch (error) {
		if (error?.code !== "EEXIST") throw error;
		const before = await lstat(path);
		if (!before.isFile() || before.isSymbolicLink() || before.nlink !== 1 || (before.mode & 0o7777) !== 0o600 ||
			before.size <= 0 || before.size > 4096 || (effectiveUID() >= 0 && before.uid !== effectiveUID())) {
			fail("P07B_C6_ARTIFACT_LOCK_INVALID", path);
		}
		let existing;
		let lockBytes;
		try {
			lockBytes = await readFile(path);
			existing = JSON.parse(lockBytes.toString("utf8"));
		} catch (failure) { fail("P07B_C6_ARTIFACT_LOCK_INVALID", failure.message); }
		const after = await lstat(path);
		if (after.isSymbolicLink() || before.dev !== after.dev || before.ino !== after.ino ||
			before.size !== after.size || before.mtimeMs !== after.mtimeMs || !canonicalCompact(existing).equals(lockBytes)) {
			fail("P07B_C6_ARTIFACT_LOCK_INVALID", path);
		}
		exactKeys(existing, ["nonce", "pid", "repository_root_sha256", "schema_version"], "P07B_C6_ARTIFACT_LOCK_SCHEMA");
		if (existing.schema_version !== artifactLockSchema || !Number.isSafeInteger(existing.pid) || existing.pid <= 0 ||
			!/^[0-9a-f]{32}$/u.test(existing.nonce) || existing.repository_root_sha256 !== record.repository_root_sha256) {
			fail("P07B_C6_ARTIFACT_LOCK_INVALID", path);
		}
		const state = liveness(existing.pid);
		if (state === "absent") {
			fail("P07B_C6_ARTIFACT_STALE_LOCK", `pid=${existing.pid}; exact operator cleanup required`);
		}
		fail("P07B_C6_ARTIFACT_ALREADY_RUNNING", `pid=${existing.pid}; liveness=${state}`);
	}
	try {
		await handle.writeFile(serialized);
		await handle.sync();
		const opened = await handle.stat();
		const atPath = await lstat(path);
		if (!opened.isFile() || atPath.isSymbolicLink() || opened.dev !== atPath.dev || opened.ino !== atPath.ino) {
			fail("P07B_C6_ARTIFACT_LOCK_INTEGRITY", path);
		}
	} catch (error) {
		let cleanupFailure;
		try {
			const openedFailure = await handle.stat();
			const currentFailure = await lstat(path);
			if (!currentFailure.isSymbolicLink() && openedFailure.dev === currentFailure.dev &&
				openedFailure.ino === currentFailure.ino) {
				await unlink(path);
				await syncPath(capturesRoot);
			}
		} catch (failure) { cleanupFailure = failure; }
		try { await handle.close(); }
		catch (failure) { cleanupFailure ??= failure; }
		if (cleanupFailure) {
			throw new AggregateError([error, cleanupFailure], "artifact lock acquisition and cleanup both failed");
		}
		throw error;
	}
	let released = false;
	return Object.freeze({
		async release() {
			if (released) return;
			let failure;
			try {
				const current = await lstat(path);
				const openedNow = await handle.stat();
				if (current.isSymbolicLink() || current.dev !== openedNow.dev || current.ino !== openedNow.ino ||
					!(await readFile(path)).equals(serialized)) fail("P07B_C6_ARTIFACT_LOCK_INTEGRITY", path);
				await unlink(path);
				await syncPath(capturesRoot);
			} catch (error) { failure = error; }
			try { await handle.close(); }
			catch (error) { failure ??= error; }
			released = true;
			if (failure) throw failure;
		},
	});
}

async function withArtifactLock(capturesRoot, root, operation) {
	const lock = await acquireArtifactLock(capturesRoot, root);
	let result;
	let operationFailure;
	try { result = await operation(); }
	catch (error) { operationFailure = error; }
	let releaseFailure;
	try { await lock.release(); }
	catch (error) { releaseFailure = error; }
	if (operationFailure && releaseFailure) {
		throw new AggregateError([operationFailure, releaseFailure], "artifact operation and lock release both failed");
	}
	if (operationFailure) throw operationFailure;
	if (releaseFailure) throw releaseFailure;
	return result;
}

async function removeUnjournaledStage(path, rows) {
	if (!await exists(path)) return;
	await requireExactArtifactDirectory(path, rows);
	await rm(path, { recursive: true, force: false });
}

function artifactCleanupPaths(capturesRoot, journal) {
	const stageNonce = journal.stage.slice(".p07b-c.stage-".length);
	const backupNonce = journal.backup.slice(".p07b-c.backup-".length);
	return Object.freeze({
		active: join(capturesRoot, `.p07b-c.cleanup-active-${stageNonce}`),
		backup: join(capturesRoot, `.p07b-c.cleanup-backup-${backupNonce}`),
		stage: join(capturesRoot, `.p07b-c.cleanup-stage-${stageNonce}`),
	});
}

async function requireNoForeignTransactionResidue(capturesRoot, journal, cleanup) {
	const expected = new Set([
		journal.stage,
		journal.backup,
		...Object.values(cleanup).map((path) => path.slice(capturesRoot.length + 1)),
	]);
	const foreign = (await readdir(capturesRoot)).filter((name) =>
		(name.startsWith(".p07b-c.stage-") || name.startsWith(".p07b-c.backup-") ||
			name.startsWith(".p07b-c.cleanup-")) && !expected.has(name));
	if (foreign.length !== 0) {
		fail("P07B_C6_ARTIFACT_TRANSACTION_RESIDUE", JSON.stringify(sorted(foreign)));
	}
}

async function requireArtifactGenerationRoot(path, generation, code) {
	const expected = validateArtifactGenerationRecord(generation, code);
	const info = await lstat(path);
	if (!info.isDirectory() || info.isSymbolicLink() || (info.mode & 0o7777) !== 0o755 ||
		(effectiveUID() >= 0 && info.uid !== effectiveUID()) ||
		info.dev !== expected.device || info.ino !== expected.inode) {
		fail(`${code}_IDENTITY`, path);
	}
}

async function removeArtifactGenerationResumable(capturesRoot, source, quarantine, generation, code, {
	fault = "none",
} = {}) {
	if (!["none", "after-quarantine", "during-cleanup"].includes(fault)) {
		fail("P07B_C6_ARTIFACT_CLEANUP_FAULT_MODE", fault);
	}
	const sourceExists = await exists(source);
	const quarantineExists = await exists(quarantine);
	if (sourceExists && quarantineExists) fail("P07B_C6_ARTIFACT_CLEANUP_AMBIGUOUS", code);
	if (!sourceExists && !quarantineExists) return;
	if (sourceExists) {
		await requireArtifactGeneration(source, generation, code);
		await rename(source, quarantine);
		await syncPath(capturesRoot);
		await requireArtifactGenerationRoot(quarantine, generation, `${code}_QUARANTINE`);
	} else {
		await requireArtifactGenerationRoot(quarantine, generation, `${code}_QUARANTINE`);
	}
	if (fault === "after-quarantine") throw new SimulatedArtifactCrash(`${code}:after-quarantine`);
	if (fault === "during-cleanup") {
		const names = sorted(await readdir(quarantine));
		if (names.length > 0) {
			const target = join(quarantine, names[0]);
			const info = await lstat(target);
			if (!info.isFile() || info.isSymbolicLink()) fail("P07B_C6_ARTIFACT_CLEANUP_ENTRY", target);
			await unlink(target);
			await syncPath(quarantine);
		}
		throw new SimulatedArtifactCrash(`${code}:during-cleanup`);
	}
	await rm(quarantine, { recursive: true, force: false });
	await syncPath(capturesRoot);
}

async function recoverArtifactTransaction(capturesRoot, { fault = "none" } = {}) {
	if (![
		"none", "after-old-restored", "after-active-quarantine", "during-active-cleanup",
		"after-stage-quarantine", "during-stage-cleanup", "after-backup-quarantine", "during-backup-cleanup",
	].includes(fault)) fail("P07B_C6_ARTIFACT_RECOVERY_FAULT_MODE", fault);
	const unexpected = (await readdir(capturesRoot)).filter((name) => name.startsWith(".p07b-c.journal-write-"));
	if (unexpected.length !== 0) fail("P07B_C6_ARTIFACT_TRANSACTION_AMBIGUOUS", JSON.stringify(sorted(unexpected)));
	const journal = await readJournal(capturesRoot);
	if (journal === null) {
		const residue = (await readdir(capturesRoot)).filter((name) =>
			name.startsWith(".p07b-c.stage-") || name.startsWith(".p07b-c.backup-") ||
			name.startsWith(".p07b-c.cleanup-"));
		if (residue.length !== 0) fail("P07B_C6_ARTIFACT_TRANSACTION_RESIDUE", JSON.stringify(sorted(residue)));
		return "clean";
	}
	const active = join(capturesRoot, "p07b-c");
	const stage = join(capturesRoot, journal.stage);
	const backup = join(capturesRoot, journal.backup);
	const cleanup = artifactCleanupPaths(capturesRoot, journal);
	await requireNoForeignTransactionResidue(capturesRoot, journal, cleanup);
	const cleanupFault = (role) => fault === `after-${role}-quarantine` ? "after-quarantine" :
		fault === `during-${role}-cleanup` ? "during-cleanup" : "none";
	if (journal.state === "committed") {
		if (await exists(cleanup.active)) fail("P07B_C6_ARTIFACT_UNEXPECTED_ACTIVE_CLEANUP", journal.state);
		if (!await exists(active)) fail("P07B_C6_ARTIFACT_COMMITTED_ACTIVE_MISSING", active);
		await requireArtifactGeneration(active, journal.new_generation, "P07B_C6_ARTIFACT_COMMITTED_ACTIVE");
		await removeArtifactGenerationResumable(
			capturesRoot, stage, cleanup.stage, journal.new_generation, "P07B_C6_ARTIFACT_STAGE",
			{ fault: cleanupFault("stage") },
		);
		if (journal.had_old) {
			await removeArtifactGenerationResumable(
				capturesRoot, backup, cleanup.backup, journal.old_generation, "P07B_C6_ARTIFACT_BACKUP",
				{ fault: cleanupFault("backup") },
			);
		} else if (await exists(backup) || await exists(cleanup.backup)) {
			fail("P07B_C6_ARTIFACT_UNEXPECTED_BACKUP", journal.state);
		}
	} else if (journal.had_old) {
		if (await exists(cleanup.backup)) fail("P07B_C6_ARTIFACT_UNEXPECTED_BACKUP_CLEANUP", journal.state);
		if (await exists(backup)) {
			await requireArtifactGeneration(backup, journal.old_generation, "P07B_C6_ARTIFACT_BACKUP");
			await removeArtifactGenerationResumable(
				capturesRoot, active, cleanup.active, journal.new_generation, "P07B_C6_ARTIFACT_ACTIVE",
				{ fault: cleanupFault("active") },
			);
			await rename(backup, active);
			await syncPath(capturesRoot);
			await requireArtifactGeneration(active, journal.old_generation, "P07B_C6_ARTIFACT_RESTORED");
		} else {
			if (await exists(cleanup.active) || !await exists(active)) {
				fail("P07B_C6_ARTIFACT_PRECOMMIT_AMBIGUOUS", journal.state);
			}
			await requireArtifactGeneration(active, journal.old_generation, "P07B_C6_ARTIFACT_RESTORED");
		}
		if (fault === "after-old-restored") throw new SimulatedArtifactCrash(fault);
		await removeArtifactGenerationResumable(
			capturesRoot, stage, cleanup.stage, journal.new_generation, "P07B_C6_ARTIFACT_STAGE",
			{ fault: cleanupFault("stage") },
		);
	} else {
		if (await exists(backup) || await exists(cleanup.backup)) {
			fail("P07B_C6_ARTIFACT_UNEXPECTED_BACKUP", journal.state);
		}
		await removeArtifactGenerationResumable(
			capturesRoot, active, cleanup.active, journal.new_generation, "P07B_C6_ARTIFACT_ACTIVE",
			{ fault: cleanupFault("active") },
		);
		await removeArtifactGenerationResumable(
			capturesRoot, stage, cleanup.stage, journal.new_generation, "P07B_C6_ARTIFACT_STAGE",
			{ fault: cleanupFault("stage") },
		);
	}
	await syncPath(capturesRoot);
	await requireNoForeignTransactionResidue(capturesRoot, journal, cleanup);
	await unlink(join(capturesRoot, artifactJournalName));
	await syncPath(capturesRoot);
	return journal.state === "committed" ? "committed" : "rolled-back";
}

class SimulatedArtifactCrash extends Error {}

async function resolveCapturesRoot(root) {
	const docsRoot = await resolveNoFollow(root, "docs", { kind: "directory" });
	let capturesRoot;
	try { capturesRoot = await resolveNoFollow(root, "docs/captures", { kind: "directory" }); }
	catch (error) {
		if (error?.code !== "ENOENT") throw error;
		capturesRoot = join(docsRoot, "captures");
		await mkdir(capturesRoot, { mode: 0o755 });
		await syncPath(docsRoot);
		capturesRoot = await resolveNoFollow(root, "docs/captures", { kind: "directory" });
	}
	return capturesRoot;
}

async function recoverArtifactsAt(root, options = {}) {
	const capturesRoot = await resolveCapturesRoot(root);
	return withArtifactLock(capturesRoot, root, () => recoverArtifactTransaction(capturesRoot, options));
}

async function installArtifactsAt(root, artifacts, {
	beforeActivate = async () => {}, fault = "none", validateActive = async () => {},
} = {}) {
	if (!["none", "after-prepared", "after-backup-rename", "after-backup", "after-backed_up",
		"after-activation-rename", "after-activated", "after-committed", "backup-cleanup", "final-sync"].includes(fault)) {
		fail("P07B_C6_ARTIFACT_FAULT_MODE", fault);
	}
	const rows = artifactRows(artifacts);
	const capturesRoot = await resolveCapturesRoot(root);
	return withArtifactLock(capturesRoot, root, async () => {
		const captureRoot = join(capturesRoot, "p07b-c");
		let stage = null;
		let transaction = null;
		let journalStarted = false;
		let committed = false;
		try {
			await recoverArtifactTransaction(capturesRoot);
			const hadOld = await exists(captureRoot);
			const oldGeneration = hadOld ? await inspectArtifactGeneration(captureRoot) : null;
			const stageName = `.p07b-c.stage-${randomBytes(12).toString("hex")}`;
			const backupName = `.p07b-c.backup-${randomBytes(12).toString("hex")}`;
			stage = join(capturesRoot, stageName);
			await mkdir(stage, { mode: 0o700 });
			for (const [relativePath, bytes] of artifacts) {
				const name = relativePath.slice("docs/captures/p07b-c/".length);
				const handle = await open(join(stage, name), "wx", 0o600);
				try {
					await handle.writeFile(bytes);
					await handle.chmod(0o644);
					await handle.sync();
				} finally { await handle.close(); }
			}
			await chmod(stage, 0o755);
			await syncPath(stage);
			await requireExactArtifactDirectory(stage, rows);
			const newGeneration = await inspectArtifactGeneration(stage);
			transaction = Object.freeze({
				artifactManifestSha256: digest(canonicalCompact(rows)), backupName, hadOld, newGeneration,
				oldGeneration, rows, stageName,
			});
			await writeJournal(capturesRoot, artifactJournalRecord("prepared", transaction));
			journalStarted = true;
			if (fault === "after-prepared") throw new SimulatedArtifactCrash(fault);
			await beforeActivate();
			const backup = join(capturesRoot, backupName);
			if (hadOld) {
				await requireArtifactGeneration(captureRoot, oldGeneration, "P07B_C6_ARTIFACT_OLD_PREMOVE");
				await rename(captureRoot, backup);
				await syncPath(capturesRoot);
				await requireArtifactGeneration(backup, oldGeneration, "P07B_C6_ARTIFACT_BACKUP");
				if (fault === "after-backup-rename") throw new SimulatedArtifactCrash(fault);
			}
			await writeJournal(capturesRoot, artifactJournalRecord("backed_up", transaction));
			if (["after-backup", "after-backed_up"].includes(fault)) throw new SimulatedArtifactCrash(fault);
			await requireArtifactGeneration(
				stage, newGeneration, "P07B_C6_ARTIFACT_STAGE_PREACTIVATION",
			);
			await rename(stage, captureRoot);
			stage = null;
			await syncPath(capturesRoot);
			await requireArtifactGeneration(
				captureRoot, newGeneration, "P07B_C6_ARTIFACT_ACTIVE_POSTACTIVATION",
			);
			if (fault === "after-activation-rename") throw new SimulatedArtifactCrash(fault);
			await writeJournal(capturesRoot, artifactJournalRecord("activated", transaction));
			if (fault === "after-activated") throw new SimulatedArtifactCrash(fault);
			await requireExactArtifactDirectory(captureRoot, rows);
			await validateActive();
			await writeJournal(capturesRoot, artifactJournalRecord("committed", transaction));
			committed = true;
			if (fault === "after-committed") throw new SimulatedArtifactCrash(fault);
			if (fault === "backup-cleanup") throw new FinalEvidenceError("P07B_C6_SELFTEST_SWAP_FAULT", fault);
			if (await exists(backup)) {
				const cleanup = artifactCleanupPaths(capturesRoot, artifactJournalRecord("committed", transaction));
				await removeArtifactGenerationResumable(
					capturesRoot, backup, cleanup.backup, oldGeneration, "P07B_C6_ARTIFACT_BACKUP",
				);
			}
			if (fault === "final-sync") throw new FinalEvidenceError("P07B_C6_SELFTEST_SWAP_FAULT", fault);
			await syncPath(capturesRoot);
			await unlink(join(capturesRoot, artifactJournalName));
			await syncPath(capturesRoot);
		} catch (error) {
			let failure = error;
			if (!(error instanceof SimulatedArtifactCrash) && journalStarted && !committed) {
				try { await recoverArtifactTransaction(capturesRoot); }
				catch (recoveryFailure) {
					failure = new AggregateError([error, recoveryFailure], "artifact installation and rollback recovery failed");
				}
			} else if (!journalStarted && stage !== null && await exists(stage)) {
				try { await removeUnjournaledStage(stage, rows); }
				catch (cleanupFailure) {
					failure = new AggregateError([error, cleanupFailure], "artifact staging and cleanup both failed");
				}
			}
			throw failure;
		}
	});
}

async function installArtifacts(artifacts, admission, runtime) {
	const descriptors = c6SelectedProfiles.map(goJSONProfileDescriptor);
	await installArtifactsAt(repositoryRoot, artifacts, {
		beforeActivate: async () => {
			const current = await collectAdmissionSnapshot(runtime, descriptors);
			if (!exact(current, admission)) fail("P07B_C6_PREACTIVATION_ADMISSION_CHANGED", "source or authority drift");
		},
		validateActive: readExactTrackedEvidence,
	});
}

async function selfTestFileBoundaries() {
	const root = await mkdtemp(join(tmpdir(), "countershape-c6-evidence-selftest-"));
	let cases = 0;
	try {
		await writeFile(join(root, "go.sum"), "unexpected\n", { flag: "wx", mode: 0o644 });
		try { await collectSourceInputs(root); }
		catch (error) {
			if (!/P07B_C6_SOURCE_INPUT_REQUIRED_ABSENCE/u.test(error.message)) throw error;
			cases += 1;
		}
		await unlink(join(root, "go.sum"));
		const archive = join(root, "archive");
		await mkdir(archive, { mode: 0o700 });
		await writeFile(join(archive, "entry"), "exact\n", { flag: "wx", mode: 0o600 });
		await describeDirectory(root, "archive");
		cases += 1;
		await symlink(join(archive, "entry"), join(archive, "alias"));
		try { await describeDirectory(root, "archive"); }
		catch (error) {
			if (!/P07B_C6_ARCHIVE_SYMLINK/u.test(error.message)) throw error;
			cases += 1;
		}
		const outside = join(root, "outside");
		await mkdir(outside, { mode: 0o700 });
		await writeFile(join(outside, "entry"), "outside\n", { flag: "wx", mode: 0o600 });
		await symlink(outside, join(root, "linked"));
		try { await resolveNoFollow(root, "linked/entry", { kind: "file" }); }
		catch (error) {
			if (!/P07B_C6_PATH_SYMLINK/u.test(error.message)) throw error;
			cases += 1;
		}
		const captureRoot = join(root, "docs", "captures", "p07b-c");
		await mkdir(captureRoot, { recursive: true, mode: 0o755 });
		const makeArtifacts = (generation) => new Map(Object.values(c6ArtifactPaths).map((path, index) =>
			[path, Buffer.from(`artifact-${generation}-${index}\n`, "utf8")]));
		const oldArtifacts = makeArtifacts("old");
		for (const [path, bytes] of oldArtifacts) {
			await writeFile(join(captureRoot, path.slice("docs/captures/p07b-c/".length)), bytes, {
				flag: "wx", mode: 0o644,
			});
		}
		const assertOld = async () => {
			await requireExactArtifactDirectory(captureRoot, artifactRows(oldArtifacts));
			for (const [path, bytes] of oldArtifacts) {
				if (!(await readFile(join(captureRoot, path.slice("docs/captures/p07b-c/".length)))).equals(bytes)) {
					fail("P07B_C6_SELFTEST_SWAP_ROLLBACK", path);
				}
			}
		};
		for (const fault of [
			"after-prepared", "after-backup-rename", "after-backed_up", "after-activation-rename", "after-activated",
		]) {
			try { await installArtifactsAt(root, makeArtifacts(fault), { fault }); }
			catch (error) { if (!(error instanceof SimulatedArtifactCrash) || error.message !== fault) throw error; }
			if (await recoverArtifactsAt(root) !== "rolled-back") fail("P07B_C6_SELFTEST_RECOVERY_STATE", fault);
			await assertOld();
			cases += 1;
		}
		try { await installArtifactsAt(root, makeArtifacts("recovery-twice"), { fault: "after-activated" }); }
		catch (error) { if (!(error instanceof SimulatedArtifactCrash)) throw error; }
		try { await recoverArtifactsAt(root, { fault: "after-old-restored" }); }
		catch (error) { if (!(error instanceof SimulatedArtifactCrash) || error.message !== "after-old-restored") throw error; }
		if (await recoverArtifactsAt(root) !== "rolled-back") fail("P07B_C6_SELFTEST_RECOVERY_STATE", "recovery-twice");
		await assertOld();
		cases += 1;
		for (const [installFault, recoveryFault, expectedFragment] of [
			["after-prepared", "during-stage-cleanup", "P07B_C6_ARTIFACT_STAGE:during-cleanup"],
			["after-activated", "during-active-cleanup", "P07B_C6_ARTIFACT_ACTIVE:during-cleanup"],
		]) {
			try { await installArtifactsAt(root, makeArtifacts(recoveryFault), { fault: installFault }); }
			catch (error) { if (!(error instanceof SimulatedArtifactCrash)) throw error; }
			try { await recoverArtifactsAt(root, { fault: recoveryFault }); }
			catch (error) {
				if (!(error instanceof SimulatedArtifactCrash) || !error.message.includes(expectedFragment)) throw error;
			}
			if (await recoverArtifactsAt(root) !== "rolled-back") {
				fail("P07B_C6_SELFTEST_RECOVERY_STATE", recoveryFault);
			}
			await assertOld();
			cases += 1;
		}
		for (const prefix of [".p07b-c.stage-foreign-", ".p07b-c.backup-foreign-"]) {
			try { await installArtifactsAt(root, makeArtifacts(`foreign-${prefix}`), { fault: "after-prepared" }); }
			catch (error) { if (!(error instanceof SimulatedArtifactCrash)) throw error; }
			const capturesRoot = join(root, "docs", "captures");
			const foreign = join(capturesRoot, `${prefix}${randomBytes(8).toString("hex")}`);
			await mkdir(foreign, { mode: 0o755 });
			try { await recoverArtifactsAt(root); }
			catch (error) {
				if (!/P07B_C6_ARTIFACT_TRANSACTION_RESIDUE/u.test(error.message) || !await exists(foreign)) throw error;
			}
			await rm(foreign, { recursive: true, force: false });
			await syncPath(capturesRoot);
			if (await recoverArtifactsAt(root) !== "rolled-back") {
				fail("P07B_C6_SELFTEST_RECOVERY_STATE", prefix);
			}
			await assertOld();
			cases += 1;
		}
		const emptyRoot = join(root, "no-prior-generation");
		await mkdir(join(emptyRoot, "docs"), { recursive: true, mode: 0o755 });
		const emptyCaptureRoot = join(emptyRoot, "docs", "captures", "p07b-c");
		for (const [installFault, recoveryFault, expectedFragment] of [
			["after-prepared", "during-stage-cleanup", "P07B_C6_ARTIFACT_STAGE:during-cleanup"],
			["after-activated", "during-active-cleanup", "P07B_C6_ARTIFACT_ACTIVE:during-cleanup"],
		]) {
			try { await installArtifactsAt(emptyRoot, makeArtifacts(`no-prior-${recoveryFault}`), { fault: installFault }); }
			catch (error) { if (!(error instanceof SimulatedArtifactCrash)) throw error; }
			try { await recoverArtifactsAt(emptyRoot, { fault: recoveryFault }); }
			catch (error) {
				if (!(error instanceof SimulatedArtifactCrash) || !error.message.includes(expectedFragment)) throw error;
			}
			if (await recoverArtifactsAt(emptyRoot) !== "rolled-back") {
				fail("P07B_C6_SELFTEST_RECOVERY_STATE", `no-prior-${recoveryFault}`);
			}
			if (await exists(emptyCaptureRoot)) {
				fail("P07B_C6_SELFTEST_FIRST_GENERATION_RETAINED", recoveryFault);
			}
			const emptyCapturesRoot = join(emptyRoot, "docs", "captures");
			const emptyResidue = (await readdir(emptyCapturesRoot)).filter((name) => name.startsWith(".p07b-c."));
			if (emptyResidue.length !== 0) {
				fail("P07B_C6_SELFTEST_TRANSACTION_RESIDUE", JSON.stringify(emptyResidue));
			}
			cases += 1;
		}
		try {
			await installArtifactsAt(root, makeArtifacts("validation-failure"), {
				validateActive: async () => { fail("P07B_C6_SELFTEST_ACTIVE_VALIDATION", "injected"); },
			});
		} catch (error) {
			if (!/P07B_C6_SELFTEST_ACTIVE_VALIDATION/u.test(error.message)) throw error;
		}
		await assertOld();
		if (await recoverArtifactsAt(root) !== "clean") fail("P07B_C6_SELFTEST_RECOVERY_STATE", "validation-failure");
		cases += 1;
		const committedArtifacts = makeArtifacts("committed-crash");
		try { await installArtifactsAt(root, committedArtifacts, { fault: "after-committed" }); }
		catch (error) { if (!(error instanceof SimulatedArtifactCrash)) throw error; }
		try { await recoverArtifactsAt(root, { fault: "during-backup-cleanup" }); }
		catch (error) {
			if (!(error instanceof SimulatedArtifactCrash) ||
				!error.message.includes("P07B_C6_ARTIFACT_BACKUP:during-cleanup")) throw error;
		}
		if (await recoverArtifactsAt(root) !== "committed") fail("P07B_C6_SELFTEST_COMMIT_RECOVERY", "after-committed");
		await requireExactArtifactDirectory(captureRoot, artifactRows(committedArtifacts));
		cases += 1;
		for (const fault of ["backup-cleanup", "final-sync"]) {
			const artifacts = makeArtifacts(fault);
			try { await installArtifactsAt(root, artifacts, { fault }); }
			catch (error) { if (!/P07B_C6_SELFTEST_SWAP_FAULT/u.test(error.message)) throw error; }
			if (await recoverArtifactsAt(root) !== "committed") fail("P07B_C6_SELFTEST_COMMIT_RECOVERY", fault);
			await requireExactArtifactDirectory(captureRoot, artifactRows(artifacts));
			cases += 1;
		}
		const capturesRoot = join(root, "docs", "captures");
		const held = await acquireArtifactLock(capturesRoot, root);
		try {
			try { await acquireArtifactLock(capturesRoot, root); }
			catch (error) {
				if (!/P07B_C6_ARTIFACT_ALREADY_RUNNING/u.test(error.message)) throw error;
				cases += 1;
			}
		} finally { await held.release(); }
		const stalePath = join(capturesRoot, artifactLockName);
		const staleBytes = canonicalCompact({
			nonce: "a".repeat(32),
			pid: 4242,
			repository_root_sha256: digest(Buffer.from(resolve(root))),
			schema_version: artifactLockSchema,
		});
		await writeFile(stalePath, staleBytes, { flag: "wx", mode: 0o600 });
		for (let attempt = 0; attempt < 2; attempt += 1) {
			let rejected = false;
			try { await acquireArtifactLock(capturesRoot, root, { liveness: () => "absent" }); }
			catch (error) {
				if (!/P07B_C6_ARTIFACT_STALE_LOCK/u.test(error.message) || !(await readFile(stalePath)).equals(staleBytes)) throw error;
				rejected = true;
				cases += 1;
			}
			if (!rejected) fail("P07B_C6_SELFTEST_STALE_LOCK_FALSE_NEGATIVE", String(attempt));
		}
		await unlink(stalePath);
		await syncPath(capturesRoot);
		const artifacts = makeArtifacts("success");
		await installArtifactsAt(root, artifacts);
		await requireExactArtifactDirectory(captureRoot, artifactRows(artifacts));
		cases += 1;
		await writeFile(join(captureRoot, "foreign.txt"), "foreign\n", { flag: "wx", mode: 0o644 });
		try { await installArtifactsAt(root, makeArtifacts("foreign-refusal")); }
		catch (error) {
			if (!/P07B_C6_ARTIFACT_OLD_ROSTER/u.test(error.message) ||
				(await readFile(join(captureRoot, "foreign.txt"), "utf8")) !== "foreign\n") throw error;
			cases += 1;
		}
		await unlink(join(captureRoot, "foreign.txt"));
		await requireExactArtifactDirectory(captureRoot, artifactRows(artifacts));
		const residue = (await readdir(capturesRoot)).filter((name) => name.startsWith(".p07b-c."));
		if (residue.length !== 0) fail("P07B_C6_SELFTEST_TRANSACTION_RESIDUE", JSON.stringify(residue));
		cases += 1;
	} finally {
		await rm(root, { recursive: true, force: true });
	}
	return cases;
}

async function runSelfTest() {
	const cases = runPureSelfTest() + await selfTestFileBoundaries();
	process.stdout.write(`P07B-C C6 final evidence defensive self-test OK (${cases} data, rendering, and file-boundary cases)\n`);
}

function usage() {
	process.stderr.write("usage: node tools/p07b-c/check-final-evidence.mjs --write|--verify|--self-test|--render 60|80|120\n");
	process.exitCode = 2;
}

async function main() {
	const argv = process.argv.slice(2);
	if (argv.length === 1 && argv[0] === "--self-test") {
		await runSelfTest();
		return;
	}
	if (argv.length === 2 && argv[0] === "--render" && c6Widths.includes(Number(argv[1])) && String(Number(argv[1])) === argv[1]) {
		const tracked = await readExactTrackedEvidence();
		const width = Number(argv[1]);
		const rendered = renderEvidence(tracked.evidence, tracked.summary, width);
		validateRenderBytes(Buffer.from(rendered, "utf8"), width);
		process.stdout.write(rendered);
		return;
	}
	if (argv.length !== 1 || !["--write", "--verify"].includes(argv[0])) {
		usage();
		return;
	}
	const runtime = await createExecutionRuntime();
	try {
		const current = await buildCurrentEvidence(runtime);
		const artifacts = expectedArtifactBytes(current.evidence, current.summary);
		if (argv[0] === "--write") {
			await installArtifacts(artifacts, current.admission, runtime);
			process.stdout.write("P07B-C C6 exact expert evidence artifacts written\n");
			return;
		}
		const tracked = await readExactTrackedEvidence();
		if (!exact(tracked.evidence, current.evidence) || !exact(tracked.summary, current.summary)) {
			fail("P07B_C6_TRACKED_EVIDENCE_DRIFT", "live exact profiles or sealed-parent provenance differ");
		}
		process.stdout.write("P07B-C C6 exact CLI HTTP Node and documentation evidence closure OK (26 exact tests; 60/80/120 columns)\n");
	} finally {
		await runtime.close();
	}
}

if (process.argv[1] && resolve(process.argv[1]) === checkerPath) {
	main().catch((error) => {
		process.stderr.write(`${error.stack ?? error}\n`);
		process.exitCode = 1;
	});
}
