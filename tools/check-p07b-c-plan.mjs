#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { constants as fsConstants } from "node:fs";
import { lstat, mkdir, mkdtemp, open, opendir, readFile, readdir, realpath, rm, rmdir, symlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, isAbsolute, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { isDeepStrictEqual } from "node:util";
import { deflateSync, inflateSync } from "node:zlib";

import { validateSpecification } from "./check-p07b-c-unit-scope.mjs";
import { qualificationCaseIDs, qualificationMatrixDigest } from "./verify-go-test-repetition.mjs";

export const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");

const c1PlanningPaths = Object.freeze({
	targetSchema: "spec/schema/v1/contract-execution-target.schema.json",
	runSchema: "spec/schema/v1/finalized-contract-run.schema.json",
	executionSchema: "spec/schema/v1/contract-execution.schema.json",
	bundleExample: "spec/examples/v1/contract-bundle.valid.json",
	targetExample: "spec/examples/v1/contract-execution-target.valid.json",
	runExample: "spec/examples/v1/finalized-contract-run.valid.json",
	executionExample: "spec/examples/v1/contract-execution.valid.json",
});

const researchShape = Object.freeze({
	"research/deep-dive/p07b-c/00-scope-and-method.md": "# P07B-C deep dive — scope and method",
	"research/deep-dive/p07b-c/01-authority-model.md": "# Lane 1 — authority and data model",
	"research/deep-dive/p07b-c/02-runtime-security.md": "# Lane 2 — runtime, process, and scoped standalone evidence",
	"research/deep-dive/p07b-c/03-store-service.md": "# Lane 3 — store, service, and crash recovery",
	"research/deep-dive/p07b-c/04-verification.md": "# Lane 4 — verification architecture",
	"research/deep-dive/p07b-c/05-dx-wake-fit.md": "# Lane 5 — operator DX and Wake fit",
	"research/deep-dive/p07b-c/06-adversarial-design.md": "# Lane 6 — alternatives and adversarial interpretation",
	"research/deep-dive/p07b-c/07-synthesis.md": "# P07B-C synthesis",
	"research/deep-dive/p07b-c/08-different-model-red-team.md": "# Different-model red team",
	"research/deep-dive/p07b-c/09-follow-up-verification.md": "# Focused follow-up verification",
	"research/deep-dive/p07b-c/10-executive-briefing.md": "# P07B-C executive briefing",
});

const promptHeadings = Object.freeze([
	"# C0 — lock the corrected P07B-C authority contract",
	"# C1 — implement strict inert semantic algebra",
	"# C2 — add nonhead persistence and private interlock substrate",
	"# C3 — publish one exact pre-spawn target",
	"# C4 — close a CLI candidate-profile run and publish its classification",
	"# C5 — add HTTP and complete standalone-scope evidence",
	"# C6 — cumulative closure, expert surfaces, and receipts",
]);

const authorityDeclarationPath = "spec/verification/p07b-c-c0-authority.json";
const receiptDeclarationPath = "spec/verification/p07b-c-c0-receipt.json";
const c1ReceiptDeclarationPath = "spec/verification/p07b-c-c1-receipt.json";
const statusPath = "docs/status/P07B-C-C0-AUTHORITY.md";
const c1MaintenanceStatusPath = "docs/status/P07B-C-C1-CUMULATIVE-MAINTENANCE.md";
const c1VerificationStatusPath = "docs/status/P07B-C-VERIFICATION-THROUGHPUT.md";
const c1EvidenceMaintenanceStatusPath = "docs/status/P07B-C-C1-LOCAL-EVIDENCE-MAINTENANCE.md";
const c1StatusPath = "docs/status/P07B-C-C1-SEMANTICS.md";
const c1DidrunBugsPath = "docs/status/DIDRUN_BUGS.md";
const sealedC0AIdentity = Object.freeze({
	commit: "1c6d9fdef339314bfcb99d7d53f3a7c5040e5020",
	tree: "97574c4ef3f967a1194b2ebdeed246a77efabfc7",
});
const sealedC0BIdentity = Object.freeze({
	commit: "47044e95f405b7252959415adb1aa0dbea8045ab",
	tree: "d0b30300c6dd80eea95583e9b97cd819a6481f4a",
});
const sealedC1Identity = Object.freeze({
	commit: "2fceacecbacb89fd7650f1570b2af33e6ea25ed3",
	tree: "573fcd0b5548f9f7368493afe801a4bbd2cc9a34",
	subject: "feat: define P07B-C execution semantics",
});
const c0ClaimLabels = Object.freeze([
	"P07B C0 authority plan coherence",
	"P07B C0 plan checker self-test",
	"P07B C0 unit scope self-test",
	"P07B C0 inherited planning validation",
	"P07B C0 cumulative verification",
	"P07B C0 exact staged scope and diff",
	"P07B C0 scoped staged credential scan",
	"P07B C0 didrun chain intact",
]);
const c1ClaimLabels = Object.freeze([
	"P07B C1 inert semantic model",
	"P07B C1 exhaustive algebra",
	"P07B C1 schema runtime example intersection",
	"P07B C1 target parser bounded fuzz",
	"P07B C1 finalized-run parser bounded fuzz",
	"P07B C1 execution parser bounded fuzz",
	"P07B C1 architecture boundary",
	"P07B C1 architecture checker self-test",
	"P07B C1 predecessor architecture compatibility",
	"P07B C1 predecessor architecture self-test",
	"P07B C1 planning generator parity",
	"P07B C1 planning validator compatibility",
	"P07B C1 evolved plan coherence",
	"P07B C1 plan checker self-test",
	"P07B C1 cumulative verifier self-test",
	"P07B C1 exact staged scope and diff",
	"P07B C1 scoped staged credential scan",
	"P07B C1 cumulative verification",
	"P07B C1 didrun chain intact",
]);
const c1ClaimTypes = Object.freeze([
	...Array(15).fill("tests-pass"),
	"command-succeeded",
	"command-succeeded",
	"tests-pass",
	"command-succeeded",
]);
const expectedC1ReceiptEvidence = Object.freeze({
	html_report: {
		path: ".countershape/evidence/p07b-c-c1-final-2fceacecbacb.html",
		sha256: "52e09b237d2afc9f4844ceef7e171d218dee2d05fb1ed3f0b22e1e41f1f8f51d",
		bytes: 10394,
		authority: "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS",
	},
	ledger_archive: {
		path: ".didrun-history/2026-07-18-p07b-c-c1-source/.didrun/",
		authority: "LOCAL_IGNORED_ARCHIVE_NOT_GIT_AUTHORITY",
		manifest_format: "sorted-relative-posix-path-tab-size-tab-sha256-newline/v1",
		sealed_event_count: 19,
		archive_session_event_count: 20,
		claim_count: 19,
		seal_count: 1,
		object_file_count: 77,
		total_file_count: 81,
		total_bytes: 478508,
		all_files_manifest_sha256: "be365f510e8e824090529a5f710b557506f3ea795f8835818dfbe5c68066ca38",
		objects_manifest_sha256: "b72ea691de9acfd1de11bc7e1203a120222dd7b5079fdba6ae2331af336f989b",
		gitignore_sha256: "cdbcae15105d6b781e620813c79c7e868740d4e9cc53ce6f5fcbbc12387adf4b",
		session_log_sha256: "2df996d7da7821ad7bb11b46a4186473293ea5cab1d5fd5899ec8e7dc16f4752",
		claims_jsonl_sha256: "a4ce64dc38932d98e32851e77e8f5f52ef6e417a581f536cf66ccbd34bcfcc49",
		seals_jsonl_sha256: "742fc31388da2d489423f8a63b7d86653398087a55426592305b1acd343224ca",
	},
	git_note: {
		version: 1,
		blob_oid: "5d990e07d9ad6960de298bdc232818aac838700e",
		body_sha256: "c7f32d8e3ecb0f788c29c205215f9335b499e7ed50a74536fb7b06fcf8a3f9ea",
	},
	seal_redaction: {
		finding_count: 346,
		finding_kinds: ["high-entropy"],
		finding_provenance: "LOCAL_TERMINAL_OBSERVATION_NOT_GIT_NOTE",
		structured_staged_scan_findings: 0,
		structured_scan_provenance: "SEALED_C1_SUPPORTING_EVENT_16",
	},
});
const c1MaintenanceIdentity = Object.freeze({
	sourceCommit: "fa3d0c12b4c599744b666b2848e38a2499f33a89",
	sourceTree: "c908c4e580144aa481f646090a1a21d92c86e1cf",
	receiptCommit: "3f31370a70979c4c5fe3a523be19e05f84271e96",
	receiptTree: "29c5708b05c562984a0b7b66d6cfe3ac6e19c626",
});
const c1MaintenanceSourceClaims = Object.freeze([
	"P07B-C C1 maintenance exact output caps stable 50x",
	"P07B-C C1 maintenance independent limit mutation guard stable 20x",
	"P07B-C C1 maintenance simultaneous overflow lifecycle stable 20x",
	"P07B-C C1 maintenance world package",
	"P07B-C C1 maintenance cumulative verifier",
	"P07B-C C1 maintenance staged diff clean",
	"P07B-C C1 maintenance exact three-path scope",
	"P07B-C C1 maintenance exact commit structured credential scan",
]);
const c1MaintenanceReceiptClaims = Object.freeze([
	"P07B-C C1 maintenance receipt reconciliation",
	"P07B-C C1 maintenance receipt cumulative verifier",
	"P07B-C C1 maintenance receipt staged diff clean",
	"P07B-C C1 maintenance receipt exact one-path scope",
	"P07B-C C1 maintenance receipt staged credential scan",
	"P07B-C C1 maintenance receipt didrun chain intact",
]);
const expectedAuthorityDeclaration = Object.freeze({
	schema_version: "countershape/p07b-c-c0-authority/v1",
	semantic_objects: [
		{ name: "ContractExecutionTarget", storage: "IMMUTABLE_NONHEAD", production_issuer: "C3" },
		{ name: "FinalizedContractRun", storage: "IMMUTABLE_NONHEAD", production_issuer: "C4" },
		{ name: "ContractExecution", storage: "IMMUTABLE_NONHEAD", production_issuer: "C4" },
	],
	operational_objects: [
		{ name: "ExecutionInterlock", storage: "STORE_PRIVATE_MUTABLE", authority: "SPAWN_EXCLUSION_ONLY", production_acquirer: "C4" },
	],
	authority_edges: {
		storage_substrate: "C2_NO_PRODUCTION_ISSUER",
		official_target: "C3_ONLY",
		start_claim_and_run_permit: "C4_RUNNER_ONLY",
		run_permit_consumer: "internal/contractexec/runner",
		process_mechanics: "internal/processmechanics_NO_SEMANTIC_OR_PERMIT_AUTHORITY",
		reset: "EXPLICIT_OPERATOR_PLUS_DISTINCT_MEASURED_BOOT_SESSION",
	},
	scope_domains: [
		"TARGET_INVENTORY",
		"CHILD_BINDINGS",
		"IMPORT_RESOLUTION",
		"SERVICE_BINDINGS",
		"NAMED_PARENT_SECRET_SENTINEL_INHERITANCE",
	],
	scope_states: ["COMPLETE", "PARTIAL", "VIOLATED"],
	reserved_surface: "P08",
	wake_boundary: "IMMUTABLE_REF_INERT_PROVENANCE_ONLY",
	unit_order: ["C0", "C1", "C2", "C3", "C4", "C5", "C6"],
});

const requiredText = Object.freeze({
	"docs/SEMANTICS.md": [
		"For P07B-C C1, `internal/contractexec/model` is the semantic authority",
		"Tracked C1 grades are authoritative only through",
		"store-private `ExecutionInterlock`",
		"Only the C4 contract runner may acquire it from an opaque C3-issued official-target capability",
		"fresh target alone is insufficient",
		"classification-only retry and never reruns a process",
	],
	"docs/ARCHITECTURE.md": [
		"tracked C1 grades require a separately sealed C1B receipt reconciliation",
		"sole mutable operational exception to the nonhead model",
		"C1's entire production topology is `internal/contractexec/model`",
		"internal/processmechanics/",
		"private interlock may only block spawn and carries no result/discovery authority",
	],
	"docs/STATE_MACHINES.md": [
		"EXECUTION_INTERLOCK_HELD_FOR_TARGET_AND_BOOT_SESSION",
		"Missing observation is `UNKNOWN`, never `NOT_STARTED`",
		"SPAWN_OUTCOME_UNKNOWN",
		"classification-publication retry performs no spawn",
		"-> CONTRACT_EXECUTION (immutable nonhead)",
	],
	"docs/THREAT_MODEL.md": [
		"Schema-only acceptance therefore grants nothing",
		"C2 storage mechanics cannot issue this authority",
		"Low-level process mechanics never sees that permit or semantic body",
		"It establishes no host-wide absence, network/registry denial, listener ownership, containment, or confidentiality",
	],
	"docs/CONCEPT_BRIEF.md": [
		"classifier-profile-bound conformance/contradiction or ineligible conclusion",
		"C0 authority sealed; C1 owns inert canonical semantics",
		"private interlock is the sole mutable operational exception",
		"C2 is gated on a separately sealed, source-only C1B receipt reconciliation",
	],
	"docs/CLAIM_VOCABULARY.md": [
		"| `ExecutionInterlock` |",
		"serialized cooperative at-most-once spawn admission only",
		"`COMPLETE | PARTIAL | VIOLATED` scope axes",
		"immutable nonhead classifier-profile-bound conclusion",
	],
	"docs/PROMPT_PACK.md": [
		"prompts/P07B-C-TARGET-RUN-EXECUTION.md",
		"Only the higher contract runner may consume official target/admission authority",
		"C1V sealed; C1E then separate C1B receipt gate",
		"C2 still requires separately sealed/strict-clean C1B",
	],
	"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md": [
		"C0 → C1 → C2 → C3 → C4 → C5 → C6",
		"## Locked v1 semantic tables",
		"`ContractExecutionTarget`",
		"`FinalizedContractRun`",
		"`ContractExecution`",
		"`ExecutionInterlock` is one store-private, store-wide operational CAS",
		"C2 may implement/test storage mechanics but has no production issuer",
		"one opaque process-local `RunPermit`",
		"`SpawnObservation` separately records",
		"Standalone scope is a separate `COMPLETE | PARTIAL | VIOLATED` sum",
		"A bounded canonical `ClosedRunWitness`",
		"Raw/large evidence remains private behind a manifest",
		"Classifier profile v1 is exact complete-tuple membership",
		"machine-readable C0–C6 path allowlist",
		"machine-readable C0 authority declaration",
		"C6a",
		"C6b",
		"Wake integration is an immutable-ref handoff only",
		"## Combined falsification matrix",
	],
	"research/deep-dive/p07b-c/00-scope-and-method.md": [
		"six read-only specialist lanes, synthesis, different-model red team, three single-claim follow-ups, and two post-write adversarial coherence reviews",
		"restored the three-object split",
		"private boot-session `ExecutionInterlock`",
	],
	"research/deep-dive/p07b-c/01-authority-model.md": [
		"Keep three jurisdictions",
		"Process control and standalone-scope closure are separate axes",
		"prevents a fresh target from bypassing unresolved prior-survivor uncertainty",
	],
	"research/deep-dive/p07b-c/02-runtime-security.md": [
		"Go-owned process edge",
		"named parent-secret sentinel inheritance",
		"higher contract runner—not low-level process mechanics",
	],
	"research/deep-dive/p07b-c/03-store-service.md": [
		"typed, store-private publication witnesses",
		"sole mutable operational exception",
		"fresh target alone is insufficient",
	],
	"research/deep-dive/p07b-c/04-verification.md": [
		"black-box/property/parity/recovery/boundary based",
		"same-boot ambiguity blocks all later subject spawn",
		"A single “P07B-C works” claim is prohibited",
	],
	"research/deep-dive/p07b-c/05-dx-wake-fit.md": [
		"contract, candidate, pinned commit, checked fields, result",
		"Countershape does not consume Wake sessions",
	],
	"research/deep-dive/p07b-c/06-adversarial-design.md": [
		"Three persisted objects",
		"Target + run with nonpersisted view",
		"private boot-session interlock",
	],
	"research/deep-dive/p07b-c/07-synthesis.md": [
		"It was wrong to remove the immutable classification",
		"new target after ambiguity",
		"Questions deliberately handed to red team",
	],
	"research/deep-dive/p07b-c/08-different-model-red-team.md": [
		"Blocker 1 — immutable classification was lost",
		"Blocker 2 — start intent is not spawn evidence",
		"## Post-write closure correction",
	],
	"research/deep-dive/p07b-c/09-follow-up-verification.md": [
		"retain the separate object",
		"fresh target/attempt alone is not retry authority",
		"Private evidence: 16 blobs / 64 MiB aggregate per finalized run",
	],
	"research/deep-dive/p07b-c/10-executive-briefing.md": [
		"Three-object jurisdiction stays",
		"different-model critic",
		"same-boot ambiguity blocks every later subject spawn",
		"UNRECEIPTED",
	],
	"docs/HANDOFF_MODE_C.md": [
		"C1 now implements strict inert target/run/execution construction",
		"C1E local-evidence checker maintenance is the active `SOURCE_FULL` unit",
		"private boot-session interlock",
	],
	[statusPath]: [
		"Machine-readable authority summary: `spec/verification/p07b-c-c0-authority.json`.",
		"C0 implements no C runtime type",
		"C0b reconciliation commit `47044e95f405b7252959415adb1aa0dbea8045ab`",
		"C0 is complete. C1 source commit `2fceacecbacb89fd7650f1570b2af33e6ea25ed3`",
	],
	"docs/VERIFICATION.md": [
		"P07B-C C1 inert-model architecture checker",
		"Composition is one-way (`B -> C1`)",
		"eight byte-exact schema-valid/runtime-invalid cases",
		"GOFLAGS=-mod=readonly -buildvcs=false -p=1",
		"sha256:eb51c52657591df8faa5d3c4737291c071e40c07eb9c32bde88af1a36e68bf76",
		"sha256:6032cc515978947743e1bfea72340e1065d77473ccc27b3b0fc036ebd4ceb2a5",
		"Direct build, vet, and the exact general-package complement override only package jobs to `-p=2`",
		"focused Go repetition runner",
		"`RECEIPT_RECONCILIATION` is admitted only for an exact empty-prefix roster",
	],
	"spec/verification/p07b-c-unit-paths.json": [
		"countershape/p07b-c-unit-paths/v3",
		"\"C0A\"",
		"\"C1M\"",
		"\"C1V\"",
		"\"C1E\"",
		"\"C6B\"",
		"\"verification_profile\"",
		"\"receipt_claims\"",
	],
	"tools/check-p07b-c-unit-scope.mjs": [
		"P07B-C unit scope self-test passed:",
		"--staged",
		"--no-renames",
		"--ignore-submodules=none",
		"--receipt-manifest",
		"receipt path must be one staged regular mode-100644 blob",
	],
	"tools/verify-current.mjs": [
		"const goGeneralCommon = Object.freeze",
		"const goSerialCommon = Object.freeze",
		"GOFLAGS: \"-mod=readonly -buildvcs=false -p=1\"",
		"id: \"go-test-general\"",
		"id: \"go-test-sensitive-serial\"",
		"acquireVerificationLock",
		"VERIFY_STALE_LOCK",
		"verification-resource-finalization",
		"finalizeVerificationResources",
		"architecture-p07b-c-plan-selftest",
		"architecture-p07b-c-unit-scope-selftest",
	],
	"tools/verify-current-selftest.mjs": [
		"GOFLAGS: \"-mod=readonly -buildvcs=false -p=1\"",
		"VERIFY_SELFTEST_PACKAGE_EXACT_UNION",
		"VERIFY_SELFTEST_LOCK_REPLACEMENT_REMOVED",
		"go-repetition-runner-selftest",
	],
});

const controllingPaths = Object.freeze([
	"docs/SEMANTICS.md",
	"docs/ARCHITECTURE.md",
	"docs/STATE_MACHINES.md",
	"docs/THREAT_MODEL.md",
	"docs/CONCEPT_BRIEF.md",
	"docs/CLAIM_VOCABULARY.md",
	"docs/PROMPT_PACK.md",
	"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md",
	"docs/HANDOFF_MODE_C.md",
]);

const forbiddenAcrossControllingDocs = Object.freeze([
	"persist exactly two semantic truth objects",
	"ContractExecution` is a pure, deterministic, nonpersisted view",
	"exactly-once execution guarantee",
	"resume the candidate process",
]);

const forbiddenByPath = Object.freeze({
	"docs/SEMANTICS.md": [
		"Process closure owns at most one control reason",
		"confidentiality_established = false",
		"another trial allocates a fresh target and attempt",
	],
	"docs/CONCEPT_BRIEF.md": [
		"B in qualification; C future",
		"explicitly nonconfidential",
		"CONTRACT_EXECUTION_EVIDENCE",
	],
	"docs/STATE_MACHINES.md": [
		"CONTRACT_EXECUTION_EVIDENCE",
		"A retry uses a fresh target/attempt",
	],
	"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md": [
		"C6 must use explicit Go `-p=1`",
		"Run 60/80/120-column match/differ/ineligible/refused/tamper captures",
	],
});

const requiredC0APaths = Object.freeze([
	...Object.keys(requiredText),
	authorityDeclarationPath,
	"tools/check-p07b-c-plan.mjs",
]);
const requiredC1Paths = Object.freeze([
	"docs/ARCHITECTURE.md",
	"docs/CLAIM_VOCABULARY.md",
	"docs/CONCEPT_BRIEF.md",
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/SEMANTICS.md",
	"docs/STATE_MACHINES.md",
	"docs/THREAT_MODEL.md",
	"docs/VERIFICATION.md",
	"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md",
	"docs/status/P07B-C-C0-AUTHORITY.md",
	"docs/status/P07B-C-C1-CUMULATIVE-MAINTENANCE.md",
	"docs/status/P07B-C-C1-SEMANTICS.md",
	"internal/contractexec/model/algebra_test.go",
	"internal/contractexec/model/codec.go",
	"internal/contractexec/model/codec_test.go",
	"internal/contractexec/model/doc.go",
	"internal/contractexec/model/errors.go",
	"internal/contractexec/model/execution.go",
	"internal/contractexec/model/fuzz_test.go",
	"internal/contractexec/model/model_test.go",
	"internal/contractexec/model/run.go",
	"internal/contractexec/model/run_parse.go",
	"internal/contractexec/model/schema_parity_test.go",
	"internal/contractexec/model/target.go",
	"internal/contractexec/model/tuple_parse.go",
	"internal/contractexec/model/types.go",
	"spec/examples/v1/contract-execution-target.valid.json",
	"spec/examples/v1/contract-execution.valid.json",
	"spec/examples/v1/finalized-contract-run.valid.json",
	"spec/schema/v1/contract-execution-target.schema.json",
	"spec/schema/v1/contract-execution.schema.json",
	"spec/schema/v1/finalized-contract-run.schema.json",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-b-architecture-selftest.mjs",
	"tools/check-p07b-b-architecture.mjs",
	"tools/check-p07b-c-architecture-selftest.mjs",
	"tools/check-p07b-c-architecture.mjs",
	"tools/check-p07b-c-plan.mjs",
	"tools/generate-p07-planning-example.mjs",
	"tools/validate-planning.mjs",
	"tools/verify-current-selftest.mjs",
	"tools/verify-current.mjs",
]);
const requiredC1MaintenancePaths = Object.freeze([
	"docs/ARCHITECTURE.md",
	"docs/CONCEPT_BRIEF.md",
	"docs/PROMPT_PACK.md",
	"docs/SEMANTICS.md",
	"docs/STATE_MACHINES.md",
	"docs/THREAT_MODEL.md",
	"docs/VERIFICATION.md",
	"docs/decisions/0002-u2-impure-boundary.md",
	"docs/status/P07B-C-C0-AUTHORITY.md",
	"docs/status/P07B-C-C1-CUMULATIVE-MAINTENANCE.md",
	"docs/status/P07B-C-C1-SEMANTICS.md",
	"docs/status/U2.md",
	"internal/world/http_portable_negative_darwin_test.go",
	"internal/world/process_darwin.go",
	"internal/world/process_darwin_test.go",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
]);
const requiredC1MaintenanceDigest = "sha256:eb51c52657591df8faa5d3c4737291c071e40c07eb9c32bde88af1a36e68bf76";
const requiredC1MaintenanceRosterText = requiredC1MaintenancePaths
	.map((path, index) => `${index === requiredC1MaintenancePaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC1MaintenanceDeclaration = `over exactly these ${requiredC1MaintenancePaths.length} paths: ${requiredC1MaintenanceRosterText}. Their sorted-newline roster digest is \`${requiredC1MaintenanceDigest}\`.`;
const requiredC1MaintenanceText = Object.freeze({
	"docs/STATE_MACHINES.md": [
		"retrying only `(present=true, EPERM)` within its bounded sub-budget",
		"skip TERM on absence, signal only after clean presence",
		"including when a transient positive-presence `EPERM` observation resolves to absence",
	],
	"docs/THREAT_MODEL.md": [
		"only clean observed presence authorizes it",
		"persistence or every other probe error retains uncertainty and authorizes no blind signal",
		"KILL requires clean continued presence after bounded grace",
	],
	"docs/decisions/0002-u2-impure-boundary.md": [
		"current behavior from the P07B-C C1B pre-enrollment maintenance boundary",
		"does not relabel or extend the sealed historical U2 receipt",
		"retried for at most one quarter of the declared teardown budget, clipped to the overall teardown deadline",
		"persistent `EPERM` and every other probe error retain uncertainty and authorize no blind signal",
	],
	"docs/status/P07B-C-C1-CUMULATIVE-MAINTENANCE.md": [
		"## Separate C1B pre-enrollment lifecycle amendment",
		"A later absence closes cleanly without a signal; later clean presence",
		"semantic disclosure, not a self-receipt:",
	],
	"docs/status/U2.md": [
		"The rows above describe the sealed U2 commit and are not relabeled",
		"to the later maintenance boundary, not the historical U2 grade.",
	],
	"docs/VERIFICATION.md": [
		"begins no retry at or after that deadline",
		"without the old timer-controlled fd-holder closure premise",
		"This maintenance boundary is a prerequisite gate for C1B but is not C1B, cannot grade C1B",
	],
});
const requiredC1VerificationPaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/THREAT_MODEL.md",
	"docs/VERIFICATION.md",
	"docs/status/P07B-C-VERIFICATION-THROUGHPUT.md",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
	"tools/verify-current-selftest.mjs",
	"tools/verify-current.mjs",
	"tools/verify-go-test-repetition.mjs",
]);
const requiredC1VerificationDigest = "sha256:6032cc515978947743e1bfea72340e1065d77473ccc27b3b0fc036ebd4ceb2a5";
const requiredQualificationMatrixDigest = "sha256:bf5803a752ffe6c2833e7869086ec6a4919e15c5e49e8cd012a5c8c16879fc37";
const requiredQualificationCaseIDs = Object.freeze([
	"world-output-caps-50",
	"world-output-independence-20",
	"world-simultaneous-overflow-20",
	"world-lifecycle-readiness-20",
	"compiler-generated-runtime-20",
	"program-lifecycle-20",
	"store-cross-process-cas-20",
	...Array.from({ length: 20 }, (_, index) => `cli-physical-reducer-${String(index + 1).padStart(2, "0")}-of-20`),
	...Array.from({ length: 20 }, (_, index) => `http-physical-reducer-${String(index + 1).padStart(2, "0")}-of-20`),
	"parity-evaluator-20",
	"parity-framing-20",
	"parity-full-package-3",
	"cli-physical-full-package-3",
	"http-physical-full-package-3",
]);
const documentedQualificationCaseSnippets = Object.freeze([
	"world-output-caps-50",
	"world-output-independence-20",
	"world-simultaneous-overflow-20",
	"world-lifecycle-readiness-20",
	"compiler-generated-runtime-20",
	"program-lifecycle-20",
	"store-cross-process-cas-20",
	"cli-physical-reducer-01-of-20",
	"cli-physical-reducer-20-of-20",
	"http-physical-reducer-01-of-20",
	"http-physical-reducer-20-of-20",
	"parity-evaluator-20",
	"parity-framing-20",
	"parity-full-package-3",
	"cli-physical-full-package-3",
	"http-physical-full-package-3",
]);
const requiredC1VerificationRosterText = requiredC1VerificationPaths
	.map((path, index) => `${index === requiredC1VerificationPaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC1VerificationDeclaration = `C1V owns exactly these ${requiredC1VerificationPaths.length} paths: ${requiredC1VerificationRosterText}. Their sorted-newline roster digest is \`${requiredC1VerificationDigest}\`.`;
const requiredC1VerificationText = Object.freeze({
	[c1VerificationStatusPath]: [
		"`DECLINED_WITH_REASON`",
		"`DEFECT_CONDITIONAL_REPAIR`",
		"`DEFECT_REPAIR`",
		"`DEFECT_NARROW_REPAIR`",
		"GOCACHE` remains fresh",
		"C1V does not automatically delete stale O_EXCL locks",
		"`RECEIPT_RECONCILIATION` only for exact empty-prefix Markdown/receipt-declaration rosters",
		"stage-zero regular mode-`100644` nonzero blob",
		"focused Go repetition runner",
		"Any tracked edit after the freeze invalidates the whole qualifying matrix",
		"exact staged-scope/diff integrity, the scoped named-pattern credential scan, and preceding didrun-chain integrity",
		"52 ordered named executions across 14 families",
		".didrun-history/2026-07-18-p07b-c-c1v-final-attempt-1-physical-aggregate-timeout/.didrun/",
		"twenty canonical count-1 cases",
		requiredQualificationMatrixDigest,
		...documentedQualificationCaseSnippets.map((id) => `\`${id}\``),
		"This declaration does not receipt itself.",
	],
	"docs/THREAT_MODEL.md": [
		"### T17 — Verifier concurrency and cross-run cache substitution",
		"fails fast for a live or indeterminate owner",
		"`GOCACHE` stays fresh per run because current Countershape includes cgo",
	],
	"docs/VERIFICATION.md": [
		"The Go build cache deliberately remains fresh",
		"A live or indeterminate owner fails immediately with `VERIFY_ALREADY_RUNNING`",
		"`SOURCE_FULL` is mandatory for source, verifier, checker, schema, generated-artifact, or runtime changes",
		"it cannot support a cumulative, full-suite, runtime, security, or unchanged-behavior claim",
		"no C1B receipt declaration, receipt block, or reconciliation claim may be introduced until C1V seals and verifies strictly",
		"The terminal `RESULT PASS` is printed only after",
		"Any tracked change after that freeze invalidates the entire qualifying matrix",
		"A scope check from the archived development ledger is admission evidence only",
		requiredQualificationMatrixDigest,
		"Explicit package/regex mode emits `qualification=false`",
		"Every one of the three qualifying cumulative runs must finish in at most `878544 ms`",
		"There are 52 ordered named executions across 14 families",
		"C1V does not lengthen the deadline or claim that partial aggregate event",
	],
	"docs/HANDOFF_MODE_C.md": [
		"C1V commit `88e62acb3023dcd6ee51950c2ad24bbcc2ca8900`, tree `6c98b3c384f01b63d1a00a002303041576896609`",
		"10/10 claims recorded-exact",
		".didrun-history/2026-07-18-p07b-c-c1v-build-loop/.didrun/",
		".didrun-history/2026-07-18-p07b-c-c1v-physical-shard-repair/.didrun/",
		".didrun-history/2026-07-18-p07b-c-c1v-final/.didrun/",
		"event `59`",
		"secrets_override: true",
		requiredQualificationMatrixDigest,
		"878544",
		"52 named",
	],
	"docs/PROMPT_PACK.md": [
		"C1V was `SOURCE_FULL` and is sealed",
		"P07B-C C1V throughput ruling",
	],
	"tools/verify-go-test-repetition.mjs": [
		"GO_REPETITION_JSON_FRAMING",
		"GO_REPETITION_PROFILE_CLASS",
		"GO_REPETITION_PACKAGE_LIFECYCLE",
		"GO_REPETITION_PARENT_LIFECYCLE",
		"GO_REPETITION_SPECIFICATION",
		"GO_REPETITION_TEST_COUNTS",
		"expectedEntrypointPattern",
		"qualificationCases",
		"qualificationMatrixDigest",
		"GO_REPETITION_AUTHORITY",
		"runRepetition",
		"Go repetition verifier self-test passed:",
		"Go repetition verification passed:",
	],
});
const requiredC1EvidenceMaintenancePaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/VERIFICATION.md",
	"docs/status/P07B-C-C1-LOCAL-EVIDENCE-MAINTENANCE.md",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
]);
const requiredC1EvidenceMaintenanceDigest = "sha256:9a6422b45c82a44e6171ae9351468df5f6ece2190d6ab47f26aec6f84fd7ccde";
const requiredC1EvidenceMaintenanceRosterText = requiredC1EvidenceMaintenancePaths
	.map((path, index) => `${index === requiredC1EvidenceMaintenancePaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC1EvidenceMaintenanceDeclaration = `C1E owns exactly these ${requiredC1EvidenceMaintenancePaths.length} paths: ${requiredC1EvidenceMaintenanceRosterText}. Their sorted-newline roster digest is \`${requiredC1EvidenceMaintenanceDigest}\`.`;
const c1EvidenceMaintenanceClaimLabels = Object.freeze([
	"P07B-C C1E checker syntax and evolved plan coherence",
	"P07B-C C1E mixed-namespace local-evidence checker self-test",
	"P07B-C C1E sealed-C1 local archive object-namespace match",
	"P07B-C C1E unit-scope defensive self-test",
	"P07B-C C1E cumulative verification",
	"P07B-C C1E exact seven-path staged scope and diff integrity",
	"P07B-C C1E scoped staged credential-pattern scan",
	"P07B-C C1E preceding didrun chain integrity",
]);
const requiredC1EvidenceMaintenanceText = Object.freeze({
		[c1EvidenceMaintenanceStatusPath]: [
			"`DEFECT_REPAIR`",
			"event `0`",
			"event `3`",
			".didrun-history/2026-07-18-p07b-c-c1e-build-loop/.didrun/",
			"20 flat didrun capture blobs",
		"57 redirected Git loose objects",
		"77 combined object files",
			"not a portable didrun-format guarantee",
			"at most 4 MiB inflated",
			"three non-overlap files",
			"^3:spec/verification/p07b-c-c1-receipt.json",
		"C1B remains unchanged",
		"`UNRECEIPTED`",
		requiredC1EvidenceMaintenanceDeclaration,
		...c1EvidenceMaintenanceClaimLabels.map((label) => `\`${label}\``),
	],
		"docs/VERIFICATION.md": [
			"## Mixed didrun object namespace for C1 local evidence",
		"`objects/<64-lowercase-hex>`",
		"`objects/<2-lowercase-hex>/<38-lowercase-hex>`",
			"--verify-sealed-c1-local-evidence",
			"resource-bounded before content reads",
			"requires the C1B receipt declaration to be absent",
			"All three sealed qualifying runs satisfied that ceiling",
			"not hostile same-user isolation against concurrent namespace replacement",
			"never validates one receipt read and consumes a second",
		requiredC1EvidenceMaintenanceDeclaration,
	],
		"docs/HANDOFF_MODE_C.md": [
		"C1E local-evidence checker maintenance is the active `SOURCE_FULL` unit",
		"C1B's exact four-path work is preserved",
			requiredC1EvidenceMaintenanceDigest,
			"4bf73e5a34bef6cf72b491d654f423cfa317ae84e3ee66852ccb9cbf1790a5dc",
			"^3:spec/verification/p07b-c-c1-receipt.json",
			".didrun-history/2026-07-18-p07b-c-c1e-build-loop/.didrun/",
	],
		"docs/PROMPT_PACK.md": [
		"C1V sealed; C1E then separate C1B receipt gate",
			"P07B-C C1E local-evidence maintenance",
			"later independently sealed, delimited handoff/receipt descendant",
	],
	"spec/verification/p07b-c-unit-paths.json": [
		"countershape/p07b-c-unit-paths/v3",
		"\"C1E\"",
	],
		"tools/check-p07b-c-plan.mjs": [
		"CAPTURE_BLOB_SHA256",
		"GIT_LOOSE_SHA1",
		"Git loose object trailing compressed data",
			"local ledger capture blob reference set",
			"local ledger directory shape does not equal the object inventory closure",
			"must be absent in sealed-maintenance mode",
			"gitBlobCanonicalAtTotalBytes",
			"readC1ReceiptDeclarationSnapshot",
		"--verify-sealed-c1-local-evidence",
	],
	"tools/check-p07b-c-unit-scope.mjs": [
		"\"C1E\"",
	],
});
const requiredC1BPaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/status/DIDRUN_BUGS.md",
	"docs/status/P07B-C-C1-SEMANTICS.md",
	"spec/verification/p07b-c-c1-receipt.json",
]);
const requiredC1BReceiptClaims = Object.freeze([
	Object.freeze({ label: "P07B-C C1B source receipt reconciliation", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C1B receipt checker defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C1B declared local source-evidence snapshot match", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C1B receipt-only Go build", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C1B exact four-path staged scope and diff integrity", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C1B scoped staged credential-pattern scan", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C1B preceding didrun chain integrity", type: "command-succeeded" }),
]);

async function readBytes(root, path, overrides) {
	if (overrides.has(path)) {
		const value = overrides.get(path);
		return Buffer.isBuffer(value) ? value : Buffer.from(value, "utf8");
	}
	return readFile(resolve(root, path));
}

async function readText(root, path, overrides) {
	return (await readBytes(root, path, overrides)).toString("utf8");
}

function canonicalJSON(value) {
	if (value === null || typeof value === "boolean" || typeof value === "string") return JSON.stringify(value);
	if (typeof value === "number") {
		if (!Number.isSafeInteger(value) || Object.is(value, -0)) throw new Error("unsafe canonical number");
		return String(value);
	}
	if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
	if (!value || typeof value !== "object") throw new Error(`unsupported canonical value ${typeof value}`);
	const keys = Object.keys(value).sort((left, right) => Buffer.compare(Buffer.from(left), Buffer.from(right)));
	return `{${keys.map((key) => `${JSON.stringify(key)}:${canonicalJSON(value[key])}`).join(",")}}`;
}

function typedDigest(kind, body) {
	return `sha256:${createHash("sha256").update(`countershape/v1/${kind}\0`).update(canonicalJSON(body)).digest("hex")}`;
}

function closedObjectRequiredRostersExact(schema) {
	let exact = true;
	const visit = (node) => {
		if (!node || typeof node !== "object" || Array.isArray(node)) return;
		if (node.type === "object" && node.additionalProperties === false && node.properties && typeof node.properties === "object") {
			const properties = Object.keys(node.properties).sort();
			const required = Array.isArray(node.required) ? [...node.required].sort() : [];
			exact &&= new Set(required).size === required.length
				&& required.every((member) => typeof member === "string")
				&& isDeepStrictEqual(required, properties);
		}
		for (const value of Object.values(node)) {
			if (Array.isArray(value)) value.forEach(visit);
			else visit(value);
		}
	};
	visit(schema);
	return exact;
}

async function checkC1PlanningProjection(root, overrides, errors) {
	let values;
	try {
		values = Object.fromEntries(await Promise.all(Object.entries(c1PlanningPaths).map(async ([name, path]) => [
			name,
			JSON.parse(await readText(root, path, overrides)),
		])));
	} catch (error) {
		errors.push(`P07B-C C1 planning projection unreadable: ${error.message}`);
		return;
	}
	const { targetSchema, runSchema, executionSchema, bundleExample, targetExample, runExample, executionExample } = values;
	const targetRoot = [
		"schema_version", "kind", "target_version", "publication_scope", "contract_bundle_digest",
		"terminal_residue_binding", "tree_binding", "attempt_binding", "boot_session_binding", "runtime_binding",
	];
	const runRoot = [
		"schema_version", "kind", "run_version", "publication_scope", "contract_execution_target_digest",
		"attempt_artifact_digest", "start_claim_ref", "closed_run_witness",
	];
	const executionRoot = [
		"schema_version", "kind", "execution_version", "publication_scope", "classifier_profile",
		"contract_execution_target_digest", "finalized_contract_run_digest", "result",
	];
	const evidenceKinds = [
		"MATERIALIZATION_REVALIDATION", "RUNTIME_REVALIDATION", "PROCESS_RESULT", "WAIT_RESULT",
		"DRAIN_RESULT", "TEARDOWN_RESULT", "ORPHAN_CHECK", "FINALIZATION_MARKER", "CAPTURED_OBSERVATION",
		"PROJECTION_RESULT", "TARGET_INVENTORY", "CHILD_BINDINGS", "IMPORT_RESOLUTION", "SERVICE_BINDINGS",
		"NAMED_PARENT_SECRET_SENTINEL_INHERITANCE", "PRIVATE_EVIDENCE_MANIFEST",
	];
	const scopeDomains = expectedAuthorityDeclaration.scope_domains;
	const schemaProjectionValid = isDeepStrictEqual(Object.keys(targetSchema.properties ?? {}), targetRoot)
		&& isDeepStrictEqual(Object.keys(runSchema.properties ?? {}), runRoot)
		&& isDeepStrictEqual(Object.keys(executionSchema.properties ?? {}), executionRoot)
		&& closedObjectRequiredRostersExact(targetSchema)
		&& closedObjectRequiredRostersExact(runSchema)
		&& closedObjectRequiredRostersExact(executionSchema)
		&& targetSchema.additionalProperties === false
		&& runSchema.additionalProperties === false
		&& executionSchema.additionalProperties === false
		&& targetSchema.properties?.terminal_residue_binding?.properties?.current_kind?.const === "ContractBundle"
		&& targetSchema.properties?.tree_binding?.properties?.tree_identity_digest?.$ref === "common.schema.json#/$defs/Digest"
		&& targetSchema.properties?.runtime_binding?.properties?.executable_identity?.properties?.bytes_digest?.$ref
			=== "common.schema.json#/$defs/Digest"
		&& runSchema.properties?.start_claim_ref?.properties?.kind?.const === "StartClaim"
		&& runSchema.properties?.closed_run_witness?.$ref === "#/$defs/ClosedRunWitness"
		&& !Object.hasOwn(runSchema.properties ?? {}, "terminal_disposition")
		&& isDeepStrictEqual(runSchema.$defs?.EvidenceKind?.enum, evidenceKinds)
		&& isDeepStrictEqual(runSchema.$defs?.ScopeCheck?.oneOf?.[0]?.properties?.domain?.enum, scopeDomains)
		&& isDeepStrictEqual(executionSchema.properties?.result?.enum, ["CONFORMS", "CONTRADICTS", "INELIGIBLE_EXECUTION"])
		&& executionSchema.properties?.classifier_profile?.const === "CONTRACT_EXECUTION_EXACT_TUPLE_V1";
	if (!schemaProjectionValid) errors.push("P07B-C C1 planning projection mismatch: schema authority");

	let graphValid = false;
	try {
		const targetDigest = typedDigest("ContractExecutionTarget", targetExample);
		const runDigest = typedDigest("FinalizedContractRun", runExample);
		graphValid = targetExample.contract_bundle_digest === typedDigest("ContractBundle", bundleExample)
			&& targetExample.terminal_residue_binding?.current_digest === targetExample.contract_bundle_digest
			&& runExample.contract_execution_target_digest === targetDigest
			&& runExample.attempt_artifact_digest === targetExample.attempt_binding?.attempt_artifact_digest
			&& runExample.start_claim_ref?.kind === "StartClaim"
			&& executionExample.contract_execution_target_digest === targetDigest
			&& executionExample.finalized_contract_run_digest === runDigest
			&& executionExample.classifier_profile === "CONTRACT_EXECUTION_EXACT_TUPLE_V1"
			&& executionExample.result === "CONFORMS"
			&& isDeepStrictEqual(
				runExample.closed_run_witness?.standalone_scope?.checks?.map((check) => check.domain),
				scopeDomains,
			);
	} catch {
		graphValid = false;
	}
	if (!graphValid) errors.push("P07B-C C1 planning projection mismatch: example graph");
}

function checkOrderedPromptHeadings(body, errors) {
	let prior = -1;
	for (const heading of promptHeadings) {
		const positions = body.split("\n").flatMap((line, index) => line === heading ? [index] : []);
		if (positions.length !== 1) {
			errors.push(`docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md: heading must occur exactly once: ${JSON.stringify(heading)}`);
			continue;
		}
		if (positions[0] <= prior) errors.push(`docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md: heading order violation: ${JSON.stringify(heading)}`);
		prior = positions[0];
	}
}

function countOccurrences(body, snippet) {
	return body.split(snippet).length - 1;
}

function requireExactlyOnce(body, snippet, path, phase, errors) {
	const count = countOccurrences(body, snippet);
	if (count !== 1) errors.push(`${path}: ${phase} requires exactly one ${JSON.stringify(snippet)}; found ${count}`);
}

function requireClaimLabelExactlyOnce(body, label, path, phase, errors) {
	requireExactlyOnce(body, `\`${label}\``, path, `${phase} claim-label uniqueness`, errors);
}

function rejectC1BSelfReceiptClaims(body, path, errors) {
	const patterns = [
		/\bC1B (?:receipt )?commit `[0-9a-f]{40}`/iu,
		/\bC1B (?:receipt )?tree `[0-9a-f]{40}`/iu,
		/\bC1B[^\n]*(?:`?tree-exact`?|claims recorded-exact|note-present|strict exit(?:\s*:\s*|\s+)`?0`?|(?:is|was|has been) (?:sealed|strict-clean|verified)|passed verification)/iu,
	];
	for (const pattern of patterns) {
		if (pattern.test(body)) errors.push(`${path}: C1B self-receipt claim is forbidden`);
	}
}

function c1ReceiptDisclosureLines(receipt) {
	const { html_report: html, ledger_archive: ledger, git_note: note, seal_redaction: redaction } = receipt.evidence;
	return Object.freeze([
		`C1 source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`, is sealed, note-present, and strict-clean with \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\`.`,
		`The C1 seal records \`secrets_override: true\` after ${redaction.finding_count} local scanner findings, all kind \`high-entropy\`; this is not evidence of secret absence.`,
		`The separately claimed structured staged credential scan reported \`${redaction.structured_staged_scan_findings}\` findings; its provenance is sealed supporting event \`16\`, not the Git note alone.`,
		`Local ignored ledger archive \`${ledger.path}\` contains \`${ledger.sealed_event_count}\` sealed events and \`${ledger.archive_session_event_count}\` archived session events; event 20 is post-seal note reconciliation and is outside the sealed manifest.`,
		`Ledger manifests use \`${ledger.manifest_format}\`; all-files SHA-256 is \`${ledger.all_files_manifest_sha256}\` and objects-only SHA-256 is \`${ledger.objects_manifest_sha256}\`.`,
		`Archive core SHA-256 values are session \`${ledger.session_log_sha256}\`, claims \`${ledger.claims_jsonl_sha256}\`, seals \`${ledger.seals_jsonl_sha256}\`, and .gitignore \`${ledger.gitignore_sha256}\`.`,
		`The local HTML snapshot is \`${html.path}\`, SHA-256 \`${html.sha256}\`, ${html.bytes} bytes; it is not a portable strict witness.`,
		`The C1 Git note blob is \`${note.blob_oid}\` with body SHA-256 \`${note.body_sha256}\`.`,
	]);
}

const c1DidrunLiveLine = "Live S6 behavior: didrun preserved failed development receipts, chained every wrapper event, buffered the long cumulative verifier without intermediate stdout, required an explicit seal override for high-entropy-only findings, wrote the Git note, and then reported 19/19 claims recorded-exact at strict exit 0.";

function sha256(bytes) {
	return createHash("sha256").update(bytes).digest("hex");
}

function sha1(bytes) {
	return createHash("sha1").update(bytes).digest("hex");
}

const c1GitLooseInflateLimit = 4 * 1024 * 1024;
const c1ReceiptSnapshotLimit = 256 * 1024;

function gitBlobCanonicalAtTotalBytes(totalBytes) {
	if (!Number.isSafeInteger(totalBytes) || totalBytes < 8) throw new Error(`invalid Git blob size fixture: ${totalBytes}`);
	let payloadBytes = totalBytes;
	for (let attempt = 0; attempt < 8; attempt += 1) {
		const header = Buffer.from(`blob ${payloadBytes}\0`, "ascii");
		const nextPayloadBytes = totalBytes - header.length;
		if (nextPayloadBytes < 0) throw new Error(`Git blob size fixture is too small: ${totalBytes}`);
		if (nextPayloadBytes === payloadBytes) return Buffer.concat([header, Buffer.alloc(payloadBytes)], totalBytes);
		payloadBytes = nextPayloadBytes;
	}
	throw new Error(`Git blob size fixture did not converge: ${totalBytes}`);
}

function validateC1LedgerObject(object) {
	const flat = /^objects\/([0-9a-f]{64})$/u.exec(object.path);
	if (flat) {
		return object.digest === flat[1]
			? Object.freeze({ kind: "CAPTURE_BLOB_SHA256", identity: flat[1] })
			: `content-addressed capture blob mismatch: ${object.path}`;
	}

	const loose = /^objects\/([0-9a-f]{2})\/([0-9a-f]{38})$/u.exec(object.path);
	if (!loose) return `unrecognized didrun object namespace: ${object.path}`;
	const identity = `${loose[1]}${loose[2]}`;
	let inflated;
	try {
		const result = inflateSync(object.bytes, { info: true, maxOutputLength: c1GitLooseInflateLimit });
		if (!result?.engine || result.engine.bytesWritten !== object.bytes.length) {
			return `Git loose object trailing compressed data: ${object.path}`;
		}
		inflated = result.buffer;
	} catch {
		return `Git loose object bounded inflate mismatch: ${object.path}`;
	}
	if (sha1(inflated) !== identity) return `Git loose object identity mismatch: ${object.path}`;
	const separator = inflated.indexOf(0);
	if (separator <= 0) return `Git loose object header mismatch: ${object.path}`;
	const header = inflated.subarray(0, separator).toString("ascii");
	const match = /^(blob|tree|commit|tag) (0|[1-9][0-9]*)$/u.exec(header);
	if (!match || !Buffer.from(header, "ascii").equals(inflated.subarray(0, separator))) {
		return `Git loose object header mismatch: ${object.path}`;
	}
	const sizeText = match[2];
	const payloadBytes = inflated.length - separator - 1;
	if (sizeText !== String(payloadBytes)) {
		return `Git loose object declared-size mismatch: ${object.path}`;
	}
	return Object.freeze({ kind: "GIT_LOOSE_SHA1", identity });
}

function admittedLocalPath(root, declaredPath) {
	const absolute = resolve(root, declaredPath);
	const relation = relative(root, absolute);
	if (relation === "" || relation === ".." || relation.startsWith(`..${sep}`) || isAbsolute(relation)) {
		throw new Error(`local evidence path escapes repository: ${declaredPath}`);
	}
	return absolute;
}

async function admittedExistingLocalPath(root, declaredPath) {
	const absolute = admittedLocalPath(root, declaredPath);
	const rootStat = await lstat(root);
	if (rootStat.isSymbolicLink() || !rootStat.isDirectory()) throw new Error("evidence root is not a real directory");
	let cursor = root;
	for (const component of relative(root, absolute).split(sep)) {
		cursor = resolve(cursor, component);
		const status = await lstat(cursor);
		if (status.isSymbolicLink()) throw new Error(`local evidence path contains symlink: ${relative(root, cursor)}`);
	}
	const realRoot = await realpath(root);
	const realTarget = await realpath(absolute);
	const relation = relative(realRoot, realTarget);
	if (relation === ".." || relation.startsWith(`..${sep}`) || isAbsolute(relation)) {
		throw new Error(`local evidence real path escapes repository: ${declaredPath}`);
	}
	return absolute;
}

async function requireC1ReceiptDeclarationPhase(root, expectedPresent) {
	const absolute = admittedLocalPath(root, c1ReceiptDeclarationPath);
	let status;
	try {
		status = await lstat(absolute);
	} catch (error) {
		if (error.code !== "ENOENT") throw error;
		if (expectedPresent) throw new Error(`${c1ReceiptDeclarationPath} must be present in declaration-bound mode`);
		return;
	}
	if (!status.isFile() || status.isSymbolicLink()) {
		throw new Error(`${c1ReceiptDeclarationPath} must be a regular non-symlink file`);
	}
	if (!expectedPresent) throw new Error(`${c1ReceiptDeclarationPath} must be absent in sealed-maintenance mode`);
}

async function readC1ReceiptDeclarationSnapshot(root) {
	await requireC1ReceiptDeclarationPhase(root, true);
	const absolute = await admittedExistingLocalPath(root, c1ReceiptDeclarationPath);
	const bytes = await readRegularNoFollow(absolute, c1ReceiptSnapshotLimit);
	return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
}

async function readRegularNoFollow(absolute, maxBytes = Number.MAX_SAFE_INTEGER) {
	if (!Number.isSafeInteger(maxBytes) || maxBytes < 0) throw new Error(`invalid local evidence read ceiling: ${maxBytes}`);
	const handle = await open(absolute, fsConstants.O_RDONLY | fsConstants.O_NOFOLLOW);
	try {
		const before = await handle.stat();
		if (!before.isFile()) throw new Error(`local evidence is not a regular file: ${absolute}`);
		if (!Number.isSafeInteger(before.size) || before.size < 0 || before.size > maxBytes) {
			throw new Error(`local evidence file exceeds read ceiling: ${absolute}`);
		}
		const chunks = [];
		let total = 0;
		for (;;) {
			const remainingWithSentinel = maxBytes - total + 1;
			if (remainingWithSentinel <= 0) throw new Error(`local evidence file exceeds read ceiling: ${absolute}`);
			const chunk = Buffer.alloc(Math.min(64 * 1024, remainingWithSentinel));
			const { bytesRead } = await handle.read(chunk, 0, chunk.length, null);
			if (bytesRead === 0) break;
			total += bytesRead;
			if (total > maxBytes) throw new Error(`local evidence file exceeds read ceiling: ${absolute}`);
			chunks.push(chunk.subarray(0, bytesRead));
		}
		const after = await handle.stat();
		if (!after.isFile() || before.dev !== after.dev || before.ino !== after.ino || before.mode !== after.mode ||
			before.size !== after.size || before.mtimeMs !== after.mtimeMs || before.ctimeMs !== after.ctimeMs || total !== after.size) {
			throw new Error(`local evidence file changed during read: ${absolute}`);
		}
		return Buffer.concat(chunks, total);
	} finally {
		await handle.close();
	}
}

async function collectBoundedRegularFiles(root, { maxFiles, maxStoredBytes, maxDirectoryDepth, maxEntries }) {
	for (const [name, value] of Object.entries({ maxFiles, maxStoredBytes, maxDirectoryDepth, maxEntries })) {
		if (!Number.isSafeInteger(value) || value < 0) throw new Error(`invalid local evidence traversal bound ${name}: ${value}`);
	}
	const queue = [{ directory: root, depth: 0 }];
	const files = [];
	const directories = [];
	let entriesSeen = 0;
	let storedBytes = 0;
	for (let cursor = 0; cursor < queue.length; cursor += 1) {
		const { directory, depth } = queue[cursor];
		const opened = await opendir(directory);
		for await (const entry of opened) {
			entriesSeen += 1;
			if (entriesSeen > maxEntries) throw new Error("local evidence traversal entry ceiling exceeded");
			const absolute = resolve(directory, entry.name);
			const manifestPath = relative(root, absolute).split(sep).join("/");
			if (manifestPath.length === 0 || /[\\\u0000-\u001f\u007f]/u.test(manifestPath)) {
				throw new Error(`local evidence path is not serializable: ${manifestPath}`);
			}
			const status = await lstat(absolute);
			if (status.isSymbolicLink()) throw new Error(`local evidence contains symlink: ${manifestPath}`);
			if (status.isDirectory()) {
				if (depth >= maxDirectoryDepth) throw new Error(`local evidence directory depth exceeded: ${manifestPath}`);
				directories.push(absolute);
				queue.push({ directory: absolute, depth: depth + 1 });
				continue;
			}
			if (!status.isFile()) throw new Error(`local evidence contains non-regular entry: ${manifestPath}`);
			files.push(absolute);
			if (files.length > maxFiles) throw new Error("local evidence file-count ceiling exceeded");
			if (!Number.isSafeInteger(status.size) || status.size < 0 || status.size > maxStoredBytes - storedBytes) {
				throw new Error("local evidence stored-byte ceiling exceeded");
			}
			storedBytes += status.size;
		}
	}
	return { directories, files, storedBytes };
}

function validateC1LedgerDirectoryShape(root, directories, files) {
	const actual = directories
		.map((absolute) => relative(root, absolute).split(sep).join("/"))
		.sort((left, right) => Buffer.compare(Buffer.from(left, "utf8"), Buffer.from(right, "utf8")));
	const expected = new Set(["objects"]);
	for (const absolute of files) {
		const path = relative(root, absolute).split(sep).join("/");
		const loose = /^objects\/([0-9a-f]{2})\/[0-9a-f]{38}$/u.exec(path);
		if (loose) expected.add(`objects/${loose[1]}`);
	}
	const canonical = [...expected]
		.sort((left, right) => Buffer.compare(Buffer.from(left, "utf8"), Buffer.from(right, "utf8")));
	if (!isDeepStrictEqual(actual, canonical)) {
		throw new Error(`local ledger directory shape does not equal the object inventory closure: actual=${JSON.stringify(actual)} expected=${JSON.stringify(canonical)}`);
	}
}

async function collectRegularFiles(root, directory = root) {
	const entries = await readdir(directory, { withFileTypes: true });
	entries.sort((left, right) => Buffer.compare(Buffer.from(left.name, "utf8"), Buffer.from(right.name, "utf8")));
	const files = [];
	for (const entry of entries) {
		const absolute = resolve(directory, entry.name);
		const status = await lstat(absolute);
		if (status.isSymbolicLink()) throw new Error(`local evidence contains symlink: ${relative(root, absolute)}`);
		if (status.isDirectory()) files.push(...await collectRegularFiles(root, absolute));
		else if (status.isFile()) files.push(absolute);
		else throw new Error(`local evidence contains non-regular entry: ${relative(root, absolute)}`);
	}
	return files;
}

function parseJSONLines(bytes, path) {
	const text = bytes.toString("utf8");
	if (!text.endsWith("\n")) throw new Error(`${path} lacks final newline`);
	return text.slice(0, -1).split("\n").map((line, index) => {
		try {
			return JSON.parse(line);
		} catch (error) {
			throw new Error(`${path} line ${index + 1} is not JSON (${error.message})`);
		}
	});
}

function manifestDigest(entries) {
	const body = entries.map(({ path, bytes, digest }) => `${path}\t${bytes.length}\t${digest}\n`).join("");
	return sha256(Buffer.from(body, "utf8"));
}

async function verifyC1LocalEvidence(root, receipt) {
	const errors = [];
	const html = receipt.evidence.html_report;
	try {
		const bytes = await readRegularNoFollow(await admittedExistingLocalPath(root, html.path), html.bytes);
		if (bytes.length !== html.bytes) errors.push("local HTML byte count");
		if (sha256(bytes) !== html.sha256) errors.push("local HTML digest");
	} catch (error) {
		errors.push(`local HTML unavailable (${error.message})`);
	}

	const ledger = receipt.evidence.ledger_archive;
	try {
		const archiveRoot = await admittedExistingLocalPath(root, ledger.path);
		const archiveStat = await lstat(archiveRoot);
		if (!archiveStat.isDirectory() || archiveStat.isSymbolicLink()) throw new Error("archive root is not a real directory");
		const { directories, files, storedBytes } = await collectBoundedRegularFiles(archiveRoot, {
			maxFiles: ledger.total_file_count,
			maxStoredBytes: ledger.total_bytes,
			maxDirectoryDepth: 2,
			maxEntries: ledger.total_file_count + ledger.object_file_count + 8,
		});
		validateC1LedgerDirectoryShape(archiveRoot, directories, files);
		const entries = [];
		let readBytes = 0;
		for (const absolute of files) {
			const manifestPath = relative(archiveRoot, absolute).split(sep).join("/");
			if (/[/\\\u0000-\u001f\u007f]/u.test(manifestPath.replaceAll("/", ""))) {
				throw new Error(`manifest path is not serializable: ${manifestPath}`);
			}
			const bytes = await readRegularNoFollow(absolute, ledger.total_bytes - readBytes);
			readBytes += bytes.length;
			entries.push({
				path: manifestPath,
				bytes,
				digest: sha256(bytes),
			});
		}
		entries.sort((left, right) => Buffer.compare(Buffer.from(left.path, "utf8"), Buffer.from(right.path, "utf8")));
		const objects = entries.filter((entry) => entry.path.startsWith("objects/"));
		if (entries.length !== ledger.total_file_count) errors.push("local ledger total file count");
		if (objects.length !== ledger.object_file_count) errors.push("local ledger object file count");
		if (storedBytes !== ledger.total_bytes || readBytes !== ledger.total_bytes) errors.push("local ledger byte count");
		if (manifestDigest(entries) !== ledger.all_files_manifest_sha256) errors.push("local ledger all-files manifest digest");
		if (manifestDigest(objects) !== ledger.objects_manifest_sha256) errors.push("local ledger objects manifest digest");
		const captureBlobIDs = new Set();
		for (const object of objects) {
			const authority = validateC1LedgerObject(object);
			if (typeof authority === "string") errors.push(authority);
			else if (authority.kind === "CAPTURE_BLOB_SHA256") captureBlobIDs.add(authority.identity);
			else if (authority.kind !== "GIT_LOOSE_SHA1") errors.push(`unexpected local ledger object authority: ${object.path}`);
		}
		const byPath = new Map(entries.map((entry) => [entry.path, entry]));
		for (const [path, expected] of [
			["session.log", ledger.session_log_sha256],
			["claims.jsonl", ledger.claims_jsonl_sha256],
			["seals.jsonl", ledger.seals_jsonl_sha256],
			[".gitignore", ledger.gitignore_sha256],
		]) {
			if (byPath.get(path)?.digest !== expected) errors.push(`local ledger core digest: ${path}`);
		}
		const sessions = parseJSONLines(byPath.get("session.log")?.bytes ?? Buffer.alloc(0), "session.log");
		const claims = parseJSONLines(byPath.get("claims.jsonl")?.bytes ?? Buffer.alloc(0), "claims.jsonl");
		const seals = parseJSONLines(byPath.get("seals.jsonl")?.bytes ?? Buffer.alloc(0), "seals.jsonl");
		if (sessions.length !== ledger.archive_session_event_count) errors.push("local ledger session event count");
		if (claims.length !== ledger.claim_count) errors.push("local ledger claim count");
		if (seals.length !== ledger.seal_count) errors.push("local ledger seal count");
		const referencedCaptureBlobIDs = new Set();
		for (let index = 0; index < sessions.length; index += 1) {
			const entry = sessions[index];
			const expectedPrev = index === 0 ? "0".repeat(64) : sessions[index - 1]?.entry_hash;
			if (entry?.index !== index || entry?.prev_hash !== expectedPrev || !/^[0-9a-f]{64}$/u.test(entry?.entry_hash ?? "")) {
				errors.push(`local ledger session chain entry ${index}`);
			}
			for (const field of ["stdout_blob", "stderr_blob", "transcript_blob"]) {
				const digest = entry?.event?.[field];
				if (digest === null) continue;
				if (!/^[0-9a-f]{64}$/u.test(digest ?? "")) {
					errors.push(`local ledger event blob reference ${index}:${field}`);
					continue;
				}
				referencedCaptureBlobIDs.add(digest);
			}
		}
		if (!isDeepStrictEqual([...referencedCaptureBlobIDs].sort(), [...captureBlobIDs].sort())) {
			errors.push("local ledger capture blob reference set");
		}
		for (let index = 0; index < c1ClaimLabels.length; index += 1) {
			const claim = claims[index];
			if (claim?.label !== c1ClaimLabels[index] || claim?.ctype !== c1ClaimTypes[index] ||
				claim?.declared_at_index !== index || !isDeepStrictEqual(claim?.event_indices, [index]) ||
				!isDeepStrictEqual(claim?.pathspecs, [])) {
				errors.push(`local ledger claim ${index + 1}`);
			}
		}
		if (seals[0]?.commit !== receipt.source_commit || seals[0]?.tree !== receipt.source_tree ||
			seals[0]?.claims_watermark !== ledger.sealed_event_count) {
			errors.push("local ledger seal identity");
		}
	} catch (error) {
		errors.push(`local ledger unavailable (${error.message})`);
	}
	return errors;
}

async function writeJSONLines(path, values) {
	await writeFile(path, `${values.map((value) => JSON.stringify(value)).join("\n")}\n`, "utf8");
}

async function buildLocalEvidenceFixture(root) {
	const htmlPath = "evidence/report.html";
	const archivePath = "archive/.didrun/";
	const htmlAbsolute = resolve(root, htmlPath);
	const archiveRoot = resolve(root, archivePath);
	const objectsRoot = resolve(archiveRoot, "objects");
	await mkdir(resolve(root, "evidence"), { recursive: true });
	await mkdir(objectsRoot, { recursive: true });
	const htmlBytes = Buffer.from("<html><body>fixture</body></html>\n", "utf8");
	await writeFile(htmlAbsolute, htmlBytes);
	const objectBytes = Buffer.from("fixture-object\n", "utf8");
	const objectDigest = sha256(objectBytes);
	const sessions = [];
	for (let index = 0; index < 20; index += 1) {
		sessions.push({
			index,
			prev_hash: index === 0 ? "0".repeat(64) : sessions[index - 1].entry_hash,
			entry_hash: sha256(Buffer.from(`fixture-entry-${index}`, "utf8")),
			event: {
				coverage: "complete",
				exit_code: 0,
				stdout_blob: objectDigest,
				stderr_blob: objectDigest,
				transcript_blob: null,
			},
		});
	}
	const claims = c1ClaimLabels.map((label, index) => ({
		label,
		ctype: c1ClaimTypes[index],
		declared_at_index: index,
		event_indices: [index],
		pathspecs: [],
	}));
	const seals = [{
		claims_watermark: c1ClaimLabels.length,
		commit: sealedC1Identity.commit,
		tree: sealedC1Identity.tree,
	}];
	await writeJSONLines(resolve(archiveRoot, "session.log"), sessions);
	await writeJSONLines(resolve(archiveRoot, "claims.jsonl"), claims);
	await writeJSONLines(resolve(archiveRoot, "seals.jsonl"), seals);
	await writeFile(resolve(archiveRoot, ".gitignore"), "*\n!.gitignore\n", "utf8");
	const objectPath = resolve(objectsRoot, objectDigest);
	await writeFile(objectPath, objectBytes);
	const gitPayloadBytes = Buffer.from("fixture-git-payload\n", "utf8");
	const gitCanonicalBytes = Buffer.concat([
		Buffer.from(`blob ${gitPayloadBytes.length}\0`, "ascii"),
		gitPayloadBytes,
	]);
	const gitObjectIdentity = sha1(gitCanonicalBytes);
	const gitObjectBytes = deflateSync(gitCanonicalBytes);
	const gitObjectPath = resolve(objectsRoot, gitObjectIdentity.slice(0, 2), gitObjectIdentity.slice(2));
	await mkdir(dirname(gitObjectPath), { recursive: true });
	await writeFile(gitObjectPath, gitObjectBytes);
	for (const [type, payload] of [
		["tree", Buffer.alloc(0)],
		["commit", Buffer.from(`tree ${"0".repeat(40)}\nauthor Fixture <fixture@example.invalid> 0 +0000\ncommitter Fixture <fixture@example.invalid> 0 +0000\n\nfixture\n`, "ascii")],
		["tag", Buffer.from(`object ${"0".repeat(40)}\ntype blob\ntag fixture\ntagger Fixture <fixture@example.invalid> 0 +0000\n\nfixture\n`, "ascii")],
	]) {
		const canonical = Buffer.concat([Buffer.from(`${type} ${payload.length}\0`, "ascii"), payload]);
		const identity = sha1(canonical);
		const path = resolve(objectsRoot, identity.slice(0, 2), identity.slice(2));
		await mkdir(dirname(path), { recursive: true });
		await writeFile(path, deflateSync(canonical));
	}
	const files = await collectRegularFiles(archiveRoot);
	const entries = await Promise.all(files.map(async (absolute) => {
		const bytes = await readRegularNoFollow(absolute);
		return {
			path: relative(archiveRoot, absolute).split(sep).join("/"),
			bytes,
			digest: sha256(bytes),
		};
	}));
	entries.sort((left, right) => Buffer.compare(Buffer.from(left.path, "utf8"), Buffer.from(right.path, "utf8")));
	const byPath = new Map(entries.map((entry) => [entry.path, entry]));
	const objects = entries.filter((entry) => entry.path.startsWith("objects/"));
	const receipt = {
		evidence: {
			html_report: {
				path: htmlPath,
				sha256: sha256(htmlBytes),
				bytes: htmlBytes.length,
				authority: "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS",
			},
			ledger_archive: {
				path: archivePath,
				authority: "LOCAL_IGNORED_ARCHIVE_NOT_GIT_AUTHORITY",
				manifest_format: "sorted-relative-posix-path-tab-size-tab-sha256-newline/v1",
				sealed_event_count: 19,
				archive_session_event_count: sessions.length,
				claim_count: claims.length,
				seal_count: seals.length,
				object_file_count: objects.length,
				total_file_count: entries.length,
				total_bytes: entries.reduce((total, entry) => total + entry.bytes.length, 0),
				all_files_manifest_sha256: manifestDigest(entries),
				objects_manifest_sha256: manifestDigest(objects),
				gitignore_sha256: byPath.get(".gitignore").digest,
				session_log_sha256: byPath.get("session.log").digest,
				claims_jsonl_sha256: byPath.get("claims.jsonl").digest,
				seals_jsonl_sha256: byPath.get("seals.jsonl").digest,
			},
		},
		source_commit: sealedC1Identity.commit,
		source_tree: sealedC1Identity.tree,
	};
	return {
		root,
		receipt,
		archiveRoot,
		htmlAbsolute,
		htmlBytes,
		objectPath,
		objectBytes,
		gitObjectPath,
		gitObjectBytes,
		gitCanonicalBytes,
		sessionPath: resolve(archiveRoot, "session.log"),
		sessions,
		claimsPath: resolve(archiveRoot, "claims.jsonl"),
		claims,
		sealPath: resolve(archiveRoot, "seals.jsonl"),
		seals,
	};
}

async function refreshLocalEvidenceFixtureDeclaration(fixture) {
	const files = await collectRegularFiles(fixture.archiveRoot);
	const entries = await Promise.all(files.map(async (absolute) => {
		const bytes = await readRegularNoFollow(absolute);
		return {
			path: relative(fixture.archiveRoot, absolute).split(sep).join("/"),
			bytes,
			digest: sha256(bytes),
		};
	}));
	entries.sort((left, right) => Buffer.compare(Buffer.from(left.path, "utf8"), Buffer.from(right.path, "utf8")));
	const objects = entries.filter((entry) => entry.path.startsWith("objects/"));
	const byPath = new Map(entries.map((entry) => [entry.path, entry]));
	Object.assign(fixture.receipt.evidence.ledger_archive, {
		object_file_count: objects.length,
		total_file_count: entries.length,
		total_bytes: entries.reduce((total, entry) => total + entry.bytes.length, 0),
		all_files_manifest_sha256: manifestDigest(entries),
		objects_manifest_sha256: manifestDigest(objects),
		gitignore_sha256: byPath.get(".gitignore").digest,
		session_log_sha256: byPath.get("session.log").digest,
		claims_jsonl_sha256: byPath.get("claims.jsonl").digest,
		seals_jsonl_sha256: byPath.get("seals.jsonl").digest,
	});
}

async function runLocalEvidenceSelfTest() {
	const root = await mkdtemp(resolve(tmpdir(), "countershape-c1-evidence-"));
	let fixture;
	let rejected = 0;
	const expectFailure = async (name, expected) => {
		const errors = await verifyC1LocalEvidence(fixture.root, fixture.receipt);
		if (!errors.some((error) => error.includes(expected))) {
			throw new Error(`P07B-C C1 local-evidence self-test false negative: ${name} (${errors.join("; ")})`);
		}
		rejected += 1;
	};
	const requireCleanBaseline = async (name) => {
		const errors = await verifyC1LocalEvidence(fixture.root, fixture.receipt);
		if (errors.length > 0) throw new Error(`P07B-C C1 local-evidence self-test dirty restore after ${name}: ${errors.join(", ")}`);
	};
	try {
		fixture = await buildLocalEvidenceFixture(root);
		const baseline = await verifyC1LocalEvidence(fixture.root, fixture.receipt);
		if (baseline.length > 0) throw new Error(`P07B-C C1 local-evidence self-test baseline failed: ${baseline.join(", ")}`);
		let activeGitObjectPath = fixture.gitObjectPath;
		const originalGitIdentity = sha1(fixture.gitCanonicalBytes);
		const removeActiveGitObject = async () => {
			const previousParent = dirname(activeGitObjectPath);
			await rm(activeGitObjectPath, { force: true });
			if (previousParent !== resolve(fixture.archiveRoot, "objects")) {
				try {
					await rmdir(previousParent);
				} catch (error) {
					if (!new Set(["ENOENT", "ENOTEMPTY", "EEXIST"]).has(error.code)) throw error;
				}
			}
		};
		const installGitObject = async (identity, bytes) => {
			await removeActiveGitObject();
			activeGitObjectPath = resolve(fixture.archiveRoot, "objects", identity.slice(0, 2), identity.slice(2));
			await mkdir(dirname(activeGitObjectPath), { recursive: true });
			await writeFile(activeGitObjectPath, bytes);
			await refreshLocalEvidenceFixtureDeclaration(fixture);
		};
		const restoreGitObject = async () => installGitObject(originalGitIdentity, fixture.gitObjectBytes);

		await writeFile(fixture.htmlAbsolute, Buffer.from("tampered-html\n", "utf8"));
		await expectFailure("HTML bytes", "local HTML");
		await writeFile(fixture.htmlAbsolute, fixture.htmlBytes);

		await rm(fixture.htmlAbsolute);
		await symlink(resolve(fixture.root, "archive"), fixture.htmlAbsolute);
		await expectFailure("HTML symlink", "symlink");
		await rm(fixture.htmlAbsolute);
		await writeFile(fixture.htmlAbsolute, fixture.htmlBytes);

		const extraLedgerFile = resolve(fixture.archiveRoot, "extra-file");
		await writeFile(extraLedgerFile, Buffer.alloc(0));
		await expectFailure("bounded file count", "file-count ceiling exceeded");
		await rm(extraLedgerFile);

		await writeFile(fixture.objectPath, Buffer.alloc(fixture.receipt.evidence.ledger_archive.total_bytes + 1, 0x61));
		await expectFailure("bounded stored bytes", "stored-byte ceiling exceeded");
		await writeFile(fixture.objectPath, fixture.objectBytes);

		const deepRoot = resolve(fixture.archiveRoot, "deep", "one", "two");
		await mkdir(deepRoot, { recursive: true });
		await writeFile(resolve(deepRoot, "file"), Buffer.alloc(0));
		await expectFailure("bounded directory depth", "directory depth exceeded");
		await rm(resolve(fixture.archiveRoot, "deep"), { recursive: true, force: true });
		await requireCleanBaseline("bounded directory depth");

		for (const name of ["pack", "info"]) {
			const unexpectedDirectory = resolve(fixture.archiveRoot, "objects", name);
			await mkdir(unexpectedDirectory);
			await expectFailure(`empty objects/${name} directory`, "directory shape");
			await rm(unexpectedDirectory, { recursive: true, force: true });
			await requireCleanBaseline(`empty objects/${name} directory`);
		}

		const extraDepthDirectory = resolve(fixture.archiveRoot, "objects", "FF", "extra");
		await mkdir(extraDepthDirectory, { recursive: true });
		await expectFailure("empty extra-depth object directory", "directory depth exceeded");
		await rm(resolve(fixture.archiveRoot, "objects", "FF"), { recursive: true, force: true });
		await requireCleanBaseline("empty extra-depth object directory");

		await writeFile(fixture.objectPath, Buffer.from("tampered-object\n", "utf8"));
		await refreshLocalEvidenceFixtureDeclaration(fixture);
		await expectFailure("content-addressed capture blob", "content-addressed capture blob mismatch");
		await writeFile(fixture.objectPath, fixture.objectBytes);
		await refreshLocalEvidenceFixtureDeclaration(fixture);

		const alternateGitPayload = Buffer.from("different-valid-git-object\n", "utf8");
		const alternateGitCanonical = Buffer.concat([
			Buffer.from(`blob ${alternateGitPayload.length}\0`, "ascii"), alternateGitPayload,
		]);
		await installGitObject(originalGitIdentity, deflateSync(alternateGitCanonical));
		await expectFailure("Git loose identity", "Git loose object identity mismatch");
		await restoreGitObject();

		await installGitObject(originalGitIdentity, Buffer.from("not-zlib", "ascii"));
		await expectFailure("Git loose compression", "Git loose object bounded inflate mismatch");
		await restoreGitObject();

		await installGitObject(originalGitIdentity, Buffer.concat([fixture.gitObjectBytes, Buffer.from([0, 1, 2])]));
		await expectFailure("Git loose trailing bytes", "Git loose object trailing compressed data");
		await restoreGitObject();

		const noncanonicalHeader = Buffer.from("blob 01\0x", "ascii");
		await installGitObject(sha1(noncanonicalHeader), deflateSync(noncanonicalHeader));
		await expectFailure("Git loose noncanonical header", "Git loose object header mismatch");
		await restoreGitObject();

		const wrongDeclaredSize = Buffer.from("blob 2\0x", "ascii");
		await installGitObject(sha1(wrongDeclaredSize), deflateSync(wrongDeclaredSize));
		await expectFailure("Git loose declared size", "Git loose object declared-size mismatch");
		await restoreGitObject();

		await removeActiveGitObject();
		const unknownObjectPath = resolve(fixture.archiveRoot, "objects", "unexpected-object");
		await writeFile(unknownObjectPath, fixture.gitObjectBytes);
		activeGitObjectPath = unknownObjectPath;
		await refreshLocalEvidenceFixtureDeclaration(fixture);
		await expectFailure("unrecognized object namespace", "unrecognized didrun object namespace");
		await restoreGitObject();

		for (const totalBytes of [c1GitLooseInflateLimit - 1, c1GitLooseInflateLimit]) {
			const canonical = gitBlobCanonicalAtTotalBytes(totalBytes);
			const identity = sha1(canonical);
			const bytes = deflateSync(canonical);
			const accepted = validateC1LedgerObject({
				path: `objects/${identity.slice(0, 2)}/${identity.slice(2)}`,
				bytes,
				digest: sha256(bytes),
			});
			if (typeof accepted === "string" || accepted.kind !== "GIT_LOOSE_SHA1") {
				throw new Error(`P07B-C C1 local-evidence self-test rejected ${totalBytes}-byte inflate boundary`);
			}
		}
		const oversizedCanonical = gitBlobCanonicalAtTotalBytes(c1GitLooseInflateLimit + 1);
		const oversizedIdentity = sha1(oversizedCanonical);
		const oversizedBytes = deflateSync(oversizedCanonical);
		const oversizedResult = validateC1LedgerObject({
			path: `objects/${oversizedIdentity.slice(0, 2)}/${oversizedIdentity.slice(2)}`,
			bytes: oversizedBytes,
			digest: sha256(oversizedBytes),
		});
		if (typeof oversizedResult !== "string" || !oversizedResult.includes("bounded inflate mismatch")) {
			throw new Error("P07B-C C1 local-evidence self-test accepted over-limit Git loose object");
		}
		rejected += 1;

		const extraBlobBytes = Buffer.from("unreferenced-capture-blob\n", "utf8");
		const extraBlobPath = resolve(fixture.archiveRoot, "objects", sha256(extraBlobBytes));
		await writeFile(extraBlobPath, extraBlobBytes);
		await refreshLocalEvidenceFixtureDeclaration(fixture);
		await expectFailure("unreferenced capture blob", "local ledger capture blob reference set");
		await rm(extraBlobPath);
		await refreshLocalEvidenceFixtureDeclaration(fixture);

		const missingReferenceSessions = structuredClone(fixture.sessions);
		missingReferenceSessions[0].event.stdout_blob = "f".repeat(64);
		await writeJSONLines(fixture.sessionPath, missingReferenceSessions);
		await refreshLocalEvidenceFixtureDeclaration(fixture);
		await expectFailure("missing capture blob reference", "local ledger capture blob reference set");
		await writeJSONLines(fixture.sessionPath, fixture.sessions);
		await refreshLocalEvidenceFixtureDeclaration(fixture);

		const mutateLedgerDeclaration = async (name, field, value, expected) => {
			const ledger = fixture.receipt.evidence.ledger_archive;
			const original = ledger[field];
			ledger[field] = value;
			await expectFailure(name, expected);
			ledger[field] = original;
		};
		await mutateLedgerDeclaration("all-files manifest digest", "all_files_manifest_sha256", "f".repeat(64), "all-files manifest digest");
		await mutateLedgerDeclaration("objects manifest digest", "objects_manifest_sha256", "f".repeat(64), "objects manifest digest");
		await mutateLedgerDeclaration("total file count", "total_file_count", fixture.receipt.evidence.ledger_archive.total_file_count + 1, "total file count");
		await mutateLedgerDeclaration("object file count", "object_file_count", fixture.receipt.evidence.ledger_archive.object_file_count + 1, "object file count");
		await mutateLedgerDeclaration("total byte count", "total_bytes", fixture.receipt.evidence.ledger_archive.total_bytes + 1, "byte count");
		for (const [field, path] of [
			["gitignore_sha256", ".gitignore"],
			["session_log_sha256", "session.log"],
			["claims_jsonl_sha256", "claims.jsonl"],
			["seals_jsonl_sha256", "seals.jsonl"],
		]) {
			await mutateLedgerDeclaration(`${path} core digest`, field, "f".repeat(64), `core digest: ${path}`);
		}
		await mutateLedgerDeclaration("session event count", "archive_session_event_count", fixture.receipt.evidence.ledger_archive.archive_session_event_count + 1, "session event count");
		await mutateLedgerDeclaration("claim count", "claim_count", fixture.receipt.evidence.ledger_archive.claim_count + 1, "claim count");
		await mutateLedgerDeclaration("seal count", "seal_count", fixture.receipt.evidence.ledger_archive.seal_count + 1, "seal count");
		await mutateLedgerDeclaration("seal watermark", "sealed_event_count", fixture.receipt.evidence.ledger_archive.sealed_event_count + 1, "seal identity");

		const hostileClaims = structuredClone(fixture.claims);
		hostileClaims[0].label = "wrong label";
		await writeJSONLines(fixture.claimsPath, hostileClaims);
		await expectFailure("claim roster", "local ledger claim 1");
		await writeJSONLines(fixture.claimsPath, fixture.claims);

		const hostileSessions = structuredClone(fixture.sessions);
		hostileSessions[1].prev_hash = "f".repeat(64);
		await writeJSONLines(fixture.sessionPath, hostileSessions);
		await expectFailure("session linkage", "session chain entry 1");
		await writeJSONLines(fixture.sessionPath, fixture.sessions);

		const hostileSeals = structuredClone(fixture.seals);
		hostileSeals[0].commit = "f".repeat(40);
		await writeJSONLines(fixture.sealPath, hostileSeals);
		await expectFailure("seal identity", "local ledger seal identity");
		await writeJSONLines(fixture.sealPath, fixture.seals);
		await requireCleanBaseline("final hostile case");
		return rejected;
	} finally {
		await rm(root, { recursive: true, force: true });
	}
}

async function runC1ReceiptModePhaseSelfTest() {
	const root = await mkdtemp(resolve(tmpdir(), "countershape-c1-receipt-phase-"));
	const absolute = resolve(root, c1ReceiptDeclarationPath);
	let rejected = 0;
	const expectFailure = async (name, expectedPresent, expected) => {
		try {
			await requireC1ReceiptDeclarationPhase(root, expectedPresent);
		} catch (error) {
			if (!error.message.includes(expected)) {
				throw new Error(`P07B-C C1 receipt-mode phase self-test wrong rejection for ${name}: ${error.message}`);
			}
			rejected += 1;
			return;
		}
		throw new Error(`P07B-C C1 receipt-mode phase self-test false negative: ${name}`);
	};
	try {
		await requireC1ReceiptDeclarationPhase(root, false);
		await expectFailure("declaration-bound mode without declaration", true, "must be present");
		await mkdir(dirname(absolute), { recursive: true });
		await writeFile(absolute, "{}\n", "utf8");
		await requireC1ReceiptDeclarationPhase(root, true);
		if (await readC1ReceiptDeclarationSnapshot(root) !== "{}\n") {
			throw new Error("P07B-C C1 receipt-mode phase self-test snapshot mismatch");
		}
		await expectFailure("sealed-maintenance mode with declaration", false, "must be absent");
		await rm(absolute);
		await symlink(resolve(root, "missing-target"), absolute);
		await expectFailure("declaration-bound symlink", true, "regular non-symlink");
		return rejected;
	} finally {
		await rm(root, { recursive: true, force: true });
	}
}

function validateReceiptDeclaration(receipt) {
	const errors = [];
	const keys = [
		"claims", "schema_version", "source_commit", "source_tree", "strict_claims_recorded_exact",
		"strict_claims_total", "strict_exit_code",
	];
	if (!receipt || typeof receipt !== "object" || Array.isArray(receipt) ||
		!isDeepStrictEqual(Object.keys(receipt).sort(), keys)) {
		return ["root field roster"];
	}
	if (receipt.schema_version !== "countershape/p07b-c-c0-receipt/v1") errors.push("schema version");
	if (!/^[0-9a-f]{40}$/u.test(receipt.source_commit ?? "")) errors.push("source commit");
	if (!/^[0-9a-f]{40}$/u.test(receipt.source_tree ?? "")) errors.push("source tree");
	if (receipt.strict_exit_code !== 0) errors.push("strict exit code");
	if (receipt.strict_claims_recorded_exact !== c0ClaimLabels.length || receipt.strict_claims_total !== c0ClaimLabels.length) {
		errors.push("strict claim counts");
	}
	if (!Array.isArray(receipt.claims) || receipt.claims.length !== c0ClaimLabels.length) {
		errors.push("claim roster length");
	} else {
		for (let index = 0; index < c0ClaimLabels.length; index += 1) {
			const claim = receipt.claims[index];
			if (!claim || typeof claim !== "object" || Array.isArray(claim) ||
				!isDeepStrictEqual(Object.keys(claim).sort(), ["grade", "label"]) ||
				claim.label !== c0ClaimLabels[index] || claim.grade !== "TREE-EXACT") {
				errors.push(`claim ${index + 1}`);
			}
		}
	}
	return errors;
}

function validateReceiptAuthority(receipt, authority) {
	const errors = [];
	if (!authority || typeof authority !== "object" || Array.isArray(authority)) return ["missing Git/didrun authority"];
	if (authority.commit !== receipt.source_commit) errors.push("source commit does not equal admitted Git commit");
	if (authority.tree !== receipt.source_tree) errors.push("source tree does not equal Git commit tree");
	if (authority.subject !== "docs: lock P07B-C execution authority") errors.push("source commit subject");
	if (authority.ancestor_of_head !== true) errors.push("source commit is not an ancestor of HEAD");
	const note = authority.note;
	if (!note || typeof note !== "object" || Array.isArray(note)) {
		errors.push("missing parsed didrun Git note");
		return errors;
	}
	if (note.commit !== receipt.source_commit) errors.push("didrun note commit");
	if (note.tree !== receipt.source_tree) errors.push("didrun note tree");
	if (!Array.isArray(note.claims) || note.claims.length !== c0ClaimLabels.length) {
		errors.push("didrun note claim roster length");
	} else {
		for (let index = 0; index < c0ClaimLabels.length; index += 1) {
			const recorded = note.claims[index];
			if (recorded?.claim?.label !== c0ClaimLabels[index] || recorded?.grade !== "tree-exact" || recorded?.exit_code !== 0) {
				errors.push(`didrun note claim ${index + 1}`);
			}
		}
	}
	if (note.coverage?.total_events !== c0ClaimLabels.length || note.coverage?.by_coverage?.complete !== c0ClaimLabels.length) {
		errors.push("didrun note exact event coverage");
	}
	return errors;
}

async function loadReceiptAuthorityFromGit(root, receipt) {
	const requested = process.env.COUNTERSHAPE_GIT || "/usr/bin/git";
	if (!isAbsolute(requested)) throw new Error("COUNTERSHAPE_GIT must be absolute");
	const git = await realpath(requested);
	const env = {
		HOME: process.env.HOME || "/",
		PATH: "/usr/bin:/bin",
		LANG: "C",
		LC_ALL: "C",
		NO_COLOR: "1",
		GIT_CONFIG_NOSYSTEM: "1",
		GIT_CONFIG_GLOBAL: "/dev/null",
		GIT_NO_LAZY_FETCH: "1",
		GIT_OPTIONAL_LOCKS: "0",
		GIT_TERMINAL_PROMPT: "0",
	};
	const run = (args, accepted = [0]) => {
		const result = spawnSync(git, ["--no-replace-objects", ...args], {
			cwd: root, encoding: "utf8", timeout: 30_000, maxBuffer: 4 * 1024 * 1024, env,
		});
		if (result.error || result.signal || !accepted.includes(result.status) || (result.stderr?.length ?? 0) !== 0) {
			throw new Error(`git ${args[0]} failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
		}
		return { status: result.status, stdout: result.stdout ?? "" };
	};
	const commit = run(["rev-parse", "--verify", `${receipt.source_commit}^{commit}`]).stdout.trim();
	if (commit !== receipt.source_commit) throw new Error("source commit did not reopen exactly");
	const tree = run(["rev-parse", "--verify", `${receipt.source_commit}^{tree}`]).stdout.trim();
	const parent = run(["rev-parse", "--verify", `${receipt.source_commit}^`]).stdout.trim();
	const subject = run(["show", "-s", "--format=%s", receipt.source_commit]).stdout.trimEnd();
	const ancestor = run(["merge-base", "--is-ancestor", receipt.source_commit, "HEAD"], [0, 1]).status === 0;
	const noteText = run(["notes", "--ref=didrun", "show", receipt.source_commit]).stdout;
	const noteListing = run(["notes", "--ref=didrun", "list", receipt.source_commit]).stdout.trim();
	if (!/^[0-9a-f]{40}$/u.test(noteListing)) {
		throw new Error("didrun Git note object identity is malformed");
	}
	let note;
	try {
		note = JSON.parse(noteText);
	} catch (error) {
		throw new Error(`didrun Git note is not JSON (${error.message})`);
	}
	return {
		commit,
		tree,
		parent,
		subject,
		ancestor_of_head: ancestor,
		note,
		note_blob_oid: noteListing,
		note_body_sha256: createHash("sha256").update(noteText, "utf8").digest("hex"),
	};
}

function validateC1ReceiptDeclaration(receipt) {
	const errors = [];
	const keys = [
		"claims", "coverage_complete_events", "coverage_total_events", "evidence", "schema_version",
		"secrets_override", "source_commit", "source_subject", "source_tree", "strict_claims_recorded_exact",
		"strict_claims_total", "strict_exit_code",
	];
	if (!receipt || typeof receipt !== "object" || Array.isArray(receipt) ||
		!isDeepStrictEqual(Object.keys(receipt).sort(), keys)) return ["root field roster"];
	if (receipt.schema_version !== "countershape/p07b-c-c1-receipt/v1") errors.push("schema version");
	if (receipt.source_commit !== sealedC1Identity.commit) errors.push("source commit");
	if (receipt.source_tree !== sealedC1Identity.tree) errors.push("source tree");
	if (receipt.source_subject !== sealedC1Identity.subject) errors.push("source subject");
	if (receipt.strict_exit_code !== 0) errors.push("strict exit code");
	if (receipt.strict_claims_recorded_exact !== c1ClaimLabels.length || receipt.strict_claims_total !== c1ClaimLabels.length) {
		errors.push("strict claim counts");
	}
	if (receipt.coverage_total_events !== c1ClaimLabels.length || receipt.coverage_complete_events !== c1ClaimLabels.length) {
		errors.push("coverage counts");
	}
	if (receipt.secrets_override !== true) errors.push("redacted seal disclosure");
	if (!Array.isArray(receipt.claims) || receipt.claims.length !== c1ClaimLabels.length) {
		errors.push("claim roster length");
	} else {
		for (let index = 0; index < c1ClaimLabels.length; index += 1) {
			const claim = receipt.claims[index];
			if (!claim || typeof claim !== "object" || Array.isArray(claim) ||
				!isDeepStrictEqual(Object.keys(claim).sort(), ["grade", "label", "supporting_event_index", "type"]) ||
				claim.supporting_event_index !== index || claim.type !== c1ClaimTypes[index] ||
				claim.label !== c1ClaimLabels[index] || claim.grade !== "TREE-EXACT") {
				errors.push(`claim ${index + 1}`);
			}
		}
	}
	if (!isDeepStrictEqual(receipt.evidence, expectedC1ReceiptEvidence)) errors.push("local evidence declaration");
	return errors;
}

function validateC1ReceiptAuthority(receipt, authority) {
	const errors = [];
	if (!authority || typeof authority !== "object" || Array.isArray(authority)) return ["missing Git/didrun authority"];
	if (authority.commit !== receipt.source_commit) errors.push("source commit does not equal admitted Git commit");
	if (authority.tree !== receipt.source_tree) errors.push("source tree does not equal Git commit tree");
	if (authority.parent !== c1MaintenanceIdentity.receiptCommit) errors.push("source parent");
	if (authority.subject !== receipt.source_subject) errors.push("source commit subject");
	if (authority.ancestor_of_head !== true) errors.push("source commit is not an ancestor of HEAD");
	if (authority.note_blob_oid !== receipt.evidence.git_note.blob_oid) errors.push("didrun note object identity");
	if (authority.note_body_sha256 !== receipt.evidence.git_note.body_sha256) errors.push("didrun note body digest");
	const note = authority.note;
	if (!note || typeof note !== "object" || Array.isArray(note)) {
		errors.push("missing parsed didrun Git note");
		return errors;
	}
	if (note.version !== receipt.evidence.git_note.version) errors.push("didrun note version");
	if (note.commit !== receipt.source_commit) errors.push("didrun note commit");
	if (note.tree !== receipt.source_tree) errors.push("didrun note tree");
	if (note.secrets_override !== receipt.secrets_override) errors.push("didrun note redacted seal disclosure");
	if (!Array.isArray(note.claims) || note.claims.length !== c1ClaimLabels.length) {
		errors.push("didrun note claim roster length");
	} else {
		for (let index = 0; index < c1ClaimLabels.length; index += 1) {
			const recorded = note.claims[index];
			const claim = recorded?.claim;
			if (claim?.label !== c1ClaimLabels[index] || claim?.ctype !== c1ClaimTypes[index] ||
				claim?.declared_at_index !== index || !isDeepStrictEqual(claim?.event_indices, [index]) ||
				!isDeepStrictEqual(claim?.pathspecs, []) || !Array.isArray(claim?.argv_preview) || claim.argv_preview.length === 0 ||
				recorded?.supporting_event_index !== index || recorded?.grade !== "tree-exact" || recorded?.exit_code !== 0 ||
				recorded?.reason !== "self-stable command ran against the sealed tree" || !isDeepStrictEqual(recorded?.delta, [])) {
				errors.push(`didrun note claim ${index + 1}`);
			}
		}
	}
	if (note.coverage?.total_events !== receipt.coverage_total_events ||
		note.coverage?.by_coverage?.complete !== receipt.coverage_complete_events ||
		!isDeepStrictEqual(Object.keys(note.coverage?.by_coverage ?? {}), ["complete"])) {
		errors.push("didrun note exact event coverage");
	}
	return errors;
}

async function checkC1ReceiptPhase(root, overrides, status, handoff, errors, injectedReceiptAuthority) {
	let receiptText;
	let didrunBugs;
	try {
		receiptText = await readText(root, c1ReceiptDeclarationPath, overrides);
	} catch (error) {
		if (error.code !== "ENOENT") errors.push(`${c1ReceiptDeclarationPath}: unreadable (${error.message})`);
	}
	try {
		didrunBugs = await readText(root, c1DidrunBugsPath, overrides);
	} catch (error) {
		errors.push(`${c1DidrunBugsPath}: unreadable (${error.message})`);
	}

	if (receiptText === undefined) {
		const sealedPendingState = `**Source state:** sealed C1 source commit \`${sealedC1Identity.commit}\`, tree \`${sealedC1Identity.tree}\`, is note-present and strict-clean with \`19/19\` claims. Tracked C1 receipt reconciliation is pending; every C1 capability below remains \`UNRECEIPTED\` until the separately sealed C1B receipt boundary binds the source note, status, and handoff without grading itself.`;
		requireExactlyOnce(
			status,
			sealedPendingState,
			c1StatusPath,
			"C1 sealed-source pending-receipt state",
			errors,
		);
		for (const label of c1ClaimLabels) {
			requireExactlyOnce(status, `\`${label}\` | \`UNRECEIPTED\``, c1StatusPath, "C1 pending source receipt map", errors);
			requireClaimLabelExactlyOnce(status, label, c1StatusPath, "C1 pending source receipt map", errors);
		}
		if (handoff.includes("<!-- P07B-C-C1-SOURCE-RECEIPTS:START -->") ||
			handoff.includes("<!-- P07B-C-C1-SOURCE-RECEIPTS:END -->")) {
			errors.push(`docs/HANDOFF_MODE_C.md: C1 pending receipt state cannot contain a source receipt block`);
		}
		if (didrunBugs?.includes("<!-- P07B-C-C1-DIDRUN-LIVE:START -->") ||
			didrunBugs?.includes("<!-- P07B-C-C1-DIDRUN-LIVE:END -->")) {
			errors.push(`${c1DidrunBugsPath}: C1 pending receipt state cannot contain a didrun live-session block`);
		}
		rejectC1BSelfReceiptClaims(status, c1StatusPath, errors);
		rejectC1BSelfReceiptClaims(handoff, "docs/HANDOFF_MODE_C.md", errors);
		if (didrunBugs !== undefined) rejectC1BSelfReceiptClaims(didrunBugs, c1DidrunBugsPath, errors);
		return;
	}

	let receipt;
	try {
		receipt = JSON.parse(receiptText);
	} catch (error) {
		errors.push(`${c1ReceiptDeclarationPath}: invalid JSON (${error.message})`);
		return;
	}
	const receiptErrors = validateC1ReceiptDeclaration(receipt);
	for (const error of receiptErrors) errors.push(`${c1ReceiptDeclarationPath}: invalid receipt declaration (${error})`);
	if (receiptErrors.length > 0) return;
	let authority = injectedReceiptAuthority;
	if (authority === undefined) {
		try {
			authority = await loadReceiptAuthorityFromGit(root, receipt);
		} catch (error) {
			errors.push(`${c1ReceiptDeclarationPath}: Git/didrun authority unavailable (${error.message})`);
			return;
		}
	}
	const authorityErrors = validateC1ReceiptAuthority(receipt, authority);
	for (const error of authorityErrors) errors.push(`${c1ReceiptDeclarationPath}: Git/didrun authority mismatch (${error})`);
	if (authorityErrors.length > 0) return;

	const state = `**Source receipt:** C1 source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`, is sealed, note-present, and strict-clean; every source grade below is \`TREE-EXACT\`. This C1B receipt-document working unit binds only that existing source and remains \`UNRECEIPTED\` until its own commit, seal, note, and strict boundary.`;
	requireExactlyOnce(status, state, c1StatusPath, "C1B reconciled source state", errors);
	requireExactlyOnce(status, "## C1 source receipt map", c1StatusPath, "C1B receipt heading", errors);
	requireExactlyOnce(status, "- C1 source strict exit: `0`", c1StatusPath, "C1B strict result", errors);
	requireExactlyOnce(status, `- C1 source strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\``, c1StatusPath, "C1B strict claim count", errors);
	requireExactlyOnce(status, "This receipt binds only the already-existing C1 source commit. C1B cannot name or grade its own commit, tree, Git note, or strict result.", c1StatusPath, "C1B no-recursion boundary", errors);
	for (const claim of receipt.claims) {
		requireExactlyOnce(status, `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`, c1StatusPath, "C1B source receipt map", errors);
		requireClaimLabelExactlyOnce(status, claim.label, c1StatusPath, "C1B source receipt map", errors);
	}
	const disclosures = c1ReceiptDisclosureLines(receipt);
	for (const index of [1, 2, 3, 4, 6]) {
		requireExactlyOnce(status, disclosures[index], c1StatusPath, "C1B source evidence disclosure", errors);
	}
	requireExactlyOnce(handoff, "<!-- P07B-C-C1-SOURCE-RECEIPTS:START -->", "docs/HANDOFF_MODE_C.md", "C1B receipt block", errors);
	requireExactlyOnce(handoff, "<!-- P07B-C-C1-SOURCE-RECEIPTS:END -->", "docs/HANDOFF_MODE_C.md", "C1B receipt block", errors);
	requireExactlyOnce(handoff, `C1 source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`.`, "docs/HANDOFF_MODE_C.md", "C1B source identity", errors);
	requireExactlyOnce(handoff, `C1 strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\`; strict exit: \`0\`.`, "docs/HANDOFF_MODE_C.md", "C1B strict result", errors);
	for (const claim of receipt.claims) {
		requireExactlyOnce(handoff, `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`, "docs/HANDOFF_MODE_C.md", "C1B claim map", errors);
		requireClaimLabelExactlyOnce(handoff, claim.label, "docs/HANDOFF_MODE_C.md", "C1B claim map", errors);
	}
	for (const disclosure of disclosures) {
		requireExactlyOnce(handoff, disclosure, "docs/HANDOFF_MODE_C.md", "C1B source evidence disclosure", errors);
	}
	if (didrunBugs !== undefined) {
		requireExactlyOnce(didrunBugs, "<!-- P07B-C-C1-DIDRUN-LIVE:START -->", c1DidrunBugsPath, "C1B didrun live-session block", errors);
		requireExactlyOnce(didrunBugs, "<!-- P07B-C-C1-DIDRUN-LIVE:END -->", c1DidrunBugsPath, "C1B didrun live-session block", errors);
		for (const disclosure of disclosures) {
			requireExactlyOnce(didrunBugs, disclosure, c1DidrunBugsPath, "C1B didrun source disclosure", errors);
		}
		requireExactlyOnce(didrunBugs, c1DidrunLiveLine, c1DidrunBugsPath, "C1B didrun live-session behavior", errors);
	}
	rejectC1BSelfReceiptClaims(status, c1StatusPath, errors);
	rejectC1BSelfReceiptClaims(handoff, "docs/HANDOFF_MODE_C.md", errors);
	if (didrunBugs !== undefined) rejectC1BSelfReceiptClaims(didrunBugs, c1DidrunBugsPath, errors);
}

async function checkStatusPhase(root, overrides, status, handoff, errors, injectedReceiptAuthority) {
	let receiptText;
	try {
		receiptText = await readText(root, receiptDeclarationPath, overrides);
	} catch (error) {
		if (error.code !== "ENOENT") errors.push(`${receiptDeclarationPath}: unreadable (${error.message})`);
	}

	if (receiptText === undefined) {
		requireExactlyOnce(
			status,
			"**State:** provisional C0a source document; every C0 row below is `UNRECEIPTED`",
			statusPath,
			"C0A provisional state",
			errors,
		);
		for (const label of c0ClaimLabels) {
			requireExactlyOnce(status, `| \`${label}\` | \`UNRECEIPTED\` |`, statusPath, "C0A provisional receipt map", errors);
		}
		if (status.includes("C0b reconciled receipt document")) {
			errors.push(`${statusPath}: C0A cannot claim reconciled receipt state without ${receiptDeclarationPath}`);
		}
		if (handoff.includes("<!-- P07B-C-C0A-RECEIPTS:START -->") || handoff.includes("<!-- P07B-C-C0A-RECEIPTS:END -->")) {
			errors.push(`docs/HANDOFF_MODE_C.md: C0A cannot contain a reconciled C0a receipt block`);
		}
		return;
	}

	let receipt;
	try {
		receipt = JSON.parse(receiptText);
	} catch (error) {
		errors.push(`${receiptDeclarationPath}: invalid JSON (${error.message})`);
		return;
	}
	const receiptErrors = validateReceiptDeclaration(receipt);
	for (const error of receiptErrors) errors.push(`${receiptDeclarationPath}: invalid receipt declaration (${error})`);
	if (receiptErrors.length > 0) return;
	let receiptAuthority = injectedReceiptAuthority;
	if (receiptAuthority === undefined) {
		try {
			receiptAuthority = await loadReceiptAuthorityFromGit(root, receipt);
		} catch (error) {
			errors.push(`${receiptDeclarationPath}: Git/didrun authority unavailable (${error.message})`);
			return;
		}
	}
	const authorityErrors = validateReceiptAuthority(receipt, receiptAuthority);
	for (const error of authorityErrors) errors.push(`${receiptDeclarationPath}: Git/didrun authority mismatch (${error})`);
	if (authorityErrors.length > 0) return;

	const sealedHistoricalState = `**State:** historical sealed C0 authority boundary. C0a source commit \`${sealedC0AIdentity.commit}\`, tree \`${sealedC0AIdentity.tree}\`, and C0b reconciliation commit \`${sealedC0BIdentity.commit}\`, tree \`${sealedC0BIdentity.tree}\`, are note-present and strict-clean.`;
	const expectedState = receipt.source_commit === sealedC0AIdentity.commit && receipt.source_tree === sealedC0AIdentity.tree
		? sealedHistoricalState
		: `**State:** C0b reconciled receipt document for source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`.`;
	requireExactlyOnce(status, expectedState, statusPath, "C0B reconciled state", errors);
	requireExactlyOnce(status, "## C0a receipt map", statusPath, "C0B receipt heading", errors);
	requireExactlyOnce(status, "- C0a strict exit: `0`", statusPath, "C0B strict result", errors);
	requireExactlyOnce(
		status,
		`- C0a strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\``,
		statusPath,
		"C0B strict claim count",
		errors,
	);
	for (const claim of receipt.claims) {
		requireExactlyOnce(status, `| \`${claim.label}\` | \`${claim.grade}\` |`, statusPath, "C0B receipt map", errors);
	}
	requireExactlyOnce(handoff, "<!-- P07B-C-C0A-RECEIPTS:START -->", "docs/HANDOFF_MODE_C.md", "C0B receipt block", errors);
	requireExactlyOnce(handoff, "<!-- P07B-C-C0A-RECEIPTS:END -->", "docs/HANDOFF_MODE_C.md", "C0B receipt block", errors);
	requireExactlyOnce(
		handoff,
		`C0a source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`.`,
		"docs/HANDOFF_MODE_C.md",
		"C0B source identity",
		errors,
	);
	requireExactlyOnce(
		handoff,
		`C0a strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\`; strict exit: \`0\`.`,
		"docs/HANDOFF_MODE_C.md",
		"C0B strict result",
		errors,
	);
	for (const claim of receipt.claims) {
		requireExactlyOnce(handoff, `| \`${claim.label}\` | \`${claim.grade}\` |`, "docs/HANDOFF_MODE_C.md", "C0B claim map", errors);
	}
}

export async function checkPlan(root = repositoryRoot, overrides = new Map(), receiptAuthority, c1ReceiptAuthority) {
	const errors = [];
	const bodies = new Map();

	for (const [path, snippets] of Object.entries(requiredText)) {
		let body;
		try {
			body = await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C0 ruling: ${JSON.stringify(snippet)}`);
		}
	}

	for (const [path, snippets] of Object.entries(requiredC1MaintenanceText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) {
				errors.push(`${path}: missing required pre-enrollment maintenance ruling: ${JSON.stringify(snippet)}`);
			}
		}
	}
	const computedC1MaintenanceDigest = `sha256:${createHash("sha256")
		.update(`${requiredC1MaintenancePaths.join("\n")}\n`, "utf8").digest("hex")}`;
	if (computedC1MaintenanceDigest !== requiredC1MaintenanceDigest) {
		errors.push(`internal C1M roster digest mismatch: ${computedC1MaintenanceDigest}`);
	}
	for (const [path, snippets] of Object.entries(requiredC1VerificationText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) {
				errors.push(`${path}: missing required verification-throughput ruling: ${JSON.stringify(snippet)}`);
			}
		}
	}
	const computedC1VerificationDigest = `sha256:${createHash("sha256")
		.update(`${requiredC1VerificationPaths.join("\n")}\n`, "utf8").digest("hex")}`;
	if (computedC1VerificationDigest !== requiredC1VerificationDigest) {
		errors.push(`internal C1V roster digest mismatch: ${computedC1VerificationDigest}`);
	}
	for (const [path, snippets] of Object.entries(requiredC1EvidenceMaintenanceText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) {
				errors.push(`${path}: missing required local-evidence maintenance ruling: ${JSON.stringify(snippet)}`);
			}
		}
	}
	const computedC1EvidenceMaintenanceDigest = `sha256:${createHash("sha256")
		.update(`${requiredC1EvidenceMaintenancePaths.join("\n")}\n`, "utf8").digest("hex")}`;
	if (computedC1EvidenceMaintenanceDigest !== requiredC1EvidenceMaintenanceDigest) {
		errors.push(`internal C1E roster digest mismatch: ${computedC1EvidenceMaintenanceDigest}`);
	}
	if (!isDeepStrictEqual(qualificationCaseIDs, requiredQualificationCaseIDs)) {
		errors.push("internal C1V Go repetition qualification case roster mismatch");
	}
	const computedQualificationMatrixDigest = `sha256:${qualificationMatrixDigest()}`;
	if (computedQualificationMatrixDigest !== requiredQualificationMatrixDigest) {
		errors.push(`internal C1V Go repetition qualification matrix digest mismatch: ${computedQualificationMatrixDigest}`);
	}
	const verificationBody = bodies.get("docs/VERIFICATION.md");
	if (verificationBody !== undefined) {
		requireExactlyOnce(
			verificationBody,
			requiredC1MaintenanceDeclaration,
			"docs/VERIFICATION.md",
			"pre-enrollment maintenance scope declaration",
			errors,
		);
		requireExactlyOnce(
			verificationBody,
			requiredC1VerificationDeclaration,
			"docs/VERIFICATION.md",
			"verification-throughput scope declaration",
			errors,
		);
		requireExactlyOnce(
			verificationBody,
			requiredC1EvidenceMaintenanceDeclaration,
			"docs/VERIFICATION.md",
			"local-evidence maintenance scope declaration",
			errors,
		);
	}
	const verificationStatusBody = bodies.get(c1VerificationStatusPath);
	if (verificationStatusBody !== undefined) {
		requireExactlyOnce(
			verificationStatusBody,
			requiredC1VerificationDeclaration,
			c1VerificationStatusPath,
			"verification-throughput scope declaration",
			errors,
		);
	}
	const evidenceMaintenanceStatusBody = bodies.get(c1EvidenceMaintenanceStatusPath);
	if (evidenceMaintenanceStatusBody !== undefined) {
		requireExactlyOnce(
			evidenceMaintenanceStatusBody,
			requiredC1EvidenceMaintenanceDeclaration,
			c1EvidenceMaintenanceStatusPath,
			"local-evidence maintenance scope declaration",
			errors,
		);
		for (const label of c1EvidenceMaintenanceClaimLabels) {
			requireExactlyOnce(
				evidenceMaintenanceStatusBody,
				`| \`${label}\` |`,
				c1EvidenceMaintenanceStatusPath,
				"local-evidence maintenance intended claim map",
				errors,
			);
		}
	}

	try {
		const maintenanceStatus = await readText(root, c1MaintenanceStatusPath, overrides);
		for (const fixed of [
			c1MaintenanceIdentity.sourceCommit,
			c1MaintenanceIdentity.sourceTree,
			c1MaintenanceIdentity.receiptCommit,
			c1MaintenanceIdentity.receiptTree,
			"8/8 claims recorded-exact",
			"6/6 claims recorded-exact",
			".didrun-history/2026-07-18-p07b-c-c1-output-cap-maintenance/.didrun/",
			".didrun-history/2026-07-18-p07b-c-c1-output-cap-maintenance-receipt/.didrun/",
			"tracked C1 grades require a separately sealed C1B receipt boundary",
		]) {
			if (!maintenanceStatus.includes(fixed)) {
				errors.push(`${c1MaintenanceStatusPath}: missing sealed maintenance authority: ${JSON.stringify(fixed)}`);
			}
		}
		for (const label of [...c1MaintenanceSourceClaims, ...c1MaintenanceReceiptClaims]) {
			requireExactlyOnce(
				maintenanceStatus,
				`| \`${label}\` | \`TREE-EXACT\` |`,
				c1MaintenanceStatusPath,
				"maintenance receipt map",
				errors,
			);
		}
	} catch (error) {
		errors.push(`${c1MaintenanceStatusPath}: unreadable (${error.message})`);
	}

	let c1Status;
	try {
		c1Status = await readText(root, c1StatusPath, overrides);
		for (const snippet of [
			"C1 implements the strict, inert canonical algebra",
			"The three JSON Schemas are closed syntax projections",
			"Target runtime facts are checkpointed path/content identity",
			"physical evidence content and chronology",
			"sha256:78f5a2c76d38dc2de96b11cf0439f33f73427160fc441ba89f9f39897dab6d4b",
			"prefixes are empty",
			"sha256:6755b26a51dc71c4565fb1c600ba6282e63f44bff3521cecc1c8ba463c2882fe",
			"go test -json",
		]) {
			if (!c1Status.includes(snippet)) errors.push(`${c1StatusPath}: missing required C1 boundary: ${JSON.stringify(snippet)}`);
		}
	} catch (error) {
		errors.push(`${c1StatusPath}: unreadable (${error.message})`);
	}

	const status = bodies.get(statusPath);
	const handoff = bodies.get("docs/HANDOFF_MODE_C.md");
	if (status !== undefined && handoff !== undefined) {
		await checkStatusPhase(root, overrides, status, handoff, errors, receiptAuthority);
		if (c1Status !== undefined) {
			await checkC1ReceiptPhase(root, overrides, c1Status, handoff, errors, c1ReceiptAuthority);
		}
	}

	for (const path of controllingPaths) {
		const body = bodies.get(path);
		if (body === undefined) continue;
		for (const snippet of forbiddenAcrossControllingDocs) {
			if (body.includes(snippet)) errors.push(`${path}: contains superseded or overbroad ruling: ${JSON.stringify(snippet)}`);
		}
		for (const snippet of forbiddenByPath[path] ?? []) {
			if (body.includes(snippet)) errors.push(`${path}: contains superseded path-specific ruling: ${JSON.stringify(snippet)}`);
		}
	}

	const prompt = bodies.get("docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md");
	if (prompt !== undefined) checkOrderedPromptHeadings(prompt, errors);

	for (const [path, heading] of Object.entries(researchShape)) {
		const body = bodies.get(path);
		if (body === undefined) continue;
		if (!body.startsWith(`${heading}\n`)) errors.push(`${path}: exact H1 mismatch`);
		if (Buffer.byteLength(body, "utf8") < 1500) errors.push(`${path}: evidence file is not substantive (under 1500 bytes)`);
	}

	await checkC1PlanningProjection(root, overrides, errors);

	try {
		const declaration = JSON.parse(await readText(root, authorityDeclarationPath, overrides));
		if (!isDeepStrictEqual(declaration, expectedAuthorityDeclaration)) {
			errors.push(`${authorityDeclarationPath}: authority declaration mismatch`);
		}
	} catch (error) {
		errors.push(`${authorityDeclarationPath}: invalid authority declaration (${error.message})`);
	}

	try {
		const specification = validateSpecification(JSON.parse(await readText(root, "spec/verification/p07b-c-unit-paths.json", overrides)));
		for (const path of requiredC0APaths) {
			if (!specification.units.C0A.exact.includes(path)) errors.push(`spec/verification/p07b-c-unit-paths.json: C0A omits required path ${path}`);
		}
		if (!isDeepStrictEqual(specification.units.C1.exact, requiredC1Paths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1 exact path roster mismatch");
		}
		if (specification.units.C0B.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C0B historical verification profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C1.prefixes, [])) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1 prefix roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C1M.exact, requiredC1MaintenancePaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1M exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C1M.prefixes, [])) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1M prefix roster mismatch");
		}
		if (specification.units.C1M.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1M verification profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C1V.exact, requiredC1VerificationPaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1V exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C1V.prefixes, [])) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1V prefix roster mismatch");
		}
		if (specification.units.C1V.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1V verification profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C1E.exact, requiredC1EvidenceMaintenancePaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1E exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C1E.prefixes, [])) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1E prefix roster mismatch");
		}
		if (specification.units.C1E.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1E verification profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C1B.exact, requiredC1BPaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1B exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C1B.prefixes, [])) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1B prefix roster mismatch");
		}
		if (specification.units.C1B.verification_profile !== "RECEIPT_RECONCILIATION") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1B verification profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C1B.receipt_claims, requiredC1BReceiptClaims)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C1B receipt claim manifest mismatch");
		}
	} catch (error) {
		errors.push(`spec/verification/p07b-c-unit-paths.json: invalid (${error.message})`);
	}

	return errors;
}

