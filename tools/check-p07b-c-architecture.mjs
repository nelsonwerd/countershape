#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { readFile, readdir, lstat } from "node:fs/promises";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const checkerPath = fileURLToPath(import.meta.url);
const repositoryRoot = resolve(dirname(checkerPath), "..");
const modulePath = "github.com/nelsonwerd/countershape";
const packagePath = `${modulePath}/internal/contractexec/model`;
const goExecutable = process.env.COUNTERSHAPE_GO ?? "/opt/homebrew/bin/go";

const expectedProductionFiles = Object.freeze([
	"codec.go",
	"doc.go",
	"errors.go",
	"execution.go",
	"run.go",
	"run_parse.go",
	"target.go",
	"tuple_parse.go",
	"types.go",
]);
const expectedTestFiles = Object.freeze([
	"algebra_test.go",
	"codec_test.go",
	"fuzz_test.go",
	"model_test.go",
	"schema_parity_test.go",
]);
const expectedModelEntries = Object.freeze([
	...expectedProductionFiles,
	...expectedTestFiles,
].sort().map((file) => `${file}:file`));
const expectedProductionImports = Object.freeze([
	"bytes",
	"encoding/base64",
	"errors",
	"fmt",
	`${modulePath}/internal/canon`,
	`${modulePath}/internal/domain`,
	`${modulePath}/internal/emit/node/model`,
	`${modulePath}/internal/portablevalue`,
	"regexp",
	"sort",
	"strconv",
	"strings",
	"unicode",
	"unicode/utf8",
]);
const expectedTestImports = Object.freeze([
	"bytes",
	"crypto/sha256",
	"encoding/base64",
	"encoding/json",
	"fmt",
	`${modulePath}/internal/canon`,
	`${modulePath}/internal/domain`,
	`${modulePath}/internal/emit/node/model`,
	`${modulePath}/internal/portablevalue`,
	"io",
	"os",
	"path/filepath",
	"runtime",
	"testing",
]);
const expectedTestSymbols = Object.freeze([
	"FuzzContractExecutionParser",
	"FuzzContractExecutionTargetParser",
	"FuzzFinalizedContractRunParser",
	"TestAllStartErrorCodesRoundTripWithExactClosure",
	"TestC1ExampleProbe",
	"TestC1SchemaRuntimeOverapproximationProbe",
	"TestCanonicalGettersAndInputsAreDefensive",
	"TestCanonicalObjectBodiesHaveNoSelfDigestOrMutableSelectors",
	"TestCanonicalObjectGraphRoundTripsAndClassifiesWithoutRequestedResult",
	"TestCheckedExamplesParseRebuildAndClassifyExactly",
	"TestClassifierProfileIdentityGolden",
	"TestClassifierTruthTableUsesOnlyEligibleCompleteTupleMembership",
	"TestCleanWitnessUsesExactSixteenTypedReferences",
	"TestClosedEnumBoundsAndTypedRoles",
	"TestClosedRunAlgebraExhaustiveCrossProduct",
	"TestParentMismatchAndNoncanonicalBytesRefuse",
	"TestPrimaryAndCleanupControlsRemainIndependent",
	"TestPrivateEvidenceCeilingsAndZeroCorrelation",
	"TestSchemaProjectionOverapproximationsRefuseAtRuntime",
	"TestScopeDerivationViolationOutranksMissing",
	"TestStrictParsersRejectCoherentlyRehashedUnknownAndDerivedMembers",
	"TestTargetPrimitiveBoundsAndDerivedRelations",
]);
const optInTestSymbols = Object.freeze([
	"TestC1ExampleProbe",
	"TestC1SchemaRuntimeOverapproximationProbe",
]);
const goJSONProfiles = Object.freeze({
	"model-suite": Object.freeze({
		pass: Object.freeze(expectedTestSymbols.filter((name) => !optInTestSymbols.includes(name))),
		skip: optInTestSymbols,
	}),
	"exhaustive-algebra": Object.freeze({
		pass: Object.freeze([
			"TestClassifierTruthTableUsesOnlyEligibleCompleteTupleMembership",
			"TestClosedRunAlgebraExhaustiveCrossProduct",
		]),
		skip: Object.freeze([]),
	}),
	"target-parser-fuzz": Object.freeze({
		pass: Object.freeze(["FuzzContractExecutionTargetParser"]),
		skip: Object.freeze([]),
	}),
	"finalized-run-parser-fuzz": Object.freeze({
		pass: Object.freeze(["FuzzFinalizedContractRunParser"]),
		skip: Object.freeze([]),
	}),
	"execution-parser-fuzz": Object.freeze({
		pass: Object.freeze(["FuzzContractExecutionParser"]),
		skip: Object.freeze([]),
	}),
});
const expectedLocalDependencies = Object.freeze([
	`${modulePath}/internal/adapters/cli/model`,
	`${modulePath}/internal/adapters/http/model`,
	`${modulePath}/internal/canon`,
	`${modulePath}/internal/contractexec/model`,
	`${modulePath}/internal/contractsource`,
	`${modulePath}/internal/domain`,
	`${modulePath}/internal/emit/node/model`,
	`${modulePath}/internal/emit/node/program/v1`,
	`${modulePath}/internal/portablevalue`,
	`${modulePath}/internal/projectionprofile`,
	`${modulePath}/internal/runnerprofile`,
]);
const expectedTargetRoot = Object.freeze([
	"schema_version",
	"kind",
	"target_version",
	"publication_scope",
	"contract_bundle_digest",
	"terminal_residue_binding",
	"tree_binding",
	"attempt_binding",
	"boot_session_binding",
	"runtime_binding",
]);
const expectedRunRoot = Object.freeze([
	"schema_version",
	"kind",
	"run_version",
	"publication_scope",
	"contract_execution_target_digest",
	"attempt_artifact_digest",
	"start_claim_ref",
	"closed_run_witness",
]);
const expectedExecutionRoot = Object.freeze([
	"schema_version",
	"kind",
	"execution_version",
	"publication_scope",
	"classifier_profile",
	"contract_execution_target_digest",
	"finalized_contract_run_digest",
	"result",
]);
const expectedObjects = Object.freeze(["ContractExecutionTarget", "FinalizedContractRun", "ContractExecution"]);
const expectedEvidenceKinds = Object.freeze([
	"MATERIALIZATION_REVALIDATION",
	"RUNTIME_REVALIDATION",
	"PROCESS_RESULT",
	"WAIT_RESULT",
	"DRAIN_RESULT",
	"TEARDOWN_RESULT",
	"ORPHAN_CHECK",
	"FINALIZATION_MARKER",
	"CAPTURED_OBSERVATION",
	"PROJECTION_RESULT",
	"TARGET_INVENTORY",
	"CHILD_BINDINGS",
	"IMPORT_RESOLUTION",
	"SERVICE_BINDINGS",
	"NAMED_PARENT_SECRET_SENTINEL_INHERITANCE",
	"PRIVATE_EVIDENCE_MANIFEST",
]);
const expectedStartErrors = Object.freeze([
	"OS_START_ERROR",
	"PRESPAWN_MATERIALIZATION_REVALIDATION_FAILED",
	"PRESPAWN_RUNTIME_REVALIDATION_FAILED",
]);
const expectedPrimaryReasons = Object.freeze([
	"NONE",
	"MATERIALIZATION_ERROR",
	"START_ERROR",
	"READINESS_ERROR",
	"PROBE_TRANSPORT_ERROR",
	"TIMEOUT",
	"CANCELLED",
	"OUTPUT_LIMIT",
	"PROJECTION_REJECTED",
]);
const expectedScopeDomains = Object.freeze([
	"TARGET_INVENTORY",
	"CHILD_BINDINGS",
	"IMPORT_RESOLUTION",
	"SERVICE_BINDINGS",
	"NAMED_PARENT_SECRET_SENTINEL_INHERITANCE",
]);
const expectedScopeStates = Object.freeze(["COMPLETE", "PARTIAL", "VIOLATED"]);
const expectedResults = Object.freeze(["CONFORMS", "CONTRADICTS", "INELIGIBLE_EXECUTION"]);
const forbiddenSchemaMembers = Object.freeze([
	"artifact_digest",
	"terminal_disposition",
	"requested_result",
	"requested_classification",
	"study_head_advanced",
	"choicepoint_freshened",
	"historical_execution_evidence_reused",
	"latest",
	"current",
	"head",
]);

