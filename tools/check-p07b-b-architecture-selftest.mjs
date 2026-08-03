#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync, symlinkSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import {
	collectFacts,
	futureSurfaceHistoricalInventory,
	futureSurfaceManifest,
	futureSurfaceManifestBytes,
	futureSymbolsInSource,
	inheritedC1Arguments,
	partitionFutureSymbols,
	runInheritedC1,
	validateFacts,
	validateFutureSurfaceManifest,
	validateFutureSurfaceState,
} from "./check-p07b-b-architecture.mjs";

const selftestPath = fileURLToPath(import.meta.url);
const root = resolve(dirname(selftestPath), "..");
const checker = resolve(root, "tools/check-p07b-b-architecture.mjs");
const cleanMarker = "P07B B architecture boundary OK\n";

const historicalCases = Object.freeze([
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
	Object.freeze({ id: "c3-surface-bundle", code: "P07B_B_SELFTEST_C3_SURFACE_BUNDLE" }),
	Object.freeze({ id: "future-surface", code: "P07B_B_PREMATURE_C_SURFACE" }),
]);
const cases = Object.freeze([
	...historicalCases,
	Object.freeze({ id: "inherited-c1-explicit", code: "P07B_B_SELFTEST_INHERITED_C1_EXPLICIT" }),
	Object.freeze({ id: "future-manifest-authority", code: "P07B_B_SELFTEST_FUTURE_MANIFEST_AUTHORITY" }),
	Object.freeze({ id: "future-state-machine", code: "P07B_B_SELFTEST_FUTURE_STATE_MACHINE" }),
]);
const expectedHistoricalRosterDigest = "d5745ef69f205bbe4ce445448e6189ba244adacc36fe3e44fdf56f9485100ef2";
const expectedRosterDigest = "8dec151002dd727fd327ec5d46b4e0fdcf060324bda52859d82589948527db4a";

