#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { constants as fsConstants } from "node:fs";
import { lstat, open, readFile, realpath } from "node:fs/promises";
import { dirname, isAbsolute, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { isDeepStrictEqual } from "node:util";

export const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const specificationPath = resolve(repositoryRoot, "spec/verification/p07b-c-unit-paths.json");
const gitStderrPrefixBytes = 256;
const gitStderrMaximumBytes = 16 * 1024;
const unitOrder = Object.freeze(["C0A", "C0B", "C1", "C1M", "C1V", "C1E", "C1B", "C2", "C2M", "C2B", "C3P", "C3V", "C3M", "C3PB", "C3A", "C3L", "C3F", "C3S", "C3", "C3R", "C3Q", "C3T", "C3U", "C3B", "C3D", "C4V", "C4M", "C4N", "C4P", "C4", "C4H", "C4I", "C4K", "C4J", "C4L", "C5", "C5V", "C6A", "C6F", "C6G", "C6M", "C6B"]);
const receiptPhaseUnitOrder = Object.freeze(unitOrder.slice(unitOrder.indexOf("C3M")));
const receiptPhaseUnitSet = new Set(receiptPhaseUnitOrder);
const receiptPhaseKeys = Object.freeze(["C3P", "C3", "C6A"]);
const receiptPhaseStates = new Set(["ABSENT", "PRESENT"]);
const receiptPhaseAuthoritySHA256 = "f48f7eafe266fad4a6908bb9319f6adb9739f95a32276a7325c9ac28d7ab8e95";
const sealedC4NReceiptPhaseAuthoritySHA256 = "8f237c7d883c212167e5a04e5dd24d6efdacf4a9ce5492215fd06892a8237c4a";
const sealedC4PReceiptPhaseAuthoritySHA256 = "66f568efb313303c92b23a51b743200f15b47a451a9b3424aaa251e0228e7928";
const sealedC4HIndependentCandidateRejected = 289;
const receiptPhaseCapsuleStart = "<!-- P07B-C-RECEIPT-PHASE:START -->";
const receiptPhaseCapsuleEnd = "<!-- P07B-C-RECEIPT-PHASE:END -->";
const receiptPhaseHandoffPath = "docs/HANDOFF_MODE_C.md";
const receiptPhaseSpecificationPath = "spec/verification/p07b-c-unit-paths.json";
const c6aSourceAuthorityPath = "spec/verification/p07b-c-c6a-source-authority.json";
const c6aReceiptDeclarationPath = "spec/verification/p07b-c-c6a-receipt.json";
const c6EvidencePath = "docs/status/P07B-C-C6-EVIDENCE.md";
const c6aReceiptBlockStart = "<!-- P07B-C-C6A-SOURCE-RECEIPTS:START -->";
const c6aReceiptBlockEnd = "<!-- P07B-C-C6A-SOURCE-RECEIPTS:END -->";
const forwardCandidateReceiptPhaseBoundaries = Object.freeze(receiptPhaseUnitOrder.slice(
	receiptPhaseUnitOrder.indexOf("C3D"),
));
const forwardCandidateReceiptPhaseBoundarySet = new Set(forwardCandidateReceiptPhaseBoundaries);
const sealedC3BCandidateParent = Object.freeze({
	commit: "3b0b56d2bdd36531caf31e3dbad555aca061616d",
	tree: "15edcc92c5d1ed79885cf9caaa85b4be36918433",
});
const sealedC3DCandidateParent = Object.freeze({
	commit: "62422a2400edd702c82fb680c6ef0c6923eb6cfb",
	tree: "7573f57017c6f56b434d1fbd5642ac585f7789c2",
});
const sealedC4VCandidateParent = Object.freeze({
	commit: "861fd49a2b35b46c101dd6f18707f9c3525c6c7f",
	tree: "c2a560bf4c7bed6f93078af93714f979d06cf997",
});
const sealedC4MCandidateParent = Object.freeze({
	commit: "d87394799d2229bd6e6d342318f446272685a86f",
	tree: "afd0157303e9025a490b2b6ac991f5f15734137f",
});
const sealedC4NCandidateParent = Object.freeze({
	commit: "b915d43cced850936c46b52620654f7506bdb993",
	tree: "3ea928443ecb32d6457babe2bce02ddec1d9632f",
});
const sealedC4CandidateParent = Object.freeze({
	commit: "c4089ab31181c08dad880bf9e5d40c622c8c91c4",
	tree: "f427e23b6fe297a49a36a99fa037073b637d82ca",
});
const sealedC4HCandidateParent = Object.freeze({
	commit: "4b4686ea30e5ddc5043eee11ba24725bcce8bb94",
	tree: "2d02d220961f3fc41b2372df5eb8dc3aac47a529",
});
const sealedC4ICandidateParent = Object.freeze({
	commit: "d784ace97dd03660c6c724e1cb8f838b869681ec",
	tree: "e38b702884b9df128b41bfa7710459dd92f901d7",
});
const sealedC4KCandidateParent = Object.freeze({
	commit: "8c1ee2df955055ee40eec1f14f0c4d96f97d5361",
	tree: "99fe7082d05c96647a144785e0ddc5af1a09f52f",
});
const sealedC4JCandidateParent = Object.freeze({
	commit: "c12c927d94e6c56529a2fb90148674b4b4e731a5",
	tree: "c99c1f394e02f17f8cf9de6b18f1a949bcaa07a8",
	phaseAuthoritySHA256: "30530a2e05fa6b7fb74983756c24f22a898f4a6da6df75d447df17278b0152db",
});
const sealedC5CandidateParent = Object.freeze({
	commit: "240060f018d5b0e86914bd27361c5899ac9ba1c0",
	tree: "4906ff4af407fa4e48cbf569f7e456694660ca06",
});
const sealedC6ACandidateParent = Object.freeze({
	commit: "3e9643f657248e0d5ff2c4bf0880218efbaeecd8",
	tree: "226a1e23da7eded933b3faabd4e8040576b2a2e0",
});
const sealedC6FCandidateParent = Object.freeze({
	commit: "dd2a011edff20d65c43934fc7c3e5a09b5829d2e",
	tree: "b0141ac2671df75e51c4fdb5b71c35cee5d0f4ea",
});
const candidateSpecificationOwnerContracts = Object.freeze({
	C4V: Object.freeze({ parent: "C3D", authority: sealedC3DCandidateParent, priorSchema: "countershape/p07b-c-unit-paths/v16" }),
	C4M: Object.freeze({ parent: "C4V", authority: sealedC4VCandidateParent, priorSchema: "countershape/p07b-c-unit-paths/v17" }),
	C4N: Object.freeze({ parent: "C4M", authority: sealedC4MCandidateParent, priorSchema: "countershape/p07b-c-unit-paths/v18" }),
	C4P: Object.freeze({ parent: "C4N", authority: sealedC4NCandidateParent, priorSchema: "countershape/p07b-c-unit-paths/v19" }),
	C4H: Object.freeze({ parent: "C4", authority: sealedC4CandidateParent, priorSchema: "countershape/p07b-c-unit-paths/v20" }),
	C4I: Object.freeze({ parent: "C4H", authority: sealedC4HCandidateParent, priorSchema: "countershape/p07b-c-unit-paths/v21" }),
	C4K: Object.freeze({ parent: "C4I", authority: sealedC4ICandidateParent, priorSchema: "countershape/p07b-c-unit-paths/v22" }),
	C4J: Object.freeze({ parent: "C4K", authority: sealedC4KCandidateParent, priorSchema: "countershape/p07b-c-unit-paths/v23" }),
	C4L: Object.freeze({ parent: "C4J", authority: sealedC4JCandidateParent, priorSchema: "countershape/p07b-c-unit-paths/v24" }),
	C5V: Object.freeze({ parent: "C5", authority: sealedC5CandidateParent, priorSchema: "countershape/p07b-c-unit-paths/v25" }),
	C6F: Object.freeze({ parent: "C6A", authority: sealedC6ACandidateParent, priorSchema: "countershape/p07b-c-unit-paths/v26" }),
	C6G: Object.freeze({ parent: "C6F", authority: sealedC6FCandidateParent, priorSchema: "countershape/p07b-c-unit-paths/v27" }),
});
const candidateSpecificationOwners = new Set(["C3D", ...Object.keys(candidateSpecificationOwnerContracts)]);
const verificationProfiles = new Set(["SOURCE_FULL", "RECEIPT_RECONCILIATION", "NON_PRODUCT_MAINTENANCE"]);
const ownerOutOfBandUnits = new Set(["C4H", "C4I", "C4K", "C4J", "C4L", "C5V", "C6F", "C6G"]);
const knownOwnerOutOfBandProvenance = Object.freeze({
	C4H: "UNEVIDENCED",
	C4I: "UNEVIDENCED",
	C4K: "UNEVIDENCED",
	C4J: "UNEVIDENCED",
	C4L: "UNEVIDENCED",
	C5V: "UNEVIDENCED",
	C6F: "UNEVIDENCED",
	C6G: "UNEVIDENCED",
});
const ownerAuthorityDisclosure =
	"OWNER_ATTRIBUTED_SESSION_INSTRUCTION_ONLY_NO_QUALIFYING_PREEXISTING_ARTIFACT";
const ownerAuthorityCommonFields = Object.freeze([
	"authentication", "classification", "predecessor_declaration", "provenance",
	"signed_authorization", "source",
]);
const nonProductMaintenanceClaimRoles = Object.freeze([
	"CANDIDATE_PHASE",
	"INDEPENDENT_TRANSITION",
	"PARENT_PLAN_SELF_TEST",
	"SEALED_PARENT_ANCESTRY",
	"PARENT_SCOPE_SELF_TEST",
	"BOUNDARY_MAINTENANCE_SELF_TEST",
	"STAGED_SCOPE_PRODUCT_PROJECTION",
	"CREDENTIAL_SCAN",
	"PREDECESSOR_CHAIN",
]);
const nonProductMaintenanceForbiddenAuthorityPaths = new Set([
	"docs/ARCHITECTURE.md",
	"docs/CLAIM_VOCABULARY.md",
	"docs/CONCEPT_BRIEF.md",
	"docs/SEMANTICS.md",
	"docs/STATE_MACHINES.md",
	"docs/THREAT_MODEL.md",
	"docs/VERIFICATION.md",
	"spec/verification/p07b-c-unit-paths.json",
	"tools/check-p07b-c-plan.mjs",
	"tools/check-p07b-c-unit-scope.mjs",
	"tools/verify-current.mjs",
	"tools/verify-current-selftest.mjs",
	"tools/verify-runtime-authority.mjs",
]);
const nonProductMaintenanceClaimSuffixByRole = Object.freeze({
	CANDIDATE_PHASE: "candidate phase plan coherence",
	INDEPENDENT_TRANSITION: "independent candidate transition authority",
	PARENT_PLAN_SELF_TEST: "parent-sealed plan checker defensive self-test",
	SEALED_PARENT_ANCESTRY: "sealed-parent Git-note and ancestry compatibility",
	PARENT_SCOPE_SELF_TEST: "parent-sealed unit-scope defensive self-test",
	BOUNDARY_MAINTENANCE_SELF_TEST: "boundary-specific maintenance defensive self-test",
	STAGED_SCOPE_PRODUCT_PROJECTION: "exact staged maintenance scope and no change outside the parent-predeclared roster relative to exact parent",
	CREDENTIAL_SCAN: "scoped staged credential-pattern scan",
	PREDECESSOR_CHAIN: "sealed-parent predecessor and preceding didrun chain integrity",
});
const receiptClaimTypes = new Set(["tests-pass", "command-succeeded"]);
function boundedErrorDescriptor(error) {
	if (error === undefined || error === null) return null;
	const message = Buffer.from(String(error.message ?? error), "utf8");
	const prefix = message.subarray(0, gitStderrPrefixBytes);
	const name = Buffer.from(String(error.name ?? "Error"), "utf8").subarray(0, 64).toString("hex");
	const code = error.code === undefined ? null :
		Buffer.from(String(error.code), "utf8").subarray(0, 64).toString("hex");
	return Object.freeze({
		name_hex: name,
		code_hex: code,
		message_bytes: message.length,
		message_prefix_hex: prefix.toString("hex"),
		message_truncated: message.length > gitStderrPrefixBytes,
	});
}
const c3FrozenUnitContracts = Object.freeze({
	C3: Object.freeze({
		verification_profile: "SOURCE_FULL",
		prefixes: Object.freeze([]),
		exact_roster_sha256: "c0050817c739e6bf7505105113ebbfd2c2504674f93f14e34daf0609f7fad0fb",
	}),
	C3R: Object.freeze({
		verification_profile: "SOURCE_FULL",
		prefixes: Object.freeze([]),
		exact_roster_sha256: "fe14d9f9985771cce0525edfca47c001b5912dacc4c2d6fef182458d3b44f58e",
	}),
	C3Q: Object.freeze({
		verification_profile: "SOURCE_FULL",
		prefixes: Object.freeze([]),
		exact_roster_sha256: "f5f27e661c1ace6516aba4b9d5766f4c6ab50b6602dbcb101bff9b6a4ebfe6fc",
	}),
	C3T: Object.freeze({
		verification_profile: "SOURCE_FULL",
		prefixes: Object.freeze([]),
		exact_roster_sha256: "496788860db6c42f5acc50de0de02be8a63bcb8e11f99fb24d59d9c0e016477e",
	}),
	C3U: Object.freeze({
		verification_profile: "SOURCE_FULL",
		prefixes: Object.freeze([]),
		exact_roster_sha256: "7af7456929b0db08abc2a31bd1cb62d84002c1a6bcdda3f4a8b4896f6b8c30cd",
	}),
	C3B: Object.freeze({
		verification_profile: "RECEIPT_RECONCILIATION",
		prefixes: Object.freeze([]),
		exact_roster_sha256: "3ef17156a2f3051171da70983f9428de2b9bdb5a179602407a36f34200012a67",
		receipt_claims: Object.freeze([
			Object.freeze({ label: "P07B-C C3B source receipt reconciliation", type: "tests-pass" }),
			Object.freeze({ label: "P07B-C C3B receipt checker defensive self-test", type: "tests-pass" }),
			Object.freeze({ label: "P07B-C C3B declared local source-evidence snapshot match", type: "tests-pass" }),
			Object.freeze({ label: "P07B-C C3B receipt-only Go build", type: "command-succeeded" }),
			Object.freeze({ label: "P07B-C C3B exact three-path staged scope and diff integrity", type: "command-succeeded" }),
			Object.freeze({ label: "P07B-C C3B scoped staged credential-pattern scan", type: "command-succeeded" }),
			Object.freeze({ label: "P07B-C C3B preceding didrun chain integrity", type: "command-succeeded" }),
		]),
	}),
	C3D: Object.freeze({
		verification_profile: "SOURCE_FULL",
		prefixes: Object.freeze([]),
		exact_roster_sha256: "ac4436680c612ad206d2786ddd3146af747e1f8ea2bdedf898afca67d46fcd63",
	}),
});
const c4vDeclaredMaintenanceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/THREAT_MODEL.md",
		"docs/VERIFICATION.md",
		"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md",
		"docs/status/P07B-C-C4V-EXECUTION-CONTRACT-MAINTENANCE.md",
		"spec/verification/p07b-c-unit-paths.json",
		"tools/check-p07b-c-plan.mjs",
		"tools/check-p07b-c-unit-scope.mjs",
		"tools/verify-current-selftest.mjs",
		"tools/verify-current.mjs",
	]),
	exact_roster_sha256: "535a64fbb9f0d3c3672b4b8197fc7815da315117b1794a668e6565faa16df3f1",
});
const c4mDeclaredMaintenanceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/VERIFICATION.md",
		"docs/status/P07B-C-C4M-HISTORICAL-RECEIPT-FIXTURE-MAINTENANCE.md",
		"spec/verification/p07b-c-unit-paths.json",
		"tools/check-p07b-c-plan.mjs",
		"tools/check-p07b-c-unit-scope.mjs",
	]),
	exact_roster_sha256: "ec49dc68ce8fcfe92720fd2e8d6afa6d19d6d1c26aa9d27173dc2df736d06a87",
});
const c4nDeclaredMaintenanceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/VERIFICATION.md",
		"docs/status/P07B-C-C4N-SELF-RECEIPT-CARDINALITY-MAINTENANCE.md",
		"spec/verification/p07b-c-unit-paths.json",
		"tools/check-p07b-c-plan.mjs",
		"tools/check-p07b-c-unit-scope.mjs",
	]),
	exact_roster_sha256: "95de7e71ca2df4a57d0544c3197bb7d917359bb3d9ecfc51a2130973faf36e6e",
});
const c4pDeclaredMaintenanceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/VERIFICATION.md",
		"docs/status/P07B-C-C4P-FUTURE-SURFACE-PHASE-MAINTENANCE.md",
		"spec/verification/p07b-c-unit-paths.json",
		"tools/check-p07b-c-plan.mjs",
		"tools/check-p07b-c-unit-scope.mjs",
	]),
	exact_roster_sha256: "5dc083fadb60003b0ba96516d0a01f91daa88b7298b142018b5eda00e48ab59b",
});
const c4DeclaredSourceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([
		"internal/contractexec/runner/",
		"internal/processmechanics/",
		"testkit/contractexec/cli/",
	]),
	exact: Object.freeze([
		"docs/ARCHITECTURE.md",
		"docs/CLAIM_VOCABULARY.md",
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/SEMANTICS.md",
		"docs/STATE_MACHINES.md",
		"docs/THREAT_MODEL.md",
		"docs/VERIFICATION.md",
		"docs/status/P07B-C-C4-CLI-PROFILE.md",
		"internal/store/contract_run_bridge.go",
		"internal/store/contract_run_bridge_test.go",
		"internal/store/execution_interlock.go",
		"internal/store/execution_interlock_test.go",
		"internal/store/nonhead_contract.go",
		"internal/store/nonhead_contract_test.go",
		"internal/store/private_contract_run.go",
		"internal/store/private_contract_run_test.go",
		"internal/store/public_api_test.go",
		"internal/world/capture.go",
		"internal/world/process.go",
		"internal/world/process_darwin.go",
		"internal/world/process_darwin_test.go",
		"internal/world/process_mutation_darwin_test.go",
		"internal/world/process_unsupported.go",
		"spec/verification/p07b-b-future-surface-authority.json",
		"tools/check-p07b-b-architecture-selftest.mjs",
		"tools/check-p07b-b-architecture.mjs",
		"tools/check-p07b-c-architecture-selftest.mjs",
		"tools/check-p07b-c-architecture.mjs",
		"tools/check-u6-architecture-selftest.mjs",
		"tools/check-u6-architecture.mjs",
		"tools/mutate-u2.mjs",
		"tools/mutate-u3.mjs",
		"tools/test-mutate-u2.mjs",
		"tools/verify-current-selftest.mjs",
		"tools/verify-current.mjs",
		"tools/verify-go-test-repetition.mjs",
		"tools/verify-runtime-authority.mjs",
	]),
	exact_roster_sha256: "99bda0d3e5c2f2a2ee4f354945188bf2e5563716c9e26511211a4bfc9197bf56",
	prefix_roster_sha256: "f5f7b8cdfeba3581e8632c0d6686287f5afa00aef224bb4babe39ee1fdc7f4b2",
});
const c4hDeclaredMaintenanceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/VERIFICATION.md",
		"docs/status/P07B-C-C4H-VERIFIER-HERMETICITY-MAINTENANCE.md",
		"research/deep-dive/p07b-c-maintenance-architecture/00-scope-and-method.md",
		"research/deep-dive/p07b-c-maintenance-architecture/01-proof-semantics.md",
		"research/deep-dive/p07b-c-maintenance-architecture/02-verifier-throughput.md",
		"research/deep-dive/p07b-c-maintenance-architecture/03-maintenance-order.md",
		"research/deep-dive/p07b-c-maintenance-architecture/04-synthesis.md",
		"research/deep-dive/p07b-c-maintenance-architecture/05-follow-up-verification.md",
		"research/deep-dive/p07b-c-maintenance-architecture/06-different-model-red-team.md",
		"research/deep-dive/p07b-c-maintenance-architecture/07-executive-briefing.md",
		"spec/verification/p07b-c-unit-paths.json",
		"tools/check-p07b-c-plan.mjs",
		"tools/check-p07b-c-unit-scope.mjs",
		"tools/generate-p07-planning-example.mjs",
		"tools/verify-current-selftest.mjs",
		"tools/verify-current.mjs",
	]),
	exact_roster_sha256: "1adc913036078ad4efdf418e78581d81369aa630d245a5b493c60a69a0478532",
});
const c4iDeclaredMaintenanceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/THREAT_MODEL.md",
		"docs/VERIFICATION.md",
		"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md",
		"docs/status/P07B-C-C4I-A2-AST-STABILITY-MAINTENANCE.md",
		"internal/world/process_darwin_test.go",
		"spec/verification/p07b-c-unit-paths.json",
		"testkit/processfixture/main.go",
		"tools/capture-p07b-a2-human-surface.mjs",
		"tools/check-p07b-a2-architecture-selftest.mjs",
		"tools/check-p07b-a2-architecture.mjs",
		"tools/check-p07b-c-plan.mjs",
		"tools/check-p07b-c-unit-scope.mjs",
	]),
	exact_roster_sha256: "522af4209bde3f769a12c3943f46bd83e566977ad8f275165310962c1e697585",
});
const c4kDeclaredMaintenanceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/THREAT_MODEL.md",
		"docs/VERIFICATION.md",
		"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md",
		"docs/status/P07B-C-C4K-C3-GO-TIMEOUT-MAINTENANCE.md",
		"spec/verification/p07b-c-unit-paths.json",
		"tools/check-p07b-c-architecture-selftest.mjs",
		"tools/check-p07b-c-architecture.mjs",
		"tools/check-p07b-c-plan.mjs",
		"tools/check-p07b-c-unit-scope.mjs",
	]),
	exact_roster_sha256: "9ab25f722aeaae583d08df271d6162c873254c51799b36b03fc85ff64d78567e",
});
const c4jDeclaredMaintenanceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/THREAT_MODEL.md",
		"docs/VERIFICATION.md",
		"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md",
		"docs/status/P07B-C-C4J-C5-VERIFIER-TIMEOUT-MAINTENANCE.md",
		"research/deep-dive/p07b-c-c5-timeout-maintenance/01-empirical-failure.md",
		"research/deep-dive/p07b-c-c5-timeout-maintenance/02-runtime-authority.md",
		"research/deep-dive/p07b-c-c5-timeout-maintenance/03-governance-phase-machine.md",
		"research/deep-dive/p07b-c-c5-timeout-maintenance/04-receipt-runbook-impact.md",
		"research/deep-dive/p07b-c-c5-timeout-maintenance/05-synthesis.md",
		"research/deep-dive/p07b-c-c5-timeout-maintenance/06-red-team.md",
		"research/deep-dive/p07b-c-c5-timeout-maintenance/07-executive-briefing.md",
		"spec/verification/p07b-c-unit-paths.json",
		"tools/check-p07b-c-plan.mjs",
		"tools/check-p07b-c-unit-scope.mjs",
		"tools/verify-current-selftest.mjs",
		"tools/verify-current.mjs",
		"tools/verify-runtime-authority.mjs",
	]),
	exact_roster_sha256: "c0920287c9dc9998f0cae11f8beced9d134c74da786a10f1886b03b2a1a50239",
});
const c4lDeclaredMaintenanceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/ARCHITECTURE.md",
		"docs/CLAIM_VOCABULARY.md",
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/SEMANTICS.md",
		"docs/STATE_MACHINES.md",
		"docs/THREAT_MODEL.md",
		"docs/VERIFICATION.md",
		"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md",
		"docs/status/P07B-C-C4L-NON-PRODUCT-MAINTENANCE-BOOTSTRAP.md",
		"spec/verification/p07b-c-unit-paths.json",
		"tools/check-p07b-c-plan.mjs",
		"tools/check-p07b-c-unit-scope.mjs",
	]),
	exact_roster_sha256: "76c8ecfba55dd8833333c0f6180bf1df38f1a7cba65373feaea281add963b1b3",
});
const c5DeclaredSourceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([
		"internal/contractexec/http/",
		"internal/contractexec/scope/",
		"testkit/contractexec/http/",
	]),
	exact: Object.freeze([
		"docs/ARCHITECTURE.md",
		"docs/CLAIM_VOCABULARY.md",
		"docs/HANDOFF_MODE_C.md",
		"docs/SEMANTICS.md",
		"docs/STATE_MACHINES.md",
		"docs/THREAT_MODEL.md",
		"docs/VERIFICATION.md",
		"docs/status/P07B-C-C5-HTTP-SCOPE.md",
		"tools/check-p07b-c-architecture-selftest.mjs",
		"tools/check-p07b-c-architecture.mjs",
		"tools/verify-current-selftest.mjs",
		"tools/verify-current.mjs",
	]),
	exact_roster_sha256: "d16ba74582e53dc5c791b8c34fbcd6200338dc066ec13230c2908eb6ba257bf5",
	prefix_roster_sha256: "a9212680066fdbddc8f25eef04a41c264ac9148a506238dd92c39978e3e35f84",
});
const c5vDeclaredSourceMaintenanceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/VERIFICATION.md",
		"docs/status/P07B-C-C5V-VERIFIER-PREFLIGHT-HYGIENE.md",
		"spec/verification/p07b-c-unit-paths.json",
		"tools/check-p07b-c-plan.mjs",
		"tools/check-p07b-c-unit-scope.mjs",
		"tools/verify-current-selftest.mjs",
		"tools/verify-current.mjs",
	]),
	exact_roster_sha256: "42d2961220c8a2d06d70fe5fa9627f220569a74b3b6d2df9a26858f770481031",
});
const c6bDeclaredReceiptContract = Object.freeze({
	verification_profile: "RECEIPT_RECONCILIATION",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/HANDOFF_MODE_C.md",
		"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md",
		"docs/status/DIDRUN_BUGS.md",
		"docs/status/P07B-C-C6-EVIDENCE.md",
		c6aReceiptDeclarationPath,
	]),
	exact_roster_sha256: "94501edc81cc170b220fb83873f70ba638a3f19a0a317e2e8df226057dc77a48",
	receipt_claims: Object.freeze([
		Object.freeze({ label: "P07B-C C6B candidate phase plan coherence", type: "tests-pass" }),
		Object.freeze({ label: "P07B-C C6B independent sealed-C6A source-authority conformance", type: "tests-pass" }),
		Object.freeze({ label: "P07B-C C6B source receipt declaration reconciliation", type: "tests-pass" }),
		Object.freeze({ label: "P07B-C C6B receipt checker defensive self-test", type: "tests-pass" }),
		Object.freeze({ label: "P07B-C C6B declared local source-evidence and private-availability snapshot match", type: "tests-pass" }),
		Object.freeze({ label: "P07B-C C6B receipt-only Go build", type: "command-succeeded" }),
		Object.freeze({ label: "P07B-C C6B exact five-path staged scope and diff integrity", type: "command-succeeded" }),
		Object.freeze({ label: "P07B-C C6B scoped staged credential-pattern scan", type: "command-succeeded" }),
		Object.freeze({ label: "P07B-C C6B preceding didrun chain integrity", type: "command-succeeded" }),
	]),
});
const c6fDeclaredMaintenanceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/VERIFICATION.md",
		"docs/status/P07B-C-C6F-HISTORICAL-AUTHORITY-FIXTURE-MAINTENANCE.md",
		"spec/verification/p07b-c-unit-paths.json",
		"tools/check-p07b-b-architecture-selftest.mjs",
		"tools/check-p07b-b-architecture.mjs",
		"tools/check-p07b-c-plan.mjs",
		"tools/check-p07b-c-unit-scope.mjs",
		"tools/check-sealed-c6a-architecture.mjs",
		"tools/verify-current-selftest.mjs",
		"tools/verify-current.mjs",
	]),
	exact_roster_sha256: "6f4d8c5b02df2d2e0df6e8f4b91b66c9f1465bce427939e8ccc1442d64b85c9c",
});
const c6gDeclaredMaintenanceContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/VERIFICATION.md",
		"docs/status/P07B-C-C6G-C3U-HISTORICAL-FIXTURE-MAINTENANCE.md",
		"spec/verification/p07b-c-unit-paths.json",
		"tools/check-p07b-c-plan.mjs",
		"tools/check-p07b-c-unit-scope.mjs",
	]),
	exact_roster_sha256: "c2b754a773ed5f1a17a3da9ca61f13525fea7235fb7e09422582c3ab2fa14724",
});
const c6mDeclaredAdapterContract = Object.freeze({
	verification_profile: "SOURCE_FULL",
	prefixes: Object.freeze([]),
	exact: Object.freeze([
		"docs/HANDOFF_MODE_C.md",
		"docs/PROMPT_PACK.md",
		"docs/VERIFICATION.md",
		"docs/status/P07B-C-C6M-RECEIPT-ADAPTER-MAINTENANCE.md",
		c6aSourceAuthorityPath,
	]),
	exact_roster_sha256: "5501a7e8c2d8f6d44392d101110edafe90a0348ef77f5ea24539f6a8ba969803",
});
const c6aSourceSubject = "test: close P07B-C cumulative evidence";
const c6fSourceSubject = "fix: isolate historical C6 authority fixtures";
const c6gSourceSubject = "fix: isolate C3U historical C6 fixtures";
const c6mSourceSubject = "fix: bind sealed C6A source authority";
const c6aSourceAuthoritySchema = "countershape/p07b-c-c6a-source-authority/v1";
const sealedSourceHermeticRedaction = "«redacted:high-entropy»";
const sealedSourceDoubleRedactionPositions = new Set([2, 4, 5, 7, 8]);
const sealedSourceBareRedactionPositions = new Set([31, 32, 33, 35, 36]);

