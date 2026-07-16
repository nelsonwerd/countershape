#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { constants } from "node:fs";
import { lstat, open, readdir } from "node:fs/promises";
import { dirname, extname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const modulePath = fileURLToPath(import.meta.url);
const root = resolve(dirname(modulePath), "..");
const modulePrefix = "github.com/nelsonwerd/countershape/";

// Every member below these roots participates. Subdirectories are an exact
// package map rather than permissive prefixes: a newly invented authority path
// closes the gate until its jurisdiction is reviewed here.
const reviewedRoots = Object.freeze([
  ["internal/domain", { ".": "domain" }],
  ["internal/compare", { ".": "compare" }],
  ["internal/observe", { ".": "observe" }],
  ["internal/reduce", { ".": "reduce" }],
  ["internal/reduction", { ".": "reduction" }],
  ["internal/store", { ".": "store" }],
  ["internal/confirmation", {
    ".": "confirmation",
    authority: "authority",
    "internal/publication": "publication",
  }],
	["internal/portablevalue", { ".": "portablevalue" }],
	["internal/projectionprofile", { ".": "projectionprofile" }],
	["internal/projectiontranslate", { ".": "projectiontranslate" }],
  ["internal/choice", {
    ".": "choice",
    promotion: "promotion",
    "promotion/authority": "authority",
    "promotion/internal/publication": "publication",
  }],
  ["internal/world", { ".": "world" }],
]);

const jsonAuthorities = Object.freeze([
  "spec/schema/v1/common.schema.json",
  "spec/schema/v1/choicepoint.schema.json",
  "spec/schema/v1/decision-record.schema.json",
  "spec/examples/v1/choicepoint.valid.json",
  "spec/examples/v1/decision-record.valid.json",
]);

const manifestToolSource = String.raw`
const { createHash } = require("node:crypto");
const { closeSync, constants, fstatSync, lstatSync, openSync, readFileSync } = require("node:fs");
const { isAbsolute, relative, resolve, sep } = require("node:path");
function slash(value) { return value.split(sep).join("/"); }
function fail(code, detail) { process.stderr.write(code + ": " + detail + "\n"); process.exit(1); }
const payload = JSON.parse(readFileSync(0, "utf8"));
if (payload.forceFailure === true) fail("U6_MANIFEST_TOOL_FORCED_FAILURE", "self-test fail-closed probe");
const digest = createHash("sha256");
for (const entry of payload.entries) {
  const absolute = resolve(payload.root, entry.path);
  const fromRoot = relative(payload.root, absolute);
  if (isAbsolute(fromRoot) || fromRoot === ".." || fromRoot.startsWith(".." + sep)) {
    fail("U6_MANIFEST_TOOL_PATH_ESCAPE", entry.path);
  }
  const before = lstatSync(absolute);
  if (before.isSymbolicLink() || !before.isFile()) fail("U6_MANIFEST_TOOL_NONREGULAR", entry.path);
  let descriptor;
  try {
    descriptor = openSync(absolute, constants.O_RDONLY | (constants.O_NOFOLLOW || 0));
    const opened = fstatSync(descriptor);
    if (!opened.isFile() || opened.dev !== before.dev || opened.ino !== before.ino || opened.size !== before.size) {
      fail("U6_MANIFEST_TOOL_FILE_CHANGED", entry.path);
    }
    const bytes = readFileSync(descriptor);
    const after = fstatSync(descriptor);
    if (after.dev !== opened.dev || after.ino !== opened.ino || after.size !== opened.size) {
      fail("U6_MANIFEST_TOOL_FILE_CHANGED", entry.path);
    }
    const sha256 = createHash("sha256").update(bytes).digest("hex");
    if (bytes.length !== entry.bytes || sha256 !== entry.sha256) fail("U6_MANIFEST_TOOL_DIGEST_MISMATCH", entry.path);
    digest.update(slash(entry.path)); digest.update("\0"); digest.update(bytes); digest.update("\0");
  } finally { if (descriptor !== undefined) closeSync(descriptor); }
}
const actual = digest.digest("hex");
if (actual !== payload.digest) fail("U6_MANIFEST_TOOL_AGGREGATE_MISMATCH", actual + " != " + payload.digest);
process.stdout.write(actual + "\n");
`;

class ArchitectureError extends Error {
  constructor(code, detail) {
    super(`${code}: ${detail}`);
    this.code = code;
  }
}

function slashPath(value) { return value.split(sep).join("/"); }

function sameStat(left, right) {
  return left.dev === right.dev && left.ino === right.ino && left.size === right.size &&
    left.mode === right.mode && left.mtimeMs === right.mtimeMs;
}

async function readRegularNoFollow(absolute, relativePath, expected) {
  let handle;
  try {
    handle = await open(absolute, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
    const opened = await handle.stat();
    if (!opened.isFile() || !sameStat(opened, expected)) {
      throw new ArchitectureError("U6_MANIFEST_FILE_CHANGED", relativePath);
    }
    const bytes = await handle.readFile();
    const after = await handle.stat();
    if (!sameStat(opened, after)) throw new ArchitectureError("U6_MANIFEST_FILE_CHANGED", relativePath);
    return bytes;
  } catch (error) {
    if (error instanceof ArchitectureError) throw error;
    throw new ArchitectureError("U6_MANIFEST_READ_FAILED", `${relativePath}: ${error.code ?? error.message}`);
  } finally {
    await handle?.close();
  }
}

function decodeUTF8(bytes, path) {
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    throw new ArchitectureError("U6_MANIFEST_INVALID_UTF8", path);
  }
}

// Produce two stable lexical views without pretending to be a full Go parser.
// code preserves identifiers/operators/braces while replacing comments and
// literals with spaces; commentless preserves literals for import and struct-
// tag parsing while removing both comment forms. Newlines stay aligned.
function lexicalViews(source, relativePath) {
  // Index by UTF-16 code units because String.length/slice do. Code-point
  // arrays would desynchronize every later source offset after an astral rune.
  const code = source.split("");
  const commentless = source.split("");
  const literals = [];
  let literal = "";
  let state = "code";
  for (let index = 0; index < source.length; index += 1) {
    const character = source[index];
    const next = source[index + 1] ?? "";
    if (state === "code") {
      if (character === "/" && next === "/") {
        code[index] = code[index + 1] = " ";
        commentless[index] = commentless[index + 1] = " ";
        index += 1;
        state = "line-comment";
      } else if (character === "/" && next === "*") {
        code[index] = code[index + 1] = " ";
        commentless[index] = commentless[index + 1] = " ";
        index += 1;
        state = "block-comment";
      } else if (character === '"' || character === "'" || character === "`") {
        code[index] = " ";
        literal = "";
        state = character === '"' ? "string" : character === "'" ? "rune" : "raw";
      }
    } else if (state === "line-comment") {
      code[index] = character === "\n" ? "\n" : " ";
      commentless[index] = character === "\n" ? "\n" : " ";
      if (character === "\n") state = "code";
    } else if (state === "block-comment") {
      code[index] = character === "\n" ? "\n" : " ";
      commentless[index] = character === "\n" ? "\n" : " ";
      if (character === "*" && next === "/") {
        code[index + 1] = commentless[index + 1] = " ";
        index += 1;
        state = "code";
      }
    } else if (state === "raw") {
      code[index] = character === "\n" ? "\n" : " ";
      if (character === "`") {
        literals.push(literal);
        state = "code";
      } else {
        literal += character;
      }
    } else {
      code[index] = character === "\n" ? "\n" : " ";
      if (character === "\\") {
        literal += character + next;
        index += 1;
        if (index < code.length) code[index] = source[index] === "\n" ? "\n" : " ";
      } else if ((state === "string" && character === '"') || (state === "rune" && character === "'")) {
        literals.push(literal);
        state = "code";
      } else {
        literal += character;
      }
    }
  }
  if (state === "line-comment") state = "code";
  if (state !== "code") throw new ArchitectureError("U6_GO_LEXICAL_INVALID", `${relativePath}: unterminated ${state}`);
  return { code: code.join(""), commentless: commentless.join(""), literals };
}

function importedPackages(commentless) {
  const imports = [];
  const declarations = /(?:^|\n)\s*import\s*(?:\(([\s\S]*?)\)|(?:([._A-Za-z][._A-Za-z0-9]*)\s+)?"([^"]+)")/gu;
  for (const declaration of commentless.matchAll(declarations)) {
    if (declaration[3]) {
      imports.push({ alias: declaration[2] ?? "", path: declaration[3] });
      continue;
    }
    for (const spec of declaration[1].matchAll(/(?:^|\s)(?:([._A-Za-z][._A-Za-z0-9]*)\s+)?"([^"]+)"/gu)) {
      imports.push({ alias: spec[1] ?? "", path: spec[2] });
    }
  }
  return imports;
}

