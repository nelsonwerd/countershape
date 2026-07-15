#!/usr/bin/env node

import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const REPOSITORY_ROOT = path.resolve(SCRIPT_DIR, "..");

const OBJECTS = [
  ["WorldPlan", "world-plan"],
  ["WorldInstance", "world-instance"],
  ["ComparisonEnvelope", "comparison-envelope"],
  ["CapturedObservation", "captured-observation"],
  ["StableBatch", "stable-batch"],
  ["OutcomeMap", "outcome-map"],
  ["ReductionRun", "reduction-run"],
  ["Choicepoint", "choicepoint"],
  ["DecisionRecord", "decision-record"],
  ["ContractBundle", "contract-bundle"],
  ["ContractExecution", "contract-execution"],
];

const REQUIRED_SCHEMA_FILES = [
  "common.schema.json",
  ...OBJECTS.map(([, stem]) => `${stem}.schema.json`),
];

const REQUIRED_EXAMPLE_FILES = OBJECTS.map(([, stem]) => `${stem}.valid.json`);

const ENUM_GROUPS = new Map([
  ["batch classification", ["OBSERVED_STABLE", "UNSTABLE", "UNCOMPARABLE", "INCOMPLETE"]],
  ["reduction decision", ["PRESERVES", "CHANGES", "UNRESOLVED"]],
  ["reduction grade", ["UNCHANGED", "BEST_KNOWN", "ONE_MINIMAL_UNDER"]],
  ["decision action", ["ALLOW_OBSERVED", "CUSTOM_EXPECTATION", "REJECT_ALL", "DEFER", "REFINE"]],
  ["contract execution", ["CONFORMS", "CONTRADICTS", "INELIGIBLE_EXECUTION"]],
  ["network mode", ["HOST_ALLOWED"]],
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

function schemaObjectNames(schema) {
  const names = new Set();
  walk(schema, (node) => {
    if (!node || typeof node !== "object" || Array.isArray(node)) return;
    if (typeof node.title === "string") names.add(normalizedObjectName(node.title));
    if (typeof node.$anchor === "string") names.add(normalizedObjectName(node.$anchor));
    for (const container of [node.$defs, node.definitions]) {
      if (container && typeof container === "object" && !Array.isArray(container)) {
        for (const key of Object.keys(container)) names.add(normalizedObjectName(key));
      }
    }
    for (const property of ["kind", "object_kind", "artifact_kind"]) {
      const constant = node.properties?.[property]?.const;
      if (typeof constant === "string") names.add(normalizedObjectName(constant));
    }
  });
  return names;
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
  return segment.replaceAll("~1", "/").replaceAll("~0", "~");
}

function pointerExists(value, fragment) {
  if (fragment === "" || fragment === "#") return true;
  if (!fragment.startsWith("#/")) return false;
  let cursor = value;
  for (const rawSegment of fragment.slice(2).split("/")) {
    const segment = decodePointerSegment(rawSegment);
    if (!cursor || typeof cursor !== "object" || !Object.hasOwn(cursor, segment)) return false;
    cursor = cursor[segment];
  }
  return true;
}

function validateSchemaReferences(root, schemaFiles, schemasByFile, problems) {
  for (const file of schemaFiles) {
    const schema = schemasByFile.get(file);
    if (!schema) continue;
    walk(schema, (node, pointer) => {
      if (!node || typeof node !== "object" || Array.isArray(node) || typeof node.$ref !== "string") return;
      const [targetPart, fragmentPart = ""] = node.$ref.split("#", 2);
      if (/^[a-z][a-z0-9+.-]*:/iu.test(targetPart)) return;
      const target = targetPart === "" ? file : path.resolve(path.dirname(file), targetPart);
      const targetSchema = schemasByFile.get(target);
      if (!targetSchema) {
        add(problems, "SCHEMA_REF_MISSING", relative(root, file), `${pointer}/$ref targets missing schema ${node.$ref}`);
        return;
      }
      const fragment = fragmentPart === "" ? "" : `#${fragmentPart}`;
      if (!pointerExists(targetSchema, fragment)) {
        add(problems, "SCHEMA_REF_POINTER_MISSING", relative(root, file), `${pointer}/$ref targets missing pointer ${node.$ref}`);
      }
    });
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
  const discoveredObjects = new Set();
  const allEnums = [];
  const termsByFile = new Map();

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
    if (typeof schema.$id !== "string" || schema.$id.length === 0) {
      add(problems, "SCHEMA_ID_MISSING", relative(root, file), "schema must have a nonempty $id");
    } else if (identifiers.has(schema.$id)) {
      add(problems, "SCHEMA_ID_DUPLICATE", relative(root, file), `$id duplicates ${identifiers.get(schema.$id)}`);
    } else {
      identifiers.set(schema.$id, relative(root, file));
    }
    for (const name of schemaObjectNames(schema)) discoveredObjects.add(name);
    const fileTerms = constantValues(schema);
    for (const value of fileTerms) {
      if (FORBIDDEN_ENUM_VALUES.has(value)) {
        add(problems, "ENUM_FORBIDDEN_TERM", relative(root, file), `schema const contains prohibited enum term ${value}`);
      }
    }
    for (const entry of enumArrays(schema)) {
      allEnums.push({ ...entry, file });
      for (const value of entry.values) {
        if (typeof value === "string") fileTerms.add(value);
      }
    }
    termsByFile.set(file, fileTerms);
  }

  for (const [objectName] of OBJECTS) {
    if (!discoveredObjects.has(normalizedObjectName(objectName))) {
      add(problems, "SCHEMA_OBJECT_MISSING", "spec/schema/v1", `no schema definition declares ${objectName}`);
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

  for (const [group, requiredValues] of ENUM_GROUPS) {
    // JSON Schema represents tagged unions either as one enum or as oneOf
    // branches with const discriminants. All terms must live in one schema so
    // unrelated constants cannot accidentally satisfy an enum contract.
    const represented = [...termsByFile.values()]
      .some((terms) => requiredValues.every((value) => terms.has(value)));
    if (!represented) {
      add(problems, "ENUM_TERM_MISSING", "spec/schema/v1", `${group} must declare exactly named terms: ${requiredValues.join(", ")}`);
    }
  }

  validateSchemaReferences(root, files, schemasByFile, problems);
  return { files, schemasByFile };
}

function exampleObjectName(value, file) {
  for (const key of ["kind", "object_kind", "artifact_kind"]) {
    if (typeof value?.[key] === "string") return normalizedObjectName(value[key]);
  }
  return normalizedObjectName(path.basename(file).replace(/\.valid\.json$/u, ""));
}

function validateExamples(root, problems) {
  const exampleDirectory = path.join(root, "spec/examples/v1");
  const files = sortedFiles(exampleDirectory, (file) => file.endsWith(".json"));
  assertNoCaseFoldedFileCollisions(root, files, problems);
  const represented = new Set();

  for (const required of REQUIRED_EXAMPLE_FILES) {
    if (!fs.existsSync(path.join(exampleDirectory, required))) {
      add(problems, "REQUIRED_FILE_MISSING", `spec/examples/v1/${required}`, "required valid example is missing");
    }
  }

  for (const file of files) {
    const value = parseJsonFile(root, file, problems);
    if (!value || typeof value !== "object" || Array.isArray(value)) {
      if (value !== null) add(problems, "EXAMPLE_ROOT_TYPE", relative(root, file), "example root must be an object");
      continue;
    }
    represented.add(exampleObjectName(value, file));
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

function markdownWithoutFences(text) {
  let inFence = false;
  return text.split(/\r?\n/u).map((line) => {
    if (/^\s*(```|~~~)/u.test(line)) {
      inFence = !inFence;
      return "";
    }
    return inFence ? "" : line;
  }).join("\n");
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
    const text = markdownWithoutFences(fs.readFileSync(file, "utf8"));
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
    const lines = fs.readFileSync(file, "utf8").split(/\r?\n/u);
    let allowDepth = 0;
    let inFence = false;
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
      if (/^\s*(```|~~~)/u.test(line)) {
        inFence = !inFence;
        continue;
      }
      const heading = /^#{1,6}\s+(.+)$/u.exec(line);
      if (heading) section = heading[1];
      if (allowDepth > 0 || inFence || NEGATIVE_SECTION.test(section)) continue;
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
  const examples = validateExamples(root, problems);
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
    id: "unresolved-boolean-collapse",
    expectedCode: "ENUM_TERM_MISSING",
    mutate: collapseUnresolved,
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