function sealedSourceExpectedRedactionProjection(position, expected) {
	if (sealedSourceDoubleRedactionPositions.has(position)) {
		return `${sealedSourceHermeticRedaction}.${sealedSourceHermeticRedaction}`;
	}
	if (sealedSourceBareRedactionPositions.has(position)) return sealedSourceHermeticRedaction;
	if (position !== 6) return undefined;
	const suffixOffset = expected.indexOf(".countershape/");
	return suffixOffset === -1 ? undefined : `${sealedSourceHermeticRedaction}${expected.slice(suffixOffset)}`;
}

function sealedSourceHermeticArgvPrefix(finalRunRoot) {
	return Object.freeze([
		"/usr/bin/env", "-i",
		`HOME=${resolve(finalRunRoot, "home")}`,
		`PWD=${repositoryRoot}`,
		`TMPDIR=${resolve(finalRunRoot, "tmp")}`,
		`GOTMPDIR=${resolve(finalRunRoot, "gotmp")}`,
		`GOCACHE=${resolve(finalRunRoot, "gocache")}`,
		`GOPATH=${resolve(finalRunRoot, "gopath")}`,
		`GOMODCACHE=${resolve(finalRunRoot, "gomodcache")}`,
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
}

function sealedSourceExpectedClaimArgv(prefix, commandTails) {
	return Object.freeze(commandTails.map((tail) => Object.freeze([...prefix, ...tail])));
}

const c6aSourceFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c6a-final");
const c6aSourceHermeticArgvPrefix = sealedSourceHermeticArgvPrefix(c6aSourceFinalRunRoot);
const c6fSourceFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c6f-final");
const c6fSourceHermeticArgvPrefix = sealedSourceHermeticArgvPrefix(c6fSourceFinalRunRoot);
const c6gSourceFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c6g-final");
const c6gSourceHermeticArgvPrefix = sealedSourceHermeticArgvPrefix(c6gSourceFinalRunRoot);
const c6mSourceFinalRunRoot = resolve(repositoryRoot, ".countershape/p07bc-c6m-final");
const c6mSourceHermeticArgvPrefix = sealedSourceHermeticArgvPrefix(c6mSourceFinalRunRoot);
const c6aSourceClaimManifest = Object.freeze([
	Object.freeze({ label: "P07B-C C6A candidate phase plan coherence", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6A independent candidate transition authority", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6A architecture conformance", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6A architecture defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6A cumulative verifier self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6A cumulative verification", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6A final evidence defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6A exact CLI HTTP Node and documentation evidence closure", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6A exact staged scope and diff integrity", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C6A scoped staged credential-pattern scan", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C6A declared C5V parent edge and preceding didrun chain integrity", type: "command-succeeded" }),
]);
const c6aSourceCommandTails = Object.freeze([
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--check-candidate-phase", "C6A"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--candidate-phase", "C6A"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-architecture.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-architecture-selftest.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/p07b-c/check-final-evidence.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/p07b-c/check-final-evidence.mjs", "--verify"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C6A", "--source-final-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C6A", "--credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c6a-preseal-ledger"]),
]);
const c6aSourceExpectedClaimArgv = sealedSourceExpectedClaimArgv(c6aSourceHermeticArgvPrefix, c6aSourceCommandTails);
const c6fSourceClaimManifest = Object.freeze([
	Object.freeze({ label: "P07B-C C6F candidate phase plan coherence", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6F independent candidate transition authority", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6F historical C6 authority fixture defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6F sealed-C6A Git-note source authority compatibility", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6F unit-scope defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6F cumulative verifier self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6F live-current verification and sealed-C6A architecture replay", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6F exact twelve-path staged scope and diff integrity", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C6F scoped staged credential-pattern scan", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C6F sealed-C6A predecessor and preceding didrun chain integrity", type: "command-succeeded" }),
]);
const c6fSourceCommandTails = Object.freeze([
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--check-candidate-phase", "C6F"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--candidate-phase", "C6F"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c6f-sealed-c6a-note"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C6F", "--source-final-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C6F", "--credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c6f-preseal-ledger"]),
]);
const c6fSourceExpectedClaimArgv = sealedSourceExpectedClaimArgv(c6fSourceHermeticArgvPrefix, c6fSourceCommandTails);
const expectedC6FSealedSourceContractDigest = "sha256:84227e43b4534cc6977d5b5d274ee415f118af98c36e3e390c86d585d11aa5fb";
export function computedC6FSealedSourceContractDigest() {
	const projection = Object.freeze({
		version: "countershape/p07b-c-c6f-sealed-source-contract/v1",
		subject: c6fSourceSubject,
		claims: c6fSourceClaimManifest,
		command_tails: c6fSourceCommandTails,
		final_run_root: ".countershape/p07bc-c6f-final",
		redaction: Object.freeze({
			double_positions: Object.freeze([2, 4, 5, 7, 8]),
			gocache_position: 6,
			bare_positions: Object.freeze([31, 32, 33, 35, 36]),
		}),
	});
	return `sha256:${createHash("sha256").update(`${JSON.stringify(projection)}\n`, "utf8").digest("hex")}`;
}
const c6gSourceClaimManifest = Object.freeze([
	Object.freeze({ label: "P07B-C C6G data-driven phase plan coherence", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6G independent candidate transition authority", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6G historical C3U future-C6 fixture isolation defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6G sealed-C6F Git-note and sealed-C6A ancestry compatibility", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6G unit-scope defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6G cumulative verifier self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6G cumulative verification", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6G exact seven-path staged scope and diff integrity", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C6G scoped staged credential-pattern scan", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C6G sealed-C6F predecessor and preceding didrun chain integrity", type: "command-succeeded" }),
]);
const c6gSourceCommandTails = Object.freeze([
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--check-candidate-phase", "C6G"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--candidate-phase", "C6G"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c6g-sealed-c6f-note"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C6G", "--source-final-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C6G", "--credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c6g-preseal-ledger"]),
]);
const c6gSourceExpectedClaimArgv = sealedSourceExpectedClaimArgv(c6gSourceHermeticArgvPrefix, c6gSourceCommandTails);
const expectedC6GSealedSourceContractDigest = "sha256:3dfffad546f7fee36e2e51856cea98a754626f395f319b0c6264fe918df3ef00";
export function computedC6GSealedSourceContractDigest() {
	const projection = Object.freeze({
		version: "countershape/p07b-c-c6g-sealed-source-contract/v1",
		subject: c6gSourceSubject,
		claims: c6gSourceClaimManifest,
		command_tails: c6gSourceCommandTails,
		final_run_root: ".countershape/p07bc-c6g-final",
		redaction: Object.freeze({
			double_positions: Object.freeze([2, 4, 5, 7, 8]),
			gocache_position: 6,
			bare_positions: Object.freeze([31, 32, 33, 35, 36]),
		}),
	});
	return `sha256:${createHash("sha256").update(`${JSON.stringify(projection)}\n`, "utf8").digest("hex")}`;
}
const c6mSourceClaimManifest = Object.freeze([
	Object.freeze({ label: "P07B-C C6M data-driven phase plan coherence", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6M independent candidate transition authority", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6M source-authority defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6M sealed-C6A source-authority conformance", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6M unit-scope defensive self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6M cumulative verifier self-test", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6M cumulative verification", type: "tests-pass" }),
	Object.freeze({ label: "P07B-C C6M exact five-path staged scope and diff integrity", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C6M scoped staged credential-pattern scan", type: "command-succeeded" }),
	Object.freeze({ label: "P07B-C C6M sealed-C6G predecessor, sealed-C6F and sealed-C6A ancestry, and preceding didrun chain integrity", type: "command-succeeded" }),
]);
const c6mSourceCommandTails = Object.freeze([
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--check-candidate-phase", "C6M"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--candidate-phase", "C6M"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C6M", "--source-authority-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/verify-current.mjs"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C6M", "--source-final-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C6M", "--credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c6m-preseal-ledger"]),
]);
const c6mSourceExpectedClaimArgv = sealedSourceExpectedClaimArgv(c6mSourceHermeticArgvPrefix, c6mSourceCommandTails);
const c6bReceiptCommandTails = Object.freeze([
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--check-candidate-phase", "C6B"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C6B", "--source-authority-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c6a-source-receipt"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test-c6a-source-receipt"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c6a-local-evidence"]),
	Object.freeze(["/opt/homebrew/bin/go", "build", "-mod=readonly", "-buildvcs=false", "-p=1", "./..."]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C6B", "--receipt-final-gate"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C6B", "--credential-scan"]),
	Object.freeze(["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c6b-preseal-ledger"]),
]);
const stagedInventoryArgs = Object.freeze([
	"diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--cached", "--name-only", "-z", "--no-renames", "--diff-filter=ACDMRTUXB", "--",
]);
const stagedIndexArgs = Object.freeze(["ls-files", "--stage", "-z", "--"]);
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

function fail(message) {
	throw new Error(`P07B-C unit scope check failed: ${message}`);
}

function validPath(path, prefix = false) {
	if (typeof path !== "string" || path.length === 0 || path.length > 4096) return false;
	if (prefix && !path.endsWith("/")) return false;
	const checked = prefix ? path.slice(0, -1) : path;
	return checked.length > 0 && !isAbsolute(checked) && !checked.startsWith("./") &&
		!checked.includes("\\") && !checked.includes("//") &&
		!checked.split("/").some((part) => part === "" || part === "." || part === "..") &&
		!/[\u0000-\u001f\u007f]/u.test(checked) && (!prefix || !markdownAuthorityPath(checked));
}

function markdownAuthorityPath(path) {
	return /\.(?:md|markdown)$/iu.test(path);
}

function sortedUnique(values) {
	return Array.isArray(values) && new Set(values).size === values.length &&
		JSON.stringify(values) === JSON.stringify([...values].sort());
}

function exactRosterDigest(paths) {
	return createHash("sha256").update(`${paths.join("\n")}\n`, "utf8").digest("hex");
}

function exactObjectKeys(value, expected) {
	return value && typeof value === "object" && !Array.isArray(value) &&
		isDeepStrictEqual(Object.keys(value).sort(), [...expected].sort());
}

function validSHA256(value) {
	return typeof value === "string" && /^sha256:[0-9a-f]{64}$/u.test(value) &&
		!/^sha256:0{64}$/u.test(value);
}

function validateAuthorityArtifactReference(value, label, expectedRole) {
	const expectedKeys = ["artifact_path", "artifact_role", "artifact_sha256", "kind", "preexistence"];
	if (!exactObjectKeys(value, expectedKeys) || value.kind !== "PROSE_ONLY" ||
		value.artifact_role !== expectedRole || value.preexistence !== "DIRECT_PARENT_TREE" ||
		!validPath(value.artifact_path) || !validSHA256(value.artifact_sha256)) {
		fail(`${label}: predecessor artifact reference`);
	}
}

function validateTransitionAuthority(unit, authority) {
	if (!exactObjectKeys(authority, ownerAuthorityCommonFields) ||
		authority.source !== "OWNER_OUT_OF_BAND" ||
		authority.classification !== "OWNER_AUTHORIZED_AUTHORITY_MIGRATION / DEFECT_REPAIR" ||
		authority.authentication !== "NOT_ESTABLISHED" ||
		authority.signed_authorization !== "NOT_IMPLEMENTED") {
		fail(`${unit}: transition authority common fields`);
	}
	if (unit === "C4J") {
		validateAuthorityArtifactReference(
			authority.predecessor_declaration, `${unit} transition authority`,
			"PREDECESSOR_RECORD_OF_OWNER_ATTRIBUTED_DIRECTION",
		);
		if (authority.predecessor_declaration.artifact_path !==
			"docs/status/P07B-C-C4K-C3-GO-TIMEOUT-MAINTENANCE.md" ||
			authority.predecessor_declaration.artifact_sha256 !==
			"sha256:fc80148ddedddb925962a15dacb20dad8979426aa3f3ef4f8fef12df9c614d6b") {
			fail(`${unit}: exact predecessor prose record`);
		}
	} else if (!exactObjectKeys(authority.predecessor_declaration, ["kind"]) ||
		authority.predecessor_declaration.kind !== "NONE") {
		fail(`${unit}: predecessor declaration`);
	}
	const provenance = authority.provenance;
	if (provenance?.kind === "UNEVIDENCED") {
		if (!exactObjectKeys(provenance, ["disclosure", "kind"]) ||
			provenance.disclosure !== ownerAuthorityDisclosure) {
			fail(`${unit}: unevidenced owner authority disclosure`);
		}
	} else if (provenance?.kind === "CITED_UNAUTHENTICATED") {
		if (!exactObjectKeys(provenance, [
			"artifact_path", "artifact_role", "artifact_sha256", "kind", "preexistence",
		]) || provenance.artifact_role !== "OWNER_INSTRUCTION_ARTIFACT" ||
			provenance.preexistence !== "DIRECT_PARENT_TREE" ||
			!validPath(provenance.artifact_path) || !validSHA256(provenance.artifact_sha256)) {
			fail(`${unit}: cited unauthenticated owner authority`);
		}
	} else {
		fail(`${unit}: owner authority provenance kind`);
	}
}

function nonProductMaintenancePath(path) {
	if (!validPath(path) || nonProductMaintenanceForbiddenAuthorityPaths.has(path) ||
		/^(?:cmd|examples?|internal|schemas?|testkit|web)\//u.test(path) ||
		/(?:^|\/)(?:go\.(?:mod|sum)|package(?:-lock)?\.json|pnpm-lock\.yaml|yarn\.lock)$/u.test(path)) {
		return false;
	}
	if (/^(?:docs\/(?:HANDOFF_MODE_C\.md|PROMPT_PACK\.md|prompts\/|status\/)|research\/)/u.test(path)) {
		return true;
	}
	return /^tools\/(?:check|verify)-[a-z0-9-]+-maintenance(?:-selftest)?\.mjs$/u.test(path);
}

function nonProductMaintenanceClaimLabel(unit, role) {
	const suffix = nonProductMaintenanceClaimSuffixByRole[role];
	if (suffix === undefined) fail(`${unit}: unknown non-product maintenance claim role`);
	return `P07B-C ${unit} ${suffix}`;
}

function validateMaintenanceClaims(unit, claims) {
	if (!Array.isArray(claims) || claims.length !== nonProductMaintenanceClaimRoles.length) {
		fail(`${unit}: non-product maintenance claim count`);
	}
	const labels = new Set();
	for (let index = 0; index < claims.length; index += 1) {
		const claim = claims[index];
		if (!exactObjectKeys(claim, ["label", "role", "type"]) ||
			claim.role !== nonProductMaintenanceClaimRoles[index] ||
			claim.label !== nonProductMaintenanceClaimLabel(unit, claim.role) || labels.has(claim.label) ||
			(index < 6 ? claim.type !== "tests-pass" : claim.type !== "command-succeeded")) {
			fail(`${unit}: non-product maintenance claim ${index}`);
		}
		labels.add(claim.label);
	}
}

function validateNonProductMaintenanceEntry(unit, entry) {
	const statusPrefix = `docs/status/P07B-C-${unit}-`;
	const statusPaths = entry.exact.filter((path) => path.startsWith(statusPrefix) && path.endsWith(".md"));
	if (entry.product_authority !== "NONE" || entry.product_behavior !== "INHERITED_UNREPROVEN" ||
		entry.product_projection !== "PARENT_FROZEN" || entry.prefixes.length !== 0 ||
		entry.exact.some((path) => !nonProductMaintenancePath(path)) ||
		entry.exact.some((path) => nonProductMaintenanceForbiddenAuthorityPaths.has(path)) ||
		!entry.exact.includes(receiptPhaseHandoffPath) || statusPaths.length !== 1) {
		fail(`${unit}: non-product maintenance authority ceiling`);
	}
	validateMaintenanceClaims(unit, entry.maintenance_claims);
}

function validateNonProductMaintenanceParentProfile(row, predecessor) {
	if (row.profile === "NON_PRODUCT_MAINTENANCE" && predecessor.profile !== "SOURCE_FULL") {
		fail(`${row.boundary}: non-product maintenance requires a SOURCE_FULL direct parent`);
	}
}

export function receiptPhaseAuthorityDigest(specification) {
	const records = receiptPhaseUnitOrder.map((boundary) => {
		const entry = specification.units[boundary];
		return {
			boundary,
			parent: entry.parent,
			profile: entry.verification_profile,
			exact: entry.exact,
			prefixes: entry.prefixes,
			receipts: Object.fromEntries(receiptPhaseKeys.map((key) => [key, entry.receipt_states[key]])),
			receipt_claims: entry.receipt_claims ?? [],
			maintenance_claims: entry.maintenance_claims ?? [],
			product_authority: entry.product_authority ?? null,
			product_behavior: entry.product_behavior ?? null,
			product_projection: entry.product_projection ?? null,
			transition_authority: entry.transition_authority ?? null,
		};
	});
	return createHash("sha256")
		.update(`${records.map((record) => JSON.stringify(record)).join("\n")}\n`, "utf8")
		.digest("hex");
}

export function validateSpecification(specification) {
	if (!specification || typeof specification !== "object" || Array.isArray(specification) ||
		JSON.stringify(Object.keys(specification).sort()) !== JSON.stringify(["schema_version", "units"])) {
		fail("specification root roster");
	}
	if (specification.schema_version !== "countershape/p07b-c-unit-paths/v28") fail("specification version");
	if (!specification.units || typeof specification.units !== "object" || Array.isArray(specification.units) ||
		JSON.stringify(Object.keys(specification.units)) !== JSON.stringify(unitOrder)) fail("unit roster/order");

	for (const unit of unitOrder) {
		const entry = specification.units[unit];
		if (!entry || typeof entry !== "object" || Array.isArray(entry)) fail(`${unit}: field roster`);
		if (!verificationProfiles.has(entry.verification_profile)) fail(`${unit}: verification profile`);
		const baseExpectedFields = entry.verification_profile === "RECEIPT_RECONCILIATION"
			? ["exact", "prefixes", "receipt_claims", "verification_profile"]
			: entry.verification_profile === "NON_PRODUCT_MAINTENANCE"
				? [
					"exact", "maintenance_claims", "prefixes", "product_authority", "product_behavior",
					"product_projection", "verification_profile",
				]
				: ["exact", "prefixes", "verification_profile"];
		const expectedFields = receiptPhaseUnitSet.has(unit)
			? [
				...baseExpectedFields, "parent", "receipt_states",
				...(ownerOutOfBandUnits.has(unit) ? ["transition_authority"] : []),
			].sort()
			: baseExpectedFields;
		if (JSON.stringify(Object.keys(entry).sort()) !== JSON.stringify(expectedFields)) fail(`${unit}: field roster`);
		if (receiptPhaseUnitSet.has(unit)) {
			const expectedParent = unitOrder[unitOrder.indexOf(unit) - 1];
			if (entry.parent !== expectedParent) fail(`${unit}: phase parent`);
			if (!entry.receipt_states || typeof entry.receipt_states !== "object" || Array.isArray(entry.receipt_states) ||
				JSON.stringify(Object.keys(entry.receipt_states)) !== JSON.stringify(receiptPhaseKeys) ||
				!receiptPhaseKeys.every((key) => receiptPhaseStates.has(entry.receipt_states[key]))) {
				fail(`${unit}: receipt states`);
			}
			if (!entry.exact.includes("docs/HANDOFF_MODE_C.md")) fail(`${unit}: phase row omits HANDOFF cursor`);
		}
		if (!sortedUnique(entry.exact) || !entry.exact.every((path) => validPath(path))) fail(`${unit}: exact paths`);
		if (!sortedUnique(entry.prefixes) || !entry.prefixes.every((path) => validPath(path, true))) fail(`${unit}: prefixes`);
		if (ownerOutOfBandUnits.has(unit)) {
			validateTransitionAuthority(unit, entry.transition_authority);
			if (entry.transition_authority.provenance.kind !== knownOwnerOutOfBandProvenance[unit]) {
				fail(`${unit}: frozen historical owner-authority provenance`);
			}
			for (const reference of [
				entry.transition_authority.predecessor_declaration,
				entry.transition_authority.provenance,
			]) {
				if (typeof reference?.artifact_path === "string" && entry.exact.includes(reference.artifact_path)) {
					fail(`${unit}: candidate-owned transition-authority artifact`);
				}
			}
		}
		if (entry.verification_profile === "NON_PRODUCT_MAINTENANCE") {
			validateNonProductMaintenanceEntry(unit, entry);
		}
		if (entry.verification_profile === "RECEIPT_RECONCILIATION" &&
			(entry.prefixes.length !== 0 || entry.exact.some((path) =>
				!markdownAuthorityPath(path) && !(/^spec\/verification\/[a-z0-9-]+-receipt\.json$/u.test(path))))) {
			fail(`${unit}: receipt reconciliation scope`);
		}
		if (entry.verification_profile === "RECEIPT_RECONCILIATION") {
			const testsClaimCount = unit === "C6B" ? 5 : 3;
			if (!Array.isArray(entry.receipt_claims) || entry.receipt_claims.length !== (unit === "C6B" ? 9 : 7)) {
				fail(`${unit}: receipt claim count`);
			}
			const labels = new Set();
			for (let index = 0; index < entry.receipt_claims.length; index += 1) {
				const claim = entry.receipt_claims[index];
				if (!claim || typeof claim !== "object" || Array.isArray(claim) ||
					JSON.stringify(Object.keys(claim).sort()) !== JSON.stringify(["label", "type"]) ||
					typeof claim.label !== "string" || claim.label.length === 0 || claim.label.length > 240 ||
					/[\u0000-\u001f\u007f]/u.test(claim.label) || labels.has(claim.label) ||
					!claim.label.startsWith(`P07B-C ${unit} `) || !receiptClaimTypes.has(claim.type) ||
					(index < testsClaimCount ? claim.type !== "tests-pass" : claim.type !== "command-succeeded") ||
					/\b(?:cumulative|suite|runtime|security)\b|unchanged[- ]behavior/iu.test(claim.label)) {
					fail(`${unit}: receipt claim ${index}`);
				}
				labels.add(claim.label);
			}
		}
		for (const exact of entry.exact) {
			if (entry.prefixes.some((prefix) => exact.startsWith(prefix))) fail(`${unit}: exact path redundantly covered by prefix: ${exact}`);
		}
		const frozen = c3FrozenUnitContracts[unit];
		if (frozen !== undefined && (entry.verification_profile !== frozen.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(frozen.prefixes) ||
			exactRosterDigest(entry.exact) !== frozen.exact_roster_sha256 ||
			(frozen.receipt_claims !== undefined && JSON.stringify(entry.receipt_claims) !== JSON.stringify(frozen.receipt_claims)))) {
			fail(`${unit}: frozen exact unit contract`);
		}
		if (unit === "C4V" && (entry.verification_profile !== c4vDeclaredMaintenanceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c4vDeclaredMaintenanceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c4vDeclaredMaintenanceContract.exact) ||
			exactRosterDigest(entry.exact) !== c4vDeclaredMaintenanceContract.exact_roster_sha256)) {
			fail("C4V: declared maintenance contract");
		}
		if (unit === "C4M" && (entry.verification_profile !== c4mDeclaredMaintenanceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c4mDeclaredMaintenanceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c4mDeclaredMaintenanceContract.exact) ||
			exactRosterDigest(entry.exact) !== c4mDeclaredMaintenanceContract.exact_roster_sha256)) {
			fail("C4M: declared maintenance contract");
		}
		if (unit === "C4N" && (entry.verification_profile !== c4nDeclaredMaintenanceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c4nDeclaredMaintenanceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c4nDeclaredMaintenanceContract.exact) ||
			exactRosterDigest(entry.exact) !== c4nDeclaredMaintenanceContract.exact_roster_sha256)) {
			fail("C4N: declared maintenance contract");
		}
		if (unit === "C4P" && (entry.verification_profile !== c4pDeclaredMaintenanceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c4pDeclaredMaintenanceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c4pDeclaredMaintenanceContract.exact) ||
			exactRosterDigest(entry.exact) !== c4pDeclaredMaintenanceContract.exact_roster_sha256)) {
			fail("C4P: declared maintenance contract");
		}
		if (unit === "C4" && (entry.verification_profile !== c4DeclaredSourceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c4DeclaredSourceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c4DeclaredSourceContract.exact) ||
			exactRosterDigest(entry.exact) !== c4DeclaredSourceContract.exact_roster_sha256 ||
			exactRosterDigest(entry.prefixes) !== c4DeclaredSourceContract.prefix_roster_sha256)) {
			fail("C4: declared source contract");
		}
		if (unit === "C4H" && (entry.verification_profile !== c4hDeclaredMaintenanceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c4hDeclaredMaintenanceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c4hDeclaredMaintenanceContract.exact) ||
			exactRosterDigest(entry.exact) !== c4hDeclaredMaintenanceContract.exact_roster_sha256)) {
			fail("C4H: declared maintenance contract");
		}
		if (unit === "C4I" && (entry.verification_profile !== c4iDeclaredMaintenanceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c4iDeclaredMaintenanceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c4iDeclaredMaintenanceContract.exact) ||
			exactRosterDigest(entry.exact) !== c4iDeclaredMaintenanceContract.exact_roster_sha256)) {
			fail("C4I: declared maintenance contract");
		}
		if (unit === "C4K" && (entry.verification_profile !== c4kDeclaredMaintenanceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c4kDeclaredMaintenanceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c4kDeclaredMaintenanceContract.exact) ||
			exactRosterDigest(entry.exact) !== c4kDeclaredMaintenanceContract.exact_roster_sha256)) {
			fail("C4K: declared maintenance contract");
		}
		if (unit === "C4J" && (entry.verification_profile !== c4jDeclaredMaintenanceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c4jDeclaredMaintenanceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c4jDeclaredMaintenanceContract.exact) ||
			exactRosterDigest(entry.exact) !== c4jDeclaredMaintenanceContract.exact_roster_sha256)) {
			fail("C4J: declared maintenance contract");
		}
		if (unit === "C4L" && (entry.verification_profile !== c4lDeclaredMaintenanceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c4lDeclaredMaintenanceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c4lDeclaredMaintenanceContract.exact) ||
			exactRosterDigest(entry.exact) !== c4lDeclaredMaintenanceContract.exact_roster_sha256)) {
			fail("C4L: declared maintenance contract");
		}
		if (unit === "C5" && (entry.verification_profile !== c5DeclaredSourceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c5DeclaredSourceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c5DeclaredSourceContract.exact) ||
			exactRosterDigest(entry.exact) !== c5DeclaredSourceContract.exact_roster_sha256 ||
			exactRosterDigest(entry.prefixes) !== c5DeclaredSourceContract.prefix_roster_sha256)) {
			fail("C5: declared source contract");
		}
		if (unit === "C5V" && (entry.verification_profile !== c5vDeclaredSourceMaintenanceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c5vDeclaredSourceMaintenanceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c5vDeclaredSourceMaintenanceContract.exact) ||
			exactRosterDigest(entry.exact) !== c5vDeclaredSourceMaintenanceContract.exact_roster_sha256)) {
			fail("C5V: declared source maintenance contract");
		}
		if (unit === "C6F" && (entry.verification_profile !== c6fDeclaredMaintenanceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c6fDeclaredMaintenanceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c6fDeclaredMaintenanceContract.exact) ||
			exactRosterDigest(entry.exact) !== c6fDeclaredMaintenanceContract.exact_roster_sha256)) {
			fail("C6F: declared maintenance contract");
		}
		if (unit === "C6G" && (entry.verification_profile !== c6gDeclaredMaintenanceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c6gDeclaredMaintenanceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c6gDeclaredMaintenanceContract.exact) ||
			exactRosterDigest(entry.exact) !== c6gDeclaredMaintenanceContract.exact_roster_sha256)) {
			fail("C6G: declared maintenance contract");
		}
		if (unit === "C6B" && (entry.verification_profile !== c6bDeclaredReceiptContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c6bDeclaredReceiptContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c6bDeclaredReceiptContract.exact) ||
			exactRosterDigest(entry.exact) !== c6bDeclaredReceiptContract.exact_roster_sha256 ||
			JSON.stringify(entry.receipt_claims) !== JSON.stringify(c6bDeclaredReceiptContract.receipt_claims))) {
			fail("C6B: declared future receipt contract");
		}
		if (unit === "C6M" && (entry.verification_profile !== c6mDeclaredAdapterContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c6mDeclaredAdapterContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c6mDeclaredAdapterContract.exact) ||
			exactRosterDigest(entry.exact) !== c6mDeclaredAdapterContract.exact_roster_sha256)) {
			fail("C6M: declared future adapter contract");
		}
	}
	const rows = receiptPhaseRows(specification);
	if (!receiptPhaseKeys.every((key) => rows[0].receipts[key] === "ABSENT")) {
		fail(`${rows[0].boundary}: initial receipt states`);
	}
	const signatures = new Set();
	for (let index = 0; index < rows.length; index += 1) {
		const row = rows[index];
		const signature = JSON.stringify(row);
		if (signatures.has(signature)) fail(`${row.boundary}: duplicate phase row`);
		signatures.add(signature);
		if (index === 0) continue;
		const predecessor = rows[index - 1];
		validateNonProductMaintenanceParentProfile(row, predecessor);
		const changed = receiptPhaseKeys.filter((key) => predecessor.receipts[key] !== row.receipts[key]);
		if (changed.some((key) => predecessor.receipts[key] !== "ABSENT" || row.receipts[key] !== "PRESENT")) {
			fail(`${row.boundary}: receipt state downgrade`);
		}
		const expectedChanges = row.profile === "RECEIPT_RECONCILIATION" ? 1 : 0;
		if (changed.length !== expectedChanges) fail(`${row.boundary}: phase/profile receipt delta`);
	}
	if (receiptPhaseAuthorityDigest(specification) !== receiptPhaseAuthoritySHA256) {
		fail("receipt phase authority digest");
	}
	return specification;
}

