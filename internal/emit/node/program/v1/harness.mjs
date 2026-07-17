import { spawn } from "node:child_process";
import { createHash, randomBytes } from "node:crypto";
import {
  closeSync, constants, fchmodSync, fstatSync, fsyncSync, lstatSync, mkdirSync, openSync,
  readFileSync, readdirSync, realpathSync, rmSync, watch, writeFileSync,
} from "node:fs";
import { createConnection } from "node:net";
import { tmpdir } from "node:os";
import { dirname, isAbsolute, join, posix, relative, resolve, sep } from "node:path";

const MAX_JSON_BYTES = 1 << 20;
const MAX_JSON_TOKENS = 1 << 17;
const MAX_CONTAINER_MEMBERS = 1 << 14;
const MAX_JSON_DEPTH = 256;
const MAX_SAFE_INTEGER = 9007199254740991;
const MAX_PORTABLE_VALUE_BYTES = 64 << 10;
const MAX_PORTABLE_LIST_MEMBERS = 256;
const MAX_PORTABLE_TUPLE_FIELDS = 64;
const MAX_PORTABLE_TUPLE_RETAINED_BYTES = 256 << 10;
const MAX_PORTABLE_TUPLE_ENCODED_BYTES = 384 << 10;
const PORTABLE_COMPATIBILITY_FIELD_OVERHEAD_BYTES = 512;

class Refusal extends Error {
  constructor(code) {
    super(code);
    this.code = code;
  }
}

function refuse(code) {
  throw new Refusal(code);
}

function validUTF8(bytes) {
  for (let index = 0; index < bytes.length;) {
    const first = bytes[index];
    if (first <= 0x7f) {
      index += 1;
      continue;
    }
    let count;
    if (first >= 0xc2 && first <= 0xdf) {
      count = 2;
    } else if (first === 0xe0 && index + 1 < bytes.length && bytes[index + 1] >= 0xa0 && bytes[index + 1] <= 0xbf) {
      count = 3;
    } else if (first >= 0xe1 && first <= 0xec) {
      count = 3;
    } else if (first === 0xed && index + 1 < bytes.length && bytes[index + 1] >= 0x80 && bytes[index + 1] <= 0x9f) {
      count = 3;
    } else if (first >= 0xee && first <= 0xef) {
      count = 3;
    } else if (first === 0xf0 && index + 1 < bytes.length && bytes[index + 1] >= 0x90 && bytes[index + 1] <= 0xbf) {
      count = 4;
    } else if (first >= 0xf1 && first <= 0xf3) {
      count = 4;
    } else if (first === 0xf4 && index + 1 < bytes.length && bytes[index + 1] >= 0x80 && bytes[index + 1] <= 0x8f) {
      count = 4;
    } else {
      return false;
    }
    if (index + count > bytes.length) return false;
    for (let offset = 1; offset < count; offset += 1) {
      if (bytes[index + offset] < 0x80 || bytes[index + offset] > 0xbf) return false;
    }
    index += count;
  }
  return true;
}

function ownObject(entries) {
  const value = Object.create(null);
  for (const [name, member] of entries) {
    Object.defineProperty(value, name, {
      configurable: false,
      enumerable: true,
      value: member,
      writable: false,
    });
  }
  return value;
}

class StrictJSONParser {
  constructor(bytes) {
    if (!Buffer.isBuffer(bytes) || bytes.length > MAX_JSON_BYTES || !validUTF8(bytes)) refuse("INVALID_JSON");
    this.bytes = bytes;
    this.offset = 0;
    this.tokens = 0;
    this.nodes = 0;
    this.stringBytes = 0;
  }

  token() {
    this.tokens += 1;
    if (this.tokens > MAX_JSON_TOKENS) refuse("INVALID_JSON");
  }

  node() {
    this.nodes += 1;
    if (this.nodes > MAX_JSON_TOKENS) refuse("INVALID_JSON");
  }

  addString(value) {
    this.stringBytes += Buffer.byteLength(value, "utf8");
    if (this.stringBytes > MAX_JSON_BYTES) refuse("INVALID_JSON");
  }

  skipWhitespace() {
    const start = this.offset;
    while (this.offset < this.bytes.length) {
      const byte = this.bytes[this.offset];
      if (byte !== 0x20 && byte !== 0x09 && byte !== 0x0a && byte !== 0x0d) break;
      this.offset += 1;
    }
    if (this.offset !== start) this.token();
  }

  punctuation(byte) {
    this.skipWhitespace();
    if (this.bytes[this.offset] !== byte) refuse("INVALID_JSON");
    this.offset += 1;
    this.token();
  }

  delimiter(byte) {
    return byte === undefined || byte === 0x20 || byte === 0x09 || byte === 0x0a || byte === 0x0d ||
      byte === 0x7b || byte === 0x7d || byte === 0x5b || byte === 0x5d || byte === 0x3a || byte === 0x2c;
  }

  scalar() {
    const start = this.offset;
    while (this.offset < this.bytes.length && !this.delimiter(this.bytes[this.offset])) this.offset += 1;
    this.token();
    return this.bytes.subarray(start, this.offset).toString("ascii");
  }

  string() {
    this.skipWhitespace();
    if (this.bytes[this.offset] !== 0x22) refuse("INVALID_JSON");
    this.offset += 1;
    this.token();
    const chunks = [];
    let rawStart = this.offset;
    while (this.offset < this.bytes.length) {
      const byte = this.bytes[this.offset];
      if (byte === 0x22) {
        if (this.offset > rawStart) chunks.push(this.bytes.subarray(rawStart, this.offset).toString("utf8"));
        this.offset += 1;
        const value = chunks.join("");
        this.addString(value);
        return value;
      }
      if (byte < 0x20) refuse("INVALID_JSON");
      if (byte !== 0x5c) {
        this.offset += byte < 0x80 ? 1 : byte < 0xe0 ? 2 : byte < 0xf0 ? 3 : 4;
        continue;
      }
      if (this.offset > rawStart) chunks.push(this.bytes.subarray(rawStart, this.offset).toString("utf8"));
      this.offset += 1;
      if (this.offset >= this.bytes.length) refuse("INVALID_JSON");
      const escape = this.bytes[this.offset];
      this.offset += 1;
      const simple = new Map([
        [0x22, '"'], [0x5c, "\\"], [0x2f, "/"], [0x62, "\b"], [0x66, "\f"],
        [0x6e, "\n"], [0x72, "\r"], [0x74, "\t"],
      ]);
      if (simple.has(escape)) {
        chunks.push(simple.get(escape));
        rawStart = this.offset;
        continue;
      }
      if (escape !== 0x75 || this.offset + 4 > this.bytes.length) refuse("INVALID_JSON");
      const first = this.hexQuad(this.offset);
      this.offset += 4;
      let codepoint = first;
      if (first >= 0xd800 && first <= 0xdbff) {
        if (this.offset + 6 > this.bytes.length || this.bytes[this.offset] !== 0x5c || this.bytes[this.offset + 1] !== 0x75) {
          refuse("INVALID_JSON");
        }
        const second = this.hexQuad(this.offset + 2);
        if (second < 0xdc00 || second > 0xdfff) refuse("INVALID_JSON");
        codepoint = 0x10000 + ((first - 0xd800) << 10) + second - 0xdc00;
        this.offset += 6;
      } else if (first >= 0xdc00 && first <= 0xdfff) {
        refuse("INVALID_JSON");
      }
      chunks.push(String.fromCodePoint(codepoint));
      rawStart = this.offset;
    }
    refuse("INVALID_JSON");
  }

  hexQuad(offset) {
    let value = 0;
    for (let index = 0; index < 4; index += 1) {
      const byte = this.bytes[offset + index];
      let digit;
      if (byte >= 0x30 && byte <= 0x39) digit = byte - 0x30;
      else if (byte >= 0x61 && byte <= 0x66) digit = byte - 0x61 + 10;
      else if (byte >= 0x41 && byte <= 0x46) digit = byte - 0x41 + 10;
      else refuse("INVALID_JSON");
      value = (value << 4) | digit;
    }
    return value;
  }

  value(depth = 0) {
    if (depth > MAX_JSON_DEPTH) refuse("INVALID_JSON");
    this.skipWhitespace();
    this.node();
    const byte = this.bytes[this.offset];
    if (byte === 0x7b) return this.object(depth + 1);
    if (byte === 0x5b) return this.array(depth + 1);
    if (byte === 0x22) return this.string();
    if (byte === 0x6e || byte === 0x74 || byte === 0x66) {
      const scalar = this.scalar();
      if (scalar === "null") return null;
      if (scalar === "true") return true;
      if (scalar === "false") return false;
      refuse("INVALID_JSON");
    }
    if (byte === 0x2d || byte === 0x2b || (byte >= 0x30 && byte <= 0x39)) {
      const scalar = this.scalar();
      if (!/^-?(0|[1-9][0-9]*)$/.test(scalar) || scalar === "-0") refuse("INVALID_JSON");
      const value = Number(scalar);
      if (!Number.isSafeInteger(value) || Math.abs(value) > MAX_SAFE_INTEGER) refuse("INVALID_JSON");
      return value;
    }
    refuse("INVALID_JSON");
  }

  array(depth) {
    this.punctuation(0x5b);
    this.skipWhitespace();
    if (this.bytes[this.offset] === 0x5d) {
      this.offset += 1;
      this.token();
      return [];
    }
    const result = [];
    for (;;) {
      if (result.length >= MAX_CONTAINER_MEMBERS) refuse("INVALID_JSON");
      result.push(this.value(depth));
      this.skipWhitespace();
      if (this.bytes[this.offset] === 0x2c) {
        this.offset += 1;
        this.token();
        continue;
      }
      if (this.bytes[this.offset] === 0x5d) {
        this.offset += 1;
        this.token();
        return result;
      }
      refuse("INVALID_JSON");
    }
  }

  object(depth) {
    this.punctuation(0x7b);
    this.skipWhitespace();
    if (this.bytes[this.offset] === 0x7d) {
      this.offset += 1;
      this.token();
      return ownObject([]);
    }
    const entries = [];
    const seen = new Set();
    for (;;) {
      if (entries.length >= MAX_CONTAINER_MEMBERS) refuse("INVALID_JSON");
      const name = this.string();
      if (seen.has(name)) refuse("INVALID_JSON");
      seen.add(name);
      this.punctuation(0x3a);
      entries.push([name, this.value(depth)]);
      this.skipWhitespace();
      if (this.bytes[this.offset] === 0x2c) {
        this.offset += 1;
        this.token();
        continue;
      }
      if (this.bytes[this.offset] === 0x7d) {
        this.offset += 1;
        this.token();
        return ownObject(entries);
      }
      refuse("INVALID_JSON");
    }
  }

  parse() {
    const result = this.value(0);
    this.skipWhitespace();
    if (this.offset !== this.bytes.length) refuse("INVALID_JSON");
    this.token();
    return result;
  }
}

function canonicalString(value) {
  let result = '"';
  for (const character of value) {
    const codepoint = character.codePointAt(0);
    if (character === '"' || character === "\\") result += `\\${character}`;
    else if (character === "\b") result += "\\b";
    else if (character === "\f") result += "\\f";
    else if (character === "\n") result += "\\n";
    else if (character === "\r") result += "\\r";
    else if (character === "\t") result += "\\t";
    else if (codepoint < 0x20) result += `\\u00${codepoint.toString(16).padStart(2, "0")}`;
    else result += character;
  }
  return `${result}"`;
}

function canonicalText(value) {
  if (value === null) return "null";
  if (value === true) return "true";
  if (value === false) return "false";
  if (typeof value === "number") return String(value);
  if (typeof value === "string") return canonicalString(value);
  if (Array.isArray(value)) return `[${value.map(canonicalText).join(",")}]`;
  if (typeof value === "object") {
    const names = Object.keys(value).sort((left, right) => Buffer.compare(Buffer.from(left, "utf8"), Buffer.from(right, "utf8")));
    return `{${names.map((name) => `${canonicalString(name)}:${canonicalText(value[name])}`).join(",")}}`;
  }
  refuse("INVALID_JSON");
}

export function canonicalizeJSON(input) {
  const value = new StrictJSONParser(input).parse();
  return Buffer.from(canonicalText(value), "utf8");
}

export function parseCanonicalJSON(input) {
  const canonical = canonicalizeJSON(input);
  if (!canonical.equals(input)) refuse("NONCANONICAL_JSON");
  return new StrictJSONParser(canonical).parse();
}

const PARITY_MANIFEST_PATHS = ["README.md", "contract.test.mjs", "decision.json", "fixture.json", "harness.mjs"];
const PARITY_CONTROL_REASONS = new Set([
  "MATERIALIZATION_ERROR", "SETUP_ERROR", "START_ERROR", "READINESS_ERROR", "PROBE_TRANSPORT_ERROR",
  "TIMEOUT", "CANCELLED", "OUTPUT_LIMIT", "PROJECTION_REJECTED", "ORPHAN_RISK", "TEARDOWN_ERROR",
  "UNSUPPORTED_GIT_MODE", "MISSING_OBJECT", "BUDGET_EXHAUSTED",
]);
const PARITY_TEARDOWN_REASONS = new Set(["ORPHAN_RISK", "TEARDOWN_ERROR"]);

function parityOK(value) {
  return ownObject([["status", "OK"], ["value", value]]);
}

function parityRefused(code) {
  return ownObject([["code", code], ["status", "REFUSED"]]);
}

function requireParityInput(input, keys) {
  if (!exactKeys(input, keys)) refuse("INVALID_OPERATION_INPUT");
}

function parityManifestEnvelope(exact) {
  try {
    if (!Buffer.isBuffer(exact) || exact.length < 2 || exact.length > 640 << 10 || exact[exact.length - 1] !== 0x0a) {
      refuse("INVALID_MANIFEST");
    }
    const manifest = parseCanonicalJSON(exact.subarray(0, -1));
    if (!exactKeys(manifest, ["files", "kind", "manifest_version", "schema_version"]) ||
        manifest.kind !== "IntegrityManifest" || manifest.manifest_version !== "countershape-manifest/v1" ||
        manifest.schema_version !== "countershape-contract/v1" || !Array.isArray(manifest.files) ||
        manifest.files.length !== PARITY_MANIFEST_PATHS.length) refuse("INVALID_MANIFEST");
    for (let index = 0; index < manifest.files.length; index += 1) {
      const file = manifest.files[index];
      if (!exactKeys(file, ["byte_count", "byte_sha256", "mode", "path"]) ||
          file.path !== PARITY_MANIFEST_PATHS[index] || file.mode !== "100644" ||
          !Number.isSafeInteger(file.byte_count) || file.byte_count <= 0 || file.byte_count > 640 << 10 ||
          typeof file.byte_sha256 !== "string" || !/^sha256:[0-9a-f]{64}$/.test(file.byte_sha256)) {
        refuse("INVALID_MANIFEST");
      }
    }
    return manifest;
  } catch (error) {
    if (error instanceof Refusal && error.code === "INVALID_MANIFEST") throw error;
    refuse("INVALID_MANIFEST");
  }
}

function parityHTTPPolicy(input) {
  const integers = [input.body_bytes, input.header_bytes, input.header_count, input.status_line_bytes];
  if (integers.some((value) => !Number.isSafeInteger(value)) || input.body_bytes < 1 || input.body_bytes > 16 << 20 ||
      input.header_bytes < 16 || input.header_bytes > 1 << 20 || input.header_count < 1 || input.header_count > 1024 ||
      input.status_line_bytes < 16 || input.status_line_bytes > 8 << 10) refuse("INVALID_OPERATION_INPUT");
  return ownObject([
    ["body_bytes", input.body_bytes], ["header_bytes", input.header_bytes], ["header_count", input.header_count],
    ["status_line_bytes", input.status_line_bytes],
  ]);
}

function parityFieldSelection(selected, profile, requireFullProfile = false) {
  if (!Array.isArray(selected) || selected.length === 0 || selected.length > profile.length) {
    refuse("INVALID_OPERATION_INPUT");
  }
  let previous = -1;
  for (const field of selected) {
    const index = profile.indexOf(field);
    if (typeof field !== "string" || index < 0 || index <= previous) refuse("INVALID_OPERATION_INPUT");
    previous = index;
  }
  if (requireFullProfile && !sameCanonical(selected, profile)) refuse("INVALID_OPERATION_INPUT");
  return [...selected];
}

function parityCLICompletion(value) {
  if (value?.kind === "EXITED") {
    if (!exactKeys(value, ["code", "kind"]) || !Number.isSafeInteger(value.code) || value.code < 0 || value.code > 255) {
      refuse("INVALID_OPERATION_INPUT");
    }
    return { kind: "EXITED", code: value.code, signal: "" };
  }
  if (value?.kind === "SIGNALED") {
    if (!exactKeys(value, ["kind", "signal"]) || typeof value.signal !== "string" ||
        Buffer.byteLength(value.signal, "utf8") === 0 || Buffer.byteLength(value.signal, "utf8") > 1024 ||
        Buffer.from(value.signal, "utf8").toString("utf8") !== value.signal) refuse("INVALID_OPERATION_INPUT");
    return { kind: "SIGNALED", code: 0, signal: value.signal };
  }
  refuse("INVALID_OPERATION_INPUT");
}

function parityTupleValue(exact) {
  return parseCanonicalJSON(exact);
}

function parityProfileFields(adapter, profileFields) {
  const registry = adapter === "CLI" ? CLI_FIELD_PROFILE : adapter === "HTTP" ? HTTP_FIELD_PROFILE : null;
  if (registry === null || !Array.isArray(profileFields) || profileFields.length === 0) refuse("INVALID_PREDICATE");
  const fieldIDs = registry.map((entry) => entry[0]);
  try {
    return parityFieldSelection(profileFields, fieldIDs, adapter === "HTTP");
  } catch {
    refuse("INVALID_PREDICATE");
  }
}

function parityExactTuple(tuple, selected, descriptors) {
  try {
    if (!exactKeys(tuple, ["fields"]) || !Array.isArray(tuple.fields) || tuple.fields.length !== selected.length) {
      refuse("INVALID_PREDICATE");
    }
    for (let index = 0; index < tuple.fields.length; index += 1) {
      const field = tuple.fields[index];
      if (!exactKeys(field, ["field_id", "value"]) || field.field_id !== selected[index]) refuse("INVALID_PREDICATE");
      validateExactValue(field.value, descriptors[index]);
    }
    validatePortableTuple(tuple.fields.map((field) => field.value), "CONTRACT_DATA_INVALID");
    return Buffer.from(canonicalText(tuple), "utf8");
  } catch {
    refuse("INVALID_PREDICATE");
  }
}

function evaluateParityPredicate(input) {
  requireParityInput(input, [
    "adapter", "allowed_tuples", "decision_action", "observed_tuple", "profile_fields", "selected_fields",
    "stimulus_digest",
  ]);
  try {
    if (!validDigest(input.stimulus_digest) || !["ALLOW_OBSERVED", "CUSTOM_EXPECTATION"].includes(input.decision_action)) {
      refuse("INVALID_PREDICATE");
    }
    const profileFields = parityProfileFields(input.adapter, input.profile_fields);
    let selected;
    try {
      selected = parityFieldSelection(input.selected_fields, profileFields);
    } catch {
      refuse("INVALID_PREDICATE");
    }
    const registry = input.adapter === "CLI" ? CLI_FIELD_PROFILE : HTTP_FIELD_PROFILE;
    const descriptors = selected.map((field) => {
      const entry = registry.find((candidate) => candidate[0] === field);
      if (!entry) refuse("INVALID_PREDICATE");
      return profileDescriptor(entry);
    });
    if (!Array.isArray(input.allowed_tuples) || input.allowed_tuples.length === 0 || input.allowed_tuples.length > 4 ||
        input.decision_action === "CUSTOM_EXPECTATION" && input.allowed_tuples.length !== 1) {
      refuse("INVALID_PREDICATE");
    }
    let previous = null;
    const allowed = input.allowed_tuples.map((tuple) => {
      const exact = parityExactTuple(tuple, selected, descriptors);
      if (previous !== null && Buffer.compare(previous, exact) >= 0) refuse("INVALID_PREDICATE");
      previous = exact;
      return exact;
    });
    const observed = parityExactTuple(input.observed_tuple, selected, descriptors);
    return ownObject([["match", allowed.some((tuple) => tuple.equals(observed))]]);
  } catch (error) {
    if (error instanceof Refusal && error.code === "INVALID_PREDICATE") throw error;
    refuse("INVALID_PREDICATE");
  }
}