class ArchitectureError extends Error {
	constructor(code, detail) {
		super(`${code}: ${detail}`);
		this.code = code;
	}
}

function sorted(values) { return [...values].sort(); }
function exact(left, right) { return JSON.stringify(left) === JSON.stringify(right); }
function sha256(bytes) { return createHash("sha256").update(bytes).digest("hex"); }
function canonicalJSON(value) {
	if (value === null || typeof value === "boolean" || typeof value === "string") return JSON.stringify(value);
	if (typeof value === "number") {
		if (!Number.isSafeInteger(value) || Object.is(value, -0)) throw new ArchitectureError("P07B_C1_EXAMPLE_CANONICAL", "unsafe number");
		return String(value);
	}
	if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
	if (!value || typeof value !== "object") throw new ArchitectureError("P07B_C1_EXAMPLE_CANONICAL", typeof value);
	const keys = Object.keys(value).sort((left, right) => Buffer.compare(Buffer.from(left), Buffer.from(right)));
	return `{${keys.map((key) => `${JSON.stringify(key)}:${canonicalJSON(value[key])}`).join(",")}}`;
}
function typedDigest(kind, body) {
	return `sha256:${createHash("sha256").update(`countershape/v1/${kind}\0`).update(canonicalJSON(body)).digest("hex")}`;
}
function slash(value) { return value.split(sep).join("/"); }

