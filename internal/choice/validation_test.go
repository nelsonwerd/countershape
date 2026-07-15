package choice

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
)

func TestValidateRulingRefusesEmptySelectedFields(t *testing.T) {
	_, confirmed := testConfirmed(t, defaultSpecs()...)
	_, err := ValidateRuling(confirmed, RulingInput{
		Action:          ActionAllowObserved,
		SelectedFields:  []string{},
		AllowedObserved: confirmed.Outcomes(),
	})
	assertRefusal(t, err, CodeEmptySelectedFields)
}

func TestAllowObservedDerivesExactCompletePartition(t *testing.T) {
	_, confirmed := testConfirmed(t, defaultSpecs()...)
	selected := findRef(t, confirmed, "candidate:a")
	validated, err := ValidateRuling(confirmed, RulingInput{
		Action:          ActionAllowObserved,
		SelectedFields:  []string{"http.status", "http.body.kind"},
		AllowedObserved: []ConfirmedOutcomeRef{selected},
	})
	if err != nil {
		t.Fatalf("ValidateRuling() error = %v", err)
	}
	compilable := validated.CompileEligibility().(CompilableRuling)
	if got := len(compilable.AllowedOutcomes()); got != 1 {
		t.Fatalf("allowed outcome count = %d, want 1", got)
	}
	if got := len(compilable.DisallowedOutcomes()); got != 2 {
		t.Fatalf("derived disallowed outcome count = %d, want 2", got)
	}
	seen := make(map[string]bool)
	for _, ref := range append(compilable.AllowedOutcomes(), compilable.DisallowedOutcomes()...) {
		if seen[ref.ID().String()] {
			t.Fatalf("confirmed outcome %s was assigned twice", ref.ID().String())
		}
		seen[ref.ID().String()] = true
	}
	if len(seen) != len(confirmed.Outcomes()) {
		t.Fatalf("partition covers %d outcomes, want %d", len(seen), len(confirmed.Outcomes()))
	}
	if validated.Separation().AllowedOutcomeCount != 1 || validated.Separation().DisallowedOutcomeCount != 2 {
		t.Fatalf("separation receipt = %#v", validated.Separation())
	}
}

func TestRulingInputHasNoCallerAuthoredDisallowedSurface(t *testing.T) {
	typeOfInput := reflect.TypeOf(RulingInput{})
	for _, forbidden := range []string{"DisallowedOutcomes", "DisallowedTuples", "AllowedTuples"} {
		if _, exists := typeOfInput.FieldByName(forbidden); exists {
			t.Fatalf("RulingInput exposes forgeable %s", forbidden)
		}
	}
}

func TestCompilableRulingDoesNotAllowCrossProduct(t *testing.T) {
	_, confirmed := testConfirmed(t, defaultSpecs()...)
	validated, err := ValidateRuling(confirmed, RulingInput{
		Action:         ActionAllowObserved,
		SelectedFields: []string{"http.status", "http.body.kind"},
		AllowedObserved: []ConfirmedOutcomeRef{
			findRef(t, confirmed, "candidate:a"),
			findRef(t, confirmed, "candidate:c"),
		},
	})
	if err != nil {
		t.Fatalf("validate ruling: %v", err)
	}
	compilable, ok := validated.CompileEligibility().(CompilableRuling)
	if !ok {
		t.Fatalf("eligibility type = %T, want CompilableRuling", validated.CompileEligibility())
	}

	allowed, err := compilable.Allows(testTuple(t, "404", "not_found"))
	if err != nil || !allowed {
		t.Fatalf("original complete tuple: allowed=%v err=%v", allowed, err)
	}
	crossProduct, err := compilable.Allows(testTuple(t, "404", "authorized_metadata"))
	if err != nil {
		t.Fatalf("cross-product membership: %v", err)
	}
	if crossProduct {
		t.Fatal("independent per-field values synthesized an unapproved complete tuple")
	}
}

func TestAllowObservedRejectsUnconfirmedAndDuplicateRefs(t *testing.T) {
	_, confirmed := testConfirmed(t, defaultSpecs()...)
	_, other := testConfirmed(t,
		outcomeSpec{candidate: "candidate:other", status: "418", kind: "other"},
		outcomeSpec{candidate: "candidate:other-2", status: "419", kind: "other-2"},
	)
	base := RulingInput{
		Action:         ActionAllowObserved,
		SelectedFields: []string{"http.status", "http.body.kind"},
	}

	base.AllowedObserved = []ConfirmedOutcomeRef{other.Outcomes()[0]}
	_, err := ValidateRuling(confirmed, base)
	assertRefusal(t, err, CodeUnconfirmedOutcomeSelection)

	base.AllowedObserved = []ConfirmedOutcomeRef{{}}
	_, err = ValidateRuling(confirmed, base)
	assertRefusal(t, err, CodeUnconfirmedOutcomeSelection)

	ref := findRef(t, confirmed, "candidate:a")
	base.AllowedObserved = []ConfirmedOutcomeRef{ref, ref}
	_, err = ValidateRuling(confirmed, base)
	assertRefusal(t, err, CodeDuplicateObservedSelection)
}

func TestObservedRefIsBoundToItsExactConfirmedUniverse(t *testing.T) {
	_, smaller := testConfirmed(t, defaultSpecs()...)
	largerSpecs := append(defaultSpecs(), outcomeSpec{candidate: "candidate:d", status: "500", kind: "newly_confirmed"})
	_, larger := testConfirmed(t, largerSpecs...)
	sharedButStale := findRef(t, smaller, "candidate:a")
	_, err := ValidateRuling(larger, RulingInput{
		Action:          ActionAllowObserved,
		SelectedFields:  []string{"http.status", "http.body.kind"},
		AllowedObserved: []ConfirmedOutcomeRef{sharedButStale},
	})
	assertRefusal(t, err, CodeUnconfirmedOutcomeSelection)
}

func TestAllowObservedRefusesOmittedOrEmptySelection(t *testing.T) {
	_, confirmed := testConfirmed(t, defaultSpecs()...)
	_, err := ValidateRuling(confirmed, RulingInput{
		Action:         ActionAllowObserved,
		SelectedFields: []string{"http.status"},
	})
	assertRefusal(t, err, CodeOmittedObservedSelections)
	_, err = ValidateRuling(confirmed, RulingInput{
		Action:          ActionAllowObserved,
		SelectedFields:  []string{"http.status"},
		AllowedObserved: []ConfirmedOutcomeRef{},
	})
	assertRefusal(t, err, CodeEmptyObservedSelection)
}

func TestSeparationChecksEveryAllowedDisallowedPair(t *testing.T) {
	_, confirmed := testConfirmed(t,
		outcomeSpec{candidate: "candidate:a", status: "200", kind: "left"},
		outcomeSpec{candidate: "candidate:b", status: "200", kind: "right"},
		outcomeSpec{candidate: "candidate:c", status: "403", kind: "third"},
	)
	_, err := ValidateRuling(confirmed, RulingInput{
		Action:          ActionAllowObserved,
		SelectedFields:  []string{"http.status"},
		AllowedObserved: []ConfirmedOutcomeRef{findRef(t, confirmed, "candidate:a")},
	})
	assertRefusal(t, err, CodeAmbiguousScope)
	var typed *RefusalError
	if !errorsAsRefusal(err, &typed) {
		t.Fatalf("error type = %T, want *RefusalError", err)
	}
	if typed.AllowedIndex < 0 || typed.DisallowedIndex < 0 {
		t.Fatalf("missing pair coordinates: %#v", typed)
	}
}

func TestCustomExpectationRequiresOneBoundReviewAndDerivesAllDisallowed(t *testing.T) {
	_, confirmed := testConfirmed(t, defaultSpecs()...)
	expectation := testTuple(t, "401", "human_authored")
	selected := []string{"http.status", "http.body.kind"}
	review, err := NewCustomExpectationReview(confirmed, selected, expectation, "reviewer@example.test", reviewDigest())
	if err != nil {
		t.Fatalf("NewCustomExpectationReview() error = %v", err)
	}
	validated, err := ValidateRuling(confirmed, RulingInput{
		Action:            ActionCustomExpectation,
		SelectedFields:    selected,
		AllowedObserved:   []ConfirmedOutcomeRef{},
		CustomExpectation: &expectation,
		CustomReview:      review,
	})
	if err != nil {
		t.Fatalf("ValidateRuling(custom) error = %v", err)
	}
	compilable := validated.CompileEligibility().(CompilableRuling)
	if got := len(compilable.AllowedOutcomes()); got != 0 {
		t.Fatalf("custom allowed confirmed outcomes = %d, want 0", got)
	}
	if got := len(compilable.AllowedTuples()); got != 1 {
		t.Fatalf("custom allowed tuples = %d, want 1", got)
	}
	if got := len(compilable.DisallowedOutcomes()); got != len(confirmed.Outcomes()) {
		t.Fatalf("custom disallowed outcomes = %d, want %d", got, len(confirmed.Outcomes()))
	}
	retainedReview, ok := compilable.CustomReview()
	if !ok || retainedReview.Reviewer() != "reviewer@example.test" || retainedReview.EvidenceDigest() != reviewDigest() {
		t.Fatalf("custom review fact was not retained: (%#v,%v)", retainedReview, ok)
	}
	if allowed, allowErr := compilable.Allows(expectation); allowErr != nil || !allowed {
		t.Fatalf("custom membership = (%v,%v), want (true,nil)", allowed, allowErr)
	}
}