function selectOwnerEligibility(facts) {
  if (!exactKeys(facts, ["kind", "reasons"]) || !Array.isArray(facts.reasons)) refuse("INVALID_OWNER_FACTS");
  if (facts.kind === "BEHAVIOR_CAPTURED") {
    if (facts.reasons.length !== 0) refuse("INVALID_OWNER_FACTS");
    return ownObject([["eligibility", "ELIGIBLE"], ["reasons", []]]);
  }
  if (facts.kind !== "CONTROL_INELIGIBLE" || facts.reasons.length < 1 || facts.reasons.length > 3 ||
      new Set(facts.reasons).size !== facts.reasons.length ||
      facts.reasons.some((reason) => typeof reason !== "string" || !PARITY_CONTROL_REASONS.has(reason))) {
    refuse("INVALID_OWNER_FACTS");
  }
  const firstIsTeardown = PARITY_TEARDOWN_REASONS.has(facts.reasons[0]);
  if (facts.reasons.slice(1).some((reason) => !PARITY_TEARDOWN_REASONS.has(reason)) ||
      firstIsTeardown && facts.reasons.some((reason) => !PARITY_TEARDOWN_REASONS.has(reason))) {
    refuse("INVALID_OWNER_FACTS");
  }
  return ownObject([["eligibility", "INELIGIBLE"], ["reasons", [...facts.reasons]]]);
}

export function evaluateParityOperation(request) {
  try {
    if (!exactKeys(request, ["input", "operation"]) || request.input === null || typeof request.input !== "object" ||
        Array.isArray(request.input) || typeof request.operation !== "string") refuse("INVALID_OPERATION_INPUT");
    switch (request.operation) {
      case "CANONICALIZE_JSON": {
        requireParityInput(request.input, ["bytes_base64"]);
        const canonical = canonicalizeJSON(strictBase64(request.input.bytes_base64));
        return parityOK(ownObject([["canonical_base64", canonical.toString("base64")]]));
      }
      case "PARSE_CANONICAL_JSON": {
        requireParityInput(request.input, ["bytes_base64"]);
        const exact = strictBase64(request.input.bytes_base64);
        parseCanonicalJSON(exact);
        return parityOK(ownObject([["canonical_base64", exact.toString("base64")]]));
      }
      case "PARSE_MANIFEST_ENVELOPE": {
        requireParityInput(request.input, ["bytes_base64"]);
        const exact = strictBase64(request.input.bytes_base64);
        return parityOK(ownObject([["manifest", parityManifestEnvelope(exact)]]));
      }
      case "PARSE_READY_FRAME": {
        requireParityInput(request.input, ["eof_observed", "frame_base64"]);
        if (typeof request.input.eof_observed !== "boolean") refuse("INVALID_OPERATION_INPUT");
        const frame = strictBase64(request.input.frame_base64);
        try {
          return parityOK(ownObject([["port", parseReadyFrame(frame, request.input.eof_observed)]]));
        } catch {
          refuse("READINESS_FAILED");
        }
      }
      case "PARSE_HTTP_RESPONSE": {
        requireParityInput(request.input, ["body_bytes", "header_bytes", "header_count", "response_base64", "status_line_bytes"]);
        const policy = parityHTTPPolicy(request.input);
        const raw = strictBase64(request.input.response_base64);
        const response = parseHTTPResponse(raw, policy);
        return parityOK(ownObject([
          ["body_base64", response.body.toString("base64")], ["content_length", response.contentLength],
          ["headers", response.headers.map((header) => ownObject([["name", header.name], ["value", header.value]]))],
          ["reason", response.reason], ["status", response.status],
        ]));
      }
      case "PROJECT_CLI_OBSERVATION": {
        requireParityInput(request.input, ["completion", "selected_fields", "stderr_base64", "stdout_base64"]);
        const selected = parityFieldSelection(request.input.selected_fields, CLI_FIELD_PROFILE.map((entry) => entry[0]));
        const profileFields = selected.map((field) => profileDescriptor(CLI_FIELD_PROFILE.find((entry) => entry[0] === field)));
        const completion = parityCLICompletion(request.input.completion);
        const stderr = strictBase64(request.input.stderr_base64);
        const stdout = strictBase64(request.input.stdout_base64);
        try {
          const tuple = projectCLI({ predicate: { selected }, profileFields }, completion, stdout, stderr);
          return parityOK(ownObject([["tuple", parityTupleValue(tuple)]]));
        } catch {
          refuse("PROJECTION_FAILED");
        }
      }
      case "PROJECT_HTTP_OBSERVATION": {
        requireParityInput(request.input, [
          "body_bytes", "header_bytes", "header_count", "response_base64", "scratch_root", "selected_fields",
          "status_line_bytes",
        ]);
        const policy = parityHTTPPolicy(request.input);
        const selected = parityFieldSelection(request.input.selected_fields, HTTP_FIELD_PROFILE.map((entry) => entry[0]));
        if (typeof request.input.scratch_root !== "string" || !request.input.scratch_root.startsWith("/") ||
            posix.resolve(request.input.scratch_root) !== request.input.scratch_root) refuse("INVALID_OPERATION_INPUT");
        const raw = strictBase64(request.input.response_base64);
        try {
          const response = parseHTTPResponse(raw, policy);
          const tuple = projectHTTP({ predicate: { selected } }, response, request.input.scratch_root);
          return parityOK(ownObject([["tuple", parityTupleValue(tuple)]]));
        } catch {
          refuse("PROJECTION_FAILED");
        }
      }
      case "EVALUATE_EXACT_PREDICATE":
        return parityOK(evaluateParityPredicate(request.input));
      case "SELECT_DIRECT_RESULT":
        return parityOK(selectDirectResult(request.input));
      case "SELECT_OWNER_ELIGIBILITY":
        return parityOK(selectOwnerEligibility(request.input));
      default:
        return parityRefused("UNSUPPORTED_OPERATION");
    }
  } catch (error) {
    const code = error instanceof Refusal ? error.code :
      error instanceof RuntimeRefusal ? error.reason : "INVALID_OPERATION_INPUT";
    return parityRefused(code);
  }
}

function strictBase64(value) {
  if (typeof value !== "string" || !/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(value)) {
    refuse("INVALID_BASE64");
  }
  const decoded = Buffer.from(value, "base64");
  if (decoded.toString("base64") !== value) refuse("INVALID_BASE64");
  return decoded;
}

const REMAINING_REASON_PRECEDENCE = [
  "OUTPUT_LIMIT", "TIMEOUT", "TRANSPORT_FAILED", "CAPTURE_FAILED", "RESPONSE_PARSE_FAILED",
  "PROJECTION_FAILED", "READINESS_FAILED", "START_FAILED", "ENVIRONMENT_INVALID",
  "FIXTURE_OVERLAY_FAILED", "SOURCE_COPY_FAILED", "EXECUTION_ROOT_FAILED", "SOURCE_INVENTORY_INVALID",
];
const DIRECT_REASON_ROSTER = new Set([
  "ORPHAN_RISK", "CLEANUP_FAILED", "TEARDOWN_FAILED", ...REMAINING_REASON_PRECEDENCE,
]);

function selectDirectResult(facts) {
  if (!exactKeys(facts, ["ineligible_reasons", "internal_failure", "predicate_match", "tamper"]) ||
      !Array.isArray(facts.ineligible_reasons) || typeof facts.internal_failure !== "boolean" ||
      typeof facts.predicate_match !== "boolean" || typeof facts.tamper !== "boolean") refuse("INVALID_RESULT_FACTS");
  const reasons = facts.ineligible_reasons;
  if (reasons.some((reason) => typeof reason !== "string" || !DIRECT_REASON_ROSTER.has(reason)) ||
      new Set(reasons).size !== reasons.length) refuse("INVALID_RESULT_FACTS");
  for (const reason of ["ORPHAN_RISK", "CLEANUP_FAILED", "TEARDOWN_FAILED"]) {
    if (reasons.includes(reason)) return ownObject([["outcome", "INELIGIBLE_EXECUTION"], ["reason", reason]]);
  }
  if (facts.tamper === true) return ownObject([["outcome", "TAMPER_DETECTED"], ["reason", "COMPANION_INTEGRITY_MISMATCH"]]);
  if (facts.internal_failure === true) return ownObject([["outcome", "HARNESS_FAILURE"], ["reason", "INTERNAL_INVARIANT_FAILED"]]);
  for (const reason of REMAINING_REASON_PRECEDENCE) {
    if (reasons.includes(reason)) return ownObject([["outcome", "INELIGIBLE_EXECUTION"], ["reason", reason]]);
  }
  if (reasons.length !== 0) refuse("INVALID_RESULT_FACTS");
  return facts.predicate_match
    ? ownObject([["outcome", "CONFORMS"], ["reason", "NONE"]])
    : ownObject([["outcome", "CONTRADICTS"], ["reason", "PREDICATE_MISMATCH"]]);
}

function exactKeys(value, expected) {
  if (value === null || typeof value !== "object" || Array.isArray(value)) return false;
  const actual = Object.keys(value);
  return actual.length === expected.length && actual.every((name, index) => name === expected[index]);
}

function sameCanonical(left, right) {
  return canonicalText(left) === canonicalText(right);
}

function typedDigest(kind, bytes) {
  return `sha256:${createHash("sha256").update("countershape/v1/").update(kind).update(Buffer.from([0])).update(bytes).digest("hex")}`;
}

