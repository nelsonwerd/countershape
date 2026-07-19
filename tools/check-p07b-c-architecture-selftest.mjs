#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import {
	collectC2Facts,
	collectFacts,
	validateC2Facts,
	validateFacts,
	validateGoJSONTranscript,
} from "./check-p07b-c-architecture.mjs";

const selftestPath = fileURLToPath(import.meta.url);
const root = resolve(dirname(selftestPath), "..");
const checker = resolve(root, "tools/check-p07b-c-architecture.mjs");
const cases = Object.freeze([
	Object.freeze({ id: "production-files", code: "P07B_C1_PRODUCTION_TOPOLOGY" }),
	Object.freeze({ id: "test-files", code: "P07B_C1_TEST_TOPOLOGY" }),
	Object.freeze({ id: "test-file-rename", code: "P07B_C1_TEST_TOPOLOGY" }),
	Object.freeze({ id: "test-import", code: "P07B_C1_TEST_IMPORT_ROSTER" }),
	Object.freeze({ id: "test-symbol", code: "P07B_C1_TEST_SYMBOL_ROSTER" }),
	Object.freeze({ id: "model-extra-entry", code: "P07B_C1_MODEL_TOPOLOGY" }),
	Object.freeze({ id: "production-import", code: "P07B_C1_PRODUCTION_IMPORT_ROSTER" }),
	Object.freeze({ id: "dependency-closure", code: "P07B_C1_DEPENDENCY_CLOSURE" }),
	Object.freeze({ id: "ignored-production", code: "P07B_C1_PRODUCTION_TOPOLOGY" }),
	Object.freeze({ id: "foreign-build-input", code: "P07B_C1_PRODUCTION_TOPOLOGY" }),
	Object.freeze({ id: "premature-importer", code: "P07B_C1_PREMATURE_IMPORTER" }),
	Object.freeze({ id: "schema-closure", code: "P07B_C1_SCHEMA_CLOSURE" }),
	Object.freeze({ id: "schema-required-roster", code: "P07B_C1_SCHEMA_REQUIRED_ROSTER" }),
	Object.freeze({ id: "target-root", code: "P07B_C1_OBJECT_ROSTER" }),
	Object.freeze({ id: "serialized-disposition", code: "P07B_C1_OBJECT_ROSTER" }),
	Object.freeze({ id: "result-roster", code: "P07B_C1_ENUM_ROSTER" }),
	Object.freeze({ id: "scope-domain", code: "P07B_C1_SCOPE_PROFILE" }),
	Object.freeze({ id: "limit-profile", code: "P07B_C1_LIMIT_PROFILE" }),
	Object.freeze({ id: "evidence-roster", code: "P07B_C1_EXAMPLE_REFERENCE_ROSTER" }),
	Object.freeze({ id: "example-graph", code: "P07B_C1_EXAMPLE_GRAPH" }),
	Object.freeze({ id: "c0-authority", code: "P07B_C1_C0_AUTHORITY" }),
	Object.freeze({ id: "topology", code: "P07B_C1_PRODUCTION_TOPOLOGY" }),
]);
const expectedRosterDigest = "70ceb6322134b002aeff648c01267d4a95b98eb00bbe0a8c7f1ea5f4ff19c044";
const c2Cases = Object.freeze([
	Object.freeze({ id: "store-production-files", code: "P07B_C2_STORE_TOPOLOGY" }),
	Object.freeze({ id: "store-directory-entry", code: "P07B_C2_STORE_TOPOLOGY" }),
	Object.freeze({ id: "production-import", code: "P07B_C2_IMPORT_ROSTER" }),
	Object.freeze({ id: "compiler-production-import", code: "P07B_C2_IMPORT_ROSTER" }),
	Object.freeze({ id: "new-production-export", code: "P07B_C2_EXPORTED_SURFACE" }),
	Object.freeze({ id: "object-store-export", code: "P07B_C2_EXPORTED_SURFACE" }),
	Object.freeze({ id: "compiler-parsed-surface", code: "P07B_C2_EXPORTED_SURFACE" }),
	Object.freeze({ id: "authority-shape", code: "P07B_C2_AUTHORITY_SHAPE" }),
	Object.freeze({ id: "test-symbol", code: "P07B_C2_TEST_SYMBOL_ROSTER" }),
	Object.freeze({ id: "test-file-roster", code: "P07B_C2_TEST_FILE_ROSTER" }),
	Object.freeze({ id: "forbidden-surface", code: "P07B_C2_FORBIDDEN_PRODUCTION_SURFACE" }),
	Object.freeze({ id: "namespace-identity", code: "P07B_C2_NAMESPACE_IDENTITY" }),
	Object.freeze({ id: "namespace-case-alias", code: "P07B_C2_NAMESPACE_IDENTITY" }),
	Object.freeze({ id: "relation-constants", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "relation-effects", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "relation-persistence-order", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "relation-open-convergence", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "run-manifest-gate", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "execution-profile-derived", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "relation-identity-guards", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "relation-identity-tests", code: "P07B_C2_RELATION_ALGEBRA" }),
	Object.freeze({ id: "boot-join", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "acquisition-order", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "release-order", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "reset-order", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "winner-freshness", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "start-claim-convergence", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "clear-receipt-transition", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "clear-receipt-convergence", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "clear-receipt-faults", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "interlock-identity-boundary", code: "P07B_C2_INTERLOCK_PROTOCOL" }),
	Object.freeze({ id: "seal-ownership", code: "P07B_C2_SEAL_OWNERSHIP" }),
	Object.freeze({ id: "private-limits", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "private-kind-roster", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "private-state-roster", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "manifest-closure", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "manifest-open-convergence", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "private-arithmetic", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "availability-convergence", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "purge-order", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "purge-create-count", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "fresh-purge-order", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "missing-pack-gate", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "canonical-isolation", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "private-identity-guards", code: "P07B_C2_PRIVATE_EVIDENCE" }),
	Object.freeze({ id: "private-identity-tests", code: "P07B_C2_PRIVATE_EVIDENCE" }),
]);
const expectedC2RosterDigest = "9cffa462cff50a197afe3733529a0eba6ebc6d29a5399ea33ee6d33019fd885f";

