#!/usr/bin/env node

import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import process from "node:process";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(SCRIPT_DIR, "..");
const BUNDLE_FILE = path.join(ROOT, "spec/examples/v1/contract-bundle.valid.json");
const TARGET_FILE = path.join(ROOT, "spec/examples/v1/contract-execution-target.valid.json");
const RUN_FILE = path.join(ROOT, "spec/examples/v1/finalized-contract-run.valid.json");
const EXECUTION_FILE = path.join(ROOT, "spec/examples/v1/contract-execution.valid.json");

function rawDigest(bytes) {
  return `sha256:${crypto.createHash("sha256").update(bytes).digest("hex")}`;
}

function canonicalJSON(value) {
  if (value === null || typeof value === "boolean" || typeof value === "string") return JSON.stringify(value);
  if (typeof value === "number") {
    if (!Number.isSafeInteger(value) || Object.is(value, -0)) throw new Error("planning example contains a noncanonical number");
    return String(value);
  }
  if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
  if (!value || typeof value !== "object") throw new Error("planning example contains an unsupported value");
  const keys = Object.keys(value).sort((left, right) => Buffer.compare(Buffer.from(left), Buffer.from(right)));
  return `{${keys.map((key) => `${JSON.stringify(key)}:${canonicalJSON(value[key])}`).join(",")}}`;
}

function typedDigest(kind, value) {
  const hash = crypto.createHash("sha256");
  hash.update(Buffer.from(`countershape/v1/${kind}\0`, "utf8"));
  hash.update(Buffer.from(canonicalJSON(value), "utf8"));
  return `sha256:${hash.digest("hex")}`;
}

function readRuntimeBundle() {
  const bundleBytes = fs.readFileSync(BUNDLE_FILE);
  assert.equal(bundleBytes.at(-1), 0x0a, "runtime ContractBundle example must end in one LF");
  const bundle = JSON.parse(bundleBytes.toString("utf8"));
  assert.equal(bundle.schema_version, "countershape/v1");
  assert.equal(bundle.kind, "ContractBundle");
  assert.equal(bundle.bundle_version, "node-core-contract-bundle/v1");
  assert.equal(bundle.emitter_version, "node-exact-emitter/v1");
  assert.deepEqual(bundle.files?.map((entry) => entry.path), [
    "README.md",
    "contract.test.mjs",
    "decision.json",
    "fixture.json",
    "harness.mjs",
    "manifest.json",
  ], "runtime ContractBundle example must retain the exact six-file roster");
  assert.equal(bundle.source_profile?.subject_entrypoint, "fixture/subject.mjs");
  assert.equal(bundle.source_profile?.adapter_domain, "CLI");
  assert.equal(bundle.decision_action, "ALLOW_OBSERVED");
  assert.deepEqual(bundle.predicate?.selected_fields, ["cli.stdout.bytes"]);
  assert.equal(bundle.predicate?.allowed_tuples?.length, 1);
  assert.deepEqual(bundle.predicate.allowed_tuples[0], {
    fields: [{ field_id: "cli.stdout.bytes", value: { base64: "b2sK", tag: "BYTES" } }],
  });
  assert.equal(bundle.files[1].byte_count > 10_000, true, "contract entrypoint unexpectedly shrank to a planning toy");
  assert.equal(bundle.files[4].byte_count > 100_000, true, "runtime harness unexpectedly shrank to a planning toy");
  for (const entry of bundle.files) {
    const bytes = Buffer.from(entry.content_base64, "base64");
    assert.equal(bytes.toString("base64"), entry.content_base64, `${entry.path} uses noncanonical base64`);
    assert.equal(entry.mode, "100644", `${entry.path} mode drifted`);
    assert.equal(bytes.length, entry.byte_count, `${entry.path} byte_count drifted`);
    assert.equal(rawDigest(bytes), entry.byte_sha256, `${entry.path} byte digest drifted`);
  }
  return bundle;
}

