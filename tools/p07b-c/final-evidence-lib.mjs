import { createHash } from "node:crypto";
import { lstat, readFile, readdir, realpath } from "node:fs/promises";
import { isAbsolute, join, relative, resolve, sep } from "node:path";
import { TextDecoder } from "node:util";

import {
	c6AbsentSourceInputPaths,
	c6GoListArguments,
	c6SourceInputPathDigest,
	c6SourceInputPaths,
} from "./source-closure.mjs";
import {
	c6ProfileDescriptors,
	c6SelectedProfiles,
	c6SourceClosurePackages,
} from "./profile-authority.mjs";

export {
	c6AbsentSourceInputPaths, c6GoListArguments, c6ProfileDescriptors, c6SelectedProfiles, c6SourceClosurePackages,
	c6SourceInputPathDigest, c6SourceInputPaths,
};

export const c6EvidenceSchema = "countershape/p07b-c-c6a-expert-evidence/v4";
export const c6SummarySchema = "countershape/p07b-c-c6a-evidence-summary/v1";
export const c6RenderGrammar = "countershape/p07b-c-c6a-terminal-evidence/v2";
export const c6EnvironmentSchema = "countershape/p07b-c-c6a-go-environment/v1";
export const c6Widths = Object.freeze([60, 80, 120]);
export const c6ExecutionEnvironmentContract = Object.freeze({
	authority_bindings: Object.freeze([
		"CC=tool:cc",
		"COUNTERSHAPE_CC=tool:cc",
		"COUNTERSHAPE_CXX=tool:cxx",
		"COUNTERSHAPE_GIT=tool:git",
		"COUNTERSHAPE_GO=tool:go",
		"COUNTERSHAPE_NODE=tool:node",
		"COUNTERSHAPE_SH=tool:sh",
		"CXX=tool:cxx",
	]),
	base: "EMPTY",
	fixed_values: Object.freeze({
		CGO_ENABLED: "1",
		GOENV: "off",
		GOFLAGS: "-mod=readonly -buildvcs=false -p=1",
		GOMAXPROCS: "2",
		GOPROXY: "off",
		GOSUMDB: "off",
		GOTOOLCHAIN: "local",
		GOVCS: "*:off",
		GOWORK: "off",
		LANG: "C",
		LC_ALL: "C",
		NO_COLOR: "1",
		TZ: "UTC",
	}),
	path: "private-authority-bin:/usr/bin:/bin",
	private_directories: Object.freeze(["GOCACHE", "GOMODCACHE", "GOPATH", "GOTMPDIR", "HOME", "TMPDIR"]),
	schema_version: c6EnvironmentSchema,
});
export const c6ArtifactPaths = Object.freeze({
	evidence: "docs/captures/p07b-c/c6a-expert-evidence.json",
	summary: "docs/captures/p07b-c/c6a-evidence-summary.json",
	render60: "docs/captures/p07b-c/c6a-diagnostics-60.txt",
	render80: "docs/captures/p07b-c/c6a-diagnostics-80.txt",
	render120: "docs/captures/p07b-c/c6a-diagnostics-120.txt",
});

export const c5vAuthority = Object.freeze({
	archive_file_count: 18,
	archive_manifest_sha256: "8b314f001b6f2b714008d246ef6ed6e850bd1c78213a858e13ccba9c0de5d949",
	archive_object_count: 14,
	archive_session_event_count: 13,
	archive_total_bytes: 176_204,
	commit: "e7f51c0a8fbd2d6fbe8eacb4a7f0d46f010af7ba",
	html_bytes: 8540,
	html_path: ".countershape/evidence/p07b-c-c5v-final-e7f51c0a8fbd.html",
	html_sha256: "1e7f90f4a9bb8d2dc809967d5152b60eb901606d492ee558df53e2c720b6915d",
	ledger_path: ".didrun-history/p07b-c-c5v-final-e7f51c0a8fbd/.didrun",
	note_blob: "a6631221b39f1fa6c5d9d5d9042f93bde67a7be1",
	note_body_sha256: "1bccdece04834c104e4a70c4ba832a24ad40a1baf5d0d911fb0456809f37f3a2",
	parent: "240060f018d5b0e86914bd27361c5899ac9ba1c0",
	subject: "fix: harden final verification hygiene",
	tree: "65bdfc39b48c3b9b1b832a12cc9be5e0083ea5c4",
});

export const c5vClaims = Object.freeze([
	Object.freeze({ label: "P07B-C C5V candidate phase plan coherence", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C5V independent candidate transition authority", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C5V plan checker defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C5V sealed-C5 Git-note and lower ancestry compatibility", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C5V unit-scope defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C5V cumulative verifier defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C5V cumulative verification pass 1", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C5V cumulative verification pass 2", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C5V cumulative verification pass 3", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C5V exact nine-path staged scope and diff integrity", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C5V scoped staged credential-pattern scan", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C5V sealed-C5 predecessor and preceding didrun chain integrity", type: "command-succeeded" }),
]);

export const c6Nonclaims = Object.freeze([
	"adoption or market demand",
	"human comprehension or taste",
	"maintainership",
	"production hardening or cross-platform support",
	"security review or hostile containment",
]);

const exactDidrunFindings = Object.freeze([
	Object.freeze({
		id: "S6-01",
		status: "OPEN",
		severity: "UNASSESSED",
		scope: "concurrent session append and chain integrity",
		effect: "Concurrent writers can fork the ledger, invalidate claim addressing, and leave the session unusable as evidence.",
		reference: "docs/status/DIDRUN_BUGS.md#s6-01--concurrent-writers-fork-one-session-chain",
		summary: "Tail read and append are not protected by an interprocess lock.",
	}),
	Object.freeze({
		id: "S6-02",
		status: "OPEN",
		severity: "UNASSESSED",
		scope: "wrapped command output and operator liveness",
		effect: "Recorded failures require direct immutable-blob inspection, while long commands provide no bounded progress signal.",
		reference: "docs/status/DIDRUN_BUGS.md#s6-02--failed-wrapper-output-is-not-surfaced-by-the-cli",
		summary: "The CLI retains wrapped output but does not expose it or stream bounded progress.",
	}),
	Object.freeze({
		id: "S6-06",
		status: "OPEN",
		severity: "UNASSESSED",
		scope: "entropy scanning and redacted evidence export",
		effect: "Ordinary digests and fixtures cause aggregate secret warnings without enough locality for precise classification.",
		reference: "docs/status/DIDRUN_BUGS.md#s6-06--entropy-scanning-blocks-ordinary-cryptographic-fixtures",
		summary: "High-entropy findings have high false-positive pressure and expose only an aggregate count.",
	}),
	Object.freeze({
		id: "S6-07",
		status: "OPEN",
		severity: "UNASSESSED",
		scope: "claim addressing and strict verification on forked ledgers",
		effect: "Duplicate stored indices can misbind claims, and strict verification can fail open unless chain integrity is checked separately.",
		reference: "docs/status/DIDRUN_BUGS.md#s6-07--a-second-concurrent-append-breaks-claim-address-stability-and-can-evade-strict",
		summary: "Claims use unstable physical offsets and sealing does not first reject a forked session chain.",
	}),
	Object.freeze({
		id: "S6-10",
		status: "OPEN",
		severity: "UNASSESSED",
		scope: "Git-note publication and local seal state",
		effect: "A denied note write can return success and advance the local seal watermark despite absent commit-bound evidence.",
		reference: "docs/status/DIDRUN_BUGS.md#s6-10--git-note-attachment-failure-advances-local-seal-state-fail-open",
		summary: "Seal publication is not atomic or fail-closed across local state and the Git note.",
	}),
	Object.freeze({
		id: "S6-13",
		status: "OPEN",
		severity: "UNASSESSED",
		scope: "portable strict verification of archived ledgers",
		effect: "A sealed commit becomes UNKNOWN when its matching ignored ledger moves, even though its Git note remains present.",
		reference: "docs/status/DIDRUN_BUGS.md#s6-13--strict-verification-is-not-self-contained-after-a-sealed-ledger-moves",
		summary: "Strict verification cannot select an archived witness ledger and the Git note is not self-sufficient.",
	}),
]);

const exactDidrunBugs = Object.freeze(exactDidrunFindings.map(({ id, status }) =>
	Object.freeze({ id, status })));

const exactPermanentNegative = Object.freeze({
	evidence: ".didrun-history/p07b-c-c5v-final-failed-20260731-operator-terminated/session.log retains event 2 with exit -15",
	id: "C5V_FINAL_ATTEMPT_OPERATOR_TERMINATED",
});

