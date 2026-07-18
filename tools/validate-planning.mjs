#!/usr/bin/env node

import fs from "node:fs";
import crypto from "node:crypto";
import { spawnSync } from "node:child_process";
import os from "node:os";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const REPOSITORY_ROOT = path.resolve(SCRIPT_DIR, "..");
const SCHEMA_ID_BASE = "https://countershape.dev/spec/schema/v1/";

const OBJECTS = [
  ["SourceSpec", "source-spec", "source-spec"],
  ["WorldPlan", "world-plan", "world-plan"],
  ["CandidateExecutionBinding", "candidate-execution-binding", "candidate-execution-binding"],
  ["WorldInstance", "world-instance", "world-instance"],
  ["InstanceMeasurements", "instance-measurements", "instance-measurements"],
  ["ComparisonEnvelope", "comparison-envelope", "comparison-envelope"],
  ["ComparisonAdmission", "comparison-admission", "comparison-assessment"],
  ["RejectedComparison", "rejected-comparison", "comparison-assessment"],
  ["CapturedObservation", "captured-observation", "captured-observation"],
  ["StableBatch", "stable-batch", "stable-batch"],
  ["CandidateOutcomeMap", "outcome-map", "outcome-map"],
  ["ReductionRun", "reduction-run", "reduction-run"],
  ["ReductionTranscript", "reduction-transcript", "reduction-transcript"],
  ["CompletedSweepDraft", "completed-sweep-draft", "completed-sweep-draft"],
  ["ReductionGrade", "reduction-grade", "reduction-grade"],
  ["Choicepoint", "choicepoint", "choicepoint"],
  ["DecisionRecord", "decision-record", "decision-record"],
  ["ContractBundle", "contract-bundle", "contract-bundle"],
  ["ContractExecutionTarget", "contract-execution-target", "contract-execution-target"],
  ["FinalizedContractRun", "finalized-contract-run", "finalized-contract-run"],
  ["ContractExecution", "contract-execution", "contract-execution"],
];

// Independent shrinkage tripwire. Required files and mappings are derived
// from OBJECTS for normal operation, while this frozen contract prevents one
// accidental tuple deletion from shrinking every derived obligation together.
const EXACT_OBJECT_CONTRACT = Object.freeze([
  "SourceSpec|source-spec|source-spec",
  "WorldPlan|world-plan|world-plan",
  "CandidateExecutionBinding|candidate-execution-binding|candidate-execution-binding",
  "WorldInstance|world-instance|world-instance",
  "InstanceMeasurements|instance-measurements|instance-measurements",
  "ComparisonEnvelope|comparison-envelope|comparison-envelope",
  "ComparisonAdmission|comparison-admission|comparison-assessment",
  "RejectedComparison|rejected-comparison|comparison-assessment",
  "CapturedObservation|captured-observation|captured-observation",
  "StableBatch|stable-batch|stable-batch",
  "CandidateOutcomeMap|outcome-map|outcome-map",
  "ReductionRun|reduction-run|reduction-run",
  "ReductionTranscript|reduction-transcript|reduction-transcript",
  "CompletedSweepDraft|completed-sweep-draft|completed-sweep-draft",
  "ReductionGrade|reduction-grade|reduction-grade",
  "Choicepoint|choicepoint|choicepoint",
  "DecisionRecord|decision-record|decision-record",
  "ContractBundle|contract-bundle|contract-bundle",
  "ContractExecutionTarget|contract-execution-target|contract-execution-target",
  "FinalizedContractRun|finalized-contract-run|finalized-contract-run",
  "ContractExecution|contract-execution|contract-execution",
]);

const actualObjectContract = OBJECTS.map((parts) => parts.join("|"));
if (
  actualObjectContract.length !== EXACT_OBJECT_CONTRACT.length
  || actualObjectContract.some((entry, index) => entry !== EXACT_OBJECT_CONTRACT[index])
) {
  throw new Error("planning object contract drifted from the exact 21-object obligation");
}

const REQUIRED_SCHEMA_FILES = [
  "common.schema.json",
  ...new Set(OBJECTS.map(([, , schemaStem]) => `${schemaStem}.schema.json`)),
];

const REQUIRED_EXAMPLE_FILES = OBJECTS.map(([, exampleStem]) => `${exampleStem}.valid.json`);

const EXAMPLE_SCHEMA_MAPPINGS = new Map(
  OBJECTS.map(([objectName, exampleStem, schemaStem]) => [
    `${exampleStem}.valid.json`,
    { objectName, schemaFile: `${schemaStem}.schema.json` },
  ]),
);

const ENUM_GROUPS = new Map([
  ["batch classification", ["OBSERVED_STABLE", "UNSTABLE", "UNCOMPARABLE", "INCOMPLETE"]],
  ["reduction decision", ["PRESERVES", "CHANGES", "UNRESOLVED"]],
  ["reduction grade", ["UNCHANGED", "BEST_KNOWN", "ONE_MINIMAL_UNDER"]],
  ["decision action", ["ALLOW_OBSERVED", "CUSTOM_EXPECTATION", "REJECT_ALL", "DEFER", "REFINE"]],
  ["contract execution", ["CONFORMS", "CONTRADICTS", "INELIGIBLE_EXECUTION"]],
  ["network mode", ["HOST_ALLOWED"]],
]);

// These pointers identify the authority-bearing declaration for each enum
// agreement. Agreement is checked at the declaration itself, rather than by
// unioning every string constant in a file (where an unrelated extra constant
// could otherwise conceal a missing term).
const ENUM_GROUP_BINDINGS = new Map([
  ["batch classification", [
    { schema: "stable-batch.schema.json", pointer: "#/properties/classification/oneOf/0/properties/status" },
    { schema: "stable-batch.schema.json", pointer: "#/properties/classification/oneOf/1/properties/status" },
    { schema: "stable-batch.schema.json", pointer: "#/properties/classification/oneOf/2/properties/status" },
    { schema: "stable-batch.schema.json", pointer: "#/properties/classification/oneOf/3/properties/status" },
  ]],
  ["reduction decision", [
    { schema: "reduction-transcript.schema.json", pointer: "#/$defs/EvaluationEntry/properties/decision" },
  ]],
  ["reduction grade", [
    { schema: "reduction-grade.schema.json", pointer: "#/properties/status" },
  ]],
  ["decision action", [
    { schema: "decision-record.schema.json", pointer: "#/properties/action" },
  ]],
  ["contract execution", [
    { schema: "contract-execution.schema.json", pointer: "#/properties/result" },
  ]],
  ["network mode", [
    { schema: "world-plan.schema.json", pointer: "#/properties/network_mode" },
  ]],
]);

const FORBIDDEN_ENUM_VALUES = new Set([
  "STABLE",
  "DETERMINISTIC",
  "SAME_GROUPS",
  "GROUPS_PRESERVED",
  "SMALLEST",
  "WINNER",
  "BEST_BRANCH",
]);

const REQUIRED_VECTORS = new Map([
  ["duplicate-candidate-keys", "DUPLICATE_CANDIDATE_KEY"],
  ["ordinal-group-identity", "ORDINAL_GROUP_IDENTITY_FORBIDDEN"],
  ["control-as-outcome", "CONTROL_AS_OUTCOME"],
  ["unresolved-one-minimality", "UNRESOLVED_FINAL_SWEEP"],
  ["reused-confirmation-evidence", "REUSED_CONFIRMATION_EVIDENCE"],
  ["empty-fields", "EMPTY_SELECTED_FIELDS"],
  ["allow-many-cross-product", "ALLOW_MANY_CROSS_PRODUCT"],
  ["noncompilable-emission", "NONCOMPILABLE_ACTION"],
  ["symlink-materialization-refusal", "UNSUPPORTED_GIT_MODE"],
  ["copied-target-authority", "COPIED_TARGET_AUTHORITY"],
  ["target-run-mismatch", "TARGET_RUN_MISMATCH"],
]);

const POSITIVE_VECTORS = new Map([
  ["comparison-envelope-nonclaim", null],
]);

const CONTROLLING_MARKDOWN = [
  "docs/CONCEPT_BRIEF.md",
  "docs/PROMPT_PACK.md",
  "docs/ARCHITECTURE.md",
  "docs/THREAT_MODEL.md",
  "docs/CLAIM_VOCABULARY.md",
  "docs/STATE_MACHINES.md",
];

const ALLOW_BEGIN = "<!-- countershape-validator: allow-prohibited-terms begin -->";
const ALLOW_END = "<!-- countershape-validator: allow-prohibited-terms end -->";

const LEGACY_CLAIM_RULES = [
  // STABLE is a reserved all-caps classification upgrade. Ordinary prose such
  // as "stable identifier" or "stable semantic object" is not that claim.
  ["unqualified STABLE", /(?<!OBSERVED_)\bSTABLE\b/gu],
  ["compatible worlds", /\bcompatible worlds?\b/giu],
  ["smallest behavior", /\bsmallest behavior\b/giu],
  ["symlink support", /\b(?:symlink support|support(?:s|ed|ing)? symlinks?|symlinks? (?:are )?supported)\b/giu],
  ["safely shareable report", /\b(?:safely shareable|safe(?:ly)?[- ]shareable|shareable report|report is safe)\b/giu],
  ["execution-evidence reuse", /\b(?:execution[- ]evidence (?:cache|reuse)|reus(?:e|ed|es|ing) execution evidence)\b/giu],
  ["winner or best branch", /\b(?:best branch|(?:finds?|selects?|picks?|declares?|recommends?) (?:a |the )?winner|winner (?:branch|candidate|implementation|outcome|selector|emphasis))\b/giu],
  ["ordinal/group semantic authority", /\b(?:ordinal (?:cluster|group) ids?|group (?:ids?|identity))\b[^.!?\n]*(?:authorit|semantic identity|preservation predicate)/giu],
  ["compilable REJECT_ALL", /\bREJECT_ALL\b[^.!?\n]*(?:(?<!non)compilable|compile[sd]?|emit(?:s|ted)? (?:code|executable))/giu],
  ["boolean UNRESOLVED", /\bUNRESOLVED\b[^.!?\n]*(?:boolean|false)/giu],
];

const NEGATIVE_CONTEXT = /\b(?:no|not|never|cannot|can't|does not|doesn't|do not|don't|without|noncompilable|prohibit(?:ed|s)?|forbid(?:den|s)?|reject(?:ed|s|ing)?|refus(?:e|ed|es|ing|al)|remove(?:d|s|ing)?|unsupported|exclude(?:d|s)?|illegal|invalid|kill|stop|fail(?:s|ed|ure)?|non[- ]claim|unclaimed|legacy|earlier|replaced|rather than|instead of|must die|must not|mutant|mutation|negative fixture|test fixture|quoted example)\b/iu;

const NEGATIVE_SECTION = /(?:^out$|\b(?:prohibited|non[- ]claims?|out of scope|kill|illegal|refusal|negative checks?|scope cuts?|reconciled discrepancies|unimplemented|claim limits?|stop conditions?)\b)/iu;

class PlanningProblem {
  constructor(code, file, message, line = 0) {
    this.code = code;
    this.file = file;
    this.message = message;
    this.line = line;
  }
}

function relative(root, file) {
  return path.relative(root, file).split(path.sep).join("/");
}

function add(problems, code, file, message, line = 0) {
  problems.push(new PlanningProblem(code, file, message, line));
}

function sortedFiles(directory, predicate = () => true) {
  if (!fs.existsSync(directory)) return [];
  const entries = fs.readdirSync(directory, { withFileTypes: true })
    .sort((a, b) => a.name.localeCompare(b.name, "en"));
  const result = [];
  for (const entry of entries) {
    const absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) result.push(...sortedFiles(absolute, predicate));
    if (entry.isFile() && predicate(absolute)) result.push(absolute);
  }
  return result;
}

function assertNoCaseFoldedFileCollisions(root, files, problems) {
  const seen = new Map();
  for (const file of files) {
    const key = relative(root, file).normalize("NFC").toLocaleLowerCase("en-US");
    const previous = seen.get(key);
    if (previous) {
      add(problems, "DUPLICATE_FILENAME", relative(root, file), `case-folded file name collides with ${previous}`);
    } else {
      seen.set(key, relative(root, file));
    }
  }
}

function strictJsonParse(text, label) {
  const value = JSON.parse(text);
  const duplicates = [];
  let cursor = 0;

  function skipWhitespace() {
    while (/\s/u.test(text[cursor] ?? "")) cursor += 1;
  }

  function scanString() {
    const start = cursor;
    cursor += 1;
    while (cursor < text.length) {
      const character = text[cursor];
      if (character === "\\") {
        cursor += 2;
      } else if (character === "\"") {
        cursor += 1;
        return JSON.parse(text.slice(start, cursor));
      } else {
        cursor += 1;
      }
    }
    throw new SyntaxError(`unterminated string in ${label}`);
  }

  function scanPrimitive() {
    const match = /^(?:-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?|true|false|null)/u.exec(text.slice(cursor));
    if (!match) throw new SyntaxError(`invalid JSON token in ${label} at byte ${cursor}`);
    cursor += match[0].length;
  }

  function scanValue(pointer) {
    skipWhitespace();
    const character = text[cursor];
    if (character === "{") {
      cursor += 1;
      skipWhitespace();
      const keys = new Set();
      if (text[cursor] === "}") {
        cursor += 1;
        return;
      }
      while (cursor < text.length) {
        skipWhitespace();
        if (text[cursor] !== "\"") throw new SyntaxError(`object key expected in ${label} at byte ${cursor}`);
        const key = scanString();
        const childPointer = `${pointer}/${String(key).replaceAll("~", "~0").replaceAll("/", "~1")}`;
        if (keys.has(key)) duplicates.push(childPointer);
        keys.add(key);
        skipWhitespace();
        if (text[cursor] !== ":") throw new SyntaxError(`colon expected in ${label} at byte ${cursor}`);
        cursor += 1;
        scanValue(childPointer);
        skipWhitespace();
        if (text[cursor] === "}") {
          cursor += 1;
          return;
        }
        if (text[cursor] !== ",") throw new SyntaxError(`comma expected in ${label} at byte ${cursor}`);
        cursor += 1;
      }
      throw new SyntaxError(`unterminated object in ${label}`);
    }
    if (character === "[") {
      cursor += 1;
      skipWhitespace();
      if (text[cursor] === "]") {
        cursor += 1;
        return;
      }
      let index = 0;
      while (cursor < text.length) {
        scanValue(`${pointer}/${index}`);
        index += 1;
        skipWhitespace();
        if (text[cursor] === "]") {
          cursor += 1;
          return;
        }
        if (text[cursor] !== ",") throw new SyntaxError(`comma expected in ${label} at byte ${cursor}`);
        cursor += 1;
      }
      throw new SyntaxError(`unterminated array in ${label}`);
    }
    if (character === "\"") {
      scanString();
      return;
    }
    scanPrimitive();
  }

  scanValue("");
  skipWhitespace();
  if (cursor !== text.length) throw new SyntaxError(`trailing JSON content in ${label} at byte ${cursor}`);
  return { value, duplicates };
}

function parseJsonFile(root, file, problems) {
  const label = relative(root, file);
  try {
    const parsed = strictJsonParse(fs.readFileSync(file, "utf8"), label);
    for (const pointer of parsed.duplicates) {
      add(problems, "DUPLICATE_JSON_KEY", label, `duplicate object key at ${pointer || "/"}`);
    }
    return parsed.value;
  } catch (error) {
    add(problems, "JSON_SYNTAX", label, error instanceof Error ? error.message : String(error));
    return null;
  }
}

function parseJsonLinesFile(root, file, problems) {
  const label = relative(root, file);
  const rows = [];
  const lines = fs.readFileSync(file, "utf8").split(/\r?\n/u);
  for (let index = 0; index < lines.length; index += 1) {
    if (lines[index].trim() === "") continue;
    try {
      const parsed = strictJsonParse(lines[index], `${label}:${index + 1}`);
      for (const pointer of parsed.duplicates) {
        add(problems, "DUPLICATE_JSON_KEY", label, `duplicate object key at ${pointer || "/"}`, index + 1);
      }
      if (!parsed.value || typeof parsed.value !== "object" || Array.isArray(parsed.value)) {
        add(problems, "JSONL_ROW_TYPE", label, "JSONL row must be an object", index + 1);
      } else {
        rows.push({ value: parsed.value, line: index + 1 });
      }
    } catch (error) {
      add(problems, "JSONL_SYNTAX", label, error instanceof Error ? error.message : String(error), index + 1);
    }
  }
  return rows;
}

function walk(value, visit, pointer = "") {
  visit(value, pointer);
  if (Array.isArray(value)) {
    value.forEach((child, index) => walk(child, visit, `${pointer}/${index}`));
  } else if (value && typeof value === "object") {
    for (const key of Object.keys(value).sort()) {
      walk(value[key], visit, `${pointer}/${key.replaceAll("~", "~0").replaceAll("/", "~1")}`);
    }
  }
}

function normalizedObjectName(value) {
  return String(value).normalize("NFKC").replaceAll(/[^a-z0-9]/giu, "").toLowerCase();
}

function rootObjectKinds(schema) {
  const declarations = [schema];
  if (Array.isArray(schema?.oneOf)) declarations.push(...schema.oneOf);
  return declarations
    .filter((declaration) => Array.isArray(declaration?.required) && declaration.required.includes("kind"))
    .map((declaration) => declaration?.properties?.kind?.const)
    .filter((kind) => typeof kind === "string");
}

function enumArrays(schema) {
  const enums = [];
  walk(schema, (node, pointer) => {
    if (!node || typeof node !== "object" || Array.isArray(node) || !Array.isArray(node.enum)) return;
    enums.push({ pointer, values: node.enum });
  });
  return enums;
}

function constantValues(schema) {
  const values = new Set();
  walk(schema, (node) => {
    if (node && typeof node === "object" && !Array.isArray(node) && typeof node.const === "string") {
      values.add(node.const);
    }
  });
  return values;
}

function decodePointerSegment(segment) {
  let decoded = "";
  for (let index = 0; index < segment.length; index += 1) {
    if (segment[index] !== "~") {
      decoded += segment[index];
      continue;
    }
    const escape = segment[index + 1];
    if (escape === "0") decoded += "~";
    else if (escape === "1") decoded += "/";
    else return { error: `JSON Pointer segment contains invalid escape ~${escape ?? ""}` };
    index += 1;
  }
  return { value: decoded };
}

function pointerValue(value, fragment) {
  if (fragment === "" || fragment === "#") return { found: true, value };
  if (!fragment.startsWith("#/")) return { found: false, value: undefined };
  let cursor = value;
  for (const rawSegment of fragment.slice(2).split("/")) {
    const decoded = decodePointerSegment(rawSegment);
    if (decoded.error) return { found: false, value: undefined, error: decoded.error };
    if (!cursor || typeof cursor !== "object" || !Object.hasOwn(cursor, decoded.value)) {
      return { found: false, value: undefined };
    }
    cursor = cursor[decoded.value];
  }
  return { found: true, value: cursor };
}

function pointerChild(pointer, segment) {
  const escaped = String(segment).replaceAll("~", "~0").replaceAll("/", "~1");
  return `${pointer}/${escaped}`;
}

function displayedPointer(pointer) {
  return pointer === "" ? "/" : pointer;
}

function isSchemaNode(value) {
  return typeof value === "boolean" || (value !== null && typeof value === "object" && !Array.isArray(value));
}

function schemaNodePointers(schema, pointer = "", found = new Set()) {
  found.add(pointer === "" ? "" : `#${pointer}`);
  if (!schema || typeof schema !== "object" || Array.isArray(schema)) return found;

  for (const keyword of ["$defs", "properties"]) {
    const container = schema[keyword];
    if (!container || typeof container !== "object" || Array.isArray(container)) continue;
    for (const [name, child] of Object.entries(container)) {
      schemaNodePointers(child, pointerChild(pointerChild(pointer, keyword), name), found);
    }
  }
  for (const keyword of ["additionalProperties", "items", "contains", "if", "then"]) {
    if (Object.hasOwn(schema, keyword)) {
      schemaNodePointers(schema[keyword], pointerChild(pointer, keyword), found);
    }
  }
  for (const keyword of ["oneOf", "allOf"]) {
    if (!Array.isArray(schema[keyword])) continue;
    schema[keyword].forEach((branch, index) => {
      schemaNodePointers(branch, pointerChild(pointerChild(pointer, keyword), index), found);
    });
  }
  return found;
}

const SCHEMA_KEYWORDS = new Set([
  "$schema", "$id", "$anchor", "$comment", "$defs", "$ref",
  "title", "description", "default", "examples", "deprecated", "readOnly", "writeOnly",
  "type", "const", "enum", "required", "properties", "additionalProperties",
  "items", "minItems", "maxItems", "uniqueItems", "contains",
  "minLength", "maxLength", "pattern", "minimum", "maximum",
  "oneOf", "allOf", "if", "then",
]);

const JSON_SCHEMA_TYPES = new Set(["null", "boolean", "object", "array", "number", "integer", "string"]);

function validateSchemaShape(root, file, schema, problems, pointer = "") {
  const label = relative(root, file);
  if (typeof schema === "boolean") return;
  if (!schema || typeof schema !== "object" || Array.isArray(schema)) {
    add(problems, "SCHEMA_NODE_TYPE", label, `${displayedPointer(pointer)} must be a schema object or boolean`);
    return;
  }

  for (const keyword of Object.keys(schema)) {
    if (!SCHEMA_KEYWORDS.has(keyword)) {
      add(
        problems,
        "SCHEMA_KEYWORD_UNSUPPORTED",
        label,
        `${displayedPointer(pointerChild(pointer, keyword))} is outside the validator's fail-closed assertion subset`,
      );
    }
  }

  for (const keyword of ["$schema", "$id", "$anchor", "$comment", "$ref", "title", "description"]) {
    if (Object.hasOwn(schema, keyword) && typeof schema[keyword] !== "string") {
      add(problems, "SCHEMA_KEYWORD_TYPE", label, `${displayedPointer(pointerChild(pointer, keyword))} must be a string`);
    }
  }
  if (pointer !== "" && (Object.hasOwn(schema, "$schema") || Object.hasOwn(schema, "$id"))) {
    add(
      problems,
      "SCHEMA_FEATURE_UNSUPPORTED",
      label,
      `${displayedPointer(pointer)} may not start a nested schema resource; nested $schema and $id resolution is outside the validator subset`,
    );
  }
  if (Object.hasOwn(schema, "$anchor")) {
    add(
      problems,
      "SCHEMA_FEATURE_UNSUPPORTED",
      label,
      `${displayedPointer(pointerChild(pointer, "$anchor"))} is unsupported because this validator resolves JSON Pointer fragments only`,
    );
  }
  for (const keyword of ["deprecated", "readOnly", "writeOnly"]) {
    if (Object.hasOwn(schema, keyword) && typeof schema[keyword] !== "boolean") {
      add(problems, "SCHEMA_KEYWORD_TYPE", label, `${displayedPointer(pointerChild(pointer, keyword))} must be a boolean`);
    }
  }
  if (Object.hasOwn(schema, "examples") && !Array.isArray(schema.examples)) {
    add(problems, "SCHEMA_KEYWORD_TYPE", label, `${displayedPointer(pointerChild(pointer, "examples"))} must be an array`);
  }

  if (Object.hasOwn(schema, "type")) {
    const declared = Array.isArray(schema.type) ? schema.type : [schema.type];
    if (declared.length === 0
        || declared.some((type) => typeof type !== "string" || !JSON_SCHEMA_TYPES.has(type))
        || new Set(declared).size !== declared.length) {
      add(
        problems,
        "SCHEMA_KEYWORD_VALUE",
        label,
        `${displayedPointer(pointerChild(pointer, "type"))} must name one or more unique supported JSON types`,
      );
    }
  }

  if (Object.hasOwn(schema, "enum")) {
    if (!Array.isArray(schema.enum) || schema.enum.length === 0) {
      add(problems, "SCHEMA_KEYWORD_TYPE", label, `${displayedPointer(pointerChild(pointer, "enum"))} must be a nonempty array`);
    } else {
      for (let left = 0; left < schema.enum.length; left += 1) {
        if (schema.enum.slice(0, left).some((value) => jsonEqual(value, schema.enum[left]))) {
          add(problems, "SCHEMA_ENUM_DUPLICATE", label, `${displayedPointer(pointerChild(pointer, "enum"))} contains a duplicate value`);
          break;
        }
      }
    }
  }

  if (Object.hasOwn(schema, "required")) {
    if (!Array.isArray(schema.required)
        || schema.required.some((entry) => typeof entry !== "string")
        || new Set(schema.required).size !== schema.required.length) {
      add(problems, "SCHEMA_KEYWORD_TYPE", label, `${displayedPointer(pointerChild(pointer, "required"))} must be an array of unique strings`);
    }
  }

  for (const keyword of ["$defs", "properties"]) {
    if (!Object.hasOwn(schema, keyword)) continue;
    const container = schema[keyword];
    if (!container || typeof container !== "object" || Array.isArray(container)) {
      add(problems, "SCHEMA_KEYWORD_TYPE", label, `${displayedPointer(pointerChild(pointer, keyword))} must be an object of schemas`);
      continue;
    }
    for (const [name, child] of Object.entries(container)) {
      validateSchemaShape(root, file, child, problems, pointerChild(pointerChild(pointer, keyword), name));
    }
  }

  for (const keyword of ["additionalProperties", "items", "contains", "if", "then"]) {
    if (!Object.hasOwn(schema, keyword)) continue;
    if (!isSchemaNode(schema[keyword])) {
      add(problems, "SCHEMA_KEYWORD_TYPE", label, `${displayedPointer(pointerChild(pointer, keyword))} must be a schema object or boolean`);
    } else {
      validateSchemaShape(root, file, schema[keyword], problems, pointerChild(pointer, keyword));
    }
  }
  if (Object.hasOwn(schema, "then") && !Object.hasOwn(schema, "if")) {
    add(problems, "SCHEMA_KEYWORD_DEPENDENCY", label, `${displayedPointer(pointerChild(pointer, "then"))} requires a sibling if schema`);
  }

  for (const keyword of ["oneOf", "allOf"]) {
    if (!Object.hasOwn(schema, keyword)) continue;
    const branches = schema[keyword];
    if (!Array.isArray(branches) || branches.length === 0) {
      add(problems, "SCHEMA_KEYWORD_TYPE", label, `${displayedPointer(pointerChild(pointer, keyword))} must be a nonempty array of schemas`);
      continue;
    }
    branches.forEach((branch, index) => {
      if (!isSchemaNode(branch)) {
        add(
          problems,
          "SCHEMA_NODE_TYPE",
          label,
          `${displayedPointer(pointerChild(pointerChild(pointer, keyword), index))} must be a schema object or boolean`,
        );
      } else {
        validateSchemaShape(root, file, branch, problems, pointerChild(pointerChild(pointer, keyword), index));
      }
    });
  }

  for (const keyword of ["minItems", "maxItems", "minLength", "maxLength"]) {
    if (Object.hasOwn(schema, keyword) && (!Number.isInteger(schema[keyword]) || schema[keyword] < 0)) {
      add(problems, "SCHEMA_KEYWORD_VALUE", label, `${displayedPointer(pointerChild(pointer, keyword))} must be a nonnegative integer`);
    }
  }
  if (Number.isInteger(schema.minItems) && Number.isInteger(schema.maxItems) && schema.minItems > schema.maxItems) {
    add(problems, "SCHEMA_KEYWORD_VALUE", label, `${displayedPointer(pointer)} has minItems greater than maxItems`);
  }
  if (Number.isInteger(schema.minLength) && Number.isInteger(schema.maxLength) && schema.minLength > schema.maxLength) {
    add(problems, "SCHEMA_KEYWORD_VALUE", label, `${displayedPointer(pointer)} has minLength greater than maxLength`);
  }
  if (Object.hasOwn(schema, "uniqueItems") && typeof schema.uniqueItems !== "boolean") {
    add(problems, "SCHEMA_KEYWORD_TYPE", label, `${displayedPointer(pointerChild(pointer, "uniqueItems"))} must be a boolean`);
  }
  if (Object.hasOwn(schema, "pattern")) {
    if (typeof schema.pattern !== "string") {
      add(problems, "SCHEMA_KEYWORD_TYPE", label, `${displayedPointer(pointerChild(pointer, "pattern"))} must be a string`);
    } else {
      try {
        new RegExp(schema.pattern, "u");
      } catch (error) {
        add(
          problems,
          "SCHEMA_PATTERN_INVALID",
          label,
          `${displayedPointer(pointerChild(pointer, "pattern"))} is not a valid ECMAScript Unicode pattern: ${error instanceof Error ? error.message : String(error)}`,
        );
      }
    }
  }
  for (const keyword of ["minimum", "maximum"]) {
    if (Object.hasOwn(schema, keyword) && (typeof schema[keyword] !== "number" || !Number.isFinite(schema[keyword]))) {
      add(problems, "SCHEMA_KEYWORD_TYPE", label, `${displayedPointer(pointerChild(pointer, keyword))} must be a finite number`);
    }
  }
  if (typeof schema.minimum === "number" && typeof schema.maximum === "number" && schema.minimum > schema.maximum) {
    add(problems, "SCHEMA_KEYWORD_VALUE", label, `${displayedPointer(pointer)} has minimum greater than maximum`);
  }
}