function parseJSONStream(source) {
	const values = [];
	let start = -1;
	let depth = 0;
	let quoted = false;
	let escaped = false;
	for (let index = 0; index < source.length; index += 1) {
		const character = source[index];
		if (start < 0) {
			if (/\s/u.test(character)) continue;
			if (character !== "{") throw new ArchitectureError("P07B_C1_GO_LIST_FRAME", `offset ${index}`);
			start = index;
			depth = 1;
			continue;
		}
		if (quoted) {
			if (escaped) escaped = false;
			else if (character === "\\") escaped = true;
			else if (character === "\"") quoted = false;
			continue;
		}
		if (character === "\"") quoted = true;
		else if (character === "{") depth += 1;
		else if (character === "}") {
			depth -= 1;
			if (depth === 0) {
				values.push(JSON.parse(source.slice(start, index + 1)));
				start = -1;
			}
		}
	}
	if (start >= 0 || quoted || depth !== 0) throw new ArchitectureError("P07B_C1_GO_LIST_FRAME", "truncated JSON stream");
	return values;
}

function run(executable, args, code) {
	const result = spawnSync(executable, args, {
		cwd: repositoryRoot,
		encoding: "utf8",
		timeout: 180_000,
		maxBuffer: 32 * 1024 * 1024,
		env: {
			...process.env,
			COUNTERSHAPE_GO: goExecutable,
			GOMAXPROCS: "2",
			GOFLAGS: "-mod=readonly -buildvcs=false -p=1",
		},
	});
	if (result.error || result.signal || result.status !== 0) {
		throw new ArchitectureError(code, `${result.status ?? result.signal}: ${result.stderr || result.stdout || result.error}`);
	}
	return result.stdout;
}

function goList(args) {
	if (!isAbsolute(goExecutable)) throw new ArchitectureError("P07B_C1_GO_EXECUTABLE", "COUNTERSHAPE_GO must be absolute");
	return parseJSONStream(run(goExecutable, ["list", "-json", ...args], "P07B_C1_GO_LIST"));
}

function goTestSymbols() {
	const output = run(goExecutable, [
		"test", "-mod=readonly", "-buildvcs=false", "-p=1", "-count=1", "-list", ".", "./internal/contractexec/model",
	], "P07B_C1_GO_TEST_LIST");
	const lines = output.trimEnd().split(/\r?\n/u);
	const trailer = lines.pop();
	const trailerParts = trailer?.trim().split(/\s+/u) ?? [];
	if (trailerParts.length !== 3 || trailerParts[0] !== "ok" || trailerParts[1] !== packagePath || !/^[0-9.]+s$/u.test(trailerParts[2])) {
		throw new ArchitectureError("P07B_C1_GO_TEST_LIST_FRAME", trailer ?? "missing trailer");
	}
	if (lines.some((line) => !/^(?:Test|Fuzz)[A-Za-z0-9_]+$/u.test(line))) {
		throw new ArchitectureError("P07B_C1_GO_TEST_LIST_FRAME", JSON.stringify(lines));
	}
	return sorted(lines);
}