async function mutatedText(path, transform) {
	return transform(await readText(repositoryRoot, path, new Map()));
}

async function runSelfTest() {
	const baseline = await checkPlan();
	if (baseline.length > 0) throw new Error(`P07B-C C1 evolved plan checker self-test baseline failed:\n${baseline.join("\n")}`);
	const localEvidenceMutations = await runLocalEvidenceSelfTest();
	const receiptModePhaseMutations = await runC1ReceiptModePhaseSelfTest();

	const promptPath = "docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md";
	const prompt = await readText(repositoryRoot, promptPath, new Map());
	const authority = JSON.parse(await readText(repositoryRoot, authorityDeclarationPath, new Map()));
	const mutateAuthority = (transform) => {
		const candidate = JSON.parse(JSON.stringify(authority));
		transform(candidate);
		return `${JSON.stringify(candidate, null, 2)}\n`;
	};
	const configPath = "spec/verification/p07b-c-unit-paths.json";
	const currentConfiguration = JSON.parse(await readText(repositoryRoot, configPath, new Map()));
	const configuration = structuredClone(currentConfiguration);
	delete configuration.units.C6B;
	const syntheticReceipt = {
		schema_version: "countershape/p07b-c-c0-receipt/v1",
		source_commit: "a".repeat(40),
		source_tree: "b".repeat(40),
		strict_exit_code: 0,
		strict_claims_recorded_exact: c0ClaimLabels.length,
		strict_claims_total: c0ClaimLabels.length,
		claims: c0ClaimLabels.map((label) => ({ label, grade: "TREE-EXACT" })),
	};
	const syntheticAuthority = {
		commit: syntheticReceipt.source_commit,
		tree: syntheticReceipt.source_tree,
		subject: "docs: lock P07B-C execution authority",
		ancestor_of_head: true,
		note: {
			commit: syntheticReceipt.source_commit,
			tree: syntheticReceipt.source_tree,
			claims: c0ClaimLabels.map((label) => ({ claim: { label }, grade: "tree-exact", exit_code: 0 })),
			coverage: { total_events: c0ClaimLabels.length, by_coverage: { complete: c0ClaimLabels.length } },
		},
	};
	const currentStatus = await readText(repositoryRoot, statusPath, new Map());
	const syntheticState = `**State:** C0b reconciled receipt document for source commit \`${syntheticReceipt.source_commit}\`, tree \`${syntheticReceipt.source_tree}\`.`;
	let syntheticStatus = currentStatus
		.replace(/^\*\*State:\*\*.*$/mu, syntheticState)
		.replace(/^## (?:Provisional receipt map|C0a receipt map)$/mu, "## C0a receipt map")
		.replace(/^- C0a strict exit:.*\n?/mu, "")
		.replace(/^- C0a strict claims:.*\n?/mu, "");
	syntheticStatus = syntheticStatus.replace(
		syntheticState,
		`${syntheticState}\n\n- C0a strict exit: \`0\`\n- C0a strict claims: \`${c0ClaimLabels.length}/${c0ClaimLabels.length} claims recorded-exact\``,
	);
	for (const label of c0ClaimLabels) {
		syntheticStatus = syntheticStatus.replace(
			`| \`${label}\` | \`UNRECEIPTED\` |`,
			`| \`${label}\` | \`TREE-EXACT\` |`,
		);
	}
	const receiptBlock = [
		"<!-- P07B-C-C0A-RECEIPTS:START -->",
		"### P07B-C C0a receipt map",
		"",
		`C0a source commit \`${syntheticReceipt.source_commit}\`, tree \`${syntheticReceipt.source_tree}\`.`,
		`C0a strict claims: \`${c0ClaimLabels.length}/${c0ClaimLabels.length} claims recorded-exact\`; strict exit: \`0\`.`,
		"",
		"| Claim label | Verbatim grade |",
		"| --- | --- |",
		...c0ClaimLabels.map((label) => `| \`${label}\` | \`TREE-EXACT\` |`),
		"<!-- P07B-C-C0A-RECEIPTS:END -->",
	].join("\n");
	const currentHandoff = await readText(repositoryRoot, "docs/HANDOFF_MODE_C.md", new Map());
	const currentC1MaintenanceStatus = await readText(repositoryRoot, c1MaintenanceStatusPath, new Map());
	const currentC1VerificationStatus = await readText(repositoryRoot, c1VerificationStatusPath, new Map());
	const currentC1EvidenceMaintenanceStatus = await readText(repositoryRoot, c1EvidenceMaintenanceStatusPath, new Map());
	const currentC1Status = await readText(repositoryRoot, c1StatusPath, new Map());
	const currentC1DidrunBugs = await readText(repositoryRoot, c1DidrunBugsPath, new Map());
	const currentPromptPack = await readText(repositoryRoot, "docs/PROMPT_PACK.md", new Map());
	const currentVerification = await readText(repositoryRoot, "docs/VERIFICATION.md", new Map());
	const syntheticHandoff = `${currentHandoff.replace(
		/\n?<!-- P07B-C-C0A-RECEIPTS:START -->[\s\S]*?<!-- P07B-C-C0A-RECEIPTS:END -->\n?/u,
		"\n",
	).trimEnd()}\n\n${receiptBlock}\n`;
	const syntheticReceiptText = `${JSON.stringify(syntheticReceipt, null, 2)}\n`;
	const syntheticOverrides = new Map([
		[statusPath, syntheticStatus],
		[receiptDeclarationPath, syntheticReceiptText],
		["docs/HANDOFF_MODE_C.md", syntheticHandoff],
	]);
	const syntheticBaseline = await checkPlan(repositoryRoot, syntheticOverrides, syntheticAuthority);
	if (syntheticBaseline.length > 0) {
		throw new Error(`P07B-C C0 plan checker synthetic C0B baseline failed:\n${syntheticBaseline.join("\n")}`);
	}

	const syntheticC1Receipt = {
		schema_version: "countershape/p07b-c-c1-receipt/v1",
		source_commit: sealedC1Identity.commit,
		source_tree: sealedC1Identity.tree,
		source_subject: sealedC1Identity.subject,
		strict_exit_code: 0,
		strict_claims_recorded_exact: c1ClaimLabels.length,
		strict_claims_total: c1ClaimLabels.length,
		coverage_total_events: c1ClaimLabels.length,
		coverage_complete_events: c1ClaimLabels.length,
		secrets_override: true,
		claims: c1ClaimLabels.map((label, index) => ({
			label,
			type: c1ClaimTypes[index],
			grade: "TREE-EXACT",
			supporting_event_index: index,
		})),
		evidence: structuredClone(expectedC1ReceiptEvidence),
	};
	const syntheticC1ReceiptText = `${JSON.stringify(syntheticC1Receipt, null, 2)}\n`;
	const syntheticC1Authority = await loadReceiptAuthorityFromGit(repositoryRoot, syntheticC1Receipt);
	const c1ReconciledState = `**Source receipt:** C1 source commit \`${syntheticC1Receipt.source_commit}\`, tree \`${syntheticC1Receipt.source_tree}\`, is sealed, note-present, and strict-clean; every source grade below is \`TREE-EXACT\`. This C1B receipt-document working unit binds only that existing source and remains \`UNRECEIPTED\` until its own commit, seal, note, and strict boundary.`;
	const c1NoRecursion = "This receipt binds only the already-existing C1 source commit. C1B cannot name or grade its own commit, tree, Git note, or strict result.";
	const c1Disclosures = c1ReceiptDisclosureLines(syntheticC1Receipt);
	let syntheticC1Status = currentC1Status
		.replace(
			/^\*\*Source state:\*\*.*$/mu,
			`${c1ReconciledState}\n\n- C1 source strict exit: \`0\`\n- C1 source strict claims: \`${c1ClaimLabels.length}/${c1ClaimLabels.length} claims recorded-exact\`\n\n${c1NoRecursion}\n\n${[1, 2, 3, 4, 6].map((index) => c1Disclosures[index]).join("\n\n")}`,
		)
		.replace("## Intended C1 receipt map", "## C1 source receipt map")
		.replace(
			"| Intended capability | Exact claim label | Source grade |\n| --- | --- | --- |",
			"| Intended capability | Exact claim label | Claim type | Source grade |\n| --- | --- | --- | --- |",
		);
	for (let index = 0; index < c1ClaimLabels.length; index += 1) {
		const label = c1ClaimLabels[index];
		syntheticC1Status = syntheticC1Status.replace(
			`| \`${label}\` | \`UNRECEIPTED\` |`,
			`| \`${label}\` | \`${c1ClaimTypes[index]}\` | \`TREE-EXACT\` |`,
		);
	}
	const c1ReceiptBlock = [
		"<!-- P07B-C-C1-SOURCE-RECEIPTS:START -->",
		"### P07B-C C1 source receipt map",
		"",
		`C1 source commit \`${syntheticC1Receipt.source_commit}\`, tree \`${syntheticC1Receipt.source_tree}\`.`,
		`C1 strict claims: \`${c1ClaimLabels.length}/${c1ClaimLabels.length} claims recorded-exact\`; strict exit: \`0\`.`,
		"",
		...c1Disclosures,
		"",
		"| Claim label | Claim type | Verbatim grade |",
		"| --- | --- | --- |",
		...syntheticC1Receipt.claims.map((claim) => `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`),
		"<!-- P07B-C-C1-SOURCE-RECEIPTS:END -->",
	].join("\n");
	const syntheticC1Handoff = `${currentHandoff.replace(
		/\n?<!-- P07B-C-C1-SOURCE-RECEIPTS:START -->[\s\S]*?<!-- P07B-C-C1-SOURCE-RECEIPTS:END -->\n?/u,
		"\n",
	).trimEnd()}\n\n${c1ReceiptBlock}\n`;
	const c1DidrunBlock = [
		"<!-- P07B-C-C1-DIDRUN-LIVE:START -->",
		"## P07B-C C1 live-session evidence",
		"",
		...c1Disclosures,
		"",
		c1DidrunLiveLine,
		"",
		"This section binds only the already-sealed C1 source. C1B remains ungraded in its own tracked files.",
		"<!-- P07B-C-C1-DIDRUN-LIVE:END -->",
	].join("\n");
	const syntheticC1DidrunBugs = `${currentC1DidrunBugs.replace(
		/\n?<!-- P07B-C-C1-DIDRUN-LIVE:START -->[\s\S]*?<!-- P07B-C-C1-DIDRUN-LIVE:END -->\n?/u,
		"\n",
	).trimEnd()}\n\n${c1DidrunBlock}\n`;
	const syntheticC1Overrides = new Map([
		[c1StatusPath, syntheticC1Status],
		[c1ReceiptDeclarationPath, syntheticC1ReceiptText],
		["docs/HANDOFF_MODE_C.md", syntheticC1Handoff],
		[c1DidrunBugsPath, syntheticC1DidrunBugs],
	]);
	const syntheticC1Baseline = await checkPlan(
		repositoryRoot,
		syntheticC1Overrides,
		undefined,
		syntheticC1Authority,
	);
	if (syntheticC1Baseline.length > 0) {
		throw new Error(`P07B-C C1 plan checker synthetic C1B baseline failed:\n${syntheticC1Baseline.join("\n")}`);
	}
	const mutateC1Receipt = (transform) => {
		const candidate = structuredClone(syntheticC1Receipt);
		transform(candidate);
		return `${JSON.stringify(candidate, null, 2)}\n`;
	};
	const mutateC1Authority = (transform) => {
		const candidate = structuredClone(syntheticC1Authority);
		transform(candidate);
		return candidate;
	};
	const mutateC1Overrides = (path, value) => {
		const candidate = new Map(syntheticC1Overrides);
		candidate.set(path, value);
		return candidate;
	};

	let currentReceiptText;
	try {
		currentReceiptText = await readText(repositoryRoot, receiptDeclarationPath, new Map());
	} catch (error) {
		if (error.code !== "ENOENT") throw error;
	}
	const phaseReceiptCase = currentReceiptText === undefined ? {
		name: "provisional receipt-status inflation", path: statusPath,
		value: currentStatus.replace(
			`| \`${c0ClaimLabels[0]}\` | \`UNRECEIPTED\` |`,
			`| \`${c0ClaimLabels[0]}\` | \`RECEIPTED\` |`,
		),
		expect: "C0A provisional receipt map",
	} : {
		name: "reconciled receipt-grade mutation", path: receiptDeclarationPath,
		value: (() => {
			const candidate = JSON.parse(currentReceiptText);
			candidate.claims[0].grade = "UNRECEIPTED";
			return `${JSON.stringify(candidate, null, 2)}\n`;
		})(),
		expect: "invalid receipt declaration",
	};
	let currentC1ReceiptText;
	try {
		currentC1ReceiptText = await readText(repositoryRoot, c1ReceiptDeclarationPath, new Map());
	} catch (error) {
		if (error.code !== "ENOENT") throw error;
	}
	const c1PhaseReceiptCase = currentC1ReceiptText === undefined ? {
		name: "pending C1 source receipt inflation", path: c1StatusPath,
		value: currentC1Status.replace(
			`\`${c1ClaimLabels[0]}\` | \`UNRECEIPTED\``,
			`\`${c1ClaimLabels[0]}\` | \`TREE-EXACT\``,
		),
		expect: "C1 pending source receipt map",
	} : {
		name: "reconciled C1 source receipt-grade mutation", path: c1ReceiptDeclarationPath,
		value: (() => {
			const candidate = JSON.parse(currentC1ReceiptText);
			candidate.claims[0].grade = "UNRECEIPTED";
			return `${JSON.stringify(candidate, null, 2)}\n`;
		})(),
		expect: "invalid receipt declaration",
	};
	const cases = [
		{
			name: "C1 maintenance receipt-grade regression", path: c1MaintenanceStatusPath,
			value: currentC1MaintenanceStatus.replace(
				`| \`${c1MaintenanceReceiptClaims[0]}\` | \`TREE-EXACT\` |`,
				`| \`${c1MaintenanceReceiptClaims[0]}\` | \`UNRECEIPTED\` |`,
			),
			expect: "maintenance receipt map",
		},
		c1PhaseReceiptCase,
		{
			name: "C1 prompt-pack receipt-gate removal", path: "docs/PROMPT_PACK.md",
			value: currentPromptPack.replace("C2 still requires separately sealed/strict-clean C1B", "C1B gate omitted"),
			expect: "missing required C0 ruling",
		},
		{
			name: "C1 verification-roster removal", path: "docs/VERIFICATION.md",
			value: currentVerification.replace("P07B-C C1 inert-model architecture checker", "P07B-C checker omitted"),
			expect: "missing required C0 ruling",
		},
		{
			name: "C1M verification roster path substitution", path: "docs/VERIFICATION.md",
			value: currentVerification.replace("`docs/THREAT_MODEL.md`", "`docs/THREAT_MODEL-RENAMED.md`"),
			expect: "pre-enrollment maintenance scope declaration",
		},
		{
			name: "C1M verification roster digest substitution", path: "docs/VERIFICATION.md",
			value: currentVerification.replace(requiredC1MaintenanceDigest, `sha256:${"0".repeat(64)}`),
			expect: "pre-enrollment maintenance scope declaration",
		},
		{
			name: "C1V verification roster path substitution", path: "docs/VERIFICATION.md",
			value: currentVerification.replace(c1VerificationStatusPath, "docs/status/P07B-C-VERIFICATION-UNREVIEWED.md"),
			expect: "verification-throughput scope declaration",
		},
		{
			name: "C1V verification roster digest substitution", path: "docs/VERIFICATION.md",
			value: currentVerification.replace(requiredC1VerificationDigest, `sha256:${"0".repeat(64)}`),
			expect: "verification-throughput scope declaration",
		},
		{
			name: "C1V status roster path substitution", path: c1VerificationStatusPath,
			value: currentC1VerificationStatus.replace(c1VerificationStatusPath, "docs/status/P07B-C-VERIFICATION-UNREVIEWED.md"),
			expect: "verification-throughput scope declaration",
		},
		{
			name: "C1V status roster digest substitution", path: c1VerificationStatusPath,
			value: currentC1VerificationStatus.replace(requiredC1VerificationDigest, `sha256:${"0".repeat(64)}`),
			expect: "verification-throughput scope declaration",
		},
		{
			name: "C1V qualification matrix digest substitution", path: c1VerificationStatusPath,
			value: currentC1VerificationStatus.replace(requiredQualificationMatrixDigest, `sha256:${"0".repeat(64)}`),
			expect: "missing required verification-throughput ruling",
		},
		{
			name: "C1V qualification case omission", path: c1VerificationStatusPath,
			value: currentC1VerificationStatus.replace("`world-output-caps-50`", "`world-output-caps-49`"),
			expect: "missing required verification-throughput ruling",
		},
		{
			name: "C1V persistent-cache overclaim", path: c1VerificationStatusPath,
			value: currentC1VerificationStatus.replace("`GOCACHE` remains fresh", "`GOCACHE` persists across runs"),
			expect: "missing required verification-throughput ruling",
		},
		{
			name: "C1V stale-lock auto-deletion", path: c1VerificationStatusPath,
			value: currentC1VerificationStatus.replace(
				"C1V does not automatically delete stale O_EXCL locks",
				"C1V automatically deletes stale O_EXCL locks",
			),
			expect: "missing required verification-throughput ruling",
		},
		{
			name: "C1V receipt-profile overclaim", path: "docs/VERIFICATION.md",
			value: currentVerification.replace(
				"it cannot support a cumulative, full-suite, runtime, security, or unchanged-behavior claim",
				"it supports cumulative and runtime claims",
			),
			expect: "missing required verification-throughput ruling",
		},
		{
			name: "C1E verification roster digest substitution", path: "docs/VERIFICATION.md",
			value: currentVerification.replace(requiredC1EvidenceMaintenanceDigest, `sha256:${"0".repeat(64)}`),
			expect: "local-evidence maintenance scope declaration",
		},
		{
			name: "C1E status mixed-namespace count substitution", path: c1EvidenceMaintenanceStatusPath,
			value: currentC1EvidenceMaintenanceStatus.replace("20 flat didrun capture blobs", "19 flat didrun capture blobs"),
			expect: "missing required local-evidence maintenance ruling",
		},
		{
			name: "C1E verification namespace removal", path: "docs/VERIFICATION.md",
			value: currentVerification.replace("`objects/<2-lowercase-hex>/<38-lowercase-hex>`", "Git objects omitted"),
			expect: "missing required local-evidence maintenance ruling",
		},
		{
			name: "C1M lifecycle signal-authority regression", path: "docs/STATE_MACHINES.md",
			value: (await readText(repositoryRoot, "docs/STATE_MACHINES.md", new Map())).replace(
				"skip TERM on absence, signal only after clean presence",
				"signal on numeric presence",
			),
			expect: "missing required pre-enrollment maintenance ruling",
		},
		{
			name: "C1M historical U2 provenance removal", path: "docs/decisions/0002-u2-impure-boundary.md",
			value: (await readText(repositoryRoot, "docs/decisions/0002-u2-impure-boundary.md", new Map())).replace(
				"does not relabel or extend the sealed historical U2 receipt",
				"inherits the historical U2 receipt",
			),
			expect: "missing required pre-enrollment maintenance ruling",
		},
		{
			name: "sealed C0 successor regression", path: statusPath,
			value: currentStatus.replace("C0 is complete. C1 source commit `2fceacecbacb89fd7650f1570b2af33e6ea25ed3`", "C0 successor omitted"),
			expect: "missing required C0 ruling",
		},
		{
			name: "required interlock authority edge", path: promptPath,
			value: prompt.replace("`ExecutionInterlock` is one store-private, store-wide operational CAS", "Interlock omitted"),
			expect: "missing required C0 ruling",
		},
		{
			name: "contradictory two-object ruling", path: "docs/SEMANTICS.md",
			value: `${await readText(repositoryRoot, "docs/SEMANTICS.md", new Map())}\npersist exactly two semantic truth objects\n`,
			expect: "contains superseded or overbroad ruling",
		},
		{
			name: "C1 derived result roster widening", path: "spec/schema/v1/contract-execution.schema.json",
			value: await mutatedText("spec/schema/v1/contract-execution.schema.json", (body) => {
				const schema = JSON.parse(body);
				schema.properties.result.enum.push("UNKNOWN");
				return `${JSON.stringify(schema, null, 2)}\n`;
			}),
			expect: "C1 planning projection mismatch: schema authority",
		},
		{
			name: "C1 required result roster weakening", path: "spec/schema/v1/contract-execution.schema.json",
			value: await mutatedText("spec/schema/v1/contract-execution.schema.json", (body) => {
				const schema = JSON.parse(body);
				const index = schema.required?.indexOf("result") ?? -1;
				if (index < 0) throw new Error("C1 required result mutation anchor absent");
				schema.required.splice(index, 1);
				return `${JSON.stringify(schema, null, 2)}\n`;
			}),
			expect: "C1 planning projection mismatch: schema authority",
		},
		{
			name: "evidence content deletion", path: "research/deep-dive/p07b-c/08-different-model-red-team.md",
			value: await mutatedText("research/deep-dive/p07b-c/08-different-model-red-team.md", (body) => body.replace("## Post-write closure correction", "## Closure")),
			expect: "missing required C0 ruling",
		},
		{
			name: "prompt heading order", path: promptPath,
			value: prompt.replace(promptHeadings[2], "# TEMP-C2").replace(promptHeadings[3], promptHeadings[2]).replace("# TEMP-C2", promptHeadings[3]),
			expect: "heading order violation",
		},
		{
			name: "substantive evidence deletion", path: "research/deep-dive/p07b-c/05-dx-wake-fit.md",
			value: `${researchShape["research/deep-dive/p07b-c/05-dx-wake-fit.md"]}\ncontract, candidate, pinned commit, checked fields, result\nCountershape does not consume Wake sessions\n`,
			expect: "not substantive",
		},
		{
			name: "unit allowlist roster", path: configPath,
			value: `${JSON.stringify(configuration, null, 2)}\n`,
			expect: "invalid",
		},
		{
			name: "C1 allowlist expansion", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1.exact.push("tools/z-c1-self-authorized.mjs");
				candidate.units.C1.exact.sort();
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1 exact path roster mismatch",
		},
		{
			name: "C1 exact model-test removal", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1.exact = candidate.units.C1.exact.filter(
					(path) => path !== "internal/contractexec/model/schema_parity_test.go",
				);
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1 exact path roster mismatch",
		},
		{
			name: "C1 maintenance reconciliation removal", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1.exact = candidate.units.C1.exact.filter(
					(path) => path !== c1MaintenanceStatusPath,
				);
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1 exact path roster mismatch",
		},
		{
			name: "C1 broad model-prefix restoration", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1.prefixes = ["internal/contractexec/model/"];
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "invalid",
		},
		{
			name: "C1 architecture enrollment removal", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1.exact = candidate.units.C1.exact.filter(
					(path) => path !== "tools/check-p07b-c-architecture.mjs",
				);
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1 exact path roster mismatch",
		},
		{
			name: "C1 predecessor architecture repair removal", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1.exact = candidate.units.C1.exact.filter(
					(path) => path !== "tools/check-p07b-b-architecture.mjs",
				);
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1 exact path roster mismatch",
		},
		{
			name: "C1M exact path deletion", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1M.exact = candidate.units.C1M.exact.filter(
					(path) => path !== "internal/world/process_darwin_test.go",
				);
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1M exact path roster mismatch",
		},
		{
			name: "C1M exact path substitution", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1M.exact = candidate.units.C1M.exact.map((path) =>
					path === "internal/world/process_darwin_test.go" ? "internal/world/process_darwin_unreviewed.go" : path,
				).sort();
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1M exact path roster mismatch",
		},
		{
			name: "C1M prefix introduction", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1M.prefixes = ["vendor/"];
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1M prefix roster mismatch",
		},
		{
			name: "C1V exact path deletion", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1V.exact = candidate.units.C1V.exact.filter(
					(path) => path !== c1VerificationStatusPath,
				);
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1V exact path roster mismatch",
		},
		{
			name: "C1V exact path substitution", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1V.exact = candidate.units.C1V.exact.map((path) =>
					path === c1VerificationStatusPath ? "docs/status/P07B-C-VERIFICATION-UNREVIEWED.md" : path,
				).sort();
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1V exact path roster mismatch",
		},
		{
			name: "C1V prefix introduction", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1V.prefixes = ["tools/"];
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "invalid",
		},
		{
			name: "C1V receipt-profile substitution", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1V.verification_profile = "RECEIPT_RECONCILIATION";
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "invalid",
		},
		{
			name: "C1E unit omission", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				delete candidate.units.C1E;
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "invalid",
		},
		{
			name: "C1E exact path deletion", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1E.exact = candidate.units.C1E.exact.filter(
					(path) => path !== c1EvidenceMaintenanceStatusPath,
				);
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1E exact path roster mismatch",
		},
		{
			name: "C1E exact path substitution", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1E.exact = candidate.units.C1E.exact.map((path) =>
					path === c1EvidenceMaintenanceStatusPath ? "docs/status/P07B-C-C1-EVIDENCE-UNREVIEWED.md" : path,
				).sort();
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1E exact path roster mismatch",
		},
		{
			name: "C1E prefix introduction", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1E.prefixes = ["tools/"];
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "invalid",
		},
		{
			name: "C1E receipt-profile substitution", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1E.verification_profile = "RECEIPT_RECONCILIATION";
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "invalid",
		},
		{
			name: "C1B unit omission", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				delete candidate.units.C1B;
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "invalid",
		},
		{
			name: "C1B unit order drift", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				const entries = Object.entries(candidate.units);
				const c1b = entries.find(([unit]) => unit === "C1B");
				candidate.units = Object.fromEntries([
					...entries.filter(([unit]) => unit !== "C1B" && unit !== "C2"),
					entries.find(([unit]) => unit === "C2"),
					c1b,
				]);
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "invalid",
		},
		{
			name: "C1B receipt path removal", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1B.exact = candidate.units.C1B.exact.filter(
					(path) => path !== c1ReceiptDeclarationPath,
				);
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1B exact path roster mismatch",
		},
		{
			name: "C1B exact-path expansion", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1B.exact.push("tools/c1b-self-authorized.mjs");
				candidate.units.C1B.exact.sort();
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "invalid",
		},
		{
			name: "C1B prefix introduction", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1B.prefixes = ["docs/status/"];
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "invalid",
		},
		{
			name: "C1B source-profile substitution", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1B.verification_profile = "SOURCE_FULL";
				delete candidate.units.C1B.receipt_claims;
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1B verification profile mismatch",
		},
		{
			name: "C1B receipt claim label drift", path: configPath,
			value: (() => {
				const candidate = structuredClone(currentConfiguration);
				candidate.units.C1B.receipt_claims[0].label = "P07B-C C1B source receipt reconciliation altered";
				return `${JSON.stringify(candidate, null, 2)}\n`;
			})(),
			expect: "C1B receipt claim manifest mismatch",
		},
		{
			name: "fourth semantic object", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => candidate.semantic_objects.push({
				name: "FourthObject", storage: "IMMUTABLE_NONHEAD", production_issuer: "C4",
			})),
			expect: "authority declaration mismatch",
		},
		{
			name: "second mutable operational object", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => candidate.operational_objects.push({
				name: "MutableStatus", storage: "STORE_PRIVATE_MUTABLE", authority: "RESULT_STATUS", production_acquirer: "C4",
			})),
			expect: "authority declaration mismatch",
		},
		{
			name: "premature C2 target issuer", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => { candidate.semantic_objects[0].production_issuer = "C2"; }),
			expect: "authority declaration mismatch",
		},
		{
			name: "same-boot reset weakening", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => { candidate.authority_edges.reset = "EXPLICIT_OPERATOR_ONLY"; }),
			expect: "authority declaration mismatch",
		},
		{
			name: "scope-domain loss", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => { candidate.scope_domains.pop(); }),
			expect: "authority declaration mismatch",
		},
		{
			name: "P08 surface reservation loss", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => { candidate.reserved_surface = "C6"; }),
			expect: "authority declaration mismatch",
		},
		{
			name: "Wake authority expansion", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => { candidate.wake_boundary = "SESSION_RESUME"; }),
			expect: "authority declaration mismatch",
		},
		phaseReceiptCase,
		{
			name: "C1B receipt with unreconciled status",
			overrides: mutateC1Overrides(
				c1StatusPath,
				syntheticC1Status.replace(c1ReconciledState, "**Source receipt:** C1 source reconciliation omitted."),
			),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "C1B reconciled source state",
		},
		{
			name: "C1B self-referential receipt field",
			overrides: mutateC1Overrides(c1ReceiptDeclarationPath, mutateC1Receipt((candidate) => {
				candidate.receipt_commit = "f".repeat(40);
			})),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "invalid receipt declaration",
		},
		{
			name: "C1B source claim order drift",
			overrides: mutateC1Overrides(c1ReceiptDeclarationPath, mutateC1Receipt((candidate) => {
				[candidate.claims[0], candidate.claims[1]] = [candidate.claims[1], candidate.claims[0]];
			})),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "invalid receipt declaration",
		},
		{
			name: "C1B local evidence digest drift",
			overrides: mutateC1Overrides(c1ReceiptDeclarationPath, mutateC1Receipt((candidate) => {
				candidate.evidence.html_report.sha256 = "0".repeat(64);
			})),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "invalid receipt declaration",
		},
		{
			name: "C1B admitted source commit mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.commit = "f".repeat(40); }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B admitted source tree mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.tree = "f".repeat(40); }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B source parent mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.parent = "f".repeat(40); }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B source subject mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.subject = "wrong subject"; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B source ancestry mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.ancestor_of_head = false; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note object mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note_blob_oid = "f".repeat(40); }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note body digest mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note_body_sha256 = "f".repeat(64); }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note label mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.claims[0].claim.label = "wrong label"; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note version mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.version = 2; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note commit mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.commit = "f".repeat(40); }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note tree mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.tree = "f".repeat(40); }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note seal override mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.secrets_override = false; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note claim type mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.claims[0].claim.ctype = "command-succeeded"; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note declared index mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.claims[0].claim.declared_at_index = 1; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note event-index mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.claims[0].claim.event_indices = [1]; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note pathspec mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.claims[0].claim.pathspecs = ["docs/"]; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note argv preview omission",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.claims[0].claim.argv_preview = []; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note support-index mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.claims[0].supporting_event_index = 1; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note exit mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.claims[0].exit_code = 1; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note reason mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.claims[0].reason = "wrong reason"; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note delta mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.claims[0].delta = ["dirty"]; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note grade mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.claims[0].grade = "tree"; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B note coverage mismatch",
			overrides: syntheticC1Overrides,
			c1ReceiptAuthority: mutateC1Authority((candidate) => { candidate.note.coverage.by_coverage.complete -= 1; }),
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "C1B no-recursion statement removal",
			overrides: mutateC1Overrides(c1StatusPath, syntheticC1Status.replace(c1NoRecursion, "C1B receipt boundary omitted.")),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "C1B no-recursion boundary",
		},
		{
			name: "C1B status grade mismatch",
			overrides: mutateC1Overrides(
				c1StatusPath,
				syntheticC1Status.replace(
					`| \`${c1ClaimLabels[0]}\` | \`${c1ClaimTypes[0]}\` | \`TREE-EXACT\` |`,
					`| \`${c1ClaimLabels[0]}\` | \`${c1ClaimTypes[0]}\` | \`UNRECEIPTED\` |`,
				),
			),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "C1B source receipt map",
		},
		{
			name: "C1B contradictory duplicate status row",
			overrides: mutateC1Overrides(
				c1StatusPath,
				`${syntheticC1Status}\n| duplicate | \`${c1ClaimLabels[0]}\` | \`${c1ClaimTypes[0]}\` | \`UNRECEIPTED\` |\n`,
			),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "claim-label uniqueness",
		},
		{
			name: "C1B status self-commit claim",
			overrides: mutateC1Overrides(
				c1StatusPath,
				`${syntheticC1Status}\nC1B commit \`${"f".repeat(40)}\` is note-present with \`TREE-EXACT\`.\n`,
			),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "C1B self-receipt claim",
		},
		{
			name: "C1B handoff self-strict claim",
			overrides: mutateC1Overrides(
				"docs/HANDOFF_MODE_C.md",
				`${syntheticC1Handoff}\nC1B strict exit: \`0\`; 7/7 claims recorded-exact.\n`,
			),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "C1B self-receipt claim",
		},
		{
			name: "C1B hash-free self-grade claim",
			overrides: mutateC1Overrides(
				c1DidrunBugsPath,
				`${syntheticC1DidrunBugs}\nC1B is sealed and strict-clean.\n`,
			),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "C1B self-receipt claim",
		},
		{
			name: "C1B didrun live-session block removal",
			overrides: mutateC1Overrides(
				c1DidrunBugsPath,
				syntheticC1DidrunBugs.replace("<!-- P07B-C-C1-DIDRUN-LIVE:START -->", "<!-- C1 LIVE OMITTED -->"),
			),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "C1B didrun live-session block",
		},
		{
			name: "C1B didrun disclosure removal",
			overrides: mutateC1Overrides(
				c1DidrunBugsPath,
				syntheticC1DidrunBugs.replace(c1Disclosures[4], "Ledger manifest disclosure omitted."),
			),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "C1B didrun source disclosure",
		},
		{
			name: "C1B handoff HTML digest mismatch",
			overrides: mutateC1Overrides(
				"docs/HANDOFF_MODE_C.md",
				syntheticC1Handoff.replace(syntheticC1Receipt.evidence.html_report.sha256, "f".repeat(64)),
			),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "C1B source evidence disclosure",
		},
		{
			name: "synthetic C0B receipt/status disagreement",
			overrides: new Map([
				[statusPath, syntheticStatus],
				[receiptDeclarationPath, syntheticReceiptText.replace("TREE-EXACT", "UNRECEIPTED")],
				["docs/HANDOFF_MODE_C.md", syntheticHandoff],
			]),
			receiptAuthority: syntheticAuthority,
			expect: "invalid receipt declaration",
		},
		{
			name: "synthetic C0B wrong Git parent",
			overrides: new Map([
				[statusPath, syntheticStatus],
				[receiptDeclarationPath, syntheticReceiptText.replace(`"${syntheticReceipt.source_commit}"`, `"${"c".repeat(40)}"`)],
				["docs/HANDOFF_MODE_C.md", syntheticHandoff],
			]),
			receiptAuthority: syntheticAuthority,
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "synthetic C0B wrong Git tree",
			overrides: new Map([
				[statusPath, syntheticStatus],
				[receiptDeclarationPath, syntheticReceiptText.replace(`"${syntheticReceipt.source_tree}"`, `"${"d".repeat(40)}"`)],
				["docs/HANDOFF_MODE_C.md", syntheticHandoff],
			]),
			receiptAuthority: syntheticAuthority,
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "synthetic C0B handoff mismatch",
			overrides: new Map([
				[statusPath, syntheticStatus],
				[receiptDeclarationPath, syntheticReceiptText],
				["docs/HANDOFF_MODE_C.md", syntheticHandoff.replace(
					`C0a source commit \`${syntheticReceipt.source_commit}\`, tree \`${syntheticReceipt.source_tree}\`.`,
					`C0a source commit \`${syntheticReceipt.source_commit}\`, tree \`${"e".repeat(40)}\`.`,
				)],
			]),
			receiptAuthority: syntheticAuthority,
			expect: "C0B source identity",
		},
	];

	for (const testCase of cases) {
		const overrides = testCase.overrides ?? new Map([[testCase.path, testCase.value]]);
		const errors = await checkPlan(
			repositoryRoot,
			overrides,
			testCase.receiptAuthority,
			testCase.c1ReceiptAuthority,
		);
		if (!errors.some((error) => error.includes(testCase.expect))) {
			throw new Error(`P07B-C C1 evolved plan checker self-test false negative: ${testCase.name}`);
		}
	}

	console.log(`P07B-C C1 evolved plan checker self-test passed: ${cases.length + localEvidenceMutations + receiptModePhaseMutations} authority, structure, local-evidence, phase-isolation, semantic-projection, and allowlist mutations rejected`);
}