function rawDigest(bytes) {
  return `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
}

function validDigest(value) {
  return typeof value === "string" && /^sha256:[0-9a-f]{64}$/.test(value);
}

function parseJSONEnvelope(exact) {
  if (!Buffer.isBuffer(exact) || exact.length < 2 || exact[exact.length - 1] !== 0x0a) refuse("CONTRACT_DATA_INVALID");
  return parseCanonicalJSON(exact.subarray(0, -1));
}

function decodeAuthority(encoded, digest, kind, limit = MAX_JSON_BYTES) {
  if (!validDigest(digest)) refuse("CONTRACT_DATA_INVALID");
  const exact = strictBase64(encoded);
  if (exact.length === 0 || exact.length > limit) refuse("CONTRACT_DATA_INVALID");
  const value = parseCanonicalJSON(exact);
  if (typedDigest(kind, exact) !== digest) refuse("CONTRACT_DATA_INVALID");
  return { exact, value };
}

const SOURCE_KEYS = [
  "adapter", "adapter_projection_definition_base64", "adapter_projection_definition_digest", "cli",
  "closed_facts", "entrypoint", "execution_binding_base64", "execution_binding_digest", "http", "kind",
  "launch_profile", "limits", "plan_environment", "plan_secret_slots", "portable_profile_base64",
  "portable_profile_digest", "projection_binding_base64", "projection_binding_digest", "schema_version",
  "source_profile", "start_profile", "stimulus_base64", "stimulus_digest", "stimulus_kind", "version",
  "world_plan_base64", "world_plan_digest",
];

const LIMIT_KEYS = [
  "http_body_bytes", "materialized_bytes_per_world", "materialized_entry_count", "probe_ms", "readiness_ms",
  "single_blob_bytes", "stderr_bytes", "stdout_bytes", "teardown_ms",
];

const CLI_FIELD_PROFILE = [
  ["cli.completion.kind", "exit", ["completion", "kind"], "UTF8_STRING", "REJECT_CAPTURE", "STRING", false],
  ["cli.exit.code", "exit", ["completion", "code"], "SAFE_INTEGER", "TAGGED_MISSING_FOR_SIGNAL", "INTEGER", true],
  ["cli.exit.signal", "exit", ["completion", "signal"], "UTF8_STRING", "TAGGED_MISSING_FOR_EXIT", "STRING", true],
  ["cli.stdout.bytes", "stdout", ["bytes"], "BYTES", "REJECT_CHANNEL", "BYTES", false],
  ["cli.stderr.text", "stderr", ["utf8_text"], "UTF8_STRING", "REJECT_CHANNEL", "STRING", false],
  ["cli.stdout.json.mode", "stdout", ["strict_json", "mode"], "UTF8_STRING", "TAGGED_MISSING", "STRING", true],
  ["cli.stdout.json.source", "stdout", ["strict_json", "source"], "UTF8_STRING", "TAGGED_MISSING", "STRING", true],
];

const HTTP_FIELD_PROFILE = [
  ["http.status", "http.status", ["status"], "SAFE_INTEGER", "REJECT_CAPTURE", "INTEGER", false],
  ["http.header.content-type", "http.headers", ["content-type"], "ORDERED_STRING_LIST", "TAGGED_MISSING", "ORDERED_STRING_LIST", true],
  ["http.body.kind", "http.body", ["strict_json", "kind"], "UTF8_STRING", "REJECT_BODY", "STRING", false],
  ["http.body.metadata", "http.body", ["strict_json", "metadata"], "CANONICAL_JSON_OBJECT", "REJECT_BODY", "CANONICAL_JSON", false],
];

function profileDescriptor(entry) {
  return ownObject([
    ["allow_missing", entry[6]], ["allow_null", false], ["channel", entry[1]], ["field_id", entry[0]],
    ["missing_policy", entry[4]], ["present_tag", entry[5]], ["source_path", entry[2]],
    ["source_value_kind", entry[3]],
  ]);
}

function validateProjectionBinding(value, source) {
  const keys = [
    "accepted_channels", "adapter_domain", "comparator", "configuration_digest", "field_registry_digest",
    "implementation_digest", "kind", "operations", "schema_version",
  ];
  const expectedChannels = source.adapter === "CLI"
    ? ["exit", "stderr", "stdout"]
    : ["http.body", "http.headers", "http.status"];
  if (!exactKeys(value, keys) || value.schema_version !== "countershape/v1" || value.kind !== "ProjectionDefinition" ||
      value.adapter_domain !== source.adapter || value.comparator !== "EXACT_CANONICAL_V1" ||
      !sameCanonical(value.accepted_channels, expectedChannels) ||
      !validDigest(value.configuration_digest) || !validDigest(value.field_registry_digest) ||
      !validDigest(value.implementation_digest) || !Array.isArray(value.operations) ||
      value.operations.length === 0 || value.operations.length > 64) {
    refuse("CONTRACT_DATA_INVALID");
  }
  const names = new Set();
  const operations = value.operations.map((operation) => {
    if (!exactKeys(operation, ["name", "rule_digest"]) || typeof operation.name !== "string" ||
        Buffer.byteLength(operation.name, "utf8") === 0 || Buffer.byteLength(operation.name, "utf8") > 128 ||
        names.has(operation.name) || !validDigest(operation.rule_digest)) refuse("CONTRACT_DATA_INVALID");
    names.add(operation.name);
    return { name: operation.name, rule_digest: operation.rule_digest };
  });
  return {
    acceptedChannels: expectedChannels,
    configurationDigest: value.configuration_digest,
    fieldRegistryDigest: value.field_registry_digest,
    implementationDigest: value.implementation_digest,
    operations,
  };
}

function projectionOperation(kind, name, semantics) {
  return ownObject([
    ["name", name], ["rule_digest", typedDigest(kind, Buffer.from(`${name}\0${semantics}`, "utf8"))],
    ["semantics", semantics],
  ]);
}

function expectedProjectionAuthority(source, profileFields) {
  const adapter = source.adapter;
  const version = adapter === "CLI" ? "cli-projection/v1" : "http-projection/v1";
  const registryKind = adapter === "CLI" ? "CLIFieldRegistry" : "HTTPFieldRegistry";
  const registrySource = adapter === "CLI" ? CLI_FIELD_PROFILE : HTTP_FIELD_PROFILE;
  const registry = registrySource.map((entry) => ownObject([
    ["channel", entry[1]], ["field_id", entry[0]], ["missing_policy", entry[4]], ["path", entry[2]],
    ["value_kind", entry[3]],
  ]));
  const fieldRegistryDigest = digestCanonical(registryKind, ownObject([
    ["fields", registry], ["kind", registryKind], ["schema_version", "countershape/v1"], ["version", version],
  ]));
  const fields = profileFields.map((field) => field.field_id);
  const configurationKind = adapter === "CLI" ? "CLIProjectionConfiguration" : "HTTPProjectionConfiguration";
  const configurationDigest = digestCanonical(configurationKind, ownObject([
    ["fields", fields], ["kind", configurationKind], ["schema_version", "countershape/v1"], ["version", version],
  ]));
  const implementationKind = adapter === "CLI" ? "CLIProjectionImplementation" : "HTTPProjectionImplementation";
  const implementationText = adapter === "CLI"
    ? "cli-projection/v1\0closed-field-registry\0visible-pure-operations\0exact-canonical"
    : "http-projection/v1\0closed-four-field-registry\0visible-validate-then-omit-operations\0exact-operation-source-links\0strict-json-object\0exact-canonical";
  const implementationDigest = typedDigest(implementationKind, Buffer.from(implementationText, "utf8"));
  const operationKind = adapter === "CLI" ? "CLIProjectionOperationRule" : "HTTPProjectionOperationRule";
  const operations = [];
  const add = (name, semantics) => operations.push(projectionOperation(operationKind, name, semantics));
  if (adapter === "CLI") {
    add("cli.require-eligible-capture/v1", "require no controls, present completion, complete stdout and stderr, present fixture overlay receipt, and validated fixture invocation receipt");
    const channels = new Set(profileFields.map((field) => field.channel));
    if (channels.has("exit")) add("cli.require-completion/v1", "require the disjoint EXITED-or-SIGNALED completion sum");
    if (channels.has("stdout")) add("cli.require-stdout/v1", "require complete stdout while preserving present-empty");
    if (channels.has("stderr")) add("cli.require-stderr/v1", "require complete stderr while preserving present-empty");
    if (fields.includes("cli.stderr.text")) add("cli.validate-stderr-utf8/v1", "validate stderr bytes as strict UTF-8 without trimming or replacement");
    if (fields.includes("cli.stdout.json.mode") || fields.includes("cli.stdout.json.source")) {
      add("cli.validate-stdout-utf8/v1", "validate stdout bytes as strict UTF-8 without trimming or replacement");
      add("cli.parse-stdout-strict-json/v1", "parse one strict canonical-profile JSON object; reject duplicates, coercion, and trailing data");
    }
    const suffixes = new Map([
      ["cli.completion.kind", "completion-kind"], ["cli.exit.code", "exit-code"],
      ["cli.exit.signal", "exit-signal"], ["cli.stdout.bytes", "stdout-bytes"],
      ["cli.stderr.text", "stderr-text"], ["cli.stdout.json.mode", "stdout-json-mode"],
      ["cli.stdout.json.source", "stdout-json-source"],
    ]);
    for (const field of fields) {
      add(`cli.select-${suffixes.get(field)}/v1`, "select the one closed typed field and retain tagged missing/present-empty semantics");
    }
    add("cli.encode-tagged-fields-canonical/v1", "encode registry-ordered field IDs and exact tagged values with the Countershape canonical profile");
  } else {
    for (const [name, semantics] of [
      ["http.require-complete-response/v1", "require no controls, accepted fixture readiness, one complete request, one complete parsed response, and clean teardown"],
      ["http.select-status/v1", "select the complete application status including 500"],
      ["http.select-content-type-multimap/v1", "preserve ordered duplicate content-type values and tagged missing versus present-empty"],
      ["http.parse-body-strict-json-object/v1", "parse one strict JSON object with duplicate rejection and no trailing data"],
      ["http.validate-then-omit-request-id/v1", "validate the top-level request_id value as a string, then omit that member from projection output"],
      ["http.validate-then-omit-scratch-root/v1", "validate the top-level scratch_root value as a string exactly equal to the captured runtime scratch-root authority, then omit that member from projection output"],
      ["http.select-body-kind/v1", "select the required top-level kind string"],
      ["http.select-body-metadata/v1", "select the required disclosure-bearing metadata object without deleting its members"],
      ["http.encode-four-fields-canonical/v1", "encode the fixed registry-ordered tagged field tuple exactly"],
    ]) add(name, semantics);
  }
  return { configurationDigest, fieldRegistryDigest, implementationDigest, operations };
}

function validateProfile(profile, binding, source) {
  const keys = [
    "adapter_domain", "fields", "kind", "projection_definition_binding_base64",
    "projection_definition_binding_digest", "schema_version", "translator_name", "translator_version",
  ];
  if (!exactKeys(profile, keys) || profile.schema_version !== "countershape/v1" ||
      profile.kind !== "PortableProjectionProfile" || profile.adapter_domain !== source.adapter ||
      profile.projection_definition_binding_digest !== source.projection_binding_digest ||
      profile.projection_definition_binding_base64 !== source.projection_binding_base64 ||
      strictBase64(profile.projection_definition_binding_base64).compare(binding.exact) !== 0 ||
      profile.translator_version !== "v1" || !Array.isArray(profile.fields)) refuse("CONTRACT_DATA_INVALID");
  const expected = source.adapter === "CLI" ? CLI_FIELD_PROFILE : HTTP_FIELD_PROFILE;
  const expectedTranslator = source.adapter === "CLI" ? "CLI_PROJECTION_TO_PORTABLE" : "HTTP_PROJECTION_TO_PORTABLE";
  if (profile.translator_name !== expectedTranslator || profile.fields.length === 0 || profile.fields.length > expected.length) {
    refuse("CONTRACT_DATA_INVALID");
  }
  let previous = -1;
  for (const field of profile.fields) {
    const index = expected.findIndex((entry) => entry[0] === field?.field_id);
    if (index < 0 || index <= previous || !sameCanonical(field, profileDescriptor(expected[index]))) {
      refuse("CONTRACT_DATA_INVALID");
    }
    previous = index;
  }
  if (source.adapter === "HTTP" && profile.fields.length !== expected.length) refuse("CONTRACT_DATA_INVALID");
  return profile.fields;
}

function validateWorldPlan(plan, source, binding, bindingInfo) {
  const keys = [
    "adapter", "budgets", "candidate_set_digest", "capture_policy_digest", "comparison_envelope_digest",
    "cwd_policy", "environment", "evidence_reuse", "execution_shape", "fixture_recipe_digest", "kind",
    "materialization_policy_digest", "network_mode", "projection_definition_digest", "readiness", "repeat_schedule",
    "required_tools", "schema_version", "secret_slots", "setup_argv", "start_argv", "trust_boundary",
  ];
  if (!exactKeys(plan, keys) || plan.schema_version !== "countershape/v1" || plan.kind !== "WorldPlan" ||
      plan.cwd_policy !== "MATERIALIZED_ROOT" || plan.network_mode !== "HOST_ALLOWED" ||
      plan.evidence_reuse !== "FORBIDDEN" || plan.trust_boundary !== "TRUSTED_LOCAL_FULL_USER_PERMISSIONS" ||
      plan.projection_definition_digest !== source.projection_binding_digest ||
      plan.projection_definition_digest !== typedDigest("ProjectionDefinition", binding.exact) ||
      !sameCanonical(plan.environment, source.plan_environment) || plan.secret_slots !== null ||
      source.plan_secret_slots !== null || !Array.isArray(plan.start_argv) || plan.start_argv.length !== 2 ||
      plan.start_argv[0] !== "node" || plan.start_argv[1] !== source.entrypoint ||
      plan.setup_argv !== null ||
      !Array.isArray(plan.required_tools) || plan.required_tools.length !== 1 ||
      !exactKeys(plan.required_tools[0], ["name", "version_constraint"]) || plan.required_tools[0].name !== "node" ||
      plan.required_tools[0].version_constraint !== "executed-major-only" ||
      !exactKeys(plan.budgets, ["candidate_count", "http_body_bytes", "materialized_bytes_per_world", "materialized_entry_count", "probe_ms", "proposed_shrink_stimuli", "readiness_ms", "shrink_wall_ms", "single_blob_bytes", "stderr_bytes", "stdout_bytes", "teardown_ms", "total_candidate_trials"])) {
    refuse("CONTRACT_DATA_INVALID");
  }
  for (const name of [
    "candidate_set_digest", "capture_policy_digest", "comparison_envelope_digest", "fixture_recipe_digest",
    "materialization_policy_digest",
  ]) {
    if (!validDigest(plan[name])) refuse("CONTRACT_DATA_INVALID");
  }
  const adapter = plan.adapter;
  const expectedRunner = source.adapter === "CLI"
    ? typedDigest("CLIStudyRunner", Buffer.from("world.ExecuteCLI/U3/opaque-binding/v1", "utf8"))
    : typedDigest("HTTPStudyRunner", Buffer.from("world.ExecuteHTTP/P07B/opaque-binding/child-bind-pipe-ready/v1", "utf8"));
  if (!exactKeys(adapter, ["adapter_version", "domain", "runner_digest"]) || adapter.domain !== source.adapter ||
      adapter.runner_digest !== expectedRunner) refuse("CONTRACT_DATA_INVALID");
  const readiness = plan.readiness;
  if (!exactKeys(readiness, ["kind", "signal_name"])) refuse("CONTRACT_DATA_INVALID");
  if (source.adapter === "CLI") {
    if (adapter.adapter_version !== "cli/v1" || plan.execution_shape !== "ONE_CLI_INVOCATION" ||
        readiness.kind !== "NONE" || readiness.signal_name !== "") refuse("CONTRACT_DATA_INVALID");
  } else if (adapter.adapter_version !== "http/v1" || plan.execution_shape !== "ONE_LOOPBACK_HTTP_REQUEST" ||
      readiness.kind !== "FIXTURE_OWNED_SIGNAL" || readiness.signal_name !== "ready-port-frame") {
    refuse("CONTRACT_DATA_INVALID");
  }
  if (!exactKeys(plan.repeat_schedule, ["concurrency", "confirmation_repeats", "discovery_repeats", "rotation"]) ||
      plan.repeat_schedule.concurrency !== "SEQUENTIAL" ||
      plan.repeat_schedule.rotation !== "ROTATE_START_BY_REPETITION_V1" ||
      !Number.isSafeInteger(plan.repeat_schedule.discovery_repeats) ||
      !Number.isSafeInteger(plan.repeat_schedule.confirmation_repeats) ||
      plan.repeat_schedule.discovery_repeats < 1 || plan.repeat_schedule.discovery_repeats > 5 ||
      plan.repeat_schedule.confirmation_repeats < 1 || plan.repeat_schedule.confirmation_repeats > 5) {
    refuse("CONTRACT_DATA_INVALID");
  }
  const budgets = plan.budgets;
  const inRange = (name, minimum, maximum) => Number.isSafeInteger(budgets[name]) && budgets[name] >= minimum && budgets[name] <= maximum;
  if (!inRange("candidate_count", 2, 4) || !inRange("materialized_entry_count", 1, 100000) ||
      !inRange("materialized_bytes_per_world", 1, 1073741824) || !inRange("single_blob_bytes", 1, 134217728) ||
      !inRange("readiness_ms", 0, 30000) || !inRange("probe_ms", 1, 30000) ||
      !inRange("teardown_ms", 1, 10000) || !inRange("stdout_bytes", 1, 16777216) ||
      !inRange("stderr_bytes", 1, 16777216) || !inRange("http_body_bytes", 1, 16777216) ||
      !inRange("proposed_shrink_stimuli", 0, 200) || !inRange("total_candidate_trials", 1, 2000) ||
      !inRange("shrink_wall_ms", 1, 3600000) || budgets.single_blob_bytes > budgets.materialized_bytes_per_world ||
      source.adapter === "HTTP" && budgets.readiness_ms < 1 ||
      budgets.total_candidate_trials < budgets.candidate_count *
        (plan.repeat_schedule.discovery_repeats + plan.repeat_schedule.confirmation_repeats)) {
    refuse("CONTRACT_DATA_INVALID");
  }
  if (!sameCanonical(plan.budgets.materialized_entry_count, source.limits.materialized_entry_count) ||
      plan.budgets.materialized_bytes_per_world !== source.limits.materialized_bytes_per_world ||
      plan.budgets.single_blob_bytes !== source.limits.single_blob_bytes ||
      plan.budgets.readiness_ms !== source.limits.readiness_ms || plan.budgets.probe_ms !== source.limits.probe_ms ||
      plan.budgets.teardown_ms !== source.limits.teardown_ms || plan.budgets.stdout_bytes !== source.limits.stdout_bytes ||
      plan.budgets.stderr_bytes !== source.limits.stderr_bytes || plan.budgets.http_body_bytes !== source.limits.http_body_bytes) {
    refuse("CONTRACT_DATA_INVALID");
  }
  if (plan.projection_definition_digest !== source.projection_binding_digest ||
      bindingInfo.acceptedChannels.length === 0) refuse("CONTRACT_DATA_INVALID");
  return plan;
}

function validateAdapterProjection(value, source, profileFields, bindingInfo) {
  const keys = [
    "configuration_digest", "field_registry_digest", "fields", "implementation_digest", "kind", "operations",
    "projection_definition_binding_digest", "schema_version", "version",
  ];
  const kind = `${source.adapter}ProjectionDefinition`;
  const version = source.adapter === "CLI" ? "cli-projection/v1" : "http-projection/v1";
  if (!exactKeys(value, keys) || value.schema_version !== "countershape/v1" || value.kind !== kind ||
      value.version !== version || value.projection_definition_binding_digest !== source.projection_binding_digest ||
      !Array.isArray(value.fields) || !sameCanonical(value.fields, profileFields.map((field) => field.field_id)) ||
      !Array.isArray(value.operations) || value.operations.length === 0 || !validDigest(value.configuration_digest) ||
      !validDigest(value.field_registry_digest) || !validDigest(value.implementation_digest)) refuse("CONTRACT_DATA_INVALID");
  const expected = expectedProjectionAuthority(source, profileFields);
  if (value.configuration_digest !== bindingInfo.configurationDigest ||
      value.field_registry_digest !== bindingInfo.fieldRegistryDigest ||
      value.implementation_digest !== bindingInfo.implementationDigest ||
      value.configuration_digest !== expected.configurationDigest ||
      value.field_registry_digest !== expected.fieldRegistryDigest ||
      value.implementation_digest !== expected.implementationDigest ||
      value.operations.length !== bindingInfo.operations.length ||
      !sameCanonical(value.operations, expected.operations)) refuse("CONTRACT_DATA_INVALID");
  for (let index = 0; index < value.operations.length; index += 1) {
    const operation = value.operations[index];
    const bound = bindingInfo.operations[index];
    if (!exactKeys(operation, ["name", "rule_digest", "semantics"]) || operation.name !== bound.name ||
        operation.rule_digest !== bound.rule_digest || typeof operation.semantics !== "string" ||
        Buffer.byteLength(operation.semantics, "utf8") === 0 || Buffer.byteLength(operation.semantics, "utf8") > 4096 ||
        operation.semantics.includes("\0")) refuse("CONTRACT_DATA_INVALID");
  }
}

function validateFileArm(files, stimulusFiles, digestKind) {
  if (!Array.isArray(files) || !Array.isArray(stimulusFiles) || files.length !== stimulusFiles.length || files.length > 256) {
    refuse("CONTRACT_DATA_INVALID");
  }
  let previous = null;
  let totalBytes = 0;
  let totalPathBytes = 0;
  const validPath = digestKind === "HTTPSeedContents" ? validHTTPSeedPath : validRelativePath;
  const result = files.map((file, index) => {
    if (!exactKeys(file, ["contents_base64", "mode", "path"]) || file.mode !== "100644" ||
        typeof file.path !== "string" || Buffer.byteLength(file.path, "utf8") > 1024 || !validPath(file.path) ||
        previous !== null && utf8Compare(previous, file.path) >= 0) {
      refuse("CONTRACT_DATA_INVALID");
    }
    previous = file.path;
    const bytes = strictBase64(file.contents_base64);
    if (bytes.length > 1 << 20) refuse("CONTRACT_DATA_INVALID");
    totalBytes += bytes.length;
    totalPathBytes += Buffer.byteLength(file.path, "utf8");
    const identity = stimulusFiles[index];
    if (!exactKeys(identity, ["contents_bytes", "contents_digest", "mode", "path"]) ||
        identity.path !== file.path || identity.mode !== file.mode || identity.contents_bytes !== bytes.length ||
        identity.contents_digest !== typedDigest(digestKind, bytes)) refuse("CONTRACT_DATA_INVALID");
    return { path: file.path, mode: file.mode, bytes };
  });
  if (totalBytes > 16 << 20 || totalPathBytes > 256 << 10) refuse("CONTRACT_DATA_INVALID");
  for (let left = 0; left < result.length; left += 1) {
    const leftFolded = simpleCaseFold(result[left].path);
    for (let right = left + 1; right < result.length; right += 1) {
      const rightFolded = simpleCaseFold(result[right].path);
      if (leftFolded === rightFolded || rightFolded.startsWith(`${leftFolded}/`) || leftFolded.startsWith(`${rightFolded}/`)) {
        refuse("CONTRACT_DATA_INVALID");
      }
    }
  }
  return result;
}

function validDirectArgv(values) {
  if (!Array.isArray(values) || values.length === 0 || values.length > 64 || values[0] !== "node") return false;
  const shellNames = new Set(["sh", "bash", "zsh", "fish", "dash", "csh", "tcsh", "ksh", "pwsh", "powershell"]);
  let total = 0;
  for (let index = 0; index < values.length; index += 1) {
    const value = values[index];
    if (typeof value !== "string" || Buffer.byteLength(value, "utf8") > 4096 || value.includes("\0") || /\p{Cc}/u.test(value)) return false;
    total += Buffer.byteLength(value, "utf8");
    const parts = value.split("/");
    if (shellNames.has(parts[parts.length - 1]) || value.startsWith("~") || value.includes("\\") ||
        /^[A-Za-z]:/.test(value) || value.startsWith("/") || value.includes("$") || value.includes("`")) return false;
    if (index > 0 && value.startsWith("-") && (!/^--[A-Za-z0-9._-]+$/.test(value))) return false;
    if (index > 0 && value.includes("/")) {
      if (parts.some((part) => part === "" || part === "." || part === "..")) return false;
    }
  }
  return total <= 65536;
}

function digestCanonical(kind, value) {
  return typedDigest(kind, Buffer.from(canonicalText(value), "utf8"));
}

function cliFixtureRecipeDigest() {
  return digestCanonical("CLIFixtureRecipe", ownObject([
    ["authority", "U3_PRIVATE_FIXTURE_ROOT_EXCLUSIVE_REOPEN_REHASH_V1"],
    ["kind", "CLIFixtureRecipe"],
    ["path_policy", "CLEAN_RELATIVE_STRICT_ORDER_NO_ALIAS_PREFIX_COLLISION_OR_GIT_METADATA"],
    ["regular_file_modes", ["100644"]],
    ["root_policy", "NEW_PRIVATE_FIXTURE_ROOT"],
    ["schema_version", "countershape/v1"],
    ["verify_policy", "REOPEN_REGULAR_MODE_SIZE_BYTES_AND_DOMAIN_DIGEST"],
    ["version", "cli-fixture-recipe/v1"],
    ["write_policy", "EXCLUSIVE_NO_OVERWRITE_SYNCED"],
  ]));
}

function cliCapturePolicyDigest(source) {
  return digestCanonical("CLICapturePolicy", ownObject([
    ["adapter", "CLI"],
    ["channels", [
      ownObject([["max_bytes", source.cli.stdout_bytes], ["name", "stdout"]]),
      ownObject([["max_bytes", source.cli.stderr_bytes], ["name", "stderr"]]),
    ]],
    ["kind", "CLICapturePolicy"],
    ["schema_version", "countershape/v1"],
    ["version", "cli-capture/v1"],
  ]));
}

function validateCLIMeasure(measure, cli, stdin, environment, fixtures) {
  const keys = [
    "argv_bytes", "argv_items", "environment_entries", "environment_value_bytes", "fixture_content_bytes",
    "fixture_files", "fixture_path_bytes", "stdin_bytes", "stdin_presence_units",
  ];
  const expected = ownObject([
    ["argv_bytes", cli.argv.reduce((total, value) => total + Buffer.byteLength(value, "utf8"), 0)],
    ["argv_items", cli.argv.length],
    ["environment_entries", environment.length],
    ["environment_value_bytes", environment.reduce((total, value) => total + Buffer.byteLength(value.value, "utf8"), 0)],
    ["fixture_content_bytes", fixtures.reduce((total, value) => total + value.bytes.length, 0)],
    ["fixture_files", fixtures.length],
    ["fixture_path_bytes", fixtures.reduce((total, value) => total + Buffer.byteLength(value.path, "utf8"), 0)],
    ["stdin_bytes", stdin.length],
    ["stdin_presence_units", cli.stdin.presence === "PRESENT" ? 1 : 0],
  ]);
  if (!exactKeys(measure, keys) || !sameCanonical(measure, expected)) refuse("CONTRACT_DATA_INVALID");
}

function validateCLIArm(source, stimulus, execution, plan) {
  const cli = source.cli;
  if (!exactKeys(cli, ["argv", "base_argv", "cwd_policy", "environment", "executable", "fixtures", "stderr_bytes", "stdin", "stdout_bytes"]) ||
      source.http !== null || cli.executable !== "node" || cli.cwd_policy !== "MATERIALIZED_ROOT" ||
      !Array.isArray(cli.base_argv) || cli.base_argv.length !== 1 || cli.base_argv[0] !== source.entrypoint ||
      !Array.isArray(cli.argv) || cli.argv.some((value) => typeof value !== "string" || value.includes("\0")) ||
      !validDirectArgv([cli.executable, ...cli.base_argv, ...cli.argv]) ||
      !Number.isSafeInteger(cli.stdout_bytes) || !Number.isSafeInteger(cli.stderr_bytes) ||
      cli.stdout_bytes !== source.limits.stdout_bytes || cli.stderr_bytes !== source.limits.stderr_bytes) refuse("CONTRACT_DATA_INVALID");
  if (!exactKeys(stimulus, ["argv", "base_argv", "cwd_policy", "environment", "executable", "execution_payload_digest", "fixtures", "kind", "measure", "schema_version", "stdin"]) ||
      stimulus.schema_version !== "countershape/v1" || stimulus.kind !== "CLIStimulus" ||
      stimulus.executable !== cli.executable || !sameCanonical(stimulus.base_argv, cli.base_argv) ||
      !sameCanonical(stimulus.argv, cli.argv) || stimulus.cwd_policy !== cli.cwd_policy ||
      !sameCanonical(stimulus.environment, cli.environment)) refuse("CONTRACT_DATA_INVALID");
  if (!exactKeys(cli.stdin, ["bytes_base64", "presence"]) || !exactKeys(stimulus.stdin, ["byte_length", "bytes_digest", "presence"])) {
    refuse("CONTRACT_DATA_INVALID");
  }
  const stdin = strictBase64(cli.stdin.bytes_base64);
  if (!(["ABSENT", "PRESENT"].includes(cli.stdin.presence)) || cli.stdin.presence === "ABSENT" && stdin.length !== 0 ||
      stdin.length > 1 << 20 ||
      stimulus.stdin.presence !== cli.stdin.presence || stimulus.stdin.byte_length !== stdin.length ||
      stimulus.stdin.bytes_digest !== (cli.stdin.presence === "PRESENT" ? typedDigest("CLIStdinBytes", stdin) : "")) {
    refuse("CONTRACT_DATA_INVALID");
  }
  const environment = validateStimulusEnvironment(cli.environment);
  const fixtures = validateFileArm(cli.fixtures, stimulus.fixtures, "CLIFixtureContents");
  validateCLIMeasure(stimulus.measure, cli, stdin, environment, fixtures);
  const executionPayload = ownObject([
    ["cwd_policy", cli.cwd_policy], ["environment", stimulus.environment], ["executable", cli.executable],
    ["fixtures", stimulus.fixtures], ["kind", "CLIExecutionPayload"],
    ["logical_argv", [cli.executable, ...cli.base_argv, ...cli.argv]], ["schema_version", "countershape/v1"],
    ["stdin", stimulus.stdin],
  ]);
  const executionPayloadDigest = digestCanonical("CLIExecutionPayload", executionPayload);
  const fixtureRecipeDigest = cliFixtureRecipeDigest();
  const capturePolicyDigest = cliCapturePolicyDigest(source);
  const executionKeys = [
    "capture_policy_digest", "cli_adapter_projection_definition_digest", "execution_payload_digest",
    "fixture_recipe_digest", "kind", "projection_definition_digest", "schema_version", "stderr_capture_bytes",
    "stdout_capture_bytes", "stimulus_digest", "world_plan_digest",
  ];
  if (!exactKeys(execution, executionKeys) || execution.kind !== "CLIExecutionBinding" || execution.schema_version !== "countershape/v1" ||
      execution.world_plan_digest !== source.world_plan_digest || execution.stimulus_digest !== source.stimulus_digest ||
      stimulus.execution_payload_digest !== executionPayloadDigest ||
      execution.execution_payload_digest !== executionPayloadDigest ||
      execution.projection_definition_digest !== source.projection_binding_digest ||
      execution.cli_adapter_projection_definition_digest !== source.adapter_projection_definition_digest ||
      execution.fixture_recipe_digest !== fixtureRecipeDigest || plan.fixture_recipe_digest !== fixtureRecipeDigest ||
      execution.capture_policy_digest !== capturePolicyDigest || plan.capture_policy_digest !== capturePolicyDigest ||
      execution.stdout_capture_bytes !== cli.stdout_bytes || execution.stderr_capture_bytes !== cli.stderr_bytes) {
    refuse("CONTRACT_DATA_INVALID");
  }
  return { argv: [...cli.base_argv, ...cli.argv], stdin, stdinPresent: cli.stdin.presence === "PRESENT", environment, fixtures };
}