export function validateGoJSONTranscript(profileName, bytes) {
	const profile = goJSONProfiles[profileName];
	if (!profile) throw new ArchitectureError("P07B_C1_GO_JSON_PROFILE", profileName);
	if (!Buffer.isBuffer(bytes) || bytes.length === 0 || bytes.length > 64 * 1024 * 1024) {
		throw new ArchitectureError("P07B_C1_GO_JSON_SIZE", bytes?.length ?? "not-buffer");
	}
	let source;
	try {
		source = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
	} catch (error) {
		throw new ArchitectureError("P07B_C1_GO_JSON_UTF8", error.message);
	}
	const lines = source.trimEnd().split(/\r?\n/u);
	if (lines.some((line) => line.length === 0 || Buffer.byteLength(line, "utf8") > 1024 * 1024)) {
		throw new ArchitectureError("P07B_C1_GO_JSON_FRAME", "blank or oversized line");
	}
	const events = lines.map((line, index) => {
		try {
			return JSON.parse(line);
		} catch (error) {
			throw new ArchitectureError("P07B_C1_GO_JSON_PARSE", `${index + 1}:${error.message}`);
		}
	});
	if (events.some((event) => !event || typeof event !== "object" || Array.isArray(event) || event.Package !== packagePath)) {
		throw new ArchitectureError("P07B_C1_GO_JSON_PACKAGE", "foreign or malformed event");
	}
	if (events.some((event) => event.Action === "fail")) {
		throw new ArchitectureError("P07B_C1_GO_JSON_FAILURE", "Go reported failure");
	}
	const expected = new Set([...profile.pass, ...profile.skip]);
	const topLevelEvents = events.filter((event) => typeof event.Test === "string" && !event.Test.includes("/"));
	const observed = new Set(topLevelEvents.map((event) => event.Test));
	if (!exact(sorted(observed), sorted(expected))) {
		throw new ArchitectureError("P07B_C1_GO_JSON_TEST_ROSTER", JSON.stringify(sorted(observed)));
	}
	for (const name of expected) {
		const named = topLevelEvents.filter((event) => event.Test === name);
		const runCount = named.filter((event) => event.Action === "run").length;
		const passCount = named.filter((event) => event.Action === "pass").length;
		const skipCount = named.filter((event) => event.Action === "skip").length;
		const wantsPass = profile.pass.includes(name);
		if (runCount !== 1 || passCount !== (wantsPass ? 1 : 0) || skipCount !== (wantsPass ? 0 : 1)) {
			throw new ArchitectureError("P07B_C1_GO_JSON_TEST_RESULT", `${name}:run=${runCount},pass=${passCount},skip=${skipCount}`);
		}
	}
	const packagePasses = events.filter((event) => event.Action === "pass" && !Object.hasOwn(event, "Test"));
	if (packagePasses.length !== 1) {
		throw new ArchitectureError("P07B_C1_GO_JSON_PACKAGE_RESULT", String(packagePasses.length));
	}
	return Object.freeze({ profile: profileName, passed: profile.pass.length, skipped: profile.skip.length });
}

async function readStandardInput() {
	const chunks = [];
	let size = 0;
	for await (const chunk of process.stdin) {
		const bytes = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
		size += bytes.length;
		if (size > 64 * 1024 * 1024) throw new ArchitectureError("P07B_C1_GO_JSON_SIZE", size);
		chunks.push(bytes);
	}
	return Buffer.concat(chunks);
}

async function readJSON(relativePath) {
	return JSON.parse(await readFile(resolve(repositoryRoot, relativePath), "utf8"));
}

function walk(value, visit, pointer = "") {
	visit(value, pointer);
	if (Array.isArray(value)) value.forEach((entry, index) => walk(entry, visit, `${pointer}/${index}`));
	else if (value && typeof value === "object") {
		for (const [key, entry] of Object.entries(value)) walk(entry, visit, `${pointer}/${key}`);
	}
}