func TestCustomExpectationCannotReuseObservedProjection(t *testing.T) {
	_, confirmed := testConfirmed(t, defaultSpecs()...)
	expectation := testTuple(t, "404", "not_found")
	selected := []string{"http.status", "http.body.kind"}
	review, err := NewCustomExpectationReview(confirmed, selected, expectation, "reviewer", reviewDigest())
	if err != nil {
		t.Fatal(err)
	}
	_, err = ValidateRuling(confirmed, RulingInput{
		Action:            ActionCustomExpectation,
		SelectedFields:    selected,
		AllowedObserved:   []ConfirmedOutcomeRef{},
		CustomExpectation: &expectation,
		CustomReview:      review,
	})
	assertRefusal(t, err, CodeCustomExpectationAlreadySeen)
}

func TestCustomReviewIsBoundToExactTupleAndFieldSelection(t *testing.T) {
	_, confirmed := testConfirmed(t, defaultSpecs()...)
	reviewed := testTuple(t, "401", "reviewed")
	review, err := NewCustomExpectationReview(
		confirmed,
		[]string{"http.status", "http.body.kind"},
		reviewed,
		"reviewer",
		reviewDigest(),
	)
	if err != nil {
		t.Fatal(err)
	}
	changed := testTuple(t, "402", "changed-after-review")
	_, err = ValidateRuling(confirmed, RulingInput{
		Action:            ActionCustomExpectation,
		SelectedFields:    []string{"http.status", "http.body.kind"},
		AllowedObserved:   []ConfirmedOutcomeRef{},
		CustomExpectation: &changed,
		CustomReview:      review,
	})
	assertRefusal(t, err, CodeReviewExpectationMismatch)
}

func TestCustomReviewCannotBeReusedAfterConfirmedUniverseChanges(t *testing.T) {
	_, first := testConfirmed(t, defaultSpecs()...)
	largerSpecs := append(defaultSpecs(), outcomeSpec{candidate: "candidate:d", status: "500", kind: "newly_confirmed"})
	_, second := testConfirmed(t, largerSpecs...)
	expectation := testTuple(t, "401", "reviewed")
	fields := []string{"http.status", "http.body.kind"}
	review, err := NewCustomExpectationReview(first, fields, expectation, "reviewer", reviewDigest())
	if err != nil {
		t.Fatal(err)
	}
	_, err = ValidateRuling(second, RulingInput{
		Action:            ActionCustomExpectation,
		SelectedFields:    fields,
		AllowedObserved:   []ConfirmedOutcomeRef{},
		CustomExpectation: &expectation,
		CustomReview:      review,
	})
	assertRefusal(t, err, CodeReviewExpectationMismatch)
}

func TestCustomExpectationRefusesBooleanReviewAndObservedSelection(t *testing.T) {
	_, confirmed := testConfirmed(t, defaultSpecs()...)
	expectation := testTuple(t, "401", "custom")
	base := RulingInput{
		Action:            ActionCustomExpectation,
		SelectedFields:    []string{"http.status", "http.body.kind"},
		AllowedObserved:   []ConfirmedOutcomeRef{},
		CustomExpectation: &expectation,
	}
	_, err := ValidateRuling(confirmed, base)
	assertRefusal(t, err, CodeCustomExpectationUnreviewed)
	review, reviewErr := NewCustomExpectationReview(confirmed, base.SelectedFields, expectation, "reviewer", reviewDigest())
	if reviewErr != nil {
		t.Fatal(reviewErr)
	}
	base.CustomReview = review
	base.AllowedObserved = []ConfirmedOutcomeRef{confirmed.Outcomes()[0]}
	_, err = ValidateRuling(confirmed, base)
	assertRefusal(t, err, CodeCustomExpectationCardinality)
}

func TestIncompleteTuplesCannotReachReviewOrMembership(t *testing.T) {
	_, confirmed := testConfirmed(t, defaultSpecs()...)
	incomplete := CompleteTuple{Fields: []FieldValue{{FieldID: "http.status", Value: mustInteger(t, "401")}}}
	_, err := NewCustomExpectationReview(confirmed, []string{"http.status"}, incomplete, "reviewer", reviewDigest())
	assertRefusal(t, err, CodeIncompleteTuple)

	validated := validateObservedForTest(t, confirmed, []string{"http.status"}, findRef(t, confirmed, "candidate:a"))
	_, err = validated.Allows(incomplete)
	assertRefusal(t, err, CodeIncompleteTuple)
}

func TestFieldRegistryTypedIdentityAndCloneRetainBothDigests(t *testing.T) {
	definition := testProjectionDefinition(t, "identity", defaultProjectionOperations(t), defaultFieldDefinitions())
	registry := definition.Registry()
	if !registry.FieldRegistryDigest().Valid() || registry.ProjectionDefinitionDigest() != definition.Digest() {
		t.Fatalf("registry identities are incomplete: field=%s projection=%s", registry.FieldRegistryDigest().String(), registry.ProjectionDefinitionDigest().String())
	}
	if registry.FieldRegistryDigest() == registry.ProjectionDefinitionDigest() {
		t.Fatal("field registry identity collapsed into the full projection definition identity")
	}

	identity := struct {
		SchemaVersion string                    `json:"schema_version"`
		Kind          string                    `json:"kind"`
		Fields        []fieldDefinitionIdentity `json:"fields"`
	}{
		SchemaVersion: domain.SchemaVersion,
		Kind:          "FieldRegistry",
		Fields:        make([]fieldDefinitionIdentity, 0, len(registry.orderedIDs)),
	}
	for _, id := range registry.orderedIDs {
		field := registry.definitions[id]
		identity.Fields = append(identity.Fields, fieldDefinitionIdentity{
			ID: field.ID, Path: append([]string(nil), field.Path...), Type: string(field.Type),
			AllowMissing: field.AllowMissing, AllowNull: field.AllowNull,
		})
	}
	expected, _, err := canon.DigestTyped("FieldRegistry", identity)
	if err != nil {
		t.Fatal(err)
	}
	if registry.FieldRegistryDigest().String() != expected.String() {
		t.Fatalf("field registry digest = %s, want typed FieldRegistry digest %s", registry.FieldRegistryDigest().String(), expected.String())
	}

	cloned := registry.clone()
	if cloned.FieldRegistryDigest() != registry.FieldRegistryDigest() || cloned.ProjectionDefinitionDigest() != registry.ProjectionDefinitionDigest() {
		t.Fatalf("clone lost identity: field=%s projection=%s", cloned.FieldRegistryDigest().String(), cloned.ProjectionDefinitionDigest().String())
	}
	firstID := cloned.orderedIDs[0]
	cloned.orderedIDs[0] = "mutated"
	clonedField := cloned.definitions[firstID]
	clonedField.Path[0] = "mutated"
	cloned.definitions[firstID] = clonedField
	if registry.orderedIDs[0] == "mutated" || registry.definitions[firstID].Path[0] == "mutated" {
		t.Fatal("field registry clone leaked mutable slice storage")
	}
}