export function receiptPhaseRows(specification) {
	return Object.freeze(receiptPhaseUnitOrder.map((boundary) => {
		const entry = specification.units[boundary];
		return Object.freeze({
			boundary,
			parent: entry.parent,
			profile: entry.verification_profile,
			product_authority: entry.product_authority ?? null,
			product_behavior: entry.product_behavior ?? null,
			product_projection: entry.product_projection ?? null,
			transition_authority: entry.transition_authority ?? null,
			receipts: Object.freeze(Object.fromEntries(receiptPhaseKeys.map((key) => [key, entry.receipt_states[key]]))),
		});
	}));
}

export function activeReceiptPhaseBoundary(specification, boundary) {
	if (!specification || typeof specification !== "object" ||
		typeof boundary !== "string" || !receiptPhaseUnitSet.has(boundary) ||
		!receiptPhaseRows(specification).some((row) => row.boundary === boundary)) {
		fail("active receipt phase boundary");
	}
	return boundary;
}

function countOccurrences(body, needle) {
	if (needle.length === 0) fail("empty receipt-phase occurrence needle");
	let count = 0;
	let offset = 0;
	while ((offset = body.indexOf(needle, offset)) !== -1) {
		count += 1;
		offset += needle.length;
	}
	return count;
}

function visibleCandidateMarkdown(body) {
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
			fail("candidate HANDOFF raw HTML block syntax");
		}
		const opening = /^ {0,3}(`{3,}|~{3,})/u.exec(line);
		if (opening) {
			fence = { character: opening[1][0], length: opening[1].length };
			visible.push("");
			continue;
		}
		visible.push(line);
	}
	if (htmlComment) fail("candidate HANDOFF unterminated Markdown HTML comment");
	if (fence !== undefined) fail("candidate HANDOFF unterminated Markdown fence");
	return visible.join("\n");
}

function renderCandidateReceiptPhaseCapsule(row) {
	return [
		receiptPhaseCapsuleStart,
		"### Active P07B-C phase contract",
		"",
		`- **Boundary:** \`${row.boundary}\``,
		`- **Parent:** \`${row.parent}\``,
		`- **Verification profile:** \`${row.profile}\``,
		`- **Receipt C3P:** \`${row.receipts.C3P}\``,
		`- **Receipt C3:** \`${row.receipts.C3}\``,
		`- **Receipt C6A:** \`${row.receipts.C6A}\``,
		receiptPhaseCapsuleEnd,
	].join("\n");
}

function classifyCandidateReceiptPhase(specification, body) {
	if (countOccurrences(body, receiptPhaseCapsuleStart) !== 1 ||
		countOccurrences(body, receiptPhaseCapsuleEnd) !== 1) {
		fail("candidate receipt phase capsule marker cardinality");
	}
	const start = body.indexOf(receiptPhaseCapsuleStart);
	const end = body.indexOf(receiptPhaseCapsuleEnd, start);
	if (end < start || (start !== 0 && body[start - 1] !== "\n") ||
		body.slice(start + receiptPhaseCapsuleStart.length, start + receiptPhaseCapsuleStart.length + 1) !== "\n" ||
		(end !== 0 && body[end - 1] !== "\n") ||
		(body.slice(end + receiptPhaseCapsuleEnd.length, end + receiptPhaseCapsuleEnd.length + 1) !== "\n" &&
			end + receiptPhaseCapsuleEnd.length !== body.length)) {
		fail("candidate receipt phase capsule marker framing");
	}
	const currentStateAnchor = "## Current state\n";
	if (countOccurrences(body, currentStateAnchor) !== 1) fail("candidate HANDOFF current-state heading cardinality");
	const currentStateStart = body.indexOf(currentStateAnchor) + currentStateAnchor.length;
	const nextHeading = body.indexOf("\n## ", currentStateStart);
	const currentStateEnd = nextHeading === -1 ? body.length : nextHeading;
	if (start < currentStateStart || end >= currentStateEnd) fail("candidate receipt phase capsule outside current state");
	const startSentinel = "COUNTERSHAPE_SCOPE_VISIBLE_RECEIPT_PHASE_START";
	const endSentinel = "COUNTERSHAPE_SCOPE_VISIBLE_RECEIPT_PHASE_END";
	if (body.includes(startSentinel) || body.includes(endSentinel)) fail("candidate receipt phase visibility sentinel collision");
	const visible = visibleCandidateMarkdown(
		body.replace(receiptPhaseCapsuleStart, startSentinel).replace(receiptPhaseCapsuleEnd, endSentinel),
	);
	if (countOccurrences(visible, startSentinel) !== 1 || countOccurrences(visible, endSentinel) !== 1 ||
		visible.indexOf(startSentinel) >= visible.indexOf(endSentinel)) {
		fail("candidate receipt phase capsule is not visible Markdown authority");
	}
	const block = body.slice(start, end + receiptPhaseCapsuleEnd.length);
	const matches = receiptPhaseRows(specification).filter((row) => block === renderCandidateReceiptPhaseCapsule(row));
	if (matches.length !== 1) fail("candidate receipt phase capsule row mismatch");
	return matches[0];
}

function decodeCandidateUTF8(bytes, label) {
	if (!Buffer.isBuffer(bytes)) fail(`${label}: not bytes`);
	if (bytes.length >= 3 && bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf) {
		fail(`${label}: UTF-8 BOM is not canonical`);
	}
	try { return new TextDecoder("utf-8", { fatal: true }).decode(bytes); }
	catch (error) { fail(`${label}: invalid UTF-8 (${error.message})`); }
}

function decodeCandidateGitOID(bytes, label) {
	const oid = decodeCandidateUTF8(bytes, label).trim();
	if (!/^(?:[0-9a-f]{40}|[0-9a-f]{64})$/u.test(oid) || /^0+$/u.test(oid)) fail(`${label}: invalid Git object ID`);
	return oid;
}

async function readCandidateWorktreeBytes(path, expectedMode = "100644") {
	const rootReal = await realpath(repositoryRoot);
	const parts = path.split("/");
	let cursor = repositoryRoot;
	for (let index = 0; index < parts.length - 1; index += 1) {
		cursor = resolve(cursor, parts[index]);
		const ancestor = await lstat(cursor);
		if (!ancestor.isDirectory() || ancestor.isSymbolicLink() ||
			await realpath(cursor) !== resolve(rootReal, ...parts.slice(0, index + 1))) {
			fail(`candidate worktree ancestor is not a stable directory: ${path}`);
		}
	}
	const target = resolve(repositoryRoot, path);
	if (await realpath(target) !== resolve(rootReal, path)) fail(`candidate worktree path escapes repository: ${path}`);
	const handle = await open(target, fsConstants.O_RDONLY | fsConstants.O_NOFOLLOW);
	try {
		const before = await handle.stat({ bigint: true });
		if (!before.isFile() || before.size <= 0n || before.size > 8n * 1024n * 1024n) fail(`candidate worktree file bounds: ${path}`);
		if ((expectedMode === "100755") !== ((before.mode & 0o111n) !== 0n)) {
			fail(`candidate worktree executable mode mismatch: ${path}`);
		}
		const bytes = await handle.readFile();
		const after = await handle.stat({ bigint: true });
		for (const field of ["dev", "ino", "size", "mtimeNs", "ctimeNs"]) {
			if (before[field] !== after[field]) fail(`candidate worktree file changed during read: ${path}`);
		}
		return bytes;
	} finally {
		await handle.close();
	}
}

function exactCandidateGitLine(bytes, label) {
	const text = decodeCandidateUTF8(bytes, label);
	if (text.length === 0 || !text.endsWith("\n") || text.slice(0, -1).includes("\n") || text.includes("\r")) {
		fail(`${label}: not one exact LF-terminated line`);
	}
	return text.slice(0, -1);
}

function c6aSourceAuthorityRecord(authority) {
	return {
		schema_version: c6aSourceAuthoritySchema,
		source_commit: authority.commit,
		source_tree: authority.tree,
		source_parent: authority.parent,
		source_subject: c6aSourceSubject,
		note_ref: "refs/notes/didrun",
		note_type: "blob",
		note_blob: authority.noteBlob,
		note_body_sha256: authority.noteBodySHA256,
		secrets_override: authority.secretsOverride,
		claims: c6aSourceClaimManifest.map(({ label, type }) => ({ label, type, grade: "TREE-EXACT" })),
	};
}

function renderC6ASourceAuthorityManifest(authority) {
	return `${JSON.stringify(c6aSourceAuthorityRecord(authority), null, 2)}\n`;
}

function validateC6ASourceAuthorityManifest(bytes, authority) {
	const text = decodeCandidateUTF8(bytes, "C6A source-authority manifest");
	let manifest;
	try { manifest = JSON.parse(text); }
	catch (error) { fail(`C6A source-authority manifest JSON: ${error.message}`); }
	const expected = c6aSourceAuthorityRecord(authority);
	if (!isDeepStrictEqual(manifest, expected) || text !== `${JSON.stringify(manifest, null, 2)}\n`) {
		fail("C6A source-authority manifest canonical authority mismatch");
	}
	return Object.freeze(manifest);
}

function renderC6ASourceReceiptBlock(manifest) {
	return [
		c6aReceiptBlockStart,
		"### Sealed C6A source receipts",
		"",
		`- **Source commit:** \`${manifest.source_commit}\``,
		`- **Source tree:** \`${manifest.source_tree}\``,
		`- **Source parent:** \`${manifest.source_parent}\``,
		`- **Source subject:** \`${manifest.source_subject}\``,
		`- **Didrun note:** \`${manifest.note_blob}\` (body SHA-256 \`${manifest.note_body_sha256}\`)`,
		`- **Seal override disclosed:** \`${String(manifest.secrets_override)}\``,
		"",
		"| # | Claim | Type | Grade |",
		"|---:|---|---|---|",
		...manifest.claims.map((claim, index) =>
			`| ${index + 1} | \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`),
		c6aReceiptBlockEnd,
	].join("\n");
}

function c6aReceiptProjectionBlock(body, manifest, label) {
	const starts = countOccurrences(body, c6aReceiptBlockStart);
	const ends = countOccurrences(body, c6aReceiptBlockEnd);
	if (starts === 0 && ends === 0) return undefined;
	if (starts !== 1 || ends !== 1) fail(`${label}: C6A source-receipt marker cardinality`);
	const start = body.indexOf(c6aReceiptBlockStart);
	const end = body.indexOf(c6aReceiptBlockEnd, start);
	if (start < 0 || end < start || (start !== 0 && body[start - 1] !== "\n") ||
		(end + c6aReceiptBlockEnd.length !== body.length && body[end + c6aReceiptBlockEnd.length] !== "\n")) {
		fail(`${label}: C6A source-receipt block framing`);
	}
	const startSentinel = "COUNTERSHAPE_SCOPE_VISIBLE_C6A_RECEIPT_START";
	const endSentinel = "COUNTERSHAPE_SCOPE_VISIBLE_C6A_RECEIPT_END";
	if (body.includes(startSentinel) || body.includes(endSentinel)) fail(`${label}: C6A receipt sentinel collision`);
	const visible = visibleCandidateMarkdown(
		body.replace(c6aReceiptBlockStart, startSentinel).replace(c6aReceiptBlockEnd, endSentinel),
	);
	if (countOccurrences(visible, startSentinel) !== 1 || countOccurrences(visible, endSentinel) !== 1 ||
		visible.indexOf(startSentinel) >= visible.indexOf(endSentinel)) {
		fail(`${label}: C6A source-receipt block is not visible Markdown authority`);
	}
	const block = body.slice(start, end + c6aReceiptBlockEnd.length);
	for (const required of [
		"### Sealed C6A source receipts",
		`- **Source commit:** \`${manifest.source_commit}\``,
		`- **Source tree:** \`${manifest.source_tree}\``,
		`- **Source parent:** \`${manifest.source_parent}\``,
		`- **Source subject:** \`${manifest.source_subject}\``,
		`- **Didrun note:** \`${manifest.note_blob}\` (body SHA-256 \`${manifest.note_body_sha256}\`)`,
		`- **Seal override disclosed:** \`${String(manifest.secrets_override)}\``,
		"| # | Claim | Type | Grade |",
		"|---:|---|---|---|",
		...manifest.claims.map((claim, index) =>
			`| ${index + 1} | \`${claim.label}\` | \`${claim.type}\` | \`${claim.grade}\` |`),
	]) {
		if (countOccurrences(block, required) !== 1) fail(`${label}: C6A portable source-receipt field mismatch`);
	}
	return block;
}

function c6aReceiptProjectionState(body, manifest, label) {
	return c6aReceiptProjectionBlock(body, manifest, label) === undefined ? "ABSENT" : "PRESENT";
}

function validateSealedSourceClaimPreview(argv, expectedArgv, hermeticPrefix) {
	if (!Array.isArray(argv) || !Array.isArray(expectedArgv) || !Array.isArray(hermeticPrefix) ||
		argv.length !== expectedArgv.length || expectedArgv.length <= hermeticPrefix.length ||
		!isDeepStrictEqual(expectedArgv.slice(0, hermeticPrefix.length), hermeticPrefix) ||
		argv.some((part) => typeof part !== "string" || part.length === 0 || part.length > 4096 ||
			/[\u0000-\u001f\u007f]/u.test(part))) {
		return false;
	}
	for (let position = 0; position < hermeticPrefix.length; position += 1) {
		if (argv[position] === expectedArgv[position]) continue;
		if (argv[position] !== sealedSourceExpectedRedactionProjection(position, expectedArgv[position])) return false;
	}
	return isDeepStrictEqual(argv.slice(hermeticPrefix.length), expectedArgv.slice(hermeticPrefix.length));
}

function validateDidrunSourceNote(note, identity, claims, expectedClaimArgv, hermeticPrefix, label) {
	const rootKeys = ["claims", "commit", "coverage", "secrets_override", "tree", "version"];
	if (!Array.isArray(claims) || !Array.isArray(expectedClaimArgv) || expectedClaimArgv.length !== claims.length ||
		!Array.isArray(hermeticPrefix) || hermeticPrefix.length === 0) {
		fail(`${label}: didrun note command plan`);
	}
	if (!note || typeof note !== "object" || Array.isArray(note) ||
		!isDeepStrictEqual(Object.keys(note).sort(), rootKeys) || note.version !== 1 ||
		note.commit !== identity.commit || note.tree !== identity.tree || typeof note.secrets_override !== "boolean" ||
		!Array.isArray(note.claims) || note.claims.length !== claims.length) {
		fail(`${label}: didrun note root/identity/claim roster`);
	}
	for (let index = 0; index < claims.length; index += 1) {
		const recorded = note.claims[index];
		const claim = recorded?.claim;
		const expected = claims[index];
		const expectedArgv = expectedClaimArgv[index];
		if (!recorded || typeof recorded !== "object" || Array.isArray(recorded) ||
			!isDeepStrictEqual(Object.keys(recorded).sort(), ["claim", "delta", "exit_code", "grade", "reason", "supporting_event_index"]) ||
			!claim || typeof claim !== "object" || Array.isArray(claim) ||
			!isDeepStrictEqual(Object.keys(claim).sort(), ["argv_preview", "ctype", "declared_at_index", "event_indices", "label", "pathspecs"]) ||
			claim.label !== expected.label || claim.ctype !== expected.type || claim.declared_at_index !== index ||
			!isDeepStrictEqual(claim.event_indices, [index]) || !isDeepStrictEqual(claim.pathspecs, []) ||
			!validateSealedSourceClaimPreview(claim.argv_preview, expectedArgv, hermeticPrefix) ||
			recorded.supporting_event_index !== index ||
			recorded.grade !== "tree-exact" || recorded.exit_code !== 0 ||
			recorded.reason !== "self-stable command ran against the sealed tree" || !isDeepStrictEqual(recorded.delta, [])) {
			fail(`${label}: didrun note claim ${index + 1}`);
		}
	}
	if (!note.coverage || typeof note.coverage !== "object" || Array.isArray(note.coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage).sort(), ["by_coverage", "total_events"]) ||
		note.coverage.total_events !== claims.length || !note.coverage.by_coverage ||
		typeof note.coverage.by_coverage !== "object" || Array.isArray(note.coverage.by_coverage) ||
		!isDeepStrictEqual(Object.keys(note.coverage.by_coverage), ["complete"]) ||
		note.coverage.by_coverage.complete !== claims.length) {
		fail(`${label}: didrun note exact event coverage`);
	}
	return note.secrets_override;
}

function syntheticDidrunSourceNote(identity, claims, expectedClaimArgv) {
	return {
		claims: claims.map((claim, index) => ({
			claim: {
				argv_preview: [...expectedClaimArgv[index]],
				ctype: claim.type, declared_at_index: index, event_indices: [index], label: claim.label, pathspecs: [],
			},
			delta: [], exit_code: 0, grade: "tree-exact",
			reason: "self-stable command ran against the sealed tree", supporting_event_index: index,
		})),
		commit: identity.commit,
		coverage: { by_coverage: { complete: claims.length }, total_events: claims.length },
		secrets_override: true,
		tree: identity.tree,
		version: 1,
	};
}

function decodeNULPaths(bytes, label) {
	if (!Buffer.isBuffer(bytes) || (bytes.length > 0 && bytes.at(-1) !== 0)) fail(`${label}: NUL framing`);
	let decoded;
	try {
		decoded = new TextDecoder("utf-8", { fatal: true }).decode(bytes.length === 0 ? bytes : bytes.subarray(0, -1));
	} catch (error) { fail(`${label}: invalid UTF-8 (${error.message})`); }
	const paths = decoded.length === 0 ? [] : decoded.split("\0");
	if (new Set(paths).size !== paths.length || paths.some((path) => !validPath(path))) fail(`${label}: path roster`);
	return paths.sort();
}

async function deriveSealedSourceAuthority(
	specification, commit, unit, subject, claims, expectedClaimArgv, hermeticPrefix,
) {
	const reopened = decodeCandidateGitOID(await gitOutput(["rev-parse", "--verify", `${commit}^{commit}`]), `${unit} commit`);
	if (reopened !== commit) fail(`${unit}: commit did not reopen exactly`);
	const tree = decodeCandidateGitOID(await gitOutput(["rev-parse", "--verify", `${commit}^{tree}`]), `${unit} tree`);
	const parentLine = exactCandidateGitLine(await gitOutput(["show", "-s", "--format=%P", commit]), `${unit} parent line`);
	const parents = parentLine.length === 0 ? [] : parentLine.split(" ");
	if (parents.length !== 1 || !/^(?:[0-9a-f]{40}|[0-9a-f]{64})$/u.test(parents[0])) fail(`${unit}: exact single parent`);
	const observedSubject = exactCandidateGitLine(await gitOutput(["show", "-s", "--format=%s", commit]), `${unit} subject`);
	if (observedSubject !== subject) fail(`${unit}: source subject`);
	const paths = decodeNULPaths(await gitOutput([
		"diff-tree", "--no-commit-id", "--name-only", "-r", "-z", "--no-renames", parents[0], commit, "--",
	]), `${unit} source diff`);
	const entry = specification.units[unit];
	if (entry.exact.some((path) => !paths.includes(path)) || unexpectedPaths(specification, unit, paths).length !== 0) {
		fail(`${unit}: sealed source diff scope`);
	}
	const noteBlob = exactCandidateGitLine(
		await gitOutput(["notes", "--ref=didrun", "list", commit]), `${unit} didrun note listing`,
	);
	if (!/^(?:[0-9a-f]{40}|[0-9a-f]{64})$/u.test(noteBlob) || /^0+$/u.test(noteBlob)) fail(`${unit}: didrun note blob`);
	const noteType = exactCandidateGitLine(await gitOutput(["cat-file", "-t", noteBlob]), `${unit} didrun note type`);
	if (noteType !== "blob") fail(`${unit}: didrun note is not a blob`);
	const noteBytes = await gitOutput(["cat-file", "blob", noteBlob]);
	let note;
	try { note = JSON.parse(decodeCandidateUTF8(noteBytes, `${unit} didrun note body`)); }
	catch (error) { fail(`${unit}: didrun note JSON (${error.message})`); }
	const secretsOverride = validateDidrunSourceNote(
		note, { commit, tree }, claims, expectedClaimArgv, hermeticPrefix, unit,
	);
	const stableNoteBlob = exactCandidateGitLine(
		await gitOutput(["notes", "--ref=didrun", "list", commit]), `${unit} stable didrun note listing`,
	);
	if (stableNoteBlob !== noteBlob) fail(`${unit}: didrun note changed during validation`);
	return Object.freeze({
		commit, tree, parent: parents[0], subject, noteBlob,
		noteBodySHA256: createHash("sha256").update(noteBytes).digest("hex"), secretsOverride,
	});
}

async function treePathBytes(tree, path, label) {
	const oid = decodeCandidateGitOID(await gitOutput(["rev-parse", "--verify", `${tree}:${path}`]), `${label} blob`);
	return Object.freeze({ oid, bytes: await gitOutput(["cat-file", "blob", oid]) });
}

async function treeHasPath(tree, path) {
	return (await gitOutput(["ls-tree", "-z", tree, "--", path])).length !== 0;
}

function validateFutureC6Material({
	boundary, authority, manifestBytes, parentManifestBytes, handoffBody, evidenceBody, changedPaths,
}) {
	if (boundary === "C6A" || boundary === "C6F" || boundary === "C6G") {
		if (manifestBytes !== undefined) fail(`${boundary}: premature C6A source-authority manifest`);
		if (c6aReceiptProjectionState(handoffBody, {}, "C6A HANDOFF") !== "ABSENT" ||
			c6aReceiptProjectionState(evidenceBody, {}, "C6A evidence") !== "ABSENT") {
			fail(`${boundary}: premature source-receipt projection`);
		}
		return undefined;
	}
	if (boundary !== "C6M" && boundary !== "C6B") fail(`future C6 material boundary: ${boundary}`);
	if (!Buffer.isBuffer(manifestBytes) || authority === undefined) fail(`${boundary}: C6A source-authority manifest absent`);
	const manifest = validateC6ASourceAuthorityManifest(manifestBytes, authority);
	if (boundary === "C6M") {
		if (parentManifestBytes !== undefined) fail("C6M: source-authority manifest pre-existed in C6G");
		if (!isDeepStrictEqual(changedPaths, c6mDeclaredAdapterContract.exact)) {
			fail("C6M: exact source-authority staged roster");
		}
		if (c6aReceiptProjectionState(handoffBody, manifest, "C6M HANDOFF") !== "ABSENT" ||
			c6aReceiptProjectionState(evidenceBody, manifest, "C6M evidence") !== "ABSENT") {
			fail("C6M: source-authority manifest cannot project C6A receipt presence");
		}
	} else {
		if (!Buffer.isBuffer(parentManifestBytes) || !parentManifestBytes.equals(manifestBytes)) {
			fail("C6B: inherited source-authority drift");
		}
		if (!isDeepStrictEqual(changedPaths, c6bDeclaredReceiptContract.exact) || changedPaths.includes(c6aSourceAuthorityPath)) {
			fail("C6B: exact receipt staged roster or inherited-manifest mutation");
		}
		if (c6aReceiptProjectionState(handoffBody, manifest, "C6B HANDOFF") !== "PRESENT" ||
			c6aReceiptProjectionState(evidenceBody, manifest, "C6B evidence") !== "PRESENT") {
			fail("C6B: canonical dual source-receipt projection");
		}
		if (c6aReceiptProjectionBlock(handoffBody, manifest, "C6B HANDOFF") !==
			c6aReceiptProjectionBlock(evidenceBody, manifest, "C6B evidence")) {
			fail("C6B: source-receipt projections differ byte-for-byte");
		}
	}
	return manifest;
}