function jsonEqual(left, right) {
  // JSON Schema numeric equality treats -0 and 0 as equal; JSON cannot carry
  // NaN, so strict equality is the correct primitive comparison here.
  if (left === right) return true;
  if (Array.isArray(left) || Array.isArray(right)) {
    return Array.isArray(left)
      && Array.isArray(right)
      && left.length === right.length
      && left.every((value, index) => jsonEqual(value, right[index]));
  }
  if (left && right && typeof left === "object" && typeof right === "object") {
    const leftKeys = Object.keys(left).sort();
    const rightKeys = Object.keys(right).sort();
    return leftKeys.length === rightKeys.length
      && leftKeys.every((key, index) => key === rightKeys[index] && jsonEqual(left[key], right[key]));
  }
  return false;
}

function instanceMatchesType(value, type) {
  switch (type) {
    case "null": return value === null;
    case "boolean": return typeof value === "boolean";
    case "object": return value !== null && typeof value === "object" && !Array.isArray(value);
    case "array": return Array.isArray(value);
    case "number": return typeof value === "number" && Number.isFinite(value);
    case "integer": return typeof value === "number" && Number.isFinite(value) && Number.isInteger(value);
    case "string": return typeof value === "string";
    default: return false;
  }
}

function resolveSchemaReference(currentFile, reference, schemasByFile) {
  if (typeof reference !== "string") return { error: "$ref is not a string" };
  const hash = reference.indexOf("#");
  if (hash !== reference.lastIndexOf("#")) return { error: `$ref has more than one fragment marker: ${reference}` };
  const targetPart = hash < 0 ? reference : reference.slice(0, hash);
  const encodedFragment = hash < 0 ? "" : reference.slice(hash + 1);
  if (/^[a-z][a-z0-9+.-]*:/iu.test(targetPart) || path.isAbsolute(targetPart)) {
    return { error: `$ref must be local or relative: ${reference}` };
  }
  const targetFile = targetPart === "" ? currentFile : path.resolve(path.dirname(currentFile), targetPart);
  const targetSchema = schemasByFile.get(targetFile);
  if (!targetSchema) return { error: `$ref targets an unavailable schema: ${reference}` };
  let decodedFragment;
  try {
    decodedFragment = decodeURIComponent(encodedFragment);
  } catch {
    return { error: `$ref has invalid fragment encoding: ${reference}` };
  }
  const fragment = decodedFragment === "" ? "" : `#${decodedFragment}`;
  if (fragment !== "" && !fragment.startsWith("#/")) {
    return { error: `$ref fragment must be an empty fragment or JSON Pointer: ${reference}` };
  }
  const resolved = pointerValue(targetSchema, fragment);
  if (resolved.error) return { error: `$ref has an invalid JSON Pointer: ${resolved.error}` };
  if (!resolved.found) return { error: `$ref targets a missing JSON Pointer: ${reference}` };
  if (!schemaNodePointers(targetSchema).has(fragment)) {
    return { error: `$ref targets a JSON value outside a schema-bearing position: ${reference}` };
  }
  if (!isSchemaNode(resolved.value)) {
    return { error: `$ref targets a JSON value that is not a schema node: ${reference}` };
  }
  return {
    file: targetFile,
    fragment,
    schema: resolved.value,
    key: `${targetFile}${fragment}`,
  };
}

function validateInstance(value, schema, context, pointer = "", referenceTrail = new Set()) {
  if (schema === true) return [];
  if (schema === false) return [`${displayedPointer(pointer)} is rejected by a false schema`];
  if (!schema || typeof schema !== "object" || Array.isArray(schema)) {
    return [`${displayedPointer(pointer)} cannot be checked because the mapped schema node is invalid`];
  }

  const errors = [];
  if (Object.hasOwn(schema, "$ref")) {
    const target = resolveSchemaReference(context.currentFile, schema.$ref, context.schemasByFile);
    if (target.error) {
      errors.push(`${displayedPointer(pointer)} cannot resolve schema reference: ${target.error}`);
    } else if (referenceTrail.has(target.key)) {
      errors.push(`${displayedPointer(pointer)} encounters a cyclic schema reference at ${schema.$ref}`);
    } else {
      const nextTrail = new Set(referenceTrail);
      nextTrail.add(target.key);
      errors.push(...validateInstance(
        value,
        target.schema,
        { ...context, currentFile: target.file },
        pointer,
        nextTrail,
      ));
    }
  }

  if (Object.hasOwn(schema, "type")) {
    const types = Array.isArray(schema.type) ? schema.type : [schema.type];
    if (!types.some((type) => instanceMatchesType(value, type))) {
      errors.push(`${displayedPointer(pointer)} must have type ${types.join(" or ")}`);
    }
  }
  if (Object.hasOwn(schema, "const") && !jsonEqual(value, schema.const)) {
    errors.push(`${displayedPointer(pointer)} must equal the schema const`);
  }
  if (Array.isArray(schema.enum) && !schema.enum.some((entry) => jsonEqual(value, entry))) {
    errors.push(`${displayedPointer(pointer)} must equal one of the declared enum values`);
  }

  const objectValue = value !== null && typeof value === "object" && !Array.isArray(value);
  if (objectValue) {
    if (Array.isArray(schema.required)) {
      for (const required of schema.required) {
        if (!Object.hasOwn(value, required)) {
          errors.push(`${displayedPointer(pointerChild(pointer, required))} is required`);
        }
      }
    }
    if (schema.properties && typeof schema.properties === "object" && !Array.isArray(schema.properties)) {
      for (const [property, propertySchema] of Object.entries(schema.properties)) {
        if (Object.hasOwn(value, property)) {
          errors.push(...validateInstance(
            value[property],
            propertySchema,
            context,
            pointerChild(pointer, property),
            referenceTrail,
          ));
        }
      }
    }
    if (Object.hasOwn(schema, "additionalProperties")) {
      const declared = new Set(Object.keys(schema.properties ?? {}));
      for (const property of Object.keys(value)) {
        if (declared.has(property)) continue;
        const propertyPointer = pointerChild(pointer, property);
        if (schema.additionalProperties === false) {
          errors.push(`${displayedPointer(propertyPointer)} is an undeclared additional property`);
        } else if (schema.additionalProperties !== true) {
          errors.push(...validateInstance(
            value[property],
            schema.additionalProperties,
            context,
            propertyPointer,
            referenceTrail,
          ));
        }
      }
    }
  }

  if (Array.isArray(value)) {
    if (Number.isInteger(schema.minItems) && value.length < schema.minItems) {
      errors.push(`${displayedPointer(pointer)} must contain at least ${schema.minItems} items`);
    }
    if (Number.isInteger(schema.maxItems) && value.length > schema.maxItems) {
      errors.push(`${displayedPointer(pointer)} must contain at most ${schema.maxItems} items`);
    }
    if (schema.uniqueItems === true) {
      for (let index = 0; index < value.length; index += 1) {
        if (value.slice(0, index).some((entry) => jsonEqual(entry, value[index]))) {
          errors.push(`${displayedPointer(pointerChild(pointer, index))} duplicates an earlier array item`);
          break;
        }
      }
    }
    if (Object.hasOwn(schema, "items")) {
      value.forEach((entry, index) => {
        errors.push(...validateInstance(entry, schema.items, context, pointerChild(pointer, index), referenceTrail));
      });
    }
    if (Object.hasOwn(schema, "contains")) {
      const matches = value.filter((entry, index) =>
        validateInstance(entry, schema.contains, context, pointerChild(pointer, index), referenceTrail).length === 0);
      if (matches.length === 0) errors.push(`${displayedPointer(pointer)} must contain at least one item matching contains`);
    }
  }

  if (typeof value === "string") {
    if (Number.isInteger(schema.minLength) && [...value].length < schema.minLength) {
      errors.push(`${displayedPointer(pointer)} must contain at least ${schema.minLength} Unicode code points`);
    }
    if (Number.isInteger(schema.maxLength) && [...value].length > schema.maxLength) {
      errors.push(`${displayedPointer(pointer)} must contain at most ${schema.maxLength} Unicode code points`);
    }
    if (typeof schema.pattern === "string") {
      try {
        if (!new RegExp(schema.pattern, "u").test(value)) {
          errors.push(`${displayedPointer(pointer)} does not match the declared pattern`);
        }
      } catch {
        errors.push(`${displayedPointer(pointer)} cannot be checked because the declared pattern is invalid`);
      }
    }
  }

  if (typeof value === "number" && Number.isFinite(value)) {
    if (typeof schema.minimum === "number" && value < schema.minimum) {
      errors.push(`${displayedPointer(pointer)} must be greater than or equal to ${schema.minimum}`);
    }
    if (typeof schema.maximum === "number" && value > schema.maximum) {
      errors.push(`${displayedPointer(pointer)} must be less than or equal to ${schema.maximum}`);
    }
  }

  if (Array.isArray(schema.oneOf)) {
    const matchCount = schema.oneOf.filter((branch) =>
      validateInstance(value, branch, context, pointer, referenceTrail).length === 0).length;
    if (matchCount !== 1) {
      errors.push(`${displayedPointer(pointer)} must match exactly one oneOf branch; matched ${matchCount}`);
    }
  }
  if (Array.isArray(schema.allOf)) {
    schema.allOf.forEach((branch, index) => {
      const branchErrors = validateInstance(value, branch, context, pointer, referenceTrail);
      errors.push(...branchErrors.map((error) => `allOf branch ${index}: ${error}`));
    });
  }
  if (Object.hasOwn(schema, "if")
      && validateInstance(value, schema.if, context, pointer, referenceTrail).length === 0
      && Object.hasOwn(schema, "then")) {
    errors.push(...validateInstance(value, schema.then, context, pointer, referenceTrail));
  }

  return errors;
}

function validateSchemaReferences(root, schemaFiles, schemasByFile, problems) {
  for (const file of schemaFiles) {
    const schema = schemasByFile.get(file);
    if (!schema) continue;
    walk(schema, (node, pointer) => {
      if (!node || typeof node !== "object" || Array.isArray(node) || !Object.hasOwn(node, "$ref")) return;
      const resolved = resolveSchemaReference(file, node.$ref, schemasByFile);
      if (resolved.error) {
        add(
          problems,
          "SCHEMA_REF_INVALID",
          relative(root, file),
          `${displayedPointer(pointerChild(pointer, "$ref"))}: ${resolved.error}`,
        );
      }
    });
  }
}

function authorityDeclarations(node) {
  if (!node || typeof node !== "object" || Array.isArray(node)) {
    return { values: [], invalidContainer: true };
  }
  const values = [];
  const hasConst = Object.hasOwn(node, "const");
  const hasEnum = Object.hasOwn(node, "enum");
  let invalidContainer = hasConst === hasEnum;
  if (hasConst) values.push(node.const);
  if (hasEnum) {
    if (Array.isArray(node.enum)) values.push(...node.enum);
    else invalidContainer = true;
  }
  return { values, invalidContainer };
}

function validateEnumGroupAgreements(root, schemaDirectory, schemasByFile, problems) {
  for (const [group, requiredValues] of ENUM_GROUPS) {
    const bindings = ENUM_GROUP_BINDINGS.get(group) ?? [];
    const declared = [];
    let invalidContainer = false;
    for (const binding of bindings) {
      const file = path.join(schemaDirectory, binding.schema);
      const schema = schemasByFile.get(file);
      if (!schema) continue;
      const resolved = pointerValue(schema, binding.pointer);
      if (!resolved.found) {
        add(
          problems,
          "ENUM_BINDING_MISSING",
          relative(root, file),
          `${group} authority pointer is missing: ${binding.pointer}`,
        );
        continue;
      }
      const authority = authorityDeclarations(resolved.value);
      invalidContainer ||= authority.invalidContainer;
      declared.push(...authority.values);
    }
    if (invalidContainer) {
      add(
        problems,
        "ENUM_AUTHORITY_SHAPE",
        "spec/schema/v1",
        `${group} authority must use schema const values or an enum array`,
      );
    }
    const nonStrings = declared.filter((value) => typeof value !== "string");
    if (nonStrings.length > 0) {
      add(
        problems,
        "ENUM_AUTHORITY_NON_STRING",
        "spec/schema/v1",
        `${group} authority contains non-string declarations: ${nonStrings.map((value) => JSON.stringify(value)).join(", ")}`,
      );
    }
    const declaredStrings = declared.filter((value) => typeof value === "string").sort();
    const requiredStrings = [...requiredValues].sort();
    const repeatedStrings = [...new Set(declaredStrings.filter((value, index) => declaredStrings.indexOf(value) !== index))];
    if (repeatedStrings.length > 0) {
      add(
        problems,
        "ENUM_AUTHORITY_MULTIPLICITY",
        "spec/schema/v1",
        `${group} authority declares terms more than once: ${repeatedStrings.join(", ")}`,
      );
    }
    const exact = nonStrings.length === 0
      && !invalidContainer
      && declaredStrings.length === requiredStrings.length
      && declaredStrings.every((value, index) => value === requiredStrings[index]);
    if (!exact) {
      add(
        problems,
        "ENUM_TERM_MISSING",
        "spec/schema/v1",
        `${group} authority declaration must equal exactly once each: ${requiredValues.join(", ")}; found: ${declared.map((value) => JSON.stringify(value)).join(", ") || "(none)"}`,
      );
    }
  }
}

function validateSchemas(root, problems) {
  const schemaDirectory = path.join(root, "spec/schema/v1");
  const files = sortedFiles(schemaDirectory, (file) => file.endsWith(".schema.json"));
  assertNoCaseFoldedFileCollisions(root, files, problems);

  for (const required of REQUIRED_SCHEMA_FILES) {
    if (!fs.existsSync(path.join(schemaDirectory, required))) {
      add(problems, "REQUIRED_FILE_MISSING", `spec/schema/v1/${required}`, "required schema file is missing");
    }
  }

  const schemasByFile = new Map();
  const identifiers = new Map();
  const allEnums = [];

  for (const file of files) {
    const schema = parseJsonFile(root, file, problems);
    if (!schema || typeof schema !== "object" || Array.isArray(schema)) {
      if (schema !== null) add(problems, "SCHEMA_ROOT_TYPE", relative(root, file), "schema root must be an object");
      continue;
    }
    schemasByFile.set(file, schema);
    if (schema.$schema !== "https://json-schema.org/draft/2020-12/schema") {
      add(problems, "SCHEMA_DIALECT", relative(root, file), "schema must declare JSON Schema 2020-12 exactly");
    }
    const expectedIdentifier = `${SCHEMA_ID_BASE}${path.basename(file)}`;
    if (typeof schema.$id !== "string" || schema.$id.length === 0) {
      add(problems, "SCHEMA_ID_MISSING", relative(root, file), "schema must have a nonempty $id");
    } else if (schema.$id !== expectedIdentifier) {
      add(problems, "SCHEMA_ID_MISMATCH", relative(root, file), `$id must equal ${expectedIdentifier}`);
    } else if (identifiers.has(schema.$id)) {
      add(problems, "SCHEMA_ID_DUPLICATE", relative(root, file), `$id duplicates ${identifiers.get(schema.$id)}`);
    } else {
      identifiers.set(schema.$id, relative(root, file));
    }
    validateSchemaShape(root, file, schema, problems);
    for (const value of constantValues(schema)) {
      if (FORBIDDEN_ENUM_VALUES.has(value)) {
        add(problems, "ENUM_FORBIDDEN_TERM", relative(root, file), `schema const contains prohibited enum term ${value}`);
      }
    }
    for (const entry of enumArrays(schema)) {
      allEnums.push({ ...entry, file });
    }
  }

  for (const { objectName, schemaFile } of EXAMPLE_SCHEMA_MAPPINGS.values()) {
    const file = path.join(schemaDirectory, schemaFile);
    const schema = schemasByFile.get(file);
    if (!schema) continue;
    const declarationCount = rootObjectKinds(schema).filter((kind) => kind === objectName).length;
    if (declarationCount !== 1) {
      add(
        problems,
        "SCHEMA_MAPPED_OBJECT_MISSING",
        relative(root, file),
        `mapped schema must require and declare root kind ${objectName} exactly once; found ${declarationCount}`,
      );
    }
  }

  for (const { file, pointer, values } of allEnums) {
    if (values.some((value) => typeof value !== "string")) {
      add(problems, "ENUM_NON_STRING", relative(root, file), `enum at ${pointer || "/"} contains a non-string term`);
    }
    for (const value of values) {
      if (FORBIDDEN_ENUM_VALUES.has(value)) {
        add(problems, "ENUM_FORBIDDEN_TERM", relative(root, file), `enum at ${pointer || "/"} contains prohibited term ${value}`);
      }
    }
  }

  validateEnumGroupAgreements(root, schemaDirectory, schemasByFile, problems);

  validateSchemaReferences(root, files, schemasByFile, problems);
  return { files, schemasByFile };
}

function validateExamples(root, schemasByFile, problems) {
  const exampleDirectory = path.join(root, "spec/examples/v1");
  const schemaDirectory = path.join(root, "spec/schema/v1");
  const files = sortedFiles(exampleDirectory, (file) => file.endsWith(".json"));
  assertNoCaseFoldedFileCollisions(root, files, problems);
  const represented = new Set();

  for (const required of REQUIRED_EXAMPLE_FILES) {
    if (!fs.existsSync(path.join(exampleDirectory, required))) {
      add(problems, "REQUIRED_FILE_MISSING", `spec/examples/v1/${required}`, "required valid example is missing");
    }
  }

  for (const file of files) {
    const examplePath = relative(exampleDirectory, file);
    const isValidExample = examplePath.endsWith(".valid.json");
    const mapping = isValidExample ? EXAMPLE_SCHEMA_MAPPINGS.get(examplePath) : undefined;
    if (isValidExample && !mapping) {
      add(
        problems,
        "EXAMPLE_SCHEMA_MAPPING_MISSING",
        relative(root, file),
        "every .valid.json example must have an explicit schema mapping",
      );
    }

    const value = parseJsonFile(root, file, problems);
    if (!value || typeof value !== "object" || Array.isArray(value)) {
      if (value !== null) add(problems, "EXAMPLE_ROOT_TYPE", relative(root, file), "example root must be an object");
      continue;
    }
    if (!isValidExample || !mapping) continue;
    if (value.kind !== mapping.objectName) {
      add(
        problems,
        "EXAMPLE_KIND_MISMATCH",
        relative(root, file),
        `mapped example kind must equal ${JSON.stringify(mapping.objectName)} exactly`,
      );
    } else {
      represented.add(normalizedObjectName(mapping.objectName));
    }
    const schemaBasename = mapping.schemaFile;
    const schemaFile = path.join(schemaDirectory, schemaBasename);
    const schema = schemasByFile.get(schemaFile);
    if (!schema) {
      add(
        problems,
        "EXAMPLE_SCHEMA_MISSING",
        relative(root, file),
        `mapped schema is unavailable: spec/schema/v1/${schemaBasename}`,
      );
      continue;
    }
    const validationErrors = validateInstance(value, schema, {
      currentFile: schemaFile,
      schemasByFile,
    });
    for (const error of validationErrors) {
      add(problems, "EXAMPLE_SCHEMA_VALIDATION", relative(root, file), error);
    }
  }

  for (const [objectName] of OBJECTS) {
    if (!represented.has(normalizedObjectName(objectName))) {
      add(problems, "EXAMPLE_OBJECT_MISSING", "spec/examples/v1", `no valid example represents ${objectName}`);
    }
  }
  return { files };
}

const P07_FILE_ROSTER = Object.freeze([
  "README.md",
  "contract.test.mjs",
  "decision.json",
  "fixture.json",
  "harness.mjs",
  "manifest.json",
]);

const P07_JSON_FILE_ROSTER = Object.freeze(["decision.json", "fixture.json", "manifest.json"]);
const P07_RUNTIME_SOURCE_SHA256 = Object.freeze({
  "README.md": "sha256:40fb600a59fddd7ea6a5648e249c11ecd4e439ddd250c8ae09a40b31ec58a8b9",
  "contract.test.mjs": "sha256:0ded06ad24d9b218fef7776835b26124483d788f2eb49f69d0ffb9806c85e021",
  "decision.json": "sha256:8ff677acdf129ff10c862c96542903f5c5b0e1c94884ce75aa8b5bc4d1bb6a53",
  "fixture.json": "sha256:e4474fa48d41662d2ac7e188f19afa20196de4530ae1edceaed03bdde3801fba",
  "harness.mjs": "sha256:51eb70171f044cd77776d36dceed8101e45c774c3b5f70327fe144061b08bb34",
  "manifest.json": "sha256:35bb1bb0103f95b4449754640fc30667d726d01240b9f521b922115228d0b3f3",
});
const P07_SAFE_INTEGER_PATTERN = "^(?:0|-?(?:[1-9][0-9]{0,14}|[1-8][0-9]{15}|900[0-6][0-9]{12}|90070[0-9]{11}|90071[0-8][0-9]{10}|900719[0-8][0-9]{9}|9007199[01][0-9]{8}|90071992[0-4][0-9]{7}|900719925[0-3][0-9]{6}|9007199254[0-6][0-9]{5}|90071992547[0-3][0-9]{4}|9007199254740[0-8][0-9]{2}|90071992547409[0-8][0-9]|900719925474099[01]))$";
const P07_SAFE_INTEGER_LIMIT = 9007199254740991n;