function rosterDigest(roster = cases) {
	const hash = createHash("sha256");
	for (const entry of roster) hash.update(entry.id).update("\0").update(entry.code).update("\0");
	return hash.digest("hex");
}

function fail(code, detail) { throw new Error(`${code}: ${detail}`); }

function requireViolation(facts, code, id) {
	const problems = validateFacts(facts);
	if (!problems.some((problem) => problem.code === code)) {
		fail("P07B_C1_SELFTEST_FALSE_NEGATIVE", `${id}:${problems.map((problem) => problem.code).join(",")}`);
	}
}

function requireC2Violation(facts, code, id) {
	const problems = validateC2Facts(facts);
	if (!problems.some((problem) => problem.code === code)) {
		fail("P07B_C2_SELFTEST_FALSE_NEGATIVE", `${id}:${problems.map((problem) => problem.code).join(",")}`);
	}
}

function runCleanChecker(phase = "c1") {
	const args = phase === "c2" ? [checker, "--c2"] : [checker];
	const marker = phase === "c2" ? "P07B-C C2 cumulative architecture boundary OK" : "P07B-C C1 architecture boundary OK";
	const result = spawnSync(process.execPath, args, {
		cwd: root,
		encoding: "utf8",
		timeout: phase === "c2" ? 420_000 : 180_000,
		maxBuffer: 32 * 1024 * 1024,
		env: {
			...process.env,
			COUNTERSHAPE_GO: process.env.COUNTERSHAPE_GO ?? "/opt/homebrew/bin/go",
			GOMAXPROCS: "2",
			GOFLAGS: "-mod=readonly -buildvcs=false -p=1",
		},
	});
	if (result.error || result.signal || result.status !== 0 || result.stderr !== "" || result.stdout.trim() !== marker) {
		fail(phase === "c2" ? "P07B_C2_SELFTEST_CLEAN_CHECKER" : "P07B_C1_SELFTEST_CLEAN_CHECKER",
			`${result.status ?? result.signal}: ${result.stderr || result.stdout || result.error}`);
	}
}

