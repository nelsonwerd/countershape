#!/usr/bin/env node

import fs from "node:fs";
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
  "ContractExecution|contract-execution|contract-execution",
]);

const actualObjectContract = OBJECTS.map((parts) => parts.join("|"));
if (
  actualObjectContract.length !== EXACT_OBJECT_CONTRACT.length
  || actualObjectContract.some((entry, index) => entry !== EXACT_OBJECT_CONTRACT[index])
) {
  throw new Error("planning object contract drifted from the exact 19-object obligation");
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
    { schema: "contract-execution.schema.json", pointer: "#/properties/result/oneOf/0/properties/conformance" },
    { schema: "contract-execution.schema.json", pointer: "#/properties/result/oneOf/1/properties/execution_class" },
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

function validateVectors(root, problems) {
  const vectorDirectory = path.join(root, "spec/vectors/v1");
  const files = sortedFiles(vectorDirectory, (file) => file.endsWith(".jsonl"));
  assertNoCaseFoldedFileCollisions(root, files, problems);
  const refusalFile = path.join(vectorDirectory, "refusals.jsonl");
  if (!fs.existsSync(refusalFile)) {
    add(problems, "REQUIRED_FILE_MISSING", "spec/vectors/v1/refusals.jsonl", "required refusal vector corpus is missing");
    return { files, rowCount: 0 };
  }

  const rows = [];
  for (const file of files) {
    for (const row of parseJsonLinesFile(root, file, problems)) rows.push({ ...row, file });
  }
  const refusalRows = rows.filter((row) => row.file === refusalFile);

  const byId = new Map();
  for (const row of rows) {
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
    id: "enum-authority-non-string-with-decoy",
    expectedCode: "ENUM_AUTHORITY_NON_STRING",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/contract-execution.schema.json");
      rewriteJsonObject(file, (schema) => {
        const authority = schema.properties?.result?.oneOf?.[1]?.properties?.execution_class;
        if (authority?.const !== "INELIGIBLE_EXECUTION" || Object.hasOwn(authority, "enum")) {
          throw new Error("self-test mutation anchor is absent: contract execution authority const");
        }
        authority.const = false;
        authority.enum = ["INELIGIBLE_EXECUTION"];
      });
    },
  },
  {
    id: "enum-authority-duplicate-declaration",
    expectedCode: "ENUM_AUTHORITY_MULTIPLICITY",
    mutate(root) {
      const file = path.join(root, "spec/schema/v1/contract-execution.schema.json");
      rewriteJsonObject(file, (schema) => {
        const authority = schema.properties?.result?.oneOf?.[1]?.properties?.execution_class;
        if (authority?.const !== "INELIGIBLE_EXECUTION" || Object.hasOwn(authority, "enum")) {
          throw new Error("self-test mutation anchor is absent: unique contract execution authority const");
        }
        authority.enum = ["INELIGIBLE_EXECUTION"];
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
    return { baseline, failures: ["baseline planning inputs are invalid; mutation results would be meaningless"], detected: [] };
  }

  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "countershape-planning-validator-"));
  const failures = [];
  const detected = [];
  try {
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
  return { baseline, failures, detected };
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
    process.stdout.write(`planning validator self-test: ok (${result.detected.length}/${SELF_TESTS.length} mutations detected)\n`);
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