async function validateFutureC6Authority(specification, snapshot) {
	const boundary = snapshot.candidateBoundary;
	if (boundary !== "C6A" && boundary !== "C6F" && boundary !== "C6G" && boundary !== "C6M" && boundary !== "C6B") return;
	const candidateHandoff = decodeCandidateUTF8(snapshot.stagedHandoffBytes, `${boundary} candidate HANDOFF`);
	const evidence = await treePathBytes(snapshot.indexTree, c6EvidencePath, `${boundary} C6 evidence`);
	const workingEvidence = await readCandidateWorktreeBytes(c6EvidencePath);
	if (!evidence.bytes.equals(workingEvidence)) fail(`${boundary}: C6 evidence staged/worktree mismatch`);
	if (boundary === "C6A" || boundary === "C6F" || boundary === "C6G") {
		validateFutureC6Material({
			boundary,
			manifestBytes: await treeHasPath(snapshot.indexTree, c6aSourceAuthorityPath) ? Buffer.alloc(0) : undefined,
			handoffBody: candidateHandoff,
			evidenceBody: decodeCandidateUTF8(evidence.bytes, "C6A evidence"),
		});
		return;
	}
	let c6aAuthority;
	if (boundary === "C6M") {
		if (await treeHasPath(snapshot.parentTree, c6aSourceAuthorityPath)) fail("C6M: source-authority manifest pre-existed in C6G");
		const c6gAuthority = await deriveSealedSourceAuthority(
			specification, snapshot.parentCommit, "C6G", c6gSourceSubject, c6gSourceClaimManifest,
			c6gSourceExpectedClaimArgv, c6gSourceHermeticArgvPrefix,
		);
		const c6fAuthority = await deriveSealedSourceAuthority(
			specification, c6gAuthority.parent, "C6F", c6fSourceSubject, c6fSourceClaimManifest,
			c6fSourceExpectedClaimArgv, c6fSourceHermeticArgvPrefix,
		);
		c6aAuthority = await deriveSealedSourceAuthority(
			specification, c6fAuthority.parent, "C6A", c6aSourceSubject, c6aSourceClaimManifest,
			c6aSourceExpectedClaimArgv, c6aSourceHermeticArgvPrefix,
		);
	} else {
		const c6mAuthority = await deriveSealedSourceAuthority(
			specification, snapshot.parentCommit, "C6M", c6mSourceSubject, c6mSourceClaimManifest,
			c6mSourceExpectedClaimArgv, c6mSourceHermeticArgvPrefix,
		);
		const c6gAuthority = await deriveSealedSourceAuthority(
			specification, c6mAuthority.parent, "C6G", c6gSourceSubject, c6gSourceClaimManifest,
			c6gSourceExpectedClaimArgv, c6gSourceHermeticArgvPrefix,
		);
		const c6fAuthority = await deriveSealedSourceAuthority(
			specification, c6gAuthority.parent, "C6F", c6fSourceSubject, c6fSourceClaimManifest,
			c6fSourceExpectedClaimArgv, c6fSourceHermeticArgvPrefix,
		);
		c6aAuthority = await deriveSealedSourceAuthority(
			specification, c6fAuthority.parent, "C6A", c6aSourceSubject, c6aSourceClaimManifest,
			c6aSourceExpectedClaimArgv, c6aSourceHermeticArgvPrefix,
		);
	}
	const manifest = await treePathBytes(snapshot.indexTree, c6aSourceAuthorityPath, `${boundary} C6A source authority`);
	const entryMatches = snapshot.indexEntries.filter((entry) => entry.path === c6aSourceAuthorityPath);
	if (entryMatches.length !== 1 || entryMatches[0].mode !== "100644" || entryMatches[0].stage !== 0 ||
		entryMatches[0].object !== manifest.oid) fail(`${boundary}: C6A source-authority index binding`);
	const workingManifest = await readCandidateWorktreeBytes(c6aSourceAuthorityPath);
	if (!manifest.bytes.equals(workingManifest)) fail(`${boundary}: C6A source-authority staged/worktree mismatch`);
	const changedPaths = await stagedPaths();
	let parentManifest;
	if (boundary === "C6M") {
	} else {
		parentManifest = await treePathBytes(snapshot.parentTree, c6aSourceAuthorityPath, "C6B parent C6A source authority");
		if (parentManifest.oid !== manifest.oid) fail("C6B: inherited source-authority object drift");
	}
	validateFutureC6Material({
		boundary, authority: c6aAuthority, manifestBytes: manifest.bytes,
		parentManifestBytes: parentManifest?.bytes,
		handoffBody: candidateHandoff,
		evidenceBody: decodeCandidateUTF8(evidence.bytes, `${boundary} evidence`),
		changedPaths,
	});
	const stableManifest = await treePathBytes(snapshot.indexTree, c6aSourceAuthorityPath, `${boundary} stable C6A source authority`);
	if (stableManifest.oid !== manifest.oid || !stableManifest.bytes.equals(manifest.bytes) ||
		!workingManifest.equals(await readCandidateWorktreeBytes(c6aSourceAuthorityPath)) ||
		!workingEvidence.equals(await readCandidateWorktreeBytes(c6EvidencePath))) {
		fail(`${boundary}: source-authority inputs changed during validation`);
	}
}

function requireCandidateTransitionAuthorityStable(before, after) {
	for (const field of ["parentCommit", "parentTree", "indexTree"]) {
		if (before[field] !== after[field]) fail(`candidate ${field} changed during transition validation`);
	}
}

async function validateDirectParentArtifact(commit, reference, label) {
	const tree = decodeCandidateGitOID(
		await gitOutput(["rev-parse", "--verify", `${commit}^{tree}`]), `${label} direct-parent tree`,
	);
	const row = await gitOutput(["ls-tree", "-z", tree, "--", `:(literal)${reference.artifact_path}`]);
	const decoded = row.toString("utf8");
	const match = /^100644 blob ([0-9a-f]{40}|[0-9a-f]{64})\t([^\0]*)\0$/u.exec(decoded);
	if (!match || match[2] !== reference.artifact_path ||
		row.length !== Buffer.byteLength(decoded, "utf8")) {
		fail(`${label}: cited artifact is not one exact direct-parent mode-100644 blob`);
	}
	const bytes = await gitOutput(["cat-file", "blob", match[1]]);
	if (bytes.length === 0 || bytes.length > 64 * 1024) fail(`${label}: cited artifact byte ceiling`);
	decodeCandidateUTF8(bytes, `${label} cited artifact`);
	const digest = `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
	if (digest !== reference.artifact_sha256) fail(`${label}: cited artifact raw-byte digest`);
}

async function validateTransitionAuthorityArtifacts(specification, candidateBoundary, candidateParentCommit) {
	const c4jAuthority = specification.units.C4J.transition_authority;
	if (c4jAuthority.predecessor_declaration.kind === "PROSE_ONLY") {
		await validateDirectParentArtifact(
			sealedC4KCandidateParent.commit,
			c4jAuthority.predecessor_declaration,
			"C4J predecessor record",
		);
	}
	const candidateAuthority = specification.units[candidateBoundary]?.transition_authority;
	if (candidateAuthority?.provenance?.kind === "CITED_UNAUTHENTICATED") {
		if (typeof candidateParentCommit !== "string") {
			fail(`${candidateBoundary}: cited authority lacks exact direct-parent commit`);
		}
		await validateDirectParentArtifact(
			candidateParentCommit,
			candidateAuthority.provenance,
			`${candidateBoundary} owner instruction`,
		);
	}
}

function validateIndependentCandidateSnapshot(specification, snapshot) {
	const boundary = snapshot.candidateBoundary;
	if (!forwardCandidateReceiptPhaseBoundarySet.has(boundary)) fail(`candidate boundary outside forward horizon: ${boundary}`);
	const unit = specification.units[boundary];
	if (!unit.exact.includes(receiptPhaseHandoffPath) ||
		(candidateSpecificationOwners.has(boundary) !== unit.exact.includes(receiptPhaseSpecificationPath))) {
		fail(`${boundary}: candidate authority ownership`);
	}
	validateExactIndexModes(boundary, [receiptPhaseHandoffPath, receiptPhaseSpecificationPath], snapshot.indexEntries);
	for (const [path, oid] of [
		[receiptPhaseHandoffPath, snapshot.stagedHandoffOID],
		[receiptPhaseSpecificationPath, snapshot.stagedSpecificationOID],
	]) {
		const entry = snapshot.indexEntries.find((candidate) => candidate.path === path);
		if (entry.object !== oid) fail(`${boundary}: candidate index/tree blob mismatch: ${path}`);
	}
	if (!snapshot.stagedHandoffBytes.equals(snapshot.workingHandoffBytes)) fail(`${boundary}: HANDOFF staged/worktree mismatch`);
	if (!snapshot.stagedSpecificationBytes.equals(snapshot.workingSpecificationBytes)) fail(`${boundary}: specification staged/worktree mismatch`);
	if (!candidateSpecificationOwners.has(boundary) && !snapshot.stagedSpecificationBytes.equals(snapshot.parentSpecificationBytes)) {
		fail(`${boundary}: candidate changed sealed topology table`);
	}
	const ownerContract = candidateSpecificationOwnerContracts[boundary];
	if (ownerContract !== undefined && snapshot.stagedSpecificationBytes.equals(snapshot.parentSpecificationBytes)) {
		fail(`${boundary}: maintenance candidate did not replace the sealed ${ownerContract.priorSchema} topology table`);
	}
	let stagedSpecification;
	try {
		stagedSpecification = validateSpecification(JSON.parse(decodeCandidateUTF8(
			snapshot.stagedSpecificationBytes, "candidate specification",
		)));
	} catch (error) {
		fail(`${boundary}: candidate specification invalid (${error.message})`);
	}
	if (!isDeepStrictEqual(stagedSpecification, specification)) fail(`${boundary}: candidate specification differs from loaded authority`);
	const candidate = classifyCandidateReceiptPhase(
		specification, decodeCandidateUTF8(snapshot.stagedHandoffBytes, "candidate HANDOFF"),
	);
	if (candidate.boundary !== boundary) fail(`${boundary}: explicit boundary/capsule mismatch`);
	const parentBody = decodeCandidateUTF8(snapshot.parentHandoffBytes, "candidate parent HANDOFF");
	let parentBoundary;
	if (parentBody.includes(receiptPhaseCapsuleStart) || parentBody.includes(receiptPhaseCapsuleEnd)) {
		parentBoundary = classifyCandidateReceiptPhase(specification, parentBody).boundary;
	}
	if (parentBoundary === undefined) {
		if (boundary !== "C3D" || candidate.parent !== "C3B" ||
			snapshot.parentCommit !== sealedC3BCandidateParent.commit || snapshot.parentTree !== sealedC3BCandidateParent.tree) {
			fail(`${boundary}: candidate bootstrap mismatch`);
		}
	} else if (boundary === "C3D" || candidate.parent !== parentBoundary) {
		fail(`${boundary}: candidate requires parent ${candidate.parent}, found ${parentBoundary}`);
	}
	if (ownerContract !== undefined && (snapshot.parentCommit !== ownerContract.authority.commit ||
		snapshot.parentTree !== ownerContract.authority.tree || parentBoundary !== ownerContract.parent)) {
		fail(`${boundary}: candidate requires the exact sealed ${ownerContract.parent} parent authority`);
	}
	return candidate;
}

export async function runIndependentCandidatePhaseGate(specification, boundary) {
	if (!forwardCandidateReceiptPhaseBoundarySet.has(boundary)) fail(`candidate boundary outside forward horizon: ${boundary}`);
	const parentCommit = decodeCandidateGitOID(await gitOutput(["rev-parse", "--verify", "HEAD^{commit}"]), "candidate parent commit");
	const parentTree = decodeCandidateGitOID(await gitOutput(["rev-parse", "--verify", `${parentCommit}^{tree}`]), "candidate parent tree");
	const indexTree = decodeCandidateGitOID(await gitOutput(["write-tree"]), "candidate index tree");
	const indexEntries = await stagedIndexEntries();
	const stagedHandoffOID = decodeCandidateGitOID(
		await gitOutput(["rev-parse", "--verify", `${indexTree}:${receiptPhaseHandoffPath}`]), "candidate HANDOFF blob",
	);
	const stagedSpecificationOID = decodeCandidateGitOID(
		await gitOutput(["rev-parse", "--verify", `${indexTree}:${receiptPhaseSpecificationPath}`]), "candidate specification blob",
	);
	const snapshot = {
		candidateBoundary: boundary,
		parentCommit,
		parentTree,
		indexTree,
		indexEntries,
		stagedHandoffOID,
		stagedSpecificationOID,
		stagedHandoffBytes: await gitOutput(["show", `${indexTree}:${receiptPhaseHandoffPath}`]),
		workingHandoffBytes: await readCandidateWorktreeBytes(receiptPhaseHandoffPath),
		parentHandoffBytes: await gitOutput(["show", `${parentCommit}:${receiptPhaseHandoffPath}`]),
		stagedSpecificationBytes: await gitOutput(["show", `${indexTree}:${receiptPhaseSpecificationPath}`]),
		workingSpecificationBytes: await readCandidateWorktreeBytes(receiptPhaseSpecificationPath),
		parentSpecificationBytes: await gitOutput(["show", `${parentCommit}:${receiptPhaseSpecificationPath}`]),
	};
	validateIndependentCandidateSnapshot(specification, snapshot);
	await validateTransitionAuthorityArtifacts(specification, boundary, snapshot.parentCommit);
	await validateFutureC6Authority(specification, snapshot);
	const finalParentCommit = decodeCandidateGitOID(await gitOutput(["rev-parse", "--verify", "HEAD^{commit}"]), "candidate final parent commit");
	const finalAuthority = {
		parentCommit: finalParentCommit,
		parentTree: decodeCandidateGitOID(await gitOutput(["rev-parse", "--verify", `${finalParentCommit}^{tree}`]), "candidate final parent tree"),
		indexTree: decodeCandidateGitOID(await gitOutput(["write-tree"]), "candidate final index tree"),
	};
	requireCandidateTransitionAuthorityStable(snapshot, finalAuthority);
	if (!snapshot.workingHandoffBytes.equals(await readCandidateWorktreeBytes(receiptPhaseHandoffPath)) ||
		!snapshot.workingSpecificationBytes.equals(await readCandidateWorktreeBytes(receiptPhaseSpecificationPath))) {
		fail(`${boundary}: candidate worktree authority changed during transition validation`);
	}
	console.log(`P07B-C ${boundary} independent candidate transition passed: immutable scope authority pins parent/index trees, exact capsule edge, table bytes, index OIDs, and stable worktree authority; persistent-drift detection only, sole-writer required, same-user ABA remains outside the claim`);
}

export function receiptManifest(specification, unit) {
	if (!unitOrder.includes(unit)) fail(`unknown unit ${unit}`);
	const entry = specification.units[unit];
	if (entry.verification_profile !== "RECEIPT_RECONCILIATION") fail(`${unit}: not a receipt reconciliation profile`);
	return entry.receipt_claims;
}

export function maintenanceManifest(specification, unit) {
	if (!unitOrder.includes(unit)) fail(`unknown unit ${unit}`);
	const entry = specification.units[unit];
	if (entry.verification_profile !== "NON_PRODUCT_MAINTENANCE") {
		fail(`${unit}: not a non-product maintenance profile`);
	}
	return entry.maintenance_claims;
}

export function unexpectedPaths(specification, unit, paths) {
	if (!unitOrder.includes(unit)) fail(`unknown unit ${unit}`);
	if (!Array.isArray(paths) || !paths.every((path) => validPath(path))) fail("candidate path roster");
	const entry = specification.units[unit];
	return paths.filter((path) => !entry.exact.includes(path) &&
		(markdownAuthorityPath(path) || !entry.prefixes.some((prefix) => path.startsWith(prefix))));
}

export function exactPathsMatch(specification, unit, paths) {
	if (!unitOrder.includes(unit)) fail(`unknown unit ${unit}`);
	if (!Array.isArray(paths) || !paths.every((path) => validPath(path))) fail("candidate path roster");
	const entry = specification.units[unit];
	if (entry.prefixes.length !== 0) fail(`${unit}: exact staged mode requires an empty prefix roster`);
	return JSON.stringify([...paths].sort()) === JSON.stringify(entry.exact);
}

async function loadSpecification() {
	let parsed;
	try {
		parsed = JSON.parse(await readFile(specificationPath, "utf8"));
	} catch (error) {
		fail(`cannot read strict specification: ${error.message}`);
	}
	return validateSpecification(parsed);
}

function admittedDarwinGitStderr(bytes) {
	if (!Buffer.isBuffer(bytes) || bytes.length > gitStderrMaximumBytes) return false;
	if (bytes.length === 0) return true;
	if (bytes.length >= 3 && bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf) return false;
	let source;
	try {
		source = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
	} catch {
		return false;
	}
	if (!source.endsWith("\n") || source.includes("\r")) return false;
	return source.slice(0, -1).split("\n").every((line) =>
		/^git: warning: confstr\(\) failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using \/tmp instead$/u.test(line) ||
		/^20[0-9]{2}-[0-9]{2}-[0-9]{2} [0-9:.]+ xcodebuild\[[0-9]+:[0-9]+\]  DVTFilePathFSEvents: Failed to start fs event stream\.$/u.test(line) ||
		/^20[0-9]{2}-[0-9]{2}-[0-9]{2} [0-9:.]+ xcodebuild\[[0-9]+:[0-9]+\] \[MT\] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr\(3\), error = Error Domain=NSPOSIXErrorDomain Code=5 "Input\/output error"\. Using NSCachesDirectory instead\.$/u.test(line));
}

function checkedGitOutput(result) {
	const stdout = result.stdout ?? Buffer.alloc(0);
	const stderr = result.stderr ?? Buffer.alloc(0);
	if (!Buffer.isBuffer(stdout) || !Buffer.isBuffer(stderr)) fail("git inventory returned non-buffer streams");
	if (result.error || result.status !== 0 || result.signal || !admittedDarwinGitStderr(stderr)) {
		const stderrPrefix = stderr.subarray(0, gitStderrPrefixBytes);
		const errorDetail = boundedErrorDescriptor(result.error);
		fail(
			`git inventory failed (status=${JSON.stringify(result.status ?? null)}, ` +
			`signal=${JSON.stringify(result.signal ?? null)}, error=${JSON.stringify(errorDetail)}, ` +
			`stderr_bytes=${stderr.length}, stderr_prefix_hex=${stderrPrefix.toString("hex")}, ` +
			`stderr_truncated=${stderr.length > gitStderrPrefixBytes})`,
		);
	}
	return stdout;
}

export async function gitOutput(args) {
	const requested = process.env.COUNTERSHAPE_GIT || "/usr/bin/git";
	if (!isAbsolute(requested)) fail("COUNTERSHAPE_GIT must be absolute");
	let git;
	try {
		git = await realpath(requested);
	} catch (error) {
		fail(`git admission failed: ${JSON.stringify(boundedErrorDescriptor(error))}`);
	}
	const result = spawnSync(git, ["--no-replace-objects", ...args], {
		cwd: repositoryRoot,
		encoding: "buffer",
		timeout: 30_000,
		maxBuffer: 20 * 1024 * 1024,
		env: {
			HOME: process.env.HOME || "/", TMPDIR: process.env.TMPDIR || "/tmp",
			PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1",
			GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
			GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
		},
	});
	return checkedGitOutput(result);
}

export async function stagedPaths() {
	const bytes = await gitOutput(stagedInventoryArgs);
	if (bytes.length > 0 && bytes[bytes.length - 1] !== 0) fail("git inventory omitted its final NUL delimiter");
	let decoded;
	try {
		decoded = new TextDecoder("utf-8", { fatal: true }).decode(bytes.length === 0 ? bytes : bytes.subarray(0, -1));
	} catch (error) {
		fail(`git inventory path is not valid UTF-8: ${error.message}`);
	}
	const parts = decoded.length === 0 ? [] : decoded.split("\0");
	if (new Set(parts).size !== parts.length) fail("duplicate staged path");
	return parts.sort();
}

export async function stagedIndexEntries() {
	const bytes = await gitOutput(stagedIndexArgs);
	if (bytes.length > 0 && bytes[bytes.length - 1] !== 0) fail("git index inventory omitted its final NUL delimiter");
	let decoded;
	try {
		decoded = new TextDecoder("utf-8", { fatal: true }).decode(bytes.length === 0 ? bytes : bytes.subarray(0, -1));
	} catch (error) {
		fail(`git index inventory is not valid UTF-8: ${error.message}`);
	}
	const rows = decoded.length === 0 ? [] : decoded.split("\0");
	const entries = [];
	for (const row of rows) {
		const match = /^([0-7]{6}) ([0-9a-f]{40}|[0-9a-f]{64}) ([0-3])\t([\s\S]+)$/u.exec(row);
		if (!match || !validPath(match[4])) fail("malformed git index inventory row");
		entries.push(Object.freeze({ mode: match[1], object: match[2], stage: Number(match[3]), path: match[4] }));
	}
	return entries;
}

export function validateReceiptIndexModes(specification, unit, paths, entries) {
	receiptManifest(specification, unit);
	validateExactIndexModes(unit, paths, entries);
}

export function validateExactIndexModes(unit, paths, entries) {
	if (!Array.isArray(entries)) fail(`${unit}: index entry roster`);
	for (const path of paths) {
		const matches = entries.filter((entry) => entry?.path === path);
		if (matches.length !== 1 || matches[0].mode !== "100644" || matches[0].stage !== 0 ||
			!(/^(?:[0-9a-f]{40}|[0-9a-f]{64})$/u.test(matches[0].object)) || /^0+$/u.test(matches[0].object)) {
			fail(`${unit}: path must be one staged regular mode-100644 blob: ${path}`);
		}
	}
}

export function credentialPatternFindings(entries) {
	if (!Array.isArray(entries) || entries.some((entry) => !entry || typeof entry.path !== "string" || typeof entry.text !== "string")) {
		fail("credential scan entry roster");
	}
	const findings = [];
	for (const entry of entries) {
		for (const pattern of credentialPatterns) {
			if (pattern.expression.test(entry.text)) findings.push(Object.freeze({ pattern: pattern.name, path: entry.path }));
		}
	}
	return findings;
}

export async function stagedBlobEntries(paths) {
	const entries = [];
	for (const path of paths) {
		const bytes = await gitOutput(["show", `:${path}`]);
		let text;
		try {
			text = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
		} catch (error) {
			fail(`staged blob is not UTF-8 text: ${path} (${error.message})`);
		}
		if (bytes.length === 0) fail(`staged blob is empty: ${path}`);
		entries.push(Object.freeze({ path, text }));
	}
	return entries;
}

async function requireExactStagedSource(specification, unit) {
	const paths = await stagedPaths();
	if (paths.length === 0) fail("staged inventory is empty");
	if (!exactPathsMatch(specification, unit, paths)) fail(`${unit} exact staged roster mismatch`);
	const entries = await stagedIndexEntries();
	const repeatedPaths = await stagedPaths();
	if (JSON.stringify(repeatedPaths) !== JSON.stringify(paths)) fail("staged inventory changed during source inspection");
	validateExactIndexModes(unit, paths, entries);
	return paths;
}

function sourceGateAdmitted(specification, unit) {
	return unitOrder.includes(unit) && specification.units[unit]?.verification_profile === "SOURCE_FULL";
}

function exactSourceGateAdmitted(specification, unit) {
	return sourceGateAdmitted(specification, unit) && specification.units[unit].prefixes.length === 0;
}

function unrealizedRequiredPrefixes(specification, unit, paths) {
	if (!new Set(["C4", "C5"]).has(unit)) return [];
	return specification.units[unit].prefixes.filter((prefix) =>
		!paths.some((path) => path.startsWith(prefix) && path.length > prefix.length));
}

async function requireAdmittedStagedSource(specification, unit) {
	if (!sourceGateAdmitted(specification, unit)) fail(`${unit}: not a SOURCE_FULL unit`);
	// An empty-prefix entry remains a declared exact-roster SOURCE_FULL unit; a nonempty-prefix
	// entry is still SOURCE_FULL but must contain every exact path and no path outside its prefixes.
	const paths = await stagedPaths();
	if (paths.length === 0) fail("staged inventory is empty");
	const entry = specification.units[unit];
	if (entry.exact.some((path) => !paths.includes(path)) || unexpectedPaths(specification, unit, paths).length !== 0 ||
		(entry.prefixes.length === 0 && !exactPathsMatch(specification, unit, paths))) {
		fail(`${unit}: staged source roster mismatch`);
	}
	const unrealizedPrefixes = unrealizedRequiredPrefixes(specification, unit, paths);
	if (unrealizedPrefixes.length > 0) fail(`${unit}: staged source unrealized prefixes: ${unrealizedPrefixes.join(", ")}`);
	const entries = await stagedIndexEntries();
	const repeatedPaths = await stagedPaths();
	if (!isDeepStrictEqual(repeatedPaths, paths)) fail("staged inventory changed during source inspection");
	validateExactIndexModes(unit, paths, entries);
	return paths;
}

async function requireAdmittedStagedMaintenance(specification, unit) {
	const entry = specification.units[unit];
	if (entry?.verification_profile !== "NON_PRODUCT_MAINTENANCE") {
		fail(`${unit}: not a NON_PRODUCT_MAINTENANCE unit`);
	}
	const paths = await stagedPaths();
	if (paths.length === 0 || !exactPathsMatch(specification, unit, paths)) {
		fail(`${unit}: exact staged maintenance roster mismatch`);
	}
	validateExactIndexModes(unit, paths, await stagedIndexEntries());
	for (const authorityPath of nonProductMaintenanceForbiddenAuthorityPaths) {
		const drift = await gitOutput([
			"diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none",
			"--name-only", "-z", "HEAD", "--", authorityPath,
		]);
		if (drift.length !== 0) fail(`${unit}: parent-owned maintenance authority drift: ${authorityPath}`);
	}
	if (!isDeepStrictEqual(await stagedPaths(), paths)) {
		fail("staged inventory changed during maintenance inspection");
	}
	return paths;
}

async function runSourceFinalGate(specification, unit) {
	const paths = await requireAdmittedStagedSource(specification, unit);
	await gitOutput(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--cached", "--check", "--"]);
	const unstaged = await gitOutput(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--name-only", "-z", "--"]);
	if (unstaged.length !== 0) fail("unstaged tracked changes are present");
	const untracked = await gitOutput(["ls-files", "--others", "--exclude-standard", "-z", "--"]);
	if (untracked.length !== 0) fail("untracked paths are present");
	const finalPaths = await stagedPaths();
	if (JSON.stringify(finalPaths) !== JSON.stringify(paths)) fail("staged inventory changed during diff-integrity checks");
	const digest = createHash("sha256").update(`${paths.join("\n")}\n`, "utf8").digest("hex");
	console.log(`P07B-C ${unit} final source gate passed: ${paths.length} admitted mode-100644 paths, clean staged diff, no unstaged or untracked paths, sorted-newline sha256:${digest}`);
}

async function runReceiptFinalGate(specification, unit) {
	if (specification.units[unit]?.verification_profile !== "RECEIPT_RECONCILIATION") {
		fail("receipt-final-gate is admitted only for a receipt reconciliation unit");
	}
	const paths = await stagedPaths();
	if (!exactPathsMatch(specification, unit, paths)) fail(`${unit}: receipt-final exact staged roster mismatch`);
	validateReceiptIndexModes(specification, unit, paths, await stagedIndexEntries());
	await gitOutput(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--cached", "--check", "--"]);
	if ((await gitOutput(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--name-only", "-z", "--"])).length !== 0) {
		fail("unstaged tracked changes are present");
	}
	if ((await gitOutput(["ls-files", "--others", "--exclude-standard", "-z", "--"])).length !== 0) {
		fail("untracked paths are present");
	}
	if (!isDeepStrictEqual(await stagedPaths(), paths)) fail("staged inventory changed during receipt diff-integrity checks");
	console.log(`P07B-C ${unit} final receipt gate passed: ${paths.length} exact mode-100644 paths, clean staged diff, no unstaged or untracked paths`);
}

async function runMaintenanceFinalGate(specification, unit) {
	const paths = await requireAdmittedStagedMaintenance(specification, unit);
	await gitOutput(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--cached", "--check", "--"]);
	if ((await gitOutput([
		"diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--name-only", "-z", "--",
	])).length !== 0) {
		fail("unstaged tracked changes are present");
	}
	if ((await gitOutput(["ls-files", "--others", "--exclude-standard", "-z", "--"])).length !== 0) {
		fail("untracked paths are present");
	}
	if (!isDeepStrictEqual(await stagedPaths(), paths)) fail("staged inventory changed during maintenance diff-integrity checks");
	const digest = createHash("sha256").update(`${paths.join("\n")}\n`, "utf8").digest("hex");
	console.log(`P07B-C ${unit} dormant non-product maintenance scope primitive passed: ${paths.length} exact mode-100644 paths, no tracked change outside the parent-predeclared roster relative to current HEAD, exact sealed-parent binding and PARENT_FROZEN receipt remain unavailable, verifier outcomes and product behavior inherited-unreproven, clean staged diff, sorted-newline sha256:${digest}`);
}

async function runCredentialScan(specification, unit) {
	const profile = specification.units[unit]?.verification_profile;
	let paths;
	if (profile === "SOURCE_FULL") paths = await requireAdmittedStagedSource(specification, unit);
	else if (profile === "RECEIPT_RECONCILIATION") {
		paths = await stagedPaths();
		if (!exactPathsMatch(specification, unit, paths)) fail(`${unit}: credential receipt roster mismatch`);
		validateReceiptIndexModes(specification, unit, paths, await stagedIndexEntries());
	} else if (profile === "NON_PRODUCT_MAINTENANCE") {
		paths = await requireAdmittedStagedMaintenance(specification, unit);
	} else fail("credential-scan unit profile");
	const findings = credentialPatternFindings(await stagedBlobEntries(paths));
	if (findings.length > 0) fail(`structured credential-pattern findings: ${JSON.stringify(findings)}`);
	console.log(`P07B-C ${unit} scoped staged structured credential-pattern scan: 0 findings across ${paths.length} exact paths and ${credentialPatterns.length} named patterns`);
}

async function runC6ASourceAuthorityGate(specification, unit) {
	if (unit !== "C6M" && unit !== "C6B") fail("source-authority-gate is admitted only for C6M or C6B");
	await runIndependentCandidatePhaseGate(specification, unit);
	console.log(`P07B-C ${unit} C6A source-authority gate passed: sealed ancestry/note, canonical manifest transport, and phase-appropriate receipt projections agree`);
}

function runIndependentCandidatePhaseSelfTest(specification) {
	const rows = Object.fromEntries(receiptPhaseRows(specification).map((row) => [row.boundary, row]));
	const specificationBytes = Buffer.from(`${JSON.stringify(specification, null, 2)}\n`, "utf8");
	const document = (boundary) => Buffer.from(
		`# Candidate fixture\n\n## Current state\n\n${renderCandidateReceiptPhaseCapsule(rows[boundary])}\n\n## Tail\n\nStable.\n`,
		"utf8",
	);
	const fixture = (candidateBoundary, parentBoundary) => {
		const bootstrapCandidate = candidateBoundary === "C3D";
		const ownerContract = candidateSpecificationOwnerContracts[candidateBoundary];
		const handoff = document(candidateBoundary);
		return {
			candidateBoundary,
			parentCommit: bootstrapCandidate ? sealedC3BCandidateParent.commit : ownerContract?.authority.commit ?? "f".repeat(40),
			parentTree: bootstrapCandidate ? sealedC3BCandidateParent.tree : ownerContract?.authority.tree ?? "e".repeat(40),
			indexTree: "3".repeat(40),
			indexEntries: [
				{ mode: "100644", object: "1".repeat(40), stage: 0, path: receiptPhaseHandoffPath },
				{ mode: "100644", object: "2".repeat(40), stage: 0, path: receiptPhaseSpecificationPath },
			],
			stagedHandoffOID: "1".repeat(40),
			stagedSpecificationOID: "2".repeat(40),
			stagedHandoffBytes: handoff,
			workingHandoffBytes: handoff,
			parentHandoffBytes: Buffer.from(parentBoundary === undefined ? "# Parent fixture without capsule.\n" : document(parentBoundary)),
			stagedSpecificationBytes: specificationBytes,
			workingSpecificationBytes: specificationBytes,
			parentSpecificationBytes: bootstrapCandidate ? Buffer.from("{\"schema_version\":\"historical-v15\"}\n") :
				ownerContract === undefined ? specificationBytes : Buffer.from(`{\"schema_version\":\"${ownerContract.priorSchema}\"}\n`),
		};
	};
	const accepted = forwardCandidateReceiptPhaseBoundaries.map((candidate) => [
		candidate,
		candidate === "C3D" ? undefined : rows[candidate].parent,
	]);
	for (const [candidate, parent] of accepted) validateIndependentCandidateSnapshot(specification, fixture(candidate, parent));
	let rejected = 0;
	const refuse = (name, baseline, mutate) => {
		const hostile = { ...baseline, indexEntries: baseline.indexEntries.map((entry) => ({ ...entry })) };
		mutate(hostile);
		let refused = false;
		try { validateIndependentCandidateSnapshot(specification, hostile); } catch { refused = true; }
		if (!refused) fail(`independent candidate self-test false negative: ${name}`);
		rejected += 1;
	};
	const integrityBoundary = forwardCandidateReceiptPhaseBoundaries[
		forwardCandidateReceiptPhaseBoundaries.indexOf(Object.keys(candidateSpecificationOwnerContracts).at(-1)) + 1
	];
	const integrityParent = rows[integrityBoundary].parent;
	refuse("CLI capsule mismatch", fixture(integrityBoundary, integrityParent), (value) => {
		value.stagedHandoffBytes = value.workingHandoffBytes = document(integrityParent);
	});
	refuse("HANDOFF worktree drift", fixture(integrityBoundary, integrityParent), (value) => { value.workingHandoffBytes = Buffer.from("drift\n"); });
	refuse("specification worktree drift", fixture(integrityBoundary, integrityParent), (value) => { value.workingSpecificationBytes = Buffer.from("drift\n"); });
	refuse("sealed specification drift", fixture(integrityBoundary, integrityParent), (value) => { value.parentSpecificationBytes = Buffer.from("drift\n"); });
	for (const [candidate, contract] of Object.entries(candidateSpecificationOwnerContracts)) {
		refuse(`${candidate} wrong sealed ${contract.parent} parent`, fixture(candidate, contract.parent), (value) => { value.parentCommit = "0".repeat(40); });
		refuse(`${candidate} wrong sealed ${contract.parent} tree`, fixture(candidate, contract.parent), (value) => { value.parentTree = "0".repeat(40); });
		refuse(`${candidate} unchanged ${contract.priorSchema} topology`, fixture(candidate, contract.parent), (value) => { value.parentSpecificationBytes = value.stagedSpecificationBytes; });
	}
	for (const candidate of forwardCandidateReceiptPhaseBoundaries.filter((boundary) =>
		boundary !== "C3D" && candidateSpecificationOwnerContracts[boundary] === undefined)) {
		refuse(`${candidate} unauthorized specification byte change`, fixture(candidate, rows[candidate].parent), (value) => {
			value.stagedSpecificationBytes = value.workingSpecificationBytes = Buffer.from(`${JSON.stringify(specification)}\n`, "utf8");
		});
	}
	refuse("bootstrap commit", fixture("C3D", undefined), (value) => { value.parentCommit = "0".repeat(40); });
	refuse("bootstrap tree", fixture("C3D", undefined), (value) => { value.parentTree = "0".repeat(40); });
	const priorStates = [undefined, ...Object.keys(rows)];
	for (const candidate of forwardCandidateReceiptPhaseBoundaries) {
		const expectedParent = candidate === "C3D" ? undefined : rows[candidate].parent;
		for (const parent of priorStates) {
			if (parent === expectedParent) continue;
			refuse(`generated invalid parent ${candidate}<-${parent ?? "ABSENT"}`, fixture(candidate, parent), () => {});
		}
	}
	refuse("HANDOFF symlink", fixture(integrityBoundary, integrityParent), (value) => { value.indexEntries[0].mode = "120000"; });
	refuse("specification executable", fixture(integrityBoundary, integrityParent), (value) => { value.indexEntries[1].mode = "100755"; });
	refuse("HANDOFF index OID", fixture(integrityBoundary, integrityParent), (value) => { value.indexEntries[0].object = "9".repeat(40); });
	refuse("specification index OID", fixture(integrityBoundary, integrityParent), (value) => { value.indexEntries[1].object = "9".repeat(40); });
	refuse("HANDOFF non-UTF8", fixture(integrityBoundary, integrityParent), (value) => {
		value.stagedHandoffBytes = value.workingHandoffBytes = Buffer.from([0xff]);
	});
	refuse("parent non-UTF8", fixture(integrityBoundary, integrityParent), (value) => { value.parentHandoffBytes = Buffer.from([0xff]); });
	refuse("hidden capsule", fixture(integrityBoundary, integrityParent), (value) => {
		const hidden = Buffer.from(`# Candidate fixture\n\n## Current state\n\n<!--\n${renderCandidateReceiptPhaseCapsule(rows.C4)}\n-->\n`, "utf8");
		value.stagedHandoffBytes = value.workingHandoffBytes = hidden;
	});
	refuse("specification semantic drift", fixture("C3D", undefined), (value) => {
		const hostile = structuredClone(specification);
		hostile.units.C4.exact = hostile.units.C4.exact.slice(1);
		value.stagedSpecificationBytes = value.workingSpecificationBytes = Buffer.from(`${JSON.stringify(hostile, null, 2)}\n`);
	});
	const stable = fixture("C6M", "C6G");
	requireCandidateTransitionAuthorityStable(stable, {
		parentCommit: stable.parentCommit, parentTree: stable.parentTree, indexTree: stable.indexTree,
	});
	for (const [name, field] of [["parent drift", "parentCommit"], ["index drift", "indexTree"]]) {
		let refused = false;
		try {
			const after = { parentCommit: stable.parentCommit, parentTree: stable.parentTree, indexTree: stable.indexTree };
			after[field] = "8".repeat(40);
			requireCandidateTransitionAuthorityStable(stable, after);
		} catch { refused = true; }
		if (!refused) fail(`independent candidate self-test false negative: ${name}`);
		rejected += 1;
	}
	if (accepted.length !== 18 || rejected !== 597) fail(`independent candidate self-test cardinality ${accepted.length}/${rejected}`);
}

