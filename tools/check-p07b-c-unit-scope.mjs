#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { readFile, realpath } from "node:fs/promises";
import { dirname, isAbsolute, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const specificationPath = resolve(repositoryRoot, "spec/verification/p07b-c-unit-paths.json");
const unitOrder = Object.freeze(["C0A", "C0B", "C1", "C1M", "C1V", "C1E", "C1B", "C2", "C2M", "C2B", "C3", "C4", "C5", "C6A", "C6B"]);
const verificationProfiles = new Set(["SOURCE_FULL", "RECEIPT_RECONCILIATION"]);
const receiptClaimTypes = new Set(["tests-pass", "command-succeeded"]);
const stagedInventoryArgs = Object.freeze([
	"diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--cached", "--name-only", "-z", "--no-renames", "--diff-filter=ACDMRTUXB", "--",
]);
const stagedIndexArgs = Object.freeze(["ls-files", "--stage", "-z", "--"]);
const credentialPatterns = Object.freeze([
	Object.freeze({ name: "pem-private-key", expression: /-----BEGIN (?:[A-Z0-9 ]+ )?PRIVATE KEY-----/u }),
	Object.freeze({ name: "aws-access-key", expression: /AKIA[0-9A-Z]{16}/u }),
	Object.freeze({ name: "github-token", expression: /gh[pousr]_[A-Za-z0-9]{30,}/u }),
	Object.freeze({ name: "gitlab-token", expression: /glpat-[A-Za-z0-9_-]{20,}/u }),
	Object.freeze({ name: "slack-token", expression: /xox[baprs]-[A-Za-z0-9-]{20,}/u }),
	Object.freeze({ name: "google-api-key", expression: /AIza[0-9A-Za-z_-]{35}/u }),
	Object.freeze({ name: "openai-key", expression: /sk-(?:proj-)?[A-Za-z0-9_-]{20,}/u }),
	Object.freeze({ name: "stripe-live-key", expression: /(?:sk|rk)_live_[A-Za-z0-9]{16,}/u }),
	Object.freeze({ name: "jwt-bearer", expression: /Bearer[ \t]+eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}/u }),
]);

function fail(message) {
	throw new Error(`P07B-C unit scope check failed: ${message}`);
}

function validPath(path, prefix = false) {
	if (typeof path !== "string" || path.length === 0 || path.length > 4096) return false;
	if (prefix && !path.endsWith("/")) return false;
	const checked = prefix ? path.slice(0, -1) : path;
	return checked.length > 0 && !isAbsolute(checked) && !checked.startsWith("./") &&
		!checked.includes("\\") && !checked.includes("//") &&
		!checked.split("/").some((part) => part === "" || part === "." || part === "..") &&
		!/[\u0000-\u001f\u007f]/u.test(checked) && (!prefix || !checked.endsWith(".md"));
}

function sortedUnique(values) {
	return Array.isArray(values) && new Set(values).size === values.length &&
		JSON.stringify(values) === JSON.stringify([...values].sort());
}

