#!/usr/bin/env node

import { readdir, readFile } from "node:fs/promises";
import { dirname, extname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const modulePrefix = "github.com/nelsonwerd/countershape/";
const genericRoots = Object.freeze([
  "internal/canon",
  "internal/domain",
  "internal/spec",
  "internal/observe",
  "internal/compare",
  "internal/reduce",
  "internal/choice",
]);
const scannedRoots = Object.freeze([
  ...genericRoots,
  "internal/adapters/cli",
  "internal/world",
  "testkit/studies/cli_precedence",
]);

class ArchitectureError extends Error {
  constructor(code, detail) {
    super(`${code}: ${detail}`);
    this.code = code;
  }
}

function slashPath(path) {
  return path.split(sep).join("/");
}

function within(relativePath, directory) {
  return relativePath === directory || relativePath.startsWith(`${directory}/`);
}

function importedPackages(source) {
  const result = [];
  for (const match of source.matchAll(/(?:^|\s)(?:[._A-Za-z][._A-Za-z0-9]*\s+)?"([^"]+)"/gmu)) {
    result.push(match[1]);
  }
  return result;
}

function importedAlias(source, packagePath, fallback) {
  const escaped = packagePath.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
  const match = source.match(new RegExp(
    `(?:^|\\n)\\s*(?:import\\s+(?:\\(\\s*)?)?(?:([._A-Za-z][._A-Za-z0-9]*)\\s+)?"${escaped}"`,
    "u",
  ));
  return match?.[1] ?? (match ? fallback : "");
}

// Return a same-length lexical view containing only Go code. Comments and all
// literal forms are blanked so authority-symbol checks cannot be satisfied or
// tripped by prose, while aliases such as `var resolve = model.BindExecution`
// remain visible. This is deliberately smaller than a Go parser but closed for
// the five lexical states relevant to symbol-reference detection.
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
      } else if (character === '"') {
        output[index] = " ";
        state = "string";
      } else if (character === "'") {
        output[index] = " ";
        state = "rune";
      } else if (character === "`") {
        output[index] = " ";
        state = "raw-string";
      }
    } else if (state === "line-comment") {
      if (character === "\n") {
        state = "code";
      } else {
        output[index] = " ";
      }
    } else if (state === "block-comment") {
      output[index] = character === "\n" ? "\n" : " ";
      if (character === "*" && next === "/") {
        output[index + 1] = " ";
        index += 1;
        state = "code";
      }
    } else if (state === "raw-string") {
      output[index] = character === "\n" ? "\n" : " ";
      if (character === "`") state = "code";
    } else {
      output[index] = character === "\n" ? "\n" : " ";
      if (character === "\\") {
        if (index + 1 < source.length) output[index + 1] = source[index + 1] === "\n" ? "\n" : " ";
        index += 1;
      } else if ((state === "string" && character === '"') || (state === "rune" && character === "'")) {
        state = "code";
      }
    }
  }
  return output.join("");
}