func TestProjectionDefinitionCommitsConfigurationOperationRulesAndPipelineOrder(t *testing.T) {
	fields := defaultFieldDefinitions()
	operations := defaultProjectionOperations(t)
	baseline := testProjectionDefinition(t, "shared", operations, fields)
	configurationChanged := testProjectionDefinition(t, "changed", operations, fields)
	operationRuleChanged := append([]ProjectionOperation(nil), operations...)
	operationRuleChanged[0].RuleDigest = testDomainDigest(t, "choice-projection-operation:changed-rule")
	changedOperation := testProjectionDefinition(t, "shared", operationRuleChanged, fields)
	permutedOperations := []ProjectionOperation{operations[1], operations[0]}
	permuted := testProjectionDefinition(t, "shared", permutedOperations, fields)
	channelPermutationConfig := projectionDefinitionConfigForTest(t, "shared", operations, fields)
	channelPermutationConfig.AcceptedChannels = []string{"stderr", "stdout", "exit"}
	channelPermutation, err := NewProjectionDefinition(channelPermutationConfig)
	if err != nil {
		t.Fatal(err)
	}
	if channelPermutation.Digest() != baseline.Digest() {
		t.Fatal("accepted-channel set permutation changed projection definition identity")
	}

	definitions := []ProjectionDefinition{configurationChanged, changedOperation, permuted}
	for _, other := range definitions {
		if other.Registry().FieldRegistryDigest() != baseline.Registry().FieldRegistryDigest() {
			t.Fatal("non-field projection configuration changed the field-registry digest")
		}
		if other.Digest() == baseline.Digest() {
			t.Fatal("distinct projection configuration collapsed to the baseline full digest")
		}
	}

	baselineRegistry := baseline.Registry()
	inputs := []ProjectionProofInput{
		trustedBytesInputForProjection(t, baseline.Digest(), "candidate:a", projectionBytes(t, "404", "not_found")),
		trustedBytesInputForProjection(t, baseline.Digest(), "candidate:b", projectionBytes(t, "403", "forbidden")),
	}
	roster := testProjectionRoster(t, inputs)
	if _, err := NewConfirmedOutcomeSet(baselineRegistry, roster, inputs); err != nil {
		t.Fatalf("baseline projection definition refused its own roster: %v", err)
	}
	for _, other := range definitions {
		_, err := NewConfirmedOutcomeSet(other.Registry(), roster, inputs)
		assertRefusal(t, err, CodeInvalidFieldRegistry)
	}
}

func TestProjectionAndReviewerInputCapsFailClosed(t *testing.T) {
	tooManyFields := make([]FieldDefinition, maxProjectionFields+1)
	for index := range tooManyFields {
		tooManyFields[index] = FieldDefinition{ID: fmt.Sprintf("field.%d", index), Type: FieldString}
	}
	_, err := NewFieldRegistry(tooManyFields)
	assertRefusal(t, err, CodeInvalidFieldRegistry)
	_, err = NewFieldRegistry([]FieldDefinition{
		{ID: "alias.one", Path: []string{"blob"}, Type: FieldCanonicalJSON},
		{ID: "alias.two", Path: []string{"blob"}, Type: FieldCanonicalJSON},
	})
	assertRefusal(t, err, CodeInvalidFieldRegistry)

	tooManyOperations := make([]ProjectionOperation, maxProjectionOperations+1)
	for index := range tooManyOperations {
		tooManyOperations[index] = ProjectionOperation{
			Name:       fmt.Sprintf("operation-%d", index),
			RuleDigest: testDomainDigest(t, fmt.Sprintf("choice-operation-cap:%d", index)),
		}
	}
	_, err = NewProjectionDefinition(projectionDefinitionConfigForTest(t, "operation-cap", tooManyOperations, defaultFieldDefinitions()))
	assertRefusal(t, err, CodeInvalidFieldRegistry)
	tooManyChannels := projectionDefinitionConfigForTest(t, "channel-cap", defaultProjectionOperations(t), defaultFieldDefinitions())
	tooManyChannels.AcceptedChannels = []string{"exit", "stderr", "stdout", "stdout"}
	_, err = NewProjectionDefinition(tooManyChannels)
	assertRefusal(t, err, CodeInvalidFieldRegistry)

	duplicateOperations := append([]ProjectionOperation(nil), defaultProjectionOperations(t)...)
	duplicateOperations = append(duplicateOperations, ProjectionOperation{
		Name:       duplicateOperations[0].Name,
		RuleDigest: testDomainDigest(t, "choice-projection-operation:duplicate-name"),
	})
	_, err = NewProjectionDefinition(projectionDefinitionConfigForTest(t, "duplicate-operation", duplicateOperations, defaultFieldDefinitions()))
	assertRefusal(t, err, CodeInvalidFieldRegistry)

	registry := testRegistry(t)
	_, err = registry.Resolve(strings.Repeat("x", maxProjectionNameBytes+1))
	assertRefusal(t, err, CodeInputLimitExceeded)
	_, err = registry.resolveSelected([]string{"http.status", "http.body.kind", "extra"})
	assertRefusal(t, err, CodeInputLimitExceeded)
	overfullTuple := testTuple(t, "401", "reviewed")
	overfullTuple.Fields = append(overfullTuple.Fields, FieldValue{FieldID: "extra", Value: mustString(t, "extra")})
	_, err = registry.validateTuple(overfullTuple)
	assertRefusal(t, err, CodeInputLimitExceeded)
	forgedOversize := testTuple(t, "401", "reviewed")
	forgedOversize.Fields[1].Value = ExactValue{tag: ValueString, text: strings.Repeat("x", maxExactStringBytes+1)}
	_, err = registry.validateTuple(forgedOversize)
	assertRefusal(t, err, CodeInputLimitExceeded)
	_, err = StringValue(strings.Repeat("x", maxExactStringBytes+1))
	assertRefusal(t, err, CodeInputLimitExceeded)
	_, err = IntegerValue(strings.Repeat("9", maxExactIntegerBytes+1))
	assertRefusal(t, err, CodeInputLimitExceeded)
	_, err = NewFieldRegistry([]FieldDefinition{{ID: "field", Type: FieldType(strings.Repeat("T", maxFieldTypeBytes+1))}})
	assertRefusal(t, err, CodeInvalidFieldRegistry)
	largeCanonical, canonicalErr := canon.String(strings.Repeat("x", 700*1024))
	if canonicalErr != nil {
		t.Fatal(canonicalErr)
	}
	largeExact, canonicalErr := CanonicalJSONValue(largeCanonical)
	if canonicalErr != nil {
		t.Fatal(canonicalErr)
	}
	largeRegistry := testRegistryForFields(t, []FieldDefinition{
		{ID: "large.one", Type: FieldCanonicalJSON},
		{ID: "large.two", Type: FieldCanonicalJSON},
		{ID: "large.three", Type: FieldCanonicalJSON},
	})
	_, err = largeRegistry.validateTuple(CompleteTuple{Fields: []FieldValue{
		{FieldID: "large.one", Value: largeExact},
		{FieldID: "large.two", Value: largeExact},
		{FieldID: "large.three", Value: largeExact},
	}})
	assertRefusal(t, err, CodeInputLimitExceeded)

	_, confirmed := testConfirmed(t, defaultSpecs()...)
	expectation := testTuple(t, "401", "reviewed")
	selected := []string{"http.status", "http.body.kind"}
	overfullObserved := append(confirmed.Outcomes(), ConfirmedOutcomeRef{})
	_, err = ValidateRuling(confirmed, RulingInput{
		Action:          ActionAllowObserved,
		SelectedFields:  selected,
		AllowedObserved: overfullObserved,
	})
	assertRefusal(t, err, CodeInputLimitExceeded)
	for _, reviewer := range []string{"", "   ", string([]byte{0xff}), strings.Repeat("r", maxReviewerBytes+1)} {
		_, reviewErr := NewCustomExpectationReview(confirmed, selected, expectation, reviewer, reviewDigest())
		assertRefusal(t, reviewErr, CodeInvalidReviewFact)
	}
	_, err = ValidateRuling(confirmed, RulingInput{
		Action: Action(strings.Repeat("A", maxActionBytes+1)), SelectedFields: []string{}, AllowedObserved: []ConfirmedOutcomeRef{},
	})
	assertRefusal(t, err, CodeInputLimitExceeded)
	if _, err := NewCustomExpectationReview(confirmed, selected, expectation, strings.Repeat("r", maxReviewerBytes), reviewDigest()); err != nil {
		t.Fatalf("reviewer at exact byte cap was refused: %v", err)
	}
}

