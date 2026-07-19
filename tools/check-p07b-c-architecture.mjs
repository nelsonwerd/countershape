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
const storePackagePath = `${modulePath}/internal/store`;
const goExecutable = process.env.COUNTERSHAPE_GO ?? "/opt/homebrew/bin/go";

const expectedC2ProductionFiles = Object.freeze([
	"execution_interlock.go", "head.go", "head_darwin.go", "nonhead_contract.go",
	"object_store.go", "private_contract_run.go", "reduction_sweep.go",
]);
const expectedC2TestFiles = Object.freeze([
	"execution_interlock_test.go", "head_test.go", "nonhead_contract_test.go",
	"object_store_test.go", "private_contract_run_test.go", "reduction_sweep_test.go",
]);
const expectedC2XTestFiles = Object.freeze(["public_api_test.go"]);
const expectedC2Imports = Object.freeze({
	"execution_interlock.go": Object.freeze([
		"bytes", "context", "errors", "fmt", `${modulePath}/internal/canon`,
		`${modulePath}/internal/domain`, "os", "path/filepath", "sync",
	]),
	"nonhead_contract.go": Object.freeze([
		"bytes", "context", "errors", "fmt", `${modulePath}/internal/canon`,
		`${modulePath}/internal/domain`, "io", "os", "path/filepath", "sort", "strings",
	]),
	"object_store.go": Object.freeze([
		"bytes", "context", "errors", "fmt", `${modulePath}/internal/canon`,
		`${modulePath}/internal/domain`, "io", "os", "path/filepath", "strings", "sync", "unicode/utf8",
	]),
	"private_contract_run.go": Object.freeze([
		"bytes", "context", "errors", "fmt", `${modulePath}/internal/canon`,
		`${modulePath}/internal/domain`, "io", "os", "path/filepath", "sort", "strings", "unicode/utf8",
	]),
});
const expectedC2PackageImports = Object.freeze([
	"bytes", "context", "encoding/json", "errors", "fmt", `${modulePath}/internal/canon`,
	`${modulePath}/internal/choice/promotion/authority`, `${modulePath}/internal/compare`,
	`${modulePath}/internal/confirmation/authority`, `${modulePath}/internal/domain`,
	`${modulePath}/internal/emit/node/authority`, `${modulePath}/internal/reduce`, "io", "os",
	"path/filepath", "sort", "strconv", "strings", "sync", "syscall", "time", "unicode", "unicode/utf8",
]);
const c2NonheadTests = Object.freeze([
	"TestC2FixturePublishesAndReopensExactNonheadObjects",
	"TestC2ExactKeyMappingsConvergeOnlyForTypedParents",
	"TestC2MappingsRejectWrongKindCrossStoreAndAlternateLinks",
	"TestC2CopiedOrParsedBodiesCannotEnterProductionMechanics",
	"TestC2NonheadFaultMatrixSeparatesNoEffectAndUnknownEffect",
	"TestC2NonheadPublicationNeverMutatesStudyHead",
]);
const c2InterlockTests = Object.freeze([
	"TestC2InterlockClaimMultiProcessRaceHasOneStorageWinnerAndNoPermit",
	"TestC2InterlockMustBeAcquiredBeforeTargetKeyedStartClaim",
	"TestC2InterlockRestartReopensIntentWithoutAuthority",
	"TestC2InterlockSameBootAmbiguityRemainsHeld",
	"TestC2InterlockResetRequiresExplicitDifferentBootSession",
	"TestC2InterlockReleaseRequiresTestOnlyDurableTerminalClosure",
	"TestC2InterlockRejectsCorruptCrossStoreAndAlternateOwnerState",
]);
const c2PrivateTests = Object.freeze([
	"TestC2PrivateManifestEnforcesCountSizeAndRosterBounds",
	"TestC2MissingPrivateEvidenceBeforeFinalizationRefuses",
	"TestC2PostFinalizationPurgeChangesAvailabilityOnly",
	"TestC2UnexpectedPrivateLossReportsMissingWithoutCanonicalMutation",
	"TestC2PrivateEvidenceFaultMatrixReopensAcrossRestart",
	"TestC2PurgeCannotDeleteObjectsHeadsOrRetentionFact",
]);
const c2PublicTests = Object.freeze([
	"TestC2StoreExportsNoOfficialIssuerOrRunPermit",
	"TestC2StoreExportsNoListLatestTraversalStatusOrHeadMutationSurface",
]);
const expectedC2TestSymbols = Object.freeze([
	...c2NonheadTests, ...c2InterlockTests, ...c2PrivateTests, ...c2PublicTests,
].sort());
const expectedC2StructFields = Object.freeze({
	attemptStorageRecord: Object.freeze(["storeInstance", "digest", "seal"]),
	targetStorageInput: Object.freeze(["attempt", "object"]),
	runStorageInput: Object.freeze(["target", "claim", "object", "manifest"]),
	executionStorageInput: Object.freeze(["run", "object"]),
	contractStorageRecord: Object.freeze([
		"storeInstance", "object", "authority", "witnessPath", "relationPath", "relation", "relationBytes",
	]),
	interlockLease: Object.freeze(["storeInstance", "state", "seal"]),
	startClaimRecord: Object.freeze([
		"storeInstance", "targetDigest", "attemptDigest", "bootDigest", "generation", "digest", "canonical", "path",
	]),
	startClaimWinner: Object.freeze(["lease", "claim", "seal"]),
	terminalClosureRecord: Object.freeze(["run", "manifest", "seal"]),
	changedBootResetAuthorization: Object.freeze(["currentBoot", "seal"]),
	privateManifestRecord: Object.freeze([
		"storeInstance", "targetDigest", "attemptDigest", "startClaimDigest", "digest", "canonical", "packDigest",
		"packBytes", "blobCount", "aggregateBytes", "entries", "manifestPath", "packPath", "purgePath", "seal",
	]),
});
const c2ProductionPaths = Object.freeze([
	"internal/store/execution_interlock.go",
	"internal/store/nonhead_contract.go",
	"internal/store/private_contract_run.go",
]);
const c2ImportPaths = Object.freeze([...c2ProductionPaths, "internal/store/object_store.go"]);
const c2ReviewedPaths = Object.freeze([
	...c2ProductionPaths,
	"internal/store/object_store.go",
	"internal/store/execution_interlock_test.go",
	"internal/store/nonhead_contract_test.go",
	"internal/store/object_store_test.go",
	"internal/store/private_contract_run_test.go",
	"internal/store/public_api_test.go",
]);
const expectedC2ObjectStoreSurface = Object.freeze([
	"Error", "Error.Error", "Error.Unwrap", "NewSemanticObject", "ObjectAuthority", "ObjectStore",
	"ObjectStore.Open", "ObjectStore.Publish", "ObjectStore.Read", "ObjectStore.Validate",
	"ObjectStore.ValidateExternalPublicationPath", "OpenObjectStore", "SemanticObject", "SemanticObject.CanonicalBytes",
	"SemanticObject.Digest", "SemanticObject.Kind", "SemanticObject.Valid",
].sort());
const expectedC2ObjectStoreFields = Object.freeze([
	"root", "objects", "digestRoot", "studies", "privateCaptures", "contractRoot", "contractLinks", "contractOps",
	"contractRuns", "rootInfo", "objectsInfo", "digestRootInfo", "studiesInfo", "privateInfo", "contractInfo",
	"contractLinkInfo", "contractOpsInfo", "contractRunInfo", "shardInfos", "studyInfos", "contractInfos", "instance",
]);
const expectedC2TestFilesByProfile = Object.freeze({
	"internal/store/nonhead_contract_test.go": c2NonheadTests,
	"internal/store/execution_interlock_test.go": c2InterlockTests,
	"internal/store/private_contract_run_test.go": c2PrivateTests,
	"internal/store/public_api_test.go": c2PublicTests,
});

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
		packagePath,
		pass: Object.freeze(expectedTestSymbols.filter((name) => !optInTestSymbols.includes(name))),
		skip: optInTestSymbols,
	}),
	"exhaustive-algebra": Object.freeze({
		packagePath,
		pass: Object.freeze([
			"TestClassifierTruthTableUsesOnlyEligibleCompleteTupleMembership",
			"TestClosedRunAlgebraExhaustiveCrossProduct",
		]),
		skip: Object.freeze([]),
	}),
	"target-parser-fuzz": Object.freeze({
		packagePath,
		pass: Object.freeze(["FuzzContractExecutionTargetParser"]),
		skip: Object.freeze([]),
	}),
	"finalized-run-parser-fuzz": Object.freeze({
		packagePath,
		pass: Object.freeze(["FuzzFinalizedContractRunParser"]),
		skip: Object.freeze([]),
	}),
	"execution-parser-fuzz": Object.freeze({
		packagePath,
		pass: Object.freeze(["FuzzContractExecutionParser"]),
		skip: Object.freeze([]),
	}),
	"c2-nonhead-persistence": Object.freeze({
		packagePath: storePackagePath, packageArgument: "./internal/store",
		pass: c2NonheadTests, skip: Object.freeze([]),
	}),
	"c2-interlock": Object.freeze({
		packagePath: storePackagePath, packageArgument: "./internal/store",
		pass: c2InterlockTests, skip: Object.freeze([]),
	}),
	"c2-private-evidence": Object.freeze({
		packagePath: storePackagePath, packageArgument: "./internal/store",
		pass: c2PrivateTests, skip: Object.freeze([]),
	}),
	"c2-public-surface": Object.freeze({
		packagePath: storePackagePath, packageArgument: "./internal/store",
		pass: c2PublicTests, skip: Object.freeze([]),
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

function goTestC2Symbols() {
	const output = run(goExecutable, [
		"test", "-mod=readonly", "-buildvcs=false", "-p=1", "-count=1", "-list", "^TestC2", "./internal/store",
	], "P07B_C2_GO_TEST_LIST");
	const lines = output.trimEnd().split(/\r?\n/u);
	const trailer = lines.pop();
	const trailerParts = trailer?.trim().split(/\s+/u) ?? [];
	if (trailerParts.length !== 3 || trailerParts[0] !== "ok" || trailerParts[1] !== storePackagePath || !/^[0-9.]+s$/u.test(trailerParts[2])) {
		throw new ArchitectureError("P07B_C2_GO_TEST_LIST_FRAME", trailer ?? "missing trailer");
	}
	if (lines.some((line) => !/^TestC2[A-Za-z0-9_]+$/u.test(line))) {
		throw new ArchitectureError("P07B_C2_GO_TEST_LIST_FRAME", JSON.stringify(lines));
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
	if (events.some((event) => !event || typeof event !== "object" || Array.isArray(event) || event.Package !== profile.packagePath)) {
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

export function runGoJSONProfile(profileName) {
	const profile = goJSONProfiles[profileName];
	if (!profile?.packageArgument || profile.skip.length !== 0 || profile.pass.length === 0 ||
		profile.pass.some((name) => !/^(?:Test|Fuzz)[A-Za-z0-9_]+$/u.test(name))) {
		throw new ArchitectureError("P07B_C2_GO_JSON_RUN_PROFILE", profileName);
	}
	const pattern = `^(?:${profile.pass.join("|")})$`;
	const output = run(goExecutable, [
		"test", "-mod=readonly", "-buildvcs=false", "-p=1", "-count=1", "-json", "-run", pattern, profile.packageArgument,
	], "P07B_C2_GO_JSON_RUN");
	return validateGoJSONTranscript(profileName, Buffer.from(output, "utf8"));
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

function count(source, expression) { return source.match(expression)?.length ?? 0; }

function goImports(source) {
	const imports = [];
	for (const block of source.matchAll(/^\s*import\s*\(([^]*?)^\s*\)/gmu)) {
		for (const match of block[1].matchAll(/^\s*(?:[._A-Za-z][A-Za-z0-9_]*\s+)?"([^"]+)"/gmu)) imports.push(match[1]);
	}
	for (const match of source.matchAll(/^\s*import\s+(?:[._A-Za-z][A-Za-z0-9_]*\s+)?"([^"]+)"/gmu)) imports.push(match[1]);
	return sorted(imports);
}

function balancedBody(source, expression) {
	const match = expression.exec(source);
	if (!match) return "";
	const openBrace = source.indexOf("{", match.index);
	if (openBrace < 0) return "";
	let depth = 0;
	for (let index = openBrace; index < source.length; index += 1) {
		if (source[index] === "{") depth += 1;
		else if (source[index] === "}") {
			depth -= 1;
			if (depth === 0) return source.slice(openBrace + 1, index);
		}
	}
	return "";
}

function functionBody(source, name) {
	return balancedBody(source, new RegExp(`^func\\s+(?:\\([^)]*\\)\\s+)?${name}\\s*\\(`, "mu"));
}

function methodBody(source, receiver, name) {
	return balancedBody(source, new RegExp(
		`^func\\s+\\(\\s*[^)]*\\*?${receiver}\\s*\\)\\s+${name}\\s*\\(`, "mu",
	));
}

function structFields(source, name) {
	const body = balancedBody(source, new RegExp(`^type\\s+${name}\\s+struct\\s*`, "mu"));
	return body.split(/\r?\n/u).map((line) => line.trim()).filter(Boolean)
		.map((line) => /^([A-Za-z_][A-Za-z0-9_]*)\b/u.exec(line)?.[1]).filter(Boolean);
}

function exportedSurface(source) {
	const functions = [...source.matchAll(/^func\s+([A-Z][A-Za-z0-9_]*)\s*(?:\[[^\]]*\]\s*)?\(/gmu)].map((match) => match[1]);
	const types = [...source.matchAll(/^type\s+([A-Z][A-Za-z0-9_]*)\b/gmu)].map((match) => match[1]);
	const values = [...source.matchAll(/^(?:var|const)\s+([A-Z][A-Za-z0-9_]*)\b/gmu)].map((match) => match[1]);
	const grouped = [...source.matchAll(/^(?:type|var|const)\s*\(([^]*?)^\)/gmu)]
		.flatMap((block) => [...block[1].matchAll(/^\s*([A-Z][A-Za-z0-9_]*)\b/gmu)].map((match) => match[1]));
	const methods = [];
	for (const match of source.matchAll(/^func\s+\(([^)]*)\)\s+([A-Z][A-Za-z0-9_]*)\s*\(/gmu)) {
		const receiver = /\*?([A-Za-z_][A-Za-z0-9_]*)\s*$/u.exec(match[1])?.[1];
		if (receiver && /^[A-Z]/u.test(receiver)) methods.push(`${receiver}.${match[2]}`);
	}
	return sorted([...functions, ...types, ...values, ...grouped, ...methods]);
}

function ordered(body, anchors) {
	let cursor = -1;
	for (const anchor of anchors) {
		cursor = body.indexOf(anchor, cursor + 1);
		if (cursor < 0) return false;
	}
	return true;
}

async function readC2Source(relativePath) {
	const absolute = resolve(repositoryRoot, relativePath);
	const fromRoot = relative(repositoryRoot, absolute);
	if (isAbsolute(fromRoot) || fromRoot === ".." || fromRoot.startsWith(`..${sep}`)) {
		throw new ArchitectureError("P07B_C2_PATH_ESCAPE", relativePath);
	}
	const before = await lstat(absolute);
	if (!before.isFile() || before.isSymbolicLink()) throw new ArchitectureError("P07B_C2_NONREGULAR_FILE", relativePath);
	const bytes = await readFile(absolute);
	const after = await lstat(absolute);
	if (!after.isFile() || after.isSymbolicLink() || before.dev !== after.dev || before.ino !== after.ino || before.size !== after.size) {
		throw new ArchitectureError("P07B_C2_FILE_CHANGED", relativePath);
	}
	let source;
	try {
		source = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
	} catch (error) {
		throw new ArchitectureError("P07B_C2_INVALID_UTF8", `${relativePath}:${error.message}`);
	}
	return Object.freeze({ path: relativePath, source, digest: sha256(bytes) });
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

export async function collectC2Facts() {
	const [storePackage] = goList(["./internal/store"]);
	const entries = await Promise.all(c2ReviewedPaths.map(readC2Source));
	const sources = Object.fromEntries(entries.map((entry) => [entry.path, entry.source]));
	const newProduction = c2ProductionPaths.map((path) => sources[path]).join("\n");
	const changedProduction = c2ImportPaths.map((path) => sources[path]).join("\n");
	const allTests = Object.entries(sources).filter(([path]) => path.endsWith("_test.go"))
		.map(([, source]) => source).join("\n");
	const nonhead = sources["internal/store/nonhead_contract.go"];
	const interlock = sources["internal/store/execution_interlock.go"];
	const privateRun = sources["internal/store/private_contract_run.go"];
	const objectStore = sources["internal/store/object_store.go"];
	const publicAPI = sources["internal/store/public_api_test.go"];
	const directoryEntries = [];
	for (const entry of await readdir(resolve(repositoryRoot, "internal/store"), { withFileTypes: true })) {
		const metadata = await lstat(resolve(repositoryRoot, "internal/store", entry.name));
		directoryEntries.push(`${entry.name}:${metadata.isSymbolicLink() ? "symlink" : entry.isFile() ? "file" : entry.isDirectory() ? "directory" : "other"}`);
	}
	const nonGoBuildFields = [
		"CFiles", "CXXFiles", "MFiles", "HFiles", "FFiles", "SFiles", "SwigFiles", "SwigCXXFiles", "SysoFiles", "EmbedFiles",
	];
	const structSources = {
		attemptStorageRecord: nonhead,
		targetStorageInput: nonhead,
		runStorageInput: nonhead,
		executionStorageInput: nonhead,
		contractStorageRecord: nonhead,
		interlockLease: interlock,
		startClaimRecord: interlock,
		startClaimWinner: interlock,
		terminalClosureRecord: interlock,
		changedBootResetAuthorization: interlock,
		privateManifestRecord: privateRun,
	};
	const privateKindsBody = /var\s+privateEvidenceKindOrder\s*=\s*\[\.\.\.\]string\s*\{([^]*?)\n\}/mu.exec(privateRun)?.[1] ?? "";
	const testFiles = {};
	for (const path of Object.keys(expectedC2TestFilesByProfile)) {
		testFiles[path] = sorted([...sources[path].matchAll(/^func\s+(TestC2[A-Za-z0-9_]+)\s*\(/gmu)].map((match) => match[1]));
	}
	const forbiddenPatterns = [
		["semantic-model-import", /internal\/contractexec\/model/u],
		["process-start", /\b(?:exec\.Command|os\.StartProcess)\s*\(/u],
		["production-capability", /\b(?:OfficialTarget|RunPermit)\b/u],
		["semantic-head", /\b(?:CreateStudy|OpenHead|AdvanceBaseline|AdvanceDivergence|AdvanceReduction|AdvanceConfirmation|AdvanceChoicepoint|AdvanceRuling|AdvanceResidue|ConfirmResiduePublication)\s*\(/u],
	];
	return structuredClone({
		package: {
			importPath: storePackage?.ImportPath,
			name: storePackage?.Name,
			modulePath: storePackage?.Module?.Path,
			moduleMain: storePackage?.Module?.Main === true,
			productionFiles: sorted(storePackage?.GoFiles ?? []),
			testFiles: sorted(storePackage?.TestGoFiles ?? []),
			xTestFiles: sorted(storePackage?.XTestGoFiles ?? []),
			ignoredGoFiles: sorted(storePackage?.IgnoredGoFiles ?? []),
			invalidGoFiles: sorted(storePackage?.InvalidGoFiles ?? []),
			nonGoBuildFiles: sorted(nonGoBuildFields.flatMap((field) => storePackage?.[field] ?? [])),
			productionImports: sorted(storePackage?.Imports ?? []),
		},
		directoryEntries: sorted(directoryEntries),
		imports: Object.fromEntries(c2ImportPaths.map((path) => [path.split("/").at(-1), goImports(sources[path])])),
		newProductionExports: Object.fromEntries(c2ProductionPaths.map((path) => [path, exportedSurface(sources[path])])),
		objectStoreExports: exportedSurface(objectStore),
		compilerParsedSurface: [
			"go/ast", "go/parser", "go/token", "parser.ParseFile", "parser.SkipObjectResolution",
			"ast.IsExported", "*ast.StructType", "field.Names", "c2ReceiverName", "c2FileExportedSurface",
			"c2ForbiddenProcessSurface", "parsed.Imports", "strconv.Unquote", "os/exec",
			"c2ExportedProductionSurface", "wantPackageSurface",
			"execution_interlock.go", "nonhead_contract.go", "object_store.go", "private_contract_run.go",
		].every((anchor) => publicAPI.includes(anchor)),
		structs: Object.fromEntries(Object.entries(structSources).map(([name, source]) => [name, structFields(source, name)])),
		testSymbols: goTestC2Symbols(),
		testFiles,
		forbiddenSurface: forbiddenPatterns.filter(([, expression]) => expression.test(changedProduction)).map(([name]) => name),
		namespaces: {
			fields: structFields(objectStore, "ObjectStore"),
			paths: [
				"contractDirectory       = \"contract-execution\"",
				"contractLinkDirectory   = \"links\"",
				"contractOpsDirectory    = \"operations\"",
				"contractRunDirectory    = \"contract-runs\"",
			].every((anchor) => objectStore.includes(anchor)),
			retained: ["s.contractRoot", "s.contractLinks", "s.contractOps", "s.contractRuns", "s.contractInfos"]
				.every((anchor) => functionBody(objectStore, "assertReady").includes(anchor)),
			replacementTest: sources["internal/store/object_store_test.go"].includes("replacement contract-operation directory retained store authority"),
			caseAliasGuard: count(functionBody(objectStore, "assertReady"), /rejectPathCaseAlias\s*\(/gu) === 2,
		},
		relations: {
			constants: [
				"CONFORMANCE_ATTEMPT_TO_TARGET", "TARGET_TO_FINALIZED_RUN", "RUN_PROFILE_TO_EXECUTION",
				"target-by-attempt", "run-by-target", "execution-by-run-profile", "contract-storage-relation/v1",
			].every((anchor) => nonhead.includes(`\"${anchor}\"`)),
			effects: sorted([...nonhead.matchAll(/contractEffect\s*=\s*"([A-Z_]+)"/gmu)].map((match) => match[1])),
			keyRoster: [
				"ContractStorageRelationKey", "parent_kind", "parent_digest", "secondary_parent_kind", "secondary_parent_digest",
				"child_kind", "child_digest", "target_digest", "attempt_digest", "start_claim_digest", "classifier_profile_digest",
			].every((anchor) => functionBody(nonhead, "contractRelationMaterial").includes(anchor)),
			algebraClosed: ["relationAttemptTarget", "relationTargetRun", "relationRunExecution"].every((anchor) =>
				functionBody(nonhead, "contractRelationAlgebraValid").includes(anchor)) &&
				["targetAttemptDigest", "finalizedRunJoins", "executionJoins"].every((anchor) =>
					functionBody(nonhead, "validateContractRelation").includes(anchor)),
			persistenceOrder: ordered(functionBody(nonhead, "persistContractRecord"), [
				"publishLocked", "createExactHardLink", "relationDirectoryLocked", "createExactPrivateFile",
				"reopenContractWitness", "readExactPrivateFile", "store.assertReady",
			]),
			openConvergence: ordered(functionBody(nonhead, "openContractRecordByParent"), [
				"relationDirectoryLocked", "filepath.Dir(relationPath)", "readExactPrivateFile", "parseContractRelation",
				"contractRelationMaterial", "createExactPrivateFile",
				"contractExactConverged", "readObjectReference", "reopenContractWitness",
			]) && ["relationDirectoryLocked", "ensureContractDirectoryLocked", "filepath.Dir(record.relationPath)",
				"filepath.Dir(record.witnessPath)", "store.assertReady"].every((anchor) =>
				methodBody(nonhead, "contractStorageRecord", "validForLocked").includes(anchor)),
			runManifestGate: ordered(functionBody(nonhead, "persistFinalizedRunRecord"), [
				"validatePrivateManifestForFinalization", "validatePrivateManifestRoster", "persistContractRecord",
			]),
			executionProfileDerived: functionBody(nonhead, "persistExecutionRecord").includes("contractClassifierProfileDigest()") &&
				!functionBody(nonhead, "persistExecutionRecord").includes("input.profile"),
			identityGuards: count(functionBody(nonhead, "ensureContractDirectoryLocked"), /rejectCaseAlias\s*\(/gu) === 2 &&
				functionBody(nonhead, "createExactHardLink").includes("rejectPathCaseAlias(destination)") &&
				functionBody(nonhead, "createExactPrivateFile").includes("rejectCaseAlias(directory, name)") &&
				count(functionBody(nonhead, "readExactPrivateFile"), /rejectPathCaseAlias\s*\(path\)/gu) === 2,
			identityReplacementTests: [
				"c2ReplaceDirectoryWithExactClone", "os.Link(from, to)",
				"typed-relation-real-parent-replacement", "typed-publication-real-parent-replacement",
				"typed-relation-directory-case-alias", "typed-relation-leaf-case-alias",
				"typed-publication-leaf-case-alias",
			].every((anchor) => allTests.includes(anchor)),
		},
		interlock: {
			bootJoin: ordered(functionBody(interlock, "acquireInterlockAndStartClaim"), [
				"targetBootDigest", "targetBoot != bootDigest", "openAndLockStudy",
			]),
			acquisitionOrder: ordered(functionBody(interlock, "acquireInterlockAndStartClaim"), [
				"openAndLockStudy", "os.Lstat(claimPath)", "readExecutionInterlockState", "clearReceiptExactLocked",
				"replaceExecutionInterlock", "createExactPrivateFile", "readExactPrivateFile",
				"readExecutionInterlockState", "startClaimWinner{",
			]),
			releaseOrder: ordered(functionBody(interlock, "releaseInterlockAfterFinalizedRun"), [
				"validatePrivateManifestLocked", "validatePrivateManifestRoster", "readExecutionInterlockState",
				"replaceExecutionInterlock", "persistClearReceiptLocked",
			]),
			resetOrder: ordered(functionBody(interlock, "resetInterlockAfterBootChange"), [
				"current.bootDigest == authorization.currentBoot", "replaceExecutionInterlock", "persistClearReceiptLocked",
			]),
			winnerFreshAndConsumed: ordered(functionBody(interlock, "validForStoreLocked"), [
				"store.assertReady", "winner.claim.validForLocked", "readExecutionInterlockState",
			]) && ordered(functionBody(interlock, "consumeStartClaimWinner"), [
				"store.assertReady", "openAndLockStudy", "validForStoreLocked", "winner.seal.consumed = true",
			]),
			startClaimConvergence: ordered(functionBody(interlock, "openStartClaim"), [
				"store.assertReady", "openAndLockStudy", "readExactPrivateFile", "parseStartClaim",
				"createExactPrivateFile", "contractExactConverged",
			]),
			clearReceiptTransition: [
				"previous_target_digest", "previous_attempt_digest", "previous_boot_digest", "previous_generation",
				"previous_revision", "deriveInterlockGeneration", "sameInterlockState(expectedClear, clear)",
			].every((anchor) => interlock.includes(anchor)),
			clearReceiptConvergence: ordered(functionBody(interlock, "clearReceiptExactLocked"), [
				"value.CanonicalChecked", "createExactPrivateFile", "contractExactConverged", "store.assertReady",
			]),
			clearReceiptFaults: [
				"before-clear-receipt-create", "after-clear-receipt-temporary-sync", "before-clear-receipt-link",
				"after-clear-receipt-link", "after-clear-receipt-directory-sync", "before-clear-receipt-reopen",
			].every((anchor) => interlock.includes(`\"${anchor}\"`)),
			identityBoundary: functionBody(interlock, "replaceExecutionInterlock").includes("rejectPathCaseAlias(path)") &&
				functionBody(interlock, "acquireInterlockAndStartClaim").includes("rejectPathCaseAlias(claimPath)") && [
				"start-claim-real-parent-replacement-preserves-unconsumed-winner",
				"start-claim-directory-case-alias-preserves-winner",
				"start-claim-leaf-case-alias-preserves-winner",
				"case-alias-claim-preflight-does-not-create-interlock",
				"fixed-ops-real-directory-replacement-refuses-consume",
				"fixed-ops-case-alias-refuses-consume",
			].every((anchor) => allTests.includes(anchor)),
			productionSeals: {
				attempt: count(newProduction, /&attemptRecordSeal\{marker:\s*1\}/gu),
				terminal: count(newProduction, /&terminalClosureSeal\{marker:\s*1\}/gu),
				reset: count(newProduction, /&changedBootResetSeal\{marker:\s*1\}/gu),
				lease: count(newProduction, /&interlockLeaseSeal\{marker:\s*1\}/gu),
				winner: count(newProduction, /&startClaimWinnerSeal\{marker:\s*1\}/gu),
				manifest: count(newProduction, /&privateManifestSeal\{marker:\s*1\}/gu),
			},
			testSeals: {
				attempt: count(allTests, /&attemptRecordSeal\{marker:\s*1\}/gu),
				terminal: count(allTests, /&terminalClosureSeal\{marker:\s*1\}/gu),
				reset: count(allTests, /&changedBootResetSeal\{marker:\s*1\}/gu),
			},
		},
		privateEvidence: (() => {
			const purgeBody = functionBody(privateRun, "purgePrivateEvidence");
			const freshPurgeBody = balancedBody(purgeBody, /if\s+!purgeDurable\s*/u);
			return {
			limits: /maxPrivateBlobs\s*=\s*16\b/u.test(privateRun) &&
				/maxPrivateEvidenceBytes\s*=\s*64\s*\*\s*1024\s*\*\s*1024\b/u.test(privateRun),
			kinds: [...privateKindsBody.matchAll(/"([A-Z_]+)"/gu)].map((match) => match[1]),
			states: sorted([...privateRun.matchAll(/privateState[A-Za-z]+\s*=\s*"([A-Z_]+)"/gmu)].map((match) => match[1])),
			manifestClosure: ordered(functionBody(privateRun, "openPrivateManifest"), [
				"parsePrivateManifest", "record.targetDigest", "record.attemptDigest", "record.startClaimDigest", "record.validFor",
			]) && ordered(functionBody(privateRun, "reopenPrivateManifestLocked"), [
				"privateRunDirectoriesLocked", "strictDigestHex", "filepath.Join(manifestDirectory, hex)",
				"filepath.Join(packDirectory, hex)", "filepath.Join(purgeDirectory, hex)",
				"readExactPrivateFile", "samePrivateManifestEntries",
			]),
			manifestOpenConvergence: ordered(functionBody(privateRun, "openPrivateManifest"), [
				"readExactPrivateFile", "parsePrivateManifest", "createExactPrivateFile", "contractExactConverged", "record.validFor",
			]),
			arithmeticBounded: functionBody(privateRun, "parsePrivateManifest").includes("countedBytes > maxPrivateEvidenceBytes-count") &&
				functionBody(privateRun, "validatePrivatePack").includes("entry.count > record.packBytes-entry.offset"),
			availabilityJoins: ordered(functionBody(privateRun, "privateAvailability"), [
				"run.validForLocked", "validatePrivateManifestRoster", "reopenPrivateManifestLocked",
				"readExactPrivateFile", "createExactPrivateFile", "validatePrivatePack",
			]),
			purgeOrder: ordered(purgeBody, [
				"run.validForLocked", "validatePrivateManifestRoster", "reopenPrivateManifestLocked",
				"privatePurgeIntent", "createExactPrivateFile", "purgeDurable = true", "os.Remove", "syncDirectory",
			]),
			purgeCreateCount: count(purgeBody, /\bcreateExactPrivateFile\s*\(/gu) === 2,
			freshPurgeOrder: ordered(freshPurgeBody, [
				"faultBeforePurgeIntent", "createExactPrivateFile", "faultAfterPurgeIntentSync", "readExactPrivateFile",
			]),
				missingPackGate: purgeBody
					.includes("else if err := validatePrivatePack(manifest.packPath, manifest); err != nil"),
				noCanonicalMutation: !/\b(?:publishLocked|advanceHead|CreateStudy|OpenHead)\s*\(/u.test(privateRun),
				identityGuards: functionBody(privateRun, "validatePrivateManifestLocked")
					.includes("rejectPathCaseAlias(record.purgePath)") &&
					count(functionBody(privateRun, "openPrivateManifest"), /rejectPathCaseAlias\s*\(/gu) === 3 &&
					count(functionBody(privateRun, "createPrivateManifest"), /rejectPathCaseAlias\s*\(/gu) === 3 &&
					["manifestAliasErr", "packAliasErr", "purgeAliasErr"].every((anchor) =>
						functionBody(privateRun, "reopenPrivateManifestLocked").includes(anchor)) &&
					count(functionBody(privateRun, "validatePrivatePack"), /rejectPathCaseAlias\s*\(path\)/gu) === 2 &&
					ordered(purgeBody, ["rejectPathCaseAlias(manifest.packPath)", "os.Remove(manifest.packPath)"]) &&
					functionBody(privateRun, "createExactPrivatePack").includes("rejectCaseAlias(directory, name)"),
				identityReplacementTests: [
					"dynamic-private-real-directory-replacement-refuses",
					"private-case-alias-preflight-is-known-no-effect",
					"private-manifest-leaf-case-alias-refuses", "private-pack-leaf-case-alias-refuses",
				].every((anchor) => allTests.includes(anchor)),
			};
		})(),
	});
}

function violation(code, detail) { return Object.freeze({ code, detail }); }

export function validateC2Facts(facts) {
	const problems = [];
	const add = (code, detail) => problems.push(violation(code, detail));
	if (facts.package?.importPath !== storePackagePath || facts.package?.name !== "store" ||
		facts.package?.modulePath !== modulePath || facts.package?.moduleMain !== true) {
		add("P07B_C2_PACKAGE_IDENTITY", JSON.stringify(facts.package));
	}
	if (!exact(facts.package?.productionFiles, expectedC2ProductionFiles) ||
		!exact(facts.package?.testFiles, expectedC2TestFiles) || !exact(facts.package?.xTestFiles, expectedC2XTestFiles) ||
		!exact(facts.package?.ignoredGoFiles, ["head_unsupported.go"]) ||
		(facts.package?.invalidGoFiles ?? []).length !== 0 || (facts.package?.nonGoBuildFiles ?? []).length !== 0) {
		add("P07B_C2_STORE_TOPOLOGY", JSON.stringify(facts.package));
	}
	if (!exact(facts.package?.productionImports, expectedC2PackageImports)) {
		add("P07B_C2_IMPORT_ROSTER", `package:${JSON.stringify(facts.package?.productionImports)}`);
	}
	const expectedEntries = sorted([
		...expectedC2ProductionFiles, ...expectedC2TestFiles, ...expectedC2XTestFiles, "head_unsupported.go",
	].map((name) => `${name}:file`));
	if (!exact(facts.directoryEntries, expectedEntries)) add("P07B_C2_STORE_TOPOLOGY", JSON.stringify(facts.directoryEntries));
	for (const [file, expected] of Object.entries(expectedC2Imports)) {
		if (!exact(facts.imports?.[file], expected)) add("P07B_C2_IMPORT_ROSTER", `${file}:${JSON.stringify(facts.imports?.[file])}`);
	}
	if (Object.values(facts.newProductionExports ?? {}).some((surface) => surface.length !== 0) ||
		!exact(facts.objectStoreExports, expectedC2ObjectStoreSurface) || facts.compilerParsedSurface !== true) {
		add("P07B_C2_EXPORTED_SURFACE", JSON.stringify({
			new: facts.newProductionExports, store: facts.objectStoreExports, compilerParsed: facts.compilerParsedSurface,
		}));
	}
	for (const [name, expected] of Object.entries(expectedC2StructFields)) {
		if (!exact(facts.structs?.[name], expected)) add("P07B_C2_AUTHORITY_SHAPE", `${name}:${JSON.stringify(facts.structs?.[name])}`);
	}
	if (!exact(facts.testSymbols, expectedC2TestSymbols)) add("P07B_C2_TEST_SYMBOL_ROSTER", JSON.stringify(facts.testSymbols));
	for (const [path, expected] of Object.entries(expectedC2TestFilesByProfile)) {
		if (!exact(facts.testFiles?.[path], sorted(expected))) add("P07B_C2_TEST_FILE_ROSTER", `${path}:${JSON.stringify(facts.testFiles?.[path])}`);
	}
	if ((facts.forbiddenSurface ?? []).length !== 0) add("P07B_C2_FORBIDDEN_PRODUCTION_SURFACE", facts.forbiddenSurface.join(","));
	if (!facts.namespaces?.paths || !facts.namespaces?.retained || !facts.namespaces?.replacementTest ||
		!facts.namespaces?.caseAliasGuard ||
		!exact(facts.namespaces?.fields, expectedC2ObjectStoreFields)) {
		add("P07B_C2_NAMESPACE_IDENTITY", JSON.stringify(facts.namespaces));
	}
	if (!facts.relations?.constants || !facts.relations?.keyRoster || !facts.relations?.algebraClosed ||
		!facts.relations?.persistenceOrder || !facts.relations?.openConvergence ||
		!facts.relations?.runManifestGate || !facts.relations?.executionProfileDerived ||
		!facts.relations?.identityGuards || !facts.relations?.identityReplacementTests ||
		!exact(facts.relations?.effects, ["AMBIGUOUS", "EXACT_CONVERGED", "KNOWN_NO_EFFECT"])) {
		add("P07B_C2_RELATION_ALGEBRA", JSON.stringify(facts.relations));
	}
	if (!facts.interlock?.bootJoin || !facts.interlock?.acquisitionOrder || !facts.interlock?.releaseOrder || !facts.interlock?.resetOrder ||
		!facts.interlock?.winnerFreshAndConsumed || !facts.interlock?.startClaimConvergence || !facts.interlock?.clearReceiptTransition ||
		!facts.interlock?.clearReceiptConvergence || !facts.interlock?.clearReceiptFaults || !facts.interlock?.identityBoundary) {
		add("P07B_C2_INTERLOCK_PROTOCOL", JSON.stringify(facts.interlock));
	}
	if (!exact(facts.interlock?.productionSeals, { attempt: 0, terminal: 0, reset: 0, lease: 1, winner: 1, manifest: 2 }) ||
		!exact(facts.interlock?.testSeals, { attempt: 1, terminal: 1, reset: 1 })) {
		add("P07B_C2_SEAL_OWNERSHIP", JSON.stringify({ production: facts.interlock?.productionSeals, test: facts.interlock?.testSeals }));
	}
	if (!facts.privateEvidence?.limits || !exact(facts.privateEvidence?.kinds, [
		"MATERIALIZATION_REVALIDATION", "RUNTIME_REVALIDATION", "PROCESS_RESULT", "WAIT_RESULT", "DRAIN_RESULT",
		"TEARDOWN_RESULT", "ORPHAN_CHECK", "FINALIZATION_MARKER", "CAPTURED_OBSERVATION", "PROJECTION_RESULT",
		"TARGET_INVENTORY", "CHILD_BINDINGS", "IMPORT_RESOLUTION", "SERVICE_BINDINGS",
		"NAMED_PARENT_SECRET_SENTINEL_INHERITANCE",
	]) || !exact(facts.privateEvidence?.states, ["MISSING_UNEXPECTED", "PURGED", "RETAINED"]) ||
		!facts.privateEvidence?.manifestClosure || !facts.privateEvidence?.manifestOpenConvergence || !facts.privateEvidence?.arithmeticBounded ||
		!facts.privateEvidence?.availabilityJoins || !facts.privateEvidence?.purgeOrder ||
		!facts.privateEvidence?.purgeCreateCount || !facts.privateEvidence?.freshPurgeOrder ||
		!facts.privateEvidence?.missingPackGate || !facts.privateEvidence?.noCanonicalMutation ||
		!facts.privateEvidence?.identityGuards || !facts.privateEvidence?.identityReplacementTests) {
		add("P07B_C2_PRIVATE_EVIDENCE", JSON.stringify(facts.privateEvidence));
	}
	return problems;
}

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

function runInheritedB() {
	const result = spawnSync(process.execPath, [resolve(repositoryRoot, "tools/check-p07b-b-architecture.mjs")], {
		cwd: repositoryRoot,
		encoding: "utf8",
		timeout: 300_000,
		maxBuffer: 32 * 1024 * 1024,
		env: process.env,
	});
	if (result.error || result.signal || result.status !== 0 || result.stderr !== "" ||
		result.stdout.trim() !== "P07B B architecture boundary OK") {
		throw new ArchitectureError(
			"P07B_C2_INHERITED_B_FAILED", `${result.status ?? result.signal}: ${result.stderr || result.stdout || result.error}`,
		);
	}
}

async function runC2Boundary() {
	const before = await snapshot(c2ReviewedPaths);
	runInheritedB();
	const facts = await collectC2Facts();
	const problems = validateC2Facts(facts);
	if (problems.length > 0) {
		for (const problem of problems) process.stderr.write(`${problem.code}: ${problem.detail}\n`);
		process.exitCode = 1;
		return;
	}
	runGoJSONProfile("c2-public-surface");
	const after = await snapshot(c2ReviewedPaths);
	if (!exact(before, after)) {
		throw new ArchitectureError("P07B_C2_SNAPSHOT_CHANGED", "C2 reviewed inputs changed during cumulative checks");
	}
	process.stdout.write("P07B-C C2 cumulative architecture boundary OK\n");
}

async function main() {
	if (process.argv[2] === "--assert-go-json") {
		if (process.argv.length !== 4) {
			throw new ArchitectureError("P07B_C1_ARGUMENTS", "--assert-go-json requires one exact profile");
		}
		const result = validateGoJSONTranscript(process.argv[3], await readStandardInput());
		const phase = result.profile.startsWith("c2-") ? "C2" : "C1";
		process.stdout.write(`P07B-C ${phase} Go JSON target execution OK (${result.profile}: ${result.passed} passed, ${result.skipped} skipped)\n`);
		return;
	}
	if (process.argv[2] === "--run-go-json") {
		if (process.argv.length !== 4 || !process.argv[3].startsWith("c2-")) {
			throw new ArchitectureError("P07B_C2_ARGUMENTS", "--run-go-json requires one exact C2 profile");
		}
		const result = runGoJSONProfile(process.argv[3]);
		process.stdout.write(`P07B-C C2 Go JSON target execution OK (${result.profile}: ${result.passed} passed, ${result.skipped} skipped)\n`);
		return;
	}
	if (process.argv[2] === "--c2") {
		if (process.argv.length !== 3) throw new ArchitectureError("P07B_C2_ARGUMENTS", "--c2 accepts no other arguments");
		await runC2Boundary();
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
