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
const FILE_ORDER = Object.freeze([
  "README.md",
  "contract.test.mjs",
  "decision.json",
  "fixture.json",
  "harness.mjs",
]);

const SOURCES = Object.freeze({
  "README.md": `# Countershape executable planning fixture

This six-file Node-core bundle is a schema and validator fixture, not a shipped emitter or portability receipt.
It launches one local child process and checks one exact CLI stdout-byte tuple.
Integrity checks detect changed declared bytes; they do not establish authorship, authenticity, confidentiality, containment, or coordinated-replacement resistance.
`,
  "contract.test.mjs": `import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import test from "node:test";

const base = new URL("./", import.meta.url);
const manifest = JSON.parse(await readFile(new URL("manifest.json", base), "utf8"));
for (const entry of manifest.files) {
  const bytes = await readFile(new URL(entry.path, base));
  assert.equal(bytes.length, entry.byte_count);
  assert.equal("sha256:" + createHash("sha256").update(bytes).digest("hex"), entry.byte_sha256);
}

const { evaluate, runFixture } = await import("./harness.mjs");
const decision = JSON.parse(await readFile(new URL("decision.json", base), "utf8"));
const fixture = JSON.parse(await readFile(new URL("fixture.json", base), "utf8"));

test("executable planning fixture evaluates one exact tuple", () => {
  const observed = runFixture(fixture);
  assert.equal(evaluate(decision, observed), "CONFORMS");
});
`,
  "decision.json": "{\"kind\":\"CompiledDecision\",\"predicate\":{\"allowed_tuples\":[{\"fields\":[{\"field_id\":\"cli.stdout.bytes\",\"value\":{\"base64\":\"b2sK\",\"tag\":\"BYTES\"}}]}],\"kind\":\"one-of-exact/v1\",\"selected_fields\":[\"cli.stdout.bytes\"]},\"schema_version\":\"countershape-contract/v1\"}\n",
  "fixture.json": "{\"argv\":[\"--subject\"],\"entrypoint\":\"harness.mjs\",\"expected_stdout_base64\":\"b2sK\",\"kind\":\"PortableFixture\",\"schema_version\":\"countershape-contract/v1\",\"source_profile\":\"PLANNING_EXECUTABLE_FIXTURE_V1\"}\n",
  "harness.mjs": `import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const selfPath = fileURLToPath(import.meta.url);
if (process.argv[1] === selfPath && process.argv[2] === "--subject") {
  process.stdout.write(Buffer.from("b2sK", "base64"));
}

export function runFixture(fixture) {
  if (fixture.entrypoint !== "harness.mjs" || fixture.expected_stdout_base64 !== "b2sK") {
    throw new Error("fixture profile mismatch");
  }
  const child = spawnSync(process.execPath, [selfPath, ...fixture.argv], {
    encoding: null,
    env: { LANG: "C", TZ: "UTC" },
    maxBuffer: 65536,
    timeout: 5000,
    windowsHide: true,
  });
  if (child.error) throw child.error;
  if (child.signal !== null || child.status !== 0) throw new Error("subject did not exit cleanly");
  return {
    fields: [{
      field_id: "cli.stdout.bytes",
      value: { base64: child.stdout.toString("base64"), tag: "BYTES" },
    }],
  };
}

export function evaluate(decision, observed) {
  const encoded = JSON.stringify(observed);
  return decision.predicate.allowed_tuples.some((tuple) => JSON.stringify(tuple) === encoded)
    ? "CONFORMS"
    : "CONTRADICTS";
}
`,
});

function rawDigest(bytes) {
  return `sha256:${crypto.createHash("sha256").update(bytes).digest("hex")}`;
}