function schemaFacts(targetSchema, runSchema, executionSchema) {
	const unclosedObjects = [];
	const incompleteRequiredObjects = [];
	const forbiddenMembers = [];
	for (const [name, schema] of Object.entries({ target: targetSchema, run: runSchema, execution: executionSchema })) {
		walk(schema, (node, pointer) => {
			if (!node || typeof node !== "object" || Array.isArray(node)) return;
			if (node.type === "object" && node.additionalProperties !== false) unclosedObjects.push(`${name}${pointer}`);
			if (node.type === "object" && node.additionalProperties === false && node.properties && typeof node.properties === "object") {
				const properties = sorted(Object.keys(node.properties));
				const required = Array.isArray(node.required) ? sorted(node.required) : [];
				if (
					new Set(required).size !== required.length
					|| required.some((member) => typeof member !== "string")
					|| !exact(required, properties)
				) incompleteRequiredObjects.push(`${name}${pointer}`);
			}
			if (node.properties && typeof node.properties === "object") {
				for (const member of Object.keys(node.properties)) {
					if (forbiddenSchemaMembers.includes(member)) forbiddenMembers.push(`${name}${pointer}/properties/${member}`);
				}
			}
		});
	}
	return {
		targetRoot: Object.keys(targetSchema.properties ?? {}),
		runRoot: Object.keys(runSchema.properties ?? {}),
		executionRoot: Object.keys(executionSchema.properties ?? {}),
		objectKinds: [
			targetSchema.properties?.kind?.const,
			runSchema.properties?.kind?.const,
			executionSchema.properties?.kind?.const,
		],
		unclosedObjects,
		incompleteRequiredObjects,
		forbiddenMembers,
		evidenceKinds: runSchema.$defs?.EvidenceKind?.enum ?? [],
		startErrors: runSchema.$defs?.SpawnObservation?.oneOf?.[0]?.properties?.error_code?.enum ?? [],
		primaryReasons: runSchema.$defs?.ProcessClosure?.properties?.primary_reason?.enum ?? [],
		cleanupControls: runSchema.$defs?.ProcessClosure?.properties?.cleanup_controls?.items?.enum ?? [],
		scopeDomains: runSchema.$defs?.ScopeCheck?.oneOf?.[0]?.properties?.domain?.enum ?? [],
		scopeStates: runSchema.$defs?.StandaloneScope?.properties?.status?.enum ?? [],
		results: executionSchema.properties?.result?.enum ?? [],
		startClaimKind: runSchema.properties?.start_claim_ref?.properties?.kind?.const,
		constants: {
			targetScope: targetSchema.properties?.publication_scope?.const,
			runScope: runSchema.properties?.publication_scope?.const,
			executionScope: executionSchema.properties?.publication_scope?.const,
			classifierProfile: executionSchema.properties?.classifier_profile?.const,
		},
		limits: {
			pathMax: targetSchema.properties?.runtime_binding?.properties?.admitted_executable_path?.maxLength,
			pidMax: runSchema.$defs?.SpawnObservation?.oneOf?.[1]?.properties?.pid?.maximum,
			processEvidenceMax: runSchema.$defs?.ProcessClosure?.properties?.evidence_refs?.maxItems,
			scopeCheckCount: runSchema.$defs?.StandaloneScope?.properties?.checks?.maxItems,
			privateBlobMax: runSchema.$defs?.PrivateEvidence?.properties?.blob_count?.maximum,
			privateByteMax: runSchema.$defs?.PrivateEvidence?.properties?.aggregate_byte_count?.maximum,
		},
	};
}

function exampleFacts(bundle, target, finalizedRun, execution) {
	const witnessRefs = [];
	walk(finalizedRun.closed_run_witness, (node) => {
		if (
			node
			&& typeof node === "object"
			&& !Array.isArray(node)
			&& Object.keys(node).length === 2
			&& Object.hasOwn(node, "kind")
			&& Object.hasOwn(node, "digest")
		) witnessRefs.push(node.kind);
	});
	const targetDigest = typedDigest("ContractExecutionTarget", target);
	const runDigest = typedDigest("FinalizedContractRun", finalizedRun);
	return {
		witnessReferenceKinds: sorted(witnessRefs),
		witnessReferenceCount: witnessRefs.length,
		scopeOrder: finalizedRun.closed_run_witness?.standalone_scope?.checks?.map((check) => check.domain) ?? [],
		result: execution.result,
		graph: {
			bundleToTarget: target.contract_bundle_digest === typedDigest("ContractBundle", bundle)
				&& target.terminal_residue_binding?.current_digest === target.contract_bundle_digest,
			targetToRun: finalizedRun.contract_execution_target_digest === targetDigest
				&& finalizedRun.attempt_artifact_digest === target.attempt_binding?.attempt_artifact_digest,
			runToExecution: execution.contract_execution_target_digest === targetDigest
				&& execution.finalized_contract_run_digest === runDigest,
		},
	};
}