func TestConfirmedOutcomeSetChecksCanonicalBytesFingerprintAndCandidateUniqueness(t *testing.T) {
	registry := testRegistry(t)
	valid := trustedInput(t, outcomeSpec{candidate: "candidate:a", status: "404", kind: "not_found"})
	second := trustedInput(t, outcomeSpec{candidate: "candidate:b", status: "403", kind: "forbidden"})
	baseline := []ProjectionProofInput{valid, second}
	roster := testProjectionRoster(t, baseline)

	tampered := valid
	tampered.CanonicalProjection = projectionBytes(t, "405", "not_found")
	_, err := NewConfirmedOutcomeSet(registry, roster, []ProjectionProofInput{tampered, second})
	assertRefusal(t, err, CodeProjectionFingerprintMismatch)

	noncanonical := valid
	noncanonical.CanonicalProjection = append([]byte(" "), valid.CanonicalProjection...)
	_, err = NewConfirmedOutcomeSet(registry, roster, []ProjectionProofInput{noncanonical, second})
	assertRefusal(t, err, CodeNoncanonicalProjection)

	_, err = NewConfirmedOutcomeSet(registry, roster, []ProjectionProofInput{valid, valid})
	assertRefusal(t, err, CodeDuplicateConfirmedCandidate)

	_, err = NewConfirmedOutcomeSet(registry, roster, []ProjectionProofInput{})
	assertRefusal(t, err, CodeEmptyConfirmedOutcomes)
}

func TestConfirmedOutcomeSetRequiresOpaqueRosterAndExactProofCoverage(t *testing.T) {
	registry := testRegistry(t)
	inputs := []ProjectionProofInput{
		trustedInput(t, outcomeSpec{candidate: "candidate:a", status: "404", kind: "not_found"}),
		trustedInput(t, outcomeSpec{candidate: "candidate:b", status: "403", kind: "forbidden"}),
		trustedInput(t, outcomeSpec{candidate: "candidate:c", status: "200", kind: "authorized_metadata"}),
	}
	roster := testProjectionRoster(t, inputs)

	_, err := NewConfirmedOutcomeSet(registry, compare.ConfirmedProjectionRoster{}, inputs)
	assertRefusal(t, err, CodeInvalidConfirmedOutcomeSet)

	_, err = NewConfirmedOutcomeSet(registry, roster, inputs[:2])
	assertRefusal(t, err, CodeConfirmedProjectionRosterMismatch)

	extra := append([]ProjectionProofInput(nil), inputs...)
	extra = append(extra, trustedInput(t, outcomeSpec{candidate: "candidate:d", status: "500", kind: "extra"}))
	_, err = NewConfirmedOutcomeSet(registry, roster, extra)
	assertRefusal(t, err, CodeConfirmedProjectionRosterMismatch)

	swapped := append([]ProjectionProofInput(nil), inputs...)
	swapped[0].CanonicalProjection = append([]byte(nil), inputs[1].CanonicalProjection...)
	swapped[1].CanonicalProjection = append([]byte(nil), inputs[0].CanonicalProjection...)
	_, err = NewConfirmedOutcomeSet(registry, roster, swapped)
	assertRefusal(t, err, CodeProjectionFingerprintMismatch)

	forgedCandidate := append([]ProjectionProofInput(nil), inputs...)
	forgedCandidate[0].CandidateExecutionKey = testCandidateKey(t, "candidate:forged")
	_, err = NewConfirmedOutcomeSet(registry, roster, forgedCandidate)
	assertRefusal(t, err, CodeConfirmedProjectionRosterMismatch)

	reinterpretingRegistry := testRegistryForFields(t, []FieldDefinition{
		{ID: "http.status", Path: []string{"http.body.kind"}, Type: FieldString},
		{ID: "http.body.kind", Path: []string{"http.status"}, Type: FieldInteger},
	})
	_, err = NewConfirmedOutcomeSet(reinterpretingRegistry, roster, inputs)
	assertRefusal(t, err, CodeInvalidFieldRegistry)
}

func TestConfirmedOutcomeSetRetainsMapAndPreservationIdentity(t *testing.T) {
	registry := testRegistry(t)
	inputs := []ProjectionProofInput{
		trustedInput(t, outcomeSpec{candidate: "candidate:a", status: "404", kind: "not_found"}),
		trustedInput(t, outcomeSpec{candidate: "candidate:b", status: "403", kind: "forbidden"}),
	}
	roster := testProjectionRoster(t, inputs)
	confirmed, err := NewConfirmedOutcomeSet(registry, roster, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.OutcomeMapDigest() != roster.OutcomeMapDigest() {
		t.Fatal("confirmed set lost its authorizing outcome-map identity")
	}
	if confirmed.PreservationDigest() != roster.PreservationDigest() {
		t.Fatal("confirmed set lost its complete labeled preservation identity")
	}
	alternateRoster := testProjectionRosterWithStimulus(t, inputs, "choice-test-alternate-stimulus")
	if roster.PreservationDigest() != alternateRoster.PreservationDigest() {
		t.Fatal("same labeled candidate/projection map changed preservation identity")
	}
	if roster.OutcomeMapDigest() == alternateRoster.OutcomeMapDigest() {
		t.Fatal("different map lineage collapsed to the same outcome-map identity")
	}
	alternate, err := NewConfirmedOutcomeSet(registry, alternateRoster, inputs)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ValidateRuling(alternate, RulingInput{
		Action:          ActionAllowObserved,
		SelectedFields:  []string{"http.status"},
		AllowedObserved: []ConfirmedOutcomeRef{confirmed.Outcomes()[0]},
	})
	assertRefusal(t, err, CodeUnconfirmedOutcomeSelection)
}

func TestConfirmedOutcomeRefsAreDeterministicAndDefensive(t *testing.T) {
	registry := testRegistry(t)
	inputs := []ProjectionProofInput{
		trustedInput(t, outcomeSpec{candidate: "candidate:a", status: "404", kind: "not_found"}),
		trustedInput(t, outcomeSpec{candidate: "candidate:b", status: "403", kind: "forbidden"}),
	}
	roster := testProjectionRoster(t, inputs)
	first, err := NewConfirmedOutcomeSet(registry, roster, inputs)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewConfirmedOutcomeSet(registry, roster, []ProjectionProofInput{inputs[1], inputs[0]})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Outcomes(), second.Outcomes()) {
		t.Fatalf("confirmation order changed refs: %#v != %#v", first.Outcomes(), second.Outcomes())
	}
	refs := first.Outcomes()
	refs[0] = ConfirmedOutcomeRef{}
	if first.Outcomes()[0].ID().String() == "" {
		t.Fatal("Outcomes leaked mutable set storage")
	}
}

func TestCanonicalJSONValuesComputeAndRetainTheirOwnProof(t *testing.T) {
	raw := []byte(`{"a":1,"b":[true,null]}`)
	value, err := CanonicalJSONBytes(raw)
	if err != nil {
		t.Fatalf("CanonicalJSONBytes() error = %v", err)
	}
	if value.Tag() != ValueCanonicalJSON || !strings.HasPrefix(value.Text(), "sha256:") {
		t.Fatalf("canonical JSON exact value = %#v", value)
	}
	original := append([]byte(nil), raw...)
	raw[2] = 'z'
	if !bytes.Equal(value.CanonicalBytes(), original) {
		t.Fatal("caller mutation changed canonical JSON proof bytes")
	}
	proof := value.CanonicalBytes()
	proof[2] = 'y'
	if !bytes.Equal(value.CanonicalBytes(), original) {
		t.Fatal("CanonicalBytes leaked mutable storage")
	}
	second, err := CanonicalJSONValue(mustParseCanon(t, original))
	if err != nil {
		t.Fatal(err)
	}
	if second.Text() != value.Text() {
		t.Fatalf("same canonical value digests differ: %s != %s", second.Text(), value.Text())
	}
	_, err = CanonicalJSONBytes([]byte(`{ "a": 1 }`))
	assertRefusal(t, err, CodeNoncanonicalProjection)
}

func TestIntegerValuesUseCanonSafeIntegerProfile(t *testing.T) {
	for _, accepted := range []string{"0", "-1", "9007199254740991", "-9007199254740991"} {
		if _, err := IntegerValue(accepted); err != nil {
			t.Fatalf("IntegerValue(%q) error = %v", accepted, err)
		}
	}
	for _, rejected := range []string{"-0", "01", "1.0", "9007199254740992", "-9007199254740992"} {
		if _, err := IntegerValue(rejected); !IsRefusal(err, CodeInvalidFieldType) {
			t.Fatalf("IntegerValue(%q) error = %v", rejected, err)
		}
	}
}