function fileEntry(filePath, text) {
  const bytes = Buffer.from(text, "utf8");
  return {
    path: filePath,
    mode: "100644",
    byte_count: bytes.length,
    byte_sha256: rawDigest(bytes),
    content_base64: bytes.toString("base64"),
  };
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

function buildExamples() {
  const files = FILE_ORDER.map((filePath) => fileEntry(filePath, SOURCES[filePath]));
  const manifest = {
    schema_version: "countershape-contract/v1",
    kind: "IntegrityManifest",
    manifest_version: "countershape-manifest/v1",
    files: files.map(({ path: filePath, mode, byte_count, byte_sha256 }) => ({
      path: filePath,
      mode,
      byte_count,
      byte_sha256,
    })),
  };
  files.push(fileEntry("manifest.json", `${JSON.stringify(manifest)}\n`));

  const exactTuple = {
    fields: [{
      field_id: "cli.stdout.bytes",
      value: { base64: "b2sK", tag: "BYTES" },
    }],
  };
  const bundle = {
    schema_version: "countershape/v1",
    kind: "ContractBundle",
    bundle_version: "node-core-contract-bundle/v1",
    decision_record_digest: "sha256:e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1e1",
    choicepoint_digest: "sha256:c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1",
    portable_source_digest: "sha256:d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1",
    portable_profile_digest: "sha256:f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1",
    decision_action: "ALLOW_OBSERVED",
    emitter_version: "node-exact-emitter/v1",
    source_profile: {
      runtime_family: "NODE",
      semantic_profile: "countershape-node-core-exact/v1",
      adapter_domain: "CLI",
      launch_profile: "NODE_REPO_SCRIPT_V1",
      subject_entrypoint: "harness.mjs",
      start_profile: "DIRECT_CHILD_V1",
      scope: "DECLARED_SOURCE_PROFILE_NOT_EXECUTION_EVIDENCE",
    },
    predicate: {
      kind: "one-of-exact/v1",
      scope: "EXACT_WITNESSED_STIMULUS",
      stimulus_digest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
      portable_profile_digest: "sha256:f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1",
      selected_fields: ["cli.stdout.bytes"],
      allowed_tuples: [exactTuple],
    },
    files,
    manifest_policy: "COVERS_OTHER_FIVE_EXCLUDES_SELF_V1",
    runtime_dependency_profile: "NODE_CORE_ONLY_V1",
    countershape_runtime_binding: "ABSENT_BY_CONSTRUCTION",
    package_registry_binding: "NONE",
    environment_profile: "EXPLICIT_SPARSE_ALLOWLIST_V1",
    external_service_binding: "NONE",
    determinism_profile: {
      contains_time: false,
      contains_random_id: false,
      contains_absolute_path: false,
      contains_candidate_identity: false,
      contains_declared_secret_value: false,
      contains_host_runtime_fact: false,
      contains_execution_receipt: false,
    },
    confidentiality_established: false,
  };
  const pinnedTreeIdentity = {
    schema_version: "countershape/v1",
    kind: "PinnedTreeIdentity",
    object_format: "sha1",
    commit_oid: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    tree_oid: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  };
  const target = {
    schema_version: "countershape/v1",
    kind: "ContractExecutionTarget",
    target_version: "contract-execution-target/v1",
    publication_scope: "IMMUTABLE_NONHEAD_PRESPAWN_AUTHORITY_V1",
    contract_bundle_digest: typedDigest("ContractBundle", bundle),
    source_binding: {
      portable_source_digest: bundle.portable_source_digest,
      portable_profile_digest: bundle.portable_profile_digest,
      source_profile_digest: typedDigest("ContractSourceProfile", bundle.source_profile),
    },
    tree_binding: {
      authority: "GIT_PIN_INSPECT_MATERIALIZE_V1",
      pinned_tree: {
        object_format: pinnedTreeIdentity.object_format,
        commit_oid: pinnedTreeIdentity.commit_oid,
        tree_oid: pinnedTreeIdentity.tree_oid,
        tree_identity_digest: typedDigest("PinnedTreeIdentity", pinnedTreeIdentity),
      },
      portable_tree_digest: "sha256:c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2c2",
      materialization_policy_digest: "sha256:c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3",
      materialization_manifest_digest: "sha256:c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4",
      execution_root_scope: "PRIVATE_PINNED_MATERIALIZATION_ONLY_V1",
    },
    attempt_binding: {
      purpose: "CONFORMANCE",
      attempt_artifact_digest: "sha256:e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5",
      instance_nonce: "0123456789abcdef0123456789abcdef",
      allocation_profile: "PRIVATE_FRESH_ROOT_V1",
      marker_ordering: "DURABLE_BEFORE_SPAWN",
    },
    runtime_binding: {
      authority: "ADMITTED_NODE_PROCESS_EXEC_PATH_V1",
      name: "node",
      version: "25.2.1",
      major: 25,
      os: "darwin",
      architecture: "arm64",
      executable_bytes_digest: "sha256:e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8",
      probe_program_digest: "sha256:e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9e9",
      child_resolution: "PROCESS_EXEC_PATH_EQUALS_ADMITTED_RUNTIME_V1",
    },
  };
  const finalizedRun = {
    schema_version: "countershape/v1",
    kind: "FinalizedContractRun",
    run_version: "finalized-contract-run/v1",
    publication_scope: "IMMUTABLE_NONHEAD_FINALIZED_RUN_V1",
    contract_execution_target_digest: typedDigest("ContractExecutionTarget", target),
    attempt_artifact_digest: target.attempt_binding.attempt_artifact_digest,
    lifecycle: {
      status: "FINALIZED",
      materialization_revalidation_digest: "sha256:f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2f2",
      process_result_digest: "sha256:f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3f3",
      teardown_result_digest: "sha256:f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4f4",
      orphan_check_digest: "sha256:f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5f5",
      finalization_marker_digest: "sha256:f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6",
    },
    terminal_disposition: {
      status: "ELIGIBLE_CLEAN",
    },
    observation: {
      status: "PROJECTED",
      captured_observation_digest: "sha256:e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6",
      projection_result_digest: "sha256:e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7e7",
      observed_tuple: exactTuple,
    },
    standalone_scope: {
      scope: "ISOLATED_TARGET_INVENTORY_AND_CHILD_BINDINGS_V1",
      target_inventory_digest: "sha256:f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7",
      child_bindings_digest: "sha256:f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8f8",
      import_resolution_digest: "sha256:f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9f9",
      service_bindings_digest: "sha256:fafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafa",
      target_inventory_countershape_source_present: false,
      target_inventory_countershape_dependency_present: false,
      target_import_resolution_reached_countershape: false,
      child_path_contains_countershape: false,
      countershape_service_binding_present: false,
      named_parent_secret_sentinels_inherited: false,
      host_wide_absence_established: false,
      network_denial_established: false,
      package_registry_denial_established: false,
      confidentiality_established: false,
    },
  };
  const execution = {
    schema_version: "countershape/v1",
    kind: "ContractExecution",
    execution_version: "contract-execution/v1",
    publication_scope: "IMMUTABLE_NONHEAD_EVIDENCE_V1",
    contract_execution_target_digest: typedDigest("ContractExecutionTarget", target),
    finalized_contract_run_digest: typedDigest("FinalizedContractRun", finalizedRun),
    result: {
      execution_class: "ELIGIBLE_OBSERVATION",
      conformance: "CONFORMS",
    },
    historical_execution_evidence_reused: false,
    choicepoint_freshened: false,
    study_head_advanced: false,
  };
  return {
    bundle,
    target,
    finalizedRun,
    execution,
    bundleBytes: Buffer.from(`${JSON.stringify(bundle, null, 2)}\n`, "utf8"),
    targetBytes: Buffer.from(`${JSON.stringify(target, null, 2)}\n`, "utf8"),
    finalizedRunBytes: Buffer.from(`${JSON.stringify(finalizedRun, null, 2)}\n`, "utf8"),
    executionBytes: Buffer.from(`${JSON.stringify(execution, null, 2)}\n`, "utf8"),
  };
}

function checkOrWrite(mode) {
  const built = buildExamples();
  if (mode === "write") {
    fs.writeFileSync(BUNDLE_FILE, built.bundleBytes);
    fs.writeFileSync(TARGET_FILE, built.targetBytes);
    fs.writeFileSync(RUN_FILE, built.finalizedRunBytes);
    fs.writeFileSync(EXECUTION_FILE, built.executionBytes);
    process.stdout.write(`P07 planning examples written; target ${built.execution.contract_execution_target_digest}\n`);
    return;
  }
  assert.deepEqual(fs.readFileSync(BUNDLE_FILE), built.bundleBytes, "ContractBundle example drifted from its deterministic generator");
  assert.deepEqual(fs.readFileSync(TARGET_FILE), built.targetBytes, "ContractExecutionTarget example drifted from its deterministic generator");
  assert.deepEqual(fs.readFileSync(RUN_FILE), built.finalizedRunBytes, "FinalizedContractRun example drifted from its deterministic generator");
  assert.deepEqual(fs.readFileSync(EXECUTION_FILE), built.executionBytes, "ContractExecution example drifted from its deterministic generator");
  process.stdout.write(`P07 planning examples: exact (${built.execution.contract_execution_target_digest})\n`);
}

function materialize(root, bundle) {
  fs.mkdirSync(root, { recursive: true, mode: 0o700 });
  for (const entry of bundle.files) {
    const bytes = Buffer.from(entry.content_base64, "base64");
    assert.equal(bytes.length, entry.byte_count);
    assert.equal(rawDigest(bytes), entry.byte_sha256);
    fs.writeFileSync(path.join(root, entry.path), bytes, { mode: 0o644, flag: "wx" });
  }
}

function runFixture(root) {
  return spawnSync(process.execPath, ["--test", "contract.test.mjs"], {
    cwd: root,
    encoding: "utf8",
    env: {
      HOME: root,
      TMPDIR: root,
      LANG: "C",
      TZ: "UTC",
      NO_COLOR: "1",
    },
    timeout: 10_000,
    windowsHide: true,
  });
}

function exercise() {
  const { bundle } = buildExamples();
  const parent = fs.mkdtempSync(path.join(os.tmpdir(), "countershape-p07-example-"));
  try {
    const clean = path.join(parent, "clean");
    materialize(clean, bundle);
    const passed = runFixture(clean);
    assert.equal(passed.status, 0, `executable planning fixture failed:\n${passed.stdout}\n${passed.stderr}`);
    assert.match(passed.stdout, /executable planning fixture evaluates one exact tuple/u);

    const tampered = path.join(parent, "tampered");
    materialize(tampered, bundle);
    const marker = path.join(parent, "tampered-harness-loaded");
    const hostileHarness = `import fs from "node:fs";\nfs.writeFileSync(${JSON.stringify(marker)}, "loaded");\n`;
    fs.writeFileSync(path.join(tampered, "harness.mjs"), hostileHarness, { mode: 0o644 });
    const refused = runFixture(tampered);
    assert.notEqual(refused.status, 0, "companion tampering unexpectedly passed");
    assert.equal(fs.existsSync(marker), false, "tampered companion loaded before manifest verification");
    process.stdout.write("P07 planning example: executable and intact-entrypoint companion tamper refused before harness load\n");
  } finally {
    fs.rmSync(parent, { recursive: true, force: true });
  }
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