function rawSha256(bytes) {
  return `sha256:${crypto.createHash("sha256").update(bytes).digest("hex")}`;
}

function p07CanonicalJSON(value) {
  if (value === null || typeof value === "boolean" || typeof value === "string") return JSON.stringify(value);
  if (typeof value === "number") {
    if (!Number.isSafeInteger(value) || Object.is(value, -0)) throw new Error("noncanonical number in P07 semantic body");
    return String(value);
  }
  if (Array.isArray(value)) return `[${value.map(p07CanonicalJSON).join(",")}]`;
  if (!value || typeof value !== "object") throw new Error("unsupported value in P07 semantic body");
  const keys = Object.keys(value).sort((left, right) => Buffer.compare(Buffer.from(left), Buffer.from(right)));
  return `{${keys.map((key) => `${JSON.stringify(key)}:${p07CanonicalJSON(value[key])}`).join(",")}}`;
}

function p07TypedDigest(kind, value) {
  const hash = crypto.createHash("sha256");
  hash.update(Buffer.from(`countershape/v1/${kind}\0`, "utf8"));
  hash.update(Buffer.from(p07CanonicalJSON(value), "utf8"));
  return `sha256:${hash.digest("hex")}`;
}

const C1_SCHEMA_RUNTIME_CASE_NAMES = Object.freeze([
  "duplicate evidence kind with distinct digest",
  "process evidence order",
  "clean process with primary",
  "spawn process mismatch",
  "scope order",
  "scope evidence role",
  "scope aggregate",
  "private zero correlation",
]);
const C1_SCHEMA_RUNTIME_CORPUS_DIGEST = "78f5a2c76d38dc2de96b11cf0439f33f73427160fc441ba89f9f39897dab6d4b";

function c1SchemaRuntimeOverapproximationCases(finalizedRun) {
  const cases = C1_SCHEMA_RUNTIME_CASE_NAMES.map((name) => ({
    name,
    body: structuredClone(finalizedRun),
  }));
  const witness = (candidate) => candidate.body.closed_run_witness;
  witness(cases[0]).process_closure.evidence_refs[1].kind = "MATERIALIZATION_REVALIDATION";
  [
    witness(cases[1]).process_closure.evidence_refs[0],
    witness(cases[1]).process_closure.evidence_refs[1],
  ] = [
    witness(cases[1]).process_closure.evidence_refs[1],
    witness(cases[1]).process_closure.evidence_refs[0],
  ];
  witness(cases[2]).process_closure.primary_reason = "TIMEOUT";
  witness(cases[3]).spawn_observation = { status: "START_ERROR", error_code: "OS_START_ERROR" };
  [
    witness(cases[4]).standalone_scope.checks[0],
    witness(cases[4]).standalone_scope.checks[1],
  ] = [
    witness(cases[4]).standalone_scope.checks[1],
    witness(cases[4]).standalone_scope.checks[0],
  ];
  witness(cases[5]).standalone_scope.checks[0].evidence_ref.kind = "SERVICE_BINDINGS";
  witness(cases[6]).standalone_scope.status = "PARTIAL";
  witness(cases[7]).private_evidence.blob_count = 0;
  witness(cases[7]).private_evidence.aggregate_byte_count = 1;
  return cases;
}

function proveC1SchemaRuntimeIntersection(root) {
  const schemaProblems = [];
  const { schemasByFile } = validateSchemas(root, schemaProblems);
  if (schemaProblems.length > 0) {
    throw new Error(`schema baseline is invalid: ${schemaProblems.map((problem) => problem.code).join(", ")}`);
  }
  const runSchemaFile = path.join(root, "spec/schema/v1/finalized-contract-run.schema.json");
  const runSchema = schemasByFile.get(runSchemaFile);
  if (!runSchema) throw new Error("FinalizedContractRun schema is unavailable");
  const finalizedRun = JSON.parse(fs.readFileSync(
    path.join(root, "spec/examples/v1/finalized-contract-run.valid.json"),
    "utf8",
  ));
  const corpusHash = crypto.createHash("sha256");
  const cases = c1SchemaRuntimeOverapproximationCases(finalizedRun).map(({ name, body }) => {
    const validationErrors = validateInstance(body, runSchema, {
      currentFile: runSchemaFile,
      schemasByFile,
    });
    if (validationErrors.length > 0) {
      throw new Error(`${name} is not schema-valid: ${validationErrors.join("; ")}`);
    }
    const exact = Buffer.from(p07CanonicalJSON(body), "utf8");
    corpusHash.update(name).update("\0").update(exact).update("\0");
    return { name, body_base64: exact.toString("base64") };
  });
  const corpusDigest = corpusHash.digest("hex");
  if (corpusDigest !== C1_SCHEMA_RUNTIME_CORPUS_DIGEST) {
    throw new Error(`schema/runtime corpus drift: sha256:${corpusDigest} != sha256:${C1_SCHEMA_RUNTIME_CORPUS_DIGEST}`);
  }
  const goExecutable = process.env.COUNTERSHAPE_GO ?? "/opt/homebrew/bin/go";
  if (!path.isAbsolute(goExecutable)) throw new Error("COUNTERSHAPE_GO must be absolute");
  const inheritedNames = ["HOME", "TMPDIR", "GOTMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE"];
  const environment = Object.fromEntries(inheritedNames.flatMap((name) =>
    Object.hasOwn(process.env, name) ? [[name, process.env[name]]] : []));
  Object.assign(environment, {
    HOME: environment.HOME ?? "/",
    TMPDIR: environment.TMPDIR ?? "/tmp",
    GOENV: "off",
    GOWORK: "off",
    GOTOOLCHAIN: "local",
    GOPROXY: "off",
    GOSUMDB: "off",
    GOVCS: "*:off",
    CGO_ENABLED: "0",
    GOMAXPROCS: "2",
    GOFLAGS: "-mod=readonly -buildvcs=false -p=1",
    LANG: "C",
    LC_ALL: "C",
    TZ: "UTC",
    NO_COLOR: "1",
    PATH: "/usr/bin:/bin",
    COUNTERSHAPE_C1_SCHEMA_RUNTIME_PROBE: "1",
  });
  const payload = JSON.stringify({
    schema_version: "countershape/c1-schema-runtime-overapproximation-probe/v1",
    cases,
  });
  environment.COUNTERSHAPE_C1_SCHEMA_RUNTIME_PAYLOAD_BASE64 = Buffer.from(payload, "utf8").toString("base64");
  const result = spawnSync(goExecutable, [
    "test", "-mod=readonly", "-buildvcs=false", "-p=1", "-count=1", "-timeout=3m",
    "-run", "^TestC1SchemaRuntimeOverapproximationProbe$", "-v", "./internal/contractexec/model",
  ], {
    cwd: root,
    encoding: "utf8",
    env: environment,
    maxBuffer: 8 * 1024 * 1024,
    timeout: 180_000,
  });
  const marker = `COUNTERSHAPE_C1_SCHEMA_RUNTIME_OVERAPPROXIMATION_V1|${cases.length}|sha256:${corpusDigest}`;
  const markerPrefix = "COUNTERSHAPE_C1_SCHEMA_RUNTIME_OVERAPPROXIMATION_V1|";
  const markerLines = (result.stdout ?? "").split(/\r?\n/u).filter((line) => line.startsWith(markerPrefix));
  if (
    result.error
    || result.signal
    || result.status !== 0
    || result.stderr !== ""
    || markerLines.length !== 1
    || markerLines[0] !== marker
  ) {
    throw new Error(`Go semantic probe failed (${result.status ?? result.signal}): ${result.stderr || result.stdout || result.error}`);
  }
  return { cases: cases.length, corpusDigest: `sha256:${corpusDigest}` };
}

function isP07ReceiptNonclaim(pointer, key, value) {
  const fullPointer = `${pointer}/${key}`;
  return (
    fullPointer === "$/properties/determinism_profile/properties/emitter_introduces_execution_receipt"
    && value?.const === false
  ) || (
    fullPointer === "$/determinism_profile/emitter_introduces_execution_receipt"
    && value === false
  );
}

function collectP07ReceiptAuthority(value, pointer = "$", found = []) {
  if (Array.isArray(value)) {
    value.forEach((entry, index) => collectP07ReceiptAuthority(entry, `${pointer}/${index}`, found));
    return found;
  }
  if (!value || typeof value !== "object") return found;

  const keys = Object.keys(value);
  if (typeof value.$ref === "string" && /#\/\$defs\/ReceiptReference$/u.test(value.$ref)) {
    found.push(`${pointer}/$ref -> ${value.$ref}`);
  }
  if (
    value.authority === "didrun"
    || ["authority", "grade_verbatim", "commit_oid", "command_digest"].every((key) => Object.hasOwn(value, key))
  ) {
    found.push(`${pointer} contains didrun-shaped authority`);
  }

  for (const key of keys) {
    if (/receipt/iu.test(key) && !isP07ReceiptNonclaim(pointer, key, value[key])) {
      found.push(`${pointer}/${key} is receipt-bearing`);
    }
    collectP07ReceiptAuthority(value[key], `${pointer}/${key}`, found);
  }
  return found;
}

function p07SchemaReachesReceipt(currentFile, schema, schemasByFile, trail = new Set()) {
  if (collectP07ReceiptAuthority(schema).length > 0) return true;
  const references = [];
  walk(schema, (node) => {
    if (node && typeof node === "object" && !Array.isArray(node) && typeof node.$ref === "string") {
      references.push(node.$ref);
    }
  });
  for (const reference of references) {
    const target = resolveSchemaReference(currentFile, reference, schemasByFile);
    if (target.error) return true;
    if (trail.has(target.key)) continue;
    const nextTrail = new Set(trail);
    nextTrail.add(target.key);
    if (p07SchemaReachesReceipt(target.file, target.schema, schemasByFile, nextTrail)) return true;
  }
  return false;
}

function exactTupleFieldIDs(tuple) {
  if (!tuple || typeof tuple !== "object" || Array.isArray(tuple) || !Array.isArray(tuple.fields)) return null;
  return tuple.fields.map((field) => field?.field_id);
}

function p07SafeIntegerText(value) {
  if (typeof value !== "string" || !new RegExp(P07_SAFE_INTEGER_PATTERN, "u").test(value)) return false;
  try {
    const integer = BigInt(value);
    return integer >= -P07_SAFE_INTEGER_LIMIT && integer <= P07_SAFE_INTEGER_LIMIT;
  } catch {
    return false;
  }
}

function validateP07IntegerValues(root, file, value, problems) {
  walk(value, (node, pointer) => {
    if (!node || typeof node !== "object" || Array.isArray(node) || node.tag !== "INTEGER") return;
    const text = typeof node.canonical === "string" ? node.canonical : node.text;
    if (!p07SafeIntegerText(text)) {
      add(
        problems,
        "P07A_INTEGER_VALUE",
        relative(root, file),
        `${displayedPointer(pointer)} INTEGER must use canonical safe-range decimal text`,
      );
    }
  });
}

function replaceP07BundleMemberAndManifest(bundle, memberPath, bytes) {
  if (memberPath === "manifest.json") throw new Error("self-test helper cannot replace the manifest through itself");
  const member = bundle.files?.find((entry) => entry.path === memberPath);
  const manifestEntry = bundle.files?.find((entry) => entry.path === "manifest.json");
  if (!member || !manifestEntry || typeof manifestEntry.content_base64 !== "string") {
    throw new Error(`self-test mutation anchor is absent: ${memberPath} and manifest entries`);
  }
  member.content_base64 = bytes.toString("base64");
  member.byte_count = bytes.length;
  member.byte_sha256 = rawSha256(bytes);

  const manifest = JSON.parse(Buffer.from(manifestEntry.content_base64, "base64").toString("utf8"));
  const manifestMember = manifest.files?.find((entry) => entry.path === memberPath);
  if (!manifestMember) throw new Error(`self-test mutation anchor is absent: manifest ${memberPath} entry`);
  manifestMember.byte_count = member.byte_count;
  manifestMember.byte_sha256 = member.byte_sha256;
  const manifestBytes = Buffer.from(`${JSON.stringify(manifest)}\n`, "utf8");
  manifestEntry.content_base64 = manifestBytes.toString("base64");
  manifestEntry.byte_count = manifestBytes.length;
  manifestEntry.byte_sha256 = rawSha256(manifestBytes);
}