function runFutureC6MaterialSelfTest(authority) {
	const manifestBytes = Buffer.from(renderC6ASourceAuthorityManifest(authority), "utf8");
	const manifest = validateC6ASourceAuthorityManifest(manifestBytes, authority);
	const block = renderC6ASourceReceiptBlock(manifest);
	const absent = "# Future C6 fixture\n\nC6A source receipts remain absent.\n";
	const present = `${absent.trimEnd()}\n\n${block}\n`;
	validateFutureC6Material({ boundary: "C6A", handoffBody: absent, evidenceBody: absent });
	validateFutureC6Material({ boundary: "C6F", handoffBody: absent, evidenceBody: absent });
	validateFutureC6Material({ boundary: "C6G", handoffBody: absent, evidenceBody: absent });
	validateFutureC6Material({
		boundary: "C6M", authority, manifestBytes, handoffBody: absent, evidenceBody: absent,
		changedPaths: c6mDeclaredAdapterContract.exact,
	});
	validateFutureC6Material({
		boundary: "C6B", authority, manifestBytes, parentManifestBytes: Buffer.from(manifestBytes),
		handoffBody: present, evidenceBody: present, changedPaths: c6bDeclaredReceiptContract.exact,
	});
	let rejected = 0;
	const refuse = (name, fixture) => {
		let refused = false;
		try { validateFutureC6Material(fixture); } catch { refused = true; }
		if (!refused) fail(`future C6 material self-test false negative: ${name}`);
		rejected += 1;
	};
	refuse("C6A premature manifest", {
		boundary: "C6A", manifestBytes, handoffBody: absent, evidenceBody: absent,
	});
	refuse("C6A premature projection", {
		boundary: "C6A", handoffBody: present, evidenceBody: absent,
	});
	refuse("C6F premature manifest", {
		boundary: "C6F", manifestBytes, handoffBody: absent, evidenceBody: absent,
	});
	refuse("C6F premature projection", {
		boundary: "C6F", handoffBody: present, evidenceBody: absent,
	});
	refuse("C6G premature manifest", {
		boundary: "C6G", manifestBytes, handoffBody: absent, evidenceBody: absent,
	});
	refuse("C6G premature projection", {
		boundary: "C6G", handoffBody: present, evidenceBody: absent,
	});
	refuse("C6M missing manifest", {
		boundary: "C6M", authority, handoffBody: absent, evidenceBody: absent,
		changedPaths: c6mDeclaredAdapterContract.exact,
	});
	refuse("C6M parent already carried manifest", {
		boundary: "C6M", authority, manifestBytes, parentManifestBytes: manifestBytes,
		handoffBody: absent, evidenceBody: absent, changedPaths: c6mDeclaredAdapterContract.exact,
	});
	refuse("C6M wrong roster", {
		boundary: "C6M", authority, manifestBytes, handoffBody: absent, evidenceBody: absent,
		changedPaths: c6mDeclaredAdapterContract.exact.slice(1),
	});
	refuse("C6M premature HANDOFF projection", {
		boundary: "C6M", authority, manifestBytes, handoffBody: present, evidenceBody: absent,
		changedPaths: c6mDeclaredAdapterContract.exact,
	});
	refuse("C6M malformed manifest", {
		boundary: "C6M", authority, manifestBytes: Buffer.from("{}\n"), handoffBody: absent, evidenceBody: absent,
		changedPaths: c6mDeclaredAdapterContract.exact,
	});
	refuse("C6B missing inherited manifest", {
		boundary: "C6B", authority, manifestBytes, handoffBody: present, evidenceBody: present,
		changedPaths: c6bDeclaredReceiptContract.exact,
	});
	refuse("C6B inherited manifest drift", {
		boundary: "C6B", authority, manifestBytes, parentManifestBytes: Buffer.from(`${manifestBytes.toString("utf8")}\n`),
		handoffBody: present, evidenceBody: present, changedPaths: c6bDeclaredReceiptContract.exact,
	});
	refuse("C6B staged manifest", {
		boundary: "C6B", authority, manifestBytes, parentManifestBytes: manifestBytes,
		handoffBody: present, evidenceBody: present,
		changedPaths: [...c6bDeclaredReceiptContract.exact, c6aSourceAuthorityPath].sort(),
	});
	refuse("C6B missing HANDOFF projection", {
		boundary: "C6B", authority, manifestBytes, parentManifestBytes: manifestBytes,
		handoffBody: absent, evidenceBody: present, changedPaths: c6bDeclaredReceiptContract.exact,
	});
	refuse("C6B missing evidence projection", {
		boundary: "C6B", authority, manifestBytes, parentManifestBytes: manifestBytes,
		handoffBody: present, evidenceBody: absent, changedPaths: c6bDeclaredReceiptContract.exact,
	});
	refuse("C6B duplicate projection", {
		boundary: "C6B", authority, manifestBytes, parentManifestBytes: manifestBytes,
		handoffBody: `${present}\n${block}\n`, evidenceBody: present, changedPaths: c6bDeclaredReceiptContract.exact,
	});
	return rejected;
}

function runC6ASourceAuthoritySelfTest() {
	const authority = Object.freeze({
		commit: "a".repeat(40), tree: "b".repeat(40), parent: "c".repeat(40),
		noteBlob: "d".repeat(40), noteBodySHA256: "e".repeat(64), secretsOverride: true,
	});
	const baselineBytes = Buffer.from(renderC6ASourceAuthorityManifest(authority), "utf8");
	const baseline = validateC6ASourceAuthorityManifest(baselineBytes, authority);
	let rejected = 0;
	const refuseManifest = (name, bytes) => {
		let refused = false;
		try { validateC6ASourceAuthorityManifest(bytes, authority); } catch { refused = true; }
		if (!refused) fail(`C6A source-authority self-test false negative: ${name}`);
		rejected += 1;
	};
	for (const [name, mutate] of [
		["schema", (value) => { value.schema_version += "-altered"; }],
		["commit", (value) => { value.source_commit = "0".repeat(40); }],
		["tree", (value) => { value.source_tree = "0".repeat(40); }],
		["parent", (value) => { value.source_parent = "0".repeat(40); }],
		["subject", (value) => { value.source_subject += " altered"; }],
		["note ref", (value) => { value.note_ref = "refs/notes/other"; }],
		["note type", (value) => { value.note_type = "tree"; }],
		["note blob", (value) => { value.note_blob = "0".repeat(40); }],
		["note digest", (value) => { value.note_body_sha256 = "0".repeat(64); }],
		["seal disclosure", (value) => { value.secrets_override = false; }],
		["claim label", (value) => { value.claims[0].label += " altered"; }],
		["claim type", (value) => { value.claims[0].type = "command-succeeded"; }],
		["claim grade", (value) => { value.claims[0].grade = "STALE"; }],
		["claim order", (value) => { value.claims.reverse(); }],
		["claim missing", (value) => { value.claims.pop(); }],
		["unknown key", (value) => { value.unknown = true; }],
	]) {
		const hostile = structuredClone(baseline);
		mutate(hostile);
		refuseManifest(name, Buffer.from(`${JSON.stringify(hostile, null, 2)}\n`, "utf8"));
	}
	refuseManifest("compact JSON", Buffer.from(JSON.stringify(baseline), "utf8"));
	refuseManifest("BOM", Buffer.concat([Buffer.from([0xef, 0xbb, 0xbf]), baselineBytes]));
	refuseManifest("trailing bytes", Buffer.concat([baselineBytes, Buffer.from("\n")]));

	const block = renderC6ASourceReceiptBlock(baseline);
	const receiptBody = `# Receipt fixture\n\n${block}\n`;
	if (c6aReceiptProjectionState("# Absent\n", baseline, "fixture") !== "ABSENT" ||
		c6aReceiptProjectionState(receiptBody, baseline, "fixture") !== "PRESENT") {
		fail("C6A source-authority receipt projection controls");
	}
	const refuseProjection = (name, body) => {
		let refused = false;
		try { c6aReceiptProjectionState(body, baseline, "fixture"); } catch { refused = true; }
		if (!refused) fail(`C6A source-authority projection self-test false negative: ${name}`);
		rejected += 1;
	};
	refuseProjection("missing start", receiptBody.replace(`${c6aReceiptBlockStart}\n`, ""));
	refuseProjection("missing end", receiptBody.replace(`\n${c6aReceiptBlockEnd}`, ""));
	refuseProjection("duplicate", `${receiptBody}\n${block}\n`);
	refuseProjection("altered identity", receiptBody.replace(authority.commit, "0".repeat(40)));
	refuseProjection("fenced", `# Fixture\n\n\`\`\`markdown\n${block}\n\`\`\`\n`);
	refuseProjection("hidden", `# Fixture\n\n<!--\n${block}\n-->\n`);

	const identity = { commit: authority.commit, tree: authority.tree };
	const note = syntheticDidrunSourceNote(identity, c6aSourceClaimManifest, c6aSourceExpectedClaimArgv);
	validateDidrunSourceNote(
		note, identity, c6aSourceClaimManifest, c6aSourceExpectedClaimArgv, c6aSourceHermeticArgvPrefix, "C6A fixture",
	);
	const admittedRedactedNote = structuredClone(note);
	for (const recorded of admittedRedactedNote.claims) {
		const preview = recorded.claim.argv_preview;
		for (const position of [...sealedSourceDoubleRedactionPositions, 6, ...sealedSourceBareRedactionPositions]) {
			preview[position] = sealedSourceExpectedRedactionProjection(position, preview[position]);
		}
	}
	validateDidrunSourceNote(
		admittedRedactedNote, identity, c6aSourceClaimManifest, c6aSourceExpectedClaimArgv,
		c6aSourceHermeticArgvPrefix, "C6A redacted fixture",
	);
	const refuseNote = (name, mutate) => {
		const hostile = structuredClone(note);
		mutate(hostile);
		let refused = false;
		try {
			validateDidrunSourceNote(
				hostile, identity, c6aSourceClaimManifest, c6aSourceExpectedClaimArgv,
				c6aSourceHermeticArgvPrefix, "C6A fixture",
			);
		}
		catch { refused = true; }
		if (!refused) fail(`C6A didrun-note self-test false negative: ${name}`);
		rejected += 1;
	};
	for (const [name, mutate] of [
		["root key", (value) => { value.unknown = true; }],
		["commit", (value) => { value.commit = "0".repeat(40); }],
		["tree", (value) => { value.tree = "0".repeat(40); }],
		["seal type", (value) => { value.secrets_override = "true"; }],
		["claim count", (value) => { value.claims.pop(); }],
		["claim label", (value) => { value.claims[0].claim.label += " altered"; }],
		["claim type", (value) => { value.claims[0].claim.ctype = "command-succeeded"; }],
		["claim index", (value) => { value.claims[0].claim.declared_at_index = 1; }],
		["event index", (value) => { value.claims[0].claim.event_indices = [1]; }],
		["argv leading injection", (value) => { value.claims[0].claim.argv_preview.unshift("/bin/true"); }],
		["argv missing prefix", (value) => { value.claims[0].claim.argv_preview.splice(2, 1); }],
		["argv mutated prefix", (value) => { value.claims[0].claim.argv_preview[3] += "-altered"; }],
		["argv redaction outside authority", (value) => {
			value.claims[0].claim.argv_preview[3] = sealedSourceHermeticRedaction;
		}],
		["argv wrong redaction form", (value) => {
			value.claims[0].claim.argv_preview[2] = sealedSourceHermeticRedaction;
		}],
		["argv wrong redaction suffix", (value) => {
			value.claims[0].claim.argv_preview[6] = `${sealedSourceHermeticRedaction}.countershape/other/gocache`;
		}],
		["argv tail", (value) => {
			const argv = value.claims[0].claim.argv_preview;
			argv[argv.length - 1] += "-altered";
		}],
		["grade", (value) => { value.claims[0].grade = "stale"; }],
		["exit", (value) => { value.claims[0].exit_code = 1; }],
		["delta", (value) => { value.claims[0].delta = ["path"]; }],
		["coverage", (value) => { value.coverage.total_events -= 1; }],
	]) refuseNote(name, mutate);
	const c6fContractDigest = computedC6FSealedSourceContractDigest();
	if (c6fContractDigest !== expectedC6FSealedSourceContractDigest) {
		fail(`C6F sealed-source contract digest ${c6fContractDigest}`);
	}
	const c6fIdentity = Object.freeze({ commit: "1".repeat(40), tree: "2".repeat(40) });
	const c6fNote = syntheticDidrunSourceNote(
		c6fIdentity, c6fSourceClaimManifest, c6fSourceExpectedClaimArgv,
	);
	validateDidrunSourceNote(
		c6fNote, c6fIdentity, c6fSourceClaimManifest, c6fSourceExpectedClaimArgv,
		c6fSourceHermeticArgvPrefix, "C6F sealed-source fixture",
	);
	const c6fHostile = structuredClone(c6fNote);
	c6fHostile.claims.at(-1).claim.argv_preview[c6fHostile.claims.at(-1).claim.argv_preview.length - 1] += "-altered";
	let c6fRejected = false;
	try {
		validateDidrunSourceNote(
			c6fHostile, c6fIdentity, c6fSourceClaimManifest, c6fSourceExpectedClaimArgv,
			c6fSourceHermeticArgvPrefix, "C6F sealed-source fixture",
		);
	} catch { c6fRejected = true; }
	if (!c6fRejected) fail("C6F sealed-source didrun-note self-test false negative: command tail");
	rejected += 1;
	const c6gContractDigest = computedC6GSealedSourceContractDigest();
	if (c6gContractDigest !== expectedC6GSealedSourceContractDigest) {
		fail(`C6G sealed-source contract digest ${c6gContractDigest}`);
	}
	const c6gIdentity = Object.freeze({ commit: "3".repeat(40), tree: "4".repeat(40) });
	const c6gNote = syntheticDidrunSourceNote(
		c6gIdentity, c6gSourceClaimManifest, c6gSourceExpectedClaimArgv,
	);
	validateDidrunSourceNote(
		c6gNote, c6gIdentity, c6gSourceClaimManifest, c6gSourceExpectedClaimArgv,
		c6gSourceHermeticArgvPrefix, "C6G sealed-source fixture",
	);
	const c6gHostile = structuredClone(c6gNote);
	c6gHostile.claims.at(-1).claim.argv_preview[c6gHostile.claims.at(-1).claim.argv_preview.length - 1] += "-altered";
	let c6gRejected = false;
	try {
		validateDidrunSourceNote(
			c6gHostile, c6gIdentity, c6gSourceClaimManifest, c6gSourceExpectedClaimArgv,
			c6gSourceHermeticArgvPrefix, "C6G sealed-source fixture",
		);
	} catch { c6gRejected = true; }
	if (!c6gRejected) fail("C6G sealed-source didrun-note self-test false negative: command tail");
	rejected += 1;
	const futureMaterialRejected = runFutureC6MaterialSelfTest(authority);
	if (futureMaterialRejected !== 17) fail(`future C6 material self-test cardinality ${futureMaterialRejected}`);
	rejected += futureMaterialRejected;
	return rejected;
}

function requireSpecificationMutationRejected(specification, label, mutate) {
	const hostile = JSON.parse(JSON.stringify(specification));
	mutate(hostile);
	let rejected = false;
	try { validateSpecification(hostile); } catch { rejected = true; }
	if (!rejected) fail(`${label} self-test false negative`);
}

function runGitOutputDiagnosticSelfTest() {
	const cleanStdout = Buffer.from([0x00, 0xff, 0x41]);
	const clean = checkedGitOutput({
		error: undefined, status: 0, signal: null, stdout: cleanStdout, stderr: Buffer.alloc(0),
	});
	if (clean !== cleanStdout) fail("Git diagnostic self-test did not preserve exact stdout Buffer identity");
	const darwinStderrLines = Object.freeze([
		"git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead",
		"2026-08-02 08:38:10.701 xcodebuild[98149:9820105]  DVTFilePathFSEvents: Failed to start fs event stream.",
		"2026-08-02 08:38:10.926 xcodebuild[98149:9820104] [MT] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr(3), error = Error Domain=NSPOSIXErrorDomain Code=5 \"Input/output error\". Using NSCachesDirectory instead.",
	]);
	for (const [label, lines] of [
		["Darwin temp-dir warning", [darwinStderrLines[0]]],
		["Darwin FSEvents diagnostic", [darwinStderrLines[1]]],
		["Darwin cache-dir diagnostic", [darwinStderrLines[2]]],
		["combined admitted Darwin diagnostics", darwinStderrLines],
	]) {
		const accepted = checkedGitOutput({
			error: undefined, status: 0, signal: null, stdout: cleanStdout,
			stderr: Buffer.from(`${lines.join("\n")}\n`, "utf8"),
		});
		if (accepted !== cleanStdout) fail(`Git diagnostic self-test did not preserve stdout for ${label}`);
	}

	const expectRejected = (label, result, expected) => {
		let message;
		try {
			checkedGitOutput(result);
		} catch (error) {
			message = String(error.message ?? error);
		}
		if (message === undefined) fail(`Git diagnostic self-test false negative: ${label}`);
		for (const fragment of expected) {
			if (!message.includes(fragment)) fail(`Git diagnostic self-test ${label} omitted ${JSON.stringify(fragment)}: ${message}`);
		}
		if (message.includes("stdout-secret")) fail(`Git diagnostic self-test ${label} reflected stdout`);
		if (/[\u0000-\u001f\u007f\ufffd]/u.test(message)) {
			fail(`Git diagnostic self-test ${label} emitted a raw control or replacement character`);
		}
		return message;
	};
	const base = {
		error: undefined, status: 0, signal: null,
		stdout: Buffer.from("stdout-secret", "utf8"), stderr: Buffer.alloc(0),
	};
	expectRejected("spawn error", { ...base, error: new Error("injected spawn failure") }, [
		"status=0", "signal=null",
		`"name_hex":"${Buffer.from("Error").toString("hex")}"`,
		`"message_prefix_hex":"${Buffer.from("injected spawn failure").toString("hex")}"`,
		'"message_bytes":22', '"message_truncated":false',
		"stderr_bytes=0", "stderr_prefix_hex=", "stderr_truncated=false",
	]);
	const oversizedErrorMessage = `${"\u0000\u0001"}${"x".repeat(gitStderrPrefixBytes + 1)}`;
	expectRejected("oversized control-bearing spawn error", {
		...base, error: new Error(oversizedErrorMessage),
	}, [
		`"message_bytes":${gitStderrPrefixBytes + 3}`,
		`"message_prefix_hex":"${Buffer.from(oversizedErrorMessage).subarray(0, gitStderrPrefixBytes).toString("hex")}"`,
		'"message_truncated":true',
		"stderr_bytes=0", "stderr_prefix_hex=", "stderr_truncated=false",
	]);
	expectRejected("signal", { ...base, status: null, signal: "SIGTERM" }, [
		"status=null", 'signal=\"SIGTERM\"', "error=null",
		"stderr_bytes=0", "stderr_prefix_hex=", "stderr_truncated=false",
	]);
	expectRejected("nonzero", { ...base, status: 7 }, [
		"status=7", "signal=null", "error=null",
		"stderr_bytes=0", "stderr_prefix_hex=", "stderr_truncated=false",
	]);
	expectRejected("stderr only", { ...base, stdout: Buffer.alloc(0), stderr: Buffer.from("diagnostic", "utf8") }, [
		"status=0", "stderr_bytes=10", "stderr_prefix_hex=646961676e6f73746963", "stderr_truncated=false",
	]);
	expectRejected("mixed streams", { ...base, stderr: Buffer.from("mixed", "utf8") }, [
		"stderr_bytes=5", "stderr_prefix_hex=6d69786564", "stderr_truncated=false",
	]);
	const exactBound = Buffer.alloc(gitStderrPrefixBytes, 0xab);
	expectRejected("exact bound", { ...base, stderr: exactBound }, [
		`stderr_bytes=${gitStderrPrefixBytes}`,
		`stderr_prefix_hex=${exactBound.toString("hex")}`,
		"stderr_truncated=false",
	]);
	const overBound = Buffer.concat([exactBound, Buffer.from([0xcd])]);
	const overBoundMessage = expectRejected("over bound", { ...base, stderr: overBound }, [
		`stderr_bytes=${gitStderrPrefixBytes + 1}`,
		`stderr_prefix_hex=${exactBound.toString("hex")}`,
		"stderr_truncated=true",
	]);
	const overBoundFields = /stderr_bytes=(\d+), stderr_prefix_hex=([0-9a-f]*), stderr_truncated=(true|false)\)$/u.exec(overBoundMessage);
	if (overBoundFields === null ||
		Number(overBoundFields[1]) !== gitStderrPrefixBytes + 1 ||
		overBoundFields[2] !== exactBound.toString("hex") ||
		overBoundFields[2].length !== gitStderrPrefixBytes * 2 ||
		overBoundFields[3] !== "true") {
		fail(`Git diagnostic self-test over-bound exact fields drifted: ${overBoundMessage}`);
	}
	expectRejected("invalid UTF-8", { ...base, stderr: Buffer.from([0xff, 0xfe, 0xfd]) }, [
		"stderr_bytes=3", "stderr_prefix_hex=fffefd", "stderr_truncated=false",
	]);
	expectRejected("NUL and controls", { ...base, stderr: Buffer.from([0x00, 0x01, 0x0a, 0x1f, 0x7f]) }, [
		"stderr_bytes=5", "stderr_prefix_hex=00010a1f7f", "stderr_truncated=false",
	]);
	for (const [label, stderr] of [
		["admitted diagnostic without final LF", Buffer.from(darwinStderrLines[0], "utf8")],
		["admitted diagnostic with CRLF", Buffer.from(`${darwinStderrLines[1]}\r\n`, "utf8")],
		["admitted diagnostic plus foreign line", Buffer.from(`${darwinStderrLines[2]}\nforeign diagnostic\n`, "utf8")],
		["malformed Darwin timestamp", Buffer.from(`${darwinStderrLines[1].replace("2026-", "26-")}\n`, "utf8")],
		["UTF-8 BOM-prefixed admitted diagnostic", Buffer.concat([
			Buffer.from([0xef, 0xbb, 0xbf]), Buffer.from(`${darwinStderrLines[0]}\n`, "utf8"),
		])],
	]) {
		expectRejected(label, { ...base, stderr }, [
			"status=0", `stderr_bytes=${stderr.length}`, "stderr_prefix_hex=", `stderr_truncated=${stderr.length > gitStderrPrefixBytes}`,
		]);
	}
	const repeatedAdmitted = Buffer.from(
		`${darwinStderrLines[0]}\n`.repeat(Math.ceil((gitStderrMaximumBytes + 1) / (darwinStderrLines[0].length + 1))),
		"utf8",
	);
	if (repeatedAdmitted.length <= gitStderrMaximumBytes) fail("Git diagnostic self-test oversized fixture did not cross its bound");
	expectRejected("oversized admitted diagnostics", { ...base, stderr: repeatedAdmitted }, [
		"status=0", `stderr_bytes=${repeatedAdmitted.length}`, "stderr_prefix_hex=", "stderr_truncated=true",
	]);
	return 21;
}

