#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { constants } from "node:fs";
import { lstat, open, readdir } from "node:fs/promises";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const checkerPath = fileURLToPath(import.meta.url);
const repositoryRoot = resolve(dirname(checkerPath), "..");
const modulePrefix = "github.com/nelsonwerd/countershape/";

const reviewedFiles = Object.freeze([
	"internal/emit/node/internal/publication/authority.go",
	"internal/emit/node/authority/authority.go",
	"internal/emit/node/publication.go",
	"internal/choice/promotion/residue.go",
	"internal/choice/promotion/service.go",
	"internal/store/head.go",
	"internal/store/object_store.go",
	"internal/contractmaterialize/materialize.go",
	"internal/contractmaterialize/publish_darwin.go",
	"internal/contractmaterialize/publish_unsupported.go",
	"internal/emit/node/internal/publication/authority_test.go",
	"internal/emit/node/publication_store_darwin_test.go",
	"internal/contractmaterialize/materialize_darwin_test.go",
	"internal/store/object_store_test.go",
	"internal/store/head_test.go",
	"internal/store/public_api_test.go",
	"testkit/studies/cli_precedence/reduction_darwin_test.go",
	"testkit/studies/http_invoices/reduction_darwin_test.go",
]);

const pinnedProductionDigests = Object.freeze({
	"internal/emit/node/internal/publication/authority.go": "5f824c367910a5dd2db5a90a69a12546eaf4f370ff990e03c73e245948a73aed",
	"internal/emit/node/authority/authority.go": "e3fee30acb9fc13bc9181f0ef2318325f41764c4f019c27c5c8d48cd5900397c",
	"internal/emit/node/publication.go": "ccf7a4dff7ba4dae731421348e78866925469074d9c571701d6521146a5d3678",
	"internal/choice/promotion/residue.go": "0ad60120b0add6e11002939c985f19e45494e72ec012534ef9df982233f29aa4",
	"internal/contractmaterialize/materialize.go": "7040c0b668629c652afc84b0b941606955b7519a9f0003e28cbd6df553621a12",
	"internal/contractmaterialize/publish_darwin.go": "b319602c888683eaba8a427441fc1f34e80b105434af71884612af652409bf22",
	"internal/contractmaterialize/publish_unsupported.go": "076883b400f3116404e0e3afb9286021a7679f0dafb1cb61b56e3a99eee96c88",
});

const exactImports = Object.freeze({
	"internal/emit/node/internal/publication/authority.go": [
		"bytes", "errors", `${modulePrefix}internal/domain`, `${modulePrefix}internal/emit/node/model`,
	],
	"internal/emit/node/authority/authority.go": [`${modulePrefix}internal/emit/node/internal/publication`],
	"internal/emit/node/publication.go": [
		"bytes", "context", "errors", `${modulePrefix}internal/choice/promotion`, `${modulePrefix}internal/domain`,
		`${modulePrefix}internal/emit/node/internal/publication`, `${modulePrefix}internal/emit/node/model`, `${modulePrefix}internal/store`,
	],
	"internal/choice/promotion/residue.go": [
		"bytes", "context", `${modulePrefix}internal/choice`, `${modulePrefix}internal/confirmation`,
		`${modulePrefix}internal/domain`, `${modulePrefix}internal/store`,
	],
	"internal/contractmaterialize/materialize.go": [
		"bytes", "context", "crypto/rand", "crypto/sha256", "encoding/hex", "errors", "fmt", "io", "os", "path/filepath",
		"strings", "unicode", `${modulePrefix}internal/domain`, `${modulePrefix}internal/emit/node`,
		`${modulePrefix}internal/emit/node/model`, `${modulePrefix}internal/store`,
	],
	"internal/contractmaterialize/publish_darwin.go": ["C", "errors", "os", "path/filepath", "strings", "syscall", "unsafe"],
	"internal/contractmaterialize/publish_unsupported.go": ["errors", "os"],
});

