#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { constants } from "node:fs";
import { chmod, mkdir, mkdtemp, lstat, open, readdir, realpath, rename, rm, symlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { isAbsolute, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { isDeepStrictEqual } from "node:util";

import { checkPlan as checkInheritedP07Plan } from "./check-p07b-c-plan.mjs";

export const repositoryRoot = resolve(fileURLToPath(new URL("..", import.meta.url)));
export const specificationPath = resolve(repositoryRoot, "spec/verification/u7-unit-paths.json");
const handoffPath = resolve(repositoryRoot, "docs/HANDOFF_MODE_C.md");
const didrunPath = "/opt/homebrew/bin/didrun";
const gitPath = "/usr/bin/git";
const unitOrder = Object.freeze(["U7P", "U7M", "U7A", "U7B", "U7C", "U7D", "U7R"]);
const expectedParents = Object.freeze({ U7P: "C6B", U7M: "U7P", U7A: "U7M", U7B: "U7A", U7C: "U7B", U7D: "U7C", U7R: "U7D" });
const expectedProfiles = Object.freeze({
	U7P: "SOURCE_FULL", U7M: "SOURCE_FULL", U7A: "SOURCE_FULL", U7B: "SOURCE_FULL",
	U7C: "SOURCE_FULL", U7D: "SOURCE_FULL", U7R: "RECEIPT_RECONCILIATION",
});
const expectedProductAuthority = Object.freeze({
	U7P: "NONE", U7M: "NONE", U7A: "U7_REFERENCE_APPLICATION", U7B: "U7_REFERENCE_APPLICATION",
	U7C: "U7_REFERENCE_APPLICATION", U7D: "U7_REFERENCE_APPLICATION", U7R: "NONE",
});
const expectedProductBehavior = Object.freeze({
	U7P: "INHERITED_UNREPROVEN", U7M: "INHERITED_UNREPROVEN", U7A: "CANDIDATE_UNRECEIPTED", U7B: "CANDIDATE_UNRECEIPTED",
	U7C: "CANDIDATE_UNRECEIPTED", U7D: "CANDIDATE_UNRECEIPTED", U7R: "SOURCE_RECEIPT_RECONCILIATION",
});
const expectedSubjects = Object.freeze({
	U7P: "chore: lock U7 execution authority",
	U7M: "fix: adapt inherited P07 receipt verification",
	U7A: "feat: add U7 reference CLI foundation",
	U7B: "feat: add U7 HTTP falsification study",
	U7C: "feat: add U7 CLI decision and contract flow",
	U7D: "test: close U7 reference study evidence",
	U7R: "docs: receipt U7 reference milestone",
});
const expectedFinalRoots = Object.freeze({
	U7P: ".countershape/u7p-final", U7M: ".countershape/u7m-final", U7A: ".countershape/u7a-final",
	U7B: ".countershape/u7b-final", U7C: ".countershape/u7c-final",
	U7D: ".countershape/u7d-final", U7R: ".countershape/u7r-final",
});
const expectedClaimCounts = Object.freeze({ U7P: 13, U7M: 13, U7A: 12, U7B: 12, U7C: 12, U7D: 13, U7R: 9 });
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
const expectedTransitionAuthority = Object.freeze({
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
	product_authority: "NONE",
	product_behavior: "INHERITED_UNREPROVEN",
	grade_transfer: "NONE",
});
const expectedTopologyAmendmentAuthority = Object.freeze({
	boundary: "U7M",
	source: "OWNER_OUT_OF_BAND",
	classification: "OWNER_AUTHORIZED_AUTHORITY_MIGRATION_DEFECT_REPAIR",
	provenance: Object.freeze({
		kind: "UNEVIDENCED",
		disclosure: "OWNER_ATTRIBUTED_SESSION_INSTRUCTION_ONLY_NO_QUALIFYING_PREEXISTING_ARTIFACT",
	}),
	authentication: "NOT_ESTABLISHED",
	signed_authorization: "NOT_IMPLEMENTED",
	predecessor_declaration: Object.freeze({ kind: "NONE" }),
	defect: "P07_C3P_RECEIPT_LIVE_ENTRYPOINT_REQUIRES_C6R_OR_C6B_HEAD_SUBJECT_AND_REJECTS_ALL_U7_DESCENDANTS",
	product_authority: "NONE",
	product_behavior: "INHERITED_UNREPROVEN",
	grade_transfer: "NONE",
});
const expectedRuntimeAuthority = Object.freeze({
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
const expectedInheritedReceipts = Object.freeze({ C3P: "PRESENT", C3: "PRESENT", C6A: "PRESENT" });
const expectedReceiptContract = Object.freeze(JSON.parse(String.raw`{"schema_version":"countershape/u7-receipt/v1","source_boundary":"U7D","study_event_command":["/opt/homebrew/bin/node","tools/check-u7-study-harness.mjs","--phase","U7D"],"study_evidence_schema":"countershape/u7-study-evidence/v1","study_execution_authority":"U7P_FROZEN_HARNESS_DIRECT_PROCESS_GIT_AND_ARTIFACT_OBSERVATION","study_harness":{"protocol":"countershape/u7-study-harness/v1","protocol_sha256":"faca5cac53ce4ca4d4edea3089edd1fad1139ff847da9618115d1e308b717678","path":"tools/check-u7-study-harness.mjs","sha256":"d6a39e44e6a4de43268dab9ff2deac8350b0b4e107b01e254d2767f0692a4121","product_result_schema":"countershape/u7-study-domain-result/v1","trial_schema":"countershape/u7-study-trial/v1","artifact_schema":"countershape/u7-study-artifact/v1","observation_authority":"U7P_FROZEN_HARNESS_DIRECT_PROCESS_GIT_AND_ARTIFACT_OBSERVATION","semantic_ceiling":"ARTIFACT_BYTES_AND_SUBJECT_PROCESS_TOPOLOGY_OBSERVED_PRODUCT_SEMANTICS_AND_FULL_HARNESS_RESOURCES_NOT_INDEPENDENTLY_ESTABLISHED","driver_protocol":"FIXTURE_ONLY_NO_EVIDENCE_ROOT","phase_verdicts":{"U7B":"LOCAL_HTTP_REFERENCE_FUNCTIONAL_GREEN_SUBJECT_RESOURCE_OBSERVED","U7C":"LOCAL_CLI_REFERENCE_FUNCTIONAL_GREEN_SUBJECT_RESOURCE_OBSERVED","U7D":"LOCAL_REFERENCE_MILESTONE_FUNCTIONAL_GREEN_FULL_STUDY_RESOURCE_UNRECEIPTED"},"observer_prefix":["/usr/bin/time","-p","-l","-o"],"driver_by_domain":{"http":"tools/run-u7-http-study.mjs","cli":"tools/run-u7-cli-study.mjs"},"phase_budgets":[{"id":"fixture","per_run_trials":1},{"id":"compile","per_run_trials":1},{"id":"search","per_run_trials":80},{"id":"confirm","per_run_trials":8},{"id":"contract","per_run_trials":10}],"deterministic_artifact_paths":{"source_spec_sha256":"deterministic/source-spec.json","world_plan_sha256":"deterministic/world-plan.json","ruling_sha256":"deterministic/ruling.json","decision_record_sha256":"deterministic/decision-record.json","contract_bundle_sha256":"deterministic/contract-bundle.json"},"fresh_artifact_paths":{"world_instance_sha256":"fresh/world-instance.json","attempts_sha256":"fresh/attempts.json","measurements_sha256":"fresh/measurements.json","captures_sha256":"fresh/captures.json","confirmation_sha256":"fresh/confirmation.json","contract_execution_target_sha256":"fresh/contract-execution-target.json","finalized_contract_run_sha256":"fresh/finalized-contract-run.json","contract_execution_sha256":"fresh/contract-execution.json"}},"study_domains":["http","cli"],"study_run_count_per_domain":3,"study_trial_budget_per_domain":300,"study_subject_process_wall_time_budget_ms_per_domain":900000,"study_subject_process_peak_rss_budget_bytes_per_domain":4294967296,"study_phase_budgets":[{"id":"fixture","trial_count":3,"trial_budget":3},{"id":"compile","trial_count":3,"trial_budget":3},{"id":"search","trial_count":240,"trial_budget":240},{"id":"confirm","trial_count":24,"trial_budget":24},{"id":"contract","trial_count":30,"trial_budget":30}],"deterministic_artifact_fields":["source_spec_sha256","world_plan_sha256","ruling_sha256","decision_record_sha256","contract_bundle_sha256","reference_binary_sha256"],"fresh_run_digest_fields":["world_instance_sha256","attempts_sha256","measurements_sha256","captures_sha256","confirmation_sha256","fixture_invocation_sha256","contract_execution_target_sha256","finalized_contract_run_sha256","contract_execution_sha256"],"html_authority":"LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS","ledger_authority":"LOCAL_SECRET_BEARING_SNAPSHOT_NOT_PORTABLE","projection_paths":["docs/HANDOFF_MODE_C.md","docs/status/U7D-EVIDENCE.md"],"projection_policy":"EXACT_PARENT_RELATIVE_PENDING_BLOCK_REPLACEMENT","self_receipt":"ABSENT","milestone_verdict":"LOCAL_REFERENCE_MILESTONE_FUNCTIONAL_GREEN_FULL_STUDY_RESOURCE_UNRECEIPTED","honest_fallback":"NOT_APPLICABLE_SOURCE_GREEN","unreceipted":["LINUX_UNRUN","WINDOWS_UNRUN","UNRUN_NODE_MAJORS","BROAD_IMPORTED_REPOSITORY_BEHAVIOR_UNVALIDATED","HOSTILE_CONTAINMENT_UNVALIDATED","NETWORK_DENIAL_UNVALIDATED","COMPREHENSION_UNVALIDATED","REVIEW_COMPRESSION_UNVALIDATED","ADOPTION_UNVALIDATED","MAINTAINABILITY_UNVALIDATED","PRODUCTION_READINESS_UNVALIDATED","SECURITY_REVIEW_NOT_PERFORMED","EXTERNAL_PLATFORM_BEHAVIOR_UNVALIDATED","IMPORTED_REPOSITORY_TIMING_UNCLAIMED","FULL_STUDY_RESOURCE_BOUND_UNVALIDATED","U7R_SELF_RECEIPT_ABSENT"]}`));
const expectedClaimBindingPolicy = Object.freeze({
	event_indices: "EXPLICIT_ZERO_BASED_EVENT_INDEX_EQUALS_CLAIM_INDEX",
	pathspecs: "EXACT_UNIT_ALLOWED_PATHS_IN_DECLARED_ORDER",
	recorded_child_process_intervals: "FINITE_NONREVERSED_LEDGER_ORDER_START_AT_OR_AFTER_PREVIOUS_END_INCLUDING_FINAL_NOTE_INSPECTION",
	operational_writer_protocol: "ONE_ROOT_OWNED_WRITER_AND_WRAPPER_PLUS_RELEVANT_CHILD_PROCESS_TERMINAL_BEFORE_NEXT_FENCE_OR_MUTATION",
});
const expectedSealedParent = Object.freeze({
	boundary: "C6B",
	commit: "4cef12b38cfcd857593a21db5952f2dfb2dfc274",
	tree: "b50aee79f41c3f498524d89d425d6dd8f4009702",
	parent: "fafff150d23d6df211b4313e73ac7cf43f28b2d3",
	subject: "docs: receipt P07B-C contract execution",
	note_ref: "refs/notes/didrun",
	note_blob: "a9f3f5484caf00f35e254f71549a754d3de3bd4b",
	note_body_sha256: "355355cd57254802dbc661c4fc9faaf8c695dfbbc787c0175b085f2709f532b3",
	claim_count: 9,
	claim_coverage: "9/9",
	grade: "tree-exact",
	strict_exit: 0,
	secrets_override: true,
	html_path: ".countershape/evidence/p07b-c-c6b-final-4cef12b38cfc.html",
	html_sha256: "006e298a8cd8556bb32cee4229df60e3735b102a96c54878c3cf2205cbb48070",
	html_authority: "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS",
	ledger_archive: ".didrun-history/p07b-c-c6b-final-4cef12b38cfc/.didrun",
	ledger_authority: "LOCAL_SECRET_BEARING_SNAPSHOT_NOT_PORTABLE", ledger_file_count: 14, ledger_bytes: 38492,
	ledger_manifest_sha256: "a09598654ee38382ea44201a7b4609b66c1bee192d1dcd8fc747508527122e37",
	ledger_event_count: 10, ledger_claim_count: 9, ledger_seal_count: 1,
});
const expectedU7PClaims = Object.freeze([
	Object.freeze({ type: "tests-pass", label: "U7P candidate plan and sealed-C6B parent authority", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-plan.mjs", "--check-candidate", "U7P"]) }),
	Object.freeze({ type: "tests-pass", label: "U7P independent candidate transition and exact staged authority", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-scope.mjs", "--unit", "U7P", "--candidate-phase"]) }),
	Object.freeze({ type: "tests-pass", label: "U7P plan contract defensive self-test", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-plan.mjs", "--self-test"]) }),
	Object.freeze({ type: "tests-pass", label: "U7P independent staged-scope defensive self-test", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-scope.mjs", "--self-test"]) }),
	Object.freeze({ type: "tests-pass", label: "U7P final runbook renderer defensive self-test", command: Object.freeze(["/opt/homebrew/bin/node", "tools/print-u7-final-runbook.mjs", "--self-test"]) }),
	Object.freeze({ type: "tests-pass", label: "U7P dormant future-surface architecture conformance", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-architecture.mjs", "--phase", "U7P"]) }),
	Object.freeze({ type: "tests-pass", label: "U7P architecture authority defensive self-test", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-architecture-selftest.mjs", "--phase", "U7P"]) }),
	Object.freeze({ type: "tests-pass", label: "U7P study-harness protocol defensive self-test", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-study-harness-selftest.mjs", "--phase", "U7P"]) }),
	Object.freeze({ type: "tests-pass", label: "U7P cumulative verifier defensive self-test", command: Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]) }),
	Object.freeze({ type: "tests-pass", label: "U7P cumulative verification on the exact staged candidate", command: Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]) }),
	Object.freeze({ type: "command-succeeded", label: "U7P exact seventeen-path staged scope and diff integrity", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-scope.mjs", "--unit", "U7P", "--final-gate"]) }),
	Object.freeze({ type: "command-succeeded", label: "U7P scoped staged credential-pattern scan", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-scope.mjs", "--unit", "U7P", "--credential-scan"]) }),
	Object.freeze({ type: "command-succeeded", label: "U7P sealed-C6B predecessor and preceding didrun chain integrity", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-plan.mjs", "--verify-preseal", "U7P"]) }),
]);
const u7mExactPaths = Object.freeze([
	"docs/ARCHITECTURE.md",
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/STATE_MACHINES.md",
	"docs/VERIFICATION.md",
	"docs/prompts/P08-U7-CLI-REFERENCE-STUDIES.md",
	"docs/status/U7M-P07-RECEIPT-SUCCESSOR-COMPATIBILITY.md",
	"spec/verification/u7-unit-paths.json",
	"tools/check-u7-architecture.mjs",
	"tools/check-u7-architecture-selftest.mjs",
	"tools/check-u7-plan.mjs",
	"tools/check-u7-scope.mjs",
	"tools/verify-current.mjs",
	"tools/verify-current-selftest.mjs",
]);
const expectedU7MClaims = Object.freeze([
	Object.freeze({ type: "tests-pass", label: "U7M candidate plan and sealed-U7P parent authority", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-plan.mjs", "--check-candidate", "U7M"]) }),
	Object.freeze({ type: "tests-pass", label: "U7M independent candidate transition and exact staged authority", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-scope.mjs", "--unit", "U7M", "--candidate-phase"]) }),
	Object.freeze({ type: "tests-pass", label: "U7M inherited P07 receipt successor compatibility", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-plan.mjs", "--verify-inherited-p07-compatibility"]) }),
	Object.freeze({ type: "tests-pass", label: "U7M plan contract defensive self-test", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-plan.mjs", "--self-test"]) }),
	Object.freeze({ type: "tests-pass", label: "U7M independent staged-scope defensive self-test", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-scope.mjs", "--self-test"]) }),
	Object.freeze({ type: "tests-pass", label: "U7M final runbook renderer defensive self-test", command: Object.freeze(["/opt/homebrew/bin/node", "tools/print-u7-final-runbook.mjs", "--self-test"]) }),
	Object.freeze({ type: "tests-pass", label: "U7M zero-product-surface architecture conformance", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-architecture.mjs", "--phase", "U7M"]) }),
	Object.freeze({ type: "tests-pass", label: "U7M architecture authority defensive self-test", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-architecture-selftest.mjs", "--phase", "U7M"]) }),
	Object.freeze({ type: "tests-pass", label: "U7M cumulative verifier defensive self-test", command: Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]) }),
	Object.freeze({ type: "tests-pass", label: "U7M cumulative verification on the exact staged maintenance candidate", command: Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]) }),
	Object.freeze({ type: "command-succeeded", label: "U7M exact fourteen-path staged scope and diff integrity", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-scope.mjs", "--unit", "U7M", "--final-gate"]) }),
	Object.freeze({ type: "command-succeeded", label: "U7M scoped staged credential-pattern scan", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-scope.mjs", "--unit", "U7M", "--credential-scan"]) }),
	Object.freeze({ type: "command-succeeded", label: "U7M sealed-U7P predecessor and preceding didrun chain integrity", command: Object.freeze(["/opt/homebrew/bin/node", "tools/check-u7-plan.mjs", "--verify-preseal", "U7M"]) }),
]);
const capsuleStart = "<!-- U7-PHASE:START -->";
const capsuleEnd = "<!-- U7-PHASE:END -->";
const inheritedP07MarkerBases = Object.freeze([
	"P07B-C-C0A-RECEIPTS", "P07B-C-C0B-RECEIPTS", "P07B-C-C1-MAINTENANCE-RECEIPTS",
	"P07B-C-C1-SOURCE-RECEIPTS", "P07B-C-C1E-RECEIPTS", "P07B-C-C1V-RECEIPTS",
	"P07B-C-C2-SOURCE-RECEIPTS", "P07B-C-C3-SOURCE-RECEIPTS", "P07B-C-C3P-SOURCE-RECEIPTS",
	"P07B-C-C6A-SOURCE-RECEIPTS", "P07B-C-RECEIPT-PHASE",
]);
const inheritedP07ProtectedPaths = Object.freeze([
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-c3p-receipt.mjs",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
]);
const inheritedP07C6AuthorityPaths = Object.freeze([
	"spec/verification/p07b-c-c6a-source-authority.json",
	"spec/verification/p07b-c-c6a-receipt.json",
	"docs/captures/p07b-c/c6a-evidence-summary.json",
	"docs/status/P07B-C-C6-EVIDENCE.md",
]);
const inheritedP07C6AuthoritySchema = "countershape/p07b-c-c6a-source-authority/v1";
const inheritedP07C6SourceSubject = "test: close P07B-C cumulative evidence";
const maximumTextBytes = 4 * 1024 * 1024;

function fail(code, detail) {
	throw new Error(`U7_PLAN_${code}: ${detail}`);
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

function uniqueStrings(values) {
	return Array.isArray(values) && values.length > 0 && values.every((value) => typeof value === "string") &&
		new Set(values).size === values.length;
}

function validCommand(command) {
	return Array.isArray(command) && command.length >= 2 && command.length <= 32 &&
		command.every((part) => typeof part === "string" && part.length > 0 && part.length <= 4096 && !/[\u0000\r\n]/u.test(part)) &&
		isAbsolute(command[0]);
}

function normalizeClaims(claims) {
	return claims.map((claim) => ({ type: claim.type, label: claim.label, command: [...claim.command] }));
}

function occurrenceCount(text, value) {
	return text.split(value).length - 1;
}

export function visibleMarkdownAuthorityBody(body) {
	const visible = [];
	let htmlComment = false;
	let fence;
	for (const originalLine of body.split("\n")) {
		if (fence !== undefined) {
			visible.push("");
			const close = new RegExp(`^ {0,3}${fence.character === "`" ? "`" : "~"}{${fence.length},}\\s*$`, "u");
			if (close.test(originalLine)) fence = undefined;
			continue;
		}
		let line = "";
		let cursor = 0;
		while (cursor < originalLine.length) {
			if (htmlComment) {
				const end = originalLine.indexOf("-->", cursor);
				if (end === -1) { cursor = originalLine.length; continue; }
				htmlComment = false;
				cursor = end + 3;
				continue;
			}
			const start = originalLine.indexOf("<!--", cursor);
			if (start === -1) { line += originalLine.slice(cursor); cursor = originalLine.length; continue; }
			line += originalLine.slice(cursor, start);
			htmlComment = true;
			cursor = start + 4;
		}
		if (/^(?: {4}|\t)/u.test(line)) { visible.push(""); continue; }
		if (/^ {0,3}<(?:\/?[A-Za-z][A-Za-z0-9-]*(?:\s|\/?>)|\?|![A-Z]|!\[CDATA\[)/u.test(line)) fail("MARKDOWN_RAW_HTML", line.slice(0, 128));
		const opening = /^ {0,3}(`{3,}|~{3,})/u.exec(line);
		if (opening) { fence = { character: opening[1][0], length: opening[1].length }; visible.push(""); continue; }
		visible.push(line);
	}
	if (htmlComment) fail("MARKDOWN_HTML_COMMENT", "unterminated");
	if (fence !== undefined) fail("MARKDOWN_FENCE", "unterminated");
	return visible.join("\n");
}

function requireVisibleMarkers(body, startMarker, endMarker, label) {
	const startSentinel = `COUNTERSHAPE_VISIBLE_${label}_START`;
	const endSentinel = `COUNTERSHAPE_VISIBLE_${label}_END`;
	if (body.includes(startSentinel) || body.includes(endSentinel)) fail("MARKDOWN_SENTINEL", label);
	const visible = visibleMarkdownAuthorityBody(body.replace(startMarker, startSentinel).replace(endMarker, endSentinel));
	if (occurrenceCount(visible, startSentinel) !== 1 || occurrenceCount(visible, endSentinel) !== 1 ||
		visible.indexOf(startSentinel) >= visible.indexOf(endSentinel)) fail("MARKDOWN_VISIBILITY", label);
}

function exactVisibleMarkedBlock(body, base, label) {
	const start = `<!-- ${base}:START -->`;
	const end = `<!-- ${base}:END -->`;
	if (occurrenceCount(body, start) !== 1 || occurrenceCount(body, end) !== 1) fail("P07_COMPATIBILITY_MARKERS", `${label}:${base}`);
	const startIndex = body.indexOf(start); const endIndex = body.indexOf(end, startIndex);
	if (startIndex < 0 || endIndex <= startIndex || (startIndex !== 0 && body[startIndex - 1] !== "\n") ||
		body[startIndex + start.length] !== "\n" || body[endIndex - 1] !== "\n" ||
		!["", "\n"].includes(body.slice(endIndex + end.length, endIndex + end.length + 1))) {
		fail("P07_COMPATIBILITY_FRAMING", `${label}:${base}`);
	}
	requireVisibleMarkers(body, start, end, `P07_${base.replaceAll("-", "_")}`);
	return body.slice(startIndex, endIndex + end.length);
}

export function validateInheritedP07Compatibility(parentHandoff, candidateHandoff, parentProtected, candidateProtected) {
	if (typeof parentHandoff !== "string" || typeof candidateHandoff !== "string" ||
		!(parentProtected instanceof Map) || !(candidateProtected instanceof Map) ||
		!isDeepStrictEqual([...parentProtected.keys()], inheritedP07ProtectedPaths) ||
		!isDeepStrictEqual([...candidateProtected.keys()], inheritedP07ProtectedPaths)) fail("P07_COMPATIBILITY_INPUT", "exact protected roster");
	for (const base of inheritedP07MarkerBases) {
		const parent = exactVisibleMarkedBlock(parentHandoff, base, "parent");
		const candidate = exactVisibleMarkedBlock(candidateHandoff, base, "candidate");
		if (candidate !== parent) fail("P07_COMPATIBILITY_BLOCK", base);
	}
	const markerPattern = /^<!-- (P07B-C-[A-Z0-9-]+):(START|END) -->$/gmu;
	const markerRoster = (body) => [...body.matchAll(markerPattern)].map((match) => `${match[1]}:${match[2]}`).sort();
	if (!isDeepStrictEqual(markerRoster(parentHandoff), markerRoster(candidateHandoff)) ||
		markerRoster(parentHandoff).length !== inheritedP07MarkerBases.length * 2) fail("P07_COMPATIBILITY_MARKER_ROSTER", "parent/candidate");
	for (const path of inheritedP07ProtectedPaths) {
		const parent = parentProtected.get(path); const candidate = candidateProtected.get(path);
		if (!Buffer.isBuffer(parent) || !Buffer.isBuffer(candidate) || !candidate.equals(parent)) fail("P07_COMPATIBILITY_PROTECTED_PATH", path);
	}
	return Object.freeze({ blocks: inheritedP07MarkerBases.length, protectedPaths: inheritedP07ProtectedPaths.length });
}

function sectionBounds(body, heading) {
	if (occurrenceCount(body, heading) !== 1) fail("MARKDOWN_HEADING", heading.trim());
	const start = body.indexOf(heading) + heading.length;
	const next = body.indexOf("\n## ", start);
	return Object.freeze({ start, end: next === -1 ? body.length : next + 1 });
}

export function validateSpecification(value) {
	if (!exactKeys(value, ["schema_version", "transition_authority", "topology_amendment_authority", "runtime_authority", "inherited_receipts", "receipt_contract", "claim_binding_policy", "sealed_parent", "units"])) fail("SPEC_KEYS", "top-level keys");
	if (value.schema_version !== "countershape/u7-unit-paths/v2") fail("SCHEMA", String(value.schema_version));
	if (!isDeepStrictEqual(value.transition_authority, expectedTransitionAuthority)) fail("AUTHORITY", "transition authority must remain exact and explicitly unevidenced");
	if (!isDeepStrictEqual(value.topology_amendment_authority, expectedTopologyAmendmentAuthority)) fail("AMENDMENT_AUTHORITY", "U7M successor-compatibility repair authority must remain exact and explicitly unevidenced");
	if (!isDeepStrictEqual(value.runtime_authority, expectedRuntimeAuthority)) fail("RUNTIME_AUTHORITY", "Darwin arm64 and Node major 25 tool epoch must remain exact");
	if (!isDeepStrictEqual(value.inherited_receipts, expectedInheritedReceipts)) fail("INHERITED_RECEIPTS", "C3P/C3/C6A must remain PRESENT");
	if (!isDeepStrictEqual(value.receipt_contract, expectedReceiptContract)) fail("RECEIPT_CONTRACT", "U7D study evidence and U7R projection authority must remain exact");
	if (!isDeepStrictEqual(value.claim_binding_policy, expectedClaimBindingPolicy)) fail("CLAIM_BINDING_POLICY", "event, pathspec, recorded child-process interval, and operational writer authority must remain explicit");
	if (!isDeepStrictEqual(value.sealed_parent, expectedSealedParent)) fail("SEALED_PARENT", "C6B identity drift");
	if (!Array.isArray(value.units) || value.units.length !== unitOrder.length ||
		!isDeepStrictEqual(value.units.map((unit) => unit?.id), unitOrder)) fail("UNIT_ORDER", "expected U7P,U7M,U7A,U7B,U7C,U7D,U7R");

	for (const unit of value.units) {
		if (!exactKeys(unit, ["id", "parent", "verification_profile", "product_authority", "product_behavior", "u7d_receipt_state", "subject", "final_root", "allowed_paths", "required_paths", "claims"])) {
			fail("UNIT_KEYS", unit?.id ?? "unknown");
		}
		if (unit.parent !== expectedParents[unit.id] || unit.verification_profile !== expectedProfiles[unit.id] ||
			unit.product_authority !== expectedProductAuthority[unit.id] || unit.product_behavior !== expectedProductBehavior[unit.id] || unit.subject !== expectedSubjects[unit.id] ||
			unit.u7d_receipt_state !== (unit.id === "U7R" ? "PRESENT" : "ABSENT") ||
			unit.final_root !== expectedFinalRoots[unit.id]) fail("UNIT_IDENTITY", unit.id);
		if (!validPath(unit.final_root) || !uniqueStrings(unit.allowed_paths) || !uniqueStrings(unit.required_paths) ||
			unit.allowed_paths.some((path) => !validPath(path)) || unit.required_paths.some((path) => !validPath(path))) fail("UNIT_PATHS", unit.id);
		if (unit.required_paths.some((path) => !unit.allowed_paths.includes(path))) fail("REQUIRED_OUTSIDE_ALLOWED", unit.id);
		if (!isDeepStrictEqual(unit.required_paths, unit.allowed_paths)) fail("NONEXACT_UNIT_ROSTER", unit.id);
		if (!Array.isArray(unit.claims) || unit.claims.length !== expectedClaimCounts[unit.id]) fail("CLAIM_COUNT", unit.id);
		const labels = new Set();
		for (const claim of unit.claims) {
			if (!exactKeys(claim, ["type", "label", "command"]) || !["tests-pass", "command-succeeded"].includes(claim.type) ||
				typeof claim.label !== "string" || claim.label.length < 8 || claim.label.length > 180 || labels.has(claim.label) ||
				!validCommand(claim.command)) fail("CLAIM", `${unit.id}:${claim?.label ?? "unknown"}`);
			labels.add(claim.label);
		}
		const finalCommand = unit.claims.at(-1).command;
		if (!isDeepStrictEqual(finalCommand, ["/opt/homebrew/bin/node", "tools/check-u7-plan.mjs", "--verify-preseal", unit.id])) {
			fail("PRESEAL_NOT_LAST", unit.id);
		}
	}
	const u7p = value.units[0];
	if (!isDeepStrictEqual(u7p.allowed_paths, u7pExactPaths) || !isDeepStrictEqual(u7p.required_paths, u7pExactPaths) ||
		!isDeepStrictEqual(normalizeClaims(u7p.claims), normalizeClaims(expectedU7PClaims))) fail("U7P_CONTRACT", "exact U7P scope or claims drift");
	const u7m = value.units[1];
	if (!isDeepStrictEqual(u7m.allowed_paths, u7mExactPaths) || !isDeepStrictEqual(u7m.required_paths, u7mExactPaths) ||
		!isDeepStrictEqual(normalizeClaims(u7m.claims), normalizeClaims(expectedU7MClaims))) fail("U7M_CONTRACT", "exact U7M scope or claims drift");
	for (const unit of value.units.slice(2)) {
		for (const protectedPath of [
			"spec/verification/u7-unit-paths.json", "tools/check-u7-plan.mjs", "tools/check-u7-scope.mjs",
			"tools/check-u7-architecture.mjs", "tools/check-u7-architecture-selftest.mjs",
			"tools/check-u7-study-harness.mjs", "tools/check-u7-study-harness-selftest.mjs",
			"tools/print-u7-final-runbook.mjs", "tools/verify-current.mjs", "tools/verify-current-selftest.mjs",
		]) {
			if (unit.allowed_paths.includes(protectedPath)) fail("FUTURE_CONTROL_PLANE_OWNERSHIP", `${unit.id}:${protectedPath}`);
		}
	}
	return value;
}

async function readRegular(path, maximum = maximumTextBytes, allowEmpty = false) {
	let handle;
	try { handle = await open(path, constants.O_RDONLY | constants.O_NOFOLLOW); }
	catch (error) { fail("FILE_OPEN", `${path}:${error.code ?? error.message}`); }
	try {
		const before = await handle.stat();
		if (!before.isFile() || (!allowEmpty && before.size <= 0) || before.size > maximum) fail("FILE_AUTHORITY", `${path}:${before.size}`);
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
			const bytes = await readRegular(absolute, 1024 * 1024, true);
			manifest.push({ path, bytes: bytes.length, sha256: sha256Hex(bytes) });
		}
	};
	await walk(root, "", 0);
	return Object.freeze({
		fileCount: manifest.length,
		bytes: manifest.reduce((sum, entry) => sum + entry.bytes, 0),
		digest: sha256Hex(Buffer.from(`${JSON.stringify(manifest)}\n`, "utf8")),
	});
}

async function didrunStartupManifest(authority) {
	const root = authority.didrun_site_packages_root;
	if (await realpath(root) !== root) fail("DIDRUN_STARTUP_ROOT", root);
	const manifest = [];
	const entries = (await readdir(root, { withFileTypes: true })).sort((left, right) => left.name < right.name ? -1 : left.name > right.name ? 1 : 0);
	for (const entry of entries) {
		if (!entry.name.endsWith(".pth") && !["sitecustomize.py", "usercustomize.py"].includes(entry.name)) continue;
		if (!entry.isFile() || entry.isSymbolicLink()) fail("DIDRUN_STARTUP_ENTRY", entry.name);
		const bytes = await readRegular(resolve(root, entry.name), 64 * 1024, true);
		manifest.push({ path: entry.name, bytes: bytes.length, sha256: sha256Hex(bytes) });
	}
	return Object.freeze({
		fileCount: manifest.length,
		bytes: manifest.reduce((sum, entry) => sum + entry.bytes, 0),
		digest: sha256Hex(Buffer.from(`${JSON.stringify(manifest)}\n`, "utf8")),
	});
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
				if (String(error?.message).startsWith("U7_PLAN_")) throw error;
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
			sha256Hex(await readRegular(tool.realpath, 32 * 1024 * 1024)) !== tool.sha256) fail("TOOL_RUNTIME_EPOCH", String(tool?.name));
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
		const output = checkedSpawn(path, args, { encoding: "utf8" });
		outputByName.set(name, output.split("\n", 1)[0]);
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
	if (sha256Hex(didrunBytes) !== authority.didrun_entrypoint_shim_sha256 ||
		sha256Hex(pythonBytes) !== authority.didrun_python_launcher_sha256 ||
		pyvenvBytes.length !== authority.didrun_pyvenv_config_bytes || sha256Hex(pyvenvBytes) !== authority.didrun_pyvenv_config_sha256 ||
		authority.didrun_include_system_site_packages !== false ||
		decodeUTF8(pyvenvBytes, "didrun pyvenv.cfg").split("\n").filter((line) => line === "include-system-site-packages = false").length !== 1 ||
		routeBytes.length !== authority.didrun_import_route_bytes || sha256Hex(routeBytes) !== authority.didrun_import_route_sha256 ||
		!routeBytes.equals(Buffer.from(`${editableRoot}\n`, "utf8")) ||
		startupManifest.fileCount !== authority.didrun_startup_file_count || startupManifest.bytes !== authority.didrun_startup_bytes ||
		startupManifest.digest !== authority.didrun_startup_manifest_sha256 || packageManifest.fileCount !== authority.didrun_package_file_count ||
		packageManifest.bytes !== authority.didrun_package_bytes || packageManifest.digest !== authority.didrun_package_manifest_sha256 ||
		repositoryConfig.length !== authority.didrun_repository_git_config_bytes || sha256Hex(repositoryConfig) !== authority.didrun_repository_git_config_sha256 ||
		authority.didrun_repository_replace_refs !== "ABSENT") fail("DIDRUN_STATIC_RUNTIME_EPOCH", didrunExecutable);
	return didrunExecutable;
}

function decodeUTF8(bytes, label) {
	try {
		return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
	} catch (error) {
		fail("UTF8", `${label}:${error.message}`);
	}
}

export async function loadSpecification() {
	const text = decodeUTF8(await readRegular(specificationPath, 1024 * 1024), "specification");
	let parsed;
	try { parsed = JSON.parse(text); } catch (error) { fail("JSON", error.message); }
	const specification = validateSpecification(parsed);
	const nodeMajor = Number.parseInt(process.versions.node.split(".")[0], 10);
	if (process.platform !== specification.runtime_authority.platform || process.arch !== specification.runtime_authority.architecture ||
		nodeMajor !== specification.runtime_authority.node_major ||
		await realpath(process.execPath) !== await realpath(specification.runtime_authority.node_path)) {
		fail("RUNTIME_EPOCH", `${process.platform}/${process.arch} node=${process.versions.node} exec=${process.execPath}`);
	}
	await validateAdmittedTools(specification.runtime_authority);
	const didrunExecutable = await validateDidrunStaticAuthority(specification.runtime_authority);
	const outerEnvironment = recorderEnvironment(specification.runtime_authority);
	const replaceRefs = checkedSpawn(specification.runtime_authority.didrun_outer_git_path, ["--no-replace-objects", "for-each-ref", "--format=%(refname)", "refs/replace"], { encoding: "utf8", env: outerEnvironment });
	if (replaceRefs !== "") fail("DIDRUN_REPLACE_REFS", replaceRefs.trim());
	const didrunVersion = checkedSpawn(specification.runtime_authority.didrun_path, ["--version"], { encoding: "utf8", env: outerEnvironment });
	const importedRoot = checkedSpawn(specification.runtime_authority.didrun_python_entrypoint, ["-c", "import pathlib, site, didrun; print(pathlib.Path(didrun.__file__).resolve().parent); print(int(bool(site.ENABLE_USER_SITE)))"], { encoding: "utf8", env: outerEnvironment });
	if (importedRoot !== `${specification.runtime_authority.didrun_package_root}\n0\n` ||
		didrunVersion !== `${specification.runtime_authority.didrun_version}\n`) fail("DIDRUN_RUNTIME_EPOCH", didrunExecutable);
	await validateDidrunStaticAuthority(specification.runtime_authority);
	await validateAdmittedToolBytes(specification.runtime_authority);
	return specification;
}

export function unitByID(specification, unitID) {
	const unit = specification.units.find((candidate) => candidate.id === unitID);
	if (unit === undefined) fail("UNIT_UNKNOWN", String(unitID));
	return unit;
}

export function parsePhaseCapsule(text) {
	if (typeof text !== "string" || text.split(capsuleStart).length !== 2 || text.split(capsuleEnd).length !== 2) fail("CAPSULE_CARDINALITY", "one start and one end marker required");
	const start = text.indexOf(capsuleStart);
	const end = text.indexOf(capsuleEnd);
	if (start < 0 || end <= start) fail("CAPSULE_ORDER", "markers");
	if ((start !== 0 && text[start - 1] !== "\n") || text[start + capsuleStart.length] !== "\n" ||
		text[end - 1] !== "\n" || !["", "\n"].includes(text.slice(end + capsuleEnd.length, end + capsuleEnd.length + 1))) fail("CAPSULE_LINES", "markers must be standalone LF lines");
	const currentState = sectionBounds(text, "## Current state\n");
	if (start < currentState.start || end >= currentState.end || !/^\s*$/u.test(text.slice(currentState.start, start))) fail("CAPSULE_JURISDICTION", "first authority in Current state");
	requireVisibleMarkers(text, capsuleStart, capsuleEnd, "U7_PHASE");
	const body = text.slice(start, end + capsuleEnd.length);
	const match = /^<!-- U7-PHASE:START -->\n### Active U7 phase contract\n\n- \*\*Namespace:\*\* `countershape\/u7-unit-paths\/v2`\n- \*\*Boundary:\*\* `([A-Z0-9]+)`\n- \*\*Parent:\*\* `([A-Z0-9.]+)`\n- \*\*Verification profile:\*\* `(SOURCE_FULL|RECEIPT_RECONCILIATION)`\n- \*\*State:\*\* `(SOURCE_CANDIDATE|RECEIPT_CANDIDATE)`\n- \*\*Topology:\*\* `U7P -> U7M -> U7A -> U7B -> U7C -> U7D -> U7R`\n- \*\*Inherited receipts:\*\* `C3P=PRESENT; C3=PRESENT; C6A=PRESENT`\n- \*\*Receipt U7D:\*\* `(ABSENT|PRESENT)`\n- \*\*Product authority:\*\* `(NONE|U7_REFERENCE_APPLICATION)`\n- \*\*Product behavior:\*\* `(INHERITED_UNREPROVEN|CANDIDATE_UNRECEIPTED|SOURCE_RECEIPT_RECONCILIATION)`\n<!-- U7-PHASE:END -->$/u.exec(body);
	if (!match) fail("CAPSULE_SHAPE", body.slice(0, 512));
	return Object.freeze({ boundary: match[1], parent: match[2], profile: match[3], state: match[4], receipt: match[5], productAuthority: match[6], productBehavior: match[7] });
}

export async function loadPhaseCapsule() {
	return parsePhaseCapsule(decodeUTF8(await readRegular(handoffPath), "HANDOFF"));
}

function gitEnvironment() {
	return {
		HOME: process.env.HOME || "/", TMPDIR: process.env.TMPDIR || "/tmp",
		PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1",
		GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
		GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
	};
}

function checkedSpawn(command, args, options = {}) {
	const result = spawnSync(command, args, {
		cwd: repositoryRoot, encoding: options.encoding ?? "buffer", timeout: options.timeout ?? 30_000,
		maxBuffer: options.maxBuffer ?? 16 * 1024 * 1024, env: options.env ?? gitEnvironment(),
	});
	const stderr = Buffer.isBuffer(result.stderr) ? result.stderr : Buffer.from(result.stderr ?? "", "utf8");
	if (result.error || result.signal || result.status !== 0 || stderr.length !== 0) {
		fail("COMMAND", `${command} ${args.join(" ")} status=${result.status} signal=${result.signal} error=${result.error?.message ?? "none"} stderr_bytes=${stderr.length}`);
	}
	return result.stdout;
}

function gitBytes(args) {
	return checkedSpawn(gitPath, ["--no-replace-objects", ...args]);
}

function gitLine(args, label) {
	const bytes = gitBytes(args);
	const text = decodeUTF8(bytes, label);
	if (!/^[^\r\n]+\n$/u.test(text)) fail("GIT_LINE", `${label}:${JSON.stringify(text.slice(0, 256))}`);
	return text.slice(0, -1);
}

function decodeNULPaths(bytes, label) {
	if (bytes.length > 0 && bytes.at(-1) !== 0) fail("NUL_DELIMITER", label);
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

async function candidateSnapshot(unit) {
	const head = gitLine(["rev-parse", "--verify", "HEAD^{commit}"], `${unit.id} candidate HEAD`);
	const indexTree = gitLine(["write-tree"], `${unit.id} candidate index tree`);
	const paths = stagedPaths();
	if (!isDeepStrictEqual(paths, [...unit.allowed_paths].sort())) fail("CANDIDATE_ROSTER", unit.id);
	gitBytes(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--cached", "--check", "--"]);
	if (gitBytes(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--name-only", "-z", "--"]).length !== 0) fail("CANDIDATE_UNSTAGED", unit.id);
	if (gitBytes(["ls-files", "--others", "--exclude-standard", "-z", "--"]).length !== 0) fail("CANDIDATE_UNTRACKED", unit.id);
	const blobDigests = [];
	for (const path of paths) {
		const indexBytes = gitBytes(["show", `:${path}`]);
		const workingBytes = await readRegular(resolve(repositoryRoot, path));
		if (!indexBytes.equals(workingBytes)) fail("CANDIDATE_INDEX_WORKTREE", path);
		blobDigests.push(createHash("sha256").update(indexBytes).digest("hex"));
	}
	if (head !== gitLine(["rev-parse", "--verify", "HEAD^{commit}"], `${unit.id} candidate final HEAD`) ||
		indexTree !== gitLine(["write-tree"], `${unit.id} candidate final index tree`) ||
		!isDeepStrictEqual(paths, stagedPaths())) fail("CANDIDATE_SNAPSHOT_CHANGED", unit.id);
	return Object.freeze({ head, indexTree, paths: Object.freeze(paths), blobDigests: Object.freeze(blobDigests) });
}

function sameCandidateSnapshot(left, right) {
	return left.head === right.head && left.indexTree === right.indexTree &&
		isDeepStrictEqual(left.paths, right.paths) && isDeepStrictEqual(left.blobDigests, right.blobDigests);
}

const noteRedaction = "«redacted:high-entropy»";
const noteRootRedactionPositions = new Set([2, 4, 5, 6, 7, 8]);
const noteBareRedactionPositions = new Set([31, 32, 33, 35, 36, 37]);

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

export function claimPreviewMatches(preview, expectedArgv, prefix) {
	if (!Array.isArray(preview) || !Array.isArray(expectedArgv) || !Array.isArray(prefix) ||
		preview.length !== expectedArgv.length || prefix.length === 0 ||
		!isDeepStrictEqual(expectedArgv.slice(0, prefix.length), prefix) ||
		preview.some((part) => typeof part !== "string" || part.length === 0 || part.length > 4096 || /[\u0000-\u001f\u007f]/u.test(part))) return false;
	for (let index = 0; index < prefix.length; index += 1) {
		if (preview[index] !== expectedArgv[index] && !expectedRedactions(index, expectedArgv[index]).includes(preview[index])) return false;
	}
	return isDeepStrictEqual(preview.slice(prefix.length), expectedArgv.slice(prefix.length));
}

function parseNote(bytes, expectedCommit, expectedTree, expectedClaims = undefined, expectedPrefix = undefined, expectedPathspecs = []) {
	let note;
	try { note = JSON.parse(decodeUTF8(bytes, "didrun note")); } catch (error) { fail("NOTE_JSON", error.message); }
	if (!exactKeys(note, ["claims", "commit", "coverage", "secrets_override", "tree", "version"]) || note.version !== 1 ||
		note.commit !== expectedCommit || note.tree !== expectedTree || !Array.isArray(note.claims) || typeof note.secrets_override !== "boolean" ||
		!exactKeys(note.coverage, ["by_coverage", "total_events"]) || !exactKeys(note.coverage.by_coverage, ["complete"]) ||
		note.coverage.by_coverage.complete !== note.claims.length || note.coverage.total_events !== note.claims.length) fail("NOTE_SHAPE", expectedCommit);
	if (expectedClaims !== undefined && note.claims.length !== expectedClaims.length) fail("NOTE_COUNT", expectedCommit);
	for (const [index, record] of note.claims.entries()) {
		const claim = record?.claim;
		if (!exactKeys(record, ["claim", "delta", "exit_code", "grade", "reason", "supporting_event_index"]) || record.grade !== "tree-exact" ||
			record.exit_code !== 0 || !isDeepStrictEqual(record.delta, []) || record.supporting_event_index !== index ||
			!exactKeys(claim, ["argv_preview", "ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) ||
			claim.declared_at_index !== index || !isDeepStrictEqual(claim.event_indices, [index]) || !isDeepStrictEqual(claim.pathspecs, expectedPathspecs) ||
			record.reason !== "self-stable command ran against the sealed tree") fail("NOTE_CLAIM", `${expectedCommit}:${index}`);
		if (expectedClaims !== undefined) {
			const expected = expectedClaims[index];
			const expectedArgv = [...expectedPrefix, ...expected.command];
			if (claim.label !== expected.label || claim.ctype !== expected.type ||
				!claimPreviewMatches(claim.argv_preview, expectedArgv, expectedPrefix)) fail("NOTE_MANIFEST", `${expectedCommit}:${index}`);
		}
	}
	return note;
}

async function validateC6BCommit(specification, candidateCommit) {
	const sealed = specification.sealed_parent;
	const commit = gitLine(["rev-parse", "--verify", `${candidateCommit}^{commit}`], "C6B commit");
	const tree = gitLine(["rev-parse", "--verify", `${commit}^{tree}`], "C6B tree");
	const parent = gitLine(["show", "-s", "--format=%P", commit], "C6B parent");
	const subject = gitLine(["show", "-s", "--format=%s", commit], "C6B subject");
	if (commit !== sealed.commit || tree !== sealed.tree || parent !== sealed.parent || subject !== sealed.subject) fail("C6B_GIT", `${commit}:${tree}`);
	const declaration = specification.transition_authority.predecessor_declaration;
	const artifactRow = decodeUTF8(gitBytes(["ls-tree", commit, "--", declaration.artifact_path]), "P08 predecessor artifact");
	const expectedArtifactRow = `${declaration.artifact_mode} blob ${declaration.artifact_blob}\t${declaration.artifact_path}\n`;
	if (artifactRow !== expectedArtifactRow) fail("PREDECESSOR_ARTIFACT_ROW", declaration.artifact_path);
	const artifactBytes = gitBytes(["show", `${commit}:${declaration.artifact_path}`]);
	if (artifactBytes.length !== declaration.artifact_bytes ||
		`sha256:${createHash("sha256").update(artifactBytes).digest("hex")}` !== declaration.artifact_sha256) {
		fail("PREDECESSOR_ARTIFACT_BYTES", declaration.artifact_path);
	}
	const noteBlob = gitLine(["notes", "--ref=didrun", "list", commit], "C6B note blob");
	if (noteBlob !== sealed.note_blob) fail("C6B_NOTE_BLOB", noteBlob);
	const noteBytes = gitBytes(["cat-file", "blob", noteBlob]);
	if (createHash("sha256").update(noteBytes).digest("hex") !== sealed.note_body_sha256) fail("C6B_NOTE_DIGEST", noteBlob);
	const note = parseNote(noteBytes, commit, tree);
	if (note.claims.length !== sealed.claim_count || note.secrets_override !== sealed.secrets_override ||
		note.claims.some((record) => record.grade !== sealed.grade)) fail("C6B_NOTE_AUTHORITY", commit);
	const htmlBytes = await readRegular(resolve(repositoryRoot, sealed.html_path), 16 * 1024 * 1024);
	if (createHash("sha256").update(htmlBytes).digest("hex") !== sealed.html_sha256) fail("C6B_HTML", sealed.html_path);
	await validateArchivedLedger(undefined, commit, tree, note, sealed.ledger_archive, {
		file_count: sealed.ledger_file_count, bytes: sealed.ledger_bytes, manifest_sha256: sealed.ledger_manifest_sha256,
		event_count: sealed.ledger_event_count, claim_count: sealed.ledger_claim_count, seal_count: sealed.ledger_seal_count,
	});
	return Object.freeze({ commit, tree, noteBlob });
}

async function validateSealedParent(specification) {
	const head = gitLine(["rev-parse", "--verify", "HEAD^{commit}"], "HEAD commit");
	if (head !== specification.sealed_parent.commit) fail("C6B_HEAD", head);
	return validateC6BCommit(specification, head);
}

function changedPathsForCommit(commit) {
	const bytes = gitBytes(["diff-tree", "--root", "--no-commit-id", "--name-only", "-r", "-z", "--no-renames", commit, "--"]);
	if (bytes.length > 0 && bytes.at(-1) !== 0) fail("PARENT_DIFF", "missing NUL delimiter");
	const text = decodeUTF8(bytes.length === 0 ? bytes : bytes.subarray(0, -1), "parent diff");
	return text === "" ? [] : text.split("\0").sort();
}

async function validateSealedUnitNote(unit, commit, tree) {
	const noteBlob = gitLine(["notes", "--ref=didrun", "list", commit], `${unit.id} note blob`);
	const note = parseNote(gitBytes(["cat-file", "blob", noteBlob]), commit, tree, unit.claims, hermeticPrefix(unit), unit.allowed_paths);
	const changed = changedPathsForCommit(commit);
	if (!isDeepStrictEqual(changed, [...unit.allowed_paths].sort())) fail("SEALED_UNIT_SCOPE", unit.id);
	await validateArchivedLedger(unit, commit, tree, note, `.didrun-history/${unit.id.toLowerCase()}-final-${commit.slice(0, 12)}/.didrun`);
	return noteBlob;
}

async function validateDeclaredAncestry(specification, parentUnit, startingCommit) {
	let commit = startingCommit;
	let immediateNoteBlob;
	for (let index = unitOrder.indexOf(parentUnit.id); index >= 0; index -= 1) {
		const row = specification.units[index];
		const observed = gitLine(["rev-parse", "--verify", `${commit}^{commit}`], `${row.id} ancestry commit`);
		const tree = gitLine(["rev-parse", "--verify", `${observed}^{tree}`], `${row.id} ancestry tree`);
		const subject = gitLine(["show", "-s", "--format=%s", observed], `${row.id} ancestry subject`);
		const parent = gitLine(["show", "-s", "--format=%P", observed], `${row.id} ancestry parent`);
		if (subject !== row.subject || !/^[0-9a-f]{40}$/u.test(parent)) fail("DECLARED_ANCESTRY", row.id);
		const noteBlob = await validateSealedUnitNote(row, observed, tree);
		if (index === unitOrder.indexOf(parentUnit.id)) {
			immediateNoteBlob = noteBlob;
		}
		commit = parent;
	}
	if (commit !== specification.sealed_parent.commit) fail("DECLARED_C6B_ANCESTRY", commit);
	await validateC6BCommit(specification, commit);
	return immediateNoteBlob;
}

export async function checkCandidate(unitID, writeSuccess = (line) => console.log(line)) {
	const specification = await loadSpecification();
	const unit = unitByID(specification, unitID);
	const candidateBefore = await candidateSnapshot(unit);
	const handoffText = decodeUTF8(await readRegular(handoffPath), "HANDOFF");
	const capsule = parsePhaseCapsule(handoffText);
	const expectedState = unit.verification_profile === "SOURCE_FULL" ? "SOURCE_CANDIDATE" : "RECEIPT_CANDIDATE";
	const expectedBehavior = expectedProductBehavior[unit.id];
	if (!isDeepStrictEqual(capsule, {
		boundary: unit.id, parent: unit.parent, profile: unit.verification_profile, state: expectedState,
		receipt: unit.u7d_receipt_state, productAuthority: unit.product_authority, productBehavior: expectedBehavior,
	})) fail("CAPSULE_UNIT", unit.id);
	validateReceiptPhase(unit.id, handoffText);
	if (unit.id === "U7D") {
		const documents = new Map();
		for (const path of specification.receipt_contract.projection_paths) {
			documents.set(path, path === "docs/HANDOFF_MODE_C.md" ? handoffText : decodeUTF8(gitBytes(["show", `:${path}`]), `${path} staged pending receipt`));
		}
		validatePendingProjectionDocuments(specification, documents);
	}
	if (unitID === "U7P") {
		const sealed = await validateSealedParent(specification);
		const candidateAfter = await candidateSnapshot(unit);
		if (!sameCandidateSnapshot(candidateBefore, candidateAfter) || !isDeepStrictEqual(specification, await loadSpecification())) fail("CANDIDATE_AUTHORITY_CHANGED", unit.id);
		writeSuccess(`U7P candidate authority exact: parent=C6B commit=${sealed.commit} tree=${sealed.tree} note=${sealed.noteBlob} profile=SOURCE_FULL product_authority=NONE provenance=UNEVIDENCED`);
		return;
	}
	const parentUnit = unitByID(specification, unit.parent);
	const commit = candidateBefore.head;
	const tree = gitLine(["rev-parse", "--verify", "HEAD^{tree}"], `${unit.id} parent tree`);
	const subject = gitLine(["show", "-s", "--format=%s", "HEAD"], `${unit.id} parent subject`);
	if (subject !== parentUnit.subject) fail("PARENT_SUBJECT", `${unit.id}:${subject}`);
	const noteBlob = await validateDeclaredAncestry(specification, parentUnit, commit);
	const candidateAfter = await candidateSnapshot(unit);
	if (!sameCandidateSnapshot(candidateBefore, candidateAfter) || !isDeepStrictEqual(specification, await loadSpecification())) fail("CANDIDATE_AUTHORITY_CHANGED", unit.id);
	writeSuccess(`${unit.id} candidate authority exact: parent=${parentUnit.id} commit=${commit} tree=${tree} note=${noteBlob} profile=${unit.verification_profile} product_authority=${unit.product_authority}`);
}

function inheritedP07CompatibilityInputs(specification) {
	const parent = specification.sealed_parent.commit;
	const parentHandoff = decodeUTF8(gitBytes(["show", `${parent}:docs/HANDOFF_MODE_C.md`]), "sealed C6B HANDOFF");
	const candidateHandoff = decodeUTF8(gitBytes(["show", ":docs/HANDOFF_MODE_C.md"]), "staged HANDOFF");
	const parentProtected = new Map(); const candidateProtected = new Map();
	for (const path of inheritedP07ProtectedPaths) {
		parentProtected.set(path, gitBytes(["show", `${parent}:${path}`]));
		candidateProtected.set(path, gitBytes(["show", `:${path}`]));
	}
	return Object.freeze({ parentHandoff, candidateHandoff, parentProtected, candidateProtected });
}

function parseInheritedP07C6AuthorityManifest(bytes) {
	let manifest;
	try { manifest = JSON.parse(decodeUTF8(bytes, "sealed C6A source-authority manifest")); }
	catch (error) { fail("P07_C6A_AUTHORITY_JSON", error.message); }
	const keys = [
		"schema_version", "source_commit", "source_tree", "source_parent", "source_subject",
		"note_ref", "note_type", "note_blob", "note_body_sha256", "secrets_override", "claims",
	];
	if (!exactKeys(manifest, keys) || manifest.schema_version !== inheritedP07C6AuthoritySchema ||
		manifest.source_subject !== inheritedP07C6SourceSubject || manifest.note_ref !== "refs/notes/didrun" ||
		manifest.note_type !== "blob" || typeof manifest.secrets_override !== "boolean" ||
		![manifest.source_commit, manifest.source_tree, manifest.source_parent, manifest.note_blob]
			.every((value) => typeof value === "string" && /^[0-9a-f]{40}$/u.test(value)) ||
		typeof manifest.note_body_sha256 !== "string" || !/^[0-9a-f]{64}$/u.test(manifest.note_body_sha256) ||
		!Array.isArray(manifest.claims) || manifest.claims.length !== 11 ||
		new Set(manifest.claims.map((claim) => claim?.label)).size !== manifest.claims.length ||
		manifest.claims.some((claim) => !exactKeys(claim, ["label", "type", "grade"]) ||
			typeof claim.label !== "string" || claim.label.length === 0 ||
			!["tests-pass", "command-succeeded"].includes(claim.type) || claim.grade !== "TREE-EXACT") ||
		!bytes.equals(Buffer.from(`${JSON.stringify(manifest, null, 2)}\n`, "utf8"))) {
		fail("P07_C6A_AUTHORITY_MANIFEST", "sealed canonical source authority");
	}
	return Object.freeze(manifest);
}

function inheritedP07C6AuthorityInputs(specification) {
	const sealedCommit = specification.sealed_parent.commit;
	const sealedTracked = new Map();
	const candidateTracked = new Map();
	for (const path of inheritedP07C6AuthorityPaths) {
		sealedTracked.set(path, gitBytes(["show", `${sealedCommit}:${path}`]));
		candidateTracked.set(path, gitBytes(["show", `:${path}`]));
	}
	const manifest = parseInheritedP07C6AuthorityManifest(sealedTracked.get(inheritedP07C6AuthorityPaths[0]));
	const sourceCommit = gitLine(["rev-parse", "--verify", `${manifest.source_commit}^{commit}`], "sealed C6A source commit");
	const source = Object.freeze({
		commit: sourceCommit,
		tree: gitLine(["rev-parse", "--verify", `${sourceCommit}^{tree}`], "sealed C6A source tree"),
		parent: gitLine(["show", "-s", "--format=%P", sourceCommit], "sealed C6A source parent"),
		subject: gitLine(["show", "-s", "--format=%s", sourceCommit], "sealed C6A source subject"),
		noteBlob: gitLine(["notes", "--ref=didrun", "list", sourceCommit], "sealed C6A note blob"),
	});
	const noteBytes = gitBytes(["cat-file", "blob", source.noteBlob]);
	const sealedSourceSummary = gitBytes(["show", `${source.tree}:${inheritedP07C6AuthorityPaths[2]}`]);
	return Object.freeze({ sealedTracked, candidateTracked, source, noteBytes, sealedSourceSummary });
}

export function validateInheritedP07C6Authority(inputs) {
	if (!exactKeys(inputs, ["sealedTracked", "candidateTracked", "source", "noteBytes", "sealedSourceSummary"]) ||
		!(inputs.sealedTracked instanceof Map) || !(inputs.candidateTracked instanceof Map) ||
		!isDeepStrictEqual([...inputs.sealedTracked.keys()], inheritedP07C6AuthorityPaths) ||
		!isDeepStrictEqual([...inputs.candidateTracked.keys()], inheritedP07C6AuthorityPaths) ||
		!Buffer.isBuffer(inputs.noteBytes) || !Buffer.isBuffer(inputs.sealedSourceSummary)) {
		fail("P07_C6A_AUTHORITY_INPUT", "exact sealed/current authority corpus");
	}
	for (const path of inheritedP07C6AuthorityPaths) {
		const sealed = inputs.sealedTracked.get(path); const candidate = inputs.candidateTracked.get(path);
		if (!Buffer.isBuffer(sealed) || !Buffer.isBuffer(candidate) || !candidate.equals(sealed)) {
			fail("P07_C6A_AUTHORITY_PROJECTION", path);
		}
	}
	const manifest = parseInheritedP07C6AuthorityManifest(inputs.sealedTracked.get(inheritedP07C6AuthorityPaths[0]));
	if (!exactKeys(inputs.source, ["commit", "tree", "parent", "subject", "noteBlob"]) ||
		inputs.source.commit !== manifest.source_commit || inputs.source.tree !== manifest.source_tree ||
		inputs.source.parent !== manifest.source_parent || inputs.source.subject !== manifest.source_subject ||
		inputs.source.noteBlob !== manifest.note_blob ||
		createHash("sha256").update(inputs.noteBytes).digest("hex") !== manifest.note_body_sha256) {
		fail("P07_C6A_AUTHORITY_IDENTITY", manifest.source_commit);
	}
	const sealedSummary = inputs.sealedTracked.get(inheritedP07C6AuthorityPaths[2]);
	if (!inputs.sealedSourceSummary.equals(sealedSummary)) {
		fail("P07_C6A_AUTHORITY_SUMMARY", inheritedP07C6AuthorityPaths[2]);
	}
	return Object.freeze({
		commit: manifest.source_commit, tree: manifest.source_tree, parent: manifest.source_parent,
		subject: manifest.source_subject, noteBlob: manifest.note_blob,
		noteBodySHA256: manifest.note_body_sha256, secretsOverride: manifest.secrets_override,
	});
}

async function runInheritedP07SuccessorPlan(authority, plan = checkInheritedP07Plan) {
	const errors = await plan(
		repositoryRoot, new Map(), ...Array(9).fill(undefined), "C6B", "C6B", undefined, authority,
	);
	if (!Array.isArray(errors) || errors.some((error) => typeof error !== "string")) {
		fail("P07_SUCCESSOR_PLAN_RESULT", "legacy checker returned malformed result");
	}
	if (errors.length !== 0) fail("P07_SUCCESSOR_PLAN", errors.join(" | "));
}

async function inheritedP07CompositeSnapshot(unit) {
	return Object.freeze({
		candidate: await candidateSnapshot(unit),
		notesTree: gitLine(["rev-parse", "--verify", "refs/notes/didrun^{tree}"], "inherited P07 notes tree"),
	});
}

function sameInheritedP07CompositeSnapshot(left, right) {
	return left?.notesTree === right?.notesTree && sameCandidateSnapshot(left?.candidate, right?.candidate);
}

export async function verifyInheritedP07Compatibility(dependencies = {}) {
	const specification = dependencies.specification ?? await loadSpecification();
	const capsule = dependencies.capsule ?? await loadPhaseCapsule();
	const unit = unitByID(specification, capsule.boundary);
	const candidate = dependencies.candidate ?? checkCandidate;
	const validate = dependencies.validate ?? validateInheritedP07Compatibility;
	const validateAuthority = dependencies.validateAuthority ?? validateInheritedP07C6Authority;
	const runPlan = dependencies.runPlan ?? runInheritedP07SuccessorPlan;
	const snapshot = dependencies.snapshot ?? inheritedP07CompositeSnapshot;
	const reload = dependencies.reload ?? loadSpecification;
	const write = dependencies.write ?? ((line) => console.log(line));
	const before = await snapshot(unit);
	await candidate(capsule.boundary, () => {});
	const inputs = dependencies.inputs ?? inheritedP07CompatibilityInputs(specification);
	const result = validate(inputs.parentHandoff, inputs.candidateHandoff, inputs.parentProtected, inputs.candidateProtected);
	const authorityInputs = dependencies.authorityInputs ?? inheritedP07C6AuthorityInputs(specification);
	const authority = validateAuthority(authorityInputs);
	await runPlan(authority);
	await candidate(capsule.boundary, () => {});
	const reloaded = await reload();
	const after = await snapshot(unit);
	if (!sameInheritedP07CompositeSnapshot(before, after) || !isDeepStrictEqual(specification, reloaded)) {
		fail("P07_SUCCESSOR_AUTHORITY_CHANGED", capsule.boundary);
	}
	const combined = Object.freeze({ blocks: result.blocks, protectedPaths: result.protectedPaths, c3pPlan: "FULL_EXPLICIT_C6A_AUTHORITY" });
	write(`U7 inherited P07 compatibility exact: parent=C6B blocks=${result.blocks} protected_paths=${result.protectedPaths} c3p_plan=FULL_EXPLICIT_C6A_AUTHORITY legacy_live_entrypoints=RETIRED_SUCCESSOR_INCOMPATIBLE`);
	return combined;
}

async function requireAbsentNoFollow(path, label) {
	try {
		const metadata = await lstat(path);
		const kind = metadata.isSymbolicLink() ? "symbolic link" : metadata.isDirectory() ? "directory" : "filesystem object";
		fail("RUNBOOK_EXPECTED_ABSENT", `${label}:${kind}`);
	} catch (error) {
		if (String(error?.message).startsWith("U7_PLAN_")) throw error;
		if (error?.code !== "ENOENT") throw error;
	}
}

async function requireOwnedDirectoryNoFollow(path, label) {
	const metadata = await lstat(path);
	if (!metadata.isDirectory() || metadata.isSymbolicLink() || metadata.uid !== process.getuid() || (metadata.mode & 0o022) !== 0) {
		fail("RUNBOOK_DIRECTORY", `${label}:mode=${(metadata.mode & 0o777).toString(8)} uid=${metadata.uid}`);
	}
}

async function requireOwnedDirectoryIfPresentNoFollow(path, label) {
	try { await requireOwnedDirectoryNoFollow(path, label); return true; }
	catch (error) {
		if (String(error?.message).startsWith("U7_PLAN_RUNBOOK_DIRECTORY")) throw error;
		if (error?.code === "ENOENT") return false;
		throw error;
	}
}

const failedAttemptSlugPattern = /^[0-9]{8}T[0-9]{6}Z-[a-z0-9](?:[a-z0-9-]{0,46}[a-z0-9])?$/u;

function requireCommitID(value, label) {
	if (typeof value !== "string" || !/^[0-9a-f]{40}$/u.test(value)) fail("RUNBOOK_RECOVERY_COMMIT", `${label}:${String(value)}`);
	return value;
}

export function precommitRecoveryPaths(unit, slug, root = repositoryRoot) {
	if (typeof slug !== "string" || !failedAttemptSlugPattern.test(slug)) {
		fail("RUNBOOK_FAILED_ATTEMPT_SLUG", String(slug));
	}
	const lower = unit.id.toLowerCase();
	const failedLedgerRootRelative = `.didrun-history/${lower}-precommit-failed-${slug}`;
	const failedFinalRootRelative = `${unit.final_root}-precommit-failed-${slug}`;
	return Object.freeze({
		root, liveLedger: resolve(root, ".didrun"), liveFinalRoot: resolve(root, unit.final_root),
		normalLedgerRoot: undefined,
		failedLedgerRootRelative, failedFinalRootRelative,
		failedLedgerRoot: resolve(root, failedLedgerRootRelative), failedFinalRoot: resolve(root, failedFinalRootRelative),
	});
}

export function postcommitRecoveryPaths(unit, commit, root = repositoryRoot) {
	requireCommitID(commit, unit.id);
	const lower = unit.id.toLowerCase();
	const normalLedgerRootRelative = `.didrun-history/${lower}-final-${commit.slice(0, 12)}`;
	const failedLedgerRootRelative = `.didrun-history/${lower}-final-failed-${commit}`;
	const failedFinalRootRelative = `${unit.final_root}-failed-${commit}`;
	return Object.freeze({
		root, liveLedger: resolve(root, ".didrun"), liveFinalRoot: resolve(root, unit.final_root),
		normalLedgerRootRelative, failedLedgerRootRelative, failedFinalRootRelative,
		normalLedgerRoot: resolve(root, normalLedgerRootRelative),
		failedLedgerRoot: resolve(root, failedLedgerRootRelative), failedFinalRoot: resolve(root, failedFinalRootRelative),
	});
}

async function optionalPrivateDirectoryNoFollow(path, label) {
	let metadata;
	try { metadata = await lstat(path); }
	catch (error) { if (error?.code === "ENOENT") return undefined; throw error; }
	if (!metadata.isDirectory() || metadata.isSymbolicLink() || metadata.uid !== process.getuid() || (metadata.mode & 0o7777) !== 0o700) {
		fail("RUNBOOK_RECOVERY_DIRECTORY", `${label}:mode=${(metadata.mode & 0o7777).toString(8)} uid=${metadata.uid}`);
	}
	return metadata;
}

async function requirePrivateDirectoryNoFollow(path, label) {
	const metadata = await optionalPrivateDirectoryNoFollow(path, label);
	if (metadata === undefined) fail("RUNBOOK_RECOVERY_DIRECTORY", `${label}:absent`);
	return metadata;
}

async function stableRecoveryEntries(path, label) {
	const before = await optionalPrivateDirectoryNoFollow(path, label);
	if (before === undefined) return undefined;
	const entries = (await readdir(path)).sort();
	const after = await optionalPrivateDirectoryNoFollow(path, label);
	if (after === undefined || before.dev !== after.dev || before.ino !== after.ino || before.mode !== after.mode || before.mtimeMs !== after.mtimeMs) {
		fail("RUNBOOK_RECOVERY_RACE", label);
	}
	return entries;
}

async function recoveryContainerState(path, label) {
	const entries = await stableRecoveryEntries(path, label);
	if (entries === undefined) return Object.freeze({ kind: "ABSENT" });
	if (entries.length === 0) return Object.freeze({ kind: "EMPTY" });
	if (!isDeepStrictEqual(entries, [".didrun"])) fail("RUNBOOK_RECOVERY_ENTRIES", `${label}:${entries.join(",")}`);
	await optionalPrivateDirectoryNoFollow(resolve(path, ".didrun"), `${label}/.didrun`).then((metadata) => {
		if (metadata === undefined) fail("RUNBOOK_RECOVERY_LEDGER", label);
	});
	return Object.freeze({ kind: "COMPLETE" });
}

async function recoveryRename(source, destination, destinationParent, label) {
	const sourceMetadata = await optionalPrivateDirectoryNoFollow(source, `${label}:source`);
	const parentMetadata = await optionalPrivateDirectoryNoFollow(destinationParent, `${label}:destination-parent`);
	if (sourceMetadata === undefined || parentMetadata === undefined) fail("RUNBOOK_RECOVERY_RENAME_SOURCE", label);
	await requireAbsentNoFollow(destination, `${label}:destination`);
	if (sourceMetadata.dev !== parentMetadata.dev) fail("RUNBOOK_RECOVERY_CROSS_DEVICE", label);
	await rename(source, destination);
	await requireAbsentNoFollow(source, `${label}:source-after`);
	if (await optionalPrivateDirectoryNoFollow(destination, `${label}:destination-after`) === undefined) fail("RUNBOOK_RECOVERY_RENAME_DESTINATION", label);
}

async function classifyRecoveryLedger(paths, allowNoLedger) {
	const live = await optionalPrivateDirectoryNoFollow(paths.liveLedger, "live didrun") !== undefined;
	const normal = paths.normalLedgerRoot === undefined
		? Object.freeze({ kind: "ABSENT" })
		: await recoveryContainerState(paths.normalLedgerRoot, "normal ledger archive");
	const failed = await recoveryContainerState(paths.failedLedgerRoot, "failed ledger archive");
	if (live && normal.kind === "ABSENT" && failed.kind === "ABSENT") return "LIVE";
	if (live && normal.kind === "EMPTY" && failed.kind === "ABSENT") return "NORMAL_EMPTY_WITH_LIVE";
	if (!live && normal.kind === "COMPLETE" && failed.kind === "ABSENT") return "NORMAL_COMPLETE";
	if (live && normal.kind === "ABSENT" && failed.kind === "EMPTY") return "FAILED_EMPTY_WITH_LIVE";
	if (!live && normal.kind === "ABSENT" && failed.kind === "COMPLETE") return "COMPLETE";
	if (!live && normal.kind === "ABSENT" && failed.kind === "ABSENT" && allowNoLedger) return "NONE";
	fail("RUNBOOK_RECOVERY_LEDGER_STATE", `live=${live} normal=${normal.kind} failed=${failed.kind}`);
}

async function classifyRecoveryFinalRoot(paths) {
	const live = await optionalPrivateDirectoryNoFollow(paths.liveFinalRoot, "live final root") !== undefined;
	const failed = await optionalPrivateDirectoryNoFollow(paths.failedFinalRoot, "failed final root") !== undefined;
	if (live === failed) fail("RUNBOOK_RECOVERY_FINAL_ROOT_STATE", `live=${live} failed=${failed}`);
	return live ? "LIVE" : "FAILED";
}

async function requireSameDevice(source, destinationParent, label) {
	const sourceMetadata = await optionalPrivateDirectoryNoFollow(source, `${label}:source`);
	const parentMetadata = await optionalPrivateDirectoryNoFollow(destinationParent, `${label}:destination-parent`);
	if (sourceMetadata === undefined || parentMetadata === undefined) fail("RUNBOOK_RECOVERY_RENAME_SOURCE", label);
	if (sourceMetadata.dev !== parentMetadata.dev) fail("RUNBOOK_RECOVERY_CROSS_DEVICE", label);
}

async function preflightRecoveryTransitions(paths, allowNoLedger, sameDevice = requireSameDevice) {
	const ledger = await classifyRecoveryLedger(paths, allowNoLedger);
	const finalRoot = await classifyRecoveryFinalRoot(paths);
	if (ledger === "LIVE") {
		await sameDevice(paths.liveLedger, resolve(paths.root, ".didrun-history"), "live ledger into failed archive");
	} else if (ledger === "NORMAL_EMPTY_WITH_LIVE") {
		await sameDevice(paths.liveLedger, paths.normalLedgerRoot, "live ledger into normal archive");
		await sameDevice(paths.normalLedgerRoot, resolve(paths.root, ".didrun-history"), "normal archive into failed archive");
	} else if (ledger === "NORMAL_COMPLETE") {
		await sameDevice(paths.normalLedgerRoot, resolve(paths.root, ".didrun-history"), "normal archive into failed archive");
	} else if (ledger === "FAILED_EMPTY_WITH_LIVE") {
		await sameDevice(paths.liveLedger, paths.failedLedgerRoot, "live ledger into failed archive");
	}
	if (finalRoot === "LIVE") {
		await sameDevice(paths.liveFinalRoot, resolve(paths.root, ".countershape"), "final root into failed archive");
	}
	return Object.freeze({ ledger, finalRoot });
}

async function normalizeRecoveryLedger(paths, allowNoLedger) {
	for (let transition = 0; transition < 8; transition += 1) {
		const state = await classifyRecoveryLedger(paths, allowNoLedger);
		if (["COMPLETE", "NONE"].includes(state)) return state;
		if (state === "LIVE") {
			await requireAbsentNoFollow(paths.failedLedgerRoot, "failed ledger archive");
			await mkdir(paths.failedLedgerRoot, { mode: 0o700 });
			await chmod(paths.failedLedgerRoot, 0o700);
			continue;
		}
		if (state === "NORMAL_EMPTY_WITH_LIVE") {
			await recoveryRename(paths.liveLedger, resolve(paths.normalLedgerRoot, ".didrun"), paths.normalLedgerRoot, "live ledger into normal archive");
			continue;
		}
		if (state === "NORMAL_COMPLETE") {
			await recoveryRename(paths.normalLedgerRoot, paths.failedLedgerRoot, resolve(paths.root, ".didrun-history"), "normal archive into failed archive");
			continue;
		}
		if (state === "FAILED_EMPTY_WITH_LIVE") {
			await recoveryRename(paths.liveLedger, resolve(paths.failedLedgerRoot, ".didrun"), paths.failedLedgerRoot, "live ledger into failed archive");
			continue;
		}
	}
	fail("RUNBOOK_RECOVERY_TRANSITIONS", "ledger did not converge");
}

async function normalizeRecoveryFinalRoot(paths) {
	const state = await classifyRecoveryFinalRoot(paths);
	if (state === "LIVE") await recoveryRename(paths.liveFinalRoot, paths.failedFinalRoot, resolve(paths.root, ".countershape"), "final root into failed archive");
	if (await optionalPrivateDirectoryNoFollow(paths.liveFinalRoot, "live final root terminal") !== undefined ||
		await optionalPrivateDirectoryNoFollow(paths.failedFinalRoot, "failed final root terminal") === undefined) {
		fail("RUNBOOK_RECOVERY_FINAL_ROOT_STATE", "terminal");
	}
}

export async function normalizeFailedAttemptArchives(paths, allowNoLedger = false, dependencies = {}) {
	await requireAbsentNoFollow(resolve(paths.root, ".countershape/verify-current/active.lock"), "verifier lock");
	await optionalPrivateDirectoryNoFollow(resolve(paths.root, ".countershape"), ".countershape").then((metadata) => {
		if (metadata === undefined) fail("RUNBOOK_RECOVERY_DIRECTORY", ".countershape:absent");
	});
	await optionalPrivateDirectoryNoFollow(resolve(paths.root, ".didrun-history"), ".didrun-history").then((metadata) => {
		if (metadata === undefined) fail("RUNBOOK_RECOVERY_DIRECTORY", ".didrun-history:absent");
	});
	await preflightRecoveryTransitions(paths, allowNoLedger, dependencies.sameDevice ?? requireSameDevice);
	const ledger = await normalizeRecoveryLedger(paths, allowNoLedger);
	await normalizeRecoveryFinalRoot(paths);
	const terminalLedger = await classifyRecoveryLedger(paths, allowNoLedger);
	if (terminalLedger !== ledger || !["COMPLETE", "NONE"].includes(terminalLedger)) fail("RUNBOOK_RECOVERY_LEDGER_STATE", `terminal:${terminalLedger}`);
	return Object.freeze({ ledger: terminalLedger, paths });
}

async function classifyCompletedLedgerArchive(paths) {
	const live = await optionalPrivateDirectoryNoFollow(paths.liveLedger, "live didrun") !== undefined;
	const normal = await recoveryContainerState(paths.normalLedgerRoot, "commit-addressed ledger archive");
	const failedLedger = await recoveryContainerState(paths.failedLedgerRoot, "failed ledger archive");
	const failedFinal = await optionalPrivateDirectoryNoFollow(paths.failedFinalRoot, "failed final root") !== undefined;
	if (failedLedger.kind !== "ABSENT" || failedFinal) {
		fail("RUNBOOK_ARCHIVE_FAILED_STATE", `ledger=${failedLedger.kind} final=${failedFinal}`);
	}
	if (live && normal.kind === "ABSENT") return "LIVE";
	if (live && normal.kind === "EMPTY") return "EMPTY_WITH_LIVE";
	if (!live && normal.kind === "COMPLETE") return "COMPLETE";
	fail("RUNBOOK_ARCHIVE_STATE", `live=${live} archive=${normal.kind}`);
}

export async function normalizeCompletedLedgerArchive(paths) {
	await requireAbsentNoFollow(resolve(paths.root, ".countershape/verify-current/active.lock"), "verifier lock");
	await optionalPrivateDirectoryNoFollow(resolve(paths.root, ".didrun-history"), ".didrun-history").then((metadata) => {
		if (metadata === undefined) fail("RUNBOOK_RECOVERY_DIRECTORY", ".didrun-history:absent");
	});
	for (let transition = 0; transition < 4; transition += 1) {
		const state = await classifyCompletedLedgerArchive(paths);
		if (state === "COMPLETE") return Object.freeze({ ledger: state, paths });
		if (state === "LIVE") {
			await requireSameDevice(paths.liveLedger, resolve(paths.root, ".didrun-history"), "live ledger into commit archive");
			await requireAbsentNoFollow(paths.normalLedgerRoot, "commit-addressed ledger archive");
			await mkdir(paths.normalLedgerRoot, { mode: 0o700 });
			await chmod(paths.normalLedgerRoot, 0o700);
			continue;
		}
		if (state === "EMPTY_WITH_LIVE") {
			await requireSameDevice(paths.liveLedger, paths.normalLedgerRoot, "live ledger into commit archive");
			await recoveryRename(paths.liveLedger, resolve(paths.normalLedgerRoot, ".didrun"), paths.normalLedgerRoot, "live ledger into commit archive");
			continue;
		}
	}
	fail("RUNBOOK_ARCHIVE_TRANSITIONS", "ledger did not converge");
}

async function validateUnitCommit(specification, unit, commit) {
	requireCommitID(commit, unit.id);
	const raw = decodeUTF8(gitBytes(["cat-file", "commit", commit]), "runbook commit object");
	const separator = raw.indexOf("\n\n");
	if (separator < 0 || raw.slice(separator + 2) !== `${unit.subject}\n`) fail("RUNBOOK_COMMIT_MESSAGE", unit.id);
	const headers = raw.slice(0, separator).split("\n");
	const tree = headers.find((line) => line.startsWith("tree "))?.slice(5);
	const parents = headers.filter((line) => line.startsWith("parent ")).map((line) => line.slice(7));
	if (!/^[0-9a-f]{40}$/u.test(tree ?? "") || parents.length !== 1 || !/^[0-9a-f]{40}$/u.test(parents[0]) ||
		!isDeepStrictEqual(changedPathsForCommit(commit), [...unit.allowed_paths].sort())) fail("RUNBOOK_COMMIT_TOPOLOGY", unit.id);
	const identity = (target, label) => {
		const bytes = gitBytes(["show", "-s", "--format=%an%x00%ae%x00%cn%x00%ce%x00", target]);
		if (bytes.length < 5 || bytes.at(-1) !== 0x0a || bytes.at(-2) !== 0x00) fail("RUNBOOK_COMMIT_IDENTITY", label);
		const values = decodeUTF8(bytes.subarray(0, -2), label).split("\0");
		if (values.length !== 4 || values.some((value) => value.length === 0 || /[\u0000-\u001f\u007f]/u.test(value))) fail("RUNBOOK_COMMIT_IDENTITY", label);
		return values;
	};
	if (!isDeepStrictEqual(identity(commit, `${unit.id} commit identity`), identity(parents[0], `${unit.id} parent identity`))) {
		fail("RUNBOOK_COMMIT_IDENTITY", `${unit.id}:must inherit declared parent author and committer identity`);
	}
	if (unit.id === "U7P") {
		if (parents[0] !== specification.sealed_parent.commit) fail("RUNBOOK_COMMIT_PARENT", `${unit.id}:${parents[0]}`);
		await validateC6BCommit(specification, parents[0]);
	} else {
		await validateDeclaredAncestry(specification, unitByID(specification, unit.parent), parents[0]);
	}
	return Object.freeze({ commit, tree, parent: parents[0] });
}

async function validateCommittedUnit(specification, unit, requestedCommit) {
	const commit = gitLine(["rev-parse", "--verify", "HEAD^{commit}"], "runbook commit");
	if (requestedCommit !== commit) fail("RUNBOOK_COMMIT_IDENTITY", `${requestedCommit}:${commit}`);
	return validateUnitCommit(specification, unit, commit);
}

function parseOptionalDirectGitRef(ref, stdout) {
	if (stdout.length === 0) return undefined;
	const value = decodeUTF8(stdout, "failed-attempt ref");
	const fields = value.endsWith("\n") ? value.slice(0, -1).split("\0") : [];
	if (fields.length !== 3 || !/^[0-9a-f]{40}$/u.test(fields[0] ?? "") || fields[1] !== "" || fields[2] !== ref) {
		fail("RUNBOOK_RECOVERY_REF", `${ref}:direct-ref-shape`);
	}
	return fields[0];
}

function optionalGitRef(ref) {
	const result = spawnSync(gitPath, ["--no-replace-objects", "for-each-ref", "--count=2", "--format=%(objectname)%00%(symref)%00%(refname)", ref], {
		cwd: repositoryRoot, encoding: "buffer", timeout: 30_000, maxBuffer: 1024 * 1024, env: gitEnvironment(),
	});
	const stdout = Buffer.isBuffer(result.stdout) ? result.stdout : Buffer.from(result.stdout ?? "", "utf8");
	const stderr = Buffer.isBuffer(result.stderr) ? result.stderr : Buffer.from(result.stderr ?? "", "utf8");
	if (result.error || result.signal || result.status !== 0 || stderr.length !== 0) {
		fail("RUNBOOK_RECOVERY_REF", `${ref}:status=${result.status} signal=${result.signal} error=${result.error?.message ?? "none"} stderr_bytes=${stderr.length}`);
	}
	return parseOptionalDirectGitRef(ref, stdout);
}

export async function recoverFailedPrecommit(unitID, slug, dependencies = {}) {
	const specification = dependencies.specification ?? await loadSpecification();
	const unit = unitByID(specification, unitID);
	const root = dependencies.root ?? repositoryRoot;
	const recoveryPaths = precommitRecoveryPaths(unit, slug, root);
	const candidate = dependencies.candidate ?? (async () => checkCandidate(unitID));
	const head = dependencies.head ?? (() => gitLine(["rev-parse", "--verify", "HEAD^{commit}"], "precommit recovery HEAD"));
	const indexTree = dependencies.indexTree ?? (() => gitLine(["write-tree"], "precommit recovery index tree"));
	const archive = dependencies.archive ?? (() => normalizeFailedAttemptArchives(recoveryPaths, true));
	const write = dependencies.write ?? console.log;
	await candidate();
	const before = Object.freeze({ head: await head(), tree: await indexTree() });
	const archived = await archive();
	await candidate();
	const after = Object.freeze({ head: await head(), tree: await indexTree() });
	if (!isDeepStrictEqual(before, after)) fail("RUNBOOK_RECOVERY_GIT_DRIFT", `${JSON.stringify(before)}:${JSON.stringify(after)}`);
	write(`${unit.id} failed precommit recovery complete: slug=${slug} tree=${after.tree} ledger=${archived.ledger} final_root=${archived.paths.failedFinalRootRelative}`);
	return Object.freeze({ ...after, ...archived });
}

export async function recoverFailedPostcommit(unitID, commit, dependencies = {}) {
	const specification = dependencies.specification ?? await loadSpecification();
	const unit = unitByID(specification, unitID);
	const root = dependencies.root ?? repositoryRoot;
	const recoveryPaths = postcommitRecoveryPaths(unit, commit, root);
	const validateCommit = dependencies.validateCommit ?? (() => validateUnitCommit(specification, unit, commit));
	const head = dependencies.head ?? (() => gitLine(["rev-parse", "--verify", "HEAD^{commit}"], "postcommit recovery HEAD"));
	const indexTree = dependencies.indexTree ?? (() => gitLine(["write-tree"], "postcommit recovery index tree"));
	const refName = `refs/countershape/failed-attempts/${unit.id.toLowerCase()}/${commit}`;
	const readRef = dependencies.readRef ?? (() => optionalGitRef(refName));
	const createRef = dependencies.createRef ?? (() => { gitBytes(["-c", "core.hooksPath=/dev/null", "update-ref", refName, commit, "0".repeat(40)]); });
	const casHead = dependencies.casHead ?? ((parent) => { gitBytes(["-c", "core.hooksPath=/dev/null", "update-ref", "HEAD", parent, commit]); });
	const candidate = dependencies.candidate ?? (async () => checkCandidate(unitID));
	const archive = dependencies.archive ?? (() => normalizeFailedAttemptArchives(recoveryPaths, false));
	const workspace = dependencies.workspace ?? (() => validateRecoveryWorkspace(sealed.tree));
	const write = dependencies.write ?? console.log;
	const sealed = await validateCommit();
	if (sealed.commit !== commit || !/^[0-9a-f]{40}$/u.test(sealed.tree) || !/^[0-9a-f]{40}$/u.test(sealed.parent)) {
		fail("RUNBOOK_RECOVERY_COMMIT", `${unit.id}:validated shape`);
	}
	const beforeHead = await head();
	const beforeRef = await readRef();
	if (![commit, sealed.parent].includes(beforeHead) || (beforeRef !== undefined && beforeRef !== commit) ||
		(beforeHead === sealed.parent && beforeRef !== commit)) {
		fail("RUNBOOK_RECOVERY_GIT_STATE", `head=${beforeHead} ref=${String(beforeRef)} commit=${commit} parent=${sealed.parent}`);
	}
	if (await indexTree() !== sealed.tree) fail("RUNBOOK_RECOVERY_INDEX", "pre-archive tree");
	await workspace();
	const archived = await archive();
	if (await indexTree() !== sealed.tree) fail("RUNBOOK_RECOVERY_INDEX", "post-archive tree");
	await workspace();
	let currentRef = await readRef();
	if (currentRef === undefined) {
		if (await head() !== commit) fail("RUNBOOK_RECOVERY_GIT_STATE", "cannot create ref after HEAD moved");
		await createRef();
		currentRef = await readRef();
	}
	if (currentRef !== commit) fail("RUNBOOK_RECOVERY_REF", `${refName}:${String(currentRef)}`);
	const currentHead = await head();
	if (currentHead === commit) await casHead(sealed.parent);
	else if (currentHead !== sealed.parent) fail("RUNBOOK_RECOVERY_GIT_STATE", `terminal head=${currentHead}`);
	if (await head() !== sealed.parent || await readRef() !== commit || await indexTree() !== sealed.tree) {
		fail("RUNBOOK_RECOVERY_TERMINAL", `${unit.id}:${commit}`);
	}
	const terminalCommit = await validateCommit();
	if (!isDeepStrictEqual(terminalCommit, sealed)) fail("RUNBOOK_RECOVERY_COMMIT_DRIFT", commit);
	await candidate();
	write(`${unit.id} failed postcommit recovery complete: commit=${commit} parent=${sealed.parent} ref=${refName} tree=${sealed.tree} ledger=${archived.ledger} final_root=${archived.paths.failedFinalRootRelative}`);
	return Object.freeze({ ...sealed, ref: refName, ...archived });
}

function validateRecoveryWorkspace(expectedTree) {
	if (gitLine(["write-tree"], "recovery index tree") !== expectedTree) fail("RUNBOOK_RECOVERY_INDEX", "workspace tree");
	if (gitBytes(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--name-only", "-z", "--"]).length !== 0) {
		fail("RUNBOOK_RECOVERY_WORKTREE", "unstaged tracked bytes");
	}
	if (gitBytes(["ls-files", "--others", "--exclude-standard", "-z", "--"]).length !== 0) {
		fail("RUNBOOK_RECOVERY_WORKTREE", "untracked bytes");
	}
}

export async function rotateFinalLedger(unitID, argument, dependencies = {}) {
	const specification = dependencies.specification ?? await loadSpecification();
	const unit = unitByID(specification, unitID);
	const root = dependencies.root ?? repositoryRoot;
	const commit = await (dependencies.commit ?? (() => gitLine(["rev-parse", "--verify", "HEAD^{commit}"], "runbook rotation commit")))();
	const expectedRoot = `.didrun-history/${unit.id.toLowerCase()}-final-${commit.slice(0, 12)}`;
	if (argument !== expectedRoot) fail("RUNBOOK_ARCHIVE_IDENTITY", `${argument}:${commit}`);
	const validateCommit = dependencies.validateCommit ?? (() => validateCommittedUnit(specification, unit, commit));
	const sealed = await validateCommit();
	if (sealed.commit !== commit || !/^[0-9a-f]{40}$/u.test(sealed.tree) || !/^[0-9a-f]{40}$/u.test(sealed.parent)) fail("RUNBOOK_COMMIT_IDENTITY", `${unit.id}:rotation shape`);
	const preflight = dependencies.preflight ?? (async () => {
		if (await realpath(repositoryRoot) !== repositoryRoot || process.cwd() !== repositoryRoot || root !== repositoryRoot) fail("RUNBOOK_ROOT", process.cwd());
		await requirePrivateDirectoryNoFollow(resolve(repositoryRoot, unit.final_root), `${unit.id} final root`);
		await requireOwnedDirectoryNoFollow(resolve(repositoryRoot, ".didrun-history"), ".didrun-history");
	});
	const workspace = dependencies.workspace ?? (() => validateRecoveryWorkspace(sealed.tree));
	const refName = `refs/countershape/failed-attempts/${unit.id.toLowerCase()}/${commit}`;
	const readFailedRef = dependencies.readFailedRef ?? (() => optionalGitRef(refName));
	const paths = postcommitRecoveryPaths(unit, commit, root);
	const normalize = dependencies.normalize ?? (() => normalizeCompletedLedgerArchive(paths));
	const validateArchive = dependencies.validateArchive ?? (async () => {
		const noteBlob = gitLine(["notes", "--ref=didrun", "list", commit], "runbook archive note blob");
		const note = parseNote(gitBytes(["cat-file", "blob", noteBlob]), commit, sealed.tree, unit.claims, hermeticPrefix(unit), unit.allowed_paths);
		await validateArchivedLedger(unit, commit, sealed.tree, note, `${argument}/.didrun`);
	});
	const write = dependencies.write ?? console.log;
	await preflight();
	await workspace();
	if (await readFailedRef() !== undefined) fail("RUNBOOK_ARCHIVE_FAILED_STATE", `${refName}:present`);
	await normalize();
	await workspace();
	if (await readFailedRef() !== undefined) fail("RUNBOOK_ARCHIVE_FAILED_STATE", `${refName}:present-terminal`);
	await validateArchive();
	await workspace();
	if (await readFailedRef() !== undefined) fail("RUNBOOK_ARCHIVE_FAILED_STATE", `${refName}:present-after-validation`);
	write(`${unit.id} final-runbook archive rotation exact: commit=${commit} tree=${sealed.tree} live_ledger=absent archive=${argument}/.didrun chain=valid claims=${unit.claims.length}`);
}

export async function verifyRunbookPreflight(stage, unitID, argument = undefined) {
	const specification = await loadSpecification();
	const unit = unitByID(specification, unitID);
	if (await realpath(repositoryRoot) !== repositoryRoot || process.cwd() !== repositoryRoot) fail("RUNBOOK_ROOT", process.cwd());
	await requireAbsentNoFollow(resolve(repositoryRoot, ".countershape/verify-current/active.lock"), "verifier lock");
	if (stage === "prepare") {
		if (argument !== undefined) fail("RUNBOOK_ARGUMENTS", "prepare argument");
		await requireOwnedDirectoryIfPresentNoFollow(resolve(repositoryRoot, ".countershape"), ".countershape");
		await requireOwnedDirectoryIfPresentNoFollow(resolve(repositoryRoot, ".countershape/evidence"), ".countershape/evidence");
		await requireOwnedDirectoryIfPresentNoFollow(resolve(repositoryRoot, ".didrun-history"), ".didrun-history");
		await checkCandidate(unitID);
		await requireAbsentNoFollow(resolve(repositoryRoot, ".didrun"), "live didrun");
		await requireAbsentNoFollow(resolve(repositoryRoot, unit.final_root), `${unit.id} final root`);
		console.log(`${unit.id} final-runbook preparation no-follow exact: live_ledger=absent lock=absent final_root=absent owned_roots_or_absent=true`);
		return;
	}
	if (typeof argument !== "string") fail("RUNBOOK_ARGUMENTS", `${stage}:${String(argument)}`);
	if (stage === "commit") {
		const sealed = await validateCommittedUnit(specification, unit, argument);
		console.log(`${unit.id} final-runbook commit exact: commit=${sealed.commit} tree=${sealed.tree} parent=${sealed.parent} message=exact roster=${unit.allowed_paths.length}`);
		return;
	}
	const commit = gitLine(["rev-parse", "--verify", "HEAD^{commit}"], `runbook ${stage} commit`);
	const subject = gitLine(["show", "-s", "--format=%s", commit], `runbook ${stage} subject`);
	if (stage === "report") {
		const expectedReport = `.countershape/evidence/${unit.id.toLowerCase()}-final-${commit.slice(0, 12)}.html`;
		if (argument !== expectedReport || subject !== unit.subject) fail("RUNBOOK_REPORT_IDENTITY", `${argument}:${subject}`);
		await requireOwnedDirectoryNoFollow(resolve(repositoryRoot, ".countershape"), ".countershape");
		await requireOwnedDirectoryNoFollow(resolve(repositoryRoot, ".countershape/evidence"), ".countershape/evidence");
		await requireAbsentNoFollow(resolve(repositoryRoot, argument), "commit-addressed HTML report");
		console.log(`${unit.id} final-runbook report no-follow exact: commit=${commit} report=${argument} absent=true`);
		return;
	}
	if (!["archive", "archive-complete"].includes(stage)) fail("RUNBOOK_ARGUMENTS", `${stage}:${argument}`);
	const expectedRoot = `.didrun-history/${unit.id.toLowerCase()}-final-${commit.slice(0, 12)}`;
	if (argument !== expectedRoot || subject !== unit.subject) fail("RUNBOOK_ARCHIVE_IDENTITY", `${argument}:${subject}`);
	await requireOwnedDirectoryNoFollow(resolve(repositoryRoot, ".didrun-history"), ".didrun-history");
	await requirePrivateDirectoryNoFollow(resolve(repositoryRoot, unit.final_root), `${unit.id} final root`);
	if (stage === "archive") {
		await requireOwnedDirectoryNoFollow(resolve(repositoryRoot, ".didrun"), ".didrun");
		await requireAbsentNoFollow(resolve(repositoryRoot, argument), "commit-addressed archive root");
		console.log(`${unit.id} final-runbook archive no-follow exact: commit=${commit} live_ledger=owned lock=absent archive=absent`);
		return;
	}
	await requireAbsentNoFollow(resolve(repositoryRoot, ".didrun"), "rotated live didrun");
	await requireOwnedDirectoryNoFollow(resolve(repositoryRoot, argument), "commit-addressed archive root");
	const tree = gitLine(["rev-parse", "--verify", `${commit}^{tree}`], "runbook archive tree");
	const noteBlob = gitLine(["notes", "--ref=didrun", "list", commit], "runbook archive note blob");
	const note = parseNote(gitBytes(["cat-file", "blob", noteBlob]), commit, tree, unit.claims, hermeticPrefix(unit), unit.allowed_paths);
	await validateArchivedLedger(unit, commit, tree, note, `${argument}/.didrun`);
	console.log(`${unit.id} final-runbook archive closure exact: commit=${commit} tree=${tree} live_ledger=absent archive=${argument}/.didrun chain=valid claims=${unit.claims.length}`);
}

export function hermeticPrefix(unit, root = repositoryRoot) {
	const runRoot = resolve(root, unit.final_root);
	return Object.freeze([
		"/usr/bin/env", "-i",
		`HOME=${resolve(runRoot, "home")}`,
		`PWD=${root}`,
		`TMPDIR=${resolve(runRoot, "tmp")}`,
		`GOTMPDIR=${resolve(runRoot, "gotmp")}`,
		`GOCACHE=${resolve(runRoot, "gocache")}`,
		`GOPATH=${resolve(runRoot, "gopath")}`,
		`GOMODCACHE=${resolve(runRoot, "gomodcache")}`,
		"GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off",
		"GOFLAGS=-mod=readonly -buildvcs=false -p=1", "CGO_ENABLED=1", "GOMAXPROCS=2",
		"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "NODE_OPTIONS=", "NODE_PATH=",
		"PATH=/opt/homebrew/bin:/usr/bin:/bin", "SHELL=/bin/sh",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_LAZY_FETCH=1",
		"GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0",
		"COUNTERSHAPE_GO=/opt/homebrew/bin/go", "COUNTERSHAPE_NODE=/opt/homebrew/bin/node",
		"COUNTERSHAPE_GIT=/usr/bin/git", "COUNTERSHAPE_SH=/bin/sh",
		"COUNTERSHAPE_GOFMT=/opt/homebrew/bin/gofmt", "COUNTERSHAPE_CC=/usr/bin/clang",
		"COUNTERSHAPE_CXX=/usr/bin/clang++", "CC=/usr/bin/clang", "CXX=/usr/bin/clang++",
			"PYTHONPATH=", "PYTHONHOME=", "PYTHONNOUSERSITE=1", "PYTHONSAFEPATH=1", "PYTHONDONTWRITEBYTECODE=1", "PYTHONHASHSEED=0", "PYTHONUTF8=1", "GIT_NO_REPLACE_OBJECTS=1",
	]);
}

function parseJSONLines(bytes, label) {
	const text = decodeUTF8(bytes, label);
	if (!text.endsWith("\n")) fail("JSONL_DELIMITER", label);
	return text.slice(0, -1).split("\n").map((line, index) => {
		try {
			const value = JSON.parse(line);
			if (canonicalJSON(value) !== line) fail("JSONL_CANONICAL", `${label}:${index}`);
			return value;
		} catch (error) { if (String(error.message).startsWith("U7_PLAN_")) throw error; fail("JSONL", `${label}:${index}:${error.message}`); }
	});
}

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

export function computeDidrunEntryHash(entry) {
	if (!entry || typeof entry !== "object" || !Number.isSafeInteger(entry.index) || typeof entry.prev_hash !== "string" || entry.event === undefined) fail("ENTRY_HASH_INPUT", String(entry?.index));
	const body = Buffer.from(canonicalJSON({ index: entry.index, prev: entry.prev_hash, event: entry.event }), "ascii");
	return sha256Hex(Buffer.concat([Buffer.from(entry.prev_hash, "ascii"), body]));
}

export function validateRecordedChildIntervals(entries, label) {
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

function requireEffectiveEnvironment(unit) {
	const prefix = hermeticPrefix(unit);
	for (const assignment of prefix.slice(2)) {
		const split = assignment.indexOf("=");
		const name = assignment.slice(0, split);
		const value = assignment.slice(split + 1);
		if (process.env[name] !== value) fail("ENVIRONMENT", `${name}:${process.env[name] ?? "<absent>"}`);
	}
}

export async function verifyPreseal(unitID) {
	const specification = await loadSpecification();
	const unit = unitByID(specification, unitID);
	const expectedInvocation = unit.claims.at(-1).command;
	const observedInvocation = [process.argv0, relative(repositoryRoot, resolve(process.argv[1])), ...process.argv.slice(2)];
	if (!isDeepStrictEqual(observedInvocation, expectedInvocation) || await realpath(process.execPath) !== await realpath(expectedInvocation[0])) fail("PRESEAL_INVOCATION", JSON.stringify(observedInvocation));
	requireEffectiveEnvironment(unit);
	await checkCandidate(unitID);
	const priorCount = unit.claims.length - 1;
	const show = checkedSpawn(didrunPath, ["show", "--session"], { encoding: "utf8", env: recorderEnvironment(specification.runtime_authority) });
	if (!show.startsWith(`session: ${priorCount} events  chain intact\n`)) fail("DIDRUN_CHAIN", show.slice(0, 160));
	const sessionBytes = await readRegular(resolve(repositoryRoot, ".didrun/session.log"), 32 * 1024 * 1024);
	const claimBytes = await readRegular(resolve(repositoryRoot, ".didrun/claims.jsonl"), 8 * 1024 * 1024);
	const session = parseJSONLines(sessionBytes, "session.log");
	const claims = parseJSONLines(claimBytes, "claims.jsonl");
	if (session.length !== priorCount || claims.length !== priorCount) fail("PRESEAL_CARDINALITY", `${session.length}:${claims.length}:${priorCount}`);
	const prefix = hermeticPrefix(unit);
	let tree;
	let environmentFingerprint;
	for (let index = 0; index < priorCount; index += 1) {
		const entry = session[index];
		const event = entry?.event;
		const previous = index === 0 ? "0".repeat(64) : session[index - 1]?.entry_hash;
		if (!exactKeys(entry, ["entry_hash", "event", "index", "prev_hash"]) || entry.index !== index || entry.prev_hash !== previous ||
			!/^[0-9a-f]{64}$/u.test(entry.entry_hash ?? "") || computeDidrunEntryHash(entry) !== entry.entry_hash) fail("LEDGER_ENTRY", String(index));
		const expectedArgv = [...prefix, ...unit.claims[index].command];
		if (!event || event.exit_code !== 0 || event.observed_via !== "wrapper" || event.coverage !== "complete" ||
			event.submodule_dirty !== false || event.cwd !== repositoryRoot || event.tree_before === null ||
			event.tree_before !== event.tree_after || !isDeepStrictEqual(event.argv, expectedArgv)) fail("LEDGER_EVENT", String(index));
		tree ??= event.tree_before;
		if (!/^[0-9a-f]{16}$/u.test(event.env_fingerprint ?? "")) fail("LEDGER_ENVIRONMENT_FINGERPRINT", String(index));
		environmentFingerprint ??= event.env_fingerprint;
		if (event.tree_before !== tree || event.env_fingerprint !== environmentFingerprint) fail("LEDGER_EPOCH", String(index));
		const claim = claims[index];
		if (!exactKeys(claim, ["ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) ||
			claim.ctype !== unit.claims[index].type || claim.label !== unit.claims[index].label || claim.declared_at_index !== index ||
			!isDeepStrictEqual(claim.event_indices, [index]) || !isDeepStrictEqual(claim.pathspecs, unit.allowed_paths)) fail("LEDGER_CLAIM", String(index));
	}
	validateRecordedChildIntervals(session, `${unitID}:preseal`);
	try {
		await lstat(resolve(repositoryRoot, ".didrun/seals.jsonl"));
		fail("PRESEAL_ALREADY_SEALED", unitID);
	} catch (error) {
		if (error?.message?.startsWith("U7_PLAN_")) throw error;
		if (error.code !== "ENOENT") throw error;
	}
	const stagedTree = gitLine(["write-tree"], "staged tree");
	if (stagedTree !== tree) fail("PRESEAL_TREE", `${stagedTree}:${tree}`);
	await checkCandidate(unitID);
	const terminalSession = await readRegular(resolve(repositoryRoot, ".didrun/session.log"), 32 * 1024 * 1024);
	const terminalClaims = await readRegular(resolve(repositoryRoot, ".didrun/claims.jsonl"), 8 * 1024 * 1024);
	if (!terminalSession.equals(sessionBytes) || !terminalClaims.equals(claimBytes) || gitLine(["write-tree"], "terminal staged tree") !== tree) {
		fail("PRESEAL_TERMINAL_DRIFT", unitID);
	}
	console.log(`${unitID} preceding didrun chain exact: events=${priorCount} claims=${priorCount} green=${priorCount} tree=${tree} chain=valid seal=absent`);
}

const receiptPath = "spec/verification/u7-receipt.json";
const receiptBlockStart = "<!-- U7D-SOURCE-RECEIPTS:START -->";
const receiptBlockEnd = "<!-- U7D-SOURCE-RECEIPTS:END -->";
const pendingReceiptBlockStart = "<!-- U7D-SOURCE-RECEIPT-PENDING:START -->";
const pendingReceiptBlockEnd = "<!-- U7D-SOURCE-RECEIPT-PENDING:END -->";
const pendingReceiptBlock = `${pendingReceiptBlockStart}\n### U7D source receipt pending\n\n- **State:** \`ABSENT\`. U7D source evidence is not receipted until the separately sealed U7R reconciliation boundary.\n- **Ceiling:** no local HTML or secret-bearing ledger snapshot is claimed by this placeholder, and U7D cannot receipt itself.\n${pendingReceiptBlockEnd}`;
const requiredLedgerFiles = Object.freeze([".gitignore", "claims.jsonl", "seals.jsonl", "session.log"]);

function validateReceiptPhase(unitID, handoffText) {
	const receiptStarts = occurrenceCount(handoffText, receiptBlockStart);
	const receiptEnds = occurrenceCount(handoffText, receiptBlockEnd);
	const pendingStarts = occurrenceCount(handoffText, pendingReceiptBlockStart);
	const pendingEnds = occurrenceCount(handoffText, pendingReceiptBlockEnd);
	if (["U7P", "U7M", "U7A", "U7B", "U7C"].includes(unitID) ? receiptStarts + receiptEnds + pendingStarts + pendingEnds !== 0 :
		unitID === "U7D" ? receiptStarts !== 0 || receiptEnds !== 0 || pendingStarts !== 1 || pendingEnds !== 1 :
		unitID === "U7R" ? receiptStarts !== 1 || receiptEnds !== 1 || pendingStarts !== 0 || pendingEnds !== 0 : true) {
		fail("RECEIPT_PHASE", `${unitID}:${receiptStarts}:${receiptEnds}:${pendingStarts}:${pendingEnds}`);
	}
	if (unitID === "U7R") requireVisibleMarkers(handoffText, receiptBlockStart, receiptBlockEnd, "U7D_RECEIPT_HANDOFF");
}

function validatePendingProjectionDocuments(specification, documents) {
	if (!(documents instanceof Map) || documents.size !== specification.receipt_contract.projection_paths.length) fail("RECEIPT_PENDING_SET", "cardinality");
	for (const path of specification.receipt_contract.projection_paths) {
		const text = documents.get(path);
		if (typeof text !== "string") fail("RECEIPT_PENDING_SET", path);
		validateReceiptJurisdiction(path, text, pendingReceiptBlock, pendingReceiptBlockStart, pendingReceiptBlockEnd);
	}
}

function sha256Hex(bytes) {
	return createHash("sha256").update(bytes).digest("hex");
}

function ledgerManifestDigest(manifest) {
	return sha256Hex(Buffer.from(`${JSON.stringify(manifest)}\n`, "utf8"));
}

function validPositiveInteger(value, maximum = Number.MAX_SAFE_INTEGER) {
	return Number.isSafeInteger(value) && value > 0 && value <= maximum;
}

function validSHA256(value) {
	return typeof value === "string" && /^[0-9a-f]{64}$/u.test(value) && !/^0+$/u.test(value);
}

function runtimeAuthorityDigest(specification) {
	return sha256Hex(Buffer.from(`${JSON.stringify(specification.runtime_authority)}\n`, "utf8"));
}

function sourceStudyEventIndex(specification) {
	const source = unitByID(specification, specification.receipt_contract.source_boundary);
	const matches = source.claims.flatMap((claim, index) => isDeepStrictEqual(claim.command, specification.receipt_contract.study_event_command) ? [index] : []);
	if (!isDeepStrictEqual(matches, [8])) fail("STUDY_EVENT_CONTRACT", JSON.stringify(matches));
	return matches[0];
}

function studyManifestPaths(contract) {
	const paths = [...Object.values(contract.study_harness.deterministic_artifact_paths), ...Object.values(contract.study_harness.fresh_artifact_paths)];
	for (const phase of contract.study_harness.phase_budgets.slice(2)) {
		for (let index = 1; index <= phase.per_run_trials; index += 1) paths.push(`phases/${phase.id}/trial-${String(index).padStart(3, "0")}.json`);
	}
	return paths.sort();
}

function parseStudyTimeReport(text) {
	if (typeof text !== "string" || text.includes("\r") || text.startsWith("\ufeff") || !text.endsWith("\n")) fail("STUDY_TIME_REPORT", "encoding or newline");
	const lines = text.slice(0, -1).split("\n");
	const labels = ["maximum resident set size", "average shared memory size", "average unshared data size", "average unshared stack size", "page reclaims", "page faults", "swaps", "block input operations", "block output operations", "messages sent", "messages received", "signals received", "voluntary context switches", "involuntary context switches", "instructions retired", "cycles elapsed", "peak memory footprint"];
	if (lines.length !== labels.length + 3 || !/^real [0-9]+\.[0-9]{2}$/u.test(lines[0]) || !/^user [0-9]+\.[0-9]{2}$/u.test(lines[1]) || !/^sys [0-9]+\.[0-9]{2}$/u.test(lines[2])) fail("STUDY_TIME_REPORT", "header or cardinality");
	let peakRSS;
	for (const [index, label] of labels.entries()) {
		const match = /^\s*([0-9]+)\s+(.+)$/u.exec(lines[index + 3]);
		if (match === null || match[2] !== label || !Number.isSafeInteger(Number(match[1]))) fail("STUDY_TIME_REPORT", `row ${index}`);
		if (label === "maximum resident set size") peakRSS = Number(match[1]);
	}
	const real = Number(lines[0].slice(5));
	if (!Number.isFinite(real) || real < 0 || !validPositiveInteger(peakRSS, 16 * 1024 ** 3)) fail("STUDY_TIME_REPORT", "real or RSS");
	return Object.freeze({ wall_time_ms: Math.max(1, Math.ceil(real * 1000)), peak_rss_bytes: peakRSS });
}

function studyProcessEnvironmentDigest(specification, study, run, name) {
	const source = unitByID(specification, "U7D");
	const assignments = new Map(hermeticPrefix(source).slice(2).map((entry) => [entry.slice(0, entry.indexOf("=")), entry.slice(entry.indexOf("=") + 1)]));
	const admitted = ["PATH", "LANG", "LC_ALL", "TZ", "NO_COLOR", "HOME", "TMPDIR", "GOTMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE", "GOFLAGS", "GOMAXPROCS", "CGO_ENABLED", "COUNTERSHAPE_NODE", "COUNTERSHAPE_GO", "COUNTERSHAPE_GIT", "COUNTERSHAPE_SH", "COUNTERSHAPE_GOFMT", "COUNTERSHAPE_CC", "COUNTERSHAPE_CXX", "CC", "CXX"];
	const environment = Object.fromEntries(admitted.map((key) => [key, assignments.get(key)]));
	const runRoot = resolve(repositoryRoot, source.final_root, "tmp/studies", study.id, `run-${run.ordinal}`);
	Object.assign(environment, { HOME: resolve(runRoot, "home"), TMPDIR: resolve(runRoot, "tmp"), COUNTERSHAPE_STUDY_DOMAIN: study.id, COUNTERSHAPE_STUDY_ORDINAL: String(run.ordinal), NODE_OPTIONS: "", NODE_PATH: "", GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1", GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0", GIT_NO_REPLACE_OBJECTS: "1" });
	if (name === "execute") environment.COUNTERSHAPE_EVIDENCE_ROOT = resolve(runRoot, "evidence");
	const rows = Object.keys(environment).sort().map((key) => `${key}=${environment[key]}`);
	return sha256Hex(Buffer.from(`${rows.join("\n")}\n`, "utf8"));
}

export function validateStudyEvidence(value, specification, expectedDriverDigests) {
	const contract = specification.receipt_contract;
	const harness = contract.study_harness;
	const expectedAuthority = { protocol: harness.protocol, path: harness.path, sha256: harness.sha256, protocol_sha256: harness.protocol_sha256, observation_authority: harness.observation_authority, semantic_ceiling: harness.semantic_ceiling };
	if (!exactKeys(value, ["schema_version", "phase", "harness_authority", "environment", "studies", "milestone_verdict", "honest_fallback", "unreceipted"]) || value.schema_version !== contract.study_evidence_schema || value.phase !== "U7D" ||
		!isDeepStrictEqual(value.harness_authority, expectedAuthority) || value.milestone_verdict !== contract.milestone_verdict || value.honest_fallback !== contract.honest_fallback ||
		!isDeepStrictEqual(value.unreceipted, contract.unreceipted) || !Array.isArray(value.studies) || !isDeepStrictEqual(value.studies.map((study) => study?.id), contract.study_domains) ||
		!exactKeys(expectedDriverDigests, contract.study_domains) || contract.study_domains.some((domain) => !validSHA256(expectedDriverDigests[domain]))) fail("STUDY_EVIDENCE_SHAPE", "root or harness authority");
	const environment = value.environment;
	const versions = Object.fromEntries(specification.runtime_authority.admitted_tools.map((tool) => [tool.name, tool.version]));
	if (!exactKeys(environment, ["platform", "kernel_release", "architecture", "git_version", "go_version", "node_version", "cpu_model", "logical_cpu_count", "memory_bytes", "runtime_authority_sha256"]) || environment.platform !== "darwin" || environment.architecture !== "arm64" ||
		!/^\d+(?:\.\d+){1,3}$/u.test(environment.kernel_release ?? "") || environment.git_version !== versions.git || environment.go_version !== versions.go || environment.node_version !== versions.node ||
		typeof environment.cpu_model !== "string" || environment.cpu_model.length < 3 || environment.cpu_model.length > 160 || /[\u0000-\u001f\u007f]/u.test(environment.cpu_model) ||
		!validPositiveInteger(environment.logical_cpu_count, 512) || !validPositiveInteger(environment.memory_bytes, 2 ** 50) || environment.runtime_authority_sha256 !== runtimeAuthorityDigest(specification)) fail("STUDY_ENVIRONMENT", "identity");
	const tools = Object.fromEntries(specification.runtime_authority.admitted_tools.map((tool) => [tool.name, tool]));
	const manifestPaths = studyManifestPaths(contract);
	const allFresh = new Set(); const allDeterministic = new Set(); const fixtureCommits = new Set(); const fixtureTrees = new Set(); const gitDirectories = new Set();
	for (const study of value.studies) {
		const studyKeys = ["id", "fixture_commit", "fixture_authority", "logical_product_command", "run_count", "trial_count", "budget_trial_count", "observed_subject_process_wall_time_ms", "budget_subject_process_wall_time_ms", "observed_subject_process_peak_rss_bytes", "budget_subject_process_peak_rss_bytes", "observation_authority", "phases", "deterministic_artifacts", "runs"];
		if (!exactKeys(study, studyKeys) || !/^[0-9a-f]{40}$/u.test(study.fixture_commit ?? "") || /^0+$/u.test(study.fixture_commit) || !isDeepStrictEqual(study.logical_product_command, ["countershape", "study", study.id, "--json"]) ||
			study.run_count !== contract.study_run_count_per_domain || study.trial_count !== contract.study_trial_budget_per_domain || study.budget_trial_count !== contract.study_trial_budget_per_domain ||
			study.budget_subject_process_wall_time_ms !== contract.study_subject_process_wall_time_budget_ms_per_domain || study.budget_subject_process_peak_rss_bytes !== contract.study_subject_process_peak_rss_budget_bytes_per_domain ||
			!validPositiveInteger(study.observed_subject_process_wall_time_ms, study.budget_subject_process_wall_time_ms) || !validPositiveInteger(study.observed_subject_process_peak_rss_bytes, study.budget_subject_process_peak_rss_bytes) ||
			study.observation_authority !== harness.observation_authority || !isDeepStrictEqual(study.phases, contract.study_phase_budgets) || !Array.isArray(study.runs) || study.runs.length !== study.run_count || !exactKeys(study.deterministic_artifacts, contract.deterministic_artifact_fields)) fail("STUDY_DOMAIN", String(study?.id));
		fixtureCommits.add(study.fixture_commit);
		for (const digest of Object.values(study.deterministic_artifacts)) { if (!validSHA256(digest)) fail("STUDY_DETERMINISTIC", study.id); allDeterministic.add(digest); }
		const fixture = study.fixture_authority;
		if (!exactKeys(fixture, ["head_commit", "tree", "clean_status_bytes", "clean_status_sha256", "repository_count", "fresh_repository_per_run", "repository_git_directories"]) || fixture.head_commit !== study.fixture_commit || !/^[0-9a-f]{40}$/u.test(fixture.tree ?? "") || /^0+$/u.test(fixture.tree) || fixture.clean_status_bytes !== 0 ||
			fixture.clean_status_sha256 !== sha256Hex(Buffer.alloc(0)) || fixture.repository_count !== study.run_count || fixture.fresh_repository_per_run !== true || !Array.isArray(fixture.repository_git_directories)) fail("STUDY_FIXTURE_AUTHORITY", study.id);
		fixtureTrees.add(fixture.tree);
		let studyWall = 0; let studyPeak = 0;
		for (const [index, run] of study.runs.entries()) {
			const runKeys = ["ordinal", "trial_count", "trial_budget", "observed_subject_process_wall_time_ms", "budget_subject_process_wall_time_ms", "observed_subject_process_peak_rss_bytes", "budget_subject_process_peak_rss_bytes", "observation_authority", "fixture", "driver_authority", "execution", "processes", "phases", "product_result", "evidence_root", "evidence_manifest", "evidence_manifest_sha256", "deterministic_artifacts", ...Object.keys(harness.fresh_artifact_paths), "fixture_invocation_sha256"];
			const runBudget = contract.study_subject_process_wall_time_budget_ms_per_domain / contract.study_run_count_per_domain;
			const expectedPhases = contract.study_phase_budgets.map((phase) => ({ id: phase.id, trial_count: phase.trial_count / contract.study_run_count_per_domain, trial_budget: phase.trial_budget / contract.study_run_count_per_domain }));
			if (!Number.isSafeInteger(runBudget) || expectedPhases.some((phase) => !Number.isSafeInteger(phase.trial_count) || !Number.isSafeInteger(phase.trial_budget)) || !exactKeys(run, runKeys) || run.ordinal !== index + 1 || run.trial_count !== contract.study_trial_budget_per_domain / contract.study_run_count_per_domain || run.trial_budget !== run.trial_count ||
				run.budget_subject_process_wall_time_ms !== runBudget || run.budget_subject_process_peak_rss_bytes !== contract.study_subject_process_peak_rss_budget_bytes_per_domain || !validPositiveInteger(run.observed_subject_process_wall_time_ms, run.budget_subject_process_wall_time_ms) ||
				!validPositiveInteger(run.observed_subject_process_peak_rss_bytes, run.budget_subject_process_peak_rss_bytes) || run.observation_authority !== harness.observation_authority || !isDeepStrictEqual(run.phases, expectedPhases) || !isDeepStrictEqual(run.deterministic_artifacts, study.deterministic_artifacts)) fail("STUDY_RUN", `${study.id}:${index + 1}`);
			const runRoot = resolve(repositoryRoot, ".countershape/u7d-final/tmp/studies", study.id, `run-${run.ordinal}`);
			const executionCwd = resolve(runRoot, "fixture"); const executionBinary = resolve(executionCwd, "countershape");
			const expectedExecution = { cwd: executionCwd, argv: [executionBinary, "study", study.id, "--json"], executable_path: executionBinary, executable_sha256: study.deterministic_artifacts.reference_binary_sha256 };
			if (!isDeepStrictEqual(run.execution, expectedExecution) || run.fixture_invocation_sha256 !== sha256Hex(Buffer.from(`${JSON.stringify(expectedExecution)}\n`, "utf8")) || run.evidence_root !== relative(repositoryRoot, resolve(runRoot, "evidence"))) fail("STUDY_EXECUTION", `${study.id}:${index + 1}`);
			if (!exactKeys(run.driver_authority, ["path", "sha256"]) || run.driver_authority.path !== harness.driver_by_domain[study.id] || run.driver_authority.sha256 !== expectedDriverDigests[study.id]) fail("STUDY_DRIVER", `${study.id}:${index + 1}`);
			const runFixture = run.fixture;
			const gitDirectory = relative(repositoryRoot, resolve(executionCwd, ".git"));
			if (!exactKeys(runFixture, ["head_commit", "tree", "commit_count", "git_directory", "git_common_directory", "git_alternates", "clean_status_bytes", "clean_status_sha256"]) || runFixture.head_commit !== study.fixture_commit || runFixture.tree !== fixture.tree || runFixture.commit_count !== 1 || runFixture.git_directory !== gitDirectory ||
				runFixture.git_common_directory !== gitDirectory || runFixture.git_alternates !== "ABSENT" || runFixture.clean_status_bytes !== 0 || runFixture.clean_status_sha256 !== sha256Hex(Buffer.alloc(0))) fail("STUDY_FIXTURE", `${study.id}:${index + 1}`);
			gitDirectories.add(gitDirectory);
			if (!exactKeys(run.product_result, ["schema_version", "domain", "ordinal", "status"]) || run.product_result.schema_version !== harness.product_result_schema || run.product_result.domain !== study.id || run.product_result.ordinal !== run.ordinal || run.product_result.status !== "GREEN" || !exactKeys(run.processes, ["prepare", "compile", "execute"])) fail("STUDY_PRODUCT_RESULT", `${study.id}:${index + 1}`);
			let processWall = 0; let processPeak = 0;
			for (const name of ["prepare", "compile", "execute"]) {
				const observation = run.processes[name];
				const observationKeys = ["name", "observer_argv", "cwd", "executable_path", "executable_sha256", "observer_path", "observer_sha256", "environment_sha256", "wall_time_ms", "peak_rss_bytes", "time_report_path", "time_report_bytes", "time_report_sha256", "time_report", "stdout_bytes", "stdout_sha256"];
				const reportBytes = Buffer.from(observation?.time_report ?? "", "utf8"); const parsed = parseStudyTimeReport(observation?.time_report);
				let cwd; let executable; let executableSha; let args; let stdout;
				if (name === "prepare") { cwd = repositoryRoot; executable = tools.node.path; executableSha = tools.node.sha256; args = [resolve(repositoryRoot, harness.driver_by_domain[study.id]), "--prepare", "--domain", study.id, "--ordinal", String(run.ordinal), "--fixture-root", executionCwd]; stdout = Buffer.alloc(0); }
				else if (name === "compile") { cwd = repositoryRoot; executable = tools.go.path; executableSha = tools.go.sha256; args = ["build", "-trimpath", "-mod=readonly", "-buildvcs=false", "-o", executionBinary, "./cmd/countershape"]; stdout = Buffer.alloc(0); }
				else { cwd = executionCwd; executable = executionBinary; executableSha = expectedExecution.executable_sha256; args = ["study", study.id, "--json"]; stdout = Buffer.from(`${JSON.stringify(run.product_result)}\n`, "utf8"); }
				const reportPath = resolve(runRoot, "observations", `${name}.time`);
				if (!exactKeys(observation, observationKeys) || observation.name !== name || observation.cwd !== cwd || observation.executable_path !== executable || observation.executable_sha256 !== executableSha || observation.observer_path !== tools.time.path || observation.observer_sha256 !== tools.time.sha256 ||
					observation.environment_sha256 !== studyProcessEnvironmentDigest(specification, study, run, name) || !isDeepStrictEqual(observation.observer_argv, [tools.time.path, "-p", "-l", "-o", reportPath, executable, ...args]) || observation.time_report_path !== relative(repositoryRoot, reportPath) ||
					observation.time_report_bytes !== reportBytes.length || observation.time_report_sha256 !== sha256Hex(reportBytes) || observation.wall_time_ms !== parsed.wall_time_ms || observation.peak_rss_bytes !== parsed.peak_rss_bytes || observation.stdout_bytes !== stdout.length || observation.stdout_sha256 !== sha256Hex(stdout)) fail("STUDY_PROCESS", `${study.id}:${index + 1}:${name}`);
				processWall += observation.wall_time_ms; processPeak = Math.max(processPeak, observation.peak_rss_bytes);
			}
			if (run.observed_subject_process_wall_time_ms !== processWall || run.observed_subject_process_peak_rss_bytes !== processPeak || !Array.isArray(run.evidence_manifest) || !isDeepStrictEqual(run.evidence_manifest.map((row) => row?.path), manifestPaths)) fail("STUDY_PROCESS_TOTALS", `${study.id}:${index + 1}`);
			for (const row of run.evidence_manifest) if (!exactKeys(row, ["path", "mode", "bytes", "sha256"]) || row.mode !== "0600" || !validPositiveInteger(row.bytes, 16 * 1024 * 1024) || !validSHA256(row.sha256)) fail("STUDY_MANIFEST", `${study.id}:${index + 1}`);
			if (run.evidence_manifest_sha256 !== sha256Hex(Buffer.from(`${JSON.stringify(run.evidence_manifest)}\n`, "utf8"))) fail("STUDY_MANIFEST_DIGEST", `${study.id}:${index + 1}`);
			const manifest = new Map(run.evidence_manifest.map((row) => [row.path, row.sha256]));
			for (const [field, path] of Object.entries(harness.deterministic_artifact_paths)) if (run.deterministic_artifacts[field] !== manifest.get(path)) fail("STUDY_MANIFEST_BINDING", `${study.id}:${field}`);
			for (const [field, path] of Object.entries(harness.fresh_artifact_paths)) if (run[field] !== manifest.get(path)) fail("STUDY_MANIFEST_BINDING", `${study.id}:${field}`);
			for (const field of contract.fresh_run_digest_fields) { const digest = run[field]; if (!validSHA256(digest) || allFresh.has(digest)) fail("STUDY_FRESH_DIGEST", `${study.id}:${index + 1}:${field}`); allFresh.add(digest); }
			studyWall += run.observed_subject_process_wall_time_ms; studyPeak = Math.max(studyPeak, run.observed_subject_process_peak_rss_bytes);
		}
		if (!isDeepStrictEqual(fixture.repository_git_directories, study.runs.map((run) => run.fixture.git_directory)) || study.observed_subject_process_wall_time_ms !== studyWall || study.observed_subject_process_peak_rss_bytes !== studyPeak) fail("STUDY_TOTALS", study.id);
	}
	if (fixtureCommits.size !== contract.study_domains.length || fixtureTrees.size !== contract.study_domains.length || gitDirectories.size !== contract.study_domains.length * contract.study_run_count_per_domain || [...allFresh].some((digest) => allDeterministic.has(digest))) fail("STUDY_DIGEST_DOMAINS", "fixture or deterministic/fresh separation");
	return value;
}

function validateLedgerManifest(value) {
	if (!Array.isArray(value) || value.length < requiredLedgerFiles.length + 1 || value.length > 10_000) fail("RECEIPT_LEDGER_MANIFEST", "cardinality");
	const paths = [];
	for (const entry of value) {
		if (!exactKeys(entry, ["path", "bytes", "sha256"]) || typeof entry.path !== "string" ||
			!validPositiveInteger(entry.bytes, 32 * 1024 * 1024) && entry.bytes !== 0 ||
			typeof entry.sha256 !== "string" || !/^[0-9a-f]{64}$/u.test(entry.sha256)) fail("RECEIPT_LEDGER_MANIFEST", "entry");
		if (!requiredLedgerFiles.includes(entry.path) && !/^objects\/[0-9a-f]{64}$/u.test(entry.path)) fail("RECEIPT_LEDGER_MANIFEST", entry.path);
		if (entry.path.startsWith("objects/") && entry.path.slice("objects/".length) !== entry.sha256) fail("RECEIPT_LEDGER_OBJECT", entry.path);
		paths.push(entry.path);
	}
	if (!isDeepStrictEqual(paths, [...paths].sort()) || new Set(paths).size !== paths.length ||
		requiredLedgerFiles.some((path) => !paths.includes(path))) fail("RECEIPT_LEDGER_MANIFEST", "order, uniqueness, or required files");
	return value;
}

export function validateReceiptDeclaration(value, specification, admittedDriverDigests) {
	const source = unitByID(specification, "U7D");
	const keys = [
		"schema_version", "source_boundary", "source_commit", "source_tree", "source_parent", "source_subject",
		"source_note_blob", "source_note_body_sha256", "source_claims", "source_strict_exit", "source_secrets_override",
		"source_html_path", "source_html_bytes", "source_html_sha256", "source_html_authority",
		"source_ledger_archive", "source_ledger_authority", "source_ledger_file_count", "source_ledger_event_count",
		"source_ledger_claim_count", "source_ledger_seal_count", "source_ledger_manifest", "source_ledger_manifest_sha256",
		"source_study_event_index", "source_study_stdout_blob", "source_study_stdout_bytes", "source_study_evidence_sha256", "source_study_evidence",
	];
	if (!exactKeys(value, keys) || value.schema_version !== "countershape/u7-receipt/v1" || value.source_boundary !== "U7D" ||
		value.source_subject !== source.subject || value.source_strict_exit !== 0 || typeof value.source_secrets_override !== "boolean") fail("RECEIPT_SHAPE", "root");
	for (const name of ["source_commit", "source_tree", "source_parent", "source_note_blob"]) {
		if (typeof value[name] !== "string" || !/^[0-9a-f]{40}$/u.test(value[name]) || /^0+$/u.test(value[name])) fail("RECEIPT_OID", name);
	}
	for (const name of ["source_note_body_sha256", "source_html_sha256", "source_ledger_manifest_sha256", "source_study_stdout_blob", "source_study_evidence_sha256"]) {
		if (!validSHA256(value[name])) fail("RECEIPT_SHA256", name);
	}
	if (!validPath(value.source_html_path) || !validPositiveInteger(value.source_html_bytes, 32 * 1024 * 1024) ||
		value.source_html_authority !== "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS" ||
		!validPath(value.source_ledger_archive) || value.source_ledger_authority !== "LOCAL_SECRET_BEARING_SNAPSHOT_NOT_PORTABLE" ||
		!Array.isArray(value.source_claims) || value.source_claims.length !== source.claims.length) fail("RECEIPT_FIELDS", "paths, authority, or claims");
	const sourceShort = value.source_commit.slice(0, 12);
	if (value.source_html_path !== `.countershape/evidence/u7d-final-${sourceShort}.html` ||
		value.source_ledger_archive !== `.didrun-history/u7d-final-${sourceShort}/.didrun`) fail("RECEIPT_LOCAL_PATH_IDENTITY", sourceShort);
	validateLedgerManifest(value.source_ledger_manifest);
	if (value.source_ledger_file_count !== value.source_ledger_manifest.length ||
		value.source_ledger_event_count !== source.claims.length + 1 || value.source_ledger_claim_count !== source.claims.length ||
		value.source_ledger_seal_count !== 1 || ledgerManifestDigest(value.source_ledger_manifest) !== value.source_ledger_manifest_sha256) {
		fail("RECEIPT_LEDGER_AUTHORITY", "file/event/claim/seal counts or manifest digest");
	}
	const studyIndex = sourceStudyEventIndex(specification);
	const driverDigests = admittedDriverDigests ?? Object.fromEntries(specification.receipt_contract.study_domains.map((domain) => {
		const path = specification.receipt_contract.study_harness.driver_by_domain[domain];
		return [domain, sha256Hex(gitBytes(["show", `${value.source_commit}:${path}`]))];
	}));
	const studyEvidence = validateStudyEvidence(value.source_study_evidence, specification, driverDigests);
	const studyBytes = Buffer.from(`${JSON.stringify(studyEvidence)}\n`, "utf8");
	if (value.source_study_event_index !== studyIndex || !validPositiveInteger(value.source_study_stdout_bytes, 1024 * 1024) ||
		value.source_study_stdout_bytes !== studyBytes.length || value.source_study_stdout_blob !== value.source_study_evidence_sha256 ||
		value.source_study_evidence_sha256 !== sha256Hex(studyBytes)) fail("RECEIPT_STUDY_AUTHORITY", String(value.source_study_event_index));
	const studyManifest = value.source_ledger_manifest.find((entry) => entry.path === `objects/${value.source_study_stdout_blob}`);
	if (studyManifest?.bytes !== studyBytes.length || studyManifest?.sha256 !== value.source_study_stdout_blob) fail("RECEIPT_STUDY_MANIFEST", value.source_study_stdout_blob);
	for (const [index, claim] of value.source_claims.entries()) {
		const expected = source.claims[index];
		if (!exactKeys(claim, ["index", "supporting_event_index", "type", "label", "pathspecs", "grade"]) || claim.index !== index ||
			claim.supporting_event_index !== index || claim.type !== expected.type || claim.label !== expected.label || claim.grade !== "tree-exact") fail("RECEIPT_CLAIM", String(index));
		if (!isDeepStrictEqual(claim.pathspecs, source.allowed_paths)) fail("RECEIPT_CLAIM_PATHS", String(index));
	}
	return value;
}

async function loadStagedReceipt(specification) {
	let parsed;
	try { parsed = JSON.parse(decodeUTF8(gitBytes(["show", `:${receiptPath}`]), receiptPath)); }
	catch (error) { if (String(error.message).startsWith("U7_PLAN_")) throw error; fail("RECEIPT_JSON", error.message); }
	return validateReceiptDeclaration(parsed, specification);
}

export function receiptProjection(receipt) {
	const rows = receipt.source_claims.map((claim) => `| ${claim.index} | ${claim.supporting_event_index} | \`${claim.type}\` | ${claim.label} | ${claim.pathspecs.map((path) => `\`${path}\``).join("<br>")} | \`${claim.grade}\` |`).join("\n");
	const studyRows = receipt.source_study_evidence.studies.map((study) => `| \`${study.id}\` | \`${study.fixture_commit}\` | ${study.run_count} | ${study.trial_count}/${study.budget_trial_count} | ${study.observed_subject_process_wall_time_ms}/${study.budget_subject_process_wall_time_ms} | ${study.observed_subject_process_peak_rss_bytes}/${study.budget_subject_process_peak_rss_bytes} |`).join("\n");
	return `${receiptBlockStart}\n### Sealed U7D source receipt\n\n` +
		`- **Source:** \`${receipt.source_commit}\` / \`${receipt.source_tree}\`; parent \`${receipt.source_parent}\`; subject \`${receipt.source_subject}\`.\n` +
		`- **didrun note:** \`${receipt.source_note_blob}\`; note-body SHA-256 \`${receipt.source_note_body_sha256}\`; strict exit \`${receipt.source_strict_exit}\`; secrets override \`${receipt.source_secrets_override}\`.\n` +
		`- **Local HTML:** \`${receipt.source_html_path}\`; bytes \`${receipt.source_html_bytes}\`; SHA-256 \`${receipt.source_html_sha256}\`; authority \`${receipt.source_html_authority}\`.\n` +
		`- **Local ledger:** \`${receipt.source_ledger_archive}\`; files/events/claims/seals \`${receipt.source_ledger_file_count}/${receipt.source_ledger_event_count}/${receipt.source_ledger_claim_count}/${receipt.source_ledger_seal_count}\`; manifest SHA-256 \`${receipt.source_ledger_manifest_sha256}\`; authority \`${receipt.source_ledger_authority}\`.\n` +
		`- **Study stdout:** source event \`${receipt.source_study_event_index}\`; object \`${receipt.source_study_stdout_blob}\`; bytes \`${receipt.source_study_stdout_bytes}\`; SHA-256 \`${receipt.source_study_evidence_sha256}\`.\n` +
		`- **Semantic ceiling:** \`${receipt.source_study_evidence.harness_authority.semantic_ceiling}\`; product semantics are not independently recomputed.\n` +
		`- **Resource ceiling:** only the three subject processes per run are measured; full-harness wall time and RSS are \`FULL_STUDY_RESOURCE_BOUND_UNVALIDATED\`. U7R cannot receipt itself; the exact \`unreceipted\` array below is authoritative.\n\n` +
		`| Domain | Fixture commit | Runs | Trials / budget | Observed subject-process wall ms / budget | Observed subject-process peak RSS / budget |\n| --- | --- | ---: | ---: | ---: | ---: |\n${studyRows}\n\n` +
		`\`\`\`json\n${JSON.stringify(receipt.source_study_evidence, null, 2)}\n\`\`\`\n\n` +
		`| Claim index | Supporting event index | Claim type | Exact label | Exact pathspecs | Verbatim grade |\n| ---: | ---: | --- | --- | --- | --- |\n${rows}\n${receiptBlockEnd}`;
}

function validateReceiptJurisdiction(path, text, block, startMarker = receiptBlockStart, endMarker = receiptBlockEnd) {
	if (occurrenceCount(text, startMarker) !== 1 || occurrenceCount(text, endMarker) !== 1 || occurrenceCount(text, block) !== 1) fail("RECEIPT_PROJECTION", path);
	requireVisibleMarkers(text, startMarker, endMarker, `U7D_RECEIPT_${path === "docs/HANDOFF_MODE_C.md" ? "HANDOFF" : "STATUS"}`);
	const start = text.indexOf(block);
	const end = start + block.length;
	const heading = path === "docs/HANDOFF_MODE_C.md" ? "## Current state\n" : "## Source receipt\n";
	const bounds = sectionBounds(text, heading);
	if (start < bounds.start || end > bounds.end || text.slice(end, bounds.end).trim() !== "") fail("RECEIPT_JURISDICTION", path);
	if (path === "docs/status/U7D-EVIDENCE.md" && (bounds.end !== text.length || text.slice(bounds.start, start).trim() !== "")) fail("RECEIPT_JURISDICTION", path);
}

function validateReceiptTransform(path, parentText, candidateText, expected) {
	validateReceiptJurisdiction(path, parentText, pendingReceiptBlock, pendingReceiptBlockStart, pendingReceiptBlockEnd);
	validateReceiptJurisdiction(path, candidateText, expected);
	const transformed = parentText.replace(pendingReceiptBlock, expected);
	if (candidateText !== transformed) fail("RECEIPT_PARENT_TRANSFORM", path);
}

async function requireExactReceiptTransform(path, expected) {
	const parent = decodeUTF8(gitBytes(["show", `HEAD:${path}`]), `${path} parent`);
	const candidate = decodeUTF8(gitBytes(["show", `:${path}`]), `${path} staged`);
	validateReceiptTransform(path, parent, candidate, expected);
}

function exactInvocation(expected) {
	const observed = [process.argv0, relative(repositoryRoot, resolve(process.argv[1])), ...process.argv.slice(2)];
	if (!isDeepStrictEqual(observed, expected)) fail("INVOCATION", JSON.stringify(observed));
}

async function validateReceiptSource(specification, receipt, { runStrict }) {
	const source = unitByID(specification, "U7D");
	const commit = gitLine(["rev-parse", "--verify", "HEAD^{commit}"], "receipt source commit");
	const tree = gitLine(["rev-parse", "--verify", "HEAD^{tree}"], "receipt source tree");
	const parent = gitLine(["show", "-s", "--format=%P", "HEAD"], "receipt source parent");
	const subject = gitLine(["show", "-s", "--format=%s", "HEAD"], "receipt source subject");
	const noteBlob = gitLine(["notes", "--ref=didrun", "list", commit], "receipt source note blob");
	const noteBytes = gitBytes(["cat-file", "blob", noteBlob]);
	if (receipt.source_commit !== commit || receipt.source_tree !== tree || receipt.source_parent !== parent ||
		receipt.source_subject !== subject || receipt.source_note_blob !== noteBlob ||
		receipt.source_note_body_sha256 !== createHash("sha256").update(noteBytes).digest("hex")) fail("RECEIPT_SOURCE_IDENTITY", commit);
	const note = parseNote(noteBytes, commit, tree, source.claims, hermeticPrefix(source), source.allowed_paths);
	if (note.secrets_override !== receipt.source_secrets_override || receipt.source_claims.some((claim, index) =>
		claim.grade !== note.claims[index].grade || claim.supporting_event_index !== note.claims[index].supporting_event_index)) fail("RECEIPT_SOURCE_NOTE", commit);
	if (runStrict) await validateArchivedLedger(source, commit, tree, note, receipt.source_ledger_archive, {
		file_count: receipt.source_ledger_file_count,
		bytes: receipt.source_ledger_manifest.reduce((sum, entry) => sum + entry.bytes, 0),
		manifest_sha256: receipt.source_ledger_manifest_sha256,
		event_count: receipt.source_ledger_event_count,
		claim_count: receipt.source_ledger_claim_count,
		seal_count: receipt.source_ledger_seal_count,
	});
	return Object.freeze({ commit, tree, noteBlob });
}

export async function verifySourceReceipt() {
	const specification = await loadSpecification();
	const unit = unitByID(specification, "U7R");
	exactInvocation(unit.claims[2].command);
	await checkCandidate("U7R");
	const receipt = await loadStagedReceipt(specification);
	const source = await validateReceiptSource(specification, receipt, { runStrict: true });
	const projection = receiptProjection(receipt);
	for (const path of specification.receipt_contract.projection_paths) await requireExactReceiptTransform(path, projection);
	console.log(`U7R source receipt reconciliation exact: source=${source.commit} tree=${source.tree} note=${source.noteBlob} claims=${receipt.source_claims.length} strict=0 projections=2 self_receipt=absent`);
}

function requireNoSymlinkComponents(relativePath) {
	if (!validPath(relativePath)) fail("LOCAL_PATH", relativePath);
	const parts = relativePath.split("/");
	return Promise.all(parts.map(async (_, index) => {
		const path = resolve(repositoryRoot, ...parts.slice(0, index + 1));
		const stat = await lstat(path);
		if (stat.isSymbolicLink()) fail("LOCAL_SYMLINK", relativePath);
		if (index < parts.length - 1 && !stat.isDirectory()) fail("LOCAL_DIRECTORY", path);
	}));
}

async function collectLedgerManifest(relativeRoot) {
	await requireNoSymlinkComponents(relativeRoot);
	const root = resolve(repositoryRoot, relativeRoot);
	const before = await lstat(root);
	if (!before.isDirectory() || before.isSymbolicLink()) fail("LOCAL_LEDGER_ROOT", relativeRoot);
	const rootEntries = await readdir(root, { withFileTypes: true });
	const rootNames = rootEntries.map((entry) => entry.name).sort();
	if (!isDeepStrictEqual(rootNames, [...requiredLedgerFiles, "objects"].sort())) fail("LOCAL_LEDGER_LAYOUT", rootNames.join(","));
	for (const entry of rootEntries) {
		if (entry.name === "objects" ? !entry.isDirectory() : !entry.isFile()) fail("LOCAL_LEDGER_TYPE", entry.name);
		if (entry.isSymbolicLink()) fail("LOCAL_LEDGER_SYMLINK", entry.name);
	}
	const objectsRoot = resolve(root, "objects");
	const objectsBefore = await lstat(objectsRoot);
	if (!objectsBefore.isDirectory() || objectsBefore.isSymbolicLink()) fail("LOCAL_LEDGER_OBJECT_ROOT", relativeRoot);
	const objectEntries = await readdir(objectsRoot, { withFileTypes: true });
	if (objectEntries.length === 0 || objectEntries.length > 9_996) fail("LOCAL_LEDGER_OBJECT_COUNT", String(objectEntries.length));
	const paths = [...requiredLedgerFiles, ...objectEntries.map((entry) => {
		if (!entry.isFile() || entry.isSymbolicLink() || !/^[0-9a-f]{64}$/u.test(entry.name)) fail("LOCAL_LEDGER_OBJECT_NAME", entry.name);
		return `objects/${entry.name}`;
	})].sort();
	const manifest = [];
	const bytesByPath = new Map();
	for (const path of paths) {
		const bytes = await readRegular(resolve(root, path), 32 * 1024 * 1024, path.startsWith("objects/"));
		manifest.push(Object.freeze({ path, bytes: bytes.length, sha256: sha256Hex(bytes) }));
		bytesByPath.set(path, bytes);
	}
	const referencedObjects = new Set(manifest.filter((entry) => entry.path.startsWith("objects/")).map((entry) => entry.path.slice("objects/".length)));
	for (const digest of referencedObjects) if (manifest.find((entry) => entry.path === `objects/${digest}`)?.sha256 !== digest) fail("LOCAL_LEDGER_OBJECT_DIGEST", digest);
	const after = await lstat(root);
	const objectsAfter = await lstat(objectsRoot);
	if (before.dev !== after.dev || before.ino !== after.ino || before.mtimeMs !== after.mtimeMs ||
		objectsBefore.dev !== objectsAfter.dev || objectsBefore.ino !== objectsAfter.ino || objectsBefore.mtimeMs !== objectsAfter.mtimeMs ||
		!isDeepStrictEqual(rootNames, (await readdir(root)).sort()) ||
		!isDeepStrictEqual(objectEntries.map((entry) => entry.name).sort(), (await readdir(objectsRoot)).sort())) fail("LOCAL_LEDGER_CHANGED", relativeRoot);
	return Object.freeze({ manifest: Object.freeze(manifest), referencedObjects, bytesByPath });
}

function previewMatchesRaw(preview, raw) {
	if (!Array.isArray(preview) || !Array.isArray(raw) || preview.length !== raw.length) return false;
	return preview.every((shown, index) => shown === raw[index] || expectedRedactions(index, raw[index]).includes(shown));
}

async function validateArchivedLedger(row, commit, tree, note, archivePath, expectedArchive = undefined) {
	const collected = await collectLedgerManifest(archivePath);
	const manifestBytes = collected.manifest.reduce((sum, entry) => sum + entry.bytes, 0);
	if (expectedArchive !== undefined && (collected.manifest.length !== expectedArchive.file_count || manifestBytes !== expectedArchive.bytes ||
		ledgerManifestDigest(collected.manifest) !== expectedArchive.manifest_sha256)) fail("PARENT_ARCHIVE_MANIFEST", archivePath);
	const session = parseJSONLines(collected.bytesByPath.get("session.log"), `${archivePath}/session.log`);
	const claims = parseJSONLines(collected.bytesByPath.get("claims.jsonl"), `${archivePath}/claims.jsonl`);
	const seals = parseJSONLines(collected.bytesByPath.get("seals.jsonl"), `${archivePath}/seals.jsonl`);
	const claimCount = row?.claims.length ?? note.claims.length;
	if (session.length !== claimCount + 1 || claims.length !== claimCount || seals.length !== 1 ||
		expectedArchive !== undefined && (session.length !== expectedArchive.event_count || claims.length !== expectedArchive.claim_count || seals.length !== expectedArchive.seal_count)) fail("PARENT_ARCHIVE_CARDINALITY", archivePath);
	let environmentFingerprint;
	const referenced = new Set();
	for (let index = 0; index < claimCount; index += 1) {
		const entry = session[index]; const event = entry?.event; const claim = claims[index]; const noteRecord = note.claims[index];
		const previous = index === 0 ? "0".repeat(64) : session[index - 1]?.entry_hash;
		if (!exactKeys(entry, ["entry_hash", "event", "index", "prev_hash"]) || entry.index !== index || entry.prev_hash !== previous ||
			computeDidrunEntryHash(entry) !== entry.entry_hash || !exactKeys(event, ["argv", "coverage", "cwd", "ended_at", "env_fingerprint", "exit_code", "observed_via", "started_at", "stderr_blob", "stdout_blob", "submodule_dirty", "transcript_blob", "tree_after", "tree_before"]) ||
			event.exit_code !== 0 || event.observed_via !== "wrapper" || event.coverage !== "complete" || event.submodule_dirty !== false || event.cwd !== repositoryRoot ||
			event.tree_before !== tree || event.tree_after !== tree || !Number.isFinite(event.started_at) || !Number.isFinite(event.ended_at) || event.ended_at < event.started_at ||
			!/^[0-9a-f]{16}$/u.test(event.env_fingerprint ?? "")) fail("PARENT_ARCHIVE_EVENT", `${archivePath}:${index}`);
		environmentFingerprint ??= event.env_fingerprint;
		if (event.env_fingerprint !== environmentFingerprint) fail("PARENT_ARCHIVE_ENVIRONMENT", `${archivePath}:${index}`);
		const { argv_preview: preview, ...noteClaim } = noteRecord.claim;
		if (!isDeepStrictEqual(noteClaim, claim) || noteRecord.supporting_event_index !== index ||
			row !== undefined && (!isDeepStrictEqual(event.argv, [...hermeticPrefix(row), ...row.claims[index].command]) || !isDeepStrictEqual(claim.pathspecs, row.allowed_paths)) ||
			row === undefined && !previewMatchesRaw(preview, event.argv)) fail("PARENT_ARCHIVE_CLAIM", `${archivePath}:${index}`);
		for (const field of ["stdout_blob", "stderr_blob", "transcript_blob"]) {
			const digest = event[field];
			if (digest !== null && !validSHA256(digest)) fail("PARENT_ARCHIVE_BLOB", `${archivePath}:${index}:${field}`);
			if (digest !== null) referenced.add(digest);
		}
	}
	const inspectionEntry = session.at(-1); const inspection = inspectionEntry?.event;
	const inspectionTail = ["/usr/bin/git", "notes", "--ref=didrun", "show", commit];
	const inspectionArgvMatches = row === undefined ? isDeepStrictEqual(inspection?.argv?.slice(-inspectionTail.length), inspectionTail) :
		isDeepStrictEqual(inspection?.argv, [...hermeticPrefix(row), ...inspectionTail]);
	if (!exactKeys(inspectionEntry, ["entry_hash", "event", "index", "prev_hash"]) || inspectionEntry.index !== claimCount ||
		inspectionEntry.prev_hash !== session.at(-2)?.entry_hash || computeDidrunEntryHash(inspectionEntry) !== inspectionEntry.entry_hash ||
		!exactKeys(inspection, ["argv", "coverage", "cwd", "ended_at", "env_fingerprint", "exit_code", "observed_via", "started_at", "stderr_blob", "stdout_blob", "submodule_dirty", "transcript_blob", "tree_after", "tree_before"]) ||
		inspection.exit_code !== 0 || inspection.observed_via !== "wrapper" || inspection.coverage !== "complete" || inspection.submodule_dirty !== false || inspection.cwd !== repositoryRoot ||
		inspection.env_fingerprint !== environmentFingerprint || inspection.tree_before !== tree || inspection.tree_after !== tree ||
		!inspectionArgvMatches) fail("PARENT_ARCHIVE_INSPECTION", archivePath);
	if (row !== undefined) validateRecordedChildIntervals(session, `${row.id}:archive`);
	for (const field of ["stdout_blob", "stderr_blob", "transcript_blob"]) { const digest = inspection[field]; if (digest !== null && !validSHA256(digest)) fail("PARENT_ARCHIVE_BLOB", `${archivePath}:inspection:${field}`); if (digest !== null) referenced.add(digest); }
	if (!isDeepStrictEqual([...referenced].sort(), [...collected.referencedObjects].sort()) ||
		!exactKeys(seals[0], ["claims_watermark", "commit", "tree"]) || seals[0].claims_watermark !== claimCount || seals[0].commit !== commit || seals[0].tree !== tree) fail("PARENT_ARCHIVE_CLOSURE", archivePath);
	const terminal = await collectLedgerManifest(archivePath);
	if (!isDeepStrictEqual(terminal.manifest, collected.manifest)) fail("PARENT_ARCHIVE_DRIFT", archivePath);
	return Object.freeze({ files: collected.manifest.length, bytes: manifestBytes, events: session.length, claims: claims.length });
}

export async function verifyLocalEvidence() {
	const specification = await loadSpecification();
	const unit = unitByID(specification, "U7R");
	exactInvocation(unit.claims[4].command);
	await checkCandidate("U7R");
	const receipt = await loadStagedReceipt(specification);
	await validateReceiptSource(specification, receipt, { runStrict: false });
	await requireNoSymlinkComponents(receipt.source_html_path);
	const html = await readRegular(resolve(repositoryRoot, receipt.source_html_path), 32 * 1024 * 1024);
	if (html.length !== receipt.source_html_bytes || sha256Hex(html) !== receipt.source_html_sha256) fail("LOCAL_HTML", receipt.source_html_path);
	const collected = await collectLedgerManifest(receipt.source_ledger_archive);
	if (!isDeepStrictEqual(collected.manifest, receipt.source_ledger_manifest) ||
		ledgerManifestDigest(collected.manifest) !== receipt.source_ledger_manifest_sha256 ||
		collected.manifest.length !== receipt.source_ledger_file_count) fail("LOCAL_LEDGER_MANIFEST", receipt.source_ledger_archive);
	const archiveRoot = resolve(repositoryRoot, receipt.source_ledger_archive);
	const session = parseJSONLines(collected.bytesByPath.get("session.log"), "source session.log");
	const claims = parseJSONLines(collected.bytesByPath.get("claims.jsonl"), "source claims.jsonl");
	const seals = parseJSONLines(collected.bytesByPath.get("seals.jsonl"), "source seals.jsonl");
	const source = unitByID(specification, "U7D");
	if (session.length !== receipt.source_ledger_event_count || claims.length !== receipt.source_ledger_claim_count ||
		seals.length !== receipt.source_ledger_seal_count) fail("LOCAL_LEDGER_CARDINALITY", `${session.length}:${claims.length}:${seals.length}`);
	const prefix = hermeticPrefix(source);
	let environmentFingerprint;
	const referencedBlobs = new Set();
	for (const [index, expected] of source.claims.entries()) {
		const entry = session[index];
		const event = session[index]?.event;
		const claim = claims[index];
		const previous = index === 0 ? "0".repeat(64) : session[index - 1]?.entry_hash;
		if (!exactKeys(entry, ["entry_hash", "event", "index", "prev_hash"]) || entry.index !== index || entry.prev_hash !== previous ||
			!/^[0-9a-f]{64}$/u.test(entry.entry_hash ?? "") || computeDidrunEntryHash(entry) !== entry.entry_hash || !exactKeys(event, ["argv", "coverage", "cwd", "ended_at", "env_fingerprint", "exit_code", "observed_via", "started_at", "stderr_blob", "stdout_blob", "submodule_dirty", "transcript_blob", "tree_after", "tree_before"]) ||
			event.exit_code !== 0 || event.observed_via !== "wrapper" || event.coverage !== "complete" || event.submodule_dirty !== false ||
			event.cwd !== repositoryRoot || !Number.isFinite(event.started_at) || !Number.isFinite(event.ended_at) || event.ended_at < event.started_at ||
			event.tree_before !== receipt.source_tree || event.tree_after !== receipt.source_tree ||
			!/^[0-9a-f]{16}$/u.test(event.env_fingerprint ?? "") || !isDeepStrictEqual(event.argv, [...prefix, ...expected.command]) ||
			!exactKeys(claim, ["ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) || claim.label !== expected.label || claim.ctype !== expected.type ||
			claim.declared_at_index !== index || !isDeepStrictEqual(claim.event_indices, [index]) || !isDeepStrictEqual(claim.pathspecs, source.allowed_paths)) fail("LOCAL_LEDGER_EVENT", String(index));
		environmentFingerprint ??= event.env_fingerprint;
		if (event.env_fingerprint !== environmentFingerprint) fail("LOCAL_LEDGER_ENVIRONMENT", String(index));
		for (const field of ["stdout_blob", "stderr_blob", "transcript_blob"]) {
			const digest = event[field];
			if (digest !== null && (typeof digest !== "string" || !/^[0-9a-f]{64}$/u.test(digest))) fail("LOCAL_LEDGER_BLOB", `${index}:${field}`);
			if (digest !== null) referencedBlobs.add(digest);
		}
	}
	const studyEvent = session[receipt.source_study_event_index]?.event;
	const studyObject = collected.bytesByPath.get(`objects/${receipt.source_study_stdout_blob}`);
	const expectedStudyBytes = Buffer.from(`${JSON.stringify(receipt.source_study_evidence)}\n`, "utf8");
	if (studyEvent?.stdout_blob !== receipt.source_study_stdout_blob || studyObject === undefined ||
		studyObject.length !== receipt.source_study_stdout_bytes || sha256Hex(studyObject) !== receipt.source_study_evidence_sha256 ||
		!studyObject.equals(expectedStudyBytes)) fail("LOCAL_STUDY_EVIDENCE", String(receipt.source_study_event_index));
	const inspectionEntry = session.at(-1);
	const inspection = inspectionEntry?.event;
	const inspectionTail = [gitPath, "notes", "--ref=didrun", "show", receipt.source_commit];
	if (!exactKeys(inspectionEntry, ["entry_hash", "event", "index", "prev_hash"]) || inspectionEntry.index !== source.claims.length ||
		inspectionEntry.prev_hash !== session.at(-2)?.entry_hash || !/^[0-9a-f]{64}$/u.test(inspectionEntry.entry_hash ?? "") ||
		computeDidrunEntryHash(inspectionEntry) !== inspectionEntry.entry_hash ||
		!exactKeys(inspection, ["argv", "coverage", "cwd", "ended_at", "env_fingerprint", "exit_code", "observed_via", "started_at", "stderr_blob", "stdout_blob", "submodule_dirty", "transcript_blob", "tree_after", "tree_before"]) ||
		inspection.exit_code !== 0 || inspection.observed_via !== "wrapper" || inspection.coverage !== "complete" || inspection.submodule_dirty !== false ||
		inspection.cwd !== repositoryRoot || inspection.env_fingerprint !== environmentFingerprint ||
		inspection.tree_before !== receipt.source_tree || inspection.tree_after !== receipt.source_tree ||
		!isDeepStrictEqual(inspection.argv, [...prefix, ...inspectionTail])) fail("LOCAL_NOTE_INSPECTION", receipt.source_commit);
	validateRecordedChildIntervals(session, "U7D:local-evidence");
	for (const field of ["stdout_blob", "stderr_blob", "transcript_blob"]) {
		const digest = inspection[field];
		if (digest !== null && (typeof digest !== "string" || !/^[0-9a-f]{64}$/u.test(digest))) fail("LOCAL_NOTE_BLOB", field);
		if (digest !== null) referencedBlobs.add(digest);
	}
	if (!isDeepStrictEqual([...referencedBlobs].sort(), [...collected.referencedObjects].sort())) fail("LOCAL_LEDGER_OBJECT_COVERAGE", receipt.source_ledger_archive);
	if (!exactKeys(seals[0], ["claims_watermark", "commit", "tree"]) || seals[0].claims_watermark !== source.claims.length ||
		seals[0].commit !== receipt.source_commit || seals[0].tree !== receipt.source_tree) fail("LOCAL_SEAL", receipt.source_commit);
	const terminal = await collectLedgerManifest(receipt.source_ledger_archive);
	if (!isDeepStrictEqual(terminal.manifest, collected.manifest)) fail("LOCAL_LEDGER_TERMINAL_DRIFT", receipt.source_ledger_archive);
	console.log(`U7R local source evidence snapshot exact: html_bytes=${html.length} html_sha256=${receipt.source_html_sha256} ledger_files=${collected.manifest.length} ledger_manifest_sha256=${receipt.source_ledger_manifest_sha256} events=${session.length} claims=${claims.length} study_event=${receipt.source_study_event_index} study_stdout=${receipt.source_study_stdout_blob} seals=1 authority=LOCAL_SNAPSHOT_ONLY`);
}

function syntheticStudyDriverDigests(specification) {
	return Object.fromEntries(specification.receipt_contract.study_domains.map((domain) => [domain, sha256Hex(Buffer.from(`synthetic-driver:${domain}`, "utf8"))]));
}

function syntheticStudyTimeReport(rss) {
	return `real 0.01\nuser 0.00\nsys 0.00\n${String(rss).padStart(20)}  maximum resident set size\n${"0".padStart(20)}  average shared memory size\n${"0".padStart(20)}  average unshared data size\n${"0".padStart(20)}  average unshared stack size\n${"1".padStart(20)}  page reclaims\n${"0".padStart(20)}  page faults\n${"0".padStart(20)}  swaps\n${"0".padStart(20)}  block input operations\n${"0".padStart(20)}  block output operations\n${"0".padStart(20)}  messages sent\n${"0".padStart(20)}  messages received\n${"0".padStart(20)}  signals received\n${"1".padStart(20)}  voluntary context switches\n${"0".padStart(20)}  involuntary context switches\n${"1".padStart(20)}  instructions retired\n${"1".padStart(20)}  cycles elapsed\n${String(rss).padStart(20)}  peak memory footprint\n`;
}

function syntheticStudyEvidence(specification) {
	const contract = specification.receipt_contract; const harness = contract.study_harness; const digest = (label) => sha256Hex(Buffer.from(label, "utf8"));
	const driverDigests = syntheticStudyDriverDigests(specification); const tools = Object.fromEntries(specification.runtime_authority.admitted_tools.map((tool) => [tool.name, tool]));
	const studies = contract.study_domains.map((id, domainIndex) => {
		const deterministic_artifacts = Object.fromEntries(contract.deterministic_artifact_fields.map((field) => [field, digest(`plan:deterministic:${id}:${field}`)]));
		const fixture_commit = createHash("sha1").update(`plan:fixture:${id}`).digest("hex"); const fixtureTree = createHash("sha1").update(`plan:tree:${id}`).digest("hex");
		const runs = [1, 2, 3].map((ordinal) => {
			const runRoot = resolve(repositoryRoot, ".countershape/u7d-final/tmp/studies", id, `run-${ordinal}`); const executionCwd = resolve(runRoot, "fixture"); const executionBinary = resolve(executionCwd, "countershape");
			const execution = { cwd: executionCwd, argv: [executionBinary, "study", id, "--json"], executable_path: executionBinary, executable_sha256: deterministic_artifacts.reference_binary_sha256 };
			const fresh = Object.fromEntries(contract.fresh_run_digest_fields.map((field) => [field, digest(`plan:fresh:${id}:${ordinal}:${field}`)])); fresh.fixture_invocation_sha256 = sha256Hex(Buffer.from(`${JSON.stringify(execution)}\n`, "utf8"));
			const manifest = studyManifestPaths(contract).map((path) => {
				const deterministicField = Object.entries(harness.deterministic_artifact_paths).find(([, candidate]) => candidate === path)?.[0];
				const freshField = Object.entries(harness.fresh_artifact_paths).find(([, candidate]) => candidate === path)?.[0];
				return { path, mode: "0600", bytes: 64, sha256: deterministicField === undefined ? freshField === undefined ? digest(`plan:trial:${id}:${ordinal}:${path}`) : fresh[freshField] : deterministic_artifacts[deterministicField] };
			});
			const product_result = { schema_version: harness.product_result_schema, domain: id, ordinal, status: "GREEN" };
			const run = { ordinal, trial_count: 100, trial_budget: 100, observed_subject_process_wall_time_ms: 30, budget_subject_process_wall_time_ms: contract.study_subject_process_wall_time_budget_ms_per_domain / 3,
				observed_subject_process_peak_rss_bytes: 1_000_000 + domainIndex * 10_000 + ordinal * 100 + 3, budget_subject_process_peak_rss_bytes: contract.study_subject_process_peak_rss_budget_bytes_per_domain, observation_authority: harness.observation_authority,
				fixture: { head_commit: fixture_commit, tree: fixtureTree, commit_count: 1, git_directory: relative(repositoryRoot, resolve(executionCwd, ".git")), git_common_directory: relative(repositoryRoot, resolve(executionCwd, ".git")), git_alternates: "ABSENT", clean_status_bytes: 0, clean_status_sha256: sha256Hex(Buffer.alloc(0)) },
				driver_authority: { path: harness.driver_by_domain[id], sha256: driverDigests[id] }, execution, processes: {}, phases: contract.study_phase_budgets.map((phase) => ({ id: phase.id, trial_count: phase.trial_count / 3, trial_budget: phase.trial_budget / 3 })), product_result,
				evidence_root: relative(repositoryRoot, resolve(runRoot, "evidence")), evidence_manifest: manifest, evidence_manifest_sha256: sha256Hex(Buffer.from(`${JSON.stringify(manifest)}\n`, "utf8")), deterministic_artifacts: { ...deterministic_artifacts }, ...fresh };
			for (const [processIndex, name] of ["prepare", "compile", "execute"].entries()) {
				const rss = 1_000_000 + domainIndex * 10_000 + ordinal * 100 + processIndex + 1; const time_report = syntheticStudyTimeReport(rss); const reportPath = resolve(runRoot, "observations", `${name}.time`);
				let cwd; let executable; let executableSha; let args; let stdout;
				if (name === "prepare") { cwd = repositoryRoot; executable = tools.node.path; executableSha = tools.node.sha256; args = [resolve(repositoryRoot, harness.driver_by_domain[id]), "--prepare", "--domain", id, "--ordinal", String(ordinal), "--fixture-root", executionCwd]; stdout = Buffer.alloc(0); }
				else if (name === "compile") { cwd = repositoryRoot; executable = tools.go.path; executableSha = tools.go.sha256; args = ["build", "-trimpath", "-mod=readonly", "-buildvcs=false", "-o", executionBinary, "./cmd/countershape"]; stdout = Buffer.alloc(0); }
				else { cwd = executionCwd; executable = executionBinary; executableSha = execution.executable_sha256; args = ["study", id, "--json"]; stdout = Buffer.from(`${JSON.stringify(product_result)}\n`, "utf8"); }
				const reportBytes = Buffer.from(time_report, "utf8"); run.processes[name] = { name, observer_argv: [tools.time.path, "-p", "-l", "-o", reportPath, executable, ...args], cwd, executable_path: executable, executable_sha256: executableSha, observer_path: tools.time.path, observer_sha256: tools.time.sha256, environment_sha256: studyProcessEnvironmentDigest(specification, { id }, run, name), wall_time_ms: 10, peak_rss_bytes: rss, time_report_path: relative(repositoryRoot, reportPath), time_report_bytes: reportBytes.length, time_report_sha256: sha256Hex(reportBytes), time_report, stdout_bytes: stdout.length, stdout_sha256: sha256Hex(stdout) };
			}
			return run;
		});
		return { id, fixture_commit, fixture_authority: { head_commit: fixture_commit, tree: fixtureTree, clean_status_bytes: 0, clean_status_sha256: sha256Hex(Buffer.alloc(0)), repository_count: 3, fresh_repository_per_run: true, repository_git_directories: runs.map((run) => run.fixture.git_directory) }, logical_product_command: ["countershape", "study", id, "--json"], run_count: 3,
			trial_count: 300, budget_trial_count: 300, observed_subject_process_wall_time_ms: 90, budget_subject_process_wall_time_ms: contract.study_subject_process_wall_time_budget_ms_per_domain, observed_subject_process_peak_rss_bytes: Math.max(...runs.map((run) => run.observed_subject_process_peak_rss_bytes)), budget_subject_process_peak_rss_bytes: contract.study_subject_process_peak_rss_budget_bytes_per_domain, observation_authority: harness.observation_authority, phases: contract.study_phase_budgets.map((phase) => ({ ...phase })), deterministic_artifacts, runs };
	});
	const versions = Object.fromEntries(specification.runtime_authority.admitted_tools.map((tool) => [tool.name, tool.version]));
	return { schema_version: contract.study_evidence_schema, phase: "U7D", harness_authority: { protocol: harness.protocol, path: harness.path, sha256: harness.sha256, protocol_sha256: harness.protocol_sha256, observation_authority: harness.observation_authority, semantic_ceiling: harness.semantic_ceiling }, environment: { platform: "darwin", kernel_release: "25.0.0", architecture: "arm64", git_version: versions.git, go_version: versions.go, node_version: versions.node, cpu_model: "synthetic self-test CPU", logical_cpu_count: 12, memory_bytes: 32 * 1024 ** 3, runtime_authority_sha256: runtimeAuthorityDigest(specification) }, studies, milestone_verdict: contract.milestone_verdict, honest_fallback: contract.honest_fallback, unreceipted: [...contract.unreceipted] };
}

function syntheticReceipt(specification) {
	const source = unitByID(specification, "U7D");
	let manifest = [
		{ path: ".gitignore", bytes: 8, sha256: "6".repeat(64) },
		{ path: "claims.jsonl", bytes: 1024, sha256: "7".repeat(64) },
		{ path: `objects/${"8".repeat(64)}`, bytes: 64, sha256: "8".repeat(64) },
		{ path: "seals.jsonl", bytes: 128, sha256: "9".repeat(64) },
		{ path: "session.log", bytes: 2048, sha256: "a".repeat(64) },
	];
	const studyEvidence = syntheticStudyEvidence(specification);
	const studyBytes = Buffer.from(`${JSON.stringify(studyEvidence)}\n`, "utf8");
	const studyDigest = sha256Hex(studyBytes);
	manifest = [...manifest, { path: `objects/${studyDigest}`, bytes: studyBytes.length, sha256: studyDigest }].sort((left, right) => left.path < right.path ? -1 : left.path > right.path ? 1 : 0);
	return {
		schema_version: "countershape/u7-receipt/v1", source_boundary: "U7D", source_commit: "1".repeat(40),
		source_tree: "2".repeat(40), source_parent: "3".repeat(40), source_subject: source.subject,
		source_note_blob: "4".repeat(40), source_note_body_sha256: "5".repeat(64),
		source_claims: source.claims.map((claim, index) => ({ index, supporting_event_index: index, type: claim.type, label: claim.label, pathspecs: [...source.allowed_paths], grade: "tree-exact" })),
		source_strict_exit: 0, source_secrets_override: false, source_html_path: ".countershape/evidence/u7d-final-111111111111.html", source_html_bytes: 4096,
		source_html_sha256: "6".repeat(64), source_html_authority: "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS",
		source_ledger_archive: ".didrun-history/u7d-final-111111111111/.didrun", source_ledger_authority: "LOCAL_SECRET_BEARING_SNAPSHOT_NOT_PORTABLE",
		source_ledger_file_count: manifest.length, source_ledger_event_count: source.claims.length + 1,
		source_ledger_claim_count: source.claims.length, source_ledger_seal_count: 1,
		source_ledger_manifest: manifest, source_ledger_manifest_sha256: ledgerManifestDigest(manifest),
		source_study_event_index: sourceStudyEventIndex(specification), source_study_stdout_blob: studyDigest,
		source_study_stdout_bytes: studyBytes.length, source_study_evidence_sha256: studyDigest, source_study_evidence: studyEvidence,
	};
}

export async function selfTestSourceReceipt() {
	const specification = await loadSpecification();
	const unit = unitByID(specification, "U7R");
	exactInvocation(unit.claims[3].command);
	const base = syntheticReceipt(specification);
	const driverDigests = syntheticStudyDriverDigests(specification);
	validateReceiptDeclaration(base, specification, driverDigests);
	const mutations = [
		["RECEIPT_SHAPE", (value) => { value.extra = true; }],
		["RECEIPT_OID", (value) => { value.source_commit = "0".repeat(40); }],
		["RECEIPT_SHA256", (value) => { value.source_html_sha256 = "bad"; }],
		["RECEIPT_SHA256", (value) => { value.source_ledger_manifest_sha256 = "bad"; }],
		["RECEIPT_FIELDS", (value) => { value.source_html_authority = "PORTABLE"; }],
		["RECEIPT_LEDGER_AUTHORITY", (value) => { value.source_ledger_event_count -= 1; }],
		["RECEIPT_LEDGER_AUTHORITY", (value) => { value.source_ledger_manifest[0].bytes += 1; }],
		["RECEIPT_LEDGER_MANIFEST", (value) => { value.source_ledger_manifest.push(value.source_ledger_manifest[0]); }],
		["RECEIPT_LEDGER_OBJECT", (value) => { value.source_ledger_manifest[2].sha256 = "b".repeat(64); }],
		["RECEIPT_FIELDS", (value) => { value.source_claims.pop(); }],
		["RECEIPT_CLAIM", (value) => { value.source_claims[0].grade = "stale"; }],
		["RECEIPT_CLAIM", (value) => { value.source_claims[1].label += " drift"; }],
		["RECEIPT_CLAIM_PATHS", (value) => { value.source_claims[0].pathspecs.pop(); }],
		["STUDY_EVIDENCE_SHAPE", (value) => { value.source_study_evidence.unreceipted.pop(); }],
		["STUDY_ENVIRONMENT", (value) => { value.source_study_evidence.environment.runtime_authority_sha256 = "f".repeat(64); }],
		["STUDY_DOMAIN", (value) => { value.source_study_evidence.studies[0].phases[0].trial_budget += 1; }],
		["STUDY_RUN", (value) => { value.source_study_evidence.studies[0].runs[0].phases[0].trial_budget += 1; }],
		["STUDY_RUN", (value) => { value.source_study_evidence.studies[1].runs[2].phases[4].trial_count = 0; }],
		["STUDY_RUN", (value) => { value.source_study_evidence.studies[0].runs[1].deterministic_artifacts.source_spec_sha256 = "e".repeat(64); }],
		["STUDY_EXECUTION", (value) => { value.source_study_evidence.studies[0].runs[1].execution.argv[3] = "--text"; }],
		["STUDY_PROCESS", (value) => { value.source_study_evidence.studies[0].runs[1].processes.prepare.executable_sha256 = "e".repeat(64); }],
		["STUDY_MANIFEST_DIGEST", (value) => { value.source_study_evidence.studies[0].runs[1].evidence_manifest_sha256 = "e".repeat(64); }],
		["STUDY_DRIVER", (value) => { value.source_study_evidence.studies[0].runs[1].driver_authority.sha256 = "e".repeat(64); }],
		["STUDY_FRESH_DIGEST", (value) => { const target = value.source_study_evidence.studies[1].runs[2]; target.world_instance_sha256 = value.source_study_evidence.studies[0].runs[0].world_instance_sha256; target.evidence_manifest.find((row) => row.path === "fresh/world-instance.json").sha256 = target.world_instance_sha256; target.evidence_manifest_sha256 = sha256Hex(Buffer.from(`${JSON.stringify(target.evidence_manifest)}\n`, "utf8")); }],
		["RECEIPT_STUDY_AUTHORITY", (value) => { value.source_study_event_index = 7; }],
		["RECEIPT_STUDY_MANIFEST", (value) => { value.source_ledger_manifest = value.source_ledger_manifest.filter((entry) => entry.path !== `objects/${value.source_study_stdout_blob}`); value.source_ledger_file_count -= 1; value.source_ledger_manifest_sha256 = ledgerManifestDigest(value.source_ledger_manifest); }],
	];
	for (const [token, mutate] of mutations) {
		const candidate = clone(base);
		mutate(candidate);
		try { validateReceiptDeclaration(candidate, specification, driverDigests); }
		catch (error) {
			if (String(error.message).includes(token)) continue;
			fail("SELFTEST_RECEIPT_WRONG_REJECTION", `${token}:${error.message}`);
		}
		fail("SELFTEST_RECEIPT_FALSE_NEGATIVE", token);
	}
	const projection = receiptProjection(base);
	if (projection.split(receiptBlockStart).length !== 2 || projection.split(receiptBlockEnd).length !== 2 ||
		projection.split("| `tree-exact` |").length - 1 !== base.source_claims.length || !projection.includes("U7R cannot receipt itself") ||
		!projection.includes(base.source_study_stdout_blob) || !projection.includes("ADOPTION_UNVALIDATED")) fail("SELFTEST_RECEIPT_PROJECTION", "canonical block");
	const handoffParent = `# Handoff\n\n## Current state\n\n- Active source: U7D.\n\n${pendingReceiptBlock}\n\n## History\n\nFrozen.\n`;
	const statusParent = `# U7D evidence\n\n## Source receipt\n\n${pendingReceiptBlock}\n`;
	validateReceiptTransform("docs/HANDOFF_MODE_C.md", handoffParent, handoffParent.replace(pendingReceiptBlock, projection), projection);
	validateReceiptTransform("docs/status/U7D-EVIDENCE.md", statusParent, statusParent.replace(pendingReceiptBlock, projection), projection);
	const transformHostiles = [
		["MARKDOWN_VISIBILITY", "docs/HANDOFF_MODE_C.md", handoffParent.replace(pendingReceiptBlock, `\`\`\`md\n${pendingReceiptBlock}\n\`\`\``), handoffParent.replace(pendingReceiptBlock, projection)],
		["MARKDOWN_VISIBILITY", "docs/HANDOFF_MODE_C.md", handoffParent.replace(pendingReceiptBlock, `<!--\n${pendingReceiptBlock}\n-->`), handoffParent.replace(pendingReceiptBlock, projection)],
		["RECEIPT_JURISDICTION", "docs/HANDOFF_MODE_C.md", handoffParent.replace(pendingReceiptBlock, "").replace("Frozen.\n", `Frozen.\n${pendingReceiptBlock}\n`), handoffParent.replace(pendingReceiptBlock, projection)],
		["RECEIPT_PARENT_TRANSFORM", "docs/HANDOFF_MODE_C.md", handoffParent, handoffParent.replace(pendingReceiptBlock, projection).replace("Frozen.", "Changed." )],
		["MARKDOWN_HEADING", "docs/status/U7D-EVIDENCE.md", statusParent, statusParent.replace(pendingReceiptBlock, projection) + "\n## Source receipt\n"],
	];
	for (const [token, path, parent, candidate] of transformHostiles) {
		try { validateReceiptTransform(path, parent, candidate, projection); }
		catch (error) { if (String(error.message).includes(token)) continue; fail("SELFTEST_RECEIPT_TRANSFORM_WRONG_REJECTION", `${token}:${error.message}`); }
		fail("SELFTEST_RECEIPT_TRANSFORM_FALSE_NEGATIVE", token);
	}
	console.log(`U7 receipt checker defensive self-test passed: ${mutations.length} declaration mutations, ${transformHostiles.length} projection-jurisdiction refusals, exact ${base.source_claims.length}-claim study-bound projection, local-authority ceilings, and self-receipt refusal`);
}

function clone(value) {
	return JSON.parse(JSON.stringify(value));
}

function expectRejected(candidate, token) {
	try { validateSpecification(candidate); } catch (error) {
		if (String(error.message).includes(token)) return;
		fail("SELFTEST_WRONG_REJECTION", `${token}:${error.message}`);
	}
	fail("SELFTEST_FALSE_NEGATIVE", token);
}

async function expectRunbookRecoveryError(promise, token) {
	try { await promise; }
	catch (error) {
		if (String(error.message).includes(token)) return;
		fail("SELFTEST_RECOVERY_WRONG_REJECTION", `${token}:${error.message}`);
	}
	fail("SELFTEST_RECOVERY_FALSE_NEGATIVE", token);
}

async function selfTestFailedAttemptRecovery(specification) {
	const unit = specification.units[0];
	const commit = "a".repeat(40);
	const slug = "20260808T120000Z-defensive-fixture";
	let exercised = 0;
	const makeDirectory = async (path) => { await mkdir(path, { recursive: true, mode: 0o700 }); await chmod(path, 0o700); };
	{
		let root = await mkdtemp(resolve(tmpdir(), "countershape-u7-private-root-mode-"));
		root = await realpath(root);
		try {
			for (const mode of [0o755, 0o1700, 0o2700, 0o4700]) {
				await chmod(root, mode);
				await expectRunbookRecoveryError(requirePrivateDirectoryNoFollow(root, "selftest final root"), "RUNBOOK_RECOVERY_DIRECTORY");
				exercised += 1;
			}
			await chmod(root, 0o700);
			await requirePrivateDirectoryNoFollow(root, "selftest final root");
			exercised += 1;
		} finally {
			await rm(root, { recursive: true, force: true });
		}
	}
	const archiveCase = async (name, shape, token = undefined, precommit = false) => {
		let root = await mkdtemp(resolve(tmpdir(), `countershape-u7-recovery-${name}-`));
		root = await realpath(root);
		try {
			await chmod(root, 0o700);
			await makeDirectory(resolve(root, ".countershape"));
			await makeDirectory(resolve(root, ".didrun-history"));
			const paths = precommit ? precommitRecoveryPaths(unit, slug, root) : postcommitRecoveryPaths(unit, commit, root);
			if (shape.liveLedger) await makeDirectory(paths.liveLedger);
			if (shape.normal === "EMPTY") await makeDirectory(paths.normalLedgerRoot);
			if (shape.normal === "COMPLETE") { await makeDirectory(paths.normalLedgerRoot); await makeDirectory(resolve(paths.normalLedgerRoot, ".didrun")); }
			if (shape.failedLedger === "EMPTY") await makeDirectory(paths.failedLedgerRoot);
			if (shape.failedLedger === "COMPLETE") { await makeDirectory(paths.failedLedgerRoot); await makeDirectory(resolve(paths.failedLedgerRoot, ".didrun")); }
			if (shape.failedLedger === "SYMLINK") await symlink("absent-ledger-target", paths.failedLedgerRoot);
			if (shape.normal === "EXTRA") { await makeDirectory(paths.normalLedgerRoot); await writeFile(resolve(paths.normalLedgerRoot, "extra"), "hostile", { mode: 0o600 }); }
			if (["LIVE", "BOTH"].includes(shape.final)) await makeDirectory(paths.liveFinalRoot);
			if (["FAILED", "BOTH"].includes(shape.final)) await makeDirectory(paths.failedFinalRoot);
			if (shape.liveLedgerMode !== undefined) await chmod(paths.liveLedger, shape.liveLedgerMode);
			const invocation = normalizeFailedAttemptArchives(paths, precommit);
			if (token !== undefined) {
				await expectRunbookRecoveryError(invocation, token);
				exercised += 1;
				return;
			}
			const first = await invocation;
			const second = await normalizeFailedAttemptArchives(paths, precommit);
			const expectedLedger = shape.liveLedger || shape.normal !== undefined || shape.failedLedger !== undefined ? "COMPLETE" : "NONE";
			if (first.ledger !== expectedLedger || second.ledger !== expectedLedger ||
				await optionalPrivateDirectoryNoFollow(paths.liveFinalRoot, "selftest live final") !== undefined ||
				await optionalPrivateDirectoryNoFollow(paths.failedFinalRoot, "selftest failed final") === undefined) {
				fail("SELFTEST_RECOVERY_TERMINAL", name);
			}
			exercised += 1;
		} finally {
			await rm(root, { recursive: true, force: true });
		}
	};
	const completedArchiveCase = async (name, shape, token = undefined) => {
		let root = await mkdtemp(resolve(tmpdir(), `countershape-u7-completed-${name}-`));
		root = await realpath(root);
		try {
			await chmod(root, 0o700);
			await makeDirectory(resolve(root, ".countershape"));
			await makeDirectory(resolve(root, ".didrun-history"));
			const paths = postcommitRecoveryPaths(unit, commit, root);
			if (shape.live) await makeDirectory(paths.liveLedger);
			if (shape.archive === "EMPTY") await makeDirectory(paths.normalLedgerRoot);
			if (shape.archive === "COMPLETE") { await makeDirectory(paths.normalLedgerRoot); await makeDirectory(resolve(paths.normalLedgerRoot, ".didrun")); }
			if (shape.archive === "EXTRA") { await makeDirectory(paths.normalLedgerRoot); await writeFile(resolve(paths.normalLedgerRoot, "extra"), "hostile", { mode: 0o600 }); }
			if (shape.archive === "SYMLINK") await symlink("absent-archive-target", paths.normalLedgerRoot);
			if (shape.failedLedger === "EMPTY") await makeDirectory(paths.failedLedgerRoot);
			if (shape.failedLedger === "COMPLETE") { await makeDirectory(paths.failedLedgerRoot); await makeDirectory(resolve(paths.failedLedgerRoot, ".didrun")); }
			if (shape.failedFinal) await makeDirectory(paths.failedFinalRoot);
			if (shape.archiveMode !== undefined) await chmod(paths.normalLedgerRoot, shape.archiveMode);
			const invocation = normalizeCompletedLedgerArchive(paths);
			if (token !== undefined) {
				await expectRunbookRecoveryError(invocation, token);
				exercised += 1;
				return;
			}
			await invocation;
			await normalizeCompletedLedgerArchive(paths);
			if (await classifyCompletedLedgerArchive(paths) !== "COMPLETE") fail("SELFTEST_ARCHIVE_TERMINAL", name);
			exercised += 1;
		} finally {
			await rm(root, { recursive: true, force: true });
		}
	};
	for (const hostile of ["", "../escape", "20260808T120000Z-UPPER", "20260808T120000Z-trailing-", "20260808T120000Z-a".padEnd(80, "a")]) {
		try { precommitRecoveryPaths(unit, hostile, repositoryRoot); }
		catch (error) {
			if (String(error.message).includes("RUNBOOK_FAILED_ATTEMPT_SLUG")) { exercised += 1; continue; }
			throw error;
		}
		fail("SELFTEST_RECOVERY_SLUG_FALSE_NEGATIVE", hostile);
	}
	await archiveCase("live", { liveLedger: true, final: "LIVE" });
	await archiveCase("normal-empty", { liveLedger: true, normal: "EMPTY", final: "LIVE" });
	await archiveCase("normal-complete", { normal: "COMPLETE", final: "LIVE" });
	await archiveCase("failed-empty", { liveLedger: true, failedLedger: "EMPTY", final: "LIVE" });
	await archiveCase("complete-resume", { failedLedger: "COMPLETE", final: "FAILED" });
	await archiveCase("precommit-no-ledger", { final: "LIVE" }, undefined, true);
	await archiveCase("ledger-both", { liveLedger: true, normal: "COMPLETE", final: "LIVE" }, "RUNBOOK_RECOVERY_LEDGER_STATE");
	await archiveCase("ledger-extra", { normal: "EXTRA", final: "LIVE" }, "RUNBOOK_RECOVERY_ENTRIES");
	await archiveCase("ledger-symlink", { failedLedger: "SYMLINK", final: "LIVE" }, "RUNBOOK_RECOVERY_DIRECTORY");
	await archiveCase("ledger-mode", { liveLedger: true, liveLedgerMode: 0o777, final: "LIVE" }, "RUNBOOK_RECOVERY_DIRECTORY");
	await archiveCase("final-both", { failedLedger: "COMPLETE", final: "BOTH" }, "RUNBOOK_RECOVERY_FINAL_ROOT_STATE");
	await archiveCase("final-neither", { failedLedger: "COMPLETE", final: "NONE" }, "RUNBOOK_RECOVERY_FINAL_ROOT_STATE");
	{
		let root = await mkdtemp(resolve(tmpdir(), "countershape-u7-recovery-cross-device-"));
		root = await realpath(root);
		try {
			await chmod(root, 0o700);
			await makeDirectory(resolve(root, ".countershape"));
			await makeDirectory(resolve(root, ".didrun-history"));
			const paths = postcommitRecoveryPaths(unit, commit, root);
			await makeDirectory(paths.liveLedger);
			await makeDirectory(paths.normalLedgerRoot);
			await makeDirectory(paths.liveFinalRoot);
			let deviceChecks = 0;
			await expectRunbookRecoveryError(normalizeFailedAttemptArchives(paths, false, {
				sameDevice: async () => { deviceChecks += 1; if (deviceChecks === 2) fail("RUNBOOK_RECOVERY_CROSS_DEVICE", "second transition fixture"); },
			}), "RUNBOOK_RECOVERY_CROSS_DEVICE");
			if (deviceChecks !== 2 || await classifyRecoveryLedger(paths, false) !== "NORMAL_EMPTY_WITH_LIVE" || await classifyRecoveryFinalRoot(paths) !== "LIVE") {
				fail("SELFTEST_RECOVERY_CROSS_DEVICE_MUTATION", `${deviceChecks}`);
			}
			exercised += 1;
		} finally {
			await rm(root, { recursive: true, force: true });
		}
	}
	await completedArchiveCase("live", { live: true });
	await completedArchiveCase("empty-resume", { live: true, archive: "EMPTY" });
	await completedArchiveCase("complete-resume", { archive: "COMPLETE" });
	await completedArchiveCase("both", { live: true, archive: "COMPLETE" }, "RUNBOOK_ARCHIVE_STATE");
	await completedArchiveCase("neither", {}, "RUNBOOK_ARCHIVE_STATE");
	await completedArchiveCase("extra", { archive: "EXTRA" }, "RUNBOOK_RECOVERY_ENTRIES");
	await completedArchiveCase("symlink", { archive: "SYMLINK" }, "RUNBOOK_RECOVERY_DIRECTORY");
	await completedArchiveCase("mode", { archive: "COMPLETE", archiveMode: 0o777 }, "RUNBOOK_RECOVERY_DIRECTORY");
	await completedArchiveCase("failed-empty", { live: true, failedLedger: "EMPTY" }, "RUNBOOK_ARCHIVE_FAILED_STATE");
	await completedArchiveCase("failed-complete", { archive: "COMPLETE", failedLedger: "COMPLETE" }, "RUNBOOK_ARCHIVE_FAILED_STATE");
	await completedArchiveCase("failed-final", { live: true, failedFinal: true }, "RUNBOOK_ARCHIVE_FAILED_STATE");

	const tree = "b".repeat(40);
	const parent = "c".repeat(40);
	const failedRefName = `refs/countershape/failed-attempts/${unit.id.toLowerCase()}/${commit}`;
	if (parseOptionalDirectGitRef(failedRefName, Buffer.alloc(0)) !== undefined ||
		parseOptionalDirectGitRef(failedRefName, Buffer.from(`${commit}\0\0${failedRefName}\n`, "utf8")) !== commit) {
		fail("SELFTEST_RECOVERY_DIRECT_REF", "positive");
	}
	for (const [name, bytes] of [
		["symbolic", Buffer.from(`${commit}\0refs/heads/current\0${failedRefName}\n`, "utf8")],
		["wrong-name", Buffer.from(`${commit}\0\0refs/countershape/failed-attempts/other\n`, "utf8")],
		["duplicate", Buffer.from(`${commit}\0\0${failedRefName}\n${commit}\0\0${failedRefName}\n`, "utf8")],
	]) {
		try { parseOptionalDirectGitRef(failedRefName, bytes); }
		catch (error) { if (String(error.message).includes("RUNBOOK_RECOVERY_REF")) { exercised += 1; continue; } throw error; }
		fail("SELFTEST_RECOVERY_DIRECT_REF_FALSE_NEGATIVE", name);
	}
	const archiveResult = Object.freeze({ ledger: "COMPLETE", paths: Object.freeze({ failedFinalRootRelative: `${unit.final_root}-failed-${commit}` }) });
	const makeGitFixture = (overrides = {}) => {
		const state = { head: overrides.head ?? commit, ref: Object.hasOwn(overrides, "ref") ? overrides.ref : undefined, tree: overrides.tree ?? tree, create: 0, cas: 0, candidate: 0, validate: 0, workspace: 0, archive: 0 };
		return Object.freeze({
			state,
			dependencies: {
				specification, root: repositoryRoot,
				validateCommit: async () => { state.validate += 1; return overrides.validated?.(state) ?? Object.freeze({ commit, tree, parent }); },
				head: async () => state.head, indexTree: async () => state.tree, readRef: async () => { if (overrides.refReadError) fail("RUNBOOK_RECOVERY_REF", "symbolic fixture"); return state.ref; },
					createRef: async () => { state.create += 1; if (state.ref !== undefined) throw new Error("overwrite"); state.ref = commit; },
					casHead: async (next) => { state.cas += 1; if (!overrides.casStalls) state.head = next; },
					workspace: async () => { state.workspace += 1; if (state.workspace === overrides.workspaceFailureAt) fail("RUNBOOK_RECOVERY_WORKTREE", String(state.workspace)); },
					candidate: async () => { state.candidate += 1; }, archive: async () => { state.archive += 1; return archiveResult; }, write: () => {},
			},
		});
	};
	const first = makeGitFixture();
	await recoverFailedPostcommit(unit.id, commit, first.dependencies);
	await recoverFailedPostcommit(unit.id, commit, first.dependencies);
	if (first.state.head !== parent || first.state.ref !== commit || first.state.create !== 1 || first.state.cas !== 1 || first.state.candidate !== 2 || first.state.workspace !== 4) {
		fail("SELFTEST_RECOVERY_GIT_TERMINAL", JSON.stringify(first.state));
	}
	exercised += 2;
	const rotationRoot = `.didrun-history/${unit.id.toLowerCase()}-final-${commit.slice(0, 12)}`;
	const rotationState = { workspace: 0, normalize: 0, archive: 0, refReads: 0 };
	const rotationDependencies = (overrides = {}) => ({
		specification, root: repositoryRoot, commit: async () => commit,
		validateCommit: async () => Object.freeze({ commit, tree, parent }), preflight: async () => {},
		workspace: async () => { rotationState.workspace += 1; if (rotationState.workspace === overrides.workspaceFailureAt) fail("RUNBOOK_RECOVERY_WORKTREE", String(rotationState.workspace)); },
		readFailedRef: async () => { rotationState.refReads += 1; return overrides.failedRefAt === rotationState.refReads ? commit : undefined; },
		normalize: async () => { rotationState.normalize += 1; }, validateArchive: async () => { rotationState.archive += 1; }, write: () => {},
	});
	await rotateFinalLedger(unit.id, rotationRoot, rotationDependencies());
	if (!isDeepStrictEqual(rotationState, { workspace: 3, normalize: 1, archive: 1, refReads: 3 })) fail("SELFTEST_ARCHIVE_WORKSPACE_TERMINAL", JSON.stringify(rotationState));
	exercised += 1;
	for (const [name, overrides, token, expectedNormalizations] of [
		["workspace-before", { workspaceFailureAt: 1 }, "RUNBOOK_RECOVERY_WORKTREE", 0],
		["failed-ref-before", { failedRefAt: 1 }, "RUNBOOK_ARCHIVE_FAILED_STATE", 0],
		["workspace-after-normalize", { workspaceFailureAt: 2 }, "RUNBOOK_RECOVERY_WORKTREE", 1],
		["failed-ref-after-normalize", { failedRefAt: 2 }, "RUNBOOK_ARCHIVE_FAILED_STATE", 1],
	]) {
		Object.assign(rotationState, { workspace: 0, normalize: 0, archive: 0, refReads: 0 });
		await expectRunbookRecoveryError(rotateFinalLedger(unit.id, rotationRoot, rotationDependencies(overrides)), token);
		if (rotationState.normalize !== expectedNormalizations || name.length === 0) fail("SELFTEST_ARCHIVE_MUTATION_ORDER", `${name}:${JSON.stringify(rotationState)}`);
		exercised += 1;
	}
	for (const [name, overrides, token] of [
		["symbolic-ref", { refReadError: true }, "RUNBOOK_RECOVERY_REF"],
		["parent-without-ref", { head: parent }, "RUNBOOK_RECOVERY_GIT_STATE"],
		["wrong-ref", { ref: "d".repeat(40) }, "RUNBOOK_RECOVERY_GIT_STATE"],
		["wrong-index", { tree: "e".repeat(40) }, "RUNBOOK_RECOVERY_INDEX"],
		["cas-stalls", { casStalls: true }, "RUNBOOK_RECOVERY_TERMINAL"],
		["workspace-drifts", { workspaceFailureAt: 2 }, "RUNBOOK_RECOVERY_WORKTREE"],
		["commit-drifts", { validated: (state) => state.validate === 1 ? Object.freeze({ commit, tree, parent }) : Object.freeze({ commit, tree: "f".repeat(40), parent }) }, "RUNBOOK_RECOVERY_COMMIT_DRIFT"],
	]) {
		const fixture = makeGitFixture(overrides);
		await expectRunbookRecoveryError(recoverFailedPostcommit(unit.id, commit, fixture.dependencies), token);
		exercised += 1;
		if (name === "symbolic-ref" && (fixture.state.archive !== 0 || fixture.state.cas !== 0 || fixture.state.create !== 0)) {
			fail("SELFTEST_RECOVERY_SYMBOLIC_REF_MUTATION", JSON.stringify(fixture.state));
		}
		if (name.length === 0) fail("SELFTEST_RECOVERY_GIT_CASE", name);
	}
	let precommitTree = tree;
	let precommitCandidateCalls = 0;
	await recoverFailedPrecommit(unit.id, slug, {
		specification, root: repositoryRoot, candidate: async () => { precommitCandidateCalls += 1; },
		head: async () => parent, indexTree: async () => precommitTree,
		archive: async () => Object.freeze({ ledger: "NONE", paths: Object.freeze({ failedFinalRootRelative: `${unit.final_root}-precommit-failed-${slug}` }) }),
		write: () => {},
	});
	if (precommitCandidateCalls !== 2) fail("SELFTEST_RECOVERY_PRECOMMIT_CANDIDATE", String(precommitCandidateCalls));
	exercised += 1;
	await expectRunbookRecoveryError(recoverFailedPrecommit(unit.id, slug, {
		specification, root: repositoryRoot, candidate: async () => {}, head: async () => parent,
		indexTree: async () => { const value = precommitTree; precommitTree = "e".repeat(40); return value; },
		archive: async () => Object.freeze({ ledger: "NONE", paths: Object.freeze({ failedFinalRootRelative: `${unit.final_root}-precommit-failed-${slug}` }) }), write: () => {},
	}), "RUNBOOK_RECOVERY_GIT_DRIFT");
	exercised += 1;
	return exercised;
}

export async function selfTest() {
	const base = clone(await loadSpecification());
	await validateC6BCommit(base, base.sealed_parent.commit);
	const mutations = [
		["SPEC_KEYS", (value) => { value.extra = true; }],
		["AUTHORITY", (value) => { value.transition_authority.provenance = "CITED_UNAUTHENTICATED"; }],
		["AMENDMENT_AUTHORITY", (value) => { value.topology_amendment_authority.authentication = "ESTABLISHED"; }],
		["RECEIPT_CONTRACT", (value) => { value.receipt_contract.study_run_count_per_domain = 2; }],
		["CLAIM_BINDING_POLICY", (value) => { value.claim_binding_policy.pathspecs = "EMPTY"; }],
		["CLAIM_BINDING_POLICY", (value) => { value.claim_binding_policy.recorded_child_process_intervals = "ALLOW_OVERLAP"; }],
		["CLAIM_BINDING_POLICY", (value) => { value.claim_binding_policy.operational_writer_protocol = "MULTI_WRITER"; }],
		["SEALED_PARENT", (value) => { value.sealed_parent.tree = "0".repeat(40); }],
		["UNIT_ORDER", (value) => { [value.units[1], value.units[2]] = [value.units[2], value.units[1]]; }],
		["UNIT_IDENTITY", (value) => { value.units[1].parent = "C6B"; }],
		["UNIT_IDENTITY", (value) => { value.units[0].product_authority = "U7_REFERENCE_APPLICATION"; }],
		["UNIT_PATHS", (value) => { value.units[0].allowed_paths.push(value.units[0].allowed_paths[0]); }],
		["UNIT_PATHS", (value) => { value.units[1].allowed_paths[0] = "../escape"; }],
		["REQUIRED_OUTSIDE_ALLOWED", (value) => { value.units[1].required_paths.push("not/allowed.txt"); }],
		["CLAIM_COUNT", (value) => { value.units[2].claims.pop(); }],
		["CLAIM", (value) => { value.units[1].claims[0].type = "builds-maybe"; }],
		["CLAIM", (value) => { value.units[1].claims[1].label = value.units[1].claims[0].label; }],
		["CLAIM", (value) => { value.units[1].claims[0].command[0] = "node"; }],
		["PRESEAL_NOT_LAST", (value) => { value.units[2].claims.at(-1).command.push("--extra"); }],
		["U7P_CONTRACT", (value) => {
			[value.units[0].allowed_paths[0], value.units[0].allowed_paths[1]] = [value.units[0].allowed_paths[1], value.units[0].allowed_paths[0]];
			[value.units[0].required_paths[0], value.units[0].required_paths[1]] = [value.units[0].required_paths[1], value.units[0].required_paths[0]];
		}],
		["U7P_CONTRACT", (value) => { value.units[0].claims[7].label += " altered"; }],
		["U7M_CONTRACT", (value) => { value.units[1].claims[2].label += " altered"; }],
		["FUTURE_CONTROL_PLANE_OWNERSHIP", (value) => {
			value.units[2].allowed_paths.push("tools/verify-current.mjs");
			value.units[2].required_paths.push("tools/verify-current.mjs");
		}],
	];
	for (const [token, mutate] of mutations) {
		const candidate = clone(base);
		mutate(candidate);
		expectRejected(candidate, token);
	}
	const intervalEntries = (intervals) => intervals.map(([started_at, ended_at]) => ({ event: { started_at, ended_at } }));
	validateRecordedChildIntervals(intervalEntries([[1, 2], [2, 2], [3.5, 4]]), "selftest-positive");
	const intervalHostiles = [
		["RECORDED_CHILD_INTERVAL_OVERLAP", intervalEntries([[1, 3], [2, 4], [4, 5]])],
		["RECORDED_CHILD_INTERVAL_OVERLAP", intervalEntries([[1, 2], [2, 4], [3.5, 5]])],
		["RECORDED_CHILD_INTERVAL", [{ event: { ended_at: 2 } }]],
		["RECORDED_CHILD_INTERVAL", intervalEntries([[1, Number.POSITIVE_INFINITY]])],
		["RECORDED_CHILD_INTERVAL", intervalEntries([[3, 2]])],
	];
	for (const [token, entries] of intervalHostiles) {
		try { validateRecordedChildIntervals(entries, "selftest-hostile"); }
		catch (error) { if (String(error.message).includes(token)) continue; fail("SELFTEST_INTERVAL_WRONG_REJECTION", `${token}:${error.message}`); }
		fail("SELFTEST_INTERVAL_FALSE_NEGATIVE", token);
	}
	const capsuleBlock = `${capsuleStart}\n### Active U7 phase contract\n\n- **Namespace:** \`countershape/u7-unit-paths/v2\`\n- **Boundary:** \`U7P\`\n- **Parent:** \`C6B\`\n- **Verification profile:** \`SOURCE_FULL\`\n- **State:** \`SOURCE_CANDIDATE\`\n- **Topology:** \`U7P -> U7M -> U7A -> U7B -> U7C -> U7D -> U7R\`\n- **Inherited receipts:** \`C3P=PRESENT; C3=PRESENT; C6A=PRESENT\`\n- **Receipt U7D:** \`ABSENT\`\n- **Product authority:** \`NONE\`\n- **Product behavior:** \`INHERITED_UNREPROVEN\`\n${capsuleEnd}`;
	const capsule = `# U7 fixture\n\n## Current state\n\n${capsuleBlock}\n\nFrozen compatibility follows.\n\n## History\n\nNone.\n`;
	if (!isDeepStrictEqual(parsePhaseCapsule(capsule), {
		boundary: "U7P", parent: "C6B", profile: "SOURCE_FULL", state: "SOURCE_CANDIDATE", receipt: "ABSENT",
		productAuthority: "NONE", productBehavior: "INHERITED_UNREPROVEN",
	})) fail("SELFTEST_CAPSULE", "positive");
	const u7mCapsule = capsule.replace("- **Boundary:** `U7P`", "- **Boundary:** `U7M`").replace("- **Parent:** `C6B`", "- **Parent:** `U7P`");
	if (!isDeepStrictEqual(parsePhaseCapsule(u7mCapsule), {
		boundary: "U7M", parent: "U7P", profile: "SOURCE_FULL", state: "SOURCE_CANDIDATE", receipt: "ABSENT",
		productAuthority: "NONE", productBehavior: "INHERITED_UNREPROVEN",
	})) fail("SELFTEST_CAPSULE", "U7M positive");
	for (const hostile of [
		capsule.replace("SOURCE_FULL", "SOURCE_PARTIAL"), `${capsule}\n${capsuleBlock}`,
		capsule.replace("SOURCE_CANDIDATE", "SEALED"),
		capsule.replace(capsuleBlock, `\`\`\`markdown\n${capsuleBlock}\n\`\`\``),
		capsule.replace(capsuleBlock, `<!--\n${capsuleBlock}\n-->`),
		capsule.replace(`\n${capsuleBlock}\n`, `\nMoved.\n\n## History\n\n${capsuleBlock}\n`),
	]) {
		try { parsePhaseCapsule(hostile); } catch { continue; }
		fail("SELFTEST_CAPSULE_FALSE_NEGATIVE", hostile.slice(0, 80));
	}
	validateReceiptPhase("U7P", capsule);
	validateReceiptPhase("U7M", u7mCapsule);
	const u7dReceiptFixture = `# U7D\n\n## Current state\n\n${pendingReceiptBlock}\n\n## History\n\nFrozen.\n`;
	const u7dStatusFixture = `# U7D evidence\n\n## Source receipt\n\n${pendingReceiptBlock}\n`;
	const u7rReceiptFixture = u7dReceiptFixture.replace(pendingReceiptBlock, `${receiptBlockStart}\nsealed\n${receiptBlockEnd}`);
	validateReceiptPhase("U7D", u7dReceiptFixture);
	validateReceiptPhase("U7R", u7rReceiptFixture);
	validatePendingProjectionDocuments(base, new Map([["docs/HANDOFF_MODE_C.md", u7dReceiptFixture], ["docs/status/U7D-EVIDENCE.md", u7dStatusFixture]]));
	for (const [unitID, hostile] of [
		["U7C", u7dReceiptFixture], ["U7D", capsule], ["U7D", u7rReceiptFixture], ["U7R", u7dReceiptFixture],
	]) {
		try { validateReceiptPhase(unitID, hostile); }
		catch (error) { if (String(error.message).includes("RECEIPT_PHASE")) continue; throw error; }
		fail("SELFTEST_RECEIPT_PHASE_FALSE_NEGATIVE", unitID);
	}
	const pendingSetHostiles = [
		["RECEIPT_PENDING_SET", new Map([["docs/HANDOFF_MODE_C.md", u7dReceiptFixture]])],
		["RECEIPT_PROJECTION", new Map([["docs/HANDOFF_MODE_C.md", `${u7dReceiptFixture}\n${pendingReceiptBlock}`], ["docs/status/U7D-EVIDENCE.md", u7dStatusFixture]])],
		["MARKDOWN_VISIBILITY", new Map([["docs/HANDOFF_MODE_C.md", u7dReceiptFixture], ["docs/status/U7D-EVIDENCE.md", u7dStatusFixture.replace(pendingReceiptBlock, `\`\`\`md\n${pendingReceiptBlock}\n\`\`\``)]])],
		["MARKDOWN_HEADING", new Map([["docs/HANDOFF_MODE_C.md", u7dReceiptFixture], ["docs/status/U7D-EVIDENCE.md", u7dStatusFixture.replace("## Source receipt", "## Historical receipt")]])],
	];
	for (const [token, documents] of pendingSetHostiles) {
		try { validatePendingProjectionDocuments(base, documents); }
		catch (error) { if (String(error.message).includes(token)) continue; fail("SELFTEST_PENDING_SET_WRONG_REJECTION", `${token}:${error.message}`); }
		fail("SELFTEST_PENDING_SET_FALSE_NEGATIVE", token);
	}
	const prefix = hermeticPrefix(base.units[0]);
	if (new Set(prefix).size !== prefix.length || prefix[0] !== "/usr/bin/env" || prefix[1] !== "-i" ||
		!prefix.includes(`GOCACHE=${resolve(repositoryRoot, ".countershape/u7p-final/gocache")}`)) fail("SELFTEST_PREFIX", prefix.join(" "));
	const rawPreview = [...prefix, ...base.units[0].claims[0].command];
	const scrubbedPreview = [...rawPreview];
	for (const position of noteRootRedactionPositions) {
		const partial = expectedRedactions(position, rawPreview[position]).find((value) => value !== `${noteRedaction}.${noteRedaction}`);
		if (partial === undefined) fail("SELFTEST_REDACTION_FIXTURE", String(position));
		scrubbedPreview[position] = partial;
	}
	for (const position of noteBareRedactionPositions) scrubbedPreview[position] = noteRedaction;
	if (!claimPreviewMatches(scrubbedPreview, rawPreview, prefix)) fail("SELFTEST_REDACTION_FIXTURE", "future positive");
	const hostilePreview = [...scrubbedPreview];
	hostilePreview[9] = noteRedaction;
	if (claimPreviewMatches(hostilePreview, rawPreview, prefix)) fail("SELFTEST_REDACTION_FIXTURE", "wrong-position accepted");
	const p07Inputs = inheritedP07CompatibilityInputs(base);
	const p07Result = validateInheritedP07Compatibility(p07Inputs.parentHandoff, p07Inputs.candidateHandoff, p07Inputs.parentProtected, p07Inputs.candidateProtected);
	if (!isDeepStrictEqual(p07Result, { blocks: 11, protectedPaths: 4 })) fail("SELFTEST_P07_COMPATIBILITY", JSON.stringify(p07Result));
	const expectP07Refusal = (name, token, mutate) => {
		const candidateHandoff = { value: p07Inputs.candidateHandoff };
		const candidateProtected = new Map(p07Inputs.candidateProtected);
		mutate(candidateHandoff, candidateProtected);
		try { validateInheritedP07Compatibility(p07Inputs.parentHandoff, candidateHandoff.value, p07Inputs.parentProtected, candidateProtected); }
		catch (error) { if (String(error.message).includes(token)) return; fail("SELFTEST_P07_COMPATIBILITY_WRONG_REJECTION", `${name}:${error.message}`); }
		fail("SELFTEST_P07_COMPATIBILITY_FALSE_NEGATIVE", name);
	};
	const firstP07Base = inheritedP07MarkerBases[0];
	const firstP07End = `<!-- ${firstP07Base}:END -->`;
	expectP07Refusal("block-byte", "P07_COMPATIBILITY_BLOCK", (handoff) => { handoff.value = handoff.value.replace(firstP07End, `tampered\n${firstP07End}`); });
	expectP07Refusal("hidden-block", "MARKDOWN_VISIBILITY", (handoff) => {
		const block = exactVisibleMarkedBlock(handoff.value, firstP07Base, "selftest-source");
		handoff.value = handoff.value.replace(block, `\`\`\`md\n${block}\n\`\`\``);
	});
	expectP07Refusal("duplicate-marker", "P07_COMPATIBILITY_MARKERS", (handoff) => { handoff.value += `\n<!-- ${firstP07Base}:START -->\n`; });
	expectP07Refusal("protected-byte", "P07_COMPATIBILITY_PROTECTED_PATH", (_handoff, protectedPaths) => {
		protectedPaths.set(inheritedP07ProtectedPaths[0], Buffer.from("hostile\n", "utf8"));
	});
	const p07AuthorityInputs = inheritedP07C6AuthorityInputs(base);
	const p07Authority = validateInheritedP07C6Authority(p07AuthorityInputs);
	if (!isDeepStrictEqual(Object.keys(p07Authority).sort(), [
		"commit", "noteBlob", "noteBodySHA256", "parent", "secretsOverride", "subject", "tree",
	])) fail("SELFTEST_P07_C6A_AUTHORITY", "positive shape");
	const cloneAuthorityInputs = () => ({
		sealedTracked: new Map([...p07AuthorityInputs.sealedTracked].map(([path, bytes]) => [path, Buffer.from(bytes)])),
		candidateTracked: new Map([...p07AuthorityInputs.candidateTracked].map(([path, bytes]) => [path, Buffer.from(bytes)])),
		source: { ...p07AuthorityInputs.source },
		noteBytes: Buffer.from(p07AuthorityInputs.noteBytes),
		sealedSourceSummary: Buffer.from(p07AuthorityInputs.sealedSourceSummary),
	});
	const authorityHostiles = [
		["P07_C6A_AUTHORITY_PROJECTION", (value) => { value.candidateTracked.set(inheritedP07C6AuthorityPaths[1], Buffer.from("{}\n")); }],
		["P07_C6A_AUTHORITY_JSON", (value) => {
			value.sealedTracked.set(inheritedP07C6AuthorityPaths[0], Buffer.from("{\n"));
			value.candidateTracked.set(inheritedP07C6AuthorityPaths[0], Buffer.from("{\n"));
		}],
		["P07_C6A_AUTHORITY_IDENTITY", (value) => { value.source.tree = "f".repeat(40); }],
		["P07_C6A_AUTHORITY_IDENTITY", (value) => { value.noteBytes = Buffer.from("hostile note\n"); }],
		["P07_C6A_AUTHORITY_SUMMARY", (value) => { value.sealedSourceSummary = Buffer.from("hostile summary\n"); }],
	];
	for (const [token, mutate] of authorityHostiles) {
		const hostile = cloneAuthorityInputs(); mutate(hostile);
		try { validateInheritedP07C6Authority(hostile); }
		catch (error) { if (String(error.message).includes(token)) continue; fail("SELFTEST_P07_C6A_AUTHORITY_WRONG_REJECTION", `${token}:${error.message}`); }
		fail("SELFTEST_P07_C6A_AUTHORITY_FALSE_NEGATIVE", token);
	}
	let capturedPlanArgs;
	await runInheritedP07SuccessorPlan(p07Authority, async (...args) => { capturedPlanArgs = args; return []; });
	if (!Array.isArray(capturedPlanArgs) || capturedPlanArgs.length !== 15 || capturedPlanArgs[0] !== repositoryRoot ||
		!(capturedPlanArgs[1] instanceof Map) || capturedPlanArgs[1].size !== 0 ||
		capturedPlanArgs.slice(2, 11).some((value) => value !== undefined) ||
		capturedPlanArgs[11] !== "C6B" || capturedPlanArgs[12] !== "C6B" || capturedPlanArgs[13] !== undefined ||
		capturedPlanArgs[14] !== p07Authority) {
		fail("SELFTEST_P07_SUCCESSOR_PLAN_ARGUMENTS", JSON.stringify(capturedPlanArgs));
	}
	const planResultHostiles = [
		["P07_SUCCESSOR_PLAN_RESULT", undefined],
		["P07_SUCCESSOR_PLAN_RESULT", [42]],
		["P07_SUCCESSOR_PLAN", ["legacy plan error"]],
	];
	for (const [token, resultValue] of planResultHostiles) {
		try { await runInheritedP07SuccessorPlan(p07Authority, async () => resultValue); }
		catch (error) { if (String(error.message).includes(token)) continue; fail("SELFTEST_P07_SUCCESSOR_PLAN_WRONG_REJECTION", `${token}:${error.message}`); }
		fail("SELFTEST_P07_SUCCESSOR_PLAN_FALSE_NEGATIVE", token);
	}
	let planThrowObserved = false;
	try { await runInheritedP07SuccessorPlan(p07Authority, async () => { throw new Error("legacy plan threw"); }); }
	catch (error) { if (String(error.message) === "legacy plan threw") planThrowObserved = true; else throw error; }
	if (!planThrowObserved) fail("SELFTEST_P07_SUCCESSOR_PLAN_FALSE_GREEN", "throw");
	const compatibilityWrites = [];
	const compatibilityOrder = [];
	let compatibilityCandidateCalls = 0;
	let compatibilitySnapshots = 0;
	const stableCompositeSnapshot = Object.freeze({
		candidate: Object.freeze({ head: "1".repeat(40), indexTree: "2".repeat(40), paths: Object.freeze([]), blobDigests: Object.freeze([]) }),
		notesTree: "3".repeat(40),
	});
	const compositeResult = await verifyInheritedP07Compatibility({
		specification: base, capsule: Object.freeze({ boundary: "U7P" }),
		candidate: async (boundary, writeSuccess) => {
			compatibilityOrder.push(`candidate-${compatibilityCandidateCalls + 1}`);
			compatibilityCandidateCalls += 1;
			if (boundary !== "U7P") fail("SELFTEST_P07_COMPATIBILITY_WRAPPER_BOUNDARY", boundary);
			writeSuccess("nested candidate narration must be suppressed");
		},
		snapshot: async () => { compatibilityOrder.push(`snapshot-${compatibilitySnapshots + 1}`); compatibilitySnapshots += 1; return stableCompositeSnapshot; },
		get inputs() {
			compatibilityOrder.push("compatibility-inputs");
			return Object.freeze({ parentHandoff: "", candidateHandoff: "", parentProtected: new Map(), candidateProtected: new Map() });
		},
		validate: () => { compatibilityOrder.push("validate"); return Object.freeze({ blocks: 11, protectedPaths: 4 }); },
		get authorityInputs() { compatibilityOrder.push("authority-inputs"); return Object.freeze({}); },
		validateAuthority: () => { compatibilityOrder.push("validate-authority"); return p07Authority; },
		runPlan: async (authority) => {
			compatibilityOrder.push("plan");
			if (!isDeepStrictEqual(authority, p07Authority)) fail("SELFTEST_P07_COMPATIBILITY_PLAN_AUTHORITY", "drift");
		},
		reload: async () => { compatibilityOrder.push("reload"); return base; },
		write: (line) => { compatibilityOrder.push("write"); compatibilityWrites.push(line); },
	});
	const compatibilityLine = "U7 inherited P07 compatibility exact: parent=C6B blocks=11 protected_paths=4 c3p_plan=FULL_EXPLICIT_C6A_AUTHORITY legacy_live_entrypoints=RETIRED_SUCCESSOR_INCOMPATIBLE";
	if (compatibilityCandidateCalls !== 2 || compatibilitySnapshots !== 2 ||
		!isDeepStrictEqual(compositeResult, { blocks: 11, protectedPaths: 4, c3pPlan: "FULL_EXPLICIT_C6A_AUTHORITY" }) ||
		!isDeepStrictEqual(compatibilityWrites, [compatibilityLine]) || !isDeepStrictEqual(compatibilityOrder, [
			"snapshot-1", "candidate-1", "compatibility-inputs", "validate", "authority-inputs",
			"validate-authority", "plan", "candidate-2", "reload", "snapshot-2", "write",
		])) {
		fail("SELFTEST_P07_COMPATIBILITY_WRAPPER_OUTPUT", JSON.stringify({ compatibilityCandidateCalls, compositeResult, compatibilityWrites, compatibilityOrder }));
	}
	let candidateFailureObserved = false;
	try {
		await verifyInheritedP07Compatibility({
			specification: base, capsule: Object.freeze({ boundary: "U7P" }),
			snapshot: async () => stableCompositeSnapshot,
			candidate: async () => { throw new Error("candidate refused"); },
			get inputs() { fail("SELFTEST_P07_COMPATIBILITY_FAILURE_SNAPSHOT", "candidate refusal"); },
			validate: () => Object.freeze({ blocks: 11, protectedPaths: 4 }),
			write: () => fail("SELFTEST_P07_COMPATIBILITY_FAILURE_NARRATED", "candidate refusal"),
		});
	} catch (error) {
		if (String(error.message) === "candidate refused") candidateFailureObserved = true;
		else throw error;
	}
	if (!candidateFailureObserved) fail("SELFTEST_P07_COMPATIBILITY_FAILURE_FALSE_GREEN", "candidate refusal");
	let planFailureObserved = false;
	try {
		await verifyInheritedP07Compatibility({
			specification: base, capsule: Object.freeze({ boundary: "U7P" }), snapshot: async () => stableCompositeSnapshot,
			candidate: async () => {},
			inputs: Object.freeze({ parentHandoff: "", candidateHandoff: "", parentProtected: new Map(), candidateProtected: new Map() }),
			validate: () => Object.freeze({ blocks: 11, protectedPaths: 4 }), authorityInputs: Object.freeze({}),
			validateAuthority: () => p07Authority, runPlan: async () => { throw new Error("plan refused"); },
			write: () => fail("SELFTEST_P07_COMPATIBILITY_FAILURE_NARRATED", "plan refusal"),
		});
	} catch (error) {
		if (String(error.message) === "plan refused") planFailureObserved = true;
		else throw error;
	}
	if (!planFailureObserved) fail("SELFTEST_P07_COMPATIBILITY_FAILURE_FALSE_GREEN", "plan refusal");
	let snapshotFailureObserved = false; let reloadIntroducedDrift = false;
	try {
		await verifyInheritedP07Compatibility({
			specification: base, capsule: Object.freeze({ boundary: "U7P" }),
			snapshot: async () => reloadIntroducedDrift ?
				Object.freeze({ ...stableCompositeSnapshot, notesTree: "4".repeat(40) }) : stableCompositeSnapshot,
			candidate: async () => {},
			inputs: Object.freeze({ parentHandoff: "", candidateHandoff: "", parentProtected: new Map(), candidateProtected: new Map() }),
			validate: () => Object.freeze({ blocks: 11, protectedPaths: 4 }), authorityInputs: Object.freeze({}),
			validateAuthority: () => p07Authority, runPlan: async () => {},
			reload: async () => { reloadIntroducedDrift = true; return base; },
			write: () => fail("SELFTEST_P07_COMPATIBILITY_FAILURE_NARRATED", "snapshot drift"),
		});
	} catch (error) {
		if (String(error.message).includes("P07_SUCCESSOR_AUTHORITY_CHANGED")) snapshotFailureObserved = true;
		else throw error;
	}
	if (!snapshotFailureObserved) fail("SELFTEST_P07_COMPATIBILITY_FAILURE_FALSE_GREEN", "snapshot drift");
	const recoveryHostiles = await selfTestFailedAttemptRecovery(base);
	console.log(`U7 plan contract defensive self-test passed: ${mutations.length} schema/authority mutations, ${intervalHostiles.length} recorded child-process interval refusals, 6 visible phase-capsule mutations, 4 cross-phase receipt refusals, ${pendingSetHostiles.length} pending-projection refusals, 4 inherited-P07 compatibility refusals, ${authorityHostiles.length} sealed-C6A authority refusals, 5 legacy-plan call-contract cases, 4 compatibility-wrapper cases, ${recoveryHostiles} failed-attempt recovery refusals, exact U7P/U7M manifests, and deterministic hermetic-prefix control`);
}

async function main() {
	if (isDeepStrictEqual(process.argv.slice(2), ["--self-test"])) return selfTest();
	if (isDeepStrictEqual(process.argv.slice(2), ["--verify-source-receipt"])) return verifySourceReceipt();
	if (isDeepStrictEqual(process.argv.slice(2), ["--self-test-source-receipt"])) return selfTestSourceReceipt();
	if (isDeepStrictEqual(process.argv.slice(2), ["--verify-local-evidence"])) return verifyLocalEvidence();
	if (isDeepStrictEqual(process.argv.slice(2), ["--verify-inherited-p07-compatibility"])) return verifyInheritedP07Compatibility();
	if (process.argv.length === 5 && process.argv[2] === "--rotate-final-ledger") return rotateFinalLedger(process.argv[3], process.argv[4]);
	if (process.argv.length === 5 && process.argv[2] === "--recover-failed-precommit") return recoverFailedPrecommit(process.argv[3], process.argv[4]);
	if (process.argv.length === 5 && process.argv[2] === "--recover-failed-postcommit") return recoverFailedPostcommit(process.argv[3], process.argv[4]);
	if (process.argv.length === 5 && process.argv[2] === "--verify-runbook-preflight" && process.argv[3] === "prepare") return verifyRunbookPreflight("prepare", process.argv[4]);
	if (process.argv.length === 6 && process.argv[2] === "--verify-runbook-preflight" &&
		["commit", "report", "archive", "archive-complete"].includes(process.argv[3])) {
		return verifyRunbookPreflight(process.argv[3], process.argv[4], process.argv[5]);
	}
	if (process.argv.length === 4 && process.argv[2] === "--check-candidate") return checkCandidate(process.argv[3]);
	if (process.argv.length === 4 && process.argv[2] === "--verify-preseal") return verifyPreseal(process.argv[3]);
	fail("USAGE", "check-u7-plan.mjs --check-candidate <unit> | --verify-preseal <unit> | --verify-inherited-p07-compatibility | --rotate-final-ledger <unit> <archive-root> | --recover-failed-precommit <unit> <utc-reason-slug> | --recover-failed-postcommit <unit> <40-hex-commit> | --verify-runbook-preflight prepare <unit> | --verify-runbook-preflight <commit|report|archive|archive-complete> <unit> <argument> | --self-test");
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
