import assert from "node:assert/strict";
import {
  chmod,
  lstat,
  mkdir,
  mkdtemp,
  readFile,
  readdir,
  rm,
  symlink,
  unlink,
  writeFile,
} from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  MutationGateError,
  NamedTestOutcome,
  applyExactMutation,
  copyRegularAllowlist,
  mutateRegularFileNoFollow,
} from "./mutate-u1.mjs";
import { runU2Mutant } from "./mutate-u2.mjs";
import {
  REQUIRED_U5_MUTANT_IDS,
  REVIEWED_U5_MUTANT_CONTRACT,
  U5_MUTANTS,
  U5_REVIEWED_EMBED_BINDINGS,
  U5_REVIEWED_NON_GO_FILES,
  U5_SANDBOX_FILE_ALLOWLIST,
  assertExactU5MutantDelta,
  assertReviewedU5Anchors,
  assertReviewedU5NamedTests,
  assertU5ManifestMatchesTree,
  assertU5MutantDefinitionSet,
  assertU5PrivateCopy,
  assertU5SourceAndSeedUnchanged,
  cleanupU5SeedSnapshot,
  createU5SeedSnapshot,
  exactU5ABADigest,
  selfTest,
} from "./mutate-u5.mjs";

const toolsRoot = dirname(fileURLToPath(import.meta.url));

function codeIs(expected) {
  return (error) => error instanceof MutationGateError && error.code === expected;
}

function classification(outcome, detail = outcome) {
  return { classification: { outcome, detail }, output: "" };
}

