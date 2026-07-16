#!/usr/bin/env node

import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { chmod, mkdir, mkdtemp, readFile, realpath, rm } from "node:fs/promises";
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
	"internal/choice/legacy_validation_fixtures_test.go",
	"internal/choice/portable.go",
	"internal/choice/portable_choice_test.go",
  "internal/choice/promotion/authority/authority.go",
  "internal/choice/promotion/internal/publication/authority.go",
  "internal/choice/promotion/internal/publication/authority_test.go",
	"internal/choice/promotion/legacy_snapshot_darwin_test.go",
  "internal/choice/promotion/service.go",
	"internal/choice/projectiontranslate_test.go",
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
	"internal/portablevalue/fuzz_test.go",
	"internal/portablevalue/value.go",
	"internal/portablevalue/value_test.go",
	"internal/projectionprofile/profile.go",
	"internal/projectionprofile/profile_test.go",
	"internal/projectiontranslate/cli.go",
	"internal/projectiontranslate/cli_test.go",
	"internal/projectiontranslate/confirmed.go",
	"internal/projectiontranslate/expectation.go",
	"internal/projectiontranslate/expectation_test.go",
	"internal/projectiontranslate/fuzz_test.go",
	"internal/projectiontranslate/http.go",
	"internal/projectiontranslate/http_test.go",
	"internal/projectiontranslate/profile_roster.go",
	"internal/projectiontranslate/resolve_test.go",
	"internal/projectiontranslate/translate.go",
	"internal/projectiontranslate/translate_test.go",
  "internal/store/head.go",
  "internal/store/head_darwin.go",
  "internal/store/head_test.go",
  "internal/store/head_unsupported.go",
  "internal/store/object_store.go",
  "internal/store/object_store_test.go",
  "internal/store/public_api_test.go",
  "internal/world/fresh_confirmation.go",
]);

export const U6_RUNTIME_SUPPORT_FILES = Object.freeze([
	"spec/examples/v1/choicepoint.valid.json",
	"spec/examples/v1/decision-record.valid.json",
]);

export const U6_SANDBOX_FILE_ALLOWLIST = Object.freeze([
  ...U5_SANDBOX_FILE_ALLOWLIST,
  ...U6_ADDITIONAL_FILES,
	...U6_RUNTIME_SUPPORT_FILES,
]);

export const U6_SOURCE_SCOPE_ROOTS = Object.freeze([
  ...U5_SOURCE_SCOPE_ROOTS,
  "internal/choice",
  "internal/confirmation",
	"internal/portablevalue",
	"internal/projectionprofile",
	"internal/projectiontranslate",
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
	"resolve-http-by-domain-alone",
	"resolve-binding-by-digest-alone",
	"skip-exact-projection-canonical-check",
	"ignore-cli-projection-roster-order",
	"ignore-http-unused-payload-slots",
	"sort-ordered-string-list",
	"join-ordered-string-list",
	"deduplicate-ordered-string-list",
	"decode-cli-bytes-through-utf8",
	"collapse-cli-missing-to-empty",
	"accept-unknown-missing-policy",
	"translate-before-all-proofs-verify",
	"fresh-choicepoint-falls-back-to-legacy",
	"legacy-choicepoint-rebuilt-portable",
	"hardcode-all-fields-as-differing",
	"allow-unselected-custom-field-smuggling",
	"sort-selected-fields-lexically",
	"skip-portable-selected-value-validation",
	"upgrade-legacy-ruling-to-portable-preparation",
	"sort-portable-profile-registry",
	"drift-cli-translator-version-without-mode-bump",
	"drift-http-profile-descriptor-without-mode-bump",
	"skip-portable-proposal-decision-preflight",
	"skip-portable-revision-decision-preflight",
	"skip-custom-expectation-realizability-check",
	"accept-incoherent-cli-completion-sum",
	"skip-cli-stdout-expectation-coherence",
	"skip-http-expectation-realizability",
	"drift-portable-choice-mode-without-mode-bump",
	"drift-portable-expectation-semantics-without-mode-bump",
	"drift-portable-tuple-identity-rule-without-mode-bump",
	"skip-confirmed-proof-byte-admission",
	"skip-blind-alias-admission",
	"skip-choicepoint-receipt-count-admission",
	"skip-choicepoint-receipt-payload-admission",
	"skip-portable-final-decision-budget",
	"skip-decision-receipt-count-admission",
	"skip-decision-receipt-payload-admission",
]);