async function runAuthorityProfileSelfTest(specification) {
	const c4lAuthority = structuredClone(specification.units.C4L.transition_authority);
	validateTransitionAuthority("C4L", c4lAuthority);
	validateTransitionAuthority("C4J", structuredClone(specification.units.C4J.transition_authority));
	validateTransitionAuthority("C5V", structuredClone(specification.units.C5V.transition_authority));
	const cited = structuredClone(c4lAuthority);
	cited.provenance = {
		kind: "CITED_UNAUTHENTICATED",
		artifact_role: "OWNER_INSTRUCTION_ARTIFACT",
		preexistence: "DIRECT_PARENT_TREE",
		artifact_path: "docs/status/owner-instruction.md",
		artifact_sha256: `sha256:${"a".repeat(64)}`,
	};
	validateTransitionAuthority("C4L", cited);

	const maintenance = {
		exact: [
			"docs/HANDOFF_MODE_C.md",
			"docs/status/P07B-C-FUTURE-MAINTENANCE.md",
			"tools/check-future-maintenance.mjs",
		],
		maintenance_claims: nonProductMaintenanceClaimRoles.map((role, index) => ({
			label: nonProductMaintenanceClaimLabel("FUTURE", role),
			role,
			type: index < 6 ? "tests-pass" : "command-succeeded",
		})),
		prefixes: [],
		product_authority: "NONE",
		product_behavior: "INHERITED_UNREPROVEN",
		product_projection: "PARENT_FROZEN",
		verification_profile: "NON_PRODUCT_MAINTENANCE",
	};
	validateNonProductMaintenanceEntry("FUTURE", maintenance);
	validateNonProductMaintenanceParentProfile(
		{ boundary: "FUTURE", profile: "NON_PRODUCT_MAINTENANCE" },
		{ boundary: "PARENT", profile: "SOURCE_FULL" },
	);

	let rejected = 0;
	const refuseAuthority = (name, unit, baseline, mutate) => {
		const hostile = structuredClone(baseline);
		mutate(hostile);
		let failed = false;
		try { validateTransitionAuthority(unit, hostile); } catch { failed = true; }
		if (!failed) fail(`transition-authority self-test false negative: ${name}`);
		rejected += 1;
	};
	for (const [name, mutate] of [
		["missing source", (value) => { delete value.source; }],
		["extra field", (value) => { value.extra = true; }],
		["source drift", (value) => { value.source = "PROSE"; }],
		["classification drift", (value) => { value.classification = "DEFECT_REPAIR"; }],
		["authentication overclaim", (value) => { value.authentication = "AUTHENTICATED"; }],
		["signature overclaim", (value) => { value.signed_authorization = "VERIFIED"; }],
		["missing provenance", (value) => { delete value.provenance; }],
		["unknown provenance", (value) => { value.provenance = { kind: "TRUSTED" }; }],
		["unevidenced extra field", (value) => { value.provenance.extra = true; }],
		["unevidenced disclosure drift", (value) => { value.provenance.disclosure += "_ALTERED"; }],
		["false predecessor prose", (value) => {
			value.predecessor_declaration = {
				kind: "PROSE_ONLY",
				artifact_role: "PREDECESSOR_RECORD_OF_OWNER_ATTRIBUTED_DIRECTION",
				preexistence: "DIRECT_PARENT_TREE",
				artifact_path: "docs/status/foreign.md",
				artifact_sha256: `sha256:${"a".repeat(64)}`,
			};
		}],
	]) {
		refuseAuthority(name, "C4L", c4lAuthority, mutate);
	}
	for (const [name, mutate] of [
		["cited mixed disclosure", (value) => { value.provenance.disclosure = ownerAuthorityDisclosure; }],
		["cited role drift", (value) => { value.provenance.artifact_role = "PREDECESSOR_RECORD_OF_OWNER_ATTRIBUTED_DIRECTION"; }],
		["cited preexistence drift", (value) => { value.provenance.preexistence = "CURRENT_TREE"; }],
		["cited absolute path", (value) => { value.provenance.artifact_path = "/tmp/instruction"; }],
		["cited dot path", (value) => { value.provenance.artifact_path = "./instruction"; }],
		["cited parent escape", (value) => { value.provenance.artifact_path = "../instruction"; }],
		["cited backslash path", (value) => { value.provenance.artifact_path = "docs\\instruction"; }],
		["cited duplicate slash", (value) => { value.provenance.artifact_path = "docs//instruction"; }],
		["cited control path", (value) => { value.provenance.artifact_path = "docs/instruction\n"; }],
		["cited digest prefix", (value) => { value.provenance.artifact_sha256 = "SHA256:" + "a".repeat(64); }],
		["cited digest case", (value) => { value.provenance.artifact_sha256 = `sha256:${"A".repeat(64)}`; }],
		["cited digest length", (value) => { value.provenance.artifact_sha256 = `sha256:${"a".repeat(63)}`; }],
		["cited zero digest", (value) => { value.provenance.artifact_sha256 = `sha256:${"0".repeat(64)}`; }],
	]) {
		refuseAuthority(name, "C4L", cited, mutate);
	}
	for (const [name, mutate] of [
		["C4J predecessor kind", (value) => { value.predecessor_declaration.kind = "NONE"; }],
		["C4J predecessor path", (value) => { value.predecessor_declaration.artifact_path = "docs/status/foreign.md"; }],
		["C4J predecessor role", (value) => { value.predecessor_declaration.artifact_role = "OWNER_INSTRUCTION_ARTIFACT"; }],
		["C4J predecessor digest", (value) => { value.predecessor_declaration.artifact_sha256 = `sha256:${"a".repeat(64)}`; }],
	]) {
		refuseAuthority(name, "C4J", specification.units.C4J.transition_authority, mutate);
	}

	for (const parentProfile of ["RECEIPT_RECONCILIATION", "NON_PRODUCT_MAINTENANCE"]) {
		let failed = false;
		try {
			validateNonProductMaintenanceParentProfile(
				{ boundary: "FUTURE", profile: "NON_PRODUCT_MAINTENANCE" },
				{ boundary: "PARENT", profile: parentProfile },
			);
		} catch {
			failed = true;
		}
		if (!failed) fail(`non-product maintenance parent-profile false negative: ${parentProfile}`);
		rejected += 1;
	}

	const refuseMaintenance = (name, mutate) => {
		const hostile = structuredClone(maintenance);
		mutate(hostile);
		let failed = false;
		try { validateNonProductMaintenanceEntry("FUTURE", hostile); } catch { failed = true; }
		if (!failed) fail(`non-product maintenance self-test false negative: ${name}`);
		rejected += 1;
	};
	for (const [name, mutate] of [
		["product authority", (value) => { value.product_authority = "SOURCE"; }],
		["product behavior", (value) => { value.product_behavior = "INHERITED_VERIFIED"; }],
		["product projection", (value) => { value.product_projection = "RECOMPUTED"; }],
		["prefix", (value) => { value.prefixes = ["tools/"]; }],
		["claim removal", (value) => { value.maintenance_claims.pop(); }],
		["claim role order", (value) => {
			[value.maintenance_claims[0].role, value.maintenance_claims[1].role] =
				[value.maintenance_claims[1].role, value.maintenance_claims[0].role];
		}],
		["claim type", (value) => { value.maintenance_claims[0].type = "command-succeeded"; }],
		["claim duplicate", (value) => { value.maintenance_claims[1].label = value.maintenance_claims[0].label; }],
		["claim cumulative", (value) => { value.maintenance_claims[0].label = "P07B-C FUTURE cumulative verification"; }],
		["claim security", (value) => { value.maintenance_claims[0].label = "P07B-C FUTURE security proof"; }],
		["claim grade borrowing", (value) => { value.maintenance_claims[0].label = "P07B-C FUTURE inherited grade"; }],
		["claim extra field", (value) => { value.maintenance_claims[0].extra = true; }],
	]) {
		refuseMaintenance(name, mutate);
	}
	for (const path of [
		...nonProductMaintenanceForbiddenAuthorityPaths,
		"internal/world/process.go",
		"testkit/processfixture/main.go",
		"cmd/countershape/main.go",
		"spec/schema/v1/contract.schema.json",
		"spec/verification/p07b-b-future-surface-authority.json",
		"tools/check-p07b-c-architecture.mjs",
		"tools/check-p07b-c-architecture-selftest.mjs",
		"tools/verify-go-test-repetition.mjs",
		"go.mod",
		"package.json",
	]) {
		refuseMaintenance(`forbidden path ${path}`, (value) => {
			value.exact = [...value.exact, path].sort();
		});
	}
	const syntheticCited = structuredClone(cited.provenance);
	syntheticCited.artifact_path = "docs/status/P07B-C-C4K-C3-GO-TIMEOUT-MAINTENANCE.md";
	syntheticCited.artifact_sha256 =
		"sha256:fc80148ddedddb925962a15dacb20dad8979426aa3f3ef4f8fef12df9c614d6b";
	await validateDirectParentArtifact(
		sealedC4KCandidateParent.commit,
		syntheticCited,
		"synthetic cited-authority transport control",
	);
	const pathspecHostile = structuredClone(syntheticCited);
	pathspecHostile.artifact_path = "docs/status/P07B-C-C4K-*-MAINTENANCE.md";
	let pathspecFailed = false;
	try {
		await validateDirectParentArtifact(
			sealedC4KCandidateParent.commit,
			pathspecHostile,
			"synthetic cited-authority literal-path hostile",
		);
	} catch {
		pathspecFailed = true;
	}
	if (!pathspecFailed) fail("cited-authority literal-path lookup interpreted pathspec syntax");
	rejected += 1;
	await validateTransitionAuthorityArtifacts(specification, "C4L", sealedC4JCandidateParent.commit);
	await validateTransitionAuthorityArtifacts(specification, "C5V", sealedC5CandidateParent.commit);
	console.log(`P07B-C transition-authority and dormant non-product-maintenance profile self-test passed: ${rejected} malformed, self-authorizing, overclaiming, parent-profile, literal-path, and forbidden-scope cases refused; synthetic cited transport plus exact C4J predecessor record reopened from their direct-parent trees`);
	return rejected;
}