function validateP07Contracts(root, problems, schemasByFile) {
  const bundleSchemaFile = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
  const targetSchemaFile = path.join(root, "spec/schema/v1/contract-execution-target.schema.json");
  const runSchemaFile = path.join(root, "spec/schema/v1/finalized-contract-run.schema.json");
  const executionSchemaFile = path.join(root, "spec/schema/v1/contract-execution.schema.json");
  const commonSchemaFile = path.join(root, "spec/schema/v1/common.schema.json");
  const choicepointSchemaFile = path.join(root, "spec/schema/v1/choicepoint.schema.json");
  const decisionSchemaFile = path.join(root, "spec/schema/v1/decision-record.schema.json");
  const decisionExampleFile = path.join(root, "spec/examples/v1/decision-record.valid.json");
  const bundleExampleFile = path.join(root, "spec/examples/v1/contract-bundle.valid.json");
  const targetExampleFile = path.join(root, "spec/examples/v1/contract-execution-target.valid.json");
  const runExampleFile = path.join(root, "spec/examples/v1/finalized-contract-run.valid.json");
  const executionExampleFile = path.join(root, "spec/examples/v1/contract-execution.valid.json");
  const bundleSchema = parseJsonFile(root, bundleSchemaFile, problems);
  const targetSchema = parseJsonFile(root, targetSchemaFile, problems);
  const runSchema = parseJsonFile(root, runSchemaFile, problems);
  const executionSchema = parseJsonFile(root, executionSchemaFile, problems);
  const commonSchema = parseJsonFile(root, commonSchemaFile, problems);
  const choicepointSchema = parseJsonFile(root, choicepointSchemaFile, problems);
  const decisionSchema = parseJsonFile(root, decisionSchemaFile, problems);
  const decisionExample = parseJsonFile(root, decisionExampleFile, problems);
  const bundle = parseJsonFile(root, bundleExampleFile, problems);
  const target = parseJsonFile(root, targetExampleFile, problems);
  const finalizedRun = parseJsonFile(root, runExampleFile, problems);
  const execution = parseJsonFile(root, executionExampleFile, problems);

  const expectedChoiceModes = ["WHOLE_EXACT_CANONICAL_PROJECTION_V1", "ADAPTER_BOUND_PORTABLE_FIELDS_V1"];
  if (!jsonEqual(choicepointSchema?.properties?.choice_projection_mode?.enum, expectedChoiceModes)) {
    add(
      problems,
      "P07A_SCHEMA_AUTHORITY",
      relative(root, choicepointSchemaFile),
      `choice_projection_mode must preserve legacy history and admit exactly ${expectedChoiceModes.join(", ")}`,
    );
  }
  const expectedDecisionValueTags = [
    "MISSING",
    "STRING",
    "INTEGER",
    "BOOLEAN",
    "NULL",
    "BYTES",
    "ORDERED_STRING_LIST",
    "CANONICAL_JSON",
  ];
  if (!jsonEqual(decisionSchema?.$defs?.ExactValue?.properties?.tag?.enum, expectedDecisionValueTags)) {
    add(
      problems,
      "P07A_SCHEMA_AUTHORITY",
      relative(root, decisionSchemaFile),
      `DecisionRecord ExactValue must admit the exact portable compatibility tags ${expectedDecisionValueTags.join(", ")}`,
    );
  }
  const commonIntegerPattern = commonSchema?.$defs?.ExactValue?.oneOf
    ?.find((branch) => branch?.properties?.tag?.const === "INTEGER")
    ?.properties?.canonical?.pattern;
  const decisionIntegerPattern = decisionSchema?.$defs?.ExactValue?.allOf
    ?.find((branch) => branch?.if?.properties?.tag?.const === "INTEGER")
    ?.then?.properties?.text?.pattern;
  const requiredSafeIntegers = ["0", "1", "-1", "9007199254740991", "-9007199254740991"];
  const forbiddenIntegers = ["-0", "00", "01", "+1", "1.0", "1e0", "9007199254740992", "-9007199254740992"];
  if (
    commonIntegerPattern !== P07_SAFE_INTEGER_PATTERN
    || decisionIntegerPattern !== P07_SAFE_INTEGER_PATTERN
    || requiredSafeIntegers.some((value) => !p07SafeIntegerText(value))
    || forbiddenIntegers.some((value) => p07SafeIntegerText(value))
  ) {
    add(
      problems,
      "P07A_INTEGER_AUTHORITY",
      relative(root, decisionSchemaFile),
      "portable and DecisionRecord compatibility integers must share the exact canonical safe-range decimal profile",
    );
  }
  validateP07IntegerValues(root, bundleExampleFile, bundle, problems);
  validateP07IntegerValues(root, targetExampleFile, target, problems);
  validateP07IntegerValues(root, runExampleFile, finalizedRun, problems);
  validateP07IntegerValues(root, executionExampleFile, execution, problems);
  validateP07IntegerValues(root, decisionExampleFile, decisionExample, problems);
  const expectedBundleRequired = [
    "schema_version",
    "kind",
    "bundle_version",
    "decision_record_digest",
    "choicepoint_digest",
    "portable_source_digest",
    "portable_profile_digest",
    "decision_action",
    "emitter_version",
    "source_profile",
    "predicate",
    "files",
    "manifest_policy",
    "runtime_dependency_profile",
    "countershape_runtime_binding",
    "package_registry_binding",
    "environment_profile",
    "external_service_binding",
    "determinism_profile",
    "confidentiality_established",
  ];
  const bundleSourceProfileSchema = bundleSchema?.properties?.source_profile;
  const bundlePredicateSchema = bundleSchema?.properties?.predicate;
  const bundleFileItemSchema = bundleSchema?.properties?.files?.items;
  const bundleDeterminismSchema = bundleSchema?.properties?.determinism_profile;
  const expectedPredicateRequired = [
    "kind",
    "scope",
    "stimulus_digest",
    "portable_profile_digest",
    "selected_fields",
    "allowed_tuples",
  ];
  const expectedBundleFileRequired = ["path", "mode", "byte_count", "byte_sha256", "content_base64"];
  const expectedBundleFileContains = P07_FILE_ROSTER.map((filePath) => ({
    contains: {
      properties: { path: { const: filePath } },
      required: ["path"],
    },
  }));
  const expectedBundleScalarConstants = Object.freeze({
    schema_version: "countershape/v1",
    kind: "ContractBundle",
    bundle_version: "node-core-contract-bundle/v1",
    emitter_version: "node-exact-emitter/v1",
    manifest_policy: "COVERS_OTHER_FIVE_EXCLUDES_SELF_V1",
    runtime_dependency_profile: "NODE_CORE_ONLY_V1",
    countershape_runtime_binding: "ABSENT_BY_CONSTRUCTION",
    package_registry_binding: "NONE",
    environment_profile: "EXPLICIT_SPARSE_ALLOWLIST_V1",
    external_service_binding: "NONE",
  });
  const expectedDecisionActions = ["ALLOW_OBSERVED", "CUSTOM_EXPECTATION"];
  const expectedBase64Pattern = "^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$";
  const expectedRootDigestFields = [
    "decision_record_digest",
    "choicepoint_digest",
    "portable_source_digest",
    "portable_profile_digest",
  ];
  const expectedDeterminismRequired = [
    "scope",
    "emitter_introduces_time",
    "emitter_introduces_random_id",
    "emitter_introduces_absolute_path",
    "emitter_introduces_concrete_candidate_identity",
    "emitter_introduces_declared_secret_value",
    "emitter_introduces_host_runtime_fact",
    "emitter_introduces_execution_receipt",
  ];
  const expectedDeterminismScope = "EMITTER_INVENTED_STRUCTURAL_FACTS_EXCLUDING_AUTHORIZED_INPUT_CONTENT";
  const expectedDeterminismFalseFlags = expectedDeterminismRequired.slice(1);
  const expectedCustomExpectationConditional = {
    if: {
      properties: { decision_action: { const: "CUSTOM_EXPECTATION" } },
      required: ["decision_action"],
    },
    then: {
      properties: {
        predicate: {
          properties: {
            allowed_tuples: { minItems: 1, maxItems: 1 },
          },
        },
      },
    },
  };
  if (
    bundleSchema?.type !== "object"
    || bundleSchema?.additionalProperties !== false
    || !jsonEqual(bundleSchema?.required, expectedBundleRequired)
    || !jsonEqual(Object.keys(bundleSchema?.properties ?? {}), expectedBundleRequired)
    || bundlePredicateSchema?.additionalProperties !== false
    || !jsonEqual(bundlePredicateSchema?.required, expectedPredicateRequired)
    || !jsonEqual(Object.keys(bundlePredicateSchema?.properties ?? {}), expectedPredicateRequired)
    || bundleFileItemSchema?.additionalProperties !== false
    || !jsonEqual(bundleFileItemSchema?.required, expectedBundleFileRequired)
    || !jsonEqual(Object.keys(bundleFileItemSchema?.properties ?? {}), expectedBundleFileRequired)
    || Object.entries(expectedBundleScalarConstants).some(
      ([property, constant]) => bundleSchema?.properties?.[property]?.const !== constant || bundle?.[property] !== constant,
    )
    || !jsonEqual(bundleSchema?.properties?.decision_action?.enum, expectedDecisionActions)
    || !expectedDecisionActions.includes(bundle?.decision_action)
    || expectedRootDigestFields.some(
      (property) => bundleSchema?.properties?.[property]?.$ref !== "common.schema.json#/$defs/Digest",
    )
    || bundlePredicateSchema?.type !== "object"
    || bundlePredicateSchema?.properties?.kind?.const !== "one-of-exact/v1"
    || bundlePredicateSchema?.properties?.scope?.const !== "EXACT_WITNESSED_STIMULUS"
    || bundlePredicateSchema?.properties?.stimulus_digest?.$ref !== "common.schema.json#/$defs/Digest"
    || bundlePredicateSchema?.properties?.portable_profile_digest?.$ref !== "common.schema.json#/$defs/Digest"
    || bundle?.predicate?.kind !== "one-of-exact/v1"
    || bundle?.predicate?.scope !== "EXACT_WITNESSED_STIMULUS"
    || bundlePredicateSchema?.properties?.selected_fields?.type !== "array"
    || bundlePredicateSchema?.properties?.selected_fields?.minItems !== 1
    || bundlePredicateSchema?.properties?.selected_fields?.maxItems !== 64
    || bundlePredicateSchema?.properties?.selected_fields?.uniqueItems !== true
    || bundlePredicateSchema?.properties?.selected_fields?.items?.$ref !== "common.schema.json#/$defs/FieldId"
    || bundlePredicateSchema?.properties?.allowed_tuples?.type !== "array"
    || bundlePredicateSchema?.properties?.allowed_tuples?.minItems !== 1
    || bundlePredicateSchema?.properties?.allowed_tuples?.maxItems !== 4
    || bundlePredicateSchema?.properties?.allowed_tuples?.uniqueItems !== true
    || bundlePredicateSchema?.properties?.allowed_tuples?.items?.$ref !== "common.schema.json#/$defs/ExactTuple"
    || bundleSchema?.properties?.files?.type !== "array"
    || bundleSchema?.properties?.files?.minItems !== 6
    || bundleSchema?.properties?.files?.maxItems !== 6
    || bundleSchema?.properties?.files?.uniqueItems !== true
    || !jsonEqual(bundleSchema?.properties?.files?.allOf, expectedBundleFileContains)
    || !jsonEqual(bundleFileItemSchema?.properties?.path?.enum, P07_FILE_ROSTER)
    || bundleFileItemSchema?.type !== "object"
    || bundleFileItemSchema?.properties?.mode?.const !== "100644"
    || bundleFileItemSchema?.properties?.byte_count?.type !== "integer"
    || bundleFileItemSchema?.properties?.byte_count?.minimum !== 1
    || bundleFileItemSchema?.properties?.byte_count?.maximum !== 655360
    || bundleFileItemSchema?.properties?.byte_sha256?.$ref !== "common.schema.json#/$defs/Digest"
    || bundleFileItemSchema?.properties?.content_base64?.type !== "string"
    || bundleFileItemSchema?.properties?.content_base64?.minLength !== 4
    || bundleFileItemSchema?.properties?.content_base64?.pattern !== expectedBase64Pattern
    || bundleSchema?.properties?.confidentiality_established?.const !== false
    || bundle?.confidentiality_established !== false
    || !jsonEqual(bundle?.files?.map((entry) => entry?.path), P07_FILE_ROSTER)
    || bundle?.files?.some((entry) => entry?.mode !== "100644")
    || bundleDeterminismSchema?.type !== "object"
    || bundleDeterminismSchema?.additionalProperties !== false
    || !jsonEqual(bundleDeterminismSchema?.required, expectedDeterminismRequired)
    || !jsonEqual(Object.keys(bundleDeterminismSchema?.properties ?? {}), expectedDeterminismRequired)
    || bundleDeterminismSchema?.properties?.scope?.const !== expectedDeterminismScope
    || expectedDeterminismFalseFlags.some(
      (property) => bundleDeterminismSchema?.properties?.[property]?.const !== false,
    )
    || bundle?.determinism_profile?.scope !== expectedDeterminismScope
    || expectedDeterminismFalseFlags.some(
      (property) => bundle?.determinism_profile?.[property] !== false,
    )
    || !jsonEqual(bundleSchema?.allOf, [expectedCustomExpectationConditional])
  ) {
    add(
      problems,
      "P07_BUNDLE_AUTHORITY",
      relative(root, bundleSchemaFile),
      "ContractBundle root constants, predicate profile, file profile, determinism constants, and CUSTOM_EXPECTATION cardinality must keep their exact closed authority",
    );
  }
  const expectedSourceProfileRequired = [
    "runtime_family",
    "semantic_profile",
    "adapter_domain",
    "launch_profile",
    "subject_entrypoint",
    "start_profile",
    "scope",
  ];
  const expectedSourceProfileConditionals = [
    {
      if: {
        properties: { adapter_domain: { const: "CLI" } },
        required: ["adapter_domain"],
      },
      then: { properties: { start_profile: { const: "DIRECT_CHILD_V1" } } },
    },
    {
      if: {
        properties: { adapter_domain: { const: "HTTP" } },
        required: ["adapter_domain"],
      },
      then: {
        properties: {
          start_profile: { const: "NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1" },
        },
      },
    },
  ];
  if (
    bundleSourceProfileSchema?.type !== "object"
    || bundleSourceProfileSchema?.additionalProperties !== false
    || !jsonEqual(bundleSourceProfileSchema?.required, expectedSourceProfileRequired)
    || !jsonEqual(Object.keys(bundleSourceProfileSchema?.properties ?? {}), expectedSourceProfileRequired)
    || bundleSourceProfileSchema?.properties?.runtime_family?.const !== "NODE"
    || bundleSourceProfileSchema?.properties?.semantic_profile?.const !== "countershape-node-core-exact/v1"
    || bundleSourceProfileSchema?.properties?.launch_profile?.const !== "NODE_REPO_SCRIPT_V1"
    || !jsonEqual(bundleSourceProfileSchema?.properties?.adapter_domain?.enum, ["CLI", "HTTP"])
    || !jsonEqual(
      bundleSourceProfileSchema?.properties?.start_profile?.enum,
      ["DIRECT_CHILD_V1", "NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1"],
    )
    || bundleSourceProfileSchema?.properties?.subject_entrypoint?.type !== "string"
    || bundleSourceProfileSchema?.properties?.subject_entrypoint?.pattern
      !== "^[A-Za-z0-9_][A-Za-z0-9._-]*(?:/[A-Za-z0-9_-][A-Za-z0-9._-]*)*\\.(?:js|mjs|cjs)$"
    || bundleSourceProfileSchema?.properties?.subject_entrypoint?.maxLength !== 4096
    || !jsonEqual(bundleSourceProfileSchema?.allOf, expectedSourceProfileConditionals)
    || bundleSourceProfileSchema?.properties?.scope?.const !== "DECLARED_SOURCE_PROFILE_NOT_EXECUTION_EVIDENCE"
    || JSON.stringify(bundleSourceProfileSchema).includes("REPO_EXECUTABLE_V1")
  ) {
    add(
      problems,
      "P07_LAUNCH_AUTHORITY",
      relative(root, bundleSchemaFile),
      "bundle source profile must admit only the antecedent-backed Node repository script launch and exact CLI/HTTP start profiles",
    );
  }

  const exactObjectSchema = (schema, required) => schema?.type === "object"
    && schema?.additionalProperties === false
    && jsonEqual(schema?.required, required)
    && jsonEqual(Object.keys(schema?.properties ?? {}), required);
  const digestRef = (schema) => schema?.$ref === "common.schema.json#/$defs/Digest";
  const localRef = (schema, name) => schema?.$ref === `#/$defs/${name}`;
  const exactTypedRefSchema = (schema, kindSchema) => exactObjectSchema(schema, ["kind", "digest"])
    && jsonEqual(schema?.properties?.kind, kindSchema)
    && digestRef(schema?.properties?.digest);

  const targetResidueSchema = targetSchema?.properties?.terminal_residue_binding;
  const targetTreeSchema = targetSchema?.properties?.tree_binding;
  const targetAttemptSchema = targetSchema?.properties?.attempt_binding;
  const targetBootSchema = targetSchema?.properties?.boot_session_binding;
  const targetRuntimeSchema = targetSchema?.properties?.runtime_binding;
  const executableIdentitySchema = targetRuntimeSchema?.properties?.executable_identity;
  const runEvidenceKindSchema = runSchema?.$defs?.EvidenceKind;
  const runEvidenceRefSchema = runSchema?.$defs?.EvidenceRef;
  const runCapturedRefSchema = runSchema?.$defs?.CapturedObservationRef;
  const runProjectionRefSchema = runSchema?.$defs?.ProjectionResultRef;
  const runSpawnSchema = runSchema?.$defs?.SpawnObservation;
  const runProcessSchema = runSchema?.$defs?.ProcessClosure;
  const runObservationSchema = runSchema?.$defs?.CaptureProjection;
  const runScopeCheckSchema = runSchema?.$defs?.ScopeCheck;
  const runStandaloneSchema = runSchema?.$defs?.StandaloneScope;
  const runPrivateSchema = runSchema?.$defs?.PrivateEvidence;
  const runWitnessSchema = runSchema?.$defs?.ClosedRunWitness;
  const expectedTargetRequired = [
    "schema_version",
    "kind",
    "target_version",
    "publication_scope",
    "contract_bundle_digest",
    "terminal_residue_binding",
    "tree_binding",
    "attempt_binding",
    "boot_session_binding",
    "runtime_binding",
  ];
  const expectedResidueRequired = [
    "profile",
    "study_id",
    "head_revision",
    "head_digest",
    "lineage_root_digest",
    "stage",
    "current_kind",
    "current_digest",
  ];
  const expectedTreeRequired = [
    "authority",
    "object_format",
    "commit_oid",
    "tree_oid",
    "tree_identity_digest",
    "portable_tree_digest",
    "materialization_policy_digest",
    "materialization_manifest_digest",
    "execution_root_scope",
  ];
  const expectedAttemptRequired = [
    "purpose",
    "attempt_artifact_digest",
    "instance_nonce",
    "allocation_profile",
    "marker_ordering",
  ];
  const expectedRuntimeRequired = [
    "authority",
    "name",
    "admitted_executable_path",
    "measured_process_exec_path",
    "version",
    "major",
    "platform",
    "architecture",
    "executable_identity",
    "probe_program_digest",
    "child_resolution",
  ];
  const expectedRunRequired = [
    "schema_version",
    "kind",
    "run_version",
    "publication_scope",
    "contract_execution_target_digest",
    "attempt_artifact_digest",
    "start_claim_ref",
    "closed_run_witness",
  ];
  const expectedExecutionRequired = [
    "schema_version",
    "kind",
    "execution_version",
    "publication_scope",
    "classifier_profile",
    "contract_execution_target_digest",
    "finalized_contract_run_digest",
    "result",
  ];
  const expectedEvidenceKinds = [
    "MATERIALIZATION_REVALIDATION",
    "RUNTIME_REVALIDATION",
    "PROCESS_RESULT",
    "WAIT_RESULT",
    "DRAIN_RESULT",
    "TEARDOWN_RESULT",
    "ORPHAN_CHECK",
    "FINALIZATION_MARKER",
    "CAPTURED_OBSERVATION",
    "PROJECTION_RESULT",
    "TARGET_INVENTORY",
    "CHILD_BINDINGS",
    "IMPORT_RESOLUTION",
    "SERVICE_BINDINGS",
    "NAMED_PARENT_SECRET_SENTINEL_INHERITANCE",
    "PRIVATE_EVIDENCE_MANIFEST",
  ];
  const expectedPrimaryReasons = [
    "NONE",
    "MATERIALIZATION_ERROR",
    "START_ERROR",
    "READINESS_ERROR",
    "PROBE_TRANSPORT_ERROR",
    "TIMEOUT",
    "CANCELLED",
    "OUTPUT_LIMIT",
    "PROJECTION_REJECTED",
  ];
  const expectedScopeDomains = [
    "TARGET_INVENTORY",
    "CHILD_BINDINGS",
    "IMPORT_RESOLUTION",
    "SERVICE_BINDINGS",
    "NAMED_PARENT_SECRET_SENTINEL_INHERITANCE",
  ];
  const expectedScopeViolations = [
    "COUNTERSHAPE_SOURCE_PRESENT",
    "COUNTERSHAPE_DEPENDENCY_PRESENT",
    "COUNTERSHAPE_SOURCE_AND_DEPENDENCY_PRESENT",
    "CHILD_BINDING_REACHES_COUNTERSHAPE",
    "IMPORT_RESOLUTION_REACHES_COUNTERSHAPE",
    "COUNTERSHAPE_SERVICE_BINDING_PRESENT",
    "NAMED_PARENT_SECRET_SENTINEL_INHERITED",
  ];
  const startErrorBranch = runSpawnSchema?.oneOf?.[0];
  const childPIDBranch = runSpawnSchema?.oneOf?.[1];
  const noCaptureBranch = runObservationSchema?.oneOf?.[0];
  const capturedBranch = runObservationSchema?.oneOf?.[1];
  const projectedBranch = runObservationSchema?.oneOf?.[2];
  const cleanScopeBranch = runScopeCheckSchema?.oneOf?.[0];
  const missingScopeBranch = runScopeCheckSchema?.oneOf?.[1];
  const violatedScopeBranch = runScopeCheckSchema?.oneOf?.[2];
  const startClaimSchema = runSchema?.properties?.start_claim_ref;
  const privateManifestSchema = runPrivateSchema?.properties?.manifest_ref;
  const targetAuthorityExact = exactObjectSchema(targetSchema, expectedTargetRequired)
    && targetSchema?.properties?.schema_version?.const === "countershape/v1"
    && targetSchema?.properties?.kind?.const === "ContractExecutionTarget"
    && targetSchema?.properties?.target_version?.const === "contract-execution-target/v1"
    && targetSchema?.properties?.publication_scope?.const === "IMMUTABLE_NONHEAD_PRESPAWN_AUTHORITY_V1"
    && digestRef(targetSchema?.properties?.contract_bundle_digest)
    && exactObjectSchema(targetResidueSchema, expectedResidueRequired)
    && targetResidueSchema?.properties?.profile?.const === "EXACT_TERMINAL_RESIDUE_HEAD_V1"
    && targetResidueSchema?.properties?.study_id?.pattern === "^study:[0-9a-f]{64}$"
    && targetResidueSchema?.properties?.head_revision?.minimum === 1
    && targetResidueSchema?.properties?.head_revision?.maximum === 9007199254740991
    && ["head_digest", "lineage_root_digest", "current_digest"].every(
      (field) => digestRef(targetResidueSchema?.properties?.[field]),
    )
    && targetResidueSchema?.properties?.stage?.const === "RESIDUE"
    && targetResidueSchema?.properties?.current_kind?.const === "ContractBundle"
    && exactObjectSchema(targetTreeSchema, expectedTreeRequired)
    && targetTreeSchema?.properties?.authority?.const === "GIT_PIN_INSPECT_MATERIALIZE_V1"
    && jsonEqual(targetTreeSchema?.properties?.object_format?.enum, ["sha1", "sha256"])
    && targetTreeSchema?.properties?.commit_oid?.pattern === "^[0-9a-f]{40}(?:[0-9a-f]{24})?$"
    && targetTreeSchema?.properties?.tree_oid?.pattern === "^[0-9a-f]{40}(?:[0-9a-f]{24})?$"
    && [
      "tree_identity_digest",
      "portable_tree_digest",
      "materialization_policy_digest",
      "materialization_manifest_digest",
    ].every((field) => digestRef(targetTreeSchema?.properties?.[field]))
    && targetTreeSchema?.properties?.execution_root_scope?.const === "PRIVATE_PINNED_MATERIALIZATION_ONLY_V1"
    && targetTreeSchema?.allOf?.length === 2
    && targetTreeSchema.allOf[0]?.if?.properties?.object_format?.const === "sha1"
    && jsonEqual(targetTreeSchema.allOf[0]?.if?.required, ["object_format"])
    && targetTreeSchema.allOf[0]?.then?.properties?.commit_oid?.pattern === "^[0-9a-f]{40}$"
    && targetTreeSchema.allOf[0]?.then?.properties?.tree_oid?.pattern === "^[0-9a-f]{40}$"
    && targetTreeSchema.allOf[1]?.if?.properties?.object_format?.const === "sha256"
    && jsonEqual(targetTreeSchema.allOf[1]?.if?.required, ["object_format"])
    && targetTreeSchema.allOf[1]?.then?.properties?.commit_oid?.pattern === "^[0-9a-f]{64}$"
    && targetTreeSchema.allOf[1]?.then?.properties?.tree_oid?.pattern === "^[0-9a-f]{64}$"
    && exactObjectSchema(targetAttemptSchema, expectedAttemptRequired)
    && targetAttemptSchema?.properties?.purpose?.const === "CONFORMANCE"
    && digestRef(targetAttemptSchema?.properties?.attempt_artifact_digest)
    && targetAttemptSchema?.properties?.instance_nonce?.pattern === "^[0-9a-f]{64}$"
    && targetAttemptSchema?.properties?.allocation_profile?.const === "PRIVATE_FRESH_ROOT_V1"
    && targetAttemptSchema?.properties?.marker_ordering?.const === "DURABLE_BEFORE_SPAWN"
    && exactObjectSchema(targetBootSchema, ["profile", "identity_digest"])
    && targetBootSchema?.properties?.profile?.const === "DARWIN_KERN_BOOTTIME_V1"
    && digestRef(targetBootSchema?.properties?.identity_digest)
    && exactObjectSchema(targetRuntimeSchema, expectedRuntimeRequired)
    && targetRuntimeSchema?.properties?.authority?.const === "ADMITTED_NODE_PROCESS_EXEC_PATH_V1"
    && targetRuntimeSchema?.properties?.name?.const === "node"
    && ["admitted_executable_path", "measured_process_exec_path"].every((field) => (
      targetRuntimeSchema?.properties?.[field]?.type === "string"
      && targetRuntimeSchema?.properties?.[field]?.minLength === 2
      && targetRuntimeSchema?.properties?.[field]?.maxLength === 4096
      && targetRuntimeSchema?.properties?.[field]?.pattern === "^/[^\\u0000-\\u001f\\u007f]+$"
    ))
    && targetRuntimeSchema?.properties?.version?.maxLength === 128
    && targetRuntimeSchema?.properties?.version?.pattern
      === "^v?[1-9][0-9]*\\.[0-9]+(?:\\.[0-9]+)?(?:[-+][0-9A-Za-z.-]+)?$"
    && targetRuntimeSchema?.properties?.major?.minimum === 1
    && targetRuntimeSchema?.properties?.major?.maximum === 9007199254740991
    && targetRuntimeSchema?.properties?.platform?.const === "darwin"
    && targetRuntimeSchema?.properties?.architecture?.const === "arm64"
    && exactObjectSchema(executableIdentitySchema, ["bytes_digest", "mode", "byte_count"])
    && digestRef(executableIdentitySchema?.properties?.bytes_digest)
    && executableIdentitySchema?.properties?.mode?.pattern === "^100(?:[1357][0-7]{2}|[0-7][1357][0-7]|[0-7]{2}[1357])$"
    && executableIdentitySchema?.properties?.byte_count?.minimum === 1
    && executableIdentitySchema?.properties?.byte_count?.maximum === 9007199254740991
    && digestRef(targetRuntimeSchema?.properties?.probe_program_digest)
    && targetRuntimeSchema?.properties?.child_resolution?.const === "PROCESS_EXEC_PATH_EQUALS_ADMITTED_RUNTIME_V1";

  const runAuthorityExact = exactObjectSchema(runSchema, expectedRunRequired)
    && runSchema?.properties?.schema_version?.const === "countershape/v1"
    && runSchema?.properties?.kind?.const === "FinalizedContractRun"
    && runSchema?.properties?.run_version?.const === "finalized-contract-run/v1"
    && runSchema?.properties?.publication_scope?.const === "IMMUTABLE_NONHEAD_FINALIZED_RUN_V1"
    && digestRef(runSchema?.properties?.contract_execution_target_digest)
    && digestRef(runSchema?.properties?.attempt_artifact_digest)
    && exactTypedRefSchema(startClaimSchema, { const: "StartClaim" })
    && localRef(runSchema?.properties?.closed_run_witness, "ClosedRunWitness")
    && jsonEqual(runEvidenceKindSchema?.enum, expectedEvidenceKinds)
    && exactTypedRefSchema(runEvidenceRefSchema, { $ref: "#/$defs/EvidenceKind" })
    && exactTypedRefSchema(runCapturedRefSchema, { const: "CAPTURED_OBSERVATION" })
    && exactTypedRefSchema(runProjectionRefSchema, { const: "PROJECTION_RESULT" })
    && runSpawnSchema?.oneOf?.length === 2
    && exactObjectSchema(startErrorBranch, ["status", "error_code"])
    && startErrorBranch?.properties?.status?.const === "START_ERROR"
    && jsonEqual(startErrorBranch?.properties?.error_code?.enum, [
      "OS_START_ERROR",
      "PRESPAWN_MATERIALIZATION_REVALIDATION_FAILED",
      "PRESPAWN_RUNTIME_REVALIDATION_FAILED",
    ])
    && exactObjectSchema(childPIDBranch, ["status", "pid"])
    && childPIDBranch?.properties?.status?.const === "CHILD_PID_OBSERVED"
    && childPIDBranch?.properties?.pid?.minimum === 1
    && childPIDBranch?.properties?.pid?.maximum === 2147483647
    && exactObjectSchema(runProcessSchema, ["status", "primary_reason", "cleanup_controls", "evidence_refs"])
    && jsonEqual(runProcessSchema?.properties?.status?.enum, ["PROCESS_CLEAN", "PROCESS_CONTROLLED"])
    && jsonEqual(runProcessSchema?.properties?.primary_reason?.enum, expectedPrimaryReasons)
    && runProcessSchema?.properties?.cleanup_controls?.type === "array"
    && runProcessSchema?.properties?.cleanup_controls?.maxItems === 2
    && runProcessSchema?.properties?.cleanup_controls?.uniqueItems === true
    && jsonEqual(runProcessSchema?.properties?.cleanup_controls?.items?.enum, ["TEARDOWN_ERROR", "ORPHAN_RISK"])
    && runProcessSchema?.properties?.evidence_refs?.type === "array"
    && runProcessSchema?.properties?.evidence_refs?.minItems === 5
    && runProcessSchema?.properties?.evidence_refs?.maxItems === 8
    && runProcessSchema?.properties?.evidence_refs?.uniqueItems === true
    && localRef(runProcessSchema?.properties?.evidence_refs?.items, "EvidenceRef")
    && runObservationSchema?.oneOf?.length === 3
    && exactObjectSchema(noCaptureBranch, ["status"])
    && noCaptureBranch?.properties?.status?.const === "NO_CAPTURE"
    && exactObjectSchema(capturedBranch, ["status", "captured_observation_ref"])
    && capturedBranch?.properties?.status?.const === "CAPTURED_UNPROJECTED"
    && localRef(capturedBranch?.properties?.captured_observation_ref, "CapturedObservationRef")
    && exactObjectSchema(projectedBranch, ["status", "captured_observation_ref", "projection_result_ref", "observed_tuple"])
    && projectedBranch?.properties?.status?.const === "PROJECTED"
    && localRef(projectedBranch?.properties?.captured_observation_ref, "CapturedObservationRef")
    && localRef(projectedBranch?.properties?.projection_result_ref, "ProjectionResultRef")
    && projectedBranch?.properties?.observed_tuple?.$ref === "common.schema.json#/$defs/ExactTuple"
    && runScopeCheckSchema?.oneOf?.length === 3
    && exactObjectSchema(cleanScopeBranch, ["domain", "status", "evidence_ref"])
    && jsonEqual(cleanScopeBranch?.properties?.domain?.enum, expectedScopeDomains)
    && cleanScopeBranch?.properties?.status?.const === "PRESENT_CLEAN"
    && localRef(cleanScopeBranch?.properties?.evidence_ref, "EvidenceRef")
    && exactObjectSchema(missingScopeBranch, ["domain", "status"])
    && jsonEqual(missingScopeBranch?.properties?.domain?.enum, expectedScopeDomains)
    && missingScopeBranch?.properties?.status?.const === "MISSING"
    && exactObjectSchema(violatedScopeBranch, ["domain", "status", "evidence_ref", "violation"])
    && jsonEqual(violatedScopeBranch?.properties?.domain?.enum, expectedScopeDomains)
    && violatedScopeBranch?.properties?.status?.const === "PRESENT_VIOLATION"
    && localRef(violatedScopeBranch?.properties?.evidence_ref, "EvidenceRef")
    && jsonEqual(violatedScopeBranch?.properties?.violation?.enum, expectedScopeViolations)
    && exactObjectSchema(runStandaloneSchema, ["status", "checks"])
    && jsonEqual(runStandaloneSchema?.properties?.status?.enum, ["COMPLETE", "PARTIAL", "VIOLATED"])
    && runStandaloneSchema?.properties?.checks?.type === "array"
    && runStandaloneSchema?.properties?.checks?.minItems === 5
    && runStandaloneSchema?.properties?.checks?.maxItems === 5
    && runStandaloneSchema?.properties?.checks?.uniqueItems === true
    && localRef(runStandaloneSchema?.properties?.checks?.items, "ScopeCheck")
    && exactObjectSchema(runPrivateSchema, [
      "profile",
      "manifest_ref",
      "blob_count",
      "aggregate_byte_count",
      "retention_at_finalization",
      "default_export",
    ])
    && runPrivateSchema?.properties?.profile?.const === "PRIVATE_EVIDENCE_MANIFEST_V1"
    && exactTypedRefSchema(privateManifestSchema, { const: "PRIVATE_EVIDENCE_MANIFEST" })
    && runPrivateSchema?.properties?.blob_count?.minimum === 0
    && runPrivateSchema?.properties?.blob_count?.maximum === 16
    && runPrivateSchema?.properties?.aggregate_byte_count?.minimum === 0
    && runPrivateSchema?.properties?.aggregate_byte_count?.maximum === 67108864
    && runPrivateSchema?.properties?.retention_at_finalization?.const === "RETAINED_AT_FINALIZATION"
    && runPrivateSchema?.properties?.default_export?.const === "OMITTED"
    && exactObjectSchema(runWitnessSchema, [
      "witness_version",
      "spawn_observation",
      "process_closure",
      "observation",
      "standalone_scope",
      "private_evidence",
    ])
    && runWitnessSchema?.properties?.witness_version?.const === "closed-run-witness/v1"
    && localRef(runWitnessSchema?.properties?.spawn_observation, "SpawnObservation")
    && localRef(runWitnessSchema?.properties?.process_closure, "ProcessClosure")
    && localRef(runWitnessSchema?.properties?.observation, "CaptureProjection")
    && localRef(runWitnessSchema?.properties?.standalone_scope, "StandaloneScope")
    && localRef(runWitnessSchema?.properties?.private_evidence, "PrivateEvidence");

  const executionAuthorityExact = exactObjectSchema(executionSchema, expectedExecutionRequired)
    && executionSchema?.properties?.schema_version?.const === "countershape/v1"
    && executionSchema?.properties?.kind?.const === "ContractExecution"
    && executionSchema?.properties?.execution_version?.const === "contract-execution/v1"
    && executionSchema?.properties?.publication_scope?.const === "IMMUTABLE_NONHEAD_CLASSIFICATION_V1"
    && executionSchema?.properties?.classifier_profile?.const === "CONTRACT_EXECUTION_EXACT_TUPLE_V1"
    && digestRef(executionSchema?.properties?.contract_execution_target_digest)
    && digestRef(executionSchema?.properties?.finalized_contract_run_digest)
    && jsonEqual(executionSchema?.properties?.result?.enum, ["CONFORMS", "CONTRADICTS", "INELIGIBLE_EXECUTION"]);

  if (!targetAuthorityExact || !runAuthorityExact || !executionAuthorityExact) {
    add(
      problems,
      "P07_EXECUTION_AUTHORITY",
      relative(root, targetSchemaFile),
      "C1 target, finalized-run witness, and derived execution schemas must retain their exact closed inert wire authorities",
    );
  }
  const exactTupleAuthorities = [
    { schema: commonSchema?.$defs?.ExactTuple, file: commonSchemaFile },
    { schema: decisionSchema?.$defs?.ExactTuple, file: decisionSchemaFile },
  ];
  for (const { schema: exactTupleSchema, file } of exactTupleAuthorities) {
    if (
      exactTupleSchema?.type !== "object"
      || exactTupleSchema?.additionalProperties !== false
      || !jsonEqual(exactTupleSchema?.required, ["fields"])
      || !jsonEqual(Object.keys(exactTupleSchema?.properties ?? {}), ["fields"])
      || exactTupleSchema?.properties?.fields?.type !== "array"
      || exactTupleSchema?.properties?.fields?.minItems !== 1
      || exactTupleSchema?.properties?.fields?.items?.$ref !== "#/$defs/ExactField"
      || Object.hasOwn(exactTupleSchema?.properties ?? {}, "tuple_digest")
    ) {
      add(
        problems,
        "P07_TUPLE_AUTHORITY",
        relative(root, file),
        "shared and DecisionRecord ExactTuple definitions must carry only one nonempty fields array; no caller-supplied tuple digest is authority",
      );
    }
  }

  const receiptNonclaimSchema = bundleSchema?.properties?.determinism_profile;
  if (
    receiptNonclaimSchema?.properties?.scope?.const
      !== "EMITTER_INVENTED_STRUCTURAL_FACTS_EXCLUDING_AUTHORIZED_INPUT_CONTENT"
    || bundle?.determinism_profile?.scope
      !== "EMITTER_INVENTED_STRUCTURAL_FACTS_EXCLUDING_AUTHORIZED_INPUT_CONTENT"
    || !receiptNonclaimSchema?.required?.includes("emitter_introduces_execution_receipt")
    || receiptNonclaimSchema?.properties?.emitter_introduces_execution_receipt?.const !== false
    || bundle?.determinism_profile?.emitter_introduces_execution_receipt !== false
  ) {
    add(
      problems,
      "P07_RECEIPT_CYCLE",
      relative(root, bundleSchemaFile),
      "bundle determinism profile must scope emitter-invented structural facts outside authorized input content and retain the exact false emitter_introduces_execution_receipt nonclaim",
    );
  }

  for (const [file, schema] of [
    [bundleSchemaFile, bundleSchema],
    [targetSchemaFile, targetSchema],
    [runSchemaFile, runSchema],
    [executionSchemaFile, executionSchema],
  ]) {
    if (!schema || typeof schema !== "object" || Array.isArray(schema)) continue;
    if (schema.required?.includes("artifact_digest") || Object.hasOwn(schema.properties ?? {}, "artifact_digest")) {
      add(
        problems,
        "P07_SELF_DIGEST_FORBIDDEN",
        relative(root, file),
        "canonical ContractBundle, ContractExecutionTarget, FinalizedContractRun, and ContractExecution bodies must not declare their own artifact_digest",
      );
    }
    if (p07SchemaReachesReceipt(file, schema, schemasByFile)) {
      add(
        problems,
        "P07_RECEIPT_CYCLE",
        relative(root, file),
        "contract schema reaches receipt authority directly or through a transitive schema alias",
      );
    }
  }

  for (const [file, value] of [
    [bundleSchemaFile, bundleSchema],
    [targetSchemaFile, targetSchema],
    [runSchemaFile, runSchema],
    [executionSchemaFile, executionSchema],
    [bundleExampleFile, bundle],
    [targetExampleFile, target],
    [runExampleFile, finalizedRun],
    [executionExampleFile, execution],
  ]) {
    for (const finding of collectP07ReceiptAuthority(value)) {
      add(problems, "P07_RECEIPT_CYCLE", relative(root, file), `contract source/evidence body contains forbidden receipt authority: ${finding}`);
    }
  }

  if (bundle && typeof bundle === "object" && !Array.isArray(bundle)) {
    if (Object.hasOwn(bundle, "artifact_digest")) {
      add(problems, "P07_SELF_DIGEST_FORBIDDEN", relative(root, bundleExampleFile), "bundle example contains a self digest");
    }
    if (bundle.portable_profile_digest !== bundle.predicate?.portable_profile_digest) {
      add(problems, "P07_PROFILE_BINDING", relative(root, bundleExampleFile), "bundle and predicate portable profile digests must match exactly");
    }
    if (Buffer.byteLength(JSON.stringify(bundle), "utf8") > 1024 * 1024) {
      add(problems, "P07_BUNDLE_RESOURCE", relative(root, bundleExampleFile), "bundle canonical example exceeds the 1 MiB durable body ceiling");
    }

    const files = Array.isArray(bundle.files) ? bundle.files : [];
    const actualRoster = files.map((entry) => entry?.path);
    if (!jsonEqual(actualRoster, P07_FILE_ROSTER)) {
      add(
        problems,
        "P07_FILE_ROSTER",
        relative(root, bundleExampleFile),
        `bundle files must equal the exact sorted six-path roster: ${P07_FILE_ROSTER.join(", ")}`,
      );
    }
    let totalRaw = 0;
    const decodedByPath = new Map();
    const parsedJSONByPath = new Map();
    for (const entry of files) {
      if (!entry || typeof entry !== "object" || Array.isArray(entry) || typeof entry.path !== "string") continue;
      if (typeof entry.content_base64 !== "string") {
        add(problems, "P07_BUNDLE_RECOVERY", relative(root, bundleExampleFile), `${entry.path} omits exact recoverable content_base64`);
        continue;
      }
      const decoded = Buffer.from(entry.content_base64, "base64");
      totalRaw += decoded.length;
      decodedByPath.set(entry.path, decoded);
      if (decoded.toString("base64") !== entry.content_base64) {
        add(problems, "P07_FILE_BYTES", relative(root, bundleExampleFile), `${entry.path} content_base64 is not canonical padded base64`);
      }
      if (entry.byte_count !== decoded.length) {
        add(problems, "P07_FILE_BYTES", relative(root, bundleExampleFile), `${entry.path} byte_count does not equal decoded content length`);
      }
      if (entry.byte_sha256 !== rawSha256(decoded)) {
        add(problems, "P07_FILE_BYTES", relative(root, bundleExampleFile), `${entry.path} byte_sha256 does not equal decoded content bytes`);
      }
      if (
        !Buffer.from(decoded.toString("utf8"), "utf8").equals(decoded)
        || decoded.includes(0)
        || decoded.includes(13)
        || decoded.length === 0
        || decoded[decoded.length - 1] !== 10
      ) {
        add(
          problems,
          "P07_FILE_TEXT",
          relative(root, bundleExampleFile),
          `${entry.path} must be exact UTF-8 text with no NUL/CR bytes and a final LF`,
        );
      }
    }
    for (const filePath of P07_JSON_FILE_ROSTER) {
      const bytes = decodedByPath.get(filePath);
      if (!bytes || !Buffer.from(bytes.toString("utf8"), "utf8").equals(bytes)) continue;
      try {
        const parsed = strictJsonParse(bytes.toString("utf8"), `embedded ${filePath}`);
        if (parsed.duplicates.length > 0) {
          add(problems, "P07_FILE_JSON", relative(root, bundleExampleFile), `${filePath} contains duplicate JSON member names`);
        }
        parsedJSONByPath.set(filePath, parsed.value);
        const canonicalEnvelope = Buffer.from(`${p07CanonicalJSON(parsed.value)}\n`, "utf8");
        if (!bytes.equals(canonicalEnvelope)) {
          add(
            problems,
            "P07_FILE_JSON",
            relative(root, bundleExampleFile),
            `${filePath} must be canonical JSON followed by exactly one LF`,
          );
        }
        for (const finding of collectP07ReceiptAuthority(parsed.value)) {
          add(
            problems,
            "P07_RECEIPT_CYCLE",
            relative(root, bundleExampleFile),
            `${filePath} contains forbidden receipt authority: ${finding}`,
          );
        }
      } catch (error) {
        add(
          problems,
          "P07_FILE_JSON",
          relative(root, bundleExampleFile),
          `${filePath} is not strict JSON: ${error instanceof Error ? error.message : String(error)}`,
        );
      }
    }
    const compiledDecision = parsedJSONByPath.get("decision.json");
    const portableFixture = parsedJSONByPath.get("fixture.json");
    const expectedCompiledPredicate = {
      ...bundle.predicate,
    };
    if (
      !jsonEqual(Object.keys(compiledDecision ?? {}), ["decision_action", "kind", "predicate", "schema_version"])
      || compiledDecision?.schema_version !== "countershape-contract/v1"
      || compiledDecision?.kind !== "CompiledDecision"
      || compiledDecision?.decision_action !== bundle.decision_action
      || !jsonEqual(compiledDecision?.predicate, expectedCompiledPredicate)
    ) {
      add(
        problems,
        "P07_FILE_ROLE",
        relative(root, bundleExampleFile),
        "decision.json must be the exact compiled decision action and predicate represented by the outer bundle",
      );
    }
    let portableFixtureValid = false;
    if (
      jsonEqual(Object.keys(portableFixture ?? {}), [
        "fixture_version", "kind", "portable_source_base64", "portable_source_digest", "schema_version",
      ])
      && portableFixture?.schema_version === "countershape-contract/v1"
      && portableFixture?.kind === "PortableFixture"
      && portableFixture?.fixture_version === "portable-source-fixture/v1"
      && portableFixture?.portable_source_digest === bundle.portable_source_digest
      && typeof portableFixture?.portable_source_base64 === "string"
    ) {
      const sourceBytes = Buffer.from(portableFixture.portable_source_base64, "base64");
      if (sourceBytes.toString("base64") === portableFixture.portable_source_base64) {
        try {
          const source = strictJsonParse(sourceBytes.toString("utf8"), "embedded PortableSource");
          portableFixtureValid = source.duplicates.length === 0
            && sourceBytes.equals(Buffer.from(p07CanonicalJSON(source.value), "utf8"))
            && p07TypedDigest("PortableSource", source.value) === portableFixture.portable_source_digest;
        } catch {
          portableFixtureValid = false;
        }
      }
    }
    if (!portableFixtureValid) {
      add(
        problems,
        "P07_FILE_ROLE",
        relative(root, bundleExampleFile),
        "fixture.json must recover the exact canonical typed PortableSource represented by the outer bundle",
      );
    }
    const contractText = decodedByPath.get("contract.test.mjs")?.toString("utf8") ?? "";
    const harnessText = decodedByPath.get("harness.mjs")?.toString("utf8") ?? "";
    const readmeText = decodedByPath.get("README.md")?.toString("utf8") ?? "";
    const verificationIndex = contractText.indexOf("for (const entry of manifest.files)");
    const dynamicHarnessIndex = contractText.indexOf("await import(\"./harness.mjs\")");
    const runtimeSourceBytesExact = P07_FILE_ROSTER.every((filePath) => (
      decodedByPath.has(filePath)
      && rawSha256(decodedByPath.get(filePath)) === P07_RUNTIME_SOURCE_SHA256[filePath]
    ));
    if (
      !runtimeSourceBytesExact
      || verificationIndex < 0
      || dynamicHarnessIndex <= verificationIndex
      || /from\s+["']\.\/harness\.mjs["']/u.test(contractText)
      || !contractText.includes("node:test")
      || !contractText.includes("harness.runContract({")
      || !harnessText.includes("spawn(process.execPath")
      || !harnessText.includes("contract.source.entrypoint")
      || !harnessText.includes("export async function runContract")
      || !readmeText.includes("WARNING: FULL USER AUTHORITY AND NETWORK ACCESS")
      || !readmeText.includes("this bundle does not provide a safe preparation command")
      || !readmeText.includes("Who created or approved the bundle")
      || !readmeText.includes("Authenticity, Git identity, or currentness")
    ) {
      add(
        problems,
        "P07_FILE_ROLE",
        relative(root, bundleExampleFile),
        "six-file example must retain the reviewed A2.2 runtime assets and verify companions before dynamic harness load",
      );
    }
    if (totalRaw > 640 * 1024) {
      add(problems, "P07_BUNDLE_RESOURCE", relative(root, bundleExampleFile), "bundle example exceeds the 640 KiB aggregate raw-file ceiling");
    }

    const manifestBytes = decodedByPath.get("manifest.json");
    if (manifestBytes) {
      if (!Buffer.from(manifestBytes.toString("utf8"), "utf8").equals(manifestBytes)) {
        add(problems, "P07_MANIFEST", relative(root, bundleExampleFile), "manifest bytes are not exact UTF-8");
      } else {
        try {
          const parsed = strictJsonParse(manifestBytes.toString("utf8"), "embedded manifest.json");
          if (parsed.duplicates.length > 0) {
            add(problems, "P07_MANIFEST", relative(root, bundleExampleFile), "manifest contains duplicate JSON member names");
          }
          const expectedEntries = files
            .filter((entry) => entry?.path !== "manifest.json")
            .map(({ path: filePath, mode, byte_count, byte_sha256 }) => ({ path: filePath, mode, byte_count, byte_sha256 }));
          const expectedManifest = {
            schema_version: "countershape-contract/v1",
            kind: "IntegrityManifest",
            manifest_version: "countershape-manifest/v1",
            files: expectedEntries,
          };
          if (!jsonEqual(parsed.value, expectedManifest)) {
            add(problems, "P07_MANIFEST", relative(root, bundleExampleFile), "manifest must cover exactly the other five bundle files and exclude itself");
          }
        } catch (error) {
          add(problems, "P07_MANIFEST", relative(root, bundleExampleFile), `manifest is not strict JSON: ${error instanceof Error ? error.message : String(error)}`);
        }
      }
    }

    const selectedFields = bundle.predicate?.selected_fields;
    if (Array.isArray(selectedFields)) {
      const allowedTuples = Array.isArray(bundle.predicate?.allowed_tuples) ? bundle.predicate.allowed_tuples : [];
      allowedTuples.forEach((tuple, index) => {
        const fieldIDs = exactTupleFieldIDs(tuple);
        if (!jsonEqual(fieldIDs, selectedFields) || new Set(fieldIDs ?? []).size !== (fieldIDs?.length ?? 0)) {
          add(
            problems,
            "P07_TUPLE_FIELDS",
            relative(root, bundleExampleFile),
            `allowed tuple ${index} fields must equal the selected-field roster exactly once in canonical order`,
          );
        }
      });
    }
  }

  if (target && typeof target === "object" && !Array.isArray(target)) {
    if (Object.hasOwn(target, "artifact_digest")) {
      add(problems, "P07_SELF_DIGEST_FORBIDDEN", relative(root, targetExampleFile), "execution-target example contains a self digest");
    }
    const pinned = target.tree_binding;
    const oidWidth = pinned?.object_format === "sha1" ? 40 : pinned?.object_format === "sha256" ? 64 : 0;
    const pinnedIdentity = {
      schema_version: "countershape/v1",
      kind: "PinnedTreeIdentity",
      object_format: pinned?.object_format,
      commit_oid: pinned?.commit_oid,
      tree_oid: pinned?.tree_oid,
    };
    if (
      oidWidth === 0
      || typeof pinned?.commit_oid !== "string"
      || typeof pinned?.tree_oid !== "string"
      || pinned.commit_oid.length !== oidWidth
      || pinned.tree_oid.length !== oidWidth
      || !/^[0-9a-f]+$/u.test(pinned.commit_oid)
      || !/^[0-9a-f]+$/u.test(pinned.tree_oid)
      || pinned.tree_identity_digest !== p07TypedDigest("PinnedTreeIdentity", pinnedIdentity)
    ) {
      add(
        problems,
        "P07_TARGET_TREE_BINDING",
        relative(root, targetExampleFile),
        "ContractExecutionTarget must carry exact-width pinned Git OIDs and their recomputed PinnedTreeIdentity digest",
      );
    }
    if (
      !bundle
      || target.contract_bundle_digest !== p07TypedDigest("ContractBundle", bundle)
      || target.terminal_residue_binding?.profile !== "EXACT_TERMINAL_RESIDUE_HEAD_V1"
      || target.terminal_residue_binding?.stage !== "RESIDUE"
      || target.terminal_residue_binding?.current_kind !== "ContractBundle"
      || target.terminal_residue_binding?.current_digest !== target.contract_bundle_digest
      || typeof target.terminal_residue_binding?.study_id !== "string"
      || !/^study:[0-9a-f]{64}$/u.test(target.terminal_residue_binding.study_id)
      || !Number.isSafeInteger(target.terminal_residue_binding?.head_revision)
      || target.terminal_residue_binding.head_revision < 1
      || !/^sha256:[0-9a-f]{64}$/u.test(target.terminal_residue_binding?.head_digest ?? "")
      || !/^sha256:[0-9a-f]{64}$/u.test(target.terminal_residue_binding?.lineage_root_digest ?? "")
    ) {
      add(
        problems,
        "P07_TARGET_SOURCE_BINDING",
        relative(root, targetExampleFile),
        "ContractExecutionTarget must bind the exact reopened bundle and exact terminal residue head without a second caller-authored source authority",
      );
    }
    const sourceProfile = bundle?.source_profile;
    if (
      sourceProfile?.launch_profile !== "NODE_REPO_SCRIPT_V1"
      || (sourceProfile?.adapter_domain === "CLI" && sourceProfile.start_profile !== "DIRECT_CHILD_V1")
      || (sourceProfile?.adapter_domain === "HTTP" && sourceProfile.start_profile !== "NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1")
      || !["CLI", "HTTP"].includes(sourceProfile?.adapter_domain)
    ) {
      add(
        problems,
        "P07_LAUNCH_AUTHORITY",
        relative(root, bundleExampleFile),
        "source profile must use the antecedent-backed Node script launch and its adapter-specific start authority",
      );
    }
    const runtime = target.runtime_binding;
    const runtimeVersion = /^v?([1-9][0-9]*)\./u.exec(runtime?.version ?? "");
    const validRuntimePath = (value) => {
      if (typeof value !== "string" || Buffer.byteLength(value, "utf8") < 2 || Buffer.byteLength(value, "utf8") > 4096) {
        return false;
      }
      if (!value.startsWith("/") || /[\u0000-\u001f\u007f]/u.test(value) || value.includes("//")) return false;
      return value.split("/").slice(1).every((segment) => segment !== "" && segment !== "." && segment !== "..");
    };
    const executableMode = runtime?.executable_identity?.mode;
    const executablePermission = typeof executableMode === "string" && /^100[0-7]{3}$/u.test(executableMode)
      ? Number.parseInt(executableMode.slice(-3), 8)
      : 0;
    if (
      target.attempt_binding?.purpose !== "CONFORMANCE"
      || target.attempt_binding?.marker_ordering !== "DURABLE_BEFORE_SPAWN"
      || target.attempt_binding?.allocation_profile !== "PRIVATE_FRESH_ROOT_V1"
      || !/^[0-9a-f]{64}$/u.test(target.attempt_binding?.instance_nonce ?? "")
      || !/^sha256:[0-9a-f]{64}$/u.test(target.attempt_binding?.attempt_artifact_digest ?? "")
      || target.boot_session_binding?.profile !== "DARWIN_KERN_BOOTTIME_V1"
      || !/^sha256:[0-9a-f]{64}$/u.test(target.boot_session_binding?.identity_digest ?? "")
      || !runtimeVersion
      || Number(runtimeVersion[1]) !== runtime?.major
      || runtime?.authority !== "ADMITTED_NODE_PROCESS_EXEC_PATH_V1"
      || runtime?.name !== "node"
      || runtime?.platform !== "darwin"
      || runtime?.architecture !== "arm64"
      || runtime?.child_resolution !== "PROCESS_EXEC_PATH_EQUALS_ADMITTED_RUNTIME_V1"
      || !validRuntimePath(runtime?.admitted_executable_path)
      || runtime?.measured_process_exec_path !== runtime?.admitted_executable_path
      || !/^sha256:[0-9a-f]{64}$/u.test(runtime?.executable_identity?.bytes_digest ?? "")
      || !Number.isSafeInteger(runtime?.executable_identity?.byte_count)
      || runtime.executable_identity.byte_count < 1
      || executablePermission === 0
      || (executablePermission & 0o111) === 0
      || !/^sha256:[0-9a-f]{64}$/u.test(runtime?.probe_program_digest ?? "")
    ) {
      add(
        problems,
        "P07_TARGET_RUNTIME_BINDING",
        relative(root, targetExampleFile),
        "ContractExecutionTarget must bind a fresh durable conformance attempt and an exact measured/revalidated Node runtime",
      );
    }
  }

  if (finalizedRun && typeof finalizedRun === "object" && !Array.isArray(finalizedRun)) {
    if (Object.hasOwn(finalizedRun, "artifact_digest")) {
      add(problems, "P07_SELF_DIGEST_FORBIDDEN", relative(root, runExampleFile), "finalized-run example contains a self digest");
    }
    if (
      !target
      || finalizedRun.contract_execution_target_digest !== p07TypedDigest("ContractExecutionTarget", target)
      || finalizedRun.attempt_artifact_digest !== target.attempt_binding?.attempt_artifact_digest
      || finalizedRun.start_claim_ref?.kind !== "StartClaim"
      || !/^sha256:[0-9a-f]{64}$/u.test(finalizedRun.start_claim_ref?.digest ?? "")
    ) {
      add(
        problems,
        "P07_TARGET_RUN_BINDING",
        relative(root, runExampleFile),
        "FinalizedContractRun must bind the exact paired target and that target's exact fresh attempt before classification",
      );
    }
    const witness = finalizedRun.closed_run_witness;
    const spawn = witness?.spawn_observation;
    const process = witness?.process_closure;
    const observation = witness?.observation;
    const standalone = witness?.standalone_scope;
    const privateEvidence = witness?.private_evidence;
    const cleanup = process?.cleanup_controls;
    const processEvidence = process?.evidence_refs;
    const expectedProcessEvidence = spawn?.status === "START_ERROR"
      ? [
        "MATERIALIZATION_REVALIDATION",
        "RUNTIME_REVALIDATION",
        "TEARDOWN_RESULT",
        "ORPHAN_CHECK",
        "FINALIZATION_MARKER",
      ]
      : [
        "MATERIALIZATION_REVALIDATION",
        "RUNTIME_REVALIDATION",
        "PROCESS_RESULT",
        "WAIT_RESULT",
        "DRAIN_RESULT",
        "TEARDOWN_RESULT",
        "ORPHAN_CHECK",
        "FINALIZATION_MARKER",
      ];
    const expectedViolationByDomain = Object.freeze({
      TARGET_INVENTORY: [
        "COUNTERSHAPE_SOURCE_PRESENT",
        "COUNTERSHAPE_DEPENDENCY_PRESENT",
        "COUNTERSHAPE_SOURCE_AND_DEPENDENCY_PRESENT",
      ],
      CHILD_BINDINGS: ["CHILD_BINDING_REACHES_COUNTERSHAPE"],
      IMPORT_RESOLUTION: ["IMPORT_RESOLUTION_REACHES_COUNTERSHAPE"],
      SERVICE_BINDINGS: ["COUNTERSHAPE_SERVICE_BINDING_PRESENT"],
      NAMED_PARENT_SECRET_SENTINEL_INHERITANCE: ["NAMED_PARENT_SECRET_SENTINEL_INHERITED"],
    });
    const checks = standalone?.checks;
    const derivedScope = Array.isArray(checks) && checks.some((check) => check?.status === "PRESENT_VIOLATION")
      ? "VIOLATED"
      : Array.isArray(checks) && checks.some((check) => check?.status === "MISSING")
        ? "PARTIAL"
        : "COMPLETE";
    const scopeChecksValid = Array.isArray(checks)
      && checks.length === expectedScopeDomains.length
      && checks.every((check, index) => {
        const expectedDomain = expectedScopeDomains[index];
        if (check?.domain !== expectedDomain) return false;
        if (check.status === "MISSING") {
          return !Object.hasOwn(check, "evidence_ref") && !Object.hasOwn(check, "violation");
        }
        if (check?.evidence_ref?.kind !== expectedDomain || !/^sha256:[0-9a-f]{64}$/u.test(check.evidence_ref?.digest ?? "")) {
          return false;
        }
        if (check.status === "PRESENT_CLEAN") return !Object.hasOwn(check, "violation");
        return check.status === "PRESENT_VIOLATION"
          && expectedViolationByDomain[expectedDomain].includes(check.violation);
      });
    const witnessRefs = [];
    walk(witness, (node) => {
      if (
        node
        && typeof node === "object"
        && !Array.isArray(node)
        && Object.keys(node).length === 2
        && Object.hasOwn(node, "digest")
        && Object.hasOwn(node, "kind")
      ) {
        witnessRefs.push(node);
      }
    });
    const refKinds = witnessRefs.map((ref) => ref.kind);
    const referencesValid = witnessRefs.length <= 16
      && witnessRefs.length === new Set(refKinds).size
      && witnessRefs.every((ref) => (
        expectedEvidenceKinds.includes(ref.kind) && /^sha256:[0-9a-f]{64}$/u.test(ref.digest)
      ));
    const processIsClean = process?.status === "PROCESS_CLEAN";
    const processIsControlled = process?.status === "PROCESS_CONTROLLED";
    const primary = process?.primary_reason;
    const primaryIsClosed = expectedPrimaryReasons.includes(primary);
    const cleanupIsClosed = Array.isArray(cleanup)
      && jsonEqual(cleanup, ["TEARDOWN_ERROR", "ORPHAN_RISK"].filter((control) => cleanup.includes(control)));
    const processStateValid = primaryIsClosed
      && cleanupIsClosed
      && ((processIsClean && primary === "NONE" && cleanup.length === 0)
        || (processIsControlled && (primary !== "NONE" || cleanup.length > 0)));
    const spawnRelationValid = (
      spawn?.status === "START_ERROR"
      && [
        "OS_START_ERROR",
        "PRESPAWN_MATERIALIZATION_REVALIDATION_FAILED",
        "PRESPAWN_RUNTIME_REVALIDATION_FAILED",
      ].includes(spawn?.error_code)
      && processIsControlled
      && primary === "START_ERROR"
      && observation?.status === "NO_CAPTURE"
      && derivedScope !== "COMPLETE"
    ) || (
      spawn?.status === "CHILD_PID_OBSERVED"
      && Number.isInteger(spawn?.pid)
      && spawn.pid >= 1
      && spawn.pid <= 2147483647
      && primary !== "START_ERROR"
    );
    const observationValid = observation?.status === "PROJECTED"
      ? observation?.captured_observation_ref?.kind === "CAPTURED_OBSERVATION"
        && observation?.projection_result_ref?.kind === "PROJECTION_RESULT"
        && Array.isArray(observation?.observed_tuple?.fields)
      : observation?.status === "CAPTURED_UNPROJECTED"
        ? observation?.captured_observation_ref?.kind === "CAPTURED_OBSERVATION" && primary !== "NONE"
        : observation?.status === "NO_CAPTURE" && primary !== "NONE";
    const processEvidenceValid = Array.isArray(processEvidence)
      && jsonEqual(processEvidence.map((ref) => ref?.kind), expectedProcessEvidence);
    const privateEvidenceValid = privateEvidence?.profile === "PRIVATE_EVIDENCE_MANIFEST_V1"
      && privateEvidence?.manifest_ref?.kind === "PRIVATE_EVIDENCE_MANIFEST"
      && Number.isInteger(privateEvidence?.blob_count)
      && privateEvidence.blob_count >= 0
      && privateEvidence.blob_count <= 16
      && Number.isSafeInteger(privateEvidence?.aggregate_byte_count)
      && privateEvidence.aggregate_byte_count >= 0
      && privateEvidence.aggregate_byte_count <= 67108864
      && ((privateEvidence.blob_count === 0 && privateEvidence.aggregate_byte_count === 0)
        || (privateEvidence.blob_count > 0 && privateEvidence.aggregate_byte_count >= privateEvidence.blob_count))
      && privateEvidence?.retention_at_finalization === "RETAINED_AT_FINALIZATION"
      && privateEvidence?.default_export === "OMITTED";
    const witnessValid = witness?.witness_version === "closed-run-witness/v1"
      && Buffer.byteLength(p07CanonicalJSON(witness), "utf8") <= 262144
      && processStateValid
      && spawnRelationValid
      && processEvidenceValid
      && observationValid
      && (!processIsClean || observation?.status === "PROJECTED")
      && scopeChecksValid
      && standalone?.status === derivedScope
      && privateEvidenceValid
      && referencesValid;
    if (!witnessValid) {
      add(
        problems,
        "P07_RUN_DISPOSITION",
        relative(root, runExampleFile),
        "FinalizedContractRun must carry one closed spawn/process/capture/scope witness whose disposition is uniquely derivable",
      );
    }
  }

  if (execution && typeof execution === "object" && !Array.isArray(execution)) {
    if (Object.hasOwn(execution, "artifact_digest")) {
      add(problems, "P07_SELF_DIGEST_FORBIDDEN", relative(root, executionExampleFile), "execution example contains a self digest");
    }
    const witness = finalizedRun?.closed_run_witness;
    const process = witness?.process_closure;
    const observation = witness?.observation;
    const standalone = witness?.standalone_scope;
    const eligible = process?.status === "PROCESS_CLEAN"
      && observation?.status === "PROJECTED"
      && standalone?.status === "COMPLETE";
    if ((execution.result === "INELIGIBLE_EXECUTION") === eligible) {
      add(
        problems,
        "P07_EXECUTION_OBSERVATION",
        relative(root, executionExampleFile),
        "classification eligibility must derive from the exact finalized witness rather than caller-selected result authority",
      );
    }
    if (
      !target
      || !finalizedRun
      || execution.contract_execution_target_digest !== p07TypedDigest("ContractExecutionTarget", target)
      || execution.finalized_contract_run_digest !== p07TypedDigest("FinalizedContractRun", finalizedRun)
      || finalizedRun.contract_execution_target_digest !== execution.contract_execution_target_digest
    ) {
      add(
        problems,
        "P07_EXAMPLE_BINDING",
        relative(root, executionExampleFile),
        "ContractExecution must address the exact paired target and exact target-bound finalized run by external typed digest",
      );
    }
    if (eligible && Array.isArray(bundle?.predicate?.selected_fields)) {
      const observedTuple = observation?.observed_tuple;
      const fieldIDs = exactTupleFieldIDs(observedTuple);
      if (
        !jsonEqual(fieldIDs, bundle.predicate.selected_fields)
        || new Set(fieldIDs ?? []).size !== (fieldIDs?.length ?? 0)
      ) {
        add(
          problems,
          "P07_TUPLE_FIELDS",
          relative(root, executionExampleFile),
          "eligible observed tuple fields must equal the bundle selected-field roster exactly once in canonical order",
        );
      }
      const tupleIsAllowed = Array.isArray(bundle.predicate?.allowed_tuples)
        && bundle.predicate.allowed_tuples.some((tuple) => jsonEqual(tuple, observedTuple));
      if (
        (execution.result === "CONFORMS" && !tupleIsAllowed)
        || (execution.result === "CONTRADICTS" && tupleIsAllowed)
        || execution.result === "INELIGIBLE_EXECUTION"
      ) {
        add(
          problems,
          "P07_EXECUTION_CONFORMANCE",
          relative(root, executionExampleFile),
          "eligible conformance must agree exactly with complete allowed-tuple membership",
        );
      }
    }
    if (
      execution.publication_scope !== "IMMUTABLE_NONHEAD_CLASSIFICATION_V1"
      || execution.classifier_profile !== "CONTRACT_EXECUTION_EXACT_TUPLE_V1"
      || [
        "study_head_advanced",
        "choicepoint_freshened",
        "historical_execution_evidence_reused",
        "current",
        "latest",
        "head",
      ].some((field) => Object.hasOwn(execution, field))
    ) {
      add(problems, "P07_HEAD_SEPARATION", relative(root, executionExampleFile), "ContractExecution must remain immutable nonhead classification evidence with no selector or history mutation claim");
    }
  }
}

const P07_PARITY_OPERATION_COUNTS = Object.freeze({
  CANONICALIZE_JSON: 72,
  PARSE_CANONICAL_JSON: 48,
  PARSE_MANIFEST_ENVELOPE: 32,
  PARSE_READY_FRAME: 24,
  PARSE_HTTP_RESPONSE: 96,
  PROJECT_CLI_OBSERVATION: 48,
  PROJECT_HTTP_OBSERVATION: 48,
  EVALUATE_EXACT_PREDICATE: 64,
  SELECT_DIRECT_RESULT: 48,
  SELECT_OWNER_ELIGIBILITY: 32,
});

function validateP07ParityCorpus(root, file, rows, problems) {
  const label = relative(root, file);
  const exact = fs.readFileSync(file);
  if (exact.length === 0 || exact.at(-1) !== 0x0a || exact.includes(0x0d)) {
    add(problems, "P07_PARITY_CORPUS_FRAME", label, "parity corpus must be nonempty, LF-only, and final-LF terminated");
  }
  if (rows.length !== 512) {
    add(problems, "P07_PARITY_CORPUS_COUNT", label, `parity corpus must contain exactly 512 vectors; got ${rows.length}`);
  }
  const exactLines = exact.toString("utf8").split("\n");
  const ids = new Set();
  const counts = new Map(Object.keys(P07_PARITY_OPERATION_COUNTS).map((operation) => [operation, 0]));
  for (const { value, line } of rows) {
    if (exactLines[line - 1] !== p07CanonicalJSON(value)) {
      add(problems, "P07_PARITY_CORPUS_CANONICAL", label, "parity vector must be exact canonical JSON", line);
    }
    if (!jsonEqual(Object.keys(value), ["description", "expected", "id", "input", "operation", "tags"])) {
      add(problems, "P07_PARITY_CORPUS_ROSTER", label, "parity vector must retain its exact six-member root roster", line);
      continue;
    }
    if (typeof value.description !== "string" || value.description.length === 0) {
      add(problems, "P07_PARITY_CORPUS_DESCRIPTION", label, "parity vector description must be nonempty text", line);
    }
    if (typeof value.id !== "string" || !/^[a-z0-9][a-z0-9._-]{0,95}$/u.test(value.id) || ids.has(value.id)) {
      add(problems, "P07_PARITY_CORPUS_ID", label, "parity vector ID must be unique closed-profile text", line);
    } else {
      ids.add(value.id);
    }
    if (!counts.has(value.operation)) {
      add(problems, "P07_PARITY_CORPUS_OPERATION", label, `parity vector operation is outside the closed roster: ${value.operation}`, line);
    } else {
      counts.set(value.operation, counts.get(value.operation) + 1);
    }
    if (!value.input || typeof value.input !== "object" || Array.isArray(value.input)) {
      add(problems, "P07_PARITY_CORPUS_INPUT", label, "parity vector input must be an object", line);
    }
    const expectedKeys = Object.keys(value.expected ?? {});
    const expectedStatus = value.expected?.status;
    if (
      !value.expected || typeof value.expected !== "object" || Array.isArray(value.expected)
      || (expectedStatus === "OK" && !jsonEqual(expectedKeys, ["status", "value"]))
      || (expectedStatus === "REFUSED" && !jsonEqual(expectedKeys, ["code", "status"]))
      || !["OK", "REFUSED"].includes(expectedStatus)
    ) {
      add(problems, "P07_PARITY_CORPUS_EXPECTED", label, "parity expected result must use the exact OK/value or REFUSED/code roster", line);
    }
    if (
      !Array.isArray(value.tags)
      || value.tags.some((tag) => typeof tag !== "string" || tag.length === 0)
      || value.tags.some((tag, index) => index > 0 && Buffer.compare(Buffer.from(value.tags[index - 1]), Buffer.from(tag)) >= 0)
    ) {
      add(problems, "P07_PARITY_CORPUS_TAGS", label, "parity tags must be nonempty, unique, and unsigned-UTF-8 sorted", line);
    }
  }
  for (const [operation, expected] of Object.entries(P07_PARITY_OPERATION_COUNTS)) {
    if (counts.get(operation) !== expected) {
      add(problems, "P07_PARITY_CORPUS_DISTRIBUTION", label, `${operation} must have exactly ${expected} vectors; got ${counts.get(operation)}`);
    }
  }
}

function validateVectors(root, problems) {
  const vectorDirectory = path.join(root, "spec/vectors/v1");
  const files = sortedFiles(vectorDirectory, (file) => file.endsWith(".jsonl"));
  assertNoCaseFoldedFileCollisions(root, files, problems);
  const refusalFile = path.join(vectorDirectory, "refusals.jsonl");
  const parityFile = path.join(vectorDirectory, "contract-parity.jsonl");
  if (!fs.existsSync(refusalFile)) {
    add(problems, "REQUIRED_FILE_MISSING", "spec/vectors/v1/refusals.jsonl", "required refusal vector corpus is missing");
    return { files, rowCount: 0 };
  }

  const rows = [];
  for (const file of files) {
    for (const row of parseJsonLinesFile(root, file, problems)) rows.push({ ...row, file });
  }
  const parityRows = rows.filter((row) => row.file === parityFile);
  if (fs.existsSync(parityFile)) validateP07ParityCorpus(root, parityFile, parityRows, problems);
  const genericRows = rows.filter((row) => row.file !== parityFile);
  const refusalRows = genericRows.filter((row) => row.file === refusalFile);

  const byId = new Map();
  for (const row of genericRows) {
    const { value, file, line } = row;
    if (value.schema_version !== "countershape-vector/v1") {
      add(problems, "VECTOR_SCHEMA_VERSION", relative(root, file), "vector must use schema_version countershape-vector/v1", line);
    }
    if (typeof value.id !== "string" || value.id.length === 0) {
      add(problems, "VECTOR_ID_MISSING", relative(root, file), "vector must have a nonempty string id", line);
      continue;
    }
    if (byId.has(value.id)) {
      add(problems, "VECTOR_ID_DUPLICATE", relative(root, file), `vector id ${value.id} duplicates ${byId.get(value.id)}`, line);
    } else {
      byId.set(value.id, `${relative(root, file)}:${line}`);
    }
    if (typeof value.artifact_kind !== "string" || value.artifact_kind.length === 0) {
      add(problems, "VECTOR_ARTIFACT_KIND_MISSING", relative(root, file), `${value.id} must declare artifact_kind`, line);
    }
    if (!value.input || typeof value.input !== "object" || Array.isArray(value.input)) {
      add(problems, "VECTOR_INPUT_MISSING", relative(root, file), `${value.id} must declare an input object`, line);
    }
    if (typeof value.expected_accept !== "boolean") {
      add(problems, "VECTOR_EXPECTATION_MISSING", relative(root, file), `${value.id} must declare boolean expected_accept`, line);
    }
    if (!Object.hasOwn(value, "expected_error")) {
      add(problems, "VECTOR_EXPECTATION_MISSING", relative(root, file), `${value.id} must declare expected_error (string or null)`, line);
    } else if (value.expected_error !== null && typeof value.expected_error !== "string") {
      add(problems, "VECTOR_EXPECTATION_TYPE", relative(root, file), `${value.id} expected_error must be a string or null`, line);
    }
    if (value.expected_accept === false && typeof value.expected_error !== "string") {
      add(problems, "VECTOR_EXPECTATION_INCONSISTENT", relative(root, file), `${value.id} refusal must name an expected_error`, line);
    }
    if (value.expected_accept === true && value.expected_error !== null) {
      add(problems, "VECTOR_EXPECTATION_INCONSISTENT", relative(root, file), `${value.id} accepted vector must use expected_error null`, line);
    }
  }

  for (const [id, error] of REQUIRED_VECTORS) {
    const row = refusalRows.find(({ value }) => value.id === id);
    if (!row) {
      add(problems, "REQUIRED_VECTOR_MISSING", "spec/vectors/v1/refusals.jsonl", `required refusal vector ${id} is missing`);
      continue;
    }
    if (row.value.expected_accept !== false || row.value.expected_error !== error) {
      add(problems, "VECTOR_EXPECTATION_MISMATCH", relative(root, row.file), `${id} must refuse with ${error}`, row.line);
    }
  }

  for (const [id, error] of POSITIVE_VECTORS) {
    const row = refusalRows.find(({ value }) => value.id === id);
    if (!row) {
      add(problems, "REQUIRED_VECTOR_MISSING", "spec/vectors/v1/refusals.jsonl", `required positive vector ${id} is missing`);
      continue;
    }
    if (row.value.expected_accept !== true || row.value.expected_error !== error) {
      add(problems, "VECTOR_EXPECTATION_MISMATCH", relative(root, row.file), `${id} must be accepted with a null expected_error`, row.line);
    }
  }

  const envelopeNonclaim = refusalRows.find(({ value }) => value.id === "comparison-envelope-nonclaim");
  if (envelopeNonclaim
      && (envelopeNonclaim.value.input?.behavioral_compatibility_established !== false
        || envelopeNonclaim.value.input?.placeholder_control_flow_irrelevance_established !== false)) {
    add(
      problems,
      "VECTOR_NONCLAIM_MISMATCH",
      relative(root, envelopeNonclaim.file),
      "comparison-envelope-nonclaim must keep compatibility and placeholder-irrelevance claims false",
      envelopeNonclaim.line,
    );
  }

  return { files, rowCount: rows.length };
}

function scanMarkdownFences(text) {
  let fence = null;
  const stripped = text.split(/\r?\n/u).map((line, index) => {
    if (fence === null) {
      const opening = /^ {0,3}(`{3,}|~{3,})(.*)$/u.exec(line);
      if (!opening || (opening[1][0] === "`" && opening[2].includes("`"))) return line;
      fence = {
        character: opening[1][0],
        length: opening[1].length,
        line: index + 1,
      };
      return "";
    }

    const closing = /^ {0,3}(`+|~+)[\t ]*$/u.exec(line);
    if (closing
        && closing[1][0] === fence.character
        && closing[1].length >= fence.length) {
      fence = null;
    }
    return "";
  });
  return {
    text: stripped.join("\n"),
    unclosedOpeningLine: fence?.line ?? 0,
  };
}

function markdownFiles(root) {
  const required = CONTROLLING_MARKDOWN.map((file) => path.join(root, file));
  const promptFiles = sortedFiles(path.join(root, "docs/prompts"), (file) => file.endsWith(".md"));
  return [...new Set([...required, ...promptFiles])]
    .sort((a, b) => relative(root, a).localeCompare(relative(root, b), "en"));
}

function validateMarkdownLinks(root, files, problems) {
  for (const file of files) {
    const label = relative(root, file);
    if (!fs.existsSync(file)) {
      add(problems, "REQUIRED_FILE_MISSING", label, "required controlling Markdown file is missing");
      continue;
    }
    const scanned = scanMarkdownFences(fs.readFileSync(file, "utf8"));
    if (scanned.unclosedOpeningLine > 0) {
      add(
        problems,
        "MARKDOWN_FENCE_UNCLOSED",
        label,
        "fenced code block has no matching closing fence",
        scanned.unclosedOpeningLine,
      );
    }
    const { text } = scanned;
    const lines = text.split("\n");
    for (let index = 0; index < lines.length; index += 1) {
      const linkPattern = /!?\[[^\]]*\]\(([^)]+)\)/gu;
      for (const match of lines[index].matchAll(linkPattern)) {
        let target = match[1].trim();
        if (target.startsWith("<") && target.endsWith(">")) target = target.slice(1, -1);
        if (target.startsWith("#") || /^[a-z][a-z0-9+.-]*:/iu.test(target)) continue;
        if (path.isAbsolute(target)) {
          add(problems, "MARKDOWN_LINK_NOT_RELATIVE", label, `local Markdown link must be relative: ${target}`, index + 1);
          continue;
        }
        const pathPart = target.split("#", 1)[0].split("?", 1)[0];
        let decoded;
        try {
          decoded = decodeURIComponent(pathPart);
        } catch {
          add(problems, "MARKDOWN_LINK_ENCODING", label, `link has invalid percent encoding: ${target}`, index + 1);
          continue;
        }
        const resolved = path.resolve(path.dirname(file), decoded);
        const insideRoot = resolved === root || resolved.startsWith(`${root}${path.sep}`);
        if (!insideRoot) {
          add(problems, "MARKDOWN_LINK_ESCAPES_ROOT", label, `local Markdown link escapes repository: ${target}`, index + 1);
        } else if (!fs.existsSync(resolved)) {
          add(problems, "MARKDOWN_LINK_MISSING", label, `local Markdown link target does not exist: ${target}`, index + 1);
        }
      }
    }
  }
}