function balancedBlock(code, openIndex) {
  if (openIndex < 0 || code[openIndex] !== "{") return null;
  let depth = 0;
  for (let index = openIndex; index < code.length; index += 1) {
    if (code[index] === "{") depth += 1;
    if (code[index] === "}") {
      depth -= 1;
      if (depth === 0) return { body: code.slice(openIndex + 1, index), end: index + 1 };
    }
  }
  return null;
}

function functionRegion(code, signature) {
  const match = signature.exec(code);
  if (!match) return null;
  const open = code.indexOf("{", match.index + match[0].length);
  const block = balancedBlock(code, open);
  if (!block) return null;
  return { body: block.body, start: open + 1, end: block.end - 1 };
}

function functionBody(code, signature) { return functionRegion(code, signature)?.body ?? null; }

function compactCode(code) { return code.replace(/\s+/gu, " ").trim(); }

async function walkPackageRoot(rootPath, packageMap, entries) {
  const absoluteRoot = resolve(root, rootPath);
  const rootStat = await lstat(absoluteRoot);
  if (rootStat.isSymbolicLink() || !rootStat.isDirectory()) {
    throw new ArchitectureError("U6_MANIFEST_ROOT_NONDIRECTORY", rootPath);
  }
  const visit = async (absoluteDirectory) => {
    const directoryRelative = slashPath(relative(absoluteRoot, absoluteDirectory)) || ".";
    const expectedPackage = packageMap[directoryRelative];
    const container = !expectedPackage && Object.keys(packageMap).some((candidate) => candidate.startsWith(`${directoryRelative}/`));
    if (!expectedPackage && !container) {
      throw new ArchitectureError("U6_MANIFEST_UNKNOWN_SUBPACKAGE", `${rootPath}/${directoryRelative}`);
    }
    const children = await readdir(absoluteDirectory, { withFileTypes: true });
    children.sort((left, right) => left.name.localeCompare(right.name, "en"));
    if (children.length === 0) throw new ArchitectureError("U6_MANIFEST_EMPTY_PACKAGE", `${rootPath}/${directoryRelative}`);
    for (const child of children) {
      const absolute = join(absoluteDirectory, child.name);
      const relativePath = slashPath(relative(root, absolute));
      const childStat = await lstat(absolute);
      if (childStat.isSymbolicLink()) throw new ArchitectureError("U6_MANIFEST_SYMLINK", relativePath);
      if (childStat.isDirectory()) {
        await visit(absolute);
        continue;
      }
      if (container) throw new ArchitectureError("U6_MANIFEST_FILE_IN_CONTAINER", relativePath);
      if (!childStat.isFile()) throw new ArchitectureError("U6_MANIFEST_NONREGULAR", relativePath);
      if (extname(child.name) !== ".go") throw new ArchitectureError("U6_MANIFEST_UNKNOWN_EXTENSION", relativePath);
      const bytes = await readRegularNoFollow(absolute, relativePath, childStat);
      const source = decodeUTF8(bytes, relativePath);
      const lexical = lexicalViews(source, relativePath);
      const match = lexical.code.match(/^\s*package\s+([A-Za-z_][A-Za-z0-9_]*)\s*$/mu);
      const allowed = child.name.endsWith("_test.go") ? [expectedPackage, `${expectedPackage}_test`] : [expectedPackage];
      if (!match || !allowed.includes(match[1])) {
        throw new ArchitectureError("U6_MANIFEST_PACKAGE_MISMATCH", `${relativePath}: ${match?.[1] ?? "absent"}`);
      }
      entries.push({ path: relativePath, kind: "go", bytes, source, lexical });
    }
  };
  await visit(absoluteRoot);
}

async function readManifest() {
  const entries = [];
  for (const [rootPath, packageMap] of reviewedRoots) await walkPackageRoot(rootPath, packageMap, entries);
  for (const relativePath of jsonAuthorities) {
    const absolute = resolve(root, relativePath);
    const stat = await lstat(absolute);
    if (stat.isSymbolicLink() || !stat.isFile()) throw new ArchitectureError("U6_MANIFEST_JSON_NONREGULAR", relativePath);
    const bytes = await readRegularNoFollow(absolute, relativePath, stat);
    entries.push({ path: relativePath, kind: "json", bytes, source: decodeUTF8(bytes, relativePath) });
  }
  entries.sort((left, right) => left.path.localeCompare(right.path, "en"));
  const digest = createHash("sha256");
  for (const entry of entries) {
    digest.update(entry.path); digest.update("\0"); digest.update(entry.bytes); digest.update("\0");
    entry.sha256 = createHash("sha256").update(entry.bytes).digest("hex");
  }
  return { entries, digest: digest.digest("hex") };
}

function verifyIndependentManifest(manifest) {
  const result = spawnSync(process.execPath, ["-e", manifestToolSource], {
    input: JSON.stringify({
      root,
      digest: manifest.digest,
      forceFailure: process.env.COUNTERSHAPE_U6_ARCHITECTURE_FORCE_TOOL_FAILURE === "1",
      entries: manifest.entries.map((entry) => ({ path: entry.path, bytes: entry.bytes.length, sha256: entry.sha256 })),
    }),
    encoding: "utf8",
    env: { NO_COLOR: "1", TZ: "UTC" },
    timeout: 30_000,
    maxBuffer: 512 * 1024,
  });
  if (result.error || result.signal || result.status !== 0 || result.stdout.trim() !== manifest.digest) {
    throw new ArchitectureError(
      "U6_MANIFEST_TOOL_FAILED",
      result.error?.message ?? result.signal ?? (result.stderr.trim() || `exit ${result.status}`),
    );
  }
}

function runInheritedU5Checker() {
  const checker = resolve(root, "tools/check-u5-architecture.mjs");
  const result = spawnSync(process.execPath, [checker], {
    cwd: root,
    encoding: "utf8",
    env: { NO_COLOR: "1", TZ: "UTC" },
    timeout: 30_000,
    maxBuffer: 512 * 1024,
  });
  if (result.error || result.signal || result.status !== 0) {
    throw new ArchitectureError(
      "U6_INHERITED_U5_FAILED",
      result.error?.message ?? result.signal ?? (result.stderr.trim() || result.stdout.trim() || `exit ${result.status}`),
    );
  }
}

function productionEntries(manifest) {
  return manifest.entries.filter((entry) => entry.kind === "go" && !entry.path.endsWith("_test.go"));
}

function entryAt(manifest, path) {
  const found = manifest.entries.find((entry) => entry.path === path);
  if (!found) throw new ArchitectureError("U6_REQUIRED_FILE_ABSENT", path);
  return found;
}

function sourceAt(manifest, path) { return entryAt(manifest, path).source; }
function codeAt(manifest, path) { return entryAt(manifest, path).lexical?.code ?? ""; }
function commentlessAt(manifest, path) { return entryAt(manifest, path).lexical?.commentless ?? sourceAt(manifest, path); }
function literalsAt(manifest, path) { return entryAt(manifest, path).lexical?.literals ?? []; }

function requireIncludes(violations, source, needle, code, path) {
  if (!source.includes(needle)) violations.push([code, path]);
}

function requireExcludes(violations, source, needle, code, path) {
  if (source.includes(needle)) violations.push([code, path]);
}

function structJSONTags(source, name) {
  const start = source.indexOf(`type ${name} struct {`);
  if (start < 0) return null;
  const end = source.indexOf("\n}", start);
  if (end < 0) return null;
  return [...source.slice(start, end).matchAll(/`json:"([^",]+)(?:,[^"]*)?"`/gu)].map((match) => match[1]);
}

function exactSet(left, right) {
  return left.length === right.length && [...left].sort().every((value, index) => value === [...right].sort()[index]);
}

function topLevelStructBodies(code, typeName) {
  const escaped = typeName.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
  const pattern = new RegExp(`\\btype\\s+${escaped}\\s+struct\\s*\\{`, "gu");
  const bodies = [];
  for (const match of code.matchAll(pattern)) {
    const open = match.index + match[0].lastIndexOf("{");
    const block = balancedBlock(code, open);
    if (block) bodies.push(block.body);
  }
  return bodies;
}

function exportedStructFields(body) {
  return [...body.matchAll(/^\s*([A-Z][A-Za-z0-9_]*)\s+/gmu)].map((match) => match[1]);
}