func TestMissingNullAndPresentEmptyRemainDistinct(t *testing.T) {
	registry := testRegistryForFields(t, []FieldDefinition{
		{ID: "kind", Type: FieldString, AllowMissing: true, AllowNull: true},
	})
	inputs := []ProjectionProofInput{
		trustedRawInputForRegistry(t, registry, "candidate:missing", `{}`),
		trustedRawInputForRegistry(t, registry, "candidate:null", `{"kind":null}`),
		trustedRawInputForRegistry(t, registry, "candidate:empty", `{"kind":""}`),
	}
	confirmed, err := NewConfirmedOutcomeSet(registry, testProjectionRoster(t, inputs), inputs)
	if err != nil {
		t.Fatal(err)
	}
	validated := validateObservedForTest(t, confirmed, []string{"kind"}, findRef(t, confirmed, "candidate:missing"))
	if allowed, allowErr := validated.Allows(CompleteTuple{Fields: []FieldValue{{FieldID: "kind", Value: MissingValue()}}}); allowErr != nil || !allowed {
		t.Fatalf("missing membership = (%v,%v)", allowed, allowErr)
	}
	empty, _ := StringValue("")
	if allowed, allowErr := validated.Allows(CompleteTuple{Fields: []FieldValue{{FieldID: "kind", Value: empty}}}); allowErr != nil || allowed {
		t.Fatalf("empty membership = (%v,%v)", allowed, allowErr)
	}
}

func TestNoncompilableActionsProduceSealedVariant(t *testing.T) {
	_, confirmed := testConfirmed(t, defaultSpecs()...)
	for _, action := range []Action{ActionRejectAll, ActionDefer, ActionRefine} {
		t.Run(string(action), func(t *testing.T) {
			validated, err := ValidateRuling(confirmed, RulingInput{
				Action:          action,
				SelectedFields:  []string{},
				AllowedObserved: []ConfirmedOutcomeRef{},
			})
			if err != nil {
				t.Fatalf("validate: %v", err)
			}
			if _, ok := validated.CompileEligibility().(NoncompilableRuling); !ok {
				t.Fatalf("eligibility type = %T", validated.CompileEligibility())
			}
			if _, ok := validated.CompileEligibility().(CompilableRuling); ok {
				t.Fatal("noncompilable action reached compilable variant")
			}
		})
	}
}

func TestZeroConfirmedSetAndNoSemanticDefaultsAreRefused(t *testing.T) {
	_, err := ValidateRuling(ConfirmedOutcomeSet{}, RulingInput{
		Action:          ActionDefer,
		SelectedFields:  []string{},
		AllowedObserved: []ConfirmedOutcomeRef{},
	})
	assertRefusal(t, err, CodeInvalidConfirmedOutcomeSet)

	_, confirmed := testConfirmed(t, defaultSpecs()...)
	tests := []struct {
		name  string
		input RulingInput
		code  RefusalCode
	}{
		{name: "action", input: RulingInput{SelectedFields: []string{}, AllowedObserved: []ConfirmedOutcomeRef{}}, code: CodeMissingAction},
		{name: "selected fields", input: RulingInput{Action: ActionDefer, AllowedObserved: []ConfirmedOutcomeRef{}}, code: CodeOmittedSelectedFields},
		{name: "observed selections", input: RulingInput{Action: ActionDefer, SelectedFields: []string{}}, code: CodeOmittedObservedSelections},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, testErr := ValidateRuling(confirmed, test.input)
			assertRefusal(t, testErr, test.code)
		})
	}
}

func TestReturnedPredicateSlicesAreDefensiveCopies(t *testing.T) {
	_, confirmed := testConfirmed(t, defaultSpecs()...)
	compilable := validateObservedForTest(t, confirmed, []string{"http.status"}, findRef(t, confirmed, "candidate:a"))
	fields := compilable.SelectedFields()
	fields[0] = FieldID{text: "changed"}
	tuples := compilable.AllowedTuples()
	tuples[0].Fields[0].FieldID = "changed"
	if compilable.SelectedFields()[0].String() != "http.status" {
		t.Fatal("selected fields leaked mutable storage")
	}
	if compilable.AllowedTuples()[0].Fields[0].FieldID != "http.status" {
		t.Fatal("allowed tuples leaked mutable storage")
	}
}

func TestSelectionAndConfirmationPermutationsDoNotChangePredicate(t *testing.T) {
	registry := testRegistry(t)
	inputs := []ProjectionProofInput{
		trustedInput(t, outcomeSpec{candidate: "candidate:a", status: "404", kind: "not_found"}),
		trustedInput(t, outcomeSpec{candidate: "candidate:b", status: "403", kind: "forbidden"}),
		trustedInput(t, outcomeSpec{candidate: "candidate:c", status: "200", kind: "authorized_metadata"}),
	}
	roster := testProjectionRoster(t, inputs)
	firstSet, err := NewConfirmedOutcomeSet(registry, roster, inputs)
	if err != nil {
		t.Fatal(err)
	}
	secondSet, err := NewConfirmedOutcomeSet(registry, roster, []ProjectionProofInput{inputs[2], inputs[0], inputs[1]})
	if err != nil {
		t.Fatal(err)
	}
	first := validateObservedForTest(t, firstSet,
		[]string{"http.status", "http.body.kind"},
		findRef(t, firstSet, "candidate:a"), findRef(t, firstSet, "candidate:c"),
	)
	second := validateObservedForTest(t, secondSet,
		[]string{"http.body.kind", "http.status"},
		findRef(t, secondSet, "candidate:c"), findRef(t, secondSet, "candidate:a"),
	)
	if !reflect.DeepEqual(first.SelectedFields(), second.SelectedFields()) ||
		!reflect.DeepEqual(first.AllowedTuples(), second.AllowedTuples()) ||
		!reflect.DeepEqual(first.DisallowedTuples(), second.DisallowedTuples()) {
		t.Fatal("input permutation changed the canonical predicate")
	}
}

func TestAllObservedSelectionSubsetsProduceACompleteNonoverlappingPartition(t *testing.T) {
	_, confirmed := testConfirmed(t, defaultSpecs()...)
	refs := confirmed.Outcomes()
	for mask := 1; mask < (1 << len(refs)); mask++ {
		selected := make([]ConfirmedOutcomeRef, 0, len(refs))
		for index, ref := range refs {
			if mask&(1<<index) != 0 {
				selected = append(selected, ref)
			}
		}
		validated := validateObservedForTest(t, confirmed, []string{"http.status", "http.body.kind"}, selected...)
		allowed := validated.AllowedOutcomes()
		disallowed := validated.DisallowedOutcomes()
		assertExactObservedPartition(t, refs, selected, allowed, disallowed)
	}
}

func FuzzCompleteTupleMembershipNeverSynthesizesCrossProduct(f *testing.F) {
	f.Add("left-x", "left-y", "right-x", "right-y")
	f.Add("", "left", "present", "")
	f.Fuzz(func(t *testing.T, firstX, firstY, secondX, secondY string) {
		if len(firstX) > 64 || len(firstY) > 64 || len(secondX) > 64 || len(secondY) > 64 {
			return
		}
		if firstX == secondX || firstY == secondY {
			return
		}
		for _, value := range []string{firstX, firstY, secondX, secondY} {
			if !utf8.ValidString(value) {
				return
			}
		}
		registry := testRegistryForFields(t, []FieldDefinition{
			{ID: "test.x", Type: FieldString},
			{ID: "test.y", Type: FieldString},
		})
		inputs := []ProjectionProofInput{
			trustedStringPairInputForRegistry(t, registry, "candidate:left", firstX, firstY),
			trustedStringPairInputForRegistry(t, registry, "candidate:right", secondX, secondY),
		}
		confirmed, err := NewConfirmedOutcomeSet(registry, testProjectionRoster(t, inputs), inputs)
		if err != nil {
			t.Fatalf("confirmed set: %v", err)
		}
		compilable := validateObservedForTest(t, confirmed, []string{"test.x", "test.y"}, confirmed.Outcomes()...)
		cross := stringPairTuple(t, firstX, secondY)
		allowed, allowErr := compilable.Allows(cross)
		if allowErr != nil {
			t.Fatalf("membership: %v", allowErr)
		}
		if allowed {
			t.Fatal("unlisted cross-product tuple became allowed")
		}
	})
}

func FuzzObservedSelectionAlwaysDerivesTheExactComplement(f *testing.F) {
	f.Add(uint8(1))
	f.Add(uint8(5))
	f.Add(uint8(7))
	_, confirmed := testConfirmed(f, defaultSpecs()...)
	refs := confirmed.Outcomes()
	f.Fuzz(func(t *testing.T, bits uint8) {
		bits &= 7
		if bits == 0 {
			bits = 1
		}
		selected := make([]ConfirmedOutcomeRef, 0, len(refs))
		for index, ref := range refs {
			if bits&(1<<index) != 0 {
				selected = append(selected, ref)
			}
		}
		compilable := validateObservedForTest(t, confirmed, []string{"http.status", "http.body.kind"}, selected...)
		assertExactObservedPartition(t, refs, selected, compilable.AllowedOutcomes(), compilable.DisallowedOutcomes())
	})
}