function sentenceAround(line, start, length) {
  const left = Math.max(line.lastIndexOf(".", start - 1), line.lastIndexOf(";", start - 1), line.lastIndexOf("|", start - 1));
  const rightCandidates = [line.indexOf(".", start + length), line.indexOf(";", start + length), line.indexOf("|", start + length)]
    .filter((index) => index >= 0);
  const right = rightCandidates.length === 0 ? line.length : Math.min(...rightCandidates);
  return line.slice(left + 1, right + 1);
}

function validateLegacyClaims(root, files, problems) {
  for (const file of files) {
    if (!fs.existsSync(file)) continue;
    const label = relative(root, file);
    const lines = scanMarkdownFences(fs.readFileSync(file, "utf8")).text.split("\n");
    let allowDepth = 0;
    let section = "";
    for (let index = 0; index < lines.length; index += 1) {
      const line = lines[index];
      if (line.includes(ALLOW_BEGIN)) {
        allowDepth += 1;
        continue;
      }
      if (line.includes(ALLOW_END)) {
        if (allowDepth === 0) add(problems, "ALLOW_MARKER_UNBALANCED", label, "allow-prohibited-terms end marker has no begin", index + 1);
        else allowDepth -= 1;
        continue;
      }
      const heading = /^#{1,6}\s+(.+)$/u.exec(line);
      if (heading) section = heading[1];
      if (allowDepth > 0 || NEGATIVE_SECTION.test(section)) continue;
      for (const [description, pattern] of LEGACY_CLAIM_RULES) {
        pattern.lastIndex = 0;
        for (const match of line.matchAll(pattern)) {
          const sentence = sentenceAround(line, match.index ?? 0, match[0].length);
          if (NEGATIVE_CONTEXT.test(sentence)) continue;
          add(problems, "LEGACY_CLAIM", label, `affirmative prohibited claim (${description}): ${match[0]}`, index + 1);
        }
      }
    }
    if (allowDepth !== 0) add(problems, "ALLOW_MARKER_UNBALANCED", label, "allow-prohibited-terms begin marker has no end", lines.length);
  }
}

