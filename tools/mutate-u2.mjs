#!/usr/bin/env node

import { createHash } from "node:crypto";
import {
  accessSync,
  closeSync,
  fstatSync,
  lstatSync,
  openSync,
  readFileSync,
} from "node:fs";
import {
  chmod,
  lstat,
  mkdir,
  mkdtemp,
  readFile,
  readdir,
  realpath,
  rm,
} from "node:fs/promises";
import { constants } from "node:fs";
import { tmpdir } from "node:os";
import {
  dirname,
  isAbsolute,
  join,
  relative as relativePath,
  resolve,
  sep,
} from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

import {
  MutationGateError,
  NamedTestOutcome,
  admitTrustedGoToolchain,
  applyExactMutation,
  classifyNamedGoTest,
  copyRegularAllowlist,
  mutateRegularFileNoFollow,
  readAllowlistManifest,
  revalidateAdmittedGoExecutable,
  runNamedGoTest,
} from "./mutate-u1.mjs";

export {
  MutationGateError,
  NamedTestOutcome,
  applyExactMutation,
  classifyNamedGoTest,
  copyRegularAllowlist,
  mutateRegularFileNoFollow,
};

const modulePath = fileURLToPath(import.meta.url);
const repoRoot = resolve(dirname(modulePath), "..");
const moduleImportPath = "github.com/nelsonwerd/countershape";

function sha256(bytes) {
  return createHash("sha256").update(bytes).digest("hex");
}

// This is a positive execution manifest. U2 mutation experiments copy no VCS,
// didrun, credential, prompt, or unrelated product files. A new file in any U2
// source scope closes the gate until this exact list receives review.
export const U2_SANDBOX_FILE_ALLOWLIST = Object.freeze([
  "go.mod",
  "internal/canon/canon_test.go",
  "internal/canon/digest.go",
  "internal/canon/doc.go",
  "internal/canon/encode.go",
  "internal/canon/error.go",
  "internal/canon/fuzz_test.go",
  "internal/canon/limits.go",
  "internal/canon/limits_test.go",
  "internal/canon/parse.go",
  "internal/canon/scan.go",
  "internal/canon/token.go",
  "internal/canon/typed.go",
  "internal/canon/value.go",
  "internal/domain/domain_test.go",
  "internal/domain/envelope.go",
  "internal/domain/errors.go",
  "internal/domain/identity.go",
  "internal/domain/projection.go",
  "internal/domain/receipt.go",
  "internal/domain/transitions.go",
  "internal/domain/world.go",
  "internal/gitobj/errors.go",
  "internal/gitobj/filesystem.go",
  "internal/gitobj/filesystem_darwin.go",
  "internal/gitobj/filesystem_unsupported.go",
  "internal/gitobj/fuzz_test.go",
  "internal/gitobj/git.go",
  "internal/gitobj/helpers_external_test.go",
  "internal/gitobj/identity_external_test.go",
  "internal/gitobj/inspect.go",
  "internal/gitobj/inspect_test.go",
  "internal/gitobj/materialize.go",
  "internal/gitobj/materialize_external_test.go",
  "internal/gitobj/mutation_contract_test.go",
  "internal/gitobj/publish_darwin.go",
  "internal/gitobj/publish_darwin_test.go",
  "internal/gitobj/publish_unsupported.go",
  "internal/gitobj/refusal_external_test.go",
  "internal/gitobj/types.go",
  "internal/gitobj/validate.go",
  "internal/world/allocate.go",
  "internal/world/api.go",
  "internal/world/capture.go",
  "internal/world/errors.go",
  "internal/world/execute_darwin_test.go",
  "internal/world/finalize.go",
  "internal/world/process.go",
  "internal/world/process_darwin.go",
  "internal/world/process_darwin_test.go",
  "internal/world/process_mutation_darwin_test.go",
  "internal/world/process_unsupported.go",
  "internal/world/receipt.go",
  "internal/world/tools.go",
  "testkit/gitrepo/gitrepo.go",
  "testkit/processfixture/main.go",
]);

export const U2_SOURCE_SCOPE_ROOTS = Object.freeze([
  "internal/canon",
  "internal/domain",
  "internal/gitobj",
  "internal/world",
  "testkit/gitrepo",
  "testkit/processfixture",
]);

// Independent frozen ID and tuple contracts make a smaller, renamed, or
// silently weakened mutant set fail before any test process executes.
export const REQUIRED_U2_MUTANT_IDS = Object.freeze([
  "enable-replacement-refs",
  "allow-lazy-fetch",
  "allow-alternate-object-directory",
  "skip-streamed-object-rehash",
  "accept-sha256-as-sha1-width",
  "accept-symlink-mode",
  "collapse-executable-mode",
  "accept-lfs-pointer",
  "validate-path-after-blob-read",
  "lexical-only-path-collision",
  "allow-duplicate-portable-tree",
  "allow-destination-overwrite",
  "inherit-ambient-environment",
  "resolve-tool-through-path",
  "spawn-through-shell",
  "skip-marker-before-spawn-gate",
  "share-stdout-limit-with-stderr",
  "map-overflow-to-success",
  "signal-direct-pid",
  "remove-kill-escalation",
  "report-drain-timeout-complete",
  "skip-final-group-probe",
  "late-control-overwrites-earlier-terminal",
  "claim-process-escape-containment",
  "pollute-preplan-candidate-set-digest",
]);

