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
const c3StoreModelImport = "internal/contractexec/model";
const c4StoreOwnerPath = "internal/store/contract_run_bridge.go";
const c4StoreHostEpochImport = "internal/hostepoch";
const c4ProcessMechanicsImport = "internal/processmechanics";
const c4WorldMechanicsPaths = Object.freeze([
	"internal/world/process.go",
	"internal/world/process_darwin.go",
]);

// Every member below these roots participates. Subdirectories are an exact
// package map rather than permissive prefixes: a newly invented authority path
// closes the gate until its jurisdiction is reviewed here.
const reviewedRoots = Object.freeze([
  ["internal/domain", { ".": "domain" }],
  ["internal/compare", { ".": "compare" }],
  ["internal/observe", {
    ".": "observe",
    eligibilitycore: "eligibilitycore",
  }],
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

function decodeGoImportLiteral(raw) {
  if (raw.startsWith("`") && raw.endsWith("`")) return raw.slice(1, -1).replaceAll("\r", "");
  if (!raw.startsWith('"') || !raw.endsWith('"')) {
    throw new ArchitectureError("U6_GO_IMPORT_LITERAL_INVALID", "import path is not a Go string literal");
  }
  let decoded = "";
  const body = raw.slice(1, -1);
  for (let index = 0; index < body.length; index += 1) {
    const character = body[index];
    if (character !== "\\") {
      if (character === "\n" || character === "\r") {
        throw new ArchitectureError("U6_GO_IMPORT_LITERAL_INVALID", "newline in interpreted import path");
      }
      decoded += character;
      continue;
    }
    const escape = body[++index];
    if (escape === undefined) throw new ArchitectureError("U6_GO_IMPORT_LITERAL_INVALID", "trailing escape");
    const simple = new Map([
      ["a", "\x07"], ["b", "\b"], ["f", "\f"], ["n", "\n"], ["r", "\r"],
      ["t", "\t"], ["v", "\x0b"], ["\\", "\\"], ['"', '"'], ["'", "'"],
    ]);
    if (simple.has(escape)) {
      decoded += simple.get(escape);
      continue;
    }
    let digits;
    let radix;
    if (/[0-7]/u.test(escape)) {
      digits = escape + body.slice(index + 1, index + 3);
      radix = 8;
      index += 2;
      if (!/^[0-7]{3}$/u.test(digits)) digits = null;
    } else if (escape === "x") {
      digits = body.slice(index + 1, index + 3);
      radix = 16;
      index += 2;
      if (!/^[0-9A-Fa-f]{2}$/u.test(digits)) digits = null;
    } else if (escape === "u" || escape === "U") {
      const width = escape === "u" ? 4 : 8;
      digits = body.slice(index + 1, index + 1 + width);
      radix = 16;
      index += width;
      if (!new RegExp(`^[0-9A-Fa-f]{${width}}$`, "u").test(digits)) digits = null;
    }
    if (digits === null || digits === undefined) {
      throw new ArchitectureError("U6_GO_IMPORT_LITERAL_INVALID", `unsupported escape \\${escape}`);
    }
    const value = Number.parseInt(digits, radix);
    if (value > 0x10ffff || (value >= 0xd800 && value <= 0xdfff)) {
      throw new ArchitectureError("U6_GO_IMPORT_LITERAL_INVALID", "invalid Unicode scalar");
    }
    decoded += String.fromCodePoint(value);
  }
  return decoded;
}

function goSourceTokens(commentless) {
  const tokens = [];
  for (let index = 0; index < commentless.length;) {
    const character = String.fromCodePoint(commentless.codePointAt(index));
    if (/\s/u.test(character)) {
      index += character.length;
      continue;
    }
    if (character === '"' || character === "'" || character === "`") {
      const start = index;
      const quote = character;
      index += quote.length;
      let escaped = false;
      while (index < commentless.length) {
        const current = String.fromCodePoint(commentless.codePointAt(index));
        index += current.length;
        if (quote === "`") {
          if (current === "`") break;
          continue;
        }
        if (!escaped && current === quote) break;
        if (!escaped && current === "\\") escaped = true;
        else escaped = false;
      }
      const raw = commentless.slice(start, index);
      if (!raw.endsWith(quote)) throw new ArchitectureError("U6_GO_LEXICAL_INVALID", "unterminated literal token");
      tokens.push({ kind: quote === "'" ? "rune" : "string", raw, value: quote === "'" ? raw : decodeGoImportLiteral(raw) });
      continue;
    }
    if (/[_\p{ID_Start}]/u.test(character)) {
      const start = index;
      index += character.length;
      while (index < commentless.length) {
        const current = String.fromCodePoint(commentless.codePointAt(index));
        if (!/[_\p{ID_Continue}]/u.test(current)) break;
        index += current.length;
      }
      tokens.push({ kind: "identifier", value: commentless.slice(start, index) });
      continue;
    }
    tokens.push({ kind: "punctuation", value: character });
    index += character.length;
  }
  return tokens;
}

function importedPackages(commentless) {
  const tokens = goSourceTokens(commentless);
  const imports = [];
  let braceDepth = 0;
  const consumeSpecs = (start, end) => {
    let cursor = start;
    while (cursor < end) {
      while (cursor < end && tokens[cursor].kind === "punctuation" && tokens[cursor].value === ";") cursor += 1;
      if (cursor >= end) break;
      let alias = "";
      const candidate = tokens[cursor];
      if (candidate.kind === "identifier" || (candidate.kind === "punctuation" && candidate.value === ".")) {
        alias = candidate.value;
        cursor += 1;
      }
      if (cursor >= end || tokens[cursor].kind !== "string") {
        throw new ArchitectureError("U6_GO_IMPORT_DECLARATION_INVALID", "import spec is not completely consumed");
      }
      imports.push({ alias, path: tokens[cursor].value });
      cursor += 1;
    }
  };
  for (let index = 0; index < tokens.length; index += 1) {
    const token = tokens[index];
    if (token.kind === "punctuation" && token.value === "{") braceDepth += 1;
    else if (token.kind === "punctuation" && token.value === "}") braceDepth -= 1;
    if (braceDepth !== 0 || token.kind !== "identifier" || token.value !== "import") continue;
    const next = tokens[index + 1];
    if (next?.kind === "punctuation" && next.value === "(") {
      let close = index + 2;
      while (close < tokens.length && !(tokens[close].kind === "punctuation" && tokens[close].value === ")")) close += 1;
      if (close >= tokens.length) throw new ArchitectureError("U6_GO_IMPORT_DECLARATION_INVALID", "unterminated import group");
      consumeSpecs(index + 2, close);
      index = close;
      continue;
    }
    let end = index + 1;
    if (tokens[end]?.kind === "identifier" || (tokens[end]?.kind === "punctuation" && tokens[end].value === ".")) end += 1;
    if (tokens[end]?.kind !== "string") {
      throw new ArchitectureError("U6_GO_IMPORT_DECLARATION_INVALID", "single import has no path literal");
    }
    consumeSpecs(index + 1, end + 1);
    index = end;
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

function isTopLevelOffset(code, offset) {
  let depth = 0;
  for (let index = 0; index < offset; index += 1) {
    if (code[index] === "{") depth += 1;
    else if (code[index] === "}") depth -= 1;
    if (depth < 0) return false;
  }
  return depth === 0;
}

function topLevelMatches(code, expression) {
  return [...code.matchAll(expression)].filter((match) => isTopLevelOffset(code, match.index));
}

function topLevelStructBodies(code, typeName) {
  const escaped = typeName.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
  const pattern = new RegExp(`\\btype\\s+${escaped}\\s+struct\\s*\\{`, "gu");
  const bodies = [];
  for (const match of topLevelMatches(code, pattern)) {
    const open = match.index + match[0].lastIndexOf("{");
    const block = balancedBlock(code, open);
    if (block) bodies.push(block.body);
  }
  return bodies;
}

function exportedStructFields(body) {
	const declarations = [];
	let start = 0;
	let roundDepth = 0;
	let squareDepth = 0;
	let braceDepth = 0;
	for (let index = 0; index <= body.length; index += 1) {
		const character = body[index] ?? "\n";
		if (character === "(") roundDepth += 1;
		else if (character === ")") roundDepth -= 1;
		else if (character === "[") squareDepth += 1;
		else if (character === "]") squareDepth -= 1;
		else if (character === "{") braceDepth += 1;
		else if (character === "}") braceDepth -= 1;
		if ((character === "\n" || character === ";") && roundDepth === 0 && squareDepth === 0 && braceDepth === 0) {
			const declaration = body.slice(start, index).trim();
			if (declaration !== "") declarations.push(declaration);
			start = index + 1;
		}
	}
	const identifier = String.raw`[_\p{ID_Start}][_\p{ID_Continue}]*`;
	const namedField = new RegExp(`^(${identifier}(?:\\s*,\\s*${identifier})*)\\s+`, "u");
	const embeddedField = new RegExp(`^\\*?\\s*(?:${identifier}\\s*\\.\\s*)?(${identifier})(?:\\s*\\[|\\s*$)`, "u");
	const exported = [];
	for (const declaration of declarations) {
		const named = namedField.exec(declaration);
		if (named) {
			for (const name of named[1].split(",").map((value) => value.trim())) {
				if (/^\p{Lu}/u.test(name)) exported.push(name);
			}
			continue;
		}
		const embedded = embeddedField.exec(declaration);
		if (embedded && /^\p{Lu}/u.test(embedded[1])) exported.push(embedded[1]);
	}
	return exported;
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
  ["internal/observe", ["internal/canon", "internal/domain", "internal/observe/eligibilitycore", "internal/world"]],
  ["internal/observe/eligibilitycore", ["internal/domain"]],
  ["internal/reduce", ["internal/canon", "internal/compare", "internal/domain"]],
  ["internal/reduction", ["internal/canon", "internal/domain", "internal/reduce", "internal/store"]],
  ["internal/store", ["internal/canon", "internal/choice/promotion/authority", "internal/compare", "internal/confirmation/authority", "internal/domain", "internal/emit/node/authority", "internal/reduce"]],
  ["internal/confirmation", ["internal/canon", "internal/compare", "internal/confirmation/authority", "internal/confirmation/internal/publication", "internal/domain", "internal/observe", "internal/reduce", "internal/reduction", "internal/world"]],
  ["internal/confirmation/authority", ["internal/confirmation/internal/publication"]],
  ["internal/confirmation/internal/publication", ["internal/canon", "internal/domain"]],
  ["internal/portablevalue", ["internal/canon"]],
  ["internal/projectionprofile", ["internal/canon", "internal/domain", "internal/portablevalue"]],
  ["internal/projectiontranslate", ["internal/adapters/cli", "internal/adapters/http", "internal/canon", "internal/compare", "internal/domain", "internal/portablevalue", "internal/projectionprofile"]],
  ["internal/choice", ["internal/canon", "internal/compare", "internal/confirmation", "internal/domain", "internal/portablevalue", "internal/projectionprofile", "internal/projectiontranslate", "internal/reduce"]],
  ["internal/choice/promotion", ["internal/choice", "internal/choice/promotion/internal/publication", "internal/confirmation", "internal/domain", "internal/projectionprofile", "internal/store"]],
  ["internal/choice/promotion/authority", ["internal/choice/promotion/internal/publication"]],
  ["internal/choice/promotion/internal/publication", ["internal/canon", "internal/domain"]],
  ["internal/world", ["internal/adapters/cli/model", "internal/adapters/http/model", "internal/canon", "internal/domain", "internal/gitobj", "internal/runnerprofile"]],
]));

function admitsC3StoreBridge(manifest, entry, imports) {
  if (entry.path !== "internal/store/nonhead_contract.go") return false;
  const modelImports = imports.filter((candidate) => candidate.path === `${modulePrefix}${c3StoreModelImport}`);
  if (modelImports.length !== 1 || modelImports[0].alias !== "contractmodel") return false;
  const ownerPath = entry.path;
  const storeEntries = productionEntries(manifest).filter((candidate) =>
    candidate.path.startsWith("internal/store/") && candidate.path.slice(0, candidate.path.lastIndexOf("/")) === "internal/store");
  const declarations = (typeName) => {
    const escaped = typeName.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
    const expression = new RegExp(`\\btype\\s+${escaped}\\b`, "gu");
    return storeEntries.flatMap((candidate) =>
      topLevelMatches(candidate.lexical.code, expression).map(() => ({ path: candidate.path, entry: candidate })));
  };
  for (const typeName of ["ConformanceAttemptInput", "ConformanceAttemptRoots", "ConformanceAttemptRecord", "ContractTargetRecord"]) {
    const found = declarations(typeName);
    if (found.length !== 1 || found[0].path !== ownerPath || topLevelStructBodies(found[0].entry.lexical.code, typeName).length !== 1) {
      return false;
    }
  }
  const code = entry.lexical.code;
  const inputMatches = topLevelMatches(code, /\btype\s+ConformanceAttemptInput\s+struct\s*\{/gu);
  if (inputMatches.length !== 1) return false;
  const inputOpen = inputMatches[0].index + inputMatches[0][0].lastIndexOf("{");
  const inputBlock = balancedBlock(code, inputOpen);
  if (!inputBlock) return false;
  const exactInputBody = [
    "ContractBundleDigest domain.Digest",
    "ResidueHeadDigest domain.Digest",
    "TreeIdentityDigest domain.Digest",
    "MaterializationPolicyDigest domain.Digest",
  ].join(" ");
  if (compactCode(entry.lexical.commentless.slice(inputOpen + 1, inputBlock.end - 1)) !== exactInputBody) return false;
  const methodExpression = /\bfunc\s*\(\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?\*ObjectStore\s*\)\s+(AllocateConformanceAttempt|OpenConformanceAttempt|PersistContractTargetRecord|OpenContractTargetRecord)\s*\(/gu;
  const methods = storeEntries.flatMap((candidate) =>
    topLevelMatches(candidate.lexical.code, methodExpression).map((match) => ({ name: match[1], path: candidate.path })));
  if (methods.some((method) => method.path !== ownerPath) || !exactSet(methods.map((method) => method.name), [
    "AllocateConformanceAttempt", "OpenConformanceAttempt", "PersistContractTargetRecord", "OpenContractTargetRecord",
  ])) return false;
  const exactMethods = [
    /\bfunc\s*\(\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?\*ObjectStore\s*\)\s+AllocateConformanceAttempt\s*\(\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?context\.Context\s*,\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?ConformanceAttemptInput\s*\)\s*\(\s*ConformanceAttemptRecord\s*,\s*error\s*\)\s*\{/gu,
    /\bfunc\s*\(\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?\*ObjectStore\s*\)\s+OpenConformanceAttempt\s*\(\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?context\.Context\s*,\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?domain\.Digest\s*\)\s*\(\s*ConformanceAttemptRecord\s*,\s*error\s*\)\s*\{/gu,
    /\bfunc\s*\(\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?\*ObjectStore\s*\)\s+PersistContractTargetRecord\s*\(\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?context\.Context\s*,\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?ConformanceAttemptRecord\s*,\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?contractmodel\.ContractExecutionTarget\s*\)\s*\(\s*ContractTargetRecord\s*,\s*error\s*\)\s*\{/gu,
    /\bfunc\s*\(\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?\*ObjectStore\s*\)\s+OpenContractTargetRecord\s*\(\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?context\.Context\s*,\s*(?:[_\p{ID_Start}][\p{ID_Continue}_]*\s+)?ConformanceAttemptRecord\s*\)\s*\(\s*ContractTargetRecord\s*,\s*error\s*\)\s*\{/gu,
  ];
  return exactMethods.every((expression) => topLevelMatches(code, expression).length === 1);
}

function admitsC4StoreBridge(manifest, entry, imports, diagnostics = []) {
	const reject = (reason) => {
		diagnostics.push(reason);
		return false;
	};
	if (entry.path !== c4StoreOwnerPath) return reject("owner-path");
	const modelImports = imports.filter((candidate) => candidate.path === `${modulePrefix}${c3StoreModelImport}`);
	const epochImports = imports.filter((candidate) => candidate.path === `${modulePrefix}${c4StoreHostEpochImport}`);
	if (modelImports.length !== 1 || modelImports[0].alias !== "contractmodel" ||
		epochImports.length !== 1 || epochImports[0].alias !== "") return reject("imports");
  const storeEntries = productionEntries(manifest).filter((candidate) =>
    candidate.path.startsWith("internal/store/") && candidate.path.slice(0, candidate.path.lastIndexOf("/")) === "internal/store");
  const typeNames = [
    "ContractRunOwner", "PrivateRunManifest", "FinalizedRunRecord", "TerminalClosure", "ContractExecutionRecord",
  ];
  for (const typeName of typeNames) {
    const escaped = typeName.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
    const found = storeEntries.flatMap((candidate) =>
      topLevelMatches(candidate.lexical.code, new RegExp(`\\btype\\s+${escaped}\\b`, "gu"))
        .map(() => ({ path: candidate.path, entry: candidate })));
		if (found.length !== 1 || found[0].path !== c4StoreOwnerPath) return reject(`type-owner:${typeName}`);
		const bodies = topLevelStructBodies(found[0].entry.lexical.code, typeName);
		if (bodies.length !== 1 || exportedStructFields(bodies[0]).length !== 0) return reject(`type-opacity:${typeName}`);
  }
  const code = entry.lexical.code;
  const types = topLevelMatches(code, /\btype\s+([A-Z][A-Za-z0-9_]*)\b/gu).map((match) => match[1]);
  const functions = topLevelMatches(code, /\bfunc\s+([A-Z][A-Za-z0-9_]*)\s*\(/gu).map((match) => match[1]);
	const c4ReceiverPattern = typeNames.join("|");
	const exportedMethodExpression = new RegExp(
		`\\bfunc\\s*\\(\\s*(?:[_\\p{ID_Start}][\\p{ID_Continue}_]*\\s+)?(\\*)?(${c4ReceiverPattern})\\s*\\)\\s+([A-Z][A-Za-z0-9_]*)\\s*\\(`,
		"gu",
	);
	const methodDeclarations = storeEntries.flatMap((candidate) =>
		topLevelMatches(candidate.lexical.code, exportedMethodExpression).map((match) => ({
			name: `${match[2]}.${match[3]}`,
			path: candidate.path,
			pointer: match[1] === "*",
		})),
	);
	const foreignMethod = methodDeclarations.find((method) => method.path !== c4StoreOwnerPath);
	if (foreignMethod) return reject(`method-owner:${foreignMethod.name}`);
	const methods = methodDeclarations.map((method) => method.name);
  const expectedSurface = [
    "AcquireContractRunOwner", "ContractExecutionRecord", "ContractExecutionRecord.Digest",
    "ContractExecutionRecord.Model", "ContractExecutionRecord.Valid", "ContractRunOwner",
    "ContractRunOwner.ConsumeForStart", "ContractRunOwner.PersistFinalizedRun",
    "ContractRunOwner.PersistPrivateRunManifest", "ContractRunOwner.PersistSpawnObservation",
    "ContractRunOwner.StartClaimDigest", "FinalizedRunRecord", "FinalizedRunRecord.Digest",
    "FinalizedRunRecord.Model", "FinalizedRunRecord.Valid", "OpenFinalizedRunRecord", "OpenTerminalClosure",
    "PersistContractExecutionRecord", "PrivateRunManifest", "PrivateRunManifest.EvidenceRef",
    "PrivateRunManifest.Summary", "PrivateRunManifest.Valid", "TerminalClosure",
    "TerminalClosure.FinalizedRun", "TerminalClosure.Release",
  ];
	if (!exactSet([...types, ...functions, ...methods], expectedSurface)) return reject("exported-surface");

  const escapedType = (value) => value.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
  const named = "(?:[_\\p{ID_Start}][\\p{ID_Continue}_]*\\s+)?";
	const callable = (receiver, name, parameters, results) => {
		const owner = receiver === null ? "" : `\\(\\s*${named}${escapedType(receiver)}\\s*\\)\\s+`;
		const params = parameters.map((type) => `${named}${escapedType(type)}`).join("\\s*,\\s*");
		const trailingParameterComma = parameters.length === 0 ? "" : "\\s*,?";
		const returns = results.length === 1 ? `\\s+${escapedType(results[0])}` :
			`\\s*\\(\\s*${results.map(escapedType).join("\\s*,\\s*")}\\s*\\)`;
		const expression = new RegExp(
			`\\bfunc\\s+${owner}${name}\\s*\\(\\s*${params}${trailingParameterComma}\\s*\\)${returns}\\s*\\{`,
			"gu",
		);
    return topLevelMatches(code, expression).length === 1;
  };
	const callables = [
    [null, "AcquireContractRunOwner", ["context.Context", "ContractTargetRecord", "hostepoch.Epoch"], ["ContractRunOwner", "error"]],
    ["ContractRunOwner", "ConsumeForStart", ["context.Context", "domain.Digest"], ["error"]],
    ["ContractRunOwner", "StartClaimDigest", [], ["domain.Digest"]],
    ["ContractRunOwner", "PersistSpawnObservation", ["context.Context", "contractmodel.SpawnObservation"], ["error"]],
    ["ContractRunOwner", "PersistPrivateRunManifest", ["context.Context", "map[contractmodel.EvidenceKind][]byte"], ["PrivateRunManifest", "error"]],
    ["ContractRunOwner", "PersistFinalizedRun", ["context.Context", "PrivateRunManifest", "contractmodel.FinalizedContractRun"], ["TerminalClosure", "error"]],
    ["PrivateRunManifest", "Valid", [], ["bool"]],
    ["PrivateRunManifest", "Summary", [], ["contractmodel.PrivateManifestSummary", "error"]],
    ["PrivateRunManifest", "EvidenceRef", ["contractmodel.EvidenceKind"], ["contractmodel.EvidenceRef", "error"]],
    ["FinalizedRunRecord", "Valid", [], ["bool"]],
    ["FinalizedRunRecord", "Digest", [], ["domain.Digest"]],
    ["FinalizedRunRecord", "Model", [], ["contractmodel.FinalizedContractRun"]],
    [null, "OpenFinalizedRunRecord", ["context.Context", "ContractTargetRecord", "contractmodel.ContractExecutionTarget"], ["FinalizedRunRecord", "error"]],
    [null, "OpenTerminalClosure", ["context.Context", "ContractTargetRecord", "contractmodel.ContractExecutionTarget"], ["TerminalClosure", "error"]],
    ["TerminalClosure", "FinalizedRun", [], ["FinalizedRunRecord"]],
    ["TerminalClosure", "Release", ["context.Context"], ["error"]],
    [null, "PersistContractExecutionRecord", ["context.Context", "FinalizedRunRecord", "contractmodel.ContractExecution"], ["ContractExecutionRecord", "error"]],
    ["ContractExecutionRecord", "Valid", [], ["bool"]],
    ["ContractExecutionRecord", "Digest", [], ["domain.Digest"]],
    ["ContractExecutionRecord", "Model", [], ["contractmodel.ContractExecution"]],
	];
	for (const [receiver, name, parameters, results] of callables) {
		if (!callable(receiver, name, parameters, results)) {
			return reject(`callable:${receiver === null ? "function" : receiver}.${name}`);
		}
	}
	return true;
}

function inspectPackageBoundaries(manifest, violations) {
  const aggregate = new Map([...exactInternalImports.keys()].map((key) => [key, new Set()]));
  let c3StoreBridge = false;
  let c4StoreBridge = false;
	const c4WorldMechanicsObserved = new Set();
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
		const hasStoreModel = internalImports.includes(c3StoreModelImport);
		const hasHostEpoch = internalImports.includes(c4StoreHostEpochImport);
		if (entry.path === c4StoreOwnerPath && (hasStoreModel || hasHostEpoch)) {
			const bridgeDiagnostics = [];
			if (!hasStoreModel || !hasHostEpoch || !admitsC4StoreBridge(manifest, entry, imports, bridgeDiagnostics) || c4StoreBridge) {
				violations.push(["U6_STORE_C4_BRIDGE_NOT_ADMITTED", `${entry.path}: ${bridgeDiagnostics[0] ?? "imports-or-duplicate"}`]);
			} else {
				c4StoreBridge = true;
			}
		} else {
			if (hasStoreModel) {
				if (!admitsC3StoreBridge(manifest, entry, imports) || c3StoreBridge) {
					violations.push(["U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED", entry.path]);
				} else {
					c3StoreBridge = true;
				}
			}
			if (hasHostEpoch) violations.push(["U6_STORE_C4_BRIDGE_NOT_ADMITTED", entry.path]);
		}
		if (internalImports.includes(c4ProcessMechanicsImport)) {
			const mechanicsImports = imports.filter((candidate) =>
				candidate.path === `${modulePrefix}${c4ProcessMechanicsImport}`);
			if (!c4WorldMechanicsPaths.includes(entry.path) || mechanicsImports.length !== 1 ||
				mechanicsImports[0].alias !== "" || c4WorldMechanicsObserved.has(entry.path)) {
				violations.push(["U6_WORLD_C4_MECHANICS_IMPORT_NOT_ADMITTED", entry.path]);
			} else {
				c4WorldMechanicsObserved.add(entry.path);
			}
		}
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
		const phaseExpected = [...expected];
		const c4WorldMechanics = exactSet([...c4WorldMechanicsObserved], [...c4WorldMechanicsPaths]);
		if (packageDirectory === "internal/store" && (c3StoreBridge || c4StoreBridge)) phaseExpected.push(c3StoreModelImport);
		if (packageDirectory === "internal/store" && c4StoreBridge) phaseExpected.push(c4StoreHostEpochImport);
		if (packageDirectory === "internal/world" && c4WorldMechanics) phaseExpected.push(c4ProcessMechanicsImport);
    if (!exactSet(actual, phaseExpected)) {
      violations.push(["U6_INTERNAL_IMPORT_LATTICE", `${packageDirectory}: ${actual.sort().join(",")}`]);
    }
  }
}

function inspectEligibilityCore(manifest, violations) {
  const coreEntries = productionEntries(manifest).filter((entry) =>
    entry.path.startsWith("internal/observe/eligibilitycore/"));
  if (coreEntries.length !== 1 || coreEntries[0].path !== "internal/observe/eligibilitycore/eligibility.go") {
    violations.push(["U6_ELIGIBILITY_CORE_FILE_MAP", coreEntries.map((entry) => entry.path).join(",") || "absent"]);
    return;
  }
  const entry = coreEntries[0];
  const core = entry.lexical.code;
  const imported = importedPackages(entry.lexical.commentless).map((candidate) => candidate.path).sort();
  const expectedImport = `${modulePrefix}internal/domain`;
  if (JSON.stringify(imported) !== JSON.stringify([expectedImport])) {
    violations.push(["U6_ELIGIBILITY_CORE_IMPORT_ROSTER", imported.join(",")]);
  }
  const exportedTypes = [...core.matchAll(/\btype\s+([A-Z][A-Za-z0-9_]*)\b/gu)].map((match) => match[1]).sort();
  const exportedFunctions = [...core.matchAll(/\bfunc\s+([A-Z][A-Za-z0-9_]*)\s*\(/gu)].map((match) => match[1]).sort();
  const exportedMethods = [...core.matchAll(/\bfunc\s*\(\s*[A-Za-z_][A-Za-z0-9_]*\s+Decision\s*\)\s*([A-Z][A-Za-z0-9_]*)\s*\(/gu)]
    .map((match) => match[1]).sort();
  const exportedConstants = [...core.matchAll(/^\s*([A-Z][A-Za-z0-9_]*)\s+FactKind\s*=/gmu)].map((match) => match[1]).sort();
  if (JSON.stringify(exportedTypes) !== JSON.stringify(["Decision", "FactKind"]) ||
      JSON.stringify(exportedFunctions) !== JSON.stringify(["Select"]) ||
      JSON.stringify(exportedMethods) !== JSON.stringify(["IsEligible", "Reasons"]) ||
      JSON.stringify(exportedConstants) !== JSON.stringify(["BehaviorCaptured", "ControlIneligible"]) ||
      /\bvar\s+[A-Z][A-Za-z0-9_]*/u.test(core)) {
    violations.push(["U6_ELIGIBILITY_CORE_API_ROSTER",
      `types=${exportedTypes};funcs=${exportedFunctions};methods=${exportedMethods};consts=${exportedConstants}`]);
  }
  const decisionBodies = topLevelStructBodies(core, "Decision");
  if (decisionBodies.length !== 1 || compactCode(decisionBodies[0]) !== "eligible bool reasons []domain.ControlReason") {
    violations.push(["U6_ELIGIBILITY_CORE_STORAGE", decisionBodies.map(compactCode).join(",") || "absent"]);
  }
  const selectBody = functionBody(core, /\bfunc\s+Select\s*\(/u) ?? "";
  for (const anchor of [
    "kind == BehaviorCaptured", "len(reasons) != 0", "kind != ControlIneligible",
    "len(reasons) == 0", "len(reasons) > 3", "if index > 0 && !isTeardown(reason)",
    "return Decision{reasons: append([]domain.ControlReason(nil), reasons...)}",
  ]) {
    requireIncludes(violations, selectBody, anchor, "U6_ELIGIBILITY_CORE_SELECT_DATAFLOW", anchor);
  }
  const owner = codeAt(manifest, "internal/observe/eligibility.go");
  const ownerBody = functionBody(owner, /\bfunc\s+Eligible\s*\(/u) ?? "";
  if ((ownerBody.match(/\beligibilitycore\s*\.\s*Select\s*\(/gu) ?? []).length !== 2 ||
      !ownerBody.includes("eligibilitycore.Select(eligibilitycore.ControlIneligible, fact.controls)") ||
      !ownerBody.includes("eligibilitycore.Select(eligibilitycore.BehaviorCaptured, nil)")) {
    violations.push(["U6_ELIGIBILITY_OWNER_DATAFLOW", "internal/observe/eligibility.go:Eligible"]);
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
	const parseBody = functionBody(profile, /\bfunc\s+Parse\s*\(/u) ?? "";
	if (internalReferences !== 3 || !exactSet(internalFiles, ["internal/projectionprofile/profile.go"]) ||
		(validBody.match(/\bNewDerived\b/gu) ?? []).length !== 1 ||
		(parseBody.match(/\bNewDerived\b/gu) ?? []).length !== 1 ||
		!parseBody.includes("bytes.Equal(rebuilt.CanonicalBytes(), exact)")) {
		violations.push(["U6_PROFILE_INTERNAL_REBUILD_SURFACE_NOT_EXACT", `${internalReferences}:${internalFiles.join(",")}`]);
	}

	const translate = codeAt(manifest, "internal/projectiontranslate/translate.go");
	const resolveBody = functionBody(translate, /\bfunc\s+Resolve\s*\(/u) ?? "";
	const firewall = resolveBody.indexOf("if err := verifyPortableProfileRosterV1(); err != nil");
	const adapterSwitch = resolveBody.indexOf("switch binding.AdapterDomain()");
	if (firewall < 0 || adapterSwitch < 0 || firewall >= adapterSwitch ||
		(resolveBody.match(/\bverifyPortableProfileRosterV1\s*\(/gu) ?? []).length !== 1) {
		violations.push(["U6_PROFILE_ROSTER_FIREWALL_NOT_EXACT", "projectiontranslate.Resolve"]);
	}
	const roster = commentlessAt(manifest, "internal/projectiontranslate/profile_roster.go");
	for (const anchor of [
		"portableProfileRosterV1EntryCount           = 128",
		'portableProfileRosterV1Digest               = "sha256:aec21f3f76f9384a09ab3f71cea3d1def86fb8ef725322f35af296aa8e1560b7"',
		"portableExpectationDomainRosterV1EntryCount = 128",
		'portableExpectationDomainRosterV1Digest     = "sha256:d0951b6681fc1893691db83accfb5aeaac2a500cba9d4814308518aeff4566c7"',
		'portableModeSemanticsV1Digest               = "sha256:9b51d42ab58078eac4d652fbaeb925d614803d5926bf63e7c59fba4134f57cae"',
		"for mask := 1; mask < 1<<len(registry); mask++",
		"newCLIResolvedProfile(definition, registry)",
		"counterhttp.NewHTTPProjectionDefinition()",
		"resolveHTTP(httpDefinition.Binding())",
		'canon.DigestTyped("PortableProjectionProfileRosterV1"',
		'canon.DigestTyped("PortableExpectationDomainRosterV1"',
		'canon.DigestTyped("PortableModeSemanticsV1"',
		"newExpectationDomain(resolved)",
		"PortableChoiceModeV1",
		"portableTupleIdentityRuleV1",
		"portableSelectionRuleV1",
		"portableProofRuleV1",
		"portablevalue.MaxTupleEncodedBytes",
		"ProjectionBindingBase64: base64.StdEncoding.EncodeToString(binding.CanonicalBytes())",
		"ProfileBase64:           base64.StdEncoding.EncodeToString(profile.CanonicalBytes())",
	]) requireIncludes(violations, roster, anchor, "U6_PROFILE_ROSTER_DERIVATION_NOT_EXACT", anchor);
	const rosterFirewall = functionBody(roster, /\bfunc\s+verifyPortableProfileRosterV1\s*\(/u) ?? "";
	for (const anchor of [
		"derivePortableProfileRosterV1()",
		"derivePortableExpectationDomainRosterV1()",
		"derivePortableModeSemanticsV1()",
		"portableProfileRosterV1EntryCount",
		"portableExpectationDomainRosterV1EntryCount",
		"portableModeSemanticsV1Digest",
	]) requireIncludes(violations, rosterFirewall, anchor, "U6_PORTABLE_MODE_SEMANTICS_FIREWALL_NOT_EXACT", anchor);
	const cliTranslator = commentlessAt(manifest, "internal/projectiontranslate/cli.go");
	const httpTranslator = commentlessAt(manifest, "internal/projectiontranslate/http.go");
	if (!cliTranslator.includes('cliTranslatorName    = "CLI_PROJECTION_TO_PORTABLE"') ||
		!cliTranslator.includes('cliTranslatorVersion = "v1"') ||
		!httpTranslator.includes('httpTranslatorName    = "HTTP_PROJECTION_TO_PORTABLE"') ||
		!httpTranslator.includes('httpTranslatorVersion = "v1"')) {
		violations.push(["U6_PROFILE_ROSTER_TRANSLATOR_IDENTITY_NOT_EXACT", "internal/projectiontranslate"]);
	}
	const rosterTest = commentlessAt(manifest, "internal/projectiontranslate/resolve_test.go");
	const rosterTestBody = functionBody(rosterTest, /\bfunc\s+TestPortableProfileRosterV1IsRuntimeFrozen\s*\(/u) ?? "";
	for (const anchor of [
		'const expected = "sha256:aec21f3f76f9384a09ab3f71cea3d1def86fb8ef725322f35af296aa8e1560b7"',
		'const expectedExpectation = "sha256:d0951b6681fc1893691db83accfb5aeaac2a500cba9d4814308518aeff4566c7"',
		'const expectedSemantics = "sha256:9b51d42ab58078eac4d652fbaeb925d614803d5926bf63e7c59fba4134f57cae"',
		"derivePortableProfileRosterV1()",
		"derivePortableExpectationDomainRosterV1()",
		"derivePortableModeSemanticsV1()",
		"count != portableProfileRosterV1EntryCount",
		"expectationCount != portableExpectationDomainRosterV1EntryCount",
		"semanticsDigest.String() != expectedSemantics",
		"digest.String() != expected",
		"verifyPortableProfileRosterV1()",
	]) requireIncludes(violations, rosterTestBody, anchor, "U6_PROFILE_ROSTER_GOLDEN_TEST_NOT_EXACT", anchor);
}

function inspectPortableRulingBoundaries(manifest, violations) {
  const choicepoint = codeAt(manifest, "internal/choice/choicepoint.go");
  const portable = codeAt(manifest, "internal/choice/portable.go");
  const validation = codeAt(manifest, "internal/choice/validation.go");
  const blind = codeAt(manifest, "internal/choice/blind.go");
  const session = codeAt(manifest, "internal/choice/session.go");
  const promotion = codeAt(manifest, "internal/choice/promotion/service.go");
  const choiceEntries = productionEntries(manifest).filter((entry) =>
    entry.path.slice(0, entry.path.lastIndexOf("/")) === "internal/choice");
  const choiceCode = choiceEntries.map((entry) => entry.lexical.code).join("\n");

  for (const entry of choiceEntries) {
    if (/\bdomain\s*\.\s*Adapter(?:CLI|HTTP)\b/gu.test(entry.lexical.code)) {
      violations.push(["U6_CHOICE_SWITCHES_ON_ADAPTER_DOMAIN", entry.path]);
    }
    if (/\bStrictTranslate\b/gu.test(entry.lexical.code)) {
      violations.push(["U6_CHOICE_BYPASSES_PROOF_FIRST_TRANSLATION", entry.path]);
    }
  }
  for (const forbidden of [
    "ProjectionDefinition",
    "ProjectionDefinitionConfig",
    "NewProjectionDefinition",
    "NewFieldRegistry",
    "NewConfirmedOutcomeSet",
    "NewWholeProjectionRegistry",
  ]) {
    if (new RegExp(`\\b${forbidden}\\b`, "u").test(choiceCode)) {
      violations.push(["U6_CALLER_AUTHORED_PROJECTION_AUTHORITY_RESIDUE", forbidden]);
    }
  }
  if (/\bprojectiontranslate\s*\.\s*Resolve\b/u.test(choiceCode)) {
    violations.push(["U6_CHOICE_BYPASSES_TRANSLATOR_RESOLVER", "internal/choice"]);
  }

  const freshConstructor = functionBody(choicepoint, /\bfunc\s+NewChoicepointRecord\s*\(/u) ?? "";
  if (compactCode(freshConstructor) !== "return buildChoicepointRecord(input, choicepointPortable, nil)") {
    violations.push(["U6_FRESH_CHOICEPOINT_MODE_NOT_EXACT", "NewChoicepointRecord"]);
  }
  if ((choiceCode.match(/\bbuildChoicepointRecord\b/gu) ?? []).length !== 3 ||
      choiceEntries.filter((entry) => /\bbuildChoicepointRecord\b/u.test(entry.lexical.code)).some((entry) => entry.path !== "internal/choice/choicepoint.go")) {
    violations.push(["U6_CHOICEPOINT_BUILD_SURFACE_NOT_EXACT", "internal/choice/choicepoint.go"]);
  }
  const buildChoicepoint = functionBody(choicepoint, /\bfunc\s+buildChoicepointRecord\s*\(/u) ?? "";
  const parseChoicepoint = functionBody(choicepoint, /\bfunc\s+ParseChoicepointRecord\s*\(/u) ?? "";
  for (const anchor of [
    "case choicepointPortable:",
    "case choicepointLegacyWhole:",
    "expectedStimulusKind = input.Plan.Adapter().Domain.CanonicalStimulusKind()",
  ]) requireIncludes(violations, buildChoicepoint, anchor, "U6_CHOICEPOINT_MODE_DISPATCH_NOT_EXACT", anchor);
  for (const anchor of [
    "case legacyWholeProjectionMode:",
    "mode = choicepointLegacyWhole",
    "case portableProjectionMode:",
    "mode = choicepointPortable",
    "default:",
  ]) requireIncludes(violations, parseChoicepoint, anchor, "U6_CHOICEPOINT_PARSE_MODE_DISPATCH_NOT_EXACT", anchor);
  if ((choiceCode.match(/\bCanonicalStimulusKind\s*\(/gu) ?? []).length !== 1) {
    violations.push(["U6_LEGACY_STIMULUS_KIND_OWNER_NOT_EXACT", "internal/choice/choicepoint.go"]);
  }

  if ((choiceCode.match(/\bprojectiontranslate\s*\.\s*TranslateConfirmed\b/gu) ?? []).length !== 1 ||
      (buildChoicepoint.match(/\bprojectiontranslate\s*\.\s*TranslateConfirmed\s*\(/gu) ?? []).length !== 1) {
    violations.push(["U6_PROOF_FIRST_TRANSLATION_SURFACE_NOT_EXACT", "internal/choice/choicepoint.go"]);
  }
  if ((choiceCode.match(/\bconfirmedOutcomeSetFromTranslations\b/gu) ?? []).length !== 2 ||
      (buildChoicepoint.match(/\bconfirmedOutcomeSetFromTranslations\s*\(/gu) ?? []).length !== 1) {
    violations.push(["U6_TRANSLATED_OUTCOME_CONSUMER_SURFACE_NOT_EXACT", "internal/choice"]);
  }
  for (const [identifier, expectedFile] of [
    ["newPortableFieldRegistry", "internal/choice/portable.go"],
    ["newWholeProjectionRegistry", "internal/choice/validation.go"],
    ["newLegacyConfirmedOutcomeSet", "internal/choice/validation.go"],
  ]) {
    const references = choiceEntries.filter((entry) => new RegExp(`\\b${identifier}\\b`, "u").test(entry.lexical.code));
    const count = references.reduce((total, entry) => total + (entry.lexical.code.match(new RegExp(`\\b${identifier}\\b`, "gu")) ?? []).length, 0);
    if (count !== 2 || !references.some((entry) => entry.path === expectedFile)) {
      violations.push(["U6_PORTABLE_LEGACY_CONSTRUCTION_SURFACE_NOT_EXACT", identifier]);
    }
  }

  const registry = functionBody(portable, /\bfunc\s+newPortableFieldRegistry\s*\(/u) ?? "";
  for (const anchor of [
    "profile.Valid()",
    "profile.Fields()",
    "newFieldRegistry(definitions, true)",
    "registry.sourceKinds",
    "profile.Digest()",
    "profile.Binding().Digest()",
    "registry.mode = fieldRegistryPortable",
  ]) requireIncludes(violations, registry, anchor, "U6_PORTABLE_REGISTRY_DERIVATION_NOT_EXACT", anchor);
	for (const anchor of [
		"expectation.Valid()",
		"expectation.ProfileDigest() != profile.Digest()",
		"bytes.Equal(expectation.ProfileBytes(), profile.CanonicalBytes())",
		"ExpectationDomainDigest: expectation.Digest().String()",
		"registry.expectationDomain = expectation",
	]) requireIncludes(violations, registry, anchor, "U6_PORTABLE_REGISTRY_EXPECTATION_BINDING_NOT_EXACT", anchor);
  if (!/\bfunc\s+newPortableFieldRegistry\s*\(\s*profile\s+projectionprofile\.Profile\s*,\s*expectation\s+projectiontranslate\.ExpectationDomain\s*\)/u.test(portable) ||
      /\b(?:proof|tuple|translation|CanonicalProjection)\b/u.test(registry)) {
    violations.push(["U6_PORTABLE_REGISTRY_ACCEPTS_NONPROFILE_AUTHORITY", "internal/choice/portable.go"]);
  }
	const expectationSource = codeAt(manifest, "internal/projectiontranslate/expectation.go");
	const expectationCommentless = commentlessAt(manifest, "internal/projectiontranslate/expectation.go");
	const expectationBodies = topLevelStructBodies(expectationSource, "ExpectationDomain");
	const expectedExpectationBody = "profile projectionprofile.Profile arm adapterArm digest domain.Digest canonical []byte seal *expectationDomainSeal";
	if (expectationBodies.length !== 1 || exportedStructFields(expectationBodies[0]).length !== 0 ||
		compactCode(expectationBodies[0]) !== expectedExpectationBody ||
		!expectationCommentless.includes('PortableChoiceModeV1 = "ADAPTER_BOUND_PORTABLE_FIELDS_V1"') ||
		!expectationCommentless.includes('portableExpectationSemanticsV1 = "EXISTS_COMPLETE_ADAPTER_PROJECTION_RESTRICTION_V1"')) {
		violations.push(["U6_EXPECTATION_DOMAIN_SURFACE_NOT_CLOSED", "internal/projectiontranslate/expectation.go"]);
	}
	const expectationConstructor = functionBody(expectationCommentless, /\bfunc\s+newExpectationDomain\s*\(/u) ?? "";
	for (const anchor of [
		"resolved.Valid()",
		"expectationRules(resolved.profile, resolved.arm)",
		"resolved.profile.Binding().Digest().String()",
		"resolved.profile.Digest().String()",
		"base64.StdEncoding.EncodeToString(resolved.profile.CanonicalBytes())",
		'canon.DigestTyped("PortableExpectationDomain"',
		"seal: expectationDomainAuthority",
	]) requireIncludes(violations, expectationConstructor, anchor, "U6_EXPECTATION_DOMAIN_IDENTITY_NOT_EXACT", anchor);
	const validateExpectation = functionBody(expectationSource, /\bfunc\s*\(d\s+ExpectationDomain\)\s+ValidateSelected\s*\(/u) ?? "";
	for (const anchor of [
		"d.Valid()",
		"position <= lastPosition",
		"portablevalue.ValidateTuple(orderedValues)",
		"validateCLISelectedExpectation(profileFields, values)",
		"validateHTTPSelectedExpectation(values)",
	]) requireIncludes(violations, validateExpectation, anchor, "U6_EXPECTATION_DOMAIN_VALIDATION_NOT_EXACT", anchor);
	const expectationValid = functionBody(expectationSource, /\bfunc\s*\(d\s+ExpectationDomain\)\s+Valid\s*\(/u) ?? "";
	const expectationValidGuard = expectationValid.indexOf("if d.seal != expectationDomainAuthority");
	if (expectationValidGuard < 0 || expectationValid.slice(0, expectationValidGuard).trim() !== "") {
		violations.push(["U6_EXPECTATION_DOMAIN_VALIDITY_NOT_EXACT", "ExpectationDomain.Valid prefix"]);
	}
	for (const anchor of [
		"d.seal != expectationDomainAuthority",
		"d.profile.Valid()",
		"d.digest.Valid()",
		"newExpectationDomain(Resolved{",
		"rebuilt.digest == d.digest",
		"bytes.Equal(rebuilt.canonical, d.canonical)",
	]) requireIncludes(violations, expectationValid, anchor, "U6_EXPECTATION_DOMAIN_VALIDITY_NOT_EXACT", anchor);
	const expectationValidationGuard = validateExpectation.indexOf("if !d.Valid()");
	if (expectationValidationGuard < 0 || validateExpectation.slice(0, expectationValidationGuard).trim() !== "") {
		violations.push(["U6_EXPECTATION_DOMAIN_VALIDATION_NOT_EXACT", "ExpectationDomain.ValidateSelected prefix"]);
	}

  const translatedSet = functionBody(portable, /\bfunc\s+confirmedOutcomeSetFromTranslations\s*\(/u) ?? "";
	requireIncludes(
		violations,
		translatedSet,
		"newPortableFieldRegistry(translations.Profile(), translations.ExpectationDomain())",
		"U6_EXPECTATION_DOMAIN_NOT_PROPAGATED_TO_CHOICE",
		"confirmedOutcomeSetFromTranslations",
	);
  const outcomeSort = translatedSet.indexOf("sort.Slice(set.ordered");
  const outcomeSeal = translatedSet.indexOf("makeConfirmedOutcomeSetSeal(");
  if (outcomeSort < 0 || outcomeSeal < 0 || outcomeSort >= outcomeSeal) {
    violations.push(["U6_PORTABLE_OUTCOME_ID_ORDER_BEFORE_SEAL", "internal/choice/portable.go"]);
  }
  for (const anchor of [
    "translated.CanonicalProjection()",
    "domain.NewProjectionFingerprint(projection)",
    "computed != fingerprint",
  ]) requireIncludes(violations, translatedSet, anchor, "U6_ORIGINAL_FINGERPRINT_AUTHORITY_NOT_RETAINED", anchor);

  const resolveSelected = functionBody(validation, /\bfunc\s*\(r\s+FieldRegistry\)\s+resolveSelected\s*\(/u) ?? "";
  requireIncludes(violations, resolveSelected, "for _, fieldID := range r.orderedIDs", "U6_SELECTED_FIELD_ORDER_NOT_PROFILE_BOUND", "FieldRegistry.resolveSelected");
  if (/\bsort\s*\./u.test(resolveSelected)) violations.push(["U6_SELECTED_FIELD_ORDER_LEXICAL", "FieldRegistry.resolveSelected"]);
  const differing = functionBody(blind, /\bfunc\s+differingFields\s*\(/u) ?? "";
  requireIncludes(violations, differing, "for fieldIndex, fieldID := range registry.orderedIDs", "U6_DIFFERING_FIELD_ORDER_NOT_PROFILE_BOUND", "differingFields");
  requireIncludes(violations, differing, "identityKey(registry.mode)", "U6_DIFFERING_FIELD_IDENTITY_NOT_EXACT", "differingFields");
  requireIncludes(violations, differing, "if differs", "U6_DIFFERING_FIELD_FILTER_NOT_EXACT", "differingFields");
  if (/\bsort\s*\./u.test(differing)) violations.push(["U6_DIFFERING_FIELD_ORDER_LEXICAL", "differingFields"]);
  const blindView = functionBody(blind, /\bfunc\s+NewBlindView\s*\(/u) ?? "";
  requireIncludes(violations, blindView, "for _, field := range record.confirmed.registry.Definitions()", "U6_SELECTABLE_FIELD_ORDER_NOT_PROFILE_BOUND", "NewBlindView");
  if (/sort\s*\.\s*(?:Strings|Slice)\s*\(\s*selectable\b/u.test(blindView)) {
    violations.push(["U6_SELECTABLE_FIELD_ORDER_LEXICAL", "NewBlindView"]);
  }

  const selectedBodies = topLevelStructBodies(validation, "SelectedTuple");
  const expectedSelectedBody = "universe canon.Digest selectedKey string tupleKey string fields []FieldValue seal *selectedTupleSeal";
  if (selectedBodies.length !== 1 || exportedStructFields(selectedBodies[0]).length !== 0 || compactCode(selectedBodies[0]) !== expectedSelectedBody) {
    violations.push(["U6_SELECTED_TUPLE_SURFACE_NOT_CLOSED", "SelectedTuple"]);
  }
  for (const [source, name] of [[validation, "RulingInput"], [session, "RulingDraftInput"]]) {
    const bodies = topLevelStructBodies(source, name);
    if (bodies.length !== 1 || !/\bCustomExpectation\s+\*SelectedTuple\b/u.test(bodies[0])) {
      violations.push(["U6_CUSTOM_EXPECTATION_NOT_SELECTED_ONLY", name]);
    }
  }
  const selectedValidation = functionBody(validation, /\bfunc\s*\(r\s+FieldRegistry\)\s+validateSelectedTuple\s*\(/u) ?? "";
  for (const anchor of [
    "len(raw) != len(selected)",
    "if _, selectedField := allowed[fieldID.text]; !selectedField",
    "r.validatePortableValues(selectedOrder, values)",
		"r.validatePortableExpectation(selectedOrder, values, true)",
  ]) requireIncludes(violations, selectedValidation, anchor, "U6_SELECTED_TUPLE_VALIDATION_NOT_EXACT", anchor);
	const completeValidation = functionBody(validation, /\bfunc\s*\(r\s+FieldRegistry\)\s+validateTuple\s*\(/u) ?? "";
	const completePortableValues = completeValidation.indexOf("r.validatePortableValues(r.orderedIDs, fields)");
	const completeExpectation = completeValidation.indexOf("r.validatePortableExpectation(r.orderedIDs, fields, false)");
	if (completePortableValues < 0 || completeExpectation < 0 || completePortableValues >= completeExpectation) {
		violations.push(["U6_COMPLETE_TUPLE_EXPECTATION_VALIDATION_NOT_EXACT", "FieldRegistry.validateTuple"]);
	}
	const portableExpectation = functionBody(validation, /\bfunc\s*\(r\s+FieldRegistry\)\s+validatePortableExpectation\s*\(/u) ?? "";
	const choiceExpectationGuard = portableExpectation.indexOf("if r.mode != fieldRegistryPortable");
	if (choiceExpectationGuard < 0 || portableExpectation.slice(0, choiceExpectationGuard).trim() !== "") {
		violations.push(["U6_CHOICE_EXPECTATION_VALIDATION_NOT_EXACT", "FieldRegistry.validatePortableExpectation prefix"]);
	}
	for (const anchor of [
		"r.mode != fieldRegistryPortable",
		"r.expectationDomain.Valid()",
		"r.expectationDomain.ProfileDigest() != r.profileDigest",
		"projectiontranslate.SelectedValue{FieldID: fieldID, Value: portable}",
		"r.expectationDomain.ValidateSelected(selected)",
		"projectiontranslate.CodeCustomExpectationNotRealizable",
		"CodeCustomExpectationNotAdapterRealizable",
	]) requireIncludes(violations, portableExpectation, anchor, "U6_CHOICE_EXPECTATION_VALIDATION_NOT_EXACT", anchor);
	if ((validation.match(/\bvalidatePortableExpectation\s*\(/gu) ?? []).length !== 3) {
		violations.push(["U6_CHOICE_EXPECTATION_VALIDATION_SURFACE_NOT_EXACT", "internal/choice/validation.go"]);
	}
	const universeSeal = functionBody(commentlessAt(manifest, "internal/choice/validation.go"), /\bfunc\s+makeConfirmedOutcomeSetSeal\s*\(/u) ?? "";
	for (const anchor of [
		"ExpectationDomainDigest",
		'json:"expectation_domain_digest"',
		"registry.expectationDomain.Valid()",
		"identity.ExpectationDomainDigest = registry.expectationDomain.Digest().String()",
	]) requireIncludes(violations, universeSeal, anchor, "U6_EXPECTATION_DOMAIN_NOT_BOUND_TO_UNIVERSE_SEAL", anchor);
  const reviewReferences = choiceEntries.filter((entry) => /\bNewCustomExpectationReview\b/u.test(entry.lexical.code));
  const reviewCount = reviewReferences.reduce((total, entry) => total + (entry.lexical.code.match(/\bNewCustomExpectationReview\b/gu) ?? []).length, 0);
  if (reviewCount !== 2 || !exactSet(reviewReferences.map((entry) => entry.path), ["internal/choice/session.go", "internal/choice/validation.go"])) {
    violations.push(["U6_CUSTOM_REVIEW_CONSTRUCTION_SURFACE_NOT_EXACT", reviewReferences.map((entry) => entry.path).join(",")]);
  }

  const valueIdentity = functionBody(validation, /\bfunc\s*\(v\s+ExactValue\)\s+identityKey\s*\(/u) ?? "";
  for (const anchor of [
    "case ValueBytes:",
    "lengthPrefix(string(v.opaque))",
    "case ValueOrderedStringList:",
    "lengthPrefix(string(v.canonical))",
    "if mode == fieldRegistryPortable",
    "lengthPrefix(string(v.canonical))",
  ]) requireIncludes(violations, valueIdentity, anchor, "U6_PORTABLE_VALUE_IDENTITY_NOT_EXACT", anchor);
  const valueIdentitySource = functionBody(
    commentlessAt(manifest, "internal/choice/validation.go"),
    /\bfunc\s*\(v\s+ExactValue\)\s+identityKey\s*\(/u,
  ) ?? "";
  if (!/case\s+ValueCanonicalJSON\s*:\s*if\s+mode\s*==\s*fieldRegistryPortable\s*\{\s*return\s+"J"\s*\+\s*lengthPrefix\(string\(v\.canonical\)\)\s*\}\s*return\s+"J"\s*\+\s*lengthPrefix\(v\.text\)/u.test(valueIdentitySource)) {
    violations.push(["U6_PORTABLE_VALUE_IDENTITY_NOT_EXACT", "ExactValue.identityKey"]);
  }
  const blindWire = functionBody(blind, /\bfunc\s+blindField\s*\(/u) ?? "";
  for (const anchor of [
    "if value.Tag() == ValueBytes",
    "result.Text = base64.StdEncoding.EncodeToString(value.Bytes())",
    "value.Tag() == ValueCanonicalJSON || value.Tag() == ValueOrderedStringList",
    "result.CanonicalJSONBase64 = base64.StdEncoding.EncodeToString(value.CanonicalBytes())",
    "result.Text =",
  ]) requireIncludes(violations, blindWire, anchor, "U6_PORTABLE_BLIND_WIRE_NOT_COMPATIBLE", anchor);
  const tupleWire = functionBody(session, /\bfunc\s+tupleToWire\s*\(/u) ?? "";
  const exactFromWire = functionBody(session, /\bfunc\s+exactValueFromWire\s*\(/u) ?? "";
  for (const anchor of ["ValueBytes", "wire.Text = base64.StdEncoding.EncodeToString(value.Bytes())", "ValueOrderedStringList", "wire.CanonicalJSONBase64 = base64.StdEncoding.EncodeToString(value.CanonicalBytes())"]) {
    requireIncludes(violations, tupleWire, anchor, "U6_PORTABLE_DECISION_WIRE_NOT_COMPATIBLE", anchor);
  }
  for (const anchor of ["case ValueBytes:", "base64.StdEncoding.Strict().DecodeString(wire.Text)", "case ValueOrderedStringList:", "base64.StdEncoding.Strict().DecodeString(wire.CanonicalJSONBase64)"]) {
    requireIncludes(violations, exactFromWire, anchor, "U6_PORTABLE_DECISION_WIRE_PARSE_NOT_STRICT", anchor);
  }

	const decision = codeAt(manifest, "internal/choice/decision.go");
	const propose = functionBody(session, /\bfunc\s*\(s\s+Session\)\s+Propose\s*\(/u) ?? "";
	const revise = functionBody(session, /\bfunc\s*\(s\s+Session\)\s+Revise\s*\(/u) ?? "";
	for (const [body, state, label] of [
		[propose, "result.state = SessionProvisionalRecorded", "Session.Propose"],
		[revise, "result.state = SessionPostRevealRecorded", "Session.Revise"],
	]) {
		const stateIndex = body.indexOf(state);
		const preflightIndex = body.indexOf("if err := preflightPortableDecisionBudget(result); err != nil");
		if (stateIndex < 0 || preflightIndex < 0 || stateIndex >= preflightIndex ||
			(body.match(/\bpreflightPortableDecisionBudget\s*\(/gu) ?? []).length !== 1) {
			violations.push(["U6_PORTABLE_DECISION_DRAFT_PREFLIGHT_NOT_EXACT", label]);
		}
	}
	const preflightReferences = session.match(/\bpreflightPortableDecisionBudget\s*\(/gu) ?? [];
	if (preflightReferences.length !== 3) {
		violations.push(["U6_PORTABLE_DECISION_DRAFT_PREFLIGHT_SURFACE", String(preflightReferences.length)]);
	}
	const preflight = functionBody(session, /\bfunc\s+preflightPortableDecisionBudget\s*\(/u) ?? "";
	for (const anchor of [
		"source.record.mode != choicepointPortable",
		"prospective := source.clone()",
		"prospective.state = SessionPostRevealRecorded",
		"prospective.revealed = true",
		"buildDecisionRecord(",
		"false,",
		"len(decision.canonicalBytes) > maxPortableDecisionBaseBytes",
	]) requireIncludes(violations, preflight, anchor, "U6_PORTABLE_DECISION_PREFLIGHT_BUILDER_NOT_EXACT", anchor);
	const builder = functionBody(commentlessAt(manifest, "internal/choice/decision.go"), /\bfunc\s+buildDecisionRecord\s*\(/u) ?? "";
	for (const anchor of [
		"maxPortableDecisionLateBytes = 256 * 1024",
		"maxPortableDecisionBaseBytes = canon.MaxInputBytes - maxPortableDecisionLateBytes",
	]) requireIncludes(violations, decision, anchor, "U6_PORTABLE_DECISION_BUDGET_CONSTANTS_NOT_EXACT", anchor);
	requireIncludes(
		violations,
		choicepoint,
		"maxChoicepointNestedRawBytes = 600 * 1024",
		"U6_CHOICEPOINT_RESOURCE_PROFILE_NOT_EXACT",
		"maxChoicepointNestedRawBytes",
	);
	for (const anchor of [
		"if enforcePortableBudget && session.record.mode == choicepointPortable",
		'buildDecisionRecord(session, "x", "", []domain.ReceiptReference{}, nil, false)',
		"len(base.canonicalBytes) > maxPortableDecisionBaseBytes",
		"lateBytes := len(canonicalBytes) - len(base.canonicalBytes)",
		"lateBytes < 0 || lateBytes > maxPortableDecisionLateBytes",
	]) requireIncludes(violations, builder, anchor, "U6_PORTABLE_DECISION_FINAL_BUDGET_NOT_EXACT", anchor);
	const budgetTests = commentlessAt(manifest, "internal/choice/session_roundtrip_test.go");
	for (const anchor of [
		"func TestPortableDraftBudgetNeverDefersStructuralOverflowToFinalize",
		"proposalRefusals++",
		"reviseRefused = true",
		"multiplicityProved = true",
		"passed draft preflight but failed minimal finalization",
		"ParseDecisionRecord(decision.CanonicalBytes(), record)",
	]) requireIncludes(violations, budgetTests, anchor, "U6_PORTABLE_DECISION_BUDGET_TEST_NOT_EXACT", anchor);
	for (const anchor of [
		"func TestPortableDecisionLateBudgetIsExactAndLegacyHistoryKeepsFullCeiling",
		"maxPortableDecisionLateBytes - fixedReceiptGrowth",
		"gradeBytes+1",
		"func TestDecisionReceiptAdmissionIsBoundedBeforeSorting",
		"canon.MaxContainerMembers+1",
		"canon.MaxInputBytes+1",
	]) requireIncludes(violations, budgetTests, anchor, "U6_PORTABLE_DECISION_LIMIT_TEST_NOT_EXACT", anchor);
	const normalizeReceipts = functionBody(decision, /\bfunc\s+normalizeDecisionReceipts\s*\(/u) ?? "";
	const normalizeReceiptsExact = functionBody(commentlessAt(manifest, "internal/choice/decision.go"), /\bfunc\s+normalizeDecisionReceipts\s*\(/u) ?? "";
	const countGuard = normalizeReceipts.indexOf("len(receipts) > canon.MaxContainerMembers");
	const rawGuard = normalizeReceipts.indexOf("consumeReceiptWireBudget(wire, &remaining)");
	const receiptAllocation = normalizeReceipts.indexOf("keyed := make([]keyedReceipt, len(receipts))");
	const receiptSort = normalizeReceipts.indexOf("sort.Slice(keyed");
	const receiptPrefix = rawGuard < 0 ? "" : normalizeReceipts.slice(0, rawGuard);
	const exactDecisionBudget = normalizeReceiptsExact.indexOf("consumeReceiptWireBudget(wire, &remaining)");
	const exactDecisionPrefix = exactDecisionBudget < 0 ? "" : compactCode(normalizeReceiptsExact.slice(0, exactDecisionBudget));
	const expectedDecisionPrefix = 'if len(receipts) > canon.MaxContainerMembers { return nil, refusal(CodeInputLimitExceeded, "DecisionRecord receipt count exceeds the canonical container profile") } remaining := canon.MaxInputBytes for _, receipt := range receipts { wire := receipt.Wire() if !';
	if (countGuard < 0 || rawGuard < 0 || receiptAllocation < 0 || receiptSort < 0 ||
		countGuard >= receiptAllocation || rawGuard >= receiptAllocation || receiptAllocation >= receiptSort) {
		violations.push(["U6_DECISION_RECEIPT_ADMISSION_NOT_BOUNDED", "normalizeDecisionReceipts"]);
	}
	if (/\b(?:make|append|copy)\s*\(|\bsort\s*\./u.test(receiptPrefix) ||
		!normalizeReceipts.includes("remaining := canon.MaxInputBytes\n") || exactDecisionPrefix !== expectedDecisionPrefix) {
		violations.push(["U6_DECISION_RECEIPT_ADMISSION_NOT_BOUNDED", "normalizeDecisionReceipts prefix"]);
	}
	const normalizeChoicepoint = functionBody(choicepoint, /\bfunc\s+normalizeChoicepointReceipts\s*\(/u) ?? "";
	const normalizeChoicepointExact = functionBody(commentlessAt(manifest, "internal/choice/choicepoint.go"), /\bfunc\s+normalizeChoicepointReceipts\s*\(/u) ?? "";
	const choiceReceiptCount = normalizeChoicepoint.indexOf("len(receipts) > canon.MaxContainerMembers");
	const choiceReceiptBudget = normalizeChoicepoint.indexOf("consumeReceiptWireBudget(wire, &remaining)");
	const choiceReceiptAllocation = normalizeChoicepoint.indexOf("keyed := make([]keyedReceipt, len(receipts))");
	const choiceReceiptSort = normalizeChoicepoint.indexOf("sort.Slice(keyed");
	const choiceReceiptPrefix = choiceReceiptBudget < 0 ? "" : normalizeChoicepoint.slice(0, choiceReceiptBudget);
	const exactChoiceBudget = normalizeChoicepointExact.indexOf("consumeReceiptWireBudget(wire, &remaining)");
	const exactChoicePrefix = exactChoiceBudget < 0 ? "" : compactCode(normalizeChoicepointExact.slice(0, exactChoiceBudget));
	const expectedChoicePrefix = 'if len(receipts) > canon.MaxContainerMembers { return nil, refusal(CodeInputLimitExceeded, "choicepoint receipt count exceeds the canonical container profile") } remaining := canon.MaxInputBytes for _, receipt := range receipts { wire := receipt.Wire() if !';
	if (choiceReceiptCount < 0 || choiceReceiptBudget < 0 || choiceReceiptAllocation < 0 || choiceReceiptSort < 0 ||
		choiceReceiptCount >= choiceReceiptAllocation || choiceReceiptBudget >= choiceReceiptAllocation || choiceReceiptAllocation >= choiceReceiptSort) {
		violations.push(["U6_CHOICEPOINT_RECEIPT_ADMISSION_NOT_BOUNDED", "normalizeChoicepointReceipts"]);
	}
	if (/\b(?:make|append|copy)\s*\(|\bsort\s*\./u.test(choiceReceiptPrefix) ||
		!normalizeChoicepoint.includes("remaining := canon.MaxInputBytes\n") || exactChoicePrefix !== expectedChoicePrefix) {
		violations.push(["U6_CHOICEPOINT_RECEIPT_ADMISSION_NOT_BOUNDED", "normalizeChoicepointReceipts prefix"]);
	}
	const normalizeAliases = functionBody(blind, /\bfunc\s*\(v\s+BlindView\)\s+normalizeAliases\s*\(/u) ?? "";
	const normalizeAliasesExact = functionBody(commentlessAt(manifest, "internal/choice/blind.go"), /\bfunc\s*\(v\s+BlindView\)\s+normalizeAliases\s*\(/u) ?? "";
	const aliasAdmission = normalizeAliases.indexOf("v.admitAliases(raw)");
	const aliasAllocation = normalizeAliases.indexOf("aliases := make([]string, len(raw))");
	const aliasSort = normalizeAliases.indexOf("sort.Strings(aliases)");
	const aliasResolve = normalizeAliases.indexOf("v.resolveAliases(aliases)");
	const aliasPrefix = aliasAdmission < 0 ? "" : normalizeAliases.slice(0, aliasAdmission);
	const aliasAdmissionExact = normalizeAliasesExact.indexOf("v.admitAliases(raw)");
	const aliasExactPrefix = aliasAdmissionExact < 0 ? "" : compactCode(normalizeAliasesExact.slice(0, aliasAdmissionExact));
	if (aliasAdmission < 0 || aliasAllocation < 0 || aliasSort < 0 || aliasResolve < 0 ||
		aliasAdmission >= aliasAllocation || aliasAllocation >= aliasSort || aliasSort >= aliasResolve) {
		violations.push(["U6_ALIAS_ADMISSION_NOT_BEFORE_COPY_SORT_RESOLVE", "BlindView.normalizeAliases"]);
	}
	if (/\b(?:make|append|copy)\s*\(|\bsort\s*\./u.test(aliasPrefix) || aliasExactPrefix !== "if err :=") {
		violations.push(["U6_ALIAS_ADMISSION_NOT_BEFORE_COPY_SORT_RESOLVE", "BlindView.normalizeAliases prefix"]);
	}
	const admitAliases = functionBody(blind, /\bfunc\s*\(v\s+BlindView\)\s+admitAliases\s*\(/u) ?? "";
	requireIncludes(
		violations,
		commentlessAt(manifest, "internal/choice/blind.go"),
		'const maxBlindAliasBytes = len("blind:") + 64',
		"U6_ALIAS_ADMISSION_PROFILE_NOT_EXACT",
		"maxBlindAliasBytes",
	);
	for (const anchor of [
		"raw == nil",
		"len(raw) > len(v.aliases)",
		"remaining := maxBlindAliasBytes * len(v.aliases)",
		"len(alias) > maxBlindAliasBytes || len(alias) > remaining",
		"remaining -= len(alias)",
	]) requireIncludes(violations, admitAliases, anchor, "U6_ALIAS_ADMISSION_PROFILE_NOT_EXACT", anchor);
	const receiptBudget = functionBody(choicepoint, /\bfunc\s+consumeReceiptWireBudget\s*\(/u) ?? "";
	for (const anchor of [
		"remaining == nil || *remaining < 0",
		"[...]string{wire.Authority, wire.GradeVerbatim, wire.CommitOID, wire.CommandDigest}",
		"len(member) > *remaining",
		"*remaining -= len(member)",
	]) requireIncludes(violations, receiptBudget, anchor, "U6_RECEIPT_WIRE_BUDGET_NOT_EXACT", anchor);
	const draft = functionBody(session, /\bfunc\s+newRulingDraft\s*\(/u) ?? "";
	const draftExact = functionBody(commentlessAt(manifest, "internal/choice/session.go"), /\bfunc\s+newRulingDraft\s*\(/u) ?? "";
	const draftAliasAdmission = draft.indexOf("view.admitAliases(input.AllowedAliases)");
	const draftSelectedNormalization = draft.indexOf("record.confirmed.registry.resolveSelected(input.SelectedFields)");
	const draftAliasNormalization = draft.indexOf("view.normalizeAliases(input.AllowedAliases)");
	const draftAliasAdmissionExact = draftExact.indexOf("view.admitAliases(input.AllowedAliases)");
	const draftExactPrefix = draftAliasAdmissionExact < 0 ? "" : compactCode(draftExact.slice(0, draftAliasAdmissionExact));
	const expectedDraftPrefix = 'if input.SelectedFields == nil || input.AllowedAliases == nil { return rulingDraft{}, refusal(CodeOmittedSelectedFields, "ruling draft selections must be explicit arrays") } if err :=';
	if (draftAliasAdmission < 0 || draftSelectedNormalization < 0 || draftAliasNormalization < 0 ||
		draftAliasAdmission >= draftSelectedNormalization || draftAliasAdmission >= draftAliasNormalization || draftExactPrefix !== expectedDraftPrefix) {
		violations.push(["U6_DRAFT_ALIAS_ADMISSION_ORDER_NOT_EXACT", "newRulingDraft"]);
	}
	const confirmed = codeAt(manifest, "internal/projectiontranslate/confirmed.go");
	const translateConfirmed = functionBody(confirmed, /\bfunc\s+TranslateConfirmed\s*\(/u) ?? "";
	const translateConfirmedExact = functionBody(commentlessAt(manifest, "internal/projectiontranslate/confirmed.go"), /\bfunc\s+TranslateConfirmed\s*\(/u) ?? "";
	const proofMembership = translateConfirmed.indexOf("if _, member := expected[candidateKey]; !member");
	const proofBudget = translateConfirmed.indexOf("len(proof.CanonicalProjection) > maxConfirmedProjectionBytes-projectionBytes");
	const proofCopy = translateConfirmed.indexOf("frozenProjection := append([]byte(nil), proof.CanonicalProjection...)");
	const proofLoop = translateConfirmed.indexOf("for index, proof := range proofs");
	const proofPrefix = proofLoop < 0 || proofBudget < 0 ? "" : translateConfirmed.slice(proofLoop, proofBudget);
	const proofLoopExact = translateConfirmedExact.indexOf("for index, proof := range proofs");
	const proofBudgetExact = translateConfirmedExact.indexOf("len(proof.CanonicalProjection) > maxConfirmedProjectionBytes-projectionBytes");
	const proofExactPrefix = proofLoopExact < 0 || proofBudgetExact < 0 ? "" : compactCode(translateConfirmedExact.slice(proofLoopExact, proofBudgetExact));
	const expectedProofPrefix = 'for index, proof := range proofs { candidateKey := proof.CandidateExecutionKey.String() if !proof.CandidateExecutionKey.Valid() { return ConfirmedTranslations{}, refuse(CodeRosterMismatch, "", "projection proof has an invalid candidate key") } if _, duplicate := seen[candidateKey]; duplicate { return ConfirmedTranslations{}, refuse(CodeRosterMismatch, "", "projection proof candidate occurs more than once") } if _, member := expected[candidateKey]; !member { return ConfirmedTranslations{}, refuse(CodeRosterMismatch, "", "projection proof candidate is not in the confirmed roster") } if';
	if (proofMembership < 0 || proofBudget < 0 || proofCopy < 0 || proofMembership >= proofBudget || proofBudget >= proofCopy) {
		violations.push(["U6_PROJECTION_PROOF_ADMISSION_NOT_BEFORE_COPY", "projectiontranslate.TranslateConfirmed"]);
	}
	if (/\b(?:make|append|copy)\s*\(/u.test(proofPrefix) || proofExactPrefix !== expectedProofPrefix) {
		violations.push(["U6_PROJECTION_PROOF_ADMISSION_NOT_BEFORE_COPY", "projectiontranslate.TranslateConfirmed proof-loop prefix"]);
	}
	for (const anchor of [
		"maxConfirmedProjectionBytes = 600 * 1024",
		"maxConfirmedTupleBytes      = 512 * 1024",
	]) requireIncludes(violations, confirmed, anchor, "U6_CONFIRMED_TRANSLATION_RESOURCE_PROFILE_NOT_EXACT", anchor);
	const confirmedBodies = topLevelStructBodies(confirmed, "ConfirmedTranslations");
	const expectedConfirmedBody = "resolved Resolved expectation ExpectationDomain outcomes []TranslatedOutcome outcomeMapDigest compare.OutcomeArtifactDigest preservationDigest compare.PreservationMapDigest seal *confirmedSeal";
	if (confirmedBodies.length !== 1 || exportedStructFields(confirmedBodies[0]).length !== 0 || compactCode(confirmedBodies[0]) !== expectedConfirmedBody ||
		!translateConfirmed.includes("resolved: resolved, expectation: expectation")) {
		violations.push(["U6_CONFIRMED_EXPECTATION_AUTHORITY_NOT_EXACT", "projectiontranslate.ConfirmedTranslations"]);
	}
	const confirmedValid = functionBody(confirmed, /\bfunc\s*\(c\s+ConfirmedTranslations\)\s+Valid\s*\(/u) ?? "";
	for (const anchor of [
		"c.expectation.Valid()",
		"c.expectation.ProfileDigest() != c.resolved.profile.Digest()",
		"projectionBytes <= maxConfirmedProjectionBytes",
		"tupleBytes <= maxConfirmedTupleBytes",
	]) requireIncludes(violations, confirmedValid, anchor, "U6_CONFIRMED_EXPECTATION_AUTHORITY_NOT_EXACT", anchor);
	const outcomeMap = codeAt(manifest, "internal/compare/outcome_map.go");
	const rosterValid = functionBody(outcomeMap, /\bfunc\s*\(r\s+ConfirmedProjectionRoster\)\s+Valid\s*\(/u) ?? "";
	for (const anchor of [
		"len(r.entries) < 2 || len(r.entries) > 4",
		"seen := make(map[string]struct{}, len(r.entries))",
		"entry.candidate.Valid()",
		"entry.fingerprint.Valid()",
		"if _, duplicate := seen[key]; duplicate",
	]) requireIncludes(violations, rosterValid, anchor, "U6_CONFIRMED_ROSTER_RESOURCE_PROFILE_NOT_EXACT", anchor);
	for (const anchor of [
		"func TestChoicepointReceiptAdmissionIsBoundedBeforeSorting",
		"func TestRulingAliasAdmissionIsBoundedBeforeNormalization",
		"func TestDecisionAnnotationV1RetainsExactInertBytes",
	]) requireIncludes(violations, budgetTests, anchor, "U6_RESOURCE_AND_INERTNESS_TEST_NOT_EXACT", anchor);
	const portableChoiceTests = commentlessAt(manifest, "internal/choice/portable_choice_test.go");
	requireIncludes(
		violations,
		portableChoiceTests,
		"func TestTranslateConfirmedAdmitsProofBytesBeforeDefensiveCopy",
		"U6_PROJECTION_PROOF_ADMISSION_TEST_NOT_EXACT",
		"internal/choice/portable_choice_test.go",
	);

  const inspection = functionBody(portable, /\bfunc\s+InspectPortableRuling\s*\(/u) ?? "";
  const legacyGuard = inspection.indexOf("decision.choicepoint.mode == choicepointLegacyWhole");
  const portableGuard = inspection.indexOf("decision.choicepoint.mode != choicepointPortable");
  if (legacyGuard < 0 || portableGuard < 0 || legacyGuard >= portableGuard || !inspection.includes("CodeLegacyWholeProjectionNotPortable")) {
    violations.push(["U6_LEGACY_PORTABLE_PREPARATION_GUARD_NOT_EXACT", "InspectPortableRuling"]);
  }
  const legacyPortableLiterals = choiceEntries.flatMap((entry) => entry.lexical.literals)
    .filter((literal) => literal === "LEGACY_WHOLE_PROJECTION_NOT_PORTABLE");
  if (legacyPortableLiterals.length !== 1) {
    violations.push(["U6_LEGACY_PORTABLE_REFUSAL_LITERAL_NOT_EXACT", "internal/choice"]);
  }
  const inspectionBodies = topLevelStructBodies(portable, "PortableRulingInspection");
  const expectedInspectionBody = "decisionDigest domain.Digest profileDigest domain.Digest selectedFields []string seal *portableInspectionSeal";
  if (inspectionBodies.length !== 1 || exportedStructFields(inspectionBodies[0]).length !== 0 || compactCode(inspectionBodies[0]) !== expectedInspectionBody ||
      (portable.match(/\bPortableRulingInspection\s*\{\s*decisionDigest\s*:/gu) ?? []).length !== 1) {
    violations.push(["U6_PORTABLE_INSPECTION_SURFACE_NOT_CLOSED", "PortableRulingInspection"]);
  }
  const inspectionReferences = productionEntries(manifest).filter((entry) => /\bInspectPortableRuling\b/u.test(entry.lexical.code));
  const inspectionCount = inspectionReferences.reduce((total, entry) => total + (entry.lexical.code.match(/\bInspectPortableRuling\b/gu) ?? []).length, 0);
	if (inspectionCount !== 5 || !exactSet(inspectionReferences.map((entry) => entry.path), ["internal/choice/portable.go", "internal/choice/promotion/service.go"])) {
    violations.push(["U6_PORTABLE_INSPECTION_REFERENCE_SURFACE_NOT_EXACT", inspectionReferences.map((entry) => entry.path).join(",")]);
  }
  const preparationBodies = topLevelStructBodies(promotion, "PortableRulingPreparation");
  const expectedPreparationBody = "ruling Ruling inspection choice.PortableRulingInspection seal *portableRulingPreparationSeal";
  if (preparationBodies.length !== 1 || exportedStructFields(preparationBodies[0]).length !== 0 || compactCode(preparationBodies[0]) !== expectedPreparationBody) {
    violations.push(["U6_PORTABLE_PREPARATION_SURFACE_NOT_CLOSED", "PortableRulingPreparation"]);
  }
  if ((promotion.match(/\bportableRulingPreparationSeal\s*\{/gu) ?? []).length !== 1) {
    violations.push(["U6_PORTABLE_PREPARATION_SEAL_ISSUANCE_NOT_EXACT", "internal/choice/promotion/service.go"]);
  }
  for (const entry of productionEntries(manifest)) {
    const packageDirectory = entry.path.slice(0, entry.path.lastIndexOf("/"));
    if (packageDirectory !== "internal/choice/promotion" || entry.path === "internal/choice/promotion/service.go") continue;
    const preparationReferences = entry.lexical.code.match(/\bPortableRulingPreparation\b/gu) ?? [];
    const sealReferences = entry.lexical.code.match(/\bportableRulingPreparationSeal\b/gu) ?? [];
    if (entry.path === "internal/choice/promotion/residue.go") {
      const studyBody = functionBody(
        entry.lexical.code,
        /\bfunc\s*\(p\s+PortableRulingPreparation\)\s+StudyID\s*\(/u,
      ) ?? "";
      const expectedStudyBody = "if !p.Valid() { return store.StudyID{} } return p.ruling.StudyID()";
      if (preparationReferences.length !== 1 || sealReferences.length !== 0 || compactCode(studyBody) !== expectedStudyBody) {
        violations.push(["U6_PORTABLE_PREPARATION_CONSTRUCTION_OUTSIDE_OWNER", entry.path]);
      }
    } else if (preparationReferences.length !== 0 || sealReferences.length !== 0) {
      violations.push(["U6_PORTABLE_PREPARATION_CONSTRUCTION_OUTSIDE_OWNER", entry.path]);
    }
  }
  for (const [signature, name] of [
    [/\bfunc\s+PreparePortableRuling\s*\(/u, "PreparePortableRuling"],
    [/\bfunc\s+ValidatePortableRulingPreparation\s*\(/u, "ValidatePortableRulingPreparation"],
  ]) {
    const body = functionBody(promotion, signature) ?? "";
    const current = body.indexOf("validateRuling(");
    const semantic = body.indexOf("choice.InspectPortableRuling(");
    if (current < 0 || semantic < 0 || current >= semantic) {
      violations.push(["U6_PORTABLE_PREPARATION_CURRENT_HEAD_ORDER", name]);
    }
  }
  requireIncludes(
    violations,
    functionBody(promotion, /\bfunc\s+PreparePortableRuling\s*\(/u) ?? "",
    "if err := validateRuling(ctx, objectStore, ruling); err != nil",
    "U6_PORTABLE_PREPARATION_CURRENT_HEAD_GUARD_NOT_EXACT",
    "PreparePortableRuling",
  );
  requireIncludes(
    violations,
    functionBody(promotion, /\bfunc\s+ValidatePortableRulingPreparation\s*\(/u) ?? "",
    "if err := validateRuling(ctx, objectStore, preparation.ruling); err != nil",
    "U6_PORTABLE_PREPARATION_CURRENT_HEAD_GUARD_NOT_EXACT",
    "ValidatePortableRulingPreparation",
  );
  const validateCurrent = functionBody(promotion, /\bfunc\s+validateRuling\s*\(/u) ?? "";
  for (const anchor of ["objectStore.OpenHead(ctx, ruling.head.StudyID())", "sameHead(current, ruling.head)", "current.Stage() != store.StageRuling", "current.CurrentDigest() != ruling.record.Digest()"] ) {
    requireIncludes(violations, validateCurrent, anchor, "U6_PORTABLE_PREPARATION_CURRENT_HEAD_BINDING", anchor);
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
    "func (s *ObjectStore) AdvanceResidue(",
    "func (s *ObjectStore) advanceHead(",
    "authority confirmationauthority.Publication",
    "authority choicepromotionauthority.Choicepoint",
    "authority choicepromotionauthority.Ruling",
    "authority nodeauthority.Publication",
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
    ["AdvanceResidue", "StageResidue"],
  ]);
  for (const [method, stage] of typedTransitions) {
    const body = functionBody(head, new RegExp(`\\bfunc\\s*\\(s\\s+\\*ObjectStore\\)\\s+${method}\\s*\\(`, "u")) ?? "";
    const exactCall = `return s.advanceHead(ctx, expected, ${stage}, object)`;
    if ((body.match(/\badvanceHead\b/gu) ?? []).length !== 1 || !body.includes(exactCall)) {
      violations.push(["U6_TYPED_TRANSITION_RAW_CALL_NOT_EXACT", method]);
    }
  }
  // One declaration plus the seven typed transition calls above. This also
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
  for (const field of ["study.text", "revision", "stage", "currentKind", "currentDigest", "previousHead", "previousObject", "lineageRoot", "headDigest"]) {
    const escaped = field.replaceAll(".", "\\.");
    const equality = new RegExp(`\\bleft\\.${escaped}\\s*==\\s*right\\.${escaped}\\b`, "u");
    if (!equality.test(tokenBody)) violations.push(["U6_CAS_TOKEN_EXACT_EQUALITY_MISSING", field]);
  }
  if ((tokenBody.match(/==/gu) ?? []).length !== 9 || tokenBody.includes("||") || /\.Valid\s*\(/u.test(tokenBody)) {
    violations.push(["U6_CAS_TOKEN_COMPARISON_NOT_EXACT", "sameHeadToken"]);
  }
  const exactTokenBody = "return left.study.text == right.study.text && left.revision == right.revision && left.stage == right.stage && " +
    "left.currentKind == right.currentKind && left.currentDigest == right.currentDigest && left.previousHead == right.previousHead && " +
    "left.previousObject == right.previousObject && left.lineageRoot == right.lineageRoot && left.headDigest == right.headDigest";
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

  const choiceSchema = strictJSONParse(
    sourceAt(manifest, "spec/schema/v1/choicepoint.schema.json"),
    "spec/schema/v1/choicepoint.schema.json",
  ).value;
  const expectedModes = ["WHOLE_EXACT_CANONICAL_PROJECTION_V1", "ADAPTER_BOUND_PORTABLE_FIELDS_V1"];
  if (JSON.stringify(choiceSchema?.properties?.choice_projection_mode?.enum) !== JSON.stringify(expectedModes)) {
    violations.push(["U6_CHOICEPOINT_MODE_SCHEMA_NOT_EXACT", "spec/schema/v1/choicepoint.schema.json"]);
  }

  const decisionSchema = strictJSONParse(
    sourceAt(manifest, "spec/schema/v1/decision-record.schema.json"),
    "spec/schema/v1/decision-record.schema.json",
  ).value;
  const exactValue = decisionSchema?.$defs?.ExactValue;
  const expectedTags = ["MISSING", "STRING", "INTEGER", "BOOLEAN", "NULL", "BYTES", "ORDERED_STRING_LIST", "CANONICAL_JSON"];
  if (JSON.stringify(exactValue?.properties?.tag?.enum) !== JSON.stringify(expectedTags)) {
    violations.push(["U6_EXACT_VALUE_TAG_SCHEMA_NOT_EXACT", "spec/schema/v1/decision-record.schema.json"]);
  }
  const expectedExactValueMembers = ["tag", "text", "boolean", "canonical_json_base64"];
  if (exactValue?.additionalProperties !== false ||
      JSON.stringify(exactValue?.required) !== JSON.stringify(expectedExactValueMembers) ||
      !exactSet(Object.keys(exactValue?.properties ?? {}), expectedExactValueMembers)) {
    violations.push(["U6_EXACT_VALUE_SCHEMA_SURFACE_NOT_CLOSED", "spec/schema/v1/decision-record.schema.json"]);
  }
  const conditionalFor = (tag) => (exactValue?.allOf ?? []).filter((entry) => entry?.if?.properties?.tag?.const === tag);
  const expectedBytesSlots = {
    text: { $ref: "#/$defs/OptionalBase64" },
    boolean: { const: false },
    canonical_json_base64: { const: "" },
  };
  const expectedListSlots = {
    text: { const: "" },
    boolean: { const: false },
    canonical_json_base64: { $ref: "#/$defs/NonemptyBase64" },
  };
  const bytesRules = conditionalFor("BYTES");
  const listRules = conditionalFor("ORDERED_STRING_LIST");
  const conditionalTags = (exactValue?.allOf ?? []).map((entry) => entry?.if?.properties?.tag?.const);
  if (!exactSet(conditionalTags, expectedTags) || expectedTags.some((tag) => conditionalFor(tag).length !== 1)) {
    violations.push(["U6_EXACT_VALUE_SCHEMA_CONDITIONALS_NOT_EXACT", "spec/schema/v1/decision-record.schema.json"]);
  }
  if (bytesRules.length !== 1 || JSON.stringify(bytesRules[0]?.then?.properties) !== JSON.stringify(expectedBytesSlots)) {
    violations.push(["U6_BYTES_COMPATIBILITY_SCHEMA_NOT_EXACT", "spec/schema/v1/decision-record.schema.json"]);
  }
  if (listRules.length !== 1 || JSON.stringify(listRules[0]?.then?.properties) !== JSON.stringify(expectedListSlots)) {
    violations.push(["U6_ORDERED_LIST_COMPATIBILITY_SCHEMA_NOT_EXACT", "spec/schema/v1/decision-record.schema.json"]);
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
  inspectEligibilityCore(manifest, violations);
  inspectPortableAuthorityBoundaries(manifest, violations);
  inspectPortableRulingBoundaries(manifest, violations);
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
