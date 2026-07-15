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

// These are package roots, not permissive prefixes. Every filesystem member
// below them participates in the manifest. A new source file is reviewed
// automatically; a symlink, special file, or non-Go artifact closes the gate.
const reviewedRoots = Object.freeze([
  Object.freeze({ path: "internal/reduce", packageName: "reduce" }),
  Object.freeze({ path: "internal/store", packageName: "store" }),
  Object.freeze({ path: "internal/reduction", packageName: "reduction" }),
]);

const forbiddenReduceImports = Object.freeze([
  "internal/gitobj",
  "internal/world",
  "internal/adapters",
  "internal/store",
  "internal/server",
  "internal/reduction",
]);

const manifestToolSource = String.raw`
const { createHash } = require("node:crypto");
const { closeSync, constants, fstatSync, lstatSync, openSync, readFileSync } = require("node:fs");
const { isAbsolute, relative, resolve, sep } = require("node:path");

function slash(value) { return value.split(sep).join("/"); }
function fail(code, detail) { process.stderr.write(code + ": " + detail + "\n"); process.exit(1); }
const payload = JSON.parse(readFileSync(0, "utf8"));
if (payload.forceFailure === true) fail("U5_MANIFEST_TOOL_FORCED_FAILURE", "self-test fail-closed probe");
const digest = createHash("sha256");
for (const entry of payload.entries) {
  const absolute = resolve(payload.root, entry.path);
  const fromRoot = relative(payload.root, absolute);
  if (isAbsolute(fromRoot) || fromRoot === ".." || fromRoot.startsWith(".." + sep)) {
    fail("U5_MANIFEST_TOOL_PATH_ESCAPE", entry.path);
  }
  const before = lstatSync(absolute);
  if (before.isSymbolicLink() || !before.isFile()) fail("U5_MANIFEST_TOOL_NONREGULAR", entry.path);
  let descriptor;
  try {
    descriptor = openSync(absolute, constants.O_RDONLY | (constants.O_NOFOLLOW || 0));
    const opened = fstatSync(descriptor);
    if (!opened.isFile() || opened.dev !== before.dev || opened.ino !== before.ino || opened.size !== before.size) {
      fail("U5_MANIFEST_TOOL_FILE_CHANGED", entry.path);
    }
    const bytes = readFileSync(descriptor);
    const after = fstatSync(descriptor);
    if (after.dev !== opened.dev || after.ino !== opened.ino || after.size !== opened.size) {
      fail("U5_MANIFEST_TOOL_FILE_CHANGED", entry.path);
    }
    const sha256 = createHash("sha256").update(bytes).digest("hex");
    if (bytes.length !== entry.bytes || sha256 !== entry.sha256) {
      fail("U5_MANIFEST_TOOL_DIGEST_MISMATCH", entry.path);
    }
    digest.update(slash(entry.path));
    digest.update("\0");
    digest.update(bytes);
    digest.update("\0");
  } finally {
    if (descriptor !== undefined) closeSync(descriptor);
  }
}
const actual = digest.digest("hex");
if (actual !== payload.digest) fail("U5_MANIFEST_TOOL_AGGREGATE_MISMATCH", actual + " != " + payload.digest);
process.stdout.write(actual + "\n");
`;

class ArchitectureError extends Error {
  constructor(code, detail) {
    super(`${code}: ${detail}`);
    this.code = code;
  }
}

function slashPath(value) {
  return value.split(sep).join("/");
}

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
      throw new ArchitectureError("U5_MANIFEST_FILE_CHANGED", relativePath);
    }
    const bytes = await handle.readFile();
    const after = await handle.stat();
    if (!sameStat(opened, after)) {
      throw new ArchitectureError("U5_MANIFEST_FILE_CHANGED", relativePath);
    }
    return bytes;
  } catch (error) {
    if (error instanceof ArchitectureError) throw error;
    throw new ArchitectureError("U5_MANIFEST_READ_FAILED", `${relativePath}: ${error.code ?? error.message}`);
  } finally {
    await handle?.close();
  }
}