function inspectGoJSONTranscriptParser() {
	const packagePath = "github.com/nelsonwerd/countershape/internal/contractexec/model";
	const first = "TestClassifierTruthTableUsesOnlyEligibleCompleteTupleMembership";
	const second = "TestClosedRunAlgebraExhaustiveCrossProduct";
	const clean = [
		{ Action: "start", Package: packagePath },
		{ Action: "run", Package: packagePath, Test: first },
		{ Action: "pass", Package: packagePath, Test: first },
		{ Action: "run", Package: packagePath, Test: second },
		{ Action: "pass", Package: packagePath, Test: second },
		{ Action: "pass", Package: packagePath },
	];
	const encode = (events) => Buffer.from(`${events.map((event) => JSON.stringify(event)).join("\n")}\n`, "utf8");
	validateGoJSONTranscript("exhaustive-algebra", encode(clean));
	const hostile = [
		clean.filter((event) => !(event.Action === "pass" && event.Test === second)),
		clean.map((event) => event.Action === "pass" && event.Test === first ? { ...event, Action: "skip" } : event),
		[...clean.slice(0, -1),
			{ Action: "run", Package: packagePath, Test: "TestUnexpected" },
			{ Action: "pass", Package: packagePath, Test: "TestUnexpected" },
			clean.at(-1)],
		[...clean.slice(0, 3), clean[2], ...clean.slice(3)],
		clean.map((event) => event.Action === "pass" && !event.Test ? { ...event, Action: "fail" } : event),
		clean.map((event, index) => index === 0 ? { ...event, Package: "example.invalid/foreign" } : event),
	];
	for (const [index, events] of hostile.entries()) {
		let rejected = false;
		try {
			validateGoJSONTranscript("exhaustive-algebra", encode(events));
		} catch {
			rejected = true;
		}
		if (!rejected) fail("P07B_C1_SELFTEST_GO_JSON_FALSE_NEGATIVE", index + 1);
	}
	return hostile.length + 1;
}

function inspectC2GoJSONTranscriptParser() {
	const packagePath = "github.com/nelsonwerd/countershape/internal/store";
	const first = "TestC2StoreExportsNoOfficialIssuerOrRunPermit";
	const second = "TestC2StoreExportsNoListLatestTraversalStatusOrHeadMutationSurface";
	const clean = [
		{ Action: "start", Package: packagePath },
		{ Action: "run", Package: packagePath, Test: first },
		{ Action: "pass", Package: packagePath, Test: first },
		{ Action: "run", Package: packagePath, Test: second },
		{ Action: "pass", Package: packagePath, Test: second },
		{ Action: "pass", Package: packagePath },
	];
	const encode = (events) => Buffer.from(`${events.map((event) => JSON.stringify(event)).join("\n")}\n`, "utf8");
	validateGoJSONTranscript("c2-public-surface", encode(clean));
	const hostile = [
		clean.filter((event) => !(event.Action === "pass" && event.Test === second)),
		clean.map((event) => event.Action === "pass" && event.Test === first ? { ...event, Action: "skip" } : event),
		[...clean.slice(0, -1),
			{ Action: "run", Package: packagePath, Test: "TestC2Unexpected" },
			{ Action: "pass", Package: packagePath, Test: "TestC2Unexpected" },
			clean.at(-1)],
		[...clean.slice(0, 3), clean[2], ...clean.slice(3)],
		clean.map((event) => event.Action === "pass" && !event.Test ? { ...event, Action: "fail" } : event),
		clean.map((event, index) => index === 0 ? { ...event, Package: "example.invalid/foreign" } : event),
	];
	for (const [index, events] of hostile.entries()) {
		let rejected = false;
		try {
			validateGoJSONTranscript("c2-public-surface", encode(events));
		} catch {
			rejected = true;
		}
		if (!rejected) fail("P07B_C2_SELFTEST_GO_JSON_FALSE_NEGATIVE", index + 1);
	}
	return hostile.length + 1;
}

