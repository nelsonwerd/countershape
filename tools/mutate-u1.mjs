#!/usr/bin/env node

import {
  chmod,
  constants,
  lstat,
  mkdir,
  mkdtemp,
  open,
  readFile,
  readdir,
  realpath,
  rm,
  writeFile,
} from "node:fs/promises";
import {
  accessSync,
  closeSync,
  fstatSync,
  lstatSync,
  openSync,
  readFileSync,
} from "node:fs";
import { createHash } from "node:crypto";
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

const modulePath = fileURLToPath(import.meta.url);
const repoRoot = resolve(dirname(modulePath), "..");

function sha256(bytes) {
  return createHash("sha256").update(bytes).digest("hex");
}

// This is deliberately an exact file manifest, not a denylist or recursive
// copy. Adding a U1 source or test file requires an explicit security review of
// this list. Unlisted files are not copied. This private copy is not an OS
// security sandbox: test processes retain the invoking user's host filesystem
// and network authority, which is reported truthfully by main().
export const SANDBOX_FILE_ALLOWLIST = Object.freeze([
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
  "internal/choice/validation.go",
  "internal/choice/validation_test.go",
  "internal/compare/outcome_map.go",
  "internal/compare/outcome_map_test.go",
  "internal/domain/domain_test.go",
  "internal/domain/envelope.go",
  "internal/domain/errors.go",
  "internal/domain/identity.go",
	"internal/domain/projection.go",
  "internal/domain/receipt.go",
  "internal/domain/transitions.go",
  "internal/domain/world.go",
  "internal/observe/eligibility.go",
  "internal/observe/eligibility_test.go",
  "internal/observe/identity.go",
  "internal/reduce/model.go",
  "internal/reduce/model_test.go",
  "internal/spec/compiler.go",
  "internal/spec/compiler_test.go",
  "internal/spec/errors.go",
  "internal/spec/parse.go",
  "spec/examples/v1/source-spec.valid.json",
]);

// P01 requires this exact eighteen-mutant set. Keep the independent ID contract so
// deleting, duplicating, or silently replacing a mutant cannot yield a smaller
// but green "N/N" report.
export const REQUIRED_MUTANT_IDS = Object.freeze([
  "remove-digest-kind-separator",
  "bypass-duplicate-key-rejection",
  "accept-negative-zero",
  "drop-candidate-key-from-map-identity",
  "replace-labeled-map-with-partition-shape",
  "turn-control-failure-eligible",
  "collapse-unresolved-into-changes",
  "permit-incomplete-one-minimal-sweep",
  "allow-empty-field-selection",
  "compile-allow-many-as-independent-sets",
  "bypass-programmatic-array-profile",
  "bypass-world-plan-binding",
  "drop-measurement-set-from-admission",
  "ignore-trial-stimulus-binding",
  "ignore-shared-admission-set",
  "ignore-reduction-phase-binding",
  "allow-reused-captured-observation",
  "ignore-projection-definition-binding",
]);

