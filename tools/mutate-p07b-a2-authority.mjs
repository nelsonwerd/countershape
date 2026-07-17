#!/usr/bin/env node

import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { constants } from "node:fs";
import { chmod, copyFile, lstat, mkdir, mkdtemp, open, readFile, realpath, rm, symlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

import {
	MutationGateError,
	NamedTestOutcome,
	applyExactMutation,
	copyRegularAllowlist,
	mutateRegularFileNoFollow,
	revalidateAdmittedGoExecutable,
} from "./mutate-u1.mjs";
import {
	admitU2DarwinToolchain,
	minimalU2GoEnvironment,
	revalidateAdmittedDarwinCCompiler,
	runNamedU2GoTest,
	runU2Mutant,
} from "./mutate-u2.mjs";
import {
	assertExactU5MutantDelta,
	assertU5ManifestMatchesTree,
	assertU5PrivateCopy,
	assertU5SourceAndSeedUnchanged,
	cleanupU5SeedSnapshot,
	createU5SeedSnapshot,
} from "./mutate-u5.mjs";
import {
	P07B_REVIEWED_EMBED_BINDINGS,
	P07B_REVIEWED_NON_GO_FILES,
	P07B_RUNTIME_SUPPORT_FILES,
	P07B_SANDBOX_FILE_ALLOWLIST,
	P07B_SOURCE_SCOPE_ROOTS,
} from "./mutate-p07b.mjs";

const modulePath = fileURLToPath(import.meta.url);
const repoRoot = resolve(dirname(modulePath), "..");
const modulePrefix = "github.com/nelsonwerd/countershape/";
const p07bA2NamedTestTimeoutMS = 5 * 60_000;
const p07bA2PrivateStageRootEnv = "COUNTERSHAPE_A2_PRIVATE_STAGE_ROOT";
const p07bA2PrivateStageManifestEnv = "COUNTERSHAPE_A2_PRIVATE_STAGE_MANIFEST";

export const P07B_A2_ADDITIONAL_FILES = Object.freeze([
	"internal/emit/node/internal/compilation/input.go",
	"internal/emit/node/model/predicate.go",
	"internal/emit/node/model/predicate_test.go",
	"internal/emit/node/model/source_profile.go",
	"internal/emit/node/service.go",
]);

export const P07B_A2_MUTATION_MODULE_FILES = Object.freeze([
	"tools/mutate-u1.mjs",
	"tools/mutate-u2.mjs",
	"tools/mutate-u3.mjs",
	"tools/mutate-u4.mjs",
	"tools/mutate-u5.mjs",
	"tools/mutate-u6.mjs",
	"tools/mutate-p07b.mjs",
	"tools/mutate-p07b-a2-authority.mjs",
]);

const P07B_A2_ALLOWED_NODE_MODULES = new Set([
	"node:assert/strict", "node:child_process", "node:crypto", "node:fs", "node:fs/promises",
	"node:os", "node:path", "node:url",
]);

const p07bA2ModuleParserProgram = String.raw`
const fs = require("node:fs");
const { SourceTextModule } = require("node:vm");
const acorn = require("internal/deps/acorn/acorn/dist/acorn");
const walk = require("internal/deps/acorn/acorn-walk/dist/walk");
const entries = JSON.parse(fs.readFileSync(0, "utf8"));
if (!Array.isArray(entries)) throw new Error("entries must be an array");
const rows = entries.map(({ file, source }) => {
	try {
		const module = new SourceTextModule(source, { identifier: file });
		const ast = acorn.parse(source, { ecmaVersion: "latest", sourceType: "module", allowHashBang: true });
		let dynamicImports = 0;
		walk.full(ast, (node) => { if (node.type === "ImportExpression") dynamicImports += 1; });
		const dependencies = Array.isArray(module.moduleRequests)
			? module.moduleRequests.map((request) => request.specifier)
			: [...module.dependencySpecifiers];
		return { file, dependencies, dynamicImports };
	} catch (error) {
		return { file, error: { name: String(error?.name ?? "Error"), message: String(error?.message ?? error).slice(0, 512) } };
	}
});
process.stdout.write(JSON.stringify({ runtime: { execPath: process.execPath, version: process.version }, rows }));
`;

export const P07B_A2_RUNTIME_SUPPORT_FILES = Object.freeze([
	...P07B_RUNTIME_SUPPORT_FILES,
	...P07B_A2_MUTATION_MODULE_FILES.filter((file) => !P07B_RUNTIME_SUPPORT_FILES.includes(file)),
	"tools/check-p07b-a2-architecture.mjs",
	"tools/check-p07b-a2-architecture-selftest.mjs",
]);

export const P07B_A2_SANDBOX_FILE_ALLOWLIST = Object.freeze([
	...P07B_SANDBOX_FILE_ALLOWLIST,
	...P07B_A2_ADDITIONAL_FILES,
	...P07B_A2_RUNTIME_SUPPORT_FILES.filter((file) => !P07B_RUNTIME_SUPPORT_FILES.includes(file)),
]);

export const P07B_A2_SOURCE_SCOPE_ROOTS = Object.freeze([
	...P07B_SOURCE_SCOPE_ROOTS,
	"internal/emit/node",
]);

export const REQUIRED_P07B_A2_MUTANT_IDS = Object.freeze([
	"collapse-missing-onto-empty-string",
	"launder-opaque-bytes-through-utf8",
	"deduplicate-ordered-string-list",
	"html-escape-canonical-json-preimage",
	"sort-selected-fields-lexically",
	"preserve-allowed-tuple-insertion-order",
	"compare-tuple-bodies-as-signed-bytes",
	"hash-source-profile-under-wrong-domain",
	"ignore-same-length-foreign-confirmation-binding",
	"drop-source-choicepoint-confirmation-join",
	"authorize-all-translated-tuples",
	"retain-only-first-allowed-tuple",
	"synthesize-cartesian-cross-tuple",
	"relabel-custom-expectation-as-allow-observed",
]);

const requiredIDDigest = "sha256:df139ff84acb8fe32eb08b5f3340a7368523355f04cae2e9d4bc5beb86fee32a";

export const REVIEWED_P07B_A2_MUTANT_CONTRACT = Object.freeze([
	Object.freeze({
		id: "collapse-missing-onto-empty-string",
		file: "internal/emit/node/model/predicate.go",
		find: "\t\treturn portablevalue.Missing(), nil",
		replace: "\t\treturn portablevalue.String(\"\")",
		package: "./internal/emit/node/model",
		testName: "TestExactValueMutationFirewallRetainsAbsenceOpaqueOrderAndJSONBytes",
	}),
	Object.freeze({
		id: "launder-opaque-bytes-through-utf8",
		file: "internal/emit/node/model/predicate.go",
		find: "\t\treturn portablevalue.Bytes(value)",
		replace: "\t\treturn portablevalue.Bytes(bytes.ToValidUTF8(value, []byte{0xef, 0xbf, 0xbd}))",
		package: "./internal/emit/node/model",
		testName: "TestExactValueMutationFirewallRetainsAbsenceOpaqueOrderAndJSONBytes",
	}),
	Object.freeze({
		id: "deduplicate-ordered-string-list",
		file: "internal/emit/node/model/predicate.go",
		find: "\t\treturn portablevalue.OrderedStringList(value)",
		replace: [
			"\t\tdeduplicated := make([]string, 0, len(value))",
			"\t\tseen := make(map[string]struct{}, len(value))",
			"\t\tfor _, member := range value {",
			"\t\t\tif _, duplicate := seen[member]; duplicate { continue }",
			"\t\t\tseen[member] = struct{}{}",
			"\t\t\tdeduplicated = append(deduplicated, member)",
			"\t\t}",
			"\t\treturn portablevalue.OrderedStringList(deduplicated)",
		].join("\n"),
		package: "./internal/emit/node/model",
		testName: "TestExactValueMutationFirewallRetainsAbsenceOpaqueOrderAndJSONBytes",
	}),
	Object.freeze({
		id: "html-escape-canonical-json-preimage",
		file: "internal/emit/node/model/predicate.go",
		find: "base64.StdEncoding.EncodeToString(exact)",
		replace: "base64.StdEncoding.EncodeToString(bytes.ReplaceAll(exact, []byte(\"<\"), []byte(\"\\\\u003c\")))",
		package: "./internal/emit/node/model",
		testName: "TestExactValueMutationFirewallRetainsAbsenceOpaqueOrderAndJSONBytes",
	}),
	Object.freeze({
		id: "sort-selected-fields-lexically",
		file: "internal/emit/node/model/predicate.go",
		find: "\tselected := append([]string(nil), selectedFields...)",
		replace: "\tselected := append([]string(nil), selectedFields...)\n\tsort.Strings(selected)",
		package: "./internal/emit/node/model",
		testName: "TestPredicateCanonicalizesCompleteTupleSetByUnsignedBytes",
	}),
	Object.freeze({
		id: "preserve-allowed-tuple-insertion-order",
		file: "internal/emit/node/model/predicate.go",
		find: [
			"\tunique := make(map[string]ExactTuple, len(allowedTuples))",
			"\tfor tupleIndex, tuple := range allowedTuples {",
			"\t\tif err := validateTupleForSelection(tuple, selected, byID); err != nil {",
			"\t\t\treturn Predicate{}, refuse(\"INVALID_PREDICATE\", fmt.Sprintf(\"allowed tuple %d is incompatible\", tupleIndex), err)",
			"\t\t}",
			"\t\tunique[string(tuple.canonical)] = tuple",
			"\t}",
			"\tallowed := make([]ExactTuple, 0, len(unique))",
			"\tfor _, tuple := range unique {",
			"\t\tallowed = append(allowed, tuple)",
			"\t}",
			"\tsort.Slice(allowed, func(i, j int) bool { return bytes.Compare(allowed[i].canonical, allowed[j].canonical) < 0 })",
		].join("\n"),
		replace: [
			"\t_ = sort.Strings // keep the reviewed import while removing canonical tuple ordering",
			"\tseenAllowed := make(map[string]struct{}, len(allowedTuples))",
			"\tallowed := make([]ExactTuple, 0, len(allowedTuples))",
			"\tfor tupleIndex, tuple := range allowedTuples {",
			"\t\tif err := validateTupleForSelection(tuple, selected, byID); err != nil {",
			"\t\t\treturn Predicate{}, refuse(\"INVALID_PREDICATE\", fmt.Sprintf(\"allowed tuple %d is incompatible\", tupleIndex), err)",
			"\t\t}",
			"\t\tkey := string(tuple.canonical)",
			"\t\tif _, duplicate := seenAllowed[key]; duplicate { continue }",
			"\t\tseenAllowed[key] = struct{}{}",
			"\t\tallowed = append(allowed, tuple)",
			"\t}",
		].join("\n"),
		package: "./internal/emit/node/model",
		testName: "TestPredicateCanonicalizesCompleteTupleSetByUnsignedBytes",
	}),
	Object.freeze({
		id: "compare-tuple-bodies-as-signed-bytes",
		file: "internal/emit/node/model/predicate.go",
		find: "\tsort.Slice(allowed, func(i, j int) bool { return bytes.Compare(allowed[i].canonical, allowed[j].canonical) < 0 })",
		replace: "\tsort.Slice(allowed, func(i, j int) bool { left, right := allowed[i].canonical, allowed[j].canonical; for index := 0; index < len(left) && index < len(right); index++ { if int8(left[index]) != int8(right[index]) { return int8(left[index]) < int8(right[index]) } }; return len(left) < len(right) })",
		package: "./internal/emit/node/model",
		testName: "TestPredicateUsesUnsignedCanonicalTupleByteOrder",
	}),
	Object.freeze({
		id: "hash-source-profile-under-wrong-domain",
		file: "internal/emit/node/model/source_profile.go",
		find: '\tContractSourceProfileDomain = "ContractSourceProfile"',
		replace: '\tContractSourceProfileDomain = "SourceProfile"',
		package: "./internal/emit/node/model",
		testName: "TestSourceProfileUsesExactSevenMemberContractDomain",
	}),
	Object.freeze({
		id: "ignore-same-length-foreign-confirmation-binding",
		file: "internal/confirmation/wire.go",
		find: "sameDigestsInOrder(parsed.executionBindings, r.executionBindings)",
		replace: "true",
		package: "./internal/confirmation",
		testName: "TestFreshConfirmationRetainsExactScheduleOrderedExecutionBindingsWithoutWireChange",
	}),
	Object.freeze({
		id: "drop-source-choicepoint-confirmation-join",
		file: "internal/emit/node/service.go",
		find: [
			"\tif err := requireSourceChoicepointJoin(exactSource, choicepoint); err != nil {",
			"\t\treturn PreparedCompilation{}, err",
			"\t}",
			"\tbindings := confirmationRecord.ExecutionBindingDigests()",
			"\tif len(bindings) == 0 {",
			"\t\treturn PreparedCompilation{}, refuse(CodeConfirmationBindingMismatch, \"confirmation retained no execution-binding roster\", nil)",
			"\t}",
			"\tfor index, binding := range bindings {",
			"\t\tif binding != exactSource.ExecutionBindingDigest() {",
		].join("\n"),
		replace: [
			"\tif err := requireSourceChoicepointJoin(exactSource, choicepoint); false && err != nil {",
			"\t\treturn PreparedCompilation{}, err",
			"\t}",
			"\tbindings := confirmationRecord.ExecutionBindingDigests()",
			"\tif len(bindings) == 0 {",
			"\t\treturn PreparedCompilation{}, refuse(CodeConfirmationBindingMismatch, \"confirmation retained no execution-binding roster\", nil)",
			"\t}",
			"\tfor index, binding := range bindings {",
			"\t\tif false && binding != exactSource.ExecutionBindingDigest() {",
		].join("\n"),
		package: "./testkit/studies/cli_precedence",
		testName: "TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
	}),
	Object.freeze({
		id: "authorize-all-translated-tuples",
		file: "internal/emit/node/service.go",
		find: "\tpredicate, err := model.NewPredicate(source.StimulusDigest(), translations.Profile(), selected, allowedRuling)",
		replace: [
			"\tallTranslated := make([]model.ExactTuple, 0, len(translated))",
			"\tfor _, tuple := range translated { allTranslated = append(allTranslated, tuple) }",
			"\tpredicate, err := model.NewPredicate(source.StimulusDigest(), translations.Profile(), selected, allTranslated)",
		].join("\n"),
		package: "./testkit/studies/cli_precedence",
		testName: "TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
	}),
	Object.freeze({
		id: "retain-only-first-allowed-tuple",
		file: "internal/emit/node/model/predicate.go",
		find: "\tunique := make(map[string]ExactTuple, len(allowedTuples))",
		replace: "\tallowedTuples = allowedTuples[:1]\n\tunique := make(map[string]ExactTuple, len(allowedTuples))",
		package: "./testkit/studies/cli_precedence",
		testName: "TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
	}),
	Object.freeze({
		id: "synthesize-cartesian-cross-tuple",
		file: "internal/emit/node/model/predicate.go",
		find: "\tunique := make(map[string]ExactTuple, len(allowedTuples))",
		replace: [
			"\tif len(allowedTuples) >= 2 && len(allowedTuples[0].fields) >= 2 {",
			"\t\tcrossFields := allowedTuples[0].Fields()",
			"\t\trightFields := allowedTuples[1].Fields()",
			"\t\tcrossFields[len(crossFields)-1] = rightFields[len(rightFields)-1]",
			"\t\tif cross, crossErr := NewExactTuple(crossFields); crossErr == nil { allowedTuples = append(allowedTuples, cross) }",
			"\t}",
			"\tunique := make(map[string]ExactTuple, len(allowedTuples))",
		].join("\n"),
		package: "./testkit/studies/cli_precedence",
		testName: "TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
	}),
	Object.freeze({
		id: "relabel-custom-expectation-as-allow-observed",
		file: "internal/emit/node/service.go",
		find: "\t\treturn compilation.ActionCustomExpectation, nil",
		replace: "\t\treturn compilation.ActionAllowObserved, nil",
		package: "./testkit/studies/http_invoices",
		testName: "TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence",
	}),
]);

const tupleFields = Object.freeze(["file", "find", "id", "package", "replace", "testName"]);
const reviewedTupleDigest = "sha256:fa7678d81d97be58142e29b98804238803bd691dda056b367e5a3348900049de";
export const P07B_A2_MUTANTS = Object.freeze(
	REVIEWED_P07B_A2_MUTANT_CONTRACT.map((tuple) => Object.freeze({ ...tuple })),
);

function sha256(bytes) { return createHash("sha256").update(bytes).digest("hex"); }
function digestText(value) { return `sha256:${sha256(Buffer.from(value, "utf8"))}`; }
function sameArray(left, right) {
	return left.length === right.length && left.every((value, index) => value === right[index]);
}
function occurrenceCount(source, target) {
	return typeof target === "string" && target.length > 0 ? source.split(target).length - 1 : 0;
}
function contractDigest(contract) {
	return digestText(JSON.stringify(contract.map((tuple) => Object.fromEntries(tupleFields.map((field) => [field, tuple[field]])))));
}

export function assertP07BA2MutantDefinitionSet(mutants = P07B_A2_MUTANTS) {
	if (!Object.isFrozen(mutants) || mutants.some((mutant) => !Object.isFrozen(mutant))) {
		throw new MutationGateError("P07B_A2_MUTANT_SET_NOT_IMMUTABLE", "roster and tuples must be recursively frozen");
	}
	if (new Set(P07B_A2_SANDBOX_FILE_ALLOWLIST).size !== P07B_A2_SANDBOX_FILE_ALLOWLIST.length) {
		throw new MutationGateError("P07B_A2_ALLOWLIST_DUPLICATE", "exact copy allowlist contains a duplicate");
	}
	const ids = mutants.map((mutant) => mutant.id);
	const idDigest = digestText(REQUIRED_P07B_A2_MUTANT_IDS.join("\n"));
	if (idDigest !== requiredIDDigest || !sameArray(ids, REQUIRED_P07B_A2_MUTANT_IDS) || new Set(ids).size !== ids.length) {
		throw new MutationGateError("P07B_A2_MUTANT_SET_MISMATCH", `${idDigest}: [${ids.join(",")}]`);
	}
	const actualTupleDigest = contractDigest(REVIEWED_P07B_A2_MUTANT_CONTRACT);
	if (mutants.length !== REVIEWED_P07B_A2_MUTANT_CONTRACT.length || actualTupleDigest !== reviewedTupleDigest) {
		throw new MutationGateError("P07B_A2_MUTANT_CONTRACT_MISMATCH", `${actualTupleDigest}: reviewed tuple digest/count drifted`);
	}
	for (let index = 0; index < mutants.length; index += 1) {
		const mutant = mutants[index];
		const reviewed = REVIEWED_P07B_A2_MUTANT_CONTRACT[index];
		if (!sameArray(Object.keys(mutant).sort(), [...tupleFields].sort()) ||
			tupleFields.some((field) => mutant[field] !== reviewed[field]) ||
			!P07B_A2_SANDBOX_FILE_ALLOWLIST.includes(mutant.file) || mutant.find.length === 0 || mutant.find === mutant.replace ||
			!/^\.\/[^\s]+$/u.test(mutant.package) || !/^Test[A-Za-z0-9_]+$/u.test(mutant.testName)) {
			throw new MutationGateError("P07B_A2_MUTANT_CONTRACT_MISMATCH", `${mutant.id ?? index} differs from reviewed authority`);
		}
	}
}

async function parseP07BA2ModuleRequests(entries, node, spawn = spawnSync) {
	if (!node?.path) throw new MutationGateError("P07B_A2_NODE_RUNTIME_REQUIRED", "module parser requires an admitted Node runtime");
	await revalidateExternalExecutable(node);
	let result;
	try {
		result = spawn(node.path, [
			"--no-warnings", "--experimental-vm-modules", "--expose-internals", "-e", p07bA2ModuleParserProgram,
		], {
			input: JSON.stringify(entries), encoding: "utf8", timeout: 10_000, maxBuffer: 1024 * 1024,
			env: { LANG: "C", LC_ALL: "C", TZ: "UTC", NO_COLOR: "1" },
		});
	} finally {
		await revalidateExternalExecutable(node);
	}
	if (result.error || result.signal || result.status !== 0) {
		throw new MutationGateError(
			"P07B_A2_RUNTIME_MODULE_PARSER_FAILED",
			`${result.status ?? result.signal ?? result.error?.message}: ${(result.stderr ?? "").slice(0, 1024)}`,
		);
	}
	let payload;
	try {
		payload = JSON.parse(result.stdout);
	} catch (error) {
		throw new MutationGateError("P07B_A2_RUNTIME_MODULE_PARSER_FAILED", `malformed parser result: ${error.message}`);
	}
	if (!payload?.runtime || typeof payload.runtime.execPath !== "string" || typeof payload.runtime.version !== "string") {
		throw new MutationGateError("P07B_A2_RUNTIME_MODULE_PARSER_FAILED", "parser omitted its exact Node runtime identity");
	}
	let reportedPath;
	try {
		reportedPath = await realpath(payload.runtime.execPath);
	} catch (error) {
		throw new MutationGateError("P07B_A2_RUNTIME_MODULE_PARSER_FAILED", `parser runtime path: ${error.message}`);
	}
	if (reportedPath !== node.path || payload.runtime.version !== node.version) {
		throw new MutationGateError("P07B_A2_NODE_RUNTIME_MISMATCH", `${reportedPath}:${payload.runtime.version}`);
	}
	const rows = payload.rows;
	if (!Array.isArray(rows) || rows.length !== entries.length) {
		throw new MutationGateError("P07B_A2_RUNTIME_MODULE_PARSER_FAILED", "parser row count differs from declared module count");
	}
	const requests = new Map();
	for (let index = 0; index < rows.length; index += 1) {
		const row = rows[index];
		const expectedFile = entries[index].file;
		if (!row || typeof row !== "object" || row.file !== expectedFile) {
			throw new MutationGateError("P07B_A2_RUNTIME_MODULE_PARSER_FAILED", `parser row ${index} lost exact file order`);
		}
		if (row.error) {
			throw new MutationGateError(
				"P07B_A2_RUNTIME_MODULE_PARSE_FAILED",
				`${expectedFile}:${String(row.error.name).slice(0, 64)}:${String(row.error.message).slice(0, 512)}`,
			);
		}
		if (!Array.isArray(row.dependencies) || row.dependencies.some((specifier) => typeof specifier !== "string") ||
			!Number.isSafeInteger(row.dynamicImports) || row.dynamicImports < 0) {
			throw new MutationGateError("P07B_A2_RUNTIME_MODULE_PARSER_FAILED", `${expectedFile}: malformed dependency row`);
		}
		if (row.dynamicImports !== 0) {
			throw new MutationGateError("P07B_A2_RUNTIME_MODULE_DYNAMIC_IMPORT", `${expectedFile}:${row.dynamicImports}`);
		}
		requests.set(expectedFile, row.dependencies);
	}
	return requests;
}

function resolveP07BA2ModuleSpecifier(sourceRoot, importer, specifier) {
	if (P07B_A2_ALLOWED_NODE_MODULES.has(specifier)) return undefined;
	if (!specifier.startsWith("./") && !specifier.startsWith("../")) {
		throw new MutationGateError("P07B_A2_RUNTIME_MODULE_SPECIFIER_UNSUPPORTED", `${importer}:${specifier}`);
	}
	if (!specifier.endsWith(".mjs")) {
		throw new MutationGateError("P07B_A2_RUNTIME_MODULE_EXTENSION", `${importer}:${specifier}`);
	}
	if (!/^(?:\.{1,2}\/)(?:[A-Za-z0-9._-]+\/)*[A-Za-z0-9._-]+\.mjs$/u.test(specifier)) {
		throw new MutationGateError("P07B_A2_RUNTIME_MODULE_SPECIFIER_INVALID", `${importer}:${specifier}`);
	}
	const importerDirectory = resolve(sourceRoot, dirname(importer));
	const absolute = resolve(importerDirectory, specifier);
	const fromRootNative = relative(sourceRoot, absolute);
	const imported = fromRootNative.split("\\").join("/");
	if (isAbsolute(fromRootNative) || imported === ".." || imported.startsWith("../")) {
		throw new MutationGateError("P07B_A2_RUNTIME_MODULE_ESCAPE", `${importer}:${specifier}`);
	}
	const relativeNative = relative(importerDirectory, absolute);
	const normalizedRelative = relativeNative.split("\\").join("/");
	const canonical = normalizedRelative.startsWith(".") ? normalizedRelative : `./${normalizedRelative}`;
	if (canonical !== specifier) {
		throw new MutationGateError("P07B_A2_RUNTIME_MODULE_SPECIFIER_INVALID", `${importer}:${specifier}`);
	}
	return imported;
}

async function readP07BA2ModuleNoFollow(sourceRoot, file) {
	const canonicalRoot = await realpath(sourceRoot);
	const absolute = resolve(canonicalRoot, file);
	const fromRoot = relative(canonicalRoot, absolute);
	if (isAbsolute(fromRoot) || fromRoot === ".." || fromRoot.startsWith(`..${sep}`)) {
		throw new MutationGateError("P07B_A2_RUNTIME_MODULE_ESCAPE", file);
	}
	const before = await lstat(absolute);
	if (!before.isFile() || before.isSymbolicLink()) {
		throw new MutationGateError("P07B_A2_RUNTIME_MODULE_NONREGULAR", file);
	}
	let handle;
	try {
		handle = await open(absolute, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
		const opened = await handle.stat();
		if (!opened.isFile() || opened.dev !== before.dev || opened.ino !== before.ino || opened.size !== before.size) {
			throw new MutationGateError("P07B_A2_RUNTIME_MODULE_CHANGED", file);
		}
		const bytes = await handle.readFile();
		const after = await handle.stat();
		if (after.dev !== opened.dev || after.ino !== opened.ino || after.size !== opened.size) {
			throw new MutationGateError("P07B_A2_RUNTIME_MODULE_CHANGED", file);
		}
		let source;
		try {
			source = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
		} catch {
			throw new MutationGateError("P07B_A2_RUNTIME_MODULE_INVALID_UTF8", file);
		}
		return Object.freeze({ file, source, bytes: bytes.length, sha256: sha256(bytes) });
	} finally {
		await handle?.close();
	}
}

export async function assertP07BA2MutationModuleClosure(node, sourceRoot = repoRoot) {
	const entries = await Promise.all(P07B_A2_MUTATION_MODULE_FILES.map((file) => readP07BA2ModuleNoFollow(sourceRoot, file)));
	const requests = await parseP07BA2ModuleRequests(entries, node);
	const declared = new Set(P07B_A2_MUTATION_MODULE_FILES);
	const pending = ["tools/mutate-p07b-a2-authority.mjs"];
	const visited = new Set();
	while (pending.length > 0) {
		const current = pending.shift();
		if (visited.has(current)) continue;
		if (!declared.has(current)) throw new MutationGateError("P07B_A2_RUNTIME_MODULE_UNBOUND", current);
		visited.add(current);
		for (const specifier of requests.get(current) ?? []) {
			const imported = resolveP07BA2ModuleSpecifier(sourceRoot, current, specifier);
			if (imported === undefined) continue;
			if (!declared.has(imported)) {
				throw new MutationGateError("P07B_A2_RUNTIME_MODULE_UNBOUND", `${current}:${specifier}->${imported}`);
			}
			pending.push(imported);
		}
	}
	const actual = [...visited].sort();
	const expected = [...declared].sort();
	if (!sameArray(actual, expected)) {
		throw new MutationGateError("P07B_A2_RUNTIME_MODULE_CLOSURE_MISMATCH", `actual=${actual.join(",")} expected=${expected.join(",")}`);
	}
	return Object.freeze(entries.map(({ file, bytes, sha256: digest }) => Object.freeze({ file, bytes, sha256: digest })));
}

export async function assertP07BA2ManifestMatchesTree(node, sourceRoot = repoRoot) {
	const modules = await assertP07BA2MutationModuleClosure(node, sourceRoot);
	const manifest = await assertU5ManifestMatchesTree(
		sourceRoot,
		P07B_A2_SANDBOX_FILE_ALLOWLIST,
		P07B_A2_SOURCE_SCOPE_ROOTS,
		P07B_REVIEWED_NON_GO_FILES,
		P07B_REVIEWED_EMBED_BINDINGS,
		P07B_A2_RUNTIME_SUPPORT_FILES,
	);
	const entries = new Map(manifest.entries.map((entry) => [entry.file, entry]));
	for (const module of modules) {
		const entry = entries.get(module.file);
		if (!entry || entry.bytes !== module.bytes || entry.sha256 !== module.sha256) {
			throw new MutationGateError("P07B_A2_RUNTIME_MODULE_MANIFEST_MISMATCH", module.file);
		}
	}
	return manifest;
}

export async function assertReviewedP07BA2Anchors(node, sourceRoot = repoRoot, mutants = P07B_A2_MUTANTS) {
	assertP07BA2MutantDefinitionSet(mutants);
	await assertP07BA2ManifestMatchesTree(node, sourceRoot);
	for (const mutant of mutants) {
		const source = await readFile(resolve(sourceRoot, mutant.file), "utf8");
		const count = occurrenceCount(source, mutant.find);
		if (count !== 1) throw new MutationGateError("P07B_A2_MUTATION_ANCHOR_INVALID", `${mutant.id}: find=${count}`);
	}
}

export async function assertReviewedP07BA2NamedTests(sourceRoot = repoRoot, mutants = P07B_A2_MUTANTS) {
	for (const mutant of mutants) {
		const prefix = `${mutant.package.slice(2)}/`;
		let count = 0;
		for (const file of P07B_A2_SANDBOX_FILE_ALLOWLIST) {
			if (!file.startsWith(prefix) || !file.endsWith("_test.go")) continue;
			const source = await readFile(resolve(sourceRoot, file), "utf8");
			count += occurrenceCount(source, `func ${mutant.testName}(`);
		}
		if (count !== 1) throw new MutationGateError("P07B_A2_NAMED_TEST_INVALID", `${mutant.id}: count=${count}`);
	}
}

function manifestEqual(left, right) {
	return left.digest === right.digest && left.entries.length === right.entries.length &&
		left.entries.every((entry, index) => JSON.stringify(entry) === JSON.stringify(right.entries[index]));
}

async function ensurePrivateDirectory(path) {
	await mkdir(path, { recursive: true, mode: 0o700 });
	await chmod(path, 0o700);
}

async function admitExternalExecutable(name, candidate, versionArgs) {
	if (!isAbsolute(candidate)) throw new MutationGateError("P07B_A2_EXTERNAL_PATH_NOT_ABSOLUTE", `${name}: ${candidate}`);
	const path = await realpath(candidate);
	const inspectionHome = await mkdtemp(join(await realpath(tmpdir()), `countershape-p07b-a2-${name}-inspection-`));
	let handle;
	try {
		await chmod(inspectionHome, 0o700);
		handle = await open(path, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
		const metadata = await handle.stat();
		if (!metadata.isFile() || (metadata.mode & 0o111) === 0) {
			throw new MutationGateError("P07B_A2_EXTERNAL_NOT_EXECUTABLE", `${name}: ${path}`);
		}
		const bytes = await handle.readFile();
		const result = spawnSync(path, versionArgs, {
			encoding: "utf8", timeout: 10_000, maxBuffer: 256 * 1024,
			env: { HOME: inspectionHome, TMPDIR: inspectionHome, LANG: "C", LC_ALL: "C", NO_COLOR: "1", PATH: "/usr/bin:/bin", TZ: "UTC" },
		});
		if (result.error || result.signal || result.status !== 0) {
			throw new MutationGateError("P07B_A2_EXTERNAL_VERSION_FAILED", `${name}: ${result.status ?? result.signal}: ${result.stderr}`);
		}
		return Object.freeze({
			name, path, dev: metadata.dev, ino: metadata.ino, size: metadata.size, mode: metadata.mode,
			sha256: sha256(bytes), version: (result.stdout || result.stderr).trim().split(/\r?\n/u)[0],
		});
	} finally {
		try { await handle?.close(); } finally { await rm(inspectionHome, { recursive: true, force: true }); }
	}
}

const externalExecutableIdentityFields = Object.freeze(["path", "dev", "ino", "size", "mode", "sha256", "version"]);

function assertSameExternalExecutable(expected, actual, code) {
	for (const field of externalExecutableIdentityFields) {
		if (actual[field] !== expected[field]) {
			throw new MutationGateError(code, `${expected.name}: ${field}`);
		}
	}
	return expected;
}

async function admitCurrentNodeRuntime(candidate) {
	if (!isAbsolute(candidate)) throw new MutationGateError("P07B_A2_EXTERNAL_PATH_NOT_ABSOLUTE", `node: ${candidate}`);
	let runningPath;
	let configuredPath;
	try {
		runningPath = await realpath(process.execPath);
		configuredPath = await realpath(candidate);
	} catch (error) {
		throw new MutationGateError("P07B_A2_NODE_RUNTIME_MISMATCH", error.message);
	}
	if (configuredPath !== runningPath) {
		throw new MutationGateError("P07B_A2_NODE_RUNTIME_MISMATCH", `${configuredPath} != ${runningPath}`);
	}
	return admitExternalExecutable("node", runningPath, ["--version"]);
}

async function revalidateExternalExecutable(admission) {
	const current = await admitExternalExecutable(admission.name, admission.path, ["--version"]);
	assertSameExternalExecutable(admission, current, "P07B_A2_EXTERNAL_CHANGED");
}

function privateStageEnvironment(node, seed) {
	const environment = {};
	for (const name of ["HOME", "TMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE"]) {
		const value = process.env[name];
		if (!value || !isAbsolute(value)) throw new MutationGateError("P07B_A2_PRIVATE_STAGE_ENVIRONMENT_REQUIRED", name);
		environment[name] = value;
	}
	const declaredTools = {
		COUNTERSHAPE_GO: process.env.COUNTERSHAPE_GO,
		COUNTERSHAPE_CC: process.env.COUNTERSHAPE_CC,
		COUNTERSHAPE_NODE: node.path,
		COUNTERSHAPE_GIT: process.env.COUNTERSHAPE_GIT ?? "/usr/bin/git",
	};
	for (const [name, value] of Object.entries(declaredTools)) {
		if (value !== undefined) {
			if (!isAbsolute(value)) throw new MutationGateError("P07B_A2_PRIVATE_STAGE_ENVIRONMENT_INVALID", name);
			environment[name] = value;
		}
	}
	environment.PATH = [...new Set(Object.values(declaredTools).filter(Boolean).map((path) => dirname(path)).concat(["/usr/bin", "/bin"]))].join(":");
	Object.assign(environment, {
		GOENV: "off", GOWORK: "off", GOTOOLCHAIN: "local", GOPROXY: "off", GOSUMDB: "off", GOVCS: "*:off",
		GOFLAGS: "-mod=readonly -buildvcs=false -p=1", GOMAXPROCS: "2", LANG: "C", LC_ALL: "C", TZ: "UTC", NO_COLOR: "1",
		[p07bA2PrivateStageRootEnv]: seed.root,
		[p07bA2PrivateStageManifestEnv]: seed.manifest.digest,
	});
	return Object.freeze(environment);
}

async function runFromPrivateStage(arguments_, node) {
	const seed = await createU5SeedSnapshot({
		sourceRoot: repoRoot,
		allowlist: P07B_A2_SANDBOX_FILE_ALLOWLIST,
		scopeRoots: P07B_A2_SOURCE_SCOPE_ROOTS,
		reviewedNonGoFiles: P07B_REVIEWED_NON_GO_FILES,
		embedBindings: P07B_REVIEWED_EMBED_BINDINGS,
		runtimeSupportFiles: P07B_A2_RUNTIME_SUPPORT_FILES,
	});
	try {
		await assertU5PrivateCopy(seed.root, seed.allowlist);
		const script = resolve(seed.root, "tools/mutate-p07b-a2-authority.mjs");
		let result;
		await revalidateExternalExecutable(node);
		try {
			result = spawnSync(node.path, [script, ...arguments_], {
				cwd: seed.root, encoding: "utf8", timeout: 60 * 60_000, maxBuffer: 32 * 1024 * 1024,
				env: privateStageEnvironment(node, seed),
			});
		} finally {
			await revalidateExternalExecutable(node);
		}
		if (result.error || result.signal || !Number.isInteger(result.status)) {
			throw new MutationGateError("P07B_A2_PRIVATE_STAGE_INFRASTRUCTURE", `${result.signal ?? result.error?.message ?? result.status}`);
		}
		if (result.status !== 0) {
			throw new MutationGateError(
				"P07B_A2_PRIVATE_STAGE_FAILED",
				`${result.status}: ${(result.stderr || result.stdout).slice(0, 16 * 1024)}`,
			);
		}
		await assertU5SourceAndSeedUnchanged(seed);
		if (result.stderr) process.stderr.write(result.stderr);
		if (result.stdout) process.stdout.write(result.stdout);
	} finally {
		await cleanupU5SeedSnapshot(seed);
	}
}

async function validatePrivateStage(node) {
	const declaredRoot = process.env[p07bA2PrivateStageRootEnv];
	const declaredManifest = process.env[p07bA2PrivateStageManifestEnv];
	if (!declaredRoot || !isAbsolute(declaredRoot) || await realpath(declaredRoot) !== repoRoot) {
		throw new MutationGateError("P07B_A2_PRIVATE_STAGE_INVALID", "loaded module root differs from the private stage");
	}
	await assertU5PrivateCopy(repoRoot, P07B_A2_SANDBOX_FILE_ALLOWLIST);
	const manifest = await assertP07BA2ManifestMatchesTree(node, repoRoot);
	if (!/^sha256:[0-9a-f]{64}$/u.test(declaredManifest ?? "") || manifest.digest !== declaredManifest) {
		throw new MutationGateError("P07B_A2_PRIVATE_STAGE_MANIFEST_MISMATCH", `${manifest.digest} != ${declaredManifest}`);
	}
}

function toolFingerprints(toolchain, external) {
	return Object.freeze({
		go: Object.freeze({ path: toolchain.executable.path, sha256: toolchain.executable.sha256, version: toolchain.version.line }),
		cc: Object.freeze({ path: toolchain.compiler.path, sha256: toolchain.compiler.sha256, facts: toolchain.cgo.digest }),
		node: Object.freeze({ path: external.node.path, sha256: external.node.sha256, version: external.node.version }),
		git: Object.freeze({ path: external.git.path, sha256: external.git.sha256, version: external.git.version }),
	});
}

function phaseCommand(mutant, goPath) {
	return Object.freeze([goPath, "test", "-json", mutant.package, "-run", `^${mutant.testName}$`, "-count=1"]);
}

async function prepareExperiment({ mutant, toolchain, phase, seed, phaseRecords, external }) {
	await assertU5SourceAndSeedUnchanged(seed);
	const parent = await mkdtemp(join(await realpath(tmpdir()), `countershape-p07b-a2-${phase}-`));
	await chmod(parent, 0o700);
	const sandbox = join(parent, "repo");
	const cache = join(parent, "cache");
	try {
		await copyRegularAllowlist(seed.root, sandbox, seed.allowlist);
		await assertU5PrivateCopy(sandbox, seed.allowlist);
		const manifest = await assertP07BA2ManifestMatchesTree(external.node, sandbox);
		if (!manifestEqual(manifest, seed.manifest)) {
			throw new MutationGateError("P07B_A2_EXPERIMENT_SEED_MISMATCH", `${mutant.id}:${phase}`);
		}
		for (const directory of ["home", "tmp", "go-tmp", "go-path", "go-build", "go-mod"]) {
			await ensurePrivateDirectory(join(cache, directory));
		}
		revalidateAdmittedGoExecutable(toolchain.executable);
		revalidateAdmittedDarwinCCompiler(toolchain.compiler);
		await revalidateExternalExecutable(external.node);
		await revalidateExternalExecutable(external.git);
		const base = minimalU2GoEnvironment(cache, toolchain);
		const environment = Object.freeze({
			...base,
			PATH: [...new Set([
				dirname(external.node.path), dirname(external.git.path), ...base.PATH.split(":"),
			])].join(":"),
			COUNTERSHAPE_NODE: external.node.path,
			COUNTERSHAPE_GIT: external.git.path,
			GOMAXPROCS: "2",
			GOFLAGS: "-mod=readonly -buildvcs=false -p=1",
		});
		return { parent, sandbox, cache, environment, phase, seed, phaseRecords, external };
	} catch (error) {
		await rm(parent, { recursive: true, force: true });
		throw error;
	}
}

async function cleanupExperiment(context) { await rm(context.parent, { recursive: true, force: true }); }

async function expectedManifest(context, mutant) {
	if (context.phase === "mutant") return assertExactU5MutantDelta(context.sandbox, context.seed, mutant);
	const manifest = await assertP07BA2ManifestMatchesTree(context.external.node, context.sandbox);
	if (!manifestEqual(manifest, context.seed.manifest)) {
		throw new MutationGateError("P07B_A2_CONTROL_SOURCE_CHANGED", `${mutant.id}:${context.phase}`);
	}
	return manifest;
}

async function runNamedTest(context, mutant, toolchain) {
	await assertU5SourceAndSeedUnchanged(context.seed);
	const before = await expectedManifest(context, mutant);
	revalidateAdmittedGoExecutable(toolchain.executable);
	revalidateAdmittedDarwinCCompiler(toolchain.compiler);
	await revalidateExternalExecutable(context.external.node);
	await revalidateExternalExecutable(context.external.git);
	const result = runNamedU2GoTest({
		sandbox: context.sandbox, mutant, environment: context.environment, toolchain,
		spawn(executable, arguments_, options) {
			return spawnSync(executable, arguments_, { ...options, timeout: p07bA2NamedTestTimeoutMS });
		},
	});
	revalidateAdmittedDarwinCCompiler(toolchain.compiler);
	revalidateAdmittedGoExecutable(toolchain.executable);
	await revalidateExternalExecutable(context.external.node);
	await revalidateExternalExecutable(context.external.git);
	const after = await expectedManifest(context, mutant);
	if (!manifestEqual(after, before)) {
		throw new MutationGateError("P07B_A2_TEST_CHANGED_SOURCE", `${mutant.id}:${context.phase}`);
	}
	const command = phaseCommand(mutant, toolchain.executable.path);
	const detail = result.classification.detail;
	const output = result.output ?? "";
	context.phaseRecords.push(Object.freeze({
		phase: context.phase,
		manifest: before.digest,
		sandbox: context.sandbox,
		outcome: result.classification.outcome,
		environment_digest: digestText(JSON.stringify(Object.entries(context.environment).sort())),
		detail_digest: digestText(detail),
		output_digest: digestText(output),
		command,
		command_digest: digestText(JSON.stringify(command)),
	}));
	return result;
}

function exactObjectKeys(value, expected) {
	return value !== null && typeof value === "object" && sameArray(Object.keys(value).sort(), [...expected].sort());
}

function validToolFingerprints(value) {
	if (!Object.isFrozen(value) || !exactObjectKeys(value, ["cc", "git", "go", "node"])) return false;
	for (const name of ["go", "node", "git"]) {
		const entry = value[name];
		if (!Object.isFrozen(entry) || !exactObjectKeys(entry, ["path", "sha256", "version"]) ||
			!isAbsolute(entry.path) || !/^[0-9a-f]{64}$/u.test(entry.sha256) || typeof entry.version !== "string" || entry.version.length === 0) return false;
	}
	return Object.isFrozen(value.cc) && exactObjectKeys(value.cc, ["facts", "path", "sha256"]) &&
		isAbsolute(value.cc.path) && /^[0-9a-f]{64}$/u.test(value.cc.sha256) && /^sha256:[0-9a-f]{64}$/u.test(value.cc.facts);
}

export function exactP07BA2ABADigest(mutant, phaseRecords, seed, fingerprints) {
	const reviewed = REVIEWED_P07B_A2_MUTANT_CONTRACT.find((tuple) => tuple.id === mutant?.id);
	if (!Object.isFrozen(mutant) || !reviewed || tupleFields.some((field) => mutant[field] !== reviewed[field]) ||
		!seed?.manifest || !seed?.sourceManifest || !/^sha256:[0-9a-f]{64}$/u.test(seed.manifest.digest) ||
		!/^sha256:[0-9a-f]{64}$/u.test(seed.sourceManifest.digest) || !validToolFingerprints(fingerprints) ||
		phaseRecords.length !== 3 || !sameArray(phaseRecords.map((record) => record.phase), ["baseline", "mutant", "post-control"]) ||
		new Set(phaseRecords.map((record) => record.sandbox)).size !== 3 || phaseRecords.some((record) => !isAbsolute(record.sandbox)) ||
		phaseRecords[0].manifest !== seed.manifest.digest || phaseRecords[2].manifest !== seed.manifest.digest ||
		phaseRecords[1].manifest === seed.manifest.digest ||
		phaseRecords.some((record) => !/^sha256:[0-9a-f]{64}$/u.test(record.manifest)) ||
		!sameArray(phaseRecords.map((record) => record.outcome), [NamedTestOutcome.Pass, NamedTestOutcome.Failure, NamedTestOutcome.Pass]) ||
		phaseRecords.some((record) => {
			const expected = phaseCommand(mutant, fingerprints.go.path);
			return !Object.isFrozen(record) || !exactObjectKeys(record, [
				"command", "command_digest", "detail_digest", "environment_digest", "manifest", "outcome", "output_digest", "phase", "sandbox",
			]) || !Object.isFrozen(record.command) || !sameArray(record.command, expected) ||
				record.command_digest !== digestText(JSON.stringify(record.command)) ||
				!/^sha256:[0-9a-f]{64}$/u.test(record.detail_digest) ||
				!/^sha256:[0-9a-f]{64}$/u.test(record.environment_digest) ||
				!/^sha256:[0-9a-f]{64}$/u.test(record.output_digest);
		})) {
		throw new MutationGateError("P07B_A2_ABA_RECEIPT_INVALID", `${mutant?.id ?? "unknown"} lacks exact fresh A/B/A closure`);
	}
	return digestText(JSON.stringify({
		schema_version: "countershape.p07b-a2.1.mutation-receipt/v1",
		mutant_tuple: Object.fromEntries(tupleFields.map((field) => [field, mutant[field]])),
		phases: phaseRecords,
		seed_manifest: seed.manifest.digest,
		source_manifest: seed.sourceManifest.digest,
		tool_fingerprints: fingerprints,
	}));
}

function fakeClassification(outcome) {
	return { classification: { outcome, detail: outcome }, output: "" };
}

function syntheticFingerprints() {
	return Object.freeze({
		go: Object.freeze({ path: "/trusted/go", sha256: "a".repeat(64), version: "go1.test" }),
		cc: Object.freeze({ path: "/trusted/cc", sha256: "b".repeat(64), facts: `sha256:${"c".repeat(64)}` }),
		node: Object.freeze({ path: "/trusted/node", sha256: "d".repeat(64), version: "v1.test" }),
		git: Object.freeze({ path: "/trusted/git", sha256: "e".repeat(64), version: "git version test" }),
	});
}

function syntheticPhaseRecords(mutant, seed) {
	const outcomes = [NamedTestOutcome.Pass, NamedTestOutcome.Failure, NamedTestOutcome.Pass];
	return Object.freeze(["baseline", "mutant", "post-control"].map((phase, index) => {
		const sandbox = `/fresh/p07b-a2/${String.fromCharCode(97 + index)}`;
		const command = phaseCommand(mutant, "/trusted/go");
		return Object.freeze({
			phase,
			manifest: index === 1 ? `sha256:${"b".repeat(64)}` : seed.manifest.digest,
			sandbox,
			outcome: outcomes[index],
			environment_digest: digestText(`environment-${phase}`),
			detail_digest: digestText(`detail-${phase}`),
			output_digest: digestText(`output-${phase}`),
			command,
			command_digest: digestText(JSON.stringify(command)),
		});
	}));
}

async function runArchitectureCheck(relativeScript, successMarker, toolchain, external) {
	const goPath = toolchain.executable.path;
	revalidateAdmittedGoExecutable(toolchain.executable);
	await revalidateExternalExecutable(external.node);
	await revalidateExternalExecutable(external.git);
	const paths = {};
	for (const name of ["HOME", "TMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE"]) {
		const value = process.env[name];
		if (!value || !isAbsolute(value)) {
			throw new MutationGateError("P07B_A2_ARCHITECTURE_ENVIRONMENT_REQUIRED", name);
		}
		paths[name] = value;
	}
	const environment = {
		...paths, PATH: [...new Set([dirname(external.node.path), dirname(external.git.path), dirname(goPath), "/usr/bin", "/bin"])].join(":"),
		GOENV: "off", GOWORK: "off", GOTOOLCHAIN: "local", GOPROXY: "off", GOSUMDB: "off", GOVCS: "*:off",
		GOFLAGS: "-mod=readonly -buildvcs=false -p=1", CGO_ENABLED: "0", GOMAXPROCS: "2",
		LANG: "C", LC_ALL: "C", TZ: "UTC", NO_COLOR: "1",
		COUNTERSHAPE_GO: goPath, COUNTERSHAPE_NODE: external.node.path, COUNTERSHAPE_GIT: external.git.path,
	};
	let result;
	try {
		result = spawnSync(external.node.path, [resolve(repoRoot, relativeScript)], {
			cwd: repoRoot, encoding: "utf8", timeout: 180_000, maxBuffer: 8 * 1024 * 1024,
			env: environment,
		});
	} finally {
		await revalidateExternalExecutable(external.node);
		await revalidateExternalExecutable(external.git);
		revalidateAdmittedGoExecutable(toolchain.executable);
	}
	if (result.error || result.signal || result.status !== 0 || !(result.stdout ?? "").includes(successMarker)) {
		throw new MutationGateError(
			"P07B_A2_ARCHITECTURE_PREFLIGHT_FAILED",
			`${relativeScript}: ${result.status ?? result.signal ?? result.error?.message}: ${result.stderr || result.stdout}`,
		);
	}
}

export async function selfTest(node) {
	assertP07BA2MutantDefinitionSet();
	await assertP07BA2ManifestMatchesTree(node, repoRoot);
	await assertReviewedP07BA2Anchors(node);
	await assertReviewedP07BA2NamedTests();
	assert.equal(applyExactMutation("before ANCHOR after", { id: "synthetic", find: "ANCHOR", replace: "MUTATED" }), "before MUTATED after");
	assert.throws(() => applyExactMutation("ANCHOR ANCHOR", { id: "synthetic", find: "ANCHOR", replace: "MUTATED" }),
		(error) => error instanceof MutationGateError);
	assert.throws(() => assertP07BA2MutantDefinitionSet(Object.freeze(P07B_A2_MUTANTS.slice(0, -1))),
		(error) => error instanceof MutationGateError);
	assert.throws(() => assertP07BA2MutantDefinitionSet(Object.freeze([P07B_A2_MUTANTS[1], P07B_A2_MUTANTS[0], ...P07B_A2_MUTANTS.slice(2)])),
		(error) => error instanceof MutationGateError);
	assert.throws(() => assertP07BA2MutantDefinitionSet(P07B_A2_MUTANTS.map((mutant) => ({ ...mutant }))),
		(error) => error instanceof MutationGateError && error.code === "P07B_A2_MUTANT_SET_NOT_IMMUTABLE");
	const tampered = P07B_A2_MUTANTS.map((mutant) => Object.freeze({ ...mutant }));
	tampered[0] = Object.freeze({ ...tampered[0], replace: `${tampered[0].replace}\n// changed` });
	assert.throws(() => assertP07BA2MutantDefinitionSet(Object.freeze(tampered)),
		(error) => error instanceof MutationGateError && error.code === "P07B_A2_MUTANT_CONTRACT_MISMATCH");
	assert.equal(node.path, await realpath(process.execPath));
	const aliasDirectory = await mkdtemp(join(await realpath(tmpdir()), "countershape-p07b-a2-node-alias-"));
	try {
		const alias = join(aliasDirectory, "node");
		await symlink(node.path, alias);
		const aliasAdmission = await admitCurrentNodeRuntime(alias);
		assertSameExternalExecutable(node, aliasAdmission, "P07B_A2_NODE_RUNTIME_MISMATCH");
	} finally {
		await rm(aliasDirectory, { recursive: true, force: true });
	}
	await assert.rejects(admitCurrentNodeRuntime("/usr/bin/git"),
		(error) => error instanceof MutationGateError && error.code === "P07B_A2_NODE_RUNTIME_MISMATCH");
	const mismatchedChild = spawnSync(node.path, [modulePath, "--self-test"], {
		cwd: repoRoot, encoding: "utf8", timeout: 10_000, maxBuffer: 1024 * 1024,
		env: {
			HOME: process.env.HOME, TMPDIR: process.env.TMPDIR, PATH: "/usr/bin:/bin",
			LANG: "C", LC_ALL: "C", TZ: "UTC", NO_COLOR: "1", COUNTERSHAPE_NODE: "/usr/bin/git",
		},
	});
	if (mismatchedChild.error || mismatchedChild.signal || !Number.isInteger(mismatchedChild.status)) {
		assert.fail(`mismatched Node child infrastructure failure: ${mismatchedChild.signal ?? mismatchedChild.error?.message ?? mismatchedChild.status}`);
	}
	assert.notEqual(mismatchedChild.status, 0);
	assert.match(mismatchedChild.stderr, /P07B_A2_NODE_RUNTIME_MISMATCH/u);
	assert.doesNotMatch(mismatchedChild.stdout, /mutation self-test passed/u);
	let parserExecutable = "";
	await parseP07BA2ModuleRequests([], node, (executable) => {
		parserExecutable = executable;
		return { status: 0, signal: null, stdout: JSON.stringify({ runtime: { execPath: node.path, version: node.version }, rows: [] }), stderr: "" };
	});
	assert.equal(parserExecutable, node.path);
	await assert.rejects(parseP07BA2ModuleRequests([], node, () => ({
		status: 0, signal: null,
		stdout: JSON.stringify({ runtime: { execPath: node.path, version: "v0-forged" }, rows: [] }), stderr: "",
	})), (error) => error instanceof MutationGateError && error.code === "P07B_A2_NODE_RUNTIME_MISMATCH");

	const moduleFixture = await mkdtemp(join(await realpath(tmpdir()), "countershape-p07b-a2-modules-"));
	try {
		for (const file of P07B_A2_MUTATION_MODULE_FILES) {
			await mkdir(dirname(resolve(moduleFixture, file)), { recursive: true, mode: 0o700 });
			await copyFile(resolve(repoRoot, file), resolve(moduleFixture, file));
		}
		await assertP07BA2MutationModuleClosure(node, moduleFixture);
		const authorityPath = resolve(moduleFixture, "tools/mutate-p07b-a2-authority.mjs");
		const authoritySource = await readFile(authorityPath, "utf8");
		await writeFile(authorityPath, `${authoritySource}\nconst parserDecoy = "import (\\n"; const quotedDecoy = "import(\\\"./not-code.mjs\\\")";\n`);
		await assertP07BA2MutationModuleClosure(node, moduleFixture);
		for (const test of [
			Object.freeze({ suffix: '\nimport "./unbound.mjs";\n', code: "P07B_A2_RUNTIME_MODULE_UNBOUND" }),
			Object.freeze({
				suffix: `\nconst dynamicModule = ${["im", "port"].join("")} /* exact local edge */ ("./unbound.mjs");\n`,
				code: "P07B_A2_RUNTIME_MODULE_DYNAMIC_IMPORT",
			}),
			Object.freeze({ suffix: '\nimport "./unbound.js";\n', code: "P07B_A2_RUNTIME_MODULE_EXTENSION" }),
		]) {
			await writeFile(authorityPath, `${authoritySource}${test.suffix}`);
			await assert.rejects(assertP07BA2MutationModuleClosure(node, moduleFixture),
				(error) => error instanceof MutationGateError && error.code === test.code);
		}
	} finally {
		await rm(moduleFixture, { recursive: true, force: true });
	}

	let invocation = 0;
	const phases = [];
	const killed = await runU2Mutant(P07B_A2_MUTANTS[0], {}, {
		async prepareExperiment({ phase }) {
			phases.push(phase);
			return { sandbox: `/fresh/p07b-a2/${phase}`, parent: `/fresh/p07b-a2/${phase}` };
		},
		async cleanupExperiment() {},
		async applyMutation() {},
		async runNamedTest() {
			return fakeClassification([NamedTestOutcome.Pass, NamedTestOutcome.Failure, NamedTestOutcome.Pass][invocation++]);
		},
	});
	assert.match(killed, /^KILLED collapse-missing-onto-empty-string/u);
	assert.deepEqual(phases, ["baseline", "mutant", "post-control"]);

	const syntheticSeed = Object.freeze({
		manifest: Object.freeze({ digest: `sha256:${"a".repeat(64)}` }),
		sourceManifest: Object.freeze({ digest: `sha256:${"f".repeat(64)}` }),
	});
	const records = syntheticPhaseRecords(P07B_A2_MUTANTS[0], syntheticSeed);
	const fingerprints = syntheticFingerprints();
	assert.match(exactP07BA2ABADigest(P07B_A2_MUTANTS[0], records, syntheticSeed, fingerprints), /^sha256:[0-9a-f]{64}$/u);
	const badCommand = Object.freeze([
		Object.freeze({ ...records[0], command_digest: `sha256:${"0".repeat(64)}` }), records[1], records[2],
	]);
	assert.throws(() => exactP07BA2ABADigest(P07B_A2_MUTANTS[0], badCommand, syntheticSeed, fingerprints),
		(error) => error instanceof MutationGateError && error.code === "P07B_A2_ABA_RECEIPT_INVALID");
	const badEnvironment = Object.freeze([
		Object.freeze({ ...records[0], environment_digest: "not-a-digest" }), records[1], records[2],
	]);
	assert.throws(() => exactP07BA2ABADigest(P07B_A2_MUTANTS[0], badEnvironment, syntheticSeed, fingerprints),
		(error) => error instanceof MutationGateError && error.code === "P07B_A2_ABA_RECEIPT_INVALID");
	const badManifest = Object.freeze([records[0], Object.freeze({ ...records[1], manifest: "not-a-digest" }), records[2]]);
	assert.throws(() => exactP07BA2ABADigest(P07B_A2_MUTANTS[0], badManifest, syntheticSeed, fingerprints),
		(error) => error instanceof MutationGateError && error.code === "P07B_A2_ABA_RECEIPT_INVALID");

	const seed = await createU5SeedSnapshot({
		sourceRoot: repoRoot,
		allowlist: P07B_A2_SANDBOX_FILE_ALLOWLIST,
		scopeRoots: P07B_A2_SOURCE_SCOPE_ROOTS,
		reviewedNonGoFiles: P07B_REVIEWED_NON_GO_FILES,
		embedBindings: P07B_REVIEWED_EMBED_BINDINGS,
		runtimeSupportFiles: P07B_A2_RUNTIME_SUPPORT_FILES,
	});
	try {
		await assertU5PrivateCopy(seed.root, seed.allowlist);
		const parent = await mkdtemp(join(await realpath(tmpdir()), "countershape-p07b-a2-selftest-"));
		try {
			const sandbox = join(parent, "repo");
			await copyRegularAllowlist(seed.root, sandbox, seed.allowlist);
			await mutateRegularFileNoFollow(sandbox, P07B_A2_MUTANTS[0]);
			await assertExactU5MutantDelta(sandbox, seed, P07B_A2_MUTANTS[0]);
			const secondFile = resolve(sandbox, "go.mod");
			const secondBytes = await readFile(secondFile);
			await writeFile(secondFile, Buffer.concat([secondBytes, Buffer.from("\n// changed\n")]));
			await assert.rejects(assertExactU5MutantDelta(sandbox, seed, P07B_A2_MUTANTS[0]),
				(error) => error instanceof MutationGateError);
		} finally {
			await rm(parent, { recursive: true, force: true });
		}
		const seedTarget = resolve(seed.root, "go.mod");
		const exactSeedBytes = await readFile(seedTarget);
		await writeFile(seedTarget, Buffer.concat([exactSeedBytes, Buffer.from("\n// changed\n")]));
		await assert.rejects(assertU5SourceAndSeedUnchanged(seed), (error) => error instanceof MutationGateError);
		await writeFile(seedTarget, exactSeedBytes);
		await assertU5SourceAndSeedUnchanged(seed);
	} finally {
		await cleanupU5SeedSnapshot(seed);
	}
	await assertP07BA2ManifestMatchesTree(node, repoRoot);
	return `P07B A2.1 mutation self-test passed: ${P07B_A2_MUTANTS.length}/${P07B_A2_MUTANTS.length} frozen tuples, exact anchors/tests/private copies, tamper refusal, and synthetic fresh A/B/A closure.`;
}

export async function main(arguments_ = process.argv.slice(2)) {
	const selfTestOnly = arguments_.length === 1 && arguments_[0] === "--self-test";
	if (!selfTestOnly && arguments_.length !== 0) {
		throw new MutationGateError("P07B_A2_UNSUPPORTED_ARGUMENT", "usage: node tools/mutate-p07b-a2-authority.mjs [--self-test]");
	}
	const node = await admitCurrentNodeRuntime(process.env.COUNTERSHAPE_NODE ?? process.execPath);
	if (!process.env[p07bA2PrivateStageRootEnv]) {
		await runFromPrivateStage(arguments_, node);
		return;
	}
	await validatePrivateStage(node);
	if (selfTestOnly) {
		process.stdout.write(`${await selfTest(node)}\n`);
		return;
	}
	await selfTest(node);
	const toolchain = await admitU2DarwinToolchain();
	const external = Object.freeze({
		node,
		git: await admitExternalExecutable("git", process.env.COUNTERSHAPE_GIT ?? "/usr/bin/git", ["--version"]),
	});
	const fingerprints = toolFingerprints(toolchain, external);
	await runArchitectureCheck("tools/check-p07b-a2-architecture.mjs", "P07B A2.1 architecture boundary OK", toolchain, external);
	await runArchitectureCheck("tools/check-p07b-a2-architecture-selftest.mjs", "P07B A2.1 architecture defensive self-test OK", toolchain, external);
	const seed = await createU5SeedSnapshot({
		sourceRoot: repoRoot,
		allowlist: P07B_A2_SANDBOX_FILE_ALLOWLIST,
		scopeRoots: P07B_A2_SOURCE_SCOPE_ROOTS,
		reviewedNonGoFiles: P07B_REVIEWED_NON_GO_FILES,
		embedBindings: P07B_REVIEWED_EMBED_BINDINGS,
		runtimeSupportFiles: P07B_A2_RUNTIME_SUPPORT_FILES,
	});
	const results = [];
	try {
		for (const mutant of P07B_A2_MUTANTS) {
			const phaseRecords = [];
			const result = await runU2Mutant(mutant, toolchain, {
				prepareExperiment({ mutant: current, toolchain: admitted, phase }) {
					return prepareExperiment({ mutant: current, toolchain: admitted, phase, seed, phaseRecords, external });
				},
				cleanupExperiment,
				applyMutation: mutateRegularFileNoFollow,
				runNamedTest(context, current) { return runNamedTest(context, current, toolchain); },
			});
			const receipt = exactP07BA2ABADigest(mutant, phaseRecords, seed, fingerprints);
			const phases = Object.freeze(phaseRecords.map((record) => Object.freeze({
				phase: record.phase, outcome: record.outcome, manifest: record.manifest,
				command_digest: record.command_digest, environment_digest: record.environment_digest,
			})));
			results.push(Object.freeze({ result, receipt, phases }));
		}
		await assertU5SourceAndSeedUnchanged(seed);
	} finally {
		await cleanupU5SeedSnapshot(seed);
	}
	await revalidateExternalExecutable(external.node);
	await revalidateExternalExecutable(external.git);
	await runArchitectureCheck("tools/check-p07b-a2-architecture.mjs", "P07B A2.1 architecture boundary OK", toolchain, external);
	await runArchitectureCheck("tools/check-p07b-a2-architecture-selftest.mjs", "P07B A2.1 architecture defensive self-test OK", toolchain, external);
	process.stdout.write(`Admitted Node path fingerprint: ${external.node.path} sha256:${external.node.sha256} ${external.node.version}\n`);
	process.stdout.write(`Admitted Git path fingerprint: ${external.git.path} sha256:${external.git.sha256} ${external.git.version}\n`);
	process.stdout.write(`Immutable P07B A2.1 seed manifest: ${seed.manifest.digest}\n`);
	process.stdout.write("Containment notice: private mutation copies retain the invoking user's host filesystem and network authority.\n");
	for (const item of results) {
		process.stdout.write(`${item.result}; exact fresh A/B/A closure digest ${item.receipt}\n`);
		for (const phase of item.phases) {
			process.stdout.write(`  ${phase.phase}: ${phase.outcome}; manifest ${phase.manifest}; command ${phase.command_digest}; environment ${phase.environment_digest}\n`);
		}
	}
	process.stdout.write(`P07B A2.1 mutation gate: ${results.length}/${P07B_A2_MUTANTS.length} required mutants killed; ${results.length}/${P07B_A2_MUTANTS.length} fresh A/B/A receipts.\n`);
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(modulePath)) {
	main().catch((error) => {
		process.stderr.write(`${error.stack ?? error}\n`);
		process.exitCode = 1;
	});
}
