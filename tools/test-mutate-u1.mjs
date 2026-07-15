import assert from "node:assert/strict";
import { chmod, lstat, mkdir, mkdtemp, readFile, readdir, realpath, rm, symlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import test from "node:test";

import {
  MUTANTS,
  MutationGateError,
  NamedTestOutcome,
  REQUIRED_MUTANT_IDS,
  REVIEWED_MUTANT_CONTRACT,
  SANDBOX_FILE_ALLOWLIST,
  applyExactMutation,
  assertExactMutantDelta,
  assertManifestMatchesTree,
  assertMutantDefinitionSet,
  classifyNamedGoTest,
  cleanupSeedSnapshot,
  copyRegularAllowlist,
  createSeedSnapshot,
  minimalGoEnvironment,
  mutateRegularFileNoFollow,
  readAdmittedGoBuildFacts,
  readAdmittedGoVersion,
  requireBaselinePass,
  resolveAdmittedGoExecutable,
  runMutant,
  runNamedGoTest,
} from "./mutate-u1.mjs";

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
  return [
    event("run", testName, packageName),
    event("pass", testName, packageName),
    event("pass", undefined, packageName),
  ].join("\n");
}

function failingEvents(testName, packageName = "example") {
  return [
    event("run", testName, packageName),
    event("fail", testName, packageName),
    event("fail", undefined, packageName),
  ].join("\n");
}

test("required mutant IDs are exactly eighteen and cannot shrink or duplicate", () => {
  assert.equal(REQUIRED_MUTANT_IDS.length, 18);
  assert.equal(new Set(REQUIRED_MUTANT_IDS).size, 18);
  assert.doesNotThrow(() => assertMutantDefinitionSet());
  assert.throws(() => assertMutantDefinitionSet(MUTANTS.slice(0, 17)), codeIs("MUTANT_SET_MISMATCH"));
  const duplicate = [...MUTANTS.slice(0, 17), { ...MUTANTS[16] }];
  assert.throws(() => assertMutantDefinitionSet(duplicate), codeIs("DUPLICATE_MUTANT_ID"));
  const replaced = [...MUTANTS];
  replaced[0] = { ...replaced[0], id: "not-a-required-mutant" };
  assert.throws(() => assertMutantDefinitionSet(replaced), codeIs("MUTANT_SET_MISMATCH"));

  assert.equal(Object.isFrozen(REVIEWED_MUTANT_CONTRACT), true);
  assert.equal(REVIEWED_MUTANT_CONTRACT.every((tuple) => Object.isFrozen(tuple)), true);
  const tupleTampered = MUTANTS.map((mutant) => ({ ...mutant }));
  tupleTampered[0].replace = `${tupleTampered[0].replace}\n// weakened under the same ID`;
  assert.throws(
    () => assertMutantDefinitionSet(tupleTampered),
    codeIs("MUTANT_CONTRACT_MISMATCH"),
  );
  const extraField = MUTANTS.map((mutant) => ({ ...mutant }));
  extraField[0].unreviewed = true;
  assert.throws(
    () => assertMutantDefinitionSet(extraField),
    codeIs("MUTANT_CONTRACT_MISMATCH"),
  );
});

test("exact mutation requires one and only one anchor", () => {
  const mutant = { id: "synthetic", file: "source", find: "ANCHOR", replace: "MUTATED" };
  assert.equal(applyExactMutation("before ANCHOR after", mutant), "before MUTATED after");
  assert.throws(() => applyExactMutation("no marker", mutant), codeIs("MUTATION_ANCHOR_COUNT"));
  assert.throws(() => applyExactMutation("ANCHOR and ANCHOR", mutant), codeIs("MUTATION_ANCHOR_COUNT"));
});

