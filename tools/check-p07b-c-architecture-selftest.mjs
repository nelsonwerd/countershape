#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { collectFacts, validateFacts, validateGoJSONTranscript } from "./check-p07b-c-architecture.mjs";

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

function rosterDigest() {
	const hash = createHash("sha256");
	for (const entry of cases) hash.update(entry.id).update("\0").update(entry.code).update("\0");
	return hash.digest("hex");
}

function fail(code, detail) { throw new Error(`${code}: ${detail}`); }

function requireViolation(facts, code, id) {
	const problems = validateFacts(facts);
	if (!problems.some((problem) => problem.code === code)) {
		fail("P07B_C1_SELFTEST_FALSE_NEGATIVE", `${id}:${problems.map((problem) => problem.code).join(",")}`);
	}
}

function runCleanChecker() {
	const result = spawnSync(process.execPath, [checker], {
		cwd: root,
		encoding: "utf8",
		timeout: 180_000,
		maxBuffer: 32 * 1024 * 1024,
		env: {
			...process.env,
			COUNTERSHAPE_GO: process.env.COUNTERSHAPE_GO ?? "/opt/homebrew/bin/go",
			GOMAXPROCS: "2",
			GOFLAGS: "-mod=readonly -buildvcs=false -p=1",
		},
	});
	if (result.error || result.signal || result.status !== 0 || !result.stdout.includes("P07B-C C1 architecture boundary OK")) {
		fail("P07B_C1_SELFTEST_CLEAN_CHECKER", `${result.status ?? result.signal}: ${result.stderr || result.stdout || result.error}`);
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

async function main() {
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