const exactAPIs = Object.freeze({
	"internal/emit/node/internal/publication/authority.go": Object.freeze({
		functions: ["Issue"], types: ["Authority"],
		methods: [
			"Authority.CanonicalBytes", "Authority.Choicepoint", "Authority.Digest", "Authority.Equal",
			"Authority.ExpectedHeadDigest", "Authority.LineageRoot", "Authority.Predecessor", "Authority.StudyID", "Authority.Valid",
		],
		fields: Object.freeze({ Authority: ["study", "expectedHead", "digest", "predecessor", "choicepoint", "lineageRoot", "canonical", "seal"] }),
	}),
	"internal/emit/node/publication.go": Object.freeze({
		functions: ["OpenResidue", "PublishPrepared", "ReopenResidue"],
		types: ["PublicationDisposition", "PublicationResult", "Residue"],
		methods: [
			"Residue.Bundle", "Residue.BundleDigest", "Residue.ChoicepointDigest", "Residue.DecisionRecordDigest",
			"Residue.HeadDigest", "Residue.StudyID", "Residue.Valid",
		],
		fields: Object.freeze({
			Residue: ["objectStore", "head", "object", "authority", "bundle", "predecessor", "seal"],
			PublicationResult: ["Residue", "Disposition"],
		}),
	}),
	"internal/choice/promotion/residue.go": Object.freeze({
		functions: ["OpenResiduePredecessor"], types: ["ResiduePredecessorSnapshot"],
		methods: [
			"PortableRulingPreparation.StudyID", "ResiduePredecessorSnapshot.ChoicepointDigest",
			"ResiduePredecessorSnapshot.ConfirmationDigest", "ResiduePredecessorSnapshot.DecisionDigest",
			"ResiduePredecessorSnapshot.StudyID", "ResiduePredecessorSnapshot.Valid",
		],
		fields: Object.freeze({ ResiduePredecessorSnapshot: ["study", "decision", "choicepoint", "confirmation", "seal"] }),
	}),
	"internal/contractmaterialize/materialize.go": Object.freeze({
		functions: ["IsCode", "Materialize"], types: ["Disposition", "Error", "MaterializedContract"],
		methods: [
			"Error.Error", "Error.Unwrap", "MaterializedContract.BundleDigest", "MaterializedContract.Destination",
			"MaterializedContract.Disposition", "MaterializedContract.ResidueHeadDigest", "MaterializedContract.State",
			"MaterializedContract.String", "MaterializedContract.Valid",
		],
		fields: Object.freeze({
			Error: ["Code", "Detail", "Cause"],
			MaterializedContract: ["destination", "bundleDigest", "headDigest", "disposition", "seal"],
		}),
	}),
});

const requiredTests = Object.freeze({
	"internal/emit/node/internal/publication/authority_test.go": [
		"TestAuthorityBindsEveryTerminalPublicationJoin", "TestIssueRejectsIncompleteTerminalPublicationJoins",
	],
	"internal/emit/node/publication_store_darwin_test.go": [
		"TestObjectStoreAdvanceResidueDirectClosure", "TestObjectStoreConfirmResiduePublicationDirectClosure",
	],
	"internal/contractmaterialize/materialize_darwin_test.go": [
		"TestCancellationRefusesBeforeRenameAndReconcilesAfterRename",
		"TestDestinationAppearanceAtExclusiveBoundaryConvergesOrRefusesImmutably",
		"TestDescriptorRelativeSixFilePublicationIsExclusiveAndExact",
		"TestDestinationSyntaxParentTrustAndNamedKindRefusals",
		"TestDestinationFilesystemNameAliasIsRefused",
		"TestDirectoryRosterCloseFailureRefusesExactness",
		"TestExactDirectoryRefusesEveryPhysicalMemberMismatch",
		"TestExactPhysicalModesRejectSpecialBits",
		"TestExactWriterHandlesShortWritesAndRefusesEveryFailurePhase",
		"TestExclusiveRenameFailureDistinguishesProvenNoEffectFromUnknownEffect",
		"TestExistingOutputObservationFailureIsAmbiguousAndRetryable",
		"TestFinalMemberReopenRereadsSameInodeBytes",
		"TestParentDescriptorWalkRefusesIntermediateSymlinkReplacement",
		"TestPartialPrivateStageCleanupIsExactAndAnchored",
		"TestPostRenameAmbiguityReturnsZeroReceiptAndRetriesExactly",
		"TestRetainedParentDescriptorCannotBeRedirectedByPathReplacement",
		"TestRetainedParentRefusesOwnershipModeTrustDrift",
		"TestStageAllocationCleansRetainedFaultsOrReportsUnknownOwnership",
	],
	"internal/store/object_store_test.go": ["TestExternalPublicationPathCannotOverlapOwnedStoreNamespace"],
	"internal/store/public_api_test.go": ["TestStudyHeadExportedTransitionsAreTypedAndClosed"],
	"internal/store/head_test.go": ["TestStudyHeadCASBindsEveryExpectedTokenFieldBeforePublication"],
	"testkit/studies/cli_precedence/reduction_darwin_test.go": [
		"TestCLIP07BBPublicationHelper", "TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence",
	],
	"testkit/studies/http_invoices/reduction_darwin_test.go": ["TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence"],
});

