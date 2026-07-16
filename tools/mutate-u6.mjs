#!/usr/bin/env node

import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { chmod, mkdir, mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import {
  MutationGateError,
  NamedTestOutcome,
  applyExactMutation,
  copyRegularAllowlist,
  mutateRegularFileNoFollow,
  revalidateAdmittedGoExecutable,
} from "./mutate-u1.mjs";
import {
  admitU2DarwinToolchain,
  minimalU2GoEnvironment,
  revalidateAdmittedDarwinCCompiler,
  runNamedU2GoTest,
  runU2Mutant,
} from "./mutate-u2.mjs";
import {
  U5_REVIEWED_EMBED_BINDINGS,
  U5_REVIEWED_NON_GO_FILES,
  U5_SANDBOX_FILE_ALLOWLIST,
  U5_SOURCE_SCOPE_ROOTS,
  assertExactU5MutantDelta,
  assertU5ManifestMatchesTree,
  assertU5PrivateCopy,
  assertU5SourceAndSeedUnchanged,
  cleanupU5SeedSnapshot,
  createU5SeedSnapshot,
} from "./mutate-u5.mjs";

const modulePath = fileURLToPath(import.meta.url);
const repoRoot = resolve(dirname(modulePath), "..");

// U6 reuses U5's reviewed private-copy and manifest machinery, but owns an
// independent positive source closure and independent frozen fault contract.
// The private copies are containment against accidental ambient-file use, not
// an OS sandbox: child tests retain the invoking user's host authority.
export const U6_ADDITIONAL_FILES = Object.freeze([
  "internal/choice/blind.go",
  "internal/choice/blind_test.go",
  "internal/choice/choicepoint.go",
  "internal/choice/decision.go",
  "internal/choice/promotion/authority/authority.go",
  "internal/choice/promotion/internal/publication/authority.go",
  "internal/choice/promotion/internal/publication/authority_test.go",
  "internal/choice/promotion/service.go",
  "internal/choice/schema_parity_test.go",
  "internal/choice/session.go",
  "internal/choice/session_roundtrip_test.go",
  "internal/choice/validation.go",
  "internal/choice/validation_test.go",
  "internal/compare/wire.go",
  "internal/confirmation/authority/authority.go",
  "internal/confirmation/internal/publication/authority.go",
  "internal/confirmation/internal/publication/authority_test.go",
  "internal/confirmation/service.go",
  "internal/confirmation/service_test.go",
  "internal/confirmation/wire.go",
  "internal/observe/executed_link.go",
  "internal/store/head.go",
  "internal/store/head_darwin.go",
  "internal/store/head_test.go",
  "internal/store/head_unsupported.go",
  "internal/store/object_store.go",
  "internal/store/object_store_test.go",
  "internal/store/public_api_test.go",
  "internal/world/fresh_confirmation.go",
]);

export const U6_SANDBOX_FILE_ALLOWLIST = Object.freeze([
  ...U5_SANDBOX_FILE_ALLOWLIST,
  ...U6_ADDITIONAL_FILES,
]);

export const U6_SOURCE_SCOPE_ROOTS = Object.freeze([
  ...U5_SOURCE_SCOPE_ROOTS,
  "internal/choice",
  "internal/confirmation",
]);

export const REQUIRED_U6_MUTANT_IDS = Object.freeze([
  "publish-before-cas-compare",
  "last-writer-wins-head-update",
  "mutable-object-overwrite",
  "nonce-only-freshness",
  "allow-discovery-evidence-reuse",
  "confirm-partition-shape-not-labeled-map",
  "accept-stale-head-digest",
  "hide-identity-only-in-presentation",
  "order-blind-cards-by-support",
  "default-first-selected-field",
  "compile-allow-many-as-cross-product",
  "collapse-missing-into-present-empty",
  "cast-reject-all-as-compilable",
  "normalize-receipt-grade",
  "bind-alias-to-first-member",
  "omit-run-challenge-from-nonce",
  "zero-confirmation-schedule-offset",
]);

// This reviewed tuple list is deliberately separate from REQUIRED_U6_MUTANT_IDS
// and the executable clone below. Missing, reordered, renamed, or weakened
// faults close the gate before any Go process starts.
export const REVIEWED_U6_MUTANT_CONTRACT = Object.freeze([
  Object.freeze({
    id: "publish-before-cas-compare",
    file: "internal/store/head.go",
    find: [
      "\t// MUTANT_U6_CAS_COMPARE_AFTER_PUBLISH: this full comparison must precede",
      "\t// every successor object or temporary-file creation.",
      "\tif !sameHeadToken(currentToken, expected) {",
    ].join("\n"),
    replace: [
      "\t// MUTANT_U6_CAS_COMPARE_AFTER_PUBLISH: this full comparison must precede",
      "\t// every successor object or temporary-file creation.",
      "\tif _, err := s.publishLocked(ctx, next); err != nil {",
      "\t\treturn HeadToken{}, err",
      "\t}",
      "\tif !sameHeadToken(currentToken, expected) {",
    ].join("\n"),
    package: "./internal/store",
    testName: "TestStaleHeadRefusesBeforeSuccessorObjectPublication",
  }),
  Object.freeze({
    id: "last-writer-wins-head-update",
    file: "internal/store/head.go",
    find: [
      "\tcurrentToken := s.token(expected.study, current)",
      "\t// MUTANT_U6_CAS_COMPARE_AFTER_PUBLISH: this full comparison must precede",
      "\t// every successor object or temporary-file creation.",
      "\tif !sameHeadToken(currentToken, expected) {",
      "\t\treturn HeadToken{}, refuse(codeCASConflict, \"expected revision/digest no longer names the current head\", nil)",
      "\t}",
    ].join("\n"),
    replace: [
      "\t// Mutant: trust the caller's stale token as last-writer-wins authority.",
      "\tcurrentToken := expected",
      "\tcurrent.identity.Revision = expected.revision",
      "\tcurrent.identity.Stage = string(expected.stage)",
      "\tcurrent.identity.CurrentKind = expected.currentKind",
      "\tcurrent.identity.CurrentDigest = expected.currentDigest.String()",
      "\tcurrent.identity.PreviousObjectDigest = expected.previousObject.String()",
      "\tcurrent.identity.LineageRootDigest = expected.lineageRoot.String()",
      "\tcurrent.digest = expected.headDigest",
      "\t// MUTANT_U6_CAS_COMPARE_AFTER_PUBLISH: comparison deliberately bypassed.",
      "\tif false && !sameHeadToken(currentToken, expected) {",
      "\t\treturn HeadToken{}, refuse(codeCASConflict, \"expected revision/digest no longer names the current head\", nil)",
      "\t}",
    ].join("\n"),
    package: "./internal/store",
    testName: "TestStudyHeadConcurrentStoreInstancesHaveExactlyOneWinner",
  }),
  Object.freeze({
    id: "mutable-object-overwrite",
    file: "internal/store/object_store.go",
    find: [
      "\tif _, err := os.Lstat(destination); err == nil {",
      "\t\treturn s.openAtLocked(ctx, object, destination, shard)",
      "\t} else if !errors.Is(err, os.ErrNotExist) {",
    ].join("\n"),
    replace: [
      "\tif _, err := os.Lstat(destination); err == nil {",
      "\t\t// Mutant: delete immutable history and republish replacement bytes.",
      "\t\tif removeErr := os.Remove(destination); removeErr != nil {",
      "\t\t\treturn ObjectAuthority{}, removeErr",
      "\t\t}",
      "\t} else if !errors.Is(err, os.ErrNotExist) {",
    ].join("\n"),
    package: "./internal/store",
    testName: "TestObjectStoreNeverReplacesExistingDestination",
  }),
  Object.freeze({
    id: "nonce-only-freshness",
    file: "internal/confirmation/service.go",
    find: "\t\tif _, duplicate := item.set[item.digest]; duplicate {",
    replace: "\t\tif _, duplicate := item.set[item.digest]; false && duplicate {",
    package: "./internal/confirmation",
    testName: "TestPhysicalLedgerRejectsReuseInEveryPhysicalEvidenceDomain",
  }),
  Object.freeze({
    id: "allow-discovery-evidence-reuse",
    file: "internal/confirmation/service.go",
    find: "\t\t\tif _, reused := set.prior[digest]; reused {",
    replace: "\t\t\tif _, reused := set.prior[digest]; false && reused {",
    package: "./internal/confirmation",
    testName: "TestPriorEvidenceLedgerRejectsEveryDiscoveryReductionDigestReuse",
  }),
  Object.freeze({
    id: "confirm-partition-shape-not-labeled-map",
    file: "internal/confirmation/service.go",
    find: [
      "\tassessment := compare.AssessPreservation(reduced, confirmed) // MUTANT_U6_CONFIRM_PARTITION_SHAPE",
      "\tif !assessment.Valid() || assessment.Relation() != compare.PreservationEqual {",
    ].join("\n"),
    replace: [
      "\tassessment := compare.AssessPreservation(reduced, confirmed) // MUTANT_U6_CONFIRM_PARTITION_SHAPE",
      "\tif len(reduced.DisplayGroups()) == len(confirmed.DisplayGroups()) {",
      "\t\treturn assessment, nil",
      "\t}",
      "\tif !assessment.Valid() || assessment.Relation() != compare.PreservationEqual {",
    ].join("\n"),
    package: "./internal/confirmation",
    testName: "TestConfirmationRequiresExactLabeledMapNotPartitionShape",
  }),
  Object.freeze({
    id: "accept-stale-head-digest",
    file: "internal/store/head.go",
    find: "\t\tleft.previousObject == right.previousObject && left.lineageRoot == right.lineageRoot && left.headDigest == right.headDigest",
    replace: "\t\tleft.previousObject == right.previousObject && left.lineageRoot == right.lineageRoot && left.headDigest.Valid() && right.headDigest.Valid()",
    package: "./internal/store",
    testName: "TestStudyHeadCASBindsEveryExpectedTokenFieldBeforePublication",
  }),
  Object.freeze({
    id: "hide-identity-only-in-presentation",
    file: "internal/choice/blind.go",
    find: "\t\tcards[index] = BlindCard{Alias: current.alias, Fields: fields} // MUTANT_U6_BLIND_PRESENT_CANDIDATE_IDENTITY",
    replace: "\t\tcards[index] = BlindCard{Alias: current.refs[0].candidate.String(), Fields: fields} // MUTANT_U6_BLIND_PRESENT_CANDIDATE_IDENTITY",
    package: "./internal/choice",
    testName: "TestBlindRenderedCardsOmitCandidateIdentityAndSupportMetadata",
  }),
  Object.freeze({
    id: "order-blind-cards-by-support",
    file: "internal/choice/blind.go",
    find: [
      "\tsort.Slice(groups, func(i, j int) bool { // MUTANT_U6_BLIND_ORDER_BY_SUPPORT",
      "\t\tif groups[i].orderKey != groups[j].orderKey {",
      "\t\t\treturn groups[i].orderKey < groups[j].orderKey",
      "\t\t}",
      "\t\treturn groups[i].fingerprint.String() < groups[j].fingerprint.String()",
      "\t})",
    ].join("\n"),
    replace: [
      "\tsort.Slice(groups, func(i, j int) bool { // MUTANT_U6_BLIND_ORDER_BY_SUPPORT",
      "\t\treturn len(groups[i].refs) > len(groups[j].refs)",
      "\t})",
    ].join("\n"),
    package: "./internal/choice",
    testName: "TestBlindDTOOrderIgnoresSupportCountAndCandidateInputOrder",
  }),
  Object.freeze({
    id: "default-first-selected-field",
    file: "internal/choice/validation.go",
    find: "\tif len(input.SelectedFields) == 0 { // MUTANT_U1_CHOICE_ALLOW_EMPTY_FIELDS",
    replace: "\tif false && len(input.SelectedFields) == 0 { // MUTANT_U1_CHOICE_ALLOW_EMPTY_FIELDS",
    package: "./internal/choice",
    testName: "TestValidateRulingRefusesEmptySelectedFields",
  }),
  Object.freeze({
    id: "compile-allow-many-as-cross-product",
    file: "internal/choice/validation.go",
    find: [
      "\t_, ok := r.allowedTupleKeys[key] // MUTANT_U1_CHOICE_INDEPENDENT_SETS",
      "\treturn ok, nil",
    ].join("\n"),
    replace: [
      "\t_ = key // MUTANT_U1_CHOICE_INDEPENDENT_SETS",
      "\tfor _, selectedField := range r.selectedFields {",
      "\t\tcandidateValue := fields[selectedField.text]",
      "\t\tfound := false",
      "\t\tfor _, allowedTuple := range r.allowedTuples {",
      "\t\t\tfor _, allowedField := range allowedTuple.Fields {",
      "\t\t\t\tif allowedField.FieldID == selectedField.text && allowedField.Value.identityKey() == candidateValue.identityKey() {",
      "\t\t\t\t\tfound = true",
      "\t\t\t\t\tbreak",
      "\t\t\t\t}",
      "\t\t\t}",
      "\t\t\tif found {",
      "\t\t\t\tbreak",
      "\t\t\t}",
      "\t\t}",
      "\t\tif !found {",
      "\t\t\treturn false, nil",
      "\t\t}",
      "\t}",
      "\treturn true, nil",
    ].join("\n"),
    package: "./internal/choice",
    testName: "TestCompilableRulingDoesNotAllowCrossProduct",
  }),
  Object.freeze({
    id: "collapse-missing-into-present-empty",
    file: "internal/choice/validation.go",
    find: [
      "\tcase ValueMissing:",
      "\t\treturn \"M\"",
    ].join("\n"),
    replace: [
      "\tcase ValueMissing:",
      "\t\treturn \"S0:\"",
    ].join("\n"),
    package: "./internal/choice",
    testName: "TestMissingNullAndPresentEmptyRemainDistinct",
  }),
  Object.freeze({
    id: "cast-reject-all-as-compilable",
    file: "internal/choice/validation.go",
    find: [
      "\t\tnoncompilable := noncompilableRuling{action: input.Action}",
      "\t\treturn ValidatedRuling{action: input.Action, eligibility: noncompilable}, nil",
    ].join("\n"),
    replace: [
      "\t\tcompilable := compilableRuling{action: input.Action}",
      "\t\treturn ValidatedRuling{action: input.Action, eligibility: compilable}, nil",
    ].join("\n"),
    package: "./internal/choice",
    testName: "TestNoncompilableActionsProduceSealedVariant",
  }),
  Object.freeze({
    id: "normalize-receipt-grade",
    file: "internal/domain/receipt.go",
    find: "\treturn NewDidrunReceipt(wire.GradeVerbatim, wire.CommitOID, digest)",
    replace: "\treturn NewDidrunReceipt(\"TREE-EXACT\", wire.CommitOID, digest)",
    package: "./internal/domain",
    testName: "TestReceiptGradeRoundTripsVerbatim",
  }),
  Object.freeze({
    id: "bind-alias-to-first-member",
    file: "internal/choice/blind.go",
    find: [
      "\t\t\talias, err := blindAliasForProjectionIdentity( // MUTANT_U6_BLIND_ALIAS_FIRST_MEMBER",
      "\t\t\t\tchoicepoint, outcome.ref.fingerprint.String(),",
      "\t\t\t)",
    ].join("\n"),
    replace: [
      "\t\t\talias, err := blindAliasForProjectionIdentity( // MUTANT_U6_BLIND_ALIAS_FIRST_MEMBER",
      "\t\t\t\tchoicepoint, outcome.ref.id.String(),",
      "\t\t\t)",
    ].join("\n"),
    package: "./internal/choice",
    testName: "TestBlindAliasBindsFingerprintNotFirstSupportingMember",
  }),
  Object.freeze({
    id: "omit-run-challenge-from-nonce",
    file: "internal/confirmation/service.go",
    find: "\t}{domain.SchemaVersion, \"ConfirmationInstanceNonce\", challenge.String(), ordinal})",
    replace: "\t}{domain.SchemaVersion, \"ConfirmationInstanceNonce\", \"\", ordinal})",
    package: "./internal/confirmation",
    testName: "TestConfirmationNonceBindsRunChallengeAndOrdinal",
  }),
  Object.freeze({
    id: "zero-confirmation-schedule-offset",
    file: "internal/observe/schedule.go",
    find: "\t\tstartOffset = 1 % len(canonicalRoster) // MUTANT_U6_CONFIRMATION_SCHEDULE_OFFSET_ZERO",
    replace: "\t\tstartOffset = 0 // MUTANT_U6_CONFIRMATION_SCHEDULE_OFFSET_ZERO",
    package: "./internal/observe",
    testName: "TestConfirmationScheduleIsPhaseBoundAndRotatedFromDiscovery",
  }),
]);

export const U6_MUTANTS = Object.freeze(
  REVIEWED_U6_MUTANT_CONTRACT.map((mutant) => Object.freeze({ ...mutant })),
);

const tupleFields = Object.freeze(["file", "find", "id", "package", "replace", "testName"]);

function sameArray(left, right) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

function exactOccurrenceCount(source, anchor) {
  if (typeof anchor !== "string" || anchor.length === 0) return 0;
  return source.split(anchor).length - 1;
}

export function assertU6MutantDefinitionSet(
  mutants = U6_MUTANTS,
  requiredIDs = REQUIRED_U6_MUTANT_IDS,
  reviewedContract = REVIEWED_U6_MUTANT_CONTRACT,
) {
  const requiredCount = 17;
  if (!Object.isFrozen(requiredIDs) || requiredIDs.length !== requiredCount || new Set(requiredIDs).size !== requiredCount) {
    throw new MutationGateError("U6_INVALID_REQUIRED_ID_SET", `U6 requires exactly ${requiredCount} unique frozen IDs`);
  }
  if (!Object.isFrozen(reviewedContract) || reviewedContract.length !== requiredCount ||
      reviewedContract.some((tuple) => !Object.isFrozen(tuple))) {
    throw new MutationGateError("U6_INVALID_REVIEWED_MUTANT_CONTRACT", "U6 reviewed tuples must be recursively frozen");
  }
  if (!Object.isFrozen(mutants) || mutants.length !== requiredCount || mutants.some((tuple) => !Object.isFrozen(tuple))) {
    throw new MutationGateError("U6_MUTANT_SET_NOT_IMMUTABLE", "U6 executable tuples must be one recursively frozen set of 17");
  }
  const ids = mutants.map((mutant) => mutant.id);
  if (!sameArray(ids, requiredIDs) || new Set(ids).size !== requiredCount ||
      !sameArray(reviewedContract.map((tuple) => tuple.id), requiredIDs)) {
    throw new MutationGateError("U6_MUTANT_SET_MISMATCH", "U6 mutant IDs, order, or independent tuple contract changed");
  }
  const sourceMutations = mutants.map((mutant) => `${mutant.file}\0${mutant.find}\0${mutant.replace}`);
  if (new Set(sourceMutations).size !== requiredCount) {
    throw new MutationGateError("U6_DUPLICATE_SOURCE_MUTATION", "U6 requires 17 distinct source faults");
  }
  const allowedFiles = new Set(U6_SANDBOX_FILE_ALLOWLIST);
  const reviewedByID = new Map(reviewedContract.map((tuple) => [tuple.id, tuple]));
  for (const mutant of mutants) {
    const reviewed = reviewedByID.get(mutant.id);
    if (!reviewed || !sameArray(Object.keys(mutant).sort(), tupleFields)) {
      throw new MutationGateError("U6_MUTANT_CONTRACT_MISMATCH", `${mutant.id} differs from the reviewed tuple shape`);
    }
    for (const field of tupleFields) {
      if (mutant[field] !== reviewed[field]) {
        throw new MutationGateError("U6_MUTANT_CONTRACT_MISMATCH", `${mutant.id}.${field} differs from the reviewed tuple`);
      }
    }
    if (!allowedFiles.has(mutant.file)) {
      throw new MutationGateError("U6_MUTANT_FILE_NOT_ALLOWLISTED", `${mutant.id} targets ${mutant.file}`);
    }
    if (typeof mutant.find !== "string" || mutant.find.length === 0 || mutant.find === mutant.replace) {
      throw new MutationGateError("U6_INVALID_MUTATION", `${mutant.id} is not one exact effective replacement`);
    }
    if (!/^\.\/internal\/(?:choice|confirmation|domain|observe|store)$/u.test(mutant.package)) {
      throw new MutationGateError("U6_INVALID_MUTATION_PACKAGE", `${mutant.id} has invalid package ${mutant.package}`);
    }
    if (!/^Test[A-Za-z0-9_]+$/u.test(mutant.testName)) {
      throw new MutationGateError("U6_INVALID_MUTATION_TEST", `${mutant.id} has invalid named test ${mutant.testName}`);
    }
  }
}

export function assertU6ManifestMatchesTree(sourceRoot = repoRoot) {
  return assertU5ManifestMatchesTree(
    sourceRoot,
    U6_SANDBOX_FILE_ALLOWLIST,
    U6_SOURCE_SCOPE_ROOTS,
    U5_REVIEWED_NON_GO_FILES,
    U5_REVIEWED_EMBED_BINDINGS,
  );
}

export async function assertReviewedU6Anchors(sourceRoot = repoRoot, mutants = U6_MUTANTS) {
  assertU6MutantDefinitionSet(mutants);
  await assertU6ManifestMatchesTree(sourceRoot);
  for (const mutant of mutants) {
    const source = await readFile(resolve(sourceRoot, mutant.file), "utf8");
    const count = exactOccurrenceCount(source, mutant.find);
    if (count !== 1) {
      throw new MutationGateError("U6_MUTATION_ANCHOR_COUNT", `${mutant.id}: exact anchor count was ${count}, want 1`);
    }
    const mutated = applyExactMutation(source, mutant);
    if (mutated === source || exactOccurrenceCount(mutated, mutant.find) !== 0) {
      throw new MutationGateError("U6_MUTATION_REPLACEMENT_INVALID", `${mutant.id}: replacement is not exact and singular`);
    }
  }
}

export async function assertReviewedU6NamedTests(sourceRoot = repoRoot, mutants = U6_MUTANTS) {
  assertU6MutantDefinitionSet(mutants);
  await assertU6ManifestMatchesTree(sourceRoot);
  for (const mutant of mutants) {
    const packageRoot = mutant.package.slice(2);
    const testFiles = U6_SANDBOX_FILE_ALLOWLIST.filter(
      (file) => file.startsWith(`${packageRoot}/`) && file.endsWith("_test.go"),
    );
    const pattern = new RegExp(`\\bfunc\\s+${mutant.testName}\\s*\\(`, "gu");
    let definitions = 0;
    for (const file of testFiles) {
      const source = await readFile(resolve(sourceRoot, file), "utf8");
      definitions += source.match(pattern)?.length ?? 0;
    }
    if (definitions !== 1) {
      throw new MutationGateError(
        "U6_NAMED_TEST_DEFINITION_COUNT",
        `${mutant.id}: ${mutant.package} ${mutant.testName} has ${definitions} admitted definitions, want 1`,
      );
    }
  }
}

function requireManifestEqual(actual, expected, code, detail) {
  const equal = actual.digest === expected.digest && actual.entries.length === expected.entries.length &&
    actual.entries.every((entry, index) => {
      const wanted = expected.entries[index];
      return entry.file === wanted.file && entry.bytes === wanted.bytes && entry.sha256 === wanted.sha256;
    });
  if (!equal) throw new MutationGateError(code, `${detail}: ${actual.digest} != ${expected.digest}`);
}

async function createU6SeedSnapshot() {
  return createU5SeedSnapshot({
    sourceRoot: repoRoot,
    allowlist: U6_SANDBOX_FILE_ALLOWLIST,
    scopeRoots: U6_SOURCE_SCOPE_ROOTS,
    reviewedNonGoFiles: U5_REVIEWED_NON_GO_FILES,
    embedBindings: U5_REVIEWED_EMBED_BINDINGS,
  });
}

async function prepareCacheRoot(cacheRoot) {
  await mkdir(cacheRoot, { mode: 0o700 });
  for (const name of ["home", "tmp", "go-tmp", "go-path", "go-build", "go-mod"]) {
    const directory = join(cacheRoot, name);
    await mkdir(directory, { recursive: true, mode: 0o700 });
    await chmod(directory, 0o700);
  }
}

function revalidateToolchain(toolchain) {
  revalidateAdmittedGoExecutable(toolchain.executable);
  revalidateAdmittedDarwinCCompiler(toolchain.compiler);
}

async function prepareExperiment({ mutant, toolchain, phase, seed, phaseRecords }) {
  await assertU5SourceAndSeedUnchanged(seed);
  const sandboxParent = await mkdtemp(join(tmpdir(), `countershape-u6-${phase}-`));
  await chmod(sandboxParent, 0o700);
  const sandbox = join(sandboxParent, "repo");
  const cacheRoot = join(sandboxParent, "cache");
  try {
    await copyRegularAllowlist(seed.root, sandbox, seed.allowlist);
    await assertU5PrivateCopy(sandbox, seed.allowlist);
    const manifest = await assertU6ManifestMatchesTree(sandbox);
    requireManifestEqual(manifest, seed.manifest, "U6_EXPERIMENT_SEED_MISMATCH", `${mutant.id} ${phase}`);
    await prepareCacheRoot(cacheRoot);
    revalidateToolchain(toolchain);
    const environment = minimalU2GoEnvironment(cacheRoot, toolchain);
    revalidateToolchain(toolchain);
    return { sandboxParent, sandbox, cacheRoot, environment, phase, seed, phaseRecords };
  } catch (error) {
    await rm(sandboxParent, { recursive: true, force: true });
    throw error;
  }
}

async function cleanupExperiment(context) {
  await rm(context.sandboxParent, { recursive: true, force: true });
}

async function expectedExperimentManifest(context, mutant) {
  if (context.phase === "mutant") {
    return assertExactU5MutantDelta(context.sandbox, context.seed, mutant);
  }
  await assertU5PrivateCopy(context.sandbox, context.seed.allowlist);
  const manifest = await assertU6ManifestMatchesTree(context.sandbox);
  requireManifestEqual(manifest, context.seed.manifest, "U6_CONTROL_SOURCE_CHANGED", `${mutant.id} ${context.phase}`);
  return manifest;
}

async function runNamedTest(context, mutant, toolchain) {
  await assertU5SourceAndSeedUnchanged(context.seed);
  const before = await expectedExperimentManifest(context, mutant);
  revalidateToolchain(toolchain);
  let result;
  try {
    result = runNamedU2GoTest({
      sandbox: context.sandbox,
      mutant,
      environment: context.environment,
      toolchain,
    });
  } finally {
    revalidateToolchain(toolchain);
  }
  const after = await expectedExperimentManifest(context, mutant);
  requireManifestEqual(after, before, "U6_TEST_CHANGED_SOURCE", `${mutant.id} ${context.phase}`);
  await assertU5SourceAndSeedUnchanged(context.seed);
  context.phaseRecords.push(Object.freeze({
    phase: context.phase,
    manifest: before.digest,
    sandbox: context.sandbox,
  }));
  return result;
}

export function exactU6ABADigest(mutant, phaseRecords, seed) {
  const reviewed = REVIEWED_U6_MUTANT_CONTRACT.find((tuple) => tuple.id === mutant?.id);
  if (!Object.isFrozen(mutant) || !reviewed || tupleFields.some((field) => mutant[field] !== reviewed[field])) {
    throw new MutationGateError("U6_ABA_RECEIPT_INVALID", `${mutant?.id ?? "unknown"} is not one frozen reviewed U6 tuple`);
  }
  const validDigest = (value) => /^sha256:[0-9a-f]{64}$/u.test(value);
  if (!Array.isArray(phaseRecords) || phaseRecords.some((record) =>
    !record || !sameArray(Object.keys(record).sort(), ["manifest", "phase", "sandbox"]))) {
    throw new MutationGateError("U6_ABA_RECEIPT_INVALID", `${mutant.id} has an unreviewed A/B/A record shape`);
  }
  const phases = phaseRecords.map((record) => record.phase);
  const manifests = phaseRecords.map((record) => record.manifest);
  const sandboxes = phaseRecords.map((record) => record.sandbox);
  if (!seed?.manifest || !validDigest(seed.manifest.digest) || phaseRecords.length !== 3 ||
      !sameArray(phases, ["baseline", "mutant", "post-control"]) || manifests.some((value) => !validDigest(value)) ||
      sandboxes.some((value) => typeof value !== "string" || !isAbsolute(value)) || new Set(sandboxes).size !== 3 ||
      manifests[0] !== seed.manifest.digest || manifests[2] !== seed.manifest.digest || manifests[1] === seed.manifest.digest) {
    throw new MutationGateError("U6_ABA_RECEIPT_INVALID", `${mutant.id} lacks exact fresh A/B/A closure facts`);
  }
  const canonical = JSON.stringify({
    schema_version: "countershape.u6.mutation-receipt/v1",
    mutant_tuple: {
      file: mutant.file,
      find: mutant.find,
      id: mutant.id,
      package: mutant.package,
      replace: mutant.replace,
      test_name: mutant.testName,
    },
    phases: phaseRecords,
    seed_manifest: seed.manifest.digest,
  });
  return `sha256:${createHash("sha256").update(canonical, "utf8").digest("hex")}`;
}

function fakeClassification(outcome) {
  return { classification: { outcome, detail: outcome }, output: "" };
}

export async function selfTest() {
  assertU6MutantDefinitionSet();
  await assertU6ManifestMatchesTree(repoRoot);
  await assertReviewedU6Anchors(repoRoot);
  await assertReviewedU6NamedTests(repoRoot);

  assert.throws(
    () => assertU6MutantDefinitionSet(U6_MUTANTS.slice(0, -1)),
    (error) => error instanceof MutationGateError && error.code === "U6_MUTANT_SET_NOT_IMMUTABLE",
  );
  const reordered = Object.freeze([U6_MUTANTS[1], U6_MUTANTS[0], ...U6_MUTANTS.slice(2)]);
  assert.throws(
    () => assertU6MutantDefinitionSet(reordered),
    (error) => error instanceof MutationGateError && error.code === "U6_MUTANT_SET_MISMATCH",
  );
  const duplicate = Object.freeze([...U6_MUTANTS.slice(0, -1), U6_MUTANTS.at(-2)]);
  assert.throws(
    () => assertU6MutantDefinitionSet(duplicate),
    (error) => error instanceof MutationGateError && error.code === "U6_MUTANT_SET_MISMATCH",
  );
  const tampered = U6_MUTANTS.map((mutant) => Object.freeze({ ...mutant }));
  tampered[0] = Object.freeze({ ...tampered[0], replace: `${tampered[0].replace}\n// weakened` });
  assert.throws(
    () => assertU6MutantDefinitionSet(Object.freeze(tampered)),
    (error) => error instanceof MutationGateError && error.code === "U6_MUTANT_CONTRACT_MISMATCH",
  );
  const extra = U6_MUTANTS.map((mutant, index) => Object.freeze(index === 0 ? { ...mutant, extra: true } : { ...mutant }));
  assert.throws(
    () => assertU6MutantDefinitionSet(Object.freeze(extra)),
    (error) => error instanceof MutationGateError && error.code === "U6_MUTANT_CONTRACT_MISMATCH",
  );
  assert.throws(
    () => assertU6MutantDefinitionSet(U6_MUTANTS.map((mutant) => ({ ...mutant }))),
    (error) => error instanceof MutationGateError && error.code === "U6_MUTANT_SET_NOT_IMMUTABLE",
  );

  const phases = [];
  let invocation = 0;
  const killed = await runU2Mutant(U6_MUTANTS[0], {}, {
    async prepareExperiment({ phase }) {
      phases.push(phase);
      return { sandbox: `/fresh/u6/${phase}`, sandboxParent: `/fresh/u6/${phase}` };
    },
    async cleanupExperiment() {},
    async applyMutation() {},
    async runNamedTest() {
      return fakeClassification([
        NamedTestOutcome.Pass,
        NamedTestOutcome.Failure,
        NamedTestOutcome.Pass,
      ][invocation++]);
    },
  });
  assert.match(killed, /^KILLED publish-before-cas-compare/u);
  assert.deepEqual(phases, ["baseline", "mutant", "post-control"]);

  const seed = { manifest: { digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" } };
  const records = [
    { phase: "baseline", manifest: seed.manifest.digest, sandbox: "/fresh/u6/a" },
    { phase: "mutant", manifest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", sandbox: "/fresh/u6/b" },
    { phase: "post-control", manifest: seed.manifest.digest, sandbox: "/fresh/u6/c" },
  ];
  assert.match(exactU6ABADigest(U6_MUTANTS[0], records, seed), /^sha256:[0-9a-f]{64}$/u);
  return "U6 mutation self-test passed: 17/17 recursively frozen tuples; exact anchors/tests/manifests; inherited hostile-copy tripwires; fresh synthetic A/B/A and receipt closure.";
}

export async function main(arguments_ = process.argv.slice(2)) {
  if (arguments_.length === 1 && arguments_[0] === "--self-test") {
    process.stdout.write(`${await selfTest()}\n`);
    return;
  }
  if (arguments_.length !== 0) {
    throw new MutationGateError("U6_UNSUPPORTED_ARGUMENT", "usage: node tools/mutate-u6.mjs [--self-test]");
  }

  assertU6MutantDefinitionSet();
  await assertU6ManifestMatchesTree(repoRoot);
  await assertReviewedU6Anchors(repoRoot);
  await assertReviewedU6NamedTests(repoRoot);
  const toolchain = await admitU2DarwinToolchain();
  const seed = await createU6SeedSnapshot();
  const results = [];
  try {
    for (const mutant of U6_MUTANTS) {
      const phaseRecords = [];
      const result = await runU2Mutant(mutant, toolchain, {
        prepareExperiment({ mutant: currentMutant, toolchain: currentToolchain, phase }) {
          return prepareExperiment({ mutant: currentMutant, toolchain: currentToolchain, phase, seed, phaseRecords });
        },
        cleanupExperiment,
        applyMutation: mutateRegularFileNoFollow,
        runNamedTest(context, currentMutant) {
          return runNamedTest(context, currentMutant, toolchain);
        },
      });
      results.push(Object.freeze({ result, receipt: exactU6ABADigest(mutant, phaseRecords, seed) }));
    }
    await assertU5SourceAndSeedUnchanged(seed);
  } finally {
    await cleanupU5SeedSnapshot(seed);
  }
  revalidateToolchain(toolchain);

  process.stdout.write(`Trusted Go realpath: ${toolchain.executable.path}\n`);
  process.stdout.write(`Trusted Go sha256: ${toolchain.executable.sha256}\n`);
  process.stdout.write(`Trusted Go version: ${toolchain.version.line}\n`);
  process.stdout.write(`Trusted C compiler realpath: ${toolchain.compiler.path}\n`);
  process.stdout.write(`Trusted C compiler sha256: ${toolchain.compiler.sha256}\n`);
  process.stdout.write(`Closed Darwin CGO facts digest: ${toolchain.cgo.digest}\n`);
  process.stdout.write(`Immutable U6 seed manifest: ${seed.manifest.digest}\n`);
  process.stdout.write("Containment notice: private mutation copies retain the invoking user's host filesystem and network authority.\n");
  for (const item of results) process.stdout.write(`${item.result}; exact fresh A/B/A closure digest ${item.receipt}\n`);
  process.stdout.write(`U6 mutation gate: ${results.length}/17 required mutants killed; ${results.length}/17 fresh A/B/A receipts.\n`);
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(modulePath)) {
  main().catch((error) => {
    process.stderr.write(`${error.stack ?? error}\n`);
    process.exitCode = 1;
  });
}
