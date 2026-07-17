#!/usr/bin/env node

import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import {
	chmodSync,
	lstatSync,
	mkdirSync,
	mkdtempSync,
	readFileSync,
	readdirSync,
	realpathSync,
	rmSync,
	statSync,
	writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { TextDecoder } from "node:util";

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const bundleExamplePath = join(repositoryRoot, "spec/examples/v1/contract-bundle.valid.json");
const capturePath = join(repositoryRoot, "docs/evidence/P07B-A2-2-HUMAN-SURFACE.json");
const captureVersion = "countershape-p07b-a2-human-surface/v1";
const renderGrammar = "countershape-terminal-capture/v1";
const tapNormalizationProfile = "countershape-node-tap-normalization/v1";
const captureKind = "HumanSurfaceCapture";
const bundleRootKeys = Object.freeze([
	"bundle_version", "choicepoint_digest", "confidentiality_established", "countershape_runtime_binding",
	"decision_action", "decision_record_digest", "determinism_profile", "emitter_version", "environment_profile",
	"external_service_binding", "files", "kind", "manifest_policy", "package_registry_binding",
	"portable_profile_digest", "portable_source_digest", "predicate", "runtime_dependency_profile", "schema_version",
	"source_profile",
].sort(unsignedByteOrder));
const widths = Object.freeze([60, 80, 120]);
const exactBundlePaths = Object.freeze([
	"README.md",
	"contract.test.mjs",
	"decision.json",
	"fixture.json",
	"harness.mjs",
	"manifest.json",
]);
const scenarios = Object.freeze([
	Object.freeze({
		id: "conforms",
		outcome: "CONFORMS",
		reason: "NONE",
		exitCode: 0,
		setupProfile: "BASELINE_EXACT_STDIN_OK",
		subject: cliSubject("ok\n"),
	}),
	Object.freeze({
		id: "contradicts",
		outcome: "CONTRADICTS",
		reason: "PREDICATE_MISMATCH",
		exitCode: 1,
		setupProfile: "BASELINE_EXACT_STDIN_DIFFERENT_OUTPUT",
		subject: cliSubject("different\n"),
	}),
	Object.freeze({
		id: "ineligible-timeout",
		outcome: "INELIGIBLE_EXECUTION",
		reason: "TIMEOUT",
		exitCode: 1,
		setupProfile: "BASELINE_NONTERMINATING_SUBJECT",
		subject: "setInterval(() => {}, 1000);\n",
	}),
	Object.freeze({
		id: "malformed-contract",
		outcome: "MALFORMED_CONTRACT",
		reason: "CONTRACT_DATA_INVALID",
		exitCode: 1,
		setupProfile: "MANIFEST_REPINNED_MALFORMED_DECISION",
		subject: cliSubject("ok\n"),
		variant: "malformed-decision",
	}),
	Object.freeze({
		id: "tamper-detected",
		outcome: "TAMPER_DETECTED",
		reason: "COMPANION_INTEGRITY_MISMATCH",
		exitCode: 1,
		setupProfile: "UNPINNED_DECISION_CHANGE",
		subject: cliSubject("ok\n"),
		variant: "tampered-decision",
	}),
	Object.freeze({
		id: "harness-failure",
		outcome: "HARNESS_FAILURE",
		reason: "INTERNAL_INVARIANT_FAILED",
		exitCode: 1,
		setupProfile: "MANIFEST_REPINNED_HARNESS_THROW",
		subject: cliSubject("ok\n"),
		variant: "failing-harness",
	}),
]);

function cliSubject(output) {
	return `const chunks = [];
process.stdin.on("data", (chunk) => chunks.push(Buffer.from(chunk)));
process.stdin.on("end", () => {
  if (!Buffer.concat(chunks).equals(Buffer.from("contract-input"))) process.exit(65);
  process.stdout.write(${JSON.stringify(output)});
});
process.stdin.resume();
`;
}

const argv = process.argv.slice(2);
if (argv[0] === "--render" && argv.length === 2) {
	const width = Number(argv[1]);
	if (!widths.includes(width) || String(width) !== argv[1]) usage();
	const files = readBundleFiles().files;
	process.stdout.write(renderMarkdown(decodeUTF8(files.get("README.md"), "README.md"), width));
} else if (argv[0] === "--self-test" && argv.length === 1) {
	selfTest();
	process.stdout.write("P07B A2.2 human-surface renderer self-test OK\n");
} else if (["--check", "--write"].includes(argv[0]) && argv.length === 1) {
	const node = admitNode();
	const rendered = Buffer.from(`${JSON.stringify(buildCapture(node), null, 2)}\n`, "utf8");
	if (argv[0] === "--write") {
		mkdirSync(dirname(capturePath), { recursive: true, mode: 0o755 });
		writeFileSync(capturePath, rendered, { mode: 0o644 });
		process.stdout.write("P07B A2.2 human-surface capture written\n");
	} else {
		assert.deepEqual(readFileSync(capturePath), rendered, "human-surface capture drifted");
		process.stdout.write("P07B A2.2 human-surface capture exact\n");
	}
} else {
	usage();
}

function usage() {
	process.stderr.write(
		"usage: node tools/capture-p07b-a2-human-surface.mjs --check|--write|--self-test|--render 60|80|120\n",
	);
	process.exit(2);
}

function admitNode() {
	const candidate = process.env.COUNTERSHAPE_NODE;
	if (typeof candidate !== "string" || !isAbsolute(candidate)) {
		throw new Error("COUNTERSHAPE_NODE must name one absolute reviewed Node executable");
	}
	const node = realpathSync(candidate);
	const info = statSync(node);
	if (!info.isFile() || (info.mode & 0o111) === 0) {
		throw new Error("COUNTERSHAPE_NODE is not a regular executable");
	}
	return node;
}

function buildCapture(node) {
	selfTest();
	const bundle = readBundleFiles();
	const files = bundle.files;
	const readme = files.get("README.md");
	const readmeText = decodeUTF8(readme, "README.md");
	const renderedReadmes = widths.map((width) => {
		const rendered = renderMarkdown(readmeText, width);
		const lineLengths = rendered.slice(0, -1).split("\n").map(codepointLength);
		assert.equal(lineLengths.every((length) => length <= width), true);
		return {
			grammar: renderGrammar,
			line_count: lineLengths.length,
			max_line_codepoints: Math.max(...lineLengths),
			rendered,
			rendered_sha256: rawSHA256(Buffer.from(rendered, "utf8")),
			width,
		};
	});
	const cliRecords = scenarios.map((scenario) => runScenario(node, files, scenario, "PRIMARY"));
	const repeated = [...scenarios].reverse().map((scenario) => runScenario(node, files, scenario, "REPEAT"));
	for (const record of repeated) {
		assert.deepEqual(record, cliRecords.find((candidate) => candidate.scenario === record.scenario), `${record.scenario}: repeated capture drifted`);
	}
	return {
		capture_version: captureVersion,
		cli_records: cliRecords,
		contract_bundle_digest: bundle.digest,
		kind: captureKind,
		readme: {
			captures: renderedReadmes,
			raw_byte_count: readme.length,
			raw_sha256: rawSHA256(readme),
			source: "spec/examples/v1/contract-bundle.valid.json#README.md",
		},
		tap_normalization_profile: tapNormalizationProfile,
	};
}

function readBundleFiles() {
	const bundleBytes = readFileSync(bundleExamplePath);
	assert.equal(bundleBytes.at(-1), 0x0a, "bundle example must end in LF");
	assert.equal(bundleBytes.includes(0x0d), false, "bundle example must not contain CR");
	const bundleText = decodeUTF8(bundleBytes, "bundle example");
	const bundle = JSON.parse(bundleText);
	assert.deepEqual(Object.keys(bundle).sort(unsignedByteOrder), bundleRootKeys, "bundle root roster drifted");
	assert.equal(bundleText, `${JSON.stringify(bundle, null, 2)}\n`, "bundle example is not exact pretty JSON plus LF");
	assert.equal(bundle.schema_version, "countershape/v1");
	assert.equal(bundle.kind, "ContractBundle");
	assert.equal(bundle.bundle_version, "node-core-contract-bundle/v1");
	assert.equal(bundle.emitter_version, "node-exact-emitter/v1");
	assert.equal(bundle.manifest_policy, "COVERS_OTHER_FIVE_EXCLUDES_SELF_V1");
	assert.equal(bundle.runtime_dependency_profile, "NODE_CORE_ONLY_V1");
	assert.equal(bundle.countershape_runtime_binding, "ABSENT_BY_CONSTRUCTION");
	assert.equal(bundle.package_registry_binding, "NONE");
	assert.equal(bundle.environment_profile, "EXPLICIT_SPARSE_ALLOWLIST_V1");
	assert.equal(bundle.external_service_binding, "NONE");
	assert.equal(bundle.confidentiality_established, false);
	assert.match(bundle.portable_source_digest, /^sha256:[0-9a-f]{64}$/u);
	assert.match(bundle.portable_profile_digest, /^sha256:[0-9a-f]{64}$/u);
	assert.equal(bundle.decision_action, "ALLOW_OBSERVED");
	assert.deepEqual(bundle.source_profile, {
		adapter_domain: "CLI",
		launch_profile: "NODE_REPO_SCRIPT_V1",
		runtime_family: "NODE",
		scope: "DECLARED_SOURCE_PROFILE_NOT_EXECUTION_EVIDENCE",
		semantic_profile: "countershape-node-core-exact/v1",
		start_profile: "DIRECT_CHILD_V1",
		subject_entrypoint: "fixture/subject.mjs",
	});
	assert.equal(bundle.predicate.kind, "one-of-exact/v1");
	assert.equal(bundle.predicate.scope, "EXACT_WITNESSED_STIMULUS");
	assert.deepEqual(bundle.predicate.selected_fields, ["cli.stdout.bytes"]);
	assert.equal(bundle.predicate.portable_profile_digest, bundle.portable_profile_digest);
	const canonicalBody = Buffer.from(canonicalJSON(bundle), "utf8");
	const digest = typedDigest("ContractBundle", canonicalBody);
	assert.deepEqual(bundle.files?.map((file) => file.path), exactBundlePaths);
	const files = new Map();
	for (const file of bundle.files) {
		assert.deepEqual(Object.keys(file).sort(), ["byte_count", "byte_sha256", "content_base64", "mode", "path"]);
		assert.equal(file.mode, "100644");
		const content = strictBase64(file.content_base64);
		assert.equal(content.length, file.byte_count);
		assert.equal(rawSHA256(content), file.byte_sha256);
		files.set(file.path, content);
	}
	return { digest, files };
}

function runScenario(node, sourceFiles, scenario, repeatProfile) {
	const root = mkdtempSync(join(realpathSync(tmpdir()), "countershape-a2-surface-"));
	chmodSync(root, 0o700);
	try {
		const bundleRoot = join(root, "bundle");
		const targetRoot = join(root, "target");
		const runtimeRoot = join(root, "runtime");
		const home = join(root, "home");
		for (const path of [bundleRoot, targetRoot, runtimeRoot, home, join(targetRoot, "fixture")]) {
			mkdirSync(path, { recursive: true, mode: 0o700 });
			chmodSync(path, 0o700);
		}
		const files = new Map([...sourceFiles].map(([path, bytes]) => [path, Buffer.from(bytes)]));
		applyVariant(files, scenario.variant);
		assertVariantPrerequisites(sourceFiles, files, scenario);
		for (const path of exactBundlePaths) {
			writeExactFile(join(bundleRoot, path), files.get(path), 0o644);
		}
		const subject = Buffer.from(`setTimeout(() => process.exit(70), 15_000).unref();\n${scenario.subject}`, "utf8");
		writeExactFile(join(targetRoot, "fixture/subject.mjs"), subject, 0o644);
		const bundleBefore = snapshotTree(bundleRoot);
		const targetBefore = snapshotTree(targetRoot);
		const run = spawnSync(
			node,
			["--test", "--test-reporter=tap", join(bundleRoot, "contract.test.mjs")],
			{
				cwd: targetRoot,
				env: {
					COUNTERSHAPE_TEST_SECRET: `must-not-reach-subject-${repeatProfile}`,
					HOME: home,
					HTTP_PROXY: repeatProfile === "PRIMARY" ? "http://127.0.0.1:1" : "http://127.0.0.1:2",
					LANG: "C", LC_ALL: "C", NODE_OPTIONS: "--no-warnings", NO_COLOR: "1",
					npm_config_registry: "https://ambient.invalid/", PATH: "/ambient/path/must/not/reach/subject",
					TMPDIR: runtimeRoot, TZ: "UTC", __CF_USER_TEXT_ENCODING: "must-not-reach-subject",
				},
				maxBuffer: 2 << 20,
				timeout: 20_000,
			},
		);
		assert.equal(run.error, undefined, `${scenario.id}: runner error`);
		assert.equal(run.signal, null, `${scenario.id}: runner signal`);
		assert.equal(run.status, scenario.exitCode, `${scenario.id}: exit code`);
		assert.equal(Buffer.isBuffer(run.stdout), true, `${scenario.id}: stdout bytes`);
		assert.equal(Buffer.isBuffer(run.stderr), true, `${scenario.id}: stderr bytes`);
		assert.equal(run.stderr.length, 0, `${scenario.id}: outer stderr`);
		assert.deepEqual(snapshotTree(bundleRoot), bundleBefore, `${scenario.id}: bundle changed`);
		assert.deepEqual(snapshotTree(targetRoot), targetBefore, `${scenario.id}: target changed`);
		assert.deepEqual(readdirSync(runtimeRoot), [], `${scenario.id}: runtime residue`);
		assert.deepEqual(readdirSync(home), [], `${scenario.id}: home residue`);
		return stableCLIRecord(scenario, run.stdout, run.stderr, run.status, run.signal);
	} finally {
		rmSync(root, { recursive: true, force: true });
		assert.equal(lstatExists(root), false, `${scenario.id}: capture root survived cleanup`);
	}
}

function applyVariant(files, variant) {
	if (variant === undefined) return;
	if (variant === "tampered-decision") {
		files.set("decision.json", Buffer.concat([files.get("decision.json"), Buffer.from("\n", "utf8")]));
		return;
	}
	if (variant === "malformed-decision") {
		files.set("decision.json", Buffer.from("{}\n", "utf8"));
		updateManifest(files, "decision.json");
		return;
	}
	if (variant === "failing-harness") {
		files.set("harness.mjs", Buffer.concat([
			files.get("harness.mjs"), Buffer.from('throw new Error("sanitized capture fixture");\n', "utf8"),
		]));
		updateManifest(files, "harness.mjs");
		return;
	}
	throw new Error(`unknown capture variant ${variant}`);
}

function updateManifest(files, changedPath) {
	const manifestBytes = files.get("manifest.json");
	assert.equal(manifestBytes.at(-1), 0x0a);
	const manifest = JSON.parse(manifestBytes.subarray(0, -1).toString("utf8"));
	const entry = manifest.files.find((candidate) => candidate.path === changedPath);
	assert.notEqual(entry, undefined, `manifest omits ${changedPath}`);
	const content = files.get(changedPath);
	entry.byte_count = content.length;
	entry.byte_sha256 = rawSHA256(content);
	files.set("manifest.json", Buffer.from(`${canonicalJSON(manifest)}\n`, "utf8"));
}

function assertVariantPrerequisites(sourceFiles, files, scenario) {
	if (scenario.variant === undefined) {
		for (const path of exactBundlePaths) assert.deepEqual(files.get(path), sourceFiles.get(path));
		return;
	}
	const changedPath = scenario.variant === "failing-harness" ? "harness.mjs" : "decision.json";
	assert.equal(files.get(changedPath).equals(sourceFiles.get(changedPath)), false, `${scenario.id}: changed file did not change`);
	if (scenario.variant === "failing-harness") {
		assert.equal(files.get(changedPath).subarray(0, sourceFiles.get(changedPath).length).equals(sourceFiles.get(changedPath)), true, `${scenario.id}: real harness prefix was not preserved`);
	}
	const manifestChanged = !files.get("manifest.json").equals(sourceFiles.get("manifest.json"));
	assert.equal(manifestChanged, scenario.variant !== "tampered-decision", `${scenario.id}: manifest repin prerequisite differs`);
	const entry = JSON.parse(decodeUTF8(files.get("manifest.json"), `${scenario.id}: manifest`)).files
		.find((candidate) => candidate.path === changedPath);
	assert.notEqual(entry, undefined, `${scenario.id}: manifest entry missing`);
	if (scenario.variant === "tampered-decision") {
		assert.notEqual(entry.byte_sha256, rawSHA256(files.get(changedPath)), `${scenario.id}: tamper was accidentally repinned`);
	} else {
		assert.equal(entry.byte_count, files.get(changedPath).length, `${scenario.id}: repinned byte count differs`);
		assert.equal(entry.byte_sha256, rawSHA256(files.get(changedPath)), `${scenario.id}: repinned digest differs`);
	}
}

function writeExactFile(path, bytes, mode) {
	const exact = Buffer.isBuffer(bytes) ? bytes : Buffer.from(bytes, "utf8");
	writeFileSync(path, exact, { flag: "wx", mode });
	const info = lstatSync(path);
	assert.equal(info.isFile(), true, `${path}: not a regular file after write`);
	assert.equal(info.isSymbolicLink(), false, `${path}: became a symlink`);
	assert.equal(info.nlink, 1, `${path}: unexpected link count`);
	assert.equal(info.mode & 0o777, mode, `${path}: mode drifted`);
	assert.deepEqual(readFileSync(path), exact, `${path}: bytes drifted`);
}

function lstatExists(path) {
	try {
		lstatSync(path);
		return true;
	} catch (error) {
		if (error?.code === "ENOENT") return false;
		throw error;
	}
}

function stableCLIRecord(scenario, stdoutBytes, stderrBytes, exitCode, signal) {
	assert.equal(Buffer.isBuffer(stdoutBytes), true, `${scenario.id}: stdout is not bytes`);
	assert.equal(Buffer.isBuffer(stderrBytes), true, `${scenario.id}: stderr is not bytes`);
	if (stderrBytes.length !== 0) throw new Error(`${scenario.id}: outer stderr is not empty`);
	if (signal !== null) throw new Error(`${scenario.id}: outer signal is not NONE`);
	if (!Number.isInteger(exitCode)) throw new Error(`${scenario.id}: outer exit is not an integer`);
	const exitClass = exitCode === 0 ? "ZERO" : "NONZERO";
	const wantPass = scenario.exitCode === 0;
	if (exitClass !== (wantPass ? "ZERO" : "NONZERO")) throw new Error(`${scenario.id}: outer exit class differs`);
	const stdout = decodeWire(stdoutBytes, `${scenario.id}: TAP stdout`);
	const parsed = parseTAP(stdout, scenario, wantPass);
	return {
		diagnostic: parsed.diagnostic,
		exit_class: exitClass,
		outer_signal: "NONE",
		outer_stderr_bytes: 0,
		outcome: scenario.outcome,
		reason: scenario.reason,
		scenario: scenario.id,
		setup_profile: scenario.setupProfile,
		tap_cancelled: parsed.cancelled,
		tap_fail: parsed.fail,
		tap_pass: parsed.pass,
		tap_skipped: parsed.skipped,
		tap_status: wantPass ? "PASS" : "FAIL",
		tap_suites: parsed.suites,
		tap_test_name: "Countershape selected-field contract",
		tap_tests: parsed.tests,
		tap_todo: parsed.todo,
	};
}

function parseTAP(stdout, scenario, wantPass) {
	if (!stdout.endsWith("\n") || stdout.endsWith("\n\n")) throw new Error(`${scenario.id}: TAP must have one terminal LF`);
	const lines = stdout.slice(0, -1).split("\n");
	let cursor = 0;
	const exact = (expected, label) => {
		if (cursor >= lines.length || lines[cursor] !== expected) throw new Error(`${scenario.id}: TAP ${label} shape differs`);
		cursor += 1;
	};
	const match = (pattern, label) => {
		if (cursor >= lines.length) throw new Error(`${scenario.id}: TAP ${label} is missing`);
		const found = pattern.exec(lines[cursor]);
		if (found === null) throw new Error(`${scenario.id}: TAP ${label} shape differs`);
		cursor += 1;
		return found;
	};
	exact("TAP version 13", "version");
	exact("# Subtest: Countershape selected-field contract", "subtest");
	exact(`${wantPass ? "ok" : "not ok"} 1 - Countershape selected-field contract`, "result");
	exact("  ---", "YAML start");
	match(/^  duration_ms: (?:0|[1-9][0-9]*)(?:\.[0-9]+)?$/u, "test duration");
	exact("  type: 'test'", "test type");
	if (!wantPass) {
		match(/^  location: '\/[^'\n]+\/bundle\/contract\.test\.mjs:[1-9][0-9]*:[1-9][0-9]*'$/u, "failure location");
		exact("  failureType: 'testCodeFailure'", "failure type");
		exact("  error: 'Countershape contract failed'", "failure message");
		exact("  code: 'ERR_TEST_FAILURE'", "failure code");
		exact("  stack: |-", "failure stack marker");
		match(/^    TestContext\.<anonymous> \(file:\/\/\/[^'\n]+\/bundle\/contract\.test\.mjs:[1-9][0-9]*:[1-9][0-9]*\)$/u, "failure stack entry");
		while (cursor < lines.length && /^    (?:async )?[A-Za-z][^\n]* \(node:(?:async_hooks|internal\/test_runner\/(?:test|harness)):[1-9][0-9]*:[1-9][0-9]*\)$/u.test(lines[cursor])) {
			cursor += 1;
		}
	}
	exact("  ...", "YAML end");
	const diagnostic = `# COUNTERSHAPE_RESULT_V1|${scenario.outcome}|${scenario.reason}`;
	exact(diagnostic, "machine diagnostic");
	exact("1..1", "plan");
	const tests = summaryCount(lines, () => cursor++, cursor, "tests", 1, scenario.id);
	const suites = summaryCount(lines, () => cursor++, cursor, "suites", 0, scenario.id);
	const pass = summaryCount(lines, () => cursor++, cursor, "pass", wantPass ? 1 : 0, scenario.id);
	const fail = summaryCount(lines, () => cursor++, cursor, "fail", wantPass ? 0 : 1, scenario.id);
	const cancelled = summaryCount(lines, () => cursor++, cursor, "cancelled", 0, scenario.id);
	const skipped = summaryCount(lines, () => cursor++, cursor, "skipped", 0, scenario.id);
	const todo = summaryCount(lines, () => cursor++, cursor, "todo", 0, scenario.id);
	if (cursor >= lines.length || !/^# duration_ms (?:0|[1-9][0-9]*)(?:\.[0-9]+)?$/u.test(lines[cursor])) {
		throw new Error(`${scenario.id}: TAP total duration shape differs`);
	}
	cursor += 1;
	if (cursor !== lines.length) throw new Error(`${scenario.id}: TAP has an unrecognized trailing line`);
	return { cancelled, diagnostic, fail, pass, skipped, suites, tests, todo };
}

function summaryCount(lines, advance, cursor, name, expected, scenarioID) {
	if (cursor >= lines.length || lines[cursor] !== `# ${name} ${expected}`) {
		throw new Error(`${scenarioID}: TAP ${name} summary differs`);
	}
	advance();
	return expected;
}

function snapshotTree(root) {
	const rows = [];
	const visit = (absolute, relative) => {
		const info = lstatSync(absolute);
		if (info.isSymbolicLink()) throw new Error(`snapshot encountered symlink ${relative}`);
		if (info.isDirectory()) {
			rows.push(`${relative || "."}|directory|${info.mode & 0o777}|${info.nlink}`);
			for (const name of readdirSync(absolute).sort(unsignedByteOrder)) {
				visit(join(absolute, name), relative === "" ? name : `${relative}/${name}`);
			}
			return;
		}
		assert.equal(info.isFile(), true, `snapshot encountered special entry ${relative}`);
		const bytes = readFileSync(absolute);
		rows.push(`${relative}|file|${info.mode & 0o777}|${info.nlink}|${bytes.length}|${rawSHA256(bytes)}`);
	};
	visit(root, "");
	return rows;
}

function renderMarkdown(source, width) {
	assert.equal(widths.includes(width), true, "unsupported capture width");
	assert.equal(typeof source, "string");
	assert.equal(source.endsWith("\n"), true, "README must end in LF");
	assert.equal(source.endsWith("\n\n"), false, "README must have exactly one terminal LF");
	assert.equal(source.includes("\r"), false, "README must not contain CR");
	assert.equal(hasActiveControl(source), false, "README contains an active control byte");
	const output = [];
	let fenced = false;
	for (const line of source.slice(0, -1).split("\n")) {
		if (line === "```text") {
			assert.equal(fenced, false, "nested text fence");
			fenced = true;
			continue;
		}
		if (line === "```") {
			assert.equal(fenced, true, "unmatched text fence");
			fenced = false;
			continue;
		}
		if (fenced) {
			output.push(...hardWrap(line, "    ", width));
			continue;
		}
		if (line === "") {
			output.push("");
			continue;
		}
		let match;
		if ((match = /^(#{1,2}) (.+)$/u.exec(line)) !== null) {
			const title = inlineText(match[2]);
			const renderedTitle = match[1] === "#" ? title.toUpperCase() : title;
			const lines = wrapWords(renderedTitle, "", "", width);
			output.push(...lines);
			const underline = match[1] === "#" ? "=" : "-";
			output.push(underline.repeat(Math.min(width, Math.max(...lines.map(codepointLength)))));
			continue;
		}
		if (/^#/u.test(line)) throw new Error(`unsupported heading Markdown: ${line}`);
		if ((match = /^> (.+)$/u.exec(line)) !== null) {
			output.push(...wrapWords(inlineText(match[1]), "! ", "! ", width));
			continue;
		}
		if ((match = /^- (.+)$/u.exec(line)) !== null) {
			output.push(...wrapWords(inlineText(match[1]), "- ", "  ", width));
			continue;
		}
		if ((match = /^  ([1-9][0-9]*)\. (.+)$/u.exec(line)) !== null) {
			const prefix = `  ${match[1]}. `;
			output.push(...wrapWords(inlineText(match[2]), prefix, " ".repeat(codepointLength(prefix)), width));
			continue;
		}
		if (/^\s/u.test(line)) throw new Error(`unsupported indented Markdown line: ${line}`);
		output.push(...wrapWords(inlineText(line), "", "", width));
	}
	assert.equal(fenced, false, "unterminated text fence");
	const rendered = `${output.join("\n")}\n`;
	assert.equal(rendered.split("\n").every((line) => codepointLength(line) <= width), true);
	assert.equal(hasActiveControl(rendered), false);
	return rendered;
}

function inlineText(text) {
	let marker = null;
	let rendered = "";
	for (let index = 0; index < text.length;) {
		if (text.startsWith("**", index)) {
			if (marker === "`") throw new Error("nested inline Markdown is unsupported");
			marker = marker === "**" ? null : "**";
			index += 2;
			continue;
		}
		if (text[index] === "`") {
			if (marker === "**") throw new Error("nested inline Markdown is unsupported");
			marker = marker === "`" ? null : "`";
			rendered += "`";
			index += 1;
			continue;
		}
		if (text[index] === "*") throw new Error("single-star inline Markdown is unsupported");
		rendered += text[index];
		index += 1;
	}
	if (marker !== null) throw new Error("unbalanced inline Markdown marker");
	return rendered;
}

function wrapWords(text, firstPrefix, continuationPrefix, width) {
	const words = text.trim().split(/[ \t]+/u);
	assert.equal(words.length > 0 && words.every((word) => word !== ""), true);
	const lines = [];
	let prefix = firstPrefix;
	let content = "";
	const flush = () => {
		if (content !== "") lines.push(`${prefix}${content}`);
		prefix = continuationPrefix;
		content = "";
	};
	for (const original of words) {
		let word = original;
		for (;;) {
			const separator = content === "" ? "" : " ";
			const capacity = width - codepointLength(prefix) - codepointLength(content) - codepointLength(separator);
			if (codepointLength(word) <= capacity) {
				content += `${separator}${word}`;
				break;
			}
			if (content !== "") {
				flush();
				continue;
			}
			const prefixCapacity = width - codepointLength(prefix);
			assert.equal(prefixCapacity > 0, true, "capture prefix consumes width");
			const codepoints = [...word];
			lines.push(`${prefix}${codepoints.slice(0, prefixCapacity).join("")}`);
			prefix = continuationPrefix;
			word = codepoints.slice(prefixCapacity).join("");
		}
	}
	flush();
	return lines;
}

function hardWrap(text, prefix, width) {
	const capacity = width - codepointLength(prefix);
	assert.equal(capacity > 0, true);
	const codepoints = [...text];
	if (codepoints.length === 0) return [prefix];
	const lines = [];
	for (let offset = 0; offset < codepoints.length; offset += capacity) {
		lines.push(`${prefix}${codepoints.slice(offset, offset + capacity).join("")}`);
	}
	return lines;
}

function selfTest() {
	const sample = "# Example 😀\n\n> **WARNING.** `alpha` beta\n\n- one two three\n  1. ordered\n\n```text\ncopy this deliberately long command with one scalar-sliced-token-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\n```\n";
	for (const width of widths) {
		const left = renderMarkdown(sample, width);
		const right = renderMarkdown(sample, width);
		assert.equal(left, right);
		assert.equal(left.includes("\x1b"), false);
		assert.equal(left.split("\n").every((line) => codepointLength(line) <= width), true);
		assert.equal(left.includes("WARNING."), true, "bold text remains visible");
		assert.equal(left.includes("**"), false, "bold authoring markers leaked into terminal capture");
		assert.equal(left.includes("`alpha`"), true, "code markers remain visible");
	}
	assert.throws(() => renderMarkdown(sample, 59), /unsupported capture width/u);
	assert.throws(() => renderMarkdown("# Missing LF", 60), /end in LF/u);
	assert.throws(() => renderMarkdown("# Doubled LF\n\n", 60), /exactly one terminal LF/u);
	assert.throws(() => renderMarkdown("# Bad\n\x1b[31mred\n", 60), /active control/u);
	assert.throws(() => renderMarkdown("# Bad\n\ttab\n", 60), /active control/u);
	assert.throws(() => renderMarkdown("# Bad\n\u0085next\n", 60), /active control/u);
	assert.throws(() => renderMarkdown("# Bad\n\u202eright-to-left\n", 60), /active control/u);
	assert.throws(() => renderMarkdown("### Unsupported\n", 60), /unsupported heading/u);
	assert.throws(() => renderMarkdown("# `Unbalanced\n", 60), /unbalanced inline/u);
	assert.throws(() => renderMarkdown("# **Nested `code`**\n", 60), /nested inline/u);
	assert.throws(() => decodeUTF8(Buffer.from([0xff]), "selftest"), /invalid UTF-8/u);
	assert.throws(() => decodeUTF8(Buffer.from([0xef, 0xbb, 0xbf, 0x61]), "selftest"), /BOM/u);
	const scalarToken = `${"😀漢e\u0301".repeat(700)}x`;
	const scalarLines = wrapWords(scalarToken, "", "", 60);
	assert.equal(scalarLines.every((line) => codepointLength(line) <= 60), true);
	assert.equal(scalarLines.join(""), scalarToken, "scalar wrapping normalized or split content");
	const maximumEntrypoint = "x".repeat(4096);
	const entrypointLines = wrapWords(maximumEntrypoint, "", "", 60);
	assert.equal(entrypointLines.join(""), maximumEntrypoint, "maximum entrypoint was truncated");
	for (const length of [59, 60, 61]) {
		const boundary = "y".repeat(length);
		assert.equal(wrapWords(boundary, "", "", 60).join(""), boundary, `capacity ${length} drifted`);
	}

	const passScenario = {
		id: "selftest-pass", outcome: "CONFORMS", reason: "NONE", exitCode: 0, setupProfile: "SELFTEST_PASS",
	};
	const passLeft = stableCLIRecord(
		passScenario, Buffer.from(tapSample(true, "1.25", "/first/root", "CONFORMS", "NONE")), Buffer.alloc(0), 0, null,
	);
	const passRight = stableCLIRecord(
		passScenario, Buffer.from(tapSample(true, "99.5", "/second/root", "CONFORMS", "NONE")), Buffer.alloc(0), 0, null,
	);
	assert.deepEqual(passLeft, passRight, "TAP timing normalization drifted");
	const failScenario = {
		id: "selftest-fail", outcome: "CONTRADICTS", reason: "PREDICATE_MISMATCH", exitCode: 1,
		setupProfile: "SELFTEST_FAIL",
	};
	const failure = tapSample(false, "2.5", "/private/first root", "CONTRADICTS", "PREDICATE_MISMATCH");
	const failLeft = stableCLIRecord(failScenario, Buffer.from(failure), Buffer.alloc(0), 1, null);
	const failRight = stableCLIRecord(
		failScenario,
		Buffer.from(tapSample(false, "200.75", "/private/second-root", "CONTRADICTS", "PREDICATE_MISMATCH")),
		Buffer.alloc(0), 9, null,
	);
	assert.deepEqual(failLeft, failRight, "TAP path or timing normalization drifted");
	assert.throws(() => stableCLIRecord(passScenario, Buffer.from(passLeftWireWithoutDiagnostic()), Buffer.alloc(0), 0, null));
	assert.throws(() => stableCLIRecord(passScenario, Buffer.from(failure), Buffer.alloc(0), 0, null));
	assert.throws(() => stableCLIRecord(passScenario, Buffer.from(tapSample(true, "1", "/x", "CONFORMS", "NONE").replace("1..1\n", "# COUNTERSHAPE_RESULT_V1|CONFORMS|NONE\n1..1\n")), Buffer.alloc(0), 0, null));
	assert.throws(() => stableCLIRecord(passScenario, Buffer.from(tapSample(true, "1", "/x", "CONFORMS", "NONE").replace("  type: 'test'\n", "  type: 'test'\n  invented: true\n")), Buffer.alloc(0), 0, null));
	assert.throws(() => stableCLIRecord(passScenario, Buffer.from(tapSample(true, "1", "/x", "CONFORMS", "NONE").replace("# tests 1\n", "# tests 2\n")), Buffer.alloc(0), 0, null));
	assert.throws(() => stableCLIRecord(passScenario, Buffer.from(`${tapSample(true, "1", "/x", "CONFORMS", "NONE")}# extra\n`), Buffer.alloc(0), 0, null));
	assert.throws(() => stableCLIRecord(passScenario, Buffer.from([0xff]), Buffer.alloc(0), 0, null), /invalid UTF-8/u);
	for (const forbidden of ["\r", "\0", "\x1b", "\u0085"]) {
		assert.throws(() => stableCLIRecord(passScenario, Buffer.from(tapSample(true, "1", "/x", "CONFORMS", "NONE").replace("TAP", `${forbidden}TAP`)), Buffer.alloc(0), 0, null), /control/u);
	}
	assert.throws(() => stableCLIRecord(passScenario, Buffer.from(tapSample(true, "1", "/x", "CONFORMS", "NONE")), Buffer.from("unexpected"), 0, null), /stderr/u);
	assert.throws(() => stableCLIRecord(passScenario, Buffer.from(tapSample(true, "1", "/x", "CONFORMS", "NONE")), Buffer.alloc(0), 0, "SIGTERM"), /signal/u);
	assert.throws(() => stableCLIRecord(passScenario, Buffer.from(tapSample(true, "1", "/x", "CONFORMS", "NONE")), Buffer.alloc(0), 1, null), /exit class/u);
	const cleanupRoot = mkdtempSync(join(realpathSync(tmpdir()), "countershape-a2-cleanup-selftest-"));
	try {
		mkdirSync(join(cleanupRoot, "partial"), { mode: 0o700 });
		throw new Error("deliberate capture failure");
	} catch (error) {
		assert.match(error.message, /deliberate capture failure/u);
	} finally {
		rmSync(cleanupRoot, { recursive: true, force: true });
	}
	assert.equal(lstatExists(cleanupRoot), false, "capture-failure cleanup left a root");
}

function tapSample(pass, duration, root, outcome, reason) {
	const result = pass ? "ok" : "not ok";
	const yaml = pass
		? `  duration_ms: ${duration}\n  type: 'test'\n`
		: `  duration_ms: ${duration}\n  type: 'test'\n  location: '${root}/bundle/contract.test.mjs:283:1'\n  failureType: 'testCodeFailure'\n  error: 'Countershape contract failed'\n  code: 'ERR_TEST_FAILURE'\n  stack: |-\n    TestContext.<anonymous> (file://${root}/bundle/contract.test.mjs:327:44)\n    async Test.run (node:internal/test_runner/test:1125:7)\n    async startSubtestAfterBootstrap (node:internal/test_runner/harness:358:3)\n`;
	return `TAP version 13\n# Subtest: Countershape selected-field contract\n${result} 1 - Countershape selected-field contract\n  ---\n${yaml}  ...\n# COUNTERSHAPE_RESULT_V1|${outcome}|${reason}\n1..1\n# tests 1\n# suites 0\n# pass ${pass ? 1 : 0}\n# fail ${pass ? 0 : 1}\n# cancelled 0\n# skipped 0\n# todo 0\n# duration_ms ${duration}\n`;
}

function passLeftWireWithoutDiagnostic() {
	return tapSample(true, "1", "/x", "CONFORMS", "NONE").replace("# COUNTERSHAPE_RESULT_V1|CONFORMS|NONE\n", "");
}

function strictBase64(text) {
	assert.match(text, /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/u);
	const bytes = Buffer.from(text, "base64");
	assert.equal(bytes.toString("base64"), text);
	return bytes;
}

function canonicalJSON(value) {
	if (value === null || typeof value === "boolean" || typeof value === "string") return JSON.stringify(value);
	if (typeof value === "number") {
		if (!Number.isSafeInteger(value) || Object.is(value, -0)) throw new Error("noncanonical integer");
		return String(value);
	}
	if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
	if (typeof value !== "object") throw new Error("unsupported canonical value");
	const keys = Object.keys(value).sort(unsignedByteOrder);
	return `{${keys.map((key) => `${JSON.stringify(key)}:${canonicalJSON(value[key])}`).join(",")}}`;
}

function unsignedByteOrder(left, right) {
	return Buffer.compare(Buffer.from(left, "utf8"), Buffer.from(right, "utf8"));
}

function rawSHA256(bytes) {
	return `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
}

function typedDigest(kind, bytes) {
	return `sha256:${createHash("sha256").update(`countershape/v1/${kind}\0`, "utf8").update(bytes).digest("hex")}`;
}

function decodeUTF8(bytes, label) {
	assert.equal(Buffer.isBuffer(bytes), true, `${label}: input is not bytes`);
	if (bytes.length >= 3 && bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf) {
		throw new Error(`${label}: UTF-8 BOM is forbidden`);
	}
	let text;
	try {
		text = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
	} catch {
		throw new Error(`${label}: invalid UTF-8`);
	}
	if (!Buffer.from(text, "utf8").equals(bytes)) throw new Error(`${label}: UTF-8 round trip differs`);
	return text;
}

function decodeWire(bytes, label) {
	const text = decodeUTF8(bytes, label);
	if (hasActiveControl(text)) throw new Error(`${label}: forbidden control code point`);
	return text;
}

function codepointLength(text) {
	return [...text].length;
}

function hasActiveControl(text) {
	return /[\u0000-\u0009\u000b-\u001f\u007f-\u009f\u200e\u200f\u2028\u2029\u202a-\u202e\u2066-\u2069\ufeff]/u.test(text);
}