async function topologyFacts() {
	const contractexec = await readdir(resolve(repositoryRoot, "internal/contractexec"), { withFileTypes: true });
	const model = await readdir(resolve(repositoryRoot, "internal/contractexec/model"), { withFileTypes: true });
	const contractexecEntries = [];
	const modelEntries = [];
	for (const entry of contractexec) {
		const stats = await lstat(resolve(repositoryRoot, "internal/contractexec", entry.name));
		contractexecEntries.push(`${entry.name}:${stats.isSymbolicLink() ? "symlink" : entry.isDirectory() ? "directory" : entry.isFile() ? "file" : "other"}`);
	}
	for (const entry of model) {
		const stats = await lstat(resolve(repositoryRoot, "internal/contractexec/model", entry.name));
		modelEntries.push(`${entry.name}:${stats.isSymbolicLink() ? "symlink" : entry.isDirectory() ? "directory" : entry.isFile() ? "file" : "other"}`);
	}
	return { contractexecEntries: sorted(contractexecEntries), modelEntries: sorted(modelEntries) };
}

export async function collectFacts() {
	const [modelPackage] = goList(["./internal/contractexec/model"]);
	const dependencyPackages = goList(["-deps", "./internal/contractexec/model"]);
	const repositoryPackages = goList(["./..."]);
	const [targetSchema, runSchema, executionSchema, bundle, target, finalizedRun, execution, c0] = await Promise.all([
		readJSON("spec/schema/v1/contract-execution-target.schema.json"),
		readJSON("spec/schema/v1/finalized-contract-run.schema.json"),
		readJSON("spec/schema/v1/contract-execution.schema.json"),
		readJSON("spec/examples/v1/contract-bundle.valid.json"),
		readJSON("spec/examples/v1/contract-execution-target.valid.json"),
		readJSON("spec/examples/v1/finalized-contract-run.valid.json"),
		readJSON("spec/examples/v1/contract-execution.valid.json"),
		readJSON("spec/verification/p07b-c-c0-authority.json"),
	]);
	const nonGoBuildFields = [
		"CFiles", "CXXFiles", "MFiles", "HFiles", "FFiles", "SFiles", "SwigFiles", "SwigCXXFiles", "SysoFiles", "EmbedFiles",
	];
	return {
		package: {
			importPath: modelPackage?.ImportPath,
			name: modelPackage?.Name,
			modulePath: modelPackage?.Module?.Path,
			moduleMain: modelPackage?.Module?.Main === true,
			productionFiles: sorted(modelPackage?.GoFiles ?? []),
			testFiles: sorted(modelPackage?.TestGoFiles ?? []),
			xTestFiles: sorted(modelPackage?.XTestGoFiles ?? []),
			productionImports: sorted(modelPackage?.Imports ?? []),
			testImports: sorted(modelPackage?.TestImports ?? []),
			xTestImports: sorted(modelPackage?.XTestImports ?? []),
			testSymbols: goTestSymbols(),
			ignoredGoFiles: sorted(modelPackage?.IgnoredGoFiles ?? []),
			invalidGoFiles: sorted(modelPackage?.InvalidGoFiles ?? []),
			nonGoBuildFiles: sorted(nonGoBuildFields.flatMap((field) => modelPackage?.[field] ?? [])),
		},
		localDependencies: sorted(dependencyPackages
			.map((entry) => entry.ImportPath)
			.filter((entry) => entry === modulePath || entry.startsWith(`${modulePath}/`))),
		externalDependencies: sorted(dependencyPackages
			.filter((entry) => !entry.Standard && entry.ImportPath !== modulePath && !entry.ImportPath.startsWith(`${modulePath}/`))
			.map((entry) => entry.ImportPath)),
		productionImporters: sorted(repositoryPackages
			.filter((entry) => entry.ImportPath !== packagePath && (entry.Imports ?? []).includes(packagePath))
			.map((entry) => entry.ImportPath)),
		topology: await topologyFacts(),
		schema: schemaFacts(targetSchema, runSchema, executionSchema),
		example: exampleFacts(bundle, target, finalizedRun, execution),
		c0: {
			objects: c0.semantic_objects?.map((entry) => entry.name) ?? [],
			scopeDomains: c0.scope_domains ?? [],
			scopeStates: c0.scope_states ?? [],
		},
	};
}

function violation(code, detail) { return Object.freeze({ code, detail }); }