export const REVIEWED_U2_MUTANT_CONTRACT = Object.freeze([
  Object.freeze({
    id: "enable-replacement-refs",
    file: "internal/gitobj/git.go",
    find: "disableReplacementInterpretation  = true // MUTANT_U2_ENABLE_REPLACEMENT_REFS",
    replace: "disableReplacementInterpretation  = false // MUTANT_U2_ENABLE_REPLACEMENT_REFS",
    package: "./internal/gitobj",
    testName: "TestReplacementRefsNeverAffectPinnedIdentity",
  }),
  Object.freeze({
    id: "allow-lazy-fetch",
    file: "internal/gitobj/git.go",
    find: "rejectPromisorAndDisableLazyFetch = true // MUTANT_U2_ALLOW_LAZY_FETCH",
    replace: "rejectPromisorAndDisableLazyFetch = false // MUTANT_U2_ALLOW_LAZY_FETCH",
    package: "./internal/gitobj",
    testName: "TestLazyFetchEnvironmentIsDisabled",
  }),
  Object.freeze({
    id: "allow-alternate-object-directory",
    file: "internal/gitobj/git.go",
    find: "rejectAlternateObjectDirectories  = true // MUTANT_U2_ALLOW_ALTERNATE_OBJECTS",
    replace: "rejectAlternateObjectDirectories  = false // MUTANT_U2_ALLOW_ALTERNATE_OBJECTS",
    package: "./internal/gitobj",
    testName: "TestAlternateObjectDirectoryIsRejectedBeforeObjectRead",
  }),
  Object.freeze({
    id: "skip-streamed-object-rehash",
    file: "internal/gitobj/inspect.go",
    find: "if computed != expected { // MUTANT_U2_SKIP_BLOB_REHASH",
    replace: "if false && computed != expected { // MUTANT_U2_SKIP_BLOB_REHASH",
    package: "./internal/gitobj",
    testName: "TestBlobRehashRejectsTamperedGitStream",
  }),
  Object.freeze({
    id: "accept-sha256-as-sha1-width",
    file: "internal/gitobj/types.go",
    find: "return 32 // MUTANT_U2_ACCEPT_SHA1_WIDTH_FOR_SHA256",
    replace: "return 20 // MUTANT_U2_ACCEPT_SHA1_WIDTH_FOR_SHA256",
    package: "./internal/gitobj",
    testName: "TestObjectFormatOIDWidthsAreClosed",
  }),
  Object.freeze({
    id: "accept-symlink-mode",
    file: "internal/gitobj/inspect.go",
    find: "acceptSymlinkMode                  = false // MUTANT_U2_ACCEPT_SYMLINK_MODE",
    replace: "acceptSymlinkMode                  = true // MUTANT_U2_ACCEPT_SYMLINK_MODE",
    package: "./internal/gitobj",
    testName: "TestSymlinkModeIsRejected",
  }),
  Object.freeze({
    id: "collapse-executable-mode",
    file: "internal/gitobj/inspect.go",
    find: "preserveExecutableMode             = true  // MUTANT_U2_COLLAPSE_EXECUTABLE_MODE",
    replace: "preserveExecutableMode             = false // MUTANT_U2_COLLAPSE_EXECUTABLE_MODE",
    package: "./internal/gitobj",
    testName: "TestExecutableModeRemainsIdentityBearing",
  }),
  Object.freeze({
    id: "accept-lfs-pointer",
    file: "internal/gitobj/inspect.go",
    find: "rejectLFSPointerContent            = true  // MUTANT_U2_ACCEPT_LFS_POINTER",
    replace: "rejectLFSPointerContent            = false // MUTANT_U2_ACCEPT_LFS_POINTER",
    package: "./internal/gitobj",
    testName: "TestLFSPointerIsRejected",
  }),
  Object.freeze({
    id: "validate-path-after-blob-read",
    file: "internal/gitobj/inspect.go",
    find: "validatePathsBeforeCandidateBytes  = true  // MUTANT_U2_VALIDATE_PATH_AFTER_WRITE",
    replace: "validatePathsBeforeCandidateBytes  = false // MUTANT_U2_VALIDATE_PATH_AFTER_WRITE",
    package: "./internal/gitobj",
    testName: "TestUnsafePathRefusesBeforeCandidateBlobRead",
  }),
  Object.freeze({
    id: "lexical-only-path-collision",
    file: "internal/gitobj/inspect.go",
    find: "requireTargetFilesystemReservation = true  // MUTANT_U2_LEXICAL_ONLY_PATH_COLLISION",
    replace: "requireTargetFilesystemReservation = false // MUTANT_U2_LEXICAL_ONLY_PATH_COLLISION",
    package: "./internal/gitobj",
    testName: "TestTargetFilesystemReservationCannotBeReplacedByLexicalCollisionCheck",
  }),
  Object.freeze({
    id: "allow-duplicate-portable-tree",
    file: "internal/gitobj/types.go",
    find: "if _, duplicate := seenPortable[candidate.portableTreeDigest]; duplicate { // MUTANT_U2_ALLOW_DUPLICATE_PORTABLE_TREE",
    replace: "if _, duplicate := seenPortable[candidate.portableTreeDigest]; false && duplicate { // MUTANT_U2_ALLOW_DUPLICATE_PORTABLE_TREE",
    package: "./internal/gitobj",
    testName: "TestCandidateSetRejectsDuplicatePortableTreeAcrossObjectFormats",
  }),
  Object.freeze({
    id: "allow-destination-overwrite",
    file: "internal/gitobj/publish_darwin.go",
    find: "const exclusivePublicationRequired = true // MUTANT_U2_ALLOW_DESTINATION_OVERWRITE",
    replace: "const exclusivePublicationRequired = false // MUTANT_U2_ALLOW_DESTINATION_OVERWRITE",
    package: "./internal/gitobj",
    testName: "TestRenameExclusiveDoesNotOverwriteExistingDestination",
  }),
  Object.freeze({
    id: "inherit-ambient-environment",
    file: "internal/world/allocate.go",
    find: "inheritAmbientEnvironment = false // MUTANT_U2_INHERIT_AMBIENT_ENVIRONMENT",
    replace: "inheritAmbientEnvironment = true // MUTANT_U2_INHERIT_AMBIENT_ENVIRONMENT",
    package: "./internal/world",
    testName: "TestBuildEnvironmentDoesNotInheritAmbientVariables",
  }),
  Object.freeze({
    id: "resolve-tool-through-path",
    file: "internal/world/tools.go",
    find: "const resolveToolsThroughAmbientPATH = false // MUTANT_U2_RESOLVE_TOOL_THROUGH_PATH",
    replace: "const resolveToolsThroughAmbientPATH = true // MUTANT_U2_RESOLVE_TOOL_THROUGH_PATH",
    package: "./internal/world",
    testName: "TestToolRegistryNeverConsultsPATHAndPinsMeasuredBytes",
  }),
  Object.freeze({
    id: "spawn-through-shell",
    file: "internal/world/process_darwin.go",
    find: "directExecOnly                 = true // MUTANT_U2_SPAWN_THROUGH_SHELL",
    replace: "directExecOnly                 = false // MUTANT_U2_SPAWN_THROUGH_SHELL",
    package: "./internal/world",
    testName: "TestDirectCommandContainsNoShellOrAmbientCommandResolution",
  }),
  Object.freeze({
    id: "skip-marker-before-spawn-gate",
    file: "internal/world/api.go",
    find: "const requireDurableMarkerBeforeSpawn = true // MUTANT_U2_CREATE_MARKER_AFTER_SPAWN",
    replace: "const requireDurableMarkerBeforeSpawn = false // MUTANT_U2_CREATE_MARKER_AFTER_SPAWN",
    package: "./internal/world",
    testName: "TestExecuteRefusesSpawnWhenDurableMarkerChanges",
  }),
  Object.freeze({
    id: "share-stdout-limit-with-stderr",
    file: "internal/world/process_darwin.go",
    find: "stderrCapture := newCappedCapture(request.stderrLimit, overflowC) // MUTANT_U2_SHARE_OUTPUT_CAP",
    replace: "stderrCapture := newCappedCapture(request.stdoutLimit, overflowC) // MUTANT_U2_SHARE_OUTPUT_CAP",
    package: "./internal/world",
    testName: "TestStdoutAndStderrLimitsAreIndependentMutationGuard",
  }),
  Object.freeze({
    id: "map-overflow-to-success",
    file: "internal/world/process_darwin.go",
    find: "outputOverflowIsPrimaryControl = true // MUTANT_U2_MAP_OUTPUT_LIMIT_TO_SUCCESS",
    replace: "outputOverflowIsPrimaryControl = false // MUTANT_U2_MAP_OUTPUT_LIMIT_TO_SUCCESS",
    package: "./internal/world",
    testName: "TestOutputOverflowCannotBecomeSuccessfulTruncation",
  }),
  Object.freeze({
    id: "signal-direct-pid",
    file: "internal/world/process_darwin.go",
    find: "signalOwnedProcessGroup        = true // MUTANT_U2_SIGNAL_DIRECT_PID",
    replace: "signalOwnedProcessGroup        = false // MUTANT_U2_SIGNAL_DIRECT_PID",
    package: "./internal/world",
    testName: "TestNegativePGIDSignalCleansDescendantsAfterParentExit",
  }),
  Object.freeze({
    id: "remove-kill-escalation",
    file: "internal/world/process_darwin.go",
    find: "killEscalationRequired         = true // MUTANT_U2_REMOVE_KILL_ESCALATION",
    replace: "killEscalationRequired         = false // MUTANT_U2_REMOVE_KILL_ESCALATION",
    package: "./internal/world",
    testName: "TestTermIgnoringProcessRequiresKillEscalation",
  }),
  Object.freeze({
    id: "report-drain-timeout-complete",
    file: "internal/world/process_darwin.go",
    find: "drainDeadlineIsFailure         = true // MUTANT_U2_REPORT_DRAIN_TIMEOUT_COMPLETE",
    replace: "drainDeadlineIsFailure         = false // MUTANT_U2_REPORT_DRAIN_TIMEOUT_COMPLETE",
    package: "./internal/world",
    testName: "TestDrainDeadlineReportsIncompleteDrain",
  }),
  Object.freeze({
    id: "skip-final-group-probe",
    file: "internal/world/process_darwin.go",
    find: "finalGroupProbeRequired        = true // MUTANT_U2_REMOVE_FINAL_GROUP_PROBE",
    replace: "finalGroupProbeRequired        = false // MUTANT_U2_REMOVE_FINAL_GROUP_PROBE",
    package: "./internal/world",
    testName: "TestFinalGroupProbeRequiresObservedAbsence",
  }),
  Object.freeze({
    id: "late-control-overwrites-earlier-terminal",
    file: "internal/world/process_darwin.go",
    find: "ownerPriorityOutputFirst       = true // MUTANT_U2_LATE_CONTROL_OVERRIDES_OUTPUT",
    replace: "ownerPriorityOutputFirst       = false // MUTANT_U2_LATE_CONTROL_OVERRIDES_OUTPUT",
    package: "./internal/world",
    testName: "TestTerminalArbiterUsesFixedOwnerObservedPriority",
  }),
  Object.freeze({
    id: "claim-process-escape-containment",
    file: "internal/world/process.go",
    find: "const processEscapeExclusion = \"PROCESS_GROUP_OR_SESSION_ESCAPE_EXCLUDED_FROM_CONTAINMENT_CLAIM\" // MUTANT_U2_CLAIM_PROCESS_ESCAPE_CONTAINMENT",
    replace: "const processEscapeExclusion = \"PROCESS_GROUP_OR_SESSION_ESCAPE_CONTAINED\" // MUTANT_U2_CLAIM_PROCESS_ESCAPE_CONTAINMENT",
    package: "./internal/world",
    testName: "TestProcessEscapeBoundaryCannotClaimContainment",
  }),
  Object.freeze({
    id: "pollute-preplan-candidate-set-digest",
    file: "internal/gitobj/types.go",
    find: [
      "func (s CandidateSetDeclaration) Digest() domain.Digest { // MUTANT_U2_POLLUTE_PREPLAN_DIGEST",
      "\treturn s.selected.digest",
      "}",
    ].join("\n"),
    replace: [
      "func (s CandidateSetDeclaration) Digest() domain.Digest { // MUTANT_U2_POLLUTE_PREPLAN_DIGEST",
      "\treturn s.policy.digest",
      "}",
    ].join("\n"),
    package: "./internal/gitobj",
    testName: "TestPrePlanCandidateSetDigestExcludesRepositoryAndPostPlanAuthority",
  }),
]);

