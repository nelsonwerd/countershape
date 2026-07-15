#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { mkdir, mkdtemp, readFile, rm, symlink, writeFile } from "node:fs/promises";
import { createServer } from "node:net";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const modulePath = fileURLToPath(import.meta.url);
const repositoryRoot = resolve(dirname(modulePath), "..");
const analyzerPath = resolve(repositoryRoot, "tools/check-u5-architecture.mjs");
const modulePrefix = "github.com/nelsonwerd/countershape/";

const cleanReduce = `package reduce

import "${modulePrefix}internal/compare"

func assess(baseline, observed compare.CandidateOutcomeMap) compare.PreservationAssessment {
	return compare.AssessPreservation(baseline, observed)
}

func samePreservationDigestForLineage(left, right compare.PreservationMapDigest) bool {
	return left == right
}
`;

const cleanStore = `package store

import "${modulePrefix}internal/reduce"

type SweepCompletionAuthority struct {
	draft reduce.SweepCompletionDraft
}

func retainDraft(draft reduce.SweepCompletionDraft) { _ = draft }
`;

const cleanReduction = `package reduction

import (
	"${modulePrefix}internal/reduce"
	"${modulePrefix}internal/store"
)

const oneMinimalUnder = "ONE_MINIMAL_UNDER"

func finalize(draft reduce.SweepCompletionDraft, authority store.SweepCompletionAuthority) string {
	_, _ = draft, authority
	return oneMinimalUnder
}
`;

class SelfTestError extends Error {
  constructor(code, detail) {
    super(`${code}: ${detail}`);
    this.code = code;
  }
}

async function writeSource(root, relativePath, source) {
  const absolute = resolve(root, relativePath);
  await mkdir(dirname(absolute), { recursive: true, mode: 0o700 });
  await writeFile(absolute, source, { mode: 0o600 });
}

async function createFixture(analyzerSource) {
  const parent = await mkdtemp(join(tmpdir(), "countershape-u5-architecture-selftest-"));
  const root = join(parent, "repo");
  await mkdir(root, { recursive: true, mode: 0o700 });
  await writeSource(root, "tools/check-u5-architecture.mjs", analyzerSource);
  await writeSource(root, "internal/reduce/reduce.go", cleanReduce);
  await writeSource(
    root,
    "internal/reduce/authority_literal_test.go",
    "package reduce\n\nconst testOnlyGradeLiteral = \"ONE_MINIMAL_UNDER\"\n",
  );
  await writeSource(root, "internal/store/store.go", cleanStore);
  await writeSource(root, "internal/reduction/reduction.go", cleanReduction);
  return { parent, root };
}

function runAnalyzer(root, environment = {}) {
  return spawnSync(process.execPath, [resolve(root, "tools/check-u5-architecture.mjs")], {
    cwd: root,
    encoding: "utf8",
    env: { ...process.env, NO_COLOR: "1", TZ: "UTC", ...environment },
    timeout: 30_000,
    maxBuffer: 512 * 1024,
  });
}

function analyzerOutput(result) {
  return `${result.stdout ?? ""}${result.stderr ?? ""}`;
}

function expectFailure(name, result, expectedCode) {
  const output = analyzerOutput(result);
  if (result.error || result.signal || result.status === 0 || !output.includes(expectedCode)) {
    throw new SelfTestError(
      "U5_ARCHITECTURE_SELFTEST_FALSE_NEGATIVE",
      `${name}: status=${result.status} signal=${result.signal} error=${result.error?.message ?? ""} expected=${expectedCode}\n${output}`,
    );
  }
}

function expectPass(name, result) {
  const output = analyzerOutput(result);
  if (result.error || result.signal || result.status !== 0 || !result.stdout.includes("U5 architecture boundary OK")) {
    throw new SelfTestError(
      "U5_ARCHITECTURE_SELFTEST_FALSE_POSITIVE",
      `${name}: status=${result.status} signal=${result.signal} error=${result.error?.message ?? ""}\n${output}`,
    );
  }
}

async function withFixture(analyzerSource, name, mutate, expectedCode, environment = {}) {
  const fixture = await createFixture(analyzerSource);
  try {
    await mutate(fixture.root);
    expectFailure(name, runAnalyzer(fixture.root, environment), expectedCode);
  } finally {
    await rm(fixture.parent, { recursive: true, force: true });
  }
}

function reduceWithImport(importPath) {
  return `package reduce

import (
	"${modulePrefix}internal/compare"
	"${importPath}"
)

func assess(baseline, observed compare.CandidateOutcomeMap) compare.PreservationAssessment {
	return compare.AssessPreservation(baseline, observed)
}
`;
}

function storeWithImport(importPath) {
  return `package store

import (
	"${modulePrefix}internal/reduce"
	"${importPath}"
)

type SweepCompletionAuthority struct { token string }
`;
}