function rosterDigest(roster = cases) {
	const hash = createHash("sha256");
	for (const entry of roster) hash.update(entry.id).update("\0").update(entry.code).update("\0");
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

function requireExplicitInheritedC1() {
	let observed;
	const result = runInheritedC1((executable, args, options) => {
		observed = { executable, args, options };
		return {
			status: 0,
			signal: null,
			error: undefined,
			stdout: "P07B-C C1 architecture boundary OK\n",
			stderr: "",
		};
	});
	if (result !== true || observed?.executable !== process.execPath ||
		JSON.stringify(observed?.args) !== JSON.stringify([checker.replace("check-p07b-b-architecture.mjs", "check-p07b-c-architecture.mjs"), ...inheritedC1Arguments]) ||
		JSON.stringify(inheritedC1Arguments) !== JSON.stringify(["--c1"]) || !Object.isFrozen(inheritedC1Arguments) ||
		observed?.options?.cwd !== root || observed.options.encoding !== "utf8" || observed.options.env !== process.env ||
		observed.options.timeout !== 180_000 || observed.options.maxBuffer !== 32 * 1024 * 1024) {
		fail("P07B_B_SELFTEST_INHERITED_C1_EXPLICIT", JSON.stringify({
			executable: observed?.executable,
			args: observed?.args,
			cwd: observed?.options?.cwd,
		}));
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
	const c3Official = [
		"internal/contractexec/target.go:ContractExecutionTarget",
		"internal/contractexec/target.go:NewContractExecutionTarget",
		"internal/contractexec/target.go:ParseContractExecutionTarget",
		"internal/store/nonhead_contract.go:ParseContractExecutionTarget",
	];
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
		"internal/contractexec/target_copy.go:NewContractExecutionTarget",
		"internal/contractexec/target_copy.go:ParseContractExecutionTarget",
		"internal/contractexec/target.go:NewContractExecutionTargetCopy",
		"internal/contractexec/target.go:ParseContractExecutionTargetCopy",
		"internal/store/object_store.go:ParseContractExecutionTarget",
		"internal/store/nonhead_contract.go:ParseContractExecutionTargetCopy",
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
	const partition = partitionFutureSymbols([admitted, ...c2Storage, c2Support, ...c3Official, modelEvil, ...c2Lookalikes, ...c3Lookalikes, ...semanticFamilyLookalikes, foreign]);
	if (JSON.stringify(partition.admitted) !== JSON.stringify([admitted, ...c2Storage, c2Support, ...c3Official]) ||
		JSON.stringify(partition.foreign) !== JSON.stringify([modelEvil, ...c2Lookalikes, ...c3Lookalikes, ...semanticFamilyLookalikes, foreign])) {
		fail("P07B_B_SELFTEST_PREFIX_PARTITION", JSON.stringify(partition));
	}
	const scannerControls = [
		["", []],
		["type ContractExecutionTarget struct{}", [c3Official[0]]],
		["func NewContractExecutionTarget() {}", [c3Official[1]]],
		["func ParseContractExecutionTarget() {}", [c3Official[2]]],
		["type ContractExecutionTargetCopy struct{}", ["internal/contractexec/target.go:ContractExecutionTargetCopy"]],
		["type CopyContractExecutionTarget struct{}", ["internal/contractexec/target.go:CopyContractExecutionTarget"]],
		["type ContractExecutionTargetβ struct{}", ["internal/contractexec/target.go:ContractExecutionTargetβ"]],
		["func NewContractExecutionTargetCopy() {}", ["internal/contractexec/target.go:NewContractExecutionTargetCopy"]],
		["func ParseContractExecutionTargetCopy() {}", ["internal/contractexec/target.go:ParseContractExecutionTargetCopy"]],
		["func ParseContractExecutionTargetβ() {}", ["internal/contractexec/target.go:ParseContractExecutionTargetβ"]],
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
	const storeParserObserved = futureSymbolsInSource(
		"internal/store/nonhead_contract.go",
		"func reopen() { contractmodel.ParseContractExecutionTarget() }",
	);
	if (JSON.stringify(storeParserObserved) !== JSON.stringify([c3Official[3]])) {
		fail("P07B_B_SELFTEST_SYMBOL_SCANNER", JSON.stringify({ storeParserObserved }));
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

function requireExactC3SurfaceBundle() {
	const exact = [
		"internal/contractexec/target.go:ContractExecutionTarget",
		"internal/contractexec/target.go:NewContractExecutionTarget",
		"internal/contractexec/target.go:ParseContractExecutionTarget",
		"internal/store/nonhead_contract.go:ParseContractExecutionTarget",
	];
	const assertRejected = (name, values) => {
		const partition = partitionFutureSymbols(values);
		if (partition.admitted.length !== 0 || JSON.stringify(partition.foreign) !== JSON.stringify(values)) {
			fail("P07B_B_SELFTEST_C3_SURFACE_BUNDLE", `${name}:${JSON.stringify(partition)}`);
		}
	};
	const absent = partitionFutureSymbols([]);
	if (absent.admitted.length !== 0 || absent.foreign.length !== 0) {
		fail("P07B_B_SELFTEST_C3_SURFACE_BUNDLE", `historical:${JSON.stringify(absent)}`);
	}
	const complete = partitionFutureSymbols(exact);
	if (JSON.stringify(complete.admitted) !== JSON.stringify(exact) || complete.foreign.length !== 0) {
		fail("P07B_B_SELFTEST_C3_SURFACE_BUNDLE", `complete:${JSON.stringify(complete)}`);
	}
	const oneShot = partitionFutureSymbols((function* () { yield* exact; })());
	if (JSON.stringify(oneShot.admitted) !== JSON.stringify(exact) || oneShot.foreign.length !== 0) {
		fail("P07B_B_SELFTEST_C3_SURFACE_BUNDLE", `one-shot:${JSON.stringify(oneShot)}`);
	}
	for (let mask = 1; mask < (1 << exact.length) - 1; mask += 1) {
		assertRejected(`partial-${mask.toString(2).padStart(exact.length, "0")}`,
			exact.filter((_, index) => (mask & (1 << index)) !== 0));
	}
	for (const [name, index, replacement] of [
		["new-relocated", 1, "internal/contractexec/target_copy.go:NewContractExecutionTarget"],
		["target-parser-relocated", 2, "internal/contractexec/target_copy.go:ParseContractExecutionTarget"],
		["store-parser-relocated", 3, "internal/store/object_store.go:ParseContractExecutionTarget"],
		["new-renamed", 1, "internal/contractexec/target.go:NewContractExecutionTargetCopy"],
		["target-parser-renamed", 2, "internal/contractexec/target.go:ParseContractExecutionTargetCopy"],
		["store-parser-renamed", 3, "internal/store/nonhead_contract.go:ParseContractExecutionTargetβ"],
	]) {
		const candidate = [...exact];
		candidate[index] = replacement;
		assertRejected(name, candidate);
	}
	for (const extra of [
		"internal/contractexec/target.go:ParseFinalizedContractRun",
		"internal/future.go:ParseContractExecutionTarget",
	]) {
		const partition = partitionFutureSymbols([...exact, extra]);
		if (JSON.stringify(partition.admitted) !== JSON.stringify(exact) ||
			JSON.stringify(partition.foreign) !== JSON.stringify([extra])) {
			fail("P07B_B_SELFTEST_C3_SURFACE_BUNDLE", `extra:${JSON.stringify(partition)}`);
		}
	}
}

function requireFutureSurfaceStateMachine() {
	const c4Rows = [
		{ boundary: "C4", path: "internal/contractexec/runner/runner.go", symbol: "ContractExecutionRunner" },
		{ boundary: "C4", path: "internal/store/contract_run_bridge.go", symbol: "ContractExecutionRecord" },
	];
	const c5Rows = [
		{ boundary: "C5", path: "internal/contractexec/http/evidence.go", symbol: "ContractExecutionTarget" },
		{ boundary: "C5", path: "internal/contractexec/http/runner.go", symbol: "ContractExecutionRecord" },
		{ boundary: "C5", path: "internal/contractexec/http/runner_darwin.go", symbol: "ContractExecutionRecord" },
		{ boundary: "C5", path: "internal/contractexec/http/runner_darwin.go", symbol: "DeriveContractExecution" },
		{ boundary: "C5", path: "internal/contractexec/http/runner_darwin.go", symbol: "NewFinalizedContractRun" },
		{ boundary: "C5", path: "internal/contractexec/http/runner_darwin.go", symbol: "PersistContractExecutionRecord" },
		{ boundary: "C5", path: "internal/contractexec/http/runner_unsupported.go", symbol: "ContractExecutionRecord" },
	];
	const rows = [...c4Rows, ...c5Rows];
	const manifest = futureSurfaceManifest(rows);
	const compareObserved = (left, right) => Buffer.compare(
		Buffer.from(`${left.path}\0${left.symbol}`, "utf8"),
		Buffer.from(`${right.path}\0${right.symbol}`, "utf8"),
	);
	const observed = (future) => [...structuredClone(futureSurfaceHistoricalInventory), ...future]
		.sort(compareObserved);
	const production = (future) => [...new Set([
		...futureSurfaceHistoricalInventory.map(({ path }) => path), ...future.map(({ path }) => path),
	])].sort((left, right) => Buffer.compare(Buffer.from(left, "utf8"), Buffer.from(right, "utf8")));
	const c4Future = rows.filter(({ boundary }) => boundary === "C4").map(({ path, symbol }) => ({ path, symbol }));
	const c5Future = rows.map(({ path, symbol }) => ({ path, symbol }));
	const c4Observed = observed(c4Future);
	const c5Observed = observed(c5Future);
	const c4Production = production(c4Future);
	const c5Production = [...production(c5Future), "internal/contractexec/scope/scope.go"]
		.sort((left, right) => Buffer.compare(Buffer.from(left, "utf8"), Buffer.from(right, "utf8")));
	if (validateFutureSurfaceManifest(manifest).length !== 0 ||
		validateFutureSurfaceState(manifest, c4Observed, c4Production, "C4").length !== 0 ||
		validateFutureSurfaceState(manifest, c5Observed, c5Production, "C5").length !== 0 ||
		!futureSurfaceManifestBytes(manifest).equals(Buffer.from(`${JSON.stringify(manifest)}\n`, "utf8"))) {
		fail("P07B_B_SELFTEST_FUTURE_SURFACE_STATE", "positive states");
	}
	let rejected = 0;
	const refuseManifest = (name, mutate, expected) => {
		const hostile = structuredClone(manifest);
		const candidate = mutate(hostile) ?? hostile;
		const errors = validateFutureSurfaceManifest(candidate);
		if (!errors.some((error) => error.includes(expected))) {
			fail("P07B_B_SELFTEST_FUTURE_SURFACE_STATE", `${name}:${errors.join(",")}`);
		}
		rejected += 1;
	};
	refuseManifest("root order", (value) => ({ rows: value.rows, schema: value.schema, schema_sha256: value.schema_sha256 }), "root keys/order");
	refuseManifest("schema digest", (value) => { value.schema_sha256 = "0".repeat(64); }, "schema digest");
	refuseManifest("row order", (value) => { [value.rows[0], value.rows[1]] = [value.rows[1], value.rows[0]]; }, "row order");
	refuseManifest("duplicate", (value) => { value.rows.splice(1, 0, structuredClone(value.rows[0])); }, "duplicate row");
	refuseManifest("boundary", (value) => { value.rows[0].boundary = "C6"; }, "boundary");
	refuseManifest("prefix lookalike", (value) => { value.rows[0].path = "internal/contractexec/runner-copy/runner.go"; }, "location");
	refuseManifest("symbol", (value) => { value.rows[0].symbol = "ExecutionRunner"; }, "symbol");
	refuseManifest("missing bridge", (value) => {
		value.rows = value.rows.filter(({ path }) => path !== "internal/store/contract_run_bridge.go");
	}, "missing required row");
	const refuseState = (name, boundary, observedRows, productionPaths, expected) => {
		const errors = validateFutureSurfaceState(manifest, observedRows, productionPaths, boundary);
		if (!errors.some((error) => error.includes(expected))) {
			fail("P07B_B_SELFTEST_FUTURE_SURFACE_STATE", `${name}:${errors.join(",")}`);
		}
		rejected += 1;
	};
	const refuseC5Instance = (name, mutateManifest, observedRows) => {
		const candidate = structuredClone(manifest);
		mutateManifest(candidate);
		candidate.rows.sort((left, right) => {
			const boundary = ["C4", "C5"].indexOf(left.boundary) - ["C4", "C5"].indexOf(right.boundary);
			return boundary || compareObserved(left, right);
		});
		const manifestErrors = validateFutureSurfaceManifest(candidate);
		if (!manifestErrors.includes("C5 reserved rows")) {
			fail("P07B_B_SELFTEST_FUTURE_SURFACE_STATE", `${name}-manifest:${manifestErrors.join(",")}`);
		}
		const stateErrors = validateFutureSurfaceState(manifest, observedRows, c5Production, "C5");
		if (!stateErrors.includes("C5 observed future rows")) {
			fail("P07B_B_SELFTEST_FUTURE_SURFACE_STATE", `${name}-state:${stateErrors.join(",")}`);
		}
		rejected += 1;
	};
	refuseState("historical deletion", "C4", c4Observed.slice(1), c4Production, "historical 27-row inventory");
	refuseState("missing C4 row", "C4", c4Observed.filter(({ symbol }) => symbol !== "ContractExecutionRunner"), c4Production, "observed future rows");
	refuseState("extra C4 row", "C4", [...c4Observed, { path: "internal/contractexec/runner/runner.go", symbol: "ContractExecutionExtra" }].sort(compareObserved), c4Production, "observed future rows");
	refuseState("relocated C4 row", "C4", c4Observed.map((row) => row.symbol === "ContractExecutionRunner" ? { ...row, path: "internal/contractexec/runner/copy.go" } : row).sort(compareObserved), c4Production, "observed future rows");
	refuseState("premature C5 rows", "C4", c5Observed, c4Production, "observed future rows");
	refuseC5Instance("missing C5 row", (value) => {
		value.rows = value.rows.filter(({ path, symbol }) =>
			path !== "internal/contractexec/http/runner_darwin.go" || symbol !== "DeriveContractExecution");
	}, c5Observed.filter(({ path, symbol }) =>
		path !== "internal/contractexec/http/runner_darwin.go" || symbol !== "DeriveContractExecution"));
	refuseC5Instance("extra C5 row", (value) => {
		value.rows.push({ boundary: "C5", path: "internal/contractexec/http/runner_darwin.go", symbol: "ContractExecutionExtra" });
	}, [...c5Observed, { path: "internal/contractexec/http/runner_darwin.go", symbol: "ContractExecutionExtra" }].sort(compareObserved));
	refuseC5Instance("relocated C5 row", (value) => {
		value.rows = value.rows.map((row) => row.path === "internal/contractexec/http/evidence.go"
			? { ...row, path: "internal/contractexec/http/evidence_copy.go" }
			: row);
	}, c5Observed.map((row) => row.path === "internal/contractexec/http/evidence.go"
		? { ...row, path: "internal/contractexec/http/evidence_copy.go" }
		: row).sort(compareObserved));
	refuseState("runner absent", "C4", c4Observed, c4Production.filter((path) => !path.startsWith("internal/contractexec/runner/")), "C4 topology missing");
	refuseState("HTTP premature", "C4", c4Observed, [...c4Production, "internal/contractexec/http/empty.go"].sort(), "C4 topology premature");
	refuseState("scope premature", "C4", c4Observed, [...c4Production, "internal/contractexec/scope/empty.go"].sort(), "C4 topology premature");
	refuseState("C5 without C4", "C5", c5Observed, c5Production.filter((path) => !path.startsWith("internal/contractexec/runner/")), "C4 topology missing");
	refuseState("C5 HTTP absent", "C5", c5Observed, c5Production.filter((path) => !path.startsWith("internal/contractexec/http/")), "C5 topology missing");
	refuseState("C5 scope absent", "C5", c5Observed, c5Production.filter((path) => !path.startsWith("internal/contractexec/scope/")), "C5 topology missing");
	if (rejected !== 22) fail("P07B_B_SELFTEST_FUTURE_SURFACE_STATE", `control cardinality ${rejected}`);
}

async function main() {
	if (process.argv.length !== 2) fail("P07B_B_SELFTEST_ARGUMENTS", "no arguments accepted");
	const historicalDigest = rosterDigest(historicalCases);
	if (historicalDigest !== expectedHistoricalRosterDigest) {
		fail("P07B_B_SELFTEST_HISTORICAL_ROSTER_DRIFT", `${historicalDigest} != ${expectedHistoricalRosterDigest}`);
	}
	const digest = rosterDigest();
	if (digest !== expectedRosterDigest) fail("P07B_B_SELFTEST_ROSTER_DRIFT", `${digest} != ${expectedRosterDigest}`);
	runCleanChecker();
	const clean = await collectFacts();
	const cleanViolations = validateFacts(clean);
	if (cleanViolations.length !== 0) {
		fail("P07B_B_SELFTEST_CLEAN_FACTS", cleanViolations.map((entry) => entry.code).join(","));
	}

	for (const test of cases) {
		if (test.id === "inherited-c1-explicit") {
			requireExplicitInheritedC1();
			continue;
		}
		if (test.id === "prefix-partition") {
			requireExactC1PrefixPartition();
			continue;
		}
		if (test.id === "c3-surface-bundle") {
			requireExactC3SurfaceBundle();
			continue;
		}
		if (test.id === "future-manifest-authority") {
			requireFutureSurfaceStateMachine();
			const facts = structuredClone(clean);
			facts.futureSurfaceErrors.push("synthetic manifest/state drift");
			requireViolation(facts, "P07B_B_FUTURE_SURFACE_AUTHORITY", test.id);
			continue;
		}
		if (test.id === "future-state-machine") {
			requireFutureSurfaceStateMachine();
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
	process.stdout.write(`P07B B architecture defensive self-test OK (${cases.length} metadata cases; exact C1 prefix, C2 storage symbols, all-or-none C3 target surface, and symlink entrypoint)\n`);
}

main().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