export const U2_MUTANTS = Object.freeze(
  REVIEWED_U2_MUTANT_CONTRACT.map((mutant) => Object.freeze({ ...mutant })),
);

const tupleFields = Object.freeze(["file", "find", "id", "package", "replace", "testName"]);

function sameArray(left, right) {
  return left.length === right.length && left.every((entry, index) => entry === right[index]);
}

function exactOccurrenceCount(source, anchor) {
  if (typeof anchor !== "string" || anchor.length === 0) return 0;
  return source.split(anchor).length - 1;
}

export function assertU2MutantDefinitionSet(
  mutants = U2_MUTANTS,
  requiredIDs = REQUIRED_U2_MUTANT_IDS,
  reviewedContract = REVIEWED_U2_MUTANT_CONTRACT,
) {
  if (requiredIDs.length !== 25 || new Set(requiredIDs).size !== 25) {
    throw new MutationGateError("INVALID_REQUIRED_ID_SET", "U2 requires exactly twenty-five unique frozen IDs");
  }
  if (mutants.length !== 25) {
    throw new MutationGateError("MUTANT_SET_MISMATCH", `defined ${mutants.length} U2 mutants, want 25`);
  }
  const ids = mutants.map((mutant) => mutant.id);
  if (new Set(ids).size !== ids.length) {
    throw new MutationGateError("DUPLICATE_MUTANT_ID", "U2 mutant IDs must be unique");
  }
  if (!sameArray(ids, requiredIDs)) {
    throw new MutationGateError("MUTANT_SET_MISMATCH", "U2 mutant IDs or their reviewed order changed");
  }
  if (reviewedContract.length !== 25 || !sameArray(reviewedContract.map((tuple) => tuple.id), requiredIDs)) {
    throw new MutationGateError("INVALID_REVIEWED_MUTANT_CONTRACT", "the independent U2 tuple contract changed shape or order");
  }
  const reviewedByID = new Map(reviewedContract.map((tuple) => [tuple.id, tuple]));
  const allowedFiles = new Set(U2_SANDBOX_FILE_ALLOWLIST);
  for (const mutant of mutants) {
    const reviewed = reviewedByID.get(mutant.id);
    const fields = Object.keys(mutant).sort();
    if (!reviewed || !sameArray(fields, tupleFields)) {
      throw new MutationGateError("MUTANT_CONTRACT_MISMATCH", `${mutant.id} differs from the reviewed tuple shape`);
    }
    for (const field of tupleFields) {
      if (mutant[field] !== reviewed[field]) {
        throw new MutationGateError("MUTANT_CONTRACT_MISMATCH", `${mutant.id}.${field} differs from the reviewed tuple`);
      }
    }
    if (!allowedFiles.has(mutant.file)) {
      throw new MutationGateError("MUTANT_FILE_NOT_ALLOWLISTED", `${mutant.id} targets ${mutant.file}`);
    }
    if (mutant.find.length === 0 || mutant.find === mutant.replace || !mutant.find.includes("MUTANT_U2_")) {
      throw new MutationGateError("INVALID_MUTATION", `${mutant.id} is not one exact effective U2 anchor replacement`);
    }
    if (!/^\.\/internal\/(?:gitobj|world)$/u.test(mutant.package)) {
      throw new MutationGateError("INVALID_MUTATION_PACKAGE", `${mutant.id} has invalid package ${mutant.package}`);
    }
    if (!/^Test[A-Za-z0-9_]+$/u.test(mutant.testName)) {
      throw new MutationGateError("INVALID_MUTATION_TEST", `${mutant.id} has invalid named test ${mutant.testName}`);
    }
  }
}

