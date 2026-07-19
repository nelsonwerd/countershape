#!/usr/bin/env node

import { lstat, readFile } from "node:fs/promises";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { isDeepStrictEqual } from "node:util";

import {
	checkPlan,
	repositoryRoot,
	runC3PReceiptSelfTest,
	verifyC3PLocalEvidence,
	verifyC3PBPresealLedger,
	verifyC3PPresealLedger,
} from "./check-p07b-c-plan.mjs";
import {
	credentialPatternFindings,
	exactPathsMatch,
	gitOutput,
	receiptManifest,
	stagedBlobEntries,
	stagedIndexEntries,
	stagedPaths,
	validateReceiptIndexModes,
	validateSpecification,
} from "./check-p07b-c-unit-scope.mjs";

const modulePath = fileURLToPath(import.meta.url);
const receiptPath = resolve(repositoryRoot, "spec/verification/p07b-c-c3p-receipt.json");

async function requireReceipt() {
	let status;
	try {
		status = await lstat(receiptPath);
	} catch (error) {
		if (error.code === "ENOENT") throw new Error("C3P receipt declaration must be present for this mode");
		throw error;
	}
	if (!status.isFile() || status.isSymbolicLink()) throw new Error("C3P receipt declaration must be a regular non-symlink file");
}

async function requirePlan() {
	const errors = await checkPlan();
	if (errors.length > 0) throw new Error(`P07B-C C3P receipt/plan check failed:\n${errors.join("\n")}`);
}

async function loadSpecification() {
	return validateSpecification(JSON.parse(await readFile(
		resolve(repositoryRoot, "spec/verification/p07b-c-unit-paths.json"),
		"utf8",
	)));
}

async function receiptStageSnapshot() {
	await requireReceipt();
	await requirePlan();
	const specification = await loadSpecification();
	const paths = await stagedPaths();
	if (!exactPathsMatch(specification, "C3PB", paths)) throw new Error("C3PB exact staged roster mismatch");
	const entries = await stagedIndexEntries();
	validateReceiptIndexModes(specification, "C3PB", paths, entries);
	const manifest = receiptManifest(specification, "C3PB");
	if (manifest.length !== 7) throw new Error("C3PB receipt claim manifest cardinality mismatch");
	const repeated = await stagedPaths();
	if (!isDeepStrictEqual(repeated, paths)) throw new Error("C3PB staged inventory changed during inspection");
	return { paths, specification };
}

async function verifyC3PBStaged() {
	const { paths } = await receiptStageSnapshot();
	await gitOutput(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--cached", "--check", "--"]);
	if ((await gitOutput(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--name-only", "-z", "--"])).length !== 0) {
		throw new Error("C3PB unstaged tracked changes are present");
	}
	if ((await gitOutput(["ls-files", "--others", "--exclude-standard", "-z", "--"])).length !== 0) {
		throw new Error("C3PB untracked paths are present");
	}
	if (!isDeepStrictEqual(await stagedPaths(), paths)) throw new Error("C3PB staged inventory changed during diff checks");
	console.log("P07B-C C3PB exact staged receipt gate passed: three mode-100644 paths, clean staged diff, no unstaged or untracked paths, exact seven-claim manifest");
}

async function verifyC3PBCredentialScan() {
	const { paths } = await receiptStageSnapshot();
	const findings = credentialPatternFindings(await stagedBlobEntries(paths));
	if (findings.length > 0) throw new Error(`C3PB structured credential-pattern findings: ${JSON.stringify(findings)}`);
	if (!isDeepStrictEqual(await stagedPaths(), paths)) throw new Error("C3PB staged inventory changed during credential scan");
	console.log(`P07B-C C3PB scoped staged structured credential-pattern scan: 0 findings across ${paths.length} exact paths and 9 named patterns`);
}

async function main() {
	const mode = process.argv[2];
	const usage = "usage: check-p07b-c-c3p-receipt.mjs [--self-test|--verify-local-evidence|--verify-preseal-ledger|--verify-c3pb-staged|--verify-c3pb-credential-scan|--verify-c3pb-preseal-ledger]";
	if (process.argv.length > (mode === undefined ? 2 : 3)) throw new Error(usage);
	switch (mode) {
		case undefined:
			await requirePlan();
			console.log("P07B-C C3P receipt check passed: phase-specific source/receipt, scope, Git-note, and documentation authority are coherent");
			return;
		case "--self-test":
			await runC3PReceiptSelfTest();
			return;
		case "--verify-local-evidence":
			await verifyC3PLocalEvidence();
			return;
		case "--verify-preseal-ledger":
			await verifyC3PPresealLedger();
			return;
		case "--verify-c3pb-staged":
			await verifyC3PBStaged();
			return;
		case "--verify-c3pb-credential-scan":
			await verifyC3PBCredentialScan();
			return;
		case "--verify-c3pb-preseal-ledger":
			await verifyC3PBPresealLedger();
			return;
		default:
			throw new Error(usage);
	}
}

if (process.argv[1] && resolve(process.argv[1]) === modulePath) {
	try {
		await main();
	} catch (error) {
		console.error(error.message);
		process.exitCode = 1;
	}
}
