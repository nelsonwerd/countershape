#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { collectFacts, partitionFutureSymbols, validateFacts } from "./check-p07b-b-architecture.mjs";

const selftestPath = fileURLToPath(import.meta.url);
const root = resolve(dirname(selftestPath), "..");
const checker = resolve(root, "tools/check-p07b-b-architecture.mjs");

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
	if (result.error || result.signal || result.status !== 0 || !result.stdout.includes("P07B B architecture boundary OK")) {
		fail("P07B_B_SELFTEST_CLEAN_CHECKER", `${result.status ?? result.signal}: ${result.stderr || result.stdout}`);
	}
}

function requireExactC1PrefixPartition() {
	const admitted = "internal/contractexec/model/execution.go:ContractExecution";
	const modelEvil = "internal/contractexec/model_evil/execution.go:ContractExecution";
	const foreign = "internal/future.go:ContractExecutionTarget";
	const partition = partitionFutureSymbols([admitted, modelEvil, foreign]);
	if (JSON.stringify(partition.admitted) !== JSON.stringify([admitted]) ||
		JSON.stringify(partition.foreign) !== JSON.stringify([modelEvil, foreign])) {
		fail("P07B_B_SELFTEST_PREFIX_PARTITION", JSON.stringify(partition));
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
	process.stdout.write(`P07B B architecture defensive self-test OK (${cases.length} metadata cases; exact C1 prefix partition)\n`);
}

main().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