function assertSafeRelativeFile(relativeFile) {
  if (
    typeof relativeFile !== "string" || relativeFile.length === 0 || isAbsolute(relativeFile) ||
    relativeFile.split(/[\\/]/u).some((part) => part === "" || part === "." || part === "..")
  ) {
    throw new MutationGateError("UNSAFE_ALLOWLIST_PATH", `invalid relative file ${JSON.stringify(relativeFile)}`);
  }
}

async function enumerateSourceScope(sourceRoot, relativeRoot) {
  assertSafeRelativeFile(relativeRoot);
  const absoluteRoot = resolve(sourceRoot, relativeRoot);
  const fromSource = relativePath(sourceRoot, absoluteRoot);
  if (fromSource === ".." || fromSource.startsWith(`..${sep}`) || isAbsolute(fromSource)) {
    throw new MutationGateError("MANIFEST_SCOPE_ESCAPE", `${relativeRoot} escapes the source root`);
  }
  let rootMetadata;
  try {
    rootMetadata = await lstat(absoluteRoot);
  } catch (error) {
    throw new MutationGateError("MANIFEST_SCOPE_MISSING", `${relativeRoot}: ${error.code ?? error.message}`);
  }
  if (rootMetadata.isSymbolicLink() || !rootMetadata.isDirectory()) {
    throw new MutationGateError("MANIFEST_SCOPE_NONREGULAR", `${relativeRoot} must be a real directory`);
  }
  const files = [];
  async function visit(directory) {
    const entries = await readdir(directory, { withFileTypes: true });
    entries.sort((left, right) => left.name.localeCompare(right.name, "en"));
    for (const entry of entries) {
      const absolute = join(directory, entry.name);
      const relative = relativePath(sourceRoot, absolute).split(sep).join("/");
      if (entry.isSymbolicLink()) {
        throw new MutationGateError("MANIFEST_TREE_SYMLINK", `${relative} is a symbolic link`);
      }
      if (entry.isDirectory()) {
        await visit(absolute);
      } else if (!entry.isFile()) {
        throw new MutationGateError("MANIFEST_TREE_NONREGULAR", `${relative} is not a regular file`);
      } else if (!relative.endsWith(".go")) {
        throw new MutationGateError("MANIFEST_TREE_UNSUPPORTED_FILE", `${relative} is an unreviewed non-Go artifact`);
      } else {
        files.push(relative);
      }
    }
  }
  await visit(absoluteRoot);
  return files;
}

export async function assertU2ManifestMatchesTree(
  sourceRoot,
  allowlist = U2_SANDBOX_FILE_ALLOWLIST,
  scopeRoots = U2_SOURCE_SCOPE_ROOTS,
) {
  if (allowlist.length === 0 || new Set(allowlist).size !== allowlist.length) {
    throw new MutationGateError("INVALID_FILE_ALLOWLIST", "U2 file allowlist must be nonempty and unique");
  }
  for (const file of allowlist) assertSafeRelativeFile(file);
  if (scopeRoots.length === 0 || new Set(scopeRoots).size !== scopeRoots.length) {
    throw new MutationGateError("INVALID_SOURCE_SCOPES", "U2 source scopes must be nonempty and unique");
  }
  const support = allowlist.filter((file) => !scopeRoots.some((root) => file.startsWith(`${root}/`))).sort();
  if (!sameArray(support, ["go.mod"])) {
    throw new MutationGateError("MANIFEST_SUPPORT_SET_MISMATCH", `support files [${support.join(",")}] differ from go.mod`);
  }
  const actual = [];
  for (const scope of scopeRoots) actual.push(...await enumerateSourceScope(sourceRoot, scope));
  actual.sort();
  const expected = allowlist.filter((file) => file !== "go.mod").sort();
  const actualSet = new Set(actual);
  const expectedSet = new Set(expected);
  const unlisted = actual.filter((file) => !expectedSet.has(file));
  const missing = expected.filter((file) => !actualSet.has(file));
  if (unlisted.length > 0 || missing.length > 0) {
    throw new MutationGateError("MANIFEST_TREE_MISMATCH", `unlisted=[${unlisted.join(",")}] missing=[${missing.join(",")}]`);
  }
}