async function runC2Selftest() {
	const digest = rosterDigest(c2Cases);
	if (digest !== expectedC2RosterDigest) fail("P07B_C2_SELFTEST_ROSTER_DRIFT", `${digest} != ${expectedC2RosterDigest}`);
	runCleanChecker("c2");
	const clean = await collectC2Facts();
	const cleanProblems = validateC2Facts(clean);
	if (cleanProblems.length > 0) fail("P07B_C2_SELFTEST_CLEAN_FACTS", cleanProblems.map((problem) => problem.code).join(","));

	for (const test of c2Cases) {
		const facts = structuredClone(clean);
		switch (test.id) {
		case "store-production-files": facts.package.productionFiles.pop(); break;
		case "store-directory-entry": facts.directoryEntries.push("foreign.go:file"); facts.directoryEntries.sort(); break;
		case "production-import": facts.imports["execution_interlock.go"].push("net/http"); facts.imports["execution_interlock.go"].sort(); break;
		case "compiler-production-import": facts.package.productionImports.push("example.invalid/process-capability"); facts.package.productionImports.sort(); break;
		case "new-production-export": facts.newProductionExports["internal/store/nonhead_contract.go"].push("OfficialTarget"); break;
		case "object-store-export": facts.objectStoreExports.pop(); break;
		case "compiler-parsed-surface": facts.compilerParsedSurface = false; break;
		case "authority-shape": facts.structs.interlockLease.pop(); break;
		case "test-symbol": facts.testSymbols.pop(); break;
		case "test-file-roster": facts.testFiles["internal/store/execution_interlock_test.go"].pop(); break;
		case "forbidden-surface": facts.forbiddenSurface.push("process-start"); break;
		case "namespace-identity": facts.namespaces.retained = false; break;
		case "namespace-case-alias": facts.namespaces.caseAliasGuard = false; break;
		case "relation-constants": facts.relations.constants = false; break;
		case "relation-effects": facts.relations.effects.pop(); break;
		case "relation-persistence-order": facts.relations.persistenceOrder = false; break;
		case "relation-open-convergence": facts.relations.openConvergence = false; break;
		case "run-manifest-gate": facts.relations.runManifestGate = false; break;
		case "execution-profile-derived": facts.relations.executionProfileDerived = false; break;
		case "relation-identity-guards": facts.relations.identityGuards = false; break;
		case "relation-identity-tests": facts.relations.identityReplacementTests = false; break;
		case "boot-join": facts.interlock.bootJoin = false; break;
		case "acquisition-order": facts.interlock.acquisitionOrder = false; break;
		case "release-order": facts.interlock.releaseOrder = false; break;
		case "reset-order": facts.interlock.resetOrder = false; break;
		case "winner-freshness": facts.interlock.winnerFreshAndConsumed = false; break;
		case "start-claim-convergence": facts.interlock.startClaimConvergence = false; break;
		case "clear-receipt-transition": facts.interlock.clearReceiptTransition = false; break;
		case "clear-receipt-convergence": facts.interlock.clearReceiptConvergence = false; break;
		case "clear-receipt-faults": facts.interlock.clearReceiptFaults = false; break;
		case "interlock-identity-boundary": facts.interlock.identityBoundary = false; break;
		case "seal-ownership": facts.interlock.productionSeals.lease += 1; break;
		case "private-limits": facts.privateEvidence.limits = false; break;
		case "private-kind-roster": facts.privateEvidence.kinds.pop(); break;
		case "private-state-roster": facts.privateEvidence.states.pop(); break;
		case "manifest-closure": facts.privateEvidence.manifestClosure = false; break;
		case "manifest-open-convergence": facts.privateEvidence.manifestOpenConvergence = false; break;
		case "private-arithmetic": facts.privateEvidence.arithmeticBounded = false; break;
		case "availability-convergence": facts.privateEvidence.availabilityJoins = false; break;
		case "purge-order": facts.privateEvidence.purgeOrder = false; break;
		case "purge-create-count": facts.privateEvidence.purgeCreateCount = false; break;
		case "fresh-purge-order": facts.privateEvidence.freshPurgeOrder = false; break;
		case "missing-pack-gate": facts.privateEvidence.missingPackGate = false; break;
		case "canonical-isolation": facts.privateEvidence.noCanonicalMutation = false; break;
		case "private-identity-guards": facts.privateEvidence.identityGuards = false; break;
		case "private-identity-tests": facts.privateEvidence.identityReplacementTests = false; break;
		default: fail("P07B_C2_SELFTEST_UNKNOWN_CASE", test.id);
		}
		requireC2Violation(facts, test.code, test.id);
	}
	const goJSONCases = inspectC2GoJSONTranscriptParser();
	process.stdout.write(`P07B-C C2 cumulative architecture defensive self-test OK (${c2Cases.length} metadata cases; ${goJSONCases} Go JSON parser cases)\n`);
}

