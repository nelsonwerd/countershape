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
const unitOrder = Object.freeze(["C0A", "C0B", "C1", "C1M", "C1V", "C1E", "C1B", "C2", "C2M", "C2B", "C3P", "C3V", "C3M", "C3PB", "C3A", "C3L", "C3F", "C3S", "C3", "C3R", "C3Q", "C3T", "C3U", "C3B", "C3D", "C4V", "C4M", "C4", "C5", "C6A", "C6M", "C6B"]);
const receiptPhaseUnitOrder = Object.freeze(unitOrder.slice(unitOrder.indexOf("C3M")));
const receiptPhaseUnitSet = new Set(receiptPhaseUnitOrder);
const receiptPhaseKeys = Object.freeze(["C3P", "C3", "C6A"]);
const receiptPhaseStates = new Set(["ABSENT", "PRESENT"]);
const receiptPhaseAuthoritySHA256 = "dd134e870c11ed4e80eb8335680b59a0ed1553c45d04746fd2fdd722f3f975a4";
const receiptPhaseCapsuleStart = "<!-- P07B-C-RECEIPT-PHASE:START -->";
const receiptPhaseCapsuleEnd = "<!-- P07B-C-RECEIPT-PHASE:END -->";
const receiptPhaseHandoffPath = "docs/HANDOFF_MODE_C.md";
const receiptPhaseSpecificationPath = "spec/verification/p07b-c-unit-paths.json";
const c6aSourceAuthorityPath = "spec/verification/p07b-c-c6a-source-authority.json";
const c6aReceiptDeclarationPath = "spec/verification/p07b-c-c6a-receipt.json";
const c6EvidencePath = "docs/status/P07B-C-C6-EVIDENCE.md";
const c6aReceiptBlockStart = "<!-- P07B-C-C6A-SOURCE-RECEIPTS:START -->";
const c6aReceiptBlockEnd = "<!-- P07B-C-C6A-SOURCE-RECEIPTS:END -->";
const forwardCandidateReceiptPhaseBoundaries = Object.freeze(["C3D", "C4V", "C4M", "C4", "C5", "C6A", "C6M", "C6B"]);
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
const verificationProfiles = new Set(["SOURCE_FULL", "RECEIPT_RECONCILIATION"]);
const receiptClaimTypes = new Set(["tests-pass", "command-succeeded"]);
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
	Object.freeze({ label: "P07B-C C6A declared C5 parent edge and preceding didrun chain integrity", type: "command-succeeded" }),
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
	Object.freeze({ label: "P07B-C C6M sealed-C6A predecessor and preceding didrun chain integrity", type: "command-succeeded" }),
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
	if (specification.schema_version !== "countershape/p07b-c-unit-paths/v18") fail("specification version");
	if (!specification.units || typeof specification.units !== "object" || Array.isArray(specification.units) ||
		JSON.stringify(Object.keys(specification.units)) !== JSON.stringify(unitOrder)) fail("unit roster/order");

	for (const unit of unitOrder) {
		const entry = specification.units[unit];
		if (!entry || typeof entry !== "object" || Array.isArray(entry)) fail(`${unit}: field roster`);
		if (!verificationProfiles.has(entry.verification_profile)) fail(`${unit}: verification profile`);
		const baseExpectedFields = entry.verification_profile === "RECEIPT_RECONCILIATION"
			? ["exact", "prefixes", "receipt_claims", "verification_profile"]
			: ["exact", "prefixes", "verification_profile"];
		const expectedFields = receiptPhaseUnitSet.has(unit)
			? [...baseExpectedFields, "parent", "receipt_states"].sort()
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
		if (unit === "C4" && (entry.verification_profile !== c4DeclaredSourceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c4DeclaredSourceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c4DeclaredSourceContract.exact) ||
			exactRosterDigest(entry.exact) !== c4DeclaredSourceContract.exact_roster_sha256 ||
			exactRosterDigest(entry.prefixes) !== c4DeclaredSourceContract.prefix_roster_sha256)) {
			fail("C4: declared source contract");
		}
		if (unit === "C5" && (entry.verification_profile !== c5DeclaredSourceContract.verification_profile ||
			JSON.stringify(entry.prefixes) !== JSON.stringify(c5DeclaredSourceContract.prefixes) ||
			JSON.stringify(entry.exact) !== JSON.stringify(c5DeclaredSourceContract.exact) ||
			exactRosterDigest(entry.exact) !== c5DeclaredSourceContract.exact_roster_sha256 ||
			exactRosterDigest(entry.prefixes) !== c5DeclaredSourceContract.prefix_roster_sha256)) {
			fail("C5: declared source contract");
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
	if (boundary === "C6A") {
		if (manifestBytes !== undefined) fail("C6A: premature C6A source-authority manifest");
		if (c6aReceiptProjectionState(handoffBody, {}, "C6A HANDOFF") !== "ABSENT" ||
			c6aReceiptProjectionState(evidenceBody, {}, "C6A evidence") !== "ABSENT") {
			fail("C6A: premature source-receipt projection");
		}
		return undefined;
	}
	if (boundary !== "C6M" && boundary !== "C6B") fail(`future C6 material boundary: ${boundary}`);
	if (!Buffer.isBuffer(manifestBytes) || authority === undefined) fail(`${boundary}: C6A source-authority manifest absent`);
	const manifest = validateC6ASourceAuthorityManifest(manifestBytes, authority);
	if (boundary === "C6M") {
		if (parentManifestBytes !== undefined) fail("C6M: source-authority manifest pre-existed in C6A");
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
	if (boundary !== "C6A" && boundary !== "C6M" && boundary !== "C6B") return;
	const candidateHandoff = decodeCandidateUTF8(snapshot.stagedHandoffBytes, `${boundary} candidate HANDOFF`);
	const evidence = await treePathBytes(snapshot.indexTree, c6EvidencePath, `${boundary} C6 evidence`);
	const workingEvidence = await readCandidateWorktreeBytes(c6EvidencePath);
	if (!evidence.bytes.equals(workingEvidence)) fail(`${boundary}: C6 evidence staged/worktree mismatch`);
	if (boundary === "C6A") {
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
		if (await treeHasPath(snapshot.parentTree, c6aSourceAuthorityPath)) fail("C6M: source-authority manifest pre-existed in C6A");
		c6aAuthority = await deriveSealedSourceAuthority(
			specification, snapshot.parentCommit, "C6A", c6aSourceSubject, c6aSourceClaimManifest,
			c6aSourceExpectedClaimArgv, c6aSourceHermeticArgvPrefix,
		);
	} else {
		const c6mAuthority = await deriveSealedSourceAuthority(
			specification, snapshot.parentCommit, "C6M", c6mSourceSubject, c6mSourceClaimManifest,
			c6mSourceExpectedClaimArgv, c6mSourceHermeticArgvPrefix,
		);
		c6aAuthority = await deriveSealedSourceAuthority(
			specification, c6mAuthority.parent, "C6A", c6aSourceSubject, c6aSourceClaimManifest,
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

function validateIndependentCandidateSnapshot(specification, snapshot) {
	const boundary = snapshot.candidateBoundary;
	if (!forwardCandidateReceiptPhaseBoundarySet.has(boundary)) fail(`candidate boundary outside forward horizon: ${boundary}`);
	const unit = specification.units[boundary];
	const specificationOwners = new Set(["C3D", "C4V", "C4M"]);
	if (!unit.exact.includes(receiptPhaseHandoffPath) ||
		(specificationOwners.has(boundary) !== unit.exact.includes(receiptPhaseSpecificationPath))) {
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
	if (!specificationOwners.has(boundary) && !snapshot.stagedSpecificationBytes.equals(snapshot.parentSpecificationBytes)) {
		fail(`${boundary}: candidate changed sealed topology table`);
	}
	if (boundary === "C4V" && snapshot.stagedSpecificationBytes.equals(snapshot.parentSpecificationBytes)) {
		fail("C4V: maintenance candidate did not replace the sealed v16 topology table");
	}
	if (boundary === "C4M" && snapshot.stagedSpecificationBytes.equals(snapshot.parentSpecificationBytes)) {
		fail("C4M: maintenance candidate did not replace the sealed v17 topology table");
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
	if (boundary === "C4V" && (snapshot.parentCommit !== sealedC3DCandidateParent.commit ||
		snapshot.parentTree !== sealedC3DCandidateParent.tree || parentBoundary !== "C3D")) {
		fail("C4V: candidate requires the exact sealed C3D parent authority");
	}
	if (boundary === "C4M" && (snapshot.parentCommit !== sealedC4VCandidateParent.commit ||
		snapshot.parentTree !== sealedC4VCandidateParent.tree || parentBoundary !== "C4V")) {
		fail("C4M: candidate requires the exact sealed C4V parent authority");
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

export async function gitOutput(args) {
	const requested = process.env.COUNTERSHAPE_GIT || "/usr/bin/git";
	if (!isAbsolute(requested)) fail("COUNTERSHAPE_GIT must be absolute");
	let git;
	try {
		git = await realpath(requested);
	} catch (error) {
		fail(`git admission failed: ${error.message}`);
	}
	const result = spawnSync(git, ["--no-replace-objects", ...args], {
		cwd: repositoryRoot,
		encoding: "buffer",
		timeout: 30_000,
		maxBuffer: 20 * 1024 * 1024,
		env: {
			HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1",
			GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
			GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
		},
	});
	if (result.error || result.status !== 0 || result.signal || (result.stderr?.length ?? 0) !== 0) {
		fail(`git inventory failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
	}
	return result.stdout ?? Buffer.alloc(0);
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

async function runCredentialScan(specification, unit) {
	const profile = specification.units[unit]?.verification_profile;
	let paths;
	if (profile === "SOURCE_FULL") paths = await requireAdmittedStagedSource(specification, unit);
	else if (profile === "RECEIPT_RECONCILIATION") {
		paths = await stagedPaths();
		if (!exactPathsMatch(specification, unit, paths)) fail(`${unit}: credential receipt roster mismatch`);
		validateReceiptIndexModes(specification, unit, paths, await stagedIndexEntries());
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
		const bootstrap = parentBoundary === undefined;
		const c4vRepair = candidateBoundary === "C4V" && parentBoundary === "C3D";
		const c4mRepair = candidateBoundary === "C4M" && parentBoundary === "C4V";
		const handoff = document(candidateBoundary);
		return {
			candidateBoundary,
			parentCommit: bootstrap ? sealedC3BCandidateParent.commit : c4vRepair ? sealedC3DCandidateParent.commit : c4mRepair ? sealedC4VCandidateParent.commit : "f".repeat(40),
			parentTree: bootstrap ? sealedC3BCandidateParent.tree : c4vRepair ? sealedC3DCandidateParent.tree : c4mRepair ? sealedC4VCandidateParent.tree : "e".repeat(40),
			indexTree: "3".repeat(40),
			indexEntries: [
				{ mode: "100644", object: "1".repeat(40), stage: 0, path: receiptPhaseHandoffPath },
				{ mode: "100644", object: "2".repeat(40), stage: 0, path: receiptPhaseSpecificationPath },
			],
			stagedHandoffOID: "1".repeat(40),
			stagedSpecificationOID: "2".repeat(40),
			stagedHandoffBytes: handoff,
			workingHandoffBytes: handoff,
			parentHandoffBytes: Buffer.from(bootstrap ? "# Sealed C3B without capsule.\n" : document(parentBoundary)),
			stagedSpecificationBytes: specificationBytes,
			workingSpecificationBytes: specificationBytes,
			parentSpecificationBytes: bootstrap ? Buffer.from("{\"schema_version\":\"historical-v15\"}\n") :
				c4vRepair ? Buffer.from("{\"schema_version\":\"countershape/p07b-c-unit-paths/v16\"}\n") :
					c4mRepair ? Buffer.from("{\"schema_version\":\"countershape/p07b-c-unit-paths/v17\"}\n") : specificationBytes,
		};
	};
	const accepted = [["C3D", undefined], ["C4V", "C3D"], ["C4M", "C4V"], ["C4", "C4M"], ["C5", "C4"], ["C6A", "C5"], ["C6M", "C6A"], ["C6B", "C6M"]];
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
	refuse("CLI capsule mismatch", fixture("C4", "C4M"), (value) => {
		value.stagedHandoffBytes = value.workingHandoffBytes = document("C5");
	});
	refuse("HANDOFF worktree drift", fixture("C4", "C4M"), (value) => { value.workingHandoffBytes = Buffer.from("drift\n"); });
	refuse("specification worktree drift", fixture("C4", "C4M"), (value) => { value.workingSpecificationBytes = Buffer.from("drift\n"); });
	refuse("sealed specification drift", fixture("C4", "C4M"), (value) => { value.parentSpecificationBytes = Buffer.from("drift\n"); });
	refuse("C4V wrong sealed C3D parent", fixture("C4V", "C3D"), (value) => { value.parentCommit = "f".repeat(40); });
	refuse("C4V wrong sealed C3D tree", fixture("C4V", "C3D"), (value) => { value.parentTree = "e".repeat(40); });
	refuse("C4V unchanged v16 topology", fixture("C4V", "C3D"), (value) => { value.parentSpecificationBytes = value.stagedSpecificationBytes; });
	refuse("C4M wrong sealed C4V parent", fixture("C4M", "C4V"), (value) => { value.parentCommit = "f".repeat(40); });
	refuse("C4M wrong sealed C4V tree", fixture("C4M", "C4V"), (value) => { value.parentTree = "e".repeat(40); });
	refuse("C4M unchanged v17 topology", fixture("C4M", "C4V"), (value) => { value.parentSpecificationBytes = value.stagedSpecificationBytes; });
	refuse("bootstrap commit", fixture("C3D", undefined), (value) => { value.parentCommit = "0".repeat(40); });
	refuse("bootstrap tree", fixture("C3D", undefined), (value) => { value.parentTree = "0".repeat(40); });
	refuse("bootstrap prior capsule", fixture("C3D", undefined), (value) => { value.parentHandoffBytes = document("C3B"); });
	refuse("C4V skip", fixture("C4M", "C3D"), () => {});
	refuse("C4M skip", fixture("C4", "C4V"), () => {});
	refuse("source phase skip", fixture("C5", "C4M"), () => {});
	refuse("C6M skip", fixture("C6B", "C6A"), () => {});
	refuse("C6M replay", fixture("C6M", "C6M"), () => {});
	refuse("HANDOFF symlink", fixture("C4", "C4M"), (value) => { value.indexEntries[0].mode = "120000"; });
	refuse("specification executable", fixture("C4", "C4M"), (value) => { value.indexEntries[1].mode = "100755"; });
	refuse("HANDOFF index OID", fixture("C4", "C4M"), (value) => { value.indexEntries[0].object = "9".repeat(40); });
	refuse("specification index OID", fixture("C4", "C4M"), (value) => { value.indexEntries[1].object = "9".repeat(40); });
	refuse("HANDOFF non-UTF8", fixture("C4", "C4M"), (value) => {
		value.stagedHandoffBytes = value.workingHandoffBytes = Buffer.from([0xff]);
	});
	refuse("parent non-UTF8", fixture("C4", "C4M"), (value) => { value.parentHandoffBytes = Buffer.from([0xff]); });
	refuse("hidden capsule", fixture("C4", "C4M"), (value) => {
		const hidden = Buffer.from(`# Candidate fixture\n\n## Current state\n\n<!--\n${renderCandidateReceiptPhaseCapsule(rows.C4)}\n-->\n`, "utf8");
		value.stagedHandoffBytes = value.workingHandoffBytes = hidden;
	});
	refuse("specification semantic drift", fixture("C3D", undefined), (value) => {
		const hostile = structuredClone(specification);
		hostile.units.C4.exact = hostile.units.C4.exact.slice(1);
		value.stagedSpecificationBytes = value.workingSpecificationBytes = Buffer.from(`${JSON.stringify(hostile, null, 2)}\n`);
	});
	const stable = fixture("C6M", "C6A");
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
	if (accepted.length !== 8 || rejected !== 28) fail(`independent candidate self-test cardinality ${accepted.length}/${rejected}`);
}

function runFutureC6MaterialSelfTest(authority) {
	const manifestBytes = Buffer.from(renderC6ASourceAuthorityManifest(authority), "utf8");
	const manifest = validateC6ASourceAuthorityManifest(manifestBytes, authority);
	const block = renderC6ASourceReceiptBlock(manifest);
	const absent = "# Future C6 fixture\n\nC6A source receipts remain absent.\n";
	const present = `${absent.trimEnd()}\n\n${block}\n`;
	validateFutureC6Material({ boundary: "C6A", handoffBody: absent, evidenceBody: absent });
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
	const note = {
		claims: c6aSourceClaimManifest.map((claim, index) => ({
			claim: {
				argv_preview: [...c6aSourceExpectedClaimArgv[index]],
				ctype: claim.type, declared_at_index: index, event_indices: [index], label: claim.label, pathspecs: [],
			},
			delta: [], exit_code: 0, grade: "tree-exact",
			reason: "self-stable command ran against the sealed tree", supporting_event_index: index,
		})),
		commit: identity.commit,
		coverage: { by_coverage: { complete: c6aSourceClaimManifest.length }, total_events: c6aSourceClaimManifest.length },
		secrets_override: true,
		tree: identity.tree,
		version: 1,
	};
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
	const futureMaterialRejected = runFutureC6MaterialSelfTest(authority);
	if (futureMaterialRejected !== 13) fail(`future C6 material self-test cardinality ${futureMaterialRejected}`);
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

async function runSelfTest() {
	const specification = await loadSpecification();
	runIndependentCandidatePhaseSelfTest(specification);
	const c6aSourceAuthorityRejections = runC6ASourceAuthoritySelfTest();
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
		isDeepStrictEqual(specification.units.C4.exact, c4DeclaredSourceContract.exact),
		isDeepStrictEqual(specification.units.C4.prefixes, c4DeclaredSourceContract.prefixes),
		isDeepStrictEqual(specification.units.C5.exact, c5DeclaredSourceContract.exact),
		isDeepStrictEqual(specification.units.C5.prefixes, c5DeclaredSourceContract.prefixes),
		unexpectedPaths(specification, "C4", ["tools/verify-runtime-authority.mjs"]).length === 0,
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
		receiptPhaseRows(specification).length === 20,
		receiptPhaseRows(specification).at(-1).boundary === "C6B",
		receiptPhaseAuthorityDigest(specification) === receiptPhaseAuthoritySHA256,
		!["C4", "C5", "C6A", "C6M", "C6B"].some((boundary) =>
			specification.units[boundary].exact.includes("spec/verification/p07b-c-unit-paths.json")),
		specification.units.C4V.exact.includes("spec/verification/p07b-c-unit-paths.json"),
		specification.units.C4M.exact.includes("spec/verification/p07b-c-unit-paths.json"),
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
	for (const boundary of ["C4V", "C4M", "C4", "C5", "C6A", "C6M", "C6B"]) {
		if (activeReceiptPhaseBoundary(specification, boundary) !== boundary) {
			fail(`${boundary} visible-capsule active-boundary pointer self-test`);
		}
	}
	requireSpecificationMutationRejected(specification, "schema v17 downgrade", (hostile) => {
		hostile.schema_version = "countershape/p07b-c-unit-paths/v17";
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
	requireSpecificationMutationRejected(specification, "C5 profile drift", (hostile) => {
		hostile.units.C5.verification_profile = "RECEIPT_RECONCILIATION";
	});
	requireSpecificationMutationRejected(specification, "C5 parent drift", (hostile) => {
		hostile.units.C5.parent = "C4V";
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
	for (const boundary of ["C4V", "C4M", "C4", "C5", "C6A", "C6M"]) {
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
	console.log(`P07B-C unit scope self-test passed: strict specification plus deletion/rename, directory-boundary, allow/refuse, path-safety, and ${c6aSourceAuthorityRejections} C6A source-authority hostile cases`);
}

async function main() {
	if (process.argv.length === 3 && process.argv[2] === "--self-test") {
		await runSelfTest();
		return;
	}
	if (process.argv.length === 4 && process.argv[2] === "--candidate-phase") {
		const specification = await loadSpecification();
		await runIndependentCandidatePhaseGate(specification, process.argv[3]);
		return;
	}
	if (process.argv.length !== 5 || process.argv[2] !== "--unit" ||
		!(["--staged", "--exact-staged", "--receipt-manifest", "--source-final-gate", "--receipt-final-gate", "--credential-scan", "--source-authority-gate"].includes(process.argv[4]))) {
		fail("usage: check-p07b-c-unit-scope.mjs --candidate-phase <C3D|C4V|C4M|C4|C5|C6A|C6M|C6B> | --unit <C0A|C0B|C1|C1M|C1V|C1E|C1B|C2|C2M|C2B|C3P|C3V|C3M|C3PB|C3A|C3L|C3F|C3S|C3|C3R|C3Q|C3T|C3U|C3B|C3D|C4V|C4M|C4|C5|C6A|C6M|C6B> <--staged|--exact-staged|--receipt-manifest|--source-final-gate|--receipt-final-gate|--credential-scan|--source-authority-gate> | --self-test");
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
	if (process.argv[4] === "--credential-scan") {
		await runCredentialScan(specification, process.argv[3]);
		return;
	}
	if (process.argv[4] === "--receipt-manifest") {
		const claims = receiptManifest(specification, process.argv[3]);
		console.log(`P07B-C ${process.argv[3]} receipt manifest exact: ${JSON.stringify(claims)}`);
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