export async function assertReviewedU2Anchors(sourceRoot = repoRoot, mutants = U2_MUTANTS) {
  assertU2MutantDefinitionSet(mutants);
  const allMarkers = [];
  for (const file of U2_SANDBOX_FILE_ALLOWLIST.filter((entry) => entry.endsWith(".go"))) {
    const source = await readFile(resolve(sourceRoot, file), "utf8");
    allMarkers.push(...(source.match(/MUTANT_U2_[A-Z0-9_]+/gu) ?? []));
  }
  const reviewedMarkers = mutants.map((mutant) => /MUTANT_U2_[A-Z0-9_]+/u.exec(mutant.find)?.[0]);
  if (
    allMarkers.length !== 25 || new Set(allMarkers).size !== 25 ||
    reviewedMarkers.some((marker) => !marker) || new Set(reviewedMarkers).size !== 25 ||
    [...new Set(allMarkers)].sort().join("\n") !== [...new Set(reviewedMarkers)].sort().join("\n")
  ) {
    throw new MutationGateError("U2_ANCHOR_SET_MISMATCH", "source and reviewed contract must contain the same 25 unique U2 anchors");
  }
  for (const mutant of mutants) {
    const source = await readFile(resolve(sourceRoot, mutant.file), "utf8");
    const count = exactOccurrenceCount(source, mutant.find);
    if (count !== 1) {
      throw new MutationGateError("MUTATION_ANCHOR_COUNT", `${mutant.id}: exact anchor count was ${count}, want 1`);
    }
  }
}

function manifestEntryMap(manifest) {
  return new Map(manifest.entries.map((entry) => [entry.file, entry]));
}

function requireManifestEqual(actual, expected, code, detail) {
  if (actual.digest !== expected.digest) {
    throw new MutationGateError(code, `${detail}: ${actual.digest} != ${expected.digest}`);
  }
}

async function validateTreeAndReadManifest(root, allowlist, scopeRoots) {
  await assertU2ManifestMatchesTree(root, allowlist, scopeRoots);
  return readAllowlistManifest(root, allowlist);
}

export async function createU2SeedSnapshot({
  sourceRoot = repoRoot,
  allowlist = U2_SANDBOX_FILE_ALLOWLIST,
  scopeRoots = U2_SOURCE_SCOPE_ROOTS,
} = {}) {
  const resolvedSource = resolve(sourceRoot);
  const frozenAllowlist = Object.freeze([...allowlist]);
  const frozenScopes = Object.freeze([...scopeRoots]);
  const sourceManifest = await validateTreeAndReadManifest(resolvedSource, frozenAllowlist, frozenScopes);
  const parent = await mkdtemp(join(tmpdir(), "countershape-u2-seed-"));
  await chmod(parent, 0o700);
  const root = join(parent, "repo");
  try {
    await copyRegularAllowlist(resolvedSource, root, frozenAllowlist);
    const manifest = await validateTreeAndReadManifest(root, frozenAllowlist, frozenScopes);
    requireManifestEqual(manifest, sourceManifest, "SEED_COPY_MISMATCH", "immutable seed differs from its admitted source");
    return Object.freeze({
      sourceRoot: resolvedSource,
      sourceManifest,
      parent,
      root,
      manifest,
      allowlist: frozenAllowlist,
      scopeRoots: frozenScopes,
    });
  } catch (error) {
    await rm(parent, { recursive: true, force: true });
    throw error;
  }
}

export async function cleanupU2SeedSnapshot(seed) {
  await rm(seed.parent, { recursive: true, force: true });
}

export async function assertU2SourceAndSeedUnchanged(seed) {
  const seedManifest = await validateTreeAndReadManifest(seed.root, seed.allowlist, seed.scopeRoots);
  requireManifestEqual(seedManifest, seed.manifest, "SEED_SNAPSHOT_CHANGED", "immutable U2 seed changed");
  const sourceManifest = await validateTreeAndReadManifest(seed.sourceRoot, seed.allowlist, seed.scopeRoots);
  requireManifestEqual(sourceManifest, seed.sourceManifest, "ADMITTED_SOURCE_CHANGED", "admitted U2 source changed after seed capture");
}

export async function assertExactU2MutantDelta(sandboxRoot, seed, mutant) {
  const actual = await validateTreeAndReadManifest(sandboxRoot, seed.allowlist, seed.scopeRoots);
  const expectedEntries = manifestEntryMap(seed.manifest);
  const actualEntries = manifestEntryMap(actual);
  const changed = [];
  for (const [file, expected] of expectedEntries) {
    const candidate = actualEntries.get(file);
    if (!candidate || candidate.bytes !== expected.bytes || candidate.sha256 !== expected.sha256) changed.push(file);
  }
  for (const file of actualEntries.keys()) if (!expectedEntries.has(file)) changed.push(file);
  changed.sort();
  if (changed.length !== 1 || changed[0] !== mutant.file) {
    throw new MutationGateError("MUTANT_DELTA_SCOPE", `${mutant.id}: changed [${changed.join(",")}], want exactly ${mutant.file}`);
  }
  const seedBytes = await readFile(resolve(seed.root, mutant.file));
  const expectedBytes = Buffer.from(applyExactMutation(seedBytes.toString("utf8"), mutant), "utf8");
  const actualBytes = await readFile(resolve(sandboxRoot, mutant.file));
  if (!actualBytes.equals(expectedBytes)) {
    throw new MutationGateError("MUTANT_DELTA_BYTES", `${mutant.id}: target bytes differ from the reviewed exact replacement`);
  }
  return actual;
}

function nativeMachOFormat(bytes) {
  if (bytes.length < 4) return false;
  return new Set([
    "feedface", "cefaedfe", "feedfacf", "cffaedfe",
    "cafebabe", "bebafeca", "cafebabf", "bfbafeca",
  ]).has(bytes.subarray(0, 4).toString("hex"));
}