async function runCases(analyzerSource) {
  const clean = await createFixture(analyzerSource);
  try {
    expectPass("clean fixture", runAnalyzer(clean.root));
  } finally {
    await rm(clean.parent, { recursive: true, force: true });
  }

  for (const owner of ["gitobj", "world", "adapters/cli", "store", "server", "reduction"]) {
    await withFixture(
      analyzerSource,
      `reduce imports ${owner}`,
      (root) => writeSource(root, "internal/reduce/reduce.go", reduceWithImport(`${modulePrefix}internal/${owner}`)),
      "U5_REDUCE_FORBIDDEN_IMPORT",
    );
  }

  for (const owner of ["adapters/http", "world"]) {
    await withFixture(
      analyzerSource,
      `store imports ${owner}`,
      (root) => writeSource(root, "internal/store/store.go", storeWithImport(`${modulePrefix}internal/${owner}`)),
      "U5_STORE_FORBIDDEN_IMPORT",
    );
  }

  await withFixture(
    analyzerSource,
    "store omits reduce",
    (root) => writeSource(root, "internal/store/store.go", "package store\ntype SweepCompletionAuthority struct { token string }\n"),
    "U5_STORE_REDUCE_IMPORT_REQUIRED",
  );
  await withFixture(
    analyzerSource,
    "reduction omits reduce",
    (root) => writeSource(
      root,
      "internal/reduction/reduction.go",
      `package reduction\nimport "${modulePrefix}internal/store"\nconst grade = "ONE_MINIMAL_UNDER"\n`,
    ),
    "U5_REDUCTION_REDUCE_IMPORT_REQUIRED",
  );
  await withFixture(
    analyzerSource,
    "reduction omits store",
    (root) => writeSource(
      root,
      "internal/reduction/reduction.go",
      `package reduction\nimport "${modulePrefix}internal/reduce"\nconst grade = "ONE_MINIMAL_UNDER"\n`,
    ),
    "U5_REDUCTION_STORE_IMPORT_REQUIRED",
  );
  await withFixture(
    analyzerSource,
    "public authority constructor",
    (root) => writeSource(
      root,
      "internal/store/store.go",
      `${cleanStore}\nfunc NewForgedSweepCompletionAuthority() SweepCompletionAuthority { return SweepCompletionAuthority{} }\n`,
    ),
    "U5_STORE_AUTHORITY_PUBLIC_CONSTRUCTOR",
  );
  await withFixture(
    analyzerSource,
    "exported authority field",
    (root) => writeSource(root, "internal/store/store.go", cleanStore.replace("draft reduce.SweepCompletionDraft", "Draft reduce.SweepCompletionDraft")),
    "U5_STORE_AUTHORITY_EXPORTED_FIELD",
  );
  await withFixture(
    analyzerSource,
    "authority type missing",
    (root) => writeSource(root, "internal/store/store.go", `package store\nimport "${modulePrefix}internal/reduce"\nvar _ reduce.SweepCompletionDraft\n`),
    "U5_STORE_AUTHORITY_DECLARATION_COUNT",
  );
  await withFixture(
    analyzerSource,
    "grade literal in reduce",
    (root) => writeSource(root, "internal/reduce/reduce.go", `${cleanReduce}\nconst forgedGrade = "ONE_MINIMAL_UNDER"\n`),
    "U5_ONE_MINIMAL_AUTHORITY_OUTSIDE_REDUCTION",
  );
  await withFixture(
    analyzerSource,
    "grade constructor in store",
    (root) => writeSource(root, "internal/store/store.go", `${cleanStore}\nfunc buildOneMinimalUnder() {}\n`),
    "U5_ONE_MINIMAL_AUTHORITY_OUTSIDE_REDUCTION",
  );
  await withFixture(
    analyzerSource,
    "outward grade authority missing",
    (root) => writeSource(
      root,
      "internal/reduction/reduction.go",
      `package reduction\nimport (\n "${modulePrefix}internal/reduce"\n "${modulePrefix}internal/store"\n)\nvar _, _ reduce.SweepCompletionDraft = reduce.SweepCompletionDraft{}, reduce.SweepCompletionDraft{}\nvar _ store.SweepCompletionAuthority\n`,
    ),
    "U5_REDUCTION_ONE_MINIMAL_AUTHORITY_REQUIRED",
  );
  await withFixture(
    analyzerSource,
    "naked preservation digest decision",
    (root) => writeSource(
      root,
      "internal/reduce/reduce.go",
      `${cleanReduce}\ntype decision string\nconst (\n Preserves decision = "PRESERVES"\n Changes decision = "CHANGES"\n)\nfunc decide(left, right compare.PreservationMapDigest) decision {\n if left == right { return Preserves }\n return Changes\n}\n`,
    ),
    "U5_NAKED_PRESERVATION_DIGEST_EQUALITY",
  );
  await withFixture(
    analyzerSource,
    "display groups authority",
    (root) => writeSource(root, "internal/reduce/reduce.go", `${cleanReduce}\nfunc shape(m compare.CandidateOutcomeMap) { _ = m.DisplayGroups() }\n`),
    "U5_DISPLAY_GROUPS_AS_AUTHORITY",
  );
  await withFixture(
    analyzerSource,
    "partition shape authority",
    (root) => writeSource(root, "internal/reduction/reduction.go", `${cleanReduction}\nfunc partitionShape() string { return "1:2" }\n`),
    "U5_PARTITION_SHAPE_AS_AUTHORITY",
  );
  await withFixture(
    analyzerSource,
    "missing comparability authority call",
    (root) => writeSource(
      root,
      "internal/reduce/reduce.go",
      `package reduce\nimport "${modulePrefix}internal/compare"\nvar _ compare.CandidateOutcomeMap\n`,
    ),
    "U5_REDUCE_ASSESS_PRESERVATION_REQUIRED",
  );
  await withFixture(
    analyzerSource,
    "cluster ordinal persistence",
    (root) => writeSource(root, "internal/store/store.go", cleanStore.replace("draft reduce.SweepCompletionDraft", "draft reduce.SweepCompletionDraft\n\tclusterOrdinal int")),
    "U5_CLUSTER_ORDINAL_PERSISTENCE_IDENTITY",
  );
  await withFixture(
    analyzerSource,
    "unknown manifest extension",
    (root) => writeSource(root, "internal/store/README.md", "unreviewed\n"),
    "U5_MANIFEST_UNKNOWN_EXTENSION",
  );
  await withFixture(
    analyzerSource,
    "package declaration mismatch",
    (root) => writeSource(root, "internal/store/store.go", "package reduction\n"),
    "U5_MANIFEST_PACKAGE_MISMATCH",
  );

  await withFixture(
    analyzerSource,
    "manifest symlink",
    async (root) => {
      const target = resolve(root, "outside.go");
      await writeFile(target, "package outside\n", { mode: 0o600 });
      await symlink(target, resolve(root, "internal/reduce/linked.go"));
    },
    "U5_MANIFEST_SYMLINK",
  );

  const socketFixture = await createFixture(analyzerSource);
  // Darwin's Unix-domain socket pathname ceiling is much shorter than the
  // hermetic verification TMPDIR used by this repository. Create the socket
  // through a short, private alias while leaving the inode inside the exact
  // fixture tree that the analyzer scans.
  const socketAliasParent = await mkdtemp(join(process.platform === "win32" ? tmpdir() : "/tmp", "cs-u5-socket-"));
  const socketAlias = join(socketAliasParent, "root");
  await symlink(socketFixture.root, socketAlias, "dir");
  const server = createServer();
  const socketPath = resolve(socketAlias, "internal/store/socket.go");
  let listening = false;
  try {
    await new Promise((resolveListen, rejectListen) => {
      server.once("error", rejectListen);
      server.listen(socketPath, resolveListen);
    });
    listening = true;
    expectFailure("manifest nonregular socket", runAnalyzer(socketFixture.root), "U5_MANIFEST_NONREGULAR");
  } finally {
    if (listening) await new Promise((resolveClose) => server.close(resolveClose));
    await rm(socketAliasParent, { recursive: true, force: true });
    await rm(socketFixture.parent, { recursive: true, force: true });
  }

  const barrier = "  // U5_SELFTEST_MANIFEST_BARRIER";
  const tamperedAnalyzer = analyzerSource.replace(
    barrier,
    `  await (await import("node:fs/promises")).appendFile(resolve(root, "internal/reduce/reduce.go"), "\\n// concurrent manifest tamper\\n");\n${barrier}`,
  );
  if (tamperedAnalyzer === analyzerSource) {
    throw new SelfTestError("U5_ARCHITECTURE_SELFTEST_SETUP", "manifest barrier marker is missing");
  }
  await withFixture(
    tamperedAnalyzer,
    "manifest changed during scan",
    async () => {},
    "U5_MANIFEST_CHANGED_DURING_SCAN",
  );

  await withFixture(
    analyzerSource,
    "manifest tool failure",
    async () => {},
    "U5_MANIFEST_TOOL_FAILED",
    { COUNTERSHAPE_U5_ARCHITECTURE_FORCE_TOOL_FAILURE: "1" },
  );
}

async function main() {
  if (process.argv.length !== 2) {
    throw new SelfTestError("U5_ARCHITECTURE_SELFTEST_ARGUMENTS", "no arguments are accepted");
  }
  const analyzerSource = await readFile(analyzerPath, "utf8");
  await runCases(analyzerSource);
  process.stdout.write("U5 architecture self-test passed: clean fixture plus every hostile topology, authority, manifest, and tool case.\n");
}

main().catch((error) => {
  process.stderr.write(`${error.stack ?? error}\n`);
  process.exitCode = 1;
});
