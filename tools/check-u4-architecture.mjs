#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { readdir, readFile } from "node:fs/promises";
import { dirname, extname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const modulePrefix = "github.com/nelsonwerd/countershape/";
const httpTop = `${modulePrefix}internal/adapters/http`;
const httpModel = `${httpTop}/model`;

// U4 is not allowed to buy its second adapter by editing generic truth or the
// sealed CLI adapter. These aggregates bind every path and byte, including
// tests, at the U3 receipt boundary. A deliberate future unit must replace the
// contract rather than silently growing an exception list.
const sealedAggregates = Object.freeze([
  Object.freeze({
    name: "generic truth",
    roots: Object.freeze([
      "internal/canon", "internal/domain", "internal/spec", "internal/observe",
      "internal/compare", "internal/reduce", "internal/choice",
    ]),
    files: 39,
    sha256: "44324a7024717bdaf8b282f9e1010c9cf84045a92b24df5b252c8eb4ebf99279",
  }),
  Object.freeze({
    name: "typed CLI adapter",
    roots: Object.freeze(["internal/adapters/cli"]),
    files: 14,
    sha256: "8cb1dc1d9633b52be093962444b5411ab23150ba8e79028aedb407ee584814ef",
  }),
]);

const scannedRoots = Object.freeze([
  "internal/adapters/http",
  "internal/world",
  "testkit/studies/http_invoices",
]);

class ArchitectureError extends Error {
  constructor(code, detail) {
    super(`${code}: ${detail}`);
    this.code = code;
  }
}

function slash(path) {
  return path.split(sep).join("/");
}

function within(path, directory) {
  return path === directory || path.startsWith(`${directory}/`);
}

async function enumerateFiles(directory, accepted = () => true) {
  const absolute = resolve(root, directory);
  const result = [];
  let entries;
  try {
    entries = await readdir(absolute, { withFileTypes: true });
  } catch (error) {
    if (error.code === "ENOENT") return result;
    throw error;
  }
  entries.sort((left, right) => left.name.localeCompare(right.name, "en"));
  for (const entry of entries) {
    const next = join(absolute, entry.name);
    if (entry.isSymbolicLink()) {
      throw new ArchitectureError("U4_ARCHITECTURE_SYMLINK", slash(relative(root, next)));
    }
    if (entry.isDirectory()) result.push(...await enumerateFiles(slash(relative(root, next)), accepted));
    if (entry.isFile() && accepted(next)) result.push(next);
  }
  return result;
}

function importedPackages(source) {
  return [...source.matchAll(/(?:^|\s)(?:[._A-Za-z][._A-Za-z0-9]*\s+)?"([^"]+)"/gmu)]
    .map((match) => match[1]);
}

// Blank comments and literals while retaining line breaks and code symbols.
function goCodeOnly(source) {
  const output = [...source];
  let state = "code";
  for (let index = 0; index < source.length; index += 1) {
    const character = source[index];
    const next = source[index + 1] ?? "";
    if (state === "code") {
      if (character === "/" && next === "/") {
        output[index] = output[index + 1] = " ";
        index += 1;
        state = "line-comment";
      } else if (character === "/" && next === "*") {
        output[index] = output[index + 1] = " ";
        index += 1;
        state = "block-comment";
      } else if (character === '"' || character === "'" || character === "`") {
        output[index] = " ";
        state = character === '"' ? "string" : character === "'" ? "rune" : "raw";
      }
    } else if (state === "line-comment") {
      output[index] = character === "\n" ? "\n" : " ";
      if (character === "\n") state = "code";
    } else if (state === "block-comment") {
      output[index] = character === "\n" ? "\n" : " ";
      if (character === "*" && next === "/") {
        output[index + 1] = " ";
        index += 1;
        state = "code";
      }
    } else if (state === "raw") {
      output[index] = character === "\n" ? "\n" : " ";
      if (character === "`") state = "code";
    } else {
      output[index] = character === "\n" ? "\n" : " ";
      if (character === "\\") {
        index += 1;
        if (index < output.length) output[index] = source[index] === "\n" ? "\n" : " ";
      } else if ((state === "string" && character === '"') || (state === "rune" && character === "'")) {
        state = "code";
      }
    }
  }
  return output.join("");
}

async function aggregate(contract) {
  const files = [];
  for (const directory of contract.roots) files.push(...await enumerateFiles(directory));
  files.sort((left, right) => slash(relative(root, left)).localeCompare(slash(relative(root, right)), "en"));
  const digest = createHash("sha256");
  for (const file of files) {
    digest.update(slash(relative(root, file)));
    digest.update("\0");
    digest.update(await readFile(file));
    digest.update("\0");
  }
  return { files: files.length, sha256: digest.digest("hex") };
}

async function assertSealedAggregates() {
  for (const contract of sealedAggregates) {
    const actual = await aggregate(contract);
    if (actual.files !== contract.files || actual.sha256 !== contract.sha256) {
      throw new ArchitectureError(
        "U4_SEALED_BOUNDARY_CHANGED",
        `${contract.name}: files=${actual.files} sha256=${actual.sha256}`,
      );
    }
  }
}

function inspectGo(relativePath, source) {
  const violations = [];
  const imports = importedPackages(source);
  const code = goCodeOnly(source);
  const production = !relativePath.endsWith("_test.go");
  const adapter = within(relativePath, "internal/adapters/http");
  const model = within(relativePath, "internal/adapters/http/model");
  const world = within(relativePath, "internal/world");
  const study = within(relativePath, "testkit/studies/http_invoices");

  if (model && imports.includes(`${modulePrefix}internal/world`)) {
    violations.push(["U4_HTTP_MODEL_IMPORTS_WORLD", relativePath]);
  }
  if (adapter && !model) {
    for (const owner of ["internal/compare", "internal/reduce", "internal/choice"]) {
      if (imports.includes(`${modulePrefix}${owner}`)) {
        violations.push(["U4_HTTP_ADAPTER_OWNS_GENERIC_TRUTH", `${relativePath} imports ${owner}`]);
      }
    }
  }
  if (world) {
    for (const imported of imports) {
      if ((imported === httpTop || imported.startsWith(`${httpTop}/`)) && imported !== httpModel) {
        violations.push(["U4_WORLD_IMPORTS_IMPURE_HTTP_ADAPTER", `${relativePath} imports ${imported}`]);
      }
    }
    if (/"http\.(?:status|header|body)[^"]*"/u.test(source)) {
      violations.push(["U4_WORLD_OWNS_HTTP_FIELD", relativePath]);
    }
  }
  const policyHTTPImport = imports.find((imported) => imported === "net/http" || imported.startsWith("net/http/"));
  if ((adapter || world || (study && production)) && policyHTTPImport) {
    violations.push(["U4_POLICY_HTTP_CLIENT_FORBIDDEN", `${relativePath} imports ${policyHTTPImport}`]);
  }
  if (production && (adapter || world) && /\b(?:ProxyFromEnvironment|HTTP_PROXY|HTTPS_PROXY|NO_PROXY|CheckRedirect)\b/u.test(source)) {
    violations.push(["U4_AMBIENT_HTTP_POLICY", relativePath]);
  }
  if (production && (adapter || world || study)) {
    if (/\b(?:SharedRoot|ReuseRoot|sharedRoot|reuseRoot)\b/u.test(code) || /COUNTERSHAPE_(?:SHARED|REUSED)_ROOT/u.test(source)) {
      violations.push(["U4_PRODUCT_SHARED_ROOT_OPTION", relativePath]);
    }
    if (source.includes("NEGATIVE_FIXTURE_NON_PRODUCT")) {
      violations.push(["U4_NEGATIVE_FIXTURE_ESCAPED_TEST", relativePath]);
    }
  }
  if (study && production && /\bNewWorldPlan\b/u.test(code)) {
    violations.push(["U4_STUDY_BYPASSES_SOURCE_COMPILER", relativePath]);
  }
  return violations;
}