// This literal is the reviewed semantic contract. MUTANTS below is a distinct
// immutable working copy so runtime validation detects tuple tampering rather
// than treating the mutable input as its own source of truth.
export const REVIEWED_MUTANT_CONTRACT = Object.freeze([
  Object.freeze({
    id: "remove-digest-kind-separator",
    file: "internal/canon/digest.go",
    find: "_, _ = h.Write([]byte{digestKindSeparator}) // MUTANT_U1_CANON_REMOVE_KIND_SEPARATOR",
    replace: "_, _ = h.Write(nil) // MUTANT_U1_CANON_REMOVE_KIND_SEPARATOR",
    package: "./internal/canon",
    testName: "TestDigestKindSeparation",
  }),
  Object.freeze({
    id: "bypass-duplicate-key-rejection",
    file: "internal/canon/value.go",
    find: "if duplicate := seen[name]; duplicate { // MUTANT_U1_CANON_BYPASS_DUPLICATE_KEY",
    replace: "if duplicate := seen[name]; false && duplicate { // MUTANT_U1_CANON_BYPASS_DUPLICATE_KEY",
    package: "./internal/canon",
    testName: "TestParseRejectsDuplicateKeys",
  }),
  Object.freeze({
    id: "accept-negative-zero",
    file: "internal/canon/scan.go",
    find: "if bytes.Equal(raw, []byte(\"-0\")) { // MUTANT_U1_CANON_ACCEPT_NEGATIVE_ZERO",
    replace: "if false && bytes.Equal(raw, []byte(\"-0\")) { // MUTANT_U1_CANON_ACCEPT_NEGATIVE_ZERO",
    package: "./internal/canon",
    testName: "TestParseRejectsNegativeZero",
  }),
  Object.freeze({
    id: "drop-candidate-key-from-map-identity",
    file: "internal/compare/outcome_map.go",
    find: [
      "\t// MUTATION_ANCHOR: candidate-key-is-part-of-labeled-map-identity",
      "\treturn entry.CandidateKey.String()",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: candidate-key-is-part-of-labeled-map-identity",
      "\treturn \"\"",
    ].join("\n"),
    package: "./internal/compare",
    testName: "TestCandidateKeyIsPartOfLabeledMapIdentity",
  }),
  Object.freeze({
    id: "replace-labeled-map-with-partition-shape",
    file: "internal/compare/outcome_map.go",
    // Mutate only the exact-map conjunct. The strengthened comparability guard
    // remains in force, so the mutant tests labeled-map identity rather than
    // weakening roster/admission/eligibility comparability at the same time.
    find: "left.preservationDigest.Valid() && left.preservationDigest == right.preservationDigest",
    replace: "samePartitionShape(left, right)",
    package: "./internal/compare",
    testName: "TestPreservationUsesLabeledMapNotPartitionShape",
  }),
  // Controlled facts return before behaviorWasCaptured. Mutating only that
  // helper would be a no-op survivor, so this mutant targets the authoritative
  // tagged-sum branch that carries control reasons to ineligibility.
  Object.freeze({
    id: "turn-control-failure-eligible",
    file: "internal/observe/eligibility.go",
    find: [
      "\tif fact.kind == trialControlled {",
      "\t\tif len(fact.controls) == 0 {",
      "\t\t\treturn Eligibility{reasons: []domain.ControlReason{domain.ControlProjectionRejected}}",
      "\t\t}",
      "\t\treturn Eligibility{reasons: append([]domain.ControlReason(nil), fact.controls...)}",
      "\t}",
    ].join("\n"),
    replace: [
      "\tif fact.kind == trialControlled {",
      "\t\treturn Eligibility{eligible: true}",
      "\t}",
    ].join("\n"),
    package: "./internal/observe",
    testName: "TestControlFailureNeverBecomesEligible",
  }),
  Object.freeze({
    id: "collapse-unresolved-into-changes",
    file: "internal/reduce/model.go",
    find: [
      "\t// MUTATION_ANCHOR: unresolved-is-not-changes",
      "\treturn Unresolved",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: unresolved-is-not-changes",
      "\treturn Changes",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestUnresolvedNeverCollapsesIntoChanges",
  }),
  Object.freeze({
    id: "permit-incomplete-one-minimal-sweep",
    file: "internal/reduce/model.go",
    find: [
      "\t// MUTATION_ANCHOR: incomplete-sweep-cannot-prove-one-minimal",
      "\treturn state == SweepComplete",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: incomplete-sweep-cannot-prove-one-minimal",
      "\treturn state == SweepComplete || state == SweepIncomplete",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestIncompleteSweepCannotConstructOneMinimal",
  }),
  Object.freeze({
    id: "allow-empty-field-selection",
    file: "internal/choice/validation.go",
    find: "if len(input.SelectedFields) == 0 { // MUTANT_U1_CHOICE_ALLOW_EMPTY_FIELDS",
    replace: "if false && len(input.SelectedFields) == 0 { // MUTANT_U1_CHOICE_ALLOW_EMPTY_FIELDS",
    package: "./internal/choice",
    testName: "TestValidateRulingRefusesEmptySelectedFields",
  }),
  Object.freeze({
    id: "compile-allow-many-as-independent-sets",
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
    id: "bypass-programmatic-array-profile",
    file: "internal/canon/value.go",
    find: [
      "\t// MUTANT_U1_CANON_BYPASS_PROGRAMMATIC_ARRAY_PROFILE: aggregate resource checks are constructor authority.",
      "\tif _, err := value.CanonicalChecked(); err != nil {",
      "\t\treturn Value{}, err",
      "\t}",
    ].join("\n"),
    replace: [
      "\t// MUTANT_U1_CANON_BYPASS_PROGRAMMATIC_ARRAY_PROFILE: aggregate resource checks are constructor authority.",
      "\tif false {",
      "\t\treturn Value{}, refusal(CodeInvalidValue, UnknownOffset, \"mutated constructor\")",
      "\t}",
    ].join("\n"),
    package: "./internal/canon",
    testName: "TestProgrammaticConstructorsEnforceTheCompleteResourceProfile",
  }),
  Object.freeze({
    id: "bypass-world-plan-binding",
    file: "internal/domain/envelope.go",
    find: [
      "\t// MUTANT_U1_DOMAIN_BYPASS_WORLD_PLAN_BINDING: retained candidate authority must match every executable plan boundary.",
      "\tif binding.identity.WorldPlanDigest != plan.digest ||",
      "\t\tbinding.identity.MaterializationPolicyDigest != plan.materializationPolicyDigest ||",
      "\t\tbinding.identity.AdapterDigest != plan.adapterDigest ||",
      "\t\tbinding.identity.RunnerDigest != plan.adapter.RunnerDigest ||",
      "\t\tbinding.identity.ProjectionDefinitionDigest != plan.projectionDefinition.Digest() {",
    ].join("\n"),
    replace: [
      "\t// MUTANT_U1_DOMAIN_BYPASS_WORLD_PLAN_BINDING: retained candidate authority must match every executable plan boundary.",
      "\tif false && (binding.identity.WorldPlanDigest != plan.digest ||",
      "\t\tbinding.identity.MaterializationPolicyDigest != plan.materializationPolicyDigest ||",
      "\t\tbinding.identity.AdapterDigest != plan.adapterDigest ||",
      "\t\tbinding.identity.RunnerDigest != plan.adapter.RunnerDigest ||",
      "\t\tbinding.identity.ProjectionDefinitionDigest != plan.projectionDefinition.Digest()) {",
    ].join("\n"),
    package: "./internal/observe",
    testName: "TestClassifyRefusesCrossPlanPurposeStimulusAndCandidateBindings",
  }),
  Object.freeze({
    id: "drop-measurement-set-from-admission",
    file: "internal/domain/envelope.go",
    find: [
      "\t\t// MUTANT_U1_DOMAIN_DROP_MEASUREMENT_SET_FROM_ADMISSION: concrete matrix identity is authority-bearing.",
      "\t\tMeasurementDigests: measurementDigests,",
    ].join("\n"),
    replace: [
      "\t\t// MUTANT_U1_DOMAIN_DROP_MEASUREMENT_SET_FROM_ADMISSION: concrete matrix identity is authority-bearing.",
      "\t\tMeasurementDigests: []string{},",
    ].join("\n"),
    package: "./internal/domain",
    testName: "TestComparisonAdmissionIdentityIncludesExactMeasurementMatrix",
  }),
  Object.freeze({
    id: "ignore-trial-stimulus-binding",
    file: "internal/observe/eligibility.go",
    find: [
      "\t\t// MUTANT_U1_OBSERVE_IGNORE_STIMULUS_BINDING: stimulus equality is authority-bearing.",
      "\t\tif world.CandidateKey() != candidateKey || world.PlanDigest() != planDigest ||",
      "\t\t\tworld.StimulusDigest() != stimulusDigest || world.EnvelopeDigest() != envelopeDigest ||",
      "\t\t\tworld.CapturePolicyDigest() != capturePolicyDigest || world.ProjectionDefinitionDigest() != projectionDefinitionDigest ||",
      "\t\t\tworld.Purpose() != phase || world.RequiredFreshTrials() != requiredFreshTrials ||",
      "\t\t\ttrial.admission.EnvelopeDigest() != envelopeDigest || trial.admission.PlanDigest() != planDigest ||",
      "\t\t\ttrial.admission.StimulusDigest() != stimulusDigest || trial.admission.Purpose() != phase ||",
    ].join("\n"),
    replace: [
      "\t\t// MUTANT_U1_OBSERVE_IGNORE_STIMULUS_BINDING: stimulus equality is authority-bearing.",
      "\t\tif world.CandidateKey() != candidateKey || world.PlanDigest() != planDigest ||",
      "\t\t\tfalse || world.EnvelopeDigest() != envelopeDigest ||",
      "\t\t\tworld.CapturePolicyDigest() != capturePolicyDigest || world.ProjectionDefinitionDigest() != projectionDefinitionDigest ||",
      "\t\t\tworld.Purpose() != phase || world.RequiredFreshTrials() != requiredFreshTrials ||",
      "\t\t\ttrial.admission.EnvelopeDigest() != envelopeDigest || trial.admission.PlanDigest() != planDigest ||",
      "\t\t\tfalse || trial.admission.Purpose() != phase ||",
    ].join("\n"),
    package: "./internal/observe",
    testName: "TestBatchIdentityCannotRelabelIdenticalTrialsToAnotherStimulus",
  }),
  Object.freeze({
    id: "ignore-shared-admission-set",
    file: "internal/compare/outcome_map.go",
    find: [
      "\t\t// MUTANT_U1_COMPARE_IGNORE_SHARED_ADMISSION_SET: every candidate must come from the same concrete matrices.",
      "\t\t} else if !sameDigestList(batchAdmissions, admissionDigests) || batch.PlanDigest() != planDigest ||",
    ].join("\n"),
    replace: [
      "\t\t// MUTANT_U1_COMPARE_IGNORE_SHARED_ADMISSION_SET: every candidate must come from the same concrete matrices.",
      "\t\t} else if batch.PlanDigest() != planDigest ||",
    ].join("\n"),
    package: "./internal/compare",
    testName: "TestCandidateOutcomeMapRejectsDifferentAdmissionSetsEvenWhenBasisMatches",
  }),
  Object.freeze({
    id: "ignore-reduction-phase-binding",
    file: "internal/reduce/model.go",
    find: [
      "\t// MUTANT_U1_REDUCTION_PHASE_BINDING: phase is authority, not a display label.",
      "\tif observed.Phase() != requiredPurpose {",
    ].join("\n"),
    replace: [
      "\t// MUTANT_U1_REDUCTION_PHASE_BINDING: phase is authority, not a display label.",
      "\tif false && observed.Phase() != requiredPurpose {",
    ].join("\n"),
    package: "./internal/reduce",
    testName: "TestEvaluationRequiresPurposeBoundEvidence",
  }),
  Object.freeze({
    id: "allow-reused-captured-observation",
    file: "internal/observe/eligibility.go",
    find: "if _, duplicate := seenObservations[observationDigest]; duplicate { // MUTANT_U1_OBSERVE_ALLOW_REUSED_CAPTURE",
    replace: "if _, duplicate := seenObservations[observationDigest]; false && duplicate { // MUTANT_U1_OBSERVE_ALLOW_REUSED_CAPTURE",
    package: "./internal/observe",
    testName: "TestClassifyRefusesZeroExcessReuseAndLineageMismatch",
  }),
  Object.freeze({
    id: "ignore-projection-definition-binding",
    file: "internal/choice/validation.go",
    find: "if registry.projectionDefinitionDigest != roster.ProjectionDefinitionDigest() { // MUTANT_U1_CHOICE_IGNORE_PROJECTION_DEFINITION",
    replace: "if false && registry.projectionDefinitionDigest != roster.ProjectionDefinitionDigest() { // MUTANT_U1_CHOICE_IGNORE_PROJECTION_DEFINITION",
    package: "./internal/choice",
    testName: "TestConfirmedOutcomeSetRequiresOpaqueRosterAndExactProofCoverage",
  }),
]);

export const MUTANTS = Object.freeze(
  REVIEWED_MUTANT_CONTRACT.map((mutant) => Object.freeze({ ...mutant })),
);

export class MutationGateError extends Error {
  constructor(code, message) {
    super(`${code}: ${message}`);
    this.name = "MutationGateError";
    this.code = code;
  }
}

export const NamedTestOutcome = Object.freeze({
  Pass: "NAMED_TEST_PASS",
  Failure: "NAMED_TEST_FAILURE",
  Missing: "NAMED_TEST_MISSING",
  Infrastructure: "INFRASTRUCTURE_FAILURE",
});

const MODULE_IMPORT_PATH = "github.com/nelsonwerd/countershape";
const MUTANT_TUPLE_FIELDS = Object.freeze([
  "file",
  "find",
  "id",
  "package",
  "replace",
  "testName",
]);
const REQUIRED_SUPPORT_FILES = Object.freeze([
  "go.mod",
  "spec/examples/v1/source-spec.valid.json",
]);

function exactOccurrenceCount(source, anchor) {
  if (anchor.length === 0) return 0;
  return source.split(anchor).length - 1;
}

function assertSafeRelativeFile(relativeFile) {
  if (
    typeof relativeFile !== "string" ||
    relativeFile.length === 0 ||
    isAbsolute(relativeFile) ||
    relativeFile.split(/[\\/]/u).some((component) => component === "" || component === "." || component === "..")
  ) {
    throw new MutationGateError("UNSAFE_ALLOWLIST_PATH", `invalid relative file ${JSON.stringify(relativeFile)}`);
  }
}

function assertContained(root, target, code) {
  const fromRoot = relativePath(root, target);
  if (fromRoot === "" || fromRoot === ".." || fromRoot.startsWith(`..${sep}`) || isAbsolute(fromRoot)) {
    throw new MutationGateError(code, `${target} is outside ${root}`);
  }
}

async function enumerateInternalSourceFiles(sourceRoot) {
  const internalRoot = resolve(sourceRoot, "internal");
  let rootMetadata;
  try {
    rootMetadata = await lstat(internalRoot);
  } catch (error) {
    throw new MutationGateError(
      "MANIFEST_INTERNAL_ROOT_MISSING",
      `${internalRoot}: ${error.code ?? error.message}`,
    );
  }
  if (rootMetadata.isSymbolicLink() || !rootMetadata.isDirectory()) {
    throw new MutationGateError(
      "MANIFEST_INTERNAL_ROOT_NONREGULAR",
      "internal must be a real directory",
    );
  }

  const found = [];
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
        continue;
      }
      if (!entry.isFile()) {
        throw new MutationGateError("MANIFEST_TREE_NONREGULAR", `${relative} is not a regular file`);
      }
      if (!relative.endsWith(".go")) {
        throw new MutationGateError(
          "MANIFEST_TREE_UNSUPPORTED_FILE",
          `${relative} is an unreviewed non-Go internal source artifact`,
        );
      }
      found.push(relative);
    }
  }
  await visit(internalRoot);
  return found.sort();
}