export function validateFacts(facts) {
	const problems = [];
	const add = (code, detail) => problems.push(violation(code, detail));
	if (
		facts.package?.importPath !== packagePath
		|| facts.package?.name !== "model"
		|| facts.package?.modulePath !== modulePath
		|| facts.package?.moduleMain !== true
	) add("P07B_C1_PACKAGE_IDENTITY", JSON.stringify(facts.package));
	if (!exact(facts.package?.productionFiles, expectedProductionFiles)) {
		add("P07B_C1_PRODUCTION_TOPOLOGY", JSON.stringify(facts.package?.productionFiles));
	}
	if (!exact(facts.package?.testFiles, expectedTestFiles) || !exact(facts.package?.xTestFiles, [])) {
		add("P07B_C1_TEST_TOPOLOGY", JSON.stringify({ test: facts.package?.testFiles, xTest: facts.package?.xTestFiles }));
	}
	if (!exact(facts.topology?.contractexecEntries, ["model:directory"])) {
		add("P07B_C1_PRODUCTION_TOPOLOGY", JSON.stringify(facts.topology?.contractexecEntries));
	}
	if (!exact(facts.topology?.modelEntries, expectedModelEntries)) {
		add("P07B_C1_MODEL_TOPOLOGY", JSON.stringify(facts.topology?.modelEntries));
	}
	if (!exact(facts.package?.productionImports, expectedProductionImports)) {
		add("P07B_C1_PRODUCTION_IMPORT_ROSTER", JSON.stringify(facts.package?.productionImports));
	}
	if (!exact(facts.package?.testImports, expectedTestImports) || !exact(facts.package?.xTestImports, [])) {
		add("P07B_C1_TEST_IMPORT_ROSTER", JSON.stringify({ test: facts.package?.testImports, xTest: facts.package?.xTestImports }));
	}
	if (!exact(facts.package?.testSymbols, expectedTestSymbols)) {
		add("P07B_C1_TEST_SYMBOL_ROSTER", JSON.stringify(facts.package?.testSymbols));
	}
	if (
		(facts.package?.ignoredGoFiles ?? []).length > 0
		|| (facts.package?.invalidGoFiles ?? []).length > 0
		|| (facts.package?.nonGoBuildFiles ?? []).length > 0
	) add("P07B_C1_PRODUCTION_TOPOLOGY", "ignored, invalid, or foreign build input");
	if (!exact(facts.localDependencies, expectedLocalDependencies) || (facts.externalDependencies ?? []).length > 0) {
		add("P07B_C1_DEPENDENCY_CLOSURE", JSON.stringify({ local: facts.localDependencies, external: facts.externalDependencies }));
	}
	if ((facts.productionImporters ?? []).length > 0) {
		add("P07B_C1_PREMATURE_IMPORTER", JSON.stringify(facts.productionImporters));
	}
	if ((facts.schema?.unclosedObjects ?? []).length > 0) add("P07B_C1_SCHEMA_CLOSURE", facts.schema.unclosedObjects.join(","));
	if ((facts.schema?.incompleteRequiredObjects ?? []).length > 0) {
		add("P07B_C1_SCHEMA_REQUIRED_ROSTER", facts.schema.incompleteRequiredObjects.join(","));
	}
	if ((facts.schema?.forbiddenMembers ?? []).length > 0) add("P07B_C1_OBJECT_ROSTER", facts.schema.forbiddenMembers.join(","));
	if (
		!exact(facts.schema?.targetRoot, expectedTargetRoot)
		|| !exact(facts.schema?.runRoot, expectedRunRoot)
		|| !exact(facts.schema?.executionRoot, expectedExecutionRoot)
		|| !exact(facts.schema?.objectKinds, expectedObjects)
		|| facts.schema?.startClaimKind !== "StartClaim"
	) add("P07B_C1_OBJECT_ROSTER", "root, kind, or StartClaim roster drift");
	if (
		!exact(facts.schema?.evidenceKinds, expectedEvidenceKinds)
		|| !exact(facts.schema?.startErrors, expectedStartErrors)
		|| !exact(facts.schema?.primaryReasons, expectedPrimaryReasons)
		|| !exact(facts.schema?.cleanupControls, ["TEARDOWN_ERROR", "ORPHAN_RISK"])
		|| !exact(facts.schema?.results, expectedResults)
	) add("P07B_C1_ENUM_ROSTER", "closed enum roster drift");
	if (
		!exact(facts.schema?.scopeDomains, expectedScopeDomains)
		|| !exact(facts.schema?.scopeStates, expectedScopeStates)
		|| !exact(facts.c0?.scopeDomains, expectedScopeDomains)
		|| !exact(facts.c0?.scopeStates, expectedScopeStates)
		|| !exact(facts.example?.scopeOrder, expectedScopeDomains)
	) add("P07B_C1_SCOPE_PROFILE", "scope declaration, schema, or example order drift");
	if (!exact(facts.c0?.objects, expectedObjects)) add("P07B_C1_C0_AUTHORITY", JSON.stringify(facts.c0?.objects));
	if (!exact(facts.schema?.constants, {
		targetScope: "IMMUTABLE_NONHEAD_PRESPAWN_AUTHORITY_V1",
		runScope: "IMMUTABLE_NONHEAD_FINALIZED_RUN_V1",
		executionScope: "IMMUTABLE_NONHEAD_CLASSIFICATION_V1",
		classifierProfile: "CONTRACT_EXECUTION_EXACT_TUPLE_V1",
	})) add("P07B_C1_OBJECT_ROSTER", JSON.stringify(facts.schema?.constants));
	if (!exact(facts.schema?.limits, {
		pathMax: 4096,
		pidMax: 2147483647,
		processEvidenceMax: 8,
		scopeCheckCount: 5,
		privateBlobMax: 16,
		privateByteMax: 67108864,
	})) add("P07B_C1_LIMIT_PROFILE", JSON.stringify(facts.schema?.limits));
	if (
		facts.example?.witnessReferenceCount !== 16
		|| !exact(facts.example?.witnessReferenceKinds, sorted(expectedEvidenceKinds))
	) add("P07B_C1_EXAMPLE_REFERENCE_ROSTER", JSON.stringify(facts.example?.witnessReferenceKinds));
	if (
		facts.example?.graph?.bundleToTarget !== true
		|| facts.example?.graph?.targetToRun !== true
		|| facts.example?.graph?.runToExecution !== true
		|| facts.example?.result !== "CONFORMS"
	) add("P07B_C1_EXAMPLE_GRAPH", JSON.stringify(facts.example));
	return problems;
}