function httpFixtureRecipeDigest() {
  return digestCanonical("HTTPFixtureRecipe", ownObject([
    ["authority", "PRIVATE_SEED_ROOT_EXCLUSIVE_REOPEN_REHASH_V1"], ["kind", "HTTPFixtureRecipe"],
    ["path_policy", "STRICT_ORDER_NO_ALIAS_PREFIX_COLLISION_OR_GIT_METADATA"],
    ["regular_file_modes", ["100644"]], ["root_policy", "NEW_PRIVATE_FIXTURE_ROOT"],
    ["schema_version", "countershape/v1"],
    ["verify_policy", "REOPEN_REGULAR_MODE_SIZE_BYTES_AND_DOMAIN_DIGEST"],
    ["version", "http-fixture-recipe/v1"], ["write_policy", "EXCLUSIVE_NO_OVERWRITE_SYNCED"],
  ]));
}

function httpCapturePolicyDigest(http) {
  return digestCanonical("HTTPCapturePolicy", ownObject([
    ["adapter", "HTTP"], ["body_bytes", http.body_bytes], ["content_encoding_mode", "IDENTITY_ONLY"],
    ["header_bytes", http.header_bytes], ["header_count", http.header_count], ["kind", "HTTPCapturePolicy"],
    ["schema_version", "countershape/v1"], ["status_line_bytes", http.status_line_bytes],
    ["transfer_mode", "CONTENT_LENGTH_ONLY"], ["version", "http-capture/v1"],
  ]));
}

function portableHTTPStartDigest(entrypoint) {
  return digestCanonical("HTTPStartSpec", ownObject([
    ["authority", "NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1"], ["kind", "HTTPStartSpec"],
    ["logical_argv", ["node", entrypoint]], ["schema_version", "countershape/v1"], ["setup_argv", []],
    ["version", "http-start/child-bind-pipe-ready/v1"],
  ]));
}

function portableHTTPReadinessDigest() {
  return digestCanonical("HTTPReadinessContract", ownObject([
    ["eof_required", true], ["frame_max_bytes", 32], ["frame_prefix", "COUNTERSHAPE_READY_V1 "],
    ["http_probe", false], ["kind", "HTTPReadinessContract"],
    ["protocol", "ASCII_COUNTERSHAPE_READY_V1_SPACE_PORT_LF_THEN_EOF_V1"],
    ["schema_version", "countershape/v1"], ["signal_name", "ready-port-frame"],
    ["version", "http-readiness/child-port-frame/v1"],
  ]));
}

function validateHTTPMeasure(measure, http, body, query, headers, seeds) {
  const keys = [
    "body_bytes", "body_presence_units", "header_bytes", "header_entries", "query_bytes", "query_entries",
    "seed_content_bytes", "seed_files", "seed_path_bytes",
  ];
  const expected = ownObject([
    ["body_bytes", body.length], ["body_presence_units", http.body.presence === "PRESENT" ? 1 : 0],
    ["header_bytes", headers.reduce((total, value) => total + Buffer.byteLength(value.name, "utf8") + Buffer.byteLength(value.value, "utf8"), 0)],
    ["header_entries", headers.length],
    ["query_bytes", query.reduce((total, value) => total + Buffer.byteLength(value.name, "utf8") + Buffer.byteLength(value.value, "utf8"), 0)],
    ["query_entries", query.length],
    ["seed_content_bytes", seeds.reduce((total, value) => total + value.bytes.length, 0)],
    ["seed_files", seeds.length],
    ["seed_path_bytes", seeds.reduce((total, value) => total + Buffer.byteLength(value.path, "utf8"), 0)],
  ]);
  if (!exactKeys(measure, keys) || !sameCanonical(measure, expected)) refuse("CONTRACT_DATA_INVALID");
}

function validateHTTPArm(source, stimulus, execution, plan) {
  const http = source.http;
  const keys = [
    "body", "body_bytes", "header_bytes", "header_count", "method", "ordered_query_multimap",
    "ordered_request_header_multimap", "path", "readiness_protocol", "readiness_signal", "seeds",
    "start_authority", "status_line_bytes",
  ];
  if (!exactKeys(http, keys) || source.cli !== null ||
      !["GET", "POST", "PUT", "PATCH", "DELETE"].includes(http.method) || !validHTTPPath(http.path) ||
      http.start_authority !== "NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1" || http.readiness_signal !== "ready-port-frame" ||
      http.readiness_protocol !== "ASCII_COUNTERSHAPE_READY_V1_SPACE_PORT_LF_THEN_EOF_V1" ||
      !exactKeys(http.body, ["bytes_base64", "presence"])) refuse("CONTRACT_DATA_INVALID");
  const body = strictBase64(http.body.bytes_base64);
  if (!(["ABSENT", "PRESENT"].includes(http.body.presence)) || http.body.presence === "ABSENT" && body.length !== 0) {
    refuse("CONTRACT_DATA_INVALID");
  }
  if (body.length > 1 << 20) refuse("CONTRACT_DATA_INVALID");
  const query = validateQuery(http.ordered_query_multimap);
  const headers = validateRequestHeaders(http.ordered_request_header_multimap);
  if (!exactKeys(stimulus, ["body", "declarative_seed_files", "execution_payload_digest", "kind", "measure", "method", "ordered_query_multimap", "ordered_request_header_multimap", "path", "schema_version"]) ||
      stimulus.schema_version !== "countershape/v1" || stimulus.kind !== "HTTPStimulus" || stimulus.method !== http.method ||
      stimulus.path !== http.path || !sameCanonical(stimulus.ordered_query_multimap, query.map(({ name, presence, value }) => ownObject([["name", name], ["value", value], ["value_presence", presence]]))) ||
      !sameCanonical(stimulus.ordered_request_header_multimap, headers.map(({ name, value }) => ownObject([["name", name], ["value", value]]))) ||
      !exactKeys(stimulus.body, ["byte_digest", "byte_length", "presence"]) || stimulus.body.presence !== http.body.presence ||
      stimulus.body.byte_length !== body.length || stimulus.body.byte_digest !== (http.body.presence === "PRESENT" ? typedDigest("HTTPRequestBodyBytes", body) : "")) {
    refuse("CONTRACT_DATA_INVALID");
  }
  const seeds = validateFileArm(http.seeds, stimulus.declarative_seed_files, "HTTPSeedContents");
  validateHTTPMeasure(stimulus.measure, http, body, query, headers, seeds);
  const executionPayload = ownObject([
    ["body", stimulus.body], ["declarative_seed_files", stimulus.declarative_seed_files],
    ["kind", "HTTPExecutionPayload"], ["method", stimulus.method],
    ["ordered_query_multimap", stimulus.ordered_query_multimap],
    ["ordered_request_header_multimap", stimulus.ordered_request_header_multimap], ["path", stimulus.path],
    ["schema_version", "countershape/v1"],
  ]);
  const executionPayloadDigest = digestCanonical("HTTPExecutionPayload", executionPayload);
  const fixtureRecipeDigest = httpFixtureRecipeDigest();
  const capturePolicyDigest = httpCapturePolicyDigest(http);
  const startSpecDigest = portableHTTPStartDigest(source.entrypoint);
  const readinessContractDigest = portableHTTPReadinessDigest();
  const executionKeys = [
    "authority", "capture_policy_digest", "execution_payload_digest", "fixture_recipe_digest",
    "http_adapter_projection_definition_digest", "http_body_capture_bytes", "kind", "projection_definition_digest",
    "readiness_contract_digest", "schema_version", "start_spec_digest", "stimulus_digest", "world_plan_digest",
  ];
  if (!exactKeys(execution, executionKeys) || execution.schema_version !== "countershape/v1" || execution.kind !== "HTTPExecutionBinding" ||
      execution.authority !== "P07B_OPAQUE_HTTP_CHILD_BIND_EXECUTION_BINDING_V1" ||
      execution.world_plan_digest !== source.world_plan_digest || execution.stimulus_digest !== source.stimulus_digest ||
      stimulus.execution_payload_digest !== executionPayloadDigest ||
      execution.execution_payload_digest !== executionPayloadDigest ||
      execution.projection_definition_digest !== source.projection_binding_digest ||
      execution.http_adapter_projection_definition_digest !== source.adapter_projection_definition_digest ||
      execution.fixture_recipe_digest !== fixtureRecipeDigest || plan.fixture_recipe_digest !== fixtureRecipeDigest ||
      execution.capture_policy_digest !== capturePolicyDigest || plan.capture_policy_digest !== capturePolicyDigest ||
      execution.start_spec_digest !== startSpecDigest ||
      execution.readiness_contract_digest !== readinessContractDigest ||
      execution.http_body_capture_bytes !== http.body_bytes) refuse("CONTRACT_DATA_INVALID");
  for (const name of ["status_line_bytes", "header_bytes", "header_count", "body_bytes"]) {
    if (!Number.isSafeInteger(http[name]) || http[name] < 1) refuse("CONTRACT_DATA_INVALID");
  }
  if (http.status_line_bytes < 16 || http.status_line_bytes > 8 << 10 ||
      http.header_bytes < 16 || http.header_bytes > 1 << 20 || http.header_count > 1024 ||
      http.body_bytes > 16 << 20 || http.body_bytes !== source.limits.http_body_bytes) refuse("CONTRACT_DATA_INVALID");
  return { method: http.method, path: http.path, query, headers, body, bodyPresent: http.body.presence === "PRESENT", seeds, capture: http };
}

function validateStimulusEnvironment(entries) {
  if (!Array.isArray(entries) || entries.length > 64) refuse("CONTRACT_DATA_INVALID");
  let previous = null;
  let totalBytes = 0;
  const reserved = new Set([
    "HOME", "PATH", "TMPDIR", "TMP", "TEMP", "PWD", "OLDPWD", "SHELL", "NODE_OPTIONS", "BASH_ENV", "ENV",
    "RUBYOPT", "PERL5OPT", "PYTHONPATH", "PYTHONHOME", "GODEBUG",
  ]);
  const result = entries.map((entry) => {
    if (!exactKeys(entry, ["name", "presence", "value"]) || !/^[A-Z_][A-Z0-9_]*$/.test(entry.name) ||
        Buffer.byteLength(entry.name, "utf8") > 128 || reserved.has(entry.name) ||
        entry.name.startsWith("XDG_") || entry.name.startsWith("COUNTERSHAPE_") || entry.name.startsWith("DYLD_") ||
        !["ABSENT", "PRESENT"].includes(entry.presence) || entry.presence === "ABSENT" && entry.value !== "" ||
        typeof entry.value !== "string" || Buffer.byteLength(entry.value, "utf8") > 16 << 10 ||
        entry.value.includes("\0") || /\p{Cc}/u.test(entry.value) || previous !== null && entry.name <= previous) {
      refuse("CONTRACT_DATA_INVALID");
    }
    previous = entry.name;
    totalBytes += Buffer.byteLength(entry.name, "utf8") + Buffer.byteLength(entry.value, "utf8");
    return { name: entry.name, presence: entry.presence, value: entry.value };
  });
  if (totalBytes > 128 << 10) refuse("CONTRACT_DATA_INVALID");
  return result;
}

function validateQuery(entries) {
  if (!Array.isArray(entries) || entries.length > 128) refuse("CONTRACT_DATA_INVALID");
  let totalBytes = 0;
  const result = entries.map((entry) => {
    if (!exactKeys(entry, ["name", "presence", "value"]) || !["ABSENT", "PRESENT"].includes(entry.presence) ||
        entry.presence === "ABSENT" && entry.value !== "" || typeof entry.name !== "string" ||
        typeof entry.value !== "string" || Buffer.byteLength(entry.name, "utf8") > 4096 ||
        Buffer.byteLength(entry.value, "utf8") > 4096 || /[\u0000-\u001f\u007f]/u.test(entry.name) ||
        /[\u0000-\u001f\u007f]/u.test(entry.value)) refuse("CONTRACT_DATA_INVALID");
    totalBytes += Buffer.byteLength(entry.name, "utf8") + Buffer.byteLength(entry.value, "utf8");
    return { name: entry.name, presence: entry.presence, value: entry.value };
  });
  if (totalBytes > 64 << 10) refuse("CONTRACT_DATA_INVALID");
  return result;
}