export async function assertManifestMatchesTree(
  sourceRoot,
  allowlist = SANDBOX_FILE_ALLOWLIST,
) {
  if (allowlist.length === 0 || new Set(allowlist).size !== allowlist.length) {
    throw new MutationGateError("INVALID_FILE_ALLOWLIST", "file allowlist must be nonempty and unique");
  }
  for (const relativeFile of allowlist) assertSafeRelativeFile(relativeFile);

  const supportFiles = allowlist.filter((file) => !file.startsWith("internal/")).sort();
  const requiredSupport = [...REQUIRED_SUPPORT_FILES].sort();
  if (
    supportFiles.length !== requiredSupport.length ||
    supportFiles.some((file, index) => file !== requiredSupport[index])
  ) {
    throw new MutationGateError(
      "MANIFEST_SUPPORT_SET_MISMATCH",
      `support files [${supportFiles.join(",")}] differ from the reviewed support set`,
    );
  }

  const expected = allowlist.filter((file) => file.startsWith("internal/")).sort();
  if (expected.some((file) => !file.endsWith(".go"))) {
    throw new MutationGateError(
      "MANIFEST_SUPPORT_SET_MISMATCH",
      "internal manifest entries must be Go source files",
    );
  }
  const actual = await enumerateInternalSourceFiles(sourceRoot);
  const expectedSet = new Set(expected);
  const actualSet = new Set(actual);
  const unlisted = actual.filter((file) => !expectedSet.has(file));
  const missing = expected.filter((file) => !actualSet.has(file));
  if (unlisted.length > 0 || missing.length > 0) {
    throw new MutationGateError(
      "MANIFEST_TREE_MISMATCH",
      `unlisted=[${unlisted.join(",")}] missing=[${missing.join(",")}]`,
    );
  }
}