async function inspectRepository() {
  const files = [];
  for (const directory of scannedRoots) {
    files.push(...await enumerateFiles(directory, (path) => extname(path) === ".go"));
  }
  files.sort((left, right) => slash(relative(root, left)).localeCompare(slash(relative(root, right)), "en"));
  const violations = [];
  let mapConstructors = 0;
  let parseSourceCalls = 0;
  let compileCalls = 0;
  let executeHTTPCalls = 0;
  let negativeFixtureTests = 0;
  for (const file of files) {
    const relativePath = slash(relative(root, file));
    const source = await readFile(file, "utf8");
    violations.push(...inspectGo(relativePath, source));
    if (within(relativePath, "testkit/studies/http_invoices") && !relativePath.endsWith("_test.go")) {
      const code = goCodeOnly(source);
      mapConstructors += [...code.matchAll(/\bNewCandidateOutcomeMap\s*\(/gu)].length;
      parseSourceCalls += [...code.matchAll(/\bParseSource\s*\(/gu)].length;
      compileCalls += [...code.matchAll(/\bCompile\s*\(/gu)].length;
      executeHTTPCalls += [...code.matchAll(/\bExecuteHTTP\s*\(/gu)].length;
    }
    if (within(relativePath, "testkit/studies/http_invoices") && relativePath.endsWith("_test.go") && source.includes("NEGATIVE_FIXTURE_NON_PRODUCT")) {
      negativeFixtureTests += 1;
    }
  }
  if (mapConstructors !== 1) violations.push(["U4_OUTCOME_MAP_AUTHORITY_COUNT", String(mapConstructors)]);
  if (parseSourceCalls < 1 || compileCalls < 1) violations.push(["U4_STUDY_SOURCE_ROUTE_MISSING", `${parseSourceCalls}/${compileCalls}`]);
  if (executeHTTPCalls !== 1) violations.push(["U4_EXECUTE_HTTP_AUTHORITY_COUNT", String(executeHTTPCalls)]);
  if (negativeFixtureTests < 1) violations.push(["U4_NEGATIVE_FIXTURE_TEST_MISSING", String(negativeFixtureTests)]);
  if (violations.length > 0) {
    throw new ArchitectureError(
      "U4_ARCHITECTURE_VIOLATION",
      violations.map(([code, detail]) => `${code}: ${detail}`).sort().join("\n"),
    );
  }
  return files.length;
}

function selfTest() {
  const cases = [
    ["internal/adapters/http/model/hostile.go", `package model\nimport "${modulePrefix}internal/world"\n`, "U4_HTTP_MODEL_IMPORTS_WORLD"],
    ["internal/world/hostile.go", `package world\nimport "${modulePrefix}internal/adapters/http"\n`, "U4_WORLD_IMPORTS_IMPURE_HTTP_ADAPTER"],
    ["internal/adapters/http/hostile.go", "package http\nimport \"net/http\"\n", "U4_POLICY_HTTP_CLIENT_FORBIDDEN"],
    ["internal/adapters/http/hostile.go", "package http\nimport \"net/http/cookiejar\"\n", "U4_POLICY_HTTP_CLIENT_FORBIDDEN"],
    ["internal/world/hostile.go", "package world\nconst field = \"http.status\"\n", "U4_WORLD_OWNS_HTTP_FIELD"],
    ["testkit/studies/http_invoices/hostile.go", "package http_invoices\nvar SharedRoot string\n", "U4_PRODUCT_SHARED_ROOT_OPTION"],
    ["testkit/studies/http_invoices/hostile.go", "package http_invoices\nvar makePlan = NewWorldPlan\n", "U4_STUDY_BYPASSES_SOURCE_COMPILER"],
  ];
  for (const [path, source, code] of cases) {
    if (!inspectGo(path, source).some(([actual]) => actual === code)) {
      throw new ArchitectureError("U4_ARCHITECTURE_SELFTEST_FALSE_NEGATIVE", `${path} did not produce ${code}`);
    }
  }
  const clean = inspectGo(
    "internal/adapters/http/model/clean.go",
    `package model\nimport "github.com/nelsonwerd/countershape/internal/domain"\nvar _ domain.WorldPlan\n`,
  );
  if (clean.length !== 0) throw new ArchitectureError("U4_ARCHITECTURE_SELFTEST_FALSE_POSITIVE", JSON.stringify(clean));
}

async function main() {
  const arguments_ = process.argv.slice(2);
  if (arguments_.length > 1 || (arguments_.length === 1 && arguments_[0] !== "--self-test")) {
    throw new ArchitectureError("U4_BOUNDARY_ARGUMENTS", "only --self-test is accepted");
  }
  if (arguments_[0] === "--self-test") selfTest();
  const prior = spawnSync(process.execPath, [resolve(root, "tools/check-u3-architecture.mjs")], {
    cwd: root,
    encoding: "utf8",
    env: { ...process.env, NO_COLOR: "1" },
  });
  if (prior.status !== 0) {
    throw new ArchitectureError("U4_PRIOR_ARCHITECTURE_FAILED", `${prior.stdout}${prior.stderr}`.trim());
  }
  await assertSealedAggregates();
  const count = await inspectRepository();
  process.stdout.write(`U4 architecture boundary OK (${count} Go files; U3 generic and CLI aggregates exact)\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.stack ?? error}\n`);
  process.exitCode = 1;
});