function runModelProbe() {
  const go = process.env.COUNTERSHAPE_GO || "/opt/homebrew/bin/go";
  assert.equal(path.isAbsolute(go), true, "COUNTERSHAPE_GO must be absolute");
  const probe = spawnSync(go, [
    "test", "-mod=readonly", "-buildvcs=false", "-p=1", "-count=1", "-run", "^TestC1ExampleProbe$", "-v",
    "./internal/contractexec/model",
  ], {
    cwd: ROOT,
    encoding: "utf8",
    env: { ...process.env, COUNTERSHAPE_C1_EXAMPLE_PROBE: "1", GOMAXPROCS: "2" },
    timeout: 120_000,
    maxBuffer: 8 << 20,
  });
  assert.equal(probe.error, undefined, `C1 Go model probe could not run: ${probe.error?.message ?? "unknown"}`);
  assert.equal(probe.signal, null, `C1 Go model probe was signalled: ${probe.signal}`);
  assert.equal(probe.status, 0, `C1 Go model probe failed:\n${probe.stdout}\n${probe.stderr}`);
  assert.equal(probe.stderr, "", "C1 Go model probe wrote stderr");
  const frames = probe.stdout.split("\n").filter((line) => line.startsWith("COUNTERSHAPE_C1_MODEL_V1|"));
  assert.equal(frames.length, 1, "C1 Go model probe must emit exactly one frame");
  const parts = frames[0].split("|");
  assert.equal(parts.length, 7, "C1 Go model frame roster differs");
  const decode = (encoded, kind, digest) => {
    const exact = Buffer.from(encoded, "base64");
    assert.equal(exact.toString("base64"), encoded, `${kind} probe frame uses noncanonical base64`);
    const value = JSON.parse(exact.toString("utf8"));
    assert.equal(canonicalJSON(value), exact.toString("utf8"), `${kind} probe frame is not exact canonical JSON`);
    assert.equal(typedDigest(kind, value), digest, `${kind} probe digest differs`);
    return { exact, value, digest };
  };
  return {
    target: decode(parts[1], "ContractExecutionTarget", parts[2]),
    run: decode(parts[3], "FinalizedContractRun", parts[4]),
    execution: decode(parts[5], "ContractExecution", parts[6]),
  };
}

function buildExamples() {
  const bundle = readRuntimeBundle();
  const model = runModelProbe();
  const target = model.target.value;
  const finalizedRun = model.run.value;
  const execution = model.execution.value;
  assert.equal(target.contract_bundle_digest, typedDigest("ContractBundle", bundle), "target bundle join differs");
  assert.equal(finalizedRun.contract_execution_target_digest, model.target.digest, "run target join differs");
  assert.equal(finalizedRun.attempt_artifact_digest, target.attempt_binding.attempt_artifact_digest, "run attempt join differs");
  assert.equal(execution.contract_execution_target_digest, model.target.digest, "execution target join differs");
  assert.equal(execution.finalized_contract_run_digest, model.run.digest, "execution run join differs");
  assert.equal(execution.result, "CONFORMS", "positive C1 model example must derive CONFORMS");
  return {
    bundle,
    target,
    finalizedRun,
    execution,
    targetBytes: Buffer.from(`${JSON.stringify(target, null, 2)}\n`, "utf8"),
    finalizedRunBytes: Buffer.from(`${JSON.stringify(finalizedRun, null, 2)}\n`, "utf8"),
    executionBytes: Buffer.from(`${JSON.stringify(execution, null, 2)}\n`, "utf8"),
  };
}

function checkOrWrite(mode) {
  const built = buildExamples();
  if (mode === "write") {
    fs.writeFileSync(TARGET_FILE, built.targetBytes);
    fs.writeFileSync(RUN_FILE, built.finalizedRunBytes);
    fs.writeFileSync(EXECUTION_FILE, built.executionBytes);
    process.stdout.write(`P07 C1 semantic examples written; target ${built.execution.contract_execution_target_digest}\n`);
    return;
  }
  assert.deepEqual(fs.readFileSync(TARGET_FILE), built.targetBytes, "ContractExecutionTarget example drifted from the Go model");
  assert.deepEqual(fs.readFileSync(RUN_FILE), built.finalizedRunBytes, "FinalizedContractRun example drifted from the Go model");
  assert.deepEqual(fs.readFileSync(EXECUTION_FILE), built.executionBytes, "ContractExecution example drifted from the Go model");
  process.stdout.write(`P07 C1 semantic examples: Go-model exact (${built.execution.contract_execution_target_digest})\n`);
}

function assertExactDirectoryMode(directory, expected) {
  const metadata = fs.lstatSync(directory);
  assert.equal(metadata.isSymbolicLink(), false, `${directory} must not be a symlink`);
  assert.equal(metadata.isDirectory(), true, `${directory} must be a directory`);
  assert.equal(metadata.mode & 0o7777, expected, `${directory} mode drifted`);
}

