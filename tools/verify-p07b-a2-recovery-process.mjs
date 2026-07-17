#!/usr/bin/env node

import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { chmodSync, mkdirSync, mkdtempSync, readFileSync, realpathSync, rmSync, statSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const goCandidate = process.env.COUNTERSHAPE_GO;
if (typeof goCandidate !== "string" || !isAbsolute(goCandidate)) {
	throw new Error("COUNTERSHAPE_GO must name one absolute reviewed Go executable");
}
const go = realpathSync(goCandidate);
if (!statSync(go).isFile()) throw new Error("COUNTERSHAPE_GO is not a regular executable");

const root = mkdtempSync(join(realpathSync(tmpdir()), "countershape-a2-recovery-"));
chmodSync(root, 0o700);
try {
	const compiler = join(root, "compiler-probe");
	const parser = join(root, "parser-probe");
	const compilerCWD = join(root, "compiler-cwd");
	const parserCWD = join(root, "parser-cwd");
	const home = join(root, "home");
	const runtimeTmp = join(root, "runtime-tmp");
	for (const directory of [compilerCWD, parserCWD, home, runtimeTmp]) {
		mkdirSync(directory, { mode: 0o700 });
		chmodSync(directory, 0o700);
	}
	const buildEnv = {
		HOME: process.env.HOME, TMPDIR: process.env.TMPDIR,
		GOCACHE: process.env.GOCACHE, GOPATH: process.env.GOPATH, GOMODCACHE: process.env.GOMODCACHE,
		PATH: `${dirname(go)}:/usr/bin:/bin`, GOENV: "off", GOWORK: "off", GOTOOLCHAIN: "local",
		GOPROXY: "off", GOSUMDB: "off", GOVCS: "*:off", GOFLAGS: "-mod=readonly -buildvcs=false -p=1",
		CGO_ENABLED: "0", GOMAXPROCS: "2", LANG: "C", LC_ALL: "C", TZ: "UTC", NO_COLOR: "1",
	};
	for (const [name, value] of Object.entries(buildEnv)) {
		if (typeof value !== "string" || (["HOME", "TMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE"].includes(name) && !isAbsolute(value))) {
			throw new Error(`${name} must be one explicit absolute path`);
		}
	}
	run(go, ["build", "-o", compiler, "./internal/emit/node/cmd/p07b-a2-compiler-probe"], repositoryRoot, buildEnv);
	run(go, ["build", "-o", parser, "./internal/emit/node/cmd/p07b-a2-parser-probe"], repositoryRoot, buildEnv);

	const runtimeEnv = { HOME: home, TMPDIR: runtimeTmp, LANG: "C", LC_ALL: "C", TZ: "UTC", NO_COLOR: "1" };
	const compiled = run(compiler, [], compilerCWD, runtimeEnv, undefined, 4 << 20);
	const lines = compiled.stdout.split("\n");
	assert.equal(lines.length, 3);
	assert.equal(lines[2], "");
	assert.match(lines[0], /^sha256:[0-9a-f]{64}$/u);
	const body = strictBase64(lines[1]);
	assert.equal(lines[0], typedDigest("ContractBundle", body));

	const recoveredRun = run(parser, [], parserCWD, runtimeEnv, compiled.stdout, 4 << 20);
	assert.equal(recoveredRun.stdout.endsWith("\n"), true);
	assert.equal(recoveredRun.stdout.includes("\r"), false);
	const recoveredText = recoveredRun.stdout.slice(0, -1);
	const recovered = JSON.parse(recoveredText);
	assert.equal(canonicalJSON(recovered), recoveredText);
	assert.deepEqual(Object.keys(recovered).sort(), [
		"bundle_digest", "files", "kind", "portable_source_base64", "portable_source_digest", "schema_version",
	]);
	assert.equal(recovered.schema_version, "countershape.p07b-a2.parser-probe/v1");
	assert.equal(recovered.kind, "RecoveredContractBundle");
	assert.equal(recovered.bundle_digest, lines[0]);
	assert.match(recovered.portable_source_digest, /^sha256:[0-9a-f]{64}$/u);
	strictBase64(recovered.portable_source_base64);
	assert.deepEqual(recovered.files.map((file) => file.path), [
		"README.md", "contract.test.mjs", "decision.json", "fixture.json", "harness.mjs", "manifest.json",
	]);
	const outer = JSON.parse(body.toString("utf8"));
	assert.deepEqual(recovered.files, outer.files.map((file) => ({
		path: file.path, mode: file.mode, byte_count: file.byte_count,
		byte_sha256: file.byte_sha256, content_base64: file.content_base64,
	})));
	assert.equal(recovered.portable_source_digest, outer.portable_source_digest);
	assert.equal(readFileSync(compiler).length > 0, true);
	assert.equal(readFileSync(parser).length > 0, true);
	process.stdout.write("P07B A2.2 fresh-process recovery OK\n");
} finally {
	rmSync(root, { recursive: true, force: true });
}

function run(executable, argv, cwd, env, input = undefined, maxBuffer = 2 << 20) {
	const result = spawnSync(executable, argv, {
		cwd, env, input, encoding: "utf8", timeout: 2 * 60_000, maxBuffer,
	});
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