function validateDocumentedTerms(root, problems) {
  const files = CONTROLLING_MARKDOWN.map((file) => path.join(root, file));
  const combined = files.filter((file) => fs.existsSync(file)).map((file) => fs.readFileSync(file, "utf8")).join("\n");
  for (const [group, terms] of ENUM_GROUPS) {
    for (const term of terms) {
      if (!combined.includes(term)) {
        add(problems, "DOCUMENTED_TERM_MISSING", "docs", `${group} term ${term} is not documented in a controlling file`);
      }
    }
  }
}

function validate(root) {
  const problems = [];
  const schema = validateSchemas(root, problems);
  const examples = validateExamples(root, schema.schemasByFile, problems);
  validateP07Contracts(root, problems, schema.schemasByFile);
  const vectors = validateVectors(root, problems);
  const markdown = markdownFiles(root);
  validateMarkdownLinks(root, markdown, problems);
  // Prompt files are executable implementation/test instructions, not public
  // product claims. Their links are still checked above, while claim language
  // is policed only in the six controlling public contracts. Those documents
  // use explicit allow markers or negative context for quoted vocabulary.
  validateLegacyClaims(
    root,
    CONTROLLING_MARKDOWN.map((file) => path.join(root, file)),
    problems,
  );
  validateDocumentedTerms(root, problems);
  problems.sort((a, b) =>
    a.file.localeCompare(b.file, "en")
      || a.line - b.line
      || a.code.localeCompare(b.code, "en")
      || a.message.localeCompare(b.message, "en"));
  return {
    problems,
    counts: {
      schemas: schema.files.length,
      examples: examples.files.length,
      vectors: vectors.rowCount,
      markdown: markdown.filter((file) => fs.existsSync(file)).length,
    },
  };
}

function copyRelevantInputs(root, destination) {
  fs.mkdirSync(destination, { recursive: true });
  for (const item of ["docs", "spec"]) {
    const source = path.join(root, item);
    if (!fs.existsSync(source)) throw new Error(`self-test input anchor is absent: ${item}`);
    fs.cpSync(source, path.join(destination, item), { recursive: true, dereference: false });
  }
}

function replaceAnchor(file, anchor, replacement) {
  const text = fs.readFileSync(file, "utf8");
  const first = text.indexOf(anchor);
  if (first < 0) throw new Error(`self-test mutation anchor is absent: ${relative(path.dirname(path.dirname(file)), file)} :: ${anchor}`);
  if (text.indexOf(anchor, first + anchor.length) >= 0) {
    throw new Error(`self-test mutation anchor is ambiguous: ${path.basename(file)} :: ${anchor}`);
  }
  fs.writeFileSync(file, `${text.slice(0, first)}${replacement}${text.slice(first + anchor.length)}`, "utf8");
}

function rewriteJsonObject(file, mutate) {
  const parsed = strictJsonParse(fs.readFileSync(file, "utf8"), file);
  if (parsed.duplicates.length > 0) throw new Error(`self-test JSON anchor contains duplicate keys: ${file}`);
  if (!parsed.value || typeof parsed.value !== "object" || Array.isArray(parsed.value)) {
    throw new Error(`self-test JSON anchor is not an object: ${file}`);
  }
  mutate(parsed.value);
  fs.writeFileSync(file, `${JSON.stringify(parsed.value, null, 2)}\n`, "utf8");
}

function collapseUnresolved(root) {
  const files = sortedFiles(path.join(root, "spec/schema/v1"), (file) => file.endsWith(".json"));
  let replacements = 0;
  for (const file of files) {
    const text = fs.readFileSync(file, "utf8");
    const matches = text.match(/"UNRESOLVED"/gu)?.length ?? 0;
    if (matches > 0) {
      fs.writeFileSync(file, text.replaceAll('"UNRESOLVED"', "false"), "utf8");
      replacements += matches;
    }
  }
  if (replacements === 0) throw new Error("self-test mutation anchor is absent: schema enum UNRESOLVED");
  const reductionFile = path.join(root, "spec/schema/v1/reduction-run.schema.json");
  rewriteJsonObject(reductionFile, (schema) => {
    schema.$defs ??= {};
    if (Object.hasOwn(schema.$defs, "ValidatorEnumDecoy")) {
      throw new Error("self-test mutation anchor already exists: ValidatorEnumDecoy");
    }
    // A stray copy of the missing string must not satisfy the agreement. Only
    // the bound decision declaration is authoritative.
    schema.$defs.ValidatorEnumDecoy = { const: "UNRESOLVED" };
  });
}