test("baseline and mutant copies derive from one manifest and the mutant changes exactly one reviewed file", async () => {
  const seed = await createSeedSnapshot();
  try {
    await withTemporaryRoot("countershape-mutator-delta-", async (root) => {
      const sandbox = join(root, "repo");
      await copyRegularAllowlist(seed.root, sandbox);
      await mutateRegularFileNoFollow(sandbox, MUTANTS[0]);
      await assert.doesNotReject(assertExactMutantDelta(sandbox, seed, MUTANTS[0]));
      await writeFile(join(sandbox, "go.mod"), "module changed.invalid/example\n", "utf8");
      await assert.rejects(
        assertExactMutantDelta(sandbox, seed, MUTANTS[0]),
        codeIs("MUTANT_DELTA_SCOPE"),
      );
    });
  } finally {
    await cleanupSeedSnapshot(seed);
  }
});

test("allowlist copy excludes top-level and nested secrets and uses private modes", async () => {
  for (const forbidden of [".env", ".git/config", ".didrun/claims.json", "credentials.json"]) {
    assert.equal(SANDBOX_FILE_ALLOWLIST.includes(forbidden), false, `${forbidden} entered the production manifest`);
  }
  assert.equal(
    SANDBOX_FILE_ALLOWLIST.some((file) => file.startsWith(".git/") || file.startsWith(".didrun/")),
    false,
    "VCS or evidence-ledger paths entered the production manifest",
  );
  await withTemporaryRoot("countershape-mutator-copy-", async (root) => {
    const source = join(root, "source");
    const destination = join(root, "destination");
    await mkdir(join(source, ".didrun", "nested"), { recursive: true });
    await writeFile(join(source, "go.mod"), "module example\n", "utf8");
    await writeFile(join(source, ".env"), "API_TOKEN=must-not-copy\n", "utf8");
    await writeFile(join(source, ".didrun", "nested", "ledger"), "secret evidence\n", "utf8");

    await copyRegularAllowlist(source, destination, ["go.mod"]);
    assert.deepEqual(await readdir(destination), ["go.mod"]);
    assert.equal(await readFile(join(destination, "go.mod"), "utf8"), "module example\n");
    assert.equal((await lstat(destination)).mode & 0o777, 0o700);
    assert.equal((await lstat(join(destination, "go.mod"))).mode & 0o777, 0o600);
  });
});

test("manifest is the exact current internal Go tree", async () => {
  await withTemporaryRoot("countershape-mutator-manifest-", async (root) => {
    const source = join(root, "source");
    const internalFile = "internal/canon/value.go";
    const exampleFile = "spec/examples/v1/source-spec.valid.json";
    const allowlist = ["go.mod", internalFile, exampleFile];
    await mkdir(join(source, "internal", "canon"), { recursive: true });
    await mkdir(join(source, "spec", "examples", "v1"), { recursive: true });
    await writeFile(join(source, "go.mod"), "module example\n", "utf8");
    await writeFile(join(source, internalFile), "package canon\n", "utf8");
    await writeFile(join(source, exampleFile), "{}\n", "utf8");

    await assert.doesNotReject(assertManifestMatchesTree(source, allowlist));

    const unlisted = join(source, "internal", "canon", "new.go");
    await writeFile(unlisted, "package canon\n", "utf8");
    await assert.rejects(
      assertManifestMatchesTree(source, allowlist),
      codeIs("MANIFEST_TREE_MISMATCH"),
    );

    await rm(unlisted);
    await rm(join(source, internalFile));
    await assert.rejects(
      assertManifestMatchesTree(source, allowlist),
      codeIs("MANIFEST_TREE_MISMATCH"),
    );
  });
});

test("allowlisted symlinks and nonregular files are rejected", async () => {
  await withTemporaryRoot("countershape-mutator-types-", async (root) => {
    const source = join(root, "source");
    await mkdir(source, { recursive: true });
    await writeFile(join(source, "real.mod"), "module example\n", "utf8");
    await symlink("real.mod", join(source, "go.mod"));
    await assert.rejects(
      copyRegularAllowlist(source, join(root, "symlink-destination"), ["go.mod"]),
      codeIs("SOURCE_SYMLINK"),
    );

    await rm(join(source, "go.mod"));
    await mkdir(join(source, "go.mod"));
    await assert.rejects(
      copyRegularAllowlist(source, join(root, "directory-destination"), ["go.mod"]),
      codeIs("SOURCE_NONREGULAR"),
    );
  });
});