async function main() {
	if (process.argv[2] === "--c2") {
		if (process.argv.length !== 3) fail("P07B_C2_SELFTEST_ARGUMENTS", "--c2 accepts no other arguments");
		await runC2Selftest();
		return;
	}
	if (process.argv.length !== 2) fail("P07B_C1_SELFTEST_ARGUMENTS", "no arguments accepted");
	const digest = rosterDigest();
	if (digest !== expectedRosterDigest) fail("P07B_C1_SELFTEST_ROSTER_DRIFT", `${digest} != ${expectedRosterDigest}`);
	runCleanChecker();
	const clean = await collectFacts();
	const cleanProblems = validateFacts(clean);
	if (cleanProblems.length > 0) fail("P07B_C1_SELFTEST_CLEAN_FACTS", cleanProblems.map((problem) => problem.code).join(","));

	for (const test of cases) {
		const facts = structuredClone(clean);
		switch (test.id) {
		case "production-files":
			facts.package.productionFiles.pop();
			break;
		case "test-files":
			facts.package.testFiles.pop();
			break;
		case "test-file-rename":
			facts.package.testFiles[0] = "algebra_renamed_test.go";
			facts.package.testFiles.sort();
			break;
		case "test-import":
			facts.package.testImports.push("net/http");
			facts.package.testImports.sort();
			break;
		case "test-symbol":
			facts.package.testSymbols.pop();
			break;
		case "model-extra-entry":
			facts.topology.modelEntries.push("extra_test.go:file");
			facts.topology.modelEntries.sort();
			break;
		case "production-import":
			facts.package.productionImports.push("os");
			facts.package.productionImports.sort();
			break;
		case "dependency-closure":
			facts.localDependencies.push("github.com/nelsonwerd/countershape/internal/store");
			facts.localDependencies.sort();
			break;
		case "ignored-production":
			facts.package.ignoredGoFiles.push("runtime_other.go");
			break;
		case "foreign-build-input":
			facts.package.nonGoBuildFiles.push("bridge.c");
			break;
		case "premature-importer":
			facts.productionImporters.push("github.com/nelsonwerd/countershape/cmd/countershape");
			break;
		case "schema-closure":
			facts.schema.unclosedObjects.push("run/$defs/ClosedRunWitness");
			break;
		case "schema-required-roster":
			facts.schema.incompleteRequiredObjects.push("execution/");
			break;
		case "target-root":
			facts.schema.targetRoot.push("result");
			break;
		case "serialized-disposition":
			facts.schema.forbiddenMembers.push("run/properties/terminal_disposition");
			break;
		case "result-roster":
			facts.schema.results.pop();
			break;
		case "scope-domain":
			facts.schema.scopeDomains.pop();
			break;
		case "limit-profile":
			facts.schema.limits.privateByteMax += 1;
			break;
		case "evidence-roster":
			facts.example.witnessReferenceKinds.pop();
			facts.example.witnessReferenceCount -= 1;
			break;
		case "example-graph":
			facts.example.graph.runToExecution = false;
			break;
		case "c0-authority":
			facts.c0.objects.pop();
			break;
		case "topology":
			facts.topology.contractexecEntries.push("runner:directory");
			facts.topology.contractexecEntries.sort();
			break;
		default:
			fail("P07B_C1_SELFTEST_UNKNOWN_CASE", test.id);
		}
		requireViolation(facts, test.code, test.id);
	}
	const goJSONCases = inspectGoJSONTranscriptParser();
	process.stdout.write(`P07B-C C1 architecture defensive self-test OK (${cases.length} metadata cases; ${goJSONCases} Go JSON parser cases)\n`);
}

main().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
