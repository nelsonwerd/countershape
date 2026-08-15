package server

import (
	"bytes"
	"errors"
	"testing"

	"github.com/nelsonwerd/countershape/internal/choice"
)

func TestStudioStateUsesThePackageChoiceSessionForTheCompleteRulingFlow(t *testing.T) {
	state, err := newStudioState(StateDecisionReady, bytes.NewReader(bytes.Repeat([]byte{0x5a}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	draft := DraftInput{
		Action:                string(choice.ActionAllowObserved),
		AllowedAliases:        []string{state.blind.Cards()[0].Alias},
		FieldAcknowledgements: []FieldAcknowledgement{},
		CustomValues:          []CustomValueInput{},
	}
	for _, field := range state.blind.DifferingFields() {
		draft.FieldAcknowledgements = append(draft.FieldAcknowledgements, FieldAcknowledgement{FieldID: field, Disposition: "ASSERT"})
	}
	if _, err := state.propose(state.revisionDigest(), draft); !hasAPIChoiceCode(err, choice.CodeRequiredSurfaceNotVisited) {
		t.Fatalf("early proposal error = %v", err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness,
		choice.SurfaceMinimizedWitness,
		choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations,
		choice.SurfaceNonassertedFields,
	} {
		if _, err := state.visit(state.revisionDigest(), surface); err != nil {
			t.Fatalf("Visit(%s) error = %v", surface, err)
		}
	}
	weak := cloneDraft(draft)
	for index := range weak.FieldAcknowledgements {
		weak.FieldAcknowledgements[index].Disposition = "CONTEXT"
	}
	if _, err := state.propose(state.revisionDigest(), weak); !hasAPIChoiceCode(err, choice.CodeEmptySelectedFields) {
		t.Fatalf("weak-field proposal error = %v", err)
	}
	if _, err := state.propose(state.revisionDigest(), draft); err != nil {
		t.Fatalf("propose error = %v", err)
	}
	if _, err := state.revealProvenance(state.revisionDigest()); err != nil {
		t.Fatalf("reveal error = %v", err)
	}
	if _, err := state.finalize(state.revisionDigest(), "operator", ""); !hasAPIChoiceCode(err, choice.CodeInvalidSessionState) {
		t.Fatalf("early finalize error = %v", err)
	}
	if _, err := state.visit(state.revisionDigest(), choice.SurfaceProvenance); err != nil {
		t.Fatalf("provenance visit error = %v", err)
	}
	if _, err := state.revise(state.revisionDigest(), draft, ""); err != nil {
		t.Fatalf("affirm error = %v", err)
	}
	response, err := state.finalize(state.revisionDigest(), "operator", "bounded local annotation")
	if err != nil {
		t.Fatalf("finalize error = %v", err)
	}
	if response.Result == nil || response.Result.Action != string(choice.ActionAllowObserved) || len(response.Result.EmittedFiles) != 0 || response.SessionState != string(choice.SessionFinalized) {
		t.Fatalf("final response = %#v", response)
	}
	if _, err := state.visit(response.RevisionDigest, choice.SurfaceProvenance); err == nil {
		t.Fatal("finalized state accepted a further transition")
	}
}

func TestStudioPresentationFixtureRosterIsClosedAndExplicit(t *testing.T) {
	for presentation, definition := range stateDefinitions {
		state, err := newStudioState(presentation, bytes.NewReader(bytes.Repeat([]byte{byte(len(presentation))}, 32)))
		if err != nil {
			t.Fatalf("newStudioState(%s) error = %v", presentation, err)
		}
		response := state.benchResponse()
		if response.Presentation != string(presentation) || response.StudyState != definition.StudyState || response.CandidateState != definition.CandidateState || response.ChoicepointState != definition.ChoicepointState || response.Mutable != definition.Mutable || response.FutureAuthority != definition.FutureAuthority {
			t.Fatalf("presentation %s drifted: %#v", presentation, response)
		}
		if (presentation == StateResolved || presentation == StateRejectAllResolved || presentation == StateDeferred) && response.Result == nil {
			t.Fatalf("resolved presentation %s omitted its valid package decision", presentation)
		}
		if response.Result != nil && (response.Result.SelectedFields == nil || response.Result.NonassertedFields == nil || response.Result.EmittedFiles == nil) {
			t.Fatalf("resolved presentation %s collapsed an explicit result array to null", presentation)
		}
		if !definition.Mutable {
			if _, err := state.visit(response.RevisionDigest, choice.SurfaceOriginalWitness); err == nil || !stringsContainsError(err, "STUDIO_STATE_IMMUTABLE") {
				t.Fatalf("immutable presentation %s visit error = %v", presentation, err)
			}
		}
	}
	if _, err := newStudioState(PresentationState("foreign"), bytes.NewReader(make([]byte, 32))); err == nil {
		t.Fatal("foreign presentation state was admitted")
	}
}

func hasAPIChoiceCode(err error, code choice.RefusalCode) bool {
	var apiError *APIError
	return errors.As(err, &apiError) && apiError.Code == "CHOICE_"+string(code)
}

func stringsContainsError(err error, value string) bool {
	return err != nil && bytes.Contains([]byte(err.Error()), []byte(value))
}
