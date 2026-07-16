package choice

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
)

func TestDecisionRecordPreservesRationalizedPostRevealChange(t *testing.T) {
	record := checkedChoicepointExample(t)
	session := visitedBlindSession(t, record)
	cards := session.BlindDTO().Cards()
	if len(cards) < 2 {
		t.Fatal("checked Choicepoint requires at least two blind cards")
	}
	provisional := RulingDraftInput{
		Action: ActionAllowObserved, SelectedFields: []string{WholeProjectionFieldID},
		AllowedAliases: []string{cards[0].Alias},
	}
	session = proposedRevealedSession(t, session, provisional)
	final := RulingDraftInput{
		Action: ActionAllowObserved, SelectedFields: []string{WholeProjectionFieldID},
		AllowedAliases: []string{cards[1].Alias},
	}
	const rationale = "Provenance changed which exact witnessed tuple I am willing to preserve."
	var err error
	session, err = session.Revise(final, rationale)
	if err != nil {
		t.Fatal(err)
	}
	finalized, decision, err := session.Finalize("local-roundtrip-operator", "changed after reveal", []domain.ReceiptReference{})
	if err != nil || finalized.State() != SessionFinalized || !decision.ChangedAfterReveal() ||
		decision.PostRevealChangeRationale() != rationale {
		t.Fatalf("rationalized changed decision = state %s changed=%t rationale=%q err=%v",
			finalized.State(), decision.ChangedAfterReveal(), decision.PostRevealChangeRationale(), err)
	}
	parsed, err := ParseDecisionRecord(decision.CanonicalBytes(), record)
	if err != nil || parsed.Digest() != decision.Digest() || !parsed.ChangedAfterReveal() ||
		parsed.PostRevealChangeRationale() != rationale {
		t.Fatalf("changed DecisionRecord did not reconstruct exactly: %v", err)
	}
}

func TestDecisionSessionNoncompilableActionMatrixRoundTrips(t *testing.T) {
	record := checkedChoicepointExample(t)
	for _, action := range []Action{ActionRejectAll, ActionDefer, ActionRefine} {
		session := visitedBlindSession(t, record)
		draft := RulingDraftInput{Action: action, SelectedFields: []string{}, AllowedAliases: []string{}}
		session = proposedRevealedSession(t, session, draft)
		var err error
		session, err = session.Revise(draft, "")
		if err != nil {
			t.Fatalf("%s revision: %v", action, err)
		}
		_, decision, err := session.Finalize("local-roundtrip-operator", string(action), []domain.ReceiptReference{})
		if err != nil || decision.Action() != action {
			t.Fatalf("%s finalization: action=%s err=%v", action, decision.Action(), err)
		}
		if _, compilable := decision.CompilableRuling(); compilable {
			t.Fatalf("%s DecisionRecord became compilable", action)
		}
		parsed, err := ParseDecisionRecord(decision.CanonicalBytes(), record)
		if err != nil || parsed.Digest() != decision.Digest() || parsed.Action() != action {
			t.Fatalf("%s DecisionRecord round trip: %v", action, err)
		}
	}
}

func TestDecisionSessionCustomExpectationRoundTripsAsCompilable(t *testing.T) {
	record := checkedChoicepointExample(t)
	customValue, err := CanonicalJSONBytes([]byte(`{"custom":"expectation"}`))
	if err != nil {
		t.Fatal(err)
	}
	expectation := CompleteTuple{Fields: []FieldValue{{FieldID: WholeProjectionFieldID, Value: customValue}}}
	draft := RulingDraftInput{
		Action: ActionCustomExpectation, SelectedFields: []string{WholeProjectionFieldID}, AllowedAliases: []string{},
		CustomExpectation: &expectation, CustomReviewer: "local-custom-reviewer",
		CustomReviewEvidence: domain.MustDigest("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
	}
	session := proposedRevealedSession(t, visitedBlindSession(t, record), draft)
	session, err = session.Revise(draft, "")
	if err != nil {
		t.Fatal(err)
	}
	_, decision, err := session.Finalize("local-roundtrip-operator", "custom exact tuple", []domain.ReceiptReference{})
	if err != nil || decision.Action() != ActionCustomExpectation {
		t.Fatalf("custom finalization = %s, %v", decision.Action(), err)
	}
	if _, compilable := decision.CompilableRuling(); !compilable {
		t.Fatal("separately reviewed custom expectation was not construction-safe compilable authority")
	}
	parsed, err := ParseDecisionRecord(decision.CanonicalBytes(), record)
	if err != nil || parsed.Digest() != decision.Digest() {
		t.Fatalf("custom DecisionRecord round trip: %v", err)
	}
}

func checkedChoicepointExample(t testing.TB) ChoicepointRecord {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller path unavailable")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(source), "..", "..", "spec", "examples", "v1", "choicepoint.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 || body[len(body)-1] != '\n' {
		t.Fatal("checked Choicepoint example lacks its exact trailing newline")
	}
	record, err := ParseChoicepointRecord(body[:len(body)-1])
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func visitedBlindSession(t testing.TB, record ChoicepointRecord) Session {
	t.Helper()
	session, err := NewSession(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []ReviewSurface{
		SurfaceOriginalWitness, SurfaceMinimizedWitness, SurfaceReductionDerivation,
		SurfaceProjectionOperations, SurfaceNonassertedFields,
	} {
		session, err = session.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	return session
}

func proposedRevealedSession(t testing.TB, session Session, draft RulingDraftInput) Session {
	t.Helper()
	var err error
	session, err = session.Propose(draft)
	if err != nil {
		t.Fatal(err)
	}
	session, _, err = session.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	session, err = session.Visit(SurfaceProvenance)
	if err != nil {
		t.Fatal(err)
	}
	return session
}
