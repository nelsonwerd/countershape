import assert from "node:assert/strict";
import {
  appendFile,
  chmod,
  copyFile,
  lstat,
  mkdir,
  mkdtemp,
  readFile,
  readdir,
  rm,
  writeFile,
} from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import test from "node:test";

import { resolveAdmittedGoExecutable } from "./mutate-u1.mjs";
import {
  MutationGateError,
  NamedTestOutcome,
  REQUIRED_U2_MUTANT_IDS,
  REVIEWED_U2_MUTANT_CONTRACT,
  U2_MUTANTS,
  U2_SANDBOX_FILE_ALLOWLIST,
  applyExactMutation,
  assertExactU2MutantDelta,
  assertReviewedU2Anchors,
  assertU2ManifestMatchesTree,
  assertU2MutantDefinitionSet,
  assertU2SourceAndSeedUnchanged,
  cleanupU2SeedSnapshot,
  classifyNamedGoTest,
  copyRegularAllowlist,
  createU2SeedSnapshot,
  minimalU2GoEnvironment,
  mutateRegularFileNoFollow,
  readAdmittedDarwinCCompilerFacts,
  readAdmittedDarwinCGOFacts,
  resolveAdmittedDarwinCCompiler,
  revalidateAdmittedDarwinCCompiler,
  runNamedU2GoTest,
  runU2Mutant,
} from "./mutate-u2.mjs";