function lexicalViews(source, relativePath) {
  const code = [...source];
  const commentless = [...source];
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
  if (state !== "code") {
    throw new ArchitectureError("U5_GO_LEXICAL_INVALID", `${relativePath}: unterminated ${state}`);
  }
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

function packageDeclaration(code) {
  return code.match(/^\s*package\s+([A-Za-z_][A-Za-z0-9_]*)\b/mu)?.[1] ?? "";
}

async function enumerateRoot(contract) {
  const absoluteRoot = resolve(root, contract.path);
  let rootStat;
  try {
    rootStat = await lstat(absoluteRoot);
  } catch (error) {
    throw new ArchitectureError("U5_MANIFEST_ROOT_MISSING", `${contract.path}: ${error.code ?? error.message}`);
  }
  if (rootStat.isSymbolicLink()) throw new ArchitectureError("U5_MANIFEST_SYMLINK", contract.path);
  if (!rootStat.isDirectory()) throw new ArchitectureError("U5_MANIFEST_ROOT_NONDIRECTORY", contract.path);

  const files = [];
  async function visit(directory) {
    let entries;
    try {
      entries = await readdir(directory, { withFileTypes: true });
    } catch (error) {
      throw new ArchitectureError("U5_MANIFEST_ENUMERATION_FAILED", `${slashPath(relative(root, directory))}: ${error.code ?? error.message}`);
    }
    entries.sort((left, right) => left.name.localeCompare(right.name, "en"));
    for (const entry of entries) {
      const absolute = join(directory, entry.name);
      const relativePath = slashPath(relative(root, absolute));
      const fromRoot = relative(root, absolute);
      if (fromRoot === ".." || fromRoot.startsWith(`..${sep}`)) {
        throw new ArchitectureError("U5_MANIFEST_PATH_ESCAPE", relativePath);
      }
      let metadata;
      try {
        metadata = await lstat(absolute);
      } catch (error) {
        throw new ArchitectureError("U5_MANIFEST_LSTAT_FAILED", `${relativePath}: ${error.code ?? error.message}`);
      }
      if (metadata.isSymbolicLink()) throw new ArchitectureError("U5_MANIFEST_SYMLINK", relativePath);
      if (metadata.isDirectory()) {
        await visit(absolute);
      } else if (!metadata.isFile()) {
        throw new ArchitectureError("U5_MANIFEST_NONREGULAR", relativePath);
      } else if (extname(entry.name) !== ".go") {
        throw new ArchitectureError("U5_MANIFEST_UNKNOWN_EXTENSION", relativePath);
      } else {
        const bytes = await readRegularNoFollow(absolute, relativePath, metadata);
        const source = bytes.toString("utf8");
        if (!Buffer.from(source, "utf8").equals(bytes)) {
          throw new ArchitectureError("U5_MANIFEST_INVALID_UTF8", relativePath);
        }
        const lexical = lexicalViews(source, relativePath);
        const declared = packageDeclaration(lexical.code);
        const permitted = relativePath.endsWith("_test.go")
          ? new Set([contract.packageName, `${contract.packageName}_test`])
          : new Set([contract.packageName]);
        if (!permitted.has(declared)) {
          throw new ArchitectureError(
            "U5_MANIFEST_PACKAGE_MISMATCH",
            `${relativePath}: package ${declared || "<missing>"}, want ${[...permitted].join(" or ")}`,
          );
        }
        files.push({
          path: relativePath,
          bytes: bytes.length,
          sha256: createHash("sha256").update(bytes).digest("hex"),
          source,
          lexical,
          packageRoot: contract.path,
          production: !relativePath.endsWith("_test.go"),
        });
      }
    }
  }
  await visit(absoluteRoot);
  if (!files.some((file) => file.production)) {
    throw new ArchitectureError("U5_MANIFEST_PRODUCTION_PACKAGE_EMPTY", contract.path);
  }
  return files;
}

async function readManifest() {
  const files = [];
  for (const contract of reviewedRoots) files.push(...await enumerateRoot(contract));
  files.sort((left, right) => left.path.localeCompare(right.path, "en"));
  if (new Set(files.map((file) => file.path)).size !== files.length) {
    throw new ArchitectureError("U5_MANIFEST_DUPLICATE_PATH", "reviewed roots overlap");
  }
  const digest = createHash("sha256");
  for (const file of files) {
    digest.update(file.path);
    digest.update("\0");
    digest.update(Buffer.from(file.source, "utf8"));
    digest.update("\0");
  }
  return { files, digest: digest.digest("hex") };
}

function verifyManifestWithTool(manifest) {
  const payload = {
    root,
    digest: manifest.digest,
    forceFailure: process.env.COUNTERSHAPE_U5_ARCHITECTURE_FORCE_TOOL_FAILURE === "1",
    entries: manifest.files.map(({ path, bytes, sha256 }) => ({ path, bytes, sha256 })),
  };
  const result = spawnSync(process.execPath, ["-e", manifestToolSource], {
    cwd: root,
    encoding: "utf8",
    input: JSON.stringify(payload),
    env: { NO_COLOR: "1", TZ: "UTC" },
    timeout: 30_000,
    maxBuffer: 256 * 1024,
  });
  if (result.error || result.signal || result.status !== 0 || result.stdout.trim() !== manifest.digest) {
    const output = `${result.stdout ?? ""}${result.stderr ?? ""}`.trim();
    throw new ArchitectureError(
      "U5_MANIFEST_TOOL_FAILED",
      result.error?.message ?? result.signal ?? output ?? `status ${result.status}`,
    );
  }
}

function importMatches(importPath, relativePackage) {
  const exact = `${modulePrefix}${relativePackage}`;
  return importPath === exact || importPath.startsWith(`${exact}/`);
}

function topLevelStructBodies(code, typeName) {
  const escaped = typeName.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
  const pattern = new RegExp(`\\btype\\s+${escaped}\\s+struct\\s*\\{`, "gu");
  const bodies = [];
  for (const match of code.matchAll(pattern)) {
    const openIndex = match.index + match[0].lastIndexOf("{");
    let depth = 0;
    for (let index = openIndex; index < code.length; index += 1) {
      if (code[index] === "{") depth += 1;
      if (code[index] === "}") {
        depth -= 1;
        if (depth === 0) {
          bodies.push(code.slice(openIndex + 1, index));
          break;
        }
      }
    }
  }
  return bodies;
}

function topLevelStatements(body) {
  const statements = [];
  let start = 0;
  let braces = 0;
  let brackets = 0;
  let parentheses = 0;
  for (let index = 0; index < body.length; index += 1) {
    const character = body[index];
    if (character === "{") braces += 1;
    if (character === "}") braces -= 1;
    if (character === "[") brackets += 1;
    if (character === "]") brackets -= 1;
    if (character === "(") parentheses += 1;
    if (character === ")") parentheses -= 1;
    if ((character === "\n" || character === ";") && braces === 0 && brackets === 0 && parentheses === 0) {
      statements.push(body.slice(start, index).trim());
      start = index + 1;
    }
  }
  statements.push(body.slice(start).trim());
  return statements.filter(Boolean);
}

function exportedAuthorityFields(body) {
  const exported = [];
  for (const statement of topLevelStatements(body)) {
    const named = statement.match(/^([A-Za-z_][A-Za-z0-9_]*(?:\s*,\s*[A-Za-z_][A-Za-z0-9_]*)*)\s+(.+)$/u);
    if (named) {
      for (const name of named[1].split(",").map((value) => value.trim())) {
        if (/^[A-Z]/u.test(name)) exported.push(name);
      }
      continue;
    }
    const embedded = statement.match(/^\*?(?:[A-Za-z_][A-Za-z0-9_]*\.)?([A-Za-z_][A-Za-z0-9_]*)\b/u)?.[1];
    if (embedded && /^[A-Z]/u.test(embedded)) exported.push(embedded);
  }
  return exported;
}

function oneMinimalReference(lexical) {
  const codeReference = [...lexical.code.matchAll(/\b[A-Za-z_][A-Za-z0-9_]*\b/gu)]
    .some((match) => /one_?minimal_?under/iu.test(match[0]));
  const literalReference = lexical.literals.some((literal) => /one[_ -]?minimal[_ -]?under/iu.test(literal));
  return codeReference || literalReference;
}

function hasIdentifierMatching(code, pattern) {
  return [...code.matchAll(/\b[A-Za-z_][A-Za-z0-9_]*\b/gu)]
    .some((match) => pattern.test(match[0]));
}

function preservationDigestEqualities(code) {
  const tracked = new Set();
  for (const match of code.matchAll(/\b([A-Za-z_][A-Za-z0-9_]*(?:\s*,\s*[A-Za-z_][A-Za-z0-9_]*)*)\s+\*?(?:[A-Za-z_][A-Za-z0-9_]*\.)?PreservationMapDigest\b/gu)) {
    for (const name of match[1].split(",")) tracked.add(name.trim());
  }
  for (const match of code.matchAll(/\b([A-Za-z_][A-Za-z0-9_]*)\s*:?=\s*[^;\n]*\.[A-Za-z0-9_]*Preservation(?:Map)?Digest\s*\(/gu)) {
    tracked.add(match[1]);
  }
  const matches = [];
  const seen = new Set();
  function retain(match) {
    const operatorOffset = match[0].search(/==|!=/u);
    const index = match.index + operatorOffset;
    if (!seen.has(index)) {
      seen.add(index);
      matches.push({ index, snippet: match[0].replace(/\s+/gu, " ") });
    }
  }
  const digestNamedOperand = String.raw`\b(?:[A-Za-z_][A-Za-z0-9_]*\.)*(?:[A-Za-z_][A-Za-z0-9_]*)?Preservation(?:Map)?Digest(?:\s*\([^()\n;{}]*\))?(?:\s*\.String\s*\(\s*\))?`;
  const namedEquality = new RegExp(
    `(?:${digestNamedOperand})\\s*(?:==|!=)|(?:==|!=)\\s*(?:${digestNamedOperand})`,
    "gu",
  );
  for (const match of code.matchAll(namedEquality)) retain(match);
  for (const name of tracked) {
    const escaped = name.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
    const trackedOperand = `\\b${escaped}\\b(?:\\s*\\[[^\\]\\n]*\\])*(?:\\s*\\.String\\s*\\(\\s*\\))?`;
    const direct = new RegExp(`(?:${trackedOperand}\\s*(?:==|!=)|(?:==|!=)\\s*${trackedOperand})`, "gu");
    for (const match of code.matchAll(direct)) retain(match);
  }
  return matches.sort((left, right) => left.index - right.index);
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

function lastTokenIndex(source, pattern) {
  let result = -1;
  for (const match of source.matchAll(pattern)) result = match.index;
  return result;
}

function nakedPreservationDecisionEquality(code) {
  const decisionToken = /\b(?:Preserves|Changes|Unresolved|PreservationEqual|PreservationDifferent|PreservationUnresolved)\b/u;
  for (const equality of preservationDigestEqualities(code)) {
    const prefixStart = Math.max(0, equality.index - 500);
    const prefix = code.slice(prefixStart, equality.index);
    const relativeIf = lastTokenIndex(prefix, /\bif\b/gu);
    if (relativeIf >= 0) {
      const ifIndex = prefixStart + relativeIf;
      const between = code.slice(ifIndex + 2, equality.index);
      const openIndex = code.indexOf("{", equality.index + 2);
      if (!/[{};]/u.test(between) && openIndex >= 0 && openIndex - equality.index < 600) {
        const consequent = balancedBlock(code, openIndex);
        if (consequent) {
          let decisionBody = consequent.body;
          const tail = code.slice(consequent.end);
          const elseMatch = tail.match(/^\s*else\s*\{/u);
          if (elseMatch) {
            const alternate = balancedBlock(code, consequent.end + elseMatch[0].lastIndexOf("{"));
            if (alternate) decisionBody += alternate.body;
          }
          if (decisionToken.test(decisionBody)) return equality.snippet;
        }
      }
    }

    const statementStart = Math.max(
      code.lastIndexOf("\n", equality.index),
      code.lastIndexOf(";", equality.index),
      code.lastIndexOf("{", equality.index),
    ) + 1;
    const statementEndCandidates = [code.indexOf("\n", equality.index), code.indexOf(";", equality.index)]
      .filter((index) => index >= 0);
    const statementEnd = statementEndCandidates.length > 0 ? Math.min(...statementEndCandidates) : code.length;
    const statement = code.slice(statementStart, statementEnd);
    const assignment = statement.match(/\b([A-Za-z_][A-Za-z0-9_]*)\s*:?=/u)?.[1] ?? "";
    // Digest equality is legitimate for exact lineage, wire, and draft identity.
    // It becomes forbidden here only when the surrounding name says that the
    // Boolean itself is a preservation classification.
    if (assignment && /(?:^|_)(?:preserves?|changes?|decision|relation|classification)(?:_|$)/iu.test(assignment)) {
      return equality.snippet;
    }
    if (assignment && /(?:isPreserving|doesPreserve|preservationDecision|preservationRelation|preservationClassification)/iu.test(assignment)) {
      return equality.snippet;
    }
    if (/\breturn\b/u.test(statement)) {
      const functions = [...code.slice(0, equality.index).matchAll(/\bfunc\s+(?:\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\b/gu)];
      const functionName = functions.at(-1)?.[1] ?? "";
      if (/(?:assess|change|decid|classif|evaluat|preservationDecision|preservationRelation|isPreserving|doesPreserve)/iu.test(functionName)) {
        return equality.snippet;
      }
    }
  }
  return "";
}

function inspectManifest(manifest) {
  const violations = [];
  let storeImportsReduce = false;
  let reductionImportsReduce = false;
  let reductionImportsStore = false;
  let reduceAssessCalls = 0;
  let authorityDeclarations = 0;
  let reductionGradeReferences = 0;

  for (const file of manifest.files) {
    if (!file.production) continue;
    const imports = importedPackages(file.lexical.commentless);
    const importedPaths = imports.map((entry) => entry.path);
    const isReduce = file.packageRoot === "internal/reduce";
    const isStore = file.packageRoot === "internal/store";
    const isReduction = file.packageRoot === "internal/reduction";

    if (isReduce) {
      for (const forbidden of forbiddenReduceImports) {
        const imported = importedPaths.find((candidate) => importMatches(candidate, forbidden));
        if (imported) violations.push(["U5_REDUCE_FORBIDDEN_IMPORT", `${file.path} imports ${imported}`]);
      }
      for (const entry of imports.filter((candidate) => candidate.path === `${modulePrefix}internal/compare`)) {
        const alias = entry.alias || "compare";
        if (alias === "." || alias === "_") continue;
        const escaped = alias.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
        reduceAssessCalls += [...file.lexical.code.matchAll(new RegExp(`\\b${escaped}\\.AssessPreservation\\s*\\(`, "gu"))].length;
      }
    }

    if (isStore) {
      if (importedPaths.includes(`${modulePrefix}internal/reduce`)) storeImportsReduce = true;
      for (const forbidden of ["internal/adapters", "internal/world"]) {
        const imported = importedPaths.find((candidate) => importMatches(candidate, forbidden));
        if (imported) violations.push(["U5_STORE_FORBIDDEN_IMPORT", `${file.path} imports ${imported}`]);
      }
      const bodies = topLevelStructBodies(file.lexical.code, "SweepCompletionAuthority");
      for (const body of bodies) {
        authorityDeclarations += 1;
        for (const field of exportedAuthorityFields(body)) {
          violations.push(["U5_STORE_AUTHORITY_EXPORTED_FIELD", `${file.path}: ${field}`]);
        }
      }
      if (/\bfunc\s+(?:\([^)]*\)\s*)?New[A-Za-z0-9_]*SweepCompletionAuthority\s*(?:\[[^\]]*\]\s*)?\(/u.test(file.lexical.code)) {
        violations.push(["U5_STORE_AUTHORITY_PUBLIC_CONSTRUCTOR", file.path]);
      }
    }

    if (isReduction) {
      if (importedPaths.includes(`${modulePrefix}internal/reduce`)) reductionImportsReduce = true;
      if (importedPaths.includes(`${modulePrefix}internal/store`)) reductionImportsStore = true;
    }

    const oneMinimal = oneMinimalReference(file.lexical);
    if (oneMinimal && !isReduction) {
      violations.push(["U5_ONE_MINIMAL_AUTHORITY_OUTSIDE_REDUCTION", file.path]);
    }
    if (oneMinimal && isReduction) reductionGradeReferences += 1;

    const nakedEquality = nakedPreservationDecisionEquality(file.lexical.code);
    if (nakedEquality) violations.push(["U5_NAKED_PRESERVATION_DIGEST_EQUALITY", `${file.path}: ${nakedEquality}`]);
    if (/\bDisplayGroups\s*\(/u.test(file.lexical.code) ||
        file.lexical.literals.some((literal) => /display[_ -]?groups/iu.test(literal))) {
      violations.push(["U5_DISPLAY_GROUPS_AS_AUTHORITY", file.path]);
    }
    if (hasIdentifierMatching(file.lexical.code, /partition_?shape/iu) ||
        file.lexical.literals.some((literal) => /partition[_ -]?shape/iu.test(literal))) {
      violations.push(["U5_PARTITION_SHAPE_AS_AUTHORITY", file.path]);
    }
    if (hasIdentifierMatching(file.lexical.code, /(?:cluster|group)_?ordinal/iu) ||
        file.lexical.literals.some((literal) => /(?:cluster|group)[_-]?ordinal|cluster[_-]?id/iu.test(literal))) {
      violations.push(["U5_CLUSTER_ORDINAL_PERSISTENCE_IDENTITY", file.path]);
    }
  }

  if (!storeImportsReduce) violations.push(["U5_STORE_REDUCE_IMPORT_REQUIRED", "internal/store"]);
  if (!reductionImportsReduce) violations.push(["U5_REDUCTION_REDUCE_IMPORT_REQUIRED", "internal/reduction"]);
  if (!reductionImportsStore) violations.push(["U5_REDUCTION_STORE_IMPORT_REQUIRED", "internal/reduction"]);
  if (reduceAssessCalls < 1) violations.push(["U5_REDUCE_ASSESS_PRESERVATION_REQUIRED", String(reduceAssessCalls)]);
  if (authorityDeclarations !== 1) violations.push(["U5_STORE_AUTHORITY_DECLARATION_COUNT", String(authorityDeclarations)]);
  if (reductionGradeReferences < 1) violations.push(["U5_REDUCTION_ONE_MINIMAL_AUTHORITY_REQUIRED", String(reductionGradeReferences)]);
  return violations;
}

function assertSameManifest(before, after) {
  if (before.digest !== after.digest || before.files.length !== after.files.length) {
    throw new ArchitectureError(
      "U5_MANIFEST_CHANGED_DURING_SCAN",
      `${before.files.length}/${before.digest} -> ${after.files.length}/${after.digest}`,
    );
  }
  for (let index = 0; index < before.files.length; index += 1) {
    const left = before.files[index];
    const right = after.files[index];
    if (left.path !== right.path || left.bytes !== right.bytes || left.sha256 !== right.sha256) {
      throw new ArchitectureError("U5_MANIFEST_CHANGED_DURING_SCAN", `${left.path} -> ${right.path}`);
    }
  }
}

async function main() {
  if (process.argv.length !== 2) {
    throw new ArchitectureError("U5_ARCHITECTURE_ARGUMENTS", "no arguments or root overrides are accepted");
  }
  const initial = await readManifest();
  verifyManifestWithTool(initial);
  // U5_SELFTEST_MANIFEST_BARRIER
  const violations = inspectManifest(initial);
  const final = await readManifest();
  assertSameManifest(initial, final);
  if (violations.length > 0) {
    const detail = violations
      .map(([code, message]) => `${code}: ${message}`)
      .sort((left, right) => left.localeCompare(right, "en"))
      .join("\n");
    throw new ArchitectureError("U5_ARCHITECTURE_VIOLATION", detail);
  }
  process.stdout.write(
    `U5 architecture boundary OK (${initial.files.length} exact Go files; manifest ${initial.digest})\n`,
  );
}

main().catch((error) => {
  process.stderr.write(`${error.stack ?? error}\n`);
  process.exitCode = 1;
});