const REQUIRED_U6_MUTANT_ID_DIGEST = "sha256:6f5f027b533d97180cc26db5ad5973a538c50e2e2aad67c28e7147b4188ccac7";

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
      "\t\t\t\tif allowedField.FieldID == selectedField.text && allowedField.Value.identityKey(r.registry.mode) == candidateValue.identityKey(r.registry.mode) {",
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
	Object.freeze({
		id: "resolve-http-by-domain-alone",
		file: "internal/projectiontranslate/http.go",
		find: "\tif !exactBindingMatch(definition.Binding(), binding) {",
		replace: "\tif definition.Binding().AdapterDomain() != binding.AdapterDomain() {",
		package: "./internal/projectiontranslate",
		testName: "TestResolveFixedHTTPBindingAndRejectNearMatch",
	}),
	Object.freeze({
		id: "resolve-binding-by-digest-alone",
		file: "internal/projectiontranslate/translate.go",
		find: "\treturn candidateDigest.Valid() && requestedDigest.Valid() && candidateDigest == requestedDigest && bytes.Equal(candidateBytes, requestedBytes)",
		replace: "\treturn candidateDigest.Valid() && requestedDigest.Valid() && candidateDigest == requestedDigest",
		package: "./internal/projectiontranslate",
		testName: "TestResolveRequiresDigestAndExactBindingBytes",
	}),
	Object.freeze({
		id: "skip-exact-projection-canonical-check",
		file: "internal/projectiontranslate/translate.go",
		find: "\tif err != nil || !bytes.Equal(canonical, exact) {",
		replace: "\tif err != nil || canonical == nil {",
		package: "./internal/projectiontranslate",
		testName: "TestCLITranslatorRejectsNoncanonicalProjectionBytes",
	}),
	Object.freeze({
		id: "ignore-cli-projection-roster-order",
		file: "internal/projectiontranslate/cli.go",
		find: [
			"\t\tfieldID, parseErr := exactText(fieldMembers[0])",
			"\t\tif parseErr != nil || fieldID != profileFields[index].FieldID { // MUTANT_P07A_IGNORE_CLI_ROSTER_ORDER",
			"\t\t\treturn Tuple{}, refuse(CodeRosterMismatch, fieldID, \"CLI projection field ID or registry order differs\")",
			"\t\t}",
			"\t\tvalue, parseErr := translateCLIValue(profileFields[index], fieldMembers[1])",
			"\t\tif parseErr != nil {",
			"\t\t\treturn Tuple{}, parseErr",
			"\t\t}",
			"\t\tfields[index] = Field{id: fieldID, value: value}",
		].join("\n"),
		replace: [
			"\t\t_, parseErr = exactText(fieldMembers[0])",
			"\t\tif parseErr != nil { // MUTANT_P07A_IGNORE_CLI_ROSTER_ORDER",
			"\t\t\treturn Tuple{}, refuse(CodeRosterMismatch, \"\", \"CLI projection field ID is not text\")",
			"\t\t}",
			"\t\tvalue, parseErr := translateCLIValue(profileFields[index], fieldMembers[1])",
			"\t\tif parseErr != nil {",
			"\t\t\treturn Tuple{}, parseErr",
			"\t\t}",
			"\t\tfields[index] = Field{id: profileFields[index].FieldID, value: value}",
		].join("\n"),
		package: "./internal/projectiontranslate",
		testName: "TestCLITranslatorRejectsSameTypedFieldReordering",
	}),
	Object.freeze({
		id: "ignore-http-unused-payload-slots",
		file: "internal/projectiontranslate/http.go",
		find: "\t\tif tag != \"CANONICAL_JSON\" || descriptor.PortableTag != portablevalue.TagCanonicalJSON || integer != 0 || !emptyStrings || text != \"\" || canonicalJSON == \"\" { // MUTANT_P07A_HTTP_UNUSED_SLOTS",
		replace: "\t\tif tag != \"CANONICAL_JSON\" || descriptor.PortableTag != portablevalue.TagCanonicalJSON || canonicalJSON == \"\" { // MUTANT_P07A_HTTP_UNUSED_SLOTS",
		package: "./internal/projectiontranslate",
		testName: "TestHTTPTranslatorRejectsDirtyMetadataUnusedSlot",
	}),
	Object.freeze({
		id: "sort-ordered-string-list",
		file: "internal/portablevalue/value.go",
		find: "\tcanonical, err := canonicalStringList(values)",
		replace: [
			"\tfor left := 0; left < len(values); left++ {",
			"\t\tfor right := left + 1; right < len(values); right++ {",
			"\t\t\tif values[right] < values[left] { values[left], values[right] = values[right], values[left] }",
			"\t\t}",
			"\t}",
			"\tcanonical, err := canonicalStringList(values[:])",
		].join("\n"),
		package: "./internal/portablevalue",
		testName: "TestOrderedStringListPreservesOrderDuplicatesAndCopies",
	}),
	Object.freeze({
		id: "join-ordered-string-list",
		file: "internal/portablevalue/value.go",
		find: "\tcanonical, err := canonicalStringList(values)",
		replace: [
			"\tjoined := \"\"",
			"\tfor _, value := range values { joined += value }",
			"\tvalues = []string{joined}",
			"\tcanonical, err := canonicalStringList(values[:])",
		].join("\n"),
		package: "./internal/portablevalue",
		testName: "TestOrderedStringListPreservesOrderDuplicatesAndCopies",
	}),
	Object.freeze({
		id: "deduplicate-ordered-string-list",
		file: "internal/portablevalue/value.go",
		find: "\tcanonical, err := canonicalStringList(values)",
		replace: [
			"\tseen := map[string]struct{}{}",
			"\tdeduplicated := make([]string, 0, len(values))",
			"\tfor _, value := range values {",
			"\t\tif _, duplicate := seen[value]; duplicate { continue }",
			"\t\tseen[value] = struct{}{}",
			"\t\tdeduplicated = append(deduplicated, value)",
			"\t}",
			"\tvalues = deduplicated",
			"\tcanonical, err := canonicalStringList(values[:])",
		].join("\n"),
		package: "./internal/portablevalue",
		testName: "TestOrderedStringListPreservesOrderDuplicatesAndCopies",
	}),
	Object.freeze({
		id: "decode-cli-bytes-through-utf8",
		file: "internal/projectiontranslate/cli.go",
		find: "\t\tvalue, err := portablevalue.Bytes(decoded) // MUTANT_P07A_BYTES_THROUGH_UTF8",
		replace: "\t\tvalue, err := portablevalue.String(string(decoded)) // MUTANT_P07A_BYTES_THROUGH_UTF8",
		package: "./internal/projectiontranslate",
		testName: "TestCLITranslationPreservesEveryHistoricalTagAndProfileOrder",
	}),
	Object.freeze({
		id: "collapse-cli-missing-to-empty",
		file: "internal/projectiontranslate/cli.go",
		find: "\t\treturn portablevalue.Missing(), nil",
		replace: [
			"\t\tempty, _ := portablevalue.String(\"\")",
			"\t\treturn empty, nil",
		].join("\n"),
		package: "./internal/projectiontranslate",
		testName: "TestCLITranslationPreservesEveryHistoricalTagAndProfileOrder",
	}),
	Object.freeze({
		id: "accept-unknown-missing-policy",
		file: "internal/projectionprofile/profile.go",
		find: "\tdefault:\n\t\treturn false, false\n\t}\n}\n\nfunc boundedText",
		replace: "\tdefault:\n\t\treturn false, true\n\t}\n}\n\nfunc boundedText",
		package: "./internal/projectionprofile",
		testName: "TestProfileRejectsUnknownMissingPolicy",
	}),
	Object.freeze({
		id: "translate-before-all-proofs-verify",
		file: "internal/projectiontranslate/confirmed.go",
		find: [
			"\t\tfingerprint, err := roster.Verify(proof.CandidateExecutionKey, frozenProjection) // MUTANT_P07A_VERIFY_BEFORE_TRANSLATE",
			"\t\tif err != nil {",
			"\t\t\treturn ConfirmedTranslations{}, refuse(CodeRosterMismatch, candidateKey, \"projection proof does not match its confirmed fingerprint\")",
			"\t\t}",
		].join("\n"),
		replace: [
			"\t\tfingerprint, err := roster.Verify(proof.CandidateExecutionKey, frozenProjection) // MUTANT_P07A_TRANSLATE_BEFORE_ALL_VERIFY",
			"\t\tif err != nil {",
			"\t\t\treturn ConfirmedTranslations{}, refuse(CodeRosterMismatch, candidateKey, \"projection proof does not match its confirmed fingerprint\")",
			"\t\t}",
			"\t\tresolvedEarly, resolveErr := Resolve(binding)",
			"\t\tif resolveErr != nil { return ConfirmedTranslations{}, resolveErr }",
			"\t\tif _, translateErr := resolvedEarly.Translate(frozenProjection); translateErr != nil { return ConfirmedTranslations{}, translateErr }",
		].join("\n"),
		package: "./internal/choice",
		testName: "TestTranslateConfirmedVerifiesCompleteRosterBeforeAnyTranslation",
	}),
	Object.freeze({
		id: "fresh-choicepoint-falls-back-to-legacy",
		file: "internal/choice/choicepoint.go",
		find: "return buildChoicepointRecord(input, choicepointPortable, nil)",
		replace: "return buildChoicepointRecord(input, choicepointLegacyWhole, nil)",
		package: "./internal/choice",
		testName: "TestFreshConstructionFromLegacyEvidenceUsesPortableMode",
	}),
	Object.freeze({
		id: "legacy-choicepoint-rebuilt-portable",
		file: "internal/choice/choicepoint.go",
		find: "mode = choicepointLegacyWhole // MUTANT_P07B_LEGACY_REBUILT_PORTABLE",
		replace: "mode = choicepointPortable // MUTANT_P07B_LEGACY_REBUILT_PORTABLE",
		package: "./internal/choice",
		testName: "TestFreshConstructionFromLegacyEvidenceUsesPortableMode",
	}),
	Object.freeze({
		id: "hardcode-all-fields-as-differing",
		file: "internal/choice/blind.go",
		find: "\t\tvar differs bool",
		replace: "\t\tdiffers := true",
		package: "./internal/choice",
		testName: "TestPortableConfirmedSetPreservesProfileOrderAndExactValues",
	}),
	Object.freeze({
		id: "allow-unselected-custom-field-smuggling",
		file: "internal/choice/validation.go",
		find: [
			"\t\tif _, selectedField := allowed[fieldID.text]; !selectedField { // MUTANT_P07B_CUSTOM_UNSELECTED_FIELD",
			"\t\t\terr := refusal(CodeIncompleteTuple, \"custom expectation contains an unselected field\")",
			"\t\t\terr.FieldID = fieldID.text",
			"\t\t\treturn nil, err",
			"\t\t}",
		].join("\n"),
		replace: [
			"\t\tif _, selectedField := allowed[fieldID.text]; !selectedField { // MUTANT_P07B_CUSTOM_UNSELECTED_FIELD",
			"\t\t\tfieldID = selected[len(values)]",
			"\t\t}",
		].join("\n"),
		package: "./internal/choice",
		testName: "TestSelectedTupleContainsExactlySelectedFieldsAndCannotSmuggleContext",
	}),
	Object.freeze({
		id: "sort-selected-fields-lexically",
		file: "internal/choice/validation.go",
		find: "\tfor _, fieldID := range r.orderedIDs { // MUTANT_P07B_SELECTED_LEXICAL_ORDER",
		replace: [
			"\tordered := append([]string(nil), r.orderedIDs...)",
			"\tsort.Strings(ordered)",
			"\tfor _, fieldID := range ordered { // MUTANT_P07B_SELECTED_LEXICAL_ORDER",
		].join("\n"),
		package: "./internal/choice",
		testName: "TestPortableSelectedFieldsCanonicalizeInProfileOrderNotLexicalOrder",
	}),
	Object.freeze({
		id: "skip-portable-selected-value-validation",
		file: "internal/choice/validation.go",
		find: "\tif err := r.validatePortableValues(selectedOrder, values); err != nil {",
		replace: "\tif err := r.validatePortableValues(selectedOrder, values); false && err != nil {",
		package: "./internal/choice",
		testName: "TestPortableCustomCanonicalJSONRetainsHTTPObjectSourceConstraint",
	}),
	Object.freeze({
		id: "upgrade-legacy-ruling-to-portable-preparation",
		file: "internal/choice/portable.go",
		find: [
			"\tif decision.choicepoint.mode == choicepointLegacyWhole {",
			"\t\treturn PortableRulingInspection{}, refusal(CodeLegacyWholeProjectionNotPortable, \"legacy whole-projection rulings are parseable history only\")",
			"\t}",
		].join("\n"),
		replace: [
			"\tif decision.choicepoint.mode == choicepointLegacyWhole {",
			"\t\tdecision.choicepoint.mode = choicepointPortable",
			"\t\tdecision.choicepoint.confirmed.registry.profileDigest = decision.choicepoint.confirmed.registry.projectionDefinitionDigest",
			"\t}",
		].join("\n"),
		package: "./internal/choice/promotion",
		testName: "TestPreUpgradeLegacyChoiceSnapshotsReopenWithoutPortableUpgrade",
	}),
	Object.freeze({
		id: "sort-portable-profile-registry",
		file: "internal/choice/portable.go",
		find: "newFieldRegistry(definitions, true)",
		replace: "newFieldRegistry(definitions, false)",
		package: "./internal/choice",
		testName: "TestPortableConfirmedSetPreservesProfileOrderAndExactValues",
	}),
	Object.freeze({
		id: "drift-cli-translator-version-without-mode-bump",
		file: "internal/projectiontranslate/cli.go",
		find: "cliTranslatorVersion = \"v1\"",
		replace: "cliTranslatorVersion = \"v1-drift\"",
		package: "./internal/projectiontranslate",
		testName: "TestPortableProfileRosterV1IsRuntimeFrozen",
	}),
	Object.freeze({
		id: "drift-http-profile-descriptor-without-mode-bump",
		file: "internal/projectiontranslate/http.go",
		find: "SourcePath: descriptor.Path(),",
		replace: "SourcePath: append(descriptor.Path(), \"portable-profile-roster-v1-drift\"),",
		package: "./internal/projectiontranslate",
		testName: "TestPortableProfileRosterV1IsRuntimeFrozen",
	}),
	Object.freeze({
		id: "skip-portable-proposal-decision-preflight",
		file: "internal/choice/session.go",
		find: [
			"\t// that would fit only after a smaller post-reveal change is refused here.",
			"\tif err := preflightPortableDecisionBudget(result); err != nil {",
		].join("\n"),
		replace: [
			"\t// that would fit only after a smaller post-reveal change is refused here.",
			"\tif err := preflightPortableDecisionBudget(result); false && err != nil {",
		].join("\n"),
		package: "./internal/choice",
		testName: "TestPortableDraftBudgetNeverDefersStructuralOverflowToFinalize",
	}),
	Object.freeze({
		id: "skip-portable-revision-decision-preflight",
		file: "internal/choice/session.go",
		find: [
			"\tresult.state = SessionPostRevealRecorded",
			"\tif err := preflightPortableDecisionBudget(result); err != nil {",
		].join("\n"),
		replace: [
			"\tresult.state = SessionPostRevealRecorded",
			"\tif err := preflightPortableDecisionBudget(result); false && err != nil {",
		].join("\n"),
		package: "./internal/choice",
		testName: "TestPortableDraftBudgetNeverDefersStructuralOverflowToFinalize",
	}),
	Object.freeze({
		id: "skip-custom-expectation-realizability-check",
		file: "internal/choice/validation.go",
		find: "\tif err := r.validatePortableExpectation(selectedOrder, values, true); err != nil {\n\t\treturn nil, err\n\t}",
		replace: "\tif err := r.validatePortableExpectation(selectedOrder, values, true); false && err != nil {\n\t\treturn nil, err\n\t}",
		package: "./internal/choice",
		testName: "TestPortableCustomExpectationRequiresAdapterRealizableExtension",
	}),
	Object.freeze({
		id: "accept-incoherent-cli-completion-sum",
		file: "internal/projectiontranslate/expectation.go",
		find: "\tif possible == 0 {\n\t\treturn refuse(CodeCustomExpectationNotRealizable, \"\", \"CLI completion fields cannot coexist in one adapter projection\")\n\t}",
		replace: "\tif false && possible == 0 {\n\t\treturn refuse(CodeCustomExpectationNotRealizable, \"\", \"CLI completion fields cannot coexist in one adapter projection\")\n\t}",
		package: "./internal/projectiontranslate",
		testName: "TestPortableExpectationDomainCLICompletionRealizability",
	}),
	Object.freeze({
		id: "skip-cli-stdout-expectation-coherence",
		file: "internal/projectiontranslate/expectation.go",
		find: "\treturn validateCLIStdoutCoherence(profileFields, values)",
		replace: "\treturn nil // Mutant: ignore selected stdout/JSON coherence.",
		package: "./internal/projectiontranslate",
		testName: "TestPortableExpectationDomainCLIStdoutCoherence",
	}),
	Object.freeze({
		id: "skip-http-expectation-realizability",
		file: "internal/projectiontranslate/expectation.go",
		find: "\t\treturn validateHTTPSelectedExpectation(values)",
		replace: "\t\treturn nil // Mutant: accept every profile-typed HTTP selection.",
		package: "./internal/projectiontranslate",
		testName: "TestPortableExpectationDomainHTTPRealizability",
	}),
	Object.freeze({
		id: "drift-portable-choice-mode-without-mode-bump",
		file: "internal/projectiontranslate/expectation.go",
		find: "\tPortableChoiceModeV1 = \"ADAPTER_BOUND_PORTABLE_FIELDS_V1\"",
		replace: "\tPortableChoiceModeV1 = \"ADAPTER_BOUND_PORTABLE_FIELDS_V1_DRIFT\"",
		package: "./internal/projectiontranslate",
		testName: "TestPortableProfileRosterV1IsRuntimeFrozen",
	}),
	Object.freeze({
		id: "drift-portable-expectation-semantics-without-mode-bump",
		file: "internal/projectiontranslate/expectation.go",
		find: "\tportableExpectationSemanticsV1 = \"EXISTS_COMPLETE_ADAPTER_PROJECTION_RESTRICTION_V1\"",
		replace: "\tportableExpectationSemanticsV1 = \"EXISTS_COMPLETE_ADAPTER_PROJECTION_RESTRICTION_V1_DRIFT\"",
		package: "./internal/projectiontranslate",
		testName: "TestPortableProfileRosterV1IsRuntimeFrozen",
	}),
	Object.freeze({
		id: "drift-portable-tuple-identity-rule-without-mode-bump",
		file: "internal/projectiontranslate/profile_roster.go",
		find: "\tportableTupleIdentityRuleV1 = \"FIELD_ID_PROFILE_ORDER_PLUS_TAGGED_EXACT_PAYLOAD_BYTES_V1\"",
		replace: "\tportableTupleIdentityRuleV1 = \"FIELD_ID_PROFILE_ORDER_PLUS_TAGGED_EXACT_PAYLOAD_BYTES_V1_DRIFT\"",
		package: "./internal/projectiontranslate",
		testName: "TestPortableProfileRosterV1IsRuntimeFrozen",
	}),
	Object.freeze({
		id: "skip-confirmed-proof-byte-admission",
		file: "internal/projectiontranslate/confirmed.go",
		find: "\t\tif len(proof.CanonicalProjection) > maxConfirmedProjectionBytes-projectionBytes {\n\t\t\treturn ConfirmedTranslations{}, refuse(CodeTranslationLimit, \"\", \"confirmed projection proofs exceed the aggregate retained-byte ceiling\")\n\t\t}",
		replace: "\t\tif false && len(proof.CanonicalProjection) > maxConfirmedProjectionBytes-projectionBytes {\n\t\t\treturn ConfirmedTranslations{}, refuse(CodeTranslationLimit, \"\", \"confirmed projection proofs exceed the aggregate retained-byte ceiling\")\n\t\t}",
		package: "./internal/choice",
		testName: "TestTranslateConfirmedAdmitsProofBytesBeforeDefensiveCopy",
	}),
	Object.freeze({
		id: "skip-blind-alias-admission",
		file: "internal/choice/blind.go",
		find: "\tif err := v.admitAliases(raw); err != nil {\n\t\treturn nil, nil, err\n\t}",
		replace: "\tif err := v.admitAliases(raw); false && err != nil {\n\t\treturn nil, nil, err\n\t}",
		package: "./internal/choice",
		testName: "TestRulingAliasAdmissionIsBoundedBeforeNormalization",
	}),
	Object.freeze({
		id: "skip-choicepoint-receipt-count-admission",
		file: "internal/choice/choicepoint.go",
		find: "\tif len(receipts) > canon.MaxContainerMembers {\n\t\treturn nil, refusal(CodeInputLimitExceeded, \"choicepoint receipt count exceeds the canonical container profile\")\n\t}",
		replace: "\tif false && len(receipts) > canon.MaxContainerMembers {\n\t\treturn nil, refusal(CodeInputLimitExceeded, \"choicepoint receipt count exceeds the canonical container profile\")\n\t}",
		package: "./internal/choice",
		testName: "TestChoicepointReceiptAdmissionIsBoundedBeforeSorting",
	}),
	Object.freeze({
		id: "skip-choicepoint-receipt-payload-admission",
		file: "internal/choice/choicepoint.go",
		find: "\t\tif !consumeReceiptWireBudget(wire, &remaining) {\n\t\t\treturn nil, refusal(CodeInputLimitExceeded, \"choicepoint receipt payload exceeds the canonical input profile\")\n\t\t}",
		replace: "\t\tif false && !consumeReceiptWireBudget(wire, &remaining) {\n\t\t\treturn nil, refusal(CodeInputLimitExceeded, \"choicepoint receipt payload exceeds the canonical input profile\")\n\t\t}",
		package: "./internal/choice",
		testName: "TestChoicepointReceiptAdmissionIsBoundedBeforeSorting",
	}),
	Object.freeze({
		id: "skip-portable-final-decision-budget",
		file: "internal/choice/decision.go",
		find: "\tif enforcePortableBudget && session.record.mode == choicepointPortable {",
		replace: "\tif false && enforcePortableBudget && session.record.mode == choicepointPortable {",
		package: "./internal/choice",
		testName: "TestPortableDecisionLateBudgetIsExactAndLegacyHistoryKeepsFullCeiling",
	}),
	Object.freeze({
		id: "skip-decision-receipt-count-admission",
		file: "internal/choice/decision.go",
		find: "\tif len(receipts) > canon.MaxContainerMembers {\n\t\treturn nil, refusal(CodeInputLimitExceeded, \"DecisionRecord receipt count exceeds the canonical container profile\")\n\t}",
		replace: "\tif false && len(receipts) > canon.MaxContainerMembers {\n\t\treturn nil, refusal(CodeInputLimitExceeded, \"DecisionRecord receipt count exceeds the canonical container profile\")\n\t}",
		package: "./internal/choice",
		testName: "TestDecisionReceiptAdmissionIsBoundedBeforeSorting",
	}),
	Object.freeze({
		id: "skip-decision-receipt-payload-admission",
		file: "internal/choice/decision.go",
		find: "\t\tif !consumeReceiptWireBudget(wire, &remaining) {\n\t\t\treturn nil, refusal(CodeInputLimitExceeded, \"DecisionRecord receipt payload exceeds the canonical input profile\")\n\t\t}",
		replace: "\t\tif false && !consumeReceiptWireBudget(wire, &remaining) {\n\t\t\treturn nil, refusal(CodeInputLimitExceeded, \"DecisionRecord receipt payload exceeds the canonical input profile\")\n\t\t}",
		package: "./internal/choice",
		testName: "TestDecisionReceiptAdmissionIsBoundedBeforeSorting",
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
	const requiredCount = 55;
  if (!Object.isFrozen(requiredIDs) || requiredIDs.length !== requiredCount || new Set(requiredIDs).size !== requiredCount) {
    throw new MutationGateError("U6_INVALID_REQUIRED_ID_SET", `U6 requires exactly ${requiredCount} unique frozen IDs`);
  }
	const requiredIDDigest = `sha256:${createHash("sha256").update(requiredIDs.join("\n"), "utf8").digest("hex")}`;
	if (requiredIDDigest !== REQUIRED_U6_MUTANT_ID_DIGEST) {
		throw new MutationGateError("U6_REQUIRED_ID_DIGEST_MISMATCH", `${requiredIDDigest} != ${REQUIRED_U6_MUTANT_ID_DIGEST}`);
	}
  if (!Object.isFrozen(reviewedContract) || reviewedContract.length !== requiredCount ||
      reviewedContract.some((tuple) => !Object.isFrozen(tuple))) {
    throw new MutationGateError("U6_INVALID_REVIEWED_MUTANT_CONTRACT", "U6 reviewed tuples must be recursively frozen");
  }
  if (!Object.isFrozen(mutants) || mutants.length !== requiredCount || mutants.some((tuple) => !Object.isFrozen(tuple))) {
		throw new MutationGateError("U6_MUTANT_SET_NOT_IMMUTABLE", `U6/P07A executable tuples must be one recursively frozen set of ${requiredCount}`);
  }
  const ids = mutants.map((mutant) => mutant.id);
  if (!sameArray(ids, requiredIDs) || new Set(ids).size !== requiredCount ||
      !sameArray(reviewedContract.map((tuple) => tuple.id), requiredIDs)) {
    throw new MutationGateError("U6_MUTANT_SET_MISMATCH", "U6 mutant IDs, order, or independent tuple contract changed");
  }
  const sourceMutations = mutants.map((mutant) => `${mutant.file}\0${mutant.find}\0${mutant.replace}`);
  if (new Set(sourceMutations).size !== requiredCount) {
		throw new MutationGateError("U6_DUPLICATE_SOURCE_MUTATION", `U6/P07A requires ${requiredCount} distinct source faults`);
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
		if (!/^\.\/internal\/(?:choice(?:\/promotion)?|confirmation|domain|observe|portablevalue|projectionprofile|projectiontranslate|store)$/u.test(mutant.package)) {
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
		U6_RUNTIME_SUPPORT_FILES,
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
		runtimeSupportFiles: U6_RUNTIME_SUPPORT_FILES,
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
	const realTemporaryRoot = await realpath(tmpdir());
	const sandboxParent = await mkdtemp(join(realTemporaryRoot, `countershape-u6-${phase}-`));
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
	const manifest = await assertU6ManifestMatchesTree(repoRoot);
	for (const file of U6_RUNTIME_SUPPORT_FILES) {
		const entry = manifest.entries.find((candidate) => candidate.file === file);
		assert.ok(entry && entry.bytes > 0 && /^[0-9a-f]{64}$/u.test(entry.sha256), `${file} lacks exact runtime-support manifest facts`);
	}
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
	return `U6/P07A-B mutation self-test passed: ${U6_MUTANTS.length}/${U6_MUTANTS.length} recursively frozen tuples; exact anchors/tests/manifests; runtime support and inherited hostile-copy tripwires; fresh synthetic A/B/A and receipt closure.`;
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
		// Run inherited filesystem/concurrency faults first so host-containment
		// regressions fail early, then newest P07A-B and earlier P07A additions.
		const executionOrder = [
			...U6_MUTANTS.slice(0, 17),
			...U6_MUTANTS.slice(41),
			...U6_MUTANTS.slice(17, 41),
		];
		for (const mutant of executionOrder) {
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
	for (const file of U6_RUNTIME_SUPPORT_FILES) {
		const entry = seed.manifest.entries.find((candidate) => candidate.file === file);
		process.stdout.write(`Runtime support ${file}: ${entry.bytes} bytes sha256:${entry.sha256}\n`);
	}
  process.stdout.write("Containment notice: private mutation copies retain the invoking user's host filesystem and network authority.\n");
  for (const item of results) process.stdout.write(`${item.result}; exact fresh A/B/A closure digest ${item.receipt}\n`);
	process.stdout.write(`U6/P07A-B mutation gate: ${results.length}/${U6_MUTANTS.length} required mutants killed; ${results.length}/${U6_MUTANTS.length} fresh A/B/A receipts.\n`);
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(modulePath)) {
  main().catch((error) => {
    process.stderr.write(`${error.stack ?? error}\n`);
    process.exitCode = 1;
  });
}