async function runSelfTest() {
	const specification = await loadSpecification();
	const gitDiagnosticCases = runGitOutputDiagnosticSelfTest();
	runIndependentCandidatePhaseSelfTest(specification);
	const c6aSourceAuthorityRejections = runC6ASourceAuthoritySelfTest();
	const authorityProfileRejections = await runAuthorityProfileSelfTest(specification);
	const cases = [
		unexpectedPaths(specification, "C0A", ["docs/SEMANTICS.md"]).length === 0,
		unexpectedPaths(specification, "C0A", ["README.md"])[0] === "README.md",
		unexpectedPaths(specification, "C4", ["internal/processmechanics/owner_darwin.go"]).length === 0,
		unexpectedPaths(specification, "C4", ["internal/processmechanics_evil/owner_darwin.go"])[0] === "internal/processmechanics_evil/owner_darwin.go",
		unexpectedPaths(specification, "C4", ["internal/contractexec/runner/runner.go"]).length === 0,
		unexpectedPaths(specification, "C4", ["internal/contractexec/runner_evil/runner.go"])[0] === "internal/contractexec/runner_evil/runner.go",
		unexpectedPaths(specification, "C4", ["testkit/contractexec/cli/cli_test.go"]).length === 0,
		unexpectedPaths(specification, "C4", ["testkit/contractexec/cli_evil/cli_test.go"])[0] === "testkit/contractexec/cli_evil/cli_test.go",
		unexpectedPaths(specification, "C5", ["internal/contractexec/http/server.go"]).length === 0,
		unexpectedPaths(specification, "C5", ["internal/contractexec/http_evil/server.go"])[0] === "internal/contractexec/http_evil/server.go",
		unexpectedPaths(specification, "C5", ["internal/contractexec/scope/scope.go"]).length === 0,
		unexpectedPaths(specification, "C5", ["internal/contractexec/scope_evil/scope.go"])[0] === "internal/contractexec/scope_evil/scope.go",
		unexpectedPaths(specification, "C5", ["testkit/contractexec/http/http_test.go"]).length === 0,
		unexpectedPaths(specification, "C5", ["testkit/contractexec/http_evil/http_test.go"])[0] === "testkit/contractexec/http_evil/http_test.go",
		unrealizedRequiredPrefixes(specification, "C4", [
			...specification.units.C4.exact,
			"internal/contractexec/runner/runner.go",
			"internal/processmechanics/process.go",
			"testkit/contractexec/cli/cli_test.go",
		]).length === 0,
		isDeepStrictEqual(unrealizedRequiredPrefixes(specification, "C5", specification.units.C5.exact), specification.units.C5.prefixes),
		unexpectedPaths(specification, "C2", ["internal/store/nonhead_backdoor.go"])[0] === "internal/store/nonhead_backdoor.go",
		unexpectedPaths(specification, "C2M", ["tools/check-p07b-c-plan.mjs"]).length === 0,
		unexpectedPaths(specification, "C2M", ["internal/store/nonhead_contract.go"])[0] === "internal/store/nonhead_contract.go",
		unexpectedPaths(specification, "C3", ["internal/contractexec/target_evil.go"])[0] === "internal/contractexec/target_evil.go",
		unexpectedPaths(specification, "C1B", ["spec/verification/p07b-c-c1-receipt.json"]).length === 0,
		unexpectedPaths(specification, "C1B", ["internal/store/nonhead_contract.go"])[0] === "internal/store/nonhead_contract.go",
		unexpectedPaths(specification, "C2B", ["spec/verification/p07b-c-c2-receipt.json"]).length === 0,
		unexpectedPaths(specification, "C2B", ["internal/store/nonhead_contract.go"])[0] === "internal/store/nonhead_contract.go",
		unexpectedPaths(specification, "C3P", ["internal/contractexec/model/target.go"]).length === 0,
		unexpectedPaths(specification, "C3P", ["internal/hostepoch/epoch.go"])[0] === "internal/hostepoch/epoch.go",
		unexpectedPaths(specification, "C3V", ["docs/status/P07B-C-C3V-VERIFICATION-THROUGHPUT.md"]).length === 0,
		unexpectedPaths(specification, "C3V", ["tools/verify-current.mjs"])[0] === "tools/verify-current.mjs",
		unexpectedPaths(specification, "C3M", ["docs/status/P07B-C-C3P-RECEIPT-PHASE-MAINTENANCE.md"]).length === 0,
		unexpectedPaths(specification, "C3M", ["spec/verification/p07b-c-c3p-receipt.json"])[0] === "spec/verification/p07b-c-c3p-receipt.json",
		unexpectedPaths(specification, "C3PB", ["spec/verification/p07b-c-c3p-receipt.json"]).length === 0,
		unexpectedPaths(specification, "C3A", ["tools/check-p07b-b-architecture.mjs"]).length === 0,
		unexpectedPaths(specification, "C3A", ["internal/contractexec/target.go"])[0] === "internal/contractexec/target.go",
		unexpectedPaths(specification, "C3L", ["tools/check-u6-architecture.mjs"]).length === 0,
		unexpectedPaths(specification, "C3L", ["internal/store/nonhead_contract.go"])[0] === "internal/store/nonhead_contract.go",
		unexpectedPaths(specification, "C3F", ["tools/check-p07b-b-architecture.mjs"]).length === 0,
		unexpectedPaths(specification, "C3F", ["internal/contractexec/target.go"])[0] === "internal/contractexec/target.go",
		unexpectedPaths(specification, "C3S", ["tools/check-u6-architecture-selftest.mjs"]).length === 0,
		unexpectedPaths(specification, "C3S", ["tools/check-u6-architecture.mjs"])[0] === "tools/check-u6-architecture.mjs",
		unexpectedPaths(specification, "C3S", ["internal/store/nonhead_contract.go"])[0] === "internal/store/nonhead_contract.go",
		unexpectedPaths(specification, "C3", ["internal/contractexec/target.go"]).length === 0,
		unexpectedPaths(specification, "C3", ["spec/verification/p07b-c-c3-receipt.json"])[0] === "spec/verification/p07b-c-c3-receipt.json",
		unexpectedPaths(specification, "C3R", ["tools/check-p07b-c-plan.mjs"]).length === 0,
		unexpectedPaths(specification, "C3R", ["spec/verification/p07b-c-c3-receipt.json"])[0] === "spec/verification/p07b-c-c3-receipt.json",
		unexpectedPaths(specification, "C3R", ["internal/contractexec/target.go"])[0] === "internal/contractexec/target.go",
		unexpectedPaths(specification, "C3Q", ["docs/VERIFICATION.md"]).length === 0,
		unexpectedPaths(specification, "C3Q", ["spec/verification/p07b-c-c3-receipt.json"])[0] === "spec/verification/p07b-c-c3-receipt.json",
		unexpectedPaths(specification, "C3Q", ["internal/contractexec/target.go"])[0] === "internal/contractexec/target.go",
		unexpectedPaths(specification, "C3T", ["docs/status/P07B-C-C3T-PHASE-SELFTEST-MAINTENANCE.md"]).length === 0,
		unexpectedPaths(specification, "C3T", ["spec/verification/p07b-c-c3-receipt.json"])[0] === "spec/verification/p07b-c-c3-receipt.json",
		unexpectedPaths(specification, "C3T", ["internal/contractexec/target.go"])[0] === "internal/contractexec/target.go",
		unexpectedPaths(specification, "C3U", ["docs/status/P07B-C-C3U-CROSS-PHASE-RECEIPT-FIXTURE-MAINTENANCE.md"]).length === 0,
		unexpectedPaths(specification, "C3U", ["spec/verification/p07b-c-c3-receipt.json"])[0] === "spec/verification/p07b-c-c3-receipt.json",
		unexpectedPaths(specification, "C3U", ["internal/contractexec/target.go"])[0] === "internal/contractexec/target.go",
		unexpectedPaths(specification, "C3B", ["spec/verification/p07b-c-c3-receipt.json"]).length === 0,
		unexpectedPaths(specification, "C3B", ["internal/contractexec/target.go"])[0] === "internal/contractexec/target.go",
		unexpectedPaths(specification, "C3D", ["docs/status/P07B-C-C3D-RECEIPT-PHASE-CONSOLIDATION.md"]).length === 0,
		unexpectedPaths(specification, "C3D", ["spec/verification/p07b-c-c3-receipt.json"])[0] === "spec/verification/p07b-c-c3-receipt.json",
		unexpectedPaths(specification, "C3D", ["internal/contractexec/target.go"])[0] === "internal/contractexec/target.go",
		unexpectedPaths(specification, "C4M", ["docs/status/P07B-C-C4M-HISTORICAL-RECEIPT-FIXTURE-MAINTENANCE.md"]).length === 0,
		unexpectedPaths(specification, "C4M", ["internal/contractexec/target.go"])[0] === "internal/contractexec/target.go",
		unexpectedPaths(specification, "C4N", ["docs/status/P07B-C-C4N-SELF-RECEIPT-CARDINALITY-MAINTENANCE.md"]).length === 0,
		unexpectedPaths(specification, "C4N", ["internal/contractexec/target.go"])[0] === "internal/contractexec/target.go",
		unexpectedPaths(specification, "C4P", ["docs/status/P07B-C-C4P-FUTURE-SURFACE-PHASE-MAINTENANCE.md"]).length === 0,
		unexpectedPaths(specification, "C4P", ["internal/contractexec/target.go"])[0] === "internal/contractexec/target.go",
		unexpectedPaths(specification, "C1M", ["internal/world/process_darwin.go"]).length === 0,
		unexpectedPaths(specification, "C1M", ["spec/verification/p07b-c-c1-receipt.json"])[0] === "spec/verification/p07b-c-c1-receipt.json",
		exactPathsMatch(specification, "C1M", specification.units.C1M.exact),
		!exactPathsMatch(specification, "C1M", specification.units.C1M.exact.slice(1)),
		unexpectedPaths(specification, "C1V", ["tools/verify-current.mjs"]).length === 0,
		unexpectedPaths(specification, "C1V", ["spec/verification/p07b-c-c1-receipt.json"])[0] === "spec/verification/p07b-c-c1-receipt.json",
		exactPathsMatch(specification, "C1V", specification.units.C1V.exact),
		!exactPathsMatch(specification, "C1V", specification.units.C1V.exact.slice(1)),
		unexpectedPaths(specification, "C1E", ["tools/check-p07b-c-plan.mjs"]).length === 0,
		unexpectedPaths(specification, "C1E", ["spec/verification/p07b-c-c1-receipt.json"])[0] === "spec/verification/p07b-c-c1-receipt.json",
		exactPathsMatch(specification, "C1E", specification.units.C1E.exact),
		!exactPathsMatch(specification, "C1E", specification.units.C1E.exact.slice(1)),
		exactPathsMatch(specification, "C1B", specification.units.C1B.exact),
		!exactPathsMatch(specification, "C1B", specification.units.C1B.exact.slice(1)),
		exactPathsMatch(specification, "C2B", specification.units.C2B.exact),
		!exactPathsMatch(specification, "C2B", specification.units.C2B.exact.slice(1)),
		exactPathsMatch(specification, "C2M", specification.units.C2M.exact),
		!exactPathsMatch(specification, "C2M", specification.units.C2M.exact.slice(1)),
		exactPathsMatch(specification, "C3P", specification.units.C3P.exact),
		!exactPathsMatch(specification, "C3P", specification.units.C3P.exact.slice(1)),
		exactPathsMatch(specification, "C3V", specification.units.C3V.exact),
		!exactPathsMatch(specification, "C3V", specification.units.C3V.exact.slice(1)),
		exactPathsMatch(specification, "C3M", specification.units.C3M.exact),
		!exactPathsMatch(specification, "C3M", specification.units.C3M.exact.slice(1)),
		exactPathsMatch(specification, "C3A", specification.units.C3A.exact),
		!exactPathsMatch(specification, "C3A", specification.units.C3A.exact.slice(1)),
		exactPathsMatch(specification, "C3L", specification.units.C3L.exact),
		!exactPathsMatch(specification, "C3L", specification.units.C3L.exact.slice(1)),
		exactPathsMatch(specification, "C3F", specification.units.C3F.exact),
		!exactPathsMatch(specification, "C3F", specification.units.C3F.exact.slice(1)),
		exactPathsMatch(specification, "C3S", specification.units.C3S.exact),
		!exactPathsMatch(specification, "C3S", specification.units.C3S.exact.slice(1)),
		exactPathsMatch(specification, "C3", specification.units.C3.exact),
		!exactPathsMatch(specification, "C3", specification.units.C3.exact.slice(1)),
		exactPathsMatch(specification, "C3R", specification.units.C3R.exact),
		!exactPathsMatch(specification, "C3R", specification.units.C3R.exact.slice(1)),
		exactPathsMatch(specification, "C3Q", specification.units.C3Q.exact),
		!exactPathsMatch(specification, "C3Q", specification.units.C3Q.exact.slice(1)),
		exactPathsMatch(specification, "C3T", specification.units.C3T.exact),
		!exactPathsMatch(specification, "C3T", specification.units.C3T.exact.slice(1)),
		exactPathsMatch(specification, "C3U", specification.units.C3U.exact),
		!exactPathsMatch(specification, "C3U", specification.units.C3U.exact.slice(1)),
		exactPathsMatch(specification, "C3B", specification.units.C3B.exact),
		!exactPathsMatch(specification, "C3B", specification.units.C3B.exact.slice(1)),
		exactPathsMatch(specification, "C3D", specification.units.C3D.exact),
		!exactPathsMatch(specification, "C3D", specification.units.C3D.exact.slice(1)),
		exactPathsMatch(specification, "C4V", specification.units.C4V.exact),
		!exactPathsMatch(specification, "C4V", specification.units.C4V.exact.slice(1)),
		exactPathsMatch(specification, "C4M", specification.units.C4M.exact),
		!exactPathsMatch(specification, "C4M", specification.units.C4M.exact.slice(1)),
		exactPathsMatch(specification, "C4N", specification.units.C4N.exact),
		!exactPathsMatch(specification, "C4N", specification.units.C4N.exact.slice(1)),
		exactPathsMatch(specification, "C4P", specification.units.C4P.exact),
		!exactPathsMatch(specification, "C4P", specification.units.C4P.exact.slice(1)),
		isDeepStrictEqual(specification.units.C4.exact, c4DeclaredSourceContract.exact),
		isDeepStrictEqual(specification.units.C4.prefixes, c4DeclaredSourceContract.prefixes),
		exactPathsMatch(specification, "C4H", c4hDeclaredMaintenanceContract.exact),
		!exactPathsMatch(specification, "C4H", c4hDeclaredMaintenanceContract.exact.slice(1)),
		exactPathsMatch(specification, "C4I", c4iDeclaredMaintenanceContract.exact),
		!exactPathsMatch(specification, "C4I", c4iDeclaredMaintenanceContract.exact.slice(1)),
		exactPathsMatch(specification, "C4K", c4kDeclaredMaintenanceContract.exact),
		!exactPathsMatch(specification, "C4K", c4kDeclaredMaintenanceContract.exact.slice(1)),
		exactPathsMatch(specification, "C4J", c4jDeclaredMaintenanceContract.exact),
		!exactPathsMatch(specification, "C4J", c4jDeclaredMaintenanceContract.exact.slice(1)),
		exactPathsMatch(specification, "C4L", c4lDeclaredMaintenanceContract.exact),
		!exactPathsMatch(specification, "C4L", c4lDeclaredMaintenanceContract.exact.slice(1)),
		isDeepStrictEqual(specification.units.C5.exact, c5DeclaredSourceContract.exact),
		isDeepStrictEqual(specification.units.C5.prefixes, c5DeclaredSourceContract.prefixes),
		isDeepStrictEqual(specification.units.C5V.exact, c5vDeclaredSourceMaintenanceContract.exact),
		isDeepStrictEqual(specification.units.C5V.prefixes, c5vDeclaredSourceMaintenanceContract.prefixes),
		exactPathsMatch(specification, "C5V", c5vDeclaredSourceMaintenanceContract.exact),
		!exactPathsMatch(specification, "C5V", c5vDeclaredSourceMaintenanceContract.exact.slice(1)),
		exactPathsMatch(specification, "C6G", c6gDeclaredMaintenanceContract.exact),
		!exactPathsMatch(specification, "C6G", c6gDeclaredMaintenanceContract.exact.slice(1)),
		unexpectedPaths(specification, "C5V", ["docs/status/P07B-C-C5V-VERIFIER-PREFLIGHT-HYGIENE.md"]).length === 0,
		unexpectedPaths(specification, "C5V", ["tools/verify-runtime-authority.mjs"])[0] === "tools/verify-runtime-authority.mjs",
		unexpectedPaths(specification, "C4", ["tools/verify-runtime-authority.mjs"]).length === 0,
		unexpectedPaths(specification, "C4H", ["tools/verify-runtime-authority.mjs"])[0] === "tools/verify-runtime-authority.mjs",
		unexpectedPaths(specification, "C4I", ["tools/verify-runtime-authority.mjs"])[0] === "tools/verify-runtime-authority.mjs",
		unexpectedPaths(specification, "C4K", ["tools/verify-runtime-authority.mjs"])[0] === "tools/verify-runtime-authority.mjs",
		unexpectedPaths(specification, "C4J", ["tools/verify-runtime-authority.mjs"]).length === 0,
		unexpectedPaths(specification, "C4L", ["tools/verify-runtime-authority.mjs"])[0] === "tools/verify-runtime-authority.mjs",
		unexpectedPaths(specification, "C5", ["tools/verify-runtime-authority.mjs"])[0] === "tools/verify-runtime-authority.mjs",
		unexpectedPaths(specification, "C4", ["spec/verification/p07b-b-future-surface-authority.json"]).length === 0,
		unexpectedPaths(specification, "C5", ["spec/verification/p07b-b-future-surface-authority.json"])[0] ===
			"spec/verification/p07b-b-future-surface-authority.json",
		unexpectedPaths(specification, "C5", ["tools/verify-go-test-repetition.mjs"])[0] === "tools/verify-go-test-repetition.mjs",
		!exactPathsMatch(specification, "C3", specification.units.C3B.exact),
		!exactPathsMatch(specification, "C3B", specification.units.C3.exact),
		exactSourceGateAdmitted(specification, "C2M"),
		exactSourceGateAdmitted(specification, "C3P"),
		exactSourceGateAdmitted(specification, "C3V"),
		exactSourceGateAdmitted(specification, "C3M"),
		exactSourceGateAdmitted(specification, "C3A"),
		exactSourceGateAdmitted(specification, "C3L"),
		exactSourceGateAdmitted(specification, "C3F"),
		exactSourceGateAdmitted(specification, "C3S"),
		exactSourceGateAdmitted(specification, "C3"),
		exactSourceGateAdmitted(specification, "C3R"),
		exactSourceGateAdmitted(specification, "C3Q"),
		exactSourceGateAdmitted(specification, "C3T"),
		exactSourceGateAdmitted(specification, "C3U"),
		exactSourceGateAdmitted(specification, "C3D"),
		exactSourceGateAdmitted(specification, "C4V"),
		exactSourceGateAdmitted(specification, "C4M"),
		exactSourceGateAdmitted(specification, "C4N"),
		exactSourceGateAdmitted(specification, "C4P"),
		exactSourceGateAdmitted(specification, "C4H"),
		exactSourceGateAdmitted(specification, "C4I"),
		exactSourceGateAdmitted(specification, "C4K"),
		exactSourceGateAdmitted(specification, "C4J"),
		exactSourceGateAdmitted(specification, "C4L"),
		exactSourceGateAdmitted(specification, "C5V"),
		exactSourceGateAdmitted(specification, "C6F"),
		exactSourceGateAdmitted(specification, "C6G"),
		exactSourceGateAdmitted(specification, "C6M"),
		!exactSourceGateAdmitted(specification, "C2B"),
		!exactSourceGateAdmitted(specification, "C3B"),
		specification.units.C3.verification_profile === "SOURCE_FULL" && specification.units.C3.prefixes.length === 0,
		specification.units.C3R.verification_profile === "SOURCE_FULL" && specification.units.C3R.prefixes.length === 0,
		specification.units.C3Q.verification_profile === "SOURCE_FULL" && specification.units.C3Q.prefixes.length === 0,
		specification.units.C3T.verification_profile === "SOURCE_FULL" && specification.units.C3T.prefixes.length === 0,
		specification.units.C3U.verification_profile === "SOURCE_FULL" && specification.units.C3U.prefixes.length === 0,
		specification.units.C3B.verification_profile === "RECEIPT_RECONCILIATION" && specification.units.C3B.prefixes.length === 0,
		specification.units.C3D.verification_profile === "SOURCE_FULL" && specification.units.C3D.prefixes.length === 0,
		specification.units.C4M.verification_profile === "SOURCE_FULL" && specification.units.C4M.prefixes.length === 0,
		specification.units.C4N.verification_profile === "SOURCE_FULL" && specification.units.C4N.prefixes.length === 0,
		specification.units.C4P.verification_profile === "SOURCE_FULL" && specification.units.C4P.prefixes.length === 0,
		specification.units.C4H.verification_profile === "SOURCE_FULL" && specification.units.C4H.prefixes.length === 0,
		specification.units.C4I.verification_profile === "SOURCE_FULL" && specification.units.C4I.prefixes.length === 0,
		specification.units.C4K.verification_profile === "SOURCE_FULL" && specification.units.C4K.prefixes.length === 0,
		specification.units.C4J.verification_profile === "SOURCE_FULL" && specification.units.C4J.prefixes.length === 0,
		specification.units.C4L.verification_profile === "SOURCE_FULL" && specification.units.C4L.prefixes.length === 0,
		specification.units.C5V.verification_profile === "SOURCE_FULL" && specification.units.C5V.prefixes.length === 0,
		specification.units.C6G.verification_profile === "SOURCE_FULL" && specification.units.C6G.prefixes.length === 0,
		receiptPhaseRows(specification).length === 30,
		receiptPhaseRows(specification).at(-1).boundary === "C6B",
		receiptPhaseAuthorityDigest(specification) === receiptPhaseAuthoritySHA256,
		!["C4", "C5", "C6A", "C6M", "C6B"].some((boundary) =>
			specification.units[boundary].exact.includes("spec/verification/p07b-c-unit-paths.json")),
		specification.units.C4V.exact.includes("spec/verification/p07b-c-unit-paths.json"),
		specification.units.C4M.exact.includes("spec/verification/p07b-c-unit-paths.json"),
		specification.units.C4N.exact.includes("spec/verification/p07b-c-unit-paths.json"),
		specification.units.C4P.exact.includes("spec/verification/p07b-c-unit-paths.json"),
		specification.units.C4H.exact.includes("spec/verification/p07b-c-unit-paths.json"),
		specification.units.C4I.exact.includes("spec/verification/p07b-c-unit-paths.json"),
		specification.units.C4K.exact.includes("spec/verification/p07b-c-unit-paths.json"),
		specification.units.C4J.exact.includes("spec/verification/p07b-c-unit-paths.json"),
		specification.units.C4L.exact.includes("spec/verification/p07b-c-unit-paths.json"),
		specification.units.C5V.exact.includes("spec/verification/p07b-c-unit-paths.json"),
		specification.units.C6F.exact.includes("spec/verification/p07b-c-unit-paths.json"),
		specification.units.C6G.exact.includes("spec/verification/p07b-c-unit-paths.json"),
		exactPathsMatch(specification, "C6M", c6mDeclaredAdapterContract.exact),
		!specification.units.C6M.exact.includes("docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md"),
		!specification.units.C6M.exact.includes("docs/status/P07B-C-C6-EVIDENCE.md"),
		!specification.units.C6M.exact.includes("tools/check-p07b-c-unit-scope.mjs"),
		!specification.units.C6M.exact.includes("tools/verify-current.mjs"),
		activeReceiptPhaseBoundary(specification, "C3D") === "C3D",
		unexpectedPaths(specification, "C0A", ["docs/SEMANTICS.md", "README.md"])[0] === "README.md",
		unexpectedPaths(specification, "C6A", ["docs/captures/p07b-c/capture.json"]).length === 0,
		unexpectedPaths(specification, "C6A", ["docs/captures/p07b-c/authority.md"])[0] ===
			"docs/captures/p07b-c/authority.md",
		unexpectedPaths(specification, "C6A", ["docs/captures/p07b-c/authority.MD"])[0] ===
			"docs/captures/p07b-c/authority.MD",
		unexpectedPaths(specification, "C6A", ["docs/captures/p07b-c/authority.markdown"])[0] ===
			"docs/captures/p07b-c/authority.markdown",
		markdownAuthorityPath("docs/future-authority.md"),
		markdownAuthorityPath("docs/future-authority.MD"),
		markdownAuthorityPath("docs/future-authority.markdown"),
		markdownAuthorityPath("docs/future-authority.MARKDOWN"),
		!markdownAuthorityPath("docs/future-authority.md.txt"),
		exactPathsMatch(specification, "C6B", c6bDeclaredReceiptContract.exact),
		!exactPathsMatch(specification, "C6B", c6bDeclaredReceiptContract.exact.slice(1)),
		JSON.stringify(stagedInventoryArgs) === JSON.stringify([
			"diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--cached", "--name-only", "-z", "--no-renames", "--diff-filter=ACDMRTUXB", "--",
		]),
		JSON.stringify(stagedIndexArgs) === JSON.stringify(["ls-files", "--stage", "-z", "--"]),
		receiptManifest(specification, "C1B").length === 7,
		receiptManifest(specification, "C2B").length === 7,
		receiptManifest(specification, "C3PB").length === 7,
		receiptManifest(specification, "C3B").length === 7,
		receiptManifest(specification, "C6B").length === 9,
		JSON.stringify(receiptManifest(specification, "C3B")) === JSON.stringify(c3FrozenUnitContracts.C3B.receipt_claims),
		JSON.stringify(receiptManifest(specification, "C6B")) === JSON.stringify(c6bDeclaredReceiptContract.receipt_claims),
	];
	if (cases.some((value) => !value)) fail("allow/refuse self-test matrix");
	for (const boundary of forwardCandidateReceiptPhaseBoundaries.slice(1)) {
		if (activeReceiptPhaseBoundary(specification, boundary) !== boundary) {
			fail(`${boundary} visible-capsule active-boundary pointer self-test`);
		}
	}
	requireSpecificationMutationRejected(specification, "schema v27 downgrade", (hostile) => {
		hostile.schema_version = "countershape/p07b-c-unit-paths/v27";
	});
	requireSpecificationMutationRejected(specification, "duplicated specification active boundary", (hostile) => {
		hostile.active_boundary = "C3D";
	});
	for (const boundary of [undefined, "UNKNOWN"]) {
		let rejected = false;
		try { activeReceiptPhaseBoundary(specification, boundary); } catch { rejected = true; }
		if (!rejected) fail(`invalid visible-capsule active-boundary pointer self-test: ${String(boundary)}`);
	}
	let c3ReceiptManifestRejected = false;
	try { receiptManifest(specification, "C3"); } catch { c3ReceiptManifestRejected = true; }
	if (!c3ReceiptManifestRejected) fail("C3 source profile receipt-manifest self-test false negative");
	requireSpecificationMutationRejected(specification, "C3 receipt-profile substitution", (hostile) => {
		hostile.units.C3.verification_profile = "RECEIPT_RECONCILIATION";
		hostile.units.C3.receipt_claims = [
			{ label: "P07B-C C3 source receipt reconciliation", type: "tests-pass" },
			{ label: "P07B-C C3 receipt checker defensive self-test", type: "tests-pass" },
			{ label: "P07B-C C3 declared local evidence snapshot match", type: "tests-pass" },
			{ label: "P07B-C C3 receipt-only Go build", type: "command-succeeded" },
			{ label: "P07B-C C3 exact receipt staged scope and diff integrity", type: "command-succeeded" },
			{ label: "P07B-C C3 scoped staged credential-pattern scan", type: "command-succeeded" },
			{ label: "P07B-C C3 preceding didrun chain integrity", type: "command-succeeded" },
		];
	});
	requireSpecificationMutationRejected(specification, "C3 prefix introduction", (hostile) => { hostile.units.C3.prefixes = ["foreign/"]; });
	requireSpecificationMutationRejected(specification, "C3B source-profile substitution", (hostile) => {
		hostile.units.C3B.verification_profile = "SOURCE_FULL";
		delete hostile.units.C3B.receipt_claims;
	});
	requireSpecificationMutationRejected(specification, "C3B prefix introduction", (hostile) => { hostile.units.C3B.prefixes = ["foreign/"]; });
	requireSpecificationMutationRejected(specification, "C3B claim order", (hostile) => {
		[hostile.units.C3B.receipt_claims[0], hostile.units.C3B.receipt_claims[1]] =
			[hostile.units.C3B.receipt_claims[1], hostile.units.C3B.receipt_claims[0]];
	});
	requireSpecificationMutationRejected(specification, "C3B claim type", (hostile) => {
		hostile.units.C3B.receipt_claims[0].type = "command-succeeded";
	});
	requireSpecificationMutationRejected(specification, "C3B claim label", (hostile) => {
		hostile.units.C3B.receipt_claims[0].label = "P07B-C C3B altered source receipt reconciliation";
	});
	requireSpecificationMutationRejected(specification, "C3/C3B unit order", (hostile) => {
		const entries = Object.entries(hostile.units);
		const c3 = entries.findIndex(([unit]) => unit === "C3");
		const c3b = entries.findIndex(([unit]) => unit === "C3B");
		[entries[c3], entries[c3b]] = [entries[c3b], entries[c3]];
		hostile.units = Object.fromEntries(entries);
	});
	requireSpecificationMutationRejected(specification, "C3D missing parent", (hostile) => {
		delete hostile.units.C3D.parent;
	});
	requireSpecificationMutationRejected(specification, "C3D wrong parent", (hostile) => {
		hostile.units.C3D.parent = "C3U";
	});
	requireSpecificationMutationRejected(specification, "C3D missing receipt key", (hostile) => {
		delete hostile.units.C3D.receipt_states.C6A;
	});
	requireSpecificationMutationRejected(specification, "C3D unknown receipt key", (hostile) => {
		hostile.units.C3D.receipt_states.UNKNOWN = "ABSENT";
	});
	requireSpecificationMutationRejected(specification, "C3D invalid receipt state", (hostile) => {
		hostile.units.C3D.receipt_states.C3 = "SEALED";
	});
	requireSpecificationMutationRejected(specification, "C3D source phase state flip", (hostile) => {
		hostile.units.C3D.receipt_states.C6A = "PRESENT";
	});
	requireSpecificationMutationRejected(specification, "C3B receipt phase no flip", (hostile) => {
		hostile.units.C3B.receipt_states.C3 = "ABSENT";
	});
	requireSpecificationMutationRejected(specification, "C3B receipt phase multiple flips", (hostile) => {
		hostile.units.C3B.receipt_states.C3 = "ABSENT";
		hostile.units.C3B.receipt_states.C6A = "PRESENT";
	});
	requireSpecificationMutationRejected(specification, "C4 receipt state downgrade", (hostile) => {
		hostile.units.C4.receipt_states.C3 = "ABSENT";
	});
	requireSpecificationMutationRejected(specification, "C4 runtime authority removal", (hostile) => {
		hostile.units.C4.exact = hostile.units.C4.exact.filter((path) => path !== "tools/verify-runtime-authority.mjs");
	});
	requireSpecificationMutationRejected(specification, "C4 future-surface manifest removal", (hostile) => {
		hostile.units.C4.exact = hostile.units.C4.exact.filter((path) =>
			path !== "spec/verification/p07b-b-future-surface-authority.json");
	});
	requireSpecificationMutationRejected(specification, "C4H profile drift", (hostile) => {
		hostile.units.C4H.verification_profile = "RECEIPT_RECONCILIATION";
	});
	requireSpecificationMutationRejected(specification, "C4H parent drift", (hostile) => {
		hostile.units.C4H.parent = "C4P";
	});
	requireSpecificationMutationRejected(specification, "C4H exact substitution", (hostile) => {
		hostile.units.C4H.exact[0] = "docs/ADVERSARY.md";
	});
	requireSpecificationMutationRejected(specification, "C4H runtime authority capture", (hostile) => {
		hostile.units.C4H.exact.push("tools/verify-runtime-authority.mjs");
		hostile.units.C4H.exact.sort();
	});
	requireSpecificationMutationRejected(specification, "C4H receipt state drift", (hostile) => {
		hostile.units.C4H.receipt_states.C6A = "PRESENT";
	});
	requireSpecificationMutationRejected(specification, "C4I profile drift", (hostile) => {
		hostile.units.C4I.verification_profile = "RECEIPT_RECONCILIATION";
	});
	requireSpecificationMutationRejected(specification, "C4I parent drift", (hostile) => {
		hostile.units.C4I.parent = "C4";
	});
	requireSpecificationMutationRejected(specification, "C4I exact substitution", (hostile) => {
		hostile.units.C4I.exact[0] = "docs/ADVERSARY.md";
	});
	requireSpecificationMutationRejected(specification, "C4I runtime authority capture", (hostile) => {
		hostile.units.C4I.exact.push("tools/verify-runtime-authority.mjs");
		hostile.units.C4I.exact.sort();
	});
	requireSpecificationMutationRejected(specification, "C4I receipt state drift", (hostile) => {
		hostile.units.C4I.receipt_states.C6A = "PRESENT";
	});
	requireSpecificationMutationRejected(specification, "C4K profile drift", (hostile) => {
		hostile.units.C4K.verification_profile = "RECEIPT_RECONCILIATION";
	});
	requireSpecificationMutationRejected(specification, "C4K parent drift", (hostile) => {
		hostile.units.C4K.parent = "C4H";
	});
	requireSpecificationMutationRejected(specification, "C4K exact substitution", (hostile) => {
		hostile.units.C4K.exact[0] = "docs/ADVERSARY.md";
	});
	requireSpecificationMutationRejected(specification, "C4K architecture checker removal", (hostile) => {
		hostile.units.C4K.exact = hostile.units.C4K.exact.filter((path) => path !== "tools/check-p07b-c-architecture.mjs");
	});
	requireSpecificationMutationRejected(specification, "C4K architecture self-test removal", (hostile) => {
		hostile.units.C4K.exact = hostile.units.C4K.exact.filter((path) => path !== "tools/check-p07b-c-architecture-selftest.mjs");
	});
	requireSpecificationMutationRejected(specification, "C4K receipt state drift", (hostile) => {
		hostile.units.C4K.receipt_states.C6A = "PRESENT";
	});
	requireSpecificationMutationRejected(specification, "C4J profile drift", (hostile) => {
		hostile.units.C4J.verification_profile = "RECEIPT_RECONCILIATION";
	});
	requireSpecificationMutationRejected(specification, "C4J parent drift", (hostile) => {
		hostile.units.C4J.parent = "C4I";
	});
	requireSpecificationMutationRejected(specification, "C4J exact substitution", (hostile) => {
		hostile.units.C4J.exact[0] = "docs/ADVERSARY.md";
	});
	requireSpecificationMutationRejected(specification, "C4J runtime authority removal", (hostile) => {
		hostile.units.C4J.exact = hostile.units.C4J.exact.filter((path) => path !== "tools/verify-runtime-authority.mjs");
	});
	requireSpecificationMutationRejected(specification, "C4J receipt state drift", (hostile) => {
		hostile.units.C4J.receipt_states.C6A = "PRESENT";
	});
	requireSpecificationMutationRejected(specification, "C4J transition authority removal", (hostile) => {
		delete hostile.units.C4J.transition_authority;
	});
	requireSpecificationMutationRejected(specification, "C4J owner-instruction overclaim", (hostile) => {
		hostile.units.C4J.transition_authority.provenance = {
			kind: "CITED_UNAUTHENTICATED",
			artifact_role: "OWNER_INSTRUCTION_ARTIFACT",
			preexistence: "DIRECT_PARENT_TREE",
			artifact_path: "docs/status/P07B-C-C4K-C3-GO-TIMEOUT-MAINTENANCE.md",
			artifact_sha256: "sha256:fc80148ddedddb925962a15dacb20dad8979426aa3f3ef4f8fef12df9c614d6b",
		};
	});
	requireSpecificationMutationRejected(specification, "C4L profile self-bootstrap", (hostile) => {
		hostile.units.C4L.verification_profile = "NON_PRODUCT_MAINTENANCE";
		hostile.units.C4L.product_authority = "NONE";
		hostile.units.C4L.product_behavior = "INHERITED_UNREPROVEN";
		hostile.units.C4L.product_projection = "PARENT_FROZEN";
		hostile.units.C4L.maintenance_claims = nonProductMaintenanceClaimRoles.map((role, index) => ({
			label: nonProductMaintenanceClaimLabel("C4L", role),
			role,
			type: index < 6 ? "tests-pass" : "command-succeeded",
		}));
	});
	requireSpecificationMutationRejected(specification, "C4L transition authority removal", (hostile) => {
		delete hostile.units.C4L.transition_authority;
	});
	requireSpecificationMutationRejected(specification, "C4L parent drift", (hostile) => {
		hostile.units.C4L.parent = "C4K";
	});
	requireSpecificationMutationRejected(specification, "C4L exact substitution", (hostile) => {
		hostile.units.C4L.exact[0] = "docs/ADVERSARY.md";
	});
	requireSpecificationMutationRejected(specification, "C4L receipt state drift", (hostile) => {
		hostile.units.C4L.receipt_states.C6A = "PRESENT";
	});
	requireSpecificationMutationRejected(specification, "C5 profile drift", (hostile) => {
		hostile.units.C5.verification_profile = "RECEIPT_RECONCILIATION";
	});
	requireSpecificationMutationRejected(specification, "C5 stale C4J parent drift", (hostile) => {
		hostile.units.C5.parent = "C4J";
	});
	requireSpecificationMutationRejected(specification, "C5 exact substitution", (hostile) => {
		hostile.units.C5.exact[0] = "docs/ADVERSARY.md";
	});
	requireSpecificationMutationRejected(specification, "C5 runtime authority capture", (hostile) => {
		hostile.units.C5.exact.push("tools/verify-runtime-authority.mjs");
		hostile.units.C5.exact.sort();
	});
	requireSpecificationMutationRejected(specification, "C5 prefix order drift", (hostile) => {
		[hostile.units.C5.prefixes[0], hostile.units.C5.prefixes[1]] =
			[hostile.units.C5.prefixes[1], hostile.units.C5.prefixes[0]];
	});
	requireSpecificationMutationRejected(specification, "C5 prefix removal", (hostile) => {
		hostile.units.C5.prefixes.pop();
	});
	requireSpecificationMutationRejected(specification, "C5 receipt state drift", (hostile) => {
		hostile.units.C5.receipt_states.C6A = "PRESENT";
	});
	requireSpecificationMutationRejected(specification, "C5V profile drift", (hostile) => {
		hostile.units.C5V.verification_profile = "RECEIPT_RECONCILIATION";
	});
	requireSpecificationMutationRejected(specification, "C5V parent drift", (hostile) => {
		hostile.units.C5V.parent = "C4L";
	});
	requireSpecificationMutationRejected(specification, "C5V exact substitution", (hostile) => {
		hostile.units.C5V.exact[0] = "docs/ADVERSARY.md";
	});
	requireSpecificationMutationRejected(specification, "C5V specification ownership removal", (hostile) => {
		hostile.units.C5V.exact = hostile.units.C5V.exact.filter((path) =>
			path !== "spec/verification/p07b-c-unit-paths.json");
	});
	requireSpecificationMutationRejected(specification, "C5V verifier self-test removal", (hostile) => {
		hostile.units.C5V.exact = hostile.units.C5V.exact.filter((path) =>
			path !== "tools/verify-current-selftest.mjs");
	});
	requireSpecificationMutationRejected(specification, "C5V prefix introduction", (hostile) => {
		hostile.units.C5V.prefixes = ["foreign/"];
	});
	requireSpecificationMutationRejected(specification, "C5V receipt state drift", (hostile) => {
		hostile.units.C5V.receipt_states.C6A = "PRESENT";
	});
	requireSpecificationMutationRejected(specification, "C5V transition authority removal", (hostile) => {
		delete hostile.units.C5V.transition_authority;
	});
	requireSpecificationMutationRejected(specification, "C5V owner-instruction overclaim", (hostile) => {
		hostile.units.C5V.transition_authority.provenance = {
			kind: "CITED_UNAUTHENTICATED",
			artifact_role: "OWNER_INSTRUCTION_ARTIFACT",
			preexistence: "DIRECT_PARENT_TREE",
			artifact_path: "docs/status/P07B-C-C5V-VERIFIER-PREFLIGHT-HYGIENE.md",
			artifact_sha256: `sha256:${"a".repeat(64)}`,
		};
	});
	requireSpecificationMutationRejected(specification, "C6F profile drift", (hostile) => {
		hostile.units.C6F.verification_profile = "RECEIPT_RECONCILIATION";
	});
	requireSpecificationMutationRejected(specification, "C6F parent drift", (hostile) => {
		hostile.units.C6F.parent = "C5V";
	});
	requireSpecificationMutationRejected(specification, "C6F exact substitution", (hostile) => {
		hostile.units.C6F.exact[0] = "docs/ADVERSARY.md";
	});
	requireSpecificationMutationRejected(specification, "C6F transition authority removal", (hostile) => {
		delete hostile.units.C6F.transition_authority;
	});
	requireSpecificationMutationRejected(specification, "C6F receipt state drift", (hostile) => {
		hostile.units.C6F.receipt_states.C6A = "PRESENT";
	});
	requireSpecificationMutationRejected(specification, "C6G profile drift", (hostile) => {
		hostile.units.C6G.verification_profile = "RECEIPT_RECONCILIATION";
	});
	requireSpecificationMutationRejected(specification, "C6G parent drift", (hostile) => {
		hostile.units.C6G.parent = "C6A";
	});
	requireSpecificationMutationRejected(specification, "C6G exact substitution", (hostile) => {
		hostile.units.C6G.exact[0] = "docs/ADVERSARY.md";
	});
	requireSpecificationMutationRejected(specification, "C6G transition authority removal", (hostile) => {
		delete hostile.units.C6G.transition_authority;
	});
	requireSpecificationMutationRejected(specification, "C6G receipt state drift", (hostile) => {
		hostile.units.C6G.receipt_states.C6A = "PRESENT";
	});
	requireSpecificationMutationRejected(specification, "C6M stale C6F parent", (hostile) => {
		hostile.units.C6M.parent = "C6F";
	});
	requireSpecificationMutationRejected(specification, "C6M stale C6A parent", (hostile) => {
		hostile.units.C6M.parent = "C6A";
	});
	requireSpecificationMutationRejected(specification, "C6A stale C5 parent", (hostile) => {
		hostile.units.C6A.parent = "C5";
	});
	for (const boundary of forwardCandidateReceiptPhaseBoundaries.filter((boundary) =>
		specification.units[boundary].verification_profile === "SOURCE_FULL")) {
		requireSpecificationMutationRejected(specification, `${boundary} declared exact path removal`, (hostile) => {
			hostile.units[boundary].exact = hostile.units[boundary].exact.slice(1);
		});
		requireSpecificationMutationRejected(specification, `${boundary} declared prefix drift`, (hostile) => {
			hostile.units[boundary].prefixes = ["foreign/"];
		});
	}
	requireSpecificationMutationRejected(specification, "C6B exact path removal", (hostile) => {
		hostile.units.C6B.exact = hostile.units.C6B.exact.filter((path) =>
			path !== "docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md");
	});
	requireSpecificationMutationRejected(specification, "C6B receipt declaration removal", (hostile) => {
		hostile.units.C6B.exact = hostile.units.C6B.exact.filter((path) => path !== c6aReceiptDeclarationPath);
	});
	requireSpecificationMutationRejected(specification, "C6M receipt-surface ownership", (hostile) => {
		hostile.units.C6M.exact = [...hostile.units.C6M.exact, "docs/status/P07B-C-C6-EVIDENCE.md"].sort();
	});
	requireSpecificationMutationRejected(specification, "C6B HANDOFF cursor removal", (hostile) => {
		hostile.units.C6B.exact = hostile.units.C6B.exact.filter((path) => path !== "docs/HANDOFF_MODE_C.md");
	});
	requireSpecificationMutationRejected(specification, "C6B receipt claim drift", (hostile) => {
		hostile.units.C6B.receipt_claims[6].label = "P07B-C C6B exact four-path staged scope and diff integrity";
	});
	requireSpecificationMutationRejected(specification, "C6B prefix introduction", (hostile) => {
		hostile.units.C6B.prefixes = ["docs/"];
	});
	const unsafePrefix = JSON.parse(JSON.stringify(specification));
	unsafePrefix.units.C4.prefixes = ["internal/processmechanics"];
	let unsafePrefixRejected = false;
	try {
		validateSpecification(unsafePrefix);
	} catch {
		unsafePrefixRejected = true;
	}
	if (!unsafePrefixRejected) fail("lexical prefix self-test false negative");
	const unsafeReceiptProfile = JSON.parse(JSON.stringify(specification));
	unsafeReceiptProfile.units.C1V.verification_profile = "RECEIPT_RECONCILIATION";
	let unsafeReceiptProfileRejected = false;
	try {
		validateSpecification(unsafeReceiptProfile);
	} catch {
		unsafeReceiptProfileRejected = true;
	}
	if (!unsafeReceiptProfileRejected) fail("receipt reconciliation source-scope self-test false negative");
	const unsafeC3VReceiptProfile = JSON.parse(JSON.stringify(specification));
	unsafeC3VReceiptProfile.units.C3V.verification_profile = "RECEIPT_RECONCILIATION";
	unsafeC3VReceiptProfile.units.C3V.receipt_claims = JSON.parse(JSON.stringify(specification.units.C3PB.receipt_claims));
	let unsafeC3VReceiptProfileRejected = false;
	try {
		validateSpecification(unsafeC3VReceiptProfile);
	} catch {
		unsafeC3VReceiptProfileRejected = true;
	}
	if (!unsafeC3VReceiptProfileRejected) fail("C3V receipt-profile substitution self-test false negative");
	const unsafeC3VPrefix = JSON.parse(JSON.stringify(specification));
	unsafeC3VPrefix.units.C3V.prefixes = ["docs/"];
	let unsafeC3VPrefixRejected = false;
	try {
		validateSpecification(unsafeC3VPrefix);
	} catch {
		unsafeC3VPrefixRejected = true;
	}
	if (!unsafeC3VPrefixRejected) fail("C3V prefix introduction self-test false negative");
	const unsafeC3MReceiptProfile = JSON.parse(JSON.stringify(specification));
	unsafeC3MReceiptProfile.units.C3M.verification_profile = "RECEIPT_RECONCILIATION";
	unsafeC3MReceiptProfile.units.C3M.receipt_claims = JSON.parse(JSON.stringify(specification.units.C3PB.receipt_claims));
	let unsafeC3MReceiptProfileRejected = false;
	try {
		validateSpecification(unsafeC3MReceiptProfile);
	} catch {
		unsafeC3MReceiptProfileRejected = true;
	}
	if (!unsafeC3MReceiptProfileRejected) fail("C3M receipt-profile substitution self-test false negative");
	const unsafeC3MPrefix = JSON.parse(JSON.stringify(specification));
	unsafeC3MPrefix.units.C3M.prefixes = ["docs/"];
	let unsafeC3MPrefixRejected = false;
	try {
		validateSpecification(unsafeC3MPrefix);
	} catch {
		unsafeC3MPrefixRejected = true;
	}
	if (!unsafeC3MPrefixRejected) fail("C3M prefix introduction self-test false negative");
	const unsafeC3AReceiptProfile = JSON.parse(JSON.stringify(specification));
	unsafeC3AReceiptProfile.units.C3A.verification_profile = "RECEIPT_RECONCILIATION";
	unsafeC3AReceiptProfile.units.C3A.receipt_claims = JSON.parse(JSON.stringify(specification.units.C3PB.receipt_claims));
	let unsafeC3AReceiptProfileRejected = false;
	try {
		validateSpecification(unsafeC3AReceiptProfile);
	} catch {
		unsafeC3AReceiptProfileRejected = true;
	}
	if (!unsafeC3AReceiptProfileRejected) fail("C3A receipt-profile substitution self-test false negative");
	const unsafeC3APrefix = JSON.parse(JSON.stringify(specification));
	unsafeC3APrefix.units.C3A.prefixes = ["docs/"];
	let unsafeC3APrefixRejected = false;
	try {
		validateSpecification(unsafeC3APrefix);
	} catch {
		unsafeC3APrefixRejected = true;
	}
	if (!unsafeC3APrefixRejected) fail("C3A prefix introduction self-test false negative");
	const unsafeC3LReceiptProfile = JSON.parse(JSON.stringify(specification));
	unsafeC3LReceiptProfile.units.C3L.verification_profile = "RECEIPT_RECONCILIATION";
	unsafeC3LReceiptProfile.units.C3L.receipt_claims = JSON.parse(JSON.stringify(specification.units.C3PB.receipt_claims));
	let unsafeC3LReceiptProfileRejected = false;
	try {
		validateSpecification(unsafeC3LReceiptProfile);
	} catch {
		unsafeC3LReceiptProfileRejected = true;
	}
	if (!unsafeC3LReceiptProfileRejected) fail("C3L receipt-profile substitution self-test false negative");
	const unsafeC3LPrefix = JSON.parse(JSON.stringify(specification));
	unsafeC3LPrefix.units.C3L.prefixes = ["docs/"];
	let unsafeC3LPrefixRejected = false;
	try {
		validateSpecification(unsafeC3LPrefix);
	} catch {
		unsafeC3LPrefixRejected = true;
	}
	if (!unsafeC3LPrefixRejected) fail("C3L prefix introduction self-test false negative");
	const unsafeC3FReceiptProfile = JSON.parse(JSON.stringify(specification));
	unsafeC3FReceiptProfile.units.C3F.verification_profile = "RECEIPT_RECONCILIATION";
	unsafeC3FReceiptProfile.units.C3F.receipt_claims = JSON.parse(JSON.stringify(specification.units.C3PB.receipt_claims));
	let unsafeC3FReceiptProfileRejected = false;
	try {
		validateSpecification(unsafeC3FReceiptProfile);
	} catch {
		unsafeC3FReceiptProfileRejected = true;
	}
	if (!unsafeC3FReceiptProfileRejected) fail("C3F receipt-profile substitution self-test false negative");
	const unsafeC3FPrefix = JSON.parse(JSON.stringify(specification));
	unsafeC3FPrefix.units.C3F.prefixes = ["tools/"];
	let unsafeC3FPrefixRejected = false;
	try {
		validateSpecification(unsafeC3FPrefix);
	} catch {
		unsafeC3FPrefixRejected = true;
	}
	if (!unsafeC3FPrefixRejected) fail("C3F prefix introduction self-test false negative");
	const unsafeC3SReceiptProfile = JSON.parse(JSON.stringify(specification));
	unsafeC3SReceiptProfile.units.C3S.verification_profile = "RECEIPT_RECONCILIATION";
	unsafeC3SReceiptProfile.units.C3S.receipt_claims = JSON.parse(JSON.stringify(specification.units.C3PB.receipt_claims));
	let unsafeC3SReceiptProfileRejected = false;
	try {
		validateSpecification(unsafeC3SReceiptProfile);
	} catch {
		unsafeC3SReceiptProfileRejected = true;
	}
	if (!unsafeC3SReceiptProfileRejected) fail("C3S receipt-profile substitution self-test false negative");
	const unsafeC3SPrefix = JSON.parse(JSON.stringify(specification));
	unsafeC3SPrefix.units.C3S.prefixes = ["tools/"];
	let unsafeC3SPrefixRejected = false;
	try {
		validateSpecification(unsafeC3SPrefix);
	} catch {
		unsafeC3SPrefixRejected = true;
	}
	if (!unsafeC3SPrefixRejected) fail("C3S prefix introduction self-test false negative");
	requireSpecificationMutationRejected(specification, "C3R receipt-profile substitution", (hostile) => {
		hostile.units.C3R.verification_profile = "RECEIPT_RECONCILIATION";
		hostile.units.C3R.receipt_claims = JSON.parse(JSON.stringify(specification.units.C3B.receipt_claims));
	});
	requireSpecificationMutationRejected(specification, "C3R prefix introduction", (hostile) => {
		hostile.units.C3R.prefixes = ["tools/"];
	});
	requireSpecificationMutationRejected(specification, "C3R path deletion", (hostile) => {
		hostile.units.C3R.exact.shift();
	});
	requireSpecificationMutationRejected(specification, "C3Q receipt-profile substitution", (hostile) => {
		hostile.units.C3Q.verification_profile = "RECEIPT_RECONCILIATION";
		hostile.units.C3Q.receipt_claims = JSON.parse(JSON.stringify(specification.units.C3B.receipt_claims));
	});
	requireSpecificationMutationRejected(specification, "C3Q prefix introduction", (hostile) => {
		hostile.units.C3Q.prefixes = ["tools/"];
	});
	requireSpecificationMutationRejected(specification, "C3Q path deletion", (hostile) => {
		hostile.units.C3Q.exact.shift();
	});
	requireSpecificationMutationRejected(specification, "C3T receipt-profile substitution", (hostile) => {
		hostile.units.C3T.verification_profile = "RECEIPT_RECONCILIATION";
		hostile.units.C3T.exact = [
			"docs/HANDOFF_MODE_C.md",
			"docs/PROMPT_PACK.md",
			"docs/VERIFICATION.md",
			"docs/status/P07B-C-C3T-PHASE-SELFTEST-MAINTENANCE.md",
			"docs/status/P07B-C-C3T-RECEIPT-CLAIMS.md",
			"docs/status/P07B-C-C3T-RECEIPT-EVIDENCE.md",
			"docs/status/P07B-C-C3T-RECEIPT-SCOPE.md",
		];
		hostile.units.C3T.receipt_claims = [
			{ label: "P07B-C C3T source receipt reconciliation", type: "tests-pass" },
			{ label: "P07B-C C3T receipt checker defensive self-test", type: "tests-pass" },
			{ label: "P07B-C C3T declared local evidence snapshot match", type: "tests-pass" },
			{ label: "P07B-C C3T receipt-only Go build", type: "command-succeeded" },
			{ label: "P07B-C C3T exact staged scope and diff integrity", type: "command-succeeded" },
			{ label: "P07B-C C3T scoped staged credential-pattern scan", type: "command-succeeded" },
			{ label: "P07B-C C3T preceding didrun chain integrity", type: "command-succeeded" },
		];
	});
	requireSpecificationMutationRejected(specification, "C3T prefix introduction", (hostile) => {
		hostile.units.C3T.prefixes = ["tools/"];
	});
	requireSpecificationMutationRejected(specification, "C3T path deletion", (hostile) => {
		hostile.units.C3T.exact.shift();
	});
	requireSpecificationMutationRejected(specification, "C3U receipt-profile substitution", (hostile) => {
		hostile.units.C3U.verification_profile = "RECEIPT_RECONCILIATION";
		hostile.units.C3U.receipt_claims = JSON.parse(JSON.stringify(specification.units.C3B.receipt_claims));
	});
	requireSpecificationMutationRejected(specification, "C3U prefix introduction", (hostile) => {
		hostile.units.C3U.prefixes = ["tools/"];
	});
	requireSpecificationMutationRejected(specification, "C3U path deletion", (hostile) => {
		hostile.units.C3U.exact.shift();
	});
	const reorderedUnits = JSON.parse(JSON.stringify(specification));
	const reorderedC2B = reorderedUnits.units.C2B;
	delete reorderedUnits.units.C2B;
	reorderedUnits.units.C2B = reorderedC2B;
	let reorderedUnitsRejected = false;
	try { validateSpecification(reorderedUnits); } catch { reorderedUnitsRejected = true; }
	if (!reorderedUnitsRejected) fail("unit order self-test false negative");
	const reorderedC3MUnits = JSON.parse(JSON.stringify(specification));
	const reorderedC3MEntries = Object.entries(reorderedC3MUnits.units);
	const c3mIndex = reorderedC3MEntries.findIndex(([unit]) => unit === "C3M");
	const c3pbIndex = reorderedC3MEntries.findIndex(([unit]) => unit === "C3PB");
	[reorderedC3MEntries[c3mIndex], reorderedC3MEntries[c3pbIndex]] = [reorderedC3MEntries[c3pbIndex], reorderedC3MEntries[c3mIndex]];
	reorderedC3MUnits.units = Object.fromEntries(reorderedC3MEntries);
	let reorderedC3MRejected = false;
	try { validateSpecification(reorderedC3MUnits); } catch { reorderedC3MRejected = true; }
	if (!reorderedC3MRejected) fail("C3M order self-test false negative");
	const reorderedC3AUnits = JSON.parse(JSON.stringify(specification));
	const reorderedC3AEntries = Object.entries(reorderedC3AUnits.units);
	const c3aIndex = reorderedC3AEntries.findIndex(([unit]) => unit === "C3A");
	const c3Index = reorderedC3AEntries.findIndex(([unit]) => unit === "C3");
	[reorderedC3AEntries[c3aIndex], reorderedC3AEntries[c3Index]] = [reorderedC3AEntries[c3Index], reorderedC3AEntries[c3aIndex]];
	reorderedC3AUnits.units = Object.fromEntries(reorderedC3AEntries);
	let reorderedC3ARejected = false;
	try { validateSpecification(reorderedC3AUnits); } catch { reorderedC3ARejected = true; }
	if (!reorderedC3ARejected) fail("C3A order self-test false negative");
	const reorderedC3LUnits = JSON.parse(JSON.stringify(specification));
	const reorderedC3LEntries = Object.entries(reorderedC3LUnits.units);
	const c3lIndex = reorderedC3LEntries.findIndex(([unit]) => unit === "C3L");
	const c3AfterLIndex = reorderedC3LEntries.findIndex(([unit]) => unit === "C3");
	[reorderedC3LEntries[c3lIndex], reorderedC3LEntries[c3AfterLIndex]] = [reorderedC3LEntries[c3AfterLIndex], reorderedC3LEntries[c3lIndex]];
	reorderedC3LUnits.units = Object.fromEntries(reorderedC3LEntries);
	let reorderedC3LRejected = false;
	try { validateSpecification(reorderedC3LUnits); } catch { reorderedC3LRejected = true; }
	if (!reorderedC3LRejected) fail("C3L order self-test false negative");
	const reorderedC3FUnits = JSON.parse(JSON.stringify(specification));
	const reorderedC3FEntries = Object.entries(reorderedC3FUnits.units);
	const c3fIndex = reorderedC3FEntries.findIndex(([unit]) => unit === "C3F");
	const c3AfterFIndex = reorderedC3FEntries.findIndex(([unit]) => unit === "C3");
	[reorderedC3FEntries[c3fIndex], reorderedC3FEntries[c3AfterFIndex]] = [reorderedC3FEntries[c3AfterFIndex], reorderedC3FEntries[c3fIndex]];
	reorderedC3FUnits.units = Object.fromEntries(reorderedC3FEntries);
	let reorderedC3FRejected = false;
	try { validateSpecification(reorderedC3FUnits); } catch { reorderedC3FRejected = true; }
	if (!reorderedC3FRejected) fail("C3F order self-test false negative");
	const reorderedC3SUnits = JSON.parse(JSON.stringify(specification));
	const reorderedC3SEntries = Object.entries(reorderedC3SUnits.units);
	const c3sIndex = reorderedC3SEntries.findIndex(([unit]) => unit === "C3S");
	const c3AfterSIndex = reorderedC3SEntries.findIndex(([unit]) => unit === "C3");
	[reorderedC3SEntries[c3sIndex], reorderedC3SEntries[c3AfterSIndex]] = [reorderedC3SEntries[c3AfterSIndex], reorderedC3SEntries[c3sIndex]];
	reorderedC3SUnits.units = Object.fromEntries(reorderedC3SEntries);
	let reorderedC3SRejected = false;
	try { validateSpecification(reorderedC3SUnits); } catch { reorderedC3SRejected = true; }
	if (!reorderedC3SRejected) fail("C3S order self-test false negative");
	const reorderedC3RUnits = JSON.parse(JSON.stringify(specification));
	const reorderedC3REntries = Object.entries(reorderedC3RUnits.units);
	const c3rIndex = reorderedC3REntries.findIndex(([unit]) => unit === "C3R");
	const c3bAfterRIndex = reorderedC3REntries.findIndex(([unit]) => unit === "C3B");
	[reorderedC3REntries[c3rIndex], reorderedC3REntries[c3bAfterRIndex]] = [reorderedC3REntries[c3bAfterRIndex], reorderedC3REntries[c3rIndex]];
	reorderedC3RUnits.units = Object.fromEntries(reorderedC3REntries);
	let reorderedC3RRejected = false;
	try { validateSpecification(reorderedC3RUnits); } catch { reorderedC3RRejected = true; }
	if (!reorderedC3RRejected) fail("C3R order self-test false negative");
	const reorderedC3QUnits = JSON.parse(JSON.stringify(specification));
	const reorderedC3QEntries = Object.entries(reorderedC3QUnits.units);
	const c3qIndex = reorderedC3QEntries.findIndex(([unit]) => unit === "C3Q");
	const c3tAfterQIndex = reorderedC3QEntries.findIndex(([unit]) => unit === "C3T");
	[reorderedC3QEntries[c3qIndex], reorderedC3QEntries[c3tAfterQIndex]] = [reorderedC3QEntries[c3tAfterQIndex], reorderedC3QEntries[c3qIndex]];
	reorderedC3QUnits.units = Object.fromEntries(reorderedC3QEntries);
	let reorderedC3QRejected = false;
	try { validateSpecification(reorderedC3QUnits); } catch { reorderedC3QRejected = true; }
	if (!reorderedC3QRejected) fail("C3Q order self-test false negative");
	const reorderedC3TUnits = JSON.parse(JSON.stringify(specification));
	const reorderedC3TEntries = Object.entries(reorderedC3TUnits.units);
	const c3tIndex = reorderedC3TEntries.findIndex(([unit]) => unit === "C3T");
	const c3uAfterTIndex = reorderedC3TEntries.findIndex(([unit]) => unit === "C3U");
	[reorderedC3TEntries[c3tIndex], reorderedC3TEntries[c3uAfterTIndex]] = [reorderedC3TEntries[c3uAfterTIndex], reorderedC3TEntries[c3tIndex]];
	reorderedC3TUnits.units = Object.fromEntries(reorderedC3TEntries);
	let reorderedC3TRejected = false;
	try { validateSpecification(reorderedC3TUnits); } catch { reorderedC3TRejected = true; }
	if (!reorderedC3TRejected) fail("C3T order self-test false negative");
	const reorderedC3UUnits = JSON.parse(JSON.stringify(specification));
	const reorderedC3UEntries = Object.entries(reorderedC3UUnits.units);
	const c3uIndex = reorderedC3UEntries.findIndex(([unit]) => unit === "C3U");
	const c3bAfterUIndex = reorderedC3UEntries.findIndex(([unit]) => unit === "C3B");
	[reorderedC3UEntries[c3uIndex], reorderedC3UEntries[c3bAfterUIndex]] = [reorderedC3UEntries[c3bAfterUIndex], reorderedC3UEntries[c3uIndex]];
	reorderedC3UUnits.units = Object.fromEntries(reorderedC3UEntries);
	let reorderedC3URejected = false;
	try { validateSpecification(reorderedC3UUnits); } catch { reorderedC3URejected = true; }
	if (!reorderedC3URejected) fail("C3U order self-test false negative");
	for (const unit of ["C1B", "C2B", "C3PB", "C3B", "C6B"]) {
		const regular = specification.units[unit].exact.map((path, index) => ({
			mode: "100644", object: String(index + 1).padStart(40, "0"), stage: 0, path,
		}));
		validateReceiptIndexModes(specification, unit, specification.units[unit].exact, regular);
		for (const [name, mutate] of [
			["symlink", (entries) => { entries[0].mode = "120000"; }],
			["executable", (entries) => { entries[0].mode = "100755"; }],
			["unmerged", (entries) => { entries[0].stage = 2; }],
			["intent-to-add", (entries) => { entries[0].object = "0".repeat(40); }],
			["deleted", (entries) => { entries.shift(); }],
		]) {
			const hostile = regular.map((entry) => ({ ...entry }));
			mutate(hostile);
			let rejected = false;
			try { validateReceiptIndexModes(specification, unit, specification.units[unit].exact, hostile); } catch { rejected = true; }
			if (!rejected) fail(`${unit} receipt index-mode self-test false negative: ${name}`);
		}
	}
	const cleanCredentialEntries = [{ path: "fixture", text: "ordinary source text" }];
	if (credentialPatternFindings(cleanCredentialEntries).length !== 0) fail("credential scan clean self-test false positive");
	const hostileCredentials = [
		"-----BEGIN " + "PRIVATE KEY-----",
		"AK" + "IA" + "A".repeat(16),
		"gh" + "p_" + "a".repeat(30),
		"gl" + "pat-" + "a".repeat(20),
		"xo" + "xb-" + "a".repeat(20),
		"AI" + "za" + "a".repeat(35),
		"s" + "k-proj-" + "a".repeat(20),
		"s" + "k_live_" + "a".repeat(16),
		"Bearer " + "eyJ" + "a".repeat(10) + "." + "b".repeat(10) + "." + "c".repeat(10),
	];
	for (let index = 0; index < hostileCredentials.length; index += 1) {
		const findings = credentialPatternFindings([{ path: `fixture-${index}`, text: hostileCredentials[index] }]);
		if (findings.length !== 1 || findings[0].pattern !== credentialPatterns[index].name) {
			fail(`credential scan hostile self-test false negative: ${credentialPatterns[index].name}`);
		}
	}
	for (const invalid of ["../escape", "./alias", "/absolute", "double//slash", "control\npath"]) {
		let rejected = false;
		try {
			unexpectedPaths(specification, "C0A", [invalid]);
		} catch {
			rejected = true;
		}
		if (!rejected) fail(`unsafe path self-test false negative: ${JSON.stringify(invalid)}`);
	}
	console.log(`P07B-C unit scope self-test passed: strict specification plus deletion/rename, directory-boundary, allow/refuse, path-safety, ${gitDiagnosticCases} byte-safe Git result cases, ${authorityProfileRejections} transition/profile hostile cases, and ${c6aSourceAuthorityRejections} C6A source-authority hostile cases`);
}