test("an allowlisted path cannot traverse a symlink parent", async () => {
  await withTemporaryRoot("countershape-mutator-parent-", async (root) => {
    const source = join(root, "source");
    const outside = join(root, "outside");
    await mkdir(source);
    await mkdir(outside);
    await writeFile(join(outside, "go.mod"), "module escaped\n", "utf8");
    await symlink(outside, join(source, "internal"));
    await assert.rejects(
      copyRegularAllowlist(source, join(root, "destination"), ["internal/go.mod"]),
      codeIs("SOURCE_SYMLINK_PARENT"),
    );

    await symlink(source, join(root, "linked-source"));
    await assert.rejects(
      copyRegularAllowlist(join(root, "linked-source"), join(root, "root-link-destination"), ["go.mod"]),
      codeIs("SOURCE_ROOT_NONREGULAR"),
    );
  });
});

test("mutation target and parent symlinks cannot redirect an outside write", async () => {
  const synthetic = {
    id: "synthetic",
    file: "internal/target.go",
    find: "ANCHOR",
    replace: "MUTATED",
  };
  await withTemporaryRoot("countershape-mutator-rewrite-", async (root) => {
    const repository = join(root, "repository");
    const outside = join(root, "outside.go");
    await mkdir(join(repository, "internal"), { recursive: true });
    await writeFile(outside, "before ANCHOR after", "utf8");
    await symlink(outside, join(repository, synthetic.file));
    await assert.rejects(
      mutateRegularFileNoFollow(repository, synthetic),
      codeIs("MUTATION_TARGET_SYMLINK"),
    );
    assert.equal(await readFile(outside, "utf8"), "before ANCHOR after");

    const secondRepository = join(root, "second-repository");
    const outsideParent = join(root, "outside-parent");
    await mkdir(secondRepository);
    await mkdir(outsideParent);
    await writeFile(join(outsideParent, "target.go"), "before ANCHOR after", "utf8");
    await symlink(outsideParent, join(secondRepository, "internal"));
    await assert.rejects(
      mutateRegularFileNoFollow(secondRepository, synthetic),
      codeIs("SOURCE_SYMLINK_PARENT"),
    );
    assert.equal(await readFile(join(outsideParent, "target.go"), "utf8"), "before ANCHOR after");
  });
});

test("minimal Go environment does not forward credentials, sockets, or parent PATH", async () => {
  const admission = await resolveAdmittedGoExecutable({ COUNTERSHAPE_GO: process.execPath });
  const environment = minimalGoEnvironment("/private/cache", admission);
  assert.equal(environment.PATH, dirname(await realpath(process.execPath)));
  assert.equal(environment.HOME, "/private/cache/home");
  assert.equal(environment.GOTMPDIR, "/private/cache/go-tmp");
  assert.equal(environment.GOPATH, "/private/cache/go-path");
  assert.equal(environment.GOPROXY, "off");
  assert.equal(environment.GOWORK, "off");
  assert.equal(environment.GOTOOLCHAIN, "local");
  assert.equal(environment.NO_COLOR, "1");
  assert.equal("API_TOKEN" in environment, false);
  assert.equal("AWS_SECRET_ACCESS_KEY" in environment, false);
  assert.equal("SSH_AUTH_SOCK" in environment, false);
});

