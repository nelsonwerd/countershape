#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync, symlinkSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { collectFacts, futureSymbolsInSource, partitionFutureSymbols, validateFacts } from "./check-p07b-b-architecture.mjs";

const selftestPath = fileURLToPath(import.meta.url);
const root = resolve(dirname(selftestPath), "..");
const checker = resolve(root, "tools/check-p07b-b-architecture.mjs");
const cleanMarker = "P07B B architecture boundary OK\n";

const cases = Object.freeze([
	Object.freeze({ id: "issuer-owner", code: "P07B_B_ISSUER_OWNERSHIP" }),
	Object.freeze({ id: "raw-output-api", code: "P07B_B_RAW_OUTPUT_API" }),
	Object.freeze({ id: "transition-cardinality", code: "P07B_B_RAW_TRANSITION_CARDINALITY" }),
	Object.freeze({ id: "source-binding", code: "P07B_B_FRESH_REOPEN_PATHS" }),
	Object.freeze({ id: "parent-trust", code: "P07B_B_PARENT_TRUST_POLICY" }),
	Object.freeze({ id: "observation-split", code: "P07B_B_EXISTING_OBSERVATION_CLASSIFICATION" }),
	Object.freeze({ id: "rename-reconciliation", code: "P07B_B_RENAME_RECONCILIATION" }),
	Object.freeze({ id: "native-policy", code: "P07B_B_NATIVE_PUBLICATION_POLICY" }),
	Object.freeze({ id: "c1-semantic-boundary", code: "P07B_B_C1_SEMANTIC_BOUNDARY" }),
	Object.freeze({ id: "prefix-partition", code: "P07B_B_SELFTEST_PREFIX_PARTITION" }),
	Object.freeze({ id: "future-surface", code: "P07B_B_PREMATURE_C_SURFACE" }),
]);
const expectedRosterDigest = "252825c79567fd8751bc5ea48a380eeea2918f37156b4e8875c040ca6fabd7a2";

function rosterDigest() {
	const hash = createHash("sha256");
	for (const entry of cases) hash.update(entry.id).update("\0").update(entry.code).update("\0");
	return hash.digest("hex");
}

function fail(code, detail) {
	throw new Error(`${code}: ${detail}`);
}

function requireViolation(facts, code, id) {
	const violations = validateFacts(facts);
	if (!violations.some((violation) => violation.code === code)) {
		fail("P07B_B_SELFTEST_FALSE_NEGATIVE", `${id}:${violations.map((entry) => entry.code).join(",")}`);
	}
}

function runCleanChecker() {
	const result = spawnSync(process.execPath, [checker], {
		cwd: root, encoding: "utf8", timeout: 420_000, maxBuffer: 16 * 1024 * 1024, env: process.env,
	});
	if (result.error || result.signal || result.status !== 0 || result.stdout !== cleanMarker || result.stderr !== "") {
		fail("P07B_B_SELFTEST_CLEAN_CHECKER", `${result.status ?? result.signal}: ${result.stderr || result.stdout}`);
	}
	const directory = mkdtempSync(join(tmpdir(), "countershape-b-checker-"));
	try {
		const alias = join(directory, "architecture-alias.mjs");
		symlinkSync(checker, alias);
		const aliased = spawnSync(process.execPath, [alias], {
			cwd: root, encoding: "utf8", timeout: 420_000, maxBuffer: 16 * 1024 * 1024, env: process.env,
		});
		if (aliased.error || aliased.signal || aliased.status !== 0 ||
			aliased.stdout !== cleanMarker || aliased.stderr !== "") {
			fail("P07B_B_SELFTEST_SYMLINK_ENTRYPOINT", `${aliased.status ?? aliased.signal}: ${aliased.stderr || aliased.stdout}`);
		}
	} finally {
		rmSync(directory, { recursive: true, force: true });
	}
}