function fingerprintDarwinCompilerSync(executablePath) {
  if (process.platform !== "darwin") {
    throw new MutationGateError("U2_PLATFORM_UNSUPPORTED", `U2 requires Darwin, got ${process.platform}`);
  }
  if (typeof executablePath !== "string" || !isAbsolute(executablePath)) {
    throw new MutationGateError("CC_EXECUTABLE_NOT_ABSOLUTE", "compiler admission requires an absolute path");
  }
  if (!Number.isInteger(constants.O_NOFOLLOW)) {
    throw new MutationGateError("CC_EXECUTABLE_NOFOLLOW_UNAVAILABLE", "O_NOFOLLOW is required for compiler admission");
  }
  let descriptor;
  try {
    const admitted = lstatSync(executablePath);
    accessSync(executablePath, constants.X_OK);
    if (admitted.isSymbolicLink() || !admitted.isFile()) {
      throw new MutationGateError("CC_EXECUTABLE_NOT_ADMITTED", `${executablePath} is not an executable regular file`);
    }
    descriptor = openSync(executablePath, constants.O_RDONLY | constants.O_NOFOLLOW);
    const opened = fstatSync(descriptor);
    if (!opened.isFile() || opened.dev !== admitted.dev || opened.ino !== admitted.ino) {
      throw new MutationGateError("CC_EXECUTABLE_CHANGED", `${executablePath} changed during no-follow admission`);
    }
    const bytes = readFileSync(descriptor);
    if (!nativeMachOFormat(bytes)) {
      throw new MutationGateError("CC_EXECUTABLE_BINARY_FORMAT", `${executablePath} is not a native Mach-O executable`);
    }
    return Object.freeze({
      path: executablePath,
      sha256: sha256(bytes),
      dev: opened.dev,
      ino: opened.ino,
      size: opened.size,
      mode: opened.mode & 0o7777,
      format: "MACH_O",
    });
  } catch (error) {
    if (error instanceof MutationGateError) throw error;
    throw new MutationGateError("CC_EXECUTABLE_NOT_ADMITTED", `${executablePath}: ${error.code ?? error.message}`);
  } finally {
    if (descriptor !== undefined) closeSync(descriptor);
  }
}

function sameCompilerFingerprint(left, right) {
  return left.path === right.path && left.sha256 === right.sha256 && left.dev === right.dev &&
    left.ino === right.ino && left.size === right.size && left.mode === right.mode && left.format === right.format;
}

export async function resolveAdmittedDarwinCCompiler(sourceEnvironment = process.env) {
  const declared = sourceEnvironment.COUNTERSHAPE_CC;
  if (typeof declared !== "string" || declared.length === 0) {
    throw new MutationGateError("CC_EXECUTABLE_NOT_DECLARED", "set COUNTERSHAPE_CC to one reviewed absolute Darwin C compiler");
  }
  if (!isAbsolute(declared)) {
    throw new MutationGateError("CC_EXECUTABLE_NOT_ABSOLUTE", "COUNTERSHAPE_CC must be absolute; PATH lookup is unsupported");
  }
  let admittedPath;
  try {
    admittedPath = await realpath(declared);
  } catch (error) {
    throw new MutationGateError("CC_EXECUTABLE_NOT_ADMITTED", `${declared}: ${error.code ?? error.message}`);
  }
  return fingerprintDarwinCompilerSync(admittedPath);
}

export function revalidateAdmittedDarwinCCompiler(admission) {
  if (!admission || typeof admission.path !== "string") {
    throw new MutationGateError("CC_EXECUTABLE_NOT_ADMITTED", "missing compiler admission record");
  }
  const current = fingerprintDarwinCompilerSync(admission.path);
  if (!sameCompilerFingerprint(current, admission)) {
    throw new MutationGateError("CC_EXECUTABLE_CHANGED", `${admission.path} no longer matches its admitted identity`);
  }
  return current;
}

function limitedOutput(result) {
  const output = `${result.stdout ?? ""}${result.stderr ?? ""}`.trim();
  return output.length > 4000 ? `${output.slice(0, 4000)}\n...[truncated]` : output;
}

function compilerInspectionEnvironment(compiler) {
  return Object.freeze({
    PATH: dirname(compiler.path),
    HOME: join(tmpdir(), "countershape-u2-cc-no-home"),
    TMPDIR: tmpdir(),
    LANG: "C",
    LC_ALL: "C",
    NO_COLOR: "1",
    TZ: "UTC",
  });
}

function spawnWithCompilerRevalidation(compiler, args, options, spawn) {
  revalidateAdmittedDarwinCCompiler(compiler);
  try {
    return spawn(compiler.path, args, options);
  } finally {
    revalidateAdmittedDarwinCCompiler(compiler);
  }
}

export function readAdmittedDarwinCCompilerFacts(compiler, spawn = spawnSync) {
  const options = {
    cwd: tmpdir(),
    encoding: "utf8",
    env: compilerInspectionEnvironment(compiler),
    timeout: 10_000,
    maxBuffer: 64 * 1024,
  };
  const versionResult = spawnWithCompilerRevalidation(compiler, ["--version"], options, spawn);
  const versionOutput = typeof versionResult.stdout === "string" ? versionResult.stdout.trim() : "";
  const firstLine = versionOutput.split(/\r?\n/u)[0] ?? "";
  if (
    versionResult.error || versionResult.signal || versionResult.status !== 0 ||
    (typeof versionResult.stderr === "string" && versionResult.stderr.trim() !== "") ||
    !/\bclang version\b/iu.test(firstLine)
  ) {
    throw new MutationGateError("CC_VERSION_INSPECTION_FAILED", `compiler did not prove a bounded clang identity: ${limitedOutput(versionResult)}`);
  }
  const targetResult = spawnWithCompilerRevalidation(compiler, ["-dumpmachine"], options, spawn);
  const target = typeof targetResult.stdout === "string" ? targetResult.stdout.trim() : "";
  const hostArchitecture = process.arch === "x64" ? "x86_64" : process.arch;
  if (
    targetResult.error || targetResult.signal || targetResult.status !== 0 ||
    (typeof targetResult.stderr === "string" && targetResult.stderr.trim() !== "") ||
    !new RegExp(`^${hostArchitecture}-apple-darwin[0-9.]*$`, "u").test(target)
  ) {
    throw new MutationGateError("CC_TARGET_INSPECTION_FAILED", `compiler target ${JSON.stringify(target)} does not match Darwin host ${hostArchitecture}`);
  }
  return Object.freeze({
    versionLine: firstLine,
    target,
    digest: `sha256:${sha256(Buffer.from(`${versionOutput}\n${target}\n`, "utf8"))}`,
  });
}