test("trusted Go admission rejects scripts, ignores PATH, fingerprints bytes, and revalidates every spawn", async () => {
  await withTemporaryRoot("countershape-mutator-go-", async (root) => {
    const poisonDirectory = join(root, "poison");
    await mkdir(poisonDirectory);
    const spoof = join(root, "spoof-go");
    await writeFile(spoof, "#!/bin/sh\nprintf 'go version go9.9 darwin/arm64\\n'\n", "utf8");
    await chmod(spoof, 0o700);
    await assert.rejects(
      resolveAdmittedGoExecutable({ COUNTERSHAPE_GO: spoof, PATH: poisonDirectory }),
      codeIs("GO_EXECUTABLE_BINARY_FORMAT"),
    );

    await writeFile(join(poisonDirectory, "go"), "#!/bin/sh\nexit 99\n", "utf8");
    await chmod(join(poisonDirectory, "go"), 0o700);

    const resolved = await resolveAdmittedGoExecutable({
      COUNTERSHAPE_GO: process.execPath,
      PATH: poisonDirectory,
    });
    assert.equal(resolved.path, await realpath(process.execPath));
    assert.match(resolved.sha256, /^[0-9a-f]{64}$/u);
    assert.equal(resolved.format, process.platform === "darwin" ? "MACH_O" : "ELF");
    await assert.rejects(
      resolveAdmittedGoExecutable({ COUNTERSHAPE_GO: "relative/go", PATH: poisonDirectory }),
      codeIs("GO_EXECUTABLE_NOT_ABSOLUTE"),
    );
    const goos = process.platform === "darwin" ? "darwin" : process.platform;
    const goarch = process.arch === "x64" ? "amd64" : process.arch;
    let versionExecutable = "";
    let versionEnvironment;
    const version = readAdmittedGoVersion(resolved, (executable, args, options) => {
      versionExecutable = executable;
      versionEnvironment = options.env;
      assert.deepEqual(args, ["version"]);
      return {
        status: 0,
        signal: null,
        error: null,
        stdout: `go version go1.test ${goos}/${goarch}\n`,
        stderr: "",
      };
    });
    assert.equal(versionExecutable, await realpath(process.execPath));
    assert.equal(version.line, `go version go1.test ${goos}/${goarch}`);
    assert.equal(versionEnvironment.GOENV, "off");
    assert.equal(versionEnvironment.GOWORK, "off");
    assert.equal(versionEnvironment.GOTOOLCHAIN, "local");

    const build = readAdmittedGoBuildFacts(resolved, () => ({
      status: 0,
      signal: null,
      error: null,
      stdout: `${resolved.path}: go1.test\n\tpath\tcmd/go\n\tbuild\tGOOS=${goos}\n\tbuild\tGOARCH=${goarch}\n`,
      stderr: "",
    }));
    assert.equal(build.goos, goos);
    assert.equal(build.goarch, goarch);

    let spawnedExecutable = "";
    const testName = "TestExactTarget";
    const result = runNamedGoTest({
      sandbox: root,
      mutant: { testName, package: "./internal/example" },
      environment: minimalGoEnvironment(join(root, "cache"), resolved),
      goToolchain: resolved,
      expectedPackage: "example",
      spawn(executable) {
        spawnedExecutable = executable;
        return {
          status: 0,
          signal: null,
          error: null,
          stdout: passingEvents(testName),
          stderr: "",
        };
      },
    });
    assert.equal(spawnedExecutable, await realpath(process.execPath));
    assert.equal(result.classification.outcome, NamedTestOutcome.Pass);
  });
});

test("state left by an unmutated run cannot become a false mutant kill", async () => {
  await withTemporaryRoot("countershape-mutator-freshness-", async (root) => {
    let sequence = 0;
    let mutationApplied = false;
    const prepareExperiment = async ({ phase }) => {
      sequence += 1;
      const sandboxParent = join(root, `${phase}-${sequence}`);
      const sandbox = join(sandboxParent, "repo");
      await mkdir(sandbox, { recursive: true });
      return { sandboxParent, sandbox, phase };
    };
    const cleanupExperiment = async (context) => {
      await rm(context.sandboxParent, { recursive: true, force: true });
    };
    const runNamedTest = async (context) => {
      if (context.phase === "mutant") assert.equal(mutationApplied, true);
      const marker = join(context.sandbox, "state-left-by-test");
      try {
        await readFile(marker);
        return {
          classification: {
            outcome: NamedTestOutcome.Failure,
            detail: "a second run in the same root would fail",
          },
          output: "",
        };
      } catch (error) {
        if (error.code !== "ENOENT") throw error;
      }
      await writeFile(marker, "state\n", "utf8");
      return {
        classification: { outcome: NamedTestOutcome.Pass, detail: "fresh-root first run passed" },
        output: "",
      };
    };
    const applyMutation = async () => {
      mutationApplied = true;
    };

    await assert.rejects(
      runMutant(MUTANTS[0], "/admitted/go", {
        prepareExperiment,
        cleanupExperiment,
        runNamedTest,
        applyMutation,
      }),
      codeIs("MUTANT_SURVIVED"),
    );
    assert.equal(sequence, 3);
  });
});