func assertExactObservedPartition(
	t testing.TB,
	confirmed []ConfirmedOutcomeRef,
	selected []ConfirmedOutcomeRef,
	allowed []ConfirmedOutcomeRef,
	disallowed []ConfirmedOutcomeRef,
) {
	t.Helper()
	confirmedIDs := make(map[string]struct{}, len(confirmed))
	for _, ref := range confirmed {
		id := ref.ID().String()
		if _, duplicate := confirmedIDs[id]; duplicate {
			t.Fatalf("confirmed outcomes duplicate %s", id)
		}
		confirmedIDs[id] = struct{}{}
	}
	selectedIDs := make(map[string]struct{}, len(selected))
	for _, ref := range selected {
		id := ref.ID().String()
		if _, exists := confirmedIDs[id]; !exists {
			t.Fatalf("selected outcome %s is not confirmed", id)
		}
		if _, duplicate := selectedIDs[id]; duplicate {
			t.Fatalf("selected outcomes duplicate %s", id)
		}
		selectedIDs[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(confirmed))
	check := func(ref ConfirmedOutcomeRef, wantSelected bool) {
		id := ref.ID().String()
		if _, exists := confirmedIDs[id]; !exists {
			t.Fatalf("partition contains unconfirmed outcome %s", id)
		}
		_, isSelected := selectedIDs[id]
		if isSelected != wantSelected {
			t.Fatalf("outcome %s selected=%t, but partition side requires selected=%t", id, isSelected, wantSelected)
		}
		if _, duplicate := seen[id]; duplicate {
			t.Fatalf("partition duplicates outcome %s", id)
		}
		seen[id] = struct{}{}
	}
	for _, ref := range allowed {
		check(ref, true)
	}
	for _, ref := range disallowed {
		check(ref, false)
	}
	if len(seen) != len(confirmedIDs) {
		for id := range confirmedIDs {
			if _, exists := seen[id]; !exists {
				t.Fatalf("partition omits confirmed outcome %s", id)
			}
		}
	}
}

type outcomeSpec struct {
	candidate string
	status    string
	kind      string
}

// choiceTestBindings is only fixture memory: public execution APIs deliberately
// refuse to turn a wire-format candidate key back into execution authority.
// Tests that allocate worlds therefore retain the opaque binding created with
// each deterministic candidate key.
var choiceTestBindings sync.Map

var choiceTestProjectionBindings sync.Map

func defaultSpecs() []outcomeSpec {
	return []outcomeSpec{
		{candidate: "candidate:a", status: "404", kind: "not_found"},
		{candidate: "candidate:b", status: "403", kind: "forbidden"},
		{candidate: "candidate:c", status: "200", kind: "authorized_metadata"},
	}
}

func testRegistry(t testing.TB) FieldRegistry {
	t.Helper()
	return testProjectionDefinition(t, "default", defaultProjectionOperations(t), defaultFieldDefinitions()).Registry()
}

func defaultFieldDefinitions() []FieldDefinition {
	return []FieldDefinition{
		{ID: "http.status", Type: FieldInteger},
		{ID: "http.body.kind", Type: FieldString, AllowMissing: true, AllowNull: true},
	}
}

func defaultProjectionOperations(t testing.TB) []ProjectionOperation {
	t.Helper()
	return []ProjectionOperation{
		{Name: "extract-exact-fields", RuleDigest: testDomainDigest(t, "choice-projection-operation:extract")},
		{Name: "emit-canonical-tuple", RuleDigest: testDomainDigest(t, "choice-projection-operation:emit")},
	}
}

func testProjectionDefinition(
	t testing.TB,
	configurationLabel string,
	operations []ProjectionOperation,
	fields []FieldDefinition,
) ProjectionDefinition {
	t.Helper()
	definition, err := NewProjectionDefinition(projectionDefinitionConfigForTest(t, configurationLabel, operations, fields))
	if err != nil {
		t.Fatalf("projection definition: %v", err)
	}
	if previous, loaded := choiceTestProjectionBindings.LoadOrStore(definition.Digest().String(), definition.Binding()); loaded {
		stored, ok := previous.(domain.ProjectionDefinitionBinding)
		if !ok || stored.Digest() != definition.Digest() || !bytes.Equal(stored.CanonicalBytes(), definition.Binding().CanonicalBytes()) {
			t.Fatalf("projection definition fixture collision for %s", definition.Digest())
		}
	}
	return definition
}

func projectionDefinitionConfigForTest(
	t testing.TB,
	configurationLabel string,
	operations []ProjectionOperation,
	fields []FieldDefinition,
) ProjectionDefinitionConfig {
	t.Helper()
	return ProjectionDefinitionConfig{
		AdapterDomain:        domain.AdapterCLI,
		ImplementationDigest: testDomainDigest(t, "choice-projection-implementation"),
		ConfigurationDigest:  testDomainDigest(t, "choice-projection-configuration:"+configurationLabel),
		AcceptedChannels:     []string{"stdout", "exit", "stderr"},
		Operations:           append([]ProjectionOperation(nil), operations...),
		Comparator:           ProjectionComparatorExact,
		Fields:               append([]FieldDefinition(nil), fields...),
	}
}

func testRegistryForFields(t testing.TB, fields []FieldDefinition) FieldRegistry {
	t.Helper()
	return testProjectionDefinition(t, "default", defaultProjectionOperations(t), fields).Registry()
}

func testConfirmed(t testing.TB, specs ...outcomeSpec) (FieldRegistry, ConfirmedOutcomeSet) {
	t.Helper()
	registry := testRegistry(t)
	inputs := make([]ProjectionProofInput, len(specs))
	for index, spec := range specs {
		inputs[index] = trustedInput(t, spec)
	}
	confirmed, err := NewConfirmedOutcomeSet(registry, testProjectionRoster(t, inputs), inputs)
	if err != nil {
		t.Fatalf("NewConfirmedOutcomeSet() error = %v", err)
	}
	return registry, confirmed
}

func trustedInput(t testing.TB, spec outcomeSpec) ProjectionProofInput {
	t.Helper()
	return trustedBytesInput(t, spec.candidate, projectionBytes(t, spec.status, spec.kind))
}

func trustedRawInput(t testing.TB, candidate, projection string) ProjectionProofInput {
	t.Helper()
	return trustedBytesInput(t, candidate, []byte(projection))
}

func trustedRawInputForRegistry(t testing.TB, registry FieldRegistry, candidate, projection string) ProjectionProofInput {
	t.Helper()
	return trustedBytesInputForProjection(t, registry.ProjectionDefinitionDigest(), candidate, []byte(projection))
}

func trustedBytesInput(t testing.TB, candidate string, projection []byte) ProjectionProofInput {
	t.Helper()
	return trustedBytesInputForProjection(t, testRegistry(t).ProjectionDefinitionDigest(), candidate, projection)
}

func trustedBytesInputForProjection(
	t testing.TB,
	projectionDefinitionDigest domain.Digest,
	candidate string,
	projection []byte,
) ProjectionProofInput {
	t.Helper()
	candidateKey := testCandidateKeyForProjection(t, candidate, projectionDefinitionDigest)
	return ProjectionProofInput{
		CandidateExecutionKey: candidateKey,
		CanonicalProjection:   append([]byte(nil), projection...),
	}
}

func testProjectionRoster(t testing.TB, inputs []ProjectionProofInput) compare.ConfirmedProjectionRoster {
	t.Helper()
	return testProjectionRosterWithStimulus(t, inputs, "choice-test-stimulus")
}

func testProjectionRosterWithStimulus(
	t testing.TB,
	inputs []ProjectionProofInput,
	stimulusLabel string,
) compare.ConfirmedProjectionRoster {
	t.Helper()
	envelope := choiceTestEnvelope(t)
	if len(inputs) == 0 {
		t.Fatal("projection roster fixture requires at least one input")
	}
	firstBinding := testBindingForKey(t, inputs[0].CandidateExecutionKey)
	projectionDefinitionDigest := firstBinding.Identity().ProjectionDefinitionDigest
	plan := choiceTestPlan(t, envelope, projectionDefinitionDigest)
	stimulus := testDomainDigest(t, stimulusLabel)
	expected := make([]domain.CandidateExecutionKey, len(inputs))
	for index, input := range inputs {
		binding := testBindingForKey(t, input.CandidateExecutionKey)
		if binding.Identity().ProjectionDefinitionDigest != projectionDefinitionDigest || binding.Identity().WorldPlanDigest != plan.Digest() {
			t.Fatal("projection roster inputs mix projection definitions or world plans")
		}
		expected[index] = input.CandidateExecutionKey
	}
	batches := testConfirmationBatches(t, plan, envelope, stimulus, inputs)
	outcomeMap, err := compare.NewCandidateOutcomeMap(stimulus, envelope.Digest(), expected, batches)
	if err != nil {
		t.Fatalf("candidate outcome map: %v", err)
	}
	confirmed, err := compare.RequireConfirmedOutcomeMap(outcomeMap)
	if err != nil {
		t.Fatalf("confirmed outcome map: %v", err)
	}
	return confirmed.ProjectionRoster()
}

func choiceTestEnvelope(t testing.TB) domain.ComparisonEnvelope {
	t.Helper()
	envelope, err := domain.NewComparisonEnvelope(domain.ComparisonEnvelopeConfig{
		Version:       "choice-test/v1",
		Measured:      []domain.MeasuredDimension{{Name: "os", Source: domain.MeasuredWorldInstance, Comparison: domain.CompareExact}},
		RequiredEqual: []domain.RequiredEqualDimension{{Name: "os"}},
		Uncontrolled:  []string{"scheduler"},
	})
	if err != nil {
		t.Fatalf("comparison envelope: %v", err)
	}
	return envelope
}

func choiceTestPlan(t testing.TB, envelope domain.ComparisonEnvelope, projectionDefinitionDigest domain.Digest) domain.WorldPlan {
	t.Helper()
	stored, ok := choiceTestProjectionBindings.Load(projectionDefinitionDigest.String())
	if !ok {
		t.Fatalf("projection definition %s has no retained binding", projectionDefinitionDigest)
	}
	projectionDefinition, ok := stored.(domain.ProjectionDefinitionBinding)
	if !ok || !projectionDefinition.Valid() || projectionDefinition.Digest() != projectionDefinitionDigest {
		t.Fatalf("projection definition %s retained an invalid binding", projectionDefinitionDigest)
	}
	plan, err := domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest: testDomainDigest(t, "choice-candidate-set"), MaterializationPolicyDigest: testDomainDigest(t, "choice-materialization"),
		ComparisonEnvelopeDigest: envelope.Digest(),
		Adapter:                  domain.Adapter{Domain: domain.AdapterCLI, AdapterVersion: "cli/v1", RunnerDigest: testDomainDigest(t, "choice-runner")},
		ExecutionShape:           domain.OneCLIInvocation, StartArgv: []string{"node", "fixture/cli.mjs", "--mode", "test"}, SetupArgv: []string{},
		Environment: []domain.EnvironmentEntry{{Name: "LANG", Value: "C"}}, SecretSlots: []domain.SecretSlot{},
		FixtureRecipeDigest: testDomainDigest(t, "choice-fixture"), Readiness: domain.Readiness{Kind: domain.ReadinessNone},
		CapturePolicyDigest: testDomainDigest(t, "choice-capture"), ProjectionDefinition: projectionDefinition,
		RepeatSchedule: domain.RepeatSchedule{DiscoveryRepeats: 3, ConfirmationRepeats: 3,
			Concurrency: domain.ScheduleSequential, Rotation: domain.ScheduleRotationStartByRepetitionV1},
		RequiredTools: []domain.RequiredTool{{Name: "node", VersionConstraint: "executed-major-only"}},
		Budgets: domain.Budgets{
			CandidateCount: 4, MaterializedEntryCount: 100, MaterializedBytesPerWorld: 1 << 20, SingleBlobBytes: 1 << 18,
			ReadinessMS: 0, ProbeMS: 1000, TeardownMS: 1000, StdoutBytes: 1 << 16, StderrBytes: 1 << 16,
			HTTPBodyBytes: 1 << 16, ProposedShrinkStimuli: 10, TotalCandidateTrials: 100, ShrinkWallMS: 60_000,
		},
	})
	if err != nil {
		t.Fatalf("choice world plan: %v", err)
	}
	return plan
}