async function withTemporaryRoot(prefix, callback) {
  const root = await mkdtemp(join(tmpdir(), prefix));
  await chmod(root, 0o700);
  try {
    return await callback(root);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
}

function codeIs(expected) {
  return (error) => error instanceof MutationGateError && error.code === expected;
}

function event(action, testName, packageName = "example") {
  const payload = { Time: "2000-01-01T00:00:00Z", Action: action, Package: packageName };
  if (testName !== undefined) payload.Test = testName;
  return JSON.stringify(payload);
}

function passingEvents(testName, packageName = "example") {
  return [event("run", testName, packageName), event("pass", testName, packageName), event("pass", undefined, packageName)].join("\n");
}

function failingEvents(testName, packageName = "example") {
  return [event("run", testName, packageName), event("fail", testName, packageName), event("fail", undefined, packageName)].join("\n");
}

function classification(outcome, detail = outcome) {
  return { classification: { outcome, detail }, output: "" };
}

async function writeSyntheticSource(root) {
  const allowlist = [
    "go.mod",
    "internal/gitobj/guard.go",
    "testkit/processfixture/main.go",
  ];
  const scopes = ["internal/gitobj", "testkit/processfixture"];
  await mkdir(join(root, "internal", "gitobj"), { recursive: true });
  await mkdir(join(root, "testkit", "processfixture"), { recursive: true });
  await writeFile(join(root, "go.mod"), "module example.invalid/u2\n", "utf8");
  await writeFile(join(root, "internal", "gitobj", "guard.go"), "package gitobj\nconst guarded = true // SYNTHETIC_U2_ANCHOR\n", "utf8");
  await writeFile(join(root, "testkit", "processfixture", "main.go"), "package main\nfunc main() {}\n", "utf8");
  return { allowlist, scopes };
}

test("U2 required IDs and reviewed tuples are an immutable exact set of twenty-five", () => {
  assert.equal(REQUIRED_U2_MUTANT_IDS.length, 25);
  assert.equal(new Set(REQUIRED_U2_MUTANT_IDS).size, 25);
  assert.equal(Object.isFrozen(REVIEWED_U2_MUTANT_CONTRACT), true);
  assert.equal(REVIEWED_U2_MUTANT_CONTRACT.every((tuple) => Object.isFrozen(tuple)), true);
  assert.doesNotThrow(() => assertU2MutantDefinitionSet());
  assert.throws(() => assertU2MutantDefinitionSet(U2_MUTANTS.slice(0, 24)), codeIs("MUTANT_SET_MISMATCH"));

  const duplicate = [...U2_MUTANTS.slice(0, 24), { ...U2_MUTANTS[23] }];
  assert.throws(() => assertU2MutantDefinitionSet(duplicate), codeIs("DUPLICATE_MUTANT_ID"));
  const reordered = [...U2_MUTANTS];
  [reordered[0], reordered[1]] = [reordered[1], reordered[0]];
  assert.throws(() => assertU2MutantDefinitionSet(reordered), codeIs("MUTANT_SET_MISMATCH"));
  const tupleTampered = U2_MUTANTS.map((mutant) => ({ ...mutant }));
  tupleTampered[0].replace += "\n// unreviewed weakening";
  assert.throws(() => assertU2MutantDefinitionSet(tupleTampered), codeIs("MUTANT_CONTRACT_MISMATCH"));
  const extraField = U2_MUTANTS.map((mutant) => ({ ...mutant }));
  extraField[0].unreviewed = true;
  assert.throws(() => assertU2MutantDefinitionSet(extraField), codeIs("MUTANT_CONTRACT_MISMATCH"));
});

test("the reviewed U2 tuples cover exactly the current twenty-five source anchors", async () => {
  await assert.doesNotReject(assertReviewedU2Anchors());
});

test("each mutation is one and only one exact replacement", () => {
  const mutant = { id: "synthetic", file: "guard.go", find: "ANCHOR", replace: "MUTATED" };
  assert.equal(applyExactMutation("before ANCHOR after", mutant), "before MUTATED after");
  assert.throws(() => applyExactMutation("no marker", mutant), codeIs("MUTATION_ANCHOR_COUNT"));
  assert.throws(() => applyExactMutation("ANCHOR then ANCHOR", mutant), codeIs("MUTATION_ANCHOR_COUNT"));
});

test("the U2 allowlist is positive, secret-free, and exact over all admitted source scopes", async () => {
  for (const forbidden of [".env", ".git/config", ".didrun/claims.json", "credentials.json", "CLAUDE.md"]) {
    assert.equal(U2_SANDBOX_FILE_ALLOWLIST.includes(forbidden), false, `${forbidden} entered the U2 execution manifest`);
  }
  assert.equal(U2_SANDBOX_FILE_ALLOWLIST.some((file) => file.startsWith(".git/") || file.startsWith(".didrun/")), false);

  await withTemporaryRoot("countershape-u2-manifest-", async (root) => {
    const { allowlist, scopes } = await writeSyntheticSource(root);
    await assert.doesNotReject(assertU2ManifestMatchesTree(root, allowlist, scopes));
    await writeFile(join(root, "internal", "gitobj", "unreviewed.go"), "package gitobj\n", "utf8");
    await assert.rejects(assertU2ManifestMatchesTree(root, allowlist, scopes), codeIs("MANIFEST_TREE_MISMATCH"));
    await rm(join(root, "internal", "gitobj", "unreviewed.go"));
    await writeFile(join(root, "internal", "gitobj", "secret.txt"), "not admitted\n", "utf8");
    await assert.rejects(assertU2ManifestMatchesTree(root, allowlist, scopes), codeIs("MANIFEST_TREE_UNSUPPORTED_FILE"));
  });
});

test("relocated process mechanics remain inside the positive U2 execution manifest", () => {
  const mechanicsFiles = [
    "internal/processmechanics/capture.go",
    "internal/processmechanics/capture_darwin_test.go",
    "internal/processmechanics/process.go",
    "internal/processmechanics/process_darwin.go",
    "internal/processmechanics/process_darwin_test.go",
    "internal/processmechanics/process_unsupported.go",
  ];
  for (const file of mechanicsFiles) {
    assert.equal(U2_SANDBOX_FILE_ALLOWLIST.includes(file), true, `${file} is missing from the positive U2 manifest`);
  }
  const relocated = new Set([
    "spawn-through-shell",
    "share-stdout-limit-with-stderr",
    "map-overflow-to-success",
    "signal-direct-pid",
    "remove-kill-escalation",
    "report-drain-timeout-complete",
    "skip-final-group-probe",
    "late-control-overwrites-earlier-terminal",
    "claim-process-escape-containment",
  ]);
  for (const tuple of REVIEWED_U2_MUTANT_CONTRACT.filter(({ id }) => relocated.has(id))) {
    assert.match(tuple.file, /^internal\/processmechanics\//u, `${tuple.id} still targets the former world owner`);
  }
  assert.equal(U2_SANDBOX_FILE_ALLOWLIST.filter((file) => file.startsWith("internal/processmechanics/")).length, mechanicsFiles.length);
});

test("U2 copies contain only regular allowlisted files with private modes", async () => {
  await withTemporaryRoot("countershape-u2-copy-", async (root) => {
    const source = join(root, "source");
    const destination = join(root, "destination");
    await mkdir(source);
    await writeFile(join(source, "go.mod"), "module example.invalid/u2\n", "utf8");
    await writeFile(join(source, ".env"), "TOKEN=must-not-copy\n", "utf8");
    await copyRegularAllowlist(source, destination, ["go.mod"]);
    assert.deepEqual(await readdir(destination), ["go.mod"]);
    assert.equal((await lstat(destination)).mode & 0o777, 0o700);
    assert.equal((await lstat(join(destination, "go.mod"))).mode & 0o777, 0o600);
  });
});

test("immutable seed revalidation catches both admitted-source and seed changes", async () => {
  await withTemporaryRoot("countershape-u2-seed-selftest-", async (root) => {
    const source = join(root, "source");
    await mkdir(source);
    const { allowlist, scopes } = await writeSyntheticSource(source);
    const seed = await createU2SeedSnapshot({ sourceRoot: source, allowlist, scopeRoots: scopes });
    try {
      await assert.doesNotReject(assertU2SourceAndSeedUnchanged(seed));
      const sourceModule = await readFile(join(source, "go.mod"), "utf8");
      await writeFile(join(source, "go.mod"), "module changed.invalid/source\n", "utf8");
      await assert.rejects(assertU2SourceAndSeedUnchanged(seed), codeIs("ADMITTED_SOURCE_CHANGED"));
      await writeFile(join(source, "go.mod"), sourceModule, "utf8");
      await writeFile(join(seed.root, "go.mod"), "module changed.invalid/seed\n", "utf8");
      await assert.rejects(assertU2SourceAndSeedUnchanged(seed), codeIs("SEED_SNAPSHOT_CHANGED"));
    } finally {
      await cleanupU2SeedSnapshot(seed);
    }
  });
});

test("mutant delta must be the exact reviewed bytes in exactly one file", async () => {
  await withTemporaryRoot("countershape-u2-delta-selftest-", async (root) => {
    const source = join(root, "source");
    await mkdir(source);
    const { allowlist, scopes } = await writeSyntheticSource(source);
    const seed = await createU2SeedSnapshot({ sourceRoot: source, allowlist, scopeRoots: scopes });
    const mutant = {
      id: "synthetic",
      file: "internal/gitobj/guard.go",
      find: "const guarded = true // SYNTHETIC_U2_ANCHOR",
      replace: "const guarded = false // SYNTHETIC_U2_ANCHOR",
    };
    try {
      const sandbox = join(root, "sandbox");
      await copyRegularAllowlist(seed.root, sandbox, allowlist);
      await mutateRegularFileNoFollow(sandbox, mutant);
      await assert.doesNotReject(assertExactU2MutantDelta(sandbox, seed, mutant));
      await writeFile(join(sandbox, "go.mod"), "module changed.invalid/extra\n", "utf8");
      await assert.rejects(assertExactU2MutantDelta(sandbox, seed, mutant), codeIs("MUTANT_DELTA_SCOPE"));
    } finally {
      await cleanupU2SeedSnapshot(seed);
    }
  });
});

test("named-test classification accepts only one exact top-level test and coherent package exit", () => {
  const testName = "TestExactTarget";
  assert.equal(classifyNamedGoTest({ status: 0, stdout: passingEvents(testName), testName, expectedPackage: "example" }).outcome, NamedTestOutcome.Pass);
  assert.equal(classifyNamedGoTest({ status: 1, stdout: failingEvents(testName), testName, expectedPackage: "example" }).outcome, NamedTestOutcome.Failure);
  assert.equal(classifyNamedGoTest({ status: 0, stdout: event("pass", undefined), testName, expectedPackage: "example" }).outcome, NamedTestOutcome.Missing);
  assert.equal(classifyNamedGoTest({
    status: 0,
    stdout: [event("run", testName), event("skip", testName), event("pass", undefined)].join("\n"),
    testName,
    expectedPackage: "example",
  }).outcome, NamedTestOutcome.Infrastructure);
  assert.equal(classifyNamedGoTest({
    status: 0,
    stdout: [event("run", testName), event("run", `${testName}/subcase`), event("pass", `${testName}/subcase`), event("pass", testName), event("pass", undefined)].join("\n"),
    testName,
    expectedPackage: "example",
  }).outcome, NamedTestOutcome.Infrastructure);
  assert.equal(classifyNamedGoTest({ status: 0, stdout: passingEvents(testName), stderr: "raw compiler warning", testName, expectedPackage: "example" }).outcome, NamedTestOutcome.Infrastructure);
  assert.equal(classifyNamedGoTest({ status: 0, stdout: passingEvents(testName, "wrong/package"), testName, expectedPackage: "example" }).outcome, NamedTestOutcome.Infrastructure);
  assert.equal(classifyNamedGoTest({ status: 0, stdout: failingEvents(testName), testName, expectedPackage: "example" }).outcome, NamedTestOutcome.Infrastructure);
});

test("A/B/A execution requires three fresh roots, two passing controls, and one exact mutant failure", async () => {
  const phases = [];
  let run = 0;
  const result = await runU2Mutant(U2_MUTANTS[0], {}, {
    async prepareExperiment({ phase }) {
      phases.push(phase);
      return { sandbox: `/fresh/${phase}`, sandboxParent: `/fresh/${phase}` };
    },
    async cleanupExperiment() {},
    async applyMutation() {},
    async runNamedTest() {
      return [
        classification(NamedTestOutcome.Pass),
        classification(NamedTestOutcome.Failure),
        classification(NamedTestOutcome.Pass),
      ][run++];
    },
  });
  assert.match(result, /^KILLED enable-replacement-refs/u);
  assert.deepEqual(phases, ["baseline", "mutant", "post-control"]);
  assert.equal(run, 3);
});

test("A/B/A rejects root reuse, survivor green, mutant skip, and failing post-control", async () => {
  const baseDependencies = {
    async cleanupExperiment() {},
    async applyMutation() {},
  };
  let calls = 0;
  await assert.rejects(runU2Mutant(U2_MUTANTS[0], {}, {
    ...baseDependencies,
    async prepareExperiment() { return { sandbox: "/same", sandboxParent: "/same" }; },
    async runNamedTest() { return classification(NamedTestOutcome.Pass); },
  }), codeIs("EXPERIMENT_ROOT_REUSED"));

  calls = 0;
  await assert.rejects(runU2Mutant(U2_MUTANTS[0], {}, {
    ...baseDependencies,
    async prepareExperiment({ phase }) { return { sandbox: `/survivor/${phase}`, sandboxParent: `/survivor/${phase}` }; },
    async runNamedTest() { calls += 1; return classification(NamedTestOutcome.Pass); },
  }), codeIs("MUTANT_SURVIVED"));
  assert.equal(calls, 3, "post-control must execute even for a surviving mutant");

  calls = 0;
  await assert.rejects(runU2Mutant(U2_MUTANTS[0], {}, {
    ...baseDependencies,
    async prepareExperiment({ phase }) { return { sandbox: `/skip/${phase}`, sandboxParent: `/skip/${phase}` }; },
    async runNamedTest() {
      const outcomes = [NamedTestOutcome.Pass, NamedTestOutcome.Infrastructure, NamedTestOutcome.Pass];
      return classification(outcomes[calls++]);
    },
  }), codeIs("MUTANT_INFRASTRUCTURE_FAILURE"));
  assert.equal(calls, 3);

  calls = 0;
  await assert.rejects(runU2Mutant(U2_MUTANTS[0], {}, {
    ...baseDependencies,
    async prepareExperiment({ phase }) { return { sandbox: `/post/${phase}`, sandboxParent: `/post/${phase}` }; },
    async runNamedTest() {
      const outcomes = [NamedTestOutcome.Pass, NamedTestOutcome.Failure, NamedTestOutcome.Missing];
      return classification(outcomes[calls++]);
    },
  }), codeIs("BASELINE_TEST_NOT_PASSING"));
  assert.equal(calls, 3);
});

test("Darwin compiler admission is explicit, native, byte-pinned, and revalidated", async () => {
  await assert.rejects(resolveAdmittedDarwinCCompiler({}), codeIs("CC_EXECUTABLE_NOT_DECLARED"));
  await assert.rejects(resolveAdmittedDarwinCCompiler({ COUNTERSHAPE_CC: "relative/clang" }), codeIs("CC_EXECUTABLE_NOT_ABSOLUTE"));
  await withTemporaryRoot("countershape-u2-cc-", async (root) => {
    const script = join(root, "clang-script");
    await writeFile(script, "#!/bin/sh\nexit 0\n", "utf8");
    await chmod(script, 0o700);
    await assert.rejects(resolveAdmittedDarwinCCompiler({ COUNTERSHAPE_CC: script }), codeIs("CC_EXECUTABLE_BINARY_FORMAT"));

    const compilerCopy = join(root, "clang-copy");
    await copyFile("/usr/bin/true", compilerCopy);
    await chmod(compilerCopy, 0o700);
    const admitted = await resolveAdmittedDarwinCCompiler({ COUNTERSHAPE_CC: compilerCopy });
    assert.match(admitted.sha256, /^[0-9a-f]{64}$/u);
    await appendFile(compilerCopy, Buffer.from([0]));
    assert.throws(() => revalidateAdmittedDarwinCCompiler(admitted), codeIs("CC_EXECUTABLE_CHANGED"));
  });
});

test("compiler probes and CGO facts are canonical and bound to the admitted paths", async () => {
  const compiler = await resolveAdmittedDarwinCCompiler({ COUNTERSHAPE_CC: "/usr/bin/true" });
  const goExecutable = await resolveAdmittedGoExecutable({ COUNTERSHAPE_GO: process.execPath });
  const architecture = process.arch === "x64" ? "x86_64" : process.arch;
  let compilerCalls = 0;
  const compilerFacts = readAdmittedDarwinCCompilerFacts(compiler, (_executable, args) => {
    compilerCalls += 1;
    if (args[0] === "--version") {
      return { status: 0, signal: null, error: null, stdout: "Apple clang version 99.0.0\n", stderr: "" };
    }
    return { status: 0, signal: null, error: null, stdout: `${architecture}-apple-darwin99.0.0\n`, stderr: "" };
  });
  assert.equal(compilerCalls, 2);
  assert.match(compilerFacts.digest, /^sha256:[0-9a-f]{64}$/u);

  const partial = { executable: goExecutable, compiler };
  const goArchitecture = process.arch === "x64" ? "amd64" : process.arch;
  const cgo = readAdmittedDarwinCGOFacts(partial, () => ({
    status: 0,
    signal: null,
    error: null,
    stdout: JSON.stringify({ GOOS: "darwin", GOARCH: goArchitecture, CGO_ENABLED: "1", CC: compiler.path }),
    stderr: "",
  }));
  assert.equal(cgo.compilerPath, compiler.path);
  assert.throws(() => readAdmittedDarwinCGOFacts(partial, () => ({
    status: 0,
    signal: null,
    error: null,
    stdout: JSON.stringify({ GOOS: "darwin", GOARCH: goArchitecture, CGO_ENABLED: "0", CC: "cc" }),
    stderr: "",
  })), codeIs("CGO_TOOLCHAIN_INSPECTION_FAILED"));
});

test("U2 Go environment enables CGO without forwarding ambient credentials or PATH", async () => {
  const compiler = await resolveAdmittedDarwinCCompiler({ COUNTERSHAPE_CC: "/usr/bin/true" });
  const executable = await resolveAdmittedGoExecutable({ COUNTERSHAPE_GO: process.execPath });
  const environment = minimalU2GoEnvironment("/private/u2-cache", { executable, compiler });
  assert.equal(environment.CGO_ENABLED, "1");
  assert.equal(environment.CC, compiler.path);
  assert.equal(environment.CXX, compiler.path);
  assert.deepEqual(new Set(environment.PATH.split(":")), new Set([dirname(executable.path), dirname(compiler.path)]));
  assert.equal(environment.GOENV, "off");
  assert.equal(environment.GOWORK, "off");
  assert.equal(environment.GOTOOLCHAIN, "local");
  assert.equal(environment.GOPROXY, "off");
  assert.equal("API_TOKEN" in environment, false);
  assert.equal("AWS_SECRET_ACCESS_KEY" in environment, false);
  assert.equal("SSH_AUTH_SOCK" in environment, false);
  assert.equal("DYLD_INSERT_LIBRARIES" in environment, false);
});

test("named U2 invocation uses the admitted Go path and one exact anchored test", async () => {
  const compiler = await resolveAdmittedDarwinCCompiler({ COUNTERSHAPE_CC: "/usr/bin/true" });
  const executable = await resolveAdmittedGoExecutable({ COUNTERSHAPE_GO: process.execPath });
  const toolchain = { executable, compiler };
  let observed;
  const mutant = U2_MUTANTS[0];
  const result = runNamedU2GoTest({
    sandbox: "/private/fresh-u2-root",
    mutant,
    environment: minimalU2GoEnvironment("/private/u2-cache", toolchain),
    toolchain,
    spawn(spawned, args, options) {
      observed = { spawned, args, options };
      return {
        status: 0,
        signal: null,
        error: null,
        stdout: passingEvents(mutant.testName, "github.com/nelsonwerd/countershape/internal/gitobj"),
        stderr: "",
      };
    },
  });
  assert.equal(result.classification.outcome, NamedTestOutcome.Pass);
  assert.equal(observed.spawned, executable.path);
  assert.deepEqual(observed.args, ["test", "-json", "./internal/gitobj", "-run", `^${mutant.testName}$`, "-count=1"]);
  assert.equal(observed.options.cwd, "/private/fresh-u2-root");
});