function requireExactC1PrefixPartition() {
	const admitted = "internal/contractexec/model/execution.go:ContractExecution";
	const c2Storage = [
		"internal/store/nonhead_contract.go:ContractExecutionTarget",
		"internal/store/nonhead_contract.go:FinalizedContractRun",
		"internal/store/nonhead_contract.go:ContractExecution",
	];
	const c2Support = "internal/store/nonhead_contract.go:ContractExecutionClassifierProfile";
	const c3Official = "internal/contractexec/target.go:ContractExecutionTarget";
	const modelEvil = "internal/contractexec/model_evil/execution.go:ContractExecution";
	const c2Lookalikes = [
		"internal/store/nonhead_contract_copy.go:ContractExecutionTarget",
		"internal/store/nonhead_contract.go:ContractExecutionTargetCopy",
		"internal/store/object_store.go:FinalizedContractRun",
	];
	const c3Lookalikes = [
		"internal/contractexec/target_copy.go:ContractExecutionTarget",
		"internal/contractexec/target.go:ContractExecutionTargetCopy",
		"internal/contractexec/target.go:CopyContractExecutionTarget",
		"internal/contractexec/target.go:ContractExecutionTargetβ",
		"internal/contractexec/runner/target.go:ContractExecutionTarget",
		"internal/contractexecx/target.go:ContractExecutionTarget",
	];
	const semanticFamilyLookalikes = [
		"internal/contractexec/target.go:FinalizedContractRunCopy",
		"internal/contractexec/target.go:CopyFinalizedContractRun",
		"internal/contractexec/target.go:FinalizedContractRunβ",
		"internal/contractexec/target.go:ContractExecutionCopy",
		"internal/contractexec/target.go:CopyContractExecution",
		"internal/contractexec/target.go:ContractExecutionβ",
		"internal/store/nonhead_contract.go:ContractExecutionClassifierProfileCopy",
		"internal/future.go:ContractExecutionClassifierProfile",
	];
	const foreign = "internal/future.go:ContractExecutionTarget";
	const partition = partitionFutureSymbols([admitted, ...c2Storage, c2Support, c3Official, modelEvil, ...c2Lookalikes, ...c3Lookalikes, ...semanticFamilyLookalikes, foreign]);
	if (JSON.stringify(partition.admitted) !== JSON.stringify([admitted, ...c2Storage, c2Support, c3Official]) ||
		JSON.stringify(partition.foreign) !== JSON.stringify([modelEvil, ...c2Lookalikes, ...c3Lookalikes, ...semanticFamilyLookalikes, foreign])) {
		fail("P07B_B_SELFTEST_PREFIX_PARTITION", JSON.stringify(partition));
	}
	const scannerControls = [
		["", []],
		["type ContractExecutionTarget struct{}", [c3Official]],
		["type ContractExecutionTargetCopy struct{}", ["internal/contractexec/target.go:ContractExecutionTargetCopy"]],
		["type CopyContractExecutionTarget struct{}", ["internal/contractexec/target.go:CopyContractExecutionTarget"]],
		["type ContractExecutionTargetβ struct{}", ["internal/contractexec/target.go:ContractExecutionTargetβ"]],
		["type FinalizedContractRunCopy struct{}", ["internal/contractexec/target.go:FinalizedContractRunCopy"]],
		["type CopyFinalizedContractRun struct{}", ["internal/contractexec/target.go:CopyFinalizedContractRun"]],
		["type FinalizedContractRunβ struct{}", ["internal/contractexec/target.go:FinalizedContractRunβ"]],
		["type ContractExecutionCopy struct{}", ["internal/contractexec/target.go:ContractExecutionCopy"]],
		["type CopyContractExecution struct{}", ["internal/contractexec/target.go:CopyContractExecution"]],
		["type ContractExecutionβ struct{}", ["internal/contractexec/target.go:ContractExecutionβ"]],
		["// CopyFinalizedContractRun", ["internal/contractexec/target.go:CopyFinalizedContractRun"]],
		["const value = \"ContractExecutionCopy\"", ["internal/contractexec/target.go:ContractExecutionCopy"]],
	];
	for (const [source, expected] of scannerControls) {
		const observed = futureSymbolsInSource("internal/contractexec/target.go", source);
		if (JSON.stringify(observed) !== JSON.stringify(expected)) {
			fail("P07B_B_SELFTEST_SYMBOL_SCANNER", JSON.stringify({ source, observed, expected }));
		}
	}
	const supportObserved = futureSymbolsInSource("internal/store/nonhead_contract.go", "const ContractExecutionClassifierProfile = 1");
	const supportPartition = partitionFutureSymbols(supportObserved);
	if (JSON.stringify(supportObserved) !== JSON.stringify([c2Support]) ||
		JSON.stringify(supportPartition.admitted) !== JSON.stringify([c2Support]) || supportPartition.foreign.length !== 0) {
		fail("P07B_B_SELFTEST_SUPPORT_SYMBOL", JSON.stringify({ supportObserved, supportPartition }));
	}
	for (const hostilePath of [
		"internal/contractexec/target_copy.go",
		"internal/contractexec/runner/target.go",
		"internal/contractexecx/target.go",
	]) {
		const observed = futureSymbolsInSource(hostilePath, "type ContractExecutionTarget struct{}");
		const hostile = partitionFutureSymbols(observed);
		if (hostile.admitted.length !== 0 || JSON.stringify(hostile.foreign) !== JSON.stringify(observed)) {
			fail("P07B_B_SELFTEST_SYMBOL_PATH", JSON.stringify({ hostilePath, hostile }));
		}
	}
}