const SELF_TESTS = [
  {
    id: "required-object-removal",
    expectedCode: "REQUIRED_FILE_MISSING",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/world-plan.schema.json");
      if (!fs.existsSync(file)) throw new Error("self-test mutation anchor is absent: world-plan.schema.json");
      fs.rmSync(file);
    },
  },
  {
    id: "required-example-field-corruption",
    expectedCode: "EXAMPLE_SCHEMA_VALIDATION",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/world-plan.valid.json");
      rewriteJsonObject(file, (value) => {
        if (!Object.hasOwn(value, "kind")) throw new Error("self-test mutation anchor is absent: WorldPlan kind");
        delete value.kind;
      });
    },
  },
  {
    id: "coordinated-example-kind-omission",
    expectedCode: "EXAMPLE_KIND_MISMATCH",
    mutate(root) {
      const exampleFile = path.join(root, "spec/examples/v1/world-plan.valid.json");
      rewriteJsonObject(exampleFile, (value) => {
        if (value.kind !== "WorldPlan") throw new Error("self-test mutation anchor is absent: exact WorldPlan example kind");
        delete value.kind;
      });
      const schemaFile = path.join(root, "spec/schema/v1/world-plan.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const index = schema.required?.indexOf("kind") ?? -1;
        if (index < 0) throw new Error("self-test mutation anchor is absent: required WorldPlan kind");
        schema.required.splice(index, 1);
      });
    },
  },
  {
    id: "mapped-schema-kind-decoy",
    expectedCode: "SCHEMA_MAPPED_OBJECT_MISSING",
    mutate(root) {
      const mappedFile = path.join(root, "spec/schema/v1/world-plan.schema.json");
      rewriteJsonObject(mappedFile, (schema) => {
        if (schema.properties?.kind?.const !== "WorldPlan") {
          throw new Error("self-test mutation anchor is absent: mapped WorldPlan kind authority");
        }
        schema.properties.kind = { type: "string" };
      });
      const decoyFile = path.join(root, "spec/schema/v1/source-spec.schema.json");
      rewriteJsonObject(decoyFile, (schema) => {
        schema.$defs ??= {};
        if (Object.hasOwn(schema.$defs, "ValidatorWorldPlanDecoy")) {
          throw new Error("self-test mutation anchor already exists: ValidatorWorldPlanDecoy");
        }
        schema.$defs.ValidatorWorldPlanDecoy = {
          type: "object",
          properties: { kind: { const: "WorldPlan" } },
        };
      });
    },
  },
  {
    id: "schema-id-rebinding",
    expectedCode: "SCHEMA_ID_MISMATCH",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/world-plan.schema.json");
      rewriteJsonObject(file, (schema) => {
        const expected = `${SCHEMA_ID_BASE}world-plan.schema.json`;
        if (schema.$id !== expected) throw new Error("self-test mutation anchor is absent: exact WorldPlan $id");
        schema.$id = "https://example.invalid/unique-but-wrong-world-plan.schema.json";
      });
    },
  },
  {
    id: "p07-self-digest-restoration",
    expectedCode: "P07_SELF_DIGEST_FORBIDDEN",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
      rewriteJsonObject(file, (schema) => {
        if (Object.hasOwn(schema.properties ?? {}, "artifact_digest")) {
          throw new Error("self-test mutation anchor already exists: ContractBundle artifact_digest");
        }
        schema.required.push("artifact_digest");
        schema.properties.artifact_digest = { $ref: "common.schema.json#/$defs/Digest" };
      });
    },
  },
  {
    id: "p07-target-optional-self-authority",
    expectedCode: "P07_EXECUTION_AUTHORITY",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/contract-execution-target.schema.json");
      rewriteJsonObject(file, (schema) => {
        if (Object.hasOwn(schema.properties ?? {}, "contract_execution_target_digest")) {
          throw new Error("self-test mutation anchor already exists: optional target self authority");
        }
        schema.properties.contract_execution_target_digest = { $ref: "common.schema.json#/$defs/Digest" };
      });
    },
  },
  {
    id: "p07-execution-optional-runtime-authority",
    expectedCode: "P07_EXECUTION_AUTHORITY",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/contract-execution.schema.json");
      rewriteJsonObject(file, (schema) => {
        if (Object.hasOwn(schema.properties ?? {}, "runtime_binding")) {
          throw new Error("self-test mutation anchor already exists: optional execution runtime authority");
        }
        schema.properties.runtime_binding = { type: "object" };
      });
    },
  },
  {
    id: "p07-c1-nested-required-roster-removal",
    expectedCode: "P07_EXECUTION_AUTHORITY",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/finalized-contract-run.schema.json");
      rewriteJsonObject(file, (schema) => {
        const required = schema.$defs?.ClosedRunWitness?.required;
        const index = required?.indexOf("private_evidence") ?? -1;
        if (index < 0) throw new Error("self-test mutation anchor is absent: ClosedRunWitness private_evidence required member");
        required.splice(index, 1);
      });
    },
  },
  {
    id: "p07-execution-tuple-reason-authority-resurrection",
    expectedCode: "P07_EXECUTION_AUTHORITY",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/contract-execution.schema.json");
      rewriteJsonObject(file, (schema) => {
        const result = schema.properties?.result;
        if (!jsonEqual(result?.enum, ["CONFORMS", "CONTRADICTS", "INELIGIBLE_EXECUTION"]) || Object.hasOwn(result, "oneOf")) {
          throw new Error("self-test mutation anchor is absent: flat derived execution result");
        }
        delete result.enum;
        result.oneOf = [
          {
            type: "object",
            additionalProperties: false,
            required: ["status", "observed_tuple"],
            properties: {
              status: { enum: ["CONFORMS", "CONTRADICTS"] },
              observed_tuple: { $ref: "common.schema.json#/$defs/ExactTuple" },
            },
          },
          {
            type: "object",
            additionalProperties: false,
            required: ["status", "reason"],
            properties: {
              status: { const: "INELIGIBLE_EXECUTION" },
              reason: { $ref: "common.schema.json#/$defs/ControlReason" },
            },
          },
        ];
      });
    },
  },
  {
    id: "p07a-portable-choice-mode-removal",
    expectedCode: "P07A_SCHEMA_AUTHORITY",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/choicepoint.schema.json");
      rewriteJsonObject(file, (schema) => {
        const modes = schema.properties?.choice_projection_mode?.enum;
        if (!jsonEqual(modes, ["WHOLE_EXACT_CANONICAL_PROJECTION_V1", "ADAPTER_BOUND_PORTABLE_FIELDS_V1"])) {
          throw new Error("self-test mutation anchor is absent: exact dual Choicepoint projection modes");
        }
        modes.pop();
      });
    },
  },
  {
    id: "p07a-portable-exact-tag-removal",
    expectedCode: "P07A_SCHEMA_AUTHORITY",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/decision-record.schema.json");
      rewriteJsonObject(file, (schema) => {
        const tags = schema.$defs?.ExactValue?.properties?.tag?.enum;
        const index = tags?.indexOf("BYTES") ?? -1;
        if (index < 0) throw new Error("self-test mutation anchor is absent: DecisionRecord BYTES tag");
        tags.splice(index, 1);
      });
    },
  },
  {
    id: "p07-execution-unbound-tree-authority",
    expectedCode: "P07_EXECUTION_AUTHORITY",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/contract-execution-target.schema.json");
      rewriteJsonObject(file, (schema) => {
        const authority = schema.properties?.tree_binding?.properties?.authority;
        if (authority?.const !== "GIT_PIN_INSPECT_MATERIALIZE_V1") {
          throw new Error("self-test mutation anchor is absent: exact Git target authority");
        }
        delete authority.const;
        authority.enum = ["GIT_PIN_INSPECT_MATERIALIZE_V1", "CALLER_DIGESTS_V1"];
      });
    },
  },
  {
    id: "p07-target-tree-identity-substitution",
    expectedCode: "P07_TARGET_TREE_BINDING",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-execution-target.valid.json");
      rewriteJsonObject(file, (target) => {
        const digest = target.tree_binding?.tree_identity_digest;
        if (typeof digest !== "string") {
          throw new Error("self-test mutation anchor is absent: recomputed pinned tree identity digest");
        }
        target.tree_binding.tree_identity_digest = "sha256:abababababababababababababababababababababababababababababababab";
      });
    },
  },
  {
    id: "p07-target-materialization-manifest-removal",
    expectedCode: "P07_EXECUTION_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-execution-target.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const tree = schema.properties?.tree_binding;
        const index = tree?.required?.indexOf("materialization_manifest_digest") ?? -1;
        if (index < 0 || !Object.hasOwn(tree.properties ?? {}, "materialization_manifest_digest")) {
          throw new Error("self-test mutation anchor is absent: target materialization manifest");
        }
        tree.required.splice(index, 1);
        delete tree.properties.materialization_manifest_digest;
      });
    },
  },
  {
    id: "p07-world-instance-authority-resurrection",
    expectedCode: "P07_EXECUTION_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-execution.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        if (Object.hasOwn(schema.properties ?? {}, "world_instance_digest")) {
          throw new Error("self-test mutation anchor already exists: ContractExecution WorldInstance");
        }
        schema.required.push("world_instance_digest");
        schema.properties.world_instance_digest = { $ref: "common.schema.json#/$defs/Digest" };
      });
      const exampleFile = path.join(root, "spec/examples/v1/contract-execution.valid.json");
      rewriteJsonObject(exampleFile, (execution) => {
        execution.world_instance_digest = "sha256:e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4e4";
      });
    },
  },
  {
    id: "p07-tuple-digest-authority-regression",
    expectedCode: "P07_TUPLE_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/common.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const tuple = schema.$defs?.ExactTuple;
        if (!jsonEqual(tuple?.required, ["fields"]) || Object.hasOwn(tuple.properties ?? {}, "tuple_digest")) {
          throw new Error("self-test mutation anchor is absent: fields-only ExactTuple");
        }
        tuple.required.unshift("tuple_digest");
        tuple.properties.tuple_digest = { $ref: "#/$defs/Digest" };
      });
      for (const name of ["contract-bundle.valid.json", "finalized-contract-run.valid.json"]) {
        const exampleFile = path.join(root, "spec/examples/v1", name);
        rewriteJsonObject(exampleFile, (value) => {
          const tuple = name.startsWith("contract-bundle")
            ? value.predicate?.allowed_tuples?.[0]
            : value.closed_run_witness?.observation?.observed_tuple;
          if (!tuple || Object.hasOwn(tuple, "tuple_digest")) {
            throw new Error(`self-test mutation anchor is absent: fields-only tuple in ${name}`);
          }
          tuple.tuple_digest = "sha256:4141414141414141414141414141414141414141414141414141414141414141";
        });
      }
    },
  },
  {
    id: "p07-tuple-fields-minimum-removal",
    expectedCode: "P07_TUPLE_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/common.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const fields = schema.$defs?.ExactTuple?.properties?.fields;
        if (fields?.type !== "array" || fields?.minItems !== 1) {
          throw new Error("self-test mutation anchor is absent: nonempty ExactTuple fields array");
        }
        delete fields.minItems;
      });
    },
  },
  {
    id: "p07-tuple-fields-type-widening",
    expectedCode: "P07_TUPLE_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/common.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const fields = schema.$defs?.ExactTuple?.properties?.fields;
        if (fields?.type !== "array") {
          throw new Error("self-test mutation anchor is absent: ExactTuple fields array type");
        }
        fields.type = ["array", "object"];
      });
    },
  },
  {
    id: "p07-decision-tuple-digest-authority-regression",
    expectedCode: "P07_TUPLE_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/decision-record.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const tuple = schema.$defs?.ExactTuple;
        if (!jsonEqual(tuple?.required, ["fields"]) || Object.hasOwn(tuple.properties ?? {}, "tuple_digest")) {
          throw new Error("self-test mutation anchor is absent: DecisionRecord fields-only ExactTuple");
        }
        tuple.required.push("tuple_digest");
        tuple.properties.tuple_digest = { $ref: "common.schema.json#/$defs/Digest" };
      });
      const exampleFile = path.join(root, "spec/examples/v1/decision-record.valid.json");
      rewriteJsonObject(exampleFile, (decision) => {
        const tuples = [
          ...(decision.allowed_complete_tuples ?? []),
          ...(decision.disallowed_complete_tuples ?? []),
        ];
        if (tuples.length === 0 || tuples.some((tuple) => Object.hasOwn(tuple, "tuple_digest"))) {
          throw new Error("self-test mutation anchor is absent: DecisionRecord tuple examples");
        }
        for (const tuple of tuples) {
          tuple.tuple_digest = "sha256:4141414141414141414141414141414141414141414141414141414141414141";
        }
      });
    },
  },
  {
    id: "p07-recoverable-content-removal",
    expectedCode: "P07_BUNDLE_RECOVERY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const item = schema.properties?.files?.items;
        const index = item?.required?.indexOf("content_base64") ?? -1;
        if (index < 0 || !Object.hasOwn(item.properties ?? {}, "content_base64")) {
          throw new Error("self-test mutation anchor is absent: recoverable file content");
        }
        item.required.splice(index, 1);
        delete item.properties.content_base64;
      });
      const exampleFile = path.join(root, "spec/examples/v1/contract-bundle.valid.json");
      rewriteJsonObject(exampleFile, (bundle) => {
        const entry = bundle.files?.find((candidate) => candidate.path === "README.md");
        if (!entry || typeof entry.content_base64 !== "string") {
          throw new Error("self-test mutation anchor is absent: README recoverable content");
        }
        delete entry.content_base64;
      });
    },
  },
  {
    id: "p07-non-utf8-file-bytes",
    expectedCode: "P07_FILE_TEXT",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-bundle.valid.json");
      rewriteJsonObject(file, (bundle) => {
        const invalidText = Buffer.from([0xff, 0x0a]);
        replaceP07BundleMemberAndManifest(bundle, "README.md", invalidText);
      });
    },
  },
  {
    id: "p07-manifest-self-coverage",
    expectedCode: "P07_MANIFEST",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-bundle.valid.json");
      rewriteJsonObject(file, (bundle) => {
        const manifestEntry = bundle.files?.find((entry) => entry.path === "manifest.json");
        if (!manifestEntry || typeof manifestEntry.content_base64 !== "string") {
          throw new Error("self-test mutation anchor is absent: embedded manifest");
        }
        const manifest = JSON.parse(Buffer.from(manifestEntry.content_base64, "base64").toString("utf8"));
        manifest.files.push({
          path: "manifest.json",
          mode: manifestEntry.mode,
          byte_count: manifestEntry.byte_count,
          byte_sha256: manifestEntry.byte_sha256,
        });
        const bytes = Buffer.from(`${JSON.stringify(manifest)}\n`, "utf8");
        manifestEntry.content_base64 = bytes.toString("base64");
        manifestEntry.byte_count = bytes.length;
        manifestEntry.byte_sha256 = rawSha256(bytes);
      });
    },
  },
  {
    id: "p07-noop-harness-role",
    expectedCode: "P07_FILE_ROLE",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-bundle.valid.json");
      rewriteJsonObject(file, (bundle) => {
        replaceP07BundleMemberAndManifest(
          bundle,
          "harness.mjs",
          Buffer.from(
            "const decoys = ['spawnSync(process.execPath', 'process.argv[2] === \\\"--subject\\\"'];\n"
              + "export function runFixture() { return { fields: [] }; }\n"
              + "export function evaluate() { return decoys.length ? \\\"CONFORMS\\\" : \\\"CONTRADICTS\\\"; }\n",
            "utf8",
          ),
        );
      });
    },
  },
  {
    id: "p07-profile-binding-substitution",
    expectedCode: "P07_PROFILE_BINDING",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-bundle.valid.json");
      rewriteJsonObject(file, (bundle) => {
        if (bundle.predicate?.portable_profile_digest !== bundle.portable_profile_digest) {
          throw new Error("self-test mutation anchor is absent: matching portable profile digests");
        }
        bundle.predicate.portable_profile_digest = "sha256:abababababababababababababababababababababababababababababababab";
      });
    },
  },
  {
    id: "p07-target-bundle-digest-substitution",
    expectedCode: "P07_TARGET_SOURCE_BINDING",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-execution-target.valid.json");
      rewriteJsonObject(file, (target) => {
        if (typeof target.contract_bundle_digest !== "string") {
          throw new Error("self-test mutation anchor is absent: target ContractBundle digest");
        }
        target.contract_bundle_digest = "sha256:abababababababababababababababababababababababababababababababab";
      });
    },
  },
  {
    id: "p07-execution-target-digest-substitution",
    expectedCode: "P07_EXAMPLE_BINDING",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-execution.valid.json");
      rewriteJsonObject(file, (execution) => {
        if (typeof execution.contract_execution_target_digest !== "string") {
          throw new Error("self-test mutation anchor is absent: paired ContractExecutionTarget digest");
        }
        execution.contract_execution_target_digest = "sha256:abababababababababababababababababababababababababababababababab";
      });
    },
  },
  {
    id: "p07-finalized-run-target-digest-substitution",
    expectedCode: "P07_TARGET_RUN_BINDING",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/finalized-contract-run.valid.json");
      rewriteJsonObject(file, (finalizedRun) => {
        if (typeof finalizedRun.contract_execution_target_digest !== "string") {
          throw new Error("self-test mutation anchor is absent: finalized-run target digest");
        }
        finalizedRun.contract_execution_target_digest = "sha256:abababababababababababababababababababababababababababababababab";
      });
    },
  },
  {
    id: "p07-finalized-run-attempt-substitution",
    expectedCode: "P07_TARGET_RUN_BINDING",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/finalized-contract-run.valid.json");
      rewriteJsonObject(file, (finalizedRun) => {
        if (typeof finalizedRun.attempt_artifact_digest !== "string") {
          throw new Error("self-test mutation anchor is absent: finalized-run attempt digest");
        }
        finalizedRun.attempt_artifact_digest = "sha256:abababababababababababababababababababababababababababababababab";
      });
    },
  },
  {
    id: "p07-execution-run-digest-substitution",
    expectedCode: "P07_EXAMPLE_BINDING",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-execution.valid.json");
      rewriteJsonObject(file, (execution) => {
        if (typeof execution.finalized_contract_run_digest !== "string") {
          throw new Error("self-test mutation anchor is absent: exact finalized-run digest");
        }
        execution.finalized_contract_run_digest = "sha256:abababababababababababababababababababababababababababababababab";
      });
    },
  },
  {
    id: "p07-target-residue-current-substitution",
    expectedCode: "P07_TARGET_SOURCE_BINDING",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-execution-target.valid.json");
      rewriteJsonObject(file, (target) => {
        if (target.terminal_residue_binding?.current_digest !== target.contract_bundle_digest) {
          throw new Error("self-test mutation anchor is absent: residue current bundle binding");
        }
        target.terminal_residue_binding.current_digest = "sha256:abababababababababababababababababababababababababababababababab";
      });
    },
  },
  {
    id: "p07-target-residue-kind-substitution",
    expectedCode: "P07_TARGET_SOURCE_BINDING",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-execution-target.valid.json");
      rewriteJsonObject(file, (target) => {
        if (target.terminal_residue_binding?.current_kind !== "ContractBundle") {
          throw new Error("self-test mutation anchor is absent: residue current kind");
        }
        target.terminal_residue_binding.current_kind = "DecisionRecord";
      });
    },
  },
  {
    id: "p07-subject-entrypoint-ceiling-widening",
    expectedCode: "P07_LAUNCH_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const entrypoint = schema.properties?.source_profile?.properties?.subject_entrypoint;
        if (entrypoint?.maxLength !== 4096) {
          throw new Error("self-test mutation anchor is absent: subject entrypoint ceiling");
        }
        entrypoint.maxLength = 8192;
      });
    },
  },
  {
    id: "p07-subject-entrypoint-root-hyphen-widening",
    expectedCode: "P07_LAUNCH_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const entrypoint = schema.properties?.source_profile?.properties?.subject_entrypoint;
        const exact = "^[A-Za-z0-9_][A-Za-z0-9._-]*(?:/[A-Za-z0-9_-][A-Za-z0-9._-]*)*\\.(?:js|mjs|cjs)$";
        if (entrypoint?.pattern !== exact) {
          throw new Error("self-test mutation anchor is absent: subject entrypoint root-hyphen refusal");
        }
        entrypoint.pattern = "^(?:[A-Za-z0-9_-][A-Za-z0-9._-]*/)*[A-Za-z0-9_-][A-Za-z0-9._-]*\\.(?:js|mjs|cjs)$";
      });
    },
  },
  {
    id: "p07-repo-executable-launch-resurrection",
    expectedCode: "P07_LAUNCH_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const launch = schema.properties?.source_profile?.properties?.launch_profile;
        if (launch?.const !== "NODE_REPO_SCRIPT_V1") {
          throw new Error("self-test mutation anchor is absent: Node-only launch profile");
        }
        delete launch.const;
        launch.enum = ["NODE_REPO_SCRIPT_V1", "REPO_EXECUTABLE_V1"];
      });
    },
  },
  {
    id: "p07-source-profile-conditional-disabled",
    expectedCode: "P07_LAUNCH_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const conditional = schema.properties?.source_profile?.allOf?.[0]?.if;
        if (!jsonEqual(conditional?.required, ["adapter_domain"])) {
          throw new Error("self-test mutation anchor is absent: CLI source-profile conditional");
        }
        conditional.required = ["validator_never"];
      });
    },
  },
  {
    id: "p07-runtime-family-widening",
    expectedCode: "P07_LAUNCH_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const runtimeFamily = schema.properties?.source_profile?.properties?.runtime_family;
        if (runtimeFamily?.const !== "NODE") {
          throw new Error("self-test mutation anchor is absent: exact Node runtime family");
        }
        delete runtimeFamily.const;
        runtimeFamily.enum = ["NODE", "PYTHON"];
      });
    },
  },
  {
    id: "p07-target-reused-attempt",
    expectedCode: "P07_TARGET_RUNTIME_BINDING",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-execution-target.valid.json");
      rewriteJsonObject(file, (target) => {
        if (target.attempt_binding?.purpose !== "CONFORMANCE") {
          throw new Error("self-test mutation anchor is absent: fresh conformance attempt");
        }
        target.attempt_binding.purpose = "DISCOVERY";
      });
    },
  },
  {
    id: "p07-target-runtime-executable-digest-removal",
    expectedCode: "P07_EXECUTION_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-execution-target.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const identity = schema.properties?.runtime_binding?.properties?.executable_identity;
        const index = identity?.required?.indexOf("bytes_digest") ?? -1;
        if (index < 0 || !Object.hasOwn(identity.properties ?? {}, "bytes_digest")) {
          throw new Error("self-test mutation anchor is absent: runtime executable digest");
        }
        identity.required.splice(index, 1);
        delete identity.properties.bytes_digest;
      });
    },
  },
  {
    id: "p07-finalized-run-tuple-substitution",
    expectedCode: "P07_EXECUTION_CONFORMANCE",
    mutate(root) {
      const runFile = path.join(root, "spec/examples/v1/finalized-contract-run.valid.json");
      rewriteJsonObject(runFile, (finalizedRun) => {
        const witness = finalizedRun.closed_run_witness;
        const value = witness?.observation?.observed_tuple?.fields?.[0]?.value;
        if (witness?.process_closure?.status !== "PROCESS_CLEAN" || value?.tag !== "BYTES" || value?.base64 !== "b2sK") {
          throw new Error("self-test mutation anchor is absent: finalized clean BYTES tuple");
        }
        value.base64 = "bm8K";
      });
      const finalizedRun = strictJsonParse(fs.readFileSync(runFile, "utf8"), runFile).value;
      const executionFile = path.join(root, "spec/examples/v1/contract-execution.valid.json");
      rewriteJsonObject(executionFile, (execution) => {
        if (execution.result !== "CONFORMS") {
          throw new Error("self-test mutation anchor is absent: conforming classification");
        }
        execution.finalized_contract_run_digest = p07TypedDigest("FinalizedContractRun", finalizedRun);
      });
    },
  },
  {
    id: "p07-eligible-without-projection",
    expectedCode: "P07_EXECUTION_OBSERVATION",
    mutate(root) {
      const executionFile = path.join(root, "spec/examples/v1/contract-execution.valid.json");
      const execution = strictJsonParse(fs.readFileSync(executionFile, "utf8"), executionFile).value;
      if (execution.result !== "CONFORMS") {
        throw new Error("self-test mutation anchor is absent: eligible execution");
      }
      const runFile = path.join(root, "spec/examples/v1/finalized-contract-run.valid.json");
      rewriteJsonObject(runFile, (finalizedRun) => {
        const witness = finalizedRun.closed_run_witness;
        const observation = witness?.observation;
        if (witness?.process_closure?.status !== "PROCESS_CLEAN" || observation?.status !== "PROJECTED") {
          throw new Error("self-test mutation anchor is absent: projected finalized run");
        }
        witness.process_closure.status = "PROCESS_CONTROLLED";
        witness.process_closure.primary_reason = "PROJECTION_REJECTED";
        witness.observation = {
          status: "CAPTURED_UNPROJECTED",
          captured_observation_ref: observation.captured_observation_ref,
        };
      });
      const finalizedRun = strictJsonParse(fs.readFileSync(runFile, "utf8"), runFile).value;
      rewriteJsonObject(executionFile, (executionValue) => {
        executionValue.finalized_contract_run_digest = p07TypedDigest("FinalizedContractRun", finalizedRun);
      });
    },
  },
  {
    id: "p07-finalized-run-control-reason-substitution",
    expectedCode: "P07_RUN_DISPOSITION",
    mutate(root) {
      const runFile = path.join(root, "spec/examples/v1/finalized-contract-run.valid.json");
      rewriteJsonObject(runFile, (finalizedRun) => {
        const process = finalizedRun.closed_run_witness?.process_closure;
        const observation = finalizedRun.closed_run_witness?.observation;
        if (process?.status !== "PROCESS_CLEAN" || process?.primary_reason !== "NONE" || observation?.status !== "PROJECTED") {
          throw new Error("self-test mutation anchor is absent: clean projected finalized run");
        }
        process.primary_reason = "TIMEOUT";
      });
      const finalizedRun = strictJsonParse(fs.readFileSync(runFile, "utf8"), runFile).value;
      const executionFile = path.join(root, "spec/examples/v1/contract-execution.valid.json");
      rewriteJsonObject(executionFile, (execution) => {
        execution.finalized_contract_run_digest = p07TypedDigest("FinalizedContractRun", finalizedRun);
      });
    },
  },
  {
    id: "p07-projected-controlled-classified-conforming",
    expectedCode: "P07_EXECUTION_OBSERVATION",
    mutate(root) {
      const runFile = path.join(root, "spec/examples/v1/finalized-contract-run.valid.json");
      rewriteJsonObject(runFile, (finalizedRun) => {
        const process = finalizedRun.closed_run_witness?.process_closure;
        const observation = finalizedRun.closed_run_witness?.observation;
        if (process?.status !== "PROCESS_CLEAN" || process?.primary_reason !== "NONE" || observation?.status !== "PROJECTED") {
          throw new Error("self-test mutation anchor is absent: clean projected finalized run");
        }
        process.status = "PROCESS_CONTROLLED";
        process.primary_reason = "TIMEOUT";
      });
      const finalizedRun = strictJsonParse(fs.readFileSync(runFile, "utf8"), runFile).value;
      const executionFile = path.join(root, "spec/examples/v1/contract-execution.valid.json");
      rewriteJsonObject(executionFile, (execution) => {
        execution.finalized_contract_run_digest = p07TypedDigest("FinalizedContractRun", finalizedRun);
      });
    },
  },
  {
    id: "p07-execution-head-advance",
    expectedCode: "P07_HEAD_SEPARATION",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-execution.valid.json");
      rewriteJsonObject(file, (execution) => {
        if (Object.hasOwn(execution, "study_head_advanced")) {
          throw new Error("self-test mutation anchor already exists: nonhead execution selector");
        }
        execution.study_head_advanced = true;
      });
    },
  },
  {
    id: "p07-receipt-cycle",
    expectedCode: "P07_RECEIPT_CYCLE",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-bundle.valid.json");
      rewriteJsonObject(file, (bundle) => {
        if (Object.hasOwn(bundle, "standalone_absence_receipt")) {
          throw new Error("self-test mutation anchor already exists: standalone absence receipt");
        }
        bundle.standalone_absence_receipt = { status: "UNRECEIPTED" };
      });
    },
  },
  {
    id: "p07-nested-execution-receipt-cycle",
    expectedCode: "P07_RECEIPT_CYCLE",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-execution.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        if (Object.hasOwn(schema.properties ?? {}, "evidence")) {
          throw new Error("self-test mutation anchor already exists: ContractExecution evidence");
        }
        schema.required.push("evidence");
        schema.properties.evidence = { $ref: "common.schema.json#/$defs/ReceiptReference" };
      });
      const exampleFile = path.join(root, "spec/examples/v1/contract-execution.valid.json");
      rewriteJsonObject(exampleFile, (execution) => {
        execution.evidence = {
          authority: "didrun",
          grade_verbatim: "TREE-EXACT",
          commit_oid: "self-test",
          command_digest: "sha256:abababababababababababababababababababababababababababababababab",
        };
      });
    },
  },
  {
    id: "p07-determinism-flag-rebinding",
    expectedCode: "P07_BUNDLE_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const property = schema.properties?.determinism_profile?.properties?.emitter_introduces_time;
        if (property?.const !== false) {
          throw new Error("self-test mutation anchor is absent: false emitter time nonclaim");
        }
        property.const = true;
      });
      const exampleFile = path.join(root, "spec/examples/v1/contract-bundle.valid.json");
      rewriteJsonObject(exampleFile, (bundle) => {
        if (bundle.determinism_profile?.emitter_introduces_time !== false) {
          throw new Error("self-test mutation anchor is absent: false example emitter time nonclaim");
        }
        bundle.determinism_profile.emitter_introduces_time = true;
      });
    },
  },
  {
    id: "p07-bundle-constant-coordinated-drift",
    expectedCode: "P07_BUNDLE_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const property = schema.properties?.runtime_dependency_profile;
        if (property?.const !== "NODE_CORE_ONLY_V1") {
          throw new Error("self-test mutation anchor is absent: bundle runtime dependency constant");
        }
        property.const = "NODE_WITH_PACKAGES_V1";
      });
      const exampleFile = path.join(root, "spec/examples/v1/contract-bundle.valid.json");
      rewriteJsonObject(exampleFile, (bundle) => {
        if (bundle.runtime_dependency_profile !== "NODE_CORE_ONLY_V1") {
          throw new Error("self-test mutation anchor is absent: example runtime dependency constant");
        }
        bundle.runtime_dependency_profile = "NODE_WITH_PACKAGES_V1";
      });
    },
  },
  {
    id: "p07-bundle-tuple-item-type-widening",
    expectedCode: "P07_BUNDLE_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const items = schema.properties?.predicate?.properties?.allowed_tuples?.items;
        if (items?.$ref !== "common.schema.json#/$defs/ExactTuple") {
          throw new Error("self-test mutation anchor is absent: allowed tuple exact type");
        }
        delete items.$ref;
        items.type = "object";
      });
    },
  },
  {
    id: "p07-bundle-file-contains-shrinkage",
    expectedCode: "P07_BUNDLE_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const allOf = schema.properties?.files?.allOf;
        if (!Array.isArray(allOf) || allOf.length !== P07_FILE_ROSTER.length) {
          throw new Error("self-test mutation anchor is absent: exact file contains roster");
        }
        allOf.pop();
      });
    },
  },
  {
    id: "p07-custom-expectation-cardinality-drift",
    expectedCode: "P07_BUNDLE_AUTHORITY",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const cardinality = schema.allOf?.[0]?.then?.properties?.predicate
          ?.properties?.allowed_tuples;
        if (cardinality?.minItems !== 1 || cardinality?.maxItems !== 1) {
          throw new Error("self-test mutation anchor is absent: custom expectation exact-one tuple");
        }
        cardinality.maxItems = 4;
      });
    },
  },
  {
    id: "p07-false-receipt-nonclaim-rebinding",
    expectedCode: "P07_RECEIPT_CYCLE",
    mutate(root) {
      const schemaFile = path.join(root, "spec/schema/v1/contract-bundle.schema.json");
      rewriteJsonObject(schemaFile, (schema) => {
        const property = schema.properties?.determinism_profile?.properties?.emitter_introduces_execution_receipt;
        if (property?.const !== false) {
          throw new Error("self-test mutation anchor is absent: false execution-receipt nonclaim");
        }
        property.const = true;
      });
      const exampleFile = path.join(root, "spec/examples/v1/contract-bundle.valid.json");
      rewriteJsonObject(exampleFile, (bundle) => {
        if (bundle.determinism_profile?.emitter_introduces_execution_receipt !== false) {
          throw new Error("self-test mutation anchor is absent: false example execution-receipt nonclaim");
        }
        bundle.determinism_profile.emitter_introduces_execution_receipt = true;
      });
    },
  },
  {
    id: "p07-transitive-receipt-alias",
    expectedCode: "P07_RECEIPT_CYCLE",
    mutate(root) {
      const commonFile = path.join(root, "spec/schema/v1/common.schema.json");
      rewriteJsonObject(commonFile, (schema) => {
        if (Object.hasOwn(schema.$defs ?? {}, "ValidatorReceiptAlias")) {
          throw new Error("self-test mutation anchor already exists: ValidatorReceiptAlias");
        }
        schema.$defs.ValidatorReceiptAlias = { $ref: "#/$defs/ReceiptReference" };
      });
      const executionFile = path.join(root, "spec/schema/v1/contract-execution.schema.json");
      rewriteJsonObject(executionFile, (schema) => {
        if (Object.hasOwn(schema.properties ?? {}, "validator_optional_evidence")) {
          throw new Error("self-test mutation anchor already exists: validator_optional_evidence");
        }
        schema.properties.validator_optional_evidence = {
          $ref: "common.schema.json#/$defs/ValidatorReceiptAlias",
        };
      });
    },
  },
  {
    id: "p07-embedded-decision-receipt-cycle",
    expectedCode: "P07_RECEIPT_CYCLE",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-bundle.valid.json");
      rewriteJsonObject(file, (bundle) => {
        const decisionEntry = bundle.files?.find((entry) => entry.path === "decision.json");
        if (!decisionEntry || typeof decisionEntry.content_base64 !== "string") {
          throw new Error("self-test mutation anchor is absent: embedded decision.json");
        }
        const decision = JSON.parse(Buffer.from(decisionEntry.content_base64, "base64").toString("utf8"));
        decision.evidence = {
          authority: "didrun",
          grade_verbatim: "TREE-EXACT",
          commit_oid: "self-test",
          command_digest: "sha256:abababababababababababababababababababababababababababababababab",
        };
        replaceP07BundleMemberAndManifest(
          bundle,
          "decision.json",
          Buffer.from(`${JSON.stringify(decision)}\n`, "utf8"),
        );
      });
    },
  },
  {
    id: "p07-cross-schema-receipt-authority",
    expectedCode: "P07_RECEIPT_CYCLE",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/contract-execution.schema.json");
      rewriteJsonObject(file, (schema) => {
        if (Object.hasOwn(schema.properties ?? {}, "validator_optional_decision")) {
          throw new Error("self-test mutation anchor already exists: validator_optional_decision");
        }
        schema.properties.validator_optional_decision = { $ref: "decision-record.schema.json" };
      });
    },
  },
  {
    id: "p07-duplicate-tuple-field",
    expectedCode: "P07_TUPLE_FIELDS",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/contract-bundle.valid.json");
      rewriteJsonObject(file, (bundle) => {
        const fields = bundle.predicate?.allowed_tuples?.[0]?.fields;
        if (!Array.isArray(fields) || fields.length !== 1) {
          throw new Error("self-test mutation anchor is absent: one-field allowed tuple");
        }
        fields.push(structuredClone(fields[0]));
      });
    },
  },
  {
    id: "p07-negative-zero-exact-integer",
    expectedCode: "EXAMPLE_SCHEMA_VALIDATION",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/finalized-contract-run.valid.json");
      rewriteJsonObject(file, (finalizedRun) => {
        const field = finalizedRun.closed_run_witness?.observation?.observed_tuple?.fields?.[0];
        if (field?.value?.tag !== "BYTES" || field.value.base64 !== "b2sK") {
          throw new Error("self-test mutation anchor is absent: executable example BYTES value");
        }
        field.value = { tag: "INTEGER", canonical: "-0" };
      });
    },
  },
  {
    id: "p07-unsafe-clean-integer",
    expectedCode: "P07A_INTEGER_VALUE",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/finalized-contract-run.valid.json");
      rewriteJsonObject(file, (finalizedRun) => {
        const field = finalizedRun.closed_run_witness?.observation?.observed_tuple?.fields?.[0];
        if (field?.value?.tag !== "BYTES" || field.value.base64 !== "b2sK") {
          throw new Error("self-test mutation anchor is absent: executable example BYTES value");
        }
        field.value = { tag: "INTEGER", canonical: "9007199254740992" };
      });
    },
  },
  {
    id: "p07-unsafe-decision-compatibility-integer",
    expectedCode: "P07A_INTEGER_VALUE",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/decision-record.valid.json");
      rewriteJsonObject(file, (decision) => {
        const field = decision.allowed_complete_tuples?.[0]?.fields?.[0];
        if (field?.value?.tag !== "CANONICAL_JSON") {
          throw new Error("self-test mutation anchor is absent: DecisionRecord compatibility value");
        }
        field.value = {
          tag: "INTEGER",
          text: "9007199254740992",
          boolean: false,
          canonical_json_base64: "",
        };
      });
    },
  },
  {
    id: "p07-integer-profile-coordinated-widening",
    expectedCode: "P07A_INTEGER_AUTHORITY",
    mutate(root) {
      const commonFile = path.join(root, "spec/schema/v1/common.schema.json");
      rewriteJsonObject(commonFile, (schema) => {
        const integer = schema.$defs?.ExactValue?.oneOf
          ?.find((branch) => branch?.properties?.tag?.const === "INTEGER");
        if (integer?.properties?.canonical?.pattern !== P07_SAFE_INTEGER_PATTERN) {
          throw new Error("self-test mutation anchor is absent: common safe-integer pattern");
        }
        integer.properties.canonical.pattern = "^(?:0|[1-9][0-9]*|-[1-9][0-9]*)$";
      });
      const decisionFile = path.join(root, "spec/schema/v1/decision-record.schema.json");
      rewriteJsonObject(decisionFile, (schema) => {
        const integer = schema.$defs?.ExactValue?.allOf
          ?.find((branch) => branch?.if?.properties?.tag?.const === "INTEGER");
        if (integer?.then?.properties?.text?.pattern !== P07_SAFE_INTEGER_PATTERN) {
          throw new Error("self-test mutation anchor is absent: DecisionRecord safe-integer pattern");
        }
        integer.then.properties.text.pattern = "^(?:0|[1-9][0-9]*|-[1-9][0-9]*)$";
      });
    },
  },
  {
    id: "enum-authority-non-string-with-decoy",
    expectedCode: "ENUM_AUTHORITY_NON_STRING",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/contract-execution.schema.json");
      rewriteJsonObject(file, (schema) => {
        const authority = schema.properties?.result;
        if (!jsonEqual(authority?.enum, ["CONFORMS", "CONTRADICTS", "INELIGIBLE_EXECUTION"]) || Object.hasOwn(authority, "const")) {
          throw new Error("self-test mutation anchor is absent: flat contract execution authority enum");
        }
        authority.const = false;
      });
    },
  },
  {
    id: "enum-authority-duplicate-declaration",
    expectedCode: "ENUM_AUTHORITY_MULTIPLICITY",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/contract-execution.schema.json");
      rewriteJsonObject(file, (schema) => {
        const authority = schema.properties?.result;
        if (!jsonEqual(authority?.enum, ["CONFORMS", "CONTRADICTS", "INELIGIBLE_EXECUTION"]) || Object.hasOwn(authority, "const")) {
          throw new Error("self-test mutation anchor is absent: unique flat contract execution authority enum");
        }
        authority.const = "INELIGIBLE_EXECUTION";
      });
    },
  },
  {
    id: "unreachable-ref-to-nonschema",
    expectedCode: "SCHEMA_REF_INVALID",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/world-plan.schema.json");
      rewriteJsonObject(file, (schema) => {
        schema.$defs ??= {};
        if (Object.hasOwn(schema.$defs, "ValidatorUnreachableNonSchemaRef")) {
          throw new Error("self-test mutation anchor already exists: ValidatorUnreachableNonSchemaRef");
        }
        schema.$defs.ValidatorUnreachableNonSchemaRef = { $ref: "#/required/0" };
      });
    },
  },
  {
    id: "unreachable-ref-to-object-instance",
    expectedCode: "SCHEMA_REF_INVALID",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/world-plan.schema.json");
      rewriteJsonObject(file, (schema) => {
        schema.$defs ??= {};
        if (Object.hasOwn(schema.$defs, "ValidatorObjectValue") || Object.hasOwn(schema.$defs, "ValidatorObjectValueRef")) {
          throw new Error("self-test mutation anchor already exists: ValidatorObjectValue");
        }
        schema.$defs.ValidatorObjectValue = { const: { looks_like_a_schema: true } };
        schema.$defs.ValidatorObjectValueRef = { $ref: "#/$defs/ValidatorObjectValue/const" };
      });
    },
  },
  {
    id: "unreachable-ref-invalid-encoding",
    expectedCode: "SCHEMA_REF_INVALID",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/world-plan.schema.json");
      rewriteJsonObject(file, (schema) => {
        schema.$defs ??= {};
        if (Object.hasOwn(schema.$defs, "ValidatorUnreachableInvalidRef")) {
          throw new Error("self-test mutation anchor already exists: ValidatorUnreachableInvalidRef");
        }
        schema.$defs.ValidatorUnreachableInvalidRef = { $ref: "#/%ZZ" };
      });
    },
  },
  {
    id: "additional-example-property",
    expectedCode: "EXAMPLE_SCHEMA_VALIDATION",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/world-plan.valid.json");
      rewriteJsonObject(file, (value) => {
        const property = "unexpected_validator_self_test_property";
        if (Object.hasOwn(value, property)) throw new Error(`self-test mutation anchor already exists: ${property}`);
        value[property] = true;
      });
    },
  },
  {
    id: "unsupported-schema-assertion",
    expectedCode: "SCHEMA_KEYWORD_UNSUPPORTED",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/world-plan.schema.json");
      rewriteJsonObject(file, (schema) => {
        if (Object.hasOwn(schema, "minProperties")) throw new Error("self-test mutation anchor already exists: minProperties");
        schema.minProperties = 1;
      });
    },
  },
  {
    id: "unresolved-boolean-collapse",
    expectedCode: "ENUM_TERM_MISSING",
    mutate: collapseUnresolved,
  },
  {
    id: "reduction-run-strong-grade-smuggling",
    expectedCode: "EXAMPLE_SCHEMA_VALIDATION",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/reduction-run.valid.json");
      rewriteJsonObject(file, (value) => {
        if (value.grade !== "BEST_KNOWN") throw new Error("self-test mutation anchor is absent: weak ReductionRun grade");
        value.grade = "ONE_MINIMAL_UNDER";
      });
    },
  },
  {
    id: "reduction-grade-missing-completed-sweep",
    expectedCode: "EXAMPLE_SCHEMA_VALIDATION",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/reduction-grade.valid.json");
      rewriteJsonObject(file, (value) => {
        if (value.status !== "ONE_MINIMAL_UNDER" || typeof value.completed_sweep_digest !== "string") {
          throw new Error("self-test mutation anchor is absent: strong ReductionGrade completion");
        }
        value.completed_sweep_digest = "";
      });
    },
  },
  {
    id: "completed-sweep-cancelled-smuggling",
    expectedCode: "EXAMPLE_SCHEMA_VALIDATION",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/completed-sweep-draft.valid.json");
      rewriteJsonObject(file, (value) => {
        if (value.cancelled !== false) throw new Error("self-test mutation anchor is absent: completed sweep cancellation flag");
        value.cancelled = true;
      });
    },
  },
  {
    id: "reduction-transcript-partial-map-digests",
    expectedCode: "EXAMPLE_SCHEMA_VALIDATION",
    mutate(root) {
      const file = path.join(root, "spec/examples/v1/reduction-transcript.valid.json");
      rewriteJsonObject(file, (value) => {
        const entry = value.entries?.[0];
        if (entry?.decision !== "PRESERVES" || typeof entry.observed_preservation_map_digest !== "string") {
          throw new Error("self-test mutation anchor is absent: observed transcript map pair");
        }
        entry.observed_preservation_map_digest = "";
      });
    },
  },
  {
    id: "required-target-run-refusal-shrinkage",
    expectedCode: "REQUIRED_VECTOR_MISSING",
    mutate(root) {
      const file = path.join(root, "spec/vectors/v1/refusals.jsonl");
      const lines = fs.readFileSync(file, "utf8").trimEnd().split("\n");
      let removed = 0;
      const kept = lines.filter((line) => {
        const value = JSON.parse(line);
        if (value.id !== "target-run-mismatch") return true;
        removed += 1;
        return false;
      });
      if (removed !== 1) throw new Error(`self-test mutation anchor expected one target-run-mismatch row, found ${removed}`);
      fs.writeFileSync(file, `${kept.join("\n")}\n`, "utf8");
    },
  },
  {
    id: "symlink-acceptance",
    expectedCode: "LEGACY_CLAIM",
    mutate(root) {
      const file = path.join(root, "docs/CONCEPT_BRIEF.md");
      const anchor = "Git modes `100644` and `100755` only.";
      replaceAnchor(file, anchor, `${anchor} Symlink support is accepted.`);
    },
  },
  {
    id: "reject-all-compilation",
    expectedCode: "LEGACY_CLAIM",
    mutate(root) {
      const file = path.join(root, "docs/CONCEPT_BRIEF.md");
      replaceAnchor(file, "`REJECT_ALL` emits no code.", "`REJECT_ALL` emits executable code.");
    },
  },
  {
    id: "ordinal-group-authority",
    expectedCode: "LEGACY_CLAIM",
    mutate(root) {
      const file = path.join(root, "docs/CONCEPT_BRIEF.md");
      replaceAnchor(file, "Display clusters are derived views with no identity.", "Ordinal group IDs are the semantic authority.");
    },
  },
  {
    id: "unclosed-markdown-fence",
    expectedCode: "MARKDOWN_FENCE_UNCLOSED",
    mutate(root) {
      const file = path.join(root, "docs/CONCEPT_BRIEF.md");
      fs.appendFileSync(file, "\n```text\nvalidator self-test leaves this fence open\n", "utf8");
    },
  },
];