export function validateSpecification(specification) {
	if (!specification || typeof specification !== "object" || Array.isArray(specification) ||
		JSON.stringify(Object.keys(specification).sort()) !== JSON.stringify(["schema_version", "units"])) {
		fail("specification root roster");
	}
	if (specification.schema_version !== "countershape/p07b-c-unit-paths/v4") fail("specification version");
	if (!specification.units || typeof specification.units !== "object" || Array.isArray(specification.units) ||
		JSON.stringify(Object.keys(specification.units)) !== JSON.stringify(unitOrder)) fail("unit roster/order");

	for (const unit of unitOrder) {
		const entry = specification.units[unit];
		if (!entry || typeof entry !== "object" || Array.isArray(entry)) fail(`${unit}: field roster`);
		if (!verificationProfiles.has(entry.verification_profile)) fail(`${unit}: verification profile`);
		const expectedFields = entry.verification_profile === "RECEIPT_RECONCILIATION"
			? ["exact", "prefixes", "receipt_claims", "verification_profile"]
			: ["exact", "prefixes", "verification_profile"];
		if (JSON.stringify(Object.keys(entry).sort()) !== JSON.stringify(expectedFields)) fail(`${unit}: field roster`);
		if (!sortedUnique(entry.exact) || !entry.exact.every((path) => validPath(path))) fail(`${unit}: exact paths`);
		if (!sortedUnique(entry.prefixes) || !entry.prefixes.every((path) => validPath(path, true))) fail(`${unit}: prefixes`);
		if (entry.verification_profile === "RECEIPT_RECONCILIATION" &&
			(entry.prefixes.length !== 0 || entry.exact.some((path) =>
				!path.endsWith(".md") && !(/^spec\/verification\/[a-z0-9-]+-receipt\.json$/u.test(path))))) {
			fail(`${unit}: receipt reconciliation scope`);
		}
		if (entry.verification_profile === "RECEIPT_RECONCILIATION") {
			if (!Array.isArray(entry.receipt_claims) || entry.receipt_claims.length !== 7) fail(`${unit}: receipt claim count`);
			const labels = new Set();
			for (let index = 0; index < entry.receipt_claims.length; index += 1) {
				const claim = entry.receipt_claims[index];
				if (!claim || typeof claim !== "object" || Array.isArray(claim) ||
					JSON.stringify(Object.keys(claim).sort()) !== JSON.stringify(["label", "type"]) ||
					typeof claim.label !== "string" || claim.label.length === 0 || claim.label.length > 240 ||
					/[\u0000-\u001f\u007f]/u.test(claim.label) || labels.has(claim.label) ||
					!claim.label.startsWith(`P07B-C ${unit} `) || !receiptClaimTypes.has(claim.type) ||
					(index < 3 ? claim.type !== "tests-pass" : claim.type !== "command-succeeded") ||
					/\b(?:cumulative|suite|runtime|security)\b|unchanged[- ]behavior/iu.test(claim.label)) {
					fail(`${unit}: receipt claim ${index}`);
				}
				labels.add(claim.label);
			}
		}
		for (const exact of entry.exact) {
			if (entry.prefixes.some((prefix) => exact.startsWith(prefix))) fail(`${unit}: exact path redundantly covered by prefix: ${exact}`);
		}
	}
	return specification;
}

export function receiptManifest(specification, unit) {
	if (!unitOrder.includes(unit)) fail(`unknown unit ${unit}`);
	const entry = specification.units[unit];
	if (entry.verification_profile !== "RECEIPT_RECONCILIATION") fail(`${unit}: not a receipt reconciliation profile`);
	return entry.receipt_claims;
}

export function unexpectedPaths(specification, unit, paths) {
	if (!unitOrder.includes(unit)) fail(`unknown unit ${unit}`);
	if (!Array.isArray(paths) || !paths.every((path) => validPath(path))) fail("candidate path roster");
	const entry = specification.units[unit];
	return paths.filter((path) => !entry.exact.includes(path) && !entry.prefixes.some((prefix) => path.startsWith(prefix)));
}

export function exactPathsMatch(specification, unit, paths) {
	if (!unitOrder.includes(unit)) fail(`unknown unit ${unit}`);
	if (!Array.isArray(paths) || !paths.every((path) => validPath(path))) fail("candidate path roster");
	const entry = specification.units[unit];
	if (entry.prefixes.length !== 0) fail(`${unit}: exact staged mode requires an empty prefix roster`);
	return JSON.stringify([...paths].sort()) === JSON.stringify(entry.exact);
}

async function loadSpecification() {
	let parsed;
	try {
		parsed = JSON.parse(await readFile(specificationPath, "utf8"));
	} catch (error) {
		fail(`cannot read strict specification: ${error.message}`);
	}
	return validateSpecification(parsed);
}