async function main() {
	if (process.argv.length === 3 && process.argv[2] === "--self-test") {
		await runSelfTest();
		return;
	}
	if (process.argv.length === 3 && process.argv[2] === "--authority-profile-self-test") {
		await runAuthorityProfileSelfTest(await loadSpecification());
		return;
	}
	if (process.argv.length === 4 && process.argv[2] === "--candidate-phase") {
		const specification = await loadSpecification();
		await runIndependentCandidatePhaseGate(specification, process.argv[3]);
		return;
	}
	if (process.argv.length !== 5 || process.argv[2] !== "--unit" ||
		!(["--staged", "--exact-staged", "--receipt-manifest", "--maintenance-manifest", "--source-final-gate", "--receipt-final-gate", "--maintenance-final-gate", "--credential-scan", "--source-authority-gate"].includes(process.argv[4]))) {
		fail("usage: check-p07b-c-unit-scope.mjs --candidate-phase <C3D|C4V|C4M|C4N|C4P|C4|C4H|C4I|C4K|C4J|C4L|C5|C5V|C6A|C6F|C6G|C6M|C6B> | --unit <C0A|C0B|C1|C1M|C1V|C1E|C1B|C2|C2M|C2B|C3P|C3V|C3M|C3PB|C3A|C3L|C3F|C3S|C3|C3R|C3Q|C3T|C3U|C3B|C3D|C4V|C4M|C4N|C4P|C4|C4H|C4I|C4K|C4J|C4L|C5|C5V|C6A|C6F|C6G|C6M|C6B> <--staged|--exact-staged|--receipt-manifest|--maintenance-manifest|--source-final-gate|--receipt-final-gate|--maintenance-final-gate|--credential-scan|--source-authority-gate> | --authority-profile-self-test | --self-test");
	}
	const specification = await loadSpecification();
	if (process.argv[4] === "--source-authority-gate") {
		await runC6ASourceAuthorityGate(specification, process.argv[3]);
		return;
	}
	if (process.argv[4] === "--source-final-gate") {
		await runSourceFinalGate(specification, process.argv[3]);
		return;
	}
	if (process.argv[4] === "--receipt-final-gate") {
		await runReceiptFinalGate(specification, process.argv[3]);
		return;
	}
	if (process.argv[4] === "--maintenance-final-gate") {
		await runMaintenanceFinalGate(specification, process.argv[3]);
		return;
	}
	if (process.argv[4] === "--credential-scan") {
		await runCredentialScan(specification, process.argv[3]);
		return;
	}
	if (process.argv[4] === "--receipt-manifest") {
		const claims = receiptManifest(specification, process.argv[3]);
		console.log(`P07B-C ${process.argv[3]} receipt manifest exact: ${JSON.stringify(claims)}`);
		return;
	}
	if (process.argv[4] === "--maintenance-manifest") {
		const claims = maintenanceManifest(specification, process.argv[3]);
		console.log(`P07B-C ${process.argv[3]} non-product maintenance manifest exact: ${JSON.stringify(claims)}`);
		return;
	}
	const paths = await stagedPaths();
	if (paths.length === 0) fail("staged inventory is empty");
	if (specification.units[process.argv[3]]?.verification_profile === "RECEIPT_RECONCILIATION") {
		const entries = await stagedIndexEntries();
		const repeatedPaths = await stagedPaths();
		if (JSON.stringify(repeatedPaths) !== JSON.stringify(paths)) fail("staged inventory changed during receipt mode inspection");
		validateReceiptIndexModes(specification, process.argv[3], paths, entries);
	}
	if (process.argv[4] === "--exact-staged") {
		if (!exactPathsMatch(specification, process.argv[3], paths)) {
			fail(`${process.argv[3]} exact staged roster mismatch`);
		}
		const digest = createHash("sha256").update(`${paths.join("\n")}\n`, "utf8").digest("hex");
		console.log(`P07B-C ${process.argv[3]} exact staged scope passed: ${paths.length} path(s), sorted-newline sha256:${digest}`);
		return;
	}
	const unexpected = unexpectedPaths(specification, process.argv[3], paths);
	if (unexpected.length > 0) fail(`${process.argv[3]} unexpected staged paths: ${unexpected.join(", ")}`);
	console.log(`P07B-C ${process.argv[3]} staged scope passed: ${paths.length} admitted path(s)`);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) await main();