function runSelfTests(root) {
  const baseline = validate(root);
  if (baseline.problems.length > 0) {
    return {
      baseline,
      failures: ["baseline planning inputs are invalid; mutation results would be meaningless"],
      detected: [],
      schemaRuntimeIntersection: undefined,
    };
  }

  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "countershape-planning-validator-"));
  const failures = [];
  const detected = [];
  let schemaRuntimeIntersection;
  try {
    try {
      schemaRuntimeIntersection = proveC1SchemaRuntimeIntersection(root);
    } catch (error) {
      failures.push(`c1-schema-runtime-intersection: ${error instanceof Error ? error.message : String(error)}`);
    }
    for (const test of SELF_TESTS) {
      const testRoot = path.join(temporary, test.id);
      copyRelevantInputs(root, testRoot);
      try {
        test.mutate(testRoot);
      } catch (error) {
        failures.push(`${test.id}: ${error instanceof Error ? error.message : String(error)}`);
        continue;
      }
      const result = validate(testRoot);
      if (result.problems.length === 0) {
        failures.push(`${test.id}: mutation was not detected`);
      } else if (!result.problems.some((problem) => problem.code === test.expectedCode)) {
        failures.push(`${test.id}: expected ${test.expectedCode}, got ${[...new Set(result.problems.map((problem) => problem.code))].sort().join(", ")}`);
      } else {
        detected.push(test.id);
      }
    }
  } finally {
    fs.rmSync(temporary, { recursive: true, force: true });
  }
  return { baseline, failures, detected, schemaRuntimeIntersection };
}

function printProblems(result) {
  for (const problem of result.problems) {
    const location = problem.line > 0 ? `${problem.file}:${problem.line}` : problem.file;
    process.stderr.write(`${location} [${problem.code}] ${problem.message}\n`);
  }
}

function main() {
  const arguments_ = process.argv.slice(2);
  if (arguments_.some((argument) => argument !== "--self-test") || arguments_.filter((argument) => argument === "--self-test").length > 1) {
    process.stderr.write("usage: node tools/validate-planning.mjs [--self-test]\n");
    process.exitCode = 2;
    return;
  }

  if (arguments_.includes("--self-test")) {
    const result = runSelfTests(REPOSITORY_ROOT);
    if (result.baseline.problems.length > 0) printProblems(result.baseline);
    for (const failure of result.failures) process.stderr.write(`self-test [FAILED] ${failure}\n`);
    if (result.failures.length > 0) {
      process.stderr.write(`planning validator self-test: failed (${result.detected.length}/${SELF_TESTS.length} mutations detected)\n`);
      process.exitCode = 1;
      return;
    }
    process.stdout.write(
      `planning validator self-test: ok (${result.detected.length}/${SELF_TESTS.length} mutations detected; `
      + `${result.schemaRuntimeIntersection.cases} schema-valid/runtime-invalid cases proven; `
      + `${result.schemaRuntimeIntersection.corpusDigest})\n`,
    );
    return;
  }

  const result = validate(REPOSITORY_ROOT);
  if (result.problems.length > 0) {
    printProblems(result);
    process.stderr.write(`planning validation: failed (${result.problems.length} problems)\n`);
    process.exitCode = 1;
    return;
  }
  const { schemas, examples, vectors, markdown } = result.counts;
  process.stdout.write(`planning validation: ok (${schemas} schemas, ${examples} examples, ${vectors} vectors, ${markdown} Markdown files)\n`);
}

try {
  main();
} catch (error) {
  process.stderr.write(`planning validator: ${error instanceof Error ? error.stack ?? error.message : String(error)}\n`);
  process.exitCode = 1;
}