async function gitOutput(args) {
	const requested = process.env.COUNTERSHAPE_GIT || "/usr/bin/git";
	if (!isAbsolute(requested)) fail("COUNTERSHAPE_GIT must be absolute");
	let git;
	try {
		git = await realpath(requested);
	} catch (error) {
		fail(`git admission failed: ${error.message}`);
	}
	const result = spawnSync(git, ["--no-replace-objects", ...args], {
		cwd: repositoryRoot,
		encoding: "buffer",
		timeout: 30_000,
		maxBuffer: 4 * 1024 * 1024,
		env: {
			HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1",
			GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: "/dev/null", GIT_NO_LAZY_FETCH: "1",
			GIT_OPTIONAL_LOCKS: "0", GIT_TERMINAL_PROMPT: "0",
		},
	});
	if (result.error || result.status !== 0 || result.signal || (result.stderr?.length ?? 0) !== 0) {
		fail(`git inventory failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
	}
	return result.stdout ?? Buffer.alloc(0);
}

async function stagedPaths() {
	const bytes = await gitOutput(stagedInventoryArgs);
	if (bytes.length > 0 && bytes[bytes.length - 1] !== 0) fail("git inventory omitted its final NUL delimiter");
	let decoded;
	try {
		decoded = new TextDecoder("utf-8", { fatal: true }).decode(bytes.length === 0 ? bytes : bytes.subarray(0, -1));
	} catch (error) {
		fail(`git inventory path is not valid UTF-8: ${error.message}`);
	}
	const parts = decoded.length === 0 ? [] : decoded.split("\0");
	if (new Set(parts).size !== parts.length) fail("duplicate staged path");
	return parts.sort();
}

async function stagedIndexEntries() {
	const bytes = await gitOutput(stagedIndexArgs);
	if (bytes.length > 0 && bytes[bytes.length - 1] !== 0) fail("git index inventory omitted its final NUL delimiter");
	let decoded;
	try {
		decoded = new TextDecoder("utf-8", { fatal: true }).decode(bytes.length === 0 ? bytes : bytes.subarray(0, -1));
	} catch (error) {
		fail(`git index inventory is not valid UTF-8: ${error.message}`);
	}
	const rows = decoded.length === 0 ? [] : decoded.split("\0");
	const entries = [];
	for (const row of rows) {
		const match = /^([0-7]{6}) ([0-9a-f]{40}|[0-9a-f]{64}) ([0-3])\t([\s\S]+)$/u.exec(row);
		if (!match || !validPath(match[4])) fail("malformed git index inventory row");
		entries.push(Object.freeze({ mode: match[1], object: match[2], stage: Number(match[3]), path: match[4] }));
	}
	return entries;
}

export function validateReceiptIndexModes(specification, unit, paths, entries) {
	receiptManifest(specification, unit);
	validateExactIndexModes(unit, paths, entries);
}

export function validateExactIndexModes(unit, paths, entries) {
	if (!Array.isArray(entries)) fail(`${unit}: index entry roster`);
	for (const path of paths) {
		const matches = entries.filter((entry) => entry?.path === path);
		if (matches.length !== 1 || matches[0].mode !== "100644" || matches[0].stage !== 0 ||
			!(/^(?:[0-9a-f]{40}|[0-9a-f]{64})$/u.test(matches[0].object)) || /^0+$/u.test(matches[0].object)) {
			fail(`${unit}: path must be one staged regular mode-100644 blob: ${path}`);
		}
	}
}

export function credentialPatternFindings(entries) {
	if (!Array.isArray(entries) || entries.some((entry) => !entry || typeof entry.path !== "string" || typeof entry.text !== "string")) {
		fail("credential scan entry roster");
	}
	const findings = [];
	for (const entry of entries) {
		for (const pattern of credentialPatterns) {
			if (pattern.expression.test(entry.text)) findings.push(Object.freeze({ pattern: pattern.name, path: entry.path }));
		}
	}
	return findings;
}

async function stagedBlobEntries(paths) {
	const entries = [];
	for (const path of paths) {
		const bytes = await gitOutput(["show", `:${path}`]);
		let text;
		try {
			text = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
		} catch (error) {
			fail(`staged blob is not UTF-8 text: ${path} (${error.message})`);
		}
		if (bytes.length === 0) fail(`staged blob is empty: ${path}`);
		entries.push(Object.freeze({ path, text }));
	}
	return entries;
}

async function requireExactStagedSource(specification, unit) {
	const paths = await stagedPaths();
	if (paths.length === 0) fail("staged inventory is empty");
	if (!exactPathsMatch(specification, unit, paths)) fail(`${unit} exact staged roster mismatch`);
	const entries = await stagedIndexEntries();
	const repeatedPaths = await stagedPaths();
	if (JSON.stringify(repeatedPaths) !== JSON.stringify(paths)) fail("staged inventory changed during source inspection");
	validateExactIndexModes(unit, paths, entries);
	return paths;
}

function exactSourceGateAdmitted(specification, unit) {
	return unitOrder.includes(unit) &&
		specification.units[unit]?.verification_profile === "SOURCE_FULL" &&
		specification.units[unit].prefixes.length === 0;
}

async function runSourceFinalGate(specification, unit) {
	if (!exactSourceGateAdmitted(specification, unit)) {
		fail("source-final-gate is admitted only for a declared exact-roster SOURCE_FULL unit");
	}
	const paths = await requireExactStagedSource(specification, unit);
	await gitOutput(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--cached", "--check", "--"]);
	const unstaged = await gitOutput(["diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--name-only", "-z", "--"]);
	if (unstaged.length !== 0) fail("unstaged tracked changes are present");
	const untracked = await gitOutput(["ls-files", "--others", "--exclude-standard", "-z", "--"]);
	if (untracked.length !== 0) fail("untracked paths are present");
	const finalPaths = await stagedPaths();
	if (JSON.stringify(finalPaths) !== JSON.stringify(paths)) fail("staged inventory changed during diff-integrity checks");
	const digest = createHash("sha256").update(`${paths.join("\n")}\n`, "utf8").digest("hex");
	console.log(`P07B-C ${unit} final source gate passed: ${paths.length} exact mode-100644 paths, clean staged diff, no unstaged or untracked paths, sorted-newline sha256:${digest}`);
}

async function runCredentialScan(specification, unit) {
	if (!exactSourceGateAdmitted(specification, unit)) {
		fail("credential-scan is admitted only for a declared exact-roster SOURCE_FULL unit");
	}
	const paths = await requireExactStagedSource(specification, unit);
	const findings = credentialPatternFindings(await stagedBlobEntries(paths));
	if (findings.length > 0) fail(`structured credential-pattern findings: ${JSON.stringify(findings)}`);
	console.log(`P07B-C ${unit} scoped staged structured credential-pattern scan: 0 findings across ${paths.length} exact paths and ${credentialPatterns.length} named patterns`);
}

async function runSelfTest() {
	const specification = await loadSpecification();
	const cases = [
		unexpectedPaths(specification, "C0A", ["docs/SEMANTICS.md"]).length === 0,
		unexpectedPaths(specification, "C0A", ["README.md"])[0] === "README.md",
		unexpectedPaths(specification, "C4", ["internal/processmechanics/owner_darwin.go"]).length === 0,
		unexpectedPaths(specification, "C4", ["internal/processmechanics_evil/owner_darwin.go"])[0] === "internal/processmechanics_evil/owner_darwin.go",
		unexpectedPaths(specification, "C2", ["internal/store/nonhead_backdoor.go"])[0] === "internal/store/nonhead_backdoor.go",
		unexpectedPaths(specification, "C2M", ["tools/check-p07b-c-plan.mjs"]).length === 0,
		unexpectedPaths(specification, "C2M", ["internal/store/nonhead_contract.go"])[0] === "internal/store/nonhead_contract.go",
		unexpectedPaths(specification, "C3", ["internal/contractexec/target_evil.go"])[0] === "internal/contractexec/target_evil.go",
		unexpectedPaths(specification, "C1B", ["spec/verification/p07b-c-c1-receipt.json"]).length === 0,
		unexpectedPaths(specification, "C1B", ["internal/store/nonhead_contract.go"])[0] === "internal/store/nonhead_contract.go",
		unexpectedPaths(specification, "C2B", ["spec/verification/p07b-c-c2-receipt.json"]).length === 0,
		unexpectedPaths(specification, "C2B", ["internal/store/nonhead_contract.go"])[0] === "internal/store/nonhead_contract.go",
		unexpectedPaths(specification, "C1M", ["internal/world/process_darwin.go"]).length === 0,
		unexpectedPaths(specification, "C1M", ["spec/verification/p07b-c-c1-receipt.json"])[0] === "spec/verification/p07b-c-c1-receipt.json",
		exactPathsMatch(specification, "C1M", specification.units.C1M.exact),
		!exactPathsMatch(specification, "C1M", specification.units.C1M.exact.slice(1)),
		unexpectedPaths(specification, "C1V", ["tools/verify-current.mjs"]).length === 0,
		unexpectedPaths(specification, "C1V", ["spec/verification/p07b-c-c1-receipt.json"])[0] === "spec/verification/p07b-c-c1-receipt.json",
		exactPathsMatch(specification, "C1V", specification.units.C1V.exact),
		!exactPathsMatch(specification, "C1V", specification.units.C1V.exact.slice(1)),
		unexpectedPaths(specification, "C1E", ["tools/check-p07b-c-plan.mjs"]).length === 0,
		unexpectedPaths(specification, "C1E", ["spec/verification/p07b-c-c1-receipt.json"])[0] === "spec/verification/p07b-c-c1-receipt.json",
		exactPathsMatch(specification, "C1E", specification.units.C1E.exact),
		!exactPathsMatch(specification, "C1E", specification.units.C1E.exact.slice(1)),
		exactPathsMatch(specification, "C1B", specification.units.C1B.exact),
		!exactPathsMatch(specification, "C1B", specification.units.C1B.exact.slice(1)),
		exactPathsMatch(specification, "C2B", specification.units.C2B.exact),
		!exactPathsMatch(specification, "C2B", specification.units.C2B.exact.slice(1)),
		exactPathsMatch(specification, "C2M", specification.units.C2M.exact),
		!exactPathsMatch(specification, "C2M", specification.units.C2M.exact.slice(1)),
		exactSourceGateAdmitted(specification, "C2M"),
		!exactSourceGateAdmitted(specification, "C3"),
		!exactSourceGateAdmitted(specification, "C2B"),
		unexpectedPaths(specification, "C0A", ["docs/SEMANTICS.md", "README.md"])[0] === "README.md",
		JSON.stringify(stagedInventoryArgs) === JSON.stringify([
			"diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--cached", "--name-only", "-z", "--no-renames", "--diff-filter=ACDMRTUXB", "--",
		]),
		JSON.stringify(stagedIndexArgs) === JSON.stringify(["ls-files", "--stage", "-z", "--"]),
		receiptManifest(specification, "C1B").length === 7,
		receiptManifest(specification, "C2B").length === 7,
	];
	if (cases.some((value) => !value)) fail("allow/refuse self-test matrix");
	const unsafePrefix = JSON.parse(JSON.stringify(specification));
	unsafePrefix.units.C4.prefixes = ["internal/processmechanics"];
	let unsafePrefixRejected = false;
	try {
		validateSpecification(unsafePrefix);
	} catch {
		unsafePrefixRejected = true;
	}
	if (!unsafePrefixRejected) fail("lexical prefix self-test false negative");
	const unsafeReceiptProfile = JSON.parse(JSON.stringify(specification));
	unsafeReceiptProfile.units.C1V.verification_profile = "RECEIPT_RECONCILIATION";
	let unsafeReceiptProfileRejected = false;
	try {
		validateSpecification(unsafeReceiptProfile);
	} catch {
		unsafeReceiptProfileRejected = true;
	}
	if (!unsafeReceiptProfileRejected) fail("receipt reconciliation source-scope self-test false negative");
	const reorderedUnits = JSON.parse(JSON.stringify(specification));
	const reorderedC2B = reorderedUnits.units.C2B;
	delete reorderedUnits.units.C2B;
	reorderedUnits.units.C2B = reorderedC2B;
	let reorderedUnitsRejected = false;
	try { validateSpecification(reorderedUnits); } catch { reorderedUnitsRejected = true; }
	if (!reorderedUnitsRejected) fail("unit order self-test false negative");
	for (const unit of ["C1B", "C2B"]) {
		const regular = specification.units[unit].exact.map((path, index) => ({
			mode: "100644", object: String(index + 1).padStart(40, "0"), stage: 0, path,
		}));
		validateReceiptIndexModes(specification, unit, specification.units[unit].exact, regular);
		for (const [name, mutate] of [
			["symlink", (entries) => { entries[0].mode = "120000"; }],
			["executable", (entries) => { entries[0].mode = "100755"; }],
			["unmerged", (entries) => { entries[0].stage = 2; }],
			["intent-to-add", (entries) => { entries[0].object = "0".repeat(40); }],
			["deleted", (entries) => { entries.shift(); }],
		]) {
			const hostile = regular.map((entry) => ({ ...entry }));
			mutate(hostile);
			let rejected = false;
			try { validateReceiptIndexModes(specification, unit, specification.units[unit].exact, hostile); } catch { rejected = true; }
			if (!rejected) fail(`${unit} receipt index-mode self-test false negative: ${name}`);
		}
	}
	const cleanCredentialEntries = [{ path: "fixture", text: "ordinary source text" }];
	if (credentialPatternFindings(cleanCredentialEntries).length !== 0) fail("credential scan clean self-test false positive");
	const hostileCredentials = [
		"-----BEGIN " + "PRIVATE KEY-----",
		"AK" + "IA" + "A".repeat(16),
		"gh" + "p_" + "a".repeat(30),
		"gl" + "pat-" + "a".repeat(20),
		"xo" + "xb-" + "a".repeat(20),
		"AI" + "za" + "a".repeat(35),
		"s" + "k-proj-" + "a".repeat(20),
		"s" + "k_live_" + "a".repeat(16),
		"Bearer " + "eyJ" + "a".repeat(10) + "." + "b".repeat(10) + "." + "c".repeat(10),
	];
	for (let index = 0; index < hostileCredentials.length; index += 1) {
		const findings = credentialPatternFindings([{ path: `fixture-${index}`, text: hostileCredentials[index] }]);
		if (findings.length !== 1 || findings[0].pattern !== credentialPatterns[index].name) {
			fail(`credential scan hostile self-test false negative: ${credentialPatterns[index].name}`);
		}
	}
	for (const invalid of ["../escape", "./alias", "/absolute", "double//slash", "control\npath"]) {
		let rejected = false;
		try {
			unexpectedPaths(specification, "C0A", [invalid]);
		} catch {
			rejected = true;
		}
		if (!rejected) fail(`unsafe path self-test false negative: ${JSON.stringify(invalid)}`);
	}
	console.log("P07B-C unit scope self-test passed: strict specification plus deletion/rename, directory-boundary, allow/refuse, and path-safety matrices");
}

async function main() {
	if (process.argv.length === 3 && process.argv[2] === "--self-test") {
		await runSelfTest();
		return;
	}
	if (process.argv.length !== 5 || process.argv[2] !== "--unit" ||
		!(["--staged", "--exact-staged", "--receipt-manifest", "--source-final-gate", "--credential-scan"].includes(process.argv[4]))) {
		fail("usage: check-p07b-c-unit-scope.mjs --unit <C0A|C0B|C1|C1M|C1V|C1E|C1B|C2|C2M|C2B|C3|C4|C5|C6A|C6B> <--staged|--exact-staged|--receipt-manifest|--source-final-gate|--credential-scan> | --self-test");
	}
	const specification = await loadSpecification();
	if (process.argv[4] === "--source-final-gate") {
		await runSourceFinalGate(specification, process.argv[3]);
		return;
	}
	if (process.argv[4] === "--credential-scan") {
		await runCredentialScan(specification, process.argv[3]);
		return;
	}
	if (process.argv[4] === "--receipt-manifest") {
		const claims = receiptManifest(specification, process.argv[3]);
		console.log(`P07B-C ${process.argv[3]} receipt manifest exact: ${JSON.stringify(claims)}`);
		return;
	}
	const paths = await stagedPaths();
	if (paths.length === 0) fail("staged inventory is empty");
	if (specification.units[process.argv[3]]?.verification_profile === "RECEIPT_RECONCILIATION") {
		const entries = await stagedIndexEntries();
		const repeatedPaths = await stagedPaths();
		if (JSON.stringify(repeatedPaths) !== JSON.stringify(paths)) fail("staged inventory changed during receipt mode inspection");
		validateReceiptIndexModes(specification, process.argv[3], paths, entries);
	}
	if (process.argv[4] === "--exact-staged") {
		if (!exactPathsMatch(specification, process.argv[3], paths)) {
			fail(`${process.argv[3]} exact staged roster mismatch`);
		}
		const digest = createHash("sha256").update(`${paths.join("\n")}\n`, "utf8").digest("hex");
		console.log(`P07B-C ${process.argv[3]} exact staged scope passed: ${paths.length} path(s), sorted-newline sha256:${digest}`);
		return;
	}
	const unexpected = unexpectedPaths(specification, process.argv[3], paths);
	if (unexpected.length > 0) fail(`${process.argv[3]} unexpected staged paths: ${unexpected.join(", ")}`);
	console.log(`P07B-C ${process.argv[3]} staged scope passed: ${paths.length} admitted path(s)`);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) await main();