async function main() {
	const mode = process.argv[2];
	if (mode === "--self-test") {
		if (process.argv.length !== 3) throw new Error("usage: check-p07b-c-plan.mjs [--self-test|--verify-sealed-c1-local-evidence|--verify-c1-local-evidence]");
		await runSelfTest();
		return;
	}
	if (mode === "--verify-sealed-c1-local-evidence") {
		if (process.argv.length !== 3) throw new Error("usage: check-p07b-c-plan.mjs [--self-test|--verify-sealed-c1-local-evidence|--verify-c1-local-evidence]");
		await requireC1ReceiptDeclarationPhase(repositoryRoot, false);
		const planErrors = await checkPlan();
		if (planErrors.length > 0) {
			throw new Error(`P07B-C sealed C1 local-evidence precondition failed:\n${planErrors.join("\n")}`);
		}
		const evidenceErrors = await verifyC1LocalEvidence(repositoryRoot, {
			source_commit: sealedC1Identity.commit,
			source_tree: sealedC1Identity.tree,
			evidence: structuredClone(expectedC1ReceiptEvidence),
		});
		if (evidenceErrors.length > 0) {
			throw new Error(`P07B-C sealed C1 local-evidence verification failed:\n${evidenceErrors.join("\n")}`);
		}
		console.log("P07B-C sealed C1 local evidence passed: 20 flat SHA-256 capture blobs plus 57 bounded Git SHA-1 loose objects match the 77-object combined manifest and exact referenced-blob set; this local snapshot is not a portable didrun-format or strict-verification guarantee");
		return;
	}
	if (mode === "--verify-c1-local-evidence") {
		if (process.argv.length !== 3) throw new Error("usage: check-p07b-c-plan.mjs [--self-test|--verify-sealed-c1-local-evidence|--verify-c1-local-evidence]");
		const receiptText = await readC1ReceiptDeclarationSnapshot(repositoryRoot);
		const planErrors = await checkPlan(repositoryRoot, new Map([[c1ReceiptDeclarationPath, receiptText]]));
		if (planErrors.length > 0) {
			throw new Error(`P07B-C C1 local-evidence precondition failed:\n${planErrors.join("\n")}`);
		}
		const receipt = JSON.parse(receiptText);
		const evidenceErrors = await verifyC1LocalEvidence(repositoryRoot, receipt);
		if (evidenceErrors.length > 0) {
			throw new Error(`P07B-C C1 local-evidence verification failed:\n${evidenceErrors.join("\n")}`);
		}
		console.log("P07B-C C1 local evidence passed: HTML plus the ignored 19-event sealed/20-event archived ledger snapshot match the closed receipt declaration; this local snapshot is not portable strict authority");
		return;
	}
	if (mode !== undefined) throw new Error("usage: check-p07b-c-plan.mjs [--self-test|--verify-sealed-c1-local-evidence|--verify-c1-local-evidence]");

	const errors = await checkPlan();
	if (errors.length > 0) {
		console.error("P07B-C C1 evolved plan check failed:");
		for (const error of errors) console.error(`- ${error}`);
		process.exitCode = 1;
		return;
	}

	console.log("P07B-C C1 evolved plan check passed: C0 authority/receipts and the inert model/schema/example projection remain coherent; this gate confers no store or runtime authority");
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
	try {
		await main();
	} catch (error) {
		console.error(error.message);
		process.exitCode = 1;
	}
}