export function assertMutantDefinitionSet(
  mutants = MUTANTS,
  requiredIDs = REQUIRED_MUTANT_IDS,
  reviewedContract = REVIEWED_MUTANT_CONTRACT,
) {
  if (requiredIDs.length !== 18 || new Set(requiredIDs).size !== 18) {
    throw new MutationGateError("INVALID_REQUIRED_ID_SET", "required mutant ID contract must contain eighteen unique IDs");
  }
  if (mutants.length !== 18) {
    throw new MutationGateError("MUTANT_SET_MISMATCH", `defined ${mutants.length} mutants, want exactly 18`);
  }
  const actualIDs = mutants.map((mutant) => mutant.id);
  if (new Set(actualIDs).size !== actualIDs.length) {
    throw new MutationGateError("DUPLICATE_MUTANT_ID", "mutant IDs must be unique");
  }
  const required = new Set(requiredIDs);
  for (const id of actualIDs) {
    if (!required.has(id)) {
      throw new MutationGateError("MUTANT_SET_MISMATCH", `unexpected mutant ID ${id}`);
    }
  }
  for (const id of requiredIDs) {
    if (!actualIDs.includes(id)) {
      throw new MutationGateError("MUTANT_SET_MISMATCH", `missing required mutant ID ${id}`);
    }
  }
  if (reviewedContract.length !== REQUIRED_MUTANT_IDS.length) {
    throw new MutationGateError(
      "INVALID_REVIEWED_MUTANT_CONTRACT",
      `reviewed contract contains ${reviewedContract.length} tuples, want ${REQUIRED_MUTANT_IDS.length}`,
    );
  }
  const reviewedByID = new Map(reviewedContract.map((mutant) => [mutant.id, mutant]));
  for (const mutant of mutants) {
    const reviewed = reviewedByID.get(mutant.id);
    if (!reviewed) {
      throw new MutationGateError("MUTANT_CONTRACT_MISMATCH", `${mutant.id} has no reviewed tuple`);
    }
    const actualFields = Object.keys(mutant).sort();
    if (
      actualFields.length !== MUTANT_TUPLE_FIELDS.length ||
      actualFields.some((field, index) => field !== MUTANT_TUPLE_FIELDS[index])
    ) {
      throw new MutationGateError(
        "MUTANT_CONTRACT_MISMATCH",
        `${mutant.id} fields [${actualFields.join(",")}] differ from the reviewed tuple shape`,
      );
    }
    for (const field of MUTANT_TUPLE_FIELDS) {
      if (mutant[field] !== reviewed[field]) {
        throw new MutationGateError(
          "MUTANT_CONTRACT_MISMATCH",
          `${mutant.id}.${field} differs from the reviewed contract`,
        );
      }
    }
  }
  const allowedFiles = new Set(SANDBOX_FILE_ALLOWLIST);
  for (const mutant of mutants) {
    if (!allowedFiles.has(mutant.file)) {
      throw new MutationGateError("MUTANT_FILE_NOT_ALLOWLISTED", `${mutant.id} targets ${mutant.file}`);
    }
    if (typeof mutant.find !== "string" || mutant.find.length === 0 || mutant.find === mutant.replace) {
      throw new MutationGateError("INVALID_MUTATION", `${mutant.id} has an empty or ineffective replacement`);
    }
    if (!/^\.\/internal\/[a-z]+$/u.test(mutant.package)) {
      throw new MutationGateError("INVALID_MUTATION_PACKAGE", `${mutant.id} has invalid package ${mutant.package}`);
    }
    if (!/^Test[A-Za-z0-9_]+$/u.test(mutant.testName)) {
      throw new MutationGateError("INVALID_MUTATION_TEST", `${mutant.id} has invalid test ${mutant.testName}`);
    }
  }
}

export function applyExactMutation(source, mutant) {
  const count = exactOccurrenceCount(source, mutant.find);
  if (count !== 1) {
    throw new MutationGateError(
      "MUTATION_ANCHOR_COUNT",
      `${mutant.id}: exact anchor count in ${mutant.file} was ${count}, want 1`,
    );
  }
  const mutated = source.replace(mutant.find, mutant.replace);
  if (mutated === source) {
    throw new MutationGateError("INEFFECTIVE_MUTATION", `${mutant.id}: replacement made no change`);
  }
  return mutated;
}

async function writeAllAt(handle, bytes) {
  let offset = 0;
  while (offset < bytes.length) {
    const result = await handle.write(bytes, offset, bytes.length - offset, offset);
    if (!Number.isInteger(result.bytesWritten) || result.bytesWritten <= 0) {
      throw new MutationGateError("MUTATION_TARGET_WRITE_FAILED", "mutation target write made no progress");
    }
    offset += result.bytesWritten;
  }
  await handle.truncate(bytes.length);
  await handle.sync();
}

export async function mutateRegularFileNoFollow(sandboxRoot, mutant) {
  assertSafeRelativeFile(mutant.file);
  const target = resolve(sandboxRoot, mutant.file);
  assertContained(sandboxRoot, target, "MUTATION_TARGET_ESCAPE");
  await assertNoSymlinkedSourceParents(sandboxRoot, mutant.file);

  let admittedMetadata;
  try {
    admittedMetadata = await lstat(target);
  } catch (error) {
    throw new MutationGateError(
      "MUTATION_TARGET_MISSING",
      `${mutant.file}: ${error.code ?? error.message}`,
    );
  }
  if (admittedMetadata.isSymbolicLink()) {
    throw new MutationGateError("MUTATION_TARGET_SYMLINK", `${mutant.file} is a symbolic link`);
  }
  if (!admittedMetadata.isFile()) {
    throw new MutationGateError("MUTATION_TARGET_NONREGULAR", `${mutant.file} is not a regular file`);
  }
  if (admittedMetadata.nlink !== 1) {
    throw new MutationGateError(
      "MUTATION_TARGET_LINK_COUNT",
      `${mutant.file} has link count ${admittedMetadata.nlink}, want exactly 1`,
    );
  }
  if (!Number.isInteger(constants.O_NOFOLLOW)) {
    throw new MutationGateError(
      "MUTATION_NOFOLLOW_UNAVAILABLE",
      "the platform does not expose O_NOFOLLOW",
    );
  }

  let handle;
  try {
    handle = await open(target, constants.O_RDWR | constants.O_NOFOLLOW);
    const openedMetadata = await handle.stat();
    if (
      !openedMetadata.isFile() ||
      openedMetadata.nlink !== 1 ||
      openedMetadata.dev !== admittedMetadata.dev ||
      openedMetadata.ino !== admittedMetadata.ino
    ) {
      throw new MutationGateError(
        "MUTATION_TARGET_CHANGED",
        `${mutant.file} changed during no-follow admission`,
      );
    }
    const sourceBytes = await handle.readFile();
    const source = sourceBytes.toString("utf8");
    if (!Buffer.from(source, "utf8").equals(sourceBytes)) {
      throw new MutationGateError(
        "MUTATION_TARGET_INVALID_UTF8",
        `${mutant.file} is not valid UTF-8`,
      );
    }
    const mutated = applyExactMutation(source, mutant);
    await writeAllAt(handle, Buffer.from(mutated, "utf8"));
    const finalMetadata = await handle.stat();
    if (
      !finalMetadata.isFile() ||
      finalMetadata.nlink !== 1 ||
      finalMetadata.dev !== admittedMetadata.dev ||
      finalMetadata.ino !== admittedMetadata.ino
    ) {
      throw new MutationGateError(
        "MUTATION_TARGET_CHANGED",
        `${mutant.file} changed while the mutation was written`,
      );
    }
  } catch (error) {
    if (error instanceof MutationGateError) throw error;
    throw new MutationGateError(
      "MUTATION_TARGET_OPEN_FAILED",
      `${mutant.file}: ${error.code ?? error.message}`,
    );
  } finally {
    await handle?.close();
  }
}

async function ensurePrivateDirectory(directory) {
  await mkdir(directory, { recursive: true, mode: 0o700 });
  await chmod(directory, 0o700);
}

async function createPrivateRoot(directory) {
  try {
    await mkdir(directory, { recursive: false, mode: 0o700 });
    await chmod(directory, 0o700);
  } catch (error) {
    throw new MutationGateError("PRIVATE_ROOT_CREATE_FAILED", `${directory}: ${error.code ?? error.message}`);
  }
}