export function minimalU2GoEnvironment(cacheRoot, toolchain) {
  revalidateAdmittedGoExecutable(toolchain.executable);
  revalidateAdmittedDarwinCCompiler(toolchain.compiler);
  const toolDirectories = [...new Set([dirname(toolchain.executable.path), dirname(toolchain.compiler.path)])];
  return Object.freeze({
    PATH: toolDirectories.join(":"),
    HOME: join(cacheRoot, "home"),
    TMPDIR: join(cacheRoot, "tmp"),
    GOTMPDIR: join(cacheRoot, "go-tmp"),
    GOPATH: join(cacheRoot, "go-path"),
    GOCACHE: join(cacheRoot, "go-build"),
    GOMODCACHE: join(cacheRoot, "go-mod"),
    GOENV: "off",
    GOWORK: "off",
    GOTOOLCHAIN: "local",
    GOPROXY: "off",
    GOSUMDB: "off",
    GOFLAGS: "-mod=readonly -buildvcs=false",
    CGO_ENABLED: "1",
    CC: toolchain.compiler.path,
    CXX: toolchain.compiler.path,
    LANG: "C",
    LC_ALL: "C",
    NO_COLOR: "1",
    TZ: "UTC",
  });
}

function spawnWithClosedToolchain(toolchain, executable, args, options, spawn) {
  revalidateAdmittedGoExecutable(toolchain.executable);
  revalidateAdmittedDarwinCCompiler(toolchain.compiler);
  try {
    return spawn(executable, args, options);
  } finally {
    revalidateAdmittedDarwinCCompiler(toolchain.compiler);
    revalidateAdmittedGoExecutable(toolchain.executable);
  }
}

export function readAdmittedDarwinCGOFacts(toolchain, spawn = spawnSync) {
  const result = spawnWithClosedToolchain(
    toolchain,
    toolchain.executable.path,
    ["env", "-json", "GOOS", "GOARCH", "CGO_ENABLED", "CC"],
    {
      cwd: tmpdir(),
      encoding: "utf8",
      env: minimalU2GoEnvironment(join(tmpdir(), "countershape-u2-cgo-inspection"), toolchain),
      timeout: 10_000,
      maxBuffer: 64 * 1024,
    },
    spawn,
  );
  let facts;
  try {
    facts = JSON.parse(typeof result.stdout === "string" ? result.stdout : "");
  } catch {
    facts = null;
  }
  const hostArchitecture = process.arch === "x64" ? "amd64" : process.arch;
  if (
    result.error || result.signal || result.status !== 0 ||
    (typeof result.stderr === "string" && result.stderr.trim() !== "") ||
    !facts || facts.GOOS !== "darwin" || facts.GOARCH !== hostArchitecture ||
    facts.CGO_ENABLED !== "1" || facts.CC !== toolchain.compiler.path
  ) {
    throw new MutationGateError("CGO_TOOLCHAIN_INSPECTION_FAILED", `Go did not bind CGO to the admitted Darwin compiler: ${limitedOutput(result)}`);
  }
  return Object.freeze({
    goos: facts.GOOS,
    goarch: facts.GOARCH,
    enabled: facts.CGO_ENABLED,
    compilerPath: facts.CC,
    digest: `sha256:${sha256(Buffer.from(JSON.stringify(facts), "utf8"))}`,
  });
}

export async function admitU2DarwinToolchain(sourceEnvironment = process.env, spawn = spawnSync) {
  if (process.platform !== "darwin") {
    throw new MutationGateError("U2_PLATFORM_UNSUPPORTED", `U2 is a Darwin-only CGO gate, got ${process.platform}`);
  }
  const go = await admitTrustedGoToolchain(sourceEnvironment, spawn);
  const compiler = await resolveAdmittedDarwinCCompiler(sourceEnvironment);
  const compilerFacts = readAdmittedDarwinCCompilerFacts(compiler, spawn);
  const partial = Object.freeze({ ...go, compiler, compilerFacts });
  const cgo = readAdmittedDarwinCGOFacts(partial, spawn);
  return Object.freeze({ ...partial, cgo });
}

async function ensurePrivateDirectory(directory) {
  await mkdir(directory, { recursive: true, mode: 0o700 });
  await chmod(directory, 0o700);
}

async function prepareCacheRoot(cacheRoot) {
  await mkdir(cacheRoot, { mode: 0o700 });
  for (const name of ["home", "tmp", "go-tmp", "go-path", "go-build", "go-mod"]) {
    await ensurePrivateDirectory(join(cacheRoot, name));
  }
}

function expectedPackage(mutant) {
  return `${moduleImportPath}/${mutant.package.slice(2)}`;
}

export function runNamedU2GoTest({ sandbox, mutant, environment, toolchain, spawn = spawnSync }) {
  const guardedSpawn = (executable, args, options) =>
    spawnWithClosedToolchain(toolchain, executable, args, options, spawn);
  return runNamedGoTest({
    sandbox,
    mutant,
    environment,
    goToolchain: toolchain,
    expectedPackage: expectedPackage(mutant),
    spawn: guardedSpawn,
  });
}

async function prepareExperiment({ mutant, toolchain, phase, seedSnapshot }) {
  if (!seedSnapshot) {
    throw new MutationGateError("SEED_SNAPSHOT_MISSING", `${mutant.id}: ${phase} has no immutable source seed`);
  }
  await assertU2SourceAndSeedUnchanged(seedSnapshot);
  const sandboxParent = await mkdtemp(join(tmpdir(), `countershape-u2-${phase}-`));
  await chmod(sandboxParent, 0o700);
  const sandbox = join(sandboxParent, "repo");
  const cacheRoot = join(sandboxParent, "cache");
  try {
    await copyRegularAllowlist(seedSnapshot.root, sandbox, seedSnapshot.allowlist);
    const sourceManifest = await validateTreeAndReadManifest(sandbox, seedSnapshot.allowlist, seedSnapshot.scopeRoots);
    requireManifestEqual(sourceManifest, seedSnapshot.manifest, "EXPERIMENT_SEED_MISMATCH", `${mutant.id} ${phase} copy differs from seed`);
    await prepareCacheRoot(cacheRoot);
    return {
      sandboxParent,
      sandbox,
      cacheRoot,
      environment: minimalU2GoEnvironment(cacheRoot, toolchain),
      toolchain,
      sourceManifest,
      seedSnapshot,
    };
  } catch (error) {
    await rm(sandboxParent, { recursive: true, force: true });
    throw error;
  }
}

async function cleanupExperiment(context) {
  await rm(context.sandboxParent, { recursive: true, force: true });
}

function defaultRunNamedTest(context, mutant) {
  return runNamedU2GoTest({
    sandbox: context.sandbox,
    mutant,
    environment: context.environment,
    toolchain: context.toolchain,
  });
}

async function assertExperimentManifest(context, expected, code, detail) {
  if (!context.seedSnapshot) return;
  const actual = await validateTreeAndReadManifest(
    context.sandbox,
    context.seedSnapshot.allowlist,
    context.seedSnapshot.scopeRoots,
  );
  requireManifestEqual(actual, expected, code, detail);
}