async function main() {
	if (process.argv.length !== 2) fail("P07B_B_SELFTEST_ARGUMENTS", "no arguments accepted");
	const digest = rosterDigest();
	if (digest !== expectedRosterDigest) fail("P07B_B_SELFTEST_ROSTER_DRIFT", `${digest} != ${expectedRosterDigest}`);
	runCleanChecker();
	const clean = await collectFacts();
	const cleanViolations = validateFacts(clean);
	if (cleanViolations.length !== 0) {
		fail("P07B_B_SELFTEST_CLEAN_FACTS", cleanViolations.map((entry) => entry.code).join(","));
	}

	for (const test of cases) {
		if (test.id === "prefix-partition") {
			requireExactC1PrefixPartition();
			continue;
		}
		const facts = structuredClone(clean);
		switch (test.id) {
		case "issuer-owner":
			facts.issuerImporters.push("internal/foreign/issuer.go");
			break;
		case "raw-output-api":
			facts.rawOutputAPI = true;
			break;
		case "transition-cardinality":
			facts.advanceResidueRawCalls = 2;
			break;
		case "source-binding":
			facts.sourceBindingShape = false;
			facts.sourceFactoryReopenCount = 0;
			break;
		case "parent-trust":
			facts.parentTrustPolicy = false;
			break;
		case "observation-split":
			facts.existingObservationShape = false;
			break;
		case "rename-reconciliation":
			facts.renameReconciliationShape = false;
			break;
		case "native-policy":
			facts.nativeExclusiveFlags = false;
			facts.nativeFallbackPresent = true;
			break;
		case "c1-semantic-boundary":
			facts.c1BoundaryPassed = false;
			break;
		case "future-surface":
			facts.foreignFutureSymbols.push("internal/future.go:ContractExecutionTarget");
			break;
		default:
			fail("P07B_B_SELFTEST_UNKNOWN_CASE", test.id);
		}
		requireViolation(facts, test.code, test.id);
	}
	process.stdout.write(`P07B B architecture defensive self-test OK (${cases.length} metadata cases; exact C1 prefix, C2 storage symbols, C3 issuer symbol, and symlink entrypoint)\n`);
}

main().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
