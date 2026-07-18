#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { readFile, realpath } from "node:fs/promises";
import { dirname, isAbsolute, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const specificationPath = resolve(repositoryRoot, "spec/verification/p07b-c-unit-paths.json");
const unitOrder = Object.freeze(["C0A", "C0B", "C1", "C2", "C3", "C4", "C5", "C6A", "C6B"]);
const stagedInventoryArgs = Object.freeze([
	"diff", "--cached", "--name-only", "-z", "--no-renames", "--diff-filter=ACDMRTUXB", "--",
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
	if (specification.schema_version !== "countershape/p07b-c-unit-paths/v1") fail("specification version");
	if (!specification.units || typeof specification.units !== "object" || Array.isArray(specification.units) ||
		JSON.stringify(Object.keys(specification.units)) !== JSON.stringify(unitOrder)) fail("unit roster/order");

	for (const unit of unitOrder) {
		const entry = specification.units[unit];
		if (!entry || typeof entry !== "object" || Array.isArray(entry) ||
			JSON.stringify(Object.keys(entry).sort()) !== JSON.stringify(["exact", "prefixes"])) fail(`${unit}: field roster`);
		if (!sortedUnique(entry.exact) || !entry.exact.every((path) => validPath(path))) fail(`${unit}: exact paths`);
		if (!sortedUnique(entry.prefixes) || !entry.prefixes.every((path) => validPath(path, true))) fail(`${unit}: prefixes`);
		for (const exact of entry.exact) {
			if (entry.prefixes.some((prefix) => exact.startsWith(prefix))) fail(`${unit}: exact path redundantly covered by prefix: ${exact}`);
		}
	}
	return specification;
}

export function unexpectedPaths(specification, unit, paths) {
	if (!unitOrder.includes(unit)) fail(`unknown unit ${unit}`);
	if (!Array.isArray(paths) || !paths.every((path) => validPath(path))) fail("candidate path roster");
	const entry = specification.units[unit];
	return paths.filter((path) => !entry.exact.includes(path) && !entry.prefixes.some((prefix) => path.startsWith(prefix)));
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

async function stagedPaths() {
	const requested = process.env.COUNTERSHAPE_GIT || "/usr/bin/git";
	if (!isAbsolute(requested)) fail("COUNTERSHAPE_GIT must be absolute");
	let git;
	try {
		git = await realpath(requested);
	} catch (error) {
		fail(`git admission failed: ${error.message}`);
	}
	const result = spawnSync(git, stagedInventoryArgs, {
		cwd: repositoryRoot,
		encoding: "buffer",
		timeout: 30_000,
		maxBuffer: 4 * 1024 * 1024,
		env: { HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1" },
	});
	if (result.error || result.status !== 0 || result.signal || (result.stderr?.length ?? 0) !== 0) {
		fail(`git staged inventory failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
	}
	const bytes = result.stdout ?? Buffer.alloc(0);
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

async function runSelfTest() {
	const specification = await loadSpecification();
	const cases = [
		unexpectedPaths(specification, "C0A", ["docs/SEMANTICS.md"]).length === 0,
		unexpectedPaths(specification, "C0A", ["README.md"])[0] === "README.md",
		unexpectedPaths(specification, "C4", ["internal/processmechanics/owner_darwin.go"]).length === 0,
		unexpectedPaths(specification, "C4", ["internal/processmechanics_evil/owner_darwin.go"])[0] === "internal/processmechanics_evil/owner_darwin.go",
		unexpectedPaths(specification, "C2", ["internal/store/nonhead_backdoor.go"])[0] === "internal/store/nonhead_backdoor.go",
		unexpectedPaths(specification, "C3", ["internal/contractexec/target_evil.go"])[0] === "internal/contractexec/target_evil.go",
		unexpectedPaths(specification, "C0A", ["docs/SEMANTICS.md", "README.md"])[0] === "README.md",
		JSON.stringify(stagedInventoryArgs) === JSON.stringify([
			"diff", "--cached", "--name-only", "-z", "--no-renames", "--diff-filter=ACDMRTUXB", "--",
		]),
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
	if (process.argv.length !== 5 || process.argv[2] !== "--unit" || process.argv[4] !== "--staged") {
		fail("usage: check-p07b-c-unit-scope.mjs --unit <C0A|C0B|C1|C2|C3|C4|C5|C6A|C6B> --staged | --self-test");
	}
	const specification = await loadSpecification();
	const paths = await stagedPaths();
	if (paths.length === 0) fail("staged inventory is empty");
	const unexpected = unexpectedPaths(specification, process.argv[3], paths);
	if (unexpected.length > 0) fail(`${process.argv[3]} unexpected staged paths: ${unexpected.join(", ")}`);
	console.log(`P07B-C ${process.argv[3]} staged scope passed: ${paths.length} admitted path(s)`);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) await main();
