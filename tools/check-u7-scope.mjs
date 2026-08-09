#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { constants } from "node:fs";
import { lstat, open, readdir, realpath } from "node:fs/promises";
import { createHash } from "node:crypto";
import { isAbsolute, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { isDeepStrictEqual } from "node:util";

const repositoryRoot = resolve(fileURLToPath(new URL("..", import.meta.url)));
const specificationPath = resolve(repositoryRoot, "spec/verification/u7-unit-paths.json");
const receiptPath = "spec/verification/u7-receipt.json";
const unitOrder = Object.freeze(["U7P", "U7A", "U7B", "U7C", "U7D", "U7R"]);
const u7pExactPaths = Object.freeze([
	"docs/ARCHITECTURE.md",
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/STATE_MACHINES.md",
	"docs/VERIFICATION.md",
	"docs/prompts/P08-U7-CLI-REFERENCE-STUDIES.md",
	"docs/status/U7P-AUTHORITY.md",
	"spec/verification/u7-unit-paths.json",
	"tools/check-u7-plan.mjs",
	"tools/check-u7-scope.mjs",
	"tools/check-u7-architecture.mjs",
	"tools/check-u7-architecture-selftest.mjs",
	"tools/check-u7-study-harness.mjs",
	"tools/check-u7-study-harness-selftest.mjs",
	"tools/print-u7-final-runbook.mjs",
	"tools/verify-current.mjs",
	"tools/verify-current-selftest.mjs",
]);
const exactC6BCommit = "4cef12b38cfcd857593a21db5952f2dfb2dfc274";
const exactSealedParent = Object.freeze({
	boundary: "C6B", commit: exactC6BCommit, tree: "b50aee79f41c3f498524d89d425d6dd8f4009702",
	parent: "fafff150d23d6df211b4313e73ac7cf43f28b2d3", subject: "docs: receipt P07B-C contract execution",
	note_ref: "refs/notes/didrun", note_blob: "a9f3f5484caf00f35e254f71549a754d3de3bd4b",
	note_body_sha256: "355355cd57254802dbc661c4fc9faaf8c695dfbbc787c0175b085f2709f532b3",
	claim_count: 9, claim_coverage: "9/9", grade: "tree-exact", strict_exit: 0, secrets_override: true,
	html_path: ".countershape/evidence/p07b-c-c6b-final-4cef12b38cfc.html",
	html_sha256: "006e298a8cd8556bb32cee4229df60e3735b102a96c54878c3cf2205cbb48070",
	html_authority: "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS",
	ledger_archive: ".didrun-history/p07b-c-c6b-final-4cef12b38cfc/.didrun",
	ledger_authority: "LOCAL_SECRET_BEARING_SNAPSHOT_NOT_PORTABLE", ledger_file_count: 14, ledger_bytes: 38492,
	ledger_manifest_sha256: "a09598654ee38382ea44201a7b4609b66c1bee192d1dcd8fc747508527122e37",
	ledger_event_count: 10, ledger_claim_count: 9, ledger_seal_count: 1,
});
const exactTransitionAuthority = Object.freeze({
	source: "OWNER_OUT_OF_BAND",
	classification: "OWNER_AUTHORIZED_ROADMAP_ACTIVATION",
	provenance: Object.freeze({
		kind: "UNEVIDENCED",
		disclosure: "OWNER_ATTRIBUTED_SESSION_INSTRUCTION_ONLY_NO_QUALIFYING_PREEXISTING_ARTIFACT",
	}),
	authentication: "NOT_ESTABLISHED",
	signed_authorization: "NOT_IMPLEMENTED",
	predecessor_declaration: Object.freeze({
		kind: "PROSE_ONLY",
		artifact_path: "docs/prompts/P08-U7-CLI-REFERENCE-STUDIES.md",
		artifact_blob: "d89d7b677ca0e924746231a9cbb732969199c359",
		artifact_sha256: "sha256:bd812bed4a72bfbc3ab7b773827ec1f14e87d8e9544ff87116e1ceb1221c6653",
		artifact_bytes: 11987,
		artifact_mode: "100644",
		artifact_role: "PREDECESSOR_ROADMAP_DIRECTION_ONLY_NOT_EXACT_U7_TOPOLOGY",
		preexistence: "DIRECT_PARENT_TREE",
	}),
	product_authority: "NONE", product_behavior: "INHERITED_UNREPROVEN", grade_transfer: "NONE",
});
const exactRuntimeAuthority = Object.freeze({
	platform: "darwin", architecture: "arm64", node_path: "/opt/homebrew/bin/node", node_major: 25,
	go_path: "/opt/homebrew/bin/go", git_path: "/usr/bin/git",
	admitted_tools: Object.freeze([
		Object.freeze({ name: "clang", path: "/usr/bin/clang", realpath: "/usr/bin/clang", sha256: "179301dcb41ea78accc3fa0048a7e6f6710d891945a751a34addd622020c1818", version: "Apple clang version 17.0.0 (clang-1700.6.3.2)" }),
		Object.freeze({ name: "clang++", path: "/usr/bin/clang++", realpath: "/usr/bin/clang++", sha256: "179301dcb41ea78accc3fa0048a7e6f6710d891945a751a34addd622020c1818", version: "Apple clang version 17.0.0 (clang-1700.6.3.2)" }),
		Object.freeze({ name: "env", path: "/usr/bin/env", realpath: "/usr/bin/env", sha256: "6e506aec3c0cff703ac1e66cedc6f1945354ad41339a38db4425c7c88227128f", version: "NO_STABLE_VERSION_INTERFACE" }),
		Object.freeze({ name: "git", path: "/usr/bin/git", realpath: "/usr/bin/git", sha256: "179301dcb41ea78accc3fa0048a7e6f6710d891945a751a34addd622020c1818", version: "git version 2.50.1 (Apple Git-155)" }),
		Object.freeze({ name: "go", path: "/opt/homebrew/bin/go", realpath: "/opt/homebrew/Cellar/go/1.26.5/libexec/bin/go", sha256: "3f947495f00cb7f8088a5cfd694da8dc43869b33f5e7377b048fb18922ffb7e0", version: "go version go1.26.5 darwin/arm64" }),
		Object.freeze({ name: "gofmt", path: "/opt/homebrew/bin/gofmt", realpath: "/opt/homebrew/Cellar/go/1.26.5/libexec/bin/gofmt", sha256: "fb5bc1241a2e1a218afb069d7ee5e4269c3fe918a38fc03190fbc9cbb2bb69b1", version: "go1.26.5-toolchain" }),
		Object.freeze({ name: "node", path: "/opt/homebrew/bin/node", realpath: "/opt/homebrew/Cellar/node/25.2.1/bin/node", sha256: "87989003817c5347d6bad48e46897e1b6509328bb7e8295d83cb6b40af836b9c", version: "v25.2.1" }),
		Object.freeze({ name: "sh", path: "/bin/sh", realpath: "/bin/sh", sha256: "ad5c194b05f83bc5e793c1cd67b148a4b680467b5a5730ab1a31fe4e6460ee9f", version: "GNU bash, version 3.2.57(1)-release (arm64-apple-darwin25)" }),
		Object.freeze({ name: "time", path: "/usr/bin/time", realpath: "/usr/bin/time", sha256: "d2210b72e8c978748a0f0ac7d0819dda4dd411bb7a2e9730476038b0d695e7a2", version: "NO_STABLE_VERSION_INTERFACE" }),
	]),
	tool_byte_authority: "LOCAL_ENTRYPOINT_SNAPSHOT_DYNAMIC_LIBRARIES_SDK_AND_TRANSITIVE_DEPENDENCIES_UNBOUND",
	didrun_path: "/opt/homebrew/bin/didrun",
	didrun_version: "didrun 0.1.0", didrun_entrypoint_realpath: "/Users/drewnelson/.venvs/didrun/bin/didrun",
	didrun_entrypoint_shim_sha256: "7cede7d470d120011dacf0e0172c1651db2c5aadfe2143215d691b7f380fb29f",
	didrun_python_entrypoint: "/Users/drewnelson/.venvs/didrun/bin/python",
	didrun_python_realpath: "/opt/homebrew/Cellar/python@3.14/3.14.5/Frameworks/Python.framework/Versions/3.14/bin/python3.14",
	didrun_python_launcher_sha256: "2477b47fa3ae65b9574eb18a15edb364e96948eaa1875ad3f1c80d780efc9c12",
	didrun_pyvenv_config_path: "/Users/drewnelson/.venvs/didrun/pyvenv.cfg",
	didrun_pyvenv_config_bytes: 300,
	didrun_pyvenv_config_sha256: "3569f574a4bda54d4292ecec2bbee396d27fa0ac71b616277e012ece75bb8af3",
	didrun_include_system_site_packages: false,
	didrun_import_route_path: "/Users/drewnelson/.venvs/didrun/lib/python3.14/site-packages/__editable__.didrun-0.0.1.pth",
	didrun_import_route_bytes: 48, didrun_import_route_sha256: "26ee19b135910255e217cf3aa5ad26fcf05dfdc59618de980a0f9facff99fac8",
	didrun_site_packages_root: "/Users/drewnelson/.venvs/didrun/lib/python3.14/site-packages",
	didrun_startup_manifest_policy: "EXACT_TOP_LEVEL_PTH_USER_SITE_DISABLED_CUSTOMIZER_FILE_OR_PACKAGE_ABSENT_IN_SITE_PACKAGES_AND_EDITABLE_SOURCE_ROOT",
	didrun_startup_file_count: 1, didrun_startup_bytes: 48,
	didrun_startup_manifest_sha256: "98b135f640be1e0ed3626ca54c01fd86fbdf1ab3c97023fc1b325b8ce215fb70",
	didrun_outer_environment_policy: "ENV_I_UNIT_FINAL_ROOT_HOME_TMP_EXACT_STATIC_ASSIGNMENTS",
	didrun_outer_dynamic_roots: "UNIT_FINAL_ROOT_HOME_AND_TMPDIR_0700",
	didrun_outer_static_environment: Object.freeze(["PATH=/usr/bin:/bin:/opt/homebrew/bin", "LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "NODE_OPTIONS=", "NODE_PATH=", "PYTHONPATH=", "PYTHONHOME=", "PYTHONNOUSERSITE=1", "PYTHONSAFEPATH=1", "PYTHONDONTWRITEBYTECODE=1", "PYTHONHASHSEED=0", "PYTHONUTF8=1", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_LAZY_FETCH=1", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0", "GIT_NO_REPLACE_OBJECTS=1"]),
	didrun_outer_git_path: "/usr/bin/git",
	didrun_repository_git_config_path: ".git/config",
	didrun_repository_git_config_bytes: 360,
	didrun_repository_git_config_sha256: "266394048a1b4279d0101e568cea7d7536e5d6a420e350bf7bfb5634c8e1ce73",
	didrun_repository_replace_refs: "ABSENT",
	didrun_package_root: "/Users/drewnelson/autopilot-dev-stress-test/src/didrun",
	didrun_package_manifest_policy: "RECURSIVE_REGULAR_EXCEPT_INACTIVE_CPYTHON_PYC", didrun_package_file_count: 18,
	didrun_package_bytes: 195116, didrun_package_manifest_sha256: "7602c7a848d68998200ebd0f2f0ab27851fe7e8fc2ceebd27f2983f9a28e5f42",
	didrun_byte_authority: "LOCAL_RUNTIME_SNAPSHOT_NOT_VENDOR_AUTHENTICATED_OUTER_PROCESS_ENVIRONMENT_STDLIB_DYNAMIC_LOADER_AND_THIRD_PARTY_TRANSITIVE_DEPENDENCIES_UNBOUND",
});
const exactInheritedReceipts = Object.freeze({ C3P: "PRESENT", C3: "PRESENT", C6A: "PRESENT" });
const exactReceiptContract = Object.freeze(JSON.parse(String.raw`{"schema_version":"countershape/u7-receipt/v1","source_boundary":"U7D","study_event_command":["/opt/homebrew/bin/node","tools/check-u7-study-harness.mjs","--phase","U7D"],"study_evidence_schema":"countershape/u7-study-evidence/v1","study_execution_authority":"U7P_FROZEN_HARNESS_DIRECT_PROCESS_GIT_AND_ARTIFACT_OBSERVATION","study_harness":{"protocol":"countershape/u7-study-harness/v1","protocol_sha256":"faca5cac53ce4ca4d4edea3089edd1fad1139ff847da9618115d1e308b717678","path":"tools/check-u7-study-harness.mjs","sha256":"d6a39e44e6a4de43268dab9ff2deac8350b0b4e107b01e254d2767f0692a4121","product_result_schema":"countershape/u7-study-domain-result/v1","trial_schema":"countershape/u7-study-trial/v1","artifact_schema":"countershape/u7-study-artifact/v1","observation_authority":"U7P_FROZEN_HARNESS_DIRECT_PROCESS_GIT_AND_ARTIFACT_OBSERVATION","semantic_ceiling":"ARTIFACT_BYTES_AND_SUBJECT_PROCESS_TOPOLOGY_OBSERVED_PRODUCT_SEMANTICS_AND_FULL_HARNESS_RESOURCES_NOT_INDEPENDENTLY_ESTABLISHED","driver_protocol":"FIXTURE_ONLY_NO_EVIDENCE_ROOT","phase_verdicts":{"U7B":"LOCAL_HTTP_REFERENCE_FUNCTIONAL_GREEN_SUBJECT_RESOURCE_OBSERVED","U7C":"LOCAL_CLI_REFERENCE_FUNCTIONAL_GREEN_SUBJECT_RESOURCE_OBSERVED","U7D":"LOCAL_REFERENCE_MILESTONE_FUNCTIONAL_GREEN_FULL_STUDY_RESOURCE_UNRECEIPTED"},"observer_prefix":["/usr/bin/time","-p","-l","-o"],"driver_by_domain":{"http":"tools/run-u7-http-study.mjs","cli":"tools/run-u7-cli-study.mjs"},"phase_budgets":[{"id":"fixture","per_run_trials":1},{"id":"compile","per_run_trials":1},{"id":"search","per_run_trials":80},{"id":"confirm","per_run_trials":8},{"id":"contract","per_run_trials":10}],"deterministic_artifact_paths":{"source_spec_sha256":"deterministic/source-spec.json","world_plan_sha256":"deterministic/world-plan.json","ruling_sha256":"deterministic/ruling.json","decision_record_sha256":"deterministic/decision-record.json","contract_bundle_sha256":"deterministic/contract-bundle.json"},"fresh_artifact_paths":{"world_instance_sha256":"fresh/world-instance.json","attempts_sha256":"fresh/attempts.json","measurements_sha256":"fresh/measurements.json","captures_sha256":"fresh/captures.json","confirmation_sha256":"fresh/confirmation.json","contract_execution_target_sha256":"fresh/contract-execution-target.json","finalized_contract_run_sha256":"fresh/finalized-contract-run.json","contract_execution_sha256":"fresh/contract-execution.json"}},"study_domains":["http","cli"],"study_run_count_per_domain":3,"study_trial_budget_per_domain":300,"study_subject_process_wall_time_budget_ms_per_domain":900000,"study_subject_process_peak_rss_budget_bytes_per_domain":4294967296,"study_phase_budgets":[{"id":"fixture","trial_count":3,"trial_budget":3},{"id":"compile","trial_count":3,"trial_budget":3},{"id":"search","trial_count":240,"trial_budget":240},{"id":"confirm","trial_count":24,"trial_budget":24},{"id":"contract","trial_count":30,"trial_budget":30}],"deterministic_artifact_fields":["source_spec_sha256","world_plan_sha256","ruling_sha256","decision_record_sha256","contract_bundle_sha256","reference_binary_sha256"],"fresh_run_digest_fields":["world_instance_sha256","attempts_sha256","measurements_sha256","captures_sha256","confirmation_sha256","fixture_invocation_sha256","contract_execution_target_sha256","finalized_contract_run_sha256","contract_execution_sha256"],"html_authority":"LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS","ledger_authority":"LOCAL_SECRET_BEARING_SNAPSHOT_NOT_PORTABLE","projection_paths":["docs/HANDOFF_MODE_C.md","docs/status/U7D-EVIDENCE.md"],"projection_policy":"EXACT_PARENT_RELATIVE_PENDING_BLOCK_REPLACEMENT","self_receipt":"ABSENT","milestone_verdict":"LOCAL_REFERENCE_MILESTONE_FUNCTIONAL_GREEN_FULL_STUDY_RESOURCE_UNRECEIPTED","honest_fallback":"NOT_APPLICABLE_SOURCE_GREEN","unreceipted":["LINUX_UNRUN","WINDOWS_UNRUN","UNRUN_NODE_MAJORS","BROAD_IMPORTED_REPOSITORY_BEHAVIOR_UNVALIDATED","HOSTILE_CONTAINMENT_UNVALIDATED","NETWORK_DENIAL_UNVALIDATED","COMPREHENSION_UNVALIDATED","REVIEW_COMPRESSION_UNVALIDATED","ADOPTION_UNVALIDATED","MAINTAINABILITY_UNVALIDATED","PRODUCTION_READINESS_UNVALIDATED","SECURITY_REVIEW_NOT_PERFORMED","EXTERNAL_PLATFORM_BEHAVIOR_UNVALIDATED","IMPORTED_REPOSITORY_TIMING_UNCLAIMED","FULL_STUDY_RESOURCE_BOUND_UNVALIDATED","U7R_SELF_RECEIPT_ABSENT"]}`));
const exactClaimBindingPolicy = Object.freeze({
	event_indices: "EXPLICIT_ZERO_BASED_EVENT_INDEX_EQUALS_CLAIM_INDEX",
	pathspecs: "EXACT_UNIT_ALLOWED_PATHS_IN_DECLARED_ORDER",
	recorded_child_process_intervals: "FINITE_NONREVERSED_LEDGER_ORDER_START_AT_OR_AFTER_PREVIOUS_END_INCLUDING_FINAL_NOTE_INSPECTION",
	operational_writer_protocol: "ONE_ROOT_OWNED_WRITER_AND_WRAPPER_PLUS_RELEVANT_CHILD_PROCESS_TERMINAL_BEFORE_NEXT_FENCE_OR_MUTATION",
});
const exactUnitIdentity = Object.freeze({
	U7P: Object.freeze({ parent: "C6B", verification_profile: "SOURCE_FULL", product_authority: "NONE", product_behavior: "INHERITED_UNREPROVEN", u7d_receipt_state: "ABSENT", subject: "chore: lock U7 execution authority", final_root: ".countershape/u7p-final", paths: 17, claims: 13 }),
	U7A: Object.freeze({ parent: "U7P", verification_profile: "SOURCE_FULL", product_authority: "U7_REFERENCE_APPLICATION", product_behavior: "CANDIDATE_UNRECEIPTED", u7d_receipt_state: "ABSENT", subject: "feat: add U7 reference CLI foundation", final_root: ".countershape/u7a-final", paths: 12, claims: 12 }),
	U7B: Object.freeze({ parent: "U7A", verification_profile: "SOURCE_FULL", product_authority: "U7_REFERENCE_APPLICATION", product_behavior: "CANDIDATE_UNRECEIPTED", u7d_receipt_state: "ABSENT", subject: "feat: add U7 HTTP falsification study", final_root: ".countershape/u7b-final", paths: 16, claims: 12 }),
	U7C: Object.freeze({ parent: "U7B", verification_profile: "SOURCE_FULL", product_authority: "U7_REFERENCE_APPLICATION", product_behavior: "CANDIDATE_UNRECEIPTED", u7d_receipt_state: "ABSENT", subject: "feat: add U7 CLI decision and contract flow", final_root: ".countershape/u7c-final", paths: 18, claims: 12 }),
	U7D: Object.freeze({ parent: "U7C", verification_profile: "SOURCE_FULL", product_authority: "U7_REFERENCE_APPLICATION", product_behavior: "CANDIDATE_UNRECEIPTED", u7d_receipt_state: "ABSENT", subject: "test: close U7 reference study evidence", final_root: ".countershape/u7d-final", paths: 15, claims: 13 }),
	U7R: Object.freeze({ parent: "U7D", verification_profile: "RECEIPT_RECONCILIATION", product_authority: "NONE", product_behavior: "SOURCE_RECEIPT_RECONCILIATION", u7d_receipt_state: "PRESENT", subject: "docs: receipt U7 reference milestone", final_root: ".countershape/u7r-final", paths: 3, claims: 9 }),
});
const exactRosterDigests = Object.freeze({
	U7P: "4dbeadaf7b0b837a08bf820a545985ede2c41a2d2e5d53d54e2095fa2de8c380",
	U7A: "7fcc7c4ca243bc6011cc7fa80aee73729db060165e5ef400d34477c224e10abd",
	U7B: "591d8b464771d6b68a994d7c314c0e9fd9468d42eaffbb1eb7f8250b6c4f4c64",
	U7C: "bdd228a731ec2cc51922aa23367f665237d786afe7b298ad5e844d8d10dd3aa4",
	U7D: "ad9d20d6a493adace402d04849c5e7847d9cb4407faabbcad7c4350d38d9b926",
	U7R: "2c21933c3e13fe32eedafb1d5ac2fdd5187df514408acf53057016cac153bccb",
});
const exactClaimDigests = Object.freeze({
	U7P: "5fe42cb7f23964bbe47b1c0e1136e20268f5823de07742d437e2f4cfe721a41c",
	U7A: "dd89349ecbf9f4dbc4344b334e117fc72d9ec2073d7c9f53a6a14807c9bef476",
	U7B: "b8aa70dde01f17ded108bebfa0560ec18b3c953d1fabb058319b63d999a58ee0",
	U7C: "1001620a4dac4f8f0acd1b6fa34b4a5bbdaaeb1444ea342e121ef6e566485411",
	U7D: "5b7a0a8ff21a18504bd84b5b32a77bb570414ef071293c4a83d4fd73fe75bb25",
	U7R: "4bb6a1f472f7b15e8d1ddbb4edb6271b0d8c08f78b1f546c5b9d3789bfb356d3",
});
const credentialPatterns = Object.freeze([
	Object.freeze({ name: "pem-private-key", expression: /-----BEGIN (?:[A-Z0-9 ]+ )?PRIVATE KEY-----/u }),
	Object.freeze({ name: "aws-access-key", expression: /AKIA[0-9A-Z]{16}/u }),
	Object.freeze({ name: "github-token", expression: /gh[pousr]_[A-Za-z0-9]{30,}/u }),
	Object.freeze({ name: "gitlab-token", expression: /glpat-[A-Za-z0-9_-]{20,}/u }),
	Object.freeze({ name: "slack-token", expression: /xox[baprs]-[A-Za-z0-9-]{20,}/u }),
	Object.freeze({ name: "google-api-key", expression: /AIza[0-9A-Za-z_-]{35}/u }),
	Object.freeze({ name: "openai-key", expression: /sk-(?:proj-)?[A-Za-z0-9_-]{20,}/u }),
	Object.freeze({ name: "stripe-live-key", expression: /(?:sk|rk)_live_[A-Za-z0-9]{16,}/u }),
	Object.freeze({ name: "jwt-bearer", expression: /Bearer[ \t]+eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}/u }),
]);
const maximumFileBytes = 4 * 1024 * 1024;
const maximumRosterBytes = 32 * 1024 * 1024;

function fail(code, detail) {
	throw new Error(`U7_SCOPE_${code}: ${detail}`);
}

function exactKeys(value, keys) {
	return value !== null && typeof value === "object" && !Array.isArray(value) &&
		isDeepStrictEqual(Object.keys(value).sort(), [...keys].sort());
}

function validPath(value) {
	return typeof value === "string" && value.length > 0 && value.length <= 4096 && !isAbsolute(value) &&
		!value.startsWith("./") && !value.includes("\\") && !value.includes("//") &&
		!value.split("/").some((part) => part === "" || part === "." || part === "..") &&
		!/[\u0000-\u001f\u007f]/u.test(value);
}

function digestJSON(value) {
	return createHash("sha256").update(`${JSON.stringify(value)}\n`, "utf8").digest("hex");
}

function decodeUTF8(bytes, label) {
	try { return new TextDecoder("utf-8", { fatal: true }).decode(bytes); }
	catch (error) { fail("UTF8", `${label}:${error.message}`); }
}

async function readRegular(path, maximum = maximumFileBytes, allowEmpty = false) {
	let handle;
	try { handle = await open(path, constants.O_RDONLY | constants.O_NOFOLLOW); }
	catch (error) { fail("OPEN", `${path}:${error.code ?? error.message}`); }
	try {
		const before = await handle.stat();
		if (!before.isFile() || (!allowEmpty && before.size <= 0) || before.size > maximum) fail("FILE", `${path}:${before.size}`);
		const bytes = await handle.readFile();
		const after = await handle.stat();
		if (bytes.length !== before.size || before.dev !== after.dev || before.ino !== after.ino || before.size !== after.size || before.mtimeMs !== after.mtimeMs) fail("FILE_CHANGED", path);
		return bytes;
	} finally { await handle.close(); }
}

async function didrunPackageManifest(authority) {
	const root = authority.didrun_package_root;
	if (await realpath(root) !== root) fail("DIDRUN_PACKAGE_ROOT", root);
	const manifest = [];
	let visited = 0;
	const walk = async (directory, prefix, depth) => {
		if (depth > 16) fail("DIDRUN_PACKAGE_DEPTH", prefix);
		const entries = (await readdir(directory, { withFileTypes: true })).sort((left, right) => left.name < right.name ? -1 : left.name > right.name ? 1 : 0);
		for (const entry of entries) {
			visited += 1;
			if (visited > 1_000 || entry.isSymbolicLink()) fail("DIDRUN_PACKAGE_ENTRY", entry.name);
			const path = prefix === "" ? entry.name : `${prefix}/${entry.name}`;
			const absolute = resolve(directory, entry.name);
			if (entry.isDirectory()) { await walk(absolute, path, depth + 1); continue; }
			if (!entry.isFile()) fail("DIDRUN_PACKAGE_TYPE", path);
			if (/\.cpython-[0-9]+\.pyc$/u.test(path) && !path.endsWith(".cpython-314.pyc")) continue;
			const bytes = await readRegular(absolute, 1024 * 1024);
			manifest.push({ path, bytes: bytes.length, sha256: createHash("sha256").update(bytes).digest("hex") });
		}
	};
	await walk(root, "", 0);
	return {
		fileCount: manifest.length, bytes: manifest.reduce((sum, entry) => sum + entry.bytes, 0),
		digest: createHash("sha256").update(`${JSON.stringify(manifest)}\n`, "utf8").digest("hex"),
	};
}

async function didrunStartupManifest(authority) {
	const root = authority.didrun_site_packages_root;
	if (await realpath(root) !== root) fail("DIDRUN_STARTUP_ROOT", root);
	const manifest = [];
	const entries = (await readdir(root, { withFileTypes: true })).sort((left, right) => left.name < right.name ? -1 : left.name > right.name ? 1 : 0);
	for (const entry of entries) {
		if (!entry.name.endsWith(".pth") && !["sitecustomize.py", "usercustomize.py"].includes(entry.name)) continue;
		if (!entry.isFile() || entry.isSymbolicLink()) fail("DIDRUN_STARTUP_ENTRY", entry.name);
		const bytes = await readRegular(resolve(root, entry.name), 64 * 1024);
		manifest.push({ path: entry.name, bytes: bytes.length, sha256: createHash("sha256").update(bytes).digest("hex") });
	}
	return { fileCount: manifest.length, bytes: manifest.reduce((sum, entry) => sum + entry.bytes, 0),
		digest: createHash("sha256").update(`${JSON.stringify(manifest)}\n`, "utf8").digest("hex") };
}

async function requireDidrunCustomizerAbsence(authority) {
	const editableRoot = resolve(authority.didrun_package_root, "..");
	for (const root of [authority.didrun_site_packages_root, editableRoot]) {
		if (await realpath(root) !== root) fail("DIDRUN_CUSTOMIZER_ROOT", root);
		for (const name of ["sitecustomize.py", "sitecustomize", "usercustomize.py", "usercustomize"]) {
			try {
				const metadata = await lstat(resolve(root, name));
				fail("DIDRUN_CUSTOMIZER_PRESENT", `${root}:${name}:${metadata.isDirectory() ? "directory" : "object"}`);
			} catch (error) {
				if (String(error?.message).startsWith("U7_SCOPE_")) throw error;
				if (error?.code !== "ENOENT") throw error;
			}
		}
	}
	return editableRoot;
}

function recorderEnvironment(authority, inherited = process.env) {
	const environment = {};
	for (const assignment of authority.didrun_outer_static_environment) {
		const separator = assignment.indexOf("=");
		if (separator <= 0) fail("DIDRUN_OUTER_ENVIRONMENT", assignment);
		environment[assignment.slice(0, separator)] = assignment.slice(separator + 1);
	}
	for (const name of ["HOME", "TMPDIR"]) {
		const value = inherited[name];
		if (typeof value !== "string" || !isAbsolute(value) || /[\u0000-\u001f\u007f]/u.test(value)) fail("DIDRUN_OUTER_DYNAMIC_ROOT", name);
		environment[name] = value;
	}
	return environment;
}

async function validateAdmittedToolBytes(authority) {
	for (const tool of authority.admitted_tools) {
		if (!exactKeys(tool, ["name", "path", "realpath", "sha256", "version"]) || await realpath(tool.path) !== tool.realpath ||
			createHash("sha256").update(await readRegular(tool.realpath, 32 * 1024 * 1024)).digest("hex") !== tool.sha256) fail("TOOL_RUNTIME_EPOCH", String(tool?.name));
	}
}

async function validateAdmittedTools(authority) {
	await validateAdmittedToolBytes(authority);
	const outputByName = new Map();
	for (const [name, path, args] of [
		["node", "/opt/homebrew/bin/node", ["--version"]], ["go", "/opt/homebrew/bin/go", ["version"]],
		["git", "/usr/bin/git", ["--version"]], ["sh", "/bin/sh", ["--version"]],
		["clang", "/usr/bin/clang", ["--version"]], ["clang++", "/usr/bin/clang++", ["--version"]],
	]) {
		outputByName.set(name, decodeUTF8(commandBytes(path, args), `${name} version`).split("\n", 1)[0]);
	}
	outputByName.set("gofmt", "go1.26.5-toolchain");
	outputByName.set("env", "NO_STABLE_VERSION_INTERFACE");
	outputByName.set("time", "NO_STABLE_VERSION_INTERFACE");
	for (const tool of authority.admitted_tools) if (outputByName.get(tool.name) !== tool.version) fail("TOOL_RUNTIME_EPOCH", String(tool?.name));
	await validateAdmittedToolBytes(authority);
}

async function validateDidrunStaticAuthority(authority) {
	const didrunExecutable = await realpath(authority.didrun_path);
	if (didrunExecutable !== authority.didrun_entrypoint_realpath) fail("DIDRUN_ENTRYPOINT", didrunExecutable);
	const didrunBytes = await readRegular(didrunExecutable, 1024 * 1024);
	const firstLine = decodeUTF8(didrunBytes, "didrun entrypoint").split("\n", 1)[0];
	if (firstLine !== `#!${authority.didrun_python_entrypoint}` ||
		await realpath(authority.didrun_python_entrypoint) !== authority.didrun_python_realpath) fail("DIDRUN_PYTHON_PATH", firstLine);
	const pythonBytes = await readRegular(authority.didrun_python_realpath, 1024 * 1024);
	if (await realpath(authority.didrun_pyvenv_config_path) !== authority.didrun_pyvenv_config_path) fail("DIDRUN_PYVENV_PATH", authority.didrun_pyvenv_config_path);
	const pyvenvBytes = await readRegular(authority.didrun_pyvenv_config_path, 64 * 1024);
	const routeBytes = await readRegular(authority.didrun_import_route_path, 64 * 1024);
	const editableRoot = await requireDidrunCustomizerAbsence(authority);
	const packageManifest = await didrunPackageManifest(authority);
	const startupManifest = await didrunStartupManifest(authority);
	const repositoryConfig = await readRegular(resolve(repositoryRoot, authority.didrun_repository_git_config_path), 1024 * 1024);
	if (createHash("sha256").update(didrunBytes).digest("hex") !== authority.didrun_entrypoint_shim_sha256 ||
		createHash("sha256").update(pythonBytes).digest("hex") !== authority.didrun_python_launcher_sha256 ||
		pyvenvBytes.length !== authority.didrun_pyvenv_config_bytes || createHash("sha256").update(pyvenvBytes).digest("hex") !== authority.didrun_pyvenv_config_sha256 ||
		authority.didrun_include_system_site_packages !== false ||
		decodeUTF8(pyvenvBytes, "didrun pyvenv.cfg").split("\n").filter((line) => line === "include-system-site-packages = false").length !== 1 ||
		routeBytes.length !== authority.didrun_import_route_bytes || createHash("sha256").update(routeBytes).digest("hex") !== authority.didrun_import_route_sha256 ||
		!routeBytes.equals(Buffer.from(`${editableRoot}\n`, "utf8")) ||
		startupManifest.fileCount !== authority.didrun_startup_file_count || startupManifest.bytes !== authority.didrun_startup_bytes ||
		startupManifest.digest !== authority.didrun_startup_manifest_sha256 || packageManifest.fileCount !== authority.didrun_package_file_count ||
		packageManifest.bytes !== authority.didrun_package_bytes || packageManifest.digest !== authority.didrun_package_manifest_sha256 ||
		repositoryConfig.length !== authority.didrun_repository_git_config_bytes || createHash("sha256").update(repositoryConfig).digest("hex") !== authority.didrun_repository_git_config_sha256 ||
		authority.didrun_repository_replace_refs !== "ABSENT") fail("DIDRUN_STATIC_RUNTIME_EPOCH", didrunExecutable);
	return didrunExecutable;
}

function validateScopeSpecification(value) {
	if (!exactKeys(value, ["schema_version", "transition_authority", "runtime_authority", "inherited_receipts", "receipt_contract", "claim_binding_policy", "sealed_parent", "units"]) ||
		value.schema_version !== "countershape/u7-unit-paths/v1" || !Array.isArray(value.units) ||
		!isDeepStrictEqual(value.units.map((unit) => unit?.id), unitOrder) || !isDeepStrictEqual(value.sealed_parent, exactSealedParent) ||
		!isDeepStrictEqual(value.transition_authority, exactTransitionAuthority) || !isDeepStrictEqual(value.runtime_authority, exactRuntimeAuthority)) fail("SPECIFICATION", "topology, authority, runtime, or sealed parent");
	if (!isDeepStrictEqual(value.inherited_receipts, exactInheritedReceipts)) fail("SPECIFICATION_RECEIPTS", "C3P/C3/C6A");
	if (!isDeepStrictEqual(value.receipt_contract, exactReceiptContract)) fail("SPECIFICATION_RECEIPT_CONTRACT", "U7D evidence and U7R projection");
	if (!isDeepStrictEqual(value.claim_binding_policy, exactClaimBindingPolicy)) fail("SPECIFICATION_CLAIM_BINDING", "event, pathspec, recorded child-process interval, and operational writer authority");
	for (const unit of value.units) {
		const identity = exactUnitIdentity[unit.id];
		if (!exactKeys(unit, ["id", "parent", "verification_profile", "product_authority", "product_behavior", "u7d_receipt_state", "subject", "final_root", "allowed_paths", "required_paths", "claims"]) ||
			identity === undefined || Object.entries(identity).some(([key, expected]) => !["paths", "claims"].includes(key) && unit[key] !== expected) ||
			!Array.isArray(unit.allowed_paths) || !Array.isArray(unit.required_paths) || unit.allowed_paths.length === 0 ||
			new Set(unit.allowed_paths).size !== unit.allowed_paths.length || new Set(unit.required_paths).size !== unit.required_paths.length ||
			unit.allowed_paths.some((path) => !validPath(path)) || unit.required_paths.some((path) => !validPath(path)) ||
			unit.required_paths.some((path) => !unit.allowed_paths.includes(path)) || !isDeepStrictEqual(unit.required_paths, unit.allowed_paths)) fail("SPECIFICATION_UNIT", unit?.id ?? "unknown");
		if (unit.allowed_paths.length !== identity.paths || digestJSON(unit.allowed_paths) !== exactRosterDigests[unit.id]) fail("SPECIFICATION_ROSTER", unit.id);
		if (!Array.isArray(unit.claims) || unit.claims.length !== identity.claims || digestJSON(unit.claims) !== exactClaimDigests[unit.id] ||
			unit.claims.some((claim) => !exactKeys(claim, ["type", "label", "command"]) || !["tests-pass", "command-succeeded"].includes(claim.type) ||
				typeof claim.label !== "string" || !Array.isArray(claim.command) || claim.command.length < 2 || !isAbsolute(claim.command[0]))) fail("SPECIFICATION_CLAIMS", unit.id);
	}
	if (!isDeepStrictEqual(value.units[0].allowed_paths, u7pExactPaths) || !isDeepStrictEqual(value.units[0].required_paths, u7pExactPaths)) fail("SPECIFICATION_U7P", "exact roster drift");
	const receipt = value.units.at(-1);
	if (receipt.verification_profile !== "RECEIPT_RECONCILIATION" || receipt.product_authority !== "NONE" ||
		!receipt.required_paths.includes(receiptPath)) fail("SPECIFICATION_RECEIPT", "U7R");
	return value;
}

async function loadSpecification() {
	let value;
	try { value = JSON.parse(decodeUTF8(await readRegular(specificationPath, 1024 * 1024), "specification")); }
	catch (error) { if (String(error.message).startsWith("U7_SCOPE_")) throw error; fail("SPECIFICATION_JSON", error.message); }
	const specification = validateScopeSpecification(value);
	const nodeMajor = Number.parseInt(process.versions.node.split(".")[0], 10);
	if (process.platform !== specification.runtime_authority.platform || process.arch !== specification.runtime_authority.architecture ||
		nodeMajor !== specification.runtime_authority.node_major ||
		await realpath(process.execPath) !== await realpath(specification.runtime_authority.node_path)) {
		fail("RUNTIME_EPOCH", `${process.platform}/${process.arch} node=${process.versions.node} exec=${process.execPath}`);
	}
	await validateAdmittedTools(specification.runtime_authority);
	const didrunExecutable = await validateDidrunStaticAuthority(specification.runtime_authority);
	const outerEnvironment = recorderEnvironment(specification.runtime_authority);
	const replaceRefs = decodeUTF8(commandBytes(specification.runtime_authority.didrun_outer_git_path, ["--no-replace-objects", "for-each-ref", "--format=%(refname)", "refs/replace"], { env: outerEnvironment }), "replace refs");
	if (replaceRefs !== "") fail("DIDRUN_REPLACE_REFS", replaceRefs.trim());
	const didrunVersion = decodeUTF8(commandBytes(specification.runtime_authority.didrun_path, ["--version"], { env: outerEnvironment }), "didrun version");
	const importedRoot = decodeUTF8(commandBytes(specification.runtime_authority.didrun_python_entrypoint, ["-c", "import pathlib, site, didrun; print(pathlib.Path(didrun.__file__).resolve().parent); print(int(bool(site.ENABLE_USER_SITE)))"], { env: outerEnvironment }), "didrun import root");
	if (importedRoot !== `${specification.runtime_authority.didrun_package_root}\n0\n` ||
		didrunVersion !== `${specification.runtime_authority.didrun_version}\n`) fail("DIDRUN_RUNTIME_EPOCH", didrunExecutable);
	await validateDidrunStaticAuthority(specification.runtime_authority);
	await validateAdmittedToolBytes(specification.runtime_authority);
	return specification;
}

function unitByID(specification, id) {
	const unit = specification.units.find((candidate) => candidate.id === id);
	if (unit === undefined) fail("UNIT", String(id));
	return unit;
}

function gitEnvironment() {
	return {
		HOME: process.env.HOME || "/", TMPDIR: process.env.TMPDIR || "/tmp", PATH: "/usr/bin:/bin",
		LANG: "C", LC_ALL: "C", NO_COLOR: "1", GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null",
		GIT_NO_LAZY_FETCH: "1", GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
	};
}

function commandBytes(command, args, { input, timeout = 30_000, maxBuffer = maximumRosterBytes, env = gitEnvironment() } = {}) {
	const result = spawnSync(command, args, { cwd: repositoryRoot, encoding: "buffer", timeout, maxBuffer, input, env });
	const stderr = result.stderr ?? Buffer.alloc(0);
	if (result.error || result.signal || result.status !== 0 || stderr.length !== 0) fail("COMMAND", `${command} ${args.join(" ")} status=${result.status} signal=${result.signal} error=${result.error?.message ?? "none"} stderr_bytes=${stderr.length}`);
	return result.stdout ?? Buffer.alloc(0);
}

function gitBytes(args) {
	return commandBytes(process.env.COUNTERSHAPE_GIT || "/usr/bin/git", ["--no-replace-objects", ...args]);
}

function gitLine(args, label) {
	const text = decodeUTF8(gitBytes(args), label);
	if (!/^[^\r\n]+\n$/u.test(text)) fail("GIT_LINE", label);
	return text.slice(0, -1);
}

function decodeNULPaths(bytes, label) {
	if (bytes.length > 0 && bytes.at(-1) !== 0) fail("NUL", label);
	const text = decodeUTF8(bytes.length === 0 ? bytes : bytes.subarray(0, -1), label);
	const paths = text === "" ? [] : text.split("\0");
	if (paths.some((path) => !validPath(path)) || new Set(paths).size !== paths.length) fail("PATH_INVENTORY", label);
	return paths.sort();
}

function stagedPaths() {
	return decodeNULPaths(gitBytes([
		"diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--cached",
		"--name-only", "-z", "--no-renames", "--diff-filter=ACDMRTUXB", "--",
	]), "staged paths");
}

function stagedIndexEntries() {
	const bytes = gitBytes(["ls-files", "--stage", "-z", "--"]);
	if (bytes.length > 0 && bytes.at(-1) !== 0) fail("INDEX_NUL", "missing final delimiter");
	const text = decodeUTF8(bytes.length === 0 ? bytes : bytes.subarray(0, -1), "index");
	const rows = text === "" ? [] : text.split("\0");
	return rows.map((row) => {
		const match = /^([0-7]{6}) ([0-9a-f]{40}|[0-9a-f]{64}) ([0-3])\t([\s\S]+)$/u.exec(row);
		if (!match || !validPath(match[4])) fail("INDEX_ROW", row.slice(0, 128));
		return Object.freeze({ mode: match[1], object: match[2], stage: Number(match[3]), path: match[4] });
	});
}

function validateModes(paths, entries) {
	for (const path of paths) {
		const matches = entries.filter((entry) => entry.path === path);
		if (matches.length !== 1 || matches[0].mode !== "100644" || matches[0].stage !== 0 || /^0+$/u.test(matches[0].object)) fail("INDEX_MODE", path);
	}
}

function validateRoster(unit, paths, receiptExact = false) {
	if (paths.length === 0 || paths.some((path) => !unit.allowed_paths.includes(path)) || unit.required_paths.some((path) => !paths.includes(path))) fail("ROSTER", unit.id);
	if ((unit.id === "U7P" || receiptExact) && !isDeepStrictEqual(paths, [...unit.allowed_paths].sort())) fail("EXACT_ROSTER", unit.id);
}

function requireCleanCandidate(unit, receiptExact = false) {
	const headBefore = gitLine(["rev-parse", "--verify", "HEAD^{commit}"], "HEAD before");
	const treeBefore = gitLine(["write-tree"], "index tree before");
	const paths = stagedPaths();
	validateRoster(unit, paths, receiptExact);
	validateModes(paths, stagedIndexEntries());
	gitBytes(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--cached", "--check", "--"]);
	if (gitBytes(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--name-only", "-z", "--"]).length !== 0) fail("UNSTAGED", unit.id);
	if (gitBytes(["ls-files", "--others", "--exclude-standard", "-z", "--"]).length !== 0) fail("UNTRACKED", unit.id);
	const headAfter = gitLine(["rev-parse", "--verify", "HEAD^{commit}"], "HEAD after");
	const treeAfter = gitLine(["write-tree"], "index tree after");
	if (headBefore !== headAfter || treeBefore !== treeAfter || !isDeepStrictEqual(paths, stagedPaths())) fail("SNAPSHOT_CHANGED", unit.id);
	if (unit.id === "U7P" && headBefore !== exactC6BCommit) fail("U7P_PARENT", headBefore);
	return Object.freeze({ head: headBefore, tree: treeBefore, paths });
}

function stagedBlobs(paths) {
	let total = 0;
	return paths.map((path) => {
		const bytes = gitBytes(["show", `:${path}`]);
		total += bytes.length;
		if (bytes.length === 0 || bytes.length > maximumFileBytes || total > maximumRosterBytes) fail("BLOB_BOUNDS", path);
		return Object.freeze({ path, bytes, text: decodeUTF8(bytes, path) });
	});
}

export function credentialPatternFindings(entries) {
	if (!Array.isArray(entries)) fail("CREDENTIAL_ENTRIES", "array required");
	const findings = [];
	for (const entry of entries) {
		if (!entry || typeof entry.path !== "string" || typeof entry.text !== "string") fail("CREDENTIAL_ENTRY", String(entry?.path));
		for (const pattern of credentialPatterns) if (pattern.expression.test(entry.text)) findings.push(Object.freeze({ path: entry.path, pattern: pattern.name }));
	}
	return findings;
}

function runFinalGate(specification, id, receiptExact = false) {
	const unit = unitByID(specification, id);
	if (receiptExact !== (unit.verification_profile === "RECEIPT_RECONCILIATION")) fail("PROFILE_GATE", id);
	const snapshot = requireCleanCandidate(unit, receiptExact);
	const digest = createHash("sha256").update(`${snapshot.paths.join("\n")}\n`, "utf8").digest("hex");
	console.log(`${id} final ${receiptExact ? "receipt" : "source"} scope exact: paths=${snapshot.paths.length} mode=100644 head=${snapshot.head} tree=${snapshot.tree} clean=true sorted-newline-sha256=${digest}`);
}

function runCredentialScan(specification, id) {
	const unit = unitByID(specification, id);
	const receiptExact = unit.verification_profile === "RECEIPT_RECONCILIATION";
	const snapshot = requireCleanCandidate(unit, receiptExact);
	const findings = credentialPatternFindings(stagedBlobs(snapshot.paths));
	if (findings.length > 0) fail("CREDENTIAL_FINDINGS", JSON.stringify(findings));
	console.log(`${id} scoped staged credential-pattern scan exact: findings=0 paths=${snapshot.paths.length} patterns=${credentialPatterns.length}`);
}

function runGofmt(specification, id) {
	const unit = unitByID(specification, id);
	if (unit.verification_profile !== "SOURCE_FULL") fail("GOFMT_PROFILE", id);
	const paths = stagedPaths();
	validateRoster(unit, paths);
	validateModes(paths, stagedIndexEntries());
	const goFiles = stagedBlobs(paths).filter((entry) => entry.path.endsWith(".go"));
	if (goFiles.length === 0) fail("GOFMT_EMPTY", id);
	const requested = process.env.COUNTERSHAPE_GOFMT || "/opt/homebrew/bin/gofmt";
	const admittedGofmt = specification.runtime_authority.admitted_tools.find((tool) => tool.name === "gofmt")?.path;
	if (!isAbsolute(requested) || requested !== admittedGofmt) fail("GOFMT_AUTHORITY", requested);
	for (const entry of goFiles) {
		const formatted = commandBytes(requested, [], { input: entry.bytes, maxBuffer: maximumFileBytes });
		if (!formatted.equals(entry.bytes)) fail("GOFMT_DRIFT", entry.path);
	}
	if (!isDeepStrictEqual(paths, stagedPaths())) fail("GOFMT_SNAPSHOT_CHANGED", id);
	console.log(`${id} staged Go formatting exact: files=${goFiles.length} gofmt=${requested}`);
}

function changedPathsForCommit(commit) {
	return decodeNULPaths(gitBytes(["diff-tree", "--root", "--no-commit-id", "--name-only", "-r", "-z", "--no-renames", commit, "--"]), `${commit} changed paths`);
}

const noteRedaction = "«redacted:high-entropy»";
const noteRootRedactionPositions = new Set([2, 4, 5, 6, 7, 8]);
const noteBareRedactionPositions = new Set([31, 32, 33, 35, 36, 37]);

function hermeticPrefix(unit) {
	const runRoot = resolve(repositoryRoot, unit.final_root);
	return [
		"/usr/bin/env", "-i", `HOME=${resolve(runRoot, "home")}`, `PWD=${repositoryRoot}`,
		`TMPDIR=${resolve(runRoot, "tmp")}`, `GOTMPDIR=${resolve(runRoot, "gotmp")}`,
		`GOCACHE=${resolve(runRoot, "gocache")}`, `GOPATH=${resolve(runRoot, "gopath")}`,
		`GOMODCACHE=${resolve(runRoot, "gomodcache")}`, "GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local",
		"GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off", "GOFLAGS=-mod=readonly -buildvcs=false -p=1",
		"CGO_ENABLED=1", "GOMAXPROCS=2", "LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "NODE_OPTIONS=",
		"NODE_PATH=", "PATH=/opt/homebrew/bin:/usr/bin:/bin", "SHELL=/bin/sh", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_LAZY_FETCH=1", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0",
		"COUNTERSHAPE_GO=/opt/homebrew/bin/go", "COUNTERSHAPE_NODE=/opt/homebrew/bin/node", "COUNTERSHAPE_GIT=/usr/bin/git",
		"COUNTERSHAPE_SH=/bin/sh", "COUNTERSHAPE_GOFMT=/opt/homebrew/bin/gofmt", "COUNTERSHAPE_CC=/usr/bin/clang",
		"COUNTERSHAPE_CXX=/usr/bin/clang++", "CC=/usr/bin/clang", "CXX=/usr/bin/clang++",
		"PYTHONPATH=", "PYTHONHOME=", "PYTHONNOUSERSITE=1", "PYTHONSAFEPATH=1", "PYTHONDONTWRITEBYTECODE=1", "PYTHONHASHSEED=0", "PYTHONUTF8=1", "GIT_NO_REPLACE_OBJECTS=1",
	];
}

function expectedRedactions(position, expected) {
	const candidates = [];
	if (noteRootRedactionPositions.has(position)) {
		candidates.push(`${noteRedaction}.${noteRedaction}`);
		const suffixOffset = expected.indexOf(".countershape/");
		if (suffixOffset !== -1) candidates.push(`${noteRedaction}${expected.slice(suffixOffset)}`);
	}
	if (noteBareRedactionPositions.has(position)) candidates.push(noteRedaction);
	return candidates;
}

function claimPreviewMatches(preview, expectedArgv, prefix) {
	if (!Array.isArray(preview) || preview.length !== expectedArgv.length || !isDeepStrictEqual(expectedArgv.slice(0, prefix.length), prefix) ||
		preview.some((part) => typeof part !== "string" || part.length === 0 || part.length > 4096 || /[\u0000-\u001f\u007f]/u.test(part))) return false;
	for (let index = 0; index < prefix.length; index += 1) {
		if (preview[index] !== expectedArgv[index] && !expectedRedactions(index, expectedArgv[index]).includes(preview[index])) return false;
	}
	return isDeepStrictEqual(preview.slice(prefix.length), expectedArgv.slice(prefix.length));
}

function validateScopeNote(row, commit, tree) {
	const noteBlob = gitLine(["notes", "--ref=didrun", "list", commit], `${row.id} note blob`);
	let note;
	try { note = JSON.parse(decodeUTF8(gitBytes(["cat-file", "blob", noteBlob]), `${row.id} note`)); }
	catch (error) { fail("CANDIDATE_NOTE_JSON", `${row.id}:${error.message}`); }
	if (!exactKeys(note, ["claims", "commit", "coverage", "secrets_override", "tree", "version"]) || note.version !== 1 ||
		note.commit !== commit || note.tree !== tree || typeof note.secrets_override !== "boolean" || !Array.isArray(note.claims) ||
		note.claims.length !== row.claims.length || !exactKeys(note.coverage, ["by_coverage", "total_events"]) ||
		!exactKeys(note.coverage.by_coverage, ["complete"]) || note.coverage.total_events !== row.claims.length ||
		note.coverage.by_coverage.complete !== row.claims.length) fail("CANDIDATE_NOTE", row.id);
	const prefix = hermeticPrefix(row);
	for (const [index, expected] of row.claims.entries()) {
		const record = note.claims[index];
		const claim = record?.claim;
		if (!exactKeys(record, ["claim", "delta", "exit_code", "grade", "reason", "supporting_event_index"]) || record.grade !== "tree-exact" ||
			record.exit_code !== 0 || !isDeepStrictEqual(record.delta, []) || record.supporting_event_index !== index ||
			record.reason !== "self-stable command ran against the sealed tree" ||
			!exactKeys(claim, ["argv_preview", "ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) ||
			claim.label !== expected.label || claim.ctype !== expected.type || claim.declared_at_index !== index ||
				!isDeepStrictEqual(claim.event_indices, [index]) || !isDeepStrictEqual(claim.pathspecs, row.allowed_paths) ||
			!claimPreviewMatches(claim.argv_preview, [...prefix, ...expected.command], prefix)) fail("CANDIDATE_NOTE_CLAIM", `${row.id}:${index}`);
	}
	if (!isDeepStrictEqual(changedPathsForCommit(commit), [...row.allowed_paths].sort())) fail("CANDIDATE_PARENT_SCOPE", row.id);
	return Object.freeze({ noteBlob, note });
}

const requiredLedgerFiles = Object.freeze([".gitignore", "claims.jsonl", "seals.jsonl", "session.log"]);

function canonicalJSON(value) {
	if (value === null || typeof value === "boolean") return JSON.stringify(value);
	if (typeof value === "number") {
		if (!Number.isFinite(value)) fail("CANONICAL_JSON_NUMBER", String(value));
		return JSON.stringify(value);
	}
	if (typeof value === "string") {
		return JSON.stringify(value).replace(/[\u0080-\uffff]/gu, (character) => `\\u${character.charCodeAt(0).toString(16).padStart(4, "0")}`);
	}
	if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
	if (typeof value === "object" && value !== undefined) {
		return `{${Object.keys(value).sort().map((key) => `${canonicalJSON(key)}:${canonicalJSON(value[key])}`).join(",")}}`;
	}
	fail("CANONICAL_JSON_TYPE", typeof value);
}

function parseCanonicalJSONLines(bytes, label) {
	const text = decodeUTF8(bytes, label);
	if (!text.endsWith("\n") || text === "\n") fail("ARCHIVE_JSONL_DELIMITER", label);
	return text.slice(0, -1).split("\n").map((line, index) => {
		try {
			const value = JSON.parse(line);
			if (canonicalJSON(value) !== line) fail("ARCHIVE_JSONL_CANONICAL", `${label}:${index}`);
			return value;
		} catch (error) {
			if (String(error.message).startsWith("U7_SCOPE_")) throw error;
			fail("ARCHIVE_JSONL", `${label}:${index}:${error.message}`);
		}
	});
}

function computeEntryHash(entry) {
	if (!entry || typeof entry !== "object" || !Number.isSafeInteger(entry.index) || typeof entry.prev_hash !== "string" || entry.event === undefined) {
		fail("ARCHIVE_ENTRY_HASH_INPUT", String(entry?.index));
	}
	const body = Buffer.from(canonicalJSON({ index: entry.index, prev: entry.prev_hash, event: entry.event }), "ascii");
	return createHash("sha256").update(Buffer.concat([Buffer.from(entry.prev_hash, "ascii"), body])).digest("hex");
}

function validateRecordedChildIntervals(entries, label) {
	if (!Array.isArray(entries) || entries.length === 0 || typeof label !== "string" || label.length === 0) {
		fail("RECORDED_CHILD_INTERVAL", `${String(label)}:shape`);
	}
	let previousEnd;
	for (const [index, entry] of entries.entries()) {
		const started = entry?.event?.started_at;
		const ended = entry?.event?.ended_at;
		if (!Number.isFinite(started) || !Number.isFinite(ended) || ended < started) {
			fail("RECORDED_CHILD_INTERVAL", `${label}:${index}`);
		}
		if (previousEnd !== undefined && started < previousEnd) {
			fail("RECORDED_CHILD_INTERVAL_OVERLAP", `${label}:${index}`);
		}
		previousEnd = ended;
	}
}

async function requireArchivePath(relativePath) {
	if (!validPath(relativePath)) fail("ARCHIVE_PATH", relativePath);
	const parts = relativePath.split("/");
	for (let index = 0; index < parts.length; index += 1) {
		const path = resolve(repositoryRoot, ...parts.slice(0, index + 1));
		let stat;
		try { stat = await lstat(path); } catch (error) { fail("ARCHIVE_PATH", `${relativePath}:${error.code ?? error.message}`); }
		if (stat.isSymbolicLink() || (index < parts.length - 1 && !stat.isDirectory())) fail("ARCHIVE_PATH_TYPE", relativePath);
	}
}

async function collectArchive(relativeRoot) {
	await requireArchivePath(relativeRoot);
	const root = resolve(repositoryRoot, relativeRoot);
	const before = await lstat(root);
	if (!before.isDirectory() || before.isSymbolicLink()) fail("ARCHIVE_ROOT", relativeRoot);
	const rootEntries = await readdir(root, { withFileTypes: true });
	const rootNames = rootEntries.map((entry) => entry.name).sort();
	if (!isDeepStrictEqual(rootNames, [...requiredLedgerFiles, "objects"].sort())) fail("ARCHIVE_LAYOUT", rootNames.join(","));
	for (const entry of rootEntries) {
		if (entry.isSymbolicLink()) fail("ARCHIVE_TYPE", entry.name);
		if (entry.name === "objects" ? !entry.isDirectory() : !entry.isFile()) fail("ARCHIVE_TYPE", entry.name);
	}
	const objectsRoot = resolve(root, "objects");
	const objectsBefore = await lstat(objectsRoot);
	if (!objectsBefore.isDirectory() || objectsBefore.isSymbolicLink()) fail("ARCHIVE_OBJECT_ROOT", relativeRoot);
	const objectEntries = await readdir(objectsRoot, { withFileTypes: true });
	if (objectEntries.length === 0 || objectEntries.length > 9_996) fail("ARCHIVE_OBJECT_COUNT", String(objectEntries.length));
	const paths = [...requiredLedgerFiles, ...objectEntries.map((entry) => {
		if (!entry.isFile() || entry.isSymbolicLink() || !/^[0-9a-f]{64}$/u.test(entry.name)) fail("ARCHIVE_OBJECT_NAME", entry.name);
		return `objects/${entry.name}`;
	})].sort();
	const manifest = [];
	const bytesByPath = new Map();
	for (const path of paths) {
		const bytes = await readRegular(resolve(root, path), 32 * 1024 * 1024, path.startsWith("objects/"));
		manifest.push(Object.freeze({ path, bytes: bytes.length, sha256: sha256(bytes) }));
		bytesByPath.set(path, bytes);
	}
	const objectDigests = new Set(objectEntries.map((entry) => entry.name));
	for (const digest of objectDigests) {
		if (manifest.find((entry) => entry.path === `objects/${digest}`)?.sha256 !== digest) fail("ARCHIVE_OBJECT_DIGEST", digest);
	}
	const after = await lstat(root);
	const objectsAfter = await lstat(objectsRoot);
	if (before.dev !== after.dev || before.ino !== after.ino || before.mtimeMs !== after.mtimeMs ||
		objectsBefore.dev !== objectsAfter.dev || objectsBefore.ino !== objectsAfter.ino || objectsBefore.mtimeMs !== objectsAfter.mtimeMs ||
		!isDeepStrictEqual(rootNames, (await readdir(root)).sort()) ||
		!isDeepStrictEqual([...objectDigests].sort(), (await readdir(objectsRoot)).sort())) fail("ARCHIVE_DRIFT", relativeRoot);
	return Object.freeze({ manifest: Object.freeze(manifest), bytesByPath, objectDigests });
}

function archiveManifestDigest(manifest) {
	return sha256(Buffer.from(`${JSON.stringify(manifest)}\n`, "utf8"));
}

function previewMatchesArchive(preview, raw) {
	if (!Array.isArray(preview) || !Array.isArray(raw) || preview.length !== raw.length) return false;
	return preview.every((shown, index) => shown === raw[index] || expectedRedactions(index, raw[index]).includes(shown));
}

async function validateScopeArchive(row, commit, tree, note, archivePath, expectedArchive = undefined) {
	const collected = await collectArchive(archivePath);
	const manifestBytes = collected.manifest.reduce((sum, entry) => sum + entry.bytes, 0);
	if (expectedArchive !== undefined && (collected.manifest.length !== expectedArchive.file_count || manifestBytes !== expectedArchive.bytes ||
		archiveManifestDigest(collected.manifest) !== expectedArchive.manifest_sha256)) fail("ARCHIVE_MANIFEST", archivePath);
	const session = parseCanonicalJSONLines(collected.bytesByPath.get("session.log"), `${archivePath}/session.log`);
	const claims = parseCanonicalJSONLines(collected.bytesByPath.get("claims.jsonl"), `${archivePath}/claims.jsonl`);
	const seals = parseCanonicalJSONLines(collected.bytesByPath.get("seals.jsonl"), `${archivePath}/seals.jsonl`);
	const claimCount = row?.claims.length ?? note.claims.length;
	if (session.length !== claimCount + 1 || claims.length !== claimCount || seals.length !== 1 ||
		expectedArchive !== undefined && (session.length !== expectedArchive.event_count || claims.length !== expectedArchive.claim_count || seals.length !== expectedArchive.seal_count)) {
		fail("ARCHIVE_CARDINALITY", archivePath);
	}
	let environmentFingerprint;
	const referenced = new Set();
	for (let index = 0; index < claimCount; index += 1) {
		const entry = session[index];
		const event = entry?.event;
		const claim = claims[index];
		const noteRecord = note.claims[index];
		const previous = index === 0 ? "0".repeat(64) : session[index - 1]?.entry_hash;
		if (!exactKeys(entry, ["entry_hash", "event", "index", "prev_hash"]) || entry.index !== index || entry.prev_hash !== previous || computeEntryHash(entry) !== entry.entry_hash ||
			!exactKeys(event, ["argv", "coverage", "cwd", "ended_at", "env_fingerprint", "exit_code", "observed_via", "started_at", "stderr_blob", "stdout_blob", "submodule_dirty", "transcript_blob", "tree_after", "tree_before"]) ||
			event.exit_code !== 0 || event.observed_via !== "wrapper" || event.coverage !== "complete" || event.submodule_dirty !== false || event.cwd !== repositoryRoot ||
			event.tree_before !== tree || event.tree_after !== tree || !Number.isFinite(event.started_at) || !Number.isFinite(event.ended_at) || event.ended_at < event.started_at ||
			!/^[0-9a-f]{16}$/u.test(event.env_fingerprint ?? "") || !exactKeys(claim, ["ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) ||
			!exactKeys(noteRecord?.claim, ["argv_preview", "ctype", "declared_at_index", "event_indices", "label", "pathspecs"])) {
			fail("ARCHIVE_EVENT", `${archivePath}:${index}`);
		}
		environmentFingerprint ??= event.env_fingerprint;
		if (event.env_fingerprint !== environmentFingerprint) fail("ARCHIVE_ENVIRONMENT", `${archivePath}:${index}`);
		const { argv_preview: preview, ...noteClaim } = noteRecord.claim;
		if (!isDeepStrictEqual(noteClaim, claim) || noteRecord.supporting_event_index !== index ||
			(row !== undefined && (!isDeepStrictEqual(event.argv, [...hermeticPrefix(row), ...row.claims[index].command]) || !isDeepStrictEqual(claim.pathspecs, row.allowed_paths))) ||
			(row === undefined && !previewMatchesArchive(preview, event.argv))) fail("ARCHIVE_CLAIM", `${archivePath}:${index}`);
		for (const field of ["stdout_blob", "stderr_blob", "transcript_blob"]) {
			const digest = event[field];
			if (digest !== null && !validSHA256(digest)) fail("ARCHIVE_BLOB", `${archivePath}:${index}:${field}`);
			if (digest !== null) referenced.add(digest);
		}
	}
	const inspectionEntry = session.at(-1);
	const inspection = inspectionEntry?.event;
	const inspectionTail = ["/usr/bin/git", "notes", "--ref=didrun", "show", commit];
	const inspectionArgvMatches = row === undefined ? isDeepStrictEqual(inspection?.argv?.slice(-inspectionTail.length), inspectionTail) :
		isDeepStrictEqual(inspection?.argv, [...hermeticPrefix(row), ...inspectionTail]);
	if (!exactKeys(inspectionEntry, ["entry_hash", "event", "index", "prev_hash"]) || inspectionEntry.index !== claimCount ||
		inspectionEntry.prev_hash !== session.at(-2)?.entry_hash || computeEntryHash(inspectionEntry) !== inspectionEntry.entry_hash ||
		!exactKeys(inspection, ["argv", "coverage", "cwd", "ended_at", "env_fingerprint", "exit_code", "observed_via", "started_at", "stderr_blob", "stdout_blob", "submodule_dirty", "transcript_blob", "tree_after", "tree_before"]) ||
		inspection.exit_code !== 0 || inspection.observed_via !== "wrapper" || inspection.coverage !== "complete" || inspection.submodule_dirty !== false || inspection.cwd !== repositoryRoot ||
		inspection.env_fingerprint !== environmentFingerprint || inspection.tree_before !== tree || inspection.tree_after !== tree || !inspectionArgvMatches) {
		fail("ARCHIVE_INSPECTION", archivePath);
	}
	if (row !== undefined) validateRecordedChildIntervals(session, `${row.id}:archive`);
	for (const field of ["stdout_blob", "stderr_blob", "transcript_blob"]) {
		const digest = inspection[field];
		if (digest !== null && !validSHA256(digest)) fail("ARCHIVE_BLOB", `${archivePath}:inspection:${field}`);
		if (digest !== null) referenced.add(digest);
	}
	if (!isDeepStrictEqual([...referenced].sort(), [...collected.objectDigests].sort()) || !exactKeys(seals[0], ["claims_watermark", "commit", "tree"]) ||
		seals[0].claims_watermark !== claimCount || seals[0].commit !== commit || seals[0].tree !== tree) fail("ARCHIVE_CLOSURE", archivePath);
	const terminal = await collectArchive(archivePath);
	if (!isDeepStrictEqual(terminal.manifest, collected.manifest)) fail("ARCHIVE_TERMINAL_DRIFT", archivePath);
	return Object.freeze({ files: collected.manifest.length, bytes: manifestBytes, events: session.length, claims: claims.length });
}

async function validateC6BAuthority(specification, commit) {
	const sealed = specification.sealed_parent;
	const observed = gitLine(["rev-parse", "--verify", `${commit}^{commit}`], "C6B commit");
	const tree = gitLine(["rev-parse", "--verify", `${observed}^{tree}`], "C6B tree");
	const parent = gitLine(["show", "-s", "--format=%P", observed], "C6B parent");
	const subject = gitLine(["show", "-s", "--format=%s", observed], "C6B subject");
	if (observed !== sealed.commit || tree !== sealed.tree || parent !== sealed.parent || subject !== sealed.subject) fail("C6B_GIT", observed);
	const noteBlob = gitLine(["notes", "--ref=didrun", "list", observed], "C6B note blob");
	const noteBytes = gitBytes(["cat-file", "blob", noteBlob]);
	if (noteBlob !== sealed.note_blob || createHash("sha256").update(noteBytes).digest("hex") !== sealed.note_body_sha256) fail("C6B_NOTE", noteBlob);
	let note;
	try { note = JSON.parse(decodeUTF8(noteBytes, "C6B note")); } catch (error) { fail("C6B_NOTE_JSON", error.message); }
	if (!exactKeys(note, ["claims", "commit", "coverage", "secrets_override", "tree", "version"]) || note.version !== 1 ||
		note.commit !== observed || note.tree !== tree || note.secrets_override !== sealed.secrets_override || !Array.isArray(note.claims) ||
		note.claims.length !== sealed.claim_count || !exactKeys(note.coverage, ["by_coverage", "total_events"]) ||
		!exactKeys(note.coverage.by_coverage, ["complete"]) || note.coverage.total_events !== sealed.claim_count ||
		note.coverage.by_coverage.complete !== sealed.claim_count || note.claims.some((record, index) => !exactKeys(record, ["claim", "delta", "exit_code", "grade", "reason", "supporting_event_index"]) ||
			record.grade !== sealed.grade || record.exit_code !== 0 || !isDeepStrictEqual(record.delta, []) || record.supporting_event_index !== index ||
			record.reason !== "self-stable command ran against the sealed tree")) fail("C6B_NOTE_SHAPE", observed);
	const declaration = specification.transition_authority.predecessor_declaration;
	const artifactRow = decodeUTF8(gitBytes(["ls-tree", observed, "--", declaration.artifact_path]), "P08 artifact row");
	if (artifactRow !== `${declaration.artifact_mode} blob ${declaration.artifact_blob}\t${declaration.artifact_path}\n`) fail("C6B_P08_ROW", declaration.artifact_path);
	const artifactBytes = gitBytes(["show", `${observed}:${declaration.artifact_path}`]);
	if (artifactBytes.length !== declaration.artifact_bytes || `sha256:${createHash("sha256").update(artifactBytes).digest("hex")}` !== declaration.artifact_sha256) fail("C6B_P08_BYTES", declaration.artifact_path);
	const htmlBytes = await readRegular(resolve(repositoryRoot, sealed.html_path), 16 * 1024 * 1024);
	if (createHash("sha256").update(htmlBytes).digest("hex") !== sealed.html_sha256) fail("C6B_HTML", sealed.html_path);
	await validateScopeArchive(undefined, observed, tree, note, sealed.ledger_archive, {
		file_count: sealed.ledger_file_count, bytes: sealed.ledger_bytes, manifest_sha256: sealed.ledger_manifest_sha256,
		event_count: sealed.ledger_event_count, claim_count: sealed.ledger_claim_count, seal_count: sealed.ledger_seal_count,
	});
}

async function validateScopeAncestry(specification, parentID, startingCommit) {
	let commit = startingCommit;
	for (let index = unitOrder.indexOf(parentID); index >= 0; index -= 1) {
		const row = specification.units[index];
		const observedCommit = gitLine(["rev-parse", "--verify", `${commit}^{commit}`], `${row.id} commit`);
		const tree = gitLine(["rev-parse", "--verify", `${observedCommit}^{tree}`], `${row.id} tree`);
		const subject = gitLine(["show", "-s", "--format=%s", observedCommit], `${row.id} subject`);
		const parent = gitLine(["show", "-s", "--format=%P", observedCommit], `${row.id} parent`);
		if (subject !== row.subject || !/^[0-9a-f]{40}$/u.test(parent)) fail("CANDIDATE_ANCESTRY", row.id);
		const { note } = validateScopeNote(row, observedCommit, tree);
		await validateScopeArchive(row, observedCommit, tree, note, `.didrun-history/${row.id.toLowerCase()}-final-${observedCommit.slice(0, 12)}/.didrun`);
		commit = parent;
	}
	if (commit !== exactC6BCommit) fail("CANDIDATE_C6B_ANCESTRY", commit);
	await validateC6BAuthority(specification, commit);
}

async function runCandidatePhase(specification, id) {
	const unit = unitByID(specification, id);
	if (unit.verification_profile !== "SOURCE_FULL") fail("CANDIDATE_PROFILE", id);
	const snapshot = requireCleanCandidate(unit);
	if (id === "U7P") {
		if (snapshot.head !== specification.sealed_parent.commit ||
			gitLine(["rev-parse", "--verify", "HEAD^{tree}"], "C6B tree") !== specification.sealed_parent.tree ||
			gitLine(["show", "-s", "--format=%s", "HEAD"], "C6B subject") !== specification.sealed_parent.subject) fail("CANDIDATE_C6B", snapshot.head);
		await validateC6BAuthority(specification, snapshot.head);
	} else {
		await validateScopeAncestry(specification, unit.parent, snapshot.head);
	}
	console.log(`${id} independent candidate transition exact: parent=${unit.parent} head=${snapshot.head} candidate_tree=${snapshot.tree} paths=${snapshot.paths.length} profile=${unit.verification_profile}`);
}

function positiveInteger(value, maximum = Number.MAX_SAFE_INTEGER) { return Number.isSafeInteger(value) && value > 0 && value <= maximum; }
function sha256(value) { return createHash("sha256").update(value).digest("hex"); }
function validSHA256(value) { return typeof value === "string" && /^[0-9a-f]{64}$/u.test(value) && !/^0+$/u.test(value); }
function scopeRuntimeDigest(specification) { return sha256(Buffer.from(`${JSON.stringify(specification.runtime_authority)}\n`, "utf8")); }

function scopeStudyEventIndex(specification) {
	const source = unitByID(specification, "U7D");
	const matches = source.claims.flatMap((claim, index) => isDeepStrictEqual(claim.command, specification.receipt_contract.study_event_command) ? [index] : []);
	if (!isDeepStrictEqual(matches, [8])) fail("RECEIPT_STUDY_EVENT", JSON.stringify(matches));
	return matches[0];
}

function scopeStudyManifestPaths(contract) {
	const paths = [...Object.values(contract.study_harness.deterministic_artifact_paths), ...Object.values(contract.study_harness.fresh_artifact_paths)];
	for (const phase of contract.study_harness.phase_budgets.slice(2)) for (let index = 1; index <= phase.per_run_trials; index += 1) paths.push(`phases/${phase.id}/trial-${String(index).padStart(3, "0")}.json`);
	return paths.sort();
}

function parseScopeStudyTimeReport(text) {
	if (typeof text !== "string" || text.includes("\r") || text.startsWith("\ufeff") || !text.endsWith("\n")) fail("RECEIPT_STUDY_TIME", "encoding");
	const lines = text.slice(0, -1).split("\n");
	const labels = ["maximum resident set size", "average shared memory size", "average unshared data size", "average unshared stack size", "page reclaims", "page faults", "swaps", "block input operations", "block output operations", "messages sent", "messages received", "signals received", "voluntary context switches", "involuntary context switches", "instructions retired", "cycles elapsed", "peak memory footprint"];
	if (lines.length !== labels.length + 3 || !/^real [0-9]+\.[0-9]{2}$/u.test(lines[0]) || !/^user [0-9]+\.[0-9]{2}$/u.test(lines[1]) || !/^sys [0-9]+\.[0-9]{2}$/u.test(lines[2])) fail("RECEIPT_STUDY_TIME", "shape");
	let rss;
	for (const [index, label] of labels.entries()) { const match = /^\s*([0-9]+)\s+(.+)$/u.exec(lines[index + 3]); if (match === null || match[2] !== label || !Number.isSafeInteger(Number(match[1]))) fail("RECEIPT_STUDY_TIME", String(index)); if (label === "maximum resident set size") rss = Number(match[1]); }
	const real = Number(lines[0].slice(5)); if (!Number.isFinite(real) || real < 0 || !positiveInteger(rss, 16 * 1024 ** 3)) fail("RECEIPT_STUDY_TIME", "values");
	return { wall_time_ms: Math.max(1, Math.ceil(real * 1000)), peak_rss_bytes: rss };
}

function scopeStudyEnvironmentDigest(specification, study, run, name) {
	const row = unitByID(specification, "U7D");
	const assignments = new Map(hermeticPrefix(row).slice(2).map((entry) => [entry.slice(0, entry.indexOf("=")), entry.slice(entry.indexOf("=") + 1)]));
	const admitted = ["PATH", "LANG", "LC_ALL", "TZ", "NO_COLOR", "HOME", "TMPDIR", "GOTMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE", "GOFLAGS", "GOMAXPROCS", "CGO_ENABLED", "COUNTERSHAPE_NODE", "COUNTERSHAPE_GO", "COUNTERSHAPE_GIT", "COUNTERSHAPE_SH", "COUNTERSHAPE_GOFMT", "COUNTERSHAPE_CC", "COUNTERSHAPE_CXX", "CC", "CXX"];
	const environment = Object.fromEntries(admitted.map((key) => [key, assignments.get(key)]));
	const runRoot = resolve(repositoryRoot, row.final_root, "tmp/studies", study.id, `run-${run.ordinal}`);
	Object.assign(environment, { HOME: resolve(runRoot, "home"), TMPDIR: resolve(runRoot, "tmp"), COUNTERSHAPE_STUDY_DOMAIN: study.id, COUNTERSHAPE_STUDY_ORDINAL: String(run.ordinal), NODE_OPTIONS: "", NODE_PATH: "", GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1", GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0", GIT_NO_REPLACE_OBJECTS: "1" });
	if (name === "execute") environment.COUNTERSHAPE_EVIDENCE_ROOT = resolve(runRoot, "evidence");
	return sha256(Buffer.from(`${Object.keys(environment).sort().map((key) => `${key}=${environment[key]}`).join("\n")}\n`, "utf8"));
}

function validateScopeStudyEvidence(value, specification, expectedDriverDigests) {
	const contract = specification.receipt_contract; const harness = contract.study_harness;
	const harnessAuthority = { protocol: harness.protocol, path: harness.path, sha256: harness.sha256, protocol_sha256: harness.protocol_sha256, observation_authority: harness.observation_authority, semantic_ceiling: harness.semantic_ceiling };
	if (!exactKeys(value, ["schema_version", "phase", "harness_authority", "environment", "studies", "milestone_verdict", "honest_fallback", "unreceipted"]) || value.schema_version !== contract.study_evidence_schema || value.phase !== "U7D" || !isDeepStrictEqual(value.harness_authority, harnessAuthority) ||
		value.milestone_verdict !== contract.milestone_verdict || value.honest_fallback !== contract.honest_fallback || !isDeepStrictEqual(value.unreceipted, contract.unreceipted) || !Array.isArray(value.studies) ||
		!isDeepStrictEqual(value.studies.map((study) => study?.id), contract.study_domains) || !exactKeys(expectedDriverDigests, contract.study_domains) || contract.study_domains.some((id) => !validSHA256(expectedDriverDigests[id]))) fail("RECEIPT_STUDY_SHAPE", "root");
	const versions = Object.fromEntries(specification.runtime_authority.admitted_tools.map((tool) => [tool.name, tool.version])); const tools = Object.fromEntries(specification.runtime_authority.admitted_tools.map((tool) => [tool.name, tool]));
	const environment = value.environment;
	if (!exactKeys(environment, ["platform", "kernel_release", "architecture", "git_version", "go_version", "node_version", "cpu_model", "logical_cpu_count", "memory_bytes", "runtime_authority_sha256"]) || environment.platform !== "darwin" || environment.architecture !== "arm64" || !/^\d+(?:\.\d+){1,3}$/u.test(environment.kernel_release ?? "") ||
		environment.git_version !== versions.git || environment.go_version !== versions.go || environment.node_version !== versions.node || typeof environment.cpu_model !== "string" || environment.cpu_model.length < 3 || environment.cpu_model.length > 160 || /[\u0000-\u001f\u007f]/u.test(environment.cpu_model) ||
		!positiveInteger(environment.logical_cpu_count, 512) || !positiveInteger(environment.memory_bytes, 2 ** 50) || environment.runtime_authority_sha256 !== scopeRuntimeDigest(specification)) fail("RECEIPT_STUDY_ENVIRONMENT", "identity");
	const paths = scopeStudyManifestPaths(contract); const fresh = new Set(); const deterministic = new Set(); const commits = new Set(); const trees = new Set(); const gitDirectories = new Set();
	for (const study of value.studies) {
		if (!exactKeys(study, ["id", "fixture_commit", "fixture_authority", "logical_product_command", "run_count", "trial_count", "budget_trial_count", "observed_subject_process_wall_time_ms", "budget_subject_process_wall_time_ms", "observed_subject_process_peak_rss_bytes", "budget_subject_process_peak_rss_bytes", "observation_authority", "phases", "deterministic_artifacts", "runs"]) || !/^[0-9a-f]{40}$/u.test(study.fixture_commit ?? "") || /^0+$/u.test(study.fixture_commit) ||
			!isDeepStrictEqual(study.logical_product_command, ["countershape", "study", study.id, "--json"]) || study.run_count !== 3 || study.trial_count !== 300 || study.budget_trial_count !== 300 || study.budget_subject_process_wall_time_ms !== contract.study_subject_process_wall_time_budget_ms_per_domain ||
			study.budget_subject_process_peak_rss_bytes !== contract.study_subject_process_peak_rss_budget_bytes_per_domain || !positiveInteger(study.observed_subject_process_wall_time_ms, study.budget_subject_process_wall_time_ms) || !positiveInteger(study.observed_subject_process_peak_rss_bytes, study.budget_subject_process_peak_rss_bytes) || study.observation_authority !== harness.observation_authority ||
			!isDeepStrictEqual(study.phases, contract.study_phase_budgets) || !exactKeys(study.deterministic_artifacts, contract.deterministic_artifact_fields) || !Array.isArray(study.runs) || study.runs.length !== 3) fail("RECEIPT_STUDY_DOMAIN", String(study?.id));
		commits.add(study.fixture_commit); for (const digest of Object.values(study.deterministic_artifacts)) { if (!validSHA256(digest)) fail("RECEIPT_STUDY_DETERMINISTIC", study.id); deterministic.add(digest); }
		const authority = study.fixture_authority;
		if (!exactKeys(authority, ["head_commit", "tree", "clean_status_bytes", "clean_status_sha256", "repository_count", "fresh_repository_per_run", "repository_git_directories"]) || authority.head_commit !== study.fixture_commit || !/^[0-9a-f]{40}$/u.test(authority.tree ?? "") || /^0+$/u.test(authority.tree) || authority.clean_status_bytes !== 0 || authority.clean_status_sha256 !== sha256(Buffer.alloc(0)) || authority.repository_count !== 3 || authority.fresh_repository_per_run !== true || !Array.isArray(authority.repository_git_directories)) fail("RECEIPT_STUDY_FIXTURE", study.id);
		trees.add(authority.tree); let studyWall = 0; let studyPeak = 0;
		for (const [index, run] of study.runs.entries()) {
			const runKeys = ["ordinal", "trial_count", "trial_budget", "observed_subject_process_wall_time_ms", "budget_subject_process_wall_time_ms", "observed_subject_process_peak_rss_bytes", "budget_subject_process_peak_rss_bytes", "observation_authority", "fixture", "driver_authority", "execution", "processes", "phases", "product_result", "evidence_root", "evidence_manifest", "evidence_manifest_sha256", "deterministic_artifacts", ...Object.keys(harness.fresh_artifact_paths), "fixture_invocation_sha256"];
			const runBudget = contract.study_subject_process_wall_time_budget_ms_per_domain / 3; const phases = contract.study_phase_budgets.map((phase) => ({ id: phase.id, trial_count: phase.trial_count / 3, trial_budget: phase.trial_budget / 3 }));
			if (!exactKeys(run, runKeys) || run.ordinal !== index + 1 || run.trial_count !== 100 || run.trial_budget !== 100 || run.budget_subject_process_wall_time_ms !== runBudget || run.budget_subject_process_peak_rss_bytes !== contract.study_subject_process_peak_rss_budget_bytes_per_domain || !positiveInteger(run.observed_subject_process_wall_time_ms, run.budget_subject_process_wall_time_ms) || !positiveInteger(run.observed_subject_process_peak_rss_bytes, run.budget_subject_process_peak_rss_bytes) || run.observation_authority !== harness.observation_authority || !isDeepStrictEqual(run.phases, phases) || !isDeepStrictEqual(run.deterministic_artifacts, study.deterministic_artifacts)) fail("RECEIPT_STUDY_RUN", `${study.id}:${index + 1}`);
			const runRoot = resolve(repositoryRoot, ".countershape/u7d-final/tmp/studies", study.id, `run-${run.ordinal}`); const cwd = resolve(runRoot, "fixture"); const binary = resolve(cwd, "countershape"); const execution = { cwd, argv: [binary, "study", study.id, "--json"], executable_path: binary, executable_sha256: study.deterministic_artifacts.reference_binary_sha256 };
			if (!isDeepStrictEqual(run.execution, execution) || run.fixture_invocation_sha256 !== sha256(Buffer.from(`${JSON.stringify(execution)}\n`, "utf8")) || run.evidence_root !== relative(repositoryRoot, resolve(runRoot, "evidence")) || !isDeepStrictEqual(run.driver_authority, { path: harness.driver_by_domain[study.id], sha256: expectedDriverDigests[study.id] })) fail("RECEIPT_STUDY_EXECUTION", `${study.id}:${index + 1}`);
			const gitDirectory = relative(repositoryRoot, resolve(cwd, ".git")); const fixture = run.fixture;
			if (!exactKeys(fixture, ["head_commit", "tree", "commit_count", "git_directory", "git_common_directory", "git_alternates", "clean_status_bytes", "clean_status_sha256"]) || fixture.head_commit !== study.fixture_commit || fixture.tree !== authority.tree || fixture.commit_count !== 1 || fixture.git_directory !== gitDirectory || fixture.git_common_directory !== gitDirectory || fixture.git_alternates !== "ABSENT" || fixture.clean_status_bytes !== 0 || fixture.clean_status_sha256 !== sha256(Buffer.alloc(0))) fail("RECEIPT_STUDY_FIXTURE", `${study.id}:${index + 1}`);
			gitDirectories.add(gitDirectory); if (!exactKeys(run.product_result, ["schema_version", "domain", "ordinal", "status"]) || run.product_result.schema_version !== harness.product_result_schema || run.product_result.domain !== study.id || run.product_result.ordinal !== run.ordinal || run.product_result.status !== "GREEN" || !exactKeys(run.processes, ["prepare", "compile", "execute"])) fail("RECEIPT_STUDY_PRODUCT", `${study.id}:${index + 1}`);
			let processWall = 0; let processPeak = 0;
			for (const name of ["prepare", "compile", "execute"]) {
				const observation = run.processes[name]; const keys = ["name", "observer_argv", "cwd", "executable_path", "executable_sha256", "observer_path", "observer_sha256", "environment_sha256", "wall_time_ms", "peak_rss_bytes", "time_report_path", "time_report_bytes", "time_report_sha256", "time_report", "stdout_bytes", "stdout_sha256"];
				const reportBytes = Buffer.from(observation?.time_report ?? "", "utf8"); const parsed = parseScopeStudyTimeReport(observation?.time_report); let expectedCwd; let executable; let executableSha; let args; let stdout;
				if (name === "prepare") { expectedCwd = repositoryRoot; executable = tools.node.path; executableSha = tools.node.sha256; args = [resolve(repositoryRoot, harness.driver_by_domain[study.id]), "--prepare", "--domain", study.id, "--ordinal", String(run.ordinal), "--fixture-root", cwd]; stdout = Buffer.alloc(0); }
				else if (name === "compile") { expectedCwd = repositoryRoot; executable = tools.go.path; executableSha = tools.go.sha256; args = ["build", "-trimpath", "-mod=readonly", "-buildvcs=false", "-o", binary, "./cmd/countershape"]; stdout = Buffer.alloc(0); }
				else { expectedCwd = cwd; executable = binary; executableSha = execution.executable_sha256; args = ["study", study.id, "--json"]; stdout = Buffer.from(`${JSON.stringify(run.product_result)}\n`, "utf8"); }
				const reportPath = resolve(runRoot, "observations", `${name}.time`);
				if (!exactKeys(observation, keys) || observation.name !== name || observation.cwd !== expectedCwd || observation.executable_path !== executable || observation.executable_sha256 !== executableSha || observation.observer_path !== tools.time.path || observation.observer_sha256 !== tools.time.sha256 || observation.environment_sha256 !== scopeStudyEnvironmentDigest(specification, study, run, name) || !isDeepStrictEqual(observation.observer_argv, [tools.time.path, "-p", "-l", "-o", reportPath, executable, ...args]) || observation.time_report_path !== relative(repositoryRoot, reportPath) || observation.time_report_bytes !== reportBytes.length || observation.time_report_sha256 !== sha256(reportBytes) || observation.wall_time_ms !== parsed.wall_time_ms || observation.peak_rss_bytes !== parsed.peak_rss_bytes || observation.stdout_bytes !== stdout.length || observation.stdout_sha256 !== sha256(stdout)) fail("RECEIPT_STUDY_PROCESS", `${study.id}:${index + 1}:${name}`);
				processWall += observation.wall_time_ms; processPeak = Math.max(processPeak, observation.peak_rss_bytes);
			}
			if (run.observed_subject_process_wall_time_ms !== processWall || run.observed_subject_process_peak_rss_bytes !== processPeak || !Array.isArray(run.evidence_manifest) || !isDeepStrictEqual(run.evidence_manifest.map((row) => row?.path), paths)) fail("RECEIPT_STUDY_PROCESS_TOTALS", `${study.id}:${index + 1}`);
			for (const row of run.evidence_manifest) if (!exactKeys(row, ["path", "mode", "bytes", "sha256"]) || row.mode !== "0600" || !positiveInteger(row.bytes, 16 * 1024 * 1024) || !validSHA256(row.sha256)) fail("RECEIPT_STUDY_MANIFEST", `${study.id}:${index + 1}`);
			if (run.evidence_manifest_sha256 !== sha256(Buffer.from(`${JSON.stringify(run.evidence_manifest)}\n`, "utf8"))) fail("RECEIPT_STUDY_MANIFEST_DIGEST", `${study.id}:${index + 1}`);
			const manifest = new Map(run.evidence_manifest.map((row) => [row.path, row.sha256])); for (const [field, path] of Object.entries(harness.deterministic_artifact_paths)) if (run.deterministic_artifacts[field] !== manifest.get(path)) fail("RECEIPT_STUDY_MANIFEST_BINDING", field); for (const [field, path] of Object.entries(harness.fresh_artifact_paths)) if (run[field] !== manifest.get(path)) fail("RECEIPT_STUDY_MANIFEST_BINDING", field);
			for (const field of contract.fresh_run_digest_fields) { const digest = run[field]; if (!validSHA256(digest) || fresh.has(digest)) fail("RECEIPT_STUDY_FRESH", `${study.id}:${index + 1}:${field}`); fresh.add(digest); }
			studyWall += run.observed_subject_process_wall_time_ms; studyPeak = Math.max(studyPeak, run.observed_subject_process_peak_rss_bytes);
		}
		if (!isDeepStrictEqual(authority.repository_git_directories, study.runs.map((run) => run.fixture.git_directory)) || study.observed_subject_process_wall_time_ms !== studyWall || study.observed_subject_process_peak_rss_bytes !== studyPeak) fail("RECEIPT_STUDY_TOTALS", study.id);
	}
	if (commits.size !== 2 || trees.size !== 2 || gitDirectories.size !== 6 || [...fresh].some((digest) => deterministic.has(digest))) fail("RECEIPT_STUDY_DOMAINS", "closure");
	return value;
}

export function validateReceiptDeclaration(value, specification, admittedDriverDigests) {
	const requiredLedgerFiles = [".gitignore", "claims.jsonl", "seals.jsonl", "session.log"];
	const keys = [
		"schema_version", "source_boundary", "source_commit", "source_tree", "source_parent", "source_subject",
		"source_note_blob", "source_note_body_sha256", "source_claims", "source_strict_exit", "source_secrets_override",
		"source_html_path", "source_html_bytes", "source_html_sha256", "source_html_authority",
		"source_ledger_archive", "source_ledger_authority", "source_ledger_file_count", "source_ledger_event_count",
		"source_ledger_claim_count", "source_ledger_seal_count", "source_ledger_manifest", "source_ledger_manifest_sha256",
		"source_study_event_index", "source_study_stdout_blob", "source_study_stdout_bytes", "source_study_evidence_sha256", "source_study_evidence",
	];
	if (!exactKeys(value, keys) || value.schema_version !== "countershape/u7-receipt/v1" || value.source_boundary !== "U7D" ||
		value.source_subject !== "test: close U7 reference study evidence" || value.source_strict_exit !== 0 ||
		typeof value.source_secrets_override !== "boolean") fail("RECEIPT_SHAPE", "root");
	for (const [name, width] of [["source_commit", 40], ["source_tree", 40], ["source_parent", 40], ["source_note_blob", 40]]) {
		if (typeof value[name] !== "string" || !new RegExp(`^[0-9a-f]{${width}}$`, "u").test(value[name]) || /^0+$/u.test(value[name])) fail("RECEIPT_OID", name);
	}
	for (const name of ["source_note_body_sha256", "source_html_sha256", "source_ledger_manifest_sha256", "source_study_stdout_blob", "source_study_evidence_sha256"]) if (!validSHA256(value[name])) fail("RECEIPT_SHA256", name);
	if (!validPath(value.source_html_path) || !Number.isSafeInteger(value.source_html_bytes) || value.source_html_bytes <= 0 || value.source_html_bytes > 32 * 1024 * 1024 ||
		value.source_html_authority !== "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS" ||
		!validPath(value.source_ledger_archive) || value.source_ledger_authority !== "LOCAL_SECRET_BEARING_SNAPSHOT_NOT_PORTABLE" ||
		!Array.isArray(value.source_claims) || value.source_claims.length !== 13) fail("RECEIPT_FIELDS", "paths, authority, or claims");
	const sourceShort = value.source_commit.slice(0, 12);
	if (value.source_html_path !== `.countershape/evidence/u7d-final-${sourceShort}.html` ||
		value.source_ledger_archive !== `.didrun-history/u7d-final-${sourceShort}/.didrun`) fail("RECEIPT_LOCAL_PATH_IDENTITY", sourceShort);
	if (!Array.isArray(value.source_ledger_manifest) || value.source_ledger_manifest.length < 5 || value.source_ledger_manifest.length > 10_000) fail("RECEIPT_LEDGER_MANIFEST", "cardinality");
	const manifestPaths = [];
	for (const entry of value.source_ledger_manifest) {
		if (!exactKeys(entry, ["path", "bytes", "sha256"]) || typeof entry.path !== "string" || !Number.isSafeInteger(entry.bytes) ||
			entry.bytes < 0 || entry.bytes > 32 * 1024 * 1024 || typeof entry.sha256 !== "string" || !/^[0-9a-f]{64}$/u.test(entry.sha256) ||
			!requiredLedgerFiles.includes(entry.path) && !/^objects\/[0-9a-f]{64}$/u.test(entry.path)) fail("RECEIPT_LEDGER_MANIFEST", String(entry?.path));
		if (entry.path.startsWith("objects/") && entry.path.slice("objects/".length) !== entry.sha256) fail("RECEIPT_LEDGER_OBJECT", entry.path);
		manifestPaths.push(entry.path);
	}
	if (!isDeepStrictEqual(manifestPaths, [...manifestPaths].sort()) || new Set(manifestPaths).size !== manifestPaths.length ||
		requiredLedgerFiles.some((path) => !manifestPaths.includes(path))) fail("RECEIPT_LEDGER_MANIFEST", "order, uniqueness, or required files");
	const manifestDigest = createHash("sha256").update(`${JSON.stringify(value.source_ledger_manifest)}\n`, "utf8").digest("hex");
	if (value.source_ledger_file_count !== value.source_ledger_manifest.length || value.source_ledger_event_count !== 14 ||
		value.source_ledger_claim_count !== 13 || value.source_ledger_seal_count !== 1 || value.source_ledger_manifest_sha256 !== manifestDigest) fail("RECEIPT_LEDGER_AUTHORITY", "counts or digest");
	for (const [index, claim] of value.source_claims.entries()) if (!exactKeys(claim, ["index", "supporting_event_index", "type", "label", "pathspecs", "grade"]) || claim.index !== index ||
		claim.supporting_event_index !== index || !["tests-pass", "command-succeeded"].includes(claim.type) || typeof claim.label !== "string" || claim.label.length < 8 || claim.grade !== "tree-exact") fail("RECEIPT_CLAIM", String(index));
	const sourcePaths = unitByID(specification, "U7D").allowed_paths;
	for (const [index, claim] of value.source_claims.entries()) if (!isDeepStrictEqual(claim.pathspecs, sourcePaths)) fail("RECEIPT_CLAIM_PATHS", String(index));
	const driverDigests = admittedDriverDigests ?? Object.fromEntries(specification.receipt_contract.study_domains.map((domain) => {
		const path = specification.receipt_contract.study_harness.driver_by_domain[domain];
		return [domain, sha256(gitBytes(["show", `${value.source_commit}:${path}`]))];
	}));
	const studyEvidence = validateScopeStudyEvidence(value.source_study_evidence, specification, driverDigests);
	const studyBytes = Buffer.from(`${JSON.stringify(studyEvidence)}\n`, "utf8");
	if (value.source_study_event_index !== scopeStudyEventIndex(specification) || !positiveInteger(value.source_study_stdout_bytes, 1024 * 1024) ||
		value.source_study_stdout_bytes !== studyBytes.length || value.source_study_stdout_blob !== value.source_study_evidence_sha256 ||
		value.source_study_evidence_sha256 !== sha256(studyBytes)) fail("RECEIPT_STUDY_AUTHORITY", String(value.source_study_event_index));
	const studyManifest = value.source_ledger_manifest.find((entry) => entry.path === `objects/${value.source_study_stdout_blob}`);
	if (studyManifest?.bytes !== studyBytes.length || studyManifest?.sha256 !== value.source_study_stdout_blob) fail("RECEIPT_STUDY_MANIFEST", value.source_study_stdout_blob);
	return value;
}

function parseNote(bytes, declaration, row) {
	let note;
	try { note = JSON.parse(decodeUTF8(bytes, "U7D note")); }
	catch (error) { fail("SOURCE_NOTE_JSON", error.message); }
	if (!exactKeys(note, ["claims", "commit", "coverage", "secrets_override", "tree", "version"]) || note.version !== 1 ||
		note.commit !== declaration.source_commit || note.tree !== declaration.source_tree || note.secrets_override !== declaration.source_secrets_override ||
		!Array.isArray(note.claims) || note.claims.length !== row.claims.length ||
		!exactKeys(note.coverage, ["by_coverage", "total_events"]) || !exactKeys(note.coverage.by_coverage, ["complete"]) ||
		note.coverage.total_events !== row.claims.length || note.coverage.by_coverage.complete !== row.claims.length) fail("SOURCE_NOTE", "root");
	const prefix = hermeticPrefix(row);
	for (const [index, expected] of row.claims.entries()) {
		const record = note.claims[index];
		const claim = record?.claim;
		if (!exactKeys(record, ["claim", "delta", "exit_code", "grade", "reason", "supporting_event_index"]) ||
			record.grade !== "tree-exact" || record.exit_code !== 0 || !isDeepStrictEqual(record.delta, []) ||
			record.supporting_event_index !== index || record.reason !== "self-stable command ran against the sealed tree" ||
			!exactKeys(claim, ["argv_preview", "ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) ||
			claim.label !== expected.label || claim.ctype !== expected.type || claim.declared_at_index !== index ||
				!isDeepStrictEqual(claim.event_indices, [index]) || !isDeepStrictEqual(claim.pathspecs, row.allowed_paths) ||
			!claimPreviewMatches(claim.argv_preview, [...prefix, ...expected.command], prefix)) fail("SOURCE_NOTE_CLAIM", String(index));
	}
	return note;
}

async function loadStagedReceipt(specification) {
	const bytes = gitBytes(["show", `:${receiptPath}`]);
	let parsed;
	try { parsed = JSON.parse(decodeUTF8(bytes, receiptPath)); }
	catch (error) { fail("RECEIPT_JSON", error.message); }
	return validateReceiptDeclaration(parsed, specification);
}

async function runSourceAuthorityGate(specification, id) {
	if (id !== "U7R") fail("SOURCE_AUTHORITY_UNIT", id);
	const unit = unitByID(specification, id);
	const snapshot = requireCleanCandidate(unit, true);
	await validateScopeAncestry(specification, "U7D", snapshot.head);
	const declaration = await loadStagedReceipt(specification);
	{
		const sourceRow = unitByID(specification, "U7D");
		const commit = gitLine(["rev-parse", "--verify", "HEAD^{commit}"], "U7D source commit");
		const tree = gitLine(["rev-parse", "--verify", "HEAD^{tree}"], "U7D source tree");
		const parent = gitLine(["show", "-s", "--format=%P", "HEAD"], "U7D source parent");
		const subject = gitLine(["show", "-s", "--format=%s", "HEAD"], "U7D source subject");
		const noteBlob = gitLine(["notes", "--ref=didrun", "list", commit], "U7D note blob");
		const noteBytes = gitBytes(["cat-file", "blob", noteBlob]);
		const note = parseNote(noteBytes, declaration, sourceRow);
		if (declaration.source_commit !== commit || declaration.source_tree !== tree || declaration.source_parent !== parent ||
			declaration.source_subject !== subject || declaration.source_note_blob !== noteBlob ||
			declaration.source_note_body_sha256 !== createHash("sha256").update(noteBytes).digest("hex") ||
				declaration.source_claims.some((claim, index) => claim.label !== note.claims[index]?.claim?.label ||
					claim.type !== note.claims[index]?.claim?.ctype || claim.grade !== note.claims[index]?.grade ||
					claim.supporting_event_index !== note.claims[index]?.supporting_event_index)) fail("SOURCE_AUTHORITY", commit);
		console.log(`U7R independent sealed-U7D source authority exact: commit=${commit} tree=${tree} note=${noteBlob} claims=13 candidate_tree=${snapshot.tree}`);
	}
}

function clone(value) { return JSON.parse(JSON.stringify(value)); }

function expectRejected(invoke, token) {
	try { invoke(); } catch (error) { if (String(error.message).includes(token)) return; fail("SELFTEST_WRONG_REJECTION", `${token}:${error.message}`); }
	fail("SELFTEST_FALSE_NEGATIVE", token);
}

function scopeSyntheticDriverDigests(specification) {
	return Object.fromEntries(specification.receipt_contract.study_domains.map((domain) => [domain, sha256(Buffer.from(`scope-driver:${domain}`, "utf8"))]));
}

function scopeSyntheticTimeReport(rss) {
	return `real 0.01\nuser 0.00\nsys 0.00\n${String(rss).padStart(20)}  maximum resident set size\n${"0".padStart(20)}  average shared memory size\n${"0".padStart(20)}  average unshared data size\n${"0".padStart(20)}  average unshared stack size\n${"1".padStart(20)}  page reclaims\n${"0".padStart(20)}  page faults\n${"0".padStart(20)}  swaps\n${"0".padStart(20)}  block input operations\n${"0".padStart(20)}  block output operations\n${"0".padStart(20)}  messages sent\n${"0".padStart(20)}  messages received\n${"0".padStart(20)}  signals received\n${"1".padStart(20)}  voluntary context switches\n${"0".padStart(20)}  involuntary context switches\n${"1".padStart(20)}  instructions retired\n${"1".padStart(20)}  cycles elapsed\n${String(rss).padStart(20)}  peak memory footprint\n`;
}

function scopeSyntheticStudyEvidence(specification) {
	const contract = specification.receipt_contract; const harness = contract.study_harness; const digest = (label) => sha256(Buffer.from(label, "utf8")); const driverDigests = scopeSyntheticDriverDigests(specification); const tools = Object.fromEntries(specification.runtime_authority.admitted_tools.map((tool) => [tool.name, tool]));
	const studies = contract.study_domains.map((id, domainIndex) => {
		const deterministic_artifacts = Object.fromEntries(contract.deterministic_artifact_fields.map((field) => [field, digest(`scope:det:${id}:${field}`)])); const fixture_commit = createHash("sha1").update(`scope:fixture:${id}`).digest("hex"); const fixtureTree = createHash("sha1").update(`scope:tree:${id}`).digest("hex");
		const runs = [1, 2, 3].map((ordinal) => {
			const runRoot = resolve(repositoryRoot, ".countershape/u7d-final/tmp/studies", id, `run-${ordinal}`); const cwd = resolve(runRoot, "fixture"); const binary = resolve(cwd, "countershape"); const execution = { cwd, argv: [binary, "study", id, "--json"], executable_path: binary, executable_sha256: deterministic_artifacts.reference_binary_sha256 };
			const fresh = Object.fromEntries(contract.fresh_run_digest_fields.map((field) => [field, digest(`scope:fresh:${id}:${ordinal}:${field}`)])); fresh.fixture_invocation_sha256 = sha256(Buffer.from(`${JSON.stringify(execution)}\n`, "utf8"));
			const manifest = scopeStudyManifestPaths(contract).map((path) => { const deterministicField = Object.entries(harness.deterministic_artifact_paths).find(([, candidate]) => candidate === path)?.[0]; const freshField = Object.entries(harness.fresh_artifact_paths).find(([, candidate]) => candidate === path)?.[0]; return { path, mode: "0600", bytes: 64, sha256: deterministicField === undefined ? freshField === undefined ? digest(`scope:trial:${id}:${ordinal}:${path}`) : fresh[freshField] : deterministic_artifacts[deterministicField] }; });
			const product_result = { schema_version: harness.product_result_schema, domain: id, ordinal, status: "GREEN" };
			const run = { ordinal, trial_count: 100, trial_budget: 100, observed_subject_process_wall_time_ms: 30, budget_subject_process_wall_time_ms: contract.study_subject_process_wall_time_budget_ms_per_domain / 3, observed_subject_process_peak_rss_bytes: 2_000_000 + domainIndex * 10_000 + ordinal * 100 + 3, budget_subject_process_peak_rss_bytes: contract.study_subject_process_peak_rss_budget_bytes_per_domain, observation_authority: harness.observation_authority,
				fixture: { head_commit: fixture_commit, tree: fixtureTree, commit_count: 1, git_directory: relative(repositoryRoot, resolve(cwd, ".git")), git_common_directory: relative(repositoryRoot, resolve(cwd, ".git")), git_alternates: "ABSENT", clean_status_bytes: 0, clean_status_sha256: sha256(Buffer.alloc(0)) }, driver_authority: { path: harness.driver_by_domain[id], sha256: driverDigests[id] }, execution, processes: {}, phases: contract.study_phase_budgets.map((phase) => ({ id: phase.id, trial_count: phase.trial_count / 3, trial_budget: phase.trial_budget / 3 })), product_result, evidence_root: relative(repositoryRoot, resolve(runRoot, "evidence")), evidence_manifest: manifest, evidence_manifest_sha256: sha256(Buffer.from(`${JSON.stringify(manifest)}\n`, "utf8")), deterministic_artifacts: { ...deterministic_artifacts }, ...fresh };
			for (const [processIndex, name] of ["prepare", "compile", "execute"].entries()) { const rss = 2_000_000 + domainIndex * 10_000 + ordinal * 100 + processIndex + 1; const time_report = scopeSyntheticTimeReport(rss); const reportPath = resolve(runRoot, "observations", `${name}.time`); let expectedCwd; let executable; let executableSha; let args; let stdout;
				if (name === "prepare") { expectedCwd = repositoryRoot; executable = tools.node.path; executableSha = tools.node.sha256; args = [resolve(repositoryRoot, harness.driver_by_domain[id]), "--prepare", "--domain", id, "--ordinal", String(ordinal), "--fixture-root", cwd]; stdout = Buffer.alloc(0); } else if (name === "compile") { expectedCwd = repositoryRoot; executable = tools.go.path; executableSha = tools.go.sha256; args = ["build", "-trimpath", "-mod=readonly", "-buildvcs=false", "-o", binary, "./cmd/countershape"]; stdout = Buffer.alloc(0); } else { expectedCwd = cwd; executable = binary; executableSha = execution.executable_sha256; args = ["study", id, "--json"]; stdout = Buffer.from(`${JSON.stringify(product_result)}\n`, "utf8"); }
				const report = Buffer.from(time_report, "utf8"); run.processes[name] = { name, observer_argv: [tools.time.path, "-p", "-l", "-o", reportPath, executable, ...args], cwd: expectedCwd, executable_path: executable, executable_sha256: executableSha, observer_path: tools.time.path, observer_sha256: tools.time.sha256, environment_sha256: scopeStudyEnvironmentDigest(specification, { id }, run, name), wall_time_ms: 10, peak_rss_bytes: rss, time_report_path: relative(repositoryRoot, reportPath), time_report_bytes: report.length, time_report_sha256: sha256(report), time_report, stdout_bytes: stdout.length, stdout_sha256: sha256(stdout) };
			}
			return run;
		});
		return { id, fixture_commit, fixture_authority: { head_commit: fixture_commit, tree: fixtureTree, clean_status_bytes: 0, clean_status_sha256: sha256(Buffer.alloc(0)), repository_count: 3, fresh_repository_per_run: true, repository_git_directories: runs.map((run) => run.fixture.git_directory) }, logical_product_command: ["countershape", "study", id, "--json"], run_count: 3, trial_count: 300, budget_trial_count: 300, observed_subject_process_wall_time_ms: 90, budget_subject_process_wall_time_ms: contract.study_subject_process_wall_time_budget_ms_per_domain, observed_subject_process_peak_rss_bytes: Math.max(...runs.map((run) => run.observed_subject_process_peak_rss_bytes)), budget_subject_process_peak_rss_bytes: contract.study_subject_process_peak_rss_budget_bytes_per_domain, observation_authority: harness.observation_authority, phases: contract.study_phase_budgets.map((phase) => ({ ...phase })), deterministic_artifacts, runs };
	});
	const versions = Object.fromEntries(specification.runtime_authority.admitted_tools.map((tool) => [tool.name, tool.version]));
	return { schema_version: contract.study_evidence_schema, phase: "U7D", harness_authority: { protocol: harness.protocol, path: harness.path, sha256: harness.sha256, protocol_sha256: harness.protocol_sha256, observation_authority: harness.observation_authority, semantic_ceiling: harness.semantic_ceiling }, environment: { platform: "darwin", kernel_release: "25.0.0", architecture: "arm64", git_version: versions.git, go_version: versions.go, node_version: versions.node, cpu_model: "synthetic scope self-test CPU", logical_cpu_count: 12, memory_bytes: 32 * 1024 ** 3, runtime_authority_sha256: scopeRuntimeDigest(specification) }, studies, milestone_verdict: contract.milestone_verdict, honest_fallback: contract.honest_fallback, unreceipted: [...contract.unreceipted] };
}

export async function selfTest() {
	const specification = await loadSpecification();
	await validateC6BAuthority(specification, specification.sealed_parent.commit);
	const mutations = [
		["SPECIFICATION", (value) => { value.schema_version = "wrong"; }],
		["SPECIFICATION_RECEIPTS", (value) => { value.inherited_receipts.C3 = "ABSENT"; }],
		["SPECIFICATION_RECEIPT_CONTRACT", (value) => { value.receipt_contract.study_run_count_per_domain = 2; }],
		["SPECIFICATION_CLAIM_BINDING", (value) => { value.claim_binding_policy.event_indices = "LATEST"; }],
		["SPECIFICATION_CLAIM_BINDING", (value) => { value.claim_binding_policy.recorded_child_process_intervals = "ALLOW_OVERLAP"; }],
		["SPECIFICATION_CLAIM_BINDING", (value) => { value.claim_binding_policy.operational_writer_protocol = "MULTI_WRITER"; }],
		["SPECIFICATION_UNIT", (value) => { value.units[0].allowed_paths.push(value.units[0].allowed_paths[0]); }],
		["SPECIFICATION_UNIT", (value) => { value.units[1].required_paths.push("outside.txt"); }],
		["SPECIFICATION_ROSTER", (value) => { value.units[0].allowed_paths.pop(); value.units[0].required_paths.pop(); }],
		["SPECIFICATION_UNIT", (value) => { value.units.at(-1).product_authority = "U7_REFERENCE_APPLICATION"; }],
		["SPECIFICATION_UNIT", (value) => { value.units[2].parent = "U7P"; }],
		["SPECIFICATION_CLAIMS", (value) => { value.units[3].claims[0].label += " drift"; }],
	];
	for (const [token, mutate] of mutations) {
		const candidate = clone(specification); mutate(candidate); expectRejected(() => validateScopeSpecification(candidate), token);
	}
	const intervalEntries = (intervals) => intervals.map(([started_at, ended_at]) => ({ event: { started_at, ended_at } }));
	validateRecordedChildIntervals(intervalEntries([[1, 2], [2, 2], [3.5, 4]]), "selftest-positive");
	const intervalHostiles = [
		["RECORDED_CHILD_INTERVAL_OVERLAP", intervalEntries([[1, 3], [2, 4], [4, 5]])],
		["RECORDED_CHILD_INTERVAL_OVERLAP", intervalEntries([[1, 2], [2, 4], [3.5, 5]])],
		["RECORDED_CHILD_INTERVAL", [{ event: { ended_at: 2 } }]],
		["RECORDED_CHILD_INTERVAL", intervalEntries([[1, Number.NaN]])],
		["RECORDED_CHILD_INTERVAL", intervalEntries([[3, 2]])],
	];
	for (const [token, entries] of intervalHostiles) {
		try { validateRecordedChildIntervals(entries, "selftest-hostile"); }
		catch (error) { if (String(error.message).includes(token)) continue; fail("SELFTEST_INTERVAL_WRONG_REJECTION", `${token}:${error.message}`); }
		fail("SELFTEST_INTERVAL_FALSE_NEGATIVE", token);
	}
	const unit = specification.units[1];
	expectRejected(() => validateRoster(unit, [unit.required_paths[0]]), "ROSTER");
	expectRejected(() => validateRoster(specification.units[0], [...u7pExactPaths].slice(1)), "ROSTER");
	expectRejected(() => validateModes(["a.go"], [{ path: "a.go", mode: "120000", stage: 0, object: "1".repeat(40) }]), "INDEX_MODE");
	const cleanCredentials = [{ path: "docs/example.md", text: "credential examples use named placeholders only" }];
	if (credentialPatternFindings(cleanCredentials).length !== 0) fail("SELFTEST_CREDENTIAL_CLEAN", "false positive");
	const credentialFixtures = [
		["-----BEGIN ", "PRIVATE KEY", "-----"].join(""), `AKIA${"A".repeat(16)}`, `ghp_${"a".repeat(30)}`,
		`glpat-${"a".repeat(20)}`, `xoxb-${"a".repeat(20)}`, `AIza${"a".repeat(35)}`,
		`sk-proj-${"a".repeat(20)}`, `sk_live_${"a".repeat(16)}`, `Bearer eyJ${"a".repeat(10)}.${"b".repeat(10)}.${"c".repeat(10)}`,
	];
	for (const [index, text] of credentialFixtures.entries()) {
		const findings = credentialPatternFindings([{ path: `fixture-${index}`, text }]);
		if (findings.length !== 1 || findings[0].pattern !== credentialPatterns[index].name) fail("SELFTEST_CREDENTIAL_HOSTILE", String(index));
	}
	const studyEvidence = scopeSyntheticStudyEvidence(specification);
	const studyBytes = Buffer.from(`${JSON.stringify(studyEvidence)}\n`, "utf8");
	const studyDigest = sha256(studyBytes);
	const receipt = {
		// Synthetic-only declaration; no local snapshot authority is implied.
		schema_version: "countershape/u7-receipt/v1", source_boundary: "U7D", source_commit: "1".repeat(40),
		source_tree: "2".repeat(40), source_parent: "3".repeat(40), source_subject: "test: close U7 reference study evidence",
		source_note_blob: "4".repeat(40), source_note_body_sha256: "5".repeat(64),
		source_claims: specification.units.find((row) => row.id === "U7D").claims.map((claim, index) => ({ index, supporting_event_index: index, type: claim.type, label: claim.label, pathspecs: [...specification.units.find((row) => row.id === "U7D").allowed_paths], grade: "tree-exact" })),
		source_strict_exit: 0, source_secrets_override: false, source_html_path: ".countershape/evidence/u7d-final-111111111111.html", source_html_bytes: 4096,
		source_html_sha256: "6".repeat(64), source_html_authority: "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS",
		source_ledger_archive: ".didrun-history/u7d-final-111111111111/.didrun", source_ledger_authority: "LOCAL_SECRET_BEARING_SNAPSHOT_NOT_PORTABLE",
		source_study_event_index: scopeStudyEventIndex(specification), source_study_stdout_blob: studyDigest,
		source_study_stdout_bytes: studyBytes.length, source_study_evidence_sha256: studyDigest, source_study_evidence: studyEvidence,
	};
	receipt.source_ledger_manifest = [
		{ path: ".gitignore", bytes: 8, sha256: "6".repeat(64) },
		{ path: "claims.jsonl", bytes: 1024, sha256: "7".repeat(64) },
		{ path: `objects/${"8".repeat(64)}`, bytes: 64, sha256: "8".repeat(64) },
		{ path: `objects/${studyDigest}`, bytes: studyBytes.length, sha256: studyDigest },
		{ path: "seals.jsonl", bytes: 128, sha256: "9".repeat(64) },
		{ path: "session.log", bytes: 2048, sha256: "a".repeat(64) },
	].sort((left, right) => left.path < right.path ? -1 : left.path > right.path ? 1 : 0);
	receipt.source_ledger_file_count = receipt.source_ledger_manifest.length;
	receipt.source_ledger_event_count = 14;
	receipt.source_ledger_claim_count = 13;
	receipt.source_ledger_seal_count = 1;
	receipt.source_ledger_manifest_sha256 = createHash("sha256").update(`${JSON.stringify(receipt.source_ledger_manifest)}\n`, "utf8").digest("hex");
	const driverDigests = scopeSyntheticDriverDigests(specification);
	validateReceiptDeclaration(receipt, specification, driverDigests);
	for (const [token, mutate] of [
		["RECEIPT_SHAPE", (value) => { value.extra = true; }],
		["RECEIPT_OID", (value) => { value.source_tree = "0".repeat(40); }],
		["RECEIPT_FIELDS", (value) => { value.source_claims.pop(); }],
		["RECEIPT_LEDGER_AUTHORITY", (value) => { value.source_ledger_event_count = 13; }],
		["RECEIPT_LEDGER_MANIFEST", (value) => { value.source_ledger_manifest.reverse(); }],
		["RECEIPT_CLAIM", (value) => { value.source_claims[0].grade = "stale"; }],
		["RECEIPT_CLAIM_PATHS", (value) => { value.source_claims[0].pathspecs.pop(); }],
		["RECEIPT_STUDY_SHAPE", (value) => { value.source_study_evidence.unreceipted.pop(); }],
		["RECEIPT_STUDY_ENVIRONMENT", (value) => { value.source_study_evidence.environment.node_version = "v0.0.0"; }],
		["RECEIPT_STUDY_DOMAIN", (value) => { value.source_study_evidence.studies[0].phases[0].trial_budget = 4; }],
		["RECEIPT_STUDY_RUN", (value) => { value.source_study_evidence.studies[0].runs[0].phases[0].trial_budget += 1; }],
		["RECEIPT_STUDY_RUN", (value) => { value.source_study_evidence.studies[1].runs[1].phases[3].trial_count = 0; }],
		["RECEIPT_STUDY_RUN", (value) => { value.source_study_evidence.studies[1].runs[2].deterministic_artifacts.ruling_sha256 = "e".repeat(64); }],
		["RECEIPT_STUDY_EXECUTION", (value) => { value.source_study_evidence.studies[1].runs[2].execution.cwd += "-drift"; }],
		["RECEIPT_STUDY_PROCESS", (value) => { value.source_study_evidence.studies[0].runs[0].processes.compile.executable_sha256 = "e".repeat(64); }],
		["RECEIPT_STUDY_MANIFEST_DIGEST", (value) => { value.source_study_evidence.studies[0].runs[0].evidence_manifest_sha256 = "e".repeat(64); }],
		["RECEIPT_STUDY_FRESH", (value) => { const target = value.source_study_evidence.studies[1].runs[0]; target.attempts_sha256 = value.source_study_evidence.studies[0].runs[0].attempts_sha256; target.evidence_manifest.find((row) => row.path === "fresh/attempts.json").sha256 = target.attempts_sha256; target.evidence_manifest_sha256 = sha256(Buffer.from(`${JSON.stringify(target.evidence_manifest)}\n`, "utf8")); }],
		["RECEIPT_STUDY_AUTHORITY", (value) => { value.source_study_stdout_bytes += 1; }],
		["RECEIPT_STUDY_MANIFEST", (value) => { value.source_ledger_manifest = value.source_ledger_manifest.filter((entry) => entry.path !== `objects/${value.source_study_stdout_blob}`); value.source_ledger_file_count -= 1; value.source_ledger_manifest_sha256 = sha256(Buffer.from(`${JSON.stringify(value.source_ledger_manifest)}\n`, "utf8")); }],
	]) {
		const candidate = clone(receipt); mutate(candidate); expectRejected(() => validateReceiptDeclaration(candidate, specification, driverDigests), token);
	}
	const prefix = hermeticPrefix(specification.units[0]);
	const rawPreview = [...prefix, ...specification.units[0].claims[0].command];
	const scrubbedPreview = [...rawPreview];
	for (const position of noteRootRedactionPositions) {
		const partial = expectedRedactions(position, rawPreview[position]).find((value) => value !== `${noteRedaction}.${noteRedaction}`);
		if (partial === undefined) fail("SELFTEST_REDACTION_FIXTURE", String(position));
		scrubbedPreview[position] = partial;
	}
	for (const position of noteBareRedactionPositions) scrubbedPreview[position] = noteRedaction;
	if (!claimPreviewMatches(scrubbedPreview, rawPreview, prefix)) fail("SELFTEST_REDACTION_FIXTURE", "future positive");
	const hostilePreview = [...scrubbedPreview]; hostilePreview[9] = noteRedaction;
	if (claimPreviewMatches(hostilePreview, rawPreview, prefix)) fail("SELFTEST_REDACTION_FIXTURE", "wrong-position accepted");
	console.log(`U7 independent staged-scope defensive self-test passed: ${mutations.length + 7} schema/roster/mode/receipt refusals, ${intervalHostiles.length} recorded child-process interval refusals, ${credentialFixtures.length} credential-pattern controls, and clean placeholder control`);
}

async function main() {
	if (isDeepStrictEqual(process.argv.slice(2), ["--self-test"])) return selfTest();
	if (process.argv.length !== 5 || process.argv[2] !== "--unit") fail("USAGE", "check-u7-scope.mjs --unit <unit> <--candidate-phase|--final-gate|--receipt-final-gate|--credential-scan|--gofmt|--source-authority-gate> | --self-test");
	const specification = await loadSpecification();
	const id = process.argv[3];
	switch (process.argv[4]) {
		case "--candidate-phase": return runCandidatePhase(specification, id);
		case "--final-gate": return runFinalGate(specification, id, false);
		case "--receipt-final-gate": return runFinalGate(specification, id, true);
		case "--credential-scan": return runCredentialScan(specification, id);
		case "--gofmt": return runGofmt(specification, id);
		case "--source-authority-gate": return runSourceAuthorityGate(specification, id);
		default: fail("USAGE", process.argv[4]);
	}
}

async function dispatch() {
	if (process.argv[1] === undefined) return;
	const requested = resolve(process.argv[1]);
	let actual;
	try { actual = await realpath(requested); } catch { return; }
	const canonical = fileURLToPath(import.meta.url);
	if (actual !== canonical) return;
	if (requested !== canonical) fail("NONCANONICAL_ENTRY", requested);
	await main();
}

dispatch().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