class ArchitectureError extends Error {
	constructor(code, detail) {
		super(`${code}: ${detail}`);
		this.code = code;
	}
}

function slash(value) { return value.split(sep).join("/"); }
function sha256(bytes) { return createHash("sha256").update(bytes).digest("hex"); }
function sorted(values) { return [...values].sort(); }
function exact(left, right) { return JSON.stringify(sorted(left)) === JSON.stringify(sorted(right)); }
function count(source, expression) { return source.match(expression)?.length ?? 0; }

async function readReviewedFile(relativePath) {
	const absolute = resolve(repositoryRoot, relativePath);
	const fromRoot = relative(repositoryRoot, absolute);
	if (isAbsolute(fromRoot) || fromRoot === ".." || fromRoot.startsWith(`..${sep}`)) {
		throw new ArchitectureError("P07B_B_PATH_ESCAPE", relativePath);
	}
	const before = await lstat(absolute);
	if (!before.isFile() || before.isSymbolicLink()) throw new ArchitectureError("P07B_B_NONREGULAR_FILE", relativePath);
	let handle;
	try {
		handle = await open(absolute, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
		const opened = await handle.stat();
		if (!opened.isFile() || opened.dev !== before.dev || opened.ino !== before.ino || opened.size !== before.size) {
			throw new ArchitectureError("P07B_B_FILE_CHANGED", relativePath);
		}
		const bytes = await handle.readFile();
		const after = await handle.stat();
		if (after.dev !== opened.dev || after.ino !== opened.ino || after.size !== opened.size) {
			throw new ArchitectureError("P07B_B_FILE_CHANGED", relativePath);
		}
		let source;
		try {
			source = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
		} catch {
			throw new ArchitectureError("P07B_B_INVALID_UTF8", relativePath);
		}
		return Object.freeze({ path: relativePath, source, digest: sha256(bytes) });
	} finally {
		await handle?.close();
	}
}

async function exactDirectory(path, files, directories = []) {
	const entries = await readdir(resolve(repositoryRoot, path), { withFileTypes: true });
	const expected = sorted([...files, ...directories]);
	const actual = sorted(entries.map((entry) => entry.name));
	if (!exact(actual, expected)) throw new ArchitectureError("P07B_B_TOPOLOGY_DRIFT", `${path}:${actual.join(",")}`);
	for (const entry of entries) {
		if (entry.isSymbolicLink()) throw new ArchitectureError("P07B_B_TOPOLOGY_SYMLINK", `${path}/${entry.name}`);
		const wantsDirectory = directories.includes(entry.name);
		if (wantsDirectory !== entry.isDirectory() || (!wantsDirectory && !entry.isFile())) {
			throw new ArchitectureError("P07B_B_TOPOLOGY_KIND", `${path}/${entry.name}`);
		}
	}
}

async function inspectTopology() {
	await exactDirectory("internal/contractmaterialize", [
		"materialize.go", "materialize_darwin_test.go", "publish_darwin.go", "publish_unsupported.go",
	]);
	await exactDirectory("internal/emit/node/authority", ["authority.go"]);
	await exactDirectory("internal/emit/node/internal/publication", ["authority.go", "authority_test.go"]);
	const promotion = await readdir(resolve(repositoryRoot, "internal/choice/promotion"), { withFileTypes: true });
	const production = sorted(promotion.filter((entry) => entry.name.endsWith(".go") && !entry.name.endsWith("_test.go"))
		.map((entry) => entry.name));
	if (!exact(production, ["residue.go", "service.go"])) {
		throw new ArchitectureError("P07B_B_PROMOTION_MAP", production.join(","));
	}
	for (const absent of [
		"internal/contractexecution", "internal/contracttarget", "internal/finalizedcontractrun", "internal/emit/node/execution",
	]) {
		try {
			await lstat(resolve(repositoryRoot, absent));
			throw new ArchitectureError("P07B_B_PREMATURE_C_PACKAGE", absent);
		} catch (error) {
			if (error instanceof ArchitectureError) throw error;
			if (error?.code !== "ENOENT") throw error;
		}
	}
}

async function walkProduction(path, output) {
	const entries = await readdir(resolve(repositoryRoot, path), { withFileTypes: true });
	for (const entry of entries) {
		const relativePath = slash(join(path, entry.name));
		if (entry.isSymbolicLink()) throw new ArchitectureError("P07B_B_PRODUCTION_SYMLINK", relativePath);
		if (entry.isDirectory()) await walkProduction(relativePath, output);
		else if (entry.isFile() && entry.name.endsWith(".go") && !entry.name.endsWith("_test.go")) output.push(relativePath);
	}
}

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

function structFields(source, name) {
	const body = balancedBody(source, new RegExp(`^type\\s+${name}\\s+struct\\s*`, "mu"));
	return body.split(/\r?\n/u).map((line) => line.trim()).filter(Boolean)
		.map((line) => /^([A-Za-z_][A-Za-z0-9_]*)\b/u.exec(line)?.[1]).filter(Boolean);
}

function exportedFunctions(source) {
	return sorted([...source.matchAll(/^func\s+([A-Z][A-Za-z0-9_]*)\s*\(/gmu)].map((match) => match[1]));
}

function exportedTypes(source) {
	return sorted([...source.matchAll(/^type\s+([A-Z][A-Za-z0-9_]*)\b/gmu)].map((match) => match[1]));
}

function exportedMethods(source) {
	const methods = [];
	for (const match of source.matchAll(/^func\s+\(([^)]*)\)\s+([A-Z][A-Za-z0-9_]*)\s*\(/gmu)) {
		const receiver = /\*?([A-Za-z_][A-Za-z0-9_]*)\s*$/u.exec(match[1])?.[1];
		if (receiver && /^[A-Z]/u.test(receiver)) methods.push(`${receiver}.${match[2]}`);
	}
	return sorted(methods);
}

function functionHeader(source, name) {
	const match = new RegExp(`^func\\s+(?:\\([^)]*\\)\\s+)?${name}\\s*\\(`, "mu").exec(source);
	if (!match) return "";
	const body = source.indexOf("{", match.index);
	return body < 0 ? "" : source.slice(match.index, body).replace(/\s+/gu, " ").trim();
}

function ordered(body, anchors) {
	let cursor = -1;
	for (const anchor of anchors) {
		cursor = body.indexOf(anchor, cursor + 1);
		if (cursor < 0) return false;
	}
	return true;
}

function runInheritedA2() {
	const environment = {};
	for (const name of ["HOME", "TMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE", "COUNTERSHAPE_GO", "COUNTERSHAPE_CC", "COUNTERSHAPE_CXX"]) {
		const value = process.env[name];
		if (!value || !isAbsolute(value)) throw new ArchitectureError("P07B_B_ENVIRONMENT_REQUIRED", name);
		environment[name] = value;
	}
	Object.assign(environment, {
		PATH: `${dirname(process.execPath)}:/usr/bin:/bin`, LANG: "C", LC_ALL: "C", TZ: "UTC", NO_COLOR: "1",
		GOENV: "off", GOWORK: "off", GOTOOLCHAIN: "local", GOPROXY: "off", GOSUMDB: "off",
		GOFLAGS: "-mod=readonly -buildvcs=false -p=1", CGO_ENABLED: "0", GOMAXPROCS: "2",
	});
	const result = spawnSync(process.execPath, [resolve(repositoryRoot, "tools/check-p07b-a2-architecture.mjs")], {
		cwd: repositoryRoot, encoding: "utf8", timeout: 120_000, maxBuffer: 16 * 1024 * 1024, env: environment,
	});
	if (result.error || result.signal || result.status !== 0 || !result.stdout.includes("P07B A2.2 architecture boundary OK")) {
		throw new ArchitectureError("P07B_B_INHERITED_A2_FAILED", `${result.status ?? result.signal}: ${result.stderr || result.stdout}`);
	}
}

export async function collectFacts() {
	await inspectTopology();
	const entries = await Promise.all(reviewedFiles.map(readReviewedFile));
	const byPath = new Map(entries.map((entry) => [entry.path, entry]));
	const productionPaths = [];
	await walkProduction("internal", productionPaths);
	const unreviewed = productionPaths.filter((path) => !byPath.has(path));
	const productionEntries = [...entries.filter((entry) => productionPaths.includes(entry.path))];
	for (const path of unreviewed) productionEntries.push(await readReviewedFile(path));
	const importerPaths = (importPath) => sorted(productionEntries.filter((entry) => goImports(entry.source).includes(importPath))
		.map((entry) => entry.path));
	const sources = Object.fromEntries(entries.map((entry) => [entry.path, entry.source]));
	const head = sources["internal/store/head.go"];
	const nodePublication = sources["internal/emit/node/publication.go"];
	const materialize = sources["internal/contractmaterialize/materialize.go"];
	const darwin = sources["internal/contractmaterialize/publish_darwin.go"];
	const unsupported = sources["internal/contractmaterialize/publish_unsupported.go"];
	const service = sources["internal/choice/promotion/service.go"];
	const objectStore = sources["internal/store/object_store.go"];
	const materializeHeader = functionHeader(materialize, "Materialize");
	const rawOutputAPI = /(?:ContractBundle|\[\]byte|domain\.Digest|interface\s*\{|func\s*\()/u.test(materializeHeader);
	const sourceFactory = functionBody(materialize, "sourceFromResidue");
	const authorizedCore = functionBody(materialize, "materializeAuthorized");
	const existingAcceptance = functionBody(materialize, "acceptExistingOpen");
	const alreadyReopenedAcceptance = functionBody(materialize, "acceptAlreadyReopened");
	const futureSymbols = productionEntries.flatMap((entry) =>
		["ContractExecutionTarget", "FinalizedContractRun", "ContractExecution"].filter((symbol) => entry.source.includes(symbol))
			.map((symbol) => `${entry.path}:${symbol}`));
	const issueCallers = productionEntries.filter((entry) => entry.source.includes("internalpublication.Issue("))
		.map((entry) => entry.path);
	const api = {};
	for (const [path, specification] of Object.entries(exactAPIs)) {
		const source = sources[path];
		api[path] = {
			functions: exportedFunctions(source), types: exportedTypes(source), methods: exportedMethods(source), fields: {},
		};
		for (const name of Object.keys(specification.fields)) api[path].fields[name] = structFields(source, name);
	}
	const tests = {};
	for (const [path] of Object.entries(requiredTests)) tests[path] = exportedFunctions(sources[path]);
	return structuredClone({
		digests: Object.fromEntries(entries.map((entry) => [entry.path, entry.digest])),
		imports: Object.fromEntries(Object.keys(exactImports).map((path) => [path, goImports(sources[path])])),
		api,
		issuerImporters: importerPaths(`${modulePrefix}internal/emit/node/internal/publication`),
		aliasImporters: importerPaths(`${modulePrefix}internal/emit/node/authority`),
		issueCallers: sorted(issueCallers),
		issueCallCount: count(nodePublication, /\binternalpublication\.Issue\s*\(/gu),
		rawOutputAPI,
		privateMaterializationCore: functionBody(materialize, "materializeAuthorized") !== "" &&
			!materialize.includes("func MaterializeAuthorized") && !materialize.includes("type MaterializationSource"),
		materializationSourceFields: structFields(materialize, "materializationSource"),
		materializationSourceLiteralCount: count(materialize, /\bmaterializationSource\s*\{/gu),
		sourceFactoryLiteralCount: count(sourceFactory, /\bmaterializationSource\s*\{/gu),
		sourceBindingShape: ordered(sourceFactory, [
			"snapshotFromResidue", "materializationSource{", "node.ReopenResidue", "ValidateExternalPublicationPath",
		]),
		advanceResidueRawCalls: count(functionBody(head, "AdvanceResidue"), /\badvanceHead\s*\(/gu),
		advanceResidueExactStage: functionBody(head, "AdvanceResidue").includes("return s.advanceHead(ctx, expected, StageResidue, object)"),
		confirmRawMutation: /\b(?:advanceHead|publishLocked|replaceHead|NewSemanticObject)\s*\(/u.test(functionBody(head, "ConfirmResiduePublication")),
		confirmReopenCount: count(functionBody(head, "ConfirmResiduePublication"), /\breopenHeadAndObject\s*\(/gu),
		confirmDurabilityShape: ordered(functionBody(head, "ConfirmResiduePublication"), [
			"s.instance.mu.Lock()", "openAndLockStudy", "reopenHeadAndObject", "syncDirectory", "reopenHeadAndObject",
		]),
		publishOrder: ordered(functionBody(nodePublication, "PublishPrepared"), [
			"ValidatePortableRulingPredecessor", "OpenHead", "StageResidue", "ValidatePortableRulingPreparation",
			"internalpublication.Issue", "AdvanceResidue", "reopenPublishedResult",
		]),
		openResidueShape: ordered(functionBody(nodePublication, "OpenResidue"), [
			"OpenHead", "objectStore.Read", "ParseContractBundle", "PreviousHeadDigest", "internalpublication.Issue",
			"ConfirmResiduePublication", "OpenResiduePredecessor",
		]),
		predecessorValidationShape: ordered(functionBody(service, "ValidatePortableRulingPredecessor"), [
			"preparation.Valid()", "validateRulingPredecessor", "choice.InspectPortableRuling",
		]),
		physicalOverlapShape: ordered(functionBody(objectStore, "ValidateExternalPublicationPath"), [
			"s.assertReady()", "physicalPathOverlapsRoot", "filesystemPathsOverlap", "s.assertReady()",
		]),
		reopenCallCount: count(materialize, /\bnode\.ReopenResidue\s*\(/gu),
		materializeEntryReopenCount: count(functionBody(materialize, "Materialize"), /\bnode\.ReopenResidue\s*\(/gu),
		sourceFactoryReopenCount: count(sourceFactory, /\bnode\.ReopenResidue\s*\(/gu),
		materializeCreateReopenCount: count(authorizedCore, /\bsource\.reopen\s*\(/gu),
		materializeExistingReopenCount: count(existingAcceptance, /\bsource\.reopen\s*\(/gu),
		materializeAuthorizedCount: count(materialize, /\bmaterializeAuthorized\s*\(/gu),
		materializeAuthorizedCallerCount: count(functionBody(materialize, "materializeReopened"), /\bmaterializeAuthorized\s*\(/gu),
		existingMutationEdge: /\b(?:mkdirAt|unlinkAt|renameExclusiveAt|writeExactBundle)\s*\(/u.test(existingAcceptance),
		existingObservationShape: ordered(existingAcceptance, ["isObservationUncertain", "ambiguous(", "incomplete("]) &&
			ordered(alreadyReopenedAcceptance, ["ambiguous(\"existing destination identity", "isObservationUncertain", "ambiguous(", "incomplete("]) &&
			count(materialize, /\bobservationUncertain\s*\(/gu) === 9,
		exactReadShape: count(functionBody(materialize, "verifyExactFileAt"), /\breadExactFile\s*\(/gu) === 2 &&
			ordered(functionBody(materialize, "readExactFile"), ["Stat()", "io.ReadAll", "Stat()", "Sync()", "Close()", "sha256.Sum256", "bytes.Equal"]),
		exactRosterReads: count(functionBody(materialize, "verifyExactDirectoryHandle"), /\breadExactRoster\s*\(/gu),
		componentWalk: ordered(functionBody(darwin, "openDirectoryPathNoFollow"), [
			"syscall.Open", "strings.Split", "openDirectoryAtNoFollow", "current.Close()",
		]) && !materialize.includes("filepath.EvalSymlinks"),
		caseAliasFullScan: functionBody(materialize, "scanDirectoryForExactName").includes("strings.EqualFold") &&
			!functionBody(materialize, "scanDirectoryForExactName").includes("if entry.Name() == name {\n\t\t\t\treturn"),
		nativeTag: darwin.startsWith("//go:build darwin && arm64 && cgo\n"),
		unsupportedTag: unsupported.startsWith("//go:build !darwin || !arm64 || !cgo\n"),
		nativeExclusiveFlags: darwin.includes("RENAME_EXCL | RENAME_NOFOLLOW_ANY"),
		nativeMemberOpenFlags: ["O_NOFOLLOW", "O_NONBLOCK", "O_CLOEXEC"].every((flag) =>
			functionBody(darwin, "openFileAtNoFollow").includes(flag)),
		parentTrustPolicy: ["stat.Uid == uint32(os.Geteuid())", "info.Mode().Perm()&0o700 == 0o700", "info.Mode().Perm()&0o022 == 0"]
			.every((anchor) => functionBody(darwin, "trustedPublicationParent").includes(anchor)) &&
			ordered(functionBody(materialize, "validateParentPathIdentity"), [
				"os.SameFile", "hasSpecialMode", "trustedPublicationParent",
			]) && count(materialize, /\bownedByEffectiveUser\s*\(/gu) === 4,
		unsupportedTrustRefusal: functionBody(unsupported, "trustedPublicationParent").includes("return false") &&
			functionBody(unsupported, "ownedByEffectiveUser").includes("return false"),
		renameReconciliationShape: ordered(authorizedCore, [
			"renameExclusiveAt", "openExactNamedDirectory", "acceptAlreadyReopened", "retainedStageStillNamed", "ambiguous(",
		]),
		nativeFallbackPresent: /\bos\.Rename\b|\brename\s*\(/u.test(darwin) || /\bos\.Rename\b/u.test(unsupported),
		forbiddenMaterializerImport: productionEntries
			.filter((entry) => entry.path.startsWith("internal/contractmaterialize/") && !entry.path.endsWith("_test.go"))
			.some((entry) => goImports(entry.source).some((imported) =>
				imported === "os/exec" || imported === "runtime" || imported === "plugin" || imported === "net" || imported.startsWith("net/") ||
				imported === `${modulePrefix}internal/gitobj` || imported.startsWith(`${modulePrefix}internal/git`) ||
				imported.startsWith(`${modulePrefix}internal/server`) || imported.startsWith(`${modulePrefix}internal/studio`))),
		futureSymbols,
		tests,
		cliEvidence: [
			"cancelled CLI terminal publication changed durable state", "runCLIP07BBPublicationRace",
			"runCLIP07BBFreshProcessOpen", "assertCLIP07BBMissingPredecessorsRefuse",
		].every((anchor) => sources["testkit/studies/cli_precedence/reduction_darwin_test.go"].includes(anchor)),
		httpEvidence: [
			"concurrent HTTP materialization did not converge", "PublicationCreated", "PublicationAlreadyCurrent",
			"StateAlreadyExact",
		].every((anchor) => sources["testkit/studies/http_invoices/reduction_darwin_test.go"].includes(anchor)),
	});
}

export function validateFacts(facts) {
	const violations = [];
	const add = (condition, code, detail) => { if (condition) violations.push(Object.freeze({ code, detail })); };
	for (const [path, digest] of Object.entries(pinnedProductionDigests)) {
		add(facts.digests[path] !== digest, "P07B_B_PRODUCTION_DIGEST_DRIFT", `${path}:${facts.digests[path]}`);
	}
	for (const [path, expected] of Object.entries(exactImports)) {
		add(!exact(facts.imports[path], expected), "P07B_B_IMPORT_ROSTER", `${path}:${facts.imports[path]?.join(",")}`);
	}
	for (const [path, expected] of Object.entries(exactAPIs)) {
		const actual = facts.api[path];
		add(!exact(actual.functions, expected.functions), "P07B_B_EXPORTED_FUNCTIONS", path);
		add(!exact(actual.types, expected.types), "P07B_B_EXPORTED_TYPES", path);
		add(!exact(actual.methods, expected.methods), "P07B_B_EXPORTED_METHODS", path);
		for (const [name, fields] of Object.entries(expected.fields)) {
			add(JSON.stringify(actual.fields[name]) !== JSON.stringify(fields), "P07B_B_STRUCT_FIELDS", `${path}:${name}`);
		}
	}
	add(!exact(facts.issuerImporters, [
		"internal/emit/node/authority/authority.go", "internal/emit/node/publication.go",
	]), "P07B_B_ISSUER_OWNERSHIP", facts.issuerImporters.join(","));
	add(!exact(facts.aliasImporters, ["internal/store/head.go"]), "P07B_B_ALIAS_OWNERSHIP", facts.aliasImporters.join(","));
	add(!exact(facts.issueCallers, ["internal/emit/node/publication.go"]) || facts.issueCallCount !== 2,
		"P07B_B_ISSUANCE_SURFACE", `${facts.issueCallers.join(",")}:${facts.issueCallCount}`);
	add(facts.rawOutputAPI, "P07B_B_RAW_OUTPUT_API", "Materialize accepts raw bundle authority");
	add(!facts.privateMaterializationCore || !exact(facts.materializationSourceFields, ["snapshot", "revalidate", "validatePath"]) ||
		facts.materializationSourceLiteralCount !== 2 || facts.sourceFactoryLiteralCount !== 2 || !facts.sourceBindingShape,
		"P07B_B_PRIVATE_MATERIALIZATION_SOURCE", facts.materializationSourceFields.join(","));
	add(facts.advanceResidueRawCalls !== 1 || !facts.advanceResidueExactStage,
		"P07B_B_RAW_TRANSITION_CARDINALITY", `${facts.advanceResidueRawCalls}`);
	add(facts.confirmRawMutation || facts.confirmReopenCount !== 2 || !facts.confirmDurabilityShape,
		"P07B_B_CONFIRMATION_MUTATION", `${facts.confirmReopenCount}`);
	add(!facts.publishOrder, "P07B_B_PUBLICATION_ORDER", "PublishPrepared");
	add(!facts.openResidueShape, "P07B_B_RESIDUE_RECONSTRUCTION", "OpenResidue");
	add(!facts.predecessorValidationShape, "P07B_B_PREDECESSOR_VALIDATION", "ValidatePortableRulingPredecessor");
	add(!facts.physicalOverlapShape, "P07B_B_STORE_OVERLAP_IDENTITY", "ValidateExternalPublicationPath");
	add(facts.reopenCallCount !== 2 || facts.materializeEntryReopenCount !== 1 || facts.sourceFactoryReopenCount !== 1 ||
		facts.materializeCreateReopenCount !== 1 || facts.materializeExistingReopenCount !== 1 ||
		facts.materializeAuthorizedCount !== 2 || facts.materializeAuthorizedCallerCount !== 1,
		"P07B_B_FRESH_REOPEN_PATHS",
		`${facts.reopenCallCount}/${facts.materializeEntryReopenCount}/${facts.sourceFactoryReopenCount}/${facts.materializeCreateReopenCount}/${facts.materializeExistingReopenCount}`);
	add(facts.existingMutationEdge, "P07B_B_EXISTING_OUTPUT_MUTATION", "acceptExistingOpen");
	add(!facts.existingObservationShape, "P07B_B_EXISTING_OBSERVATION_CLASSIFICATION", "ambiguous observation versus physical mismatch");
	add(!facts.exactReadShape || facts.exactRosterReads !== 2, "P07B_B_FINAL_EXACT_OBSERVATION", `${facts.exactRosterReads}`);
	add(!facts.componentWalk, "P07B_B_PARENT_COMPONENT_WALK", "openDirectoryPathNoFollow");
	add(!facts.caseAliasFullScan, "P07B_B_CASE_ALIAS_SCAN", "scanDirectoryForExactName");
	add(!facts.nativeTag || !facts.unsupportedTag || !facts.nativeExclusiveFlags || !facts.nativeMemberOpenFlags ||
		facts.nativeFallbackPresent,
		"P07B_B_NATIVE_PUBLICATION_POLICY", "Darwin exclusive no-follow policy");
	add(!facts.parentTrustPolicy || !facts.unsupportedTrustRefusal,
		"P07B_B_PARENT_TRUST_POLICY", "effective-user ownership and repeated parent trust");
	add(!facts.renameReconciliationShape, "P07B_B_RENAME_RECONCILIATION", "exact destination, retained stage, or ambiguity");
	add(facts.forbiddenMaterializerImport, "P07B_B_MATERIALIZER_CAPABILITY", "forbidden production import");
	add(facts.futureSymbols.length !== 0, "P07B_B_PREMATURE_C_SURFACE", facts.futureSymbols.join(","));
	for (const [path, expected] of Object.entries(requiredTests)) {
		add(expected.some((name) => !facts.tests[path].includes(name)), "P07B_B_REQUIRED_TEST_MISSING",
			`${path}:${expected.filter((name) => !facts.tests[path].includes(name)).join(",")}`);
	}
	add(!facts.cliEvidence, "P07B_B_CLI_EVIDENCE_SURFACE", "CLI publication/restart/refusal anchors");
	add(!facts.httpEvidence, "P07B_B_HTTP_EVIDENCE_SURFACE", "HTTP publication/materialization convergence anchors");
	return Object.freeze(violations);
}

async function main() {
	if (process.argv.length !== 2) throw new ArchitectureError("P07B_B_ARGUMENTS", "no arguments accepted");
	runInheritedA2();
	const facts = await collectFacts();
	const violations = validateFacts(facts);
	if (violations.length > 0) {
		throw new ArchitectureError("P07B_B_ARCHITECTURE_VIOLATION",
			violations.map((violation) => `${violation.code}: ${violation.detail}`).join("\n"));
	}
	process.stdout.write("P07B B architecture boundary OK\n");
}

if (process.argv[1] && resolve(process.argv[1]) === checkerPath) {
	main().catch((error) => {
		process.stderr.write(`${error.stack ?? error}\n`);
		process.exitCode = 1;
	});
}
