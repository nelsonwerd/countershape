import { createHash } from "node:crypto";
import { closeSync, constants, fstatSync, lstatSync, openSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const RESULT_PREFIX = "COUNTERSHAPE_RESULT_V1|";
const bundleRoot = dirname(fileURLToPath(import.meta.url));
const MAX_MANIFEST_BYTES = 64 << 10;
const MAX_TOKENS = 1 << 14;
const MAX_MEMBERS = 1 << 10;
const MAX_DEPTH = 32;
const PROTECTED_PATHS = ["README.md", "contract.test.mjs", "decision.json", "fixture.json", "harness.mjs"];

class StageError extends Error {
  constructor(outcome, reason) {
    super(`${outcome}/${reason}`);
    this.outcome = outcome;
    this.reason = reason;
  }
}

function ownObject(entries) {
  const value = Object.create(null);
  for (const [name, member] of entries) {
    Object.defineProperty(value, name, { enumerable: true, value: member });
  }
  return value;
}

function validUTF8(bytes) {
  for (let index = 0; index < bytes.length;) {
    const first = bytes[index];
    if (first <= 0x7f) {
      index += 1;
      continue;
    }
    let count;
    if (first >= 0xc2 && first <= 0xdf) count = 2;
    else if (first === 0xe0 && bytes[index + 1] >= 0xa0 && bytes[index + 1] <= 0xbf) count = 3;
    else if (first >= 0xe1 && first <= 0xec) count = 3;
    else if (first === 0xed && bytes[index + 1] >= 0x80 && bytes[index + 1] <= 0x9f) count = 3;
    else if (first >= 0xee && first <= 0xef) count = 3;
    else if (first === 0xf0 && bytes[index + 1] >= 0x90 && bytes[index + 1] <= 0xbf) count = 4;
    else if (first >= 0xf1 && first <= 0xf3) count = 4;
    else if (first === 0xf4 && bytes[index + 1] >= 0x80 && bytes[index + 1] <= 0x8f) count = 4;
    else return false;
    if (index + count > bytes.length) return false;
    for (let offset = 1; offset < count; offset += 1) {
      if (bytes[index + offset] < 0x80 || bytes[index + offset] > 0xbf) return false;
    }
    index += count;
  }
  return true;
}

class CanonicalManifestParser {
  constructor(bytes) {
    if (!Buffer.isBuffer(bytes) || bytes.length === 0 || bytes.length > MAX_MANIFEST_BYTES || !validUTF8(bytes)) {
      throw new StageError("MALFORMED_CONTRACT", "CONTRACT_DATA_INVALID");
    }
    this.bytes = bytes;
    this.offset = 0;
    this.tokens = 0;
  }

  fail() {
    throw new StageError("MALFORMED_CONTRACT", "CONTRACT_DATA_INVALID");
  }

  token() {
    this.tokens += 1;
    if (this.tokens > MAX_TOKENS) this.fail();
  }

  take(byte) {
    if (this.bytes[this.offset] !== byte) this.fail();
    this.offset += 1;
    this.token();
  }

  hex(offset) {
    let value = 0;
    for (let index = 0; index < 4; index += 1) {
      const byte = this.bytes[offset + index];
      let digit;
      if (byte >= 0x30 && byte <= 0x39) digit = byte - 0x30;
      else if (byte >= 0x61 && byte <= 0x66) digit = byte - 0x61 + 10;
      else this.fail();
      value = (value << 4) | digit;
    }
    return value;
  }

  string() {
    this.take(0x22);
    const chunks = [];
    let rawStart = this.offset;
    while (this.offset < this.bytes.length) {
      const byte = this.bytes[this.offset];
      if (byte === 0x22) {
        if (this.offset > rawStart) chunks.push(this.bytes.subarray(rawStart, this.offset).toString("utf8"));
        this.offset += 1;
        return chunks.join("");
      }
      if (byte < 0x20) this.fail();
      if (byte !== 0x5c) {
        this.offset += byte < 0x80 ? 1 : byte < 0xe0 ? 2 : byte < 0xf0 ? 3 : 4;
        continue;
      }
      if (this.offset > rawStart) chunks.push(this.bytes.subarray(rawStart, this.offset).toString("utf8"));
      this.offset += 1;
      const escape = this.bytes[this.offset];
      this.offset += 1;
      const simple = new Map([
        [0x22, '"'], [0x5c, "\\"], [0x62, "\b"], [0x66, "\f"],
        [0x6e, "\n"], [0x72, "\r"], [0x74, "\t"],
      ]);
      if (simple.has(escape)) {
        chunks.push(simple.get(escape));
        rawStart = this.offset;
        continue;
      }
      if (escape !== 0x75 || this.offset + 4 > this.bytes.length) this.fail();
      const codepoint = this.hex(this.offset);
      this.offset += 4;
      const short = codepoint <= 0x07 || codepoint === 0x0b || (codepoint >= 0x0e && codepoint <= 0x1f);
      if (!short || this.bytes.subarray(this.offset - 4, this.offset).toString("ascii") !== codepoint.toString(16).padStart(4, "0")) {
        this.fail();
      }
      chunks.push(String.fromCodePoint(codepoint));
      rawStart = this.offset;
    }
    this.fail();
  }

  integer() {
    const start = this.offset;
    if (this.bytes[this.offset] === 0x2d) this.offset += 1;
    while (this.bytes[this.offset] >= 0x30 && this.bytes[this.offset] <= 0x39) this.offset += 1;
    this.token();
    const text = this.bytes.subarray(start, this.offset).toString("ascii");
    if (!/^-?(0|[1-9][0-9]*)$/.test(text) || text === "-0") this.fail();
    const value = Number(text);
    if (!Number.isSafeInteger(value)) this.fail();
    return value;
  }

  value(depth = 0) {
    if (depth > MAX_DEPTH) this.fail();
    const byte = this.bytes[this.offset];
    if (byte === 0x7b) return this.object(depth + 1);
    if (byte === 0x5b) return this.array(depth + 1);
    if (byte === 0x22) return this.string();
    if (byte === 0x2d || (byte >= 0x30 && byte <= 0x39)) return this.integer();
    for (const [literal, value] of [["null", null], ["true", true], ["false", false]]) {
      if (this.bytes.subarray(this.offset, this.offset + literal.length).toString("ascii") === literal) {
        this.offset += literal.length;
        this.token();
        return value;
      }
    }
    this.fail();
  }

  array(depth) {
    this.take(0x5b);
    if (this.bytes[this.offset] === 0x5d) {
      this.take(0x5d);
      return [];
    }
    const values = [];
    for (;;) {
      if (values.length >= MAX_MEMBERS) this.fail();
      values.push(this.value(depth));
      if (this.bytes[this.offset] === 0x2c) {
        this.take(0x2c);
        continue;
      }
      this.take(0x5d);
      return values;
    }
  }

  object(depth) {
    this.take(0x7b);
    if (this.bytes[this.offset] === 0x7d) {
      this.take(0x7d);
      return ownObject([]);
    }
    const entries = [];
    let previous = null;
    for (;;) {
      if (entries.length >= MAX_MEMBERS) this.fail();
      const name = this.string();
      if (previous !== null && Buffer.compare(Buffer.from(previous), Buffer.from(name)) >= 0) this.fail();
      previous = name;
      this.take(0x3a);
      entries.push([name, this.value(depth)]);
      if (this.bytes[this.offset] === 0x2c) {
        this.take(0x2c);
        continue;
      }
      this.take(0x7d);
      return ownObject(entries);
    }
  }

  parse() {
    const value = this.value();
    if (this.offset !== this.bytes.length) this.fail();
    this.token();
    return value;
  }
}

function exactKeys(value, expected) {
  return value !== null && typeof value === "object" && !Array.isArray(value) &&
    Object.keys(value).length === expected.length && expected.every((name, index) => Object.keys(value)[index] === name);
}

function parseManifestEnvelope(bytes) {
  if (!Buffer.isBuffer(bytes) || bytes.length < 2 || bytes[bytes.length - 1] !== 0x0a) {
    throw new StageError("MALFORMED_CONTRACT", "CONTRACT_DATA_INVALID");
  }
  const manifest = new CanonicalManifestParser(bytes.subarray(0, -1)).parse();
  if (!exactKeys(manifest, ["files", "kind", "manifest_version", "schema_version"]) ||
      manifest.kind !== "IntegrityManifest" || manifest.manifest_version !== "countershape-manifest/v1" ||
      manifest.schema_version !== "countershape-contract/v1" || !Array.isArray(manifest.files) ||
      manifest.files.length !== PROTECTED_PATHS.length) {
    throw new StageError("MALFORMED_CONTRACT", "CONTRACT_DATA_INVALID");
  }
  for (let index = 0; index < manifest.files.length; index += 1) {
    const entry = manifest.files[index];
    if (!exactKeys(entry, ["byte_count", "byte_sha256", "mode", "path"]) ||
        entry.path !== PROTECTED_PATHS[index] || entry.mode !== "100644" ||
        !Number.isSafeInteger(entry.byte_count) || entry.byte_count <= 0 || entry.byte_count > 640 << 10 ||
        typeof entry.byte_sha256 !== "string" || !/^sha256:[0-9a-f]{64}$/.test(entry.byte_sha256)) {
      throw new StageError("MALFORMED_CONTRACT", "CONTRACT_DATA_INVALID");
    }
  }
  return manifest;
}

function rawDigest(bytes) {
  return `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
}

function readRegular(path, expectedMode, recognized) {
  let before;
  try {
    before = lstatSync(path);
  } catch (error) {
    if (["ENOENT", "EACCES", "EPERM"].includes(error?.code)) throw recognized;
    throw error;
  }
  if (!before.isFile() || before.isSymbolicLink()) throw recognized;
  if (process.platform !== "win32" && (before.mode & 0o777) !== expectedMode) throw recognized;
  let descriptor;
  try {
    descriptor = openSync(path, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
  } catch (error) {
    if (["ENOENT", "EACCES", "EPERM", "ELOOP"].includes(error?.code)) throw recognized;
    throw error;
  }
  try {
    const opened = fstatSync(descriptor);
    if (!opened.isFile() || opened.dev !== before.dev || opened.ino !== before.ino) {
      throw recognized;
    }
    return readFileSync(descriptor);
  } finally {
    closeSync(descriptor);
  }
}

function emitOnce(state, context, outcome, reason) {
  if (state.emitted) throw new Error("result already emitted");
  state.emitted = true;
  context.diagnostic(`${RESULT_PREFIX}${outcome}|${reason}`);
}

test("Countershape selected-field contract", async (context) => {
  const state = { emitted: false };
  let result;
  try {
    const malformed = new StageError("MALFORMED_CONTRACT", "CONTRACT_DATA_INVALID");
    const tampered = new StageError("TAMPER_DETECTED", "COMPANION_INTEGRITY_MISMATCH");
    const manifestBytes = readRegular(join(bundleRoot, "manifest.json"), 0o644, malformed);
    const manifest = parseManifestEnvelope(manifestBytes);
    const verified = new Map();
    for (const entry of manifest.files) {
      const bytes = readRegular(join(bundleRoot, entry.path), 0o644, tampered);
      if (bytes.length !== entry.byte_count || rawDigest(bytes) !== entry.byte_sha256) {
        throw tampered;
      }
      verified.set(entry.path, bytes);
    }
    const harness = await import("./harness.mjs");
    result = await harness.runContract({
      bundleRoot,
      targetRoot: process.cwd(),
      decisionBytes: verified.get("decision.json"),
      fixtureBytes: verified.get("fixture.json"),
    });
  } catch (error) {
    result = error instanceof StageError
      ? { outcome: error.outcome, reason: error.reason }
      : { outcome: "HARNESS_FAILURE", reason: "INTERNAL_INVARIANT_FAILED" };
  }
  const exactPair = new Set([
    "CONFORMS/NONE", "CONTRADICTS/PREDICATE_MISMATCH", "INELIGIBLE_EXECUTION/CAPTURE_FAILED",
    "INELIGIBLE_EXECUTION/CLEANUP_FAILED", "INELIGIBLE_EXECUTION/ENVIRONMENT_INVALID",
    "INELIGIBLE_EXECUTION/EXECUTION_ROOT_FAILED", "INELIGIBLE_EXECUTION/FIXTURE_OVERLAY_FAILED",
    "INELIGIBLE_EXECUTION/ORPHAN_RISK", "INELIGIBLE_EXECUTION/OUTPUT_LIMIT",
    "INELIGIBLE_EXECUTION/PROJECTION_FAILED", "INELIGIBLE_EXECUTION/READINESS_FAILED",
    "INELIGIBLE_EXECUTION/RESPONSE_PARSE_FAILED", "INELIGIBLE_EXECUTION/SOURCE_COPY_FAILED",
    "INELIGIBLE_EXECUTION/SOURCE_INVENTORY_INVALID", "INELIGIBLE_EXECUTION/START_FAILED",
    "INELIGIBLE_EXECUTION/TEARDOWN_FAILED", "INELIGIBLE_EXECUTION/TIMEOUT",
    "INELIGIBLE_EXECUTION/TRANSPORT_FAILED", "MALFORMED_CONTRACT/CONTRACT_DATA_INVALID",
    "TAMPER_DETECTED/COMPANION_INTEGRITY_MISMATCH", "HARNESS_FAILURE/INTERNAL_INVARIANT_FAILED",
  ]);
  if (result === null || typeof result !== "object" || !exactPair.has(`${result.outcome}/${result.reason}`)) {
    result = { outcome: "HARNESS_FAILURE", reason: "INTERNAL_INVARIANT_FAILED" };
  }
  emitOnce(state, context, result.outcome, result.reason);
  if (result.outcome !== "CONFORMS") throw new Error("Countershape contract failed");
});
