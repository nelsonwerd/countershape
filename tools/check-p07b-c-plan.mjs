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

import {
	credentialPatternFindings,
	exactPathsMatch,
	gitOutput,
	stagedBlobEntries,
	stagedIndexEntries,
	stagedPaths,
	validateReceiptIndexModes,
	validateSpecification,
} from "./check-p07b-c-unit-scope.mjs";
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
const c3ReceiptDeclarationPath = "spec/verification/p07b-c-c3-receipt.json";
const statusPath = "docs/status/P07B-C-C0-AUTHORITY.md";
const c1MaintenanceStatusPath = "docs/status/P07B-C-C1-CUMULATIVE-MAINTENANCE.md";
const c1VerificationStatusPath = "docs/status/P07B-C-VERIFICATION-THROUGHPUT.md";
const c1EvidenceMaintenanceStatusPath = "docs/status/P07B-C-C1-LOCAL-EVIDENCE-MAINTENANCE.md";
const c1StatusPath = "docs/status/P07B-C-C1-SEMANTICS.md";
const c2StatusPath = "docs/status/P07B-C-C2-PERSISTENCE.md";
const c2MaintenanceStatusPath = "docs/status/P07B-C-C2-RECEIPT-PHASE-MAINTENANCE.md";
const c3pStatusPath = "docs/status/P07B-C-C3P-RUNTIME-EPOCH.md";
const c3vStatusPath = "docs/status/P07B-C-C3V-VERIFICATION-THROUGHPUT.md";
const c3MaintenanceStatusPath = "docs/status/P07B-C-C3P-RECEIPT-PHASE-MAINTENANCE.md";
const c3aStatusPath = "docs/status/P07B-C-C3A-CUMULATIVE-ADMISSION-MAINTENANCE.md";
const c3lStatusPath = "docs/status/P07B-C-C3L-LEGACY-LATTICE-MAINTENANCE.md";
const c3fStatusPath = "docs/status/P07B-C-C3F-B-FUTURE-SURFACE-ADMISSION.md";
const c3sStatusPath = "docs/status/P07B-C-C3S-CUMULATIVE-SELFTEST-MAINTENANCE.md";
const c3StatusPath = "docs/status/P07B-C-C3-TARGET.md";
const c3rStatusPath = "docs/status/P07B-C-C3R-NOTE-PREVIEW-MAINTENANCE.md";
const c3qStatusPath = "docs/status/P07B-C-C3Q-SELF-RECEIPT-MAINTENANCE.md";
const c3tStatusPath = "docs/status/P07B-C-C3T-PHASE-SELFTEST-MAINTENANCE.md";
const c3uStatusPath = "docs/status/P07B-C-C3U-CROSS-PHASE-RECEIPT-FIXTURE-MAINTENANCE.md";
const c3pReceiptBlockStart = "<!-- P07B-C-C3P-SOURCE-RECEIPTS:START -->";
const c3pReceiptBlockEnd = "<!-- P07B-C-C3P-SOURCE-RECEIPTS:END -->";
const c3ReceiptBlockStart = "<!-- P07B-C-C3-SOURCE-RECEIPTS:START -->";
const c3ReceiptBlockEnd = "<!-- P07B-C-C3-SOURCE-RECEIPTS:END -->";
const c3pbActiveHandoffState = "C3PB receipt reconciliation is the active exact three-path `RECEIPT_RECONCILIATION` unit and remains `UNRECEIPTED`; C3M is treated as a closed interposed prerequisite in this receipt-present phase. C3 stays blocked until C3PB independently closes.";
const c3pReceiptNoRecursionBoundary = "This receipt binds only the already-existing C3P source commit. The independently sealed C3PB descendant is outside this C3P source grade map, and the active C3A unit cannot grade itself here.";
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
const sealedC3PIdentity = Object.freeze({
	commit: "f7b6e6bda7a8864969415ab8636c495902e78dd9",
	tree: "2d5555db63d46c837cf4e12f2dab2d04a41f4654",
});
const sealedC3VIdentity = Object.freeze({
	commit: "725933fa3c7826a281e84b51f729736cbcfac6ec",
	tree: "21f0e1bf46e3776e1b496a9c2783f549923c4510",
});
const sealedC3MIdentity = Object.freeze({
	commit: "c211d0534864e078fd0ea2c15adabca63ca3fe51",
	tree: "21cc76ed3979d1b6ed6ee24bd9babd9dbc1c6791",
});
const sealedC3PBIdentity = Object.freeze({
	commit: "13369122ba7d5557eba1949095c1135a41843070",
	tree: "d710d9b249786f3fb648f566ae542f1d9ddec180",
	noteBlob: "29c2d16db73d82c970b2eba6a838ec400a6f0139",
	noteBodySHA256: "fa8cbe6a18e83f72343948dba511a08033d2bbba749696404ec450304a6094b1",
});
const sealedC3AIdentity = Object.freeze({
	commit: "2b84d2841971568784d2ac955775b4a99ca7f0f6",
	tree: "b8d858033431fe51c262ce725b9d9b051ba9e8c9",
	noteBlob: "3e516946f4b86e536ae9ea0124938bd534573983",
	noteBodySHA256: "07a0d04dc667963ea16ec6bb458b929d3a7659637e705bb642b5267766609a8c",
});
const sealedC3LIdentity = Object.freeze({
	commit: "efa918928246c7793f4ad7201c003020e3a42193",
	tree: "d9187c6449392c269a385b53c563ba4411112460",
	parent: sealedC3AIdentity.commit,
	subject: "fix: admit exact C3 store model edge through U6",
	noteRef: "refs/notes/didrun",
	noteType: "blob",
	noteBlob: "b21365e156719fbc84e480108a7e8672644f5a71",
	noteBodySHA256: "d4721b344adb95442c5273e596c83ea9cb1b855d26655198a9803771f83e2e9d",
});
const sealedC3FIdentity = Object.freeze({
	commit: "63038644ba347d7b934a5557a490af95bb4428a4",
	tree: "3b4c39cfa907c2907d01dcd21399cafe52075872",
	parent: sealedC3LIdentity.commit,
	subject: "fix: admit exact C3 target codec surface through B",
	noteRef: "refs/notes/didrun",
	noteType: "blob",
	noteBlob: "098958ba7a9b2d9d237cee901a34279424b68128",
	noteBodySHA256: "a23982f84573505fc966378dbcc4f825daf11da8a08b2bfd24f7f71de588d315",
});
const sealedC3SIdentity = Object.freeze({
	commit: "47a65b45a50c0f39fc39e1336fc7744a97a72b8f",
	tree: "fce8593ad1fe7af1474b4e113f03e82a9a052f79",
	parent: sealedC3FIdentity.commit,
	subject: "fix: make U6 C3 fixture phase-stable",
	noteRef: "refs/notes/didrun",
	noteType: "blob",
	noteBlob: "7dff66800e79dc5b1a149ffe16df4dbf9178836a",
	noteBodySHA256: "b65f71e25dd53509f8df876a28fb4027559464f77861a34d1aa0c5854bfdbae6",
});
const sealedC3Identity = Object.freeze({
	commit: "029c5cf43853eb3cb46520f6e0dea8431a3de5c0",
	tree: "de3e02a343f8ecfb344bdb28d014342f5b423959",
	parent: sealedC3SIdentity.commit,
	subject: "feat: publish exact contract execution targets",
	noteRef: "refs/notes/didrun",
	noteType: "blob",
	noteBlob: "9ad8a864911ca30914463c61d3447a88beeda927",
	noteBodySHA256: "f5eaa66d0cae152a9edc7f4ff2d49ba66c0d1eb8f3459171a2888daaaaf80fa8",
});
const sealedC3RIdentity = Object.freeze({
	commit: "424273b000f87925a87f0b3041b2e1e8c5750d39",
	tree: "0e4694362d0242a3b3da0ed2f8f78971ee595f6c",
	parent: sealedC3Identity.commit,
	subject: "fix: validate sealed C3 note previews",
	noteRef: "refs/notes/didrun",
	noteType: "blob",
	noteBlob: "7d0c90aaec4f1a36d8603e5c1ff6195a02221314",
	noteBodySHA256: "dcd8991ef2446da78137319d7069d89999eda9748338fbdb5f8927bb4460e09d",
});
const sealedC3QIdentity = Object.freeze({
	commit: "818ed90ebbf99ae38916307d9f2342921d38ebbf",
	tree: "157241ac2ea7c390b2f67a788b26b9f7666b2893",
	parent: sealedC3RIdentity.commit,
	subject: "fix: close C3B external self-receipt checks",
	noteRef: "refs/notes/didrun",
	noteType: "blob",
	noteBlob: "06cd81c05ce2e6f2348d0765298846ac048caf7e",
	noteBodySHA256: "b61f53351614361f3646186827ed96e7aace8980bcd3be52f470a366a59fe855",
});
const sealedC3TIdentity = Object.freeze({
	commit: "9d8430aa70b1536a85dd7352a3d339667f2c1e88",
	tree: "3be4d7cde4d043b87438506483ffb8c256719df2",
	parent: sealedC3QIdentity.commit,
	subject: "fix: make C3 receipt self-test phase-stable",
	noteRef: "refs/notes/didrun",
	noteType: "blob",
	noteBlob: "d9e371310b360b183d890ed401cb36bd877becda",
	noteBodySHA256: "05914bf85ce1d3bc48f5debf902babb526faf2500af318c35f08ffdf071ff735",
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
		"C1/C2/C3P grades enter tracked authority only through their separate receipt descendants",
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
		"through C3S sealed; C3 exact-target publication active",
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
		"C3 exact-target publication active",
	],
	"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md": [
		"C0 → C1 → C1B → C2 → C2B → C3P → C3PB → C3 → C3B → C4 → C5 → C6",
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
		"countershape/p07b-c-unit-paths/v15",
		"\"C0A\"",
		"\"C1M\"",
		"\"C1V\"",
		"\"C1E\"",
		"\"C2M\"",
		"\"C3V\"",
		"\"C3M\"",
		"\"C3L\"",
		"\"C3F\"",
		"\"C3S\"",
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
		"## Historical mixed didrun object namespace for C1 local evidence",
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
			"countershape/p07b-c-unit-paths/v15",
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
		"## Historical C2 receipt-phase checker maintenance",
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
const requiredC3VPaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/VERIFICATION.md",
	"docs/status/P07B-C-C3V-VERIFICATION-THROUGHPUT.md",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
]);
const requiredC3VDigest = "sha256:6cc5d8fe929f0f1dfa6ffc223d2e8bcee5ba4596a09caa9dc032103bbb5bafbd";
const requiredC3VRosterText = requiredC3VPaths
	.map((path, index) => `${index === requiredC3VPaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC3VDeclaration = `C3V owns exactly these ${requiredC3VPaths.length} paths: ${requiredC3VRosterText}. Their sorted-newline roster digest is \`${requiredC3VDigest}\`.`;
const c3vClaimLabels = Object.freeze([
	"P07B-C C3V throughput proposal disposition coherence",
	"P07B-C C3V disposition and frozen-verifier defensive self-test",
	"P07B-C C3V unit-scope defensive self-test",
	"P07B-C C3V cumulative verifier self-test",
	"P07B-C C3V cumulative verification",
	"P07B-C C3V exact seven-path staged scope and diff integrity",
	"P07B-C C3V scoped staged credential-pattern scan",
	"P07B-C C3V preceding didrun chain integrity",
]);
const c3vClaimTypes = Object.freeze([
	...Array(5).fill("tests-pass"),
	...Array(3).fill("command-succeeded"),
]);
const c3vExpectedClaimArgv = Object.freeze([
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3V", "--source-final-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3V", "--credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3v-preseal-ledger"]),
]);
const c3vFrozenVerifierDigests = Object.freeze({
	"tools/verify-current.mjs": "ce76506d67c8d79b6359f5825dcc85abbc60b87344c8fb4884335faa03babbc3",
	"tools/verify-current-selftest.mjs": "b18771c16cb85b5a85fdadb0ee0aa39d7357217e302ea3b0b690ba56cfc2d558",
	"tools/verify-go-test-repetition.mjs": "1321fb1381f1f74d0959b9ceb40b4dcc4abc637bb281ff2af759863ede3bfd5a",
});
const requiredC3MaintenancePaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/VERIFICATION.md",
	"docs/status/P07B-C-C3P-RECEIPT-PHASE-MAINTENANCE.md",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
]);
const requiredC3MaintenanceDigest = "sha256:58e1f6fba95cf3b4312136eea5bb5c64af89a6958711aca84181a1874aa03240";
const requiredC3MaintenanceRosterText = requiredC3MaintenancePaths
	.map((path, index) => `${index === requiredC3MaintenancePaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC3MaintenanceDeclaration = `C3M owns exactly these ${requiredC3MaintenancePaths.length} paths: ${requiredC3MaintenanceRosterText}. Their sorted-newline roster digest is \`${requiredC3MaintenanceDigest}\`.`;
const c3MaintenanceClaimLabels = Object.freeze([
	"P07B-C C3M receipt-phase plan coherence",
	"P07B-C C3M pre-receipt and receipt-phase checker self-test",
	"P07B-C C3M unit-scope defensive self-test",
	"P07B-C C3M cumulative verifier self-test",
	"P07B-C C3M cumulative verification",
	"P07B-C C3M exact seven-path staged scope and diff integrity",
	"P07B-C C3M scoped staged credential-pattern scan",
	"P07B-C C3M preceding didrun chain integrity",
]);
const c3MaintenanceClaimTypes = Object.freeze([
	...Array(5).fill("tests-pass"),
	...Array(3).fill("command-succeeded"),
]);
const c3MaintenanceExpectedClaimArgv = Object.freeze([
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3M", "--source-final-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3M", "--credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3m-preseal-ledger"]),
]);
const requiredC3PBPaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/status/P07B-C-C3P-RUNTIME-EPOCH.md",
	"spec/verification/p07b-c-c3p-receipt.json",
]);
const requiredC3PBDigest = "sha256:531cbe600cf2747752749890ea9a15c3d0126e498da6838d4f6faeb1a0d7e885";
const requiredC3APaths = Object.freeze([
	"docs/ARCHITECTURE.md",
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/VERIFICATION.md",
	"docs/status/P07B-C-C3A-CUMULATIVE-ADMISSION-MAINTENANCE.md",
	"docs/status/P07B-C-C3P-RUNTIME-EPOCH.md",
	"internal/contractexec/model/codec_test.go",
	"internal/contractexec/model/target.go",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-b-architecture-selftest.mjs",
	"tools/check-p07b-b-architecture.mjs",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
]);
const requiredC3ADigest = "sha256:cc9bb98ceeb2f281336d3e235377d3cdb8920664f833baea830a84b847621188";
const requiredC3ARosterText = requiredC3APaths
	.map((path, index) => `${index === requiredC3APaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC3ADeclaration = `C3A owns exactly these ${requiredC3APaths.length} paths: ${requiredC3ARosterText}. Their sorted-newline roster digest is \`${requiredC3ADigest}\`.`;
const c3aActiveHandoffState = `C3A cumulative-admission maintenance is the active exact thirteen-path \`SOURCE_FULL\` unit and remains \`UNRECEIPTED\`; sealed C3PB commit \`${sealedC3PBIdentity.commit}\`, tree \`${sealedC3PBIdentity.tree}\`, is the closed receipt prerequisite. C3 stays blocked until C3A independently closes.`;
const requiredC3LPaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/VERIFICATION.md",
	"docs/status/P07B-C-C3L-LEGACY-LATTICE-MAINTENANCE.md",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
	"tools/check-u6-architecture-selftest.mjs",
	"tools/check-u6-architecture.mjs",
]);
const requiredC3LDigest = "sha256:b3624e1337681e0909e6c074246103e066118a5924e8f790e7d1f42359da9dc0";
const requiredC3LRosterText = requiredC3LPaths
	.map((path, index) => `${index === requiredC3LPaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC3LDeclaration = `C3L owns exactly these ${requiredC3LPaths.length} paths: ${requiredC3LRosterText}. Their sorted-newline roster digest is \`${requiredC3LDigest}\`.`;
const c3lActiveHandoffState = `C3L legacy-lattice maintenance is the active exact nine-path \`SOURCE_FULL\` unit and remains \`UNRECEIPTED\`; sealed C3A commit \`${sealedC3AIdentity.commit}\`, tree \`${sealedC3AIdentity.tree}\`, is the closed cumulative-admission prerequisite. C3 stays blocked until C3L independently closes.`;
const c3lStatusBoundaryLine = "- **Boundary:** active pre-seal `SOURCE_FULL` maintenance after sealed C3A and before the frozen C3 source unit. Classification: `DEFECT_REPAIR`.";
const c3lStatusParentLine = `- **Parent:** sealed C3A commit \`${sealedC3AIdentity.commit}\`, tree \`${sealedC3AIdentity.tree}\`, is note-present with note blob \`${sealedC3AIdentity.noteBlob}\`, note-body SHA-256 \`${sealedC3AIdentity.noteBodySHA256}\`, \`11/11 claims recorded-exact\`, and strict exit \`0\`.`;
const c3lStatusUnreceiptedLine = "Every intended C3L grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3L itself.";
const requiredC3FPaths = Object.freeze([
	"docs/ARCHITECTURE.md",
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/VERIFICATION.md",
	"docs/status/P07B-C-C3F-B-FUTURE-SURFACE-ADMISSION.md",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-b-architecture-selftest.mjs",
	"tools/check-p07b-b-architecture.mjs",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
]);
const requiredC3FDigest = "sha256:f64293895a1b824faafddf236757d381c813b59c40286225bb1c352d5c812c1a";
const requiredC3FRosterText = requiredC3FPaths
	.map((path, index) => `${index === requiredC3FPaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC3FDeclaration = `C3F owns exactly these ${requiredC3FPaths.length} paths: ${requiredC3FRosterText}. Their sorted-newline roster digest is \`${requiredC3FDigest}\`.`;
const c3fActiveHandoffState = `C3F B future-surface admission maintenance is the active exact ten-path \`SOURCE_FULL\` unit and remains \`UNRECEIPTED\`; sealed C3L commit \`${sealedC3LIdentity.commit}\`, tree \`${sealedC3LIdentity.tree}\`, is the closed legacy-lattice prerequisite. C3 stays blocked until C3F independently closes.`;
const c3fStatusBoundaryLine = "- **Boundary:** active pre-seal `SOURCE_FULL` maintenance after sealed C3L and before the unchanged frozen C3 source unit. Classification: `DEFECT_REPAIR`.";
const c3fStatusParentLine = `- **Parent:** sealed C3L commit \`${sealedC3LIdentity.commit}\`, tree \`${sealedC3LIdentity.tree}\`, is note-present with note blob \`${sealedC3LIdentity.noteBlob}\`, note-body SHA-256 \`${sealedC3LIdentity.noteBodySHA256}\`, \`12/12 claims recorded-exact\`, strict exit \`0\`, and a logged redacted-export override.`;
const c3fStatusUnreceiptedLine = "Every intended C3F grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3F itself.";
const requiredC3SPaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/VERIFICATION.md",
	"docs/status/P07B-C-C3S-CUMULATIVE-SELFTEST-MAINTENANCE.md",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
	"tools/check-u6-architecture-selftest.mjs",
]);
const requiredC3SDigest = "sha256:1d5f793e70caea0ad7379e92bbf781da50846276c28a2aca2d3e7e2ed8c17a26";
const requiredC3SRosterText = requiredC3SPaths
	.map((path, index) => `${index === requiredC3SPaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC3SDeclaration = `C3S owns exactly these ${requiredC3SPaths.length} paths: ${requiredC3SRosterText}. Their sorted-newline roster digest is \`${requiredC3SDigest}\`.`;
const c3sActiveHandoffState = `C3S cumulative U6 self-test maintenance is the active exact eight-path \`SOURCE_FULL\` unit and remains \`UNRECEIPTED\`; sealed C3F commit \`${sealedC3FIdentity.commit}\`, tree \`${sealedC3FIdentity.tree}\`, is the closed future-surface prerequisite. C3 stays blocked until C3S independently closes.`;
const c3sStatusBoundaryLine = "- **Boundary:** active pre-seal `SOURCE_FULL` maintenance after sealed C3F and before the unchanged frozen C3 source unit. Classification: `DEFECT_REPAIR`.";
const c3sStatusParentLine = `- **Parent:** sealed C3F commit \`${sealedC3FIdentity.commit}\`, tree \`${sealedC3FIdentity.tree}\`, is note-present with note blob \`${sealedC3FIdentity.noteBlob}\`, note-body SHA-256 \`${sealedC3FIdentity.noteBodySHA256}\`, \`10/10 claims recorded-exact\`, strict exit \`0\`, and a logged redacted-export override.`;
const c3sStatusUnreceiptedLine = "Every intended C3S grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3S itself.";
const c3ActiveHandoffState = `C3 exact-target publication is the active exact forty-path \`SOURCE_FULL\` unit and remains \`UNRECEIPTED\`; sealed C3S commit \`${sealedC3SIdentity.commit}\`, tree \`${sealedC3SIdentity.tree}\`, is the direct source parent. C3B stays blocked until C3 independently closes.`;
const c3rActiveHandoffState = `C3R note-preview checker maintenance is the active exact seven-path \`SOURCE_FULL\` unit and remains \`UNRECEIPTED\`; sealed C3 commit \`${sealedC3Identity.commit}\`, tree \`${sealedC3Identity.tree}\`, is the closed source boundary. C3B stays blocked until C3R independently closes.`;
const c3qActiveHandoffState = `C3Q self-receipt checker maintenance is the active exact seven-path \`SOURCE_FULL\` unit and remains \`UNRECEIPTED\`; sealed C3R commit \`${sealedC3RIdentity.commit}\`, tree \`${sealedC3RIdentity.tree}\`, is the closed checker prerequisite. C3B stays blocked until C3Q independently closes.`;
const c3tActiveHandoffState = `C3T phase-transition self-test maintenance is the active exact seven-path \`SOURCE_FULL\` unit and remains \`UNRECEIPTED\`; sealed C3Q commit \`${sealedC3QIdentity.commit}\`, tree \`${sealedC3QIdentity.tree}\`, is the closed checker prerequisite. C3B stays blocked until C3T independently closes.`;
const c3uActiveHandoffState = `C3U cross-phase receipt-fixture maintenance is the active exact seven-path \`SOURCE_FULL\` unit and remains \`UNRECEIPTED\`; sealed C3T commit \`${sealedC3TIdentity.commit}\`, tree \`${sealedC3TIdentity.tree}\`, is the closed checker prerequisite. C3B stays blocked until C3U independently closes.`;
const c3bActiveHandoffState = (receipt) => `C3B receipt reconciliation is the active exact three-path \`RECEIPT_RECONCILIATION\` unit and remains \`UNRECEIPTED\`; sealed C3 source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`, is the closed source boundary, and sealed C3U is the immediate checker prerequisite. C4 stays blocked until C3B independently closes.`;
const c3pbHandoffTransitions = Object.freeze([
	Object.freeze({
		name: "active state",
		before: "C3M receipt-phase checker maintenance is the active `SOURCE_FULL` unit; every C3M grade remains `UNRECEIPTED`. C3PB and C3 stay blocked until C3M closes and the later exact three-path C3PB receipt reconciliation independently closes.",
		after: `${c3pbActiveHandoffState} Historical compatibility anchor: the sentence “C3M receipt-phase checker maintenance is the active \`SOURCE_FULL\` unit” is retained as inert phase history.`,
		descendant: `${c3aActiveHandoffState} Historical compatibility anchor: the sentence “C3M receipt-phase checker maintenance is the active \`SOURCE_FULL\` unit” is retained as inert phase history.`,
		c3l: c3lActiveHandoffState,
		c3f: c3fActiveHandoffState,
		c3s: c3sActiveHandoffState,
		c3: c3ActiveHandoffState,
		c3r: c3rActiveHandoffState,
		c3q: c3qActiveHandoffState,
		c3t: c3tActiveHandoffState,
		c3u: c3uActiveHandoffState,
		c3b: c3bActiveHandoffState,
	}),
	Object.freeze({
		name: "dirty scope",
		before: `The dirty tree is limited to the active C3M checker/documentation declaration and must converge to its exact seven-path roster digest \`${requiredC3MaintenanceDigest}\`.`,
		after: `The receipt-present dirty tree is limited to C3PB's exact three-path roster at \`${requiredC3PBDigest}\`; the closed interposed C3M roster remains \`${requiredC3MaintenanceDigest}\`.`,
		descendant: `The dirty tree is limited to C3A's exact thirteen-path roster at \`${requiredC3ADigest}\`; sealed C3PB retains its exact three-path roster at \`${requiredC3PBDigest}\`, and the closed interposed C3M roster remains \`${requiredC3MaintenanceDigest}\`.`,
		c3l: `The dirty tree is limited to C3L's exact nine-path roster at \`${requiredC3LDigest}\`; sealed C3A retains its exact thirteen-path roster at \`${requiredC3ADigest}\`, and C3 remains frozen at \`sha256:c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb\`.`,
		c3f: `The dirty tree is limited to C3F's exact ten-path roster at \`${requiredC3FDigest}\`; sealed C3L retains its exact nine-path roster at \`${requiredC3LDigest}\`, and C3 remains frozen at \`sha256:c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb\`.`,
		c3s: `The dirty tree is limited to C3S's exact eight-path roster at \`${requiredC3SDigest}\`; sealed C3F retains its exact ten-path roster at \`${requiredC3FDigest}\`, and C3 remains frozen at \`sha256:c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb\`.`,
		c3: "The dirty tree is limited to C3's exact forty-path roster at `sha256:c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb`; sealed C3S retains its exact eight-path roster at `sha256:1d5f793e70caea0ad7379e92bbf781da50846276c28a2aca2d3e7e2ed8c17a26`, and C3B remains frozen at `sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67`.",
		c3r: "The dirty tree is limited to C3R's exact seven-path roster at `sha256:fe14d9f9985771cce0525edfca47c001b5912dacc4c2d6fef182458d3b44f58e`; the sealed C3 source roster remains `sha256:c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb`, and C3B remains frozen at `sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67`.",
		c3q: "The dirty tree is limited to C3Q's exact seven-path roster at `sha256:f5f27e661c1ace6516aba4b9d5766f4c6ab50b6602dbcb101bff9b6a4ebfe6fc`; sealed C3R retains its exact seven-path roster at `sha256:fe14d9f9985771cce0525edfca47c001b5912dacc4c2d6fef182458d3b44f58e`, and C3B remains frozen at `sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67`.",
		c3t: "The dirty tree is limited to C3T's exact seven-path roster at `sha256:496788860db6c42f5acc50de0de02be8a63bcb8e11f99fb24d59d9c0e016477e`; sealed C3Q retains its exact seven-path roster at `sha256:f5f27e661c1ace6516aba4b9d5766f4c6ab50b6602dbcb101bff9b6a4ebfe6fc`, and C3B remains frozen at `sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67`.",
		c3u: "The dirty tree is limited to C3U's exact seven-path roster at `sha256:7af7456929b0db08abc2a31bd1cb62d84002c1a6bcdda3f4a8b4896f6b8c30cd`; sealed C3T retains its exact seven-path roster at `sha256:496788860db6c42f5acc50de0de02be8a63bcb8e11f99fb24d59d9c0e016477e`, and C3B remains frozen at `sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67`.",
		c3b: "The receipt-only dirty tree is limited to C3B's exact three-path roster at `sha256:3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67`; the sealed C3 source roster remains `sha256:c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb`.",
	}),
	Object.freeze({
		name: "ledger profile",
		before: "C3M must use a zero-claim development ledger and a separate fresh eight-claim final ledger; no C3V event is reused.",
		after: "C3PB uses a separate fresh seven-claim receipt ledger; no C3M event is reused.",
		descendant: "C3A uses a zero-claim development ledger and a separate fresh eleven-claim final ledger; no C3PB event is reused.",
		c3l: "C3L uses a zero-claim development ledger and a separate fresh twelve-claim final ledger; no C3A event is reused.",
		c3f: "C3F uses a zero-claim development ledger and a separate fresh ten-claim final ledger; no C3L event is reused.",
		c3s: "C3S uses a zero-claim development ledger and a separate fresh ten-claim final ledger; no C3F event is reused.",
		c3: "C3 uses a zero-claim development ledger and a separate fresh eighteen-claim final ledger; no C3S event is reused.",
		c3r: "C3R uses a zero-claim development ledger and a separate fresh nine-claim final ledger; no C3 event is reused.",
		c3q: "C3Q uses a zero-claim development ledger and a separate fresh nine-claim final ledger; no C3R event is reused.",
		c3t: "C3T uses a zero-claim development ledger and a separate fresh nine-claim final ledger; no C3Q event is reused.",
		c3u: "C3U uses a zero-claim development ledger and a separate fresh nine-claim final ledger; no C3T event is reused.",
		c3b: "C3B uses a separate fresh seven-claim receipt ledger; no C3U event is reused.",
	}),
	Object.freeze({
		name: "verification battery",
		before: "C3M keeps the verifier implementation byte-for-byte unchanged, repairs the plan/scope checker authority, and must run the complete cumulative profile plus planning/scope self-tests through didrun. A direct pass has no didrun grade. The later C3PB unit remains deliberately narrower and may bind only the already-sealed C3P source evidence after C3M independently seals.",
		after: "C3PB keeps the verifier implementation byte-for-byte unchanged and runs only its explicit narrow receipt gates through didrun. A direct pass has no didrun grade. This receipt-present phase may bind only the already-sealed C3P source evidence after the interposed C3M boundary.",
		descendant: "C3A repairs the inherited B architecture admission and the canonical runtime-version bound, then runs its direct model/architecture gates plus the full cumulative verifier through didrun. A direct pass has no didrun grade. The sealed C3P receipt block remains immutable source evidence.",
		c3l: "C3L conditionally admits exactly the future C3 store-model edge through U6, preserves the old lattice when that bridge is absent, and then runs U6, B, their hostile self-tests, and the full cumulative verifier through didrun. A direct pass has no didrun grade. The sealed C3P receipt block remains immutable source evidence.",
		c3f: "C3F conditionally admits only the complete exact four-pair future C3 target-codec surface through B, preserves the historical zero-member state, and then runs B, its hostile self-test, and the full cumulative verifier through didrun. A direct pass has no didrun grade. The sealed C3P receipt block remains immutable source evidence.",
		c3s: "C3S makes U6's C3 bridge fixtures phase-stable without changing production U6 policy, runs historical U6, its 147-case hostile self-test, and the full cumulative verifier through didrun. A direct pass has no didrun grade. The sealed C3P receipt block remains immutable source evidence.",
		c3: "C3 publishes one exact inert pre-spawn target, exercises its five exact authority profiles plus the focused race suite, and then runs B compatibility and the full cumulative verifier through didrun. A direct pass has no didrun grade. OfficialTarget.Valid is structural only; C4 must freshly reopen immediately before process start.",
		c3r: "C3R admits only one exact didrun note-preview projection for C3 claim eight while preserving byte-exact raw-ledger argv authority, then runs plan/scope self-tests and the full cumulative verifier through didrun. A direct pass has no didrun grade. C3R changes no product, verifier, didrun install, sealed C3 note, or frozen C3B scope.",
		c3q: "C3Q closes active-unit self-receipt rejection across the exact plan-authority Markdown corpus in both C3Q-active and C3B-active phases, then runs plan/scope self-tests and the full cumulative verifier through didrun. A direct pass has no didrun grade. C3Q changes no product, verifier roster, didrun install, sealed C3/C3R evidence, or frozen C3B scope.",
		c3t: "C3T reconstructs one validated receipt-absent C3T fixture before every hostile receipt mutation, proves absent-to-present and present-to-absent-to-present byte stability, and then runs plan/scope self-tests plus the full cumulative verifier through didrun. A direct pass has no didrun grade. C3T changes no product, verifier roster, didrun install, sealed C3/C3R/C3Q evidence, or frozen C3B scope.",
		c3u: "C3U inserts every synthetic legacy receipt block before an optional exact terminal C3 block, preserves that C3 authority byte-for-byte, and runs the plan/scope self-tests plus the full cumulative verifier through didrun. A direct pass has no didrun grade. C3U changes no product, verifier roster, didrun install, sealed C3/C3R/C3Q/C3T evidence, or frozen C3B scope.",
		c3b: "C3B keeps source and verifier code byte-for-byte unchanged and runs only its explicit narrow receipt gates through didrun. It binds the already-sealed C3 source evidence and cannot claim cumulative execution, runtime freshness, or C4 authority.",
	}),
	Object.freeze({
		name: "maintenance heading",
		before: "### Active C3M receipt-phase checker maintenance",
		after: "### Closed interposed C3M boundary — synthetic C3PB receipt phase",
		descendant: "### Active C3A cumulative-admission maintenance",
		c3l: "### Active C3L legacy-lattice maintenance",
		c3f: "### Active C3F B future-surface admission maintenance",
		c3s: "### Active C3S cumulative U6 self-test maintenance",
		c3: "### Active C3 exact-target publication",
		c3r: "### Active C3R note-preview checker maintenance",
		c3q: "### Active C3Q self-receipt checker maintenance",
		c3t: "### Active C3T phase-transition self-test maintenance",
		c3u: "### Active C3U cross-phase receipt-fixture maintenance",
		c3b: "### Active C3B source receipt reconciliation",
	}),
	Object.freeze({
		name: "receipt presence",
		before: "No C3P receipt declaration or C3P source-receipt block is live in C3M. C3PB's exact three-path digest, seven labels/types, and narrow profile remain unchanged.",
		after: "C3M's live tree contained no C3P receipt declaration or source-receipt block. This receipt-present phase contains exactly one closed declaration and block; C3PB's exact three-path digest, seven labels/types, and narrow profile remain unchanged.",
		descendant: "The C3P receipt declaration and one exact source-receipt block remain closed source evidence. C3PB is independently sealed; C3A changes neither receipt payload nor any C3P/C3PB grade.",
		c3l: "The C3P receipt declaration and one exact source-receipt block remain closed source evidence. C3A is independently sealed; C3L changes neither receipt payload nor any C3P/C3PB/C3A grade.",
		c3f: "The C3P receipt declaration and one exact source-receipt block remain closed source evidence. C3L is independently sealed; C3F changes neither receipt payload nor any C3P/C3PB/C3A/C3L grade.",
		c3s: "The C3P receipt declaration and one exact source-receipt block remain closed source evidence. C3F is independently sealed; C3S changes neither receipt payload nor any C3P/C3PB/C3A/C3L/C3F grade.",
		c3: "The C3P receipt declaration and one exact source-receipt block remain closed source evidence. C3F and C3S are independently sealed; active C3 changes neither payload nor any predecessor grade, and no C3 source receipt exists yet.",
		c3r: "The C3P receipt declaration and one exact source-receipt block remain closed source evidence. C3 is independently sealed; active C3R changes neither C3 payload nor any source grade, and no C3 source receipt exists yet.",
		c3q: "The C3P receipt declaration and one exact source-receipt block remain closed source evidence. C3 and C3R are independently sealed; active C3Q changes neither C3 payload nor any source grade, and no C3 source receipt exists yet.",
		c3t: "The C3P receipt declaration and one exact source-receipt block remain closed source evidence. C3, C3R, and C3Q are independently sealed; active C3T changes neither C3 payload nor any source grade, and no C3 source receipt exists yet.",
		c3u: "The C3P receipt declaration and one exact source-receipt block remain closed source evidence. C3, C3R, C3Q, and C3T are independently sealed; active C3U changes neither C3 payload nor any source grade, and no C3 source receipt exists yet.",
		c3b: "The C3P and C3 receipt declarations and their two exact source-receipt blocks remain closed source evidence. C3, C3R, C3Q, C3T, and C3U are independently sealed; C3B changes neither payload nor any source grade.",
	}),
]);
const c3pbPhaseMetadata = Object.freeze({
	before: Object.freeze({
		sealedThrough: "C3V", identity: sealedC3VIdentity, active: "C3M",
		ledger: ".didrun-history/2026-07-19-p07b-c-c3v-final/.didrun/",
		html: ".countershape/evidence/p07b-c-c3v-final-725933fa3c78.html",
	}),
	after: Object.freeze({
		sealedThrough: "C3M", identity: sealedC3MIdentity, active: "C3PB",
		ledger: ".didrun-history/2026-07-19-p07b-c-c3m-final/.didrun/",
		html: ".countershape/evidence/p07b-c-c3m-final-c211d0534864.html",
	}),
	descendant: Object.freeze({
		sealedThrough: "C3PB", identity: sealedC3PBIdentity, active: "C3A",
		ledger: ".didrun-history/2026-07-19-p07b-c-c3pb-final/.didrun/",
		html: ".countershape/evidence/p07b-c-c3pb-final-13369122ba7d.html",
	}),
	c3l: Object.freeze({
		sealedThrough: "C3A", identity: sealedC3AIdentity, active: "C3L",
		ledger: ".didrun-history/2026-07-19-p07b-c-c3a-final/.didrun/",
		html: ".countershape/evidence/p07b-c-c3a-2b84d284.html",
	}),
	c3f: Object.freeze({
		sealedThrough: "C3L", identity: sealedC3LIdentity, active: "C3F",
		ledger: ".didrun-history/2026-07-19-p07b-c-c3l-final/.didrun/",
		html: ".countershape/evidence/p07b-c-c3l-final-efa918928246.html",
	}),
	c3s: Object.freeze({
		sealedThrough: "C3F", identity: sealedC3FIdentity, active: "C3S",
		ledger: ".didrun-history/2026-07-19-p07b-c-c3f-final/.didrun/",
		html: ".countershape/evidence/p07b-c-c3f-final-63038644ba34.html",
	}),
	c3: Object.freeze({
		sealedThrough: "C3S", identity: sealedC3SIdentity, active: "C3",
		ledger: ".didrun-history/2026-07-20-p07b-c-c3s-final/.didrun/",
		html: ".countershape/evidence/p07b-c-c3s-final-47a65b45a50c.html",
	}),
	c3r: Object.freeze({
		sealedThrough: "C3", identity: sealedC3Identity, active: "C3R",
		ledger: ".didrun-history/2026-07-19-p07b-c-c3-final/.didrun/",
		html: ".countershape/evidence/p07b-c-c3-final-029c5cf43853.html",
	}),
	c3q: Object.freeze({
		sealedThrough: "C3R", identity: sealedC3RIdentity, active: "C3Q",
		ledger: ".didrun-history/2026-07-20-p07b-c-c3r-final/.didrun/",
		html: ".countershape/evidence/p07b-c-c3r-final-424273b000f8.html",
	}),
	c3t: Object.freeze({
		sealedThrough: "C3Q", identity: sealedC3QIdentity, active: "C3T",
		ledger: ".didrun-history/2026-07-20-p07b-c-c3q-final/.didrun/",
		html: ".countershape/evidence/p07b-c-c3q-final-818ed90ebbf9.html",
	}),
	c3u: Object.freeze({
		sealedThrough: "C3T", identity: sealedC3TIdentity, active: "C3U",
		ledger: ".didrun-history/2026-07-20-p07b-c-c3t-final/.didrun/",
		html: ".countershape/evidence/p07b-c-c3t-final-9d8430aa70b1.html",
	}),
});

function c3pbPhaseMetadataFor(field, c3Receipt, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority) {
	if (field !== "c3b") return c3pbPhaseMetadata[field];
	if (c3Receipt === undefined) throw new Error("C3B lifecycle metadata requires the validated C3 receipt");
	if (!c3rAuthority || typeof c3rAuthority !== "object") {
		throw new Error("C3B lifecycle metadata requires validated C3R authority");
	}
	if (!c3qAuthority || typeof c3qAuthority !== "object") {
		throw new Error("C3B lifecycle metadata requires validated C3Q authority");
	}
	if (!c3tAuthority || typeof c3tAuthority !== "object") {
		throw new Error("C3B lifecycle metadata requires validated C3T authority");
	}
	if (!c3uAuthority || typeof c3uAuthority !== "object") {
		throw new Error("C3B lifecycle metadata requires validated C3U authority");
	}
	return Object.freeze({
		sealedThrough: "C3U",
		identity: Object.freeze({ commit: c3uAuthority.commit, tree: c3uAuthority.tree }),
		active: "C3B",
		ledger: ".didrun-history/2026-07-20-p07b-c-c3u-final/.didrun/",
		html: `.countershape/evidence/p07b-c-c3u-final-${c3uAuthority.commit.slice(0, 12)}.html`,
	});
}

function c3pbTransitionValue(transition, field, c3Receipt) {
	const value = transition[field];
	return typeof value === "function" ? value(c3Receipt) : value;
}

function c3rSealOutcomeDisclosure(c3rAuthority) {
	const outcome = c3rAuthority?.note?.secrets_override;
	if (typeof outcome !== "boolean") {
		throw new Error("C3B lifecycle metadata requires exact C3R secrets override disclosure");
	}
	return `Sealed C3R's Git note records \`secrets_override: ${outcome}\`; this is the exact recorded seal disclosure, not proof of plain-seal-first sequencing, refusal, review, secret absence, a general credential audit, or publication authority.`;
}

function c3qSealOutcomeDisclosure(c3qAuthority) {
	const outcome = c3qAuthority?.note?.secrets_override;
	const noteBlob = c3qAuthority?.note_blob_oid;
	const noteBodyDigest = c3qAuthority?.note_body_sha256;
	if (!/^[0-9a-f]{40}$/u.test(noteBlob ?? "") || !/^[0-9a-f]{64}$/u.test(noteBodyDigest ?? "") ||
		typeof outcome !== "boolean") {
		throw new Error("C3B lifecycle metadata requires exact C3Q note identity and secrets override disclosure");
	}
	return `Sealed C3Q's Git note blob is \`${noteBlob}\`, its body SHA-256 is \`${noteBodyDigest}\`, and it records \`secrets_override: ${outcome}\`; these are exact recorded note disclosures, not proof of plain-seal-first sequencing, refusal, review, secret absence, a general credential audit, or publication authority.`;
}

function c3tSealOutcomeDisclosure(c3tAuthority) {
	const outcome = c3tAuthority?.note?.secrets_override;
	const noteBlob = c3tAuthority?.note_blob_oid;
	const noteBodyDigest = c3tAuthority?.note_body_sha256;
	if (!/^[0-9a-f]{40}$/u.test(noteBlob ?? "") || !/^[0-9a-f]{64}$/u.test(noteBodyDigest ?? "") ||
		typeof outcome !== "boolean") {
		throw new Error("C3B lifecycle metadata requires exact C3T note identity and secrets override disclosure");
	}
	return `Sealed C3T's Git note blob is \`${noteBlob}\`, its body SHA-256 is \`${noteBodyDigest}\`, and it records \`secrets_override: ${outcome}\`; these are exact recorded note disclosures, not proof of plain-seal-first sequencing, refusal, review, secret absence, a general credential audit, or publication authority.`;
}

function c3uSealOutcomeDisclosure(c3uAuthority) {
	const outcome = c3uAuthority?.note?.secrets_override;
	const noteBlob = c3uAuthority?.note_blob_oid;
	const noteBodyDigest = c3uAuthority?.note_body_sha256;
	if (!/^[0-9a-f]{40}$/u.test(noteBlob ?? "") || !/^[0-9a-f]{64}$/u.test(noteBodyDigest ?? "") ||
		typeof outcome !== "boolean") {
		throw new Error("C3B lifecycle metadata requires exact C3U note identity and secrets override disclosure");
	}
	return `Sealed C3U's Git note blob is \`${noteBlob}\`, its body SHA-256 is \`${noteBodyDigest}\`, and it records \`secrets_override: ${outcome}\`; these are exact recorded note disclosures, not proof of plain-seal-first sequencing, refusal, review, secret absence, a general credential audit, or publication authority.`;
}

function currentStateSubsectionRange(body, heading) {
	const rawLines = body.split("\n");
	const lines = visibleMarkdownLifecycleBody(body).split("\n");
	if (lines.length !== rawLines.length) {
		throw new Error("visible Markdown subsection projection changed line cardinality");
	}
	const offsets = [];
	let offset = 0;
	for (const line of rawLines) {
		offsets.push(offset);
		offset += Buffer.byteLength(line, "utf8") + 1;
	}
	const currentState = lines.flatMap((line, index) => line === "## Current state" ? [index] : []);
	if (currentState.length !== 1) throw new Error(`current-state heading cardinality ${currentState.length}`);
	const outerEnd = lines.findIndex((line, index) => index > currentState[0] && line.startsWith("## "));
	const boundedOuterEnd = outerEnd === -1 ? lines.length : outerEnd;
	const headings = lines.flatMap((line, index) =>
		index > currentState[0] && index < boundedOuterEnd && line === heading ? [index] : []);
	if (headings.length !== 1) throw new Error(`${heading} cardinality ${headings.length}`);
	const startLine = headings[0];
	const endLine = lines.findIndex((line, index) =>
		index > startLine && index < boundedOuterEnd && (line.startsWith("### ") || line.startsWith("## ")));
	const boundedEndLine = endLine === -1 ? boundedOuterEnd : endLine;
	return Object.freeze({
		start: offsets[startLine],
		end: boundedEndLine < offsets.length ? offsets[boundedEndLine] : Buffer.byteLength(body, "utf8"),
	});
}

function currentStateSubsection(body, heading) {
	const range = currentStateSubsectionRange(body, heading);
	return Buffer.from(body, "utf8").subarray(range.start, range.end).toString("utf8");
}

function replaceCurrentStateSubsection(body, heading, replacement) {
	const bytes = Buffer.from(body, "utf8");
	const range = currentStateSubsectionRange(body, heading);
	return Buffer.concat([
		bytes.subarray(0, range.start),
		Buffer.from(replacement, "utf8"),
		bytes.subarray(range.end),
	]).toString("utf8");
}

function c3tActiveMaintenanceSection() {
	const receiptPresence = c3pbTransitionValue(
		c3pbHandoffTransitions.find(({ name }) => name === "receipt presence"),
		"c3t",
	);
	if (typeof receiptPresence !== "string") {
		throw new Error("C3T active maintenance section requires the canonical receipt-absence ruling");
	}
	return [
		"### Active C3T phase-transition self-test maintenance",
		"",
		`C3T is the active exact seven-path \`SOURCE_FULL\` defect repair at \`${requiredC3TDigest}\` and remains \`UNRECEIPTED\`. Its direct parent is sealed C3Q commit \`${sealedC3QIdentity.commit}\`, tree \`${sealedC3QIdentity.tree}\`; C3B's exact three-path roster, receipt profile, and seven labels/types remain frozen.`,
		"",
		`Sealed C3Q's note blob is \`${sealedC3QIdentity.noteBlob}\` and its note-body SHA-256 is \`${sealedC3QIdentity.noteBodySHA256}\`. Current C3T/C3B refusal authority covers 43 exact Markdown paths at \`sha256:52e78638d70f4e229e0b54c0a8311b6e0d1d2acc0d4e439cacf10cde65c62959\`: 2 phases × 43 paths × 15 positive forms, or 1,290 required rejections, plus 430 accepted controls.`,
		"",
		`The permanent trigger ledger \`.didrun-history/2026-07-20-p07b-c-c3b-build-loop-post-c3q-selftest-red/.didrun/\` records a successful C3B plan event followed by the failed outer receipt self-test event. That event exited \`1\` at \`P07B-C phase fixture C3 absent downgrade active state anchor count 0\`; an operator shell-control mistake then declared the failed event. Neither event nor claim is relabeled, and the ledger supports no C3B, C3T, or C3Q claim.`,
		"",
		"C3Q's production active-unit refusal helper and synthetic receipt renderer were exercised on its sealed C3Q-active tree. The later failure narrows—not erases—that evidence: the outer C3 receipt self-test obtained its nominal absent fixture by hiding only the receipt declaration while retaining live C3B status and handoff bytes, so several absent mutations were invalid-baseline false positives and the first exact C3Q transition anchor was absent.",
		"",
		"The sealed C3Q matrix remains exact historical evidence on its sealed tree: it repaired phase-asymmetric self-receipt enforcement across the exact plan-authority Markdown corpus at `sha256:16d63155a00c9e87b7eec79707ad6d47d6f5d70205d18243b8bc687e401e4b66`, with 2 phases × 42 paths × 15 positive forms, 1,260 required rejections, 420 controls, eight supplemental source-identity aliases, and two strict-zero prohibition controls. Its direct synthetic transition used visible-Markdown heading boundaries. That bounded result does not make the outer receipt self-test phase-neutral after later lifecycle evolution. The earlier stashes `1d677db4b9049084ea066263b1bb6c25ab62e5f0` and `fc02f6722716f0600b1e1d7af28cb91925d1047c`, plus red ledger `.didrun-history/2026-07-20-p07b-c-c3b-build-loop-pre-c3q/.didrun/`, remain permanent history and support no current claim.",
		"",
		"C3T repairs the fixture boundary, not the refusal policy. It inverse-renders only the lifecycle-owned C3B subsection and metadata, the exact C3 status receipt projection, and the terminal receipt declaration/block into one validated C3T-active absent fixture. Every absent hostile mutation starts from that passing fixture. The forward renderer then proves absent → present and present → absent → present byte stability across the three controlled receipt paths, with exact-one anchors and visible-Markdown topology.",
		"",
		"Receipt-present inputs must use the canonical one-line receipt declaration and exact terminal handoff block. Ordinary sealed-descendant plan coherence accepts exact C3T ancestry after C3B commits; only the dedicated C3B preseal chain gate requires C3T to be current `HEAD`.",
		"",
		"Every intended C3T grade remains `UNRECEIPTED` until its exact nine-command full profile runs through a fresh ledger, commit, seal, and strict-clean boundary. No C3B receipt, C3 runtime freshness, process-start, C4, general Markdown, or general security claim is implied.",
		"",
		receiptPresence,
		"",
		"",
	].join("\n");
}

function c3uActiveMaintenanceSection() {
	const receiptPresence = c3pbTransitionValue(
		c3pbHandoffTransitions.find(({ name }) => name === "receipt presence"),
		"c3u",
	);
	if (typeof receiptPresence !== "string") {
		throw new Error("C3U active maintenance section requires the canonical receipt-absence ruling");
	}
	return [
		"### Active C3U cross-phase receipt-fixture maintenance",
		"",
		`C3U is the active exact seven-path \`SOURCE_FULL\` defect repair at \`${requiredC3UDigest}\` and remains \`UNRECEIPTED\`. Its direct parent is sealed C3T commit \`${sealedC3TIdentity.commit}\`, tree \`${sealedC3TIdentity.tree}\`; C3B's exact three-path roster, receipt profile, and seven labels/types remain frozen.`,
		"",
		`Sealed C3T's note blob is \`${sealedC3TIdentity.noteBlob}\` and its note-body SHA-256 is \`${sealedC3TIdentity.noteBodySHA256}\`. Current C3U/C3B refusal authority covers 44 exact Markdown paths at \`sha256:1c5bf120f1ab948586bfa594e6ae634fc6acf7a5544febcccb2c4dc475a59a49\`: 2 phases × 44 paths × 15 positive forms, or 1,320 required rejections, plus 440 core accepted controls; eight supplemental natural aliases reject and two explicit strict-zero prohibition controls remain accepted.`,
		"",
		"The permanent trigger ledger `.didrun-history/2026-07-20-p07b-c-c3b-build-loop-post-c3t-c2-terminal-red/.didrun/` records a green live plan followed by the failed receipt self-test at `docs/HANDOFF_MODE_C.md: C3 source receipt block must be the exact terminal handoff block`. It supports no C3B or C3U claim and remains unchanged.",
		"",
		"C3U preserves the terminal invariant. One fail-closed helper partitions an optional already-canonical terminal C3 block, inserts a synthetic C0, C1, C2, or C3P legacy receipt immediately before it, and reattaches the exact captured C3 bytes without parsing or regenerating their payload. Marker-free historical behavior remains byte-stable.",
		"",
		"The helper rejects malformed, hidden, duplicated, nonterminal, or non-LF C3 marker/terminal topology. The actual C0, C1, C2, and C3P fixture callers additionally prove their own strip/parse inverse through terminal-C3 full-plan baselines that revalidate the preserved C3 payload and authority. The helper alone is not evidence that a C3 receipt payload is correct.",
		"",
		"Every intended C3U grade remains `UNRECEIPTED` until its exact nine-command full profile runs through a fresh ledger, commit, seal, and strict-clean boundary. No C3B receipt, runtime freshness, process-start, C4, general Markdown, or general security claim is implied.",
		"",
		receiptPresence,
		"",
		"",
	].join("\n");
}

function c3bActiveMaintenanceSection(receipt, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority) {
	const receiptPresence = c3pbTransitionValue(
		c3pbHandoffTransitions.find(({ name }) => name === "receipt presence"),
		"c3b",
		receipt,
	);
	if (typeof receiptPresence !== "string") {
		throw new Error("C3B active maintenance section requires the canonical receipt-presence ruling");
	}
	return [
		"### Active C3B source receipt reconciliation",
		"",
		`C3B is the active exact three-path \`RECEIPT_RECONCILIATION\` unit at \`${requiredC3BDigest}\` and remains \`UNRECEIPTED\`. It may reconcile only sealed C3 source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`; source and verifier bytes remain unchanged, and no cumulative execution, runtime freshness, or C4 authority is claimed.`,
		"",
		`Sealed C3U commit \`${c3uAuthority.commit}\`, tree \`${c3uAuthority.tree}\`, is the unique direct child of sealed C3T commit \`${c3tAuthority.commit}\` and the immediate checker predecessor. ${c3uSealOutcomeDisclosure(c3uAuthority)}`,
		"",
		`Sealed C3T remains the exact direct child of sealed C3Q commit \`${c3qAuthority.commit}\`. ${c3tSealOutcomeDisclosure(c3tAuthority)}`,
		"",
		`Sealed C3Q remains the exact direct child of sealed C3R commit \`${c3rAuthority.commit}\`. ${c3qSealOutcomeDisclosure(c3qAuthority)}`,
		"",
		"The sealed C3Q matrix remains exact historical evidence on its sealed tree: it repaired phase-asymmetric self-receipt enforcement across the exact plan-authority Markdown corpus at `sha256:16d63155a00c9e87b7eec79707ad6d47d6f5d70205d18243b8bc687e401e4b66`, with 2 phases × 42 paths × 15 positive forms, 1,260 required rejections, 420 controls, eight supplemental source-identity aliases, and two strict-zero prohibition controls. Its direct synthetic transition used visible-Markdown heading boundaries. Earlier red ledger `.didrun-history/2026-07-20-p07b-c-c3b-build-loop-pre-c3q/.didrun/` remains permanent and supports no current claim.",
		"",
		"C3Q's production refusal logic and synthetic C3Q/C3B renderer passed on its sealed tree, but the later outer C3 receipt self-test fixture remained phase-coupled. C3T repaired that invocation by reconstructing one validated C3T-active absent fixture before every hostile mutation and requiring the controlled absent → present and present → absent → present round trips to be byte-stable. The permanent post-C3Q rehearsal failed at `P07B-C phase fixture C3 absent downgrade active state anchor count 0`. C3T's sealed authority covers 2 phases × 43 paths × 15 positive forms over corpus `sha256:52e78638d70f4e229e0b54c0a8311b6e0d1d2acc0d4e439cacf10cde65c62959`, or 1,290 required rejections plus 430 controls.",
		"",
		`C3U then repaired the distinct cross-phase fixture defect exposed only after the terminal C3 source block existed: every synthetic C0, C1, C2, and C3P legacy receipt is inserted before an optional already-canonical terminal C3 block, whose captured bytes remain unchanged. The permanent post-C3T rehearsal ledger \`.didrun-history/2026-07-20-p07b-c-c3b-build-loop-post-c3t-c2-terminal-red/.didrun/\` failed at \`C3 source receipt block must be the exact terminal handoff block\` and supports no claim. Current C3U/C3B refusal authority covers 2 phases × 44 paths × 15 positive forms over corpus \`${planAuthorityMarkdownDigest}\`, or 1,320 required rejections plus 440 core accepted controls; eight supplemental natural aliases reject and two explicit strict-zero prohibition controls remain accepted. This is bounded checker evidence, not a general Markdown or security audit.`,
		"",
		"Receipt-present inputs use the canonical one-line receipt declaration and exact terminal handoff block. Ordinary sealed-descendant plan coherence accepts exact C3U ancestry after this C3B commit; the dedicated C3B preseal chain gate alone required C3U to be current `HEAD`.",
		"",
		"The canonical post-C3T C3B draft remains stash object `cca3045b55ddc00e011c4331f4731bd1a433ea36`; earlier checkpoints `eab9839c902abf52e9729144a2244740228a41eb`, `1d677db4b9049084ea066263b1bb6c25ab62e5f0`, and `fc02f6722716f0600b1e1d7af28cb91925d1047c` remain retained without dropping. Permanent failed rehearsal ledgers remain unchanged; no failed event is relabeled into C3B, C3U, C3T, or older evidence.",
		"",
		"Every intended C3B grade remains `UNRECEIPTED` until its exact seven narrow receipt commands run through a fresh ledger, commit, seal, and strict-clean boundary.",
		"",
		receiptPresence,
		"",
		"",
	].join("\n");
}

const c3pbMetadataLineContracts = Object.freeze({
	"active state": Object.freeze({
		prefix: "- **Pipeline phase:**",
		render: (value, field, c3Receipt, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority) => `- **Pipeline phase:** U0–U6, P07A/U6b, every P07B-A/B boundary, and P07B-C through ${c3pbPhaseMetadataFor(field, c3Receipt, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority).sealedThrough} are sealed and strict-clean. ${value}`,
	}),
	"dirty scope": Object.freeze({
		prefix: "- **Git:**",
		render: (value, field, c3Receipt, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority) => {
			const metadata = c3pbPhaseMetadataFor(field, c3Receipt, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority);
			const stashState = field === "c3"
				? "Applied C3 stash object `5d50090882888c115c54a5bee9edb9411c826832` and earlier checkpoints `f7160bbbc09245a1026eb6396ac76282ab913192`, `fa737661e61c36c10ce793f64852836edec20065`, and `3e766715609680adaef4678fc9aeaa9e3a8bae09` remain retained without dropping; older receipt-history stashes remain untouched."
				: field === "c3r"
					? "C3B draft stash object `fc02f6722716f0600b1e1d7af28cb91925d1047c` and earlier checkpoints remain retained without dropping; older receipt-history stashes remain untouched."
					: field === "c3q"
						? "C3B draft stash objects `1d677db4b9049084ea066263b1bb6c25ab62e5f0` and `fc02f6722716f0600b1e1d7af28cb91925d1047c` and earlier checkpoints remain retained without dropping; the permanent red pre-C3Q development ledger remains archived."
					: field === "c3t"
						? "Post-C3Q C3B draft stash object `eab9839c902abf52e9729144a2244740228a41eb` and earlier checkpoints remain retained without dropping; the permanent red post-C3Q C3B rehearsal ledger remains archived."
					: field === "c3u"
						? "Canonical post-C3T C3B draft stash object `cca3045b55ddc00e011c4331f4731bd1a433ea36` and earlier checkpoints remain retained without dropping; the permanent post-C3T terminal-block red ledger remains archived."
					: field === "c3b"
						? "Applied canonical post-C3T C3B draft stash object `cca3045b55ddc00e011c4331f4731bd1a433ea36` without dropping; earlier stashes `eab9839c902abf52e9729144a2244740228a41eb`, `1d677db4b9049084ea066263b1bb6c25ab62e5f0`, and `fc02f6722716f0600b1e1d7af28cb91925d1047c` remain retained."
				: field === "c3s"
					? "C3 stash object `5d50090882888c115c54a5bee9edb9411c826832` and earlier checkpoints `f7160bbbc09245a1026eb6396ac76282ab913192`, `fa737661e61c36c10ce793f64852836edec20065`, and `3e766715609680adaef4678fc9aeaa9e3a8bae09` remain retained without dropping; older receipt-history stashes remain untouched."
					: field === "c3f"
				? "C3 stash object `f7160bbbc09245a1026eb6396ac76282ab913192` and earlier checkpoints `fa737661e61c36c10ce793f64852836edec20065` and `3e766715609680adaef4678fc9aeaa9e3a8bae09` remain retained without dropping; older receipt-history stashes remain untouched."
				: field === "c3l"
					? "C3 stash objects `fa737661e61c36c10ce793f64852836edec20065` and `3e766715609680adaef4678fc9aeaa9e3a8bae09` remain retained without dropping; older receipt-history stashes remain untouched."
					: "Stash objects `3e766715609680adaef4678fc9aeaa9e3a8bae09`, `5eeb848c334ad3a8a44e4cf61fa298e452188ab1`, `229a60c0630e296b67faa3c81b91fac73f9f63ff`, and `1d2c14f160ff42d509d96219512fa413830a9b1d` remain retained; address each by full identity.";
			return `- **Git:** repository is on \`codex/countershape-autopilot\` at sealed ${metadata.sealedThrough} HEAD \`${metadata.identity.commit}\`, tree \`${metadata.identity.tree}\`. ${value} ${stashState}`;
		},
	}),
	"ledger profile": Object.freeze({
		prefix: "- **didrun:**",
		render: (value, field, c3Receipt, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority) => {
			const metadata = c3pbPhaseMetadataFor(field, c3Receipt, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority);
			const predecessorSealDisclosure = field === "c3b"
				? ` ${c3rSealOutcomeDisclosure(c3rAuthority)} ${c3qSealOutcomeDisclosure(c3qAuthority)} ${c3tSealOutcomeDisclosure(c3tAuthority)} ${c3uSealOutcomeDisclosure(c3uAuthority)}`
				: "";
			const liveLedgerState = field === "c3b"
				? "Before final-ledger rotation, any live `.didrun/` is C3B development evidence only; after explicit rotation, the fresh seven-claim ledger is C3B final receipt evidence pending commit, seal, and strict closure."
				: field === "c3r"
					? "Before final-ledger rotation, any live `.didrun/` is C3R development evidence only; after explicit rotation, the fresh nine-claim ledger is C3R final evidence pending commit, seal, and strict closure."
					: field === "c3q"
						? "Before final-ledger rotation, any live `.didrun/` is C3Q development evidence only; after explicit rotation, the fresh nine-claim ledger is C3Q final evidence pending commit, seal, and strict closure."
					: field === "c3t"
						? "Before final-ledger rotation, any live `.didrun/` is C3T development evidence only; after explicit rotation, the fresh nine-claim ledger is C3T final evidence pending commit, seal, and strict closure."
					: field === "c3u"
						? "Before final-ledger rotation, any live `.didrun/` is C3U development evidence only; after explicit rotation, the fresh nine-claim ledger is C3U final evidence pending commit, seal, and strict closure."
				: field === "c3"
					? "Before final-ledger rotation, any live `.didrun/` is C3 development evidence only; after explicit rotation, the fresh eighteen-claim ledger is C3 final evidence pending commit, seal, and strict closure."
					: field === "c3s"
						? "Before final-ledger rotation, any live `.didrun/` is C3S development evidence only; after explicit rotation, the fresh ten-claim ledger is C3S final evidence pending commit, seal, and strict closure."
						: field === "c3f"
				? "Before final-ledger rotation, any live `.didrun/` is C3F development evidence only; after explicit rotation, the fresh ten-claim ledger is C3F final evidence pending commit, seal, and strict closure."
				: field === "c3l"
					? "Before final-ledger rotation, any live `.didrun/` is C3L development evidence only; after explicit rotation, the fresh twelve-claim ledger is C3L final evidence pending commit, seal, and strict closure."
					: `Any live \`.didrun/\` created now is ${metadata.active} development evidence only.`;
			return `- **didrun:** installed globally and unchanged. ${metadata.sealedThrough}'s sealed final ledger is \`${metadata.ledger}\`; its exact-commit HTML is \`${metadata.html}\`.${predecessorSealDisclosure} ${value} ${liveLedgerState}`;
		},
	}),
	"verification battery": Object.freeze({
		prefix: "- **Current baseline:**",
		render: (value) => `- **Current baseline:** from a fresh repository-root shell, run \`/opt/homebrew/bin/node tools/verify-current.mjs\`. ${value}`,
	}),
});
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
const c3aClaimLabels = Object.freeze([
	"P07B-C C3A cumulative-admission plan coherence",
	"P07B-C C3A cumulative-admission plan defensive self-test",
	"P07B-C C3A runtime-version exact 128-byte model boundary",
	"P07B-C C3A cumulative B architecture compatibility",
	"P07B-C C3A B admission and entrypoint defensive self-test",
	"P07B-C C3A unit-scope defensive self-test",
	"P07B-C C3A cumulative verifier self-test",
	"P07B-C C3A cumulative verification",
	"P07B-C C3A exact thirteen-path staged scope and diff integrity",
	"P07B-C C3A scoped staged credential-pattern scan",
	"P07B-C C3A preceding didrun chain integrity",
]);
const c3aClaimTypes = Object.freeze([
	...Array(8).fill("tests-pass"),
	...Array(3).fill("command-succeeded"),
]);
const c3aFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c3a-final");
const c3aHermeticArgvPrefix = Object.freeze([
	"/usr/bin/env", "-i",
	`HOME=${resolve(c3aFinalRunRoot, "home")}`,
	`PWD=${repositoryRoot}`,
	`TMPDIR=${resolve(c3aFinalRunRoot, "tmp")}`,
	`GOTMPDIR=${resolve(c3aFinalRunRoot, "gotmp")}`,
	`GOCACHE=${resolve(c3aFinalRunRoot, "gocache")}`,
	`GOPATH=${resolve(c3aFinalRunRoot, "gopath")}`,
	`GOMODCACHE=${resolve(c3aFinalRunRoot, "gomodcache")}`,
	"GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off",
	"GOFLAGS=-mod=readonly -buildvcs=false -p=1", "CGO_ENABLED=1", "GOMAXPROCS=2",
	"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "NODE_OPTIONS=", "NODE_PATH=",
	"PATH=/opt/homebrew/bin:/usr/bin:/bin", "SHELL=/bin/sh",
	"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_LAZY_FETCH=1",
	"GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0",
	"COUNTERSHAPE_GO=/opt/homebrew/bin/go", "COUNTERSHAPE_NODE=/opt/homebrew/bin/node",
	"COUNTERSHAPE_GIT=/usr/bin/git", "COUNTERSHAPE_SH=/bin/sh",
	"COUNTERSHAPE_CC=/usr/bin/clang", "COUNTERSHAPE_CXX=/usr/bin/clang++",
	"CC=/usr/bin/clang", "CXX=/usr/bin/clang++",
]);
const c3aHermeticArgv = (...tail) => Object.freeze([...c3aHermeticArgvPrefix, ...tail]);
const c3aExpectedClaimArgv = Object.freeze([
	c3aHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"),
	c3aHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"),
	c3aHermeticArgv("/opt/homebrew/bin/go", "test", "-mod=readonly", "-buildvcs=false", "-p=1", "-count=1", "-run", "^TestTargetPrimitiveBoundsAndDerivedRelations$", "./internal/contractexec/model"),
	c3aHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-b-architecture.mjs"),
	c3aHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-b-architecture-selftest.mjs"),
	c3aHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"),
	c3aHermeticArgv("/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"),
	c3aHermeticArgv("/opt/homebrew/bin/node", "tools/verify-current.mjs"),
	c3aHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3A", "--source-final-gate"),
	c3aHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3A", "--credential-scan"),
	c3aHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3a-preseal-ledger"),
]);
const c3aFinalRunDirectories = Object.freeze([
	c3aFinalRunRoot,
	...[
		"home", "tmp", "gotmp", "gocache", "gopath", "gomodcache",
	].map((name) => resolve(c3aFinalRunRoot, name)),
]);

function parseHermeticArgvPrefix(phase, prefix) {
	if (prefix.length < 3 || prefix[0] !== "/usr/bin/env" || prefix[1] !== "-i") {
		throw new Error(`P07B-C ${phase} hermetic prefix must start with /usr/bin/env -i`);
	}
	const bindings = new Map();
	for (const assignment of prefix.slice(2)) {
		const match = /^([A-Z][A-Z0-9_]*)=(.*)$/su.exec(assignment);
		if (!match || bindings.has(match[1])) {
			throw new Error(`P07B-C ${phase} hermetic prefix assignment invalid: ${assignment}`);
		}
		bindings.set(match[1], match[2]);
	}
	return bindings;
}

function requireHermeticCommandPlan(phase, expectedArgv, prefix) {
	parseHermeticArgvPrefix(phase, prefix);
	for (let index = 0; index < expectedArgv.length; index += 1) {
		if (!isDeepStrictEqual(expectedArgv[index].slice(0, prefix.length), prefix) ||
			expectedArgv[index].length === prefix.length) {
			throw new Error(`P07B-C ${phase} command ${index} does not carry the exact hermetic prefix`);
		}
	}
}

function requireEffectiveHermeticContext(phase, prefix, cwd, environment) {
	if (cwd !== repositoryRoot) {
		throw new Error(`P07B-C ${phase} live cwd mismatch: ${cwd}`);
	}
	for (const [name, expected] of parseHermeticArgvPrefix(phase, prefix)) {
		if (environment[name] !== expected) {
			throw new Error(`P07B-C ${phase} effective environment mismatch: ${name}`);
		}
	}
}

function requireBoundPresealEventContext(phase, event, expectedArgv, expectedFingerprint) {
	if (event.cwd !== repositoryRoot || !/^[0-9a-f]{16}$/u.test(event.env_fingerprint ?? "") ||
		(expectedFingerprint !== undefined && event.env_fingerprint !== expectedFingerprint) ||
		!isDeepStrictEqual(event.argv, expectedArgv)) {
		throw new Error(`P07B-C ${phase} live event wrapper context mismatch`);
	}
	return event.env_fingerprint;
}

function runC3AHermeticPresealSelfTest() {
	requireHermeticCommandPlan("C3A self-test", c3aExpectedClaimArgv, c3aHermeticArgvPrefix);
	const environment = Object.fromEntries(parseHermeticArgvPrefix("C3A self-test", c3aHermeticArgvPrefix));
	requireEffectiveHermeticContext("C3A self-test", c3aHermeticArgvPrefix, repositoryRoot, environment);
	const event = {
		argv: [...c3aExpectedClaimArgv[0]], cwd: repositoryRoot, env_fingerprint: "0123456789abcdef",
	};
	requireBoundPresealEventContext("C3A self-test", event, c3aExpectedClaimArgv[0]);
	let rejected = 0;
	const requireRejected = (name, run) => {
		let message = "";
		try { run(); } catch (error) { message = String(error.message); }
		if (!message.includes("P07B-C C3A self-test")) {
			throw new Error(`P07B-C C3A hermetic preseal self-test false negative: ${name} (${message || "accepted"})`);
		}
		rejected += 1;
	};
	requireRejected("cwd drift", () => requireBoundPresealEventContext(
		"C3A self-test", { ...event, cwd: resolve(repositoryRoot, "docs") }, c3aExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("malformed wrapper fingerprint", () => requireBoundPresealEventContext(
		"C3A self-test", { ...event, env_fingerprint: "not-a-digest" }, c3aExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("wrapper fingerprint drift", () => requireBoundPresealEventContext(
		"C3A self-test", { ...event, env_fingerprint: "fedcba9876543210" }, c3aExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("removed hermetic prefix", () => requireBoundPresealEventContext(
		"C3A self-test", { ...event, argv: event.argv.slice(c3aHermeticArgvPrefix.length) }, c3aExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("one command lacks common prefix", () => requireHermeticCommandPlan(
		"C3A self-test", [...c3aExpectedClaimArgv.slice(0, -1), c3aExpectedClaimArgv.at(-1).slice(c3aHermeticArgvPrefix.length)], c3aHermeticArgvPrefix,
	));
	for (const name of ["NODE_OPTIONS", "NODE_PATH", "PATH", "COUNTERSHAPE_NODE", "COUNTERSHAPE_GIT", "COUNTERSHAPE_SH"]) {
		requireRejected(`${name} drift`, () => requireEffectiveHermeticContext(
			"C3A self-test", c3aHermeticArgvPrefix, repositoryRoot, { ...environment, [name]: `${environment[name]}hostile` },
		));
	}
	return rejected;
}

async function requirePrivateC3AFinalRunDirectories() {
	for (const path of c3aFinalRunDirectories) {
		const stat = await lstat(path);
		const resolved = await realpath(path);
		if (!stat.isDirectory() || stat.isSymbolicLink() || (stat.mode & 0o777) !== 0o700 ||
			(typeof process.getuid === "function" && stat.uid !== process.getuid()) || resolved !== path) {
			throw new Error(`P07B-C C3A final run directory is not exact private authority: ${path}`);
		}
	}
}
const c3lClaimLabels = Object.freeze([
	"P07B-C C3L legacy-lattice plan coherence",
	"P07B-C C3L legacy-lattice plan defensive self-test",
	"P07B-C C3L historical U6 architecture compatibility",
	"P07B-C C3L U6 phase-admission defensive self-test",
	"P07B-C C3L cumulative B architecture compatibility",
	"P07B-C C3L B defensive self-test",
	"P07B-C C3L unit-scope defensive self-test",
	"P07B-C C3L cumulative verifier self-test",
	"P07B-C C3L cumulative verification",
	"P07B-C C3L exact nine-path staged scope and diff integrity",
	"P07B-C C3L scoped staged credential-pattern scan",
	"P07B-C C3L sealed-C3A predecessor and preceding didrun chain integrity",
]);
const c3lClaimTypes = Object.freeze([
	...Array(9).fill("tests-pass"),
	...Array(3).fill("command-succeeded"),
]);
const c3lFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c3l-final");
const c3lHermeticArgvPrefix = Object.freeze([
	"/usr/bin/env", "-i",
	`HOME=${resolve(c3lFinalRunRoot, "home")}`,
	`PWD=${repositoryRoot}`,
	`TMPDIR=${resolve(c3lFinalRunRoot, "tmp")}`,
	`GOTMPDIR=${resolve(c3lFinalRunRoot, "gotmp")}`,
	`GOCACHE=${resolve(c3lFinalRunRoot, "gocache")}`,
	`GOPATH=${resolve(c3lFinalRunRoot, "gopath")}`,
	`GOMODCACHE=${resolve(c3lFinalRunRoot, "gomodcache")}`,
	"GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off",
	"GOFLAGS=-mod=readonly -buildvcs=false -p=1", "CGO_ENABLED=1", "GOMAXPROCS=2",
	"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "NODE_OPTIONS=", "NODE_PATH=",
	"PATH=/opt/homebrew/bin:/usr/bin:/bin", "SHELL=/bin/sh",
	"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_LAZY_FETCH=1",
	"GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0",
	"COUNTERSHAPE_GO=/opt/homebrew/bin/go", "COUNTERSHAPE_NODE=/opt/homebrew/bin/node",
	"COUNTERSHAPE_GIT=/usr/bin/git", "COUNTERSHAPE_SH=/bin/sh",
	"COUNTERSHAPE_CC=/usr/bin/clang", "COUNTERSHAPE_CXX=/usr/bin/clang++",
	"CC=/usr/bin/clang", "CXX=/usr/bin/clang++",
]);
const c3lHermeticArgv = (...tail) => Object.freeze([...c3lHermeticArgvPrefix, ...tail]);
const c3lExpectedClaimArgv = Object.freeze([
	c3lHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"),
	c3lHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"),
	c3lHermeticArgv("/opt/homebrew/bin/node", "tools/check-u6-architecture.mjs"),
	c3lHermeticArgv("/opt/homebrew/bin/node", "tools/check-u6-architecture-selftest.mjs"),
	c3lHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-b-architecture.mjs"),
	c3lHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-b-architecture-selftest.mjs"),
	c3lHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"),
	c3lHermeticArgv("/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"),
	c3lHermeticArgv("/opt/homebrew/bin/node", "tools/verify-current.mjs"),
	c3lHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3L", "--source-final-gate"),
	c3lHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3L", "--credential-scan"),
	c3lHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3l-preseal-ledger"),
]);
const c3lFinalRunDirectories = Object.freeze([
	c3lFinalRunRoot,
	...["home", "tmp", "gotmp", "gocache", "gopath", "gomodcache"]
		.map((name) => resolve(c3lFinalRunRoot, name)),
]);
const c3fClaimLabels = Object.freeze([
	"P07B-C C3F B future-surface plan coherence",
	"P07B-C C3F B future-surface plan defensive self-test",
	"P07B-C C3F historical B architecture compatibility",
	"P07B-C C3F B conditional future-surface defensive self-test",
	"P07B-C C3F unit-scope defensive self-test",
	"P07B-C C3F cumulative verifier self-test",
	"P07B-C C3F cumulative verification",
	"P07B-C C3F exact ten-path staged scope and diff integrity",
	"P07B-C C3F scoped staged credential-pattern scan",
	"P07B-C C3F sealed-C3L predecessor and preceding didrun chain integrity",
]);
const c3fClaimTypes = Object.freeze([
	...Array(7).fill("tests-pass"),
	...Array(3).fill("command-succeeded"),
]);
const c3fFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c3f-final");
const c3fHermeticArgvPrefix = Object.freeze([
	"/usr/bin/env", "-i",
	`HOME=${resolve(c3fFinalRunRoot, "home")}`,
	`PWD=${repositoryRoot}`,
	`TMPDIR=${resolve(c3fFinalRunRoot, "tmp")}`,
	`GOTMPDIR=${resolve(c3fFinalRunRoot, "gotmp")}`,
	`GOCACHE=${resolve(c3fFinalRunRoot, "gocache")}`,
	`GOPATH=${resolve(c3fFinalRunRoot, "gopath")}`,
	`GOMODCACHE=${resolve(c3fFinalRunRoot, "gomodcache")}`,
	"GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off",
	"GOFLAGS=-mod=readonly -buildvcs=false -p=1", "CGO_ENABLED=1", "GOMAXPROCS=2",
	"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "NODE_OPTIONS=", "NODE_PATH=",
	"PATH=/opt/homebrew/bin:/usr/bin:/bin", "SHELL=/bin/sh",
	"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_LAZY_FETCH=1",
	"GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0",
	"COUNTERSHAPE_GO=/opt/homebrew/bin/go", "COUNTERSHAPE_NODE=/opt/homebrew/bin/node",
	"COUNTERSHAPE_GIT=/usr/bin/git", "COUNTERSHAPE_SH=/bin/sh",
	"COUNTERSHAPE_CC=/usr/bin/clang", "COUNTERSHAPE_CXX=/usr/bin/clang++",
	"CC=/usr/bin/clang", "CXX=/usr/bin/clang++",
]);
const c3fHermeticArgv = (...tail) => Object.freeze([...c3fHermeticArgvPrefix, ...tail]);
const c3fExpectedClaimArgv = Object.freeze([
	c3fHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"),
	c3fHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"),
	c3fHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-b-architecture.mjs"),
	c3fHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-b-architecture-selftest.mjs"),
	c3fHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"),
	c3fHermeticArgv("/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"),
	c3fHermeticArgv("/opt/homebrew/bin/node", "tools/verify-current.mjs"),
	c3fHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3F", "--source-final-gate"),
	c3fHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3F", "--credential-scan"),
	c3fHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3f-preseal-ledger"),
]);
const c3fFinalRunDirectories = Object.freeze([
	c3fFinalRunRoot,
	...["home", "tmp", "gotmp", "gocache", "gopath", "gomodcache"]
		.map((name) => resolve(c3fFinalRunRoot, name)),
]);
const c3sClaimLabels = Object.freeze([
	"P07B-C C3S cumulative-selftest plan coherence",
	"P07B-C C3S cumulative-selftest plan defensive self-test",
	"P07B-C C3S historical U6 architecture compatibility",
	"P07B-C C3S U6 phase-stable fixture defensive self-test",
	"P07B-C C3S unit-scope defensive self-test",
	"P07B-C C3S cumulative verifier self-test",
	"P07B-C C3S cumulative verification",
	"P07B-C C3S exact eight-path staged scope and diff integrity",
	"P07B-C C3S scoped staged credential-pattern scan",
	"P07B-C C3S sealed-C3F predecessor and preceding didrun chain integrity",
]);
const c3sClaimTypes = Object.freeze([
	...Array(7).fill("tests-pass"),
	...Array(3).fill("command-succeeded"),
]);
const c3sFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c3s-final");
const c3sHermeticArgvPrefix = Object.freeze([
	"/usr/bin/env", "-i",
	`HOME=${resolve(c3sFinalRunRoot, "home")}`,
	`PWD=${repositoryRoot}`,
	`TMPDIR=${resolve(c3sFinalRunRoot, "tmp")}`,
	`GOTMPDIR=${resolve(c3sFinalRunRoot, "gotmp")}`,
	`GOCACHE=${resolve(c3sFinalRunRoot, "gocache")}`,
	`GOPATH=${resolve(c3sFinalRunRoot, "gopath")}`,
	`GOMODCACHE=${resolve(c3sFinalRunRoot, "gomodcache")}`,
	"GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off",
	"GOFLAGS=-mod=readonly -buildvcs=false -p=1", "CGO_ENABLED=1", "GOMAXPROCS=2",
	"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "NODE_OPTIONS=", "NODE_PATH=",
	"PATH=/opt/homebrew/bin:/usr/bin:/bin", "SHELL=/bin/sh",
	"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_LAZY_FETCH=1",
	"GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0",
	"COUNTERSHAPE_GO=/opt/homebrew/bin/go", "COUNTERSHAPE_NODE=/opt/homebrew/bin/node",
	"COUNTERSHAPE_GIT=/usr/bin/git", "COUNTERSHAPE_SH=/bin/sh",
	"COUNTERSHAPE_CC=/usr/bin/clang", "COUNTERSHAPE_CXX=/usr/bin/clang++",
	"CC=/usr/bin/clang", "CXX=/usr/bin/clang++",
]);
const c3ClaimLabels = Object.freeze([
	"P07B-C C3 plan, source scope, and C3B dual-phase coherence",
	"P07B-C C3 plan and receipt-phase defensive self-test",
	"P07B-C C3 exact official-target authority profile",
	"P07B-C C3 direct single-target Git materialization profile",
	"P07B-C C3 Darwin boot-session measurement profile",
	"P07B-C C3 admitted Node runtime profile",
	"P07B-C C3 inert attempt and target store-bridge profile",
	"P07B-C C3 focused authority race suite",
	"P07B-C C3 cumulative architecture boundary",
	"P07B-C C3 architecture defensive self-test",
	"P07B-C C3 predecessor B compatibility",
	"P07B-C C3 predecessor B defensive self-test",
	"P07B-C C3 unit-scope defensive self-test",
	"P07B-C C3 cumulative verifier self-test",
	"P07B-C C3 cumulative verification",
	"P07B-C C3 exact forty-path staged scope and diff integrity",
	"P07B-C C3 scoped staged credential-pattern scan",
	"P07B-C C3 sealed-C3S predecessor, C3F/C3L/C3A ancestry, and preceding didrun chain integrity",
]);
const c3ClaimTypes = Object.freeze([
	...Array(15).fill("tests-pass"),
	...Array(3).fill("command-succeeded"),
]);
const c3FinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c3-final");
const c3HermeticArgvPrefix = Object.freeze(c3sHermeticArgvPrefix.map((argument) =>
	argument.replaceAll(c3sFinalRunRoot, c3FinalRunRoot)));
const c3sHermeticArgv = (...tail) => Object.freeze([...c3sHermeticArgvPrefix, ...tail]);
const c3sExpectedClaimArgv = Object.freeze([
	c3sHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"),
	c3sHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"),
	c3sHermeticArgv("/opt/homebrew/bin/node", "tools/check-u6-architecture.mjs"),
	c3sHermeticArgv("/opt/homebrew/bin/node", "tools/check-u6-architecture-selftest.mjs"),
	c3sHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"),
	c3sHermeticArgv("/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"),
	c3sHermeticArgv("/opt/homebrew/bin/node", "tools/verify-current.mjs"),
	c3sHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3S", "--source-final-gate"),
	c3sHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3S", "--credential-scan"),
	c3sHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3s-preseal-ledger"),
]);
const c3sFinalRunDirectories = Object.freeze([
	c3sFinalRunRoot,
	...["home", "tmp", "gotmp", "gocache", "gopath", "gomodcache"]
		.map((name) => resolve(c3sFinalRunRoot, name)),
]);
const c3HermeticArgv = (...tail) => Object.freeze([...c3HermeticArgvPrefix, ...tail]);
const c3RaceTestPattern = "^(TestC3OfficialTargetClosedCapabilityAndDefensiveGetters|TestC3HostEpochConcurrentMeasurementsNeverCache|TestC3NodeRuntimeCopiedCapabilitiesSerializeRevalidation|TestC3ConformanceAttemptConcurrentValidationAndReopenAreRaceFree)$";
const c3RaceNotePreviewPattern = "^(«redacted:high-entropy»|TestC3HostEpochConcurrentMeasurementsNeverCache|TestC3NodeRuntimeCopiedCapabilitiesSerializeRevalidation|TestC3ConformanceAttemptConcurrentValidationAndReopenAreRaceFree)$";
const c3ExpectedClaimArgv = Object.freeze([
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture.mjs", "--run-go-json", "c3-official-target"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture.mjs", "--run-go-json", "c3-single-target"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture.mjs", "--run-go-json", "c3-hostepoch"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture.mjs", "--run-go-json", "c3-noderuntime"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture.mjs", "--run-go-json", "c3-store-bridge"),
	c3HermeticArgv("/opt/homebrew/bin/go", "test", "-mod=readonly", "-buildvcs=false", "-p=1", "-race", "-count=1", "-timeout=20m", "-run", c3RaceTestPattern, "./internal/contractexec", "./internal/hostepoch", "./internal/noderuntime", "./internal/store"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture.mjs", "--c3"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-architecture-selftest.mjs", "--c3"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-b-architecture.mjs"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-b-architecture-selftest.mjs"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/verify-current.mjs"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3", "--source-final-gate"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3", "--credential-scan"),
	c3HermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3-preseal-ledger"),
]);
const c3FinalRunDirectories = Object.freeze([
	c3FinalRunRoot,
	...["home", "tmp", "gotmp", "gocache", "gopath", "gomodcache"]
		.map((name) => resolve(c3FinalRunRoot, name)),
]);
const c3rClaimLabels = Object.freeze([
	"P07B-C C3R note-preview plan coherence",
	"P07B-C C3R note-preview and dual-phase defensive self-test",
	"P07B-C C3R sealed-C3 Git-note argv-preview compatibility",
	"P07B-C C3R unit-scope defensive self-test",
	"P07B-C C3R cumulative verifier self-test",
	"P07B-C C3R cumulative verification",
	"P07B-C C3R exact seven-path staged scope and diff integrity",
	"P07B-C C3R scoped staged credential-pattern scan",
	"P07B-C C3R sealed-C3 predecessor and preceding didrun chain integrity",
]);
const c3rClaimTypes = Object.freeze([
	...Array(6).fill("tests-pass"),
	...Array(3).fill("command-succeeded"),
]);
const c3rFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c3r-final");
const c3rHermeticArgvPrefix = Object.freeze(c3HermeticArgvPrefix.map((argument) =>
	argument.replaceAll(c3FinalRunRoot, c3rFinalRunRoot)));
const c3rHermeticArgv = (...tail) => Object.freeze([...c3rHermeticArgvPrefix, ...tail]);
const c3rExpectedClaimArgv = Object.freeze([
	c3rHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"),
	c3rHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"),
	c3rHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3r-sealed-c3-note"),
	c3rHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"),
	c3rHermeticArgv("/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"),
	c3rHermeticArgv("/opt/homebrew/bin/node", "tools/verify-current.mjs"),
	c3rHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3R", "--source-final-gate"),
	c3rHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3R", "--credential-scan"),
	c3rHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3r-preseal-ledger"),
]);
const c3rFinalRunDirectories = Object.freeze([
	c3rFinalRunRoot,
	...["home", "tmp", "gotmp", "gocache", "gopath", "gomodcache"]
		.map((name) => resolve(c3rFinalRunRoot, name)),
]);
const c3qClaimLabels = Object.freeze([
	"P07B-C C3Q self-receipt plan coherence",
	"P07B-C C3Q dual-phase and authority-surface defensive self-test",
	"P07B-C C3Q sealed-C3R Git-note compatibility",
	"P07B-C C3Q unit-scope defensive self-test",
	"P07B-C C3Q cumulative verifier self-test",
	"P07B-C C3Q cumulative verification",
	"P07B-C C3Q exact seven-path staged scope and diff integrity",
	"P07B-C C3Q scoped staged credential-pattern scan",
	"P07B-C C3Q sealed-C3R predecessor and preceding didrun chain integrity",
]);
const c3qClaimTypes = Object.freeze([
	...Array(6).fill("tests-pass"),
	...Array(3).fill("command-succeeded"),
]);
const c3qFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c3q-final");
const c3qHermeticArgvPrefix = Object.freeze(c3rHermeticArgvPrefix.map((argument) =>
	argument.replaceAll(c3rFinalRunRoot, c3qFinalRunRoot)));
const c3qHermeticArgv = (...tail) => Object.freeze([...c3qHermeticArgvPrefix, ...tail]);
const c3qFinalCommandTails = Object.freeze([
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3q-sealed-c3r-note"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3Q", "--source-final-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3Q", "--credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3q-preseal-ledger"]),
]);
const c3qExpectedClaimArgv = Object.freeze(c3qFinalCommandTails.map((tail) => c3qHermeticArgv(...tail)));
const c3qFinalCommandOrderMarkdown = c3qFinalCommandTails
	.map((tail, index) => `${index + 1}. \`${tail.join(" ")}\``).join("\n");
const c3qFinalRunDirectories = Object.freeze([
	c3qFinalRunRoot,
	...["home", "tmp", "gotmp", "gocache", "gopath", "gomodcache"]
		.map((name) => resolve(c3qFinalRunRoot, name)),
]);
const c3tClaimLabels = Object.freeze([
	"P07B-C C3T phase-selftest plan coherence",
	"P07B-C C3T receipt-phase fixture defensive self-test",
	"P07B-C C3T sealed-C3Q Git-note and ancestry compatibility",
	"P07B-C C3T unit-scope defensive self-test",
	"P07B-C C3T cumulative verifier self-test",
	"P07B-C C3T cumulative verification",
	"P07B-C C3T exact seven-path staged scope and diff integrity",
	"P07B-C C3T scoped staged credential-pattern scan",
	"P07B-C C3T sealed-C3Q predecessor and preceding didrun chain integrity",
]);
const c3tClaimTypes = Object.freeze([
	...Array(6).fill("tests-pass"),
	...Array(3).fill("command-succeeded"),
]);
const c3tFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c3t-final");
const c3tHermeticArgvPrefix = Object.freeze(c3qHermeticArgvPrefix.map((argument) =>
	argument.replaceAll(c3qFinalRunRoot, c3tFinalRunRoot)));
const c3tHermeticArgv = (...tail) => Object.freeze([...c3tHermeticArgvPrefix, ...tail]);
const c3tFinalCommandTails = Object.freeze([
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3t-sealed-c3q-note"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3T", "--source-final-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3T", "--credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3t-preseal-ledger"]),
]);
const c3tExpectedClaimArgv = Object.freeze(c3tFinalCommandTails.map((tail) => c3tHermeticArgv(...tail)));
const c3tFinalCommandOrderMarkdown = c3tFinalCommandTails
	.map((tail, index) => `${index + 1}. \`${tail.join(" ")}\``).join("\n");
const c3tFinalRunDirectories = Object.freeze([
	c3tFinalRunRoot,
	...["home", "tmp", "gotmp", "gocache", "gopath", "gomodcache"]
		.map((name) => resolve(c3tFinalRunRoot, name)),
]);
const c3uClaimLabels = Object.freeze([
	"P07B-C C3U cross-phase receipt-fixture plan coherence",
	"P07B-C C3U legacy receipt fixture defensive self-test",
	"P07B-C C3U sealed-C3T Git-note and ancestry compatibility",
	"P07B-C C3U unit-scope defensive self-test",
	"P07B-C C3U cumulative verifier self-test",
	"P07B-C C3U cumulative verification",
	"P07B-C C3U exact seven-path staged scope and diff integrity",
	"P07B-C C3U scoped staged credential-pattern scan",
	"P07B-C C3U sealed-C3T predecessor and preceding didrun chain integrity",
]);
const c3uClaimTypes = Object.freeze([
	...Array(6).fill("tests-pass"),
	...Array(3).fill("command-succeeded"),
]);
const c3uFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c3u-final");
const c3uHermeticArgvPrefix = Object.freeze(c3tHermeticArgvPrefix.map((argument) =>
	argument.replaceAll(c3tFinalRunRoot, c3uFinalRunRoot)));
const c3uHermeticArgv = (...tail) => Object.freeze([...c3uHermeticArgvPrefix, ...tail]);
const c3uFinalCommandTails = Object.freeze([
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3u-sealed-c3t-note"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3U", "--source-final-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3U", "--credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3u-preseal-ledger"]),
]);
const c3uExpectedClaimArgv = Object.freeze(c3uFinalCommandTails.map((tail) => c3uHermeticArgv(...tail)));
const c3uFinalCommandOrderMarkdown = c3uFinalCommandTails
	.map((tail, index) => `${index + 1}. \`${tail.join(" ")}\``).join("\n");
const c3uFinalRunDirectories = Object.freeze([
	c3uFinalRunRoot,
	...["home", "tmp", "gotmp", "gocache", "gopath", "gomodcache"]
		.map((name) => resolve(c3uFinalRunRoot, name)),
]);
const c3bClaimLabels = Object.freeze([
	"P07B-C C3B source receipt reconciliation",
	"P07B-C C3B receipt checker defensive self-test",
	"P07B-C C3B declared local source-evidence snapshot match",
	"P07B-C C3B receipt-only Go build",
	"P07B-C C3B exact three-path staged scope and diff integrity",
	"P07B-C C3B scoped staged credential-pattern scan",
	"P07B-C C3B preceding didrun chain integrity",
]);
const c3bClaimTypes = Object.freeze([
	...Array(3).fill("tests-pass"),
	...Array(4).fill("command-succeeded"),
]);
const c3bFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c3b-final");
const c3bHermeticArgvPrefix = Object.freeze(c3HermeticArgvPrefix.map((argument) =>
	argument.replaceAll(c3FinalRunRoot, c3bFinalRunRoot)));
const c3bHermeticArgv = (...tail) => Object.freeze([...c3bHermeticArgvPrefix, ...tail]);
const c3bExpectedClaimArgv = Object.freeze([
	c3bHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"),
	c3bHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"),
	c3bHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3-local-evidence"),
	c3bHermeticArgv("/opt/homebrew/bin/go", "build", "-mod=readonly", "-buildvcs=false", "-p=1", "./..."),
	c3bHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3b-staged"),
	c3bHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3b-credential-scan"),
	c3bHermeticArgv("/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3b-preseal-ledger"),
]);
const c3bFinalRunDirectories = Object.freeze([
	c3bFinalRunRoot,
	...["home", "tmp", "gotmp", "gocache", "gopath", "gomodcache"]
		.map((name) => resolve(c3bFinalRunRoot, name)),
]);
const c3PredecessorManifestPath = "spec/verification/p07b-c-c3-predecessors.json";
const c3PredecessorAuthorities = Object.freeze([
	Object.freeze({
		unit: "C3S", identity: sealedC3SIdentity, parent: sealedC3FIdentity.commit,
		subject: sealedC3SIdentity.subject, scopeDigest: requiredC3SDigest,
		claimLabels: c3sClaimLabels, claimTypes: c3sClaimTypes, validateNote: validateSealedC3SNote,
	}),
	Object.freeze({
		unit: "C3F", identity: sealedC3FIdentity, parent: sealedC3LIdentity.commit,
		subject: sealedC3FIdentity.subject, scopeDigest: requiredC3FDigest,
		claimLabels: c3fClaimLabels, claimTypes: c3fClaimTypes, validateNote: validateSealedC3FNote,
	}),
	Object.freeze({
		unit: "C3L", identity: sealedC3LIdentity, parent: sealedC3AIdentity.commit,
		subject: sealedC3LIdentity.subject, scopeDigest: requiredC3LDigest,
		claimLabels: c3lClaimLabels, claimTypes: c3lClaimTypes, validateNote: validateSealedC3LNote,
	}),
	Object.freeze({
		unit: "C3A", identity: sealedC3AIdentity, parent: sealedC3PBIdentity.commit,
		subject: "fix: align pre-C3 architecture and runtime bounds",
		scopeDigest: "sha256:cc9bb98ceeb2f281336d3e235377d3cdb8920664f833baea830a84b847621188",
		claimLabels: c3aClaimLabels, claimTypes: c3aClaimTypes, validateNote: validateSealedC3ANote,
	}),
]);

function expectedC3PredecessorRecord(authority) {
	return {
		unit: authority.unit,
		commit: authority.identity.commit,
		tree: authority.identity.tree,
		subject: authority.subject,
		parent: authority.parent,
		scope_digest: authority.scopeDigest,
		note_ref: "refs/notes/didrun",
		note_type: "blob",
		note_blob: authority.identity.noteBlob,
		note_body_sha256: authority.identity.noteBodySHA256,
		claim_count: authority.claimLabels.length,
		event_coverage: { complete: authority.claimLabels.length, total_events: authority.claimLabels.length },
		secrets_override: true,
		claims: authority.claimLabels.map((label, index) => ({
			index, label, type: authority.claimTypes[index], grade: "TREE-EXACT",
		})),
	};
}

function expectedC3PredecessorManifest() {
	return {
		schema_version: "p07b-c-c3-predecessors/v3",
		unit: "C3",
		source_parent: expectedC3PredecessorRecord(c3PredecessorAuthorities[0]),
		ancestry: c3PredecessorAuthorities.slice(1).map(expectedC3PredecessorRecord),
	};
}

function expectedC3PredecessorManifestBytes() {
	return Buffer.from(`${JSON.stringify(expectedC3PredecessorManifest(), null, 2)}\n`, "utf8");
}

function validateSealedNote(note, identity, claimLabels, claimTypes, expectedArgv, hermeticPrefix) {
	const errors = [];
	const rootKeys = ["claims", "commit", "coverage", "secrets_override", "tree", "version"];
	if (!note || typeof note !== "object" || Array.isArray(note) ||
		!isDeepStrictEqual(Object.keys(note).sort(), rootKeys)) return ["note root field roster"];
	if (note.version !== 1) errors.push("note version");
	if (note.commit !== identity.commit) errors.push("note commit");
	if (note.tree !== identity.tree) errors.push("note tree");
	if (note.secrets_override !== true) errors.push("note redacted seal disclosure");
	if (!Array.isArray(note.claims) || note.claims.length !== claimLabels.length) {
		errors.push("note claim roster length");
	} else {
		for (let index = 0; index < claimLabels.length; index += 1) {
			const recorded = note.claims[index];
			const claim = recorded?.claim;
			const expectedTail = expectedArgv[index].slice(hermeticPrefix.length);
			if (!recorded || typeof recorded !== "object" || Array.isArray(recorded) ||
				!isDeepStrictEqual(Object.keys(recorded).sort(), ["claim", "delta", "exit_code", "grade", "reason", "supporting_event_index"]) ||
				!claim || typeof claim !== "object" || Array.isArray(claim) ||
				!isDeepStrictEqual(Object.keys(claim).sort(), ["argv_preview", "ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) ||
				claim.label !== claimLabels[index] || claim.ctype !== claimTypes[index] ||
				claim.declared_at_index !== index || !isDeepStrictEqual(claim.event_indices, [index]) ||
				!isDeepStrictEqual(claim.pathspecs, []) || !Array.isArray(claim.argv_preview) ||
				claim.argv_preview.length === 0 || !claim.argv_preview.every((part) => typeof part === "string") ||
				!isDeepStrictEqual(claim.argv_preview.slice(-expectedTail.length), expectedTail) ||
				recorded.supporting_event_index !== index || recorded.grade !== "tree-exact" || recorded.exit_code !== 0 ||
				recorded.reason !== "self-stable command ran against the sealed tree" || !isDeepStrictEqual(recorded.delta, [])) {
				errors.push(`note claim ${index + 1}`);
			}
		}
	}
	if (!note.coverage || typeof note.coverage !== "object" || Array.isArray(note.coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage).sort(), ["by_coverage", "total_events"]) ||
		note.coverage.total_events !== claimLabels.length ||
		!note.coverage.by_coverage || typeof note.coverage.by_coverage !== "object" || Array.isArray(note.coverage.by_coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage.by_coverage), ["complete"]) ||
		note.coverage.by_coverage.complete !== claimLabels.length) {
		errors.push("note exact event coverage");
	}
	return errors;
}

function validateSealedC3ANote(note, identity = sealedC3AIdentity) {
	return validateSealedNote(note, identity, c3aClaimLabels, c3aClaimTypes, c3aExpectedClaimArgv, c3aHermeticArgvPrefix);
}

function validateSealedC3LNote(note, identity = sealedC3LIdentity) {
	return validateSealedNote(note, identity, c3lClaimLabels, c3lClaimTypes, c3lExpectedClaimArgv, c3lHermeticArgvPrefix);
}

function validateSealedC3FNote(note, identity = sealedC3FIdentity) {
	return validateSealedNote(note, identity, c3fClaimLabels, c3fClaimTypes, c3fExpectedClaimArgv, c3fHermeticArgvPrefix);
}

function validateSealedC3SNote(note, identity = sealedC3SIdentity) {
	return validateSealedNote(note, identity, c3sClaimLabels, c3sClaimTypes, c3sExpectedClaimArgv, c3sHermeticArgvPrefix);
}

function runHermeticPresealSelfTest(phase, expectedArgv, prefix) {
	requireHermeticCommandPlan(`${phase} self-test`, expectedArgv, prefix);
	const environment = Object.fromEntries(parseHermeticArgvPrefix(`${phase} self-test`, prefix));
	requireEffectiveHermeticContext(`${phase} self-test`, prefix, repositoryRoot, environment);
	const event = { argv: [...expectedArgv[0]], cwd: repositoryRoot, env_fingerprint: "0123456789abcdef" };
	requireBoundPresealEventContext(`${phase} self-test`, event, expectedArgv[0]);
	let rejected = 0;
	const requireRejected = (name, run) => {
		let message = "";
		try { run(); } catch (error) { message = String(error.message); }
		if (!message.includes(`P07B-C ${phase} self-test`)) {
			throw new Error(`P07B-C ${phase} hermetic preseal self-test false negative: ${name} (${message || "accepted"})`);
		}
		rejected += 1;
	};
	requireRejected("cwd drift", () => requireBoundPresealEventContext(
		`${phase} self-test`, { ...event, cwd: resolve(repositoryRoot, "docs") }, expectedArgv[0], event.env_fingerprint,
	));
	requireRejected("wrapper fingerprint drift", () => requireBoundPresealEventContext(
		`${phase} self-test`, { ...event, env_fingerprint: "fedcba9876543210" }, expectedArgv[0], event.env_fingerprint,
	));
	requireRejected("removed hermetic prefix", () => requireBoundPresealEventContext(
		`${phase} self-test`, { ...event, argv: event.argv.slice(prefix.length) }, expectedArgv[0], event.env_fingerprint,
	));
	requireRejected("one command lacks common prefix", () => requireHermeticCommandPlan(
		`${phase} self-test`, [...expectedArgv.slice(0, -1), expectedArgv.at(-1).slice(prefix.length)], prefix,
	));
	for (const name of Object.keys(environment)) {
		requireRejected(`${name} drift`, () => requireEffectiveHermeticContext(
			`${phase} self-test`, prefix, repositoryRoot, { ...environment, [name]: `${environment[name]}hostile` },
		));
	}
	return rejected;
}

async function requirePrivateFinalRunDirectories(phase, paths) {
	for (const path of paths) {
		const stat = await lstat(path);
		const resolved = await realpath(path);
		if (!stat.isDirectory() || stat.isSymbolicLink() || (stat.mode & 0o777) !== 0o700 ||
			(typeof process.getuid === "function" && stat.uid !== process.getuid()) || resolved !== path) {
			throw new Error(`P07B-C ${phase} final run directory is not exact private authority: ${path}`);
		}
	}
}

function requireSealedPredecessorSnapshot(phase, predecessor, snapshot, identity, validateNote) {
	const errors = [];
	if (snapshot.headCommit !== identity.commit) errors.push("HEAD commit");
	if (snapshot.headTree !== identity.tree) errors.push("HEAD tree");
	if (identity.parent !== undefined && !isDeepStrictEqual(snapshot.headParents, [identity.parent])) errors.push("HEAD parent");
	if (identity.subject !== undefined && snapshot.headSubject !== identity.subject) errors.push("HEAD subject");
	if (snapshot.noteBlob !== identity.noteBlob) errors.push("didrun note object identity");
	if (identity.noteType !== undefined && snapshot.noteType !== identity.noteType) errors.push("didrun note object type");
	if (typeof snapshot.noteBody !== "string" ||
		createHash("sha256").update(snapshot.noteBody, "utf8").digest("hex") !== identity.noteBodySHA256) {
		errors.push("didrun note body digest");
	}
	let note;
	try {
		note = JSON.parse(snapshot.noteBody);
	} catch (error) {
		errors.push(`didrun note JSON (${error.message})`);
	}
	if (note !== undefined) errors.push(...validateNote(note, identity));
	if (errors.length > 0) throw new Error(`P07B-C ${phase} sealed ${predecessor} predecessor mismatch: ${errors.join(", ")}`);
}

function requireSealedC3APredecessorSnapshot(phase, snapshot, identity = sealedC3AIdentity) {
	requireSealedPredecessorSnapshot(phase, "C3A", snapshot, identity, validateSealedC3ANote);
}

function requireSealedC3LPredecessorSnapshot(phase, snapshot, identity = sealedC3LIdentity) {
	requireSealedPredecessorSnapshot(phase, "C3L", snapshot, identity, validateSealedC3LNote);
}

function requireSealedC3FPredecessorSnapshot(phase, snapshot, identity = sealedC3FIdentity) {
	requireSealedPredecessorSnapshot(phase, "C3F", snapshot, identity, validateSealedC3FNote);
}

function c3aPredecessorSelfTestFixture() {
	const note = {
		claims: c3aClaimLabels.map((label, index) => ({
			claim: {
				argv_preview: [...c3aExpectedClaimArgv[index]], ctype: c3aClaimTypes[index], declared_at_index: index,
				event_indices: [index], label, pathspecs: [],
			},
			delta: [], exit_code: 0, grade: "tree-exact",
			reason: "self-stable command ran against the sealed tree", supporting_event_index: index,
		})),
		commit: sealedC3AIdentity.commit,
		coverage: { by_coverage: { complete: c3aClaimLabels.length }, total_events: c3aClaimLabels.length },
		secrets_override: true,
		tree: sealedC3AIdentity.tree,
		version: 1,
	};
	const noteBody = `${JSON.stringify(note)}\n`;
	const identity = {
		...sealedC3AIdentity,
		noteBlob: "a".repeat(40),
		noteBodySHA256: createHash("sha256").update(noteBody, "utf8").digest("hex"),
	};
	return { identity, note, snapshot: {
		headCommit: identity.commit, headTree: identity.tree, noteBlob: identity.noteBlob, noteBody,
	} };
}

function c3lPredecessorSelfTestFixture() {
	const note = {
		claims: c3lClaimLabels.map((label, index) => ({
			claim: {
				argv_preview: [...c3lExpectedClaimArgv[index]], ctype: c3lClaimTypes[index], declared_at_index: index,
				event_indices: [index], label, pathspecs: [],
			},
			delta: [], exit_code: 0, grade: "tree-exact",
			reason: "self-stable command ran against the sealed tree", supporting_event_index: index,
		})),
		commit: sealedC3LIdentity.commit,
		coverage: { by_coverage: { complete: c3lClaimLabels.length }, total_events: c3lClaimLabels.length },
		secrets_override: true,
		tree: sealedC3LIdentity.tree,
		version: 1,
	};
	const noteBody = `${JSON.stringify(note)}\n`;
	const identity = {
		...sealedC3LIdentity,
		noteBlob: "c".repeat(40),
		noteBodySHA256: createHash("sha256").update(noteBody, "utf8").digest("hex"),
	};
	return { identity, note, snapshot: {
		headCommit: identity.commit, headTree: identity.tree, headParents: [identity.parent], headSubject: identity.subject,
		noteBlob: identity.noteBlob, noteType: identity.noteType, noteBody,
	} };
}

function c3fPredecessorSelfTestFixture() {
	const note = {
		claims: c3fClaimLabels.map((label, index) => ({
			claim: {
				argv_preview: [...c3fExpectedClaimArgv[index]], ctype: c3fClaimTypes[index], declared_at_index: index,
				event_indices: [index], label, pathspecs: [],
			},
			delta: [], exit_code: 0, grade: "tree-exact",
			reason: "self-stable command ran against the sealed tree", supporting_event_index: index,
		})),
		commit: sealedC3FIdentity.commit,
		coverage: { by_coverage: { complete: c3fClaimLabels.length }, total_events: c3fClaimLabels.length },
		secrets_override: true,
		tree: sealedC3FIdentity.tree,
		version: 1,
	};
	const noteBody = `${JSON.stringify(note)}\n`;
	const identity = {
		...sealedC3FIdentity,
		noteBlob: "e".repeat(40),
		noteBodySHA256: createHash("sha256").update(noteBody, "utf8").digest("hex"),
	};
	return { identity, note, snapshot: {
		headCommit: identity.commit, headTree: identity.tree, headParents: [identity.parent], headSubject: identity.subject,
		noteBlob: identity.noteBlob, noteType: identity.noteType, noteBody,
	} };
}

function runC3LHermeticPresealSelfTest() {
	requireHermeticCommandPlan("C3L self-test", c3lExpectedClaimArgv, c3lHermeticArgvPrefix);
	const environment = Object.fromEntries(parseHermeticArgvPrefix("C3L self-test", c3lHermeticArgvPrefix));
	requireEffectiveHermeticContext("C3L self-test", c3lHermeticArgvPrefix, repositoryRoot, environment);
	const event = {
		argv: [...c3lExpectedClaimArgv[0]], cwd: repositoryRoot, env_fingerprint: "0123456789abcdef",
	};
	requireBoundPresealEventContext("C3L self-test", event, c3lExpectedClaimArgv[0]);
	let rejected = 0;
	const requireRejected = (name, run) => {
		let message = "";
		try { run(); } catch (error) { message = String(error.message); }
		if (!message.includes("P07B-C C3L self-test")) {
			throw new Error(`P07B-C C3L hermetic preseal self-test false negative: ${name} (${message || "accepted"})`);
		}
		rejected += 1;
	};
	requireRejected("cwd drift", () => requireBoundPresealEventContext(
		"C3L self-test", { ...event, cwd: resolve(repositoryRoot, "docs") }, c3lExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("malformed wrapper fingerprint", () => requireBoundPresealEventContext(
		"C3L self-test", { ...event, env_fingerprint: "not-a-digest" }, c3lExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("wrapper fingerprint drift", () => requireBoundPresealEventContext(
		"C3L self-test", { ...event, env_fingerprint: "fedcba9876543210" }, c3lExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("removed hermetic prefix", () => requireBoundPresealEventContext(
		"C3L self-test", { ...event, argv: event.argv.slice(c3lHermeticArgvPrefix.length) }, c3lExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("one command lacks common prefix", () => requireHermeticCommandPlan(
		"C3L self-test", [...c3lExpectedClaimArgv.slice(0, -1), c3lExpectedClaimArgv.at(-1).slice(c3lHermeticArgvPrefix.length)], c3lHermeticArgvPrefix,
	));
	for (const name of ["NODE_OPTIONS", "NODE_PATH", "PATH", "COUNTERSHAPE_NODE", "COUNTERSHAPE_GIT", "COUNTERSHAPE_SH"]) {
		requireRejected(`${name} drift`, () => requireEffectiveHermeticContext(
			"C3L self-test", c3lHermeticArgvPrefix, repositoryRoot, { ...environment, [name]: `${environment[name]}hostile` },
		));
	}
	const predecessor = c3aPredecessorSelfTestFixture();
	requireSealedC3APredecessorSnapshot("C3L self-test", predecessor.snapshot, predecessor.identity);
	const rejectSnapshot = (name, mutateSnapshot, identity = predecessor.identity) => requireRejected(name, () => {
		const snapshot = structuredClone(predecessor.snapshot);
		mutateSnapshot(snapshot);
		requireSealedC3APredecessorSnapshot("C3L self-test", snapshot, identity);
	});
	rejectSnapshot("sealed predecessor HEAD commit drift", (snapshot) => { snapshot.headCommit = "b".repeat(40); });
	rejectSnapshot("sealed predecessor HEAD tree drift", (snapshot) => { snapshot.headTree = "b".repeat(40); });
	rejectSnapshot("sealed predecessor note object drift", (snapshot) => { snapshot.noteBlob = "b".repeat(40); });
	rejectSnapshot("sealed predecessor note body drift", (snapshot) => { snapshot.noteBody += " "; });
	const rejectNote = (name, mutate) => {
		const note = structuredClone(predecessor.note);
		mutate(note);
		const noteBody = `${JSON.stringify(note)}\n`;
		rejectSnapshot(name, (snapshot) => { snapshot.noteBody = noteBody; }, {
			...predecessor.identity,
			noteBodySHA256: createHash("sha256").update(noteBody, "utf8").digest("hex"),
		});
	};
	rejectNote("sealed predecessor note root drift", (note) => { note.extra = true; });
	rejectNote("sealed predecessor note version drift", (note) => { note.version = 2; });
	rejectNote("sealed predecessor note identity drift", (note) => { note.commit = "b".repeat(40); });
	rejectNote("sealed predecessor note tree drift", (note) => { note.tree = "b".repeat(40); });
	rejectNote("sealed predecessor note override drift", (note) => { note.secrets_override = false; });
	rejectNote("sealed predecessor note claim roster drift", (note) => { note.claims.pop(); });
	rejectNote("sealed predecessor note claim order drift", (note) => { [note.claims[0], note.claims[1]] = [note.claims[1], note.claims[0]]; });
	rejectNote("sealed predecessor note claim label drift", (note) => { note.claims[0].claim.label += " altered"; });
	rejectNote("sealed predecessor note claim type drift", (note) => { note.claims[0].claim.ctype = "command-succeeded"; });
	rejectNote("sealed predecessor note claim semantics drift", (note) => { note.claims[0].claim.event_indices = [1]; });
	rejectNote("sealed predecessor note declared index drift", (note) => { note.claims[0].claim.declared_at_index = 1; });
	rejectNote("sealed predecessor note supporting index drift", (note) => { note.claims[0].supporting_event_index = 1; });
	rejectNote("sealed predecessor note pathspec drift", (note) => { note.claims[0].claim.pathspecs = ["."]; });
	rejectNote("sealed predecessor note argv drift", (note) => { note.claims[0].claim.argv_preview = []; });
	rejectNote("sealed predecessor note argv tail drift", (note) => {
		note.claims[0].claim.argv_preview[note.claims[0].claim.argv_preview.length - 1] += "-altered";
	});
	rejectNote("sealed predecessor note grade drift", (note) => { note.claims[0].grade = "recorded-exact"; });
	rejectNote("sealed predecessor note exit drift", (note) => { note.claims[0].exit_code = 1; });
	rejectNote("sealed predecessor note reason drift", (note) => { note.claims[0].reason = "changed"; });
	rejectNote("sealed predecessor note delta drift", (note) => { note.claims[0].delta = ["changed"]; });
	rejectNote("sealed predecessor note claim field drift", (note) => { note.claims[0].claim.extra = true; });
	rejectNote("sealed predecessor note coverage total drift", (note) => { note.coverage.total_events -= 1; });
	rejectNote("sealed predecessor note coverage complete drift", (note) => { note.coverage.by_coverage.complete -= 1; });
	rejectNote("sealed predecessor note coverage drift", (note) => { note.coverage.by_coverage.partial = 1; });
	const malformedBody = "{not-json}\n";
	rejectSnapshot("sealed predecessor malformed note JSON", (snapshot) => { snapshot.noteBody = malformedBody; }, {
		...predecessor.identity,
		noteBodySHA256: createHash("sha256").update(malformedBody, "utf8").digest("hex"),
	});
	return rejected;
}

async function requirePrivateC3LFinalRunDirectories() {
	for (const path of c3lFinalRunDirectories) {
		const stat = await lstat(path);
		const resolved = await realpath(path);
		if (!stat.isDirectory() || stat.isSymbolicLink() || (stat.mode & 0o777) !== 0o700 ||
			(typeof process.getuid === "function" && stat.uid !== process.getuid()) || resolved !== path) {
			throw new Error(`P07B-C C3L final run directory is not exact private authority: ${path}`);
		}
	}
}

function runC3FHermeticPresealSelfTest() {
	requireHermeticCommandPlan("C3F self-test", c3fExpectedClaimArgv, c3fHermeticArgvPrefix);
	const environment = Object.fromEntries(parseHermeticArgvPrefix("C3F self-test", c3fHermeticArgvPrefix));
	requireEffectiveHermeticContext("C3F self-test", c3fHermeticArgvPrefix, repositoryRoot, environment);
	const event = {
		argv: [...c3fExpectedClaimArgv[0]], cwd: repositoryRoot, env_fingerprint: "0123456789abcdef",
	};
	requireBoundPresealEventContext("C3F self-test", event, c3fExpectedClaimArgv[0]);
	let rejected = 0;
	const requireRejected = (name, run) => {
		let message = "";
		try { run(); } catch (error) { message = String(error.message); }
		if (!message.includes("P07B-C C3F self-test")) {
			throw new Error(`P07B-C C3F hermetic preseal self-test false negative: ${name} (${message || "accepted"})`);
		}
		rejected += 1;
	};
	requireRejected("cwd drift", () => requireBoundPresealEventContext(
		"C3F self-test", { ...event, cwd: resolve(repositoryRoot, "docs") }, c3fExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("malformed wrapper fingerprint", () => requireBoundPresealEventContext(
		"C3F self-test", { ...event, env_fingerprint: "not-a-digest" }, c3fExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("wrapper fingerprint drift", () => requireBoundPresealEventContext(
		"C3F self-test", { ...event, env_fingerprint: "fedcba9876543210" }, c3fExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("removed hermetic prefix", () => requireBoundPresealEventContext(
		"C3F self-test", { ...event, argv: event.argv.slice(c3fHermeticArgvPrefix.length) }, c3fExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("one command lacks common prefix", () => requireHermeticCommandPlan(
		"C3F self-test", [...c3fExpectedClaimArgv.slice(0, -1), c3fExpectedClaimArgv.at(-1).slice(c3fHermeticArgvPrefix.length)], c3fHermeticArgvPrefix,
	));
	for (const name of Object.keys(environment)) {
		requireRejected(`${name} drift`, () => requireEffectiveHermeticContext(
			"C3F self-test", c3fHermeticArgvPrefix, repositoryRoot, { ...environment, [name]: `${environment[name]}hostile` },
		));
	}
	const predecessor = c3lPredecessorSelfTestFixture();
	requireSealedC3LPredecessorSnapshot("C3F self-test", predecessor.snapshot, predecessor.identity);
	const rejectSnapshot = (name, mutateSnapshot, identity = predecessor.identity) => requireRejected(name, () => {
		const snapshot = structuredClone(predecessor.snapshot);
		mutateSnapshot(snapshot);
		requireSealedC3LPredecessorSnapshot("C3F self-test", snapshot, identity);
	});
	rejectSnapshot("sealed predecessor HEAD commit drift", (snapshot) => { snapshot.headCommit = "d".repeat(40); });
	rejectSnapshot("sealed predecessor HEAD tree drift", (snapshot) => { snapshot.headTree = "d".repeat(40); });
	rejectSnapshot("sealed predecessor HEAD parent drift", (snapshot) => { snapshot.headParents = ["d".repeat(40)]; });
	rejectSnapshot("sealed predecessor merge-parent drift", (snapshot) => { snapshot.headParents.push("d".repeat(40)); });
	rejectSnapshot("sealed predecessor subject drift", (snapshot) => { snapshot.headSubject += " altered"; });
	rejectSnapshot("sealed predecessor subject trailing space", (snapshot) => { snapshot.headSubject += " "; });
	rejectSnapshot("sealed predecessor note object drift", (snapshot) => { snapshot.noteBlob = "d".repeat(40); });
	rejectSnapshot("sealed predecessor note object type drift", (snapshot) => { snapshot.noteType = "tree"; });
	rejectSnapshot("sealed predecessor note body drift", (snapshot) => { snapshot.noteBody += " "; });
	const rejectNote = (name, mutate) => {
		const note = structuredClone(predecessor.note);
		mutate(note);
		const noteBody = `${JSON.stringify(note)}\n`;
		rejectSnapshot(name, (snapshot) => { snapshot.noteBody = noteBody; }, {
			...predecessor.identity,
			noteBodySHA256: createHash("sha256").update(noteBody, "utf8").digest("hex"),
		});
	};
	rejectNote("sealed predecessor note root drift", (note) => { note.extra = true; });
	rejectNote("sealed predecessor note version drift", (note) => { note.version = 2; });
	rejectNote("sealed predecessor note identity drift", (note) => { note.commit = "d".repeat(40); });
	rejectNote("sealed predecessor note tree drift", (note) => { note.tree = "d".repeat(40); });
	rejectNote("sealed predecessor note override drift", (note) => { note.secrets_override = false; });
	rejectNote("sealed predecessor note claim roster drift", (note) => { note.claims.pop(); });
	rejectNote("sealed predecessor note claim order drift", (note) => { [note.claims[0], note.claims[1]] = [note.claims[1], note.claims[0]]; });
	rejectNote("sealed predecessor note claim label drift", (note) => { note.claims[0].claim.label += " altered"; });
	rejectNote("sealed predecessor note claim type drift", (note) => { note.claims[0].claim.ctype = "command-succeeded"; });
	rejectNote("sealed predecessor note claim semantics drift", (note) => { note.claims[0].claim.event_indices = [1]; });
	rejectNote("sealed predecessor note declared index drift", (note) => { note.claims[0].claim.declared_at_index = 1; });
	rejectNote("sealed predecessor note supporting index drift", (note) => { note.claims[0].supporting_event_index = 1; });
	rejectNote("sealed predecessor note pathspec drift", (note) => { note.claims[0].claim.pathspecs = ["."]; });
	rejectNote("sealed predecessor note argv drift", (note) => { note.claims[0].claim.argv_preview = []; });
	rejectNote("sealed predecessor note argv tail drift", (note) => {
		note.claims[0].claim.argv_preview[note.claims[0].claim.argv_preview.length - 1] += "-altered";
	});
	rejectNote("sealed predecessor note grade drift", (note) => { note.claims[0].grade = "scope-exact"; });
	rejectNote("sealed predecessor note exit drift", (note) => { note.claims[0].exit_code = 1; });
	rejectNote("sealed predecessor note reason drift", (note) => { note.claims[0].reason = "changed"; });
	rejectNote("sealed predecessor note delta drift", (note) => { note.claims[0].delta = ["changed"]; });
	rejectNote("sealed predecessor note claim field drift", (note) => { note.claims[0].claim.extra = true; });
	rejectNote("sealed predecessor note coverage total drift", (note) => { note.coverage.total_events -= 1; });
	rejectNote("sealed predecessor note coverage complete drift", (note) => { note.coverage.by_coverage.complete -= 1; });
	rejectNote("sealed predecessor note coverage drift", (note) => { note.coverage.by_coverage.partial = 1; });
	const malformedBody = "{not-json}\n";
	rejectSnapshot("sealed predecessor malformed note JSON", (snapshot) => { snapshot.noteBody = malformedBody; }, {
		...predecessor.identity,
		noteBodySHA256: createHash("sha256").update(malformedBody, "utf8").digest("hex"),
	});
	return rejected;
}

async function requirePrivateC3FFinalRunDirectories() {
	for (const path of c3fFinalRunDirectories) {
		const stat = await lstat(path);
		const resolved = await realpath(path);
		if (!stat.isDirectory() || stat.isSymbolicLink() || (stat.mode & 0o777) !== 0o700 ||
			(typeof process.getuid === "function" && stat.uid !== process.getuid()) || resolved !== path) {
			throw new Error(`P07B-C C3F final run directory is not exact private authority: ${path}`);
		}
	}
}

function runC3SHermeticPresealSelfTest() {
	requireHermeticCommandPlan("C3S self-test", c3sExpectedClaimArgv, c3sHermeticArgvPrefix);
	const environment = Object.fromEntries(parseHermeticArgvPrefix("C3S self-test", c3sHermeticArgvPrefix));
	requireEffectiveHermeticContext("C3S self-test", c3sHermeticArgvPrefix, repositoryRoot, environment);
	const event = {
		argv: [...c3sExpectedClaimArgv[0]], cwd: repositoryRoot, env_fingerprint: "0123456789abcdef",
	};
	requireBoundPresealEventContext("C3S self-test", event, c3sExpectedClaimArgv[0]);
	let rejected = 0;
	const requireRejected = (name, run) => {
		let message = "";
		try { run(); } catch (error) { message = String(error.message); }
		if (!message.includes("P07B-C C3S self-test")) {
			throw new Error(`P07B-C C3S hermetic preseal self-test false negative: ${name} (${message || "accepted"})`);
		}
		rejected += 1;
	};
	requireRejected("cwd drift", () => requireBoundPresealEventContext(
		"C3S self-test", { ...event, cwd: resolve(repositoryRoot, "docs") }, c3sExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("malformed wrapper fingerprint", () => requireBoundPresealEventContext(
		"C3S self-test", { ...event, env_fingerprint: "not-a-digest" }, c3sExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("wrapper fingerprint drift", () => requireBoundPresealEventContext(
		"C3S self-test", { ...event, env_fingerprint: "fedcba9876543210" }, c3sExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("removed hermetic prefix", () => requireBoundPresealEventContext(
		"C3S self-test", { ...event, argv: event.argv.slice(c3sHermeticArgvPrefix.length) }, c3sExpectedClaimArgv[0], event.env_fingerprint,
	));
	requireRejected("one command lacks common prefix", () => requireHermeticCommandPlan(
		"C3S self-test", [...c3sExpectedClaimArgv.slice(0, -1), c3sExpectedClaimArgv.at(-1).slice(c3sHermeticArgvPrefix.length)], c3sHermeticArgvPrefix,
	));
	for (const name of Object.keys(environment)) {
		requireRejected(`${name} drift`, () => requireEffectiveHermeticContext(
			"C3S self-test", c3sHermeticArgvPrefix, repositoryRoot, { ...environment, [name]: `${environment[name]}hostile` },
		));
	}
	const predecessor = c3fPredecessorSelfTestFixture();
	requireSealedC3FPredecessorSnapshot("C3S self-test", predecessor.snapshot, predecessor.identity);
	const rejectSnapshot = (name, mutateSnapshot, identity = predecessor.identity) => requireRejected(name, () => {
		const snapshot = structuredClone(predecessor.snapshot);
		mutateSnapshot(snapshot);
		requireSealedC3FPredecessorSnapshot("C3S self-test", snapshot, identity);
	});
	rejectSnapshot("sealed predecessor HEAD commit drift", (snapshot) => { snapshot.headCommit = "f".repeat(40); });
	rejectSnapshot("sealed predecessor HEAD tree drift", (snapshot) => { snapshot.headTree = "f".repeat(40); });
	rejectSnapshot("sealed predecessor HEAD parent drift", (snapshot) => { snapshot.headParents = ["f".repeat(40)]; });
	rejectSnapshot("sealed predecessor merge-parent drift", (snapshot) => { snapshot.headParents.push("f".repeat(40)); });
	rejectSnapshot("sealed predecessor subject drift", (snapshot) => { snapshot.headSubject += " altered"; });
	rejectSnapshot("sealed predecessor note object drift", (snapshot) => { snapshot.noteBlob = "f".repeat(40); });
	rejectSnapshot("sealed predecessor note object type drift", (snapshot) => { snapshot.noteType = "tree"; });
	rejectSnapshot("sealed predecessor note body drift", (snapshot) => { snapshot.noteBody += " "; });
	const rejectNote = (name, mutate) => {
		const note = structuredClone(predecessor.note);
		mutate(note);
		const noteBody = `${JSON.stringify(note)}\n`;
		rejectSnapshot(name, (snapshot) => { snapshot.noteBody = noteBody; }, {
			...predecessor.identity,
			noteBodySHA256: createHash("sha256").update(noteBody, "utf8").digest("hex"),
		});
	};
	rejectNote("sealed predecessor note root drift", (note) => { note.extra = true; });
	rejectNote("sealed predecessor note version drift", (note) => { note.version = 2; });
	rejectNote("sealed predecessor note identity drift", (note) => { note.commit = "f".repeat(40); });
	rejectNote("sealed predecessor note tree drift", (note) => { note.tree = "f".repeat(40); });
	rejectNote("sealed predecessor note override drift", (note) => { note.secrets_override = false; });
	rejectNote("sealed predecessor note claim roster drift", (note) => { note.claims.pop(); });
	rejectNote("sealed predecessor note claim order drift", (note) => { [note.claims[0], note.claims[1]] = [note.claims[1], note.claims[0]]; });
	rejectNote("sealed predecessor note claim label drift", (note) => { note.claims[0].claim.label += " altered"; });
	rejectNote("sealed predecessor note claim type drift", (note) => { note.claims[0].claim.ctype = "command-succeeded"; });
	rejectNote("sealed predecessor note claim semantics drift", (note) => { note.claims[0].claim.event_indices = [1]; });
	rejectNote("sealed predecessor note declared index drift", (note) => { note.claims[0].claim.declared_at_index = 1; });
	rejectNote("sealed predecessor note supporting index drift", (note) => { note.claims[0].supporting_event_index = 1; });
	rejectNote("sealed predecessor note pathspec drift", (note) => { note.claims[0].claim.pathspecs = ["."]; });
	rejectNote("sealed predecessor note argv drift", (note) => { note.claims[0].claim.argv_preview = []; });
	rejectNote("sealed predecessor note argv tail drift", (note) => {
		note.claims[0].claim.argv_preview[note.claims[0].claim.argv_preview.length - 1] += "-altered";
	});
	rejectNote("sealed predecessor note grade drift", (note) => { note.claims[0].grade = "scope-exact"; });
	rejectNote("sealed predecessor note exit drift", (note) => { note.claims[0].exit_code = 1; });
	rejectNote("sealed predecessor note reason drift", (note) => { note.claims[0].reason = "changed"; });
	rejectNote("sealed predecessor note delta drift", (note) => { note.claims[0].delta = ["changed"]; });
	rejectNote("sealed predecessor note claim field drift", (note) => { note.claims[0].claim.extra = true; });
	rejectNote("sealed predecessor note coverage total drift", (note) => { note.coverage.total_events -= 1; });
	rejectNote("sealed predecessor note coverage complete drift", (note) => { note.coverage.by_coverage.complete -= 1; });
	rejectNote("sealed predecessor note coverage drift", (note) => { note.coverage.by_coverage.partial = 1; });
	const malformedBody = "{not-json}\n";
	rejectSnapshot("sealed predecessor malformed note JSON", (snapshot) => { snapshot.noteBody = malformedBody; }, {
		...predecessor.identity,
		noteBodySHA256: createHash("sha256").update(malformedBody, "utf8").digest("hex"),
	});
	return rejected;
}

async function requirePrivateC3SFinalRunDirectories() {
	for (const path of c3sFinalRunDirectories) {
		const stat = await lstat(path);
		const resolved = await realpath(path);
		if (!stat.isDirectory() || stat.isSymbolicLink() || (stat.mode & 0o777) !== 0o700 ||
			(typeof process.getuid === "function" && stat.uid !== process.getuid()) || resolved !== path) {
			throw new Error(`P07B-C C3S final run directory is not exact private authority: ${path}`);
		}
	}
}
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
const requiredC3RPaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/VERIFICATION.md",
	"docs/status/P07B-C-C3R-NOTE-PREVIEW-MAINTENANCE.md",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
]);
const requiredC3RDigest = "sha256:fe14d9f9985771cce0525edfca47c001b5912dacc4c2d6fef182458d3b44f58e";
const requiredC3RRosterText = requiredC3RPaths
	.map((path, index) => `${index === requiredC3RPaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC3RDeclaration = `C3R owns exactly these ${requiredC3RPaths.length} paths: ${requiredC3RRosterText}. Their sorted-newline roster digest is \`${requiredC3RDigest}\`.`;
const c3rSourceSubject = "fix: validate sealed C3 note previews";
const c3rStatusBoundaryLine = "- **Boundary:** active pre-seal `SOURCE_FULL` checker maintenance after sealed C3 and before the unchanged frozen C3B receipt unit. Classification: `DEFECT_REPAIR`.";
const c3rStatusParentLine = `- **Parent:** sealed C3 commit \`${sealedC3Identity.commit}\`, tree \`${sealedC3Identity.tree}\`, is note-present with note blob \`${sealedC3Identity.noteBlob}\`, note-body SHA-256 \`${sealedC3Identity.noteBodySHA256}\`, \`18/18 claims recorded-exact\`, strict exit \`0\`, and a logged redacted-export override.`;
const c3rStatusUnreceiptedLine = "Every intended C3R grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3R itself.";
const requiredC3QPaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/VERIFICATION.md",
	"docs/status/P07B-C-C3Q-SELF-RECEIPT-MAINTENANCE.md",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
]);
const requiredC3QDigest = "sha256:f5f27e661c1ace6516aba4b9d5766f4c6ab50b6602dbcb101bff9b6a4ebfe6fc";
const requiredC3QRosterText = requiredC3QPaths
	.map((path, index) => `${index === requiredC3QPaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC3QDeclaration = `C3Q owns exactly these ${requiredC3QPaths.length} paths: ${requiredC3QRosterText}. Their sorted-newline roster digest is \`${requiredC3QDigest}\`.`;
const c3qSourceSubject = "fix: close C3B external self-receipt checks";
const c3qStatusBoundaryLine = "- **Boundary:** active pre-seal `SOURCE_FULL` checker maintenance after sealed C3R and before the unchanged frozen C3B receipt unit. Classification: `DEFECT_REPAIR`.";
const c3qStatusParentLine = `- **Parent:** sealed C3R commit \`${sealedC3RIdentity.commit}\`, tree \`${sealedC3RIdentity.tree}\`, is note-present with note blob \`${sealedC3RIdentity.noteBlob}\`, note-body SHA-256 \`${sealedC3RIdentity.noteBodySHA256}\`, \`9/9 claims recorded-exact\`, strict exit \`0\`, and recorded \`secrets_override: true\`.`;
const c3qStatusUnreceiptedLine = "Every intended C3Q grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3Q itself.";
const requiredC3TPaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/VERIFICATION.md",
	"docs/status/P07B-C-C3T-PHASE-SELFTEST-MAINTENANCE.md",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
]);
const requiredC3TDigest = "sha256:496788860db6c42f5acc50de0de02be8a63bcb8e11f99fb24d59d9c0e016477e";
const requiredC3TRosterText = requiredC3TPaths
	.map((path, index) => `${index === requiredC3TPaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC3TDeclaration = `C3T owns exactly these ${requiredC3TPaths.length} paths: ${requiredC3TRosterText}. Their sorted-newline roster digest is \`${requiredC3TDigest}\`.`;
const c3tSourceSubject = "fix: make C3 receipt self-test phase-stable";
const c3tStatusBoundaryLine = "- **Boundary:** active pre-seal `SOURCE_FULL` checker maintenance after sealed C3Q and before the unchanged frozen C3B receipt unit. Classification: `DEFECT_REPAIR`.";
const c3tStatusParentLine = `- **Parent:** sealed C3Q commit \`${sealedC3QIdentity.commit}\`, tree \`${sealedC3QIdentity.tree}\`, is note-present with note blob \`${sealedC3QIdentity.noteBlob}\`, note-body SHA-256 \`${sealedC3QIdentity.noteBodySHA256}\`, \`9/9 claims recorded-exact\`, strict exit \`0\`, and recorded \`secrets_override: true\`.`;
const c3tStatusUnreceiptedLine = "Every intended C3T grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3T itself.";
const requiredC3UPaths = Object.freeze([
	"docs/HANDOFF_MODE_C.md",
	"docs/PROMPT_PACK.md",
	"docs/VERIFICATION.md",
	"docs/status/P07B-C-C3U-CROSS-PHASE-RECEIPT-FIXTURE-MAINTENANCE.md",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
]);
const requiredC3UDigest = "sha256:7af7456929b0db08abc2a31bd1cb62d84002c1a6bcdda3f4a8b4896f6b8c30cd";
const requiredC3URosterText = requiredC3UPaths
	.map((path, index) => `${index === requiredC3UPaths.length - 1 ? "and " : ""}\`${path}\``)
	.join(", ");
const requiredC3UDeclaration = `C3U owns exactly these ${requiredC3UPaths.length} paths: ${requiredC3URosterText}. Their sorted-newline roster digest is \`${requiredC3UDigest}\`.`;
const c3uSourceSubject = "fix: preserve terminal C3 receipts in legacy fixtures";
const c3uStatusBoundaryLine = "- **Boundary:** active pre-seal `SOURCE_FULL` checker maintenance after sealed C3T and before the unchanged frozen C3B receipt unit. Classification: `DEFECT_REPAIR`.";
const c3uStatusParentLine = `- **Parent:** sealed C3T commit \`${sealedC3TIdentity.commit}\`, tree \`${sealedC3TIdentity.tree}\`, is note-present with note blob \`${sealedC3TIdentity.noteBlob}\`, note-body SHA-256 \`${sealedC3TIdentity.noteBodySHA256}\`, \`9/9 claims recorded-exact\`, strict exit \`0\`, and recorded \`secrets_override: true\`.`;
const c3uStatusUnreceiptedLine = "Every intended C3U grade remains `UNRECEIPTED` until one exact staged tree runs the declared commands through a fresh didrun ledger, commits, seals, exposes a readable Git note, and passes `NO_COLOR=1 didrun verify --strict`. This status cannot receipt C3U itself.";
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
const requiredC3AddedPaths = new Set([
	"docs/status/P07B-C-C3-TARGET.md",
	"internal/contractexec/target.go",
	"internal/contractexec/target_test.go",
	"internal/gitobj/single_target.go",
	"internal/gitobj/single_target_test.go",
	"internal/hostepoch/epoch.go",
	"internal/hostepoch/epoch_darwin_test.go",
	"internal/hostepoch/epoch_test.go",
	"internal/hostepoch/public_api_test.go",
	"internal/hostepoch/source_darwin_cgo.go",
	"internal/hostepoch/source_unsupported.go",
	"internal/noderuntime/identity_darwin.go",
	"internal/noderuntime/probe_darwin.go",
	"internal/noderuntime/probe_darwin_test.go",
	"internal/noderuntime/public_api_test.go",
	"internal/noderuntime/runtime.go",
	"internal/noderuntime/runtime_darwin_test.go",
	"internal/noderuntime/runtime_test.go",
	"internal/noderuntime/unsupported.go",
	"spec/verification/p07b-c-c3-predecessors.json",
]);
const c3SourceSubject = "feat: publish exact contract execution targets";
const c3PendingStatusState = "- **State:** active pre-seal `SOURCE_FULL` source unit; every C3 grade below is `UNRECEIPTED`";
const c3ReceiptNoRecursionBoundary = "This receipt binds only the already-existing C3 source commit. The active C3B descendant cannot grade itself here and remains `UNRECEIPTED`.";
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
const c3vDispositionRows = Object.freeze([
	"| 1. Persistent content-addressed private `GOCACHE` keyed by admitted Go authority | `DECLINED_WITH_REASON` | Go's cache does not detect imported C-library changes. Countershape has real Darwin/cgo code; a Go-only key omits clang, SDK/header, libSystem, and prior writable-cache authority. `GOCACHE` therefore remains fresh below each mode-`0700` run root and is removed at finalization. |",
	"| 2. Direct `p=8` general lane plus serialized sensitive lane | `INTENTIONAL; P8_DECLINED_WITH_REASON` | C1V already replaced global package serialization with exact tiering: direct build/vet/general tests use `-p=2`, while six frozen sensitive packages run separately at `-p=1` and every nested command inherits `-p=1`. The `p=4` candidate passed Go packages but caused two permanent later AST-timeout events; `p=8` was therefore declined for this shared host. The accepted `p=2` profile passed the historical 50x/20x matrix and three cumulative ceilings. |",
	"| 3. Exclusive single-verifier lock | `INTENTIONAL` | The verifier already acquires `.countershape/verify-current/active.lock` with exclusive creation before authority admission or work. A second live/indeterminate owner fails fast. An absent PID produces review-first `VERIFY_STALE_LOCK`; automatic stale deletion remains declined because two recovery contenders can race and remove a new owner's lock. |",
	"| 4. Narrow docs-only receipt boundary | `INTENTIONAL` | `RECEIPT_RECONCILIATION` already admits only an exact empty-prefix Markdown/receipt-declaration roster with seven machine-declared claims: reconciliation, checker self-test, local snapshot match, receipt-only Go build, exact staged scope/diff, credential scan, and chain integrity. It cannot claim the cumulative suite, runtime, security, or unchanged behavior. Source units remain `SOURCE_FULL`. |",
]);
const c3vDidrunInterruptFinding = "`C3V-S6-DIDRUN-INTERRUPT-1`: during an unclaimed development run, operator SIGINT caused didrun to exit `130` with a Python `KeyboardInterrupt` traceback before it appended an event for the in-flight cumulative verifier; process inspection found no surviving verifier child. It left the mode-`0600` lock for PID `84735` in place; review proved the PID absent, no process held the file, and the owner, repository-root digest, and mode matched before one explicit didrun-wrapped recovery event removed only that lock. The interrupted command therefore has no didrun receipt and supports no claim. This is a didrun interruption-capture bug finding, not a verifier failure or a sealed-tree result; final evidence must complete normally in a fresh ledger.";
const requiredC3VText = Object.freeze({
	"docs/HANDOFF_MODE_C.md": [
		"Active C3V throughput-proposal reconciliation",
		requiredC3VDigest,
		sealedC3PIdentity.commit,
		sealedC3PIdentity.tree,
		"88e62acb3023dcd6ee51950c2ad24bbcc2ca8900",
		"C3V changes only its seven declared documentation/checker-authority paths",
		c3vDidrunInterruptFinding,
	],
	"docs/PROMPT_PACK.md": [
		"status/P07B-C-C3V-VERIFICATION-THROUGHPUT.md",
		requiredC3VDigest,
		"active no-delta maintenance",
	],
	"docs/VERIFICATION.md": [
		"## C3V verification-throughput proposal reconciliation",
		requiredC3VDeclaration,
		"DECLINED_WITH_REASON",
		"INTENTIONAL; P8_DECLINED_WITH_REASON",
		"Item 4 is `INTENTIONAL`: exact empty-prefix receipt units already use the explicit seven-claim `RECEIPT_RECONCILIATION` profile",
		"C3V owns exactly these 7 paths",
	],
	[c3vStatusPath]: [
		"# P07B-C C3V — verification-throughput proposal reconciliation",
		"active pre-seal `SOURCE_FULL` maintenance; every C3V grade below is `UNRECEIPTED`",
		requiredC3VDigest,
		sealedC3PIdentity.commit,
		sealedC3PIdentity.tree,
		...c3vDispositionRows,
		"automatic stale deletion remains declined",
		"Because C3V changes no execution setting",
		"may not claim a new qualification or a new performance result",
		"Its only behavior change is verification-scope/checker authority",
		c3vDidrunInterruptFinding,
		...Object.entries(c3vFrozenVerifierDigests).map(([path, digest]) => `\`${path}\` is \`sha256:${digest}\``),
		...c3vClaimLabels.map((label) => `\`${label}\``),
	],
});
const requiredC3MaintenanceText = Object.freeze({
	"docs/HANDOFF_MODE_C.md": [
		"C3M receipt-phase checker maintenance is the active `SOURCE_FULL` unit",
		requiredC3MaintenanceDigest,
		"725933fa3c7826a281e84b51f729736cbcfac6ec",
		"21f0e1bf46e3776e1b496a9c2783f549923c4510",
		"5eeb848c334ad3a8a44e4cf61fa298e452188ab1",
	],
	"docs/PROMPT_PACK.md": [
		"P07B-C C3M receipt-phase checker maintenance",
		requiredC3MaintenanceDigest,
		"C3PB remains unchanged",
	],
	"docs/VERIFICATION.md": [
		"## C3P receipt-phase checker maintenance",
		"current handoff after fail-closed removal of at most one exact C3P receipt block",
		"standalone LF lines",
		"residue outside that block",
		"C3V and C3M authority survives",
		requiredC3MaintenanceDeclaration,
	],
	[c3MaintenanceStatusPath]: [
		"# P07B-C C3M — C3P receipt-phase checker maintenance",
		"Classification: `DEFECT_REPAIR`",
		requiredC3MaintenanceDigest,
		"5eeb848c334ad3a8a44e4cf61fa298e452188ab1",
		"withoutC3PReceiptBlock",
		"parseC3PReceiptBlock",
		"c3pReceiptHandoffBlock",
		"readSealedC3PStatusFixture",
		"syntheticC3PAbsentPhaseFixture",
		"The checker must never reopen the C3P-era handoff.",
		"C3V and C3M authority survives",
		"explicit C3PB-active state",
		...c3MaintenanceClaimLabels.map((label) => `\`${label}\``),
	],
	"tools/check-p07b-c-plan.mjs": [
		"withoutC3PReceiptBlock",
		"parseC3PReceiptBlock",
		"c3pReceiptHandoffBlock",
		"requireSyntheticC3SActiveHandoff",
		"requireSyntheticC3ActiveHandoff",
		"readSealedC3PStatusFixture",
		"syntheticC3PAbsentPhaseFixture",
		"--verify-c3m-preseal-ledger",
	],
	"tools/check-p07b-c-unit-scope.mjs": [
		"\"C3M\"",
		"declared exact-roster SOURCE_FULL unit",
	],
});
const requiredC3AText = Object.freeze({
	"docs/ARCHITECTURE.md": [
		"C3A cumulative-admission maintenance",
		"internal/contractexec/target.go:ContractExecutionTarget",
		"128-byte runtime-version bound",
		"internal production-Go policy",
	],
	"docs/HANDOFF_MODE_C.md": [
		requiredC3ADigest,
		sealedC3PBIdentity.commit,
		sealedC3PBIdentity.tree,
		sealedC3PBIdentity.noteBlob,
		sealedC3PBIdentity.noteBodySHA256,
		"3e766715609680adaef4678fc9aeaa9e3a8bae09",
		"relocated-to-history",
		"Event eleven cannot self-prove its outer launcher",
	],
	"docs/PROMPT_PACK.md": [
		"P07B-C C3A cumulative-admission maintenance",
		requiredC3ADigest,
		"C3 remains frozen to forty paths",
	],
	"docs/VERIFICATION.md": [
		"## C3A cumulative-admission maintenance",
		requiredC3ADeclaration,
		"128-byte runtime-version ceiling",
		"C3 remains frozen to forty exact paths",
		"canonical Markdown locations",
		"cannot prove event eleven's outer `env` argv",
	],
	[c3aStatusPath]: [
		"# P07B-C C3A — cumulative-admission maintenance",
		"Classification: `DEFECT_REPAIR`",
		requiredC3ADigest,
		sealedC3PBIdentity.commit,
		sealedC3PBIdentity.tree,
		sealedC3PBIdentity.noteBlob,
		sealedC3PBIdentity.noteBodySHA256,
		"fix: align pre-C3 architecture and runtime bounds",
		"128-byte runtime-version ceiling",
		"C3 remains frozen to forty exact paths",
		"owned mode-`0700` nonsymlink directories",
		"relocated",
		...c3aClaimLabels.map((label) => `\`${label}\``),
	],
	"internal/contractexec/model/target.go": ["len(runtime.Version) > 128"],
	"internal/contractexec/model/codec_test.go": ["runtime version schema boundary", "128-byte runtime version", "129-byte constructor", "129-byte parser"],
	"tools/check-p07b-b-architecture.mjs": ["c3OfficialTargetSurface", "futureSymbolsInSource", "realpathSync"],
	"tools/check-p07b-b-architecture-selftest.mjs": ["P07B_B_SELFTEST_SYMLINK_ENTRYPOINT", "ContractExecutionTargetβ", "P07B_B_SELFTEST_SYMBOL_SCANNER"],
	"tools/check-p07b-c-plan.mjs": [
		"requiredC3APaths", "C3A_ACTIVE", "--verify-c3a-preseal-ledger", "requirePrivateC3AFinalRunDirectories",
	],
	"tools/check-p07b-c-unit-scope.mjs": ["\"C3A\"", "p07b-c-unit-paths/v15"],
});
const requiredC3LText = Object.freeze({
	"docs/HANDOFF_MODE_C.md": [
		requiredC3LDigest,
		sealedC3LIdentity.commit,
		sealedC3LIdentity.tree,
		sealedC3LIdentity.noteBlob,
		sealedC3LIdentity.noteBodySHA256,
		sealedC3AIdentity.commit,
		sealedC3AIdentity.tree,
		sealedC3AIdentity.noteBlob,
		sealedC3AIdentity.noteBodySHA256,
		"f7160bbbc09245a1026eb6396ac76282ab913192",
		"U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED",
		"12/12 claims recorded-exact",
		".didrun-history/2026-07-19-p07b-c-c3l-final/.didrun/",
		".countershape/evidence/p07b-c-c3l-final-efa918928246.html",
	],
	"docs/PROMPT_PACK.md": [
		"P07B-C C3L legacy-lattice maintenance",
		requiredC3LDigest,
		sealedC3LIdentity.commit,
		"C3 remains frozen to forty paths",
	],
	"docs/VERIFICATION.md": [
		"## C3L legacy-lattice maintenance",
		requiredC3LDeclaration,
		"U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED",
		"C3 remains frozen to forty exact paths",
		sealedC3LIdentity.commit,
		"12/12 claims recorded-exact",
	],
	[c3lStatusPath]: [
		"# P07B-C C3L — legacy-lattice maintenance",
		"Classification: `DEFECT_REPAIR`",
		requiredC3LDeclaration,
		sealedC3AIdentity.commit,
		sealedC3AIdentity.tree,
		sealedC3AIdentity.noteBlob,
		sealedC3AIdentity.noteBodySHA256,
		"fix: admit exact C3 store model edge through U6",
		"U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED",
		c3lStatusBoundaryLine,
		c3lStatusParentLine,
		c3lStatusUnreceiptedLine,
		"binds current `HEAD` to sealed C3A's exact commit/tree and exact `refs/notes/didrun` blob/body/11-claim coverage",
		...c3lClaimLabels.map((label) => `\`${label}\``),
	],
	"tools/check-p07b-c-plan.mjs": [
		"requiredC3LPaths", "C3L_ACTIVE", "--verify-c3l-preseal-ledger", "requirePrivateC3LFinalRunDirectories",
		"requireSealedC3APredecessorAuthority", "validateSealedC3ANote",
	],
	"tools/check-p07b-c-unit-scope.mjs": ["\"C3L\"", "p07b-c-unit-paths/v15"],
	"tools/check-u6-architecture.mjs": [
		"admitsC3StoreBridge", "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED", "decodeGoImportLiteral", "goSourceTokens", "topLevelMatches",
	],
	"tools/check-u6-architecture-selftest.mjs": [
		"c3-store-bridge",
		"c3-store-bridge-raw-import",
		"c3-store-bridge-renamed-identifiers",
		"c3-store-bridge-wrong-import-alias",
		"c3-store-model-import-without-bridge",
		"c3-store-bridge-signature-drift",
		"c3-store-bridge-field-type-drift",
		"c3-store-bridge-field-tag",
		"c3-store-bridge-swapped-field-order",
		"c3-store-bridge-hidden-field",
		"c3-store-bridge-exported-fifth-field",
		"c3-store-bridge-extra-parameter",
		"c3-store-bridge-result-drift",
		"c3-store-model-import-foreign-file",
		"c3-store-model-import-raw-foreign-file",
		"c3-store-model-import-escaped-foreign-file",
		"c3-store-import-camouflage-raw-string",
		"c3-store-bridge-local-shadow",
		"c3-store-bridge-type-alias-foreign-owner",
		"c3-store-bridge-duplicate-top-level-type",
		"c3-store-bridge-duplicate-method",
	],
});
const requiredC3FText = Object.freeze({
	"docs/ARCHITECTURE.md": [
		"C3F B future-surface admission maintenance is sealed",
		"internal/contractexec/target.go:{ContractExecutionTarget,NewContractExecutionTarget,ParseContractExecutionTarget}",
		"internal/store/nonhead_contract.go:ParseContractExecutionTarget",
		"all-or-none",
	],
	"docs/HANDOFF_MODE_C.md": [
		requiredC3FDigest,
		sealedC3FIdentity.commit,
		sealedC3FIdentity.tree,
		sealedC3FIdentity.noteBlob,
		sealedC3FIdentity.noteBodySHA256,
		"5d50090882888c115c54a5bee9edb9411c826832",
		"10/10 claims recorded-exact",
		".didrun-history/2026-07-19-p07b-c-c3f-final/.didrun/",
		"complete exact four-pair bundle",
		"f7160bbbc09245a1026eb6396ac76282ab913192",
	],
	"docs/PROMPT_PACK.md": [
		"P07B-C C3F B future-surface admission maintenance",
		requiredC3FDigest,
		sealedC3FIdentity.commit,
		"C3 remains frozen to forty paths",
		"10/10 TREE-EXACT",
	],
	"docs/VERIFICATION.md": [
		"## C3F B future-surface admission maintenance",
		requiredC3FDeclaration,
		"complete exact four-pair bundle",
		"ten-claim final ledger",
		sealedC3FIdentity.commit,
		"10/10 claims recorded-exact",
	],
	[c3fStatusPath]: [
		"# P07B-C C3F — B future-surface admission maintenance",
		"Classification: `DEFECT_REPAIR`",
		requiredC3FDeclaration,
		sealedC3LIdentity.commit,
		sealedC3LIdentity.tree,
		sealedC3LIdentity.noteBlob,
		sealedC3LIdentity.noteBodySHA256,
		"fix: admit exact C3 target codec surface through B",
		"P07B_B_PREMATURE_C_SURFACE",
		"The bundle is admitted only when all four exact pairs are present",
		c3fStatusBoundaryLine,
		c3fStatusParentLine,
		c3fStatusUnreceiptedLine,
		...c3fClaimLabels.map((label) => `\`${label}\``),
	],
	"spec/verification/p07b-c-unit-paths.json": ["countershape/p07b-c-unit-paths/v15", "\"C3F\""],
	"tools/check-p07b-b-architecture.mjs": [
		"c3OfficialTargetSurface", "completeC3Surface", "partitionFutureSymbols",
		"internal/store/nonhead_contract.go:ParseContractExecutionTarget",
	],
	"tools/check-p07b-b-architecture-selftest.mjs": [
		"c3-surface-bundle", "requireExactC3SurfaceBundle", "P07B_B_SELFTEST_C3_SURFACE_BUNDLE",
		"d5745ef69f205bbe4ce445448e6189ba244adacc36fe3e44fdf56f9485100ef2",
	],
	"tools/check-p07b-c-plan.mjs": [
		"requiredC3FPaths", "C3F_ACTIVE", "--verify-c3f-preseal-ledger", "requirePrivateC3FFinalRunDirectories",
		"requireSealedC3LPredecessorAuthority", "validateSealedC3LNote",
	],
	"tools/check-p07b-c-unit-scope.mjs": ["\"C3F\"", "p07b-c-unit-paths/v15"],
});
const requiredC3SText = Object.freeze({
	"docs/HANDOFF_MODE_C.md": [
		requiredC3SDigest,
		sealedC3SIdentity.commit,
		sealedC3SIdentity.tree,
		sealedC3SIdentity.noteBlob,
		sealedC3SIdentity.noteBodySHA256,
		"5d50090882888c115c54a5bee9edb9411c826832",
		"architecture-u6-selftest",
		"10/10 claims recorded-exact",
		".didrun-history/2026-07-20-p07b-c-c3s-final/.didrun/",
		".countershape/evidence/p07b-c-c3s-final-47a65b45a50c.html",
	],
	"docs/PROMPT_PACK.md": [
		"P07B-C C3S cumulative U6 self-test maintenance",
		requiredC3SDigest,
		"C3 remains frozen to forty paths",
	],
	"docs/VERIFICATION.md": [
		"## C3S cumulative U6 self-test maintenance",
		requiredC3SDeclaration,
		"architecture-u6-selftest",
		"147-case roster",
		sealedC3SIdentity.commit,
		"10/10 claims recorded-exact",
		"C3 binds sealed C3S as direct predecessor",
		"C3 remains frozen to forty exact paths",
	],
	[c3sStatusPath]: [
		"# P07B-C C3S — cumulative U6 self-test maintenance",
		"Classification: `DEFECT_REPAIR`",
		requiredC3SDeclaration,
		sealedC3FIdentity.commit,
		sealedC3FIdentity.tree,
		sealedC3FIdentity.noteBlob,
		sealedC3FIdentity.noteBodySHA256,
		"fix: make U6 C3 fixture phase-stable",
		"architecture-u6-selftest",
		"The U6 self-test now accepts exactly two starting phases:",
		"checksum-pinned 147-case roster",
		c3sStatusBoundaryLine,
		c3sStatusParentLine,
		c3sStatusUnreceiptedLine,
		...c3sClaimLabels.map((label) => `\`${label}\``),
	],
	"spec/verification/p07b-c-unit-paths.json": ["countershape/p07b-c-unit-paths/v15", "\"C3S\""],
	"tools/check-p07b-c-plan.mjs": [
		"requiredC3SPaths", "C3S_ACTIVE", "--verify-c3s-preseal-ledger", "requirePrivateC3SFinalRunDirectories",
		"requireSealedC3FPredecessorAuthority", "validateSealedC3FNote",
	],
	"tools/check-p07b-c-unit-scope.mjs": ["\"C3S\"", "p07b-c-unit-paths/v15"],
	"tools/check-u6-architecture-selftest.mjs": [
		"requireExact", "c3StoreModelImport", "c3StoreInput", "c3StoreRoots", "c3StoreMethodSignatures",
		"requireC3StoreBridge", "ensureC3StoreBridge",
	],
});
const requiredC3Text = Object.freeze({
	"docs/ARCHITECTURE.md": [
		"P07B-C through C3S is sealed and strict-clean; C3 exact-target publication is active",
		"content-portable across independently opened repositories",
		"OfficialTarget.Valid()` is structural",
		"first-durable-writer-wins",
		"structurally substitutes instantiated generic arguments",
		"module-wide named-type and function graphs",
	],
	"docs/CLAIM_VOCABULARY.md": [
		"content-portable object-format/commit/tree identity",
		"C4 requires immediate physical reopen",
		"not Node vendor authenticity",
	],
	"docs/CONCEPT_BRIEF.md": [
		"through C3S sealed; C3 exact-target publication active",
		"content-portable object format plus immutable commit/tree OIDs",
		"first durable writer",
	],
	"docs/HANDOFF_MODE_C.md": [
		requiredC3Digest,
		requiredC3BDigest,
		sealedC3SIdentity.noteBlob,
		sealedC3SIdentity.noteBodySHA256,
		"explicitly supersedes rather than literally satisfies C3V's blanket future-byte trigger",
		"52-case C1V load-sensitive qualification matrix was not rerun",
		"2908.769",
	],
	"docs/PROMPT_PACK.md": [
		"C3 exact-target publication is the active forty-path source unit",
		"P07B-C C3 exact-target publication",
		requiredC3Digest,
		requiredC3BDigest,
	],
	"docs/SEMANTICS.md": [
		"C3 exact-target publication is active",
		"OfficialTarget.Valid()` rechecks structural joins only",
		"first durable writer",
	],
	"docs/STATE_MACHINES.md": [
		"C3 exact-target publication is active",
		"ReopenOfficialTarget",
		"first-durable-writer-wins",
	],
	"docs/THREAT_MODEL.md": [
		"C3 exact-target publication is active",
		"OfficialTarget.Valid()` is structural",
		"not that it is vendor-authentic Node",
	],
	"docs/VERIFICATION.md": [
		"## C3 exact-target publication",
		"unit-scope v11 specification",
		requiredC3Digest,
		"20 added and 20 modified",
		"five authority profiles",
		"eighteen-claim final ledger",
		"C3B changes no source or verifier behavior",
		"positive input-only type/function/method consumers",
		"tuple-positioned module function/method call results",
		"direct and promoted generic method calls/values/expressions",
		"Non-call comma-ok tuple projection and local value/function shadowing are not modeled exactly",
		"roster-only verifier enrollment",
		"explicitly supersedes rather than literally satisfies sealed C3V's blanket future-byte trigger",
		"52-case C1V load-sensitive qualification matrix was not rerun",
		"does not establish repeated load-flake stability",
		"four exact deliberate-concurrency tests",
		"machine-checked C3 source inventory",
		"complete Git materialization and all non-concurrency behavior remain in the five normal authority profiles",
	],
	"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md": [
		"content-portable tree identity",
		"24-phase",
		"C3S/C3F/C3L/C3A predecessor authority manifest",
		"eighteen-claim source ledger",
		"seven-claim C3B receipt ledger",
		"snapshots every scoped store path before refusal",
		"OfficialTarget.Valid()` is snapshot-only and performs no storage repair",
		"import-qualified module-wide named-type and function graphs",
		"true declared import names",
		"dependent/renamed receiver constraints",
		"shallowest-promoted selected fields",
		"parenthesized callee and built-in forms normalize before resolution",
		"TestC3ConformanceAttemptConcurrentValidationAndReopenAreRaceFree",
		"four owning packages",
		"roster-only enrollment of the seven C3 steps",
		"not as an execution-setting change",
		"explicit superseding exception to C3V's blanket future-byte trigger",
		"52-case C1V load-sensitive qualification matrix is not rerun",
		"cooperating-code shape policy rather than whole-program dataflow proof",
	],
	[c3StatusPath]: [
		"# P07B-C C3 — exact contract-execution target publication",
		`- **Scope:** 40 exact mode-\`100644\` paths, no prefixes; sorted-newline roster \`${requiredC3Digest}\``,
		`- **Commit subject:** \`${c3SourceSubject}\``,
		"20 added and 20 modified regular files",
		c3PredecessorManifestPath,
		"preserve source-ordered lexical block/function/receiver type-parameter scopes",
		"omitted output type parameters fail closed",
		"direct or shallowest-promoted generic method call/value/expression results",
		"Local value/function shadowing and non-call comma-ok tuple projection are not modeled exactly",
		"roster-only verifier enrollment of seven exact C3 steps",
		"explicitly supersedes rather than literally satisfies the sealed C3V blanket future-byte trigger",
		"52-case C1V load-sensitive qualification matrix was not rerun",
		"does not establish repeated load-flake stability",
		"future execution-setting, sensitive-package classification, lock-semantics, cache-lifetime, or repetition-semantics change",
		"four exact deliberate-concurrency tests",
		"machine-checks their exact direct `go`-statement counts",
		...c3ClaimLabels.map((label) => `\`${label}\``),
		...c3bClaimLabels.map((label) => `\`${label}\``),
	],
	[c3PredecessorManifestPath]: [
		"p07b-c-c3-predecessors/v3",
		sealedC3SIdentity.commit, sealedC3FIdentity.commit, sealedC3LIdentity.commit, sealedC3AIdentity.commit,
	],
	"tools/check-p07b-c-plan.mjs": [
		"runC3ReceiptSelfTest", "--verify-c3-preseal-ledger", "--verify-c3-local-evidence",
		"--verify-c3b-staged", "--verify-c3b-credential-scan", "--verify-c3b-preseal-ledger",
	],
	"tools/check-p07b-c-architecture.mjs": ["c3-official-target", "predecessorAuthority", "P07B_C3_"],
	"tools/check-p07b-c-architecture-selftest.mjs": ["--c3", "c3-official-target"],
	"tools/check-p07b-c-unit-scope.mjs": ["\"C3\"", "\"C3R\"", "\"C3Q\"", "\"C3T\"", "\"C3U\"", "\"C3B\"", "p07b-c-unit-paths/v15"],
	"tools/verify-current.mjs": ["c3-official-target", "c3-single-target", "c3-hostepoch", "c3-noderuntime", "c3-store-bridge"],
});
const requiredC3RText = Object.freeze({
	"docs/HANDOFF_MODE_C.md": [
		requiredC3RDigest,
		requiredC3QDigest,
		requiredC3BDigest,
		sealedC3RIdentity.commit,
		sealedC3RIdentity.tree,
		sealedC3RIdentity.noteBlob,
		sealedC3RIdentity.noteBodySHA256,
		"1d677db4b9049084ea066263b1bb6c25ab62e5f0",
		".didrun-history/2026-07-20-p07b-c-c3b-build-loop-pre-c3q/.didrun/",
		"one exact sealed-note argv projection",
		"9/9 claims recorded-exact",
		"secrets_override: true",
		"exact recorded seal disclosure",
		"does not prove plain-seal-first sequencing",
	],
	"docs/PROMPT_PACK.md": [
		"P07B-C C3R note-preview checker maintenance",
		requiredC3RDigest,
		sealedC3RIdentity.commit,
		"9/9 TREE-EXACT",
		"C3B remains frozen to three paths",
		"countershape/p07b-c-unit-paths/v13",
	],
	"docs/VERIFICATION.md": [
		"## C3R sealed-note preview compatibility maintenance",
		requiredC3RDeclaration,
		"countershape/p07b-c-unit-paths/v13",
		"c3RaceNotePreviewPattern",
		"byte-exact raw local-ledger argv",
		"nine-claim final ledger",
		sealedC3RIdentity.commit,
		sealedC3RIdentity.noteBlob,
		"9/9 claims recorded-exact",
		"secrets_override: true",
		"outcome-dependent",
		"does not prove plain-seal-first sequencing",
	],
	[c3rStatusPath]: [
		"# P07B-C C3R — sealed-note preview compatibility maintenance",
		"Classification: `DEFECT_REPAIR`",
		requiredC3RDeclaration,
		c3rStatusBoundaryLine,
		c3rStatusParentLine,
		c3rStatusUnreceiptedLine,
		`- **Commit subject:** \`${c3rSourceSubject}\``,
		"fc02f6722716f0600b1e1d7af28cb91925d1047c",
		".didrun-history/2026-07-20-p07b-c-c3b-build-loop-pre-c3r/.didrun/",
		"one exact sealed-note argv projection",
		"byte-exact raw local-ledger argv",
		"plain seal first",
		"secrets_override: false",
		"secrets_override: true",
		"outcome-dependent",
		"does not prove plain-seal-first sequencing",
		...c3rClaimLabels.map((label) => `\`${label}\``),
	],
	"spec/verification/p07b-c-unit-paths.json": ["countershape/p07b-c-unit-paths/v15", "\"C3R\""],
	"tools/check-p07b-c-plan.mjs": [
		"requiredC3RPaths", "C3R_ACTIVE", "validateC3RClaimPreview", "validateC3RAuthority",
		"c3rSealOutcomeDisclosure",
		"--verify-c3r-sealed-c3-note", "--verify-c3r-preseal-ledger", "verifyC3RPresealLedger",
	],
	"tools/check-p07b-c-unit-scope.mjs": ["\"C3R\"", "p07b-c-unit-paths/v15"],
});
const requiredC3QText = Object.freeze({
	"docs/HANDOFF_MODE_C.md": [
		requiredC3QDigest,
		requiredC3BDigest,
		sealedC3RIdentity.commit,
		sealedC3RIdentity.tree,
		"1d677db4b9049084ea066263b1bb6c25ab62e5f0",
		"fc02f6722716f0600b1e1d7af28cb91925d1047c",
		".didrun-history/2026-07-20-p07b-c-c3b-build-loop-pre-c3q/.didrun/",
		"phase-asymmetric self-receipt enforcement",
		"exact plan-authority Markdown corpus",
		"sealed C3Q matrix remains exact historical evidence on its sealed tree",
		"visible-Markdown heading boundaries",
		"eight supplemental source-identity",
		"two strict-zero prohibition controls",
		"sha256:16d63155a00c9e87b7eec79707ad6d47d6f5d70205d18243b8bc687e401e4b66",
		"2 phases × 42 paths × 15 positive forms",
	],
	"docs/PROMPT_PACK.md": [
		"P07B-C C3Q self-receipt checker maintenance",
		requiredC3QDigest,
		"countershape/p07b-c-unit-paths/v13",
		"sha256:16d63155a00c9e87b7eec79707ad6d47d6f5d70205d18243b8bc687e401e4b66",
		"C3B remains frozen to three paths",
	],
	"docs/VERIFICATION.md": [
		"## C3Q active-unit self-receipt maintenance",
		requiredC3QDeclaration,
		"countershape/p07b-c-unit-paths/v13",
		"phase-asymmetric self-receipt enforcement",
		"exact plan-authority Markdown corpus",
		"sha256:16d63155a00c9e87b7eec79707ad6d47d6f5d70205d18243b8bc687e401e4b66",
		"2 phases × 42 paths × 15 positive forms",
		"Eight supplemental natural aliases",
		"two explicit prohibition controls",
		"bounded lexical grammar",
		"projected through visible Markdown",
		"fails closed if the declared constant has drifted",
		"nine-claim final ledger",
		"--print-plan-authority-corpus",
		"C3Q changes no verifier roster",
	],
	[c3qStatusPath]: [
		"# P07B-C C3Q — active-unit self-receipt maintenance",
		"Classification: `DEFECT_REPAIR`",
		requiredC3QDeclaration,
		c3qStatusBoundaryLine,
		c3qStatusParentLine,
		c3qStatusUnreceiptedLine,
		`- **Commit subject:** \`${c3qSourceSubject}\``,
		"1d677db4b9049084ea066263b1bb6c25ab62e5f0",
		"fc02f6722716f0600b1e1d7af28cb91925d1047c",
		".didrun-history/2026-07-20-p07b-c-c3b-build-loop-pre-c3q/.didrun/",
		"phase-asymmetric self-receipt enforcement",
		"exact plan-authority Markdown corpus",
		"sha256:16d63155a00c9e87b7eec79707ad6d47d6f5d70205d18243b8bc687e401e4b66",
		"2 phases × 42 paths × 15 positive forms",
		"Eight supplemental C3Q/C3B",
		"two explicit strict-zero prohibition controls",
		"bounded lexical grammar",
		"visible Markdown",
		"inert comment/fence heading decoys",
		"## Final command order",
		c3qFinalCommandOrderMarkdown,
		"/opt/homebrew/bin/node tools/check-p07b-c-plan.mjs --print-plan-authority-corpus",
		"Report mode recomputes the sorted-path digest",
		"scope: positive-receipt-form refusal only; not a general Markdown or security audit",
		"C3Q's exact validated note blob OID and note-body SHA-256",
		"current installation does not prevent concurrent verifier work",
		"C3R remains an exact sealed ancestor",
		"C3Q changes no verifier roster",
		...c3qClaimLabels.map((label) => `\`${label}\``),
	],
	"spec/verification/p07b-c-unit-paths.json": ["countershape/p07b-c-unit-paths/v15", "\"C3Q\""],
	"tools/check-p07b-c-plan.mjs": [
		"requiredC3QPaths", "C3Q_ACTIVE", "validateC3QClaimPreview", "validateC3QAuthority",
		"planAuthorityMarkdownPaths", "planAuthorityMarkdownDigest", "computedPlanAuthorityMarkdownDigest",
		"planAuthorityMarkdownCorpusReport",
		"--print-plan-authority-corpus", "rejectActiveUnitSelfReceipts",
		"runActiveUnitSelfReceiptMatrixSelfTest", "c3qSealOutcomeDisclosure", "c3bActiveMaintenanceSection",
		"currentStateSubsection", "replaceCurrentStateSubsection",
		"active-phases=C3U,C3B", "result=positive-form refusal only, not a general Markdown or security audit",
		"supplemental-aliases=", "supplemental-controls=", "visible Markdown subsection projection",
		"declared sealed-C3R identity",
		"--verify-c3q-sealed-c3r-note", "--verify-c3q-preseal-ledger", "verifyC3QPresealLedger",
	],
	"tools/check-p07b-c-unit-scope.mjs": ["\"C3Q\"", "p07b-c-unit-paths/v15"],
});

const requiredC3TText = Object.freeze({
	"docs/HANDOFF_MODE_C.md": [
		requiredC3TDigest, requiredC3BDigest,
		sealedC3QIdentity.commit, sealedC3QIdentity.tree,
		sealedC3QIdentity.noteBlob, sealedC3QIdentity.noteBodySHA256,
		"eab9839c902abf52e9729144a2244740228a41eb",
		".didrun-history/2026-07-20-p07b-c-c3b-build-loop-post-c3q-selftest-red/.didrun/",
		"P07B-C phase fixture C3 absent downgrade active state anchor count 0",
		"validated C3T-active absent fixture", "present → absent → present",
		"canonical one-line receipt declaration", "exact terminal handoff block", "sealed-descendant plan coherence",
		"sha256:52e78638d70f4e229e0b54c0a8311b6e0d1d2acc0d4e439cacf10cde65c62959",
		"2 phases × 43 paths × 15 positive forms",
	],
	"docs/PROMPT_PACK.md": [
		"P07B-C C3T receipt-phase self-test maintenance", requiredC3TDigest,
		sealedC3QIdentity.commit, "countershape/p07b-c-unit-paths/v14",
		"sha256:52e78638d70f4e229e0b54c0a8311b6e0d1d2acc0d4e439cacf10cde65c62959",
		"C3B remains frozen to three paths",
	],
	"docs/VERIFICATION.md": [
		"## C3T receipt-phase self-test maintenance", requiredC3TDeclaration,
		"countershape/p07b-c-unit-paths/v14", "validated C3T-active absent fixture",
		"present → absent → present", "2 phases × 43 paths × 15 positive forms",
		"canonical one-line receipt declaration", "exact terminal handoff block", "sealed-descendant plan coherence",
		"1,290 required rejections", "430", "nine-claim final ledger",
		"C3T changes no verifier roster",
	],
	[c3tStatusPath]: [
		"# P07B-C C3T — receipt-phase self-test maintenance", "Classification: `DEFECT_REPAIR`",
		requiredC3TDeclaration, c3tStatusBoundaryLine, c3tStatusParentLine, c3tStatusUnreceiptedLine,
		`- **Commit subject:** \`${c3tSourceSubject}\``, "countershape/p07b-c-unit-paths/v14",
		"eab9839c902abf52e9729144a2244740228a41eb",
		".didrun-history/2026-07-20-p07b-c-c3b-build-loop-post-c3q-selftest-red/.didrun/",
		".countershape/history/p07bc-c3b-post-c3q-selftest-red-root",
		"P07B-C phase fixture C3 absent downgrade active state anchor count 0",
		"validated C3T-active absent fixture", "present → absent → present",
		"canonical one-line receipt declaration", "exact terminal handoff block", "sealed-descendant plan coherence",
		"sha256:52e78638d70f4e229e0b54c0a8311b6e0d1d2acc0d4e439cacf10cde65c62959",
		"2 phases × 43 paths × 15 positive forms", "1,290 required rejections", "430",
		"## Final command order", c3tFinalCommandOrderMarkdown,
		...c3tClaimLabels.map((label) => `\`${label}\``),
	],
	"spec/verification/p07b-c-unit-paths.json": ["countershape/p07b-c-unit-paths/v15", "\"C3T\""],
	"tools/check-p07b-c-plan.mjs": [
		"requiredC3TPaths", "C3T_ACTIVE", "syntheticC3AbsentPlanFixture",
		"validateC3TClaimPreview", "validateC3TAuthority", "loadC3TAuthorityWithRunner",
		"active-phases=C3U,C3B", "--verify-c3t-sealed-c3q-note",
		"--verify-c3t-preseal-ledger", "verifyC3TPresealLedger",
	],
	"tools/check-p07b-c-unit-scope.mjs": ["\"C3T\"", "p07b-c-unit-paths/v15"],
});

const requiredC3UText = Object.freeze({
	"docs/HANDOFF_MODE_C.md": [
		requiredC3UDigest, requiredC3BDigest,
		sealedC3TIdentity.commit, sealedC3TIdentity.tree,
		sealedC3TIdentity.noteBlob, sealedC3TIdentity.noteBodySHA256,
		"cca3045b55ddc00e011c4331f4731bd1a433ea36",
		".didrun-history/2026-07-20-p07b-c-c3b-build-loop-post-c3t-c2-terminal-red/.didrun/",
		"C3 source receipt block must be the exact terminal handoff block",
		"optional already-canonical terminal C3 block", "byte-for-byte",
		"sha256:1c5bf120f1ab948586bfa594e6ae634fc6acf7a5544febcccb2c4dc475a59a49",
		"2 phases × 44 paths × 15 positive forms", "440 core accepted controls",
		"eight supplemental natural aliases reject", "two explicit strict-zero prohibition controls remain accepted",
	],
	"docs/PROMPT_PACK.md": [
		"P07B-C C3U cross-phase receipt-fixture maintenance", requiredC3UDigest,
		sealedC3TIdentity.commit, "countershape/p07b-c-unit-paths/v15",
		"sha256:1c5bf120f1ab948586bfa594e6ae634fc6acf7a5544febcccb2c4dc475a59a49",
		"C3B remains frozen to three paths",
	],
	"docs/VERIFICATION.md": [
		"## C3U cross-phase receipt-fixture maintenance", requiredC3UDeclaration,
		"countershape/p07b-c-unit-paths/v15", "insertLegacyReceiptBlockBeforeOptionalTerminalC3",
		"C0, C1, C2, or C3P", "exact terminal C3 block", "byte-for-byte",
		".didrun-history/2026-07-20-p07b-c-c3b-build-loop-post-c3t-c2-terminal-red/.didrun/",
		"2 phases × 44 paths × 15 positive forms", "1,320 required rejections", "440",
		"Eight supplemental natural aliases must reject", "two explicit strict-zero prohibition controls must remain accepted",
		"nine-claim final ledger", "C3U changes no verifier roster",
	],
	[c3uStatusPath]: [
		"# P07B-C C3U — cross-phase receipt-fixture maintenance", "Classification: `DEFECT_REPAIR`",
		requiredC3UDeclaration, c3uStatusBoundaryLine, c3uStatusParentLine, c3uStatusUnreceiptedLine,
		`- **Commit subject:** \`${c3uSourceSubject}\``, "countershape/p07b-c-unit-paths/v15",
		"cca3045b55ddc00e011c4331f4731bd1a433ea36",
		".didrun-history/2026-07-20-p07b-c-c3b-build-loop-post-c3t-c2-terminal-red/.didrun/",
		"C3 source receipt block must be the exact terminal handoff block",
		"insertLegacyReceiptBlockBeforeOptionalTerminalC3", "C0, C1, C2, and C3P",
		"The actual C0, C1, C2, and C3P fixture callers each run as a full-plan terminal-C3 control",
		"marker-free", "exact terminal C3 block", "byte-for-byte", "full-plan",
		"sha256:1c5bf120f1ab948586bfa594e6ae634fc6acf7a5544febcccb2c4dc475a59a49",
		"2 phases × 44 paths × 15 positive forms", "1,320 required rejections", "440",
		"Eight supplemental natural aliases must reject", "two explicit strict-zero prohibition controls must remain accepted",
		"## Final command order", c3uFinalCommandOrderMarkdown,
		...c3uClaimLabels.map((label) => `\`${label}\``),
	],
	"spec/verification/p07b-c-unit-paths.json": ["countershape/p07b-c-unit-paths/v15", "\"C3U\""],
	"tools/check-p07b-c-plan.mjs": [
		"requiredC3UPaths", "C3U_ACTIVE", "insertLegacyReceiptBlockBeforeOptionalTerminalC3",
		"validateC3UClaimPreview", "validateC3UAuthority", "loadC3UAuthorityWithRunner",
		"--verify-c3u-sealed-c3t-note", "--verify-c3u-preseal-ledger", "verifyC3UPresealLedger",
	],
	"tools/check-p07b-c-unit-scope.mjs": ["\"C3U\"", "p07b-c-unit-paths/v15"],
});

const planAuthorityMarkdownPaths = Object.freeze([...new Set([
	statusPath,
	c1StatusPath,
	c2StatusPath,
	c1DidrunBugsPath,
	c3pStatusPath,
	c3StatusPath,
	...controllingPaths,
	...[
		requiredText,
		requiredC1MaintenanceText,
		requiredC1VerificationText,
		requiredC1EvidenceMaintenanceText,
		requiredC2MaintenanceText,
		requiredC3PText,
		requiredC3VText,
		requiredC3MaintenanceText,
		requiredC3AText,
		requiredC3LText,
		requiredC3FText,
		requiredC3SText,
		requiredC3Text,
		requiredC3RText,
		requiredC3QText,
		requiredC3TText,
		requiredC3UText,
	].flatMap((authority) => Object.keys(authority)),
].filter((path) => path.endsWith(".md")))].sort((left, right) =>
	Buffer.compare(Buffer.from(left, "utf8"), Buffer.from(right, "utf8"))));
const planAuthorityMarkdownDigest = "sha256:1c5bf120f1ab948586bfa594e6ae634fc6acf7a5544febcccb2c4dc475a59a49";

function computedPlanAuthorityMarkdownDigest() {
	return `sha256:${createHash("sha256")
		.update(`${planAuthorityMarkdownPaths.join("\n")}\n`, "utf8").digest("hex")}`;
}

export function planAuthorityMarkdownCorpusReport() {
	const computedDigest = computedPlanAuthorityMarkdownDigest();
	if (computedDigest !== planAuthorityMarkdownDigest) {
		throw new Error(`plan-authority Markdown corpus digest mismatch: declared=${planAuthorityMarkdownDigest} computed=${computedDigest}`);
	}
	return [
		`P07B-C plan-authority Markdown corpus: paths=${planAuthorityMarkdownPaths.length} digest=${computedDigest}`,
		...planAuthorityMarkdownPaths.map((path) => `- ${path}`),
		"scope: positive-receipt-form refusal only; not a general Markdown or security audit",
		"",
	].join("\n");
}

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
	const exactUnit = "\\b" + unit + "(?![A-Z0-9])";
	const exactClaim = "P07B-C " + unit + " [^`\\n]+";
	const patterns = [
		["commit-identity", new RegExp(exactUnit + " (?:(?:receipt|source) )?commit `[0-9a-f]{40}`", "iu")],
		["tree-identity", new RegExp(exactUnit + " (?:(?:receipt|source) )?tree `[0-9a-f]{40}`", "iu")],
		["closure-state", new RegExp(exactUnit + "(?: (?:receipt|source|unit|boundary|evidence))? (?:is|was|has been) (?:sealed|strict-clean|verified|complete|closed)", "iu")],
		["strict-zero", new RegExp(exactUnit + "(?: source)? strict exit(?:\\s*:\\s*|\\s+)`?0`?(?![^\\n]{0,64}\\b(?:forbidden|invalid|unacceptable|disallowed))", "iu")],
		["recorded-exact-count", new RegExp(exactUnit + "(?:(?: source)? strict claims[^\\n]{0,96}claims recorded-exact| (?:has|records?) [0-9]+/[0-9]+ claims recorded-exact)", "iu")],
		["evidence-association", new RegExp(exactUnit + "(?: source)? (?:evidence|receipt|grade|result)(?![A-Za-z0-9_-])[^\\n|]{0,256}(?:`?(?:TREE-EXACT|RECORDED-EXACT)`?|claims recorded-exact|note-present|strict exit)", "iu")],
		["split-evidence-association", new RegExp(exactUnit + "(?: source)? (?:evidence|receipt|grade|result)(?![A-Za-z0-9_-])[ \\t]*:?[ \\t]*\\n(?:[ \\t]*\\n)?[^\\n|]{0,256}(?:`?(?:TREE-EXACT|RECORDED-EXACT)`?|claims recorded-exact|note-present|strict exit)", "iu")],
		["canonical-claim-table", new RegExp("\\|\\s*`P07B-C " + unit + " [^`]+`\\s*\\|\\s*`(?:tests-pass|command-succeeded)`\\s*\\|\\s*`(?:TREE-EXACT|RECORDED-EXACT)`", "iu")],
		["claim-table-grade", new RegExp("^\\|(?=[^\\n]*`" + exactClaim + "`)(?=[^\\n]*`(?:TREE-EXACT|RECORDED-EXACT)`)[^\\n]*\\|$", "imu")],
		["passed-verification", new RegExp(exactUnit + " (?:passed verification|verification passed)", "iu")],
		["recorded-exact-verdict", new RegExp(exactUnit + "[^\\n]{0,96}ALL RECORDED-EXACT", "iu")],
		["html-receipt-marker", new RegExp("<!--\\s*P07B-C-" + unit + "[^>]*(?:RECEIPT|RECEIPTS)[^>]*(?:START|END)[^>]*-->", "iu")],
		["plain-receipt-marker", new RegExp(exactUnit + "\\s+(?:receipt\\s+)?(?:START|END)(?![A-Z0-9])", "iu")],
	];
	const matched = patterns.find(([, pattern]) => pattern.test(body));
	if (matched !== undefined) {
		errors.push(`${path}: ${unit} self-receipt forbidden (${matched[0]}); remove or reframe the positive ${unit} result`);
	}
}

function rejectActiveUnitSelfReceipts(bodies, unit, errors) {
	const paths = [...bodies.keys()].filter((path) => path.endsWith(".md")).sort();
	for (const path of paths) rejectReceiptSelfClaims(bodies.get(path), path, unit, errors);
	return paths;
}

const activeUnitSelfReceiptPositiveForms = Object.freeze([
	Object.freeze({ name: "commit identity", label: "commit-identity", render: (unit) => `${unit} receipt commit \`${"a".repeat(40)}\`` }),
	Object.freeze({ name: "tree identity", label: "tree-identity", render: (unit) => `${unit} receipt tree \`${"b".repeat(40)}\`` }),
	Object.freeze({ name: "sealed state", label: "closure-state", render: (unit) => `${unit} is sealed and strict-clean.` }),
	Object.freeze({ name: "strict zero", label: "strict-zero", render: (unit) => `${unit} strict exit: 0` }),
	Object.freeze({ name: "recorded-exact count", label: "recorded-exact-count", render: (unit) => `${unit} strict claims: 9/9 claims recorded-exact` }),
	Object.freeze({ name: "same-line evidence grade", label: "evidence-association", render: (unit) => `${unit} evidence: TREE-EXACT` }),
	Object.freeze({ name: "split evidence grade", label: "split-evidence-association", render: (unit) => `${unit} evidence:\n\nTREE-EXACT` }),
	Object.freeze({ name: "canonical claim table", label: "canonical-claim-table", render: (unit) => `| \`P07B-C ${unit} synthetic claim\` | \`tests-pass\` | \`TREE-EXACT\` |` }),
	Object.freeze({ name: "reordered claim table", label: "claim-table-grade", render: (unit) => `| \`TREE-EXACT\` | \`P07B-C ${unit} synthetic claim\` |` }),
	Object.freeze({ name: "passed verification", label: "passed-verification", render: (unit) => `${unit} passed verification.` }),
	Object.freeze({ name: "recorded-exact verdict", label: "recorded-exact-verdict", render: (unit) => `${unit} verdict: ALL RECORDED-EXACT` }),
	Object.freeze({ name: "HTML receipt start", label: "html-receipt-marker", render: (unit) => `<!-- P07B-C-${unit}-SOURCE-RECEIPTS:START -->` }),
	Object.freeze({ name: "HTML receipt end", label: "html-receipt-marker", render: (unit) => `<!-- P07B-C-${unit}-SOURCE-RECEIPTS:END -->` }),
	Object.freeze({ name: "plain receipt start", label: "plain-receipt-marker", render: (unit) => `${unit} receipt START` }),
	Object.freeze({ name: "plain receipt end", label: "plain-receipt-marker", render: (unit) => `${unit} receipt END` }),
]);

const activeUnitSelfReceiptPositiveControls = Object.freeze([
	(unit) => `${unit} remains \`UNRECEIPTED\` until its own later boundary.`,
	(unit) => `${unit} may be sealed only after all pending gates pass.`,
	(unit) => `${unit} is blocked pending a future commit and receipt.`,
	(unit) => `${unit} development strict exit: \`1\`; zero claims were declared.`,
	() => "Sealed C3R is predecessor evidence; the active descendant remains unreceipted.",
]);

function runActiveUnitSelfReceiptMatrixSelfTest() {
	const units = Object.freeze(["C3U", "C3B"]);
	let rejected = 0;
	let controls = 0;
	let supplementalAliases = 0;
	let supplementalControls = 0;
	for (const unit of units) {
		for (const path of planAuthorityMarkdownPaths) {
			for (const form of activeUnitSelfReceiptPositiveForms) {
				const errors = [];
				rejectActiveUnitSelfReceipts(new Map([[path, `${form.render(unit)}\n`]]), unit, errors);
				const expected = `${path}: ${unit} self-receipt forbidden (${form.label}); remove or reframe the positive ${unit} result`;
				if (!isDeepStrictEqual(errors, [expected])) {
					throw new Error(`P07B-C active-unit self-receipt matrix false negative: ${unit}/${path}/${form.name}`);
				}
				rejected += 1;
			}
			for (const render of activeUnitSelfReceiptPositiveControls) {
				const errors = [];
				rejectActiveUnitSelfReceipts(new Map([[path, `${render(unit)}\n`]]), unit, errors);
				if (errors.length > 0) {
					throw new Error(`P07B-C active-unit self-receipt matrix false positive: ${unit}/${path} (${errors.join("; ")})`);
				}
				controls += 1;
			}
		}
	}
	for (const unit of units) {
		for (const [name, label, rendered] of [
			["source commit", "commit-identity", `${unit} source commit \`${"c".repeat(40)}\``],
			["source tree", "tree-identity", `${unit} source tree \`${"d".repeat(40)}\``],
			["verification passed", "passed-verification", `${unit} verification passed.`],
			["natural claim count", "recorded-exact-count", `${unit} has 9/9 claims recorded-exact.`],
		]) {
			const path = "docs/VERIFICATION.md";
			const errors = [];
			rejectActiveUnitSelfReceipts(new Map([[path, `${rendered}\n`]]), unit, errors);
			const expected = `${path}: ${unit} self-receipt forbidden (${label}); remove or reframe the positive ${unit} result`;
			if (!isDeepStrictEqual(errors, [expected])) {
				throw new Error(`P07B-C active-unit self-receipt supplemental alias false negative: ${unit}/${name}`);
			}
			supplementalAliases += 1;
		}
		const path = "docs/VERIFICATION.md";
		const errors = [];
		rejectActiveUnitSelfReceipts(
			new Map([[path, `${unit} strict exit 0 is forbidden before its own sealed boundary.\n`]]),
			unit,
			errors,
		);
		if (errors.length > 0) {
			throw new Error(`P07B-C active-unit self-receipt supplemental prohibition false positive: ${unit} (${errors.join("; ")})`);
		}
		supplementalControls += 1;
	}
	return Object.freeze({
		phases: units.length,
		paths: planAuthorityMarkdownPaths.length,
		forms: activeUnitSelfReceiptPositiveForms.length,
		rejected,
		controls,
		supplementalAliases,
		supplementalControls,
	});
}

function rejectC1BSelfReceiptClaims(body, path, errors) {
	rejectReceiptSelfClaims(body, path, "C1B", errors);
}

function rejectC2BSelfReceiptClaims(body, path, errors) {
	rejectReceiptSelfClaims(body, path, "C2B", errors);
}

function rejectC3ACanonicalSelfClaims(body, path, errors) {
	for (const pattern of [
		/<!--\s*P07B-C-C3A-[A-Z-]*RECEIPTS:START\s*-->/iu,
		/\bC3A\s+(?:source\s+)?evidence\s*:\s*[\s\S]{0,256}(?:`?TREE-EXACT`?|claims recorded-exact|note-present|strict exit(?:\s*:\s*|\s+)`?0`?)/iu,
		/\|\s*`P07B-C C3A [^`]+`\s*\|\s*`(?:tests-pass|command-succeeded)`\s*\|\s*`(?:TREE-EXACT|RECORDED-EXACT)`/iu,
	]) {
		if (pattern.test(body)) errors.push(`${path}: C3A canonical self-receipt marker is forbidden`);
	}
}

function rejectActiveC3ASelfClaims(body, path, errors) {
	rejectReceiptSelfClaims(body, path, "C3A", errors);
	rejectC3ACanonicalSelfClaims(body, path, errors);
}

function rejectC3LCanonicalSelfClaims(body, path, errors) {
	for (const pattern of [
		/<!--\s*P07B-C-C3L-[A-Z-]*RECEIPTS:START\s*-->/iu,
		/\bC3L\s+(?:source\s+)?evidence\s*:\s*[\s\S]{0,256}(?:`?TREE-EXACT`?|claims recorded-exact|note-present|strict exit(?:\s*:\s*|\s+)`?0`?)/iu,
		/\|\s*`P07B-C C3L [^`]+`\s*\|\s*`(?:tests-pass|command-succeeded)`\s*\|\s*`(?:TREE-EXACT|RECORDED-EXACT)`/iu,
	]) {
		if (pattern.test(body)) errors.push(`${path}: C3L canonical self-receipt marker is forbidden`);
	}
}

function rejectActiveC3LSelfClaims(body, path, errors) {
	rejectReceiptSelfClaims(body, path, "C3L", errors);
	rejectC3LCanonicalSelfClaims(body, path, errors);
}

function rejectC3FCanonicalSelfClaims(body, path, errors) {
	for (const pattern of [
		/<!--\s*P07B-C-C3F-[A-Z-]*RECEIPTS:START\s*-->/iu,
		/\bC3F\s+(?:source\s+)?evidence\s*:\s*[\s\S]{0,256}(?:`?TREE-EXACT`?|claims recorded-exact|note-present|strict exit(?:\s*:\s*|\s+)`?0`?)/iu,
		/\|\s*`P07B-C C3F [^`]+`\s*\|\s*`(?:tests-pass|command-succeeded)`\s*\|\s*`(?:TREE-EXACT|RECORDED-EXACT)`/iu,
	]) {
		if (pattern.test(body)) errors.push(`${path}: C3F canonical self-receipt marker is forbidden`);
	}
}

function rejectActiveC3FSelfClaims(body, path, errors) {
	rejectReceiptSelfClaims(body, path, "C3F", errors);
	rejectC3FCanonicalSelfClaims(body, path, errors);
}

function rejectC3SCanonicalSelfClaims(body, path, errors) {
	for (const pattern of [
		/<!--\s*P07B-C-C3S-[A-Z-]*RECEIPTS:START\s*-->/iu,
		/\bC3S\s+(?:source\s+)?evidence\s*:\s*[\s\S]{0,256}(?:`?TREE-EXACT`?|claims recorded-exact|note-present|strict exit(?:\s*:\s*|\s+)`?0`?)/iu,
		/\|\s*`P07B-C C3S [^`]+`\s*\|\s*`(?:tests-pass|command-succeeded)`\s*\|\s*`(?:TREE-EXACT|RECORDED-EXACT)`/iu,
	]) {
		if (pattern.test(body)) errors.push(`${path}: C3S canonical self-receipt marker is forbidden`);
	}
}

function rejectActiveC3SSelfClaims(body, path, errors) {
	rejectReceiptSelfClaims(body, path, "C3S", errors);
	rejectC3SCanonicalSelfClaims(body, path, errors);
}

function rejectC3CanonicalSelfClaims(body, path, errors) {
	for (const pattern of [
		/<!--\s*P07B-C-C3-SOURCE-RECEIPTS:START\s*-->/iu,
		/\bC3\s+(?:source\s+)?evidence\s*:\s*[\s\S]{0,256}(?:`?TREE-EXACT`?|claims recorded-exact|note-present|strict exit(?:\s*:\s*|\s+)`?0`?)/iu,
		/\|\s*`P07B-C C3 [^`]+`\s*\|\s*`(?:tests-pass|command-succeeded)`\s*\|\s*`(?:TREE-EXACT|RECORDED-EXACT)`/iu,
	]) {
		if (pattern.test(body)) errors.push(`${path}: C3 canonical self-receipt marker is forbidden`);
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

function decodeGitUTF8(bytes, label) {
	try {
		return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
	} catch (error) {
		throw new Error(`${label} is not valid UTF-8 (${error.message})`);
	}
}

function decodeExactGitLine(bytes, label) {
	const text = Buffer.isBuffer(bytes) ? decodeGitUTF8(bytes, label) : bytes;
	if (typeof text !== "string" || text.length === 0 || !text.endsWith("\n") ||
		text.slice(0, -1).includes("\n") || text.includes("\r")) {
		throw new Error(`${label} is not one exact LF-terminated line`);
	}
	return text.slice(0, -1);
}

function loadReceiptAuthorityWithRunner(receipt, run) {
	const runLine = (args, accepted = [0]) => {
		const result = run(args, accepted);
		return { status: result.status, stdout: decodeExactGitLine(result.stdout, `git ${args[0]} stdout`) };
	};
	const commit = runLine(["rev-parse", "--verify", `${receipt.source_commit}^{commit}`]).stdout;
	if (commit !== receipt.source_commit) throw new Error("source commit did not reopen exactly");
	const headCommit = runLine(["rev-parse", "--verify", "HEAD^{commit}"]).stdout;
	const tree = runLine(["rev-parse", "--verify", `${receipt.source_commit}^{tree}`]).stdout;
	const parentText = runLine(["show", "-s", "--format=%P", receipt.source_commit]).stdout;
	const parents = parentText.length === 0 ? [] : parentText.split(" ");
	if (parents.some((parent) => !/^[0-9a-f]{40}$/u.test(parent))) throw new Error("source parent line is malformed");
	const parent = parents[0] ?? "";
	let sourceDiff = [];
	if (parent !== "") {
		const diffBytes = run([
			"diff-tree", "--no-commit-id", "--raw", "-r", "-z", "--no-renames", "--full-index", parent, commit, "--",
		]).stdout;
		sourceDiff = parseRawSourceDiff(decodeGitUTF8(diffBytes, "git diff-tree stdout"));
	}
	const subject = runLine(["show", "-s", "--format=%s", receipt.source_commit]).stdout;
	const ancestorResult = run(["merge-base", "--is-ancestor", receipt.source_commit, "HEAD"], [0, 1]);
	if (ancestorResult.stdout.length !== 0) throw new Error("source ancestry check emitted unexpected stdout");
	const ancestor = ancestorResult.status === 0;
	const noteListing = runLine(["notes", "--ref=didrun", "list", receipt.source_commit]).stdout;
	if (!/^[0-9a-f]{40}$/u.test(noteListing)) {
		throw new Error("didrun Git note object identity is malformed");
	}
	const noteType = runLine(["cat-file", "-t", noteListing]).stdout;
	if (noteType !== "blob") throw new Error(`didrun Git note object is ${noteType}, not blob`);
	const noteBytes = run(["cat-file", "blob", noteListing]).stdout;
	const noteText = decodeGitUTF8(noteBytes, "didrun Git note body");
	const stableNoteListing = runLine(["notes", "--ref=didrun", "list", receipt.source_commit]).stdout;
	if (stableNoteListing !== noteListing) {
		throw new Error("didrun Git note reference changed while receipt authority was read");
	}
	let note;
	try {
		note = JSON.parse(noteText);
	} catch (error) {
		throw new Error(`didrun Git note is not JSON (${error.message})`);
	}
	return {
		head_commit: headCommit,
		commit,
		tree,
		parent,
		parents,
		source_diff: sourceDiff,
		subject,
		ancestor_of_head: ancestor,
		note,
		note_type: noteType,
		note_blob_oid: noteListing,
		note_body_sha256: createHash("sha256").update(noteBytes).digest("hex"),
	};
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
			cwd: root, encoding: "buffer", timeout: 30_000, maxBuffer: 4 * 1024 * 1024, env,
		});
		if (result.error || result.signal || !accepted.includes(result.status) || (result.stderr?.length ?? 0) !== 0) {
			throw new Error(`git ${args[0]} failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
		}
		return { status: result.status, stdout: result.stdout ?? Buffer.alloc(0) };
	};
	return loadReceiptAuthorityWithRunner(receipt, run);
}

function loadC3RAuthorityWithRunner(run, expectedParent = sealedC3Identity.commit) {
	const result = run(["rev-list", "--parents", "--ancestry-path", `${expectedParent}..HEAD`]);
	const text = decodeGitUTF8(result.stdout, "git C3R ancestry stdout");
	if (text.length === 0 || !text.endsWith("\n") || text.includes("\r")) {
		throw new Error("C3R ancestry is absent or not exact LF-framed output");
	}
	const rows = text.slice(0, -1).split("\n").map((line) => line.split(" "));
	if (rows.some((row) => row.length < 2 || row.some((oid) => !/^[0-9a-f]{40}$/u.test(oid)))) {
		throw new Error("C3R ancestry row is malformed");
	}
	const directChildren = rows.filter((row) => row.length === 2 && row[1] === expectedParent);
	if (directChildren.length !== 1) {
		throw new Error(`C3R direct-child cardinality ${directChildren.length}`);
	}
	return loadReceiptAuthorityWithRunner({ source_commit: directChildren[0][0] }, run);
}

async function loadC3RAuthorityFromGit(root) {
	const requested = process.env.COUNTERSHAPE_GIT || "/usr/bin/git";
	if (!isAbsolute(requested)) throw new Error("COUNTERSHAPE_GIT must be absolute");
	const git = await realpath(requested);
	const env = {
		HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1",
		GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
		GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
	};
	const run = (args, accepted = [0]) => {
		const result = spawnSync(git, ["--no-replace-objects", ...args], {
			cwd: root, encoding: "buffer", timeout: 30_000, maxBuffer: 4 * 1024 * 1024, env,
		});
		if (result.error || result.signal || !accepted.includes(result.status) || (result.stderr?.length ?? 0) !== 0) {
			throw new Error(`git ${args[0]} failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
		}
		return { status: result.status, stdout: result.stdout ?? Buffer.alloc(0) };
	};
	return loadC3RAuthorityWithRunner(run);
}

function loadC3QAuthorityWithRunner(run, expectedParent = sealedC3RIdentity.commit) {
	const result = run(["rev-list", "--parents", "--ancestry-path", `${expectedParent}..HEAD`]);
	const text = decodeGitUTF8(result.stdout, "git C3Q ancestry stdout");
	if (text.length === 0 || !text.endsWith("\n") || text.includes("\r")) {
		throw new Error("C3Q ancestry is absent or not exact LF-framed output");
	}
	const rows = text.slice(0, -1).split("\n").map((line) => line.split(" "));
	if (rows.some((row) => row.length < 2 || row.some((oid) => !/^[0-9a-f]{40}$/u.test(oid)))) {
		throw new Error("C3Q ancestry row is malformed");
	}
	const directChildren = rows.filter((row) => row.length === 2 && row[1] === expectedParent);
	if (directChildren.length !== 1) {
		throw new Error(`C3Q direct-child cardinality ${directChildren.length}`);
	}
	return loadReceiptAuthorityWithRunner({ source_commit: directChildren[0][0] }, run);
}

async function loadC3QAuthorityFromGit(root, expectedParent = sealedC3RIdentity.commit) {
	const requested = process.env.COUNTERSHAPE_GIT || "/usr/bin/git";
	if (!isAbsolute(requested)) throw new Error("COUNTERSHAPE_GIT must be absolute");
	const git = await realpath(requested);
	const env = {
		HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1",
		GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
		GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
	};
	const run = (args, accepted = [0]) => {
		const result = spawnSync(git, ["--no-replace-objects", ...args], {
			cwd: root, encoding: "buffer", timeout: 30_000, maxBuffer: 4 * 1024 * 1024, env,
		});
		if (result.error || result.signal || !accepted.includes(result.status) || (result.stderr?.length ?? 0) !== 0) {
			throw new Error(`git ${args[0]} failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
		}
		return { status: result.status, stdout: result.stdout ?? Buffer.alloc(0) };
	};
	return loadC3QAuthorityWithRunner(run, expectedParent);
}

function loadC3TAuthorityWithRunner(run, expectedParent = sealedC3QIdentity.commit) {
	const result = run(["rev-list", "--parents", "--ancestry-path", `${expectedParent}..HEAD`]);
	const text = decodeGitUTF8(result.stdout, "git C3T ancestry stdout");
	if (text.length === 0 || !text.endsWith("\n") || text.includes("\r")) {
		throw new Error("C3T ancestry is absent or not exact LF-framed output");
	}
	const rows = text.slice(0, -1).split("\n").map((line) => line.split(" "));
	if (rows.some((row) => row.length < 2 || row.some((oid) => !/^[0-9a-f]{40}$/u.test(oid)))) {
		throw new Error("C3T ancestry row is malformed");
	}
	const directChildren = rows.filter((row) => row.length === 2 && row[1] === expectedParent);
	if (directChildren.length !== 1) {
		throw new Error(`C3T direct-child cardinality ${directChildren.length}`);
	}
	return loadReceiptAuthorityWithRunner({ source_commit: directChildren[0][0] }, run);
}

async function loadC3TAuthorityFromGit(root, expectedParent = sealedC3QIdentity.commit) {
	const requested = process.env.COUNTERSHAPE_GIT || "/usr/bin/git";
	if (!isAbsolute(requested)) throw new Error("COUNTERSHAPE_GIT must be absolute");
	const git = await realpath(requested);
	const env = {
		HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1",
		GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
		GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
	};
	const run = (args, accepted = [0]) => {
		const result = spawnSync(git, ["--no-replace-objects", ...args], {
			cwd: root, encoding: "buffer", timeout: 30_000, maxBuffer: 4 * 1024 * 1024, env,
		});
		if (result.error || result.signal || !accepted.includes(result.status) || (result.stderr?.length ?? 0) !== 0) {
			throw new Error(`git ${args[0]} failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
		}
		return { status: result.status, stdout: result.stdout ?? Buffer.alloc(0) };
	};
	return loadC3TAuthorityWithRunner(run, expectedParent);
}

function loadC3UAuthorityWithRunner(run, expectedParent = sealedC3TIdentity.commit) {
	const result = run(["rev-list", "--parents", "--ancestry-path", `${expectedParent}..HEAD`]);
	const text = decodeGitUTF8(result.stdout, "git C3U ancestry stdout");
	if (text.length === 0 || !text.endsWith("\n") || text.includes("\r")) {
		throw new Error("C3U ancestry is absent or not exact LF-framed output");
	}
	const rows = text.slice(0, -1).split("\n").map((line) => line.split(" "));
	if (rows.some((row) => row.length < 2 || row.some((oid) => !/^[0-9a-f]{40}$/u.test(oid)))) {
		throw new Error("C3U ancestry row is malformed");
	}
	const directChildren = rows.filter((row) => row.length === 2 && row[1] === expectedParent);
	if (directChildren.length !== 1) {
		throw new Error(`C3U direct-child cardinality ${directChildren.length}`);
	}
	return loadReceiptAuthorityWithRunner({ source_commit: directChildren[0][0] }, run);
}

async function loadC3UAuthorityFromGit(root, expectedParent = sealedC3TIdentity.commit) {
	const requested = process.env.COUNTERSHAPE_GIT || "/usr/bin/git";
	if (!isAbsolute(requested)) throw new Error("COUNTERSHAPE_GIT must be absolute");
	const git = await realpath(requested);
	const env = {
		HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1",
		GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
		GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
	};
	const run = (args, accepted = [0]) => {
		const result = spawnSync(git, ["--no-replace-objects", ...args], {
			cwd: root, encoding: "buffer", timeout: 30_000, maxBuffer: 4 * 1024 * 1024, env,
		});
		if (result.error || result.signal || !accepted.includes(result.status) || (result.stderr?.length ?? 0) !== 0) {
			throw new Error(`git ${args[0]} failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
		}
		return { status: result.status, stdout: result.stdout ?? Buffer.alloc(0) };
	};
	return loadC3UAuthorityWithRunner(run, expectedParent);
}

function sealedC3ReceiptProjection() {
	return {
		source_commit: sealedC3Identity.commit,
		source_tree: sealedC3Identity.tree,
		source_subject: sealedC3Identity.subject,
		secrets_override: true,
		coverage_complete_events: c3ClaimLabels.length,
		coverage_total_events: c3ClaimLabels.length,
		evidence: {
			git_note: {
				version: 1,
				blob_oid: sealedC3Identity.noteBlob,
				body_sha256: sealedC3Identity.noteBodySHA256,
			},
		},
	};
}

export async function verifyC3RSealedC3Note() {
	const receipt = sealedC3ReceiptProjection();
	const authority = await loadReceiptAuthorityFromGit(repositoryRoot, receipt);
	const errors = validateC3ReceiptAuthority(receipt, authority);
	if (authority.head_commit !== sealedC3Identity.commit) errors.push("sealed C3 is not current HEAD");
	if (errors.length > 0) throw new Error(`P07B-C C3R sealed-C3 authority mismatch: ${errors.join(", ")}`);
	console.log(`P07B-C C3R sealed-C3 note exact: commit=${authority.commit} tree=${authority.tree} note=${authority.note_blob_oid} claims=${c3ClaimLabels.length}`);
}

function sealedC3RReceiptProjection() {
	return {
		source_commit: sealedC3RIdentity.commit,
		source_tree: sealedC3RIdentity.tree,
		source_subject: sealedC3RIdentity.subject,
		secrets_override: true,
		coverage_complete_events: c3rClaimLabels.length,
		coverage_total_events: c3rClaimLabels.length,
		evidence: {
			git_note: {
				version: 1,
				blob_oid: sealedC3RIdentity.noteBlob,
				body_sha256: sealedC3RIdentity.noteBodySHA256,
			},
		},
	};
}

export async function verifyC3QSealedC3RNote() {
	const authority = await loadReceiptAuthorityFromGit(repositoryRoot, sealedC3RReceiptProjection());
	const errors = validateSealedC3RAuthority(authority, true);
	if (errors.length > 0) throw new Error(`P07B-C C3Q sealed-C3R authority mismatch: ${errors.join(", ")}`);
	console.log(`P07B-C C3Q sealed-C3R note exact: commit=${authority.commit} tree=${authority.tree} note=${authority.note_blob_oid} claims=${c3rClaimLabels.length}`);
}

function sealedC3QReceiptProjection() {
	return {
		source_commit: sealedC3QIdentity.commit,
		source_tree: sealedC3QIdentity.tree,
		source_subject: sealedC3QIdentity.subject,
		secrets_override: true,
		coverage_complete_events: c3qClaimLabels.length,
		coverage_total_events: c3qClaimLabels.length,
		evidence: {
			git_note: {
				version: 1,
				blob_oid: sealedC3QIdentity.noteBlob,
				body_sha256: sealedC3QIdentity.noteBodySHA256,
			},
		},
	};
}

export async function verifyC3TSealedC3QNote() {
	const authority = await loadReceiptAuthorityFromGit(repositoryRoot, sealedC3QReceiptProjection());
	const errors = validateC3TPredecessorAuthority(authority);
	if (errors.length > 0) throw new Error(`P07B-C C3T sealed-C3Q authority mismatch: ${errors.join(", ")}`);
	console.log(`P07B-C C3T sealed-C3Q note exact: commit=${authority.commit} tree=${authority.tree} note=${authority.note_blob_oid} claims=${c3qClaimLabels.length}`);
}

function sealedC3TReceiptProjection() {
	return {
		source_commit: sealedC3TIdentity.commit,
		source_tree: sealedC3TIdentity.tree,
		source_subject: sealedC3TIdentity.subject,
		secrets_override: true,
		coverage_complete_events: c3tClaimLabels.length,
		coverage_total_events: c3tClaimLabels.length,
		evidence: {
			git_note: {
				version: 1,
				blob_oid: sealedC3TIdentity.noteBlob,
				body_sha256: sealedC3TIdentity.noteBodySHA256,
			},
		},
	};
}

export async function verifyC3USealedC3TNote() {
	const authority = await loadReceiptAuthorityFromGit(repositoryRoot, sealedC3TReceiptProjection());
	const errors = validateC3UPredecessorAuthority(authority);
	if (errors.length > 0) throw new Error(`P07B-C C3U sealed-C3T authority mismatch: ${errors.join(", ")}`);
	console.log(`P07B-C C3U sealed-C3T note exact: commit=${authority.commit} tree=${authority.tree} note=${authority.note_blob_oid} claims=${c3tClaimLabels.length}`);
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

function parseC3PReceiptBlock(body) {
	const startCount = countOccurrences(body, c3pReceiptBlockStart);
	const endCount = countOccurrences(body, c3pReceiptBlockEnd);
	if (startCount === 0 && endCount === 0) return Object.freeze({ block: undefined, stripped: body });
	if (startCount !== 1 || endCount !== 1) {
		throw new Error(`receipt marker cardinality start=${startCount} end=${endCount}`);
	}
	const start = body.indexOf(c3pReceiptBlockStart);
	const end = body.indexOf(c3pReceiptBlockEnd);
	const standalone = (index, marker) =>
		(index === 0 || body[index - 1] === "\n") &&
		(index + marker.length === body.length || body[index + marker.length] === "\n");
	if (!standalone(start, c3pReceiptBlockStart) || !standalone(end, c3pReceiptBlockEnd)) {
		throw new Error("receipt markers must be standalone LF lines");
	}
	if (end <= start) throw new Error("receipt marker order");
	const blockEnd = end + c3pReceiptBlockEnd.length;
	const removalEnd = body[blockEnd] === "\n" ? blockEnd + 1 : blockEnd;
	return Object.freeze({
		block: body.slice(start, blockEnd),
		stripped: `${body.slice(0, start)}${body.slice(removalEnd)}`,
	});
}

function c3pReceiptHandoffBlock(receipt) {
	const rows = receipt.claims.map((claim) => `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`);
	const disclosureParagraphs = c3pReceiptDisclosureLines(receipt).flatMap((line) => [line, ""]);
	return [
		c3pReceiptBlockStart,
		"### P07B-C C3P source receipt map",
		"",
		`C3P source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`.`,
		`C3P strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\`; strict exit: \`0\`.`,
		"",
		...disclosureParagraphs,
		"| Claim label | Claim type | Verbatim grade |",
		"| --- | --- | --- |",
		...rows,
		c3pReceiptBlockEnd,
	].join("\n");
}

function c3pReceiptSpecificHandoffLines(receipt) {
	return Object.freeze([
		"### P07B-C C3P source receipt map",
		`C3P source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`.`,
		`C3P strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\`; strict exit: \`0\`.`,
		...c3pReceiptDisclosureLines(receipt),
		...receipt.claims.map((claim) => `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`),
	]);
}

function c3pReconciledStatusState(receipt) {
	return `- **Source receipt:** C3P source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`, is sealed, note-present, and strict-clean; every C3P source grade below is \`TREE-EXACT\`. This source-receipt record binds only that existing C3P source; the independently sealed C3PB descendant is outside these C3P grades.`;
}

function validateC3ReceiptEvidence(receipt) {
	const errors = [];
	const evidence = receipt.evidence;
	if (!evidence || typeof evidence !== "object" || Array.isArray(evidence) ||
		!isDeepStrictEqual(Object.keys(evidence).sort(), ["git_note", "html_report", "ledger_archive", "seal_redaction"])) {
		return ["local evidence field roster"];
	}
	const sourcePrefix = typeof receipt.source_commit === "string" ? receipt.source_commit.slice(0, 12) : "";
	const html = evidence.html_report;
	if (!html || typeof html !== "object" || Array.isArray(html) ||
		!isDeepStrictEqual(Object.keys(html).sort(), ["authority", "bytes", "path", "sha256"]) ||
		html.path !== `.countershape/evidence/p07b-c-c3-final-${sourcePrefix}.html` ||
		html.authority !== "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS" ||
		!Number.isSafeInteger(html.bytes) || html.bytes <= 0 || html.bytes > 16 * 1024 * 1024 ||
		!/^[0-9a-f]{64}$/u.test(html.sha256 ?? "")) {
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
		ledger.path !== ".didrun-history/2026-07-19-p07b-c-c3-final/.didrun/" ||
		ledger.authority !== "LOCAL_IGNORED_ARCHIVE_NOT_GIT_AUTHORITY" ||
		ledger.manifest_format !== "sorted-relative-posix-path-tab-size-tab-sha256-newline/v1" ||
		ledger.sealed_event_count !== c3ClaimLabels.length || ledger.claim_count !== c3ClaimLabels.length ||
		ledger.seal_count !== 1 || !Number.isSafeInteger(ledger.archive_session_event_count) ||
		ledger.archive_session_event_count < c3ClaimLabels.length ||
		ledger.archive_session_event_count > c3ClaimLabels.length + 4 ||
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
		!/^[0-9a-f]{40}$/u.test(note.blob_oid ?? "") || !/^[0-9a-f]{64}$/u.test(note.body_sha256 ?? "")) {
		errors.push("Git note evidence declaration");
	}
	const redaction = evidence.seal_redaction;
	if (!redaction || typeof redaction !== "object" || Array.isArray(redaction) ||
		!isDeepStrictEqual(Object.keys(redaction).sort(), [
			"finding_count", "finding_kinds", "finding_provenance", "structured_scan_provenance", "structured_staged_scan_findings",
		]) || !Number.isSafeInteger(redaction.finding_count) || redaction.finding_count < 0 || redaction.finding_count > 100_000 ||
		!Array.isArray(redaction.finding_kinds) ||
		!isDeepStrictEqual(redaction.finding_kinds, redaction.finding_count === 0 ? [] : ["high-entropy"]) ||
		redaction.finding_provenance !== "LOCAL_TERMINAL_OBSERVATION_NOT_GIT_NOTE" ||
		redaction.structured_staged_scan_findings !== 0 ||
		redaction.structured_scan_provenance !== "SEALED_C3_SUPPORTING_EVENT_16" ||
		receipt.secrets_override !== (redaction.finding_count > 0)) {
		errors.push("seal redaction evidence declaration");
	}
	return errors;
}

function validateC3ReceiptDeclaration(receipt) {
	const errors = [];
	const keys = [
		"claims", "coverage_complete_events", "coverage_total_events", "evidence", "schema_version",
		"secrets_override", "source_commit", "source_subject", "source_tree", "strict_claims_recorded_exact",
		"strict_claims_total", "strict_exit_code",
	];
	if (!receipt || typeof receipt !== "object" || Array.isArray(receipt) ||
		!isDeepStrictEqual(Object.keys(receipt).sort(), keys)) return ["root field roster"];
	if (receipt.schema_version !== "countershape/p07b-c-c3-receipt/v1") errors.push("schema version");
	if (!/^[0-9a-f]{40}$/u.test(receipt.source_commit ?? "") || /^0+$/u.test(receipt.source_commit ?? "")) errors.push("source commit");
	if (!/^[0-9a-f]{40}$/u.test(receipt.source_tree ?? "") || /^0+$/u.test(receipt.source_tree ?? "")) errors.push("source tree");
	if (receipt.source_commit !== sealedC3Identity.commit) errors.push("sealed C3 source commit identity");
	if (receipt.source_tree !== sealedC3Identity.tree) errors.push("sealed C3 source tree identity");
	if (receipt.source_subject !== c3SourceSubject) errors.push("source subject");
	if (receipt.secrets_override !== true) errors.push("sealed C3 secrets override disclosure");
	if (receipt.strict_exit_code !== 0) errors.push("strict exit code");
	if (receipt.strict_claims_recorded_exact !== c3ClaimLabels.length || receipt.strict_claims_total !== c3ClaimLabels.length) {
		errors.push("strict claim counts");
	}
	if (receipt.coverage_complete_events !== c3ClaimLabels.length || receipt.coverage_total_events !== c3ClaimLabels.length) {
		errors.push("coverage counts");
	}
	if (!Array.isArray(receipt.claims) || receipt.claims.length !== c3ClaimLabels.length) {
		errors.push("claim roster length");
	} else {
		for (let index = 0; index < c3ClaimLabels.length; index += 1) {
			const claim = receipt.claims[index];
			if (!claim || typeof claim !== "object" || Array.isArray(claim) ||
				!isDeepStrictEqual(Object.keys(claim).sort(), ["grade", "label", "supporting_event_index", "type"]) ||
				claim.supporting_event_index !== index || claim.label !== c3ClaimLabels[index] ||
				claim.type !== c3ClaimTypes[index] || claim.grade !== "TREE-EXACT") {
				errors.push(`claim ${index + 1}`);
			}
		}
	}
	errors.push(...validateC3ReceiptEvidence(receipt));
	if (receipt.evidence?.git_note?.blob_oid !== sealedC3Identity.noteBlob ||
		receipt.evidence?.git_note?.body_sha256 !== sealedC3Identity.noteBodySHA256) {
		errors.push("sealed C3 Git-note identity");
	}
	return errors;
}

function validateC3ClaimPreview(index, argv) {
	const expected = c3ExpectedClaimArgv[index];
	if (!expected || !Array.isArray(argv) || argv.length !== expected.length ||
		argv.some((argument) => typeof argument !== "string" || argument.length === 0 || argument.length > 4096 || /[\u0000-\u001f\u007f]/u.test(argument))) {
		return false;
	}
	const redactedPositions = new Set([2, 4, 5, 6, 7, 8, 31, 32, 33, 35, 36]);
	const redacted = /^(?:«redacted:high-entropy»)(?:\.countershape\/[^\u0000-\u001f\u007f]+|\.?:?«redacted:high-entropy»)?$/u;
	for (let position = 0; position < c3HermeticArgvPrefix.length; position += 1) {
		if (argv[position] === expected[position]) continue;
		if (!redactedPositions.has(position) || !redacted.test(argv[position])) return false;
	}
	const tail = argv.slice(c3HermeticArgvPrefix.length);
	const expectedTail = expected.slice(c3HermeticArgvPrefix.length);
	if (isDeepStrictEqual(tail, expectedTail)) return true;
	if (index !== 7) return false;
	const noteTail = [...expectedTail];
	const patternIndex = noteTail.indexOf(c3RaceTestPattern);
	if (patternIndex === -1 || noteTail.lastIndexOf(c3RaceTestPattern) !== patternIndex) return false;
	noteTail[patternIndex] = c3RaceNotePreviewPattern;
	return isDeepStrictEqual(tail, noteTail);
}

function validateC3RClaimPreview(index, argv) {
	const expected = c3rExpectedClaimArgv[index];
	if (!expected || !Array.isArray(argv) || argv.length !== expected.length ||
		argv.some((argument) => typeof argument !== "string" || argument.length === 0 || argument.length > 4096 || /[\u0000-\u001f\u007f]/u.test(argument))) {
		return false;
	}
	const redactedPositions = new Set([2, 4, 5, 6, 7, 8, 31, 32, 33, 35, 36]);
	const redacted = /^(?:«redacted:high-entropy»)(?:\.countershape\/[^\u0000-\u001f\u007f]+|\.?:?«redacted:high-entropy»)?$/u;
	for (let position = 0; position < c3rHermeticArgvPrefix.length; position += 1) {
		if (argv[position] === expected[position]) continue;
		if (!redactedPositions.has(position) || !redacted.test(argv[position])) return false;
	}
	return isDeepStrictEqual(
		argv.slice(c3rHermeticArgvPrefix.length),
		expected.slice(c3rHermeticArgvPrefix.length),
	);
}

function validateC3RSourceDiff(authority) {
	const errors = [];
	const entries = authority.source_diff;
	if (!Array.isArray(entries)) return ["source diff roster"];
	const paths = entries.map((entry) => entry?.path);
	if (new Set(paths).size !== paths.length || !isDeepStrictEqual(paths, requiredC3RPaths)) {
		errors.push("source diff exact path roster");
	}
	const addedPath = c3rStatusPath;
	let added = 0;
	let modified = 0;
	for (const path of requiredC3RPaths) {
		const entry = entries.find((candidate) => candidate?.path === path);
		if (!entry || !isDeepStrictEqual(Object.keys(entry).sort(), ["new_mode", "new_oid", "old_mode", "old_oid", "path", "status"])) {
			errors.push(`source diff row ${path}`);
			continue;
		}
		const objectPattern = /^(?:[0-9a-f]{40}|[0-9a-f]{64})$/u;
		const newObjectValid = objectPattern.test(entry.new_oid) && !/^0+$/u.test(entry.new_oid);
		if (path === addedPath) {
			added += 1;
			if (entry.status !== "A" || entry.old_mode !== "000000" || entry.new_mode !== "100644" ||
				!objectPattern.test(entry.old_oid) || !/^0+$/u.test(entry.old_oid) || !newObjectValid ||
				entry.old_oid.length !== entry.new_oid.length) {
				errors.push(`source diff added row ${path}`);
			}
		} else {
			modified += 1;
			if (entry.status !== "M" || entry.old_mode !== "100644" || entry.new_mode !== "100644" ||
				!objectPattern.test(entry.old_oid) || /^0+$/u.test(entry.old_oid) || !newObjectValid ||
				entry.old_oid.length !== entry.new_oid.length || entry.old_oid === entry.new_oid) {
				errors.push(`source diff modified row ${path}`);
			}
		}
	}
	if (added !== 1 || modified !== 6) errors.push(`source diff status partition A=${added} M=${modified}`);
	return errors;
}

function validateC3RAuthority(authority, expectedParent = sealedC3Identity.commit) {
	const errors = [];
	if (!authority || typeof authority !== "object" || Array.isArray(authority)) return ["missing Git/didrun authority"];
	if (!/^[0-9a-f]{40}$/u.test(authority.commit ?? "") || /^0+$/u.test(authority.commit ?? "")) errors.push("source commit");
	if (!/^[0-9a-f]{40}$/u.test(authority.tree ?? "") || /^0+$/u.test(authority.tree ?? "")) errors.push("source tree");
	if (authority.parent !== expectedParent || !isDeepStrictEqual(authority.parents, [expectedParent])) {
		errors.push("source parent");
	}
	errors.push(...validateC3RSourceDiff(authority));
	if (authority.subject !== c3rSourceSubject) errors.push("source commit subject");
	if (authority.ancestor_of_head !== true) errors.push("source commit is not an ancestor of HEAD");
	if (authority.note_type !== "blob") errors.push("didrun note object type");
	if (!/^[0-9a-f]{40}$/u.test(authority.note_blob_oid ?? "")) errors.push("didrun note object identity");
	if (!/^[0-9a-f]{64}$/u.test(authority.note_body_sha256 ?? "")) errors.push("didrun note body digest");
	const note = authority.note;
	if (!note || typeof note !== "object" || Array.isArray(note) ||
		!isDeepStrictEqual(Object.keys(note).sort(), ["claims", "commit", "coverage", "secrets_override", "tree", "version"])) {
		errors.push("missing or malformed parsed didrun Git note");
		return errors;
	}
	if (note.version !== 1) errors.push("didrun note version");
	if (note.commit !== authority.commit) errors.push("didrun note commit");
	if (note.tree !== authority.tree) errors.push("didrun note tree");
	if (typeof note.secrets_override !== "boolean") errors.push("didrun note secrets override disclosure");
	if (!Array.isArray(note.claims) || note.claims.length !== c3rClaimLabels.length) {
		errors.push("didrun note claim roster length");
	} else {
		for (let index = 0; index < c3rClaimLabels.length; index += 1) {
			const recorded = note.claims[index];
			const claim = recorded?.claim;
			if (!recorded || typeof recorded !== "object" || Array.isArray(recorded) ||
				!isDeepStrictEqual(Object.keys(recorded).sort(), ["claim", "delta", "exit_code", "grade", "reason", "supporting_event_index"]) ||
				!claim || typeof claim !== "object" || Array.isArray(claim) ||
				!isDeepStrictEqual(Object.keys(claim).sort(), ["argv_preview", "ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) ||
				claim.label !== c3rClaimLabels[index] || claim.ctype !== c3rClaimTypes[index] ||
				claim.declared_at_index !== index || !isDeepStrictEqual(claim.event_indices, [index]) ||
				!isDeepStrictEqual(claim.pathspecs, []) || recorded.supporting_event_index !== index ||
				recorded.grade !== "tree-exact" || recorded.exit_code !== 0 ||
				recorded.reason !== "self-stable command ran against the sealed tree" || !isDeepStrictEqual(recorded.delta, [])) {
				errors.push(`didrun note claim ${index + 1}`);
			}
			if (!validateC3RClaimPreview(index, claim?.argv_preview)) errors.push(`didrun note claim ${index + 1} argv`);
		}
	}
	if (!note.coverage || typeof note.coverage !== "object" || Array.isArray(note.coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage).sort(), ["by_coverage", "total_events"]) ||
		note.coverage.total_events !== c3rClaimLabels.length ||
		!note.coverage.by_coverage || typeof note.coverage.by_coverage !== "object" || Array.isArray(note.coverage.by_coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage.by_coverage), ["complete"]) ||
		note.coverage.by_coverage.complete !== c3rClaimLabels.length) {
		errors.push("didrun note exact event coverage");
	}
	return errors;
}

function validateSealedC3RAuthority(authority, requireCurrentHead = false) {
	const errors = validateC3RAuthority(authority, sealedC3Identity.commit);
	if (authority?.commit !== sealedC3RIdentity.commit) errors.push("sealed C3R commit identity");
	if (authority?.tree !== sealedC3RIdentity.tree) errors.push("sealed C3R tree identity");
	if (authority?.note_type !== sealedC3RIdentity.noteType) errors.push("sealed C3R note type");
	if (authority?.note_blob_oid !== sealedC3RIdentity.noteBlob) errors.push("sealed C3R note object identity");
	if (authority?.note_body_sha256 !== sealedC3RIdentity.noteBodySHA256) errors.push("sealed C3R note body digest");
	if (authority?.note?.secrets_override !== true) errors.push("sealed C3R secrets override disclosure");
	if (requireCurrentHead && authority?.head_commit !== authority?.commit) errors.push("sealed C3R is not current HEAD");
	return errors;
}

function validateC3QClaimPreview(index, argv) {
	const expected = c3qExpectedClaimArgv[index];
	if (!expected || !Array.isArray(argv) || argv.length !== expected.length ||
		argv.some((argument) => typeof argument !== "string" || argument.length === 0 || argument.length > 4096 || /[\u0000-\u001f\u007f]/u.test(argument))) {
		return false;
	}
	const redactedPositions = new Set([2, 4, 5, 6, 7, 8, 31, 32, 33, 35, 36]);
	const redacted = /^(?:«redacted:high-entropy»)(?:\.countershape\/[^\u0000-\u001f\u007f]+|\.?:?«redacted:high-entropy»)?$/u;
	for (let position = 0; position < c3qHermeticArgvPrefix.length; position += 1) {
		if (argv[position] === expected[position]) continue;
		if (!redactedPositions.has(position) || !redacted.test(argv[position])) return false;
	}
	return isDeepStrictEqual(
		argv.slice(c3qHermeticArgvPrefix.length),
		expected.slice(c3qHermeticArgvPrefix.length),
	);
}

function validateC3QSourceDiff(authority) {
	const errors = [];
	const entries = authority.source_diff;
	if (!Array.isArray(entries)) return ["source diff roster"];
	const paths = entries.map((entry) => entry?.path);
	if (new Set(paths).size !== paths.length || !isDeepStrictEqual(paths, requiredC3QPaths)) {
		errors.push("source diff exact path roster");
	}
	let added = 0;
	let modified = 0;
	for (const path of requiredC3QPaths) {
		const entry = entries.find((candidate) => candidate?.path === path);
		if (!entry || !isDeepStrictEqual(Object.keys(entry).sort(), ["new_mode", "new_oid", "old_mode", "old_oid", "path", "status"])) {
			errors.push(`source diff row ${path}`);
			continue;
		}
		const objectPattern = /^(?:[0-9a-f]{40}|[0-9a-f]{64})$/u;
		const newObjectValid = objectPattern.test(entry.new_oid) && !/^0+$/u.test(entry.new_oid);
		if (path === c3qStatusPath) {
			added += 1;
			if (entry.status !== "A" || entry.old_mode !== "000000" || entry.new_mode !== "100644" ||
				!objectPattern.test(entry.old_oid) || !/^0+$/u.test(entry.old_oid) || !newObjectValid ||
				entry.old_oid.length !== entry.new_oid.length) {
				errors.push(`source diff added row ${path}`);
			}
		} else {
			modified += 1;
			if (entry.status !== "M" || entry.old_mode !== "100644" || entry.new_mode !== "100644" ||
				!objectPattern.test(entry.old_oid) || /^0+$/u.test(entry.old_oid) || !newObjectValid ||
				entry.old_oid.length !== entry.new_oid.length || entry.old_oid === entry.new_oid) {
				errors.push(`source diff modified row ${path}`);
			}
		}
	}
	if (added !== 1 || modified !== 6) errors.push(`source diff status partition A=${added} M=${modified}`);
	return errors;
}

function validateC3QAuthority(authority, expectedParent = sealedC3RIdentity.commit) {
	const errors = [];
	if (!authority || typeof authority !== "object" || Array.isArray(authority)) return ["missing Git/didrun authority"];
	if (!/^[0-9a-f]{40}$/u.test(authority.commit ?? "") || /^0+$/u.test(authority.commit ?? "")) errors.push("source commit");
	if (!/^[0-9a-f]{40}$/u.test(authority.tree ?? "") || /^0+$/u.test(authority.tree ?? "")) errors.push("source tree");
	if (authority.parent !== expectedParent || !isDeepStrictEqual(authority.parents, [expectedParent])) errors.push("source parent");
	errors.push(...validateC3QSourceDiff(authority));
	if (authority.subject !== c3qSourceSubject) errors.push("source commit subject");
	if (authority.ancestor_of_head !== true) errors.push("source commit is not an ancestor of HEAD");
	if (authority.note_type !== "blob") errors.push("didrun note object type");
	if (!/^[0-9a-f]{40}$/u.test(authority.note_blob_oid ?? "")) errors.push("didrun note object identity");
	if (!/^[0-9a-f]{64}$/u.test(authority.note_body_sha256 ?? "")) errors.push("didrun note body digest");
	const note = authority.note;
	if (!note || typeof note !== "object" || Array.isArray(note) ||
		!isDeepStrictEqual(Object.keys(note).sort(), ["claims", "commit", "coverage", "secrets_override", "tree", "version"])) {
		errors.push("missing or malformed parsed didrun Git note");
		return errors;
	}
	if (note.version !== 1) errors.push("didrun note version");
	if (note.commit !== authority.commit) errors.push("didrun note commit");
	if (note.tree !== authority.tree) errors.push("didrun note tree");
	if (typeof note.secrets_override !== "boolean") errors.push("didrun note secrets override disclosure");
	if (!Array.isArray(note.claims) || note.claims.length !== c3qClaimLabels.length) {
		errors.push("didrun note claim roster length");
	} else {
		for (let index = 0; index < c3qClaimLabels.length; index += 1) {
			const recorded = note.claims[index];
			const claim = recorded?.claim;
			if (!recorded || typeof recorded !== "object" || Array.isArray(recorded) ||
				!isDeepStrictEqual(Object.keys(recorded).sort(), ["claim", "delta", "exit_code", "grade", "reason", "supporting_event_index"]) ||
				!claim || typeof claim !== "object" || Array.isArray(claim) ||
				!isDeepStrictEqual(Object.keys(claim).sort(), ["argv_preview", "ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) ||
				claim.label !== c3qClaimLabels[index] || claim.ctype !== c3qClaimTypes[index] ||
				claim.declared_at_index !== index || !isDeepStrictEqual(claim.event_indices, [index]) ||
				!isDeepStrictEqual(claim.pathspecs, []) || recorded.supporting_event_index !== index ||
				recorded.grade !== "tree-exact" || recorded.exit_code !== 0 ||
				recorded.reason !== "self-stable command ran against the sealed tree" || !isDeepStrictEqual(recorded.delta, [])) {
				errors.push(`didrun note claim ${index + 1}`);
			}
			if (!validateC3QClaimPreview(index, claim?.argv_preview)) errors.push(`didrun note claim ${index + 1} argv`);
		}
	}
	if (!note.coverage || typeof note.coverage !== "object" || Array.isArray(note.coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage).sort(), ["by_coverage", "total_events"]) ||
		note.coverage.total_events !== c3qClaimLabels.length ||
		!note.coverage.by_coverage || typeof note.coverage.by_coverage !== "object" || Array.isArray(note.coverage.by_coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage.by_coverage), ["complete"]) ||
		note.coverage.by_coverage.complete !== c3qClaimLabels.length) {
		errors.push("didrun note exact event coverage");
	}
	return errors;
}

function validateSealedC3QAuthority(authority, requireCurrentHead = false) {
	const errors = validateC3QAuthority(authority, sealedC3RIdentity.commit);
	if (authority?.commit !== sealedC3QIdentity.commit) errors.push("sealed C3Q commit identity");
	if (authority?.tree !== sealedC3QIdentity.tree) errors.push("sealed C3Q tree identity");
	if (authority?.note_type !== sealedC3QIdentity.noteType) errors.push("sealed C3Q note type");
	if (authority?.note_blob_oid !== sealedC3QIdentity.noteBlob) errors.push("sealed C3Q note object identity");
	if (authority?.note_body_sha256 !== sealedC3QIdentity.noteBodySHA256) errors.push("sealed C3Q note body digest");
	if (authority?.note?.secrets_override !== true) errors.push("sealed C3Q secrets override disclosure");
	if (requireCurrentHead && authority?.head_commit !== authority?.commit) errors.push("sealed C3Q is not current HEAD");
	return errors;
}

function validateC3TClaimPreview(index, argv) {
	const expected = c3tExpectedClaimArgv[index];
	if (!expected || !Array.isArray(argv) || argv.length !== expected.length ||
		argv.some((argument) => typeof argument !== "string" || argument.length === 0 || argument.length > 4096 || /[\u0000-\u001f\u007f]/u.test(argument))) {
		return false;
	}
	const redactedPositions = new Set([2, 4, 5, 6, 7, 8, 31, 32, 33, 35, 36]);
	const redacted = /^(?:«redacted:high-entropy»)(?:\.countershape\/[^\u0000-\u001f\u007f]+|\.?:?«redacted:high-entropy»)?$/u;
	for (let position = 0; position < c3tHermeticArgvPrefix.length; position += 1) {
		if (argv[position] === expected[position]) continue;
		if (!redactedPositions.has(position) || !redacted.test(argv[position])) return false;
	}
	return isDeepStrictEqual(
		argv.slice(c3tHermeticArgvPrefix.length),
		expected.slice(c3tHermeticArgvPrefix.length),
	);
}

function validateC3TSourceDiff(authority) {
	const errors = [];
	const entries = authority.source_diff;
	if (!Array.isArray(entries)) return ["source diff roster"];
	const paths = entries.map((entry) => entry?.path);
	if (new Set(paths).size !== paths.length || !isDeepStrictEqual(paths, requiredC3TPaths)) {
		errors.push("source diff exact path roster");
	}
	let added = 0;
	let modified = 0;
	for (const path of requiredC3TPaths) {
		const entry = entries.find((candidate) => candidate?.path === path);
		if (!entry || !isDeepStrictEqual(Object.keys(entry).sort(), ["new_mode", "new_oid", "old_mode", "old_oid", "path", "status"])) {
			errors.push(`source diff row ${path}`);
			continue;
		}
		const objectPattern = /^(?:[0-9a-f]{40}|[0-9a-f]{64})$/u;
		const newObjectValid = objectPattern.test(entry.new_oid) && !/^0+$/u.test(entry.new_oid);
		if (path === c3tStatusPath) {
			added += 1;
			if (entry.status !== "A" || entry.old_mode !== "000000" || entry.new_mode !== "100644" ||
				!objectPattern.test(entry.old_oid) || !/^0+$/u.test(entry.old_oid) || !newObjectValid ||
				entry.old_oid.length !== entry.new_oid.length) {
				errors.push(`source diff added row ${path}`);
			}
		} else {
			modified += 1;
			if (entry.status !== "M" || entry.old_mode !== "100644" || entry.new_mode !== "100644" ||
				!objectPattern.test(entry.old_oid) || /^0+$/u.test(entry.old_oid) || !newObjectValid ||
				entry.old_oid.length !== entry.new_oid.length || entry.old_oid === entry.new_oid) {
				errors.push(`source diff modified row ${path}`);
			}
		}
	}
	if (added !== 1 || modified !== 6) errors.push(`source diff status partition A=${added} M=${modified}`);
	return errors;
}

function validateC3TAuthority(authority, expectedParent = sealedC3QIdentity.commit) {
	const errors = [];
	if (!authority || typeof authority !== "object" || Array.isArray(authority)) return ["missing Git/didrun authority"];
	if (!/^[0-9a-f]{40}$/u.test(authority.commit ?? "") || /^0+$/u.test(authority.commit ?? "")) errors.push("source commit");
	if (!/^[0-9a-f]{40}$/u.test(authority.tree ?? "") || /^0+$/u.test(authority.tree ?? "")) errors.push("source tree");
	if (authority.parent !== expectedParent || !isDeepStrictEqual(authority.parents, [expectedParent])) errors.push("source parent");
	errors.push(...validateC3TSourceDiff(authority));
	if (authority.subject !== c3tSourceSubject) errors.push("source commit subject");
	if (authority.ancestor_of_head !== true) errors.push("source commit is not an ancestor of HEAD");
	if (authority.note_type !== "blob") errors.push("didrun note object type");
	if (!/^[0-9a-f]{40}$/u.test(authority.note_blob_oid ?? "")) errors.push("didrun note object identity");
	if (!/^[0-9a-f]{64}$/u.test(authority.note_body_sha256 ?? "")) errors.push("didrun note body digest");
	const note = authority.note;
	if (!note || typeof note !== "object" || Array.isArray(note) ||
		!isDeepStrictEqual(Object.keys(note).sort(), ["claims", "commit", "coverage", "secrets_override", "tree", "version"])) {
		errors.push("missing or malformed parsed didrun Git note");
		return errors;
	}
	if (note.version !== 1) errors.push("didrun note version");
	if (note.commit !== authority.commit) errors.push("didrun note commit");
	if (note.tree !== authority.tree) errors.push("didrun note tree");
	if (typeof note.secrets_override !== "boolean") errors.push("didrun note secrets override disclosure");
	if (!Array.isArray(note.claims) || note.claims.length !== c3tClaimLabels.length) {
		errors.push("didrun note claim roster length");
	} else {
		for (let index = 0; index < c3tClaimLabels.length; index += 1) {
			const recorded = note.claims[index];
			const claim = recorded?.claim;
			if (!recorded || typeof recorded !== "object" || Array.isArray(recorded) ||
				!isDeepStrictEqual(Object.keys(recorded).sort(), ["claim", "delta", "exit_code", "grade", "reason", "supporting_event_index"]) ||
				!claim || typeof claim !== "object" || Array.isArray(claim) ||
				!isDeepStrictEqual(Object.keys(claim).sort(), ["argv_preview", "ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) ||
				claim.label !== c3tClaimLabels[index] || claim.ctype !== c3tClaimTypes[index] ||
				claim.declared_at_index !== index || !isDeepStrictEqual(claim.event_indices, [index]) ||
				!isDeepStrictEqual(claim.pathspecs, []) || recorded.supporting_event_index !== index ||
				recorded.grade !== "tree-exact" || recorded.exit_code !== 0 ||
				recorded.reason !== "self-stable command ran against the sealed tree" || !isDeepStrictEqual(recorded.delta, [])) {
				errors.push(`didrun note claim ${index + 1}`);
			}
			if (!validateC3TClaimPreview(index, claim?.argv_preview)) errors.push(`didrun note claim ${index + 1} argv`);
		}
	}
	if (!note.coverage || typeof note.coverage !== "object" || Array.isArray(note.coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage).sort(), ["by_coverage", "total_events"]) ||
		note.coverage.total_events !== c3tClaimLabels.length ||
		!note.coverage.by_coverage || typeof note.coverage.by_coverage !== "object" || Array.isArray(note.coverage.by_coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage.by_coverage), ["complete"]) ||
		note.coverage.by_coverage.complete !== c3tClaimLabels.length) {
		errors.push("didrun note exact event coverage");
	}
	return errors;
}

function validateC3TPredecessorAuthority(authority) {
	return validateSealedC3QAuthority(authority, true);
}

function validateSealedC3TAuthority(authority, requireCurrentHead = false) {
	const errors = validateC3TAuthority(authority, sealedC3QIdentity.commit);
	if (authority?.commit !== sealedC3TIdentity.commit) errors.push("sealed C3T commit identity");
	if (authority?.tree !== sealedC3TIdentity.tree) errors.push("sealed C3T tree identity");
	if (authority?.note_type !== sealedC3TIdentity.noteType) errors.push("sealed C3T note type");
	if (authority?.note_blob_oid !== sealedC3TIdentity.noteBlob) errors.push("sealed C3T note object identity");
	if (authority?.note_body_sha256 !== sealedC3TIdentity.noteBodySHA256) errors.push("sealed C3T note body digest");
	if (authority?.note?.secrets_override !== true) errors.push("sealed C3T secrets override disclosure");
	if (requireCurrentHead && authority?.head_commit !== authority?.commit) errors.push("sealed C3T is not current HEAD");
	return errors;
}

function validateC3UClaimPreview(index, argv) {
	const expected = c3uExpectedClaimArgv[index];
	if (!expected || !Array.isArray(argv) || argv.length !== expected.length ||
		argv.some((argument) => typeof argument !== "string" || argument.length === 0 || argument.length > 4096 || /[\u0000-\u001f\u007f]/u.test(argument))) {
		return false;
	}
	const redactedPositions = new Set([2, 4, 5, 6, 7, 8, 31, 32, 33, 35, 36]);
	const redacted = /^(?:«redacted:high-entropy»)(?:\.countershape\/[^\u0000-\u001f\u007f]+|\.?:?«redacted:high-entropy»)?$/u;
	for (let position = 0; position < c3uHermeticArgvPrefix.length; position += 1) {
		if (argv[position] === expected[position]) continue;
		if (!redactedPositions.has(position) || !redacted.test(argv[position])) return false;
	}
	return isDeepStrictEqual(
		argv.slice(c3uHermeticArgvPrefix.length),
		expected.slice(c3uHermeticArgvPrefix.length),
	);
}

function validateC3USourceDiff(authority) {
	const errors = [];
	const entries = authority.source_diff;
	if (!Array.isArray(entries)) return ["source diff roster"];
	const paths = entries.map((entry) => entry?.path);
	if (new Set(paths).size !== paths.length || !isDeepStrictEqual(paths, requiredC3UPaths)) {
		errors.push("source diff exact path roster");
	}
	let added = 0;
	let modified = 0;
	for (const path of requiredC3UPaths) {
		const entry = entries.find((candidate) => candidate?.path === path);
		if (!entry || !isDeepStrictEqual(Object.keys(entry).sort(), ["new_mode", "new_oid", "old_mode", "old_oid", "path", "status"])) {
			errors.push(`source diff row ${path}`);
			continue;
		}
		const objectPattern = /^(?:[0-9a-f]{40}|[0-9a-f]{64})$/u;
		const newObjectValid = objectPattern.test(entry.new_oid) && !/^0+$/u.test(entry.new_oid);
		if (path === c3uStatusPath) {
			added += 1;
			if (entry.status !== "A" || entry.old_mode !== "000000" || entry.new_mode !== "100644" ||
				!objectPattern.test(entry.old_oid) || !/^0+$/u.test(entry.old_oid) || !newObjectValid ||
				entry.old_oid.length !== entry.new_oid.length) {
				errors.push(`source diff added row ${path}`);
			}
		} else {
			modified += 1;
			if (entry.status !== "M" || entry.old_mode !== "100644" || entry.new_mode !== "100644" ||
				!objectPattern.test(entry.old_oid) || /^0+$/u.test(entry.old_oid) || !newObjectValid ||
				entry.old_oid.length !== entry.new_oid.length || entry.old_oid === entry.new_oid) {
				errors.push(`source diff modified row ${path}`);
			}
		}
	}
	if (added !== 1 || modified !== 6) errors.push(`source diff status partition A=${added} M=${modified}`);
	return errors;
}

function validateC3UAuthority(authority, expectedParent = sealedC3TIdentity.commit) {
	const errors = [];
	if (!authority || typeof authority !== "object" || Array.isArray(authority)) return ["missing Git/didrun authority"];
	if (!/^[0-9a-f]{40}$/u.test(authority.commit ?? "") || /^0+$/u.test(authority.commit ?? "")) errors.push("source commit");
	if (!/^[0-9a-f]{40}$/u.test(authority.tree ?? "") || /^0+$/u.test(authority.tree ?? "")) errors.push("source tree");
	if (authority.parent !== expectedParent || !isDeepStrictEqual(authority.parents, [expectedParent])) errors.push("source parent");
	errors.push(...validateC3USourceDiff(authority));
	if (authority.subject !== c3uSourceSubject) errors.push("source commit subject");
	if (authority.ancestor_of_head !== true) errors.push("source commit is not an ancestor of HEAD");
	if (authority.note_type !== "blob") errors.push("didrun note object type");
	if (!/^[0-9a-f]{40}$/u.test(authority.note_blob_oid ?? "")) errors.push("didrun note object identity");
	if (!/^[0-9a-f]{64}$/u.test(authority.note_body_sha256 ?? "")) errors.push("didrun note body digest");
	const note = authority.note;
	if (!note || typeof note !== "object" || Array.isArray(note) ||
		!isDeepStrictEqual(Object.keys(note).sort(), ["claims", "commit", "coverage", "secrets_override", "tree", "version"])) {
		errors.push("missing or malformed parsed didrun Git note");
		return errors;
	}
	if (note.version !== 1) errors.push("didrun note version");
	if (note.commit !== authority.commit) errors.push("didrun note commit");
	if (note.tree !== authority.tree) errors.push("didrun note tree");
	if (typeof note.secrets_override !== "boolean") errors.push("didrun note secrets override disclosure");
	if (!Array.isArray(note.claims) || note.claims.length !== c3uClaimLabels.length) {
		errors.push("didrun note claim roster length");
	} else {
		for (let index = 0; index < c3uClaimLabels.length; index += 1) {
			const recorded = note.claims[index];
			const claim = recorded?.claim;
			if (!recorded || typeof recorded !== "object" || Array.isArray(recorded) ||
				!isDeepStrictEqual(Object.keys(recorded).sort(), ["claim", "delta", "exit_code", "grade", "reason", "supporting_event_index"]) ||
				!claim || typeof claim !== "object" || Array.isArray(claim) ||
				!isDeepStrictEqual(Object.keys(claim).sort(), ["argv_preview", "ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) ||
				claim.label !== c3uClaimLabels[index] || claim.ctype !== c3uClaimTypes[index] ||
				claim.declared_at_index !== index || !isDeepStrictEqual(claim.event_indices, [index]) ||
				!isDeepStrictEqual(claim.pathspecs, []) || recorded.supporting_event_index !== index ||
				recorded.grade !== "tree-exact" || recorded.exit_code !== 0 ||
				recorded.reason !== "self-stable command ran against the sealed tree" || !isDeepStrictEqual(recorded.delta, [])) {
				errors.push(`didrun note claim ${index + 1}`);
			}
			if (!validateC3UClaimPreview(index, claim?.argv_preview)) errors.push(`didrun note claim ${index + 1} argv`);
		}
	}
	if (!note.coverage || typeof note.coverage !== "object" || Array.isArray(note.coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage).sort(), ["by_coverage", "total_events"]) ||
		note.coverage.total_events !== c3uClaimLabels.length ||
		!note.coverage.by_coverage || typeof note.coverage.by_coverage !== "object" || Array.isArray(note.coverage.by_coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage.by_coverage), ["complete"]) ||
		note.coverage.by_coverage.complete !== c3uClaimLabels.length) {
		errors.push("didrun note exact event coverage");
	}
	return errors;
}

function validateC3UPredecessorAuthority(authority) {
	return validateSealedC3TAuthority(authority, true);
}

function validateC3BPredecessorAuthority(authority, expectedParent = sealedC3TIdentity.commit) {
	const errors = validateC3UAuthority(authority, expectedParent);
	if (authority?.head_commit !== authority?.commit) errors.push("C3U is not current HEAD");
	return errors;
}

function validateC3SourceDiff(authority) {
	const errors = [];
	if (!isDeepStrictEqual(authority.parents, [sealedC3SIdentity.commit])) errors.push("source parents");
	const entries = authority.source_diff;
	if (!Array.isArray(entries)) return [...errors, "source diff roster"];
	const paths = entries.map((entry) => entry?.path);
	if (new Set(paths).size !== paths.length || !isDeepStrictEqual(paths, requiredC3Paths)) {
		errors.push("source diff exact path roster");
	}
	let added = 0;
	let modified = 0;
	for (const path of requiredC3Paths) {
		const entry = entries.find((candidate) => candidate?.path === path);
		if (!entry || !isDeepStrictEqual(Object.keys(entry).sort(), ["new_mode", "new_oid", "old_mode", "old_oid", "path", "status"])) {
			errors.push(`source diff row ${path}`);
			continue;
		}
		const objectPattern = /^(?:[0-9a-f]{40}|[0-9a-f]{64})$/u;
		const newObjectValid = objectPattern.test(entry.new_oid) && !/^0+$/u.test(entry.new_oid);
		if (requiredC3AddedPaths.has(path)) {
			added += 1;
			if (entry.status !== "A" || entry.old_mode !== "000000" || entry.new_mode !== "100644" ||
				!objectPattern.test(entry.old_oid) || !/^0+$/u.test(entry.old_oid) || !newObjectValid ||
				entry.old_oid.length !== entry.new_oid.length) {
				errors.push(`source diff added row ${path}`);
			}
		} else {
			modified += 1;
			if (entry.status !== "M" || entry.old_mode !== "100644" || entry.new_mode !== "100644" ||
				!objectPattern.test(entry.old_oid) || /^0+$/u.test(entry.old_oid) || !newObjectValid ||
				entry.old_oid.length !== entry.new_oid.length || entry.old_oid === entry.new_oid) {
				errors.push(`source diff modified row ${path}`);
			}
		}
	}
	if (added !== 20 || modified !== 20) errors.push(`source diff status partition A=${added} M=${modified}`);
	return errors;
}

function validateC3ReceiptAuthority(receipt, authority) {
	const errors = [];
	if (!authority || typeof authority !== "object" || Array.isArray(authority)) return ["missing Git/didrun authority"];
	if (authority.commit !== receipt.source_commit) errors.push("source commit does not equal admitted Git commit");
	if (authority.tree !== receipt.source_tree) errors.push("source tree does not equal Git commit tree");
	if (authority.parent !== sealedC3SIdentity.commit) errors.push("source parent");
	errors.push(...validateC3SourceDiff(authority));
	if (authority.subject !== receipt.source_subject) errors.push("source commit subject");
	if (authority.ancestor_of_head !== true) errors.push("source commit is not an ancestor of HEAD");
	if (authority.note_type !== "blob") errors.push("didrun note object type");
	if (authority.note_blob_oid !== receipt.evidence.git_note.blob_oid) errors.push("didrun note object identity");
	if (authority.note_body_sha256 !== receipt.evidence.git_note.body_sha256) errors.push("didrun note body digest");
	const note = authority.note;
	if (!note || typeof note !== "object" || Array.isArray(note) ||
		!isDeepStrictEqual(Object.keys(note).sort(), ["claims", "commit", "coverage", "secrets_override", "tree", "version"])) {
		errors.push("missing or malformed parsed didrun Git note");
		return errors;
	}
	if (note.version !== receipt.evidence.git_note.version) errors.push("didrun note version");
	if (note.commit !== receipt.source_commit) errors.push("didrun note commit");
	if (note.tree !== receipt.source_tree) errors.push("didrun note tree");
	if (note.secrets_override !== receipt.secrets_override) errors.push("didrun note redacted seal disclosure");
	if (!Array.isArray(note.claims) || note.claims.length !== c3ClaimLabels.length) {
		errors.push("didrun note claim roster length");
	} else {
		for (let index = 0; index < c3ClaimLabels.length; index += 1) {
			const recorded = note.claims[index];
			const claim = recorded?.claim;
			if (!recorded || typeof recorded !== "object" || Array.isArray(recorded) ||
				!isDeepStrictEqual(Object.keys(recorded).sort(), ["claim", "delta", "exit_code", "grade", "reason", "supporting_event_index"]) ||
				!claim || typeof claim !== "object" || Array.isArray(claim) ||
				!isDeepStrictEqual(Object.keys(claim).sort(), ["argv_preview", "ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) ||
				claim.label !== c3ClaimLabels[index] || claim.ctype !== c3ClaimTypes[index] ||
				claim.declared_at_index !== index || !isDeepStrictEqual(claim.event_indices, [index]) ||
				!isDeepStrictEqual(claim.pathspecs, []) || recorded.supporting_event_index !== index ||
				recorded.grade !== "tree-exact" || recorded.exit_code !== 0 ||
				recorded.reason !== "self-stable command ran against the sealed tree" || !isDeepStrictEqual(recorded.delta, [])) {
				errors.push(`didrun note claim ${index + 1}`);
			}
			if (!validateC3ClaimPreview(index, claim?.argv_preview)) errors.push(`didrun note claim ${index + 1} argv`);
		}
	}
	if (!note.coverage || typeof note.coverage !== "object" || Array.isArray(note.coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage).sort(), ["by_coverage", "total_events"]) ||
		note.coverage.total_events !== receipt.coverage_total_events ||
		!note.coverage.by_coverage || typeof note.coverage.by_coverage !== "object" || Array.isArray(note.coverage.by_coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage.by_coverage), ["complete"]) ||
		note.coverage.by_coverage.complete !== receipt.coverage_complete_events) {
		errors.push("didrun note exact event coverage");
	}
	return errors;
}

function c3ReceiptDisclosureLines(receipt) {
	const { html_report: html, ledger_archive: ledger, git_note: note, seal_redaction: redaction } = receipt.evidence;
	return Object.freeze([
		`C3 source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`, is sealed, note-present, and strict-clean with \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\`.`,
		`The C3 seal records \`secrets_override: ${receipt.secrets_override}\` after ${redaction.finding_count} local scanner findings with kinds \`${redaction.finding_kinds.join(",") || "none"}\`; this is not evidence of secret absence.`,
		`The separately claimed C3 structured staged credential scan reported \`${redaction.structured_staged_scan_findings}\` findings; its provenance is sealed supporting event \`16\`, not the Git note alone.`,
		`Local ignored C3 ledger archive \`${ledger.path}\` contains \`${ledger.sealed_event_count}\` sealed events and \`${ledger.archive_session_event_count}\` archived session events; post-seal events, if any, are outside the sealed manifest.`,
		`C3 ledger manifests use \`${ledger.manifest_format}\`; all-files SHA-256 is \`${ledger.all_files_manifest_sha256}\` and objects-only SHA-256 is \`${ledger.objects_manifest_sha256}\`.`,
		`C3 archive core SHA-256 values are session \`${ledger.session_log_sha256}\`, claims \`${ledger.claims_jsonl_sha256}\`, seals \`${ledger.seals_jsonl_sha256}\`, and .gitignore \`${ledger.gitignore_sha256}\`.`,
		`The local C3 HTML snapshot is \`${html.path}\`, SHA-256 \`${html.sha256}\`, ${html.bytes} bytes; it is not a portable strict witness.`,
		`The C3 Git note blob is \`${note.blob_oid}\` with body SHA-256 \`${note.body_sha256}\`.`,
	]);
}

function parseC3ReceiptBlock(body) {
	const startCount = countOccurrences(body, c3ReceiptBlockStart);
	const endCount = countOccurrences(body, c3ReceiptBlockEnd);
	if (startCount === 0 && endCount === 0) return Object.freeze({ block: undefined, stripped: body });
	if (startCount !== 1 || endCount !== 1) {
		throw new Error(`C3 receipt marker cardinality start=${startCount} end=${endCount}`);
	}
	const start = body.indexOf(c3ReceiptBlockStart);
	const end = body.indexOf(c3ReceiptBlockEnd);
	const standalone = (index, marker) => (index === 0 || body[index - 1] === "\n") &&
		(index + marker.length === body.length || body[index + marker.length] === "\n");
	if (!standalone(start, c3ReceiptBlockStart) || !standalone(end, c3ReceiptBlockEnd)) {
		throw new Error("C3 receipt markers must be standalone LF lines");
	}
	if (end <= start) throw new Error("C3 receipt marker order");
	const blockEnd = end + c3ReceiptBlockEnd.length;
	const removalEnd = body[blockEnd] === "\n" ? blockEnd + 1 : blockEnd;
	return Object.freeze({ block: body.slice(start, blockEnd), stripped: `${body.slice(0, start)}${body.slice(removalEnd)}` });
}

function insertLegacyReceiptBlockBeforeOptionalTerminalC3(body, block, label, markerFreeMode) {
	if (typeof body !== "string" || typeof block !== "string" || block.length === 0 || block.endsWith("\n") ||
		block.includes(c3ReceiptBlockStart) || block.includes(c3ReceiptBlockEnd)) {
		throw new Error(`${label} legacy receipt block precondition`);
	}
	if (markerFreeMode !== "direct" && markerFreeMode !== "trimmed-blank") {
		throw new Error(`${label} unknown marker-free render mode`);
	}
	const parsed = parseC3ReceiptBlock(body);
	if (parsed.block === undefined) {
		if (markerFreeMode === "direct") {
			if (!body.endsWith("\n")) throw new Error(`${label} marker-free handoff must end with LF`);
			return `${body}${block}\n`;
		}
		if (markerFreeMode === "trimmed-blank") return `${body.trimEnd()}\n\n${block}\n`;
	}
	requireVisibleMarkdownReceiptBlock(body, c3ReceiptBlockStart, c3ReceiptBlockEnd, `${label} terminal C3 receipt block`);
	const terminal = `${parsed.block}\n`;
	if (!parsed.stripped.endsWith("\n") || body !== `${parsed.stripped}${terminal}`) {
		throw new Error(`${label} C3 receipt block must already be exact and terminal`);
	}
	const rendered = `${parsed.stripped}${block}\n${terminal}`;
	const reparsed = parseC3ReceiptBlock(rendered);
	if (reparsed.block !== parsed.block || rendered !== `${reparsed.stripped}${terminal}` ||
		reparsed.stripped !== `${parsed.stripped}${block}\n`) {
		throw new Error(`${label} terminal C3 receipt preservation mismatch`);
	}
	return rendered;
}

function c3ReceiptHandoffBlock(receipt) {
	const rows = receipt.claims.map((claim) => `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`);
	const disclosures = c3ReceiptDisclosureLines(receipt).flatMap((line) => [line, ""]);
	return [
		c3ReceiptBlockStart,
		"### P07B-C C3 source receipt map",
		"",
		`C3 source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`.`,
		`C3 strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\`; strict exit: \`0\`.`,
		"",
		...disclosures,
		"| Claim label | Claim type | Verbatim grade |",
		"| --- | --- | --- |",
		...rows,
		c3ReceiptBlockEnd,
	].join("\n");
}

function c3ReceiptSpecificHandoffLines(receipt) {
	return Object.freeze([
		"### P07B-C C3 source receipt map",
		`C3 source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`.`,
		`C3 strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\`; strict exit: \`0\`.`,
		...c3ReceiptDisclosureLines(receipt),
		...receipt.claims.map((claim) => `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`),
	]);
}

function c3ReconciledStatusState(receipt) {
	return `- **Source receipt:** C3 source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`, is sealed, note-present, and strict-clean; every C3 source grade below is \`TREE-EXACT\`. The independently active C3B descendant is outside these source grades.`;
}

function visibleMarkdownLifecycleBody(body) {
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
				if (end === -1) {
					cursor = originalLine.length;
					continue;
				}
				htmlComment = false;
				cursor = end + 3;
				continue;
			}
			const start = originalLine.indexOf("<!--", cursor);
			if (start === -1) {
				line += originalLine.slice(cursor);
				cursor = originalLine.length;
				continue;
			}
			line += originalLine.slice(cursor, start);
			htmlComment = true;
			cursor = start + 4;
		}
		if (/^(?: {4}|\t)/u.test(line)) {
			visible.push("");
			continue;
		}
		if (/^ {0,3}<(?:\/?[A-Za-z][A-Za-z0-9-]*(?:\s|\/?>)|\?|![A-Z]|!\[CDATA\[)/u.test(line)) {
			throw new Error("raw HTML block syntax is not admitted in lifecycle authority");
		}
		const opening = /^ {0,3}(`{3,}|~{3,})/u.exec(line);
		if (opening) {
			fence = { character: opening[1][0], length: opening[1].length };
			visible.push("");
			continue;
		}
		visible.push(line);
	}
	if (htmlComment) throw new Error("unterminated Markdown HTML comment");
	if (fence !== undefined) throw new Error("unterminated Markdown fence");
	return visible.join("\n");
}

function requireVisibleMarkdownReceiptBlock(body, startMarker, endMarker, label) {
	const startSentinel = `COUNTERSHAPE_VISIBLE_${label}_START`;
	const endSentinel = `COUNTERSHAPE_VISIBLE_${label}_END`;
	if (body.includes(startSentinel) || body.includes(endSentinel)) {
		throw new Error(`${label} visibility sentinel collision`);
	}
	const projected = visibleMarkdownLifecycleBody(
		body.replace(startMarker, startSentinel).replace(endMarker, endSentinel),
	);
	if (countOccurrences(projected, startSentinel) !== 1 || countOccurrences(projected, endSentinel) !== 1 ||
		projected.indexOf(startSentinel) >= projected.indexOf(endSentinel)) {
		throw new Error(`${label} is not visible Markdown authority`);
	}
}

const c3pbLifecyclePhases = Object.freeze([
	Object.freeze({ field: "before", phase: "C3M_ACTIVE" }),
	Object.freeze({ field: "after", phase: "C3PB_ACTIVE" }),
	Object.freeze({ field: "descendant", phase: "C3A_ACTIVE" }),
	Object.freeze({ field: "c3l", phase: "C3L_ACTIVE" }),
	Object.freeze({ field: "c3f", phase: "C3F_ACTIVE" }),
	Object.freeze({ field: "c3s", phase: "C3S_ACTIVE" }),
	Object.freeze({ field: "c3", phase: "C3_ACTIVE" }),
	Object.freeze({ field: "c3r", phase: "C3R_ACTIVE" }),
	Object.freeze({ field: "c3q", phase: "C3Q_ACTIVE" }),
	Object.freeze({ field: "c3t", phase: "C3T_ACTIVE" }),
	Object.freeze({ field: "c3u", phase: "C3U_ACTIVE" }),
	Object.freeze({ field: "c3b", phase: "C3B_ACTIVE" }),
]);

function classifyC3PBHandoffPhase(body, c3Receipt, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority) {
	const visibleBody = visibleMarkdownLifecycleBody(body);
	if (c3Receipt === undefined) {
		const prematureC3B = visibleBody.includes("C3B receipt reconciliation is the active") ||
			c3pbHandoffTransitions.some((transition) =>
				typeof transition.c3b === "string" && countOccurrences(visibleBody, transition.c3b) > 0);
		if (prematureC3B) throw new Error("C3B lifecycle text is premature without a validated C3 receipt");
	}
	const lines = visibleBody.split("\n");
	const currentStateHeadings = lines.flatMap((line, index) => line === "## Current state" ? [index] : []);
	if (currentStateHeadings.length !== 1) {
		throw new Error(`current-state heading cardinality ${currentStateHeadings.length}`);
	}
	const currentStateStart = currentStateHeadings[0];
	const currentStateEnd = lines.findIndex((line, index) => index > currentStateStart && line.startsWith("## "));
	const boundedCurrentStateEnd = currentStateEnd === -1 ? lines.length : currentStateEnd;
	const firstSubheading = lines.findIndex((line, index) =>
		index > currentStateStart && index < boundedCurrentStateEnd && line.startsWith("### "));
	if (firstSubheading === -1) throw new Error("current-state operational subheading missing");
	const headingTransition = c3pbHandoffTransitions.find(({ name }) => name === "maintenance heading");
	const locatedCount = (transition, field) => {
		const value = c3pbTransitionValue(transition, field, c3Receipt);
		const metadataContract = c3pbMetadataLineContracts[transition.name];
		if (metadataContract !== undefined) {
			const matchingSlots = lines.slice(currentStateStart + 1, firstSubheading)
				.filter((line) => line.startsWith(metadataContract.prefix));
			return matchingSlots.length === 1 && matchingSlots[0] === metadataContract.render(value, field, c3Receipt, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority) ? 1 : 0;
		}
		if (transition.name === "maintenance heading") {
			return lines.slice(firstSubheading, boundedCurrentStateEnd).filter((line) => line === value).length;
		}
		if (transition.name === "receipt presence") {
			const heading = headingTransition[field];
			const sectionStarts = lines.flatMap((line, index) =>
				index >= firstSubheading && index < boundedCurrentStateEnd && line === heading ? [index] : []);
			if (sectionStarts.length !== 1) return 0;
			const sectionStart = sectionStarts[0];
			const sectionEnd = lines.findIndex((line, index) =>
				index > sectionStart && index < boundedCurrentStateEnd && (line.startsWith("### ") || line.startsWith("## ")));
			const boundedSectionEnd = sectionEnd === -1 ? boundedCurrentStateEnd : sectionEnd;
			return lines.slice(sectionStart + 1, boundedSectionEnd).filter((line) => line === value).length;
		}
		return 0;
	};
	const admittedPhases = c3Receipt === undefined
		? c3pbLifecyclePhases.filter(({ field }) => field !== "c3b")
		: c3pbLifecyclePhases;
	const counts = c3pbHandoffTransitions.map((transition) => Object.freeze({
		name: transition.name,
		fields: Object.freeze(Object.fromEntries(admittedPhases.map(({ field }) => {
			const value = c3pbTransitionValue(transition, field, c3Receipt);
			return [field, Object.freeze({
				count: countOccurrences(visibleBody, value),
				located: locatedCount(transition, field),
			})];
		}))),
	}));
	const active = admittedPhases.filter(({ field }) => counts.every(({ fields }) =>
		admittedPhases.every(({ field: candidate }) => fields[candidate].count === (candidate === field ? 1 : 0)) &&
		fields[field].located === 1));
	if (active.length !== 1) {
		const summary = counts.map(({ name, fields }) => `${name}=${admittedPhases
			.map(({ field }) => `${field}:${fields[field].count}:${fields[field].located}`).join("/")}`).join(", ");
		throw new Error(`mixed, duplicate, or incomplete C3M/C3PB/C3A/C3L/C3F/C3S/C3/C3R/C3Q/C3T/C3U/C3B operational phase (${summary})`);
	}
	return active[0].phase;
}

function requireSyntheticC3SActiveHandoff(body) {
	const phase = classifyC3PBHandoffPhase(body);
	if (phase !== "C3S_ACTIVE") {
		throw new Error(`P07B-C historical operational phase must be C3S-active, found ${phase}`);
	}
	return body;
}

function requireSyntheticC3ActiveHandoff(body) {
	const phase = classifyC3PBHandoffPhase(body);
	if (phase !== "C3_ACTIVE") {
		throw new Error(`P07B-C current operational phase must be C3-active, found ${phase}`);
	}
	return body;
}

function requireSyntheticC3RActiveHandoff(body) {
	const phase = classifyC3PBHandoffPhase(body);
	if (phase !== "C3R_ACTIVE") {
		throw new Error(`P07B-C current operational phase must be C3R-active, found ${phase}`);
	}
	return body;
}

function requireSyntheticC3QActiveHandoff(body) {
	const phase = classifyC3PBHandoffPhase(body);
	if (phase !== "C3Q_ACTIVE") {
		throw new Error(`P07B-C current operational phase must be C3Q-active, found ${phase}`);
	}
	return body;
}

function requireSyntheticC3TActiveHandoff(body) {
	const phase = classifyC3PBHandoffPhase(body);
	if (phase !== "C3T_ACTIVE") {
		throw new Error(`P07B-C current operational phase must be C3T-active, found ${phase}`);
	}
	return body;
}

function requireSyntheticC3UActiveHandoff(body) {
	const phase = classifyC3PBHandoffPhase(body);
	if (phase !== "C3U_ACTIVE") {
		throw new Error(`P07B-C current operational phase must be C3U-active, found ${phase}`);
	}
	return body;
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
		try {
			const parsed = parseC3PReceiptBlock(handoff);
			if (parsed.block !== undefined) errors.push("docs/HANDOFF_MODE_C.md: C3P pending receipt state cannot contain a source receipt block");
		} catch (error) {
			errors.push(`docs/HANDOFF_MODE_C.md: C3P pending receipt marker topology invalid (${error.message})`);
		}
		for (const digest of [
			requiredC3PDigest, requiredC3PBDigest, requiredC3SDigest, requiredC3Digest,
			requiredC3RDigest, requiredC3QDigest, requiredC3TDigest, requiredC3UDigest, requiredC3BDigest,
		]) {
			if (!handoff.includes(digest)) errors.push(`docs/HANDOFF_MODE_C.md: C3P prerequisite digest missing: ${digest}`);
		}
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

	const state = c3pReconciledStatusState(receipt);
	requireExactlyOnce(status, state, c3pStatusPath, "C3PB reconciled source state", errors);
	requireExactlyOnce(status, "## C3P source receipt map", c3pStatusPath, "C3PB receipt heading", errors);
	requireExactlyOnce(status, "- C3P source strict exit: `0`", c3pStatusPath, "C3PB strict result", errors);
	requireExactlyOnce(status, `- C3P source strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\``, c3pStatusPath, "C3PB strict claim count", errors);
	requireExactlyOnce(status, c3pReceiptNoRecursionBoundary, c3pStatusPath, "C3P no-recursion boundary", errors);
	for (const claim of receipt.claims) {
		requireExactlyOnce(status, `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`, c3pStatusPath, "C3PB source receipt map", errors);
		requireClaimLabelExactlyOnce(status, claim.label, c3pStatusPath, "C3PB source receipt map", errors);
	}
	const disclosures = c3pReceiptDisclosureLines(receipt);
	for (const disclosure of disclosures) requireExactlyOnce(status, disclosure, c3pStatusPath, "C3PB source evidence disclosure", errors);
	try {
		const parsed = parseC3PReceiptBlock(handoff);
		if (parsed.block === undefined) {
			errors.push("docs/HANDOFF_MODE_C.md: C3PB receipt block missing");
		} else {
			const expectedBlock = c3pReceiptHandoffBlock(receipt);
			if (parsed.block !== expectedBlock) errors.push("docs/HANDOFF_MODE_C.md: C3PB receipt block payload mismatch");
			for (const line of c3pReceiptSpecificHandoffLines(receipt)) {
				if (parsed.stripped.includes(line)) {
					errors.push(`docs/HANDOFF_MODE_C.md: C3PB receipt residue outside block: ${JSON.stringify(line)}`);
				}
			}
		}
	} catch (error) {
		errors.push(`docs/HANDOFF_MODE_C.md: C3PB receipt block invalid (${error.message})`);
	}
}

async function checkC3ReceiptPhase(
	root,
	overrides,
	status,
	handoff,
	errors,
	injectedReceiptAuthority,
	injectedC3RAuthority,
	injectedC3QAuthority,
	injectedC3TAuthority,
	injectedC3UAuthority,
) {
	try {
		status = visibleMarkdownLifecycleBody(status);
	} catch (error) {
		errors.push(`${c3StatusPath}: visible receipt authority invalid (${error.message})`);
		return;
	}
	let receiptText;
	try {
		receiptText = await readText(root, c3ReceiptDeclarationPath, overrides);
	} catch (error) {
		if (error.code !== "ENOENT") errors.push(`${c3ReceiptDeclarationPath}: unreadable (${error.message})`);
	}
	const sourceScope = `- **Scope:** 40 exact mode-\`100644\` paths, no prefixes; sorted-newline roster \`${requiredC3Digest}\``;
	const sourceSubject = `- **Commit subject:** \`${c3SourceSubject}\``;
	const requireC3BPendingRows = () => {
		for (let index = 0; index < c3bClaimLabels.length; index += 1) {
			requireExactlyOnce(status, `| \`${c3bClaimLabels[index]}\` | \`${c3bClaimTypes[index]}\` | \`UNRECEIPTED\` |`,
				c3StatusPath, "C3B pending receipt map", errors);
			requireClaimLabelExactlyOnce(status, c3bClaimLabels[index], c3StatusPath, "C3B pending receipt map", errors);
		}
	};
	requireExactlyOnce(status, sourceScope, c3StatusPath, "C3 source scope", errors);
	requireExactlyOnce(status, sourceSubject, c3StatusPath, "C3 source subject", errors);
	requireC3BPendingRows();

	if (receiptText === undefined) {
		requireExactlyOnce(status, c3PendingStatusState, c3StatusPath, "C3 pending source state", errors);
		requireExactlyOnce(status, "## Intended C3 source receipt map", c3StatusPath, "C3 pending source receipt heading", errors);
		if (countOccurrences(status, "## C3 source receipt map") !== 0) {
			errors.push(`${c3StatusPath}: C3 pending source state cannot contain the reconciled source receipt heading`);
		}
		for (let index = 0; index < c3ClaimLabels.length; index += 1) {
			requireExactlyOnce(status, `| \`${c3ClaimLabels[index]}\` | \`${c3ClaimTypes[index]}\` | \`UNRECEIPTED\` |`,
				c3StatusPath, "C3 pending source receipt map", errors);
			requireClaimLabelExactlyOnce(status, c3ClaimLabels[index], c3StatusPath, "C3 pending source receipt map", errors);
		}
		try {
			const parsed = parseC3ReceiptBlock(handoff);
			if (parsed.block !== undefined) errors.push("docs/HANDOFF_MODE_C.md: C3 pending receipt state cannot contain a source receipt block");
		} catch (error) {
			errors.push(`docs/HANDOFF_MODE_C.md: C3 pending receipt marker topology invalid (${error.message})`);
		}
		try {
			const phase = classifyC3PBHandoffPhase(handoff);
			if (phase !== "C3U_ACTIVE") errors.push(`docs/HANDOFF_MODE_C.md: C3 receipt-absent phase must be C3U-active, found ${phase}`);
		} catch (error) {
			errors.push(`docs/HANDOFF_MODE_C.md: C3 receipt-absent operational phase invalid (${error.message})`);
		}
		rejectReceiptSelfClaims(status, c3StatusPath, "C3", errors);
		rejectC3CanonicalSelfClaims(handoff, "docs/HANDOFF_MODE_C.md", errors);
		return;
	}

	let receipt;
	try {
		receipt = JSON.parse(receiptText);
	} catch (error) {
		errors.push(`${c3ReceiptDeclarationPath}: invalid JSON (${error.message})`);
		return;
	}
	const receiptErrors = validateC3ReceiptDeclaration(receipt);
	for (const error of receiptErrors) errors.push(`${c3ReceiptDeclarationPath}: invalid receipt declaration (${error})`);
	if (receiptErrors.length > 0) return;
	if (receiptText !== `${JSON.stringify(receipt)}\n`) {
		errors.push(`${c3ReceiptDeclarationPath}: receipt declaration must be exact canonical one-line JSON`);
		return;
	}
	let authority = injectedReceiptAuthority;
	if (authority === undefined) {
		try {
			authority = await loadReceiptAuthorityFromGit(root, receipt);
		} catch (error) {
			errors.push(`${c3ReceiptDeclarationPath}: Git/didrun authority unavailable (${error.message})`);
			return;
		}
	}
	const authorityErrors = validateC3ReceiptAuthority(receipt, authority);
	for (const error of authorityErrors) errors.push(`${c3ReceiptDeclarationPath}: Git/didrun authority mismatch (${error})`);
	if (authorityErrors.length > 0) return;
	let c3rAuthority = injectedC3RAuthority;
	if (c3rAuthority === undefined) {
		try {
			c3rAuthority = await loadC3RAuthorityFromGit(root);
		} catch (error) {
			errors.push(`${c3ReceiptDeclarationPath}: sealed C3R authority unavailable (${error.message})`);
			return;
		}
	}
	const c3rErrors = validateSealedC3RAuthority(c3rAuthority);
	for (const error of c3rErrors) errors.push(`${c3ReceiptDeclarationPath}: sealed C3R authority mismatch (${error})`);
	if (c3rErrors.length > 0) return;
	let c3qAuthority = injectedC3QAuthority;
	if (c3qAuthority === undefined) {
		try {
			c3qAuthority = await loadC3QAuthorityFromGit(root, c3rAuthority.commit);
		} catch (error) {
			errors.push(`${c3ReceiptDeclarationPath}: sealed C3Q authority unavailable (${error.message})`);
			return;
		}
	}
	const c3qErrors = validateSealedC3QAuthority(c3qAuthority);
	for (const error of c3qErrors) errors.push(`${c3ReceiptDeclarationPath}: sealed C3Q authority mismatch (${error})`);
	if (c3qErrors.length > 0) return;
	let c3tAuthority = injectedC3TAuthority;
	if (c3tAuthority === undefined) {
		try {
			c3tAuthority = await loadC3TAuthorityFromGit(root, c3qAuthority.commit);
		} catch (error) {
			errors.push(`${c3ReceiptDeclarationPath}: sealed C3T authority unavailable (${error.message})`);
			return;
		}
	}
	const c3tErrors = validateSealedC3TAuthority(c3tAuthority);
	for (const error of c3tErrors) errors.push(`${c3ReceiptDeclarationPath}: sealed C3T authority mismatch (${error})`);
	if (c3tErrors.length > 0) return;
	let c3uAuthority = injectedC3UAuthority;
	if (c3uAuthority === undefined) {
		try {
			c3uAuthority = await loadC3UAuthorityFromGit(root, c3tAuthority.commit);
		} catch (error) {
			errors.push(`${c3ReceiptDeclarationPath}: sealed C3U authority unavailable (${error.message})`);
			return;
		}
	}
	const c3uErrors = validateC3UAuthority(c3uAuthority, c3tAuthority.commit);
	for (const error of c3uErrors) errors.push(`${c3ReceiptDeclarationPath}: sealed C3U authority mismatch (${error})`);
	if (c3uErrors.length > 0) return;

	requireExactlyOnce(status, c3ReconciledStatusState(receipt), c3StatusPath, "C3B reconciled source state", errors);
	requireExactlyOnce(status, "## C3 source receipt map", c3StatusPath, "C3 source receipt heading", errors);
	if (countOccurrences(status, c3PendingStatusState) !== 0) {
		errors.push(`${c3StatusPath}: C3 receipt-present state retains the pending source state`);
	}
	if (countOccurrences(status, "## Intended C3 source receipt map") !== 0) {
		errors.push(`${c3StatusPath}: C3 receipt-present state retains the pending source receipt heading`);
	}
	requireExactlyOnce(status, "- C3 source strict exit: `0`", c3StatusPath, "C3 source strict result", errors);
	requireExactlyOnce(status,
		`- C3 source strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\``,
		c3StatusPath, "C3 source strict claim count", errors);
	requireExactlyOnce(status, c3ReceiptNoRecursionBoundary, c3StatusPath, "C3 no-recursion boundary", errors);
	for (const claim of receipt.claims) {
		requireExactlyOnce(status, `| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`,
			c3StatusPath, "C3 source receipt map", errors);
		requireClaimLabelExactlyOnce(status, claim.label, c3StatusPath, "C3 source receipt map", errors);
	}
	for (const disclosure of c3ReceiptDisclosureLines(receipt)) {
		requireExactlyOnce(status, disclosure, c3StatusPath, "C3 source evidence disclosure", errors);
	}
	try {
		const phase = classifyC3PBHandoffPhase(handoff, receipt, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority);
		if (phase !== "C3B_ACTIVE") errors.push(`docs/HANDOFF_MODE_C.md: C3 receipt-present phase must be C3B-active, found ${phase}`);
	} catch (error) {
		errors.push(`docs/HANDOFF_MODE_C.md: C3 receipt-present operational phase invalid (${error.message})`);
	}
	try {
		const observedSection = currentStateSubsection(handoff, "### Active C3B source receipt reconciliation");
		const expectedSection = c3bActiveMaintenanceSection(receipt, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority);
		if (observedSection !== expectedSection) {
			const staleC3Q = observedSection.includes("Its planned nine-claim final ledger") ||
				observedSection.includes("Every intended grade remains `UNRECEIPTED` at this boundary.");
			errors.push(`docs/HANDOFF_MODE_C.md: C3B active maintenance section ${staleC3Q ? "retains stale C3Q active prose" : "payload mismatch"}`);
		}
	} catch (error) {
		errors.push(`docs/HANDOFF_MODE_C.md: C3B active maintenance section invalid (${error.message})`);
	}
	try {
		const parsed = parseC3ReceiptBlock(handoff);
		if (parsed.block === undefined) {
			errors.push("docs/HANDOFF_MODE_C.md: C3 source receipt block missing");
		} else {
			try {
				requireVisibleMarkdownReceiptBlock(handoff, c3ReceiptBlockStart, c3ReceiptBlockEnd, "C3 source receipt block");
			} catch (error) {
				errors.push(`docs/HANDOFF_MODE_C.md: ${error.message}`);
				}
				const expectedBlock = c3ReceiptHandoffBlock(receipt);
				if (parsed.block !== expectedBlock) errors.push("docs/HANDOFF_MODE_C.md: C3 source receipt block payload mismatch");
				if (handoff !== `${parsed.stripped}${expectedBlock}\n`) {
					errors.push("docs/HANDOFF_MODE_C.md: C3 source receipt block must be the exact terminal handoff block");
				}
				for (const line of c3ReceiptSpecificHandoffLines(receipt)) {
				if (parsed.stripped.includes(line)) {
					errors.push(`docs/HANDOFF_MODE_C.md: C3 source receipt residue outside block: ${JSON.stringify(line)}`);
				}
			}
		}
	} catch (error) {
		errors.push(`docs/HANDOFF_MODE_C.md: C3 source receipt block invalid (${error.message})`);
	}
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

async function syntheticC3PReceiptPlanFixture(fixture, baseOverrides = new Map(), injectedLifecycle) {
	const absent = await syntheticC3PAbsentPhaseFixture(baseOverrides);
	let status = absent.get(c3pStatusPath);
	const sourceState = "- **State:** active pre-seal `SOURCE_FULL` prerequisite; every C3P grade below is `UNRECEIPTED`";
	const receiptState = c3pReconciledStatusState(fixture.receipt);
	status = replaceFixtureExactlyOnce(status, sourceState, receiptState, "C3P receipt-present state");
	const metadata = [
		"## C3P source receipt map",
		"- C3P source strict exit: `0`",
		`- C3P source strict claims: \`${fixture.receipt.strict_claims_recorded_exact}/${fixture.receipt.strict_claims_total} claims recorded-exact\``,
		c3pReceiptNoRecursionBoundary,
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
	const lifecycle = injectedLifecycle ?? await currentC3LifecycleFixture(absent);
	const receiptPhaseHandoff = absent.get("docs/HANDOFF_MODE_C.md");
	const receiptPhase = classifyC3PBHandoffPhase(
		receiptPhaseHandoff,
		lifecycle.receipt,
		lifecycle.c3rAuthority,
		lifecycle.c3qAuthority,
		lifecycle.c3tAuthority,
		lifecycle.c3uAuthority,
	);
	if (receiptPhase !== lifecycle.phase) {
		throw new Error(`P07B-C C3PB synthetic handoff must preserve ${lifecycle.phase}, found ${receiptPhase}`);
	}
	if (!receiptPhaseHandoff.endsWith("\n") || parseC3PReceiptBlock(receiptPhaseHandoff).block !== undefined) {
		throw new Error("P07B-C C3PB synthetic handoff precondition mismatch");
	}
	const block = c3pReceiptHandoffBlock(fixture.receipt);
	const handoffWithReceipt = insertLegacyReceiptBlockBeforeOptionalTerminalC3(
		receiptPhaseHandoff,
		block,
		"C3P synthetic receipt fixture",
		"direct",
	);
	const roundTrip = parseC3PReceiptBlock(handoffWithReceipt);
	if (roundTrip.block !== block || roundTrip.stripped !== receiptPhaseHandoff) {
		throw new Error("P07B-C C3PB synthetic receipt block round-trip mismatch");
	}
	return {
		overrides: new Map([
			...absent,
			[c3pStatusPath, status],
			["docs/HANDOFF_MODE_C.md", handoffWithReceipt],
			[c3pReceiptDeclarationPath, `${JSON.stringify(fixture.receipt)}\n`],
		]),
		authority: fixture.authority,
		priorHandoff: receiptPhaseHandoff,
	};
}

function readSealedC3PStatusFixture() {
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
			throw new Error(`P07B-C C3P sealed-source fixture git ${args[0]} failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
		}
		return result.stdout;
	};
	const tree = run(["rev-parse", "--verify", `${sealedC3PIdentity.commit}^{tree}`]).trim();
	if (tree !== sealedC3PIdentity.tree) throw new Error(`P07B-C C3P sealed-source fixture tree drift: ${tree}`);
	const status = run(["cat-file", "blob", `${sealedC3PIdentity.commit}:${c3pStatusPath}`], 2 * 1024 * 1024);
	if (!status.endsWith("\n") || Buffer.byteLength(status, "utf8") > 1024 * 1024) {
		throw new Error("P07B-C C3P sealed-source status fixture is not bounded canonical text");
	}
	return status;
}

function withoutC3PReceiptBlock(body) {
	return parseC3PReceiptBlock(body).stripped;
}

async function syntheticC3PAbsentPhaseFixture(baseOverrides = new Map()) {
	const status = readSealedC3PStatusFixture();
	const currentHandoff = await readText(repositoryRoot, "docs/HANDOFF_MODE_C.md", baseOverrides);
	const handoff = withoutC3PReceiptBlock(currentHandoff);
	return new Map([
		...baseOverrides,
		[c3pStatusPath, status],
		["docs/HANDOFF_MODE_C.md", handoff],
		[c3pReceiptDeclarationPath, ABSENT_FIXTURE_PATH],
	]);
}

async function currentC3LifecycleFixture(overrides = new Map()) {
	let receiptText;
	try {
		receiptText = await readText(repositoryRoot, c3ReceiptDeclarationPath, overrides);
	} catch (error) {
		if (error.code !== "ENOENT") throw error;
	}
	if (receiptText === undefined) {
		return Object.freeze({
			field: "c3u", phase: "C3U_ACTIVE", receipt: undefined,
			c3rAuthority: undefined, c3qAuthority: undefined, c3tAuthority: undefined, c3uAuthority: undefined,
		});
	}
	let receipt;
	try {
		receipt = JSON.parse(receiptText);
	} catch (error) {
		throw new Error(`P07B-C current C3 lifecycle receipt is invalid JSON (${error.message})`);
	}
	const c3rAuthority = await loadC3RAuthorityFromGit(repositoryRoot);
	const c3qAuthority = await loadC3QAuthorityFromGit(repositoryRoot, c3rAuthority.commit);
	const c3tAuthority = await loadC3TAuthorityFromGit(repositoryRoot, c3qAuthority.commit);
	const c3uAuthority = await loadC3UAuthorityFromGit(repositoryRoot, c3tAuthority.commit);
	return Object.freeze({ field: "c3b", phase: "C3B_ACTIVE", receipt, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority });
}

export async function runC3PReceiptSelfTest() {
	const liveErrors = await checkPlan();
	if (liveErrors.length > 0) throw new Error(`P07B-C C3P receipt checker live baseline failed:\n${liveErrors.join("\n")}`);
	let rejected = 0;
	let phaseStabilityControls = 0;
	rejected += runC3AHermeticPresealSelfTest();
	rejected += runC3LHermeticPresealSelfTest();
	rejected += runC3FHermeticPresealSelfTest();
	rejected += runC3SHermeticPresealSelfTest();
	rejected += runHermeticPresealSelfTest("C3", c3ExpectedClaimArgv, c3HermeticArgvPrefix);
	rejected += runHermeticPresealSelfTest("C3R", c3rExpectedClaimArgv, c3rHermeticArgvPrefix);
	rejected += runHermeticPresealSelfTest("C3Q", c3qExpectedClaimArgv, c3qHermeticArgvPrefix);
	rejected += runHermeticPresealSelfTest("C3T", c3tExpectedClaimArgv, c3tHermeticArgvPrefix);
	rejected += runHermeticPresealSelfTest("C3U", c3uExpectedClaimArgv, c3uHermeticArgvPrefix);
	rejected += runHermeticPresealSelfTest("C3B", c3bExpectedClaimArgv, c3bHermeticArgvPrefix);
	const requireError = (name, errors, expected) => {
		if (!errors.some((error) => error.includes(expected))) {
			throw new Error(`P07B-C C3P receipt checker self-test false negative: ${name} (${errors.join("; ")})`);
		}
		rejected += 1;
	};
	const requireStripError = (name, body, expected) => {
		let message = "";
		try {
			withoutC3PReceiptBlock(body);
		} catch (error) {
			message = String(error.message);
		}
		if (!message.includes(expected)) {
			throw new Error(`P07B-C C3P receipt-block self-test false negative: ${name} (${message || "accepted"})`);
		}
		rejected += 1;
	};
	const markerStart = "<!-- P07B-C-C3P-SOURCE-RECEIPTS:START -->";
	const markerEnd = "<!-- P07B-C-C3P-SOURCE-RECEIPTS:END -->";
	const liveUnstrippedHandoff = await readText(repositoryRoot, "docs/HANDOFF_MODE_C.md", new Map());
	const lifecycle = await currentC3LifecycleFixture();
	const lifecycleValue = (transition, field = lifecycle.field) =>
		c3pbTransitionValue(transition, field, lifecycle.receipt);
	const lifecycleText = (transition, field = lifecycle.field) => {
		const value = lifecycleValue(transition, field);
		const metadata = c3pbMetadataLineContracts[transition.name];
		return metadata === undefined
			? value
			: metadata.render(value, field, lifecycle.receipt, lifecycle.c3rAuthority, lifecycle.c3qAuthority, lifecycle.c3tAuthority, lifecycle.c3uAuthority);
	};
	const phaseInvalidDiagnostic = lifecycle.receipt === undefined
		? "C3 receipt-absent operational phase invalid"
		: "C3 receipt-present operational phase invalid";
	const phaseMismatchDiagnostic = lifecycle.receipt === undefined
		? "C3 receipt-absent phase must be C3U-active"
		: "C3 receipt-present phase must be C3B-active";
	if (withoutC3PReceiptBlock("prefix\nsuffix\n") !== "prefix\nsuffix\n") {
		throw new Error("P07B-C C3P receipt-block marker-free identity baseline mismatch");
	}
	const strippedLiveHandoff = withoutC3PReceiptBlock(liveUnstrippedHandoff);
	for (const anchor of [
		c3vDidrunInterruptFinding,
		requiredC3MaintenanceDigest,
		requiredC3ADigest,
		requiredC3LDigest,
		requiredC3FDigest,
		requiredC3SDigest,
		requiredC3RDigest,
		requiredC3QDigest,
		requiredC3TDigest,
		requiredC3UDigest,
		lifecycleValue(c3pbHandoffTransitions[0]),
	]) {
		if (!strippedLiveHandoff.includes(anchor)) throw new Error("P07B-C C3P live receipt stripping dropped interposed authority");
	}
	const oneBlock = `prefix\n${markerStart}\nreceipt\n${markerEnd}\nsuffix\n`;
	if (withoutC3PReceiptBlock(oneBlock) !== "prefix\nsuffix\n") {
		throw new Error("P07B-C C3P receipt-block one-block removal baseline mismatch");
	}
	requireStripError("duplicate complete blocks", `${oneBlock}${markerStart}\nreceipt\n${markerEnd}\n`, "receipt marker cardinality");
	requireStripError("nested start marker", `prefix\n${markerStart}\n${markerStart}\nreceipt\n${markerEnd}\nsuffix\n`, "receipt marker cardinality");
	requireStripError("nested end marker", `prefix\n${markerStart}\nreceipt\n${markerEnd}\n${markerEnd}\nsuffix\n`, "receipt marker cardinality");
	requireStripError("start-only marker", `prefix\n${markerStart}\nsuffix\n`, "receipt marker cardinality");
	requireStripError("end-only marker", `prefix\n${markerEnd}\nsuffix\n`, "receipt marker cardinality");
	requireStripError("reversed markers", `prefix\n${markerEnd}\nreceipt\n${markerStart}\nsuffix\n`, "receipt marker order");
	requireStripError("inline start marker", `prefix ${markerStart}\nreceipt\n${markerEnd}\nsuffix\n`, "standalone LF lines");
	requireStripError("inline end marker", `prefix\n${markerStart}\nreceipt\n${markerEnd} suffix\n`, "standalone LF lines");
	const absentOverrides = await syntheticC3PAbsentPhaseFixture();
	const absentErrors = await checkPlan(repositoryRoot, absentOverrides);
	if (absentErrors.length > 0) throw new Error(`P07B-C C3P absent-phase baseline failed:\n${absentErrors.join("\n")}`);
	const liveStatus = absentOverrides.get(c3pStatusPath);
	const liveHandoff = absentOverrides.get("docs/HANDOFF_MODE_C.md");
	const liveVerification = await readText(repositoryRoot, "docs/VERIFICATION.md", absentOverrides);
	for (const [name, anchor] of [
		["C3V didrun finding", c3vDidrunInterruptFinding],
		["C3M scope digest", requiredC3MaintenanceDigest],
		["C3M active phase", "C3M receipt-phase checker maintenance is the active `SOURCE_FULL` unit"],
		["C3A scope digest", requiredC3ADigest],
		["C3L scope digest", requiredC3LDigest],
		["C3F scope digest", requiredC3FDigest],
		["C3S scope digest", requiredC3SDigest],
		["C3R scope digest", requiredC3RDigest],
		[`${lifecycle.phase} phase`, lifecycleValue(c3pbHandoffTransitions[0])],
	]) {
		if (!liveHandoff.includes(anchor)) throw new Error(`P07B-C C3P absent-phase fixture dropped carried ${name} authority`);
	}
	const rewriteOperationalPhase = (body, field, label) => c3pbHandoffTransitions.reduce(
		(current, transition) => {
			const metadataContract = c3pbMetadataLineContracts[transition.name];
			const before = lifecycleText(transition);
			const afterValue = c3pbTransitionValue(transition, field, lifecycle.receipt);
			const after = metadataContract === undefined
				? afterValue
				: metadataContract.render(
					afterValue, field, lifecycle.receipt, lifecycle.c3rAuthority, lifecycle.c3qAuthority, lifecycle.c3tAuthority, lifecycle.c3uAuthority,
				);
			return replaceFixtureExactlyOnce(current, before, after, `${label} ${transition.name}`);
		},
		body,
	);
	const relocateActiveTupleToHistory = (body, label) => {
		const transition = c3pbHandoffTransitions[0];
		const contradicted = replaceFixtureExactlyOnce(
			body,
			lifecycleValue(transition),
			"A differently worded current-state line claims an older phase is active.",
			label,
		);
		return `${contradicted}\n### Historical relocated operational text — inert\n\n> ${lifecycleValue(transition)}\n`;
	};
	const duplicateContradictoryMetadataSlot = (body, label) => {
		const firstSubheading = "\n### C3P/C3V/C3M/C3PB/C3A/C3L/C3F/C3S/C3/C3R/C3Q/C3T/C3U/C3B scope declarations\n";
		return replaceFixtureExactlyOnce(
			body,
			firstSubheading,
			`\n- **Pipeline phase:** contradictory duplicate says C3PB is active.\n${firstSubheading}`,
			label,
		);
	};
	const embedActiveTupleAsSameLineHistory = (body, label) => {
		const transition = c3pbHandoffTransitions[0];
		return replaceFixtureExactlyOnce(
			body,
			lifecycleValue(transition),
			`A contradictory current phrase says C3PB is active; historical bytes only: ${lifecycleValue(transition)}`,
			label,
		);
	};
	const wrapCurrentStateAsInertMarkdown = (body, opening, closing, label) => {
		const heading = "## Current state\n";
		const start = body.indexOf(heading);
		if (start < 0 || body.indexOf(heading, start + heading.length) >= 0) {
			throw new Error(`${label}: current-state heading anchor is not unique`);
		}
		const contentStart = start + heading.length;
		const nextHeading = body.indexOf("\n## ", contentStart);
		const contentEnd = nextHeading === -1 ? body.length : nextHeading;
		return `${body.slice(0, contentStart)}${opening}\n${body.slice(contentStart, contentEnd)}\n${closing}${body.slice(contentEnd)}`;
	};
	const absentOperationalPhaseCases = c3pbHandoffTransitions.flatMap((transition) => [
		{
			name: `receipt-absent missing ${transition.name}`, path: "docs/HANDOFF_MODE_C.md",
			value: replaceFixtureExactlyOnce(
				liveHandoff,
				lifecycleText(transition),
				`C3 ${transition.name} omitted.`,
				`receipt-absent missing ${transition.name}`,
			),
			expect: phaseInvalidDiagnostic,
		},
		{
			name: `receipt-absent duplicate ${transition.name}`, path: "docs/HANDOFF_MODE_C.md",
			value: `${liveHandoff}${lifecycleText(transition)}\n`,
			expect: phaseInvalidDiagnostic,
		},
		{
			name: `receipt-absent mixed ${transition.name}`, path: "docs/HANDOFF_MODE_C.md",
			value: `${liveHandoff}${transition.after}\n`,
			expect: phaseInvalidDiagnostic,
		},
	]);
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
			expect: "pending receipt marker topology invalid",
		},
		{
			name: "carried C3V authority drift", path: "docs/HANDOFF_MODE_C.md",
			value: replaceFixtureExactlyOnce(liveHandoff, c3vDidrunInterruptFinding, "C3V finding omitted.", "carried C3V authority"),
			expect: "missing required C3V ruling",
		},
		{
			name: "carried C3M authority drift", path: "docs/HANDOFF_MODE_C.md",
			value: replaceFixtureExactlyOnce(
				liveHandoff,
				"C3M receipt-phase checker maintenance is the active `SOURCE_FULL` unit",
				"C3M phase omitted",
				"carried C3M authority",
			),
			expect: "missing required C3M ruling",
		},
		{
			name: "receipt-absent coherent C3PB downgrade", path: "docs/HANDOFF_MODE_C.md",
			value: rewriteOperationalPhase(liveHandoff, "after", "receipt-absent C3PB downgrade"),
			expect: phaseMismatchDiagnostic,
		},
		{
			name: "receipt-absent coherent C3M downgrade", path: "docs/HANDOFF_MODE_C.md",
			value: rewriteOperationalPhase(liveHandoff, "before", "receipt-absent C3M downgrade"),
			expect: phaseMismatchDiagnostic,
		},
		{
			name: "receipt-absent coherent C3A downgrade", path: "docs/HANDOFF_MODE_C.md",
			value: rewriteOperationalPhase(liveHandoff, "descendant", "receipt-absent C3A downgrade"),
			expect: phaseMismatchDiagnostic,
		},
		{
			name: "receipt-absent coherent C3L downgrade", path: "docs/HANDOFF_MODE_C.md",
			value: rewriteOperationalPhase(liveHandoff, "c3l", "receipt-absent C3L downgrade"),
			expect: phaseMismatchDiagnostic,
		},
		{
			name: "receipt-absent coherent C3F downgrade", path: "docs/HANDOFF_MODE_C.md",
			value: rewriteOperationalPhase(liveHandoff, "c3f", "receipt-absent C3F downgrade"),
			expect: phaseMismatchDiagnostic,
		},
		{
			name: "receipt-absent coherent C3S downgrade", path: "docs/HANDOFF_MODE_C.md",
			value: rewriteOperationalPhase(liveHandoff, "c3s", "receipt-absent C3S downgrade"),
			expect: phaseMismatchDiagnostic,
		},
		{
			name: `receipt-absent ${lifecycle.receipt === undefined ? "C3U" : "C3B"} self receipt`, path: "docs/HANDOFF_MODE_C.md",
			value: `${liveHandoff}\n${lifecycle.receipt === undefined ? "C3U" : "C3B"} is sealed and strict-clean.\n`,
			expect: `${lifecycle.receipt === undefined ? "C3U" : "C3B"} self-receipt`,
		},
		{
			name: "receipt-absent relocated active tuple", path: "docs/HANDOFF_MODE_C.md",
			value: relocateActiveTupleToHistory(liveHandoff, "receipt-absent relocated active tuple"),
			expect: phaseInvalidDiagnostic,
		},
		{
			name: `receipt-absent external ${lifecycle.receipt === undefined ? "C3U" : "C3B"} self receipt`, path: "docs/VERIFICATION.md",
			value: `${liveVerification}\n${lifecycle.receipt === undefined ? "C3U" : "C3B"} strict exit: 0\n`,
			expect: `${lifecycle.receipt === undefined ? "C3U" : "C3B"} self-receipt`,
		},
		{
			name: "receipt-absent duplicate metadata slot", path: "docs/HANDOFF_MODE_C.md",
			value: duplicateContradictoryMetadataSlot(liveHandoff, "receipt-absent duplicate metadata slot"),
			expect: phaseInvalidDiagnostic,
		},
		{
			name: "receipt-absent same-line historical embedding", path: "docs/HANDOFF_MODE_C.md",
			value: embedActiveTupleAsSameLineHistory(liveHandoff, "receipt-absent same-line historical embedding"),
			expect: phaseInvalidDiagnostic,
		},
		{
			name: "receipt-absent HTML-commented active tuple", path: "docs/HANDOFF_MODE_C.md",
			value: wrapCurrentStateAsInertMarkdown(liveHandoff, "<!--", "-->", "receipt-absent HTML-commented active tuple"),
			expect: phaseInvalidDiagnostic,
		},
		{
			name: "receipt-absent fenced active tuple", path: "docs/HANDOFF_MODE_C.md",
			value: wrapCurrentStateAsInertMarkdown(liveHandoff, "```markdown", "```", "receipt-absent fenced active tuple"),
			expect: phaseInvalidDiagnostic,
		},
		{
			name: "receipt-absent raw-HTML active tuple", path: "docs/HANDOFF_MODE_C.md",
			value: wrapCurrentStateAsInertMarkdown(liveHandoff, '<script type="text/plain">', "</script>", "receipt-absent raw-HTML active tuple"),
			expect: phaseInvalidDiagnostic,
		},
		...absentOperationalPhaseCases,
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
	for (const anchor of [
		c3vDidrunInterruptFinding,
		requiredC3MaintenanceDigest,
		requiredC3ADigest,
		requiredC3LDigest,
		requiredC3FDigest,
		requiredC3SDigest,
		requiredC3RDigest,
		requiredC3QDigest,
		requiredC3TDigest,
		requiredC3UDigest,
		lifecycleValue(c3pbHandoffTransitions[0]),
	]) {
		if (!phase.overrides.get("docs/HANDOFF_MODE_C.md").includes(anchor)) {
			throw new Error("P07B-C C3P receipt-present fixture dropped interposed authority");
		}
	}
	const phaseErrors = await checkPlan(repositoryRoot, phase.overrides, undefined, undefined, undefined, phase.authority);
	if (phaseErrors.length > 0) throw new Error(`P07B-C C3P receipt-present phase baseline failed:\n${phaseErrors.join("\n")}`);
	const rerunPhase = await syntheticC3PReceiptPlanFixture(
		fixture,
		phase.overrides,
	);
	for (const path of [c3pStatusPath, "docs/HANDOFF_MODE_C.md", c3pReceiptDeclarationPath]) {
		if (rerunPhase.overrides.get(path) !== phase.overrides.get(path)) {
			throw new Error(`P07B-C C3P receipt-present rerun changed ${path}`);
		}
	}
	const rerunErrors = await checkPlan(repositoryRoot, rerunPhase.overrides, undefined, undefined, undefined, rerunPhase.authority);
	if (rerunErrors.length > 0) throw new Error(`P07B-C C3P receipt-present rerun baseline failed:\n${rerunErrors.join("\n")}`);
	phaseStabilityControls += 1;
	const operationalPhaseCases = c3pbHandoffTransitions.flatMap((transition) => [
		[
			`receipt-present missing ${transition.name}`,
			"docs/HANDOFF_MODE_C.md",
			(body) => replaceFixtureExactlyOnce(
				body,
				lifecycleText(transition),
				`C3 ${transition.name} omitted.`,
				`receipt-present missing ${transition.name}`,
			),
			phaseInvalidDiagnostic,
		],
		[
			`receipt-present duplicate ${transition.name}`,
			"docs/HANDOFF_MODE_C.md",
			(body) => `${body}${lifecycleText(transition)}\n`,
			phaseInvalidDiagnostic,
		],
		[
			`receipt-present mixed ${transition.name}`,
			"docs/HANDOFF_MODE_C.md",
			(body) => `${body}${transition.after}\n`,
			phaseInvalidDiagnostic,
		],
	]);
	for (const [name, path, mutate, expected] of [
		["receipt status grade", c3pStatusPath, (body) => body.replace(
			`| \`${c3pClaimLabels[0]}\` | \`${c3pClaimTypes[0]}\` | \`TREE-EXACT\` |`,
			`| \`${c3pClaimLabels[0]}\` | \`${c3pClaimTypes[0]}\` | \`UNRECEIPTED\` |`,
		), "source receipt map"],
		["receipt no-recursion", c3pStatusPath, (body) => body.replace(c3pReceiptNoRecursionBoundary, "Receipt recursion omitted."), "no-recursion"],
		["receipt handoff identity", "docs/HANDOFF_MODE_C.md", (body) => body.replace(fixture.receipt.source_tree, "5".repeat(40)), "receipt block payload mismatch"],
		["receipt-present C3V authority", "docs/HANDOFF_MODE_C.md", (body) => replaceFixtureExactlyOnce(
			body, c3vDidrunInterruptFinding, "C3V finding omitted.", "receipt-present C3V authority",
		), "missing required C3V ruling"],
		["receipt-present C3M authority", "docs/HANDOFF_MODE_C.md", (body) => replaceFixtureExactlyOnce(
			body,
			"C3M receipt-phase checker maintenance is the active `SOURCE_FULL` unit",
			"C3M phase omitted",
			"receipt-present C3M authority",
		), "missing required C3M ruling"],
		["receipt-present active phase", "docs/HANDOFF_MODE_C.md", (body) => replaceFixtureExactlyOnce(
			body,
			lifecycleValue(c3pbHandoffTransitions[0]),
			`${lifecycle.phase} phase omitted.`,
			"receipt-present active phase",
		), phaseInvalidDiagnostic],
		["receipt residue outside block", "docs/HANDOFF_MODE_C.md", (body) => `${body}C3P source commit \`${fixture.receipt.source_commit}\`, tree \`${fixture.receipt.source_tree}\`.\n`, "receipt residue outside block"],
		["receipt partially externalized payload", "docs/HANDOFF_MODE_C.md", (body) => {
			const parsed = parseC3PReceiptBlock(body);
			const disclosure = c3pReceiptDisclosureLines(fixture.receipt)[0];
			const movedBlock = replaceFixtureExactlyOnce(parsed.block, `${disclosure}\n\n`, "", "externalized receipt disclosure");
			return `${body.replace(parsed.block, movedBlock)}${disclosure}\n`;
		}, "receipt block payload mismatch"],
		["receipt nested marker", "docs/HANDOFF_MODE_C.md", (body) => body.replace(c3pReceiptBlockStart, `${c3pReceiptBlockStart}\n${c3pReceiptBlockStart}`), "receipt block invalid"],
		[`${lifecycle.receipt === undefined ? "C3U" : "C3B"} self receipt`, "docs/HANDOFF_MODE_C.md", (body) =>
			`${body}\n${lifecycle.receipt === undefined ? "C3U" : "C3B"} is sealed and strict-clean.\n`,
			`${lifecycle.receipt === undefined ? "C3U" : "C3B"} self-receipt`],
		["receipt-present coherent C3PB downgrade", "docs/HANDOFF_MODE_C.md", (body) => rewriteOperationalPhase(
			body, "after", "receipt-present C3PB downgrade",
			), phaseMismatchDiagnostic],
		["receipt-present coherent C3M downgrade", "docs/HANDOFF_MODE_C.md", (body) => rewriteOperationalPhase(
			body, "before", "receipt-present C3M downgrade",
		), phaseMismatchDiagnostic],
		["receipt-present coherent C3A downgrade", "docs/HANDOFF_MODE_C.md", (body) => rewriteOperationalPhase(
			body, "descendant", "receipt-present C3A downgrade",
		), phaseMismatchDiagnostic],
		["receipt-present coherent C3L downgrade", "docs/HANDOFF_MODE_C.md", (body) => rewriteOperationalPhase(
			body, "c3l", "receipt-present C3L downgrade",
		), phaseMismatchDiagnostic],
		["receipt-present coherent C3F downgrade", "docs/HANDOFF_MODE_C.md", (body) => rewriteOperationalPhase(
			body, "c3f", "receipt-present C3F downgrade",
		), phaseMismatchDiagnostic],
		["receipt-present coherent C3S downgrade", "docs/HANDOFF_MODE_C.md", (body) => rewriteOperationalPhase(
			body, "c3s", "receipt-present C3S downgrade",
		), phaseMismatchDiagnostic],
		["receipt-present relocated active tuple", "docs/HANDOFF_MODE_C.md", (body) => relocateActiveTupleToHistory(
			body, "receipt-present relocated active tuple",
		), phaseInvalidDiagnostic],
		["receipt-present external C3 self receipt", "docs/PROMPT_PACK.md", (body) =>
				`${body}\nC3 source evidence:\nstrict exit: 0\n`, "C3 canonical self-receipt marker"],
		["receipt-present duplicate metadata slot", "docs/HANDOFF_MODE_C.md", (body) =>
			duplicateContradictoryMetadataSlot(body, "receipt-present duplicate metadata slot"), phaseInvalidDiagnostic],
		["receipt-present same-line historical embedding", "docs/HANDOFF_MODE_C.md", (body) =>
			embedActiveTupleAsSameLineHistory(body, "receipt-present same-line historical embedding"), phaseInvalidDiagnostic],
		["receipt-present HTML-commented active tuple", "docs/HANDOFF_MODE_C.md", (body) =>
			wrapCurrentStateAsInertMarkdown(body, "<!--", "-->", "receipt-present HTML-commented active tuple"), phaseInvalidDiagnostic],
		["receipt-present fenced active tuple", "docs/HANDOFF_MODE_C.md", (body) =>
			wrapCurrentStateAsInertMarkdown(body, "```markdown", "```", "receipt-present fenced active tuple"), phaseInvalidDiagnostic],
		["receipt-present raw-HTML active tuple", "docs/HANDOFF_MODE_C.md", (body) =>
			wrapCurrentStateAsInertMarkdown(body, '<script type="text/plain">', "</script>", "receipt-present raw-HTML active tuple"), phaseInvalidDiagnostic],
		...operationalPhaseCases,
	]) {
		const overrides = new Map(phase.overrides);
		const sourceBody = overrides.has(path) ? overrides.get(path) : await readText(repositoryRoot, path, overrides);
		overrides.set(path, mutate(sourceBody));
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
	console.log(`P07B-C C3P receipt checker self-test passed: ${rejected} absent-phase, declaration, Git/note, receipt-phase, self-receipt, and local-snapshot mutations rejected; ${phaseStabilityControls} receipt-present rerun control passed`);
	return rejected;
}

function syntheticC3ReceiptFixture(receiptOverride) {
	const sourceCommit = sealedC3Identity.commit;
	const sourceTree = sealedC3Identity.tree;
	const sourceDiff = requiredC3Paths.map((path, index) => {
		const added = requiredC3AddedPaths.has(path);
		return {
			old_mode: added ? "000000" : "100644",
			new_mode: "100644",
			old_oid: added ? "0".repeat(40) : (index + 1).toString(16).padStart(40, "1").slice(-40),
			new_oid: (index + 1).toString(16).padStart(40, "2").slice(-40),
			status: added ? "A" : "M",
			path,
		};
	});
	const syntheticReceipt = {
		schema_version: "countershape/p07b-c-c3-receipt/v1",
		source_commit: sourceCommit,
		source_tree: sourceTree,
		source_subject: c3SourceSubject,
		secrets_override: true,
		strict_exit_code: 0,
		strict_claims_recorded_exact: c3ClaimLabels.length,
		strict_claims_total: c3ClaimLabels.length,
		coverage_complete_events: c3ClaimLabels.length,
		coverage_total_events: c3ClaimLabels.length,
		claims: c3ClaimLabels.map((label, index) => ({
			label, type: c3ClaimTypes[index], grade: "TREE-EXACT", supporting_event_index: index,
		})),
		evidence: {
			html_report: {
				path: `.countershape/evidence/p07b-c-c3-final-${sourceCommit.slice(0, 12)}.html`,
				sha256: "3".repeat(64), bytes: 8192, authority: "LOCAL_SNAPSHOT_NOT_PORTABLE_STRICT_WITNESS",
			},
			ledger_archive: {
				path: ".didrun-history/2026-07-19-p07b-c-c3-final/.didrun/",
				authority: "LOCAL_IGNORED_ARCHIVE_NOT_GIT_AUTHORITY",
				manifest_format: "sorted-relative-posix-path-tab-size-tab-sha256-newline/v1",
				sealed_event_count: c3ClaimLabels.length,
				archive_session_event_count: c3ClaimLabels.length,
				claim_count: c3ClaimLabels.length, seal_count: 1,
				object_file_count: 50, total_file_count: 54, total_bytes: 250_000,
				all_files_manifest_sha256: "4".repeat(64), objects_manifest_sha256: "5".repeat(64),
				gitignore_sha256: "6".repeat(64), session_log_sha256: "7".repeat(64),
				claims_jsonl_sha256: "8".repeat(64), seals_jsonl_sha256: "9".repeat(64),
			},
			git_note: {
				version: 1,
				blob_oid: sealedC3Identity.noteBlob,
				body_sha256: sealedC3Identity.noteBodySHA256,
			},
			seal_redaction: {
				finding_count: 1, finding_kinds: ["high-entropy"],
				finding_provenance: "LOCAL_TERMINAL_OBSERVATION_NOT_GIT_NOTE",
				structured_staged_scan_findings: 0,
				structured_scan_provenance: "SEALED_C3_SUPPORTING_EVENT_16",
			},
		},
	};
	const receipt = receiptOverride === undefined ? syntheticReceipt : structuredClone(receiptOverride);
	const authority = {
		commit: sourceCommit,
		tree: sourceTree,
		parent: sealedC3SIdentity.commit,
		parents: [sealedC3SIdentity.commit],
		source_diff: sourceDiff,
		subject: c3SourceSubject,
		ancestor_of_head: true,
		note_type: "blob",
		note_blob_oid: receipt.evidence.git_note.blob_oid,
		note_body_sha256: receipt.evidence.git_note.body_sha256,
		note: {
			version: 1,
			commit: sourceCommit,
			tree: sourceTree,
			secrets_override: true,
			claims: c3ClaimLabels.map((label, index) => ({
				claim: {
					argv_preview: [...c3ExpectedClaimArgv[index]], ctype: c3ClaimTypes[index],
					declared_at_index: index, event_indices: [index], label, pathspecs: [],
				},
				delta: [], exit_code: 0, grade: "tree-exact",
				reason: "self-stable command ran against the sealed tree", supporting_event_index: index,
			})),
			coverage: { by_coverage: { complete: c3ClaimLabels.length }, total_events: c3ClaimLabels.length },
		},
	};
	const c3rCommit = sealedC3RIdentity.commit;
	const c3rTree = sealedC3RIdentity.tree;
	const c3qCommit = sealedC3QIdentity.commit;
	const c3qTree = sealedC3QIdentity.tree;
	const c3tCommit = sealedC3TIdentity.commit;
	const c3tTree = sealedC3TIdentity.tree;
	const c3uCommit = "3".repeat(40);
	const c3uTree = "4".repeat(40);
	const c3rSourceDiff = requiredC3RPaths.map((path, index) => {
		const added = path === c3rStatusPath;
		return {
			old_mode: added ? "000000" : "100644",
			new_mode: "100644",
			old_oid: added ? "0".repeat(40) : (index + 17).toString(16).padStart(40, "3").slice(-40),
			new_oid: (index + 29).toString(16).padStart(40, "4").slice(-40),
			status: added ? "A" : "M",
			path,
		};
	});
	const c3rAuthority = {
		head_commit: c3uCommit,
		commit: c3rCommit,
		tree: c3rTree,
		parent: sourceCommit,
		parents: [sourceCommit],
		source_diff: c3rSourceDiff,
		subject: c3rSourceSubject,
		ancestor_of_head: true,
		note_type: "blob",
		note_blob_oid: sealedC3RIdentity.noteBlob,
		note_body_sha256: sealedC3RIdentity.noteBodySHA256,
		note: {
			version: 1,
			commit: c3rCommit,
			tree: c3rTree,
			secrets_override: true,
			claims: c3rClaimLabels.map((label, index) => ({
				claim: {
					argv_preview: [...c3rExpectedClaimArgv[index]], ctype: c3rClaimTypes[index],
					declared_at_index: index, event_indices: [index], label, pathspecs: [],
				},
				delta: [], exit_code: 0, grade: "tree-exact",
				reason: "self-stable command ran against the sealed tree", supporting_event_index: index,
			})),
			coverage: { by_coverage: { complete: c3rClaimLabels.length }, total_events: c3rClaimLabels.length },
		},
	};
	const c3qSourceDiff = requiredC3QPaths.map((path, index) => {
		const added = path === c3qStatusPath;
		return {
			old_mode: added ? "000000" : "100644",
			new_mode: "100644",
			old_oid: added ? "0".repeat(40) : (index + 41).toString(16).padStart(40, "5").slice(-40),
			new_oid: (index + 53).toString(16).padStart(40, "6").slice(-40),
			status: added ? "A" : "M",
			path,
		};
	});
	const c3qAuthority = {
		head_commit: c3uCommit,
		commit: c3qCommit,
		tree: c3qTree,
		parent: c3rCommit,
		parents: [c3rCommit],
		source_diff: c3qSourceDiff,
		subject: c3qSourceSubject,
		ancestor_of_head: true,
		note_type: "blob",
		note_blob_oid: sealedC3QIdentity.noteBlob,
		note_body_sha256: sealedC3QIdentity.noteBodySHA256,
		note: {
			version: 1,
			commit: c3qCommit,
			tree: c3qTree,
			secrets_override: true,
			claims: c3qClaimLabels.map((label, index) => ({
				claim: {
					argv_preview: [...c3qExpectedClaimArgv[index]], ctype: c3qClaimTypes[index],
					declared_at_index: index, event_indices: [index], label, pathspecs: [],
				},
				delta: [], exit_code: 0, grade: "tree-exact",
				reason: "self-stable command ran against the sealed tree", supporting_event_index: index,
			})),
			coverage: { by_coverage: { complete: c3qClaimLabels.length }, total_events: c3qClaimLabels.length },
		},
	};
	const c3tSourceDiff = requiredC3TPaths.map((path, index) => {
		const added = path === c3tStatusPath;
		return {
			old_mode: added ? "000000" : "100644",
			new_mode: "100644",
			old_oid: added ? "0".repeat(40) : (index + 67).toString(16).padStart(40, "7").slice(-40),
			new_oid: (index + 79).toString(16).padStart(40, "8").slice(-40),
			status: added ? "A" : "M",
			path,
		};
	});
	const c3tAuthority = {
		head_commit: c3uCommit,
		commit: c3tCommit,
		tree: c3tTree,
		parent: c3qCommit,
		parents: [c3qCommit],
		source_diff: c3tSourceDiff,
		subject: c3tSourceSubject,
		ancestor_of_head: true,
		note_type: "blob",
		note_blob_oid: sealedC3TIdentity.noteBlob,
		note_body_sha256: sealedC3TIdentity.noteBodySHA256,
		note: {
			version: 1,
			commit: c3tCommit,
			tree: c3tTree,
			secrets_override: true,
			claims: c3tClaimLabels.map((label, index) => ({
				claim: {
					argv_preview: [...c3tExpectedClaimArgv[index]], ctype: c3tClaimTypes[index],
					declared_at_index: index, event_indices: [index], label, pathspecs: [],
				},
				delta: [], exit_code: 0, grade: "tree-exact",
				reason: "self-stable command ran against the sealed tree", supporting_event_index: index,
			})),
			coverage: { by_coverage: { complete: c3tClaimLabels.length }, total_events: c3tClaimLabels.length },
		},
	};
	const c3uSourceDiff = requiredC3UPaths.map((path, index) => {
		const added = path === c3uStatusPath;
		return {
			old_mode: added ? "000000" : "100644",
			new_mode: "100644",
			old_oid: added ? "0".repeat(40) : (index + 91).toString(16).padStart(40, "9").slice(-40),
			new_oid: (index + 103).toString(16).padStart(40, "a").slice(-40),
			status: added ? "A" : "M",
			path,
		};
	});
	const c3uAuthority = {
		head_commit: c3uCommit,
		commit: c3uCommit,
		tree: c3uTree,
		parent: c3tCommit,
		parents: [c3tCommit],
		source_diff: c3uSourceDiff,
		subject: c3uSourceSubject,
		ancestor_of_head: true,
		note_type: "blob",
		note_blob_oid: "b".repeat(40),
		note_body_sha256: "c".repeat(64),
		note: {
			version: 1,
			commit: c3uCommit,
			tree: c3uTree,
			secrets_override: false,
			claims: c3uClaimLabels.map((label, index) => ({
				claim: {
					argv_preview: [...c3uExpectedClaimArgv[index]], ctype: c3uClaimTypes[index],
					declared_at_index: index, event_indices: [index], label, pathspecs: [],
				},
				delta: [], exit_code: 0, grade: "tree-exact",
				reason: "self-stable command ran against the sealed tree", supporting_event_index: index,
			})),
			coverage: { by_coverage: { complete: c3uClaimLabels.length }, total_events: c3uClaimLabels.length },
		},
	};
	return { receipt, authority, c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority };
}

function c3ReceiptStatusMetadata(receipt) {
	return [
		"## C3 source receipt map",
		"- C3 source strict exit: `0`",
		`- C3 source strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\``,
		c3ReceiptNoRecursionBoundary,
		...c3ReceiptDisclosureLines(receipt),
	].join("\n");
}

async function syntheticC3AbsentPlanFixture(fixture, current = new Map()) {
	let status = current.get(c3StatusPath) ?? await readText(repositoryRoot, c3StatusPath, current);
	let handoff = current.get("docs/HANDOFF_MODE_C.md") ?? await readText(repositoryRoot, "docs/HANDOFF_MODE_C.md", current);
	let phase;
	const receiptOverride = current.get(c3ReceiptDeclarationPath);
	const receiptPresent = receiptOverride !== ABSENT_FIXTURE_PATH &&
		(receiptOverride !== undefined || !status.includes(c3PendingStatusState));
	if (receiptPresent) {
		phase = classifyC3PBHandoffPhase(
			handoff, fixture.receipt, fixture.c3rAuthority, fixture.c3qAuthority, fixture.c3tAuthority, fixture.c3uAuthority,
		);
		if (phase !== "C3B_ACTIVE") {
			throw new Error(`P07B-C C3 inverse fixture requires C3B-active input, found ${phase}`);
		}
		status = replaceFixtureExactlyOnce(
			status, c3ReconciledStatusState(fixture.receipt), c3PendingStatusState, "C3B to C3U source state",
		);
		status = replaceFixtureExactlyOnce(
			status, c3ReceiptStatusMetadata(fixture.receipt), "## Intended C3 source receipt map", "C3B to C3U receipt metadata",
		);
		for (const claim of fixture.receipt.claims) {
			status = replaceFixtureExactlyOnce(
				status,
				`| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`,
				`| \`${claim.label}\` | \`${claim.type}\` | \`UNRECEIPTED\` |`,
				`C3B to C3U source receipt row ${claim.supporting_event_index}`,
			);
		}
		handoff = replaceCurrentStateSubsection(
			handoff,
			"### Active C3B source receipt reconciliation",
			c3uActiveMaintenanceSection(),
		);
		handoff = c3pbHandoffTransitions.filter(({ name }) =>
			name !== "maintenance heading" && name !== "receipt presence").reduce((body, transition) => {
			const metadata = c3pbMetadataLineContracts[transition.name];
			const beforeValue = c3pbTransitionValue(transition, "c3b", fixture.receipt);
			const before = metadata === undefined
				? beforeValue
				: metadata.render(
					beforeValue, "c3b", fixture.receipt, fixture.c3rAuthority, fixture.c3qAuthority, fixture.c3tAuthority, fixture.c3uAuthority,
				);
			const after = metadata === undefined
				? transition.c3u
				: metadata.render(transition.c3u, "c3u");
			return replaceFixtureExactlyOnce(body, before, after, `C3B to C3U ${transition.name}`);
		}, handoff);
		const parsed = parseC3ReceiptBlock(handoff);
		const expectedBlock = c3ReceiptHandoffBlock(fixture.receipt);
		if (parsed.block !== expectedBlock) {
			throw new Error("P07B-C C3 inverse fixture requires the exact canonical receipt block");
		}
		handoff = parsed.stripped;
	} else {
		if (!status.includes(c3PendingStatusState) || !status.includes("## Intended C3 source receipt map")) {
			throw new Error("P07B-C C3 absent fixture status is not the canonical pending projection");
		}
		phase = classifyC3PBHandoffPhase(handoff);
		if (phase !== "C3U_ACTIVE") {
			throw new Error(`P07B-C C3 absent fixture requires C3U-active input, found ${phase}`);
		}
		if (parseC3ReceiptBlock(handoff).block !== undefined) {
			throw new Error("P07B-C C3 absent fixture retains a receipt block");
		}
	}
	const absent = new Map([
		...current,
		[c3StatusPath, status],
		["docs/HANDOFF_MODE_C.md", handoff],
		[c3ReceiptDeclarationPath, ABSENT_FIXTURE_PATH],
	]);
	const errors = await checkPlan(repositoryRoot, absent);
	if (errors.length > 0) {
		throw new Error(`P07B-C C3 validated C3U-active absent fixture failed:\n${errors.join("\n")}`);
	}
	return absent;
}

async function syntheticC3ReceiptPlanFixture(fixture, current = new Map()) {
	const absent = await syntheticC3AbsentPlanFixture(fixture, current);
	let status = absent.get(c3StatusPath);
	let handoff = absent.get("docs/HANDOFF_MODE_C.md");
	status = replaceFixtureExactlyOnce(
		status, c3PendingStatusState, c3ReconciledStatusState(fixture.receipt), "C3U to C3B source state",
	);
	status = replaceFixtureExactlyOnce(
		status, "## Intended C3 source receipt map", c3ReceiptStatusMetadata(fixture.receipt), "C3U to C3B receipt metadata",
	);
	for (const claim of fixture.receipt.claims) {
		status = replaceFixtureExactlyOnce(
			status,
			`| \`${claim.label}\` | \`${claim.type}\` | \`UNRECEIPTED\` |`,
			`| \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`,
			`C3U to C3B source receipt row ${claim.supporting_event_index}`,
		);
	}
	const phase = classifyC3PBHandoffPhase(handoff);
	if (phase !== "C3U_ACTIVE") {
		throw new Error(`P07B-C C3 receipt-present renderer requires C3U-active input, found ${phase}`);
	}
	handoff = replaceCurrentStateSubsection(
		handoff,
		"### Active C3U cross-phase receipt-fixture maintenance",
		c3bActiveMaintenanceSection(fixture.receipt, fixture.c3rAuthority, fixture.c3qAuthority, fixture.c3tAuthority, fixture.c3uAuthority),
	);
	handoff = c3pbHandoffTransitions.filter(({ name }) =>
		name !== "maintenance heading" && name !== "receipt presence").reduce((body, transition) => {
		const metadata = c3pbMetadataLineContracts[transition.name];
		const before = metadata === undefined
			? transition.c3u
			: metadata.render(transition.c3u, "c3u");
		const afterValue = c3pbTransitionValue(transition, "c3b", fixture.receipt);
		const after = metadata === undefined
			? afterValue
			: metadata.render(
				afterValue, "c3b", fixture.receipt, fixture.c3rAuthority, fixture.c3qAuthority, fixture.c3tAuthority, fixture.c3uAuthority,
			);
		return replaceFixtureExactlyOnce(body, before, after, `C3U to C3B ${transition.name}`);
	}, handoff);
	if (!handoff.endsWith("\n") || parseC3ReceiptBlock(handoff).block !== undefined) {
		throw new Error("P07B-C C3 receipt-present renderer requires a terminal-LF block-free handoff");
	}
	handoff = `${handoff}${c3ReceiptHandoffBlock(fixture.receipt)}\n`;
	return new Map([
		...absent,
		[c3StatusPath, status],
		["docs/HANDOFF_MODE_C.md", handoff],
		[c3ReceiptDeclarationPath, `${JSON.stringify(fixture.receipt)}\n`],
	]);
}

export async function runC3ReceiptSelfTest() {
	const liveErrors = await checkPlan();
	if (liveErrors.length > 0) throw new Error(`P07B-C C3 receipt checker live baseline failed:\n${liveErrors.join("\n")}`);
	let liveReceipt;
	let liveReceiptText;
	let liveControlledPaths;
	try {
		liveReceiptText = await readText(repositoryRoot, c3ReceiptDeclarationPath, new Map());
		liveReceipt = JSON.parse(liveReceiptText);
	} catch (error) {
		if (error.code !== "ENOENT") throw new Error(`P07B-C C3 live receipt fixture unavailable: ${error.message}`);
	}
	const fixture = syntheticC3ReceiptFixture(liveReceipt);
	if (liveReceipt !== undefined) {
		const liveReceiptErrors = validateC3ReceiptDeclaration(liveReceipt);
		if (liveReceiptErrors.length > 0) {
			throw new Error(`P07B-C C3 live receipt fixture invalid: ${liveReceiptErrors.join(", ")}`);
		}
		fixture.authority = await loadReceiptAuthorityFromGit(repositoryRoot, liveReceipt);
		fixture.c3rAuthority = await loadC3RAuthorityFromGit(repositoryRoot);
		fixture.c3qAuthority = await loadC3QAuthorityFromGit(repositoryRoot, fixture.c3rAuthority.commit);
		fixture.c3tAuthority = await loadC3TAuthorityFromGit(repositoryRoot, fixture.c3qAuthority.commit);
		fixture.c3uAuthority = await loadC3UAuthorityFromGit(repositoryRoot, fixture.c3tAuthority.commit);
		liveControlledPaths = new Map([
			[c3StatusPath, await readText(repositoryRoot, c3StatusPath, new Map())],
			["docs/HANDOFF_MODE_C.md", await readText(repositoryRoot, "docs/HANDOFF_MODE_C.md", new Map())],
			[c3ReceiptDeclarationPath, liveReceiptText],
		]);
	}
	let rejected = 0;
	const requireError = (name, errors, expected) => {
		if (!errors.some((error) => error.includes(expected))) {
			throw new Error(`P07B-C C3 receipt checker self-test false negative: ${name} (${errors.join("; ")})`);
		}
		rejected += 1;
	};
	const declarationBaseline = validateC3ReceiptDeclaration(fixture.receipt);
	if (declarationBaseline.length > 0) throw new Error(`P07B-C C3 receipt declaration baseline failed: ${declarationBaseline.join(", ")}`);
	const authorityBaseline = validateC3ReceiptAuthority(fixture.receipt, fixture.authority);
	if (authorityBaseline.length > 0) throw new Error(`P07B-C C3 receipt authority baseline failed: ${authorityBaseline.join(", ")}`);
	const c3rAuthorityBaseline = validateSealedC3RAuthority(fixture.c3rAuthority);
	if (c3rAuthorityBaseline.length > 0) {
		throw new Error(`P07B-C C3R predecessor authority baseline failed: ${c3rAuthorityBaseline.join(", ")}`);
	}
	const c3qPredecessorBaseline = validateSealedC3QAuthority(fixture.c3qAuthority);
	if (c3qPredecessorBaseline.length > 0) {
		throw new Error(`P07B-C sealed C3Q predecessor baseline failed: ${c3qPredecessorBaseline.join(", ")}`);
	}
	const c3tPresealC3Q = structuredClone(fixture.c3qAuthority);
	c3tPresealC3Q.head_commit = c3tPresealC3Q.commit;
	const c3tPresealC3QErrors = validateC3TPredecessorAuthority(c3tPresealC3Q);
	if (c3tPresealC3QErrors.length > 0) {
		throw new Error(`P07B-C C3T sealed-C3Q current-HEAD baseline failed: ${c3tPresealC3QErrors.join(", ")}`);
	}
	const staleC3TPresealC3Q = structuredClone(c3tPresealC3Q);
	staleC3TPresealC3Q.head_commit = fixture.c3tAuthority.commit;
	requireError("C3T sealed-C3Q descendant HEAD", validateC3TPredecessorAuthority(staleC3TPresealC3Q), "sealed C3Q is not current HEAD");
	const c3tAuthorityBaseline = validateC3TAuthority(
		fixture.c3tAuthority,
		fixture.c3qAuthority.commit,
	);
	if (c3tAuthorityBaseline.length > 0) {
		throw new Error(`P07B-C C3T sealed-ancestor baseline failed: ${c3tAuthorityBaseline.join(", ")}`);
	}
	const sealedC3TBaseline = validateSealedC3TAuthority(fixture.c3tAuthority);
	if (sealedC3TBaseline.length > 0) {
		throw new Error(`P07B-C sealed C3T predecessor baseline failed: ${sealedC3TBaseline.join(", ")}`);
	}
	const c3uPresealC3T = structuredClone(fixture.c3tAuthority);
	c3uPresealC3T.head_commit = c3uPresealC3T.commit;
	const c3uPredecessorBaseline = validateC3UPredecessorAuthority(c3uPresealC3T);
	if (c3uPredecessorBaseline.length > 0) {
		throw new Error(`P07B-C C3U sealed-C3T current-HEAD baseline failed: ${c3uPredecessorBaseline.join(", ")}`);
	}
	const staleC3UPresealC3T = structuredClone(c3uPresealC3T);
	staleC3UPresealC3T.head_commit = fixture.c3uAuthority.commit;
	requireError("C3U sealed-C3T descendant HEAD", validateC3UPredecessorAuthority(staleC3UPresealC3T), "sealed C3T is not current HEAD");
	const c3uAuthorityBaseline = validateC3UAuthority(fixture.c3uAuthority, fixture.c3tAuthority.commit);
	if (c3uAuthorityBaseline.length > 0) {
		throw new Error(`P07B-C C3U sealed-ancestor baseline failed: ${c3uAuthorityBaseline.join(", ")}`);
	}
	const c3bPresealC3U = structuredClone(fixture.c3uAuthority);
	c3bPresealC3U.head_commit = c3bPresealC3U.commit;
	const c3bPredecessorBaseline = validateC3BPredecessorAuthority(c3bPresealC3U, fixture.c3tAuthority.commit);
	if (c3bPredecessorBaseline.length > 0) {
		throw new Error(`P07B-C C3B preseal predecessor-head baseline failed: ${c3bPredecessorBaseline.join(", ")}`);
	}
	const plainSealC3RAuthority = structuredClone(fixture.c3rAuthority);
	plainSealC3RAuthority.note.secrets_override = false;
	const plainSealC3TAuthority = structuredClone(fixture.c3tAuthority);
	plainSealC3TAuthority.note.secrets_override = false;
	const plainSealC3UAuthority = structuredClone(fixture.c3uAuthority);
	plainSealC3UAuthority.note.secrets_override = false;
	const c3bLedgerTransition = c3pbHandoffTransitions.find((transition) => transition.name === "ledger profile");
	if (c3bLedgerTransition === undefined) throw new Error("P07B-C C3B ledger-profile transition is absent");
	const c3bLedgerValue = c3pbTransitionValue(c3bLedgerTransition, "c3b", fixture.receipt);
	const renderC3BLedgerProfile = (c3rAuthority, c3qAuthority, c3tAuthority, c3uAuthority) => c3pbMetadataLineContracts["ledger profile"].render(
		c3bLedgerValue,
		"c3b",
		fixture.receipt,
		c3rAuthority,
		c3qAuthority,
		c3tAuthority,
		c3uAuthority,
	);
	const mixedLedgerProfile = renderC3BLedgerProfile(fixture.c3rAuthority, fixture.c3qAuthority, fixture.c3tAuthority, fixture.c3uAuthority);
	const plainC3RLedgerProfile = renderC3BLedgerProfile(plainSealC3RAuthority, fixture.c3qAuthority, fixture.c3tAuthority, fixture.c3uAuthority);
	const plainC3TLedgerProfile = renderC3BLedgerProfile(fixture.c3rAuthority, fixture.c3qAuthority, plainSealC3TAuthority, fixture.c3uAuthority);
	const plainC3ULedgerProfile = renderC3BLedgerProfile(fixture.c3rAuthority, fixture.c3qAuthority, fixture.c3tAuthority, plainSealC3UAuthority);
	if (!mixedLedgerProfile.includes(c3rSealOutcomeDisclosure(fixture.c3rAuthority)) ||
		!mixedLedgerProfile.includes(c3qSealOutcomeDisclosure(fixture.c3qAuthority)) ||
		!mixedLedgerProfile.includes(c3tSealOutcomeDisclosure(fixture.c3tAuthority)) ||
		!mixedLedgerProfile.includes(c3uSealOutcomeDisclosure(fixture.c3uAuthority))) {
		throw new Error("P07B-C C3B four-seal disclosure renderer baseline failed");
	}
	if (!plainC3RLedgerProfile.includes(c3rSealOutcomeDisclosure(plainSealC3RAuthority)) ||
		!plainC3RLedgerProfile.includes(c3qSealOutcomeDisclosure(fixture.c3qAuthority)) ||
		!plainC3TLedgerProfile.includes(c3tSealOutcomeDisclosure(plainSealC3TAuthority)) ||
		!plainC3ULedgerProfile.includes(c3uSealOutcomeDisclosure(plainSealC3UAuthority))) {
		throw new Error("P07B-C C3B independent seal disclosure renderer baseline failed");
	}
	const malformedSealC3RAuthority = structuredClone(fixture.c3rAuthority);
	malformedSealC3RAuthority.note.secrets_override = "false";
	let malformedSealRendererError = "";
	try {
		renderC3BLedgerProfile(malformedSealC3RAuthority, fixture.c3qAuthority, fixture.c3tAuthority, fixture.c3uAuthority);
	} catch (error) {
		malformedSealRendererError = error.message;
	}
	if (!malformedSealRendererError.includes("exact C3R secrets override disclosure")) {
		throw new Error(`P07B-C C3B seal disclosure renderer self-test false negative: ${malformedSealRendererError || "accepted"}`);
	}
	rejected += 1;
	const malformedSealC3QAuthority = structuredClone(fixture.c3qAuthority);
	malformedSealC3QAuthority.note.secrets_override = "false";
	malformedSealRendererError = "";
	try {
		renderC3BLedgerProfile(fixture.c3rAuthority, malformedSealC3QAuthority, fixture.c3tAuthority, fixture.c3uAuthority);
	} catch (error) {
		malformedSealRendererError = error.message;
	}
	if (!malformedSealRendererError.includes("exact C3Q note identity and secrets override disclosure")) {
		throw new Error(`P07B-C C3B C3Q seal disclosure renderer self-test false negative: ${malformedSealRendererError || "accepted"}`);
	}
	rejected += 1;
	const malformedSealC3TAuthority = structuredClone(fixture.c3tAuthority);
	malformedSealC3TAuthority.note.secrets_override = "false";
	malformedSealRendererError = "";
	try {
		renderC3BLedgerProfile(fixture.c3rAuthority, fixture.c3qAuthority, malformedSealC3TAuthority, fixture.c3uAuthority);
	} catch (error) {
		malformedSealRendererError = error.message;
	}
	if (!malformedSealRendererError.includes("exact C3T note identity and secrets override disclosure")) {
		throw new Error(`P07B-C C3B C3T seal disclosure renderer self-test false negative: ${malformedSealRendererError || "accepted"}`);
	}
	rejected += 1;
	const malformedSealC3UAuthority = structuredClone(fixture.c3uAuthority);
	malformedSealC3UAuthority.note.secrets_override = "false";
	malformedSealRendererError = "";
	try {
		renderC3BLedgerProfile(fixture.c3rAuthority, fixture.c3qAuthority, fixture.c3tAuthority, malformedSealC3UAuthority);
	} catch (error) {
		malformedSealRendererError = error.message;
	}
	if (!malformedSealRendererError.includes("exact C3U note identity and secrets override disclosure")) {
		throw new Error(`P07B-C C3B C3U seal disclosure renderer self-test false negative: ${malformedSealRendererError || "accepted"}`);
	}
	rejected += 1;
	const descendantHead = structuredClone(c3bPresealC3U);
	descendantHead.head_commit = "d".repeat(40);
	requireError(
		"C3B descendant HEAD",
		validateC3BPredecessorAuthority(descendantHead, fixture.c3tAuthority.commit),
		"C3U is not current HEAD",
	);
	const exactC3RNoteBytes = Buffer.from(JSON.stringify(fixture.c3rAuthority.note), "utf8");
	const rawC3RDiff = Buffer.from(fixture.c3rAuthority.source_diff.map((row) =>
		`:${row.old_mode} ${row.new_mode} ${row.old_oid} ${row.new_oid} ${row.status}\0${row.path}\0`).join(""), "utf8");
	const c3rLoaderRunner = (ancestry = Buffer.from(
		`${fixture.c3qAuthority.commit} ${fixture.c3rAuthority.commit}\n${fixture.c3rAuthority.commit} ${fixture.receipt.source_commit}\n`,
		"utf8",
	)) => {
		let noteListCalls = 0;
		return (args, accepted = [0]) => {
			let status = 0;
			let stdout;
			if (args[0] === "rev-list") {
				stdout = ancestry;
			} else if (args[0] === "rev-parse" && args[2] === "HEAD^{commit}") {
				stdout = Buffer.from(`${fixture.c3rAuthority.head_commit}\n`, "utf8");
			} else if (args[0] === "rev-parse" && args[2] === `${fixture.c3rAuthority.commit}^{commit}`) {
				stdout = Buffer.from(`${fixture.c3rAuthority.commit}\n`, "utf8");
			} else if (args[0] === "rev-parse" && args[2] === `${fixture.c3rAuthority.commit}^{tree}`) {
				stdout = Buffer.from(`${fixture.c3rAuthority.tree}\n`, "utf8");
			} else if (args[0] === "show" && args[2] === "--format=%P") {
				stdout = Buffer.from(`${fixture.c3rAuthority.parents.join(" ")}\n`, "utf8");
			} else if (args[0] === "diff-tree") {
				stdout = rawC3RDiff;
			} else if (args[0] === "show" && args[2] === "--format=%s") {
				stdout = Buffer.from(`${fixture.c3rAuthority.subject}\n`, "utf8");
			} else if (args[0] === "merge-base") {
				stdout = Buffer.alloc(0);
			} else if (args[0] === "notes" && args[2] === "list") {
				noteListCalls += 1;
				stdout = Buffer.from(`${fixture.c3rAuthority.note_blob_oid}\n`, "utf8");
			} else if (args[0] === "cat-file" && args[1] === "-t") {
				stdout = Buffer.from("blob\n", "utf8");
			} else if (args[0] === "cat-file" && args[1] === "blob") {
				stdout = exactC3RNoteBytes;
			} else {
				throw new Error(`synthetic C3R loader received unexpected Git argv: ${args.join(" ")}`);
			}
			if (!accepted.includes(status)) throw new Error(`synthetic C3R loader status ${status} not accepted`);
			if (noteListCalls > 2) throw new Error("synthetic C3R loader listed the note too many times");
			return { status, stdout };
		};
	};
	const loadedC3RAuthority = loadC3RAuthorityWithRunner(c3rLoaderRunner(), fixture.receipt.source_commit);
	const loadedC3RErrors = validateC3RAuthority(loadedC3RAuthority, fixture.receipt.source_commit);
	if (loadedC3RErrors.length > 0) {
		throw new Error(`P07B-C C3R direct-child loader baseline failed: ${loadedC3RErrors.join(", ")}`);
	}
	for (const [name, ancestry, expected] of [
		["absent ancestry", Buffer.alloc(0), "absent or not exact LF-framed"],
		["ancestry framing", Buffer.from(`${fixture.c3rAuthority.commit} ${fixture.receipt.source_commit}`, "utf8"), "absent or not exact LF-framed"],
		["malformed ancestry", Buffer.from("not-an-object\n", "utf8"), "ancestry row is malformed"],
		["zero direct children", Buffer.from(`${fixture.c3rAuthority.commit} ${"3".repeat(40)}\n`, "utf8"), "direct-child cardinality 0"],
		["multiple direct children", Buffer.from(
			`${fixture.c3rAuthority.commit} ${fixture.receipt.source_commit}\n${"d".repeat(40)} ${fixture.receipt.source_commit}\n`,
			"utf8",
		), "direct-child cardinality 2"],
	]) {
		let message = "";
		try {
			loadC3RAuthorityWithRunner(c3rLoaderRunner(ancestry), fixture.receipt.source_commit);
		} catch (error) {
			message = error.message;
		}
		if (!message.includes(expected)) {
			throw new Error(`P07B-C C3R loader self-test false negative: ${name} (${message || "accepted"})`);
		}
		rejected += 1;
	}
	const exactC3QNoteBytes = Buffer.from(JSON.stringify(fixture.c3qAuthority.note), "utf8");
	const rawC3QDiff = Buffer.from(fixture.c3qAuthority.source_diff.map((row) =>
		`:${row.old_mode} ${row.new_mode} ${row.old_oid} ${row.new_oid} ${row.status}\0${row.path}\0`).join(""), "utf8");
	const c3qLoaderRunner = (ancestry = Buffer.from(
		`${fixture.c3qAuthority.commit} ${fixture.c3rAuthority.commit}\n`,
		"utf8",
	)) => {
		let noteListCalls = 0;
		let ancestryCalls = 0;
		return (args, accepted = [0]) => {
			let status = 0;
			let stdout;
			if (args[0] === "rev-list") {
				ancestryCalls += 1;
				if (ancestryCalls !== 1 || !isDeepStrictEqual(args, [
					"rev-list", "--parents", "--ancestry-path", `${fixture.c3rAuthority.commit}..HEAD`,
				])) {
					throw new Error(`synthetic C3Q loader received noncanonical ancestry argv: ${args.join(" ")}`);
				}
				stdout = ancestry;
			} else if (args[0] === "rev-parse" && args[2] === "HEAD^{commit}") {
				stdout = Buffer.from(`${fixture.c3qAuthority.head_commit}\n`, "utf8");
			} else if (args[0] === "rev-parse" && args[2] === `${fixture.c3qAuthority.commit}^{commit}`) {
				stdout = Buffer.from(`${fixture.c3qAuthority.commit}\n`, "utf8");
			} else if (args[0] === "rev-parse" && args[2] === `${fixture.c3qAuthority.commit}^{tree}`) {
				stdout = Buffer.from(`${fixture.c3qAuthority.tree}\n`, "utf8");
			} else if (args[0] === "show" && args[2] === "--format=%P") {
				stdout = Buffer.from(`${fixture.c3qAuthority.parents.join(" ")}\n`, "utf8");
			} else if (args[0] === "diff-tree") {
				stdout = rawC3QDiff;
			} else if (args[0] === "show" && args[2] === "--format=%s") {
				stdout = Buffer.from(`${fixture.c3qAuthority.subject}\n`, "utf8");
			} else if (args[0] === "merge-base") {
				stdout = Buffer.alloc(0);
			} else if (args[0] === "notes" && args[2] === "list") {
				noteListCalls += 1;
				stdout = Buffer.from(`${fixture.c3qAuthority.note_blob_oid}\n`, "utf8");
			} else if (args[0] === "cat-file" && args[1] === "-t") {
				stdout = Buffer.from("blob\n", "utf8");
			} else if (args[0] === "cat-file" && args[1] === "blob") {
				stdout = exactC3QNoteBytes;
			} else {
				throw new Error(`synthetic C3Q loader received unexpected Git argv: ${args.join(" ")}`);
			}
			if (!accepted.includes(status)) throw new Error(`synthetic C3Q loader status ${status} not accepted`);
			if (noteListCalls > 2) throw new Error("synthetic C3Q loader listed the note too many times");
			return { status, stdout };
		};
	};
	const loadedC3QAuthority = loadC3QAuthorityWithRunner(c3qLoaderRunner(), fixture.c3rAuthority.commit);
	const loadedC3QErrors = validateC3QAuthority(loadedC3QAuthority, fixture.c3rAuthority.commit);
	if (loadedC3QErrors.length > 0) {
		throw new Error(`P07B-C C3Q direct-child loader baseline failed: ${loadedC3QErrors.join(", ")}`);
	}
	for (const [name, ancestry, expected] of [
		["C3Q absent ancestry", Buffer.alloc(0), "absent or not exact LF-framed"],
		["C3Q ancestry framing", Buffer.from(`${fixture.c3qAuthority.commit} ${fixture.c3rAuthority.commit}`, "utf8"), "absent or not exact LF-framed"],
		["C3Q ancestry CR framing", Buffer.from(`${fixture.c3qAuthority.commit} ${fixture.c3rAuthority.commit}\r\n`, "utf8"), "absent or not exact LF-framed"],
		["C3Q malformed ancestry", Buffer.from("not-an-object\n", "utf8"), "ancestry row is malformed"],
		["C3Q zero direct children", Buffer.from(`${fixture.c3qAuthority.commit} ${"9".repeat(40)}\n`, "utf8"), "direct-child cardinality 0"],
		["C3Q multiple direct children", Buffer.from(
			`${fixture.c3qAuthority.commit} ${fixture.c3rAuthority.commit}\n${"9".repeat(40)} ${fixture.c3rAuthority.commit}\n`,
			"utf8",
		), "direct-child cardinality 2"],
	]) {
		let message = "";
		try {
			loadC3QAuthorityWithRunner(c3qLoaderRunner(ancestry), fixture.c3rAuthority.commit);
		} catch (error) {
			message = error.message;
		}
		if (!message.includes(expected)) {
			throw new Error(`P07B-C C3Q loader self-test false negative: ${name} (${message || "accepted"})`);
		}
		rejected += 1;
	}
	const exactC3TNoteBytes = Buffer.from(JSON.stringify(fixture.c3tAuthority.note), "utf8");
	const rawC3TDiff = Buffer.from(fixture.c3tAuthority.source_diff.map((row) =>
		`:${row.old_mode} ${row.new_mode} ${row.old_oid} ${row.new_oid} ${row.status}\0${row.path}\0`).join(""), "utf8");
	const c3tLoaderRunner = (ancestry = Buffer.from(
		`${fixture.c3tAuthority.commit} ${fixture.c3qAuthority.commit}\n`,
		"utf8",
	)) => {
		let noteListCalls = 0;
		let ancestryCalls = 0;
		return (args, accepted = [0]) => {
			let status = 0;
			let stdout;
			if (args[0] === "rev-list") {
				ancestryCalls += 1;
				if (ancestryCalls !== 1 || !isDeepStrictEqual(args, [
					"rev-list", "--parents", "--ancestry-path", `${fixture.c3qAuthority.commit}..HEAD`,
				])) {
					throw new Error(`synthetic C3T loader received noncanonical ancestry argv: ${args.join(" ")}`);
				}
				stdout = ancestry;
			} else if (args[0] === "rev-parse" && args[2] === "HEAD^{commit}") {
				stdout = Buffer.from(`${fixture.c3tAuthority.head_commit}\n`, "utf8");
			} else if (args[0] === "rev-parse" && args[2] === `${fixture.c3tAuthority.commit}^{commit}`) {
				stdout = Buffer.from(`${fixture.c3tAuthority.commit}\n`, "utf8");
			} else if (args[0] === "rev-parse" && args[2] === `${fixture.c3tAuthority.commit}^{tree}`) {
				stdout = Buffer.from(`${fixture.c3tAuthority.tree}\n`, "utf8");
			} else if (args[0] === "show" && args[2] === "--format=%P") {
				stdout = Buffer.from(`${fixture.c3tAuthority.parents.join(" ")}\n`, "utf8");
			} else if (args[0] === "diff-tree") {
				stdout = rawC3TDiff;
			} else if (args[0] === "show" && args[2] === "--format=%s") {
				stdout = Buffer.from(`${fixture.c3tAuthority.subject}\n`, "utf8");
			} else if (args[0] === "merge-base") {
				stdout = Buffer.alloc(0);
			} else if (args[0] === "notes" && args[2] === "list") {
				noteListCalls += 1;
				stdout = Buffer.from(`${fixture.c3tAuthority.note_blob_oid}\n`, "utf8");
			} else if (args[0] === "cat-file" && args[1] === "-t") {
				stdout = Buffer.from("blob\n", "utf8");
			} else if (args[0] === "cat-file" && args[1] === "blob") {
				stdout = exactC3TNoteBytes;
			} else {
				throw new Error(`synthetic C3T loader received unexpected Git argv: ${args.join(" ")}`);
			}
			if (!accepted.includes(status)) throw new Error(`synthetic C3T loader status ${status} not accepted`);
			if (noteListCalls > 2) throw new Error("synthetic C3T loader listed the note too many times");
			return { status, stdout };
		};
	};
	const loadedC3TAuthority = loadC3TAuthorityWithRunner(c3tLoaderRunner(), fixture.c3qAuthority.commit);
	const loadedC3TErrors = validateC3TAuthority(loadedC3TAuthority, fixture.c3qAuthority.commit);
	if (loadedC3TErrors.length > 0) {
		throw new Error(`P07B-C C3T direct-child loader baseline failed: ${loadedC3TErrors.join(", ")}`);
	}
	for (const [name, ancestry, expected] of [
		["C3T absent ancestry", Buffer.alloc(0), "absent or not exact LF-framed"],
		["C3T ancestry framing", Buffer.from(`${fixture.c3tAuthority.commit} ${fixture.c3qAuthority.commit}`, "utf8"), "absent or not exact LF-framed"],
		["C3T malformed ancestry", Buffer.from("not-an-object\n", "utf8"), "ancestry row is malformed"],
		["C3T zero direct children", Buffer.from(`${fixture.c3tAuthority.commit} ${"b".repeat(40)}\n`, "utf8"), "direct-child cardinality 0"],
		["C3T multiple direct children", Buffer.from(
			`${fixture.c3tAuthority.commit} ${fixture.c3qAuthority.commit}\n${"b".repeat(40)} ${fixture.c3qAuthority.commit}\n`,
			"utf8",
		), "direct-child cardinality 2"],
	]) {
		let message = "";
		try {
			loadC3TAuthorityWithRunner(c3tLoaderRunner(ancestry), fixture.c3qAuthority.commit);
		} catch (error) {
			message = error.message;
		}
		if (!message.includes(expected)) {
			throw new Error(`P07B-C C3T loader self-test false negative: ${name} (${message || "accepted"})`);
		}
		rejected += 1;
	}
	const exactC3UNoteBytes = Buffer.from(JSON.stringify(fixture.c3uAuthority.note), "utf8");
	const rawC3UDiff = Buffer.from(fixture.c3uAuthority.source_diff.map((row) =>
		`:${row.old_mode} ${row.new_mode} ${row.old_oid} ${row.new_oid} ${row.status}\0${row.path}\0`).join(""), "utf8");
	const c3uLoaderRunner = (ancestry = Buffer.from(
		`${fixture.c3uAuthority.commit} ${fixture.c3tAuthority.commit}\n`,
		"utf8",
	)) => {
		let noteListCalls = 0;
		let ancestryCalls = 0;
		return (args, accepted = [0]) => {
			let status = 0;
			let stdout;
			if (args[0] === "rev-list") {
				ancestryCalls += 1;
				if (ancestryCalls !== 1 || !isDeepStrictEqual(args, [
					"rev-list", "--parents", "--ancestry-path", `${fixture.c3tAuthority.commit}..HEAD`,
				])) {
					throw new Error(`synthetic C3U loader received noncanonical ancestry argv: ${args.join(" ")}`);
				}
				stdout = ancestry;
			} else if (args[0] === "rev-parse" && args[2] === "HEAD^{commit}") {
				stdout = Buffer.from(`${fixture.c3uAuthority.head_commit}\n`, "utf8");
			} else if (args[0] === "rev-parse" && args[2] === `${fixture.c3uAuthority.commit}^{commit}`) {
				stdout = Buffer.from(`${fixture.c3uAuthority.commit}\n`, "utf8");
			} else if (args[0] === "rev-parse" && args[2] === `${fixture.c3uAuthority.commit}^{tree}`) {
				stdout = Buffer.from(`${fixture.c3uAuthority.tree}\n`, "utf8");
			} else if (args[0] === "show" && args[2] === "--format=%P") {
				stdout = Buffer.from(`${fixture.c3uAuthority.parents.join(" ")}\n`, "utf8");
			} else if (args[0] === "diff-tree") {
				stdout = rawC3UDiff;
			} else if (args[0] === "show" && args[2] === "--format=%s") {
				stdout = Buffer.from(`${fixture.c3uAuthority.subject}\n`, "utf8");
			} else if (args[0] === "merge-base") {
				stdout = Buffer.alloc(0);
			} else if (args[0] === "notes" && args[2] === "list") {
				noteListCalls += 1;
				stdout = Buffer.from(`${fixture.c3uAuthority.note_blob_oid}\n`, "utf8");
			} else if (args[0] === "cat-file" && args[1] === "-t") {
				stdout = Buffer.from("blob\n", "utf8");
			} else if (args[0] === "cat-file" && args[1] === "blob") {
				stdout = exactC3UNoteBytes;
			} else {
				throw new Error(`synthetic C3U loader received unexpected Git argv: ${args.join(" ")}`);
			}
			if (!accepted.includes(status)) throw new Error(`synthetic C3U loader status ${status} not accepted`);
			if (noteListCalls > 2) throw new Error("synthetic C3U loader listed the note too many times");
			return { status, stdout };
		};
	};
	const loadedC3UAuthority = loadC3UAuthorityWithRunner(c3uLoaderRunner(), fixture.c3tAuthority.commit);
	const loadedC3UErrors = validateC3UAuthority(loadedC3UAuthority, fixture.c3tAuthority.commit);
	if (loadedC3UErrors.length > 0) {
		throw new Error(`P07B-C C3U direct-child loader baseline failed: ${loadedC3UErrors.join(", ")}`);
	}
	for (const [name, ancestry, expected] of [
		["C3U absent ancestry", Buffer.alloc(0), "absent or not exact LF-framed"],
		["C3U ancestry framing", Buffer.from(`${fixture.c3uAuthority.commit} ${fixture.c3tAuthority.commit}`, "utf8"), "absent or not exact LF-framed"],
		["C3U ancestry CR framing", Buffer.from(`${fixture.c3uAuthority.commit} ${fixture.c3tAuthority.commit}\r\n`, "utf8"), "absent or not exact LF-framed"],
		["C3U malformed ancestry", Buffer.from("not-an-object\n", "utf8"), "ancestry row is malformed"],
		["C3U zero direct children", Buffer.from(`${fixture.c3uAuthority.commit} ${"c".repeat(40)}\n`, "utf8"), "direct-child cardinality 0"],
		["C3U multiple direct children", Buffer.from(
			`${fixture.c3uAuthority.commit} ${fixture.c3tAuthority.commit}\n${"c".repeat(40)} ${fixture.c3tAuthority.commit}\n`,
			"utf8",
		), "direct-child cardinality 2"],
	]) {
		let message = "";
		try {
			loadC3UAuthorityWithRunner(c3uLoaderRunner(ancestry), fixture.c3tAuthority.commit);
		} catch (error) {
			message = error.message;
		}
		if (!message.includes(expected)) {
			throw new Error(`P07B-C C3U loader self-test false negative: ${name} (${message || "accepted"})`);
		}
		rejected += 1;
	}
	const exactNoteBytes = Buffer.from(JSON.stringify(fixture.authority.note), "utf8");
	const loaderReceipt = structuredClone(fixture.receipt);
	loaderReceipt.evidence.git_note.body_sha256 = createHash("sha256").update(exactNoteBytes).digest("hex");
	const rawDiff = Buffer.from(fixture.authority.source_diff.map((row) =>
		`:${row.old_mode} ${row.new_mode} ${row.old_oid} ${row.new_oid} ${row.status}\0${row.path}\0`).join(""), "utf8");
	const receiptLoaderRunner = (options = {}) => {
		let noteListCalls = 0;
		return (args, accepted = [0]) => {
			let status = 0;
			let stdout;
			if (args[0] === "rev-parse" && args[2] === "HEAD^{commit}") {
				stdout = Buffer.from(`${fixture.c3uAuthority.commit}\n`, "utf8");
			} else if (args[0] === "rev-parse" && args[2] === `${loaderReceipt.source_commit}^{commit}`) {
				stdout = Buffer.from(`${fixture.authority.commit}\n`, "utf8");
			} else if (args[0] === "rev-parse" && args[2] === `${loaderReceipt.source_commit}^{tree}`) {
				stdout = Buffer.from(`${fixture.authority.tree}\n`, "utf8");
			} else if (args[0] === "show" && args[2] === "--format=%P") {
				stdout = Buffer.from(`${fixture.authority.parents.join(" ")}\n`, "utf8");
			} else if (args[0] === "diff-tree") {
				stdout = rawDiff;
			} else if (args[0] === "show" && args[2] === "--format=%s") {
				stdout = Buffer.from(`${fixture.authority.subject}\n`, "utf8");
			} else if (args[0] === "merge-base") {
				stdout = Buffer.alloc(0);
			} else if (args[0] === "notes" && args[2] === "list") {
				noteListCalls += 1;
				if (noteListCalls === 1 && options.firstNoteListing !== undefined) {
					stdout = options.firstNoteListing;
				} else {
					const oid = noteListCalls === 2 && options.secondNoteOID !== undefined
						? options.secondNoteOID : loaderReceipt.evidence.git_note.blob_oid;
					stdout = Buffer.from(`${oid}\n`, "utf8");
				}
			} else if (args[0] === "cat-file" && args[1] === "-t") {
				stdout = Buffer.from(`${options.noteType ?? "blob"}\n`, "utf8");
			} else if (args[0] === "cat-file" && args[1] === "blob") {
				stdout = options.noteBytes ?? exactNoteBytes;
			} else {
				throw new Error(`synthetic receipt loader received unexpected Git argv: ${args.join(" ")}`);
			}
			if (!accepted.includes(status)) throw new Error(`synthetic receipt loader status ${status} not accepted`);
			return { status, stdout };
		};
	};
	const loadedAuthority = loadReceiptAuthorityWithRunner(loaderReceipt, receiptLoaderRunner());
	if (loadedAuthority.note_blob_oid !== loaderReceipt.evidence.git_note.blob_oid ||
		loadedAuthority.note_body_sha256 !== loaderReceipt.evidence.git_note.body_sha256 ||
		!isDeepStrictEqual(loadedAuthority.note, fixture.authority.note)) {
		throw new Error("P07B-C C3 receipt loader exact-blob baseline mismatch");
	}
	for (const [name, options, expected] of [
		["note ref changed during read", { secondNoteOID: "e".repeat(40) }, "note reference changed"],
		["note listing framing", { firstNoteListing: Buffer.from(`${loaderReceipt.evidence.git_note.blob_oid}\n\n`, "utf8") }, "exact LF-terminated line"],
		["note body fatal UTF-8", { noteBytes: Buffer.from([0xff]) }, "not valid UTF-8"],
		["note object type", { noteType: "tree" }, "not blob"],
	]) {
		let rejectedLoader = false;
		try {
			loadReceiptAuthorityWithRunner(loaderReceipt, receiptLoaderRunner(options));
		} catch (error) {
			rejectedLoader = error.message.includes(expected);
		}
		if (!rejectedLoader) throw new Error(`P07B-C C3 receipt loader self-test false negative: ${name}`);
		rejected += 1;
	}
	for (const [name, mutate, expected] of [
		["schema", (value) => { value.schema_version = "countershape/p07b-c-c3-receipt/v2"; }, "schema version"],
		["sealed source commit", (value) => { value.source_commit = "a".repeat(40); }, "sealed C3 source commit identity"],
		["sealed source tree", (value) => { value.source_tree = "b".repeat(40); }, "sealed C3 source tree identity"],
		["subject", (value) => { value.source_subject += " altered"; }, "source subject"],
		["sealed override", (value) => { value.secrets_override = false; }, "sealed C3 secrets override disclosure"],
		["claim order", (value) => { [value.claims[0], value.claims[1]] = [value.claims[1], value.claims[0]]; }, "claim 1"],
		["claim grade", (value) => { value.claims[0].grade = "UNRECEIPTED"; }, "claim 1"],
		["coverage", (value) => { value.coverage_complete_events -= 1; }, "coverage counts"],
		["HTML", (value) => { value.evidence.html_report.path = "evidence/other.html"; }, "local HTML declaration"],
		["ledger count", (value) => { value.evidence.ledger_archive.sealed_event_count -= 1; }, "local ledger declaration"],
		["sealed note", (value) => { value.evidence.git_note.blob_oid = "c".repeat(40); }, "sealed C3 Git-note identity"],
		["scan provenance", (value) => { value.evidence.seal_redaction.structured_scan_provenance = "UNBOUND"; }, "seal redaction"],
	]) {
		const candidate = structuredClone(fixture.receipt);
		mutate(candidate);
		requireError(name, validateC3ReceiptDeclaration(candidate), expected);
	}
	for (const [name, mutate, expected] of [
		["commit", (value) => { value.commit = "e".repeat(40); }, "source commit"],
		["tree", (value) => { value.tree = "e".repeat(40); }, "source tree"],
		["parent", (value) => { value.parent = "e".repeat(40); value.parents = [value.parent]; }, "source parent"],
		["merge parent", (value) => { value.parents.push("e".repeat(40)); }, "source parents"],
		["diff roster", (value) => { value.source_diff.pop(); }, "source diff exact path roster"],
		["diff partition", (value) => { value.source_diff.find((row) => row.status === "A").status = "M"; }, "source diff added row"],
		["subject", (value) => { value.subject += " altered"; }, "source commit subject"],
		["subject trailing space", (value) => {
			value.subject = decodeExactGitLine(Buffer.from(`${c3SourceSubject} \n`, "utf8"), "synthetic C3 subject");
		}, "source commit subject"],
		["ancestor", (value) => { value.ancestor_of_head = false; }, "ancestor"],
		["note type", (value) => { value.note_type = "tree"; }, "note object type"],
		["note blob", (value) => { value.note_blob_oid = "e".repeat(40); }, "note object identity"],
		["note body", (value) => { value.note_body_sha256 = "e".repeat(64); }, "note body digest"],
		["note claim order", (value) => { [value.note.claims[0], value.note.claims[1]] = [value.note.claims[1], value.note.claims[0]]; }, "note claim 1"],
		["note argv", (value) => { value.note.claims[0].claim.argv_preview.at(-1); value.note.claims[0].claim.argv_preview[value.note.claims[0].claim.argv_preview.length - 1] += "-altered"; }, "note claim 1 argv"],
		["note coverage", (value) => { value.note.coverage.by_coverage.complete -= 1; }, "event coverage"],
	]) {
		const candidate = structuredClone(fixture.authority);
		mutate(candidate);
		requireError(name, validateC3ReceiptAuthority(fixture.receipt, candidate), expected);
	}
	for (const [name, mutate, expected] of [
		["C3R commit", (value) => { value.commit = "0".repeat(39) + "1"; }, "sealed C3R commit identity"],
		["C3R tree", (value) => { value.tree = "0".repeat(39) + "2"; }, "sealed C3R tree identity"],
		["C3R parent", (value) => { value.parent = "0".repeat(40); value.parents = [value.parent]; }, "source parent"],
		["C3R merge parent", (value) => { value.parents.push("0".repeat(40)); }, "source parent"],
		["C3R diff roster", (value) => { value.source_diff.pop(); }, "source diff exact path roster"],
		["C3R diff partition", (value) => { value.source_diff.find((row) => row.status === "A").status = "M"; }, "source diff added row"],
		["C3R subject", (value) => { value.subject += " altered"; }, "source commit subject"],
		["C3R ancestry", (value) => { value.ancestor_of_head = false; }, "ancestor"],
		["C3R note type", (value) => { value.note_type = "tree"; }, "note object type"],
		["C3R note blob", (value) => { value.note_blob_oid = "0".repeat(39) + "3"; }, "sealed C3R note object identity"],
		["C3R note body", (value) => { value.note_body_sha256 = "0".repeat(63) + "4"; }, "sealed C3R note body digest"],
		["C3R note override type", (value) => { value.note.secrets_override = "false"; }, "secrets override disclosure"],
		["C3R note order", (value) => { [value.note.claims[0], value.note.claims[1]] = [value.note.claims[1], value.note.claims[0]]; }, "didrun note claim 1"],
		["C3R note argv", (value) => { value.note.claims[2].claim.argv_preview.push("--extra"); }, "didrun note claim 3 argv"],
		["C3R coverage", (value) => { value.note.coverage.by_coverage.complete -= 1; }, "event coverage"],
	]) {
		const candidate = structuredClone(fixture.c3rAuthority);
		mutate(candidate);
		requireError(name, validateSealedC3RAuthority(candidate), expected);
	}
	for (const [name, mutate, expected] of [
		["C3Q malformed commit", (value) => { value.commit = "z".repeat(40); }, "source commit"],
		["C3Q malformed tree", (value) => { value.tree = "z".repeat(40); }, "source tree"],
		["C3Q commit", (value) => { value.commit = "9".repeat(40); }, "didrun note commit"],
		["C3Q tree", (value) => { value.tree = "9".repeat(40); }, "didrun note tree"],
		["C3Q parent", (value) => { value.parent = "9".repeat(40); value.parents = [value.parent]; }, "source parent"],
		["C3Q merge parent", (value) => { value.parents.push("9".repeat(40)); }, "source parent"],
		["C3Q diff roster", (value) => { value.source_diff.pop(); }, "source diff exact path roster"],
		["C3Q diff partition", (value) => { value.source_diff.find((row) => row.status === "A").status = "M"; }, "source diff added row"],
		["C3Q subject", (value) => { value.subject += " altered"; }, "source commit subject"],
		["C3Q ancestry", (value) => { value.ancestor_of_head = false; }, "ancestor"],
		["C3Q note type", (value) => { value.note_type = "tree"; }, "note object type"],
		["C3Q note blob shape", (value) => { value.note_blob_oid = "z".repeat(40); }, "note object identity"],
		["C3Q note body shape", (value) => { value.note_body_sha256 = "z".repeat(64); }, "note body digest"],
		["C3Q note root", (value) => { value.note.extra = true; }, "missing or malformed parsed didrun Git note"],
		["C3Q note version", (value) => { value.note.version = 2; }, "didrun note version"],
		["C3Q note claim count", (value) => { value.note.claims.pop(); }, "didrun note claim roster length"],
		["C3Q note override type", (value) => { value.note.secrets_override = "false"; }, "secrets override disclosure"],
		["C3Q note order", (value) => { [value.note.claims[0], value.note.claims[1]] = [value.note.claims[1], value.note.claims[0]]; }, "didrun note claim 1"],
		["C3Q note claim type", (value) => { value.note.claims[0].claim.ctype = "command-succeeded"; }, "didrun note claim 1"],
		["C3Q note claim index", (value) => { value.note.claims[0].claim.declared_at_index = 1; }, "didrun note claim 1"],
		["C3Q note claim events", (value) => { value.note.claims[0].claim.event_indices = [1]; }, "didrun note claim 1"],
		["C3Q note claim pathspec", (value) => { value.note.claims[0].claim.pathspecs = ["."]; }, "didrun note claim 1"],
		["C3Q note claim grade", (value) => { value.note.claims[0].grade = "scope-exact"; }, "didrun note claim 1"],
		["C3Q note claim exit", (value) => { value.note.claims[0].exit_code = 1; }, "didrun note claim 1"],
		["C3Q note claim reason", (value) => { value.note.claims[0].reason = "altered"; }, "didrun note claim 1"],
		["C3Q note claim delta", (value) => { value.note.claims[0].delta = ["dirty"]; }, "didrun note claim 1"],
		["C3Q note argv", (value) => { value.note.claims[2].claim.argv_preview.push("--extra"); }, "didrun note claim 3 argv"],
		["C3Q coverage", (value) => { value.note.coverage.by_coverage.complete -= 1; }, "event coverage"],
		["C3Q coverage total", (value) => { value.note.coverage.total_events -= 1; }, "event coverage"],
		["C3Q coverage extra key", (value) => { value.note.coverage.by_coverage.partial = 1; }, "event coverage"],
	]) {
		const candidate = structuredClone(fixture.c3qAuthority);
		mutate(candidate);
		requireError(name, validateSealedC3QAuthority(candidate), expected);
	}
	for (const [name, mutate, expected] of [
		["C3T malformed commit", (value) => { value.commit = "z".repeat(40); }, "source commit"],
		["C3T malformed tree", (value) => { value.tree = "z".repeat(40); }, "source tree"],
		["C3T parent", (value) => { value.parent = "b".repeat(40); value.parents = [value.parent]; }, "source parent"],
		["C3T merge parent", (value) => { value.parents.push("b".repeat(40)); }, "source parent"],
		["C3T diff roster", (value) => { value.source_diff.pop(); }, "source diff exact path roster"],
		["C3T diff partition", (value) => { value.source_diff.find((row) => row.status === "A").status = "M"; }, "source diff added row"],
		["C3T subject", (value) => { value.subject += " altered"; }, "source commit subject"],
		["C3T ancestry", (value) => { value.ancestor_of_head = false; }, "ancestor"],
		["C3T current HEAD", (value) => { value.head_commit = "b".repeat(40); }, "sealed C3T is not current HEAD"],
		["C3T note type", (value) => { value.note_type = "tree"; }, "note object type"],
		["C3T note root", (value) => { value.note.extra = true; }, "missing or malformed parsed didrun Git note"],
		["C3T note claim order", (value) => { [value.note.claims[0], value.note.claims[1]] = [value.note.claims[1], value.note.claims[0]]; }, "didrun note claim 1"],
		["C3T note argv", (value) => { value.note.claims[2].claim.argv_preview.push("--extra"); }, "didrun note claim 3 argv"],
		["C3T coverage", (value) => { value.note.coverage.by_coverage.complete -= 1; }, "event coverage"],
		["C3T override type", (value) => { value.note.secrets_override = "false"; }, "secrets override disclosure"],
	]) {
		const candidate = structuredClone(c3uPresealC3T);
		mutate(candidate);
		requireError(name, validateC3UPredecessorAuthority(candidate), expected);
	}
	for (const [name, mutate, expected] of [
		["sealed C3T coherent commit substitution", (value) => {
			value.commit = "c".repeat(40); value.note.commit = value.commit;
		}, "sealed C3T commit identity"],
		["sealed C3T coherent tree substitution", (value) => {
			value.tree = "d".repeat(40); value.note.tree = value.tree;
		}, "sealed C3T tree identity"],
		["sealed C3T valid alternate note blob", (value) => { value.note_blob_oid = "e".repeat(40); }, "sealed C3T note object identity"],
		["sealed C3T valid alternate note body", (value) => { value.note_body_sha256 = "f".repeat(64); }, "sealed C3T note body digest"],
		["sealed C3T alternate seal outcome", (value) => { value.note.secrets_override = false; }, "sealed C3T secrets override disclosure"],
	]) {
		const candidate = structuredClone(fixture.c3tAuthority);
		mutate(candidate);
		requireError(name, validateSealedC3TAuthority(candidate), expected);
	}
	for (const [name, mutate, expected] of [
		["C3U malformed commit", (value) => { value.commit = "z".repeat(40); }, "source commit"],
		["C3U malformed tree", (value) => { value.tree = "z".repeat(40); }, "source tree"],
		["C3U parent", (value) => { value.parent = "b".repeat(40); value.parents = [value.parent]; }, "source parent"],
		["C3U merge parent", (value) => { value.parents.push("b".repeat(40)); }, "source parent"],
		["C3U diff roster", (value) => { value.source_diff.pop(); }, "source diff exact path roster"],
		["C3U diff partition", (value) => { value.source_diff.find((row) => row.status === "A").status = "M"; }, "source diff added row"],
		["C3U subject", (value) => { value.subject += " altered"; }, "source commit subject"],
		["C3U ancestry", (value) => { value.ancestor_of_head = false; }, "ancestor"],
		["C3U current HEAD", (value) => { value.head_commit = "b".repeat(40); }, "C3U is not current HEAD"],
		["C3U note type", (value) => { value.note_type = "tree"; }, "note object type"],
		["C3U note root", (value) => { value.note.extra = true; }, "missing or malformed parsed didrun Git note"],
		["C3U note claim order", (value) => { [value.note.claims[0], value.note.claims[1]] = [value.note.claims[1], value.note.claims[0]]; }, "didrun note claim 1"],
		["C3U note argv", (value) => { value.note.claims[2].claim.argv_preview.push("--extra"); }, "didrun note claim 3 argv"],
		["C3U coverage", (value) => { value.note.coverage.by_coverage.complete -= 1; }, "event coverage"],
		["C3U override type", (value) => { value.note.secrets_override = "false"; }, "secrets override disclosure"],
	]) {
		const candidate = structuredClone(c3bPresealC3U);
		mutate(candidate);
		requireError(name, validateC3BPredecessorAuthority(candidate, fixture.c3tAuthority.commit), expected);
	}
	for (const [name, mutate, expected] of [
		["sealed C3Q coherent commit substitution", (value) => {
			value.commit = "c".repeat(40); value.note.commit = value.commit;
		}, "sealed C3Q commit identity"],
		["sealed C3Q coherent tree substitution", (value) => {
			value.tree = "d".repeat(40); value.note.tree = value.tree;
		}, "sealed C3Q tree identity"],
		["sealed C3Q valid alternate note blob", (value) => { value.note_blob_oid = "e".repeat(40); }, "sealed C3Q note object identity"],
		["sealed C3Q valid alternate note body", (value) => { value.note_body_sha256 = "f".repeat(64); }, "sealed C3Q note body digest"],
		["sealed C3Q alternate seal outcome", (value) => { value.note.secrets_override = false; }, "sealed C3Q secrets override disclosure"],
	]) {
		const candidate = structuredClone(fixture.c3qAuthority);
		mutate(candidate);
		requireError(name, validateSealedC3QAuthority(candidate), expected);
	}
	let malformedSubjectFramingRejected = false;
	try { decodeExactGitLine(Buffer.from(`${c3SourceSubject}\n\n`, "utf8"), "synthetic C3 subject"); } catch {
		malformedSubjectFramingRejected = true;
	}
	if (!malformedSubjectFramingRejected) throw new Error("P07B-C C3 exact Git subject framing self-test false negative");
	const absent = await syntheticC3AbsentPlanFixture(fixture);
	for (const [name, path, mutate, expected] of [
		["pending status", c3StatusPath, (body) => body.replace(c3PendingStatusState, "- **State:** prematurely sealed"), "pending source state"],
		["pending grade", c3StatusPath, (body) => body.replace("| `UNRECEIPTED` |", "| `TREE-EXACT` |"), "pending source receipt map"],
		["pending status HTML-commented", c3StatusPath, (body) => `<!--\n${body}-->\n`, "pending source state"],
		["pending status fenced", c3StatusPath, (body) => `\`\`\`markdown\n${body}\`\`\`\n`, "pending source state"],
		["pending status indented code", c3StatusPath, (body) => body.split("\n").map((line) => `    ${line}`).join("\n"), "pending source state"],
		["premature marker", "docs/HANDOFF_MODE_C.md", (body) => `${body}\n${c3ReceiptBlockStart}\n`, "pending receipt marker"],
		["C3F downgrade", "docs/HANDOFF_MODE_C.md", (body) => c3pbHandoffTransitions.reduce((currentBody, transition) => {
			const metadata = c3pbMetadataLineContracts[transition.name];
			const before = metadata === undefined ? transition.c3u : metadata.render(transition.c3u, "c3u");
			const after = metadata === undefined ? transition.c3f : metadata.render(transition.c3f, "c3f");
			return replaceFixtureExactlyOnce(currentBody, before, after, `C3 absent downgrade ${transition.name}`);
		}, body), "must be C3U-active"],
	]) {
		const overrides = new Map(absent);
		const body = await readText(repositoryRoot, path, overrides);
		overrides.set(path, mutate(body));
		requireError(name, await checkPlan(repositoryRoot, overrides), expected);
	}
	const predecessorManifest = expectedC3PredecessorManifest();
	predecessorManifest.source_parent.note_type = "tree";
	const manifestOverrides = new Map(absent);
	manifestOverrides.set(c3PredecessorManifestPath, `${JSON.stringify(predecessorManifest)}\n`);
	requireError(
		"exact predecessor manifest",
		await checkPlan(repositoryRoot, manifestOverrides),
		"exact C3S/C3F/C3L/C3A authority manifest mismatch",
	);
	const exactPredecessorText = expectedC3PredecessorManifestBytes().toString("utf8");
	for (const [name, duplicate] of [
		["duplicate predecessor root key", exactPredecessorText.replace(
			'  "unit": "C3",', '  "unit": "NOT-C3",\n  "unit": "C3",',
		)],
		["duplicate predecessor nested key", exactPredecessorText.replace(
			'        "grade": "TREE-EXACT"', '        "grade": "UNRECEIPTED",\n        "grade": "TREE-EXACT"',
		)],
	]) {
		const duplicateOverrides = new Map(absent);
		duplicateOverrides.set(c3PredecessorManifestPath, duplicate);
		requireError(
			name,
			await checkPlan(repositoryRoot, duplicateOverrides),
			"exact C3S/C3F/C3L/C3A authority manifest mismatch",
		);
	}
	let subsectionDecoyControls = 0;
	for (const [name, marker, inertDecoy] of [
		["HTML-commented subsection heading", "C3Q-INERT-HTML-SUBSECTION-DECOY", "<!--\n### C3Q-INERT-HTML-SUBSECTION-DECOY\n-->"],
		["fenced subsection heading", "C3Q-INERT-FENCE-SUBSECTION-DECOY", "```text\n### C3Q-INERT-FENCE-SUBSECTION-DECOY\n```"],
	]) {
		const liveHandoff = absent.get("docs/HANDOFF_MODE_C.md");
		const heading = "### Active C3U cross-phase receipt-fixture maintenance";
		const liveSection = currentStateSubsection(liveHandoff, heading);
		const decoySection = replaceFixtureExactlyOnce(
			liveSection,
			`${heading}\n\n`,
			`${heading}\n\n${inertDecoy}\n\n`,
			name,
		);
		const decoyCurrent = new Map(absent);
		decoyCurrent.set(
			"docs/HANDOFF_MODE_C.md",
			replaceCurrentStateSubsection(liveHandoff, heading, decoySection),
		);
		const transitioned = await syntheticC3ReceiptPlanFixture(fixture, decoyCurrent);
		if (transitioned.get("docs/HANDOFF_MODE_C.md").includes(marker)) {
			throw new Error(`P07B-C C3 receipt checker retained ${name}`);
		}
		const transitionedErrors = await checkPlan(
			repositoryRoot,
			transitioned,
			undefined,
			undefined,
			undefined,
			undefined,
			fixture.authority,
			fixture.c3rAuthority,
			fixture.c3qAuthority,
			fixture.c3tAuthority,
			fixture.c3uAuthority,
		);
		if (transitionedErrors.length > 0) {
			throw new Error(`P07B-C C3 receipt checker ${name} transition failed:\n${transitionedErrors.join("\n")}`);
		}
		subsectionDecoyControls += 1;
	}
	const present = await syntheticC3ReceiptPlanFixture(fixture);
	const presentErrors = await checkPlan(
		repositoryRoot,
		present,
		undefined,
		undefined,
		undefined,
		undefined,
		fixture.authority,
		fixture.c3rAuthority,
		fixture.c3qAuthority,
		fixture.c3tAuthority,
		fixture.c3uAuthority,
	);
	if (presentErrors.length > 0) throw new Error(`P07B-C C3 receipt-present baseline failed:\n${presentErrors.join("\n")}`);
	const presentHandoff = present.get("docs/HANDOFF_MODE_C.md");
	const terminalC3 = parseC3ReceiptBlock(presentHandoff);
	if (terminalC3.block === undefined || presentHandoff !== `${terminalC3.stripped}${terminalC3.block}\n`) {
		throw new Error("P07B-C C3U helper baseline requires one exact terminal C3 block");
	}
	const legacyProbe = "<!-- P07B-C-C3U-LEGACY-PROBE:START -->\nlegacy\n<!-- P07B-C-C3U-LEGACY-PROBE:END -->";
	if (insertLegacyReceiptBlockBeforeOptionalTerminalC3("base\n", legacyProbe, "C3U direct marker-free control", "direct") !==
		`base\n${legacyProbe}\n` ||
		insertLegacyReceiptBlockBeforeOptionalTerminalC3("base  \n\n", legacyProbe, "C3U trimmed marker-free control", "trimmed-blank") !==
		`base\n\n${legacyProbe}\n`) {
		throw new Error("P07B-C C3U marker-free compatibility control failed");
	}
	for (const mode of ["direct", "trimmed-blank"]) {
		const layered = insertLegacyReceiptBlockBeforeOptionalTerminalC3(
			presentHandoff, legacyProbe, `C3U terminal preservation ${mode}`, mode,
		);
		const reparsed = parseC3ReceiptBlock(layered);
		if (reparsed.block !== terminalC3.block || layered !== `${reparsed.stripped}${terminalC3.block}\n` ||
			reparsed.stripped !== `${terminalC3.stripped}${legacyProbe}\n`) {
			throw new Error(`P07B-C C3U ${mode} terminal preservation control failed`);
		}
	}
	const requireHelperError = (name, body, mode, expected) => {
		let message = "";
		try {
			insertLegacyReceiptBlockBeforeOptionalTerminalC3(body, legacyProbe, name, mode);
		} catch (error) {
			message = error.message;
		}
		if (!message.includes(expected)) {
			throw new Error(`P07B-C C3U helper self-test false negative: ${name} (${message || "accepted"})`);
		}
		rejected += 1;
	};
	for (const [name, body, mode, expected] of [
		["unknown mode", presentHandoff, "other", "unknown marker-free render mode"],
		["missing end", `${terminalC3.stripped}${c3ReceiptBlockStart}\n`, "direct", "marker cardinality"],
		["duplicate terminal block", `${presentHandoff}${terminalC3.block}\n`, "direct", "marker cardinality"],
		["reversed markers", `${terminalC3.stripped}${c3ReceiptBlockEnd}\n${c3ReceiptBlockStart}\n`, "direct", "marker order"],
		["inline start marker", presentHandoff.replace(c3ReceiptBlockStart, `inline ${c3ReceiptBlockStart}`), "direct", "standalone LF lines"],
		["nonterminal block", `${presentHandoff}trailing\n`, "direct", "must already be exact and terminal"],
		["non-LF terminal", `${presentHandoff.slice(0, -1)}\r\n`, "direct", "standalone LF lines"],
		["hidden terminal block", `${terminalC3.stripped}<!--\n${terminalC3.block}\n-->\n`, "direct", "not visible Markdown authority"],
		["fenced terminal block", `${terminalC3.stripped}~~~markdown\n${terminalC3.block}\n~~~\n`, "direct", "not visible Markdown authority"],
	]) {
		requireHelperError(name, body, mode, expected);
	}
	const c3pProbeFixture = syntheticC3PReceiptFixture();
	const c3pProbeBlock = c3pReceiptHandoffBlock(c3pProbeFixture.receipt);
	const c3OnlyHandoff = `base\n${terminalC3.block}\n`;
	const c3pLayeredHandoff = insertLegacyReceiptBlockBeforeOptionalTerminalC3(
		c3OnlyHandoff, c3pProbeBlock, "C3P-over-C3 terminal control", "direct",
	);
	if (parseC3PReceiptBlock(c3pLayeredHandoff).stripped !== c3OnlyHandoff ||
		parseC3ReceiptBlock(c3pLayeredHandoff).block !== terminalC3.block) {
		throw new Error("P07B-C C3P-over-C3 inverse or terminal preservation control failed");
	}
	const c3pOverC3Lifecycle = Object.freeze({
		field: "c3b",
		phase: "C3B_ACTIVE",
		receipt: fixture.receipt,
		c3rAuthority: fixture.c3rAuthority,
		c3qAuthority: fixture.c3qAuthority,
		c3tAuthority: fixture.c3tAuthority,
		c3uAuthority: fixture.c3uAuthority,
	});
	const c3pOverC3 = await syntheticC3PReceiptPlanFixture(c3pProbeFixture, present, c3pOverC3Lifecycle);
	const c3pOverC3Handoff = c3pOverC3.overrides.get("docs/HANDOFF_MODE_C.md");
	if (c3pOverC3.priorHandoff !== withoutC3PReceiptBlock(presentHandoff) ||
		withoutC3PReceiptBlock(c3pOverC3Handoff) !== c3pOverC3.priorHandoff ||
		parseC3ReceiptBlock(c3pOverC3Handoff).block !== terminalC3.block) {
		throw new Error("P07B-C C3P-over-C3 actual fixture inverse or terminal preservation control failed");
	}
	const c3pOverC3Errors = await checkPlan(
		repositoryRoot,
		c3pOverC3.overrides,
		undefined,
		undefined,
		undefined,
		c3pOverC3.authority,
		fixture.authority,
		fixture.c3rAuthority,
		fixture.c3qAuthority,
		fixture.c3tAuthority,
		fixture.c3uAuthority,
	);
	if (c3pOverC3Errors.length > 0) {
		throw new Error(`P07B-C C3P-over-C3 actual fixture full-plan control failed:\n${c3pOverC3Errors.join("\n")}`);
	}
	const c2ProbeFixture = syntheticC2ReceiptFixture();
	const c2OverC3 = await syntheticC2ReceiptPlanFixture(c2ProbeFixture, present);
	const c2OverC3Handoff = c2OverC3.overrides.get("docs/HANDOFF_MODE_C.md");
	if (c2OverC3.priorHandoff !== withoutC2ReceiptBlock(presentHandoff) ||
		withoutC2ReceiptBlock(c2OverC3Handoff) !== c2OverC3.priorHandoff ||
		parseC3ReceiptBlock(c2OverC3Handoff).block !== terminalC3.block) {
		throw new Error("P07B-C C2-over-C3 inverse or terminal preservation control failed");
	}
	const c2OverC3Errors = await checkPlan(
		repositoryRoot,
		c2OverC3.overrides,
		undefined,
		undefined,
		c2OverC3.authority,
		undefined,
		fixture.authority,
		fixture.c3rAuthority,
		fixture.c3qAuthority,
		fixture.c3tAuthority,
		fixture.c3uAuthority,
	);
	if (c2OverC3Errors.length > 0) {
		throw new Error(`P07B-C C2-over-C3 full-plan control failed:\n${c2OverC3Errors.join("\n")}`);
	}
	if (liveControlledPaths !== undefined) {
		for (const path of [c3StatusPath, "docs/HANDOFF_MODE_C.md", c3ReceiptDeclarationPath]) {
			if (present.get(path) !== liveControlledPaths.get(path)) {
				throw new Error(`P07B-C C3 live present round trip changed canonical ${path}`);
			}
		}
	}
	const sealedDescendantC3U = structuredClone(fixture.c3uAuthority);
	sealedDescendantC3U.head_commit = "d".repeat(40);
	const sealedDescendantErrors = await checkPlan(
		repositoryRoot,
		present,
		undefined,
		undefined,
		undefined,
		undefined,
		fixture.authority,
		fixture.c3rAuthority,
		fixture.c3qAuthority,
		fixture.c3tAuthority,
		sealedDescendantC3U,
	);
	if (sealedDescendantErrors.length > 0) {
		throw new Error(`P07B-C C3 sealed-descendant-head plan baseline failed:\n${sealedDescendantErrors.join("\n")}`);
	}
	const inverse = await syntheticC3AbsentPlanFixture(fixture, present);
	for (const path of [c3StatusPath, "docs/HANDOFF_MODE_C.md", c3ReceiptDeclarationPath]) {
		if (inverse.get(path) !== absent.get(path)) {
			throw new Error(`P07B-C C3 present-to-absent inverse changed canonical ${path}`);
		}
	}
	const rerun = await syntheticC3ReceiptPlanFixture(fixture, present);
	for (const path of [c3StatusPath, "docs/HANDOFF_MODE_C.md", c3ReceiptDeclarationPath]) {
		if (rerun.get(path) !== present.get(path)) throw new Error(`P07B-C C3 receipt rerun changed ${path}`);
	}
	for (const [name, mutate] of [
		["C3Q note blob handoff binding", (authority) => { authority.note_blob_oid = "d".repeat(40); }],
		["C3Q note body handoff binding", (authority) => { authority.note_body_sha256 = "e".repeat(64); }],
		["C3Q seal outcome handoff binding", (authority) => { authority.note.secrets_override = !authority.note.secrets_override; }],
	]) {
		const hostileC3QAuthority = structuredClone(fixture.c3qAuthority);
		mutate(hostileC3QAuthority);
		const handoffPath = "docs/HANDOFF_MODE_C.md";
		const heading = "### Active C3B source receipt reconciliation";
		const liveHandoff = present.get(handoffPath);
		const liveSection = currentStateSubsection(liveHandoff, heading);
		const hostileSection = replaceFixtureExactlyOnce(
			liveSection,
			c3qSealOutcomeDisclosure(fixture.c3qAuthority),
			c3qSealOutcomeDisclosure(hostileC3QAuthority),
			name,
		);
		const hostilePresent = new Map(present);
		hostilePresent.set(handoffPath, replaceCurrentStateSubsection(liveHandoff, heading, hostileSection));
		requireError(name, await checkPlan(
			repositoryRoot,
			hostilePresent,
			undefined,
			undefined,
			undefined,
			undefined,
			fixture.authority,
			fixture.c3rAuthority,
			fixture.c3qAuthority,
			fixture.c3tAuthority,
			fixture.c3uAuthority,
		), "C3B active maintenance section payload mismatch");
	}
	for (const [name, mutate] of [
		["C3T note blob handoff binding", (authority) => { authority.note_blob_oid = "c".repeat(40); }],
		["C3T note body handoff binding", (authority) => { authority.note_body_sha256 = "d".repeat(64); }],
		["C3T seal outcome handoff binding", (authority) => { authority.note.secrets_override = !authority.note.secrets_override; }],
	]) {
		const hostileC3TAuthority = structuredClone(fixture.c3tAuthority);
		mutate(hostileC3TAuthority);
		const handoffPath = "docs/HANDOFF_MODE_C.md";
		const heading = "### Active C3B source receipt reconciliation";
		const liveHandoff = present.get(handoffPath);
		const liveSection = currentStateSubsection(liveHandoff, heading);
		const hostileSection = replaceFixtureExactlyOnce(
			liveSection,
			c3tSealOutcomeDisclosure(fixture.c3tAuthority),
			c3tSealOutcomeDisclosure(hostileC3TAuthority),
			name,
		);
		const hostilePresent = new Map(present);
		hostilePresent.set(handoffPath, replaceCurrentStateSubsection(liveHandoff, heading, hostileSection));
		requireError(name, await checkPlan(
			repositoryRoot,
			hostilePresent,
			undefined,
			undefined,
			undefined,
			undefined,
			fixture.authority,
			fixture.c3rAuthority,
			fixture.c3qAuthority,
			fixture.c3tAuthority,
			fixture.c3uAuthority,
		), "C3B active maintenance section payload mismatch");
	}
	for (const [name, mutate] of [
		["C3U note blob handoff binding", (authority) => { authority.note_blob_oid = "a".repeat(40); }],
		["C3U note body handoff binding", (authority) => { authority.note_body_sha256 = "b".repeat(64); }],
		["C3U seal outcome handoff binding", (authority) => { authority.note.secrets_override = !authority.note.secrets_override; }],
	]) {
		const hostileC3UAuthority = structuredClone(fixture.c3uAuthority);
		mutate(hostileC3UAuthority);
		const handoffPath = "docs/HANDOFF_MODE_C.md";
		const heading = "### Active C3B source receipt reconciliation";
		const liveHandoff = present.get(handoffPath);
		const liveSection = currentStateSubsection(liveHandoff, heading);
		const hostileSection = replaceFixtureExactlyOnce(
			liveSection,
			c3uSealOutcomeDisclosure(fixture.c3uAuthority),
			c3uSealOutcomeDisclosure(hostileC3UAuthority),
			name,
		);
		const hostilePresent = new Map(present);
		hostilePresent.set(handoffPath, replaceCurrentStateSubsection(liveHandoff, heading, hostileSection));
		requireError(name, await checkPlan(
			repositoryRoot,
			hostilePresent,
			undefined,
			undefined,
			undefined,
			undefined,
			fixture.authority,
			fixture.c3rAuthority,
			fixture.c3qAuthority,
			fixture.c3tAuthority,
			fixture.c3uAuthority,
		), "C3B active maintenance section payload mismatch");
	}
	for (const [name, path, mutate, expected] of [
		["source grade", c3StatusPath, (body) => body.replace("| `TREE-EXACT` |", "| `UNRECEIPTED` |"), "source receipt map"],
		["receipt declaration pretty JSON", c3ReceiptDeclarationPath,
			(body) => `${JSON.stringify(JSON.parse(body), null, 2)}\n`, "exact canonical one-line JSON"],
		["receipt declaration duplicate key", c3ReceiptDeclarationPath,
			(body) => body.replace('{"schema_version":', '{"schema_version":"ignored","schema_version":'),
			"exact canonical one-line JSON"],
		["source status HTML-commented", c3StatusPath, (body) => `<!--\n${body}-->\n`, "reconciled source state"],
		["source status fenced", c3StatusPath, (body) => `\`\`\`markdown\n${body}\`\`\`\n`, "reconciled source state"],
		["source status indented code", c3StatusPath, (body) => body.split("\n").map((line) => `    ${line}`).join("\n"), "reconciled source state"],
		["pending-state residue", c3StatusPath, (body) => `${body}\n${c3PendingStatusState}\n`, "retains the pending source state"],
		["pending-heading residue", c3StatusPath, (body) => `${body}\n## Intended C3 source receipt map\n`, "retains the pending source receipt heading"],
		["no recursion", c3StatusPath, (body) => body.replace(c3ReceiptNoRecursionBoundary, "Receipt recursion omitted."), "no-recursion"],
		["receipt identity", "docs/HANDOFF_MODE_C.md", (body) => {
			const parsed = parseC3ReceiptBlock(body);
			if (parsed.block === undefined) throw new Error("C3 receipt identity fixture lacks the canonical block");
			return body.replace(parsed.block, parsed.block.replace(fixture.receipt.source_tree, "e".repeat(40)));
		}, "block payload mismatch"],
		["receipt block HTML-commented", "docs/HANDOFF_MODE_C.md", (body) => {
			const parsed = parseC3ReceiptBlock(body);
			return body.replace(parsed.block, `<!--\n${parsed.block}\n-->`);
		}, "source receipt block is not visible Markdown authority"],
		["receipt block fenced", "docs/HANDOFF_MODE_C.md", (body) => {
			const parsed = parseC3ReceiptBlock(body);
			return body.replace(parsed.block, `\`\`\`markdown\n${parsed.block}\n\`\`\``);
		}, "source receipt block is not visible Markdown authority"],
		["receipt residue", "docs/HANDOFF_MODE_C.md", (body) => `${body}C3 source commit \`${fixture.receipt.source_commit}\`, tree \`${fixture.receipt.source_tree}\`.\n`, "residue outside block"],
		["receipt post-block tail", "docs/HANDOFF_MODE_C.md", (body) => `${body}Unrelated trailing receipt-era prose.\n`,
			"exact terminal handoff block"],
		["C3Q lifecycle downgrade", "docs/HANDOFF_MODE_C.md", (body) => body.replace(c3bActiveHandoffState(fixture.receipt), c3qActiveHandoffState), "receipt-present operational phase"],
		["stale C3Q planned prose", "docs/HANDOFF_MODE_C.md", (body) => {
			const heading = "### Active C3B source receipt reconciliation";
			const section = currentStateSubsection(body, heading);
			return replaceCurrentStateSubsection(body, heading,
				section.replace(/\n\n$/u, "\n\nIts planned nine-claim final ledger remains pending.\n\n"));
		}, "C3B active maintenance section retains stale C3Q active prose"],
		["inert raw-heading boundary decoy with stale C3Q prose", "docs/HANDOFF_MODE_C.md", (body) => {
			const heading = "### Active C3B source receipt reconciliation";
			const section = currentStateSubsection(body, heading);
			return replaceCurrentStateSubsection(body, heading, section.replace(
				`${heading}\n\n`,
				`${heading}\n\n<!--\n### C3B-INERT-RAW-HEADING-DECOY\n-->\n\nIts planned nine-claim final ledger remains pending.\n\n`,
			));
		}, "C3B active maintenance section retains stale C3Q active prose"],
		["hidden stale C3Q unreceipted prose", "docs/HANDOFF_MODE_C.md", (body) => {
			const heading = "### Active C3B source receipt reconciliation";
			const section = currentStateSubsection(body, heading);
			return replaceCurrentStateSubsection(body, heading,
				section.replace(/\n\n$/u, "\n\n<!-- Every intended grade remains `UNRECEIPTED` at this boundary. -->\n\n"));
		}, "C3B active maintenance section retains stale C3Q active prose"],
		["phase-stable C3T corpus ruling removal", "docs/HANDOFF_MODE_C.md", (body) => body.replace(
			"C3Q's production refusal logic and synthetic C3Q/C3B renderer passed on its sealed tree",
			"The closed checker repair is omitted:",
		), "C3B active maintenance section payload mismatch"],
		["C3B self receipt", "docs/HANDOFF_MODE_C.md", (body) => `${body}\nC3B is sealed and strict-clean.\n`, "C3B self-receipt"],
		["C3B external strict-zero self receipt", "docs/VERIFICATION.md", (body) => `${body}\nC3B strict exit: 0\n`, "C3B self-receipt"],
	]) {
		const overrides = new Map(present);
		overrides.set(path, mutate(overrides.get(path) ?? await readText(repositoryRoot, path, overrides)));
		requireError(name, await checkPlan(
			repositoryRoot,
			overrides,
			undefined,
			undefined,
			undefined,
			undefined,
			fixture.authority,
			fixture.c3rAuthority,
			fixture.c3qAuthority,
			fixture.c3tAuthority,
			fixture.c3uAuthority,
		), expected);
	}
	const root = await mkdtemp(resolve(tmpdir(), "countershape-c3-local-evidence-"));
	try {
		const local = await buildLocalEvidenceFixture(root, {
			claimLabels: c3ClaimLabels, claimTypes: c3ClaimTypes, claimArgv: c3ExpectedClaimArgv,
			sourceCommit: fixture.receipt.source_commit, sourceTree: fixture.receipt.source_tree,
			archiveSessionEventCount: c3ClaimLabels.length,
		});
		const localErrors = await verifyLocalEvidence(root, local.receipt, c3ClaimLabels, c3ClaimTypes, c3ExpectedClaimArgv);
		if (localErrors.length > 0) throw new Error(`P07B-C C3 local-evidence baseline failed: ${localErrors.join(", ")}`);
		const hostile = structuredClone(local.receipt);
		hostile.evidence.ledger_archive.claims_jsonl_sha256 = "0".repeat(64);
		requireError("local ledger mutation", await verifyLocalEvidence(root, hostile, c3ClaimLabels, c3ClaimTypes, c3ExpectedClaimArgv), "local ledger core digest: claims.jsonl");
		const hostileSessions = structuredClone(local.sessions);
		const localRacePatternPosition = hostileSessions[7].event.argv.indexOf(c3RaceTestPattern);
		if (localRacePatternPosition === -1 ||
			hostileSessions[7].event.argv.lastIndexOf(c3RaceTestPattern) !== localRacePatternPosition) {
			throw new Error("P07B-C C3 local-evidence self-test cannot locate the exact race pattern");
		}
		hostileSessions[7].event.argv[localRacePatternPosition] = c3RaceNotePreviewPattern;
		await writeJSONLines(local.sessionPath, hostileSessions);
		requireError(
			"local ledger note-preview substitution",
			await verifyLocalEvidence(root, local.receipt, c3ClaimLabels, c3ClaimTypes, c3ExpectedClaimArgv),
			"local ledger event 8 argv",
		);
		await writeJSONLines(local.sessionPath, local.sessions);
	} finally {
		await rm(root, { recursive: true, force: true });
	}
	console.log(`P07B-C C3 receipt checker self-test passed: ${rejected} declaration, exact-diff, Git-note, lifecycle, block, no-recursion, and local-evidence mutations rejected; absent-to-present and present-to-absent-to-present controlled paths are byte-stable; ${subsectionDecoyControls} inert raw-heading decoys removed by visible-boundary transition`);
	return Object.freeze({ rejected, present, fixture });
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

export async function checkPlan(
	root = repositoryRoot,
	overrides = new Map(),
	receiptAuthority,
	c1ReceiptAuthority,
	c2ReceiptAuthority,
	c3pReceiptAuthority,
	c3ReceiptAuthority,
	c3rReceiptAuthority,
	c3qReceiptAuthority,
	c3tReceiptAuthority,
	c3uReceiptAuthority,
) {
	const errors = [];
	const bodies = new Map();
	const computedMarkdownDigest = computedPlanAuthorityMarkdownDigest();
	if (computedMarkdownDigest !== planAuthorityMarkdownDigest) {
		errors.push(`internal plan-authority Markdown corpus digest mismatch: ${computedMarkdownDigest}`);
	}
	for (const path of planAuthorityMarkdownPaths) {
		try {
			bodies.set(path, await readText(root, path, overrides));
		} catch (error) {
			errors.push(`${path}: plan-authority Markdown corpus unreadable (${error.message})`);
		}
	}

	for (const [path, snippets] of Object.entries(requiredText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
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
	for (const [path, snippets] of Object.entries(requiredC3VText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C3V ruling: ${JSON.stringify(snippet)}`);
		}
	}
	for (const [path, snippets] of Object.entries(requiredC3MaintenanceText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C3M ruling: ${JSON.stringify(snippet)}`);
		}
	}
	for (const [path, snippets] of Object.entries(requiredC3AText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C3A ruling: ${JSON.stringify(snippet)}`);
		}
	}
	for (const [path, snippets] of Object.entries(requiredC3LText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C3L ruling: ${JSON.stringify(snippet)}`);
		}
	}
	for (const [path, snippets] of Object.entries(requiredC3FText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C3F ruling: ${JSON.stringify(snippet)}`);
		}
	}
	for (const [path, snippets] of Object.entries(requiredC3SText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C3S ruling: ${JSON.stringify(snippet)}`);
		}
	}
	for (const [path, snippets] of Object.entries(requiredC3Text)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C3 ruling: ${JSON.stringify(snippet)}`);
		}
	}
	for (const [path, snippets] of Object.entries(requiredC3RText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C3R ruling: ${JSON.stringify(snippet)}`);
		}
	}
	for (const [path, snippets] of Object.entries(requiredC3QText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		if (path === c3qStatusPath) {
			try {
				body = visibleMarkdownLifecycleBody(body);
			} catch (error) {
				errors.push(`${path}: C3Q visible authority invalid (${error.message})`);
				continue;
			}
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C3Q ruling: ${JSON.stringify(snippet)}`);
		}
	}
	for (const [path, snippets] of Object.entries(requiredC3TText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		if (path === c3tStatusPath) {
			try {
				body = visibleMarkdownLifecycleBody(body);
			} catch (error) {
				errors.push(`${path}: C3T visible authority invalid (${error.message})`);
				continue;
			}
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C3T ruling: ${JSON.stringify(snippet)}`);
		}
	}
	for (const [path, snippets] of Object.entries(requiredC3UText)) {
		let body;
		try {
			body = bodies.get(path) ?? await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		if (path === c3uStatusPath) {
			try {
				body = visibleMarkdownLifecycleBody(body);
			} catch (error) {
				errors.push(`${path}: C3U visible authority invalid (${error.message})`);
				continue;
			}
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C3U ruling: ${JSON.stringify(snippet)}`);
		}
	}
	try {
		const manifestBytes = await readBytes(root, c3PredecessorManifestPath, overrides);
		const manifest = JSON.parse(decodeGitUTF8(manifestBytes, "C3 predecessor declaration"));
		if (!manifestBytes.equals(expectedC3PredecessorManifestBytes()) ||
			!isDeepStrictEqual(manifest, expectedC3PredecessorManifest())) {
			errors.push(`${c3PredecessorManifestPath}: exact C3S/C3F/C3L/C3A authority manifest mismatch`);
		}
	} catch (error) {
		errors.push(`${c3PredecessorManifestPath}: invalid exact predecessor authority manifest (${error.message})`);
	}
	for (const path of requiredC3Paths.filter((candidate) =>
		candidate.endsWith(".md") && candidate !== c3StatusPath && candidate !== "docs/HANDOFF_MODE_C.md")) {
		const body = bodies.get(path);
		if (body !== undefined) rejectC3CanonicalSelfClaims(body, path, errors);
	}
	const c3aStatusForSelfReceipt = bodies.get(c3aStatusPath);
	if (c3aStatusForSelfReceipt !== undefined) {
		rejectC3ACanonicalSelfClaims(c3aStatusForSelfReceipt, c3aStatusPath, errors);
	}
	for (const path of [c3fStatusPath]) {
		const body = bodies.get(path);
		if (body !== undefined) rejectC3FCanonicalSelfClaims(body, path, errors);
	}
	const c3sStatusForSelfReceipt = bodies.get(c3sStatusPath);
	if (c3sStatusForSelfReceipt !== undefined) {
		rejectC3SCanonicalSelfClaims(c3sStatusForSelfReceipt, c3sStatusPath, errors);
	}
	// C3V's frozen-byte observation is historical authority carried by its sealed status.
	// C3 deliberately owns both verifier files, so later exact-roster source units must not
	// compare the current bytes to the older C3V tree. Current verifier behavior is instead
	// re-established by C3's dedicated verifier self-test and cumulative verification claims.
	const computedC2Digest = `sha256:${createHash("sha256").update(`${requiredC2Paths.join("\n")}\n`, "utf8").digest("hex")}`;
	if (computedC2Digest !== requiredC2Digest) errors.push(`internal C2 roster digest mismatch: ${computedC2Digest}`);
	const computedC2BDigest = `sha256:${createHash("sha256").update(`${requiredC2BPaths.join("\n")}\n`, "utf8").digest("hex")}`;
	if (computedC2BDigest !== requiredC2BDigest) errors.push(`internal C2B roster digest mismatch: ${computedC2BDigest}`);
	for (const [unit, paths, digest] of [
		["C3P", requiredC3PPaths, requiredC3PDigest],
		["C3V", requiredC3VPaths, requiredC3VDigest],
		["C3M", requiredC3MaintenancePaths, requiredC3MaintenanceDigest],
		["C3PB", requiredC3PBPaths, requiredC3PBDigest],
		["C3A", requiredC3APaths, requiredC3ADigest],
		["C3L", requiredC3LPaths, requiredC3LDigest],
		["C3F", requiredC3FPaths, requiredC3FDigest],
		["C3S", requiredC3SPaths, requiredC3SDigest],
		["C3", requiredC3Paths, requiredC3Digest],
		["C3R", requiredC3RPaths, requiredC3RDigest],
		["C3Q", requiredC3QPaths, requiredC3QDigest],
		["C3T", requiredC3TPaths, requiredC3TDigest],
		["C3U", requiredC3UPaths, requiredC3UDigest],
		["C3B", requiredC3BPaths, requiredC3BDigest],
	]) {
		const computed = `sha256:${createHash("sha256").update(`${paths.join("\n")}\n`, "utf8").digest("hex")}`;
		if (computed !== digest) errors.push(`internal ${unit} roster digest mismatch: ${computed}`);
	}
	if (requiredC3AddedPaths.size !== 20 ||
		![...requiredC3AddedPaths].every((path) => requiredC3Paths.includes(path)) ||
		requiredC3Paths.length - requiredC3AddedPaths.size !== 20) {
		errors.push("internal C3 exact 20-added/20-modified source partition mismatch");
	}
	if (!isDeepStrictEqual(requiredC3BReceiptClaims,
		c3bClaimLabels.map((label, index) => ({ label, type: c3bClaimTypes[index] })))) {
		errors.push("internal C3B receipt claim roster mismatch");
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
		requireExactlyOnce(
			verificationBody,
			requiredC3VDeclaration,
			"docs/VERIFICATION.md",
			"C3V throughput reconciliation scope declaration",
			errors,
		);
		requireExactlyOnce(
			verificationBody,
			requiredC3MaintenanceDeclaration,
			"docs/VERIFICATION.md",
			"C3P receipt-phase maintenance scope declaration",
			errors,
		);
		requireExactlyOnce(
			verificationBody,
			requiredC3ADeclaration,
			"docs/VERIFICATION.md",
			"C3A cumulative-admission maintenance scope declaration",
			errors,
		);
		requireExactlyOnce(
			verificationBody,
			requiredC3LDeclaration,
			"docs/VERIFICATION.md",
			"C3L legacy-lattice maintenance scope declaration",
			errors,
		);
		requireExactlyOnce(
			verificationBody,
			requiredC3FDeclaration,
			"docs/VERIFICATION.md",
			"C3F B future-surface admission maintenance scope declaration",
			errors,
		);
		requireExactlyOnce(
			verificationBody,
			requiredC3SDeclaration,
			"docs/VERIFICATION.md",
			"C3S cumulative U6 self-test maintenance scope declaration",
			errors,
		);
		requireExactlyOnce(
			verificationBody,
			requiredC3RDeclaration,
			"docs/VERIFICATION.md",
			"C3R sealed-note preview maintenance scope declaration",
			errors,
		);
		requireExactlyOnce(
			verificationBody,
			requiredC3QDeclaration,
			"docs/VERIFICATION.md",
			"C3Q active-unit self-receipt maintenance scope declaration",
			errors,
		);
		requireExactlyOnce(
			verificationBody,
			requiredC3TDeclaration,
			"docs/VERIFICATION.md",
			"C3T receipt-phase self-test maintenance scope declaration",
			errors,
		);
		requireExactlyOnce(
			verificationBody,
			requiredC3UDeclaration,
			"docs/VERIFICATION.md",
			"C3U cross-phase receipt-fixture maintenance scope declaration",
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
	const c3vStatusBody = bodies.get(c3vStatusPath);
	if (c3vStatusBody !== undefined) {
		requireExactlyOnce(
			c3vStatusBody,
			`- **Scope:** 7 exact mode-\`100644\` paths, no prefixes; sorted-newline roster \`${requiredC3VDigest}\``,
			c3vStatusPath,
			"C3V source scope",
			errors,
		);
		requireExactlyOnce(c3vStatusBody, "- **Commit subject:** `docs: reconcile verifier throughput proposal`", c3vStatusPath, "C3V commit subject", errors);
		for (let index = 0; index < c3vClaimLabels.length; index += 1) {
			requireExactlyOnce(
				c3vStatusBody,
				`| \`${c3vClaimLabels[index]}\` | \`${c3vClaimTypes[index]}\` | \`UNRECEIPTED\` |`,
				c3vStatusPath,
				"C3V planned claim map",
				errors,
			);
			requireClaimLabelExactlyOnce(c3vStatusBody, c3vClaimLabels[index], c3vStatusPath, "C3V planned claim map", errors);
		}
		if (/\|\s*`P07B-C C3V [^`]+`\s*\|\s*`(?:tests-pass|command-succeeded)`\s*\|\s*`(?:TREE-EXACT|RECORDED-EXACT)`/u.test(c3vStatusBody)) {
			errors.push(`${c3vStatusPath}: C3V pre-seal status cannot grade itself`);
		}
	}
	const c3MaintenanceStatusBody = bodies.get(c3MaintenanceStatusPath);
	if (c3MaintenanceStatusBody !== undefined) {
		requireExactlyOnce(
			c3MaintenanceStatusBody,
			requiredC3MaintenanceDeclaration,
			c3MaintenanceStatusPath,
			"C3M source scope",
			errors,
		);
		requireExactlyOnce(
			c3MaintenanceStatusBody,
			"- **Commit subject:** `fix: preserve interposed receipt-phase authority`",
			c3MaintenanceStatusPath,
			"C3M commit subject",
			errors,
		);
		for (let index = 0; index < c3MaintenanceClaimLabels.length; index += 1) {
			requireExactlyOnce(
				c3MaintenanceStatusBody,
				`| \`${c3MaintenanceClaimLabels[index]}\` | \`${c3MaintenanceClaimTypes[index]}\` | \`UNRECEIPTED\` |`,
				c3MaintenanceStatusPath,
				"C3M intended claim map",
				errors,
			);
			requireClaimLabelExactlyOnce(
				c3MaintenanceStatusBody,
				c3MaintenanceClaimLabels[index],
				c3MaintenanceStatusPath,
				"C3M intended claim map",
				errors,
			);
		}
		if (/\|\s*`P07B-C C3M [^`]+`\s*\|\s*`(?:tests-pass|command-succeeded)`\s*\|\s*`(?:TREE-EXACT|RECORDED-EXACT)`/u.test(c3MaintenanceStatusBody)) {
			errors.push(`${c3MaintenanceStatusPath}: C3M pre-seal status cannot grade itself`);
		}
	}
	const c3aStatusBody = bodies.get(c3aStatusPath);
	if (c3aStatusBody !== undefined) {
		requireExactlyOnce(c3aStatusBody, requiredC3ADeclaration, c3aStatusPath, "C3A source scope", errors);
		requireExactlyOnce(
			c3aStatusBody,
			"- **Commit subject:** `fix: align pre-C3 architecture and runtime bounds`",
			c3aStatusPath,
			"C3A commit subject",
			errors,
		);
		for (let index = 0; index < c3aClaimLabels.length; index += 1) {
			requireExactlyOnce(
				c3aStatusBody,
				`| \`${c3aClaimLabels[index]}\` | \`${c3aClaimTypes[index]}\` | \`UNRECEIPTED\` |`,
				c3aStatusPath,
				"C3A intended claim map",
				errors,
			);
			requireClaimLabelExactlyOnce(c3aStatusBody, c3aClaimLabels[index], c3aStatusPath, "C3A intended claim map", errors);
		}
		if (/\|\s*`P07B-C C3A [^`]+`\s*\|\s*`(?:tests-pass|command-succeeded)`\s*\|\s*`(?:TREE-EXACT|RECORDED-EXACT)`/u.test(c3aStatusBody)) {
			errors.push(`${c3aStatusPath}: C3A pre-seal status cannot grade itself`);
		}
		rejectReceiptSelfClaims(c3aStatusBody, c3aStatusPath, "C3A", errors);
	}
	const c3lStatusBody = bodies.get(c3lStatusPath);
	if (c3lStatusBody !== undefined) {
		requireExactlyOnce(c3lStatusBody, requiredC3LDeclaration, c3lStatusPath, "C3L source scope", errors);
		requireExactlyOnce(c3lStatusBody, c3lStatusBoundaryLine, c3lStatusPath, "C3L active boundary", errors);
		requireExactlyOnce(c3lStatusBody, c3lStatusParentLine, c3lStatusPath, "C3L sealed parent identity", errors);
		requireExactlyOnce(c3lStatusBody, c3lStatusUnreceiptedLine, c3lStatusPath, "C3L unreceipted state", errors);
		requireExactlyOnce(
			c3lStatusBody,
			"- **Commit subject:** `fix: admit exact C3 store model edge through U6`",
			c3lStatusPath,
			"C3L commit subject",
			errors,
		);
		for (let index = 0; index < c3lClaimLabels.length; index += 1) {
			requireExactlyOnce(
				c3lStatusBody,
				`| \`${c3lClaimLabels[index]}\` | \`${c3lClaimTypes[index]}\` | \`UNRECEIPTED\` |`,
				c3lStatusPath,
				"C3L intended claim map",
				errors,
			);
			requireClaimLabelExactlyOnce(c3lStatusBody, c3lClaimLabels[index], c3lStatusPath, "C3L intended claim map", errors);
		}
		rejectActiveC3LSelfClaims(c3lStatusBody, c3lStatusPath, errors);
	}
	const c3fStatusBody = bodies.get(c3fStatusPath);
	if (c3fStatusBody !== undefined) {
		requireExactlyOnce(c3fStatusBody, requiredC3FDeclaration, c3fStatusPath, "C3F source scope", errors);
		requireExactlyOnce(c3fStatusBody, c3fStatusBoundaryLine, c3fStatusPath, "C3F active boundary", errors);
		requireExactlyOnce(c3fStatusBody, c3fStatusParentLine, c3fStatusPath, "C3F sealed parent identity", errors);
		requireExactlyOnce(c3fStatusBody, c3fStatusUnreceiptedLine, c3fStatusPath, "C3F unreceipted state", errors);
		requireExactlyOnce(
			c3fStatusBody,
			"- **Commit subject:** `fix: admit exact C3 target codec surface through B`",
			c3fStatusPath,
			"C3F commit subject",
			errors,
		);
		for (let index = 0; index < c3fClaimLabels.length; index += 1) {
			requireExactlyOnce(
				c3fStatusBody,
				`| \`${c3fClaimLabels[index]}\` | \`${c3fClaimTypes[index]}\` | \`UNRECEIPTED\` |`,
				c3fStatusPath,
				"C3F intended claim map",
				errors,
			);
			requireClaimLabelExactlyOnce(c3fStatusBody, c3fClaimLabels[index], c3fStatusPath, "C3F intended claim map", errors);
		}
		rejectActiveC3FSelfClaims(c3fStatusBody, c3fStatusPath, errors);
	}
	const c3sStatusBody = bodies.get(c3sStatusPath);
	if (c3sStatusBody !== undefined) {
		requireExactlyOnce(c3sStatusBody, requiredC3SDeclaration, c3sStatusPath, "C3S source scope", errors);
		requireExactlyOnce(c3sStatusBody, c3sStatusBoundaryLine, c3sStatusPath, "C3S active boundary", errors);
		requireExactlyOnce(c3sStatusBody, c3sStatusParentLine, c3sStatusPath, "C3S sealed parent identity", errors);
		requireExactlyOnce(c3sStatusBody, c3sStatusUnreceiptedLine, c3sStatusPath, "C3S unreceipted state", errors);
		requireExactlyOnce(
			c3sStatusBody,
			"- **Commit subject:** `fix: make U6 C3 fixture phase-stable`",
			c3sStatusPath,
			"C3S commit subject",
			errors,
		);
		for (let index = 0; index < c3sClaimLabels.length; index += 1) {
			requireExactlyOnce(
				c3sStatusBody,
				`| \`${c3sClaimLabels[index]}\` | \`${c3sClaimTypes[index]}\` | \`UNRECEIPTED\` |`,
				c3sStatusPath,
				"C3S intended claim map",
				errors,
			);
			requireClaimLabelExactlyOnce(c3sStatusBody, c3sClaimLabels[index], c3sStatusPath, "C3S intended claim map", errors);
		}
		rejectActiveC3SSelfClaims(c3sStatusBody, c3sStatusPath, errors);
	}
	const c3rStatusBody = bodies.get(c3rStatusPath);
	if (c3rStatusBody !== undefined) {
		requireExactlyOnce(c3rStatusBody, requiredC3RDeclaration, c3rStatusPath, "C3R source scope", errors);
		requireExactlyOnce(c3rStatusBody, c3rStatusBoundaryLine, c3rStatusPath, "C3R active boundary", errors);
		requireExactlyOnce(c3rStatusBody, c3rStatusParentLine, c3rStatusPath, "C3R sealed parent identity", errors);
		requireExactlyOnce(c3rStatusBody, c3rStatusUnreceiptedLine, c3rStatusPath, "C3R unreceipted state", errors);
		requireExactlyOnce(
			c3rStatusBody,
			`- **Commit subject:** \`${c3rSourceSubject}\``,
			c3rStatusPath,
			"C3R commit subject",
			errors,
		);
		for (let index = 0; index < c3rClaimLabels.length; index += 1) {
			requireExactlyOnce(
				c3rStatusBody,
				`| \`${c3rClaimLabels[index]}\` | \`${c3rClaimTypes[index]}\` | \`UNRECEIPTED\` |`,
				c3rStatusPath,
				"C3R intended claim map",
				errors,
			);
			requireClaimLabelExactlyOnce(c3rStatusBody, c3rClaimLabels[index], c3rStatusPath, "C3R intended claim map", errors);
		}
		rejectReceiptSelfClaims(c3rStatusBody, c3rStatusPath, "C3R", errors);
	}
	const c3qRawStatusBody = bodies.get(c3qStatusPath);
	let c3qStatusBody;
	if (c3qRawStatusBody !== undefined) {
		try { c3qStatusBody = visibleMarkdownLifecycleBody(c3qRawStatusBody); } catch { /* reported above */ }
	}
	if (c3qStatusBody !== undefined) {
		requireExactlyOnce(c3qStatusBody, requiredC3QDeclaration, c3qStatusPath, "C3Q source scope", errors);
		requireExactlyOnce(c3qStatusBody, c3qStatusBoundaryLine, c3qStatusPath, "C3Q active boundary", errors);
		requireExactlyOnce(c3qStatusBody, c3qStatusParentLine, c3qStatusPath, "C3Q sealed parent identity", errors);
		requireExactlyOnce(c3qStatusBody, c3qStatusUnreceiptedLine, c3qStatusPath, "C3Q unreceipted state", errors);
		requireExactlyOnce(c3qStatusBody, `- **Commit subject:** \`${c3qSourceSubject}\``, c3qStatusPath, "C3Q commit subject", errors);
		for (let index = 0; index < c3qClaimLabels.length; index += 1) {
			requireExactlyOnce(
				c3qStatusBody,
				`| \`${c3qClaimLabels[index]}\` | \`${c3qClaimTypes[index]}\` | \`UNRECEIPTED\` |`,
				c3qStatusPath,
				"C3Q intended claim map",
				errors,
			);
			requireClaimLabelExactlyOnce(c3qStatusBody, c3qClaimLabels[index], c3qStatusPath, "C3Q intended claim map", errors);
		}
		rejectReceiptSelfClaims(c3qStatusBody, c3qStatusPath, "C3Q", errors);
	}
	const c3tRawStatusBody = bodies.get(c3tStatusPath);
	let c3tStatusBody;
	if (c3tRawStatusBody !== undefined) {
		try { c3tStatusBody = visibleMarkdownLifecycleBody(c3tRawStatusBody); } catch { /* reported above */ }
	}
	if (c3tStatusBody !== undefined) {
		requireExactlyOnce(c3tStatusBody, requiredC3TDeclaration, c3tStatusPath, "C3T source scope", errors);
		requireExactlyOnce(c3tStatusBody, c3tStatusBoundaryLine, c3tStatusPath, "C3T active boundary", errors);
		requireExactlyOnce(c3tStatusBody, c3tStatusParentLine, c3tStatusPath, "C3T sealed parent identity", errors);
		requireExactlyOnce(c3tStatusBody, c3tStatusUnreceiptedLine, c3tStatusPath, "C3T unreceipted state", errors);
		requireExactlyOnce(c3tStatusBody, `- **Commit subject:** \`${c3tSourceSubject}\``, c3tStatusPath, "C3T commit subject", errors);
		for (let index = 0; index < c3tClaimLabels.length; index += 1) {
			requireExactlyOnce(
				c3tStatusBody,
				`| \`${c3tClaimLabels[index]}\` | \`${c3tClaimTypes[index]}\` | \`UNRECEIPTED\` |`,
				c3tStatusPath,
				"C3T intended claim map",
				errors,
			);
			requireClaimLabelExactlyOnce(c3tStatusBody, c3tClaimLabels[index], c3tStatusPath, "C3T intended claim map", errors);
		}
		rejectReceiptSelfClaims(c3tStatusBody, c3tStatusPath, "C3T", errors);
	}
	const c3uRawStatusBody = bodies.get(c3uStatusPath);
	let c3uStatusBody;
	if (c3uRawStatusBody !== undefined) {
		try { c3uStatusBody = visibleMarkdownLifecycleBody(c3uRawStatusBody); } catch { /* reported above */ }
	}
	if (c3uStatusBody !== undefined) {
		requireExactlyOnce(c3uStatusBody, requiredC3UDeclaration, c3uStatusPath, "C3U source scope", errors);
		requireExactlyOnce(c3uStatusBody, c3uStatusBoundaryLine, c3uStatusPath, "C3U active boundary", errors);
		requireExactlyOnce(c3uStatusBody, c3uStatusParentLine, c3uStatusPath, "C3U sealed parent identity", errors);
		requireExactlyOnce(c3uStatusBody, c3uStatusUnreceiptedLine, c3uStatusPath, "C3U unreceipted state", errors);
		requireExactlyOnce(c3uStatusBody, `- **Commit subject:** \`${c3uSourceSubject}\``, c3uStatusPath, "C3U commit subject", errors);
		for (let index = 0; index < c3uClaimLabels.length; index += 1) {
			requireExactlyOnce(
				c3uStatusBody,
				`| \`${c3uClaimLabels[index]}\` | \`${c3uClaimTypes[index]}\` | \`UNRECEIPTED\` |`,
				c3uStatusPath,
				"C3U intended claim map",
				errors,
			);
			requireClaimLabelExactlyOnce(c3uStatusBody, c3uClaimLabels[index], c3uStatusPath, "C3U intended claim map", errors);
		}
		rejectReceiptSelfClaims(c3uStatusBody, c3uStatusPath, "C3U", errors);
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
	let c3Status;
	try {
		c3Status = await readText(root, c3StatusPath, overrides);
		bodies.set(c3StatusPath, c3Status);
	} catch (error) {
		errors.push(`${c3StatusPath}: unreadable (${error.message})`);
	}
	let c3ReceiptPresentForPhase = true;
	try {
		await readText(root, c3ReceiptDeclarationPath, overrides);
	} catch (error) {
		if (error.code === "ENOENT") c3ReceiptPresentForPhase = false;
		else errors.push(`${c3ReceiptDeclarationPath}: unreadable (${error.message})`);
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
			if (c3Status !== undefined) {
				await checkC3ReceiptPhase(
					root,
					overrides,
					c3Status,
					handoff,
					errors,
					c3ReceiptAuthority,
					c3rReceiptAuthority,
					c3qReceiptAuthority,
					c3tReceiptAuthority,
					c3uReceiptAuthority,
				);
			}
		}
	const activeSelfReceiptUnit = c3ReceiptPresentForPhase ? "C3B" : "C3U";
	rejectActiveUnitSelfReceipts(bodies, activeSelfReceiptUnit, errors);

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
		if (!isDeepStrictEqual(specification.units.C3V.exact, requiredC3VPaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3V exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3V.prefixes, []) || specification.units.C3V.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3V source profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3M.exact, requiredC3MaintenancePaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3M exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3M.prefixes, []) || specification.units.C3M.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3M source profile mismatch");
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
		if (!isDeepStrictEqual(specification.units.C3A.exact, requiredC3APaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3A exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3A.prefixes, []) || specification.units.C3A.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3A source profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3L.exact, requiredC3LPaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3L exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3L.prefixes, []) || specification.units.C3L.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3L source profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3F.exact, requiredC3FPaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3F exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3F.prefixes, []) || specification.units.C3F.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3F source profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3S.exact, requiredC3SPaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3S exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3S.prefixes, []) || specification.units.C3S.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3S source profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3.exact, requiredC3Paths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3 exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3.prefixes, []) || specification.units.C3.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3 source profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3R.exact, requiredC3RPaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3R exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3R.prefixes, []) || specification.units.C3R.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3R source profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3Q.exact, requiredC3QPaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3Q exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3Q.prefixes, []) || specification.units.C3Q.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3Q source profile mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3T.exact, requiredC3TPaths)) {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3T exact path roster mismatch");
		}
		if (!isDeepStrictEqual(specification.units.C3T.prefixes, []) || specification.units.C3T.verification_profile !== "SOURCE_FULL") {
			errors.push("spec/verification/p07b-c-unit-paths.json: C3T source profile mismatch");
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
		throw new Error(`P07B-C phase fixture ${name} anchor count ${occurrences}`);
	}
	return body.replace(needle, replacement);
}

async function syntheticC2SealedSourcePhaseFixture() {
	const sourceStatus = readSealedC2StatusFixture();
	const currentHandoff = await readText(repositoryRoot, "docs/HANDOFF_MODE_C.md", new Map());
	const strippedHandoff = withoutC2ReceiptBlock(currentHandoff);
	const sourcePhaseHandoff = replaceFixtureExactlyOnce(
		strippedHandoff,
		`Sealed C2B commit \`${sealedC2BIdentity.commit}\`, tree \`${sealedC2BIdentity.tree}\`, remains`,
		`Later predecessor identity inputs are \`${sealedC2BIdentity.commit}\` and \`${sealedC2BIdentity.tree}\`; the chain remains`,
		"later C2B evidence neutralization",
	);
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

async function syntheticC2ReceiptPlanFixture(fixture, baseOverrides = new Map()) {
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
	const currentHandoff = withoutC2ReceiptBlock(await readText(repositoryRoot, "docs/HANDOFF_MODE_C.md", baseOverrides));
	const handoff = insertLegacyReceiptBlockBeforeOptionalTerminalC3(
		currentHandoff,
		fixture.handoff,
		"C2 synthetic receipt fixture",
		"trimmed-blank",
	);
	return {
		overrides: new Map([
			...baseOverrides,
			[c2StatusPath, status],
			["docs/HANDOFF_MODE_C.md", handoff],
			[c2ReceiptDeclarationPath, `${JSON.stringify(fixture.receipt)}\n`],
		]),
		authority: fixture.authority,
		priorHandoff: currentHandoff,
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
		["C2B self receipt", `${fixture.status}\nC2B commit \`${"8".repeat(40)}\``, fixture.handoff, "C2B self-receipt forbidden"],
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
	const corpusReport = planAuthorityMarkdownCorpusReport();
	const corpusReportLines = corpusReport.split("\n");
	const renderedCorpusPaths = corpusReportLines.slice(1, -2).map((line) => line.startsWith("- ") ? line.slice(2) : undefined);
	if (corpusReportLines[0] !== `P07B-C plan-authority Markdown corpus: paths=44 digest=${planAuthorityMarkdownDigest}` ||
		corpusReportLines.length !== planAuthorityMarkdownPaths.length + 3 ||
		!isDeepStrictEqual(renderedCorpusPaths, planAuthorityMarkdownPaths) ||
		corpusReportLines.at(-2) !== "scope: positive-receipt-form refusal only; not a general Markdown or security audit" ||
		corpusReportLines.at(-1) !== "") {
		throw new Error("P07B-C plan-authority Markdown corpus report is not deterministic and complete");
	}
	let invalidGitUTF8Rejected = false;
	try { decodeGitUTF8(Buffer.from([0xff]), "self-test Git output"); } catch { invalidGitUTF8Rejected = true; }
	if (!invalidGitUTF8Rejected) throw new Error("P07B-C plan checker self-test accepted invalid Git UTF-8");
	const redactedC3Preview = [...c3ExpectedClaimArgv[0]];
	for (const position of [2, 4, 5, 6, 7, 8, 31, 32, 33, 35, 36]) {
		redactedC3Preview[position] = "«redacted:high-entropy»";
	}
	if (!validateC3ClaimPreview(0, redactedC3Preview)) {
		throw new Error("P07B-C plan checker self-test rejected the exact C3 didrun redaction shape");
	}
	const overRedactedC3Preview = [...redactedC3Preview];
	overRedactedC3Preview[30] = "«redacted:high-entropy»";
	if (validateC3ClaimPreview(0, overRedactedC3Preview)) {
		throw new Error("P07B-C plan checker self-test accepted C3 didrun redaction outside the admitted positions");
	}
	const redactedC3RacePreview = [...c3ExpectedClaimArgv[7]];
	for (const position of [2, 4, 5, 6, 7, 8, 31, 32, 33, 35, 36]) {
		redactedC3RacePreview[position] = "«redacted:high-entropy»";
	}
	const racePatternPosition = redactedC3RacePreview.indexOf(c3RaceTestPattern);
	if (racePatternPosition === -1 || redactedC3RacePreview.lastIndexOf(c3RaceTestPattern) !== racePatternPosition) {
		throw new Error("P07B-C plan checker self-test cannot locate the exact C3 race pattern");
	}
	redactedC3RacePreview[racePatternPosition] = c3RaceNotePreviewPattern;
	if (!validateC3ClaimPreview(7, redactedC3RacePreview)) {
		throw new Error("P07B-C plan checker self-test rejected the exact sealed C3 race-note preview redaction");
	}
	for (const [name, mutate] of [
		["second race alternative redaction", (preview) => { preview[racePatternPosition] = c3RaceNotePreviewPattern.replace("TestC3HostEpochConcurrentMeasurementsNeverCache", "«redacted:high-entropy»"); }],
		["redaction suffix", (preview) => { preview[racePatternPosition] = c3RaceNotePreviewPattern.replace("«redacted:high-entropy»", "«redacted:high-entropy»suffix"); }],
		["whole race selector redaction", (preview) => { preview[racePatternPosition] = "«redacted:high-entropy»"; }],
		["race tail drift", (preview) => { preview[preview.length - 1] = "./internal/store-evil"; }],
		["unrelated tail redaction", (preview) => { preview[preview.length - 1] = "«redacted:high-entropy»"; }],
	]) {
		const hostile = [...redactedC3RacePreview];
		mutate(hostile);
		if (validateC3ClaimPreview(7, hostile)) {
			throw new Error(`P07B-C plan checker self-test accepted hostile C3 ${name}`);
		}
	}
	const unrelatedClaimTailRedaction = [...c3ExpectedClaimArgv[6]];
	unrelatedClaimTailRedaction[unrelatedClaimTailRedaction.length - 1] = "«redacted:high-entropy»";
	if (validateC3ClaimPreview(6, unrelatedClaimTailRedaction)) {
		throw new Error("P07B-C plan checker self-test accepted C3 note-tail redaction on an unrelated claim");
	}
	const localEvidenceMutations = await runLocalEvidenceSelfTest();
	const receiptModePhaseMutations = await runC1ReceiptModePhaseSelfTest();
	const c2ReceiptMutations = await runC2ReceiptCheckerSelfTest();
	const c2SealedSourcePhaseMutations = await runC2SealedSourcePhaseSelfTest();
	const c2LocalEvidenceMutations = await runC2LocalEvidencePositiveSelfTest();
	const c3pReceiptMutations = await runC3PReceiptSelfTest();
	const c3ReceiptResult = await runC3ReceiptSelfTest();
	const c3ReceiptMutations = c3ReceiptResult.rejected;
	const activeSelfReceiptMatrix = runActiveUnitSelfReceiptMatrixSelfTest();

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
	const currentC3VStatus = await readText(repositoryRoot, c3vStatusPath, new Map());
	const currentC3MaintenanceStatus = await readText(repositoryRoot, c3MaintenanceStatusPath, new Map());
	const currentC3AStatus = await readText(repositoryRoot, c3aStatusPath, new Map());
	const currentC3LStatus = await readText(repositoryRoot, c3lStatusPath, new Map());
	const currentC3FStatus = await readText(repositoryRoot, c3fStatusPath, new Map());
	const currentC3SStatus = await readText(repositoryRoot, c3sStatusPath, new Map());
	const currentC3RStatus = await readText(repositoryRoot, c3rStatusPath, new Map());
	const currentC3QStatus = await readText(repositoryRoot, c3qStatusPath, new Map());
	const currentC3TStatus = await readText(repositoryRoot, c3tStatusPath, new Map());
	const currentC3UStatus = await readText(repositoryRoot, c3uStatusPath, new Map());
	const syntheticReceiptText = `${JSON.stringify(syntheticReceipt, null, 2)}\n`;
	const c0FixturePattern = /\n?<!-- P07B-C-C0A-RECEIPTS:START -->[\s\S]*?<!-- P07B-C-C0A-RECEIPTS:END -->\n?/gu;
	const stripC0FixtureBlock = (body) => {
		const startCount = countOccurrences(body, "<!-- P07B-C-C0A-RECEIPTS:START -->");
		const endCount = countOccurrences(body, "<!-- P07B-C-C0A-RECEIPTS:END -->");
		if (startCount !== endCount || startCount > 1) {
			throw new Error(`P07B-C C0 fixture receipt marker cardinality start=${startCount} end=${endCount}`);
		}
		return startCount === 0 ? body : body.replace(c0FixturePattern, "\n");
	};
	const syntheticC0ReceiptPlanFixture = async (baseOverrides = new Map()) => {
		const priorHandoff = stripC0FixtureBlock(await readText(repositoryRoot, "docs/HANDOFF_MODE_C.md", baseOverrides));
		const handoff = insertLegacyReceiptBlockBeforeOptionalTerminalC3(
			priorHandoff,
			receiptBlock,
			"C0 synthetic receipt fixture",
			"trimmed-blank",
		);
		return Object.freeze({
			overrides: new Map([
				...baseOverrides,
				[statusPath, syntheticStatus],
				[receiptDeclarationPath, syntheticReceiptText],
				["docs/HANDOFF_MODE_C.md", handoff],
			]),
			authority: syntheticAuthority,
			priorHandoff,
		});
	};
	const syntheticC0Plan = await syntheticC0ReceiptPlanFixture();
	const syntheticHandoff = syntheticC0Plan.overrides.get("docs/HANDOFF_MODE_C.md");
	const syntheticOverrides = syntheticC0Plan.overrides;
	const syntheticBaseline = await checkPlan(repositoryRoot, syntheticOverrides, syntheticAuthority);
	if (syntheticBaseline.length > 0) {
		throw new Error(`P07B-C C0 plan checker synthetic C0B baseline failed:\n${syntheticBaseline.join("\n")}`);
	}
	const terminalC0Plan = await syntheticC0ReceiptPlanFixture(c3ReceiptResult.present);
	const terminalC0Handoff = terminalC0Plan.overrides.get("docs/HANDOFF_MODE_C.md");
	const c3PresentHandoff = c3ReceiptResult.present.get("docs/HANDOFF_MODE_C.md");
	const c3PresentBlock = parseC3ReceiptBlock(c3PresentHandoff).block;
	if (c3PresentBlock === undefined || terminalC0Plan.priorHandoff !== stripC0FixtureBlock(c3PresentHandoff) ||
		stripC0FixtureBlock(terminalC0Handoff) !== terminalC0Plan.priorHandoff ||
		parseC3ReceiptBlock(terminalC0Handoff).block !== c3PresentBlock) {
		throw new Error("P07B-C C0-over-C3 actual fixture inverse or terminal preservation control failed");
	}
	const terminalC0Errors = await checkPlan(
		repositoryRoot,
		terminalC0Plan.overrides,
		terminalC0Plan.authority,
		undefined,
		undefined,
		undefined,
		c3ReceiptResult.fixture.authority,
		c3ReceiptResult.fixture.c3rAuthority,
		c3ReceiptResult.fixture.c3qAuthority,
		c3ReceiptResult.fixture.c3tAuthority,
		c3ReceiptResult.fixture.c3uAuthority,
	);
	if (terminalC0Errors.length > 0) {
		throw new Error(`P07B-C C0-over-C3 actual fixture full-plan control failed:\n${terminalC0Errors.join("\n")}`);
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
	const c1FixturePattern = /\n?<!-- P07B-C-C1-SOURCE-RECEIPTS:START -->[\s\S]*?<!-- P07B-C-C1-SOURCE-RECEIPTS:END -->\n?/gu;
	const stripC1FixtureBlock = (body) => {
		const startCount = countOccurrences(body, "<!-- P07B-C-C1-SOURCE-RECEIPTS:START -->");
		const endCount = countOccurrences(body, "<!-- P07B-C-C1-SOURCE-RECEIPTS:END -->");
		if (startCount !== endCount || startCount > 1) {
			throw new Error(`P07B-C C1 fixture receipt marker cardinality start=${startCount} end=${endCount}`);
		}
		return startCount === 0 ? body : body.replace(c1FixturePattern, "\n");
	};
	const syntheticC1ReceiptPlanFixture = async (baseOverrides = new Map()) => {
		const priorHandoff = stripC1FixtureBlock(await readText(repositoryRoot, "docs/HANDOFF_MODE_C.md", baseOverrides));
		const handoff = insertLegacyReceiptBlockBeforeOptionalTerminalC3(
			priorHandoff,
			c1ReceiptBlock,
			"C1 synthetic receipt fixture",
			"trimmed-blank",
		);
		return Object.freeze({
			overrides: new Map([
				...baseOverrides,
				[c1StatusPath, syntheticC1Status],
				[c1ReceiptDeclarationPath, syntheticC1ReceiptText],
				["docs/HANDOFF_MODE_C.md", handoff],
				[c1DidrunBugsPath, syntheticC1DidrunBugs],
			]),
			authority: syntheticC1Authority,
			priorHandoff,
		});
	};
	const syntheticC1Plan = await syntheticC1ReceiptPlanFixture();
	const syntheticC1Handoff = syntheticC1Plan.overrides.get("docs/HANDOFF_MODE_C.md");
	const syntheticC1Overrides = syntheticC1Plan.overrides;
	const syntheticC1Baseline = await checkPlan(
		repositoryRoot,
		syntheticC1Overrides,
		undefined,
		syntheticC1Authority,
	);
	if (syntheticC1Baseline.length > 0) {
		throw new Error(`P07B-C C1 plan checker synthetic C1B baseline failed:\n${syntheticC1Baseline.join("\n")}`);
	}
	const terminalC1Plan = await syntheticC1ReceiptPlanFixture(c3ReceiptResult.present);
	const terminalC1Handoff = terminalC1Plan.overrides.get("docs/HANDOFF_MODE_C.md");
	if (terminalC1Plan.priorHandoff !== stripC1FixtureBlock(c3PresentHandoff) ||
		stripC1FixtureBlock(terminalC1Handoff) !== terminalC1Plan.priorHandoff ||
		parseC3ReceiptBlock(terminalC1Handoff).block !== c3PresentBlock) {
		throw new Error("P07B-C C1-over-C3 actual fixture inverse or terminal preservation control failed");
	}
	const terminalC1Errors = await checkPlan(
		repositoryRoot,
		terminalC1Plan.overrides,
		undefined,
		terminalC1Plan.authority,
		undefined,
		undefined,
		c3ReceiptResult.fixture.authority,
		c3ReceiptResult.fixture.c3rAuthority,
		c3ReceiptResult.fixture.c3qAuthority,
		c3ReceiptResult.fixture.c3tAuthority,
		c3ReceiptResult.fixture.c3uAuthority,
	);
	if (terminalC1Errors.length > 0) {
		throw new Error(`P07B-C C1-over-C3 actual fixture full-plan control failed:\n${terminalC1Errors.join("\n")}`);
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
			name: "C3V persistent cache acceptance", path: c3vStatusPath,
			value: currentC3VStatus.replace(c3vDispositionRows[0], c3vDispositionRows[0].replace("`DECLINED_WITH_REASON`", "`INTENTIONAL`")),
			expect: "missing required C3V ruling",
		},
		{
			name: "C3V p8 acceptance", path: c3vStatusPath,
			value: currentC3VStatus.replace(c3vDispositionRows[1], c3vDispositionRows[1].replace("`INTENTIONAL; P8_DECLINED_WITH_REASON`", "`INTENTIONAL; P8_ACCEPTED`")),
			expect: "missing required C3V ruling",
		},
		{
			name: "C3V automatic stale lock deletion", path: c3vStatusPath,
			value: currentC3VStatus.replace("automatic stale deletion remains declined", "automatic stale deletion is enabled"),
			expect: "missing required C3V ruling",
		},
		{
			name: "C3V receipt-tier disposition drift", path: c3vStatusPath,
			value: currentC3VStatus.replace(c3vDispositionRows[3], c3vDispositionRows[3].replace("`RECEIPT_RECONCILIATION` already admits only", "`SOURCE_FULL` now admits")),
			expect: "missing required C3V ruling",
		},
		{
			name: "C3V new qualification overclaim", path: c3vStatusPath,
			value: currentC3VStatus.replace("may not claim a new qualification or a new performance result", "claims a new qualification and performance result"),
			expect: "missing required C3V ruling",
		},
		{
			name: "C3V didrun interrupt finding removal", path: c3vStatusPath,
			value: currentC3VStatus.replace(c3vDidrunInterruptFinding, "Interrupted commands are fully receipted."),
			expect: "missing required C3V ruling",
		},
		{
			name: "C3V planned claim label drift", path: c3vStatusPath,
			value: currentC3VStatus.replace(c3vClaimLabels[0], `${c3vClaimLabels[0]} altered`),
			expect: "C3V planned claim map",
		},
		{
			name: "C3M historical handoff regression", path: c3MaintenanceStatusPath,
			value: currentC3MaintenanceStatus.replace(
				"The checker must never reopen the C3P-era handoff.",
				"The checker may reopen the C3P-era handoff.",
			),
			expect: "missing required C3M ruling",
		},
		{
			name: "C3M carried-authority survival removal", path: c3MaintenanceStatusPath,
			value: currentC3MaintenanceStatus.replace(
				"C3V and C3M authority survives the absent-phase reconstruction.",
				"Interposed authority may be discarded.",
			),
			expect: "missing required C3M ruling",
		},
		{
			name: "C3M intended claim label drift", path: c3MaintenanceStatusPath,
			value: currentC3MaintenanceStatus.replace(c3MaintenanceClaimLabels[0], `${c3MaintenanceClaimLabels[0]} altered`),
			expect: "C3M intended claim map",
		},
		{
			name: "C3A predecessor identity removal", path: c3aStatusPath,
			value: currentC3AStatus.replace(sealedC3PBIdentity.commit, "0".repeat(40)),
			expect: "missing required C3A ruling",
		},
		{
			name: "C3A runtime-bound ruling removal", path: c3aStatusPath,
			value: currentC3AStatus.replace("128-byte runtime-version ceiling", "unbounded runtime version"),
			expect: "missing required C3A ruling",
		},
		{
			name: "C3A intended claim label drift", path: c3aStatusPath,
			value: currentC3AStatus.replace(c3aClaimLabels[0], `${c3aClaimLabels[0]} altered`),
			expect: "C3A intended claim map",
		},
		{
			name: "C3L active boundary drift", path: c3lStatusPath,
			value: currentC3LStatus.replace(c3lStatusBoundaryLine, "- **Boundary:** sealed."),
			expect: "C3L active boundary",
		},
		{
			name: "C3L predecessor identity removal", path: c3lStatusPath,
			value: currentC3LStatus.replace(sealedC3AIdentity.commit, "0".repeat(40)),
			expect: "C3L sealed parent identity",
		},
		{
			name: "C3L unreceipted state drift", path: c3lStatusPath,
			value: currentC3LStatus.replace(c3lStatusUnreceiptedLine, "C3L is partially receipted."),
			expect: "C3L unreceipted state",
		},
		{
			name: "C3L conditional-admission ruling removal", path: c3lStatusPath,
			value: currentC3LStatus.replace("U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED", "U6_C3_EDGE_ACCEPTED"),
			expect: "missing required C3L ruling",
		},
		{
			name: "C3L intended claim label drift", path: c3lStatusPath,
			value: currentC3LStatus.replace(c3lClaimLabels[0], `${c3lClaimLabels[0]} altered`),
			expect: "C3L intended claim map",
		},
		{
			name: "C3F active boundary drift", path: c3fStatusPath,
			value: currentC3FStatus.replace(c3fStatusBoundaryLine, "- **Boundary:** sealed."),
			expect: "C3F active boundary",
		},
		{
			name: "C3F predecessor identity removal", path: c3fStatusPath,
			value: currentC3FStatus.replace(sealedC3LIdentity.commit, "0".repeat(40)),
			expect: "C3F sealed parent identity",
		},
		{
			name: "C3F predecessor note identity removal", path: c3fStatusPath,
			value: currentC3FStatus.replace(sealedC3LIdentity.noteBlob, "0".repeat(40)),
			expect: "C3F sealed parent identity",
		},
		{
			name: "C3F unreceipted state drift", path: c3fStatusPath,
			value: currentC3FStatus.replace(c3fStatusUnreceiptedLine, "C3F is partially receipted."),
			expect: "C3F unreceipted state",
		},
		{
			name: "C3F all-or-none ruling removal", path: c3fStatusPath,
			value: currentC3FStatus.replace("The bundle is admitted only when all four exact pairs are present", "Any subset is admitted"),
			expect: "missing required C3F ruling",
		},
		{
			name: "C3F intended claim label drift", path: c3fStatusPath,
			value: currentC3FStatus.replace(c3fClaimLabels[0], `${c3fClaimLabels[0]} altered`),
			expect: "C3F intended claim map",
		},
		{
			name: "C3S active boundary drift", path: c3sStatusPath,
			value: currentC3SStatus.replace(c3sStatusBoundaryLine, "- **Boundary:** sealed."),
			expect: "C3S active boundary",
		},
		{
			name: "C3S predecessor identity removal", path: c3sStatusPath,
			value: currentC3SStatus.replace(sealedC3FIdentity.commit, "0".repeat(40)),
			expect: "C3S sealed parent identity",
		},
		{
			name: "C3S predecessor note identity removal", path: c3sStatusPath,
			value: currentC3SStatus.replace(sealedC3FIdentity.noteBlob, "0".repeat(40)),
			expect: "C3S sealed parent identity",
		},
		{
			name: "C3S unreceipted state drift", path: c3sStatusPath,
			value: currentC3SStatus.replace(c3sStatusUnreceiptedLine, "C3S is partially receipted."),
			expect: "C3S unreceipted state",
		},
		{
			name: "C3S phase-stable fixture ruling removal", path: c3sStatusPath,
			value: currentC3SStatus.replace("The U6 self-test now accepts exactly two starting phases:", "The U6 self-test has no fixed starting phases:"),
			expect: "missing required C3S ruling",
		},
		{
			name: "C3S intended claim label drift", path: c3sStatusPath,
			value: currentC3SStatus.replace(c3sClaimLabels[0], `${c3sClaimLabels[0]} altered`),
			expect: "C3S intended claim map",
		},
		{
			name: "C3R active boundary drift", path: c3rStatusPath,
			value: currentC3RStatus.replace(c3rStatusBoundaryLine, "- **Boundary:** sealed."),
			expect: "C3R active boundary",
		},
		{
			name: "C3R predecessor identity removal", path: c3rStatusPath,
			value: currentC3RStatus.replace(sealedC3Identity.commit, "0".repeat(40)),
			expect: "C3R sealed parent identity",
		},
		{
			name: "C3R predecessor note identity removal", path: c3rStatusPath,
			value: currentC3RStatus.replace(sealedC3Identity.noteBlob, "0".repeat(40)),
			expect: "C3R sealed parent identity",
		},
		{
			name: "C3R subject drift", path: c3rStatusPath,
			value: currentC3RStatus.replace(c3rSourceSubject, `${c3rSourceSubject} altered`),
			expect: "C3R commit subject",
		},
		{
			name: "C3R unreceipted state drift", path: c3rStatusPath,
			value: currentC3RStatus.replace(c3rStatusUnreceiptedLine, "C3R is partially receipted."),
			expect: "C3R unreceipted state",
		},
		{
			name: "C3R seal outcome policy removal", path: c3rStatusPath,
			value: currentC3RStatus.replace(
				"C3R's operator seal procedure is outcome-dependent: run the plain seal first.",
				"C3R's seal protocol is unspecified.",
			),
			expect: "missing required C3R ruling",
		},
		{
			name: "C3R intended claim label drift", path: c3rStatusPath,
			value: currentC3RStatus.replace(c3rClaimLabels[0], `${c3rClaimLabels[0]} altered`),
			expect: "C3R intended claim map",
		},
		{
			name: "C3R self receipt", path: c3rStatusPath,
			value: `${currentC3RStatus}\nC3R is sealed and strict-clean.\n`,
			expect: "C3R self-receipt",
		},
		{
			name: "C3Q active boundary drift", path: c3qStatusPath,
			value: currentC3QStatus.replace(c3qStatusBoundaryLine, "- **Boundary:** sealed."),
			expect: "C3Q active boundary",
		},
		{
			name: "C3Q predecessor identity removal", path: c3qStatusPath,
			value: currentC3QStatus.replace(sealedC3RIdentity.commit, "0".repeat(40)),
			expect: "C3Q sealed parent identity",
		},
		{
			name: "C3Q predecessor note identity removal", path: c3qStatusPath,
			value: currentC3QStatus.replace(sealedC3RIdentity.noteBlob, "0".repeat(40)),
			expect: "C3Q sealed parent identity",
		},
		{
			name: "C3Q subject drift", path: c3qStatusPath,
			value: currentC3QStatus.replace(c3qSourceSubject, `${c3qSourceSubject} altered`),
			expect: "C3Q commit subject",
		},
		{
			name: "C3Q unreceipted state drift", path: c3qStatusPath,
			value: currentC3QStatus.replace(c3qStatusUnreceiptedLine, "C3Q is partially receipted."),
			expect: "C3Q unreceipted state",
		},
		{
			name: "C3Q intended claim label drift", path: c3qStatusPath,
			value: currentC3QStatus.replace(c3qClaimLabels[0], `${c3qClaimLabels[0]} altered`),
			expect: "C3Q intended claim map",
		},
		{
			name: "C3Q self receipt", path: c3qStatusPath,
			value: `${currentC3QStatus}\nC3Q is sealed and strict-clean.\n`,
			expect: "C3Q self-receipt",
		},
		{
			name: "C3Q status HTML-commented authority", path: c3qStatusPath,
			value: `<!--\n${currentC3QStatus}-->\n`,
			expect: "missing required C3Q ruling",
		},
		{
			name: "C3Q status fenced authority", path: c3qStatusPath,
			value: `\`\`\`markdown\n${currentC3QStatus}\`\`\`\n`,
			expect: "missing required C3Q ruling",
		},
		{
			name: "C3Q status indented-code authority", path: c3qStatusPath,
			value: currentC3QStatus.split("\n").map((line) => `    ${line}`).join("\n"),
			expect: "missing required C3Q ruling",
		},
		{
			name: "C3Q status raw-HTML authority", path: c3qStatusPath,
			value: `<section>\n${currentC3QStatus}</section>\n`,
			expect: "C3Q visible authority invalid",
		},
		{
			name: "C3T active boundary drift", path: c3tStatusPath,
			value: currentC3TStatus.replace(c3tStatusBoundaryLine, "- **Boundary:** sealed."),
			expect: "C3T active boundary",
		},
		{
			name: "C3T predecessor identity removal", path: c3tStatusPath,
			value: currentC3TStatus.replace(sealedC3QIdentity.commit, "0".repeat(40)),
			expect: "C3T sealed parent identity",
		},
		{
			name: "C3T predecessor note identity removal", path: c3tStatusPath,
			value: currentC3TStatus.replace(sealedC3QIdentity.noteBlob, "0".repeat(40)),
			expect: "C3T sealed parent identity",
		},
		{
			name: "C3T subject drift", path: c3tStatusPath,
			value: currentC3TStatus.replace(c3tSourceSubject, `${c3tSourceSubject} altered`),
			expect: "C3T commit subject",
		},
		{
			name: "C3T unreceipted state drift", path: c3tStatusPath,
			value: currentC3TStatus.replace(c3tStatusUnreceiptedLine, "C3T is partially receipted."),
			expect: "C3T unreceipted state",
		},
		{
			name: "C3T intended claim label drift", path: c3tStatusPath,
			value: currentC3TStatus.replace(c3tClaimLabels[0], `${c3tClaimLabels[0]} altered`),
			expect: "C3T intended claim map",
		},
		{
			name: "C3T self receipt", path: c3tStatusPath,
			value: `${currentC3TStatus}\nC3T is sealed and strict-clean.\n`,
			expect: "C3T self-receipt",
		},
		{
			name: "C3T status HTML-commented authority", path: c3tStatusPath,
			value: `<!--\n${currentC3TStatus}-->\n`,
			expect: "missing required C3T ruling",
		},
		{
			name: "C3T status fenced authority", path: c3tStatusPath,
			value: `\`\`\`markdown\n${currentC3TStatus}\`\`\`\n`,
			expect: "missing required C3T ruling",
		},
		{
			name: "C3T status raw-HTML authority", path: c3tStatusPath,
			value: `<section>\n${currentC3TStatus}</section>\n`,
			expect: "C3T visible authority invalid",
		},
		{
			name: "C3U active boundary drift", path: c3uStatusPath,
			value: currentC3UStatus.replace(c3uStatusBoundaryLine, "- **Boundary:** sealed."),
			expect: "C3U active boundary",
		},
		{
			name: "C3U predecessor identity removal", path: c3uStatusPath,
			value: currentC3UStatus.replace(sealedC3TIdentity.commit, "0".repeat(40)),
			expect: "C3U sealed parent identity",
		},
		{
			name: "C3U predecessor note identity removal", path: c3uStatusPath,
			value: currentC3UStatus.replace(sealedC3TIdentity.noteBlob, "0".repeat(40)),
			expect: "C3U sealed parent identity",
		},
		{
			name: "C3U subject drift", path: c3uStatusPath,
			value: currentC3UStatus.replace(c3uSourceSubject, `${c3uSourceSubject} altered`),
			expect: "C3U commit subject",
		},
		{
			name: "C3U unreceipted state drift", path: c3uStatusPath,
			value: currentC3UStatus.replace(c3uStatusUnreceiptedLine, "C3U is partially receipted."),
			expect: "C3U unreceipted state",
		},
		{
			name: "C3U fixture ruling removal", path: c3uStatusPath,
			value: currentC3UStatus.replace(
				"The actual C0, C1, C2, and C3P fixture callers each run as a full-plan terminal-C3 control",
				"The terminal-C3 fixture caller controls are omitted",
			),
			expect: "missing required C3U ruling",
		},
		{
			name: "C3U supplemental matrix accounting removal", path: c3uStatusPath,
			value: currentC3UStatus.replace(
				"Eight supplemental natural aliases must reject, while two explicit strict-zero prohibition controls must remain accepted.",
				"Supplemental matrix accounting omitted.",
			),
			expect: "missing required C3U ruling",
		},
		{
			name: "C3U intended claim label drift", path: c3uStatusPath,
			value: currentC3UStatus.replace(c3uClaimLabels[0], `${c3uClaimLabels[0]} altered`),
			expect: "C3U intended claim map",
		},
		{
			name: "C3U self receipt", path: c3uStatusPath,
			value: `${currentC3UStatus}\nC3U is sealed and strict-clean.\n`,
			expect: "C3U self-receipt",
		},
		{
			name: "C3U status HTML-commented authority", path: c3uStatusPath,
			value: `<!--\n${currentC3UStatus}-->\n`,
			expect: "missing required C3U ruling",
		},
		{
			name: "C3U status fenced authority", path: c3uStatusPath,
			value: `\`\`\`markdown\n${currentC3UStatus}\`\`\`\n`,
			expect: "missing required C3U ruling",
		},
		{
			name: "C3U status indented-code authority", path: c3uStatusPath,
			value: currentC3UStatus.split("\n").map((line) => `    ${line}`).join("\n"),
			expect: "missing required C3U ruling",
		},
		{
			name: "C3U status raw-HTML authority", path: c3uStatusPath,
			value: `<section>\n${currentC3UStatus}</section>\n`,
			expect: "C3U visible authority invalid",
		},
		...Object.entries(c3vFrozenVerifierDigests).map(([path, digest]) => ({
			name: `C3V historical verifier digest removal: ${path}`,
			path: c3vStatusPath,
			value: currentC3VStatus.replace(`\`${path}\` is \`sha256:${digest}\``, `\`${path}\` digest omitted`),
			expect: "missing required C3V ruling",
		})),
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
			value: currentPromptPack.replace("C3 exact-target publication active", "C3 exact-target publication omitted"),
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
				name: "C3V unit removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					delete candidate.units.C3V;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3V unit order drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					const entries = Object.entries(candidate.units);
					const c3vIndex = entries.findIndex(([unit]) => unit === "C3V");
					const c3mIndex = entries.findIndex(([unit]) => unit === "C3M");
					[entries[c3vIndex], entries[c3mIndex]] = [entries[c3mIndex], entries[c3vIndex]];
					candidate.units = Object.fromEntries(entries);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3V exact path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3V.exact = candidate.units.C3V.exact.filter((path) => path !== c3vStatusPath);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3V exact path roster mismatch",
			},
			{
				name: "C3V prefix introduction", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3V.prefixes = ["docs/"];
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3V receipt profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3V.verification_profile = "RECEIPT_RECONCILIATION";
					candidate.units.C3V.receipt_claims = structuredClone(requiredC3PBReceiptClaims);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3M unit removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					delete candidate.units.C3M;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3M unit order drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					const entries = Object.entries(candidate.units);
					const c3mIndex = entries.findIndex(([unit]) => unit === "C3M");
					const c3pbIndex = entries.findIndex(([unit]) => unit === "C3PB");
					[entries[c3mIndex], entries[c3pbIndex]] = [entries[c3pbIndex], entries[c3mIndex]];
					candidate.units = Object.fromEntries(entries);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3M exact path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3M.exact = candidate.units.C3M.exact.filter((path) => path !== c3MaintenanceStatusPath);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3M exact path roster mismatch",
			},
			{
				name: "C3M prefix introduction", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3M.prefixes = ["docs/"];
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3M receipt profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3M.verification_profile = "RECEIPT_RECONCILIATION";
					candidate.units.C3M.receipt_claims = structuredClone(requiredC3PBReceiptClaims);
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
				name: "C3A unit removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					delete candidate.units.C3A;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3A unit order drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					const entries = Object.entries(candidate.units);
					const c3aIndex = entries.findIndex(([unit]) => unit === "C3A");
					const c3Index = entries.findIndex(([unit]) => unit === "C3");
					[entries[c3aIndex], entries[c3Index]] = [entries[c3Index], entries[c3aIndex]];
					candidate.units = Object.fromEntries(entries);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3A exact path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3A.exact = candidate.units.C3A.exact.filter((path) => path !== c3aStatusPath);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3A exact path roster mismatch",
			},
			{
				name: "C3A prefix introduction", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3A.prefixes = ["docs/"];
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3A receipt profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3A.verification_profile = "RECEIPT_RECONCILIATION";
					candidate.units.C3A.receipt_claims = structuredClone(requiredC3PBReceiptClaims);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3L unit removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					delete candidate.units.C3L;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3L unit order drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					const entries = Object.entries(candidate.units);
					const c3lIndex = entries.findIndex(([unit]) => unit === "C3L");
					const c3Index = entries.findIndex(([unit]) => unit === "C3");
					[entries[c3lIndex], entries[c3Index]] = [entries[c3Index], entries[c3lIndex]];
					candidate.units = Object.fromEntries(entries);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3L exact path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3L.exact = candidate.units.C3L.exact.filter((path) => path !== c3lStatusPath);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3L exact path roster mismatch",
			},
			{
				name: "C3L prefix introduction", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3L.prefixes = ["tools/"];
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3L receipt profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3L.verification_profile = "RECEIPT_RECONCILIATION";
					candidate.units.C3L.receipt_claims = structuredClone(requiredC3PBReceiptClaims);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3F unit removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					delete candidate.units.C3F;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3F unit order drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					const entries = Object.entries(candidate.units);
					const c3fIndex = entries.findIndex(([unit]) => unit === "C3F");
					const c3Index = entries.findIndex(([unit]) => unit === "C3");
					[entries[c3fIndex], entries[c3Index]] = [entries[c3Index], entries[c3fIndex]];
					candidate.units = Object.fromEntries(entries);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3F exact path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3F.exact = candidate.units.C3F.exact.filter((path) => path !== c3fStatusPath);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3F exact path roster mismatch",
			},
			{
				name: "C3F prefix introduction", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3F.prefixes = ["tools/"];
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3F receipt profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3F.verification_profile = "RECEIPT_RECONCILIATION";
					candidate.units.C3F.receipt_claims = structuredClone(requiredC3PBReceiptClaims);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3S unit removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					delete candidate.units.C3S;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3S unit order drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					const entries = Object.entries(candidate.units);
					const c3sIndex = entries.findIndex(([unit]) => unit === "C3S");
					const c3Index = entries.findIndex(([unit]) => unit === "C3");
					[entries[c3sIndex], entries[c3Index]] = [entries[c3Index], entries[c3sIndex]];
					candidate.units = Object.fromEntries(entries);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3S exact path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3S.exact = candidate.units.C3S.exact.filter((path) => path !== c3sStatusPath);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3S exact path roster mismatch",
			},
			{
				name: "C3S prefix introduction", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3S.prefixes = ["tools/"];
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3S receipt profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3S.verification_profile = "RECEIPT_RECONCILIATION";
					candidate.units.C3S.receipt_claims = structuredClone(requiredC3PBReceiptClaims);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
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
				expect: "invalid",
			},
			{
				name: "C3R unit removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					delete candidate.units.C3R;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3R unit order drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					const entries = Object.entries(candidate.units);
					const c3rIndex = entries.findIndex(([unit]) => unit === "C3R");
					const c3bIndex = entries.findIndex(([unit]) => unit === "C3B");
					[entries[c3rIndex], entries[c3bIndex]] = [entries[c3bIndex], entries[c3rIndex]];
					candidate.units = Object.fromEntries(entries);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3R exact path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3R.exact = candidate.units.C3R.exact.filter((path) => path !== c3rStatusPath);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3R: frozen exact unit contract",
			},
			{
				name: "C3R prefix introduction", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3R.prefixes = ["tools/"];
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3R receipt profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3R.verification_profile = "RECEIPT_RECONCILIATION";
					candidate.units.C3R.receipt_claims = structuredClone(requiredC3BReceiptClaims);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3Q unit removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					delete candidate.units.C3Q;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3R C3Q unit order drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					const entries = Object.entries(candidate.units);
					const c3rIndex = entries.findIndex(([unit]) => unit === "C3R");
					const c3qIndex = entries.findIndex(([unit]) => unit === "C3Q");
					[entries[c3rIndex], entries[c3qIndex]] = [entries[c3qIndex], entries[c3rIndex]];
					candidate.units = Object.fromEntries(entries);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3Q C3T unit order drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					const entries = Object.entries(candidate.units);
					const c3qIndex = entries.findIndex(([unit]) => unit === "C3Q");
					const c3tIndex = entries.findIndex(([unit]) => unit === "C3T");
					[entries[c3qIndex], entries[c3tIndex]] = [entries[c3tIndex], entries[c3qIndex]];
					candidate.units = Object.fromEntries(entries);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3Q exact path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3Q.exact = candidate.units.C3Q.exact.filter((path) => path !== c3qStatusPath);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3Q: frozen exact unit contract",
			},
			{
				name: "C3Q prefix introduction", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3Q.prefixes = ["tools/"];
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3Q receipt profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3Q.verification_profile = "RECEIPT_RECONCILIATION";
					candidate.units.C3Q.receipt_claims = structuredClone(requiredC3BReceiptClaims);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3T unit removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					delete candidate.units.C3T;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3T C3U unit order drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					const entries = Object.entries(candidate.units);
					const c3tIndex = entries.findIndex(([unit]) => unit === "C3T");
					const c3uIndex = entries.findIndex(([unit]) => unit === "C3U");
					[entries[c3tIndex], entries[c3uIndex]] = [entries[c3uIndex], entries[c3tIndex]];
					candidate.units = Object.fromEntries(entries);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3T exact path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3T.exact = candidate.units.C3T.exact.filter((path) => path !== c3tStatusPath);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3T: frozen exact unit contract",
			},
			{
				name: "C3T prefix introduction", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3T.prefixes = ["tools/"];
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3T receipt profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3T.verification_profile = "RECEIPT_RECONCILIATION";
					candidate.units.C3T.receipt_claims = structuredClone(requiredC3BReceiptClaims);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3U unit removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					delete candidate.units.C3U;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3U C3B unit order drift", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					const entries = Object.entries(candidate.units);
					const c3uIndex = entries.findIndex(([unit]) => unit === "C3U");
					const c3bIndex = entries.findIndex(([unit]) => unit === "C3B");
					[entries[c3uIndex], entries[c3bIndex]] = [entries[c3bIndex], entries[c3uIndex]];
					candidate.units = Object.fromEntries(entries);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3U exact path removal", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3U.exact = candidate.units.C3U.exact.filter((path) => path !== c3uStatusPath);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "C3U: frozen exact unit contract",
			},
			{
				name: "C3U prefix introduction", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3U.prefixes = ["tools/"];
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3U receipt profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3U.verification_profile = "RECEIPT_RECONCILIATION";
					candidate.units.C3U.receipt_claims = structuredClone(requiredC3BReceiptClaims);
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
			},
			{
				name: "C3B source profile substitution", path: configPath,
				value: (() => {
					const candidate = structuredClone(currentConfiguration);
					candidate.units.C3B.verification_profile = "SOURCE_FULL";
					delete candidate.units.C3B.receipt_claims;
					return `${JSON.stringify(candidate, null, 2)}\n`;
				})(),
				expect: "invalid",
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
			expect: "C1B self-receipt forbidden",
		},
		{
			name: "C1B handoff self-strict claim",
			overrides: mutateC1Overrides(
				"docs/HANDOFF_MODE_C.md",
				`${syntheticC1Handoff}\nC1B strict exit: \`0\`; 7/7 claims recorded-exact.\n`,
			),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "C1B self-receipt forbidden",
		},
		{
			name: "C1B hash-free self-grade claim",
			overrides: mutateC1Overrides(
				c1DidrunBugsPath,
				`${syntheticC1DidrunBugs}\nC1B is sealed and strict-clean.\n`,
			),
			c1ReceiptAuthority: syntheticC1Authority,
			expect: "C1B self-receipt forbidden",
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

	console.log(`P07B-C evolved plan checker self-test passed: ${cases.length + localEvidenceMutations + receiptModePhaseMutations + c2ReceiptMutations + c2SealedSourcePhaseMutations + c2LocalEvidenceMutations + c3pReceiptMutations + c3ReceiptMutations + activeSelfReceiptMatrix.rejected} authority, structure, C1/C2/C3P/C3 local-evidence, phase-isolation, semantic-projection, receipt, and allowlist mutations rejected; active-unit matrix phases=${activeSelfReceiptMatrix.phases} active-phases=C3U,C3B paths=${activeSelfReceiptMatrix.paths} forms=${activeSelfReceiptMatrix.forms} rejected=${activeSelfReceiptMatrix.rejected} positive-controls=${activeSelfReceiptMatrix.controls} supplemental-aliases=${activeSelfReceiptMatrix.supplementalAliases} supplemental-controls=${activeSelfReceiptMatrix.supplementalControls}; result=positive-form refusal only, not a general Markdown or security audit`);
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

async function verifyLivePresealLedger({ phase, expectedArgv, claims, receiptPresent, receiptStates, hermeticArgvPrefix }) {
	if (hermeticArgvPrefix !== undefined) {
		requireHermeticCommandPlan(phase, expectedArgv, hermeticArgvPrefix);
		requireEffectiveHermeticContext(phase, hermeticArgvPrefix, process.cwd(), process.env);
	}
	const expectedInvocation = hermeticArgvPrefix === undefined
		? expectedArgv.at(-1)
		: expectedArgv.at(-1).slice(hermeticArgvPrefix.length);
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
	const declarations = receiptStates ?? [{ path: c3pReceiptDeclarationPath, present: receiptPresent }];
	for (const declaration of declarations) {
		await requireReceiptDeclarationPhase(repositoryRoot, declaration.path, declaration.present, phase);
	}
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
	let wrapperFingerprint;
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
			(hermeticArgvPrefix === undefined && !isDeepStrictEqual(event.argv, expectedArgv[index]))) {
			throw new Error(`P07B-C ${phase} live event ${index} authority mismatch`);
		}
		if (hermeticArgvPrefix !== undefined) {
			wrapperFingerprint = requireBoundPresealEventContext(
				phase, event, expectedArgv[index], wrapperFingerprint,
			);
		}
		tree ??= event.tree_before;
		if (event.tree_before !== tree) throw new Error(`P07B-C ${phase} live event ${index} tree mismatch`);
		const claim = recordedClaims[index];
		if (!claim || typeof claim !== "object" || Array.isArray(claim) ||
			!isDeepStrictEqual(Object.keys(claim).sort(), ["ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) ||
			claim.label !== claims[index].label || claim.ctype !== claims[index].type ||
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
	const git = spawnSync("/usr/bin/git", ["--no-replace-objects", "write-tree"], {
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

export async function verifyC3VPresealLedger() {
	await verifyLivePresealLedger({
		phase: "C3V",
		expectedArgv: c3vExpectedClaimArgv,
		claims: c3vClaimLabels.map((label, index) => ({ label, type: c3vClaimTypes[index] })),
		receiptPresent: false,
	});
}

export async function verifyC3MaintenancePresealLedger() {
	await verifyLivePresealLedger({
		phase: "C3M",
		expectedArgv: c3MaintenanceExpectedClaimArgv,
		claims: c3MaintenanceClaimLabels.map((label, index) => ({ label, type: c3MaintenanceClaimTypes[index] })),
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

export async function verifyC3APresealLedger() {
	await requirePrivateC3AFinalRunDirectories();
	await verifyLivePresealLedger({
		phase: "C3A",
		expectedArgv: c3aExpectedClaimArgv,
		claims: c3aClaimLabels.map((label, index) => ({ label, type: c3aClaimTypes[index] })),
		receiptPresent: true,
		hermeticArgvPrefix: c3aHermeticArgvPrefix,
	});
}

function requireSealedC3APredecessorAuthority() {
	const environment = {
		HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1",
		GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
		GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
	};
	const run = (args) => {
		const result = spawnSync("/usr/bin/git", ["--no-replace-objects", ...args], {
			cwd: repositoryRoot, encoding: "buffer", timeout: 30_000, maxBuffer: 4 * 1024 * 1024, env: environment,
		});
		if (result.error || result.signal || result.status !== 0 || (result.stderr?.length ?? 0) !== 0) {
			throw new Error(`P07B-C C3L sealed C3A predecessor Git authority failed for ${args.join(" ")} (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
		}
		return result.stdout ?? Buffer.alloc(0);
	};
	const line = (args) => decodeExactGitLine(run(args), `C3L predecessor git ${args[0]} stdout`);
	const snapshot = {
		headCommit: line(["rev-parse", "--verify", "HEAD^{commit}"]),
		headTree: line(["rev-parse", "--verify", "HEAD^{tree}"]),
		noteBlob: line(["notes", "--ref=didrun", "list", sealedC3AIdentity.commit]),
		noteBody: decodeGitUTF8(run(["notes", "--ref=didrun", "show", sealedC3AIdentity.commit]), "sealed C3A note body"),
	};
	requireSealedC3APredecessorSnapshot("C3L", snapshot);
	console.log(`P07B-C C3L sealed C3A predecessor exact: commit=${snapshot.headCommit} tree=${snapshot.headTree} note=${snapshot.noteBlob} claims=${c3aClaimLabels.length}`);
}

export async function verifyC3LPresealLedger() {
	await requirePrivateC3LFinalRunDirectories();
	requireSealedC3APredecessorAuthority();
	await verifyLivePresealLedger({
		phase: "C3L",
		expectedArgv: c3lExpectedClaimArgv,
		claims: c3lClaimLabels.map((label, index) => ({ label, type: c3lClaimTypes[index] })),
		receiptPresent: true,
		hermeticArgvPrefix: c3lHermeticArgvPrefix,
	});
}

function requireSealedC3LPredecessorAuthority() {
	const environment = {
		HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1",
		GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
		GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
	};
	const run = (args) => {
		const result = spawnSync("/usr/bin/git", ["--no-replace-objects", ...args], {
			cwd: repositoryRoot, encoding: "buffer", timeout: 30_000, maxBuffer: 4 * 1024 * 1024, env: environment,
		});
		if (result.error || result.signal || result.status !== 0 || (result.stderr?.length ?? 0) !== 0) {
			throw new Error(`P07B-C C3F sealed C3L predecessor Git authority failed for ${args.join(" ")} (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
		}
		return result.stdout ?? Buffer.alloc(0);
	};
	const line = (args) => decodeExactGitLine(run(args), `C3F predecessor git ${args[0]} stdout`);
	const parentLine = line(["show", "-s", "--format=%P", "HEAD"]);
	const headParents = parentLine === "" ? [] : parentLine.split(" ");
	const noteBlob = line(["notes", `--ref=${sealedC3LIdentity.noteRef}`, "list", sealedC3LIdentity.commit]);
	const snapshot = {
		headCommit: line(["rev-parse", "--verify", "HEAD^{commit}"]),
		headTree: line(["rev-parse", "--verify", "HEAD^{tree}"]),
		headParents,
		headSubject: line(["show", "-s", "--format=%s", "HEAD"]),
		noteBlob,
		noteType: line(["cat-file", "-t", noteBlob]),
		noteBody: decodeGitUTF8(run(["notes", `--ref=${sealedC3LIdentity.noteRef}`, "show", sealedC3LIdentity.commit]), "sealed C3L note body"),
	};
	requireSealedC3LPredecessorSnapshot("C3F", snapshot);
	console.log(`P07B-C C3F sealed C3L predecessor exact: commit=${snapshot.headCommit} tree=${snapshot.headTree} note=${snapshot.noteBlob} claims=${c3lClaimLabels.length}`);
}

export async function verifyC3FPresealLedger() {
	await requirePrivateC3FFinalRunDirectories();
	requireSealedC3LPredecessorAuthority();
	await verifyLivePresealLedger({
		phase: "C3F",
		expectedArgv: c3fExpectedClaimArgv,
		claims: c3fClaimLabels.map((label, index) => ({ label, type: c3fClaimTypes[index] })),
		receiptPresent: true,
		hermeticArgvPrefix: c3fHermeticArgvPrefix,
	});
}

function requireSealedC3FPredecessorAuthority() {
	const environment = {
		HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1",
		GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
		GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
	};
	const run = (args) => {
		const result = spawnSync("/usr/bin/git", ["--no-replace-objects", ...args], {
			cwd: repositoryRoot, encoding: "utf8", timeout: 30_000, maxBuffer: 4 * 1024 * 1024, env: environment,
		});
		if (result.error || result.signal || result.status !== 0 || result.stderr !== "") {
			throw new Error(`P07B-C C3S sealed C3F predecessor Git authority failed for ${args.join(" ")} (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
		}
		return result.stdout;
	};
	const headParents = run(["show", "-s", "--format=%P", "HEAD"]).trim().split(" ").filter(Boolean);
	const noteBlob = run(["notes", `--ref=${sealedC3FIdentity.noteRef}`, "list", sealedC3FIdentity.commit]).trim();
	const snapshot = {
		headCommit: run(["rev-parse", "--verify", "HEAD^{commit}"]).trim(),
		headTree: run(["rev-parse", "--verify", "HEAD^{tree}"]).trim(),
		headParents,
		headSubject: run(["show", "-s", "--format=%s", "HEAD"]).trimEnd(),
		noteBlob,
		noteType: run(["cat-file", "-t", noteBlob]).trim(),
		noteBody: run(["notes", `--ref=${sealedC3FIdentity.noteRef}`, "show", sealedC3FIdentity.commit]),
	};
	requireSealedC3FPredecessorSnapshot("C3S", snapshot);
	console.log(`P07B-C C3S sealed C3F predecessor exact: commit=${snapshot.headCommit} tree=${snapshot.headTree} note=${snapshot.noteBlob} claims=${c3fClaimLabels.length}`);
}

export async function verifyC3SPresealLedger() {
	await requirePrivateC3SFinalRunDirectories();
	requireSealedC3FPredecessorAuthority();
	await verifyLivePresealLedger({
		phase: "C3S",
		expectedArgv: c3sExpectedClaimArgv,
		claims: c3sClaimLabels.map((label, index) => ({ label, type: c3sClaimTypes[index] })),
		receiptPresent: true,
		hermeticArgvPrefix: c3sHermeticArgvPrefix,
	});
}

async function requireSealedC3SPredecessorAuthority() {
	let manifest;
	try {
		const manifestBytes = await readRegularNoFollow(resolve(repositoryRoot, c3PredecessorManifestPath), 2 * 1024 * 1024);
		if (!manifestBytes.equals(expectedC3PredecessorManifestBytes())) {
			throw new Error("bytes do not equal the exact canonical C3S/C3F/C3L/C3A authority manifest");
		}
		manifest = JSON.parse(decodeGitUTF8(manifestBytes, "C3 predecessor declaration"));
	} catch (error) {
		throw new Error(`P07B-C C3 predecessor declaration unreadable: ${error.message}`);
	}
	const expectedManifest = expectedC3PredecessorManifest();
	if (!isDeepStrictEqual(manifest, expectedManifest)) {
		throw new Error("P07B-C C3 predecessor declaration does not equal the exact C3S/C3F/C3L/C3A authority manifest");
	}
	const environment = {
		HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1",
		GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
		GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
	};
	const run = (args) => {
		const result = spawnSync("/usr/bin/git", ["--no-replace-objects", ...args], {
			cwd: repositoryRoot, encoding: "buffer", timeout: 30_000, maxBuffer: 4 * 1024 * 1024, env: environment,
		});
		if (result.error || result.signal || result.status !== 0 || (result.stderr?.length ?? 0) !== 0) {
			throw new Error(`P07B-C C3 predecessor Git authority failed for ${args.join(" ")} (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
		}
		return result.stdout ?? Buffer.alloc(0);
	};
	const line = (args) => decodeExactGitLine(run(args), `C3 predecessor git ${args[0]} stdout`);
	const head = line(["rev-parse", "--verify", "HEAD^{commit}"]);
	if (head !== sealedC3SIdentity.commit) {
		throw new Error(`P07B-C C3 source parent must be sealed C3S HEAD ${sealedC3SIdentity.commit}, found ${head}`);
	}
	for (const authority of c3PredecessorAuthorities) {
		const errors = [];
		const commit = authority.identity.commit;
		const tree = line(["rev-parse", "--verify", `${commit}^{tree}`]);
		const parentLine = line(["show", "-s", "--format=%P", commit]);
		const parents = parentLine === "" ? [] : parentLine.split(" ");
		const subject = line(["show", "-s", "--format=%s", commit]);
		const noteBlob = line(["notes", "--ref=didrun", "list", commit]);
		const noteType = line(["cat-file", "-t", noteBlob]);
		const noteBytes = run(["notes", "--ref=didrun", "show", commit]);
		let noteBody;
		try { noteBody = decodeGitUTF8(noteBytes, `sealed ${authority.unit} note body`); } catch (error) {
			errors.push(error.message);
		}
		if (tree !== authority.identity.tree) errors.push("tree");
		if (!isDeepStrictEqual(parents, [authority.parent])) errors.push("parent");
		if (subject !== authority.subject) errors.push("subject");
		if (noteBlob !== authority.identity.noteBlob) errors.push("note blob");
		if (noteType !== "blob") errors.push("note type");
		if (createHash("sha256").update(noteBytes).digest("hex") !== authority.identity.noteBodySHA256) {
			errors.push("note body digest");
		}
		let note;
		if (noteBody !== undefined) {
			try { note = JSON.parse(noteBody); } catch (error) { errors.push(`note JSON (${error.message})`); }
		}
		if (note !== undefined) errors.push(...authority.validateNote(note, authority.identity));
		if (errors.length > 0) {
			throw new Error(`P07B-C C3 sealed ${authority.unit} authority mismatch: ${errors.join(", ")}`);
		}
	}
	console.log(`P07B-C C3 sealed predecessor chain exact: C3S=${sealedC3SIdentity.commit} C3F=${sealedC3FIdentity.commit} C3L=${sealedC3LIdentity.commit} C3A=${sealedC3AIdentity.commit} notes=blob claims=${c3sClaimLabels.length}+${c3fClaimLabels.length}+${c3lClaimLabels.length}+${c3aClaimLabels.length}`);
}

export async function verifyC3PresealLedger() {
	await requirePrivateFinalRunDirectories("C3", c3FinalRunDirectories);
	await requireSealedC3SPredecessorAuthority();
	await verifyLivePresealLedger({
		phase: "C3",
		expectedArgv: c3ExpectedClaimArgv,
		claims: c3ClaimLabels.map((label, index) => ({ label, type: c3ClaimTypes[index] })),
		receiptStates: [
			{ path: c3pReceiptDeclarationPath, present: true },
			{ path: c3ReceiptDeclarationPath, present: false },
		],
		hermeticArgvPrefix: c3HermeticArgvPrefix,
	});
}

export async function verifyC3RPresealLedger() {
	await requirePrivateFinalRunDirectories("C3R", c3rFinalRunDirectories);
	await verifyC3RSealedC3Note();
	await verifyLivePresealLedger({
		phase: "C3R",
		expectedArgv: c3rExpectedClaimArgv,
		claims: c3rClaimLabels.map((label, index) => ({ label, type: c3rClaimTypes[index] })),
		receiptStates: [
			{ path: c3pReceiptDeclarationPath, present: true },
			{ path: c3ReceiptDeclarationPath, present: false },
		],
		hermeticArgvPrefix: c3rHermeticArgvPrefix,
	});
}

export async function verifyC3QPresealLedger() {
	await requirePrivateFinalRunDirectories("C3Q", c3qFinalRunDirectories);
	await verifyC3QSealedC3RNote();
	await verifyLivePresealLedger({
		phase: "C3Q",
		expectedArgv: c3qExpectedClaimArgv,
		claims: c3qClaimLabels.map((label, index) => ({ label, type: c3qClaimTypes[index] })),
		receiptStates: [
			{ path: c3pReceiptDeclarationPath, present: true },
			{ path: c3ReceiptDeclarationPath, present: false },
		],
		hermeticArgvPrefix: c3qHermeticArgvPrefix,
	});
}

export async function verifyC3TPresealLedger() {
	await requirePrivateFinalRunDirectories("C3T", c3tFinalRunDirectories);
	await verifyC3TSealedC3QNote();
	await verifyLivePresealLedger({
		phase: "C3T",
		expectedArgv: c3tExpectedClaimArgv,
		claims: c3tClaimLabels.map((label, index) => ({ label, type: c3tClaimTypes[index] })),
		receiptStates: [
			{ path: c3pReceiptDeclarationPath, present: true },
			{ path: c3ReceiptDeclarationPath, present: false },
		],
		hermeticArgvPrefix: c3tHermeticArgvPrefix,
	});
}

export async function verifyC3UPresealLedger() {
	await requirePrivateFinalRunDirectories("C3U", c3uFinalRunDirectories);
	await verifyC3USealedC3TNote();
	await verifyLivePresealLedger({
		phase: "C3U",
		expectedArgv: c3uExpectedClaimArgv,
		claims: c3uClaimLabels.map((label, index) => ({ label, type: c3uClaimTypes[index] })),
		receiptStates: [
			{ path: c3pReceiptDeclarationPath, present: true },
			{ path: c3ReceiptDeclarationPath, present: false },
		],
		hermeticArgvPrefix: c3uHermeticArgvPrefix,
	});
}

export async function verifyC3BPresealLedger() {
	await requirePrivateFinalRunDirectories("C3B", c3bFinalRunDirectories);
	const c3rAuthority = await loadC3RAuthorityFromGit(repositoryRoot);
	const c3rErrors = validateSealedC3RAuthority(c3rAuthority);
	if (c3rErrors.length > 0) {
		throw new Error(`P07B-C C3B sealed-C3R predecessor mismatch: ${c3rErrors.join(", ")}`);
	}
	const c3qAuthority = await loadC3QAuthorityFromGit(repositoryRoot, c3rAuthority.commit);
	const c3qErrors = validateSealedC3QAuthority(c3qAuthority);
	if (c3qErrors.length > 0) {
		throw new Error(`P07B-C C3B sealed-C3Q predecessor mismatch: ${c3qErrors.join(", ")}`);
	}
	const c3tAuthority = await loadC3TAuthorityFromGit(repositoryRoot, c3qAuthority.commit);
	const c3tErrors = validateSealedC3TAuthority(c3tAuthority);
	if (c3tErrors.length > 0) {
		throw new Error(`P07B-C C3B sealed-C3T predecessor mismatch: ${c3tErrors.join(", ")}`);
	}
	const c3uAuthority = await loadC3UAuthorityFromGit(repositoryRoot, c3tAuthority.commit);
	const c3uErrors = validateC3BPredecessorAuthority(c3uAuthority, c3tAuthority.commit);
	if (c3uErrors.length > 0) {
		throw new Error(`P07B-C C3B sealed-C3U predecessor mismatch: ${c3uErrors.join(", ")}`);
	}
	await verifyLivePresealLedger({
		phase: "C3B",
		expectedArgv: c3bExpectedClaimArgv,
		claims: c3bClaimLabels.map((label, index) => ({ label, type: c3bClaimTypes[index] })),
		receiptStates: [
			{ path: c3pReceiptDeclarationPath, present: true },
			{ path: c3ReceiptDeclarationPath, present: true },
		],
		hermeticArgvPrefix: c3bHermeticArgvPrefix,
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

export async function verifyC3LocalEvidence() {
	const receiptText = await readReceiptDeclarationSnapshot(repositoryRoot, c3ReceiptDeclarationPath, "C3");
	const planErrors = await checkPlan(repositoryRoot, new Map([[c3ReceiptDeclarationPath, receiptText]]));
	if (planErrors.length > 0) throw new Error(`P07B-C C3 local-evidence precondition failed:\n${planErrors.join("\n")}`);
	const receipt = JSON.parse(receiptText);
	const evidenceErrors = await verifyLocalEvidence(repositoryRoot, receipt, c3ClaimLabels, c3ClaimTypes, c3ExpectedClaimArgv);
	if (evidenceErrors.length > 0) throw new Error(`P07B-C C3 local-evidence verification failed:\n${evidenceErrors.join("\n")}`);
	console.log("P07B-C C3 local evidence passed: HTML plus the ignored sealed-source ledger snapshot match the closed C3 receipt declaration; this local snapshot is not portable strict authority");
}

async function c3bReceiptStageSnapshot() {
	await requireReceiptDeclarationPhase(repositoryRoot, c3ReceiptDeclarationPath, true, "C3B");
	const planErrors = await checkPlan();
	if (planErrors.length > 0) throw new Error(`P07B-C C3B receipt/plan check failed:\n${planErrors.join("\n")}`);
	const specificationBytes = await readRegularNoFollow(
		resolve(repositoryRoot, "spec/verification/p07b-c-unit-paths.json"), 4 * 1024 * 1024,
	);
	const specification = validateSpecification(JSON.parse(decodeGitUTF8(specificationBytes, "C3B unit specification")));
	const paths = await stagedPaths();
	if (!exactPathsMatch(specification, "C3B", paths)) throw new Error("P07B-C C3B exact staged roster mismatch");
	const entries = await stagedIndexEntries();
	validateReceiptIndexModes(specification, "C3B", paths, entries);
	const tree = decodeGitUTF8(await gitOutput(["write-tree"]), "C3B staged tree").trim();
	if (!/^(?:[0-9a-f]{40}|[0-9a-f]{64})$/u.test(tree) ||
		!isDeepStrictEqual(await stagedPaths(), paths) || !isDeepStrictEqual(await stagedIndexEntries(), entries) ||
		decodeGitUTF8(await gitOutput(["write-tree"]), "C3B repeated staged tree").trim() !== tree) {
		throw new Error("P07B-C C3B staged index changed during inspection");
	}
	return { paths, entries, tree };
}

async function requireC3BStageStable(snapshot, label) {
	if (!isDeepStrictEqual(await stagedPaths(), snapshot.paths) ||
		!isDeepStrictEqual(await stagedIndexEntries(), snapshot.entries) ||
		decodeGitUTF8(await gitOutput(["write-tree"]), `C3B ${label} staged tree`).trim() !== snapshot.tree) {
		throw new Error(`P07B-C C3B staged index changed during ${label}`);
	}
}

export async function verifyC3BStaged() {
	const snapshot = await c3bReceiptStageSnapshot();
	const { paths } = snapshot;
	await gitOutput(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--cached", "--check", "--"]);
	if ((await gitOutput(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--name-only", "-z", "--"])).length !== 0) {
		throw new Error("P07B-C C3B unstaged tracked changes are present");
	}
	if ((await gitOutput(["ls-files", "--others", "--exclude-standard", "-z", "--"])).length !== 0) {
		throw new Error("P07B-C C3B untracked paths are present");
	}
	await requireC3BStageStable(snapshot, "diff checks");
	console.log("P07B-C C3B exact staged receipt gate passed: three mode-100644 paths, clean staged diff, no unstaged or untracked paths, exact seven-claim manifest");
}

export async function verifyC3BCredentialScan() {
	const snapshot = await c3bReceiptStageSnapshot();
	const { paths } = snapshot;
	const findings = credentialPatternFindings(await stagedBlobEntries(paths));
	if (findings.length > 0) throw new Error(`P07B-C C3B structured credential-pattern findings: ${JSON.stringify(findings)}`);
	await requireC3BStageStable(snapshot, "credential scan");
	console.log(`P07B-C C3B scoped staged structured credential-pattern scan: 0 findings across ${paths.length} exact paths and 9 named patterns`);
}

async function main() {
	const mode = process.argv[2];
	const usage = "usage: check-p07b-c-plan.mjs [--self-test|--print-plan-authority-corpus|--verify-sealed-c1-local-evidence|--verify-c1-local-evidence|--verify-c2-local-evidence|--verify-c2m-preseal-ledger|--verify-c2-preseal-ledger|--verify-c3v-preseal-ledger|--verify-c3m-preseal-ledger|--verify-c3a-preseal-ledger|--verify-c3l-preseal-ledger|--verify-c3f-preseal-ledger|--verify-c3s-preseal-ledger|--verify-c3-preseal-ledger|--verify-c3r-sealed-c3-note|--verify-c3r-preseal-ledger|--verify-c3q-sealed-c3r-note|--verify-c3q-preseal-ledger|--verify-c3t-sealed-c3q-note|--verify-c3t-preseal-ledger|--verify-c3u-sealed-c3t-note|--verify-c3u-preseal-ledger|--verify-c3-local-evidence|--verify-c3b-staged|--verify-c3b-credential-scan|--verify-c3b-preseal-ledger]";
	if (mode === "--self-test") {
		if (process.argv.length !== 3) throw new Error(usage);
		await runSelfTest();
		return;
	}
	if (mode === "--print-plan-authority-corpus") {
		if (process.argv.length !== 3) throw new Error(usage);
		process.stdout.write(planAuthorityMarkdownCorpusReport());
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
	if (mode === "--verify-c3v-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3VPresealLedger();
		return;
	}
	if (mode === "--verify-c3m-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3MaintenancePresealLedger();
		return;
	}
	if (mode === "--verify-c3a-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3APresealLedger();
		return;
	}
	if (mode === "--verify-c3l-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3LPresealLedger();
		return;
	}
	if (mode === "--verify-c3f-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3FPresealLedger();
		return;
	}
	if (mode === "--verify-c3s-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3SPresealLedger();
		return;
	}
	if (mode === "--verify-c3-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3PresealLedger();
		return;
	}
	if (mode === "--verify-c3r-sealed-c3-note") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3RSealedC3Note();
		return;
	}
	if (mode === "--verify-c3r-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3RPresealLedger();
		return;
	}
	if (mode === "--verify-c3q-sealed-c3r-note") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3QSealedC3RNote();
		return;
	}
	if (mode === "--verify-c3q-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3QPresealLedger();
		return;
	}
	if (mode === "--verify-c3t-sealed-c3q-note") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3TSealedC3QNote();
		return;
	}
	if (mode === "--verify-c3t-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3TPresealLedger();
		return;
	}
	if (mode === "--verify-c3u-sealed-c3t-note") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3USealedC3TNote();
		return;
	}
	if (mode === "--verify-c3u-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3UPresealLedger();
		return;
	}
	if (mode === "--verify-c3-local-evidence") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3LocalEvidence();
		return;
	}
	if (mode === "--verify-c3b-staged") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3BStaged();
		return;
	}
	if (mode === "--verify-c3b-credential-scan") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3BCredentialScan();
		return;
	}
	if (mode === "--verify-c3b-preseal-ledger") {
		if (process.argv.length !== 3) throw new Error(usage);
		await verifyC3BPresealLedger();
		return;
	}
	if (mode !== undefined) throw new Error(usage);

	const errors = await checkPlan();
	if (errors.length > 0) {
		console.error("P07B-C evolved plan check failed:");
		for (const error of errors) console.error(`- ${error}`);
		process.exitCode = 1;
		return;
	}

	let c3ReceiptPresent = true;
	try { await lstat(resolve(repositoryRoot, c3ReceiptDeclarationPath)); } catch (error) {
		if (error.code === "ENOENT") c3ReceiptPresent = false;
		else throw error;
	}
	console.log(c3ReceiptPresent
		? "P07B-C C3B plan check passed: sealed C3 source evidence, exact sealed C3R/C3Q/C3T/C3U checker ancestry, and the active receipt-only descendant are coherent; this gate does not re-run or upgrade any C3 source claim"
		: "P07B-C C3U plan check passed: declared sealed-C3T identity, validated cross-phase legacy receipt fixtures with terminal C3 preservation, corpus-wide active-unit self-receipt refusal, pending C3 receipt grades, and the frozen C3B receipt boundary are coherent; live sealed-note authority is verified only by the dedicated compatibility and chain gates, and this gate confers no process-start or C4 authority");
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
	try {
		await main();
	} catch (error) {
		console.error(error.message);
		process.exitCode = 1;
	}
}