function ensureExactDirectory(directory, expected = 0o700) {
  try {
    const prior = fs.lstatSync(directory);
    assert.equal(prior.isSymbolicLink(), false, `${directory} must not be a symlink`);
    assert.equal(prior.isDirectory(), true, `${directory} must be a directory`);
  } catch (error) {
    if (error.code !== "ENOENT") throw error;
    fs.mkdirSync(directory, { recursive: true, mode: expected });
  }
  const prior = fs.lstatSync(directory);
  assert.equal(prior.isSymbolicLink(), false, `${directory} must not be a symlink`);
  assert.equal(prior.isDirectory(), true, `${directory} must be a directory`);
  assert.equal(Number.isInteger(fs.constants.O_DIRECTORY), true, "O_DIRECTORY must be available");
  assert.equal(Number.isInteger(fs.constants.O_NOFOLLOW), true, "O_NOFOLLOW must be available");
  const descriptor = fs.openSync(
    directory,
    fs.constants.O_RDONLY | fs.constants.O_DIRECTORY | fs.constants.O_NOFOLLOW,
  );
  try {
    const opened = fs.fstatSync(descriptor);
    assert.equal(opened.isDirectory(), true, `${directory} opened target must be a directory`);
    assert.equal(opened.dev, prior.dev, `${directory} directory device changed`);
    assert.equal(opened.ino, prior.ino, `${directory} directory inode changed`);
    fs.fchmodSync(descriptor, expected);
    assert.equal(fs.fstatSync(descriptor).mode & 0o7777, expected, `${directory} opened mode drifted`);
  } finally {
    fs.closeSync(descriptor);
  }
  assertExactDirectoryMode(directory, expected);
}

function assertExactRegularMode(file, expected) {
  const metadata = fs.lstatSync(file);
  assert.equal(metadata.isSymbolicLink(), false, `${file} must not be a symlink`);
  assert.equal(metadata.isFile(), true, `${file} must be a regular file`);
  assert.equal(metadata.mode & 0o7777, expected, `${file} mode drifted`);
}

function writeExactFixtureFile(file, bytes, expected = 0o644, replace = false) {
  let prior;
  if (replace) {
    prior = fs.lstatSync(file);
    assert.equal(prior.isSymbolicLink(), false, `${file} replacement target must not be a symlink`);
    assert.equal(prior.isFile(), true, `${file} replacement target must be a regular file`);
  }
  assert.equal(Number.isInteger(fs.constants.O_NOFOLLOW), true, "O_NOFOLLOW must be available");
  const noFollow = fs.constants.O_NOFOLLOW;
  const flags = replace
    ? fs.constants.O_WRONLY | noFollow
    : fs.constants.O_WRONLY | fs.constants.O_CREAT | fs.constants.O_EXCL | noFollow;
  const descriptor = fs.openSync(file, flags, expected);
  try {
    const opened = fs.fstatSync(descriptor);
    assert.equal(opened.isFile(), true, `${file} opened target must be regular`);
    if (prior) {
      assert.equal(opened.dev, prior.dev, `${file} replacement device changed`);
      assert.equal(opened.ino, prior.ino, `${file} replacement inode changed`);
    }
    if (replace) fs.ftruncateSync(descriptor, 0);
    fs.writeFileSync(descriptor, bytes);
    fs.fchmodSync(descriptor, expected);
    fs.fsyncSync(descriptor);
    const finalOpened = fs.fstatSync(descriptor);
    assert.equal(finalOpened.mode & 0o7777, expected, `${file} opened mode drifted`);
  } finally {
    fs.closeSync(descriptor);
  }
  assertExactRegularMode(file, expected);
}

function assertBundleModes(root, bundle) {
  assertExactDirectoryMode(root, 0o700);
  for (const entry of bundle.files) assertExactRegularMode(path.join(root, entry.path), 0o644);
}

function materialize(root, bundle) {
  ensureExactDirectory(root);
  for (const entry of bundle.files) {
    const bytes = Buffer.from(entry.content_base64, "base64");
    assert.equal(bytes.length, entry.byte_count);
    assert.equal(rawDigest(bytes), entry.byte_sha256);
    assert.equal(entry.mode, "100644", `${entry.path} logical mode drifted`);
    writeExactFixtureFile(path.join(root, entry.path), bytes);
  }
  assertBundleModes(root, bundle);
}

function runFixture(bundleRoot, targetRoot, runtimeRoot, runnerHome) {
  return spawnSync(process.execPath, ["--test", "--test-reporter=tap", path.join(bundleRoot, "contract.test.mjs")], {
    cwd: targetRoot,
    encoding: "utf8",
    env: {
      COUNTERSHAPE_TEST_SECRET: "must-not-reach-subject",
      HOME: runnerHome,
      HTTP_PROXY: "http://127.0.0.1:1",
      LANG: "C",
      LC_ALL: "C",
      NODE_OPTIONS: "--no-warnings",
      TZ: "UTC",
      NO_COLOR: "1",
      PATH: "/ambient/path/must/not/reach/subject",
      TMPDIR: runtimeRoot,
      npm_config_registry: "https://ambient.invalid/",
    },
    argv0: process.execPath,
    timeout: 10_000,
    windowsHide: true,
    shell: false,
    maxBuffer: 4 << 20,
  });
}