function validateRequestHeaders(entries) {
  if (!Array.isArray(entries) || entries.length > 128) refuse("CONTRACT_DATA_INVALID");
  const forbidden = new Set(["host", "connection", "content-length", "transfer-encoding", "content-encoding", "expect", "cookie", "authorization", "proxy-authorization", "accept-encoding", "upgrade", "te"]);
  let totalBytes = 0;
  const result = entries.map((entry) => {
    if (!exactKeys(entry, ["name", "value"]) || typeof entry.name !== "string" || typeof entry.value !== "string" ||
        Buffer.byteLength(entry.name, "ascii") > 128 || Buffer.byteLength(entry.value, "ascii") > 8192 ||
        !/^[!#$%&'*+.^_`|~0-9a-z-]+$/.test(entry.name) || forbidden.has(entry.name) || entry.name.startsWith("proxy-") ||
        !/^[\x20-\x7e]*$/.test(entry.value)) refuse("CONTRACT_DATA_INVALID");
    totalBytes += entry.name.length + entry.value.length;
    return { name: entry.name, value: entry.value };
  });
  if (totalBytes > 64 << 10) refuse("CONTRACT_DATA_INVALID");
  return result;
}

function parsePortableSource(exact, expectedDigest) {
  if (exact.length === 0 || exact.length > 384 << 10 || typedDigest("PortableSource", exact) !== expectedDigest) {
    refuse("CONTRACT_DATA_INVALID");
  }
  const source = parseCanonicalJSON(exact);
  if (!exactKeys(source, SOURCE_KEYS) || source.schema_version !== "countershape/v1" || source.kind !== "PortableSource" ||
      source.version !== "portable-source/v1" || source.source_profile !== "countershape-node-core-exact/v1" ||
      source.launch_profile !== "NODE_REPO_SCRIPT_V1" || !["CLI", "HTTP"].includes(source.adapter) ||
      !validEntrypoint(source.entrypoint) || !exactKeys(source.limits, LIMIT_KEYS) ||
      !(source.plan_environment === null || Array.isArray(source.plan_environment)) || source.plan_secret_slots !== null ||
      !exactKeys(source.closed_facts, ["ambient_executable_lookup", "confidentiality_established", "environment_profile", "external_host", "private_root_policy", "setup_command", "shell_execution"]) ||
      source.closed_facts.private_root_policy !== "NEW_PRIVATE_ROOT_COPY_VERIFIED_SOURCE_V1" ||
      source.closed_facts.environment_profile !== "EXPLICIT_PLAN_PLUS_STIMULUS_NO_AMBIENT_INHERITANCE_V1" ||
      ["ambient_executable_lookup", "confidentiality_established", "external_host", "setup_command", "shell_execution"].some((name) => source.closed_facts[name] !== false)) {
    refuse("CONTRACT_DATA_INVALID");
  }
  const planEnvironment = source.plan_environment ?? [];
  let previousEnvironment = null;
  const publicPlanEnvironment = new Map([
    ["LANG", "C"], ["LC_ALL", "C"], ["NODE_NO_WARNINGS", "1"], ["NO_COLOR", "1"], ["TZ", "UTC"],
  ]);
  for (const entry of planEnvironment) {
    if (!exactKeys(entry, ["name", "value"]) || publicPlanEnvironment.get(entry.name) !== entry.value ||
        previousEnvironment !== null && entry.name <= previousEnvironment) refuse("CONTRACT_DATA_INVALID");
    previousEnvironment = entry.name;
  }
  for (const name of LIMIT_KEYS) {
    if (!Number.isSafeInteger(source.limits[name]) || source.limits[name] < 0) refuse("CONTRACT_DATA_INVALID");
  }
  if (source.limits.materialized_entry_count < 1 || source.limits.materialized_bytes_per_world < 1 ||
      source.limits.single_blob_bytes < 1 || source.limits.probe_ms < 1 || source.limits.teardown_ms < 1) {
    refuse("CONTRACT_DATA_INVALID");
  }
  const binding = decodeAuthority(source.projection_binding_base64, source.projection_binding_digest, "ProjectionDefinition");
  const plan = decodeAuthority(source.world_plan_base64, source.world_plan_digest, "WorldPlan");
  const profile = decodeAuthority(source.portable_profile_base64, source.portable_profile_digest, "PortableProjectionProfile");
  const projection = decodeAuthority(source.adapter_projection_definition_base64, source.adapter_projection_definition_digest, `${source.adapter}ProjectionDefinition`);
  const stimulus = decodeAuthority(source.stimulus_base64, source.stimulus_digest, source.stimulus_kind);
  const execution = decodeAuthority(source.execution_binding_base64, source.execution_binding_digest, `${source.adapter}ExecutionBinding`);
  const bindingInfo = validateProjectionBinding(binding.value, source);
  const profileFields = validateProfile(profile.value, binding, source);
  const planValue = validateWorldPlan(plan.value, source, binding, bindingInfo);
  validateAdapterProjection(projection.value, source, profileFields, bindingInfo);
  const runtime = source.adapter === "CLI"
    ? validateCLIArm(source, stimulus.value, execution.value, planValue)
    : validateHTTPArm(source, stimulus.value, execution.value, planValue);
  if (source.adapter === "CLI" && (source.start_profile !== "DIRECT_CHILD_V1" || source.stimulus_kind !== "CLIStimulus")) refuse("CONTRACT_DATA_INVALID");
  if (source.adapter === "HTTP" && (source.start_profile !== "NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1" || source.stimulus_kind !== "HTTPStimulus")) refuse("CONTRACT_DATA_INVALID");
  return { source, planEnvironment, profileFields, runtime };
}

function validateExactValue(value, descriptor) {
  if (value === null || typeof value !== "object" || Array.isArray(value) || typeof value.tag !== "string") refuse("CONTRACT_DATA_INVALID");
  if (value.tag === "MISSING") {
    if (!descriptor.allow_missing || !exactKeys(value, ["tag"])) refuse("CONTRACT_DATA_INVALID");
    return;
  }
  if (value.tag === "NULL") {
    if (!descriptor.allow_null || !exactKeys(value, ["tag"])) refuse("CONTRACT_DATA_INVALID");
    return;
  }
  if (value.tag !== descriptor.present_tag) refuse("CONTRACT_DATA_INVALID");
  switch (value.tag) {
    case "BOOLEAN":
      if (!exactKeys(value, ["tag", "value"]) || typeof value.value !== "boolean") refuse("CONTRACT_DATA_INVALID");
      break;
    case "INTEGER":
      if (!exactKeys(value, ["canonical", "tag"]) || typeof value.canonical !== "string" ||
          !/^-?(0|[1-9][0-9]*)$/.test(value.canonical) || value.canonical === "-0" || !Number.isSafeInteger(Number(value.canonical))) refuse("CONTRACT_DATA_INVALID");
      break;
    case "STRING":
      if (!exactKeys(value, ["tag", "value"]) || typeof value.value !== "string" ||
          Buffer.byteLength(value.value, "utf8") > MAX_PORTABLE_VALUE_BYTES) refuse("CONTRACT_DATA_INVALID");
      break;
    case "BYTES":
      if (!exactKeys(value, ["base64", "tag"]) || strictBase64(value.base64).length > MAX_PORTABLE_VALUE_BYTES) refuse("CONTRACT_DATA_INVALID");
      break;
    case "ORDERED_STRING_LIST": {
      if (!exactKeys(value, ["tag", "values"]) || !Array.isArray(value.values) ||
          value.values.length > MAX_PORTABLE_LIST_MEMBERS || value.values.some((item) => typeof item !== "string")) {
        refuse("CONTRACT_DATA_INVALID");
      }
      let retained = 0;
      for (const member of value.values) {
        const length = Buffer.byteLength(member, "utf8");
        if (length > MAX_PORTABLE_VALUE_BYTES) refuse("CONTRACT_DATA_INVALID");
        retained += length;
        if (retained > MAX_PORTABLE_VALUE_BYTES) refuse("CONTRACT_DATA_INVALID");
      }
      if (Buffer.byteLength(canonicalText(value.values), "utf8") > MAX_PORTABLE_VALUE_BYTES) refuse("CONTRACT_DATA_INVALID");
      break;
    }
    case "CANONICAL_JSON": {
      if (!exactKeys(value, ["canonical_base64", "tag"])) refuse("CONTRACT_DATA_INVALID");
      const exact = strictBase64(value.canonical_base64);
      if (exact.length > MAX_PORTABLE_VALUE_BYTES) refuse("CONTRACT_DATA_INVALID");
      parseCanonicalJSON(exact);
      break;
    }
    default:
      refuse("CONTRACT_DATA_INVALID");
  }
}

function parsePredicate(predicate, sourceInfo) {
  if (!exactKeys(predicate, ["allowed_tuples", "kind", "portable_profile_digest", "scope", "selected_fields", "stimulus_digest"]) ||
      predicate.kind !== "one-of-exact/v1" || predicate.scope !== "EXACT_WITNESSED_STIMULUS" ||
      predicate.portable_profile_digest !== sourceInfo.source.portable_profile_digest ||
      predicate.stimulus_digest !== sourceInfo.source.stimulus_digest || !Array.isArray(predicate.selected_fields) ||
      predicate.selected_fields.length === 0 || !Array.isArray(predicate.allowed_tuples) ||
      predicate.allowed_tuples.length === 0 || predicate.allowed_tuples.length > 4) refuse("CONTRACT_DATA_INVALID");
  const profileByID = new Map(sourceInfo.profileFields.map((field, index) => [field.field_id, { ...field, index }]));
  let previous = -1;
  const descriptors = predicate.selected_fields.map((field) => {
    const descriptor = profileByID.get(field);
    if (!descriptor || descriptor.index <= previous) refuse("CONTRACT_DATA_INVALID");
    previous = descriptor.index;
    return descriptor;
  });
  let previousTuple = null;
  const allowed = predicate.allowed_tuples.map((tuple) => {
    if (!exactKeys(tuple, ["fields"]) || !Array.isArray(tuple.fields) || tuple.fields.length !== descriptors.length) refuse("CONTRACT_DATA_INVALID");
    for (let index = 0; index < tuple.fields.length; index += 1) {
      const field = tuple.fields[index];
      if (!exactKeys(field, ["field_id", "value"]) || field.field_id !== predicate.selected_fields[index]) refuse("CONTRACT_DATA_INVALID");
      validateExactValue(field.value, descriptors[index]);
    }
    validatePortableTuple(tuple.fields.map((field) => field.value), "CONTRACT_DATA_INVALID");
    const exact = Buffer.from(canonicalText(tuple), "utf8");
    if (previousTuple !== null && Buffer.compare(previousTuple, exact) >= 0) refuse("CONTRACT_DATA_INVALID");
    previousTuple = exact;
    return exact;
  });
  return { selected: predicate.selected_fields, descriptors, allowed };
}

function parseContractData(decisionBytes, fixtureBytes) {
  const fixture = parseJSONEnvelope(fixtureBytes);
  if (!exactKeys(fixture, ["fixture_version", "kind", "portable_source_base64", "portable_source_digest", "schema_version"]) ||
      fixture.schema_version !== "countershape-contract/v1" || fixture.kind !== "PortableFixture" ||
      fixture.fixture_version !== "portable-source-fixture/v1" || !validDigest(fixture.portable_source_digest)) {
    refuse("CONTRACT_DATA_INVALID");
  }
  const sourceBytes = strictBase64(fixture.portable_source_base64);
  const sourceInfo = parsePortableSource(sourceBytes, fixture.portable_source_digest);
  const decision = parseJSONEnvelope(decisionBytes);
  if (!exactKeys(decision, ["decision_action", "kind", "predicate", "schema_version"]) ||
      decision.schema_version !== "countershape-contract/v1" || decision.kind !== "CompiledDecision" ||
      !["ALLOW_OBSERVED", "CUSTOM_EXPECTATION"].includes(decision.decision_action)) refuse("CONTRACT_DATA_INVALID");
  const predicate = parsePredicate(decision.predicate, sourceInfo);
  if (decision.decision_action === "CUSTOM_EXPECTATION" && predicate.allowed.length !== 1) refuse("CONTRACT_DATA_INVALID");
  return { ...sourceInfo, predicate, action: decision.decision_action };
}

function observedTuple(predicate, values) {
  const fields = predicate.selected.map((fieldID) => {
    if (!values.has(fieldID)) refuse("PROJECTION_FAILED");
    return ownObject([["field_id", fieldID], ["value", values.get(fieldID)]]);
  });
  validatePortableTuple(fields.map((field) => field.value), "PROJECTION_FAILED");
  return Buffer.from(canonicalText(ownObject([["fields", fields]])), "utf8");
}

function predicateMatches(predicate, tuple) {
  return predicate.allowed.some((allowed) => allowed.equals(tuple));
}

function utf8Compare(left, right) {
  return Buffer.compare(Buffer.from(left, "utf8"), Buffer.from(right, "utf8"));
}

// Go strings.ToLower applies Unicode simple case mappings one rune at a time.
// JavaScript's full-string lowercase may expand or contextually rewrite a
// character, so fixture/path alias checks deliberately retain only the first
// code point of each single-rune lowercase result.
function simpleCaseFold(value) {
  let result = "";
  for (const character of value) {
    const lowered = [...character.toLowerCase()];
    result += lowered[0] ?? character;
  }
  return result;
}

function validEntrypoint(value) {
  return typeof value === "string" && value.length > 0 && value.length <= 4096 && !value.startsWith("/") &&
    !value.startsWith("-") && !value.includes("\\") && /\.(?:js|mjs|cjs)$/.test(value) &&
    value.split("/").every((part) => /^[A-Za-z0-9_-][A-Za-z0-9_.-]*$/.test(part));
}

function validRelativePath(value) {
  return typeof value === "string" && value.length > 0 && value.length <= 4096 && !isAbsolute(value) &&
    !value.includes("\\") && !value.includes("\0") && value.split("/").every((part) => part !== "" && part !== "." && part !== ".." && simpleCaseFold(part) !== ".git");
}

function validHTTPSeedPath(value) {
  return typeof value === "string" && Buffer.byteLength(value, "utf8") > 0 && Buffer.byteLength(value, "utf8") <= 1024 &&
    !value.startsWith("/") && !value.includes("\\") && value !== "." &&
    value.split("/").every((part) => part !== "" && part !== "." && part !== ".." && simpleCaseFold(part) !== ".git");
}

function validHTTPPath(value) {
  return typeof value === "string" && value.length > 0 && value.length <= 4096 && value.startsWith("/") &&
    !value.includes("//") && !/[?#\\\x00]/.test(value) && /^[\x21-\x7e]+$/.test(value);
}

class RuntimeRefusal extends Error {
  constructor(reason) {
    super(reason);
    this.reason = reason;
  }
}

function addReason(facts, reason) {
  if (!facts.ineligible_reasons.includes(reason)) facts.ineligible_reasons.push(reason);
}

function within(parent, candidate) {
  const path = relative(parent, candidate);
  return path === "" || path !== ".." && !path.startsWith(`..${sep}`) && !isAbsolute(path);
}

function canonicalDirectory(input) {
  if (typeof input !== "string" || !isAbsolute(input) || resolve(input) !== input) throw new RuntimeRefusal("EXECUTION_ROOT_FAILED");
  let before;
  try {
    before = lstatSync(input, { bigint: true });
  } catch {
    throw new RuntimeRefusal("EXECUTION_ROOT_FAILED");
  }
  if (!before.isDirectory() || before.isSymbolicLink()) throw new RuntimeRefusal("EXECUTION_ROOT_FAILED");
  let canonical;
  try {
    canonical = realpathSync(input);
  } catch {
    throw new RuntimeRefusal("EXECUTION_ROOT_FAILED");
  }
  if (canonical !== input) throw new RuntimeRefusal("EXECUTION_ROOT_FAILED");
  return canonical;
}

function rootIdentity(input) {
  const stat = lstatSync(input, { bigint: true });
  if (!stat.isDirectory() || stat.isSymbolicLink() || realpathSync(input) !== input) {
    throw new RuntimeRefusal("EXECUTION_ROOT_FAILED");
  }
  return `${stat.dev}:${stat.ino}`;
}

function watcherEnvironmentError(error) {
  return new Set([
    "EACCES", "EPERM", "ENOENT", "ENOTDIR", "ENOSPC", "EMFILE", "ENFILE", "EINVAL",
    "ERR_FEATURE_UNAVAILABLE_ON_PLATFORM",
  ]).has(String(error?.code ?? ""));
}

function closeRootWatchers(...watchers) {
  const errors = [];
  for (const watcher of watchers) {
    if (watcher === null || watcher === undefined || watcher.closed || watcher.handle === null) continue;
    try {
      watcher.handle.close();
      watcher.closed = true;
    } catch (error) {
      errors.push(error);
    }
  }
  return errors;
}

function startRootWatcher(root) {
  const state = { closed: false, events: [], errors: [], handle: null };
  try {
    state.handle = watch(root, { persistent: false, recursive: true }, (event, name) => {
      state.events.push(`${event}:${Buffer.isBuffer(name) ? name.toString("hex") : String(name ?? "")}`);
    });
    state.handle.on("error", (error) => state.errors.push(String(error?.code ?? "WATCH_ERROR")));
  } catch (error) {
    const closeErrors = closeRootWatchers(state);
    if (closeErrors.length !== 0) throw new Error("root watcher acquisition cleanup failed", { cause: error });
    if (watcherEnvironmentError(error)) throw new RuntimeRefusal("ENVIRONMENT_INVALID");
    throw error;
  }
  return state;
}

function startRootWatcherPair(bundle, target) {
  let bundleWatcher = null;
  try {
    bundleWatcher = startRootWatcher(bundle);
    const targetWatcher = startRootWatcher(target);
    return [bundleWatcher, targetWatcher];
  } catch (error) {
    const closeErrors = closeRootWatchers(bundleWatcher);
    if (closeErrors.length !== 0) throw new Error("partial root watcher cleanup failed", { cause: error });
    throw error;
  }
}

function stableFile(path, maximum) {
  let before;
  try {
    before = lstatSync(path, { bigint: true });
  } catch {
    throw new RuntimeRefusal("SOURCE_INVENTORY_INVALID");
  }
  if (!before.isFile() || before.isSymbolicLink() || before.nlink !== 1n || before.size > BigInt(maximum)) {
    throw new RuntimeRefusal("SOURCE_INVENTORY_INVALID");
  }
  let descriptor;
  try {
    descriptor = openSync(path, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
  } catch {
    throw new RuntimeRefusal("SOURCE_INVENTORY_INVALID");
  }
  try {
    const opened = fstatSync(descriptor, { bigint: true });
    if (!opened.isFile() || opened.dev !== before.dev || opened.ino !== before.ino || opened.size !== before.size ||
        opened.mode !== before.mode || opened.nlink !== 1n || opened.mtimeNs !== before.mtimeNs ||
        opened.ctimeNs !== before.ctimeNs) {
      throw new RuntimeRefusal("SOURCE_INVENTORY_INVALID");
    }
    const bytes = readFileSync(descriptor);
    const after = fstatSync(descriptor, { bigint: true });
    if (BigInt(bytes.length) !== opened.size || after.dev !== opened.dev || after.ino !== opened.ino ||
        after.size !== opened.size || after.mode !== opened.mode || after.nlink !== opened.nlink ||
        after.mtimeNs !== opened.mtimeNs || after.ctimeNs !== opened.ctimeNs) {
      throw new RuntimeRefusal("SOURCE_INVENTORY_INVALID");
    }
    return { bytes, stat: opened };
  } finally {
    closeSync(descriptor);
  }
}

function validInventoryPath(value) {
  if (typeof value !== "string" || value === "" || isAbsolute(value) || value.includes("\\") ||
      /^[A-Za-z]:/.test(value) || /[\u0000-\u001f\u007f-\u009f]/u.test(value)) return false;
  const components = value.split("/");
  return components.length <= 128 && Buffer.byteLength(value, "utf8") <= 4096 && components.every((part) =>
    part !== "" && part !== "." && part !== ".." && simpleCaseFold(part) !== ".git" &&
    Buffer.byteLength(part, "utf8") <= 255 && !/[\u0000-\u001f\u007f-\u009f]/u.test(part));
}

function scanInventory(root, limits, failure = "SOURCE_INVENTORY_INVALID") {
  const records = [];
  const fileIdentities = new Set();
  let files = 0;
  let directories = 0;
  let aggregate = 0;
  function walk(physical, logical) {
    let names;
    try {
      names = readdirSync(physical, { encoding: "buffer", withFileTypes: true });
    } catch {
      throw new RuntimeRefusal(failure);
    }
    if (logical !== "" && names.length === 0) throw new RuntimeRefusal(failure);
    names.sort((left, right) => Buffer.compare(Buffer.from(left.name), Buffer.from(right.name)));
    for (const entry of names) {
      const nameBytes = Buffer.from(entry.name);
      if (!validUTF8(nameBytes)) throw new RuntimeRefusal(failure);
      const name = nameBytes.toString("utf8");
      if (!Buffer.from(name, "utf8").equals(nameBytes) || name === "" || name === "." || name === ".." || simpleCaseFold(name) === ".git") {
        throw new RuntimeRefusal(failure);
      }
      const childLogical = logical === "" ? name : `${logical}/${name}`;
      if (!validInventoryPath(childLogical)) throw new RuntimeRefusal(failure);
      const child = join(physical, name);
      let metadata;
      try {
        metadata = lstatSync(child, { bigint: true });
      } catch {
        throw new RuntimeRefusal(failure);
      }
      if (metadata.isDirectory() && !metadata.isSymbolicLink()) {
        directories += 1;
        if (directories > 8192) throw new RuntimeRefusal(failure);
        records.push({ kind: "directory", path: childLogical });
        walk(child, childLogical);
        continue;
      }
      if (!metadata.isFile() || metadata.isSymbolicLink() || metadata.nlink !== 1n) throw new RuntimeRefusal(failure);
      files += 1;
      if (files > limits.materialized_entry_count) throw new RuntimeRefusal(failure);
      const mode = Number(metadata.mode & 0o777n);
      if (mode !== 0o644 && mode !== 0o755 || metadata.size > BigInt(limits.single_blob_bytes)) throw new RuntimeRefusal(failure);
      let opened;
      try {
        opened = stableFile(child, limits.single_blob_bytes);
      } catch {
        throw new RuntimeRefusal(failure);
      }
      const identity = `${opened.stat.dev}:${opened.stat.ino}`;
      if (fileIdentities.has(identity)) throw new RuntimeRefusal(failure);
      fileIdentities.add(identity);
      aggregate += opened.bytes.length;
      if (aggregate > limits.materialized_bytes_per_world) throw new RuntimeRefusal(failure);
      records.push({ kind: "file", path: childLogical, mode, bytes: opened.bytes, digest: rawDigest(opened.bytes) });
    }
  }
  walk(root, "");
  records.sort((left, right) => {
    const order = utf8Compare(left.path, right.path);
    if (order !== 0 || left.kind === right.kind) return order;
    return left.kind === "directory" ? -1 : 1;
  });
  return records;
}

function inventoryIdentity(records) {
  return records.map((record) => record.kind === "directory"
    ? `d\0${record.path}`
    : `f\0${record.path}\0${record.mode.toString(8)}\0${record.bytes.length}\0${record.digest}`).join("\n");
}

function writeExactFile(path, bytes, mode, failure) {
  let descriptor;
  try {
    descriptor = openSync(path, constants.O_WRONLY | constants.O_CREAT | constants.O_EXCL | (constants.O_NOFOLLOW ?? 0), mode);
    writeFileSync(descriptor, bytes);
    fchmodSync(descriptor, mode);
    fsyncSync(descriptor);
    const written = fstatSync(descriptor, { bigint: true });
    if (!written.isFile() || written.nlink !== 1n || written.size !== BigInt(bytes.length) || Number(written.mode & 0o777n) !== mode) {
      throw new Error("write verification failed");
    }
  } catch {
    throw new RuntimeRefusal(failure);
  } finally {
    if (descriptor !== undefined) closeSync(descriptor);
  }
  let reopened;
  try {
    reopened = stableFile(path, bytes.length);
  } catch {
    throw new RuntimeRefusal(failure);
  }
  if (!reopened.bytes.equals(bytes) || rawDigest(reopened.bytes) !== rawDigest(bytes)) throw new RuntimeRefusal(failure);
}

function verifyPrivateDirectory(path, parent, failure) {
  try {
    const metadata = lstatSync(path, { bigint: true });
    if (!metadata.isDirectory() || metadata.isSymbolicLink() || Number(metadata.mode & 0o777n) !== 0o700 ||
        realpathSync(path) !== path || dirname(path) !== parent) throw new Error("private directory verification failed");
  } catch {
    throw new RuntimeRefusal(failure);
  }
}

function allocateExecutionRoot(bundleRoot, targetRoot) {
  let parent;
  try {
    parent = canonicalDirectory(realpathSync(tmpdir()));
  } catch {
    throw new RuntimeRefusal("EXECUTION_ROOT_FAILED");
  }
  if (within(bundleRoot, parent) || within(targetRoot, parent)) throw new RuntimeRefusal("EXECUTION_ROOT_FAILED");
  for (let attempt = 0; attempt < 8; attempt += 1) {
    const path = join(parent, `countershape-${randomBytes(32).toString("hex")}`);
    try {
      mkdirSync(path, { mode: 0o700 });
      const canonical = realpathSync(path);
      if (canonical !== path || dirname(canonical) !== parent || within(bundleRoot, canonical) || within(canonical, bundleRoot) ||
          within(targetRoot, canonical) || within(canonical, targetRoot)) {
        rmSync(path, { recursive: false });
        throw new RuntimeRefusal("EXECUTION_ROOT_FAILED");
      }
      verifyPrivateDirectory(canonical, parent, "EXECUTION_ROOT_FAILED");
      return { parent, root: canonical };
    } catch (error) {
      if (error instanceof RuntimeRefusal) throw error;
      if (error?.code !== "EEXIST") throw new RuntimeRefusal("EXECUTION_ROOT_FAILED");
    }
  }
  throw new RuntimeRefusal("EXECUTION_ROOT_FAILED");
}

function createSubroots(root) {
  const names = ["candidate", "evidence", "fixture", "home", "state", "tmp", "xdg-cache", "xdg-config", "xdg-data", "xdg-state"];
  const result = Object.create(null);
  for (const name of names) {
    const path = join(root, name);
    try {
      mkdirSync(path, { mode: 0o700 });
      verifyPrivateDirectory(path, root, "EXECUTION_ROOT_FAILED");
    } catch {
      throw new RuntimeRefusal("EXECUTION_ROOT_FAILED");
    }
    result[name] = path;
  }
  return result;
}

function copyInventory(records, candidateRoot, limits) {
  for (const record of records) {
    const destination = join(candidateRoot, ...record.path.split("/"));
    if (record.kind === "directory") {
      try {
        mkdirSync(destination, { mode: 0o700 });
        verifyPrivateDirectory(destination, dirname(destination), "SOURCE_COPY_FAILED");
      } catch {
        throw new RuntimeRefusal("SOURCE_COPY_FAILED");
      }
    } else {
      writeExactFile(destination, record.bytes, record.mode, "SOURCE_COPY_FAILED");
    }
  }
  const copied = scanInventory(candidateRoot, limits, "SOURCE_COPY_FAILED");
  if (inventoryIdentity(copied) !== inventoryIdentity(records)) throw new RuntimeRefusal("SOURCE_COPY_FAILED");
}

function validateCombinedHTTPMaterializationBudgets(records, seeds, limits) {
  const candidateFiles = records.filter((record) => record.kind === "file");
  let totalBytes = 0;
  for (const file of candidateFiles) {
    if (file.bytes.length > limits.single_blob_bytes) throw new RuntimeRefusal("FIXTURE_OVERLAY_FAILED");
    totalBytes += file.bytes.length;
  }
  for (const seed of seeds) {
    if (seed.bytes.length > limits.single_blob_bytes) throw new RuntimeRefusal("FIXTURE_OVERLAY_FAILED");
    totalBytes += seed.bytes.length;
  }
  if (candidateFiles.length + seeds.length > limits.materialized_entry_count ||
      totalBytes > limits.materialized_bytes_per_world) {
    throw new RuntimeRefusal("FIXTURE_OVERLAY_FAILED");
  }
}

function materializeOverlay(files, fixtureRoot) {
  const directories = new Set([""]);
  for (const file of files) {
    const parts = file.path.split("/");
    let logical = "";
    for (const part of parts.slice(0, -1)) {
      logical = logical === "" ? part : `${logical}/${part}`;
      if (!directories.has(logical)) {
        try {
          const directory = join(fixtureRoot, ...logical.split("/"));
          mkdirSync(directory, { mode: 0o700 });
          verifyPrivateDirectory(directory, dirname(directory), "FIXTURE_OVERLAY_FAILED");
        } catch {
          throw new RuntimeRefusal("FIXTURE_OVERLAY_FAILED");
        }
        directories.add(logical);
      }
    }
    writeExactFile(join(fixtureRoot, ...parts), file.bytes, 0o644, "FIXTURE_OVERLAY_FAILED");
  }
}

function buildEnvironment(contract, roots) {
  const entries = new Map();
  const add = (name, value) => {
    if (entries.has(name) || typeof value !== "string" || value.includes("\0")) throw new RuntimeRefusal("ENVIRONMENT_INVALID");
    entries.set(name, value);
  };
  for (const entry of contract.planEnvironment) add(entry.name, entry.value);
  if (contract.source.adapter === "CLI") {
    for (const entry of contract.runtime.environment) if (entry.presence === "PRESENT") add(entry.name, entry.value);
  }
  add("HOME", roots.home);
  add("TMPDIR", roots.tmp);
  add("XDG_CONFIG_HOME", roots["xdg-config"]);
  add("XDG_CACHE_HOME", roots["xdg-cache"]);
  add("XDG_DATA_HOME", roots["xdg-data"]);
  add("XDG_STATE_HOME", roots["xdg-state"]);
  add("COUNTERSHAPE_STATE_ROOT", roots.state);
  add("COUNTERSHAPE_EVIDENCE_ROOT", roots.evidence);
  add("COUNTERSHAPE_FIXTURE_ROOT", roots.fixture);
  add("COUNTERSHAPE_ATTEMPT_ID", `attempt:${randomBytes(32).toString("hex")}`);
  add("COUNTERSHAPE_SCHEDULE_ORDINAL", "0");
  add("COUNTERSHAPE_SCHEDULE_REPETITION", "0");
  if (contract.source.adapter === "HTTP") {
    add("COUNTERSHAPE_HTTP_STIMULUS_DIGEST", contract.source.stimulus_digest);
    add("COUNTERSHAPE_HTTP_READINESS_FD", "3");
  }
  const sorted = [...entries.entries()].sort(([left], [right]) => utf8Compare(left, right));
  return ownObject(sorted);
}

function delay(milliseconds) {
  return new Promise((resolvePromise) => setTimeout(resolvePromise, Math.max(0, milliseconds)));
}

function eventLoopTurn() {
  return new Promise((resolvePromise) => setImmediate(resolvePromise));
}

function processGroupProbe(pid) {
  try {
    process.kill(-pid, 0);
    return { error: false, present: true };
  } catch (error) {
    if (error?.code === "ESRCH") return { error: false, present: false };
    if (error?.code === "EPERM") return { error: true, present: true };
    return { error: true, present: false };
  }
}

function monotonicNow() {
  return process.hrtime.bigint();
}

function monotonicDeadline(milliseconds) {
  return monotonicNow() + BigInt(Math.max(0, Math.ceil(milliseconds * 1_000_000)));
}

function monotonicDeadlineFrom(start, milliseconds) {
  return start + BigInt(Math.max(0, Math.ceil(milliseconds * 1_000_000)));
}

function remainingDeadlineMS(deadline) {
  const remaining = deadline - monotonicNow();
  if (remaining <= 0n) return 0;
  return Number((remaining + 999_999n) / 1_000_000n);
}

function earlierDeadline(left, right) {
  return left < right ? left : right;
}

function makeRotatingLatch() {
  let generation = 0;
  let settle;
  let promise = new Promise((resolvePromise) => { settle = resolvePromise; });
  return Object.freeze({
    snapshot() {
      return { generation, promise };
    },
    wake() {
      const previous = settle;
      generation += 1;
      promise = new Promise((resolvePromise) => { settle = resolvePromise; });
      previous();
    },
  });
}

async function waitForLatch(latch, snapshot, deadline) {
  if (latch.snapshot().generation !== snapshot.generation) return "state";
  const remaining = remainingDeadlineMS(deadline);
  if (remaining === 0) return "deadline";
  const timeout = timeoutSignal(remaining, "deadline");
  const result = await Promise.race([snapshot.promise.then(() => "state"), timeout.promise]);
  timeout.cancel();
  if (result === "state") await eventLoopTurn();
  return result;
}

async function awaitOwnerDecision(deadline, latch, poll) {
  while (true) {
    let decision = poll(monotonicNow(), deadline);
    if (decision !== null) return decision;
    const snapshot = latch.snapshot();
    decision = poll(monotonicNow(), deadline);
    if (decision !== null) return decision;
    await waitForLatch(latch, snapshot, deadline);
  }
}

async function awaitSpawnConfirmation(owner, latch, budget) {
  const deadline = monotonicDeadline(budget);
  const decision = await awaitOwnerDecision(deadline, latch, (now) => {
    if (observedBefore(owner.spawned, deadline)) return { kind: "spawned" };
    if (observedBefore(owner.spawnError, deadline) || observedBefore(owner.lifecycleError, deadline) ||
        observedBefore(owner.exit, deadline)) return { kind: "error" };
    if (now >= deadline) return { kind: "timeout" };
    return null;
  });
  return decision.kind === "spawned";
}

function observedBefore(event, deadline) {
  return event !== null && event.observedAt < deadline;
}

function childOwner(child, latch) {
  // Node may publish child.pid before it emits the spawn event, including on
  // asynchronous spawn failure. Only the event freezes signal authority; an
  // unconfirmed PID is never used to prove or perform clean group teardown.
  const owner = {
    child, exit: null, lifecycleError: null, spawnError: null, spawned: null, successfulSignals: [],
  };
  const markLifecycleError = (code = "CHILD_LIFECYCLE_ERROR") => {
    if (owner.lifecycleError === null) owner.lifecycleError = { code, observedAt: monotonicNow() };
    latch.wake();
  };
  const markSpawned = () => {
    if (owner.spawned !== null) return;
    if (owner.spawnError !== null || owner.exit !== null || !Number.isInteger(child?.pid) || child.pid <= 0) {
      markLifecycleError("INVALID_SPAWN_TRANSITION");
      return;
    }
    owner.spawned = { observedAt: monotonicNow(), pid: child.pid };
    latch.wake();
  };
  const markError = (error) => {
    const code = String(error?.code ?? "CHILD_ERROR");
    if (owner.spawned !== null || owner.exit !== null) {
      markLifecycleError(code);
      return;
    }
    if (owner.spawnError !== null) return;
    owner.spawnError = { code, observedAt: monotonicNow() };
    latch.wake();
  };
  const markExit = (code, signal) => {
    if (owner.exit !== null) return;
    if (owner.spawned === null) markLifecycleError("EXIT_BEFORE_SPAWN");
    owner.exit = {
      code, kind: "exit", observedAt: monotonicNow(), signal,
      signalCountAtObservation: owner.successfulSignals.length,
    };
    latch.wake();
  };
  child.once("spawn", markSpawned);
  child.on("error", markError);
  child.once("exit", markExit);
  owner.signal = (signal) => {
    const pid = owner.spawned?.pid;
    if (!Number.isInteger(pid) || pid <= 0) return "error";
    try {
      process.kill(-pid, signal);
      owner.successfulSignals.push(signal);
      return "sent";
    } catch (error) {
      if (error?.code === "ESRCH") return "absent";
      return "error";
    }
  };
  return owner;
}

async function waitForGroupAbsence(pid, deadline) {
  let positiveProbeError = false;
  while (true) {
    if (monotonicNow() >= deadline) return { absent: false, probeError: positiveProbeError };
    const probe = processGroupProbe(pid);
    if (monotonicNow() >= deadline) return { absent: false, probeError: positiveProbeError || probe.error };
    if (!probe.present) {
      return { absent: !probe.error, probeError: probe.error };
    }
    positiveProbeError = probe.error;
    const remaining = remainingDeadlineMS(deadline);
    if (remaining === 0) return { absent: false, probeError: positiveProbeError };
    await delay(Math.min(1, remaining));
  }
}

function addUncertainTeardown(facts) {
  addReason(facts, "TEARDOWN_FAILED");
  addReason(facts, "ORPHAN_RISK");
}

function cappedCapture(stream, maximum, latch, overflow) {
  const state = {
    activated: false, chunks: [], closeEvent: null, destroyError: false, endEvent: null,
    error: false, errorEvent: null, observed: 0, overflow: null, retained: 0,
    role: "capture", settled: false, settledAt: null,
  };
  let settle;
  state.done = new Promise((resolvePromise) => { settle = resolvePromise; });
  const markError = () => {
    const observedAt = monotonicNow();
    state.error = true;
    if (state.errorEvent === null) state.errorEvent = { observedAt };
    latch.wake();
  };
  stream.on("data", (chunk) => {
    const bytes = Buffer.from(chunk);
    state.observed += bytes.length;
    const available = Math.max(0, maximum - state.retained);
    if (available > 0) {
      const retained = bytes.subarray(0, available);
      state.chunks.push(retained);
      state.retained += retained.length;
    }
    if (state.observed > maximum && state.overflow === null) {
      state.overflow = { observedAt: monotonicNow() };
      overflow(state.overflow);
      latch.wake();
    }
  });
  stream.on("error", markError);
  stream.once("end", () => {
    if (state.endEvent === null) state.endEvent = { observedAt: monotonicNow() };
    latch.wake();
  });
  stream.once("close", () => {
    if (state.settled) return;
    const observedAt = monotonicNow();
    state.closeEvent = { observedAt };
    if (state.endEvent === null) markError();
    state.settled = true;
    state.settledAt = observedAt;
    settle();
    latch.wake();
  });
  state.destroy = () => {
    try { stream.destroy(); } catch { state.destroyError = true; latch.wake(); }
  };
  state.activate = () => {
    if (state.activated) return false;
    state.activated = true;
    latch.wake();
    return true;
  };
  state.bytes = () => Buffer.concat(state.chunks, state.retained);
  return state;
}

function timeoutSignal(milliseconds, value) {
  let timer;
  const promise = new Promise((resolvePromise) => {
    timer = setTimeout(() => resolvePromise(value), milliseconds);
  });
  return { promise, cancel: () => clearTimeout(timer) };
}

function beginStdinHandoff(stream, bytes, latch) {
  const state = {
    activated: false, closeEvent: null, complete: false, destroyError: false, failure: null,
    finishEvent: null, preactivationError: null, present: true, role: "stdin",
    settled: false, settledAt: null,
  };
  let settle;
  state.done = new Promise((resolvePromise) => { settle = resolvePromise; });
  const fail = (destroy = true, observedAt = monotonicNow()) => {
    if (state.failure === null) state.failure = { observedAt, reason: "TRANSPORT_FAILED" };
    latch.wake();
    if (destroy) {
      try { stream.destroy(); } catch { /* finalizer will retain the failed closure proof */ }
    }
  };
  stream.on("error", () => {
    const observedAt = monotonicNow();
    if (!state.activated) {
      if (state.preactivationError === null) state.preactivationError = { observedAt };
      latch.wake();
      return;
    }
    fail(true, observedAt);
  });
  stream.once("finish", () => {
    const observedAt = monotonicNow();
    state.finishEvent = { observedAt };
    if (!state.activated) {
      latch.wake();
      return;
    }
    if (state.failure === null && stream.writableFinished === true && stream.writableLength === 0) {
      state.complete = true;
      latch.wake();
      return;
    }
    fail(true, observedAt);
  });
  stream.once("close", () => {
    const observedAt = monotonicNow();
    state.closeEvent = { observedAt };
    if (state.activated && !state.complete && state.failure === null) fail(false, observedAt);
    if (!state.settled) {
      state.settled = true;
      state.settledAt = observedAt;
      settle();
    }
    latch.wake();
  });
  state.destroy = () => {
    try { stream.destroy(); } catch { state.destroyError = true; latch.wake(); }
  };
  state.activate = () => {
    if (state.activated) return false;
    state.activated = true;
    if (state.settled || state.preactivationError !== null || state.finishEvent !== null || state.closeEvent !== null) {
      fail(false);
      return false;
    }
    try {
      stream.end(bytes);
    } catch {
      fail();
      return false;
    }
    return true;
  };
  return state;
}

function pollCLIPhase(state, now, deadline) {
  if (observedBefore(state.outputOverflow.event, deadline)) return { kind: "overflow" };
  if (now >= deadline) return { kind: "timeout" };
  if (state.stdin !== null && observedBefore(state.stdin.failure, deadline)) return { kind: "stdin-failure" };
  if (observedBefore(state.owner.lifecycleError, deadline)) return { kind: "internal" };
  if (observedBefore(state.owner.spawnError, deadline)) return { kind: "error" };
  if (observedBefore(state.owner.exit, deadline)) {
    if (state.stdin !== null && !state.stdin.complete) return null;
    return state.owner.exit;
  }
  return null;
}

function pollReadinessPhase(state, now, deadline) {
  if (observedBefore(state.outputOverflow.event, deadline)) return { kind: "overflow" };
  if (now >= deadline) return { kind: "timeout" };
  if (observedBefore(state.owner.spawnError, deadline)) return { kind: "error" };
  if (observedBefore(state.owner.lifecycleError, deadline)) return { kind: "internal" };
  if (observedBefore(state.owner.exit, deadline)) return { kind: "exit" };
  if (observedBefore(state.readiness.errorEvent, deadline)) return { kind: "failure" };
  if (observedBefore(state.readiness.decisionResult, deadline)) return state.readiness.decisionResult;
  return null;
}

function recordFinalOutputOverflow(facts, event, behavioralDeadline, observationComplete) {
  if (event !== null && behavioralDeadline !== null &&
      (observedBefore(event, behavioralDeadline) || observationComplete)) addReason(facts, "OUTPUT_LIMIT");
}

function pollHTTPProbePhase(state, now, deadline) {
  if (observedBefore(state.outputOverflow.event, deadline) || observedBefore(state.socket.overflow, deadline)) {
    return { kind: "overflow" };
  }
  if (now >= deadline) return { kind: "timeout" };
  if (observedBefore(state.owner.lifecycleError, deadline)) return { kind: "internal" };
  if (observedBefore(state.owner.spawnError, deadline) || observedBefore(state.socket.transportFailure, deadline)) {
    return { kind: "transport" };
  }
  if (observedBefore(state.owner.exit, deadline) &&
      (state.owner.exit.signal !== null || state.owner.exit.code !== 0)) return { kind: "transport" };
  if (observedBefore(state.socket.success, deadline)) return { kind: "success", value: state.socket.bytes() };
  return null;
}

function resourceSettledBefore(resource, deadline) {
  return resource.settled && typeof resource.settledAt === "bigint" && resource.settledAt < deadline;
}

function childCompletedBefore(owner, deadline) {
  const validPID = Number.isInteger(owner.spawned?.pid) && owner.spawned.pid > 0;
  const childEvent = validPID ? owner.exit : owner.spawnError;
  return observedBefore(childEvent, deadline);
}

function finalJoinComplete(owner, resources, deadline = null) {
  if (deadline === null) {
    const validPID = Number.isInteger(owner.spawned?.pid) && owner.spawned.pid > 0;
    return (validPID ? owner.exit !== null : owner.spawnError !== null) &&
      resources.every((resource) => resource.settled);
  }
  return childCompletedBefore(owner, deadline) && resources.every((resource) => resourceSettledBefore(resource, deadline));
}

function refreshNaturalTerminal(owner, deadline = null) {
  return Number.isInteger(owner.spawned?.pid) && owner.spawned.pid > 0 && owner.exit !== null &&
    owner.exit.signalCountAtObservation === 0 &&
    (deadline === null || observedBefore(owner.exit, deadline)) ? owner.exit : null;
}

function recordResourceFailure(facts, resource, unsettled = false) {
  if (resource.destroyError === true) facts.internal_failure = true;
  if (resource.role === "capture" && resource.activated && (resource.error || unsettled)) {
    addReason(facts, "CAPTURE_FAILED");
  }
  if (resource.role === "readiness" && resource.activated && (resource.error || unsettled)) {
    addReason(facts, "READINESS_FAILED");
  }
  if (resource.role === "stdin" && (resource.failure !== null || resource.activated && unsettled)) {
    addReason(facts, resource.failure?.reason ?? "TRANSPORT_FAILED");
  }
  if (resource.role === "socket" && (resource.transportFailure !== null || unsettled)) {
    addReason(facts, "TRANSPORT_FAILED");
  }
}

async function finalizeChild(owner, resources, budget, facts, allowNaturalGrace, latch) {
  const deadline = monotonicDeadline(budget);
  let naturalTerminal = refreshNaturalTerminal(owner, deadline);
  if (owner.lifecycleError !== null) facts.internal_failure = true;
  try {
    if (owner.spawned === null && owner.spawnError === null && owner.exit === null) {
      await awaitOwnerDecision(deadline, latch, (now) => {
        if (observedBefore(owner.spawned, deadline) || observedBefore(owner.spawnError, deadline) ||
            observedBefore(owner.exit, deadline)) return { kind: "state" };
        if (now >= deadline) return { kind: "deadline" };
        return null;
      });
    }
    if (owner.exit === null && allowNaturalGrace) {
      const naturalDeadline = earlierDeadline(deadline, monotonicDeadline(budget / 4));
      await awaitOwnerDecision(naturalDeadline, latch, (now) => {
        if (observedBefore(owner.exit, naturalDeadline) || observedBefore(owner.spawnError, naturalDeadline)) {
          return { kind: "state" };
        }
        if (now >= naturalDeadline) return { kind: "deadline" };
        return null;
      });
      naturalTerminal = refreshNaturalTerminal(owner, deadline);
    }

    const pid = owner.spawned?.pid;
    if (!Number.isInteger(pid) || pid <= 0) {
      if (owner.spawnError === null) addUncertainTeardown(facts);
      for (const resource of resources) if (!resource.settled) resource.destroy();
    } else {
      let present = false;
      if (monotonicNow() >= deadline) {
        addUncertainTeardown(facts);
      } else {
        try {
          const probe = processGroupProbe(pid);
          if (monotonicNow() >= deadline || probe.error) throw new RuntimeRefusal("ORPHAN_RISK");
          present = probe.present;
        } catch (error) {
          if (!(error instanceof RuntimeRefusal)) facts.internal_failure = true;
          addUncertainTeardown(facts);
        }
      }
      if (present) {
        naturalTerminal = refreshNaturalTerminal(owner, deadline) ?? naturalTerminal;
        const term = monotonicNow() < deadline ? owner.signal("SIGTERM") : "late";
        if (term === "late") addUncertainTeardown(facts);
        if (term === "error") addUncertainTeardown(facts);
        if (term === "sent") {
          const graceDeadline = earlierDeadline(deadline, monotonicDeadline(budget / 3));
          try {
            const absence = await waitForGroupAbsence(pid, graceDeadline);
            if (!absence.absent) {
              if (monotonicNow() >= deadline) {
                addUncertainTeardown(facts);
              } else {
                const probe = processGroupProbe(pid);
                if (monotonicNow() >= deadline || probe.error) {
                  addUncertainTeardown(facts);
                } else if (probe.present) {
                  const killed = monotonicNow() < deadline ? owner.signal("SIGKILL") : "late";
                  if (killed === "late" || killed === "error") addUncertainTeardown(facts);
                }
              }
            }
          } catch (error) {
            if (!(error instanceof RuntimeRefusal)) facts.internal_failure = true;
            addUncertainTeardown(facts);
          }
        }
      }
    }

    const joined = await awaitOwnerDecision(deadline, latch, (now) => {
      if (finalJoinComplete(owner, resources, deadline)) return { joined: true };
      if (now >= deadline) return { joined: false };
      return null;
    });
    if (!joined.joined) {
      if (!childCompletedBefore(owner, deadline)) addUncertainTeardown(facts);
      for (const resource of resources) {
        if (!resourceSettledBefore(resource, deadline)) {
          recordResourceFailure(facts, resource, true);
          addUncertainTeardown(facts);
          if (!resource.settled) resource.destroy();
        }
      }
    }
    for (const resource of resources) recordResourceFailure(facts, resource);

    if (Number.isInteger(pid) && pid > 0) {
      try {
        const absence = await waitForGroupAbsence(pid, deadline);
        if (!absence.absent) {
          if (absence.probeError) addUncertainTeardown(facts);
          else addReason(facts, "ORPHAN_RISK");
        }
      } catch (error) {
        if (!(error instanceof RuntimeRefusal)) facts.internal_failure = true;
        addUncertainTeardown(facts);
      }
    }
  } catch {
    facts.internal_failure = true;
    addUncertainTeardown(facts);
    for (const resource of resources) if (!resource.settled) resource.destroy();
  }
  if (owner.exit === null && Number.isInteger(owner.spawned?.pid) && owner.spawned.pid > 0) {
    addUncertainTeardown(facts);
    for (const resource of resources) if (!resource.settled) resource.destroy();
    try {
      owner.child.unref();
    } catch {
      facts.internal_failure = true;
      addUncertainTeardown(facts);
    }
  }
  for (const resource of resources) recordResourceFailure(facts, resource);
  naturalTerminal = refreshNaturalTerminal(owner, deadline) ?? naturalTerminal;
  if (owner.lifecycleError !== null) facts.internal_failure = true;
  return { naturalTerminal, terminal: owner.exit };
}

function signalText(signal) {
  const values = new Map([
    ["SIGHUP", "hangup"], ["SIGINT", "interrupt"], ["SIGQUIT", "quit"], ["SIGILL", "illegal instruction"],
    ["SIGTRAP", "trace/BPT trap"], ["SIGABRT", "abort trap"], ["SIGEMT", "EMT trap"],
    ["SIGFPE", "floating point exception"], ["SIGKILL", "killed"], ["SIGBUS", "bus error"],
    ["SIGSEGV", "segmentation fault"], ["SIGSYS", "bad system call"], ["SIGPIPE", "broken pipe"],
    ["SIGALRM", "alarm clock"], ["SIGTERM", "terminated"], ["SIGURG", "urgent I/O condition"],
    ["SIGSTOP", "suspended (signal)"], ["SIGTSTP", "suspended"], ["SIGCONT", "continued"],
    ["SIGCHLD", "child exited"], ["SIGTTIN", "stopped (tty input)"], ["SIGTTOU", "stopped (tty output)"],
    ["SIGIO", "I/O possible"], ["SIGXCPU", "cputime limit exceeded"], ["SIGXFSZ", "filesize limit exceeded"],
    ["SIGVTALRM", "virtual timer expired"], ["SIGPROF", "profiling timer expired"],
    ["SIGWINCH", "window size changes"], ["SIGINFO", "information request"],
    ["SIGUSR1", "user defined signal 1"], ["SIGUSR2", "user defined signal 2"],
  ]);
  return values.get(signal) ?? "";
}

function portableFailure(reason) {
  if (reason === "CONTRACT_DATA_INVALID") refuse(reason);
  throw new RuntimeRefusal(reason);
}

function portableCanonicalBytes(value, failure) {
  try {
    return Buffer.from(canonicalText(value), "utf8");
  } catch {
    portableFailure(failure);
  }
}

function portableBase64Bytes(value, failure) {
  try {
    return strictBase64(value);
  } catch {
    portableFailure(failure);
  }
}

function portableValueMetrics(value, failure) {
  if (value === null || typeof value !== "object" || Array.isArray(value) || typeof value.tag !== "string") {
    portableFailure(failure);
  }
  let retained = Buffer.byteLength(value.tag, "utf8");
  let encoded = 0;
  switch (value.tag) {
    case "MISSING":
    case "NULL":
      if (!exactKeys(value, ["tag"])) portableFailure(failure);
      break;
    case "BOOLEAN":
      if (!exactKeys(value, ["tag", "value"]) || typeof value.value !== "boolean") portableFailure(failure);
      encoded = 5;
      break;
    case "INTEGER":
      if (!exactKeys(value, ["canonical", "tag"]) || typeof value.canonical !== "string" ||
          Buffer.byteLength(value.canonical, "utf8") > 32 || !/^-?(0|[1-9][0-9]*)$/.test(value.canonical) ||
          value.canonical === "-0" || !Number.isSafeInteger(Number(value.canonical)) ||
          String(Number(value.canonical)) !== value.canonical) portableFailure(failure);
      retained += Buffer.byteLength(value.canonical, "utf8");
      encoded = Buffer.byteLength(value.canonical, "utf8") + 2;
      break;
    case "STRING": {
      if (!exactKeys(value, ["tag", "value"]) || typeof value.value !== "string") portableFailure(failure);
      const length = Buffer.byteLength(value.value, "utf8");
      if (length > MAX_PORTABLE_VALUE_BYTES) portableFailure(failure);
      retained += length;
      encoded = portableCanonicalBytes(value.value, failure).length;
      break;
    }
    case "BYTES": {
      if (!exactKeys(value, ["base64", "tag"]) || typeof value.base64 !== "string") portableFailure(failure);
      const exact = portableBase64Bytes(value.base64, failure);
      if (exact.length > MAX_PORTABLE_VALUE_BYTES) portableFailure(failure);
      retained += exact.length;
      encoded = value.base64.length + 2;
      break;
    }
    case "ORDERED_STRING_LIST": {
      if (!exactKeys(value, ["tag", "values"]) || !Array.isArray(value.values) ||
          value.values.length > MAX_PORTABLE_LIST_MEMBERS) portableFailure(failure);
      let aggregate = 0;
      for (const member of value.values) {
        if (typeof member !== "string") portableFailure(failure);
        const length = Buffer.byteLength(member, "utf8");
        if (length > MAX_PORTABLE_VALUE_BYTES) portableFailure(failure);
        aggregate += length;
        if (aggregate > MAX_PORTABLE_VALUE_BYTES) portableFailure(failure);
      }
      const canonical = portableCanonicalBytes(value.values, failure);
      if (canonical.length > MAX_PORTABLE_VALUE_BYTES) portableFailure(failure);
      retained += canonical.length;
      encoded = 4 * Math.ceil(canonical.length / 3) + 2;
      break;
    }
    case "CANONICAL_JSON": {
      if (!exactKeys(value, ["canonical_base64", "tag"]) || typeof value.canonical_base64 !== "string") {
        portableFailure(failure);
      }
      const exact = portableBase64Bytes(value.canonical_base64, failure);
      if (exact.length === 0 || exact.length > MAX_PORTABLE_VALUE_BYTES) portableFailure(failure);
      try {
        parseCanonicalJSON(exact);
      } catch {
        portableFailure(failure);
      }
      retained += exact.length;
      encoded = value.canonical_base64.length + 2;
      break;
    }
    default:
      portableFailure(failure);
  }
  return { encoded, retained };
}

function validatePortableTuple(values, failure) {
  if (!Array.isArray(values) || values.length === 0 || values.length > MAX_PORTABLE_TUPLE_FIELDS) {
    portableFailure(failure);
  }
  let retained = 0;
  let encoded = 0;
  for (const value of values) {
    const metrics = portableValueMetrics(value, failure);
    retained += metrics.retained;
    encoded += PORTABLE_COMPATIBILITY_FIELD_OVERHEAD_BYTES + metrics.encoded;
    if (retained > MAX_PORTABLE_TUPLE_RETAINED_BYTES || encoded > MAX_PORTABLE_TUPLE_ENCODED_BYTES) {
      portableFailure(failure);
    }
  }
}

function checkedPortable(value) {
  portableValueMetrics(value, "PROJECTION_FAILED");
  return value;
}

function portableMissing() { return checkedPortable(ownObject([["tag", "MISSING"]])); }
function portableString(value) { return checkedPortable(ownObject([["tag", "STRING"], ["value", value]])); }
function portableInteger(value) { return checkedPortable(ownObject([["canonical", String(value)], ["tag", "INTEGER"]])); }
function portableBytes(value) {
  if (!Buffer.isBuffer(value)) throw new RuntimeRefusal("PROJECTION_FAILED");
  return checkedPortable(ownObject([["base64", value.toString("base64")], ["tag", "BYTES"]]));
}
function portableStrings(values) {
  if (!Array.isArray(values)) throw new RuntimeRefusal("PROJECTION_FAILED");
  return checkedPortable(ownObject([["tag", "ORDERED_STRING_LIST"], ["values", [...values]]]));
}
function portableJSON(value) {
  const exact = portableCanonicalBytes(value, "PROJECTION_FAILED");
  return checkedPortable(ownObject([["canonical_base64", exact.toString("base64")], ["tag", "CANONICAL_JSON"]]));
}

function projectCLI(contract, completion, stdout, stderr) {
  const values = new Map();
  let parsedStdout = null;
  const profile = contract.profileFields.map((field) => field.field_id);
  const included = new Set(profile);
  if (included.has("cli.stderr.text") && !validUTF8(stderr)) throw new RuntimeRefusal("PROJECTION_FAILED");
  if (included.has("cli.stdout.json.mode") || included.has("cli.stdout.json.source")) {
    try {
      parsedStdout = new StrictJSONParser(stdout).parse();
    } catch {
      throw new RuntimeRefusal("PROJECTION_FAILED");
    }
    if (parsedStdout === null || typeof parsedStdout !== "object" || Array.isArray(parsedStdout)) throw new RuntimeRefusal("PROJECTION_FAILED");
  }
  for (const field of profile) {
    switch (field) {
      case "cli.completion.kind": values.set(field, portableString(completion.kind)); break;
      case "cli.exit.code": values.set(field, completion.kind === "EXITED" ? portableInteger(completion.code) : portableMissing()); break;
      case "cli.exit.signal": values.set(field, completion.kind === "SIGNALED" ? portableString(completion.signal) : portableMissing()); break;
      case "cli.stdout.bytes": values.set(field, portableBytes(stdout)); break;
      case "cli.stderr.text": values.set(field, portableString(stderr.toString("utf8"))); break;
      case "cli.stdout.json.mode":
      case "cli.stdout.json.source": {
        const name = field.endsWith("mode") ? "mode" : "source";
        if (!Object.hasOwn(parsedStdout, name)) values.set(field, portableMissing());
        else if (typeof parsedStdout[name] !== "string") throw new RuntimeRefusal("PROJECTION_FAILED");
        else values.set(field, portableString(parsedStdout[name]));
        break;
      }
      default: throw new RuntimeRefusal("PROJECTION_FAILED");
    }
  }
  validatePortableTuple(profile.map((field) => values.get(field)), "PROJECTION_FAILED");
  return observedTuple(contract.predicate, values);
}

function parseReadyFrame(frame, eofObserved = true) {
  if (!Buffer.isBuffer(frame) || !eofObserved || frame.length > 32 || !frame.toString("ascii").startsWith("COUNTERSHAPE_READY_V1 ")) {
    throw new RuntimeRefusal("READINESS_FAILED");
  }
  const match = /^COUNTERSHAPE_READY_V1 ([1-9][0-9]{0,4})\n$/.exec(frame.toString("ascii"));
  if (!match || !Buffer.from(match[0], "ascii").equals(frame)) throw new RuntimeRefusal("READINESS_FAILED");
  const port = Number(match[1]);
  if (port < 1 || port > 65535) throw new RuntimeRefusal("READINESS_FAILED");
  return port;
}

function percentEncode(value) {
  const hexadecimal = "0123456789ABCDEF";
  let result = "";
  for (const byte of Buffer.from(value, "utf8")) {
    if (byte >= 0x41 && byte <= 0x5a || byte >= 0x61 && byte <= 0x7a || byte >= 0x30 && byte <= 0x39 || [0x2d, 0x2e, 0x5f, 0x7e].includes(byte)) {
      result += String.fromCharCode(byte);
    } else result += `%${hexadecimal[byte >> 4]}${hexadecimal[byte & 15]}`;
  }
  return result;
}

function requestBytes(runtime, port) {
  let target = runtime.path;
  if (runtime.query.length > 0) {
    target += "?" + runtime.query.map((entry) => `${percentEncode(entry.name)}${entry.presence === "PRESENT" ? `=${percentEncode(entry.value)}` : ""}`).join("&");
  }
  const lines = [`${runtime.method} ${target} HTTP/1.1`, `host: 127.0.0.1:${port}`, "connection: close"];
  for (const header of runtime.headers) lines.push(`${header.name}: ${header.value}`);
  if (runtime.bodyPresent) lines.push(`content-length: ${runtime.body.length}`);
  return Buffer.concat([Buffer.from(`${lines.join("\r\n")}\r\n\r\n`, "ascii"), runtime.body]);
}

function beginRawHTTPExchange(runtime, port, latch) {
  const maximum = runtime.capture.status_line_bytes + 2 + runtime.capture.header_bytes + 4 + runtime.capture.body_bytes;
  const state = {
    chunks: [], closeEvent: null, destroyError: false, destroyed: false, localEndRequested: false,
    observed: 0, overflow: null, requestFinished: null, responseEOF: null, role: "socket", settled: false,
    settledAt: null, socket: null, success: null, transportFailure: null,
  };
  state.bytes = () => Buffer.concat(state.chunks, state.observed);
  const transportFailure = (observedAt = monotonicNow()) => {
    if (state.transportFailure === null) state.transportFailure = { observedAt };
    latch.wake();
  };
  state.destroy = () => {
    if (state.destroyed) return;
    state.destroyed = true;
    try { state.socket?.destroy(); } catch { state.destroyError = true; transportFailure(); }
  };
  let socket;
  try {
    socket = createConnection({ host: "127.0.0.1", port, family: 4 });
    state.socket = socket;
  } catch {
    transportFailure();
    state.settled = true;
    state.settledAt = state.transportFailure.observedAt;
    return state;
  }
  socket.once("connect", () => {
    try {
      state.localEndRequested = true;
      socket.end(requestBytes(runtime, port));
    } catch {
      transportFailure();
      state.destroy();
    }
  });
  socket.once("finish", () => {
    if (socket.writableFinished !== true || socket.writableLength !== 0) {
      transportFailure();
      state.destroy();
      return;
    }
    state.requestFinished = { observedAt: monotonicNow() };
    latch.wake();
  });
  socket.on("data", (chunk) => {
    state.observed += chunk.length;
    if (state.observed > maximum) {
      if (state.overflow === null) state.overflow = { observedAt: monotonicNow() };
      latch.wake();
      state.destroy();
      return;
    }
    state.chunks.push(Buffer.from(chunk));
  });
  socket.once("end", () => {
    state.responseEOF = { observedAt: monotonicNow() };
    latch.wake();
  });
  socket.once("close", () => {
    const observedAt = monotonicNow();
    state.closeEvent = { observedAt };
    state.settled = true;
    state.settledAt = observedAt;
    if (state.transportFailure === null && state.overflow === null && state.localEndRequested &&
        state.requestFinished !== null && state.responseEOF !== null) {
      state.success = { observedAt: state.closeEvent.observedAt };
      latch.wake();
    } else if (state.overflow === null) {
      transportFailure(observedAt);
    } else {
      latch.wake();
    }
  });
  socket.once("error", () => {
    transportFailure();
    state.destroy();
  });
  return state;
}

async function rawHTTPExchange(runtime, port, deadline, lifecycle) {
  const socket = beginRawHTTPExchange(runtime, port, lifecycle.latch);
  lifecycle.socket = socket;
  if (lifecycle.outputOverflow.event !== null) socket.destroy();
  const decision = await awaitOwnerDecision(
    deadline,
    lifecycle.latch,
    (now) => pollHTTPProbePhase({
      outputOverflow: lifecycle.outputOverflow, owner: lifecycle.owner, socket,
    }, now, deadline),
  );
  if (decision.kind === "success") return decision.value;
  socket.destroy();
  if (decision.kind === "overflow") throw new RuntimeRefusal("OUTPUT_LIMIT");
  if (decision.kind === "timeout") throw new RuntimeRefusal("TIMEOUT");
  if (decision.kind === "internal") throw new Error("child lifecycle invariant failed");
  throw new RuntimeRefusal("TRANSPORT_FAILED");
}

function parseHTTPResponse(raw, policy) {
  const maximum = policy.status_line_bytes + 2 + policy.header_bytes + 4 + policy.body_bytes;
  if (!Buffer.isBuffer(raw) || raw.length > maximum) throw new RuntimeRefusal("OUTPUT_LIMIT");
  const statusEnd = raw.indexOf(Buffer.from("\r\n"));
  if (statusEnd < 0) throw new RuntimeRefusal("RESPONSE_PARSE_FAILED");
  if (statusEnd > policy.status_line_bytes) throw new RuntimeRefusal("OUTPUT_LIMIT");
  const statusLine = raw.subarray(0, statusEnd);
  if (statusLine.length < 13 || !statusLine.subarray(0, 9).equals(Buffer.from("HTTP/1.1 ", "ascii")) || statusLine[12] !== 0x20) {
    throw new RuntimeRefusal("RESPONSE_PARSE_FAILED");
  }
  const statusBytes = statusLine.subarray(9, 12);
  if ([...statusBytes].some((byte) => byte < 0x30 || byte > 0x39)) throw new RuntimeRefusal("RESPONSE_PARSE_FAILED");
  const status = Number(statusBytes.toString("ascii"));
  if (status < 200 || status > 599) throw new RuntimeRefusal("RESPONSE_PARSE_FAILED");
  const reasonBytes = statusLine.subarray(13);
  if ([...reasonBytes].some((byte) => byte < 0x20 || byte > 0x7e)) throw new RuntimeRefusal("RESPONSE_PARSE_FAILED");
  const headerStart = statusEnd + 2;
  const relativeEnd = raw.subarray(headerStart).indexOf(Buffer.from("\r\n\r\n"));
  if (relativeEnd < 0) throw new RuntimeRefusal("RESPONSE_PARSE_FAILED");
  if (relativeEnd > policy.header_bytes) throw new RuntimeRefusal("OUTPUT_LIMIT");
  const headerEnd = headerStart + relativeEnd;
  const block = raw.subarray(headerStart, headerEnd);
  if (block.length === 0) throw new RuntimeRefusal("RESPONSE_PARSE_FAILED");
  const lines = block.toString("latin1").split("\r\n");
  if (lines.length > policy.header_count) throw new RuntimeRefusal("OUTPUT_LIMIT");
  const headers = [];
  let contentLength = null;
  for (const line of lines) {
    const colon = line.indexOf(":");
    if (line === "" || line.startsWith(" ") || line.startsWith("\t") || colon < 1 || line[colon + 1] !== " ") {
      throw new RuntimeRefusal("RESPONSE_PARSE_FAILED");
    }
    const name = line.slice(0, colon).toLowerCase();
    const value = line.slice(colon + 2);
    if (!/^[!#$%&'*+.^_`|~0-9a-z-]+$/.test(name) || !/^[\x20-\x7e]*$/.test(value) ||
        name === "transfer-encoding" || name === "content-encoding") throw new RuntimeRefusal("RESPONSE_PARSE_FAILED");
    if (name === "content-length") {
      if (contentLength !== null || !/^(?:0|[1-9][0-9]*)$/.test(value)) throw new RuntimeRefusal("RESPONSE_PARSE_FAILED");
      const parsed = BigInt(value);
      if (parsed > 9223372036854775807n) throw new RuntimeRefusal("RESPONSE_PARSE_FAILED");
      if (parsed > BigInt(policy.body_bytes)) throw new RuntimeRefusal("OUTPUT_LIMIT");
      contentLength = Number(parsed);
    }
    headers.push({ name, value });
  }
  const body = raw.subarray(headerEnd + 4);
  if (contentLength === null || body.length !== contentLength) throw new RuntimeRefusal("RESPONSE_PARSE_FAILED");
  return { status, reason: reasonBytes.toString("ascii"), headers, body, contentLength };
}

function projectHTTP(contract, response, scratchRoot) {
  let body;
  try {
    body = new StrictJSONParser(response.body).parse();
  } catch {
    throw new RuntimeRefusal("PROJECTION_FAILED");
  }
  if (body === null || typeof body !== "object" || Array.isArray(body) || typeof body.request_id !== "string" ||
      typeof body.scratch_root !== "string" || body.scratch_root !== scratchRoot || typeof body.kind !== "string" ||
      body.metadata === null || typeof body.metadata !== "object" || Array.isArray(body.metadata)) {
    throw new RuntimeRefusal("PROJECTION_FAILED");
  }
  const contentTypes = response.headers.filter((header) => header.name === "content-type").map((header) => header.value);
  const values = new Map([
    ["http.status", portableInteger(response.status)],
    ["http.header.content-type", contentTypes.length === 0 ? portableMissing() : portableStrings(contentTypes)],
    ["http.body.kind", portableString(body.kind)],
    ["http.body.metadata", portableJSON(body.metadata)],
  ]);
  validatePortableTuple(HTTP_FIELD_PROFILE.map((entry) => values.get(entry[0])), "PROJECTION_FAILED");
  return observedTuple(contract.predicate, values);
}

async function runCLI(contract, roots, environment, facts) {
  const latch = makeRotatingLatch();
  let child;
  try {
    child = spawn(process.execPath, contract.runtime.argv, {
      argv0: "node", cwd: roots.candidate, env: environment, shell: false, detached: true,
      stdio: [contract.runtime.stdinPresent ? "pipe" : "ignore", "pipe", "pipe"],
    });
  } catch {
    addReason(facts, "START_FAILED");
    return;
  }
  const owner = childOwner(child, latch);
  const resources = [];
  const outputOverflow = { event: null };
  const markOverflow = (event) => {
    if (outputOverflow.event === null) outputOverflow.event = event;
  };
  let stdout = null;
  let stderr = null;
  let stdin = null;
  let terminal = null;
  let finalized = null;
  let probeDeadline = null;
  try {
    if (child.stdout === null) facts.internal_failure = true;
    else {
      stdout = cappedCapture(child.stdout, contract.source.limits.stdout_bytes, latch, markOverflow);
      resources.push(stdout);
    }
    if (child.stderr === null) facts.internal_failure = true;
    else {
      stderr = cappedCapture(child.stderr, contract.source.limits.stderr_bytes, latch, markOverflow);
      resources.push(stderr);
    }
    if (contract.runtime.stdinPresent) {
      if (child.stdin === null) facts.internal_failure = true;
      else {
        stdin = beginStdinHandoff(child.stdin, contract.runtime.stdin, latch);
        resources.push(stdin);
      }
    }
    const spawned = await awaitSpawnConfirmation(owner, latch, contract.source.limits.probe_ms);
    if (!spawned) addReason(facts, "START_FAILED");
    else {
      stdout?.activate();
      stderr?.activate();
      stdin?.activate();
    }
    if (spawned && facts.ineligible_reasons.length === 0 && !facts.internal_failure) {
      probeDeadline = monotonicDeadlineFrom(owner.spawned.observedAt, contract.source.limits.probe_ms);
      terminal = await awaitOwnerDecision(
        probeDeadline,
        latch,
        (now) => pollCLIPhase({ owner, outputOverflow, stdin }, now, probeDeadline),
      );
      if (terminal.kind === "overflow") addReason(facts, "OUTPUT_LIMIT");
      else if (terminal.kind === "stdin-failure") addReason(facts, "TRANSPORT_FAILED");
      else if (terminal.kind === "timeout") addReason(facts, "TIMEOUT");
      else if (terminal.kind === "error") addReason(facts, "START_FAILED");
      else if (terminal.kind === "internal") facts.internal_failure = true;
    }
  } catch {
    facts.internal_failure = true;
  } finally {
    finalized = await finalizeChild(
      owner, resources, contract.source.limits.teardown_ms, facts, false, latch,
    );
  }
  const finalTerminal = finalized.terminal;
  if (stdin !== null && stdin.failure !== null) addReason(facts, stdin.failure.reason);
  recordFinalOutputOverflow(
    facts, outputOverflow.event, probeDeadline,
    terminal?.kind === "exit" && probeDeadline !== null && observedBefore(terminal, probeDeadline),
  );
  if (facts.ineligible_reasons.length !== 0 || facts.internal_failure || terminal?.kind !== "exit" ||
      finalTerminal?.kind !== "exit") return;
  let completion;
  if (Number.isInteger(terminal.code) && terminal.signal === null) completion = { kind: "EXITED", code: terminal.code, signal: "" };
  else if (terminal.code === null && typeof terminal.signal === "string" && signalText(terminal.signal) !== "") {
    completion = { kind: "SIGNALED", code: 0, signal: signalText(terminal.signal) };
  } else {
    facts.internal_failure = true;
    return;
  }
  try {
    const tuple = projectCLI(contract, completion, stdout.bytes(), stderr.bytes());
    facts.predicate_match = predicateMatches(contract.predicate, tuple);
  } catch (error) {
    if (error instanceof RuntimeRefusal) addReason(facts, error.reason);
    else facts.internal_failure = true;
  }
}

function readinessCapture(stream, maximum, latch) {
  const state = {
    activated: false, chunks: [], closeEvent: null, decisionResult: null, destroyError: false,
    endEvent: null, ended: false, error: false, errorEvent: null, observed: 0, role: "readiness",
    settled: false, settledAt: null,
  };
  let settleDone;
  state.done = new Promise((resolvePromise) => { settleDone = resolvePromise; });
  const decide = (value, observedAt = monotonicNow()) => {
    if (state.decisionResult !== null) return;
    state.decisionResult = { ...value, observedAt };
    latch.wake();
  };
  const markError = (observedAt = monotonicNow()) => {
    state.error = true;
    if (state.errorEvent === null) state.errorEvent = { observedAt };
    decide({ kind: "failure" }, observedAt);
    latch.wake();
  };
  const settle = () => {
    if (state.settled) return;
    const observedAt = monotonicNow();
    state.closeEvent = { observedAt };
    if (!state.ended) markError(observedAt);
    state.settled = true;
    state.settledAt = observedAt;
    settleDone();
    latch.wake();
  };
  stream.on("data", (chunk) => {
    const bytes = Buffer.from(chunk);
    state.observed += bytes.length;
    if (state.observed <= maximum) state.chunks.push(bytes);
    else decide({ kind: "failure" });
  });
  stream.on("error", () => markError());
  stream.once("end", () => {
    const observedAt = monotonicNow();
    state.ended = true;
    state.endEvent = { observedAt };
    if (state.error || state.observed > maximum) decide({ kind: "failure" }, observedAt);
    else decide({ frame: Buffer.concat(state.chunks, state.observed), kind: "frame" }, observedAt);
    latch.wake();
  });
  stream.once("close", settle);
  state.destroy = () => {
    try { stream.destroy(); } catch { state.destroyError = true; latch.wake(); }
  };
  state.activate = () => {
    if (state.activated) return false;
    state.activated = true;
    latch.wake();
    return true;
  };
  return state;
}

async function awaitHTTPReadiness(owner, readiness, outputOverflow, deadline, latch, facts) {
  const decision = await awaitOwnerDecision(
    deadline,
    latch,
    (now) => pollReadinessPhase({ owner, outputOverflow, readiness }, now, deadline),
  );
  if (decision.kind === "overflow") {
    addReason(facts, "OUTPUT_LIMIT");
    return undefined;
  }
  if (decision.kind === "timeout" || decision.kind === "exit" || decision.kind === "failure") {
    addReason(facts, "READINESS_FAILED");
    return undefined;
  }
  if (decision.kind === "error") {
    addReason(facts, "START_FAILED");
    return undefined;
  }
  if (decision.kind === "internal") {
    facts.internal_failure = true;
    return undefined;
  }
  try {
    return parseReadyFrame(decision.frame, true);
  } catch {
    addReason(facts, "READINESS_FAILED");
    return undefined;
  }
}

async function runHTTP(contract, roots, environment, facts) {
  const latch = makeRotatingLatch();
  let child;
  try {
    child = spawn(process.execPath, [contract.source.entrypoint], {
      argv0: "node", cwd: roots.candidate, env: environment, shell: false, detached: true,
      stdio: ["ignore", "pipe", "pipe", "pipe"],
    });
  } catch {
    addReason(facts, "START_FAILED");
    return;
  }
  const owner = childOwner(child, latch);
  const resources = [];
  const outputOverflow = { event: null };
  const lifecycle = { latch, outputOverflow, owner, socket: null };
  const markOverflow = (event) => {
    if (outputOverflow.event === null) outputOverflow.event = event;
    lifecycle.socket?.destroy();
  };
  let stdout = null;
  let stderr = null;
  let readiness = null;
  let readinessAccepted = false;
  let response = null;
  let finalized = null;
  let behavioralDeadline = null;
  try {
    if (child.stdout === null) facts.internal_failure = true;
    else {
      stdout = cappedCapture(child.stdout, contract.source.limits.stdout_bytes, latch, markOverflow);
      resources.push(stdout);
    }
    if (child.stderr === null) facts.internal_failure = true;
    else {
      stderr = cappedCapture(child.stderr, contract.source.limits.stderr_bytes, latch, markOverflow);
      resources.push(stderr);
    }
    const readinessStream = Array.isArray(child.stdio) ? child.stdio[3] : null;
    if (readinessStream === null || readinessStream === undefined) facts.internal_failure = true;
    else {
      readiness = readinessCapture(readinessStream, 32, latch);
      resources.push(readiness);
    }
    const spawned = await awaitSpawnConfirmation(owner, latch, contract.source.limits.readiness_ms);
    if (!spawned) addReason(facts, "START_FAILED");
    else {
      stdout?.activate();
      stderr?.activate();
      readiness?.activate();
    }
    if (spawned && facts.ineligible_reasons.length === 0 && !facts.internal_failure) {
      const readinessDeadline = monotonicDeadlineFrom(
        owner.spawned.observedAt, contract.source.limits.readiness_ms,
      );
      behavioralDeadline = readinessDeadline;
      const ready = await awaitHTTPReadiness(
        owner, readiness, outputOverflow, readinessDeadline, latch, facts,
      );
      readinessAccepted = ready !== undefined;
      if (ready !== undefined) {
        const probeDeadline = monotonicDeadline(contract.source.limits.probe_ms);
        behavioralDeadline = probeDeadline;
        try {
          const raw = await rawHTTPExchange(contract.runtime, ready, probeDeadline, lifecycle);
          response = parseHTTPResponse(raw, contract.runtime.capture);
        } catch (error) {
          if (error instanceof RuntimeRefusal) addReason(facts, error.reason);
          else facts.internal_failure = true;
        }
      }
    }
  } catch {
    facts.internal_failure = true;
  } finally {
    if (lifecycle.socket !== null) resources.push(lifecycle.socket);
    finalized = await finalizeChild(
      owner, resources, contract.source.limits.teardown_ms, facts, true, latch,
    );
  }
  const natural = finalized.naturalTerminal;
  if (readinessAccepted && natural?.kind === "exit" && (natural.signal !== null || natural.code !== 0)) {
    addReason(facts, "TRANSPORT_FAILED");
  }
  recordFinalOutputOverflow(facts, outputOverflow.event, behavioralDeadline, response !== null);
  if (response === null || facts.ineligible_reasons.length !== 0 || facts.internal_failure) return;
  try {
    const tuple = projectHTTP(contract, response, roots.state);
    facts.predicate_match = predicateMatches(contract.predicate, tuple);
  } catch (error) {
    if (error instanceof RuntimeRefusal) addReason(facts, error.reason);
    else facts.internal_failure = true;
  }
}

async function executeContract(contract, bundleRoot, targetRoot, facts) {
  if (process.platform !== "darwin" || process.arch !== "arm64") {
    addReason(facts, "ENVIRONMENT_INVALID");
    return;
  }
  let bundle;
  let target;
  let bundleIdentity;
  let targetIdentity;
  try {
    bundle = canonicalDirectory(bundleRoot);
    target = canonicalDirectory(targetRoot);
    bundleIdentity = rootIdentity(bundle);
    targetIdentity = rootIdentity(target);
  } catch (error) {
    addReason(facts, error.reason ?? "EXECUTION_ROOT_FAILED");
    return;
  }
  if (within(bundle, target) || within(target, bundle)) {
    addReason(facts, "EXECUTION_ROOT_FAILED");
    return;
  }
  let bundleWatcher;
  let targetWatcher;
  try {
    [bundleWatcher, targetWatcher] = startRootWatcherPair(bundle, target);
  } catch (error) {
    if (error instanceof RuntimeRefusal) addReason(facts, error.reason);
    else facts.internal_failure = true;
    return;
  }
  let sourceRecords;
  let execution = null;
  let roots = null;
  try {
    sourceRecords = scanInventory(target, contract.source.limits);
    if (!sourceRecords.some((record) => record.kind === "file" && record.path === contract.source.entrypoint)) {
      throw new RuntimeRefusal("SOURCE_INVENTORY_INVALID");
    }
    if (contract.source.adapter === "HTTP") {
      validateCombinedHTTPMaterializationBudgets(sourceRecords, contract.runtime.seeds, contract.source.limits);
    }
    execution = allocateExecutionRoot(bundle, target);
    roots = createSubroots(execution.root);
    copyInventory(sourceRecords, roots.candidate, contract.source.limits);
    materializeOverlay(contract.source.adapter === "CLI" ? contract.runtime.fixtures : contract.runtime.seeds, roots.fixture);
    const environment = buildEnvironment(contract, roots);
    if (contract.source.adapter === "CLI") await runCLI(contract, roots, environment, facts);
    else await runHTTP(contract, roots, environment, facts);
  } catch (error) {
    if (error instanceof RuntimeRefusal) addReason(facts, error.reason);
    else facts.internal_failure = true;
  } finally {
    if (sourceRecords !== undefined) {
      try {
        const sourceAfter = scanInventory(target, contract.source.limits);
        if (inventoryIdentity(sourceAfter) !== inventoryIdentity(sourceRecords)) addReason(facts, "SOURCE_INVENTORY_INVALID");
      } catch {
        addReason(facts, "SOURCE_INVENTORY_INVALID");
      }
    }
    if (execution !== null) {
      try {
        if (realpathSync(execution.root) !== execution.root || dirname(execution.root) !== execution.parent) throw new Error("cleanup topology");
        rmSync(execution.root, { recursive: true, force: false });
      } catch (error) {
        if (["EACCES", "EPERM", "ENOTEMPTY", "EBUSY"].includes(error?.code)) addReason(facts, "CLEANUP_FAILED");
        else facts.internal_failure = true;
      }
    }
    await eventLoopTurn();
    await eventLoopTurn();
    try {
      if (rootIdentity(bundle) !== bundleIdentity) facts.tamper = true;
    } catch (error) {
      if (error instanceof RuntimeRefusal) facts.tamper = true;
      else facts.internal_failure = true;
    }
    try {
      if (rootIdentity(target) !== targetIdentity) addReason(facts, "SOURCE_INVENTORY_INVALID");
    } catch (error) {
      if (error instanceof RuntimeRefusal) addReason(facts, "SOURCE_INVENTORY_INVALID");
      else facts.internal_failure = true;
    }
    if (bundleWatcher.events.length > 0) facts.tamper = true;
    if (targetWatcher.events.length > 0) addReason(facts, "SOURCE_INVENTORY_INVALID");
    if (bundleWatcher.errors.length > 0 || targetWatcher.errors.length > 0) addReason(facts, "ENVIRONMENT_INVALID");
    if (closeRootWatchers(bundleWatcher, targetWatcher).length !== 0) facts.internal_failure = true;
  }
}

export async function runContract({ bundleRoot, targetRoot, decisionBytes, fixtureBytes }) {
  let contract;
  try {
    if (typeof bundleRoot !== "string" || typeof targetRoot !== "string" || !Buffer.isBuffer(decisionBytes) || !Buffer.isBuffer(fixtureBytes)) {
      refuse("CONTRACT_DATA_INVALID");
    }
    contract = parseContractData(decisionBytes, fixtureBytes);
  } catch (error) {
    if (error instanceof Refusal && ["CONTRACT_DATA_INVALID", "INVALID_JSON", "NONCANONICAL_JSON", "INVALID_BASE64"].includes(error.code)) {
      return { outcome: "MALFORMED_CONTRACT", reason: "CONTRACT_DATA_INVALID" };
    }
    return { outcome: "HARNESS_FAILURE", reason: "INTERNAL_INVARIANT_FAILED" };
  }
  const facts = { ineligible_reasons: [], internal_failure: false, predicate_match: false, tamper: false };
  try {
    await executeContract(contract, bundleRoot, targetRoot, facts);
    return selectDirectResult(facts);
  } catch {
    facts.internal_failure = true;
    return selectDirectResult(facts);
  }
}