test("baseline requires the exact named test to run and pass", () => {
  const testName = "TestExactTarget";
  const passed = classifyNamedGoTest({
    status: 0,
    stdout: passingEvents(testName),
    testName,
    expectedPackage: "example",
  });
  assert.equal(passed.outcome, NamedTestOutcome.Pass);
  assert.doesNotThrow(() => requireBaselinePass(passed, "synthetic"));

  const missing = classifyNamedGoTest({ status: 0, stdout: "", testName, expectedPackage: "example" });
  assert.equal(missing.outcome, NamedTestOutcome.Missing);
  assert.throws(() => requireBaselinePass(missing, "synthetic"), codeIs("BASELINE_TEST_NOT_PASSING"));
});

test("only an exact named go-test failure is a mutation kill", () => {
  const testName = "TestExactTarget";
  const failed = classifyNamedGoTest({
    status: 1,
    stdout: failingEvents(testName),
    testName,
    expectedPackage: "example",
  });
  assert.equal(failed.outcome, NamedTestOutcome.Failure);

  const other = classifyNamedGoTest({
    status: 1,
    stdout: failingEvents("TestOther"),
    testName,
    expectedPackage: "example",
  });
  assert.equal(other.outcome, NamedTestOutcome.Infrastructure);
});

test("wrong-package, duplicate terminal, and raw-stderr events are infrastructure", () => {
  const testName = "TestExactTarget";
  const wrongPackage = classifyNamedGoTest({
    status: 1,
    stdout: failingEvents(testName, "example/other"),
    testName,
    expectedPackage: "example",
  });
  assert.equal(wrongPackage.outcome, NamedTestOutcome.Infrastructure);

  const duplicatePass = classifyNamedGoTest({
    status: 0,
    stdout: `${passingEvents(testName)}\n${event("pass", testName)}`,
    testName,
    expectedPackage: "example",
  });
  assert.equal(duplicatePass.outcome, NamedTestOutcome.Infrastructure);

  const rawStderr = classifyNamedGoTest({
    status: 1,
    stdout: failingEvents(testName),
    stderr: "unstructured tool failure",
    testName,
    expectedPackage: "example",
  });
  assert.equal(rawStderr.outcome, NamedTestOutcome.Infrastructure);
});

test("build failures, malformed JSON, signals, and spawn errors remain infrastructure", () => {
  const testName = "TestExactTarget";
  const buildFailure = classifyNamedGoTest({
    status: 1,
    stdout: JSON.stringify({ Action: "fail", Package: "example" }),
    testName,
    expectedPackage: "example",
  });
  assert.equal(buildFailure.outcome, NamedTestOutcome.Infrastructure);

  const malformed = classifyNamedGoTest({
    status: 1,
    stdout: "not-json",
    testName,
    expectedPackage: "example",
  });
  assert.equal(malformed.outcome, NamedTestOutcome.Infrastructure);

  const signalled = classifyNamedGoTest({
    status: null,
    signal: "SIGKILL",
    stdout: "",
    testName,
    expectedPackage: "example",
  });
  assert.equal(signalled.outcome, NamedTestOutcome.Infrastructure);

  const spawnError = classifyNamedGoTest({
    status: null,
    error: new Error("ENOENT"),
    stdout: "",
    testName,
    expectedPackage: "example",
  });
  assert.equal(spawnError.outcome, NamedTestOutcome.Infrastructure);
});