function fail(code, detail) {
	throw new Error(`${code}: ${detail}`);
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

function safeText(value, code, maximum = 512) {
	if (typeof value !== "string" || value.length === 0 || value.length > maximum ||
		/[\u0000-\u001f\u007f-\u009f\u202a-\u202e\u2066-\u2069]/u.test(value)) {
		fail(code, JSON.stringify(value));
	}
	return value;
}

function digest(value) {
	return createHash("sha256").update(value).digest("hex");
}

function clone(value) {
	return structuredClone(value);
}

function validateEnvironmentContract(value) {
	exactKeys(value, [
		"authority_bindings", "base", "fixed_values", "path", "private_directories", "schema_version",
	], "P07B_C6_ENVIRONMENT_KEYS");
	if (!exact(value, c6ExecutionEnvironmentContract)) {
		fail("P07B_C6_ENVIRONMENT_VALUE", JSON.stringify(value));
	}
	return true;
}

export function validateExecutionAuthority(value) {
	exactKeys(value, ["authority_scope", "environment_contract", "go_env", "tools", "version_output"],
		"P07B_C6_EXECUTION_AUTHORITY_KEYS");
	if (value.authority_scope !== "FRONT_DOOR_EXECUTABLES_AND_DECLARED_GO_ENVIRONMENT_ONLY") {
		fail("P07B_C6_EXECUTION_AUTHORITY_SCOPE", JSON.stringify(value.authority_scope));
	}
	validateEnvironmentContract(value.environment_contract);
	exactKeys(value.go_env, ["goarch", "goos", "goroot", "goversion"], "P07B_C6_GO_ENV_KEYS");
	if (["goarch", "goos", "goroot", "goversion"].some((key) => typeof value.go_env[key] !== "string") ||
		!/^[a-z0-9_]{2,32}$/u.test(value.go_env.goarch) || !/^[a-z0-9_]{2,32}$/u.test(value.go_env.goos) ||
		!isAbsolute(value.go_env.goroot) || value.go_env.goroot.length > 4096 ||
		/[\u0000-\u001f\u007f]/u.test(value.go_env.goroot) ||
		!/^go1\.[0-9]+(?:\.[0-9]+)?(?:[a-z0-9.-]+)?$/u.test(value.go_env.goversion)) {
		fail("P07B_C6_GO_ENV_VALUE", JSON.stringify(value.go_env));
	}
	const expectedToolNames = ["cc", "cxx", "git", "go", "node", "sh"];
	if (!Array.isArray(value.tools) || value.tools.length !== expectedToolNames.length ||
		!exact(value.tools.map((tool) => tool?.name), expectedToolNames)) {
		fail("P07B_C6_EXECUTION_TOOL_ROSTER", JSON.stringify(value.tools));
	}
	for (const tool of value.tools) {
		exactKeys(tool, ["bytes", "name", "path", "sha256"], "P07B_C6_EXECUTION_TOOL_KEYS");
		if (!Number.isSafeInteger(tool.bytes) || tool.bytes < 1 || tool.bytes > 512 * 1024 * 1024 ||
			typeof tool.path !== "string" || !isAbsolute(tool.path) || tool.path.length > 4096 ||
			/[\u0000-\u001f\u007f]/u.test(tool.path) || typeof tool.sha256 !== "string" ||
			!/^[0-9a-f]{64}$/u.test(tool.sha256)) {
			fail("P07B_C6_EXECUTION_TOOL_VALUE", tool.name);
		}
	}
	if (value.version_output !==
		`go version ${value.go_env.goversion} ${value.go_env.goos}/${value.go_env.goarch}`) {
		fail("P07B_C6_GO_VERSION_OUTPUT", JSON.stringify(value.version_output));
	}
	return true;
}

export function buildSourceClosure() {
	return Object.freeze({
		absent_paths: Object.freeze([...c6AbsentSourceInputPaths]),
		derivation: "hermetic-go-list-deps-test-json-plus-explicit-runtime-inputs/v1",
		go_list_arguments: Object.freeze([...c6GoListArguments]),
		packages: Object.freeze([...c6SourceClosurePackages]),
		path_count: c6SourceInputPaths.length,
		paths_sha256: c6SourceInputPathDigest,
	});
}

export function validateSourceClosure(value) {
	exactKeys(value, ["absent_paths", "derivation", "go_list_arguments", "packages", "path_count", "paths_sha256"],
		"P07B_C6_SOURCE_CLOSURE_KEYS");
	if (c6SourceInputPathDigest !== `sha256:${digest(canonicalCompact(c6SourceInputPaths))}`) {
		fail("P07B_C6_SOURCE_CLOSURE_AUTHORITY", c6SourceInputPathDigest);
	}
	if (!exact(value, buildSourceClosure())) fail("P07B_C6_SOURCE_CLOSURE_VALUE", JSON.stringify(value));
	return true;
}

export function computeAdmissionSha256(executionAuthority, sourceClosure, parentEvidence, sourceInputs) {
	validateExecutionAuthority(executionAuthority);
	validateSourceClosure(sourceClosure);
	validateParentEvidence(parentEvidence);
	validateSourceInputs(sourceInputs);
	return digest(canonicalCompact({
		execution_authority: executionAuthority,
		parent_evidence: parentEvidence,
		profile_descriptors: c6ProfileDescriptors,
		source_closure: sourceClosure,
		source_inputs: sourceInputs,
	}));
}

export async function resolveNoFollow(root, relativePath, { allowMissingFinal = false, kind = "any" } = {}) {
	if (typeof relativePath !== "string" || relativePath.length === 0 || relativePath.includes("\\") ||
		isAbsolute(relativePath) || !["any", "directory", "file"].includes(kind)) {
		fail("P07B_C6_PATH_INPUT", JSON.stringify(relativePath));
	}
	const components = relativePath.split("/");
	if (components.some((component) => component.length === 0 || component === "." || component === "..")) {
		fail("P07B_C6_PATH_COMPONENT", relativePath);
	}
	const logicalRoot = resolve(root);
	const physicalRoot = await realpath(logicalRoot);
	const absolute = resolve(physicalRoot, ...components);
	const containment = relative(physicalRoot, absolute);
	if (containment === "" || containment === ".." || containment.startsWith(`..${sep}`) || isAbsolute(containment)) {
		fail("P07B_C6_PATH_CONTAINMENT", relativePath);
	}
	let cursor = physicalRoot;
	for (let index = 0; index < components.length; index += 1) {
		cursor = join(cursor, components[index]);
		const final = index === components.length - 1;
		let info;
		try { info = await lstat(cursor); }
		catch (error) {
			if (final && allowMissingFinal && error?.code === "ENOENT") return cursor;
			throw error;
		}
		if (info.isSymbolicLink()) fail("P07B_C6_PATH_SYMLINK", relativePath);
		if (!final && !info.isDirectory()) fail("P07B_C6_PATH_ANCESTOR", relativePath);
		if (final && kind === "directory" && !info.isDirectory()) fail("P07B_C6_PATH_KIND", relativePath);
		if (final && kind === "file" && !info.isFile()) fail("P07B_C6_PATH_KIND", relativePath);
	}
	return cursor;
}

export function canonicalPretty(value) {
	return Buffer.from(`${JSON.stringify(value, null, 2)}\n`, "utf8");
}

export function canonicalCompact(value) {
	return Buffer.from(`${JSON.stringify(value)}\n`, "utf8");
}

export function decodeCanonicalJSON(bytes, label, style = "pretty") {
	if (!Buffer.isBuffer(bytes) || bytes.length === 0 || bytes.length > 8 * 1024 * 1024 ||
		bytes.at(-1) !== 0x0a || bytes.includes(0x0d)) fail("P07B_C6_JSON_ENVELOPE", label);
	let text;
	try { text = new TextDecoder("utf-8", { fatal: true }).decode(bytes); }
	catch (error) { fail("P07B_C6_JSON_UTF8", `${label}:${error.message}`); }
	let value;
	try { value = JSON.parse(text); }
	catch (error) { fail("P07B_C6_JSON_PARSE", `${label}:${error.message}`); }
	const expected = style === "compact" ? canonicalCompact(value) : canonicalPretty(value);
	if (!bytes.equals(expected)) fail("P07B_C6_JSON_CANONICAL", label);
	return value;
}

function validateArchiveDescriptor(value) {
	exactKeys(value, [
		"file_count", "manifest_sha256", "object_count", "path", "session_event_count", "total_bytes",
	], "P07B_C6_PARENT_ARCHIVE_KEYS");
	if (value.path !== c5vAuthority.ledger_path || value.file_count !== c5vAuthority.archive_file_count ||
		value.object_count !== c5vAuthority.archive_object_count ||
		value.session_event_count !== c5vAuthority.archive_session_event_count ||
		value.total_bytes !== c5vAuthority.archive_total_bytes ||
		value.manifest_sha256 !== c5vAuthority.archive_manifest_sha256) {
		fail("P07B_C6_PARENT_ARCHIVE_VALUE", JSON.stringify(value));
	}
}

function validateParentEvidence(value) {
	exactKeys(value, [
		"archive", "claims", "commit", "html", "note_blob", "note_body_sha256", "parent", "secrets_override",
		"strict_grade_projection", "subject", "tree",
	], "P07B_C6_PARENT_KEYS");
	for (const key of ["commit", "note_blob", "note_body_sha256", "parent", "subject", "tree"]) {
		if (value[key] !== c5vAuthority[key]) fail("P07B_C6_PARENT_IDENTITY", key);
	}
	if (value.secrets_override !== true || value.strict_grade_projection !== "ALL_TREE_EXACT_FROM_SEALED_NOTE") {
		fail("P07B_C6_PARENT_STRICT", JSON.stringify(value));
	}
	validateArchiveDescriptor(value.archive);
	exactKeys(value.html, ["bytes", "path", "sha256"], "P07B_C6_PARENT_HTML_KEYS");
	if (value.html.bytes !== c5vAuthority.html_bytes || value.html.path !== c5vAuthority.html_path ||
		value.html.sha256 !== c5vAuthority.html_sha256) fail("P07B_C6_PARENT_HTML", JSON.stringify(value.html));
	if (!Array.isArray(value.claims) || value.claims.length !== c5vClaims.length) fail("P07B_C6_PARENT_CLAIMS", "cardinality");
	for (let index = 0; index < value.claims.length; index += 1) {
		const claim = value.claims[index];
		exactKeys(claim, ["grade", "label", "supporting_event_index", "type"], "P07B_C6_PARENT_CLAIM_KEYS");
		if (claim.label !== c5vClaims[index].label || claim.type !== c5vClaims[index].type ||
			claim.grade !== "TREE-EXACT" || claim.supporting_event_index !== index) {
			fail("P07B_C6_PARENT_CLAIM", String(index));
		}
	}
}

function targetAdapter(packageArgument) {
	if (packageArgument === "./internal/contractexec/http" || packageArgument === "./testkit/contractexec/http") return "HTTP";
	if (packageArgument === "./testkit/contractexec/cli") return "CLI";
	if (packageArgument === "./internal/emit/node/parity") return "NODE";
	return null;
}

function validateProfileRun(value, executionAuthority) {
	exactKeys(value, [
		"arguments", "arguments_sha256", "environment_contract", "executable", "invocation", "invocation_sha256",
		"passed", "profile", "replay_policy", "result", "skipped", "targets", "working_directory",
	], "P07B_C6_PROFILE_KEYS");
	validateExecutionAuthority(executionAuthority);
	validateEnvironmentContract(value.environment_contract);
	const goTool = executionAuthority.tools.find((tool) => tool.name === "go");
	const expectedDescriptor = c6ProfileDescriptors.find((descriptor) => descriptor.name === value.profile);
	if (!c6SelectedProfiles.includes(value.profile) || !Array.isArray(value.arguments) || value.arguments.length < 8 ||
		value.arguments.length > 256 || value.arguments.some((entry) => typeof entry !== "string" || entry.length === 0 ||
			entry.length > 16_384 || /[\u0000-\u001f\u007f]/u.test(entry) || entry.startsWith("/")) ||
		value.arguments_sha256 !== digest(canonicalCompact(value.arguments)) ||
		value.executable !== goTool.path || !exact(value.environment_contract, executionAuthority.environment_contract) ||
		!Array.isArray(value.invocation) || !exact(value.invocation, [value.executable, ...value.arguments]) ||
		value.invocation_sha256 !== digest(canonicalCompact(value.invocation)) ||
		value.working_directory !== "repository-root" ||
		!exact(value.replay_policy, {
			copy_paste_safe: false,
			requires_environment_contract: true,
			standalone_argv: false,
		}) ||
		value.result !== "PASSED_EXACT_ROSTER" || value.skipped !== 0 || !Array.isArray(value.targets) ||
		expectedDescriptor === undefined || !exact(value.arguments, expectedDescriptor.arguments)) {
		fail("P07B_C6_PROFILE_VALUE", JSON.stringify(value));
	}
	const expectedCount = value.profile === "c5-http-behavior" ? 4 : 22;
	const expectedAdapters = value.profile === "c5-http-behavior" ? ["HTTP", "HTTP"] : ["HTTP", "CLI", "NODE"];
	const expectedTargetCounts = value.profile === "c5-http-behavior" ? [2, 2] : [1, 4, 17];
	if (value.passed !== expectedCount || value.targets.length !== expectedAdapters.length) {
		fail("P07B_C6_PROFILE_CARDINALITY", value.profile);
	}
	const allTests = [];
	for (let index = 0; index < value.targets.length; index += 1) {
		const target = value.targets[index];
		exactKeys(target, ["adapter", "package_argument", "package_path", "tests"], "P07B_C6_TARGET_KEYS");
		const expectedTarget = expectedDescriptor.targets[index];
		if (target.adapter !== expectedAdapters[index] || targetAdapter(target.package_argument) !== target.adapter ||
			typeof target.package_path !== "string" || !target.package_path.startsWith("github.com/nelsonwerd/countershape/") ||
			!Array.isArray(target.tests) || target.tests.length !== expectedTargetCounts[index] ||
			!exact(target.tests, sorted(target.tests)) || target.tests.some((name) => !/^(?:Fuzz|Test)[A-Za-z0-9_]+$/u.test(name)) ||
			(target.adapter !== "NODE" && target.tests.some((name) => name.startsWith("Fuzz"))) ||
			(target.adapter === "NODE" && !exact(target.tests.filter((name) => name.startsWith("Fuzz")),
				["FuzzParseContractParityCorpusLine"])) || expectedTarget === undefined ||
			target.package_argument !== expectedTarget.packageArgument || target.package_path !== expectedTarget.packagePath ||
			!exact(target.tests, sorted(expectedTarget.pass))) {
			fail("P07B_C6_TARGET_VALUE", `${value.profile}:${index}`);
		}
		allTests.push(...target.tests);
	}
	if (new Set(allTests).size !== allTests.length || allTests.length !== expectedCount) {
		fail("P07B_C6_PROFILE_TEST_ROSTER", value.profile);
	}
}

export function buildProfileRun(descriptor, result, executionAuthority) {
	if (!descriptor || typeof descriptor !== "object" || Array.isArray(descriptor) ||
		!c6SelectedProfiles.includes(descriptor.name) || !Array.isArray(descriptor.arguments) ||
		!Array.isArray(descriptor.targets) || !result || typeof result !== "object" || Array.isArray(result)) {
		fail("P07B_C6_PROFILE_BUILD_INPUT", descriptor?.name ?? "missing");
	}
	validateProfileDescriptor(descriptor);
	exactKeys(result, ["passed", "profile", "skipped"], "P07B_C6_PROFILE_RESULT_KEYS");
	if (result.profile !== descriptor.name) fail("P07B_C6_PROFILE_RESULT_IDENTITY", result.profile);
	validateExecutionAuthority(executionAuthority);
	const goTool = executionAuthority.tools.find((tool) => tool.name === "go");
	const argumentsCopy = [...descriptor.arguments];
	const invocation = [goTool.path, ...argumentsCopy];
	const targets = descriptor.targets.map((target) => Object.freeze({
		adapter: targetAdapter(target.packageArgument),
		package_argument: target.packageArgument,
		package_path: target.packagePath,
		tests: Object.freeze(sorted(target.pass)),
	}));
	const profile = {
		arguments: Object.freeze(argumentsCopy),
		arguments_sha256: digest(canonicalCompact(argumentsCopy)),
		environment_contract: clone(executionAuthority.environment_contract),
		executable: goTool.path,
		invocation: Object.freeze(invocation),
		invocation_sha256: digest(canonicalCompact(invocation)),
		passed: result.passed,
		profile: descriptor.name,
		replay_policy: {
			copy_paste_safe: false,
			requires_environment_contract: true,
			standalone_argv: false,
		},
		result: "PASSED_EXACT_ROSTER",
		skipped: result.skipped,
		targets: Object.freeze(targets),
		working_directory: "repository-root",
	};
	validateProfileRun(profile, executionAuthority);
	return Object.freeze(profile);
}

function validateSourceInputs(value) {
	if (!Array.isArray(value) || value.length !== c6SourceInputPaths.length ||
		!exact(value.map((entry) => entry?.path), c6SourceInputPaths)) {
		fail("P07B_C6_SOURCE_INPUT_ROSTER", JSON.stringify(value));
	}
	for (const input of value) {
		exactKeys(input, ["bytes", "mode", "path", "sha256"], "P07B_C6_SOURCE_INPUT_KEYS");
		if (!Number.isSafeInteger(input.bytes) || input.bytes < 1 || input.bytes > 32 * 1024 * 1024 ||
			input.mode !== "100644" || !/^[0-9a-f]{64}$/u.test(input.sha256)) {
			fail("P07B_C6_SOURCE_INPUT_VALUE", input.path);
		}
	}
	return true;
}

export function validateEvidenceDocument(value) {
	exactKeys(value, [
		"admission_sha256", "artifact_state", "boundary", "didrun_findings", "documentation", "execution_authority",
		"parent_evidence", "profile_runs", "schema_version", "source_closure", "source_inputs", "surfaces",
	], "P07B_C6_EVIDENCE_KEYS");
	if (value.schema_version !== c6EvidenceSchema || value.boundary !== "C6A" || !exact(value.artifact_state, {
		boundary: "C6A",
		document_epoch: "C6A_SOURCE_CANDIDATE",
		receipt: "ABSENT",
		state: "UNRECEIPTED",
	})) {
		fail("P07B_C6_EVIDENCE_IDENTITY", JSON.stringify(value));
	}
	validateExecutionAuthority(value.execution_authority);
	validateSourceClosure(value.source_closure);
	if (value.admission_sha256 !== computeAdmissionSha256(
		value.execution_authority, value.source_closure, value.parent_evidence, value.source_inputs,
	)) {
		fail("P07B_C6_ADMISSION_DIGEST", JSON.stringify(value.admission_sha256));
	}
	validateParentEvidence(value.parent_evidence);
	if (!exact(value.didrun_findings, exactDidrunFindings)) {
		fail("P07B_C6_DIDRUN_FINDINGS", JSON.stringify(value.didrun_findings));
	}
	if (!Array.isArray(value.profile_runs) || value.profile_runs.length !== c6SelectedProfiles.length ||
		!exact(value.profile_runs.map((entry) => entry.profile), c6SelectedProfiles)) {
		fail("P07B_C6_PROFILE_ORDER", JSON.stringify(value.profile_runs));
	}
	for (const profile of value.profile_runs) validateProfileRun(profile, value.execution_authority);
	validateSourceInputs(value.source_inputs);
	exactKeys(value.documentation, ["render_grammar", "source", "widths"], "P07B_C6_DOCUMENTATION_KEYS");
	if (value.documentation.render_grammar !== c6RenderGrammar ||
		value.documentation.source !== "canonical expert evidence plus derived C6A summary" ||
		!exact(value.documentation.widths, c6Widths)) fail("P07B_C6_DOCUMENTATION_VALUE", JSON.stringify(value.documentation));
	if (!Array.isArray(value.surfaces) || value.surfaces.length !== 3) fail("P07B_C6_SURFACE_CARDINALITY", "three");
	const expectedSurfaces = [
		{ adapter: "CLI", conclusion: "PHYSICAL_CLI_CLOSURE_EXERCISED", profile: "c5-cross-profile-parity", test_count: 4 },
		{ adapter: "HTTP", conclusion: "PHYSICAL_HTTP_BEHAVIOR_AND_PARITY_EXERCISED", profile: "c5-http-behavior+c5-cross-profile-parity", test_count: 5 },
		{ adapter: "NODE", conclusion: "GENERATED_NODE_PARITY_EXERCISED", profile: "c5-cross-profile-parity", test_count: 17 },
	];
	for (let index = 0; index < expectedSurfaces.length; index += 1) {
		exactKeys(value.surfaces[index], ["adapter", "conclusion", "profile", "test_count"], "P07B_C6_SURFACE_KEYS");
		if (!exact(value.surfaces[index], expectedSurfaces[index])) fail("P07B_C6_SURFACE_VALUE", String(index));
	}
	return Object.freeze(value);
}

export function validateSummaryDocument(value) {
	exactKeys(value, [
		"didrun_bugs", "nonclaims", "permanent_negatives", "private_evidence", "schema_version", "timings",
	], "P07B_C6_SUMMARY_KEYS");
	if (value.schema_version !== c6SummarySchema || !exact(value.nonclaims, c6Nonclaims) ||
		!exact(value.didrun_bugs, exactDidrunBugs) || !exact(value.permanent_negatives, [exactPermanentNegative])) {
		fail("P07B_C6_SUMMARY_FIXED_FIELDS", JSON.stringify(value));
	}
	if (!Array.isArray(value.timings) || value.timings.length !== 3 ||
		!exact(value.timings.map((entry) => entry.label), [
			"sealed-c5v/cumulative-verification/pass-1",
			"sealed-c5v/cumulative-verification/pass-2",
			"sealed-c5v/cumulative-verification/pass-3",
		])) fail("P07B_C6_SUMMARY_TIMINGS", JSON.stringify(value.timings));
	for (const timing of value.timings) {
		exactKeys(timing, ["elapsed_ms", "label"], "P07B_C6_SUMMARY_TIMING_KEYS");
		if (!Number.isSafeInteger(timing.elapsed_ms) || timing.elapsed_ms < 1 || timing.elapsed_ms > 86_400_000) {
			fail("P07B_C6_SUMMARY_TIMING_VALUE", JSON.stringify(timing));
		}
	}
	exactKeys(value.private_evidence, ["availability", "declared_blobs", "declared_bytes", "note"], "P07B_C6_PRIVATE_KEYS");
	if (value.private_evidence.availability !== "UNAVAILABLE" || value.private_evidence.declared_blobs !== 0 ||
		value.private_evidence.declared_bytes !== 0 || value.private_evidence.note !==
		"C6A source-era state is UNRECEIPTED. It declares no dedicated private capture blobs; tracked captures are sanitized and confidentiality is not established.") {
		fail("P07B_C6_PRIVATE_VALUE", JSON.stringify(value.private_evidence));
	}
	return Object.freeze(value);
}

export function buildSummary(elapsedMS) {
	const timings = elapsedMS.map((elapsed_ms, index) => ({
		elapsed_ms,
		label: `sealed-c5v/cumulative-verification/pass-${index + 1}`,
	}));
	const summary = {
		didrun_bugs: clone(exactDidrunBugs),
		nonclaims: [...c6Nonclaims],
		permanent_negatives: [clone(exactPermanentNegative)],
		private_evidence: {
			availability: "UNAVAILABLE",
			declared_blobs: 0,
			declared_bytes: 0,
			note: "C6A source-era state is UNRECEIPTED. It declares no dedicated private capture blobs; tracked captures are sanitized and confidentiality is not established.",
		},
		schema_version: c6SummarySchema,
		timings,
	};
	return validateSummaryDocument(summary);
}

export function buildEvidenceDocument(parentEvidence, profileRuns, sourceInputs, options) {
	exactKeys(options, ["executionAuthority", "sourceClosure"], "P07B_C6_EVIDENCE_BUILD_OPTIONS");
	const { executionAuthority, sourceClosure } = options;
	validateExecutionAuthority(executionAuthority);
	validateSourceClosure(sourceClosure);
	const evidence = {
		admission_sha256: computeAdmissionSha256(executionAuthority, sourceClosure, parentEvidence, sourceInputs),
		artifact_state: {
			boundary: "C6A",
			document_epoch: "C6A_SOURCE_CANDIDATE",
			receipt: "ABSENT",
			state: "UNRECEIPTED",
		},
		boundary: "C6A",
		didrun_findings: clone(exactDidrunFindings),
		documentation: {
			render_grammar: c6RenderGrammar,
			source: "canonical expert evidence plus derived C6A summary",
			widths: [...c6Widths],
		},
		execution_authority: executionAuthority,
		parent_evidence: parentEvidence,
		profile_runs: profileRuns,
		schema_version: c6EvidenceSchema,
		source_closure: sourceClosure,
		source_inputs: sourceInputs,
		surfaces: [
			{ adapter: "CLI", conclusion: "PHYSICAL_CLI_CLOSURE_EXERCISED", profile: "c5-cross-profile-parity", test_count: 4 },
			{ adapter: "HTTP", conclusion: "PHYSICAL_HTTP_BEHAVIOR_AND_PARITY_EXERCISED", profile: "c5-http-behavior+c5-cross-profile-parity", test_count: 5 },
			{ adapter: "NODE", conclusion: "GENERATED_NODE_PARITY_EXERCISED", profile: "c5-cross-profile-parity", test_count: 17 },
		],
	};
	return validateEvidenceDocument(evidence);
}

function wrapWords(text, prefix, continuation, width) {
	const words = safeText(text, "P07B_C6_RENDER_SOURCE", 4096).split(/ +/u);
	const lines = [];
	let current = prefix;
	let linePrefix = prefix;
	for (let word of words) {
		for (;;) {
			const separator = current === linePrefix ? "" : " ";
			const available = width - [...current].length - separator.length;
			if ([...word].length <= available) {
				current += `${separator}${word}`;
				break;
			}
			if (current !== linePrefix) {
				lines.push(current);
				linePrefix = continuation;
				current = continuation;
				continue;
			}
			const capacity = width - [...linePrefix].length;
			if (capacity < 1) fail("P07B_C6_RENDER_PREFIX", prefix);
			const codepoints = [...word];
			lines.push(`${linePrefix}${codepoints.slice(0, capacity).join("")}`);
			word = codepoints.slice(capacity).join("");
			linePrefix = continuation;
			current = continuation;
		}
	}
	if (current !== linePrefix) lines.push(current);
	return lines;
}

function formatDuration(elapsedMS) {
	const hours = Math.floor(elapsedMS / 3_600_000);
	const minutes = Math.floor((elapsedMS % 3_600_000) / 60_000);
	const seconds = Math.floor((elapsedMS % 60_000) / 1_000);
	const milliseconds = elapsedMS % 1_000;
	return `${hours}h ${minutes}m ${seconds}.${String(milliseconds).padStart(3, "0")}s`;
}

export function renderEvidence(evidence, summary, width) {
	validateEvidenceDocument(evidence);
	validateSummaryDocument(summary);
	if (!c6Widths.includes(width)) fail("P07B_C6_RENDER_WIDTH", String(width));
	const title = "COUNTERSHAPE C6A EVIDENCE";
	const lines = [title, "=".repeat(title.length), "", `STATE: ${evidence.artifact_state.state}`, ""];
	const paragraph = (text) => lines.push(...wrapWords(text, "", "", width), "");
	const bullet = (text) => lines.push(...wrapWords(text, "- ", "  ", width));
	paragraph("This source candidate closes evidence over exact sealed C5V. It adds no Countershape product command, product package, or external service call.");
	lines.push("EVIDENCE BOUNDARY", "-----------------");
	bullet(`${evidence.artifact_state.state} source candidate; do not treat this capture as standalone proof.`);
	bullet(`${evidence.didrun_findings.length} didrun findings are OPEN; dedicated private capture is ${summary.private_evidence.availability}.`);
	bullet(`Retained negative ${summary.permanent_negatives[0].id}: one C5V final attempt was operator-terminated (exit -15), and its failed ledger remains permanent history.`);
	paragraph("Execution authority covers measured front-door executables and the declared Go environment only. Security review, hostile containment, confidentiality, adoption, maintainership, production hardening, and human comprehension are not established.");
	lines.push("SEALED PARENT", "-------------");
	bullet(`Commit ${evidence.parent_evidence.commit.slice(0, 12)}; tree ${evidence.parent_evidence.tree.slice(0, 12)}.`);
	bullet(`${evidence.parent_evidence.claims.length} of ${evidence.parent_evidence.claims.length} claims are projected verbatim as TREE-EXACT from the sealed note.`);
	bullet(`Local HTML ${evidence.parent_evidence.html.sha256.slice(0, 12)} has ${evidence.parent_evidence.html.bytes} bytes; the ignored archive has ${evidence.parent_evidence.archive.session_event_count} events.`);
	lines.push("", "EXERCISED SURFACES", "------------------");
	for (const surface of evidence.surfaces) {
		bullet(`${surface.adapter}: ${surface.test_count} exact tests through ${surface.profile}.`);
		lines.push(...wrapWords(surface.conclusion, "  Result: ", "          ", width));
	}
	lines.push("", "EXACT PROFILE RUNS", "------------------");
	for (const profile of evidence.profile_runs) {
		const testCount = profile.targets.reduce((sum, target) => sum + target.tests.length, 0);
		bullet(`${profile.profile}: ${profile.passed} passed; ${profile.skipped} skipped.`);
		lines.push(...wrapWords(`${testCount} exact tests across ${profile.targets.length} package targets.`, "  Scope: ", "         ", width));
		lines.push(...wrapWords(profile.executable, "  Runtime: ", "           ", width));
		lines.push(...wrapWords(`${profile.working_directory}; empty base, fixed values, private run directories, and admitted-tool bindings.`, "  Context: ", "           ", width));
		const replay = profile.replay_policy.copy_paste_safe ? "Copy/paste-safe command." :
			"Not a copy/paste command. Exact argv is machine-readable in c6a-expert-evidence.json and intentionally omitted from this terminal diagnostic.";
		lines.push(...wrapWords(replay, "  Replay: ", "          ", width));
		lines.push(...wrapWords(`Exact roster; invocation digest ${profile.invocation_sha256.slice(0, 12)}.`, "  Proof: ", "         ", width));
	}
	lines.push("", "EXECUTION ADMISSION", "-------------------");
	bullet(`Admission ${evidence.admission_sha256.slice(0, 12)} binds ${evidence.execution_authority.tools.length} measured tools to ${evidence.source_closure.path_count} source inputs.`);
	bullet("Closure: hermetic Go dependency/test inventory plus explicit runtime inputs; the exact derivation identifier is in c6a-expert-evidence.json.");
	paragraph("Execution authority covers the measured front-door executables and declared Go environment only; complete Go toolchain, SDK, dynamic-library, and host provenance are not established.");
	lines.push("SEALED C5V VERIFIER TIMES", "-------------------------");
	for (let index = 0; index < summary.timings.length; index += 1) {
		bullet(`Pass ${index + 1}: ${formatDuration(summary.timings[index].elapsed_ms)} (${summary.timings[index].elapsed_ms} ms).`);
	}
	lines.push("", "EVIDENCE HEALTH", "---------------");
	for (const finding of evidence.didrun_findings) {
		bullet(`${finding.id} ${finding.status}, severity ${finding.severity}; scope: ${finding.scope}.`);
		lines.push(...wrapWords(finding.summary, "  Summary: ", "           ", width));
		lines.push(...wrapWords(finding.effect, "  Effect: ", "          ", width));
		lines.push(...wrapWords(`See ${finding.id} in docs/status/DIDRUN_BUGS.md.`, "  Detail: ", "          ", width));
	}
	bullet(`Retained permanent negative: ${summary.permanent_negatives[0].id}; operator-terminated C5V final attempt, exit -15.`);
	bullet(`Dedicated private capture: ${summary.private_evidence.availability}; ${summary.private_evidence.declared_blobs} blobs, ${summary.private_evidence.declared_bytes} bytes.`);
	paragraph("Tracked captures are sanitized; confidentiality is not established.");
	paragraph("The sealed parent's secrets_override=true records use of didrun's redacted-export override after scoped review. It does not establish secret absence, confidentiality, or permission to publish.");
	lines.push("SOURCE-EVIDENCE LIMITS", "----------------------");
	for (const nonclaim of summary.nonclaims) bullet(`Not claimed: ${nonclaim}.`);
	paragraph("The deterministic commands and bytes may receive didrun receipts. A blind critic's qualitative judgment remains UNRECEIPTED and has no Countershape semantic authority.");
	const rendered = `${lines.join("\n").replace(/\n+$/u, "")}\n`;
	validateRenderBytes(Buffer.from(rendered, "utf8"), width);
	return rendered;
}

export function validateRenderBytes(bytes, width) {
	if (!Buffer.isBuffer(bytes) || !c6Widths.includes(width) || bytes.length === 0 || bytes.length > 256 * 1024 ||
		bytes.at(-1) !== 0x0a || bytes.includes(0x0d)) fail("P07B_C6_RENDER_ENVELOPE", String(width));
	let text;
	try { text = new TextDecoder("utf-8", { fatal: true }).decode(bytes); }
	catch (error) { fail("P07B_C6_RENDER_UTF8", error.message); }
	if (/[\u0000-\u0009\u000b-\u001f\u007f-\u009f\u202a-\u202e\u2066-\u2069]/u.test(text) ||
		/\x1b\[/u.test(text)) fail("P07B_C6_RENDER_CONTROL", String(width));
	if (/(?:\/Users\/|\/private\/|[A-Za-z]:\\)/u.test(text)) fail("P07B_C6_RENDER_PRIVATE_PATH", String(width));
	if (/\b(?:Matches|Differs|Could not judge)\b/iu.test(text)) fail("P07B_C6_RENDER_P08_HEADLINE", String(width));
	if (/[^\x0a\x20-\x7e]/u.test(text)) fail("P07B_C6_RENDER_NON_ASCII", String(width));
	const lines = text.slice(0, -1).split("\n");
	if (lines.some((line) => line.length > width)) fail("P07B_C6_RENDER_OVERFLOW", String(width));
	return Object.freeze({ lines: lines.length, max: Math.max(...lines.map((line) => line.length)) });
}

export function expectedArtifactBytes(evidence, summary) {
	return Object.freeze(new Map([
		[c6ArtifactPaths.evidence, canonicalPretty(evidence)],
		[c6ArtifactPaths.summary, canonicalCompact(summary)],
		[c6ArtifactPaths.render60, Buffer.from(renderEvidence(evidence, summary, 60), "utf8")],
		[c6ArtifactPaths.render80, Buffer.from(renderEvidence(evidence, summary, 80), "utf8")],
		[c6ArtifactPaths.render120, Buffer.from(renderEvidence(evidence, summary, 120), "utf8")],
	]));
}

async function readRegular(root, relativePath, maximum = 8 * 1024 * 1024) {
	const absolute = await resolveNoFollow(root, relativePath, { kind: "file" });
	const before = await lstat(absolute);
	if (!before.isFile() || before.isSymbolicLink() || before.nlink !== 1 || (before.mode & 0o7777) !== 0o644 ||
		before.size < 1 || before.size > maximum) fail("P07B_C6_ARTIFACT_MODE", relativePath);
	const bytes = await readFile(absolute);
	const after = await lstat(absolute);
	if (!after.isFile() || after.isSymbolicLink() || before.dev !== after.dev || before.ino !== after.ino ||
		before.size !== after.size || before.mtimeMs !== after.mtimeMs) fail("P07B_C6_ARTIFACT_CHANGED", relativePath);
	return bytes;
}

export async function collectSourceInputs(root) {
	for (const path of c6AbsentSourceInputPaths) {
		const absolute = await resolveNoFollow(root, path, { allowMissingFinal: true });
		try {
			await lstat(absolute);
			fail("P07B_C6_SOURCE_INPUT_REQUIRED_ABSENCE", path);
		} catch (error) {
			if (error?.code !== "ENOENT") throw error;
		}
	}
	const descriptors = [];
	for (const path of c6SourceInputPaths) {
		const bytes = await readRegular(root, path, 32 * 1024 * 1024);
		descriptors.push({ bytes: bytes.length, mode: "100644", path, sha256: digest(bytes) });
	}
	validateSourceInputs(descriptors);
	return Object.freeze(descriptors.map((entry) => Object.freeze(entry)));
}

export async function validateCurrentSourceInputs(root, expected) {
	validateSourceInputs(expected);
	const current = await collectSourceInputs(root);
	if (!exact(current, expected)) fail("P07B_C6_SOURCE_INPUT_DRIFT", "tracked source descriptor changed");
	return current;
}

export async function readTrackedEvidence(root) {
	const evidenceBytes = await readRegular(root, c6ArtifactPaths.evidence);
	const summaryBytes = await readRegular(root, c6ArtifactPaths.summary);
	const evidence = validateEvidenceDocument(decodeCanonicalJSON(evidenceBytes, c6ArtifactPaths.evidence, "pretty"));
	const summary = validateSummaryDocument(decodeCanonicalJSON(summaryBytes, c6ArtifactPaths.summary, "compact"));
	await validateCurrentSourceInputs(root, evidence.source_inputs);
	const expected = expectedArtifactBytes(evidence, summary);
	const descriptors = [];
	for (const [path, bytes] of expected) {
		const actual = path === c6ArtifactPaths.evidence ? evidenceBytes :
			path === c6ArtifactPaths.summary ? summaryBytes : await readRegular(root, path, 256 * 1024);
		if (!actual.equals(bytes)) fail("P07B_C6_ARTIFACT_DRIFT", path);
		descriptors.push({ bytes: actual.length, path, sha256: digest(actual) });
	}
	const captureRoot = await resolveNoFollow(root, "docs/captures/p07b-c", { kind: "directory" });
	const entries = sorted(await readdir(captureRoot));
	if (!exact(entries, sorted([...expected.keys()].map((path) => path.slice("docs/captures/p07b-c/".length))))) {
		fail("P07B_C6_ARTIFACT_ROSTER", JSON.stringify(entries));
	}
	return Object.freeze({ descriptors: Object.freeze(descriptors), evidence, summary });
}

function validateProfileDescriptor(descriptor) {
	exactKeys(descriptor, ["arguments", "name", "targets"], "P07B_C6_PROFILE_DESCRIPTOR_KEYS");
	if (!c6SelectedProfiles.includes(descriptor.name) || !Array.isArray(descriptor.arguments) ||
		descriptor.arguments.length < 8 || descriptor.arguments.length > 256 || descriptor.arguments.some((argument) =>
			typeof argument !== "string" || argument.length === 0 || argument.length > 16_384 ||
			/[\u0000-\u001f\u007f]/u.test(argument) || argument.startsWith("/")) || !Array.isArray(descriptor.targets) ||
		descriptor.targets.length === 0) {
		fail("P07B_C6_PROFILE_DESCRIPTOR_VALUE", descriptor.name ?? "missing");
	}
	const packages = new Set();
	const argumentsSet = new Set(descriptor.arguments);
	for (const target of descriptor.targets) {
		exactKeys(target, ["packageArgument", "packagePath", "pass", "skip"],
			"P07B_C6_PROFILE_DESCRIPTOR_TARGET_KEYS");
		if (typeof target.packageArgument !== "string" || !target.packageArgument.startsWith("./") ||
			!argumentsSet.has(target.packageArgument) || typeof target.packagePath !== "string" ||
			!target.packagePath.startsWith("github.com/nelsonwerd/countershape/") || packages.has(target.packagePath) ||
			!Array.isArray(target.pass) || !Array.isArray(target.skip) || target.pass.length === 0 || target.skip.length !== 0 ||
			target.pass.length > 1024) {
			fail("P07B_C6_PROFILE_DESCRIPTOR_TARGET_VALUE", target.packagePath ?? "missing");
		}
		packages.add(target.packagePath);
		const tests = [...target.pass, ...target.skip];
		if (tests.some((name) => !/^(?:Fuzz|Test)[A-Za-z0-9_]+$/u.test(name)) ||
			new Set(tests).size !== tests.length) {
			fail("P07B_C6_PROFILE_DESCRIPTOR_TESTS", target.packagePath);
		}
	}
	return descriptor;
}

function decodeGoJSONLines(bytes) {
	if (!Buffer.isBuffer(bytes) || bytes.length === 0 || bytes.length > 64 * 1024 * 1024 ||
		bytes.at(-1) !== 0x0a || bytes.includes(0x0d) || bytes.includes(0x00)) {
		fail("P07B_C6_GO_JSON_ENVELOPE", bytes?.length ?? "not-buffer");
	}
	let source;
	try { source = new TextDecoder("utf-8", { fatal: true }).decode(bytes); }
	catch (error) { fail("P07B_C6_GO_JSON_UTF8", error.message); }
	const lines = source.slice(0, -1).split("\n");
	if (lines.length > 20_000 || lines.some((line) => line.length === 0 || Buffer.byteLength(line) > 1024 * 1024)) {
		fail("P07B_C6_GO_JSON_FRAME", "blank, oversized, or excessive line roster");
	}
	return lines.map((line, index) => {
		let event;
		try { event = JSON.parse(line); }
		catch (error) { fail("P07B_C6_GO_JSON_PARSE", `${index + 1}:${error.message}`); }
		return event;
	});
}

function validateTestName(name, expectedRoots, packagePath) {
	safeText(name, "P07B_C6_GO_JSON_TEST_NAME", 4096);
	const segments = name.split("/");
	if (segments.some((segment) => segment.length === 0) || !expectedRoots.has(segments[0])) {
		fail("P07B_C6_GO_JSON_TEST_ROSTER", `${packagePath}:${name}`);
	}
	return segments;
}

export function validateGoJSONTranscript(descriptor, bytes) {
	validateProfileDescriptor(descriptor);
	const events = decodeGoJSONLines(bytes);
	const targetsByPackage = new Map(descriptor.targets.map((target) => [target.packagePath, target]));
	const admittedActions = new Set(["cont", "fail", "output", "pass", "pause", "run", "skip", "start"]);
	for (const event of events) {
		if (!event || typeof event !== "object" || Array.isArray(event) || typeof event.Package !== "string" ||
			!targetsByPackage.has(event.Package) || typeof event.Action !== "string" ||
			!admittedActions.has(event.Action) || (event.Action === "output" && typeof event.Output !== "string") ||
			(Object.hasOwn(event, "Test") && typeof event.Test !== "string")) {
			fail("P07B_C6_GO_JSON_EVENT", "foreign or malformed event");
		}
		if (event.Action === "fail") fail("P07B_C6_GO_JSON_FAILURE", event.Package);
	}
	let passed = 0;
	let skipped = 0;
	for (const target of descriptor.targets) {
		const packageEvents = events.map((event, index) => ({ event, index })).filter((row) =>
			row.event.Package === target.packagePath);
		const packageRows = packageEvents.filter((row) => !Object.hasOwn(row.event, "Test"));
		const packageStarts = packageRows.filter((row) => row.event.Action === "start");
		const packagePasses = packageRows.filter((row) => row.event.Action === "pass");
		if (packageStarts.length !== 1 || packagePasses.length !== 1 || packageEvents[0]?.index !== packageStarts[0]?.index ||
			packageEvents.at(-1)?.index !== packagePasses[0]?.index || packageRows.some((row) =>
				!["output", "pass", "start"].includes(row.event.Action)) || packageRows.some((row) =>
				row.event.Action === "output" && (row.index <= packageStarts[0].index || row.index >= packagePasses[0].index))) {
			fail("P07B_C6_GO_JSON_PACKAGE_LIFECYCLE", target.packagePath);
		}
		const expectedRoots = new Set([...target.pass, ...target.skip]);
		const testRows = packageEvents.filter((row) => Object.hasOwn(row.event, "Test"));
		const byName = new Map();
		for (const row of testRows) {
			validateTestName(row.event.Test, expectedRoots, target.packagePath);
			if (["start"].includes(row.event.Action)) {
				fail("P07B_C6_GO_JSON_TEST_ACTION", `${target.packagePath}:${row.event.Test}:${row.event.Action}`);
			}
			const named = byName.get(row.event.Test) ?? [];
			named.push(row);
			byName.set(row.event.Test, named);
		}
		const observedRoots = sorted([...byName.keys()].filter((name) => !name.includes("/")));
		if (!exact(observedRoots, sorted(expectedRoots))) {
			fail("P07B_C6_GO_JSON_TEST_ROSTER", `${target.packagePath}:${JSON.stringify(observedRoots)}`);
		}
		const lifecycle = new Map();
		for (const [name, rows] of byName) {
			let state = "BEFORE_RUN";
			let runIndex = -1;
			let terminalIndex = -1;
			let terminalAction = null;
			let pauseCount = 0;
			let continueCount = 0;
			for (const row of rows) {
				switch (row.event.Action) {
				case "run":
					if (state !== "BEFORE_RUN") fail("P07B_C6_GO_JSON_TEST_LIFECYCLE", `${target.packagePath}:${name}:run`);
					state = "LIVE";
					runIndex = row.index;
					break;
				case "pause":
					if (state !== "LIVE") fail("P07B_C6_GO_JSON_TEST_LIFECYCLE", `${target.packagePath}:${name}:pause`);
					pauseCount += 1;
					state = "PAUSED";
					break;
				case "cont":
					if (state !== "PAUSED") fail("P07B_C6_GO_JSON_TEST_LIFECYCLE", `${target.packagePath}:${name}:cont`);
					continueCount += 1;
					state = "LIVE";
					break;
				case "output":
					if (state !== "LIVE") fail("P07B_C6_GO_JSON_TEST_OUTPUT_STATE", `${target.packagePath}:${name}`);
					break;
				case "pass":
				case "skip":
					if (state !== "LIVE") fail("P07B_C6_GO_JSON_TEST_LIFECYCLE", `${target.packagePath}:${name}:terminal`);
					state = "TERMINAL";
					terminalIndex = row.index;
					terminalAction = row.event.Action;
					break;
				default:
					fail("P07B_C6_GO_JSON_TEST_ACTION", `${target.packagePath}:${name}:${row.event.Action}`);
				}
			}
			const nested = name.includes("/");
			const expectedTerminal = nested || target.pass.includes(name) ? "pass" : "skip";
			if (state !== "TERMINAL" || runIndex < 0 || terminalIndex <= runIndex || terminalAction !== expectedTerminal ||
				(nested && terminalAction !== "pass") || terminalIndex >= packagePasses[0].index ||
				!((pauseCount === 0 && continueCount === 0) || (pauseCount === 1 && continueCount === 1))) {
				fail("P07B_C6_GO_JSON_TEST_RESULT", `${target.packagePath}:${name}:${terminalAction ?? "none"}`);
			}
			lifecycle.set(name, { runIndex, terminalIndex });
		}
		for (const [name, state] of lifecycle) {
			if (!name.includes("/")) continue;
			const rootName = name.slice(0, name.indexOf("/"));
			const root = lifecycle.get(rootName);
			// A slash may be part of one t.Run name and the stream carries no parent
			// id. Require the admitted top-level root to enclose this lifecycle, but
			// never invent lifecycle nodes for textual path segments.
			if (root === undefined || root.runIndex >= state.runIndex || state.terminalIndex >= root.terminalIndex) {
				fail("P07B_C6_GO_JSON_DESCENDANT_ORDER", `${target.packagePath}:${name}`);
			}
		}
		passed += target.pass.length;
		skipped += target.skip.length;
	}
	return Object.freeze({ passed, profile: descriptor.name, skipped });
}

export function assertProfileRunsMatchDescriptors(profileRuns, descriptors) {
	if (!Array.isArray(descriptors) || descriptors.length !== c6SelectedProfiles.length) {
		fail("P07B_C6_PROFILE_DESCRIPTOR_CARDINALITY", JSON.stringify(descriptors));
	}
	for (let index = 0; index < descriptors.length; index += 1) {
		const descriptor = descriptors[index];
		const run = profileRuns[index];
		validateProfileDescriptor(descriptor);
		if (descriptor.name !== c6SelectedProfiles[index] || descriptor.name !== run.profile ||
			!exact(descriptor.arguments, run.arguments) || digest(canonicalCompact(descriptor.arguments)) !== run.arguments_sha256 ||
			descriptor.targets.length !== run.targets.length) fail("P07B_C6_PROFILE_DESCRIPTOR", String(index));
		for (let targetIndex = 0; targetIndex < descriptor.targets.length; targetIndex += 1) {
			const source = descriptor.targets[targetIndex];
			const captured = run.targets[targetIndex];
			if (source.packageArgument !== captured.package_argument || source.packagePath !== captured.package_path ||
				!exact(sorted(source.pass), captured.tests) || source.skip.length !== 0) {
				fail("P07B_C6_PROFILE_TARGET_DESCRIPTOR", `${index}:${targetIndex}`);
			}
		}
	}
	return true;
}

function syntheticExecutionAuthority() {
	const tools = [
		["cc", "/usr/bin/clang"],
		["cxx", "/usr/bin/clang++"],
		["git", "/usr/bin/git"],
		["go", "/opt/homebrew/bin/go"],
		["node", "/opt/homebrew/bin/node"],
		["sh", "/bin/sh"],
	].map(([name, path], index) => ({
		bytes: index + 1,
		name,
		path,
		sha256: String(index).repeat(64),
	}));
	const authority = {
		authority_scope: "FRONT_DOOR_EXECUTABLES_AND_DECLARED_GO_ENVIRONMENT_ONLY",
		environment_contract: clone(c6ExecutionEnvironmentContract),
		go_env: { goarch: "arm64", goos: "darwin", goroot: "/opt/homebrew/opt/go/libexec", goversion: "go1.26.0" },
		tools,
		version_output: "go version go1.26.0 darwin/arm64",
	};
	validateExecutionAuthority(authority);
	return authority;
}

function syntheticProfiles(executionAuthority) {
	return c6ProfileDescriptors.map((descriptor) => buildProfileRun(descriptor, {
		passed: descriptor.targets.reduce((total, target) => total + target.pass.length, 0),
		profile: descriptor.name,
		skipped: 0,
	}, executionAuthority));
}

function runTranscriptSelfTests() {
	const packagePath = "github.com/nelsonwerd/countershape/testkit/contractexec/http";
	const rootTest = "TestNestedLifecycle";
	const childTest = `${rootTest}/child`;
	const grandchildTest = `${childTest}/branch/leaf`;
	const descriptor = {
		arguments: [
			"test", "-mod=readonly", "-buildvcs=false", "-p=1", "-count=1", "-json", "-run",
			`^(?:${rootTest})$`, "./testkit/contractexec/http",
		],
		name: "c5-http-behavior",
		targets: [{
			packageArgument: "./testkit/contractexec/http",
			packagePath,
			pass: [rootTest],
			skip: [],
		}],
	};
	const clean = [
		{ Action: "start", Package: packagePath },
		{ Action: "run", Package: packagePath, Test: rootTest },
		{ Action: "output", Output: "root live\n", Package: packagePath, Test: rootTest },
		{ Action: "run", Package: packagePath, Test: childTest },
		{ Action: "output", Output: "child live\n", Package: packagePath, Test: childTest },
		{ Action: "run", Package: packagePath, Test: grandchildTest },
		{ Action: "pass", Package: packagePath, Test: grandchildTest },
		{ Action: "pass", Package: packagePath, Test: childTest },
		{ Action: "pass", Package: packagePath, Test: rootTest },
		{ Action: "output", Output: "ok\n", Package: packagePath },
		{ Action: "pass", Package: packagePath },
	];
	const encode = (events) => Buffer.from(`${events.map((event) => JSON.stringify(event)).join("\n")}\n`, "utf8");
	const result = validateGoJSONTranscript(descriptor, encode(clean));
	if (!exact(result, { passed: 1, profile: "c5-http-behavior", skipped: 0 })) {
		fail("P07B_C6_SELFTEST_GO_JSON_RESULT", JSON.stringify(result));
	}
	let cases = 1;
	const rejects = (id, events, pattern) => {
		try { validateGoJSONTranscript(descriptor, encode(events)); }
		catch (error) {
			if (!pattern.test(error.message)) throw error;
			cases += 1;
			return;
		}
		fail("P07B_C6_SELFTEST_GO_JSON_FALSE_NEGATIVE", id);
	};
	const nestedSkip = clone(clean);
	nestedSkip[7].Action = "skip";
	rejects("nested-skip", nestedSkip, /P07B_C6_GO_JSON_TEST_RESULT/u);
	const parentTerminalFirst = clone(clean);
	[parentTerminalFirst[7], parentTerminalFirst[8]] = [parentTerminalFirst[8], parentTerminalFirst[7]];
	rejects("descendant-after-parent-terminal", parentTerminalFirst, /P07B_C6_GO_JSON_DESCENDANT_ORDER/u);
	const childRunFirst = [clean[0], clean[3], clean[1], clean[2], ...clean.slice(4)];
	rejects("descendant-before-parent-run", childRunFirst, /P07B_C6_GO_JSON_DESCENDANT_ORDER/u);
	const outputBeforeRun = clone(clean);
	[outputBeforeRun[1], outputBeforeRun[2]] = [outputBeforeRun[2], outputBeforeRun[1]];
	rejects("output-before-run", outputBeforeRun, /P07B_C6_GO_JSON_TEST_OUTPUT_STATE/u);
	const outputAfterTerminal = clone(clean);
	outputAfterTerminal.splice(9, 0, {
		Action: "output", Output: "late\n", Package: packagePath, Test: rootTest,
	});
	rejects("output-after-terminal", outputAfterTerminal, /P07B_C6_GO_JSON_TEST_OUTPUT_STATE/u);
	const slashBearingDirectChild = clean.filter((event) => event.Test !== childTest);
	const directResult = validateGoJSONTranscript(descriptor, encode(slashBearingDirectChild));
	if (!exact(directResult, { passed: 1, profile: "c5-http-behavior", skipped: 0 })) {
		fail("P07B_C6_SELFTEST_GO_JSON_RESULT", JSON.stringify(directResult));
	}
	cases += 1;
	const duplicateNestedRun = clone(clean);
	duplicateNestedRun.splice(6, 0, { Action: "run", Package: packagePath, Test: grandchildTest });
	rejects("duplicate-nested-run", duplicateNestedRun, /P07B_C6_GO_JSON_TEST_LIFECYCLE/u);
	const packagePassEarly = clone(clean);
	packagePassEarly.splice(6, 0, { Action: "pass", Package: packagePath });
	rejects("package-pass-before-terminals", packagePassEarly, /P07B_C6_GO_JSON_PACKAGE_LIFECYCLE/u);
	const nestedFail = clone(clean);
	nestedFail[6].Action = "fail";
	rejects("nested-fail", nestedFail, /P07B_C6_GO_JSON_FAILURE/u);
	const pausedOutput = clone(clean);
	pausedOutput.splice(4, 0,
		{ Action: "pause", Package: packagePath, Test: childTest },
		{ Action: "output", Output: "not live\n", Package: packagePath, Test: childTest },
		{ Action: "cont", Package: packagePath, Test: childTest });
	rejects("output-while-paused", pausedOutput, /P07B_C6_GO_JSON_TEST_OUTPUT_STATE/u);
	const repeatedPause = clone(clean);
	repeatedPause.splice(4, 0,
		{ Action: "pause", Package: packagePath, Test: childTest },
		{ Action: "cont", Package: packagePath, Test: childTest },
		{ Action: "pause", Package: packagePath, Test: childTest },
		{ Action: "cont", Package: packagePath, Test: childTest });
	rejects("repeated-pause-cycle", repeatedPause, /P07B_C6_GO_JSON_TEST_RESULT/u);
	return cases;
}

export function runPureSelfTest() {
	const executionAuthority = syntheticExecutionAuthority();
	const sourceClosure = buildSourceClosure();
	const parent = {
		archive: {
			file_count: c5vAuthority.archive_file_count,
			manifest_sha256: c5vAuthority.archive_manifest_sha256,
			object_count: c5vAuthority.archive_object_count,
			path: c5vAuthority.ledger_path,
			session_event_count: c5vAuthority.archive_session_event_count,
			total_bytes: c5vAuthority.archive_total_bytes,
		},
		claims: c5vClaims.map((claim, index) => ({ grade: "TREE-EXACT", label: claim.label, supporting_event_index: index, type: claim.type })),
		commit: c5vAuthority.commit,
		html: { bytes: c5vAuthority.html_bytes, path: c5vAuthority.html_path, sha256: c5vAuthority.html_sha256 },
		note_blob: c5vAuthority.note_blob,
		note_body_sha256: c5vAuthority.note_body_sha256,
		parent: c5vAuthority.parent,
		secrets_override: true,
		strict_grade_projection: "ALL_TREE_EXACT_FROM_SEALED_NOTE",
		subject: c5vAuthority.subject,
		tree: c5vAuthority.tree,
	};
	const evidence = buildEvidenceDocument(parent, syntheticProfiles(executionAuthority), c6SourceInputPaths.map((path, index) => ({
		bytes: index + 1,
		mode: "100644",
		path,
		sha256: String((index % 10)).repeat(64),
	})), { executionAuthority, sourceClosure });
	const summary = buildSummary([1, 2, 3]);
	let cases = runTranscriptSelfTests();
	const rejects = (mutate, pattern) => {
		const candidate = clone(evidence);
		mutate(candidate);
		try { validateEvidenceDocument(candidate); }
		catch (error) {
			if (!pattern.test(error.message)) throw error;
			cases += 1;
			return;
		}
		fail("P07B_C6_SELFTEST_FALSE_NEGATIVE", String(pattern));
	};
	rejects((candidate) => { candidate.extra = true; }, /P07B_C6_EVIDENCE_KEYS/u);
	rejects((candidate) => { candidate.artifact_state.state = "TREE-EXACT"; }, /P07B_C6_EVIDENCE_IDENTITY/u);
	rejects((candidate) => { candidate.profile_runs.reverse(); }, /P07B_C6_PROFILE_ORDER/u);
	rejects((candidate) => { candidate.profile_runs[0].targets[1].tests[1] = candidate.profile_runs[0].targets[1].tests[0]; }, /P07B_C6_TARGET_VALUE/u);
	rejects((candidate) => { candidate.profile_runs[1].passed = 3; }, /P07B_C6_PROFILE_CARDINALITY/u);
	rejects((candidate) => { candidate.admission_sha256 = "0".repeat(64); }, /P07B_C6_ADMISSION_DIGEST/u);
	rejects((candidate) => { candidate.execution_authority.environment_contract.base = "INHERITED"; }, /P07B_C6_ENVIRONMENT_VALUE/u);
	rejects((candidate) => { candidate.source_closure.path_count += 1; }, /P07B_C6_SOURCE_CLOSURE_VALUE/u);
	rejects((candidate) => { candidate.source_inputs[0].sha256 = "f".repeat(64); }, /P07B_C6_ADMISSION_DIGEST/u);
	rejects((candidate) => { candidate.profile_runs[0].targets[0].tests[0] = "TestValidButSubstitutedName"; }, /P07B_C6_TARGET_VALUE/u);
	rejects((candidate) => { candidate.profile_runs[0].invocation.shift(); }, /P07B_C6_PROFILE_VALUE/u);
	rejects((candidate) => { candidate.profile_runs[0].replay_policy.copy_paste_safe = true; }, /P07B_C6_PROFILE_VALUE/u);
	rejects((candidate) => { candidate.surfaces[0].test_count = 5; }, /P07B_C6_SURFACE_VALUE/u);
	rejects((candidate) => { candidate.parent_evidence.claims[0].grade = "RECORDED"; }, /P07B_C6_PARENT_CLAIM/u);
	const badSummary = clone(summary);
	badSummary.nonclaims.reverse();
	try { validateSummaryDocument(badSummary); }
	catch (error) {
		if (!/P07B_C6_SUMMARY_FIXED_FIELDS/u.test(error.message)) throw error;
		cases += 1;
	}
	const badSummaryState = clone(summary);
	badSummaryState.private_evidence.note = "C6A declares no dedicated private capture blobs.";
	try { validateSummaryDocument(badSummaryState); }
	catch (error) {
		if (!/P07B_C6_PRIVATE_VALUE/u.test(error.message)) throw error;
		cases += 1;
	}
	rejects((candidate) => { delete candidate.didrun_findings[0].effect; }, /P07B_C6_DIDRUN_FINDINGS/u);
	for (const [text, pattern] of [
		["\u001b[31mred\n", /P07B_C6_RENDER_CONTROL/u],
		["/Users/example/private\n", /P07B_C6_RENDER_PRIVATE_PATH/u],
		["Matches\n", /P07B_C6_RENDER_P08_HEADLINE/u],
		[`${"x".repeat(61)}\n`, /P07B_C6_RENDER_OVERFLOW/u],
		["wide \u754c\n", /P07B_C6_RENDER_NON_ASCII/u],
	]) {
		try { validateRenderBytes(Buffer.from(text, "utf8"), 60); }
		catch (error) {
			if (!pattern.test(error.message)) throw error;
			cases += 1;
		}
	}
	for (const width of c6Widths) {
		const left = renderEvidence(evidence, summary, width);
		const right = renderEvidence(evidence, summary, width);
		if (left !== right) fail("P07B_C6_SELFTEST_NONDETERMINISTIC", String(width));
		const normalized = left.replace(/\s+/gu, " ");
		if (!normalized.includes("does not establish secret absence, confidentiality, or permission to publish.")) {
			fail("P07B_C6_SELFTEST_OVERRIDE_SEMANTICS", String(width));
		}
		if (left.includes("  Command:") || !normalized.includes("Not a copy/paste command. Exact argv is machine-readable") ||
			!normalized.includes("STATE: UNRECEIPTED") || !normalized.includes("UNRECEIPTED source candidate") ||
			!normalized.includes("C5V_FINAL_ATTEMPT_OPERATOR_TERMINATED") || !normalized.includes("exit -15") ||
			!normalized.includes("See S6-01 in docs/status/DIDRUN_BUGS.md.") ||
			!normalized.includes("Closure: hermetic Go dependency/test inventory plus explicit runtime inputs") ||
			left.indexOf("\nEVIDENCE BOUNDARY\n") < 0 ||
			left.indexOf("\nEVIDENCE BOUNDARY\n") > left.indexOf("\nEXACT PROFILE RUNS\n")) {
			fail("P07B_C6_SELFTEST_TERMINAL_HIERARCHY", String(width));
		}
		cases += 1;
	}
	return cases;
}

export async function describeDirectory(root, relativePath) {
	const absolute = await resolveNoFollow(root, relativePath, { kind: "directory" });
	const rootInfo = await lstat(absolute);
	if (!rootInfo.isDirectory() || rootInfo.isSymbolicLink()) fail("P07B_C6_ARCHIVE_ROOT", relativePath);
	const rows = [];
	let objectCount = 0;
	let totalBytes = 0;
	const visit = async (directory, prefix = "") => {
		const entries = (await readdir(directory, { withFileTypes: true })).sort((left, right) =>
			Buffer.compare(Buffer.from(left.name), Buffer.from(right.name)));
		for (const entry of entries) {
			const relative = prefix === "" ? entry.name : `${prefix}/${entry.name}`;
			const path = join(directory, entry.name);
			const info = await lstat(path);
			if (info.isSymbolicLink()) fail("P07B_C6_ARCHIVE_SYMLINK", relative);
			if (info.isDirectory()) {
				await visit(path, relative);
				continue;
			}
			if (!info.isFile() || info.nlink !== 1 || info.size > 128 * 1024 * 1024) fail("P07B_C6_ARCHIVE_FILE", relative);
			const bytes = await readFile(path);
			rows.push(`${relative}\t${bytes.length}\t${digest(bytes)}\n`);
			totalBytes += bytes.length;
			if (relative.startsWith("objects/")) objectCount += 1;
			if (rows.length > 10_000 || totalBytes > 1024 * 1024 * 1024) fail("P07B_C6_ARCHIVE_BOUND", relative);
		}
	};
	await visit(absolute);
	return Object.freeze({
		file_count: rows.length,
		manifest_sha256: digest(Buffer.from(rows.join(""), "utf8")),
		object_count: objectCount,
		path: relativePath,
		total_bytes: totalBytes,
	});
}