func testConfirmationBatches(
	t testing.TB,
	plan domain.WorldPlan,
	envelope domain.ComparisonEnvelope,
	stimulus domain.Digest,
	inputs []ProjectionProofInput,
) []observe.StableBatch {
	t.Helper()
	trialsByCandidate := make([][]observe.TrialFact, len(inputs))
	admissionDigests := make([]domain.Digest, 0, 3)
	for repeat := 0; repeat < 3; repeat++ {
		worlds := make([]domain.WorldInstance, len(inputs))
		attempts := make([]domain.FinalizedAttempt, len(inputs))
		measurements := make([]domain.InstanceMeasurements, len(inputs))
		measuredValue, err := canon.String("fixed:test-os")
		if err != nil {
			t.Fatalf("measurement value: %v", err)
		}
		for candidateIndex, input := range inputs {
			binding := testBindingForKey(t, input.CandidateExecutionKey)
			ordinal := repeat*len(inputs) + candidateIndex
			identity := fmt.Sprintf("%s-%d", input.CandidateExecutionKey.String(), ordinal)
			attemptDigest := testDomainDigest(t, "attempt-"+identity)
			attempts[candidateIndex] = testFinalizedAttempt(t, attemptDigest)
			world, worldErr := domain.NewWorldInstance(plan, binding, domain.WorldInstanceConfig{
				StimulusDigest:        stimulus,
				AttemptArtifactDigest: attemptDigest,
				Purpose:               domain.AttemptConfirmation,
				InstanceNonce:         "nonce:" + identity,
				ScheduleOrdinal:       ordinal,
			})
			if worldErr != nil {
				t.Fatalf("world instance: %v", worldErr)
			}
			worlds[candidateIndex] = world
			row, measurementErr := domain.NewInstanceMeasurements(envelope, world, []domain.MeasurementValue{{
				Name: "os", Source: domain.MeasuredWorldInstance, Value: measuredValue,
			}})
			if measurementErr != nil {
				t.Fatalf("instance measurements: %v", measurementErr)
			}
			measurements[candidateIndex] = row
		}

		assessment, err := domain.AssessComparison(envelope, measurements)
		if err != nil {
			t.Fatalf("comparison assessment: %v", err)
		}
		admitted, ok := assessment.(domain.AdmittedComparison)
		if !ok {
			t.Fatalf("comparison assessment = %T, want admitted", assessment)
		}
		for _, previous := range admissionDigests {
			if admitted.Digest() == previous {
				t.Fatalf("fresh comparison matrices collapsed to one admission digest")
			}
		}
		admissionDigests = append(admissionDigests, admitted.Digest())

		for candidateIndex, input := range inputs {
			token, tokenErr := admitted.AdmissionFor(measurements[candidateIndex])
			if tokenErr != nil {
				t.Fatalf("comparison admission token: %v", tokenErr)
			}
			ordinal := repeat*len(inputs) + candidateIndex
			identity := fmt.Sprintf("%s-%d", input.CandidateExecutionKey.String(), ordinal)
			observationDigest := testDomainDigest(t, "observation-"+identity)
			capture, captureErr := observe.NewStructuralCapture(
				worlds[candidateIndex], attempts[candidateIndex],
				observationDigest,
				input.CanonicalProjection,
				testProjectionDerivation(t, worlds[candidateIndex], observationDigest, input.CanonicalProjection),
			)
			if captureErr != nil {
				t.Fatalf("structural capture: %v", captureErr)
			}
			trial, trialErr := observe.NewCapturedTrial(worlds[candidateIndex], attempts[candidateIndex], token, capture)
			if trialErr != nil {
				t.Fatalf("captured trial: %v", trialErr)
			}
			trialsByCandidate[candidateIndex] = append(trialsByCandidate[candidateIndex], trial)
		}
	}

	batches := make([]observe.StableBatch, len(inputs))
	for index := range inputs {
		batch, err := observe.Classify(observe.BatchInput{Trials: trialsByCandidate[index]})
		if err != nil {
			t.Fatalf("confirmation batch: %v", err)
		}
		batches[index] = batch
	}
	for index := 1; index < len(batches); index++ {
		if !reflect.DeepEqual(batches[0].AdmissionDigests(), batches[index].AdmissionDigests()) {
			t.Fatalf("candidate batches do not share the exact comparison-matrix admission set")
		}
	}
	return batches
}