function embedsAuthorityInterface(code) {
  if (/\btype\s+[A-Za-z_][A-Za-z0-9_]*\s*(?:=\s*)?(?:CompileEligibility|CompilableRuling|NoncompilableRuling)\b/u.test(code)) {
    return true;
  }
  for (const match of code.matchAll(/\b(?:struct|interface)\s*\{/gu)) {
    const open = match.index + match[0].lastIndexOf("{");
    const body = balancedBlock(code, open)?.body ?? "";
    if (/^\s*(?:CompileEligibility|CompilableRuling|NoncompilableRuling)\s*$/gmu.test(body)) return true;
  }
  return false;
}

const exactInternalImports = Object.freeze(new Map([
  ["internal/domain", ["internal/canon"]],
  ["internal/compare", ["internal/canon", "internal/domain", "internal/observe"]],
  ["internal/observe", ["internal/canon", "internal/domain", "internal/world"]],
  ["internal/reduce", ["internal/canon", "internal/compare", "internal/domain"]],
  ["internal/reduction", ["internal/canon", "internal/domain", "internal/reduce", "internal/store"]],
  ["internal/store", ["internal/canon", "internal/choice/promotion/authority", "internal/compare", "internal/confirmation/authority", "internal/domain", "internal/reduce"]],
  ["internal/confirmation", ["internal/canon", "internal/compare", "internal/confirmation/authority", "internal/confirmation/internal/publication", "internal/domain", "internal/observe", "internal/reduce", "internal/reduction", "internal/world"]],
  ["internal/confirmation/authority", ["internal/confirmation/internal/publication"]],
  ["internal/confirmation/internal/publication", ["internal/canon", "internal/domain"]],
	["internal/portablevalue", ["internal/canon"]],
	["internal/projectionprofile", ["internal/canon", "internal/domain", "internal/portablevalue"]],
	["internal/projectiontranslate", ["internal/adapters/cli", "internal/adapters/http", "internal/canon", "internal/compare", "internal/domain", "internal/portablevalue", "internal/projectionprofile"]],
  ["internal/choice", ["internal/canon", "internal/compare", "internal/confirmation", "internal/domain", "internal/reduce"]],
  ["internal/choice/promotion", ["internal/choice", "internal/choice/promotion/internal/publication", "internal/confirmation", "internal/domain", "internal/store"]],
  ["internal/choice/promotion/authority", ["internal/choice/promotion/internal/publication"]],
  ["internal/choice/promotion/internal/publication", ["internal/canon", "internal/domain"]],
  ["internal/world", ["internal/adapters/cli/model", "internal/adapters/http/model", "internal/canon", "internal/domain", "internal/gitobj"]],
]));

function inspectPackageBoundaries(manifest, violations) {
  const aggregate = new Map([...exactInternalImports.keys()].map((key) => [key, new Set()]));
  for (const entry of productionEntries(manifest)) {
    const imports = importedPackages(entry.lexical.commentless);
    const paths = imports.map((candidate) => candidate.path);
    if (imports.some((candidate) => candidate.alias === ".")) {
      violations.push(["U6_DOT_IMPORT_FORBIDDEN", entry.path]);
    }
    const forbiddenRuntime = ["net", "net/http", "os/exec"].find((candidate) => paths.includes(candidate));
    if ((entry.path.startsWith("internal/choice/") || entry.path.startsWith("internal/confirmation/")) && forbiddenRuntime) {
      violations.push(["U6_DECISION_LAYER_FORBIDDEN_RUNTIME_IMPORT", `${entry.path}: ${forbiddenRuntime}`]);
    }
    const internalImports = paths.filter((candidate) => candidate.startsWith(modulePrefix)).map((candidate) => candidate.slice(modulePrefix.length));
    const packageDirectory = entry.path.slice(0, entry.path.lastIndexOf("/"));
    if (!aggregate.has(packageDirectory)) violations.push(["U6_INTERNAL_PACKAGE_UNMAPPED", entry.path]);
    for (const imported of internalImports) aggregate.get(packageDirectory)?.add(imported);
		for (const imported of internalImports.filter((candidate) => candidate.startsWith("internal/adapters/"))) {
			const translatorOwned = packageDirectory === "internal/projectiontranslate" &&
				(imported === "internal/adapters/cli" || imported === "internal/adapters/http");
			const worldModelOwned = packageDirectory === "internal/world" &&
				(imported === "internal/adapters/cli/model" || imported === "internal/adapters/http/model");
			if (!translatorOwned && !worldModelOwned) {
				violations.push(["U6_ADAPTER_IMPORT_OUTSIDE_TRANSLATOR_OR_WORLD_MODEL", `${entry.path}: ${imported}`]);
			}
		}
    if (entry.path.startsWith("internal/choice/") && !entry.path.startsWith("internal/choice/promotion/") &&
        internalImports.some((candidate) => candidate === "internal/store" || candidate.startsWith("internal/choice/promotion"))) {
      violations.push(["U6_CHOICE_FORBIDDEN_AUTHORITY_IMPORT", entry.path]);
    }
    if (entry.path.startsWith("internal/confirmation/") && !entry.path.includes("/authority/") && !entry.path.includes("/internal/publication/") &&
        internalImports.some((candidate) => candidate === "internal/store" || candidate.startsWith("internal/choice"))) {
      violations.push(["U6_CONFIRMATION_FORBIDDEN_AUTHORITY_IMPORT", entry.path]);
    }
    if (entry.path.startsWith("internal/world/") &&
        internalImports.some((candidate) => candidate.startsWith("internal/choice") || candidate === "internal/store" || candidate.startsWith("internal/confirmation"))) {
      violations.push(["U6_WORLD_FORBIDDEN_DECISION_IMPORT", entry.path]);
    }
    if (entry.path.startsWith("internal/store/") && internalImports.some((candidate) =>
      (candidate.startsWith("internal/choice") && candidate !== "internal/choice/promotion/authority") ||
      (candidate.startsWith("internal/confirmation") && candidate !== "internal/confirmation/authority"))) {
      violations.push(["U6_STORE_FORBIDDEN_SEMANTIC_IMPORT", entry.path]);
    }
  }
  for (const [packageDirectory, expected] of exactInternalImports) {
    const actual = [...(aggregate.get(packageDirectory) ?? [])];
    if (!exactSet(actual, expected)) {
      violations.push(["U6_INTERNAL_IMPORT_LATTICE", `${packageDirectory}: ${actual.sort().join(",")}`]);
    }
  }
}

function inspectPortableAuthorityBoundaries(manifest, violations) {
	const profileImport = `${modulePrefix}internal/projectionprofile`;
	const externalReferences = [];
	for (const entry of productionEntries(manifest)) {
		const packageDirectory = entry.path.slice(0, entry.path.lastIndexOf("/"));
		if (packageDirectory === "internal/projectionprofile") continue;
		for (const imported of importedPackages(entry.lexical.commentless)) {
			if (imported.path !== profileImport || imported.alias === "_" || imported.alias === ".") continue;
			const alias = imported.alias || "projectionprofile";
			const escaped = alias.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
			const references = entry.lexical.code.match(new RegExp(`\\b${escaped}\\s*\\.\\s*NewDerived\\b`, "gu")) ?? [];
			for (let index = 0; index < references.length; index += 1) externalReferences.push(entry.path);
		}
	}
	if (externalReferences.length !== 1 || externalReferences[0] !== "internal/projectiontranslate/translate.go") {
		violations.push(["U6_PROFILE_DERIVATION_REFERENCE_SURFACE_NOT_EXACT", externalReferences.join(",") || "absent"]);
	}
	const translator = codeAt(manifest, "internal/projectiontranslate/translate.go");
	const resolverBody = functionBody(translator, /\bfunc\s+newResolvedProfile\s*\(/u) ?? "";
	if ((resolverBody.match(/\bprojectionprofile\s*\.\s*NewDerived\b/gu) ?? []).length !== 1) {
		violations.push(["U6_PROFILE_DERIVATION_OUTSIDE_RESOLVER", "internal/projectiontranslate/translate.go:newResolvedProfile"]);
	}
	let internalReferences = 0;
	let internalFiles = [];
	for (const entry of productionEntries(manifest).filter((candidate) => candidate.path.startsWith("internal/projectionprofile/"))) {
		const count = entry.lexical.code.match(/\bNewDerived\b/gu)?.length ?? 0;
		if (count > 0) internalFiles.push(entry.path);
		internalReferences += count;
	}
	const profile = codeAt(manifest, "internal/projectionprofile/profile.go");
	const validBody = functionBody(profile, /\bfunc\s*\(p\s+Profile\)\s+Valid\s*\(/u) ?? "";
	if (internalReferences !== 2 || !exactSet(internalFiles, ["internal/projectionprofile/profile.go"]) ||
		(validBody.match(/\bNewDerived\b/gu) ?? []).length !== 1) {
		violations.push(["U6_PROFILE_INTERNAL_REBUILD_SURFACE_NOT_EXACT", `${internalReferences}:${internalFiles.join(",")}`]);
	}
}

function inspectTypedPublication(manifest, violations) {
  const head = codeAt(manifest, "internal/store/head.go");
  requireExcludes(violations, head, "func (s *ObjectStore) AdvanceHead(", "U6_RAW_HEAD_API_EXPORTED", "internal/store/head.go");
  for (const anchor of [
    "func (s *ObjectStore) AdvanceBaseline(",
    "func (s *ObjectStore) AdvanceDivergence(",
    "func (s *ObjectStore) AdvanceReduction(",
    "func (s *ObjectStore) AdvanceConfirmation(",
    "func (s *ObjectStore) AdvanceChoicepoint(",
    "func (s *ObjectStore) AdvanceRuling(",
    "func (s *ObjectStore) advanceHead(",
    "authority confirmationauthority.Publication",
    "authority choicepromotionauthority.Choicepoint",
    "authority choicepromotionauthority.Ruling",
  ]) requireIncludes(violations, head, anchor, "U6_TYPED_HEAD_API_MISSING", anchor);

  for (const entry of productionEntries(manifest)) {
    if (entry.path !== "internal/store/head.go" && /\badvanceHead\b/u.test(entry.lexical.code)) {
      violations.push(["U6_RAW_HEAD_CALL_OUTSIDE_OWNER", entry.path]);
    }
    if (/\bAdvanceHead\s*\(/u.test(entry.lexical.code)) violations.push(["U6_RAW_HEAD_API_RESIDUE", entry.path]);
  }

  const typedTransitions = new Map([
    ["AdvanceBaseline", "StageBaseline"],
    ["AdvanceDivergence", "StageDivergence"],
    ["AdvanceReduction", "StageReduction"],
    ["AdvanceConfirmation", "StageConfirmation"],
    ["AdvanceChoicepoint", "StageChoicepointReady"],
    ["AdvanceRuling", "StageRuling"],
  ]);
  for (const [method, stage] of typedTransitions) {
    const body = functionBody(head, new RegExp(`\\bfunc\\s*\\(s\\s+\\*ObjectStore\\)\\s+${method}\\s*\\(`, "u")) ?? "";
    const exactCall = `return s.advanceHead(ctx, expected, ${stage}, object)`;
    if ((body.match(/\badvanceHead\b/gu) ?? []).length !== 1 || !body.includes(exactCall)) {
      violations.push(["U6_TYPED_TRANSITION_RAW_CALL_NOT_EXACT", method]);
    }
  }
  // One declaration plus the six typed transition calls above. This also
  // closes method-value exports and extra wrappers inside the raw owner file.
  if ((head.match(/\badvanceHead\b/gu) ?? []).length !== typedTransitions.size + 1) {
    violations.push(["U6_RAW_HEAD_REFERENCE_SURFACE_NOT_EXACT", "internal/store/head.go"]);
  }

  const advance = functionBody(head, /\bfunc\s*\(s\s+\*ObjectStore\)\s+advanceHead\s*\(/u) ?? "";
  const ordered = ["reopenHeadAndObject", "sameHeadToken(currentToken, expected)", "permittedTransition", "publishLocked", "newStoredHead", "replaceHead", "reopenHeadAndObject"];
  let cursor = -1;
  for (const anchor of ordered) {
    const next = advance.indexOf(anchor, cursor + 1);
    if (next < 0) violations.push(["U6_CAS_PUBLICATION_ORDER", anchor]);
    else cursor = next;
  }
  requireIncludes(violations, advance, "if !sameHeadToken(currentToken, expected)", "U6_FULL_CAS_COMPARE_REQUIRED", "internal/store/head.go");
  requireIncludes(violations, advance, "currentToken := s.token(expected.study, current)", "U6_CAS_CURRENT_TOKEN_DERIVATION", "internal/store/head.go");

  const tokenBody = functionBody(head, /\bfunc\s+sameHeadToken\s*\(/u) ?? "";
  for (const field of ["study.text", "revision", "stage", "currentKind", "currentDigest", "previousObject", "lineageRoot", "headDigest"]) {
    const escaped = field.replaceAll(".", "\\.");
    const equality = new RegExp(`\\bleft\\.${escaped}\\s*==\\s*right\\.${escaped}\\b`, "u");
    if (!equality.test(tokenBody)) violations.push(["U6_CAS_TOKEN_EXACT_EQUALITY_MISSING", field]);
  }
  if ((tokenBody.match(/==/gu) ?? []).length !== 8 || tokenBody.includes("||") || /\.Valid\s*\(/u.test(tokenBody)) {
    violations.push(["U6_CAS_TOKEN_COMPARISON_NOT_EXACT", "sameHeadToken"]);
  }
  const exactTokenBody = "return left.study.text == right.study.text && left.revision == right.revision && left.stage == right.stage && " +
    "left.currentKind == right.currentKind && left.currentDigest == right.currentDigest && left.previousObject == right.previousObject && " +
    "left.lineageRoot == right.lineageRoot && left.headDigest == right.headDigest";
  if (compactCode(tokenBody) !== exactTokenBody) {
    violations.push(["U6_CAS_TOKEN_BODY_NOT_EXACT", "sameHeadToken"]);
  }

  const transitionBody = functionBody(head, /\bfunc\s+permittedTransition\s*\(/u) ?? "";
  const transitionPairs = [
    ["StageSourcePlan", "StageBaseline"],
    ["StageBaseline", "StageDivergence"],
    ["StageDivergence", "StageReduction"],
    ["StageReduction", "StageConfirmation"],
    ["StageConfirmation", "StageChoicepointReady"],
    ["StageChoicepointReady", "StageRuling"],
    ["StageRuling", "StageResidue"],
  ];
  for (const [current, next] of transitionPairs) {
    const exact = new RegExp(`\\bcase\\s+${current}\\s*:\\s*return\\s+next\\s*==\\s*${next}\\b`, "u");
    if (!exact.test(transitionBody)) violations.push(["U6_TRANSITION_EDGE_MISSING", `${current}->${next}`]);
  }
  if ((transitionBody.match(/\bcase\b/gu) ?? []).length !== transitionPairs.length ||
      (transitionBody.match(/\breturn\s+next\s*==/gu) ?? []).length !== transitionPairs.length || transitionBody.includes("||")) {
    violations.push(["U6_TRANSITION_GRAPH_NOT_EXACT", "permittedTransition"]);
  }
  const exactTransitionBody = "switch current { case StageSourcePlan: return next == StageBaseline case StageBaseline: return next == StageDivergence " +
    "case StageDivergence: return next == StageReduction case StageReduction: return next == StageConfirmation " +
    "case StageConfirmation: return next == StageChoicepointReady case StageChoicepointReady: return next == StageRuling " +
    "case StageRuling: return next == StageResidue default: return false }";
  if (compactCode(transitionBody) !== exactTransitionBody) {
    violations.push(["U6_TRANSITION_BODY_NOT_EXACT", "permittedTransition"]);
  }

  const confirmationPublicationPath = "internal/confirmation/internal/publication/authority.go";
  const promotionPublicationPath = "internal/choice/promotion/internal/publication/authority.go";
  const confirmationPublication = codeAt(manifest, confirmationPublicationPath);
  const promotionPublication = codeAt(manifest, promotionPublicationPath);
  const confirmationPublicationTagged = commentlessAt(manifest, confirmationPublicationPath);
  const promotionPublicationTagged = commentlessAt(manifest, promotionPublicationPath);
  const confirmationValidator = functionRegion(confirmationPublication, /\bfunc\s+validateBody\s*\(/u);
  const promotionValidator = functionRegion(promotionPublication, /\bfunc\s+validateBody\s*\(/u);
  const confirmationValidatorCode = confirmationValidator?.body ?? "";
  const promotionValidatorCode = promotionValidator?.body ?? "";
  const confirmationValidatorTagged = confirmationValidator ? confirmationPublicationTagged.slice(confirmationValidator.start, confirmationValidator.end) : "";
  const promotionValidatorTagged = promotionValidator ? promotionPublicationTagged.slice(promotionValidator.start, promotionValidator.end) : "";
  for (const [source, path, types] of [
    [confirmationPublication, confirmationPublicationPath, ["Authority"]],
    [promotionPublication, promotionPublicationPath, ["Choicepoint", "Ruling"]],
  ]) {
    for (const type of types) {
      const bodies = topLevelStructBodies(source, type);
      if (bodies.length !== 1 || exportedStructFields(bodies[0]).length > 0) {
        violations.push(["U6_PUBLICATION_AUTHORITY_EXPORTED_FIELD", `${path}:${type}`]);
      }
      requireIncludes(violations, bodies[0] ?? "", "seal", "U6_PUBLICATION_AUTHORITY_SEAL_MISSING", `${path}:${type}`);
    }
  }

  const exportedDeclarations = (code, kind) => [...code.matchAll(new RegExp(
    `(?:^|\\n)\\s*${kind}\\s+${kind === "func" ? "(?:\\([^)]*\\)\\s*)?" : ""}([A-Z][A-Za-z0-9_]*)`, "gu",
  ))].map((match) => match[1]);
  const exactPublicationSurface = [
    [confirmationPublicationPath, ["Authority"], ["Issue", "Valid", "Digest", "Predecessor", "CanonicalBytes", "Equal"]],
    [promotionPublicationPath, ["Choicepoint", "Ruling"], ["IssueChoicepoint", "Valid", "Digest", "Predecessor", "CanonicalBytes", "IssueRuling", "Valid", "Digest", "Predecessor", "Action", "CanonicalBytes"]],
  ];
  for (const [path, types, functions] of exactPublicationSurface) {
    const code = codeAt(manifest, path);
    if (!exactSet(exportedDeclarations(code, "type"), types) || !exactSet(exportedDeclarations(code, "func"), functions) ||
        /(?:^|\n)\s*(?:var|const)\b/mu.test(code)) {
      violations.push(["U6_INTERNAL_PUBLICATION_SURFACE_NOT_EXACT", path]);
    }
  }

  for (const [source, anchor, code] of [
    [confirmationValidatorTagged, 'value.LookupMember("kind")', "U6_CONFIRMATION_PUBLICATION_KIND_BINDING"],
    [confirmationValidatorTagged, 'value.LookupMember("reduction_run_digest")', "U6_CONFIRMATION_PUBLICATION_PREDECESSOR_BINDING"],
    [promotionValidatorTagged, 'value.LookupMember("kind")', "U6_PROMOTION_PUBLICATION_KIND_BINDING"],
    [promotionValidatorTagged, "value.LookupMember(predecessorMember)", "U6_PROMOTION_PUBLICATION_PREDECESSOR_BINDING"],
    [promotionValidatorTagged, 'value.LookupMember("action")', "U6_RULING_PUBLICATION_ACTION_BINDING"],
  ]) requireIncludes(violations, source, anchor, code, anchor);
  for (const [source, anchor, code] of [
    [confirmationValidatorCode, "bytes.Equal(rebuilt, canonical)", "U6_CONFIRMATION_PUBLICATION_CANONICAL_BINDING"],
    [confirmationValidatorCode, "parsedPredecessor != predecessor", "U6_CONFIRMATION_PUBLICATION_PREDECESSOR_BINDING"],
    [confirmationValidatorCode, "computed.String() != digest.String()", "U6_CONFIRMATION_PUBLICATION_DIGEST_BINDING"],
    [promotionValidatorCode, "bytes.Equal(rebuilt, canonical)", "U6_PROMOTION_PUBLICATION_CANONICAL_BINDING"],
    [promotionValidatorCode, "parsedPredecessor != predecessor", "U6_PROMOTION_PUBLICATION_PREDECESSOR_BINDING"],
    [promotionValidatorCode, "computed.String() != digest.String()", "U6_PROMOTION_PUBLICATION_DIGEST_BINDING"],
    [promotionValidatorCode, "bodyAction != action", "U6_RULING_PUBLICATION_ACTION_BINDING"],
    [promotionValidatorCode, "durableRulingAction(action)", "U6_RULING_DURABLE_ACTION_CLOSURE"],
  ]) requireIncludes(violations, source, anchor, code, anchor);
  for (const [body, label] of [
    [confirmationValidatorCode, "FreshConfirmation"],
    [promotionValidatorCode, "Choicepoint/DecisionRecord"],
  ]) {
    if ((body.match(/\breturn\s+nil\b/gu) ?? []).length !== 1 || !compactCode(body).endsWith("return nil")) {
      violations.push(["U6_PUBLICATION_VALIDATOR_SUCCESS_PATH_NOT_EXACT", label]);
    }
    const order = ["canon.Parse(canonical)", "value.CanonicalChecked()", "value.LookupMember(", "canon.DigestBytes(", "return nil"];
    let cursor = -1;
    for (const anchor of order) {
      const next = body.indexOf(anchor, cursor + 1);
      if (next < 0) violations.push(["U6_PUBLICATION_VALIDATOR_ORDER_NOT_EXACT", `${label}:${anchor}`]);
      else cursor = next;
    }
  }

  const aliasSurfaces = [
    ["internal/confirmation/authority/authority.go", "internal/confirmation/internal/publication", new Map([
      ["Publication", "publication.Authority"],
    ])],
    ["internal/choice/promotion/authority/authority.go", "internal/choice/promotion/internal/publication", new Map([
      ["Choicepoint", "publication.Choicepoint"],
      ["Ruling", "publication.Ruling"],
    ])],
  ];
  for (const [path, importPath, aliases] of aliasSurfaces) {
    const entry = entryAt(manifest, path);
    const imports = importedPackages(entry.lexical.commentless);
    const code = entry.lexical.code;
    if (imports.length !== 1 || imports[0].path !== `${modulePrefix}${importPath}` || imports[0].alias !== "" ||
        !exactSet(exportedDeclarations(code, "type"), [...aliases.keys()]) || exportedDeclarations(code, "func").length !== 0 ||
        /(?:^|\n)\s*(?:var|const)\b/mu.test(code)) {
      violations.push(["U6_PUBLIC_AUTHORITY_ALIAS_SURFACE_NOT_EXACT", path]);
    }
    if (/\btype\s*\(/u.test(code) || (code.match(/\btype\b/gu) ?? []).length !== aliases.size) {
      violations.push(["U6_PUBLIC_AUTHORITY_ALIAS_DECLARATION_NOT_EXACT", path]);
    }
    for (const [name, target] of aliases) {
      const exactAlias = new RegExp(`\\btype\\s+${name}\\s*=\\s*${target.replaceAll(".", "\\.")}\\b`, "u");
      if (!exactAlias.test(code)) violations.push(["U6_PUBLIC_AUTHORITY_ALIAS_RHS_NOT_EXACT", `${path}:${name}`]);
    }
  }

  const issuerOwners = new Map([
    [`${modulePrefix}internal/confirmation/internal/publication`, new Set([
      "internal/confirmation/service.go", "internal/confirmation/authority/authority.go",
    ])],
    [`${modulePrefix}internal/choice/promotion/internal/publication`, new Set([
      "internal/choice/promotion/service.go", "internal/choice/promotion/authority/authority.go",
    ])],
  ]);
  for (const entry of productionEntries(manifest)) {
    const imports = importedPackages(entry.lexical.commentless);
    for (const imported of imports) {
      const owners = issuerOwners.get(imported.path);
      if (owners && !owners.has(entry.path)) violations.push(["U6_PUBLICATION_IMPORT_OUTSIDE_OWNER", `${entry.path}: ${imported.path}`]);
    }
    const packageDirectory = entry.path.slice(0, entry.path.lastIndexOf("/"));
    if (packageDirectory === "internal/confirmation" && entry.path !== "internal/confirmation/service.go" &&
        /\b(?:Draft|executionSeal)\b/u.test(entry.lexical.code)) {
      violations.push(["U6_CONFIRMATION_LIVE_AUTHORITY_CONSTRUCTION_OUTSIDE_OWNER", entry.path]);
    }
  }
  const confirmationService = codeAt(manifest, "internal/confirmation/service.go");
  if ((confirmationService.match(/\bconfirmationpublication\.Issue\s*\(/gu) ?? []).length !== 1 ||
      (confirmationService.match(/\bdraft\s*:=\s*Draft\s*\{/gu) ?? []).length !== 1 ||
      (confirmationService.match(/\bexecutionSeal\s*\{/gu) ?? []).length !== 1) {
    violations.push(["U6_CONFIRMATION_ISSUANCE_SURFACE_NOT_EXACT", "internal/confirmation/service.go"]);
  }
  const confirmationWire = codeAt(manifest, "internal/confirmation/wire.go");
  if (/\b(?:Draft|executionSeal|Publication)\b/u.test(confirmationWire)) {
    violations.push(["U6_CONFIRMATION_WIRE_RECREATES_LIVE_AUTHORITY", "internal/confirmation/wire.go"]);
  }
  const promotionService = codeAt(manifest, "internal/choice/promotion/service.go");
  if ((promotionService.match(/\bchoicepublication\.IssueChoicepoint\s*\(/gu) ?? []).length !== 1 ||
      (promotionService.match(/\bchoicepublication\.IssueRuling\s*\(/gu) ?? []).length !== 1) {
    violations.push(["U6_PROMOTION_ISSUANCE_SURFACE_NOT_EXACT", "internal/choice/promotion/service.go"]);
  }
  for (const [type, seal] of [
    ["StoredConfirmation", "storedConfirmationSeal"],
    ["Ready", "readySeal"],
    ["Ruling", "rulingSeal"],
  ]) {
    const bodies = topLevelStructBodies(promotionService, type);
    if (bodies.length !== 1 || exportedStructFields(bodies[0]).length !== 0 || !bodies[0].includes("seal")) {
      violations.push(["U6_PROMOTION_CAPABILITY_SURFACE_NOT_CLOSED", type]);
    }
    if ((promotionService.match(new RegExp(`\\b${seal}\\s*\\{`, "gu")) ?? []).length !== 1) {
      violations.push(["U6_PROMOTION_LIVE_SEAL_ISSUANCE_NOT_EXACT", seal]);
    }
  }
  for (const entry of productionEntries(manifest)) {
    const packageDirectory = entry.path.slice(0, entry.path.lastIndexOf("/"));
    if (packageDirectory === "internal/choice/promotion" && entry.path !== "internal/choice/promotion/service.go" &&
        /\b(?:StoredConfirmation|Ready|Ruling|storedConfirmationSeal|readySeal|rulingSeal)\b/u.test(entry.lexical.code)) {
      violations.push(["U6_PROMOTION_LIVE_CAPABILITY_CONSTRUCTION_OUTSIDE_OWNER", entry.path]);
    }
  }

  const objectStore = codeAt(manifest, "internal/store/object_store.go");
  const publish = functionBody(objectStore, /\bfunc\s*\(s\s+\*ObjectStore\)\s+publishLocked\s*\(/u) ?? "";
  const publishOrder = ["os.Lstat(destination)", "writeSyncedTemporary", "reopenExactObject(temporary, object)", "os.Link(temporary, destination)", "reopenExactObject(destination, object)"];
  let publishCursor = -1;
  for (const anchor of publishOrder) {
    const next = publish.indexOf(anchor, publishCursor + 1);
    if (next < 0) violations.push(["U6_IMMUTABLE_PUBLICATION_ORDER", anchor]);
    else publishCursor = next;
  }
  for (const pattern of [
    /os\.Remove\s*\(\s*destination\s*\)/u,
    /os\.Rename\s*\([^,]+,\s*destination\s*\)/u,
    /os\.OpenFile\s*\(\s*destination\b/u,
    /os\.WriteFile\s*\(\s*destination\b/u,
    /os\.Create\s*\(\s*destination\b/u,
  ]) {
    if (pattern.test(publish)) violations.push(["U6_IMMUTABLE_DESTINATION_MUTATION", pattern.source]);
  }
}

function inspectPromotionAndFreshness(manifest, violations) {
  const service = codeAt(manifest, "internal/choice/promotion/service.go");
  const confirmationEntry = entryAt(manifest, "internal/confirmation/service.go");
  const confirmation = confirmationEntry.lexical.code;
  const wire = codeAt(manifest, "internal/confirmation/wire.go");
  const worldFact = codeAt(manifest, "internal/world/fresh_confirmation.go");

  for (const anchor of [
    "head.LineageRootDigest() != record.PlanDigest()",
    "previous != record.OriginalBaselineMap().ArtifactDigest().DomainDigest()",
    "objectStore.AdvanceConfirmation(ctx, expectedReduction, draft.PublicationAuthority())",
    "objectStore.AdvanceChoicepoint(ctx, stored.head, publicationAuthority)",
    "objectStore.AdvanceRuling(ctx, ready.head, publicationAuthority)",
  ]) requireIncludes(violations, service, anchor, "U6_PROMOTION_BINDING_MISSING", anchor);

  const finalize = functionBody(service, /\bfunc\s+Finalize\s*\(/u) ?? "";
  const refine = finalize.indexOf("if decision.Action() == choice.ActionRefine");
  const issue = finalize.indexOf("choicepublication.IssueRuling(", refine);
  const advance = finalize.indexOf("objectStore.AdvanceRuling(", refine);
  if (refine < 0 || issue < 0 || advance < 0 || !(refine < issue && issue < advance)) {
    violations.push(["U6_REFINE_GUARD_ORDER", "REFINE must refuse before authority issuance and store advance"]);
  }
  requireIncludes(violations, service, "if record.Action() == choice.ActionRefine", "U6_OPEN_RULING_REFINE_GUARD", "internal/choice/promotion/service.go");

  if (!importedPackages(confirmationEntry.lexical.commentless).some((entry) => entry.path === "crypto/rand")) {
    violations.push(["U6_PHYSICAL_CONFIRMATION_ANCHOR", "crypto/rand"]);
  }
  for (const anchor of [
    "confirmationNonce(challenge",
    "observe.RunObservation",
    "FreshConfirmationEvidence",
    "BindExecuted",
    "compare.AssessPreservation",
    "priorEvidenceLedger",
    "confirmationpublication.Issue",
  ]) requireIncludes(violations, confirmation, anchor, "U6_PHYSICAL_CONFIRMATION_ANCHOR", anchor);
  requireIncludes(violations, wire, "candidate != scheduledTrials[index].CandidateKey()", "U6_CONFIRMATION_ORDINAL_CANDIDATE_BINDING", "internal/confirmation/wire.go");
  requireIncludes(violations, wire, "world.InspectFreshExecutionFactWire", "U6_PHYSICAL_FACT_STRICT_PARSE_REQUIRED", "internal/confirmation/wire.go");
  for (const anchor of ["func InspectFreshExecutionFactWire(", "len(members) != 25", "fact.Valid()"] ) {
    requireIncludes(violations, worldFact, anchor, "U6_PHYSICAL_FACT_CLOSED_PARSE_REQUIRED", anchor);
  }

  const nonce = functionBody(confirmation, /\bfunc\s+confirmationNonce\s*\(/u) ?? "";
  for (const anchor of ["ChallengeDigest", "Ordinal", "challenge.String()", "ordinal"]) {
    requireIncludes(violations, nonce, anchor, "U6_CONFIRMATION_NONCE_BINDING", anchor);
  }
  const schedule = codeAt(manifest, "internal/observe/schedule.go");
  const scheduleBody = functionBody(schedule, /\bfunc\s+NewPhaseRotatedSchedule\s*\(/u) ?? "";
  requireIncludes(violations, scheduleBody, "startOffset = 1 % len(canonicalRoster)", "U6_CONFIRMATION_OFFSET_NOT_EXACT", "internal/observe/schedule.go");
}

function inspectBlindAndDecision(manifest, violations) {
  const blind = codeAt(manifest, "internal/choice/blind.go");
  const blindTagged = commentlessAt(manifest, "internal/choice/blind.go");
  const validation = codeAt(manifest, "internal/choice/validation.go");
  const session = codeAt(manifest, "internal/choice/session.go");
  const expected = new Map([
    ["blindDTOIdentity", ["schema_version", "kind", "choicepoint_digest", "scenario", "scope", "original_stimulus_base64", "minimized_stimulus_base64", "reduction", "discovery_repeats_per_candidate", "confirmation_repeats_per_candidate", "projection_mode", "projection_operations", "selectable_fields", "differing_fields", "cards", "trust_warnings"]],
    ["BlindCard", ["alias", "fields"]],
    ["BlindField", ["field_id", "tag", "text", "boolean", "canonical_json_base64"]],
    ["BlindProjectionOperation", ["name", "rule_digest"]],
    ["BlindReductionFacts", ["grade_status", "proposal_limit", "candidate_trial_limit", "wall_limit_ms", "evaluation_count", "unresolved_count", "limitations", "reducer_set_digest", "final_sweep_state"]],
  ]);
  for (const [name, tags] of expected) {
    const actual = structJSONTags(blindTagged, name);
    if (!actual || !exactSet(actual, tags)) violations.push(["U6_BLIND_DTO_SHAPE", `${name}: ${actual?.join(",") ?? "absent"}`]);
  }
  // Ruling inputs intentionally have no JSON tags; close their caller-authored
  // field surface from the Go declarations instead.
  for (const [source, name, fields] of [
    [validation, "RulingInput", ["Action", "SelectedFields", "AllowedObserved", "CustomExpectation", "CustomReview"]],
    [session, "RulingDraftInput", ["Action", "SelectedFields", "AllowedAliases", "CustomExpectation", "CustomReviewer", "CustomReviewEvidence"]],
  ]) {
    const start = source.indexOf(`type ${name} struct {`);
    const end = source.indexOf("\n}", start);
    const body = start >= 0 && end > start ? source.slice(start, end) : "";
    const actual = [...body.matchAll(/^\s*([A-Z][A-Za-z0-9_]*)\s+/gmu)].map((match) => match[1]);
    if (!exactSet(actual, fields)) violations.push(["U6_RULING_INPUT_SHAPE", `${name}: ${actual.join(",")}`]);
  }
  requireIncludes(violations, blind, "blindAliasForProjectionIdentity(", "U6_BLIND_ALIAS_ISSUER_MISSING", "internal/choice/blind.go");
  const groupsBody = functionBody(blind, /\bfunc\s+buildBlindGroups\s*\(/u) ?? "";
  if (!/blindAliasForProjectionIdentity\s*\(\s*choicepoint\s*,\s*outcome\.ref\.fingerprint\.String\s*\(\s*\)\s*,?\s*\)/u.test(groupsBody)) {
    violations.push(["U6_BLIND_ALIAS_NOT_FINGERPRINT_BOUND", "internal/choice/blind.go"]);
  }
  requireIncludes(violations, blind, "Alias: current.alias", "U6_BLIND_ALIAS_FIRST_MEMBER", "internal/choice/blind.go");
  requireIncludes(violations, validation, "containsControl(reviewer)", "U6_REVIEWER_CONTROL_TEXT_ALLOWED", "internal/choice/validation.go");
  for (const [path, literal, code] of [
    ["internal/choice/decision.go", "AUTHENTICITY_NOT_ESTABLISHED_IN_U6", "U6_ACTOR_AUTHENTICITY_OVERCLAIM"],
    ["internal/choice/decision.go", "PRESENTED_NOT_COMPREHENDED_OR_DEBIASED", "U6_PRESENTATION_NONCLAIM_MISSING"],
  ]) {
    if (!literalsAt(manifest, path).includes(literal)) violations.push([code, path]);
  }

  const sortAnchor = groupsBody.indexOf("sort.Slice(groups");
  if ((groupsBody.match(/sort\.Slice\s*\(\s*groups\b/gu) ?? []).length !== 1) {
    violations.push(["U6_BLIND_SORT_SURFACE_NOT_EXACT", "buildBlindGroups"]);
  }
  const comparatorOpen = groupsBody.indexOf("{", groupsBody.indexOf("func(i, j int) bool", sortAnchor));
  const comparator = balancedBlock(groupsBody, comparatorOpen)?.body ?? "";
  for (const anchor of [
    "groups[i].orderKey != groups[j].orderKey",
    "return groups[i].orderKey < groups[j].orderKey",
    "return groups[i].fingerprint.String() < groups[j].fingerprint.String()",
  ]) requireIncludes(violations, comparator, anchor, "U6_BLIND_COMPARATOR_NOT_EXACT", anchor);
  if (/\b(?:refs|alias|candidate)\b/u.test(comparator) || /\blen\s*\(/u.test(comparator)) {
    violations.push(["U6_BLIND_COMPARATOR_READS_HIDDEN_AUTHORITY", "internal/choice/blind.go"]);
  }
  const exactComparator = "if groups[i].orderKey != groups[j].orderKey { return groups[i].orderKey < groups[j].orderKey } " +
    "return groups[i].fingerprint.String() < groups[j].fingerprint.String()";
  if (compactCode(comparator) !== exactComparator) {
    violations.push(["U6_BLIND_COMPARATOR_BODY_NOT_EXACT", "internal/choice/blind.go"]);
  }

  for (const marker of ["isCompileEligibility()", "isCompilableRuling()", "isNoncompilableRuling()"]) {
    requireIncludes(violations, validation, marker, "U6_COMPILE_ELIGIBILITY_MARKER_MISSING", marker);
  }
  for (const [identifier, count] of [
    ["isCompileEligibility", 3],
    ["isCompilableRuling", 2],
    ["isNoncompilableRuling", 2],
  ]) {
    const pattern = new RegExp(`\\b${identifier}\\b`, "gu");
    if ((validation.match(pattern) ?? []).length !== count) {
      violations.push(["U6_COMPILE_ELIGIBILITY_MARKER_SURFACE_NOT_EXACT", identifier]);
    }
  }
  for (const entry of productionEntries(manifest)) {
    const packageDirectory = entry.path.slice(0, entry.path.lastIndexOf("/"));
    if (packageDirectory === "internal/choice" && entry.path !== "internal/choice/validation.go" &&
        /\b(?:compileEligibility|compilableRuling|noncompilableRuling|isCompileEligibility|isCompilableRuling|isNoncompilableRuling)\b/u.test(entry.lexical.code)) {
      violations.push(["U6_COMPILE_ELIGIBILITY_AUTHORITY_OUTSIDE_OWNER", entry.path]);
    }
    if (packageDirectory === "internal/choice" && entry.path !== "internal/choice/validation.go" && embedsAuthorityInterface(entry.lexical.code)) {
      violations.push(["U6_COMPILE_ELIGIBILITY_EMBEDDING_OUTSIDE_OWNER", entry.path]);
    }
  }
  if ((validation.match(/\bcompilableRuling\s*\{/gu) ?? []).length !== 1 ||
      (validation.match(/\bnoncompilableRuling\s*\{/gu) ?? []).length !== 1) {
    violations.push(["U6_COMPILE_ELIGIBILITY_CONSTRUCTION_NOT_CLOSED", "internal/choice/validation.go"]);
  }
  const validateRuling = functionBody(validation, /\bfunc\s+ValidateRuling\s*\(/u) ?? "";
  const validateCompilableBody = functionBody(validation, /\bfunc\s+validateCompilable\s*\(/u) ?? "";
  requireIncludes(violations, validateRuling, "case ActionRejectAll, ActionDefer, ActionRefine:", "U6_NONCOMPILABLE_ACTION_SET", "ValidateRuling");
  requireIncludes(violations, validateRuling, "noncompilableRuling{action: input.Action}", "U6_NONCOMPILABLE_CONSTRUCTOR_OWNER", "ValidateRuling");
  requireIncludes(violations, validateRuling, "case ActionAllowObserved, ActionCustomExpectation:", "U6_COMPILABLE_ACTION_SET", "ValidateRuling");
  requireIncludes(violations, validateRuling, "return validateCompilable(confirmed, input)", "U6_COMPILABLE_CONSTRUCTOR_ROUTE", "ValidateRuling");
  requireIncludes(violations, validateCompilableBody, "compilableRuling{", "U6_COMPILABLE_CONSTRUCTOR_OWNER", "validateCompilable");
}

function strictJSONParse(text, label) {
  const value = JSON.parse(text);
  const duplicates = [];
  let cursor = 0;
  const whitespace = () => { while (/\s/u.test(text[cursor] ?? "")) cursor += 1; };
  const string = () => {
    const start = cursor++;
    while (cursor < text.length) {
      if (text[cursor] === "\\") cursor += 2;
      else if (text[cursor] === '"') return JSON.parse(text.slice(start, ++cursor));
      else cursor += 1;
    }
    throw new SyntaxError(`unterminated string in ${label}`);
  };
  const primitive = () => {
    const match = /^(?:-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?|true|false|null)/u.exec(text.slice(cursor));
    if (!match) throw new SyntaxError(`invalid JSON token in ${label} at ${cursor}`);
    cursor += match[0].length;
  };
  const scan = (pointer) => {
    whitespace();
    if (text[cursor] === "{") {
      cursor += 1; whitespace();
      const keys = new Set();
      if (text[cursor] === "}") { cursor += 1; return; }
      while (cursor < text.length) {
        whitespace();
        if (text[cursor] !== '"') throw new SyntaxError(`object key expected in ${label} at ${cursor}`);
        const key = string();
        const child = `${pointer}/${String(key).replaceAll("~", "~0").replaceAll("/", "~1")}`;
        if (keys.has(key)) duplicates.push(child);
        keys.add(key); whitespace();
        if (text[cursor++] !== ":") throw new SyntaxError(`colon expected in ${label} at ${cursor - 1}`);
        scan(child); whitespace();
        if (text[cursor] === "}") { cursor += 1; return; }
        if (text[cursor++] !== ",") throw new SyntaxError(`comma expected in ${label} at ${cursor - 1}`);
      }
      throw new SyntaxError(`unterminated object in ${label}`);
    }
    if (text[cursor] === "[") {
      cursor += 1; whitespace();
      if (text[cursor] === "]") { cursor += 1; return; }
      let index = 0;
      while (cursor < text.length) {
        scan(`${pointer}/${index++}`); whitespace();
        if (text[cursor] === "]") { cursor += 1; return; }
        if (text[cursor++] !== ",") throw new SyntaxError(`comma expected in ${label} at ${cursor - 1}`);
      }
      throw new SyntaxError(`unterminated array in ${label}`);
    }
    if (text[cursor] === '"') string(); else primitive();
  };
  scan(""); whitespace();
  if (cursor !== text.length) throw new SyntaxError(`trailing JSON content in ${label} at ${cursor}`);
  return { value, duplicates };
}

function canonicalJSONString(value) {
  if (value === null || typeof value !== "object") return JSON.stringify(value);
  if (Array.isArray(value)) return `[${value.map(canonicalJSONString).join(",")}]`;
  return `{${Object.keys(value).sort().map((key) => `${JSON.stringify(key)}:${canonicalJSONString(value[key])}`).join(",")}}`;
}

function inspectSchemas(manifest, violations) {
  const choiceSource = commentlessAt(manifest, "internal/choice/choicepoint.go");
  const decisionSource = commentlessAt(manifest, "internal/choice/decision.go");
  for (const [name, sourcePath, schemaPath, examplePath, count] of [
    ["choicepointIdentity", choiceSource, "spec/schema/v1/choicepoint.schema.json", "spec/examples/v1/choicepoint.valid.json", 27],
    ["decisionIdentity", decisionSource, "spec/schema/v1/decision-record.schema.json", "spec/examples/v1/decision-record.valid.json", 40],
  ]) {
    const runtime = structJSONTags(sourcePath, name);
    let schema;
    let example;
    try {
      const parsedSchema = strictJSONParse(sourceAt(manifest, schemaPath), schemaPath);
      const parsedExample = strictJSONParse(sourceAt(manifest, examplePath), examplePath);
      if (parsedSchema.duplicates.length > 0 || parsedExample.duplicates.length > 0) {
        violations.push(["U6_DUPLICATE_JSON_MEMBER", `${name}: ${[...parsedSchema.duplicates, ...parsedExample.duplicates].join(",")}`]);
      }
      schema = parsedSchema.value;
      example = parsedExample.value;
    } catch (error) {
      violations.push(["U6_SCHEMA_JSON_INVALID", `${name}: ${error.message}`]);
      continue;
    }
    const required = schema.required ?? [];
    const properties = Object.keys(schema.properties ?? {});
    const exampleKeys = Object.keys(example);
    if (!runtime || runtime.length !== count || !exactSet(runtime, required) || !exactSet(runtime, properties) || !exactSet(runtime, exampleKeys)) {
      violations.push(["U6_SCHEMA_RUNTIME_PARITY", `${name}: runtime=${runtime?.length ?? 0} required=${required.length} properties=${properties.length} example=${exampleKeys.length}`]);
    }
    if (schema.additionalProperties !== false) violations.push(["U6_SCHEMA_OPEN_OBJECT", schemaPath]);
    if (runtime?.includes("artifact_digest")) violations.push(["U6_SELF_DIGEST_IN_CANONICAL_BODY", name]);
    const exampleRaw = sourceAt(manifest, examplePath);
    if (!exampleRaw.endsWith("\n") || exampleRaw.slice(0, -1).includes("\n")) violations.push(["U6_EXAMPLE_NOT_ONE_CANONICAL_LINE", examplePath]);
    if (exampleRaw.slice(0, -1) !== canonicalJSONString(example)) violations.push(["U6_EXAMPLE_NOT_CANONICAL_JSON", examplePath]);
  }
  const confirmationTags = structJSONTags(commentlessAt(manifest, "internal/confirmation/service.go"), "freshConfirmationIdentity");
  const expectedConfirmationTags = [
    "schema_version", "kind", "world_plan_digest", "original_baseline_map_base64", "reduced_baseline_map_base64",
    "reduced_baseline_artifact_digest", "reduced_baseline_preservation_digest", "reduction_run_base64",
    "reduction_transcript_base64", "reduction_run_digest", "reduction_transcript_digest", "reduction_grade_base64",
    "reduction_grade_digest", "reduction_grade_status", "confirmed_map_base64", "confirmed_artifact_digest",
    "confirmed_preservation_digest", "projection_proofs", "preservation_relation", "preservation_reason", "challenge_digest",
    "confirmation_phase", "confirmation_schedule_digest", "confirmation_schedule_offset", "confirmation_trial_count",
    "physical_fact_bytes_base64", "physical_fact_digests", "process_digests", "root_layout_digests",
    "invocation_receipt_digests", "invocation_file_digests", "batch_digests", "attempt_digests", "world_digests",
    "observation_digests", "prior_evidence_ledger_digest", "scope", "invocation_trust_nonclaim",
  ];
  if (!confirmationTags || !exactSet(confirmationTags, expectedConfirmationTags)) {
    violations.push(["U6_FRESH_CONFIRMATION_WIRE_SHAPE", confirmationTags?.join(",") ?? "absent"]);
  }
  const commonParsed = strictJSONParse(sourceAt(manifest, "spec/schema/v1/common.schema.json"), "common.schema.json");
  if (commonParsed.duplicates.length > 0) violations.push(["U6_DUPLICATE_JSON_MEMBER", `common: ${commonParsed.duplicates.join(",")}`]);
  const grade = commonParsed.value?.$defs?.ReceiptReference?.properties?.grade_verbatim;
  if (JSON.stringify(grade) !== JSON.stringify({ type: "string", minLength: 1 })) {
    violations.push(["U6_RECEIPT_GRADE_NOT_VERBATIM", "spec/schema/v1/common.schema.json"]);
  }
}

function inspectNonclaims(manifest, violations) {
  const production = productionEntries(manifest);
  const literals = new Set(production.flatMap((entry) => entry.lexical.literals));
  for (const anchor of [
    "EXACT_FRESH_CONFIRMED_WITNESS_V1",
    "CANDIDATE_NEUTRAL_EXACT_WITNESS_CHOICE_V1",
    "EXACT_WITNESSED_STIMULUS_V1",
    "PRESENTED_NOT_COMPREHENDED_OR_DEBIASED",
    "AUTHENTICITY_NOT_ESTABLISHED_IN_U6",
    "OPAQUE_VERBATIM_REFERENCE_NOT_CONFORMANCE",
    "UNRECEIPTED",
    "CANDIDATE_WRITTEN_FIXTURE_EVIDENCE_IS_NOT_HOSTILE_PROCESS_ATTESTATION",
    "FIXTURE_OBSERVED_LOCAL_PROTOCOL_NOT_HOSTILE_PROCESS_ATTESTATION",
    "RELATIVE_FRESHNESS_REQUIRES_CROSS_LINEAGE_DISJOINTNESS",
  ]) {
    if (!literals.has(anchor)) violations.push(["U6_NONCLAIM_ANCHOR_MISSING", anchor]);
  }
  const receipt = codeAt(manifest, "internal/domain/receipt.go");
  requireIncludes(
    violations,
    receipt,
    "NewDidrunReceipt(wire.GradeVerbatim, wire.CommitOID, digest)",
    "U6_RECEIPT_GRADE_NORMALIZED",
    "internal/domain/receipt.go",
  );
  for (const entry of production) {
    for (const forbidden of ["WakeEvent", "WakeEffect", "WakeEpoch", "WakeGate", "ReplayEngine", "CommandRecorder", "CrashResume"]) {
      if (new RegExp(`\\b${forbidden}\\b`, "u").test(entry.lexical.code)) {
        violations.push(["U6_WAKE_SURFACE_FORBIDDEN", `${entry.path}: ${forbidden}`]);
      }
    }
    if (/\bDisplayGroups\s*\(/u.test(entry.lexical.code) &&
        (entry.path.startsWith("internal/confirmation/") || entry.path.startsWith("internal/choice/") || entry.path.startsWith("internal/store/"))) {
      violations.push(["U6_DISPLAY_GROUPS_AS_AUTHORITY", entry.path]);
    }
    if (/\bGeneralizationClaimed\s*:\s*true\b/u.test(entry.lexical.code)) {
      violations.push(["U6_GENERALIZATION_OVERCLAIM", entry.path]);
    }
    if (importedPackages(entry.lexical.commentless).some((candidate) => /(?:^|\/)wake(?:\/|$)/iu.test(candidate.path))) {
      violations.push(["U6_WAKE_IMPORT_FORBIDDEN", entry.path]);
    }
  }
}

function inspectManifest(manifest) {
  const violations = [];
  inspectPackageBoundaries(manifest, violations);
	inspectPortableAuthorityBoundaries(manifest, violations);
  inspectTypedPublication(manifest, violations);
  inspectPromotionAndFreshness(manifest, violations);
  inspectBlindAndDecision(manifest, violations);
  inspectSchemas(manifest, violations);
  inspectNonclaims(manifest, violations);
  return violations;
}

function assertSameManifest(before, after) {
  if (before.digest !== after.digest || before.entries.length !== after.entries.length) {
    throw new ArchitectureError("U6_MANIFEST_CHANGED_DURING_SCAN", `${before.entries.length}/${before.digest} -> ${after.entries.length}/${after.digest}`);
  }
  for (let index = 0; index < before.entries.length; index += 1) {
    const left = before.entries[index];
    const right = after.entries[index];
    if (left.path !== right.path || left.bytes.length !== right.bytes.length || left.sha256 !== right.sha256) {
      throw new ArchitectureError("U6_MANIFEST_CHANGED_DURING_SCAN", `${left.path} -> ${right.path}`);
    }
  }
}

async function main() {
  if (process.argv.length !== 2) throw new ArchitectureError("U6_ARCHITECTURE_ARGUMENTS", "no arguments or root overrides are accepted");
  const initial = await readManifest();
  verifyIndependentManifest(initial);
  runInheritedU5Checker();
  // U6_SELFTEST_MANIFEST_BARRIER
  const violations = inspectManifest(initial);
  const final = await readManifest();
  assertSameManifest(initial, final);
  if (violations.length > 0) {
    const detail = violations.map(([code, message]) => `${code}: ${message}`).sort((a, b) => a.localeCompare(b, "en")).join("\n");
    throw new ArchitectureError("U6_ARCHITECTURE_VIOLATION", detail);
  }
  const goCount = initial.entries.filter((entry) => entry.kind === "go").length;
  process.stdout.write(`U6 architecture boundary OK (${goCount} exact Go files; ${jsonAuthorities.length} exact JSON authorities; manifest ${initial.digest})\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.stack ?? error}\n`);
  process.exitCode = 1;
});
