#!/usr/bin/env node

import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { chmodSync, mkdtempSync, readFileSync, realpathSync, rmSync, statSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const examplePath = join(repositoryRoot, "spec/examples/v1/contract-bundle.valid.json");
const mode = process.argv[2];
if (!['--check', '--write'].includes(mode) || process.argv.length !== 3) {
	process.stderr.write("usage: node tools/generate-p07b-a2-runtime-example.mjs --check|--write\n");
	process.exit(2);
}
const goCandidate = process.env.COUNTERSHAPE_GO;
if (typeof goCandidate !== "string" || !isAbsolute(goCandidate)) {
	throw new Error("COUNTERSHAPE_GO must name one absolute reviewed Go executable");
}
const go = realpathSync(goCandidate);
if (!statSync(go).isFile()) throw new Error("COUNTERSHAPE_GO is not a regular executable");

const root = mkdtempSync(join(realpathSync(tmpdir()), "countershape-a2-example-"));
chmodSync(root, 0o700);
try {
	const probe = join(root, "compiler-probe");
	const buildEnv = {
		HOME: process.env.HOME, TMPDIR: process.env.TMPDIR,
		GOCACHE: process.env.GOCACHE, GOPATH: process.env.GOPATH, GOMODCACHE: process.env.GOMODCACHE,
		PATH: `${dirname(go)}:/usr/bin:/bin`, GOENV: "off", GOWORK: "off", GOTOOLCHAIN: "local",
		GOPROXY: "off", GOSUMDB: "off", GOVCS: "*:off", GOFLAGS: "-mod=readonly -buildvcs=false -p=1",
		CGO_ENABLED: "0", GOMAXPROCS: "2", LANG: "C", LC_ALL: "C", TZ: "UTC", NO_COLOR: "1",
	};
	for (const name of ["HOME", "TMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE"]) {
		if (typeof buildEnv[name] !== "string" || !isAbsolute(buildEnv[name])) throw new Error(`${name} must be explicit and absolute`);
	}
	run(go, ["build", "-o", probe, "./internal/emit/node/cmd/p07b-a2-compiler-probe"], repositoryRoot, buildEnv);
	const result = run(probe, [], root, { HOME: root, TMPDIR: root, LANG: "C", LC_ALL: "C", TZ: "UTC", NO_COLOR: "1" }, 4 << 20);
	const lines = result.stdout.split("\n");
	assert.equal(lines.length, 3);
	assert.equal(lines[2], "");
	const canonical = strictBase64(lines[1]);
	assert.equal(lines[0], typedDigest("ContractBundle", canonical));
	const bundle = JSON.parse(canonical.toString("utf8"));
	assert.equal(canonicalJSON(bundle), canonical.toString("utf8"));
	assert.deepEqual(bundle.files.map((file) => file.path), [
		"README.md", "contract.test.mjs", "decision.json", "fixture.json", "harness.mjs", "manifest.json",
	]);
	assert.equal(bundle.files[1].byte_count > 10_000, true);
	assert.equal(bundle.files[4].byte_count > 100_000, true);
	const rendered = Buffer.from(`${JSON.stringify(bundle, null, 2)}\n`, "utf8");
	if (mode === "--write") {
		writeFileSync(examplePath, rendered);
		process.stdout.write(`P07B A2.2 runtime ContractBundle example written (${lines[0]})\n`);
	} else {
		assert.deepEqual(readFileSync(examplePath), rendered, "runtime ContractBundle example drifted from compiler probe");
		process.stdout.write(`P07B A2.2 runtime ContractBundle example exact (${lines[0]})\n`);
	}
} finally {
	rmSync(root, { recursive: true, force: true });
}

function run(executable, argv, cwd, env, maxBuffer) {
	const result = spawnSync(executable, argv, { cwd, env, encoding: "utf8", timeout: 2 * 60_000, maxBuffer });
	if (result.error || result.signal !== null || result.status !== 0 || result.stderr !== "") {
		throw new Error(`${executable} ${argv.join(" ")} failed: ${result.status ?? result.signal ?? result.error?.message}\n${result.stderr}`);
	}
	return result;
}

function strictBase64(text) {
	assert.match(text, /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/u);
	const bytes = Buffer.from(text, "base64");
	assert.equal(bytes.toString("base64"), text);
	return bytes;
}

function typedDigest(kind, bytes) {
	return `sha256:${createHash("sha256").update(`countershape/v1/${kind}\0`, "utf8").update(bytes).digest("hex")}`;
}

function canonicalJSON(value) {
	if (value === null || typeof value === "boolean" || typeof value === "string") return JSON.stringify(value);
	if (typeof value === "number") {
		if (!Number.isSafeInteger(value) || Object.is(value, -0)) throw new Error("noncanonical integer");
		return String(value);
	}
	if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
	if (typeof value !== "object") throw new Error("unsupported canonical value");
	const keys = Object.keys(value).sort((left, right) => Buffer.compare(Buffer.from(left), Buffer.from(right)));
	return `{${keys.map((key) => `${JSON.stringify(key)}:${canonicalJSON(value[key])}`).join(",")}}`;
}