func testFinalizedAttempt(t testing.TB, artifactDigest domain.Digest) domain.FinalizedAttempt {
	t.Helper()
	attempt, err := domain.NewAttempt("attempt:"+artifactDigest.String(), artifactDigest, domain.AttemptConfirmation)
	if err != nil {
		t.Fatalf("new attempt: %v", err)
	}
	for _, state := range []domain.AttemptState{
		domain.AttemptMaterializing,
		domain.AttemptStarting,
		domain.AttemptReady,
		domain.AttemptProbing,
		domain.AttemptCapturing,
		domain.AttemptTearingDown,
		domain.AttemptFinalized,
	} {
		attempt, err = attempt.Advance(state)
		if err != nil {
			t.Fatalf("advance attempt to %s: %v", state, err)
		}
	}
	finalized, err := attempt.FinalizedEvidence()
	if err != nil {
		t.Fatalf("finalized attempt evidence: %v", err)
	}
	return finalized
}

func testDomainDigest(t testing.TB, label string) domain.Digest {
	t.Helper()
	digest, err := canon.DigestBytes("ChoiceTestFixture", []byte(label))
	if err != nil {
		t.Fatalf("fixture digest %q: %v", label, err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		t.Fatalf("fixture domain digest %q: %v", label, err)
	}
	return parsed
}

func testProjectionDerivation(
	t testing.TB,
	world domain.WorldInstance,
	observationDigest domain.Digest,
	projection []byte,
) observe.ProjectionDerivation {
	t.Helper()
	derivation, err := observe.NewProjectionDerivation(
		observationDigest,
		world.ProjectionDefinitionDigest(),
		projection,
		[]byte(`{"kind":"TEST_PROJECTION_DERIVATION","operations":["test"],"source_links":["test"]}`),
	)
	if err != nil {
		t.Fatalf("projection derivation: %v", err)
	}
	return derivation
}

func projectionBytes(t testing.TB, status, kind string) []byte {
	t.Helper()
	statusValue, err := canon.IntegerFromString(status)
	if err != nil {
		t.Fatalf("status %q: %v", status, err)
	}
	kindValue, err := canon.String(kind)
	if err != nil {
		t.Fatalf("kind %q: %v", kind, err)
	}
	projection, err := canon.Object(
		canon.Member{Name: "http.status", Value: statusValue},
		canon.Member{Name: "http.body.kind", Value: kindValue},
	)
	if err != nil {
		t.Fatalf("projection object: %v", err)
	}
	return projection.Canonical()
}

func trustedStringPairInput(t testing.TB, candidate, x, y string) ProjectionProofInput {
	t.Helper()
	return trustedStringPairInputForRegistry(t, testRegistry(t), candidate, x, y)
}

func trustedStringPairInputForRegistry(t testing.TB, registry FieldRegistry, candidate, x, y string) ProjectionProofInput {
	t.Helper()
	xValue, err := canon.String(x)
	if err != nil {
		t.Fatalf("x string: %v", err)
	}
	yValue, err := canon.String(y)
	if err != nil {
		t.Fatalf("y string: %v", err)
	}
	projection, err := canon.Object(
		canon.Member{Name: "test.x", Value: xValue},
		canon.Member{Name: "test.y", Value: yValue},
	)
	if err != nil {
		t.Fatalf("pair projection: %v", err)
	}
	return trustedBytesInputForProjection(t, registry.ProjectionDefinitionDigest(), candidate, projection.Canonical())
}

func testTuple(t testing.TB, status, kind string) CompleteTuple {
	t.Helper()
	return CompleteTuple{Fields: []FieldValue{
		{FieldID: "http.status", Value: mustInteger(t, status)},
		{FieldID: "http.body.kind", Value: mustString(t, kind)},
	}}
}

func stringPairTuple(t testing.TB, x, y string) CompleteTuple {
	t.Helper()
	return CompleteTuple{Fields: []FieldValue{
		{FieldID: "test.x", Value: mustString(t, x)},
		{FieldID: "test.y", Value: mustString(t, y)},
	}}
}

func mustInteger(t testing.TB, value string) ExactValue {
	t.Helper()
	exact, err := IntegerValue(value)
	if err != nil {
		t.Fatalf("integer %q: %v", value, err)
	}
	return exact
}

func mustString(t testing.TB, value string) ExactValue {
	t.Helper()
	exact, err := StringValue(value)
	if err != nil {
		t.Fatalf("string %q: %v", value, err)
	}
	return exact
}

func mustParseCanon(t testing.TB, input []byte) canon.Value {
	t.Helper()
	value, err := canon.Parse(input)
	if err != nil {
		t.Fatalf("canon.Parse(%q): %v", input, err)
	}
	return value
}

func findRef(t testing.TB, confirmed ConfirmedOutcomeSet, candidate string) ConfirmedOutcomeRef {
	t.Helper()
	wanted := testCandidateKeyForProjection(t, candidate, confirmed.registry.ProjectionDefinitionDigest())
	for _, ref := range confirmed.Outcomes() {
		if ref.CandidateExecutionKey() == wanted {
			return ref
		}
	}
	t.Fatalf("confirmed set does not contain %s", candidate)
	return ConfirmedOutcomeRef{}
}

func testCandidateKey(t testing.TB, shorthand string) domain.CandidateExecutionKey {
	t.Helper()
	return testCandidateKeyForProjection(t, shorthand, testRegistry(t).ProjectionDefinitionDigest())
}

func testCandidateKeyForProjection(
	t testing.TB,
	shorthand string,
	projectionDefinitionDigest domain.Digest,
) domain.CandidateExecutionKey {
	t.Helper()
	envelope := choiceTestEnvelope(t)
	plan := choiceTestPlan(t, envelope, projectionDefinitionDigest)
	binding, err := domain.NewCandidateExecutionBinding(domain.CandidateExecutionIdentity{
		TreeIdentityDigest:          testDomainDigest(t, "choice-tree:"+shorthand),
		MaterializationPolicyDigest: plan.MaterializationPolicyDigest(),
		WorldPlanDigest:             plan.Digest(),
		AdapterDigest:               plan.AdapterDigest(),
		RunnerDigest:                plan.Adapter().RunnerDigest,
		ProjectionDefinitionDigest:  plan.ProjectionDefinitionDigest(),
	})
	if err != nil {
		t.Fatalf("candidate execution binding %q: %v", shorthand, err)
	}
	key := binding.Key()
	if previous, loaded := choiceTestBindings.LoadOrStore(key.String(), binding); loaded {
		stored, ok := previous.(domain.CandidateExecutionBinding)
		if !ok || stored.Key() != key || !bytes.Equal(stored.CanonicalBytes(), binding.CanonicalBytes()) {
			t.Fatalf("candidate fixture binding collision for %q", shorthand)
		}
	}
	return key
}

func testBindingForKey(t testing.TB, key domain.CandidateExecutionKey) domain.CandidateExecutionBinding {
	t.Helper()
	value, ok := choiceTestBindings.Load(key.String())
	if !ok {
		t.Fatalf("candidate key %s has no retained execution binding", key.String())
	}
	binding, ok := value.(domain.CandidateExecutionBinding)
	if !ok || !binding.Valid() || binding.Key() != key {
		t.Fatalf("candidate key %s resolved to an invalid execution binding", key.String())
	}
	return binding
}

func validateObservedForTest(t testing.TB, confirmed ConfirmedOutcomeSet, fields []string, refs ...ConfirmedOutcomeRef) CompilableRuling {
	t.Helper()
	validated, err := ValidateRuling(confirmed, RulingInput{
		Action:          ActionAllowObserved,
		SelectedFields:  fields,
		AllowedObserved: refs,
	})
	if err != nil {
		t.Fatalf("ValidateRuling() error = %v", err)
	}
	compilable, ok := validated.CompileEligibility().(CompilableRuling)
	if !ok {
		t.Fatalf("eligibility type = %T", validated.CompileEligibility())
	}
	return compilable
}

func reviewDigest() domain.Digest {
	return domain.MustDigest("sha256:" + strings.Repeat("a", 64))
}

func assertRefusal(t testing.TB, err error, code RefusalCode) {
	t.Helper()
	if !IsRefusal(err, code) {
		t.Fatalf("error = %v, want refusal %s", err, code)
	}
}

func errorsAsRefusal(err error, target **RefusalError) bool {
	if err == nil {
		return false
	}
	typed, ok := err.(*RefusalError)
	if ok {
		*target = typed
	}
	return ok
}