async function assertNoSymlinkedSourceParents(sourceRoot, relativeFile) {
  let rootMetadata;
  try {
    rootMetadata = await lstat(sourceRoot);
  } catch (error) {
    throw new MutationGateError("SOURCE_ROOT_MISSING", `${sourceRoot}: ${error.code ?? error.message}`);
  }
  if (rootMetadata.isSymbolicLink() || !rootMetadata.isDirectory()) {
    throw new MutationGateError("SOURCE_ROOT_NONREGULAR", "source root must be a real directory");
  }
  const parent = dirname(relativeFile);
  if (parent === ".") return;
  let current = sourceRoot;
  for (const component of parent.split(/[\\/]/u)) {
    current = join(current, component);
    let metadata;
    try {
      metadata = await lstat(current);
    } catch (error) {
      throw new MutationGateError("ALLOWLIST_PARENT_MISSING", `${relativeFile}: ${error.code ?? error.message}`);
    }
    if (metadata.isSymbolicLink()) {
      throw new MutationGateError("SOURCE_SYMLINK_PARENT", `${relativeFile} traverses a symbolic-link parent`);
    }
    if (!metadata.isDirectory()) {
      throw new MutationGateError("SOURCE_NONREGULAR_PARENT", `${relativeFile} traverses a nondirectory parent`);
    }
  }
}