function requireControlPass(result, mutantID) {
  if (result.classification.outcome !== NamedTestOutcome.Pass) {
    throw new MutationGateError(
      "BASELINE_TEST_NOT_PASSING",
      `${mutantID}: ${result.classification.outcome}: ${result.classification.detail}\n${result.output ?? ""}`,
    );
  }
}

export async function runU2Mutant(mutant, toolchain, dependencies = {}) {
  const prepare = dependencies.prepareExperiment ?? prepareExperiment;
  const cleanup = dependencies.cleanupExperiment ?? cleanupExperiment;
  const runNamed = dependencies.runNamedTest ?? defaultRunNamedTest;
  const applyMutation = dependencies.applyMutation ?? mutateRegularFileNoFollow;
  const usingDefaultPrepare = prepare === prepareExperiment;
  let ownedSeed;
  const seedSnapshot = dependencies.seedSnapshot ?? (usingDefaultPrepare ? (ownedSeed = await createU2SeedSnapshot()) : undefined);
  try {
    const baseline = await prepare({ mutant, toolchain, phase: "baseline", seedSnapshot });
    const roots = new Set([baseline.sandbox]);
    let baselineResult;
    try {
      baselineResult = await runNamed(baseline, mutant);
      if (baseline.seedSnapshot) {
        await assertExperimentManifest(
          baseline,
          baseline.seedSnapshot.manifest,
          "BASELINE_SOURCE_CHANGED",
          `${mutant.id}: baseline source changed while its named test ran`,
        );
      }
    } finally {
      await cleanup(baseline);
    }
    requireControlPass(baselineResult, mutant.id);

    const mutated = await prepare({ mutant, toolchain, phase: "mutant", seedSnapshot });
    if (roots.has(mutated.sandbox)) {
      await cleanup(mutated);
      throw new MutationGateError("EXPERIMENT_ROOT_REUSED", `${mutant.id}: baseline and mutant reused a root`);
    }
    roots.add(mutated.sandbox);
    let mutantResult;
    let mutatedManifest;
    try {
      await applyMutation(mutated.sandbox, mutant);
      if (mutated.seedSnapshot) {
        mutatedManifest = await assertExactU2MutantDelta(mutated.sandbox, mutated.seedSnapshot, mutant);
      }
      mutantResult = await runNamed(mutated, mutant);
      if (mutatedManifest) {
        await assertExperimentManifest(
          mutated,
          mutatedManifest,
          "MUTANT_SOURCE_CHANGED",
          `${mutant.id}: exact mutant source changed while its named test ran`,
        );
      }
    } finally {
      await cleanup(mutated);
    }

    const postControl = await prepare({ mutant, toolchain, phase: "post-control", seedSnapshot });
    if (roots.has(postControl.sandbox)) {
      await cleanup(postControl);
      throw new MutationGateError("EXPERIMENT_ROOT_REUSED", `${mutant.id}: A/B/A controls did not use three fresh roots`);
    }
    let postResult;
    try {
      postResult = await runNamed(postControl, mutant);
      if (postControl.seedSnapshot) {
        await assertExperimentManifest(
          postControl,
          postControl.seedSnapshot.manifest,
          "POST_CONTROL_SOURCE_CHANGED",
          `${mutant.id}: post-control source changed while its named test ran`,
        );
      }
    } finally {
      await cleanup(postControl);
    }
    requireControlPass(postResult, `${mutant.id} post-control`);
    if (seedSnapshot) await assertU2SourceAndSeedUnchanged(seedSnapshot);

    if (mutantResult.classification.outcome === NamedTestOutcome.Failure) {
      return `KILLED ${mutant.id} (${mutant.package} ${mutant.testName})`;
    }
    if (mutantResult.classification.outcome === NamedTestOutcome.Pass) {
      throw new MutationGateError("MUTANT_SURVIVED", `${mutant.id}: ${mutant.testName} passed after mutation\n${mutantResult.output ?? ""}`);
    }
    throw new MutationGateError(
      "MUTANT_INFRASTRUCTURE_FAILURE",
      `${mutant.id}: ${mutantResult.classification.outcome}: ${mutantResult.classification.detail}\n${mutantResult.output ?? ""}`,
    );
  } finally {
    if (ownedSeed) await cleanupU2SeedSnapshot(ownedSeed);
  }
}

export async function main() {
  assertU2MutantDefinitionSet();
  await assertU2ManifestMatchesTree(repoRoot);
  await assertReviewedU2Anchors(repoRoot);
  const toolchain = await admitU2DarwinToolchain();
  const seed = await createU2SeedSnapshot();
  const results = [];
  try {
    for (const mutant of U2_MUTANTS) {
      results.push(await runU2Mutant(mutant, toolchain, { seedSnapshot: seed }));
    }
    await assertU2SourceAndSeedUnchanged(seed);
  } finally {
    await cleanupU2SeedSnapshot(seed);
  }
  revalidateAdmittedGoExecutable(toolchain.executable);
  revalidateAdmittedDarwinCCompiler(toolchain.compiler);
  process.stdout.write(`Trusted Go realpath: ${toolchain.executable.path}\n`);
  process.stdout.write(`Trusted Go sha256: ${toolchain.executable.sha256}\n`);
  process.stdout.write(`Trusted Go version: ${toolchain.version.line}\n`);
  process.stdout.write(`Trusted C compiler realpath: ${toolchain.compiler.path}\n`);
  process.stdout.write(`Trusted C compiler sha256: ${toolchain.compiler.sha256}\n`);
  process.stdout.write(`Trusted C compiler: ${toolchain.compilerFacts.versionLine}\n`);
  process.stdout.write(`Trusted C compiler target: ${toolchain.compilerFacts.target}\n`);
  process.stdout.write(`Closed CGO facts digest: ${toolchain.cgo.digest}\n`);
  process.stdout.write(`Immutable U2 seed manifest: ${seed.manifest.digest}\n`);
  process.stdout.write("Containment notice: private mutation copies retain the invoking user's host filesystem and network authority.\n");
  for (const result of results) process.stdout.write(`${result}\n`);
  process.stdout.write(`U2 mutation gate killed ${results.length}/${REQUIRED_U2_MUTANT_IDS.length} exact required mutants.\n`);
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(modulePath)) {
  main().catch((error) => {
    process.stderr.write(`${error.stack ?? error}\n`);
    process.exitCode = 1;
  });
}