function exerciseAtUmask(bundle, mask) {
  const previousUmask = process.umask(mask);
  let parent;
  try {
    parent = fs.mkdtempSync(path.join(os.tmpdir(), `countershape-p07-example-${mask.toString(8)}-`));
    fs.chmodSync(parent, 0o700);
    assertExactDirectoryMode(parent, 0o700);
    const target = path.join(parent, "prepared target");
    const targetFixture = path.join(target, "fixture");
    ensureExactDirectory(target);
    ensureExactDirectory(targetFixture);
    const subject = `setTimeout(() => process.exit(70), 15_000).unref();\n` +
      `const chunks = [];\n` +
      `process.stdin.on("data", (chunk) => chunks.push(Buffer.from(chunk)));\n` +
      `process.stdin.on("end", () => {\n` +
      `  if (!Buffer.concat(chunks).equals(Buffer.from("contract-input"))) process.exit(65);\n` +
      `  process.stdout.write("ok\\n");\n` +
      `});\n` +
      `process.stdin.resume();\n`;
    const subjectPath = path.join(targetFixture, "subject.mjs");
    writeExactFixtureFile(subjectPath, Buffer.from(subject, "utf8"));

    const cleanBundle = path.join(parent, "clean bundle");
    const cleanRuntime = path.join(parent, "clean runtime");
    const cleanHome = path.join(parent, "clean home");
    materialize(cleanBundle, bundle);
    ensureExactDirectory(cleanRuntime);
    ensureExactDirectory(cleanHome);
    const passed = runFixture(cleanBundle, target, cleanRuntime, cleanHome);
    assert.equal(passed.status, 0, `runtime ContractBundle example failed:\n${passed.stdout}\n${passed.stderr}`);
    assert.equal(passed.stderr, "", "conforming runtime ContractBundle wrote outer stderr");
    assert.match(passed.stdout, /# COUNTERSHAPE_RESULT_V1\|CONFORMS\|NONE\n/u);
    assert.equal((passed.stdout.match(/COUNTERSHAPE_RESULT_V1/gu) ?? []).length, 1);
    assertExactRegularMode(subjectPath, 0o644);
    assertBundleModes(cleanBundle, bundle);

    const tampered = path.join(parent, "tampered bundle");
    const tamperedRuntime = path.join(parent, "tampered runtime");
    const tamperedHome = path.join(parent, "tampered home");
    materialize(tampered, bundle);
    ensureExactDirectory(tamperedRuntime);
    ensureExactDirectory(tamperedHome);
    const marker = path.join(parent, "tampered-harness-loaded");
    const hostileHarness = `import fs from "node:fs";\nfs.writeFileSync(${JSON.stringify(marker)}, "loaded");\n`;
    writeExactFixtureFile(path.join(tampered, "harness.mjs"), Buffer.from(hostileHarness, "utf8"), 0o644, true);
    const refused = runFixture(tampered, target, tamperedRuntime, tamperedHome);
    assert.notEqual(refused.status, 0, "companion tampering unexpectedly passed");
    assert.equal(refused.stderr, "", "tamper refusal wrote outer stderr");
    assert.match(refused.stdout, /# COUNTERSHAPE_RESULT_V1\|TAMPER_DETECTED\|COMPANION_INTEGRITY_MISMATCH\n/u);
    assert.equal((refused.stdout.match(/COUNTERSHAPE_RESULT_V1/gu) ?? []).length, 1);
    assert.equal(fs.existsSync(marker), false, "tampered companion loaded before manifest verification");
    assertExactRegularMode(subjectPath, 0o644);
    assertBundleModes(tampered, bundle);
  } finally {
    try {
      if (parent !== undefined) fs.rmSync(parent, { recursive: true, force: true });
    } finally {
      process.umask(previousUmask);
    }
  }
}

function exercise() {
  const { bundle } = buildExamples();
  exerciseAtUmask(bundle, 0o077);
  exerciseAtUmask(bundle, 0o022);
  process.stdout.write("P07 planning example: real A2.2 bundle conformed and intact entrypoint refused companion tamper before harness load; caller umasks 077 and 022 restored exact 0700 directories and 0644 files\n");
}

const arguments_ = process.argv.slice(2);
if (arguments_.length !== 1 || !["--check", "--write", "--exercise"].includes(arguments_[0])) {
  process.stderr.write("usage: node tools/generate-p07-planning-example.mjs --check|--write|--exercise\n");
  process.exitCode = 2;
} else if (arguments_[0] === "--write") {
  checkOrWrite("write");
} else if (arguments_[0] === "--check") {
  checkOrWrite("check");
} else {
  checkOrWrite("check");
  exercise();
}