async function readRegularFileNoFollow(sourceRoot, relativeFile) {
  assertSafeRelativeFile(relativeFile);
  const source = resolve(sourceRoot, relativeFile);
  assertContained(sourceRoot, source, "SOURCE_PATH_ESCAPE");
  await assertNoSymlinkedSourceParents(sourceRoot, relativeFile);
  let metadata;
  try {
    metadata = await lstat(source);
  } catch (error) {
    throw new MutationGateError("ALLOWLIST_FILE_MISSING", `${relativeFile}: ${error.code ?? error.message}`);
  }
  if (metadata.isSymbolicLink()) {
    throw new MutationGateError("SOURCE_SYMLINK", `${relativeFile} is a symbolic link`);
  }
  if (!metadata.isFile()) {
    throw new MutationGateError("SOURCE_NONREGULAR", `${relativeFile} is not a regular file`);
  }

  let handle;
  try {
    handle = await open(source, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
    const opened = await handle.stat();
    if (!opened.isFile()) {
      throw new MutationGateError("SOURCE_NONREGULAR", `${relativeFile} changed to a nonregular file`);
    }
    if (opened.dev !== metadata.dev || opened.ino !== metadata.ino) {
      throw new MutationGateError("SOURCE_CHANGED_DURING_COPY", `${relativeFile} changed during admission`);
    }
    return await handle.readFile();
  } catch (error) {
    if (error instanceof MutationGateError) throw error;
    throw new MutationGateError("SOURCE_OPEN_FAILED", `${relativeFile}: ${error.code ?? error.message}`);
  } finally {
    await handle?.close();
  }
}

export async function copyRegularAllowlist(sourceRoot, destinationRoot, allowlist = SANDBOX_FILE_ALLOWLIST) {
  if (allowlist.length === 0 || new Set(allowlist).size !== allowlist.length) {
    throw new MutationGateError("INVALID_FILE_ALLOWLIST", "file allowlist must be nonempty and unique");
  }
  for (const relativeFile of allowlist) assertSafeRelativeFile(relativeFile);
  await createPrivateRoot(destinationRoot);
  for (const relativeFile of allowlist) {
    const bytes = await readRegularFileNoFollow(sourceRoot, relativeFile);
    const destination = resolve(destinationRoot, relativeFile);
    assertContained(destinationRoot, destination, "DESTINATION_PATH_ESCAPE");
    await ensurePrivateDirectory(dirname(destination));
    await writeFile(destination, bytes, { flag: "wx", mode: 0o600 });
    await chmod(destination, 0o600);
  }
}

export async function readAllowlistManifest(root, allowlist = SANDBOX_FILE_ALLOWLIST) {
  const entries = [];
  for (const relativeFile of allowlist) {
    assertSafeRelativeFile(relativeFile);
    const bytes = await readRegularFileNoFollow(root, relativeFile);
    entries.push(Object.freeze({
      file: relativeFile,
      bytes: bytes.length,
      sha256: sha256(bytes),
    }));
  }
  entries.sort((left, right) => left.file.localeCompare(right.file, "en"));
  const canonical = Buffer.from(
    `${entries.map((entry) => `${entry.file}\u0000${entry.bytes}\u0000${entry.sha256}`).join("\n")}\n`,
    "utf8",
  );
  return Object.freeze({
    entries: Object.freeze(entries),
    digest: `sha256:${sha256(canonical)}`,
  });
}

function manifestEntryMap(manifest) {
  return new Map(manifest.entries.map((entry) => [entry.file, entry]));
}

function assertManifestEqual(actual, expected, code, detail) {
  if (actual.digest !== expected.digest) {
    throw new MutationGateError(code, `${detail}: ${actual.digest} != ${expected.digest}`);
  }
}

export async function createSeedSnapshot() {
  await assertManifestMatchesTree(repoRoot);
  const parent = await mkdtemp(join(tmpdir(), "countershape-u1-seed-"));
  await chmod(parent, 0o700);
  const root = join(parent, "repo");
  try {
    await copyRegularAllowlist(repoRoot, root);
    const manifest = await readAllowlistManifest(root);
    return Object.freeze({ parent, root, manifest });
  } catch (error) {
    await rm(parent, { recursive: true, force: true });
    throw error;
  }
}

export async function cleanupSeedSnapshot(seed) {
  await rm(seed.parent, { recursive: true, force: true });
}

async function assertSeedUnchanged(seed) {
  const current = await readAllowlistManifest(seed.root);
  assertManifestEqual(current, seed.manifest, "SEED_SNAPSHOT_CHANGED", "immutable seed snapshot changed");
}

export async function assertExactMutantDelta(sandboxRoot, seed, mutant) {
  const actualManifest = await readAllowlistManifest(sandboxRoot);
  const seedEntries = manifestEntryMap(seed.manifest);
  const actualEntries = manifestEntryMap(actualManifest);
  const changed = [];
  for (const [file, expected] of seedEntries) {
    const actual = actualEntries.get(file);
    if (!actual || actual.bytes !== expected.bytes || actual.sha256 !== expected.sha256) {
      changed.push(file);
    }
  }
  for (const file of actualEntries.keys()) {
    if (!seedEntries.has(file)) changed.push(file);
  }
  if (changed.length !== 1 || changed[0] !== mutant.file) {
    throw new MutationGateError(
      "MUTANT_DELTA_SCOPE",
      `${mutant.id}: changed files [${changed.join(",")}], want exactly ${mutant.file}`,
    );
  }

  const seedBytes = await readRegularFileNoFollow(seed.root, mutant.file);
  const expectedBytes = Buffer.from(applyExactMutation(seedBytes.toString("utf8"), mutant), "utf8");
  const actualBytes = await readRegularFileNoFollow(sandboxRoot, mutant.file);
  if (!actualBytes.equals(expectedBytes)) {
    throw new MutationGateError(
      "MUTANT_DELTA_BYTES",
      `${mutant.id}: mutated target bytes differ from the reviewed exact replacement`,
    );
  }
  return actualManifest;
}

function nativeExecutableFormat() {
  if (process.platform === "darwin") return "MACH_O";
  if (process.platform === "linux") return "ELF";
  throw new MutationGateError(
    "GO_EXECUTABLE_PLATFORM_UNSUPPORTED",
    `native executable admission is not implemented for ${process.platform}`,
  );
}

function classifyExecutableFormat(bytes) {
  if (bytes.length < 4) return "UNKNOWN";
  const magic = bytes.subarray(0, 4).toString("hex");
  if (magic === "7f454c46") return "ELF";
  if (new Set([
    "feedface", "cefaedfe", "feedfacf", "cffaedfe",
    "cafebabe", "bebafeca", "cafebabf", "bfbafeca",
  ]).has(magic)) return "MACH_O";
  return "UNKNOWN";
}

function fingerprintExecutableSync(executablePath) {
  if (typeof executablePath !== "string" || !isAbsolute(executablePath)) {
    throw new MutationGateError("GO_EXECUTABLE_NOT_ABSOLUTE", "executable fingerprint requires an absolute path");
  }
  if (!Number.isInteger(constants.O_NOFOLLOW)) {
    throw new MutationGateError("GO_EXECUTABLE_NOFOLLOW_UNAVAILABLE", "O_NOFOLLOW is required for toolchain admission");
  }
  let admittedMetadata;
  let descriptor;
  try {
    admittedMetadata = lstatSync(executablePath);
    accessSync(executablePath, constants.X_OK);
    if (admittedMetadata.isSymbolicLink() || !admittedMetadata.isFile()) {
      throw new MutationGateError(
        "GO_EXECUTABLE_NOT_ADMITTED",
        `${executablePath} is not an executable regular file`,
      );
    }
    descriptor = openSync(executablePath, constants.O_RDONLY | constants.O_NOFOLLOW);
    const openedMetadata = fstatSync(descriptor);
    if (
      !openedMetadata.isFile() ||
      openedMetadata.dev !== admittedMetadata.dev ||
      openedMetadata.ino !== admittedMetadata.ino
    ) {
      throw new MutationGateError("GO_EXECUTABLE_CHANGED", `${executablePath} changed during admission`);
    }
    const bytes = readFileSync(descriptor);
    const format = classifyExecutableFormat(bytes);
    if (format !== nativeExecutableFormat()) {
      throw new MutationGateError(
        "GO_EXECUTABLE_BINARY_FORMAT",
        `${executablePath} has ${format} format, want native ${nativeExecutableFormat()}`,
      );
    }
    return Object.freeze({
      path: executablePath,
      sha256: sha256(bytes),
      dev: openedMetadata.dev,
      ino: openedMetadata.ino,
      size: openedMetadata.size,
      mode: openedMetadata.mode & 0o7777,
      format,
    });
  } catch (error) {
    if (error instanceof MutationGateError) throw error;
    throw new MutationGateError(
      "GO_EXECUTABLE_NOT_ADMITTED",
      `${executablePath}: ${error.code ?? error.message}`,
    );
  } finally {
    if (descriptor !== undefined) closeSync(descriptor);
  }
}

function sameExecutableFingerprint(left, right) {
  return left.path === right.path && left.sha256 === right.sha256 && left.dev === right.dev &&
    left.ino === right.ino && left.size === right.size && left.mode === right.mode &&
    left.format === right.format;
}

export function revalidateAdmittedGoExecutable(admission) {
  if (!admission || typeof admission.path !== "string") {
    throw new MutationGateError("GO_EXECUTABLE_NOT_ADMITTED", "missing executable admission record");
  }
  const current = fingerprintExecutableSync(admission.path);
  if (!sameExecutableFingerprint(current, admission)) {
    throw new MutationGateError(
      "GO_EXECUTABLE_CHANGED",
      `${admission.path} no longer matches admitted sha256/dev/inode/size/mode/format`,
    );
  }
  return current;
}

export async function resolveAdmittedGoExecutable(sourceEnvironment = process.env) {
  const declared = sourceEnvironment.COUNTERSHAPE_GO;
  if (typeof declared !== "string" || declared.length === 0) {
    throw new MutationGateError(
      "GO_EXECUTABLE_NOT_DECLARED",
      "set COUNTERSHAPE_GO to the absolute trusted Go executable reviewed for this receipt",
    );
  }
  if (!isAbsolute(declared)) {
    throw new MutationGateError(
      "GO_EXECUTABLE_NOT_ABSOLUTE",
      "COUNTERSHAPE_GO must be an absolute path; PATH lookup is deliberately unsupported",
    );
  }
  let admittedPath;
  try {
    admittedPath = await realpath(declared);
  } catch (error) {
    throw new MutationGateError("GO_EXECUTABLE_NOT_ADMITTED", `${declared}: ${error.code ?? error.message}`);
  }
  if (!isAbsolute(admittedPath)) {
    throw new MutationGateError("GO_EXECUTABLE_NOT_ADMITTED", `${declared} did not resolve absolutely`);
  }
  return fingerprintExecutableSync(admittedPath);
}

function sparseGoInspectionEnvironment(admission) {
  return Object.freeze({
    PATH: dirname(admission.path),
    HOME: join(tmpdir(), "countershape-u1-no-home"),
    TMPDIR: tmpdir(),
    GOENV: "off",
    GOWORK: "off",
    GOTOOLCHAIN: "local",
    GOPROXY: "off",
    GOSUMDB: "off",
    GOFLAGS: "-mod=readonly -buildvcs=false",
    CGO_ENABLED: "0",
    LANG: "C",
    LC_ALL: "C",
    NO_COLOR: "1",
    TZ: "UTC",
  });
}

function spawnWithExecutableRevalidation(admission, args, options, spawn) {
  revalidateAdmittedGoExecutable(admission);
  let result;
  try {
    result = spawn(admission.path, args, options);
  } finally {
    revalidateAdmittedGoExecutable(admission);
  }
  return result;
}

export function readAdmittedGoVersion(admission, spawn = spawnSync) {
  const result = spawnWithExecutableRevalidation(admission, ["version"], {
    cwd: tmpdir(),
    encoding: "utf8",
    env: sparseGoInspectionEnvironment(admission),
    timeout: 10_000,
    maxBuffer: 64 * 1024,
  }, spawn);
  const line = typeof result.stdout === "string" ? result.stdout.trim() : "";
  const match = /^go version (go[^\s\r\n]{1,100}) ([a-z0-9]+)\/([a-z0-9]+)$/u.exec(line);
  if (
    result.error || result.signal || result.status !== 0 ||
    (typeof result.stderr === "string" && result.stderr.trim() !== "") || !match
  ) {
    throw new MutationGateError(
      "GO_VERSION_INSPECTION_FAILED",
      `trusted executable did not emit one canonical native go version line: ${limitedOutput(result)}`,
    );
  }
  return Object.freeze({ line, release: match[1], goos: match[2], goarch: match[3] });
}

export function readAdmittedGoBuildFacts(admission, spawn = spawnSync) {
  const result = spawnWithExecutableRevalidation(admission, ["version", "-m", admission.path], {
    cwd: tmpdir(),
    encoding: "utf8",
    env: sparseGoInspectionEnvironment(admission),
    timeout: 10_000,
    maxBuffer: 64 * 1024,
  }, spawn);
  const output = typeof result.stdout === "string" ? result.stdout.trim() : "";
  const lines = output.split(/\r?\n/u).map((line) => line.trim());
  const pathLine = lines.find((line) => line === "path\tcmd/go");
  const goosLine = lines.find((line) => line.startsWith("build\tGOOS="));
  const goarchLine = lines.find((line) => line.startsWith("build\tGOARCH="));
  if (
    result.error || result.signal || result.status !== 0 ||
    (typeof result.stderr === "string" && result.stderr.trim() !== "") ||
    !pathLine || !goosLine || !goarchLine
  ) {
    throw new MutationGateError(
      "GO_BUILD_INFO_INSPECTION_FAILED",
      `trusted executable lacks cmd/go native build facts: ${limitedOutput(result)}`,
    );
  }
  return Object.freeze({
    goos: goosLine.slice("build\tGOOS=".length),
    goarch: goarchLine.slice("build\tGOARCH=".length),
    digest: `sha256:${sha256(Buffer.from(`${output}\n`, "utf8"))}`,
  });
}

export async function admitTrustedGoToolchain(sourceEnvironment = process.env, spawn = spawnSync) {
  const executable = await resolveAdmittedGoExecutable(sourceEnvironment);
  const version = readAdmittedGoVersion(executable, spawn);
  const build = readAdmittedGoBuildFacts(executable, spawn);
  const hostGoos = process.platform === "darwin" ? "darwin" : process.platform;
  const hostGoarch = process.arch === "x64" ? "amd64" : process.arch;
  if (
    version.goos !== build.goos || version.goarch !== build.goarch ||
    version.goos !== hostGoos || version.goarch !== hostGoarch
  ) {
    throw new MutationGateError(
      "GO_TOOLCHAIN_PLATFORM_MISMATCH",
      `version=${version.goos}/${version.goarch} build=${build.goos}/${build.goarch} host=${hostGoos}/${hostGoarch}`,
    );
  }
  return Object.freeze({ executable, version, build });
}

function admittedExecutable(admissionOrToolchain) {
  return admissionOrToolchain?.executable ?? admissionOrToolchain;
}

export function minimalGoEnvironment(cacheRoot, admissionOrToolchain) {
  const admission = admittedExecutable(admissionOrToolchain);
  revalidateAdmittedGoExecutable(admission);
  return Object.freeze({
    // The parent PATH is never forwarded. spawnSync receives the admitted path as an
    // absolute path; this minimal PATH only exposes its admitted real directory.
    PATH: dirname(admission.path),
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
    CGO_ENABLED: "0",
    LANG: "C",
    LC_ALL: "C",
    NO_COLOR: "1",
    TZ: "UTC",
  });
}

function escapeRegularExpression(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
}

function actionCount(actions, target) {
  return actions.filter((action) => action === target).length;
}

export function classifyNamedGoTest({
  status,
  signal = null,
  error = null,
  stdout = "",
  stderr = "",
  testName,
  expectedPackage,
}) {
  if (error || signal || !Number.isInteger(status)) {
    return {
      outcome: NamedTestOutcome.Infrastructure,
      detail: error?.message ?? (signal ? `terminated by ${signal}` : "missing integer exit status"),
    };
  }
  if (typeof expectedPackage !== "string" || expectedPackage.length === 0) {
    return { outcome: NamedTestOutcome.Infrastructure, detail: "expected package identity is missing" };
  }
  if (typeof stderr === "string" && stderr.trim() !== "") {
    return { outcome: NamedTestOutcome.Infrastructure, detail: "go test -json emitted raw stderr" };
  }

  const events = [];
  for (const line of stdout.split(/\r?\n/u)) {
    if (line.trim() === "") continue;
    try {
      events.push(JSON.parse(line));
    } catch {
      return { outcome: NamedTestOutcome.Infrastructure, detail: "go test -json emitted a non-JSON line" };
    }
  }
  if (events.some((event) => event === null || typeof event !== "object" || event.Package !== expectedPackage)) {
    return {
      outcome: NamedTestOutcome.Infrastructure,
      detail: `go test event package differed from ${expectedPackage}`,
    };
  }
  const unexpectedTests = events.filter((event) => event.Test && event.Test !== testName);
  if (unexpectedTests.length > 0) {
    return { outcome: NamedTestOutcome.Infrastructure, detail: "anchored command ran an unexpected named test" };
  }
  const namedActions = events.filter((event) => event.Test === testName).map((event) => event.Action);
  const packageActions = events.filter((event) => !event.Test).map((event) => event.Action);
  const runCount = actionCount(namedActions, "run");
  const passCount = actionCount(namedActions, "pass");
  const failCount = actionCount(namedActions, "fail");
  const skipCount = actionCount(namedActions, "skip");
  const packagePassCount = actionCount(packageActions, "pass");
  const packageFailCount = actionCount(packageActions, "fail");

  if (
    status === 0 && runCount === 1 && passCount === 1 && failCount === 0 && skipCount === 0 &&
    packagePassCount === 1 && packageFailCount === 0
  ) {
    return { outcome: NamedTestOutcome.Pass, detail: `${testName} passed` };
  }
  if (
    status !== 0 && runCount === 1 && failCount === 1 && passCount === 0 && skipCount === 0 &&
    packageFailCount === 1 && packagePassCount === 0
  ) {
    return { outcome: NamedTestOutcome.Failure, detail: `${testName} failed` };
  }
  if (status === 0 && runCount === 0 && passCount === 0 && failCount === 0 && skipCount === 0) {
    return { outcome: NamedTestOutcome.Missing, detail: `${testName} did not run` };
  }
  return {
    outcome: NamedTestOutcome.Infrastructure,
    detail: `${testName} had inconsistent test actions [${namedActions.join(",")}] and package actions [${packageActions.join(",")}] at exit ${status}`,
  };
}

export function requireBaselinePass(classification, mutantID, output = "") {
  if (classification.outcome !== NamedTestOutcome.Pass) {
    const diagnostic = output ? `\n${output}` : "";
    throw new MutationGateError(
      "BASELINE_TEST_NOT_PASSING",
      `${mutantID}: ${classification.outcome}: ${classification.detail}${diagnostic}`,
    );
  }
}

function limitedOutput(result) {
  const combined = `${result.stdout ?? ""}${result.stderr ?? ""}`.trim();
  return combined.length > 4000 ? `${combined.slice(0, 4000)}\n...[truncated]` : combined;
}

export function runNamedGoTest({
  sandbox,
  mutant,
  environment,
  goToolchain,
  expectedPackage,
  spawn = spawnSync,
}) {
  const admission = admittedExecutable(goToolchain);
  if (!admission || typeof admission.path !== "string" || !isAbsolute(admission.path)) {
    return {
      classification: {
        outcome: NamedTestOutcome.Infrastructure,
        detail: "named test did not receive an admitted absolute Go executable",
      },
      output: "",
    };
  }
  const exactPattern = `^${escapeRegularExpression(mutant.testName)}$`;
  let result;
  try {
    result = spawnWithExecutableRevalidation(
      admission,
      ["test", "-json", mutant.package, "-run", exactPattern, "-count=1"],
      {
        cwd: sandbox,
        encoding: "utf8",
        env: environment,
        timeout: 120_000,
      },
      spawn,
    );
  } catch (error) {
    return {
      classification: {
        outcome: NamedTestOutcome.Infrastructure,
        detail: error instanceof MutationGateError ? `${error.code}: ${error.message}` : error.message,
      },
      output: "",
    };
  }
  const classification = classifyNamedGoTest({
    status: result.status,
    signal: result.signal,
    error: result.error,
    stdout: result.stdout,
    stderr: result.stderr,
    testName: mutant.testName,
    expectedPackage,
  });
  return { classification, output: limitedOutput(result) };
}

async function prepareCacheRoots(cacheRoot) {
  for (const directory of ["home", "tmp", "go-tmp", "go-path", "go-build", "go-mod"]) {
    await ensurePrivateDirectory(join(cacheRoot, directory));
  }
}

function expectedPackageFor(mutant) {
  return `${MODULE_IMPORT_PATH}/${mutant.package.slice(2)}`;
}

async function prepareExperiment({ mutant, goToolchain, phase, seedSnapshot }) {
  if (!seedSnapshot) {
    throw new MutationGateError("SEED_SNAPSHOT_MISSING", `${mutant.id}: experiment has no immutable seed`);
  }
  await assertSeedUnchanged(seedSnapshot);
  const sandboxParent = await mkdtemp(join(tmpdir(), `countershape-u1-${phase}-`));
  await chmod(sandboxParent, 0o700);
  const sandbox = join(sandboxParent, "repo");
  const cacheRoot = join(sandboxParent, "cache");
  try {
    await copyRegularAllowlist(seedSnapshot.root, sandbox);
    const sourceManifest = await readAllowlistManifest(sandbox);
    assertManifestEqual(
      sourceManifest,
      seedSnapshot.manifest,
      "EXPERIMENT_SEED_MISMATCH",
      `${mutant.id} ${phase} copy differs from the immutable seed`,
    );
    await createPrivateRoot(cacheRoot);
    await prepareCacheRoots(cacheRoot);
    const environment = minimalGoEnvironment(cacheRoot, goToolchain);
    return {
      sandboxParent,
      sandbox,
      cacheRoot,
      environment,
      goToolchain,
      expectedPackage: expectedPackageFor(mutant),
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
  return runNamedGoTest({
    sandbox: context.sandbox,
    mutant,
    environment: context.environment,
    goToolchain: context.goToolchain,
    expectedPackage: context.expectedPackage,
  });
}

async function assertExperimentManifest(context, expected, code, detail) {
  if (!context.seedSnapshot) return;
  const actual = await readAllowlistManifest(context.sandbox);
  assertManifestEqual(actual, expected, code, detail);
}

export async function runMutant(mutant, goToolchain, dependencies = {}) {
  const prepare = dependencies.prepareExperiment ?? prepareExperiment;
  const cleanup = dependencies.cleanupExperiment ?? cleanupExperiment;
  const runNamed = dependencies.runNamedTest ?? defaultRunNamedTest;
  const applyMutation = dependencies.applyMutation ?? mutateRegularFileNoFollow;
  const usingDefaultPrepare = prepare === prepareExperiment;
  let ownedSeed;
  const seedSnapshot = dependencies.seedSnapshot ?? (usingDefaultPrepare ? (ownedSeed = await createSeedSnapshot()) : undefined);
  try {
    const baselineContext = await prepare({ mutant, goToolchain, phase: "baseline", seedSnapshot });
    const baselineSandbox = baselineContext.sandbox;
    let baseline;
    try {
      baseline = await runNamed(baselineContext, mutant);
      if (baselineContext.seedSnapshot) {
        await assertExperimentManifest(
          baselineContext,
          baselineContext.seedSnapshot.manifest,
          "BASELINE_SOURCE_CHANGED",
          `${mutant.id}: baseline source changed while its named test ran`,
        );
      }
    } finally {
      await cleanup(baselineContext);
    }
    requireBaselinePass(baseline.classification, mutant.id, baseline.output);

    const mutantContext = await prepare({ mutant, goToolchain, phase: "mutant", seedSnapshot });
    if (mutantContext.sandbox === baselineSandbox) {
      await cleanup(mutantContext);
      throw new MutationGateError(
        "EXPERIMENT_ROOT_REUSED",
        `${mutant.id}: baseline and mutant must use distinct fresh roots`,
      );
    }
    let result;
    let mutatedManifest;
    try {
      // No repository code has executed in this copy. Mutate through the admitted
      // no-follow descriptor before the first and only named test process starts.
      await applyMutation(mutantContext.sandbox, mutant);
      if (mutantContext.seedSnapshot) {
        mutatedManifest = await assertExactMutantDelta(mutantContext.sandbox, mutantContext.seedSnapshot, mutant);
      }
      result = await runNamed(mutantContext, mutant);
      if (mutatedManifest) {
        await assertExperimentManifest(
          mutantContext,
          mutatedManifest,
          "MUTANT_SOURCE_CHANGED",
          `${mutant.id}: exact mutant source changed while its named test ran`,
        );
      }
    } finally {
      await cleanup(mutantContext);
    }

    const postControlContext = await prepare({ mutant, goToolchain, phase: "post-control", seedSnapshot });
    if (postControlContext.sandbox === baselineSandbox || postControlContext.sandbox === mutantContext.sandbox) {
      await cleanup(postControlContext);
      throw new MutationGateError(
        "EXPERIMENT_ROOT_REUSED",
        `${mutant.id}: A/B/A controls must use three distinct fresh roots`,
      );
    }
    let postControl;
    try {
      postControl = await runNamed(postControlContext, mutant);
      if (postControlContext.seedSnapshot) {
        await assertExperimentManifest(
          postControlContext,
          postControlContext.seedSnapshot.manifest,
          "POST_CONTROL_SOURCE_CHANGED",
          `${mutant.id}: post-control source changed while its named test ran`,
        );
      }
    } finally {
      await cleanup(postControlContext);
    }
    requireBaselinePass(postControl.classification, `${mutant.id} post-control`, postControl.output);

    if (result.classification.outcome === NamedTestOutcome.Failure) {
      return `KILLED ${mutant.id} (${mutant.package} ${mutant.testName})`;
    }
    if (result.classification.outcome === NamedTestOutcome.Pass) {
      throw new MutationGateError(
        "MUTANT_SURVIVED",
        `${mutant.id}: ${mutant.testName} passed after mutation\n${result.output}`,
      );
    }
    throw new MutationGateError(
      "MUTANT_INFRASTRUCTURE_FAILURE",
      `${mutant.id}: ${result.classification.outcome}: ${result.classification.detail}\n${result.output}`,
    );
  } finally {
    if (ownedSeed) await cleanupSeedSnapshot(ownedSeed);
  }
}

export async function main() {
  assertMutantDefinitionSet();
  await assertManifestMatchesTree(repoRoot);
  const goToolchain = await admitTrustedGoToolchain();
  const seedSnapshot = await createSeedSnapshot();
  const results = [];
  try {
    for (const mutant of MUTANTS) {
      results.push(await runMutant(mutant, goToolchain, { seedSnapshot }));
    }
    await assertSeedUnchanged(seedSnapshot);
  } finally {
    await cleanupSeedSnapshot(seedSnapshot);
  }
  revalidateAdmittedGoExecutable(goToolchain.executable);
  process.stdout.write(`Trusted Go executable realpath: ${goToolchain.executable.path}\n`);
  process.stdout.write(`Trusted Go executable sha256: ${goToolchain.executable.sha256}\n`);
  process.stdout.write(
    `Trusted Go executable identity: format=${goToolchain.executable.format} dev=${goToolchain.executable.dev} inode=${goToolchain.executable.ino} size=${goToolchain.executable.size} mode=${goToolchain.executable.mode.toString(8)}\n`,
  );
  process.stdout.write(`Trusted Go version: ${goToolchain.version.line}\n`);
  process.stdout.write(`Trusted Go build-info digest: ${goToolchain.build.digest}\n`);
  process.stdout.write(`Immutable mutation seed manifest: ${seedSnapshot.manifest.digest}\n`);
  process.stdout.write(
    "Containment notice: mutation copies are private but test processes retain host filesystem and network authority.\n",
  );
  for (const result of results) {
    process.stdout.write(`${result}\n`);
  }
  process.stdout.write(
    `U1 mutation gate killed ${results.length}/${REQUIRED_MUTANT_IDS.length} exact required mutants.\n`,
  );
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(modulePath)) {
  main().catch((error) => {
    process.stderr.write(`${error.stack ?? error}\n`);
    process.exitCode = 1;
  });
}