function genericCaptureDTO(source) {
  const structs = source.matchAll(/\btype\s+([A-Za-z0-9_]+)\s+struct\s*\{([\s\S]*?)\}/gu);
  for (const match of structs) {
    const name = match[1];
    const body = match[2];
    const serialized = /`json:"/u.test(body);
    const untyped = /\bjson\.RawMessage\b|\bmap\[string\](?:any|interface\s*\{\})/u.test(body);
    const tags = [...body.matchAll(/`json:"([^",]+)(?:,[^"]*)?"`/gu)].map((tag) => tag[1]);
    const adapterUnion = tags.includes("cli") && tags.includes("http");
    const channelTag = tags.some((tag) => [
      "stdout", "stderr", "exit_code", "exit_signal", "http_body", "http_headers", "http_status",
    ].includes(tag));
    const captureNamed = /(?:Capture|Observation)/u.test(name);
    if (serialized && (adapterUnion || (captureNamed && (untyped || channelTag)))) {
      return name;
    }
  }
  return "";
}

function inspectSource(relativePath, source) {
  const violations = [];
  const codeOnly = goCodeOnly(source);
  const generic = genericRoots.some((directory) => within(relativePath, directory));
  const imports = importedPackages(source);
  const topLevelCLI = `${modulePrefix}internal/adapters/cli`;
  const cliModel = `${topLevelCLI}/model`;
  const domainPackage = `${modulePrefix}internal/domain`;

  if (generic) {
    for (const imported of imports) {
      if (imported === topLevelCLI || imported.startsWith(`${topLevelCLI}/`)) {
        violations.push(["U3_GENERIC_IMPORTS_CLI_ADAPTER", `${relativePath} imports ${imported}`]);
      }
    }
    const fieldLiteral = source.match(/"cli\.(?:completion|exit|stdout|stderr)(?:[._-][a-z0-9_-]+)+"/u)?.[0];
    if (fieldLiteral) {
      violations.push(["U3_GENERIC_SWITCHES_ON_CLI_FIELD", `${relativePath} contains ${fieldLiteral}`]);
    }
    const dto = genericCaptureDTO(source);
    if (dto) {
      violations.push(["U3_GENERIC_ADAPTER_CAPTURE_DTO", `${relativePath} defines ${dto}`]);
    }
  }

  if (within(relativePath, "internal/world")) {
    for (const imported of imports) {
      if ((imported === topLevelCLI || imported.startsWith(`${topLevelCLI}/`)) && imported !== cliModel) {
        violations.push([
          "U3_WORLD_IMPORTS_IMPURE_CLI_ADAPTER",
          `${relativePath} imports ${imported}; only ${cliModel} is cycle-safe execution authority`,
        ]);
      }
    }
  }
  if (within(relativePath, "internal/adapters/cli/model") && imports.includes(`${modulePrefix}internal/world`)) {
    violations.push(["U3_CLI_MODEL_IMPORTS_WORLD", `${relativePath} creates a model/world authority cycle`]);
  }
  if (within(relativePath, "internal/adapters/cli") && !within(relativePath, "internal/adapters/cli/model")) {
    for (const forbidden of ["internal/compare", "internal/reduce", "internal/choice"]) {
      const imported = `${modulePrefix}${forbidden}`;
      if (imports.includes(imported)) {
        violations.push(["U3_CLI_ADAPTER_OWNS_GENERIC_TRUTH", `${relativePath} imports ${imported}`]);
      }
    }
  }
  const production = !relativePath.endsWith("_test.go");
  const modelAlias = importedAlias(source, cliModel, "model");
  if (production && modelAlias && relativePath !== "internal/adapters/cli/stimulus.go") {
    const escapedAlias = modelAlias.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
    const projectionReference = modelAlias === "."
      ? /\bResolveCLIProjectionAuthority\b/u
      : new RegExp(`\\b${escapedAlias}\\.ResolveCLIProjectionAuthority\\b`, "u");
    const bindingReference = modelAlias === "."
      ? /\bBindExecution\b/u
      : new RegExp(`\\b${escapedAlias}\\.BindExecution\\b`, "u");
    if (projectionReference.test(codeOnly)) {
      violations.push([
        "U3_PRODUCTION_BYPASSES_CLI_PROJECTION_RESOLVER",
        `${relativePath} constructs cycle-free projection authority outside the top-level CLI registry wrapper`,
      ]);
    }
    if (bindingReference.test(codeOnly)) {
      violations.push([
        "U3_PRODUCTION_BYPASSES_CLI_EXECUTION_BINDING",
        `${relativePath} constructs CLI execution authority outside the top-level resolved-contract wrapper`,
      ]);
    }
  }
  if (production && relativePath === "internal/adapters/cli/stimulus.go") {
    for (const authority of ["ResolveCLIProjectionAuthority", "BindExecution"]) {
      const count = [...codeOnly.matchAll(new RegExp(`\\bmodel\\.${authority}\\b`, "gu"))].length;
      if (count !== 1) {
        violations.push([
          "U3_CLI_WRAPPER_AUTHORITY_REFERENCE_COUNT",
          `${relativePath} contains ${count} model.${authority} references; expected exactly 1`,
        ]);
      }
    }
  }
  if (production && within(relativePath, "internal/adapters/cli/model")) {
    const samePackageAuthorities = [
      {
        name: "ResolveCLIProjectionAuthority",
        implementation: "internal/adapters/cli/model/projection_authority.go",
        expectedCalls: 2,
        code: "U3_CLI_MODEL_BYPASSES_PROJECTION_RESOLVER",
      },
      {
        name: "BindExecution",
        implementation: "internal/adapters/cli/model/binding.go",
        expectedCalls: 2,
        code: "U3_CLI_MODEL_BYPASSES_EXECUTION_BINDING",
      },
    ];
    for (const authority of samePackageAuthorities) {
      const escapedName = authority.name.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
      const callCount = [...codeOnly.matchAll(new RegExp(`\\b${escapedName}\\b`, "gu"))].length;
      const expected = relativePath === authority.implementation ? authority.expectedCalls : 0;
      if (callCount !== expected) {
        violations.push([
          authority.code,
          `${relativePath} contains ${callCount} same-package ${authority.name} symbol references; expected ${expected}`,
        ]);
      }
    }
  }
  if (production && within(relativePath, "testkit/studies/cli_precedence")) {
    const domainAlias = importedAlias(source, domainPackage, "domain");
    if (domainAlias) {
      const escapedAlias = domainAlias.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
      const directPlanReference = domainAlias === "."
        ? /\bNewWorldPlan\b/u
        : new RegExp(`\\b${escapedAlias}\\.NewWorldPlan\\b`, "u");
      if (directPlanReference.test(codeOnly)) {
        violations.push([
          "U3_STUDY_BYPASSES_SOURCE_COMPILER",
          `${relativePath} references direct WorldPlan construction instead of ParseSource -> Compile`,
        ]);
      }
    }
  }
  return violations;
}

async function enumerateGoFiles(directory) {
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
    const path = join(absolute, entry.name);
    if (entry.isDirectory()) {
      result.push(...await enumerateGoFiles(slashPath(relative(root, path))));
    } else if (entry.isFile() && extname(entry.name) === ".go") {
      result.push(path);
    } else if (entry.isSymbolicLink()) {
      throw new ArchitectureError("U3_ARCHITECTURE_SYMLINK", slashPath(relative(root, path)));
    }
  }
  return result;
}

async function inspectRepository() {
  const files = [];
  for (const directory of scannedRoots) files.push(...await enumerateGoFiles(directory));
  files.sort((left, right) => left.localeCompare(right, "en"));
  const seen = new Set();
  const violations = [];
  for (const absolute of files) {
    const relativePath = slashPath(relative(root, absolute));
    if (seen.has(relativePath)) continue;
    seen.add(relativePath);
    const source = await readFile(absolute, "utf8");
    violations.push(...inspectSource(relativePath, source));
  }
  if (violations.length > 0) {
    const ordered = violations
      .map(([code, detail]) => `${code}: ${detail}`)
      .sort((left, right) => left.localeCompare(right, "en"));
    throw new ArchitectureError("U3_ARCHITECTURE_VIOLATION", ordered.join("\n"));
  }
  return seen.size;
}

function assertSelfTestViolation(name, path, source, code) {
  const violations = inspectSource(path, source);
  if (!violations.some(([candidate]) => candidate === code)) {
    throw new ArchitectureError("U3_ARCHITECTURE_SELFTEST_FALSE_NEGATIVE", `${name} did not produce ${code}`);
  }
}

function runSelfTest() {
  assertSelfTestViolation(
    "generic imports adapter",
    "internal/observe/hostile.go",
    `package observe\nimport "${modulePrefix}internal/adapters/cli"\n`,
    "U3_GENERIC_IMPORTS_CLI_ADAPTER",
  );
  assertSelfTestViolation(
    "canon imports adapter",
    "internal/canon/hostile.go",
    `package canon\nimport "${modulePrefix}internal/adapters/cli"\n`,
    "U3_GENERIC_IMPORTS_CLI_ADAPTER",
  );
  assertSelfTestViolation(
    "spec switches on field",
    "internal/spec/hostile.go",
    "package spec\nconst field = \"cli.stderr.text\"\n",
    "U3_GENERIC_SWITCHES_ON_CLI_FIELD",
  );
  assertSelfTestViolation(
    "generic switches on field",
    "internal/compare/hostile.go",
    "package compare\nconst field = \"cli.stdout.json.mode\"\n",
    "U3_GENERIC_SWITCHES_ON_CLI_FIELD",
  );
  assertSelfTestViolation(
    "generic union DTO",
    "internal/domain/hostile.go",
    "package domain\ntype GenericObservation struct {\n CLI json.RawMessage `json:\"cli\"`\n HTTP json.RawMessage `json:\"http\"`\n}\n",
    "U3_GENERIC_ADAPTER_CAPTURE_DTO",
  );
  assertSelfTestViolation(
    "typed generic union DTO",
    "internal/domain/hostile.go",
    "package domain\ntype CLIView struct{}\ntype HTTPView struct{}\ntype GenericCapture struct { CLI CLIView `json:\"cli\"`; HTTP HTTPView `json:\"http\"` }\n",
    "U3_GENERIC_ADAPTER_CAPTURE_DTO",
  );
  assertSelfTestViolation(
    "typed generic channel bag",
    "internal/observe/hostile.go",
    "package observe\ntype ChannelCapture struct {\n Stdout string `json:\"stdout\"`\n HTTPStatus int `json:\"http_status\"`\n}\n",
    "U3_GENERIC_ADAPTER_CAPTURE_DTO",
  );
  assertSelfTestViolation(
    "world imports top-level adapter",
    "internal/world/hostile.go",
    `package world\nimport "${modulePrefix}internal/adapters/cli"\n`,
    "U3_WORLD_IMPORTS_IMPURE_CLI_ADAPTER",
  );
  assertSelfTestViolation(
    "world imports sibling adapter package",
    "internal/world/hostile.go",
    `package world\nimport "${modulePrefix}internal/adapters/cli/impure"\n`,
    "U3_WORLD_IMPORTS_IMPURE_CLI_ADAPTER",
  );
  assertSelfTestViolation(
    "model imports world",
    "internal/adapters/cli/model/hostile.go",
    `package model\nimport "${modulePrefix}internal/world"\n`,
    "U3_CLI_MODEL_IMPORTS_WORLD",
  );
  assertSelfTestViolation(
    "adapter imports generic truth owner",
    "internal/adapters/cli/hostile.go",
    `package cli\nimport "${modulePrefix}internal/compare"\n`,
    "U3_CLI_ADAPTER_OWNS_GENERIC_TRUTH",
  );
  assertSelfTestViolation(
    "production bypasses projection resolver",
    "internal/world/hostile.go",
    `package world\nimport climodel "${modulePrefix}internal/adapters/cli/model"\nfunc hostile() { _, _ = climodel.ResolveCLIProjectionAuthority() }\n`,
    "U3_PRODUCTION_BYPASSES_CLI_PROJECTION_RESOLVER",
  );
  assertSelfTestViolation(
    "production bypasses execution binding",
    "internal/world/hostile.go",
    `package world\nimport model "${modulePrefix}internal/adapters/cli/model"\nfunc hostile() { _, _ = model.BindExecution() }\n`,
    "U3_PRODUCTION_BYPASSES_CLI_EXECUTION_BINDING",
  );
  assertSelfTestViolation(
    "same-package production bypasses projection resolver",
    "internal/adapters/cli/model/hostile.go",
    "package model\nfunc hostile() { _, _ = ResolveCLIProjectionAuthority() }\n",
    "U3_CLI_MODEL_BYPASSES_PROJECTION_RESOLVER",
  );
  assertSelfTestViolation(
    "same-package production bypasses execution binding",
    "internal/adapters/cli/model/hostile.go",
    "package model\nfunc hostile() { _, _ = BindExecution() }\n",
    "U3_CLI_MODEL_BYPASSES_EXECUTION_BINDING",
  );
  assertSelfTestViolation(
    "production aliases projection resolver",
    "internal/world/hostile.go",
    `package world\nimport model "${modulePrefix}internal/adapters/cli/model"\nvar resolve = model.ResolveCLIProjectionAuthority\n`,
    "U3_PRODUCTION_BYPASSES_CLI_PROJECTION_RESOLVER",
  );
  assertSelfTestViolation(
    "production dot-imports projection resolver",
    "internal/world/hostile.go",
    `package world\nimport . "${modulePrefix}internal/adapters/cli/model"\nfunc hostile() { _, _ = ResolveCLIProjectionAuthority() }\n`,
    "U3_PRODUCTION_BYPASSES_CLI_PROJECTION_RESOLVER",
  );
  assertSelfTestViolation(
    "production dot-imports execution binding",
    "internal/world/hostile.go",
    `package world\nimport . "${modulePrefix}internal/adapters/cli/model"\nvar bind = BindExecution\n`,
    "U3_PRODUCTION_BYPASSES_CLI_EXECUTION_BINDING",
  );
  assertSelfTestViolation(
    "same-package production aliases execution binding",
    "internal/adapters/cli/model/hostile.go",
    "package model\nvar bind = BindExecution\n",
    "U3_CLI_MODEL_BYPASSES_EXECUTION_BINDING",
  );
  assertSelfTestViolation(
    "study bypasses source compiler",
    "testkit/studies/cli_precedence/hostile.go",
    `package cli_precedence\nimport "${modulePrefix}internal/domain"\nfunc hostile() { _, _ = domain.NewWorldPlan(domain.WorldPlanConfig{}) }\n`,
    "U3_STUDY_BYPASSES_SOURCE_COMPILER",
  );
  assertSelfTestViolation(
    "study aliases direct plan construction",
    "testkit/studies/cli_precedence/hostile.go",
    `package cli_precedence\nimport d "${modulePrefix}internal/domain"\nvar makePlan = d.NewWorldPlan\n`,
    "U3_STUDY_BYPASSES_SOURCE_COMPILER",
  );
  assertSelfTestViolation(
    "study dot-imports direct plan construction",
    "testkit/studies/cli_precedence/hostile.go",
    `package cli_precedence\nimport . "${modulePrefix}internal/domain"\nvar makePlan = NewWorldPlan\n`,
    "U3_STUDY_BYPASSES_SOURCE_COMPILER",
  );
  const clean = inspectSource(
    "internal/observe/clean.go",
    "package observe\ntype StructuralCapture struct { observationDigest Digest }\n",
  );
  if (clean.length !== 0) {
    throw new ArchitectureError("U3_ARCHITECTURE_SELFTEST_FALSE_POSITIVE", JSON.stringify(clean));
  }
  const cleanBudgets = inspectSource(
    "internal/domain/clean.go",
    "package domain\ntype Budgets struct { StdoutBytes int `json:\"stdout_bytes\"`; HTTPBodyBytes int `json:\"http_body_bytes\"` }\n",
  );
  if (cleanBudgets.length !== 0) {
    throw new ArchitectureError("U3_ARCHITECTURE_SELFTEST_FALSE_POSITIVE", JSON.stringify(cleanBudgets));
  }
  const cleanStudyProse = inspectSource(
    "testkit/studies/cli_precedence/clean.go",
    `package cli_precedence\nimport domain "${modulePrefix}internal/domain"\n// domain.NewWorldPlan is forbidden here.\nconst note = "domain.NewWorldPlan"\nvar _ domain.WorldPlan\n`,
  );
  if (cleanStudyProse.length !== 0) {
    throw new ArchitectureError("U3_ARCHITECTURE_SELFTEST_FALSE_POSITIVE", JSON.stringify(cleanStudyProse));
  }
}

async function main() {
  const arguments_ = process.argv.slice(2);
  if (arguments_.length > 1 || (arguments_.length === 1 && arguments_[0] !== "--self-test")) {
    throw new ArchitectureError("U3_BOUNDARY_ARGUMENTS", "only the fixed --self-test mode is accepted");
  }
  if (arguments_[0] === "--self-test") runSelfTest();
  const fileCount = await inspectRepository();
  process.stdout.write(`U3 architecture boundary OK (${fileCount} Go files)\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.stack ?? error}\n`);
  process.exitCode = 1;
});
