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
const ABSENT_FIXTURE_PATH = Symbol("ABSENT_FIXTURE_PATH");

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
	"research/deep-dive/p07b-c/11-c3-runtime-epoch-audit.md": "# C3 runtime and host-epoch audit",
});

const promptHeadings = Object.freeze([
	"# C0 — lock the corrected P07B-C authority contract",
	"# C1 — implement strict inert semantic algebra",
	"# C2 — add nonhead persistence and private interlock substrate",
	"# C3P — correct the runtime epoch before live authority",
	"# C3 — publish one exact pre-spawn target",
	"# C4 — close a CLI candidate-profile run and publish its classification",
	"# C5 — add HTTP and complete standalone-scope evidence",
	"# C6 — cumulative closure, expert surfaces, and receipts",
]);

const authorityDeclarationPath = "spec/verification/p07b-c-c0-authority.json";
const receiptDeclarationPath = "spec/verification/p07b-c-c0-receipt.json";
const c1ReceiptDeclarationPath = "spec/verification/p07b-c-c1-receipt.json";
const c2ReceiptDeclarationPath = "spec/verification/p07b-c-c2-receipt.json";
const c3pReceiptDeclarationPath = "spec/verification/p07b-c-c3p-receipt.json";
const statusPath = "docs/status/P07B-C-C0-AUTHORITY.md";
const c1MaintenanceStatusPath = "docs/status/P07B-C-C1-CUMULATIVE-MAINTENANCE.md";
const c1VerificationStatusPath = "docs/status/P07B-C-VERIFICATION-THROUGHPUT.md";
const c1EvidenceMaintenanceStatusPath = "docs/status/P07B-C-C1-LOCAL-EVIDENCE-MAINTENANCE.md";
const c1StatusPath = "docs/status/P07B-C-C1-SEMANTICS.md";
const c2StatusPath = "docs/status/P07B-C-C2-PERSISTENCE.md";
const c2MaintenanceStatusPath = "docs/status/P07B-C-C2-RECEIPT-PHASE-MAINTENANCE.md";
const c3pStatusPath = "docs/status/P07B-C-C3P-RUNTIME-EPOCH.md";
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
const sealedC1BIdentity = Object.freeze({
	commit: "46c48507fe998ea04e121470d6aa8ba0b38b3dae",
	tree: "6a6ee49048c2639d102641cecab4e6d99f31377e",
});
const sealedC2Identity = Object.freeze({
	commit: "19c90786d9832e21ec86daea96ca514cc7304836",
	tree: "1557f1e26762d717327a2ec770f8153daa0baf3e",
});
const sealedC2BIdentity = Object.freeze({
	commit: "ab5e2dd5b66702a1d1cd13eb0047b57a48469487",
	tree: "02fd396858ca41cff6d0ee561dce7ee65a3f94d0",
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
		"C1 and C2 grades enter tracked authority only through their separately sealed receipt descendants",
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
		"through C2B sealed; C3P runtime-epoch correction active",
		"private interlock is the sole mutable operational exception",
		"C3 can rejoin the sealed inert C1/C2 semantics",
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
		"C3P defines the pre-C3 runtime-epoch prerequisite",
		"C3P prerequisite declared",
	],
	"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md": [
		"C0 → C1 → C2 → C2B → C3P → C3PB → C3 → C3B → C4 → C5 → C6",
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
		"C1 implements strict inert target/run/execution semantics",
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
		"C2 then composes the exact C2 boundary over B and C1 (`C2 -> B -> C1`)",
		"eight byte-exact schema-valid/runtime-invalid cases",
		"GOFLAGS=-mod=readonly -buildvcs=false -p=1",
		"sha256:eb51c52657591df8faa5d3c4737291c071e40c07eb9c32bde88af1a36e68bf76",
		"sha256:6032cc515978947743e1bfea72340e1065d77473ccc27b3b0fc036ebd4ceb2a5",
		"Direct build, vet, and the exact general-package complement override only package jobs to `-p=2`",
		"focused Go repetition runner",
		"`RECEIPT_RECONCILIATION` is admitted only for an exact empty-prefix roster",
	],
	"spec/verification/p07b-c-unit-paths.json": [
		"countershape/p07b-c-unit-paths/v4",
		"\"C0A\"",
		"\"C1M\"",
		"\"C1V\"",
		"\"C1E\"",
		"\"C2M\"",
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
		"path must be one staged regular mode-100644 blob",
		"--source-final-gate",
		"--credential-scan",
		"validateExactIndexModes",
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
		"validates didrun's flat SHA-256 capture blobs",
			"P07B-C C1E local-evidence maintenance",
			"later independently sealed, delimited handoff/receipt descendant",
	],
	"spec/verification/p07b-c-unit-paths.json": [
		"countershape/p07b-c-unit-paths/v4",
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
const requiredC2Paths = Object.freeze([
	"docs/ARCHITECTURE.md",
	"docs/HANDOFF_MODE_C.md",
	"docs/SEMANTICS.md",
	"docs/STATE_MACHINES.md",
	"docs/THREAT_MODEL.md",
	"docs/VERIFICATION.md",
	"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md",
	"docs/status/P07B-C-C2-PERSISTENCE.md",
	"internal/store/execution_interlock.go",
	"internal/store/execution_interlock_test.go",
	"internal/store/nonhead_contract.go",
	"internal/store/nonhead_contract_test.go",
	"internal/store/object_store.go",
	"internal/store/object_store_test.go",
	"internal/store/private_contract_run.go",
	"internal/store/private_contract_run_test.go",
	"internal/store/public_api_test.go",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-b-architecture-selftest.mjs",
	"tools/check-p07b-b-architecture.mjs",
	"tools/check-p07b-c-architecture-selftest.mjs",
	"tools/check-p07b-c-architecture.mjs",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
	"tools/verify-current-selftest.mjs",
	"tools/verify-current.mjs",
]);
const requiredC2Digest = "sha256:a463e4e9ad9830cb420bbad1da494c26d55e3f68b983abfd683a0c8cb1c3a522";
const c2ClaimLabels = Object.freeze([
	"P07B C2 exact typed nonhead persistence profile",
	"P07B C2 interlock StartClaim and receipt-backed clear profile",
	"P07B C2 private evidence lifecycle profile",
	"P07B C2 exported authority and discovery absence",
	"P07B C2 focused store race suite",
	"P07B C2 cumulative architecture boundary",
	"P07B C2 cumulative architecture defensive self-test",
	"P07B C2 predecessor B compatibility",
	"P07B C2 predecessor B defensive self-test",
	"P07B C2 cumulative verifier self-test",
	"P07B C2 cumulative verification",
	"P07B C2 exact 26-path staged scope and diff integrity",
	"P07B C2 scoped staged credential-pattern scan",
	"P07B C2 preceding didrun chain integrity",
]);
const c2ClaimTypes = Object.freeze([
	...Array(11).fill("tests-pass"),
	"command-succeeded",
	"command-succeeded",
	"command-succeeded",
]);
const c2FinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c2-final");
const c2HermeticArgvPrefix = Object.freeze([
	"/usr/bin/env", "-i",
	`HOME=${resolve(c2FinalRunRoot, "home")}`,
	`TMPDIR=${resolve(c2FinalRunRoot, "tmp")}`,
	`GOTMPDIR=${resolve(c2FinalRunRoot, "gotmp")}`,
	`GOCACHE=${resolve(c2FinalRunRoot, "gocache")}`,
	`GOPATH=${resolve(c2FinalRunRoot, "gopath")}`,
	`GOMODCACHE=${resolve(c2FinalRunRoot, "gomodcache")}`,
	"GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off",
	"GOFLAGS=-mod=readonly -buildvcs=false -p=1", "CGO_ENABLED=1", "GOMAXPROCS=2",
	"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "PATH=/opt/homebrew/bin:/usr/bin:/bin",
	"COUNTERSHAPE_GO=/opt/homebrew/bin/go", "COUNTERSHAPE_CC=/usr/bin/clang", "COUNTERSHAPE_CXX=/usr/bin/clang++",
	"CC=/usr/bin/clang", "CXX=/usr/bin/clang++",
]);
const c2HermeticArgv = (...tail) => Object.freeze([...c2HermeticArgvPrefix, ...tail]);
const c2ExpectedClaimArgv = Object.freeze([
	c2HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture.mjs", "--run-go-json", "c2-nonhead-persistence"),
	c2HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture.mjs", "--run-go-json", "c2-interlock"),
	c2HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture.mjs", "--run-go-json", "c2-private-evidence"),
	c2HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture.mjs", "--run-go-json", "c2-public-surface"),
	c2HermeticArgv("/opt/homebrew/bin/go", "test", "-mod=readonly", "-buildvcs=false", "-p=1", "-race", "-count=1", "-run", "^TestC2", "./internal/store"),
	c2HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture.mjs", "--c2"),
	c2HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture-selftest.mjs", "--c2"),
	c2HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-b-architecture.mjs"),
	c2HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-b-architecture-selftest.mjs"),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C2", "--source-final-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C2", "--credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c2-preseal-ledger"]),
]);
const requiredC2AddedPaths = new Set([
	"docs/status/P07B-C-C2-PERSISTENCE.md",
	"internal/store/execution_interlock.go",
	"internal/store/execution_interlock_test.go",
	"internal/store/nonhead_contract.go",
	"internal/store/nonhead_contract_test.go",
	"internal/store/private_contract_run.go",
	"internal/store/private_contract_run_test.go",
]);
const requiredC2MaintenancePaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/VERIFICATION.md",
	"docs/status/P07B-C-C2-RECEIPT-PHASE-MAINTENANCE.md",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
]);
const requiredC2MaintenanceDigest = "sha256:75dcb6a88a9faec8aa89cfbe67c0870e88d95709a1bf07e2fb494e92362688cd";
const requiredC2MaintenanceRosterText = requiredC2MaintenancePaths
	.map((path, index) => `${index === requiredC2MaintenancePaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC2MaintenanceDeclaration = `C2M owns exactly these ${requiredC2MaintenancePaths.length} paths: ${requiredC2MaintenanceRosterText}. Their sorted-newline roster digest is \`${requiredC2MaintenanceDigest}\`.`;
const c2MaintenanceClaimLabels = Object.freeze([
	"P07B-C C2M receipt-phase plan coherence",
	"P07B-C C2M pre-receipt and receipt-phase checker self-test",
	"P07B-C C2M unit-scope defensive self-test",
	"P07B-C C2M cumulative verifier self-test",
	"P07B-C C2M cumulative verification",
	"P07B-C C2M exact seven-path staged scope and diff integrity",
	"P07B-C C2M scoped staged credential-pattern scan",
	"P07B-C C2M preceding didrun chain integrity",
]);
const c2MaintenanceClaimTypes = Object.freeze([
	...Array(5).fill("tests-pass"),
	...Array(3).fill("command-succeeded"),
]);
const c2MaintenanceExpectedClaimArgv = Object.freeze([
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C2M", "--source-final-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C2M", "--credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c2m-preseal-ledger"]),
]);
const requiredC2MaintenanceText = Object.freeze({
	[c2MaintenanceStatusPath]: [
		"`DEFECT_REPAIR`",
		"receipt-present plan self-test",
		"four pre-receipt-only hostile mutations",
		".didrun-history/2026-07-19-p07b-c-c2b-build-loop-pre-maintenance/.didrun/",
		".didrun-history/2026-07-19-p07b-c-c2m-build-loop/.didrun/",
		"229a60c0630e296b67faa3c81b91fac73f9f63ff",
		"19c90786d9832e21ec86daea96ca514cc7304836",
		requiredC2MaintenanceDeclaration,
		...c2MaintenanceClaimLabels.map((label) => `\`${label}\``),
	],
	"docs/VERIFICATION.md": [
		"## C2 receipt-phase checker maintenance",
		"synthetic, `readFile`-equivalent `ENOENT` signal",
		"receipt-present working-tree baseline",
		"receipt-present synthetic full-plan baseline",
		requiredC2MaintenanceDeclaration,
	],
	"docs/HANDOFF_MODE_C.md": [
		"C2M receipt-phase checker maintenance is the active `SOURCE_FULL` unit",
		requiredC2MaintenanceDigest,
		"229a60c0630e296b67faa3c81b91fac73f9f63ff",
		".didrun-history/2026-07-19-p07b-c-c2b-build-loop-pre-maintenance/.didrun/",
	],
	"docs/PROMPT_PACK.md": [
		"P07B-C C2M receipt-phase checker maintenance",
		requiredC2MaintenanceDigest,
		"C2B later reconciled the source evidence",
	],
	"tools/check-p07b-c-plan.mjs": [
		"ABSENT_FIXTURE_PATH",
		"syntheticC2SealedSourcePhaseFixture",
		"runC2SealedSourcePhaseSelfTest",
		"--verify-c2m-preseal-ledger",
	],
	"tools/check-p07b-c-unit-scope.mjs": [
		"\"C2M\"",
		"declared exact-roster SOURCE_FULL unit",
	],
});
const requiredC2BPaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/status/P07B-C-C2-PERSISTENCE.md",
	"spec/verification/p07b-c-c2-receipt.json",
]);
const requiredC2BDigest = "sha256:5e30e009bb29672d585ece9893bd7c8db198915335a73ded3f3c790ad18f665b";
const requiredC2BReceiptClaims = Object.freeze([
	Object.freeze({ label: "P07B-C C2B source receipt reconciliation", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C2B receipt checker defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C2B declared local source-evidence snapshot match", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C2B receipt-only Go build", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C2B exact three-path staged scope and diff integrity", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C2B scoped staged credential-pattern scan", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C2B preceding didrun chain integrity", type: "command-succeeded" }),
]);
const requiredC3PPaths = Object.freeze([
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
	"docs/status/P07B-C-C3P-RUNTIME-EPOCH.md",
	"internal/contractexec/model/codec_test.go",
	"internal/contractexec/model/target.go",
	"research/deep-dive/p07b-c/11-c3-runtime-epoch-audit.md",
	"spec/examples/v1/contract-execution-target.valid.json",
	"spec/examples/v1/contract-execution.valid.json",
	"spec/examples/v1/finalized-contract-run.valid.json",
	"spec/schema/v1/contract-execution-target.schema.json",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-c3p-receipt.mjs",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
	"tools/validate-planning.mjs",
	"tools/verify-current-selftest.mjs",
	"tools/verify-current.mjs",
]);
const requiredC3PDigest = "sha256:8b047704df047cb96f1c5d19418c4cfe8502e4b4746bb4f24c0851dac8d018c0";
const c3pClaimLabels = Object.freeze([
	"P07B-C C3P planning and scope coherence",
	"P07B-C C3P legacy clock-derived profile refusal",
	"P07B-C C3P joined semantic example regeneration",
	"P07B-C C3P contract model profile",
	"P07B-C C3P cumulative C1 architecture compatibility",
	"P07B-C C3P architecture defensive self-test",
	"P07B-C C3P unit-scope defensive self-test",
	"P07B-C C3P cumulative verifier self-test",
	"P07B-C C3P cumulative verification",
	"P07B-C C3P exact twenty-five-path staged scope and diff integrity",
	"P07B-C C3P scoped staged credential-pattern scan",
	"P07B-C C3P preceding didrun chain integrity",
]);
const c3pClaimTypes = Object.freeze([
	...Array(9).fill("tests-pass"),
	...Array(3).fill("command-succeeded"),
]);
const c3pFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c3p-final");
const c3pHermeticArgvPrefix = Object.freeze([
	"/usr/bin/env", "-i",
	`HOME=${resolve(c3pFinalRunRoot, "home")}`,
	`TMPDIR=${resolve(c3pFinalRunRoot, "tmp")}`,
	`GOTMPDIR=${resolve(c3pFinalRunRoot, "gotmp")}`,
	`GOCACHE=${resolve(c3pFinalRunRoot, "gocache")}`,
	`GOPATH=${resolve(c3pFinalRunRoot, "gopath")}`,
	`GOMODCACHE=${resolve(c3pFinalRunRoot, "gomodcache")}`,
	"GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off",
	"GOFLAGS=-mod=readonly -buildvcs=false -p=1", "CGO_ENABLED=1", "GOMAXPROCS=2",
	"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "PATH=/opt/homebrew/bin:/usr/bin:/bin",
	"COUNTERSHAPE_GO=/opt/homebrew/bin/go", "COUNTERSHAPE_CC=/usr/bin/clang", "COUNTERSHAPE_CXX=/usr/bin/clang++",
	"CC=/usr/bin/clang", "CXX=/usr/bin/clang++",
]);
const c3pHermeticArgv = (...tail) => Object.freeze([...c3pHermeticArgvPrefix, ...tail]);
const c3pExpectedClaimArgv = Object.freeze([
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/validate-planning.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/generate-p07-planning-example.mjs", "--check"]),
	c3pHermeticArgv("/opt/homebrew/bin/go", "test", "-mod=readonly", "-buildvcs=false", "-p=1", "-count=1", "-run", "^TestTargetPrimitiveBoundsAndDerivedRelations$", "./internal/contractexec/model"),
	c3pHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture.mjs"),
	c3pHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture-selftest.mjs"),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3P", "--source-final-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3P", "--credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-c3p-receipt.mjs", "--verify-preseal-ledger"]),
]);
const requiredC3PAddedPaths = new Set([
	"docs/status/P07B-C-C3P-RUNTIME-EPOCH.md",
	"research/deep-dive/p07b-c/11-c3-runtime-epoch-audit.md",
	"tools/check-p07b-c-c3p-receipt.mjs",
]);
const requiredC3PBPaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/status/P07B-C-C3P-RUNTIME-EPOCH.md",
	"spec/verification/p07b-c-c3p-receipt.json",
]);
const requiredC3PBDigest = "sha256:531cbe600cf2747752749890ea9a15c3d0126e498da6838d4f6faeb1a0d7e885";
const requiredC3PBReceiptClaims = Object.freeze([
	Object.freeze({ label: "P07B-C C3PB source receipt reconciliation", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C3PB receipt checker defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C3PB declared local source-evidence snapshot match", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C3PB receipt-only Go build", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C3PB exact three-path staged scope and diff integrity", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C3PB scoped staged credential-pattern scan", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C3PB preceding didrun chain integrity", type: "command-succeeded" }),
]);
const c3pbFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c3pb-final");
const c3pbHermeticArgvPrefix = Object.freeze([
	"/usr/bin/env", "-i",
	`HOME=${resolve(c3pbFinalRunRoot, "home")}`,
	`TMPDIR=${resolve(c3pbFinalRunRoot, "tmp")}`,
	`GOTMPDIR=${resolve(c3pbFinalRunRoot, "gotmp")}`,
	`GOCACHE=${resolve(c3pbFinalRunRoot, "gocache")}`,
	`GOPATH=${resolve(c3pbFinalRunRoot, "gopath")}`,
	`GOMODCACHE=${resolve(c3pbFinalRunRoot, "gomodcache")}`,
	"GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off",
	"GOFLAGS=-mod=readonly -buildvcs=false -p=1", "CGO_ENABLED=1", "GOMAXPROCS=2",
	"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "PATH=/opt/homebrew/bin:/usr/bin:/bin",
	"CC=/usr/bin/clang", "CXX=/usr/bin/clang++",
]);
const c3pbExpectedClaimArgv = Object.freeze([
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-c3p-receipt.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-c3p-receipt.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-c3p-receipt.mjs", "--verify-local-evidence"]),
	Object.freeze([...c3pbHermeticArgvPrefix, "/opt/homebrew/bin/go", "build", "-mod=readonly", "-buildvcs=false", "-p=1", "./..."]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-c3p-receipt.mjs", "--verify-c3pb-staged"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-c3p-receipt.mjs", "--verify-c3pb-credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-c3p-receipt.mjs", "--verify-c3pb-preseal-ledger"]),
]);
const requiredC3Paths = Object.freeze([
	"docs/ARCHITECTURE.md", "docs/CLAIM_VOCABULARY.md", "docs/CONCEPT_BRIEF.md", "docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md", "docs/SEMANTICS.md", "docs/STATE_MACHINES.md", "docs/THREAT_MODEL.md",
	"docs/VERIFICATION.md", "docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md", "docs/status/P07B-C-C3-TARGET.md",
	"internal/contractexec/target.go", "internal/contractexec/target_test.go", "internal/gitobj/single_target.go",
	"internal/gitobj/single_target_test.go", "internal/hostepoch/epoch.go", "internal/hostepoch/epoch_darwin_test.go",
	"internal/hostepoch/epoch_test.go", "internal/hostepoch/public_api_test.go", "internal/hostepoch/source_darwin_cgo.go",
	"internal/hostepoch/source_unsupported.go", "internal/noderuntime/identity_darwin.go", "internal/noderuntime/probe_darwin.go",
	"internal/noderuntime/probe_darwin_test.go", "internal/noderuntime/public_api_test.go", "internal/noderuntime/runtime.go",
	"internal/noderuntime/runtime_darwin_test.go", "internal/noderuntime/runtime_test.go", "internal/noderuntime/unsupported.go",
	"internal/store/nonhead_contract.go", "internal/store/nonhead_contract_test.go", "internal/store/public_api_test.go",
	"spec/verification/p07b-c-c3-predecessors.json", "spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-architecture-selftest.mjs", "tools/check-p07b-c-architecture.mjs", "tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs", "tools/verify-current-selftest.mjs", "tools/verify-current.mjs",
]);
const requiredC3Digest = "sha256:c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb";
const requiredC3BPaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/status/P07B-C-C3-TARGET.md",
	"spec/verification/p07b-c-c3-receipt.json",
]);
const requiredC3BDigest = "sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67";
const requiredC3BReceiptClaims = Object.freeze([
	Object.freeze({ label: "P07B-C C3B source receipt reconciliation", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C3B receipt checker defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C3B declared local source-evidence snapshot match", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C3B receipt-only Go build", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C3B exact three-path staged scope and diff integrity", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C3B scoped staged credential-pattern scan", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C3B preceding didrun chain integrity", type: "command-succeeded" }),
]);
const requiredC3PText = Object.freeze({
	"docs/ARCHITECTURE.md": ["DARWIN_KERN_BOOTSESSIONUUID_V1", "kern.bootsessionuuid", "never falls back to `kern.boottime`"],
	"docs/CLAIM_VOCABULARY.md": ["opaque `DARWIN_KERN_BOOTSESSIONUUID_V1` authority", "OS-reported boot-session UUID"],
	"docs/CONCEPT_BRIEF.md": ["DARWIN_KERN_BOOTSESSIONUUID_V1", "reads Darwin `kern.bootsessionuuid` twice", "fake one-member comparison worlds"],
	"docs/HANDOFF_MODE_C.md": [requiredC3PDigest, requiredC3PBDigest, requiredC3Digest, requiredC3BDigest, sealedC2BIdentity.commit, sealedC2BIdentity.tree],
	"docs/PROMPT_PACK.md": ["C3P defines the pre-C3 runtime-epoch prerequisite", requiredC3PDigest, "DARWIN_KERN_BOOTSESSIONUUID_V1"],
	"docs/SEMANTICS.md": ["The exact boot binding is profile `DARWIN_KERN_BOOTSESSIONUUID_V1`", "The legacy `DARWIN_KERN_BOOTTIME_V1` profile is invalid"],
	"docs/STATE_MACHINES.md": ["The live boot checkpoint has one exact v1 profile: `DARWIN_KERN_BOOTSESSIONUUID_V1`", "caller-authored boot value"],
	"docs/THREAT_MODEL.md": ["The only admitted boot profile is `DARWIN_KERN_BOOTSESSIONUUID_V1`", "`/usr/sbin/sysctl`"],
	"docs/VERIFICATION.md": ["## C3P stable boot-session prerequisite", requiredC3PDigest, requiredC3PBDigest, requiredC3Digest, requiredC3BDigest],
	"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md": ["# C3P — correct the runtime epoch before live authority", requiredC3PDigest, requiredC3PBDigest, requiredC3Digest, requiredC3BDigest, "implement both absent-receipt and receipt-present C3B validation"],
	[c3pStatusPath]: [requiredC3PDigest, requiredC3PBDigest, requiredC3Digest, requiredC3BDigest, ...c3pClaimLabels.map((label) => `\`${label}\``)],
	"research/deep-dive/p07b-c/11-c3-runtime-epoch-audit.md": ["DARWIN_KERN_BOOTSESSIONUUID_V1", "pre-authority semantic correction", "exact forty-path roster"],
	"tools/check-p07b-c-c3p-receipt.mjs": ["--verify-preseal-ledger", "--verify-local-evidence", "--self-test"],
});

async function readBytes(root, path, overrides) {
	if (overrides.has(path)) {
		const value = overrides.get(path);
		if (value === ABSENT_FIXTURE_PATH) {
			const error = new Error(`synthetic absent fixture path: ${path}`);
			error.code = "ENOENT";
			throw error;
		}
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
		&& targetSchema.properties?.boot_session_binding?.properties?.profile?.const === "DARWIN_KERN_BOOTSESSIONUUID_V1"
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
			&& targetExample.boot_session_binding?.profile === "DARWIN_KERN_BOOTSESSIONUUID_V1"
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

function rejectReceiptSelfClaims(body, path, unit, errors) {
	const patterns = [
		new RegExp("\\b" + unit + " (?:receipt )?commit `[0-9a-f]{40}`", "iu"),
		new RegExp("\\b" + unit + " (?:receipt )?tree `[0-9a-f]{40}`", "iu"),
		new RegExp("\\b" + unit + "[^\\n]*(?:`?tree-exact`?|claims recorded-exact|note-present|strict exit(?:\\s*:\\s*|\\s+)`?0`?|(?:is|was|has been) (?:sealed|strict-clean|verified)|passed verification)", "iu"),
	];
	for (const pattern of patterns) {
		if (pattern.test(body)) errors.push(`${path}: ${unit} self-receipt claim is forbidden`);
	}
}

function rejectC1BSelfReceiptClaims(body, path, errors) {
	rejectReceiptSelfClaims(body, path, "C1B", errors);
}

function rejectC2BSelfReceiptClaims(body, path, errors) {
	rejectReceiptSelfClaims(body, path, "C2B", errors);
}

function rejectC3PBSelfReceiptClaims(body, path, errors) {
	rejectReceiptSelfClaims(body, path, "C3PB", errors);
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

async function requireReceiptDeclarationPhase(root, declarationPath, expectedPresent, phase) {
	const absolute = admittedLocalPath(root, declarationPath);
	let status;
	try {
		status = await lstat(absolute);
	} catch (error) {
		if (error.code !== "ENOENT") throw error;
		if (expectedPresent) throw new Error(`${declarationPath} must be present in ${phase} declaration-bound mode`);
		return;
	}
	if (!status.isFile() || status.isSymbolicLink()) {
		throw new Error(`${declarationPath} must be a regular non-symlink file`);
	}
	if (!expectedPresent) throw new Error(`${declarationPath} must be absent in ${phase} pre-receipt mode`);
}

async function requireC1ReceiptDeclarationPhase(root, expectedPresent) {
	await requireReceiptDeclarationPhase(root, c1ReceiptDeclarationPath, expectedPresent, "C1");
}

async function readReceiptDeclarationSnapshot(root, declarationPath, phase) {
	await requireReceiptDeclarationPhase(root, declarationPath, true, phase);
	const absolute = await admittedExistingLocalPath(root, declarationPath);
	const bytes = await readRegularNoFollow(absolute, c1ReceiptSnapshotLimit);
	return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
}

async function readC1ReceiptDeclarationSnapshot(root) {
	return readReceiptDeclarationSnapshot(root, c1ReceiptDeclarationPath, "C1");
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

async function verifyLocalEvidence(root, receipt, claimLabels, claimTypes, expectedArgv = null) {
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
		for (let index = 0; index < claimLabels.length; index += 1) {
			const claim = claims[index];
			if (claim?.label !== claimLabels[index] || claim?.ctype !== claimTypes[index] ||
				claim?.declared_at_index !== index || !isDeepStrictEqual(claim?.event_indices, [index]) ||
				!isDeepStrictEqual(claim?.pathspecs, [])) {
				errors.push(`local ledger claim ${index + 1}`);
			}
			if (expectedArgv !== null && !isDeepStrictEqual(sessions[index]?.event?.argv, expectedArgv[index])) {
				errors.push(`local ledger event ${index + 1} argv`);
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

async function verifyC1LocalEvidence(root, receipt) {
	return verifyLocalEvidence(root, receipt, c1ClaimLabels, c1ClaimTypes);
}

async function writeJSONLines(path, values) {
	await writeFile(path, `${values.map((value) => JSON.stringify(value)).join("\n")}\n`, "utf8");
}

async function buildLocalEvidenceFixture(root, {
	claimLabels = c1ClaimLabels,
	claimTypes = c1ClaimTypes,
	claimArgv = null,
	sourceCommit = sealedC1Identity.commit,
	sourceTree = sealedC1Identity.tree,
	archiveSessionEventCount = claimLabels.length + 1,
} = {}) {
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
	for (let index = 0; index < archiveSessionEventCount; index += 1) {
		sessions.push({
			index,
			prev_hash: index === 0 ? "0".repeat(64) : sessions[index - 1].entry_hash,
			entry_hash: sha256(Buffer.from(`fixture-entry-${index}`, "utf8")),
			event: {
				argv: claimArgv === null || index >= claimArgv.length ? ["fixture"] : [...claimArgv[index]],
				coverage: "complete",
				exit_code: 0,
				stdout_blob: objectDigest,
				stderr_blob: objectDigest,
				transcript_blob: null,
			},
		});
	}
	const claims = claimLabels.map((label, index) => ({
		label,
		ctype: claimTypes[index],
		declared_at_index: index,
		event_indices: [index],
		pathspecs: [],
	}));
	const seals = [{
		claims_watermark: claimLabels.length,
		commit: sourceCommit,
		tree: sourceTree,
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
					sealed_event_count: claimLabels.length,
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
		source_commit: sourceCommit,
		source_tree: sourceTree,
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

function parseRawSourceDiff(text) {
	if (typeof text !== "string" || (text.length > 0 && !text.endsWith("\0"))) {
		throw new Error("raw source diff omitted its final NUL delimiter");
	}
	const tokens = text.length === 0 ? [] : text.slice(0, -1).split("\0");
	if (tokens.length % 2 !== 0) throw new Error("raw source diff has an unpaired header/path");
	const entries = [];
	for (let index = 0; index < tokens.length; index += 2) {
		const match = /^:([0-7]{6}) ([0-7]{6}) ([0-9a-f]{40}|[0-9a-f]{64}) ([0-9a-f]{40}|[0-9a-f]{64}) ([A-Z])$/u.exec(tokens[index]);
		const path = tokens[index + 1];
		if (!match || typeof path !== "string" || path.length === 0 || /[\u0000-\u001f\u007f]/u.test(path)) {
			throw new Error("raw source diff row is malformed");
		}
		entries.push(Object.freeze({
			old_mode: match[1], new_mode: match[2], old_oid: match[3], new_oid: match[4], status: match[5], path,
		}));
	}
	return entries.sort((left, right) => Buffer.compare(Buffer.from(left.path, "utf8"), Buffer.from(right.path, "utf8")));
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
	const parentText = run(["show", "-s", "--format=%P", receipt.source_commit]).stdout.trim();
	const parents = parentText.length === 0 ? [] : parentText.split(" ");
	const parent = parents[0] ?? "";
	const sourceDiff = parent === "" ? [] : parseRawSourceDiff(run([
		"diff-tree", "--no-commit-id", "--raw", "-r", "-z", "--no-renames", "--full-index", parent, commit, "--",
	]).stdout);
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
		parents,
		source_diff: sourceDiff,
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

function validateC2ReceiptEvidence(receipt) {
	const errors = [];
	const evidence = receipt.evidence;
	if (!evidence || typeof evidence !== "object" || Array.isArray(evidence) ||
		!isDeepStrictEqual(Object.keys(evidence).sort(), ["git_note", "html_report", "ledger_archive", "seal_redaction"])) {
		return ["local evidence field roster"];
	}
	const html = evidence.html_report;
	const sourcePrefix = typeof receipt.source_commit === "string" ? receipt.source_commit.slice(0, 12) : "";
	if (!html || typeof html !== "object" || Array.isArray(html) ||
		!isDeepStrictEqual(Object.keys(html).sort(), ["authority", "bytes", "path", "sha256"]) ||
		html.path !== `.countershape/evidence/p07b-c-c2-final-${sourcePrefix}.html` ||
		html.authority !== "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS" ||
		!Number.isSafeInteger(html.bytes) || html.bytes <= 0 || html.bytes > 16 * 1024 * 1024 ||
		!(/^[0-9a-f]{64}$/u.test(html.sha256 ?? ""))) {
		errors.push("local HTML declaration");
	}
	const ledger = evidence.ledger_archive;
	const ledgerKeys = [
		"all_files_manifest_sha256", "archive_session_event_count", "authority", "claim_count",
		"claims_jsonl_sha256", "gitignore_sha256", "manifest_format", "object_file_count", "objects_manifest_sha256",
		"path", "seal_count", "sealed_event_count", "seals_jsonl_sha256", "session_log_sha256", "total_bytes", "total_file_count",
	];
	if (!ledger || typeof ledger !== "object" || Array.isArray(ledger) ||
		!isDeepStrictEqual(Object.keys(ledger).sort(), ledgerKeys) ||
		ledger.path !== ".didrun-history/2026-07-19-p07b-c-c2-source/.didrun/" ||
		ledger.authority !== "LOCAL_IGNORED_ARCHIVE_NOT_GIT_AUTHORITY" ||
		ledger.manifest_format !== "sorted-relative-posix-path-tab-size-tab-sha256-newline/v1" ||
		ledger.sealed_event_count !== c2ClaimLabels.length || ledger.claim_count !== c2ClaimLabels.length ||
		ledger.seal_count !== 1 || !Number.isSafeInteger(ledger.archive_session_event_count) ||
		ledger.archive_session_event_count < c2ClaimLabels.length || ledger.archive_session_event_count > c2ClaimLabels.length + 4 ||
		!Number.isSafeInteger(ledger.object_file_count) || ledger.object_file_count <= 0 || ledger.object_file_count > 10_000 ||
		ledger.total_file_count !== ledger.object_file_count + 4 || !Number.isSafeInteger(ledger.total_bytes) ||
		ledger.total_bytes <= 0 || ledger.total_bytes > 1024 * 1024 * 1024 ||
		![ledger.all_files_manifest_sha256, ledger.objects_manifest_sha256, ledger.gitignore_sha256,
			ledger.session_log_sha256, ledger.claims_jsonl_sha256, ledger.seals_jsonl_sha256]
			.every((digest) => /^[0-9a-f]{64}$/u.test(digest ?? ""))) {
		errors.push("local ledger declaration");
	}
	const note = evidence.git_note;
	if (!note || typeof note !== "object" || Array.isArray(note) ||
		!isDeepStrictEqual(Object.keys(note).sort(), ["blob_oid", "body_sha256", "version"]) || note.version !== 1 ||
		!(/^[0-9a-f]{40}$/u.test(note.blob_oid ?? "")) || !(/^[0-9a-f]{64}$/u.test(note.body_sha256 ?? ""))) {
		errors.push("Git note evidence declaration");
	}
	const redaction = evidence.seal_redaction;
	if (!redaction || typeof redaction !== "object" || Array.isArray(redaction) ||
		!isDeepStrictEqual(Object.keys(redaction).sort(), [
			"finding_count", "finding_kinds", "finding_provenance", "structured_scan_provenance", "structured_staged_scan_findings",
		]) || !Number.isSafeInteger(redaction.finding_count) || redaction.finding_count < 0 || redaction.finding_count > 100_000 ||
		!Array.isArray(redaction.finding_kinds) || !isDeepStrictEqual(redaction.finding_kinds, redaction.finding_count === 0 ? [] : ["high-entropy"]) ||
		redaction.finding_provenance !== "LOCAL_TERMINAL_OBSERVATION_NOT_GIT_NOTE" ||
		redaction.structured_staged_scan_findings !== 0 ||
		redaction.structured_scan_provenance !== "SEALED_C2_SUPPORTING_EVENT_12" ||
		(receipt.secrets_override !== (redaction.finding_count > 0))) {
		errors.push("seal redaction evidence declaration");
	}
	return errors;
}

function validateC2ReceiptDeclaration(receipt) {
	const errors = [];
	const keys = [
		"claims", "coverage_complete_events", "coverage_total_events", "evidence", "schema_version",
		"secrets_override", "source_commit", "source_subject", "source_tree", "strict_claims_recorded_exact",
		"strict_claims_total", "strict_exit_code",
	];
	if (!receipt || typeof receipt !== "object" || Array.isArray(receipt) ||
		!isDeepStrictEqual(Object.keys(receipt).sort(), keys)) return ["root field roster"];
	if (receipt.schema_version !== "countershape/p07b-c-c2-receipt/v1") errors.push("schema version");
	if (!/^[0-9a-f]{40}$/u.test(receipt.source_commit ?? "") || /^0+$/u.test(receipt.source_commit ?? "")) errors.push("source commit");
	if (!/^[0-9a-f]{40}$/u.test(receipt.source_tree ?? "") || /^0+$/u.test(receipt.source_tree ?? "")) errors.push("source tree");
	if (receipt.source_subject !== "feat: add contract execution persistence substrate") errors.push("source subject");
	if (typeof receipt.secrets_override !== "boolean") errors.push("secrets override disclosure");
	if (receipt.strict_exit_code !== 0) errors.push("strict exit code");
	if (receipt.strict_claims_recorded_exact !== c2ClaimLabels.length || receipt.strict_claims_total !== c2ClaimLabels.length) {
		errors.push("strict claim counts");
	}
	if (receipt.coverage_total_events !== c2ClaimLabels.length || receipt.coverage_complete_events !== c2ClaimLabels.length) {
		errors.push("coverage counts");
	}
	if (!Array.isArray(receipt.claims) || receipt.claims.length !== c2ClaimLabels.length) {
		errors.push("claim roster length");
	} else {
		for (let index = 0; index < c2ClaimLabels.length; index += 1) {
			const claim = receipt.claims[index];
			if (!claim || typeof claim !== "object" || Array.isArray(claim) ||
				!isDeepStrictEqual(Object.keys(claim).sort(), ["grade", "label", "supporting_event_index", "type"]) ||
				claim.supporting_event_index !== index || claim.type !== c2ClaimTypes[index] ||
				claim.label !== c2ClaimLabels[index] || claim.grade !== "TREE-EXACT") {
				errors.push(`claim ${index + 1}`);
			}
		}
	}
	errors.push(...validateC2ReceiptEvidence(receipt));
	return errors;
}

function validateC2RawClaimArgv(index, argv) {
	return Number.isSafeInteger(index) && index >= 0 && index < c2ExpectedClaimArgv.length &&
		Array.isArray(argv) && argv.length > 0 && argv.length <= 64 &&
		argv.every((argument) => typeof argument === "string" && argument.length > 0 && argument.length <= 4096 &&
			!/[\u0000-\u001f\u007f]/u.test(argument)) &&
		isDeepStrictEqual(argv, c2ExpectedClaimArgv[index]);
}

function validateC2ClaimPreview(index, argv) {
	if (index >= 9) return validateC2RawClaimArgv(index, argv);
	const expected = c2ExpectedClaimArgv[index];
	if (!Array.isArray(argv) || argv.length !== expected.length || argv.some((entry) => typeof entry !== "string")) return false;
	for (let position = 0; position < expected.length; position += 1) {
		if (argv[position] === expected[position]) continue;
		if (position >= 2 && position <= 7) {
			const suffixOffset = expected[position].indexOf(".countershape/");
			const suffix = suffixOffset < 0 ? "" : expected[position].slice(suffixOffset);
			if (suffix !== "" && (argv[position] === `«redacted:high-entropy»${suffix}` ||
				argv[position] === "«redacted:high-entropy».«redacted:high-entropy»")) continue;
		}
		if (position >= 22 && position <= 24 && argv[position] === "«redacted:high-entropy»") continue;
		return false;
	}
	return true;
}

function validateC2SourceDiff(authority) {
	const errors = [];
	if (!isDeepStrictEqual(authority.parents, [sealedC1BIdentity.commit])) errors.push("source parents");
	const entries = authority.source_diff;
	if (!Array.isArray(entries)) return [...errors, "source diff roster"];
	const paths = entries.map((entry) => entry?.path);
	if (new Set(paths).size !== paths.length || !isDeepStrictEqual(paths, requiredC2Paths)) {
		errors.push("source diff exact path roster");
	}
	for (const path of requiredC2Paths) {
		const entry = entries.find((candidate) => candidate?.path === path);
		if (!entry || !isDeepStrictEqual(Object.keys(entry).sort(), ["new_mode", "new_oid", "old_mode", "old_oid", "path", "status"])) {
			errors.push(`source diff row ${path}`);
			continue;
		}
		const objectPattern = /^(?:[0-9a-f]{40}|[0-9a-f]{64})$/u;
		const newObjectValid = objectPattern.test(entry.new_oid) && !/^0+$/u.test(entry.new_oid);
		if (requiredC2AddedPaths.has(path)) {
			if (entry.status !== "A" || entry.old_mode !== "000000" || entry.new_mode !== "100644" ||
				!objectPattern.test(entry.old_oid) || !/^0+$/u.test(entry.old_oid) || !newObjectValid ||
				entry.old_oid.length !== entry.new_oid.length) {
				errors.push(`source diff added row ${path}`);
			}
		} else if (entry.status !== "M" || entry.old_mode !== "100644" || entry.new_mode !== "100644" ||
			!objectPattern.test(entry.old_oid) || /^0+$/u.test(entry.old_oid) || !newObjectValid ||
			entry.old_oid.length !== entry.new_oid.length || entry.old_oid === entry.new_oid) {
			errors.push(`source diff modified row ${path}`);
		}
	}
	return errors;
}

function validateC2ReceiptAuthority(receipt, authority) {
	const errors = [];
	if (!authority || typeof authority !== "object" || Array.isArray(authority)) return ["missing Git/didrun authority"];
	if (authority.commit !== receipt.source_commit) errors.push("source commit does not equal admitted Git commit");
	if (authority.tree !== receipt.source_tree) errors.push("source tree does not equal Git commit tree");
	if (authority.parent !== sealedC1BIdentity.commit) errors.push("source parent");
	errors.push(...validateC2SourceDiff(authority));
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
	if (!Array.isArray(note.claims) || note.claims.length !== c2ClaimLabels.length) {
		errors.push("didrun note claim roster length");
	} else {
		for (let index = 0; index < c2ClaimLabels.length; index += 1) {
			const recorded = note.claims[index];
			const claim = recorded?.claim;
			if (claim?.label !== c2ClaimLabels[index] || claim?.ctype !== c2ClaimTypes[index] ||
				claim?.declared_at_index !== index || !isDeepStrictEqual(claim?.event_indices, [index]) ||
				!isDeepStrictEqual(claim?.pathspecs, []) ||
				recorded?.supporting_event_index !== index || recorded?.grade !== "tree-exact" || recorded?.exit_code !== 0 ||
				recorded?.reason !== "self-stable command ran against the sealed tree" || !isDeepStrictEqual(recorded?.delta, [])) {
				errors.push(`didrun note claim ${index + 1}`);
			}
			if (!validateC2ClaimPreview(index, claim?.argv_preview)) errors.push(`didrun note claim ${index + 1} argv`);
		}
	}
	if (note.coverage?.total_events !== receipt.coverage_total_events ||
		note.coverage?.by_coverage?.complete !== receipt.coverage_complete_events ||
		!isDeepStrictEqual(Object.keys(note.coverage?.by_coverage ?? {}), ["complete"])) {
		errors.push("didrun note exact event coverage");
	}
	return errors;
}

function c2ReceiptDisclosureLines(receipt) {
	const { html_report: html, ledger_archive: ledger, git_note: note, seal_redaction: redaction } = receipt.evidence;
	return Object.freeze([
		`C2 source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`, is sealed, note-present, and strict-clean with \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\`.`,
		`The C2 seal records \`secrets_override: ${receipt.secrets_override}\` after ${redaction.finding_count} local scanner findings with kinds \`${redaction.finding_kinds.join(",") || "none"}\`; this is not evidence of secret absence.`,
		`The separately claimed C2 structured staged credential scan reported \`${redaction.structured_staged_scan_findings}\` findings; its provenance is sealed supporting event \`12\`, not the Git note alone.`,
		`Local ignored C2 ledger archive \`${ledger.path}\` contains \`${ledger.sealed_event_count}\` sealed events and \`${ledger.archive_session_event_count}\` archived session events; post-seal events, if any, are outside the sealed manifest.`,
		`C2 ledger manifests use \`${ledger.manifest_format}\`; all-files SHA-256 is \`${ledger.all_files_manifest_sha256}\` and objects-only SHA-256 is \`${ledger.objects_manifest_sha256}\`.`,
		`C2 archive core SHA-256 values are session \`${ledger.session_log_sha256}\`, claims \`${ledger.claims_jsonl_sha256}\`, seals \`${ledger.seals_jsonl_sha256}\`, and .gitignore \`${ledger.gitignore_sha256}\`.`,
		`The local C2 HTML snapshot is \`${html.path}\`, SHA-256 \`${html.sha256}\`, ${html.bytes} bytes; it is not a portable strict witness.`,
		`The C2 Git note blob is \`${note.blob_oid}\` with body SHA-256 \`${note.body_sha256}\`.`,
	]);
}

function validateC3PReceiptEvidence(receipt) {
	const errors = [];
	const evidence = receipt.evidence;
	if (!evidence || typeof evidence !== "object" || Array.isArray(evidence) ||
		!isDeepStrictEqual(Object.keys(evidence).sort(), ["git_note", "html_report", "ledger_archive", "seal_redaction"])) {
		return ["local evidence field roster"];
	}
	const html = evidence.html_report;
	const sourcePrefix = typeof receipt.source_commit === "string" ? receipt.source_commit.slice(0, 12) : "";
	if (!html || typeof html !== "object" || Array.isArray(html) ||
		!isDeepStrictEqual(Object.keys(html).sort(), ["authority", "bytes", "path", "sha256"]) ||
		html.path !== `.countershape/evidence/p07b-c-c3p-final-${sourcePrefix}.html` ||
		html.authority !== "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS" ||
		!Number.isSafeInteger(html.bytes) || html.bytes <= 0 || html.bytes > 16 * 1024 * 1024 ||
		!(/^[0-9a-f]{64}$/u.test(html.sha256 ?? ""))) {
		errors.push("local HTML declaration");
	}
	const ledger = evidence.ledger_archive;
	const ledgerKeys = [
		"all_files_manifest_sha256", "archive_session_event_count", "authority", "claim_count",
		"claims_jsonl_sha256", "gitignore_sha256", "manifest_format", "object_file_count", "objects_manifest_sha256",
		"path", "seal_count", "sealed_event_count", "seals_jsonl_sha256", "session_log_sha256", "total_bytes", "total_file_count",
	];
	if (!ledger || typeof ledger !== "object" || Array.isArray(ledger) ||
		!isDeepStrictEqual(Object.keys(ledger).sort(), ledgerKeys) ||
		ledger.path !== ".didrun-history/2026-07-19-p07b-c-c3p-final/.didrun/" ||
		ledger.authority !== "LOCAL_IGNORED_ARCHIVE_NOT_GIT_AUTHORITY" ||
		ledger.manifest_format !== "sorted-relative-posix-path-tab-size-tab-sha256-newline/v1" ||
		ledger.sealed_event_count !== c3pClaimLabels.length || ledger.claim_count !== c3pClaimLabels.length ||
		ledger.seal_count !== 1 || !Number.isSafeInteger(ledger.archive_session_event_count) ||
		ledger.archive_session_event_count < c3pClaimLabels.length || ledger.archive_session_event_count > c3pClaimLabels.length + 4 ||
		!Number.isSafeInteger(ledger.object_file_count) || ledger.object_file_count <= 0 || ledger.object_file_count > 10_000 ||
		ledger.total_file_count !== ledger.object_file_count + 4 || !Number.isSafeInteger(ledger.total_bytes) ||
		ledger.total_bytes <= 0 || ledger.total_bytes > 1024 * 1024 * 1024 ||
		![ledger.all_files_manifest_sha256, ledger.objects_manifest_sha256, ledger.gitignore_sha256,
			ledger.session_log_sha256, ledger.claims_jsonl_sha256, ledger.seals_jsonl_sha256]
			.every((digest) => /^[0-9a-f]{64}$/u.test(digest ?? ""))) {
		errors.push("local ledger declaration");
	}
	const note = evidence.git_note;
	if (!note || typeof note !== "object" || Array.isArray(note) ||
		!isDeepStrictEqual(Object.keys(note).sort(), ["blob_oid", "body_sha256", "version"]) || note.version !== 1 ||
		!(/^[0-9a-f]{40}$/u.test(note.blob_oid ?? "")) || !(/^[0-9a-f]{64}$/u.test(note.body_sha256 ?? ""))) {
		errors.push("Git note evidence declaration");
	}
	const redaction = evidence.seal_redaction;
	if (!redaction || typeof redaction !== "object" || Array.isArray(redaction) ||
		!isDeepStrictEqual(Object.keys(redaction).sort(), [
			"finding_count", "finding_kinds", "finding_provenance", "structured_scan_provenance", "structured_staged_scan_findings",
		]) || !Number.isSafeInteger(redaction.finding_count) || redaction.finding_count < 0 || redaction.finding_count > 100_000 ||
		!Array.isArray(redaction.finding_kinds) || !isDeepStrictEqual(redaction.finding_kinds, redaction.finding_count === 0 ? [] : ["high-entropy"]) ||
		redaction.finding_provenance !== "LOCAL_TERMINAL_OBSERVATION_NOT_GIT_NOTE" ||
		redaction.structured_staged_scan_findings !== 0 ||
		redaction.structured_scan_provenance !== "SEALED_C3P_SUPPORTING_EVENT_10" ||
		(receipt.secrets_override !== (redaction.finding_count > 0))) {
		errors.push("seal redaction evidence declaration");
	}
	return errors;
}

function validateC3PReceiptDeclaration(receipt) {
	const errors = [];
	const keys = [
		"claims", "coverage_complete_events", "coverage_total_events", "evidence", "schema_version",
		"secrets_override", "source_commit", "source_subject", "source_tree", "strict_claims_recorded_exact",
		"strict_claims_total", "strict_exit_code",
	];
	if (!receipt || typeof receipt !== "object" || Array.isArray(receipt) ||
		!isDeepStrictEqual(Object.keys(receipt).sort(), keys)) return ["root field roster"];
	if (receipt.schema_version !== "countershape/p07b-c-c3p-receipt/v1") errors.push("schema version");
	if (!/^[0-9a-f]{40}$/u.test(receipt.source_commit ?? "") || /^0+$/u.test(receipt.source_commit ?? "")) errors.push("source commit");
	if (!/^[0-9a-f]{40}$/u.test(receipt.source_tree ?? "") || /^0+$/u.test(receipt.source_tree ?? "")) errors.push("source tree");
	if (receipt.source_subject !== "fix: use stable Darwin boot-session identity") errors.push("source subject");
	if (typeof receipt.secrets_override !== "boolean") errors.push("secrets override disclosure");
	if (receipt.strict_exit_code !== 0) errors.push("strict exit code");
	if (receipt.strict_claims_recorded_exact !== c3pClaimLabels.length || receipt.strict_claims_total !== c3pClaimLabels.length) {
		errors.push("strict claim counts");
	}
	if (receipt.coverage_total_events !== c3pClaimLabels.length || receipt.coverage_complete_events !== c3pClaimLabels.length) {
		errors.push("coverage counts");
	}
	if (!Array.isArray(receipt.claims) || receipt.claims.length !== c3pClaimLabels.length) {
		errors.push("claim roster length");
	} else {
		for (let index = 0; index < c3pClaimLabels.length; index += 1) {
			const claim = receipt.claims[index];
			if (!claim || typeof claim !== "object" || Array.isArray(claim) ||
				!isDeepStrictEqual(Object.keys(claim).sort(), ["grade", "label", "supporting_event_index", "type"]) ||
				claim.supporting_event_index !== index || claim.type !== c3pClaimTypes[index] ||
				claim.label !== c3pClaimLabels[index] || claim.grade !== "TREE-EXACT") {
				errors.push(`claim ${index + 1}`);
			}
		}
	}
	errors.push(...validateC3PReceiptEvidence(receipt));
	return errors;
}

function validateC3PRawClaimArgv(index, argv) {
	return Number.isSafeInteger(index) && index >= 0 && index < c3pExpectedClaimArgv.length &&
		Array.isArray(argv) && argv.length > 0 && argv.length <= 64 &&
		argv.every((argument) => typeof argument === "string" && argument.length > 0 && argument.length <= 4096 &&
			!/[\u0000-\u001f\u007f]/u.test(argument)) &&
		isDeepStrictEqual(argv, c3pExpectedClaimArgv[index]);
}

function validateC3PClaimPreview(index, argv) {
	if (![3, 4, 5].includes(index)) return validateC3PRawClaimArgv(index, argv);
	const expected = c3pExpectedClaimArgv[index];
	if (!Array.isArray(argv) || argv.length !== expected.length || argv.some((entry) => typeof entry !== "string")) return false;
	for (let position = 0; position < expected.length; position += 1) {
		if (argv[position] === expected[position]) continue;
		if (position >= 2 && position <= 7) {
			const suffixOffset = expected[position].indexOf(".countershape/");
			const suffix = suffixOffset < 0 ? "" : expected[position].slice(suffixOffset);
			if (suffix !== "" && (argv[position] === `«redacted:high-entropy»${suffix}` ||
				argv[position] === "«redacted:high-entropy».«redacted:high-entropy»")) continue;
		}
		if (position >= 22 && position <= 24 && argv[position] === "«redacted:high-entropy»") continue;
		return false;
	}
	return true;
}

function validateC3PSourceDiff(authority) {
	const errors = [];
	if (!isDeepStrictEqual(authority.parents, [sealedC2BIdentity.commit])) errors.push("source parents");
	const entries = authority.source_diff;
	if (!Array.isArray(entries)) return [...errors, "source diff roster"];
	const paths = entries.map((entry) => entry?.path);
	if (new Set(paths).size !== paths.length || !isDeepStrictEqual(paths, requiredC3PPaths)) {
		errors.push("source diff exact path roster");
	}
	for (const path of requiredC3PPaths) {
		const entry = entries.find((candidate) => candidate?.path === path);
		if (!entry || !isDeepStrictEqual(Object.keys(entry).sort(), ["new_mode", "new_oid", "old_mode", "old_oid", "path", "status"])) {
			errors.push(`source diff row ${path}`);
			continue;
		}
		const objectPattern = /^(?:[0-9a-f]{40}|[0-9a-f]{64})$/u;
		const newObjectValid = objectPattern.test(entry.new_oid) && !/^0+$/u.test(entry.new_oid);
		if (requiredC3PAddedPaths.has(path)) {
			if (entry.status !== "A" || entry.old_mode !== "000000" || entry.new_mode !== "100644" ||
				!objectPattern.test(entry.old_oid) || !/^0+$/u.test(entry.old_oid) || !newObjectValid ||
				entry.old_oid.length !== entry.new_oid.length) {
				errors.push(`source diff added row ${path}`);
			}
		} else if (entry.status !== "M" || entry.old_mode !== "100644" || entry.new_mode !== "100644" ||
			!objectPattern.test(entry.old_oid) || /^0+$/u.test(entry.old_oid) || !newObjectValid ||
			entry.old_oid.length !== entry.new_oid.length || entry.old_oid === entry.new_oid) {
			errors.push(`source diff modified row ${path}`);
		}
	}
	return errors;
}

function validateC3PReceiptAuthority(receipt, authority) {
	const errors = [];
	if (!authority || typeof authority !== "object" || Array.isArray(authority)) return ["missing Git/didrun authority"];
	if (authority.commit !== receipt.source_commit) errors.push("source commit does not equal admitted Git commit");
	if (authority.tree !== receipt.source_tree) errors.push("source tree does not equal Git commit tree");
	if (authority.parent !== sealedC2BIdentity.commit) errors.push("source parent");
	errors.push(...validateC3PSourceDiff(authority));
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
	if (!Array.isArray(note.claims) || note.claims.length !== c3pClaimLabels.length) {
		errors.push("didrun note claim roster length");
	} else {
		for (let index = 0; index < c3pClaimLabels.length; index += 1) {
			const recorded = note.claims[index];
			const claim = recorded?.claim;
			if (claim?.label !== c3pClaimLabels[index] || claim?.ctype !== c3pClaimTypes[index] ||
				claim?.declared_at_index !== index || !isDeepStrictEqual(claim?.event_indices, [index]) ||
				!isDeepStrictEqual(claim?.pathspecs, []) ||
				recorded?.supporting_event_index !== index || recorded?.grade !== "tree-exact" || recorded?.exit_code !== 0 ||
				recorded?.reason !== "self-stable command ran against the sealed tree" || !isDeepStrictEqual(recorded?.delta, [])) {
				errors.push(`didrun note claim ${index + 1}`);
			}
			if (!validateC3PClaimPreview(index, claim?.argv_preview)) errors.push(`didrun note claim ${index + 1} argv`);
		}
	}
	if (note.coverage?.total_events !== receipt.coverage_total_events ||
		note.coverage?.by_coverage?.complete !== receipt.coverage_complete_events ||
		!isDeepStrictEqual(Object.keys(note.coverage?.by_coverage ?? {}), ["complete"])) {
		errors.push("didrun note exact event coverage");
	}
	return errors;
}

function c3pReceiptDisclosureLines(receipt) {
	const { html_report: html, ledger_archive: ledger, git_note: note, seal_redaction: redaction } = receipt.evidence;
	return Object.freeze([
		`C3P source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`, is sealed, note-present, and strict-clean with \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\`.`,
		`The C3P seal records \`secrets_override: ${receipt.secrets_override}\` after ${redaction.finding_count} local scanner findings with kinds \`${redaction.finding_kinds.join(",") || "none"}\`; this is not evidence of secret absence.`,
		`The separately claimed C3P structured staged credential scan reported \`${redaction.structured_staged_scan_findings}\` findings; its provenance is sealed supporting event \`10\`, not the Git note alone.`,
		`Local ignored C3P ledger archive \`${ledger.path}\` contains \`${ledger.sealed_event_count}\` sealed events and \`${ledger.archive_session_event_count}\` archived session events; post-seal events, if any, are outside the sealed manifest.`,
		`C3P ledger manifests use \`${ledger.manifest_format}\`; all-files SHA-256 is \`${ledger.all_files_manifest_sha256}\` and objects-only SHA-256 is \`${ledger.objects_manifest_sha256}\`.`,
		`C3P archive core SHA-256 values are session \`${ledger.session_log_sha256}\`, claims \`${ledger.claims_jsonl_sha256}\`, seals \`${ledger.seals_jsonl_sha256}\`, and .gitignore \`${ledger.gitignore_sha256}\`.`,
		`The local C3P HTML snapshot is \`${html.path}\`, SHA-256 \`${html.sha256}\`, ${html.bytes} bytes; it is not a portable strict witness.`,
		`The C3P Git note blob is \`${note.blob_oid}\` with body SHA-256 \`${note.body_sha256}\`.`,
	]);
}

async function checkC3PReceiptPhase(root, overrides, status, handoff, errors, injectedReceiptAuthority) {
	let receiptText;
	try {
		receiptText = await readText(root, c3pReceiptDeclarationPath, overrides);
	} catch (error) {
		if (error.code !== "ENOENT") errors.push(`${c3pReceiptDeclarationPath}: unreadable (${error.message})`);
	}

	if (receiptText === undefined) {
		requireExactlyOnce(status, "- **State:** active pre-seal `SOURCE_FULL` prerequisite; every C3P grade below is `UNRECEIPTED`", c3pStatusPath, "C3P pre-seal source state", errors);
		requireExactlyOnce(status, `- **Scope:** 25 exact mode-\`100644\` paths, no prefixes; sorted-newline roster \`${requiredC3PDigest}\``, c3pStatusPath, "C3P source scope", errors);
		requireExactlyOnce(status, "- **Commit subject:** `fix: use stable Darwin boot-session identity`", c3pStatusPath, "C3P source subject", errors);
		for (let index = 0; index < c3pClaimLabels.length; index += 1) {
			requireExactlyOnce(status, `| \`${c3pClaimLabels[index]}\` | \`${c3pClaimTypes[index]}\` | \`UNRECEIPTED\` |`, c3pStatusPath, "C3P pending source receipt map", errors);
			requireClaimLabelExactlyOnce(status, c3pClaimLabels[index], c3pStatusPath, "C3P pending source receipt map", errors);
		}
		if (handoff.includes("<!-- P07B-C-C3P-SOURCE-RECEIPTS:START -->") ||
			handoff.includes("<!-- P07B-C-C3P-SOURCE-RECEIPTS:END -->")) {
			errors.push("docs/HANDOFF_MODE_C.md: C3P pending receipt state cannot contain a source receipt block");
		}
		for (const digest of [requiredC3PDigest, requiredC3PBDigest, requiredC3Digest, requiredC3BDigest]) {
			if (!handoff.includes(digest)) errors.push(`docs/HANDOFF_MODE_C.md: C3P prerequisite digest missing: ${digest}`);
		}
		rejectC3PBSelfReceiptClaims(status, c3pStatusPath, errors);
		rejectC3PBSelfReceiptClaims(handoff, "docs/HANDOFF_MODE_C.md", errors);
		return;
	}

	let receipt;
	try {
		receipt = JSON.parse(receiptText);
	} catch (error) {
		errors.push(`${c3pReceiptDeclarationPath}: invalid JSON (${error.message})`);
		return;
	}
	const receiptErrors = validateC3PReceiptDeclaration(receipt);
	for (const error of receiptErrors) errors.push(`${c3pReceiptDeclarationPath}: invalid receipt declaration (${error})`);
	if (receiptErrors.length > 0) return;
	let authority = injectedReceiptAuthority;
	if (authority === undefined) {
		try {
			authority = await loadReceiptAuthorityFromGit(root, receipt);
		} catch (error) {
			errors.push(`${c3pReceiptDeclarationPath}: Git/didrun authority unavailable (${error.message})`);
			return;
		}
	}
	const authorityErrors = validateC3PReceiptAuthority(receipt, authority);
	for (const error of authorityErrors) errors.push(`${c3pReceiptDeclarationPath}: Git/didrun authority mismatch (${error})`);
	if (authorityErrors.length > 0) return;

	const state = `- **Source receipt:** C3P source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`, is sealed, note-present, and strict-clean; every C3P source grade below is \`TREE-EXACT\`. This C3PB receipt-document working unit binds only that existing source and remains \`UNRECEIPTED\` until its own commit, seal, note, and strict boundary.`;
	requireExactlyOnce(status, state, c3pStatusPath, "C3PB reconciled source state", errors);
	requireExactlyOnce(status, "## C3P source receipt map", c3pStatusPath, "C3PB receipt heading", errors);
	requireExactlyOnce(status, "- C3P source strict exit: `0`", c3pStatusPath, "C3PB strict result", errors);
	requireExactlyOnce(status, `- C3P source strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\``, c3pStatusPath, "C3PB strict claim count", errors);
	requireExactlyOnce(status, "This receipt binds only the already-existing C3P source commit. C3PB cannot name or grade its own commit, tree, Git note, or strict result.", c3pStatusPath, "C3PB no-recursion boundary", errors);
	for (const claim of receipt.claims) {
		requireExactlyOnce(status, `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`, c3pStatusPath, "C3PB source receipt map", errors);
		requireClaimLabelExactlyOnce(status, claim.label, c3pStatusPath, "C3PB source receipt map", errors);
	}
	const disclosures = c3pReceiptDisclosureLines(receipt);
	for (const disclosure of disclosures) requireExactlyOnce(status, disclosure, c3pStatusPath, "C3PB source evidence disclosure", errors);
	requireExactlyOnce(handoff, "<!-- P07B-C-C3P-SOURCE-RECEIPTS:START -->", "docs/HANDOFF_MODE_C.md", "C3PB receipt block", errors);
	requireExactlyOnce(handoff, "<!-- P07B-C-C3P-SOURCE-RECEIPTS:END -->", "docs/HANDOFF_MODE_C.md", "C3PB receipt block", errors);
	requireExactlyOnce(handoff, `C3P source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`.`, "docs/HANDOFF_MODE_C.md", "C3PB source identity", errors);
	requireExactlyOnce(handoff, `C3P strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\`; strict exit: \`0\`.`, "docs/HANDOFF_MODE_C.md", "C3PB strict result", errors);
	for (const claim of receipt.claims) {
		requireExactlyOnce(handoff, `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`, "docs/HANDOFF_MODE_C.md", "C3PB claim map", errors);
		requireClaimLabelExactlyOnce(handoff, claim.label, "docs/HANDOFF_MODE_C.md", "C3PB claim map", errors);
	}
	for (const disclosure of disclosures) requireExactlyOnce(handoff, disclosure, "docs/HANDOFF_MODE_C.md", "C3PB source evidence disclosure", errors);
	rejectC3PBSelfReceiptClaims(status, c3pStatusPath, errors);
	rejectC3PBSelfReceiptClaims(handoff, "docs/HANDOFF_MODE_C.md", errors);
}

function syntheticC3PReceiptFixture() {
	const sourceCommit = "6".repeat(40);
	const sourceTree = "7".repeat(40);
	const sourceDiff = requiredC3PPaths.map((path, index) => {
		const added = requiredC3PAddedPaths.has(path);
		return {
			old_mode: added ? "000000" : "100644",
			new_mode: "100644",
			old_oid: added ? "0".repeat(40) : (index + 1).toString(16).padStart(40, "8").slice(-40),
			new_oid: (index + 1).toString(16).padStart(40, "9").slice(-40),
			status: added ? "A" : "M",
			path,
		};
	});
	const receipt = {
		schema_version: "countershape/p07b-c-c3p-receipt/v1",
		source_commit: sourceCommit,
		source_tree: sourceTree,
		source_subject: "fix: use stable Darwin boot-session identity",
		strict_exit_code: 0,
		strict_claims_recorded_exact: c3pClaimLabels.length,
		strict_claims_total: c3pClaimLabels.length,
		coverage_complete_events: c3pClaimLabels.length,
		coverage_total_events: c3pClaimLabels.length,
		secrets_override: false,
		claims: c3pClaimLabels.map((label, index) => ({
			label, type: c3pClaimTypes[index], supporting_event_index: index, grade: "TREE-EXACT",
		})),
		evidence: {
			html_report: {
				path: `.countershape/evidence/p07b-c-c3p-final-${sourceCommit.slice(0, 12)}.html`,
				sha256: "a".repeat(64), bytes: 8192,
				authority: "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS",
			},
			ledger_archive: {
				path: ".didrun-history/2026-07-19-p07b-c-c3p-final/.didrun/",
				authority: "LOCAL_IGNORED_ARCHIVE_NOT_GIT_AUTHORITY",
				manifest_format: "sorted-relative-posix-path-tab-size-tab-sha256-newline/v1",
				sealed_event_count: c3pClaimLabels.length,
				archive_session_event_count: c3pClaimLabels.length + 1,
				claim_count: c3pClaimLabels.length,
				seal_count: 1,
				object_file_count: 20,
				total_file_count: 24,
				total_bytes: 100_000,
				all_files_manifest_sha256: "b".repeat(64),
				objects_manifest_sha256: "c".repeat(64),
				gitignore_sha256: "d".repeat(64),
				session_log_sha256: "e".repeat(64),
				claims_jsonl_sha256: "f".repeat(64),
				seals_jsonl_sha256: "1".repeat(64),
			},
			git_note: { version: 1, blob_oid: "2".repeat(40), body_sha256: "3".repeat(64) },
			seal_redaction: {
				finding_count: 0,
				finding_kinds: [],
				finding_provenance: "LOCAL_TERMINAL_OBSERVATION_NOT_GIT_NOTE",
				structured_staged_scan_findings: 0,
				structured_scan_provenance: "SEALED_C3P_SUPPORTING_EVENT_10",
			},
		},
	};
	const authority = {
		commit: sourceCommit,
		tree: sourceTree,
		parent: sealedC2BIdentity.commit,
		parents: [sealedC2BIdentity.commit],
		source_diff: sourceDiff,
		subject: receipt.source_subject,
		ancestor_of_head: true,
		note_blob_oid: receipt.evidence.git_note.blob_oid,
		note_body_sha256: receipt.evidence.git_note.body_sha256,
		note: {
			version: 1,
			commit: sourceCommit,
			tree: sourceTree,
			secrets_override: false,
			claims: c3pClaimLabels.map((label, index) => ({
				claim: {
					label, ctype: c3pClaimTypes[index], declared_at_index: index, event_indices: [index],
					pathspecs: [], argv_preview: [...c3pExpectedClaimArgv[index]],
				},
				supporting_event_index: index,
				grade: "tree-exact",
				exit_code: 0,
				reason: "self-stable command ran against the sealed tree",
				delta: [],
			})),
			coverage: { total_events: c3pClaimLabels.length, by_coverage: { complete: c3pClaimLabels.length } },
		},
	};
	return { receipt, authority };
}

async function syntheticC3PReceiptPlanFixture(fixture) {
	const absent = await syntheticC3PAbsentPhaseFixture();
	let status = absent.get(c3pStatusPath);
	const sourceState = "- **State:** active pre-seal `SOURCE_FULL` prerequisite; every C3P grade below is `UNRECEIPTED`";
	const receiptState = `- **Source receipt:** C3P source commit \`${fixture.receipt.source_commit}\`, tree \`${fixture.receipt.source_tree}\`, is sealed, note-present, and strict-clean; every C3P source grade below is \`TREE-EXACT\`. This C3PB receipt-document working unit binds only that existing source and remains \`UNRECEIPTED\` until its own commit, seal, note, and strict boundary.`;
	status = replaceFixtureExactlyOnce(status, sourceState, receiptState, "C3P receipt-present state");
	const metadata = [
		"## C3P source receipt map",
		"- C3P source strict exit: `0`",
		`- C3P source strict claims: \`${fixture.receipt.strict_claims_recorded_exact}/${fixture.receipt.strict_claims_total} claims recorded-exact\``,
		"This receipt binds only the already-existing C3P source commit. C3PB cannot name or grade its own commit, tree, Git note, or strict result.",
		...c3pReceiptDisclosureLines(fixture.receipt),
	].join("\n");
	status = replaceFixtureExactlyOnce(status, "## Planned source claim map", metadata, "C3P receipt-present heading");
	for (const claim of fixture.receipt.claims) {
		status = replaceFixtureExactlyOnce(
			status,
			`| \`${claim.label}\` | \`${claim.type}\` | \`UNRECEIPTED\` |`,
			`| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`,
			`C3P receipt-present row ${claim.label}`,
		);
	}
	const currentHandoff = absent.get("docs/HANDOFF_MODE_C.md");
	const rows = fixture.receipt.claims.map((claim) => `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`).join("\n");
	const block = [
		"<!-- P07B-C-C3P-SOURCE-RECEIPTS:START -->",
		`C3P source commit \`${fixture.receipt.source_commit}\`, tree \`${fixture.receipt.source_tree}\`.`,
		`C3P strict claims: \`${fixture.receipt.strict_claims_recorded_exact}/${fixture.receipt.strict_claims_total} claims recorded-exact\`; strict exit: \`0\`.`,
		rows,
		...c3pReceiptDisclosureLines(fixture.receipt),
		"<!-- P07B-C-C3P-SOURCE-RECEIPTS:END -->",
	].join("\n");
	return {
		overrides: new Map([
			[c3pStatusPath, status],
			["docs/HANDOFF_MODE_C.md", `${currentHandoff.trimEnd()}\n\n${block}\n`],
			[c3pReceiptDeclarationPath, `${JSON.stringify(fixture.receipt)}\n`],
		]),
		authority: fixture.authority,
	};
}

async function syntheticC3PAbsentPhaseFixture() {
	let status;
	let handoff;
	try {
		await requireReceiptDeclarationPhase(repositoryRoot, c3pReceiptDeclarationPath, false, "C3P");
		status = await readText(repositoryRoot, c3pStatusPath, new Map());
		handoff = await readText(repositoryRoot, "docs/HANDOFF_MODE_C.md", new Map());
	} catch (error) {
		if (!String(error.message).includes("must be absent")) throw error;
		const receiptText = await readReceiptDeclarationSnapshot(repositoryRoot, c3pReceiptDeclarationPath, "C3P");
		const receipt = JSON.parse(receiptText);
		const declarationErrors = validateC3PReceiptDeclaration(receipt);
		if (declarationErrors.length > 0) throw new Error(`C3P absent fixture receipt invalid: ${declarationErrors.join(", ")}`);
		const env = {
			HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1",
			GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
			GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
		};
		const readBlob = (path) => {
			const result = spawnSync("/usr/bin/git", ["--no-replace-objects", "cat-file", "blob", `${receipt.source_commit}:${path}`], {
				cwd: repositoryRoot, encoding: "utf8", timeout: 30_000, maxBuffer: 4 * 1024 * 1024, env,
			});
			if (result.error || result.signal || result.status !== 0 || result.stderr !== "" || !result.stdout.endsWith("\n")) {
				throw new Error(`C3P absent fixture cannot reopen ${path}`);
			}
			return result.stdout;
		};
		status = readBlob(c3pStatusPath);
		handoff = readBlob("docs/HANDOFF_MODE_C.md");
	}
	return new Map([
		[c3pStatusPath, status],
		["docs/HANDOFF_MODE_C.md", handoff],
		[c3pReceiptDeclarationPath, ABSENT_FIXTURE_PATH],
	]);
}

export async function runC3PReceiptSelfTest() {
	const liveErrors = await checkPlan();
	if (liveErrors.length > 0) throw new Error(`P07B-C C3P receipt checker live baseline failed:\n${liveErrors.join("\n")}`);
	let rejected = 0;
	const requireError = (name, errors, expected) => {
		if (!errors.some((error) => error.includes(expected))) {
			throw new Error(`P07B-C C3P receipt checker self-test false negative: ${name} (${errors.join("; ")})`);
		}
		rejected += 1;
	};
	const absentOverrides = await syntheticC3PAbsentPhaseFixture();
	const absentErrors = await checkPlan(repositoryRoot, absentOverrides);
	if (absentErrors.length > 0) throw new Error(`P07B-C C3P absent-phase baseline failed:\n${absentErrors.join("\n")}`);
	const liveStatus = absentOverrides.get(c3pStatusPath);
	const liveHandoff = absentOverrides.get("docs/HANDOFF_MODE_C.md");
	for (const testCase of [
		{
			name: "pre-receipt state drift", path: c3pStatusPath,
			value: liveStatus.replace("- **State:** active pre-seal `SOURCE_FULL` prerequisite; every C3P grade below is `UNRECEIPTED`", "- **State:** prematurely sealed"),
			expect: "C3P pre-seal source state",
		},
		{
			name: "pre-receipt pending grade drift", path: c3pStatusPath,
			value: liveStatus.replace(
				`| \`${c3pClaimLabels[0]}\` | \`${c3pClaimTypes[0]}\` | \`UNRECEIPTED\` |`,
				`| \`${c3pClaimLabels[0]}\` | \`${c3pClaimTypes[0]}\` | \`TREE-EXACT\` |`,
			),
			expect: "C3P pending source receipt map",
		},
		{
			name: "pre-receipt marker injection", path: "docs/HANDOFF_MODE_C.md",
			value: `${liveHandoff}\n<!-- P07B-C-C3P-SOURCE-RECEIPTS:START -->\n`,
			expect: "pending receipt state cannot contain",
		},
	]) {
		const overrides = new Map(absentOverrides);
		overrides.set(testCase.path, testCase.value);
		const errors = await checkPlan(repositoryRoot, overrides);
		requireError(testCase.name, errors, testCase.expect);
	}

	const fixture = syntheticC3PReceiptFixture();
	const declarationBaseline = validateC3PReceiptDeclaration(fixture.receipt);
	if (declarationBaseline.length > 0) throw new Error(`P07B-C C3P receipt declaration baseline failed: ${declarationBaseline.join(", ")}`);
	const authorityBaseline = validateC3PReceiptAuthority(fixture.receipt, fixture.authority);
	if (authorityBaseline.length > 0) throw new Error(`P07B-C C3P receipt authority baseline failed: ${authorityBaseline.join(", ")}`);
	for (const [name, mutate, expected] of [
		["schema drift", (candidate) => { candidate.schema_version = "countershape/p07b-c-c3p-receipt/v2"; }, "schema version"],
		["source subject drift", (candidate) => { candidate.source_subject = "fix: reinterpret clock"; }, "source subject"],
		["claim grade drift", (candidate) => { candidate.claims[0].grade = "UNRECEIPTED"; }, "claim 1"],
		["HTML path drift", (candidate) => { candidate.evidence.html_report.path = "evidence/other.html"; }, "local HTML declaration"],
		["scan provenance drift", (candidate) => { candidate.evidence.seal_redaction.structured_scan_provenance = "UNBOUND"; }, "seal redaction"],
	]) {
		const candidate = structuredClone(fixture.receipt);
		mutate(candidate);
		requireError(name, validateC3PReceiptDeclaration(candidate), expected);
	}
	for (const [name, mutate, expected] of [
		["parent drift", (candidate) => { candidate.parents = ["4".repeat(40)]; candidate.parent = candidate.parents[0]; }, "source parent"],
		["source roster drift", (candidate) => { candidate.source_diff.pop(); }, "source diff exact path roster"],
		["note argv drift", (candidate) => { candidate.note.claims[0].claim.argv_preview = ["node", "wrong.mjs"]; }, "claim 1 argv"],
		["note grade drift", (candidate) => { candidate.note.claims[0].grade = "scope-exact"; }, "claim 1"],
		["coverage drift", (candidate) => { candidate.note.coverage.by_coverage.complete -= 1; }, "event coverage"],
	]) {
		const candidate = structuredClone(fixture.authority);
		mutate(candidate);
		requireError(name, validateC3PReceiptAuthority(fixture.receipt, candidate), expected);
	}
	const phase = await syntheticC3PReceiptPlanFixture(fixture);
	const phaseErrors = await checkPlan(repositoryRoot, phase.overrides, undefined, undefined, undefined, phase.authority);
	if (phaseErrors.length > 0) throw new Error(`P07B-C C3P receipt-present phase baseline failed:\n${phaseErrors.join("\n")}`);
	for (const [name, path, mutate, expected] of [
		["receipt status grade", c3pStatusPath, (body) => body.replace(
			`| \`${c3pClaimLabels[0]}\` | \`${c3pClaimTypes[0]}\` | \`TREE-EXACT\` |`,
			`| \`${c3pClaimLabels[0]}\` | \`${c3pClaimTypes[0]}\` | \`UNRECEIPTED\` |`,
		), "source receipt map"],
		["receipt no-recursion", c3pStatusPath, (body) => body.replace("This receipt binds only the already-existing C3P source commit.", "Receipt recursion omitted."), "no-recursion"],
		["receipt handoff identity", "docs/HANDOFF_MODE_C.md", (body) => body.replace(fixture.receipt.source_tree, "5".repeat(40)), "source identity"],
		["C3PB self receipt", "docs/HANDOFF_MODE_C.md", (body) => `${body}\nC3PB is sealed and strict-clean.\n`, "C3PB self-receipt"],
	]) {
		const overrides = new Map(phase.overrides);
		overrides.set(path, mutate(overrides.get(path)));
		const errors = await checkPlan(repositoryRoot, overrides, undefined, undefined, undefined, phase.authority);
		requireError(name, errors, expected);
	}

	const root = await mkdtemp(resolve(tmpdir(), "countershape-c3p-local-evidence-"));
	try {
		const local = await buildLocalEvidenceFixture(root, {
			claimLabels: c3pClaimLabels,
			claimTypes: c3pClaimTypes,
			claimArgv: c3pExpectedClaimArgv,
			sourceCommit: fixture.receipt.source_commit,
			sourceTree: fixture.receipt.source_tree,
			archiveSessionEventCount: c3pClaimLabels.length,
		});
		const localErrors = await verifyLocalEvidence(root, local.receipt, c3pClaimLabels, c3pClaimTypes, c3pExpectedClaimArgv);
		if (localErrors.length > 0) throw new Error(`P07B-C C3P local-evidence baseline failed: ${localErrors.join(", ")}`);
		const mutated = structuredClone(local.receipt);
		mutated.evidence.html_report.sha256 = "0".repeat(64);
		requireError("local HTML mutation", await verifyLocalEvidence(root, mutated, c3pClaimLabels, c3pClaimTypes, c3pExpectedClaimArgv), "local HTML digest");
	} finally {
		await rm(root, { recursive: true, force: true });
	}
	console.log(`P07B-C C3P receipt checker self-test passed: ${rejected} absent-phase, declaration, Git/note, receipt-phase, self-receipt, and local-snapshot mutations rejected`);
	return rejected;
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

async function checkC2ReceiptPhase(root, overrides, status, handoff, errors, injectedReceiptAuthority) {
	let receiptText;
	try {
		receiptText = await readText(root, c2ReceiptDeclarationPath, overrides);
	} catch (error) {
		if (error.code !== "ENOENT") errors.push(`${c2ReceiptDeclarationPath}: unreadable (${error.message})`);
	}

	if (receiptText === undefined) {
		requireExactlyOnce(
			status,
			"- **State:** active pre-seal source boundary; every C2 grade below is `UNRECEIPTED`",
			c2StatusPath,
			"C2 pre-seal source state",
			errors,
		);
		requireExactlyOnce(status, `- **Scope:** 26 exact paths, no prefixes; sorted-newline roster \`${requiredC2Digest}\``, c2StatusPath, "C2 source scope", errors);
		for (let index = 0; index < c2ClaimLabels.length; index += 1) {
			requireExactlyOnce(
				status,
				`| \`${c2ClaimLabels[index]}\` | \`${c2ClaimTypes[index]}\` | \`UNRECEIPTED\` |`,
				c2StatusPath,
				"C2 pending source receipt map",
				errors,
			);
			requireClaimLabelExactlyOnce(status, c2ClaimLabels[index], c2StatusPath, "C2 pending source receipt map", errors);
		}
		if (handoff.includes("<!-- P07B-C-C2-SOURCE-RECEIPTS:START -->") ||
			handoff.includes("<!-- P07B-C-C2-SOURCE-RECEIPTS:END -->")) {
			errors.push("docs/HANDOFF_MODE_C.md: C2 pending receipt state cannot contain a source receipt block");
		}
		requireExactlyOnce(handoff, requiredC2Digest, "docs/HANDOFF_MODE_C.md", "C2 source scope digest", errors);
		requireExactlyOnce(handoff, requiredC2BDigest, "docs/HANDOFF_MODE_C.md", "C2B receipt scope digest", errors);
		rejectC2BSelfReceiptClaims(status, c2StatusPath, errors);
		rejectC2BSelfReceiptClaims(handoff, "docs/HANDOFF_MODE_C.md", errors);
		return;
	}

	let receipt;
	try {
		receipt = JSON.parse(receiptText);
	} catch (error) {
		errors.push(`${c2ReceiptDeclarationPath}: invalid JSON (${error.message})`);
		return;
	}
	const receiptErrors = validateC2ReceiptDeclaration(receipt);
	for (const error of receiptErrors) errors.push(`${c2ReceiptDeclarationPath}: invalid receipt declaration (${error})`);
	if (receiptErrors.length > 0) return;
	let authority = injectedReceiptAuthority;
	if (authority === undefined) {
		try {
			authority = await loadReceiptAuthorityFromGit(root, receipt);
		} catch (error) {
			errors.push(`${c2ReceiptDeclarationPath}: Git/didrun authority unavailable (${error.message})`);
			return;
		}
	}
	const authorityErrors = validateC2ReceiptAuthority(receipt, authority);
	for (const error of authorityErrors) errors.push(`${c2ReceiptDeclarationPath}: Git/didrun authority mismatch (${error})`);
	if (authorityErrors.length > 0) return;

	const state = `- **Source receipt:** C2 source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`, is sealed, note-present, and strict-clean; every C2 source grade below is \`TREE-EXACT\`. This C2B receipt-document working unit binds only that existing source and remains \`UNRECEIPTED\` until its own commit, seal, note, and strict boundary.`;
	requireExactlyOnce(status, state, c2StatusPath, "C2B reconciled source state", errors);
	requireExactlyOnce(status, "## C2 source receipt map", c2StatusPath, "C2B receipt heading", errors);
	requireExactlyOnce(status, "- C2 source strict exit: `0`", c2StatusPath, "C2B strict result", errors);
	requireExactlyOnce(status, `- C2 source strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\``, c2StatusPath, "C2B strict claim count", errors);
	requireExactlyOnce(status, "This receipt binds only the already-existing C2 source commit. C2B cannot name or grade its own commit, tree, Git note, or strict result.", c2StatusPath, "C2B no-recursion boundary", errors);
	for (const claim of receipt.claims) {
		requireExactlyOnce(status, `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`, c2StatusPath, "C2B source receipt map", errors);
		requireClaimLabelExactlyOnce(status, claim.label, c2StatusPath, "C2B source receipt map", errors);
	}
	const disclosures = c2ReceiptDisclosureLines(receipt);
	for (const disclosure of disclosures) {
		requireExactlyOnce(status, disclosure, c2StatusPath, "C2B source evidence disclosure", errors);
	}
	requireExactlyOnce(handoff, "<!-- P07B-C-C2-SOURCE-RECEIPTS:START -->", "docs/HANDOFF_MODE_C.md", "C2B receipt block", errors);
	requireExactlyOnce(handoff, "<!-- P07B-C-C2-SOURCE-RECEIPTS:END -->", "docs/HANDOFF_MODE_C.md", "C2B receipt block", errors);
	requireExactlyOnce(handoff, `C2 source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`.`, "docs/HANDOFF_MODE_C.md", "C2B source identity", errors);
	requireExactlyOnce(handoff, `C2 strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\`; strict exit: \`0\`.`, "docs/HANDOFF_MODE_C.md", "C2B strict result", errors);
	for (const claim of receipt.claims) {
		requireExactlyOnce(handoff, `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`, "docs/HANDOFF_MODE_C.md", "C2B claim map", errors);
		requireClaimLabelExactlyOnce(handoff, claim.label, "docs/HANDOFF_MODE_C.md", "C2B claim map", errors);
	}
	for (const disclosure of disclosures) {
		requireExactlyOnce(handoff, disclosure, "docs/HANDOFF_MODE_C.md", "C2B source evidence disclosure", errors);
	}
	rejectC2BSelfReceiptClaims(status, c2StatusPath, errors);
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

export async function checkPlan(root = repositoryRoot, overrides = new Map(), receiptAuthority, c1ReceiptAuthority, c2ReceiptAuthority, c3pReceiptAuthority) {
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
	for (const [path, snippets] of Object.entries(requiredC2MaintenanceText)) {
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
				errors.push(`${path}: missing required C2 receipt-phase maintenance ruling: ${JSON.stringify(snippet)}`);
			}
		}
	}
	const computedC2MaintenanceDigest = `sha256:${createHash("sha256")
		.update(`${requiredC2MaintenancePaths.join("\n")}\n`, "utf8").digest("hex")}`;
	if (computedC2MaintenanceDigest !== requiredC2MaintenanceDigest) {
		errors.push(`internal C2M roster digest mismatch: ${computedC2MaintenanceDigest}`);
	}
	for (const [path, snippets] of Object.entries(requiredC3PText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C3P ruling: ${JSON.stringify(snippet)}`);
		}
	}
	const computedC2Digest = `sha256:${createHash("sha256").update(`${requiredC2Paths.join("\n")}\n`, "utf8").digest("hex")}`;
	if (computedC2Digest !== requiredC2Digest) errors.push(`internal C2 roster digest mismatch: ${computedC2Digest}`);
	const computedC2BDigest = `sha256:${createHash("sha256").update(`${requiredC2BPaths.join("\n")}\n`, "utf8").digest("hex")}`;
	if (computedC2BDigest !== requiredC2BDigest) errors.push(`internal C2B roster digest mismatch: ${computedC2BDigest}`);
	for (const [unit, paths, digest] of [
		["C3P", requiredC3PPaths, requiredC3PDigest],
		["C3PB", requiredC3PBPaths, requiredC3PBDigest],
		["C3", requiredC3Paths, requiredC3Digest],
		["C3B", requiredC3BPaths, requiredC3BDigest],
	]) {
		const computed = `sha256:${createHash("sha256").update(`${paths.join("\n")}\n`, "utf8").digest("hex")}`;
		if (computed !== digest) errors.push(`internal ${unit} roster digest mismatch: ${computed}`);
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
		requireExactlyOnce(
			verificationBody,
			requiredC2MaintenanceDeclaration,
			"docs/VERIFICATION.md",
			"C2 receipt-phase maintenance scope declaration",
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
	const c2MaintenanceStatusBody = bodies.get(c2MaintenanceStatusPath);
	if (c2MaintenanceStatusBody !== undefined) {
		requireExactlyOnce(
			c2MaintenanceStatusBody,
			requiredC2MaintenanceDeclaration,
			c2MaintenanceStatusPath,
			"C2 receipt-phase maintenance scope declaration",
			errors,
		);
		for (const label of c2MaintenanceClaimLabels) {
			requireExactlyOnce(
				c2MaintenanceStatusBody,
				`| \`${label}\` |`,
				c2MaintenanceStatusPath,
				"C2 receipt-phase maintenance intended claim map",
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
	let c2Status;
	try {
		c2Status = await readText(root, c2StatusPath, overrides);
		for (const snippet of [
			"C2 is the Darwin-private persistence substrate",
			"C2 production never issues official target or permit authority",
			"`ExecutionInterlockClearReceipt`",
			"compiler-parsed Go AST enumeration",
			requiredC2Digest,
			requiredC2BDigest,
		]) {
			if (!c2Status.includes(snippet)) errors.push(`${c2StatusPath}: missing required C2 boundary: ${JSON.stringify(snippet)}`);
		}
	} catch (error) {
		errors.push(`${c2StatusPath}: unreadable (${error.message})`);
	}
	let c3pStatus;
	try {
		c3pStatus = await readText(root, c3pStatusPath, overrides);
	} catch (error) {
		errors.push(`${c3pStatusPath}: unreadable (${error.message})`);
	}

	const status = bodies.get(statusPath);
	const handoff = bodies.get("docs/HANDOFF_MODE_C.md");
	if (status !== undefined && handoff !== undefined) {
		await checkStatusPhase(root, overrides, status, handoff, errors, receiptAuthority);
			if (c1Status !== undefined) {
				await checkC1ReceiptPhase(root, overrides, c1Status, handoff, errors, c1ReceiptAuthority);
			}
			if (c2Status !== undefined) {
				await checkC2ReceiptPhase(root, overrides, c2Status, handoff, errors, c2ReceiptAuthority);
			}
			if (c3pStatus !== undefined) {
				await checkC3PReceiptPhase(root, overrides, c3pStatus, handoff, errors, c3pReceiptAuthority);
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
		if (!isDeepStrictEqual(specification.units.C2.exact, requiredC2Paths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C2 exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C2.prefixes, []) || specification.units.C2.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C2 source profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C2M.exact, requiredC2MaintenancePaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C2M exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C2M.prefixes, []) ||
			specification.units.C2M.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C2M source profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C2B.exact, requiredC2BPaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C2B exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C2B.prefixes, []) ||
			specification.units.C2B.verification_profile !== "RECEIPT_RECONCILIATION") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C2B receipt profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C2B.receipt_claims, requiredC2BReceiptClaims)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C2B receipt claim manifest mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3P.exact, requiredC3PPaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3P exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3P.prefixes, []) || specification.units.C3P.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3P source profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3PB.exact, requiredC3PBPaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3PB exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3PB.prefixes, []) ||
			specification.units.C3PB.verification_profile !== "RECEIPT_RECONCILIATION") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3PB receipt profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3PB.receipt_claims, requiredC3PBReceiptClaims)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3PB receipt claim manifest mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3.exact, requiredC3Paths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3 exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3.prefixes, []) || specification.units.C3.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3 source profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3B.exact, requiredC3BPaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3B exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3B.prefixes, []) ||
			specification.units.C3B.verification_profile !== "RECEIPT_RECONCILIATION") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3B receipt profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3B.receipt_claims, requiredC3BReceiptClaims)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3B receipt claim manifest mismatch");
		}
	} catch (error) {
		errors.push(`spec/verification/p07b-c-unit-paths.json: invalid (${error.message})`);
	}

	return errors;
}

function readSealedC2StatusFixture() {
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
	const run = (args, maxBuffer = 1024 * 1024) => {
		const result = spawnSync("/usr/bin/git", ["--no-replace-objects", ...args], {
			cwd: repositoryRoot,
			encoding: "utf8",
			timeout: 30_000,
			maxBuffer,
			env,
		});
		if (result.error || result.signal || result.status !== 0 || result.stderr !== "") {
			throw new Error(`P07B-C C2 sealed-source fixture git ${args[0]} failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
		}
		return result.stdout;
	};
	const tree = run(["rev-parse", "--verify", `${sealedC2Identity.commit}^{tree}`]).trim();
	if (tree !== sealedC2Identity.tree) throw new Error(`P07B-C C2 sealed-source fixture tree drift: ${tree}`);
	const status = run(["cat-file", "blob", `${sealedC2Identity.commit}:${c2StatusPath}`], 2 * 1024 * 1024);
	if (!status.endsWith("\n") || Buffer.byteLength(status, "utf8") > 1024 * 1024) {
		throw new Error("P07B-C C2 sealed-source status fixture is not bounded canonical text");
	}
	return status;
}

function withoutC2ReceiptBlock(body) {
	const pattern = /\n?<!-- P07B-C-C2-SOURCE-RECEIPTS:START -->[\s\S]*?<!-- P07B-C-C2-SOURCE-RECEIPTS:END -->\n?/gu;
	const matches = [...body.matchAll(pattern)];
	if (matches.length > 1) throw new Error("P07B-C C2 sealed-source phase fixture found duplicate receipt blocks");
	const stripped = matches.length === 0 ? body : body.replace(pattern, "\n");
	if (stripped.includes("<!-- P07B-C-C2-SOURCE-RECEIPTS:START -->") ||
		stripped.includes("<!-- P07B-C-C2-SOURCE-RECEIPTS:END -->")) {
		throw new Error("P07B-C C2 sealed-source phase fixture found an unmatched receipt marker");
	}
	return stripped;
}

function replaceFixtureExactlyOnce(body, needle, replacement, name) {
	const occurrences = countOccurrences(body, needle);
	if (occurrences !== 1) {
		throw new Error(`P07B-C C2 phase fixture ${name} anchor count ${occurrences}`);
	}
	return body.replace(needle, replacement);
}

async function syntheticC2SealedSourcePhaseFixture() {
	const sourceStatus = readSealedC2StatusFixture();
	const currentHandoff = await readText(repositoryRoot, "docs/HANDOFF_MODE_C.md", new Map());
	const sourcePhaseHandoff = withoutC2ReceiptBlock(currentHandoff);
	return new Map([
		[c2StatusPath, sourceStatus],
		["docs/HANDOFF_MODE_C.md", sourcePhaseHandoff],
		[c2ReceiptDeclarationPath, ABSENT_FIXTURE_PATH],
	]);
}

async function runC2SealedSourcePhaseSelfTest() {
	const baseOverrides = await syntheticC2SealedSourcePhaseFixture();
	const baseline = await checkPlan(repositoryRoot, baseOverrides);
	if (baseline.length > 0) {
		throw new Error(`P07B-C C2 sealed-source phase self-test baseline failed:\n${baseline.join("\n")}`);
	}
	const cases = [
		{
			name: "handoff source digest drift",
			path: "docs/HANDOFF_MODE_C.md",
			needle: requiredC2Digest,
			replacement: `sha256:${"0".repeat(64)}`,
			expect: "C2 source scope digest",
		},
		{
			name: "handoff receipt digest drift",
			path: "docs/HANDOFF_MODE_C.md",
			needle: requiredC2BDigest,
			replacement: `sha256:${"0".repeat(64)}`,
			expect: "C2B receipt scope digest",
		},
		{
			name: "status source digest drift",
			path: c2StatusPath,
			needle: requiredC2Digest,
			replacement: `sha256:${"0".repeat(64)}`,
			expect: "C2 source scope",
		},
		{
			name: "status pending grade drift",
			path: c2StatusPath,
			needle: `| \`${c2ClaimLabels[0]}\` | \`${c2ClaimTypes[0]}\` | \`UNRECEIPTED\` |`,
			replacement: `| \`${c2ClaimLabels[0]}\` | \`${c2ClaimTypes[0]}\` | \`TREE-EXACT\` |`,
			expect: "C2 pending source receipt map",
		},
	];
	for (const testCase of cases) {
		const overrides = new Map(baseOverrides);
		const body = overrides.get(testCase.path);
		overrides.set(testCase.path, replaceFixtureExactlyOnce(
			body,
			testCase.needle,
			testCase.replacement,
			testCase.name,
		));
		const errors = await checkPlan(repositoryRoot, overrides);
		if (!errors.some((error) => error.includes(testCase.expect))) {
			throw new Error(`P07B-C C2 sealed-source phase self-test false negative: ${testCase.name} (${errors.join("; ")})`);
		}
	}
	return cases.length;
}

function syntheticC2ReceiptFixture() {
	const sourceCommit = "a".repeat(40);
	const sourceTree = "b".repeat(40);
	const sourceDiff = requiredC2Paths.map((path, index) => {
		const added = requiredC2AddedPaths.has(path);
		return {
			old_mode: added ? "000000" : "100644",
			new_mode: "100644",
			old_oid: added ? "0".repeat(40) : (index + 1).toString(16).padStart(40, "1").slice(-40),
			new_oid: (index + 1).toString(16).padStart(40, "2").slice(-40),
			status: added ? "A" : "M",
			path,
		};
	});
	const receipt = {
		schema_version: "countershape/p07b-c-c2-receipt/v1",
		source_commit: sourceCommit,
		source_tree: sourceTree,
		source_subject: "feat: add contract execution persistence substrate",
		strict_exit_code: 0,
		strict_claims_recorded_exact: c2ClaimLabels.length,
		strict_claims_total: c2ClaimLabels.length,
		coverage_complete_events: c2ClaimLabels.length,
		coverage_total_events: c2ClaimLabels.length,
		secrets_override: false,
		claims: c2ClaimLabels.map((label, index) => ({
			label, type: c2ClaimTypes[index], supporting_event_index: index, grade: "TREE-EXACT",
		})),
		evidence: {
			html_report: {
				path: `.countershape/evidence/p07b-c-c2-final-${sourceCommit.slice(0, 12)}.html`,
				sha256: "c".repeat(64), bytes: 8192,
				authority: "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS",
			},
			ledger_archive: {
				path: ".didrun-history/2026-07-19-p07b-c-c2-source/.didrun/",
				authority: "LOCAL_IGNORED_ARCHIVE_NOT_GIT_AUTHORITY",
				manifest_format: "sorted-relative-posix-path-tab-size-tab-sha256-newline/v1",
				sealed_event_count: c2ClaimLabels.length,
				archive_session_event_count: c2ClaimLabels.length + 1,
				claim_count: c2ClaimLabels.length,
				seal_count: 1,
				object_file_count: 20,
				total_file_count: 24,
				total_bytes: 100_000,
				all_files_manifest_sha256: "d".repeat(64),
				objects_manifest_sha256: "e".repeat(64),
				gitignore_sha256: "f".repeat(64),
				session_log_sha256: "1".repeat(64),
				claims_jsonl_sha256: "2".repeat(64),
				seals_jsonl_sha256: "3".repeat(64),
			},
			git_note: { version: 1, blob_oid: "4".repeat(40), body_sha256: "5".repeat(64) },
			seal_redaction: {
				finding_count: 0,
				finding_kinds: [],
				finding_provenance: "LOCAL_TERMINAL_OBSERVATION_NOT_GIT_NOTE",
				structured_staged_scan_findings: 0,
				structured_scan_provenance: "SEALED_C2_SUPPORTING_EVENT_12",
			},
		},
	};
	const authority = {
		commit: sourceCommit,
		tree: sourceTree,
		parent: sealedC1BIdentity.commit,
		parents: [sealedC1BIdentity.commit],
		source_diff: sourceDiff,
		subject: receipt.source_subject,
		ancestor_of_head: true,
		note_blob_oid: receipt.evidence.git_note.blob_oid,
		note_body_sha256: receipt.evidence.git_note.body_sha256,
		note: {
			version: 1,
			commit: sourceCommit,
			tree: sourceTree,
			secrets_override: false,
			claims: c2ClaimLabels.map((label, index) => ({
				claim: {
					label, ctype: c2ClaimTypes[index], declared_at_index: index, event_indices: [index],
					pathspecs: [], argv_preview: [...c2ExpectedClaimArgv[index]],
				},
				supporting_event_index: index,
				grade: "tree-exact",
				exit_code: 0,
				reason: "self-stable command ran against the sealed tree",
				delta: [],
			})),
			coverage: { total_events: c2ClaimLabels.length, by_coverage: { complete: c2ClaimLabels.length } },
		},
	};
	const disclosures = c2ReceiptDisclosureLines(receipt);
	const receiptRows = receipt.claims.map((claim) => `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`).join("\n");
	const status = [
		`- **Source receipt:** C2 source commit \`${sourceCommit}\`, tree \`${sourceTree}\`, is sealed, note-present, and strict-clean; every C2 source grade below is \`TREE-EXACT\`. This C2B receipt-document working unit binds only that existing source and remains \`UNRECEIPTED\` until its own commit, seal, note, and strict boundary.`,
		"## C2 source receipt map",
		"- C2 source strict exit: `0`",
		`- C2 source strict claims: \`${c2ClaimLabels.length}/${c2ClaimLabels.length} claims recorded-exact\``,
		"This receipt binds only the already-existing C2 source commit. C2B cannot name or grade its own commit, tree, Git note, or strict result.",
		receiptRows,
		...disclosures,
	].join("\n");
	const handoff = [
		"<!-- P07B-C-C2-SOURCE-RECEIPTS:START -->",
		`C2 source commit \`${sourceCommit}\`, tree \`${sourceTree}\`.`,
		`C2 strict claims: \`${c2ClaimLabels.length}/${c2ClaimLabels.length} claims recorded-exact\`; strict exit: \`0\`.`,
		receiptRows,
		...disclosures,
		"<!-- P07B-C-C2-SOURCE-RECEIPTS:END -->",
	].join("\n");
	return { receipt, authority, status, handoff };
}

async function syntheticC2ReceiptPlanFixture(fixture) {
	let status = readSealedC2StatusFixture();
	const sourceState = "- **State:** active pre-seal source boundary; every C2 grade below is `UNRECEIPTED`";
	const receiptState = `- **Source receipt:** C2 source commit \`${fixture.receipt.source_commit}\`, tree \`${fixture.receipt.source_tree}\`, is sealed, note-present, and strict-clean; every C2 source grade below is \`TREE-EXACT\`. This C2B receipt-document working unit binds only that existing source and remains \`UNRECEIPTED\` until its own commit, seal, note, and strict boundary.`;
	status = replaceFixtureExactlyOnce(status, sourceState, receiptState, "receipt-present status state");
	const receiptMetadata = [
		"## C2 source receipt map",
		"- C2 source strict exit: `0`",
		`- C2 source strict claims: \`${fixture.receipt.strict_claims_recorded_exact}/${fixture.receipt.strict_claims_total} claims recorded-exact\``,
		"This receipt binds only the already-existing C2 source commit. C2B cannot name or grade its own commit, tree, Git note, or strict result.",
		...c2ReceiptDisclosureLines(fixture.receipt),
	].join("\n");
	status = replaceFixtureExactlyOnce(status, "## Intended final receipt map", receiptMetadata, "receipt-present status heading");
	for (const claim of fixture.receipt.claims) {
		status = replaceFixtureExactlyOnce(
			status,
			`| \`${claim.label}\` | \`${claim.type}\` | \`UNRECEIPTED\` |`,
			`| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`,
			`receipt-present row ${claim.label}`,
		);
	}
	const currentHandoff = withoutC2ReceiptBlock(await readText(repositoryRoot, "docs/HANDOFF_MODE_C.md", new Map()));
	const handoff = `${currentHandoff.trimEnd()}\n\n${fixture.handoff}\n`;
	return {
		overrides: new Map([
			[c2StatusPath, status],
			["docs/HANDOFF_MODE_C.md", handoff],
			[c2ReceiptDeclarationPath, `${JSON.stringify(fixture.receipt)}\n`],
		]),
		authority: fixture.authority,
	};
}

async function runC2ReceiptCheckerSelfTest() {
	const fixture = syntheticC2ReceiptFixture();
	let rejected = 0;
	const requireError = (name, errors, expected) => {
		if (!errors.some((error) => error.includes(expected))) {
			throw new Error(`P07B-C C2 receipt checker self-test false negative: ${name} (${errors.join("; ")})`);
		}
		rejected += 1;
	};
	const declarationBaseline = validateC2ReceiptDeclaration(fixture.receipt);
	if (declarationBaseline.length > 0) throw new Error(`P07B-C C2 receipt declaration self-test baseline failed: ${declarationBaseline.join(", ")}`);
	const authorityBaseline = validateC2ReceiptAuthority(fixture.receipt, fixture.authority);
	if (authorityBaseline.length > 0) throw new Error(`P07B-C C2 receipt authority self-test baseline failed: ${authorityBaseline.join(", ")}`);
	const redactedAuthority = structuredClone(fixture.authority);
	for (let index = 0; index < 9; index += 1) {
		const preview = redactedAuthority.note.claims[index].claim.argv_preview;
		for (let position = 2; position <= 7; position += 1) {
			const suffix = preview[position].slice(preview[position].indexOf(".countershape/"));
			preview[position] = position === 5
				? `«redacted:high-entropy»${suffix}`
				: "«redacted:high-entropy».«redacted:high-entropy»";
		}
		for (let position = 22; position <= 24; position += 1) preview[position] = "«redacted:high-entropy»";
	}
	const redactedBaseline = validateC2ReceiptAuthority(fixture.receipt, redactedAuthority);
	if (redactedBaseline.length > 0) throw new Error(`P07B-C C2 redacted-preview self-test baseline failed: ${redactedBaseline.join(", ")}`);
	const phaseBaseline = [];
	await checkC2ReceiptPhase(
		repositoryRoot,
		new Map([[c2ReceiptDeclarationPath, `${JSON.stringify(fixture.receipt)}\n`]]),
		fixture.status,
		fixture.handoff,
		phaseBaseline,
		fixture.authority,
	);
	if (phaseBaseline.length > 0) throw new Error(`P07B-C C2 receipt phase self-test baseline failed: ${phaseBaseline.join(", ")}`);
	const planFixture = await syntheticC2ReceiptPlanFixture(fixture);
	const planBaseline = await checkPlan(repositoryRoot, planFixture.overrides, undefined, undefined, planFixture.authority);
	if (planBaseline.length > 0) throw new Error(`P07B-C C2 receipt-present full-plan self-test baseline failed: ${planBaseline.join(", ")}`);
	const hostilePlanOverrides = new Map(planFixture.overrides);
	hostilePlanOverrides.set(
		"docs/HANDOFF_MODE_C.md",
		replaceFixtureExactlyOnce(
			hostilePlanOverrides.get("docs/HANDOFF_MODE_C.md"),
			`C2 source commit \`${fixture.receipt.source_commit}\`, tree \`${fixture.receipt.source_tree}\`.`,
			`C2 source commit \`${fixture.receipt.source_commit}\`, tree \`${"7".repeat(40)}\`.`,
			"receipt-present full-plan handoff identity",
		),
	);
	const hostilePlanErrors = await checkPlan(repositoryRoot, hostilePlanOverrides, undefined, undefined, planFixture.authority);
	requireError("receipt-present full-plan handoff wiring", hostilePlanErrors, "C2B source identity");

	const wrongSchema = structuredClone(fixture.receipt);
	wrongSchema.schema_version = "countershape/p07b-c-c2-receipt/v2";
	requireError("schema", validateC2ReceiptDeclaration(wrongSchema), "schema version");
	const wrongClaim = structuredClone(fixture.receipt);
	wrongClaim.claims[0].grade = "UNRECEIPTED";
	requireError("claim grade", validateC2ReceiptDeclaration(wrongClaim), "claim 1");
	const wrongEvidence = structuredClone(fixture.receipt);
	wrongEvidence.evidence.html_report.sha256 = "wrong";
	requireError("HTML evidence", validateC2ReceiptDeclaration(wrongEvidence), "local HTML declaration");
	const wrongParent = structuredClone(fixture.authority);
	wrongParent.parent = "6".repeat(40);
	requireError("Git parent", validateC2ReceiptAuthority(fixture.receipt, wrongParent), "source parent");
	const wrongNote = structuredClone(fixture.authority);
	wrongNote.note.claims[0].grade = "unreceipted";
	requireError("note grade", validateC2ReceiptAuthority(fixture.receipt, wrongNote), "didrun note claim 1");
	const swappedProfile = structuredClone(fixture.authority);
	swappedProfile.note.claims[0].claim.argv_preview = [...c2ExpectedClaimArgv[1]];
	requireError("profile command substitution", validateC2ReceiptAuthority(fixture.receipt, swappedProfile), "didrun note claim 1 argv");
	const benignVerifierSubstitution = structuredClone(fixture.authority);
	benignVerifierSubstitution.note.claims[10].claim.argv_preview = ["/usr/bin/true"];
	requireError("cumulative command substitution", validateC2ReceiptAuthority(fixture.receipt, benignVerifierSubstitution), "didrun note claim 11 argv");
	const appendedArgument = structuredClone(fixture.authority);
	appendedArgument.note.claims[5].claim.argv_preview.push("--extra");
	requireError("appended command argument", validateC2ReceiptAuthority(fixture.receipt, appendedArgument), "didrun note claim 6 argv");
	const nonStringArgument = structuredClone(fixture.authority);
	nonStringArgument.note.claims[7].claim.argv_preview[0] = 7;
	requireError("non-string command argument", validateC2ReceiptAuthority(fixture.receipt, nonStringArgument), "didrun note claim 8 argv");
	const mergeParent = structuredClone(fixture.authority);
	mergeParent.parents.push("6".repeat(40));
	requireError("merge source", validateC2ReceiptAuthority(fixture.receipt, mergeParent), "source parents");
	for (const [name, mutate, expected] of [
		["extra source path", (authority) => { authority.source_diff.push({ ...authority.source_diff.at(-1), path: "wrong/extra.go" }); }, "source diff exact path roster"],
		["missing source path", (authority) => { authority.source_diff.shift(); }, "source diff exact path roster"],
		["duplicate source path", (authority) => { authority.source_diff[1].path = authority.source_diff[0].path; }, "source diff exact path roster"],
		["added status drift", (authority) => { authority.source_diff.find((entry) => entry.status === "A").status = "M"; }, "source diff added row"],
		["modified status drift", (authority) => { authority.source_diff.find((entry) => entry.status === "M").status = "A"; }, "source diff modified row"],
		["executable final mode", (authority) => { authority.source_diff[0].new_mode = "100755"; }, "source diff modified row"],
		["symlink final mode", (authority) => { authority.source_diff[0].new_mode = "120000"; }, "source diff modified row"],
		["submodule final mode", (authority) => { authority.source_diff[0].new_mode = "160000"; }, "source diff modified row"],
		["deleted final object", (authority) => { authority.source_diff[0].status = "D"; authority.source_diff[0].new_mode = "000000"; authority.source_diff[0].new_oid = "0".repeat(40); }, "source diff modified row"],
		["unchanged modified object", (authority) => { authority.source_diff.find((entry) => entry.status === "M").new_oid = authority.source_diff.find((entry) => entry.status === "M").old_oid; }, "source diff modified row"],
	]) {
		const hostile = structuredClone(fixture.authority);
		mutate(hostile);
		requireError(name, validateC2ReceiptAuthority(fixture.receipt, hostile), expected);
	}
	for (const [name, status, handoff, expected] of [
		["status disagreement", fixture.status.replace(fixture.receipt.claims[0].label, "wrong label"), fixture.handoff, "C2B source receipt map"],
		["handoff disagreement", fixture.status, fixture.handoff.replace(fixture.receipt.source_tree, "7".repeat(40)), "C2B source identity"],
		["C2B self receipt", `${fixture.status}\nC2B commit \`${"8".repeat(40)}\``, fixture.handoff, "C2B self-receipt claim"],
	]) {
		const errors = [];
		await checkC2ReceiptPhase(
			repositoryRoot,
			new Map([[c2ReceiptDeclarationPath, `${JSON.stringify(fixture.receipt)}\n`]]),
			status,
			handoff,
			errors,
			fixture.authority,
		);
		requireError(name, errors, expected);
	}
	return rejected;
}

async function runC2LocalEvidencePositiveSelfTest() {
	const root = await mkdtemp(resolve(tmpdir(), "countershape-c2-evidence-"));
	try {
		const fixture = await buildLocalEvidenceFixture(root, {
			claimLabels: c2ClaimLabels,
			claimTypes: c2ClaimTypes,
			claimArgv: c2ExpectedClaimArgv,
			sourceCommit: "a".repeat(40),
			sourceTree: "b".repeat(40),
			archiveSessionEventCount: c2ClaimLabels.length + 1,
		});
		const baseline = await verifyLocalEvidence(fixture.root, fixture.receipt, c2ClaimLabels, c2ClaimTypes, c2ExpectedClaimArgv);
		if (baseline.length > 0) throw new Error(`P07B-C C2 local-evidence self-test baseline failed: ${baseline.join(", ")}`);
		const hostileClaims = structuredClone(fixture.claims);
		hostileClaims[0].label = "wrong label";
		await writeJSONLines(fixture.claimsPath, hostileClaims);
		const errors = await verifyLocalEvidence(fixture.root, fixture.receipt, c2ClaimLabels, c2ClaimTypes, c2ExpectedClaimArgv);
		if (!errors.some((error) => error.includes("local ledger claim 1"))) {
			throw new Error(`P07B-C C2 local-evidence self-test false negative: ${errors.join(", ")}`);
		}
		await writeJSONLines(fixture.claimsPath, fixture.claims);
		const hostileSessions = structuredClone(fixture.sessions);
		hostileSessions[0].event.argv = ["/usr/bin/true"];
		await writeJSONLines(fixture.sessionPath, hostileSessions);
		const argvErrors = await verifyLocalEvidence(fixture.root, fixture.receipt, c2ClaimLabels, c2ClaimTypes, c2ExpectedClaimArgv);
		if (!argvErrors.some((error) => error.includes("local ledger event 1 argv"))) {
			throw new Error(`P07B-C C2 local-evidence argv self-test false negative: ${argvErrors.join(", ")}`);
		}
		return 2;
	} finally {
		await rm(root, { recursive: true, force: true });
	}
}

async function mutatedText(path, transform) {
	return transform(await readText(repositoryRoot, path, new Map()));
}

async function runSelfTest() {
	const baseline = await checkPlan();
	if (baseline.length > 0) throw new Error(`P07B-C evolved plan checker self-test baseline failed:\n${baseline.join("\n")}`);
	const localEvidenceMutations = await runLocalEvidenceSelfTest();
	const receiptModePhaseMutations = await runC1ReceiptModePhaseSelfTest();
	const c2ReceiptMutations = await runC2ReceiptCheckerSelfTest();
	const c2SealedSourcePhaseMutations = await runC2SealedSourcePhaseSelfTest();
	const c2LocalEvidenceMutations = await runC2LocalEvidencePositiveSelfTest();
	const c3pReceiptMutations = await runC3PReceiptSelfTest();

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
			name: "current prompt-pack prerequisite-gate removal", path: "docs/PROMPT_PACK.md",
			value: currentPromptPack.replace("C3P prerequisite declared", "C3P prerequisite omitted"),
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
				name: "C2 receipt checker enrollment removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C2.exact = candidate.units.C2.exact.filter((path) => path !== "tools/check-p07b-c-plan.mjs");
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C2 exact path roster mismatch",
			},
			{
				name: "C2 source profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C2.verification_profile = "RECEIPT_RECONCILIATION";
					candidate.units.C2.receipt_claims = structuredClone(requiredC2BReceiptClaims);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C2M unit removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					delete candidate.units.C2M;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C2M unit order drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					const entries = Object.entries(candidate.units);
					const c2m = entries.find(([unit]) => unit === "C2M");
					candidate.units = Object.fromEntries([
						...entries.filter(([unit]) => unit !== "C2M" && unit !== "C2B"),
						entries.find(([unit]) => unit === "C2B"),
						c2m,
					]);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C2M exact path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C2M.exact = candidate.units.C2M.exact.filter((path) => path !== c2MaintenanceStatusPath);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C2M exact path roster mismatch",
			},
			{
				name: "C2M prefix introduction", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C2M.prefixes = ["tools/"];
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C2M receipt-profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C2M.verification_profile = "RECEIPT_RECONCILIATION";
					candidate.units.C2M.receipt_claims = structuredClone(requiredC2BReceiptClaims);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C2B unit removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					delete candidate.units.C2B;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C2B unit order drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					const entries = Object.entries(candidate.units);
					const c2b = entries.find(([unit]) => unit === "C2B");
					candidate.units = Object.fromEntries([
						...entries.filter(([unit]) => unit !== "C2B" && unit !== "C3"),
						entries.find(([unit]) => unit === "C3"),
						c2b,
					]);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C2B receipt path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C2B.exact = candidate.units.C2B.exact.filter((path) => path !== c2ReceiptDeclarationPath);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C2B exact path roster mismatch",
			},
			{
				name: "C2B source profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C2B.verification_profile = "SOURCE_FULL";
					delete candidate.units.C2B.receipt_claims;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C2B receipt profile mismatch",
			},
			{
				name: "C2B receipt claim label drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C2B.receipt_claims[0].label = "P07B-C C2B source receipt reconciliation altered";
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C2B receipt claim manifest mismatch",
			},
			{
				name: "C3P unit removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					delete candidate.units.C3P;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3P exact path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3P.exact = candidate.units.C3P.exact.filter((path) => path !== "tools/check-p07b-c-c3p-receipt.mjs");
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3P exact path roster mismatch",
			},
			{
				name: "C3P prefix introduction", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3P.prefixes = ["docs/"];
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3PB receipt claim drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3PB.receipt_claims[0].label += " altered";
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3PB receipt claim manifest mismatch",
			},
			{
				name: "C3 directory prefix restoration", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3.prefixes = ["internal/hostepoch/"];
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3 compiler-frozen store path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3.exact = candidate.units.C3.exact.filter((path) => path !== "internal/store/public_api_test.go");
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3 exact path roster mismatch",
			},
			{
				name: "C3B source profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3B.verification_profile = "SOURCE_FULL";
					delete candidate.units.C3B.receipt_claims;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3B receipt profile mismatch",
			},
		{
			name: "legacy boot profile restored in schema", path: c1PlanningPaths.targetSchema,
			value: (await readText(repositoryRoot, c1PlanningPaths.targetSchema, new Map()))
				.replace("DARWIN_KERN_BOOTSESSIONUUID_V1", "DARWIN_KERN_BOOTTIME_V1"),
			expect: "schema authority",
		},
		{
			name: "legacy boot profile restored in example", path: c1PlanningPaths.targetExample,
			value: (await readText(repositoryRoot, c1PlanningPaths.targetExample, new Map()))
				.replace("DARWIN_KERN_BOOTSESSIONUUID_V1", "DARWIN_KERN_BOOTTIME_V1"),
			expect: "example graph",
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
			throw new Error(`P07B-C evolved plan checker self-test false negative: ${testCase.name}`);
		}
	}

	console.log(`P07B-C evolved plan checker self-test passed: ${cases.length + localEvidenceMutations + receiptModePhaseMutations + c2ReceiptMutations + c2SealedSourcePhaseMutations + c2LocalEvidenceMutations + c3pReceiptMutations} authority, structure, C1/C2/C3P local-evidence, phase-isolation, semantic-projection, receipt, and allowlist mutations rejected`);
}

async function verifyC2MaintenancePresealLedger() {
	const expectedInvocation = c2MaintenanceExpectedClaimArgv.at(-1);
	const observedInvocation = [
		process.argv0,
		relative(repositoryRoot, resolve(process.argv[1])),
		...process.argv.slice(2),
	];
	const requestedRuntime = await realpath(process.argv0);
	const runningRuntime = await realpath(process.execPath);
	if (!isDeepStrictEqual(observedInvocation, expectedInvocation) || requestedRuntime !== runningRuntime) {
		throw new Error(`P07B-C C2M live chain-gate invocation mismatch: ${JSON.stringify({ argv: observedInvocation, requestedRuntime, runningRuntime })}`);
	}
	const python = "/Users/drewnelson/.venvs/didrun/bin/python";
	const program = String.raw`
from pathlib import Path
import json
import os
import subprocess
from didrun.ledger import Session

root = Path(".didrun")
session = Session(root)
ok, broken = session.verify_chain()
assert ok and broken is None, (ok, broken)
entries = list(session.entries())
assert [entry.index for entry in entries] == list(range(7))
events = [entry.event for entry in entries]
assert all(event.exit_code == 0 for event in events)
assert all(event.observed_via == "wrapper" for event in events)
assert all(event.coverage == "complete" for event in events)
assert all(event.self_stable() for event in events)
assert not any(event.submodule_dirty for event in events)
tree = events[0].tree_before
assert all(event.tree_before == tree and event.tree_after == tree for event in events)
assert subprocess.check_output(["/usr/bin/git", "write-tree"], text=True).strip() == tree
expected_argv = json.loads(os.environ["COUNTERSHAPE_C2M_EXPECTED_ARGV"])
assert [list(event.argv) for event in events] == expected_argv
expected_claims = json.loads(os.environ["COUNTERSHAPE_C2M_EXPECTED_CLAIMS"])
claims = [json.loads(line) for line in (root / "claims.jsonl").read_text(encoding="utf8").splitlines() if line]
assert len(claims) == len(expected_claims) == 7
for index, (claim, expected) in enumerate(zip(claims, expected_claims, strict=True)):
    assert claim["ctype"] == expected["type"]
    assert claim["label"] == expected["label"]
    assert claim["declared_at_index"] == index
    assert claim["event_indices"] == [index]
    assert claim["pathspecs"] == []
assert not (root / "seals.jsonl").exists()
print(f"P07B-C C2M preceding chain exact: events=7 claims=7 green=7 tree={tree} chain=valid seal=absent")
`;
	const expectedClaims = c2MaintenanceClaimLabels.slice(0, 7)
		.map((label, index) => ({ label, type: c2MaintenanceClaimTypes[index] }));
	const result = spawnSync(python, ["-c", program], {
		cwd: repositoryRoot,
		encoding: "utf8",
		timeout: 30_000,
		maxBuffer: 4 * 1024 * 1024,
		env: {
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
			COUNTERSHAPE_C2M_EXPECTED_ARGV: JSON.stringify(c2MaintenanceExpectedClaimArgv.slice(0, 7)),
			COUNTERSHAPE_C2M_EXPECTED_CLAIMS: JSON.stringify(expectedClaims),
		},
	});
	if (result.error || result.signal || result.status !== 0 || result.stderr !== "" ||
		!/^P07B-C C2M preceding chain exact: events=7 claims=7 green=7 tree=[0-9a-f]{40,64} chain=valid seal=absent\n$/u.test(result.stdout)) {
		throw new Error(`P07B-C C2M preceding-ledger verification failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"}): ${result.stderr || result.stdout}`);
	}
	process.stdout.write(result.stdout);
}

async function verifyC2PresealLedger() {
	const python = "/Users/drewnelson/.venvs/didrun/bin/python";
	const program = String.raw`
from pathlib import Path
import json
import os
import subprocess
from didrun.ledger import Session

root = Path(".didrun")
session = Session(root)
ok, broken = session.verify_chain()
assert ok and broken is None, (ok, broken)
entries = list(session.entries())
assert [entry.index for entry in entries] == list(range(13))
events = [entry.event for entry in entries]
assert all(event.exit_code == 0 for event in events)
assert all(event.observed_via == "wrapper" for event in events)
assert all(event.coverage == "complete" for event in events)
assert all(event.self_stable() for event in events)
assert not any(event.submodule_dirty for event in events)
tree = events[0].tree_before
assert all(event.tree_before == tree and event.tree_after == tree for event in events)
assert subprocess.check_output(["/usr/bin/git", "write-tree"], text=True).strip() == tree
expected_argv = json.loads(os.environ["COUNTERSHAPE_C2_EXPECTED_ARGV"])
assert [list(event.argv) for event in events] == expected_argv
expected_claims = json.loads(os.environ["COUNTERSHAPE_C2_EXPECTED_CLAIMS"])
claims = [json.loads(line) for line in (root / "claims.jsonl").read_text(encoding="utf8").splitlines() if line]
assert len(claims) == len(expected_claims) == 13
for index, (claim, expected) in enumerate(zip(claims, expected_claims, strict=True)):
    assert claim["ctype"] == expected["type"]
    assert claim["label"] == expected["label"]
    assert claim["declared_at_index"] == index
    assert claim["event_indices"] == [index]
    assert claim["pathspecs"] == []
assert not (root / "seals.jsonl").exists()
print(f"P07B-C C2 preceding chain exact: events=13 claims=13 green=13 tree={tree} chain=valid seal=absent")
`;
	const expectedClaims = c2ClaimLabels.slice(0, 13).map((label, index) => ({ label, type: c2ClaimTypes[index] }));
	const result = spawnSync(python, ["-c", program], {
		cwd: repositoryRoot,
		encoding: "utf8",
		timeout: 30_000,
		maxBuffer: 4 * 1024 * 1024,
		env: {
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
			COUNTERSHAPE_C2_EXPECTED_ARGV: JSON.stringify(c2ExpectedClaimArgv.slice(0, 13)),
			COUNTERSHAPE_C2_EXPECTED_CLAIMS: JSON.stringify(expectedClaims),
		},
	});
	if (result.error || result.signal || result.status !== 0 || result.stderr !== "" ||
		!/^P07B-C C2 preceding chain exact: events=13 claims=13 green=13 tree=[0-9a-f]{40,64} chain=valid seal=absent\n$/u.test(result.stdout)) {
		throw new Error(`P07B-C C2 preceding-ledger verification failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"}): ${result.stderr || result.stdout}`);
	}
	process.stdout.write(result.stdout);
}

async function verifyLivePresealLedger({ phase, expectedArgv, claims, receiptPresent }) {
	const expectedInvocation = expectedArgv.at(-1);
	const observedInvocation = [
		process.argv0,
		relative(repositoryRoot, resolve(process.argv[1])),
		...process.argv.slice(2),
	];
	const requestedRuntime = await realpath(process.argv0);
	const runningRuntime = await realpath(process.execPath);
	if (!isDeepStrictEqual(observedInvocation, expectedInvocation) || requestedRuntime !== runningRuntime) {
		throw new Error(`P07B-C ${phase} live chain-gate invocation mismatch: ${JSON.stringify({ argv: observedInvocation, requestedRuntime, runningRuntime })}`);
	}
	await requireReceiptDeclarationPhase(repositoryRoot, c3pReceiptDeclarationPath, receiptPresent, phase);
	const priorCount = expectedArgv.length - 1;
	const show = spawnSync("/opt/homebrew/bin/didrun", ["show", "--session"], {
		cwd: repositoryRoot,
		encoding: "utf8",
		timeout: 30_000,
		maxBuffer: 4 * 1024 * 1024,
		env: {
			HOME: process.env.HOME || "/",
			PATH: "/opt/homebrew/bin:/usr/bin:/bin",
			LANG: "C", LC_ALL: "C", NO_COLOR: "1",
			GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null",
			GIT_NO_LAZY_FETCH: "1", GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
		},
	});
	if (show.error || show.signal || show.status !== 0 || show.stderr !== "" ||
		!show.stdout.startsWith(`session: ${priorCount} events  chain intact\n`)) {
		throw new Error(`P07B-C ${phase} didrun chain audit failed (status=${show.status}, signal=${show.signal}, error=${show.error?.message ?? "none"})`);
	}
	const sessionBytes = await readRegularNoFollow(resolve(repositoryRoot, ".didrun/session.log"), 16 * 1024 * 1024);
	const claimBytes = await readRegularNoFollow(resolve(repositoryRoot, ".didrun/claims.jsonl"), 4 * 1024 * 1024);
	const entries = parseJSONLines(sessionBytes, ".didrun/session.log");
	const recordedClaims = parseJSONLines(claimBytes, ".didrun/claims.jsonl");
	if (entries.length !== priorCount || recordedClaims.length !== priorCount || claims.length !== expectedArgv.length) {
		throw new Error(`P07B-C ${phase} live ledger cardinality mismatch`);
	}
	let tree;
	for (let index = 0; index < priorCount; index += 1) {
		const entry = entries[index];
		const event = entry?.event;
		const expectedPrev = index === 0 ? "0".repeat(64) : entries[index - 1]?.entry_hash;
		if (!entry || !isDeepStrictEqual(Object.keys(entry).sort(), ["entry_hash", "event", "index", "prev_hash"]) ||
			entry.index !== index || entry.prev_hash !== expectedPrev || !/^[0-9a-f]{64}$/u.test(entry.entry_hash ?? "")) {
			throw new Error(`P07B-C ${phase} live ledger entry ${index} malformed`);
		}
		if (!event || event.exit_code !== 0 || event.observed_via !== "wrapper" || event.coverage !== "complete" ||
			event.submodule_dirty !== false || event.tree_before === null || event.tree_before !== event.tree_after ||
			!isDeepStrictEqual(event.argv, expectedArgv[index])) {
			throw new Error(`P07B-C ${phase} live event ${index} authority mismatch`);
		}
		tree ??= event.tree_before;
		if (event.tree_before !== tree) throw new Error(`P07B-C ${phase} live event ${index} tree mismatch`);
		const claim = recordedClaims[index];
		if (!claim || claim.label !== claims[index].label || claim.ctype !== claims[index].type ||
			claim.declared_at_index !== index || !isDeepStrictEqual(claim.event_indices, [index]) ||
			!isDeepStrictEqual(claim.pathspecs, [])) {
			throw new Error(`P07B-C ${phase} live claim ${index} mismatch`);
		}
	}
	let sealsPresent = true;
	try {
		await lstat(resolve(repositoryRoot, ".didrun/seals.jsonl"));
	} catch (error) {
		if (error.code === "ENOENT") sealsPresent = false;
		else throw error;
	}
	if (sealsPresent) throw new Error(`P07B-C ${phase} preseal ledger already contains a seal`);
	const git = spawnSync("/usr/bin/git", ["write-tree"], {
		cwd: repositoryRoot, encoding: "utf8", timeout: 30_000, maxBuffer: 4 * 1024 * 1024,
		env: {
			HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C",
			GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
			GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
		},
	});
	if (git.error || git.signal || git.status !== 0 || git.stderr !== "" || git.stdout.trim() !== tree) {
		throw new Error(`P07B-C ${phase} staged tree does not equal the live ledger tree`);
	}
	console.log(`P07B-C ${phase} preceding chain exact: events=${priorCount} claims=${priorCount} green=${priorCount} tree=${tree} chain=valid seal=absent`);
}

export async function verifyC3PPresealLedger() {
	await verifyLivePresealLedger({
		phase: "C3P",
		expectedArgv: c3pExpectedClaimArgv,
		claims: c3pClaimLabels.map((label, index) => ({ label, type: c3pClaimTypes[index] })),
		receiptPresent: false,
	});
}

export async function verifyC3PBPresealLedger() {
	await verifyLivePresealLedger({
		phase: "C3PB",
		expectedArgv: c3pbExpectedClaimArgv,
		claims: requiredC3PBReceiptClaims,
		receiptPresent: true,
	});
}

export async function verifyC3PLocalEvidence() {
	const receiptText = await readReceiptDeclarationSnapshot(repositoryRoot, c3pReceiptDeclarationPath, "C3P");
	const planErrors = await checkPlan(repositoryRoot, new Map([[c3pReceiptDeclarationPath, receiptText]]));
	if (planErrors.length > 0) throw new Error(`P07B-C C3P local-evidence precondition failed:\n${planErrors.join("\n")}`);
	const receipt = JSON.parse(receiptText);
	const evidenceErrors = await verifyLocalEvidence(repositoryRoot, receipt, c3pClaimLabels, c3pClaimTypes, c3pExpectedClaimArgv);
	if (evidenceErrors.length > 0) throw new Error(`P07B-C C3P local-evidence verification failed:\n${evidenceErrors.join("\n")}`);
	console.log("P07B-C C3P local evidence passed: HTML plus the ignored sealed-source ledger snapshot match the closed C3P receipt declaration; this local snapshot is not portable strict authority");
}

async function main() {
	const mode = process.argv[2];
	const usage = "usage: check-p07b-c-plan.mjs [--self-test|--verify-sealed-c1-local-evidence|--verify-c1-local-evidence|--verify-c2-local-evidence|--verify-c2m-preseal-ledger|--verify-c2-preseal-ledger]";
	if (mode === "--self-test") {
		if (process.argv.length !== 3) throw new Error(usage);
		await runSelfTest();
		return;
	}
	if (mode === "--verify-sealed-c1-local-evidence") {
		if (process.argv.length !== 3) throw new Error(usage);
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
		if (process.argv.length !== 3) throw new Error(usage);
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
	if (mode === "--verify-c2-local-evidence") {
		if (process.argv.length !== 3) throw new Error(usage);
		const receiptText = await readReceiptDeclarationSnapshot(repositoryRoot, c2ReceiptDeclarationPath, "C2");
		const planErrors = await checkPlan(repositoryRoot, new Map([[c2ReceiptDeclarationPath, receiptText]]));
		if (planErrors.length > 0) {
			throw new Error(`P07B-C C2 local-evidence precondition failed:\n${planErrors.join("\n")}`);
		}
		const receipt = JSON.parse(receiptText);
		const evidenceErrors = await verifyLocalEvidence(repositoryRoot, receipt, c2ClaimLabels, c2ClaimTypes, c2ExpectedClaimArgv);
		if (evidenceErrors.length > 0) {
			throw new Error(`P07B-C C2 local-evidence verification failed:\n${evidenceErrors.join("\n")}`);
		}
		console.log("P07B-C C2 local evidence passed: HTML plus the ignored sealed-source ledger snapshot match the closed C2 receipt declaration; this local snapshot is not portable strict authority");
		return;
	}
	if (mode === "--verify-c2m-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC2MaintenancePresealLedger();
		return;
	}
	if (mode === "--verify-c2-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC2PresealLedger();
		return;
	}
	if (mode !== undefined) throw new Error(usage);

	const errors = await checkPlan();
	if (errors.length > 0) {
		console.error("P07B-C C1 evolved plan check failed:");
		for (const error of errors) console.error(`- ${error}`);
		process.exitCode = 1;
		return;
	}

	console.log("P07B-C C3P plan check passed: historical receipts, corrected boot-session model/schema/examples, exact C3P/C3 receipt order, and exact source rosters remain coherent; this gate confers no live host, runtime, Git, target, or process authority");
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
	try {
		await main();
	} catch (error) {
		console.error(error.message);
		process.exitCode = 1;
	}
}