async function withTemporaryRoot(prefix, callback) {
  const root = await mkdtemp(join(tmpdir(), prefix));
  await chmod(root, 0o700);
  try {
    return await callback(root);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
}

async function writeSyntheticSource(root) {
  const allowlist = Object.freeze([
    "go.mod",
    "pkg/fixture.go",
    "pkg/fixture.mjs",
    "pkg/guard.go",
  ]);
  const scopeRoots = Object.freeze(["pkg"]);
  const reviewedNonGoFiles = Object.freeze(["pkg/fixture.mjs"]);
  const embedBindings = Object.freeze([
    Object.freeze({
      owner: "pkg/fixture.go",
      asset: "pkg/fixture.mjs",
      directive: "//go:embed fixture.mjs",
    }),
  ]);
  await mkdir(join(root, "pkg"), { recursive: true });
  await writeFile(join(root, "go.mod"), "module example.invalid/u5\n", "utf8");
  await writeFile(
    join(root, "pkg", "fixture.go"),
    "package fixture\n\nimport _ \"embed\"\n\n//go:embed fixture.mjs\nvar fixture string\n",
    "utf8",
  );
  await writeFile(join(root, "pkg", "fixture.mjs"), "export const fixture = true;\n", "utf8");
  await writeFile(join(root, "pkg", "guard.go"), "package fixture\nconst guarded = true // SYNTHETIC_U5_ANCHOR\n", "utf8");
  return Object.freeze({ allowlist, scopeRoots, reviewedNonGoFiles, embedBindings });
}

function inspectSynthetic(root, contract) {
  return assertU5ManifestMatchesTree(
    root,
    contract.allowlist,
    contract.scopeRoots,
    contract.reviewedNonGoFiles,
    contract.embedBindings,
  );
}

test("U5 required IDs and reviewed tuples are one recursively frozen exact set of twenty-two", () => {
  assert.equal(REQUIRED_U5_MUTANT_IDS.length, 22);
  assert.equal(new Set(REQUIRED_U5_MUTANT_IDS).size, 22);
  assert.equal(Object.isFrozen(REQUIRED_U5_MUTANT_IDS), true);
  assert.equal(Object.isFrozen(REVIEWED_U5_MUTANT_CONTRACT), true);
  assert.equal(REVIEWED_U5_MUTANT_CONTRACT.every((tuple) => Object.isFrozen(tuple)), true);
  assert.equal(Object.isFrozen(U5_MUTANTS), true);
  assert.equal(U5_MUTANTS.every((tuple) => Object.isFrozen(tuple)), true);
  assert.doesNotThrow(() => assertU5MutantDefinitionSet());

  const byID = new Map(U5_MUTANTS.map((mutant) => [mutant.id, mutant]));
  const skippedSweep = byID.get("skip-fresh-final-sweep");
  assert.equal(skippedSweep.find.split("neighbors :=").length - 1, 1);
  assert.equal(skippedSweep.replace.split("neighbors :=").length - 1, 1);
  assert.match(byID.get("ignore-parent-cancellation").find, /return ReasonReductionCancelled/u);
  assert.match(byID.get("ignore-parent-cancellation").replace, /return UnresolvedReason\(""\)/u);
  assert.match(byID.get("ignore-parent-cancellation-during-final-enumeration").find, /s\.stopReason\(ctx\)/u);
  assert.match(
    byID.get("ignore-parent-cancellation-during-final-enumeration").replace,
    /s\.parentContext, ctx = context\.Background\(\), context\.Background\(\)/u,
  );
  assert.equal(
    new Set(U5_MUTANTS.map((mutant) => `${mutant.file}\0${mutant.find}\0${mutant.replace}`)).size,
    22,
  );
  assert.equal(
    byID.get("accept-map-backed-trial-count-mismatch").testName,
    "TestMapBackedEvaluationRejectsReportedTrialCountMismatch",
  );
  const evaluatorAccounting = byID.get("drop-evaluator-error-trial-accounting");
  assert.match(evaluatorAccounting.find, /UnresolvedReason: ReasonEvaluatorError, UnresolvedEvidence: evidence,[\s\S]*CandidateTrials: observation\.CandidateTrials/u);
  assert.match(evaluatorAccounting.replace, /UnresolvedReason: ReasonEvaluatorError, UnresolvedEvidence: evidence/u);
  assert.equal(evaluatorAccounting.replace.includes("CandidateTrials:"), false);
  assert.equal(
    byID.get("allow-reused-evaluation-evidence").testName,
    "TestCrossProposalEvidenceReuseIsRecordedAsTerminalRefusal",
  );

  assert.throws(() => assertU5MutantDefinitionSet(U5_MUTANTS.slice(0, -1)), codeIs("U5_MUTANT_SET_MISMATCH"));
  const reordered = [...U5_MUTANTS];
  [reordered[0], reordered[1]] = [reordered[1], reordered[0]];
  assert.throws(() => assertU5MutantDefinitionSet(reordered), codeIs("U5_MUTANT_SET_MISMATCH"));
  const duplicate = [...U5_MUTANTS.slice(0, -1), U5_MUTANTS.at(-2)];
  assert.throws(() => assertU5MutantDefinitionSet(duplicate), codeIs("U5_MUTANT_SET_MISMATCH"));
  const tampered = U5_MUTANTS.map((mutant) => ({ ...mutant }));
  tampered[0].replace += "\n// unreviewed weakening";
  assert.throws(() => assertU5MutantDefinitionSet(tampered), codeIs("U5_MUTANT_CONTRACT_MISMATCH"));
  const extraField = U5_MUTANTS.map((mutant) => ({ ...mutant }));
  extraField[0].unreviewed = true;
  assert.throws(() => assertU5MutantDefinitionSet(extraField), codeIs("U5_MUTANT_CONTRACT_MISMATCH"));
  const mutableExactClone = U5_MUTANTS.map((mutant) => ({ ...mutant }));
  assert.throws(() => assertU5MutantDefinitionSet(mutableExactClone), codeIs("U5_MUTANT_SET_NOT_IMMUTABLE"));
  assert.throws(
    () => assertU5MutantDefinitionSet(U5_MUTANTS, [...REQUIRED_U5_MUTANT_IDS], REVIEWED_U5_MUTANT_CONTRACT),
    codeIs("U5_INVALID_REQUIRED_ID_SET"),
  );
});

test("the twenty-two reviewed tuples bind exactly one current top-level killer each", async () => {
  await assert.doesNotReject(assertReviewedU5Anchors());
  await assert.doesNotReject(assertReviewedU5NamedTests());
});

test("each U5 mutation remains one exact effective replacement", () => {
  const mutant = { id: "synthetic", file: "guard.go", find: "ANCHOR", replace: "MUTATED" };
  assert.equal(applyExactMutation("before ANCHOR after", mutant), "before MUTATED after");
  assert.throws(() => applyExactMutation("no marker", mutant), codeIs("MUTATION_ANCHOR_COUNT"));
  assert.throws(() => applyExactMutation("ANCHOR then ANCHOR", mutant), codeIs("MUTATION_ANCHOR_COUNT"));
});

test("the U5 positive manifest is exact and excludes ambient secrets and repository authority", async () => {
  assert.equal(new Set(U5_SANDBOX_FILE_ALLOWLIST).size, U5_SANDBOX_FILE_ALLOWLIST.length);
  for (const forbidden of [".env", ".git/config", ".didrun/claims.json", "credentials.json", "CLAUDE.md"]) {
    assert.equal(U5_SANDBOX_FILE_ALLOWLIST.includes(forbidden), false, `${forbidden} entered the U5 closure`);
  }
  assert.equal(U5_SANDBOX_FILE_ALLOWLIST.some((file) => file.startsWith(".git/") || file.startsWith(".didrun/")), false);
  const manifest = await assertU5ManifestMatchesTree();
  assert.equal(manifest.entries.length, U5_SANDBOX_FILE_ALLOWLIST.length);
  for (const file of U5_REVIEWED_NON_GO_FILES) {
    const entry = manifest.entries.find((candidate) => candidate.file === file);
    assert.ok(entry && entry.bytes > 0 && /^[0-9a-f]{64}$/u.test(entry.sha256));
  }
  assert.deepEqual(
    U5_REVIEWED_EMBED_BINDINGS.map((binding) => binding.asset).sort(),
    [...U5_REVIEWED_NON_GO_FILES].sort(),
  );
});

test("unlisted Go, unsupported assets, symlinks, escapes, and overlapping scopes all fail closed", async () => {
  await withTemporaryRoot("countershape-u5-manifest-test-", async (root) => {
    const contract = await writeSyntheticSource(root);
    await assert.doesNotReject(inspectSynthetic(root, contract));

    const unlisted = join(root, "pkg", "unlisted.go");
    await writeFile(unlisted, "package fixture\n", "utf8");
    await assert.rejects(inspectSynthetic(root, contract), codeIs("U5_MANIFEST_TREE_MISMATCH"));
    await unlink(unlisted);

    const unsupported = join(root, "pkg", "credentials.json");
    await writeFile(unsupported, "{}\n", "utf8");
    await assert.rejects(inspectSynthetic(root, contract), codeIs("U5_MANIFEST_TREE_UNREVIEWED_NON_GO"));
    await unlink(unsupported);

    const linked = join(root, "pkg", "linked.go");
    await symlink(join(root, "pkg", "guard.go"), linked);
    await assert.rejects(inspectSynthetic(root, contract), codeIs("U5_MANIFEST_TREE_SYMLINK"));
    await unlink(linked);

    await assert.rejects(
      assertU5ManifestMatchesTree(
        root,
        [...contract.allowlist, "../escape.go"],
        contract.scopeRoots,
        contract.reviewedNonGoFiles,
        contract.embedBindings,
      ),
      codeIs("U5_UNSAFE_ALLOWLIST_PATH"),
    );

    await mkdir(join(root, "pkg", "nested"));
    await writeFile(join(root, "pkg", "nested", "nested.go"), "package nested\n", "utf8");
    await assert.rejects(
      assertU5ManifestMatchesTree(
        root,
        [...contract.allowlist, "pkg/nested/nested.go"],
        ["pkg", "pkg/nested"],
        contract.reviewedNonGoFiles,
        contract.embedBindings,
      ),
      codeIs("U5_OVERLAPPING_SOURCE_SCOPES"),
    );

    const linkedRoot = `${root}-linked-root`;
    await symlink(root, linkedRoot);
    try {
      await assert.rejects(inspectSynthetic(linkedRoot, contract), codeIs("U5_MANIFEST_ROOT_INVALID"));
    } finally {
      await unlink(linkedRoot);
    }
  });
});

test("non-Go admission is exactly paired with one reviewed go:embed owner", async () => {
  await withTemporaryRoot("countershape-u5-embed-test-", async (root) => {
    const contract = await writeSyntheticSource(root);
    await assert.rejects(
      assertU5ManifestMatchesTree(
        root,
        contract.allowlist.filter((file) => file !== "pkg/fixture.mjs"),
        contract.scopeRoots,
        contract.reviewedNonGoFiles,
        contract.embedBindings,
      ),
      codeIs("U5_NON_GO_ALLOWLIST_MISMATCH"),
    );
    const wrongBinding = [{ ...contract.embedBindings[0], directive: "//go:embed missing.mjs" }];
    await assert.rejects(
      assertU5ManifestMatchesTree(
        root,
        contract.allowlist,
        contract.scopeRoots,
        contract.reviewedNonGoFiles,
        wrongBinding,
      ),
      codeIs("U5_EMBED_DIRECTIVE_COUNT"),
    );
    const owner = join(root, "pkg", "fixture.go");
    const original = await readFile(owner, "utf8");
    await writeFile(owner, `${original}\n//go:embed fixture.mjs\nvar duplicate string\n`, "utf8");
    await assert.rejects(inspectSynthetic(root, contract), codeIs("U5_EMBED_DIRECTIVE_COUNT"));
  });
});

test("private U5 copies contain only regular allowlisted files at private modes", async () => {
  await withTemporaryRoot("countershape-u5-private-test-", async (root) => {
    const source = join(root, "source");
    await mkdir(source);
    const contract = await writeSyntheticSource(source);
    await writeFile(join(source, ".env"), "TOKEN=must-not-copy\n", "utf8");
    const destination = join(root, "destination");
    await copyRegularAllowlist(source, destination, contract.allowlist);
    await assert.doesNotReject(assertU5PrivateCopy(destination, contract.allowlist));
    assert.equal((await lstat(destination)).mode & 0o777, 0o700);
    for (const file of contract.allowlist) assert.equal((await lstat(join(destination, file))).mode & 0o777, 0o600);
    assert.equal((await readdir(destination)).includes(".env"), false);

    const extra = join(destination, "pkg", "extra.go");
    await writeFile(extra, "package fixture\n", { mode: 0o600 });
    await chmod(extra, 0o600);
    await assert.rejects(assertU5PrivateCopy(destination, contract.allowlist), codeIs("U5_PRIVATE_COPY_FILE_SET"));
    await unlink(extra);
    const linked = join(destination, "pkg", "linked.go");
    await symlink(join(destination, "pkg", "guard.go"), linked);
    await assert.rejects(assertU5PrivateCopy(destination, contract.allowlist), codeIs("U5_PRIVATE_COPY_SYMLINK"));
  });
});

test("immutable U5 seed revalidation catches both source and seed tampering", async () => {
  await withTemporaryRoot("countershape-u5-seed-test-", async (root) => {
    const source = join(root, "source");
    await mkdir(source);
    const contract = await writeSyntheticSource(source);
    const seed = await createU5SeedSnapshot({ sourceRoot: source, ...contract });
    try {
      await assert.doesNotReject(assertU5SourceAndSeedUnchanged(seed));
      const sourceGuard = join(source, "pkg", "guard.go");
      const sourceBytes = await readFile(sourceGuard);
      await writeFile(sourceGuard, "package fixture\nconst guarded = false // SYNTHETIC_U5_ANCHOR\n", "utf8");
      await assert.rejects(assertU5SourceAndSeedUnchanged(seed), codeIs("U5_ADMITTED_SOURCE_CHANGED"));
      await writeFile(sourceGuard, sourceBytes);
      await assert.doesNotReject(assertU5SourceAndSeedUnchanged(seed));

      const seedGuard = join(seed.root, "pkg", "guard.go");
      await writeFile(seedGuard, "package fixture\nconst guarded = false // SYNTHETIC_U5_ANCHOR\n", "utf8");
      await assert.rejects(assertU5SourceAndSeedUnchanged(seed), codeIs("U5_SEED_SNAPSHOT_CHANGED"));
    } finally {
      await cleanupU5SeedSnapshot(seed);
    }
  });
});

test("mutant delta is the reviewed replacement in exactly one admitted file", async () => {
  await withTemporaryRoot("countershape-u5-delta-test-", async (root) => {
    const source = join(root, "source");
    await mkdir(source);
    const contract = await writeSyntheticSource(source);
    const seed = await createU5SeedSnapshot({ sourceRoot: source, ...contract });
    const mutant = {
      id: "synthetic-u5",
      file: "pkg/guard.go",
      find: "const guarded = true // SYNTHETIC_U5_ANCHOR",
      replace: "const guarded = false // SYNTHETIC_U5_ANCHOR",
    };
    try {
      const sandbox = join(root, "sandbox");
      await copyRegularAllowlist(seed.root, sandbox, seed.allowlist);
      await mutateRegularFileNoFollow(sandbox, mutant);
      await assert.doesNotReject(assertExactU5MutantDelta(sandbox, seed, mutant));
      await writeFile(join(sandbox, "go.mod"), "module changed.invalid/u5\n", "utf8");
      await assert.rejects(assertExactU5MutantDelta(sandbox, seed, mutant), codeIs("U5_MUTANT_DELTA_SCOPE"));
    } finally {
      await cleanupU5SeedSnapshot(seed);
    }
  });
});

test("A/B/A execution requires fresh roots, passing controls, and one exact named-test failure", async () => {
  const phases = [];
  let invocation = 0;
  const result = await runU2Mutant(U5_MUTANTS[0], {}, {
    async prepareExperiment({ phase }) {
      phases.push(phase);
      return { sandbox: `/fresh/u5/${phase}`, sandboxParent: `/fresh/u5/${phase}` };
    },
    async cleanupExperiment() {},
    async applyMutation() {},
    async runNamedTest() {
      const outcomes = [NamedTestOutcome.Pass, NamedTestOutcome.Failure, NamedTestOutcome.Pass];
      return classification(outcomes[invocation++]);
    },
  });
  assert.match(result, /^KILLED replace-full-map-preservation-with-partition-shape/u);
  assert.deepEqual(phases, ["baseline", "mutant", "post-control"]);
  assert.equal(invocation, 3);

  let badControlRun = 0;
  await assert.rejects(
    runU2Mutant(U5_MUTANTS[0], {}, {
      async prepareExperiment({ phase }) {
        return { sandbox: `/fresh/u5/bad-${phase}`, sandboxParent: `/fresh/u5/bad-${phase}` };
      },
      async cleanupExperiment() {},
      async applyMutation() {},
      async runNamedTest() {
        const outcomes = [NamedTestOutcome.Pass, NamedTestOutcome.Failure, NamedTestOutcome.Failure];
        return classification(outcomes[badControlRun++]);
      },
    }),
    codeIs("BASELINE_TEST_NOT_PASSING"),
  );
});

test("the U5 receipt binds exact A/B/A order, manifests, tuple identity, and three roots", () => {
  const seed = { manifest: { digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" } };
  const records = [
    { phase: "baseline", manifest: seed.manifest.digest, sandbox: "/fresh/u5/a" },
    { phase: "mutant", manifest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", sandbox: "/fresh/u5/b" },
    { phase: "post-control", manifest: seed.manifest.digest, sandbox: "/fresh/u5/c" },
  ];
  const first = exactU5ABADigest(U5_MUTANTS[0], records, seed);
  assert.match(first, /^sha256:[0-9a-f]{64}$/u);
  assert.equal(exactU5ABADigest(U5_MUTANTS[0], records, seed), first);
  assert.notEqual(exactU5ABADigest(U5_MUTANTS[1], records, seed), first);
  assert.throws(
    () => exactU5ABADigest(Object.freeze({ ...U5_MUTANTS[0], replace: `${U5_MUTANTS[0].replace}\n// tampered` }), records, seed),
    codeIs("U5_ABA_RECEIPT_INVALID"),
  );
  assert.throws(
    () => exactU5ABADigest(U5_MUTANTS[0], records.map((record) => ({ ...record, sandbox: "/fresh/u5/reused" })), seed),
    codeIs("U5_ABA_RECEIPT_INVALID"),
  );
  assert.throws(
    () => exactU5ABADigest(U5_MUTANTS[0], [records[1], records[0], records[2]], seed),
    codeIs("U5_ABA_RECEIPT_INVALID"),
  );
  assert.throws(
    () => exactU5ABADigest(U5_MUTANTS[0], records.map((record) => ({ ...record, manifest: seed.manifest.digest })), seed),
    codeIs("U5_ABA_RECEIPT_INVALID"),
  );
  assert.throws(
    () => exactU5ABADigest(U5_MUTANTS[0], records.map((record, index) => index === 0 ? { ...record, extra: true } : record), seed),
    codeIs("U5_ABA_RECEIPT_INVALID"),
  );
});

test("the execution seam keeps Go and C compiler identity checks around every named test", async () => {
  const source = await readFile(join(toolsRoot, "mutate-u5.mjs"), "utf8");
  assert.match(source, /function revalidateU5Toolchain\(toolchain\)[\s\S]*revalidateAdmittedGoExecutable\(toolchain\.executable\);[\s\S]*revalidateAdmittedDarwinCCompiler\(toolchain\.compiler\);/u);
  assert.match(source, /async function runNamedU5GoTest[\s\S]*revalidateU5Toolchain\(toolchain\);[\s\S]*try \{[\s\S]*runNamedU2GoTest[\s\S]*finally \{[\s\S]*revalidateU5Toolchain\(toolchain\);/u);
  assert.match(source, /assertU5SourceAndSeedUnchanged\(context\.u5Seed\)[\s\S]*expectedU5ExperimentManifest\(context, mutant\)/u);
  assert.match(source, /U5 mutation gate: \$\{results\.length\}\/\$\{REQUIRED_U5_MUTANT_IDS\.length\} required mutants killed/u);
});

test("the integrated U5 self-test closes all hostile and deterministic receipt tripwires", async () => {
  const message = await selfTest();
  assert.match(message, /^U5 mutation self-test passed: 22\/22 frozen tuples; 1\/1 deterministic synthetic A\/B\/A runner proof;/u);
});