async function snapshot(paths) {
	const result = {};
	for (const relativePath of sorted(paths)) {
		const absolute = resolve(repositoryRoot, relativePath);
		const fromRoot = relative(repositoryRoot, absolute);
		if (isAbsolute(fromRoot) || fromRoot === ".." || fromRoot.startsWith(`..${sep}`)) {
			throw new ArchitectureError("P07B_C1_SNAPSHOT_PATH", relativePath);
		}
		result[slash(relativePath)] = sha256(await readFile(absolute));
	}
	return result;
}

async function runIntersection(facts) {
	const paths = [
		"spec/schema/v1/contract-execution-target.schema.json",
		"spec/schema/v1/finalized-contract-run.schema.json",
		"spec/schema/v1/contract-execution.schema.json",
		"spec/examples/v1/contract-execution-target.valid.json",
		"spec/examples/v1/finalized-contract-run.valid.json",
		"spec/examples/v1/contract-execution.valid.json",
		...facts.topology.modelEntries.map((entry) => `internal/contractexec/model/${entry.slice(0, entry.lastIndexOf(":"))}`),
	];
	const before = await snapshot(paths);
	run(process.execPath, [resolve(repositoryRoot, "tools/generate-p07-planning-example.mjs"), "--check"], "P07B_C1_MODEL_EXAMPLE_INTERSECTION");
	run(process.execPath, [resolve(repositoryRoot, "tools/validate-planning.mjs")], "P07B_C1_MODEL_EXAMPLE_INTERSECTION");
	const after = await snapshot(paths);
	if (!exact(before, after)) throw new ArchitectureError("P07B_C1_SNAPSHOT_CHANGED", "model/schema/example inputs changed during read-only checks");
}

async function main() {
	if (process.argv[2] === "--assert-go-json") {
		if (process.argv.length !== 4) {
			throw new ArchitectureError("P07B_C1_ARGUMENTS", "--assert-go-json requires one exact profile");
		}
		const result = validateGoJSONTranscript(process.argv[3], await readStandardInput());
		process.stdout.write(`P07B-C C1 Go JSON target execution OK (${result.profile}: ${result.passed} passed, ${result.skipped} skipped)\n`);
		return;
	}
	if (process.argv.length !== 2) throw new ArchitectureError("P07B_C1_ARGUMENTS", "no arguments accepted");
	const facts = await collectFacts();
	const problems = validateFacts(facts);
	if (problems.length > 0) {
		for (const problem of problems) process.stderr.write(`${problem.code}: ${problem.detail}\n`);
		process.exitCode = 1;
		return;
	}
	await runIntersection(facts);
	process.stdout.write("P07B-C C1 architecture boundary OK\n");
}

if (process.argv[1] && resolve(process.argv[1]) === checkerPath) {
	main().catch((error) => {
		process.stderr.write(`${error.stack ?? error}\n`);
		process.exitCode = 1;
	});
}
